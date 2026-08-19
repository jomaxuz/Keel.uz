package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/printer"
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
	Printers []models.Printer `json:"printers"`
}

// cleanPrinters is what is safe to store from a form.
//
// ⚠️ **The address is validated here, not when the kitchen is waiting.** A typo
// in "tcp://192.168.1.5o:9100" is discovered at eight o'clock otherwise, by a
// ticket that never comes out, and the person who typed it went home at five.
func cleanPrinters(in []models.Printer) []models.Printer {
	out := make([]models.Printer, 0, len(in))
	for _, p := range in {
		p.Name = clampText(p.Name, 60)
		p.Target = strings.TrimSpace(p.Target)
		if p.Target == "" {
			continue
		}
		if _, err := printer.Parse(p.Target); err != nil {
			continue
		}
		if p.ID == "" {
			p.ID = lineID()
		}
		if p.Copies < 1 {
			p.Copies = 1
		}
		kinds := make([]string, 0, len(p.Kinds))
		for _, k := range p.Kinds {
			switch receipt.Kind(k) {
			case receipt.Kitchen, receipt.Till, receipt.Customer, receipt.Precheck:
				kinds = append(kinds, k)
			}
		}
		p.Kinds = kinds
		out = append(out, p)
	}
	return out
}

// AdminTestPrint sends a sample receipt to one printer.
//
// ⚠️ **The most useful button on the page**, and for the same reason the SMS
// page's is: the address can be typed correctly and the printer still be off,
// on another subnet, or shared under a different name — and every one of those
// looks identical from here until a real ticket fails. It prints the *sample*
// receipt, so what comes out is what the owner has been designing.
func (h *Handler) AdminTestPrint(w http.ResponseWriter, r *http.Request) {
	branchID, err := h.posBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		PrinterID string `json:"printerId"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := h.receiptSettingsOf(r.Context(), branchID)
	var target *models.Printer
	for i := range set.Printers {
		if set.Printers[i].ID == req.PrinterID {
			target = &set.Printers[i]
			break
		}
	}
	if target == nil {
		httpx.Error(w, http.StatusNotFound, "printer topilmadi")
		return
	}
	// ⚠️ Queued like any other job rather than printed from here: the server
	// cannot reach a printer inside a restaurant, and a test that took a
	// different path from real printing would be a test of the wrong thing.
	tpl := set.Customer
	kind := receipt.Customer
	if target.Prints(string(receipt.Kitchen)) {
		tpl, kind = set.Kitchen, receipt.Kitchen
	}
	job := *target
	job.Kinds = []string{string(kind)}
	saved := set.Printers
	set.Printers = []models.Printer{job}
	n := h.queueReceiptTo(r.Context(), branchID, set, kind, tpl,
		h.sampleReceipt(r, branchID))
	set.Printers = saved
	httpx.JSON(w, http.StatusOK, map[string]any{"queued": n})
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
		"printers":  cleanPrinters(req.Printers),
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
	// ⚠️ The preview says **whether** a logo will be printed, not what it looks
	// like as dots: the picture is drawn by the browser, at a resolution no
	// thermal head has. A restaurant checking "will my logo be on the receipt"
	// is answered; a restaurant checking "will it come out as a smudge" is
	// answered by the test print, which is the real paper.
	logo := ""
	if req.Customer.Logo || req.Till.Logo {
		logo = h.logoURL(r.Context())
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"kitchen":  receipt.Render(receipt.Kitchen, cleanTemplate(req.Kitchen), d),
		"till":     receipt.Render(receipt.Till, cleanTemplate(req.Till), d),
		"customer": receipt.Render(receipt.Customer, cleanTemplate(req.Customer), d),
		"logoUrl":  logo,
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
