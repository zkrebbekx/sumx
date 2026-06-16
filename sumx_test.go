package sumx_test

import (
	"testing"

	"github.com/zkrebbekx/sumx"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAsIs(t *testing.T) {
	Convey("Given a Shape holding a Rect", t, func() {
		var s Shape = Rect{W: 2, H: 3}

		Convey("When asserted As the right variant", func() {
			r, ok := sumx.As[Rect](s)

			Convey("Then it succeeds and returns the value", func() {
				So(ok, ShouldBeTrue)
				So(r.W, ShouldEqual, 2)
			})
		})

		Convey("When asserted As the wrong variant", func() {
			_, ok := sumx.As[Circle](s)

			Convey("Then it reports false", func() {
				So(ok, ShouldBeFalse)
				So(sumx.Is[Circle](s), ShouldBeFalse)
				So(sumx.Is[Rect](s), ShouldBeTrue)
			})
		})
	})
}

func TestMatcherEval(t *testing.T) {
	Convey("Given a matcher over Shape with both variants handled", t, func() {
		m := sumx.NewMatcher[string]()
		sumx.On(m, func(Circle) string { return "circle" })
		sumx.On(m, func(Rect) string { return "rect" })

		Convey("When evaluating each variant", func() {
			Convey("Then the matching handler runs", func() {
				So(m.Eval(Circle{}), ShouldEqual, "circle")
				So(m.Eval(Rect{}), ShouldEqual, "rect")
			})
		})
	})

	Convey("Given a matcher missing a variant", t, func() {
		m := sumx.NewMatcher[string]()
		sumx.On(m, func(Circle) string { return "circle" })

		Convey("When evaluating the unhandled variant with no default", func() {
			Convey("Then Eval panics", func() {
				So(func() { m.Eval(Rect{}) }, ShouldPanic)
			})
		})

		Convey("When a default is set", func() {
			m.Default(func(any) string { return "other" })

			Convey("Then the default catches the unhandled variant", func() {
				So(m.Eval(Rect{}), ShouldEqual, "other")
				So(m.Eval(Circle{}), ShouldEqual, "circle")
			})
		})
	})
}

func TestMatcherEvalSafe(t *testing.T) {
	Convey("Given a matcher handling only Circle", t, func() {
		m := sumx.NewMatcher[int]()
		sumx.On(m, func(Circle) int { return 1 })

		Convey("When EvalSafe is called on a handled value", func() {
			v, ok := m.EvalSafe(Circle{})

			Convey("Then it returns the result and true", func() {
				So(v, ShouldEqual, 1)
				So(ok, ShouldBeTrue)
			})
		})

		Convey("When EvalSafe is called on an unhandled value", func() {
			v, ok := m.EvalSafe(Rect{})

			Convey("Then it returns the zero value and false", func() {
				So(v, ShouldEqual, 0)
				So(ok, ShouldBeFalse)
			})
		})
	})
}

func TestOnReplacesHandler(t *testing.T) {
	Convey("Given a variant registered twice", t, func() {
		m := sumx.NewMatcher[string]()
		sumx.On(m, func(Circle) string { return "first" })
		sumx.On(m, func(Circle) string { return "second" })

		Convey("When evaluated", func() {
			Convey("Then the later handler wins", func() {
				So(m.Eval(Circle{}), ShouldEqual, "second")
			})
		})
	})
}

func TestExhaustiveness(t *testing.T) {
	Convey("Given the recorded Shape variant set", t, func() {
		Convey("Then it lists every variant by name", func() {
			So(Shapes.Variants(), ShouldResemble, []string{"sumx_test.Circle", "sumx_test.Rect"})
		})

		Convey("When a matcher handles every variant", func() {
			m := sumx.NewMatcher[float64]()
			sumx.On(m, func(Circle) float64 { return 0 })
			sumx.On(m, func(Rect) float64 { return 0 })

			Convey("Then Missing reports nothing (exhaustive)", func() {
				So(sumx.Missing(Shapes, m), ShouldBeEmpty)
			})
		})

		Convey("When a matcher omits a variant", func() {
			m := sumx.NewMatcher[float64]()
			sumx.On(m, func(Circle) float64 { return 0 })

			Convey("Then Missing names the gap", func() {
				So(sumx.Missing(Shapes, m), ShouldResemble, []string{"sumx_test.Rect"})
			})
		})
	})
}

// Node is a sealed sum type whose variants are pointers, to exercise
// pointer-variant handling.
type Node interface{ sealedNode() }

type Leaf struct{ V int }
type Branch struct{ L, R Node }

func (*Leaf) sealedNode()   {}
func (*Branch) sealedNode() {}

func TestPointerVariants(t *testing.T) {
	Convey("Given a matcher over pointer variants", t, func() {
		m := sumx.NewMatcher[int]()
		sumx.On(m, func(l *Leaf) int { return l.V })
		sumx.On(m, func(*Branch) int { return 100 })

		Convey("When evaluating each pointer variant", func() {
			Convey("Then dispatch resolves the pointer type correctly", func() {
				So(m.Eval(&Leaf{V: 7}), ShouldEqual, 7)
				So(m.Eval(&Branch{}), ShouldEqual, 100)
			})
		})

		Convey("And the variant set is exhaustive", func() {
			nodes := sumx.Define[Node](&Leaf{}, &Branch{})
			So(sumx.Missing(nodes, m), ShouldBeEmpty)
		})
	})
}
