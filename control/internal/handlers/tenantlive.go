package handlers

import (
	"net/http"

	"keel-control/internal/httpx"
	"keel-control/internal/models"
	"keel-control/internal/tenantstats"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TenantLive is one customer's card, read from their own database.
//
// **A separate endpoint from GetTenant on purpose.** The tenant record loads
// instantly out of the control plane's own collection; this dials the
// customer's database and can be slow, or hang, or fail entirely — a stopped
// container, a tenant mid-migration, a Mongo under load. Folded into GetTenant,
// any of those would take down the page that shows the container status, which
// is precisely the page somebody opens *because* something is wrong.
//
// Split, the card renders immediately with everything the control plane knows,
// and the live numbers arrive after — or do not, with a sentence saying why.
//
// It also means the expensive half is only paid when a human is looking at one
// customer. The overview never calls this; that is what the nightly aggregate
// is for, and the two must not be confused: a list that dialled every database
// gets slower with every customer sold.
func (h *Handler) TenantLive(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	// ⚠️ **An agent never sees a customer's business figures.** They need to know who
	// they signed up and what they promised; a restaurant's turnover is that
	// restaurant's business, and handing it to a salesperson is a leak with a
	// friendly name. Refused as "no permission" rather than returned empty, so the
	// console can hide the panel instead of drawing zeros.
	actor, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	if !actor.Can(models.CanSeeStats) {
		fail(w, errForbidden)
		return
	}
	var t models.Tenant
	if err := h.Store.Tenants.FindOne(r.Context(), bson.M{"_id": id}).Decode(&t); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}

	snap := tenantstats.Collect(r.Context(), h.Store.TenantDB(t.DBName()))

	// A stopped tenant is not a broken one, and the difference matters on this
	// screen: zeroes from a suspended customer are correct, zeroes from a live
	// one mean something is wrong. Said here rather than left for the reader
	// to infer from a status badge somewhere else on the page.
	if t.Offline() {
		snap.Error = "mijoz to'xtatilgan — raqamlar oxirgi holatni ko'rsatadi"
	}
	httpx.JSON(w, http.StatusOK, snap)
}
