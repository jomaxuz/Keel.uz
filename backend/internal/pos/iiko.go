package pos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// iikoCloud — api-ru.iiko.services.
//
// Everything is POST-with-JSON, including the reads. The access token is
// exchanged for an apiLogin and lives an hour, so it is cached rather than
// fetched per call — the mapping screen alone would otherwise burn a token per
// dropdown.
//
// The one genuinely awkward part is that creating a delivery is **asynchronous**:
// `deliveries/create` answers with a correlationId and an order in state
// `InProgress`, and whether the till actually took it is only known from a
// later `commands/status`. An integration that treats the first 200 as success
// reports "sent to the kitchen" for orders the kitchen never saw. So the
// adapter waits, briefly, for the till to make up its mind.
type iikoClient struct {
	cfg  Config
	http *http.Client
	base string
	// Which of the two this is. Syrve is iiko's international edition — the
	// same API on a different host — so one adapter serves both. The name is
	// carried rather than hard-coded because it goes into every error the
	// owner reads: somebody who bought Syrve and is told "iiko: apiLogin
	// qabul qilinmadi" has no idea the message is about their till.
	name string

	mu      sync.Mutex
	token   string
	expires time.Time
}

func newIiko(cfg Config, client *http.Client) *iikoClient {
	base := strings.TrimRight(cfg.IikoBaseURL, "/")
	name := IIKO
	if cfg.Provider == Syrve {
		name = Syrve
		if base == "" {
			base = "https://api-eu.syrve.live"
		}
	}
	if base == "" {
		base = "https://api-ru.iiko.services"
	}
	return &iikoClient{cfg: cfg, http: client, base: base, name: name}
}

func (c *iikoClient) Name() string { return c.name }

// auth returns a live token, fetching one only when the cached one is gone.
// iiko's tokens last an hour; the margin keeps a call that starts at 59:59 from
// arriving expired.
func (c *iikoClient) auth(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expires) {
		return c.token, nil
	}
	body, _ := json.Marshal(map[string]string{"apiLogin": c.cfg.IikoAPILogin})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.base+"/api/1/access_token", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: ulanib bo'lmadi: %w", c.name, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var out struct {
		Token            string `json:"token"`
		ErrorDescription string `json:"errorDescription"`
	}
	_ = json.Unmarshal(raw, &out)
	if out.Token == "" {
		return "", fmt.Errorf("%s: apiLogin qabul qilinmadi (%s)", c.name,
			firstNonEmpty(out.ErrorDescription, res.Status))
	}
	c.token, c.expires = out.Token, time.Now().Add(50*time.Minute)
	return c.token, nil
}

// call posts a JSON body to an iiko endpoint and decodes the answer.
func (c *iikoClient) call(ctx context.Context, path string, in, out any) error {
	token, err := c.auth(ctx)
	if err != nil {
		return err
	}
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path,
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", c.name, path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode >= 400 {
		// iiko puts a readable reason in the body; the status alone ("400 Bad
		// Request") tells the owner nothing they can act on.
		var e struct {
			ErrorDescription string `json:"errorDescription"`
			Error            string `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		return fmt.Errorf("%s %s: %s", c.name, path,
			firstNonEmpty(e.ErrorDescription, e.Error, res.Status))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func (c *iikoClient) Ping(ctx context.Context) (string, error) {
	var orgs struct {
		Organizations []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"organizations"`
	}
	if err := c.call(ctx, "/api/1/organizations", map[string]any{
		"organizationIds": []string{c.cfg.IikoOrganizationID},
	}, &orgs); err != nil {
		return "", err
	}
	if len(orgs.Organizations) == 0 {
		return "", fmt.Errorf("%s: bu apiLogin uchun tashkilot topilmadi — organizationId ni tekshiring", c.name)
	}
	name := orgs.Organizations[0].Name

	// The terminal group is what actually prints. A configuration that names a
	// dead terminal connects fine and then silently swallows orders, so the
	// check says whether it is alive rather than only that it exists.
	if c.cfg.IikoTerminalGroup != "" {
		var alive struct {
			IsAliveStatus []struct {
				TerminalGroupID string `json:"terminalGroupId"`
				IsAlive         bool   `json:"isAlive"`
			} `json:"isAliveStatus"`
		}
		if err := c.call(ctx, "/api/1/terminal_groups/is_alive", map[string]any{
			"organizationIds":  []string{c.cfg.IikoOrganizationID},
			"terminalGroupIds": []string{c.cfg.IikoTerminalGroup},
		}, &alive); err == nil && len(alive.IsAliveStatus) > 0 {
			if !alive.IsAliveStatus[0].IsAlive {
				return name, fmt.Errorf("%s: %s — terminal guruhi javob bermayapti (kassa o'chiqmi?)", c.name, name)
			}
			return name + " · terminal ishlayapti", nil
		}
	}
	return name, nil
}

func (c *iikoClient) Products(ctx context.Context) ([]Product, error) {
	var menu struct {
		Groups []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"groups"`
		Products []struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			ParentGroup string  `json:"parentGroup"`
			GroupID     string  `json:"groupId"`
			Price       float64 `json:"price"`
			IsDeleted   bool    `json:"isDeleted"`
			Type        string  `json:"type"`
			SizePrices  []struct {
				Price struct {
					CurrentPrice float64 `json:"currentPrice"`
					IsIncludedIn bool    `json:"isIncludedInMenu"`
				} `json:"price"`
			} `json:"sizePrices"`
		} `json:"products"`
	}
	if err := c.call(ctx, "/api/1/nomenclature", map[string]any{
		"organizationId": c.cfg.IikoOrganizationID,
	}, &menu); err != nil {
		return nil, err
	}
	groups := map[string]string{}
	for _, g := range menu.Groups {
		groups[g.ID] = g.Name
	}

	// What the till has stopped today. Shown on the mapping screen, because
	// mapping a dish to something already stopped is the mistake worth
	// catching there rather than on a live order.
	stopped := map[string]bool{}
	var stop struct {
		TerminalGroupStopLists []struct {
			Items []struct {
				ProductID string `json:"productId"`
			} `json:"items"`
		} `json:"terminalGroupStopLists"`
	}
	if err := c.call(ctx, "/api/1/stop_lists", map[string]any{
		"organizationIds": []string{c.cfg.IikoOrganizationID},
	}, &stop); err == nil {
		for _, tg := range stop.TerminalGroupStopLists {
			for _, it := range tg.Items {
				stopped[it.ProductID] = true
			}
		}
	}

	out := make([]Product, 0, len(menu.Products))
	for _, p := range menu.Products {
		if p.IsDeleted {
			continue
		}
		// "Modifier" rows are not orderable on their own.
		if p.Type != "" && !strings.EqualFold(p.Type, "Dish") &&
			!strings.EqualFold(p.Type, "Good") && !strings.EqualFold(p.Type, "Product") {
			continue
		}
		price := p.Price
		if price == 0 && len(p.SizePrices) > 0 {
			price = p.SizePrices[0].Price.CurrentPrice
		}
		group := p.ParentGroup
		if group == "" {
			group = p.GroupID
		}
		out = append(out, Product{
			ID:          p.ID,
			Name:        p.Name,
			Category:    groups[group],
			Price:       int(price),
			Unavailable: stopped[p.ID],
		})
	}
	return out, nil
}

func (c *iikoClient) SendOrder(ctx context.Context, o Order) (Result, error) {
	if err := CheckMapped(o.Items); err != nil {
		return Result{}, err
	}

	items := make([]map[string]any, 0, len(o.Items))
	for _, it := range o.Items {
		line := map[string]any{
			"productId": it.POSID,
			"type":      "Product",
			"amount":    it.Qty,
			"price":     it.Price,
		}
		if it.Comment != "" {
			line["comment"] = it.Comment
		}
		if len(it.Modifiers) > 0 {
			mods := make([]map[string]any, 0, len(it.Modifiers))
			for _, m := range it.Modifiers {
				mods = append(mods, map[string]any{
					"productId": m.POSID, "amount": m.Qty, "price": m.Price,
				})
			}
			line["modifiers"] = mods
		}
		items = append(items, line)
	}

	order := map[string]any{
		"externalNumber": o.Number,
		"phone":          iikoPhone(o.Phone),
		"comment":        o.Comment,
		"customer":       map[string]any{"name": o.CustomerName, "type": "one-time"},
		"items":          items,
	}
	// Courier delivery or the guest coming to fetch it. iiko draws the line
	// here and files the order differently for each.
	if o.Type == "delivery" {
		order["orderServiceType"] = "DeliveryByCourier"
		point := map[string]any{
			"address": map[string]any{
				"street": map[string]any{"name": o.Address},
				"house":  "-",
			},
		}
		if o.Lat != 0 || o.Lng != 0 {
			point["coordinates"] = map[string]any{
				"latitude": o.Lat, "longitude": o.Lng,
			}
		}
		if o.Address != "" {
			point["comment"] = o.Address
		}
		order["deliveryPoint"] = point
	} else {
		order["orderServiceType"] = "DeliveryByClient"
	}
	if c.cfg.IikoOrderTypeID != "" {
		order["orderTypeId"] = c.cfg.IikoOrderTypeID
	}
	// Money already taken is declared as processed elsewhere, so the till does
	// not ask the courier to collect it again.
	if o.Paid && c.cfg.IikoPaymentTypeID != "" {
		order["payments"] = []map[string]any{{
			"paymentTypeKind":       "Card",
			"sum":                   o.Total,
			"paymentTypeId":         c.cfg.IikoPaymentTypeID,
			"isProcessedExternally": true,
		}}
	}

	body := map[string]any{
		"organizationId": c.cfg.IikoOrganizationID,
		"order":          order,
	}
	if c.cfg.IikoTerminalGroup != "" {
		body["terminalGroupId"] = c.cfg.IikoTerminalGroup
	}

	var created struct {
		CorrelationID string `json:"correlationId"`
		OrderInfo     struct {
			ID             string `json:"id"`
			OrderNumber    string `json:"posOrderNumber"`
			CreationStatus string `json:"creationStatus"`
			ErrorInfo      *struct {
				Message     string `json:"message"`
				Description string `json:"description"`
			} `json:"errorInfo"`
		} `json:"orderInfo"`
	}
	if err := c.call(ctx, "/api/1/deliveries/create", body, &created); err != nil {
		return Result{}, err
	}
	if created.OrderInfo.ErrorInfo != nil {
		return Result{}, fmt.Errorf("%s: %s", c.name,
			firstNonEmpty(created.OrderInfo.ErrorInfo.Message,
				created.OrderInfo.ErrorInfo.Description))
	}

	res := Result{POSOrderID: created.OrderInfo.ID}
	// Creation is asynchronous. "InProgress" means the till has not accepted it
	// yet, and reporting that as sent is how an order goes missing between two
	// systems that both think the other has it.
	if strings.EqualFold(created.OrderInfo.CreationStatus, "InProgress") ||
		created.OrderInfo.CreationStatus == "" {
		if err := c.awaitCommand(ctx, created.CorrelationID); err != nil {
			return res, err
		}
	}
	if created.OrderInfo.OrderNumber != "" {
		res.Note = c.name + " №" + created.OrderInfo.OrderNumber
	}
	return res, nil
}

// awaitCommand polls until the till says yes or no. Bounded on purpose: the
// operator pressed a button and is waiting, so a till that has not answered in
// a few seconds is reported as "sent, not confirmed" rather than hanging the
// panel. The order carries the id either way, so it can be re-checked.
func (c *iikoClient) awaitCommand(ctx context.Context, correlationID string) error {
	if correlationID == "" {
		return nil
	}
	deadline := time.Now().Add(12 * time.Second)
	for attempt := 0; time.Now().Before(deadline); attempt++ {
		var st struct {
			State string `json:"state"`
			Error *struct {
				Message     string `json:"message"`
				Description string `json:"description"`
			} `json:"error"`
		}
		if err := c.call(ctx, "/api/1/commands/status", map[string]any{
			"organizationId": c.cfg.IikoOrganizationID,
			"correlationId":  correlationID,
		}, &st); err != nil {
			return err
		}
		switch strings.ToLower(st.State) {
		case "success":
			return nil
		case "error":
			msg := c.name + " buyurtmani qabul qilmadi"
			if st.Error != nil {
				msg = firstNonEmpty(st.Error.Message, st.Error.Description, msg)
			}
			return fmt.Errorf("%s", msg)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(500+attempt*250) * time.Millisecond):
		}
	}
	return fmt.Errorf("%s: buyurtma yuborildi, lekin kassa hali tasdiqlamadi — holatini tekshiring", c.name)
}

func (c *iikoClient) OrderStatus(ctx context.Context, posOrderID string) (Status, error) {
	if posOrderID == "" {
		return Status{}, ErrUnsupported
	}
	var out struct {
		Orders []struct {
			ID    string `json:"id"`
			Order struct {
				Status         string `json:"status"`
				DeliveryStatus string `json:"deliveryStatus"`
			} `json:"order"`
		} `json:"orders"`
	}
	if err := c.call(ctx, "/api/1/deliveries/by_id", map[string]any{
		"organizationId": c.cfg.IikoOrganizationID,
		"orderIds":       []string{posOrderID},
	}, &out); err != nil {
		return Status{}, err
	}
	if len(out.Orders) == 0 {
		return Status{State: "unknown"}, nil
	}
	raw := firstNonEmpty(out.Orders[0].Order.Status, out.Orders[0].Order.DeliveryStatus)
	return Status{State: iikoState(raw), Raw: raw}, nil
}

// iikoState folds iiko's pipeline onto the coarse set the panel shows.
func iikoState(s string) string {
	switch strings.ToLower(s) {
	case "unconfirmed", "waitcooking", "new":
		return "accepted"
	case "readyforcooking", "cookingstarted":
		return "cooking"
	case "cookingcompleted", "waiting", "onway":
		return "ready"
	case "delivered", "closed":
		return "closed"
	case "cancelled":
		return "cancelled"
	}
	return "unknown"
}

func (c *iikoClient) Cancel(ctx context.Context, posOrderID string) error {
	if posOrderID == "" {
		return ErrUnsupported
	}
	return c.call(ctx, "/api/1/deliveries/cancel", map[string]any{
		"organizationId": c.cfg.IikoOrganizationID,
		"orderId":        posOrderID,
	}, nil)
}

// iikoPhone puts a number into the +998… shape iiko expects.
func iikoPhone(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "+") {
		return p
	}
	return "+" + p
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
