// Command sumx generates a compile-time-exhaustive Match function for a
// sealed sum type. This file holds the parse + emit core; main.go wires the
// CLI.
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"sort"
	"strings"
)

// variant is one concrete type implementing the sealed interface.
type variant struct {
	Name    string // base type name, e.g. "Circle" or "Leaf"
	Pointer bool   // implemented with a pointer receiver -> variant type is *Name
	order   int    // declaration order, for stable output
}

// Type returns the variant's type expression, e.g. "Circle" or "*Leaf".
func (v variant) Type() string {
	if v.Pointer {
		return "*" + v.Name
	}
	return v.Name
}

// param returns the handler parameter name, e.g. "circle" for Circle.
func (v variant) param() string {
	name := lowerFirst(v.Name)
	if isGoKeyword(name) {
		name += "_"
	}
	return name
}

// sumSpec is everything needed to emit one Match function.
type sumSpec struct {
	Package  string
	Iface    string
	Variants []variant
}

// parsePackage scans every non-test .go file in dir, locates the sealed
// interface ifaceName, and collects the package's types that implement it.
func parsePackage(dir, ifaceName string) (*sumSpec, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", dir, err)
	}

	// Merge files across the (single expected) package, ordered by filename
	// so variant order is deterministic — ParseDir returns files in a map.
	var pkgName string
	var names []string
	byName := map[string]*ast.File{}
	for name, pkg := range pkgs {
		pkgName = name
		for fname, f := range pkg.Files {
			names = append(names, fname)
			byName[fname] = f
		}
	}
	if pkgName == "" {
		return nil, fmt.Errorf("no Go package found in %s", dir)
	}
	sort.Strings(names)
	files := make([]*ast.File, len(names))
	for i, n := range names {
		files[i] = byName[n]
	}

	methods, ok := interfaceMethods(files, ifaceName)
	if !ok {
		return nil, fmt.Errorf("sealed interface %s not found in %s", ifaceName, dir)
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("interface %s has no (non-embedded) methods to seal on", ifaceName)
	}

	order := typeDeclOrder(files)
	impls := implementers(files, methods)

	var variants []variant
	for name, info := range impls {
		variants = append(variants, variant{
			Name:    name,
			Pointer: info.pointer,
			order:   order[name],
		})
	}
	if len(variants) == 0 {
		return nil, fmt.Errorf("no types in %s implement %s", dir, ifaceName)
	}
	sort.Slice(variants, func(i, j int) bool { return variants[i].order < variants[j].order })

	return &sumSpec{Package: pkgName, Iface: ifaceName, Variants: variants}, nil
}

// interfaceMethods returns the set of method names declared directly on the
// named interface (embedded interfaces are not followed).
func interfaceMethods(files []*ast.File, ifaceName string) (map[string]struct{}, bool) {
	for _, f := range files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name.Name != ifaceName {
					continue
				}
				it, ok := ts.Type.(*ast.InterfaceType)
				if !ok {
					return nil, false
				}
				methods := map[string]struct{}{}
				for _, m := range it.Methods.List {
					for _, n := range m.Names { // embedded interfaces have no Names
						methods[n.Name] = struct{}{}
					}
				}
				return methods, true
			}
		}
	}
	return nil, false
}

type implInfo struct {
	got     map[string]struct{}
	pointer bool
}

// implementers returns the package types whose method set covers every name
// in want, along with whether a pointer receiver is required.
func implementers(files []*ast.File, want map[string]struct{}) map[string]implInfo {
	seen := map[string]*implInfo{}
	for _, f := range files {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) != 1 {
				continue
			}
			base, pointer := receiverType(fd.Recv.List[0].Type)
			if base == "" {
				continue
			}
			if _, isMarker := want[fd.Name.Name]; !isMarker {
				continue
			}
			info := seen[base]
			if info == nil {
				info = &implInfo{got: map[string]struct{}{}}
				seen[base] = info
			}
			info.got[fd.Name.Name] = struct{}{}
			if pointer {
				info.pointer = true
			}
		}
	}

	out := map[string]implInfo{}
	for name, info := range seen {
		if len(info.got) == len(want) {
			out[name] = *info
		}
	}
	return out
}

// receiverType returns the base type name of a receiver and whether it is a
// pointer receiver.
func receiverType(expr ast.Expr) (string, bool) {
	switch t := expr.(type) {
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name, true
		}
	case *ast.Ident:
		return t.Name, false
	}
	return "", false
}

// typeDeclOrder maps each top-level type name to its declaration order. The
// files slice is already sorted by filename and each file's Decls are in
// source order, so iterating in sequence is deterministic.
func typeDeclOrder(files []*ast.File) map[string]int {
	order := map[string]int{}
	i := 0
	for _, f := range files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					order[ts.Name.Name] = i
					i++
				}
			}
		}
	}
	return order
}

// generate renders the Match function and variant set for a spec.
func generate(s *sumSpec) ([]byte, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Code generated by \"sumx -type %s\"; DO NOT EDIT.\n\n", s.Iface)
	fmt.Fprintf(&buf, "package %s\n\n", s.Package)
	buf.WriteString("import \"github.com/zkrebbekx/sumx\"\n\n")

	// Variant set.
	fmt.Fprintf(&buf, "// %sVariants is the recorded variant set of %s.\n", s.Iface, s.Iface)
	fmt.Fprintf(&buf, "var %sVariants = sumx.Define[%s](\n", s.Iface, s.Iface)
	for _, v := range s.Variants {
		fmt.Fprintf(&buf, "\t*new(%s),\n", v.Type())
	}
	buf.WriteString(")\n\n")

	// Match function: one positional handler per variant. Adding or removing
	// a variant changes this signature, breaking every call site until
	// updated — that is the compile-time exhaustiveness guarantee.
	fmt.Fprintf(&buf, "// Match%s dispatches v to exactly one handler and returns the result.\n", s.Iface)
	fmt.Fprintf(&buf, "// Adding or removing a %s variant changes this signature, so every call\n", s.Iface)
	buf.WriteString("// site must handle every variant — compile-time exhaustiveness.\n")
	fmt.Fprintf(&buf, "func Match%s[R any](\n\tv %s,\n", s.Iface, s.Iface)
	for _, vr := range s.Variants {
		fmt.Fprintf(&buf, "\t%s func(%s) R,\n", vr.param(), vr.Type())
	}
	buf.WriteString(") R {\n\tswitch x := v.(type) {\n")
	for _, vr := range s.Variants {
		fmt.Fprintf(&buf, "\tcase %s:\n\t\treturn %s(x)\n", vr.Type(), vr.param())
	}
	fmt.Fprintf(&buf, "\tdefault:\n\t\tpanic(\"sumx: unhandled %s variant\")\n\t}\n}\n", s.Iface)

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("gofmt generated source for %s: %w\n%s", s.Iface, err, buf.String())
	}
	return formatted, nil
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

var goKeywords = map[string]struct{}{
	"break": {}, "case": {}, "chan": {}, "const": {}, "continue": {}, "default": {},
	"defer": {}, "else": {}, "fallthrough": {}, "for": {}, "func": {}, "go": {},
	"goto": {}, "if": {}, "import": {}, "interface": {}, "map": {}, "package": {},
	"range": {}, "return": {}, "select": {}, "struct": {}, "switch": {}, "type": {}, "var": {},
}

func isGoKeyword(s string) bool {
	_, ok := goKeywords[s]
	return ok
}
