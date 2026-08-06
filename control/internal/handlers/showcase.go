package handlers

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// What a freshly seeded tenant calls itself before its owner renames it
// (backend `internal/seed`). Treated as "not set" rather than as a name: it is
// a placeholder, and it is the one thing a reference strip must never print.
const seedRestaurantName = "My Restaurant"

// The customers keel.uz shows as references.
//
// Two decisions shape this:
//
//   - **Opt-in.** Only tenants an operator has ticked appear. Putting a
//     restaurant's brand on our marketing page is their decision; a customer
//     who finds their logo there without being asked is a customer with a
//     complaint, and a reference page cannot survive being resented.
//
//   - **The logo comes from the customer's own site, live.** It is read from
//     their database and served from their own domain, so a restaurant that
//     rebrands on Tuesday is rebranded here on Tuesday. Copying the image into
//     the control plane would mean a wall of logos slowly going out of date
//     with nobody responsible for noticing.
//
// Cached, because this is the landing page: it is the most-requested page on
// the platform and it must not dial N tenant databases per visitor. Five
// minutes is far fresher than a marketing page needs and far cheaper than the
// alternative.

// Partner is one reference on the landing page.
type Partner struct {
	Name string `json:"name"`
	// Absolute, on the customer's own domain — their site serves it already.
	LogoURL string `json:"logoUrl"`
	// Where the logo links: the customer's own site.
	URL string `json:"url"`
}

var showcase = struct {
	sync.Mutex
	at    time.Time
	items []Partner
}{}

const showcaseTTL = 5 * time.Minute

// Partners is the public list keel.uz renders. No auth: it is a marketing
// page, and everything in it is already published on the customers' own sites.
func (h *Handler) Partners(w http.ResponseWriter, r *http.Request) {
	showcase.Lock()
	fresh := time.Since(showcase.at) < showcaseTTL && showcase.items != nil
	items := showcase.items
	showcase.Unlock()

	if !fresh {
		items = h.collectPartners(r.Context())
		showcase.Lock()
		showcase.at, showcase.items = time.Now(), items
		showcase.Unlock()
	}
	// Never null: the landing page maps over this.
	if items == nil {
		items = []Partner{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"partners": items})
}

func (h *Handler) collectPartners(ctx context.Context) []Partner {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cur, err := h.Store.Tenants.Find(ctx, bson.M{
		"showcase": true,
		// Active only. A suspended customer's site does not answer, so its
		// logo would be a broken image linking to a "payment pending" page —
		// on the page whose whole job is to look like a working platform.
		"status": models.StatusActive,
	})
	if err != nil {
		return nil
	}
	var tenants []models.Tenant
	if err := cur.All(ctx, &tenants); err != nil {
		return nil
	}

	out := make([]Partner, 0, len(tenants))
	for _, t := range tenants {
		if len(t.Domains) == 0 {
			continue
		}
		site := "https://" + t.Domains[0]

		// The face the guest sees, in the order the tenant's own site resolves
		// it: **brand first, company second, our console last.**
		//
		// ⚠️ Reading only `restaurant` was wrong, and wrong in a way that
		// looked like an empty field. Once a tenant has a brand — every one of
		// them does — its settings page saves the name, the logo and the cover
		// onto the **brand** document and deletes them from the company
		// payload (frontend BRAND_FIELDS). So `restaurant.logoUrl` stays empty
		// forever and `restaurant.name` keeps the seeded "My Restaurant",
		// which is exactly what the strip showed: a placeholder and no logo,
		// for a customer who had set both. Same overlay as `applyBrand` in the
		// tenant server — one order of preference, two places.
		db := h.Store.TenantDB(t.DBName())

		var brand, rest siteIdentity
		// The primary brand: the first active one, in the tenant's own order —
		// the same one its site opens on.
		_ = db.Collection("brand").FindOne(ctx,
			bson.M{"isActive": true},
			options.FindOne().SetSort(bson.D{{Key: "sortOrder", Value: 1}}),
		).Decode(&brand)
		_ = db.Collection("restaurant").FindOne(ctx, bson.M{}).Decode(&rest)

		name, logo := resolveIdentity(brand, rest, t.Name)
		// A reference without a logo still belongs on the strip — the name is
		// rendered as a wordmark. Dropping it would silently punish the
		// customer who never uploaded one.
		out = append(out, Partner{Name: name, LogoURL: absolute(site, logo), URL: site})
	}
	return out
}

// siteIdentity is the name-and-logo pair as either document stores it.
type siteIdentity struct {
	Name    string `bson:"name"`
	LogoURL string `bson:"logoUrl"`
}

// resolveIdentity picks the face to show, brand first.
//
// The order matters and is not obvious: a tenant's settings page saves the
// name, logo and cover onto the **brand** once one exists, and deletes them
// from the company payload. Read the company alone and every showcased
// customer comes back as the seeded "My Restaurant" with no logo — which is
// what shipped.
//
// The console name is the last resort rather than the first because it is what
// *we* typed when the account was opened; the other two are what the owner
// maintains. But a placeholder is worse than either, so it never wins.
func resolveIdentity(brand, rest siteIdentity, consoleName string) (name, logo string) {
	name, logo = strings.TrimSpace(brand.Name), strings.TrimSpace(brand.LogoURL)
	if unset(name) {
		name = strings.TrimSpace(rest.Name)
	}
	if logo == "" {
		logo = strings.TrimSpace(rest.LogoURL)
	}
	if unset(name) {
		name = strings.TrimSpace(consoleName)
	}
	return name, logo
}

// unset treats the seeded placeholder as no name at all.
func unset(name string) bool { return name == "" || name == seedRestaurantName }

// absolute turns the tenant's own stored path ("/uploads/logo.png") into a URL
// the browser on keel.uz can load. Already-absolute values are left alone: a
// restaurant that pasted a CDN link keeps it.
func absolute(site, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return strings.TrimRight(site, "/") + "/" + strings.TrimLeft(path, "/")
}
