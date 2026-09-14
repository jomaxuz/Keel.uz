package handlers

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// ⚠️ What goes to Uzum is only what Uzum can sell without asking a question it
// has nowhere to ask — and a dish switched off stays listed, so turning it back
// on reaches Uzum in five minutes rather than an hour.
func TestTezkorCatalogKeepsOnlyWhatCanBeSold(t *testing.T) {
	soups, hidden, empty := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()
	cats := []models.Category{
		{ID: soups, Name: "Sho'rvalar", IsActive: true, SortOrder: 2},
		{ID: hidden, Name: "Yopiq", IsActive: false},
		{ID: empty, Name: "Bo'sh", IsActive: true, SortOrder: 1},
	}
	old, same := 40000, 30000
	vat := 12
	plain := models.MenuItem{
		ID: primitive.NewObjectID(), CategoryID: soups, Name: "Mastava", Price: 30000,
		OldPrice: &old, VatPercent: &vat, Ikpu: "10202001010000004", IsAvailable: true,
		ImageURL: "/uploads/seed/mastava.jpg", Images: []string{"/uploads/seed/mastava.jpg", "https://other.site/x.jpg"},
	}
	off := models.MenuItem{ID: primitive.NewObjectID(), CategoryID: soups, Name: "Sho'rva", Price: 30000, OldPrice: &same}
	withBarcode := models.MenuItem{ID: primitive.NewObjectID(), CategoryID: soups, Name: "Suv", Price: 5000, Barcode: "4780001234567"}
	model := models.MenuItem{ID: primitive.NewObjectID(), CategoryID: soups, Name: "Ko'ylak", Price: 1, VariantAxes: []string{"size"}}
	required := models.MenuItem{ID: primitive.NewObjectID(), CategoryID: soups, Name: "Pitsa", Price: 50000,
		Options: []models.MenuOption{{Name: "Hajmi", Required: true}}}
	optional := models.MenuItem{ID: primitive.NewObjectID(), CategoryID: soups, Name: "Choy", Price: 3000,
		Options: []models.MenuOption{{Name: "Shakar", Required: false}}}
	free := models.MenuItem{ID: primitive.NewObjectID(), CategoryID: soups, Name: "Non", Price: 0}
	inHidden := models.MenuItem{ID: primitive.NewObjectID(), CategoryID: hidden, Name: "Eski", Price: 1000}

	images := func(u string) (tezkorImage, bool) {
		if name, ok := tezkorUploadName(u); ok {
			return tezkorImage{Hash: "h:" + name, URL: "https://r.uz/uploads/" + name}, true
		}
		return tezkorImage{}, false
	}
	got := tezkorCatalog(cats, []models.MenuItem{plain, off, withBarcode, model, required, optional, free, inHidden}, images)

	names := map[string]tezkorProduct{}
	for _, p := range got.Items {
		names[p.Name] = p
		if p.Images == nil {
			t.Fatalf("%s: images is nil, which marshals as null", p.Name)
		}
	}
	for _, want := range []string{"Mastava", "Sho'rva", "Suv", "Choy"} {
		if _, ok := names[want]; !ok {
			t.Fatalf("%s is missing from the catalogue: %+v", want, got.Items)
		}
	}
	if len(got.Items) != 4 {
		t.Fatalf("got %d items, want 4 (no model, no required choice, no zero price, no hidden category)", len(got.Items))
	}
	if len(got.Categories) != 1 || got.Categories[0].Name != "Sho'rvalar" {
		t.Fatalf("categories = %+v, want only the one with dishes in it", got.Categories)
	}

	m := names["Mastava"]
	if m.OldPrice == nil || *m.OldPrice != 40000 {
		t.Fatalf("a higher old price is the promotion: %+v", m.OldPrice)
	}
	if names["Sho'rva"].OldPrice != nil {
		t.Fatal("an old price equal to the price is a stale field, not a promotion")
	}
	if len(m.Images) != 1 {
		t.Fatalf("images = %+v, want the one local file once", m.Images)
	}
	if m.ServiceCodesUz == nil || m.ServiceCodesUz.Mxik != "10202001010000004" || m.Vat == nil || *m.Vat != 12 {
		t.Fatalf("fiscal codes lost: %+v vat=%v", m.ServiceCodesUz, m.Vat)
	}
	if names["Suv"].ServiceCodesUz != nil {
		t.Fatal("a dish with no ИКПУ must send none rather than an empty code")
	}
	if names["Suv"].Barcode.Value != "4780001234567" || m.Barcode.Value != plain.ID.Hex() {
		t.Fatalf("barcode: %+v / %+v", names["Suv"].Barcode, m.Barcode)
	}
	if m.Barcode.WeightEncoding != "none" || m.VendorCode != plain.ID.Hex() {
		t.Fatalf("required fields: %+v", m)
	}
}

// Every reason our own site refuses a sale, and the real number a daily limit
// leaves.
func TestTezkorStock(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	osh := models.MenuItem{ID: primitive.NewObjectID(), IsAvailable: true}
	manti := models.MenuItem{ID: primitive.NewObjectID(), IsAvailable: true}
	off := models.MenuItem{ID: primitive.NewObjectID(), IsAvailable: false}
	combo := models.MenuItem{ID: primitive.NewObjectID(), IsAvailable: true,
		ComboItems: []models.ComboLine{{MenuItemID: manti.ID, Qty: 1}}}
	limited := models.MenuItem{ID: primitive.NewObjectID(), IsAvailable: true}

	branch := &models.Branch{
		DailyLimits: []models.DailyLimit{{MenuItemID: limited.ID, Limit: 5}},
		POSSoldOut:  []primitive.ObjectID{manti.ID},
		LimitDate:   today,
	}
	sold := map[primitive.ObjectID]int{limited.ID: 3}

	cases := []struct {
		name string
		item *models.MenuItem
		want float64
	}{
		{"on, no limit", &osh, tezkorOpenStock},
		{"on a stop list", &manti, 0},
		{"switched off", &off, 0},
		{"combo with a stopped member", &combo, 0},
		{"daily limit, 3 of 5 sold", &limited, 2},
	}
	for _, c := range cases {
		if got := tezkorStock(c.item, branch, sold); got != c.want {
			t.Fatalf("%s: stock %v, want %v", c.name, got, c.want)
		}
	}
	sold[limited.ID] = 9
	if got := tezkorStock(&limited, branch, sold); got != 0 {
		t.Fatalf("sold past the limit: stock %v, want 0", got)
	}
}

// Both stored shapes resolve to a file under the upload directory, and nothing
// else does.
func TestTezkorUploadName(t *testing.T) {
	cases := map[string]string{
		"/uploads/seed/osh.jpg":                  "seed/osh.jpg",
		"https://r.uz/uploads/abc.jpg?w=600":     "abc.jpg",
		"/uploads/../../etc/passwd":              "etc/passwd", // cleaned inside the root, never above it
		"/uploads/.thumb/600/abc.jpg":            "",
		"https://cdn.example.com/photos/osh.jpg": "",
		"":                                       "",
	}
	for in, want := range cases {
		got, ok := tezkorUploadName(in)
		if want == "" {
			if ok {
				t.Fatalf("%q resolved to %q, want nothing", in, got)
			}
			continue
		}
		if !ok || got != want {
			t.Fatalf("%q resolved to %q (%v), want %q", in, got, ok, want)
		}
	}
}
