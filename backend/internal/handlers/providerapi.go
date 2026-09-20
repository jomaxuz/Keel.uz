package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/delivery"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// clientFor builds the API client for a provider, or explains why it cannot.
//
// ⚠️ **Returns the interface, not a carrier.** There are two of them now, and
// the second one (BTS Express) has no published contract — see
// docs/vendor/bts-express.md. Everything above this line stays carrier-agnostic
// so that correcting BTS's wire format against their real document is one file.
//
// ⚠️ **An empty `apiProvider` is Yandex**, which is every provider record
// written before there was a second carrier. The usual zero-value rule, and
// here it is also the only safe reading: the alternative would take the API
// button away from every restaurant already using one.
func clientFor(p *models.DeliveryProvider) (delivery.Service, error) {
	if p.Kind != "api" {
		return nil, fmt.Errorf("%s API orqali ishlamaydi", p.Name)
	}
	if strings.TrimSpace(p.APIToken) == "" {
		return nil, fmt.Errorf("%s uchun API token kiritilmagan", p.Name)
	}
	switch p.APIProvider {
	case delivery.ProviderYandex, "":
		return delivery.NewYandex(p.APIBaseURL, p.APIToken, p.APITariff), nil
	case delivery.ProviderBTS:
		// ⚠️ **Refused here rather than at the request**, because the address
		// is the one thing about BTS nobody can guess and its absence is
		// otherwise completely silent: the form looks filled in, the save
		// succeeds, and the first parcel goes nowhere. No production host is
		// compiled in — BTS publishes none.
		if strings.TrimSpace(p.APIBaseURL) == "" {
			return nil, fmt.Errorf("%s uchun API manzili (base URL) kiritilmagan — BTS shartnomasidan olinadi", p.Name)
		}
		return delivery.NewBTS(p.APIBaseURL, p.APIToken, p.APITariff), nil
	default:
		return nil, fmt.Errorf("noma'lum integratsiya: %s", p.APIProvider)
	}
}

// orderContents renders the items as one readable line for the courier.
func orderContents(o *models.Order) string {
	parts := make([]string, 0, len(o.Items))
	for _, it := range o.Items {
		parts = append(parts, fmt.Sprintf("%s × %d", it.Name, it.Qty))
	}
	return strings.Join(parts, ", ")
}

func (h *Handler) loadOrderAndProvider(r *http.Request, orderID primitive.ObjectID) (models.Order, models.DeliveryProvider, error) {
	var order models.Order
	if err := h.Store.Orders.FindOne(r.Context(), bson.M{"_id": orderID}).Decode(&order); err != nil {
		return order, models.DeliveryProvider{}, fmt.Errorf("buyurtma topilmadi")
	}
	var p models.DeliveryProvider
	if order.ExternalDelivery == nil || order.ExternalDelivery.ProviderID.IsZero() {
		return order, p, fmt.Errorf("bu buyurtmaga xizmat chaqirilmagan")
	}
	if err := h.Store.Providers.FindOne(r.Context(),
		bson.M{"_id": order.ExternalDelivery.ProviderID}).Decode(&p); err != nil {
		return order, p, fmt.Errorf("xizmat topilmadi")
	}
	return order, p, nil
}

func (h *Handler) saveExternal(r *http.Request, orderID primitive.ObjectID, ext models.ExternalDelivery) error {
	_, err := h.Store.Orders.UpdateByID(r.Context(), orderID, bson.M{
		"$set": bson.M{"externalDelivery": ext, "updatedAt": time.Now()},
	})
	return err
}

// AdminCallProviderAPI files the delivery request with the provider itself:
// creates the claim, then confirms it so a courier is actually dispatched.
//
// The claim is created with a request id derived from the order, so a retry
// after a network timeout re-uses the same claim instead of ordering a second
// courier.
func (h *Handler) AdminCallProviderAPI(w http.ResponseWriter, r *http.Request) {
	orderID, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		ProviderID string `json:"providerId" validate:"required"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	providerID, err := objectID(req.ProviderID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	var order models.Order
	if err := h.Store.Orders.FindOne(r.Context(), bson.M{"_id": orderID}).Decode(&order); err != nil {
		httpx.Error(w, http.StatusNotFound, "buyurtma topilmadi")
		return
	}
	if order.Type != "delivery" || order.Address.Lat == 0 {
		httpx.Error(w, http.StatusBadRequest, "buyurtmada yetkazish manzili yo'q")
		return
	}
	var p models.DeliveryProvider
	if err := h.Store.Providers.FindOne(r.Context(), bson.M{"_id": providerID}).Decode(&p); err != nil {
		httpx.Error(w, http.StatusNotFound, "xizmat topilmadi")
		return
	}
	client, err := clientFor(&p)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var rest models.Restaurant
	if err := h.Store.Restaurant.FindOne(r.Context(), bson.M{}).Decode(&rest); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "restoran profili yo'q")
		return
	}
	restPhone := ""
	if len(rest.Phones) > 0 {
		restPhone = rest.Phones[0]
	}

	claim, err := client.Create(r.Context(), delivery.Order{
		Number: order.Number,
		Pickup: delivery.Point{
			Address: rest.Address.Text,
			Lat:     rest.Address.Lat,
			Lng:     rest.Address.Lng,
			Name:    rest.Name,
			Phone:   restPhone,
		},
		Destination: delivery.Point{
			Address: order.Address.Text,
			Lat:     order.Address.Lat,
			Lng:     order.Address.Lng,
			Name:    order.Customer.Name,
			Phone:   "+" + strings.TrimPrefix(order.Customer.Phone, "+"),
			Comment: order.Address.Comment,
		},
		Contents: orderContents(&order),
		Price:    fmt.Sprintf("%d", order.Subtotal),
	}, "order-"+order.ID.Hex())
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}

	// A created claim is only a draft — accepting it dispatches the courier.
	accepted, err := client.Accept(r.Context(), claim.ID, claim.Version)
	if err != nil {
		// The claim exists; store it so the operator can retry or cancel
		// instead of silently ordering another one.
		ext := models.ExternalDelivery{
			ProviderID: p.ID, ProviderName: p.Name, ClaimID: claim.ID,
			Status: claim.Status, Price: claim.Price,
			CalledAt: time.Now(), SyncedAt: time.Now(),
			Note: "tasdiqlanmadi: " + err.Error(),
		}
		_ = h.saveExternal(r, orderID, ext)
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	if accepted.Status != "" {
		claim = accepted
	}

	ext := models.ExternalDelivery{
		ProviderID:   p.ID,
		ProviderName: p.Name,
		ClaimID:      claim.ID,
		TrackingID:   claim.ID,
		Status:       claim.Status,
		Price:        claim.Price,
		CourierName:  claim.CourierName,
		CourierPhone: claim.CourierPhone,
		TrackURL:     claim.TrackURL,
		CalledAt:     time.Now(),
		SyncedAt:     time.Now(),
	}
	if err := h.saveExternal(r, orderID, ext); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Our own courier, if any, is released — one order, one carrier.
	if !order.CourierID.IsZero() {
		_, _ = h.Store.Orders.UpdateByID(r.Context(), orderID,
			bson.M{"$unset": bson.M{"courierId": "", "courierName": ""}})
		h.syncCourierBusy(r, order.CourierID)
	}
	h.logAction(r, ActOrderExternal, "order", orderID.Hex(), "#"+order.Number,
		p.Name+" (API) · "+claim.ID)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "externalDelivery": ext})
}

// AdminSyncProviderAPI refreshes the claim: status, price and the courier once
// the provider has assigned one.
func (h *Handler) AdminSyncProviderAPI(w http.ResponseWriter, r *http.Request) {
	orderID, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	order, p, err := h.loadOrderAndProvider(r, orderID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if order.ExternalDelivery.ClaimID == "" {
		httpx.Error(w, http.StatusBadRequest, "bu buyurtma API orqali chaqirilmagan")
		return
	}
	client, err := clientFor(&p)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	claim, err := client.Info(r.Context(), order.ExternalDelivery.ClaimID)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	ext := *order.ExternalDelivery
	ext.Status = claim.Status
	if claim.Price != "" {
		ext.Price = claim.Price
	}
	ext.CourierName = claim.CourierName
	ext.CourierPhone = claim.CourierPhone
	if claim.TrackURL != "" {
		ext.TrackURL = claim.TrackURL
	}
	ext.SyncedAt = time.Now()
	if err := h.saveExternal(r, orderID, ext); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "externalDelivery": ext})
}

// AdminCancelProviderAPI cancels the claim with the provider and clears it from
// the order. Cancelling late may be charged — that is the provider's rule, so
// the caller states which case it is.
func (h *Handler) AdminCancelProviderAPI(w http.ResponseWriter, r *http.Request) {
	orderID, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Paid bool `json:"paid"`
	}
	_ = httpx.Decode(r, &req)

	order, p, err := h.loadOrderAndProvider(r, orderID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	client, err := clientFor(&p)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	claim, err := client.Info(r.Context(), order.ExternalDelivery.ClaimID)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	if err := client.Cancel(r.Context(), order.ExternalDelivery.ClaimID, claim.Version, req.Paid); err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	if _, err := h.Store.Orders.UpdateByID(r.Context(), orderID, bson.M{
		"$unset": bson.M{"externalDelivery": ""},
		"$set":   bson.M{"updatedAt": time.Now()},
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActOrderExternal, "order", orderID.Hex(), "#"+order.Number,
		p.Name+" (API) zayavkasi bekor qilindi")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
