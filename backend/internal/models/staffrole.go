package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Roles: what a job title is allowed to do ----
//
// # ⚠️ The tension that shapes this whole file
//
// **Too many permissions produce one shared PIN.** A screen where every button
// answers "you may not" teaches a room to solve it once and for all: the
// cashier's code gets told to everybody. After that the journal still records a
// name on every void — always the same name — and the attribution the PIN was
// built for is gone. So a permission exists in exactly two cases: the action
// **takes money out** or **destroys a record**. Everything else simply works.
//
// # ⚠️ A role is a record, not a typed word
//
// CLAUDE.md's warning stands: `Staff.Position` is free text and the system must
// never read it, because treating a typed job title as a permission hands a key
// to whoever spells it the same way. A role is the answer to that, not an
// exception to it — it is chosen from a list, carries an id, and the spelling
// of its name changes nothing.

// The six permissions. Deliberately few — see the note above.
const (
	// PermVoid: removing food the kitchen has already cooked, cancelling a
	// whole check, or reopening a closed one. The oldest way to take money out
	// of a restaurant, and the reason any of this exists.
	PermVoid = "void"
	// PermDiscount: reducing what a table owes. The same route as a void
	// through a different door — which is why they are separate switches: a
	// restaurant that trusts somebody with one may not trust them with both.
	PermDiscount = "discount"
	// PermShift: opening and closing the cash shift, and closing the fiscal
	// day. One permission for both because it is one moment in the evening,
	// and the person counting the drawer is the person filing the Z-report.
	PermShift = "shift"
	// PermKitchen: the pass screen. Already existed as a flag on the employee.
	PermKitchen = "kitchen"
	// PermStock: counting the store from a phone.
	//
	// ⚠️ **It fits the rule rather than bending it.** A count is not a button
	// that takes money out, but it *writes the baseline every later shortfall
	// is measured from* — a saved count silently forgives whatever went missing
	// before it, which is the destroy-a-record half of the test. It is also the
	// one screen that would otherwise put every buying price in the building on
	// a shared tablet.
	//
	// ⚠️ Unlike `kitchen`, nothing has to be grandfathered: this is a new
	// screen, so refusing by default takes nothing away from anybody.
	PermStock = "stock"
)

// AllPerms is every permission a role can carry, in the order the panel draws
// them: floor first, money after, kitchen last.
var AllPerms = []string{
	PermWaiter, PermCashier, PermVoid, PermDiscount, PermShift, PermKitchen,
	PermStock,
}

// StaffRole is a job title and the permissions that come with it.
type StaffRole struct {
	ID   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name string             `bson:"name" json:"name"` // uz (base)

	// Optional translations; empty means "fall back to the base name". Same
	// rule as a category or a dish — the role's name is text the restaurant
	// typed, not a UI string, so it cannot come from the dictionary.
	//
	// ⚠️ **The spelling still changes nothing.** A role is read by its id
	// everywhere (see the note above); these three fields only decide which
	// word a Russian-speaking manager sees in the picker.
	NameRu string `bson:"nameRu" json:"nameRu"`
	NameEn string `bson:"nameEn" json:"nameEn"`

	// What this role may do. A set rather than six booleans so adding a
	// seventh permission does not rewrite every stored document.
	Perms []string `bson:"perms" json:"perms"`

	// ⚠️ Seeded roles are marked so the panel can say which ones it created.
	// They are **fully editable and deletable** anyway: a restaurant whose
	// "Kassir" may not match ours is the ordinary case, and a role that cannot
	// be changed is a role that gets worked around by giving somebody the
	// wrong one.
	Seeded bool `bson:"seeded" json:"seeded"`

	// Whether the one-off grant of `stock` to the shipped Texnolog role has
	// already visited this document.
	//
	// ⚠️ It marks the *visit*, not the outcome, and that is the whole point:
	// matching on "Texnolog without stock" alone would find the role a
	// restaurant had just deliberately unticked, on every boot. A permission
	// that grows back overnight is worse than one that was never granted. See
	// grantTechnologistStock.
	StockGranted bool `bson:"stockGranted,omitempty" json:"-"`

	// Sort order in the panel, so the list reads top-down by authority rather
	// than by whenever somebody happened to add a role.
	Sort      int       `bson:"sort" json:"sort"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// Allows reports whether this role carries a permission.
//
// ⚠️ **Cashier implies waiter here too**, exactly as Staff.Can() has always
// resolved it. The implication lives in one place so a floor screen and a till
// screen cannot one day disagree about it — the disagreement would show up as
// "the button works on his tablet and not on mine".
func (r *StaffRole) Allows(perm string) bool {
	if r == nil {
		return false
	}
	for _, p := range r.Perms {
		if p == perm {
			return true
		}
		if perm == PermWaiter && p == PermCashier {
			return true
		}
	}
	return false
}

// SeedRoles is the set every new restaurant starts with.
//
// ⚠️ **Ish boshqaruvchi and Menejer carry the same permissions on purpose.**
// They are different answers in the journal — "Menejer Dilnoza" and "Ish
// boshqaruvchi Aziz" are not the same sentence — and a restaurant that wants to
// tighten one without the other can, because the roles are editable. Identical
// today is not identical forever.
//
// ⚠️ **Kassir ships without void and discount**, which is a change from the old
// `canCashier` flag: taking payment and writing off cooked food are different
// decisions, and a restaurant that wants them together adds them in one tap.
// Existing cashiers keep both — see EnsureStaffRoles, which must not take away
// a right somebody has been using.
func SeedRoles() []StaffRole {
	rows := SeedRoleRows()
	out := make([]StaffRole, 0, len(rows))
	now := time.Now()
	for i, r := range rows {
		perms := r.Perms
		if perms == nil {
			// ⚠️ Not nil: a nil slice marshals to `null` and the panel would
			// render `null.length`. The JSON trap this codebase has been bitten
			// by twice.
			perms = []string{}
		}
		out = append(out, StaffRole{
			Name: r.Name, NameRu: r.NameRu, NameEn: r.NameEn,
			Perms: perms, Seeded: true, Sort: i,
			CreatedAt: now, UpdatedAt: now,
		})
	}
	return out
}

// SeedRoleRow is one shipped role: its name in the three languages the panel
// speaks, and what it may do.
type SeedRoleRow struct {
	Name   string // uz (base)
	NameRu string
	NameEn string
	Perms  []string
}

// SeedRoleRows is the shipped list itself.
//
// ⚠️ **Exported so the migration can reach it.** The roles we ship were seeded
// with Uzbek names only, and a restaurant whose panel is in Russian saw eleven
// Uzbek words in the picker with no way to fix them short of retyping the list.
// The migration fills the translations in by matching this base name — which
// works only if there is exactly one list to match against.
func SeedRoleRows() []SeedRoleRow {
	return []SeedRoleRow{
		{"Ish boshqaruvchi", "Управляющий", "General manager", []string{
			PermWaiter, PermCashier, PermVoid, PermDiscount, PermShift, PermKitchen}},
		{"Menejer", "Менеджер", "Manager", []string{
			PermWaiter, PermCashier, PermVoid, PermDiscount, PermShift, PermKitchen}},
		{"Zal administratori", "Администратор зала", "Floor supervisor", []string{
			PermWaiter, PermCashier, PermVoid, PermDiscount, PermShift}},
		{"Kassir", "Кассир", "Cashier", []string{PermWaiter, PermCashier, PermShift}},
		{"Barmen", "Бармен", "Bartender", []string{PermWaiter, PermCashier, PermKitchen}},
		{"Ofitsiant", "Официант", "Waiter", []string{PermWaiter}},
		{"Xostes", "Хостес", "Host", nil},
		{"Oshxona boshlig'i", "Шеф-повар", "Head chef", []string{PermKitchen}},
		{"Oshpaz", "Повар", "Cook", []string{PermKitchen}},
		// ⚠️ **The one seeded role that ships with `stock`.** A technologist
		// writes the tech cards and runs the counts — the work is the store —
		// and the panel's own door for them is that permission
		// (handlers/stocklogin.go). Shipping it empty meant every restaurant
		// hired a technologist and then discovered, at the first count, that
		// the account they were given refuses the login form: "login yoki parol
		// noto'g'ri", which reads as a broken password rather than as a
		// permission nobody switched on.
		//
		// It stays editable like every other seeded role: a restaurant whose
		// technologist should not price the store unticks it in one tap.
		{"Texnolog", "Технолог", "Food technologist", []string{PermStock}},
		{"Yordamchi xodim", "Подсобный работник", "Kitchen porter", nil},
	}
}
