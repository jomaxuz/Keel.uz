// Package delivery talks to outside delivery services that publish an API.
//
// Today that is Yandex Delivery (Yandex Go's B2B "cargo claims" API). The
// client is written against their documented contract:
//
//	POST {base}/b2b/cargo/integration/v2/claims/create?request_id=<uuid>
//	POST {base}/b2b/cargo/integration/v2/claims/accept?claim_id=<id>
//	POST {base}/b2b/cargo/integration/v2/claims/info?claim_id=<id>
//	POST {base}/b2b/cargo/integration/v2/claims/cancel?claim_id=<id>
//
// with `Authorization: Bearer <oauth token>` from the restaurant's Yandex
// Delivery dashboard.
//
// Two things to know before switching a restaurant on:
//   - the token is per-restaurant and comes from their own contract with the
//     service; there is no way to test against production without it,
//   - BaseURL is configurable precisely so it can be pointed at a sandbox (or
//     at the mock used by our tests) and so a future endpoint change can be
//     absorbed without a redeploy.
package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// YandexBase is the production host for the B2B cargo API.
const YandexBase = "https://b2b.taxi.yandex.net"

// Client files delivery requests with Yandex Delivery.
type Client struct {
	BaseURL string
	Token   string
	Tariff  string // "express" | "courier" | "cargo"; empty = "express"
	HTTP    *http.Client
}

// NewYandex builds a client; an empty base falls back to production.
func NewYandex(base, token, tariff string) *Client {
	if strings.TrimSpace(base) == "" {
		base = YandexBase
	}
	return &Client{
		BaseURL: strings.TrimRight(base, "/"),
		Token:   token,
		Tariff:  tariff,
		HTTP:    &http.Client{Timeout: 20 * time.Second},
	}
}

// Point is one end of the route.
type Point struct {
	Address string
	Lat     float64
	Lng     float64
	Name    string
	Phone   string
	Comment string
}

// Order is what we hand over: where to pick up, where to drop off, what it is.
type Order struct {
	Number      string
	Pickup      Point
	Destination Point
	// Human-readable contents ("Osh × 2, Lag'mon × 1").
	Contents string
	// Declared value in the smallest currency unit is not used by the API; the
	// price is sent as a decimal string.
	Price string
}

// Claim is the provider's view of a delivery request.
type Claim struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Version      int    `json:"version"`
	Price        string `json:"price"`
	CourierName  string `json:"courier_name"`
	CourierPhone string `json:"courier_phone"`
	TrackURL     string `json:"track_url"`
}

// ---- wire format ----

type ydxContact struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type ydxAddress struct {
	Fullname    string     `json:"fullname"`
	Coordinates [2]float64 `json:"coordinates"` // [lon, lat]
	Comment     string     `json:"comment,omitempty"`
}

type ydxRoutePoint struct {
	PointID       int        `json:"point_id"`
	VisitOrder    int        `json:"visit_order"`
	Type          string     `json:"type"` // "source" | "destination"
	Address       ydxAddress `json:"address"`
	Contact       ydxContact `json:"contact"`
	ExternalOrder string     `json:"external_order_id,omitempty"`
}

type ydxItemSize struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type ydxItem struct {
	Title     string      `json:"title"`
	Quantity  int         `json:"quantity"`
	CostValue string      `json:"cost_value"`
	CostCur   string      `json:"cost_currency"`
	Weight    float64     `json:"weight"`
	Size      ydxItemSize `json:"size"`
	PickupID  int         `json:"pickup_point"`
	DropoffID int         `json:"droppof_point"`
}

type ydxCreateRequest struct {
	Items              []ydxItem       `json:"items"`
	RoutePoints        []ydxRoutePoint `json:"route_points"`
	EmergencyContact   ydxContact      `json:"emergency_contact"`
	Comment            string          `json:"comment,omitempty"`
	ClientRequirements *struct {
		TaxiClass string `json:"taxi_class"`
	} `json:"client_requirements,omitempty"`
}

type ydxClaimResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Version int    `json:"version"`
	Pricing *struct {
		Offer *struct {
			Price string `json:"price"`
		} `json:"offer"`
	} `json:"pricing"`
	Performer *struct {
		CourierName string `json:"courier_name"`
		Phone       string `json:"phone"`
	} `json:"performer_info"`
	RoutePoints []struct {
		Type string `json:"type"`
	} `json:"route_points"`
	SharingLink string `json:"sharing_link"`
}

func (r *ydxClaimResponse) toClaim() Claim {
	c := Claim{ID: r.ID, Status: r.Status, Version: r.Version, TrackURL: r.SharingLink}
	if r.Pricing != nil && r.Pricing.Offer != nil {
		c.Price = r.Pricing.Offer.Price
	}
	if r.Performer != nil {
		c.CourierName = r.Performer.CourierName
		c.CourierPhone = r.Performer.Phone
	}
	return c
}

// ---- transport ----

// APIError carries the provider's own message so the panel can show something
// actionable instead of "request failed".
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("yandex delivery: HTTP %d", e.Status)
}

func (c *Client) post(ctx context.Context, path string, query string, body any, out any) error {
	if strings.TrimSpace(c.Token) == "" {
		return errors.New("API token sozlanmagan")
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := c.BaseURL + path
	if query != "" {
		url += "?" + query
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept-Language", "ru")

	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))

	if res.StatusCode >= 400 {
		var e struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(raw, &e)
		return &APIError{Status: res.StatusCode, Code: e.Code, Message: e.Message}
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// ---- operations ----

// Create files a draft claim. requestID must be stable per order so a retry
// after a timeout does not create a second delivery.
func (c *Client) Create(ctx context.Context, o Order, requestID string) (Claim, error) {
	tariff := c.Tariff
	if tariff == "" {
		tariff = "express"
	}
	req := ydxCreateRequest{
		Items: []ydxItem{{
			Title:     firstNonEmpty(o.Contents, "Restoran buyurtmasi"),
			Quantity:  1,
			CostValue: firstNonEmpty(o.Price, "0"),
			CostCur:   "UZS",
			Weight:    3,
			Size:      ydxItemSize{Length: 0.3, Width: 0.3, Height: 0.2},
			PickupID:  1,
			DropoffID: 2,
		}},
		RoutePoints: []ydxRoutePoint{
			{
				PointID: 1, VisitOrder: 1, Type: "source",
				Address: ydxAddress{
					Fullname:    o.Pickup.Address,
					Coordinates: [2]float64{o.Pickup.Lng, o.Pickup.Lat},
				},
				Contact:       ydxContact{Name: o.Pickup.Name, Phone: o.Pickup.Phone},
				ExternalOrder: o.Number,
			},
			{
				PointID: 2, VisitOrder: 2, Type: "destination",
				Address: ydxAddress{
					Fullname:    o.Destination.Address,
					Coordinates: [2]float64{o.Destination.Lng, o.Destination.Lat},
					Comment:     o.Destination.Comment,
				},
				Contact:       ydxContact{Name: o.Destination.Name, Phone: o.Destination.Phone},
				ExternalOrder: o.Number,
			},
		},
		EmergencyContact: ydxContact{Name: o.Pickup.Name, Phone: o.Pickup.Phone},
		Comment:          o.Contents,
	}
	req.ClientRequirements = &struct {
		TaxiClass string `json:"taxi_class"`
	}{TaxiClass: tariff}

	var out ydxClaimResponse
	if err := c.post(ctx, "/b2b/cargo/integration/v2/claims/create",
		"request_id="+requestID, req, &out); err != nil {
		return Claim{}, err
	}
	return out.toClaim(), nil
}

// Accept confirms a draft claim — until this call the courier is not sent.
func (c *Client) Accept(ctx context.Context, claimID string, version int) (Claim, error) {
	var out ydxClaimResponse
	body := map[string]any{"version": version}
	if err := c.post(ctx, "/b2b/cargo/integration/v2/claims/accept",
		"claim_id="+claimID, body, &out); err != nil {
		return Claim{}, err
	}
	return out.toClaim(), nil
}

// Info reads the current state: status, price, and the courier once assigned.
func (c *Client) Info(ctx context.Context, claimID string) (Claim, error) {
	var out ydxClaimResponse
	if err := c.post(ctx, "/b2b/cargo/integration/v2/claims/info",
		"claim_id="+claimID, map[string]any{}, &out); err != nil {
		return Claim{}, err
	}
	return out.toClaim(), nil
}

// Cancel drops the claim. "free" is only possible before a courier is on the
// way; afterwards the provider may charge, hence the explicit state.
func (c *Client) Cancel(ctx context.Context, claimID string, version int, paid bool) error {
	state := "free"
	if paid {
		state = "paid"
	}
	body := map[string]any{"version": version, "cancel_state": state}
	return c.post(ctx, "/b2b/cargo/integration/v2/claims/cancel",
		"claim_id="+claimID, body, nil)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
