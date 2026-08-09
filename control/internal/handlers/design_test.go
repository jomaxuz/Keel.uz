package handlers

import (
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
