package menuimport

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Reading a menu that is inside the page but not inside one tidy blob.
//
// ⚠️ **`__NEXT_DATA__` is the old Next.js.** A Pages Router app ships its data
// as one `<script id="__NEXT_DATA__" type="application/json">`, which is a
// single document and is what FromInline was written against. The App Router —
// what every Next.js site built since has used — streams instead: dozens of
//
//	self.__next_f.push([1,"<a chunk of a string>"])
//
// calls whose decoded contents, concatenated, make up the payload. That payload
// is **not** a JSON document. It is React's flight format: numbered lines,
// component references, and — sitting inside it, whole — the JSON the server
// handed the page.
//
// ⚠️ **So this is a scanner, not a parser, and deliberately.** Writing a flight
// decoder would tie us to the internals of a format with no specification and
// no compatibility promise, which changes with the framework. What does not
// change is that the restaurant's dishes are in there as ordinary JSON. So the
// text is swept for balanced JSON values and every one that parses is handed to
// the same walker that reads every other menu — the reader stays one reader.
//
// This was found on a live import that returned nothing: yamato.delever.uz
// carries **119 dishes and 20 categories** in its page, and every reader we had
// looked straight past them. Delever runs a large share of the delivery sites
// in the country, so this is not one restaurant.

// nextFlight pulls the App Router's streamed payload out of a page.
//
// Returns "" when the page is not one — the ordinary answer for three sites in
// four, and not a failure.
var flightChunk = regexp.MustCompile(
	`(?s)self\.__next_f\.push\(\s*\[\s*\d+\s*,\s*("(?:[^"\\]|\\.)*")`)

func nextFlight(page string) string {
	var b strings.Builder
	for _, m := range flightChunk.FindAllStringSubmatch(page, -1) {
		var chunk string
		// ⚠️ Decoded as a JSON string rather than unescaped by hand: the
		// payload is full of `\"` and `\n`, and the dish names are full of
		// apostrophes and Cyrillic. Getting this wrong does not fail — it
		// produces a slightly corrupted document that parses to fewer dishes.
		if json.Unmarshal([]byte(m[1]), &chunk) != nil {
			continue
		}
		b.WriteString(chunk)
	}
	return b.String()
}

// jsonValuesIn sweeps text for complete JSON objects and arrays.
//
// ⚠️ **Balanced, by the decoder, not by counting braces.** A brace counter is
// the obvious implementation and it is wrong on the first dish whose
// description contains a `}` — which, in a menu, is never; and on the first one
// containing a quoted brace, which is eventually. `json.Decoder` already knows
// where a value ends, including inside strings, so it is asked.
//
// ⚠️ Only `{"` and `[{` start a candidate. Every opening brace would make this
// quadratic on a 250 KB page for nothing: a JSON value worth reading has a
// key or an object in it, and a bare `[1,2]` is not a menu.
func jsonValuesIn(text string) []json.RawMessage {
	var out []json.RawMessage
	for i := 0; i < len(text); {
		j := nextCandidate(text, i)
		if j < 0 {
			break
		}
		dec := json.NewDecoder(strings.NewReader(text[j:]))
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			// Not the start of a value after all — a `{"` inside a string, a
			// truncated tail. Step past it and keep sweeping.
			i = j + 2
			continue
		}
		out = append(out, v)
		// ⚠️ Resume **after** the value, not inside it. Sweeping the inside
		// again would return every nested object as its own candidate: the
		// same dish a dozen times, and a page's worth of parsing per dish.
		i = j + int(dec.InputOffset())
		if len(out) >= maxEmbeddedValues {
			break
		}
	}
	return out
}

// A page cannot need more than this many separate JSON values to state a menu.
// The cap is a runaway guard, not a limit anybody should reach: the live page
// this was written for yields twenty-one.
const maxEmbeddedValues = 4000

func nextCandidate(text string, from int) int {
	obj := strings.Index(text[from:], `{"`)
	arr := strings.Index(text[from:], `[{`)
	switch {
	case obj < 0 && arr < 0:
		return -1
	case obj < 0:
		return from + arr
	case arr < 0:
		return from + obj
	default:
		return from + min(obj, arr)
	}
}

// FromEmbedded reads a menu out of JSON embedded anywhere in the page.
//
// ⚠️ **Tried after the tidy blobs and before the site's API**, which is the
// order of certainty: a document that says it is the page's data is better
// evidence than a value that merely parses, and both are better than a guessed
// URL. It is still free and needs no second request — the dishes are in the
// page we already have.
func FromEmbedded(page string) []Dish {
	text := nextFlight(page)
	if strings.TrimSpace(text) == "" {
		// Not an App Router page. The rest of the sweep still applies: plenty
		// of sites simply assign a menu to a variable in an inline script,
		// with no framework and no wrapper we could name.
		text = page
	}
	values := jsonValuesIn(text)
	if len(values) == 0 {
		return nil
	}

	// ⚠️ **Two passes, and the first one is the categories.** These payloads
	// keep the category list and the product list as *siblings*, with products
	// naming their category by id — so the walker's "nearest enclosing name"
	// rule, which is right for a nested menu, finds nothing at all here. Every
	// dish would import with no section and the owner would sort a hundred and
	// nineteen of them by hand.
	src := &catalog{names: map[string]string{}, imageBase: imageBase(page)}
	for _, v := range values {
		var doc any
		if json.Unmarshal(v, &doc) != nil {
			continue
		}
		indexNames(doc, src)
	}

	var out []Dish
	for _, v := range values {
		var doc any
		if json.Unmarshal(v, &doc) != nil {
			continue
		}
		walkJSON(doc, "", src, &out)
	}
	return dedupe(out)
}

// uuidTail matches an address whose last segment is a uuid — the shape these
// platforms serve images at.
var uuidTail = regexp.MustCompile(
	`https?://[^\s"'\\<>]+/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// imageBase learns how this site turns an image id into an address.
//
// ⚠️ **A menu API that stores `"image": "553fb012-…"` is publishing an id, not a
// photograph**, and no amount of URL-shaped checking will make it one. The site
// itself knows the answer and states it in its own markup: its favicon, its
// logo and its open-graph card are all served from the same CDN under the same
// path. So the shape is read off the page rather than guessed — and if the page
// never shows one, dishes import without photographs, which is the honest
// outcome and not a broken link on every card.
func imageBase(page string) string {
	m := uuidTail.FindString(page)
	if m == "" {
		return ""
	}
	cut := strings.LastIndex(m, "/")
	if cut < 0 {
		return ""
	}
	return m[:cut+1]
}
