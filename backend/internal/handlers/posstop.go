package handlers

// The till's stop list, mirrored onto the site.
//
// A kitchen that runs iiko or Poster already has a place where "we are out of
// lag'mon" is said once, at the counter, by the person who found the empty pot.
// Until now the website did not hear it: the dish stayed orderable, a guest paid
// for it, and somebody had to ring back and apologise. Asking the owner to say
// it a second time in our panel is asking them to keep two stop lists in sync by
// hand during a rush — which is another way of saying the site's list is wrong.
//
// So the till is polled and its stopped products are mapped back through the
// dish→product links the owner already set up (/admin/pos). Three decisions
// shape the rest:
//
//   - **The mirror is a second list** (branch.posSoldOut), never merged into the
//     one the counter taps. Two writers on one field undo each other, and each
//     undo looks like the feature is broken rather than busy.
//   - **An empty answer is not "nothing is stopped".** A till that returns no
//     products at all has failed in a way that returns 200, and treating that as
//     good news would put every stopped dish back on sale at once. The previous
//     list is kept and the reason is stored.
//   - **A disconnected till clears the mirror.** Otherwise a restaurant that
//     switched POS would carry a frozen stop list forever, with nothing in the
//     panel able to lift it.

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/pos"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// How often every connected branch is asked. Short enough that a dish stopped
// mid-service disappears from the site while the guest is still browsing, long
// enough that a restaurant with four branches is not hammering its own till.
const posStopSyncEvery = 3 * time.Minute

// errPOSEmptyCatalogue is the guard above: a successful call that answered with
// nothing. Recorded like any other failure, and the mirror is left alone.
var errPOSEmptyCatalogue = errors.New("kassa bo'sh ro'yxat qaytardi — stop list o'zgartirilmadi")

// StartPOSStopSync polls every connected till in the background.
//
// Started from cmd/server and stopped with the process. Failures are logged and
// nothing else: a till that is down must never take the site with it, and the
// panel already shows when each branch was last read successfully.
func (h *Handler) StartPOSStopSync(ctx context.Context) {
	go func() {
		// A short delay rather than an immediate run: the process has just come
		// up and the first request of the morning should not queue behind five
		// outbound calls to somebody else's cloud.
		timer := time.NewTimer(20 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		ticker := time.NewTicker(posStopSyncEvery)
		defer ticker.Stop()
		for {
			h.syncAllPOSStopLists(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (h *Handler) syncAllPOSStopLists(ctx context.Context) {
	cur, err := h.Store.POSSettings.Find(ctx, bson.M{
		"enabled":  true,
		"provider": bson.M{"$ne": ""},
	})
	if err != nil {
		log.Printf("pos stop list: %v", err)
		return
	}
	var rows []models.POSSettings
	if err := cur.All(ctx, &rows); err != nil {
		log.Printf("pos stop list: %v", err)
		return
	}
	for _, s := range rows {
		if s.BranchID.IsZero() {
			continue
		}
		if _, err := h.syncPOSStopList(ctx, s.BranchID); err != nil {
			log.Printf("pos stop list (branch %s): %v", s.BranchID.Hex(), err)
		}
	}
}

// syncPOSStopList reads one branch's till and rewrites that branch's mirror.
//
// Returns how many of our dishes ended up stopped. The error is both returned
// and stored on the branch: the caller may be a button in the panel, but the
// caller is usually a ticker with nobody watching, and the owner still needs to
// find out that the mirror stopped updating an hour ago.
func (h *Handler) syncPOSStopList(ctx context.Context, branchID primitive.ObjectID) (int, error) {
	provider, _, err := h.posFor(ctx, branchID)
	if err != nil {
		h.recordStopSync(ctx, branchID, nil, err)
		return 0, err
	}
	if provider == nil {
		// Nothing connected: the mirror is emptied rather than frozen. What the
		// counter marked by hand is untouched — that list is not ours.
		h.clearPOSStopList(ctx, branchID)
		return 0, nil
	}

	callCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	products, err := provider.Products(callCtx)
	if err != nil {
		h.recordStopSync(ctx, branchID, nil, err)
		return 0, err
	}
	if len(products) == 0 {
		h.recordStopSync(ctx, branchID, nil, errPOSEmptyCatalogue)
		return 0, errPOSEmptyCatalogue
	}

	mappings, err := h.posMappingsOf(ctx, branchID)
	if err != nil {
		h.recordStopSync(ctx, branchID, nil, err)
		return 0, err
	}
	ids := stoppedMenuItems(products, mappings)

	h.recordStopSync(ctx, branchID, ids, nil)
	return len(ids), nil
}

// stoppedMenuItems is the translation itself: which of our dishes the till's
// stopped products correspond to.
//
// Kept as a plain function so it can be tested without a database or a till —
// the two rules that matter (an unmapped dish is never stopped, a product id is
// compared trimmed) are both the kind that look obviously right and go wrong
// silently.
func stoppedMenuItems(products []pos.Product, mappings []models.POSMapping) []primitive.ObjectID {
	stopped := map[string]bool{}
	for _, p := range products {
		if id := strings.TrimSpace(p.ID); p.Unavailable && id != "" {
			stopped[id] = true
		}
	}
	ids := []primitive.ObjectID{}
	for _, m := range mappings {
		if m.MenuItemID.IsZero() {
			continue
		}
		if stopped[strings.TrimSpace(m.POSProductID)] {
			ids = append(ids, m.MenuItemID)
		}
	}
	return ids
}

// posMappingsOf loads a branch's dish→product links.
func (h *Handler) posMappingsOf(ctx context.Context, branchID primitive.ObjectID) ([]models.POSMapping, error) {
	cur, err := h.Store.POSMappings.Find(ctx, bson.M{"branchId": branchID})
	if err != nil {
		return nil, err
	}
	rows := []models.POSMapping{}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// recordStopSync writes the outcome. On failure only the reason and the time are
// written: the previous mirror is the last thing the till actually said, and a
// stale stop list is safer than an empty one — it refuses orders the kitchen may
// still be unable to cook, rather than accepting orders it certainly cannot.
func (h *Handler) recordStopSync(
	ctx context.Context, branchID primitive.ObjectID,
	ids []primitive.ObjectID, cause error,
) {
	set := bson.M{"posSoldOutAt": time.Now()}
	if cause != nil {
		set["posSoldOutError"] = clampText(cause.Error(), 300)
	} else {
		if ids == nil {
			ids = []primitive.ObjectID{}
		}
		set["posSoldOut"] = ids
		set["posSoldOutError"] = ""
	}
	if _, err := h.Store.Branches.UpdateByID(ctx, branchID, bson.M{"$set": set}); err != nil {
		log.Printf("pos stop list save (branch %s): %v", branchID.Hex(), err)
	}
}

func (h *Handler) clearPOSStopList(ctx context.Context, branchID primitive.ObjectID) {
	_, _ = h.Store.Branches.UpdateByID(ctx, branchID, bson.M{"$set": bson.M{
		"posSoldOut":      []primitive.ObjectID{},
		"posSoldOutError": "",
	}, "$unset": bson.M{"posSoldOutAt": ""}})
}

// ---- Panel ----

// AdminSyncPOSStopList is the "read the till now" button.
//
// The poller already runs, so this exists for the two moments it does not cover:
// just after the dish links were edited, and while somebody is standing in the
// panel wondering whether the connection works at all.
func (h *Handler) AdminSyncPOSStopList(w http.ResponseWriter, r *http.Request) {
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	count, err := h.syncPOSStopList(r.Context(), branchID)
	if err != nil {
		// 200 with ok:false, like the POS ping: the request was handled, the
		// till is what failed, and the panel shows the reason in its own words.
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": false, "message": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "stopped": count})
}

// stopListRow is one dish on the stop-list screen.
type stopListRow struct {
	MenuItemID string `json:"menuItemId"`
	Name       string `json:"name"`
	CategoryID string `json:"categoryId"`
	Category   string `json:"category"`
	ImageURL   string `json:"imageUrl"`
	Price      int    `json:"price"`
	// Off the menu entirely, everywhere. Shown greyed rather than hidden: an
	// owner looking for a missing dish should find it here with the reason.
	Hidden bool `json:"hidden"`
	// Marked by hand, at this branch, for today.
	Manual bool `json:"manual"`
	// Stopped in the till. Not togglable from here.
	POS bool `json:"pos"`
	// What this dish is linked to over there, when it is linked at all. An
	// unmapped dish can never be stopped by the till, and that is worth seeing
	// on this screen rather than discovering when a guest orders it.
	POSProduct string `json:"posProduct"`
	Mapped     bool   `json:"mapped"`
}

// AdminStopList is the whole stop-list screen in one response.
//
// One request rather than three (menu, categories, mapping) because the screen
// is a single question — "what is off sale right now, and why" — and answering
// it from three calls means three chances to render a half-true page.
func (h *Handler) AdminStopList(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	branchID := h.scopeBranch(r, scope)
	if branchID.IsZero() {
		// Deliberately not an empty list: what has run out belongs to one
		// kitchen, and a company-wide view of it would be a list of dishes with
		// no answer to "where".
		httpx.Error(w, http.StatusBadRequest, "filial tanlanmagan")
		return
	}
	branch, err := h.branchByID(r, branchID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "filial topilmadi")
		return
	}

	ctx := r.Context()
	cur, err := h.Store.Menu.Find(ctx, scope.brandFilter(bson.M{}),
		options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := []models.MenuItem{}
	_ = cur.All(ctx, &items)

	catNames := map[primitive.ObjectID]string{}
	if ccur, err := h.Store.Categories.Find(ctx, scope.brandFilter(bson.M{})); err == nil {
		cats := []models.Category{}
		_ = ccur.All(ctx, &cats)
		for _, c := range cats {
			catNames[c.ID] = c.Name
		}
	}

	mapped := map[primitive.ObjectID]models.POSMapping{}
	if rows, err := h.posMappingsOf(ctx, branchID); err == nil {
		for _, m := range rows {
			if strings.TrimSpace(m.POSProductID) != "" {
				mapped[m.MenuItemID] = m
			}
		}
	}

	rows := make([]stopListRow, 0, len(items))
	for _, it := range items {
		m, ok := mapped[it.ID]
		rows = append(rows, stopListRow{
			MenuItemID: it.ID.Hex(),
			Name:       it.Name,
			CategoryID: it.CategoryID.Hex(),
			Category:   catNames[it.CategoryID],
			ImageURL:   it.ImageURL,
			Price:      it.Price,
			Hidden:     !it.IsAvailable,
			Manual:     containsID(branch.SoldOut, it.ID),
			POS:        branch.IsPOSSoldOut(it.ID),
			POSProduct: m.POSProductName,
			Mapped:     ok,
		})
	}
	// Everything that is off sale first, then by category and name. The screen
	// is opened to answer "what is off right now", and that answer must not be
	// somewhere in the middle of a 200-dish menu.
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if off(a) != off(b) {
			return off(a)
		}
		if a.Category != b.Category {
			return a.Category < b.Category
		}
		return a.Name < b.Name
	})

	settings := h.posSettingsOf(ctx, branchID)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"branchId":   branchID.Hex(),
		"branchName": branch.Name,
		"items":      rows,
		"pos": map[string]any{
			"connected":  settings.Enabled && settings.Provider != "",
			"provider":   settings.Provider,
			"syncedAt":   branch.POSSoldOutAt,
			"syncError":  branch.POSSoldOutError,
			"everyMins":  int(posStopSyncEvery / time.Minute),
			"mappedItem": len(mapped),
		},
	})
}

func off(r stopListRow) bool { return r.Manual || r.POS }
