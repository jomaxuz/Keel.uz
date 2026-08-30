package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// Login authenticates an admin and returns a JWT.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var user models.AdminUser
	if err := h.Store.Admins.FindOne(r.Context(), bson.M{"username": req.Username}).Decode(&user); err != nil {
		// ⚠️ A storekeeper's own credentials, second and never first: an owner
		// who happens to share a username with an employee still signs in as
		// the owner. See stocklogin.go.
		if h.stockLogin(w, r, req) {
			return
		}
		httpx.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		if h.stockLogin(w, r, req) {
			return
		}
		httpx.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, err := auth.Generate(h.Cfg.JWTSecret, user.ID.Hex(), user.Role)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	now := time.Now()
	_, _ = h.Store.Admins.UpdateByID(r.Context(), user.ID,
		bson.M{"$set": bson.M{"lastLoginAt": now}})
	user.LastLoginAt = &now
	h.logLogin(r, &user)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  user,
	})
}

// Me returns the authenticated admin's profile.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	id, _ := objectID(claims.UserID)
	var user models.AdminUser
	if err := h.Store.Admins.FindOne(r.Context(), bson.M{"_id": id}).Decode(&user); err != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

type credentialsRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewUsername     string `json:"newUsername"`
	NewPassword     string `json:"newPassword"`
}

// ChangeCredentials lets the authenticated admin update their username and/or
// password after verifying the current password. Clears mustChangePassword.
func (h *Handler) ChangeCredentials(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	id, _ := objectID(claims.UserID)

	var req credentialsRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	var user models.AdminUser
	if err := h.Store.Admins.FindOne(r.Context(), bson.M{"_id": id}).Decode(&user); err != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)) != nil {
		httpx.Error(w, http.StatusUnauthorized, "joriy parol noto'g'ri")
		return
	}

	set := bson.M{"mustChangePassword": false}

	if req.NewUsername != "" && req.NewUsername != user.Username {
		// Ensure the new username is not taken by another admin.
		other := h.Store.Admins.FindOne(r.Context(), bson.M{
			"username": req.NewUsername,
			"_id":      bson.M{"$ne": id},
		})
		if other.Err() == nil {
			httpx.Error(w, http.StatusConflict, "bu login band")
			return
		}
		set["username"] = req.NewUsername
	}

	if req.NewPassword != "" {
		if len(req.NewPassword) < 6 {
			httpx.Error(w, http.StatusBadRequest, "parol kamida 6 belgi bo'lishi kerak")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		set["passwordHash"] = string(hash)
	}

	if _, err := h.Store.Admins.UpdateOne(r.Context(), bson.M{"_id": id}, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	changed := "parol"
	if _, ok := set["username"]; ok {
		changed = "login va parol"
	}
	h.logAction(r, ActAdminCredentials, "admin", id.Hex(), user.Username, changed)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// UpdateRestaurant updates the company profile.
//
// Deliberately a **partial** update: only the fields the request actually
// carries are written. The settings page no longer sends everything — name,
// logo and theme belong to the brand now, hours and delivery to the branch —
// and a whole-document $set would write those back as empty strings, quietly
// erasing the company's own record of them.
func (h *Handler) UpdateRestaurant(w http.ResponseWriter, r *http.Request) {
	// The company profile — currency, socials, the shared identity — is the
	// owner's. A branch manager edits their branch through /admin/branches.
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var rest models.Restaurant
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := json.Unmarshal(body, &rest); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// Which keys were present, so absent ones stay untouched. The bson and json
	// names of every Restaurant field are identical, so one lookup serves both.
	var present map[string]json.RawMessage
	if err := json.Unmarshal(body, &present); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	full, err := bson.Marshal(rest)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var doc bson.M
	if err := bson.Unmarshal(full, &doc); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// _id is never the client's to set: the profile is a singleton upserted by
	// an empty filter.
	delete(doc, "_id")
	set := bson.M{}
	for key, value := range doc {
		if _, sent := present[key]; sent {
			set[key] = value
		}
	}
	set["updatedAt"] = time.Now()
	if rest.Currency == "" {
		// Only as a default for a brand-new document; an existing one keeps its
		// currency because the key is simply not in `set`.
		if _, sent := present["currency"]; !sent {
			delete(set, "currency")
		} else {
			set["currency"] = "UZS"
		}
	}

	opts := options.Update().SetUpsert(true)
	if _, err := h.Store.Restaurant.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": set}, opts); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActSettingsUpdate, "settings", "", rest.Name, "")

	var saved models.Restaurant
	_ = h.Store.Restaurant.FindOne(r.Context(), bson.M{}).Decode(&saved)
	httpx.JSON(w, http.StatusOK, saved)
}

// ---- Categories CRUD ----

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var c models.Category
	if err := httpx.Decode(r, &c); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	c.ID = primitiveNil
	if c.BrandID.IsZero() {
		// New rows land in whichever brand the panel is looking at, so a
		// single-brand restaurant never has to think about it.
		if scope, err := h.adminScope(r); err == nil {
			c.BrandID = h.scopeBrand(r, scope)
		}
	}
	res, err := h.Store.Categories.InsertOne(r.Context(), c)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	c.ID = oidOf(res.InsertedID)
	h.logAction(r, ActCategoryCreate, "category", c.ID.Hex(), c.Name, "")
	httpx.JSON(w, http.StatusCreated, c)
}

func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var c models.Category
	if err := httpx.Decode(r, &c); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	c.ID = id
	c.BrandID = h.keepBrandID(r, h.Store.Categories, id, c.BrandID)
	if _, err := h.Store.Categories.ReplaceOne(r.Context(), bson.M{"_id": id}, c); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActCategoryUpdate, "category", id.Hex(), c.Name, "")
	httpx.JSON(w, http.StatusOK, c)
}

func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var removed models.Category
	_ = h.Store.Categories.FindOne(r.Context(), bson.M{"_id": id}).Decode(&removed)
	if _, err := h.Store.Categories.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Also remove menu items in this category.
	res, _ := h.Store.Menu.DeleteMany(r.Context(), bson.M{"categoryId": id})
	details := ""
	if res != nil && res.DeletedCount > 0 {
		details = strconv.FormatInt(res.DeletedCount, 10) + " ta taom bilan"
	}
	h.logAction(r, ActCategoryDelete, "category", id.Hex(), removed.Name, details)
	httpx.JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// AdminListCategories returns all categories of the selected brand (including
// inactive ones).
func (h *Handler) AdminListCategories(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	opts := options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}})
	cur, err := h.Store.Categories.Find(r.Context(), scope.brandFilter(bson.M{}), opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var cats []models.Category
	_ = cur.All(r.Context(), &cats)
	if cats == nil {
		cats = []models.Category{}
	}
	httpx.JSON(w, http.StatusOK, cats)
}

// ---- Menu CRUD ----

// AdminListMenu returns all menu items (including unavailable) sorted by
// sortOrder. The admin UI groups them by category client-side.
func (h *Handler) AdminListMenu(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := scope.brandFilter(bson.M{})
	if c := r.URL.Query().Get("categoryId"); c != "" {
		if id, err := objectID(c); err == nil {
			filter["categoryId"] = id
		}
	}
	opts := options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}})
	cur, err := h.Store.Menu.Find(r.Context(), filter, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var items []models.MenuItem
	_ = cur.All(r.Context(), &items)
	if items == nil {
		items = []models.MenuItem{}
	}
	// The admin list shows what each combo contains and what it saves, so the
	// owner can see at a glance whether a set is still priced sensibly.
	// The panel's lens is explicit: with a branch selected it is that branch's
	// stop list, and with none it is nobody's — the owner is looking at the
	// menu, not at one kitchen's evening.
	var soldOut soldOutLens
	if !scope.BranchID.IsZero() {
		if branch, err := h.branchByID(r, scope.BranchID); err == nil {
			soldOut = branch.IsSoldOut
		}
	}
	h.decorateCombos(r.Context(), items, soldOut)
	// ⚠️ The panel is the only reader that gets the cost — see menucost.go.
	httpx.JSON(w, http.StatusOK, h.pricedCards(r.Context(), withCosts(items)))
}

// normalizeIkpu cleans a state classifier code typed into the menu form.
//
// ⚠️ **Digits only, and anything else clears the field rather than being
// stored.** The code is copied off an accountant's spreadsheet, so it arrives
// with spaces, dashes and the occasional stray letter; it then goes onto a
// fiscal receipt, where a code that is nearly right is not better than a code
// that is absent — it is a receipt filed against the wrong product. Empty is a
// supported, ordinary state (see MenuItem.Ikpu), so falling back to it is safe
// in a way that guessing is not.
func normalizeIkpu(v string) string {
	out := codeDigits(v)
	// ИКПУ is 17 digits. A shorter string is a half-typed code and a longer one
	// is two codes run together; both would be filed against nothing.
	if len(out) != 17 {
		return ""
	}
	return out
}

// digitsOnly keeps the digits of a code typed into a form, drops the separators
// people add for readability, and gives back nothing at all when it finds a
// letter or a symbol — that is not a mistyped code, it is a different kind of
// value (a note, or the wrong spreadsheet column), and salvaging the digits out
// of it would file a real-looking number that nobody chose.
func codeDigits(v string) string {
	var b strings.Builder
	for _, c := range v {
		switch {
		case c >= '0' && c <= '9':
			b.WriteRune(c)
		case c == ' ' || c == '-' || c == '\t':
			// Separators people type for readability. Dropped, not refused.
		default:
			return ""
		}
	}
	return b.String()
}

// fiscalUnitCodes are the measure units the state classifier defines. Anything
// outside the list falls back to 0 (piece) rather than being stored: the number
// is chosen from a dropdown, so a value that is not on it arrived from an older
// client or a hand-written request, and a made-up unit puts a portion of soup
// on the receipt as metres.
var fiscalUnitCodes = map[int]bool{
	0:  true, // dona / штука
	10: true, // gramm
	11: true, // kilogramm
	22: true, // metr
	41: true, // litr
}

// normalizeFiscal cleans every field that ends up on a fiscal receipt.
//
// ⚠️ **One function, called on both create and update**, because the fields
// constrain each other and a rule split across four call sites is a rule that
// the fifth one will not have. Specifically: the package code names a packaging
// *of* the ИКПУ, so clearing the classifier code has to clear it too — pasted
// into two places by hand, that pairing is exactly what the next edit forgets.
func normalizeFiscal(m *models.MenuItem) {
	m.Ikpu = normalizeIkpu(m.Ikpu)

	// Digits only, on the same reasoning as the ИКПУ: it is copied off the
	// accountant's sheet and lands on a filed document. No length check — unlike
	// the ИКПУ's 17, the packaging code has no fixed width.
	m.PackageCode = codeDigits(m.PackageCode)
	if m.Ikpu == "" {
		// No classifier code, so there is nothing for a packaging to belong to.
		m.PackageCode = ""
	}

	// nil stays nil — that is "use the branch's rate", and it is the state
	// almost every dish is in. Only a value that was actually sent is clamped.
	if m.VatPercent != nil {
		v := *m.VatPercent
		if v < 0 || v > 100 {
			// Out of range means a typo (1200 for 12), and a typo here is a
			// wrong tax figure on a filed receipt. Drop back to the branch rate
			// rather than storing a number nobody meant.
			m.VatPercent = nil
		} else {
			m.VatPercent = &v
		}
	}

	if !fiscalUnitCodes[m.UnitCode] {
		m.UnitCode = 0
	}
}

func (h *Handler) CreateMenuItem(w http.ResponseWriter, r *http.Request) {
	var in menuItemIO
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	m := in.MenuItem
	m.Recipe = h.keepRecipe(r.Context(), primitiveNil, in.Recipe)
	m.Cost = h.keepCost(r.Context(), primitiveNil, in.Cost)
	m.ID = primitiveNil
	m.UpdatedAt = time.Now()
	normalizeFiscal(&m)
	// The dish has no id yet, so it cannot recommend itself here — the zero id
	// is passed for the duplicate and empty-id cleaning the same function does.
	m.RecommendedIDs = normalizeRecommended(primitiveNil, m.RecommendedIDs)
	if m.BrandID.IsZero() {
		if scope, err := h.adminScope(r); err == nil {
			m.BrandID = h.scopeBrand(r, scope)
		}
	}
	if err := h.validateCombo(r.Context(), &m); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.Store.Menu.InsertOne(r.Context(), m)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	m.ID = oidOf(res.InsertedID)
	h.logAction(r, ActMenuCreate, "menu", m.ID.Hex(), m.Name, "")
	httpx.JSON(w, http.StatusCreated,
		h.pricedCards(r.Context(), []menuItemIO{withCost(m)})[0])
}

func (h *Handler) UpdateMenuItem(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in menuItemIO
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	m := in.MenuItem
	// ⚠️ Kept when the form did not send it — and the dish form never sends it
	// any more: the card is written on its own screen. See keepRecipe.
	m.Recipe = h.keepRecipe(r.Context(), id, in.Recipe)
	// ⚠️ Kept when the form did not send it: this is a whole-document replace,
	// so an older tab saving a dish's name would otherwise erase its cost.
	m.Cost = h.keepCost(r.Context(), id, in.Cost)
	m.ID = id
	m.UpdatedAt = time.Now()
	normalizeFiscal(&m)
	m.RecommendedIDs = normalizeRecommended(id, m.RecommendedIDs)
	m.BrandID = h.keepBrandID(r, h.Store.Menu, id, m.BrandID)
	if err := h.validateCombo(r.Context(), &m); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// ⚠️ **Read before the replace, because after it there is nothing to
	// compare against.** This is a whole-document write, so the previous card
	// exists only in the moment between these two lines — and a tech card is
	// the one thing in this product that can be edited to make a theft
	// arithmetically invisible.
	changes := h.recipeDiff(r.Context(), id, m.Recipe)
	if _, err := h.Store.Menu.ReplaceOne(r.Context(), bson.M{"_id": id}, m); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ The journal used to say `menu.update · Lag'mon` and stop. True, filed,
	// and answering nothing: which ingredient, and from what to what, is the
	// entire difference between a record and a receipt for having one.
	h.logAction(r, ActMenuUpdate, "menu", id.Hex(), m.Name, describeRecipeDiff(changes))
	h.alertOnRecipeIncrease(r, m, changes)
	httpx.JSON(w, http.StatusOK,
		h.pricedCards(r.Context(), []menuItemIO{withCost(m)})[0])
}

func (h *Handler) DeleteMenuItem(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var removed models.MenuItem
	_ = h.Store.Menu.FindOne(r.Context(), bson.M{"_id": id}).Decode(&removed)

	// A dish inside a combo cannot just disappear: the set would stay on the
	// menu missing a course, and the guest would find out at checkout. Name the
	// combos so the admin knows exactly what to fix.
	cur, err := h.Store.Menu.Find(r.Context(), bson.M{"comboItems.menuItemId": id})
	if err == nil {
		var combos []models.MenuItem
		if err := cur.All(r.Context(), &combos); err == nil && len(combos) > 0 {
			names := make([]string, 0, len(combos))
			for _, c := range combos {
				names = append(names, c.Name)
			}
			httpx.Error(w, http.StatusConflict,
				"bu taom to'plam(lar)da ishlatilgan: "+strings.Join(names, ", ")+
					" — avval o'sha to'plamlardan olib tashlang")
			return
		}
	}

	if _, err := h.Store.Menu.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActMenuDelete, "menu", id.Hex(), removed.Name, "")

	// ⚠️ **Deleting a dish is the commonest way an imported photograph becomes
	// rubbish**, and hooking only the importer would mean a restaurant that
	// imported once and then tidied its menu keeps every deleted dish's picture
	// for ever — the sweep would never run again. Detached and silent: the
	// dish is already gone and the owner is not waiting for housekeeping.
	go func() {
		ctx, cancel := context.WithTimeout(
			context.WithoutCancel(r.Context()), 2*time.Minute)
		defer cancel()
		_, _ = h.sweepImportAssets(ctx)
	}()

	httpx.JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// ---- Orders ----

// AdminListOrders returns orders, filtered by ?status=, ?userId= and a free
// text ?q= over the order number, database id, customer name, phone, address
// and courier.
//
// ⚠️ **Checks rung up on our own till never appear here at all.** Not hidden
// behind a filter — absent. This board is the delivery desk: the person reading
// it is taking calls and watching couriers, and a table's running tab is not
// something they can act on, accept, assign or dispatch. Offering it as a
// toggle was still offering it, and a control that only ever adds noise is a
// control somebody eventually leaves switched on.
//
// The dining room has its own screens, which are better at this: the till for
// what is open now, /admin/cash for the money, and the reports for the history.
//
// ⚠️ The distinguishing field is **`check`, never `type == "dinein"`**. A guest
// scanning the table QR and ordering from their own phone also produces a
// `dinein` order — and that one *must* appear here, because somebody has to
// accept it. Splitting on the order type instead would make QR orders vanish
// silently, which is the one failure nobody would notice until a guest
// complained.
//
// Statistics, reports and revenue are untouched: there a till sale counts, and
// always did.
func (h *Handler) AdminListOrders(w http.ResponseWriter, r *http.Request) {
	filter, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter["check"] = bson.M{"$exists": false}
	if s := r.URL.Query().Get("status"); s != "" {
		filter["status"] = s
	}
	if uid := r.URL.Query().Get("userId"); uid != "" {
		id, err := objectID(uid)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid userId")
			return
		}
		filter["userId"] = id
	}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		// Operators paste the number straight off the receipt, often with the
		// leading "#" the UI shows.
		q = strings.TrimPrefix(q, "#")
		// The raw input goes into a regex, so metacharacters have to be
		// neutralised — otherwise "(" simply returns nothing.
		rx := bson.M{"$regex": regexp.QuoteMeta(q), "$options": "i"}
		or := []bson.M{
			{"number": rx}, {"customer.name": rx}, {"customer.phone": rx},
			{"address.text": rx}, {"courierName": rx},
		}
		// The receipt also shows the database id; searching by it must work.
		if id, err := objectID(q); err == nil {
			or = append(or, bson.M{"_id": id})
		}
		filter["$or"] = or
	}
	limit := int64(200)
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 && v <= 500 {
		limit = int64(v)
	}
	// Newest first — except for pre-orders, where the question is the opposite
	// one. "What came in last?" is what an orders board is for; "what is due
	// next?" is what a pre-order list is for, and a list of Saturday's parties
	// sorted by when somebody happened to ring is not a plan anybody can work
	// from. Cancelled ones are left out for the same reason: the tab is a
	// timetable, not a history.
	sort := bson.D{{Key: "createdAt", Value: -1}}
	if r.URL.Query().Get("scheduled") == "1" {
		filter["scheduledAt"] = bson.M{"$ne": nil}
		if _, ok := filter["status"]; !ok {
			filter["status"] = bson.M{"$ne": string(models.StatusCancelled)}
		}
		sort = bson.D{{Key: "scheduledAt", Value: 1}}
	}
	opts := options.Find().SetSort(sort).SetLimit(limit)
	cur, err := h.Store.Orders.Find(r.Context(), filter, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var orders []models.Order
	_ = cur.All(r.Context(), &orders)
	if orders == nil {
		orders = []models.Order{}
	}
	httpx.JSON(w, http.StatusOK, orders)
}

// AdminGetOrder returns one order with everything the panel shows (receipt).
func (h *Handler) AdminGetOrder(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	filter, err := h.scopedOrderFilter(r, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var order models.Order
	if err := h.Store.Orders.FindOne(r.Context(), filter).Decode(&order); err != nil {
		httpx.Error(w, http.StatusNotFound, "order not found")
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}

type statusRequest struct {
	Status models.OrderStatus `json:"status" validate:"required"`
	// Why the order is being cancelled — required by the panel, shown to the
	// customer on the tracking page. Ignored for every other status.
	Reason string `json:"reason"`
}

func (h *Handler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req statusRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// Scoped: a manager cannot cancel or re-stage another branch's order by
	// pasting its id. The branch is in the filter, so the UpdateOne below
	// matches nothing when the order is out of scope.
	filter, err := h.scopedOrderFilter(r, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	set := bson.M{"status": req.Status, "updatedAt": now}
	update := bson.M{
		"$push": bson.M{"statusHistory": models.StatusEvent{Status: req.Status, At: now}},
	}
	unset := bson.M{}
	if req.Status == models.StatusCancelled {
		set["cancelReason"] = clampText(req.Reason, 300)
	} else {
		// Reinstating an order must not leave the old reason on it.
		unset["cancelReason"] = ""
	}
	// A ticket pushed back to the kitchen is not ready any more.
	//
	// `readyAt` is the kitchen's own "done" mark (see models.Order.ReadyAt), and
	// an order moved back to pending, confirmed or preparing is one somebody
	// decided was **not** finished. Left in place, the flag would say "ready" on
	// a dish nobody has cooked yet, and the kitchen screen — which hides ready
	// tickets — would never show it again.
	switch req.Status {
	case models.StatusPending, models.StatusConfirmed, models.StatusPreparing:
		unset["readyAt"] = ""
	}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	update["$set"] = set
	res, err := h.Store.Orders.UpdateOne(r.Context(), filter, update)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Nothing matched: either no such order, or one this admin may not touch.
	// Same answer for both — a manager should not learn another branch's order
	// exists by watching this endpoint's replies.
	if res.MatchedCount == 0 {
		httpx.Error(w, http.StatusNotFound, "order not found")
		return
	}
	var order models.Order
	_ = h.Store.Orders.FindOne(r.Context(), filter).Decode(&order)

	// Loyalty follows the order's fate. Cashback is paid on delivery, not on
	// placement — an order that never arrives must not mint points; and a
	// cancellation gives back what was spent and takes back what was earned.
	// Both are idempotent, so flipping a status twice changes nothing.
	switch req.Status {
	case models.StatusDelivered:
		h.awardPoints(r.Context(), &order)
	case models.StatusCancelled:
		h.revokePoints(r.Context(), &order)
	case models.StatusConfirmed:
		// Confirming is what puts an order in front of the kitchen, so it is
		// where it goes to the till. Deliberately not on creation: a till is
		// somebody's accounting, and a mistyped or fraudulent order that
		// reaches it has to be voided at the register by hand.
		//
		// Never blocks the confirmation. A till being down is the restaurant's
		// problem to see on the receipt, not a reason an operator cannot move
		// an order along.
		h.autoSendToPOS(r.Context(), &order)
	}

	// The guest hears about it through the bot, when they came in through it.
	// Free, unlike the SMS this replaces — see handlers/notify.go — and never
	// able to fail the status change.
	h.notifyOrderStatus(r.Context(), &order)

	if req.Status == models.StatusCancelled {
		h.logAction(r, ActOrderCancel, "order", id.Hex(), "#"+order.Number,
			order.CancelReason)
	} else {
		h.logAction(r, ActOrderStatus, "order", id.Hex(), "#"+order.Number,
			string(req.Status))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"status": req.Status, "at": now, "cancelReason": set["cancelReason"],
	})
}

type orderAddressRequest struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Text    string  `json:"text"`
	Comment string  `json:"comment"`
}

// AdminUpdateOrderAddress moves the delivery point of an existing order.
//
// Customers drop the pin themselves and sometimes drop it on the wrong building
// — and the pin is not cosmetic: the courier app refuses "delivered" until the
// courier is within `arrivalRadiusM` of it. So the panel has to be able to
// correct it after the fact.
//
// The money is deliberately left alone: the customer already agreed a total, and
// silently repricing a confirmed order behind their back is worse than a fee
// that no longer matches the zone. The zone and distance are recomputed (they
// are descriptive), and the recomputed fee is returned so the panel can tell the
// operator when the correction crossed into a differently priced zone.
func (h *Handler) AdminUpdateOrderAddress(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req orderAddressRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Lat < -90 || req.Lat > 90 || req.Lng < -180 || req.Lng > 180 ||
		(req.Lat == 0 && req.Lng == 0) {
		httpx.Error(w, http.StatusBadRequest, "xaritada joyni belgilang")
		return
	}

	filter, err := h.scopedOrderFilter(r, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var order models.Order
	if err := h.Store.Orders.FindOne(r.Context(), filter).Decode(&order); err != nil {
		httpx.Error(w, http.StatusNotFound, "buyurtma topilmadi")
		return
	}
	if order.Type != "delivery" {
		httpx.Error(w, http.StatusBadRequest, "olib ketish buyurtmasida manzil yo'q")
		return
	}

	text := clampText(req.Text, 300)
	if text == "" {
		text = order.Address.Text
	}
	address := models.OrderAddress{
		Text:    text,
		Lat:     req.Lat,
		Lng:     req.Lng,
		Comment: clampText(req.Comment, 300),
	}

	// Descriptive fields only — see the note above on leaving the fee alone.
	// Repriced against the branch that took the order, not whichever branch is
	// nearest today: moving a pin must not silently hand the order to another
	// kitchen that never agreed to cook it.
	zone, km := order.DeliveryZone, order.DistanceKm
	fee, available := order.DeliveryFee, true
	if branch, err := h.branchByID(r, order.BranchID); err == nil {
		q := quoteDeliveryFrom(branch.Delivery, branch.Address, req.Lat, req.Lng, order.Subtotal)
		fee, zone, km, available = q.Fee, q.Zone, q.DistanceKm, q.Available
	}

	if _, err := h.Store.Orders.UpdateByID(r.Context(), id, bson.M{
		"$set": bson.M{
			"address":      address,
			"deliveryZone": zone,
			"distanceKm":   km,
			"updatedAt":    time.Now(),
		},
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.logAction(r, ActOrderAddress, "order", id.Hex(), "#"+order.Number,
		"manzil tuzatildi: "+address.Text)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"address":      address,
		"deliveryZone": zone,
		"distanceKm":   km,
		// What this point would cost today, next to the fee the order carries.
		"quotedFee":   fee,
		"available":   available,
		"deliveryFee": order.DeliveryFee,
	})
}
