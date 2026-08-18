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
func EnsureBrandAndBranch(ctx context.Context, s *Store) error {
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
		Features: models.BrandFeatures{
			Delivery: true,
			Pickup:   true,
			DineIn:   true,
			Booking:  rest.Booking.Enabled,
		},
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
		// The push send path reads "every browser this customer registered".
		// The endpoint's own unique index is created separately, below, for the
		// same reason pos_settings.branchId is: different options, same keys.
		{s.PushSubscriptions, bson.D{{Key: "userId", Value: 1}}},
		// pos_settings.branchId is deliberately absent here: it is created
		// further down as a *unique* index. Listing it here too would ask Mongo
		// for the same keys with different options, which it refuses.
	}
	for _, spec := range specs {
		model := mongo.IndexModel{Keys: spec.keys, Options: options.Index()}
		if _, err := spec.coll.Indexes().CreateOne(ctx, model); err != nil {
			return err
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
	return assignStaffRoles(ctx, s)
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
