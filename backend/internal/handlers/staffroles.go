package handlers

import (
	"errors"
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

// Roles: the job titles a restaurant hands out, and what each one may do.
//
// ⚠️ **Company-wide, not per branch.** A "Kassir" who may write off food in one
// kitchen and may not in the next is a rule nobody can hold in their head, and
// the person it confuses is the manager who moves between them.
//
// ⚠️ **Editing is owner-only; reading is not.** Deciding who may take money out
// of the restaurant is not a shift-level decision, and a branch manager who
// could widen a role could widen their own. But every screen that assigns
// somebody a role has to be able to list them, so reading stays open to anyone
// who can already manage staff.

// AdminListRoles returns the roles, in the order the panel draws them.
func (h *Handler) AdminListRoles(w http.ResponseWriter, r *http.Request) {
	cur, err := h.Store.StaffRoles.Find(r.Context(), bson.M{},
		options.Find().SetSort(bson.D{{Key: "sort", Value: 1}, {Key: "name", Value: 1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	roles := []models.StaffRole{}
	if err := cur.All(r.Context(), &roles); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ How many people hold each one, because that is the number an owner
	// needs before editing it: widening "Ofitsiant" when eleven people hold it
	// is a different act from widening one nobody uses.
	counts := map[string]int{}
	if agg, err := h.Store.Staff.Aggregate(r.Context(), []bson.M{
		{"$match": bson.M{"roleId": bson.M{"$exists": true}}},
		{"$group": bson.M{"_id": "$roleId", "n": bson.M{"$sum": 1}}},
	}); err == nil {
		var rows []struct {
			ID any `bson:"_id"`
			N  int `bson:"n"`
		}
		if agg.All(r.Context(), &rows) == nil {
			for _, row := range rows {
				if id, ok := row.ID.(interface{ Hex() string }); ok {
					counts[id.Hex()] = row.N
				}
			}
		}
	}

	out := make([]map[string]any, 0, len(roles))
	for _, role := range roles {
		out = append(out, map[string]any{
			"id": role.ID.Hex(), "name": role.Name,
			// ⚠️ Never nil: a nil slice serialises as `null` and the editor
			// would render `null.length`. The trap this codebase has hit twice.
			"perms":  nonNilPerms(role.Perms),
			"seeded": role.Seeded, "sort": role.Sort,
			"staffCount": counts[role.ID.Hex()],
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"roles": out,
		// The permission vocabulary comes from the server so the panel cannot
		// offer a switch the server does not understand — the same reason the
		// fiscal provider list is an endpoint rather than a constant.
		"perms": permCatalogue(w),
	})
}

func nonNilPerms(p []string) []string {
	if p == nil {
		return []string{}
	}
	return p
}

// permCatalogue is what the role editor draws, in a fixed order: the floor
// first, the money after it, the kitchen last.
func permCatalogue(w http.ResponseWriter) []map[string]string {
	out := make([]map[string]string, 0, len(models.AllPerms))
	for _, p := range models.AllPerms {
		out = append(out, map[string]string{"id": p, "name": permLabel(w, p)})
	}
	return out
}

type roleRequest struct {
	Name  string   `json:"name"`
	Perms []string `json:"perms"`
}

// clean normalises a submitted role.
//
// ⚠️ Unknown permissions are **dropped, not stored**. A client sending
// `"superuser"` must not leave a word in the database that a future release
// might one day give a meaning to.
func (req roleRequest) clean() (string, []string, error) {
	name := clampText(req.Name, 60)
	if name == "" {
		return "", nil, errRoleName
	}
	known := map[string]bool{}
	for _, p := range models.AllPerms {
		known[p] = true
	}
	seen := map[string]bool{}
	perms := []string{}
	for _, p := range req.Perms {
		p = strings.TrimSpace(p)
		if known[p] && !seen[p] {
			seen[p] = true
			perms = append(perms, p)
		}
	}
	return name, perms, nil
}

var errRoleName = errors.New("rol nomini yozing")

func (h *Handler) AdminCreateRole(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req roleRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	name, perms, err := req.clean()
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	role := models.StaffRole{
		Name: name, Perms: perms, Seeded: false,
		// New roles sort after the seeded ones rather than jumping to the top:
		// the list reads by authority, and a role added on Tuesday has not
		// earned the first position.
		Sort: 100, CreatedAt: now, UpdatedAt: now,
	}
	res, err := h.Store.StaffRoles.InsertOne(r.Context(), role)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	role.ID = oidOf(res.InsertedID)
	h.logAction(r, ActStaffUpdate, "role", role.ID.Hex(), name, strings.Join(perms, ","))
	httpx.JSON(w, http.StatusOK, role)
}

func (h *Handler) AdminUpdateRole(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req roleRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	name, perms, err := req.clean()
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.Store.StaffRoles.UpdateByID(r.Context(), id, bson.M{"$set": bson.M{
		"name": name, "perms": perms, "updatedAt": time.Now(),
	}})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.MatchedCount == 0 {
		httpx.Error(w, http.StatusNotFound, "rol topilmadi")
		return
	}
	// ⚠️ Logged with the permission list, not just the name. "Menejer roli
	// o'zgartirildi" a month later answers nothing; the question being asked
	// then is which abilities moved and when.
	h.logAction(r, ActStaffUpdate, "role", id.Hex(), name, strings.Join(perms, ","))
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// AdminDeleteRole removes a role nobody holds.
//
// ⚠️ **Refused while anybody holds it**, and this is not tidiness. A staff
// record whose role has vanished falls back to the legacy booleans (see
// withRole), which for most people means losing the till mid-shift — and the
// tablet would say nothing about why. Reassigning first is the only order that
// cannot surprise anybody.
func (h *Handler) AdminDeleteRole(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	n, err := h.Store.Staff.CountDocuments(r.Context(), bson.M{"roleId": id})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n > 0 {
		httpx.Error(w, http.StatusConflict,
			"bu rol ishchilarga berilgan — avval ularga boshqa rol bering")
		return
	}
	if _, err := h.Store.StaffRoles.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActStaffDelete, "role", id.Hex(), "", "")
	httpx.JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// roleRef turns a submitted role id into one that exists.
//
// ⚠️ Validated rather than trusted. An id naming no role leaves that person
// falling back to the legacy booleans (see withRole) — which for most people
// means losing the till mid-shift, with nothing on the tablet saying why.
//
// An empty string is allowed and clears the role: taking somebody off the till
// without deleting their account is a real thing to want.
func (h *Handler) roleRef(r *http.Request, id string) (primitive.ObjectID, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return primitive.NilObjectID, nil
	}
	oid, err := objectID(id)
	if err != nil {
		return primitive.NilObjectID, errors.New("noma'lum rol")
	}
	n, err := h.Store.StaffRoles.CountDocuments(r.Context(), bson.M{"_id": oid})
	if err != nil {
		return primitive.NilObjectID, err
	}
	if n == 0 {
		return primitive.NilObjectID, errors.New("noma'lum rol")
	}
	return oid, nil
}
