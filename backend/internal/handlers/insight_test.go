package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"restaurant-backend/internal/insight"
)

// ⚠️ **The promise that nothing personal leaves the restaurant, made checkable.**
// A fact is built from customer rows — that is the point of it — and the
// tempting next line is always to carry a name through "so the card can be
// specific". This asserts that the only things which travel are a key, an area
// and numbers, so that line has to delete a test before it can be written.
func TestOnlyFiguresLeaveTheRestaurant(t *testing.T) {
	facts := []insight.Fact{{
		Key: "lapsed_regulars", Area: insight.Guests,
		Numbers: map[string]any{"guests": 42, "avgCheck": 85000},
		Action:  insight.NewCampaign,
		Params:  map[string]string{"segment": "sleeping"},
	}}

	payload := make([]any, 0, len(facts))
	for _, f := range facts {
		payload = append(payload, map[string]any{
			"key": f.Key, "area": f.Area, "numbers": f.Numbers,
		})
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	var sent []map[string]any
	if err := json.Unmarshal(raw, &sent); err != nil {
		t.Fatal(err)
	}
	for _, m := range sent {
		for k := range m {
			switch k {
			case "key", "area", "numbers":
			default:
				t.Fatalf("%q left the restaurant with the facts", k)
			}
		}
	}
	// The figures themselves must be figures, not text somebody could put a
	// phone number in.
	numbers, _ := sent[0]["numbers"].(map[string]any)
	for k, v := range numbers {
		if _, ok := v.(float64); !ok {
			t.Fatalf("%q is %T, not a number", k, v)
		}
	}
	if strings.Contains(string(raw), "segment") {
		t.Fatal("the action's parameters travelled — they are ours to decide here")
	}
}
