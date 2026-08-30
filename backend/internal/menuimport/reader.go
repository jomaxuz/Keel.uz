package menuimport

import "context"

// The readers, named and listed.
//
// ⚠️ **Every site is a different shape, so the shapes are written down rather
// than discovered.** One restaurant publishes schema.org markup, the next ships
// its menu inside a JavaScript bundle's state, the next fetches it from its own
// JSON API, and the one after that has a page of plain HTML somebody typed. A
// chain of `if len(dishes) == 0` hidden inside a handler is the same logic with
// none of that said out loud — and it cannot tell an owner *why* their page
// worked, or which of four things to try when it did not.
//
// So each is a named reader with an order, and the answer says which one read
// the page. That is the difference between "import did not work" and "this page
// has nothing published on it, use the file import".

// ReaderID identifies how a page was read.
const (
	// schema.org JSON-LD in the page. What puts an aggregator into Google's
	// results, so aggregators almost always carry it.
	ReaderStructured = "structured"
	// The framework's own state blob — `__NEXT_DATA__`, `__NUXT__`. The whole
	// menu already inside the document we downloaded.
	ReaderInline = "inline"
	// JSON sitting loose in the page rather than in a named blob. ⚠️ What a
	// Next.js App Router site ships: dozens of streamed `self.__next_f.push`
	// chunks whose contents are not a JSON document but do contain the menu.
	ReaderEmbedded = "embedded"
	// The site's own JSON menu API. ⚠️ The ordinary case for anything built
	// this decade: the page that arrives is an empty shell.
	ReaderAPI = "api"
	// The assistant, on the page's text. ⚠️ **Last, always.** Every reader
	// above it reads numbers the site published; this one reads a sentence. It
	// is for a photograph of a menu turned into a web page.
	ReaderText = "text"
)

// Reader is one way of getting dishes off a page.
type Reader struct {
	ID string
	// What the panel calls it, in the owner's terms rather than ours.
	Label string
	// Whether it needs the platform's assistant. Kept on the reader so the
	// panel can say "this one costs a call" without knowing which id that is.
	Assisted bool
	// Read returns the dishes, or nothing if this reader does not apply.
	// ⚠️ Nothing rather than an error: "this page has no JSON-LD" is not a
	// failure, it is the ordinary answer for three of the four.
	Read func(ctx context.Context, page, url string) []Dish
}

// Readers is every way a page can be read, in the order they are tried.
//
// ⚠️ **Ordered by how certain the answer is, not by how likely it is to
// match.** The first three read published numbers — exact, free, no model. Only
// the last reads prose, and asking it to read a price that is sitting in a JSON
// field two readers up would be paying to be less accurate.
//
// The assisted reader is not in this list: it needs the handler's link to the
// platform, so it is appended by the caller. Everything here is offline.
func Readers() []Reader {
	return []Reader{
		{
			ID:    ReaderStructured,
			Label: "Sahifadagi schema.org ma'lumoti",
			Read: func(_ context.Context, page, _ string) []Dish {
				return FromStructured(page)
			},
		},
		{
			ID:    ReaderInline,
			Label: "Sahifa ichidagi JavaScript ma'lumoti",
			Read: func(_ context.Context, page, _ string) []Dish {
				return FromInline(page)
			},
		},
		{
			ID:    ReaderEmbedded,
			Label: "Sahifa ichidagi menyu ma'lumoti (yangi Next.js saytlari)",
			Read: func(_ context.Context, page, _ string) []Dish {
				return FromEmbedded(page)
			},
		},
		{
			ID:    ReaderAPI,
			Label: "Saytning o'z JSON API'si",
			Read: func(ctx context.Context, _, u string) []Dish {
				return FromSiteAPI(ctx, u)
			},
		},
	}
}

// ReaderLabel names a reader for the panel, including the assisted one.
func ReaderLabel(id string) string {
	for _, r := range Readers() {
		if r.ID == id {
			return r.Label
		}
	}
	if id == ReaderText {
		return "Matnni AI o'qidi"
	}
	return id
}
