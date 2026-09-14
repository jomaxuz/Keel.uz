package handlers

import (
	"context"
	"testing"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ Against a real database, like the rest of the live alert tests: each new
// kind fires above its threshold, stays quiet below it, and says nothing about
// what the owner did themselves. Skipped with no Mongo.

func noAlert(t *testing.T, h *Handler, kind models.AlertKind) {
	t.Helper()
	time.Sleep(300 * time.Millisecond)
	n, err := h.Store.LossAlerts.CountDocuments(context.Background(), bson.M{"kind": kind})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("%s raised %d alert(s) where it should have stayed quiet", kind, n)
	}
}

func TestARefundAboveTheThresholdRaisesAnAlert(t *testing.T) {
	h, branch := liveHandler(t)
	o := &models.Order{ID: primitive.NewObjectID(), BranchID: branch, Number: "T-9", TableNumber: "3"}

	h.alertOnRefund(o, models.CheckRefund{At: time.Now(), By: "Aziz", Reason: "sovuq", Amount: 1_000}, false)
	noAlert(t, h, models.AlertCheckRefunded)

	h.alertOnRefund(o, models.CheckRefund{At: time.Now(), By: "Aziz", Reason: "sovuq", Amount: 240_000}, true)
	noAlert(t, h, models.AlertCheckRefunded) // the owner's own refund

	h.alertOnRefund(o, models.CheckRefund{At: time.Now(), By: "Aziz", Reason: "sovuq", Amount: 240_000}, false)
	a := raised(t, h, models.AlertCheckRefunded)
	if a == nil || a.Amount != 240_000 || a.By != "Aziz" || a.Number != "T-9" {
		t.Fatalf("refund alert: %+v", a)
	}
}

func TestAnOrderCancelledAfterCookingRaisesAnAlert(t *testing.T) {
	h, branch := liveHandler(t)
	now := time.Now()
	pending := &models.Order{
		ID: primitive.NewObjectID(), BranchID: branch, Number: "A-1", Total: 90_000,
		StatusHistory: []models.StatusEvent{{Status: models.StatusPending, At: now}, {Status: models.StatusCancelled, At: now}},
	}
	h.alertOnOrderCancelled(pending, "Operator", false)
	noAlert(t, h, models.AlertOrderCancelled) // a guest who changed their mind

	cooked := &models.Order{
		ID: primitive.NewObjectID(), BranchID: branch, Number: "A-2", Total: 90_000, CancelReason: "mijoz javob bermadi",
		StatusHistory: []models.StatusEvent{
			{Status: models.StatusPending, At: now}, {Status: models.StatusPreparing, At: now},
			{Status: models.StatusCancelled, At: now},
		},
	}
	h.alertOnOrderCancelled(cooked, "Operator", false)
	a := raised(t, h, models.AlertOrderCancelled)
	if a == nil || a.Number != "A-2" || a.Reason != "mijoz javob bermadi" {
		t.Fatalf("order cancel alert: %+v", a)
	}
}

func TestALargeCashWithdrawalRaisesAnAlert(t *testing.T) {
	h, branch := liveHandler(t)
	h.alertOnCashOut(models.CashEntry{ID: primitive.NewObjectID(), BranchID: branch, Kind: models.CashOut, Amount: 20_000, Category: "Non", At: time.Now()})
	noAlert(t, h, models.AlertCashOut)
	h.alertOnCashOut(models.CashEntry{ID: primitive.NewObjectID(), BranchID: branch, Kind: models.CashIn, Amount: 900_000, Category: "Almashtirish", At: time.Now()})
	noAlert(t, h, models.AlertCashOut) // money in is not money out

	h.alertOnCashOut(models.CashEntry{ID: primitive.NewObjectID(), BranchID: branch, Kind: models.CashOut, Amount: 800_000, Category: "Yetkazib beruvchi", By: "Dilnoza", At: time.Now()})
	a := raised(t, h, models.AlertCashOut)
	if a == nil || a.Amount != 800_000 || a.Subject != "Yetkazib beruvchi" {
		t.Fatalf("cash out alert: %+v", a)
	}
}

func TestALargeWriteOffRaisesAnAlert(t *testing.T) {
	h, branch := liveHandler(t)
	h.alertOnWriteOff(models.WriteOff{ID: primitive.NewObjectID(), BranchID: branch, Value: 400_000, Reason: "buzildi", By: "Ega"}, "Go'sht", true)
	noAlert(t, h, models.AlertBigWriteoff) // the owner's own write-off

	h.alertOnWriteOff(models.WriteOff{ID: primitive.NewObjectID(), BranchID: branch, Value: 400_000, Reason: "buzildi", By: "Omborchi"}, "Go'sht", false)
	a := raised(t, h, models.AlertBigWriteoff)
	if a == nil || a.Subject != "Go'sht" || a.Amount != 400_000 {
		t.Fatalf("write-off alert: %+v", a)
	}
}

func TestALargeDebtRaisesAnAlert(t *testing.T) {
	h, branch := liveHandler(t)
	set := h.alertSettingsOf(context.Background(), branch)
	h.alertOnDebt(&models.Order{ID: primitive.NewObjectID(), BranchID: branch, Total: 10_000}, set, "Kassir", "")
	noAlert(t, h, models.AlertDebtWritten)

	h.alertOnDebt(&models.Order{ID: primitive.NewObjectID(), BranchID: branch, Number: "C-5", Total: 450_000}, set, "Kassir", "juma kuni")
	a := raised(t, h, models.AlertDebtWritten)
	if a == nil || a.Amount != 450_000 || a.Reason != "juma kuni" {
		t.Fatalf("debt alert: %+v", a)
	}
}

// Every new kind has a heading in every language, so a phone never shows the
// "unknown" line for one.
func TestEveryAlertKindHasAHeading(t *testing.T) {
	for _, lang := range []string{"uz", "ru", "en"} {
		for _, k := range models.AlertKindsInOrder {
			if alertTitle(models.LossAlert{Kind: k}, lang) == notifyWordsFor(lang).Unknown {
				t.Fatalf("%s has no heading in %s", k, lang)
			}
		}
	}
}
