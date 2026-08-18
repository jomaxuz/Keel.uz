package handlers

import (
	"net/http"
	"strconv"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/receipt"

	"go.mongodb.org/mongo-driver/bson"
)

// Printing from the till.
//
// ⚠️ **The paper is laid out by the same code that draws the preview**
// (internal/receipt), so what a restaurant designed in the panel is what comes
// out of the printer. A second layout here would drift, and the drift would be
// discovered by a guest holding a receipt that does not match the one the owner
// approved — the same rule the reports follow: one calculation, two outputs.
//
// ⚠️ **The browser prints, not the server.** There is no printer driver yet
// (see docs/pos-reja.md §2: the Windows app is paused), and a monoblock has the
// receipt printer installed as an ordinary Windows printer — so the screen asks
// for the lines and prints them itself. That also means a branch with no
// printer still gets a readable bill on screen to read out, which is what half
// of them will do on the first evening anyway.

type printRequest struct {
	// kitchen · till · customer · precheck
	Kind string `json:"kind"`
}

// StaffPrintCheck renders one of a check's receipts as lines of text.
//
// ⚠️ **The pre-check is a POST and it writes**, unlike the other three: handing
// a table its bill is an event on that check — the floor screen draws it, the
// next waiter past the table knows not to ask again, and "asked twenty minutes
// ago" is a different situation from "asked just now". Re-printing is allowed
// and does not move the time: the first ask is the one that matters, and a
// guest who lost the paper has not started waiting again.
func (h *Handler) StaffPrintCheck(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok {
		return
	}
	var req printRequest
	if r.ContentLength > 0 {
		if err := httpx.Decode(r, &req); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	kind := receipt.Kind(req.Kind)
	switch kind {
	case receipt.Kitchen, receipt.Till, receipt.Customer, receipt.Precheck:
	default:
		// ⚠️ Not a silent fallback to the customer copy: that one carries the
		// fiscal sign, and a typo in a screen's request must never be the
		// reason a document claiming a registered sale comes out of a printer.
		httpx.Error(w, http.StatusBadRequest, "qanday chek ekani noma'lum")
		return
	}

	settings := h.receiptSettingsOf(r.Context(), s.BranchID)
	tpl := settings.Kitchen
	switch kind {
	case receipt.Till:
		tpl = settings.Till
	case receipt.Customer, receipt.Precheck:
		// ⚠️ The bill borrows the **customer** template on purpose: it is the
		// document the guest reads, and a restaurant that set its paper width,
		// its header and its footer for the receipt meant them for anything
		// that leaves its hand.
		tpl = settings.Customer
	}

	now := time.Now()
	if kind == receipt.Precheck {
		if o.Check == nil || o.Check.ClosedAt != nil {
			httpx.Error(w, http.StatusConflict, "chek yopilgan")
			return
		}
		if o.Check.PrecheckAt == nil {
			o.Check.PrecheckAt = &now
			if _, err := h.Store.Orders.UpdateOne(r.Context(),
				checkFilter(o.ID, s.BranchID),
				bson.M{"$set": bson.M{"check.precheckAt": now, "updatedAt": now}},
			); err != nil {
				httpx.Error(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"lines":   receipt.Render(kind, tpl, h.checkReceipt(r, o)),
		"widthMM": tpl.WidthMM,
		"check":   viewCheck(o, now),
	})
}

// checkReceipt turns an open or closed check into the renderer's flat data.
//
// ⚠️ **Voided lines are left off.** They stay on screen, where the running
// total has to be explainable to the person watching it being typed; on paper
// they are food the guest never had and is not being charged for, and a line
// they cannot be charged for is a line they will ask about.
func (h *Handler) checkReceipt(r *http.Request, o *models.Order) receipt.Data {
	d := receipt.Data{
		Number:   o.Number,
		Currency: "so'm",
		Table:    o.TableNumber,
	}
	if d.Table != "" {
		d.Table += "-stol"
	}
	if o.Check != nil {
		d.Guests = o.Check.Guests
		d.Server = o.Check.ServerName
		d.OpenedAt = o.Check.OpenedAt.In(time.Local).Format("02.01 15:04")
		if o.Check.ClosedAt != nil {
			d.ClosedAt = o.Check.ClosedAt.In(time.Local).Format("02.01 15:04")
			d.Cashier = o.Check.ClosedBy
		}
	}
	for _, it := range o.Items {
		if !it.Live() {
			continue
		}
		line := receipt.Line{
			Name:    it.Name,
			Qty:     it.Qty,
			Price:   it.Price,
			Sum:     it.Price * it.Qty,
			Comment: it.Comment,
		}
		for i, opt := range it.Options {
			if i > 0 {
				line.Options += ", "
			}
			line.Options += opt.Choice
		}
		// Whose it is, when the table asked to be billed separately: a guest
		// handed a bill for the whole table is a guest who has to work out
		// their own share at the table, which is the thing splitting exists to
		// avoid.
		if it.Guest > 0 {
			line.Name = strconv.Itoa(it.Guest) + ". " + line.Name
		}
		d.Subtotal += line.Sum
		d.Lines = append(d.Lines, line)
	}
	d.Discount = o.DiscountTotal
	if len(o.Discounts) > 0 {
		d.DiscountName = o.Discounts[0].Name
	}
	d.Total = d.Subtotal - d.Discount
	if d.Total < 0 {
		d.Total = 0
	}
	if o.Fiscal != nil {
		// Only ever printed on the customer copy — the renderer decides that,
		// not this function.
		d.FiscalSign = o.Fiscal.FiscalSign
		d.QRText = o.Fiscal.QRText
	}

	var rest models.Restaurant
	if err := h.Store.Restaurant.FindOne(r.Context(), bson.M{}).Decode(&rest); err == nil {
		d.Title = rest.Name
		if rest.Currency != "" {
			d.Currency = rest.Currency
		}
	}
	if branch, err := h.branchByID(r, o.BranchID); err == nil && branch != nil {
		if branch.Name != "" {
			d.Title = branch.Name
		}
		d.Address = branch.Address.Text
		if len(branch.Phones) > 0 {
			d.Phone = branch.Phones[0]
		}
	}
	if d.Title == "" {
		d.Title = "Restoran"
	}
	return d
}
