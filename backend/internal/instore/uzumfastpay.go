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

// Uzum FastPay v2 — docs/vendor/uzum-fastpay.md (developer.uzumbank.uz, read
// 2026-08-30).
//
// The same counter flow as CLICK Pass, with three differences that all matter:
//
//   - money is in **tiyin**, not so'm;
//   - the timestamp is in **milliseconds** and has a fifty-second window;
//   - the bank wants the **fiscal receipt's link** afterwards, so the guest can
//     open it from inside the Uzum app. That is the one call CLICK has no
//     equivalent for, and it is why Charger carries Fiscal.

// uzumHost is the live host. Overridable through Config.BaseURL, because Uzum
// does issue test environments through the account manager — unlike CLICK,
// where a settable host would be a box with one correct value.
const uzumHost = "https://mobile.apelsin.uz"

type uzumFastPay struct {
	cfg  Config
	http *http.Client
	now  func() time.Time
}

func newUzumFastPay(cfg Config, client *http.Client) (Charger, error) {
	if cfg.ServiceID == "" || cfg.UserID == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("Uzum FastPay: servis id, kassa id va maxfiy kalit kerak")
	}
	return &uzumFastPay{cfg: cfg, http: client, now: time.Now}, nil
}

func (u *uzumFastPay) host() string {
	if h := strings.TrimRight(strings.TrimSpace(u.cfg.BaseURL), "/"); h != "" {
		return h
	}
	return uzumHost
}

func (u *uzumFastPay) serviceID() int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(u.cfg.ServiceID), 10, 64)
	return n
}

func (u *uzumFastPay) auth() string {
	stamp := msStamp(u.now())
	return u.cfg.UserID + ":" + sign(stamp, u.cfg.SecretKey) + ":" + stamp
}

// uzumReply is the envelope. ⚠️ Uzum's payment_id is a **UUID string**, where
// CLICK's is a number — the two adapters are not interchangeable at this level
// however similar the flow looks.
type uzumReply struct {
	PaymentID     string `json:"payment_id"`
	PaymentStatus string `json:"payment_status"`
	ErrorCode     int    `json:"error_code"`
	ErrorMessage  string `json:"error_message"`
	OperationTime string `json:"operation_time"`
	ClientPhone   string `json:"client_phone_number"`
	CardType      int    `json:"card_type"`
	RefNumber     string `json:"processing_reference_number"`
}

func (u *uzumFastPay) call(
	ctx context.Context, method, path string, body any,
) (uzumReply, error) {
	var buf io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return uzumReply{}, err
		}
		buf = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.host()+path, buf)
	if err != nil {
		return uzumReply{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	// ⚠️ `Authorization`, and it is not a scheme — no "Bearer", no "Basic".
	// Uzum validates the raw value against `^\d*:(\d{40}):\d*$`, so a helper
	// that helpfully prefixes a scheme turns every call into a 401.
	req.Header.Set("Authorization", u.auth())

	res, err := u.http.Do(req)
	if err != nil {
		return uzumReply{}, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return uzumReply{}, err
	}
	var out uzumReply
	if err := json.Unmarshal(raw, &out); err != nil {
		// ⚠️ **Every FastPay method answers 200, even when it refuses** — the
		// documentation says so outright. So a non-2xx status here is not a
		// declined payment, it is the gateway or a proxy answering instead of
		// Uzum, and it has no error_code to read.
		if res.StatusCode >= 400 {
			return uzumReply{}, fmt.Errorf("Uzum FastPay: %s", http.StatusText(res.StatusCode))
		}
		return uzumReply{}, fmt.Errorf("Uzum FastPay: javob o'qilmadi")
	}
	return out, nil
}

// uzumStatusOf reads the state word.
//
// ⚠️ **`error_code == 0` is the success test, not the status word**, because a
// refusal comes back with a 200 and often with no status at all. The word is
// then used to separate a completed payment from one the processing centre is
// still deciding on (`operation.is.inProcess`) — which must never close a check.
func uzumStatusOf(r uzumReply) string {
	if r.ErrorCode != 0 {
		return StatusFailed
	}
	switch strings.ToUpper(strings.TrimSpace(r.PaymentStatus)) {
	case "SUCCESS", "CONFIRMED", "PAID":
		return StatusPaid
	case "REVERSED", "CANCELED", "CANCELLED", "FAILED", "DECLINED":
		return StatusFailed
	default:
		return StatusPending
	}
}

func uzumResult(r uzumReply) Result {
	return Result{
		PaymentID:  r.PaymentID,
		Status:     uzumStatusOf(r),
		Phone:      r.ClientPhone,
		Processing: r.RefNumber,
	}
}

func (u *uzumFastPay) Charge(ctx context.Context, ch Charge) (Result, error) {
	reply, err := u.call(ctx, http.MethodPost, "/api/apelsin-pay/merchant/v2/payment",
		map[string]any{
			// ⚠️ **Tiyin.** The unit CLICK does not use, in the file next door.
			"amount":         int64(ch.Amount) * 100,
			"cashbox_code":   u.cfg.Cashbox,
			"otp_data":       ch.OTPData,
			"order_id":       ch.OrderID,
			"transaction_id": ch.TxnID,
			"service_id":     u.serviceID(),
		})
	if err != nil {
		return Result{}, err
	}
	out := uzumResult(reply)
	if reply.ErrorCode != 0 {
		return out, fmt.Errorf("%s", uzumMessage(reply))
	}
	return out, nil
}

// Confirm does nothing: FastPay has no hold-and-confirm mode. A successful
// payment is final the moment it answers, which is why the thirty-second fuse
// in clickpass.go has no counterpart here.
func (u *uzumFastPay) Confirm(context.Context, string) error { return nil }

func (u *uzumFastPay) Status(ctx context.Context, paymentID, _ string) (Result, error) {
	reply, err := u.call(ctx, http.MethodPost, "/api/apelsin-pay/merchant/payment/status",
		map[string]any{"payment_id": paymentID, "service_id": u.serviceID()})
	if err != nil {
		return Result{}, err
	}
	out := uzumResult(reply)
	if out.PaymentID == "" {
		out.PaymentID = paymentID
	}
	if reply.ErrorCode != 0 {
		return out, fmt.Errorf("%s", uzumMessage(reply))
	}
	return out, nil
}

// Reverse gives the whole payment back. ⚠️ **Partial reversal is not
// supported** — the caller refuses a part-refund before it reaches here, rather
// than sending one and having Uzum return the whole amount.
func (u *uzumFastPay) Reverse(ctx context.Context, paymentID, orderID string) error {
	reply, err := u.call(ctx, http.MethodPut,
		"/api/apelsin-pay/merchant/v2/payment/reversal/"+orderID,
		map[string]any{"service_id": u.serviceID(), "payment_id": paymentID})
	if err != nil {
		return err
	}
	if reply.ErrorCode != 0 {
		return fmt.Errorf("%s", uzumMessage(reply))
	}
	return nil
}

// Fiscal hands Uzum the link to the filed receipt, so the guest can open it
// from the payment inside their own app.
//
// ⚠️ **Not part of taking the money, and it must never be able to undo it.**
// This runs after the register has answered, which is minutes later and over a
// different network; a failure here means the guest's app shows a payment
// without a receipt link, while the receipt itself is filed, printed and
// correct. The caller logs it and moves on.
func (u *uzumFastPay) Fiscal(ctx context.Context, paymentID, url string) error {
	if strings.TrimSpace(url) == "" {
		return nil
	}
	reply, err := u.call(ctx, http.MethodPost, "/api/apelsin-pay/merchant/payment/fiscal",
		map[string]any{
			"payment_id": paymentID,
			"service_id": u.serviceID(),
			"fiscal_url": url,
		})
	if err != nil {
		return err
	}
	if reply.ErrorCode != 0 {
		return fmt.Errorf("%s", uzumMessage(reply))
	}
	return nil
}

// uzumMessage turns FastPay's machine words into a sentence for the counter.
//
// ⚠️ **Two of these are the cashier's problem to solve and the rest are not**,
// which is the whole reason this function exists rather than passing the raw
// string through. An expired or already-used QR is fixed by asking the guest to
// open the app again — a five-second fix that reads, untranslated, like the
// till is broken.
func uzumMessage(r uzumReply) string {
	switch strings.TrimSpace(r.ErrorMessage) {
	case "apelsin.pay.user.otp.data.expired":
		return "QR kodning muddati o'tgan — mijozdan yangisini ochishini so'rang"
	case "apelsin.pay.wrong.prefix.otp.data", "receipt.qr.only.yours":
		return "Bu QR Uzum to'lov kodi emas — mijoz ilovada «To'lash» QR'ini ochsin"
	case "qr.duplicated":
		return "Bu QR bilan allaqachon to'langan — mijozdan yangi QR so'rang"
	case "apelsin.pay.safe.mode.on":
		return "Mijoz kartasi Safe Mode'da — Uzum ilovasida o'chirilishi kerak"
	case "operation.is.inProcess":
		return "Bank hali javob bermadi — statusni tekshiring, qayta urinmang"
	case "order.id.duplicated", "transaction.duplicated":
		return "Bu chek uchun to'lov allaqachon yuborilgan"
	case "apelsin.pay.reverse.not.allowed":
		return "Bu hamkor uchun qaytarish ruxsat etilmagan"
	case "":
		if r.ErrorCode != 0 {
			return "to'lov rad etildi (kod " + strconv.Itoa(r.ErrorCode) + ")"
		}
		return "to'lov rad etildi"
	default:
		// Anything unlisted is passed through: an unfamiliar sentence from the
		// bank is more useful to whoever is called about it than a guess of
		// ours, and this list will always be behind theirs.
		return r.ErrorMessage
	}
}
