package handlers

import (
	"context"
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
	ActOrderCreate = "order.create"
	ActOrderStatus = "order.status"
	ActOrderCancel = "order.cancel"
	// An order handed to a different kitchen by hand. Logged because the money
	// deliberately does not follow it: "why is this Chilonzor order being
	// cooked in Sergeli" needs an answer with a name on it.
	ActOrderMoveBranch = "order.branch"
	ActOrderCourier    = "order.courier"
	ActOrderExternal   = "order.external"
	ActOrderAddress    = "order.address"

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
	// Somebody's private words moved onto the public site, or came back off.
	ActFeedbackPublished   = "feedback.published"
	ActFeedbackUnpublished = "feedback.unpublished"

	ActCourierCreate = "courier.create"
	ActCourierUpdate = "courier.update"
	ActCourierDelete = "courier.delete"
	// The courier handed collected cash back to the restaurant.
	ActCourierSettle = "courier.settle"

	ActMenuCreate = "menu.create"
	ActMenuUpdate = "menu.update"
	ActMenuDelete = "menu.delete"

	// Moving in from another till system. ⚠️ Logged as its own action rather
	// than as a hundred separate creates: what the journal is asked afterwards
	// is "where did all of this come from", and a hundred rows answers it worse
	// than one does.
	ActPosImport = "pos.import"

	ActCategoryCreate = "category.create"
	ActCategoryUpdate = "category.update"
	ActCategoryDelete = "category.delete"

	ActSettingsUpdate = "settings.update"
	// The till. Logged because these three are the only admin actions that
	// move physical cash, and a shortfall with no trail is an argument.
	ActCashShiftOpen  = "cash.shift.open"
	ActCashShiftClose = "cash.shift.close"
	ActCashEntry      = "cash.entry"
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
	// The SMS gateway login codes go out through. The gateway that was chosen
	// is recorded; its password never touches the log.
	// The restaurant's own Telegram bot: whether it is on, never the token.
	ActTelegramSettings = "settings.telegram"
	ActSMSSettings      = "settings.sms"
	ActSMSTest          = "settings.sms.test"
	// The whole business leaving as one archive. Logged like a cash movement
	// and for the same reason: it is the single action here with no undo and no
	// other trace, and the question afterwards is always "who, and when".
	ActDataExport = "data.export"
	// A message sent to a whole segment. Logged because it spends money and
	// reaches people's phones, and because it cannot be recalled.
	ActCampaignSend = "campaign.send"

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

	// The strip the restaurant edits itself, and the jobs it advertises. Logged like
	// every other change: "who put that banner up" is asked the day one is wrong.
	ActWarehouseSave   = "warehouse.save"
	ActWarehouseDelete = "warehouse.delete"
	ActBannerSave      = "banner.save"
	ActBannerDelete    = "banner.delete"
	ActVacancySave     = "vacancy.save"
	ActVacancyDelete   = "vacancy.delete"
	ActJobStatus       = "job.status"

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
	h.alertOnSensitiveAction(r, entry)
}

// logActionAs records an action taken by work that outlives its request.
//
// ⚠️ **Its own function because the request is gone by then.** A background
// import finishes minutes after the handler returned; reading claims off a
// `*http.Request` the server has since reused is a data race whose symptom is
// a journal entry attributed to whoever happened to be logged in next. The
// name is read on the request and carried in.
//
// ⚠️ It does not raise the sensitive-action alert. Nothing that runs in the
// background is on that list, and an alert needs a request to know which branch
// and which chat it belongs to.
func (h *Handler) logActionAs(
	ctx context.Context, who, action, targetType, targetID, label, details string,
) {
	_, _ = h.Store.AdminLogs.InsertOne(ctx, models.AdminLog{
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		TargetLabel: label,
		Details:     details,
		AdminName:   who,
		At:          time.Now(),
	})
}

// sensitiveActions are the panel actions worth waking an owner for.
//
// ⚠️ **Chosen from what the journal already records, not written again.** Every
// one of these was being logged the whole time — the journal is complete and
// nobody reads it, which is the same as it not existing on the evening it
// matters. This layer decides which lines are worth a message tonight rather
// than a discovery in March.
//
// ⚠️ **Single events only, and each is unusual on its own.** Editing a menu
// price is normal work; downloading the entire customer base is not. The rule
// the notification channel is built on holds here: patterns go to the briefing,
// and a message that arrives most days is a message that gets muted.
var sensitiveActions = map[string]string{
	// ⚠️ **The classic departure.** An operator with a laptop and a notice
	// period takes the customer list with them, and the restaurant finds out
	// when a competitor starts calling its regulars. The export already
	// required a grant and was already logged; nobody was ever told.
	ActDataExport: "Mijozlar bazasi yuklab olindi",

	// Somebody giving themselves, or somebody else, a way in.
	ActAdminCreate:      "Yangi panel hisobi ochildi",
	ActAdminCredentials: "Panel hisobining paroli o'zgartirildi",
	ActAdminDelete:      "Panel hisobi o'chirildi",

	// ⚠️ **Customer deletion is deliberately absent**, and that is a finding
	// rather than an omission: nothing in the panel logs it under its own
	// action today. When it gets one, it belongs on this list — a customer
	// removed takes their order history with them, which is also the record of
	// everything that was ever done to their orders.
}

// alertOnSensitiveAction tells the owner about a panel action worth knowing.
//
// ⚠️ **An owner's own actions are reported too, and that reverses a call I
// made and got wrong.**
//
// The argument for skipping them was that an owner is who the message is for,
// and a channel that reports the reader to themselves gets muted. It sounds
// right and it fails on first contact: an owner setting this up tests it as
// themselves, sees nothing, and concludes the feature is broken — which is
// exactly what happened. A channel that cannot be made to fire by the person
// who owns it is a channel nobody can ever verify.
//
// And the noise argument was weaker than it looked. These are four actions, not
// four hundred: an export, and accounts being created, deleted or re-credentialed.
// An owner who does one of those a month gets one message a month, and it is a
// message about something worth a record.
func (h *Handler) alertOnSensitiveAction(r *http.Request, e models.AdminLog) {
	head, ok := sensitiveActions[e.Action]
	if !ok {
		return
	}
	branch := h.alertBranchFor(r)
	if !h.alertSettingsOf(r.Context(), branch).Enabled {
		return
	}
	who := e.AdminName
	if who == "" {
		who = e.AdminRole
	}
	h.raiseAlert(models.LossAlert{
		BranchID: branch,
		Kind:     models.AlertPanelAction,
		At:       e.At,
		ByID:     e.AdminID,
		By:       who,
		// ⚠️ No money figure, and none invented. What a downloaded customer
		// list is worth is not a number anybody can put on it, and a zero here
		// prints nothing rather than "0 so'm".
		Amount:  0,
		Subject: head,
		Reason:  strings.TrimSpace(e.TargetLabel + " " + e.Details),
	})
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
			// ⚠️ Escaped: the group name comes from ?action= and reaches Mongo's
			// regex engine, so a metacharacter in it is a pattern rather than a
			// literal — a crafted value is a catastrophic-backtracking DoS.
			filter["action"] = bson.M{"$regex": "^" + regexp.QuoteMeta(ids[0]) + "\\."}
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
