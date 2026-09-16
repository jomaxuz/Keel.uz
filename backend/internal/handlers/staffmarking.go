package handlers

// ---- Unpacking a delivery with a phone ----
//
// ⚠️ **The scanner was a pistol wired to a counter, and the box is in the store
// room.** Scanning at goods-in is the whole point of the marking feature — a
// code that was never received is refused in front of a guest otherwise — but
// until now it could only be done at a machine with a scanner attached. So a
// delivery was either carried to the counter or scanned later from memory,
// which is the same as not scanning it.
//
// The phone in the pocket of whoever opened the box already has a camera. What
// it lacked was a door: the panel's endpoints take an admin session, and an
// employee holds a staff token.
//
// ⚠️ **Nothing here is a second implementation.** The codes are filed by
// `saveMarks` and the labels are queued by `queueLabels` — the same functions
// the panel calls. A phone that filed marks its own way would disagree with the
// counter, and the disagreement surfaces weeks later as a sale the till
// refuses.
//
// ⚠️ **The branch comes off the employee, never the request** — the rule every
// staff screen follows. A phone that could name a branch could file somebody
// else's delivery, and a marking code is unique across the whole platform:
// filing it wrongly takes it away from the branch that actually has the bottle.

import (
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// staffPermGuard is the three checks every endpoint below starts with.
//
// ⚠️ **Two refusals with different words**, the pattern `stockDenial` already
// set: one sends the person to whoever switched their account off, the other to
// whoever owns the permissions. A single "forbidden" sends them to neither.
func (h *Handler) staffPermGuard(
	w http.ResponseWriter, r *http.Request, perm, denial string,
) (models.Staff, bool) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return s, false
	}
	if !s.IsActive {
		httpx.Error(w, http.StatusForbidden,
			"hisob o'chirilgan — ma'muriyat bilan bog'laning")
		return s, false
	}
	if !s.Can(perm) {
		// 403 rather than 404: this person works here and can see the tab is
		// there, so pretending the screen does not exist would only confuse.
		httpx.Error(w, http.StatusForbidden, denial)
		return s, false
	}
	return s, true
}

const (
	denyMarking = "markirovka skanerlashga ruxsat berilmagan — administratorga murojaat qiling"
	denyLabel   = "yorliq bosishga ruxsat berilmagan — administratorga murojaat qiling"
)

// StaffMarkItems is what carries a marking code, for the picker on the phone.
//
// ⚠️ **Only marked products.** A list of the whole menu would bury the six
// things this screen is for under four hundred that carry no code — the same
// judgement the panel's screen makes.
func (h *Handler) StaffMarkItems(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffPermGuard(w, r, models.PermMarking, denyMarking)
	if !ok {
		return
	}
	_, brand := h.staffStockScope(r, s)
	filter := bson.M{"marked": true}
	if !brand.IsZero() {
		filter["brandId"] = brand
	}
	cur, err := h.Store.Menu.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}).SetLimit(500))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cur.Close(r.Context())
	// ⚠️ Never nil: a restaurant that has flagged nothing as marked is the
	// ordinary case, and `null.length` in a phone is a blank screen with no
	// error — the JSON trap this codebase has been bitten by twice.
	items := []map[string]any{}
	for cur.Next(r.Context()) {
		var it models.MenuItem
		if cur.Decode(&it) != nil {
			continue
		}
		items = append(items, map[string]any{
			"id": it.ID.Hex(), "name": it.Name, "barcode": it.Barcode,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

// StaffReceiveMarks files what a camera read off a delivery.
func (h *Handler) StaffReceiveMarks(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffPermGuard(w, r, models.PermMarking, denyMarking)
	if !ok {
		return
	}
	var req receiveMarksRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Codes) == 0 {
		httpx.Error(w, http.StatusBadRequest, "hech narsa skanerlanmadi")
		return
	}
	// ⚠️ **The same cap as the panel's**, and for the phone it binds sooner: a
	// camera fills the list one code at a time, and a screen that let somebody
	// scan a pallet into a single request would lose the lot to one dropped
	// connection.
	if len(req.Codes) > maxCodesPerScan {
		httpx.Error(w, http.StatusBadRequest, "bir marta 500 tagacha kod saqlanadi")
		return
	}
	itemID, _ := objectID(req.MenuItemID)
	purchaseID, _ := objectID(req.PurchaseID)

	added, dup, bad, err := h.saveMarks(
		r.Context(), s.BranchID, itemID, purchaseID, req.Codes, s.Name)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"added": added, "duplicates": dup, "bad": bad,
	})
}

// StaffMarkStock is how many of one product this branch is still holding.
func (h *Handler) StaffMarkStock(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffPermGuard(w, r, models.PermMarking, denyMarking)
	if !ok {
		return
	}
	filter := bson.M{"branchId": s.BranchID, "soldAt": nil}
	if id, err := objectID(r.URL.Query().Get("menuItemId")); err == nil && !id.IsZero() {
		filter["menuItemId"] = id
	}
	held, err := h.Store.MarkedUnits.CountDocuments(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"held": held})
}

// ---- Labels, from the same phone ----
//
// ⚠️ **This is the shop's own code, never the state's.** Asl Belgisi codes are
// issued to a producer or an importer, per unit and for money; a shop that
// resells scans them and never invents one. What is printed here is an EAN-13
// in the range GS1 reserves for in-store use — see internal/barcode, and the
// note at the top of labels.go.

// StaffLabelCandidates is what needs a sticker, for a phone standing at the
// shelf.
func (h *Handler) StaffLabelCandidates(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffPermGuard(w, r, models.PermLabel, denyLabel)
	if !ok {
		return
	}
	_, brand := h.staffStockScope(r, s)
	rows, err := h.labelCandidates(r.Context(), brand)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := []map[string]any{}
	for _, it := range rows {
		reason := staleReason(it)
		if reason == "" {
			continue
		}
		out = append(out, map[string]any{
			"id": it.ID.Hex(), "name": it.Name, "price": it.Price,
			"barcode": it.Barcode,
			// Why this one is on the list: no code at all stops a sale, a
			// changed price only misdescribes it, and the two are worth
			// different urgency in somebody's hand.
			"reason": reason, "wasPrice": it.LabelPrice,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out})
}

// StaffPrintLabels queues stickers for the branch's label printer.
//
// ⚠️ **How many is the person's answer, not the delivery's.** A crate of forty
// may need forty stickers or one for the shelf edge, and only the person
// holding it knows which — so the number is asked for and never inferred. The
// panel's screen makes the same choice, and the receiving screen proposes a
// figure rather than acting on one.
func (h *Handler) StaffPrintLabels(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffPermGuard(w, r, models.PermLabel, denyLabel)
	if !ok {
		return
	}
	var req labelRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Items) == 0 {
		httpx.Error(w, http.StatusBadRequest, "hech narsa tanlanmagan")
		return
	}
	total := 0
	for i := range req.Items {
		if req.Items[i].Copies < 1 {
			req.Items[i].Copies = 1
		}
		total += req.Items[i].Copies
	}
	if total > maxLabelsPerRun {
		httpx.Error(w, http.StatusBadRequest,
			"bir marta 300 tagacha yorliq chiqarish mumkin")
		return
	}
	_, brand := h.staffStockScope(r, s)

	queued, created, err := h.queueLabels(r.Context(), s.BranchID, brand, req.Items)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if queued == 0 {
		// ⚠️ **Said out loud rather than answered with a silent success.** A
		// branch with no printer set to take labels queues nothing, and "it
		// printed" followed by no paper is the report we would get instead —
		// from somebody standing at a shelf, who would then press it again.
		httpx.Error(w, http.StatusBadRequest, "yorliq bosadigan printer sozlanmagan")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"queued": queued,
		// Which products were given a barcode on the way: a code invented here
		// is a fact about the catalogue somebody may need to know.
		"barcoded": created,
	})
}
