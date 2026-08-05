package handlers

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Action ids. They are stored, not shown: the panel translates each id, so
// renaming a label never rewrites history.
const (
	ActLogin = "admin.login"

	// An order an operator typed in over the phone. Orders the guest placed
	// themselves are not logged here — the panel did not do anything.
	ActOrderCreate   = "order.create"
	ActOrderStatus   = "order.status"
	ActOrderCancel   = "order.cancel"
	ActOrderCourier  = "order.courier"
	ActOrderExternal = "order.external"
	ActOrderAddress  = "order.address"

	ActReservationStatus = "reservation.status"
	ActReservationCancel = "reservation.cancel"
	ActReservationDelete = "reservation.delete"

	ActAdminCreate      = "admin.create"
	ActAdminUpdate      = "admin.update"
	ActAdminDelete      = "admin.delete"
	ActAdminCredentials = "admin.credentials"
	// Reset from the login page with an SMS code — nobody was signed in.
	ActAdminPasswordReset = "admin.password.reset"
	ActAdminPhone         = "admin.phone"

	// Notes the restaurant keeps about a customer.
	ActUserUpdate = "user.update"
	// A complaint closed with a note on what was done about it.
	ActFeedbackHandled = "feedback.handled"

	ActCourierCreate = "courier.create"
	ActCourierUpdate = "courier.update"
	ActCourierDelete = "courier.delete"
	// The courier handed collected cash back to the restaurant.
	ActCourierSettle = "courier.settle"

	ActMenuCreate = "menu.create"
	ActMenuUpdate = "menu.update"
	ActMenuDelete = "menu.delete"

	ActCategoryCreate = "category.create"
	ActCategoryUpdate = "category.update"
	ActCategoryDelete = "category.delete"

	ActSettingsUpdate = "settings.update"
	// Online payment credentials. Which providers went on or off is recorded;
	// the keys themselves never touch the log.
	ActPaymentSettings = "settings.payments"
	// The till the restaurant runs: its connection, the dish mapping, and one
	// order pushed across by hand.
	ActPOSSettings = "settings.pos"
	ActPOSMapping  = "settings.pos.menu"
	ActPOSSend     = "order.pos"
	// The phone system the call centre listens to.
	ActPBXSettings = "settings.pbx"

	ActStaffCreate = "staff.create"
	ActStaffUpdate = "staff.update"
	ActStaffDelete = "staff.delete"
	// A shift written or corrected by hand — a dead phone, a forgotten
	// clock-out. Attendance somebody typed must be distinguishable from
	// attendance somebody clocked.
	ActShiftEdit = "staff.shift"
	// Salary handed over.
	ActStaffPay = "staff.pay"
	// The branch kiosk key was replaced, revoking every screen token.
	ActKioskRotate = "staff.kiosk"

	ActBrandCreate  = "brand.create"
	ActBrandUpdate  = "brand.update"
	ActBrandDelete  = "brand.delete"
	ActBranchCreate = "branch.create"
	ActBranchUpdate = "branch.update"
	ActBranchDelete = "branch.delete"

	ActPromotionCreate = "promotion.create"
	ActPromotionUpdate = "promotion.update"
	ActPromotionDelete = "promotion.delete"

	ActProviderCreate = "provider.create"
	ActProviderUpdate = "provider.update"
	ActProviderDelete = "provider.delete"
)

// logAction records what the acting admin just did. Never fails the request:
// a missing log line must not undo a completed action.
func (h *Handler) logAction(r *http.Request, action, targetType, targetID, label, details string) {
	claims := middleware.ClaimsFrom(r.Context())
	entry := models.AdminLog{
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		TargetLabel: label,
		Details:     details,
		At:          time.Now(),
	}
	if claims != nil {
		entry.AdminRole = claims.Role
		if id, err := objectID(claims.UserID); err == nil {
			entry.AdminID = id
			var admin models.AdminUser
			if err := h.Store.Admins.FindOne(r.Context(),
				bson.M{"_id": id}).Decode(&admin); err == nil {
				entry.AdminName = admin.Username
			}
		}
	}
	_, _ = h.Store.AdminLogs.InsertOne(r.Context(), entry)
}

// logLogin records a successful sign-in. Separate from logAction because at
// that point there are no auth claims on the request yet.
func (h *Handler) logLogin(r *http.Request, admin *models.AdminUser) {
	_, _ = h.Store.AdminLogs.InsertOne(r.Context(), models.AdminLog{
		AdminID:   admin.ID,
		AdminName: admin.Username,
		AdminRole: admin.Role,
		Action:    ActLogin,
		At:        time.Now(),
	})
}

// AdminListLogs returns the activity log, newest first. Owner-only (see router):
// it is the record of what everyone else in the panel did.
//
// Filters: ?adminId= (one person), ?action= (comma-separated ids or a prefix
// like "order"), ?limit= (default 100, max 500), ?before= (RFC3339 — paging).
func (h *Handler) AdminListLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := bson.M{}

	if raw := strings.TrimSpace(q.Get("adminId")); raw != "" {
		id, err := objectID(raw)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid adminId")
			return
		}
		filter["adminId"] = id
	}

	if raw := strings.TrimSpace(q.Get("action")); raw != "" {
		ids := []string{}
		for _, part := range strings.Split(raw, ",") {
			if p := strings.TrimSpace(part); p != "" {
				ids = append(ids, p)
			}
		}
		switch {
		case len(ids) == 1 && !strings.Contains(ids[0], "."):
			// A whole group: "order" matches order.status, order.cancel, …
			filter["action"] = bson.M{"$regex": "^" + ids[0] + "\\."}
		case len(ids) > 0:
			filter["action"] = bson.M{"$in": ids}
		}
	}

	// Free text over what an entry actually shows: the order number or id, the
	// admin's name, the target label and the note. Operators paste an order
	// number ("#AL23-9004") or a database id straight out of a receipt.
	if raw := strings.TrimSpace(q.Get("q")); raw != "" {
		raw = strings.TrimPrefix(raw, "#")
		rx := bson.M{"$regex": regexp.QuoteMeta(raw), "$options": "i"}
		filter["$or"] = []bson.M{
			{"targetId": rx}, {"targetLabel": rx},
			{"details": rx}, {"adminName": rx}, {"action": rx},
		}
	}

	if raw := strings.TrimSpace(q.Get("before")); raw != "" {
		if ts, err := time.Parse(time.RFC3339, raw); err == nil {
			filter["at"] = bson.M{"$lt": ts}
		}
	}

	limit := 100
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 {
		limit = n
	}
	if limit > 500 {
		limit = 500
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "at", Value: -1}}).
		SetLimit(int64(limit))
	cur, err := h.Store.AdminLogs.Find(r.Context(), filter, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	logs := []models.AdminLog{}
	_ = cur.All(r.Context(), &logs)
	httpx.JSON(w, http.StatusOK, logs)
}
