package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The brand/branch lens.
//
// Every list in the panel and every read on the site is answered through a
// Scope. A zero id means "no filter", which is exactly what a single-brand,
// single-branch install produces — so the queries below are byte-for-byte what
// they were before brands existed, and that customer sees nothing new.
//
// The one rule worth stating: a manager tied to a branch can never widen their
// own scope. The query string is a request, not an authority — clampToAdmin
// has the last word.
type Scope struct {
	BrandID  primitive.ObjectID
	BranchID primitive.ObjectID
}

// brandFilter narrows a query to one brand's menu.
func (s Scope) brandFilter(f bson.M) bson.M {
	if f == nil {
		f = bson.M{}
	}
	if !s.BrandID.IsZero() {
		f["brandId"] = s.BrandID
	}
	return f
}

// branchFilter narrows a query to one branch. When only a brand is chosen the
// caller passes the brand's branch ids in `fallback`, because orders, couriers
// and bookings carry a branch but no brand of their own.
func (s Scope) branchFilter(f bson.M, fallback []primitive.ObjectID) bson.M {
	if f == nil {
		f = bson.M{}
	}
	switch {
	case !s.BranchID.IsZero():
		f["branchId"] = s.BranchID
	case fallback != nil:
		f["branchId"] = bson.M{"$in": fallback}
	}
	return f
}

// adminScope reads the lens the panel asked for and clamps it to what the
// signed-in admin is allowed to see.
func (h *Handler) adminScope(r *http.Request) (Scope, error) {
	var s Scope
	q := r.URL.Query()
	if v := strings.TrimSpace(q.Get("brandId")); v != "" {
		id, err := objectID(v)
		if err != nil {
			return s, errors.New("invalid brandId")
		}
		s.BrandID = id
	}
	if v := strings.TrimSpace(q.Get("branchId")); v != "" {
		id, err := objectID(v)
		if err != nil {
			return s, errors.New("invalid branchId")
		}
		s.BranchID = id
	}
	return h.clampToAdmin(r, s)
}

// clampToAdmin pins the scope to the admin's own branch when they have one. An
// owner has no branch and keeps whatever they asked for.
func (h *Handler) clampToAdmin(r *http.Request, s Scope) (Scope, error) {
	claims := middleware.ClaimsFrom(r.Context())
	if claims == nil || claims.UserID == "" {
		return s, nil
	}
	id, err := objectID(claims.UserID)
	if err != nil {
		return s, nil
	}
	var admin models.AdminUser
	if err := h.Store.Admins.FindOne(r.Context(), bson.M{"_id": id}).Decode(&admin); err != nil {
		return s, nil
	}
	if admin.BranchID.IsZero() {
		return s, nil
	}
	// A branch manager sees their branch and nothing else, whatever the URL says.
	s.BranchID = admin.BranchID
	var branch models.Branch
	if err := h.Store.Branches.FindOne(r.Context(),
		bson.M{"_id": admin.BranchID}).Decode(&branch); err == nil {
		s.BrandID = branch.BrandID
	}
	return s, nil
}

// scopeBrand is the brand a newly created row belongs to: the one the panel is
// looking at, or the install's only brand when no lens is set.
func (h *Handler) scopeBrand(r *http.Request, s Scope) primitive.ObjectID {
	if !s.BrandID.IsZero() {
		return s.BrandID
	}
	brands, err := h.listBrands(r, false)
	if err != nil || len(brands) == 0 {
		return primitive.NilObjectID
	}
	return brands[0].ID
}

// scopeBranch is the branch a newly created row belongs to: the one the panel
// is looking at, or the brand's first branch when the lens is only on a brand.
// A single-branch install lands on its only branch either way.
func (h *Handler) scopeBranch(r *http.Request, s Scope) primitive.ObjectID {
	if !s.BranchID.IsZero() {
		return s.BranchID
	}
	if b, err := h.defaultBranch(r, s.BrandID); err == nil {
		return b.ID
	}
	return primitive.NilObjectID
}

// keepBrandID preserves the brand of a document being replaced wholesale. The
// panel PUTs the full object back, and an older client that does not know about
// brands would otherwise silently orphan the dish it just renamed.
func (h *Handler) keepBrandID(r *http.Request, coll interface {
	FindOne(ctx context.Context, filter any, opts ...*options.FindOneOptions) *mongo.SingleResult
}, id primitive.ObjectID, incoming primitive.ObjectID) primitive.ObjectID {
	if !incoming.IsZero() {
		return incoming
	}
	var row struct {
		BrandID primitive.ObjectID `bson:"brandId"`
	}
	if err := coll.FindOne(r.Context(), bson.M{"_id": id}).Decode(&row); err == nil {
		return row.BrandID
	}
	return incoming
}

// ---- Who may touch what ----

// adminUser loads the signed-in panel account.
func (h *Handler) adminUser(r *http.Request) (*models.AdminUser, error) {
	claims := middleware.ClaimsFrom(r.Context())
	if claims == nil || claims.UserID == "" {
		return nil, errors.New("forbidden")
	}
	id, err := objectID(claims.UserID)
	if err != nil {
		return nil, errors.New("forbidden")
	}
	var admin models.AdminUser
	if err := h.Store.Admins.FindOne(r.Context(), bson.M{"_id": id}).Decode(&admin); err != nil {
		return nil, errors.New("forbidden")
	}
	return &admin, nil
}

// requireBranchAccess allows an owner anything and a branch manager only their
// own branch. The panel already hides what a manager cannot do, but hiding a
// button is not a rule — this is.
func (h *Handler) requireBranchAccess(r *http.Request, branchID primitive.ObjectID) error {
	admin, err := h.adminUser(r)
	if err != nil {
		return err
	}
	if admin.BranchID.IsZero() {
		return nil // an owner, or a manager who was never pinned to a branch
	}
	if admin.BranchID != branchID {
		return errors.New("bu filial sizga biriktirilmagan")
	}
	return nil
}

// requireOwner guards what belongs to the company rather than to a branch: the
// company profile, the brands, and creating or removing branches. A branch
// manager runs their own kitchen; they do not rename the company.
func (h *Handler) requireOwner(r *http.Request) error {
	claims := middleware.ClaimsFrom(r.Context())
	if claims == nil || claims.Role != "owner" {
		return errors.New("bu amal faqat egasi uchun")
	}
	return nil
}

// branchIDsOf lists the branches of a brand — the fallback branchFilter needs
// when the panel is looking at a whole brand. Returns nil when no brand is
// selected, which reads as "every branch".
func (h *Handler) branchIDsOf(r *http.Request, brandID primitive.ObjectID) ([]primitive.ObjectID, error) {
	if brandID.IsZero() {
		return nil, nil
	}
	cur, err := h.Store.Branches.Find(r.Context(), bson.M{"brandId": brandID})
	if err != nil {
		return nil, err
	}
	var rows []models.Branch
	if err := cur.All(r.Context(), &rows); err != nil {
		return nil, err
	}
	// An empty slice, not nil: a brand with no branches must match no orders
	// rather than silently matching all of them.
	ids := make([]primitive.ObjectID, 0, len(rows))
	for _, b := range rows {
		ids = append(ids, b.ID)
	}
	return ids, nil
}

// orderScope is the filter admin order/courier/booking lists start from.
func (h *Handler) orderScope(r *http.Request) (bson.M, Scope, error) {
	s, err := h.adminScope(r)
	if err != nil {
		return nil, s, err
	}
	var fallback []primitive.ObjectID
	if s.BranchID.IsZero() {
		fallback, err = h.branchIDsOf(r, s.BrandID)
		if err != nil {
			return nil, s, err
		}
	}
	return s.branchFilter(bson.M{}, fallback), s, nil
}

// ---- Public: which brand is the guest looking at ----

// publicBrand resolves ?brand= (an id or a slug) to a brand. With nothing asked
// for it returns the first active brand, which is the only one a single-brand
// restaurant has. A missing brand collection (a very old install that has not
// booted the migration yet) yields a zero brand and no filtering.
func (h *Handler) publicBrand(r *http.Request) (*models.Brand, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("brand"))
	if raw != "" {
		filter := bson.M{"slug": raw}
		if id, err := objectID(raw); err == nil {
			filter = bson.M{"_id": id}
		}
		var b models.Brand
		if err := h.Store.Brands.FindOne(r.Context(), filter).Decode(&b); err != nil {
			return nil, errors.New("brend topilmadi")
		}
		return &b, nil
	}
	brands, err := h.listBrands(r, true)
	if err != nil || len(brands) == 0 {
		return nil, nil
	}
	return &brands[0], nil
}

// publicScope is publicBrand as a Scope, for the read paths that only need to
// filter a menu.
func (h *Handler) publicScope(r *http.Request) (Scope, *models.Brand, error) {
	brand, err := h.publicBrand(r)
	if err != nil {
		return Scope{}, nil, err
	}
	if brand == nil {
		return Scope{}, nil, nil
	}
	return Scope{BrandID: brand.ID}, brand, nil
}

// ---- Branch resolution ----

// branchByID loads one branch.
func (h *Handler) branchByID(r *http.Request, id primitive.ObjectID) (*models.Branch, error) {
	if id.IsZero() {
		return nil, errors.New("filial tanlanmagan")
	}
	var b models.Branch
	if err := h.Store.Branches.FindOne(r.Context(), bson.M{"_id": id}).Decode(&b); err != nil {
		return nil, errors.New("filial topilmadi")
	}
	return &b, nil
}

// defaultBranch is the branch a brand is served from when the guest has not
// picked one and the request carries no address — dine-in, pickup listings, the
// floor plan. The first active branch of the brand.
func (h *Handler) defaultBranch(r *http.Request, brandID primitive.ObjectID) (*models.Branch, error) {
	filter := bson.M{"isActive": true}
	if !brandID.IsZero() {
		filter["brandId"] = brandID
	}
	branches, err := h.listBranchesFiltered(r, filter)
	if err != nil {
		return nil, err
	}
	if len(branches) == 0 {
		return nil, errors.New("filial topilmadi")
	}
	return &branches[0], nil
}

// deliveryBranch picks which branch cooks and carries an order to a point.
//
// The guest never chooses: they gave an address, and the company knows which
// kitchen covers it. Among the branches that do cover it, the nearest wins —
// a shorter run is a hotter samsa, and the fee follows the branch anyway.
// Returns the branch together with its quote so the caller does not price the
// address twice.
//
// ⚠️ **A branch that can cook the whole basket beats a nearer one that cannot.**
// Sold-out used to be checked only *after* this had chosen, which meant one
// finished dish at the nearest kitchen refused the order outright — while a
// branch three kilometres further on, inside its own delivery area, had
// everything. The guest was told "lag'mon tugadi" by a company that had
// lag'mon.
//
// The preference is deliberately narrow: it never widens the area anybody
// delivers to, because every candidate has already passed its own `maxKm` or
// its own zones. It only changes *which* of the kitchens that already agreed
// to carry this address gets the ticket.
//
// `dishes` may be empty (the plain "what would delivery cost here" question),
// and then this is exactly what it always was: nearest wins.
func (h *Handler) deliveryBranch(
	r *http.Request, brandID primitive.ObjectID,
	lat, lng float64, subtotal int, dishes []primitive.ObjectID,
) (*models.Branch, deliveryQuote, error) {
	filter := bson.M{"isActive": true}
	if !brandID.IsZero() {
		filter["brandId"] = brandID
	}
	branches, err := h.listBranchesFiltered(r, filter)
	if err != nil {
		return nil, deliveryQuote{}, err
	}

	cands := make([]branchCandidate, 0, len(branches))
	for i := range branches {
		b := &branches[i]
		q := quoteDeliveryFrom(b.Delivery, b.Address, lat, lng, subtotal)
		if !q.Available {
			continue
		}
		cands = append(cands, branchCandidate{
			Branch: b,
			Quote:  q,
			Dist:   haversineKm(b.Address.Lat, b.Address.Lng, lat, lng),
			Full:   !anySoldOut(b, dishes),
		})
	}
	pick := bestBranch(cands)
	if pick == nil {
		return nil, deliveryQuote{}, errNoBranchCovers
	}
	return pick.Branch, pick.Quote, nil
}

// branchCandidate is one branch that has already agreed to carry this address:
// its own zones or its own maxKm said yes.
type branchCandidate struct {
	Branch *models.Branch
	Quote  deliveryQuote
	Dist   float64
	// Whether this kitchen has everything in the basket.
	Full bool
}

// bestBranch applies the choice: a kitchen that can cook the whole basket beats
// a nearer one that cannot, and among equals the nearest wins.
//
// ⚠️ Pulled out as a pure function because it is the rule, not the plumbing —
// and because the fallback is the half that is easy to lose. A basket **nobody**
// can fill still has to resolve to a branch: that is the one whose name and
// stop list produce the honest "lag'mon tugadi", and returning nothing instead
// would turn a nameable refusal into "we don't deliver here".
func bestBranch(cands []branchCandidate) *branchCandidate {
	var best, full *branchCandidate
	for i := range cands {
		c := &cands[i]
		if best == nil || c.Dist < best.Dist {
			best = c
		}
		if c.Full && (full == nil || c.Dist < full.Dist) {
			full = c
		}
	}
	if full != nil {
		return full
	}
	return best
}

// anySoldOut reports whether this branch has run out of anything in the basket.
// Asks the branch itself, so the counter's list and the till's stop list are
// both covered — a dish stopped in iiko is as unsellable as one the counter
// ticked off.
func anySoldOut(b *models.Branch, dishes []primitive.ObjectID) bool {
	for _, id := range dishes {
		if b.IsSoldOut(id) {
			return true
		}
	}
	return false
}

// errNoBranchCovers means the address is outside every branch's delivery area —
// a normal answer, not a failure, so callers turn it into a plain message.
var errNoBranchCovers = errors.New("bu manzilga yetkazib berilmaydi")

func (h *Handler) listBranchesFiltered(r *http.Request, filter bson.M) ([]models.Branch, error) {
	cur, err := h.Store.Branches.Find(r.Context(), filter, branchSort())
	if err != nil {
		return nil, err
	}
	out := []models.Branch{}
	if err := cur.All(r.Context(), &out); err != nil {
		return nil, err
	}
	// Nil slices marshal as `null`, and the panel reads both lists straight into
	// a Set. A branch that has never had anything stopped is the common case, so
	// this is the path that would break first.
	for i := range out {
		if out[i].SoldOut == nil {
			out[i].SoldOut = []primitive.ObjectID{}
		}
		if out[i].POSSoldOut == nil {
			out[i].POSSoldOut = []primitive.ObjectID{}
		}
	}
	return out, nil
}
