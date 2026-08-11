package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connect establishes a MongoDB connection and pings the server.
//
// ⚠️ **The pool is capped, and the reason is the neighbours.**
//
// One restaurant is one container, but every container on the platform talks to
// **one shared mongod**. The driver's default pool is 100 connections *per
// client*, so the ceiling is not this restaurant's — it is a hundred times the
// number of customers we have sold, and each connection costs the server a
// thread and its stack whether or not anybody is ordering. A busy Friday at one
// restaurant would quietly take capacity from every other restaurant on the box,
// and the only symptom next door is a site that got slower for no visible reason.
//
// 20 is far above what a single restaurant's traffic needs (the whole platform
// idles at about twenty connections *in total*) and low enough that fifty
// tenants cannot exhaust the server between them. `MinPoolSize` keeps a couple
// warm so the first order after a quiet hour does not pay for a handshake.
//
// The timeouts matter for the same reason: a request that waits forever for a
// connection holds a goroutine, a socket and the guest's patience. Failing in
// five seconds is a visible error; hanging is a site that "sometimes doesn't
// work".
func Connect(ctx context.Context, uri, dbName string) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	opts := options.Client().ApplyURI(uri).
		SetMaxPoolSize(20).
		SetMinPoolSize(2).
		// Idle connections are returned to the server rather than held for the
		// life of the container: a restaurant that is busy at lunch should not
		// still be holding lunch's connections at midnight.
		SetMaxConnIdleTime(5 * time.Minute).
		SetServerSelectionTimeout(5 * time.Second)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return client.Database(dbName), nil
}
