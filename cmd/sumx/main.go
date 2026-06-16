package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "sumx:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		typeList = flag.String("type", "", "comma-separated sealed interface names to generate Match for (required)")
		dir      = flag.String("dir", "", "package directory to scan; defaults to the directory of $GOFILE, or the current directory")
	)
	flag.Parse()

	if *typeList == "" {
		return fmt.Errorf("-type is required (e.g. -type Shape)")
	}

	scanDir := *dir
	if scanDir == "" {
		if gofile := os.Getenv("GOFILE"); gofile != "" {
			scanDir = filepath.Dir(gofile)
		} else {
			scanDir = "."
		}
	}
	if scanDir == "" {
		scanDir = "."
	}

	for _, iface := range strings.Split(*typeList, ",") {
		iface = strings.TrimSpace(iface)
		if iface == "" {
			continue
		}
		spec, err := parsePackage(scanDir, iface)
		if err != nil {
			return err
		}
		out, err := generate(spec)
		if err != nil {
			return err
		}
		outPath := filepath.Join(scanDir, strings.ToLower(iface)+"_match.go")
		if err := os.WriteFile(outPath, out, 0o644); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "sumx: wrote", outPath)
	}
	return nil
}
