package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Sales a till took while it had no network.
//
// ⚠️ **The connection is the thing that fails, not the restaurant.** A till on a
// restaurant's wifi loses the server for a minute several times a week and for
// an hour when the provider has a bad day — and during that hour the kitchen
// still cooks, the drawer still opens and the fiscal register, which is on the
// same local network, still registers the sale. What stops is us. So the till
// keeps its checks on its own disk and hands them over afterwards, and this is
// where they land.
//
// ⚠️ **Every rule here exists because the sender retries.** A program that
// cannot know whether its last attempt arrived will send again, and a second
// arrival must be the same sale rather than a second dinner.

type syncLine struct {
	MenuItemID string                   `json:"menuItemId"`
	Name       string                   `json:"name"`
	Price      int                      `json:"price"`
	Qty        int                      `json:"qty"`
	Options    []models.OrderItemOption `json:"options,omitempty"`
	Comment    string                   `json:"comment,omitempty"`
	Guest      int                      `json:"guest,omitempty"`
	Course     int                      `json:"course,omitempty"`
	FiredAt    *time.Time               `json:"firedAt,omitempty"`
}

type syncCheck struct {
	// The id the till minted before anybody saw this sale. Required: without it
	// there is no way to tell a retry from a second dinner.
	ClientID string `json:"clientId"`
	// The number printed on the guest's receipt. Kept, because the paper in
	// somebody's pocket says it.
	Number   string     `json:"number"`
	OpenedAt time.Time  `json:"openedAt"`
	ClosedAt *time.Time `json:"closedAt,omitempty"`

	TableID     string `json:"tableId,omitempty"`
	TableNumber string `json:"tableNumber,omitempty"`
	Guests      int    `json:"guests,omitempty"`
	ServerName  string `json:"serverName,omitempty"`
	ClosedBy    string `json:"closedBy,omitempty"`

	Lines []syncLine `json:"lines"`

	PaymentMethod  string `json:"paymentMethod,omitempty"`
	Discount       int    `json:"discount,omitempty"`
	DiscountReason string `json:"discountReason,omitempty"`

	// What the local register said, when there is one. ⚠️ Filed **offline**:
	// the register is on the restaurant's own network, so a sale rung up with
	// no internet is still registered — the one part of this that does not have
	// to wait for us.
	FiscalSign string `json:"fiscalSign,omitempty"`
	QRText     string `json:"qrText,omitempty"`
}

type syncRequest struct {
	Checks []syncCheck `json:"checks"`
}

type syncResult struct {
	ClientID string `json:"clientId"`
	ID       string `json:"id,omitempty"`
	Number   string `json:"number,omitempty"`
	// Empty when it landed. A sale that cannot be accepted says why, once, and
	// the till stops resending it — a queue that retries a malformed check
	// forever never delivers the good ones behind it.
	Error string `json:"error,omitempty"`
	// True when this sale was already here: the ordinary answer to a retry.
	Duplicate bool `json:"duplicate,omitempty"`
}

// StaffSyncChecks accepts sales a till took offline.
func (h *Handler) StaffSyncChecks(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermCashier)
	if !ok {
		return
	}
	var req syncRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// ⚠️ Bounded: a till that has been offline for a day sends its evening in
	// pieces rather than in one request that times out halfway through and is
	// retried whole.
	if len(req.Checks) > 50 {
		httpx.Error(w, http.StatusBadRequest, "bir so'rovda 50 tagacha chek")
		return
	}

	now := time.Now()
	out := make([]syncResult, 0, len(req.Checks))
	for _, c := range req.Checks {
		res := syncResult{ClientID: c.ClientID}
		id, number, dup, err := h.acceptOfflineCheck(r, s, c, now)
		switch {
		case err != nil:
			res.Error = err.Error()
		default:
			res.ID = id.Hex()
			res.Number = number
			res.Duplicate = dup
		}
		out = append(out, res)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"results": out,
		// ⚠️ The server's own clock, in every reply. An offline till has been
		// timestamping sales from a monoblock whose CMOS battery may be dead,
		// and this is how it finds out — see clampOfflineTime.
		"serverTime": now,
	})
}

// acceptOfflineCheck writes one sale, or recognises one already written.
func (h *Handler) acceptOfflineCheck(
	r *http.Request, s models.Staff, c syncCheck, now time.Time,
) (primitive.ObjectID, string, bool, error) {
	if strings.TrimSpace(c.ClientID) == "" {
		return primitive.NilObjectID, "", false, errors.New("clientId yo'q")
	}
	if len(c.Lines) == 0 {
		return primitive.NilObjectID, "", false, errors.New("chek bo'sh")
	}

	// Already here? The ordinary answer to a retry, and the reason nothing
	// below has to be idempotent by itself.
	var existing models.Order
	err := h.Store.Orders.FindOne(r.Context(),
		bson.M{"clientId": c.ClientID}).Decode(&existing)
	if err == nil {
		return existing.ID, existing.Number, true, nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return primitive.NilObjectID, "", false, err
	}

	o := models.Order{
		ClientID:    c.ClientID,
		BranchID:    s.BranchID,
		Type:        "dinein",
		Status:      models.StatusDelivered,
		TableID:     c.TableID,
		TableNumber: c.TableNumber,
		Customer:    models.OrderCustomer{Name: guestLabel(c.TableNumber)},
		// The same channel an online till sale carries, so every report that
		// splits by channel keeps one answer for "sold at the counter".
		Channel:     "pos",
	}
	opened := clampOfflineTime(c.OpenedAt, now)
	closed := opened
	if c.ClosedAt != nil {
		closed = clampOfflineTime(*c.ClosedAt, now)
	}
	if closed.Before(opened) {
		closed = opened
	}

	// ⚠️ **The prices are the till's, and this is the one place that is right.**
	// Everywhere else the tablet says which dish and the server says what it
	// costs — because the guest has not paid yet. Here they have: the money is
	// in the drawer and the receipt is in their pocket, and re-pricing an hour
	// later against a menu somebody has since edited would make the day's
	// takings disagree with the cash count.
	subtotal := 0
	for _, l := range c.Lines {
		if l.Qty < 1 || l.Price < 0 {
			return primitive.NilObjectID, "", false, errors.New("qator noto'g'ri")
		}
		item := models.OrderItem{
			Name:    clampText(l.Name, 120),
			Price:   l.Price,
			Qty:     l.Qty,
			Options: l.Options,
			Comment: clampText(l.Comment, 200),
			Guest:   l.Guest,
			Course:  l.Course,
			LineID:  lineID(),
		}
		if id, err := primitive.ObjectIDFromHex(l.MenuItemID); err == nil {
			item.MenuItemID = id
		}
		if l.FiredAt != nil {
			at := clampOfflineTime(*l.FiredAt, now)
			item.FiredAt = &at
		}
		subtotal += item.Price * item.Qty
		o.Items = append(o.Items, item)
	}

	discount := c.Discount
	if discount < 0 {
		discount = 0
	}
	if discount > subtotal {
		discount = subtotal
	}
	o.Subtotal = subtotal
	o.DiscountTotal = discount
	o.Total = subtotal - discount

	method := c.PaymentMethod
	if !tillMethods[method] {
		method = models.ProviderCash
	}
	o.PaymentMethod = method
	o.PaymentStatus = models.PayPaid
	o.CreatedAt = opened
	o.UpdatedAt = now
	// ⚠️ **Queued when the kitchen was told, not when we heard about it.** Every
	// report that asks "how long did this take" reads this, and stamping it now
	// would make an evening's cooking look instantaneous.
	o.QueuedAt = firstFired(o.Items, opened)
	o.StatusHistory = []models.StatusEvent{
		{Status: models.StatusConfirmed, At: opened},
		{Status: models.StatusDelivered, At: closed},
	}
	o.Check = &models.OrderCheck{
		OpenedAt:   opened,
		OpenedBy:   c.ServerName,
		ServerName: c.ServerName,
		Guests:     c.Guests,
		ClosedAt:   &closed,
		ClosedBy:   orDefault(c.ClosedBy, s.Name),
		// ⚠️ Marked as printed: the paper came out of the printer in the
		// restaurant hours ago. Queueing it now would print an evening's
		// receipts in one burst at whatever hour the connection returned.
		ReceiptAt: &closed,
	}
	if c.FiscalSign != "" {
		filed := closed
		o.Fiscal = &models.FiscalReceipt{
			Status:     models.FiscalFiled,
			FiscalSign: c.FiscalSign,
			QRText:     c.QRText,
			FiledAt:    &filed,
		}
	}
	// ⚠️ **The number is kept, unless somebody already has it.** The paper in
	// the guest's pocket says it, so it is not renamed for tidiness — but two
	// tills offline at the same time can mint the same one, and a receipt that
	// silently replaced another restaurant's sale would be discovered by a
	// missing sale, not by an error.
	o.Number = strings.TrimSpace(c.Number)
	if o.Number == "" {
		o.Number = orderNumber()
	} else {
		taken, err := h.Store.Orders.CountDocuments(r.Context(),
			bson.M{"number": o.Number})
		if err == nil && taken > 0 {
			o.Number = orderNumber()
		}
	}

	res, err := h.Store.Orders.InsertOne(r.Context(), o)
	if err != nil {
		// ⚠️ A duplicate key here is not a failure: two sends crossed, and the
		// first one won. The till is told the sale is here, which is what it
		// asked.
		if mongo.IsDuplicateKeyError(err) {
			var again models.Order
			if e := h.Store.Orders.FindOne(r.Context(),
				bson.M{"clientId": c.ClientID}).Decode(&again); e == nil {
				return again.ID, again.Number, true, nil
			}
		}
		return primitive.NilObjectID, "", false, err
	}
	return oidOf(res.InsertedID), o.Number, false, nil
}

// clampOfflineTime keeps a till's clock from writing history.
//
// ⚠️ **The monoblock's clock is the quietest thing that breaks here.** On an
// older machine the CMOS battery is dead, and a power cut sends the date back to
// 2010 — the till then stamps an evening of sales into a year the restaurant did
// not exist, and no screen anywhere says so. It reaches the reports, the shift
// count and, on a fiscal receipt, a tax document.
//
// So a time from a till is trusted only inside a window that could be true: not
// in the future (nothing is sold tomorrow) and not older than a fortnight (a
// till that has been offline for longer has a bigger problem than its clock).
// Anything outside becomes the moment we heard about it — visibly wrong-ish
// rather than invisibly wrong.
func clampOfflineTime(t, now time.Time) time.Time {
	if t.IsZero() {
		return now
	}
	if t.After(now.Add(2 * time.Minute)) {
		return now
	}
	if t.Before(now.AddDate(0, 0, -14)) {
		return now
	}
	return t
}

// firstFired is when the kitchen was first told about this table.
func firstFired(items []models.OrderItem, fallback time.Time) *time.Time {
	var first *time.Time
	for i := range items {
		at := items[i].FiredAt
		if at == nil {
			continue
		}
		if first == nil || at.Before(*first) {
			first = at
		}
	}
	if first == nil {
		return &fallback
	}
	return first
}

func orDefault(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
