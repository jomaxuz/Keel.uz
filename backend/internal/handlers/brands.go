package handlers

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

// Brands and branches.
//
// A brand is a menu with a face (the restaurant, the samsa chain); a branch is
// a place that cooks it. Both always exist — a single restaurant simply has one
// of each and never sees the difference.

func (h *Handler) listBrands(r *http.Request, onlyActive bool) ([]models.Brand, error) {
	filter := bson.M{}
	if onlyActive {
		filter["isActive"] = true
	}
	opts := options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: 1}})
	cur, err := h.Store.Brands.Find(r.Context(), filter, opts)
	if err != nil {
		return nil, err
	}
	out := []models.Brand{}
	if err := cur.All(r.Context(), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// branchSort is the order branches are always listed in: the owner's own
// ordering first, then oldest, so a list never reshuffles itself between loads.
func branchSort() *options.FindOptions {
	return options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: 1}})
}

func (h *Handler) listBranches(r *http.Request, brandID string, onlyActive bool) ([]models.Branch, error) {
	filter := bson.M{}
	if brandID != "" {
		id, err := objectID(brandID)
		if err != nil {
			return nil, err
		}
		filter["brandId"] = id
	}
	if onlyActive {
		filter["isActive"] = true
	}
	opts := options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: 1}})
	cur, err := h.Store.Branches.Find(r.Context(), filter, opts)
	if err != nil {
		return nil, err
	}
	out := []models.Branch{}
	if err := cur.All(r.Context(), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ---- Public ----

// GetBrands is what the site opens with: the brands on offer and, for each, the
// branches a guest can be served from.
func (h *Handler) GetBrands(w http.ResponseWriter, r *http.Request) {
	brands, err := h.listBrands(r, true)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	branches, err := h.listBranches(r, "", true)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"brands":   brands,
		"branches": branches,
	})
}

// ---- Admin: brands ----

func (h *Handler) AdminListBrands(w http.ResponseWriter, r *http.Request) {
	brands, err := h.listBrands(r, false)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, brands)
}

func (h *Handler) AdminCreateBrand(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var brand models.Brand
	if err := httpx.Decode(r, &brand); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(brand.Name) == "" {
		httpx.Error(w, http.StatusBadRequest, "brend nomini yozing")
		return
	}
	// ⚠️ **The template is applied here and nowhere else.** What a brand offers
	// is decided once, when it is created; from that moment the panel's own
	// switches are the truth. A type re-read on every request would be a mode —
	// and a mode is what refuses a flower shop that starts selling coffee.
	//
	// ⚠️ Only when the caller sent nothing: a console or a panel that filled the
	// switches in deliberately must not have them overwritten by a default.
	if brand.Features == (models.BrandFeatures{}) {
		brand.Features = brand.BusinessType.Defaults()
	}
	now := time.Now()
	brand.ID = primitiveNil
	brand.CreatedAt, brand.UpdatedAt = now, now
	res, err := h.Store.Brands.InsertOne(r.Context(), brand)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	brand.ID = oidOf(res.InsertedID)
	h.logAction(r, ActBrandCreate, "brand", brand.ID.Hex(), brand.Name, "")
	httpx.JSON(w, http.StatusCreated, brand)
}

func (h *Handler) AdminUpdateBrand(w http.ResponseWriter, r *http.Request) {
	// The brand is the company's face — its name, its logo, its colours. A
	// branch manager runs a kitchen; they do not rebrand the company.
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var brand models.Brand
	if err := httpx.Decode(r, &brand); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	brand.ID = primitiveNil
	brand.UpdatedAt = time.Now()
	if _, err := h.Store.Brands.UpdateByID(r.Context(), id, bson.M{"$set": brand}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActBrandUpdate, "brand", id.Hex(), brand.Name, "")
	brand.ID = id
	httpx.JSON(w, http.StatusOK, brand)
}

// AdminDeleteBrand refuses while the brand still has branches: deleting it
// would orphan their orders, and an orphaned order is unanswerable when the
// customer rings about it.
func (h *Handler) AdminDeleteBrand(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	total, err := h.Store.Brands.CountDocuments(r.Context(), bson.M{})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if total <= 1 {
		httpx.Error(w, http.StatusBadRequest, "oxirgi brendni o'chirib bo'lmaydi")
		return
	}
	branches, err := h.Store.Branches.CountDocuments(r.Context(), bson.M{"brandId": id})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if branches > 0 {
		httpx.Error(w, http.StatusBadRequest, "avval shu brendning filiallarini o'chiring")
		return
	}
	var brand models.Brand
	_ = h.Store.Brands.FindOne(r.Context(), bson.M{"_id": id}).Decode(&brand)
	if _, err := h.Store.Brands.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActBrandDelete, "brand", id.Hex(), brand.Name, "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Admin: branches ----

func (h *Handler) AdminListBranches(w http.ResponseWriter, r *http.Request) {
	branches, err := h.listBranches(r, r.URL.Query().Get("brandId"), false)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// ⚠️ **Empty arrays, never nulls.** The panel takes the plan straight from
	// here and hands it to the editors, and a branch whose room was never
	// drawn — or never split into zones — arrived as `null`. The settings page
	// did not degrade, it *threw*: the whole screen became "something went
	// wrong in the kitchen", with nothing naming the section or the field.
	for i := range branches {
		branches[i].Booking = bookingSlices(branches[i].Booking)
	}
	httpx.JSON(w, http.StatusOK, branches)
}

func (h *Handler) AdminCreateBranch(w http.ResponseWriter, r *http.Request) {
	// Opening a branch is the company's decision, not a manager's.
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var branch models.Branch
	if err := httpx.Decode(r, &branch); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(branch.Name) == "" {
		httpx.Error(w, http.StatusBadRequest, "filial nomini yozing")
		return
	}
	if branch.BrandID.IsZero() {
		httpx.Error(w, http.StatusBadRequest, "brendni tanlang")
		return
	}
	now := time.Now()
	branch.ID = primitiveNil
	branch.CreatedAt, branch.UpdatedAt = now, now
	if branch.PrepMinutes <= 0 {
		branch.PrepMinutes = 30
	}
	// Attendance geofence. Wider than the door on purpose: a phone indoors is
	// lucky to be accurate to 20 m, and a radius tighter than the error keeps
	// honest staff standing outside pressing a button that will not work.
	if branch.StaffRadiusM <= 0 {
		branch.StaffRadiusM = 50
	}
	// Pre-orders start off — the owner switches them on per kitchen — but the
	// numbers around the switch are filled in, so the section is usable the
	// moment it is opened rather than showing four zeros that would refuse
	// every order the guest tried to place.
	branch.Preorder = clampPreorder(branch.Preorder)
	// ⚠️ **The plan's colours are checked here, not trusted from the form.**
	// They are rendered straight into an SVG `fill` on the public booking page,
	// so an arbitrary string is a way to put something that is not a colour
	// onto every guest's screen. Anything unrecognised falls back to the
	// default rather than being refused: a colour is decoration, and rejecting
	// a whole floor plan over one is a save the owner cannot complete.
	branch.Booking = clampFloorColors(branch.Booking)
	// Empty arrays, not nil: a nil slice is stored as `null`, and the counter's
	// first "we're out of samsa" is an $addToSet, which refuses a non-array.
	// A new branch is exactly the one that has never had anything run out.
	if branch.SoldOut == nil {
		branch.SoldOut = []primitive.ObjectID{}
	}
	if branch.POSSoldOut == nil {
		branch.POSSoldOut = []primitive.ObjectID{}
	}
	res, err := h.Store.Branches.InsertOne(r.Context(), branch)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	branch.ID = oidOf(res.InsertedID)
	h.logAction(r, ActBranchCreate, "branch", branch.ID.Hex(), branch.Name, "")
	httpx.JSON(w, http.StatusCreated, branch)
}

func (h *Handler) AdminUpdateBranch(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	// A manager may set their own branch's hours, zones and floor plan — that is
	// the job — but not another branch's.
	if err := h.requireBranchAccess(r, id); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var branch models.Branch
	if err := httpx.Decode(r, &branch); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	branch.ID = primitiveNil
	branch.UpdatedAt = time.Now()
	// ⚠️ **A branch never supplies itself.** Read literally that would build a
	// dispatch whose two ends are the same shelf — the same kilo subtracted and
	// added, and a slip nobody can accept. Corrected rather than refused: the
	// value means "my own store room", which is what an empty field already
	// means, so there is nothing to tell anybody about.
	if branch.SupplyBranchID == id {
		branch.SupplyBranchID = primitiveNil
	}
	// Narrowed on save rather than on read, so an owner who typed 5000 minutes
	// sees the number that was actually kept — a value silently corrected on
	// every read is one they would keep re-typing.
	branch.Preorder = clampPreorder(branch.Preorder)

	// The sold-out list is deliberately **not** written here. It is edited one
	// tap at a time at the counter (AdminSetSoldOut) while this form may have
	// been open for an hour; saving the form's stale copy would put back the
	// dishes the kitchen has since run out of.
	full, err := bson.Marshal(branch)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var set bson.M
	if err := bson.Unmarshal(full, &set); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	delete(set, "soldOut")
	// The till's own stop list and the record of when it was last read belong to
	// the sync (handlers/posstop.go). Written from here they would be zeroed by
	// any settings save, putting dishes the kitchen system has stopped back on
	// the site until the next poll.
	delete(set, "posSoldOut")
	delete(set, "posSoldOutAt")
	delete(set, "posSoldOutError")
	delete(set, "_id")
	// ⚠️ **Written explicitly, because `omitempty` cannot say "cleared".** An
	// owner who points a branch back at its own store room sends an empty id,
	// which the marshaller drops — and the branch would go on shipping vans from
	// a central store it no longer uses, with the form showing the setting they
	// thought they had removed.
	set["supplyBranchId"] = branch.SupplyBranchID
	// Same trap as soldOut, one level nastier: the settings form does not know
	// about the kiosk key or its revocation counter, so saving the form would
	// write zeros over both — silently killing the branch screen's token and
	// every code it was showing. They are managed by their own endpoint.
	delete(set, "kioskSecret")
	delete(set, "kioskVersion")
	// The third stop list, and the third time this trap has been worth writing
	// down: the settings form does not know what the shelves hold, so saving it
	// would put back every dish the stock sync has stopped — until the next run
	// took them off again, which looks exactly like the toggle not working. The
	// switch itself (`stockStop`) is the owner's and is written from here; the
	// list it produces is not.
	delete(set, "stockSoldOut")
	delete(set, "stockSoldOutAt")

	if _, err := h.Store.Branches.UpdateByID(r.Context(), id, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActBranchUpdate, "branch", id.Hex(), branch.Name, "")
	branch.ID = id
	httpx.JSON(w, http.StatusOK, branch)
}

type soldOutRequest struct {
	MenuItemID string `json:"menuItemId"`
	SoldOut    bool   `json:"soldOut"`
	// How long the stop should hold. ⚠️ **A duration, never a moment** — the
	// server owns the clock here, see `stopUntil`. Zero means "until somebody
	// says otherwise", which is what the button did before this existed and
	// stays the default.
	Minutes int `json:"minutes,omitempty"`
	// Hold it until this branch shuts today.
	//
	// ⚠️ Its own field rather than a magic value in `Minutes`, because it is a
	// different question: "two hours" is arithmetic and "until we close" is a
	// fact about this branch's schedule, which the screen does not have.
	UntilClose bool `json:"untilClose,omitempty"`
}

// AdminSetSoldOut marks one dish as run out (or back on) at one branch.
//
// A separate endpoint rather than a field on the branch form on purpose: this
// is pressed mid-service, by whoever is at the counter, one tap at a time. It
// must not require loading and re-saving the whole branch — two people doing
// that at once would each overwrite the other's list.
func (h *Handler) AdminSetSoldOut(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req soldOutRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	itemID, err := objectID(req.MenuItemID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "taom noto'g'ri")
		return
	}
	if err := h.requireBranchAccess(r, id); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}

	// A dish another list is holding cannot be put back from here, and saying
	// "ok" would be a lie with a three-minute fuse: the next sync would stop it
	// again, and the counter would conclude the button does not work. Refused in
	// words, naming where the switch actually is.
	//
	// ⚠️ The rule lives in soldOutHeldBy (tillstop.go) because the till offers
	// the same button on the same list. Two copies would eventually disagree,
	// and a restaurant where the panel refuses what the counter allows has no
	// way to tell which screen is broken.
	// ⚠️ Read once and used twice: the refusal above and the closing time below
	// are both facts about this branch, and two reads would be two chances to
	// answer from different documents.
	var b models.Branch
	_ = h.Store.Branches.FindOne(r.Context(), bson.M{"_id": id}).Decode(&b)
	if !req.SoldOut {
		if reason := soldOutHeldBy(b, itemID); reason != "" {
			httpx.Error(w, http.StatusConflict, reason)
			return
		}
	}

	until := stopUntil(b, req.Minutes, req.UntilClose, time.Now())
	if err := h.setBranchSoldOut(r, id, itemID, req.SoldOut, until); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	var item models.MenuItem
	_ = h.Store.Menu.FindOne(r.Context(), bson.M{"_id": itemID}).Decode(&item)
	var branch models.Branch
	_ = h.Store.Branches.FindOne(r.Context(), bson.M{"_id": id}).Decode(&branch)
	details := "qaytadan bor"
	if req.SoldOut {
		details = "tugadi"
	}
	h.logAction(r, ActBranchUpdate, "branch", id.Hex(), branch.Name,
		item.Name+" — "+details)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "soldOut": branch.SoldOut})
}

// AdminDeleteBranch keeps the last branch and refuses to strand orders: a
// branch that has ever taken one is deactivated instead of removed.
func (h *Handler) AdminDeleteBranch(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	total, err := h.Store.Branches.CountDocuments(r.Context(), bson.M{})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if total <= 1 {
		httpx.Error(w, http.StatusBadRequest, "oxirgi filialni o'chirib bo'lmaydi")
		return
	}
	orders, err := h.Store.Orders.CountDocuments(r.Context(), bson.M{"branchId": id})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var branch models.Branch
	_ = h.Store.Branches.FindOne(r.Context(), bson.M{"_id": id}).Decode(&branch)

	if orders > 0 {
		if _, err := h.Store.Branches.UpdateByID(r.Context(), id,
			bson.M{"$set": bson.M{"isActive": false, "updatedAt": time.Now()}},
		); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		h.logAction(r, ActBranchUpdate, "branch", id.Hex(), branch.Name, "filial o'chirildi (buyurtmalari bor)")
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "deactivated": true})
		return
	}

	if _, err := h.Store.Branches.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActBranchDelete, "branch", id.Hex(), branch.Name, "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// clampFloorColors keeps only colours the renderers know — see models.FloorColor.
func clampFloorColors(b models.BookingSettings) models.BookingSettings {
	for i := range b.Shapes {
		b.Shapes[i].Color = models.FloorColor(b.Shapes[i].Color)
	}
	for i := range b.Tables {
		b.Tables[i].Color = models.FloorColor(b.Tables[i].Color)
	}
	return b
}
