package db

import (
	"context"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// documentsAsMaps makes the driver decode a document into `bson.M` rather than
// `bson.D` wherever it is read into an `interface{}`.
//
// ⚠️ **This is not a preference, and the constructor is what it broke.** The
// design pipeline passes a band's canvas, its settings, the saved styles, the
// theme and the navigation bar through as `any` — deliberately, so the control
// plane holds no second copy of a schema whose boundary is the tenant's own
// `Sanitize`. Writing them that way is fine. **Reading them back is not**:
// `bson.D` is an ordered *slice of pairs*, so `encoding/json` renders it as
// `[{"Key":"height","Value":86}, …]` rather than as `{"height":86}`.
//
// Nothing errors. The save returns a count, the document in the database is
// correct, and the tenant — which decodes into typed structs — keeps rendering
// the site perfectly. Only the console is wrong, and only after a reload: the
// editor reopens a design whose canvas it cannot read, so the elements are
// gone from the inspector and every settings panel shows its defaults. An
// operator who then saves writes that shape back, and *that* is the version
// the tenant cannot decode.
//
// Registered on the client so it applies to every read in the process, not to
// the one call somebody remembered.
func documentsAsMaps() *bsoncodec.Registry {
	rb := bson.NewRegistryBuilder()
	rb.RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{}))
	return rb.Build()
}

// Connect opens one client. Tenant databases are reached through the same
// client with Database(name) — a pooled connection per tenant would be a
// hundred idle sockets for nothing.
func Connect(ctx context.Context, uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx,
		options.Client().ApplyURI(uri).SetRegistry(documentsAsMaps()))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return client, nil
}
