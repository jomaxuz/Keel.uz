package handlers

import (
	"net/http"
	"slices"
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
	"about", "gallery", "cta",
}

type designSection struct {
	Type    string `bson:"type" json:"type"`
	Variant string `bson:"variant,omitempty" json:"variant,omitempty"`
	Span    int    `bson:"span" json:"span"`
	Hidden  bool   `bson:"hidden,omitempty" json:"hidden,omitempty"`
	Style   struct {
		Tone    string `bson:"tone,omitempty" json:"tone,omitempty"`
		Padding string `bson:"padding,omitempty" json:"padding,omitempty"`
		Align   string `bson:"align,omitempty" json:"align,omitempty"`
		Rounded bool   `bson:"rounded,omitempty" json:"rounded,omitempty"`
	} `bson:"style,omitempty" json:"style,omitempty"`
	Binding struct {
		Categories  []string `bson:"categories,omitempty" json:"categories,omitempty"`
		PopularOnly bool     `bson:"popularOnly,omitempty" json:"popularOnly,omitempty"`
		Limit       int      `bson:"limit,omitempty" json:"limit,omitempty"`
	} `bson:"binding,omitempty" json:"binding,omitempty"`
}

type designDoc struct {
	ID          string             `bson:"_id" json:"-"`
	BrandID     primitive.ObjectID `bson:"brandId,omitempty" json:"brandId,omitempty"`
	Status      string             `bson:"status" json:"status"`
	Sections    []designSection    `bson:"sections" json:"sections"`
	DrawnBy     string             `bson:"drawnBy,omitempty" json:"drawnBy,omitempty"`
	PublishedAt *time.Time         `bson:"publishedAt,omitempty" json:"publishedAt,omitempty"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
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
		"draft": map[string]any{
			"sections":  draft.Sections,
			"updatedAt": draft.UpdatedAt,
			"drawnBy":   draft.DrawnBy,
		},
		"live": map[string]any{
			"sections":    live.Sections,
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
		"brandId":   h.primaryBrandID(r, t.DBName()),
		"status":    "draft",
		"sections":  sections,
		"drawnBy":   currentOperator(r),
		"updatedAt": time.Now(),
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
			"status":      "published",
			"sections":    sections,
			"drawnBy":     currentOperator(r),
			"publishedAt": now,
			"updatedAt":   now,
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

func (h *Handler) ListDesignTemplates(w http.ResponseWriter, r *http.Request) {
	cur, err := h.Store.DesignTemplates.Find(r.Context(), bson.M{},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := []designTemplate{}
	_ = cur.All(r.Context(), &items)
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
