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
	"context"
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
		// ⚠️ **The refusal names the role it read.** "Ruxsat yo'q" sends
		// somebody to check a permission they may have just granted — and the
		// commonest cause is that they granted it to a *different* role from
		// the one this account holds, or renamed a role so the shipped grant
		// skipped it. Saying which role was read turns a dead end into an
		// instruction.
		msg := "bozorlik ro'yxatini yozishga ruxsat berilmagan — administratorga murojaat qiling"
		if s.RoleName != "" {
			msg = "«" + s.RoleName + "» rolida bozorlik ro'yxatini yozish ruxsati yo'q — " +
				"administrator uni Xodimlar → Rollar bo'limidan qo'shishi kerak"
		}
		httpx.Error(w, http.StatusForbidden, msg)
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
	// ⚠️ **The whole catalogue, not only what is short**, and that was a real
	// hole rather than a convenience. A list can perfectly well ask for
	// something that is above its minimum — a holiday is coming, a supplier is
	// closing — and with only the shortage on offer the writer had to type the
	// name by hand, which creates a *second* ingredient no tech card points at.
	// The screen was quietly manufacturing duplicates.
	//
	// ⚠️ **Without prices.** `StaffBuyCatalog` carries what things cost because
	// the buyer needs it to spot a typo at a stall; this screen is opened on a
	// till shared by the room, and every buying price in the building is not a
	// thing to leave on it — the same concern `PermStock` is written around.
	packs := map[primitive.ObjectID]models.Ingredient{}
	sources := map[primitive.ObjectID]string{}
	catalog := []map[string]any{}
	for _, in := range h.scopedIngredients(r.Context(), brand) {
		if in.HasPack() {
			packs[in.ID] = in
		}
		sources[in.ID] = in.From()
		// A prep item is cooked, not bought — offering sauce would put a list
		// in somebody's hand asking them to buy something nobody sells.
		if in.DerivedOnly() {
			continue
		}
		row := map[string]any{
			"ingredientId": in.ID.Hex(), "name": in.Name, "unit": in.Unit,
			// ⚠️ **Shown while the list is being written, not only after it is
			// sent.** The split happens on the server either way, but a barman
			// who cannot see that lemons are going to a market and cola is
			// coming off a shelf has no way to notice the one line that is
			// filed wrongly — and the wrong filing surfaces as a request that
			// sat all morning on a phone belonging to somebody who was never
			// going to answer it.
			"source": in.From(),
		}
		if in.HasPack() {
			row["packName"], row["packQty"] = in.PackName, in.PackQty
		}
		catalog = append(catalog, row)
	}
	sort.SliceStable(catalog, func(i, j int) bool {
		return catalog[i]["name"].(string) < catalog[j]["name"].(string)
	})
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
				"source":       models.SourceMarket,
			}
			// How the market sells it, so the form can offer the choice — see
			// models/ingredient.go.
			if id, err := primitive.ObjectIDFromHex(row.IngredientID); err == nil {
				if in, found := packs[id]; found {
					out["packName"], out["packQty"] = in.PackName, in.PackQty
				}
				if in, found := sources[id]; found {
					out["source"] = in
				}
			}
			rows = append(rows, out)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i]["name"].(string) < rows[j]["name"].(string)
	})
	httpx.JSON(w, http.StatusOK, map[string]any{
		"rows": rows, "since": since, "catalog": catalog,
	})
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

// StaffCreateBuyOrder writes a list and sends it — to one desk or to two.
//
// ⚠️ **The person writing it never chooses who answers it.** A barman with an
// empty bar writes "5 blocks of cola, 5 kg of sugar, 5 kg of lemons, 5 kg of
// oranges" and is done. Cola and sugar are in the building; the fruit is at a
// market. Sorting that is knowledge about the store, which is exactly the thing
// his job does not involve — so the catalogue answers it (`ingredient.From`) and
// the list splits itself.
//
// ⚠️ **The split makes two documents, not two kinds of line.** See
// models/shoppingorder.go: the storekeeper answers in ten minutes and the buyer
// at nine, and a shared document would spend the morning in a state neither of
// them has a word for.
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

	// The catalogue, read once: every line needs its unit and its source, and a
	// lookup per line would be a query per row on a list of thirty.
	known := map[primitive.ObjectID]models.Ingredient{}
	for _, in := range h.scopedIngredients(ctx, brand) {
		known[in.ID] = in
	}

	// ⚠️ **Keyed by source, and the ids keep counting across both.** A line id
	// only has to be unique inside its own document, but two lists written in
	// the same second and read side by side on a panel are much easier to talk
	// about when no id appears twice.
	split := map[string][]models.ShoppingLine{}
	for i, l := range req.Lines {
		name := clampText(strings.TrimSpace(l.Name), 80)
		line := models.ShoppingLine{
			ID:   "l" + itoa(i+1),
			Qty:  l.Qty,
			Note: clampText(strings.TrimSpace(l.Note), 120),
			Name: name,
		}
		// ⚠️ **A name the catalogue does not have goes to the market.** It is
		// something nobody has ever put on a shelf here, so a storekeeper
		// opening the list would be asked to find a thing the store has never
		// heard of — and the request would sit unanswered while looking, on
		// every screen, exactly like one that was being dealt with.
		source := models.SourceMarket
		if id, err := primitive.ObjectIDFromHex(strings.TrimSpace(l.IngredientID)); err == nil {
			if in, found := known[id]; found {
				line.IngredientID = in.ID
				source = in.From()
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
		split[source] = append(split[source], line)
	}
	if len(split) == 0 {
		httpx.Error(w, http.StatusBadRequest, "hech bo'lmasa bitta qator kerak")
		return
	}

	// ⚠️ The day is a local calendar date, formatted here rather than parsed
	// out of a timestamp — see models/shoppingorder.go for why it is a string.
	forDate := strings.TrimSpace(req.ForDate)
	if forDate == "" {
		forDate = time.Now().In(time.Local).Format("2006-01-02")
	}
	note := clampText(strings.TrimSpace(req.Note), 200)
	supply := h.supplyBranchFor(ctx, s.BranchID)

	now := time.Now()
	// ⚠️ **One group id for the whole request, minted even when only one half
	// exists.** The panel's question is "what did the barman ask for at six",
	// and an id that appears only on split requests would make that question two
	// questions with two answers.
	group := primitive.NewObjectID()
	out := []models.ShoppingOrder{}
	// ⚠️ Iterated in a fixed order rather than over the map: a response whose
	// two halves swap places between saves is a response that reads as a bug on
	// the screen showing it.
	for _, source := range []string{models.SourceMarket, models.SourceStore} {
		lines := split[source]
		if len(lines) == 0 {
			continue
		}
		order := models.ShoppingOrder{
			BranchID: s.BranchID, ForDate: forDate,
			Source: source, GroupID: group,
			// ⚠️ **The market half is answered where it was asked.** A buyer
			// brings food back to the kitchen he shops for; only the store half
			// can belong to another branch.
			SupplyBranchID: s.BranchID,
			Status:         models.ShoppingSent, Lines: lines,
			Note:      note,
			CreatedBy: s.Name, CreatedByID: s.ID,
			CreatedAt: now, UpdatedAt: now,
		}
		if source == models.SourceStore {
			order.SupplyBranchID = supply
		}
		res, err := h.Store.BuyOrders.InsertOne(ctx, order)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		order.ID = oidOf(res.InsertedID)
		out = append(out, order)
		h.logAction(r, "buyorder.create", "buyorder", order.ID.Hex(), s.Name,
			forDate+" — "+source+", "+itoa(len(lines))+" qator")
		// ⚠️ The phone that has to answer it is told, or the routing is only
		// half a feature — see handlers/buyorderflow.go.
		h.notifyOrderQueue(order, len(lines))
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{"orders": out})
}

// supplyBranchFor is the branch that answers this one's store requests.
//
// ⚠️ **Its own store room unless the restaurant says otherwise**, and that is
// every install that has ever existed: `branch.supplyBranchId` is empty, the
// request stays where it was written, and answering it moves no stock. A chain
// that points a branch at its central store gets the other reading, and the
// difference is a van rather than an errand — see StaffShipBuyOrder.
//
// ⚠️ **A branch never supplies itself.** Read literally, that would build a
// dispatch whose two ends are the same shelf: the same kilo subtracted and
// added, and a slip nobody can accept.
func (h *Handler) supplyBranchFor(
	ctx context.Context, branch primitive.ObjectID,
) primitive.ObjectID {
	if branch.IsZero() {
		return branch
	}
	var b models.Branch
	if err := h.Store.Branches.FindOne(ctx, bson.M{"_id": branch}).Decode(&b); err != nil {
		return branch
	}
	if b.SupplyBranchID.IsZero() || b.SupplyBranchID == branch {
		return branch
	}
	return b.SupplyBranchID
}

// StaffBuyOrders is every list this account has a part in, newest first.
//
// ⚠️ **One endpoint for all three jobs, and the same document for each of
// them.** The person who wrote the list, the buyer who shops it and the
// storekeeper who picks it are looking at one record — a "my orders" view per
// job would be three readings of the same morning, free to disagree about what
// was asked for, which is the one thing a supervision cannot afford. Which rows
// a phone *draws* is the app's business; what it may read is this.
func (h *Handler) StaffBuyOrders(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	// Any of the three halves of the job may read it.
	if !s.Can(models.PermBuyOrder) && !s.Can(models.PermBuy) &&
		!s.Can(models.PermStockIssue) {
		httpx.Error(w, http.StatusForbidden, "ruxsat berilmagan")
		return
	}
	// ⚠️ **Two ways in, and the second one is the only reason a chain works.**
	// A list belongs to the branch that wrote it; a store request can be
	// answered by another branch's storekeeper, and their phone would otherwise
	// never see a request addressed to their shelf.
	//
	// ⚠️ The store condition names the source as well as the branch. Without it
	// a branch that supplies itself — every ordinary restaurant — would match
	// its own market lists twice, and `$or` would hand the app each of them
	// once, which is right today and quietly wrong the first time the two
	// conditions stop overlapping.
	filter := bson.M{"$or": []bson.M{
		{"branchId": s.BranchID},
		{"supplyBranchId": s.BranchID, "source": models.SourceStore},
	}}
	// ⚠️ **"Open" means "not finished", which now includes what is on its way.**
	// A list somebody has shopped and nobody has counted is the most open thing
	// in this feature: it is the exact moment goods are in a bag in a corridor,
	// and a screen that hid it would hide the only step this was built for.
	if r.URL.Query().Get("open") == "1" {
		filter["status"] = bson.M{"$ne": models.ShoppingDone}
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

// orderActor loads one list and answers whether this employee is the person it
// is waiting on.
//
// ⚠️ **The permission depends on the document, so it cannot be a middleware.**
// The same route carries a market run and a store issue, and they are answered
// by two different people with two different permissions in — in a chain — two
// different buildings. A single gate in front of the route would have to grant
// the union of both, which is to say it would grant a buyer the storekeeper's
// button and call it access control.
func (h *Handler) orderActor(
	w http.ResponseWriter, r *http.Request, id primitive.ObjectID,
) (models.Staff, models.ShoppingOrder, bool) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return models.Staff{}, models.ShoppingOrder{}, false
	}
	var order models.ShoppingOrder
	if err := h.Store.BuyOrders.FindOne(r.Context(), bson.M{"_id": id}).Decode(&order); err != nil {
		httpx.Error(w, http.StatusNotFound, "ro'yxat topilmadi")
		return models.Staff{}, models.ShoppingOrder{}, false
	}
	// ⚠️ **The branch is compared here rather than put in the filter, unlike
	// everywhere else**, and only because it is two different branches for the
	// two sources: a store list is answered where it is picked, a market list
	// where it was written. The refusal is the same either way, and the document
	// is never returned to somebody who fails it.
	if order.FromStore() {
		if !s.Can(models.PermStockIssue) || order.SupplyBranchID != s.BranchID {
			httpx.Error(w, http.StatusForbidden,
				"skladdan tovar chiqarishga ruxsat berilmagan — administratorga murojaat qiling")
			return models.Staff{}, models.ShoppingOrder{}, false
		}
		return s, order, true
	}
	if msg := buyDenial(s); msg != "" || order.BranchID != s.BranchID {
		if msg == "" {
			msg = "bu ro'yxat boshqa filialniki"
		}
		httpx.Error(w, http.StatusForbidden, msg)
		return models.Staff{}, models.ShoppingOrder{}, false
	}
	return s, order, true
}

// StaffMarkBuyOrderLine records what one line came back with, or was picked as.
//
// ⚠️ **One handler for the buyer and the storekeeper**, because what they do to
// a row is the same act: they say how much of it they have. Only the price is
// the buyer's — a case of cola taken off our own shelf was bought once already,
// and asking a storekeeper what it costs would put a made-up figure into the
// price history that every dish with cola in it is then costed from.
func (h *Handler) StaffMarkBuyOrderLine(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	s, order, ok := h.orderActor(w, r, id)
	if !ok {
		return
	}
	var req struct {
		Qty   float64  `json:"qty"`
		Price wholeSom `json:"price"`
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

	// ⚠️ A list that has left is a list nothing may re-tick. Letting a tick
	// change it afterwards would leave the shelf and the record saying different
	// things about the same morning, with nothing to say which was right.
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
			qty, price := req.Qty, int(req.Price)
			// ⚠️ **A store issue carries no price**, whatever the phone sent.
			// The goods were bought once, and a second price written here would
			// enter the history as a purchase and recost every dish that uses
			// them — which is the loudest possible way for an errand between two
			// rooms to be wrong.
			if order.FromStore() {
				price = 0
			}
			if req.Pack && !order.Lines[i].IngredientID.IsZero() {
				var in models.Ingredient
				if err := h.Store.Ingredients.FindOne(r.Context(),
					bson.M{"_id": order.Lines[i].IngredientID}).Decode(&in); err == nil && in.HasPack() {
					qty = req.Qty * in.PackQty
					// Divided, not multiplied: a 3 000 so'm bunch weighing
					// 0.05 kg is 60 000 a kilo.
					if price > 0 {
						price = int(float64(price)/in.PackQty + 0.5)
					}
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
	_ = s
	if _, err := h.Store.BuyOrders.UpdateByID(r.Context(), id,
		bson.M{"$set": bson.M{"lines": order.Lines, "updatedAt": now}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}
