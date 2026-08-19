package handlers

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/receipt"
)

// ---- X and Z ----
//
// ⚠️ **The two questions a drawer gets asked, and they are not the same one.**
// X is read mid-shift and changes nothing: how much has been sold, how much
// cash should be in there right now. Z is the end of the day — it closes the
// shift, freezes the expected figure and produces the paper the owner keeps.
//
// So X is a **GET**: it can be pressed at four in the afternoon by somebody
// checking the drawer against a suspicion, as many times as they like, and the
// worst thing it can do is use paper. Z is what the existing close already
// does; this file only gives it something to print.
//
// ⚠️ **Sold is not the same as in the drawer, and both lines are on the
// paper.** The takings are what the shift sold by payment method; the drawer is
// float + cash sales + courier handovers ± manual entries. A card sale is in
// the first and not the second, and a report showing only one of them is the
// report somebody uses to accuse a cashier.

// shiftSales is what a shift sold, from the checks themselves.
//
// ⚠️ **Counted from the orders, never from a running total on the shift.** A
// counter incremented as sales close drifts the first time anything is written
// twice or a process restarts mid-service, and the drift is invisible until a
// Z report disagrees with the reports — at which point neither number can be
// trusted and there is no way to find out which was wrong.
type shiftSales struct {
	Checks    int
	Guests    int
	Sales     int
	Cash      int
	Card      int
	Transfer  int
	// Taken away on the slate. ⚠️ **Not part of Sales**: no money arrived, and
	// a shift whose "sold" figure includes debts hands the cashier a total the
	// drawer can never match.
	Debt      int
	Discount  int
	Service   int
	Refunded  int
	Cancelled int
}

func (h *Handler) salesInShift(
	ctx context.Context, branchID primitive.ObjectID, from time.Time, to *time.Time,
) (shiftSales, error) {
	var out shiftSales
	rng := bson.M{"$gte": from}
	if to != nil {
		rng["$lte"] = *to
	}
	filter := bson.M{
		"check":          bson.M{"$exists": true},
		"check.closedAt": rng,
	}
	if !branchID.IsZero() {
		filter["branchId"] = branchID
	}
	cur, err := h.Store.Orders.Find(ctx, filter)
	if err != nil {
		return out, err
	}
	var orders []models.Order
	if err := cur.All(ctx, &orders); err != nil {
		return out, err
	}
	for _, o := range orders {
		// ⚠️ A cancelled check is not a sale and not a cover — its total is
		// still on the document, so counting it would book money nobody paid.
		if o.Status == models.StatusCancelled {
			out.Cancelled++
			continue
		}
		out.Checks++
		out.Guests += o.Check.Guests
		out.Discount += o.DiscountTotal
		out.Service += o.ServiceCharge
		// ⚠️ Refunded money is off the takings and on its own line. Folding it
		// into the sales figure would make a shift with two refunds look like a
		// quiet evening rather than a busy one that gave money back.
		if o.PaymentStatus == models.PayRefunded {
			out.Refunded += o.Total
			continue
		}
		// ⚠️ A debt is closed, delivered and unpaid — it belongs on the paper
		// (somebody has to be told the slate grew tonight) but not in the
		// takings. Before this it fell through to "transfer", which claimed
		// money had arrived by bank and made the shift impossible to
		// reconcile — silently, because the total still looked plausible.
		if o.PaymentMethod == models.MethodDebt && o.PaymentStatus != models.PayPaid {
			out.Debt += o.Total
			continue
		}
		out.Sales += o.Total
		switch o.PaymentMethod {
		case models.ProviderCash:
			out.Cash += o.Total
		case "card":
			out.Card += o.Total
		default:
			out.Transfer += o.Total
		}
	}
	return out, nil
}

// StaffShiftReport lays out the X report for the open shift.
func (h *Handler) StaffShiftReport(w http.ResponseWriter, r *http.Request) {
	// ⚠️ The waiter's permission, like reading the drawer's figures on screen:
	// this is the same numbers on paper, and requiring a manager to print what
	// is already displayed teaches people that permissions are theatre.
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	shift, err := h.openCashShift(r, bson.M{"branchId": s.BranchID})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if shift == nil {
		httpx.Error(w, http.StatusBadRequest, "ochiq smena yo'q")
		return
	}
	figures, _, err := h.shiftFigures(r, shift)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	data, tpl, err := h.shiftReportData(r.Context(), shift, figures, "X")
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"lines":   receipt.RenderShift(tpl, data),
		"widthMM": tpl.WidthMM,
	})
}

// shiftReportData fills the paper for either report.
func (h *Handler) shiftReportData(
	ctx context.Context, shift *models.CashShift, f cashFigures, kind string,
) (receipt.ShiftData, receipt.Template, error) {
	settings := h.receiptSettingsOf(ctx, shift.BranchID)
	// ⚠️ The **till** template, not the customer's: this is the document for
	// the person with the drawer, and the guest's header, footer and paper
	// width were chosen for something that leaves the building.
	tpl := settings.Till

	d := receipt.ShiftData{
		Kind:         kind,
		Currency:     "so'm",
		OpenedAt:     shift.OpenedAt.In(time.Local).Format("02.01.2006 15:04"),
		OpenedBy:     shift.OpenedBy,
		PrintedAt:    time.Now().In(time.Local).Format("02.01.2006 15:04"),
		OpeningFloat: f.OpeningFloat,
		CounterCash:  f.CounterCash,
		Settlements:  f.Settlements,
		ManualIn:     f.ManualIn,
		ManualOut:    f.ManualOut,
		Expected:     f.Expected,
	}
	if shift.ClosedAt != nil {
		d.ClosedAt = shift.ClosedAt.In(time.Local).Format("02.01.2006 15:04")
		d.ClosedBy = shift.ClosedBy
		d.Counted = shift.Counted
		d.Variance = shift.Variance
		d.VarianceNote = shift.VarianceNote
	}
	sales, err := h.salesInShift(ctx, shift.BranchID, shift.OpenedAt, shift.ClosedAt)
	if err != nil {
		return d, tpl, err
	}
	d.Checks, d.Guests, d.Sales = sales.Checks, sales.Guests, sales.Sales
	d.Cash, d.Card, d.Transfer = sales.Cash, sales.Card, sales.Transfer
	d.Discount, d.Service = sales.Discount, sales.Service
	d.Refunded, d.Cancelled = sales.Refunded, sales.Cancelled
	d.Debt = sales.Debt

	var rest models.Restaurant
	if err := h.Store.Restaurant.FindOne(ctx, bson.M{}).Decode(&rest); err == nil {
		d.Title = rest.Name
		if rest.Currency != "" {
			d.Currency = rest.Currency
		}
	}
	if branch, err := h.branchByIDCtx(ctx, shift.BranchID); err == nil && branch != nil {
		d.Branch = branch.Name
		d.Address = branch.Address.Text
	}
	if d.Title == "" {
		d.Title = "Restoran"
	}
	return d, tpl, nil
}
