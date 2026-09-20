package delivery

// BTS Express (bts.uz) — the post an online store's parcel leaves on.
//
// ⚠️ **Read docs/vendor/bts-express.md before changing anything here.** BTS
// publishes no API documentation: `api.bts.uz` exists and answers every path
// with the same empty `noindex` page, the cabinet at `new.bts.uz` is a
// server-rendered Yii2 form, and there is nothing on the open web, on GitHub or
// in any package registry. The contract comes with the customer's own agreement
// with BTS, and this file is written to be corrected against it.
//
// Two decisions follow from that, and both are about failing loudly:
//
//   - **No production host is compiled in.** Yandex's is published
//     (`b2b.taxi.yandex.net`); guessing BTS's would be the worst kind of wrong —
//     the form looks filled in, the save succeeds, nothing errors, and the first
//     parcel simply never goes anywhere. `BaseURL` is required, and the panel
//     says so by name.
//   - **The wire format is in one file, behind `Service`.** When the real
//     contract arrives the correction is this file and nothing else; the order
//     screens never see a BTS type.
//
// ⚠️ **There is no draft step.** Yandex creates a claim and dispatches nobody
// until `accept`; a parcel handed to a post office is handed over once. So
// `Accept` returns what `Create` already produced rather than calling again —
// calling again is how one order becomes two waybills.

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

// BTS files parcels with BTS Express.
type BTS struct {
	BaseURL string
	Token   string
	// The service the shop has contracted for — "express", "standard", and
	// whatever else their agreement names. Passed through rather than
	// validated: the list is theirs and it is not published.
	Tariff string
	HTTP   *http.Client
}

// NewBTS builds a client.
//
// ⚠️ **No fallback base URL**, unlike NewYandex. An empty one is kept empty so
// that the first call fails with a sentence somebody can act on, rather than
// posting a customer's parcel details at a hostname we invented.
func NewBTS(base, token, tariff string) *BTS {
	return &BTS{
		BaseURL: strings.TrimRight(strings.TrimSpace(base), "/"),
		Token:   strings.TrimSpace(token),
		Tariff:  strings.TrimSpace(tariff),
		HTTP:    &http.Client{Timeout: 20 * time.Second},
	}
}

// ---- wire format ----
//
// ⚠️ **Unverified.** Every field below is the ordinary shape of a courier API
// and none of it is confirmed against BTS's own document — see the vendor note.
// Kept in one place so the correction is a diff rather than an investigation.

type btsPoint struct {
	Address string  `json:"address"`
	Lat     float64 `json:"lat,omitempty"`
	Lng     float64 `json:"lng,omitempty"`
	Name    string  `json:"name,omitempty"`
	Phone   string  `json:"phone,omitempty"`
	Comment string  `json:"comment,omitempty"`
}

type btsCreateRequest struct {
	// Our own order number, so a parcel can be traced from either side.
	ExternalID string   `json:"external_id"`
	Sender     btsPoint `json:"sender"`
	Receiver   btsPoint `json:"receiver"`
	// What is in the parcel, in words.
	Description string `json:"description,omitempty"`
	// Declared value, in so'm. ⚠️ A string for the same reason Yandex takes
	// one: this number is money and JSON numbers are float64, which is the one
	// type money must never pass through.
	DeclaredValue string `json:"declared_value,omitempty"`
	Tariff        string `json:"tariff,omitempty"`
}

type btsOrderResponse struct {
	// Their id for the parcel, and the number printed on the waybill. Often
	// the same string; kept apart because the tracking page takes one of them
	// and we do not know which until the contract says.
	ID       string `json:"id"`
	OrderID  string `json:"order_id"`
	Waybill  string `json:"waybill"`
	Status   string `json:"status"`
	Price    any    `json:"price"`
	TrackURL string `json:"track_url"`
	Courier  *struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
	} `json:"courier"`
}

func (r *btsOrderResponse) toClaim() Claim {
	c := Claim{
		ID:       firstNonEmpty(r.ID, r.OrderID, r.Waybill),
		Status:   r.Status,
		TrackURL: r.TrackURL,
	}
	// ⚠️ **`any`, then rendered.** Half of these APIs send a price as a number
	// and half as a string, and a typed field would fail the whole decode on
	// the half that guessed wrong — losing the waybill number along with it,
	// which is the one field that cannot be recovered afterwards.
	switch v := r.Price.(type) {
	case string:
		c.Price = v
	case float64:
		c.Price = fmt.Sprintf("%.0f", v)
	}
	if r.Courier != nil {
		c.CourierName = r.Courier.Name
		c.CourierPhone = r.Courier.Phone
	}
	return c
}

// ---- transport ----

func (c *BTS) do(ctx context.Context, method, path string, body any, out any) error {
	if c.BaseURL == "" {
		// ⚠️ Named, because this is the one setting an owner cannot guess and
		// the one whose absence is otherwise silent — see the note at the top.
		return errors.New("BTS uchun API manzili (base URL) kiritilmagan")
	}
	if c.Token == "" {
		return errors.New("API token sozlanmagan")
	}
	var rd io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, rd)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	// ⚠️ **The same token twice, under two names.** Which header BTS reads is
	// exactly the kind of detail the missing document would settle; an ignored
	// header costs nothing, and a missing one costs a parcel. Removed the day
	// the contract says which.
	req.Header.Set("X-API-KEY", c.Token)

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
			Error   string `json:"error"`
			Detail  string `json:"detail"`
		}
		_ = json.Unmarshal(raw, &e)
		msg := firstNonEmpty(e.Message, e.Error, e.Detail)
		if msg == "" {
			// ⚠️ **Their own body, trimmed**, when it is not JSON we recognise.
			// "HTTP 422" tells a shop nothing; the sentence underneath it is
			// usually the missing field — and with no published error list,
			// their words are all anybody has to go on.
			msg = strings.TrimSpace(string(raw))
			if len(msg) > 300 {
				msg = msg[:300]
			}
		}
		return &APIError{Status: res.StatusCode, Code: e.Code, Message: msg}
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// ---- operations ----

// Create files the parcel.
func (c *BTS) Create(ctx context.Context, o Order, requestID string) (Claim, error) {
	req := btsCreateRequest{
		ExternalID: firstNonEmpty(o.Number, requestID),
		Sender: btsPoint{
			Address: o.Pickup.Address, Lat: o.Pickup.Lat, Lng: o.Pickup.Lng,
			Name: o.Pickup.Name, Phone: o.Pickup.Phone,
		},
		Receiver: btsPoint{
			Address: o.Destination.Address, Lat: o.Destination.Lat, Lng: o.Destination.Lng,
			Name: o.Destination.Name, Phone: o.Destination.Phone,
			Comment: o.Destination.Comment,
		},
		Description:   firstNonEmpty(o.Contents, "Onlayn do'kon buyurtmasi"),
		DeclaredValue: o.Price,
		Tariff:        c.Tariff,
	}
	var out btsOrderResponse
	if err := c.do(ctx, http.MethodPost, "/orders", req, &out); err != nil {
		return Claim{}, err
	}
	claim := out.toClaim()
	if claim.ID == "" {
		// ⚠️ A 200 with no id is worse than an error: the parcel may well have
		// been filed, and an order saved without the number can neither be
		// tracked nor cancelled. Said out loud so somebody checks the cabinet.
		return Claim{}, errors.New("BTS javobida buyurtma raqami yo'q — kabinetdan tekshiring")
	}
	return claim, nil
}

// Accept confirms nothing: a parcel handed over is handed over once.
//
// ⚠️ **Returns rather than calls.** The draft step is Yandex's, and posting
// `/orders` a second time here is how one order becomes two waybills — with the
// shop charged for both and only one of them on the order.
func (c *BTS) Accept(_ context.Context, claimID string, _ int) (Claim, error) {
	return Claim{ID: claimID}, nil
}

// Info reads the parcel's current state.
func (c *BTS) Info(ctx context.Context, claimID string) (Claim, error) {
	var out btsOrderResponse
	if err := c.do(ctx, http.MethodGet, "/orders/"+claimID, nil, &out); err != nil {
		return Claim{}, err
	}
	claim := out.toClaim()
	// ⚠️ The id we asked about, kept whatever comes back. A response that
	// omits it would otherwise blank the waybill on the order — losing the only
	// string that can find the parcel again.
	claim.ID = claimID
	return claim, nil
}

// Cancel withdraws the parcel.
//
// ⚠️ `paid` is passed on rather than decided here: whether a late cancellation
// is charged for is BTS's rule, and a client that assumed either answer would be
// telling the shop something we do not know.
func (c *BTS) Cancel(ctx context.Context, claimID string, _ int, paid bool) error {
	return c.do(ctx, http.MethodPost, "/orders/"+claimID+"/cancel",
		map[string]any{"paid": paid}, nil)
}
