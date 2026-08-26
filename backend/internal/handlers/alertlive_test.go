package handlers

// ---- Every alert trigger, against a real database ----
//
// ⚠️ **These exist because three of the six kinds shipped not firing, and every
// one of them was a different silence.**
//
//   - The recipe and panel-action alerts read settings under a zero branch and
//     found none, so they returned before anything was recorded.
//   - The cancelled-check alert did not exist: every trigger hung on the close
//     path, which a cancelled check never reaches.
//   - The discount alert read `o.Discounts` while the handler had written the
//     discount into the update map and never onto the order.
//
// Not one of them failed a test, because every test asserted the *shape* of the
// code — the right constant, the right ordering, the right comparison — and
// none of them ran the thing. A source-inspecting test cannot see that a struct
// was never filled in.
//
// ⚠️ **So these run the real functions against a real Mongo.** They are skipped
// when there is none, because a test that cannot run must not be a test that
// fails — but on any machine with a database they are the only place that
// answers "does an alert actually come out of this".

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"restaurant-backend/internal/config"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// liveHandler is a Handler pointed at a throwaway database.
//
// ⚠️ Its own database per run, dropped at the end: these write orders and
// settings, and a test that shares a database with a developer's own data is a
// test somebody eventually stops running.
func liveHandler(t *testing.T) (*Handler, primitive.ObjectID) {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Skipf("no mongo at %s: %v", uri, err)
	}
	if err := cli.Ping(ctx, nil); err != nil {
		t.Skipf("no mongo at %s: %v", uri, err)
	}
	name := "keel_alert_test_" + primitive.NewObjectID().Hex()
	db := cli.Database(name)
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = db.Drop(c)
		_ = cli.Disconnect(c)
	})

	branch := primitive.NewObjectID()
	h := &Handler{Store: repository.New(db), Cfg: &config.Config{}}
	// Alerts on, thresholds at the shipped defaults.
	if _, err := h.Store.AlertSettings.InsertOne(context.Background(),
		models.AlertSettings{BranchID: branch, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	return h, branch
}

// raised waits for the goroutine `raiseAlert` starts and returns what it wrote.
//
// ⚠️ Polled rather than slept once: a fixed sleep is either flaky on a slow
// machine or wasted time on a fast one, and this runs on both.
func raised(t *testing.T, h *Handler, kind models.AlertKind) *models.LossAlert {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var a models.LossAlert
		err := h.Store.LossAlerts.FindOne(context.Background(),
			bson.M{"kind": kind}).Decode(&a)
		if err == nil {
			return &a
		}
		time.Sleep(25 * time.Millisecond)
	}
	return nil
}

// ⚠️ **The bug that shipped: the discount went into the update map and never
// onto the order.** The database had it, the receipt printed it, the response
// carried it — and the alert was handed an empty slice. No shape test could see
// that; this one runs the function against an order built the way the handler
// builds it.
func TestADiscountOnAnOrderRaisesAnAlert(t *testing.T) {
	h, branch := liveHandler(t)
	set := h.alertSettingsOf(context.Background(), branch)

	o := &models.Order{
		ID: primitive.NewObjectID(), BranchID: branch, Number: "T-1",
		TableNumber: "6",
		Discounts: []models.OrderDiscount{{
			Name: "Kassa chegirmasi: test", Amount: 585_000,
			ByID: primitive.NewObjectID(), By: "Yusuf", Reason: "test",
		}},
	}
	h.alertOnDiscount(o, set)

	a := raised(t, h, models.AlertBigDiscount)
	if a == nil {
		t.Fatal("a 585 000 discount with a name on it raised nothing")
	}
	if a.Amount != 585_000 || a.By != "Yusuf" {
		t.Fatalf("the alert lost the facts: %+v", a)
	}
}

// ⚠️ A discount nobody chose measures nobody: a promotion that matched, or
// points spent, has no name and must not put the busiest cashier at the top of
// a list about judgement.
func TestARuleDrivenDiscountRaisesNothing(t *testing.T) {
	h, branch := liveHandler(t)
	set := h.alertSettingsOf(context.Background(), branch)

	h.alertOnDiscount(&models.Order{
		ID: primitive.NewObjectID(), BranchID: branch,
		Discounts: []models.OrderDiscount{{Name: "Aksiya", Amount: 585_000}},
	}, set)

	// Give the goroutine the same chance the positive case gets, so a pass here
	// means "nothing arrived", not "we did not wait".
	time.Sleep(400 * time.Millisecond)
	n, err := h.Store.LossAlerts.CountDocuments(context.Background(), bson.M{})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("a promotion was reported as somebody's judgement (%d alerts)", n)
	}
}

// ⚠️ **The case the feature was asked for and shipped without.** Every trigger
// hung on the close path, which a cancelled check never reaches.
func TestACancelledCheckWithCookedFoodRaisesAnAlert(t *testing.T) {
	h, branch := liveHandler(t)
	fired := time.Now()
	who := actor{ByID: primitive.NewObjectID(), By: "Boris"}

	h.alertOnCancelledCheck(&models.Order{
		ID: primitive.NewObjectID(), BranchID: branch, TableNumber: "4",
		Check: &models.OrderCheck{OpenedAt: fired},
		Items: []models.OrderItem{
			{Name: "Lag'mon", Price: 85_000, Qty: 2, FiredAt: &fired},
		},
	}, who, cancelCheckRequest{Reason: "test"})

	a := raised(t, h, models.AlertCheckCancelled)
	if a == nil {
		t.Fatal("a cancelled check with cooked food on it raised nothing")
	}
	if a.Amount != 170_000 || a.By != "Boris" {
		t.Fatalf("the alert lost the facts: %+v", a)
	}
}

// ⚠️ A table opened by mistake and closed again is the commonest cancellation
// in any restaurant. Alerting on it would put a message on somebody's phone
// several times a day, which is how the ones that matter stop being read.
func TestAnEmptyCancelledCheckRaisesNothing(t *testing.T) {
	h, branch := liveHandler(t)

	h.alertOnCancelledCheck(&models.Order{
		ID: primitive.NewObjectID(), BranchID: branch,
		Check: &models.OrderCheck{OpenedAt: time.Now()},
	}, actor{By: "Boris"}, cancelCheckRequest{Reason: "xato ochildi"})

	time.Sleep(400 * time.Millisecond)
	if n, _ := h.Store.LossAlerts.CountDocuments(context.Background(), bson.M{}); n != 0 {
		t.Fatalf("an empty check raised %d alerts", n)
	}
}

// ⚠️ Cooked food, whether or not a bill was printed first. This required a
// precheck once, which meant a restaurant could void a main course and hear
// nothing — and be right to think the feature did not work.
func TestVoidingCookedFoodRaisesAnAlert(t *testing.T) {
	h, branch := liveHandler(t)
	set := h.alertSettingsOf(context.Background(), branch)
	at := time.Now()

	h.alertOnVoidsAfterPrecheck(&models.Order{
		ID: primitive.NewObjectID(), BranchID: branch, TableNumber: "2",
		Check: &models.OrderCheck{OpenedAt: at},
		Items: []models.OrderItem{{
			Name: "Sho'rva", Price: 120_000, Qty: 1, FiredAt: &at,
			Void: &models.CheckLineVoid{At: at, By: "Aziz", Reason: "mehmon qaytardi"},
		}},
	}, set)

	a := raised(t, h, models.AlertVoidAfterPrecheck)
	if a == nil {
		t.Fatal("voiding a cooked dish raised nothing")
	}
	if a.Amount != 120_000 || a.Reason != "mehmon qaytardi" {
		t.Fatalf("the alert lost the facts: %+v", a)
	}
}

// ⚠️ A dish nobody cooked is a typo being corrected. Alerting on it would fire
// on every kitchen mistake in the building.
func TestVoidingAnUncookedLineRaisesNothing(t *testing.T) {
	h, branch := liveHandler(t)
	set := h.alertSettingsOf(context.Background(), branch)
	at := time.Now()

	h.alertOnVoidsAfterPrecheck(&models.Order{
		ID: primitive.NewObjectID(), BranchID: branch,
		Check: &models.OrderCheck{OpenedAt: at},
		Items: []models.OrderItem{{
			Name: "Choy", Price: 120_000, Qty: 1, // never fired
			Void: &models.CheckLineVoid{At: at, By: "Aziz"},
		}},
	}, set)

	time.Sleep(400 * time.Millisecond)
	if n, _ := h.Store.LossAlerts.CountDocuments(context.Background(), bson.M{}); n != 0 {
		t.Fatalf("a line nobody cooked raised %d alerts", n)
	}
}

// ⚠️ **The zero branch, which silently switched two kinds off entirely.** A
// recipe belongs to the brand and a panel action to whoever is logged in, so
// neither has a branch — and an owner of the whole company is pinned to none.
// Reading zero found no settings, the defaults said "disabled", and the event
// was dropped before it was even recorded.
func TestACompanyWideEventFindsTheSettings(t *testing.T) {
	h, _ := liveHandler(t)

	got := h.alertSettingsOf(context.Background(), primitive.NilObjectID)
	if !got.Enabled {
		t.Fatal("a company-wide event still cannot find the restaurant's settings")
	}
	if got.VoidFrom != models.DefaultVoidFrom {
		t.Fatalf("the thresholds came back unset: %+v", got)
	}
}

// A branch that has switched alerts off is not overridden by another branch
// that has them on: its own document wins.
func TestABranchWithItsOwnSettingsKeepsThem(t *testing.T) {
	h, _ := liveHandler(t)
	quiet := primitive.NewObjectID()
	if _, err := h.Store.AlertSettings.InsertOne(context.Background(),
		models.AlertSettings{BranchID: quiet, Enabled: false}); err != nil {
		t.Fatal(err)
	}
	if h.alertSettingsOf(context.Background(), quiet).Enabled {
		t.Fatal("a branch that switched alerts off was overruled by another branch")
	}
}

// ⚠️ **An Uzbek word was being baked into stored data and read out in a Russian
// message.** "(X tasdiqladi)" was appended to the reason for the receipt's
// label — which is right for a slip of paper and wrong for a notification, and
// the notification is where it ended up. Who approved something is a field, and
// fields are worded at send time.
func TestTheAlertKeepsTheReasonAsItWasTyped(t *testing.T) {
	h, branch := liveHandler(t)
	fired := time.Now()

	h.alertOnCancelledCheck(&models.Order{
		ID: primitive.NewObjectID(), BranchID: branch,
		Number: "YAA8-55D6", TableNumber: "6",
		Check: &models.OrderCheck{OpenedAt: fired, PrecheckAt: &fired},
		Items: []models.OrderItem{{Name: "Osh", Price: 90_000, Qty: 1, FiredAt: &fired}},
	}, actor{By: "Boris", AuthBy: "Yusuf"}, cancelCheckRequest{Reason: "mehmon ketdi"})

	a := raised(t, h, models.AlertCheckCancelled)
	if a == nil {
		t.Fatal("nothing was raised")
	}
	if a.Reason != "mehmon ketdi" {
		t.Fatalf("the reason was decorated: %q", a.Reason)
	}
	// ⚠️ The number is what somebody types into a search box an hour later —
	// "6-stol" alone names a table that has had nine checks today.
	if a.Number != "YAA8-55D6" || a.Table != "6" {
		t.Fatalf("the check cannot be found from the message: %+v", a)
	}
	if !a.AfterPrecheck {
		t.Fatal("the strongest fact about this cancellation was dropped")
	}

	// ⚠️ And the whole message reads in the group's language, with the facts
	// unchanged inside it.
	ru := alertText(*a, "B5 Somsa", "ru")
	for _, want := range []string{"Стол 6", "#YAA8-55D6", "Boris", "Yusuf",
		"подтвердил", "mehmon ketdi", "счёт уже был распечатан"} {
		if !strings.Contains(ru, want) {
			t.Fatalf("%q missing from the Russian message:\n%s", want, ru)
		}
	}
	if strings.Contains(ru, "tasdiqladi") || strings.Contains(ru, "-stol") {
		t.Fatalf("an Uzbek word survived into the Russian message:\n%s", ru)
	}
}
