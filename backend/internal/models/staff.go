package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Staff (kitchen, waiters, cashiers — everyone who clocks in) ----
//
// A staff account is not a courier: it carries no orders and no map. What it
// owns is a shift — the minutes between clocking in and clocking out — and the
// schedule those minutes are measured against.
//
// Accounts are created by the admin, exactly like couriers: there is no
// self-signup, and the password is only ever stored hashed.

// StaffSchedule is one weekday of the roster the admin wrote down. The whole
// point of the feature lives here: without an expected start and end, "worked
// too much" and "worked too little" have nothing to be measured against.
type StaffSchedule struct {
	Day   int    `bson:"day" json:"day"` // 0=Sunday .. 6=Saturday
	Start string `bson:"start" json:"start"`
	End   string `bson:"end" json:"end"`
	// A rostered day off. Working on one is not "extra hours" against a
	// baseline of zero — it is reported separately, as an unscheduled day.
	IsOff bool `bson:"isOff" json:"isOff"`
}

// StaffPayMode decides what an hour of work is worth.
//
//	"hourly"  — HourlyRate per worked hour (pro-rated by the minute)
//	"shift"   — ShiftRate for a day with any attendance at all
//	"monthly" — MonthlyRate spread over the month's rostered working days
type StaffPayMode string

const (
	PayHourly  StaffPayMode = "hourly"
	PayShift   StaffPayMode = "shift"
	PayMonthly StaffPayMode = "monthly"
)

// StaffPayPeriod is how often the employee is settled up. It decides which
// window the payroll screen totals — nothing else.
type StaffPayPeriod string

const (
	PeriodDaily   StaffPayPeriod = "daily"
	PeriodTenDay  StaffPayPeriod = "10days"
	PeriodHalfMon StaffPayPeriod = "15days"
	PeriodMonthly StaffPayPeriod = "monthly"
)

type Staff struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	// Which branch this person works at. Attendance is measured against that
	// branch's address, so a staff account without one can never clock in.
	BranchID     primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	Name         string             `bson:"name" json:"name" validate:"required"`
	Phone        string             `bson:"phone" json:"phone"`
	Username     string             `bson:"username" json:"username" validate:"required"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	// Free text the admin writes: "oshpaz", "ofitsiant", "kassir". Deliberately
	// not an enum — every restaurant names its jobs differently.
	Position string `bson:"position" json:"position"`

	Schedule []StaffSchedule `bson:"schedule" json:"schedule"`

	PayMode     StaffPayMode   `bson:"payMode" json:"payMode"`
	HourlyRate  int            `bson:"hourlyRate" json:"hourlyRate"`
	ShiftRate   int            `bson:"shiftRate" json:"shiftRate"`
	MonthlyRate int            `bson:"monthlyRate" json:"monthlyRate"`
	PayPeriod   StaffPayPeriod `bson:"payPeriod" json:"payPeriod"`

	// May open the kitchen screen (/staff/kitchen).
	//
	// ⚠️ **Granted per person, not implied by having a staff account.** Until
	// this existed any staff token could read the KDS, which means every
	// waiter, cashier and cleaner could see every ticket in the branch and,
	// worse, mark them cooked — a button that makes an order vanish from the
	// pass and tells the panel the kitchen is done. A shared tablet is the
	// whole point of the staff role, so "has a login" cannot be the same
	// question as "runs the kitchen screen".
	//
	// ⚠️ The zero value is **false**, which is the opposite of this codebase's
	// usual rule, and the reason is that the usual rule would defeat the
	// feature: a permission everybody has by default is not a permission.
	// Existing staff are grandfathered by EnsureKitchenAccess instead, so no
	// live kitchen loses its screen mid-service on the deploy that adds this.
	CanKitchen bool `bson:"canKitchen" json:"canKitchen"`

	// May run the floor screen: open checks, add dishes, fire them to the
	// kitchen, hand the check over to be paid.
	CanWaiter bool `bson:"canWaiter" json:"canWaiter"`
	// May run the till: everything a waiter may do, plus taking payment,
	// voiding food the kitchen has already cooked, and giving discounts.
	//
	// ⚠️ **Implies waiter** — see Can(). A cashier who could take money but not
	// add a dish would send every correction back across the room, and the
	// restaurant's answer to that is one shared login for everybody, which is
	// the thing these two fields exist to prevent.
	CanCashier bool `bson:"canCashier" json:"canCashier"`

	// May write a shopping list, whatever their role says.
	//
	// ⚠️ **A grant beside the role rather than inside it, and it is the only
	// permission shaped this way.** Everything else on this screen is a job:
	// a cashier takes money, a cook runs the pass, and the answer is the same
	// for every person holding that job. Who notices the sugar has run out is
	// not — it is the barman on Tuesdays and the porter on Fridays, and a
	// restaurant that had to invent "Barmen who may write lists" as a second
	// role would end up with a role per person, which is the thing roles exist
	// to prevent.
	//
	// ⚠️ **It only ever adds.** A role that grants `buyorder` is not taken away
	// by leaving this unticked — see Can(). One switch that sometimes grants and
	// sometimes revokes is a switch whose meaning has to be read out of a second
	// document, and the reading gets done wrong on the day somebody is in a
	// hurry.
	//
	// ⚠️ Zero value is false, and nothing is grandfathered: the roles that
	// already write lists keep writing them, so no restaurant loses anything on
	// the deploy that adds this.
	CanBuyOrder bool `bson:"canBuyOrder,omitempty" json:"canBuyOrder"`

	// Which role this person holds. ⚠️ **The role is where permissions live
	// now**; the three booleans above are kept only so tills and kitchens
	// installed before roles existed keep working, and so the migration has
	// something to read. New code asks Can(), which prefers the role.
	RoleID   primitive.ObjectID `bson:"roleId,omitempty" json:"roleId,omitempty"`
	RoleName string             `bson:"-" json:"roleName,omitempty"`
	// The role's name in the other two languages, carried alongside the base
	// one so the till can print the word the person standing at it reads.
	// Computed like RoleName, and empty when the role has no translation —
	// the screen falls back to the base name, exactly as a dish does.
	RoleNameRu string `bson:"-" json:"roleNameRu,omitempty"`
	RoleNameEn string `bson:"-" json:"roleNameEn,omitempty"`
	// Resolved permissions, filled on the way out for the panel and the till.
	// Never stored: a second copy of what the role says is a second thing that
	// can disagree with it.
	Perms []string `bson:"-" json:"perms,omitempty"`
	// Whether a role was actually found and applied.
	//
	// ⚠️ **A role that grants nothing was indistinguishable from no role at
	// all, and that was a hole somebody walked through.** `Can` fell back to
	// the pre-role booleans whenever `Perms` was empty — and seeded roles do
	// grant an empty list on purpose: Xostes and Yordamchi xodim (Texnolog was
	// the third until it was given `stock`). So a technologist assigned the
	// role that granted nothing kept whatever
	// `CanCashier` had been left at, opened the till, sent food to the kitchen
	// and cancelled a check, and was never asked for anybody's code.
	//
	// The flag exists rather than switching on `RoleID` because a *missing*
	// role document has to keep the old behaviour: deleting a role must not
	// lock a shift out of the till mid-service. Three states, and only one of
	// them is the legacy path.
	RoleApplied bool `bson:"-" json:"-"`

	// The code this person taps to take over the till screen.
	//
	// ⚠️ **A PIN is not a password and must never be treated as one.** Four
	// digits are guessable; what makes this safe is that it is only ever
	// accepted from a device that already holds a branch token — the monoblock
	// standing in the restaurant. Possession of the till plus the PIN is two
	// facts; either alone is nothing.
	//
	// ⚠️ **Its job is attribution, not access.** Without it a shared till holds
	// one login all evening, so every void and every discount is recorded
	// against whoever unlocked the screen at six — which is exactly the
	// question those records exist to answer. See handlers/tillpin.go.
	//
	// Hashed like a password (the cost is paid once per unlock, not per tap)
	// and never returned; the panel sees only whether one is set.
	PinHash string `bson:"pinHash,omitempty" json:"-"`
	// Whether a PIN is set, for the panel. ⚠️ Computed on the way out and never
	// stored (`bson:"-"`): the panel must be able to show "PIN o'rnatilgan"
	// without the hash ever leaving the server, and a second stored copy of the
	// same fact is a second thing that can disagree with it.
	PinSet bool `bson:"-" json:"hasPin"`

	IsActive  bool      `bson:"isActive" json:"isActive"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// Can reports whether this employee holds a till permission.
//
// ⚠️ **Cashier implies waiter**, and the implication lives here rather than in
// the two places that check it: a rule duplicated across a floor screen and a
// till screen is a rule that will one day disagree with itself, and the
// disagreement shows up as "the button works on his tablet but not on mine".
//
// ⚠️ **IsActive is part of the answer, not a separate check.** A staff token
// outlives a shift by days, so an employee dismissed this morning still holds a
// working one tonight. That gap already existed once — the kitchen screen never
// asked — so the question is answered once, here, and cannot be forgotten by
// the next screen that needs it.
func (s *Staff) Can(perm string) bool {
	if s == nil || !s.IsActive {
		return false
	}
	// ⚠️ **Asked before the role, and it only ever says yes.** Writing a
	// shopping list is granted per person as well as per job — see CanBuyOrder
	// for why — and a grant that the role could then withdraw would make the
	// tick on the employee's card mean nothing on half the cards it appears on.
	if perm == PermBuyOrder && s.CanBuyOrder {
		return true
	}
	// ⚠️ **The role wins when there is one, including when it grants nothing.**
	// Perms is filled from the role on the way in (see withRole); the booleans
	// below are the pre-role world and answer only for accounts no role has
	// ever been applied to.
	if s.RoleApplied {
		return grants(s.Perms, perm)
	}
	if len(s.Perms) > 0 {
		return grants(s.Perms, perm)
	}
	switch perm {
	case PermCashier:
		return s.CanCashier
	case PermWaiter:
		return s.CanWaiter || s.CanCashier
	case PermKitchen:
		return s.CanKitchen
	}
	// ⚠️ An unknown permission on an unmigrated account is **refused**, not
	// granted. `void` and `discount` did not exist before roles, so there is no
	// old flag that means yes — and guessing yes would hand every legacy
	// waiter the ability to write off cooked food.
	return false
}

// grants is the list itself, with the one implication the till relies on.
//
// ⚠️ Cashier implies waiter: somebody trusted with the drawer is trusted to
// carry a plate, and making a restaurant tick both would mean discovering the
// second one is missing at the counter on a Friday.
func grants(perms []string, perm string) bool {
	for _, p := range perms {
		if p == perm {
			return true
		}
		if perm == PermWaiter && p == PermCashier {
			return true
		}
	}
	return false
}

// HasPin reports whether this employee can unlock a till screen.
//
// Its own method so the panel and the till agree on the question, and so the
// hash itself never has to leave the model to answer it.
func (s *Staff) HasPin() bool { return s != nil && s.PinHash != "" }

// WithPinFlag fills the outgoing flag from the stored hash.
//
// ⚠️ Applied where staff rows are handed to the panel. A method rather than a
// field the handlers set by hand, so adding a third place that returns staff
// is one call rather than a silent "PIN o'rnatilmagan" on a person who has one.
func WithPinFlag(rows []Staff) []Staff {
	for i := range rows {
		rows[i].PinSet = rows[i].HasPin()
	}
	return rows
}

// ScheduleFor returns the roster line for a weekday, and whether one was set.
// A weekday the admin never filled in reads as a day off — an empty roster
// must not accuse everybody of being absent every day.
func (s *Staff) ScheduleFor(day int) (StaffSchedule, bool) {
	for _, row := range s.Schedule {
		if row.Day == day {
			return row, !row.IsOff && row.Start != "" && row.End != ""
		}
	}
	return StaffSchedule{Day: day, IsOff: true}, false
}

// ShiftPunch is where the employee stood when they pressed the button. Kept on
// the record rather than merely checked: "the app let me in from home" is an
// argument that only a stored coordinate can settle.
type ShiftPunch struct {
	Lat      float64   `bson:"lat" json:"lat"`
	Lng      float64   `bson:"lng" json:"lng"`
	Accuracy float64   `bson:"accuracy" json:"accuracy"`
	Meters   float64   `bson:"meters" json:"meters"` // distance from the branch
	At       time.Time `bson:"at" json:"at"`
}

// Shift is one clock-in / clock-out pair.
//
// Date is the *working day* the shift is filed under, as a local YYYY-MM-DD
// string rather than a timestamp: a cook who clocks out at 00:40 finished
// Tuesday's shift, not Wednesday's, and every calendar in the product has to
// agree on that.
type Shift struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StaffID  primitive.ObjectID `bson:"staffId" json:"staffId"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	Date     string             `bson:"date" json:"date"`

	In  time.Time  `bson:"in" json:"in"`
	Out *time.Time `bson:"out,omitempty" json:"out,omitempty"`

	InAt  *ShiftPunch `bson:"inAt,omitempty" json:"inAt,omitempty"`
	OutAt *ShiftPunch `bson:"outAt,omitempty" json:"outAt,omitempty"`

	// Hash of the kiosk code each punch was made with, so one code can open or
	// close exactly one thing.
	//
	// Without this a second scan of the same code — a double tap on the camera
	// notification, a reload, "nothing happened so I scanned again" — would
	// clock the employee straight back out again, because the server decides
	// the direction from whether a shift is open. Hashed, not stored plainly:
	// there is no reason to keep a live code readable in the database.
	InCode  string `bson:"inCode,omitempty" json:"-"`
	OutCode string `bson:"outCode,omitempty" json:"-"`

	// Worked minutes, written on clock-out. Stored rather than recomputed on
	// read so a later edit to the schedule cannot rewrite what someone did.
	Minutes int `bson:"minutes" json:"minutes"`

	// Set when an admin opened or closed the shift by hand (a forgotten
	// clock-out, a dead phone). The employee's own punches are never edited
	// silently — the record says who wrote it.
	EditedBy string `bson:"editedBy,omitempty" json:"editedBy,omitempty"`
	Note     string `bson:"note,omitempty" json:"note,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// StaffPayment is money actually handed to the employee.
//
// A ledger entry, not a counter reset — the same reason courier settlements
// are: "how much did we pay Aziz on the 10th?" has to stay answerable after
// the next period has started.
type StaffPayment struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StaffID  primitive.ObjectID `bson:"staffId" json:"staffId"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	Amount   int                `bson:"amount" json:"amount"`
	// The window this payment settles, as local YYYY-MM-DD. Filled from the
	// employee's pay period, so a payment always says what it was for.
	From string `bson:"from" json:"from"`
	To   string `bson:"to" json:"to"`
	// Who handed it over. An unsigned cash payment is the record that gets
	// disputed later.
	PaidBy string    `bson:"paidBy" json:"paidBy"`
	Note   string    `bson:"note,omitempty" json:"note,omitempty"`
	At     time.Time `bson:"at" json:"at"`
}
