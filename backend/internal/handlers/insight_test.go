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

// ⚠️ **One branch means there was never a choice to make, and half the briefing
// depended on somebody making it.**
//
// The stock facts need a single branch — a shortfall spread across three
// fridges is a number nobody can act on, the rule the whole stock module is
// built on. But an owner of a one-branch restaurant is pinned to no branch, so
// their scope arrives as zero and those facts were skipped in silence. The two
// most reliable things the briefing has to say — an uncounted store and an
// unexplained shortfall — never appeared, and the panel drew nothing at all.
func TestASingleBranchResolvesItself(t *testing.T) {
	src := readLossSource(t, "insightapi.go")
	if !strings.Contains(src, "h.onlyBranch(r, scope.BrandID)") {
		t.Fatal("a one-branch restaurant gets no stock facts again")
	}
}

// ⚠️ **An empty answer is not stored, and storing it was a day-long mistake.**
//
// The reasoning was that "nothing to say" should not be recomputed all morning.
// But gathering facts is a handful of Mongo queries — the expensive part is the
// model, which is not reached at all when there are no facts — and writing
// "nothing" into the day's slot meant a restaurant that turned the feature on
// at nine saw an empty panel until the following morning, with nothing anywhere
// saying why.
func TestNothingToSayIsNotCachedForTheDay(t *testing.T) {
	src := readLossSource(t, "insightapi.go")
	i := strings.Index(src, "if len(facts) == 0 {")
	if i < 0 {
		t.Fatal("the empty-facts path is gone")
	}
	branch := src[i : i+900]
	if strings.Contains(branch, "h.storeBriefing(") {
		t.Fatal("an empty briefing is being kept for the rest of the day")
	}
}

// ⚠️ **A reported error is not an answer, and treating it as one cost a whole
// day.**
//
// The control plane answers 200 with an `error` field when the model could not
// be reached — a quota, a dead key, a network — because the panel's own figures
// do not depend on it and a red banner over a working dashboard is worse than
// no cards. But the tenant read any 200 as "the platform answered", stored the
// empty result for the day, and the briefing then stayed blank until tomorrow
// over a rate limit that cleared in eighteen seconds.
func TestAFailedCallIsNotCachedForTheDay(t *testing.T) {
	src := readLossSource(t, "insightapi.go")
	if !strings.Contains(src, `if msg, _ := res["error"].(string); msg != "" {`) {
		t.Fatal("a platform error is being stored as an empty briefing again")
	}
	// It must return *before* the answered flag is set, or the store still runs.
	fail := strings.Index(src, `res["error"].(string)`)
	answered := strings.Index(src, "state.answered = true")
	if fail < 0 || answered < 0 || fail > answered {
		t.Fatal("the error check runs after the answer is accepted")
	}
	// ⚠️ And the reason travels: "quota exceeded, retry in 18s" is a completely
	// different morning from "no key configured", and an owner who enabled this
	// and sees nothing needs the sentence.
	if !strings.Contains(src, `out["error"] = state.failed`) {
		t.Fatal("the reason is dropped before it reaches anybody")
	}
}
