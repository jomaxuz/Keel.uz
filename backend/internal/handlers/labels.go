package handlers

// ---- Shelf labels and barcode stickers ----
//
// ⚠️ **A shop cannot sell what it cannot scan.** Half of what a shop stocks
// arrives with a barcode printed by whoever made it; the other half — repacked,
// weighed out, baked in the back, bought at a market stall — arrives with
// nothing, and a till has no way to ring it up. Printing a code is not a
// convenience there: it is the difference between a product existing at the
// counter and not.
//
// ⚠️ **And a price on a shelf is the law's business, not ours to be casual
// about.** A label is wrong the moment somebody changes a price in the panel,
// and nothing on any screen said which ones had gone stale — so the answer was a
// shop reprinting everything, or reprinting nothing. That is what
// `LabelPrice` is for: the price *as printed*, so "which shelves are lying" is a
// question with an answer.
//
// ⚠️ **This is not Asl Belgisi, and cannot be.** The state's marking codes are
// issued to a producer or an importer, per unit and for money; a shop that
// resells scans them and never invents one. What is printed here is the shop's
// own code, in the range GS1 reserves for exactly that — see internal/barcode.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/barcode"
	"restaurant-backend/internal/escpos"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/receipt"
)

// maxLabelsPerRun caps one press of the button.
//
// ⚠️ **Because a delivery of two hundred crates is a real Tuesday**, and a
// queue that accepted it would hold a roll of paper somebody has to stand and
// watch come out. The refusal names the number, so the person can split the run
// rather than guess why nothing printed.
const maxLabelsPerRun = 300

type labelRequest struct {
	Items []labelItem `json:"items"`
}

type labelItem struct {
	ID string `json:"id"`
	// How many stickers. ⚠️ Zero means one: the commonest press of this button
	// is a single product whose price changed, and asking for a number there
	// would be a form standing in front of a one-tap job.
	Copies int `json:"copies"`
}

// AdminPrintLabels queues one label per copy asked for.
func (h *Handler) AdminPrintLabels(w http.ResponseWriter, r *http.Request) {
	branchID, brandID, err := h.labelScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
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
			fmt.Sprintf("bir marta %d tagacha yorliq chiqarish mumkin", maxLabelsPerRun))
		return
	}

	queued, created, err := h.queueLabels(r.Context(), branchID, brandID, req.Items)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if queued == 0 {
		// ⚠️ **Said out loud rather than answered with a silent success.** A
		// branch with no printer set to take labels queues nothing, and "it
		// printed" followed by no paper is the report we would get instead.
		httpx.Error(w, http.StatusBadRequest, "yorliq bosadigan printer sozlanmagan")
		return
	}
	h.logAction(r, "label.print", "menu", "", fmt.Sprintf("%d ta yorliq", queued), "")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"queued": queued,
		// Which products were given a barcode on the way, so the panel can say
		// so — a code invented here is a fact somebody may need to know.
		"barcoded": created,
	})
}

// AdminStaleLabels is every product whose shelf label no longer says the truth.
//
// ⚠️ **Computed from the printed price, never from a flag.** A boolean set at
// print time goes stale the moment somebody edits a price, which is exactly the
// event this list exists to catch; the price as printed does not.
func (h *Handler) AdminStaleLabels(w http.ResponseWriter, r *http.Request) {
	_, brandID, err := h.labelScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	rows, err := h.labelCandidates(r.Context(), brandID)
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
			"id":      it.ID.Hex(),
			"name":    it.Name,
			"price":   it.Price,
			"barcode": it.Barcode,
			// What changed since the sticker was printed, so somebody can tell a
			// price rise from a product that has never had a label at all.
			"reason":   reason,
			"wasPrice": it.LabelPrice,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out})
}

// staleReason names why this product's shelf is wrong, or "" when it is right.
func staleReason(it models.MenuItem) string {
	if it.Barcode == "" {
		// ⚠️ First, because it is the one that stops a sale rather than
		// misdescribing it: no code, no scan, no ring-up.
		return "noBarcode"
	}
	if it.LabelAt == nil {
		return "never"
	}
	if it.LabelPrice != it.Price {
		return "price"
	}
	return ""
}

// labelCandidates is the goods a shop puts labels on.
//
// ⚠️ **Only what sells itself.** A portion of osh has no shelf and no barcode
// and never will; a list that offered one would bury the forty things that do
// under four hundred that never can.
func (h *Handler) labelCandidates(
	ctx context.Context, brandID primitive.ObjectID,
) ([]models.MenuItem, error) {
	filter := bson.M{"sellsItself": true}
	if !brandID.IsZero() {
		filter["brandId"] = brandID
	}
	cur, err := h.Store.Menu.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var out []models.MenuItem
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// labelScope answers which branch prints and which catalogue is being labelled.
//
// ⚠️ **A branch is required, and that is not a formality.** The printers belong
// to a branch, and "print the whole chain's labels" is a sentence with no
// printer behind it. A single-branch shop never sees the question — the branch
// is worked out for it, exactly as the stock screens do.
func (h *Handler) labelScope(r *http.Request) (branch, brand primitive.ObjectID, err error) {
	_, branch, brand, err = h.stockBranch(r)
	return branch, brand, err
}

// queueLabels builds the bytes and puts them in front of the agent.
func (h *Handler) queueLabels(
	ctx context.Context, branchID, brandID primitive.ObjectID, items []labelItem,
) (queued int, barcoded []string, err error) {
	set := h.receiptSettingsOf(ctx, branchID)
	printers := []models.Printer{}
	for _, p := range set.Printers {
		if p.Prints(string(receipt.Label)) {
			printers = append(printers, p)
		}
	}
	if len(printers) == 0 {
		return 0, nil, nil
	}

	branch, err := h.branchByIDCtx(ctx, branchID)
	if err != nil {
		return 0, nil, err
	}
	now := time.Now()

	for _, want := range items {
		id, err := objectID(want.ID)
		if err != nil {
			continue
		}
		var it models.MenuItem
		filter := bson.M{"_id": id}
		if !brandID.IsZero() {
			filter["brandId"] = brandID
		}
		if err := h.Store.Menu.FindOne(ctx, filter).Decode(&it); err != nil {
			continue
		}

		// ⚠️ **The code is invented at the moment it is printed, not before.**
		// A barcode that exists in a database and on no packet is a code nobody
		// can scan; allocating it here means every internal code in the
		// catalogue is one that has been stuck to something.
		if it.Barcode == "" {
			code, err := h.allocateBarcode(ctx, brandID, *branch)
			if err != nil {
				return queued, barcoded, err
			}
			if _, err := h.Store.Menu.UpdateByID(ctx, it.ID,
				bson.M{"$set": bson.M{"barcode": code}}); err != nil {
				return queued, barcoded, err
			}
			it.Barcode = code
			barcoded = append(barcoded, it.Name)
		}

		lines := labelLines(it, *branch)
		bars := escpos.EncodeBarcode(it.Barcode, barcode.Valid(it.Barcode))
		for _, p := range printers {
			// ⚠️ **Built once per printer, sent once per copy.** Six stickers of
			// the same product are six jobs, not one job with six inside: the
			// agent can fail halfway through a delivery and the queue has to be
			// able to say which of them came out.
			payload := labelPayload(lines, bars, p)
			for range want.Copies {
				job := models.PrintJob{
					BranchID:    branchID,
					PrinterID:   p.ID,
					PrinterName: p.Name,
					Target:      p.Target,
					Kind:        string(receipt.Label),
					Number:      it.Barcode,
					Payload:     payload,
					CreatedAt:   now,
				}
				if _, err := h.Store.PrintJobs.InsertOne(ctx, job); err == nil {
					queued++
				}
			}
		}

		// ⚠️ Recorded per product rather than per job: what matters later is
		// "what price was on the sticker", and that is one fact however many
		// copies came out.
		_, _ = h.Store.Menu.UpdateByID(ctx, it.ID, bson.M{"$set": bson.M{
			"labelAt":    now,
			"labelPrice": it.Price,
		}})
	}
	return queued, barcoded, nil
}

// labelPayload is one sticker: the words, then the bars, then the cut.
//
// ⚠️ **The cut belongs at the end and only once.** `Encode` writes it with the
// text, so the barcode would land after the paper had been cut — a sticker with
// no code on it and a stray code on the next one.
func labelPayload(lines []string, bars []byte, p models.Printer) []byte {
	body := escpos.Encode(lines, escpos.Options{Charset: charsetOf(p)})
	out := make([]byte, 0, len(body)+len(bars)+8)
	out = append(out, body...)
	out = append(out, bars...)
	out = append(out, escpos.Encode(nil, escpos.Options{
		Charset:   charsetOf(p),
		FeedLines: 1,
		Cut:       p.Cut,
		FullCut:   p.FullCut,
	})...)
	return out
}

// labelLines is what is printed above the bars.
//
// ⚠️ **The price is the largest thing on it, and the name is second.** A shelf
// label is read from a metre away by somebody deciding whether to pick the
// packet up; a sticker on the packet is read at the till by a scanner, which
// needs none of the words. One layout serves both, so it is written for the
// harder reader.
func labelLines(it models.MenuItem, branch models.Branch) []string {
	name := strings.TrimSpace(it.Name)
	lines := []string{"!" + name}
	if branch.Name != "" {
		lines = append(lines, branch.Name)
	}
	// ⚠️ The unit beside the price, because "18 500" means nothing on a shelf of
	// loose goods until it says whether that is a kilo or a packet.
	price := formatSom(it.Price)
	if u := unitWord(it); u != "" {
		price += " / " + u
	}
	lines = append(lines, "!!"+price)
	return lines
}

// unitWord is how this product is sold, in the words the shop uses.
func unitWord(it models.MenuItem) string {
	// ⚠️ The state classifier's codes, which is what the field holds: 10 is a
	// gram and 112 a litre. A word invented here would disagree with the receipt.
	switch it.UnitCode {
	case 10:
		return "kg"
	case 112:
		return "l"
	}
	return ""
}

// allocateBarcode hands out the next internal code for this brand.
//
// ⚠️ **Checked against the branch's own scale reader before it is handed out.**
// A code this shop's till would decode as a weight is the failure
// `models.ScaleLabel` is written about: the counter beeps, shows the product and
// charges for a quantity nobody weighed. See internal/barcode.
func (h *Handler) allocateBarcode(
	ctx context.Context, brandID primitive.ObjectID, branch models.Branch,
) (string, error) {
	scale := branch.Scale.Defaults()
	readsAsScale := func(code string) bool {
		_, ok := scale.Read(code)
		return ok
	}

	// The highest internal code already in this catalogue, so numbering carries
	// on rather than restarting into a collision.
	filter := bson.M{"barcode": bson.M{"$regex": "^" + barcode.InternalPrefix}}
	if !brandID.IsZero() {
		filter["brandId"] = brandID
	}
	var last models.MenuItem
	seq := int64(0)
	err := h.Store.Menu.FindOne(ctx, filter,
		options.FindOne().SetSort(bson.D{{Key: "barcode", Value: -1}})).Decode(&last)
	if err == nil && len(last.Barcode) == 13 {
		if n, e := parseInt64(last.Barcode[len(barcode.InternalPrefix):12]); e == nil {
			seq = n + 1
		}
	}

	code, err := barcode.Allocate(seq, readsAsScale)
	if err != nil {
		// ⚠️ **Named rather than swallowed**, because the fix is a setting and
		// not a retry: a branch whose scale prefix is the bare "2" has claimed
		// the whole in-store range, and every code we could invent would be
		// read back as a weight.
		return "", errors.New(
			"ichki shtrix-kod chiqmadi — tarozi prefiksi butun diapazonni egallagan, " +
				"Sozlamalar → Tarozi da aniqroq prefiks tanlang")
	}
	return code, nil
}

func parseInt64(s string) (int64, error) {
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errors.New("not a number")
		}
		n = n*10 + int64(r-'0')
	}
	return n, nil
}

// LabelsDue is what a delivery just made worth printing.
//
// ⚠️ **Offered, never printed on its own.** A delivery of two hundred packets
// that answered itself with two hundred stickers would be a roll of paper
// nobody asked for, and the printer would be switched off within a week. What
// the delivery removes is the typing: the products and the counts are already
// known, so the panel can put one button in front of somebody who would
// otherwise have hunted through a catalogue.
//
// ⚠️ **Only what sells itself, and only what a label would change.** A
// restaurant's sack of flour has no shelf; a product whose sticker already says
// the price it sells for does not need another.
type LabelDue struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Price   int    `json:"price"`
	Barcode string `json:"barcode,omitempty"`
	// How many arrived, which is how many stickers a shop would want.
	//
	// ⚠️ **Whole units, floored, and never zero.** Four and a half kilos of
	// loose rice is one shelf label, not five stickers — and a fraction rounded
	// up would print a sticker for a packet that did not arrive.
	Copies int    `json:"copies"`
	Reason string `json:"reason"`
}

// labelsDueFor turns a delivery into the labels it makes sense to print.
func (h *Handler) labelsDueFor(
	ctx context.Context, p models.Purchase,
) []LabelDue {
	if len(p.Lines) == 0 {
		return nil
	}
	byStock := map[primitive.ObjectID]float64{}
	for _, l := range p.Lines {
		if l.IngredientID.IsZero() {
			continue
		}
		byStock[l.IngredientID] += l.Qty
	}
	if len(byStock) == 0 {
		return nil
	}

	ids := make([]primitive.ObjectID, 0, len(byStock))
	for id := range byStock {
		ids = append(ids, id)
	}
	// ⚠️ Keyed on the stock row, which is what a shop's product *is*: the
	// delivery knows nothing about menu items, and it should not have to.
	cur, err := h.Store.Menu.Find(ctx, bson.M{
		"sellsItself": true,
		"stockId":     bson.M{"$in": ids},
	})
	if err != nil {
		return nil
	}
	var rows []models.MenuItem
	if err := cur.All(ctx, &rows); err != nil {
		return nil
	}

	out := []LabelDue{}
	for _, it := range rows {
		reason := staleReason(it)
		if reason == "" {
			continue
		}
		copies := int(byStock[it.StockID])
		if copies < 1 {
			copies = 1
		}
		if copies > maxLabelsPerRun {
			copies = maxLabelsPerRun
		}
		out = append(out, LabelDue{
			ID:      it.ID.Hex(),
			Name:    it.Name,
			Price:   it.Price,
			Barcode: it.Barcode,
			Copies:  copies,
			Reason:  reason,
		})
	}
	return out
}
