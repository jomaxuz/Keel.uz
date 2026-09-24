package handlers

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// ⚠️ **Every write that grows an order's statusHistory raises a webhook.**
//
// There is no single place an order changes status — the site, the till, the
// kitchen, the courier, Uzum, the offline sync and the merge each write it
// themselves — and no change stream to notice them all (single mongod). So the
// rule is a count, per file: at least as many `orderEvent` calls as writes that
// create an order or push onto its history. A new write without one is a
// receiver that silently never hears about that kind of change.
func TestEveryStatusChangeRaisesAWebhook(t *testing.T) {
	writes := regexp.MustCompile(`Store\.Orders\.InsertOne\(|\{"statusHistory":`)
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, f := range files {
		name := f.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		// A booking has a history of its own, and it is not an order.
		if name == "reservations.go" {
			continue
		}
		src := readSource(t, name)
		n := len(writes.FindAllString(src, -1))
		if n == 0 {
			continue
		}
		checked++
		if got := strings.Count(src, "h.orderEvent("); got < n {
			t.Errorf("%s: %d order writes that change status, %d orderEvent calls", name, n, got)
		}
	}
	if checked < 8 {
		t.Fatalf("only %d files matched — did the write shape change?", checked)
	}
}

func TestOrderEventsOnlySinceTheEndpointExisted(t *testing.T) {
	base := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	o := &models.Order{StatusHistory: []models.StatusEvent{
		{Status: models.StatusPending, At: base},
		{Status: models.StatusConfirmed, At: base.Add(time.Minute)},
		{Status: models.StatusPreparing, At: base.Add(2 * time.Minute)},
	}}
	if got := orderEventsFor(o, base.Add(90*time.Second)); len(got) != 1 || got[0] != 2 {
		t.Fatalf("got %v", got)
	}
	// ⚠️ Both of two quick changes, not only the last.
	if got := orderEventsFor(o, base.Add(30*time.Second)); len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	// A status set twice is one change.
	o.StatusHistory = append(o.StatusHistory, models.StatusEvent{Status: models.StatusPreparing, At: base.Add(3 * time.Minute)})
	if got := orderEventsFor(o, base.Add(90*time.Second)); len(got) != 1 {
		t.Fatalf("a repeated status raised an event: %v", got)
	}
	if eventTypeOf(0) != models.EventOrderCreated || eventTypeOf(2) != models.EventOrderStatusChanged {
		t.Fatal("event types")
	}
	id := primitive.NewObjectID()
	if orderEventID(id, 1) != orderEventID(id, 1) || orderEventID(id, 1) == orderEventID(id, 2) {
		t.Fatal("event id must be stable per entry and distinct between entries")
	}
}

// The published order must not carry the two Go habits (CLAUDE.md §10): an
// unset id as "000…" and an empty list as null. Built from data shaped the way
// Mongo returns it, not from what a test would naturally write.
func TestOpenOrderHasNoZeroIDsOrNulls(t *testing.T) {
	o := &models.Order{
		ID: primitive.NewObjectID(), Number: "1001", Type: "pickup", Status: models.StatusPending,
		Items: []models.OrderItem{{Name: "Osh", Qty: 1, Price: 30000}},
	}
	b, err := json.Marshal(openOrderOf(o))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "000000000000000000000000") {
		t.Errorf("zero id leaked: %s", s)
	}
	if strings.Contains(s, "null") {
		t.Errorf("null in the published order: %s", s)
	}
	if strings.Contains(s, `"address"`) {
		t.Errorf("a pickup with no address must not claim 0,0: %s", s)
	}
}

func TestOpenCursorRoundTrips(t *testing.T) {
	at := time.UnixMilli(1_758_700_000_123)
	id := primitive.NewObjectID()
	gotAt, gotID, ok := parseOpenCursor(openCursor(at, id))
	if !ok || !gotAt.Equal(at) || gotID != id {
		t.Fatalf("got %v %v %v", gotAt, gotID, ok)
	}
	for _, bad := range []string{"", "x", "MTIz", openCursor(at, id) + "!"} {
		if _, _, ok := parseOpenCursor(bad); ok {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestCleanChoicesKeepsOnlyKnownOnce(t *testing.T) {
	got := cleanChoices([]string{"orders:read", "admin", "orders:read", " menu:read "}, models.APIScopes)
	if strings.Join(got, ",") != "menu:read,orders:read" {
		t.Fatalf("got %v", got)
	}
	if got := cleanChoices(nil, models.APIScopes); got == nil {
		t.Fatal("nil, not an empty list")
	}
}

func TestAPIKeyShape(t *testing.T) {
	k, err := newAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(k, apiKeyPrefix) || len(k) < 40 {
		t.Fatalf("key %q", k)
	}
	if apiKeyHash(k) == apiKeyHash(k+"x") || len(apiKeyHash(k)) != 64 {
		t.Fatal("hash")
	}
}
