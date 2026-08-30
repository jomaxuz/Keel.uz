package menuimport

import (
	"strings"

	"golang.org/x/net/html"
)

// What is sent to the model when the page has no structured data.
//
// ⚠️ **The page is reduced first, and not to save money — though it does.** A
// menu page is 300 KB of markup around 4 KB of dishes: navigation, cookie
// banners, footers, three scripts and a chat widget. Sent whole it does not fit
// in a request, and the part that would be cut is the part at the bottom, which
// is where the last third of the menu is.

// maxText is what the model is given. Enough for a long menu, small enough that
// the request is one call.
const maxText = 24000

// PageText strips a page to the words a person would read.
//
// ⚠️ Images are kept as bare URLs on their own line, because a photograph per
// dish is half the value of importing at all — and the model can only pair a
// picture with a name if it can see both. Only images that are plausibly a
// dish: a page's logo, icons and tracking pixels are noise that would be paired
// with whatever dish they happened to sit near.
func PageText(page string) string {
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "noscript", "svg", "nav", "footer", "header":
				// Not read by a person and never a dish.
				return
			case "img":
				if src := plausibleImage(n); src != "" {
					b.WriteString("\n[img] " + src + "\n")
				}
			case "br", "p", "div", "li", "tr", "h1", "h2", "h3", "h4", "section":
				b.WriteString("\n")
			}
		}
		if n.Type == html.TextNode {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				b.WriteString(t + " ")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// Blank lines collapse; a menu read as text is a list, and forty empty
	// lines between two dishes is forty lines of budget.
	lines := []string{}
	for _, l := range strings.Split(b.String(), "\n") {
		if t := strings.TrimSpace(l); t != "" {
			lines = append(lines, t)
		}
	}
	out := strings.Join(lines, "\n")
	if len(out) > maxText {
		out = out[:maxText]
	}
	return out
}

// plausibleImage keeps the pictures that could be food.
//
// ⚠️ Guessing from the URL rather than fetching each one. Fetching a hundred
// images to measure them, before the owner has agreed to import anything, is
// a hundred requests to somebody else's server for a preview that may be
// discarded.
func plausibleImage(n *html.Node) string {
	var src, alt string
	for _, a := range n.Attr {
		switch strings.ToLower(a.Key) {
		case "src":
			if src == "" {
				src = a.Val
			}
		// Lazy-loaded pages put the real address here and a placeholder in
		// `src` — taking `src` on those imports the same grey square as every
		// dish's photograph.
		case "data-src", "data-original", "data-lazy-src":
			src = a.Val
		case "alt":
			alt = a.Val
		}
	}
	src = strings.TrimSpace(src)
	if src == "" || strings.HasPrefix(src, "data:") {
		return ""
	}
	low := strings.ToLower(src + " " + alt)
	for _, junk := range []string{
		"logo", "icon", "favicon", "sprite", "placeholder", "avatar",
		"banner", "pixel", "tracking", "spacer", "flag", "arrow",
	} {
		if strings.Contains(low, junk) {
			return ""
		}
	}
	return src
}
