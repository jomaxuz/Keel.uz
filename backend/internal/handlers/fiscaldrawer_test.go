package handlers

import (
	"testing"

	"restaurant-backend/internal/fiscal"
	"restaurant-backend/internal/models"
)

// ⚠️ **Every provider the panel offers must have a drawer, everywhere.**
//
// This was three switches, a struct literal and a `$set` naming six providers
// each. Adding a seventh meant editing four lists, and missing one is silent in
// the worst way this file has available: the provider appears in the panel, the
// owner types their login, it saves — and the filing code reads an empty drawer
// and files nothing at all. Nobody finds out until an inspector asks for a
// receipt that was never sent.
func TestEveryOfferedProviderHasSomewhereToStoreItsLogin(t *testing.T) {
	stored := drawers(&models.FiscalSettings{})
	sent := fiscalUpdateRequest{}.sent()
	for _, p := range fiscal.Providers() {
		if _, ok := stored[p.ID]; !ok {
			t.Errorf("%q is offered in the panel but has no stored drawer", p.ID)
		}
		if _, ok := sent[p.ID]; !ok {
			t.Errorf("%q is offered in the panel but the save request cannot carry it", p.ID)
		}
	}
}

// And the other direction: a drawer nothing offers is a field nobody can fill.
func TestNoDrawerBelongsToAProviderThatIsNotOffered(t *testing.T) {
	for id := range drawers(&models.FiscalSettings{}) {
		if !fiscal.Known(id) {
			t.Errorf("there is a drawer for %q, which the panel does not offer", id)
		}
	}
}

// ⚠️ Rahmat's cloud register is not the program Rahmat resells for the till
// computer. One is dialled from our server with an account; the other sits on
// the restaurant's LAN with no authentication at all. Sharing an id would point
// a cloud customer at their own office network.
func TestRahmatCloudIsNotTheLocalProgram(t *testing.T) {
	if fiscal.Rahmat == fiscal.Multikassa {
		t.Fatal("the cloud register and the on-premise program share an id")
	}
	if fiscal.IsLocal(fiscal.Rahmat) {
		t.Error("the cloud register is marked as running inside the restaurant")
	}
	if !fiscal.IsLocal(fiscal.Multikassa) {
		t.Error("the on-premise program is no longer marked local")
	}
}

// ⚠️ A provider with no adapter must not be enable-able. Selecting it and
// saving credentials is right — an owner sets this up before the contract
// completes — but a restaurant that believes it is filing and is not has a
// legal problem, not a missing feature.
func TestAProviderWithNoAdapterIsNotReady(t *testing.T) {
	for _, id := range []string{fiscal.Rahmat, fiscal.QPOS, fiscal.Arca} {
		if fiscal.Ready(id) {
			t.Errorf("%q claims an adapter that has not been written", id)
		}
	}
}

// ⚠️ **A stale browser tab must not wipe working credentials.**
//
// The panel used to send a field per provider and now sends one map. For the
// minutes after a deploy a tab is still running the old panel; if the server
// ignored its shape, that tab's next save would write empty drawers over
// credentials that were filing receipts — silently, with no error anywhere, and
// the restaurant would simply stop being registered.
func TestAnOldPanelsSaveStillCarriesItsCredentials(t *testing.T) {
	old := fiscalUpdateRequest{
		Provider:   fiscal.Multikassa,
		Multikassa: models.FiscalCreds{Login: "kassa-1", BaseURL: "http://192.168.14.65:9090"},
	}
	got := old.sent()[fiscal.Multikassa]
	if got.Login != "kassa-1" || got.BaseURL == "" {
		t.Fatalf("an old panel's credentials were dropped: %+v", got)
	}
}

// And the current shape wins where both are present.
func TestTheCurrentShapeWinsOverTheLegacyFields(t *testing.T) {
	req := fiscalUpdateRequest{
		Multikassa: models.FiscalCreds{Login: "old"},
		Creds:      map[string]models.FiscalCreds{fiscal.Multikassa: {Login: "new"}},
	}
	if got := req.sent()[fiscal.Multikassa].Login; got != "new" {
		t.Fatalf("login = %q, want the map's value", got)
	}
}

// ⚠️ A provider id nobody offers must not create a drawer: the map arrives from
// a browser, and `$set` uses these ids as bson keys.
func TestAnUnknownProviderIdIsIgnored(t *testing.T) {
	req := fiscalUpdateRequest{
		Creds: map[string]models.FiscalCreds{"$where": {Login: "nope"}},
	}
	if _, ok := req.sent()["$where"]; ok {
		t.Fatal("an unknown provider id became a stored field name")
	}
}
