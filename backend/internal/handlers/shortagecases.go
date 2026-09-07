package handlers

// ---- What is missing, what it is worth, and who is going to answer for it ----
//
// ⚠️ **The arithmetic was already finished and nobody could act on it.** A
// count freezes an expected figure, a counted one, a difference and what that
// difference was worth — per line, since stocktake.go shipped. All of it was
// reachable only by opening one count and reading down forty lines of an
// insert-only document, where the twelve kilos of meat that matter sit between
// two lines of parsley that were out by a gram. The restaurant had the finding
// and no queue: nothing said *which* shortfall to look at first, and nothing
// recorded that anybody ever had.
//
// This is that queue. It invents no number — every figure on it is the one the
// count froze — and it adds the three things that turn a row into a case:
//
//  1. **Money, sorted.** A kilo of parsley and a kilo of beef are the same
//     line in units and two different mornings in som. By value, the first
//     three rows are almost always the whole answer.
//  2. **A period, named.** A shortfall belongs to the stretch between two
//     counts of that store, not to the day it was found. Saying so is what
//     stops it being read as "last night".
//  3. **An answer, recorded once.** See models/shortagecase.go.
//
// ⚠️ **Derived on every read, never stored.** The case list is the counts,
// re-read; the only document written is the verdict. A stored queue would go
// stale exactly the way a stored health flag did (CLAUDE.md, `provisionStatus`)
// — showing a shortfall that a later count has already superseded, on the one
// screen whose whole claim is that it is what is outstanding *now*.
//
// ⚠️ **Across branches, unlike every other stock screen.** The store screens
// demand a single branch on purpose (`stockBranch`): "the company holds 9 kg of
// beef" spread over three fridges is a number nobody can count, order against
// or cook with. Nothing is summed here — each row names its own branch and its
// own store, and the ordering is by money — so the one question an owner of
// three branches actually asks ("where is the worst of it") is answerable
// without opening three screens and comparing by hand.

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// How far back the queue looks when nobody says. Ninety days: long enough that
// a restaurant counting monthly has three counts in it, short enough that the
// list is this quarter's work rather than the company's history.
const shortageDays = 90

// The most rows sent at once. ⚠️ A cap rather than paging, because the list is
// sorted by money: row two hundred is worth a few thousand som and nobody
// scrolls to it. The count of what was left out is sent instead, so the screen
// can say so rather than pretending the list is complete.
const shortageMax = 200

// Below this share of sales the shortfall figures do not mean what they say.
//
// ⚠️ **The trap the whole feature stands on.** A shortage is "what the books
// expected, less what was found", and the books only expect what a tech card
// told them to expect. In a restaurant where half the revenue leaves the shelf
// with no card, the expected figure is too high for every ingredient those
// dishes use — so the queue fills with confident, precise, entirely invented
// shortfalls, and the first one an owner investigates costs them their trust in
// the screen. Below the line the queue still draws, and says what it is: a
// coverage problem wearing a shortage's clothes.
const shortageTrustFrom = 70

// How close in money a surplus has to be before it is offered as this
// shortfall's twin. A quarter: a mis-scan is rarely exact — the two packets
// differ a little, and part of the pair may be genuine.
const twinTolerance = 0.25

// shortageRow is one shortfall: one ingredient, on one count, in one store.
type shortageRow struct {
	// ⚠️ The pair *is* the id. There is no case document until somebody
	// answers one, so nothing else could be.
	StocktakeID  string `json:"stocktakeId"`
	IngredientID string `json:"ingredientId"`

	Name string `json:"name"`
	Unit string `json:"unit"`

	BranchID    string `json:"branchId,omitempty"`
	Branch      string `json:"branch,omitempty"`
	WarehouseID string `json:"warehouseId,omitempty"`
	Warehouse   string `json:"warehouse,omitempty"`

	// When it was found, and what stretch of time it accumulated over.
	//
	// ⚠️ **`Since` is the previous count of *this store*, and null where there
	// was none.** A shortfall measured from nothing is "everything that ever
	// arrived, less everything accounted for" — a real number, and not one to
	// hand somebody as a month's loss. The screen says which it is holding.
	At    time.Time  `json:"at"`
	Since *time.Time `json:"since,omitempty"`

	// The frozen figures, exactly as the count saved them.
	Expected float64 `json:"expected"`
	Counted  float64 `json:"counted"`
	Diff     float64 `json:"diff"`
	// What is missing, in som, as a positive number. ⚠️ The count stores it
	// negative because there it is a signed component of a total; here it is
	// the size of a problem, and a queue sorted by "most negative" reads
	// backwards to everybody who is not its author.
	Value int `json:"value"`

	// Who counted, and what the count as a whole was explained as. ⚠️ Not an
	// accusation and not this line's answer: a count's note covers forty lines,
	// and it is shown because a manager who wrote "freezer broke down" should
	// not be asked the same question forty more times.
	By        string  `json:"by,omitempty"`
	CountNote string  `json:"countNote,omitempty"`
	CountLoss int     `json:"countLoss"`
	Share     float64 `json:"share"`

	// ---- What makes one shortfall worse than another the same size ----
	//
	// ⚠️ **Money alone ranks a quarter's slow shrinkage above a week's theft.**
	// These are the four figures that separate them, and every one of them is
	// arithmetic over facts already on the screen — none is a new claim.

	// How long the period was, and what the shortfall works out to a day.
	//
	// ⚠️ **The figure that reorders the queue in an owner's head.** A million
	// som over ninety days is shrinkage; the same million over four days is
	// something that is still happening. Sorting stays on the total — that is
	// the money — but a row that cannot say which of the two it is cannot be
	// acted on either.
	Days   int `json:"days,omitempty"`
	PerDay int `json:"perDay,omitempty"`

	// What share of what should have been there is gone. ⚠️ Twelve kilos out of
	// four hundred is trade; twelve out of fourteen is an event. The same
	// twelve kilos and the same money.
	Pct float64 `json:"pct,omitempty"`

	// How much of this ingredient the tech cards accounted for over the same
	// period, in purchase units.
	//
	// ⚠️ **Zero is the most important thing this screen can say, and it is not
	// a shortfall.** If nothing consumed the ingredient in the books, then
	// "expected" is simply everything that ever arrived and the whole counted
	// difference is a gap in the cards — an unwritten recipe, a dish sold
	// uncarded — rather than something that left. Reported per row because the
	// coverage percentage at the top is a fact about the *restaurant*: an
	// ingredient used only by carded dishes is trustworthy at 30% coverage, and
	// one used by uncarded dishes is not at 90%.
	Used float64 `json:"used"`

	// How many of this store's counts in the window found this ingredient
	// short. ⚠️ Two is a pattern and one is an evening — and the pattern is
	// what a person can actually do something about.
	Repeat int `json:"repeat,omitempty"`

	// A surplus on the same count worth about as much as this shortfall.
	//
	// ⚠️ **The mis-scan, named before somebody calls it a loss.** Two similar
	// packets and one barcode leave this row short and its twin over by nearly
	// the same money on the same day — and a queue that shows only shortfalls
	// hides exactly the half that explains this one. It is a question, never a
	// verdict: the manager still picks the answer.
	Twin string `json:"twin,omitempty"`

	// The answer, once there is one.
	Verdict     models.ShortageVerdict `json:"verdict,omitempty"`
	VerdictNote string                 `json:"verdictNote,omitempty"`
	VerdictBy   string                 `json:"verdictBy,omitempty"`
	VerdictAt   *time.Time             `json:"verdictAt,omitempty"`
}

// AdminShortageCases is the queue: what is missing, worst first.
func (h *Handler) AdminShortageCases(w http.ResponseWriter, r *http.Request) {
	scope, sc, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	days := shortageDays
	if v, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil &&
		v > 0 && v <= 365 {
		days = v
	}
	to := time.Now()
	from := to.AddDate(0, 0, -days)

	takes, err := h.stocktakesSince(r.Context(), scope, from)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	since := h.previousCounts(r.Context(), scope, takes)
	answered := h.verdictsFor(r.Context(), takes)

	ing := map[primitive.ObjectID]models.Ingredient{}
	for _, x := range h.scopedIngredients(r.Context(), sc.BrandID) {
		ing[x.ID] = x
	}
	stores := h.warehouseNames(r.Context(), scope)
	branches := h.branchNames(r)

	// ⚠️ **Read from the oldest period start, not from the window's start.**
	// The first count in a ninety-day window is measured from whenever that
	// store was counted before — which may be months earlier — and a
	// consumption figure that began later would report part of a period as
	// unaccounted for, on the one screen whose whole job is to say what is
	// unaccounted for.
	oldest := from
	for _, t := range since {
		if t != nil && t.Before(oldest) {
			oldest = *t
		}
	}
	used := h.consumedByDay(r.Context(), scope, oldest)

	got := shortageRows(takes, shortageNames{
		ingredients: ing, stores: stores, branches: branches, usedByDay: used,
	}, since, answered)

	// ⚠️ **The caveat is computed over the same window the rows are.** Asked
	// over a default month while the queue held a quarter, the share would
	// describe a period the reader is not looking at — and this figure exists
	// precisely to tell them whether to believe the one they are.
	share := 100
	if cov, err := h.stockCoverage(r.Context(), scope, sc.BrandID, from, to); err == nil {
		share = cov.Share
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"rows":  got.Rows,
		"more":  got.More,
		"from":  local(from),
		"to":    local(to),
		"days":  days,
		"open":  got.Open,
		"total": got.Total,
		// What is still unanswered, in money. ⚠️ The headline figure, and
		// deliberately not the total: the total only ever grows, and a number
		// that only grows is one nobody reads twice.
		"openValue": got.OpenValue,
		"coverage":  share,
		"weak":      share < shortageTrustFrom,
	})
}

// shortageNames is what the rows are labelled from — the catalogue, the rooms
// and the branches, read once each.
type shortageNames struct {
	ingredients map[primitive.ObjectID]models.Ingredient
	stores      map[primitive.ObjectID]string
	branches    map[primitive.ObjectID]string
	// What the cards accounted for, per ingredient per day, over the whole
	// window — so each row's own period can be summed without another query.
	usedByDay map[primitive.ObjectID]map[string]float64
}

// shortageQueue is the answer: the rows, and the totals a screen leads with.
type shortageQueue struct {
	Rows []shortageRow
	More int
	// How many are unanswered, and what they are worth.
	Open      int
	OpenValue int
	Total     int
}

// shortageRows turns counts into the queue.
//
// ⚠️ **Its own function, and the response goes through it.** The lesson
// `downloadsJSON` cost this codebase: a rule that lives inside a handler is a
// rule the next edit walks past without noticing, and the compiler says nothing.
// The ordering, the sign and the "only shortfalls" filter are the product here;
// each of them has a test beside it, and the only path to the browser is this
// function.
func shortageRows(
	takes []models.Stocktake, names shortageNames,
	since map[primitive.ObjectID]*time.Time,
	answered map[caseKey]models.ShortageCase,
) shortageQueue {
	out := shortageQueue{Rows: []shortageRow{}}
	for _, t := range takes {
		for _, l := range t.Lines {
			// ⚠️ Only shortfalls. A surplus is a real finding and a different
			// one — usually a delivery booked twice — and a queue that mixes
			// the two cannot be sorted by "worst", because the worst of one is
			// the best of the other.
			if l.Value >= 0 {
				continue
			}
			in, ok := names.ingredients[l.IngredientID]
			if !ok {
				// An ingredient deleted since the count. The shortfall stands
				// — it was frozen — but there is nothing left to name it with,
				// and a row called "" is a row nobody can act on.
				continue
			}
			row := shortageRow{
				StocktakeID:  t.ID.Hex(),
				IngredientID: l.IngredientID.Hex(),
				Name:         in.Name,
				Unit:         in.Unit,
				BranchID:     hexOrEmpty(t.BranchID),
				Branch:       names.branches[t.BranchID],
				WarehouseID:  hexOrEmpty(t.WarehouseID),
				Warehouse:    names.stores[t.WarehouseID],
				At:           local(t.At),
				Since:        since[t.ID],
				Expected:     l.Expected,
				Counted:      l.Counted,
				Diff:         l.Diff,
				Value:        -l.Value,
				By:           t.By,
				CountNote:    t.Note,
				CountLoss:    -t.Value,
			}
			// ⚠️ **What share of the whole count's shortfall this one line
			// is.** A count out by two million across forty lines is a
			// different event from one out by two million on beef alone: the
			// first is a process — a card set, a supplier, a month of small
			// slippage — and the second is one thing that happened. The queue
			// cannot tell them apart by row value, because the row value is
			// the same on both.
			if t.Value < 0 {
				row.Share = float64(-l.Value) / float64(-t.Value)
			}
			// ---- How bad, how fast, and against what ----
			if row.Since != nil {
				days := int(math.Round(row.At.Sub(*row.Since).Hours() / 24))
				if days < 1 {
					// ⚠️ Two counts on one day is one day, not zero: a per-day
					// figure divided by nothing is an infinity on a screen.
					days = 1
				}
				row.Days = days
				row.PerDay = row.Value / days
			}
			// ⚠️ Against what *should* have been there, not against what was
			// found: a shelf counted at zero would otherwise be short by an
			// infinite percentage, and one counted above expectation is not on
			// this list at all.
			if l.Expected > 0 {
				row.Pct = math.Abs(l.Diff) / l.Expected
			}
			row.Used = round3(usedIn(names.usedByDay[l.IngredientID],
				row.Since, row.At))
			row.Twin = twinOf(t, l, names.ingredients)
			if v, ok := answered[caseKey{t.ID, l.IngredientID}]; ok {
				row.Verdict = v.Verdict
				row.VerdictNote = v.Note
				row.VerdictBy = v.By
				at := local(v.At)
				row.VerdictAt = &at
			} else {
				out.Open++
				out.OpenValue += -l.Value
			}
			out.Total += -l.Value
			out.Rows = append(out.Rows, row)
		}
	}

	// ---- How many times this has happened here ----
	//
	// ⚠️ **Counted across the rows themselves rather than queried.** Every count
	// in the window is already in hand, so "the third time this store has been
	// short of beef" costs a pass over a slice — and it is the difference
	// between an evening and a habit, which is the difference between what a
	// manager says and what they do.
	seen := map[storeKey]map[string]int{}
	for i := range out.Rows {
		key := storeKey{}
		if id, err := primitive.ObjectIDFromHex(out.Rows[i].BranchID); err == nil {
			key.branch = id
		}
		if id, err := primitive.ObjectIDFromHex(out.Rows[i].WarehouseID); err == nil {
			key.store = id
		}
		if seen[key] == nil {
			seen[key] = map[string]int{}
		}
		seen[key][out.Rows[i].IngredientID]++
	}
	for i := range out.Rows {
		key := storeKey{}
		if id, err := primitive.ObjectIDFromHex(out.Rows[i].BranchID); err == nil {
			key.branch = id
		}
		if id, err := primitive.ObjectIDFromHex(out.Rows[i].WarehouseID); err == nil {
			key.store = id
		}
		if n := seen[key][out.Rows[i].IngredientID]; n > 1 {
			out.Rows[i].Repeat = n
		}
	}

	// ⚠️ **Open first, then by money.** Not by date: a shortfall does not get
	// less true for being three weeks old, and a queue ordered by recency puts
	// last night's forgotten kilo of onions above the beef nobody has explained
	// since March. An answered row stays on the list — a month of verdicts is
	// the sentence about the process — but never above an unanswered one.
	sort.SliceStable(out.Rows, func(i, j int) bool {
		a, b := out.Rows[i], out.Rows[j]
		if (a.Verdict == "") != (b.Verdict == "") {
			return a.Verdict == ""
		}
		if a.Value != b.Value {
			return a.Value > b.Value
		}
		return a.At.After(b.At)
	})
	if len(out.Rows) > shortageMax {
		out.More = len(out.Rows) - shortageMax
		out.Rows = out.Rows[:shortageMax]
	}
	return out
}

// AdminCloseShortageCase records what one shortfall turned out to be.
func (h *Handler) AdminCloseShortageCase(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		StocktakeID  string                 `json:"stocktakeId"`
		IngredientID string                 `json:"ingredientId"`
		Verdict      models.ShortageVerdict `json:"verdict"`
		Note         string                 `json:"note"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	takeID, err := objectID(req.StocktakeID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	ingID, err := objectID(req.IngredientID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	if !models.ValidVerdict(req.Verdict) {
		httpx.Error(w, http.StatusBadRequest, "sababini tanlang")
		return
	}
	note := clampText(req.Note, 400)
	// ⚠️ **The sentence is required beside the kind, and this is not a form
	// nicety.** "Isrof" on its own is a category; "muzlatgich juma kuni
	// buzilgan, go'sht tashlandi" is the thing a person can check next month.
	// The five kinds exist so a month of these can be counted, not so a
	// shortfall can be dismissed in one click.
	if note == "" {
		httpx.Error(w, http.StatusBadRequest, "sababini yozing")
		return
	}

	// ⚠️ The branch comes off the count, and the count is looked up **inside
	// the scope filter** — an id alone must never select a document, or a
	// manager pinned to one branch closes another branch's findings by pasting
	// an id (handlers/scope.go says this at length).
	filter := bson.M{"_id": takeID}
	for k, v := range scope {
		filter[k] = v
	}
	var take models.Stocktake
	if err := h.Store.Stocktakes.FindOne(r.Context(), filter).Decode(&take); err != nil {
		httpx.Error(w, http.StatusNotFound, "sanoq topilmadi")
		return
	}
	// ⚠️ And the line has to be one that is actually short. Otherwise this is
	// an endpoint for writing verdicts about shortfalls that never happened —
	// rows that would then appear on nobody's queue and in everybody's monthly
	// tally of what the shortfalls turned out to be.
	short := false
	for _, l := range take.Lines {
		if l.IngredientID == ingID && l.Value < 0 {
			short = true
			break
		}
	}
	if !short {
		httpx.Error(w, http.StatusNotFound, "bu qatorda kamomad yo'q")
		return
	}

	doc := models.ShortageCase{
		BranchID:     take.BranchID,
		StocktakeID:  takeID,
		IngredientID: ingID,
		Verdict:      req.Verdict,
		Note:         note,
		By:           h.adminName(r),
		At:           time.Now(),
	}
	// ⚠️ **Insert, not upsert, and the unique index is what refuses the
	// second one.** Two managers reading the same queue both see an open case
	// and both answer it; an upsert would silently keep the later one, which
	// is the answer of whoever happened to click second. See
	// models/shortagecase.go — the same shape as the count's own explanation.
	if _, err := h.Store.ShortageCases.InsertOne(r.Context(), doc); err != nil {
		httpx.Error(w, http.StatusConflict, "bu kamomad allaqachon izohlangan")
		return
	}
	h.logAction(r, "shortage.close", "stocktake", takeID.Hex(),
		string(req.Verdict), note)
	httpx.JSON(w, http.StatusCreated, map[string]any{"ok": true})
}

// usedIn is what the cards took off this shelf between two counts.
//
// ⚠️ **Summed from a per-day map rather than queried per row.** A quarter's
// queue is a few hundred rows over a dozen counts; one aggregation bucketed by
// day answers all of them, and a query per row would be a screen nobody opens
// twice.
//
// ⚠️ **Both ends in the restaurant's own days.** The buckets are day strings
// built in Mongo with the restaurant's timezone (mongoTZ), so the bounds are
// formatted the same way — comparing them against UTC timestamps would move an
// evening's cooking into the wrong period, and here that is the difference
// between "the cards account for this" and "nothing does".
func usedIn(byDay map[string]float64, from *time.Time, to time.Time) float64 {
	if len(byDay) == 0 {
		return 0
	}
	end := local(to).Format("2006-01-02")
	start := ""
	if from != nil {
		start = local(*from).Format("2006-01-02")
	}
	total := 0.0
	for day, qty := range byDay {
		if day > end {
			continue
		}
		// ⚠️ The opening day itself belongs to the previous period — it was
		// counted, so whatever left the shelf before that moment is already in
		// the figure the count froze.
		if start != "" && day <= start {
			continue
		}
		total += qty
	}
	return total
}

// twinOf names a surplus on the same count worth about what this line is short.
//
// ⚠️ **A question, not a verdict.** Two similar packets and one barcode leave
// this row short and its twin over by nearly the same money on the same day;
// the queue shows only shortfalls, so without this the half that explains the
// row is invisible. What it produces is a name beside a number — the manager
// still picks the answer, and "swap" is one of six.
//
// ⚠️ **By money and not by quantity**, because that is what makes it evidence:
// twelve kilos of beef and twelve kilos of onions are the same quantity and
// nobody confuses them at a till. A tolerance of a quarter, because a mis-scan
// is rarely exact — the packets differ a little, and some of the pair may be
// genuine.
func twinOf(
	t models.Stocktake, short models.StocktakeLine,
	ing map[primitive.ObjectID]models.Ingredient,
) string {
	best, bestGap := "", 0.0
	want := float64(-short.Value)
	if want <= 0 {
		return ""
	}
	for _, l := range t.Lines {
		if l.Value <= 0 || l.IngredientID == short.IngredientID {
			continue
		}
		gap := math.Abs(float64(l.Value)-want) / want
		if gap > twinTolerance {
			continue
		}
		if best == "" || gap < bestGap {
			best, bestGap = ing[l.IngredientID].Name, gap
		}
	}
	return best
}

// consumedByDay is what the cards took off each shelf, per ingredient per day.
//
// ⚠️ **One aggregation for the whole queue.** The alternative is
// `consumedInPeriod` once per row, which on a quarter of counts is hundreds of
// passes over the movement collection for a screen that is opened every
// morning.
//
// ⚠️ **The day is the restaurant's**, built in Mongo with mongoTZ for the
// reason every other pipeline here is: the driver speaks UTC, and an evening's
// cooking filed under tomorrow lands in the wrong count's period — which is
// exactly the boundary this figure exists to describe.
func (h *Handler) consumedByDay(
	ctx context.Context, scope bson.M, from time.Time,
) map[primitive.ObjectID]map[string]float64 {
	out := map[primitive.ObjectID]map[string]float64{}
	match := bson.M{"reversedAt": nil, "at": bson.M{"$gte": from}}
	for k, v := range scope {
		match[k] = v
	}
	cur, err := h.Store.StockMoves.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$unwind", Value: "$lines"}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"ing": "$lines.ingredientId",
				"day": bson.M{"$dateToString": bson.M{
					"format": "%Y-%m-%d", "date": "$at", "timezone": mongoTZ(),
				}},
			},
			"qty": bson.M{"$sum": bson.M{"$multiply": []any{"$lines.qty", "$qty"}}},
		}}},
	})
	if err != nil {
		return out
	}
	defer cur.Close(ctx)
	var rows []struct {
		ID struct {
			Ing primitive.ObjectID `bson:"ing"`
			Day string             `bson:"day"`
		} `bson:"_id"`
		Qty float64 `bson:"qty"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return out
	}
	for _, row := range rows {
		if out[row.ID.Ing] == nil {
			out[row.ID.Ing] = map[string]float64{}
		}
		out[row.ID.Ing][row.ID.Day] = row.Qty
	}
	return out
}

// ---- The reads behind the queue ----

// storeKey names one room in one branch — the unit a count is taken of.
type storeKey struct {
	branch primitive.ObjectID
	store  primitive.ObjectID
}

// caseKey names one line of one count, which is a case's whole identity.
type caseKey struct {
	take       primitive.ObjectID
	ingredient primitive.ObjectID
}

// stocktakesSince reads the counts the queue is built from, oldest first.
func (h *Handler) stocktakesSince(
	ctx context.Context, scope bson.M, from time.Time,
) ([]models.Stocktake, error) {
	filter := bson.M{"at": bson.M{"$gte": from}}
	for k, v := range scope {
		filter[k] = v
	}
	cur, err := h.Store.Stocktakes.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var rows []models.Stocktake
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// previousCounts says what each count's shortfall accumulated from: the
// previous count of that same store, keyed by the count it belongs to.
//
// ⚠️ **Per count, not per store.** A store counted three times in the window
// has three different period starts, and giving every row the oldest of them
// would describe two of the three shortfalls as a quarter's loss.
//
// ⚠️ **One database query per store, not per count.** Only the oldest count of
// each store needs looking up; every later one is preceded by a count already
// in this list. On a chain with four stores counting weekly that is four reads
// instead of fifty.
func (h *Handler) previousCounts(
	ctx context.Context, scope bson.M, takes []models.Stocktake,
) map[primitive.ObjectID]*time.Time {
	out := map[primitive.ObjectID]*time.Time{}
	// The last count seen in each store, as the list is walked oldest first.
	last := map[storeKey]*time.Time{}
	for _, t := range takes {
		key := storeKey{t.BranchID, t.WarehouseID}
		if prev, ok := last[key]; ok {
			out[t.ID] = prev
		} else {
			out[t.ID] = h.countBefore(ctx, scope, key, t.At)
		}
		at := local(t.At)
		last[key] = &at
	}
	return out
}

// countBefore is when this store was last counted before a moment, if ever.
func (h *Handler) countBefore(
	ctx context.Context, scope bson.M, key storeKey, before time.Time,
) *time.Time {
	filter := bson.M{"at": bson.M{"$lt": before}}
	for k, v := range scope {
		filter[k] = v
	}
	if !key.branch.IsZero() {
		filter["branchId"] = key.branch
	}
	// ⚠️ The undivided store matches counts with no warehouse **and** counts
	// saved before the field existed — the same filter expectedStockByWarehouse
	// builds, and for the same reason: a query on the zero id does not match a
	// document that has no such field.
	if key.store.IsZero() {
		filter["$or"] = []bson.M{
			{"warehouseId": bson.M{"$exists": false}},
			{"warehouseId": primitive.NilObjectID},
		}
	} else {
		filter["warehouseId"] = key.store
	}
	var prev models.Stocktake
	if err := h.Store.Stocktakes.FindOne(ctx, filter,
		options.FindOne().SetSort(bson.D{{Key: "at", Value: -1}})).
		Decode(&prev); err != nil {
		return nil
	}
	at := local(prev.At)
	return &at
}

// verdictsFor reads the answers already given for these counts.
func (h *Handler) verdictsFor(
	ctx context.Context, takes []models.Stocktake,
) map[caseKey]models.ShortageCase {
	out := map[caseKey]models.ShortageCase{}
	if len(takes) == 0 {
		return out
	}
	ids := make([]primitive.ObjectID, 0, len(takes))
	for _, t := range takes {
		ids = append(ids, t.ID)
	}
	cur, err := h.Store.ShortageCases.Find(ctx,
		bson.M{"stocktakeId": bson.M{"$in": ids}})
	if err != nil {
		return out
	}
	var rows []models.ShortageCase
	if err := cur.All(ctx, &rows); err != nil {
		return out
	}
	for _, c := range rows {
		out[caseKey{c.StocktakeID, c.IngredientID}] = c
	}
	return out
}

// warehouseNames maps store ids to names, so a row can say which room.
func (h *Handler) warehouseNames(
	ctx context.Context, scope bson.M,
) map[primitive.ObjectID]string {
	out := map[primitive.ObjectID]string{}
	filter := bson.M{}
	for k, v := range scope {
		filter[k] = v
	}
	cur, err := h.Store.Warehouses.Find(ctx, filter)
	if err != nil {
		return out
	}
	var rows []models.Warehouse
	if err := cur.All(ctx, &rows); err != nil {
		return out
	}
	for _, x := range rows {
		out[x.ID] = x.Name
	}
	return out
}

// hexOrEmpty is an id, or nothing where there is no id.
//
// ⚠️ **Never the zero hex.** `omitempty` does not apply to an ObjectID (it is
// an array), so an unset branch reaches the browser as
// "000000000000000000000000", which JavaScript reads as **truthy** — the trap
// CLAUDE.md documents and this codebase has shipped twice.
func hexOrEmpty(id primitive.ObjectID) string {
	if id.IsZero() {
		return ""
	}
	return id.Hex()
}
