package models

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// `Sanitize` is a security boundary, not a tidy-up: this document is turned into
// styles on a page, and it is written by a different codebase (the console) on a
// different deploy schedule. Everything below is a way a document could arrive
// that the renderer must not be handed.

func TestSanitizeDropsWhatItDoesNotRecognise(t *testing.T) {
	d := &PageDesign{Sections: []DesignSection{
		{Type: BlockHero, Variant: "full", Span: 12},
		// A block type from a newer console, or a typo. Dropped: a band nobody
		// recognises should disappear, not appear as a mystery.
		{Type: "carousel-3d", Variant: "full", Span: 12},
		// A known block with an unknown variant is still worth rendering — it
		// falls back to the block's first variant rather than vanishing.
		{Type: BlockMenuGrid, Variant: "masonry", Span: 6},
	}}
	d.Sanitize()

	if len(d.Sections) != 2 {
		t.Fatalf("bo'limlar soni %d, kutilgan 2: %+v", len(d.Sections), d.Sections)
	}
	if d.Sections[1].Type != BlockMenuGrid || d.Sections[1].Variant != "cards" {
		t.Errorf("noma'lum variant birinchisiga tushmadi: %+v", d.Sections[1])
	}
	if d.Sections[1].Span != 6 {
		t.Errorf("to'g'ri span o'zgardi: %d", d.Sections[1].Span)
	}
}

func TestSanitizeClampsTheGrid(t *testing.T) {
	// Zero is what an older document (written before the field existed) and a
	// cleared field both look like, and a full-width band is the safe reading of
	// both. Everything outside 1-12 is not a layout, it is a number somebody
	// typed.
	for _, span := range []int{0, -3, 13, 999} {
		d := &PageDesign{Sections: []DesignSection{{Type: BlockHero, Span: span}}}
		d.Sanitize()
		if d.Sections[0].Span != 12 {
			t.Errorf("span %d -> %d, kutilgan 12", span, d.Sections[0].Span)
		}
	}
	for _, span := range []int{1, 6, 12} {
		d := &PageDesign{Sections: []DesignSection{{Type: BlockHero, Span: span}}}
		d.Sanitize()
		if d.Sections[0].Span != span {
			t.Errorf("haqiqiy span %d o'zgardi -> %d", span, d.Sections[0].Span)
		}
	}
}

// The reason there is no free-form style field anywhere: this document becomes
// CSS. A tone that is not a design token is dropped rather than passed through,
// so there is nothing to escape and nothing to inject.
func TestSanitizeRefusesStyleValuesOutsideTheDesignSystem(t *testing.T) {
	d := &PageDesign{Sections: []DesignSection{{
		Type: BlockHero, Variant: "full", Span: 12,
		Style: DesignStyle{
			Tone:    "#f00; background-image:url(//evil/x)",
			Padding: "999px",
			Align:   "justify",
		},
	}}}
	d.Sanitize()
	got := d.Sections[0].Style
	if got.Tone != "" || got.Padding != "" || got.Align != "" {
		t.Fatalf("dizayn tizimidan tashqaridagi qiymat o'tib ketdi: %+v", got)
	}

	d = &PageDesign{Sections: []DesignSection{{
		Type: BlockHero, Span: 12,
		Style: DesignStyle{Tone: "charcoal", Padding: "lg", Align: "center"},
	}}}
	d.Sanitize()
	if d.Sections[0].Style.Tone != "charcoal" || d.Sections[0].Style.Padding != "lg" {
		t.Fatalf("haqiqiy tokenlar tushib qoldi: %+v", d.Sections[0].Style)
	}
}

func TestSanitizeIgnoresAnImpossibleLimit(t *testing.T) {
	d := &PageDesign{Sections: []DesignSection{
		{Type: BlockMenuGrid, Span: 12, Binding: DesignBinding{Limit: 5000}},
		{Type: BlockMenuGrid, Span: 12, Binding: DesignBinding{Limit: -1}},
		{Type: BlockMenuGrid, Span: 12, Binding: DesignBinding{Limit: 8}},
	}}
	d.Sanitize()
	if d.Sections[0].Binding.Limit != 0 || d.Sections[1].Binding.Limit != 0 {
		t.Errorf("mumkin bo'lmagan limit o'tdi: %+v", d.Sections)
	}
	if d.Sections[2].Binding.Limit != 8 {
		t.Errorf("haqiqiy limit o'zgardi: %d", d.Sections[2].Binding.Limit)
	}
}

// An empty page is worse than an unstyled one, so a design that survives nothing
// must not replace the template.
func TestRenderableFallsBackRatherThanShowingNothing(t *testing.T) {
	var missing *PageDesign
	if missing.Renderable() {
		t.Error("dizayn yo'q, lekin renderable")
	}

	draft := &PageDesign{Status: DesignDraft, Sections: DefaultSections()}
	if draft.Renderable() {
		t.Error("qoralama jonli saytga chiqdi")
	}

	emptied := &PageDesign{Status: DesignPublished, Sections: []DesignSection{
		{Type: "unknown-block", Span: 12},
	}}
	emptied.Sanitize()
	if emptied.Renderable() {
		t.Error("tozalangandan keyin bo'sh qolgan dizayn renderable bo'lib qoldi")
	}

	live := &PageDesign{Status: DesignPublished, Sections: DefaultSections()}
	live.Sanitize()
	if !live.Renderable() {
		t.Error("haqiqiy dizayn render qilinmadi")
	}
}

// The fallback is the feature's precondition: existing customers must see no
// change at all. So the default list has to be the page they already have —
// every band, in the order app/(site)/page.tsx renders them.
func TestDefaultSectionsAreTodaysPage(t *testing.T) {
	got := DefaultSections()
	want := []string{
		BlockHero, BlockPerks, BlockCategories, BlockMenuGrid, BlockHoursAddress,
	}
	if len(got) != len(want) {
		t.Fatalf("bo'limlar soni %d, kutilgan %d", len(got), len(want))
	}
	for i, tp := range want {
		if got[i].Type != tp {
			t.Errorf("%d-o'rinda %q, kutilgan %q", i, got[i].Type, tp)
		}
		if got[i].Span != 12 {
			t.Errorf("%s to'liq kenglikda emas: %d", got[i].Type, got[i].Span)
		}
	}
	if got[0].Style.Tone != "charcoal" {
		t.Errorf("hero ohangi o'zgardi: %q", got[0].Style.Tone)
	}
	if !got[3].Binding.PopularOnly || got[3].Binding.Limit != 8 {
		t.Errorf("mashhur taomlar bloki o'zgardi: %+v", got[3].Binding)
	}

	// And it survives its own sanitiser — a default list that Sanitize would
	// drop is a default list nobody would ever see.
	d := &PageDesign{Status: DesignPublished, Sections: got}
	d.Sanitize()
	if len(d.Sections) != len(want) {
		t.Fatalf("standart ro'yxat o'z tozalovchisidan o'tmadi: %d", len(d.Sections))
	}
}

func TestSanitizeKeepsHiddenBandsAndIds(t *testing.T) {
	id := primitive.NewObjectID()
	d := &PageDesign{Sections: []DesignSection{{
		Type: BlockMenuGrid, Variant: "cards", Span: 12, Hidden: true,
		Binding: DesignBinding{Categories: []primitive.ObjectID{id}},
	}}}
	d.Sanitize()
	// Hidden is kept: an operator trying a layout needs to take a band out and
	// put it back, and deleting it loses everything that was tuned on it.
	if !d.Sections[0].Hidden {
		t.Error("yashirilgan bo'lim tozalovchida yo'qoldi")
	}
	if len(d.Sections[0].Binding.Categories) != 1 || d.Sections[0].Binding.Categories[0] != id {
		t.Error("kategoriya bog'lanishi yo'qoldi")
	}
}
