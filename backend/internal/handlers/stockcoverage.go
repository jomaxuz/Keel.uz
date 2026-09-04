package handlers

// ---- How much of what was sold can the store account for ----
//
// ⚠️ **"Which dishes have no tech card" is the wrong question, and asking it is
// how this screen becomes noise.** Every restaurant has uncarded dishes and
// always will — two hundred dishes are not written up by anybody, and the ten
// that matter are written up first. A warning that is true on the day it ships
// and true a year later is a warning that gets muted, and the mute is not
// selective (see models/alert.go, which pays for this lesson at length).
//
// The question that shrinks, and can therefore be finished, is:
//
//	**what share of the money that came in left the shelf untraceably?**
//
// A restaurant that has carded forty of two hundred dishes has 20% of a menu
// and may have 85% of its revenue — in which case its store works and nobody
// should be nagged. The same restaurant with the *other* forty carded has the
// same menu figure and a store that means nothing. One number tells those two
// apart and the other cannot.
//
// ⚠️ **This is not the cost coverage the ABC report prints, and the two must
// never be merged.** That one asks whether a *margin* can be computed, and a
// hand-typed `cost` answers it (costledger.go falls back to exactly that). A
// typed cost writes nothing off a shelf. So a restaurant can sit at 100% cost
// coverage and 0% stock coverage, and folding the two into one figure would
// tell it the store is fine on the strength of numbers typed into a different
// field for a different purpose.
//
// ⚠️ **Sold, not on the menu.** A dish nobody ordered has correctly consumed
// nothing and carding it changes no figure — the same rule `factDeadDishes`
// draws. Uncarded dishes with no sales are counted separately and never lead.

import (
	"context"
	"net/http"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// How far back the question is asked when nobody says. A month: long enough
// that a weekly special is in it, short enough that carding work done last week
// shows up as progress.
const coverageDays = 30

// coverageRow is one dish whose sales the store cannot account for.
type coverageRow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Portions sold, and what they were rung up for.
	Qty     int `json:"qty"`
	Revenue int `json:"revenue"`
	// Why it does not count: "none" — no card at all; "partial" — a card
	// whose lines no longer all resolve.
	//
	// ⚠️ **Two reasons, not one, because the fixes are different and so is the
	// blame.** "No card" is work nobody has done yet. "Partial" is a card
	// somebody *did* write, which has since been broken by a deleted
	// ingredient or a prep item that lost its output — and until now that
	// silently made the dish cheaper (`recipeCost` skips the missing line),
	// which on a margin screen reads as good news.
	Issue string `json:"issue"`
	// For a set: the member that is missing its card. The set itself has no
	// recipe by construction, so naming it alone sends the reader to a screen
	// where there is nothing to fix.
	Via string `json:"via,omitempty"`
}

// prepGap is a prep item that cannot be priced, and therefore silently drops
// every dish built on it.
//
// ⚠️ **Listed apart from the dishes, because one of these is the cause of many
// of those.** A sushi rice with no output makes every roll "partial"; a reader
// given only the dish list carding twelve rolls would fix none of it.
type prepGap struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// "no_output" — a card with no yield, so nothing can be divided by it;
	// "incomplete" — its own inputs do not all resolve.
	Issue string `json:"issue"`
}

// coverageResult is the answer, whole, for whoever asked.
//
// ⚠️ **One computation, two readers.** The panel screen draws it and the
// morning briefing raises it as a fact; a second implementation for the second
// reader is how the panel and the phone start disagreeing about the same
// restaurant on the same morning.
type coverageResult struct {
	SoldTotal   int
	CostedTotal int
	Share       int
	Rows        []coverageRow
	Preps       []prepGap
}

// AdminStockCoverage answers what share of sales the tech cards account for.
func (h *Handler) AdminStockCoverage(w http.ResponseWriter, r *http.Request) {
	scope, sc, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	from, to, err := parseRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// ⚠️ A default window rather than "everything": this figure is opened to
	// decide what to card *next*, and a restaurant's first month is not the
	// question. Both ends, so the answer names a period it can be checked
	// against.
	if to == nil {
		t := time.Now()
		to = &t
	}
	if from == nil {
		f := to.AddDate(0, 0, -coverageDays)
		from = &f
	}

	got, err := h.stockCoverage(r.Context(), scope, sc.BrandID, *from, *to)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	warnFrom, warnOn := h.cardWarnSetting(r.Context(), scope)
	warnOff := !warnOn

	httpx.JSON(w, http.StatusOK, map[string]any{
		"from": *from,
		"to":   *to,
		// ⚠️ Sent as a period the reader can check the share against. A bare
		// percentage is a number an owner has to trust; one with its dates is
		// one they can argue with.
		"soldTotal":   got.SoldTotal,
		"costedTotal": got.CostedTotal,
		"share":       got.Share,
		"rows":        got.Rows,
		"preps":       got.Preps,
		// ⚠️ Sent with the figure so the switch that silences it can sit beside
		// it. A reminder whose off-switch is three screens away in "Sozlamalar"
		// is one people silence by ignoring the screen it appears on.
		"warnOff":  warnOff,
		"warnFrom": warnFrom,
	})
}

// stockCoverage is the measurement itself.
func (h *Handler) stockCoverage(
	ctx context.Context, scope bson.M, brand primitive.ObjectID, from, to time.Time,
) (coverageResult, error) {
	orders, err := h.ordersBetween(ctx, scope, &from, &to)
	if err != nil {
		return coverageResult{}, err
	}

	rates := h.ingredientRates(ctx)
	menu := h.coverageMenu(ctx, orders)

	// carded says whether one dish takes something off a shelf when it sells,
	// and names the member responsible when the dish is a set.
	//
	// ⚠️ **A set is carded only if every member is.** Its own recipe is empty
	// by construction — the tomatoes are in its members — so the plain test
	// passes it and a hundred family sets read as fully accounted for while
	// consuming nothing. The stock arithmetic already learned this
	// (`soldDishes`); this is the same fact asked as a question.
	var carded func(primitive.ObjectID, int) (string, string)
	carded = func(id primitive.ObjectID, depth int) (issue, via string) {
		d, ok := menu[id]
		// A dish that has since been deleted cannot be carded and cannot be
		// blamed. Counted as uncovered — it did leave the shelf — under its
		// name from the order line, which the caller supplies.
		if !ok {
			return "none", ""
		}
		if len(d.ComboItems) > 0 && depth < 1 {
			for _, m := range d.ComboItems {
				if is, _ := carded(m.MenuItemID, depth+1); is != "" {
					name := ""
					if member, ok := menu[m.MenuItemID]; ok {
						name = member.Name
					}
					return is, name
				}
			}
			return "", ""
		}
		if len(d.Recipe) == 0 {
			return "none", ""
		}
		if !recipeComplete(d.Recipe, rates) {
			return "partial", ""
		}
		return "", ""
	}

	type agg struct {
		name    string
		qty     int
		revenue int
		issue   string
		via     string
	}
	gaps := map[primitive.ObjectID]*agg{}
	soldTotal, costedTotal := 0, 0
	for _, o := range orders {
		// The same basis every other sales figure uses, so this share and the
		// revenue it is a share *of* cannot disagree.
		if o.Status == models.StatusCancelled {
			continue
		}
		for _, it := range o.Items {
			if !it.Live() || it.MenuItemID.IsZero() {
				continue
			}
			// ⚠️ The frozen line price, never a recomputed one: this is a share
			// of what was actually rung up.
			revenue := it.Price * it.Qty
			soldTotal += revenue
			issue, via := carded(it.MenuItemID, 0)
			if issue == "" {
				costedTotal += revenue
				continue
			}
			row := gaps[it.MenuItemID]
			if row == nil {
				name := it.Name
				if d, ok := menu[it.MenuItemID]; ok && d.Name != "" {
					name = d.Name
				}
				row = &agg{name: name, issue: issue, via: via}
				gaps[it.MenuItemID] = row
			}
			row.qty += it.Qty
			row.revenue += revenue
		}
	}

	rows := make([]coverageRow, 0, len(gaps))
	for id, g := range gaps {
		rows = append(rows, coverageRow{
			ID: id.Hex(), Name: g.name, Qty: g.qty,
			Revenue: g.revenue, Issue: g.issue, Via: g.via,
		})
	}
	// ⚠️ **By money, and this ordering is the feature.** Alphabetical, the list
	// is two hundred names and an infinite job; by revenue, the first ten lines
	// are most of the answer and the work has an end somebody can see.
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Revenue != rows[j].Revenue {
			return rows[i].Revenue > rows[j].Revenue
		}
		return rows[i].Name < rows[j].Name
	})

	// ⚠️ **A restaurant that sold nothing is covered, not uncovered.** Zero
	// over zero is the shape that makes a quiet Monday look like a broken store,
	// and the fact built on this would raise an alarm about a day off.
	share := 100.0
	if soldTotal > 0 {
		share = float64(costedTotal) / float64(soldTotal) * 100
	}

	return coverageResult{
		SoldTotal:   soldTotal,
		CostedTotal: costedTotal,
		Share:       int(share + 0.5),
		Rows:        rows,
		Preps:       h.unpricedPreps(ctx, brand, rates),
	}, nil
}

// coverageMenu loads every dish the period sold, plus the members of every set
// among them.
//
// ⚠️ Two queries at most, never one per dish: a month of orders on a menu of two
// hundred is two hundred round trips on a screen somebody opens while thinking
// about something else — the same reason `costLedgerFor` loads members in a
// second pass.
type coverageDish struct {
	ID         primitive.ObjectID  `bson:"_id"`
	Name       string              `bson:"name"`
	Recipe     []models.RecipeLine `bson:"recipe"`
	ComboItems []models.ComboLine  `bson:"comboItems"`
}

func (h *Handler) coverageMenu(
	ctx context.Context, orders []models.Order,
) map[primitive.ObjectID]coverageDish {
	out := map[primitive.ObjectID]coverageDish{}
	ids := make([]primitive.ObjectID, 0, 64)
	seen := map[primitive.ObjectID]bool{}
	add := func(id primitive.ObjectID) {
		if id.IsZero() || seen[id] {
			return
		}
		seen[id] = true
		ids = append(ids, id)
	}
	for _, o := range orders {
		for _, it := range o.Items {
			add(it.MenuItemID)
		}
	}
	load := func(want []primitive.ObjectID) []coverageDish {
		if len(want) == 0 {
			return nil
		}
		var rows []coverageDish
		if cur, err := h.Store.Menu.Find(ctx,
			bson.M{"_id": bson.M{"$in": want}}); err == nil {
			_ = cur.All(ctx, &rows)
		}
		return rows
	}
	first := load(ids)
	// ⚠️ A set's members were not in the list asked about — nothing sold them
	// by name — and without them every combo reads as carded.
	more := make([]primitive.ObjectID, 0, 8)
	for _, d := range first {
		out[d.ID] = d
		for _, m := range d.ComboItems {
			if !seen[m.MenuItemID] {
				seen[m.MenuItemID] = true
				more = append(more, m.MenuItemID)
			}
		}
	}
	for _, d := range load(more) {
		out[d.ID] = d
	}
	return out
}

// unpricedPreps is the prep items nothing can be costed through.
func (h *Handler) unpricedPreps(
	ctx context.Context, brand primitive.ObjectID, rates map[primitive.ObjectID]float64,
) []prepGap {
	out := []prepGap{}
	for _, in := range h.scopedIngredients(ctx, brand) {
		if len(in.Recipe) == 0 {
			continue
		}
		if _, ok := rates[in.ID]; ok {
			continue
		}
		// ⚠️ The yield first, because it is the one an owner reads as done: a
		// card with rows in it looks finished, and `MadeInHouse` says it is not
		// a prep item at all until somebody writes down what comes out of the
		// pot.
		issue := "incomplete"
		if in.Output <= 0 {
			issue = "no_output"
		}
		out = append(out, prepGap{ID: in.ID.Hex(), Name: in.Name, Issue: issue})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ---- The restaurant's own answer to being told ----

// AdminSetCardWarn stores whether the morning briefing should raise the gap,
// and from what share.
//
// ⚠️ **Its own call, not a field on a settings form.** `AdminUpdateBranch`
// replaces what it is given, and the fields it deliberately never writes
// (`soldOut`, `kioskSecret`) are the scars from the last two times a form
// zeroed something it had never heard of. A switch that lives beside the thing
// it silences is also the only kind anybody finds.
func (h *Handler) AdminSetCardWarn(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	branchID := h.scopeBranch(r, scope)
	if branchID.IsZero() {
		httpx.Error(w, http.StatusBadRequest, "filial tanlanmagan")
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		// ⚠️ Pointers: a body that omits a field must leave it alone. The
		// screen sends the switch and the threshold from two different
		// controls, and the one being changed must not reset the other.
		Off  *bool `json:"off"`
		From *int  `json:"from"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := bson.M{"updatedAt": time.Now()}
	if req.Off != nil {
		set["stockCardWarnOff"] = *req.Off
	}
	if req.From != nil {
		// ⚠️ Clamped rather than refused. "Tell me below 150%" is a typo, and
		// a form that answers a typo with a red box on a screen somebody opened
		// to silence a reminder is a form they close instead.
		v := *req.From
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		set["stockCardWarnFrom"] = v
	}
	if _, err := h.Store.Branches.UpdateByID(r.Context(), branchID,
		bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var saved models.Branch
	_ = h.Store.Branches.FindOne(r.Context(), bson.M{"_id": branchID}).Decode(&saved)
	h.logAction(r, ActSettingsUpdate, "branch", branchID.Hex(), "",
		"texkarta qamrovi haqida eslatish: "+onOff(!saved.StockCardWarnOff))
	from := saved.StockCardWarnFrom
	if from <= 0 {
		from = models.DefaultStockCardWarnFrom
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"off": saved.StockCardWarnOff, "from": from,
	})
}
