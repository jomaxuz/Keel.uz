package meta

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

// ⚠️ **A phone that reaches Meta without a country code matches nobody, and
// nothing says so.** The event is accepted, the order is simply never
// attributed, and the whole section then looks like it does not work rather
// than like it is misconfigured. Uzbek numbers are stored at least three ways
// in this database and all three are one person.
func TestUzbekNumbersReachMetaInOneShape(t *testing.T) {
	want := "998901234567"
	for _, in := range []string{
		"+998 90 123 45 67", "998901234567", "90 123 45 67", "901234567",
		"(998) 90-123-45-67", "0901234567",
	} {
		if got := phoneDigits(in); got != want {
			t.Fatalf("%q normalised to %q, want %q — this order will be "+
				"reported and never attributed", in, got, want)
		}
	}
}

// ⚠️ **Nothing identifying a guest may leave this server in the clear.** The
// person ordered dinner; they did not agree to be named to Meta. What goes is
// a hash of a normalised string, which is all Meta matches on anyway.
func TestNothingAboutAGuestLeavesUnhashed(t *testing.T) {
	u := Person("Aziz Karimov", "+998901234567", "Toshkent")
	sum := sha256.Sum256([]byte("998901234567"))
	if len(u.Phone) != 1 || u.Phone[0] != hex.EncodeToString(sum[:]) {
		t.Fatal("the phone number did not reach Meta as a SHA-256 of the " +
			"normalised number")
	}
	for _, field := range [][]string{u.Phone, u.FirstName, u.City, u.Country, u.ExternalID} {
		for _, v := range field {
			if len(v) != 64 || strings.ContainsAny(v, "+ @") {
				t.Fatalf("%q is not a hash — something about a guest is "+
					"travelling in the clear", v)
			}
		}
	}
}

// ⚠️ **An empty value has a hash too, and sending it is sending a field that
// matches nobody while looking like data.** Absent is the correct shape.
func TestAnEmptyFieldIsAbsentRatherThanHashed(t *testing.T) {
	u := Person("", "", "")
	if u.Phone != nil || u.FirstName != nil || u.City != nil || u.ExternalID != nil {
		t.Fatal("an empty field was hashed and sent")
	}
}

// ⚠️ **Meta counts money in the account currency's minor units.** Fifty
// thousand is $500.00 on a dollar account and ₩50,000 on a won one; getting
// this wrong does not fail, it spends a hundred times what was typed.
func TestMinorUnitsKnowsTheCurrenciesWithNoCents(t *testing.T) {
	if MinorUnits("USD") != 100 || MinorUnits("usd") != 100 {
		t.Fatal("a dollar account stopped being counted in cents")
	}
	for _, zero := range []string{"JPY", "KRW", "COP", "VND"} {
		if MinorUnits(zero) != 1 {
			t.Fatalf("%s was given a minor unit it does not have", zero)
		}
	}
	// An unknown code falls back to cents, which is the common case — and the
	// account's own `min_daily_budget` catches the rest before money moves.
	if MinorUnits("") != 100 {
		t.Fatal("an unknown currency stopped defaulting to cents")
	}
}

// ⚠️ **Revoked and rate-limited are different answers to different questions.**
// One means reconnect the account, the other means wait an hour; a single
// "Meta error" sentence sends the owner to neither.
func TestTheTwoFailuresThatNeedDifferentAnswersStayApart(t *testing.T) {
	revoked := &Error{Code: 190}
	if !revoked.Revoked() || revoked.RateLimited() {
		t.Fatal("a revoked token is no longer told apart from a rate limit")
	}
	for _, code := range []int{4, 17, 32, 613, 80000, 80004, 80014} {
		if e := (&Error{Code: code}); !e.RateLimited() {
			t.Fatalf("code %d is no longer read as a rate limit", code)
		}
	}
	if (&Error{Code: 100}).RateLimited() {
		t.Fatal("an ordinary bad-request became a rate limit")
	}
}

// ⚠️ **A country and a place inside it may not travel together**: Meta rejects
// the pair as an overlap, and the message does not say which two fields
// overlapped.
func TestTargetingSendsACircleAndNoCountry(t *testing.T) {
	spec := Targeting{Lat: 41.31, Lng: 69.24, RadiusKm: 5}.spec()
	geo, _ := spec["geo_locations"].(map[string]any)
	if geo == nil || geo["countries"] != nil {
		t.Fatal("a country reached the targeting spec beside a custom location")
	}
	if geo["custom_locations"] == nil {
		t.Fatal("the circle around the kitchen is gone")
	}
	auto, _ := spec["targeting_automation"].(map[string]any)
	if auto == nil || auto["advantage_audience"] != 1 {
		t.Fatal("Meta is no longer asked to choose the audience — which is " +
			"the API flag a targetolog charges for pressing")
	}
}

// ⚠️ **Meta refuses a campaign that does not say whether its ad sets may share
// a budget**, whenever the budget is on the ad set rather than the campaign —
// which is always, here. It arrives as a message about a field nobody has
// heard of, at the moment an owner presses the button that spends money.
//
// ⚠️ **And the answer is false.** True lets ad sets lend each other a fifth of
// their budget; ours carry one ad set, and a ceiling the owner set does not get
// to be approximate.
func TestACampaignSaysWhetherItsAdSetsShareBudget(t *testing.T) {
	src, err := os.ReadFile("campaign.go")
	if err != nil {
		t.Fatal(err)
	}
	fn := string(src)
	at := strings.Index(fn, "func (c *Client) CreateCampaign")
	if at < 0 {
		t.Fatal("CreateCampaign is gone")
	}
	body := fn[at:]
	if end := strings.Index(body, "\n}\n"); end > 0 {
		body = body[:end]
	}
	if !strings.Contains(body, `"is_adset_budget_sharing_enabled": {"false"}`) {
		t.Fatal("the campaign no longer states that its ad sets do not share " +
			"a budget; Meta refuses the create with a message about a field " +
			"the owner has never heard of")
	}
}

// ⚠️ **An ad set says how it bids, because an unstated strategy is whatever
// the account was last set to.** An account defaulting to a bid cap refuses
// the create — "For bid cap you must provide bid amount field" — at the button
// that spends money, about a setting nobody on that screen has seen.
//
// ⚠️ **And it bids without a cap.** A cap is a promise about what one result
// may cost, and a restaurant advertising for the first time has no number to
// make it with; set too low it spends nothing while looking like it is
// running. The ceiling this product enforces is the daily budget.
func TestAnAdSetSaysHowItBids(t *testing.T) {
	src, err := os.ReadFile("campaign.go")
	if err != nil {
		t.Fatal(err)
	}
	fn := string(src)
	at := strings.Index(fn, "func (c *Client) CreateAdSet")
	if at < 0 {
		t.Fatal("CreateAdSet is gone")
	}
	body := fn[at:]
	if end := strings.Index(body, "\n}\n"); end > 0 {
		body = body[:end]
	}
	if !strings.Contains(body, `"bid_strategy": {"LOWEST_COST_WITHOUT_CAP"}`) {
		t.Fatal("the ad set no longer states its bid strategy; an account " +
			"defaulting to a bid cap will refuse every campaign")
	}
	if strings.Contains(body, "bid_amount") {
		t.Fatal("a bid cap crept in — this product has no number to set one with")
	}
}
