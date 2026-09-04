package handlers

// ---- Who is holding the restaurant's cash ----
//
// ⚠️ **An advance is not an expense.** The money is spent when it buys
// something, and that is the `purchase` document the financial report already
// counts. Recording the hand-over as an outgoing as well would count the same
// money twice — once as cash leaving and once as food arriving — and the month
// would read far worse than it was. See models/advance.go.
//
// ⚠️ **The balance is subtracted from documents, never stored.** A kept running
// total is a second copy of an answer the ledger already contains, and it drifts
// the first time a delivery is deleted or an advance corrected — silently, in a
// figure about money. Three sums and one subtraction cost one query each and
// cannot disagree with what they are made of.

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// advanceBalances is every account this branch is carrying.
//
// ⚠️ **Aggregated rather than walked.** A restaurant that has been buying at a
// market daily for two years has thousands of rows here and hundreds of
// deliveries a month; loading them to add up two numbers is the difference
// between a screen and a screen nobody opens.
func (h *Handler) advanceBalances(
	ctx context.Context, branch primitive.ObjectID, staff primitive.ObjectID,
) ([]models.AdvanceBalance, error) {
	match := bson.M{}
	if !branch.IsZero() {
		match["branchId"] = branch
	}
	if !staff.IsZero() {
		match["staffId"] = staff
	}

	rows := map[primitive.ObjectID]*models.AdvanceBalance{}
	last := map[primitive.ObjectID]time.Time{}
	names := map[primitive.ObjectID]string{}
	at := func(id primitive.ObjectID) *models.AdvanceBalance {
		if rows[id] == nil {
			rows[id] = &models.AdvanceBalance{StaffID: id.Hex()}
		}
		return rows[id]
	}

	cur, err := h.Store.Advances.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id":  bson.M{"staff": "$staffId", "kind": "$kind"},
			"sum":  bson.M{"$sum": "$amount"},
			"last": bson.M{"$max": "$at"},
			"name": bson.M{"$last": "$staffName"},
		}}},
	})
	if err != nil {
		return nil, err
	}
	var ledger []struct {
		ID struct {
			Staff primitive.ObjectID `bson:"staff"`
			Kind  string             `bson:"kind"`
		} `bson:"_id"`
		Sum  int       `bson:"sum"`
		Last time.Time `bson:"last"`
		Name string    `bson:"name"`
	}
	if err := cur.All(ctx, &ledger); err != nil {
		return nil, err
	}
	for _, l := range ledger {
		row := at(l.ID.Staff)
		if l.ID.Kind == models.AdvanceBack {
			row.Returned += l.Sum
		} else {
			row.Issued += l.Sum
			// ⚠️ Only a hand-over dates the account. "Last given anything" is
			// what says whether a balance is this morning's or from March; a
			// return is the end of a run, not the start of one.
			if l.Last.After(last[l.ID.Staff]) {
				last[l.ID.Staff] = l.Last
			}
		}
		if l.Name != "" {
			names[l.ID.Staff] = l.Name
		}
	}

	// ⚠️ **Only deliveries this person paid for.** An invoice recorded on
	// credit is money the restaurant still owes a supplier — it never passed
	// through anybody's hands, and taking it off a buyer's balance would show
	// them as having spent cash they still hold. The market runs the phone
	// writes are marked paid at the stall, which is what they are.
	spendMatch := bson.M{"paid": true, "createdById": bson.M{"$ne": primitive.NilObjectID}}
	if !branch.IsZero() {
		spendMatch["branchId"] = branch
	}
	if !staff.IsZero() {
		spendMatch["createdById"] = staff
	}
	scur, err := h.Store.Purchases.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: spendMatch}},
		{{Key: "$group", Value: bson.M{"_id": "$createdById", "sum": bson.M{"$sum": "$total"}}}},
	})
	if err != nil {
		return nil, err
	}
	var spent []struct {
		ID  primitive.ObjectID `bson:"_id"`
		Sum int                `bson:"sum"`
	}
	if err := scur.All(ctx, &spent); err != nil {
		return nil, err
	}
	for _, sp := range spent {
		// ⚠️ Somebody who has spent but never been given anything is skipped
		// rather than listed at a negative balance. A manager entering an
		// invoice from the panel is not holding petty cash, and a screen that
		// said they owed the restaurant money would be wrong about a person.
		if rows[sp.ID] == nil {
			continue
		}
		rows[sp.ID].Spent = sp.Sum
	}

	out := make([]models.AdvanceBalance, 0, len(rows))
	for id, row := range rows {
		row.StaffName = names[id]
		if row.StaffName == "" {
			row.StaffName = h.staffNameByID(ctx, id)
		}
		if t, ok := last[id]; ok {
			when := t
			row.LastAt = &when
		}
		row.Balance = row.Issued - row.Returned - row.Spent
		out = append(out, *row)
	}
	// Largest holding first: the question this screen is opened with is who is
	// carrying the most of the restaurant's money.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Balance != out[j].Balance {
			return out[i].Balance > out[j].Balance
		}
		return out[i].StaffName < out[j].StaffName
	})
	return out, nil
}

func (h *Handler) staffNameByID(ctx context.Context, id primitive.ObjectID) string {
	var s models.Staff
	if err := h.Store.Staff.FindOne(ctx, bson.M{"_id": id}).Decode(&s); err != nil {
		return ""
	}
	return s.Name
}

// AdminAdvances is the ledger and the balances behind it.
func (h *Handler) AdminAdvances(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	branch := h.scopeBranch(r, scope)
	staff, _ := objectID(r.URL.Query().Get("staffId"))

	balances, err := h.advanceBalances(r.Context(), branch, staff)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	filter := bson.M{}
	if !branch.IsZero() {
		filter["branchId"] = branch
	}
	if !staff.IsZero() {
		filter["staffId"] = staff
	}
	cur, err := h.Store.Advances.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(100))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	entries := []models.StaffAdvance{}
	_ = cur.All(r.Context(), &entries)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"balances": balances,
		"entries":  entries,
	})
}

// AdminCreateAdvance hands money over, or takes it back.
func (h *Handler) AdminCreateAdvance(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		StaffID string `json:"staffId"`
		Kind    string `json:"kind"`
		Amount  int    `json:"amount"`
		Note    string `json:"note"`
		// Whether the notes came out of the safe, or went back into it.
		//
		// ⚠️ **Asked rather than assumed.** A float can just as easily come
		// from an owner's own pocket, and a safe balance that quietly counted
		// every hand-over would be a confident figure about a box nobody
		// opened. When it is ticked one linked row is written, and the link
		// makes writing it twice impossible — see handlers/safe.go.
		FromSafe bool `json:"fromSafe"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	staffID, err := objectID(req.StaffID)
	if err != nil || staffID.IsZero() {
		httpx.Error(w, http.StatusBadRequest, "xodim tanlanmagan")
		return
	}
	if req.Amount <= 0 {
		httpx.Error(w, http.StatusBadRequest, "summani yozing")
		return
	}
	kind := models.AdvanceOut
	if req.Kind == models.AdvanceBack {
		kind = models.AdvanceBack
	}

	// ⚠️ The employee has to be one this admin can see. Without it a pasted id
	// files a hand-over against somebody in another branch's ledger — the same
	// rule every branch-scoped write here follows.
	var st models.Staff
	if err := h.Store.Staff.FindOne(r.Context(), bson.M{"_id": staffID}).Decode(&st); err != nil {
		httpx.Error(w, http.StatusNotFound, "xodim topilmadi")
		return
	}
	branch := st.BranchID
	if branch.IsZero() {
		branch = h.scopeBranch(r, scope)
	}
	if err := h.requireBranchAccess(r, branch); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}

	now := time.Now()
	in := models.StaffAdvance{
		BranchID: branch, StaffID: staffID, StaffName: st.Name,
		Kind: kind, Amount: req.Amount, At: now,
		Note:      clampText(req.Note, 200),
		By:        h.adminName(r),
		CreatedAt: now,
	}
	res, err := h.Store.Advances.InsertOne(r.Context(), in)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	in.ID = oidOf(res.InsertedID)

	// ⚠️ Money changing hands is written down with both names — the same reason
	// a void carries one. "Who gave Sanjar two million on Tuesday" is asked the
	// following month, and the answer has to be a record rather than a memory.
	what := "podotchet berildi"
	if kind == models.AdvanceBack {
		what = "podotchet qaytarildi"
	}
	h.logAction(r, "advance.create", "staff", staffID.Hex(), st.Name,
		what+": "+formatSum(req.Amount))

	if req.FromSafe {
		// Giving money out of the safe takes it out; taking change back puts it
		// in. ⚠️ The direction is the mirror of the advance's, not a copy of it.
		kind := models.SafeOut
		if in.Kind == models.AdvanceBack {
			kind = models.SafeIn
		}
		h.recordSafeMovement(r.Context(), models.SafeEntry{
			BranchID: branch, Kind: kind, Amount: in.Amount, At: in.At,
			Category: "podotchet", Note: st.Name,
			By:      in.By,
			RefKind: models.SafeRefAdvance, RefID: in.ID,
		})
	}

	balances, _ := h.advanceBalances(r.Context(), branch, staffID)
	out := map[string]any{"entry": in}
	if len(balances) > 0 {
		out["balance"] = balances[0]
	}
	httpx.JSON(w, http.StatusCreated, out)
}

// AdminDeleteAdvance removes an entry that was written by mistake.
//
// ⚠️ **Deleted rather than reversed, and only here.** A counter-entry would be
// the tidier accounting answer and the wrong one for this ledger: a mistyped
// amount corrected with a second row leaves two figures that both look like
// real hand-overs, and the person reading it a month later cannot tell a
// correction from a second trip to the safe. The action is journalled, which is
// where the trail belongs.
func (h *Handler) AdminDeleteAdvance(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var row models.StaffAdvance
	if err := h.Store.Advances.FindOne(r.Context(), bson.M{"_id": id}).Decode(&row); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, row.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	if _, err := h.Store.Advances.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, "advance.delete", "staff", row.StaffID.Hex(), row.StaffName,
		"podotchet yozuvi o'chirildi: "+formatSum(row.Amount))
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// StaffBuyBalance is what this buyer is still holding.
//
// ⚠️ **On their own screen, not only the owner's.** The number decides whether
// they set off at all, and a buyer who has to ring somebody to find out how much
// they are carrying will guess instead — which is the state this ledger exists
// to end.
func (h *Handler) StaffBuyBalance(w http.ResponseWriter, r *http.Request) {
	s, ok := h.buyStaff(w, r)
	if !ok {
		return
	}
	balances, err := h.advanceBalances(r.Context(), s.BranchID, s.ID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ A buyer nobody has given anything to has an account of zeroes rather
	// than no account: an empty response would leave the screen unable to tell
	// "you hold nothing" from "this could not be read".
	out := models.AdvanceBalance{StaffID: s.ID.Hex(), StaffName: s.Name}
	if len(balances) > 0 {
		out = balances[0]
	}
	httpx.JSON(w, http.StatusOK, out)
}
