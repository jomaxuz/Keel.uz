package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/fiscal"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ---- Fiscalisation settings ----
//
// One virtual cash register per branch. The shape follows the POS settings next
// door almost exactly, because the questions are the same: which provider, are
// the credentials there, and is it actually reaching anything right now.

// fiscalSettingsOf loads one branch's connection. A missing document is not an
// error — a restaurant that has not connected a cash register yet is the
// ordinary case, and will be for every install until an owner opens this page.
func (h *Handler) fiscalSettingsOf(ctx context.Context, branchID primitive.ObjectID) *models.FiscalSettings {
	var s models.FiscalSettings
	if branchID.IsZero() {
		return &models.FiscalSettings{}
	}
	if err := h.Store.FiscalSettings.FindOne(ctx, bson.M{"branchId": branchID}).Decode(&s); err != nil {
		return &models.FiscalSettings{BranchID: branchID}
	}
	return &s
}

// credsOf turns the stored drawer into what an adapter needs.
//
// ⚠️ The selection happens **here and only here**, so no adapter ever reads the
// settings document. That is what keeps one provider from being handed
// another's password — the mistake the separate drawers exist to prevent would
// come straight back if the picking were done in six places.
// drawers is which stored credentials belong to which provider.
//
// ⚠️ **One map, because this was three switches and a struct literal.** Adding
// a provider meant editing four lists, and forgetting one of them is silent in
// the worst way available here: the provider appears in the panel, the owner
// fills in their login, it saves — and the filing code reads an empty drawer
// and files nothing. Now the four places read this, so a provider is either
// wired up everywhere or nowhere.
func drawers(s *models.FiscalSettings) map[string]models.FiscalCreds {
	return map[string]models.FiscalCreds{
		fiscal.Multikassa: s.Multikassa,
		fiscal.FirstOFD:   s.FirstOFD,
		fiscal.EPOS:       s.EPOS,
		fiscal.Regos:      s.Regos,
		fiscal.Hippo:      s.Hippo,
		fiscal.Simurg:     s.Simurg,
		fiscal.Rahmat:     s.Rahmat,
		fiscal.QPOS:       s.QPOS,
		fiscal.Arca:       s.Arca,
	}
}

func credsOf(s *models.FiscalSettings) fiscal.Creds {
	c := drawers(s)[s.Provider]
	return fiscal.Creds{
		Login:      c.Login,
		Password:   c.Password,
		Token:      c.Token,
		RegisterID: c.RegisterID,
		BaseURL:    c.BaseURL,
		TIN:        s.TIN,
	}
}

// fiscalFor builds the client for a branch, or nil when nothing is connected.
func (h *Handler) fiscalFor(ctx context.Context, branchID primitive.ObjectID) (fiscal.Client, *models.FiscalSettings, error) {
	s := h.fiscalSettingsOf(ctx, branchID)
	if !s.Enabled || s.Provider == "" {
		return nil, s, nil
	}
	c, err := fiscal.New(s.Provider, credsOf(s))
	return c, s, err
}

// AdminFiscalProviders lists the virtual cash registers an owner can pick.
//
// Its own endpoint rather than a constant in the panel, for the reason the POS
// list is: the set changes with what we have built, and a hard-coded copy in
// the frontend goes stale in the direction that matters — offering a provider
// the server cannot dial.
func (h *Handler) AdminFiscalProviders(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, fiscal.Providers())
}

// AdminGetFiscal returns the branch's settings, without the secrets.
func (h *Handler) AdminGetFiscal(w http.ResponseWriter, r *http.Request) {
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	s := h.fiscalSettingsOf(r.Context(), branchID)
	httpx.JSON(w, http.StatusOK, fiscalResponse(s))
}

// fiscalResponse is what the panel is allowed to see.
//
// ⚠️ A **separate, narrow struct** rather than the model with `json:"-"` doing
// the work. The model already hides the secrets, but this response is read by a
// page anybody with a manager login can open, and the next field added to
// FiscalSettings would otherwise appear here by default. The rule that has
// caught real leaks in this codebase is that the public shape is written out by
// hand — see publicReview.
type fiscalFlags struct {
	Provider string `json:"provider"`
	Enabled  bool   `json:"enabled"`
	TIN      string `json:"tin"`
	// Null when the branch has not declared a rate. The panel must be able to
	// tell that from zero — see FiscalSettings.VatPercent.
	VatPercent *int `json:"vatPercent"`
	// Per provider: is a secret stored, and what non-secret settings are there.
	Creds map[string]fiscalCredFlags `json:"creds"`

	LastCheckAt   *time.Time `json:"lastCheckAt,omitempty"`
	LastCheckOK   bool       `json:"lastCheckOk"`
	LastCheck     string     `json:"lastCheck"`
	LastReceiptAt *time.Time `json:"lastReceiptAt,omitempty"`
	LastErrorAt   *time.Time `json:"lastErrorAt,omitempty"`
	LastError     string     `json:"lastError"`
}

type fiscalCredFlags struct {
	Login      string `json:"login"`
	RegisterID string `json:"registerId"`
	BaseURL    string `json:"baseUrl"`
	HasSecret  bool   `json:"hasSecret"`
}

func fiscalResponse(s *models.FiscalSettings) fiscalFlags {
	out := fiscalFlags{
		Provider:      s.Provider,
		Enabled:       s.Enabled,
		TIN:           s.TIN,
		VatPercent:    s.VatPercent,
		Creds:         map[string]fiscalCredFlags{},
		LastCheckAt:   s.LastCheckAt,
		LastCheckOK:   s.LastCheckOK,
		LastCheck:     s.LastCheck,
		LastReceiptAt: s.LastReceiptAt,
		LastErrorAt:   s.LastErrorAt,
		LastError:     s.LastError,
	}
	for id, c := range drawers(s) {
		out.Creds[id] = fiscalCredFlags{
			Login:      c.Login,
			RegisterID: c.RegisterID,
			BaseURL:    c.BaseURL,
			HasSecret:  c.HasSecret(),
		}
	}
	return out
}

type fiscalUpdateRequest struct {
	Provider   string `json:"provider"`
	Enabled    bool   `json:"enabled"`
	TIN        string `json:"tin"`
	VatPercent *int   `json:"vatPercent"`

	// One drawer per provider, keyed by id. ⚠️ A map rather than a field each,
	// for the reason `drawers` gives: a provider missed from one of four lists
	// saves nothing and says nothing.
	Creds map[string]models.FiscalCreds `json:"creds"`

	// ---- The shape the panel sent before that ----
	//
	// ⚠️ **Frozen. A new provider does NOT go here.** These exist only for the
	// minutes after a deploy when a browser tab is still running the previous
	// panel: without them that tab's next save would write empty drawers over
	// working credentials, silently, and the restaurant would stop filing
	// receipts without a single error anywhere. Anything added below would be
	// dead the day it was written.
	Multikassa models.FiscalCreds `json:"multikassa"`
	FirstOFD   models.FiscalCreds `json:"firstofd"`
	EPOS       models.FiscalCreds `json:"epos"`
	Regos      models.FiscalCreds `json:"regos"`
	Hippo      models.FiscalCreds `json:"hippo"`
	Simurg     models.FiscalCreds `json:"simurg"`
}

// sent is the drawer this request carries for each provider.
func (r fiscalUpdateRequest) sent() map[string]models.FiscalCreds {
	out := map[string]models.FiscalCreds{
		fiscal.Multikassa: r.Multikassa,
		fiscal.FirstOFD:   r.FirstOFD,
		fiscal.EPOS:       r.EPOS,
		fiscal.Regos:      r.Regos,
		fiscal.Hippo:      r.Hippo,
		fiscal.Simurg:     r.Simurg,
		fiscal.Rahmat:     {},
		fiscal.QPOS:       {},
		fiscal.Arca:       {},
	}
	// The current panel's shape wins wherever it said anything.
	for id, c := range r.Creds {
		if fiscal.Known(id) {
			out[id] = c
		}
	}
	return out
}

// AdminUpdateFiscal saves the branch's connection.
//
// Owner only. A branch manager runs a kitchen; this decides which taxpayer the
// restaurant's sales are filed under, and getting it wrong is not a mistake a
// shift can absorb.
func (h *Handler) AdminUpdateFiscal(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req fiscalUpdateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Provider != "" && !fiscal.Known(req.Provider) {
		httpx.Error(w, http.StatusBadRequest, "noma'lum fiskal provayder")
		return
	}
	if req.VatPercent != nil && (*req.VatPercent < 0 || *req.VatPercent > 100) {
		httpx.Error(w, http.StatusBadRequest, "QQS stavkasi 0 dan 100 gacha bo'lishi kerak")
		return
	}
	if err := fiscalEnableRefusal(req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	current := h.fiscalSettingsOf(r.Context(), branchID)
	set := bson.M{
		"branchId":  branchID,
		"provider":  req.Provider,
		"enabled":   req.Enabled,
		"tin":       req.TIN,
		"updatedAt": time.Now(),
	}
	// ⚠️ **Every drawer, from the one list.** This was six lines naming six
	// providers, and the seventh provider added to the panel would have been
	// saveable, selectable, and stored nowhere — the field simply absent from
	// the `$set`. The bson keys are the provider ids, which is what the model's
	// tags already say.
	stored := drawers(current)
	for id, sent := range req.sent() {
		set[id] = mergeCreds(sent, stored[id])
	}
	if req.VatPercent != nil {
		set["vatPercent"] = *req.VatPercent
	}

	// ⚠️ Changing the connection invalidates the last check, exactly as editing
	// the SMS template clears `lastTestOk`: a green tick sitting above
	// credentials the provider has never seen says "verified", and the owner
	// finds out it was lying at the counter with a guest waiting.
	if req.Provider != current.Provider || req.TIN != current.TIN {
		set["lastCheckOk"] = false
		set["lastCheck"] = ""
		set["lastCheckAt"] = nil
	}

	if _, err := h.Store.FiscalSettings.UpdateOne(r.Context(),
		bson.M{"branchId": branchID},
		bson.M{"$set": set},
		options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActSettingsUpdate, "fiscal", branchID.Hex(), fiscal.Name(req.Provider), "")
	httpx.JSON(w, http.StatusOK, fiscalResponse(h.fiscalSettingsOf(r.Context(), branchID)))
}

// fiscalEnableRefusal says why fiscalisation cannot be switched on.
//
// Saving a half-filled form is always allowed — an owner sets this up over
// several days, between a contract and a tax office visit — but *enabling* it
// is a claim that sales are being registered, and a claim that turns out to be
// false is discovered by an inspector rather than by us.
//
// A pure function so the reasons can be tested without a database, and so the
// panel and the server cannot drift on what "ready" means.
func fiscalEnableRefusal(req fiscalUpdateRequest) error {
	if !req.Enabled {
		return nil
	}
	if req.Provider == "" {
		return errors.New("fiskal provayder tanlanmagan")
	}
	if !fiscal.Ready(req.Provider) {
		// Ours to fix, not theirs — so the message does not send an owner back
		// to re-check credentials that were never wrong.
		return errors.New(fiscal.Name(req.Provider) +
			" bilan ulanish hali tayyor emas — provayder API hujjati kutilmoqda")
	}
	if req.TIN == "" {
		return errors.New("СТИР (ИНН) kiritilmagan")
	}
	if req.VatPercent == nil {
		// See FiscalSettings.VatPercent: zero is a real answer here, so it has
		// to be given rather than assumed. An unset rate defaulting to 0 would
		// declare every restaurant exempt.
		return errors.New("QQS stavkasi ko'rsatilmagan (QQS to'lovchisi bo'lmasangiz 0 kiriting)")
	}
	// ⚠️ A register on the restaurant's own network is reached by address and
	// nothing else — there is no account to fall back on and no default host to
	// try. Enabling without one switches on a claim that sales are being
	// registered and then files nothing, which is the exact failure this whole
	// function exists to refuse.
	if fiscal.IsLocal(req.Provider) && strings.TrimSpace(credsFor(req).BaseURL) == "" {
		return errors.New("kassa dasturining manzili kiritilmagan " +
			"(masalan http://192.168.1.50:9090)")
	}
	return nil
}

// credsFor picks the drawer belonging to the provider being saved.
//
// The read-side twin of credsOf, and separate from it only because one takes
// the stored document and the other the incoming form. Both exist so the
// picking happens in one place per direction — see the note on credsOf.
func credsFor(req fiscalUpdateRequest) models.FiscalCreds {
	return req.sent()[req.Provider]
}

// mergeCreds applies "empty secret means keep the stored one".
//
// The non-secret fields are taken as sent, because they are visible in the form
// and an owner clearing one means it. The secrets are the opposite: the form
// cannot show them, so it sends them empty on every save that did not retype
// them — and treating that as "delete" would unfiscalise a restaurant whose
// owner only came in to fix the TIN.
func mergeCreds(incoming, stored models.FiscalCreds) models.FiscalCreds {
	incoming.Password = keepSecret(incoming.Password, stored.Password)
	incoming.Token = keepSecret(incoming.Token, stored.Token)
	return incoming
}

// AdminFiscalPing asks the provider what we are connected to.
//
// ⚠️ Reports **what**, not just "ok" — the POS package's lesson: a bare
// "connected" cannot tell the right cash register from somebody else's, and
// this is the one screen where that distinction is the entire point.
func (h *Handler) AdminFiscalPing(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	s := h.fiscalSettingsOf(r.Context(), branchID)
	if s.Provider == "" {
		httpx.Error(w, http.StatusBadRequest, "fiskal provayder tanlanmagan")
		return
	}
	// ⚠️ A register on the restaurant's own network cannot be reached from this
	// server, and saying so plainly matters more here than anywhere else: this
	// page is very often open on a laptop somewhere else entirely, where a
	// failed check would be perfectly true and mean nothing at all. So the
	// button does not pretend to try — it says which screen owns the question.
	if fiscal.IsLocal(s.Provider) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok": false, "local": true,
			"message": httpx.T(w, fiscal.Name(s.Provider)+" restoran ichidagi tarmoqda ishlaydi — "+
				"ulanishni kassa ekranidan (/kassa) tekshiring."),
		})
		return
	}
	client, err := fiscal.New(s.Provider, credsOf(s))
	var msg string
	if err == nil {
		msg, err = client.Ping(r.Context())
	}

	now := time.Now()
	set := bson.M{"lastCheckAt": now, "lastCheckOk": err == nil}
	if err != nil {
		set["lastCheck"] = err.Error()
	} else {
		set["lastCheck"] = msg
	}
	_, _ = h.Store.FiscalSettings.UpdateOne(r.Context(),
		bson.M{"branchId": branchID}, bson.M{"$set": set},
		options.Update().SetUpsert(true))

	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": false, "message": httpx.T(w, err.Error())})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "message": httpx.T(w, msg)})
}
