package handlers

// ---- Where the restaurant's money actually is ----
//
// ⚠️ **Three kinds of having, and they must never be added into one number.**
// Cash in a box can be spent tonight. Money in the bank account can be spent
// this week. Money an aggregator is still holding can be spent when somebody
// else decides. A single "we have X" would be the most quotable and least true
// figure on the platform — and it is precisely the number an owner would take
// to a bank or a landlord.
//
// ⚠️ **Every figure says whether it was counted or added up**, because they go
// wrong in opposite directions: a counted one (a drawer at close, a balance read
// off a bank app) goes stale, and a summed one (the safe's ledger, a rail's
// unsettled sales) goes wrong when a document is missing. An owner reading a
// number needs to know which failure to look for.

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// AdminMoney is the whole position: cash, bank, and what is still on the way.
func (h *Handler) AdminMoney(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx := r.Context()
	pos := models.MoneyPosition{
		Cash: []models.MoneyPlace{}, Bank: []models.MoneyPlace{},
		Rails: []models.MoneyPlace{},
	}

	// ---- Cash ----
	if bal, err := h.safeBalance(ctx, scopeFilter(scope)); err == nil {
		pos.Cash = append(pos.Cash, models.MoneyPlace{
			Kind: "safe", Name: "Seyf", Amount: bal.Balance, At: bal.LastAt,
		})
	}
	pos.Cash = append(pos.Cash, h.drawerCash(r, scope)...)
	if c := h.courierCash(r, scope); c.Amount != 0 {
		pos.Cash = append(pos.Cash, c)
	}
	if a := h.advanceCash(r, scope); a.Amount != 0 {
		pos.Cash = append(pos.Cash, a)
	}

	// ---- Bank ----
	pos.Bank = h.bankPlaces(r, scope)

	// ---- Still with a provider ----
	//
	// ⚠️ Reuses the payouts screen's arithmetic rather than repeating it: two
	// answers to "what does Uzum owe us" is one answer too many, and the one on
	// the wrong screen is the one somebody quotes.
	rows := []models.Payout{}
	if cur, err := h.Store.Payouts.Find(ctx, scopeFilter(scope)); err == nil {
		_ = cur.All(ctx, &rows)
	}
	for _, b := range h.payoutBalances(r, scope, rows) {
		if b.Sold == 0 {
			continue
		}
		note := ""
		if b.SettledThrough != "" {
			note = b.SettledThrough
		}
		pos.Rails = append(pos.Rails, models.MoneyPlace{
			Kind: "rail", Name: b.Name, Amount: b.Sold, Note: note,
		})
	}

	for _, p := range pos.Cash {
		pos.CashTotal += p.Amount
	}
	for _, p := range pos.Bank {
		pos.BankTotal += p.Amount
	}
	for _, p := range pos.Rails {
		pos.RailsTotal += p.Amount
	}

	// ⚠️ **The legal ceiling, and the reason this screen has one at all.** Cash
	// above the limit agreed with the bank must be handed over for crediting
	// (Правила ведения кассовых операций, ст. 7) — so a safe that is merely
	// "quite full" is, past a number, a breach with a date on it. Only checked
	// when a limit has been entered: a zero means nobody has told us the bank's
	// figure, and inventing one would be worse than silence.
	limit, limitBranch := h.branchCashLimit(r, scope)
	pos.CashLimit = limit
	// ⚠️ Which branch the ceiling belongs to travels with it: a single-branch
	// restaurant never picks a lens, and the screen still has to know where to
	// save the bank's figure.
	pos.BranchID = limitBranch

	pos.OverLimit = pos.CashLimit > 0 && pos.CashTotal > pos.CashLimit

	httpx.JSON(w, http.StatusOK, pos)
}

// drawerCash is what the open tills are holding right now.
//
// ⚠️ **Open shifts only.** A closed one has been counted and handed on; adding
// it would count last night's takings again this morning.
func (h *Handler) drawerCash(r *http.Request, scope bson.M) []models.MoneyPlace {
	filter := scopeFilter(scope)
	filter["closedAt"] = bson.M{"$exists": false}
	cur, err := h.Store.CashShifts.Find(r.Context(), filter)
	if err != nil {
		return nil
	}
	var shifts []models.CashShift
	if err := cur.All(r.Context(), &shifts); err != nil {
		return nil
	}
	out := []models.MoneyPlace{}
	for i := range shifts {
		s := shifts[i]
		figures, _, err := h.shiftFigures(r, &s)
		if err != nil {
			continue
		}
		at := s.OpenedAt
		out = append(out, models.MoneyPlace{
			Kind: "drawer", Name: "Kassa yashigi", Amount: figures.Expected,
			Note: s.OpenedBy, At: &at,
		})
	}
	return out
}

// courierCash is money collected on our behalf and not yet handed back.
//
// ⚠️ **Summed across every delivery, not the last two hundred.** The courier
// card samples recent orders because it draws a list; a figure about money owed
// cannot sample — a window that drops old deliveries while keeping every
// handover makes the debt shrink on its own.
func (h *Handler) courierCash(r *http.Request, scope bson.M) models.MoneyPlace {
	collected := scopeFilter(scope)
	collected["paymentMethod"] = models.ProviderCash
	collected["status"] = models.StatusDelivered
	collected["courierId"] = bson.M{"$exists": true, "$ne": nil}
	got, _, _ := h.sumField(r.Context(), h.Store.Orders, collected, "$total")

	handed, _, _ := h.sumField(r.Context(), h.Store.Settlements, bson.M{}, "$amount")

	return models.MoneyPlace{
		Kind: "courier", Name: "Kuryerlar qo'lida", Amount: got - handed,
	}
}

// advanceCash is the petty cash the buyers are holding.
func (h *Handler) advanceCash(r *http.Request, scope bson.M) models.MoneyPlace {
	branch, _ := scope["branchId"].(primitive.ObjectID)
	rows, err := h.advanceBalances(r.Context(), branch, primitiveNil)
	if err != nil {
		return models.MoneyPlace{Kind: "advance", Name: "Podotchet"}
	}
	sum := 0
	for _, b := range rows {
		sum += b.Balance
	}
	return models.MoneyPlace{Kind: "advance", Name: "Podotchet (bozorchilarda)", Amount: sum}
}

// bankPlaces is the last counted balance of each account, and what has arrived
// since.
//
// ⚠️ **A counted figure with a date, never a derived balance.** Money reaches
// that account from rails this system records and from a dozen it does not — an
// owner's own deposit, a loan, a transfer between the company's own accounts.
// A balance built from the movements we happen to see would be wrong by
// everything we cannot see, and wrong in a way that looks exactly like a
// balance.
func (h *Handler) bankPlaces(r *http.Request, scope bson.M) []models.MoneyPlace {
	cur, err := h.Store.BankBalances.Find(r.Context(), scopeFilter(scope),
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(100))
	if err != nil {
		return []models.MoneyPlace{}
	}
	var rows []models.BankBalance
	if err := cur.All(r.Context(), &rows); err != nil {
		return []models.MoneyPlace{}
	}
	out := []models.MoneyPlace{}
	seen := map[string]bool{}
	for i := range rows {
		b := rows[i]
		// Newest first, so the first row for an account is its latest count.
		if seen[b.Account] {
			continue
		}
		seen[b.Account] = true
		at := b.At
		out = append(out, models.MoneyPlace{
			Kind: "bank", Name: b.Account, Amount: b.Amount,
			Counted: true, At: &at, Note: b.Note,
		})
	}
	return out
}

// branchCashLimit is the ceiling the bank agreed with this branch.
// ⚠️ A single-branch restaurant resolves its own branch, the way the stock
// screens do: the legal ceiling is not a thing to make somebody pick a lens for.
func (h *Handler) branchCashLimit(
	r *http.Request, scope bson.M,
) (int, primitive.ObjectID) {
	id, ok := scope["branchId"].(primitive.ObjectID)
	if !ok || id.IsZero() {
		only, err := h.onlyBranch(r, primitive.NilObjectID)
		if err != nil {
			return 0, primitive.NilObjectID
		}
		id = only
	}
	var b models.Branch
	if err := h.Store.Branches.FindOne(r.Context(), bson.M{"_id": id}).Decode(&b); err != nil {
		return 0, id
	}
	return b.CashLimit, id
}

// AdminSaveBankBalance records a counted bank balance.
func (h *Handler) AdminSaveBankBalance(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		Account string `json:"account"`
		Amount  int    `json:"amount"`
		At      string `json:"at"`
		Note    string `json:"note"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	account := clampText(strings.TrimSpace(req.Account), 80)
	if account == "" {
		httpx.Error(w, http.StatusBadRequest, "hisob nomini yozing")
		return
	}
	branch := h.scopeBranch(r, scope)
	if err := h.requireBranchAccess(r, branch); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	now := time.Now()
	at := now
	if day, err := parseDay(req.At); err == nil {
		at = day
	}
	if at.After(now) {
		at = now
	}
	// ⚠️ **A new row every time, never an update in place.** The point of this
	// figure is that it was true on a date; overwriting the last one would
	// throw away the only evidence of when the account was last actually
	// looked at.
	b := models.BankBalance{
		BranchID: branch, Account: account, Amount: req.Amount, At: at,
		Note: clampText(strings.TrimSpace(req.Note), 200),
		By:   h.adminName(r), CreatedAt: now,
	}
	if _, err := h.Store.BankBalances.InsertOne(r.Context(), b); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, "bank.balance", "bank", account, account, formatSum(b.Amount))
	httpx.JSON(w, http.StatusCreated, b)
}

// AdminSetCashLimit stores the ceiling the bank agreed with a branch.
//
// ⚠️ Its own endpoint rather than a field on the branch form: the branch form
// replaces the whole document and may have been open for an hour, and this
// number decides whether a warning about a legal deadline appears.
func (h *Handler) AdminSetCashLimit(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.requireBranchAccess(r, id); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		CashLimit int `json:"cashLimit"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.CashLimit < 0 {
		req.CashLimit = 0
	}
	if _, err := h.Store.Branches.UpdateOne(r.Context(), bson.M{"_id": id},
		bson.M{"$set": bson.M{"cashLimit": req.CashLimit, "updatedAt": time.Now()}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, "branch.cashLimit", "branch", id.Hex(), "", formatSum(req.CashLimit))
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "cashLimit": req.CashLimit})
}
