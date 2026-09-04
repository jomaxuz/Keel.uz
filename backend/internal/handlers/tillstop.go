package handlers

// ---- The stop list, on the counter's own screen ----
//
// The manual stop list already existed and had exactly one door: `/admin/stop-list`,
// behind a panel login. That is the wrong door for the only event that ever
// opens it. Lag'mon runs out at eight in the evening, and the person who finds
// out is the cashier being told by the kitchen — who then either walks to the
// office, or phones whoever holds the panel password, or does neither and keeps
// selling a dish nobody can cook. The third outcome is the usual one, and it
// ends with a guest being told twenty minutes later that their order is not
// coming.
//
// So the same list gets a second door on the till, and deliberately the *same*
// list: `branch.soldOut`, the one a human writes. The till's own automatic list
// (`posSoldOut`) and the stockroom's (`stockSoldOut`) stay separate, for the
// third time in this codebase and for the same reason — three writers into one
// field cancel each other out, and each cancellation looks like a button that
// does not work.
//
// ⚠️ **No new permission, and that is the rule rather than an exception to it.**
// staffrole.go states when a permission exists: the action takes money out, or
// destroys a record. Stopping a dish does neither. It is reversible with one
// tap, it is visible to the whole room the moment it happens, and the worst
// outcome of a mistake is a dish briefly off the menu — while the cost of
// gating it is the evening described above. It rides on `PermWaiter`, which
// `PermCashier` implies, so the cashier the feature was asked for has it and so
// does the waiter standing closer to the kitchen door.

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// soldOutHeldBy names the other list holding a dish off sale, or "" when the
// manual switch is free to act.
//
// ⚠️ **One function, called by the panel and the till.** Both screens show the
// same three lists and both offer the same button, so a rule written twice is a
// rule that will one day let the till reopen something the panel refuses — and
// the restaurant would be right to conclude that one of the two screens is
// lying. The refusal names where the real switch is, because "no" without a
// destination is what teaches a room to stop reading our messages.
func soldOutHeldBy(b models.Branch, itemID primitive.ObjectID) string {
	if b.IsPOSSoldOut(itemID) {
		return "bu taom kassa tizimida stop listda — uni kassadan qaytaring"
	}
	if b.IsStockSoldOut(itemID) {
		return "bu taom ombor hisobi bo'yicha stop listda — kirimni yozing yoki qoldiqni sanang"
	}
	return ""
}

// setBranchSoldOut moves one dish on or off the branch's manual stop list.
//
// ⚠️ **A branch that has never had anything run out holds `null` here, not
// `[]`** — Go marshals a nil slice that way — and `$addToSet`/`$pull` refuse a
// non-array field. Without the first update the very first tap at a counter
// fails with a write error, and that is precisely the tap that has to work.
//
// `$addToSet` / `$pull` rather than rewriting the list: the update touches one
// element, so two people stopping two different dishes in the same second
// cannot lose one another's.
func (h *Handler) setBranchSoldOut(
	r *http.Request, branchID, itemID primitive.ObjectID, soldOut bool, until *time.Time,
) error {
	ctx := r.Context()
	if _, err := h.Store.Branches.UpdateOne(ctx,
		bson.M{"_id": branchID, "soldOut": bson.M{"$not": bson.M{"$type": "array"}}},
		bson.M{"$set": bson.M{"soldOut": []primitive.ObjectID{}}},
	); err != nil {
		return err
	}
	// ⚠️ **The old deadline is dropped first, every time, whichever way this
	// goes.** Stopping a dish again with no deadline has to mean "until I say
	// so", and a leftover timer from two hours ago would put it back on the
	// menu on the next read — with the screen showing it as stopped, because
	// the cashier had just stopped it. `$pull` and `$push` cannot touch the
	// same array in one update, so this is its own write.
	if _, err := h.Store.Branches.UpdateByID(ctx, branchID, bson.M{
		"$pull": bson.M{"soldOutUntil": bson.M{"menuItemId": itemID}},
	}); err != nil {
		return err
	}
	op := "$addToSet"
	if !soldOut {
		op = "$pull"
	}
	update := bson.M{
		op:     bson.M{"soldOut": itemID},
		"$set": bson.M{"updatedAt": time.Now()},
	}
	if soldOut && until != nil {
		update["$push"] = bson.M{"soldOutUntil": models.SoldOutTimer{
			MenuItemID: itemID, Until: *until,
		}}
	}
	_, err := h.Store.Branches.UpdateByID(ctx, branchID, update)
	return err
}

// stopUntil turns what the screen asked for into a moment, on the server's
// clock.
//
// ⚠️ **A duration in, an instant out, and the conversion happens here.** The
// till sends "two hours", never "at 21:40": a monoblock whose CMOS battery has
// died reports 2010 after a power cut — the reason offline check times are
// clamped — and a deadline computed on that machine would either lift the
// moment it was written or never lift at all. Neither failure says anything on
// screen; the dish is simply wrong about being available.
//
// ⚠️ **"Until closing" is worked out from this branch's own hours**, because
// the alternative — local midnight — puts a dish back on the menu at midnight
// in a place that serves until two, which is the middle of service.
func stopUntil(b models.Branch, minutes int, untilClose bool, now time.Time) *time.Time {
	if untilClose {
		at := closingAfter(b.WorkingHours, now)
		return &at
	}
	if minutes <= 0 {
		return nil
	}
	// A day is the ceiling: past that "until I say so" is the honest setting,
	// and a deadline nobody will be present for is one that surprises the next
	// shift.
	if minutes > 24*60 {
		minutes = 24 * 60
	}
	at := now.Add(time.Duration(minutes) * time.Minute)
	return &at
}

// closingAfter is when this branch next shuts.
//
// ⚠️ **Falls back to the end of the local day**, and that is the safe direction:
// a branch with no hours filled in is most of them, and a fallback that lifted
// the stop immediately would make the button look broken on exactly those
// installs. Same reading as an empty `mapProvider`.
func closingAfter(hours []models.WorkingHour, now time.Time) time.Time {
	local := now.In(time.Local)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local).
		AddDate(0, 0, 1)
	for _, h := range hours {
		if h.Day != int(local.Weekday()) || h.IsClosed || h.Close == "" {
			continue
		}
		close, err := time.ParseInLocation("15:04", h.Close, time.Local)
		if err != nil {
			break
		}
		at := time.Date(local.Year(), local.Month(), local.Day(),
			close.Hour(), close.Minute(), 0, 0, time.Local)
		// ⚠️ A room that shuts at two in the morning shuts *tomorrow*. Read as
		// today, the deadline is already in the past and the stop lifts on the
		// next read — during the evening it was meant to cover.
		if !at.After(local) {
			at = at.AddDate(0, 0, 1)
		}
		return at
	}
	return midnight
}

// StaffStopList is the whole screen in one response: every dish on this
// branch's menu, and why each one is off sale.
//
// ⚠️ **The whole menu, not only what is stopped.** The screen answers two
// questions with one list — "what is off right now" and "put this off" — and
// the second one needs the dishes that are still on. What is off is sorted to
// the top instead, which is the same order the panel's screen uses.
//
// One request rather than three (menu, categories, branch) because a screen
// assembled from three calls has three chances to render a half-true page in
// front of a guest.
func (h *Handler) StaffStopList(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	ctx := r.Context()

	// ⚠️ The branch comes from the employee, never from the request — the same
	// rule as every other till endpoint. A cashier who typed another branch's
	// id would otherwise empty a kitchen they have never stood in.
	var branch models.Branch
	if err := h.Store.Branches.FindOne(ctx, bson.M{"_id": s.BranchID}).Decode(&branch); err != nil {
		httpx.Error(w, http.StatusNotFound, "filial topilmadi")
		return
	}

	filter := bson.M{}
	// The menu belongs to the brand, so a company running two brands from one
	// building must not be shown the other one's dishes.
	if !branch.BrandID.IsZero() {
		filter["brandId"] = branch.BrandID
	}
	cur, err := h.Store.Menu.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := []models.MenuItem{}
	_ = cur.All(ctx, &items)

	catNames := map[primitive.ObjectID]string{}
	if ccur, err := h.Store.Categories.Find(ctx, filter); err == nil {
		cats := []models.Category{}
		_ = ccur.All(ctx, &cats)
		for _, c := range cats {
			catNames[c.ID] = c.Name
		}
	}

	// ⚠️ Counted once for the screen rather than per row: this is the only
	// place the numbers are read for display, and a query per dish would be a
	// hundred round trips on a menu somebody opens twenty times an evening.
	// An error is not fatal — the screen is worth more with the limits and no
	// counts than not at all.
	sold, _ := h.soldToday(ctx, s.BranchID)

	rows := make([]stopListRow, 0, len(items))
	for _, it := range items {
		// ⚠️ A dish that is not on sale at all is left out entirely. On the
		// panel it is worth seeing — the owner is the person who hid it — but
		// on a counter screen it is a dish that cannot be ordered either way,
		// and every row here is a row somebody has to read past at eight in the
		// evening.
		if !it.IsAvailable {
			continue
		}
		rows = append(rows, stopListRow{
			MenuItemID: it.ID.Hex(),
			Name:       it.Name,
			CategoryID: it.CategoryID.Hex(),
			Category:   catNames[it.CategoryID],
			ImageURL:   it.ImageURL,
			Price:      it.Price,
			Manual:     branch.IsManualSoldOut(it.ID),
			Until:      branch.SoldOutUntilFor(it.ID),
			POS:        branch.IsPOSSoldOut(it.ID),
			Stock:      branch.IsStockSoldOut(it.ID),
			Limit:      branch.LimitFor(it.ID),
			Sold:       sold[it.ID],
			LimitOff:   branch.IsLimitSoldOut(it.ID),
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		offA := a.Manual || a.POS || a.Stock || a.LimitOff
		offB := b.Manual || b.POS || b.Stock || b.LimitOff
		if offA != offB {
			return offA
		}
		if a.Category != b.Category {
			return a.Category < b.Category
		}
		return a.Name < b.Name
	})

	httpx.JSON(w, http.StatusOK, map[string]any{
		"items": rows,
		// The branch by name, because a cashier covering a second site should
		// see whose kitchen they are about to close a dish in.
		"branch": branch.Name,
	})
}

// StaffSetSoldOut marks one dish run out, or puts it back, at the employee's
// own branch.
func (h *Handler) StaffSetSoldOut(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	var req soldOutRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	itemID, err := objectID(strings.TrimSpace(req.MenuItemID))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "taom noto'g'ri")
		return
	}
	ctx := r.Context()

	var branch models.Branch
	if err := h.Store.Branches.FindOne(ctx, bson.M{"_id": s.BranchID}).Decode(&branch); err != nil {
		httpx.Error(w, http.StatusNotFound, "filial topilmadi")
		return
	}
	// ⚠️ Only on the way *back*. Putting a dish on the manual list while the
	// till already holds it is harmless and honest — it is off sale either way,
	// and the kitchen said so too — while refusing would be a "no" to somebody
	// who is right.
	if !req.SoldOut {
		if reason := soldOutHeldBy(branch, itemID); reason != "" {
			httpx.Error(w, http.StatusConflict, reason)
			return
		}
	}

	// The dish has to be on this brand's menu. Not a formality: without it the
	// list accumulates ids of dishes that no longer exist, and the panel's stop
	// list quietly grows rows it cannot name.
	var item models.MenuItem
	menuFilter := bson.M{"_id": itemID}
	if !branch.BrandID.IsZero() {
		menuFilter["brandId"] = branch.BrandID
	}
	if err := h.Store.Menu.FindOne(ctx, menuFilter).Decode(&item); err != nil {
		httpx.Error(w, http.StatusNotFound, "taom topilmadi")
		return
	}

	until := stopUntil(branch, req.Minutes, req.UntilClose, time.Now())
	if err := h.setBranchSoldOut(r, s.BranchID, itemID, req.SoldOut, until); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// ⚠️ Written down with the name of whoever tapped it. "Why was lag'mon off
	// on Friday" is asked the following week, and the answer that matters is
	// which shift decided it — the same reason a void carries a name.
	details := "qaytadan bor"
	if req.SoldOut {
		details = "tugadi"
		// ⚠️ The deadline belongs in the journal too. "Why was lag'mon off on
		// Friday" is asked the following week, and "until 21:00" is a different
		// answer from "all evening" — the first one somebody chose, the second
		// one somebody forgot.
		if until != nil {
			details += " (" + until.In(time.Local).Format("15:04") + " gacha)"
		}
	}
	h.logAction(r, ActBranchUpdate, "branch", s.BranchID.Hex(), branch.Name,
		item.Name+" — "+details+" ("+s.Name+", kassa)")

	resp := map[string]any{
		"ok":         true,
		"menuItemId": itemID.Hex(),
		"soldOut":    req.SoldOut,
	}
	// ⚠️ Sent back so the card can show the countdown without a second round
	// trip — and so the screen shows the moment **the server** computed, not
	// one the browser worked out from a clock that may disagree.
	if until != nil {
		resp["until"] = until
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// StaffSetDailyLimit sets how many of a dish this branch sells today.
//
// ⚠️ **The same permission as the stop list itself**, not a management one.
// "We cooked ten portions" is said by the person who cooked them, and the
// screen exists precisely so that fact does not have to travel to an office and
// back. It takes no money out and destroys no record — the test staffrole.go
// sets — and the worst mistake is a dish briefly off the menu, one tap from
// being back on.
//
// ⚠️ **Zero clears the limit rather than stopping the dish.** A field somebody
// empties means "never mind", and reading it as "sell none" would take a dish
// off the menu through a control that says nothing of the kind. Stopping a dish
// is the switch beside it, which says what it does.
func (h *Handler) StaffSetDailyLimit(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	var req struct {
		MenuItemID string `json:"menuItemId"`
		Limit      int    `json:"limit"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	itemID, err := primitive.ObjectIDFromHex(req.MenuItemID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "taom topilmadi")
		return
	}
	if req.Limit < 0 {
		req.Limit = 0
	}
	ctx := r.Context()

	// ⚠️ Pulled and pushed rather than edited in place: `$pull` then `$addToSet`
	// on an array of objects cannot express "replace the one with this id", and
	// two documents for one dish would have LimitFor answer whichever came
	// first — a limit that changes when nothing changed it.
	if _, err := h.Store.Branches.UpdateByID(ctx, s.BranchID, bson.M{
		"$pull": bson.M{"dailyLimits": bson.M{"menuItemId": itemID}},
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.Limit > 0 {
		if _, err := h.Store.Branches.UpdateByID(ctx, s.BranchID, bson.M{
			"$push": bson.M{"dailyLimits": models.DailyLimit{
				MenuItemID: itemID, Limit: req.Limit,
			}},
			"$set": bson.M{"updatedAt": time.Now()},
		}); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// ⚠️ **Recomputed immediately, and this is the half that makes the screen
	// honest.** Raising a limit past what has already been sold has to put the
	// dish back on the menu now — not at the next sale, which for a dish that
	// is stopped is a sale that cannot happen. Without this the control would
	// only ever work in one direction.
	h.applyDailyLimits(ctx, s.BranchID)

	var branch models.Branch
	if err := h.Store.Branches.FindOne(ctx, bson.M{"_id": s.BranchID}).
		Decode(&branch); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	sold, _ := h.soldToday(ctx, s.BranchID)
	h.logAction(r, ActBranchUpdate, "branch", s.BranchID.Hex(), branch.Name,
		"kunlik chegara")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"menuItemId": req.MenuItemID,
		"limit":      req.Limit,
		"sold":       sold[itemID],
		"limitOff":   branch.IsLimitSoldOut(itemID),
	})
}
