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

	IsActive  bool      `bson:"isActive" json:"isActive"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
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
