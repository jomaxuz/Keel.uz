package menuimport

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
)

// Readers for the aggregators restaurants around here actually list on.
//
// ⚠️ **These pages publish nothing at all — not even text.** Yandex Eats sends
// 122 KB of markup with no dish, no price and no readable sentence in it: the
// menu arrives afterwards, from the site's own API, into a page the browser
// draws. So all four general readers correctly find nothing, the model is then
// handed an empty string, and the owner is told their page has no menu on it
// while looking at a screen full of dishes. Correct, useless, and indis-
// tinguishable from a broken import — which is exactly how it was reported.
//
// ⚠️ **The generic API reader cannot find these either, and it is not a bug in
// it.** It tries `/api/v1/menu` and five siblings at the site root, which is
// where a restaurant's own site keeps its menu. An aggregator's menu lives at
// an address containing *the restaurant's* slug, and no amount of guessing at
// paths gets there without knowing that the slug exists and where to read it.
//
// ⚠️ **The slug in the path is not always the slug the API wants.** For
// `/en-uz/tashkent/r/sam_plov_restaurant?placeSlug=sam_plov` the API answers to
// `sam_plov` and returns 404 for `sam_plov_restaurant`. The query parameter
// wins where it exists — and it is present precisely because the two differ.
// Reading the path alone gives an empty menu for a link that is perfectly good.
//
// ⚠️ **Named per site, and it stays that way.** Every aggregator has its own
// address shape and its own JSON, and they change on their own schedule. A
// named list says which ones are known to work, so a page that fails can be
// answered with "this site is not one of the four" rather than a shrug.

// aggSite is one aggregator we know how to read.
type aggSite struct {
	// What the panel calls it — the owner's word for the site, not ours.
	Name string
	// Whether this site can answer for the page in hand.
	Matches func(u *url.URL) bool
	// Dishes, or nothing when the page is on this site but says nothing.
	Read func(ctx context.Context, u *url.URL) []Dish
}

func aggSites() []aggSite {
	return []aggSite{yandexEats()}
}

// FromAggregator reads a listing on a delivery aggregator.
//
// Returns the dishes and the site's name, so the panel can say "Yandex Eats"
// rather than "the aggregator reader".
func FromAggregator(ctx context.Context, pageURL string) ([]Dish, string) {
	u, err := url.Parse(pageURL)
	if err != nil {
		return nil, ""
	}
	for _, s := range aggSites() {
		if !s.Matches(u) {
			continue
		}
		// ⚠️ The name is returned even when the read is empty: "Yandex Eats
		// published nothing for this restaurant" and "we do not read Yandex
		// Eats" are different answers and lead somewhere different.
		return s.Read(ctx, u), s.Name
	}
	return nil, ""
}

// ---- Yandex Eats (eats.yandex.com / eda.yandex.ru) ----

func yandexEats() aggSite {
	return aggSite{
		Name: "Yandex Eats",
		Matches: func(u *url.URL) bool {
			h := strings.ToLower(u.Hostname())
			if !strings.Contains(h, ".yandex.") {
				return false
			}
			first, _, _ := strings.Cut(h, ".")
			return first == "eats" || first == "eda"
		},
		Read: func(ctx context.Context, u *url.URL) []Dish {
			slug := yandexSlug(u)
			if slug == "" {
				return nil
			}
			api := u.Scheme + "://" + u.Host + "/api/v2/menu/retrieve/" + url.PathEscape(slug)
			body, _, err := Fetch(ctx, api)
			if err != nil {
				return nil
			}
			return yandexDishes(body, u.Scheme+"://"+u.Host)
		},
	}
}

// yandexSlug is the name the API answers to.
//
// ⚠️ `?placeSlug=` first. It is the one the application itself passes to the
// API, and where it differs from the path segment the path segment is a 404.
func yandexSlug(u *url.URL) string {
	if s := strings.TrimSpace(u.Query().Get("placeSlug")); s != "" {
		return s
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i, p := range parts {
		if p == "r" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	if n := len(parts); n > 0 {
		return parts[n-1]
	}
	return ""
}

// yandexDishes reads the menu document.
//
// ⚠️ Unavailable items are kept. This is a proposal the owner reads line by
// line, and a dish missing from it is a dish they have to type — while a dish
// they do not want is one tick. "Sold out on the aggregator today" is not
// "not on the menu".
func yandexDishes(body, origin string) []Dish {
	var doc struct {
		Payload struct {
			Categories []struct {
				Name  string `json:"name"`
				Items []struct {
					Name        string  `json:"name"`
					Description string  `json:"description"`
					Price       float64 `json:"price"`
					Picture     struct {
						URI string `json:"uri"`
					} `json:"picture"`
				} `json:"items"`
			} `json:"categories"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		return nil
	}

	out := []Dish{}
	for _, c := range doc.Payload.Categories {
		for _, it := range c.Items {
			name := strings.TrimSpace(it.Name)
			if name == "" {
				continue
			}
			out = append(out, Dish{
				Name:        name,
				Description: strings.TrimSpace(it.Description),
				Price:       int(it.Price),
				ImageURL:    yandexImage(it.Picture.URI, origin),
				Category:    strings.TrimSpace(c.Name),
			})
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// yandexImage turns the picture template into an address.
//
// ⚠️ The `uri` is a template, not a URL: `/images/207.../abc-{w}x{h}.jpeg`.
// Stored unfilled it is a broken image on every dish, and the break shows up
// after the import, in the menu, one placeholder per dish.
func yandexImage(uri, origin string) string {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return ""
	}
	// 600 is what the site's own cards ask for: large enough for our menu
	// card, and not the 2.8 MB original, ninety of which is a slow import and
	// a large disk.
	uri = strings.ReplaceAll(uri, "{w}x{h}", "600x600")
	uri = strings.ReplaceAll(uri, "{w}", "600")
	uri = strings.ReplaceAll(uri, "{h}", "600")
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return uri
	}
	return origin + "/" + strings.TrimPrefix(uri, "/")
}

// AggregatorName names the aggregator a link belongs to, without fetching.
//
// ⚠️ For the message, not the read. "Yandex Eats did not publish a menu for
// this restaurant" and "we do not read this site" send the owner to two
// different places, and only the first is worth their retrying.
func AggregatorName(pageURL string) string {
	u, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}
	for _, s := range aggSites() {
		if s.Matches(u) {
			return s.Name
		}
	}
	return ""
}
