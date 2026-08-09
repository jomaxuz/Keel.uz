package models

import (
	"slices"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// The page layout, as data.
//
// Until now the site was one template with nine knobs (colour, radius, font
// pairing…). This is the layout itself: which bands the page has, in what order,
// how wide each one is and how it looks. Drawn in the Keel console — see
// CONSTRUCTOR.md — and read by the tenant's own site.
//
// Three decisions shape the whole file:
//
//   - **A grid, not a canvas.** Width is measured in columns of twelve, never in
//     pixels. A block dragged to a pixel on a 1400 px screen cannot be
//     re-placed on a 380 px phone — there is no arithmetic that recovers what
//     the designer meant — and the same layout has to serve the Telegram mini
//     app. In columns the answer is trivial: on a phone everything is twelve.
//
//   - **Blocks carry no content.** A block names a *source* (`menu-grid` reads
//     the menu, `hours-address` reads the branch) and never stores words. The
//     site's copy is translated into three languages and lives in the
//     dictionary; a layout tool that could edit it would be a layout tool that
//     breaks Russian. It also makes a design portable: applying one restaurant's
//     layout to another is copying this list, with nothing to scrub out.
//
//   - **Every styling value is an enum or a small integer.** There is no
//     free-form CSS field anywhere, on purpose: this document is turned into
//     styles on a page, so a string field would be an injection hole with a
//     nice name. `Sanitize` drops anything it does not recognise rather than
//     passing it through.

// Block types. Stored, so never renamed.
//
// ⚠️ The first five are **not new** — they are the bands the home page already
// renders today, named. That is what makes the fallback honest: with no design
// at all, the renderer walks `DefaultSections()` and produces the page the
// customer already has, band for band. A block set that only covered the new
// ideas would have quietly dropped the perks strip from every existing site.
const (
	BlockHero         = "hero"          // name, tagline, cover, price-from
	BlockPerks        = "perks"         // the three-card strip under the hero
	BlockCategories   = "categories"    // category tiles
	BlockMenuGrid     = "menu-grid"     // dish cards: popular, or chosen categories
	BlockHoursAddress = "hours-address" // opening hours, address, phone
	BlockAbout        = "about"         // content.aboutTitle + aboutText
	BlockGallery      = "gallery"       // uploaded photographs
	BlockCTA          = "cta"           // order / book buttons
	// The site's own chrome, now drawable. Variants only — the links, the cart
	// and the language switch inside them are function, not decoration.
	BlockNavbar = "navbar"
	BlockFooter = "footer"
	// ⚠️ A band with **freely placed elements**: the answer to "make it look like
	// this picture". Everything else in this file is a band with a fixed inner
	// layout; this one is a box an operator draws inside.
	BlockCanvas = "canvas"
	// A popup, drawn like a canvas but shown over the page once per visit.
	BlockPopup = "popup"
)

// The variants each block offers. Kept here rather than in the frontend so the
// console, the renderer and `Sanitize` cannot drift apart.
var blockVariants = map[string][]string{
	BlockHero:         {"full", "split", "compact"},
	BlockPerks:        {"cards", "inline"},
	BlockCategories:   {"tiles", "list"},
	BlockMenuGrid:     {"cards", "rows"},
	BlockHoursAddress: {"map", "plain"},
	BlockAbout:        {"text", "text-image"},
	BlockGallery:      {"grid", "strip"},
	BlockCTA:          {"banner", "buttons"},
	BlockNavbar:       {"classic", "centered", "minimal", "transparent"},
	BlockFooter:       {"columns", "compact", "centered"},
	BlockCanvas:       {"free"},
	BlockPopup:        {"center", "bottom"},
}

// Element types inside a canvas. Stored, so never renamed.
const (
	ElText    = "text"    // a paragraph or heading, typed by the designer
	ElImage   = "image"   // an uploaded photograph
	ElButton  = "button"  // a link to one of the site's own pages
	ElBox     = "box"     // a coloured rectangle: the shape behind everything else
	ElDivider = "divider" // a rule
	// ⚠️ **Functional widgets keep their own insides.** A menu grid placed on a
	// canvas is positioned and sized freely, and what it renders is still the
	// menu — with prices, translations, options and the cart button that already
	// work. Rebuilding those out of text and boxes is how a "free" editor
	// produces a beautiful page that cannot take an order.
	ElMenu       = "widget-menu"
	ElCategories = "widget-categories"
	ElHours      = "widget-hours"
	ElMap        = "widget-map"
	ElCart       = "widget-cart"
)

var elementTypes = map[string]bool{
	ElText: true, ElImage: true, ElButton: true, ElBox: true, ElDivider: true,
	ElMenu: true, ElCategories: true, ElHours: true, ElMap: true, ElCart: true,
}

// Where a button may point. ⚠️ An allowlist, not a URL field: this value becomes
// an `href` on every visitor's page, and "any string" there is an open redirect
// with a nice name — plus a way to send a restaurant's own guests elsewhere.
var elementLinks = map[string]bool{
	"": true, "/": true, "/menu": true, "/cart": true, "/checkout": true,
	"/bron": true, "/about": true, "/profile": true, "/login": true,
}

var elementFonts = map[string]bool{"": true, "sans": true, "display": true}
var elementWeights = map[string]bool{"": true, "normal": true, "bold": true, "black": true}
var elementColors = map[string]bool{
	"": true, "ink": true, "soft": true, "muted": true, "white": true,
	"brand": true, "surface": true, "charcoal": true,
}

// Background tones a block may use. Deliberately the design system's own tokens
// (globals.css) rather than colours: a band painted `#f4f1ea` would stay that
// colour in dark mode, and the restaurant's own accent would stop applying.
var designTones = map[string]bool{
	"":         true, // page background
	"surface":  true,
	"raised":   true,
	"charcoal": true, // always-dark band, like today's hero
	"brand":    true,
}

var designPads = map[string]bool{"": true, "sm": true, "md": true, "lg": true}
var designAligns = map[string]bool{"": true, "left": true, "center": true}

// DesignStyle is how one band looks. Enums and booleans only.
type DesignStyle struct {
	Tone    string `bson:"tone,omitempty" json:"tone,omitempty"`
	Padding string `bson:"padding,omitempty" json:"padding,omitempty"`
	Align   string `bson:"align,omitempty" json:"align,omitempty"`
	// Corners follow the theme's radius when set; a band with square corners is
	// the exception, not a pixel value.
	Rounded bool `bson:"rounded,omitempty" json:"rounded,omitempty"`
}

// DesignBinding says which data a block shows — never what it says.
type DesignBinding struct {
	// menu-grid / categories: which categories, empty meaning "all".
	Categories []primitive.ObjectID `bson:"categories,omitempty" json:"categories,omitempty"`
	// menu-grid: popular dishes only. The default, and what the home page shows
	// today.
	PopularOnly bool `bson:"popularOnly,omitempty" json:"popularOnly,omitempty"`
	// How many items to show. 0 = the block's own sensible default.
	Limit int `bson:"limit,omitempty" json:"limit,omitempty"`
}

// DesignBox is where an element sits, in **percent of the band**.
//
// ⚠️ Percent, never pixels, and this is the one decision that keeps a freely
// drawn page from falling apart. A box placed at "x=340px" on the designer's
// 1400px screen has no defensible position on a 380px phone; the same box at
// "x=24%" has exactly one. It also means the design survives the Telegram mini
// app, whose viewport is decided by Telegram rather than by us.
//
// Height is percent of the band's own height, and the band's height is set in
// `Canvas.Height` — one number, so a band can be short or tall without every
// element inside it needing to agree.
type DesignBox struct {
	X int `bson:"x" json:"x"`
	Y int `bson:"y" json:"y"`
	W int `bson:"w" json:"w"`
	H int `bson:"h" json:"h"`
	// Stacking order. Small integer: a canvas with more than a handful of
	// overlapping elements is a canvas nobody can edit.
	Z int `bson:"z,omitempty" json:"z,omitempty"`
}

// DesignElement is one freely placed thing on a canvas.
type DesignElement struct {
	Type string    `bson:"type" json:"type"`
	Box  DesignBox `bson:"box" json:"box"`
	// ⚠️ **The phone layout, drawn separately.** Wix, Figma-to-site and every
	// other free-placement tool ends up here, because there is no arithmetic that
	// turns a desktop composition into a phone one: elements that sit side by side
	// on a wide screen have to become one column, and only a person knows in what
	// order. When this is nil the renderer stacks elements in their drawn order at
	// full width, which is the only safe default — a phone showing a squashed
	// desktop composition is the failure this feature would otherwise ship with.
	Mobile *DesignBox `bson:"mobile,omitempty" json:"mobile,omitempty"`
	Hidden bool       `bson:"hidden,omitempty" json:"hidden,omitempty"`
	// ⚠️ Hidden on a phone only. The commonest real fix for a busy composition:
	// a decorative shape that works on a desktop and is noise on a phone.
	HiddenMobile bool `bson:"hiddenMobile,omitempty" json:"hiddenMobile,omitempty"`

	// text/button: what it says, per language. ⚠️ The one place the constructor
	// carries words — a page drawn from a picture needs a headline, and a headline
	// nobody can type is a headline that stays in Lorem Ipsum. Three languages,
	// because the rest of the site has three; an empty ru/en falls back to uz, the
	// same rule `content` follows.
	Text  LocalizedText `bson:"text,omitempty" json:"text,omitempty"`
	Image string        `bson:"image,omitempty" json:"image,omitempty"`
	Link  string        `bson:"link,omitempty" json:"link,omitempty"`

	Style   ElementStyle  `bson:"style,omitempty" json:"style,omitempty"`
	Binding DesignBinding `bson:"binding,omitempty" json:"binding,omitempty"`
}

// ElementStyle is how one element looks. Enums and small integers only — the
// same rule the rest of this file follows, for the same reason: this becomes
// inline style on a public page.
type ElementStyle struct {
	Font   string `bson:"font,omitempty" json:"font,omitempty"`
	Weight string `bson:"weight,omitempty" json:"weight,omitempty"`
	Align  string `bson:"align,omitempty" json:"align,omitempty"`
	Color  string `bson:"color,omitempty" json:"color,omitempty"`
	// Background of the element itself (a box, or a panel behind text).
	Tone string `bson:"tone,omitempty" json:"tone,omitempty"`
	// Font size as a **step**, not a pixel value: 0 is the body size and the
	// steps follow the type scale, so a headline stays proportional when the
	// theme's root size changes.
	Size int `bson:"size,omitempty" json:"size,omitempty"`
	// Percent. Applies to the element's own background and image, never to text —
	// half-transparent words are unreadable, and this is the control most often
	// reached for to dim a photograph behind a headline.
	Opacity int  `bson:"opacity,omitempty" json:"opacity,omitempty"`
	Rounded bool `bson:"rounded,omitempty" json:"rounded,omitempty"`
	Shadow  bool `bson:"shadow,omitempty" json:"shadow,omitempty"`
}

// DesignCanvas is the band an operator draws inside.
type DesignCanvas struct {
	// Band height, in **viewport-height percent** on a desktop. A canvas is the
	// one block whose height nothing else can imply.
	Height int `bson:"height,omitempty" json:"height,omitempty"`
	// Mobile height, separately: the desktop composition's proportions rarely
	// survive, and a phone band that keeps a 90vh hero pushes everything else
	// below three scrolls of empty space.
	HeightMobile int    `bson:"heightMobile,omitempty" json:"heightMobile,omitempty"`
	Background   string `bson:"background,omitempty" json:"background,omitempty"`
	// Percent, applied to the background image only.
	BackgroundOpacity int             `bson:"backgroundOpacity,omitempty" json:"backgroundOpacity,omitempty"`
	Elements          []DesignElement `bson:"elements,omitempty" json:"elements,omitempty"`
}

// DesignSection is one band of the page.
type DesignSection struct {
	Type    string `bson:"type" json:"type"`
	Variant string `bson:"variant,omitempty" json:"variant,omitempty"`
	// Width in columns of twelve. 12 is a full-width band; two sixes sit side by
	// side on a desktop and stack on a phone.
	//
	// ⚠️ **This is the entire responsive story.** Because width is columns
	// rather than pixels, the phone rule is one line — everything is twelve —
	// and it cannot be forgotten or drawn wrong.
	Span int `bson:"span" json:"span"`
	// Hidden keeps a band in the design without rendering it. An operator trying
	// a layout needs to take a band out and put it back; deleting and redrawing
	// it loses the settings that were tuned.
	Hidden  bool          `bson:"hidden,omitempty" json:"hidden,omitempty"`
	Style   DesignStyle   `bson:"style,omitempty" json:"style,omitempty"`
	Binding DesignBinding `bson:"binding,omitempty" json:"binding,omitempty"`
	// Only for `canvas` and `popup`: what was drawn inside.
	Canvas *DesignCanvas `bson:"canvas,omitempty" json:"canvas,omitempty"`
}

// Design statuses. Stored, so never renamed.
const (
	DesignDraft     = "draft"
	DesignPublished = "published"
)

// PageDesign is one brand's home page layout.
//
// It lives in the **tenant's own database** and the Keel console writes it
// there directly — the same shape as `export_grant`: one document, one writer
// (the console), one reader (this app). Nothing to keep in sync, and no new
// path from a tenant container back to the control plane. That path not
// existing is what makes one restaurant unable to reach another's data.
type PageDesign struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BrandID primitive.ObjectID `bson:"brandId" json:"brandId"`
	// draft is invisible to the site; only `published` is rendered. An operator
	// mid-layout must not be showing a half-drawn page to the restaurant's
	// guests.
	Status   string          `bson:"status" json:"status"`
	Sections []DesignSection `bson:"sections" json:"sections"`
	// ⚠️ **The one free-form field in this file, and it is deliberately narrow.**
	//
	// Everything else here is an enum because this document becomes styles on a
	// public page. But a design copied from a picture always has one detail no
	// enum anticipated, and without an escape hatch that detail becomes a code
	// change per customer — which is the opposite of why the constructor exists.
	//
	// It is CSS **text**, injected inside a `<style>` element, and it is cleaned
	// by `sanitizeCSS`: no `</style`, no `<`, no `javascript:`, no `@import`, no
	// `url(` pointing anywhere but our own uploads, and a length cap. It is also
	// written only by the console — a tenant owner cannot reach this field at all,
	// which is what makes it a designer's tool rather than an XSS hole with a nice
	// name.
	CustomCSS string `bson:"customCss,omitempty" json:"customCss,omitempty"`
	// The theme, drawn in the console alongside the layout. Empty means "leave
	// the tenant's own": a design that only rearranges bands must not silently
	// repaint a restaurant that spent an afternoon choosing its accent.
	Theme *SiteTheme `bson:"theme,omitempty" json:"theme,omitempty"`
	// Which console operator drew it, and when it went live.
	//
	// ⚠️ These two fields are also the **record that this customer has a paid
	// design**. The fee is settled outside the platform, so nothing bills for
	// it — but "is this one drawn?" still has to be answerable, and the
	// document's own existence answers it. The tenant's theme editor locks off
	// this, which is why it is not a separate flag somebody could forget to set.
	DrawnBy     string     `bson:"drawnBy,omitempty" json:"drawnBy,omitempty"`
	PublishedAt *time.Time `bson:"publishedAt,omitempty" json:"publishedAt,omitempty"`
	UpdatedAt   time.Time  `bson:"updatedAt" json:"updatedAt"`
}

// DefaultSections is the page every restaurant has today, expressed as blocks.
//
// ⚠️ **This is the feature's precondition, not a nicety.** Existing customers
// must see no change at all on the day this ships, so the renderer's fallback is
// not "an empty page" or "a reasonable default" — it is *this*, in this order,
// which is the order `app/(site)/page.tsx` renders in now.
func DefaultSections() []DesignSection {
	return []DesignSection{
		{Type: BlockHero, Variant: "full", Span: 12, Style: DesignStyle{Tone: "charcoal"}},
		{Type: BlockPerks, Variant: "cards", Span: 12},
		{Type: BlockCategories, Variant: "tiles", Span: 12},
		{Type: BlockMenuGrid, Variant: "cards", Span: 12,
			Style:   DesignStyle{Tone: "surface"},
			Binding: DesignBinding{PopularOnly: true, Limit: 8}},
		{Type: BlockHoursAddress, Variant: "map", Span: 12},
	}
}

// Sanitize makes a stored design safe to render.
//
// Called on every read rather than only on write, because the write is not the
// only way a document gets here: a hand-edited Mongo document, a half-applied
// migration or an older console are all real. Unknown values are **dropped**,
// not corrected to something plausible — a band nobody recognises should
// disappear, not appear as a mystery.
func (d *PageDesign) Sanitize() {
	out := make([]DesignSection, 0, len(d.Sections))
	for _, s := range d.Sections {
		variants, ok := blockVariants[s.Type]
		if !ok {
			continue // unknown block type: drop it
		}
		if !slices.Contains(variants, s.Variant) {
			// An unknown variant is a known block drawn by a newer console. The
			// block itself is still worth rendering, in its first variant.
			s.Variant = variants[0]
		}
		// Columns of twelve, clamped. Zero means the document predates the field
		// or somebody cleared it, and a full-width band is the safe reading.
		if s.Span < 1 || s.Span > 12 {
			s.Span = 12
		}
		if !designTones[s.Style.Tone] {
			s.Style.Tone = ""
		}
		if !designPads[s.Style.Padding] {
			s.Style.Padding = ""
		}
		if !designAligns[s.Style.Align] {
			s.Style.Align = ""
		}
		if s.Binding.Limit < 0 || s.Binding.Limit > 48 {
			// 48 is the seeded menu's size: a "limit" larger than any real menu
			// is a number somebody typed, not a decision.
			s.Binding.Limit = 0
		}
		// A canvas is only meaningful on the two blocks that have one; a `canvas`
		// object hanging off a hero is a document somebody hand-edited.
		if s.Type != BlockCanvas && s.Type != BlockPopup {
			s.Canvas = nil
		} else if s.Canvas == nil {
			// An empty canvas renders nothing at all, which on a `canvas` band is
			// an invisible gap the operator cannot see to fix. Given one it is at
			// least a band with a height.
			s.Canvas = &DesignCanvas{}
		}
		if s.Canvas != nil {
			sanitizeCanvas(s.Canvas)
		}
		out = append(out, s)
	}
	d.Sections = out
	d.CustomCSS = sanitizeCSS(d.CustomCSS)
}

// sanitizeCanvas clamps a freely drawn band into values that can be rendered.
//
// ⚠️ Every number here is clamped rather than rejected. A design arriving with a
// nonsense box is a console bug or an older document, and dropping the element
// silently loses somebody's work; putting it back on screen at a sane size means
// they can see it and move it. The exception is the *type* — an unknown element
// cannot be drawn at all.
func sanitizeCanvas(c *DesignCanvas) {
	// Heights in vh. Under 10 is a band nothing fits in; over 200 is two screens
	// of scrolling before the next band, which reads as a broken page.
	c.Height = clampInt(c.Height, 10, 200, 60)
	c.HeightMobile = clampInt(c.HeightMobile, 0, 200, 0)
	if !designTones[c.Background] {
		c.Background = ""
	}
	c.BackgroundOpacity = clampInt(c.BackgroundOpacity, 0, 100, 100)

	out := make([]DesignElement, 0, len(c.Elements))
	for _, e := range c.Elements {
		if !elementTypes[e.Type] {
			continue
		}
		e.Box = sanitizeBox(e.Box)
		if e.Mobile != nil {
			box := sanitizeBox(*e.Mobile)
			e.Mobile = &box
		}
		if !elementLinks[e.Link] {
			// ⚠️ Silently cleared rather than kept: this value becomes an `href` on
			// a public page, and an unrecognised one is either a typo or somebody
			// sending a restaurant's guests to their own site.
			e.Link = ""
		}
		if !elementFonts[e.Style.Font] {
			e.Style.Font = ""
		}
		if !elementWeights[e.Style.Weight] {
			e.Style.Weight = ""
		}
		if !designAligns[e.Style.Align] {
			e.Style.Align = ""
		}
		if !elementColors[e.Style.Color] {
			e.Style.Color = ""
		}
		if !designTones[e.Style.Tone] {
			e.Style.Tone = ""
		}
		e.Style.Size = clampInt(e.Style.Size, -2, 8, 0)
		e.Style.Opacity = clampInt(e.Style.Opacity, 0, 100, 100)
		e.Image = sanitizeImagePath(e.Image)
		out = append(out, e)
	}
	// A canvas with more elements than this is not a design, it is a document
	// somebody generated — and it would be unusable in the editor.
	if len(out) > 60 {
		out = out[:60]
	}
	c.Elements = out
}

func sanitizeBox(b DesignBox) DesignBox {
	// Positions may sit slightly outside the band (a shape bleeding off the edge
	// is a real technique), but not so far that it vanishes.
	b.X = clampInt(b.X, -50, 150, 0)
	b.Y = clampInt(b.Y, -50, 150, 0)
	b.W = clampInt(b.W, 2, 200, 30)
	b.H = clampInt(b.H, 2, 200, 20)
	b.Z = clampInt(b.Z, 0, 20, 0)
	return b
}

func clampInt(v, min, max, fallback int) int {
	if v == 0 && fallback != 0 {
		return fallback
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// sanitizeImagePath keeps only images this site actually serves.
//
// ⚠️ An arbitrary URL here would load a third party's file into every visitor's
// page — a tracking pixel by another name, and a picture that can be swapped for
// something else long after the design was approved. Uploads only.
func sanitizeImagePath(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "/uploads/") && !strings.Contains(v, "..") {
		return v
	}
	return ""
}

// sanitizeCSS is the guard on the file's only free-form field.
//
// ⚠️ It is a **whitelist of shape**, not a parser. The dangerous constructs in a
// `<style>` element are few and well known, and each one is refused outright
// rather than escaped:
//
//   - `</style` ends the element early and turns everything after it into markup.
//     This is the one that matters; the rest are defence in depth.
//   - `<` has no legitimate use in CSS and is how the above is spelled in
//     variations.
//   - `javascript:` and `expression(` are script in a style position.
//   - `@import` fetches a stylesheet from anywhere, which moves the whole problem
//     to a server we do not control and cannot review.
//   - `url(` is allowed **only** for our own uploads: otherwise a background
//     image is a third-party request on every visitor's page.
//
// Anything refused clears the whole field rather than being patched out: a
// half-removed rule is a stylesheet nobody wrote, and the operator needs to see
// that their CSS was rejected rather than wonder why it half works.
func sanitizeCSS(css string) string {
	css = strings.TrimSpace(css)
	if css == "" {
		return ""
	}
	// Long enough for a real design's worth of corrections, short enough that
	// nothing can be hidden in it.
	if len(css) > 20000 {
		return ""
	}
	lower := strings.ToLower(css)
	for _, bad := range []string{"<", "javascript:", "expression(", "@import", "\\3c"} {
		if strings.Contains(lower, bad) {
			return ""
		}
	}
	// Every url( must point at our own uploads.
	rest := lower
	for {
		i := strings.Index(rest, "url(")
		if i < 0 {
			break
		}
		rest = rest[i+4:]
		arg := strings.TrimLeft(rest, " \t'\"")
		if !strings.HasPrefix(arg, "/uploads/") {
			return ""
		}
	}
	return css
}

// Renderable reports whether this design should replace the default layout.
//
// A published design with every band dropped by Sanitize is **not** renderable:
// the site falls back to the template rather than showing a blank page. An empty
// page is the one outcome worse than an unstyled one.
func (d *PageDesign) Renderable() bool {
	return d != nil && d.Status == DesignPublished && len(d.Sections) > 0
}
