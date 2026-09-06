package handlers

// ---- Choosing what a label looks like ----
//
// ⚠️ **The shop chooses, and it chooses by looking.** A dropdown of six words
// ("shelf", "compact", "full") asks somebody to imagine paper, and the person
// setting this up is standing at a counter with a roll in their hand. So the
// chooser shows all six designs rendered at once, and the choice is made the way
// it would be made off a shelf.
//
// ⚠️ **Rendered by the printer's own code** (internal/receipt), never redrawn in
// the browser. Two layout engines drift, and the drift is discovered by a shop
// whose stickers do not look like the design they picked. Same rule as the
// receipts preview and the reports: one calculation, two outputs.

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/receipt"
)

// AdminLabelDesign is the stored choice, with all six drawn out.
func (h *Handler) AdminLabelDesign(w http.ResponseWriter, r *http.Request) {
	branchID, brandID, err := h.labelScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := h.receiptSettingsOf(r.Context(), branchID)
	h.writeLabelDesign(w, r, branchID, brandID, set.Label.Defaults())
}

// AdminPreviewLabelDesign draws a design that has not been saved.
//
// ⚠️ **A draft, deliberately.** Changing the paper width or switching the shop's
// name off has to be visible before it is kept — a chooser that only shows what
// is already stored asks the shop to save in order to look.
func (h *Handler) AdminPreviewLabelDesign(w http.ResponseWriter, r *http.Request) {
	branchID, brandID, err := h.labelScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req receipt.LabelTemplate
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	h.writeLabelDesign(w, r, branchID, brandID, cleanLabelTemplate(req))
}

// AdminSaveLabelDesign keeps the choice.
func (h *Handler) AdminSaveLabelDesign(w http.ResponseWriter, r *http.Request) {
	branchID, brandID, err := h.labelScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req receipt.LabelTemplate
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	tpl := cleanLabelTemplate(req)
	// ⚠️ **One field's `$set`, not the whole document.** The receipts, the
	// printers and this design live together because they are one machine's
	// settings, and a save that wrote all of them would let the labels screen
	// blank a branch's printers. The same shape as the recipe's own endpoint.
	if _, err := h.Store.Receipts.UpdateOne(r.Context(),
		bson.M{"branchId": branchID},
		bson.M{"$set": bson.M{"branchId": branchID, "label": tpl, "updatedAt": time.Now()}},
		options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActSettingsUpdate, "labels", branchID.Hex(), tpl.Style, "")
	h.writeLabelDesign(w, r, branchID, brandID, tpl)
}

// cleanLabelTemplate keeps a submitted design inside what a printer can do.
func cleanLabelTemplate(t receipt.LabelTemplate) receipt.LabelTemplate {
	t = t.Defaults()
	if t.Lang != "ru" && t.Lang != "en" {
		t.Lang = ""
	}
	if t.Fields == nil {
		t.Fields = map[string]bool{}
	}
	// ⚠️ Only the switches this renderer reads. A field somebody's browser
	// invented would be stored for ever and mean nothing.
	fields := map[string]bool{}
	for _, k := range []string{"shop", "unit", "date"} {
		if v, ok := t.Fields[k]; ok {
			fields[k] = v
		}
	}
	t.Fields = fields
	return t
}

// writeLabelDesign answers with the design and every design.
func (h *Handler) writeLabelDesign(
	w http.ResponseWriter, r *http.Request,
	branchID, brandID primitive.ObjectID, tpl receipt.LabelTemplate,
) {
	sample := h.sampleLabel(r.Context(), branchID, brandID)
	// Named `drawn` rather than `options`: the mongo package of that name is
	// imported here, and a shadowed package is a compile error waiting for the
	// next edit that needs it.
	drawn := make([]map[string]any, 0, len(receipt.LabelStyles))
	for _, style := range receipt.LabelStyles {
		one := tpl
		one.Style = string(style)
		drawn = append(drawn, map[string]any{
			"style": string(style),
			"lines": receipt.RenderLabel(one, sample),
			// Whether the printer draws bars under it. ⚠️ The panel needs this to
			// draw the preview honestly: a price tag with a barcode sketched
			// under it is the one thing the shop would be choosing wrongly.
			"bars": style.Bars(),
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"design":  tpl,
		"options": drawn,
		"sample": map[string]any{
			"name":    sample.Name,
			"barcode": sample.Barcode,
		},
	})
}

// sampleLabel is the product the six designs are drawn against.
//
// ⚠️ **The shop's own product where there is one.** The lengths that matter are
// this catalogue's — a name that fits on 58 mm paper here and not there is the
// whole question the preview answers, and a made-up "Product" would answer it
// for a word nobody sells.
//
// ⚠️ **With an old price, always.** One of the six designs is the sale label,
// and it steps aside for the plain one when nothing is reduced — so a sample
// taken from a product at its ordinary price would show the chooser two
// identical stickers and no way to tell what the sale design does. The number
// is invented; the name, the code and the unit are the shop's.
func (h *Handler) sampleLabel(
	ctx context.Context, branchID, brandID primitive.ObjectID,
) receipt.LabelData {
	d := receipt.LabelData{
		Name:     "Guruch, Lazer, 1 kg",
		Price:    18500,
		Unit:     "kg",
		Barcode:  "2100000000017",
		Currency: h.currencyOf(ctx),
		Date:     time.Now().Format("02.01.2006"),
	}
	filter := bson.M{"sellsItself": true, "price": bson.M{"$gt": 0}}
	if !brandID.IsZero() {
		filter["brandId"] = brandID
	}
	var it models.MenuItem
	if err := h.Store.Menu.FindOne(ctx, filter).Decode(&it); err == nil {
		d.Name = it.Name
		d.Price = it.Price
		d.Unit = unitWord(it)
		if it.Barcode != "" {
			d.Barcode = it.Barcode
		}
	}
	if branch, err := h.branchByIDCtx(ctx, branchID); err == nil && branch != nil {
		d.Shop = branch.Name
	}
	// A quarter more than it costs now, rounded to something a shelf would
	// actually have said.
	d.OldPrice = (d.Price*5/4+499)/500*500 + 500
	return d
}
