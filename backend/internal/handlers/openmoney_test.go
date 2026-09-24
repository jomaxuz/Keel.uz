package handlers

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// Dates the way Mongo returns them (UTC) — CLAUDE.md §10.
var (
	mFrom = time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local).UTC()
	mTo   = time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local).UTC()
	mDay  = time.Date(2026, 9, 10, 12, 0, 0, 0, time.Local).UTC()
)

func mOrder(total int, pay string, status models.OrderStatus) models.Order {
	return models.Order{
		ID: primitive.NewObjectID(), Number: "A1", Total: total, Type: "dinein",
		PaymentMethod: "cash", PaymentStatus: pay, Status: status, CreatedAt: mDay,
		Customer: models.OrderCustomer{Name: "Aziz Karimov", Phone: "+998901234567"},
	}
}

func byID(entries []MoneyEntry) map[string]MoneyEntry {
	m := map[string]MoneyEntry{}
	for _, e := range entries {
		m[e.ID] = e
	}
	return m
}

// ⚠️ **The accountant and the owner see one number.** The P&L entries of the
// ledger must add up to the money report's "Kirim − chiqim" for the same
// documents — the report reads orders with financeLines and the rest with its
// own queries, so both halves are summed here the way the report sums them.
func TestMoneyLedgerAgreesWithFinanceReport(t *testing.T) {
	sold := mOrder(100_000, models.PayPaid, models.StatusDelivered)
	refunded := mOrder(40_000, models.PayRefunded, models.StatusDelivered)
	refunded.Refund = &models.CheckRefund{At: mDay.Add(time.Hour), Amount: 40_000, Method: "cash"}
	pending := mOrder(70_000, models.PayUnpaid, models.StatusPreparing)
	withCourier := mOrder(50_000, models.PayPaid, models.StatusDelivered)
	withCourier.ExternalDelivery = &models.ExternalDelivery{Cost: 15_000, ProviderName: "Yandex"}
	orders := []models.Order{sold, refunded, pending, withCourier}

	src := &moneySources{
		Orders:      orders,
		Purchases:   []models.Purchase{{ID: primitive.NewObjectID(), At: mDay, Total: 30_000, Supplier: "Bozor"}},
		Expenses:    []models.Expense{{ID: primitive.NewObjectID(), At: mDay, Amount: 20_000, Category: "ijara"}},
		StaffPays:   []models.StaffPayment{{ID: primitive.NewObjectID(), At: mDay, Amount: 10_000}},
		CourierPays: []models.CourierPayment{{ID: primitive.NewObjectID(), At: mDay, Amount: 5_000}},
		Payouts: []models.Payout{{ID: primitive.NewObjectID(), ReceivedAt: mDay,
			Gross: 100_000, Commission: 8_000, Net: 92_000, Provider: "uzum"}},
		// None of these is profit or loss.
		Collections: []models.Collection{{ID: primitive.NewObjectID(), At: mDay, Amount: 60_000, To: "bank"}},
		Advances:    []models.StaffAdvance{{ID: primitive.NewObjectID(), At: mDay, Amount: 200_000, Kind: models.AdvanceOut}},
		CashEntries: []models.CashEntry{{ID: primitive.NewObjectID(), At: mDay, Amount: 9_000, Kind: models.CashOut, Category: "mahsulot"}},
	}
	entries := moneyEntries(src, mFrom, mTo)
	_, net := moneyTotals(entries)

	_, in, out, _, _ := financeLines(orders, "uz", nil)
	ext, _ := externalDeliveryCost(orders)
	out += ext + 30_000 + 20_000 + 10_000 + 5_000 + 8_000
	if net != in-out {
		t.Fatalf("ledger P&L net %d, money report in−out %d", net, in-out)
	}
}

func TestMoneyClassesKeepPlacesOutOfProfit(t *testing.T) {
	adv := models.StaffAdvance{ID: primitive.NewObjectID(), At: mDay, Amount: 200_000, Kind: models.AdvanceOut, StaffName: "Bozorchi"}
	col := models.Collection{ID: primitive.NewObjectID(), At: mDay, Amount: 60_000, To: "bank"}
	pay := models.Payout{ID: primitive.NewObjectID(), ReceivedAt: mDay, Commission: 8_000, Net: 91_000}
	// The safe half of a wage: already the salary entry, must not be a second.
	mirror := models.SafeEntry{ID: primitive.NewObjectID(), At: mDay, Amount: 10_000, Kind: models.SafeOut,
		RefKind: models.SafeRefSalary, RefID: primitive.NewObjectID()}
	manual := models.SafeEntry{ID: primitive.NewObjectID(), At: mDay, Amount: 7_000, Kind: models.SafeIn, Category: "qaytim"}

	got := byID(moneyEntries(&moneySources{
		Advances: []models.StaffAdvance{adv}, Collections: []models.Collection{col},
		Payouts: []models.Payout{pay}, SafeEntries: []models.SafeEntry{mirror, manual},
	}, mFrom, mTo))

	for id, class := range map[string]string{
		"advance_" + adv.ID.Hex():    MoneyAdvance,
		"collection_" + col.ID.Hex(): MoneyTransfer,
		"payout_" + pay.ID.Hex():     MoneyTransfer,
		"safe_" + manual.ID.Hex():    MoneyManual,
	} {
		e, ok := got[id]
		if !ok || e.Class != class || e.PnL {
			t.Errorf("%s: %+v, want class %s outside P&L", id, e, class)
		}
	}
	// ⚠️ The aggregator's cut is a cost; the arrival is not income.
	if c := got["commission_"+pay.ID.Hex()]; c.Class != MoneyCommission || !c.PnL || c.Amount != 8_000 {
		t.Errorf("commission: %+v", c)
	}
	if got["payout_"+pay.ID.Hex()].Amount != 91_000 {
		t.Error("net must be the aggregator's own figure, not gross − commission")
	}
	if _, ok := got["safe_"+mirror.ID.Hex()]; ok {
		t.Error("a safe entry mirroring another document was listed twice")
	}
}

// A refund given in October for a September sale belongs to October.
func TestMoneyRefundOnItsOwnDay(t *testing.T) {
	o := mOrder(40_000, models.PayRefunded, models.StatusDelivered)
	o.Refund = &models.CheckRefund{At: mTo.Add(48 * time.Hour), Amount: 40_000}
	sep := byID(moneyEntries(&moneySources{Orders: []models.Order{o}}, mFrom, mTo))
	if _, ok := sep["sale_"+o.ID.Hex()]; !ok {
		t.Error("the sale left September")
	}
	if _, ok := sep["refund_"+o.ID.Hex()]; ok {
		t.Error("the refund was dated September")
	}
	oct := byID(moneyEntries(&moneySources{Orders: []models.Order{o}}, mTo, mTo.AddDate(0, 1, 0)))
	if _, ok := oct["refund_"+o.ID.Hex()]; !ok {
		t.Error("the refund is missing from October")
	}
}

// ⚠️ An unpaid debt is food eaten and money not received — nothing to book.
func TestMoneyUnpaidDebtIsNotRevenue(t *testing.T) {
	o := mOrder(30_000, models.PayUnpaid, models.StatusDelivered)
	o.PaymentMethod = models.MethodDebt
	if n := len(moneyEntries(&moneySources{Orders: []models.Order{o}}, mFrom, mTo)); n != 0 {
		t.Fatalf("%d entries for an unpaid debt", n)
	}
}

// The ledger names suppliers and staff, never a guest.
func TestMoneyLedgerCarriesNoCustomer(t *testing.T) {
	o := mOrder(100_000, models.PayPaid, models.StatusDelivered)
	b, _ := json.Marshal(moneyEntries(&moneySources{Orders: []models.Order{o}}, mFrom, mTo))
	for _, leak := range []string{"Aziz", "998901234567", "customer"} {
		if strings.Contains(string(b), leak) {
			t.Errorf("%q in the ledger: %s", leak, b)
		}
	}
	if strings.Contains(string(b), "null") || strings.Contains(string(b), "000000000000000000000000") {
		t.Errorf("null or zero id: %s", b)
	}
}

// The day is the restaurant's, computed on the server: 23:30 in Tashkent is
// 18:30 UTC the same day, 01:00 is the previous day in UTC.
func TestMoneyDayIsLocal(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		t.Skip(err)
	}
	old := time.Local
	time.Local = loc
	defer func() { time.Local = old }()
	early := time.Date(2026, 9, 10, 1, 0, 0, 0, loc).UTC()
	if got := moneyDay(early); got != "2026-09-10" {
		t.Fatalf("day %s, want 2026-09-10", got)
	}
}
