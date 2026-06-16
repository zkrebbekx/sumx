// Package sumx brings sealed sum types (a.k.a. tagged unions / Rust-style
// enums) to Go, with ergonomic matching and exhaustiveness you can assert.
//
// A sum type in Go is a sealed interface: an interface with an unexported
// method, so only types in the same package can implement it. The variants
// are the concrete types that implement it.
//
//	type Shape interface{ sealedShape() }
//
//	type Circle struct{ R float64 }
//	type Rect   struct{ W, H float64 }
//
//	func (Circle) sealedShape() {}
//	func (Rect) sealedShape()   {}
//
// sumx gives you three things on top of that pattern:
//
//   - [As] / [Is]: typed assertion helpers.
//   - [Matcher] with [On] / [Eval]: dispatch a value to a handler chosen by
//     its dynamic type, returning a result.
//   - [Define] / [Missing]: record a sum type's variant set so a test can
//     assert that a matcher handles every variant — exhaustiveness today,
//     without code generation.
//
// A go:generate tool that emits a fully compile-time-exhaustive Match
// function is planned (v0.2); it targets the same sealed-interface shape.
package sumx

import (
	"fmt"
	"reflect"
)

// As returns v as type T, reporting whether v's dynamic type is T. It is a
// thin, generic wrapper over a type assertion, handy for variant types.
func As[T any](v any) (T, bool) {
	t, ok := v.(T)
	return t, ok
}

// Is reports whether v's dynamic type is T.
func Is[T any](v any) bool {
	_, ok := v.(T)
	return ok
}

// Matcher dispatches a value to a handler chosen by the value's dynamic
// type, producing a result of type R. Build one with [NewMatcher], register
// handlers with [On], then call [Matcher.Eval].
//
// A Matcher is not safe for concurrent registration, but once built it is
// safe for concurrent Eval.
type Matcher[R any] struct {
	handlers map[reflect.Type]func(any) R
	fallback func(any) R
}

// NewMatcher returns an empty Matcher producing values of type R.
func NewMatcher[R any]() *Matcher[R] {
	return &Matcher[R]{handlers: map[reflect.Type]func(any) R{}}
}

// On registers h as the handler for variant C. It is a free function rather
// than a method so it can introduce its own type parameter C; a method
// cannot. Registering C twice replaces the earlier handler.
func On[C, R any](m *Matcher[R], h func(C) R) *Matcher[R] {
	m.handlers[typeOf[C]()] = func(v any) R { return h(v.(C)) }
	return m
}

// Default sets a fallback handler used when no variant handler matches.
// Without it, [Matcher.Eval] panics on an unhandled value.
func (m *Matcher[R]) Default(h func(any) R) *Matcher[R] {
	m.fallback = h
	return m
}

// Eval dispatches v to its handler and returns the result. If no handler is
// registered for v's dynamic type and no [Matcher.Default] is set, Eval
// panics.
func (m *Matcher[R]) Eval(v any) R {
	if h, ok := m.handlers[reflect.TypeOf(v)]; ok {
		return h(v)
	}
	if m.fallback != nil {
		return m.fallback(v)
	}
	panic(fmt.Sprintf("sumx: no handler for %T (and no Default)", v))
}

// EvalSafe is like [Matcher.Eval] but reports whether a handler (or default)
// matched instead of panicking. On no match it returns the zero R and false.
func (m *Matcher[R]) EvalSafe(v any) (R, bool) {
	if h, ok := m.handlers[reflect.TypeOf(v)]; ok {
		return h(v), true
	}
	if m.fallback != nil {
		return m.fallback(v), true
	}
	var zero R
	return zero, false
}

// Sum records the set of variant types of a sealed interface T. Build one
// with [Define] and use [Missing] to check a matcher for completeness.
type Sum[T any] struct {
	variants []reflect.Type
}

// Define records the variants of sealed type T. Because the parameters are
// typed T, every listed variant must implement T — a value that does not is
// a compile error, so the variant list cannot drift from the interface.
//
//	var Shapes = sumx.Define[Shape](Circle{}, Rect{})
func Define[T any](variants ...T) Sum[T] {
	s := Sum[T]{variants: make([]reflect.Type, 0, len(variants))}
	for _, v := range variants {
		s.variants = append(s.variants, reflect.TypeOf(v))
	}
	return s
}

// Variants returns the names of the recorded variant types, in declaration
// order.
func (s Sum[T]) Variants() []string {
	out := make([]string, len(s.variants))
	for i, t := range s.variants {
		out[i] = typeName(t)
	}
	return out
}

// Missing returns the names of variants in s that m does not handle. An
// empty result means m is exhaustive over s. Assert this in a test to get
// exhaustiveness checking without code generation:
//
//	So(sumx.Missing(Shapes, m), ShouldBeEmpty)
func Missing[T, R any](s Sum[T], m *Matcher[R]) []string {
	var out []string
	for _, t := range s.variants {
		if _, ok := m.handlers[t]; !ok {
			out = append(out, typeName(t))
		}
	}
	return out
}

// typeOf returns the reflect.Type of T even when T is an interface or its
// zero value is nil (e.g. a pointer variant).
func typeOf[T any]() reflect.Type {
	var zero T
	return reflect.TypeOf(&zero).Elem()
}

func typeName(t reflect.Type) string {
	if t == nil {
		return "<nil>"
	}
	return t.String()
}
