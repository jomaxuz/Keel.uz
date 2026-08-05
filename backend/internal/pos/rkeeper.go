package pos

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// r_keeper 7 XML interface.
//
// Read the transport before the code: this one does **not** live in a cloud.
// The interface answers on the restaurant's own machine —
// `https://<ip>:<port>/rk7api/v0/xmlinterface.xml`, Basic auth with an r_keeper
// employee's login and password — so a server in a data centre can only reach
// it if the restaurant deliberately exposes it (a forwarded port with TLS, or a
// VPN between the two). Nothing in this file can fix that, and the panel says so
// next to the settings rather than letting an owner switch it on and then
// wonder why the kitchen never prints.
//
// UCS themselves now point new integrations at r_k White Server instead. This
// adapter is the direct XML route: it works, it is what an existing r_keeper 7
// site already has, and it needs no extra contract.
//
// Units are the other trap. **Prices are in kopecks** and **quantities in
// thousandths** — an order sent in whole so'm arrives a hundred times too
// cheap, and the till accepts it without complaint.
type rkeeperClient struct {
	cfg  Config
	http *http.Client
}

// newRKeeper deliberately ignores the shared client: this is the one provider
// that needs its own transport, because it dials a box on a restaurant's LAN.
func newRKeeper(cfg Config, _ *http.Client) *rkeeperClient {
	// Restaurant boxes almost always carry a self-signed certificate, and the
	// alternative to accepting it is the integration simply never connecting.
	// The link is to one named machine over the restaurant's own network or a
	// VPN, and the credentials are checked by r_keeper regardless.
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout: 8 * time.Second,
		}).DialContext,
	}
	return &rkeeperClient{
		cfg:  cfg,
		http: &http.Client{Timeout: 30 * time.Second, Transport: transport},
	}
}

func (c *rkeeperClient) Name() string { return RKeeper }

// query posts one RK7Query and returns the raw XML answer.
func (c *rkeeperClient) query(ctx context.Context, body string) ([]byte, error) {
	payload := `<?xml version="1.0" encoding="UTF-8"?>` + body
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.RKeeperURL,
		strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.SetBasicAuth(c.cfg.RKeeperLogin, c.cfg.RKeeperPassword)

	res, err := c.http.Do(req)
	if err != nil {
		// By far the most common failure, and the one worth explaining: the
		// box is simply not reachable from here.
		return nil, fmt.Errorf("r_keeper: %s manzilga ulanib bo'lmadi — "+
			"server restoran tarmog'i ichidami? (%w)", c.cfg.RKeeperURL, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if res.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("r_keeper: login yoki parol qabul qilinmadi")
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("r_keeper: %s", res.Status)
	}
	return raw, nil
}

// rk7Answer is the envelope every command answers with. `Status` is "Ok" or an
// error, and the text of the failure is in `ErrorText`.
type rk7Answer struct {
	XMLName   xml.Name `xml:"RK7QueryResult"`
	Status    string   `xml:"Status,attr"`
	ErrorText string   `xml:"ErrorText,attr"`
	CMD       string   `xml:"CMD,attr"`
	// CreateOrder / SaveOrder answer with the order's identity.
	Order struct {
		Visit      string `xml:"visit,attr"`
		OrderIdent string `xml:"orderIdent,attr"`
		Guid       string `xml:"guid,attr"`
		Ident      string `xml:"Ident,attr"`
		Name       string `xml:"name,attr"`
	} `xml:"Order"`
	VisitID string `xml:"VisitID,attr"`
	OrderID string `xml:"OrderID,attr"`
	Guid    string `xml:"guid,attr"`
	// GetSystemInfo
	ServerVersion string `xml:"ServerVersion,attr"`
	RestName      string `xml:"RestaurantName,attr"`
}

func (c *rkeeperClient) call(ctx context.Context, body string) (*rk7Answer, error) {
	raw, err := c.query(ctx, body)
	if err != nil {
		return nil, err
	}
	var out rk7Answer
	if err := xml.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("r_keeper: javobni o'qib bo'lmadi: %w", err)
	}
	if out.Status != "" && !strings.EqualFold(out.Status, "Ok") {
		return &out, fmt.Errorf("r_keeper: %s", firstNonEmpty(out.ErrorText, out.Status))
	}
	return &out, nil
}

func (c *rkeeperClient) Ping(ctx context.Context) (string, error) {
	out, err := c.call(ctx, `<RK7Query><RK7CMD CMD="GetSystemInfo"/></RK7Query>`)
	if err != nil {
		return "", err
	}
	name := firstNonEmpty(out.RestName, "r_keeper")
	if out.ServerVersion != "" {
		name += " · " + out.ServerVersion
	}
	return name, nil
}

// rk7MenuItems is the dish reference.
type rk7MenuItems struct {
	XMLName xml.Name `xml:"RK7QueryResult"`
	Items   []struct {
		Ident           string `xml:"Ident,attr"`
		Code            string `xml:"Code,attr"`
		Name            string `xml:"Name,attr"`
		Parent          string `xml:"Parent,attr"`
		Price           string `xml:"Price,attr"`
		Status          string `xml:"Status,attr"`
		ItemIdent       string `xml:"ItemIdent,attr"`
		MainParentIdent string `xml:"MainParentIdent,attr"`
	} `xml:"RK7Reference>Items>Item"`
}

// rk7OrderMenu is what the station can actually sell right now — r_keeper's
// answer to "what is on the stop list", expressed the other way round.
type rk7OrderMenu struct {
	XMLName xml.Name `xml:"RK7QueryResult"`
	Dishes  []struct {
		ID       string `xml:"id,attr"`
		Ident    string `xml:"Ident,attr"`
		Price    string `xml:"price,attr"`
		Quantity string `xml:"quantity,attr"`
	} `xml:"Dishes>Item"`
}

func (c *rkeeperClient) Products(ctx context.Context) ([]Product, error) {
	// The full reference: names and grouping.
	raw, err := c.query(ctx,
		`<RK7Query><RK7CMD CMD="GetRefData" RefName="MENUITEMS" onlyActive="1"/></RK7Query>`)
	if err != nil {
		return nil, err
	}
	var ref rk7MenuItems
	if err := xml.Unmarshal(raw, &ref); err != nil {
		return nil, fmt.Errorf("r_keeper: menyuni o'qib bo'lmadi: %w", err)
	}

	// What the station may sell right now, with live quantities. Two requests
	// rather than one because the reference alone lists dishes the till will
	// refuse; a mapping screen built from it offers things that cannot be sold.
	sellable := map[string]bool{}
	if c.cfg.RKeeperStation != "" {
		if rawMenu, err := c.query(ctx, fmt.Sprintf(
			`<RK7Query><RK7CMD CMD="GetOrderMenu"><Station code="%s"/></RK7CMD></RK7Query>`,
			xmlAttr(c.cfg.RKeeperStation))); err == nil {
			var menu rk7OrderMenu
			if xml.Unmarshal(rawMenu, &menu) == nil {
				for _, d := range menu.Dishes {
					sellable[firstNonEmpty(d.Ident, d.ID)] = true
				}
			}
		}
	}

	out := make([]Product, 0, len(ref.Items))
	for _, it := range ref.Items {
		if it.Name == "" {
			continue
		}
		id := firstNonEmpty(it.Ident, it.ItemIdent, it.Code)
		if id == "" {
			continue
		}
		out = append(out, Product{
			ID:   id,
			Name: it.Name,
			// Prices come back in kopecks.
			Price: rk7Kopecks(it.Price),
			// Only judged when the station answered; without one, everything
			// is shown rather than everything being marked unsellable.
			Unavailable: len(sellable) > 0 && !sellable[id],
		})
	}
	return out, nil
}

func (c *rkeeperClient) SendOrder(ctx context.Context, o Order) (Result, error) {
	if err := CheckMapped(o.Items); err != nil {
		return Result{}, err
	}
	if c.cfg.RKeeperStation == "" {
		return Result{}, fmt.Errorf("r_keeper: stansiya (kassa) kodi sozlanmagan")
	}

	// One command: create the order and save it with its lines. Splitting
	// create and save leaves an empty order behind in the till whenever the
	// second call fails, and those have to be cleared by hand.
	var b strings.Builder
	b.WriteString(`<RK7Query>`)
	b.WriteString(c.licence())
	fmt.Fprintf(&b, `<RK7CMD CMD="CreateOrder"><Order guid=""><Station code="%s"/>`,
		xmlAttr(c.cfg.RKeeperStation))
	fmt.Fprintf(&b, `<OrderComment>%s</OrderComment>`, xmlText(c.comment(o)))
	b.WriteString(`<Session><Dishes>`)
	for _, it := range o.Items {
		// Quantity is in thousandths.
		fmt.Fprintf(&b, `<Dish id="%s" quantity="%d"`,
			xmlAttr(it.POSID), it.Qty*1000)
		if it.Comment != "" {
			fmt.Fprintf(&b, ` comment="%s"`, xmlAttr(it.Comment))
		}
		b.WriteString(`/>`)
	}
	b.WriteString(`</Dishes></Session></Order></RK7CMD></RK7Query>`)

	out, err := c.call(ctx, b.String())
	if err != nil {
		return Result{}, err
	}
	id := firstNonEmpty(out.Order.Guid, out.Order.OrderIdent, out.OrderID, out.Guid)
	if id == "" {
		return Result{}, fmt.Errorf("r_keeper: buyurtma yaratildi, lekin identifikator qaytmadi")
	}
	note := ""
	if out.VisitID != "" {
		note = "r_keeper visit " + out.VisitID
	}
	return Result{POSOrderID: id, Note: note}, nil
}

// comment is everything the kitchen and the courier need that r_keeper has no
// field for: our own order number first, so a ticket on the pass can be tied
// back to the site without asking anyone.
func (c *rkeeperClient) comment(o Order) string {
	parts := []string{"№" + o.Number}
	switch o.Type {
	case "delivery":
		parts = append(parts, "Yetkazib berish", o.Address)
	case "pickup":
		parts = append(parts, "Olib ketish")
	case "dinein":
		parts = append(parts, "Stol "+o.TableNumber)
	}
	if o.CustomerName != "" || o.Phone != "" {
		parts = append(parts, strings.TrimSpace(o.CustomerName+" "+o.Phone))
	}
	if o.Paid {
		parts = append(parts, "TO'LANGAN")
	}
	if o.Comment != "" {
		parts = append(parts, o.Comment)
	}
	return strings.Join(parts, " · ")
}

// licence is the block subscription-licensed sites must send with commands that
// change money. Empty for a lifetime licence, which needs nothing.
func (c *rkeeperClient) licence() string {
	if c.cfg.RKeeperAnchor == "" || c.cfg.RKeeperToken == "" {
		return ""
	}
	return fmt.Sprintf(
		`<LicenseInfo anchor="%s" licenseToken="%s"><LicenseInstance guid="softmax" seqNumber="0"/></LicenseInfo>`,
		xmlAttr(c.cfg.RKeeperAnchor), xmlAttr(c.cfg.RKeeperToken))
}

// OrderStatus is not offered: r_keeper's order lifecycle is a till session, not
// a delivery pipeline, and there is no equivalent of "is it cooked yet".
func (c *rkeeperClient) OrderStatus(context.Context, string) (Status, error) {
	return Status{}, ErrUnsupported
}

// Cancel is deliberately not automated. Deleting a saved order in r_keeper is a
// cash-register operation with its own permissions and audit trail, and doing it
// from a website would put a hole in the restaurant's own reporting. The panel
// says to void it at the till.
func (c *rkeeperClient) Cancel(context.Context, string) error {
	return ErrUnsupported
}

// rk7Kopecks turns r_keeper's kopeck price into whole so'm.
func rk7Kopecks(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return int(v / 100)
	}
	return 0
}

// xmlAttr and xmlText escape values going into the query we build by hand.
// Dish comments are typed by guests, so this is not decoration.
func xmlAttr(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func xmlText(s string) string { return xmlAttr(s) }
