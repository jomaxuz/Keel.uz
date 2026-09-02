package handlers

import (
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// Panel accounts (owner-only endpoints — see the router).
//
// A new admin is never invented out of thin air: they must already be a site
// customer who signed in with their phone and a one-time SMS code. The owner
// picks that person and hands them a login and a password, which the account
// must change on its first sign-in (`mustChangePassword`).

type adminAccountPayload struct {
	// The site user this account belongs to. Required on create.
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// normaliseRole is what a panel account may be.
//
// ⚠️ **A closed list with "manager" as the fallback**, so an unknown word from
// a client cannot become a role nobody can reason about — and the fallback is
// the *narrower* of the two general roles, never "owner".
func normaliseRole(role string) string {
	switch role {
	case "owner", RoleOperator:
		return role
	default:
		return "manager"
	}
}

// AdminListAccounts returns every panel account, owners first.
func (h *Handler) AdminListAccounts(w http.ResponseWriter, r *http.Request) {
	opts := options.Find().SetSort(bson.D{
		{Key: "role", Value: 1}, {Key: "username", Value: 1},
	})
	cur, err := h.Store.Admins.Find(r.Context(), bson.M{}, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	accounts := []models.AdminUser{}
	_ = cur.All(r.Context(), &accounts)
	httpx.JSON(w, http.StatusOK, accounts)
}

func (h *Handler) AdminCreateAccount(w http.ResponseWriter, r *http.Request) {
	var req adminAccountPayload
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	username := strings.ToLower(strings.TrimSpace(req.Username))
	if len(username) < 3 {
		httpx.Error(w, http.StatusBadRequest, "login kamida 3 belgi bo'lishi kerak")
		return
	}
	if len(req.Password) < 6 {
		httpx.Error(w, http.StatusBadRequest, "parol kamida 6 belgi bo'lishi kerak")
		return
	}

	// The person must already exist as a phone-verified site customer.
	userID, err := objectID(strings.TrimSpace(req.UserID))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "foydalanuvchi tanlanmagan")
		return
	}
	var user models.User
	if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": userID}).Decode(&user); err != nil {
		httpx.Error(w, http.StatusNotFound, "foydalanuvchi topilmadi")
		return
	}

	if h.Store.Admins.FindOne(r.Context(), bson.M{"username": username}).Err() == nil {
		httpx.Error(w, http.StatusConflict, "bu login band")
		return
	}
	if h.Store.Admins.FindOne(r.Context(), bson.M{"userId": userID}).Err() == nil {
		httpx.Error(w, http.StatusConflict, "bu foydalanuvchida allaqachon panel hisobi bor")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	account := models.AdminUser{
		Username:     username,
		PasswordHash: string(hash),
		Role:         normaliseRole(req.Role),
		// The password was typed by someone else, so it cannot stay.
		MustChangePassword: true,
		UserID:             userID,
		Name:               strings.TrimSpace(user.FirstName + " " + user.LastName),
		Phone:              user.Phone,
		CreatedBy:          h.actingAdminName(r),
		CreatedAt:          time.Now(),
	}
	res, err := h.Store.Admins.InsertOne(r.Context(), account)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	account.ID = oidOf(res.InsertedID)
	h.logAction(r, ActAdminCreate, "admin", account.ID.Hex(), account.Username,
		account.Role+" · "+account.Phone)
	httpx.JSON(w, http.StatusCreated, account)
}

// AdminUpdateAccount changes the role and/or resets the password. A reset
// always re-arms mustChangePassword — a password someone else typed is
// temporary by definition.
func (h *Handler) AdminUpdateAccount(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req adminAccountPayload
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var account models.AdminUser
	if err := h.Store.Admins.FindOne(r.Context(), bson.M{"_id": id}).Decode(&account); err != nil {
		httpx.Error(w, http.StatusNotFound, "hisob topilmadi")
		return
	}

	set := bson.M{}
	details := []string{}

	if req.Role != "" && normaliseRole(req.Role) != account.Role {
		role := normaliseRole(req.Role)
		// Never leave the panel without an owner.
		if account.Role == "owner" && role != "owner" {
			owners, err := h.Store.Admins.CountDocuments(r.Context(), bson.M{"role": "owner"})
			if err == nil && owners <= 1 {
				httpx.Error(w, http.StatusBadRequest, "oxirgi egani boshqa rolga o'tkazib bo'lmaydi")
				return
			}
		}
		set["role"] = role
		details = append(details, account.Role+" → "+role)
	}

	if req.Password != "" {
		if len(req.Password) < 6 {
			httpx.Error(w, http.StatusBadRequest, "parol kamida 6 belgi bo'lishi kerak")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		set["passwordHash"] = string(hash)
		set["mustChangePassword"] = true
		details = append(details, "parol tiklandi")
	}

	if len(set) == 0 {
		httpx.JSON(w, http.StatusOK, account)
		return
	}
	if _, err := h.Store.Admins.UpdateByID(r.Context(), id, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.Store.Admins.FindOne(r.Context(), bson.M{"_id": id}).Decode(&account)
	h.logAction(r, ActAdminUpdate, "admin", id.Hex(), account.Username,
		strings.Join(details, ", "))
	httpx.JSON(w, http.StatusOK, account)
}

func (h *Handler) AdminDeleteAccount(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	claims := middleware.ClaimsFrom(r.Context())
	if claims != nil && claims.UserID == id.Hex() {
		httpx.Error(w, http.StatusBadRequest, "o'z hisobingizni o'chira olmaysiz")
		return
	}
	var account models.AdminUser
	if err := h.Store.Admins.FindOne(r.Context(), bson.M{"_id": id}).Decode(&account); err != nil {
		httpx.Error(w, http.StatusNotFound, "hisob topilmadi")
		return
	}
	if account.Role == "owner" {
		owners, err := h.Store.Admins.CountDocuments(r.Context(), bson.M{"role": "owner"})
		if err == nil && owners <= 1 {
			httpx.Error(w, http.StatusBadRequest, "oxirgi egani o'chirib bo'lmaydi")
			return
		}
	}
	if _, err := h.Store.Admins.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// The person keeps their customer account and order history; only the panel
	// login goes away.
	h.logAction(r, ActAdminDelete, "admin", id.Hex(), account.Username, account.Role)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// actingAdminName is the username behind the current request, for log lines and
// "created by" fields.
func (h *Handler) actingAdminName(r *http.Request) string {
	claims := middleware.ClaimsFrom(r.Context())
	if claims == nil {
		return ""
	}
	id, err := objectID(claims.UserID)
	if err != nil {
		return ""
	}
	var admin models.AdminUser
	if err := h.Store.Admins.FindOne(r.Context(), bson.M{"_id": id}).Decode(&admin); err != nil {
		return ""
	}
	return admin.Username
}
