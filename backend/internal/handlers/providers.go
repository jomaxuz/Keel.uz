package handlers

import (
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ---- Provider CRUD (admin) ----

type providerPayload struct {
	Name      string `json:"name" validate:"required"`
	Kind      string `json:"kind"`
	URL       string `json:"url"`
	Phone     string `json:"phone"`
	Note      string `json:"note"`
	IsActive  *bool  `json:"isActive"`
	SortOrder int    `json:"sortOrder"`

	APIProvider string `json:"apiProvider"`
	// Empty on update = keep the stored token (the form never receives it).
	APIToken   string `json:"apiToken"`
	APIBaseURL string `json:"apiBaseUrl"`
	APITariff  string `json:"apiTariff"`
}

func normaliseKind(kind string) string {
	switch kind {
	case "phone", "api":
		return kind
	default:
		return "link"
	}
}

// withTokenFlag hides the secret but tells the panel whether one is stored.
func withTokenFlag(list []models.DeliveryProvider) []models.DeliveryProvider {
	for i := range list {
		list[i].HasToken = strings.TrimSpace(list[i].APIToken) != ""
		list[i].APIToken = ""
	}
	return list
}

// AdminListProviders returns the outside delivery services, ordered for the
// dispatcher's dropdown.
func (h *Handler) AdminListProviders(w http.ResponseWriter, r *http.Request) {
	opts := options.Find().SetSort(bson.D{
		{Key: "sortOrder", Value: 1}, {Key: "name", Value: 1},
	})
	cur, err := h.Store.Providers.Find(r.Context(), bson.M{}, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	providers := []models.DeliveryProvider{}
	_ = cur.All(r.Context(), &providers)
	httpx.JSON(w, http.StatusOK, withTokenFlag(providers))
}

func (h *Handler) AdminCreateProvider(w http.ResponseWriter, r *http.Request) {
	var req providerPayload
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	p := models.DeliveryProvider{
		Name:        strings.TrimSpace(req.Name),
		Kind:        normaliseKind(req.Kind),
		URL:         strings.TrimSpace(req.URL),
		Phone:       strings.TrimSpace(req.Phone),
		Note:        strings.TrimSpace(req.Note),
		IsActive:    req.IsActive == nil || *req.IsActive,
		SortOrder:   req.SortOrder,
		APIProvider: strings.TrimSpace(req.APIProvider),
		APIToken:    strings.TrimSpace(req.APIToken),
		APIBaseURL:  strings.TrimSpace(req.APIBaseURL),
		APITariff:   strings.TrimSpace(req.APITariff),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	res, err := h.Store.Providers.InsertOne(r.Context(), p)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	p.ID = oidOf(res.InsertedID)
	p.HasToken = p.APIToken != ""
	p.APIToken = ""
	h.logAction(r, ActProviderCreate, "provider", p.ID.Hex(), p.Name, p.Kind)
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *Handler) AdminUpdateProvider(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req providerPayload
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := bson.M{
		"name":        strings.TrimSpace(req.Name),
		"kind":        normaliseKind(req.Kind),
		"url":         strings.TrimSpace(req.URL),
		"phone":       strings.TrimSpace(req.Phone),
		"note":        strings.TrimSpace(req.Note),
		"sortOrder":   req.SortOrder,
		"apiProvider": strings.TrimSpace(req.APIProvider),
		"apiBaseUrl":  strings.TrimSpace(req.APIBaseURL),
		"apiTariff":   strings.TrimSpace(req.APITariff),
		"updatedAt":   time.Now(),
	}
	// An empty token means "leave the stored one alone" — the form never sees
	// the secret, so it cannot send it back.
	if strings.TrimSpace(req.APIToken) != "" {
		set["apiToken"] = strings.TrimSpace(req.APIToken)
	}
	if req.IsActive != nil {
		set["isActive"] = *req.IsActive
	}
	if _, err := h.Store.Providers.UpdateByID(r.Context(), id, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var p models.DeliveryProvider
	_ = h.Store.Providers.FindOne(r.Context(), bson.M{"_id": id}).Decode(&p)
	p.HasToken = p.APIToken != ""
	p.APIToken = ""
	h.logAction(r, ActProviderUpdate, "provider", id.Hex(), p.Name, p.Kind)
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handler) AdminDeleteProvider(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var removed models.DeliveryProvider
	_ = h.Store.Providers.FindOne(r.Context(), bson.M{"_id": id}).Decode(&removed)
	if _, err := h.Store.Providers.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActProviderDelete, "provider", id.Hex(), removed.Name, removed.Kind)
	// Past orders keep `externalDelivery.providerName`, so their history still
	// reads correctly after the service is removed.
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Handing an order to a provider ----

type callProviderRequest struct {
	ProviderID string `json:"providerId"`
	TrackingID string `json:"trackingId"`
	Note       string `json:"note"`
	// What the service charged (optional, link/phone providers only).
	Cost int `json:"cost"`
}

// AdminCallProvider records that an outside service was asked to carry this
// order. An empty providerId clears the record (called by mistake / cancelled).
//
// The actual "call" happens in the browser — the panel opens the provider's
// link with the order details filled in, or dials their number. We only store
// who was called and when, so the order history answers "who is bringing it?".
func (h *Handler) AdminCallProvider(w http.ResponseWriter, r *http.Request) {
	orderID, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req callProviderRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(req.ProviderID) == "" {
		if _, err := h.Store.Orders.UpdateByID(r.Context(), orderID, bson.M{
			"$unset": bson.M{"externalDelivery": ""},
			"$set":   bson.M{"updatedAt": time.Now()},
		}); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		h.logAction(r, ActOrderExternal, "order", orderID.Hex(), "",
			"tashqi xizmat bekor qilindi")
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	providerID, err := objectID(req.ProviderID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid provider id")
		return
	}
	var p models.DeliveryProvider
	if err := h.Store.Providers.FindOne(r.Context(), bson.M{"_id": providerID}).Decode(&p); err != nil {
		httpx.Error(w, http.StatusNotFound, "xizmat topilmadi")
		return
	}

	ext := models.ExternalDelivery{
		ProviderID:   p.ID,
		ProviderName: p.Name,
		TrackingID:   strings.TrimSpace(req.TrackingID),
		Note:         strings.TrimSpace(req.Note),
		CalledAt:     time.Now(),
		Cost:         req.Cost,
	}
	var previous models.Order
	_ = h.Store.Orders.FindOne(r.Context(), bson.M{"_id": orderID}).Decode(&previous)
	// Editing the record afterwards (adding the reference number or the price
	// once the service quotes it) must not move the "called at" time — that
	// timestamp is what answers "when did we hand it over?".
	if prev := previous.ExternalDelivery; prev != nil && prev.ProviderID == p.ID &&
		!prev.CalledAt.IsZero() {
		ext.CalledAt = prev.CalledAt
	}
	// Handing the order outside releases our own courier, if one was assigned.
	update := bson.M{
		"$set":   bson.M{"externalDelivery": ext, "updatedAt": time.Now()},
		"$unset": bson.M{"courierId": "", "courierName": ""},
	}
	if _, err := h.Store.Orders.UpdateByID(r.Context(), orderID, update); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !previous.CourierID.IsZero() {
		h.syncCourierBusy(r, previous.CourierID)
	}
	h.logAction(r, ActOrderExternal, "order", orderID.Hex(), "#"+previous.Number, p.Name)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "externalDelivery": ext})
}
