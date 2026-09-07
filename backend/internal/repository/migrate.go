package repository

import (
	"context"
	"log"
	"strings"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EnsureBrandAndBranch gives every install the one brand and one branch the
// rest of the code can rely on.
//
// A restaurant that has been running since before brands existed keeps working
// untouched: its profile becomes brand #1, its address, hours, delivery zones
// and floor plan become branch #1, and every order, courier and booking it
// already had is attached to that branch. Nothing is deleted — the old fields
// stay on the company document, so a rollback loses nothing.
//
// Safe to run on every boot: it does nothing once a brand exists.
// ⚠️ **Taken as a parameter rather than read from the environment here.** This
// runs on every boot of every tenant, including the twelve that already have a
// brand — and a function that reached for `os.Getenv` would be a second place
// the business type is decided, which is how one brand ends up disagreeing with
// itself. The caller reads it once; this only ever applies it to a brand it is
// creating for the first time.
func EnsureBrandAndBranch(ctx context.Context, s *Store, biz models.BusinessType) error {
	count, err := s.Brands.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	// The company profile, if this install has one yet.
	var rest models.Restaurant
	_ = s.Restaurant.FindOne(ctx, bson.M{}).Decode(&rest)

	now := time.Now()
	name := rest.Name
	if name == "" {
		name = "Restoran"
	}

	brand := models.Brand{
		Name:        name,
		Slug:        "main",
		Description: rest.Description,
		LogoURL:     rest.LogoURL,
		CoverURL:    rest.CoverURL,
		Content:     rest.Content,
		Theme:       rest.Theme,
		// ⚠️ The template, applied to the first brand and only here. Empty is a
		// restaurant, which is what every install that predates the field is.
		BusinessType: biz,
		// ⚠️ **`Booking` still comes from the profile, not from the template.**
		// An install being migrated already answered that question; overwriting
		// it with a default would switch bookings on for a restaurant that had
		// deliberately turned them off.
		Features: withDefaults(biz, models.BrandFeatures{
			Delivery: true,
			Pickup:   true,
			DineIn:   true,
			Booking:  rest.Booking.Enabled,
		}),
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	brandRes, err := s.Brands.InsertOne(ctx, brand)
	if err != nil {
		return err
	}
	brandID, _ := brandRes.InsertedID.(primitive.ObjectID)

	branch := models.Branch{
		BrandID:      brandID,
		Name:         name,
		Phones:       rest.Phones,
		Address:      rest.Address,
		WorkingHours: rest.WorkingHours,
		Delivery:     rest.Delivery,
		Booking:      rest.Booking,
		PrepMinutes:  30,
		// Set here as well as in EnsureStaffDefaults: Go writes the zero value
		// rather than omitting the field, so a branch created by this migration
		// would have `staffRadiusM: 0` — which reads as "no geofence at all"
		// and is invisible to the `$exists: false` backfill.
		StaffRadiusM: 50,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	branchRes, err := s.Branches.InsertOne(ctx, branch)
	if err != nil {
		return err
	}
	branchID, _ := branchRes.InsertedID.(primitive.ObjectID)

	// Everything that existed before belongs to that first branch/brand.
	missing := bson.M{"brandId": bson.M{"$exists": false}}
	setBrand := bson.M{"$set": bson.M{"brandId": brandID}}
	for _, coll := range []*mongo.Collection{s.Categories, s.Menu} {
		if _, err := coll.UpdateMany(ctx, missing, setBrand); err != nil {
			return err
		}
	}
	if _, err := s.Orders.UpdateMany(ctx, bson.M{"branchId": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"brandId": brandID, "branchId": branchID}},
	); err != nil {
		return err
	}
	for _, coll := range []*mongo.Collection{s.Couriers, s.Reservations} {
		if _, err := coll.UpdateMany(ctx, bson.M{"branchId": bson.M{"$exists": false}},
			bson.M{"$set": bson.M{"branchId": branchID}},
		); err != nil {
			return err
		}
	}

	log.Printf("migrated to brand %q / branch %q", brand.Name, branch.Name)
	return nil
}

// EnsureStaffDefaults gives branches that predate staff attendance a working
// geofence.
//
// The field defaults to 0, and 0 means "no check" — so without this every
// existing branch would quietly accept a clock-in from the other side of the
// city. Only branches that have never seen the field are touched, so an owner
// who deliberately switched the check off keeps it off.
func EnsureStaffDefaults(ctx context.Context, s *Store) error {
	_, err := s.Branches.UpdateMany(ctx,
		bson.M{"staffRadiusM": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"staffRadiusM": 50}},
	)
	return err
}

// EnsureSoldOutArrays turns a branch's two stop lists into real arrays.
//
// ⚠️ **A nil slice marshals to BSON `null`, not `[]`** — the same Go habit that
// bites in JSON, one layer down and with a sharper edge: `$addToSet` and `$pull`
// refuse a non-array field outright ("Cannot apply $addToSet to non-array field").
// So every branch written before anything had ever run out — which is every
// branch, on the day it is created — could not be given a first sold-out dish at
// all. The failure surfaced at the counter, on the first tap, as a save error.
//
// Fixed in three places, because one is not enough: here for documents that
// already exist, at creation time so new branches start with `[]`, and in the
// handler itself, which cannot assume this migration ever ran against the
// database it is talking to.
func EnsureSoldOutArrays(ctx context.Context, s *Store) error {
	for _, field := range []string{"soldOut", "posSoldOut"} {
		if _, err := s.Branches.UpdateMany(ctx,
			bson.M{field: bson.M{"$not": bson.M{"$type": "array"}}},
			bson.M{"$set": bson.M{field: []primitive.ObjectID{}}},
		); err != nil {
			return err
		}
	}
	return nil
}

// EnsureQueuedAt backfills "when the kitchen may start on this order".
//
// Orders written before online payment existed were all settled at the door, so
// the answer for every one of them is the moment they were placed. Orders still
// waiting on a bank are skipped deliberately — setting queuedAt on those would
// call the kitchen to food nobody has paid for.
func EnsureQueuedAt(ctx context.Context, s *Store) error {
	_, err := s.Orders.UpdateMany(ctx,
		bson.M{
			"queuedAt":      bson.M{"$exists": false},
			"paymentStatus": bson.M{"$ne": models.PayPending},
		},
		[]bson.M{{"$set": bson.M{"queuedAt": "$createdAt"}}},
	)
	return err
}

// EnsureReviewsBand gives a hand-drawn design the guests' reviews band.
//
// ⚠️ **Nobody removed it — it was never offered.** Designs drawn in the console
// were seeded from a list that left the band out, and the console's editor did
// not know the block existed at all, so no operator has ever chosen to leave it
// off. That makes adding it safe in a way it would not be later: there is no
// deliberate absence here to overrule.
//
// It is added once, to designs that lack it. An operator who removes it after
// this runs keeps it removed — the migration does not put it back, because by
// then the absence means something.
//
// The band renders nothing until the restaurant switches reviews on, so this
// changes no page that was not already asking for one.
func EnsureReviewsBand(ctx context.Context, s *Store) error {
	// Every design is visited exactly once, marked whether or not it needed the
	// band. Matching on "has no reviews band" alone would find a design an
	// operator had just removed it from, every single boot.
	cur, err := s.Designs.Find(ctx, bson.M{"reviewsBandAdded": bson.M{"$ne": true}})
	if err != nil {
		return err
	}
	var designs []models.PageDesign
	if err := cur.All(ctx, &designs); err != nil {
		return err
	}
	for _, d := range designs {
		set := bson.M{"reviewsBandAdded": true}
		if next, ok := WithReviewsBand(d.Sections); ok {
			set["sections"] = next
			set["updatedAt"] = time.Now()
			log.Printf("migrate: reviews band added to design %s (%s)", d.ID, d.Status)
		}
		if _, err := s.Designs.UpdateOne(ctx,
			bson.M{"_id": d.ID}, bson.M{"$set": set}); err != nil {
			return err
		}
	}
	return nil
}

// WithReviewsBand puts the guests' reviews where the site's own fallback puts
// them: after the menu, before the address. Reports false when the design
// already has one, or has no bands at all.
//
// ⚠️ Appended at the end when there is no address band — a layout an operator
// built differently is theirs, and the only wrong answer is to guess at a
// middle. A footer is the one band nothing belongs under, so the band goes
// above it rather than after it.
//
// An empty design is the console's "not drawn yet"; seeding a lone reviews
// band into it would render a page that is nothing but other people's opinions.
func WithReviewsBand(sections []models.DesignSection) ([]models.DesignSection, bool) {
	if len(sections) == 0 || indexOfBand(sections, models.BlockReviews) >= 0 {
		return sections, false
	}
	at := indexOfBand(sections, models.BlockHoursAddress)
	if at < 0 {
		at = indexOfBand(sections, models.BlockFooter)
	}
	if at < 0 {
		at = len(sections)
	}
	out := make([]models.DesignSection, 0, len(sections)+1)
	out = append(out, sections[:at]...)
	out = append(out, models.DesignSection{
		Type: models.BlockReviews, Variant: "cards", Span: 12,
	})
	return append(out, sections[at:]...), true
}

func indexOfBand(sections []models.DesignSection, kind string) int {
	for i, s := range sections {
		if s.Type == kind {
			return i
		}
	}
	return -1
}

// EnsureIndexes creates the lookups the scoped queries lean on. Cheap and
// idempotent; Mongo ignores an index it already has.
func EnsureIndexes(ctx context.Context, s *Store) error {
	specs := []struct {
		coll *mongo.Collection
		keys bson.D
	}{
		{s.Branches, bson.D{{Key: "brandId", Value: 1}}},
		{s.Categories, bson.D{{Key: "brandId", Value: 1}, {Key: "sortOrder", Value: 1}}},
		{s.Menu, bson.D{{Key: "brandId", Value: 1}, {Key: "categoryId", Value: 1}}},
		// ---- The three reads every visitor makes ----
		//
		// ⚠️ **The filter field has to sit between the brand and the sort, or
		// the sort is not the index's.** `/menu` asks for one brand's available
		// dishes ordered by `sortOrder`; against `(brandId, categoryId)` Mongo
		// can only use the brand, then sorts the result in memory — and an
		// in-memory sort is both the slowest part of the query and the one that
		// **fails outright** past 32 MB, which is a menu with photographs and
		// long descriptions rather than an unreasonable menu.
		//
		// The load test of 2026-09-03 named mongod as the one saturated
		// component on the box, and these are the queries it was running: the
		// menu, the categories and the promotions are three of the five reads
		// on every single visit. The cache in front of them (see
		// middleware/pubcache.go) means they run once per window instead of once
		// per guest — this is so that the once is cheap too, and so that the
		// first request after every deploy, when the cache is empty and everyone
		// arrives at once, is not the slow one.
		{s.Menu, bson.D{
			{Key: "brandId", Value: 1},
			{Key: "isAvailable", Value: 1},
			{Key: "sortOrder", Value: 1},
		}},
		{s.Categories, bson.D{
			{Key: "brandId", Value: 1},
			{Key: "isActive", Value: 1},
			{Key: "sortOrder", Value: 1},
		}},
		// No sort on this one — the handler filters what is live today in Go,
		// because "live" depends on the hour and on the branch.
		{s.Promotions, bson.D{{Key: "brandId", Value: 1}, {Key: "isActive", Value: 1}}},
		{s.Orders, bson.D{{Key: "branchId", Value: 1}, {Key: "createdAt", Value: -1}}},
		// ⚠️ **`createdAt` on its own, and it is not a duplicate of the line
		// above.** A compound index only answers queries that start at its first
		// field, so `{createdAt: {$gte: …}}` with no branch cannot use
		// `(branchId, createdAt)` — it is a full collection scan.
		//
		// That is exactly the shape of the nightly billing pass, which reads
		// every order in the window for every tenant. Measured on a live tenant:
		// COLLSCAN. It costs nothing at seven orders and it is the whole order
		// history of the busiest restaurant on the box once a night at scale —
		// on the **shared** mongod, which makes it everybody else's problem
		// rather than that restaurant's.
		{s.Orders, bson.D{{Key: "createdAt", Value: -1}}},
		// The till's floor screen: "what is open in this branch, oldest first",
		// polled by every waiter's tablet and the cashier's screen for the whole
		// service. Without it that is a scan of the branch's entire order
		// history, several times a minute, on the shared mongod — the busiest
		// query in the building reading the one collection that only grows.
		//
		// ⚠️ Not partial. `check.closedAt: {$exists: false}` is what the query
		// actually filters on and Mongo's partialFilterExpression cannot express
		// it ($exists: false is not allowed). Indexing all checks and letting the
		// closed ones be skipped costs a little space and answers the query;
		// a clever half-index would answer nothing.
		{s.Orders, bson.D{{Key: "branchId", Value: 1}, {Key: "check.openedAt", Value: 1}}},
		// One open check per table, checked every time a check is opened or a
		// party is moved. A short lookup that must stay short: it runs while a
		// waiter is standing at the table.
		{s.Orders, bson.D{{Key: "branchId", Value: 1}, {Key: "tableId", Value: 1}}},
		// ---- Looking one order up, four different ways ----
		//
		// ⚠️ **Every one of these was a scan of the collection that only
		// grows**, and none of them showed as slow while a tenant had a few
		// hundred orders. That is the whole problem with this shape of bug: it
		// arrives with success, on the busiest restaurant on the shared mongod,
		// as "the site got slow" with nothing pointing at a cause.
		//
		// `number` is how a guest tracks an order, rates it, and reaches the
		// bank link — public, unauthenticated, and linked to from an SMS, so it
		// is also the one a stranger can call in a loop. Not unique: uniqueness
		// is not what is being fixed here, and an index that refuses to build
		// stops the server (the lesson pos_settings.branchId taught).
		{s.Orders, bson.D{{Key: "number", Value: 1}}},
		// clientId is deliberately absent: the till's idempotency key already
		// has a *unique sparse* index further down. Asking Mongo for the same
		// keys with different options is refused — and this loop returns on the
		// first error, so the duplicate did not cost one index, it cost **every
		// index defined after it**. Caught on a live database rather than by a
		// test, which is the only place it is visible.
		// A customer's own order history, newest first — the profile page.
		{s.Orders, bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
		// A booking is tracked by number exactly as an order is.
		{s.Reservations, bson.D{{Key: "number", Value: 1}}},
		{s.Couriers, bson.D{{Key: "branchId", Value: 1}}},
		{s.Reservations, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: 1}}},
		{s.Staff, bson.D{{Key: "branchId", Value: 1}}},
		// Every attendance read is "this person, this date range".
		{s.Shifts, bson.D{{Key: "staffId", Value: 1}, {Key: "date", Value: 1}}},
		{s.Shifts, bson.D{{Key: "branchId", Value: 1}, {Key: "date", Value: 1}}},
		{s.StaffPayments, bson.D{{Key: "staffId", Value: 1}, {Key: "at", Value: -1}}},
		// The till. Every read here is "the open shift for this branch" or
		// "closed shifts, newest first", and both are this index.
		{s.CashShifts, bson.D{{Key: "branchId", Value: 1}, {Key: "openedAt", Value: -1}}},
		{s.CashEntries, bson.D{{Key: "shiftId", Value: 1}, {Key: "at", Value: 1}}},
		// The call log is read newest-first for a branch, and by number when
		// the same person rings twice.
		{s.Calls, bson.D{{Key: "branchId", Value: 1}, {Key: "createdAt", Value: -1}}},
		{s.Calls, bson.D{{Key: "phone", Value: 1}, {Key: "createdAt", Value: -1}}},
		{s.Payments, bson.D{{Key: "orderId", Value: 1}}},
		// ---- The warehouse ----
		//
		// ⚠️ Every one of these is read as "this branch, this period", and two
		// of them are read **on every screen that shows a cost**: the flow
		// report walks the period's deliveries and write-offs, and the
		// ingredient list computes what should be on the shelf on every load.
		// Without an index that is a scan of collections which only grow, on
		// the shared mongod, several times a minute during a delivery.
		{s.Purchases, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}},
		{s.WriteOffs, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}},
		{s.Transfers, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}},
		// ⚠️ **The busiest of them by a wide margin**, because this collection
		// grows by a row per dish sold rather than a row per delivery — and it
		// is now what every balance, every stop list and every stocktake reads.
		// Unindexed it is a scan of the restaurant's entire sales history, on
		// the shared mongod, on a screen the owner leaves open.
		{s.StockMoves, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}},
		// The reconciler asks "what has been written for this order" on every
		// tap of a tile.
		{s.StockMoves, bson.D{{Key: "orderId", Value: 1}}},
		// The buyer's own runs, newest first — the screen that answers "did my
		// delivery go through" on a phone with a bad signal.
		{s.Purchases, bson.D{{Key: "createdById", Value: 1}, {Key: "at", Value: -1}}},
		// One person's petty-cash account, and the ledger behind it.
		{s.Advances, bson.D{{Key: "staffId", Value: 1}, {Key: "at", Value: -1}}},
		{s.Advances, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}},
		{s.SafeEntries, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}},
		// Read by the financial report for every period it draws.
		{s.Expenses, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}},
		// One courier's pay, and the report's total for a period.
		{s.CourierPayments, bson.D{{Key: "courierId", Value: 1}, {Key: "at", Value: -1}}},
		{s.CourierPayments, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}},
		// The payouts screen, and the per-rail balances behind it.
		{s.Payouts, bson.D{{Key: "branchId", Value: 1}, {Key: "receivedAt", Value: -1}}},
		// The last counted balance of each account, newest first.
		{s.BankBalances, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}},
		// "When was the last collection" runs on every money screen.
		{s.Collections, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}},
		// ⚠️ "What have we sold through this rail since it last paid us" runs
		// once per rail on that screen. Unindexed it is a full scan of the sales
		// history, several times over, on a page an owner leaves open.
		{s.Orders, bson.D{{Key: "paymentMethod", Value: 1}, {Key: "createdAt", Value: -1}}},
		// The shopping lists a branch has open, newest first — read by the till
		// and by every buyer's phone.
		{s.BuyOrders, bson.D{{Key: "branchId", Value: 1}, {Key: "createdAt", Value: -1}}},
		// And the movement card asks for one ingredient across a period.
		{s.StockMoves, bson.D{{Key: "lines.ingredientId", Value: 1}, {Key: "at", Value: -1}}},
		// ⚠️ The sweep reads "orders touched since", every two minutes, forever.
		{s.Orders, bson.D{{Key: "updatedAt", Value: -1}}},
		// A count is looked up as "the most recent one before this moment",
		// which is this index read backwards — and it runs before every
		// expected-stock figure, including the one behind "running out".
		{s.Stocktakes, bson.D{{Key: "branchId", Value: 1}, {Key: "at", Value: -1}}},
		// The shortfall queue asks "which of these counts has been answered",
		// for every count in a quarter, on a screen that is left open.
		// ⚠️ Its unique twin is created below, on the pair of ids: the same keys
		// with different options is the one thing Mongo refuses, and the refusal
		// used to take every index after it down with it.
		{s.ShortageCases, bson.D{{Key: "stocktakeId", Value: 1}}},
		// The card resolver asks "which dishes use this ingredient" when one
		// is deleted, and the flow report asks it for every dish sold.
		{s.Menu, bson.D{{Key: "recipe.ingredientId", Value: 1}}},
		// The print agent's poll: "the oldest job for this branch that nobody
		// has finished", every few seconds, for as long as the restaurant is
		// open.
		{s.PrintJobs, bson.D{{Key: "branchId", Value: 1}, {Key: "createdAt", Value: 1}}},
		// The push send path reads "every browser this customer registered".
		// The endpoint's own unique index is created separately, below, for the
		// same reason pos_settings.branchId is: different options, same keys.
		{s.PushSubscriptions, bson.D{{Key: "userId", Value: 1}}},
		// pos_settings.branchId is deliberately absent here: it is created
		// further down as a *unique* index. Listing it here too would ask Mongo
		// for the same keys with different options, which it refuses.
	}
	// ⚠️ **One bad spec must not take the rest of the list with it.** This loop
	// used to return on the first error, and the failure that taught us was a
	// duplicate: a plain index asked for keys that already had a *unique* one,
	// Mongo refused it, and **every index defined after that line was never
	// created** — on a database that was already running. Nothing broke that
	// day; the queries simply started scanning, which is a bug that arrives
	// months later disguised as growth.
	//
	// Logged and carried on, because a missing index is a slow restaurant and a
	// server that refuses to boot is a closed one.
	for _, spec := range specs {
		model := mongo.IndexModel{Keys: spec.keys, Options: options.Index()}
		if _, err := spec.coll.Indexes().CreateOne(ctx, model); err != nil {
			log.Printf("index setup: %s %v: %v", spec.coll.Name(), spec.keys, err)
		}
	}

	// ⚠️ One subscription per endpoint, enforced rather than assumed.
	//
	// A service worker re-registers silently after a browser update, and the
	// subscribe handler upserts on this key. Without the unique index two
	// simultaneous registrations both miss the existing document and both
	// insert — and from then on that guest receives every campaign twice. The
	// only symptom is a complaint the restaurant cannot reproduce.
	if _, err := s.PushSubscriptions.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "endpoint", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	// ⚠️ **One answer per shortfall, enforced rather than assumed.** Two
	// managers read the same queue, both see an open case and both write a
	// verdict; without this the second insert succeeds and the screen then
	// shows whichever of the two `Find` reaches first — so a shortfall's
	// explanation changes between refreshes with nobody having edited it. The
	// handler relies on this index to refuse the second one, exactly as the
	// count's own explanation relies on a filter to.
	if _, err := s.ShortageCases.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "stocktakeId", Value: 1}, {Key: "ingredientId", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	// ⚠️ **One barcode per brand, enforced rather than assumed.** A shop's
	// counter finds a product by scanning, and a duplicate makes `FindOne` return
	// whichever row it reaches first — so the same packet rings up at two
	// different prices on two different days and nobody can reproduce it. The
	// owner who typed the second one in has no way to notice, because both rows
	// look correct on their own screen.
	//
	// ⚠️ **Sparse, because almost nothing has a barcode.** Every dish ever
	// written has none; without this the index would treat them all as the same
	// empty value and refuse the second one.
	//
	// ⚠️ **Per brand, not global.** Two brands under one owner may genuinely
	// stock the same EAN — a grocery and a pharmacy both sell the same water —
	// and a catalogue belongs to a brand.
	if _, err := s.Menu.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "brandId", Value: 1}, {Key: "barcode", Value: 1}},
		Options: options.Index().SetUnique(true).SetPartialFilterExpression(
			bson.M{"barcode": bson.M{"$type": "string", "$gt": ""}},
		),
	}); err != nil {
		return err
	}

	// ⚠️ **One row per bottle, enforced rather than assumed.** A marking code
	// names a physical object: the same one arriving twice is a duplicate scan
	// or a counterfeit, and both are things somebody has to look at rather than
	// facts to store twice. Without the index a re-scan at goods-in would put
	// two rows in the store, and the till would then let the same bottle be sold
	// twice — the exact failure the state's marking exists to make impossible.
	//
	// ⚠️ **Global, not per branch.** Two branches of one chain cannot
	// legitimately hold the same bottle either, and a code that turned up in two
	// of them is worth refusing loudly.
	if _, err := s.MarkedUnits.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "code", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	// What a branch is holding, which is the question the receiving screen and
	// the till both ask.
	if _, err := s.MarkedUnits.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "branchId", Value: 1},
			{Key: "menuItemId", Value: 1},
			{Key: "soldAt", Value: 1},
		},
	}); err != nil {
		return err
	}

	// ⚠️ **One briefing per day per lens, enforced rather than assumed.** It is
	// written with an upsert, and two dashboard tabs opened in the same second
	// both miss the existing document and both insert. The second one is a
	// second call to a paid API, and from then on `FindOne` returns whichever
	// of the two it happens to reach — so an owner watches the morning's advice
	// change between refreshes with nothing having happened. The same shape as
	// the push and POS indexes, and there for the same reason.
	if _, err := s.Briefings.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "day", Value: 1}, {Key: "scope", Value: 1}, {Key: "lang", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	// ⚠️ **One row per phone, enforced rather than assumed** — the same lesson
	// the push endpoint above already taught. The app re-registers on every
	// launch, because the token is re-read from the operating system and can be
	// re-issued after a reinstall. Without this index a device accumulates a
	// row per launch and every kitchen notification is delivered as many times
	// as the app has been opened — which reads as the server being broken and
	// is the fastest way to teach a waiter to turn notifications off.
	if _, err := s.StaffDevices.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "token", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	// Sending reads by person: "whose check is this, and what phones do they
	// have".
	if _, err := s.StaffDevices.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "staffId", Value: 1}},
	}); err != nil {
		return err
	}

	// ⚠️ **The courier's phone, and the same two indexes for the same two
	// reasons.** One row per token, because the app re-registers on every
	// launch; and a lookup by courier, because that is the only question a send
	// ever asks — "this order is Aziz's, what does Aziz carry".
	if _, err := s.CourierDevices.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "token", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	if _, err := s.CourierDevices.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "courierId", Value: 1}},
	}); err != nil {
		return err
	}

	// The owner's phone: one row per token, and a lookup by branch — every
	// send here asks "who watches this branch", never "who is this person".
	if _, err := s.AdminDevices.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "token", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	if _, err := s.AdminDevices.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "branchId", Value: 1}},
	}); err != nil {
		return err
	}

	// ---- The device binding, and both halves of it are indexes ----
	//
	// ⚠️ **The uniqueness is the feature, not a safeguard around it.** One
	// account may hold one install per app, and one install may hold one
	// account per app; without these two indexes a race between two sign-ins
	// seconds apart writes both, and the rule is quietly gone on exactly the
	// evening two people are trying to use one login.
	if _, err := s.LoginDevices.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "kind", Value: 1}, {Key: "subjectId", Value: 1}, {Key: "app", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	if _, err := s.LoginDevices.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "app", Value: 1}, {Key: "deviceId", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	// Pre-orders: "what is this branch due to cook next", which is also what
	// every open panel tab asks every fifteen seconds (AdminAlerts).
	//
	// ⚠️ **Partial, not sparse.** A sparse compound index would buy nothing
	// here — Mongo only skips a document missing *every* indexed field, and
	// `branchId` is on all of them, so all of them would be indexed anyway.
	// The partial filter is the one that actually says what is meant: index the
	// orders that have a scheduled time, which is the small minority, and leave
	// the ordinary ones — the overwhelming majority, and never an answer to
	// this question — out of it entirely.
	if _, err := s.Orders.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "branchId", Value: 1}, {Key: "scheduledAt", Value: 1}},
		Options: options.Index().SetPartialFilterExpression(
			bson.M{"scheduledAt": bson.M{"$exists": true}}),
	}); err != nil {
		return err
	}

	// ⚠️ One store per ingredient **per branch** — the rule warehouse.go states,
	// now enforceable because the placement is its own row. Without the index a
	// concurrent save writes two placements for one shelf and `FindOne` picks
	// one of them, which reads as "the store I chose changed back by itself".
	if _, err := s.Placements.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "branchId", Value: 1}, {Key: "ingredientId", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	// One row per visitor per day. The upsert relies on it: without the unique
	// key a returning visitor becomes a second row and the "unique visitors"
	// figure quietly turns into a page-view count.
	if _, err := s.Visits.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "date", Value: 1}, {Key: "vid", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	// And they expire. Visit rows are the one collection here that grows with
	// traffic rather than with the business, so they are given an end: the
	// daily totals worth keeping are rolled up by the platform long before.
	if _, err := s.Visits.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "firstAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(100 * 24 * 60 * 60),
	}); err != nil {
		return err
	}

	// One transaction per provider id. This is the guard that makes a retried
	// callback harmless: all three providers retry, and two rows for one
	// payment would mean an order paid twice in the ledger.
	if _, err := s.Payments.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "provider", Value: 1}, {Key: "providerTxnId", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	// One row per call the exchange announces. Five events arrive for a single
	// call and all five must land on the same row, so the exchange's own id is
	// the key. Sparse: calls typed in by hand have no such id, and they are
	// the majority on an install with no PBX.
	if _, err := s.Calls.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "pbxCallId", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}); err != nil {
		return err
	}

	// One sale per id the till minted.
	//
	// ⚠️ **This index is the offline guarantee**, not the code that reads it. A
	// till with no network keeps its checks on its own disk and sends them when
	// the connection returns; the send is retried by a program that cannot know
	// whether the first attempt landed, and two attempts a second apart would
	// otherwise be two dinners — charged twice and counted twice in the day's
	// takings. Sparse: every sale rung up online has no such id, and they are
	// almost all of them.
	if _, err := s.Orders.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "clientId", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}); err != nil {
		return err
	}

	// ⚠️ **One delivery per id the buyer's phone minted, and the same guarantee
	// for the same reason.** A market has worse signal than a dining room: the
	// app holds the run and retries, and two attempts a second apart would be a
	// shelf raised twice, an invoice paid twice, and the same price written into
	// the history twice. Sparse, because every delivery typed into the panel has
	// no such id and they are almost all of them.
	if _, err := s.Purchases.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "clientId", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}); err != nil {
		return err
	}

	// ⚠️ **One safe movement per thing that caused it.** The automatic entries
	// are written by handlers that can be retried, and a duplicate row here is a
	// wrong balance that looks exactly like a right one — plausible, and
	// invisible to everything downstream. Sparse, because a movement somebody
	// typed in by hand has no reference and they are most of them.
	if _, err := s.SafeEntries.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "refKind", Value: 1}, {Key: "refId", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}); err != nil {
		return err
	}

	// The kitchen pass, and the order board on the dining room's wall.
	//
	// ⚠️ **The board is why this is here.** The pass is read by one tablet when
	// a cook looks at it; the board is polled by every television in the branch
	// every few seconds, all evening, against a collection that only grows.
	// Without an index that is a full scan of every order the restaurant has
	// ever taken, several times a minute, for a screen nobody is even touching.
	if _, err := s.Orders.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "branchId", Value: 1},
			{Key: "status", Value: 1},
			{Key: "queuedAt", Value: 1},
		},
	}); err != nil {
		return err
	}
	// The ready half of the board: a short window, newest first. Partial, so it
	// holds only orders the kitchen has finished — on a year-old collection
	// that is a fraction of the rows, and the ones being asked for.
	if _, err := s.Orders.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "branchId", Value: 1}, {Key: "readyAt", Value: -1}},
		Options: options.Index().SetPartialFilterExpression(
			bson.M{"readyAt": bson.M{"$exists": true}}),
	}); err != nil {
		return err
	}

	// One mapping per dish per branch: the same lag'mon cannot point at two
	// different products in one till.
	if _, err := s.POSMappings.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "branchId", Value: 1}, {Key: "menuItemId", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	// One till connection per branch. The settings page saves with an upsert
	// keyed on `branchId`, so two requests arriving together could each insert
	// a document — and then `FindOne` returns whichever the server happens to
	// hand back. The symptom is a saved setting that "comes back on its own",
	// which reads as the form losing data rather than as two rows existing.
	if err := dropDuplicatePOSSettings(ctx, s); err != nil {
		return err
	}
	posBranchIdx := mongo.IndexModel{
		Keys:    bson.D{{Key: "branchId", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	if _, err := s.POSSettings.Indexes().CreateOne(ctx, posBranchIdx); err != nil {
		// Installs that booted an earlier build already have this key as a
		// plain index, and Mongo refuses to redefine one index with different
		// options. Replacing it is safe — the keys are identical, so nothing
		// reads worse in between, and the duplicates are already gone above.
		if _, dropErr := s.POSSettings.Indexes().DropOne(ctx, "branchId_1"); dropErr != nil {
			return err
		}
		if _, err := s.POSSettings.Indexes().CreateOne(ctx, posBranchIdx); err != nil {
			return err
		}
	}

	// One virtual cash register per branch, for exactly the reason above: the
	// settings page upserts on `branchId`, and two rows would make FindOne
	// return either of them.
	//
	// Unlike pos_settings this needs no duplicate sweep and no drop-and-retry:
	// the collection is new, so no install can already hold a plain index or a
	// second document. If that ever stops being true, copy the block above —
	// the failure it prevents is the same one, and here it would mean a branch
	// filing its receipts under whichever taxpayer Mongo handed back.
	if _, err := s.FiscalSettings.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "branchId", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	// ⚠️ **One account per phone number**, guaranteed by the database rather than
	// only by the four handlers that look before they insert.
	//
	// Every write path already checks: the SMS login finds-or-creates, the
	// operator's order screen finds-or-creates, changing a number refuses one held
	// by somebody else (409), and a Telegram number does the same. What none of
	// them can prevent is the race — two requests both finding nothing and both
	// inserting, which is the `pos_settings` trap with a customer's order history
	// on the other side of it.
	//
	// Partial, because the number is genuinely optional: a Telegram guest signs in
	// with no phone at all, and a unique index over an absent field would let
	// exactly one such account exist.
	if err := ensureUserPhoneUnique(ctx, s); err != nil {
		return err
	}

	// Preview tokens find themselves by token and expire on their own. The TTL is
	// the point: a preview link that outlives the session that made it is the
	// unpublished draft left publicly reachable.
	if _, err := s.DesignPreviews.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "token", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	if _, err := s.DesignPreviews.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expiresAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}); err != nil {
		return err
	}

	// ⚠️ **Every message the bot receives resolves its sender this way**,
	// including the ones that are only a button press — and it was a scan of the
	// customer collection each time. Partial, because most guests have no
	// Telegram at all and indexing a field that is absent from almost every
	// document is mostly wasted pages. Not unique: that guarantee belongs to the
	// phone index above, and a unique index that refuses to build stops the
	// server.
	if _, err := s.Users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "telegramId", Value: 1}},
		Options: options.Index().SetPartialFilterExpression(
			bson.M{"telegramId": bson.M{"$exists": true}}),
	}); err != nil {
		return err
	}

	// One row per chat: the bot's memory of a conversation that has no account yet.
	if _, err := s.TelegramChats.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "chatId", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	// Feedback with no branch is filtered out of every panel view — see
	// backfillFeedbackBranch for how the bot's first rows were lost that way.
	if err := backfillFeedbackBranch(ctx, s); err != nil {
		return err
	}

	// One pairing per television, so asking for a new code replaces the old
	// one rather than leaving a second working code behind — see
	// models.TVPairing.
	if _, err := s.TVPairings.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "installId", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	// Typed into the panel, so it is looked up by code.
	if _, err := s.TVPairings.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "code", Value: 1}},
	}); err != nil {
		return err
	}
	// ⚠️ Expired pairings remove themselves. Without this the collection only
	// grows, and a row left behind by a television somebody unplugged mid-setup
	// keeps its install id — so that set could never ask for a code again.
	if _, err := s.TVPairings.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expiresAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}); err != nil {
		return err
	}
	// The panel's list, and the count the per-screen price is billed from.
	if _, err := s.TVScreens.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "branchId", Value: 1}, {Key: "version", Value: 1}},
	}); err != nil {
		return err
	}
	// One row per television: a set re-paired must replace its row rather than
	// quietly spend a second paid slot.
	if _, err := s.TVScreens.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "installId", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}); err != nil {
		return err
	}

	// The playlist, read in loop order by every television in the branch and
	// rewritten whole whenever somebody drags a row.
	if _, err := s.TVSlides.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "branchId", Value: 1}, {Key: "order", Value: 1}},
	}); err != nil {
		return err
	}

	// One pending code per phone **per purpose**: a customer login code and an
	// admin password reset must not overwrite each other (see models.PhoneCode).
	if _, err := s.PhoneCodes.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "phone", Value: 1}, {Key: "purpose", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	// Expired codes remove themselves. Without this the collection only ever
	// grows, and a code left behind by an interrupted flow lingers for good.
	if _, err := s.PhoneCodes.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expiresAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}); err != nil {
		return err
	}
	return nil
}

// dropDuplicatePOSSettings collapses a branch's till settings to one document,
// keeping the one saved last.
//
// It runs before the unique index because creating that index on a collection
// that already holds duplicates fails, and a failed migration stops the server
// — the cure would be worse than the disease it prevents. Keeping the newest is
// the only defensible choice: it is the one the owner last typed, and the one
// the settings page has been showing them (a `FindOne` with no sort returns
// natural order, which in practice is the oldest).
func dropDuplicatePOSSettings(ctx context.Context, s *Store) error {
	cur, err := s.POSSettings.Find(ctx, bson.M{},
		options.Find().
			SetSort(bson.D{{Key: "branchId", Value: 1}, {Key: "updatedAt", Value: -1}, {Key: "_id", Value: -1}}).
			SetProjection(bson.M{"_id": 1, "branchId": 1}))
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	var rows []struct {
		ID       primitive.ObjectID `bson:"_id"`
		BranchID primitive.ObjectID `bson:"branchId"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return err
	}

	seen := map[primitive.ObjectID]bool{}
	stale := []primitive.ObjectID{}
	for _, row := range rows {
		if seen[row.BranchID] {
			stale = append(stale, row.ID)
			continue
		}
		seen[row.BranchID] = true
	}
	if len(stale) == 0 {
		return nil
	}
	if _, err := s.POSSettings.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": stale}}); err != nil {
		return err
	}
	log.Printf("migrate: dropped %d duplicate pos_settings document(s)", len(stale))
	return nil
}

// ensureUserPhoneUnique adds the one-account-per-number guarantee — unless the
// data already contradicts it.
//
// ⚠️ **Duplicates are never merged and never deleted here, and boot is never
// blocked.** Both of the obvious shortcuts are wrong:
//
//   - Deleting the "extra" account destroys somebody's order history, loyalty
//     balance and saved addresses, for a row a human has never looked at. This is
//     the opposite of the POS settings case, where a duplicate held a typed
//     setting and the newest was defensibly right.
//   - Letting the index creation fail stops the server. A restaurant whose site
//     will not start because two guests once shared a number is a far worse
//     outcome than the race the index prevents, and it would happen at the worst
//     possible moment — during an upgrade nobody was watching.
//
// So the index is created when it can be, skipped loudly when it cannot, and the
// offending numbers are named so somebody can merge the accounts by hand. The
// handlers keep enforcing the rule either way.
func ensureUserPhoneUnique(ctx context.Context, s *Store) error {
	dupes, err := duplicatePhones(ctx, s)
	if err != nil {
		return err
	}
	if len(dupes) > 0 {
		log.Printf("users: %d phone number(s) held by more than one account, "+
			"skipping the unique index — merge them by hand: %s",
			len(dupes), strings.Join(dupes, ", "))
		return nil
	}
	idx := mongo.IndexModel{
		Keys: bson.D{{Key: "phone", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("phone_unique").
			// Only rows that actually have a number. A Telegram guest with no
			// phone yet is a normal account, and there can be many of them.
			SetPartialFilterExpression(bson.M{"phone": bson.M{"$type": "string"}}),
	}
	if _, err := s.Users.Indexes().CreateOne(ctx, idx); err != nil {
		// An earlier build may have created a plain index on the same key, and
		// Mongo refuses to redefine one with different options — the same
		// situation as pos_settings.branchId.
		if _, dropErr := s.Users.Indexes().DropOne(ctx, "phone_1"); dropErr != nil {
			log.Printf("users: could not create the unique phone index: %v", err)
			return nil
		}
		if _, err := s.Users.Indexes().CreateOne(ctx, idx); err != nil {
			log.Printf("users: could not create the unique phone index: %v", err)
		}
	}
	return nil
}

// duplicatePhones lists the numbers held by more than one account.
func duplicatePhones(ctx context.Context, s *Store) ([]string, error) {
	cur, err := s.Users.Aggregate(ctx, []bson.M{
		{"$match": bson.M{"phone": bson.M{"$type": "string", "$ne": ""}}},
		{"$group": bson.M{"_id": "$phone", "n": bson.M{"$sum": 1}}},
		{"$match": bson.M{"n": bson.M{"$gt": 1}}},
		// Enough to act on; a list of hundreds in a boot log is not read.
		{"$limit": 20},
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rows []struct {
		Phone string `bson:"_id"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Phone)
	}
	return out, nil
}

// backfillFeedbackBranch gives every feedback row a branch, so it can be seen.
//
// ⚠️ **A row with no branch is invisible, not wrong.** The panel's lists are scoped by
// branch (`orderScope`), so a feedback without one is filtered out of every view — it
// exists, it is correct, and nobody will ever read it. That is how the bot's first
// feedback rows were lost: written successfully, and nowhere anybody looks.
//
// A migration rather than a script run on the server, for the reason migrations exist:
// a script fixes one machine, and the next install would grow the same invisible rows.
// It touches only rows missing the field, so on every boot after the first it is one
// indexed count that finds nothing.
//
// Where the branch comes from, in order of how much it is actually known:
//
//  1. the order the feedback is about — that is where they ate;
//  2. failing that, the guest's most recent order;
//  3. failing that, the first active branch, because a single-branch restaurant is
//     every restaurant until it is not.
func backfillFeedbackBranch(ctx context.Context, s *Store) error {
	missing := bson.M{"$or": []bson.M{
		{"branchId": bson.M{"$exists": false}},
		{"branchId": nil},
		{"branchId": primitive.NilObjectID},
	}}
	n, err := s.Feedback.CountDocuments(ctx, missing)
	if err != nil || n == 0 {
		return err
	}

	// The fallback, read once: it is the same answer for every row that needs it.
	var fallback primitive.ObjectID
	var branch struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	if err := s.Branches.FindOne(ctx, bson.M{"isActive": true},
		options.FindOne().SetSort(bson.D{
			{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: 1},
		})).Decode(&branch); err == nil {
		fallback = branch.ID
	}

	cur, err := s.Feedback.Find(ctx, missing)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	var rows []struct {
		ID      primitive.ObjectID `bson:"_id"`
		OrderID primitive.ObjectID `bson:"orderId"`
		UserID  primitive.ObjectID `bson:"userId"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return err
	}

	fixed := 0
	for _, row := range rows {
		branchID := fallback
		var order struct {
			BranchID primitive.ObjectID `bson:"branchId"`
		}
		switch {
		case !row.OrderID.IsZero():
			if err := s.Orders.FindOne(ctx, bson.M{"_id": row.OrderID}).Decode(&order); err == nil &&
				!order.BranchID.IsZero() {
				branchID = order.BranchID
			}
		case !row.UserID.IsZero():
			if err := s.Orders.FindOne(ctx, bson.M{"userId": row.UserID},
				options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
			).Decode(&order); err == nil && !order.BranchID.IsZero() {
				branchID = order.BranchID
			}
		}
		if branchID.IsZero() {
			// No branches at all: a brand-new install with nothing seeded yet. Left
			// alone rather than given a zero id, which is the value that made it
			// invisible in the first place.
			continue
		}
		if _, err := s.Feedback.UpdateByID(ctx, row.ID,
			bson.M{"$set": bson.M{"branchId": branchID}}); err == nil {
			fixed++
		}
	}
	if fixed > 0 {
		log.Printf("feedback: %d row(s) had no branch and were invisible in the panel — "+
			"attached to a branch", fixed)
	}
	return nil
}

// EnsureKitchenAccess grandfathers staff who could already open the KDS.
//
// ⚠️ **A security tightening cannot use the usual "zero value is today's
// behaviour" rule** — if the default meant "allowed", the permission would not
// be one. But defaulting to false without this would, on the deploy that adds
// it, blank the kitchen screen of every restaurant already using one, in the
// middle of service, with nothing on the tablet explaining why.
//
// So documents that predate the field get true and new staff get false. Only
// staff that have never seen the field are touched, so an owner who has already
// revoked somebody keeps them revoked.
func EnsureKitchenAccess(ctx context.Context, s *Store) error {
	_, err := s.Staff.UpdateMany(ctx,
		bson.M{"canKitchen": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"canKitchen": true}},
	)
	return err
}

// EnsureIngredientPlacements moves each ingredient's store onto a per-branch row.
//
// ⚠️ **Without it, splitting the field empties every stock screen at once.**
// `ingredient.warehouseId` is the only record of where anything is kept, and
// the readers now ask the placement collection — so on the deploy that ships
// this, a restaurant with named stores would find every shelf reading zero and
// every count measured from nothing. That is not a rough edge: the balance is
// what the kitchen orders against.
//
// ⚠️ **The branch comes from the warehouse**, which is the only honest source:
// a store already belongs to exactly one branch, so the ingredient was, in
// effect, already placed in that branch and nowhere else. Ingredients with no
// store need no row — the undivided store is the zero value, per branch.
//
// ⚠️ Idempotent: rows are inserted only where none exists, so a restart changes
// nothing and an owner who has since moved something keeps their choice.
func EnsureIngredientPlacements(ctx context.Context, s *Store) error {
	cur, err := s.Ingredients.Find(ctx, bson.M{
		"warehouseId": bson.M{"$exists": true, "$ne": primitive.NilObjectID},
	})
	if err != nil {
		return err
	}
	var ingredients []struct {
		ID          primitive.ObjectID `bson:"_id"`
		WarehouseID primitive.ObjectID `bson:"warehouseId"`
	}
	if err := cur.All(ctx, &ingredients); err != nil {
		return err
	}
	if len(ingredients) == 0 {
		return nil
	}

	branchOf := map[primitive.ObjectID]primitive.ObjectID{}
	wcur, err := s.Warehouses.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	var stores []struct {
		ID       primitive.ObjectID `bson:"_id"`
		BranchID primitive.ObjectID `bson:"branchId"`
	}
	if err := wcur.All(ctx, &stores); err != nil {
		return err
	}
	for _, w := range stores {
		branchOf[w.ID] = w.BranchID
	}

	now := time.Now()
	for _, in := range ingredients {
		branch, ok := branchOf[in.WarehouseID]
		// A store that no longer exists cannot name a branch, and guessing one
		// would file the ingredient into somebody else's building.
		if !ok || branch.IsZero() {
			continue
		}
		_, err := s.Placements.UpdateOne(ctx,
			bson.M{"branchId": branch, "ingredientId": in.ID},
			bson.M{"$setOnInsert": bson.M{
				"branchId":     branch,
				"ingredientId": in.ID,
				"warehouseId":  in.WarehouseID,
				"updatedAt":    now,
			}},
			options.Update().SetUpsert(true),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// EnsureDeliveriesSettled marks every delivery entered before invoices could be
// unpaid as paid.
//
// ⚠️ **Without it, switching this on invents a debt.** `paid` is absent on
// every existing purchase, and the report reads absent as owing — so a
// restaurant with two years of deliveries would open the supplier page to a
// total in the hundreds of millions, owed to people it settled with long ago.
// A figure that large and that wrong is not a rough edge: it is the reason
// somebody stops opening the page.
//
// ⚠️ Matched on the field being **missing**, not on `paid: false`, so it runs
// exactly once and never re-settles an invoice an owner has deliberately marked
// unpaid since. Idempotent on a restart, like every migration beside it.
func EnsureDeliveriesSettled(ctx context.Context, s *Store) error {
	now := time.Now()
	_, err := s.Purchases.UpdateMany(ctx,
		bson.M{"paid": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"paid": true, "paidAt": now}},
	)
	return err
}

// EnsureStaffRoles seeds the role list and moves existing staff onto it.
//
// ⚠️ **Nobody may lose a right they were using.** Before roles, `canCashier`
// meant "may take payment, write off cooked food and give discounts" — all
// three, because nothing distinguished them. The seeded Kassir role ships
// without void and discount on purpose, which is the right default for a new
// restaurant and the wrong thing to impose on a live one: the deploy that
// added roles would otherwise take two abilities away from every cashier in
// the country, mid-service, with nothing on the screen explaining it.
//
// So existing accounts are migrated to a role that carries **what they can do
// today**, and only new restaurants get the tighter default. The same shape as
// EnsureKitchenAccess, one step larger.
//
// ⚠️ Idempotent in both halves: roles are seeded only into an empty collection,
// and staff are matched only while they have no role. A second run — a restart,
// a redeploy — changes nothing, and an owner who has already retuned a role or
// reassigned somebody keeps their work.
func EnsureStaffRoles(ctx context.Context, s *Store) error {
	if err := seedStaffRoles(ctx, s); err != nil {
		return err
	}
	if err := translateSeededRoles(ctx, s); err != nil {
		return err
	}
	if err := grantTechnologistStock(ctx, s); err != nil {
		return err
	}
	return assignStaffRoles(ctx, s)
}

// EnsureBuyerRole adds the shipped "Zakupshik" role to a restaurant that was
// already running when buying from a phone did not exist.
//
// ⚠️ **`seedStaffRoles` cannot do this and must not learn to.** It refuses to
// upsert by name on purpose: a restaurant that renamed "Ofitsiant" would get a
// second one back on every restart, and one that deleted a role it does not use
// would find it resurrected. So a new shipped role reaches existing installs
// through its own one-off pass.
//
// ⚠️ **Marked once, and the marker records the visit rather than the outcome.**
// Matching on "no Zakupshik role exists" would recreate it every boot for the
// restaurant that has just deliberately deleted it — a role that grows back
// overnight is worse than one that never arrived. Same shape as
// EnsureReviewsBand and grantTechnologistStock, for the same reason.
//
// ⚠️ It grants nothing to anybody: the role is created empty of people, and an
// owner assigns it. Adding a permission to accounts that already exist is the
// one thing this file does only under the narrowest possible match.
func EnsureBuyerRole(ctx context.Context, s *Store) error {
	return addShippedRole(ctx, s, "buyer_role_v1", "Zakupshik")
}

// EnsureStorekeeperRole does the same for the role that writes the shopping
// list — and gives the roles that already run the floor the permission to write
// one.
func EnsureStorekeeperRole(ctx context.Context, s *Store) error {
	if err := addShippedRole(ctx, s, "storekeeper_role_v1", "Omborchi"); err != nil {
		return err
	}
	// ⚠️ **This widens permissions on roles that already exist**, which almost
	// nothing in this file does — so it is as narrow as it can be. A manager, a
	// general manager and a cashier are the people who hear the kitchen say
	// something has run out; shipping the feature without them would mean every
	// restaurant discovers at the counter that the one account standing there
	// cannot write a list.
	//
	// ⚠️ Matched on `name` **and** `seeded`, so a role the restaurant renamed or
	// created itself is never touched. `$addToSet`, so one that already has it
	// keeps exactly what it has. And the marker records the **visit**: matching
	// on "manager without buyorder" would find the role a restaurant had just
	// deliberately unticked, on every boot — a permission that grows back
	// overnight is worse than one that never arrived.
	_, err := s.StaffRoles.UpdateMany(ctx,
		bson.M{
			"name":            bson.M{"$in": []string{"Ish boshqaruvchi", "Menejer", "Kassir"}},
			"seeded":          true,
			"buyOrderGranted": bson.M{"$ne": true},
		},
		bson.M{
			"$addToSet": bson.M{"perms": models.PermBuyOrder},
			"$set":      bson.M{"buyOrderGranted": true, "updatedAt": time.Now()},
		},
	)
	return err
}

// addShippedRole brings one of the roles we ship to an install that predates it.
//
// ⚠️ **`seedStaffRoles` cannot do this and must not learn to.** It refuses to
// upsert by name on purpose: a restaurant that renamed "Ofitsiant" would get a
// second one back on every restart, and one that deleted a role it does not use
// would find it resurrected. So a new shipped role reaches existing installs
// through its own one-off pass, marked once.
//
// ⚠️ The marker records the visit rather than the outcome — recreating a role
// somebody has just deliberately deleted is the failure this shape prevents.
func addShippedRole(ctx context.Context, s *Store, key, name string) error {
	err := s.MigrationState.FindOne(ctx, bson.M{"_id": key}).Err()
	if err == nil {
		return nil
	}
	if err != mongo.ErrNoDocuments {
		return err
	}
	// A brand-new install has no roles yet; `seedStaffRoles` will bring this one
	// in with the rest, so there is nothing to add and the visit still counts.
	n, err := s.StaffRoles.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if n > 0 {
		var row *models.SeedRoleRow
		for _, r := range models.SeedRoleRows() {
			if r.Name == name {
				found := r
				row = &found
				break
			}
		}
		// ⚠️ Read out of the shipped list rather than written again here. Two
		// spellings of one role is how a restaurant ends up with both.
		if row == nil {
			return nil
		}
		existing, err := s.StaffRoles.CountDocuments(ctx, bson.M{"name": row.Name})
		if err != nil {
			return err
		}
		if existing == 0 {
			now := time.Now()
			if _, err := s.StaffRoles.InsertOne(ctx, models.StaffRole{
				Name: row.Name, NameRu: row.NameRu, NameEn: row.NameEn,
				Perms: row.Perms, Seeded: true,
				// Last in the picker: a role nobody has been given yet should
				// not sit above the ones the room is staffed with.
				Sort:      99,
				CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				return err
			}
		}
	}
	_, err = s.MigrationState.InsertOne(ctx, bson.M{"_id": key, "at": time.Now()})
	return err
}

// grantTechnologistStock gives the shipped Texnolog role the store.
//
// ⚠️ **This widens a permission on an existing install, which nothing else in
// this file does** — so it is narrow on purpose and worth reading twice.
// A technologist writes the tech cards and runs the counts; that permission is
// also the panel's only door for them (handlers/stocklogin.go). We shipped the
// role with nothing, so every restaurant that hired one found out at the first
// count: the account refuses the login form with "login yoki parol noto'g'ri",
// which reads as a broken password rather than as a switch nobody turned on.
//
// ⚠️ Matched on `name` **and** `seeded`, so a role the restaurant renamed or
// created itself is never touched — and `$addToSet`, so one that already has
// the box ticked keeps exactly what it has.
//
// ⚠️ **Visited once, and the marker is what makes unticking stick.** Matching
// on "Texnolog without stock" would find the role a restaurant had just
// deliberately unticked, every boot: a permission that grows back overnight is
// worse than one that was never granted. Same shape as EnsureReviewsBand, for
// the same reason — the flag records the visit, not the outcome.
//
// ⚠️ It does **not** grant anything beyond the store: `stock` is one
// permission, and it is the only one added here.
func grantTechnologistStock(ctx context.Context, s *Store) error {
	res, err := s.StaffRoles.UpdateMany(ctx,
		bson.M{
			"name":         "Texnolog",
			"seeded":       true,
			"stockGranted": bson.M{"$ne": true},
		},
		bson.M{
			"$addToSet": bson.M{"perms": models.PermStock},
			"$set":      bson.M{"stockGranted": true, "updatedAt": time.Now()},
		},
	)
	if err == nil && res.ModifiedCount > 0 {
		log.Printf("migrate: Texnolog roli ombor ruxsatini oldi (%d)", res.ModifiedCount)
	}
	return err
}

// translateSeededRoles fills in the RU/EN names of the roles we shipped.
//
// ⚠️ **Every install that predates this has eleven Uzbek words in the role
// picker**, and no owner is going to retype our own list to see them in the
// language their panel is already in. The seed only runs into an empty
// collection, so the names have to be backfilled where the roles already are.
//
// ⚠️ Matched by base name **and** only where the translation is missing: a
// restaurant that renamed a role, or typed its own Russian name, keeps what it
// wrote. That also makes the pass idempotent — the second run matches nothing.
func translateSeededRoles(ctx context.Context, s *Store) error {
	for _, r := range models.SeedRoleRows() {
		set := bson.M{}
		if r.NameRu != "" {
			set["nameRu"] = r.NameRu
		}
		if r.NameEn != "" {
			set["nameEn"] = r.NameEn
		}
		if len(set) == 0 {
			continue
		}
		for field, value := range set {
			if _, err := s.StaffRoles.UpdateMany(ctx, bson.M{
				"name":   r.Name,
				"seeded": true,
				// Missing or empty — documents written before the field
				// existed have neither.
				"$or": []bson.M{
					{field: bson.M{"$exists": false}},
					{field: ""},
				},
			}, bson.M{"$set": bson.M{field: value}}); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedStaffRoles(ctx context.Context, s *Store) error {
	n, err := s.StaffRoles.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if n > 0 {
		// ⚠️ Not upserted by name. A restaurant that renamed "Ofitsiant" would
		// otherwise get a second one back on every restart, and a restaurant
		// that deleted a role it does not use would get it resurrected.
		return nil
	}
	docs := make([]any, 0, len(models.SeedRoles()))
	for _, r := range models.SeedRoles() {
		docs = append(docs, r)
	}
	_, err = s.StaffRoles.InsertMany(ctx, docs)
	return err
}

// assignStaffRoles gives a role to everybody who has none.
//
// The mapping reads the old flags and picks the seeded role whose permissions
// are the closest **superset** of what that person could already do:
//
//	canCashier → Zal administratori (payment + void + discount + shift)
//	canWaiter  → Ofitsiant
//	canKitchen → Oshpaz
//
// ⚠️ A cashier becomes "Zal administratori" rather than "Kassir", and that
// looks wrong until you read the permissions: today's cashier *is* what this
// codebase now calls a floor administrator. Naming them Kassir would be tidier
// and would silently remove two abilities.
func assignStaffRoles(ctx context.Context, s *Store) error {
	byName := map[string]primitive.ObjectID{}
	cur, err := s.StaffRoles.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	var roles []models.StaffRole
	if err := cur.All(ctx, &roles); err != nil {
		return err
	}
	for _, r := range roles {
		byName[r.Name] = r.ID
	}

	// Most specific first: a cashier is also a waiter, and matching the other
	// way round would demote every cashier in the building.
	steps := []struct {
		filter bson.M
		role   string
	}{
		{bson.M{"canCashier": true}, "Zal administratori"},
		{bson.M{"canWaiter": true}, "Ofitsiant"},
		{bson.M{"canKitchen": true}, "Oshpaz"},
		// Everybody else — a cleaner, a technologist — gets a role with no till
		// permissions at all, so that "has an account" and "may run the till"
		// stay different questions.
		{bson.M{}, "Yordamchi xodim"},
	}
	for _, step := range steps {
		id, ok := byName[step.role]
		if !ok {
			continue
		}
		filter := bson.M{"roleId": bson.M{"$exists": false}}
		for k, v := range step.filter {
			filter[k] = v
		}
		if _, err := s.Staff.UpdateMany(ctx, filter,
			bson.M{"$set": bson.M{"roleId": id}}); err != nil {
			return err
		}
	}
	return nil
}

// withDefaults keeps what an existing install already decided and lets the
// template answer only what it never did.
//
// ⚠️ **A migration must not re-answer a question somebody has answered.** These
// twelve installs have been running for months; a shop template that arrived and
// switched their dine-in off would be a silent change to a live restaurant. So
// the template applies to a genuinely new brand — one whose profile is empty —
// and an install being migrated keeps its own answers.
func withDefaults(biz models.BusinessType, existing models.BrandFeatures) models.BrandFeatures {
	if biz == models.BizRestaurant {
		return existing
	}
	return biz.Defaults()
}
