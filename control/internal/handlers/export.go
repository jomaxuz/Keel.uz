package handlers

import (
	"net/http"
	"strings"
	"time"

	"keel-control/internal/httpx"
	"keel-control/internal/middleware"
	"keel-control/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Opening and closing a customer's "download everything" button.
//
// **The data belongs to the restaurant and they must be able to leave with
// it.** What the console decides is not *whether* they may have it, but *when
// the button exists* — because the archive is one file holding every customer's
// name, phone and address plus the whole commercial history, and a permanent
// button turns any borrowed panel session into a silent complete copy.
//
// So the grant is deliberate, written down, attributable and dated:
//
//   - somebody at Keel turned it on, and their name is on it;
//   - they wrote why, in a field that cannot be left empty;
//   - it closes itself on a date, because the way this leaks is a switch left
//     on after a migration everybody has forgotten.
//
// The document is written straight into the **tenant's own database**, which is
// where their app reads it from. One document, one writer, one reader: nothing
// to keep in sync, and — the part that matters — no new path from a tenant
// container back to the control plane. That path not existing is exactly what
// makes one restaurant unable to reach another's data.
const exportGrantID = "export"

// Bounds on the grant window. A day is enough to notice a mistake; a month is
// long enough for a real migration and short enough that nobody forgets. The
// ceiling is the actual control here — an operator can always grant again.
const (
	exportMinDays = 1
	exportMaxDays = 30
)

type exportGrantDoc struct {
	ID        string    `bson:"_id" json:"-"`
	Enabled   bool      `bson:"enabled" json:"enabled"`
	Reason    string    `bson:"reason" json:"reason"`
	GrantedBy string    `bson:"grantedBy" json:"grantedBy"`
	GrantedAt time.Time `bson:"grantedAt" json:"grantedAt"`
	ExpiresAt time.Time `bson:"expiresAt" json:"expiresAt"`
	Downloads []struct {
		At    time.Time `bson:"at" json:"at"`
		By    string    `bson:"by" json:"by"`
		Bytes int64     `bson:"bytes" json:"bytes"`
		Files int       `bson:"files" json:"files"`
	} `bson:"downloads" json:"downloads"`
}

// tenantFromURL resolves the {id} in the path. Named apart from the rollout's
// own lookup because that one takes a context and an id — this is the request
// shape every console endpoint uses.
func (h *Handler) tenantFromURL(r *http.Request) *models.Tenant {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		return nil
	}
	return h.tenantByID(r.Context(), id)
}

// currentOperator names whoever is holding this console session. It goes on the
// grant: a permission with no name attached is one nobody answers for.
func currentOperator(r *http.Request) string {
	if c := middleware.From(r.Context()); c != nil {
		return c.Username
	}
	return "?"
}

// GetTenantExport reports the current grant, expired ones included.
//
// An expired grant is shown rather than hidden, with its download history: the
// question an operator asks months later is "did we ever open this, and did
// they take a copy", and a screen that only shows live grants answers neither.
func (h *Handler) GetTenantExport(w http.ResponseWriter, r *http.Request) {
	t := h.tenantFromURL(r)
	if t == nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	var doc exportGrantDoc
	err := h.Store.TenantDB(t.DBName()).Collection("export_grant").
		FindOne(r.Context(), bson.M{"_id": exportGrantID}).Decode(&doc)
	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"enabled": false, "downloads": []any{}})
		return
	}
	downloads := make([]any, 0, len(doc.Downloads))
	for _, d := range doc.Downloads {
		downloads = append(downloads, d)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"enabled":   doc.Enabled,
		"reason":    doc.Reason,
		"grantedBy": doc.GrantedBy,
		"grantedAt": doc.GrantedAt,
		"expiresAt": doc.ExpiresAt,
		// Computed here rather than compared in the browser: a clock-based
		// permission judged by the client's clock is not a permission.
		"active":    doc.Enabled && time.Now().Before(doc.ExpiresAt),
		"downloads": doc.Downloads,
	})
}

type exportGrantRequest struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
	// How long it stays open. Capped, never unbounded.
	Days int `json:"days"`
}

// PutTenantExport opens or closes the grant.
func (h *Handler) PutTenantExport(w http.ResponseWriter, r *http.Request) {
	t := h.tenantFromURL(r)
	if t == nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	var req exportGrantRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	coll := h.Store.TenantDB(t.DBName()).Collection("export_grant")

	// Closing needs no reason and no ceremony: the safe direction is always
	// allowed, immediately, by anyone who can reach this screen.
	if !req.Enabled {
		_, err := coll.UpdateOne(r.Context(), bson.M{"_id": exportGrantID},
			bson.M{"$set": bson.M{"enabled": false}}, options.Update().SetUpsert(true))
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}

	reason := strings.TrimSpace(req.Reason)
	if len(reason) < 5 {
		// Required, and long enough to be a sentence. "test" tells the next
		// person nothing, and this is the permission most likely to be asked
		// about a year later — usually by the customer.
		httpx.Error(w, http.StatusBadRequest, "sabab yozilishi shart (kamida 5 belgi)")
		return
	}
	days := req.Days
	if days < exportMinDays || days > exportMaxDays {
		httpx.Error(w, http.StatusBadRequest, "muddat 1–30 kun oralig'ida bo'lishi kerak")
		return
	}

	now := time.Now()
	set := bson.M{
		"enabled":   true,
		"reason":    reason,
		"grantedBy": currentOperator(r),
		"grantedAt": now,
		"expiresAt": now.AddDate(0, 0, days),
	}
	// `$set` only: the download history is never rewritten by opening a new
	// grant. It is the record of what already left, and it has to outlive the
	// permission that allowed it.
	if _, err := coll.UpdateOne(r.Context(), bson.M{"_id": exportGrantID},
		bson.M{"$set": set}, options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"enabled":   true,
		"reason":    reason,
		"expiresAt": set["expiresAt"],
		"grantedBy": set["grantedBy"],
	})
}
