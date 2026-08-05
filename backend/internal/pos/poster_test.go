package pos

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The two things Poster can get wrong quietly are money and acceptance:
// prices are in tiyin, and an incoming order with status 0 is filed but not
// yet taken by anyone at the till. Both are cheap to assert and expensive to
// discover in a restaurant, so they are pinned here.

func posterStub(t *testing.T, sent *map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/spots.getSpots"):
			io.WriteString(w, `{"response":[
				{"spot_id":"1","spot_name":"Chilonzor","spot_adress":"Bunyodkor 12"},
				{"spot_id":"2","spot_name":"Yunusobod"}]}`)
		case strings.HasSuffix(r.URL.Path, "/menu.getProducts"):
			// Prices as strings in tiyin, ids as strings — Poster's actual shape.
			io.WriteString(w, `{"response":[
				{"product_id":"169","product_name":"Lag'mon","category_name":"Issiq",
				 "hidden":"0","price":{"1":"3500000","2":"3600000"},
				 "spots":[{"spot_id":"1","price":"3500000","visible":"1"}]},
				{"product_id":"170","product_name":"Somsa","category_name":"Non",
				 "hidden":"0","price":{"1":"1200000"},
				 "spots":[{"spot_id":"1","price":"1200000","visible":"0"}]},
				{"product_id":"171","product_name":"Arxiv taom","hidden":"1",
				 "price":{"1":"100000"}}]}`)
		case strings.HasSuffix(r.URL.Path, "/incomingOrders.createIncomingOrder"):
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, sent)
			io.WriteString(w, `{"response":{"incoming_order_id":42,"status":0}}`)
		case strings.HasSuffix(r.URL.Path, "/incomingOrders.getIncomingOrder"):
			io.WriteString(w, `{"response":{"status":1,"transaction_id":1949}}`)
		default:
			io.WriteString(w, `{"error":35,"message":"Incorrect method"}`)
		}
	}))
}

func posterFor(t *testing.T, base string, spot int) *posterClient {
	t.Helper()
	return newPoster(Config{
		Provider: Poster, PosterToken: "tkn", PosterSpotID: spot, PosterBaseURL: base,
	}, &http.Client{})
}

func TestPosterPingNamesTheSpot(t *testing.T) {
	var sent map[string]any
	srv := posterStub(t, &sent)
	defer srv.Close()

	got, err := posterFor(t, srv.URL, 1).Ping(context.Background())
	if err != nil {
		t.Fatalf("ping: %v", err)
	}
	if !strings.Contains(got, "Chilonzor") || !strings.Contains(got, "Bunyodkor") {
		t.Fatalf("ping should name the spot it reached, got %q", got)
	}

	// A token that works but points at a branch that does not exist is the
	// failure worth catching here — it looks identical to a correct setup
	// until an order prints in another building.
	if _, err := posterFor(t, srv.URL, 9).Ping(context.Background()); err == nil {
		t.Fatal("unknown spot_id must fail the check")
	} else if !strings.Contains(err.Error(), "Yunusobod") {
		t.Fatalf("the error should list the real spots, got %q", err)
	}
}

func TestPosterProductsConvertTiyinAndFlagStock(t *testing.T) {
	var sent map[string]any
	srv := posterStub(t, &sent)
	defer srv.Close()

	items, err := posterFor(t, srv.URL, 1).Products(context.Background())
	if err != nil {
		t.Fatalf("products: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("want 3 products, got %d", len(items))
	}
	if items[0].Price != 35000 {
		t.Fatalf("3500000 tiyin is 35000 so'm, got %d", items[0].Price)
	}
	if items[0].Unavailable {
		t.Fatal("a visible product must not be flagged unavailable")
	}
	// visible:0 for this spot — sold elsewhere, not here.
	if !items[1].Unavailable {
		t.Fatal("a product invisible at this spot must be flagged")
	}
	// hidden:1 globally.
	if !items[2].Unavailable {
		t.Fatal("a hidden product must be flagged")
	}
}

func TestPosterSendOrderSendsTiyinAndCarriesNotes(t *testing.T) {
	var sent map[string]any
	srv := posterStub(t, &sent)
	defer srv.Close()

	res, err := posterFor(t, srv.URL, 1).SendOrder(context.Background(), Order{
		Number: "YS19-1745",
		Type:   "delivery",
		Phone:  "+998 90 111 22 33",
		Items: []Item{
			{POSID: "169", Name: "Lag'mon", Qty: 2, Price: 35000, Comment: "piyozsiz"},
		},
		DeliveryFee: 15000,
		Total:       85000,
		Paid:        false,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	products, _ := sent["products"].([]any)
	if len(products) != 1 {
		t.Fatalf("want 1 line, got %v", sent["products"])
	}
	line, _ := products[0].(map[string]any)
	if line["price"].(float64) != 3500000 {
		t.Fatalf("35000 so'm must go over as 3500000 tiyin, got %v", line["price"])
	}
	if line["product_id"].(float64) != 169 {
		t.Fatalf("product_id must be numeric 169, got %v", line["product_id"])
	}

	// Poster has no per-line note field, so the guest's request must survive
	// in the order comment — it is the line that stops a dish coming back.
	comment, _ := sent["comment"].(string)
	if !strings.Contains(comment, "piyozsiz") {
		t.Fatalf("item comment lost, got %q", comment)
	}
	if !strings.Contains(comment, "Yetkazish narxi") {
		t.Fatalf("delivery fee has no field in Poster and must land in the comment, got %q", comment)
	}
	if _, ok := sent["payment"]; ok {
		t.Fatal("an unpaid order must not be declared prepaid")
	}
	if sent["phone"] != "+998901112233" {
		t.Fatalf("phone must be normalised, got %v", sent["phone"])
	}

	// status 0 is "filed, nobody has taken it" — the operator has to see that.
	if !strings.Contains(res.Note, "kutilmoqda") {
		t.Fatalf("status 0 must not read as accepted, got %q", res.Note)
	}
	if res.POSOrderID != "42" {
		t.Fatalf("want order id 42, got %q", res.POSOrderID)
	}
}

func TestPosterPaidOrderDeclaresPrepayment(t *testing.T) {
	var sent map[string]any
	srv := posterStub(t, &sent)
	defer srv.Close()

	_, err := posterFor(t, srv.URL, 1).SendOrder(context.Background(), Order{
		Phone: "998901112233",
		Items: []Item{{POSID: "169", Name: "Lag'mon", Qty: 1, Price: 35000}},
		Total: 35000,
		Paid:  true,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	pay, ok := sent["payment"].(map[string]any)
	if !ok {
		t.Fatal("a paid order must declare the prepayment")
	}
	if pay["sum"].(float64) != 3500000 {
		t.Fatalf("payment sum must be in tiyin, got %v", pay["sum"])
	}
}

func TestPosterStatusAndCancel(t *testing.T) {
	var sent map[string]any
	srv := posterStub(t, &sent)
	defer srv.Close()
	c := posterFor(t, srv.URL, 1)

	st, err := c.OrderStatus(context.Background(), "42")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if st.State != "accepted" {
		t.Fatalf("status 1 is accepted, got %q", st.State)
	}
	if !strings.Contains(st.Raw, "1949") {
		t.Fatalf("the till's transaction number is what a manager looks for, got %q", st.Raw)
	}
	if posterState(0) == "accepted" {
		t.Fatal("status 0 must never map to accepted")
	}

	// Poster's API has no cancel. Saying so beats a no-op that reads as success.
	if err := c.Cancel(context.Background(), "42"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("cancel must report ErrUnsupported, got %v", err)
	}
}

func TestPosterReportsApiErrorBody(t *testing.T) {
	// Poster answers 200 with an error body more often than it answers 4xx.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"error":"35","message":"Incorrect token"}`)
	}))
	defer srv.Close()

	_, err := posterFor(t, srv.URL, 1).Ping(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Incorrect token") {
		t.Fatalf("the till's own reason must reach the owner, got %v", err)
	}
}

func TestSyrveIsIikoOnItsOwnHost(t *testing.T) {
	p, err := New(Config{
		Provider: Syrve, IikoAPILogin: "x", IikoOrganizationID: "y",
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if p.Name() != Syrve {
		t.Fatalf("a Syrve install must call itself Syrve, got %q", p.Name())
	}
	if got := p.(*iikoClient).base; got != "https://api-eu.syrve.live" {
		t.Fatalf("Syrve must default to its own host, got %q", got)
	}
	// And an explicit host still wins, for whoever is on another region.
	p2, _ := New(Config{
		Provider: Syrve, IikoAPILogin: "x", IikoOrganizationID: "y",
		IikoBaseURL: "https://api-ae.syrve.live/",
	})
	if got := p2.(*iikoClient).base; got != "https://api-ae.syrve.live" {
		t.Fatalf("configured host must win, got %q", got)
	}
}
