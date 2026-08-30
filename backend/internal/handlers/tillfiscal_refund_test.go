package handlers

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

func filedSale(t *testing.T) (*models.Order, *models.FiscalSettings) {
	t.Helper()
	sold := time.Date(2026, 8, 20, 13, 0, 0, 0, time.Local)
	back := time.Date(2026, 8, 27, 10, 30, 0, 0, time.Local)
	o := &models.Order{
		Number: "A-17",
		Items:  []models.OrderItem{{Name: "Lag'mon", Price: 42000, Qty: 1}},
		PaidAt: &sold,
		Fiscal: &models.FiscalReceipt{
			Status:     models.FiscalFiled,
			FiscalSign: "884412300071",
			ReceiptID:  "119",
			FiledAt:    &sold,
		},
		Refund: &models.CheckRefund{At: back},
	}
	s := &models.FiscalSettings{Provider: "multikassa", TIN: "302345678"}
	return o, s
}

// ⚠️ A reversal is only a reversal because it names the sale it undoes. Filed
// without the original's sign it is a **second document with a minus on it** —
// the register may even accept it, and the two receipts then sit in the tax
// committee's copy unrelated to one another.
func TestAReversalCarriesTheSaleItUndoes(t *testing.T) {
	o, s := filedSale(t)
	got := refundFor(o, s, "Nodira", map[primitive.ObjectID]menuFiscalInfo{})

	if !got.IsRefund {
		t.Fatal("the reversal is being filed as an ordinary sale")
	}
	if got.Original.Sign != "884412300071" || got.Original.SaleID != "119" {
		t.Fatalf("original=%+v — the reversal does not name the sale", got.Original)
	}
	if !got.Original.At.Equal(*o.Fiscal.FiledAt) {
		t.Fatalf("original.At=%v, want the sale's filing time", got.Original.At)
	}
	// ⚠️ The two times are different days here on purpose: the check was paid a
	// week before it was refunded. Dating the reversal from the meal would file
	// it into a day that has already been totalled and closed.
	if !got.Time.Equal(o.Refund.At) {
		t.Fatalf("time=%v, want the refund's own moment %v", got.Time, o.Refund.At)
	}
}

// A sale that never reached the register has nothing to reverse, and the
// reversal must not invent an original out of an empty record.
func TestAnUnfiledSaleLeavesTheOriginalEmpty(t *testing.T) {
	o, s := filedSale(t)
	o.Fiscal = nil

	got := refundFor(o, s, "Nodira", map[primitive.ObjectID]menuFiscalInfo{})
	if got.Original.Known() {
		t.Fatalf("original=%+v — a sale that was never filed is being reversed", got.Original)
	}
}

// ⚠️ The Z-report guard and the unfiled alert both used to look at `fiscal`
// only. An unfiled reversal is the same hole read from the other side: the sale
// stays in the day's figures and the money that went back out of the drawer
// does not.
func TestAnUnfiledReversalAlsoBlocksTheDay(t *testing.T) {
	src := readSource(t, "tillfiscal.go")
	for _, fn := range []string{"func anyUnfiledFilter", "func unfiledFiscalFilter"} {
		body := between(t, src, fn, "\n}\n")
		if !contains(body, "fiscalRefund.status") {
			t.Fatalf("%s ignores an unfiled reversal", fn)
		}
	}
}
