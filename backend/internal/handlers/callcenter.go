package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The call centre desk.
//
// An operator picks up the phone and has, at most, the few seconds before they
// have to say something. Everything they need in those seconds is one answer:
// who this is, what they ordered last time, whether anything of theirs is in
// the kitchen right now, and whether the restaurant currently owes them an
// apology. Making them open three screens to assemble that is how a guest ends
// up being asked their own address.
//
// Hence one endpoint, keyed by the number that rang.

// callerOrder is an order trimmed to what an operator reads out loud.
type callerOrder struct {
	ID           primitive.ObjectID  `json:"id"`
	Number       string              `json:"number"`
	Status       models.OrderStatus  `json:"status"`
	Type         string              `json:"type"`
	Total        int                 `json:"total"`
	Items        []models.OrderItem  `json:"items"`
	Address      models.OrderAddress `json:"address"`
	CourierName  string              `json:"courierName,omitempty"`
	CancelReason string              `json:"cancelReason,omitempty"`
	CreatedAt    time.Time           `json:"createdAt"`
}

// callerFavourite is a dish this customer keeps ordering. The single most
// useful sentence an operator can open with is "the usual?", and this is what
// makes it possible.
type callerFavourite struct {
	MenuItemID primitive.ObjectID `json:"menuItemId"`
	Name       string             `json:"name"`
	Qty        int                `json:"qty"`
	Times      int                `json:"times"`
}

type callerLookup struct {
	Phone string `json:"phone"`
	// Nil when nobody with this number has ever ordered — a first-time caller.
	// The panel treats that as a normal, common case rather than an error.
	User *models.User `json:"user"`

	OrdersCount int        `json:"ordersCount"`
	OrdersTotal int        `json:"ordersTotal"`
	AvgOrder    int        `json:"avgOrder"`
	LastOrderAt *time.Time `json:"lastOrderAt"`
	// Which groups they fall into, computed the same way the customer list
	// computes them so the two screens never disagree.
	Segments []string `json:"segments"`

	// Anything still being cooked or carried. This is what most calls are
	// about, so it comes first and separately from the history.
	ActiveOrders []callerOrder     `json:"activeOrders"`
	RecentOrders []callerOrder     `json:"recentOrders"`
	Favourites   []callerFavourite `json:"favourites"`

	// Bookings still ahead of them.
	Reservations []models.Reservation `json:"reservations"`
	// Complaints nobody has answered yet. An operator must not greet someone
	// cheerfully while an unanswered complaint of theirs is sitting in the
	// panel — so it is put in front of them before they speak.
	OpenComplaints []models.Feedback `json:"openComplaints"`

	// What was said the last few times this number rang.
	RecentCalls []models.Call `json:"recentCalls"`
}

// AdminCallerLookup answers "who is ringing?" — ?phone= is required.
func (h *Handler) AdminCallerLookup(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("phone"))
	if raw == "" {
		httpx.Error(w, http.StatusBadRequest, "telefon raqam kerak")
		return
	}
	// Normalised where possible, but a number that does not fit the Uzbek shape
	// is still searched for as typed: a call centre also takes calls from abroad
	// and from office switchboards, and refusing to look those up would send the
	// operator back to asking the guest everything.
	phone := raw
	if norm, ok := normalizePhone(raw); ok {
		phone = norm
	}

	out := callerLookup{
		Phone:          phone,
		Segments:       []string{},
		ActiveOrders:   []callerOrder{},
		RecentOrders:   []callerOrder{},
		Favourites:     []callerFavourite{},
		Reservations:   []models.Reservation{},
		OpenComplaints: []models.Feedback{},
		RecentCalls:    []models.Call{},
	}

	ctx := r.Context()

	// ⚠️ Scoped by branch, and this endpoint is the reason the whole class of
	// by-id holes mattered: it hands back a customer's orders, addresses,
	// bookings, complaints and call history keyed by nothing but a phone number.
	// Unscoped, a manager pinned to one branch could read every guest's home
	// address and order history across the whole company, and harvest the order
	// ids the other endpoints act on. Here the branch narrows every operational
	// list below; the ids the operator then works with are already their own.
	branchScope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	scoped := func(f bson.M) bson.M {
		for k, v := range branchScope {
			f[k] = v
		}
		return f
	}

	var user models.User
	if err := h.Store.Users.FindOne(ctx, bson.M{"phone": phone}).Decode(&user); err == nil {
		out.User = &user
	}

	// Orders are found by the number on the order, not only by the account.
	// A guest who ordered before signing up, or who rang from their spouse's
	// phone, still has to be recognised — the phone is what the operator has.
	orderFilter := scoped(bson.M{"customer.phone": phone})
	if out.User != nil {
		orderFilter = scoped(bson.M{"$or": []bson.M{
			{"customer.phone": phone}, {"userId": user.ID},
		}})
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(50)
	var orders []models.Order
	if cur, err := h.Store.Orders.Find(ctx, orderFilter, opts); err == nil {
		_ = cur.All(ctx, &orders)
	}

	var firstAt, lastAt *time.Time
	dishes := map[primitive.ObjectID]*callerFavourite{}
	for i := range orders {
		o := &orders[i]
		row := callerOrder{
			ID: o.ID, Number: o.Number, Status: o.Status, Type: o.Type,
			Total: o.Total, Items: o.Items, Address: o.Address,
			CourierName: o.CourierName, CancelReason: o.CancelReason,
			CreatedAt: o.CreatedAt,
		}
		if isActiveStatus(o.Status) {
			out.ActiveOrders = append(out.ActiveOrders, row)
		} else if len(out.RecentOrders) < 10 {
			out.RecentOrders = append(out.RecentOrders, row)
		}

		out.OrdersCount++
		// Cancelled orders count as calls that happened but money that never
		// arrived — the same convention the dashboard and the customer list use.
		if o.Status != models.StatusCancelled {
			out.OrdersTotal += o.Total
		}
		at := o.CreatedAt
		if lastAt == nil {
			lastAt = &at
		}
		firstAt = &at

		// "The usual" is counted from what was actually delivered, so a basket
		// the guest cancelled does not become their favourite dish.
		if o.Status == models.StatusCancelled {
			continue
		}
		for _, it := range o.Items {
			f, ok := dishes[it.MenuItemID]
			if !ok {
				f = &callerFavourite{MenuItemID: it.MenuItemID, Name: it.Name}
				dishes[it.MenuItemID] = f
			}
			f.Qty += it.Qty
			f.Times++
		}
	}
	out.LastOrderAt = lastAt
	if out.OrdersCount > 0 {
		out.AvgOrder = out.OrdersTotal / out.OrdersCount
	}
	// Top few by how many separate orders they appear in: something ordered
	// once in bulk is not a habit, something ordered every time is.
	for _, f := range dishes {
		out.Favourites = append(out.Favourites, *f)
	}
	sortFavourites(out.Favourites)
	if len(out.Favourites) > 5 {
		out.Favourites = out.Favourites[:5]
	}

	if out.User != nil {
		out.Segments = segmentsFor(customerFacts{
			OrdersCount: out.OrdersCount,
			OrdersTotal: out.OrdersTotal,
			FirstOrder:  firstAt,
			LastOrder:   lastAt,
			Birthday:    user.Birthday,
			Unhappy:     h.unhappyUsers(r)[user.ID.Hex()],
		}, h.vipFloorNow(r), time.Now())
		if out.Segments == nil {
			out.Segments = []string{}
		}
	}

	// Bookings still ahead of them, soonest first.
	bookingFilter := scoped(bson.M{
		"customer.phone": phone,
		"at":             bson.M{"$gte": time.Now().Add(-2 * time.Hour)},
		"status":         bson.M{"$ne": string(models.ReservationCancelled)},
	})
	bopts := options.Find().SetSort(bson.D{{Key: "at", Value: 1}}).SetLimit(10)
	if cur, err := h.Store.Reservations.Find(ctx, bookingFilter, bopts); err == nil {
		_ = cur.All(ctx, &out.Reservations)
	}
	if out.Reservations == nil {
		out.Reservations = []models.Reservation{}
	}

	// Unanswered complaints.
	fbFilter := scoped(bson.M{"customer.phone": phone, "handled": false,
		"rating": bson.M{"$lte": lowRating}})
	fopts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(5)
	if cur, err := h.Store.Feedback.Find(ctx, fbFilter, fopts); err == nil {
		_ = cur.All(ctx, &out.OpenComplaints)
	}
	if out.OpenComplaints == nil {
		out.OpenComplaints = []models.Feedback{}
	}

	// What was said last time they rang.
	copts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(8)
	if cur, err := h.Store.Calls.Find(ctx, scoped(bson.M{"phone": phone}), copts); err == nil {
		_ = cur.All(ctx, &out.RecentCalls)
	}
	if out.RecentCalls == nil {
		out.RecentCalls = []models.Call{}
	}

	httpx.JSON(w, http.StatusOK, out)
}

// sortFavourites orders dishes by how many separate orders they appear in,
// then by quantity — a habit beats a one-off bulk order.
func sortFavourites(rows []callerFavourite) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0; j-- {
			a, b := rows[j-1], rows[j]
			if b.Times > a.Times || (b.Times == a.Times && b.Qty > a.Qty) {
				rows[j-1], rows[j] = b, a
				continue
			}
			break
		}
	}
}

// isActiveStatus reports whether an order is still in play — the ones a caller
// is most likely ringing about.
func isActiveStatus(s models.OrderStatus) bool {
	switch s {
	case models.StatusDelivered, models.StatusCancelled:
		return false
	}
	return true
}

// vipFloorNow is the spend that puts a customer in the top decile, computed
// over the whole base. Needed here because a lookup sees one customer and VIP
// is a relative rank — the cut cannot be worked out from the person in front
// of you.
func (h *Handler) vipFloorNow(r *http.Request) int {
	pipeline := []bson.M{
		{"$match": bson.M{"userId": bson.M{"$exists": true}}},
		{"$group": bson.M{"_id": "$userId", "total": bson.M{"$sum": bson.M{"$cond": []any{
			bson.M{"$eq": []any{"$status", string(models.StatusCancelled)}}, 0, "$total",
		}}}}},
	}
	cur, err := h.Store.Orders.Aggregate(r.Context(), pipeline)
	if err != nil {
		return 0
	}
	var rows []struct {
		Total int `bson:"total"`
	}
	if err := cur.All(r.Context(), &rows); err != nil {
		return 0
	}
	totals := make([]int, 0, len(rows))
	for _, row := range rows {
		totals = append(totals, row.Total)
	}
	return vipFloor(totals)
}

// ---- Taking an order over the phone ----

type adminCreateOrderRequest struct {
	createOrderRequest
	// The customer this order is for. When empty, the number in Customer.Phone
	// decides: an existing account is reused, and a caller who has none gets one
	// created for them.
	UserID string `json:"userId"`
	// Link the order to a call being logged right now, so the log row can say
	// what the conversation produced.
	CallID string `json:"callId"`
}

// AdminCreateOrder places an order on behalf of somebody on the phone.
//
// It runs the same composeOrder as the site: same menu, same sold-out list,
// same discounts, same delivery minimum. An operator is not allowed to sell at
// a price the site would not — the point of the call centre is to take the
// order, not to negotiate it.
//
// The one thing it may do that the site cannot is create the customer. A caller
// has proved nothing by ringing, so the account is written with
// authProvider "operator": it is a record of a phone order, not a sign-in. The
// person still has to pass the usual SMS check the first time they use the site
// themselves, and until then the account holds nothing but a name, a number and
// their order history.
func (h *Handler) AdminCreateOrder(w http.ResponseWriter, r *http.Request) {
	var req adminCreateOrderRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	phone, ok := normalizePhone(req.Customer.Phone)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "telefon raqam noto'g'ri")
		return
	}
	req.Customer.Phone = phone
	req.Customer.Name = clampText(req.Customer.Name, 60)
	if req.Customer.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "mijoz ismini yozing")
		return
	}

	userID, err := h.findOrCreateCaller(r, req.UserID, phone, req.Customer.Name)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	operator := h.adminName(r)
	order, status, err := h.composeOrder(r, req.createOrderRequest, userID, operator)
	if err != nil {
		httpx.Error(w, status, err.Error())
		return
	}

	// The address the operator just took down is worth keeping: the next call
	// from this number should offer it rather than ask for it again. Only added
	// when it is genuinely new, so a regular does not accumulate ten copies of
	// their own front door.
	if order.Type == "delivery" {
		h.rememberAddress(r, userID, order.Address)
	}

	h.logAction(r, ActOrderCreate, "order", order.ID.Hex(), "#"+order.Number,
		order.Customer.Phone)
	// Close the loop on the call that produced it.
	if id, err := objectID(strings.TrimSpace(req.CallID)); err == nil {
		_, _ = h.Store.Calls.UpdateByID(r.Context(), id, bson.M{"$set": bson.M{
			"outcome":     models.CallOutcomeOrder,
			"orderId":     order.ID,
			"orderNumber": order.Number,
			"userId":      userID,
			"updatedAt":   time.Now(),
		}})
	}
	// The bank link goes back with the order: an operator taking a card order
	// has to be able to send the caller somewhere to pay, and reading a
	// base64 checkout URL down the phone is not it.
	httpx.JSON(w, http.StatusCreated, h.withPayLink(r, order))
}

// findOrCreateCaller resolves the customer an operator is taking an order for.
func (h *Handler) findOrCreateCaller(
	r *http.Request, rawID, phone, name string,
) (primitive.ObjectID, error) {
	ctx := r.Context()
	if raw := strings.TrimSpace(rawID); raw != "" {
		id, err := objectID(raw)
		if err != nil {
			return primitive.NilObjectID, errNoCustomer
		}
		var u models.User
		if err := h.Store.Users.FindOne(ctx, bson.M{"_id": id}).Decode(&u); err != nil {
			return primitive.NilObjectID, errNoCustomer
		}
		return u.ID, nil
	}

	var existing models.User
	if err := h.Store.Users.FindOne(ctx, bson.M{"phone": phone}).Decode(&existing); err == nil {
		// A name the operator heard does not overwrite one the guest typed
		// themselves — but it does fill an empty one.
		if strings.TrimSpace(existing.FirstName) == "" && name != "" {
			_, _ = h.Store.Users.UpdateByID(ctx, existing.ID, bson.M{
				"$set": bson.M{"firstName": name, "updatedAt": time.Now()}})
		}
		return existing.ID, nil
	}

	now := time.Now()
	user := models.User{
		FirstName: name,
		Phone:     phone,
		// Not "phone": that is the site's SMS sign-in, and this account has
		// passed no check. See the note on AdminCreateOrder.
		AuthProvider: "operator",
		Addresses:    []models.UserAddress{},
		Source:       "phone",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	res, err := h.Store.Users.InsertOne(ctx, user)
	if err != nil {
		// ⚠️ The number is unique in the database now (see ensureUserPhoneUnique),
		// so a rejected insert here is almost always the race this endpoint could
		// always lose: two operators taking calls from the same household at once,
		// or a guest signing in on the site mid-call. Resolving it by using the
		// account that won is the correct answer — the alternative is telling an
		// operator with a customer on the line that their order failed.
		var existing models.User
		if e := h.Store.Users.FindOne(ctx, bson.M{"phone": phone}).Decode(&existing); e == nil {
			return existing.ID, nil
		}
		return primitive.NilObjectID, err
	}
	return res.InsertedID.(primitive.ObjectID), nil
}

// rememberAddress saves a delivery address an operator typed onto the customer
// profile, unless they already have it.
func (h *Handler) rememberAddress(r *http.Request, userID primitive.ObjectID, addr models.OrderAddress) {
	if userID.IsZero() || strings.TrimSpace(addr.Text) == "" {
		return
	}
	var u models.User
	if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": userID}).Decode(&u); err != nil {
		return
	}
	for _, a := range u.Addresses {
		// Same text, or near enough the same point (~50 m) that it is the same
		// door reached two different ways.
		if strings.EqualFold(strings.TrimSpace(a.Text), strings.TrimSpace(addr.Text)) {
			return
		}
		if a.Lat != 0 && addr.Lat != 0 && haversineKm(a.Lat, a.Lng, addr.Lat, addr.Lng) < 0.05 {
			return
		}
	}
	if len(u.Addresses) >= 12 {
		return
	}
	_, _ = h.Store.Users.UpdateByID(r.Context(), userID, bson.M{
		"$push": bson.M{"addresses": models.UserAddress{
			Text: addr.Text, Lat: addr.Lat, Lng: addr.Lng, Comment: addr.Comment,
		}},
		"$set": bson.M{"updatedAt": time.Now()},
	})
}

// adminID is the signed-in operator's id, or a zero id when the token carries
// no usable one. The name goes with it through adminName (staffattendance.go).
func (h *Handler) adminID(r *http.Request) primitive.ObjectID {
	admin, err := h.adminUser(r)
	if err != nil {
		return primitive.NilObjectID
	}
	return admin.ID
}

// errNoCustomer is what a lookup by id answers with when the operator's screen
// is pointing at an account that is no longer there.
var errNoCustomer = errors.New("mijoz topilmadi")
