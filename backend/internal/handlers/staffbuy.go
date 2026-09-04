package handlers

// ---- The market run, recorded where it happens ----
//
// ⚠️ **The delivery a restaurant actually has, and the one the panel was built
// for the least.** A supplier sends an invoice; a market run sends somebody
// with cash at six in the morning, and what they brought back was written on a
// scrap of paper, carried to an office, and typed in by whoever had time. Half
// the time it was typed in a day late, and once in a while it was not typed in
// at all — and every figure downstream is built on it: the shelf, the shopping
// list that decides the *next* run, the cost of every dish, the stop list.
//
// ⚠️ **This is not a second way to record a delivery.** It writes the same
// `purchase` document the panel writes, through the same price resolver, with
// the same freezing rules — see AdminCreatePurchase. What differs is only who
// is standing where, and that the buyer's name is already on the record because
// the model has always carried it.
//
// ⚠️ **The buyer never sees the panel.** That is the opposite of the
// storekeeper, whose whole permission is a panel login (stocklogin.go), and it
// is deliberate: this account travels to a market on a phone that is out of the
// building all morning. It reaches exactly the four endpoints below.

import (
	"context"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// buyDenial says why this employee may not record a purchase, or "" if they
// may.
//
// Two refusals with different words, because they send the person somewhere
// different: one to whoever switched the account off, one to their manager.
func buyDenial(s models.Staff) string {
	if !s.IsActive {
		return "hisob o'chirilgan — ma'muriyat bilan bog'laning"
	}
	if !s.Can(models.PermBuy) {
		return "xarid kiritishga ruxsat berilmagan — administratorga murojaat qiling"
	}
	return ""
}

// buyStaff authenticates the request and checks the permission.
func (h *Handler) buyStaff(w http.ResponseWriter, r *http.Request) (models.Staff, bool) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return models.Staff{}, false
	}
	if msg := buyDenial(s); msg != "" {
		httpx.Error(w, http.StatusForbidden, msg)
		return models.Staff{}, false
	}
	return s, true
}

// StaffBuyList is what this branch is short of, as the panel computes it.
//
// ⚠️ **The list the buyer leaves with is the list the owner reads**, from one
// function (`shoppingList`). The alternative is two answers to "are we out of
// beef" — one on a phone at a market and one in an office — and the argument
// happens after the money is spent.
func (h *Handler) StaffBuyList(w http.ResponseWriter, r *http.Request) {
	s, ok := h.buyStaff(w, r)
	if !ok {
		return
	}
	// ⚠️ The branch comes off the employee, never the request — the rule every
	// staff endpoint follows. A buyer who typed another branch's id would
	// otherwise be shopping for a kitchen they have never stood in.
	scope, brand := h.staffStockScope(r, s)
	groups, total, since, err := h.shoppingList(r, scope, s.BranchID, brand)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"groups": groups,
		"cost":   total,
		// ⚠️ The caveat travels with the number, the same way it does on the
		// balance screen: a list computed from shelves nobody has counted is a
		// list somebody should sanity-check before spending against it.
		"since": since,
	})
}

// buyCatalogRow is one ingredient as the buyer's phone needs it.
type buyCatalogRow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// The purchase unit — kilos, litres, pieces. What is bought and counted in,
	// never the recipe's grams.
	Unit string `json:"unit"`
	// ⚠️ **What it cost last time, shown beside the box the buyer types into.**
	// This is the only guard the price has: a market price typed on a phone
	// reprices every dish that uses the ingredient, and `9 000` entered as
	// `90 000` looks exactly like an ordinary number afterwards. Nothing
	// downstream can tell the two apart — but a person standing at the stall
	// can, if the last one is in front of them.
	LastPrice int `json:"lastPrice"`
}

// StaffBuyCatalog is the ingredient list the buyer picks from.
func (h *Handler) StaffBuyCatalog(w http.ResponseWriter, r *http.Request) {
	s, ok := h.buyStaff(w, r)
	if !ok {
		return
	}
	_, brand := h.staffStockScope(r, s)
	now := time.Now()
	rows := []buyCatalogRow{}
	for _, in := range h.scopedIngredients(r.Context(), brand) {
		// ⚠️ A prep item is cooked, not bought. Offering sauce at a market is
		// how a buyer records a delivery of something nobody sells, and the
		// shelf then holds a tub that was never on it.
		if in.DerivedOnly() {
			continue
		}
		rows = append(rows, buyCatalogRow{
			ID: in.ID.Hex(), Name: in.Name, Unit: in.Unit,
			LastPrice: in.PriceAt(now),
		})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	httpx.JSON(w, http.StatusOK, map[string]any{"ingredients": rows})
}

// buyRequest is one market run as the phone sends it.
type buyRequest struct {
	// ⚠️ **The id the phone minted, and the whole of offline safety.** A market
	// has no signal; the app holds the run on its own disk and sends it when
	// the connection returns, retried by a program that cannot know whether the
	// first attempt arrived. Without an id from the phone a retry is a second
	// delivery — counted twice on the shelf and paid for twice in the reports.
	// Same field and same reason as an offline check's `clientId`.
	ClientID string `json:"clientId"`
	// Free text: which market, which stall. Optional, like the panel's.
	Supplier string             `json:"supplier"`
	Note     string             `json:"note"`
	Lines    []buyRequestLine   `json:"lines"`
	At       *time.Time         `json:"at"`
	SupplyID primitive.ObjectID `json:"supplierId"`
}

type buyRequestLine struct {
	IngredientID string  `json:"ingredientId"`
	Qty          float64 `json:"qty"`
	// Per purchase unit, in so'm.
	Price int `json:"price"`
	// A name typed for something the catalogue does not have yet.
	//
	// ⚠️ **The quietest failure this feature has.** Refuse it and the buyer
	// cannot record half a market run, so they stop recording any of it — the
	// lesson this codebase has already paid for with the supplier field and
	// with the void reason. Accept it freely and the catalogue fills with
	// "pomidor", "Pomidor" and "tomat", none of which any tech card points at.
	// So it is accepted and **marked**, and the panel is given a list to merge.
	NewName string `json:"newName"`
}

// StaffBuyCreate records one market run.
func (h *Handler) StaffBuyCreate(w http.ResponseWriter, r *http.Request) {
	s, ok := h.buyStaff(w, r)
	if !ok {
		return
	}
	var req buyRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ClientID != "" {
		var again models.Purchase
		if err := h.Store.Purchases.FindOne(r.Context(),
			bson.M{"clientId": req.ClientID}).Decode(&again); err == nil {
			httpx.JSON(w, http.StatusOK, map[string]any{"purchase": again, "already": true})
			return
		}
	}
	created, p, err := h.recordMarketRun(r, s, req)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"purchase": p,
		// What now costs something different, and what had to be invented. Both
		// are things the person who just pressed the button should see rather
		// than discover from a manager later.
		"repriced": h.applyDeliveryPrices(r, p),
		"created":  created,
	})
}

// recordMarketRun writes one delivery from a phone, whichever screen it came
// from.
//
// ⚠️ **One function because there are now two doors into it** — a free-form
// market run and a finished shopping list — and they must produce the same
// document. Two writers would mean two answers to "was this run paid for", "was
// it stamped with the trip's own time", "what happened to a name the catalogue
// does not have": three rules, each of which took a paragraph to get right.
func (h *Handler) recordMarketRun(
	r *http.Request, s models.Staff, req buyRequest,
) ([]string, models.Purchase, error) {
	ctx := r.Context()

	// ⚠️ **Answered before anything is written, and answered with the row that
	// already exists.** A phone that retried because it never saw our reply
	// must be told the delivery is here, not given a second one.
	if req.ClientID != "" {
		var again models.Purchase
		if err := h.Store.Purchases.FindOne(ctx,
			bson.M{"clientId": req.ClientID}).Decode(&again); err == nil {
			return nil, again, nil
		}
	}

	_, brand := h.staffStockScope(r, s)
	lines, created, err := h.buyLines(ctx, s, brand, req.Lines)
	if err != nil {
		return nil, models.Purchase{}, err
	}

	now := time.Now()
	at := now
	if req.At != nil && !req.At.IsZero() {
		// ⚠️ A run cannot be dated into the future: prices from a day that has
		// not happened apply to nothing today and to everything from then on —
		// a mistake with a delayed effect nobody would connect back to a phone.
		// The panel's form draws the same line.
		if req.At.After(now) {
			at = now
		} else {
			at = *req.At
		}
	}

	p := models.Purchase{
		ClientID: req.ClientID,
		BranchID: s.BranchID,
		At:       at,
		Note:     clampText(req.Note, 200),
		Lines:    lines,
		// ⚠️ **A market run is paid on the spot, by definition**, and saying so
		// is not a convenience. `paid` is what the supplier-debt report reads,
		// and an unpaid run has no supplier to owe — so every run recorded here
		// would sit in that report forever as money owed to nobody, in a total
		// an owner is meant to act on. The panel's form asks because an invoice
		// genuinely can be unpaid; a man with cash at a stall cannot.
		Paid:   true,
		PaidAt: &now,
		// ⚠️ **The name was always in the model and nothing filled it from a
		// phone.** "Who brought this and when" is the first question asked about
		// a market run, and until now the answer for one was blank.
		CreatedBy:   s.Name,
		CreatedByID: s.ID,
		CreatedAt:   now,
	}
	p.SupplierID, p.Supplier = h.supplierNameFor(r, Scope{BrandID: brand}, req.SupplyID, req.Supplier)
	for _, l := range lines {
		p.Total += l.Sum()
	}

	res, err := h.Store.Purchases.InsertOne(ctx, p)
	if err != nil {
		// A duplicate key here is two sends crossing, and the first one won —
		// the till's sync draws the same conclusion.
		if mongo.IsDuplicateKeyError(err) {
			var again models.Purchase
			if e := h.Store.Purchases.FindOne(ctx,
				bson.M{"clientId": req.ClientID}).Decode(&again); e == nil {
				return nil, again, nil
			}
		}
		return nil, models.Purchase{}, err
	}
	p.ID = oidOf(res.InsertedID)
	h.notifyPurchase(p, s, created)
	return created, p, nil
}

// buyLines turns what the phone sent into purchase lines, inventing the
// ingredients it named and could not find.
func (h *Handler) buyLines(
	ctx context.Context, s models.Staff, brand primitive.ObjectID, in []buyRequestLine,
) ([]models.PurchaseLine, []string, error) {
	out := make([]models.PurchaseLine, 0, len(in))
	created := []string{}
	for _, l := range in {
		if l.Qty <= 0 || l.Price < 0 {
			continue
		}
		id, err := primitive.ObjectIDFromHex(strings.TrimSpace(l.IngredientID))
		if err != nil || id.IsZero() {
			name := clampText(strings.TrimSpace(l.NewName), 80)
			if name == "" {
				continue
			}
			newID, err := h.newBoughtIngredient(ctx, brand, name, l.Price)
			if err != nil {
				return nil, nil, err
			}
			id = newID
			created = append(created, name)
		}
		out = append(out, models.PurchaseLine{
			IngredientID: id, Qty: l.Qty, Price: l.Price,
		})
	}
	if len(out) == 0 {
		return nil, nil, errNoPurchaseLines
	}
	return out, created, nil
}

// newBoughtIngredient adds something the market had and the catalogue did not.
//
// ⚠️ **Matched against the existing names first, case-insensitively.** The
// commonest way this goes wrong is not a genuinely new product — it is
// "Pomidor" typed where "pomidor" already exists, and a second row that every
// tech card ignores. The phone suggests as somebody types; this is the guard
// behind it, for the times they typed past the suggestion.
//
// ⚠️ **Marked for review rather than dropped into the catalogue silently.** An
// ingredient invented at a market has no minimum, no store and no card — it is
// half a record, and the owner is the only person who can finish it.
func (h *Handler) newBoughtIngredient(
	ctx context.Context, brand primitive.ObjectID, name string, price int,
) (primitive.ObjectID, error) {
	filter := bson.M{"name": bson.M{
		"$regex": "^" + regexp.QuoteMeta(name) + "$", "$options": "i",
	}}
	if !brand.IsZero() {
		filter["brandId"] = brand
	}
	var found models.Ingredient
	if err := h.Store.Ingredients.FindOne(ctx, filter).Decode(&found); err == nil {
		return found.ID, nil
	}
	now := time.Now()
	in := models.Ingredient{
		BrandID: brand,
		Name:    name,
		// ⚠️ **Pieces, not kilos.** A guessed unit is worse than a plain one: a
		// card written against "kg" for something sold by the bunch produces a
		// cost that is wrong by three orders of magnitude and looks deliberate.
		// The owner sets it when they finish the record.
		Unit:      models.UnitPcs,
		Price:     price,
		NeedsCare: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	res, err := h.Store.Ingredients.InsertOne(ctx, in)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return oidOf(res.InsertedID), nil
}

// StaffBuyHistory is what this buyer has recorded, most recent first.
//
// ⚠️ **Their own runs, not the branch's.** The screen answers "did my delivery
// go through" — the question a phone with a bad signal creates — and a list of
// everybody's deliveries answers a different one, on a device that leaves the
// building.
func (h *Handler) StaffBuyHistory(w http.ResponseWriter, r *http.Request) {
	s, ok := h.buyStaff(w, r)
	if !ok {
		return
	}
	cur, err := h.Store.Purchases.Find(r.Context(),
		bson.M{"branchId": s.BranchID, "createdById": s.ID},
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(50))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.Purchase{}
	_ = cur.All(r.Context(), &rows)
	httpx.JSON(w, http.StatusOK, map[string]any{"purchases": rows})
}

// notifyPurchase tells the owner a market run was recorded.
//
// ⚠️ **Because the shelf moved without them, and they chose that.** A run
// entered from a phone raises the stock, rewrites prices and can put dishes
// back on the menu — all before anybody in an office has seen it. That is the
// right trade (nobody approves a delivery at six in the morning, and the goods
// are physically in the building), and the price of it is that the owner is
// told rather than left to find out at a stocktake.
//
// ⚠️ **A notification, never a `LossAlert`.** models/alert.go draws that line:
// the bell is for what is unusual as a single event, and a market run is the
// most ordinary thing a restaurant does. Filing it there would spend the daily
// ceiling on routine and mute the channel for the night it matters.
//
// ⚠️ **In its own goroutine, and it cannot fail the thing that triggered it.**
// The buyer is standing at a stall watching a spinner; a Telegram round trip to
// another country must not be why their delivery does not save. Same rule the
// alert bell and the print queue follow.
func (h *Handler) notifyPurchase(p models.Purchase, s models.Staff, created []string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		h.notifyAdmins(p.BranchID, true, func(lang string) (string, string) {
			return buyTitle(lang), buyText(p, s, created, lang)
		}, map[string]any{"type": "purchase", "id": p.ID.Hex()})

		_ = h.sendToOwners(ctx, p.BranchID, buyText(p, s, created, h.notifyLang(ctx)))
	}()
}

func buyTitle(lang string) string {
	return tr{"Bozordan kirim", "Приход с рынка", "Market delivery"}.in(lang)
}

// buyText is the sentence the owner reads on a phone.
//
// ⚠️ **It leads with the money and names the person.** "A delivery was
// recorded" is a sentence somebody skims; the amount and the name are what
// decides whether this one is worth opening — the same shape every loss alert
// follows, and for the same reason.
func buyText(p models.Purchase, s models.Staff, created []string, lang string) string {
	var b strings.Builder
	b.WriteString(buyTitle(lang))
	b.WriteString(": ")
	b.WriteString(formatSum(p.Total))
	b.WriteString(" · ")
	b.WriteString(s.Name)
	b.WriteString(" · ")
	b.WriteString(tr{
		itoa(len(p.Lines)) + " qator",
		itoa(len(p.Lines)) + " строк",
		itoa(len(p.Lines)) + " lines",
	}.in(lang))
	// ⚠️ Said in the message rather than left for somebody to notice in the
	// catalogue. A new ingredient is a half-record — no unit, no minimum, no
	// card — and the only person who can finish it is the one reading this.
	if len(created) > 0 {
		b.WriteString("\n")
		b.WriteString(tr{
			"Yangi masalliq (to'ldirish kerak): ",
			"Новый ингредиент (нужно заполнить): ",
			"New ingredient (needs finishing): ",
		}.in(lang))
		b.WriteString(strings.Join(created, ", "))
	}
	return b.String()
}
