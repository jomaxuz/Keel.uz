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

// The permissions. Deliberately few — see the note above.
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
	// PermBuy: recording a market run from a phone — what was bought, how much
	// of it, and at what price.
	//
	// ⚠️ **Its own permission rather than a corner of `stock`, and the two are
	// nearly opposites.** A storekeeper counts what is on the shelf and gets a
	// panel login with it (handlers/stocklogin.go); a buyer never sees the
	// panel and instead does the one thing counting cannot: he *writes prices*,
	// and a price written at a market repriced every dish that uses it. Folding
	// the two together would hand each of them the other's most expensive
	// button.
	//
	// ⚠️ Nothing is grandfathered: the screen is new, so refusing by default
	// takes nothing away from anybody.
	PermBuy = "buy"
	// PermBuyOrder: writing the shopping list somebody is sent to the market
	// with, and sending it.
	//
	// ⚠️ **Separate from `buy`, and the two are the two halves of a
	// supervision.** One person decides what the restaurant needs; another goes
	// and gets it and writes down what it cost. Held by the same account, the
	// list stops being a check on the trip and becomes a note the buyer wrote
	// to themselves — which is the whole thing this feature was asked for.
	PermBuyOrder = "buyorder"
	// PermStockIssue: handing out what somebody asked for from the store —
	// picking the lines of a shopping request whose goods are already in the
	// building and sending them to the person who asked.
	//
	// ⚠️ **Its own permission rather than a corner of `stock`, for the same
	// reason `buy` is not one.** Counting a shelf and emptying it are different
	// acts: a count writes down what is there, an issue decides who gets it.
	// Folded together, every person given a phone to count the fridge would
	// also hold the button that sends a case of vodka across town — and the
	// restaurant would have granted that without ever being asked.
	//
	// ⚠️ Nothing is grandfathered except the shipped «Omborchi» role, which is
	// the job itself (EnsureStockIssue): the screen is new, so refusing by
	// default takes nothing away from anybody.
	PermStockIssue = "stockissue"
	// PermMarking: scanning the state's marking codes off a delivery, from a
	// phone or from the panel.
	//
	// ⚠️ **Its own permission because a scan is a record, not a reading.** A
	// code filed here is filed for good: `marked_unit.code` is unique across the
	// whole platform, so a bottle scanned into the wrong branch cannot then be
	// received by the branch that actually has it, and a code typed in by
	// mistake refuses a real sale weeks later at a counter. That is the
	// destroy-a-record half of the test this file opens with.
	//
	// ⚠️ **Not a corner of `stock`.** Counting a shelf is a monthly job for one
	// person; unpacking a delivery is a daily job for whoever is nearest the
	// door, and folding them together means either the storekeeper unpacks every
	// box or everybody who unpacks can reset the baseline a shortfall is
	// measured from.
	PermMarking = "marking"
	// PermLabel: printing the shop's own barcode stickers and shelf tags.
	//
	// ⚠️ **The permission is about the price, not about the paper.** A printed
	// shelf tag is a promise to the guest standing in front of it, and a wrong
	// one is settled at the counter with a queue behind them. It also invents
	// the product's barcode the first time it runs (handlers/labels.go), which
	// is a fact about the catalogue that nothing afterwards can quietly undo.
	//
	// ⚠️ Deliberately separate from `marking`: one scans what the state issued,
	// the other prints what the shop invented, and a shop may well trust the
	// same person with neither, either or both.
	PermLabel = "label"
)

// AllPerms is every permission a role can carry, in the order the panel draws
// them: floor first, money after, kitchen last.
var AllPerms = []string{
	PermWaiter, PermCashier, PermVoid, PermDiscount, PermShift, PermKitchen,
	PermStock, PermBuy, PermBuyOrder, PermStockIssue,
	PermMarking, PermLabel,
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

	// The same, for the one-off grant of `buyorder` to the roles that run the
	// floor. ⚠️ Its own flag rather than a shared "migrated" one: two
	// migrations sharing a marker means the second never runs on an install the
	// first already visited.
	BuyOrderGranted bool `bson:"buyOrderGranted,omitempty" json:"-"`

	// The same again, for the one-off grant of `stockissue` to the shipped
	// «Omborchi» role. ⚠️ A third flag rather than a reused one, for the reason
	// the second one gives.
	StockIssueGranted bool `bson:"stockIssueGranted,omitempty" json:"-"`

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
			PermWaiter, PermCashier, PermVoid, PermDiscount, PermShift, PermKitchen,
			PermBuyOrder}},
		{"Menejer", "Менеджер", "Manager", []string{
			PermWaiter, PermCashier, PermVoid, PermDiscount, PermShift, PermKitchen,
			PermBuyOrder}},
		{"Zal administratori", "Администратор зала", "Floor supervisor", []string{
			PermWaiter, PermCashier, PermVoid, PermDiscount, PermShift}},
		// ⚠️ **A cashier writes lists but does not go to the market.** They stand
		// at the counter all evening and are the first to hear the kitchen say
		// something has run out — and the till is where they already are. Buying
		// is the other half, and it is deliberately somebody else's.
		{"Kassir", "Кассир", "Cashier", []string{
			PermWaiter, PermCashier, PermShift, PermBuyOrder}},
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
		// ⚠️ **The one seeded role that ships with `buy`**, and it is the only
		// one that should: this is the person who goes to the market, and the
		// permission writes prices that reprice every dish on the menu. A
		// manager who also does the buying is given it explicitly, in one tap —
		// which is a decision somebody made rather than one they inherited.
		{"Zakupshik", "Закупщик", "Buyer", []string{PermBuy}},
		// ⚠️ **Counts the shelves and writes the list, and does not do the
		// buying.** That split is the supervision: the person who says what is
		// needed is not the person who comes back with a receipt.
		// ⚠️ **The one seeded role that ships with `stockissue`.** A request
		// whose goods are already in the building is this person's morning: they
		// pick it off the shelf and send it to whoever asked. Counting and
		// writing the list were already theirs; handing out is the third half of
		// the same job, and the only role it belongs to by default.
		{"Omborchi", "Кладовщик", "Storekeeper", []string{PermStock, PermBuyOrder, PermStockIssue}},
		{"Yordamchi xodim", "Подсобный работник", "Kitchen porter", nil},
	}
}
