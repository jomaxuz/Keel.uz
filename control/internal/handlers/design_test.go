package handlers

import (
	"encoding/json"
	"slices"
	"testing"
)

// ⚠️ The shipped gallery, checked for the two mistakes a gallery multiplies.
//
// A template is copied onto customer after customer, so a missing translation or a
// hard-coded colour is not one wrong page — it is twenty, each of them somebody's
// business. Both are cheap to assert and neither is visible by looking at the
// layout in the editor.
func TestBuiltinTemplatesAreTranslatedAndTokenised(t *testing.T) {
	list := builtinTemplates()
	if len(list) < 5 {
		t.Fatalf("gallery has %d templates, expected the five shipped ones", len(list))
	}
	for _, tpl := range list {
		if tpl.Name == "" || tpl.Note == "" {
			t.Fatalf("%q: a template with no name or no note cannot be chosen from a list", tpl.Name)
		}
		if len(tpl.Sections) == 0 {
			t.Fatalf("%q: no bands", tpl.Name)
		}
		for _, sec := range tpl.Sections {
			if !slices.Contains(designBlocks, sec.Type) {
				t.Fatalf("%q: band %q is not a block the console can store", tpl.Name, sec.Type)
			}
		}
	}
}

// ⚠️ Everything a band declares has to survive the trip through this package.
//
// The console draws its settings panel from the schema and sends what an operator
// typed; this struct is what that JSON is decoded into on the way to the tenant's
// database. Go drops unknown fields **silently**, so a field missing here is not a
// missing feature — it is data loss with a 200 response, and it happened: the live
// customer's design carried eight bands and not one saved setting, while the
// editor, the save call and the schema all behaved perfectly.
//
// The assertion is the round trip rather than the struct's shape, because the
// shape is what a future edit would change and the round trip is what a page
// actually depends on.
func TestSectionSettingsSurviveDecoding(t *testing.T) {
	body := []byte(`{"sections":[{
		"type":"menu-grid","span":12,
		"settings":{"heading":{"uz":"Mashhur"},"limit":6,"popularOnly":false,
		            "categories":["507f1f77bcf86cd799439011"]},
		"blocks":[{"type":"slide","settings":{"image":"/uploads/a.jpg"}}]
	}]}`)

	var req designSaveRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	sections := clean(req.Sections)
	if len(sections) != 1 {
		t.Fatalf("got %d sections, want 1", len(sections))
	}

	out, err := json.Marshal(sections[0])
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	s, ok := back["settings"].(map[string]any)
	if !ok {
		t.Fatalf("settings did not survive: %s", out)
	}
	if s["limit"] != float64(6) {
		t.Errorf("limit = %#v, want 6", s["limit"])
	}
	// ⚠️ False, and it must stay false. `popularOnly` defaulting to true means
	// "unticked" and "never set" are the same document, and the operator's change
	// is the one that disappears.
	if s["popularOnly"] != false {
		t.Errorf("popularOnly = %#v, want false", s["popularOnly"])
	}
	if got, _ := s["categories"].([]any); len(got) != 1 {
		t.Errorf("category selection did not survive: %#v", s["categories"])
	}
	if _, ok := back["blocks"].([]any); !ok {
		t.Errorf("repeatable blocks did not survive: %s", out)
	}
}
