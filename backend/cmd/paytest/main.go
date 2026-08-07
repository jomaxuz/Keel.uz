// Command paytest plays the part of Payme, Click or Uzum against your own
// server, so the whole payment flow can be exercised without a bank.
//
// Why this exists: the providers only ever talk to a public HTTPS address, and
// getting one of those in front of a laptop means a merchant cabinet, a signed
// contract and a tunnel. None of that is needed to answer the question you
// actually have while building — *does my server do the right thing when the
// bank calls?* This sends exactly the calls the bank would, signed exactly the
// way the bank signs them, and prints what came back.
//
// It reads the merchant credentials straight from the database, which is why it
// belongs on the server rather than in a test file: it has to sign with the same
// key the real provider was given.
//
//	# take an order that is waiting to be paid and pay it
//	go run ./cmd/paytest -order AB12-3456
//
//	# the whole set of refusals a provider must be given
//	go run ./cmd/paytest -order AB12-3456 -suite
//
//	# one step at a time
//	go run ./cmd/paytest -order AB12-3456 -step create
//	go run ./cmd/paytest -order AB12-3456 -step perform
//	go run ./cmd/paytest -order AB12-3456 -step cancel
//
//	In Docker: docker compose -f docker-compose.prod.yml exec backend \
//	               /app/paytest -order AB12-3456 -api http://localhost:8080/api/v1
package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"restaurant-backend/internal/config"
	"restaurant-backend/internal/db"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
)

var (
	api      string
	settings models.PaymentSettings
	order    models.Order
	// One id per run, so a second run does not collide with the first.
	txnID  string
	failed int
)

func main() {
	orderNumber := flag.String("order", "", "order number to pay (required)")
	provider := flag.String("provider", "", "payme | click | uzum | atmos (default: the order's own method)")
	step := flag.String("step", "", "create | perform | cancel | status — one step instead of the full flow")
	suite := flag.Bool("suite", false, "run the refusal cases too: wrong key, wrong amount, unknown order, replays")
	apiFlag := flag.String("api", "http://localhost:8080/api/v1", "base URL of the running server")
	dbName := flag.String("db", "", "database name (default: MONGO_DB)")
	flag.Parse()

	if strings.TrimSpace(*orderNumber) == "" {
		fmt.Fprintln(os.Stderr, "need -order <number>; see -h")
		os.Exit(2)
	}
	api = strings.TrimRight(*apiFlag, "/")

	cfg := config.Load()
	if *dbName != "" {
		cfg.MongoDB = *dbName
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("mongo: %v", err)
	}
	store := repository.New(database)

	if err := store.PaymentSettings.FindOne(ctx, bson.M{}).Decode(&settings); err != nil {
		log.Fatalf("no payment settings saved yet — fill them in at /admin/settings first")
	}
	number := strings.TrimPrefix(strings.TrimSpace(*orderNumber), "#")
	if err := store.Orders.FindOne(ctx, bson.M{"number": number}).Decode(&order); err != nil {
		log.Fatalf("order %q not found", number)
	}

	name := *provider
	if name == "" {
		name = order.PaymentMethod
	}
	if name == models.ProviderCash {
		log.Fatalf("order %s is a cash order — there is nothing for a bank to do", order.Number)
	}
	if !settings.Configured(name) {
		log.Fatalf("%s is not switched on or not fully configured in /admin/settings", name)
	}

	fmt.Printf("order  %s\n", order.Number)
	fmt.Printf("total  %d so'm (%d tiyin)\n", order.Total, order.Total*100)
	fmt.Printf("paid   %s\n", statusOf(&order))
	fmt.Printf("acting as %s → %s\n\n", name, api)

	txnID = fmt.Sprintf("paytest-%d", time.Now().UnixNano())

	switch name {
	case models.ProviderPayme:
		runPayme(*step, *suite)
	case models.ProviderClick:
		runClick(*step, *suite)
	case models.ProviderUzum:
		runUzum(*step, *suite)
	case models.ProviderAtmos:
		runAtmos(*suite)
	}

	// Re-read, so the last line is the truth from the database rather than from
	// the response we just parsed.
	var after models.Order
	if err := store.Orders.FindOne(ctx, bson.M{"_id": order.ID}).Decode(&after); err == nil {
		fmt.Printf("\norder %s is now: %s\n", after.Number, statusOf(&after))
		if after.QueuedAt != nil {
			fmt.Printf("kitchen may start from %s\n", after.QueuedAt.Format("15:04:05"))
		} else {
			fmt.Println("not in the kitchen queue yet (nothing paid)")
		}
	}
	if failed > 0 {
		fmt.Printf("\n%d check(s) did not do what they should\n", failed)
		os.Exit(1)
	}
}

func statusOf(o *models.Order) string {
	if o.PaymentStatus == "" {
		return models.PayUnpaid
	}
	return o.PaymentStatus
}

// ---- Payme ----

func runPayme(step string, suite bool) {
	account := map[string]any{paymeField(): order.Number}
	amount := int64(order.Total) * 100

	if suite {
		fmt.Println("— refusals —")
		// Wrong key: the one check that must fail before anything is looked up.
		report("wrong merchant key", -32504, paymeRaw("wrong-key-on-purpose", rpc("CheckPerformTransaction", map[string]any{
			"amount": amount, "account": account,
		})))
		report("wrong amount", -31001, payme(rpc("CheckPerformTransaction", map[string]any{
			"amount": 100, "account": account,
		})))
		report("unknown order", -31050, payme(rpc("CheckPerformTransaction", map[string]any{
			"amount": amount, "account": map[string]any{paymeField(): "NO-SUCH-0000"},
		})))
		report("perform an unknown transaction", -31003, payme(rpc("PerformTransaction", map[string]any{
			"id": "never-created",
		})))
		fmt.Println()
	}

	if step == "" || step == "create" {
		fmt.Println("— CheckPerformTransaction —")
		report("may this be paid", 0, payme(rpc("CheckPerformTransaction", map[string]any{
			"amount": amount, "account": account,
		})))

		fmt.Println("— CreateTransaction —")
		report("create", 0, payme(rpc("CreateTransaction", map[string]any{
			"id": txnID, "time": time.Now().UnixMilli(),
			"amount": amount, "account": account,
		})))
		if suite {
			// Providers retry; a repeat must answer the same, not error.
			report("create again (retry)", 0, payme(rpc("CreateTransaction", map[string]any{
				"id": txnID, "time": time.Now().UnixMilli(),
				"amount": amount, "account": account,
			})))
		}
	}

	if step == "" || step == "perform" {
		fmt.Println("— PerformTransaction —")
		report("perform", 0, payme(rpc("PerformTransaction", map[string]any{"id": txnID})))
		if suite {
			report("perform again (retry)", 0, payme(rpc("PerformTransaction", map[string]any{"id": txnID})))
			report("pay an order that is already paid", -31051, payme(rpc("CheckPerformTransaction", map[string]any{
				"amount": amount, "account": account,
			})))
		}
	}

	if step == "cancel" {
		fmt.Println("— CancelTransaction —")
		report("cancel", 0, payme(rpc("CancelTransaction", map[string]any{"id": txnID, "reason": 5})))
	}

	if step == "" || step == "status" {
		fmt.Println("— CheckTransaction —")
		report("status", 0, payme(rpc("CheckTransaction", map[string]any{"id": txnID})))
	}
}

func paymeField() string {
	if f := strings.TrimSpace(settings.Payme.AccountField); f != "" {
		return f
	}
	return "order_id"
}

func paymeKey() string {
	if settings.Payme.TestMode {
		return settings.Payme.TestKey
	}
	return settings.Payme.Key
}

func rpc(method string, params map[string]any) []byte {
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": time.Now().UnixNano() % 100000,
		"method": method, "params": params,
	})
	return body
}

func payme(body []byte) int { return paymeRaw(paymeKey(), body) }

// paymeRaw sends one JSON-RPC call and returns the error code it answered
// with, or 0 for a successful result.
func paymeRaw(key string, body []byte) int {
	req, _ := http.NewRequest(http.MethodPost, api+"/payments/payme", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+
		base64.StdEncoding.EncodeToString([]byte("Paycom:"+key)))
	text := send(req)
	var parsed struct {
		Error *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal([]byte(text), &parsed)
	if parsed.Error != nil {
		return parsed.Error.Code
	}
	return 0
}

// ---- Click ----

func runClick(step string, suite bool) {
	amount := fmt.Sprintf("%d.00", order.Total)
	signTime := time.Now().Format("2006-01-02 15:04:05")
	prepareID := ""

	if suite {
		fmt.Println("— refusals —")
		bad := url.Values{
			"click_trans_id": {txnID}, "service_id": {settings.Click.ServiceID},
			"click_paydoc_id": {"1"}, "merchant_trans_id": {order.Number},
			"amount": {amount}, "action": {"0"}, "error": {"0"}, "error_note": {""},
			"sign_time": {signTime}, "sign_string": {"0000000000000000"},
		}
		wantClick(clickPost("prepare", bad), -1, "wrong signature")

		unknown := clickForm("0", "NO-SUCH-0000", amount, "", signTime)
		wantClick(clickPost("prepare", unknown), -5, "unknown order")

		wrongAmount := clickForm("0", order.Number, "1.00", "", signTime)
		wantClick(clickPost("prepare", wrongAmount), -2, "wrong amount")
		fmt.Println()
	}

	if step == "" || step == "create" {
		fmt.Println("— Prepare —")
		text := clickPost("prepare", clickForm("0", order.Number, amount, "", signTime))
		wantClick(text, 0, "prepare")
		var parsed struct {
			MerchantPrepareID string `json:"merchant_prepare_id"`
		}
		_ = json.Unmarshal([]byte(text), &parsed)
		prepareID = parsed.MerchantPrepareID
	}

	if (step == "" || step == "perform") && prepareID != "" {
		fmt.Println("— Complete —")
		form := clickForm("1", order.Number, amount, prepareID, signTime)
		wantClick(clickPost("complete", form), 0, "complete")
		if suite {
			wantClick(clickPost("complete", form), -4, "complete again (already paid)")
		}
	} else if step == "perform" {
		fmt.Println("Complete needs the prepare id from a Prepare in the same run — " +
			"use the full flow rather than -step perform.")
	}
}

func clickForm(action, merchantTrans, amount, prepareID, signTime string) url.Values {
	secret := settings.Click.SecretKey
	inPrepare := ""
	if action == "1" {
		inPrepare = prepareID
	}
	raw := txnID + settings.Click.ServiceID + secret + merchantTrans +
		inPrepare + amount + action + signTime
	sum := md5.Sum([]byte(raw))
	v := url.Values{
		"click_trans_id": {txnID}, "service_id": {settings.Click.ServiceID},
		"click_paydoc_id": {"1"}, "merchant_trans_id": {merchantTrans},
		"amount": {amount}, "action": {action}, "error": {"0"}, "error_note": {""},
		"sign_time": {signTime}, "sign_string": {hex.EncodeToString(sum[:])},
	}
	if prepareID != "" {
		v.Set("merchant_prepare_id", prepareID)
	}
	return v
}

func clickPost(path string, form url.Values) string {
	req, _ := http.NewRequest(http.MethodPost, api+"/payments/click/"+path,
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return send(req)
}

// wantClick reads the `error` field Click answers with and checks it.
func wantClick(text string, want int, label string) {
	var parsed struct {
		Error json.Number `json:"error"`
	}
	_ = json.Unmarshal([]byte(text), &parsed)
	got, _ := strconv.Atoi(parsed.Error.String())
	report(label, want, got)
}

// ---- Uzum ----

func runUzum(step string, suite bool) {
	field := strings.TrimSpace(settings.Uzum.AccountField)
	if field == "" {
		field = "order_id"
	}
	params := map[string]any{field: order.Number}
	amount := int64(order.Total) * 100

	if suite {
		fmt.Println("— refusals —")
		wantUzum(uzumRaw("wrong", "credentials", "check", map[string]any{
			"serviceId": settings.Uzum.ServiceID, "params": params,
		}), 10001, "wrong credentials")
		wantUzum(uzum("check", map[string]any{
			"serviceId": "000000", "params": params,
		}), 10006, "wrong service id")
		wantUzum(uzum("create", map[string]any{
			"serviceId": settings.Uzum.ServiceID, "transId": txnID,
			"amount": 100, "params": params,
		}), 99999, "wrong amount")
		wantUzum(uzum("confirm", map[string]any{
			"serviceId": settings.Uzum.ServiceID, "transId": "never-created",
		}), 10008, "confirm an unknown transaction")
		fmt.Println()
	}

	if step == "" || step == "create" {
		fmt.Println("— check —")
		wantUzum(uzum("check", map[string]any{
			"serviceId": settings.Uzum.ServiceID, "params": params,
		}), 0, "check")

		fmt.Println("— create —")
		wantUzum(uzum("create", map[string]any{
			"serviceId": settings.Uzum.ServiceID, "transId": txnID,
			"amount": amount, "params": params,
		}), 0, "create")
		if suite {
			wantUzum(uzum("create", map[string]any{
				"serviceId": settings.Uzum.ServiceID, "transId": txnID,
				"amount": amount, "params": params,
			}), 0, "create again (retry)")
		}
	}

	if step == "" || step == "perform" {
		fmt.Println("— confirm —")
		wantUzum(uzum("confirm", map[string]any{
			"serviceId": settings.Uzum.ServiceID, "transId": txnID,
		}), 0, "confirm")
		if suite {
			wantUzum(uzum("confirm", map[string]any{
				"serviceId": settings.Uzum.ServiceID, "transId": txnID,
			}), 0, "confirm again (retry)")
		}
	}

	if step == "cancel" {
		fmt.Println("— reverse —")
		wantUzum(uzum("reverse", map[string]any{
			"serviceId": settings.Uzum.ServiceID, "transId": txnID,
		}), 0, "reverse")
	}

	if step == "" || step == "status" {
		fmt.Println("— status —")
		wantUzum(uzum("status", map[string]any{
			"serviceId": settings.Uzum.ServiceID, "transId": txnID,
		}), 0, "status")
	}
}

func uzum(path string, body map[string]any) string {
	return uzumRaw(settings.Uzum.Login, settings.Uzum.Password, path, body)
}

func uzumRaw(login, password, path string, body map[string]any) string {
	if _, ok := body["timestamp"]; !ok {
		body["timestamp"] = time.Now().UnixMilli()
	}
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, api+"/payments/uzum/"+path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+
		base64.StdEncoding.EncodeToString([]byte(login+":"+password)))
	return send(req)
}

// wantUzum reads the numeric errorCode Uzum answers with (absent = success).
func wantUzum(text string, want int, label string) {
	var parsed struct {
		ErrorCode int `json:"errorCode"`
	}
	_ = json.Unmarshal([]byte(text), &parsed)
	report(label, want, parsed.ErrorCode)
}

// ---- ATMOS ----
//
// One endpoint, and the answer decides whether a real guest is charged: ATMOS
// only takes the money once this server answers `status: 1`. So the refusals
// matter more here than anywhere else — every one of them is a case where
// charging would be wrong, and a bug that answers "yes" to any of them takes
// money for an order nobody can honour.
func runAtmos(suite bool) {
	amount := strconv.FormatInt(int64(order.Total)*100, 10)
	storeID := strings.TrimSpace(settings.Atmos.StoreID)

	if suite {
		fmt.Println("— refusals —")
		// A signature made with the wrong key. This is the case a real
		// misconfiguration produces: the OAuth secret pasted into the callback
		// field looks right in the panel and fails only here.
		wantAtmos(atmosRaw(storeID, txnID, order.Number, amount, "wrong-key"),
			0, "wrong signing key")
		// No signature at all — an unsigned probe finding the endpoint.
		wantAtmos(atmosSend(map[string]any{
			"store_id": json.RawMessage(storeID), "transaction_id": txnID,
			"invoice": order.Number, "amount": json.RawMessage(amount),
		}), 0, "no signature")
		// The amount is the order's, never the callback's.
		wantAtmos(atmosRaw(storeID, txnID+"-cheap", order.Number, "100",
			settings.Atmos.APIKey), 0, "wrong amount")
		wantAtmos(atmosRaw(storeID, txnID+"-ghost", "NO-SUCH-ORDER", amount,
			settings.Atmos.APIKey), 0, "unknown order")
		fmt.Println()
	}

	fmt.Println("— confirm —")
	wantAtmos(atmosRaw(storeID, txnID, order.Number, amount, settings.Atmos.APIKey),
		1, "charge the guest")

	if suite {
		// ATMOS retries; a repeat must be answered the same way and must not
		// take the money twice.
		wantAtmos(atmosRaw(storeID, txnID, order.Number, amount, settings.Atmos.APIKey),
			0, "replay is refused once the order is paid")
	}
}

// atmosRaw signs the callback exactly as ATMOS documents it and sends it.
func atmosRaw(storeID, txn, invoice, amount, key string) string {
	sum := md5.Sum([]byte(storeID + txn + invoice + amount + key))
	return atmosSend(map[string]any{
		"store_id": json.RawMessage(storeID), "transaction_id": txn,
		"invoice": invoice, "amount": json.RawMessage(amount),
		"sign": hex.EncodeToString(sum[:]),
	})
}

// atmosSend posts the callback with `store_id` and `transaction_id` as JSON
// **numbers** — the likelier real shape, and the one that would have been
// refused before the handler learned to read either.
func atmosSend(body map[string]any) string {
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, api+"/payments/atmos", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return send(req)
}

// wantAtmos reads the `status` ATMOS looks at: 1 charges, anything else does not.
//
// ⚠️ **A body that is not JSON is a failure, never a refusal.** ATMOS's contract
// is "no `status: 1`, no money", so an unrouted endpoint answering
// `404 page not found` parses as status 0 and every refusal check passes —
// against a server that cannot take a payment at all. That happened on the
// first run of this suite, and the four green ticks above the one red one were
// the only warning. A refusal has to be a refusal *we sent*.
func wantAtmos(text string, want int, label string) {
	var parsed struct {
		Status *int `json:"status"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil || parsed.Status == nil {
		fmt.Printf("   ✗ %s — no JSON answer (is /payments/atmos routed?)\n", label)
		failed++
		return
	}
	report(label, want, *parsed.Status)
}

// ---- Output ----

func send(req *http.Request) string {
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("   ! %v\n", err)
		failed++
		return "{}"
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	text := strings.TrimSpace(string(body))
	fmt.Printf("   %s\n", text)
	return text
}

// report prints whether a call answered with the code it had to.
func report(label string, want, got int) {
	if want == got {
		fmt.Printf("   ✓ %s\n", label)
		return
	}
	fmt.Printf("   ✗ %s — expected %d, got %d\n", label, want, got)
	failed++
}
