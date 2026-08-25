package handlers

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/fiscal"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The relay: filing from registers no browser can reach.
//
// # Why this exists beside the browser route
//
// Filing from the till screen works only if three things line up, and we
// control none of them: the page must not be blocked as mixed content, it must
// be allowed to reach a private address, and the register must answer CORS so
// the reply can be read. The first two are solved by running the till on the
// register's own PC (`http://localhost:8080`, which is the deployment
// Multikassa's documentation assumes). The third is the register's choice.
//
// So there is a second route that depends on none of them: a small program on
// the register's PC that connects **outwards** to us, asks for filings, makes
// them against localhost, and posts back what it saw. An outbound connection is
// subject to no browser rule at all, and needs nothing opened on the
// restaurant's network — the same shape as every other agent that lives behind
// somebody's firewall.
//
// # ⚠️ There is no job queue, and that is deliberate
//
// The work is **derived, not stored**: an order whose `fiscal.status` is
// "pending" is a filing waiting to happen, and that field already existed for a
// different reason. A jobs collection would be a second record of the same
// fact, and the two would disagree the first time a container restarted
// mid-filing — leaving either a receipt filed twice or a sale filed never.
//
// It also makes the whole path restart-safe for free: nothing is held in
// memory, so a deploy in the middle of a shift loses no filing.
//
// # ⚠️ Which route is used is decided by a timestamp, never by a setting
//
// If an agent has asked for work in the last little while, filings go to it;
// otherwise the till tries the call itself. No toggle to get wrong, and it
// degrades in the right direction — an agent whose PC was rebooted for Windows
// updates simply stops being preferred, rather than silently swallowing every
// receipt until somebody notices a checkbox. Same reasoning as `attention:
// "down"` in the console: a stored flag goes stale the moment the hour moves
// past it.

// agentAlive is how recently the relay must have asked for work to be trusted
// with the next filing.
//
// Comfortably longer than its poll cycle, so one slow round trip does not flip
// the route back and forth mid-shift, and short enough that a machine switched
// off at closing time is not still "connected" the next morning.
const agentAlive = 90 * time.Second

// agentPollWait is how long a request for work is held open before answering
// "nothing".
//
// ⚠️ Long-polling rather than a socket, which is this codebase's standing
// choice (see AlertBell): it survives every proxy, needs no new dependency, and
// reconnects by simply asking again. Kept under the common 60-second idle
// timeout of intermediaries so the connection is closed by us rather than by
// something in between, which is the difference between "no work" and an error
// in the agent's log every minute.
const agentPollWait = 25 * time.Second

// newAgentToken mints a relay's credential.
//
// ⚠️ 32 bytes from crypto/rand, for the reason order numbers are: this string
// is the entire authentication for something that can read a branch's sales as
// they happen. Predictable is the same as public.
func newAgentToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// agentBranch identifies the caller from its token.
//
// ⚠️ Compared in **constant time**, and the lookup is deliberately not a Mongo
// query on the token: a database comparison is not constant time and this
// endpoint is open to the internet. The candidate set is one document per
// branch, so scanning it costs nothing and leaks nothing about how close a
// guess was.
//
// ⚠️ An empty stored token never matches. Every branch that has not set up a
// relay holds "" — and an empty header matching them all would hand the first
// scanner the sales of every restaurant that never enabled this.
func (h *Handler) agentBranch(r *http.Request) (*models.FiscalSettings, bool) {
	// ⚠️ **A paired till is also an agent, and it carries the device token it
	// already has.** The Windows till runs this same loop in-process, on the
	// machine with the printer, so it needs to authenticate here — but minting
	// it a relay token is destructive: AdminFiscalAgentToken rotates on every
	// call, so a till asking for one would silently kill whatever agent that
	// branch was already running. Two secrets for one machine, where fetching
	// the second revokes the first, is what made automatic pairing impossible.
	//
	// The device token is the better credential anyway: it is branch-scoped,
	// version-counted and revocable from the panel without touching anything
	// else (branch.TillVersion, handlers/tillpin.go).
	if set, ok := h.agentBranchByDevice(r); ok {
		return set, true
	}
	token := strings.TrimSpace(r.Header.Get("X-Agent-Token"))
	if token == "" {
		return nil, false
	}
	cur, err := h.Store.FiscalSettings.Find(r.Context(),
		bson.M{"agentToken": bson.M{"$nin": bson.A{"", nil}}})
	if err != nil {
		return nil, false
	}
	var rows []models.FiscalSettings
	if err := cur.All(r.Context(), &rows); err != nil {
		return nil, false
	}
	return matchAgentToken(rows, token)
}

// agentBranchByDevice authenticates a till monoblock by its device token.
//
// ⚠️ **The version is checked, exactly as tillDeviceBranch checks it.** Without
// it the revocation counter would be decoration here while working everywhere
// else — and the one machine it failed to lock out would be the stolen one,
// still holding a token that reaches the branch's print and filing queue.
//
// ⚠️ Parsed from the header rather than read from the request context: this
// route is deliberately outside the auth middleware, because its other caller
// (the standalone relay) authenticates with a shared secret and no JWT at all.
//
// A branch with no fiscal settings row still gets an answer, carrying only its
// id. The till is here for the print queue, which is branch-scoped and does not
// need a cash register to exist — refusing would mean no restaurant could print
// through the app until it had registered for fiscal filing.
func (h *Handler) agentBranchByDevice(r *http.Request) (*models.FiscalSettings, bool) {
	id, ver, ok := tillDeviceToken(h.Cfg.JWTSecret, r.Header.Get("Authorization"))
	if !ok {
		return nil, false
	}
	branch, err := h.branchByIDCtx(r.Context(), id)
	if err != nil || ver != branch.TillVersion {
		return nil, false
	}
	var set models.FiscalSettings
	if err := h.Store.FiscalSettings.FindOne(r.Context(),
		bson.M{"branchId": id}).Decode(&set); err != nil {
		return &models.FiscalSettings{BranchID: id}, true
	}
	return &set, true
}

// tillDeviceToken reads a device token out of an Authorization header.
//
// A pure function because both rules it enforces fail open in the direction
// nobody would notice:
//
//   - **The role must be "tilldevice".** Every screen in the building carries a
//     JWT signed with the same secret — a cashier's "till" token, a waiter's
//     "staff" token, a guest's "user" token. Accepting any valid signature here
//     would let a guest's phone drain the branch's print queue, and the check
//     that prevents it is one string comparison that reads like a formality.
//   - **The version comes back to the caller**, so the freshness check cannot
//     be quietly skipped by a future caller that only wanted the branch id.
func tillDeviceToken(secret, header string) (primitive.ObjectID, int, bool) {
	raw := strings.TrimSpace(header)
	if !strings.HasPrefix(raw, "Bearer ") {
		return primitive.NilObjectID, 0, false
	}
	claims, err := auth.Parse(secret, strings.TrimSpace(strings.TrimPrefix(raw, "Bearer ")))
	if err != nil || claims.Role != "tilldevice" {
		return primitive.NilObjectID, 0, false
	}
	id, err := objectID(claims.UserID)
	if err != nil {
		return primitive.NilObjectID, 0, false
	}
	return id, claims.Ver, true
}

// matchAgentToken picks the branch whose token this is.
//
// Its own function so the two rules that matter can be tested without a
// database, because both fail open in the worst possible direction:
//
//   - **An empty presented token never matches.** Callers already guard this,
//     but a guard in one place is a guard that the next caller forgets.
//   - **An empty stored token never matches either.** Every branch that has not
//     set up a relay holds "", so a single missed check would hand the first
//     scanner with a blank header the live sales of every restaurant that never
//     enabled this.
func matchAgentToken(rows []models.FiscalSettings, token string) (*models.FiscalSettings, bool) {
	if token == "" {
		return nil, false
	}
	for i := range rows {
		if rows[i].AgentToken == "" {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(rows[i].AgentToken), []byte(token)) == 1 {
			return &rows[i], true
		}
	}
	return nil, false
}

// agentUsable reports whether the relay has been heard from recently enough to
// be given the next filing.
func agentUsable(s *models.FiscalSettings) bool {
	return s != nil && s.AgentSeenAt != nil &&
		time.Since(*s.AgentSeenAt) < agentAlive
}

// What kind of work the relay has been handed.
//
// ⚠️ Named rather than inferred from whether an order id is present. The two
// have different consequences — one files a receipt, the other closes a tax day
// — and a relay guessing from a missing field would guess wrong exactly once.
const (
	jobFiling   = "filing"
	jobCloseDay = "closeDay"
	jobPrint    = "print"
)

type agentJobResponse struct {
	Kind string `json:"kind"`
	// A print job: the bytes, and the printer to write them to.
	Print *agentPrintJob `json:"print,omitempty"`
	// Which sale this filing belongs to, for a filing. The agent quotes it
	// back; it is not a secret and not a capability — the token is.
	OrderID string        `json:"orderId,omitempty"`
	Number  string        `json:"number,omitempty"`
	Job     tillFiscalJob `json:"job"`
}

// agentPrintJob is one receipt for one printer.
//
// ⚠️ **Opaque bytes and an address, nothing else.** The layout, the code page
// and the cut are decided on the server; the agent opens a socket or a file
// handle and writes. Anything cleverer here is business logic on an unattended
// PC in a restaurant.
type agentPrintJob struct {
	ID string `json:"id"`
	// tcp://192.168.1.50:9100 · usb://XP-58 · serial://COM3 · /dev/usb/lp0
	Target string `json:"target"`
	Name   string `json:"name,omitempty"`
	// base64, because this is JSON and ESC/POS is not text.
	Payload string `json:"payload"`
}

// encFor builds a branch's encoder, or nil. A small helper because the job loop
// needs it twice and a failure there is not worth an error path: a branch whose
// provider cannot be built has no work either way.
func encFor(set *models.FiscalSettings) fiscal.Encoder {
	enc, err := fiscal.EncoderFor(set.Provider, credsOf(set))
	if err != nil {
		return nil
	}
	return enc
}

// FiscalAgentJob hands the relay the next filing, waiting if there is none.
//
// ⚠️ **Answering "nothing" is the normal case and must stay cheap.** A restaurant
// files a few dozen receipts a day and this endpoint is asked continuously, so
// it holds the request open and polls the database slowly rather than tightly:
// the alternative is one query per second per branch, forever, against the
// Mongo every tenant shares.
func (h *Handler) FiscalAgentJob(w http.ResponseWriter, r *http.Request) {
	set, ok := h.agentBranch(r)
	if !ok {
		// ⚠️ 401 here, unlike the webhooks, and the difference is who is asking.
		// A webhook's caller is a stranger's server and a refusal tells a
		// scanner it found something; this caller is a program we shipped, with
		// nobody watching its screen, and it has to be able to say "my token is
		// wrong" in its own log rather than retrying forever in silence.
		httpx.Error(w, http.StatusUnauthorized, "noma'lum agent kaliti")
		return
	}
	h.markAgentSeen(r.Context(), set)

	deadline := time.Now().Add(agentPollWait)
	for {
		// ⚠️ **Printing goes first.** A kitchen ticket is a plate nobody has
		// started cooking and its guest is sitting at a table; a filing is a
		// document the state will accept a minute later and which is retried by
		// being asked for again. Ordering them the other way would hold a
		// dinner behind paperwork.
		if pj, err := h.nextPrintJob(r.Context(), set.BranchID); err == nil && pj != nil {
			httpx.JSON(w, http.StatusOK, agentJobResponse{
				Kind:    jobPrint,
				OrderID: pj.OrderID.Hex(),
				Number:  pj.Number,
				Print: &agentPrintJob{
					ID:      pj.ID.Hex(),
					Target:  pj.Target,
					Name:    pj.PrinterName,
					Payload: base64.StdEncoding.EncodeToString(pj.Payload),
				},
			})
			return
		}
		o, err := h.nextPendingFiling(r.Context(), set.BranchID)
		if err == nil && o != nil {
			enc, jerr := fiscal.EncoderFor(set.Provider, credsOf(set))
			if jerr != nil || enc == nil {
				httpx.Error(w, http.StatusServiceUnavailable, "fiskal provayder sozlanmagan")
				return
			}
			req, berr := enc.Sale(fiscal.Build(
				receiptFor(o, set, agentCashier(o), h.menuFiscal(r.Context(), o))))
			if berr != nil {
				// The document cannot be built — a check with nothing sellable
				// on it. Failing it here rather than handing the agent something
				// it will never manage stops it retrying the same sale forever.
				_, _ = h.recordFiling(r.Context(), o, set, enc, fiscalReplyRequest{
					NetworkError: berr.Error(),
				})
				continue
			}
			httpx.JSON(w, http.StatusOK, agentJobResponse{
				Kind:    jobFiling,
				OrderID: o.ID.Hex(),
				Number:  o.Number,
				Job:     tillFiscalJob{}.withBase(agentBase(set), req),
			})
			return
		}
		// ⚠️ **Only once there is nothing left to file.** A Z-report totals the
		// day and hands that total to the state, so it must never overtake a
		// receipt still waiting — which is exactly what checking it second, and
		// only when the loop above found nothing, guarantees. The ordering is
		// the guard; there is no separate check to forget.
		if job, has := closeDayJob(set, encFor(set)); has {
			httpx.JSON(w, http.StatusOK, agentJobResponse{
				Kind: jobCloseDay,
				Job:  job,
			})
			return
		}

		if time.Now().After(deadline) || r.Context().Err() != nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
}

// agentBase is the address the relay dials.
//
// ⚠️ Falls back to localhost, because that is where the relay is: it runs on the
// register's own PC, which is the entire reason it exists. An owner who left the
// address blank meant "the machine this is installed on", and refusing to file
// over an empty field would be refusing the only setup the relay has.
func agentBase(s *models.FiscalSettings) string {
	if b := strings.TrimRight(strings.TrimSpace(credsOf(s).BaseURL), "/"); b != "" {
		return b
	}
	return "http://localhost:8080"
}

// agentCashier names who took the money, for the receipt.
//
// The relay has no person behind it, so the name comes off the check — which is
// where it belongs anyway: the receipt should say who served the guest, not
// which program submitted the paperwork.
func agentCashier(o *models.Order) string {
	if o.Check != nil && o.Check.ClosedBy != "" {
		return o.Check.ClosedBy
	}
	return ""
}

// nextPendingFiling finds the oldest sale waiting to be registered.
//
// ⚠️ **Oldest first**, as the kitchen screen orders its tickets and for the same
// reason: the receipt that has been waiting longest is the one closest to being
// a problem, and any other order leaves it waiting indefinitely under a steady
// trickle of newer ones.
func (h *Handler) nextPendingFiling(
	ctx context.Context, branchID any,
) (*models.Order, error) {
	var o models.Order
	err := h.Store.Orders.FindOne(ctx, bson.M{
		"branchId":      branchID,
		"check":         bson.M{"$exists": true},
		"fiscal.status": models.FiscalPending,
	}, options.FindOne().SetSort(bson.M{"check.closedAt": 1})).Decode(&o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// markAgentSeen records that the relay is alive.
//
// Written on every request for work, which is what makes the route decision a
// live fact rather than a setting somebody has to remember to turn off.
func (h *Handler) markAgentSeen(ctx context.Context, s *models.FiscalSettings) {
	now := time.Now()
	s.AgentSeenAt = &now
	_, _ = h.Store.FiscalSettings.UpdateOne(ctx,
		bson.M{"branchId": s.BranchID},
		bson.M{"$set": bson.M{"agentSeenAt": now}})
}

// FiscalAgentResult records what the register told the relay.
// FiscalAgentPrintResult records what a printer did with one job.
//
// ⚠️ **A failure is kept, not retried into silence.** A ticket that cannot be
// printed is food nobody is making, and the useful thing is that somebody is
// told — the queue gives up after three tries and the reason stays on the job.
func (h *Handler) FiscalAgentPrintResult(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.agentBranch(r); !ok {
		httpx.Error(w, http.StatusUnauthorized, "noma'lum agent kaliti")
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Error string `json:"error"`
	}
	// ⚠️ **The most expensive place this idiom could have been.** A body skipped
	// because the length header said -1 leaves `error` empty — and an empty
	// error is how this endpoint spells *success*, so a printer that refused
	// the job would be recorded as having printed it. The queue would go quiet,
	// the panel would show the work finished, and the kitchen would be waiting
	// for a ticket nothing was still trying to send.
	if err := httpx.DecodeOptional(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.finishPrintJob(r.Context(), id, req.Error); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) FiscalAgentResult(w http.ResponseWriter, r *http.Request) {
	set, ok := h.agentBranch(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "noma'lum agent kaliti")
		return
	}
	h.markAgentSeen(r.Context(), set)

	var req fiscalReplyRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "buyurtma topilmadi")
		return
	}

	var o models.Order
	// ⚠️ The branch is in the filter, not merely checked afterwards — the same
	// rule the till and the kitchen screen follow. A relay in one restaurant
	// must not be able to name another one's order id and read its sale back.
	if err := h.Store.Orders.FindOne(r.Context(),
		checkFilter(id, set.BranchID)).Decode(&o); err != nil {
		httpx.Error(w, http.StatusNotFound, "buyurtma topilmadi")
		return
	}
	enc, err := fiscal.EncoderFor(set.Provider, credsOf(set))
	if err != nil || enc == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "fiskal provayder sozlanmagan")
		return
	}
	// ⚠️ The morning case, and the one that would otherwise spin forever: the
	// register refuses because its day has not been opened, the sale stays
	// pending, the relay asks for work, gets the same sale, and is refused
	// again — a tight loop that files nothing and fills a log.
	//
	// So the answer to "that failed" can be "do this first". The sale is left
	// pending on purpose: nothing was wrong with the receipt, and recording a
	// failure would put every restaurant's first sale of the day into the
	// unfiled alert every morning.
	if job, ok := h.shiftJobFor(set, enc, req, agentCashier(&o)); ok {
		httpx.JSON(w, http.StatusOK, map[string]any{"next": job})
		return
	}

	rec, err := h.recordFiling(r.Context(), &o, set, enc, req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, rec)
}

// FiscalAgentCloseDay records the Z-report the relay obtained.
//
// ⚠️ Its own endpoint rather than the filing one with a flag. The two write to
// different documents and mean different things — one is a receipt, the other
// ends a tax day — and a single endpoint branching on a field is one wrong
// branch away from recording a Z-report as a sale's outcome.
func (h *Handler) FiscalAgentCloseDay(w http.ResponseWriter, r *http.Request) {
	set, ok := h.agentBranch(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "noma'lum agent kaliti")
		return
	}
	h.markAgentSeen(r.Context(), set)

	var req fiscalReplyRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	enc := encFor(set)
	if enc == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "fiskal provayder sozlanmagan")
		return
	}
	httpx.JSON(w, http.StatusOK, h.recordCloseDay(r.Context(), set, enc, req))
}

// AdminFiscalAgentToken mints or rotates the relay's credential.
//
// ⚠️ **Returned exactly once, at this moment.** It is stored to compare against
// and never sent again, which is what makes "rotate" a real revocation rather
// than a second working key. The panel says so beside the button, because a
// token shown once and not copied is a reinstall.
func (h *Handler) AdminFiscalAgentToken(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	token := newAgentToken()
	if token == "" {
		httpx.Error(w, http.StatusInternalServerError, "kalit yaratib bo'lmadi")
		return
	}
	if _, err := h.Store.FiscalSettings.UpdateOne(r.Context(),
		bson.M{"branchId": branchID},
		// ⚠️ The old "last seen" goes with the old token. Leaving it would make
		// a freshly rotated relay look connected before it has ever run, which
		// is precisely the moment somebody needs to know whether the new token
		// reached the machine.
		bson.M{"$set": bson.M{"agentToken": token, "branchId": branchID},
			"$unset": bson.M{"agentSeenAt": ""}},
		options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActSettingsUpdate, "fiscal", branchID.Hex(), "agent token", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"token": token})
}
