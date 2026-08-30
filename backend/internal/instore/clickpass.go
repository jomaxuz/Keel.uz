package instore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CLICK Pass — docs/vendor/click-pass.md (docs.click.uz, read 2026-08-30).
//
// The flow is one request: the cashier scans, we post the code and the amount,
// and the answer says whether the card was charged. What makes it more than one
// request is CLICK's **confirm mode**, and that is worth reading before
// changing anything here.

// clickHost is the only host CLICK Pass answers on. No sandbox is published, so
// unlike every other integration in this codebase the base URL is not
// overridable: a box the owner can type into, whose only correct value is the
// default, is a box that eventually holds something else.
const clickHost = "https://api.click.uz"

type clickPass struct {
	cfg  Config
	http *http.Client
	now  func() time.Time
}

func newClickPass(cfg Config, client *http.Client) (Charger, error) {
	if cfg.ServiceID == "" || cfg.UserID == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("CLICK Pass: servis id, foydalanuvchi id va maxfiy kalit kerak")
	}
	return &clickPass{cfg: cfg, http: client, now: time.Now}, nil
}

// serviceID as CLICK wants it — a number, not a string.
func (c *clickPass) serviceID() int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(c.cfg.ServiceID), 10, 64)
	return n
}

func (c *clickPass) auth() string {
	stamp := secStamp(c.now())
	return c.cfg.UserID + ":" + sign(stamp, c.cfg.SecretKey) + ":" + stamp
}

// clickReply is the envelope every CLICK Pass call answers with.
//
// ⚠️ `payment_id` is a bigint in the successful case and **absent** in most
// failures, so it is decoded as a number and rendered back to a string once,
// here. Reading it as a string elsewhere would work until the first payment id
// long enough to be printed in scientific notation.
type clickReply struct {
	ErrorCode      int    `json:"error_code"`
	ErrorNote      string `json:"error_note"`
	PaymentID      int64  `json:"payment_id"`
	PaymentStatus  int    `json:"payment_status"`
	ConfirmMode    int    `json:"confirm_mode"`
	CardType       string `json:"card_type"`
	ProcessingType string `json:"processing_type"`
	CardNumber     string `json:"card_number"`
	PhoneNumber    string `json:"phone_number"`
}

func (c *clickPass) call(
	ctx context.Context, method, path string, body any,
) (clickReply, error) {
	var buf io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return clickReply{}, err
		}
		buf = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, clickHost+path, buf)
	if err != nil {
		return clickReply{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth", c.auth())

	res, err := c.http.Do(req)
	if err != nil {
		return clickReply{}, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return clickReply{}, err
	}
	var out clickReply
	if err := json.Unmarshal(raw, &out); err != nil {
		// ⚠️ The HTTP status is checked **only here**, in the branch where the
		// body did not parse. CLICK signals refusals inside a 200 body, so a
		// status check placed first would turn every ordinary decline into
		// "server error" and hide the sentence the cashier needs to read.
		if res.StatusCode >= 400 {
			return clickReply{}, fmt.Errorf("CLICK Pass: %s", http.StatusText(res.StatusCode))
		}
		return clickReply{}, fmt.Errorf("CLICK Pass: javob o'qilmadi")
	}
	return out, nil
}

// clickStatusOf maps CLICK's payment_status onto ours.
//
//	< 0  error   0  created   1  processing   2  paid
//
// ⚠️ **`0` and `1` are not success.** They are the states a payment sits in
// while the processing centre is still deciding, and a till that reads either
// as "paid" hands over food on a card that may yet decline. See Charge, which
// only ever reports paid on 2 — or on a confirmed payment, which CLICK answers
// for separately.
func clickStatusOf(r clickReply) string {
	switch {
	case r.ErrorCode < 0, r.PaymentStatus < 0:
		return StatusFailed
	case r.PaymentStatus == 2:
		return StatusPaid
	default:
		return StatusPending
	}
}

func clickResult(r clickReply) Result {
	return Result{
		PaymentID:    strconv.FormatInt(r.PaymentID, 10),
		Status:       clickStatusOf(r),
		CardMask:     r.CardNumber,
		Processing:   r.ProcessingType,
		Phone:        r.PhoneNumber,
		NeedsConfirm: r.ConfirmMode == 1,
	}
}

func (c *clickPass) Charge(ctx context.Context, ch Charge) (Result, error) {
	reply, err := c.call(ctx, http.MethodPost, "/v2/merchant/click_pass/payment",
		map[string]any{
			"service_id": c.serviceID(),
			"otp_data":   ch.OTPData,
			// ⚠️ So'm, not tiyin — CLICK counts in the unit the guest sees.
			// This is the opposite of Uzum, one file away, and getting it
			// backwards charges a hundredth or a hundred times the bill.
			"amount":         ch.Amount,
			"cashbox_code":   c.cfg.Cashbox,
			"transaction_id": ch.TxnID,
		})
	if err != nil {
		return Result{}, err
	}
	out := clickResult(reply)
	if reply.ErrorCode < 0 {
		return out, fmt.Errorf("%s", declined(reply.ErrorNote))
	}

	// ⚠️ **Confirm mode, and it is a thirty-second fuse.** When the service is
	// configured for it, CLICK holds the payment and reverses it automatically
	// unless we confirm — so a till that ignored this would show the cashier a
	// successful charge, let the guest leave, and have the money handed back
	// half a minute later with nothing on any screen to say so.
	//
	// Confirmed here rather than after the check closes, because the rest of
	// closing a check — firing the kitchen, filing the receipt, printing —
	// takes an unbounded amount of time, and thirty seconds is not a budget.
	if out.NeedsConfirm && out.Status != StatusFailed {
		if err := c.Confirm(ctx, out.PaymentID); err != nil {
			// The money was taken and we could not confirm it, so it is about
			// to be given back. Said plainly: the cashier's next move is to ask
			// for the code again, not to hand over the food.
			return Result{Status: StatusFailed, PaymentID: out.PaymentID},
				fmt.Errorf("to'lov tasdiqlanmadi va bank uni qaytaradi: %w", err)
		}
		out.Status = StatusPaid
	}
	return out, nil
}

func (c *clickPass) Confirm(ctx context.Context, paymentID string) error {
	id, _ := strconv.ParseInt(paymentID, 10, 64)
	reply, err := c.call(ctx, http.MethodPost, "/v2/merchant/click_pass/confirm",
		map[string]any{"service_id": c.serviceID(), "payment_id": id})
	if err != nil {
		return err
	}
	if reply.ErrorCode < 0 {
		return fmt.Errorf("%s", declined(reply.ErrorNote))
	}
	return nil
}

func (c *clickPass) Status(ctx context.Context, paymentID, _ string) (Result, error) {
	reply, err := c.call(ctx, http.MethodGet, fmt.Sprintf(
		"/v2/merchant/payment/status/%d/%s", c.serviceID(), paymentID), nil)
	if err != nil {
		return Result{}, err
	}
	out := clickResult(reply)
	// ⚠️ The status reply carries no payment_id, so it decodes as zero and
	// would otherwise overwrite the id we asked about with "0".
	out.PaymentID = paymentID
	if reply.ErrorCode < 0 {
		return out, fmt.Errorf("%s", declined(reply.ErrorNote))
	}
	return out, nil
}

// Reverse gives the whole payment back.
//
// ⚠️ **CLICK's window is a calendar month, not thirty days**: a payment from
// last month can only be reversed on the first day of this one. Nothing here
// can widen that, and a refund refused for this reason is not a bug — CLICK's
// own sentence is passed through so the owner reads the actual rule rather
// than "xatolik".
func (c *clickPass) Reverse(ctx context.Context, paymentID, _ string) error {
	reply, err := c.call(ctx, http.MethodDelete, fmt.Sprintf(
		"/v2/merchant/payment/reversal/%d/%s", c.serviceID(), paymentID), nil)
	if err != nil {
		return err
	}
	if reply.ErrorCode < 0 {
		return fmt.Errorf("%s", declined(reply.ErrorNote))
	}
	return nil
}

// Fiscal does nothing: CLICK Pass has no call for the receipt link. The guest's
// fiscal QR comes off our own printer, as it does for cash.
func (c *clickPass) Fiscal(context.Context, string, string) error { return nil }

// declined turns a provider's note into something a cashier can act on.
func declined(note string) string {
	note = strings.TrimSpace(note)
	if note == "" {
		return "to'lov rad etildi"
	}
	return note
}
