package handlers

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Attendance: turning punches into a day, a week and a wage.
//
// Everything the employee, the admin dashboard and the payroll screen show is
// derived here, from two inputs only: the roster the admin wrote down and the
// shifts the employee actually clocked. Nothing about a day is stored twice —
// "worked too much" is a comparison, not a flag someone has to remember to set.
//
// Dates are local YYYY-MM-DD strings throughout. A shift is filed under the
// day it *started*, so a cook who clocks out at 00:40 closes Tuesday rather
// than opening Wednesday. Set TZ (Asia/Tashkent) on the server, or every
// calendar in the product is drawn in UTC.

const dayLayout = "2006-01-02"

// dayKey is the working day a moment belongs to.
func dayKey(t time.Time) string { return t.Local().Format(dayLayout) }

// parseDay reads a YYYY-MM-DD string as local midnight.
func parseDay(s string) (time.Time, error) {
	return time.ParseInLocation(dayLayout, strings.TrimSpace(s), time.Local)
}

// startOfDay is local midnight of t.
func startOfDay(t time.Time) time.Time {
	t = t.Local()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// clockMinutes parses "HH:MM" into minutes since midnight; -1 when unreadable.
func clockMinutes(s string) int {
	parts := strings.SplitN(strings.TrimSpace(s), ":", 2)
	if len(parts) != 2 {
		return -1
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 24 || m < 0 || m > 59 {
		return -1
	}
	return h*60 + m
}

// scheduledMinutes is how long a rostered day is meant to last. An end time
// before the start means the shift crosses midnight (18:00 → 02:00), which is
// ordinary in a restaurant and would otherwise come out negative.
func scheduledMinutes(sch models.StaffSchedule) int {
	start, end := clockMinutes(sch.Start), clockMinutes(sch.End)
	if start < 0 || end < 0 {
		return 0
	}
	if end <= start {
		end += 24 * 60
	}
	return end - start
}

// Day statuses. They are the whole point of the roster: without an expected
// length, every day looks the same.
const (
	DayOff      = "off"      // rostered day off, and nobody came in
	DayAbsent   = "absent"   // rostered, nobody came in
	DayUnder    = "under"    // came in, worked less than rostered
	DayOK       = "ok"       // worked what was asked, give or take the tolerance
	DayOver     = "over"     // worked longer than rostered
	DayExtra    = "extra"    // worked on a rostered day off
	DayOpen     = "open"     // clocked in, not yet out
	DayUpcoming = "upcoming" // rostered, but the day has not happened yet
)

// dayTolerance is how far either side of the roster still counts as "on time".
// Nobody clocks in at exactly 11:00, and flagging a nine-minute overrun as
// overtime would make the whole column meaningless.
const dayTolerance = 15

// StaffSession is one clock-in / clock-out pair as the UI shows it.
type StaffSession struct {
	ID  string     `json:"id"`
	In  time.Time  `json:"in"`
	Out *time.Time `json:"out,omitempty"`
	// Worked minutes. An open session counts the time up to now, so the
	// employee's own screen ticks along with the shift.
	Minutes  int    `json:"minutes"`
	Open     bool   `json:"open"`
	EditedBy string `json:"editedBy,omitempty"`
	Note     string `json:"note,omitempty"`
}

// StaffDay is one square of the calendar.
type StaffDay struct {
	Date    string `json:"date"`
	Weekday int    `json:"weekday"`
	// Minutes the roster asks for; 0 on a day off.
	Expected int `json:"expected"`
	// Minutes actually clocked.
	Worked int `json:"worked"`
	// Difference, signed: positive is overtime.
	Diff     int            `json:"diff"`
	Sessions []StaffSession `json:"sessions"`
	// First clock-in and last clock-out of the day, "HH:MM" or empty. This is
	// the answer to "what time exactly did they arrive today".
	First string `json:"first"`
	Last  string `json:"last"`
	// Rostered window, for the UI to show alongside the real one.
	PlanStart string `json:"planStart"`
	PlanEnd   string `json:"planEnd"`
	Open      bool   `json:"open"`
	Status    string `json:"status"`
	// What the day earned under the employee's pay rule.
	Pay int `json:"pay"`
}

// StaffTotals sums a range of days.
type StaffTotals struct {
	Days     int `json:"days"`     // days with any attendance
	Absent   int `json:"absent"`   // rostered days nobody came in
	Expected int `json:"expected"` // rostered minutes
	Worked   int `json:"worked"`   // clocked minutes
	Diff     int `json:"diff"`
	Overtime int `json:"overtime"` // minutes above the roster, summed
	Shortage int `json:"shortage"` // minutes below the roster, summed
	Pay      int `json:"pay"`
}

// StaffTrend compares a window with the one before it. Percent is of the
// previous window; when that was zero there is nothing to compare against and
// HasPrev is false — "+100%" against nothing is a lie the dashboard should not
// tell.
type StaffTrend struct {
	Current  int     `json:"current"`
	Previous int     `json:"previous"`
	Percent  float64 `json:"percent"`
	HasPrev  bool    `json:"hasPrev"`
}

func trendOf(current, previous int) StaffTrend {
	t := StaffTrend{Current: current, Previous: previous, HasPrev: previous > 0}
	if previous > 0 {
		t.Percent = math.Round(float64(current-previous)/float64(previous)*1000) / 10
	}
	return t
}

// ---- Loading ----

// shiftsInRange loads one employee's shifts for an inclusive date range.
func (h *Handler) shiftsInRange(ctx context.Context, staffID primitive.ObjectID, from, to string) ([]models.Shift, error) {
	opts := options.Find().SetSort(bson.D{{Key: "in", Value: 1}})
	cur, err := h.Store.Shifts.Find(ctx, bson.M{
		"staffId": staffID,
		"date":    bson.M{"$gte": from, "$lte": to},
	}, opts)
	if err != nil {
		return nil, err
	}
	out := []models.Shift{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// openShift returns the employee's shift that has been clocked in but not out,
// if any. There can only ever be one — clocking in twice is refused.
func (h *Handler) openShift(ctx context.Context, staffID primitive.ObjectID) (*models.Shift, error) {
	var s models.Shift
	err := h.Store.Shifts.FindOne(ctx,
		bson.M{"staffId": staffID, "out": bson.M{"$exists": false}},
		options.FindOne().SetSort(bson.D{{Key: "in", Value: -1}}),
	).Decode(&s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ---- Building days ----

// buildDays turns a roster plus a set of shifts into one entry per calendar day
// between from and to (inclusive). Days with no shift are not skipped: an
// absence is exactly the thing the admin opened this screen to find.
func buildDays(staff *models.Staff, shifts []models.Shift, from, to time.Time) []StaffDay {
	now := time.Now()
	byDate := map[string][]models.Shift{}
	for _, s := range shifts {
		byDate[s.Date] = append(byDate[s.Date], s)
	}

	// Monthly pay is spread over the month's rostered days, so each month in
	// the range needs its own divisor. Cached: the loop below asks per day.
	monthDays := map[string]int{}
	rosteredDaysInMonth := func(t time.Time) int {
		key := t.Format("2006-01")
		if n, ok := monthDays[key]; ok {
			return n
		}
		n := 0
		first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
		for d := first; d.Month() == t.Month(); d = d.AddDate(0, 0, 1) {
			if _, works := staff.ScheduleFor(int(d.Weekday())); works {
				n++
			}
		}
		monthDays[key] = n
		return n
	}

	days := []StaffDay{}
	for d := startOfDay(from); !d.After(startOfDay(to)); d = d.AddDate(0, 0, 1) {
		key := d.Format(dayLayout)
		sch, works := staff.ScheduleFor(int(d.Weekday()))

		day := StaffDay{
			Date:     key,
			Weekday:  int(d.Weekday()),
			Sessions: []StaffSession{},
		}
		if works {
			day.Expected = scheduledMinutes(sch)
			day.PlanStart, day.PlanEnd = sch.Start, sch.End
		}

		for _, s := range byDate[key] {
			sess := StaffSession{
				ID:       s.ID.Hex(),
				In:       s.In,
				Out:      s.Out,
				EditedBy: s.EditedBy,
				Note:     s.Note,
			}
			if s.Out == nil {
				sess.Open = true
				sess.Minutes = int(now.Sub(s.In).Minutes())
				if sess.Minutes < 0 {
					sess.Minutes = 0
				}
				day.Open = true
			} else {
				sess.Minutes = s.Minutes
			}
			day.Worked += sess.Minutes
			day.Sessions = append(day.Sessions, sess)
		}

		if len(day.Sessions) > 0 {
			day.First = day.Sessions[0].In.Local().Format("15:04")
			last := day.Sessions[len(day.Sessions)-1]
			if last.Out != nil {
				day.Last = last.Out.Local().Format("15:04")
			}
		}
		day.Diff = day.Worked - day.Expected
		day.Status = dayStatus(day, works, d, now)
		day.Pay = payForDay(staff, day, rosteredDaysInMonth(d))
		days = append(days, day)
	}
	return days
}

// dayStatus names what happened, in the order the reader cares about: is it
// still running, did anyone come in, has the day even happened yet, and only
// then how long they stayed.
//
// "Did anyone come in" is asked of the punches, not of the minutes: somebody
// who clocked in and straight back out worked zero minutes but is not absent,
// and calling that day "upcoming" would hide a real problem.
func dayStatus(day StaffDay, works bool, d, now time.Time) string {
	came := len(day.Sessions) > 0
	switch {
	case day.Open:
		return DayOpen
	case came && !works:
		return DayExtra
	case came && day.Diff > dayTolerance:
		return DayOver
	case came && day.Diff < -dayTolerance:
		return DayUnder
	case came:
		return DayOK
	case !works:
		return DayOff
	// A rostered day that has not arrived yet is not an absence. Today counts
	// as upcoming until it is over — somebody on the late shift has not failed
	// to turn up at nine in the morning.
	case d.AddDate(0, 0, 1).After(now):
		return DayUpcoming
	default:
		return DayAbsent
	}
}

// payForDay values one day under the employee's pay rule.
//
// Monthly salary is divided by the month's rostered days and paid per day
// attended: it is the only reading that survives someone joining mid-month or
// missing a week, and it makes the payroll screen agree with the calendar.
func payForDay(staff *models.Staff, day StaffDay, rosteredThisMonth int) int {
	switch staff.PayMode {
	case models.PayShift:
		if day.Worked > 0 {
			return staff.ShiftRate
		}
		return 0
	case models.PayMonthly:
		if day.Worked == 0 || rosteredThisMonth == 0 {
			return 0
		}
		return staff.MonthlyRate / rosteredThisMonth
	default: // PayHourly, and empty on documents written before this field
		return staff.HourlyRate * day.Worked / 60
	}
}

// sumDays totals a range for the summary strip above the calendar.
func sumDays(days []StaffDay) StaffTotals {
	var t StaffTotals
	for _, d := range days {
		came := len(d.Sessions) > 0
		t.Expected += d.Expected
		t.Worked += d.Worked
		t.Pay += d.Pay
		if came {
			t.Days++
		}
		if d.Status == DayAbsent {
			t.Absent++
		}
		// Only days somebody turned up for carry a shortage: an absence is
		// already counted above, and adding it here too would drown the figure
		// that means "came in but left early".
		if came {
			if d.Diff > 0 {
				t.Overtime += d.Diff
			} else {
				t.Shortage += -d.Diff
			}
		}
	}
	t.Diff = t.Worked - t.Expected
	return t
}

// workedBetween sums worked minutes over a sub-range of an already-built list.
// Cheaper than re-querying, and it keeps every trend on the same numbers as the
// calendar the reader is looking at.
func workedBetween(days []StaffDay, from, to string) int {
	total := 0
	for _, d := range days {
		if d.Date >= from && d.Date <= to {
			total += d.Worked
		}
	}
	return total
}

// ---- Pay periods ----

// payPeriodBounds is the window the employee is currently owed for, given how
// often they are settled up.
//
// The ten- and fifteen-day periods are anchored to the calendar month (1–10,
// 11–20, 21–end) rather than rolling from the hire date: that is how the pay
// day is actually spoken about, and it keeps every employee's periods lined up.
func payPeriodBounds(period models.StaffPayPeriod, now time.Time) (time.Time, time.Time) {
	now = startOfDay(now)
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastOfMonth := first.AddDate(0, 1, -1)
	day := now.Day()

	switch period {
	case models.PeriodDaily:
		return now, now
	case models.PeriodTenDay:
		switch {
		case day <= 10:
			return first, first.AddDate(0, 0, 9)
		case day <= 20:
			return first.AddDate(0, 0, 10), first.AddDate(0, 0, 19)
		default:
			return first.AddDate(0, 0, 20), lastOfMonth
		}
	case models.PeriodHalfMon:
		if day <= 15 {
			return first, first.AddDate(0, 0, 14)
		}
		return first.AddDate(0, 0, 15), lastOfMonth
	default: // PeriodMonthly, and empty on older documents
		return first, lastOfMonth
	}
}

// paidBetween totals what has actually been handed to an employee inside a
// window, matched on the period the payment says it settles rather than on the
// day it was handed over — money paid on the 3rd for last month belongs to last
// month.
func (h *Handler) paidBetween(ctx context.Context, staffID primitive.ObjectID, from, to string) (int, error) {
	cur, err := h.Store.StaffPayments.Find(ctx, bson.M{
		"staffId": staffID,
		"from":    bson.M{"$gte": from},
		"to":      bson.M{"$lte": to},
	})
	if err != nil {
		return 0, err
	}
	var rows []models.StaffPayment
	if err := cur.All(ctx, &rows); err != nil {
		return 0, err
	}
	total := 0
	for _, p := range rows {
		total += p.Amount
	}
	return total, nil
}

// ---- Range parsing ----

// dateRange reads ?from=&to= (YYYY-MM-DD, `to` inclusive). Both empty means the
// current calendar month, which is what every one of these screens opens on.
func dateRange(q map[string][]string) (time.Time, time.Time, error) {
	get := func(k string) string {
		if v, ok := q[k]; ok && len(v) > 0 {
			return strings.TrimSpace(v[0])
		}
		return ""
	}
	now := time.Now()
	rawFrom, rawTo := get("from"), get("to")
	if rawFrom == "" && rawTo == "" {
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return first, first.AddDate(0, 1, -1), nil
	}
	from, to := startOfDay(now), startOfDay(now)
	if rawFrom != "" {
		v, err := parseDay(rawFrom)
		if err != nil {
			return from, to, errors.New("invalid from")
		}
		from = v
	}
	if rawTo != "" {
		v, err := parseDay(rawTo)
		if err != nil {
			return from, to, errors.New("invalid to")
		}
		to = v
	}
	if to.Before(from) {
		from, to = to, from
	}
	// A range nobody meant: guard the loop that walks it day by day.
	if to.Sub(from) > 400*24*time.Hour {
		to = from.AddDate(0, 0, 400)
	}
	return from, to, nil
}

// Errors the staff screens hand straight back to the operator.
var (
	errInvalidBranch = errors.New("filial tanlanmagan yoki topilmadi")
	errInvalidDate   = errors.New("sana noto'g'ri")
	errInvalidTime   = errors.New("vaqt noto'g'ri (HH:MM)")

	errKioskToken   = errors.New("kiosk tokeni yaroqsiz")
	errKioskRevoked = errors.New("kiosk tokeni bekor qilingan — paneldan yangisini oling")
)

// adminName is who is acting, for the records that have to be signed: a
// hand-written shift, a salary payment.
func (h *Handler) adminName(r *http.Request) string {
	if admin, err := h.adminUser(r); err == nil {
		if admin.Name != "" {
			return admin.Name
		}
		return admin.Username
	}
	return ""
}
