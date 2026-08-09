package handlers

import (
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Showing the console an unpublished design on the real site.
//
// The editor's whole job is answering "does this look like the picture the
// customer sent us", and a schematic preview cannot answer it: the fonts, the
// photographs, the menu's real dish names and the accent colour are what make a
// layout look right or wrong. So the preview is the site itself, in an iframe,
// with the draft applied.
//
// That creates the obvious risk, and everything here exists to bound it:
//
//   - **The draft must never reach a visitor.** A half-drawn page served to a
//     restaurant's guests is worse than no constructor at all.
//   - So access is a **token**, not a flag: created by the console, stored in
//     this tenant's own database with an expiry, and checked on every request.
//   - It is **short-lived** (2 hours). A preview link is a working tool for the
//     hour somebody is drawing, not a URL to paste into a chat — and one that
//     lived for ever would be the draft made public by accident.
//   - It carries **no permissions**: holding it shows the draft layout of one
//     brand and nothing else. It is not a session and cannot become one.
//
// The token document is written by the control plane straight into the tenant's
// database, exactly as an export grant is — one writer, one reader, and no new
// path from a tenant container back to the control plane.

// PreviewTTL is how long a preview link works for.
const PreviewTTL = 2 * time.Hour

// previewDesign returns the draft when the request carries a valid token.
//
// Returns nil for everything else — an absent token, an expired one, a token for
// another brand — and never explains which. A visitor who guesses at the
// parameter learns nothing, and the console operator either sees their draft or
// sees the published site, which is itself the diagnosis.
func (h *Handler) previewDesign(r *http.Request, brandID primitive.ObjectID) *models.PageDesign {
	token := strings.TrimSpace(r.URL.Query().Get("preview"))
	// Long enough that it cannot be typed at, and refused before touching the
	// database so a flood of short guesses costs nothing.
	if len(token) < 24 || len(token) > 128 {
		return nil
	}

	var grant struct {
		Token     string             `bson:"token"`
		BrandID   primitive.ObjectID `bson:"brandId"`
		ExpiresAt time.Time          `bson:"expiresAt"`
	}
	if err := h.Store.DesignPreviews.FindOne(r.Context(),
		bson.M{"token": token}).Decode(&grant); err != nil {
		return nil
	}
	// ⚠️ Checked in code as well as by the TTL index. Mongo's expiry sweep runs
	// about once a minute, so "the document is gone" is not the same fact as "the
	// token is still valid" — and the gap is exactly when somebody would test
	// whether an old link still works.
	if time.Now().After(grant.ExpiresAt) {
		return nil
	}
	// A token for one brand does not show another's draft. Cheap to check, and
	// the alternative is a multi-brand company where one preview link opens every
	// unpublished design in the account.
	if grant.BrandID != brandID {
		return nil
	}

	var d models.PageDesign
	if err := h.Store.Designs.FindOne(r.Context(), bson.M{
		"brandId": brandID,
		"status":  models.DesignDraft,
	}).Decode(&d); err != nil {
		return nil
	}
	d.Sanitize()
	if len(d.Sections) == 0 {
		// An empty draft previews as the template rather than as a blank page —
		// the same rule the published path follows, and the honest answer to "I
		// have not drawn anything yet".
		return nil
	}
	// ⚠️ **Marked as published, deliberately**, and not because the draft is.
	//
	// `status` is not a description here, it is an instruction: it is what both
	// `Renderable()` and the site's renderer read as "draw this instead of the
	// template". A preview is precisely a request to draw the draft, so leaving the
	// status alone made every preview fall through to the template — the whole
	// feature failing while every individual piece looked correct.
	//
	// This copy exists only inside this response; nothing writes it back.
	d.Status = models.DesignPublished
	return &d
}
