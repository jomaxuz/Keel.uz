package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Table booking.
//
// The owner draws the room in the admin panel (walls, then tables with their
// numbers); guests see that same plan on the site, tap a free table and leave a
// name, a phone and a time. Whether a table is free is decided **here**, never
// in the browser: two guests can tap the same table in the same second, and the
// site's own "busy" shading is only as fresh as its last poll.

// A booking that still holds its table. Cancelled ones release it immediately;
// finished ones ("done") are history.
var activeReservationStatuses = []models.ReservationStatus{
	models.ReservationPending,
	models.ReservationConfirmed,
	models.ReservationSeated,
}

// bookingSettings fills in the defaults a plan needs to be drawable. Settings
// live on the branch — each room is its own — so this takes the raw value.
func bookingSettings(b models.BookingSettings) models.BookingSettings {
	if b.SlotMinutes <= 0 {
		b.SlotMinutes = 90
	}
	if b.MaxDaysAhead <= 0 {
		b.MaxDaysAhead = 30
	}
	if b.MaxGuests <= 0 {
		b.MaxGuests = 20
	}
	if b.Width <= 0 {
		b.Width = 1000
	}
	if b.Height <= 0 {
		b.Height = 700
	}
	return bookingSlices(b)
}

// bookingSlices turns the plan's empty slices into empty arrays.
//
// ⚠️ **A nil slice marshals to JSON `null`, not `[]`**, and this codebase has
// now been bitten by it three times on this one struct. A room that was never
// drawn hands the browser `tables: null`, a restaurant that never split its
// floor hands it `zones: null`, and every consumer has to remember to guard
// before `.map` or `.filter`. One of them did not: the settings page threw on
// load — the whole page, not the section — because the zone editor filtered a
// null, and the only thing on screen was "something went wrong in the kitchen".
//
// ⚠️ **Split out from the defaults above on purpose.** The full
// `bookingSettings` also fills in slot lengths and plan sizes, which is right
// for a screen that has to draw a room and wrong for one that edits and saves
// the document back — that one would quietly write defaults into every branch
// somebody merely opened. Emptiness is not a default; it is the true shape of
// the same fact.
func bookingSlices(b models.BookingSettings) models.BookingSettings {
	if b.Tables == nil {
		b.Tables = []models.FloorTable{}
	}
	if b.Shapes == nil {
		b.Shapes = []models.FloorShape{}
	}
	if b.Zones == nil {
		b.Zones = []models.TableZone{}
	}
	return b
}

// bookingBranch is the room a request is about: the one asked for by id, or the
// brand's first active branch when the guest simply opened /bron.
func (h *Handler) bookingBranch(r *http.Request, rawID string) (*models.Branch, error) {
	if id := strings.TrimSpace(rawID); id != "" {
		oid, err := objectID(id)
		if err != nil {
			return nil, errors.New("filial noto'g'ri")
		}
		return h.branchByID(r, oid)
	}
	brand, err := h.publicBrand(r)
	if err != nil {
		return nil, err
	}
	var brandID primitive.ObjectID
	if brand != nil {
		brandID = brand.ID
	}
	return h.defaultBranch(r, brandID)
}

// overlapping finds active bookings that collide with [at, endsAt) — optionally
// for one table only. Half-open on both sides: a table freed at 19:00 can be
// booked again at 19:00. Always scoped to one branch: table "7" exists in every
// room, and one branch's bookings must never grey out another's plan.
func (h *Handler) overlapping(r *http.Request, branchID primitive.ObjectID, tableID string, at, endsAt time.Time) ([]models.Reservation, error) {
	filter := bson.M{
		"status": bson.M{"$in": activeReservationStatuses},
		"at":     bson.M{"$lt": endsAt},
		"endsAt": bson.M{"$gt": at},
	}
	if !branchID.IsZero() {
		filter["branchId"] = branchID
	}
	if tableID != "" {
		filter["tableId"] = tableID
	}
	cur, err := h.Store.Reservations.Find(r.Context(), filter)
	if err != nil {
		return nil, err
	}
	out := []models.Reservation{}
	if err := cur.All(r.Context(), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ---- Public ----

// BookingPlan returns the floor plan plus, for the requested moment, which
// tables are taken. Without ?at= it answers for right now.
func (h *Handler) BookingPlan(w http.ResponseWriter, r *http.Request) {
	branch, err := h.bookingBranch(r, r.URL.Query().Get("branchId"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	b := bookingSettings(branch.Booking)

	at := time.Now()
	if raw := strings.TrimSpace(r.URL.Query().Get("at")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "vaqt formati noto'g'ri")
			return
		}
		at = parsed
	}
	endsAt := at.Add(time.Duration(b.SlotMinutes) * time.Minute)

	busy, err := h.overlapping(r, branch.ID, "", at, endsAt)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Only what a guest may know: which table, and until when it is held.
	type busyRow struct {
		TableID string    `json:"tableId"`
		At      time.Time `json:"at"`
		EndsAt  time.Time `json:"endsAt"`
	}
	rows := make([]busyRow, 0, len(busy))
	for _, res := range busy {
		rows = append(rows, busyRow{TableID: res.TableID, At: res.At, EndsAt: res.EndsAt})
	}

	// ⚠️ **Unbookable zones are filtered out here, not hidden on the site.** The
	// takeaway counter's numbers are tables the till needs and a guest cannot
	// reserve, and a plan that carried them to the browser would put them one
	// CSS rule away from being reservable — and would tell every visitor how
	// the restaurant numbers its takeaway orders.
	plan := b
	plan.Tables = make([]models.FloorTable, 0, len(b.Tables))
	for _, tb := range b.Tables {
		if b.Bookable(tb) {
			plan.Tables = append(plan.Tables, tb)
		}
	}
	// Likewise the zone list: a guest has no use for one they cannot book into.
	plan.Zones = make([]models.TableZone, 0, len(b.Zones))
	for _, z := range b.Zones {
		if z.Bookable {
			plan.Zones = append(plan.Zones, z)
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"booking":    plan,
		"at":         at,
		"endsAt":     endsAt,
		"busy":       rows,
		"branchId":   branch.ID,
		"branchName": branch.Name,
	})
}

type createReservationRequest struct {
	TableID string `json:"tableId"`
	// Which room the table is in. Empty = the brand's first active branch, which
	// is all a single-restaurant install has.
	BranchID string `json:"branchId"`
	At       string `json:"at"`
	Guests   int    `json:"guests"`
	Customer struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
	} `json:"customer"`
	Comment string `json:"comment"`
}

// CreateReservation books one table for a signed-in guest. Every rule is
// re-checked here: the browser only proposes.
func (h *Handler) CreateReservation(w http.ResponseWriter, r *http.Request) {
	h.createReservation(w, r, false)
}

// AdminCreateReservation books a table from the panel — the phone rings more
// often than the website is used. Bookings taken by staff ignore the public
// rules: they may be for a room that is closed to online booking, and "in ten
// minutes" is a normal thing to say on the phone.
func (h *Handler) AdminCreateReservation(w http.ResponseWriter, r *http.Request) {
	h.createReservation(w, r, true)
}

func (h *Handler) createReservation(w http.ResponseWriter, r *http.Request, byStaff bool) {
	var req createReservationRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	branch, err := h.bookingBranch(r, req.BranchID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	b := bookingSettings(branch.Booking)
	if !b.Enabled && !byStaff {
		httpx.Error(w, http.StatusBadRequest, "bron vaqtincha yopiq")
		return
	}

	// Who is booking. Guests reach this endpoint only with a customer token —
	// their phone was verified by SMS at login — while staff bookings carry the
	// phone the caller gave over the line.
	var guest *models.User
	claims := middleware.ClaimsFrom(r.Context())
	if claims != nil && claims.UserID != "" {
		if id, err := objectID(claims.UserID); err == nil {
			var u models.User
			if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": id}).Decode(&u); err == nil {
				guest = &u
			}
		}
	}
	if guest == nil && !byStaff {
		httpx.Error(w, http.StatusUnauthorized, "bron uchun telefon raqamini SMS bilan tasdiqlang")
		return
	}

	name := clampText(req.Customer.Name, 80)
	phone := clampText(req.Customer.Phone, 30)
	if guest != nil {
		if name == "" {
			name = clampText(strings.TrimSpace(guest.FirstName+" "+guest.LastName), 80)
		}
		// The verified number is the one the restaurant will ring: a guest may
		// type a different one into the form, but only the confirmed number
		// stands behind the booking.
		phone = guest.Phone
	}
	if len([]rune(name)) < 2 {
		httpx.Error(w, http.StatusBadRequest, "ismingizni yozing")
		return
	}
	if len(strings.Map(keepDigits, phone)) < 9 {
		httpx.Error(w, http.StatusBadRequest, "telefon raqamini to'liq yozing")
		return
	}

	at, err := time.Parse(time.RFC3339, strings.TrimSpace(req.At))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "vaqtni tanlang")
		return
	}
	now := time.Now()
	if !byStaff {
		if at.Before(now.Add(time.Duration(b.MinNoticeMinutes) * time.Minute)) {
			httpx.Error(w, http.StatusBadRequest, "bu vaqt allaqachon o'tib ketgan")
			return
		}
		if at.After(now.AddDate(0, 0, b.MaxDaysAhead)) {
			httpx.Error(w, http.StatusBadRequest, "bu sana juda uzoq")
			return
		}
	}

	guests := req.Guests
	if guests <= 0 {
		guests = 1
	}
	if guests > b.MaxGuests {
		httpx.Error(w, http.StatusBadRequest, "mehmonlar soni juda ko'p")
		return
	}

	endsAt := at.Add(time.Duration(b.SlotMinutes) * time.Minute)

	var table *models.FloorTable
	if strings.TrimSpace(req.TableID) == "" {
		// Nobody named a table. That is the normal path for a restaurant that
		// hides its plan — the guest asked for a time and a party size, which is
		// what most people want to say — and it is also what a half-filled form
		// looks like, so the refusal below still has to exist.
		if !b.HidePlan {
			httpx.Error(w, http.StatusBadRequest, "stolni tanlang")
			return
		}
		free, err := h.freeTables(r, branch.ID, b, at, endsAt)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		table = pickTable(free, guests)
		if table == nil {
			// The same 409 the guest would have got by tapping a table that was
			// taken a second earlier: the restaurant is full at that moment, and
			// the honest answer is to pick another time rather than to accept a
			// booking there is no room for.
			httpx.Error(w, http.StatusConflict, "bu vaqtga bo'sh stol qolmadi")
			return
		}
	} else {
		for i := range b.Tables {
			if b.Tables[i].ID == req.TableID {
				table = &b.Tables[i]
				break
			}
		}
		// ⚠️ Bookable, not merely active: a takeaway counter's numbers are
		// tables the till needs and a guest cannot reserve. Asked through the
		// one function all three screens use, so a table cannot be reservable
		// on the booking page and not here.
		if table == nil || !b.Bookable(*table) {
			httpx.Error(w, http.StatusBadRequest, "bu stol mavjud emas")
			return
		}
		if table.Seats > 0 && guests > table.Seats {
			httpx.Error(w, http.StatusBadRequest, "bu stol buncha mehmonga kichik")
			return
		}
		clashes, err := h.overlapping(r, branch.ID, table.ID, at, endsAt)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if len(clashes) > 0 {
			// 409: the guest's own screen said it was free, and it no longer is.
			httpx.Error(w, http.StatusConflict, "bu stol shu vaqtga band")
			return
		}
	}

	res := models.Reservation{
		BranchID: branch.ID,
		// Bookings carry the branch prefix too: staff read both off the same
		// screen and must not have to guess which room a number belongs to.
		Number:      branchOrderNumber(branch.Code),
		TableID:     table.ID,
		TableNumber: table.Number,
		Customer:    models.OrderCustomer{Name: name, Phone: phone},
		Guests:      guests,
		At:          at,
		EndsAt:      endsAt,
		Status:      models.ReservationPending,
		Comment:     clampText(req.Comment, 300),
		StatusHistory: []models.ReservationEvent{
			{Status: models.ReservationPending, At: now},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	// The booking shows up on the guest's profile.
	if guest != nil {
		res.UserID = guest.ID
	}

	ins, err := h.Store.Reservations.InsertOne(r.Context(), res)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	res.ID = oidOf(ins.InsertedID)
	httpx.JSON(w, http.StatusCreated, res)
}

// freeTables is every table that is not taken during [at, endsAt).
//
// One query for the whole branch rather than one per table: the alternative
// walks the plan asking the database about each table in turn, which is the
// same shape as the dashboard bug — a loop of queries that gets slower with
// every table the restaurant draws.
func (h *Handler) freeTables(
	r *http.Request, branchID primitive.ObjectID, b models.BookingSettings, at, endsAt time.Time,
) ([]models.FloorTable, error) {
	taken, err := h.overlapping(r, branchID, "", at, endsAt)
	if err != nil {
		return nil, err
	}
	busy := make(map[string]bool, len(taken))
	for _, res := range taken {
		busy[res.TableID] = true
	}
	out := make([]models.FloorTable, 0, len(b.Tables))
	for _, tb := range b.Tables {
		if b.Bookable(tb) && !busy[tb.ID] {
			out = append(out, tb)
		}
	}
	return out, nil
}

// pickTable chooses which free table a party gets when they did not choose one.
//
// ⚠️ **The smallest one that fits.** Seating two people at the ten-seater
// because it happened to be first in the list is how a restaurant ends up
// turning away the party of ten an hour later — and it is invisible, because
// every individual booking looks fine. A table with no seat count is treated as
// "fits anything", which is what an unfilled field means on the plan editor, but
// it sorts last so a table somebody actually measured is preferred.
func pickTable(free []models.FloorTable, guests int) *models.FloorTable {
	var best *models.FloorTable
	for i := range free {
		tb := &free[i]
		if tb.Seats > 0 && tb.Seats < guests {
			continue
		}
		switch {
		case best == nil:
			best = tb
		case best.Seats == 0 && tb.Seats > 0:
			best = tb
		case tb.Seats > 0 && best.Seats > 0 && tb.Seats < best.Seats:
			best = tb
		}
	}
	return best
}

func keepDigits(r rune) rune {
	if r >= '0' && r <= '9' {
		return r
	}
	return -1
}

// TrackReservation lets a guest follow their booking by its number, the same
// way an order is tracked.
func (h *Handler) TrackReservation(w http.ResponseWriter, r *http.Request) {
	number := strings.TrimPrefix(strings.TrimSpace(chi.URLParam(r, "number")), "#")
	var res models.Reservation
	if err := h.Store.Reservations.FindOne(r.Context(), bson.M{"number": number}).Decode(&res); err != nil {
		httpx.Error(w, http.StatusNotFound, "bron topilmadi")
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

// UserReservations lists the signed-in customer's bookings.
func (h *Handler) UserReservations(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	id, err := objectID(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	opts := options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(100)
	cur, err := h.Store.Reservations.Find(r.Context(), bson.M{"userId": id}, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := []models.Reservation{}
	_ = cur.All(r.Context(), &out)
	httpx.JSON(w, http.StatusOK, out)
}

// ---- Admin ----

// AdminListReservations returns bookings, newest slot first.
// ?scope=upcoming|today|past|all, ?status=, ?q= (number, name, phone, table).
func (h *Handler) AdminListReservations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	if s := strings.TrimSpace(q.Get("status")); s != "" {
		filter["status"] = s
	}
	now := time.Now()
	switch q.Get("scope") {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		filter["at"] = bson.M{"$gte": start, "$lt": start.AddDate(0, 0, 1)}
	case "past":
		filter["endsAt"] = bson.M{"$lt": now}
	case "all":
		// everything
	default: // upcoming: still to happen, or happening right now
		filter["endsAt"] = bson.M{"$gte": now}
	}
	if raw := strings.TrimSpace(q.Get("q")); raw != "" {
		rx := textSearch(raw)
		filter["$or"] = []bson.M{
			{"number": rx}, {"customer.name": rx}, {"customer.phone": rx},
			{"tableNumber": rx},
		}
	}

	sort := 1 // upcoming reads best soonest-first
	if q.Get("scope") == "past" || q.Get("scope") == "all" {
		sort = -1
	}
	opts := options.Find().SetSort(bson.D{{Key: "at", Value: sort}}).SetLimit(300)
	cur, err := h.Store.Reservations.Find(r.Context(), filter, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := []models.Reservation{}
	_ = cur.All(r.Context(), &out)
	httpx.JSON(w, http.StatusOK, out)
}

type reservationStatusRequest struct {
	Status models.ReservationStatus `json:"status"`
	Reason string                   `json:"reason"`
}

// AdminUpdateReservationStatus moves a booking along: confirmed → seated → done,
// or cancelled with a reason the guest can read.
func (h *Handler) AdminUpdateReservationStatus(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req reservationStatusRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	switch req.Status {
	case models.ReservationPending, models.ReservationConfirmed,
		models.ReservationSeated, models.ReservationDone, models.ReservationCancelled:
	default:
		httpx.Error(w, http.StatusBadRequest, "noma'lum holat")
		return
	}
	if req.Status == models.ReservationCancelled && strings.TrimSpace(req.Reason) == "" {
		httpx.Error(w, http.StatusBadRequest, "bekor qilish sababini yozing")
		return
	}

	now := time.Now()
	set := bson.M{"status": req.Status, "updatedAt": now}
	update := bson.M{
		"$push": bson.M{"statusHistory": models.ReservationEvent{Status: req.Status, At: now}},
	}
	if req.Status == models.ReservationCancelled {
		set["cancelReason"] = clampText(req.Reason, 300)
	} else {
		update["$unset"] = bson.M{"cancelReason": ""}
	}
	update["$set"] = set
	if _, err := h.Store.Reservations.UpdateByID(r.Context(), id, update); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	var res models.Reservation
	_ = h.Store.Reservations.FindOne(r.Context(), bson.M{"_id": id}).Decode(&res)
	action := ActReservationStatus
	details := string(req.Status)
	if req.Status == models.ReservationCancelled {
		action = ActReservationCancel
		details = res.CancelReason
	}
	h.logAction(r, action, "reservation", id.Hex(), "#"+res.Number, details)
	httpx.JSON(w, http.StatusOK, res)
}

// AdminDeleteReservation removes a booking outright (mistaken entries).
func (h *Handler) AdminDeleteReservation(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var res models.Reservation
	_ = h.Store.Reservations.FindOne(r.Context(), bson.M{"_id": id}).Decode(&res)
	if _, err := h.Store.Reservations.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActReservationDelete, "reservation", id.Hex(), "#"+res.Number, "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
