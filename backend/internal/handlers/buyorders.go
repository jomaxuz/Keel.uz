package handlers

// ---- The shopping list somebody is sent to the market with ----
//
// ⚠️ **The half the buying had no record of.** A delivery says what came back.
// Nothing said what was *asked for* — so "we ran out of beef again" had no
// answer: nobody could tell whether it was never on the list, was on the list
// and not bought, or was bought and eaten faster than expected. Three different
// problems, three different fixes, and the restaurant could only see the
// symptom.
//
// ⚠️ **Two permissions, and the split is the supervision.** `buyorder` writes
// the list; `buy` goes and gets it. Held by one account the list stops being a
// check on the trip and becomes a note the buyer wrote to themselves — which is
// the whole thing this was asked for.
//
// ⚠️ **The list starts from the shortage the store already computed.** A blank
// page would be a second, competing shopping list — one the arithmetic produces
// and one a person types — and the buyer would have no way to tell which is
// real. So `shoppingList` fills the form and the person edits it: the same
// function the panel reads, for the same reason `soldOutHeldBy` is one
// function.

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// orderStaff authenticates somebody who may write a shopping list.
//
// ⚠️ **One permission for the till and the phone.** A cashier writes lists at
// the counter and a storekeeper writes them in the app; making that two
// permissions would be two switches an owner has to understand for one job. Who
// sees the section on which screen is a rule about screens — the Team app
// simply does not draw it for a cashier — and it is deliberately not a security
// boundary, because it is not one: both people may write a list.
func (h *Handler) orderStaff(w http.ResponseWriter, r *http.Request) (models.Staff, bool) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return models.Staff{}, false
	}
	if !s.IsActive {
		httpx.Error(w, http.StatusForbidden, "hisob o'chirilgan — ma'muriyat bilan bog'laning")
		return models.Staff{}, false
	}
	if !s.Can(models.PermBuyOrder) {
		httpx.Error(w, http.StatusForbidden,
			"bozorlik ro'yxatini yozishga ruxsat berilmagan — administratorga murojaat qiling")
		return models.Staff{}, false
	}
	if s.BranchID.IsZero() {
		httpx.Error(w, http.StatusForbidden,
			"hisobingiz filialga biriktirilmagan — administratorga murojaat qiling")
		return models.Staff{}, false
	}
	return s, true
}

// StaffBuyOrderDraft is the shortage the store has worked out, as a starting
// point for the list.
func (h *Handler) StaffBuyOrderDraft(w http.ResponseWriter, r *http.Request) {
	s, ok := h.orderStaff(w, r)
	if !ok {
		return
	}
	scope, brand := h.staffStockScope(r, s)
	groups, _, since, err := h.shoppingList(r, scope, s.BranchID, brand)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	packs := map[primitive.ObjectID]models.Ingredient{}
	for _, in := range h.scopedIngredients(r.Context(), brand) {
		if in.HasPack() {
			packs[in.ID] = in
		}
	}
	// ⚠️ Flattened and de-grouped. The panel groups by supplier because its
	// question is who to ring; somebody writing a list has one page to fill in.
	rows := []map[string]any{}
	for _, g := range groups {
		for _, row := range g.Rows {
			out := map[string]any{
				"ingredientId": row.IngredientID,
				"name":         row.Name,
				"unit":         row.Unit,
				"qty":          row.Suggested,
				"onHand":       row.OnHand,
			}
			// How the market sells it, so the form can offer the choice — see
			// models/ingredient.go.
			if id, err := primitive.ObjectIDFromHex(row.IngredientID); err == nil {
				if in, found := packs[id]; found {
					out["packName"], out["packQty"] = in.PackName, in.PackQty
				}
			}
			rows = append(rows, out)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i]["name"].(string) < rows[j]["name"].(string)
	})
	httpx.JSON(w, http.StatusOK, map[string]any{"rows": rows, "since": since})
}

type buyOrderRequest struct {
	// Which day the shopping is for, "YYYY-MM-DD". Empty means today.
	ForDate string                `json:"forDate"`
	Note    string                `json:"note"`
	Lines   []buyOrderLineRequest `json:"lines"`
}

type buyOrderLineRequest struct {
	IngredientID string  `json:"ingredientId"`
	Name         string  `json:"name"`
	Qty          float64 `json:"qty"`
	Note         string  `json:"note"`
	// Whether `Qty` counts packs — two sacks of flour — rather than the unit
	// the store keeps it in. ⚠️ Converted on the server for the reason
	// `buyRequestLine.Pack` gives: the factor is a fact about the ingredient
	// and the result is what somebody is sent to buy.
	Pack bool `json:"pack,omitempty"`
}

// StaffCreateBuyOrder writes a list and sends it.
func (h *Handler) StaffCreateBuyOrder(w http.ResponseWriter, r *http.Request) {
	s, ok := h.orderStaff(w, r)
	if !ok {
		return
	}
	var req buyOrderRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx := r.Context()
	_, brand := h.staffStockScope(r, s)

	// The catalogue, read once: every line needs its unit, and a lookup per
	// line would be a query per row on a list of thirty.
	known := map[primitive.ObjectID]models.Ingredient{}
	for _, in := range h.scopedIngredients(ctx, brand) {
		known[in.ID] = in
	}

	lines := make([]models.ShoppingLine, 0, len(req.Lines))
	for i, l := range req.Lines {
		name := clampText(strings.TrimSpace(l.Name), 80)
		line := models.ShoppingLine{
			// Position at the moment of writing is enough to mint an id: the
			// list is created once and never has rows inserted into it later.
			ID:   "l" + itoa(i+1),
			Qty:  l.Qty,
			Note: clampText(strings.TrimSpace(l.Note), 120),
			Name: name,
		}
		if id, err := primitive.ObjectIDFromHex(strings.TrimSpace(l.IngredientID)); err == nil {
			if in, found := known[id]; found {
				line.IngredientID = in.ID
				// ⚠️ **Stored in the unit the store keeps it in, always.** The
				// list travels to somebody else's morning, and a document that
				// sometimes held sacks and sometimes kilos would need every
				// reader to know which — including the arithmetic that turns
				// the finished trip into a delivery.
				if l.Pack && in.HasPack() {
					line.Qty = l.Qty * in.PackQty
				}
				// ⚠️ **Name and unit come off the catalogue, not off the
				// request.** A screen posting both would let the two disagree
				// from the first save, and the unit is the field a wrong value
				// in is invisible: five bunches typed into a field measured in
				// kilos is five kilos on the shelf.
				line.Name = in.Name
				line.Unit = in.Unit
			}
		}
		if line.Name == "" || line.Qty <= 0 {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		httpx.Error(w, http.StatusBadRequest, "hech bo'lmasa bitta qator kerak")
		return
	}

	// ⚠️ The day is a local calendar date, formatted here rather than parsed
	// out of a timestamp — see models/shoppingorder.go for why it is a string.
	forDate := strings.TrimSpace(req.ForDate)
	if forDate == "" {
		forDate = time.Now().In(time.Local).Format("2006-01-02")
	}

	now := time.Now()
	order := models.ShoppingOrder{
		BranchID: s.BranchID, ForDate: forDate,
		Status: models.ShoppingSent, Lines: lines,
		Note:      clampText(strings.TrimSpace(req.Note), 200),
		CreatedBy: s.Name, CreatedByID: s.ID,
		CreatedAt: now, UpdatedAt: now,
	}
	res, err := h.Store.BuyOrders.InsertOne(ctx, order)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	order.ID = oidOf(res.InsertedID)

	h.logAction(r, "buyorder.create", "buyorder", order.ID.Hex(), s.Name,
		forDate+" — "+itoa(len(lines))+" qator")
	httpx.JSON(w, http.StatusCreated, order)
}

// StaffBuyOrders is this branch's lists, newest first.
//
// ⚠️ **The same list for the person who wrote it and the person shopping it.**
// A separate "my orders" view would let the two disagree about what was asked
// for, which is the one thing a supervision cannot afford.
func (h *Handler) StaffBuyOrders(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	// Either half of the job may read it: whoever writes the list and whoever
	// shops it are looking at the same page.
	if !s.Can(models.PermBuyOrder) && !s.Can(models.PermBuy) {
		httpx.Error(w, http.StatusForbidden, "ruxsat berilmagan")
		return
	}
	filter := bson.M{"branchId": s.BranchID}
	if r.URL.Query().Get("open") == "1" {
		filter["status"] = models.ShoppingSent
	}
	cur, err := h.Store.BuyOrders.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(50))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.ShoppingOrder{}
	_ = cur.All(r.Context(), &rows)
	httpx.JSON(w, http.StatusOK, map[string]any{"orders": rows})
}

// StaffMarkBuyOrderLine records what came back for one line.
func (h *Handler) StaffMarkBuyOrderLine(w http.ResponseWriter, r *http.Request) {
	s, ok := h.buyStaff(w, r)
	if !ok {
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Qty   float64 `json:"qty"`
		Price int     `json:"price"`
		// Whether the figures count packs — three bunches at three thousand a
		// bunch. ⚠️ Converted here for the reason `buyRequestLine.Pack` gives:
		// the factor is a fact about the ingredient and the result lands on a
		// shelf when the trip is finished.
		Pack    bool `json:"pack,omitempty"`
		Missing bool `json:"missing"`
		// Undo a tick: the commonest correction at a market is "I ticked the
		// wrong row", and it must not need a manager.
		Clear bool `json:"clear"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	var order models.ShoppingOrder
	if err := h.Store.BuyOrders.FindOne(r.Context(),
		bson.M{"_id": id, "branchId": s.BranchID}).Decode(&order); err != nil {
		httpx.Error(w, http.StatusNotFound, "ro'yxat topilmadi")
		return
	}
	// ⚠️ A finished trip is a delivery now. Letting a tick change it afterwards
	// would leave the shelf and the list saying different things about the same
	// morning, with nothing to say which was right.
	if order.Status != models.ShoppingSent {
		httpx.Error(w, http.StatusConflict, "bu ro'yxat yopilgan")
		return
	}

	wanted := chi.URLParam(r, "lineId")
	found := false
	now := time.Now()
	for i := range order.Lines {
		if order.Lines[i].ID != wanted {
			continue
		}
		found = true
		switch {
		case req.Clear:
			order.Lines[i].GotQty, order.Lines[i].Price = 0, 0
			order.Lines[i].GotAt, order.Lines[i].Missing = nil, false
		case req.Missing:
			order.Lines[i].GotQty, order.Lines[i].Price = 0, 0
			order.Lines[i].Missing = true
			order.Lines[i].GotAt = &now
		default:
			if req.Qty <= 0 {
				httpx.Error(w, http.StatusBadRequest, "miqdorni yozing")
				return
			}
			qty, price := req.Qty, req.Price
			if req.Pack && !order.Lines[i].IngredientID.IsZero() {
				var in models.Ingredient
				if err := h.Store.Ingredients.FindOne(r.Context(),
					bson.M{"_id": order.Lines[i].IngredientID}).Decode(&in); err == nil && in.HasPack() {
					qty = req.Qty * in.PackQty
					// Divided, not multiplied: a 3 000 so'm bunch weighing
					// 0.05 kg is 60 000 a kilo.
					price = int(float64(req.Price)/in.PackQty + 0.5)
				}
			}
			order.Lines[i].GotQty = qty
			order.Lines[i].Price = price
			order.Lines[i].Missing = false
			order.Lines[i].GotAt = &now
		}
	}
	if !found {
		httpx.Error(w, http.StatusNotFound, "qator topilmadi")
		return
	}
	if _, err := h.Store.BuyOrders.UpdateByID(r.Context(), id,
		bson.M{"$set": bson.M{"lines": order.Lines, "updatedAt": now}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}

// StaffFinishBuyOrder turns what came back into one delivery.
//
// ⚠️ **The two screens meet here and nowhere else.** Ticking a line is a fact
// about the list; the shelf only moves when the trip is finished, in one
// document, through the same path the panel's delivery form takes. A line that
// raised stock the moment it was ticked would put food on the shelf while the
// buyer was still at the market — and an untick would have to take it off
// again, which is a correction nothing downstream could tell from a theft.
func (h *Handler) StaffFinishBuyOrder(w http.ResponseWriter, r *http.Request) {
	s, ok := h.buyStaff(w, r)
	if !ok {
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Supplier string `json:"supplier"`
		// ⚠️ The same offline guarantee the free-form market run has: a phone
		// with no signal retries, and without an id it minted the retry is a
		// second delivery.
		ClientID string `json:"clientId"`
	}
	_ = httpx.DecodeOptional(r, &req)

	var order models.ShoppingOrder
	if err := h.Store.BuyOrders.FindOne(r.Context(),
		bson.M{"_id": id, "branchId": s.BranchID}).Decode(&order); err != nil {
		httpx.Error(w, http.StatusNotFound, "ro'yxat topilmadi")
		return
	}
	if order.Status != models.ShoppingSent {
		// Answered with the order rather than an error: a phone that retried
		// after a timeout must be told the trip is closed, not that it failed.
		httpx.JSON(w, http.StatusOK, map[string]any{"order": order, "already": true})
		return
	}

	lines := []buyRequestLine{}
	for _, l := range order.Lines {
		if !l.Got() {
			continue
		}
		lines = append(lines, buyRequestLine{
			IngredientID: l.IngredientID.Hex(),
			// ⚠️ A line typed by hand keeps its name so the delivery can invent
			// the ingredient the same way a free-form run does — marked for
			// somebody to finish, never silently completed.
			NewName: l.Name,
			Qty:     l.GotQty,
			Price:   l.Price,
		})
	}
	if len(lines) == 0 {
		httpx.Error(w, http.StatusBadRequest, "hech narsa olinmagan")
		return
	}

	created, p, err := h.recordMarketRun(r, s, buyRequest{
		ClientID: req.ClientID,
		Supplier: req.Supplier,
		Note:     "bozorlik: " + order.ForDate,
		Lines:    lines,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	now := time.Now()
	if _, err := h.Store.BuyOrders.UpdateByID(r.Context(), id, bson.M{"$set": bson.M{
		"status": models.ShoppingBought, "purchaseId": p.ID,
		"doneAt": now, "updatedAt": now,
	}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	order.Status, order.PurchaseID, order.DoneAt = models.ShoppingBought, p.ID, &now

	httpx.JSON(w, http.StatusOK, map[string]any{
		"order": order, "purchase": p, "created": created,
	})
}
