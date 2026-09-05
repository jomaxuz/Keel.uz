package handlers

import (
	"math"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/bson"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- Finding a product by what the scanner read ----
//
// ⚠️ **A shop's counter is one action repeated, and this is it.** A restaurant
// is tapped: a waiter knows the menu and reaches for a tile. A shop is scanned —
// three thousand packets nobody has memorised, and the cashier's whole job is
// beep, beep, total. Everything about this endpoint follows from that: it has to
// answer in one request, it has to say *why* when it cannot, and it must never
// make somebody read a list.

// StaffScanBarcode answers with the product a scanned code names.
func (h *Handler) StaffScanBarcode(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	// ⚠️ **Trimmed, because scanners append a newline.** Most of them send the
	// code followed by Enter, and a keyboard-wedge reader on a web page delivers
	// that as part of the value. A code with a stray `\n` matches nothing, and
	// the failure reads as "this product is not in the system" — which sends
	// somebody to the catalogue to add a product that is already there.
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		httpx.Error(w, http.StatusBadRequest, "kod bo'sh")
		return
	}

	branch, err := h.branchByID(r, s.BranchID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// ⚠️ **A scale label is read before the catalogue is asked.** The digits a
	// scale prints are not a product's barcode — they are a short item code with
	// a weight or a price wrapped around it, so looking the whole string up
	// finds nothing and the counter says "unknown product" for goods that are
	// sitting in the catalogue.
	lookup := code
	var weighed *models.ScaleRead
	if read, ok := branch.Scale.Read(code); ok {
		lookup = read.ItemCode
		weighed = &read
	}

	// ⚠️ **Scoped to the till's own brand.** Two brands under one owner may
	// stock the same EAN, and a counter that reached across them would ring up
	// the other shop's price — which is right on the receipt and wrong in the
	// drawer.
	filter := bson.M{"barcode": lookup}
	if !branch.BrandID.IsZero() {
		filter["brandId"] = branch.BrandID
	}
	var item models.MenuItem
	if err := h.Store.Menu.FindOne(r.Context(), filter).Decode(&item); err != nil {
		// ⚠️ **Said as its own sentence, not as a 404.** The cashier is holding
		// the packet with a queue behind them; "not found" sends them looking
		// for a fault, and the honest next step is to add the product or call
		// somebody who can. The code is echoed back because it is what they will
		// be asked for.
		httpx.JSON(w, http.StatusOK, map[string]any{
			"found": false,
			// ⚠️ The scanned string, not the extracted item code: it is what the
			// cashier will be asked for and what is printed on the sticker in
			// their hand.
			"code": code,
		})
		return
	}

	// ⚠️ **Sold-out is answered here rather than at "add".** The branch already
	// knows, the cashier is still holding the packet, and finding out one tap
	// later is finding out after they have told the customer a price.
	out := map[string]any{
		"found":   true,
		"item":    item,
		"soldOut": branch.IsSoldOut(item.ID),
	}
	if weighed != nil {
		out["weighed"] = true
		kg := weighed.Kg

		// ⚠️ **A price label becomes a weight here, once, on the server.** The
		// till speaks quantities and nothing else; a line that carried its own
		// price would have to be understood by the check, the receipt, the fiscal
		// filing and the stock consumption — four places it could be forgotten.
		//
		// ⚠️ **And when the two prices disagree, the cashier is told rather than
		// the difference being swallowed.** The scale computes money from a unit
		// price of its own; if the shelf price changed and nobody updated the
		// scale, the sticker and the catalogue no longer agree. Charging the
		// catalogue price silently hides a scale that has been wrong all week —
		// while the customer is holding a sticker that says otherwise.
		if weighed.Price > 0 && item.Price > 0 {
			kg = float64(weighed.Price) / float64(item.Price)
			// Whole som on both sides, because that is what this product counts
			// in: one is rounding, more is a stale scale.
			back := int(math.Round(kg * float64(item.Price)))
			if diff := back - weighed.Price; diff > 1 || diff < -1 {
				out["priceMismatch"] = weighed.Price
			}
		}

		if kg > 0 {
			out["kg"] = kg
		}
	}
	httpx.JSON(w, http.StatusOK, out)
}
