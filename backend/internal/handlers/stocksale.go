package handlers

// ---- Taking the food off the shelf when it is rung up ----
//
// ⚠️ **A reconciler, not ten hooks.** An order is mutated from a dozen places —
// lines added, a quantity corrected, a line voided, a check split, merged,
// cancelled, a website order placed, a week of offline checks arriving at once —
// and a `writeOff()` call bolted onto each of them is a rule maintained in
// twelve copies. The twelfth is the one somebody forgets, and what it produces
// is not an error: it is a shelf that is quietly wrong, found at a stocktake, by
// the person holding the clipboard.
//
// So there is one function. It reads an order, works out what its lines *should*
// have taken, compares that with what has been written, and writes the
// difference. Every call site is the same single line, and it is **idempotent** —
// calling it twice changes nothing, calling it late fixes everything, and the
// backfill is the same function run over history.
//
// ⚠️ **It must not fail the thing that triggered it.** A cashier who cannot take
// money because a stock row would not write is a worse outcome than a row
// written a minute late — and because this is a reconciler, a minute late is all
// it ever is: the next tap on that check repairs it. Same rule the alert bell
// and the print queue follow.

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/models"
)

// syncOrderStock brings the shelf into line with one order, whatever just
// happened to it.
//
// ⚠️ Errors are logged, never returned to the caller's response. See the note
// at the top of the file.
func (h *Handler) syncOrderStock(ctx context.Context, o *models.Order) {
	if err := h.reconcileOrderStock(ctx, o, nil); err != nil {
		log.Printf("stock movement (order %s): %v", o.Number, err)
	}
}

// reconcileOrderStock is the same work with its error, for the backfill and the
// tests — the two callers that want to know.
func (h *Handler) reconcileOrderStock(
	ctx context.Context, o *models.Order, cache *recipeCache,
) error {
	if o == nil || o.ID.IsZero() {
		return nil
	}
	if cache == nil {
		cache = h.newRecipeCache(ctx, o.BrandID)
	}

	// What has been written for this order already.
	have := map[string]models.StockMovement{}
	cur, err := h.Store.StockMoves.Find(ctx, bson.M{"orderId": o.ID})
	if err != nil {
		return err
	}
	var rows []models.StockMovement
	if err := cur.All(ctx, &rows); err != nil {
		return err
	}
	for _, m := range rows {
		have[m.LineKey] = m
	}

	now := time.Now()
	want := wantedMovements(o)
	seen := make(map[string]bool, len(want))
	var writes []mongo.WriteModel

	for _, w := range want {
		seen[w.key] = true
		old, exists := have[w.key]
		if !exists {
			lines, valuePer, err := cache.portion(ctx, w.item)
			if err != nil {
				return err
			}
			// ⚠️ A line whose card is empty writes no row at all rather than a
			// row of nothing. Most of a menu is legitimately uncosted, and
			// thousands of empty rows would make the audit unreadable while
			// changing no figure. What is missing is already measured, by name,
			// on the coverage screen.
			if len(lines) == 0 {
				continue
			}
			doc := models.StockMovement{
				BranchID:    o.BranchID,
				At:          movementTime(o, w.item),
				OrderID:     o.ID,
				OrderNumber: o.Number,
				LineID:      w.item.LineID,
				LineKey:     w.key,
				MenuItemID:  w.item.MenuItemID,
				DishName:    w.item.Name,
				Qty:         w.qty,
				Lines:       lines,
				ValuePer:    valuePer,
				Wasted:      w.wasted,
				Reason:      w.reason,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			writes = append(writes, mongo.NewInsertOneModel().SetDocument(doc))
			continue
		}
		// ⚠️ A reversed row never comes back to life. A voided line that
		// reappears is a new line with a new id — the till mints one on every
		// split and every re-add — so resurrecting this one would attach
		// tonight's food to a row somebody has already explained.
		if !old.Counts() {
			continue
		}
		if old.Qty == w.qty && old.Wasted == w.wasted {
			continue
		}
		writes = append(writes, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": old.ID}).
			SetUpdate(bson.M{"$set": bson.M{
				"qty": w.qty, "wasted": w.wasted,
				"reason": w.reason, "updatedAt": now,
			}}))
	}

	// Anything written that the order no longer asks for: a line removed before
	// the kitchen saw it, a line voided and not wasted, a line moved to another
	// check, a cancelled order that never got cooked.
	for key, m := range have {
		if seen[key] || !m.Counts() {
			continue
		}
		writes = append(writes, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": m.ID}).
			SetUpdate(bson.M{"$set": bson.M{
				"reversedAt":     now,
				"reversedReason": reversalReason(o),
				"updatedAt":      now,
			}}))
	}

	if len(writes) == 0 {
		return nil
	}
	_, err = h.Store.StockMoves.BulkWrite(ctx, writes,
		options.BulkWrite().SetOrdered(false))
	return err
}

// wantLine is one line of an order as the shelf should see it.
type wantLine struct {
	key    string
	item   models.OrderItem
	qty    float64
	wasted bool
	reason string
}

// wantedMovements is what this order's lines should have taken off the shelf.
//
// ⚠️ **The whole of the "rung up, not cooked" question lives here**, and the
// till already answered it. A line the kitchen was never told about takes
// nothing: `StaffVoidCheckLine` drops it from the check outright, with no reason
// asked, because nothing happened. A line voided *after* it was fired was made —
// the till asks whether it was thrown away (`CheckLineVoid.Wasted`), and where
// it was, the ingredients are gone and the row stays. Putting them back would
// file real waste as a correction, which is the one outcome a waste report
// exists to prevent.
func wantedMovements(o *models.Order) []wantLine {
	out := make([]wantLine, 0, len(o.Items))
	for i, it := range o.Items {
		if it.MenuItemID.IsZero() {
			continue
		}
		qty := float64(it.Qty) * it.PortionFactor()
		if qty <= 0 {
			continue
		}
		w := wantLine{key: lineKeyOf(it, i), item: it, qty: qty}
		switch {
		case !o.MergedIntoID.IsZero():
			// ⚠️ **A merged check is cancelled and keeps its lines**, on purpose:
			// they are the record of where this table's food went (§ tillmerge).
			// Read as an ordinary cancellation the food would be counted twice —
			// once on the check that now carries it and once here as waste — and
			// the shelf would come up short by a whole merged table every time
			// two bills were joined.
			continue
		case o.Status == models.StatusCancelled:
			// ⚠️ A cancelled check is voiding all of it, line by line, and the
			// same test applies: food the kitchen was told to make was made.
			// `cookedValue` draws this line for the loss alert already, and one
			// product must not have two answers to "was anything cooked".
			if !cookedLine(o, it) {
				continue
			}
			w.wasted = true
			w.reason = o.CancelReason
		case it.Void != nil:
			if !it.Void.Wasted {
				continue
			}
			w.wasted = true
			w.reason = it.Void.Reason
		}
		out = append(out, w)
	}
	return out
}

// cookedLine reports whether the kitchen had been told about this line.
//
// ⚠️ **Per line on a till check and per order everywhere else**, because that is
// how the two work. A table's courses are fired one at a time — the starters
// forty minutes before the mains — so the check's own `queuedAt` says nothing
// about a particular line. A website order has no firing step at all: the
// kitchen sees the whole of it the moment it is accepted.
func cookedLine(o *models.Order, it models.OrderItem) bool {
	if o.Check != nil {
		return it.FiredAt != nil
	}
	return o.QueuedAt != nil
}

// lineKeyOf names one line stably across edits.
//
// ⚠️ **The till's own id where there is one, and the position where there is
// not.** A check is edited all evening and two guests ordering the same dish are
// deliberately two lines, which is exactly why `LineID` exists; a website order
// is written once and never edited, so its position cannot shift under us. Using
// position on a check would attach a movement to whatever line slid into that
// slot when an earlier one was removed.
func lineKeyOf(it models.OrderItem, i int) string {
	if it.LineID != "" {
		return it.LineID
	}
	return "#" + strconv.Itoa(i)
}

// movementTime is when the food was sold.
//
// ⚠️ **The order's own clock, never the writer's.** A till that was offline
// sends a week of checks the moment it reconnects; stamped with the write, all
// of them would land on one afternoon and the stocktakes either side of the gap
// would be wrong in opposite directions. Same reason a purchase keeps its
// invoice date.
func movementTime(o *models.Order, it models.OrderItem) time.Time {
	if it.FiredAt != nil {
		return *it.FiredAt
	}
	if !o.CreatedAt.IsZero() {
		return o.CreatedAt
	}
	return time.Now()
}

func reversalReason(o *models.Order) string {
	if o.Status == models.StatusCancelled {
		return "chek bekor qilindi"
	}
	return "qator olib tashlandi"
}

// ---- Working out what one portion takes ----

// recipeCache answers "what does one portion of this line take off the shelf"
// without asking twice.
//
// ⚠️ **It calls the existing expansion rather than repeating it.** Combos open
// into their frozen members, a pour takes the measure that was ticked, a half
// portion takes half — three rules with three scars behind them, all of them in
// `consumedBy`. A second implementation here would be the fourth copy of the
// combo rule in this codebase, and the previous three each cost a month of
// wrong figures before anybody noticed.
//
// The trick that makes reuse affordable is the synthetic order: one line, one
// portion, no void. Whatever the aggregate expansion would have counted for a
// thousand of them, it counts for one — which is the per-portion card.
type recipeCache struct {
	h           *Handler
	ingredients []models.Ingredient
	rates       map[primitive.ObjectID]float64
	names       map[primitive.ObjectID]string
	units       map[primitive.ObjectID]string
	byShape     map[string]cachedPortion
}

type cachedPortion struct {
	lines    []models.StockMovementLine
	valuePer int
}

func (h *Handler) newRecipeCache(ctx context.Context, brand primitive.ObjectID) *recipeCache {
	ing := h.scopedIngredients(ctx, brand)
	c := &recipeCache{
		h:           h,
		ingredients: ing,
		rates:       h.ingredientRates(ctx),
		names:       make(map[primitive.ObjectID]string, len(ing)),
		units:       make(map[primitive.ObjectID]string, len(ing)),
		byShape:     map[string]cachedPortion{},
	}
	for _, in := range ing {
		c.names[in.ID] = in.Name
		c.units[in.ID] = in.Unit
	}
	return c
}

// portion is one portion of this line, in purchase units, with what it is worth.
func (c *recipeCache) portion(
	ctx context.Context, it models.OrderItem,
) ([]models.StockMovementLine, int, error) {
	shape := lineShape(it)
	if got, ok := c.byShape[shape]; ok {
		return got.lines, got.valuePer, nil
	}
	// ⚠️ One portion, whole, and never voided: this asks what the dish is made
	// of, not what happened to it.
	one := it
	one.Void = nil
	one.Qty = 1
	one.Portion = 100
	used := c.h.consumedBy(ctx,
		[]models.Order{{Items: []models.OrderItem{one}}}, c.ingredients)

	lines := make([]models.StockMovementLine, 0, len(used))
	value := 0.0
	for id, qty := range used {
		if qty <= 0 {
			continue
		}
		lines = append(lines, models.StockMovementLine{
			IngredientID: id, Name: c.names[id], Qty: round3(qty),
		})
		// The rate is per recipe unit and the quantity is in purchase units, so
		// one has to be converted to meet the other — the same conversion the
		// balance screen makes.
		value += qty * float64(models.PerUnit(c.units[id])) * c.rates[id]
	}
	sortMovementLines(lines)
	got := cachedPortion{lines: lines, valuePer: int(value + 0.5)}
	c.byShape[shape] = got
	return got.lines, got.valuePer, nil
}

// lineShape is what makes two lines take the same thing off the shelf.
//
// ⚠️ The dish alone is not enough: a 50 ml pour and a 100 ml pour of the same
// bottle are one dish and two shapes, and a family set's members are frozen on
// the line rather than read from a menu that may have been rebuilt since.
func lineShape(it models.OrderItem) string {
	k := optionKeyOf(it)
	var b strings.Builder
	b.WriteString(k.dish.Hex())
	b.WriteString("|")
	b.WriteString(k.choices)
	for _, c := range it.ComboItems {
		b.WriteString("|")
		b.WriteString(c.MenuItemID.Hex())
		b.WriteString(":")
		b.WriteString(strconv.Itoa(c.Qty))
	}
	return b.String()
}

func sortMovementLines(lines []models.StockMovementLine) {
	for i := 1; i < len(lines); i++ {
		for j := i; j > 0 && lines[j].Name < lines[j-1].Name; j-- {
			lines[j], lines[j-1] = lines[j-1], lines[j]
		}
	}
}

// ---- Catching whatever the hooks missed ----
//
// ⚠️ **The hooks are the fast path; this is the one that has to be right.** Ten
// call sites is ten chances to forget the eleventh, and what a forgotten one
// produces is not an error — it is a shelf that is quietly wrong, discovered
// weeks later by the person holding the clipboard, who is then asked to explain
// it. A sweep over recently touched orders makes every one of those a delay of
// minutes instead of a permanent divergence, and it costs one indexed query.
//
// ⚠️ **It also covers what no hook can.** A write that failed on a flaky
// connection, a handler that returned early, a payment callback that changed a
// status from outside the till — all of them leave an order whose `updatedAt`
// moved and whose rows did not.
const (
	stockMoveSweepEvery = 2 * time.Minute
	// How far back a sweep looks. Comfortably longer than the interval, so a
	// tick that runs late or a restart mid-service does not leave a hole.
	stockMoveSweepWindow = 30 * time.Minute
)

func (h *Handler) StartStockMoveSync(ctx context.Context) {
	go func() {
		// A short delay, like the stop list's: the first request of the morning
		// should not queue behind this.
		timer := time.NewTimer(45 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		ticker := time.NewTicker(stockMoveSweepEvery)
		defer ticker.Stop()
		for {
			h.sweepStockMovements(ctx, time.Now().Add(-stockMoveSweepWindow))
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// sweepStockMovements reconciles every order touched since `since`.
func (h *Handler) sweepStockMovements(ctx context.Context, since time.Time) {
	cur, err := h.Store.Orders.Find(ctx,
		bson.M{"updatedAt": bson.M{"$gte": since}})
	if err != nil {
		log.Printf("stock movement sweep: %v", err)
		return
	}
	defer cur.Close(ctx)
	// ⚠️ One cache for the whole sweep. Without it this is an ingredient list
	// and a menu read per order, every two minutes, forever.
	caches := map[primitive.ObjectID]*recipeCache{}
	for cur.Next(ctx) {
		var o models.Order
		if err := cur.Decode(&o); err != nil {
			continue
		}
		cache, ok := caches[o.BrandID]
		if !ok {
			cache = h.newRecipeCache(ctx, o.BrandID)
			caches[o.BrandID] = cache
		}
		if err := h.reconcileOrderStock(ctx, &o, cache); err != nil {
			log.Printf("stock movement sweep (order %s): %v", o.Number, err)
		}
	}
}

// ---- The one-off pass over everything that came before ----

// stockBackfillKey names this migration in the state collection.
const stockBackfillKey = "stock_movements_v1"

// BackfillStockMovements writes the rows for every order placed before sales
// started writing them.
//
// ⚠️ **The server must not serve without it, and this is not caution.** The
// balance reads these rows and nothing else. An install that starts with the
// collection empty and a year of orders behind it reads consumption as zero:
// every shelf apparently full, every stop list released, every stocktake
// reporting a surplus the size of a year's cooking — and none of it looks like
// an error, which is what makes it expensive. The duplicate `pos_settings`
// index refuses to boot for the same reason.
//
// ⚠️ **Old orders are expanded with today's cards**, which is exactly what the
// derived arithmetic did on every read until now. So the backfill reproduces the
// figures the restaurant already had, and switching source changes nothing on
// any screen. From here on the card is frozen at the sale, and it is *new* sales
// that stop being rewritten by a later recipe edit.
func (h *Handler) BackfillStockMovements(ctx context.Context) error {
	var state struct {
		ID string `bson:"_id"`
	}
	err := h.Store.MigrationState.FindOne(ctx, bson.M{"_id": stockBackfillKey}).Decode(&state)
	if err == nil {
		return nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}

	started := time.Now()
	cur, err := h.Store.Orders.Find(ctx, bson.M{},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	caches := map[primitive.ObjectID]*recipeCache{}
	done := 0
	for cur.Next(ctx) {
		var o models.Order
		if err := cur.Decode(&o); err != nil {
			return err
		}
		cache, ok := caches[o.BrandID]
		if !ok {
			cache = h.newRecipeCache(ctx, o.BrandID)
			caches[o.BrandID] = cache
		}
		if err := h.reconcileOrderStock(ctx, &o, cache); err != nil {
			return err
		}
		done++
		// A restaurant with years of history takes a while, and a boot that
		// looks hung is a boot somebody kills halfway.
		if done%500 == 0 {
			log.Printf("stock movement backfill: %d orders", done)
		}
	}
	if err := cur.Err(); err != nil {
		return err
	}

	// ⚠️ Marked only after the pass finished. A marker written first would turn
	// a killed boot into a permanently half-filled ledger, which is the one
	// state nothing downstream could detect.
	_, err = h.Store.MigrationState.InsertOne(ctx, bson.M{
		"_id": stockBackfillKey, "at": time.Now(),
		"orders": done, "seconds": int(time.Since(started).Seconds()),
	})
	if err != nil {
		return err
	}
	log.Printf("stock movement backfill: %d orders in %s", done, time.Since(started))
	return nil
}
