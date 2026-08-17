package handlers

import (
	"context"
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/receipt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Receipt designs, and the preview that proves they are right.
//
// ⚠️ **The preview is rendered by the printer's own code.** internal/receipt
// lays out the paper, and this endpoint hands the same function a made-up sale.
// A preview drawn in the browser would be a second implementation of the same
// character grid, and the two would drift — with the difference discovered by a
// restaurant whose receipts do not look like the thing they designed. Same rule
// as the reports: one calculation, two outputs.

// receiptSettingsOf loads a branch's designs, falling back to the defaults.
//
// ⚠️ A missing document is the ordinary case and must not mean "print nothing":
// the zero value of the template is disabled and zero-width, which for a
// restaurant that just plugged in a printer is indistinguishable from a broken
// printer. Defaults that print something are the only ones anybody can debug
// from the paper in their hand.
func (h *Handler) receiptSettingsOf(
	ctx context.Context, branchID primitive.ObjectID,
) models.ReceiptSettings {
	out := models.DefaultReceipts(branchID)
	if branchID.IsZero() {
		return out
	}
	var stored models.ReceiptSettings
	if err := h.Store.Receipts.FindOne(ctx, bson.M{"branchId": branchID}).
		Decode(&stored); err != nil {
		return out
	}
	return stored
}

func (h *Handler) AdminGetReceipts(w http.ResponseWriter, r *http.Request) {
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, h.receiptSettingsOf(r.Context(), branchID))
}

type receiptsRequest struct {
	Kitchen  receipt.Template `json:"kitchen"`
	Till     receipt.Template `json:"till"`
	Customer receipt.Template `json:"customer"`
}

func (h *Handler) AdminUpdateReceipts(w http.ResponseWriter, r *http.Request) {
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req receiptsRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := bson.M{
		"branchId":  branchID,
		"kitchen":   cleanTemplate(req.Kitchen),
		"till":      cleanTemplate(req.Till),
		"customer":  cleanTemplate(req.Customer),
		"updatedAt": time.Now(),
	}
	if _, err := h.Store.Receipts.UpdateOne(r.Context(),
		bson.M{"branchId": branchID}, bson.M{"$set": set},
		options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActSettingsUpdate, "receipts", branchID.Hex(), "", "")
	httpx.JSON(w, http.StatusOK, h.receiptSettingsOf(r.Context(), branchID))
}

// cleanTemplate keeps a submitted design inside what a printer can do.
//
// ⚠️ **The paper width is clamped to the two that exist.** A template saved with
// `widthMm: 0` renders at the 80 mm default and prints half off the edge of a
// 58 mm roll — silently, because nothing in the software can see the paper.
func cleanTemplate(t receipt.Template) receipt.Template {
	if t.WidthMM != 58 {
		t.WidthMM = 80
	}
	t.Header = clampText(t.Header, 120)
	t.Footer = clampText(t.Footer, 200)
	// A feed of forty lines is a roll of blank paper per order, and somebody
	// will type it while experimenting.
	if t.FeedLines < 0 {
		t.FeedLines = 0
	}
	if t.FeedLines > 10 {
		t.FeedLines = 10
	}
	if t.Fields == nil {
		t.Fields = map[string]bool{}
	}
	return t
}

// AdminPreviewReceipt renders a design against a made-up sale.
//
// ⚠️ **A sample, not a real order.** An owner designing a receipt at ten in the
// morning has nothing to preview against, and reaching for "the last check" ties
// the look of the design to whatever the restaurant happened to sell. The sample
// is built to exercise the awkward parts: a long dish name that must wrap, a
// comment, a discount, and change.
func (h *Handler) AdminPreviewReceipt(w http.ResponseWriter, r *http.Request) {
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req receiptsRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	d := h.sampleReceipt(r, branchID)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"kitchen":  receipt.Render(receipt.Kitchen, cleanTemplate(req.Kitchen), d),
		"till":     receipt.Render(receipt.Till, cleanTemplate(req.Till), d),
		"customer": receipt.Render(receipt.Customer, cleanTemplate(req.Customer), d),
	})
}

// sampleReceipt is the fake sale the preview is drawn against.
//
// ⚠️ Uses the restaurant's **real** name, address and phone, because those are
// the lines an owner is actually checking the length of — a preview headed
// "Restoran" tells them nothing about whether their own name fits on 58 mm
// paper.
func (h *Handler) sampleReceipt(r *http.Request, branchID primitive.ObjectID) receipt.Data {
	d := receipt.Data{
		Number: "MRC-A1-1745", Table: "7-stol", Guests: 2,
		OpenedAt: "17.08 19:05", ClosedAt: "17.08 19:42",
		Server: "Aziz", Cashier: "Dilnoza", OrderType: "Olib ketish",
		Lines: []receipt.Line{
			// Long enough to wrap on 58 mm, which is the case an owner cannot
			// picture and will not think to test.
			{Name: "Qaymoqli achchiq lag'mon, katta porsiya", Qty: 2,
				Price: 45000, Sum: 90000, Comment: "piyozsiz"},
			{Name: "Choy", Qty: 1, Price: 5000, Sum: 5000, Options: "ko'k"},
		},
		Subtotal: 95000, Discount: 5000, DiscountName: "Chegirma 5%",
		Total: 90000, Paid: 100000, Change: 10000, Method: "Naqd",
		FiscalSign: "002519286194",
		QRText:     "https://ofd.soliq.uz/check?t=UZ21&r=538",
		Currency:   "so'm",
	}

	var rest models.Restaurant
	if err := h.Store.Restaurant.FindOne(r.Context(), bson.M{}).Decode(&rest); err == nil {
		d.Title = rest.Name
		d.Currency = rest.Currency
	}
	if branch, err := h.branchByID(r, branchID); err == nil && branch != nil {
		if branch.Name != "" {
			d.Title = branch.Name
		}
		d.Address = branch.Address.Text
		// The first number only: a receipt is 32 characters wide and a list of
		// three phones would push the address off the paper.
		if len(branch.Phones) > 0 {
			d.Phone = branch.Phones[0]
		}
	}
	if d.Title == "" {
		d.Title = "Restoran"
	}
	if d.Currency == "" {
		d.Currency = "so'm"
	}
	return d
}
