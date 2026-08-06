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
)

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

		// The name and logo as the restaurant maintains them, not as we typed
		// them into the console when the account was opened.
		name, logo := t.Name, ""
		var rest struct {
			Name    string `bson:"name"`
			LogoURL string `bson:"logoUrl"`
		}
		err := h.Store.TenantDB(t.DBName()).Collection("restaurant").
			FindOne(ctx, bson.M{}).Decode(&rest)
		if err == nil {
			if rest.Name != "" {
				name = rest.Name
			}
			logo = absolute(site, rest.LogoURL)
		}
		// A reference without a logo still belongs on the strip — the name is
		// rendered as a wordmark. Dropping it would silently punish the
		// customer who never uploaded one.
		out = append(out, Partner{Name: name, LogoURL: logo, URL: site})
	}
	return out
}

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
