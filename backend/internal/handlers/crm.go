package handlers

import (
	"math"
	"sort"
	"time"

	"restaurant-backend/internal/models"
)

// Customer segments.
//
// Every segment here has to be explainable in one sentence to the person who
// will act on it — an owner will not send anything to a group they cannot
// describe. So the rules are plain counts and dates, not a score.
//
// The thresholds are days, and they are the argument, not the code: "asleep"
// means different things to a pizza place and to a banquet hall. They live here
// as named constants so there is one place to change them, and they are stated
// in the panel so nobody has to guess what a label means.
const (
	// A customer counts as new while their first order is still recent.
	newDays = 30
	// Ordering within this window is what "active" means.
	activeDays = 30
	// Nothing for this long, from someone who used to order: worth a nudge.
	sleepingDays = 60
	// Long enough that they have probably found somewhere else.
	lostDays = 180
	// How many orders make someone a regular rather than a passer-by.
	regularOrders = 3
	// Birthday greetings go out this far in advance.
	birthdayWindowDays = 7
	// The share of the customer base, by money spent, that counts as VIP.
	// Relative on purpose: a fixed "1 000 000 so'm" means something different
	// in a samsa shop and a banquet hall, and would quietly rot as prices rise.
	vipTopPercent = 10
)

// Segment ids. The panel translates them; these are stable.
const (
	SegNew      = "new"
	SegRegular  = "regular"
	SegVIP      = "vip"
	SegSleeping = "sleeping"
	SegLost     = "lost"
	SegBirthday = "birthday"
	// Left a complaint nobody has answered yet. The only segment that is about
	// the restaurant's own behaviour rather than the customer's.
	SegUnhappy = "unhappy"
	// Has an account but has never ordered.
	SegNoOrders = "noOrders"
)

// customerFacts is what the segment rules are decided from.
type customerFacts struct {
	OrdersCount int
	OrdersTotal int
	FirstOrder  *time.Time
	LastOrder   *time.Time
	Birthday    string
	// An open complaint from the last couple of months.
	Unhappy bool
}

// segmentsFor labels one customer. A customer can be in several segments at
// once — a VIP who has gone quiet is both, and that combination is exactly the
// one worth acting on, so it must not be collapsed into a single label.
//
// ⚠️ `rfmCell` is this customer's RFM position, already computed against the
// whole base — passed in rather than derived here, because the cut points are a
// fact about the *base* and cannot be worked out from one customer's row. Empty
// when the base is too small to rank (see `rfmMinBase`), in which case the
// customer simply has no RFM label and every rule segment behaves as before.
func segmentsFor(f customerFacts, vipFloor int, now time.Time, rfmCell string) []string {
	var out []string

	if f.OrdersCount == 0 {
		out = append(out, SegNoOrders)
	} else {
		daysSinceLast := math.MaxInt32
		if f.LastOrder != nil {
			daysSinceLast = int(now.Sub(*f.LastOrder).Hours() / 24)
		}
		switch {
		case daysSinceLast >= lostDays:
			out = append(out, SegLost)
		case daysSinceLast >= sleepingDays:
			out = append(out, SegSleeping)
		}
		if f.FirstOrder != nil && now.Sub(*f.FirstOrder).Hours() < float64(newDays*24) {
			out = append(out, SegNew)
		}
		if f.OrdersCount >= regularOrders && daysSinceLast < activeDays {
			out = append(out, SegRegular)
		}
		// vipFloor is 0 when there is not enough of a customer base to rank.
		if vipFloor > 0 && f.OrdersTotal >= vipFloor {
			out = append(out, SegVIP)
		}
	}

	if birthdaySoon(f.Birthday, now) {
		out = append(out, SegBirthday)
	}
	if f.Unhappy {
		out = append(out, SegUnhappy)
	}
	// Namespaced, and beside the rules rather than instead of them: a customer
	// is in every rule segment that describes them **and** in exactly one RFM
	// cell. "Sleeping VIP who is at risk" is three true statements about one
	// person, and collapsing them into one label was the thing not to do.
	if rfmCell != "" {
		out = append(out, rfmSegmentID(rfmCell))
	}
	return out
}

// birthdaySoon reports whether a "MM-DD" birthday falls within the next few
// days. Wraps around the new year, which is the case a naive string compare
// gets wrong every 31 December.
func birthdaySoon(birthday string, now time.Time) bool {
	if len(birthday) != 5 || birthday[2] != '-' {
		return false
	}
	for i := 0; i < birthdayWindowDays; i++ {
		if now.AddDate(0, 0, i).Format("01-02") == birthday {
			return true
		}
	}
	return false
}

// vipFloor is the amount a customer must have spent to sit in the top decile.
//
// Returns 0 when the base is too small to rank meaningfully: calling the single
// customer of a brand-new restaurant a VIP tells the owner nothing.
func vipFloor(totals []int) int {
	if len(totals) < 10 {
		return 0
	}
	sorted := append([]int(nil), totals...)
	sort.Sort(sort.Reverse(sort.IntSlice(sorted)))
	cut := len(sorted) * vipTopPercent / 100
	if cut < 1 {
		cut = 1
	}
	floor := sorted[cut-1]
	if floor <= 0 {
		return 0
	}
	return floor
}

// sourceFromOrder guesses where a customer came from, the first time they
// order. A guess the operator can correct beats an empty field nobody fills in.
func sourceFromOrder(order *models.Order) string {
	switch {
	case order.TakenBy != "":
		// An operator typed it, so the guest reached the restaurant by ringing
		// it — whatever they may browse later.
		return "phone"
	case order.Type == "dinein":
		return "qr"
	}
	return "site"
}
