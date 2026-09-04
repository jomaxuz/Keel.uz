package handlers

import (
	"testing"

	"restaurant-backend/internal/models"
)

// Who may connect a printer from the counter.
//
// ⚠️ **Sealed because this endpoint writes branch-wide configuration from a
// screen standing in a public room.** Everything else a cashier reaches from
// the till affects one check; this changes where every receipt in the building
// comes out, for every till, until somebody changes it back. The rail hides the
// section, but hiding is politeness — the refusal is here.
//
// ⚠️ The permission is `void`, not a new one, and the set of people is the
// point rather than the word: every seeded management role carries it and none
// of the counter roles do. If that ever stops being true, this test is where it
// shows up rather than in a restaurant.
func TestPrinterSettingsAreForManagementRolesOnly(t *testing.T) {
	may := map[string]bool{
		"Ish boshqaruvchi":   true,
		"Menejer":            true,
		"Zal administratori": true,
		"Kassir":             false,
		"Barmen":             false,
		"Ofitsiant":          false,
		"Xostes":             false,
		"Oshxona boshlig'i":  false,
		"Oshpaz":             false,
		"Texnolog":           false,
		// ⚠️ A buyer is out of the building all morning with a phone. The
		// printers are a fact about the room, and this role never stands in it.
		"Zakupshik": false,
		// ⚠️ A storekeeper counts shelves and writes shopping lists. Neither is
		// the dining room, and the printers are the dining room's.
		"Omborchi":        false,
		"Yordamchi xodim": false,
	}
	seen := map[string]bool{}
	for _, role := range models.SeedRoles() {
		want, known := may[role.Name]
		if !known {
			// ⚠️ A new seeded role must be decided here rather than inheriting
			// an answer. "It defaulted to no" is how a manager role ships
			// without the screen its job needs.
			t.Errorf("seeded role %q is not listed in this test — decide whether "+
				"it may configure printers", role.Name)
			continue
		}
		seen[role.Name] = true
		staff := models.Staff{IsActive: true, Perms: role.Perms}
		if got := staff.Can(models.PermVoid); got != want {
			t.Errorf("%s: may configure printers = %v, want %v", role.Name, got, want)
		}
	}
	for name := range may {
		if !seen[name] {
			t.Errorf("role %q is listed here but is no longer seeded", name)
		}
	}
}

// ⚠️ A dismissed employee's token outlives their shift by days, and this
// endpoint is worth more to somebody who no longer works here than any single
// check is. `Can` answers it — the rule lives in one place — but a printer list
// is exactly the kind of screen a later refactor reaches for a role's perms
// directly, which would skip it.
func TestDismissedManagerCannotConfigurePrinters(t *testing.T) {
	perms := []string{models.PermVoid, models.PermCashier}
	if (&models.Staff{IsActive: false, Perms: perms}).Can(models.PermVoid) {
		t.Error("a dismissed manager could still reconfigure the branch's printers")
	}
}

// The empty-slice rule, in the one place a nil list reaches a browser.
//
// ⚠️ A branch that has never connected a printer is the common case for this
// endpoint — it is what the screen exists to fix — so `null` here would render
// `null.length` on the first restaurant to open it. This codebase has shipped
// that bug twice.
func TestPrintersJSONIsNeverNull(t *testing.T) {
	if got := printersJSON(nil); got == nil {
		t.Fatal("a branch with no printers would serialise as null")
	}
	if got := printersJSON(nil); len(got) != 0 {
		t.Fatalf("invented %d printers", len(got))
	}
}
