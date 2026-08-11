package models

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

// A brand with no copy of its own must keep showing the company's, and a brand
// whose *only* edit was the perks strip must not be mistaken for one.
//
// ⚠️ The check used to be `b.Content != SiteContent{}`, which the compiler
// enforced for free. A slice field ended that, so the rule now lives in a
// method — and a method can be left behind when a field is added. That is what
// this test is for.
func TestSiteContentIsEmpty(t *testing.T) {
	if !(SiteContent{}).IsEmpty() {
		t.Fatal("zero SiteContent should be empty")
	}
	cases := map[string]SiteContent{
		"tagline":    {Tagline: LocalizedText{Uz: "Issiq va tez"}},
		"aboutTitle": {AboutTitle: LocalizedText{Uz: "Biz haqimizda"}},
		"aboutText":  {AboutText: LocalizedText{Uz: "..."}},
		"footerNote": {FooterNote: LocalizedText{Uz: "..."}},
		"perks":      {Perks: []PerkCard{{Icon: "truck"}}},
		// The one that is nothing but a boolean: a restaurant that only switched
		// the strip off has still edited its site, and reading that as "empty"
		// would hand it the company's copy and turn the strip back on.
		"hidePerks": {HidePerks: true},
	}
	for name, c := range cases {
		if c.IsEmpty() {
			t.Errorf("%s: content with a value reported empty", name)
		}
	}
}

// The strip is stored on the same document the panel patches field by field, so
// what matters is that a card survives the round trip under the names the
// frontend reads.
func TestPerkCardRoundTrip(t *testing.T) {
	in := SiteContent{
		Perks: []PerkCard{{
			Icon:  "leaf",
			Title: LocalizedText{Uz: "Yangi mahsulot", Ru: "Свежие продукты"},
			Text:  LocalizedText{Uz: "Har kuni bozordan"},
		}},
	}
	raw, err := bson.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var doc bson.M
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["perks"]; !ok {
		t.Fatalf("perks missing from document: %v", doc)
	}
	var out SiteContent
	if err := bson.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Perks) != 1 || out.Perks[0].Icon != "leaf" ||
		out.Perks[0].Title.Ru != "Свежие продукты" {
		t.Fatalf("card did not survive the round trip: %+v", out.Perks)
	}
}
