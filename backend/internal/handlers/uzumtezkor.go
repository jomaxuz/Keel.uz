package handlers

// ---- Uzum Tezkor: the marketplace's orders, straight onto the pass ----
//
// Uzum Tezkor calls **us** (docs/vendor/uzum-tezkor-retail.md): it takes a token
// from `/security/oauth/token`, POSTs each order to `/order`, and asks for its
// status once a minute. Uzum's courier always carries the food, so an order from
// it is cooked, marked ready and handed over — nothing else.
//
// ⚠️ **Their wire, not ours.** These endpoints answer in the shapes Uzum's
// validator expects: errors are a *list* of `{code, description}`, a refused
// token is `{reason}`, and nothing goes through `httpx.Error`. That also keeps
// the descriptions out of the translation catalogue on purpose — they are read
// by Uzum's support engineers while untangling a failed order, not by anybody
// in the restaurant, and "translated" would mean a Russian sentence one day and
// an Uzbek one the next depending on a header nobody sets.
//
// ⚠️ **15 minutes.** An order not ACCEPTED_BY_RESTAURANT within fifteen minutes
// is cancelled by Uzum. It rings like any new order (queuedAt is set), and the
// queue watch nudges the owner at ten.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

const (
	// tezkorRole is the token role. ⚠️ Its subject is the client id, never an
	// ObjectID, so a Uzum token handed to any other route names no user, no
	// admin and no courier.
	tezkorRole = "uzum_tezkor"
	// An hour: Uzum asks for a token before its requests anyway, and a short
	// life bounds a leaked one without costing anything.
	tezkorTokenTTL = time.Hour
	tezkorMaxBody  = 1 << 20
	// Uzum's time format: RFC 3339 with fractional seconds (`Y-m-d\TH:i:s.uP`).
	tezkorTimeLayout = "2006-01-02T15:04:05.000000Z07:00"
	// Where the routes are mounted, for the panel to show the owner.
	tezkorBasePath = "/api/v1/uzum-tezkor"
)

// Error codes in Uzum's error list. Ours to choose; kept few and stable so a
// support ticket can quote one.
const (
	tezkorCodeInvalid  = 100
	tezkorCodeAuth     = 101
	tezkorCodeNotFound = 102
	tezkorCodeState    = 103
	tezkorCodeInternal = 500
)

type tezkorErr struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
}

func tezkorFail(w http.ResponseWriter, status int, errs ...tezkorErr) {
	httpx.JSON(w, status, errs)
}

// tezkorDenied is the 401 body the spec names (`AuthorizationRequiredResponse`) —
// an object, unlike every other error.
func tezkorDenied(w http.ResponseWriter, reason string) {
	httpx.JSON(w, http.StatusUnauthorized, map[string]string{"reason": reason})
}

// tezkorNoCache sets the headers the spec lists on every answer. An order status
// served from a cache is a courier sent for food that is not ready.
func tezkorNoCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, max-age=0, no-cache, no-store")
	w.Header().Set("Pragma", "no-cache")
}

func (h *Handler) tezkorSettings(ctx context.Context) *models.UzumTezkorSettings {
	var s models.UzumTezkorSettings
	if err := h.Store.UzumTezkorSettings.FindOne(ctx, bson.M{}).Decode(&s); err != nil {
		return &models.UzumTezkorSettings{}
	}
	return &s
}

func tezkorSecretHash(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// tezkorSecretMatches compares in constant time, and never matches an unset
// secret — an empty hash compared with the hash of an empty string is exactly
// the comparison that must not succeed.
func tezkorSecretMatches(s *models.UzumTezkorSettings, given string) bool {
	if s.SecretHash == "" || given == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(tezkorSecretHash(given)), []byte(s.SecretHash)) == 1
}

// ---- Token ----

// TezkorToken is OAuth2 client credentials, form-encoded, as the spec defines it.
func (h *Handler) TezkorToken(w http.ResponseWriter, r *http.Request) {
	tezkorNoCache(w)
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		tezkorFail(w, http.StatusBadRequest, tezkorErr{tezkorCodeInvalid,
			"body must be application/x-www-form-urlencoded"})
		return
	}
	s := h.tezkorSettings(r.Context())
	if !s.Enabled || s.ClientID == "" || s.SecretHash == "" {
		tezkorFail(w, http.StatusBadRequest, tezkorErr{tezkorCodeAuth,
			"Uzum Tezkor integration is not enabled for this restaurant"})
		return
	}
	if g := r.PostForm.Get("grant_type"); g != "client_credentials" {
		tezkorFail(w, http.StatusBadRequest, tezkorErr{tezkorCodeInvalid,
			"grant_type must be client_credentials"})
		return
	}
	id := r.PostForm.Get("client_id")
	if subtle.ConstantTimeCompare([]byte(id), []byte(s.ClientID)) != 1 ||
		!tezkorSecretMatches(s, r.PostForm.Get("client_secret")) {
		tezkorFail(w, http.StatusBadRequest, tezkorErr{tezkorCodeAuth,
			"invalid client_id or client_secret"})
		return
	}
	token, err := auth.GenerateLong(h.Cfg.JWTSecret, s.ClientID, tezkorRole, s.TokenVersion, tezkorTokenTTL)
	if err != nil {
		tezkorFail(w, http.StatusInternalServerError, tezkorErr{tezkorCodeInternal, "token could not be issued"})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "bearer",
		"expires_in":   int(tezkorTokenTTL.Seconds()),
		"scope":        "read write",
	})
}

// TezkorAuth admits a request carrying a live Uzum token.
//
// ⚠️ **The settings are read on every request**, like the console reads its
// accounts: switching the integration off or minting a new secret has to stop
// Uzum on its next call, not when an hour-old token happens to lapse.
func (h *Handler) TezkorAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tezkorNoCache(w)
		raw := strings.TrimSpace(r.Header.Get("Authorization"))
		if len(raw) < 7 || !strings.EqualFold(raw[:7], "Bearer ") {
			tezkorDenied(w, "Authorization: Bearer <token> is required")
			return
		}
		claims, err := auth.Parse(h.Cfg.JWTSecret, strings.TrimSpace(raw[7:]))
		if err != nil || claims.Role != tezkorRole {
			tezkorDenied(w, "Access token is invalid or has expired. You should request a new one")
			return
		}
		s := h.tezkorSettings(r.Context())
		if !s.Enabled || claims.Ver != s.TokenVersion || claims.UserID != s.ClientID {
			tezkorDenied(w, "Access token has been revoked. You should request a new one")
			return
		}
		next(w, r)
	}
}

// ---- The order, as Uzum sends it (`YGroceryOrderV2`) ----

type tezkorOrderIn struct {
	Discriminator string            `json:"discriminator"`
	EatsID        string            `json:"eatsId"`
	RestaurantID  string            `json:"restaurantId"`
	Comment       string            `json:"comment"`
	Persons       int               `json:"persons"`
	Items         []tezkorItemIn    `json:"items"`
	PaymentInfo   *tezkorPaymentIn  `json:"paymentInfo"`
	DeliveryInfo  *tezkorDeliveryIn `json:"deliveryInfo"`
}

type tezkorItemIn struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// ⚠️ **One unit with its modifications already added** — the spec says so
	// and says it will change "in the next version". Used as the line price as
	// it stands; modifiers are never added on top.
	Price float64 `json:"price"`
	// ⚠️ A float on the wire (`3.5` for a weighed product). A restaurant line
	// is whole portions, so a fraction is refused rather than rounded.
	Quantity      float64       `json:"quantity"`
	Modifications []tezkorModIn `json:"modifications"`
}

type tezkorModIn struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

type tezkorPaymentIn struct {
	ItemsCost   float64 `json:"itemsCost"`
	PaymentType string  `json:"paymentType"`
}

type tezkorDeliveryIn struct {
	ClientName            string `json:"clientName"`
	CourierArrivementDate string `json:"courierArrivementDate"`
	PhoneNumber           string `json:"phoneNumber"`
	ClientPhoneNumber     string `json:"clientPhoneNumber"`
}

// tezkorCheck is everything wrong with an order that can be said without the
// database. Every problem at once: Uzum's support reads the list, and one error
// per retry is a ticket that takes an afternoon.
func tezkorCheck(in *tezkorOrderIn) []tezkorErr {
	var errs []tezkorErr
	add := func(format string, a ...any) {
		errs = append(errs, tezkorErr{tezkorCodeInvalid, fmt.Sprintf(format, a...)})
	}
	if strings.TrimSpace(in.EatsID) == "" {
		add("eatsId is required")
	}
	if _, err := primitive.ObjectIDFromHex(strings.TrimSpace(in.RestaurantID)); err != nil {
		add("restaurantId %q is not a store id of this restaurant", in.RestaurantID)
	}
	if len(in.Items) == 0 {
		add("items must not be empty")
	}
	for i, it := range in.Items {
		if _, err := primitive.ObjectIDFromHex(strings.TrimSpace(it.ID)); err != nil {
			add("items[%d].id %q is not a product id of this restaurant", i, it.ID)
		}
		if it.Quantity <= 0 || it.Quantity != math.Trunc(it.Quantity) {
			add("items[%d].quantity %v must be a whole number above zero", i, it.Quantity)
		}
		if it.Price < 0 {
			add("items[%d].price must not be negative", i)
		}
	}
	return errs
}

// tezkorModsText is the modifications as the kitchen reads them, on the line's
// comment. ⚠️ **Not as options**: an option carries a price delta, and the line
// price already includes these — a delta would count them twice on the receipt.
func tezkorModsText(mods []tezkorModIn) string {
	parts := make([]string, 0, len(mods))
	for _, m := range mods {
		name := strings.TrimSpace(m.Name)
		if name == "" {
			name = m.ID
		}
		if m.Quantity > 1 {
			name = fmt.Sprintf("%s ×%d", name, m.Quantity)
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, ", ")
}

func (h *Handler) tezkorByEatsID(ctx context.Context, eatsID string) (*models.Order, bool) {
	var o models.Order
	err := h.Store.Orders.FindOne(ctx, bson.M{
		"aggregator.provider":   models.ProviderUzumTezkor,
		"aggregator.externalId": strings.TrimSpace(eatsID),
	}).Decode(&o)
	return &o, err == nil
}

// tezkorOrderByID finds an order Uzum placed. ⚠️ Only those: the order id is ours
// and guessable in shape, and this token must not read a website order.
func (h *Handler) tezkorOrderByID(r *http.Request) (*models.Order, bool) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "orderId"))
	if err != nil {
		return nil, false
	}
	var o models.Order
	err = h.Store.Orders.FindOne(r.Context(), bson.M{
		"_id": id, "aggregator.provider": models.ProviderUzumTezkor,
	}).Decode(&o)
	return &o, err == nil
}

func tezkorCreated(w http.ResponseWriter, o *models.Order) {
	httpx.JSON(w, http.StatusOK, map[string]any{"orderId": o.ID.Hex(), "result": "OK"})
}

// TezkorCreateOrder takes a new order from Uzum Tezkor.
func (h *Handler) TezkorCreateOrder(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, tezkorMaxBody))
	if err != nil {
		tezkorFail(w, http.StatusBadRequest, tezkorErr{tezkorCodeInvalid, "request body could not be read"})
		return
	}
	var in tezkorOrderIn
	if err := json.Unmarshal(body, &in); err != nil {
		tezkorFail(w, http.StatusBadRequest, tezkorErr{tezkorCodeInvalid, "body is not a valid order: " + err.Error()})
		return
	}
	if errs := tezkorCheck(&in); len(errs) > 0 {
		tezkorFail(w, http.StatusBadRequest, errs...)
		return
	}
	ctx := r.Context()

	// ⚠️ **A retry answers with the first order**: same 200, same orderId. See
	// the index in repository/migrate.go for the race this look-up cannot close.
	if prev, ok := h.tezkorByEatsID(ctx, in.EatsID); ok {
		tezkorCreated(w, prev)
		return
	}

	branchID, _ := primitive.ObjectIDFromHex(strings.TrimSpace(in.RestaurantID))
	branch, err := h.branchByIDCtx(ctx, branchID)
	if err != nil || branch == nil {
		tezkorFail(w, http.StatusNotFound, tezkorErr{tezkorCodeNotFound,
			"store " + in.RestaurantID + " was not found"})
		return
	}

	// ⚠️ **Priced as Uzum sold it, not re-priced from our menu.** The guest has
	// already paid Uzum that amount; a website order is re-priced because the
	// browser is not trusted, but here the price is the marketplace's contract,
	// and a different number on our receipt is a reconciliation nobody can do.
	// ⚠️ **An unavailable dish is still accepted**: Uzum has sold it, and
	// refusing starts retries that end in a cancellation the guest sees. The
	// kitchen sees the order and decides.
	items := make([]models.OrderItem, 0, len(in.Items))
	subtotal := 0
	var errs []tezkorErr
	for i, it := range in.Items {
		id, _ := primitive.ObjectIDFromHex(strings.TrimSpace(it.ID))
		var dish models.MenuItem
		if err := h.Store.Menu.FindOne(ctx, bson.M{"_id": id}).Decode(&dish); err != nil {
			errs = append(errs, tezkorErr{tezkorCodeInvalid,
				fmt.Sprintf("items[%d].id %s is not on this restaurant's menu", i, it.ID)})
			continue
		}
		name := dish.Name
		if name == "" {
			name = it.Name
		}
		line := models.OrderItem{
			MenuItemID: id,
			Name:       name,
			Price:      int(math.Round(it.Price)),
			Qty:        int(it.Quantity),
			// ⚠️ Never nil: an absent list reaches the panel as `null`.
			Options: []models.OrderItemOption{},
			Comment: tezkorModsText(it.Modifications),
		}
		subtotal += line.Price * line.Qty
		items = append(items, line)
	}
	if len(errs) > 0 {
		tezkorFail(w, http.StatusBadRequest, errs...)
		return
	}

	now := time.Now()
	agg := &models.OrderAggregator{
		Provider:   models.ProviderUzumTezkor,
		ExternalID: strings.TrimSpace(in.EatsID),
		Persons:    in.Persons,
		Comment:    strings.TrimSpace(in.Comment),
		Payload:    string(body),
	}
	customer := models.OrderCustomer{}
	if d := in.DeliveryInfo; d != nil {
		customer = models.OrderCustomer{
			Name:  strings.TrimSpace(d.ClientName),
			Phone: strings.TrimSpace(d.ClientPhoneNumber),
		}
		agg.CourierPhone = strings.TrimSpace(d.PhoneNumber)
		if t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(d.CourierArrivementDate)); err == nil {
			local := t.In(time.Local)
			agg.CourierAt = &local
		}
	}
	if p := in.PaymentInfo; p != nil {
		agg.PaymentType = p.PaymentType
		agg.ItemsCost = int(math.Round(p.ItemsCost))
	}

	order := models.Order{
		BrandID:  branch.BrandID,
		BranchID: branch.ID,
		Number:   branchOrderNumber(branch.Code),
		Status:   models.StatusPending,
		Customer: customer,
		Type:     models.OrderTypeUzumTezkor,
		Address:  models.OrderAddress{Comment: agg.Comment},
		Items:    items,
		Subtotal: subtotal,
		Total:    subtotal,
		// ⚠️ **Paid, to Uzum.** Nobody at the counter takes money for this, and
		// the marketplace settles it later — the payouts screen already counts
		// paid `uzum_tezkor` sales as money Uzum holds.
		PaymentMethod: models.ProviderUzumTezkor,
		PaymentStatus: models.PayPaid,
		PaidAt:        &now,
		// It is the kitchen's the moment it arrives: this is what rings.
		QueuedAt:      &now,
		StatusHistory: []models.StatusEvent{{Status: models.StatusPending, At: now}},
		Channel:       models.ProviderUzumTezkor,
		Aggregator:    agg,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	res, err := h.Store.Orders.InsertOne(ctx, order)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			if prev, ok := h.tezkorByEatsID(ctx, in.EatsID); ok {
				tezkorCreated(w, prev)
				return
			}
		}
		tezkorFail(w, http.StatusInternalServerError, tezkorErr{tezkorCodeInternal, "order could not be saved"})
		return
	}
	order.ID = res.InsertedID.(primitive.ObjectID)
	// The same after-sale steps as every other order path — see CreateOrder.
	h.syncOrderStock(ctx, &order)
	h.applyDailyLimits(ctx, order.BranchID)
	h.notifyAdmins(order.BranchID, false, adminText("Yangi buyurtma",
		fmt.Sprintf("#%s · %d so'm", order.Number, order.Total)),
		map[string]any{"type": "order", "orderId": order.ID.Hex()})
	tezkorCreated(w, &order)
}

// TezkorGetOrder answers with the order as it arrived. See OrderAggregator.Payload
// for why it is never rebuilt.
func (h *Handler) TezkorGetOrder(w http.ResponseWriter, r *http.Request) {
	o, ok := h.tezkorOrderByID(r)
	if !ok || o.Aggregator == nil || o.Aggregator.Payload == "" {
		tezkorFail(w, http.StatusNotFound, tezkorErr{tezkorCodeNotFound, "order not found"})
		return
	}
	w.Header().Set("Content-Type", "application/vnd.eats.order.v2+json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, o.Aggregator.Payload)
}

// tezkorStatusOf is our order in Uzum's vocabulary.
//
// ⚠️ **READY is our `readyAt`, not a status** — models.Order.ReadyAt explains
// why the kitchen's "done" is a timestamp. It only counts once the order has
// been accepted: a pending order is NEW whatever else is on it, because Uzum
// allows no step backwards and no skipping acceptance.
func tezkorStatusOf(o *models.Order) string {
	switch o.Status {
	case models.StatusCancelled:
		return "CANCELLED"
	case models.StatusDelivered:
		return "DELIVERED"
	case models.StatusOnTheWay:
		return "TAKEN_BY_COURIER"
	case models.StatusConfirmed, models.StatusPreparing:
		if o.ReadyAt != nil {
			return "READY"
		}
		if o.Status == models.StatusPreparing {
			return "COOKING"
		}
		return "ACCEPTED_BY_RESTAURANT"
	}
	return "NEW"
}

// tezkorUpdatedAt is when the status last moved, in Uzum's format.
func tezkorUpdatedAt(o *models.Order) string {
	at := o.UpdatedAt
	if n := len(o.StatusHistory); n > 0 {
		at = o.StatusHistory[n-1].At
	}
	if o.ReadyAt != nil && o.ReadyAt.After(at) {
		at = *o.ReadyAt
	}
	return at.In(time.Local).Format(tezkorTimeLayout)
}

// TezkorOrderStatus is what Uzum polls once a minute.
func (h *Handler) TezkorOrderStatus(w http.ResponseWriter, r *http.Request) {
	o, ok := h.tezkorOrderByID(r)
	if !ok {
		tezkorFail(w, http.StatusNotFound, tezkorErr{tezkorCodeNotFound, "order not found"})
		return
	}
	out := map[string]any{"status": tezkorStatusOf(o), "updatedAt": tezkorUpdatedAt(o)}
	if o.Status == models.StatusCancelled && o.CancelReason != "" {
		out["comment"] = clampText(o.CancelReason, 500)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// TezkorUpdateOrder is Uzum changing an order. ⚠️ Refused with the spec's 422:
// the spec calls it rare for shops and switched on by agreement, and accepting
// a new composition silently would change food the kitchen may already be
// cooking.
func (h *Handler) TezkorUpdateOrder(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.tezkorOrderByID(r); !ok {
		tezkorFail(w, http.StatusNotFound, tezkorErr{tezkorCodeNotFound, "order not found"})
		return
	}
	tezkorFail(w, http.StatusUnprocessableEntity, tezkorErr{tezkorCodeState,
		"order updates from Uzum Tezkor are not enabled for this restaurant"})
}

// TezkorCancelOrder is Uzum cancelling. The body carries `eatsId` and an optional
// comment, which becomes the reason the panel shows.
func (h *Handler) TezkorCancelOrder(w http.ResponseWriter, r *http.Request) {
	o, ok := h.tezkorOrderByID(r)
	if !ok {
		tezkorFail(w, http.StatusNotFound, tezkorErr{tezkorCodeNotFound, "order not found"})
		return
	}
	var in struct {
		EatsID  string `json:"eatsId"`
		Comment string `json:"comment"`
	}
	body, _ := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &in); err != nil {
			tezkorFail(w, http.StatusBadRequest, tezkorErr{tezkorCodeInvalid, "body is not valid JSON"})
			return
		}
	}
	if e := strings.TrimSpace(in.EatsID); e != "" && o.Aggregator != nil && e != o.Aggregator.ExternalID {
		tezkorFail(w, http.StatusBadRequest, tezkorErr{tezkorCodeInvalid, "eatsId does not match this order"})
		return
	}
	switch o.Status {
	case models.StatusCancelled:
		// Cancelling twice is a retry, not a mistake.
		httpx.JSON(w, http.StatusOK, map[string]any{"result": "OK"})
		return
	case models.StatusDelivered:
		tezkorFail(w, http.StatusBadRequest, tezkorErr{tezkorCodeState, "order is already delivered"})
		return
	}

	reason := "Uzum Tezkor"
	if c := strings.TrimSpace(in.Comment); c != "" {
		reason += ": " + c
	}
	now := time.Now()
	// ⚠️ Guarded on the status in the filter, so a cancel racing the kitchen's
	// "delivered" cannot undo it.
	res, err := h.Store.Orders.UpdateOne(r.Context(), bson.M{
		"_id": o.ID,
		"status": bson.M{"$nin": []models.OrderStatus{
			models.StatusCancelled, models.StatusDelivered,
		}},
	}, bson.M{
		"$set": bson.M{
			"status": models.StatusCancelled, "cancelReason": clampText(reason, 300), "updatedAt": now,
		},
		"$push": bson.M{"statusHistory": models.StatusEvent{Status: models.StatusCancelled, At: now}},
	})
	if err != nil {
		tezkorFail(w, http.StatusInternalServerError, tezkorErr{tezkorCodeInternal, "order could not be cancelled"})
		return
	}
	if res.ModifiedCount > 0 {
		var fresh models.Order
		if h.Store.Orders.FindOne(r.Context(), bson.M{"_id": o.ID}).Decode(&fresh) == nil {
			// The shelf follows the cancellation exactly as it does for an admin
			// cancel — see UpdateOrderStatus.
			h.syncOrderStock(r.Context(), &fresh)
			// ⚠️ **Told, not merely recorded**: the kitchen may be cooking it.
			h.notifyAdmins(fresh.BranchID, false, adminText("Uzum Tezkor",
				fmt.Sprintf("#%s buyurtma bekor qilindi: %s", fresh.Number, fresh.CancelReason)),
				map[string]any{"type": "order", "orderId": fresh.ID.Hex()})
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"result": "OK"})
}

// ---- The panel: credentials the owner hands to Uzum ----

// AdminUzumTezkor is the integration's state and the ids Uzum's manager needs.
func (h *Handler) AdminUzumTezkor(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	s := h.tezkorSettings(r.Context())
	// ⚠️ **Every branch, with its id**: the id is the `restaurantId` Uzum sends
	// each order to, and it is handed to their manager by hand.
	stores := []map[string]string{}
	if cur, err := h.Store.Branches.Find(r.Context(), bson.M{},
		options.Find().SetProjection(bson.M{"name": 1})); err == nil {
		var rows []struct {
			ID   primitive.ObjectID `bson:"_id"`
			Name string             `bson:"name"`
		}
		if cur.All(r.Context(), &rows) == nil {
			for _, b := range rows {
				stores = append(stores, map[string]string{"id": b.ID.Hex(), "name": b.Name})
			}
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"enabled":   s.Enabled,
		"clientId":  s.ClientID,
		"hasSecret": s.SecretHash != "",
		"rotatedAt": s.RotatedAt,
		"basePath":  tezkorBasePath,
		"stores":    stores,
	})
}

func tezkorRandom(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// AdminUzumTezkorCredentials mints a new secret (and a client id, the first
// time) and switches the integration on.
//
// ⚠️ **The secret is in this answer and nowhere else, ever.** Minting a new one
// ends every token issued under the old one (TokenVersion), so it is also the
// answer to "the secret was sent to the wrong chat".
func (h *Handler) AdminUzumTezkorCredentials(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	s := h.tezkorSettings(r.Context())
	secret, err := tezkorRandom(32)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	clientID := s.ClientID
	if clientID == "" {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		clientID = "keel-" + hex.EncodeToString(b)
	}
	now := time.Now()
	if _, err := h.Store.UzumTezkorSettings.UpdateOne(r.Context(), bson.M{}, bson.M{
		"$set": bson.M{
			"clientId": clientID, "secretHash": tezkorSecretHash(secret),
			"enabled": true, "rotatedAt": now, "updatedAt": now,
		},
		"$inc": bson.M{"tokenVersion": 1},
	}, options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"enabled": true, "clientId": clientID, "clientSecret": secret, "rotatedAt": now,
	})
}

// AdminUzumTezkorEnable switches the integration on or off without touching the
// credentials. Off takes effect on Uzum's next request (see TezkorAuth).
func (h *Handler) AdminUzumTezkorEnable(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := h.Store.UzumTezkorSettings.UpdateOne(r.Context(), bson.M{}, bson.M{
		"$set": bson.M{"enabled": req.Enabled, "updatedAt": time.Now()},
	}, options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"enabled": req.Enabled})
}
