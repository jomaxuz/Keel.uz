package handlers

import (
	"math"
	"net/http"
	"sort"
	"time"

	"restaurant-backend/internal/httpx"
)

// RFM: how recently, how often, how much.
//
// The segments already here (§ handlers/crm.go) are **rules** — "nothing for
// sixty days", "three orders or more" — and each is explainable in one
// sentence, which is why an owner will act on them. RFM is a different kind of
// answer and sits **beside** them rather than replacing them: it does not ask
// whether a customer crossed a line, it asks where they stand **relative to
// everybody else** on three axes at once.
//
// That distinction is the whole reason to have both:
//
//   - A rule survives a bad month. "Asleep" means the same thing in January and
//     in July, which is what makes "sleeping VIP" a thing you can chase.
//   - A ranking survives a price change. A restaurant that raised prices 30%
//     has not acquired a room full of big spenders, and a fixed money threshold
//     says it has.
//
// Neither can be derived from the other, and "birthday" and "unhappy" fit in
// neither — they are not purchase behaviour at all, which is why folding the
// old segments into RFM was rejected.
//
// ⚠️ **One customer, one cell.** Unlike the rule segments — where "sleeping"
// and "VIP" together is the most useful thing the panel can say — the RFM cells
// are a partition. A grid position that a customer could occupy twice is not a
// position, and the counts down the side of it would add up to more than the
// customer base.

// RFM cell ids. Stored in nothing, but sent to the panel and usable as a
// campaign audience, so they are stable and never renamed.
const (
	// Recent, frequent. The restaurant's actual business.
	RFMChampions = "champions"
	// Used to order often and has gone quiet. The most valuable thing on this
	// screen: nothing else finds a regular in the act of leaving.
	RFMAtRisk = "atRisk"
	// Still ordering often, just not lately. Worth a nudge, not a rescue.
	RFMLoyal = "loyal"
	// Spends a lot, rarely. Banquets and office orders — a customer whose
	// value is invisible to a frequency rule.
	RFMBigSpender = "bigSpender"
	// Ordered recently, not yet a habit. The group where one good message
	// changes the most.
	RFMPromising = "promising"
	// Long gone.
	RFMLost = "lost"
	// The middle: neither recent nor frequent enough to be anything else.
	RFMNeedsAttention = "needsAttention"
)

// RFMCells is every cell, in the order the panel shows them: best first, so the
// screen reads top to bottom as "who we have and who we are losing".
var RFMCells = []string{
	RFMChampions, RFMLoyal, RFMBigSpender, RFMPromising,
	RFMAtRisk, RFMNeedsAttention, RFMLost,
}

// rfmMinBase is how many customers with orders it takes before ranking means
// anything.
//
// ⚠️ The same reasoning as `vipFloor`, and the same number on purpose: calling
// the fourth customer of a brand-new restaurant a "champion" is a label that
// describes the restaurant's age, not the customer. Below this the whole
// feature reports itself as unavailable rather than producing five cells over
// nine people.
const rfmMinBase = 10

// rfmScale is where the cut points fell for this particular customer base.
//
// ⚠️ **Relative, not absolute** — the `vipFloor` decision applied to all three
// axes. "Spent over a million" means something different in a samsa shop and a
// banquet hall, and it quietly rots as prices rise: a restaurant that put its
// prices up 30% would find itself with a room full of new big spenders and no
// new money.
type rfmScale struct {
	// Cut points, ascending, at the 20th/40th/60th/80th percentile of the base.
	// Days since the last order for recency, order count for frequency, so'm
	// spent for monetary.
	RecencyDays [4]int `json:"recencyDays"`
	Frequency   [4]int `json:"frequency"`
	Monetary    [4]int `json:"monetary"`
	// How many customers the cuts were computed from. Shown next to them: a
	// percentile over eleven people is arithmetically fine and worth saying out
	// loud, exactly like the "sold on 2 days" column in ABC/XYZ.
	Base int `json:"base"`
}

// rfmScores is one customer's position, 1–5 on each axis, 5 being best.
type rfmScores struct {
	R int `json:"r"`
	F int `json:"f"`
	M int `json:"m"`
}

// newRFMScale computes the cut points from the whole customer base.
//
// Returns nil when there are too few customers to rank, and every caller treats
// nil as "no RFM" rather than as "everybody scores 1".
func newRFMScale(facts map[string]customerFacts, now time.Time) *rfmScale {
	var days, freq, money []int
	for _, f := range facts {
		if f.OrdersCount == 0 || f.LastOrder == nil {
			// Someone with an account and no orders has no position on any of
			// these axes. Scoring them 1/1/1 would put them in "lost", which
			// says they left — they never arrived.
			continue
		}
		days = append(days, daysSince(*f.LastOrder, now))
		freq = append(freq, f.OrdersCount)
		money = append(money, f.OrdersTotal)
	}
	if len(days) < rfmMinBase {
		return nil
	}
	return &rfmScale{
		RecencyDays: quintileCuts(days),
		Frequency:   quintileCuts(freq),
		Monetary:    quintileCuts(money),
		Base:        len(days),
	}
}

// daysSince is whole days between two moments, never negative.
func daysSince(then, now time.Time) int {
	d := int(now.Sub(then).Hours() / 24)
	if d < 0 {
		// A clock skew or a back-dated order should read as "today", not as a
		// customer who ordered in the future and ranks above everyone.
		return 0
	}
	return d
}

// quintileCuts is the four values that split a set into fifths.
func quintileCuts(values []int) [4]int {
	sorted := append([]int(nil), values...)
	sort.Ints(sorted)
	var cuts [4]int
	for i := range cuts {
		// The value at the 20/40/60/80th percentile, by position.
		idx := int(math.Round(float64(len(sorted)) * float64(i+1) / 5))
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		cuts[i] = sorted[idx]
	}
	return cuts
}

// scoreAscending places a value on a 1–5 scale where larger is better.
//
// ⚠️ **Scored by value, not by position in the sorted list.** Half a
// restaurant's customers have ordered exactly once, so splitting by position
// hands identical customers different frequency scores — and the owner who
// sorts the list by F sees two people with one order each in different
// segments, which is indefensible and destroys trust in the whole screen.
// Comparing against the cut *values* means equal inputs always score equally,
// even when that leaves a band empty.
func scoreAscending(v int, cuts [4]int) int {
	score := 1
	for _, cut := range cuts {
		if v > cut {
			score++
		}
	}
	if score > 5 {
		score = 5
	}
	return score
}

// scoreFor is one customer's R, F and M.
func scoreFor(f customerFacts, scale *rfmScale, now time.Time) (rfmScores, bool) {
	if scale == nil || f.OrdersCount == 0 || f.LastOrder == nil {
		return rfmScores{}, false
	}
	// ⚠️ Recency is inverted: the axis is measured in days since the last order,
	// where **small is good**. Scoring it ascending like the other two would put
	// the customers who left longest ago at the top of the screen, and the
	// mistake would look like data — the numbers would all be plausible.
	return rfmScores{
		R: 6 - scoreAscending(daysSince(*f.LastOrder, now), scale.RecencyDays),
		F: scoreAscending(f.OrdersCount, scale.Frequency),
		M: scoreAscending(f.OrdersTotal, scale.Monetary),
	}, true
}

// rfmCell names a customer's cell.
//
// The order of these checks **is** the definition — they are not independent
// conditions, and rearranging them silently reclassifies people. Two in
// particular:
//
//   - `atRisk` is tested before `loyal`, because a customer with a high
//     frequency score and a low recency one is precisely a regular who has
//     stopped coming. Testing `loyal` first would file them as healthy, and the
//     one thing this whole feature exists to surface would never appear.
//   - `bigSpender` is tested before `lost`, so a lapsed banquet customer is
//     named by what makes them worth a phone call rather than by their absence.
func rfmCell(s rfmScores) string {
	switch {
	case s.R >= 4 && s.F >= 4:
		return RFMChampions
	case s.R <= 2 && s.F >= 3:
		return RFMAtRisk
	case s.F >= 4:
		return RFMLoyal
	case s.M >= 4 && s.F <= 2:
		return RFMBigSpender
	case s.R >= 4:
		return RFMPromising
	case s.R == 1:
		return RFMLost
	}
	return RFMNeedsAttention
}

// cellFor is the cell id for one customer, or "" when they cannot be ranked.
func cellFor(f customerFacts, scale *rfmScale, now time.Time) string {
	s, ok := scoreFor(f, scale, now)
	if !ok {
		return ""
	}
	return rfmCell(s)
}

// rfmPrefix namespaces an RFM cell when it is used as a campaign audience.
//
// ⚠️ **Required, not decoration.** The rule segments already contain `lost`,
// and it means something different there — "no order for 180 days", a fixed
// line — from what it means here, which is "in the bottom fifth of this
// restaurant's customers by recency". Sharing the id would let a campaign
// aimed at one silently go to the other, and the two lists overlap enough that
// nobody would notice from the count.
const rfmPrefix = "rfm:"

// rfmSegmentID is a cell's id in the segment namespace.
func rfmSegmentID(cell string) string { return rfmPrefix + cell }

// RFMSegmentIDs is every cell as a campaign audience, in screen order.
func RFMSegmentIDs() []string {
	out := make([]string, 0, len(RFMCells))
	for _, cell := range RFMCells {
		out = append(out, rfmSegmentID(cell))
	}
	return out
}

// rfmScaleNow computes the cut points from the whole customer base, now.
//
// The counterpart of `vipFloorNow`, and it exists for the same reason: a
// single-customer screen still needs a fact about everybody, because a rank is
// meaningless without the field it is a rank within.
func (h *Handler) rfmScaleNow(r *http.Request) *rfmScale {
	facts, _ := h.customerFactsByUser(r.Context())
	return newRFMScale(facts, time.Now())
}

// ---- The screen ----

type rfmCellRow struct {
	Cell  string `json:"cell"`
	Count int    `json:"count"`
	// What the cell is worth, so the panel can say "23 people, 14% of the base,
	// 41% of the money". A cell holding a tenth of the customers and half the
	// takings is the whole reason to look at this grid.
	Revenue int `json:"revenue"`
	// Average scores, for the grid's own axis labels.
	AvgR float64 `json:"avgR"`
	AvgF float64 `json:"avgF"`
	AvgM float64 `json:"avgM"`
}

// AdminRFM is the customer base split into cells.
func (h *Handler) AdminRFM(w http.ResponseWriter, r *http.Request) {
	// Owner only, exactly like the segments screen: this is the customer base,
	// and the base belongs to the company rather than to one branch.
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	facts, _ := h.customerFactsByUser(r.Context())
	now := time.Now()
	scale := newRFMScale(facts, now)

	if scale == nil {
		// Reported as unavailable, with the reason and the number needed.
		// Drawing an empty grid would look like a base with no customers in it,
		// which is a different and much worse piece of news.
		httpx.JSON(w, http.StatusOK, map[string]any{
			"available": false,
			"minBase":   rfmMinBase,
			"base":      len(facts),
			// ⚠️ An empty array, never nil: Go marshals a nil slice as `null`
			// and the panel maps over this.
			"cells": []rfmCellRow{},
		})
		return
	}

	type acc struct {
		count, revenue  int
		sumR, sumF, sumM int
	}
	byCell := map[string]*acc{}
	total, totalRevenue := 0, 0

	for _, f := range facts {
		s, ok := scoreFor(f, scale, now)
		if !ok {
			continue
		}
		cell := rfmCell(s)
		a := byCell[cell]
		if a == nil {
			a = &acc{}
			byCell[cell] = a
		}
		a.count++
		a.revenue += f.OrdersTotal
		a.sumR += s.R
		a.sumF += s.F
		a.sumM += s.M
		total++
		totalRevenue += f.OrdersTotal
	}

	// Every cell, in the fixed order, including the empty ones. An empty
	// "atRisk" is good news and has to be visible as such — a cell that
	// disappears when it empties makes a healthy base and a broken query look
	// identical.
	rows := make([]rfmCellRow, 0, len(RFMCells))
	for _, cell := range RFMCells {
		a := byCell[cell]
		if a == nil {
			rows = append(rows, rfmCellRow{Cell: cell})
			continue
		}
		rows = append(rows, rfmCellRow{
			Cell: cell, Count: a.count, Revenue: a.revenue,
			AvgR: round1(float64(a.sumR) / float64(a.count)),
			AvgF: round1(float64(a.sumF) / float64(a.count)),
			AvgM: round1(float64(a.sumM) / float64(a.count)),
		})
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"available": true,
		"scale":     scale,
		"total":     total,
		"revenue":   totalRevenue,
		"cells":     rows,
	})
}

