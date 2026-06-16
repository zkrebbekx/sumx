// Package main shows a compile-time-exhaustive Match generated from a sealed
// sum type. Run `make gen` to (re)generate payment_match.go after changing
// the variant set.
package main

import "fmt"

//go:generate sumx -type Payment

// Payment is a sealed sum type: only the variants below can implement it.
type Payment interface{ sealedPayment() }

// Card, Cash, and Transfer are the variants.
type Card struct {
	Last4 string
	Cents int64
}

type Cash struct {
	Cents int64
}

type Transfer struct {
	IBAN  string
	Cents int64
}

func (Card) sealedPayment()     {}
func (Cash) sealedPayment()     {}
func (Transfer) sealedPayment() {}

// describe uses the generated MatchPayment. If a new Payment variant is added
// and the code regenerated, this call stops compiling until the new case is
// handled — that is the exhaustiveness guarantee.
func describe(p Payment) string {
	return MatchPayment(p,
		func(c Card) string { return fmt.Sprintf("card ****%s", c.Last4) },
		func(c Cash) string { return fmt.Sprintf("cash %d", c.Cents) },
		func(t Transfer) string { return fmt.Sprintf("transfer %s", t.IBAN) },
	)
}

func main() {
	fmt.Println(describe(Card{Last4: "4242", Cents: 1500}))
	fmt.Println(describe(Cash{Cents: 700}))
	fmt.Println(describe(Transfer{IBAN: "DE89", Cents: 9900}))
	fmt.Println("variants:", PaymentVariants.Variants())
}
