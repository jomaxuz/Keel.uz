package handlers

import (
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- Splitting a check ----
//
// ⚠️ **The guest numbers were being collected and could not be acted on.** A
// waiter could tag every line with the seat that ordered it — the field is on
// the line, the till draws it — and at the end of the meal the only way to hand
// a table two bills was to have opened two checks before anybody had ordered.
// That means guessing who will eat what, at the moment the party sits down, and
// it is the reason dining rooms end up doing the arithmetic on paper.
//
// ⚠️ **Splitting is a waiter's action, not a cashier's.** It moves no money and
// takes nothing off a bill: the same food, the same prices, on two pieces of
// paper. Requiring the cashier's code would mean the person holding the screen
// at the table has to fetch somebody for the most ordinary request in a dining
// room — and the way that ends is the cashier's PIN being told to everyone,
// which costs far more than this decision is worth.
//
// ⚠️ **A split is not a new sale**, and the sales screen must not read it as
// one: a table that asked for four bills had one dinner, and a report counting
// four would show a busier night than the room had. `check.splitFromId` is what
// says so, and it is written here, once.

type splitRequest struct {
	// Which lines move onto the new check. Empty is refused rather than
	// interpreted: "split off nothing" and "split off everything" are both
	// almost certainly a misread screen.
	LineIDs []string `json:"lineIds"`
}

// StaffSplitCheck moves part of a check onto a new one at the same table.
func (h *Handler) StaffSplitCheck(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	from, ok := h.loadCheck(w, r, s)
	if !ok || !requireOpen(w, from) {
		return
	}
	var req splitRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	wanted := map[string]bool{}
	for _, id := range req.LineIDs {
		wanted[id] = true
	}

	var moved, kept []models.OrderItem
	for _, it := range from.Items {
		// ⚠️ A voided line never moves — the same rule as moving lines between
		// checks. It is the record of food written off *this* check, and
		// carrying it across would carry the blame with it.
		if wanted[it.LineID] && it.Live() {
			// ⚠️ The seat number is dropped on the way over. It exists to
			// divide one bill; once the bill is divided it is a number nobody
			// can act on, and on a printed receipt it reads as a table the
			// guest was not sitting at.
			it.Guest = 0
			moved = append(moved, it)
			continue
		}
		kept = append(kept, it)
	}
	if len(moved) == 0 {
		httpx.Error(w, http.StatusNotFound, "qator topilmadi")
		return
	}
	// ⚠️ Splitting off everything is refused. It produces an empty check that
	// nobody can pay and a new one identical to the old, which is not what
	// anybody meant by pressing this — and the empty one then sits on the floor
	// screen looking like a table waiting to order.
	if !anyLive(kept) {
		httpx.Error(w, http.StatusBadRequest,
			"hammasini bo'lib bo'lmaydi — chekda kamida bitta taom qolishi kerak")
		return
	}

	branch, err := h.branchByID(r, s.BranchID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	now := time.Now()
	split := models.Order{
		BranchID:    from.BranchID,
		BrandID:     from.BrandID,
		Number:      branchOrderNumber(branch.Code),
		Status:      models.StatusPending,
		Type:        from.Type,
		Customer:    from.Customer,
		TableID:     from.TableID,
		TableNumber: from.TableNumber,
		Items:       moved,
		// ⚠️ The half inherits the rate the table sat down under. Reading the
		// branch again here would charge two halves of one dinner differently
		// if the setting changed during the meal.
		ServicePercent: from.ServicePercent,
		// ⚠️ Nobody is counted twice. The guests are sitting at the table the
		// original check holds; copying the number onto both halves would
		// double the covers of every split table, and covers-per-table is one
		// of the two numbers a dining room is run on.
		PaymentStatus: "unpaid",
		Channel:       from.Channel,
		StatusHistory: []models.StatusEvent{{Status: models.StatusPending, At: now}},
		Check: &models.OrderCheck{
			OpenedAt:   from.Check.OpenedAt,
			OpenedByID: s.ID,
			OpenedBy:   s.Name,
			// The section is the table's, not the person splitting it: a
			// cashier dividing a bill at the counter does not become the
			// waiter of that table.
			ServerID:    from.Check.ServerID,
			ServerName:  from.Check.ServerName,
			SplitFromID: from.ID,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	// ⚠️ Food already cooking makes the new check the kitchen's too, exactly as
	// moving lines does. Otherwise fired lines end up on a check the kitchen
	// screen has never heard of.
	for _, it := range moved {
		if it.FiredAt != nil {
			split.QueuedAt = &now
			break
		}
	}
	applyCheckTotals(&split, bson.M{})

	res, err := h.Store.Orders.InsertOne(r.Context(), split)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	split.ID = res.InsertedID.(primitive.ObjectID)

	from.Items = kept
	set := bson.M{"items": kept, "updatedAt": now}
	applyCheckTotals(from, set)
	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		checkFilter(from.ID, s.BranchID), bson.M{"$set": set}); err != nil {
		// ⚠️ The new check is already in the database. Delete it rather than
		// leaving both halves holding the same food: a guest charged twice for
		// one dish is the one outcome worse than the split failing.
		_, _ = h.Store.Orders.DeleteOne(r.Context(),
			checkFilter(split.ID, s.BranchID))
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"check": viewCheck(from, now, s.ID),
		"split": viewCheck(&split, now, s.ID),
	})
}

func anyLive(items []models.OrderItem) bool {
	for _, it := range items {
		if it.Live() {
			return true
		}
	}
	return false
}
