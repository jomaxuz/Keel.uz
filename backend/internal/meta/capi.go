package meta

// ---- The Conversions API: telling Meta what actually got ordered ----
//
// ⚠️ **This is the whole advantage.** A browser pixel sees somebody reach a
// page; this server sees the order that was cooked, its total, and whether it
// was cancelled afterwards. Reporting the second one is why this section can
// answer "the advertising brought forty-seven orders" instead of "three hundred
// clicks", and it is the one thing a targetolog with access to the ads account
// and nothing else cannot do at all.
//
// ⚠️ **What leaves this server is hashed, and it is the guest's data.** The
// phone number and the name belong to a person who ordered dinner, not to Meta:
// they travel as SHA-256 of a normalised string, which is what Meta matches on
// and all it ever receives. The normalisation is theirs, copied exactly — a
// lowercase that we skip is a match that silently never happens, and the
// feature then looks like it does not work rather than like it is misconfigured.
//
// ⚠️ **`event_id` is the order number.** The same order can reach Meta twice —
// from the browser pixel and from here — and Meta collapses the pair only when
// both carry the same id. Without it the restaurant's own report shows every
// order twice, which is worse than showing none: it is a number an owner will
// believe.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// UserData is who ordered, in the form Meta matches on.
//
// ⚠️ Every field here is already hashed by the time it is in this struct — see
// `Person`. A raw value assigned by hand would be sent raw, and Meta accepts it
// without complaint.
type UserData struct {
	Email      []string `json:"em,omitempty"`
	Phone      []string `json:"ph,omitempty"`
	FirstName  []string `json:"fn,omitempty"`
	City       []string `json:"ct,omitempty"`
	Country    []string `json:"country,omitempty"`
	ExternalID []string `json:"external_id,omitempty"`
	// Not hashed, and Meta says so explicitly: these identify the browser
	// rather than the person, and hashing them would simply break the match.
	ClientIP  string `json:"client_ip_address,omitempty"`
	UserAgent string `json:"client_user_agent,omitempty"`
	FBC       string `json:"fbc,omitempty"`
	FBP       string `json:"fbp,omitempty"`
}

// hash normalises and hashes one value, or returns nothing for an empty one.
//
// ⚠️ **An empty string has a SHA-256 too**, and sending it is sending a field
// that matches nobody while looking like data. Absent is the correct shape.
func hash(s string) []string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(s))
	return []string{hex.EncodeToString(sum[:])}
}

// phoneDigits is Meta's phone rule: digits only, country code included, no
// leading zeros.
//
// ⚠️ **Uzbek numbers are stored several ways** — `+998901234567`,
// `998901234567`, `90 123 45 67` — and all three are one person. A number that
// reaches Meta without the country code matches nobody, and the failure is
// invisible: the event is accepted, the order is simply never attributed.
func phoneDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := strings.TrimLeft(b.String(), "0")
	// A bare Uzbek mobile number — 90 123 45 67 — is nine digits and needs the
	// country code Meta matches on. Anything longer already carries one.
	if len(d) == 9 {
		d = "998" + d
	}
	return d
}

// Person builds the hashed identity of one guest.
func Person(name, phone, city string) UserData {
	u := UserData{Country: hash("uz")}
	if p := phoneDigits(phone); p != "" {
		u.Phone = hash(p)
		// ⚠️ The phone doubles as the external id because it is what this
		// system identifies a returning guest by. It lets Meta connect two
		// orders from one person without us sending anything more about them.
		u.ExternalID = hash(p)
	}
	// Only the first name: a full name is not what Meta matches on, and the
	// surname adds nothing but another field about somebody who ordered lunch.
	if first, _, _ := strings.Cut(strings.TrimSpace(name), " "); first != "" {
		u.FirstName = hash(first)
	}
	if city != "" {
		u.City = hash(strings.ReplaceAll(city, " ", ""))
	}
	return u
}

// CustomData is what was bought.
type CustomData struct {
	Value    float64 `json:"value"`
	Currency string  `json:"currency"`
	OrderID  string  `json:"order_id,omitempty"`
	// Which dishes, by the ids this system uses. ⚠️ The same ids the catalogue
	// would carry if the restaurant ever connects one — never invented codes.
	ContentIDs  []string `json:"content_ids,omitempty"`
	ContentType string   `json:"content_type,omitempty"`
}

// Event is one thing that happened.
type Event struct {
	EventName string     `json:"event_name"`
	EventTime int64      `json:"event_time"`
	EventID   string     `json:"event_id"`
	SourceURL string     `json:"event_source_url,omitempty"`
	Source    string     `json:"action_source"`
	UserData  UserData   `json:"user_data"`
	Custom    CustomData `json:"custom_data"`
}

// SendResult is Meta's receipt for a batch.
type SendResult struct {
	EventsReceived int    `json:"events_received"`
	FBTraceID      string `json:"fbtrace_id"`
}

// SendEvents posts a batch to one pixel.
//
// ⚠️ **Batched, because the quota is per call rather than per event.** A
// restaurant closing sixty orders an hour would otherwise spend sixty calls of
// somebody else's hourly ceiling on bookkeeping.
func (c *Client) SendEvents(ctx context.Context, pixelID string, events []Event, testCode string) (SendResult, error) {
	body := map[string]any{"data": events}
	if testCode != "" {
		// Meta only shows an event in the Test Events tab when this is set, and
		// it must be absent in production — an event sent with it is not
		// counted towards anything.
		body["test_event_code"] = testCode
	}
	var out SendResult
	err := c.PostJSON(ctx, pixelID+"/events", body, &out)
	return out, err
}
