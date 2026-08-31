package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Store is the control plane's own collections, plus the handle used to reach
// into a tenant's database when one tenant's live numbers are wanted.
type Store struct {
	DB      *mongo.Database
	Tenants *mongo.Collection
	// What each tenant has spent on briefings, for the daily cap and for
	// knowing what the feature costs before an invoice says so.
	BriefingLog *mongo.Collection
	Days        *mongo.Collection
	Users       *mongo.Collection
	// Console staff actions, and the visits agents plan. Both owner-facing.
	ConsoleLogs *mongo.Collection
	Visits      *mongo.Collection
	// The single record of the last (or running) tenant image rollout.
	Rollouts *mongo.Collection
	// Page-layout templates: one drawing reused across customers, which is the
	// commercial point of the constructor. Ours rather than any tenant's, so they
	// live in the control database.
	DesignTemplates *mongo.Collection
	// What each customer was billed, and what was actually collected.
	Invoices *mongo.Collection
	// What the nightly aggregate did last time it ran — the difference
	// between "nobody has ordered yet" and "this has never worked".
	Collector *mongo.Collection
	// One document per hour of measured uptime, behind keel.uz/status.
	Status *mongo.Collection
	// Support: the restaurants' questions and everything said about them.
	// ⚠️ Here rather than in each tenant's database — see models/support.go: an
	// operator answers thirty restaurants in a morning, and the one that is
	// down is the one writing to us.
	SupportThreads  *mongo.Collection
	SupportMessages *mongo.Collection
	// Crash reports from every app, grouped by fault. See models/report.go.
	Reports *mongo.Collection
	// Who sends us customers from outside, and on what terms. See
	// models/referral.go — not the same thing as the landing's partner logos.
	Referrers *mongo.Collection

	// The client tenant databases hang off. Separate from DB so the day tenant
	// data moves to another server, only this changes.
	tenantClient *mongo.Client
}

func New(db *mongo.Database, tenantClient *mongo.Client) *Store {
	return &Store{
		DB:              db,
		Tenants:         db.Collection("tenant"),
		BriefingLog:     db.Collection("briefing_log"),
		Days:            db.Collection("tenant_day"),
		Users:           db.Collection("user"),
		ConsoleLogs:     db.Collection("console_log"),
		Visits:          db.Collection("visit"),
		Rollouts:        db.Collection("rollout"),
		DesignTemplates: db.Collection("design_template"),
		Invoices:        db.Collection("invoice"),
		Collector:       db.Collection("collector_run"),
		Status:          db.Collection("status_hour"),
		SupportThreads:  db.Collection("support_thread"),
		SupportMessages: db.Collection("support_message"),
		Reports:         db.Collection("error_group"),
		Referrers:       db.Collection("referrer"),
		tenantClient:    tenantClient,
	}
}

// TenantDB opens one customer's database by name.
func (s *Store) TenantDB(name string) *mongo.Database {
	return s.tenantClient.Database(name)
}

// EnsureIndexes makes the two lookups that happen on every request cheap, and
// the two that must never duplicate impossible.
func (s *Store) EnsureIndexes(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// A slug decides a database name and a container name; two tenants sharing
	// one would mean two businesses sharing a database.
	if _, err := s.Tenants.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "slug", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	// Every request that arrives is resolved by Host, so this one is read more
	// often than anything else here.
	if _, err := s.Tenants.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "domains", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}); err != nil {
		return err
	}
	// A referrer's code is the address on their leaflet. Two rows sharing one
	// means the link resolves to whichever the driver returned first, and the
	// commission goes to whichever that was — a coin toss nobody can audit.
	if _, err := s.Referrers.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "code", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	// One row per tenant per day: the aggregator re-runs (a crash, a manual
	// re-collection) and must overwrite rather than double the month's bill.
	if _, err := s.Days.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "date", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	// One invoice per tenant per period. Pressing "issue" twice — a double
	// click, a retried request, two operators on the same customer — must not
	// create a second debt for a month already billed.
	if _, err := s.Invoices.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "tenantId", Value: 1}, {Key: "from", Value: 1}, {Key: "to", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	if _, err := s.Invoices.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "number", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	// The support queue, in the two orders it is ever read: an operator opens
	// the list sorted by who has been waiting longest, and a restaurant opens
	// its own history.
	if _, err := s.SupportThreads.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "status", Value: 1}, {Key: "lastAt", Value: -1}},
	}); err != nil {
		return err
	}
	if _, err := s.SupportThreads.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "slug", Value: 1}, {Key: "lastAt", Value: -1}},
	}); err != nil {
		return err
	}
	// ⚠️ **Not unique, and worth saying why.** Every other index here stops a
	// duplicate; this one only makes a thread's lines cheap to read in order.
	// Two messages sent in the same millisecond are two messages.
	if _, err := s.SupportMessages.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "threadId", Value: 1}, {Key: "at", Value: 1}},
	}); err != nil {
		return err
	}
	// ⚠️ **No text index, and that is a correction rather than an omission.**
	// One was created here first, on the reasoning that a regex scan gets
	// slower every month. It also never matched anything: Mongo's text search
	// tokenises against a stemmer with no Uzbek in it, and Uzbek is
	// agglutinative — "printer" does not match "printerdan". See
	// `supportSearch` for what replaced it and why the scan is affordable.

	// ⚠️ **Unique, and the whole grouping depends on it.** Intake is an upsert
	// on (slug, app, key); without the index two reports arriving in the same
	// second create two groups for one fault, and from then on the count is
	// split across rows that look like different bugs. It is the same failure
	// `pos_settings.branchId` had, with a louder symptom: the list stops being
	// a list of faults and becomes a list of occurrences.
	if _, err := s.Reports.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "slug", Value: 1}, {Key: "app", Value: 1}, {Key: "key", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	// The console's own order: what is happening now, across every restaurant.
	if _, err := s.Reports.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "lastAt", Value: -1}},
	}); err != nil {
		return err
	}

	_, err := s.Users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}
