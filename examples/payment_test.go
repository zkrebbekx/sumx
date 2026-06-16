package main

import (
	"testing"

	"github.com/zkrebbekx/sumx"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGeneratedMatchPayment(t *testing.T) {
	Convey("Given the generated MatchPayment", t, func() {
		Convey("When each variant is matched", func() {
			Convey("Then the matching handler runs", func() {
				So(describe(Card{Last4: "4242"}), ShouldEqual, "card ****4242")
				So(describe(Cash{Cents: 700}), ShouldEqual, "cash 700")
				So(describe(Transfer{IBAN: "DE89"}), ShouldEqual, "transfer DE89")
			})
		})
	})

	Convey("Given the generated PaymentVariants set", t, func() {
		Convey("Then it lists every variant in declaration order", func() {
			So(PaymentVariants.Variants(), ShouldResemble,
				[]string{"main.Card", "main.Cash", "main.Transfer"})
		})

		Convey("When a matcher handles every variant", func() {
			m := sumx.NewMatcher[bool]()
			sumx.On(m, func(Card) bool { return true })
			sumx.On(m, func(Cash) bool { return true })
			sumx.On(m, func(Transfer) bool { return true })

			Convey("Then it is exhaustive over the generated variant set", func() {
				So(sumx.Missing(PaymentVariants, m), ShouldBeEmpty)
			})
		})
	})
}
