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

	// The client tenant databases hang off. Separate from DB so the day tenant
	// data moves to another server, only this changes.
	tenantClient *mongo.Client
}

func New(db *mongo.Database, tenantClient *mongo.Client) *Store {
	return &Store{
		DB:           db,
		Tenants:      db.Collection("tenant"),
		Days:         db.Collection("tenant_day"),
		Users:        db.Collection("user"),
		tenantClient: tenantClient,
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
	_, err := s.Users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}
