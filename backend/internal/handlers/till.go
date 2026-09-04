package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
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
	// Which guest is paying for it, and which course it belongs to. Zero means
	// "the table" and "with everything else" — the answer for every check
	// written before either existed.
	Guest  int `json:"guest,omitempty"`
	Course int `json:"course,omitempty"`
	// The dish itself, for a till rebuilding this check on its own disk.
	MenuItemID string `json:"menuItemId,omitempty"`
	// Present on voided lines, which stay visible on the till screen and count
	// for nothing. Hiding them would make the running total unexplainable.
	Void *models.CheckLineVoid `json:"void,omitempty"`
	// ---- Where this dish has got to ----
	//
	// ⚠️ **Timestamps, not flags, and every screen prints them as an age.**
	// "Ready two minutes ago" and "ready twenty minutes ago" are the difference
	// between a plate to collect and a plate to apologise for, and a boolean
	// says the same thing for both. The screens compute the age themselves from
	// these — see `dishstate.go` for who writes them.
	ReadyAt  *time.Time `json:"readyAt,omitempty"`
	ServedAt *time.Time `json:"servedAt,omitempty"`
	// How much of one portion this line is, as a percent. Absent means a whole
	// one, which is what every line was before parts existed.
	Portion int `json:"portion,omitempty"`
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
	// Who has this check open on another screen right now, if anybody.
	//
	// ⚠️ **Sent so the room can say so before somebody taps.** The server
	// refuses the edit either way, but a table that opens and then refuses
	// every button reads as a broken till; a table that says "Dilnoza is on
	// this one" reads as a colleague. Empty once the hold goes stale — see
	// models.CheckHoldTTL.
	HeldBy   string    `json:"heldBy,omitempty"`
	OpenedAt time.Time `json:"openedAt"`
	// How long this table has been open, computed here rather than in the
	// browser: the tablet by the till has the wrong clock as often as the one
	// at the pass does.
	OpenMin  int         `json:"openMin"`
	Lines    []checkLine `json:"lines"`
	Subtotal int         `json:"subtotal"`
	// Lines typed but not yet sent to the kitchen. The single number the floor
	// screen is read for: a table with unfired lines is a waiter who has not
	// finished, and it is the thing that gets forgotten during a rush.
	Unfired int `json:"unfired"`
	// Dishes the kitchen has finished and nobody has carried out yet.
	//
	// ⚠️ **The number the floor screen was missing.** "Unfired" says what the
	// waiter has not sent; this says what is standing under the lamp waiting
	// for them — which is the half of the job that goes cold, and the half the
	// room could only find out by walking to the pass.
	ReadyWaiting int    `json:"readyWaiting,omitempty"`
	Served       int    `json:"served,omitempty"`
	Comment      string `json:"comment,omitempty"`
	// What the room adds for service, and the rate that produced it. ⚠️ Both
	// on the check before it is paid: a total that grew between the bill and
	// the card machine is an argument at the door.
	Service        int `json:"service,omitempty"`
	ServicePercent int `json:"servicePercent,omitempty"`
	Total          int `json:"total"`
	// When the bill was printed — the table has asked to pay.
	PrecheckAt *time.Time `json:"precheckAt,omitempty"`
	ClosedAt   *time.Time `json:"closedAt,omitempty"`
	// ---- Only ever filled in for a closed check ----
	//
	// ⚠️ On the same shape rather than a second one: the sales list and the
	// open floor draw the same rows, and two shapes for one check is how a
	// dish's comment ends up visible in one place and missing in the other.
	ClosedBy      string              `json:"closedBy,omitempty"`
	PaymentMethod string              `json:"paymentMethod,omitempty"`
	PaymentStatus string              `json:"paymentStatus,omitempty"`
	Refund        *models.CheckRefund `json:"refund,omitempty"`
	// The fiscal filing, once there is one. Carried on the check rather than
	// fetched separately because the screen that needs it is the one showing the
	// guest their QR, and it is showing it while they wait.
	Fiscal *models.FiscalReceipt `json:"fiscal,omitempty"`
	// The reversal's own filing. ⚠️ Beside the sale's rather than replacing it:
	// the check ends its life carrying both documents, and the screen that says
	// "this filing is stuck" has to be able to say *which*.
	FiscalRefund *models.FiscalReceipt `json:"fiscalRefund,omitempty"`
}

// viewCheck renders an order as a check. The totals are computed from the live
// lines every time rather than read off the document, so a voided line cannot
// stay in a stored subtotal and quietly overcharge a table.
// viewCheck is one check as a screen sees it.
//
// ⚠️ `viewer` is who is looking, and it is not decoration: `heldBy` means
// "somebody **else** is working on this", which cannot be decided without
// knowing who is asking. Pass the employee's id; the zero id means the panel,
// which is nobody in particular.
func viewCheck(o *models.Order, now time.Time, viewer primitive.ObjectID) checkView {
	v := checkView{
		ID:           o.ID.Hex(),
		Number:       o.Number,
		Status:       o.Status,
		TableID:      o.TableID,
		TableNumber:  o.TableNumber,
		Comment:      o.Address.Comment,
		Lines:        make([]checkLine, 0, len(o.Items)),
		Fiscal:       o.Fiscal,
		FiscalRefund: o.FiscalRefund,
	}
	if o.Check != nil {
		v.Guests = o.Check.Guests
		v.ServerName = o.Check.ServerName
		// ⚠️ **"Somebody else has this open" is a fact about the person
		// asking, and this function did not know who that was.**
		//
		// Opening a check takes the hold *for you* — so the very next thing
		// this returned was your own name in `heldBy`, and both screens dutifully
		// warned you that somebody was editing the table you had just opened.
		// The predicate to answer this correctly already existed and was
		// already tested; nothing called it here.
		//
		// A zero viewer is nobody in particular — the panel reading a check
		// rather than a person working on one — and for them any live hold is
		// somebody else's, which is the honest answer.
		if o.Check.HeldByOther(viewer, now) {
			v.HeldBy = o.Check.HeldBy
		}
		v.OpenedAt = o.Check.OpenedAt
		v.ClosedAt = o.Check.ClosedAt
		v.PrecheckAt = o.Check.PrecheckAt
		v.ClosedBy = o.Check.ClosedBy
		if !o.Check.ServerID.IsZero() {
			v.ServerID = o.Check.ServerID.Hex()
		}
		if mins := int(now.Sub(o.Check.OpenedAt).Minutes()); mins > 0 {
			v.OpenMin = mins
		}
	}
	// ⚠️ Carried only once the check is closed. On an open table the payment
	// fields are either empty or, worse, left over from a provider QR that was
	// put up and never paid — and a row that says "payme" while the guests are
	// still eating is a row somebody will read as settled.
	if o.Check != nil && o.Check.ClosedAt != nil {
		v.PaymentMethod = o.PaymentMethod
		v.PaymentStatus = paymentStatusOf(o)
		v.Refund = o.Refund
	}
	for _, it := range o.Items {
		line := checkLine{
			LineID:   it.LineID,
			Name:     it.Name,
			Price:    it.Price,
			Qty:      it.Qty,
			Sum:      it.Price * it.Qty,
			Options:  it.Options,
			Comment:  it.Comment,
			Fired:    it.FiredAt != nil,
			Void:     it.Void,
			Guest:    it.Guest,
			Course:   it.Course,
			ReadyAt:  it.ReadyAt,
			ServedAt: it.ServedAt,
			Portion:  it.Portion,
		}
		// ⚠️ Which dish, not just its printed name. A till that has to rebuild a
		// check offline — or re-price one — cannot do either from a name, and
		// two dishes in a menu are allowed to share one.
		if !it.MenuItemID.IsZero() {
			line.MenuItemID = it.MenuItemID.Hex()
		}
		if it.Live() {
			v.Subtotal += line.Sum
			if it.FiredAt == nil {
				v.Unfired++
			}
		}
		v.Lines = append(v.Lines, line)
	}
	v.Served, v.ReadyWaiting = servedCount(o.Items)
	payable := v.Subtotal - o.DiscountTotal
	if payable < 0 {
		payable = 0
	}
	// ⚠️ Computed from the order's **own** copied rate rather than from the
	// branch settings, which is why this function still needs nothing but the
	// order. Twenty call sites draw a check; a rate passed in beside it would
	// eventually be forgotten at one of them, and the screen showing a total
	// different from the one the till charges is the worst version of this bug.
	v.ServicePercent = o.ServicePercent
	v.Service = serviceOn(payable, o.ServicePercent)
	v.Total = payable + v.Service
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
	// ⚠️ **The rate is copied onto the check when the table sits down**, not
	// read at payment. A restaurant that changes its service charge at eight
	// must not re-price the tables already eating, and a receipt reprinted next
	// month has to say what the guest actually paid — the same rule that copies
	// a discount onto the order by name and amount.
	//
	// ⚠️ **Tables only.** A service charge on a takeaway coffee is the version
	// of this feature guests complain about, and the branch setting cannot know
	// the difference — this line does.
	servicePercent := 0
	if tableID != "" && branch.Service.Enabled {
		servicePercent = branch.Service.Percent
	}
	order := models.Order{
		BranchID:       s.BranchID,
		ServicePercent: servicePercent,
		Number:         branchOrderNumber(branch.Code),
		Status:         models.StatusPending,
		Type:           "dinein",
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
	httpx.JSON(w, http.StatusCreated, viewCheck(&order, now, s.ID))
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
		// ⚠️ **Cancelled is not open, and nothing here used to say so.** A
		// check is closed by `closedAt` and abandoned by its status, and only
		// the first was being asked about — so a table merged onto another
		// stayed in this list forever, with its old lines and its old total.
		// The waiter saw the food on two tables and the room never let the
		// first one go. Same omission in both filters, because they were
		// written from the same idea of what "open" means.
		"status": bson.M{"$ne": models.StatusCancelled},
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
		// See openCheckOnTable: a cancelled check is not an open one, and a
		// merged-away table stayed in this list with its old total until
		// somebody noticed the food was on two bills.
		"status": bson.M{"$ne": models.StatusCancelled},
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
		out = append(out, viewCheck(&o, now, s.ID))
	}
	// ⚠️ **The stop list rides along with the poll the till already makes.**
	// The menu is loaded once, when the screen opens, so a dish that ran out
	// afterwards — tapped on another tablet, stopped by the kitchen system, or
	// past its batch for today — stayed pressable until somebody restarted the
	// till. Sending the ids here costs nothing (this list is asked for every
	// twenty seconds regardless) and needs no second request, no socket and no
	// second thing to poll.
	//
	// ⚠️ All four lists merged, because the screen asks one question — may I
	// sell this — and the four answers to "why not" belong on the stop-list
	// screen, which names them.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"checks":  out,
		"soldOut": h.soldOutIDs(r.Context(), s.BranchID),
	})
}

// soldOutIDs is every dish this branch cannot sell right now, as hex ids.
//
// ⚠️ Never nil: a branch with nothing stopped must serialise as `[]`, or the
// screen reads `null.length`. The trap this codebase has shipped twice.
func (h *Handler) soldOutIDs(ctx context.Context, branchID primitive.ObjectID) []string {
	out := []string{}
	var branch models.Branch
	if err := h.Store.Branches.FindOne(ctx, bson.M{"_id": branchID}).
		Decode(&branch); err != nil {
		return out
	}
	seen := map[primitive.ObjectID]bool{}
	// ⚠️ The manual list is asked through its method for the same reason the
	// limit list below is: a timed stop expires by being read, and a raw read
	// would keep a dish greyed on the sales grid after its hour was up — with
	// the stop list screen, which does ask properly, showing it as available.
	// Two screens, one branch, opposite answers.
	for _, id := range branch.SoldOut {
		if branch.IsManualSoldOut(id) && !seen[id] {
			seen[id] = true
			out = append(out, id.Hex())
		}
	}
	for _, list := range [][]primitive.ObjectID{
		branch.POSSoldOut, branch.StockSoldOut,
	} {
		for _, id := range list {
			if !seen[id] {
				seen[id] = true
				out = append(out, id.Hex())
			}
		}
	}
	// ⚠️ Asked through the method rather than read from the field: the limit
	// list expires by being read against today's date, and a raw read would
	// keep yesterday's batch stopping a dish all morning.
	for _, id := range branch.LimitSoldOut {
		if branch.IsLimitSoldOut(id) && !seen[id] {
			seen[id] = true
			out = append(out, id.Hex())
		}
	}
	return out
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
	httpx.JSON(w, http.StatusOK, viewCheck(o, time.Now(), s.ID))
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
	// ---- One person at a time ----
	//
	// ⚠️ **Claimed here rather than by a button**, because here is the one place
	// every edit of a check passes through. A separate "open this table" call
	// would have to be remembered by twelve handlers and by two screens, and
	// the first one to forget would be the one that overwrites somebody's work.
	//
	// ⚠️ **Reading is never blocked.** A cashier looking at a table a waiter is
	// serving is not a conflict — it is how a bill gets answered for over the
	// phone — and a screen that refused to *show* a check would be a worse
	// version of the problem it is solving.
	if r.Method != http.MethodGet && o.Check.IsOpen() {
		if !h.holdCheck(r.Context(), &o, s) {
			// 409, and it names the person. "Somebody else is editing this"
			// sends a waiter to look for a manager; a name sends them to the
			// colleague two metres away, which is the fix.
			httpx.Error(w, http.StatusConflict, heldByRefusal(o.Check.HeldBy))
			return nil, false
		}
	}
	return &o, true
}

// holdCheck takes or renews this person's hold, or reports that somebody else
// has it. See models.OrderCheck's hold fields for why it expires.
func (h *Handler) holdCheck(
	ctx context.Context, o *models.Order, s models.Staff,
) bool {
	now := time.Now()
	if o.Check.HeldByOther(s.ID, now) {
		return false
	}
	o.Check.HeldByID = s.ID
	o.Check.HeldBy = s.Name
	o.Check.HeldAt = &now
	// ⚠️ Written with `$set` on the three fields rather than by saving the
	// check: this runs before the handler has made its own change, and writing
	// the whole sub-document back would be exactly the overwrite the hold
	// exists to prevent.
	_, _ = h.Store.Orders.UpdateByID(ctx, o.ID, bson.M{"$set": bson.M{
		"check.heldById": s.ID,
		"check.heldBy":   s.Name,
		"check.heldAt":   now,
	}})
	return true
}

// releaseCheck lets go of a hold, if it is this person's to let go of.
//
// ⚠️ Guarded by the holder's id: a screen that released somebody else's hold
// would be a lock anybody could pick, which is not a lock.
func (h *Handler) releaseCheck(
	ctx context.Context, id primitive.ObjectID, s models.Staff,
) {
	_, _ = h.Store.Orders.UpdateOne(ctx,
		bson.M{"_id": id, "check.heldById": s.ID},
		bson.M{"$unset": bson.M{
			"check.heldById": "", "check.heldBy": "", "check.heldAt": "",
		}})
}

// heldByRefusal says who has the table, in words a waiter can act on.
func heldByRefusal(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "bu chekni hozir boshqa xodim tahrirlayapti"
	}
	return name + " hozir bu chekni tahrirlayapti"
}

// StaffReleaseCheck is the screen saying it has finished with a table.
//
// ⚠️ **A courtesy, not the mechanism.** The hold expires on its own, and that
// is what makes it safe; this only makes the wait shorter when somebody has
// actually walked away rather than crashed. A lock that depended on being
// released politely would be a lock that never lifts on the one evening a
// monoblock loses power.
func (h *Handler) StaffReleaseCheck(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	h.releaseCheck(r.Context(), id, s)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// requireOpen refuses to edit a check that has already been paid.
func requireOpen(w http.ResponseWriter, o *models.Order) bool {
	// ⚠️ **A cancelled check is not an open one, and `IsOpen` cannot see that.**
	// It asks about `ClosedAt` alone — which a cancellation never sets, because
	// a cancelled check was never closed, it was abandoned. So a check that had
	// already been merged away, or voided outright, read as perfectly editable.
	//
	// What that cost: merging a table onto a check that had itself been merged
	// away moved the food onto a bill that counts as nothing anywhere. The
	// screen showed it as an open table with a total, the takings did not
	// include it, and the only trace was a cancelled document nobody looks at.
	// Found by merging twice in a row, which is a thing a waiter does when two
	// tables join and then a third.
	if o.Status == models.StatusCancelled {
		httpx.Error(w, http.StatusConflict, "chek bekor qilingan — tahrirlab bo'lmaydi")
		return false
	}
	if o.Check.IsOpen() {
		return true
	}
	// A closed check is a receipt. Editing one is how a night's takings stop
	// matching the drawer, so it is refused here rather than merely discouraged
	// on the screen.
	httpx.Error(w, http.StatusConflict, "chek yopilgan — tahrirlab bo'lmaydi")
	return false
}
