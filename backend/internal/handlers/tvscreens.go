package handlers

// ---- The panel's side of the televisions ----
//
// Adding a screen is typing the code that television is showing. Removing one
// is a single row, because the blunt tool — rotating the branch's version —
// blanks every screen in the building and means somebody re-pairs each of them
// standing on a chair.
//
// ⚠️ **Sold per screen, so the count has to be a fact.** The same argument as
// registers (see tilldevices.go): a limit nobody enforces is worse than no
// limit, because the customers who read the price list honestly end up paying
// for the ones who did not. And the same shape of enforcement — at the door
// only, never on a screen already hanging on a wall.

import (
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// tvScreenFilter is every live screen of a branch.
//
// ⚠️ Version-matched, so revoking a branch's screens frees their slots too: a
// manager who revokes after a theft has, in the same act, told us those screens
// are gone. Leaving the rows counting against the cap would mean the
// replacement television is refused by a limit filled with stolen hardware.
func tvScreenFilter(branchID primitive.ObjectID, version int) bson.M {
	return bson.M{"branchId": branchID, "version": version}
}

// tvCapExceeded decides whether one more screen is one too many.
//
// Pure, so the two answers that must never be got backwards can be argued with
// in a test rather than in a restaurant: an install with no subscription and a
// plan with no screen count both mean **no limit at all**.
//
// ⚠️ **No subscription means no cap.** Every install that predates this, and
// every one we host without a counter, must keep pairing screens exactly as
// before — the same rule the module gate follows, and for the same reason: a
// limit that switches itself on for customers who were never sold one takes a
// working feature away on a deploy.
func tvCapExceeded(s *models.Subscription, count int) bool {
	if s == nil || !s.Enabled || s.Screens <= 0 {
		return false
	}
	return count >= s.Screens
}

// countTVScreens is how many televisions this branch currently has paired.
func (h *Handler) countTVScreens(r *http.Request, branch models.Branch) (int, error) {
	n, err := h.Store.TVScreens.CountDocuments(
		r.Context(), tvScreenFilter(branch.ID, branch.TVVersion))
	return int(n), err
}

// AdminTVScreens lists the televisions of one branch.
func (h *Handler) AdminTVScreens(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.requireBranchAccess(r, id); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	branch, err := h.branchByID(r, id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "filial topilmadi")
		return
	}
	cur, err := h.Store.TVScreens.Find(r.Context(),
		tvScreenFilter(id, branch.TVVersion),
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ An empty slice, never nil: the panel reads `.length` off this, and Go
	// marshals a nil slice as `null`. The JSON trap this codebase has hit twice.
	rows := []models.TVScreen{}
	if err := cur.All(r.Context(), &rows); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	limit := 0
	if s := h.subscription(r.Context()); s != nil && s.Enabled {
		limit = s.Screens
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"screens": rows,
		// 0 means no cap, and the panel says "cheksiz" rather than "0 ekran".
		"limit": limit,
	})
}

type tvClaimRequest struct {
	Code     string `json:"code"`
	BranchID string `json:"branchId"`
	Name     string `json:"name"`
	Mode     string `json:"mode"`
}

// AdminTVClaim pairs the television showing this code to a branch.
//
// ⚠️ **This is the act that spends a paid slot and puts a screen in a room**,
// so it needs a person with access to that branch — the code alone is not
// authority. Anybody in the dining room can read the code off the wall; only
// somebody signed into the panel can decide it belongs to Chilonzor.
func (h *Handler) AdminTVClaim(w http.ResponseWriter, r *http.Request) {
	var req tvClaimRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	branchID, err := objectID(strings.TrimSpace(req.BranchID))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "filialni tanlang")
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	branch, err := h.branchByID(r, branchID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "filial topilmadi")
		return
	}
	code := normalizeTVCode(req.Code)
	if code == "" {
		httpx.Error(w, http.StatusBadRequest, "ekrandagi kodni kiriting")
		return
	}
	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = models.TVModeContent
	}
	if !models.ValidTVMode(mode) {
		httpx.Error(w, http.StatusBadRequest, "ekran rejimi noma'lum")
		return
	}

	pairing, err := h.tvPairingByCode(r, code)
	if err != nil {
		if err == errTVCodeUnknown {
			// ⚠️ 404 rather than 400, and the message names the next step: the
			// ordinary cause is that the code rotated while somebody was
			// typing, and "look at the screen again" is the whole fix.
			httpx.Error(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if pairing.Token != "" {
		httpx.Error(w, http.StatusConflict, "bu kod allaqachon ishlatilgan")
		return
	}

	// ⚠️ **The same television re-paired replaces its row.** A set that was
	// factory-reset, or moved from one branch to another, asks for a new code
	// with the same install id — and adding a second row would quietly spend a
	// second paid slot for one wall.
	var existing models.TVScreen
	reused := h.Store.TVScreens.FindOne(r.Context(),
		bson.M{"installId": pairing.InstallID}).Decode(&existing) == nil

	if !reused {
		n, err := h.countTVScreens(r, *branch)
		// ⚠️ A failed count does not refuse: Mongo being briefly unreachable is
		// not the restaurant exceeding its plan, and the visible result of
		// getting this backwards is a screen that cannot be set up during an
		// incident somebody is already dealing with.
		if err == nil && tvCapExceeded(h.subscription(r.Context()), n) {
			s := h.subscription(r.Context())
			httpx.JSON(w, http.StatusPaymentRequired, map[string]any{
				"error":   httpx.T(w, "tarifingizdagi ekranlar soni to'lgan"),
				"screens": n,
				"limit":   s.Screens,
			})
			return
		}
	}

	now := time.Now()
	name := clampText(req.Name, 60)
	if name == "" {
		name = branch.Name
	}
	screen := models.TVScreen{
		BranchID:  branchID,
		Name:      name,
		Mode:      mode,
		Version:   branch.TVVersion,
		InstallID: pairing.InstallID,
		PairedBy:  h.adminName(r),
		CreatedAt: now,
	}
	if reused {
		screen.ID = existing.ID
		screen.CreatedAt = existing.CreatedAt
		if _, err := h.Store.TVScreens.UpdateByID(r.Context(), existing.ID,
			bson.M{"$set": bson.M{
				"branchId": branchID, "name": name, "mode": mode,
				"version": branch.TVVersion, "pairedBy": screen.PairedBy,
			}}); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		res, err := h.Store.TVScreens.InsertOne(r.Context(), screen)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		screen.ID = oidOf(res.InsertedID)
	}

	token, err := h.issueTVToken(screen)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// The television collects this on its next poll. ⚠️ Written to the pairing
	// rather than returned to the panel: the token belongs to the wall, and a
	// browser that has held it once is a browser it can leak from.
	if _, err := h.Store.TVPairings.UpdateByID(r.Context(), pairing.ID,
		bson.M{"$set": bson.M{"token": token, "screenId": screen.ID}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.logAction(r, ActSettingsUpdate, "branch", branchID.Hex(), name,
		"TV ekran ulandi")
	httpx.JSON(w, http.StatusOK, screen)
}

type tvScreenPatch struct {
	Name *string `json:"name"`
	Mode *string `json:"mode"`
	// Which room this set hangs in. A pointer, so an empty string can mean
	// "unplace it" while an absent field leaves it where it is.
	Zone *string `json:"zone"`
}

// AdminUpdateTVScreen renames a screen or changes what it shows.
func (h *Handler) AdminUpdateTVScreen(w http.ResponseWriter, r *http.Request) {
	branchID, screen, ok := h.tvScreenOfBranch(w, r)
	if !ok {
		return
	}
	var req tvScreenPatch
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := bson.M{}
	unset := bson.M{}
	if req.Name != nil {
		if name := clampText(*req.Name, 60); name != "" {
			set["name"] = name
		}
	}
	if req.Mode != nil {
		mode := strings.TrimSpace(*req.Mode)
		if !models.ValidTVMode(mode) {
			httpx.Error(w, http.StatusBadRequest, "ekran rejimi noma'lum")
			return
		}
		set["mode"] = mode
	}
	// Where this set hangs. Lowercased on the way in for the reason cleanZones
	// carries: the zone is matched by string equality against the slide's list,
	// and "Zal" against "zal" is a screen that plays nothing.
	zoneChanged := false
	if req.Zone != nil {
		zone := strings.ToLower(strings.TrimSpace(*req.Zone))
		zoneChanged = zone != screen.Zone
		if zone == "" {
			unset["zone"] = ""
		} else {
			set["zone"] = clampText(zone, 40)
		}
	}
	if len(set) == 0 && len(unset) == 0 {
		httpx.JSON(w, http.StatusOK, screen)
		return
	}
	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	if _, err := h.Store.TVScreens.UpdateByID(r.Context(), screen.ID,
		update); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ **Moving a screen between zones has to bump the branch's content
	// version, and nothing about the playlist changed.** A television re-reads
	// its list *only* when that number moves; without this the set keeps
	// playing the zone it used to be in — for ever, silently, on a wall
	// somebody has just told the panel about. The cost is one extra fetch by
	// every screen in the branch; the alternative is a screen that can never be
	// moved.
	if zoneChanged {
		h.bumpTVContent(r, branchID)
	}
	h.logAction(r, ActSettingsUpdate, "branch", branchID.Hex(), screen.Name,
		"TV ekran sozlandi")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// AdminRemoveTVScreen unpairs one television, freeing its slot.
//
// ⚠️ **One screen, unlike revoking the branch.** Revocation exists for a set
// that walked out of the building and darkens every television at once; this is
// the ordinary case — a screen replaced, moved, or paired by mistake.
func (h *Handler) AdminRemoveTVScreen(w http.ResponseWriter, r *http.Request) {
	branchID, screen, ok := h.tvScreenOfBranch(w, r)
	if !ok {
		return
	}
	if _, err := h.Store.TVScreens.DeleteOne(r.Context(),
		bson.M{"_id": screen.ID, "branchId": branchID}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActSettingsUpdate, "branch", branchID.Hex(), screen.Name,
		"TV ekran uzildi")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// AdminRevokeTVScreens unpairs every television of a branch at once.
//
// ⚠️ **The answer to a set leaving the building**, and deliberately separate
// from removing one row: after this, every screen here goes back to showing a
// pairing code, and somebody walks round the restaurant. That is a real cost —
// which is why the panel asks first and why the single-screen button exists.
func (h *Handler) AdminRevokeTVScreens(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.requireBranchAccess(r, id); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	branch, err := h.branchByID(r, id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "filial topilmadi")
		return
	}
	next := branch.TVVersion + 1
	if _, err := h.Store.Branches.UpdateByID(r.Context(), id, bson.M{"$set": bson.M{
		"tvVersion": next, "updatedAt": time.Now(),
	}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ The rows go with the version, so the paid slots come back. A screen
	// nobody can reach still counting against the plan is a restaurant that
	// cannot replace the television it just lost.
	_, _ = h.Store.TVScreens.DeleteMany(r.Context(),
		bson.M{"branchId": id, "version": bson.M{"$lt": next}})

	h.logAction(r, ActSettingsUpdate, "branch", id.Hex(), branch.Name,
		"barcha TV ekranlar uzildi")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "version": next})
}

// tvScreenOfBranch loads the screen named in the URL, inside the branch named
// in the URL.
//
// ⚠️ **The branch is in the filter, not merely checked**: `_id` alone must
// never select a document — the rule the security section of CLAUDE.md is built
// on. A manager typing another branch's screen id gets nothing rather than
// unpairing somebody else's dining room.
func (h *Handler) tvScreenOfBranch(w http.ResponseWriter, r *http.Request) (primitive.ObjectID, models.TVScreen, bool) {
	var screen models.TVScreen
	branchID, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return branchID, screen, false
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return branchID, screen, false
	}
	screenID, err := objectID(chi.URLParam(r, "screenId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return branchID, screen, false
	}
	if err := h.Store.TVScreens.FindOne(r.Context(), bson.M{
		"_id": screenID, "branchId": branchID,
	}).Decode(&screen); err != nil {
		httpx.Error(w, http.StatusNotFound, "ekran topilmadi")
		return branchID, screen, false
	}
	return branchID, screen, true
}
