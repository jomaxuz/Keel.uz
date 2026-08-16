package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/pos"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Pushing orders to the restaurant's till.
//
// The menu stays ours — the panel is where photos, translations and combos
// live, and where the price is set. What the till gets is the **order**, with
// each line named by the id it has over there. That mapping is the only thing
// an owner has to set up, and it is done once from a screen that shows both
// lists side by side.
//
// Three rules shape the code below:
//
//   - **Nothing goes to the kitchen twice.** Sending is guarded by the order's
//     own POS state, so a double-click, a retry and an auto-send racing an
//     operator all end with one ticket.
//   - **A half-mappable order is refused entirely.** Sending the lines that do
//     have ids would put an incomplete ticket on the pass, and the kitchen
//     would cook exactly what it saw.
//   - **The till not answering is not the order's problem.** A failed push
//     leaves the order alone and visible, with the reason and a retry button.
//     An order that exists in our system and not in theirs is recoverable; an
//     order silently dropped is not.

// posConfig turns a branch's stored settings into the adapter's config.
func posConfig(s *models.POSSettings) pos.Config {
	if s == nil {
		return pos.Config{}
	}
	cfg := pos.Config{
		Provider:           s.Provider,
		IikoAPILogin:       s.Iiko.APILogin,
		IikoOrganizationID: s.Iiko.OrganizationID,
		IikoTerminalGroup:  s.Iiko.TerminalGroup,
		IikoOrderTypeID:    s.Iiko.OrderTypeID,
		IikoPaymentTypeID:  s.Iiko.PaymentTypeID,
		IikoBaseURL:        s.Iiko.BaseURL,
		PosterToken:        s.Poster.Token,
		PosterSpotID:       s.Poster.SpotID,
		PosterBaseURL:      s.Poster.BaseURL,
		CloposClientID:     s.Clopos.ClientID,
		CloposClientSecret: s.Clopos.ClientSecret,
		CloposBrand:        s.Clopos.Brand,
		CloposIntegratorID: s.Clopos.IntegratorID,
		CloposVenueID:      s.Clopos.VenueID,
		CloposSaleTypeID:   s.Clopos.SaleTypeID,
		CloposBaseURL:      s.Clopos.BaseURL,
		RKeeperURL:         s.RKeeper.URL,
		RKeeperLogin:       s.RKeeper.Login,
		RKeeperPassword:    s.RKeeper.Password,
		RKeeperStation:     s.RKeeper.Station,
		RKeeperAnchor:      s.RKeeper.Anchor,
		RKeeperToken:       s.RKeeper.Token,
	}
	// Syrve rides the iiko adapter, so its stored credentials are lifted into
	// the iiko fields here — the one place that knows both. Doing it inside the
	// adapter would mean every future caller had to remember the aliasing.
	if s.Provider == pos.Syrve {
		cfg.IikoAPILogin = s.Syrve.APILogin
		cfg.IikoOrganizationID = s.Syrve.OrganizationID
		cfg.IikoTerminalGroup = s.Syrve.TerminalGroup
		cfg.IikoOrderTypeID = s.Syrve.OrderTypeID
		cfg.IikoPaymentTypeID = s.Syrve.PaymentTypeID
		cfg.IikoBaseURL = s.Syrve.BaseURL
	}
	return cfg
}

// posSettingsOf loads one branch's till connection. A missing document is not
// an error: a restaurant without a till is the normal case.
func (h *Handler) posSettingsOf(ctx context.Context, branchID primitive.ObjectID) *models.POSSettings {
	var s models.POSSettings
	if branchID.IsZero() {
		return &models.POSSettings{}
	}
	if err := h.Store.POSSettings.FindOne(ctx, bson.M{"branchId": branchID}).Decode(&s); err != nil {
		return &models.POSSettings{BranchID: branchID}
	}
	return &s
}

// posFor builds the adapter for a branch, or nil when nothing is connected.
func (h *Handler) posFor(ctx context.Context, branchID primitive.ObjectID) (pos.Provider, *models.POSSettings, error) {
	s := h.posSettingsOf(ctx, branchID)
	if !s.Enabled || s.Provider == "" {
		return nil, s, nil
	}
	p, err := pos.New(posConfig(s))
	return p, s, err
}

// ---- Settings ----

// AdminGetPOS returns the branch's till settings, without the secrets.
func (h *Handler) AdminGetPOS(w http.ResponseWriter, r *http.Request) {
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	s := h.posSettingsOf(r.Context(), branchID)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"branchId": branchID,
		"provider": s.Provider,
		"enabled":  s.Enabled,
		"autoSend": s.AutoSend,
		"iiko": map[string]any{
			"organizationId": s.Iiko.OrganizationID,
			"terminalGroup":  s.Iiko.TerminalGroup,
			"orderTypeId":    s.Iiko.OrderTypeID,
			"paymentTypeId":  s.Iiko.PaymentTypeID,
			"baseUrl":        s.Iiko.BaseURL,
			"hasApiLogin":    s.Iiko.APILogin != "",
		},
		"syrve": map[string]any{
			"organizationId": s.Syrve.OrganizationID,
			"terminalGroup":  s.Syrve.TerminalGroup,
			"orderTypeId":    s.Syrve.OrderTypeID,
			"paymentTypeId":  s.Syrve.PaymentTypeID,
			"baseUrl":        s.Syrve.BaseURL,
			"hasApiLogin":    s.Syrve.APILogin != "",
		},
		"poster": map[string]any{
			"spotId":   s.Poster.SpotID,
			"baseUrl":  s.Poster.BaseURL,
			"hasToken": s.Poster.Token != "",
		},
		"clopos": map[string]any{
			"clientId":     s.Clopos.ClientID,
			"brand":        s.Clopos.Brand,
			"integratorId": s.Clopos.IntegratorID,
			"venueId":      s.Clopos.VenueID,
			"saleTypeId":   s.Clopos.SaleTypeID,
			"baseUrl":      s.Clopos.BaseURL,
			"hasSecret":    s.Clopos.ClientSecret != "",
		},
		"rkeeper": map[string]any{
			"url":         s.RKeeper.URL,
			"login":       s.RKeeper.Login,
			"station":     s.RKeeper.Station,
			"anchor":      s.RKeeper.Anchor,
			"hasPassword": s.RKeeper.Password != "",
			"hasToken":    s.RKeeper.Token != "",
		},
		"lastCheckAt": s.LastCheckAt,
		"lastCheckOk": s.LastCheckOK,
		"lastCheck":   s.LastCheck,
	})
}

type posSettingsRequest struct {
	Provider string `json:"provider"`
	Enabled  bool   `json:"enabled"`
	AutoSend bool   `json:"autoSend"`
	Iiko     struct {
		APILogin       string `json:"apiLogin"`
		OrganizationID string `json:"organizationId"`
		TerminalGroup  string `json:"terminalGroup"`
		OrderTypeID    string `json:"orderTypeId"`
		PaymentTypeID  string `json:"paymentTypeId"`
		BaseURL        string `json:"baseUrl"`
	} `json:"iiko"`
	Syrve struct {
		APILogin       string `json:"apiLogin"`
		OrganizationID string `json:"organizationId"`
		TerminalGroup  string `json:"terminalGroup"`
		OrderTypeID    string `json:"orderTypeId"`
		PaymentTypeID  string `json:"paymentTypeId"`
		BaseURL        string `json:"baseUrl"`
	} `json:"syrve"`
	Poster struct {
		Token   string `json:"token"`
		SpotID  int    `json:"spotId"`
		BaseURL string `json:"baseUrl"`
	} `json:"poster"`
	Clopos struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
		Brand        string `json:"brand"`
		IntegratorID string `json:"integratorId"`
		VenueID      int    `json:"venueId"`
		SaleTypeID   int    `json:"saleTypeId"`
		BaseURL      string `json:"baseUrl"`
	} `json:"clopos"`
	RKeeper struct {
		URL      string `json:"url"`
		Login    string `json:"login"`
		Password string `json:"password"`
		Station  string `json:"station"`
		Anchor   string `json:"anchor"`
		Token    string `json:"token"`
	} `json:"rkeeper"`
}

// AdminUpdatePOS saves the branch's till connection.
//
// Secrets follow the same rule as the payment keys: **empty means keep the
// stored one**. The form cannot show an apiLogin it never received, so a blank
// field must never be read as "erase it" — that would silently disconnect the
// kitchen the next time somebody corrected a terminal id.
func (h *Handler) AdminUpdatePOS(w http.ResponseWriter, r *http.Request) {
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req posSettingsRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	provider := strings.TrimSpace(req.Provider)
	switch provider {
	case "", pos.IIKO, pos.Syrve, pos.Clopos, pos.Poster, pos.RKeeper:
	default:
		httpx.Error(w, http.StatusBadRequest, "noma'lum POS tizimi")
		return
	}

	current := h.posSettingsOf(r.Context(), branchID)
	next := *current
	next.BranchID = branchID
	next.Provider = provider
	next.Enabled = req.Enabled
	next.AutoSend = req.AutoSend
	next.UpdatedAt = time.Now()

	next.Iiko.APILogin = keepSecret(req.Iiko.APILogin, current.Iiko.APILogin)
	next.Iiko.OrganizationID = strings.TrimSpace(req.Iiko.OrganizationID)
	next.Iiko.TerminalGroup = strings.TrimSpace(req.Iiko.TerminalGroup)
	next.Iiko.OrderTypeID = strings.TrimSpace(req.Iiko.OrderTypeID)
	next.Iiko.PaymentTypeID = strings.TrimSpace(req.Iiko.PaymentTypeID)
	next.Iiko.BaseURL = strings.TrimSpace(req.Iiko.BaseURL)

	next.Syrve.APILogin = keepSecret(req.Syrve.APILogin, current.Syrve.APILogin)
	next.Syrve.OrganizationID = strings.TrimSpace(req.Syrve.OrganizationID)
	next.Syrve.TerminalGroup = strings.TrimSpace(req.Syrve.TerminalGroup)
	next.Syrve.OrderTypeID = strings.TrimSpace(req.Syrve.OrderTypeID)
	next.Syrve.PaymentTypeID = strings.TrimSpace(req.Syrve.PaymentTypeID)
	next.Syrve.BaseURL = strings.TrimSpace(req.Syrve.BaseURL)

	next.Poster.Token = keepSecret(req.Poster.Token, current.Poster.Token)
	next.Poster.SpotID = req.Poster.SpotID
	next.Poster.BaseURL = strings.TrimSpace(req.Poster.BaseURL)

	next.Clopos.ClientID = strings.TrimSpace(req.Clopos.ClientID)
	next.Clopos.ClientSecret = keepSecret(req.Clopos.ClientSecret, current.Clopos.ClientSecret)
	next.Clopos.Brand = strings.TrimSpace(req.Clopos.Brand)
	next.Clopos.IntegratorID = strings.TrimSpace(req.Clopos.IntegratorID)
	next.Clopos.VenueID = req.Clopos.VenueID
	next.Clopos.SaleTypeID = req.Clopos.SaleTypeID
	next.Clopos.BaseURL = strings.TrimSpace(req.Clopos.BaseURL)

	next.RKeeper.URL = strings.TrimSpace(req.RKeeper.URL)
	next.RKeeper.Login = strings.TrimSpace(req.RKeeper.Login)
	next.RKeeper.Password = keepSecret(req.RKeeper.Password, current.RKeeper.Password)
	next.RKeeper.Station = strings.TrimSpace(req.RKeeper.Station)
	next.RKeeper.Anchor = strings.TrimSpace(req.RKeeper.Anchor)
	next.RKeeper.Token = keepSecret(req.RKeeper.Token, current.RKeeper.Token)

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
	if _, err := h.Store.POSSettings.UpdateOne(r.Context(), bson.M{"branchId": branchID},
		bson.M{"$set": set}, options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Switching the till on or off changes whether the question applies at all.
	forgetUnmapped(branchID)
	h.logAction(r, ActPOSSettings, "settings", branchID.Hex(), "POS", provider)
	h.AdminGetPOS(w, r)
}

// AdminPingPOS proves the connection works and says what it connected to.
//
// The answer is stored, so the settings screen can show "connected to
// Maracanda, terminal alive — checked 10 minutes ago" without dialling the till
// on every page load.
func (h *Handler) AdminPingPOS(w http.ResponseWriter, r *http.Request) {
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	provider, _, err := h.posFor(r.Context(), branchID)
	if err != nil {
		h.recordCheck(r.Context(), branchID, false, err.Error())
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": false, "message": err.Error()})
		return
	}
	if provider == nil {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok": false, "message": "POS tizimi yoqilmagan",
		})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	name, err := provider.Ping(ctx)
	if err != nil {
		h.recordCheck(r.Context(), branchID, false, err.Error())
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": false, "message": err.Error()})
		return
	}
	h.recordCheck(r.Context(), branchID, true, name)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "message": name})
}

func (h *Handler) recordCheck(ctx context.Context, branchID primitive.ObjectID, ok bool, msg string) {
	now := time.Now()
	_, _ = h.Store.POSSettings.UpdateOne(ctx, bson.M{"branchId": branchID},
		bson.M{"$set": bson.M{
			"lastCheckAt": now, "lastCheckOk": ok, "lastCheck": clampText(msg, 300),
		}}, options.Update().SetUpsert(true))
}

// AdminPOSProducts lists what the till can sell, for the mapping screen.
func (h *Handler) AdminPOSProducts(w http.ResponseWriter, r *http.Request) {
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	provider, _, err := h.posFor(r.Context(), branchID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if provider == nil {
		httpx.JSON(w, http.StatusOK, []pos.Product{})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	products, err := provider.Products(ctx)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	if products == nil {
		products = []pos.Product{}
	}
	httpx.JSON(w, http.StatusOK, products)
}

// ---- Dish mapping ----

// AdminPOSMapping returns what each of our dishes points at in this till.
func (h *Handler) AdminPOSMapping(w http.ResponseWriter, r *http.Request) {
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	cur, err := h.Store.POSMappings.Find(r.Context(), bson.M{"branchId": branchID})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.POSMapping{}
	_ = cur.All(r.Context(), &rows)
	httpx.JSON(w, http.StatusOK, rows)
}

type posMappingRequest struct {
	Items []struct {
		MenuItemID     string `json:"menuItemId"`
		POSProductID   string `json:"posProductId"`
		POSProductName string `json:"posProductName"`
	} `json:"items"`
}

// AdminSavePOSMapping stores the dish→product links the operator picked.
//
// An empty product id removes the link rather than storing a blank one: "not
// mapped" is a real state the send path checks for, and a row holding "" would
// pass a presence test and fail at the till.
func (h *Handler) AdminSavePOSMapping(w http.ResponseWriter, r *http.Request) {
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req posMappingRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	saved, removed := 0, 0
	for _, it := range req.Items {
		menuID, err := objectID(strings.TrimSpace(it.MenuItemID))
		if err != nil {
			continue
		}
		key := bson.M{"branchId": branchID, "menuItemId": menuID}
		if strings.TrimSpace(it.POSProductID) == "" {
			if res, err := h.Store.POSMappings.DeleteOne(r.Context(), key); err == nil {
				removed += int(res.DeletedCount)
			}
			continue
		}
		if _, err := h.Store.POSMappings.UpdateOne(r.Context(), key, bson.M{"$set": bson.M{
			"branchId":       branchID,
			"menuItemId":     menuID,
			"posProductId":   strings.TrimSpace(it.POSProductID),
			"posProductName": clampText(it.POSProductName, 200),
			"updatedAt":      now,
		}}, options.Update().SetUpsert(true)); err == nil {
			saved++
		}
	}
	forgetUnmapped(branchID)
	h.logAction(r, ActPOSMapping, "settings", branchID.Hex(), "POS menyu bog'lash", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"saved": saved, "removed": removed})
}

type posMappingCopyRequest struct {
	FromBranchID string `json:"fromBranchId"`
	// Overwrite replaces links this branch already has. Off by default: the
	// existing link is the one someone checked against this branch's own till.
	Overwrite bool `json:"overwrite"`
}

// AdminCopyPOSMapping copies another branch's dish→product links into this one.
//
// The mapping is per branch on purpose (the menu is the brand's, the till is the
// branch's), but a chain usually runs its kitchens off **one** iiko
// organisation — the product ids are then identical, and mapping a 200-dish menu
// a second time by hand is transcription, not a decision. This is the only place
// that difference is worth automating.
//
// Two guards decide whether copying means anything at all:
//
//   - **Same brand.** A dish id belongs to a brand's menu; carrying links across
//     brands would write rows keyed to dishes the target branch does not serve.
//   - **Same POS provider.** Product ids are that till's namespace. An iiko id
//     pasted into Clopos is not a wrong product, it is a link that fails at send
//     time — and it fails looking exactly like a POS outage.
//
// A target with nothing connected yet is allowed: connecting the till and
// mapping the menu are two steps, and owners do them in that order.
func (h *Handler) AdminCopyPOSMapping(w http.ResponseWriter, r *http.Request) {
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req posMappingCopyRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	fromID, err := objectID(strings.TrimSpace(req.FromBranchID))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "manba filial tanlanmagan")
		return
	}
	if fromID == branchID {
		httpx.Error(w, http.StatusBadRequest, "manba va nishon bir xil filial")
		return
	}
	// Reading another branch's mapping is reading its till, so a pinned manager
	// is held to the same rule as when writing.
	if err := h.requireBranchAccess(r, fromID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}

	ctx := r.Context()
	var src, dst models.Branch
	if err := h.Store.Branches.FindOne(ctx, bson.M{"_id": fromID}).Decode(&src); err != nil {
		httpx.Error(w, http.StatusNotFound, "manba filial topilmadi")
		return
	}
	if err := h.Store.Branches.FindOne(ctx, bson.M{"_id": branchID}).Decode(&dst); err != nil {
		httpx.Error(w, http.StatusNotFound, "filial topilmadi")
		return
	}
	if src.BrandID != dst.BrandID {
		httpx.Error(w, http.StatusBadRequest,
			"bu ikki filial har xil brendga tegishli — menyusi ham boshqa, bog'lashni ko'chirish ma'nosiz")
		return
	}
	srcPOS := h.posSettingsOf(ctx, fromID)
	dstPOS := h.posSettingsOf(ctx, branchID)
	if srcPOS.Provider != "" && dstPOS.Provider != "" && srcPOS.Provider != dstPOS.Provider {
		httpx.Error(w, http.StatusBadRequest,
			"filiallar har xil kassada ("+srcPOS.Provider+" va "+dstPOS.Provider+
				") — mahsulot id'lari mos kelmaydi, har birini alohida bog'lash kerak")
		return
	}

	cur, err := h.Store.POSMappings.Find(ctx, bson.M{"branchId": fromID})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.POSMapping{}
	if err := cur.All(ctx, &rows); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	now := time.Now()
	copied, skipped := 0, 0
	for _, row := range rows {
		if row.MenuItemID.IsZero() || strings.TrimSpace(row.POSProductID) == "" {
			continue
		}
		key := bson.M{"branchId": branchID, "menuItemId": row.MenuItemID}
		fields := bson.M{
			"branchId":       branchID,
			"menuItemId":     row.MenuItemID,
			"posProductId":   row.POSProductID,
			"posProductName": row.POSProductName,
			"updatedAt":      now,
		}
		op := bson.M{"$setOnInsert": fields}
		if req.Overwrite {
			op = bson.M{"$set": fields}
		}
		res, err := h.Store.POSMappings.UpdateOne(ctx, key, op, options.Update().SetUpsert(true))
		if err != nil {
			continue
		}
		if res.UpsertedCount > 0 || res.ModifiedCount > 0 {
			copied++
			continue
		}
		skipped++
	}

	forgetUnmapped(branchID)
	h.logAction(r, ActPOSMapping, "settings", branchID.Hex(),
		"POS bog'lashni ko'chirish", src.Name)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"copied":  copied,
		"skipped": skipped,
		"total":   len(rows),
	})
}

// ---- Sending an order ----

// sendToPOS pushes one order to its branch's till.
//
// Idempotent by the order's own POS state: a second call on an order already
// sent does nothing and says so. That guard matters more than it looks —
// auto-send on confirm, an operator's retry and a double-clicked button can all
// arrive at once, and each extra ticket is a dish the kitchen actually cooks.
func (h *Handler) sendToPOS(ctx context.Context, order *models.Order) (*models.OrderPOS, error) {
	provider, settings, err := h.posFor(ctx, order.BranchID)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, nil // no till connected: not a failure
	}
	if order.POS != nil && order.POS.Status == models.POSSent {
		return order.POS, nil
	}

	items, err := h.posItems(ctx, order)
	if err != nil {
		state := h.recordPOS(ctx, order, settings.Provider, models.POSFailed, "", "", err)
		return state, err
	}

	payload := pos.Order{
		Number:        order.Number,
		Type:          order.Type,
		CustomerName:  order.Customer.Name,
		Phone:         order.Customer.Phone,
		Address:       order.Address.Text,
		Lat:           order.Address.Lat,
		Lng:           order.Address.Lng,
		Comment:       posComment(order),
		Items:         items,
		DeliveryFee:   order.DeliveryFee,
		Total:         order.Total,
		PaymentMethod: order.PaymentMethod,
		Paid:          paymentStatusOf(order) == models.PayPaid,
		TableNumber:   order.TableNumber,
		CreatedAt:     order.CreatedAt,
	}

	callCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	res, err := provider.SendOrder(callCtx, payload)
	if err != nil {
		// The order is left exactly as it was. An order that exists here and
		// not at the till is recoverable — the operator sees why and presses
		// retry; one that was quietly dropped is not.
		state := h.recordPOS(ctx, order, settings.Provider, models.POSFailed,
			res.POSOrderID, res.Note, err)
		return state, err
	}
	state := h.recordPOS(ctx, order, settings.Provider, models.POSSent,
		res.POSOrderID, res.Note, nil)
	return state, nil
}

// posItems turns the order's lines into till lines, naming each by its id over
// there. A dish with no mapping stops the whole order (see pos.CheckMapped).
//
// A combo is not sent as itself: it is expanded into the dishes it is made of,
// with its price split across them. See poscombo.go for why — in short, the
// till has no product for a bundle this site invented, and one line for it
// would stop every order containing a set.
func (h *Handler) posItems(ctx context.Context, order *models.Order) ([]pos.Item, error) {
	// Flatten first, so the mapping lookup covers the dishes actually being
	// sent rather than the sets they came in.
	lines := make([]posLine, 0, len(order.Items))
	for _, it := range order.Items {
		if !orderLineIsCombo(it) {
			lines = append(lines, posLine{
				Source: it.MenuItemID,
				Item: pos.Item{
					Name: it.Name, Qty: it.Qty, Price: it.Price, Comment: it.Comment,
				},
			})
			continue
		}
		members := h.comboMembersFor(ctx, it)
		expanded := expandCombo(it, members)
		if len(expanded) == 0 {
			// The set cannot be resolved — a member was deleted from the menu.
			// Refused by name rather than sent incomplete: a kitchen cooks what
			// the ticket says, and half a set is not a set.
			return nil, fmt.Errorf(
				"%q to'plamining tarkibi aniqlanmadi — menyuda tekshiring", it.Name)
		}
		lines = append(lines, expanded...)
	}

	ids := make([]primitive.ObjectID, 0, len(lines))
	for _, l := range lines {
		ids = append(ids, l.Source)
	}
	mapped := map[primitive.ObjectID]models.POSMapping{}
	cur, err := h.Store.POSMappings.Find(ctx, bson.M{
		"branchId": order.BranchID, "menuItemId": bson.M{"$in": ids},
	})
	if err == nil {
		var rows []models.POSMapping
		if err := cur.All(ctx, &rows); err == nil {
			for _, m := range rows {
				mapped[m.MenuItemID] = m
			}
		}
	}

	items := make([]pos.Item, 0, len(lines))
	for _, l := range lines {
		l.Item.POSID = mapped[l.Source].POSProductID
		items = append(items, l.Item)
	}
	if err := pos.CheckMapped(items); err != nil {
		return nil, err
	}
	return items, nil
}

// orderLineIsCombo reports whether this line was sold as a set.
//
// Read off the order rather than the menu: a dish turned into a combo after the
// fact must not change what last night's ticket contained.
func orderLineIsCombo(it models.OrderItem) bool {
	return len(it.ComboItems) > 0
}

// posComment is everything the kitchen needs that the till has no field for.
func posComment(order *models.Order) string {
	parts := []string{}
	if order.Address.Comment != "" {
		parts = append(parts, order.Address.Comment)
	}
	// Every discount by name: a ticket whose lines add up to more than the
	// receipt is the first thing a shift manager queries.
	for _, d := range order.Discounts {
		parts = append(parts, d.Name)
	}
	if order.TakenBy != "" {
		parts = append(parts, "Telefon: "+order.TakenBy)
	}
	return strings.Join(parts, " · ")
}

// recordPOS writes the outcome onto the order and returns it.
func (h *Handler) recordPOS(
	ctx context.Context, order *models.Order,
	provider, status, posOrderID, note string, cause error,
) *models.OrderPOS {
	attempts := 1
	if order.POS != nil {
		attempts = order.POS.Attempts + 1
	}
	state := &models.OrderPOS{
		Provider: provider, Status: status, POSOrderID: posOrderID,
		Note: note, Attempts: attempts,
	}
	if cause != nil {
		state.Error = clampText(cause.Error(), 400)
	}
	if status == models.POSSent {
		now := time.Now()
		state.SentAt = &now
	}
	_, _ = h.Store.Orders.UpdateByID(ctx, order.ID, bson.M{"$set": bson.M{"pos": state}})
	order.POS = state
	return state
}

// AdminSendOrderToPOS is the retry button: push this order to the till now.
func (h *Handler) AdminSendOrderToPOS(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var order models.Order
	if err := h.Store.Orders.FindOne(r.Context(), bson.M{"_id": id}).Decode(&order); err != nil {
		httpx.Error(w, http.StatusNotFound, "buyurtma topilmadi")
		return
	}
	state, err := h.sendToPOS(r.Context(), &order)
	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok": false, "message": err.Error(), "pos": state,
		})
		return
	}
	if state == nil {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok": false, "message": "bu filialga POS tizimi ulanmagan",
		})
		return
	}
	h.logAction(r, ActPOSSend, "order", id.Hex(), "#"+order.Number, state.Provider)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "pos": state})
}

// autoSendToPOS is the hook the status change calls. Deliberately quiet: a till
// that is down must never stop an operator from confirming an order, so the
// failure is recorded on the order and shown there, not thrown at the caller.
func (h *Handler) autoSendToPOS(ctx context.Context, order *models.Order) {
	settings := h.posSettingsOf(ctx, order.BranchID)
	if !settings.Enabled || settings.Provider == "" || !settings.AutoSend {
		return
	}
	_, _ = h.sendToPOS(ctx, order)
}

// posBranch is which branch's till is being configured: the panel's current
// lens, clamped to what the admin may see.
func (h *Handler) posBranch(r *http.Request) (primitive.ObjectID, error) {
	scope, err := h.adminScope(r)
	if err != nil {
		return primitive.NilObjectID, err
	}
	id := h.scopeBranch(r, scope)
	if id.IsZero() {
		return primitive.NilObjectID, errors.New("filial tanlanmagan")
	}
	return id, nil
}
