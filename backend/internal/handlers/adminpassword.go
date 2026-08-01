package handlers

import (
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

// Recovering a forgotten panel password.
//
// Every install runs on its customer's own VPS, so "I forgot the password" used
// to mean an SSH session and `cmd/adminreset` — a support call for a restaurant
// owner who has never opened a terminal. This flow does it from the login page
// instead: a one-time code to the number already stored on the account.
//
// Two rules hold it together:
//   - The code is issued for **purposeAdminReset** only. The owner is usually
//     also a customer on the same phone, and a code texted for "sign in and
//     order lunch" must not be able to open the panel (see models.PhoneCode).
//   - Recovery is only possible for an account that has a phone. The seeded
//     first owner has none until one is set on /admin/account, and the message
//     below says so rather than failing silently.

// maskPhone hides the middle of a number: "998901234567" → "+998 ** *** 45 67".
// Enough for the owner to recognise which of their numbers the code went to,
// not enough to be worth harvesting.
func maskPhone(phone string) string {
	if len(phone) < 4 {
		return ""
	}
	return "+" + phone[:3] + " ** *** ** " + phone[len(phone)-2:]
}

type forgotPasswordRequest struct {
	Username string `json:"username"`
}

// AdminForgotPassword texts a one-time code to the account's recovery number.
//
// It answers honestly when the username does not exist. That does leak whether
// an account exists — a deliberate trade: this is a single restaurant's own
// panel, the reset still needs the phone in the owner's hand, and a blank
// "if the account exists…" leaves someone who mistyped their username with
// nothing to go on, which is the support call this whole flow exists to remove.
func (h *Handler) AdminForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	username := strings.TrimSpace(strings.ToLower(req.Username))
	if username == "" {
		httpx.Error(w, http.StatusBadRequest, "loginni yozing")
		return
	}

	var admin models.AdminUser
	if err := h.Store.Admins.FindOne(r.Context(),
		bson.M{"username": username}).Decode(&admin); err != nil {
		httpx.Error(w, http.StatusNotFound, "bunday login topilmadi")
		return
	}
	phone, ok := normalizePhone(admin.Phone)
	if !ok {
		// Nothing to text. Say exactly what to do instead of "try again".
		httpx.Error(w, http.StatusBadRequest,
			"bu hisobga tiklash uchun telefon raqami biriktirilmagan — "+
				"parolni serverda tiklash kerak (DEPLOY.md)")
		return
	}

	code, err := h.issueCode(r.Context(), phone, purposeAdminReset)
	if err != nil {
		// The 60-second cooldown lives in issueCode; it also caps how often
		// someone can make an owner's phone buzz.
		httpx.Error(w, http.StatusTooManyRequests, err.Error())
		return
	}

	res := map[string]any{
		"ok":        true,
		"phone":     maskPhone(phone),
		"expiresIn": int(codeTTL.Seconds()),
		"demo":      h.SMS.Demo(),
	}
	if h.SMS.Demo() {
		res["code"] = code
	}
	httpx.JSON(w, http.StatusOK, res)
}

type resetPasswordRequest struct {
	Username    string `json:"username"`
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}

// AdminResetPassword sets a new password once the texted code checks out.
func (h *Handler) AdminResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	username := strings.TrimSpace(strings.ToLower(req.Username))
	if len(req.NewPassword) < 6 {
		httpx.Error(w, http.StatusBadRequest, "parol kamida 6 belgi bo'lishi kerak")
		return
	}

	var admin models.AdminUser
	if err := h.Store.Admins.FindOne(r.Context(),
		bson.M{"username": username}).Decode(&admin); err != nil {
		httpx.Error(w, http.StatusNotFound, "bunday login topilmadi")
		return
	}
	phone, ok := normalizePhone(admin.Phone)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "bu hisobda telefon raqami yo'q")
		return
	}
	if err := h.checkCode(r.Context(), phone, purposeAdminReset, req.Code); err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := h.Store.Admins.UpdateByID(r.Context(), admin.ID, bson.M{
		"$set": bson.M{
			"passwordHash": string(hash),
			// The owner chose this password themselves, so there is nothing to
			// force them to change on the next sign-in.
			"mustChangePassword": false,
		},
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// A password reset always leaves a trail, even though nobody was signed in
	// when it happened: "who changed the owner's password?" must be answerable.
	_, _ = h.Store.AdminLogs.InsertOne(r.Context(), models.AdminLog{
		AdminID:     admin.ID,
		AdminName:   admin.Username,
		AdminRole:   admin.Role,
		Action:      ActAdminPasswordReset,
		TargetType:  "admin",
		TargetID:    admin.ID.Hex(),
		TargetLabel: admin.Username,
		Details:     "SMS orqali tiklandi: " + maskPhone(phone),
		At:          time.Now(),
	})

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Recovery phone (set from inside the panel) ----

type adminPhoneRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

// AdminPhoneRequest texts a code to a number the signed-in admin wants to use
// for recovery. Requiring the code proves the number is theirs and reachable —
// a mistyped digit here would only be discovered on the day it is needed.
func (h *Handler) AdminPhoneRequest(w http.ResponseWriter, r *http.Request) {
	if _, err := h.adminUser(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req adminPhoneRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	phone, ok := normalizePhone(req.Phone)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "telefon raqami noto'g'ri")
		return
	}
	code, err := h.issueCode(r.Context(), phone, purposeAdminPhone)
	if err != nil {
		httpx.Error(w, http.StatusTooManyRequests, err.Error())
		return
	}
	res := map[string]any{"ok": true, "phone": phone, "demo": h.SMS.Demo()}
	if h.SMS.Demo() {
		res["code"] = code
	}
	httpx.JSON(w, http.StatusOK, res)
}

// AdminPhoneVerify stores the recovery number after the code checks out.
func (h *Handler) AdminPhoneVerify(w http.ResponseWriter, r *http.Request) {
	admin, err := h.adminUser(r)
	if err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req adminPhoneRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	phone, ok := normalizePhone(req.Phone)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "telefon raqami noto'g'ri")
		return
	}
	if err := h.checkCode(r.Context(), phone, purposeAdminPhone, req.Code); err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	// Two panel accounts on one phone would make "reset by username" ambiguous
	// to the person reading the SMS: both codes look identical.
	n, err := h.Store.Admins.CountDocuments(r.Context(), bson.M{
		"phone": phone,
		"_id":   bson.M{"$ne": admin.ID},
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n > 0 {
		httpx.Error(w, http.StatusConflict, "bu raqam boshqa admin hisobida band")
		return
	}
	if _, err := h.Store.Admins.UpdateByID(r.Context(), admin.ID,
		bson.M{"$set": bson.M{"phone": phone}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActAdminPhone, "admin", admin.ID.Hex(), admin.Username,
		maskPhone(phone))
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "phone": phone})
}
