package pos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Clopos Open API v2 — integrations.clopos.com/open-api/v2.
//
// The tidiest of the three: an OpenAPI document describes it, reads are GETs
// with pagination, and the order body is flat. Credentials are exchanged for a
// JWT which goes in an `x-token` header (not `Authorization`), and the brand
// and venue are baked into the token rather than repeated per call.
//
// Two things to know. Prices are decimal, not whole so'm, so they are converted
// at the edge. And an order is created PENDING: `auto_order_accept` and
// `auto_order_sent_to_station` are what actually put it in front of the
// kitchen, which is why they are sent as true — an order that sits unaccepted
// in the till is indistinguishable, from the guest's side, from one that never
// arrived.
type cloposClient struct {
	cfg  Config
	http *http.Client
	base string

	mu      sync.Mutex
	token   string
	expires time.Time
}

func newClopos(cfg Config, client *http.Client) *cloposClient {
	base := strings.TrimRight(cfg.CloposBaseURL, "/")
	if base == "" {
		base = "https://integrations.clopos.com/open-api/v2"
	}
	return &cloposClient{cfg: cfg, http: client, base: base}
}

func (c *cloposClient) Name() string { return Clopos }

func (c *cloposClient) auth(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expires) {
		return c.token, nil
	}
	body, _ := json.Marshal(map[string]string{
		"client_id":     c.cfg.CloposClientID,
		"client_secret": c.cfg.CloposClientSecret,
		"brand":         c.cfg.CloposBrand,
		"integrator_id": c.cfg.CloposIntegratorID,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/auth",
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("clopos: ulanib bo'lmadi: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var out struct {
		Success   bool   `json:"success"`
		Token     string `json:"token"`
		ExpiresIn int    `json:"expires_in"`
		Message   string `json:"message"`
		Error     string `json:"error"`
	}
	_ = json.Unmarshal(raw, &out)
	if out.Token == "" {
		// The body carries the reason ("integrator_id is required", a bad
		// brand); the bare status does not, and the owner can only act on the
		// reason. Falls back to a trimmed body when the shape is unfamiliar.
		return "", fmt.Errorf("clopos: kirish rad etildi (%s)",
			firstNonEmpty(out.Message, out.Error, clampBody(raw), res.Status))
	}
	ttl := time.Duration(out.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = time.Hour
	}
	// A minute of margin, so a call started at the very end of the window does
	// not arrive with a token that expired in flight.
	c.token, c.expires = out.Token, time.Now().Add(ttl-time.Minute)
	return c.token, nil
}

func (c *cloposClient) do(ctx context.Context, method, path string, in, out any) error {
	token, err := c.auth(ctx)
	if err != nil {
		return err
	}
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("x-token", token)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("clopos %s: %w", path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode >= 400 {
		var e struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		return fmt.Errorf("clopos %s: %s", path,
			firstNonEmpty(e.Message, e.Error, res.Status))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func (c *cloposClient) Ping(ctx context.Context) (string, error) {
	var out struct {
		Data []struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			IsMain int    `json:"is_main"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/venues", nil, &out); err != nil {
		return "", err
	}
	for _, v := range out.Data {
		if v.ID == c.cfg.CloposVenueID {
			return v.Name, nil
		}
	}
	// Connected, but pointed at a venue this brand does not have — worth
	// saying plainly, with the ids that do exist.
	names := make([]string, 0, len(out.Data))
	for _, v := range out.Data {
		names = append(names, fmt.Sprintf("%d — %s", v.ID, v.Name))
	}
	return "", fmt.Errorf("clopos: venue_id %d topilmadi. Mavjudlari: %s",
		c.cfg.CloposVenueID, strings.Join(names, "; "))
}

func (c *cloposClient) Products(ctx context.Context) ([]Product, error) {
	categories := map[int]string{}
	var cats struct {
		Data []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/categories?limit=200", nil, &cats); err == nil {
		for _, x := range cats.Data {
			categories[x.ID] = x.Name
		}
	}

	// What the till has stopped. `limit` here is Clopos's remaining stock, so
	// zero or less is the thing that cannot be sold.
	stopped := map[int]bool{}
	var stop struct {
		Data []struct {
			ID    int `json:"id"`
			Limit int `json:"limit"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/products/stop-list", nil, &stop); err == nil {
		for _, s := range stop.Data {
			if s.Limit <= 0 {
				stopped[s.ID] = true
			}
		}
	}

	// Paged: a real menu runs to hundreds of rows and the default page is far
	// smaller than that.
	out := []Product{}
	for page := 1; page <= 20; page++ {
		var resp struct {
			Data []struct {
				ID         int     `json:"id"`
				Name       string  `json:"name"`
				FullName   string  `json:"full_name"`
				Price      float64 `json:"price"`
				Status     int     `json:"status"`
				CategoryID int     `json:"category_id"`
				Type       string  `json:"type"`
			} `json:"data"`
			Meta struct {
				CurrentPage int `json:"current_page"`
				LastPage    int `json:"last_page"`
			} `json:"meta"`
		}
		q := url.Values{"limit": {"200"}, "page": {strconv.Itoa(page)}}
		if err := c.do(ctx, http.MethodGet, "/products?"+q.Encode(), nil, &resp); err != nil {
			return nil, err
		}
		for _, p := range resp.Data {
			out = append(out, Product{
				ID:       strconv.Itoa(p.ID),
				Name:     firstNonEmpty(p.FullName, p.Name),
				Category: categories[p.CategoryID],
				Price:    int(p.Price),
				// status 0 is a product switched off in the till.
				Unavailable: stopped[p.ID] || p.Status == 0,
			})
		}
		if len(resp.Data) == 0 || resp.Meta.LastPage == 0 ||
			resp.Meta.CurrentPage >= resp.Meta.LastPage {
			break
		}
	}
	return out, nil
}

func (c *cloposClient) SendOrder(ctx context.Context, o Order) (Result, error) {
	if err := CheckMapped(o.Items); err != nil {
		return Result{}, err
	}

	products := make([]map[string]any, 0, len(o.Items))
	for i, it := range o.Items {
		id, err := strconv.Atoi(it.POSID)
		if err != nil {
			return Result{}, fmt.Errorf("clopos: %s uchun mahsulot id raqam bo'lishi kerak (%q)",
				it.Name, it.POSID)
		}
		line := map[string]any{
			"product_id":   id,
			"product_name": it.Name,
			"count":        it.Qty,
			"price":        it.Price,
			"status":       "new",
			// Required by the API and only has to be unique within the order;
			// the line's position plus the order number is exactly that.
			"product_hash": fmt.Sprintf("%s-%d", o.Number, i+1),
			"portion_size": 1,
		}
		if len(it.Modifiers) > 0 {
			mods := make([]map[string]any, 0, len(it.Modifiers))
			for _, m := range it.Modifiers {
				mid, err := strconv.Atoi(m.POSID)
				if err != nil {
					continue
				}
				mods = append(mods, map[string]any{
					"modifier_id": mid, "modifier_name": m.Name,
					"count": m.Qty, "price": m.Price,
				})
			}
			line["modifiers"] = mods
		}
		products = append(products, line)
	}

	comment := o.Comment
	if o.Type == "pickup" {
		comment = strings.TrimSpace("Olib ketish. " + comment)
	}
	body := map[string]any{
		"venue_id":     c.cfg.CloposVenueID,
		"sale_type_id": c.cfg.CloposSaleTypeID,
		// Without these the order sits unaccepted in the till, which from the
		// guest's side is indistinguishable from never having arrived.
		"auto_order_accept":          true,
		"auto_order_sent_to_station": true,
		"comment":                    comment,
		"customer": map[string]any{
			"id":      0,
			"name":    o.CustomerName,
			"phone":   o.Phone,
			"address": o.Address,
		},
		"products": products,
	}
	if o.DeliveryFee > 0 {
		body["delivery_fee"] = o.DeliveryFee
	}

	var out struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Status  string `json:"status"`
			Payload struct {
				ID int `json:"id"`
			} `json:"payload"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodPost, "/orders", body, &out); err != nil {
		return Result{}, err
	}
	if out.Data.Payload.ID == 0 {
		return Result{}, fmt.Errorf("clopos: buyurtma yaratilmadi (%s)",
			firstNonEmpty(out.Message, out.Data.Status))
	}
	return Result{
		POSOrderID: strconv.Itoa(out.Data.Payload.ID),
		Note:       "Clopos #" + strconv.Itoa(out.Data.Payload.ID),
	}, nil
}

func (c *cloposClient) OrderStatus(ctx context.Context, posOrderID string) (Status, error) {
	if posOrderID == "" {
		return Status{}, ErrUnsupported
	}
	var out struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/orders/"+url.PathEscape(posOrderID), nil, &out); err != nil {
		return Status{}, err
	}
	return Status{State: cloposState(out.Data.Status), Raw: out.Data.Status}, nil
}

func cloposState(s string) string {
	switch strings.ToUpper(s) {
	case "PENDING", "NEW":
		return "accepted"
	case "PREPARING", "COOKING":
		return "cooking"
	case "READY", "PREPARED":
		return "ready"
	case "CLOSED", "COMPLETED", "PAID":
		return "closed"
	case "IGNORE", "CANCELLED", "CANCELED":
		return "cancelled"
	}
	return "unknown"
}

// Cancel withdraws the order. Clopos does this by setting the status to
// IGNORE — there is no separate cancel endpoint.
func (c *cloposClient) Cancel(ctx context.Context, posOrderID string) error {
	if posOrderID == "" {
		return ErrUnsupported
	}
	return c.do(ctx, http.MethodPut, "/orders/"+url.PathEscape(posOrderID),
		map[string]any{"status": "IGNORE"}, nil)
}

// clampBody is an unrecognised error body, cut to something a settings page can
// show without becoming a wall of JSON.
func clampBody(raw []byte) string {
	text := strings.TrimSpace(string(raw))
	if len(text) > 200 {
		return text[:200] + "…"
	}
	return text
}
