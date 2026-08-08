package models

import (
	"slices"
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
		out = append(out, s)
	}
	d.Sections = out
}

// Renderable reports whether this design should replace the default layout.
//
// A published design with every band dropped by Sanitize is **not** renderable:
// the site falls back to the template rather than showing a blank page. An empty
// page is the one outcome worse than an unstyled one.
func (d *PageDesign) Renderable() bool {
	return d != nil && d.Status == DesignPublished && len(d.Sections) > 0
}
