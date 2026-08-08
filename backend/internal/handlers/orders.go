package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type quoteRequest struct {
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Subtotal int     `json:"subtotal"`
	// Which brand's kitchen should carry it. Empty = the first active brand,
	// which is all a single-brand restaurant has.
	BrandID string `json:"brandId"`
}

// DeliveryQuote computes the delivery fee for a destination.
//
// The guest never picks a branch: they give an address and the server finds the
// kitchen that covers it (see deliveryBranch). The branch it settled on comes
// back with the price, so the checkout page can name who is cooking.
func (h *Handler) DeliveryQuote(w http.ResponseWriter, r *http.Request) {
	var req quoteRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	brandID, err := h.brandIDFrom(r, req.BrandID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	branch, quote, err := h.deliveryBranch(r, brandID, req.Lat, req.Lng, req.Subtotal)
	if err != nil {
		// Outside every zone is a normal answer, not an error: the page shows
		// "we don't deliver here" rather than a broken screen.
		httpx.JSON(w, http.StatusOK, map[string]any{
			"available":   false,
			"deliveryFee": 0,
			"zone":        "",
			"distanceKm":  0,
			"minOrder":    h.minOrderOf(r, brandID),
		})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"available":   quote.Available,
		"deliveryFee": quote.Fee,
		"zone":        quote.Zone,
		"distanceKm":  quote.DistanceKm,
		"minOrder":    quote.MinOrder,
		"branchId":    branch.ID,
		"branchName":  branch.Name,
		"prepMinutes": branch.PrepMinutes,
	})
}

// brandIDFrom resolves an optional brand sent in a request body, falling back
// to the install's first active brand. Accepts an id or a slug, because the
// site's brand cookie holds whichever the printed QR card carried.
func (h *Handler) brandIDFrom(r *http.Request, raw string) (primitive.ObjectID, error) {
	if raw = strings.TrimSpace(raw); raw != "" {
		filter := bson.M{"slug": raw}
		if id, err := objectID(raw); err == nil {
			filter = bson.M{"_id": id}
		}
		var b models.Brand
		if err := h.Store.Brands.FindOne(r.Context(), filter).Decode(&b); err != nil {
			return primitive.NilObjectID, errors.New("brend topilmadi")
		}
		return b.ID, nil
	}
	brand, err := h.publicBrand(r)
	if err != nil || brand == nil {
		return primitive.NilObjectID, nil
	}
	return brand.ID, nil
}

// minOrderOf is the "you still need N more" figure shown when no branch covers
// the address: the lowest minimum any of the brand's branches asks for.
func (h *Handler) minOrderOf(r *http.Request, brandID primitive.ObjectID) int {
	branch, err := h.defaultBranch(r, brandID)
	if err != nil {
		return 0
	}
	return branch.Delivery.MinOrder
}

type createOrderRequest struct {
	Customer models.OrderCustomer `json:"customer" validate:"required"`
	Type     string               `json:"type" validate:"required,oneof=delivery pickup dinein"`
	// Dine-in: the table whose QR code the guest scanned.
	TableID string `json:"tableId"`
	// Pickup and dine-in: which branch the guest walked into. Delivery ignores
	// it — the address decides, not the browser.
	BranchID string              `json:"branchId"`
	Address  models.OrderAddress `json:"address"`
	Items    []models.OrderItem  `json:"items" validate:"required,min=1,dive"`
	// A promo code the guest typed. A wrong one is not fatal — the order still
	// goes through, at full price, with the reason returned alongside it.
	PromoCode string `json:"promoCode"`
	// How many loyalty points to put towards this order. The server caps it
	// against the real balance — this is a request, not an instruction.
	UsePoints     int    `json:"usePoints"`
	PaymentMethod string `json:"paymentMethod" validate:"required,oneof=cash payme click uzum"`
	// Which door this came in through — "web" or "telegram". See Order.Channel:
	// attribution rather than authorisation, which is why the browser is allowed
	// to say and why anything unrecognised becomes "web".
	Channel string `json:"channel"`
}

// CreateOrder validates items server-side, computes totals, and stores the order.
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// Who is ordering, if they signed in.
	userID, _ := h.optionalUserID(r)
	order, status, err := h.composeOrder(r, req, userID, "")
	if err != nil {
		httpx.Error(w, status, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, h.withPayLink(r, order))
}

// withPayLink attaches the provider checkout URL to a freshly created order, so
// the checkout page can send the guest straight to it instead of making them
// find a "pay" button on the next screen.
func (h *Handler) withPayLink(r *http.Request, order *models.Order) map[string]any {
	out := map[string]any{}
	raw, err := json.Marshal(order)
	if err == nil {
		_ = json.Unmarshal(raw, &out)
	}
	if url := h.payURL(r.Context(), order, ""); url != "" {
		out["payUrl"] = url
	}
	return out
}

// composeOrder is everything that turns a filled cart into a stored order:
// re-pricing against the live menu, choosing the branch, applying discounts and
// points, and writing the row.
//
// It is shared by the two ways an order is placed — the guest on the site and
// an operator on the phone (AdminCreateOrder). Deliberately one function: the
// moment the call centre gets its own copy of this, the two start drifting, and
// the drift shows up as a phone order priced differently from the same basket
// ordered on the site. `takenBy` is the only difference between them, and it is
// a label on the receipt, not a rule.
//
// Returns the HTTP status to answer with alongside the error.
func (h *Handler) composeOrder(
	r *http.Request,
	req createOrderRequest,
	userID primitive.ObjectID,
	takenBy string,
) (*models.Order, int, error) {
	// Recompute item prices from DB to prevent client-side tampering.
	subtotal := 0
	items := make([]models.OrderItem, 0, len(req.Items))
	// Which brand this cart belongs to, taken from the dishes themselves rather
	// than a field the browser sends: the cart is per-brand by construction, so
	// the menu is the honest source. A cart that spans two brands is refused —
	// two kitchens cannot fill one receipt.
	var brandID primitive.ObjectID
	// Dishes reached through a combo. They are checked against the branch's
	// sold-out list too — a set is only sellable if every course in it is.
	var comboMembers []models.ComboLine
	for _, it := range req.Items {
		var dbItem models.MenuItem
		if err := h.Store.Menu.FindOne(r.Context(), bson.M{"_id": it.MenuItemID}).Decode(&dbItem); err != nil {
			// The dish was removed or renamed since the cart was filled.
			name := it.Name
			if name == "" {
				name = "Taom"
			}
			return nil, http.StatusBadRequest,
				errors.New(name + " menyuda topilmadi — savatni yangilang")
		}
		if !dbItem.IsAvailable {
			return nil, http.StatusBadRequest, errors.New(dbItem.Name + " hozircha mavjud emas")
		}
		if brandID.IsZero() {
			brandID = dbItem.BrandID
		} else if dbItem.BrandID != brandID {
			return nil, http.StatusBadRequest,
				errors.New("savatda ikki xil brend taomi bor — alohida buyurtma bering")
		}
		// Options are re-resolved against the menu: the client only says which
		// choice it picked, the price delta always comes from the DB.
		opts, err := resolveOptions(&dbItem, it.Options)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		unit := dbItem.Price
		for _, o := range opts {
			unit += o.PriceDelta
		}
		if unit < 0 {
			unit = 0
		}
		line := models.OrderItem{
			MenuItemID: dbItem.ID,
			Name:       dbItem.Name,
			Price:      unit,
			Qty:        it.Qty,
			Options:    opts,
			// Kept as typed, only trimmed and capped — a note the kitchen reads.
			Comment: clampText(it.Comment, 200),
		}
		// A combo carries its contents onto the receipt: "Oilaviy combo" alone
		// is not something a kitchen can cook from. Resolved here, against the
		// live menu, so a set whose dish was deleted or pulled cannot be sold.
		if dbItem.IsCombo() {
			res, err := h.resolveCombo(r.Context(), &dbItem, nil)
			if err != nil {
				return nil, http.StatusInternalServerError, err
			}
			if res.Blocked != "" {
				return nil, http.StatusBadRequest, errors.New(dbItem.Name + ": " + res.Blocked)
			}
			line.ComboItems = res.Contents
			comboMembers = append(comboMembers, dbItem.ComboItems...)
		}
		subtotal += line.Price * line.Qty
		items = append(items, line)
	}

	// Which kitchen takes this order.
	//
	// Delivery is decided by the address alone — the guest gave a place, the
	// company knows which branch reaches it. Pickup and dine-in are decided by
	// where the guest physically is, so those do come from the request.
	var branch *models.Branch
	deliveryFee, zone, km := 0, "", 0.0
	minOrder := 0
	if req.Type == "delivery" {
		var quote deliveryQuote
		var err error
		branch, quote, err = h.deliveryBranch(r, brandID, req.Address.Lat, req.Address.Lng, subtotal)
		if err != nil {
			return nil, http.StatusBadRequest, errors.New("bu manzilga yetkazib berilmaydi")
		}
		// Checked further down, against the discounted subtotal.
		minOrder = quote.MinOrder
		deliveryFee, zone, km = quote.Fee, quote.Zone, quote.DistanceKm
	} else {
		var err error
		if strings.TrimSpace(req.BranchID) != "" {
			id, idErr := objectID(req.BranchID)
			if idErr != nil {
				return nil, http.StatusBadRequest, errors.New("filial noto'g'ri")
			}
			branch, err = h.branchByID(r, id)
		} else {
			branch, err = h.defaultBranch(r, brandID)
		}
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
	}

	// What the branch has run out of today. Checked **after** the branch is
	// known, because that is the only point at which the question has an answer:
	// the guest browsed one branch's menu, but delivery may be taken by another.
	for _, line := range items {
		if branch.IsSoldOut(line.MenuItemID) {
			return nil, http.StatusBadRequest, errors.New(line.Name + " bugun tugadi")
		}
	}
	// Same question for what is inside the combos: the set itself is not on the
	// sold-out list, but one of its courses may be.
	for _, member := range comboMembers {
		if !branch.IsSoldOut(member.MenuItemID) {
			continue
		}
		var dish models.MenuItem
		_ = h.Store.Menu.FindOne(r.Context(), bson.M{"_id": member.MenuItemID}).Decode(&dish)
		name := dish.Name
		if name == "" {
			name = "To'plamdagi taom"
		}
		return nil, http.StatusBadRequest, errors.New(name + " bugun tugadi")
	}

	// Dine-in: the guest is sitting at a table they reached by scanning its QR
	// code. No address, no fee — but the table has to be a real one **at this
	// branch**, or the kitchen gets a receipt it cannot carry anywhere.
	tableID, tableNumber := "", ""
	if req.Type == "dinein" {
		b := bookingSettings(branch.Booking)
		for _, tb := range b.Tables {
			if tb.ID == req.TableID {
				tableID, tableNumber = tb.ID, tb.Number
				break
			}
		}
		if tableID == "" {
			return nil, http.StatusBadRequest,
				errors.New("stol topilmadi — QR kodni qayta skaner qiling")
		}
	}

	// Every discount, recomputed here from scratch. The checkout showed the
	// guest a preview; this is the number that counts.
	price, err := h.computePrice(r.Context(), priceInput{
		Items:        items,
		Subtotal:     subtotal,
		Type:         req.Type,
		BrandID:      brandID,
		BranchID:     branch.ID,
		UserID:       userID,
		Code:         req.PromoCode,
		UsePoints:    req.UsePoints,
		DeliveryFee:  deliveryFee,
		ItemCategory: h.itemCategories(r.Context(), items),
	})
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	// The delivery minimum is checked against what the restaurant actually
	// receives for the food. A promo code that drops the basket under the
	// branch's floor would otherwise have it delivering at a loss.
	if req.Type == "delivery" && subtotal-price.DiscountTotal < minOrder {
		return nil, http.StatusBadRequest, errors.New("order below minimum")
	}

	now := time.Now()
	order := models.Order{
		BrandID:       brandID,
		BranchID:      branch.ID,
		Number:        branchOrderNumber(branch.Code),
		Status:        models.StatusPending,
		Customer:      req.Customer,
		Type:          req.Type,
		TableID:       tableID,
		TableNumber:   tableNumber,
		Address:       req.Address,
		Items:         items,
		Subtotal:      subtotal,
		Discounts:     price.Discounts,
		DiscountTotal: price.DiscountTotal,
		PointsSpent:   price.PointsSpent,
		DeliveryFee:   price.DeliveryFee,
		Total:         price.Total,
		PaymentMethod: req.PaymentMethod,
		DeliveryZone:  zone,
		DistanceKm:    km,
		StatusHistory: []models.StatusEvent{{Status: models.StatusPending, At: now}},
		// Empty for an order the guest placed themselves; the operator's name
		// when it came in over the phone. It is on the receipt because "who
		// typed this in?" is the first question asked about a wrong address.
		TakenBy: takenBy,
		// An operator session outranks whatever the browser said: this field is
		// read later as "who is answerable for this order", and the one value that
		// must never be forgeable is the one naming a member of staff.
		Channel:   orderChannel(req.Channel, takenBy),
		CreatedAt: now,
		UpdatedAt: now,
	}
	// Cash is settled at the door, so the order joins the kitchen queue the
	// moment it is placed. An online order does not: it waits for the bank, and
	// until that lands nobody should cook it or be chimed about it.
	if req.PaymentMethod == models.ProviderCash {
		order.PaymentStatus = models.PayUnpaid
		order.QueuedAt = &now
	} else {
		order.PaymentStatus = models.PayPending
	}
	order.UserID = userID
	res, err := h.Store.Orders.InsertOne(r.Context(), order)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	order.ID = res.InsertedID.(primitive.ObjectID)
	// Count the redemptions only once the order exists — a code must not be
	// burned by an attempt that failed on the line above.
	h.redeem(r.Context(), price.Discounts)
	// Where this customer came from, recorded once, on their first order. Only
	// filled when still empty, so an operator's correction is never overwritten.
	if !order.UserID.IsZero() {
		_, _ = h.Store.Users.UpdateOne(r.Context(),
			bson.M{"_id": order.UserID, "source": bson.M{"$in": []any{nil, ""}}},
			bson.M{"$set": bson.M{"source": sourceFromOrder(&order)}})
	}
	// Points come off the balance now: the same points must not be promised to
	// a second order while this one is still being cooked. Cashback is paid the
	// other way round — only once the order is delivered (see awardPoints).
	if price.PointsSpent > 0 {
		h.moveBalance(r.Context(), order.UserID, -price.PointsSpent,
			models.LoyaltySpend, &order, "")
	}
	return &order, http.StatusCreated, nil
}

// TrackOrder returns the public-safe status of an order by its number.
func (h *Handler) TrackOrder(w http.ResponseWriter, r *http.Request) {
	number := chi.URLParam(r, "number")
	var order models.Order
	if err := h.Store.Orders.FindOne(r.Context(), bson.M{"number": number}).Decode(&order); err != nil {
		httpx.Error(w, http.StatusNotFound, "order not found")
		return
	}
	resp := map[string]any{
		"number":      order.Number,
		"status":      order.Status,
		"type":        order.Type,
		"total":       order.Total,
		"createdAt":   order.CreatedAt,
		"address":     order.Address,
		"tableNumber": order.TableNumber,
		// Whether the money arrived, and how it was meant to. The guest needs
		// this more than anyone: an order that is waiting on a half-finished
		// card payment looks, from the kitchen's silence, exactly like an order
		// that was ignored.
		"paymentMethod": order.PaymentMethod,
		"paymentStatus": paymentStatusOf(&order),
	}
	// Still owed money and still worth paying: hand back the link so the page
	// can offer to finish it.
	if paymentStatusOf(&order) == models.PayPending && payable(&order) == nil {
		if url := h.payURL(r.Context(), &order, ""); url != "" {
			resp["payUrl"] = url
		}
	}
	// Why it was cancelled — the customer is owed the reason without ringing.
	if order.Status == models.StatusCancelled && order.CancelReason != "" {
		resp["cancelReason"] = order.CancelReason
	}
	// While the order is on its way the customer may follow the courier. The
	// position is only exposed for that one stage — never before or after.
	if !order.CourierID.IsZero() {
		resp["courierName"] = order.CourierName
		if order.Status == models.StatusOnTheWay {
			var c models.Courier
			if err := h.Store.Couriers.FindOne(r.Context(),
				bson.M{"_id": order.CourierID}).Decode(&c); err == nil && c.Location != nil {
				resp["courier"] = map[string]any{
					"name":     c.Name,
					"phone":    c.Phone,
					"location": c.Location,
				}
			}
		}
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// orderChannel narrows what the client claimed to the three values that exist.
//
// Unknown becomes "web" rather than being kept or rejected: a new client version
// sending something we have not seen yet must not fail an order, and a label
// nobody recognises is worse in a report than the common case.
func orderChannel(claimed, takenBy string) string {
	if strings.TrimSpace(takenBy) != "" {
		return "operator"
	}
	if strings.TrimSpace(claimed) == "telegram" {
		return "telegram"
	}
	return "web"
}
