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
	Days    *mongo.Collection
	Users   *mongo.Collection
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

	// The client tenant databases hang off. Separate from DB so the day tenant
	// data moves to another server, only this changes.
	tenantClient *mongo.Client
}

func New(db *mongo.Database, tenantClient *mongo.Client) *Store {
	return &Store{
		DB:              db,
		Tenants:         db.Collection("tenant"),
		Days:            db.Collection("tenant_day"),
		Users:           db.Collection("user"),
		Rollouts:        db.Collection("rollout"),
		DesignTemplates: db.Collection("design_template"),
		Invoices:        db.Collection("invoice"),
		Collector:       db.Collection("collector_run"),
		Status:          db.Collection("status_hour"),
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
	_, err := s.Users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}
