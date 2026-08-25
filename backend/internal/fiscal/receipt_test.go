package fiscal

import "testing"

// The tax inside a price that already includes it is price*rate/(100+rate).
// The obvious formula, price*rate/100, overstates it by about the rate itself
// and produces numbers that look entirely reasonable on a receipt — which is
// why this is a test and not a comment.
func TestVATIsIncludedNotAdded(t *testing.T) {
	// 112 000 so'm at 12% contains exactly 12 000 so'm of VAT.
	if got := vatOf(112_000*100, 12); got != 12_000*100 {
		t.Fatalf("vat = %d tiyin, want %d", got, 12_000*100)
	}
	// The wrong formula would give 13 440 so'm here.
	if got := vatOf(112_000*100, 12); got == 13_440*100 {
		t.Fatal("VAT was added to the price instead of taken out of it")
	}
	if got := vatOf(50_000*100, 0); got != 0 {
		t.Fatalf("zero-rated line got %d tiyin of VAT", got)
	}
}

// A discount spread across lines has to come out exact. Proportional division
// with independent rounding does not, and the shortfall lands on a filed
// document.
func TestDiscountSpreadIsExact(t *testing.T) {
	// Weights chosen so no share divides evenly.
	weights := []int64{3333, 3333, 3334}
	parts := distribute(1000, weights)
	var sum int64
	for _, p := range parts {
		sum += p
	}
	if sum != 1000 {
		t.Fatalf("parts sum to %d, want 1000 (%v)", sum, parts)
	}
}

func TestDiscountLargerThanBasketIsCapped(t *testing.T) {
	parts := distribute(10_000, []int64{1000, 1000})
	var sum int64
	for _, p := range parts {
		sum += p
	}
	if sum != 2000 {
		t.Fatalf("capped sum = %d, want 2000 — a line cannot owe money", sum)
	}
}

// The one invariant a provider will reject the receipt over: what was sold and
// what was taken must be the same number.
func TestPaymentSplitMatchesTheLines(t *testing.T) {
	r := Build(Sale{
		OrderNumber: "MRC-1",
		Lines: []Line{
			{Name: "Lag'mon", Price: 35_000, Qty: 2},
			{Name: "Choy", Price: 5_000, Qty: 3},
		},
		DeliveryFee: 10_000,
		Discount:    7_000,
		// Deliberately only one half of the payment given, which is how a
		// caller taking a single payment type will use this.
		Card:       0,
		Cash:       0,
		VatPercent: 12,
	})
	var total int64
	for _, it := range r.Items {
		total += it.Total()
	}
	if r.ReceivedCash+r.ReceivedCard != total {
		t.Fatalf("payment %d+%d does not match lines %d",
			r.ReceivedCash, r.ReceivedCard, total)
	}
	// 85 000 sold + 10 000 delivery − 7 000 discount = 88 000 so'm.
	if total != 88_000*100 {
		t.Fatalf("total = %d tiyin, want %d", total, 88_000*100)
	}
}

func TestCardAmountIsKeptAndCashTakesTheDifference(t *testing.T) {
	r := Build(Sale{
		Lines:      []Line{{Name: "Osh", Price: 40_000, Qty: 1}},
		Card:       30_000,
		VatPercent: 12,
	})
	if r.ReceivedCard != 30_000*100 {
		t.Fatalf("card = %d, want %d — the terminal's figure was moved",
			r.ReceivedCard, 30_000*100)
	}
	if r.ReceivedCash != 10_000*100 {
		t.Fatalf("cash = %d, want %d", r.ReceivedCash, 10_000*100)
	}
}

// The dish's rate wins over the branch's, and nil means "use the branch's" —
// which is the state almost every dish is in.
func TestDishRateOverridesTheBranchRate(t *testing.T) {
	zero := 0
	r := Build(Sale{
		Lines: []Line{
			{Name: "Standart", Price: 10_000, Qty: 1},
			{Name: "Nol stavka", Price: 10_000, Qty: 1, VatPercent: &zero},
		},
		VatPercent: 12,
	})
	if r.Items[0].VATPercent != 12 {
		t.Fatalf("unset dish got rate %d, want the branch's 12", r.Items[0].VATPercent)
	}
	if r.Items[1].VATPercent != 0 || r.Items[1].VAT != 0 {
		t.Fatalf("zero-rated dish got rate %d / vat %d, want 0 / 0 — a pointer "+
			"is the only way to tell exempt from unfilled",
			r.Items[1].VATPercent, r.Items[1].VAT)
	}
}

// Delivery is a line so the guest can reconcile the receipt, but it carries no
// classifier code: that code is the restaurant's accountant's to declare, and a
// guess would be filed against the wrong product on every delivery.
func TestDeliveryIsALineWithoutAClassifierCode(t *testing.T) {
	r := Build(Sale{
		Lines:       []Line{{Name: "Osh", Price: 40_000, Qty: 1, SPIC: "12345678901234567"}},
		DeliveryFee: 12_000,
		VatPercent:  12,
	})
	if len(r.Items) != 2 {
		t.Fatalf("got %d lines, want 2", len(r.Items))
	}
	if r.Items[1].SPIC != "" {
		t.Fatalf("delivery got SPIC %q, want none", r.Items[1].SPIC)
	}
	if r.Items[1].Total() != 12_000*100 {
		t.Fatalf("delivery line = %d tiyin, want %d", r.Items[1].Total(), 12_000*100)
	}
}

// A voided or zero-quantity line is not a sale, and filing it as one puts a
// product on a state document that nobody bought.
func TestZeroQuantityLinesAreNotFiled(t *testing.T) {
	r := Build(Sale{
		Lines: []Line{
			{Name: "Osh", Price: 40_000, Qty: 1},
			{Name: "Bekor qilingan", Price: 40_000, Qty: 0},
		},
		VatPercent: 12,
	})
	if len(r.Items) != 1 {
		t.Fatalf("got %d lines, want 1", len(r.Items))
	}
}

// Money is in tiyin here and nowhere else in the codebase. A factor-of-100
// mistake on a filed document is not a rounding bug.
func TestMoneyIsConvertedToTiyinExactlyOnce(t *testing.T) {
	r := Build(Sale{
		Lines:      []Line{{Name: "Choy", Price: 5_000, Qty: 2}},
		VatPercent: 12,
	})
	if r.Items[0].Price != 5_000*100 {
		t.Fatalf("unit price = %d tiyin, want %d", r.Items[0].Price, 5_000*100)
	}
	if r.Items[0].Qty != 2 {
		t.Fatalf("qty = %d, want 2 — quantity is scaled by the adapter, not here",
			r.Items[0].Qty)
	}
}

// Every provider in the list is answerable, by whichever of the two transports
// it belongs to. A provider that can be picked in the panel and then falls
// through to an unhelpful default is worse than one that is not offered.
//
// ⚠️ A ready **local** provider must fail New with ErrLocalProvider rather than
// ErrNoAdapter. The two messages send an owner to opposite places — one to wait
// for us to build something, the other to check that the till tablet is on the
// restaurant's own network — and only the second is ever true here.
func TestEveryListedProviderIsHandled(t *testing.T) {
	for _, p := range Providers() {
		if !Known(p.ID) {
			t.Fatalf("%s is listed but Known says no", p.ID)
		}
		if Name(p.ID) == p.ID {
			t.Fatalf("%s has no display name", p.ID)
		}
		if IsLocal(p.ID) != p.Local {
			t.Fatalf("%s: IsLocal disagrees with the listing", p.ID)
		}

		_, err := New(p.ID, Creds{})
		switch {
		case !p.Ready:
			if err != ErrNoAdapter {
				t.Fatalf("%s is not ready, want ErrNoAdapter, got %v", p.ID, err)
			}
		case p.Local:
			if err != ErrLocalProvider {
				t.Fatalf("%s is local, want ErrLocalProvider from New, got %v", p.ID, err)
			}
		default:
			if err != nil {
				t.Fatalf("%s is marked ready but New failed: %v", p.ID, err)
			}
		}

		// ⚠️ "Ready" claims an **adapter exists**, not that this branch has
		// filled the form in. A provider that genuinely needs a login says
		// ErrNotConfigured on empty credentials, and conflating the two would
		// force every future adapter to pretend it can work with nothing.
		_, err = EncoderFor(p.ID, Creds{})
		if p.Ready && err == ErrNoAdapter {
			t.Fatalf("%s is marked ready but has no adapter", p.ID)
		}
		if !p.Ready && err != ErrNoAdapter {
			t.Fatalf("%s is not ready, want ErrNoAdapter from EncoderFor, got %v", p.ID, err)
		}

		// And with credentials, a ready provider must actually build.
		if p.Ready {
			enc, err := EncoderFor(p.ID, Creds{
				Login: "kassa", Password: "secret", RegisterID: "1",
			})
			if err != nil || enc == nil {
				t.Fatalf("%s could not build with credentials: %v", p.ID, err)
			}
		}
	}
}

// ⚠️ **A marked line has to reach the register with its code, and an unmarked
// one has to reach it with no field at all.** Both halves matter: without the
// code the bottle is never withdrawn from circulation and the restaurant is
// non-compliant on a sale it made correctly; with an empty field the register
// reads "marked, no code" and refuses the receipt — in front of a guest, at the
// moment the money is being taken.
func TestMarkingCodeTravelsOnTheLineItWasScannedFor(t *testing.T) {
	const code = "0104607034170203215Fw2R\x1d93dGVz"
	r := Build(Sale{
		Lines: []Line{
			{Name: "Suv 0.5", Price: 8000, Qty: 1, MarkCode: code},
			{Name: "Osh", Price: 30000, Qty: 2},
		},
		Cash:       46000,
		VatPercent: 12,
	})
	if len(r.Items) != 2 {
		t.Fatalf("items = %d", len(r.Items))
	}
	if r.Items[0].MarkCode != code {
		t.Fatalf("marked line lost its code: %q", r.Items[0].MarkCode)
	}
	if r.Items[1].MarkCode != "" {
		t.Fatalf("unmarked line carries %q", r.Items[1].MarkCode)
	}
}
