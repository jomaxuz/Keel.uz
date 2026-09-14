package handlers

// Closing a customer's account for good.
//
// Until now "o'chirish" was one thing and it was reversible: the status went to
// `deleted`, the container stopped, the site went dark — and the database, the
// photographs and the container were all still there. That is the right default
// (a customer who leaves in March often comes back in May, and the one thing they
// cannot recreate is their own data), but it is **not** what the word promises,
// and a console that says "deleted" while keeping every order, address and phone
// number is a console that quietly turns us into a place data goes to stay. When
// the customer is the one asking, "we stopped billing you" is not an answer.
//
// So there are two, and they are named for what they do rather than for how final
// they feel:
//
//   - **Vaqtincha o'chirish** — the existing status change. Reversible, keeps
//     everything, and is what almost every departure should be.
//   - **To'liq o'chirish** — this file. The container, the database and the
//     photographs are gone afterwards, and nothing here can bring them back.
//
// ⚠️ **What is deliberately kept: our own books.** Invoices and the nightly
// `tenant_day` rows are not the customer's data, they are the record of what we
// charged and collected — deleting them to be thorough would mean answering "what
// did this customer pay in March?" with nothing, which is a worse outcome than
// keeping a row with a name on it. The tenant document survives for the same
// reason, marked purged: it is what the invoices point at, and it is what stops
// the slug being handed to somebody else.
//
// The guards below are the whole point of the file. Each one exists because of a
// specific way this goes wrong, and none of them is a confirmation dialog.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"keel-control/internal/httpx"
	"keel-control/internal/models"
	"keel-control/internal/sysstat"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type purgeRequest struct {
	// The slug, typed out. ⚠️ Not a checkbox and not the customer's *name*: the
	// name is what an operator is looking at when they pick the wrong row, and
	// two restaurants called "Osh Markazi" are ordinary. Typing the slug means
	// reading the identifier that actually decides which database is dropped.
	Confirm string `json:"confirm"`
	// Why. Required, and stored on the tenant as well as in the log — "who
	// deleted this?" is answerable from the log, but "why is this customer
	// gone?" is asked months later by somebody looking at the row.
	Reason string `json:"reason"`
}

// purgeStep is one thing that was attempted, and what came of it.
//
// ⚠️ Reported per step rather than as one success flag, because these fail
// independently and the fixes are completely different: a container that would
// not stop is a Docker problem, an undropped database is the one that still holds
// the customer's data, and "the files are gone but the row still says active" is
// a half-purge somebody has to finish by hand. One boolean would flatten all of
// that into "xato" — the same mistake as chaining independent work with `else if`.
type purgeStep struct {
	Step  string `json:"step"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// purgeRefusal is every reason this must not run, in one place.
//
// ⚠️ A function rather than four `if`s in the handler, and it has a test.
// These are the checks most likely to be "simplified" by somebody adding a
// fifth one, and the difference between a guard that exists and a guard that is
// exercised is the whole lesson of the nil-slice crash: the rule lives where a
// later edit cannot walk past it.
//
// Returns the HTTP status to refuse with and what to say, or 0 when the purge
// may proceed.
func purgeRefusal(status, slug, confirm, reason string, backupStale bool) (int, string) {
	// **Two steps, never one.** A live customer cannot be erased by a single
	// request, however well confirmed: switching them off first is reversible, it
	// makes the site go dark where somebody will notice, and it puts a night
	// between the decision and the irreversible half. Almost every wrong deletion
	// is a wrong *row*, and this is the guard that catches it while it still
	// costs nothing.
	if status != models.StatusSuspended && status != models.StatusDeleted {
		return http.StatusBadRequest,
			"avval mijozni vaqtincha o'chiring (to'xtatilgan yoki o'chirilgan holatda bo'lishi shart)"
	}
	if strings.TrimSpace(confirm) != slug {
		return http.StatusBadRequest, "tasdiqlash uchun mijozning slug'ini aynan yozing: " + slug
	}
	if strings.TrimSpace(reason) == "" {
		return http.StatusBadRequest, "sabab yozilishi shart"
	}
	// **No fresh backup, no purge.** The backup is the only thing standing
	// between "we closed the account they asked us to close" and "we destroyed a
	// business's records because a row was misread" — so the moment it is stale
	// is exactly the moment this button must not work. It is also the honest
	// reading of the cron failure found the same day: copies stopped for three
	// nights and every screen stayed green.
	//
	// A deliberate exception is the machine with no backups configured at all
	// (the caller passes false): a laptop is not where a real customer is
	// purged, and refusing there would make the feature untestable.
	if backupStale {
		return http.StatusConflict,
			"zaxira nusxa eskirgan — to'liq o'chirishdan oldin nusxa olinishi shart"
	}
	return 0, ""
}

// PurgeTenant erases a customer's data and infrastructure for good.
func (h *Handler) PurgeTenant(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	// ⚠️ Owner only, and not merely "provision". Everything else behind that
	// permission is recoverable — a wrongly restarted container comes back, a
	// wrongly suspended customer is one click from live again. This is the only
	// button in the console that destroys something no button can restore, and
	// the account that presses it should be the one that owns the consequences.
	if !actor.Has(models.RoleOwner) {
		fail(w, errForbidden)
		return
	}

	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var t models.Tenant
	if err := h.Store.Tenants.FindOne(r.Context(), bson.M{"_id": id}).Decode(&t); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}

	var req purgeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "so'rovni o'qib bo'lmadi")
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)

	b := sysstat.ReadBackup(h.Cfg.BackupPath)
	if code, msg := purgeRefusal(t.Status, t.Slug, req.Confirm, req.Reason, b.Present && b.Stale); code != 0 {
		httpx.Error(w, code, msg)
		return
	}

	steps := make([]purgeStep, 0, 4)
	record := func(name string, err error) {
		s := purgeStep{Step: name, OK: err == nil}
		if err != nil {
			s.Error = err.Error()
		}
		steps = append(steps, s)
	}

	// ⚠️ Not the request's context. Half of this is irreversible and none of it
	// is resumable: an operator closing the tab mid-way would leave a dropped
	// database with a running container in front of it, and no record of which
	// steps had run. The work is short and bounded; it finishes.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 3*time.Minute)
	defer cancel()

	// The container first: it is the only thing still holding the database open,
	// and dropping a database out from under a running server produces errors in
	// a log nobody will read afterwards.
	if h.Docker != nil {
		record("konteyner", h.Docker.Remove(ctx, t.Slug))
		record("rasmlar", h.Docker.PurgeUploads(ctx, t.Slug))
	}

	// The customer's own database, which is where every order, address and phone
	// number lives. This is the step the whole feature exists for.
	record("baza", h.Store.TenantDB(t.DBName()).Drop(ctx))

	now := time.Now()
	_, err = h.Store.Tenants.UpdateOne(ctx, bson.M{"_id": t.ID}, bson.M{"$set": bson.M{
		"status":      models.StatusDeleted,
		"purgedAt":    now,
		"purgedBy":    actor.Username,
		"purgeReason": req.Reason,
		"updatedAt":   now,
	}})
	record("yozuv", err)

	// The edge last, and unconditionally — the lesson from the vanished domain:
	// two things that can fail on their own are never chained, and a customer
	// whose container refused to stop must not also stop every *other* tenant's
	// domain changes from being written.
	record("chekka", h.syncEdge(ctx))

	h.logConsole(ctx, actor, "tenant.purge", t.Slug, req.Reason)

	ok := true
	for _, s := range steps {
		if !s.OK {
			ok = false
		}
	}
	// 200 even when a step failed, with the steps in the body. The work is done
	// and cannot be retried as a whole; what the operator needs is which half to
	// finish by hand, and an error status would hide the list that says so.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"purged": ok,
		"steps":  steps,
		"slug":   t.Slug,
	})
}
