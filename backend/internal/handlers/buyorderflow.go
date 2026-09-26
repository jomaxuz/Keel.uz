package handlers

// ---- The two halves nobody had a record of: it left, and it arrived ----
//
// ⚠️ **"Bought" was never the end of the morning, and treating it as the end is
// where things disappear.** The old flow had two states: a list was written, and
// then it was a delivery. Between those two facts a person carries a bag across
// a city, hands it to somebody at a back door, and goes home. Nothing recorded
// that hand-over, so a kilo that never arrived and a kilo that was never bought
// looked identical afterwards — and the count that eventually found the gap
// could only blame whoever was holding the clipboard.
//
// So there are three states now (models/shoppingorder.go), and the shelf moves
// on the third one:
//
//	sent     — somebody asked
//	shipped  — the buyer bought it, or the storekeeper picked it off the shelf
//	done     — the person who asked counted it and signed
//
// ⚠️ **The delivery is written at the far end, from what was counted.** The
// buyer's figures say what he believes he handed over; the shelf gains what the
// restaurant counted. Where the two differ, the difference is the record — and
// it is exactly the thing that had nowhere to be written before. A purchase
// created at the market instead would put food on a shelf while it was still in
// a bag on a bus, and every later correction would be indistinguishable from a
// theft.
//
// ⚠️ **A free-form market run is untouched and still lands immediately**
// (handlers/staffbuy.go). Nobody asked for it, so there is nobody to count it
// off: inventing an acceptance step for it would mean a delivery that sits
// unaccepted forever because the person it is waiting on does not exist.

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// StaffShipBuyOrder says the list has been answered and is on its way.
//
// ⚠️ **It does not create the delivery.** See the file header: what reaches a
// shelf is what somebody at the restaurant counted, and nobody has counted
// anything yet.
func (h *Handler) StaffShipBuyOrder(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	s, order, ok := h.orderActor(w, r, id)
	if !ok {
		return
	}
	if order.Status != models.ShoppingSent {
		// Answered with the order rather than an error: a phone that retried
		// after a timeout must be told the list has gone, not that it failed.
		httpx.JSON(w, http.StatusOK, map[string]any{"order": order, "already": true})
		return
	}
	anything := false
	for _, l := range order.Lines {
		if l.Got() {
			anything = true
			break
		}
	}
	if !anything {
		httpx.Error(w, http.StatusBadRequest, "hech narsa olinmagan")
		return
	}

	now := time.Now()
	set := bson.M{
		"status": models.ShoppingShipped, "shippedAt": now, "shippedBy": s.Name,
		"updatedAt": now,
	}

	// ⚠️ **A van only when two branches are genuinely involved.** A restaurant
	// whose store room is down the corridor moves nothing between shelves when
	// the barman is handed a case of cola: the ingredient has one home
	// (models/warehouse.go) and the till writes it off when the drink is rung
	// up. Recording an issue as stock leaving would subtract the same bottle
	// twice, and the count that found the gap would blame the barman.
	if order.FromStore() && order.SupplyBranchID != order.BranchID {
		d, err := h.dispatchFromOrder(r, s, order, now)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		set["dispatchId"] = d.ID
		order.DispatchID = d.ID
	}

	if _, err := h.Store.BuyOrders.UpdateByID(r.Context(), id, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	order.Status, order.ShippedAt, order.ShippedBy = models.ShoppingShipped, &now, s.Name
	h.logAction(r, "buyorder.ship", "buyorder", order.ID.Hex(), s.Name, order.ForDate)
	h.notifyOrderShipped(order, s.Name)
	httpx.JSON(w, http.StatusOK, map[string]any{"order": order})
}

// dispatchFromOrder turns a picked store list into the slip a chain already
// signs three times.
//
// ⚠️ **The existing document, not a second one that means the same thing.** A
// dispatch is what "one branch's goods on their way to another's" already is in
// this system — it is what the movement report reads, what the two balances are
// computed from, and what prints. A parallel record would be a second answer to
// "what left the central store this morning", and the two would disagree by the
// end of the first week.
func (h *Handler) dispatchFromOrder(
	r *http.Request, s models.Staff, order models.ShoppingOrder, at time.Time,
) (models.Dispatch, error) {
	ctx := r.Context()
	_, brand := h.staffStockScope(r, s)
	ings := map[primitive.ObjectID]models.Ingredient{}
	for _, in := range h.scopedIngredients(ctx, brand) {
		ings[in.ID] = in
	}
	rates := h.ingredientRates(ctx)

	lines := make([]models.DispatchLine, 0, len(order.Lines))
	value := 0.0
	for _, l := range order.Lines {
		if !l.Got() || l.IngredientID.IsZero() {
			continue
		}
		in, ok := ings[l.IngredientID]
		if !ok {
			continue
		}
		qty := round3(l.GotQty)
		lines = append(lines, models.DispatchLine{
			IngredientID: in.ID, Name: in.Name, Unit: in.Unit, Qty: qty,
		})
		// Value is carried, not created — the rule a transfer follows.
		value += qty * float64(models.PerUnit(in.Unit)) * rates[in.ID]
	}
	if len(lines) == 0 {
		return models.Dispatch{}, errDispatchEmpty
	}

	d := models.Dispatch{
		BrandID:      brand,
		FromBranchID: order.SupplyBranchID,
		ToBranchID:   order.BranchID,
		Number: fmt.Sprintf("%s/%d", local(at).Format("02.01"),
			h.dispatchCount(ctx, order.SupplyBranchID, at)+1),
		At:        at,
		Lines:     lines,
		Value:     int(math.Round(value)),
		Note:      "bozorlik: " + order.ForDate,
		By:        s.Name,
		ByID:      s.ID,
		CreatedAt: time.Now(),
	}
	res, err := h.Store.Dispatches.InsertOne(ctx, d)
	if err != nil {
		return models.Dispatch{}, err
	}
	d.ID = oidOf(res.InsertedID)
	return d, nil
}

// StaffAcceptBuyOrder is the person who asked, counting what turned up.
//
// ⚠️ **Whoever holds `buyorder` at the receiving branch may sign**, not only the
// account that wrote the list. Requests are written at six and answered at nine,
// and the barman who wrote this one has gone home — a rule that named him would
// leave every list nobody can close, which within a week teaches the restaurant
// to stop closing them at all.
//
// ⚠️ **The shipper is not refused**, though the two names are stored separately.
// A restaurant where one person does both on a quiet Sunday must still be able
// to finish the morning; what the record has to answer is *who said it arrived*,
// and two fields answer that whether or not they hold the same name.
func (h *Handler) StaffAcceptBuyOrder(w http.ResponseWriter, r *http.Request) {
	s, ok := h.orderStaff(w, r)
	if !ok {
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Lines []struct {
			LineID string  `json:"lineId"`
			Qty    float64 `json:"qty"`
		} `json:"lines"`
		// ⚠️ The same offline guarantee the market run has: a phone with no
		// signal retries, and without an id the retry is a second delivery.
		ClientID string `json:"clientId"`
		Supplier string `json:"supplier"`
	}
	_ = httpx.DecodeOptional(r, &req)

	// ⚠️ **The branch is in the filter, not in a check after it.** An id alone
	// must never select a document — scope.go at length. Signing for another
	// branch's list would put food on our shelf that arrived at theirs.
	var order models.ShoppingOrder
	if err := h.Store.BuyOrders.FindOne(r.Context(),
		bson.M{"_id": id, "branchId": s.BranchID}).Decode(&order); err != nil {
		httpx.Error(w, http.StatusNotFound, "ro'yxat topilmadi")
		return
	}
	if order.Status == models.ShoppingDone {
		httpx.JSON(w, http.StatusOK, map[string]any{"order": order, "already": true})
		return
	}
	if order.Status != models.ShoppingShipped {
		httpx.Error(w, http.StatusConflict, "bu ro'yxat hali yuborilmagan")
		return
	}

	counted := map[string]float64{}
	for _, l := range req.Lines {
		if l.Qty >= 0 {
			counted[l.LineID] = round3(l.Qty)
		}
	}
	// ⚠️ **A row nobody retyped keeps what it was told**, and does not become a
	// zero. Accepting without touching anything means "this is right", which is
	// the ordinary case; a default of zero would empty the whole list of the
	// person who was quickest to agree with it. See ShoppingLine.Took.
	for i := range order.Lines {
		if v, ok := counted[order.Lines[i].ID]; ok && v != order.Lines[i].GotQty {
			q := v
			order.Lines[i].TookQty = &q
		}
	}

	now := time.Now()
	set := bson.M{
		"lines": order.Lines, "status": models.ShoppingDone,
		"acceptedAt": now, "acceptedBy": s.Name,
		"doneAt": now, "updatedAt": now,
	}

	switch {
	case !order.FromStore():
		// The market half becomes the delivery, in the quantities the
		// restaurant counted — see the file header.
		lines := []buyRequestLine{}
		for _, l := range order.Lines {
			if !l.Got() || l.Took() <= 0 {
				continue
			}
			lines = append(lines, buyRequestLine{
				IngredientID: l.IngredientID.Hex(),
				// ⚠️ A line typed by hand keeps its name so the delivery can
				// invent the ingredient the same way a free-form run does —
				// marked for somebody to finish, never silently completed.
				NewName: l.Name,
				Qty:     l.Took(),
				Price:   wholeSom(l.Price),
			})
		}
		if len(lines) == 0 {
			httpx.Error(w, http.StatusBadRequest, "hech narsa olinmagan")
			return
		}
		_, p, err := h.recordMarketRun(r, s, buyRequest{
			ClientID: req.ClientID,
			Supplier: req.Supplier,
			Note:     "bozorlik: " + order.ForDate,
			Lines:    lines,
		})
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		set["purchaseId"] = p.ID
		order.PurchaseID = p.ID

	case !order.DispatchID.IsZero():
		// ⚠️ **The van is signed for through its own document**, because that is
		// what the two branches' balances are computed from. Writing the
		// acceptance only onto the shopping list would leave the sending shelf
		// permanently short of goods this branch has already received.
		if err := h.acceptOrderDispatch(r, order, s.Name, now); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if _, err := h.Store.BuyOrders.UpdateByID(r.Context(), id, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	order.Status = models.ShoppingDone
	order.AcceptedAt, order.AcceptedBy, order.DoneAt = &now, s.Name, &now
	h.logAction(r, "buyorder.accept", "buyorder", order.ID.Hex(), s.Name, order.ForDate)
	httpx.JSON(w, http.StatusOK, map[string]any{"order": order})
}

// acceptOrderDispatch signs for the van this list became.
//
// ⚠️ **Only a quantity that differs is written.** `DispatchLine.Got` is absent
// until somebody says otherwise, and absent means "what was loaded" — filling it
// in everywhere would turn every ordinary arrival into a row the movement report
// has to compare against itself.
func (h *Handler) acceptOrderDispatch(
	r *http.Request, order models.ShoppingOrder, by string, at time.Time,
) error {
	took := map[primitive.ObjectID]float64{}
	for _, l := range order.Lines {
		if l.IngredientID.IsZero() || !l.Got() {
			continue
		}
		took[l.IngredientID] = l.Took()
	}
	var d models.Dispatch
	err := h.Store.Dispatches.FindOne(r.Context(), bson.M{
		"_id": order.DispatchID, "toBranchId": order.BranchID,
		"acceptedAt": bson.M{"$exists": false},
	}).Decode(&d)
	if err != nil {
		// Already signed for, from the panel's own dispatch screen. Not an
		// error: the goods are on the shelf either way, and refusing here would
		// leave the list nobody can close.
		return nil
	}
	lines := make([]models.DispatchLine, 0, len(d.Lines))
	for _, l := range d.Lines {
		if v, ok := took[l.IngredientID]; ok && v != l.Qty {
			q := v
			l.Got = &q
		}
		lines = append(lines, l)
	}
	_, err = h.Store.Dispatches.UpdateOne(r.Context(), bson.M{"_id": d.ID},
		bson.M{"$set": bson.M{"lines": lines, "acceptedAt": at, "acceptedBy": by}})
	return err
}

// ---- Telling the person whose morning it is ----
//
// ⚠️ **A queue nobody is told about is a queue nobody opens.** The whole point
// of routing a request is that it lands on the right phone; without a
// notification "landing" means the storekeeper happens to pull down the list
// between two other jobs. The old free-form market run already learned this in
// the other direction — the owner is told when food arrives — and this is the
// same fact one step earlier.
//
// ⚠️ **A notification, never a `LossAlert`.** The alert channel is for the one
// unusual event in a week; a restaurant asks for things off its own shelf every
// morning, and putting that there would spend the daily ceiling on the most
// ordinary act in the building and silence the channel on the night it matters.
// The same sentence buying's own note makes.

// notifyOrderQueue tells whoever answers this kind of request that one is
// waiting.
//
// ⚠️ **Sent to a permission at a branch, not to a person.** "The storekeeper"
// is a job two people share on alternate weeks, and a request addressed to
// whoever happens to hold an account would go unread every second week — which
// looks, from the panel, exactly like a storekeeper who ignores requests.
func (h *Handler) notifyOrderQueue(order models.ShoppingOrder, lines int) {
	perm, branch := models.PermBuy, order.BranchID
	title, body := "Yangi bozorlik ro'yxati", "Bozordan olinadigan "+itoa(lines)+" ta mahsulot"
	if order.FromStore() {
		perm, branch = models.PermStockIssue, order.SupplyBranchID
		title, body = "Skladdan so'rov", "Chiqarib berish kerak: "+itoa(lines)+" ta mahsulot"
	}
	h.notifyPerm(branch, perm, title, body, map[string]any{
		"type": "buyorder", "tab": "zakup", "id": order.ID.Hex(),
	})
}

// notifyOrderShipped tells the branch that asked that something is on its way
// and has to be counted.
func (h *Handler) notifyOrderShipped(order models.ShoppingOrder, by string) {
	h.notifyPerm(order.BranchID, models.PermBuyOrder,
		"Bozorlik yo'lda", by+" yubordi — sanab, qabul qilishingiz kerak",
		map[string]any{"type": "buyorder", "tab": "zakup", "id": order.ID.Hex()})
}

// notifyPerm sends one message to everybody at a branch who holds a permission.
//
// ⚠️ **Resolved through `Can`, not through a role name.** A restaurant that
// renamed «Omborchi» or gave the permission to a second role must still be
// reached — the rule CLAUDE.md states about `Staff.Position` applies to every
// other typed word too.
func (h *Handler) notifyPerm(
	branch primitive.ObjectID, perm, title, body string, data map[string]any,
) {
	if branch.IsZero() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		cur, err := h.Store.Staff.Find(ctx, bson.M{"branchId": branch, "isActive": true})
		if err != nil {
			return
		}
		var rows []models.Staff
		if err := cur.All(ctx, &rows); err != nil {
			return
		}
		h.withRoles(ctx, rows)
		for i := range rows {
			if rows[i].Can(perm) {
				h.notifyStaff(rows[i].ID, title, body, data)
			}
		}
	}()
}
