package handlers

import (
	crand "crypto/rand"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var primitiveNil = primitive.NilObjectID

func objectID(s string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(s)
}

// oidOf extracts an ObjectID from an InsertOne result's InsertedID.
func oidOf(v any) primitive.ObjectID {
	if id, ok := v.(primitive.ObjectID); ok {
		return id
	}
	return primitive.NilObjectID
}

// orderNumber returns a short human-readable order number, e.g. "K7F3-2M9P".
//
// ⚠️ **This is a capability, so it is drawn from crypto/rand — not math/rand,
// and not the clock.** `GET /orders/{number}` is public and keyed by nothing
// but this string, and it returns the customer's full address, their map pin
// and the courier's phone. That is the deliberate "the link is the key" design
// the payment flow also uses — but a key is only a key if it cannot be guessed.
//
// The old number leaked on both counts: two of its characters were
// `time.Now().Unix()%100`, which anybody watching the clock could narrow to a
// handful of values, and the rest came from an unseeded math/rand shared across
// the process. An attacker did not need a victim's number; they could walk the
// small space and read strangers' addresses. Every character now comes from a
// CSPRNG over an unambiguous 8-character alphabet — ~30^8 ≈ 6.5e11 — so walking
// it is not worth anybody's time, and the string stays short enough to read
// down a phone.
func orderNumber() string {
	// No I/O/0/1 — a number read aloud or copied off a receipt must not turn
	// into a different valid one.
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 8)
	if _, err := crand.Read(b); err != nil {
		// crypto/rand failing is not a condition to paper over with a weaker
		// source: a guessable order number is exactly what this avoids.
		panic("order number: no randomness: " + err.Error())
	}
	out := make([]byte, 0, 9)
	for i, v := range b {
		if i == 4 {
			out = append(out, '-')
		}
		out = append(out, alphabet[int(v)%len(alphabet)])
	}
	return string(out)
}

// branchOrderNumber prefixes an order number with the branch's short code, so a
// number read aloud on the phone says which kitchen it belongs to before anyone
// has to look it up: "CHL-A71-4509" is Chilonzor's.
//
// Empty on a single-branch install, where the prefix would only add noise.
func branchOrderNumber(code string) string {
	code = strings.ToUpper(clampText(code, 6))
	code = strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, code)
	if code == "" {
		return orderNumber()
	}
	return code + "-" + orderNumber()
}

// clampText trims free text from a client and caps its length in runes, so a
// pasted essay cannot bloat an order document.
func clampText(s string, max int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > max {
		return strings.TrimSpace(string(r[:max]))
	}
	return s
}

// haversineKm returns the great-circle distance between two coordinates in km.
func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// pointInPolygon uses the ray-casting algorithm. Polygon points are [lat, lng].
func pointInPolygon(lat, lng float64, polygon [][]float64) bool {
	inside := false
	n := len(polygon)
	if n < 3 {
		return false
	}
	j := n - 1
	for i := 0; i < n; i++ {
		yi, xi := polygon[i][0], polygon[i][1]
		yj, xj := polygon[j][0], polygon[j][1]
		if (xi > lng) != (xj > lng) &&
			lat < (yj-yi)*(lng-xi)/(xj-xi)+yi {
			inside = !inside
		}
		j = i
	}
	return inside
}

// deliveryQuote is what an address costs to deliver to, from one origin.
type deliveryQuote struct {
	Fee        int     `json:"deliveryFee"`
	Zone       string  `json:"zone"`
	DistanceKm float64 `json:"distanceKm"`
	Available  bool    `json:"available"`
	MinOrder   int     `json:"minOrder"`
}

// quoteDeliveryFrom prices a destination against one kitchen's rules.
//
// Delivery settings belong to a branch, not the company: one branch may close
// at 21:00 and reach 3 km, another may cover half the city. The origin is that
// branch's own address, which is what the radius model and per-km zones measure
// from.
func quoteDeliveryFrom(d models.DeliverySettings, origin models.GeoPoint, lat, lng float64, subtotal int) deliveryQuote {
	fee, zone, km, ok := quoteDeliveryRules(d, origin, lat, lng, subtotal)
	return deliveryQuote{
		Fee: fee, Zone: zone, DistanceKm: km, Available: ok, MinOrder: d.MinOrder,
	}
}

func quoteDeliveryRules(d models.DeliverySettings, origin models.GeoPoint, lat, lng float64, subtotal int) (fee int, zone string, km float64, ok bool) {
	if !d.Enabled {
		return 0, "", 0, false
	}
	// Only zones that are actually drawable count — a half-finished zone must
	// not disable delivery everywhere.
	zones := make([]models.DeliveryZone, 0, len(d.Zones))
	for _, z := range d.Zones {
		if len(z.Polygon) >= 3 {
			zones = append(zones, z)
		}
	}
	// Explicit mode wins; an empty mode (older documents) infers "zones when
	// some are drawn". Choosing zones without drawing any falls back to the
	// radius model rather than refusing every address.
	useZones := len(zones) > 0 && d.Mode != "radius"
	if useZones {
		for _, z := range zones {
			if pointInPolygon(lat, lng, z.Polygon) {
				zone, ok = z.Name, true
				if z.Pricing == "perKm" {
					km = haversineKm(origin.Lat, origin.Lng, lat, lng)
					fee = z.BaseFee + int(math.Ceil(km))*z.PerKm
				} else {
					fee = z.Fee
				}
				break
			}
		}
		if !ok {
			return 0, "", 0, false // outside all zones
		}
	} else {
		// Radius model.
		km = haversineKm(origin.Lat, origin.Lng, lat, lng)
		if d.MaxKm > 0 && km > d.MaxKm {
			return 0, "", km, false
		}
		fee = d.BaseFee + int(math.Ceil(km))*d.PerKm
		ok = true
	}
	// Free delivery threshold.
	if d.FreeDeliveryFrom != nil && subtotal >= *d.FreeDeliveryFrom {
		fee = 0
	}
	return fee, zone, km, ok
}

// resolveOptions validates the choices a client sent for a dish against the
// menu and returns them with the current price deltas attached. The client is
// never trusted with prices, and required groups must be answered.
func resolveOptions(item *models.MenuItem, sel []models.OrderItemOption) ([]models.OrderItemOption, error) {
	out := make([]models.OrderItemOption, 0, len(sel))
	seen := map[string]int{} // option group -> how many choices picked

	for _, s := range sel {
		group, ok := findOption(item.Options, s.Name)
		if !ok {
			return nil, fmt.Errorf("%s: \"%s\" varianti menyuda yo'q — savatni yangilang", item.Name, s.Name)
		}
		choice, ok := findChoice(group.Choices, s.Choice)
		if !ok {
			return nil, fmt.Errorf("%s: \"%s\" uchun \"%s\" tanlovi yo'q — savatni yangilang", item.Name, s.Name, s.Choice)
		}
		if seen[group.Name] > 0 && !group.Multiple {
			return nil, fmt.Errorf("%s: \"%s\" uchun faqat bitta variant tanlanadi", item.Name, group.Name)
		}
		seen[group.Name]++
		out = append(out, models.OrderItemOption{
			Name:       group.Name,
			Choice:     choice.Name,
			PriceDelta: choice.PriceDelta,
		})
	}

	for _, g := range item.Options {
		if g.Required && len(g.Choices) > 0 && seen[g.Name] == 0 {
			return nil, fmt.Errorf("%s: \"%s\" tanlanmagan", item.Name, g.Name)
		}
	}
	return out, nil
}

func findOption(opts []models.MenuOption, name string) (models.MenuOption, bool) {
	for _, o := range opts {
		if o.Name == name {
			return o, true
		}
	}
	return models.MenuOption{}, false
}

func findChoice(choices []models.OptionChoice, name string) (models.OptionChoice, bool) {
	for _, c := range choices {
		if c.Name == name {
			return c, true
		}
	}
	return models.OptionChoice{}, false
}

// textSearch builds a case-insensitive "contains" matcher for free text typed
// by an operator. The input is escaped: a stray "(" must not silently match
// nothing (or worse, blow up the query).
func textSearch(q string) bson.M {
	return bson.M{"$regex": regexp.QuoteMeta(strings.TrimSpace(q)), "$options": "i"}
}

// normalizeUsername is the one spelling a login is ever stored or looked up
// under. Case and stray spaces are a phone keyboard's doing, not the user's.
func normalizeUsername(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// formatUZS renders a sum the way the audit log reads it back: "450 000 so'm".
func formatUZS(v int) string {
	s := strconv.Itoa(v)
	if v < 0 {
		s = strconv.Itoa(-v)
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ' ')
		}
		out = append(out, c)
	}
	if v < 0 {
		return "-" + string(out) + " so'm"
	}
	return string(out) + " so'm"
}
