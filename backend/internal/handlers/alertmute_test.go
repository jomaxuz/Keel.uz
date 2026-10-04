package handlers

import (
	"context"
	"testing"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

// ⚠️ **Muted means not sent, never not recorded.** Against a real database: a
// muted kind lands in the list the panel reads and is never handed to the
// phone or the chat; a kind left on is. Skipped with no Mongo, like the rest of
// the live alert tests.
func TestAMutedKindIsRecordedButNotSent(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	if _, err := h.Store.AlertSettings.UpdateOne(ctx, bson.M{"branchId": branch},
		bson.M{"$set": bson.M{"muted": []models.AlertKind{models.AlertBigDiscount}}}); err != nil {
		t.Fatal(err)
	}

	h.raiseAlertSync(ctx, models.LossAlert{BranchID: branch, Kind: models.AlertBigDiscount, Amount: 1})
	h.raiseAlertSync(ctx, models.LossAlert{BranchID: branch, Kind: models.AlertCashShort, Amount: 1})

	var muted bson.M
	if err := h.Store.LossAlerts.FindOne(ctx, bson.M{"kind": models.AlertBigDiscount}).Decode(&muted); err != nil {
		t.Fatalf("a muted kind was not recorded: %v", err)
	}
	if muted["sentAt"] != nil || muted["sendErr"] != nil {
		t.Fatalf("a muted kind was handed to delivery: %v", muted)
	}

	var live bson.M
	if err := h.Store.LossAlerts.FindOne(ctx, bson.M{"kind": models.AlertCashShort}).Decode(&live); err != nil {
		t.Fatalf("a kind left on was not recorded: %v", err)
	}
	// With no chat configured in the test database the send fails — which is the
	// proof it was attempted.
	if live["sentAt"] == nil && live["sendErr"] == nil {
		t.Fatalf("a kind left on was never handed to delivery: %v", live)
	}
}

// ⚠️ The test button must ring even for a muted kind, or an owner who muted
// discounts concludes the channel is broken.
func TestTheTestAlertIgnoresMutes(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	if _, err := h.Store.AlertSettings.UpdateOne(ctx, bson.M{"branchId": branch},
		bson.M{"$set": bson.M{"muted": []models.AlertKind{models.AlertBigDiscount}}}); err != nil {
		t.Fatal(err)
	}
	h.deliverAlert(ctx, models.LossAlert{BranchID: branch, Kind: models.AlertBigDiscount, Subject: "Sinov"}, true)

	var a bson.M
	if err := h.Store.LossAlerts.FindOne(ctx, bson.M{"subject": "Sinov"}).Decode(&a); err != nil {
		t.Fatal(err)
	}
	if a["sentAt"] == nil && a["sendErr"] == nil {
		t.Fatalf("the test alert was muted: %v", a)
	}
}

// Unknown names are dropped, and every kind the server raises has a place in
// the settings list.
func TestMutedKindsAndTheListOfKinds(t *testing.T) {
	got := mutedKinds([]models.AlertKind{"big_discount", "typo", "big_discount", "recipe_up"})
	if len(got) != 2 || got[0] != models.AlertBigDiscount || got[1] != models.AlertRecipeUp {
		t.Fatalf("mutedKinds = %v", got)
	}
	for _, k := range []models.AlertKind{
		models.AlertVoidAfterPrecheck, models.AlertBigDiscount, models.AlertCashShort,
		models.AlertStockShort, models.AlertRecipeUp, models.AlertPanelAction,
		models.AlertCheckCancelled, models.AlertShiftOverdue,
	} {
		if !models.IsAlertKind(k) {
			t.Fatalf("%s is raised but has no checkbox", k)
		}
	}
	if (models.AlertSettings{}).WithDefaults().Muted == nil {
		t.Fatal("defaults left muted as nil, which reaches the page as null")
	}
}

// ⚠️ **Past the ceiling: held, then handed to the digest — never dropped.**
// Against a real database. The ceiling is filled with one already-sent record,
// the next alert must come back held rather than silently finished, and the
// sweeper must then try it (no chat is configured here, so the attempt fails —
// which is the proof it was made, and that it will be made again).
func TestAHeldAlertGoesToTheDigest(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	if _, err := h.Store.AlertSettings.UpdateOne(ctx, bson.M{"branchId": branch},
		bson.M{"$set": bson.M{"dailyMax": 1}}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if _, err := h.Store.LossAlerts.InsertOne(ctx, models.LossAlert{
		BranchID: branch, Kind: models.AlertCashOut, At: now, SentAt: &now,
	}); err != nil {
		t.Fatal(err)
	}

	h.raiseAlertSync(ctx, models.LossAlert{
		BranchID: branch, Kind: models.AlertBigDiscount, Amount: 1,
		At: now.Add(-time.Hour),
	})
	var held models.LossAlert
	if err := h.Store.LossAlerts.FindOne(ctx, bson.M{"kind": models.AlertBigDiscount}).Decode(&held); err != nil {
		t.Fatal(err)
	}
	if held.HeldAt == nil || held.SentAt != nil {
		t.Fatalf("past the ceiling the alert was not held: %+v", held)
	}

	h.sweepAlerts(ctx)
	if err := h.Store.LossAlerts.FindOne(ctx, bson.M{"_id": held.ID}).Decode(&held); err != nil {
		t.Fatal(err)
	}
	if held.Attempts != 1 || held.SendErr == "" || held.NextTryAt == nil {
		t.Fatalf("the sweeper never tried the held alert: %+v", held)
	}
	if held.ClaimUntil != nil {
		t.Fatal("a failed digest left its records claimed, so no retry can take them")
	}
}
