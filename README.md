# sumx

Sealed sum types (tagged unions / Rust-style enums) for Go — with ergonomic
matching and **exhaustiveness you can actually enforce**.

Go has no enums. The idiomatic stand-in is a *sealed interface* (an interface
with an unexported method, so only types in its own package can implement it).
What's missing is the part Rust gives you for free: a guarantee that you
handled every variant. `sumx` supplies it — at test time with the library, and
at **compile time** with the generator.

Library is zero-dependency. Go 1.24+.

```go
import "github.com/zkrebbekx/sumx"
```

## The sealed type

```go
type Payment interface{ sealedPayment() } // unexported method = sealed

type Card     struct{ Last4 string }
type Cash     struct{ Cents int64 }
type Transfer struct{ IBAN string }

func (Card) sealedPayment()     {}
func (Cash) sealedPayment()     {}
func (Transfer) sealedPayment() {}
```

## Library: match + assert exhaustiveness

`On` is a free function (not a method) so it can carry its own type parameter —
that keeps each handler fully typed:

```go
m := sumx.NewMatcher[string]()
sumx.On(m, func(c Card) string { return "card " + c.Last4 })
sumx.On(m, func(c Cash) string { return "cash" })
sumx.On(m, func(t Transfer) string { return "wire " + t.IBAN })

label := m.Eval(p)            // dispatch by dynamic type; panics if unhandled
label, ok := m.EvalSafe(p)    // ...or report a miss instead of panicking
```

Record the variant set once, then assert completeness in a test — no codegen
required. `Define[Payment](...)` only compiles if every listed value actually
implements `Payment`, so the list can't silently drift:

```go
var Payments = sumx.Define[Payment](Card{}, Cash{}, Transfer{})

func TestExhaustive(t *testing.T) {
	// ... build m ...
	if missing := sumx.Missing(Payments, m); len(missing) > 0 {
		t.Fatalf("unhandled variants: %v", missing)
	}
}
```

Plus typed assertion sugar: `sumx.As[Card](p)` and `sumx.Is[Cash](p)`.

## Generator: compile-time exhaustiveness

For the real guarantee — *the code won't build if you forget a variant* — let
the generator emit a `Match` function with one positional handler per variant:

```go
//go:generate sumx -type Payment
```

```sh
go install github.com/zkrebbekx/sumx/cmd/sumx@latest
go generate ./...   # writes payment_match.go
```

```go
func describe(p Payment) string {
	return MatchPayment(p,
		func(c Card) string { return "card " + c.Last4 },
		func(c Cash) string { return "cash" },
		func(t Transfer) string { return "wire " + t.IBAN },
	)
}
```

Add a `Voucher` variant and regenerate, and every `MatchPayment` call stops
compiling until you handle it:

```
./pay.go:12: not enough arguments in call to MatchPayment
	have (Payment, func(Card) string, func(Cash) string, func(Transfer) string)
	want (Payment, func(Card) R, func(Cash) R, func(Transfer) R, func(Voucher) R)
```

That's the whole point: adding a case can't silently skip a `switch` you forgot
about. The generator also emits `PaymentVariants` (a `sumx.Define` set) for the
test-time check above.

How it works: the generator scans the package for every type that implements
the interface's marker method(s) and orders them deterministically (by filename,
then source order). Value and pointer receivers are both handled (`Card` vs
`*Node`).

## Choosing a layer

| Want | Use |
|---|---|
| Match without codegen, panic/`ok` on a miss | `Matcher` + `On` + `Eval`/`EvalSafe` |
| Catch missing variants in a test | `Define` + `Missing` |
| Catch missing variants at **compile time** | `//go:generate sumx` → `Match<Iface>` |

## Limitations

- The generator matches implementers by marker-method **name**; types that
  satisfy the interface only through embedding aren't detected.
- Embedded interface methods on the sealed type are not followed.

## Develop

```sh
make test   # library tests, race detector
make lint
# generator + examples:
cd cmd/sumx && make test
cd examples && make gen && make test
```

## License

MIT
