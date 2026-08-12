package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"regexp"
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

// Online payment, the parts all three providers share.
//
// Payme, Click and Uzum differ in their wire formats and in almost nothing
// else. Each one sends the guest to its own checkout page, takes the money, and
// then calls this server to say so. Everything that actually protects the
// restaurant lives here rather than in the three adapters, because a rule
// implemented three times is a rule implemented twice:
//
//   - **the amount is the order's, never the callback's.** A callback saying
//     "paid 1 000 so'm" for a 100 000 so'm order is refused, not recorded.
//   - **marking an order paid is idempotent.** All three providers retry, and
//     Payme will happily call PerformTransaction again after a timeout it
//     decided was its own fault. Paying twice must be impossible, and so must
//     awarding the cashback twice.
//   - **the browser proves nothing.** The guest returning from the bank with a
//     success-looking URL changes no state at all; only the server-to-server
//     call does. This is not paranoia — the return trip also simply does not
//     happen when somebody pays and closes the tab.

// paymentSettings loads the provider credentials. A missing document is not an
// error: an install that has never opened the settings page simply takes cash.
func (h *Handler) paymentSettings(ctx context.Context) *models.PaymentSettings {
	var s models.PaymentSettings
	if err := h.Store.PaymentSettings.FindOne(ctx, bson.M{}).Decode(&s); err != nil {
		return &models.PaymentSettings{}
	}
	return &s
}

// orderByNumber finds the order a payment refers to.
//
// Providers address an order by the number printed on the receipt, not by its
// database id: the number is what a human types into the bank's form when the
// automatic flow fails, and what support asks for on the phone.
func (h *Handler) orderByNumber(ctx context.Context, number string) (*models.Order, error) {
	number = strings.TrimSpace(strings.TrimPrefix(number, "#"))
	if number == "" {
		return nil, errors.New("order number is empty")
	}
	var order models.Order
	// Case-insensitive exact match: branch prefixes are upper-case and guests
	// type them however their keyboard feels.
	filter := bson.M{"number": bson.M{"$regex": "^" + regexp.QuoteMeta(number) + "$", "$options": "i"}}
	if err := h.Store.Orders.FindOne(ctx, filter).Decode(&order); err != nil {
		return nil, errors.New("order not found")
	}
	return &order, nil
}

// payable reports why an order cannot be paid, or nil when it can.
//
// A cancelled order is the case that matters: without this check a guest who
// cancelled and then finished a half-open bank page would hand over money for
// food nobody is cooking.
func payable(order *models.Order) error {
	if order.Status == models.StatusCancelled {
		return errors.New("order is cancelled")
	}
	if order.PaymentStatus == models.PayPaid {
		return errors.New("order is already paid")
	}
	return nil
}

// findTxn loads a provider transaction by the provider's own id.
func (h *Handler) findTxn(ctx context.Context, provider, txnID string) (*models.Payment, error) {
	var p models.Payment
	err := h.Store.Payments.FindOne(ctx, bson.M{
		"provider": provider, "providerTxnId": txnID,
	}).Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// createTxn records that a provider has started taking money for an order.
//
// The unique (provider, providerTxnId) index does the real work: a retried
// "create" callback hits the duplicate-key error and the existing row is
// returned, so the provider sees the same answer it saw the first time.
func (h *Handler) createTxn(
	ctx context.Context, provider, txnID string, order *models.Order, createMs int64,
) (*models.Payment, error) {
	now := time.Now()
	if createMs == 0 {
		createMs = now.UnixMilli()
	}
	p := models.Payment{
		OrderID:       order.ID,
		OrderNumber:   order.Number,
		BranchID:      order.BranchID,
		Provider:      provider,
		ProviderTxnID: txnID,
		Amount:        order.Total,
		State:         models.TxnCreated,
		CreateTimeMs:  createMs,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	res, err := h.Store.Payments.InsertOne(ctx, p)
	if mongo.IsDuplicateKeyError(err) {
		return h.findTxn(ctx, provider, txnID)
	}
	if err != nil {
		return nil, err
	}
	p.ID = res.InsertedID.(primitive.ObjectID)
	// The order is now waiting on a bank rather than on the kitchen.
	_, _ = h.Store.Orders.UpdateByID(ctx, order.ID, bson.M{"$set": bson.M{
		"paymentStatus": models.PayPending, "updatedAt": now,
	}})
	return &p, nil
}

// performTxn marks a transaction — and its order — paid.
//
// Idempotent by the transaction's own state, not by a flag on the order: two
// providers could in principle both have a live transaction against one order,
// and it is the transaction that either did or did not go through.
func (h *Handler) performTxn(ctx context.Context, txn *models.Payment) error {
	if txn.State == models.TxnPerformed {
		return nil // already done; say so calmly, the provider is retrying
	}
	if txn.State != models.TxnCreated {
		return errors.New("transaction is not in a payable state")
	}
	now := time.Now()
	ms := now.UnixMilli()
	// Guarded by the state in the filter: two concurrent callbacks race here,
	// and exactly one of them matches.
	res, err := h.Store.Payments.UpdateOne(ctx,
		bson.M{"_id": txn.ID, "state": models.TxnCreated},
		bson.M{"$set": bson.M{
			"state": models.TxnPerformed, "performTimeMs": ms, "updatedAt": now,
		}})
	if err != nil {
		return err
	}
	txn.State, txn.PerformTimeMs = models.TxnPerformed, ms
	if res.ModifiedCount == 0 {
		return nil // the other caller won; the money is banked either way
	}

	_, err = h.Store.Orders.UpdateOne(ctx,
		bson.M{"_id": txn.OrderID, "paymentStatus": bson.M{"$ne": models.PayPaid}},
		bson.M{"$set": bson.M{
			"paymentStatus": models.PayPaid, "paidAt": now, "updatedAt": now,
		}})
	return err
}

// cancelTxn reverses a transaction. `state` is the resulting ledger state,
// which differs by whether the money had already been taken.
func (h *Handler) cancelTxn(ctx context.Context, txn *models.Payment, reason int) error {
	if txn.State == models.TxnCancelledSetup || txn.State == models.TxnCancelledPaid {
		return nil
	}
	state := models.TxnCancelledSetup
	if txn.State == models.TxnPerformed {
		state = models.TxnCancelledPaid
	}
	now := time.Now()
	ms := now.UnixMilli()
	if _, err := h.Store.Payments.UpdateByID(ctx, txn.ID, bson.M{"$set": bson.M{
		"state": state, "reason": reason, "cancelTimeMs": ms, "updatedAt": now,
	}}); err != nil {
		return err
	}
	wasPaid := txn.State == models.TxnPerformed
	txn.State, txn.CancelTimeMs, txn.Reason = state, ms, reason

	// Money that was taken and given back leaves the order refunded; money that
	// was never taken leaves it simply unpaid, free to be tried again.
	status := models.PayUnpaid
	if wasPaid {
		status = models.PayRefunded
	}
	_, err := h.Store.Orders.UpdateByID(ctx, txn.OrderID, bson.M{"$set": bson.M{
		"paymentStatus": status, "updatedAt": now,
	}})
	return err
}

// afterPaid is what happens once the money is actually in: the order joins the
// kitchen queue, and only now.
//
// This is the point of separating QueuedAt from CreatedAt. An online order sits
// created-but-unpaid for as long as the guest fiddles with their card, and
// during that time nobody should be cooking it or hearing a chime for it. The
// moment the provider confirms, it becomes a normal new order — and the chime
// fires, because the panel watches this timestamp.
func (h *Handler) afterPaid(r *http.Request, txn *models.Payment) {
	now := time.Now()
	// ⚠️ A pre-order paid for by card joins the queue at **its** time, not at
	// the moment the bank answered. Without this the two halves disagree: cash
	// pre-orders would wait for their lead and card ones would land on the pass
	// the instant they were paid — days early, in the worst case — and the bug
	// would look like the kitchen screen having a mind of its own.
	queued := now
	if at, ok := h.preorderQueueOf(r, txn.OrderID, now); ok {
		queued = at
	}
	_, _ = h.Store.Orders.UpdateOne(r.Context(),
		bson.M{"_id": txn.OrderID, "queuedAt": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"queuedAt": queued}})
	log.Printf("payment: order %s paid via %s (%s)",
		txn.OrderNumber, txn.Provider, txn.ProviderTxnID)
}

// ---- The link the guest is sent to ----

// payURL builds the provider's checkout URL for an order.
//
// Returns an empty string for cash and for a provider that is not fully set up:
// the checkout then simply shows no "pay now" button, which is the honest
// answer. Half-configured credentials must fail here, in the restaurant's own
// panel, rather than on the bank's error page in front of a guest.
func (h *Handler) payURL(ctx context.Context, order *models.Order, returnTo string) string {
	s := h.paymentSettings(ctx)
	if !s.Configured(order.PaymentMethod) {
		return ""
	}
	if returnTo == "" {
		returnTo = strings.TrimRight(s.ReturnURL, "/")
	}
	switch order.PaymentMethod {
	case models.ProviderPayme:
		return paymeCheckoutURL(s, order, returnTo)
	case models.ProviderClick:
		return clickCheckoutURL(s, order, returnTo)
	case models.ProviderUzum:
		return uzumCheckoutURL(s, order, returnTo)
	case models.ProviderAtmos:
		// ⚠️ The only provider whose link is **fetched, not built**: ATMOS
		// mints the invoice and hands back a checkout.atmos.uz address.
		//
		// So this one can fail for reasons the others cannot — the gateway
		// being down, a rejected credential, a closed contract. An empty
		// string is returned in that case, which the checkout already handles
		// as "no pay button", and the reason is logged rather than shown: the
		// guest can do nothing with "atmos invoice: STORE_NOT_FOUND", and the
		// restaurant finds it in the log with the order number beside it.
		url, err := h.atmosCreateInvoice(ctx, &s.Atmos, order, returnTo)
		if err != nil {
			log.Printf("atmos: no checkout link for order %s: %v", order.Number, err)
			return ""
		}
		return url
	}
	return ""
}

// OrderPayLink hands the checkout URL back to the site.
//
// Public and keyed by the order number alone, deliberately: the guest who has
// just ordered may close the tab, come back on another device, and still need
// to pay. The number is on their receipt and the link leads to a bank page
// asking for a card — knowing it gives nobody anything but the chance to pay
// somebody else's bill.
func (h *Handler) OrderPayLink(w http.ResponseWriter, r *http.Request) {
	order, err := h.orderByNumber(r.Context(), chi.URLParam(r, "number"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "order not found")
		return
	}
	if err := payable(order); err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"url": "", "status": order.PaymentStatus, "reason": err.Error(),
		})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"url":      h.payURL(r.Context(), order, strings.TrimSpace(r.URL.Query().Get("return"))),
		"provider": order.PaymentMethod,
		"status":   order.PaymentStatus,
	})
}

// ---- Admin: the credentials ----

// AdminGetPaymentSettings returns the settings **without the secrets** — only
// whether each one is stored. The panel shows "key saved" rather than the key:
// a page that renders a merchant key puts it in every screenshot and every
// browser cache from then on.
func (h *Handler) AdminGetPaymentSettings(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	s := h.paymentSettings(r.Context())
	httpx.JSON(w, http.StatusOK, map[string]any{
		"returnUrl": s.ReturnURL,
		"payme": map[string]any{
			"enabled":      s.Payme.Enabled,
			"merchantId":   s.Payme.MerchantID,
			"testMode":     s.Payme.TestMode,
			"accountField": paymeAccountField(s),
			"hasKey":       s.Payme.Key != "",
			"hasTestKey":   s.Payme.TestKey != "",
		},
		"click": map[string]any{
			"enabled":        s.Click.Enabled,
			"serviceId":      s.Click.ServiceID,
			"merchantId":     s.Click.MerchantID,
			"merchantUserId": s.Click.MerchantUserID,
			"hasSecretKey":   s.Click.SecretKey != "",
		},
		"uzum": map[string]any{
			"enabled":      s.Uzum.Enabled,
			"serviceId":    s.Uzum.ServiceID,
			"login":        s.Uzum.Login,
			"accountField": uzumAccountField(s),
			"hasPassword":  s.Uzum.Password != "",
		},
		"atmos": map[string]any{
			"enabled": s.Atmos.Enabled,
			"storeId": s.Atmos.StoreID,
			"baseUrl": s.Atmos.BaseURL,
			// Three secrets, three flags. Shown apart because they fail apart:
			// a wrong OAuth pair means no checkout link at all, while a wrong
			// callback key means the link works, the guest pays, and the
			// confirmation is refused — the worse of the two, and the one
			// nobody would think to check.
			"hasConsumerKey":    s.Atmos.ConsumerKey != "",
			"hasConsumerSecret": s.Atmos.ConsumerSecret != "",
			"hasApiKey":         s.Atmos.APIKey != "",
		},
	})
}

type paymentSettingsRequest struct {
	ReturnURL string `json:"returnUrl"`
	Payme     struct {
		Enabled      bool   `json:"enabled"`
		MerchantID   string `json:"merchantId"`
		Key          string `json:"key"`
		TestKey      string `json:"testKey"`
		TestMode     bool   `json:"testMode"`
		AccountField string `json:"accountField"`
	} `json:"payme"`
	Click struct {
		Enabled        bool   `json:"enabled"`
		ServiceID      string `json:"serviceId"`
		MerchantID     string `json:"merchantId"`
		MerchantUserID string `json:"merchantUserId"`
		SecretKey      string `json:"secretKey"`
	} `json:"click"`
	Uzum struct {
		Enabled      bool   `json:"enabled"`
		ServiceID    string `json:"serviceId"`
		Login        string `json:"login"`
		Password     string `json:"password"`
		AccountField string `json:"accountField"`
	} `json:"uzum"`
	Atmos struct {
		Enabled        bool   `json:"enabled"`
		StoreID        string `json:"storeId"`
		BaseURL        string `json:"baseUrl"`
		ConsumerKey    string `json:"consumerKey"`
		ConsumerSecret string `json:"consumerSecret"`
		APIKey         string `json:"apiKey"`
	} `json:"atmos"`
}

// AdminUpdatePaymentSettings saves the credentials.
//
// **An empty secret means "keep the stored one"**, never "erase it". The panel
// cannot show the key it is editing, so an owner changing the merchant id would
// otherwise submit a blank key field and silently switch payment off — the same
// trap the kiosk secret fell into, with money on the other side of it.
func (h *Handler) AdminUpdatePaymentSettings(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req paymentSettingsRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	current := h.paymentSettings(r.Context())

	next := models.PaymentSettings{
		ReturnURL: strings.TrimSpace(req.ReturnURL),
		Payme: models.PaymeSettings{
			Enabled:      req.Payme.Enabled,
			MerchantID:   strings.TrimSpace(req.Payme.MerchantID),
			Key:          keepSecret(req.Payme.Key, current.Payme.Key),
			TestKey:      keepSecret(req.Payme.TestKey, current.Payme.TestKey),
			TestMode:     req.Payme.TestMode,
			AccountField: strings.TrimSpace(req.Payme.AccountField),
		},
		Click: models.ClickSettings{
			Enabled:        req.Click.Enabled,
			ServiceID:      strings.TrimSpace(req.Click.ServiceID),
			MerchantID:     strings.TrimSpace(req.Click.MerchantID),
			MerchantUserID: strings.TrimSpace(req.Click.MerchantUserID),
			SecretKey:      keepSecret(req.Click.SecretKey, current.Click.SecretKey),
		},
		Uzum: models.UzumSettings{
			Enabled:      req.Uzum.Enabled,
			ServiceID:    strings.TrimSpace(req.Uzum.ServiceID),
			Login:        strings.TrimSpace(req.Uzum.Login),
			Password:     keepSecret(req.Uzum.Password, current.Uzum.Password),
			AccountField: strings.TrimSpace(req.Uzum.AccountField),
		},
		Atmos: models.AtmosSettings{
			Enabled:        req.Atmos.Enabled,
			StoreID:        strings.TrimSpace(req.Atmos.StoreID),
			BaseURL:        strings.TrimSpace(req.Atmos.BaseURL),
			ConsumerKey:    keepSecret(req.Atmos.ConsumerKey, current.Atmos.ConsumerKey),
			ConsumerSecret: keepSecret(req.Atmos.ConsumerSecret, current.Atmos.ConsumerSecret),
			APIKey:         keepSecret(req.Atmos.APIKey, current.Atmos.APIKey),
		},
		UpdatedAt: time.Now(),
	}

	doc, err := bson.Marshal(next)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var set bson.M
	if err := bson.Unmarshal(doc, &set); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	delete(set, "_id")
	if _, err := h.Store.PaymentSettings.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": set}, options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Which providers were touched, never what the keys are.
	h.logAction(r, ActPaymentSettings, "settings", "payments", "To'lov tizimlari",
		enabledProviders(&next))
	h.AdminGetPaymentSettings(w, r)
}

// enabledProviders is the human-readable half of the audit entry.
func enabledProviders(s *models.PaymentSettings) string {
	var on []string
	for _, p := range []string{
		models.ProviderPayme, models.ProviderClick,
		models.ProviderUzum, models.ProviderAtmos,
	} {
		if s.Configured(p) {
			on = append(on, p)
		}
	}
	if len(on) == 0 {
		return "faqat naqd"
	}
	return strings.Join(on, ", ")
}

// keepSecret implements "empty means unchanged".
func keepSecret(incoming, stored string) string {
	if strings.TrimSpace(incoming) == "" {
		return stored
	}
	return strings.TrimSpace(incoming)
}

// AdminOrderPayments lists what happened to an order's money — every attempt,
// not just the successful one. "The guest says they paid twice" is a question
// only the ledger can answer.
func (h *Handler) AdminOrderPayments(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	cur, err := h.Store.Payments.Find(r.Context(), bson.M{"orderId": id})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.Payment{}
	_ = cur.All(r.Context(), &rows)
	httpx.JSON(w, http.StatusOK, rows)
}

// PublicPaymentMethods tells the checkout which methods it may offer.
//
// Cash is always in the list. A provider appears only when it is switched on
// *and* fully credentialed — offering a button that leads to a bank error page
// costs the order, and the guest blames the restaurant, not the bank.
func (h *Handler) PublicPaymentMethods(w http.ResponseWriter, r *http.Request) {
	s := h.paymentSettings(r.Context())
	methods := []string{models.ProviderCash}
	for _, p := range []string{
		models.ProviderPayme, models.ProviderClick,
		models.ProviderUzum, models.ProviderAtmos,
	} {
		if s.Configured(p) {
			methods = append(methods, p)
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"methods": methods})
}

// paymentStatusOf is the order's payment state, defaulting for the orders that
// predate online payment: those were all cash, and cash is "unpaid" for its
// whole life without that meaning anything is wrong.
func paymentStatusOf(order *models.Order) string {
	if order.PaymentStatus == "" {
		return models.PayUnpaid
	}
	return order.PaymentStatus
}
