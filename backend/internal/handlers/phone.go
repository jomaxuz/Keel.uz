package handlers

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

const (
	codeTTL        = 3 * time.Minute
	codeResendWait = 60 * time.Second
	codeMaxTries   = 5
)

// What a one-time code may be used for. See models.PhoneCode.Purpose: the owner
// is usually a customer on the same number, so a code issued for one flow must
// never be accepted by another.
const (
	purposeLogin = "login"
	// Resetting a forgotten panel password, and setting the recovery number the
	// reset is sent to.
	purposeAdminReset = "admin_reset"
	purposeAdminPhone = "admin_phone"
)

var digitsOnly = regexp.MustCompile(`\D`)

// normalizePhone turns user input into the 998XXXXXXXXX form used by both Uzbek
// SMS gateways. Accepts "+998 90 123 45 67", "998901234567" and "901234567".
func normalizePhone(raw string) (string, bool) {
	d := digitsOnly.ReplaceAllString(raw, "")
	switch {
	case len(d) == 9: // local number without the country code
		d = "998" + d
	case len(d) == 12 && strings.HasPrefix(d, "998"):
	default:
		return "", false
	}
	return d, true
}

func randomCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

type phoneRequest struct {
	Phone string `json:"phone"`
}

type phoneVerifyRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
	Name  string `json:"name"` // optional, used when creating the account
}

// issueCode stores a fresh code for the phone and sends it, enforcing the
// resend cooldown. Returns the plain code (only ever exposed in demo mode).
func (h *Handler) issueCode(ctx context.Context, phone, purpose string) (string, error) {
	key := bson.M{"phone": phone, "purpose": purpose}
	var existing models.PhoneCode
	err := h.Store.PhoneCodes.FindOne(ctx, key).Decode(&existing)
	if err == nil && time.Since(existing.CreatedAt) < codeResendWait {
		wait := int((codeResendWait - time.Since(existing.CreatedAt)).Seconds())
		return "", fmt.Errorf("kod yaqinda yuborilgan, %d soniyadan keyin qayta urining", wait)
	}
	if err != nil && err != mongo.ErrNoDocuments {
		return "", err
	}

	code, err := randomCode()
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	now := time.Now()
	_, err = h.Store.PhoneCodes.UpdateOne(ctx, key,
		bson.M{"$set": models.PhoneCode{
			Phone:     phone,
			Purpose:   purpose,
			CodeHash:  string(hash),
			Attempts:  0,
			ExpiresAt: now.Add(codeTTL),
			CreatedAt: now,
		}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return "", err
	}

	// The wording comes from the settings the owner had moderated, not from a
	// constant: see smsCodeText.
	text := smsCodeText(smsTemplateOf(h.smsSettings(ctx)), code)
	if err := h.sender(ctx).Send(ctx, phone, text); err != nil {
		// ⚠️ **Recorded where the owner will see it.** A gateway that starts
		// refusing says nothing on its own: the restaurant finds out from a
		// guest who could not sign in, and hears it as "your site is broken".
		// The settings page already answers "did the last test pass"; this
		// answers the more important question, "is a real guest getting in
		// right now", and it is the only one nobody is watching for.
		h.recordSMSFailure(ctx, err)
		return "", smsSendError{err: err}
	}
	return code, nil
}

// smsSendError marks a failure that came from the gateway rather than from the
// caller.
//
// The distinction is the status code and the wording: a cooldown is the
// person's own doing and is safe to explain in full, a gateway refusal is the
// restaurant's configuration and must not be quoted at a stranger.
type smsSendError struct{ err error }

func (e smsSendError) Error() string { return e.err.Error() }
func (e smsSendError) Unwrap() error { return e.err }

// errSMSSendFailed is what a guest is told. Deliberately says nothing about the
// gateway.
//
// ⚠️ **The public endpoint used to return the gateway's own words**, so a guest
// on an unmoderated Eskiz account received a Russian JSON blob with a request
// id in it: `SMS yuborilmadi: eskiz send: 400 {"message":"Для теста можно…"}`.
// Two things wrong with that at once — it is unreadable to the person it is
// shown to, and it publishes the restaurant's gateway state to anybody who
// types a phone number into a login form.
var errSMSSendFailed = errors.New(
	"kod yuborilmadi. Birozdan keyin urinib ko'ring yoki restoran bilan bog'laning")

// recordSMSFailure stores the real reason for the panel.
//
// Written on the settings document next to the test results, because that is
// the page somebody opens when SMS is suspected — and a failure with no
// timestamp beside it cannot be told from one that was fixed a week ago.
func (h *Handler) recordSMSFailure(ctx context.Context, cause error) {
	log.Printf("sms send: %v", cause)
	_, _ = h.Store.SMSSettings.UpdateOne(ctx, bson.M{},
		bson.M{"$set": bson.M{
			"lastErrorAt": time.Now(),
			"lastError":   clampText(cause.Error(), 300),
		}}, options.Update().SetUpsert(true))
}

// smsRequestFailed answers a code request that could not be sent.
//
// One helper because all three flows — guest login, panel password reset,
// changing the recovery number — face the same two failures and used to answer
// **both with 429**, i.e. "too many requests" for a gateway that refused. A
// status code that is wrong in the common case is one nothing can be built on.
func smsRequestFailed(w http.ResponseWriter, err error) {
	var sendErr smsSendError
	if errors.As(err, &sendErr) {
		httpx.Error(w, http.StatusServiceUnavailable, errSMSSendFailed.Error())
		return
	}
	// The cooldown, and it says how many seconds are left: this one is the
	// caller's own doing and explaining it fully is the helpful answer.
	httpx.Error(w, http.StatusTooManyRequests, err.Error())
}

// PhoneRequestCode sends a one-time login code to the given phone number.
func (h *Handler) PhoneRequestCode(w http.ResponseWriter, r *http.Request) {
	var req phoneRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	phone, ok := normalizePhone(req.Phone)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "telefon raqami noto'g'ri")
		return
	}
	// Refused before a code is even generated: no gateway means no login, not
	// a login anybody can complete by reading the response.
	expose, err := h.smsUsableFor(r, phone)
	if err != nil {
		httpx.Error(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	code, err := h.issueCode(r.Context(), phone, purposeLogin)
	if err != nil {
		smsRequestFailed(w, err)
		return
	}

	res := map[string]any{
		"ok":        true,
		"phone":     phone,
		"expiresIn": int(codeTTL.Seconds()),
		"demo":      expose,
	}
	// Only a deployment that deliberately asked for it — a developer's own
	// .env — ever sees the code. See smsUsable.
	if expose {
		res["code"] = code
	}
	httpx.JSON(w, http.StatusOK, res)
}

// checkCode validates the submitted code and consumes it on success.
func (h *Handler) checkCode(ctx context.Context, phone, purpose, code string) error {
	key := bson.M{"phone": phone, "purpose": purpose}
	var rec models.PhoneCode
	if err := h.Store.PhoneCodes.FindOne(ctx, key).Decode(&rec); err != nil {
		return fmt.Errorf("kod topilmadi, qaytadan so'rang")
	}
	if time.Now().After(rec.ExpiresAt) {
		_, _ = h.Store.PhoneCodes.DeleteOne(ctx, key)
		return fmt.Errorf("kod muddati tugagan, qaytadan so'rang")
	}
	if rec.Attempts >= codeMaxTries {
		_, _ = h.Store.PhoneCodes.DeleteOne(ctx, key)
		return fmt.Errorf("urinishlar soni tugadi, qaytadan so'rang")
	}
	if bcrypt.CompareHashAndPassword([]byte(rec.CodeHash), []byte(strings.TrimSpace(code))) != nil {
		_, _ = h.Store.PhoneCodes.UpdateOne(ctx, key,
			bson.M{"$inc": bson.M{"attempts": 1}})
		return fmt.Errorf("kod noto'g'ri")
	}
	_, _ = h.Store.PhoneCodes.DeleteOne(ctx, key)
	return nil
}

// PhoneVerify checks the code, creates the customer if needed and returns a JWT.
func (h *Handler) PhoneVerify(w http.ResponseWriter, r *http.Request) {
	var req phoneVerifyRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	phone, ok := normalizePhone(req.Phone)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "telefon raqami noto'g'ri")
		return
	}
	if err := h.checkCode(r.Context(), phone, purposeLogin, req.Code); err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	now := time.Now()
	name := strings.TrimSpace(req.Name)
	setOnInsert := bson.M{
		"createdAt":    now,
		"authProvider": "phone",
		"addresses":    []models.UserAddress{},
	}
	if name != "" {
		setOnInsert["firstName"] = name
	}
	_, err := h.Store.Users.UpdateOne(r.Context(),
		bson.M{"phone": phone},
		bson.M{
			"$set":         bson.M{"phone": phone, "updatedAt": now},
			"$setOnInsert": setOnInsert,
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	var user models.User
	if err := h.Store.Users.FindOne(r.Context(), bson.M{"phone": phone}).Decode(&user); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// A sign-up bonus, if the restaurant offers one. Idempotent, so signing in
	// again on a new phone does not hand it out a second time.
	h.grantWelcomePoints(r.Context(), &user)
	_ = h.Store.Users.FindOne(r.Context(), bson.M{"_id": user.ID}).Decode(&user)

	token, err := auth.Generate(h.Cfg.JWTSecret, user.ID.Hex(), "user")
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"token": token, "user": user})
}

// ---- profile ----

type updateProfileRequest struct {
	FirstName string               `json:"firstName"`
	LastName  string               `json:"lastName"`
	Addresses []models.UserAddress `json:"addresses"`
}

// UpdateUserMe lets the customer edit their name and saved addresses. The phone
// number is changed through the verified flow below, never here.
func (h *Handler) UpdateUserMe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	id, err := objectID(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var req updateProfileRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Addresses == nil {
		req.Addresses = []models.UserAddress{}
	}
	if len(req.Addresses) > 10 {
		req.Addresses = req.Addresses[:10]
	}
	update := bson.M{"$set": bson.M{
		"firstName": strings.TrimSpace(req.FirstName),
		"lastName":  strings.TrimSpace(req.LastName),
		"addresses": req.Addresses,
		"updatedAt": time.Now(),
	}}
	if _, err := h.Store.Users.UpdateOne(r.Context(), bson.M{"_id": id}, update); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var user models.User
	if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": id}).Decode(&user); err != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

// ChangePhoneRequest sends a code to the *new* number of a logged-in customer.
func (h *Handler) ChangePhoneRequest(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	if _, err := objectID(claims.UserID); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var req phoneRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	phone, ok := normalizePhone(req.Phone)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "telefon raqami noto'g'ri")
		return
	}
	taken, err := h.phoneTakenByOther(r.Context(), phone, claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if taken {
		httpx.Error(w, http.StatusConflict, "bu raqam boshqa foydalanuvchida band")
		return
	}
	expose, err := h.smsUsableFor(r, phone)
	if err != nil {
		httpx.Error(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	code, err := h.issueCode(r.Context(), phone, purposeLogin)
	if err != nil {
		smsRequestFailed(w, err)
		return
	}
	res := map[string]any{"ok": true, "phone": phone, "demo": expose}
	if expose {
		res["code"] = code
	}
	httpx.JSON(w, http.StatusOK, res)
}

// ChangePhoneVerify confirms the code and moves the customer to the new number.
func (h *Handler) ChangePhoneVerify(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	id, err := objectID(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var req phoneVerifyRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	phone, ok := normalizePhone(req.Phone)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "telefon raqami noto'g'ri")
		return
	}
	taken, err := h.phoneTakenByOther(r.Context(), phone, claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if taken {
		httpx.Error(w, http.StatusConflict, "bu raqam boshqa foydalanuvchida band")
		return
	}
	if err := h.checkCode(r.Context(), phone, purposeLogin, req.Code); err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	_, err = h.Store.Users.UpdateOne(r.Context(), bson.M{"_id": id},
		bson.M{"$set": bson.M{"phone": phone, "updatedAt": time.Now()}})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var user models.User
	if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": id}).Decode(&user); err != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (h *Handler) phoneTakenByOther(ctx context.Context, phone, userID string) (bool, error) {
	id, err := objectID(userID)
	if err != nil {
		return false, err
	}
	n, err := h.Store.Users.CountDocuments(ctx, bson.M{
		"phone": phone,
		"_id":   bson.M{"$ne": id},
	})
	return n > 0, err
}
