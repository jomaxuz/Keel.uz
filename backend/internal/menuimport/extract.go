package menuimport

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// Turning a page into dishes.
//
// ⚠️ **Structured data first, the model second, and never the other way
// round.** Most menu pages carry schema.org JSON-LD — an aggregator's listing
// almost always does, because it is what puts them in Google's results. Parsing
// that is exact and free: the price is the price the site published, not a
// number read out of a sentence. Asking a model to read prices off a page that
// already states them is strictly worse and costs money to be worse.
//
// The model is for the pages that have nothing: a photograph of a menu turned
// into a web page, a restaurant's own site built by hand, a Telegram channel.

// Dish is one proposed menu item, before anybody has agreed to it.
type Dish struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// In so'm, whole. ⚠️ Zero means "no price found", which the panel shows as
	// an empty box rather than as free: a dish imported at 0 that nobody
	// noticed is a dish the restaurant gives away.
	Price    int    `json:"price"`
	ImageURL string `json:"imageUrl,omitempty"`
	Category string `json:"category,omitempty"`
}

// Result is a whole proposed menu.
type Result struct {
	Dishes []Dish `json:"dishes"`
	// Where it came from, so the panel can say what it read.
	Source string `json:"source"`
	// Whether the model was needed. Shown to the owner, because it changes how
	// carefully they should read the list.
	Guessed bool `json:"guessed"`
}

// FromStructured reads schema.org data out of a page.
//
// Returns nothing rather than an error when the page has none: that is the
// ordinary case and the caller's next step is the model, not a message.
func FromStructured(page string) []Dish {
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		return nil
	}
	var out []Dish
	for _, blob := range jsonLDBlocks(doc) {
		out = append(out, dishesFromLD(blob, "")...)
	}
	return dedupe(out)
}

func jsonLDBlocks(n *html.Node) []json.RawMessage {
	var out []json.RawMessage
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			for _, a := range n.Attr {
				if strings.EqualFold(a.Key, "type") &&
					strings.EqualFold(strings.TrimSpace(a.Val), "application/ld+json") {
					if n.FirstChild != nil {
						out = append(out, json.RawMessage(n.FirstChild.Data))
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

// ldNode is the handful of schema.org shapes a menu ever arrives in. Held
// loosely on purpose: sites disagree about almost everything except the names
// of these fields.
type ldNode struct {
	Type        any             `json:"@type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Image       any             `json:"image"`
	Offers      json.RawMessage `json:"offers"`
	Price       any             `json:"price"`
	// Menu → hasMenuSection → hasMenuItem, and the graph/list wrappers.
	HasMenuSection json.RawMessage `json:"hasMenuSection"`
	HasMenuItem    json.RawMessage `json:"hasMenuItem"`
	HasMenu        json.RawMessage `json:"hasMenu"`
	ItemListElem   json.RawMessage `json:"itemListElement"`
	Item           json.RawMessage `json:"item"`
	Graph          json.RawMessage `json:"@graph"`
}

func dishesFromLD(raw json.RawMessage, section string) []Dish {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return nil
	}
	// A list at any level: pages publish arrays of products as often as single
	// objects, and the wrapper is not consistent between two sites.
	if raw[0] == '[' {
		var list []json.RawMessage
		if json.Unmarshal(raw, &list) != nil {
			return nil
		}
		var out []Dish
		for _, item := range list {
			out = append(out, dishesFromLD(item, section)...)
		}
		return out
	}

	var n ldNode
	if json.Unmarshal(raw, &n) != nil {
		return nil
	}

	var out []Dish
	// Containers, walked before the node itself is considered a dish: a
	// MenuSection has a name and no price, and taking it as a dish would put
	// "Salatlar" on the menu at nothing so'm.
	next := section
	if hasType(n.Type, "MenuSection") && n.Name != "" {
		next = n.Name
	}
	for _, child := range []json.RawMessage{
		n.Graph, n.HasMenu, n.HasMenuSection, n.HasMenuItem, n.ItemListElem, n.Item,
	} {
		out = append(out, dishesFromLD(child, next)...)
	}

	if name := strings.TrimSpace(n.Name); name != "" && looksLikeDish(n) {
		out = append(out, Dish{
			Name:        clean(name),
			Description: clean(strings.TrimSpace(n.Description)),
			Price:       priceFromLD(n),
			ImageURL:    firstImage(n.Image),
			Category:    clean(section),
		})
	}
	return out
}

// looksLikeDish decides whether a node is something sold.
//
// ⚠️ **A price is not required, and requiring one was wrong.** Plenty of menu
// pages list the dish and put the price in a separate element the JSON-LD does
// not carry. Dropping those loses the names, descriptions and photographs —
// the parts that actually take an hour to type — over the one field that takes
// five seconds.
func looksLikeDish(n ldNode) bool {
	return hasType(n.Type, "MenuItem") || hasType(n.Type, "Product") ||
		hasType(n.Type, "Offer") || hasType(n.Type, "Dish")
}

func hasType(v any, want string) bool {
	switch t := v.(type) {
	case string:
		return strings.EqualFold(t, want)
	case []any:
		for _, one := range t {
			if s, ok := one.(string); ok && strings.EqualFold(s, want) {
				return true
			}
		}
	}
	return false
}

func priceFromLD(n ldNode) int {
	if p := toPrice(n.Price); p > 0 {
		return p
	}
	if len(n.Offers) == 0 {
		return 0
	}
	// Offers is an object, a list of objects, or an AggregateOffer with a low
	// price inside. All three appear on real pages.
	var offers []struct {
		Price      any `json:"price"`
		LowPrice   any `json:"lowPrice"`
		PriceSpec  any `json:"priceSpecification"`
		Currency   any `json:"priceCurrency"`
		Additional any `json:"offers"`
	}
	trimmed := strings.TrimSpace(string(n.Offers))
	if strings.HasPrefix(trimmed, "[") {
		_ = json.Unmarshal(n.Offers, &offers)
	} else {
		offers = offers[:0]
		var one struct {
			Price      any `json:"price"`
			LowPrice   any `json:"lowPrice"`
			PriceSpec  any `json:"priceSpecification"`
			Currency   any `json:"priceCurrency"`
			Additional any `json:"offers"`
		}
		if json.Unmarshal(n.Offers, &one) == nil {
			offers = append(offers, one)
		}
	}
	for _, o := range offers {
		if p := toPrice(o.Price); p > 0 {
			return p
		}
		if p := toPrice(o.LowPrice); p > 0 {
			return p
		}
	}
	return 0
}

var priceDigits = regexp.MustCompile(`[0-9]+`)

// toPrice turns whatever a page called a price into whole so'm.
//
// ⚠️ **Separators, and this is where a menu quietly becomes free.** Uzbek sites
// write forty-five thousand as `45 000`, `45,000`, `45.000` and `45000`, and
// two of those are a decimal point somewhere else in the world. Parsing
// `45.000` as a float gives **45**, which imports as forty-five so'm — a price
// nobody reads as wrong because it is a plausible number in a box.
//
// So: every digit group is joined and nothing is treated as a decimal, because
// menu prices in this currency have no fractional part. `45.000` and `45,000`
// and `45 000` all mean 45000, and a genuine `45000.00` is not something any
// menu here writes.
func toPrice(v any) int {
	var s string
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case string:
		s = t
	default:
		return 0
	}
	groups := priceDigits.FindAllString(s, -1)
	if len(groups) == 0 {
		return 0
	}
	joined := strings.Join(groups, "")
	// A trailing ",00" or ".00" is a currency's minor unit, not part of the
	// figure. Only ever two zeros, and only at the end.
	if len(groups) > 1 && groups[len(groups)-1] == "00" &&
		(strings.Contains(s, ".00") || strings.Contains(s, ",00")) {
		joined = strings.Join(groups[:len(groups)-1], "")
	}
	n, err := strconv.Atoi(joined)
	if err != nil {
		return 0
	}
	return n
}

func firstImage(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		for _, one := range t {
			if s := firstImage(one); s != "" {
				return s
			}
		}
	case map[string]any:
		if s, ok := t["url"].(string); ok {
			return s
		}
		if s, ok := t["contentUrl"].(string); ok {
			return s
		}
	}
	return ""
}

var spaces = regexp.MustCompile(`\s+`)

func clean(s string) string {
	return strings.TrimSpace(spaces.ReplaceAllString(html.UnescapeString(s), " "))
}

// dedupe keeps one of each dish.
//
// ⚠️ Pages list the same dish more than once — a "popular" carousel above the
// menu it is taken from is the usual reason — and importing it twice puts two
// identical rows in front of the owner to delete by hand.
func dedupe(in []Dish) []Dish {
	seen := map[string]bool{}
	out := make([]Dish, 0, len(in))
	for _, d := range in {
		key := strings.ToLower(d.Name) + "|" + strconv.Itoa(d.Price)
		if d.Name == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, d)
	}
	return out
}
