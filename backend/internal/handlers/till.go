package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The in-house till: open checks, fired courses, payment.
//
// This is the server behind two screens that look nothing alike and share every
// rule: the **waiter's** floor screen (open a table, add dishes, send them to
// the kitchen) and the **cashier's** till (the same, plus money). Writing them
// against one set of endpoints is deliberate — the alternative is two code
// paths that price the same table differently depending on who is holding the
// tablet.
//
// Four rules shape everything below, and three of them are inherited rather
// than invented:
//
//   - **The branch comes from the employee, never from the request** (the KDS
//     rule). A waiter cannot reach another branch's floor by editing a URL.
//   - **`_id` alone never selects a document.** Every filter carries branchId,
//     so a guessed id is a 404 and not somebody else's dinner.
//   - **Money is recomputed on the server, always.** The tablet says which dish
//     and how many; it never says what it costs.
//   - **A check is invisible to the kitchen until a line is fired.** That is
//     `queuedAt` doing the job it already did for unpaid online orders, one
//     level down — see OrderItem.FiredAt.

// ---- Access ----

// tillDenial reports why this employee may not use a till screen, or "" when
// they may. Pure, so both permissions can be tested without a database — the
// same shape as kitchenDenial, for the same reason.
func tillDenial(s models.Staff, perm string) string {
	if !s.IsActive {
		return "hisob o'chirilgan — ma'muriyat bilan bog'laning"
	}
	if s.Can(perm) {
		return ""
	}
	if perm == models.PermCashier {
		// Deliberately specific. A waiter who has just been refused the payment
		// button needs to know it is the *payment* they lack, not the login —
		// otherwise the branch's answer is to share the cashier's password,
		// which is the outcome these two permissions exist to prevent.
		return "kassa amallariga ruxsat yo'q — kassirni chaqiring"
	}
	return "zal ekraniga ruxsat berilmagan — administratorga murojaat qiling"
}

// tillStaff authenticates the request and checks one permission.
func (h *Handler) tillStaff(w http.ResponseWriter, r *http.Request, perm string) (models.Staff, bool) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return models.Staff{}, false
	}
	if reason := tillDenial(s, perm); reason != "" {
		// 403, not 404: this person works here and can see the screen exists.
		// The scope rules below are the ones that answer with 404.
		httpx.Error(w, http.StatusForbidden, reason)
		return models.Staff{}, false
	}
	if s.BranchID.IsZero() {
		// Without a branch there is no floor to serve, no table map and no
		// kitchen to fire to. Said plainly rather than failing later on an
		// empty list, which reads as "the till is broken".
		httpx.Error(w, http.StatusForbidden, "hisobingiz filialga biriktirilmagan — administratorga murojaat qiling")
		return models.Staff{}, false
	}
	return s, true
}

// checkFilter scopes a lookup to one check inside the employee's own branch.
//
// ⚠️ The branchId is not an optimisation. Without it a waiter who types another
// branch's id into the URL edits that branch's table — the exact hole
// scopedOrderFilter closes for the panel.
func checkFilter(id, branchID primitive.ObjectID) bson.M {
	return bson.M{"_id": id, "branchId": branchID, "check": bson.M{"$exists": true}}
}

// ---- What the screens read ----

// checkLine is one line as the floor and till screens draw it.
type checkLine struct {
	LineID  string                   `json:"lineId"`
	Name    string                   `json:"name"`
	Price   int                      `json:"price"`
	Qty     int                      `json:"qty"`
	Sum     int                      `json:"sum"`
	Options []models.OrderItemOption `json:"options,omitempty"`
	Comment string                   `json:"comment,omitempty"`
	// Whether the kitchen has this line. Drives the only colour distinction on
	// the screen: what is cooking versus what is still a draft on the tablet.
	Fired bool `json:"fired"`
	// Present on voided lines, which stay visible on the till screen and count
	// for nothing. Hiding them would make the running total unexplainable.
	Void *models.CheckLineVoid `json:"void,omitempty"`
}

// checkView is one check. Everything a screen needs in one response — the till
// is used standing up, and a second round trip to price the table is a second
// chance for the network to be the reason the queue is not moving.
type checkView struct {
	ID          string             `json:"id"`
	Number      string             `json:"number"`
	Status      models.OrderStatus `json:"status"`
	TableID     string             `json:"tableId,omitempty"`
	TableNumber string             `json:"tableNumber,omitempty"`
	Guests      int                `json:"guests,omitempty"`
	ServerID    string             `json:"serverId,omitempty"`
	ServerName  string             `json:"serverName,omitempty"`
	OpenedAt    time.Time          `json:"openedAt"`
	// How long this table has been open, computed here rather than in the
	// browser: the tablet by the till has the wrong clock as often as the one
	// at the pass does.
	OpenMin  int         `json:"openMin"`
	Lines    []checkLine `json:"lines"`
	Subtotal int         `json:"subtotal"`
	// Lines typed but not yet sent to the kitchen. The single number the floor
	// screen is read for: a table with unfired lines is a waiter who has not
	// finished, and it is the thing that gets forgotten during a rush.
	Unfired  int        `json:"unfired"`
	Comment  string     `json:"comment,omitempty"`
	Total    int        `json:"total"`
	ClosedAt *time.Time `json:"closedAt,omitempty"`
	// The fiscal filing, once there is one. Carried on the check rather than
	// fetched separately because the screen that needs it is the one showing the
	// guest their QR, and it is showing it while they wait.
	Fiscal *models.FiscalReceipt `json:"fiscal,omitempty"`
}

// viewCheck renders an order as a check. The totals are computed from the live
// lines every time rather than read off the document, so a voided line cannot
// stay in a stored subtotal and quietly overcharge a table.
func viewCheck(o *models.Order, now time.Time) checkView {
	v := checkView{
		ID:          o.ID.Hex(),
		Number:      o.Number,
		Status:      o.Status,
		TableID:     o.TableID,
		TableNumber: o.TableNumber,
		Comment:     o.Address.Comment,
		Lines:       make([]checkLine, 0, len(o.Items)),
		Fiscal:      o.Fiscal,
	}
	if o.Check != nil {
		v.Guests = o.Check.Guests
		v.ServerName = o.Check.ServerName
		v.OpenedAt = o.Check.OpenedAt
		v.ClosedAt = o.Check.ClosedAt
		if !o.Check.ServerID.IsZero() {
			v.ServerID = o.Check.ServerID.Hex()
		}
		if mins := int(now.Sub(o.Check.OpenedAt).Minutes()); mins > 0 {
			v.OpenMin = mins
		}
	}
	for _, it := range o.Items {
		line := checkLine{
			LineID:  it.LineID,
			Name:    it.Name,
			Price:   it.Price,
			Qty:     it.Qty,
			Sum:     it.Price * it.Qty,
			Options: it.Options,
			Comment: it.Comment,
			Fired:   it.FiredAt != nil,
			Void:    it.Void,
		}
		if it.Live() {
			v.Subtotal += line.Sum
			if it.FiredAt == nil {
				v.Unfired++
			}
		}
		v.Lines = append(v.Lines, line)
	}
	v.Total = v.Subtotal - o.DiscountTotal
	if v.Total < 0 {
		v.Total = 0
	}
	return v
}

// ---- Opening a check ----

type openCheckRequest struct {
	// The table from the branch's floor plan. Optional: a counter sale has no
	// table, and refusing one would make the till unusable in half the places
	// that would buy it.
	TableID string `json:"tableId"`
	Guests  int    `json:"guests"`
	// Whose section this table is. Empty means the person opening it — the
	// common case, and the one a waiter must not have to answer twice.
	ServerID string `json:"serverId"`
}

// StaffOpenCheck opens a check on the floor.
func (h *Handler) StaffOpenCheck(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	var req openCheckRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	branch, err := h.branchByID(r, s.BranchID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	tableID, tableNumber := "", ""
	if req.TableID != "" {
		t, found := branchTable(branch, req.TableID)
		if !found {
			// The floor plan was redrawn under an open tablet. Named plainly:
			// "table not found" sends the waiter to look at the screen, an
			// empty success sends them to carry food to a check nobody can find.
			httpx.Error(w, http.StatusBadRequest, "bunday stol xaritada yo'q — ekranni yangilang")
			return
		}
		tableID, tableNumber = t.ID, t.Number
		// One open check per table. Two would mean the second waiter's dishes
		// land on a bill the first one closes, and the argument that follows
		// happens in front of the guest.
		busy, err := h.openCheckOnTable(r.Context(), s.BranchID, tableID)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if busy != nil {
			httpx.Error(w, http.StatusConflict,
				tableNumber+"-stolda ochiq chek bor ("+busy.Number+") — o'shanga qo'shing yoki yoping")
			return
		}
	}

	serverID, serverName := s.ID, s.Name
	if req.ServerID != "" {
		other, err := h.staffInBranch(r.Context(), req.ServerID, s.BranchID)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "ofitsiant topilmadi")
			return
		}
		serverID, serverName = other.ID, other.Name
	}

	now := time.Now()
	order := models.Order{
		BranchID: s.BranchID,
		Number:   branchOrderNumber(branch.Code),
		Status:   models.StatusPending,
		Type:     "dinein",
		// A dining room has no customer record, and inventing an empty one
		// would leave the panel's order list showing a blank row. The table is
		// who this is for, so the table is what it is called.
		Customer:    models.OrderCustomer{Name: guestLabel(tableNumber)},
		TableID:     tableID,
		TableNumber: tableNumber,
		Items:       []models.OrderItem{},
		// ⚠️ No QueuedAt. The kitchen learns about this check when a line is
		// fired and not one second earlier.
		PaymentStatus: "unpaid",
		Channel:       "pos",
		StatusHistory: []models.StatusEvent{{Status: models.StatusPending, At: now}},
		Check: &models.OrderCheck{
			OpenedAt:   now,
			OpenedByID: s.ID,
			OpenedBy:   s.Name,
			ServerID:   serverID,
			ServerName: serverName,
			Guests:     req.Guests,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	res, err := h.Store.Orders.InsertOne(r.Context(), order)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	order.ID = res.InsertedID.(primitive.ObjectID)
	httpx.JSON(w, http.StatusCreated, viewCheck(&order, now))
}

// branchTable finds a table on the branch's floor plan.
//
// ⚠️ Inactive tables are refused. A table taken out of service is usually
// broken, being repaired or standing in a section that is closed tonight, and
// seating a party there is a decision the floor plan already made.
func branchTable(b *models.Branch, id string) (models.FloorTable, bool) {
	if b == nil {
		return models.FloorTable{}, false
	}
	for _, t := range b.Booking.Tables {
		if t.ID == id && t.IsActive {
			return t, true
		}
	}
	return models.FloorTable{}, false
}

// staffInBranch loads a colleague, refusing anyone outside the caller's branch.
// A check may only be handed to somebody who actually works that floor.
func (h *Handler) staffInBranch(
	ctx context.Context, rawID string, branchID primitive.ObjectID,
) (*models.Staff, error) {
	id, err := objectID(rawID)
	if err != nil {
		return nil, err
	}
	var s models.Staff
	if err := h.Store.Staff.FindOne(ctx, bson.M{
		"_id": id, "branchId": branchID, "isActive": true,
	}).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// guestLabel names a till check for every screen that expects a customer.
func guestLabel(tableNumber string) string {
	if tableNumber == "" {
		return "Kassa"
	}
	return tableNumber + "-stol"
}

// openCheckOnTable returns the check already open on a table, if any.
func (h *Handler) openCheckOnTable(
	ctx context.Context, branchID primitive.ObjectID, tableID string,
) (*models.Order, error) {
	var o models.Order
	err := h.Store.Orders.FindOne(ctx, bson.M{
		"branchId":       branchID,
		"tableId":        tableID,
		"check.closedAt": bson.M{"$exists": false},
		"check.openedAt": bson.M{"$exists": true},
	}).Decode(&o)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// ---- Listing ----

// StaffChecks lists the checks open in the employee's branch.
//
// `?mine=1` narrows to the caller's own tables — the floor screen's default,
// because a waiter carrying three plates is answering "what have *I* not sent
// to the kitchen", and a list of the whole room buries it.
func (h *Handler) StaffChecks(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	filter := bson.M{
		"branchId":       s.BranchID,
		"check.openedAt": bson.M{"$exists": true},
		"check.closedAt": bson.M{"$exists": false},
	}
	if r.URL.Query().Get("mine") == "1" {
		filter["check.serverId"] = s.ID
	}
	// Oldest first: the table that has been open longest is the one nobody is
	// looking at, which is the same reason the pass sorts this way.
	cur, err := h.Store.Orders.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "check.openedAt", Value: 1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cur.Close(r.Context())

	now := time.Now()
	out := make([]checkView, 0, 32)
	for cur.Next(r.Context()) {
		var o models.Order
		if err := cur.Decode(&o); err != nil {
			continue
		}
		out = append(out, viewCheck(&o, now))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"checks": out})
}

// StaffCheck returns one check.
func (h *Handler) StaffCheck(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok {
		return
	}
	httpx.JSON(w, http.StatusOK, viewCheck(o, time.Now()))
}

// loadCheck fetches the check named in the URL, scoped to the caller's branch.
func (h *Handler) loadCheck(w http.ResponseWriter, r *http.Request, s models.Staff) (*models.Order, bool) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return nil, false
	}
	var o models.Order
	if err := h.Store.Orders.FindOne(r.Context(), checkFilter(id, s.BranchID)).Decode(&o); err != nil {
		httpx.Error(w, http.StatusNotFound, "chek topilmadi")
		return nil, false
	}
	return &o, true
}

// requireOpen refuses to edit a check that has already been paid.
func requireOpen(w http.ResponseWriter, o *models.Order) bool {
	if o.Check.IsOpen() {
		return true
	}
	// A closed check is a receipt. Editing one is how a night's takings stop
	// matching the drawer, so it is refused here rather than merely discouraged
	// on the screen.
	httpx.Error(w, http.StatusConflict, "chek yopilgan — tahrirlab bo'lmaydi")
	return false
}
