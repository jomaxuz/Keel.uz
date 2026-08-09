package handlers

import (
	"embed"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"keel-control/internal/httpx"
)

// The starting gallery: five layouts, ready to apply.
//
// ⚠️ **Built into the binary rather than seeded into the database**, and the
// reason is what a seed would cost: a row written once at first boot cannot be
// corrected — an improved template would have to be migrated into every
// installation, and a deleted one would come back. In the binary they simply are
// what this version ships, and `git` is their history.
//
// They are read-only. Applying one **copies** its bands into the tenant's draft,
// exactly as applying a saved template does, so editing a layout afterwards never
// reaches back into the gallery and a customer's page is never redrawn by an
// update to ours.
//
// ⚠️ **Every string is in three languages and every colour is a token.** That is
// not decoration: a template with Uzbek-only text would arrive on a Russian
// customer's site as a half-translated page, and one with a hex colour would keep
// that colour in dark mode and ignore the accent the restaurant chose. A gallery
// is where those mistakes get copied twenty times.
//
// The photographs are the **seeded dish images** already shipped with every
// install (`/uploads/seed/...`), not files fetched from the web. Two reasons, and
// the second is the one that matters: they exist on every tenant from first boot,
// so a template never lands with a broken image — and a stock photo pulled off a
// search result carries a licence nobody in this chain has read.

//go:embed templates/templates.json templates/schema.json
var templateFS embed.FS

type builtinTemplate struct {
	Name string `json:"name"`
	// One line on what the layout is for. Shown in the gallery, because a list of
	// five names is a list nobody can choose from.
	Note     string          `json:"note"`
	Sections []designSection `json:"sections"`
}

var (
	builtinsOnce sync.Once
	builtins     []builtinTemplate
)

// builtinTemplates parses the embedded gallery once.
//
// A parse failure returns an empty gallery rather than stopping anything: the
// console's own templates and every other screen keep working, and the missing
// gallery is visible in one place instead of taking the page with it.
func builtinTemplates() []builtinTemplate {
	builtinsOnce.Do(func() {
		raw, err := templateFS.ReadFile("templates/templates.json")
		if err != nil {
			log.Printf("design templates: %v", err)
			return
		}
		if err := json.Unmarshal(raw, &builtins); err != nil {
			log.Printf("design templates: %v", err)
			builtins = nil
		}
	})
	return builtins
}

// DesignSchema is what the console draws its settings panel from.
//
// ⚠️ **Served rather than hardcoded in the console**, which is the whole change:
// until now every element type needed its own hand-written panel, so adding one
// meant editing the editor. A section now declares what it can be asked and the
// console renders the controls — the same arrangement Shopify's theme editor uses,
// and the reason theirs stays consistent as sections are added.
//
// Public within the console session, and deliberately unversioned: it describes
// the sections this deployment can render, so a console and the tenant it is
// editing are never out of step by construction.
func (h *Handler) DesignSchema(w http.ResponseWriter, r *http.Request) {
	raw, err := templateFS.ReadFile("templates/schema.json")
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	// Passed through as bytes: parsing and re-encoding it here would only add a
	// place for the two shapes to disagree.
	_, _ = w.Write(raw)
}
