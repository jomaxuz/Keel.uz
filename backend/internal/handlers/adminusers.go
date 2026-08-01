package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// adminUserRow is a customer plus the aggregates the dashboard shows.
type adminUserRow struct {
	models.User
	OrdersCount  int        `json:"ordersCount"`
	OrdersTotal  int        `json:"ordersTotal"`
	AvgOrder     int        `json:"avgOrder"`
	FirstOrderAt *time.Time `json:"firstOrderAt"`
	LastOrderAt  *time.Time `json:"lastOrderAt"`
	// Days since the last order — the single number that says whether this
	// customer needs anything doing about them.
	DaysSinceLast *int `json:"daysSinceLast"`
	// Which groups they fall into. Several at once is normal and meaningful:
	// a VIP who has gone quiet is both, and that is the pair worth acting on.
	Segments []string `json:"segments"`
}

// AdminListUsers returns site customers with their order counts.
// Supports ?q= (name/phone search) and ?limit=.
func (h *Handler) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	filter := bson.M{}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		// Case-insensitive "contains" across the human-readable fields.
		rx := bson.M{"$regex": q, "$options": "i"}
		filter["$or"] = []bson.M{
			{"firstName": rx}, {"lastName": rx}, {"phone": rx},
		}
	}
	limit := int64(200)
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 && v <= 500 {
		limit = int64(v)
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(limit)
	cur, err := h.Store.Users.Find(r.Context(), filter, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var users []models.User
	if err := cur.All(r.Context(), &users); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// One aggregation for all order counts instead of a query per user.
	type agg struct {
		ID    any       `bson:"_id"`
		Count int       `bson:"count"`
		Total int       `bson:"total"`
		First time.Time `bson:"first"`
		Last  time.Time `bson:"last"`
	}
	counts := map[string]agg{}
	// Cancelled orders are counted but not banked, the same convention the
	// dashboard uses: the guest did order, the till never saw the money.
	pipeline := []bson.M{
		{"$match": bson.M{"userId": bson.M{"$exists": true}}},
		{"$group": bson.M{
			"_id":   "$userId",
			"count": bson.M{"$sum": 1},
			"total": bson.M{"$sum": bson.M{"$cond": []any{
				bson.M{"$eq": []any{"$status", string(models.StatusCancelled)}},
				0, "$total",
			}}},
			"first": bson.M{"$min": "$createdAt"},
			"last":  bson.M{"$max": "$createdAt"},
		}},
	}
	if cur, err := h.Store.Orders.Aggregate(r.Context(), pipeline); err == nil {
		var rows []agg
		if err := cur.All(r.Context(), &rows); err == nil {
			for _, row := range rows {
				counts[toHex(row.ID)] = row
			}
		}
	}

	// VIP is the top spenders of *this* restaurant's base, so the cut has to be
	// worked out from everyone — not from the page being shown.
	allTotals := make([]int, 0, len(counts))
	for _, a := range counts {
		allTotals = append(allTotals, a.Total)
	}
	floor := vipFloor(allTotals)
	now := time.Now()
	unhappy := h.unhappyUsers(r)

	out := make([]adminUserRow, 0, len(users))
	for _, u := range users {
		row := adminUserRow{User: u}
		facts := customerFacts{Birthday: u.Birthday, Unhappy: unhappy[u.ID.Hex()]}
		if a, ok := counts[u.ID.Hex()]; ok {
			row.OrdersCount, row.OrdersTotal = a.Count, a.Total
			if a.Count > 0 {
				row.AvgOrder = a.Total / a.Count
			}
			if !a.First.IsZero() {
				first := a.First
				row.FirstOrderAt = &first
			}
			if !a.Last.IsZero() {
				last := a.Last
				row.LastOrderAt = &last
				days := int(now.Sub(last).Hours() / 24)
				row.DaysSinceLast = &days
			}
			facts.OrdersCount, facts.OrdersTotal = a.Count, a.Total
			facts.FirstOrder, facts.LastOrder = row.FirstOrderAt, row.LastOrderAt
		}
		row.Segments = segmentsFor(facts, floor, now)
		if row.Segments == nil {
			row.Segments = []string{}
		}
		out = append(out, row)
	}

	// Filters are applied here rather than in the query because a segment is
	// computed from the orders, not stored on the customer.
	if seg := strings.TrimSpace(r.URL.Query().Get("segment")); seg != "" {
		kept := out[:0]
		for _, row := range out {
			for _, s := range row.Segments {
				if s == seg {
					kept = append(kept, row)
					break
				}
			}
		}
		out = kept
	}
	if tag := strings.TrimSpace(r.URL.Query().Get("tag")); tag != "" {
		kept := out[:0]
		for _, row := range out {
			for _, x := range row.Tags {
				if x == tag {
					kept = append(kept, row)
					break
				}
			}
		}
		out = kept
	}
	httpx.JSON(w, http.StatusOK, out)
}

// AdminGetUser returns one customer with their full order history — the view
// used when a guest calls about an order they placed some time ago.
func (h *Handler) AdminGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var user models.User
	if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": id}).Decode(&user); err != nil {
		httpx.Error(w, http.StatusNotFound, "user not found")
		return
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(200)
	cur, err := h.Store.Orders.Find(r.Context(), bson.M{"userId": id}, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var orders []models.Order
	if err := cur.All(r.Context(), &orders); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if orders == nil {
		orders = []models.Order{}
	}

	total := 0
	delivered := 0
	cancelled := 0
	for _, o := range orders {
		total += o.Total
		switch o.Status {
		case models.StatusDelivered:
			delivered++
		case models.StatusCancelled:
			cancelled++
		}
	}
	var lastAt, firstAt *time.Time
	if len(orders) > 0 {
		l := orders[0].CreatedAt
		f := orders[len(orders)-1].CreatedAt
		lastAt, firstAt = &l, &f
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"user":   user,
		"orders": orders,
		"stats": map[string]any{
			"ordersCount":  len(orders),
			"ordersTotal":  total,
			"delivered":    delivered,
			"cancelled":    cancelled,
			"lastOrderAt":  lastAt,
			"firstOrderAt": firstAt,
		},
	})
}

func toHex(v any) string {
	if oid, ok := v.(interface{ Hex() string }); ok {
		return oid.Hex()
	}
	return ""
}

// adminUserPatch is what an operator may change about a customer.
//
// Deliberately narrow: the name and the phone belong to the guest — the phone
// was proved by SMS and changing it from the panel would break the one thing
// the account is anchored to. Everything here is the restaurant's own notes.
type adminUserPatch struct {
	Note     *string   `json:"note"`
	Tags     *[]string `json:"tags"`
	Source   *string   `json:"source"`
	Birthday *string   `json:"birthday"`
}

// AdminUpdateUser saves the restaurant's notes about a customer.
func (h *Handler) AdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req adminUserPatch
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	set := bson.M{"updatedAt": time.Now()}
	if req.Note != nil {
		set["note"] = clampText(*req.Note, 500)
	}
	if req.Source != nil {
		set["source"] = clampText(*req.Source, 40)
	}
	if req.Birthday != nil {
		// Stored as "MM-DD": the day is all a greeting needs, and a birth year
		// is personal data the restaurant has no use for. A date input sends
		// "YYYY-MM-DD", so the year is dropped here rather than stored and
		// ignored.
		b := strings.TrimSpace(*req.Birthday)
		if len(b) == 10 && b[4] == '-' {
			b = b[5:]
		}
		if b != "" && !validMonthDay(b) {
			httpx.Error(w, http.StatusBadRequest, "sana formati noto'g'ri")
			return
		}
		set["birthday"] = b
	}
	if req.Tags != nil {
		tags := make([]string, 0, len(*req.Tags))
		seen := map[string]bool{}
		for _, t := range *req.Tags {
			t = clampText(t, 30)
			if t == "" || seen[t] {
				continue
			}
			seen[t] = true
			tags = append(tags, t)
		}
		if len(tags) > 12 {
			tags = tags[:12]
		}
		set["tags"] = tags
	}

	if _, err := h.Store.Users.UpdateByID(r.Context(), id, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var u models.User
	if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": id}).Decode(&u); err != nil {
		httpx.Error(w, http.StatusNotFound, "mijoz topilmadi")
		return
	}
	h.logAction(r, ActUserUpdate, "user", id.Hex(),
		strings.TrimSpace(u.FirstName+" "+u.LastName), "")
	httpx.JSON(w, http.StatusOK, u)
}

// validMonthDay checks a "MM-DD" string against a real calendar, so 02-31
// cannot be stored and then never match.
func validMonthDay(s string) bool {
	if len(s) != 5 || s[2] != '-' {
		return false
	}
	// 2024 is a leap year, so 02-29 is accepted.
	_, err := time.Parse("2006-01-02", "2024-"+s)
	return err == nil
}

// AdminTags lists every tag currently in use, so the panel can offer them
// instead of letting a typo quietly create a second "VIP " group.
func (h *Handler) AdminTags(w http.ResponseWriter, r *http.Request) {
	values, err := h.Store.Users.Distinct(r.Context(), "tags", bson.M{})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if s, ok := v.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	httpx.JSON(w, http.StatusOK, out)
}
