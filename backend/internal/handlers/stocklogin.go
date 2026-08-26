package handlers

// ---- The storekeeper's way into the panel ----
//
// ⚠️ **Two account systems, and a technologist is in the wrong one.** Panel
// logins are `admin_user` (owner, manager); the people who work in the building
// are `staff`, with roles a restaurant names itself. A technologist typing
// their till credentials into the panel got "login yoki parol noto'g'ri", which
// is true and reads like a broken account rather than like a boundary.
//
// The counting, the recipes and the purchase records they are responsible for
// all live in the panel. So a staff account holding `stock` gets in — and gets
// nothing else.
//
// ⚠️ **The safe direction is deny-by-default, and it is the whole design.** The
// alternative — let the token into the admin group and gate the sections it
// must not see — means every route added afterwards is visible to a
// storekeeper until somebody remembers otherwise. Here a path that is not on
// the list is refused, so the failure of a future edit is a storekeeper seeing
// too little.

import (
	"net/http"
	"strings"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

// RoleStock is what a storekeeper's panel token says it is.
//
// ⚠️ **Its own role rather than "manager" with a flag.** Every `requireOwner`
// and every `owner, manager` group in the router refuses it without being
// edited, which means the gate is the router's existing shape rather than a
// list somebody has to keep up to date.
const RoleStock = "stock"

// stockPaths is everything a storekeeper's token may reach, by prefix.
//
// ⚠️ **Prefixes, and deliberately narrow ones.** `/admin/stock` covers balances
// and movement; it does not cover `/admin/stop-list`, which is a different
// screen with a similar name. Each line here was chosen by opening the router
// and reading what it serves, not by pattern.
var stockPaths = []string{
	"/admin/ingredients",
	"/admin/warehouses",
	"/admin/purchases",
	"/admin/suppliers",
	"/admin/writeoffs",
	"/admin/transfers",
	"/admin/stocktake",
	"/admin/stock/",
	"/admin/production",
	// The two reports that are about the store rather than about trade.
	"/admin/reports/stock",
	"/admin/reports/suppliers",
	// ⚠️ Read-only needs of the screens above: an ingredient's card names a
	// dish, and a purchase names a branch. Without these the pages load empty
	// and look broken.
	"/admin/menu",
	"/admin/categories",
	"/admin/me",
	"/admin/branches",
	"/admin/brands",
}

// stockAllowed reports whether a storekeeper may reach this path.
func stockAllowed(path string) bool {
	for _, p := range stockPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// StockGate refuses a storekeeper's token anything outside the store.
//
// ⚠️ **Applied to the whole admin group**, so it is one place rather than a
// check per handler — and a handler added tomorrow is covered by having been
// added, not by somebody remembering.
func (h *Handler) StockGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.ClaimsFrom(r.Context())
		if claims != nil && claims.Role == RoleStock && !stockAllowed(r.URL.Path) {
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// stockLogin is the second half of the panel's login: a staff account with the
// stock permission, when no admin account matched.
//
// ⚠️ **Tried only after the admin lookup fails**, so nothing about an existing
// login changes — and an owner who happens to share a username with an employee
// still signs in as the owner.
//
// ⚠️ **The same refusal either way.** A staff account without `stock` gets
// "invalid credentials" rather than "you do not have permission": the login form
// is reachable by anybody, and telling a stranger which usernames exist and
// what they are missing is how a password gets guessed at leisure.
func (h *Handler) stockLogin(w http.ResponseWriter, r *http.Request, req loginRequest) bool {
	var s models.Staff
	if err := h.Store.Staff.FindOne(r.Context(),
		bson.M{"username": normalizeUsername(req.Username)}).Decode(&s); err != nil {
		return false
	}
	if bcrypt.CompareHashAndPassword([]byte(s.PasswordHash), []byte(req.Password)) != nil {
		return false
	}
	h.withRole(r.Context(), &s)
	if !s.Can(models.PermStock) {
		return false
	}
	token, err := auth.Generate(h.Cfg.JWTSecret, s.ID.Hex(), RoleStock)
	if err != nil {
		return false
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"token": token,
		// Shaped like an admin user because the panel's session code reads it
		// that way, and a second shape would be a second code path drawing the
		// same header.
		"user": map[string]any{
			"id":       s.ID.Hex(),
			"username": s.Username,
			"name":     s.Name,
			"role":     RoleStock,
			"branchId": s.BranchID.Hex(),
		},
	})
	return true
}
