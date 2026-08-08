package repository

import (
	"context"
	"log"
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
