package db

import (
	"encoding/json"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

// ⚠️ **The shape the constructor is served, and the one it cannot read.**
//
// The design pipeline passes a band's canvas, its settings, the saved styles,
// the theme and the navigation bar through as `any`, so the control plane holds
// no second copy of a schema whose boundary is the tenant's own Sanitize.
// Written that way they are correct in the database. Read back into an
// `interface{}` with the driver's defaults they come out as `bson.D` — an
// ordered slice of pairs — and `encoding/json` renders that as
// `[{"Key":"height","Value":86}]` rather than `{"height":86}`.
//
// Nothing errors anywhere: the save returns a count, the stored document is
// right, and the tenant keeps rendering the site because it decodes into typed
// structs. Only the console is wrong, and only after a reload — the editor
// reopens a design whose canvas it cannot read. An operator who saves from
// there writes that shape back, and that version is the one the tenant cannot
// decode.
//
// Hermetic on purpose: it needs no server, so it runs in the same suite as
// everything else rather than in the one nobody remembers to start Mongo for.
func TestDocumentsComeBackAsObjectsRatherThanKeyValuePairs(t *testing.T) {
	stored, err := bson.Marshal(bson.M{"canvas": bson.M{
		"height": 86,
		"elements": bson.A{
			bson.M{"type": "box", "box": bson.M{"x": 64, "y": 0}},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}

	// ⚠️ **Decoded into a struct field, because that is the only place the type
	// map is consulted.** Unmarshalling straight into a bare `interface{}`
	// still yields `bson.D` even with this registry — a real difference, and
	// the one that made the first version of this test fail against a fix that
	// works. The pipeline decodes into `designDoc`, whose `Canvas`, `Settings`,
	// `Nav` and `Theme` are `any` fields on a struct; that is what is modelled
	// here.
	var out struct {
		Canvas any `bson:"canvas"`
	}
	if err := bson.UnmarshalWithRegistry(documentsAsMaps(), stored, &out); err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(out.Canvas)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(got), `"Key"`) {
		t.Fatalf("a document came back as ordered pairs, which the console reads as nothing:\n%s", got)
	}
	// Nested documents too: the failure that matters is an element's box, four
	// levels down, not the top-level object.
	if !strings.Contains(string(got), `"x":64`) {
		t.Errorf("a nested document lost its keys:\n%s", got)
	}
	if !strings.Contains(string(got), `"height":86`) {
		t.Errorf("a top-level key was lost:\n%s", got)
	}
}

// And the default registry is the counter-example, so the test above is
// measuring the fix rather than something that was always true.
func TestTheDriverDefaultIsWhatBrokeIt(t *testing.T) {
	stored, _ := bson.Marshal(bson.M{"canvas": bson.M{"height": 86}})
	var out struct {
		Canvas any `bson:"canvas"`
	}
	if err := bson.Unmarshal(stored, &out); err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(out.Canvas)
	if !strings.Contains(string(got), `"Key"`) {
		t.Skip("the driver's default no longer yields bson.D — the registry may be unnecessary")
	}
}
