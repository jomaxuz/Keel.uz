package handlers

// ---- The open API: other programs reading this restaurant ----
//
// `/api/open/v1/*`, authenticated by a key the owner made in the panel
// (`Authorization: Bearer keel_…`). See docs/DECISIONS.md → "Ochiq API:
// kalitlar va webhook'lar" and docs/open-api.md for the published contract.
//
// ⚠️ **Its own shapes, never a model.** Every answer here is a struct in this
// file, written field by field. The internal API changes every week with the
// screens that use it; a program somebody else wrote cannot change with it, and
// a Mongo model marshalled straight out would also carry the two Go habits we
// already guard our own browser against — an unset ObjectID as
// "000000000000000000000000" (truthy everywhere) and an empty list as `null`.
//
// ⚠️ **Errors are English with a stable code**, like the Uzum Tezkor wire and
// for the same reason: they are read by another company's developer, and a
// message that is Russian or Uzbek depending on a cookie their program never
// sets is one they cannot search for.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

const (
	// OpenAPIBasePath is where the routes are mounted — shown to the owner.
	OpenAPIBasePath = "/api/open/v1"
	// apiKeyPrefix makes a leaked key recognisable to a secret scanner and to
	// a person: "keel_" in a public repo is ours.
	apiKeyPrefix = "keel_"
	// How many characters of the key the panel keeps to name it.
	apiKeyShown = len(apiKeyPrefix) + 6
	// How stale lastUsedAt may be before a request writes it.
	apiKeyTouchEvery = 5 * time.Minute
	// A page of orders. A program that wants more asks again with the cursor.
	openMaxLimit = 100
)

// Error codes. ⚠️ Published: a program branches on these.
const (
	openCodeUnauthorized = "unauthorized"
	openCodeForbidden    = "insufficient_scope"
	openCodeInvalid      = "invalid_request"
	openCodeNotFound     = "not_found"
	openCodeInternal     = "internal_error"
)

func openFail(w http.ResponseWriter, status int, code, message string) {
	httpx.JSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

func apiKeyHash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// newAPIKey mints a key: the prefix and 32 random bytes.
//
// ⚠️ **Hex, not base64.** A base64url key can contain `-`, and a double-click
// in a terminal or an email selects up to the dash — the owner pastes half a
// key into the partner's form and both sides spend an hour on "invalid key".
func newAPIKey() (string, error) {
	return randomHex(apiKeyPrefix, 32)
}

func randomHex(prefix string, n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b), nil
}

type openKeyCtx struct{}

func openKeyFrom(ctx context.Context) *models.APIKey {
	k, _ := ctx.Value(openKeyCtx{}).(*models.APIKey)
	return k
}

// OpenAuth admits a request carrying a live key that has scope ("" for any
// live key).
//
// ⚠️ **Read from the database on every request**, as TezkorAuth reads its
// settings: revoking a key in the panel has to stop the program on its next
// call. The lookup is one indexed read by hash.
func (h *Handler) OpenAuth(scope string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			raw := strings.TrimSpace(r.Header.Get("Authorization"))
			if len(raw) < 7 || !strings.EqualFold(raw[:7], "Bearer ") {
				openFail(w, http.StatusUnauthorized, openCodeUnauthorized,
					"Authorization: Bearer <api key> is required")
				return
			}
			given := strings.TrimSpace(raw[7:])
			if !strings.HasPrefix(given, apiKeyPrefix) {
				openFail(w, http.StatusUnauthorized, openCodeUnauthorized, "API key is invalid or revoked")
				return
			}
			var key models.APIKey
			err := h.Store.APIKeys.FindOne(r.Context(), bson.M{
				"hash": apiKeyHash(given), "revokedAt": nil,
			}).Decode(&key)
			if err != nil {
				openFail(w, http.StatusUnauthorized, openCodeUnauthorized, "API key is invalid or revoked")
				return
			}
			if scope != "" && !key.Has(scope) {
				openFail(w, http.StatusForbidden, openCodeForbidden,
					"this API key does not have the "+scope+" scope")
				return
			}
			h.touchAPIKey(r.Context(), &key)
			next(w, r.WithContext(context.WithValue(r.Context(), openKeyCtx{}, &key)))
		}
	}
}

// touchAPIKey records that a key is in use — at most once per apiKeyTouchEvery,
// and conditionally, so a program polling every second costs one write every
// few minutes rather than one per call.
func (h *Handler) touchAPIKey(ctx context.Context, k *models.APIKey) {
	now := time.Now()
	if k.LastUsedAt != nil && now.Sub(*k.LastUsedAt) < apiKeyTouchEvery {
		return
	}
	_, _ = h.Store.APIKeys.UpdateOne(ctx, bson.M{
		"_id": k.ID,
		"$or": bson.A{
			bson.M{"lastUsedAt": nil},
			bson.M{"lastUsedAt": bson.M{"$lt": now.Add(-apiKeyTouchEvery)}},
		},
	}, bson.M{"$set": bson.M{"lastUsedAt": now}})
}

// ---- GET /ping: "is my key right?" ----

// OpenPing answers the first question anybody integrating asks, before they
// have read anything else: which restaurant is this, and what may I do.
func (h *Handler) OpenPing(w http.ResponseWriter, r *http.Request) {
	k := openKeyFrom(r.Context())
	scopes := append([]string{}, k.Scopes...)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"restaurant": h.restaurantName(r.Context()),
		"key":        map[string]any{"name": k.Name, "prefix": k.Prefix, "scopes": scopes},
		"apiVersion": "v1",
	})
}

// ---- GET /branches ----

type openBranch struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Phones  []string `json:"phones"`
	Address string   `json:"address"`
	Lat     float64  `json:"lat"`
	Lng     float64  `json:"lng"`
}

// OpenBranches lists the places that cook — the id every other call is scoped
// by.
func (h *Handler) OpenBranches(w http.ResponseWriter, r *http.Request) {
	cur, err := h.Store.Branches.Find(r.Context(), bson.M{},
		options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		openFail(w, http.StatusInternalServerError, openCodeInternal, "branches could not be read")
		return
	}
	var rows []models.Branch
	if err := cur.All(r.Context(), &rows); err != nil {
		openFail(w, http.StatusInternalServerError, openCodeInternal, "branches could not be read")
		return
	}
	out := make([]openBranch, 0, len(rows))
	for _, b := range rows {
		phones := append([]string{}, b.Phones...)
		out = append(out, openBranch{
			ID: b.ID.Hex(), Name: b.Name, Phones: phones,
			Address: b.Address.Text, Lat: b.Address.Lat, Lng: b.Address.Lng,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"branches": out})
}

// ---- GET /branches/{branchId}/menu ----

type openCategory struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	NameRu    string `json:"nameRu"`
	NameEn    string `json:"nameEn"`
	SortOrder int    `json:"sortOrder"`
}

type openMenuItem struct {
	ID          string   `json:"id"`
	CategoryID  string   `json:"categoryId"`
	Name        string   `json:"name"`
	NameRu      string   `json:"nameRu"`
	NameEn      string   `json:"nameEn"`
	Description string   `json:"description"`
	Price       int      `json:"price"`
	ImageURL    string   `json:"imageUrl"`
	Available   bool     `json:"available"`
	Tags        []string `json:"tags"`
	Barcode     string   `json:"barcode,omitempty"`
}

// OpenMenu is the branch's menu with what can be sold there right now.
//
// ⚠️ **`available` is every reason we already refuse a sale, and no new one** —
// switched off, any stop list, a combo with a member that ran out, a daily
// limit reached — the same answer Uzum Tezkor's availability gets (tezkorStock),
// so no program is told something our own site would refuse.
func (h *Handler) OpenMenu(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "branchId"))
	if err != nil {
		openFail(w, http.StatusNotFound, openCodeNotFound, "branch not found")
		return
	}
	branch, err := h.branchByIDCtx(r.Context(), id)
	if err != nil || branch == nil {
		openFail(w, http.StatusNotFound, openCodeNotFound, "branch not found")
		return
	}
	cats, items, err := h.tezkorMenu(r.Context(), branch)
	if err != nil {
		openFail(w, http.StatusInternalServerError, openCodeInternal, "menu could not be read")
		return
	}
	sold := map[primitive.ObjectID]int{}
	if len(branch.DailyLimits) > 0 {
		if s, err := h.soldToday(r.Context(), branch.ID); err == nil {
			sold = s
		}
	}
	active := map[primitive.ObjectID]bool{}
	outCats := make([]openCategory, 0, len(cats))
	for _, c := range cats {
		if !c.IsActive {
			continue
		}
		active[c.ID] = true
		outCats = append(outCats, openCategory{
			ID: c.ID.Hex(), Name: c.Name, NameRu: c.NameRu, NameEn: c.NameEn, SortOrder: c.SortOrder,
		})
	}
	outItems := make([]openMenuItem, 0, len(items))
	for i := range items {
		it := &items[i]
		// A model row (sizes, colours) is a heading, not a thing to sell — the
		// same rule as the public menu.
		if !active[it.CategoryID] || len(it.VariantAxes) > 0 {
			continue
		}
		tags := append([]string{}, it.Tags...)
		outItems = append(outItems, openMenuItem{
			ID: it.ID.Hex(), CategoryID: it.CategoryID.Hex(),
			Name: it.Name, NameRu: it.NameRu, NameEn: it.NameEn,
			Description: it.Description, Price: it.Price,
			ImageURL:  h.absoluteURL(it.ImageURL),
			Available: tezkorStock(it, branch, sold) > 0,
			Tags:      tags, Barcode: it.Barcode,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"branchId": branch.ID.Hex(), "categories": outCats, "items": outItems,
	})
}

// absoluteURL makes a stored upload path usable from another server: the seed
// stores `/uploads/x.jpg`, and a program has no page to resolve it against.
func (h *Handler) absoluteURL(u string) string {
	if u == "" || strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	return strings.TrimRight(h.Cfg.PublicBaseURL, "/") + "/" + strings.TrimLeft(u, "/")
}

// ---- Orders ----

// OpenOrder is the published shape of an order — also the `order` inside every
// webhook, so a receiver parses one thing.
type OpenOrder struct {
	ID            string            `json:"id"`
	Number        string            `json:"number"`
	BranchID      string            `json:"branchId"`
	Type          string            `json:"type"`
	Channel       string            `json:"channel"`
	Status        string            `json:"status"`
	CancelReason  string            `json:"cancelReason,omitempty"`
	Customer      openCustomer      `json:"customer"`
	Address       *openAddress      `json:"address,omitempty"`
	TableNumber   string            `json:"tableNumber,omitempty"`
	Items         []openOrderItem   `json:"items"`
	Subtotal      int               `json:"subtotal"`
	DiscountTotal int               `json:"discountTotal"`
	DeliveryFee   int               `json:"deliveryFee"`
	ServiceCharge int               `json:"serviceCharge"`
	Total         int               `json:"total"`
	PaymentMethod string            `json:"paymentMethod"`
	PaymentStatus string            `json:"paymentStatus"`
	ScheduledAt   *time.Time        `json:"scheduledAt,omitempty"`
	StatusHistory []openStatusEvent `json:"statusHistory"`
	CreatedAt     time.Time         `json:"createdAt"`
}

type openCustomer struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type openAddress struct {
	Text    string  `json:"text"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Comment string  `json:"comment"`
}

type openOrderItem struct {
	MenuItemID string           `json:"menuItemId"`
	Name       string           `json:"name"`
	Price      int              `json:"price"`
	Qty        int              `json:"qty"`
	Options    []openItemOption `json:"options"`
	Comment    string           `json:"comment,omitempty"`
}

type openItemOption struct {
	Group      string `json:"group"`
	Choice     string `json:"choice"`
	PriceDelta int    `json:"priceDelta"`
}

type openStatusEvent struct {
	Status string    `json:"status"`
	At     time.Time `json:"at"`
}

// openOrderOf writes the published order from ours.
//
// ⚠️ Every id through hexOrEmpty and every list made non-nil: see the file
// comment. The address is left out rather than zeroed when there is none — a
// pickup "at 0,0" is a point in the Gulf of Guinea.
func openOrderOf(o *models.Order) OpenOrder {
	out := OpenOrder{
		ID: o.ID.Hex(), Number: o.Number, BranchID: hexOrEmpty(o.BranchID),
		Type: o.Type, Channel: o.Channel, Status: string(o.Status),
		CancelReason: o.CancelReason,
		Customer:     openCustomer{Name: o.Customer.Name, Phone: o.Customer.Phone},
		TableNumber:  o.TableNumber,
		Subtotal:     o.Subtotal, DiscountTotal: o.DiscountTotal,
		DeliveryFee: o.DeliveryFee, ServiceCharge: o.ServiceCharge, Total: o.Total,
		PaymentMethod: o.PaymentMethod, PaymentStatus: o.PaymentStatus,
		ScheduledAt: o.ScheduledAt, CreatedAt: o.CreatedAt,
		Items:         make([]openOrderItem, 0, len(o.Items)),
		StatusHistory: make([]openStatusEvent, 0, len(o.StatusHistory)),
	}
	if o.Address.Text != "" || o.Address.Lat != 0 || o.Address.Lng != 0 {
		out.Address = &openAddress{
			Text: o.Address.Text, Lat: o.Address.Lat, Lng: o.Address.Lng, Comment: o.Address.Comment,
		}
	}
	for _, it := range o.Items {
		line := openOrderItem{
			MenuItemID: hexOrEmpty(it.MenuItemID), Name: it.Name, Price: it.Price, Qty: it.Qty,
			Comment: it.Comment, Options: make([]openItemOption, 0, len(it.Options)),
		}
		for _, op := range it.Options {
			line.Options = append(line.Options, openItemOption{
				Group: op.Name, Choice: op.Choice, PriceDelta: op.PriceDelta,
			})
		}
		out.Items = append(out.Items, line)
	}
	for _, ev := range o.StatusHistory {
		out.StatusHistory = append(out.StatusHistory, openStatusEvent{Status: string(ev.Status), At: ev.At})
	}
	return out
}

// OpenGetOrder is one order, by its number or its id.
func (h *Handler) OpenGetOrder(w http.ResponseWriter, r *http.Request) {
	ref := strings.TrimSpace(chi.URLParam(r, "ref"))
	filter := bson.M{"number": ref}
	if id, err := primitive.ObjectIDFromHex(ref); err == nil {
		filter = bson.M{"_id": id}
	}
	var o models.Order
	if err := h.Store.Orders.FindOne(r.Context(), filter).Decode(&o); err != nil {
		openFail(w, http.StatusNotFound, openCodeNotFound, "order not found")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"order": openOrderOf(&o)})
}

// OpenListOrders pages through orders, newest first.
//
// Filters: `branchId`, `status`, `createdFrom` / `createdTo` (RFC 3339),
// `limit` (≤ 100). Paging: `cursor` from the previous page's `nextCursor`.
//
// ⚠️ **A cursor, not an offset.** Orders arrive while a program is paging;
// with `?page=2` every new order pushes one row from page one onto page two,
// and it is read twice — or, going the other way, one is never read.
func (h *Handler) OpenListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := bson.M{}
	if v := q.Get("branchId"); v != "" {
		id, err := primitive.ObjectIDFromHex(v)
		if err != nil {
			openFail(w, http.StatusBadRequest, openCodeInvalid, "branchId is not a valid id")
			return
		}
		filter["branchId"] = id
	}
	if v := q.Get("status"); v != "" {
		filter["status"] = v
	}
	created := bson.M{}
	for param, op := range map[string]string{"createdFrom": "$gte", "createdTo": "$lt"} {
		if v := q.Get(param); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				openFail(w, http.StatusBadRequest, openCodeInvalid, param+" must be RFC 3339, e.g. 2026-09-24T00:00:00+05:00")
				return
			}
			created[op] = t
		}
	}
	if len(created) > 0 {
		filter["createdAt"] = created
	}
	limit := 50
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > openMaxLimit {
			openFail(w, http.StatusBadRequest, openCodeInvalid, "limit must be between 1 and 100")
			return
		}
		limit = n
	}
	if v := q.Get("cursor"); v != "" {
		at, id, ok := parseOpenCursor(v)
		if !ok {
			openFail(w, http.StatusBadRequest, openCodeInvalid, "cursor is not one this API issued")
			return
		}
		// Strictly after the last row of the previous page, in (createdAt, _id)
		// order — two orders in the same millisecond are both read, once each.
		filter["$or"] = bson.A{
			bson.M{"createdAt": bson.M{"$lt": at}},
			bson.M{"createdAt": at, "_id": bson.M{"$lt": id}},
		}
	}
	cur, err := h.Store.Orders.Find(r.Context(), filter, options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(limit+1)))
	if err != nil {
		openFail(w, http.StatusInternalServerError, openCodeInternal, "orders could not be read")
		return
	}
	var rows []models.Order
	if err := cur.All(r.Context(), &rows); err != nil {
		openFail(w, http.StatusInternalServerError, openCodeInternal, "orders could not be read")
		return
	}
	next := ""
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		next = openCursor(last.CreatedAt, last.ID)
	}
	out := make([]OpenOrder, 0, len(rows))
	for i := range rows {
		out = append(out, openOrderOf(&rows[i]))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"orders": out, "nextCursor": next})
}

func openCursor(at time.Time, id primitive.ObjectID) string {
	return base64.RawURLEncoding.EncodeToString(
		[]byte(strconv.FormatInt(at.UnixMilli(), 10) + "_" + id.Hex()))
}

func parseOpenCursor(s string) (time.Time, primitive.ObjectID, bool) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, primitive.NilObjectID, false
	}
	ms, hexID, ok := strings.Cut(string(b), "_")
	if !ok {
		return time.Time{}, primitive.NilObjectID, false
	}
	n, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, primitive.NilObjectID, false
	}
	id, err := primitive.ObjectIDFromHex(hexID)
	if err != nil {
		return time.Time{}, primitive.NilObjectID, false
	}
	return time.UnixMilli(n), id, true
}
