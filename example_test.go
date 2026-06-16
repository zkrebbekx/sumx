package sumx_test

import (
	"fmt"
	"math"

	"github.com/zkrebbekx/sumx"
)

// Shape is a sealed sum type: the unexported method means only types in this
// package can implement it.
type Shape interface{ sealedShape() }

type Circle struct{ R float64 }
type Rect struct{ W, H float64 }

func (Circle) sealedShape() {}
func (Rect) sealedShape()   {}

// Shapes records the variant set once, so tests can assert exhaustiveness.
var Shapes = sumx.Define[Shape](Circle{}, Rect{})

func area(s Shape) float64 {
	m := sumx.NewMatcher[float64]()
	sumx.On(m, func(c Circle) float64 { return math.Pi * c.R * c.R })
	sumx.On(m, func(r Rect) float64 { return r.W * r.H })
	return m.Eval(s)
}

func Example() {
	fmt.Printf("%.2f\n", area(Rect{W: 2, H: 3}))
	fmt.Printf("%.2f\n", area(Circle{R: 1}))
	// Output:
	// 6.00
	// 3.14
}
