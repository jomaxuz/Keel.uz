package handlers

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

// ⚠️ **Sealed because this failed in production on the first sale.** The
// mirrored document carried an `_id` field nothing ever filled in, so every
// attempt to put a restaurant on a till plan died with "performing an update on
// the path '_id' would modify the immutable field '_id'". Mongo refuses `_id`
// inside a `$set` — an empty string is still a value — and the error names a
// path nobody wrote, which is why it reads as a database fault rather than as a
// struct with one field too many.
//
// The test is on the marshalled document rather than on a live update: what
// matters is that the bytes handed to `$set` contain no identity at all.
func TestTillGrantDocCarriesNoID(t *testing.T) {
	raw, err := bson.Marshal(tillGrantDoc{Enabled: true, Plan: "start"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m bson.M
	if err := bson.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["_id"]; ok {
		t.Error("the mirrored subscription names its own _id — every $set " +
			"carrying it is refused by Mongo as an immutable field")
	}
	// The fields the restaurant's own screens read must still be there: a fix
	// that emptied the document would be quieter and worse.
	for _, k := range []string{"enabled", "plan", "modules"} {
		if _, ok := m[k]; !ok {
			t.Errorf("the mirrored subscription lost %q", k)
		}
	}
}
