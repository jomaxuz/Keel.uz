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
)

// Poster — joinposter.com.
//
// The simplest of the four to talk to: reads are GET with a `token` query
// parameter, the one write is a POST of JSON. There is no session, no token
// exchange, no signature.
//
// Two things about it are not simple, and both can be got wrong quietly:
//
//  1. **Money is in tiyin.** Poster's `price` is the minor unit throughout —
//     `"30000"` is 300 so'm. An order pushed in whole so'm arrives a hundred
//     times cheaper and the till accepts it without a murmur. Exactly the
//     r_keeper trap, in a different currency.
//
//  2. **An incoming order is not yet an order.** `createIncomingOrder` files
//     it as an *online order* with `status: 0`, and somebody at the till has
//     to accept it before it becomes a real transaction. Treating the 200 as
//     "the kitchen has it" produces the same lie iiko's async create does.
//     So `status` is mapped honestly: 0 is pending, not accepted.
//
// Poster has no method to cancel an incoming order — `incomingOrders` offers
// create and read only. Cancel therefore returns ErrUnsupported rather than
// pretending, which is the truth an operator needs: withdraw it at the till.
type posterClient struct {
	cfg  Config
	http *http.Client
	base string
}

func newPoster(cfg Config, client *http.Client) *posterClient {
	base := strings.TrimRight(cfg.PosterBaseURL, "/")
	if base == "" {
		base = "https://joinposter.com/api"
	}
	return &posterClient{cfg: cfg, http: client, base: base}
}

func (c *posterClient) Name() string { return Poster }

// posterError is the shape Poster uses when it refuses. `error` is 0 or absent
// on success, so its presence alone is not a failure.
type posterError struct {
	Error   flexInt `json:"error"`
	Message string  `json:"message"`
}

func (c *posterClient) endpoint(method string, q url.Values) string {
	if q == nil {
		q = url.Values{}
	}
	q.Set("token", c.cfg.PosterToken)
	return c.base + "/" + method + "?" + q.Encode()
}

// do runs one call and unwraps Poster's `{"response": …}` envelope.
func (c *posterClient) do(ctx context.Context, method string, q url.Values, body any, out any) error {
	var req *http.Request
	var err error
	if body == nil {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint(method, q), nil)
	} else {
		var raw []byte
		if raw, err = json.Marshal(body); err == nil {
			req, err = http.NewRequestWithContext(ctx, http.MethodPost,
				c.endpoint(method, q), bytes.NewReader(raw))
			if req != nil {
				req.Header.Set("Content-Type", "application/json")
			}
		}
	}
	if err != nil {
		return err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("poster %s: ulanib bo'lmadi: %w", method, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))

	// Poster answers 200 with an error body more often than it answers 4xx, so
	// the body is inspected before the status.
	var e posterError
	_ = json.Unmarshal(raw, &e)
	if int(e.Error) != 0 {
		return fmt.Errorf("poster %s: %s", method,
			firstNonEmpty(e.Message, "xato kodi "+strconv.Itoa(int(e.Error))))
	}
	if res.StatusCode >= 400 {
		return fmt.Errorf("poster %s: %s", method, firstNonEmpty(e.Message, res.Status))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func (c *posterClient) Ping(ctx context.Context) (string, error) {
	var out struct {
		Response []struct {
			SpotID  flexInt `json:"spot_id"`
			Name    string  `json:"spot_name"`
			Address string  `json:"spot_adress"`
		} `json:"response"`
	}
	if err := c.do(ctx, "spots.getSpots", nil, nil, &out); err != nil {
		return "", err
	}
	if len(out.Response) == 0 {
		return "", fmt.Errorf("poster: bu token uchun savdo nuqtasi topilmadi")
	}
	// Naming the spot the orders will actually land in is the whole point of
	// the check: a valid token pointed at the wrong branch looks identical to
	// a correct setup until the first order prints in another building.
	for _, s := range out.Response {
		if int(s.SpotID) == c.cfg.PosterSpotID {
			label := firstNonEmpty(s.Name, "savdo nuqtasi "+strconv.Itoa(int(s.SpotID)))
			if s.Address != "" {
				label += " · " + s.Address
			}
			return label, nil
		}
	}
	names := make([]string, 0, len(out.Response))
	for _, s := range out.Response {
		names = append(names, strconv.Itoa(int(s.SpotID))+" — "+s.Name)
	}
	return "", fmt.Errorf("poster: %d raqamli savdo nuqtasi yo'q. Mavjudlari: %s",
		c.cfg.PosterSpotID, strings.Join(names, ", "))
}

func (c *posterClient) Products(ctx context.Context) ([]Product, error) {
	var out struct {
		Response []struct {
			ProductID   flexInt            `json:"product_id"`
			ProductName string             `json:"product_name"`
			Category    string             `json:"category_name"`
			Hidden      flexInt            `json:"hidden"`
			Price       map[string]flexInt `json:"price"`
			Spots       []struct {
				SpotID  flexInt `json:"spot_id"`
				Price   flexInt `json:"price"`
				Visible flexInt `json:"visible"`
			} `json:"spots"`
		} `json:"response"`
	}
	if err := c.do(ctx, "menu.getProducts", nil, nil, &out); err != nil {
		return nil, err
	}
	spot := strconv.Itoa(c.cfg.PosterSpotID)
	items := make([]Product, 0, len(out.Response))
	for _, p := range out.Response {
		if p.ProductID == 0 {
			continue
		}
		item := Product{
			ID:          strconv.Itoa(int(p.ProductID)),
			Name:        p.ProductName,
			Category:    p.Category,
			Unavailable: p.Hidden != 0,
		}
		// Prices come both as a spot→price map and inside `spots`. The map is
		// the reliable one; `spots` additionally says whether this branch sells
		// the thing at all, which the global `hidden` flag does not.
		if v, ok := p.Price[spot]; ok {
			item.Price = int(v) / 100
		}
		for _, s := range p.Spots {
			if int(s.SpotID) != c.cfg.PosterSpotID {
				continue
			}
			if s.Price != 0 {
				item.Price = int(s.Price) / 100
			}
			if s.Visible == 0 {
				item.Unavailable = true
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func (c *posterClient) SendOrder(ctx context.Context, o Order) (Result, error) {
	type posterProduct struct {
		ProductID     int `json:"product_id"`
		Count         int `json:"count"`
		Price         int `json:"price"`
		ModificatorID int `json:"modificator_id,omitempty"`
	}
	products := make([]posterProduct, 0, len(o.Items))
	for _, it := range o.Items {
		id, err := strconv.Atoi(it.POSID)
		if err != nil {
			return Result{}, fmt.Errorf("poster: %q taomining kassadagi id'si raqam emas (%s)",
				it.Name, it.POSID)
		}
		p := posterProduct{
			ProductID: id,
			Count:     it.Qty,
			// Tiyinda. Bu qator o'chirilsa kassa o'z narxini qo'yadi — mijoz
			// bilan kelishilgan summa emas.
			Price: it.Price * 100,
		}
		if len(it.Modifiers) > 0 {
			if mid, err := strconv.Atoi(it.Modifiers[0].POSID); err == nil {
				p.ModificatorID = mid
			}
		}
		products = append(products, p)
	}

	body := map[string]any{
		"spot_id":  c.cfg.PosterSpotID,
		"phone":    posterPhone(o.Phone),
		"products": products,
		"comment":  posterComment(o),
	}
	if name := strings.TrimSpace(o.CustomerName); name != "" {
		body["first_name"] = name
	}
	if o.Address != "" {
		body["address"] = o.Address
	}
	// Only claim money already taken. Poster reads `type: 1` as prepaid and
	// will not ask the courier for cash; saying so about an unpaid order is how
	// a shift comes up short at the end of the day.
	if o.Paid {
		body["payment"] = map[string]any{
			"type":     1,
			"sum":      o.Total * 100,
			"currency": "UZS",
		}
	}

	var out struct {
		Response struct {
			IncomingOrderID flexInt `json:"incoming_order_id"`
			Status          flexInt `json:"status"`
		} `json:"response"`
	}
	if err := c.do(ctx, "incomingOrders.createIncomingOrder", nil, body, &out); err != nil {
		return Result{}, err
	}
	if out.Response.IncomingOrderID == 0 {
		return Result{}, fmt.Errorf("poster: buyurtma raqami qaytmadi — kassada tekshiring")
	}
	res := Result{POSOrderID: strconv.Itoa(int(out.Response.IncomingOrderID))}
	res.Note = "Poster №" + res.POSOrderID
	if out.Response.Status == 0 {
		// Said plainly, because it is the difference between "the kitchen is
		// cooking" and "the ticket is waiting for somebody to press accept".
		res.Note += " · kassada qabul qilinishi kutilmoqda"
	}
	return res, nil
}

func (c *posterClient) OrderStatus(ctx context.Context, posOrderID string) (Status, error) {
	q := url.Values{}
	q.Set("incoming_order_id", posOrderID)
	var out struct {
		Response struct {
			Status        flexInt `json:"status"`
			TransactionID flexInt `json:"transaction_id"`
		} `json:"response"`
	}
	if err := c.do(ctx, "incomingOrders.getIncomingOrder", q, nil, &out); err != nil {
		return Status{}, err
	}
	st := Status{State: posterState(int(out.Response.Status))}
	switch int(out.Response.Status) {
	case 0:
		st.Raw = "kassada qabul qilinishi kutilmoqda"
	case 1:
		st.Raw = "qabul qilindi"
		// Once accepted the online order becomes a real till transaction; its
		// number is what the manager will look for in the day's takings.
		if out.Response.TransactionID != 0 {
			st.Raw += " · chek №" + strconv.Itoa(int(out.Response.TransactionID))
		}
	case 7:
		st.Raw = "bekor qilindi"
	default:
		st.Raw = "holat kodi " + strconv.Itoa(int(out.Response.Status))
	}
	return st, nil
}

// posterState maps Poster's incoming-order status onto ours.
//
// `0` is deliberately not "accepted": the order is filed but nobody at the
// till has taken it yet, and reporting that as accepted is the one mistake
// that makes the integration lie about the kitchen.
func posterState(s int) string {
	switch s {
	case 0:
		return "unknown"
	case 1:
		return "accepted"
	case 7:
		return "cancelled"
	}
	return "unknown"
}

// Cancel is not offered by Poster's API — the incomingOrders group has create
// and read only. Saying so is better than a no-op that looks like success.
func (c *posterClient) Cancel(ctx context.Context, posOrderID string) error {
	return ErrUnsupported
}

// posterComment folds everything Poster has no field for into the one place
// the kitchen will actually read.
//
// The API takes a single order-level comment: no per-line notes, no delivery
// fee, no order type. Dropping them would be tidier and would also lose the
// guest's "piyozsiz" — which is the one line on the ticket that stops a dish
// coming back.
func posterComment(o Order) string {
	parts := make([]string, 0, len(o.Items)+3)
	switch o.Type {
	case "delivery":
		parts = append(parts, "Yetkazib berish")
	case "pickup":
		parts = append(parts, "Olib ketish")
	case "dinein":
		if o.TableNumber != "" {
			parts = append(parts, "Stol "+o.TableNumber)
		} else {
			parts = append(parts, "Zalda")
		}
	}
	if o.Number != "" {
		parts = append(parts, "Buyurtma "+o.Number)
	}
	for _, it := range o.Items {
		if c := strings.TrimSpace(it.Comment); c != "" {
			parts = append(parts, it.Name+": "+c)
		}
	}
	if c := strings.TrimSpace(o.Comment); c != "" {
		parts = append(parts, c)
	}
	if o.DeliveryFee > 0 {
		parts = append(parts, "Yetkazish narxi: "+strconv.Itoa(o.DeliveryFee)+" so'm")
	}
	return strings.Join(parts, " · ")
}

// posterPhone keeps digits and a leading +. Poster stores the number as given
// and matches guests by it, so a number carrying spaces and brackets creates a
// second client card for somebody who already has one.
func posterPhone(p string) string {
	var b strings.Builder
	for i, r := range p {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if r == '+' && i == 0 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// flexInt reads a number that Poster may send as a JSON number or as a string.
// Its API does both, in the same response, for the same field across versions —
// a plain int here fails at runtime on a field that worked yesterday.
type flexInt int

func (f *flexInt) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	// Prices arrive as "30000" but also, occasionally, as "300.00".
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	*f = flexInt(v)
	return nil
}
