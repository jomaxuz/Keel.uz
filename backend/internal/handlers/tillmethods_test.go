package handlers

import (
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ The owner's buttons are names over three kinds, and the kind is what the
// drawer and the reports read — so a row that cannot say which kind it is has
// no place on the till.
func TestTillMethodsFromCleansTheForm(t *testing.T) {
	got, err := tillMethodsFrom([]tillMethodInput{
		{ID: "cash", Kind: "cash", Enabled: true},                       // a default, nameless: kept
		{ID: "", Name: "Humo terminal", Kind: "card", Enabled: true},    // new: gets an id
		{ID: "m_old", Name: "", Kind: "transfer", Enabled: true},        // custom with no name: dropped
		{ID: "m_bad", Name: "Dollar", Kind: "usd", Enabled: true},       // unknown kind: dropped
		{ID: "cash", Name: "Naqd 2", Kind: "cash", Enabled: true},       // duplicate id: dropped
		{ID: "", Name: "", Kind: "card", Enabled: true},                 // empty row: dropped
		{ID: "m_off", Name: "Beznal", Kind: "transfer", Enabled: false}, // switched off: kept
	})
	if err != nil {
		t.Fatalf("a form with enabled buttons was refused: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d buttons, want 3: %+v", len(got), got)
	}
	if got[0].ID != "cash" || got[0].Name != "" {
		t.Fatalf("the default cash button changed: %+v", got[0])
	}
	if len(got[1].ID) < 3 || got[1].ID[:2] != "m_" || got[1].Name != "Humo terminal" || got[1].Kind != "card" {
		t.Fatalf("a new button was not given its own id: %+v", got[1])
	}
	if got[2].ID != "m_off" || got[2].Enabled {
		t.Fatalf("a switched-off button was not kept as off: %+v", got[2])
	}

	if _, err := tillMethodsFrom([]tillMethodInput{
		{ID: "cash", Kind: "cash", Enabled: false},
	}); err != errNoTillMethod {
		t.Fatalf("a till with nothing switched on was accepted: %v", err)
	}
}

// Before the owner touches anything the till offers what it always did, and a
// button switched off still resolves — a sale queued offline on it must keep
// its name.
func TestTillMethodDefaultsAndLookup(t *testing.T) {
	var s models.PaymentSettings
	if n := len(s.EnabledTillMethods()); n != 3 {
		t.Fatalf("defaults: %d buttons, want cash, card and transfer", n)
	}
	s.TillMethods = []models.TillMethod{
		{ID: "cash", Kind: "cash", Enabled: true},
		{ID: "m_humo", Name: "Humo", Kind: "card", Enabled: false},
	}
	if n := len(s.EnabledTillMethods()); n != 1 {
		t.Fatalf("enabled: %d, want 1", n)
	}
	m, ok := s.TillMethodByID("m_humo")
	if !ok || m.Kind != "card" {
		t.Fatalf("a switched-off button did not resolve: %+v %v", m, ok)
	}
	if _, ok := s.TillMethodByID("transfer"); ok {
		t.Fatal("a default the owner replaced still resolved")
	}
}

// ⚠️ Uzum Tezkor orders arrive by themselves, already paid to Uzum — a till
// button for them is an invitation to ring the same order up twice.
func TestTillOfferLeavesOutUzumTezkor(t *testing.T) {
	s := &models.PaymentSettings{
		Aggregators: []models.AggregatorAccount{
			{ID: models.ProviderUzumTezkor, Name: "Uzum Tezkor", Enabled: true},
			{ID: models.ProviderYandexEats, Name: "Yandex Eats", Enabled: true},
		},
		TillMethods: []models.TillMethod{
			{ID: "cash", Kind: "cash", Enabled: true},
			{ID: "m_humo", Name: "Humo", Kind: "card", Enabled: true},
			{ID: "m_uzcard", Name: "Uzcard", Kind: "card", Enabled: true},
		},
	}
	methods, options := tillOffer(s)
	for _, m := range methods {
		if m == models.ProviderUzumTezkor {
			t.Fatalf("Uzum Tezkor is offered on the till: %v", methods)
		}
	}
	want := []string{"cash", "card", models.ProviderYandexEats, models.MethodDebt}
	if len(methods) != len(want) {
		t.Fatalf("methods = %v, want %v", methods, want)
	}
	for i := range want {
		if methods[i] != want[i] {
			t.Fatalf("methods = %v, want %v", methods, want)
		}
	}
	if len(options) != 3 {
		t.Fatalf("options = %+v, want all three buttons", options)
	}
}
