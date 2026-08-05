package handlers

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The call log.
//
// Two things justify keeping it, and both are filters rather than the list
// itself: **who still has to be rung back**, and **which calls turned into
// orders**. A log nobody filters is a log nobody reads, so the endpoint leads
// with those two questions.

type callRequest struct {
	Direction string `json:"direction"`
	Phone     string `json:"phone"`
	UserID    string `json:"userId"`
	Name      string `json:"name"`
	Outcome   string `json:"outcome"`
	Note      string `json:"note"`
	// ISO time; empty clears the promise.
	CallbackAt   string `json:"callbackAt"`
	CallbackDone *bool  `json:"callbackDone"`
	Seconds      int    `json:"seconds"`
	// What the call produced, when it produced something.
	OrderID       string `json:"orderId"`
	ReservationID string `json:"reservationId"`
}

// AdminListCalls returns the call log, newest first.
//
// Filters: ?q= (number, name, note, order number), ?outcome= (comma-separated),
// ?direction=, ?operatorId=, ?from=&to= (YYYY-MM-DD, `to` inclusive),
// ?callback=open|done|any, ?limit=, ?before= (RFC3339 cursor).
func (h *Handler) AdminListCalls(w http.ResponseWriter, r *http.Request) {
	filter, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	q := r.URL.Query()

	if raw := strings.TrimSpace(q.Get("outcome")); raw != "" {
		ids := []string{}
		for _, part := range strings.Split(raw, ",") {
			if p := strings.TrimSpace(part); p != "" {
				ids = append(ids, p)
			}
		}
		if len(ids) > 0 {
			filter["outcome"] = bson.M{"$in": ids}
		}
	}
	if raw := strings.TrimSpace(q.Get("direction")); raw == "in" || raw == "out" {
		filter["direction"] = raw
	}
	if raw := strings.TrimSpace(q.Get("operatorId")); raw != "" {
		id, err := objectID(raw)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid operatorId")
			return
		}
		filter["operatorId"] = id
	}

	// "Who is still owed a call back" is the one filter the shift is run from,
	// so it gets to be a first-class question rather than a note somebody has
	// to read for.
	switch strings.TrimSpace(q.Get("callback")) {
	case "open":
		filter["callbackAt"] = bson.M{"$ne": nil}
		filter["callbackDone"] = false
	case "done":
		filter["callbackAt"] = bson.M{"$ne": nil}
		filter["callbackDone"] = true
	case "any":
		filter["callbackAt"] = bson.M{"$ne": nil}
	}

	// The date window and the paging cursor both narrow `createdAt`, so they
	// are collected into one condition instead of overwriting each other.
	window := bson.M{}
	if raw := strings.TrimSpace(q.Get("from")); raw != "" {
		day, err := time.ParseInLocation("2006-01-02", raw, time.Local)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "from: sana noto'g'ri")
			return
		}
		window["$gte"] = day
	}
	if raw := strings.TrimSpace(q.Get("to")); raw != "" {
		day, err := time.ParseInLocation("2006-01-02", raw, time.Local)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "to: sana noto'g'ri")
			return
		}
		// Inclusive: "to 12 August" means the whole of the 12th.
		window["$lt"] = day.AddDate(0, 0, 1)
	}
	if raw := strings.TrimSpace(q.Get("before")); raw != "" {
		if ts, err := time.Parse(time.RFC3339, raw); err == nil {
			window["$lt"] = ts
		}
	}
	if len(window) > 0 {
		filter["createdAt"] = window
	}

	if raw := strings.TrimSpace(q.Get("q")); raw != "" {
		raw = strings.TrimPrefix(raw, "#")
		// A number typed as "+998 90 123 45 67" has to find a row stored as
		// 998901234567, so the search runs on the normalised form when the
		// input looks like a phone number at all.
		needle := raw
		if norm, ok := normalizePhone(raw); ok {
			needle = norm
		}
		rx := bson.M{"$regex": regexp.QuoteMeta(needle), "$options": "i"}
		rxRaw := bson.M{"$regex": regexp.QuoteMeta(raw), "$options": "i"}
		filter["$or"] = []bson.M{
			{"phone": rx}, {"customerName": rxRaw}, {"note": rxRaw},
			{"orderNumber": rxRaw}, {"reservationNumber": rxRaw},
			{"operatorName": rxRaw},
		}
	}

	limit := int64(50)
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 && v <= 300 {
		limit = int64(v)
	}
	// Open callbacks read best soonest-first: the list is a queue of promises,
	// not a history.
	sort := bson.D{{Key: "createdAt", Value: -1}}
	if q.Get("callback") == "open" {
		sort = bson.D{{Key: "callbackAt", Value: 1}}
	}
	opts := options.Find().SetSort(sort).SetLimit(limit)
	cur, err := h.Store.Calls.Find(r.Context(), filter, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	calls := []models.Call{}
	_ = cur.All(r.Context(), &calls)
	httpx.JSON(w, http.StatusOK, calls)
}

// AdminCreateCall records a conversation.
func (h *Handler) AdminCreateCall(w http.ResponseWriter, r *http.Request) {
	var req callRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	phone := strings.TrimSpace(req.Phone)
	if phone == "" {
		httpx.Error(w, http.StatusBadRequest, "telefon raqam kerak")
		return
	}
	if norm, ok := normalizePhone(phone); ok {
		phone = norm
	}
	outcome := strings.TrimSpace(req.Outcome)
	if !validCallOutcome(outcome) {
		httpx.Error(w, http.StatusBadRequest, "natijani tanlang")
		return
	}
	callbackAt, err := parseCallbackAt(req.CallbackAt)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// A callback with no time is a promise with no date — the list of people
	// owed a call back is the whole point, and it cannot hold a row that never
	// comes due.
	if outcome == models.CallOutcomeCallback && callbackAt == nil {
		httpx.Error(w, http.StatusBadRequest, "qayta qo'ng'iroq vaqtini yozing")
		return
	}

	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	call := models.Call{
		BranchID:     h.scopeBranch(r, scope),
		Direction:    directionOf(req.Direction),
		Phone:        phone,
		CustomerName: clampText(req.Name, 60),
		Outcome:      outcome,
		Note:         clampText(req.Note, 500),
		CallbackAt:   callbackAt,
		OperatorID:   h.adminID(r),
		OperatorName: h.adminName(r),
		Seconds:      clampSeconds(req.Seconds),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if id, err := objectID(strings.TrimSpace(req.UserID)); err == nil {
		call.UserID = id
	}
	h.attachCallResult(r, &call, req.OrderID, req.ReservationID)

	res, err := h.Store.Calls.InsertOne(r.Context(), call)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	call.ID = res.InsertedID.(primitive.ObjectID)
	httpx.JSON(w, http.StatusCreated, call)
}

// AdminUpdateCall corrects a record: the outcome, the note, the callback.
//
// Deliberately editable, unlike the audit log. A call is written down while
// somebody is talking — the outcome is often only clear a minute later, and an
// operator who cannot fix a mistyped row stops filling the log in at all.
func (h *Handler) AdminUpdateCall(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req callRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var call models.Call
	if err := h.Store.Calls.FindOne(r.Context(), bson.M{"_id": id}).Decode(&call); err != nil {
		httpx.Error(w, http.StatusNotFound, "qo'ng'iroq topilmadi")
		return
	}

	set := bson.M{"updatedAt": time.Now()}
	if outcome := strings.TrimSpace(req.Outcome); outcome != "" {
		if !validCallOutcome(outcome) {
			httpx.Error(w, http.StatusBadRequest, "natija noto'g'ri")
			return
		}
		set["outcome"] = outcome
	}
	set["note"] = clampText(req.Note, 500)
	if name := clampText(req.Name, 60); name != "" {
		set["customerName"] = name
	}
	if req.Seconds > 0 {
		set["seconds"] = clampSeconds(req.Seconds)
	}
	if req.CallbackDone != nil {
		set["callbackDone"] = *req.CallbackDone
	}
	// An empty string clears the callback; a missing field leaves it alone.
	if strings.TrimSpace(req.CallbackAt) != "" {
		at, err := parseCallbackAt(req.CallbackAt)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		set["callbackAt"] = at
	}
	h.attachCallResult(r, &call, req.OrderID, req.ReservationID)
	if !call.OrderID.IsZero() {
		set["orderId"], set["orderNumber"] = call.OrderID, call.OrderNumber
	}
	if !call.ReservationID.IsZero() {
		set["reservationId"] = call.ReservationID
		set["reservationNumber"] = call.ReservationNumber
	}

	if _, err := h.Store.Calls.UpdateByID(r.Context(), id, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.Store.Calls.FindOne(r.Context(), bson.M{"_id": id}).Decode(&call)
	httpx.JSON(w, http.StatusOK, call)
}

// attachCallResult links an order or a booking to a call, copying the number
// across so the log row reads on its own.
func (h *Handler) attachCallResult(r *http.Request, call *models.Call, orderID, reservationID string) {
	if id, err := objectID(strings.TrimSpace(orderID)); err == nil {
		var o models.Order
		if err := h.Store.Orders.FindOne(r.Context(), bson.M{"_id": id}).Decode(&o); err == nil {
			call.OrderID, call.OrderNumber = o.ID, o.Number
		}
	}
	if id, err := objectID(strings.TrimSpace(reservationID)); err == nil {
		var res models.Reservation
		if err := h.Store.Reservations.FindOne(r.Context(), bson.M{"_id": id}).Decode(&res); err == nil {
			call.ReservationID, call.ReservationNumber = res.ID, res.Number
		}
	}
}

// callStats is what the top of the call centre screen shows.
type callStats struct {
	Total     int            `json:"total"`
	Incoming  int            `json:"incoming"`
	Outgoing  int            `json:"outgoing"`
	ByOutcome map[string]int `json:"byOutcome"`
	// Calls that produced an order, and what share of the day that is. The
	// number a shift is actually judged by.
	Orders     int `json:"orders"`
	Conversion int `json:"conversion"` // percent
	// Promises outstanding, and how many of those are already late. Overdue is
	// separate because it is the only one that needs doing something about
	// right now.
	CallbacksOpen    int `json:"callbacksOpen"`
	CallbacksOverdue int `json:"callbacksOverdue"`
	// Per-operator tally for the same window.
	ByOperator []callOperatorStat `json:"byOperator"`
}

type callOperatorStat struct {
	OperatorID   string `json:"operatorId"`
	OperatorName string `json:"operatorName"`
	Calls        int    `json:"calls"`
	Orders       int    `json:"orders"`
}

// AdminCallStats summarises a window of the log: ?from=&to= (YYYY-MM-DD, both
// optional; empty = today).
func (h *Handler) AdminCallStats(w http.ResponseWriter, r *http.Request) {
	filter, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	q := r.URL.Query()
	from, to := strings.TrimSpace(q.Get("from")), strings.TrimSpace(q.Get("to"))
	// A call centre is run by the day. With no window asked for, today is the
	// honest default — an all-time average tells the person on shift nothing.
	if from == "" && to == "" {
		today := time.Now().Format("2006-01-02")
		from, to = today, today
	}
	window := bson.M{}
	if from != "" {
		day, err := time.ParseInLocation("2006-01-02", from, time.Local)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "from: sana noto'g'ri")
			return
		}
		window["$gte"] = day
	}
	if to != "" {
		day, err := time.ParseInLocation("2006-01-02", to, time.Local)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "to: sana noto'g'ri")
			return
		}
		window["$lt"] = day.AddDate(0, 0, 1)
	}
	if len(window) > 0 {
		filter["createdAt"] = window
	}

	cur, err := h.Store.Calls.Find(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var calls []models.Call
	_ = cur.All(r.Context(), &calls)

	out := callStats{ByOutcome: map[string]int{}, ByOperator: []callOperatorStat{}}
	byOp := map[string]*callOperatorStat{}
	for i := range calls {
		c := &calls[i]
		out.Total++
		if c.Direction == "out" {
			out.Outgoing++
		} else {
			out.Incoming++
		}
		out.ByOutcome[c.Outcome]++
		if c.Outcome == models.CallOutcomeOrder {
			out.Orders++
		}
		key := c.OperatorID.Hex()
		op, ok := byOp[key]
		if !ok {
			op = &callOperatorStat{OperatorID: key, OperatorName: c.OperatorName}
			byOp[key] = op
		}
		op.Calls++
		if c.Outcome == models.CallOutcomeOrder {
			op.Orders++
		}
	}
	if out.Total > 0 {
		out.Conversion = out.Orders * 100 / out.Total
	}
	for _, op := range byOp {
		out.ByOperator = append(out.ByOperator, *op)
	}

	// Outstanding callbacks are counted over the whole log, not the window: a
	// promise made last Tuesday is still owed today, and a screen that only
	// looked at today would quietly lose it.
	openFilter, _, err := h.orderScope(r)
	if err == nil {
		openFilter["callbackAt"] = bson.M{"$ne": nil}
		openFilter["callbackDone"] = false
		if n, err := h.Store.Calls.CountDocuments(r.Context(), openFilter); err == nil {
			out.CallbacksOpen = int(n)
		}
		openFilter["callbackAt"] = bson.M{"$ne": nil, "$lte": time.Now()}
		if n, err := h.Store.Calls.CountDocuments(r.Context(), openFilter); err == nil {
			out.CallbacksOverdue = int(n)
		}
	}
	httpx.JSON(w, http.StatusOK, out)
}

func validCallOutcome(s string) bool {
	for _, v := range models.CallOutcomes {
		if v == s {
			return true
		}
	}
	return false
}

func directionOf(s string) string {
	if strings.TrimSpace(s) == "out" {
		return "out"
	}
	return "in"
}

// clampSeconds keeps a stuck timer out of the numbers: a card left open
// overnight must not report an eight-hour phone call.
func clampSeconds(v int) int {
	const maxCall = 2 * 60 * 60
	if v < 0 {
		return 0
	}
	if v > maxCall {
		return maxCall
	}
	return v
}

// parseCallbackAt reads the datetime the panel sends. An empty string is not an
// error — it means there is no callback.
func parseCallbackAt(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	// `datetime-local` inputs send "2006-01-02T15:04" with no zone, which is
	// local time by definition. Full RFC3339 is accepted too, for anything
	// calling this API directly.
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04"} {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, errInvalidTime
}
