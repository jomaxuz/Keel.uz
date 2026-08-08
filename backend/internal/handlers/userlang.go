package handlers

import (
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

// The language a guest **chose**, stored on their account.
//
// The site has always had a language switch, and it has always been a cookie —
// which is right for the site: it is a per-device display setting, and a guest
// who reads Russian on their phone and Uzbek on a shared tablet is not
// contradicting themselves.
//
// Telegram breaks that model in two places, which is why this exists:
//
//   - **The bot writes to them later.** An order notification is sent from a
//     background goroutine, minutes or hours after the page that would have
//     carried the cookie is gone. Guessing at that moment is the one thing we
//     cannot do, and guessing wrong sends a Russian speaker an Uzbek message
//     about their own money.
//   - **A mini app has no address bar**, so the `/ru/` URL that carries the
//     choice on the open web is not available to be shared or bookmarked. The
//     choice has to be remembered somewhere, and the account is the only place
//     that survives a reinstall and follows the guest to a second device.
//
// ⚠️ This does not replace the cookie, and both are written together: the cookie
// is what the *next server render* reads, and it is read on every page by code
// that must not wait on a database. The account copy is what everything without
// a browser reads. Keeping one and dropping the other breaks a different half.

// langAllowed is the whitelist, and it is a whitelist on purpose.
//
// ⚠️ This value chooses a **message template** (see orderStatusMessage) and, in
// the browser, a dictionary key. Anything a client can type must be reduced to
// one of three known strings before it is stored — not because an unknown value
// would crash, but because it would silently fall through to Uzbek forever and
// look like the guest's choice was ignored.
func langAllowed(v string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "uz":
		return "uz", true
	case "ru":
		return "ru", true
	case "en":
		return "en", true
	}
	return "", false
}

// notifyLang decides which language a message to this guest is written in.
//
// The order is the point: an **explicit** choice beats an inferred one, always.
//
//	Lang          — they picked it, in the mini app's first screen
//	TelegramLang  — Telegram's UI setting; a decent guess, nothing more
//	""            — Uzbek, the base language everywhere in this app
//
// Written as a pure function so the precedence is testable without a bot, a
// database or an order — and so the fallback chain is visible in one place
// rather than spread across the call sites that need it.
func notifyLang(u *models.User) string {
	if u == nil {
		return "uz"
	}
	if l, ok := langAllowed(u.Lang); ok {
		return l
	}
	return normalizeLang(u.TelegramLang)
}

type userLangRequest struct {
	Lang string `json:"lang"`
}

// UserSetLang stores the guest's language choice.
//
// Deliberately its own endpoint rather than a field on `PUT /users/me`: that one
// is the profile form and it writes name and addresses on every call, so sending
// a language through it from the mini app's first screen would blank the name of
// a guest who had one. The two are also reached at completely different moments —
// this one before the guest has seen the app at all.
func (h *Handler) UserSetLang(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	id, err := objectID(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var req userLangRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	lang, ok := langAllowed(req.Lang)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "unsupported language")
		return
	}
	update := bson.M{"$set": bson.M{"lang": lang, "updatedAt": time.Now()}}
	if _, err := h.Store.Users.UpdateOne(r.Context(), bson.M{"_id": id}, update); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "lang": lang})
}
