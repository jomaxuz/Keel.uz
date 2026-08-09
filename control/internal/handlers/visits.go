package handlers

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// An agent's day: where they mean to go, and what happened when they got there.
//
// ⚠️ **Planned before, judged after** — that order is the feature. A list of places
// visited is a report; a list of places to visit is a tool, and the same rows become
// the report once they are closed. So a visit is written with a date and a name, and
// closed with an outcome and a sentence.
//
// ⚠️ **A negative outcome requires the sentence.** "No" with no reason teaches
// nobody anything and is indistinguishable from "I did not go" a month later — and
// the reason is the only part that makes the next visit worth planning.
//
// Agents see their own; the owner sees everyone's. Same rule as the customer list,
// expressed the same way: a filter, not a check after the read.

type visitRequest struct {
	Place      string `json:"place"`
	Address    string `json:"address"`
	Phone      string `json:"phone"`
	PlannedFor string `json:"plannedFor"`
	// Closing it in the same call is normal: an agent who walked in unplanned writes
	// the row and the outcome together.
	Status   string `json:"status"`
	Outcome  string `json:"outcome"`
	Comment  string `json:"comment"`
	NextAt   string `json:"nextAt"`
	TenantID string `json:"tenantId"`
}

func (h *Handler) ListVisits(w http.ResponseWriter, r *http.Request) {
	u, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	filter := bson.M{}
	// ⚠️ Only the roles that see every customer see every visit. An agent reading a
	// colleague's calls is the same leak as reading their customers.
	if !models.CanSeeAllTenants(u.RoleOf()) {
		filter["agentId"] = u.ID
	} else if a := strings.TrimSpace(r.URL.Query().Get("agentId")); a != "" {
		if id, e := primitive.ObjectIDFromHex(a); e == nil {
			filter["agentId"] = id
		}
	}
	switch strings.TrimSpace(r.URL.Query().Get("status")) {
	case models.VisitPlanned:
		filter["status"] = models.VisitPlanned
	case models.VisitDone:
		filter["status"] = models.VisitDone
	}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		filter["place"] = bson.M{"$regex": regexp.QuoteMeta(q), "$options": "i"}
	}

	// Planned first and soonest first, then what is already done, newest first.
	// ⚠️ A promise whose date is not visible is not a promise — the same reasoning
	// the call centre's callbacks follow.
	cur, err := h.Store.Visits.Find(r.Context(), filter, options.Find().SetSort(bson.D{
		{Key: "status", Value: 1},
		{Key: "plannedFor", Value: 1},
		{Key: "createdAt", Value: -1},
	}).SetLimit(400))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.Visit{}
	_ = cur.All(r.Context(), &rows)

	// The three numbers an agent's own page is about, and the owner's overview too.
	var planned, positive, negative int
	for _, v := range rows {
		switch {
		case v.Status == models.VisitPlanned:
			planned++
		case v.Outcome == models.OutcomePositive:
			positive++
		case v.Outcome == models.OutcomeNegative:
			negative++
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"items": rows,
		"summary": map[string]int{
			"planned": planned, "positive": positive, "negative": negative,
		},
		"canSeeAll": models.CanSeeAllTenants(u.RoleOf()),
	})
}

func (h *Handler) CreateVisit(w http.ResponseWriter, r *http.Request) {
	u, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	var req visitRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	place := strings.TrimSpace(req.Place)
	if place == "" {
		httpx.Error(w, http.StatusBadRequest, "joy nomi kerak")
		return
	}
	day := strings.TrimSpace(req.PlannedFor)
	if day == "" {
		// Today, because a visit with no date sits in a list nobody sorts and never
		// gets made.
		day = time.Now().Format("2006-01-02")
	}
	v := models.Visit{
		// ⚠️ From the session, never from the request: a visit log an agent can
		// attribute to somebody else is not a log.
		AgentID:    u.ID,
		AgentName:  strings.TrimSpace(u.Name + " (" + u.Username + ")"),
		Place:      place,
		Address:    strings.TrimSpace(req.Address),
		Phone:      strings.TrimSpace(req.Phone),
		PlannedFor: day,
		Status:     models.VisitPlanned,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if id, e := primitive.ObjectIDFromHex(strings.TrimSpace(req.TenantID)); e == nil {
		v.TenantID = id
	}
	if strings.TrimSpace(req.Status) == models.VisitDone {
		if err := closeVisit(&v, req); err != nil {
			fail(w, err)
			return
		}
	}
	res, err := h.Store.Visits.InsertOne(r.Context(), v)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	v.ID, _ = res.InsertedID.(primitive.ObjectID)
	h.logConsole(r.Context(), u, "visit.create", place, v.PlannedFor)
	httpx.JSON(w, http.StatusOK, v)
}

func (h *Handler) UpdateVisit(w http.ResponseWriter, r *http.Request) {
	u, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	var existing models.Visit
	if err := h.Store.Visits.FindOne(r.Context(), bson.M{"_id": id}).Decode(&existing); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	// ⚠️ Own rows only, and 404 rather than 403 for somebody else's: an agent
	// guessing ids should not learn that a colleague's visit exists.
	if !models.CanSeeAllTenants(u.RoleOf()) && existing.AgentID != u.ID {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	var req visitRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if p := strings.TrimSpace(req.Place); p != "" {
		existing.Place = p
	}
	if a := strings.TrimSpace(req.Address); a != "" {
		existing.Address = a
	}
	if p := strings.TrimSpace(req.Phone); p != "" {
		existing.Phone = p
	}
	if d := strings.TrimSpace(req.PlannedFor); d != "" {
		existing.PlannedFor = d
	}
	if strings.TrimSpace(req.Status) == models.VisitDone {
		if err := closeVisit(&existing, req); err != nil {
			fail(w, err)
			return
		}
	} else if c := strings.TrimSpace(req.Comment); c != "" {
		existing.Comment = c
	}
	existing.UpdatedAt = time.Now()
	if _, err := h.Store.Visits.ReplaceOne(r.Context(), bson.M{"_id": id}, existing); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logConsole(r.Context(), u, "visit.update", existing.Place, existing.Outcome)
	httpx.JSON(w, http.StatusOK, existing)
}

// closeVisit records the outcome, and refuses a negative one with no reason.
func closeVisit(v *models.Visit, req visitRequest) error {
	outcome := strings.TrimSpace(req.Outcome)
	switch outcome {
	case models.OutcomePositive, models.OutcomeNegative, models.OutcomeCallback:
	default:
		return &httpError{code: http.StatusBadRequest, msg: "natijani tanlang"}
	}
	comment := strings.TrimSpace(req.Comment)
	if outcome == models.OutcomeNegative && comment == "" {
		return &httpError{code: http.StatusBadRequest,
			msg: "salbiy natija uchun izoh majburiy — sababsiz \"yo'q\" keyingi tashrifga yordam bermaydi"}
	}
	if outcome == models.OutcomeCallback && strings.TrimSpace(req.NextAt) == "" {
		// Same rule the call centre's callbacks follow: a commitment with no date is
		// not a commitment.
		return &httpError{code: http.StatusBadRequest, msg: "qayta borish sanasini kiriting"}
	}
	now := time.Now()
	v.Status = models.VisitDone
	v.Outcome = outcome
	v.Comment = comment
	v.NextAt = strings.TrimSpace(req.NextAt)
	v.VisitedAt = &now
	return nil
}

func (h *Handler) DeleteVisit(w http.ResponseWriter, r *http.Request) {
	u, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	filter := bson.M{"_id": id}
	if !models.CanSeeAllTenants(u.RoleOf()) {
		filter["agentId"] = u.ID
	}
	res, err := h.Store.Visits.DeleteOne(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.DeletedCount == 0 {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	h.logConsole(r.Context(), u, "visit.delete", id.Hex(), "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
