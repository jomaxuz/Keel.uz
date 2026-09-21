package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"slices"
	"strconv"
	"time"

	"keel-control/internal/httpx"

	"github.com/go-chi/chi/v5"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Drawing a restaurant's page layout, from here.
//
// The design is sold as a service, so the tool lives in the console and the
// restaurant's own panel has no layout editor at all (see CONSTRUCTOR.md). The
// document itself is written **into the tenant's own database** — the same shape
// as `export_grant`: one document, one writer (this), one reader (their site).
// Nothing to keep in sync, and no new path from a tenant container back here.
//
// Two rules shape the endpoints:
//
//   - **A draft is invisible.** Saving never changes the live site; publishing
//     copies the draft across. An operator mid-layout must not be showing a
//     half-drawn page to the restaurant's guests.
//   - **Applying a template copies, never links.** Ten customers can share a
//     layout, and editing the template afterwards must not silently redraw ten
//     live sites. That is the difference between a starting point and a
//     dependency.
//
// ⚠️ The **tenant** sanitises this document on every read (models/design.go), and
// that is deliberately where the guard lives: this console and that server deploy
// separately, so the side that renders the page is the side that has to be sure.
// The list below is a courtesy — it stops an obvious mistake early — not the
// boundary.

const designDocID = "home"

// The block types the tenant knows how to render. Kept in step with
// `models.BlockHero…` by hand; a name only in this list produces a band the
// tenant drops, which is why the tenant is the authority.
var designBlocks = []string{
	"hero", "perks", "categories", "menu-grid", "hours-address",
	// ⚠️ The guests' own words. Missing here for as long as the block existed,
	// which is why no console-drawn page had one: the tenant would render it,
	// the editor had no button for it, and this list would have refused to
	// store it anyway. It draws nothing until the restaurant switches reviews
	// on, so placing it costs an operator nothing.
	"reviews",
	"about", "gallery", "cta",
	// Schema-driven sections: they read their own settings, so the console's panel
	// is generated from design_schema rather than written per type.
	"rich-text", "image-text", "banner",
	// The restaurant's own strip: a band the console can place, with nothing to
	// configure here — the pictures are the owner's.
	"banners",
	// The site's own chrome, and the two free-drawing blocks. Kept in step with
	// models/design.go by hand: this list only decides what the console may
	// **store**, and the tenant's Sanitize decides what may be rendered — so a
	// name missing here is a block the console cannot save, not a hole.
	"navbar", "footer", "canvas", "popup",
}

type designSection struct {
	Type    string `bson:"type" json:"type"`
	Variant string `bson:"variant,omitempty" json:"variant,omitempty"`
	Span    int    `bson:"span" json:"span"`
	Hidden  bool   `bson:"hidden,omitempty" json:"hidden,omitempty"`
	// ⚠️ **Mirrored field by field, which makes it the one struct here that can
	// lose data by omission** — and it very nearly did again. Go's decoder drops
	// an unknown JSON field without a word, so a `style.width` written by the
	// console and missing from this struct would arrive at the database as
	// nothing: the save returns a count, the editor keeps showing the value
	// until it is reloaded, and the live site quietly ignores it. That is the
	// `settings` bug, and this comment is here so the next field added to a
	// band's style is added here too.
	Style struct {
		Tone    string `bson:"tone,omitempty" json:"tone,omitempty"`
		Padding string `bson:"padding,omitempty" json:"padding,omitempty"`
		Align   string `bson:"align,omitempty" json:"align,omitempty"`
		Width   string `bson:"width,omitempty" json:"width,omitempty"`
		Rounded bool   `bson:"rounded,omitempty" json:"rounded,omitempty"`
	} `bson:"style,omitempty" json:"style,omitempty"`
	// ⚠️ Passed through as-is, not mirrored field by field.
	//
	// A canvas holds freely placed elements, each with a box, a mobile override, a
	// style and text in three languages. Restating all of that here would create a
	// second copy of the model that has to be kept in step with the tenant's — and
	// the copy that drifts is always the one that is not the security boundary.
	//
	// **The boundary is `models.PageDesign.Sanitize()` on the tenant side**, which
	// runs on every read before anything is rendered: unknown element types are
	// dropped, every number clamped, images restricted to our own uploads, links
	// to the site's own pages, and the CSS field refused outright if it contains
	// anything that could close a `<style>` element. A value that survives storage
	// here still cannot reach a page unsanitised.
	Canvas any `bson:"canvas,omitempty" json:"canvas,omitempty"`
	// The band's declared settings and its repeatable items, passed through for
	// exactly the reason the canvas is — the tenant's `sanitizeSettings` is the
	// boundary, and a second copy of the shape here would be the copy that drifts.
	//
	// ⚠️ **Their absence was a silent data loss, not a missing feature.** Go's
	// decoder drops unknown JSON fields without a word, so every value the schema
	// panel wrote — a band's heading, how many dishes it shows, which categories —
	// was discarded on the way to the database. Nothing failed: the save returned
	// a count, the editor kept showing what had been typed until it was reloaded,
	// and the live tenant's document carried eight bands with **no settings at
	// all**. The half that made a schema real (SchemaBlocks) had a hole in the
	// middle of the pipe.
	Settings any `bson:"settings,omitempty" json:"settings,omitempty"`
	Blocks   any `bson:"blocks,omitempty" json:"blocks,omitempty"`
	Binding  struct {
		Categories  []string `bson:"categories,omitempty" json:"categories,omitempty"`
		PopularOnly bool     `bson:"popularOnly,omitempty" json:"popularOnly,omitempty"`
		Limit       int      `bson:"limit,omitempty" json:"limit,omitempty"`
	} `bson:"binding,omitempty" json:"binding,omitempty"`
}

type designDoc struct {
	ID       string             `bson:"_id" json:"-"`
	BrandID  primitive.ObjectID `bson:"brandId,omitempty" json:"brandId,omitempty"`
	Status   string             `bson:"status" json:"status"`
	Sections []designSection    `bson:"sections" json:"sections"`
	// The navigation bar, when this site's is not the built-in one.
	//
	// ⚠️ **`any`, like the theme and the saved styles**, and for the same
	// reason: the shape belongs to `models.NavLink` in the tenant's repository,
	// which sanitises it on every read. A second copy of the struct here would
	// be a schema the control plane has to keep in step with a codebase it does
	// not import — and the way that failure shows up is Go's decoder dropping
	// the fields it does not know, silently, which is exactly how `settings`
	// was lost on this very pipe.
	Nav any `bson:"nav,omitempty" json:"nav,omitempty"`
	// Read back so the editor can reopen with what was saved — the CSS field and
	// the saved styles are as much part of a draft as the bands are.
	CustomCSS string `bson:"customCss,omitempty" json:"customCss,omitempty"`
	// The palette the design was drawn with. ⚠️ Read back so the editor
	// reopens with it, and copied on publish — the tenant lays it over the
	// brand's own theme field by field (handlers/public.go, mergeTheme).
	Theme        any        `bson:"theme,omitempty" json:"theme,omitempty"`
	StylePresets any        `bson:"stylePresets,omitempty" json:"stylePresets,omitempty"`
	DrawnBy      string     `bson:"drawnBy,omitempty" json:"drawnBy,omitempty"`
	PublishedAt  *time.Time `bson:"publishedAt,omitempty" json:"publishedAt,omitempty"`
	UpdatedAt    time.Time  `bson:"updatedAt" json:"updatedAt"`
}

// primaryBrandID is the brand a layout belongs to.
//
// ⚠️ Written into the document because the **tenant reads by brand**: the layout
// is the brand's face, not the company's (a company running two brands shows two
// different sites). A document saved without it is a document the site never
// finds — and the failure is silent, because the site falls back to the template
// and looks exactly as it did before.
//
// Read from the tenant's own `brand` collection, first active by sort order —
// the same one its site opens on.
func (h *Handler) primaryBrandID(r *http.Request, dbName string) primitive.ObjectID {
	var b struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	err := h.Store.TenantDB(dbName).Collection("brand").FindOne(r.Context(),
		bson.M{"isActive": true},
		options.FindOne().SetSort(bson.D{{Key: "sortOrder", Value: 1}}),
	).Decode(&b)
	if err != nil {
		return primitive.NilObjectID
	}
	return b.ID
}

// tenantCategories lists the restaurant's menu categories for the band pickers.
//
// Read straight from the tenant's database, the same way the brand is — the
// console has no other way in, and inventing one (an HTTP call into the
// customer's own container) would be a path from the control plane into a tenant
// that has to be authenticated, kept alive and reasoned about forever.
//
// ⚠️ Names only, and only the fields a chip needs. This response is a design
// document; pulling whole menu items into it would put prices and stock into a
// payload nobody reviews for that.
func (h *Handler) tenantCategories(r *http.Request, dbName string) []map[string]string {
	cur, err := h.Store.TenantDB(dbName).Collection("category").Find(r.Context(),
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}}).SetLimit(60),
	)
	if err != nil {
		// An empty list, never an error: the editor opens without a picker rather
		// than not at all. A design is mostly other things.
		return []map[string]string{}
	}
	var rows []struct {
		ID   primitive.ObjectID `bson:"_id"`
		Name string             `bson:"name"`
	}
	if err := cur.All(r.Context(), &rows); err != nil {
		return []map[string]string{}
	}
	// ⚠️ Built as an empty slice, not a nil one: a nil slice marshals to `null`
	// and the console reads `.map` off it. The same trap this codebase has been
	// bitten by twice — see CLAUDE.md.
	out := make([]map[string]string, 0, len(rows))
	for _, c := range rows {
		out = append(out, map[string]string{"id": c.ID.Hex(), "name": c.Name})
	}
	return out
}

// clean drops bands the tenant would drop anyway and clamps the grid, so an
// obvious mistake is caught while the operator is still looking at it.
func clean(sections []designSection) []designSection {
	out := make([]designSection, 0, len(sections))
	for _, s := range sections {
		if !slices.Contains(designBlocks, s.Type) {
			continue
		}
		if s.Span < 1 || s.Span > 12 {
			s.Span = 12
		}
		if s.Binding.Limit < 0 || s.Binding.Limit > 48 {
			s.Binding.Limit = 0
		}
		out = append(out, s)
		// A page with more bands than this is not a design somebody drew.
		if len(out) >= 40 {
			break
		}
	}
	return out
}

// GetTenantDesign returns both the draft and what is live.
//
// Both, because the operator's first question is "what does this customer have
// now, and what have I changed since" — and a screen that showed only one of them
// answers neither.
func (h *Handler) GetTenantDesign(w http.ResponseWriter, r *http.Request) {
	t := h.tenantFromURL(r)
	if t == nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	coll := h.Store.TenantDB(t.DBName()).Collection("page_design")

	var draft, live designDoc
	_ = coll.FindOne(r.Context(), bson.M{"_id": designDocID}).Decode(&draft)
	_ = coll.FindOne(r.Context(), bson.M{"_id": designDocID + ":live"}).Decode(&live)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"blocks": designBlocks,
		// The restaurant's own categories, so a band can be pointed at some of them
		// rather than at all of them.
		//
		// ⚠️ Sent with the design rather than fetched separately, because the editor
		// cannot usefully open without them: a picker that arrives a moment later
		// shows a saved selection as a list of blank chips, and an operator's first
		// reading of that is that their choice was lost.
		"categories": h.tenantCategories(r, t.DBName()),
		// ⚠️ **Every field the editor sends must come back out**, and two did
		// not: the stylesheet and the saved styles were decoded out of the
		// database into `designDoc` — which says in as many words that they are
		// "as much part of a draft as the bands are" — and then left out of this
		// map.
		//
		// That is not a field missing from a screen, it is data loss with a 200.
		// The console is a read-modify-write client: it opens the draft, changes
		// one box and writes the whole document back — so a field it never
		// received is a field it blanks on the next save. A live site lost its
		// corrections mid-session exactly this way, and nothing said so; the save
		// returned a count and the page quietly stopped having the rules it had.
		//
		// The same trap as `settings` on the way *in* (see designSection), one
		// direction along.
		"draft": map[string]any{
			"sections":     draft.Sections,
			"nav":          draft.Nav,
			"theme":        draft.Theme,
			"customCss":    draft.CustomCSS,
			"stylePresets": draft.StylePresets,
			"updatedAt":    draft.UpdatedAt,
			"drawnBy":      draft.DrawnBy,
		},
		"live": map[string]any{
			"sections":    live.Sections,
			"nav":         live.Nav,
			"theme":       live.Theme,
			"publishedAt": live.PublishedAt,
			"drawnBy":     live.DrawnBy,
		},
		// True once anything is live: the tenant's own theme editor locks off
		// this, so the console has to be able to say why.
		"published": len(live.Sections) > 0,
	})
}

type designSaveRequest struct {
	Sections []designSection `json:"sections"`
	// The design's own corrections, and the theme it was drawn against. Both
	// pass through to the tenant, which sanitises them — see designSection.Canvas.
	CustomCSS string `json:"customCss"`
	Theme     any    `json:"theme"`
	// Named styles the designer saved while drawing. Passed through and cleaned by
	// the tenant's Sanitize, like the canvas — see designSection.Canvas.
	StylePresets any `json:"stylePresets"`
	// The navigation bar. Passed through and cleaned by the tenant, which is
	// where the address is checked — see models.sanitizeNav.
	Nav any `json:"nav"`
}

// PutTenantDesign saves the draft. The live site does not change.
func (h *Handler) PutTenantDesign(w http.ResponseWriter, r *http.Request) {
	t := h.tenantFromURL(r)
	if t == nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	var req designSaveRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sections := clean(req.Sections)
	coll := h.Store.TenantDB(t.DBName()).Collection("page_design")
	_, err := coll.UpdateOne(r.Context(), bson.M{"_id": designDocID}, bson.M{"$set": bson.M{
		"brandId":      h.primaryBrandID(r, t.DBName()),
		"status":       "draft",
		"sections":     sections,
		"nav":          req.Nav,
		"customCss":    req.CustomCSS,
		"theme":        req.Theme,
		"stylePresets": req.StylePresets,
		"drawnBy":      currentOperator(r),
		"updatedAt":    time.Now(),
	}}, options.Update().SetUpsert(true))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"saved": len(sections)})
}

// PublishTenantDesign copies the draft onto the live site.
func (h *Handler) PublishTenantDesign(w http.ResponseWriter, r *http.Request) {
	t := h.tenantFromURL(r)
	if t == nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	coll := h.Store.TenantDB(t.DBName()).Collection("page_design")

	var draft designDoc
	if err := coll.FindOne(r.Context(), bson.M{"_id": designDocID}).Decode(&draft); err != nil {
		httpx.Error(w, http.StatusBadRequest, "qoralama yo'q")
		return
	}
	sections := clean(draft.Sections)
	if len(sections) == 0 {
		// Publishing nothing would leave the site rendering the template anyway
		// (the tenant falls back rather than showing a blank page), so this is
		// refused here where it can still be explained.
		httpx.Error(w, http.StatusBadRequest, "bo'sh dizaynni chop etib bo'lmaydi")
		return
	}
	now := time.Now()
	if _, err := coll.UpdateOne(r.Context(), bson.M{"_id": designDocID + ":live"},
		bson.M{"$set": bson.M{
			"brandId": h.primaryBrandID(r, t.DBName()),
			// ⚠️ The tenant reads `status: "published"`; the id keeps the two
			// documents apart so a draft can keep being edited while a design is
			// live. Changing either name breaks the reader silently.
			"status":   "published",
			"sections": sections,
			// ⚠️ **The CSS has to be copied too**, and it was not.
			//
			// Publishing copied only the bands, so a designer's corrections lived in
			// the draft for ever: saved, visible in the preview (which reads the
			// draft), and absent from the live site. That is the worst shape for this
			// particular bug — the preview is exactly where somebody checks their
			// work, so it would look correct right up until the customer looked.
			"customCss": draft.CustomCSS,
			// ⚠️ **Copied for the reason the CSS is.** A palette edited in the
			// draft, visible in the preview (which reads the draft) and absent
			// from the live site is a change that looks right up until the
			// customer looks. The tenant merges it field by field over the
			// brand's own — see handlers/public.go, mergeTheme.
			"theme": draft.Theme,
			// ⚠️ **Copied for the reason the CSS is**, which is the bug this line
			// exists to not repeat: a bar edited in the draft, visible in the
			// preview (which reads the draft) and absent from the live site is a
			// change that looks right up until the customer looks.
			"nav": draft.Nav,
			// The saved styles travel with the design. The renderer never reads them
			// (applying a preset copies it into the element), but a published design
			// reopened later should still offer the styles it was drawn with.
			"stylePresets": draft.StylePresets,
			"drawnBy":      currentOperator(r),
			"publishedAt":  now,
			"updatedAt":    now,
		}}, options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"published": len(sections), "publishedAt": now,
		// Said out loud because the operator's next move is to open the site and
		// judge whether it worked: the page cache holds each page for 30 seconds
		// (caddy/pagecache.conf), so "nothing changed" for half a minute is
		// expected rather than a failed publish.
		"note": "o'zgarish jonli saytda 30 soniya ichida ko'rinadi (sahifa keshi)",
	})
}

// RevertTenantDesign takes the design off the live site.
//
// Not a delete of the draft: the customer may be dropping the design for now, or
// the operator may be undoing a bad publish, and losing the drawing in either
// case would be the wrong trade. The site falls back to the template, which is
// what it renders when no design exists at all.
func (h *Handler) RevertTenantDesign(w http.ResponseWriter, r *http.Request) {
	t := h.tenantFromURL(r)
	if t == nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	coll := h.Store.TenantDB(t.DBName()).Collection("page_design")
	if _, err := coll.DeleteOne(r.Context(),
		bson.M{"_id": designDocID + ":live"}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"reverted": true})
}

// ---- Templates ----
//
// The commercial point of the tool: one layout, several customers. Stored in the
// control database because they belong to us, not to any tenant.

type designTemplate struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	Sections  []designSection    `bson:"sections" json:"sections"`
	CreatedBy string             `bson:"createdBy" json:"createdBy"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

// What the gallery returns. Built-ins and saved templates are the same shape on
// purpose — the console applies both the same way, and a screen that had to branch
// on which kind it was holding would eventually branch wrong.
type galleryItem struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Note     string          `json:"note,omitempty"`
	Sections []designSection `json:"sections"`
	// ⚠️ Read-only: it lives in the binary, so there is nothing to delete. The
	// console hides the delete button rather than offering one that cannot work.
	Builtin   bool   `json:"builtin,omitempty"`
	CreatedBy string `json:"createdBy,omitempty"`
}

func (h *Handler) ListDesignTemplates(w http.ResponseWriter, r *http.Request) {
	cur, err := h.Store.DesignTemplates.Find(r.Context(), bson.M{},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	saved := []designTemplate{}
	_ = cur.All(r.Context(), &saved)

	// Built-ins first: they are the answer to "I have nothing and a customer is
	// waiting", which is the state this list is opened in most often.
	items := []galleryItem{}
	for i, t := range builtinTemplates() {
		items = append(items, galleryItem{
			ID:       "builtin:" + strconv.Itoa(i),
			Name:     t.Name,
			Note:     t.Note,
			Sections: t.Sections,
			Builtin:  true,
		})
	}
	for _, t := range saved {
		items = append(items, galleryItem{
			ID:        t.ID.Hex(),
			Name:      t.Name,
			Sections:  t.Sections,
			CreatedBy: t.CreatedBy,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

type templateSaveRequest struct {
	Name     string          `json:"name"`
	Sections []designSection `json:"sections"`
}

func (h *Handler) CreateDesignTemplate(w http.ResponseWriter, r *http.Request) {
	var req templateSaveRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	name := req.Name
	if len(name) < 2 {
		// A gallery of "template 1 … template 9" is a gallery nobody uses, and
		// the person who has to pick from it is you in three months.
		httpx.Error(w, http.StatusBadRequest, "shablonga nom bering")
		return
	}
	sections := clean(req.Sections)
	if len(sections) == 0 {
		httpx.Error(w, http.StatusBadRequest, "bo'sh shablon saqlanmaydi")
		return
	}
	doc := designTemplate{
		Name: name, Sections: sections,
		CreatedBy: currentOperator(r), CreatedAt: time.Now(),
	}
	res, err := h.Store.DesignTemplates.InsertOne(r.Context(), doc)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	doc.ID, _ = res.InsertedID.(primitive.ObjectID)
	httpx.JSON(w, http.StatusCreated, doc)
}

func (h *Handler) DeleteDesignTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	if _, err := h.Store.DesignTemplates.DeleteOne(r.Context(),
		bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// PreviewTenantDesign hands back a link that shows the draft on the real site.
//
// ⚠️ **This is what makes the editor an editor.** A schematic preview cannot
// answer the only question that matters when a customer has sent a picture and
// asked for something like it — the fonts, the photographs, the real dish names
// and the accent colour are what make a layout look right or wrong. So the
// preview is the site itself, in an iframe, with the unpublished draft applied.
//
// Everything about the token is chosen to keep the draft away from visitors:
//
//   - Written into the **tenant's own database**, exactly as an export grant is.
//     One writer (this console), one reader (that tenant), and no new path from a
//     tenant container back here — the absence of that path is what keeps one
//     restaurant unable to reach another.
//   - **Two hours.** A preview link is a tool for the hour somebody is drawing,
//     not a URL to paste into a chat. One that lived for ever would be the draft
//     published by accident.
//   - **Random, and long.** It is the only thing standing between an unfinished
//     page and anybody who tries the parameter.
//   - It carries **no permissions**. Holding it shows one brand's draft layout.
//     It is not a session and cannot become one.
func (h *Handler) PreviewTenantDesign(w http.ResponseWriter, r *http.Request) {
	t := h.tenantFromURL(r)
	if t == nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "token yaratilmadi")
		return
	}
	token := hex.EncodeToString(buf)
	expires := time.Now().Add(2 * time.Hour)

	coll := h.Store.TenantDB(t.DBName()).Collection("design_preview")
	if _, err := coll.InsertOne(r.Context(), bson.M{
		"token":     token,
		"brandId":   h.primaryBrandID(r, t.DBName()),
		"createdBy": currentOperator(r),
		"createdAt": time.Now(),
		"expiresAt": expires,
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		// ⚠️ `publicDomain`, not `Domains[0]`: the first entry is the *technical*
		// primary (the free subdomain), and a preview of a restaurant with its own
		// domain should open on the domain the design will actually live on — the
		// fonts, the redirects and the edge behaviour all belong to that host.
		"url":       "https://" + publicDomain(t.Domains, h.Cfg.BaseDomain) + "/?preview=" + token,
		"expiresAt": expires,
	})
}
