package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// The admin side of staff attendance: the accounts, the board that says who is
// in and who is short, and the money owed.

// staffPayload is what the admin form sends. Password is optional on update
// (empty = keep the current one), same rule as couriers.
type staffPayload struct {
	Name     string `json:"name" validate:"required"`
	Phone    string `json:"phone"`
	Username string `json:"username" validate:"required"`
	Password string `json:"password"`
	// ⚠️ No `position`: the title is the role's name and the server writes it
	// (EnsurePositionFromRole). An old panel that still sends one is ignored.
	BranchID string `json:"branchId"`
	IsActive *bool  `json:"isActive"`
	// May open the kitchen screen. A pointer so an older client that does not
	// send the field leaves the stored value alone rather than revoking it —
	// the same rule as isActive, and here it would silently lock a cook out
	// mid-service.
	CanKitchen *bool `json:"canKitchen"`
	// ⚠️ **A permission on the person rather than on the job**, and the only one
	// shaped that way — see models/staff.go. Who notices the sugar has run out
	// is the barman on Tuesdays and the porter on Fridays, and inventing "Barmen
	// who may write lists" as a second role ends in a role per person.
	CanBuyOrder *bool `json:"canBuyOrder"`
	// Till permissions, pointers for the same reason: a form that does not show
	// a field must never be able to revoke it. See models.Staff — cashier
	// implies waiter, so the two boxes are not independent on screen either.
	CanWaiter  *bool `json:"canWaiter"`
	CanCashier *bool `json:"canCashier"`

	// Which role this person holds. ⚠️ **A pointer to a string, so an older
	// client that does not send it leaves the stored role alone.** Sending it
	// empty clears the role, which drops the person back to the legacy flags —
	// so the panel only ever sends a real id.
	RoleID *string `json:"roleId"`

	Schedule []models.StaffSchedule `json:"schedule"`

	PayMode     models.StaffPayMode   `json:"payMode"`
	HourlyRate  int                   `json:"hourlyRate"`
	ShiftRate   int                   `json:"shiftRate"`
	MonthlyRate int                   `json:"monthlyRate"`
	PayPeriod   models.StaffPayPeriod `json:"payPeriod"`
}

// normalizeSchedule keeps one row per weekday and drops anything unreadable, so
// a malformed roster can never make a day's expected length negative.
func normalizeSchedule(rows []models.StaffSchedule) []models.StaffSchedule {
	seen := map[int]bool{}
	out := []models.StaffSchedule{}
	for _, row := range rows {
		if row.Day < 0 || row.Day > 6 || seen[row.Day] {
			continue
		}
		seen[row.Day] = true
		if clockMinutes(row.Start) < 0 || clockMinutes(row.End) < 0 {
			row.Start, row.End, row.IsOff = "", "", true
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Day < out[j].Day })
	return out
}

// payRuleOf normalises the pay fields coming from the form.
func payRuleOf(req staffPayload) (models.StaffPayMode, models.StaffPayPeriod) {
	mode := req.PayMode
	switch mode {
	case models.PayShift, models.PayMonthly, models.PayHourly:
	default:
		mode = models.PayHourly
	}
	period := req.PayPeriod
	switch period {
	case models.PeriodDaily, models.PeriodTenDay, models.PeriodHalfMon, models.PeriodMonthly:
	default:
		period = models.PeriodMonthly
	}
	return mode, period
}

func nonNegative(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

// ---- Accounts ----

// StaffRow is one line of the admin list: the account plus how the chosen
// period actually went. The numbers are the reason the screen exists — a list
// of names alone is a phonebook.
type StaffRow struct {
	models.Staff
	BranchName string      `json:"branchName"`
	Totals     StaffTotals `json:"totals"`
	// Clocked in right now.
	OnShift bool `json:"onShift"`
	// Today's first clock-in and last clock-out, "HH:MM" or empty.
	TodayIn  string `json:"todayIn"`
	TodayOut string `json:"todayOut"`
	// Worked minutes today against what today was rostered for.
	TodayWorked   int    `json:"todayWorked"`
	TodayExpected int    `json:"todayExpected"`
	TodayStatus   string `json:"todayStatus"`
	// The current pay period, so "who do I owe" is answerable from the list.
	PeriodFrom string `json:"periodFrom"`
	PeriodTo   string `json:"periodTo"`
	PeriodPay  int    `json:"periodPay"`
	PeriodPaid int    `json:"periodPaid"`
	PeriodDue  int    `json:"periodDue"`
}

// staffScope narrows staff queries to the branch the panel is looking at, with
// the same clamp every other admin list uses.
func (h *Handler) staffScope(r *http.Request) (bson.M, Scope, error) {
	return h.orderScope(r)
}

// AdminListStaff returns the branch's employees with their attendance for the
// requested range (default: this calendar month).
func (h *Handler) AdminListStaff(w http.ResponseWriter, r *http.Request) {
	filter, _, err := h.staffScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	from, to, err := dateRange(r.URL.Query())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		rx := textSearch(q)
		filter["$or"] = []bson.M{
			{"name": rx}, {"phone": rx}, {"username": rx}, {"position": rx},
		}
	}

	cur, err := h.Store.Staff.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	staff := []models.Staff{}
	if err := cur.All(r.Context(), &staff); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	models.WithPinFlag(staff)
	// One query for every role in the list rather than one per person — the
	// same reason shiftsByStaff exists.
	h.withRoles(r.Context(), staff)

	// One pass for every shift in the range instead of a query per person: a
	// branch with twenty employees would otherwise open this screen with
	// forty round trips.
	byStaff, err := h.shiftsByStaff(r, staff, from.Format(dayLayout), to.Format(dayLayout))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	branchNames := h.branchNames(r)

	now := time.Now()
	todayKey := dayKey(now)
	rows := make([]StaffRow, 0, len(staff))
	for i := range staff {
		s := &staff[i]
		days := buildDays(s, byStaff[s.ID], from, to)
		row := StaffRow{
			Staff:      *s,
			BranchName: branchNames[s.BranchID],
			Totals:     sumDays(days),
		}

		// Today is read on its own, because the range being browsed may not
		// contain it — and "is Aziz in right now" is the first thing anybody
		// looks at.
		todayShifts, err := h.shiftsInRange(r.Context(), s.ID, todayKey, todayKey)
		if err == nil {
			today := buildDays(s, todayShifts, now, now)
			if len(today) == 1 {
				row.TodayIn, row.TodayOut = today[0].First, today[0].Last
				row.TodayWorked, row.TodayExpected = today[0].Worked, today[0].Expected
				row.TodayStatus, row.OnShift = today[0].Status, today[0].Open
			}
		}

		pFrom, pTo := payPeriodBounds(s.PayPeriod, now)
		row.PeriodFrom, row.PeriodTo = pFrom.Format(dayLayout), pTo.Format(dayLayout)
		if periodShifts, err := h.shiftsInRange(r.Context(), s.ID, row.PeriodFrom, row.PeriodTo); err == nil {
			row.PeriodPay = sumDays(buildDays(s, periodShifts, pFrom, pTo)).Pay
		}
		row.PeriodPaid, _ = h.paidBetween(r.Context(), s.ID, row.PeriodFrom, row.PeriodTo)
		row.PeriodDue = row.PeriodPay - row.PeriodPaid
		rows = append(rows, row)
	}
	httpx.JSON(w, http.StatusOK, rows)
}

// shiftsByStaff loads a date range's shifts for a set of employees in one go.
func (h *Handler) shiftsByStaff(r *http.Request, staff []models.Staff, from, to string) (map[primitive.ObjectID][]models.Shift, error) {
	out := map[primitive.ObjectID][]models.Shift{}
	if len(staff) == 0 {
		return out, nil
	}
	ids := make([]primitive.ObjectID, 0, len(staff))
	for i := range staff {
		ids = append(ids, staff[i].ID)
	}
	cur, err := h.Store.Shifts.Find(r.Context(), bson.M{
		"staffId": bson.M{"$in": ids},
		"date":    bson.M{"$gte": from, "$lte": to},
	}, options.Find().SetSort(bson.D{{Key: "in", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var rows []models.Shift
	if err := cur.All(r.Context(), &rows); err != nil {
		return nil, err
	}
	for _, s := range rows {
		out[s.StaffID] = append(out[s.StaffID], s)
	}
	return out, nil
}

// branchNames maps branch ids to names, for the rows an owner sees across a
// whole brand.
func (h *Handler) branchNames(r *http.Request) map[primitive.ObjectID]string {
	names := map[primitive.ObjectID]string{}
	cur, err := h.Store.Branches.Find(r.Context(), bson.M{})
	if err != nil {
		return names
	}
	var rows []models.Branch
	if err := cur.All(r.Context(), &rows); err != nil {
		return names
	}
	for _, b := range rows {
		names[b.ID] = b.Name
	}
	return names
}

// AdminCreateStaff adds an employee account by hand.
func (h *Handler) AdminCreateStaff(w http.ResponseWriter, r *http.Request) {
	var req staffPayload
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	username := normalizeUsername(req.Username)
	if len(req.Password) < 5 {
		httpx.Error(w, http.StatusBadRequest, "parol kamida 5 belgi bo'lishi kerak")
		return
	}
	if taken, err := h.usernameTaken(r, username, primitive.NilObjectID); err != nil || taken {
		httpx.Error(w, http.StatusConflict, "bu login band")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	branchID, err := h.staffBranch(r, req.BranchID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	newStaffRole := primitiveNil
	if req.RoleID != nil {
		newStaffRole, err = h.roleRef(r, *req.RoleID)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	mode, period := payRuleOf(req)
	now := time.Now()
	s := models.Staff{
		BranchID:     branchID,
		Name:         strings.TrimSpace(req.Name),
		Phone:        strings.TrimSpace(req.Phone),
		Username:     username,
		PasswordHash: string(hash),
		// ⚠️ The role's name, never typed text — see EnsurePositionFromRole.
		Position:    h.roleTitle(r.Context(), newStaffRole),
		Schedule:    normalizeSchedule(req.Schedule),
		PayMode:     mode,
		PayPeriod:   period,
		HourlyRate:  nonNegative(req.HourlyRate),
		ShiftRate:   nonNegative(req.ShiftRate),
		MonthlyRate: nonNegative(req.MonthlyRate),
		// ⚠️ New staff start **without** it, deliberately: a permission
		// everybody gets on creation is not a permission. Existing staff were
		// grandfathered once by EnsureKitchenAccess so no live pass went dark;
		// from here it is a decision somebody makes per person.
		CanKitchen:  req.CanKitchen != nil && *req.CanKitchen,
		CanBuyOrder: req.CanBuyOrder != nil && *req.CanBuyOrder,
		// Same rule, and here it never needed grandfathering: nobody could work
		// a till before the feature existed, so there is no behaviour to
		// preserve and false is simply the truth.
		CanWaiter:  req.CanWaiter != nil && *req.CanWaiter,
		CanCashier: req.CanCashier != nil && *req.CanCashier,
		RoleID:     newStaffRole,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	res, err := h.Store.Staff.InsertOne(r.Context(), s)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.ID = oidOf(res.InsertedID)
	h.logAction(r, ActStaffCreate, "staff", s.ID.Hex(), s.Name, s.Position)
	httpx.JSON(w, http.StatusCreated, s)
}

// staffBranch resolves which branch a new or edited employee belongs to. An
// employee without one could never clock in — attendance is measured against
// the branch address — so this refuses rather than defaulting to nothing.
func (h *Handler) staffBranch(r *http.Request, raw string) (primitive.ObjectID, error) {
	if id := strings.TrimSpace(raw); id != "" {
		oid, err := objectID(id)
		if err != nil {
			return primitive.NilObjectID, errInvalidBranch
		}
		if err := h.requireBranchAccess(r, oid); err != nil {
			return primitive.NilObjectID, err
		}
		return oid, nil
	}
	scope, err := h.adminScope(r)
	if err != nil {
		return primitive.NilObjectID, err
	}
	if !scope.BranchID.IsZero() {
		return scope.BranchID, nil
	}
	b, err := h.defaultBranch(r, scope.BrandID)
	if err != nil {
		return primitive.NilObjectID, errInvalidBranch
	}
	return b.ID, nil
}

// usernameTaken checks the staff logins, excluding one id on update.
func (h *Handler) usernameTaken(r *http.Request, username string, except primitive.ObjectID) (bool, error) {
	filter := bson.M{"username": username}
	if !except.IsZero() {
		filter["_id"] = bson.M{"$ne": except}
	}
	n, err := h.Store.Staff.CountDocuments(r.Context(), filter)
	return n > 0, err
}

// AdminUpdateStaff edits the account; an empty password keeps the old one.
func (h *Handler) AdminUpdateStaff(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req staffPayload
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	existing, err := h.staffByID(r, id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "ishchi topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, existing.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	username := normalizeUsername(req.Username)
	if taken, err := h.usernameTaken(r, username, id); err != nil || taken {
		httpx.Error(w, http.StatusConflict, "bu login band")
		return
	}
	branchID, err := h.staffBranch(r, req.BranchID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	mode, period := payRuleOf(req)
	set := bson.M{
		"name":        strings.TrimSpace(req.Name),
		"phone":       strings.TrimSpace(req.Phone),
		"username":    username,
		"branchId":    branchID,
		"schedule":    normalizeSchedule(req.Schedule),
		"payMode":     mode,
		"payPeriod":   period,
		"hourlyRate":  nonNegative(req.HourlyRate),
		"shiftRate":   nonNegative(req.ShiftRate),
		"monthlyRate": nonNegative(req.MonthlyRate),
		"updatedAt":   time.Now(),
	}
	if req.IsActive != nil {
		set["isActive"] = *req.IsActive
	}
	if req.CanKitchen != nil {
		set["canKitchen"] = *req.CanKitchen
	}
	if req.CanBuyOrder != nil {
		set["canBuyOrder"] = *req.CanBuyOrder
	}
	if req.CanWaiter != nil {
		set["canWaiter"] = *req.CanWaiter
	}
	if req.CanCashier != nil {
		set["canCashier"] = *req.CanCashier
	}
	// ⚠️ The role is where permissions live now. Validated rather than trusted:
	// an id that names no role would leave this person falling back to the
	// legacy flags, which for most people means losing the till mid-shift with
	// nothing on the tablet explaining why.
	if req.RoleID != nil {
		roleID, err := h.roleRef(r, *req.RoleID)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		set["roleId"] = roleID
		// ⚠️ The title follows the role and nothing else writes it — see
		// EnsurePositionFromRole. Only when a role was sent: an old till or
		// script that omits it must not blank the title it cannot see.
		set["position"] = h.roleTitle(r.Context(), roleID)
	}
	if req.Password != "" {
		if len(req.Password) < 5 {
			httpx.Error(w, http.StatusBadRequest, "parol kamida 5 belgi bo'lishi kerak")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		set["passwordHash"] = string(hash)
	}
	if _, err := h.Store.Staff.UpdateByID(r.Context(), id, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	updated, _ := h.staffByID(r, id)
	// ⚠️ **Only on the edges, and only downwards for the account.** Re-saving
	// the same form must not announce anything; switching somebody off is
	// otherwise discovered as a login that stopped working mid-week, and a
	// roster change is otherwise discovered by turning up on the wrong day.
	if req.IsActive != nil && existing.IsActive && !*req.IsActive {
		h.notifyStaff(id, "Hisob o'chirildi",
			"Hisobingiz vaqtincha o'chirildi — ma'muriyat bilan bog'laning",
			map[string]any{"type": "account"})
	} else if updated != nil && !sameSchedule(existing.Schedule, updated.Schedule) {
		h.notifyStaff(id, "Grafik o'zgardi",
			"Ish grafikingiz yangilandi — ilovadan ko'rib qo'ying",
			map[string]any{"type": "schedule"})
	}
	details := ""
	if req.Password != "" {
		details = "parol o'zgartirildi"
	}
	h.logAction(r, ActStaffUpdate, "staff", id.Hex(), updated.Name, details)
	httpx.JSON(w, http.StatusOK, updated)
}

// AdminDeleteStaff removes the account. The shifts stay: payroll history and
// the branch's own record of who was on that night must survive the account.
func (h *Handler) AdminDeleteStaff(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	removed, err := h.staffByID(r, id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "ishchi topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, removed.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	if _, err := h.Store.Staff.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActStaffDelete, "staff", id.Hex(), removed.Name, removed.Phone)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) staffByID(r *http.Request, id primitive.ObjectID) (*models.Staff, error) {
	var s models.Staff
	if err := h.Store.Staff.FindOne(r.Context(), bson.M{"_id": id}).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// ---- One employee's card ----

// AdminStaffDetail is the staff card: the account, the calendar, the totals and
// the money already handed over.
type AdminStaffDetail struct {
	Staff      models.Staff          `json:"staff"`
	BranchName string                `json:"branchName"`
	Report     StaffReport           `json:"report"`
	Payments   []models.StaffPayment `json:"payments"`
	PaidTotal  int                   `json:"paidTotal"`
}

// AdminGetStaff answers the staff card for a date range.
func (h *Handler) AdminGetStaff(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	s, err := h.staffByID(r, id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "ishchi topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, s.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	from, to, err := dateRange(r.URL.Query())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	rep, err := h.staffReport(r, s, from, to)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	cur, err := h.Store.StaffPayments.Find(r.Context(), bson.M{"staffId": id},
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	payments := []models.StaffPayment{}
	_ = cur.All(r.Context(), &payments)
	paid := 0
	for _, p := range payments {
		paid += p.Amount
	}

	httpx.JSON(w, http.StatusOK, AdminStaffDetail{
		Staff:      *s,
		BranchName: h.branchNames(r)[s.BranchID],
		Report:     *rep,
		Payments:   payments,
		PaidTotal:  paid,
	})
}

// ---- Correcting a shift by hand ----

type shiftPayload struct {
	Date string `json:"date" validate:"required"`
	In   string `json:"in" validate:"required"` // "HH:MM"
	Out  string `json:"out"`                    // "" = still open
	Note string `json:"note"`
}

// shiftTimes turns a date plus two clock strings into timestamps, letting the
// end fall on the next day when the shift crosses midnight.
func shiftTimes(date, in, out string) (time.Time, *time.Time, error) {
	day, err := parseDay(date)
	if err != nil {
		return time.Time{}, nil, errInvalidDate
	}
	inM := clockMinutes(in)
	if inM < 0 {
		return time.Time{}, nil, errInvalidTime
	}
	start := day.Add(time.Duration(inM) * time.Minute)
	if strings.TrimSpace(out) == "" {
		return start, nil, nil
	}
	outM := clockMinutes(out)
	if outM < 0 {
		return time.Time{}, nil, errInvalidTime
	}
	end := day.Add(time.Duration(outM) * time.Minute)
	if !end.After(start) {
		end = end.AddDate(0, 0, 1)
	}
	return start, &end, nil
}

// AdminCreateShift writes a shift the employee could not: a dead phone, a
// forgotten clock-out, a day worked before the app was installed.
//
// It carries the admin's name, so the calendar always distinguishes a punch the
// employee made from one written for them.
func (h *Handler) AdminCreateShift(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	s, err := h.staffByID(r, id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "ishchi topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, s.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req shiftPayload
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	start, end, err := shiftTimes(req.Date, req.In, req.Out)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	shift := models.Shift{
		StaffID:   s.ID,
		BranchID:  s.BranchID,
		Date:      start.Format(dayLayout),
		In:        start,
		Out:       end,
		EditedBy:  h.adminName(r),
		Note:      clampText(req.Note, 200),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if end != nil {
		shift.Minutes = int(end.Sub(start).Minutes())
	}
	res, err := h.Store.Shifts.InsertOne(r.Context(), shift)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	shift.ID = oidOf(res.InsertedID)
	h.logAction(r, ActShiftEdit, "staff", s.ID.Hex(), s.Name,
		"smena qo'shildi: "+shift.Date)
	httpx.JSON(w, http.StatusCreated, shift)
}

// AdminUpdateShift corrects an existing shift.
func (h *Handler) AdminUpdateShift(w http.ResponseWriter, r *http.Request) {
	shiftID, err := objectID(chi.URLParam(r, "shiftId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var existing models.Shift
	if err := h.Store.Shifts.FindOne(r.Context(), bson.M{"_id": shiftID}).Decode(&existing); err != nil {
		httpx.Error(w, http.StatusNotFound, "smena topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, existing.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req shiftPayload
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	start, end, err := shiftTimes(req.Date, req.In, req.Out)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := bson.M{
		"date":      start.Format(dayLayout),
		"in":        start,
		"minutes":   0,
		"editedBy":  h.adminName(r),
		"note":      clampText(req.Note, 200),
		"updatedAt": time.Now(),
	}
	update := bson.M{"$set": set}
	if end != nil {
		set["out"] = *end
		set["minutes"] = int(end.Sub(start).Minutes())
	} else {
		// Reopening a shift: the closing punch has to go, or the calendar shows
		// a finished day that is somehow still running.
		update["$unset"] = bson.M{"out": "", "outAt": ""}
	}
	if _, err := h.Store.Shifts.UpdateByID(r.Context(), shiftID, update); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ **The correction reaches the person it is about.** A shift edited in
	// the office changes what somebody is paid for a day they have already
	// worked, and until now the only place it showed was the payslip — which
	// is the worst moment to find out and the hardest one to argue about.
	h.notifyStaff(existing.StaffID, "Smena tuzatildi",
		set["date"].(string)+" kungi smenangiz o'zgartirildi",
		map[string]any{"type": "shift"})
	h.logAction(r, ActShiftEdit, "staff", existing.StaffID.Hex(), "",
		"smena tahrirlandi: "+set["date"].(string))
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// AdminDeleteShift removes a shift written by mistake.
func (h *Handler) AdminDeleteShift(w http.ResponseWriter, r *http.Request) {
	shiftID, err := objectID(chi.URLParam(r, "shiftId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var existing models.Shift
	if err := h.Store.Shifts.FindOne(r.Context(), bson.M{"_id": shiftID}).Decode(&existing); err != nil {
		httpx.Error(w, http.StatusNotFound, "smena topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, existing.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	if _, err := h.Store.Shifts.DeleteOne(r.Context(), bson.M{"_id": shiftID}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActShiftEdit, "staff", existing.StaffID.Hex(), "",
		"smena o'chirildi: "+existing.Date)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Payroll (hisob-kitob) ----

// PayrollRow is one employee on the money screen.
type PayrollRow struct {
	StaffID string `json:"staffId"`
	// Set instead of StaffID on a courier's row.
	//
	// ⚠️ **Two fields rather than one id plus a "kind" flag.** The pay button
	// posts to a different endpoint for each, and a single id field would let a
	// row that got its flag wrong pay a cook through the courier endpoint —
	// silently, because both ids are valid ObjectIDs. Empty is the signal, and
	// the browser has `hasId()` for exactly this.
	CourierID  string                `json:"courierId,omitempty"`
	Name       string                `json:"name"`
	Position   string                `json:"position"`
	BranchName string                `json:"branchName"`
	PayMode    models.StaffPayMode   `json:"payMode"`
	PayPeriod  models.StaffPayPeriod `json:"payPeriod"`
	Rate       int                   `json:"rate"`
	IsActive   bool                  `json:"isActive"`

	// The period the employee is currently owed for.
	From string `json:"from"`
	To   string `json:"to"`
	// Hours behind the money, so the figure can be checked rather than trusted.
	Days     int `json:"days"`
	Worked   int `json:"worked"`
	Expected int `json:"expected"`
	Earned   int `json:"earned"`
	Paid     int `json:"paid"`
	Due      int `json:"due"`
}

// AdminPayroll is the money screen: what each employee has earned this pay
// period, what has been handed over and what is still owed.
//
// Each row is read over that employee's *own* period — a cook paid every ten
// days and a manager paid monthly are both answered correctly on one screen,
// which is the whole reason the period lives on the employee.
func (h *Handler) AdminPayroll(w http.ResponseWriter, r *http.Request) {
	filter, _, err := h.staffScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	cur, err := h.Store.Staff.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	staff := []models.Staff{}
	_ = cur.All(r.Context(), &staff)

	branchNames := h.branchNames(r)
	now := time.Now()
	rows := make([]PayrollRow, 0, len(staff))
	totalDue, totalEarned, totalPaid := 0, 0, 0

	for i := range staff {
		s := &staff[i]
		pFrom, pTo := payPeriodBounds(s.PayPeriod, now)
		fromKey, toKey := pFrom.Format(dayLayout), pTo.Format(dayLayout)
		shifts, err := h.shiftsInRange(r.Context(), s.ID, fromKey, toKey)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		totals := sumDays(buildDays(s, shifts, pFrom, pTo))
		paid, _ := h.paidBetween(r.Context(), s.ID, fromKey, toKey)

		rate := s.HourlyRate
		switch s.PayMode {
		case models.PayShift:
			rate = s.ShiftRate
		case models.PayMonthly:
			rate = s.MonthlyRate
		}
		row := PayrollRow{
			StaffID: s.ID.Hex(), Name: s.Name, Position: s.Position,
			BranchName: branchNames[s.BranchID],
			PayMode:    s.PayMode, PayPeriod: s.PayPeriod, Rate: rate,
			IsActive: s.IsActive,
			From:     fromKey, To: toKey,
			Days: totals.Days, Worked: totals.Worked, Expected: totals.Expected,
			Earned: totals.Pay, Paid: paid, Due: totals.Pay - paid,
		}
		rows = append(rows, row)
		totalEarned += row.Earned
		totalPaid += row.Paid
		totalDue += row.Due
	}

	// ⚠️ **Couriers on the same screen, in the same shape.** An owner settling
	// up at the end of the month should not read two screens with two meanings
	// of "earned" and "paid" — the kitchen and the bikes are the same question
	// about different people. Their rows carry `courierId` instead of
	// `staffId`, because the pay button posts somewhere else.
	for _, row := range h.courierPayrollRows(r, now) {
		rows = append(rows, row)
		totalEarned += row.Earned
		totalPaid += row.Paid
		totalDue += row.Due
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"rows":   rows,
		"earned": totalEarned,
		"paid":   totalPaid,
		"due":    totalDue,
	})
}

type paymentPayload struct {
	Amount int    `json:"amount" validate:"required,gt=0"`
	From   string `json:"from"`
	To     string `json:"to"`
	Note   string `json:"note"`
	// Whether the notes came out of the safe.
	//
	// ⚠️ **Asked rather than assumed.** A wage can just as easily be
	// transferred to a card or handed over from a till, and a safe balance that
	// counted every payment would be a confident figure about a box nobody
	// opened. Ticked, it writes one linked row — see handlers/safe.go.
	FromSafe bool `json:"fromSafe"`
}

// AdminPayStaff records money handed to an employee.
//
// A ledger entry, not a counter reset: it says which period it settles, so the
// payroll screen can still answer "what did we pay in March" after April has
// started.
func (h *Handler) AdminPayStaff(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	s, err := h.staffByID(r, id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "ishchi topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, s.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req paymentPayload
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// Default to the period the employee is currently owed for — the case the
	// button is pressed in nine times out of ten.
	pFrom, pTo := payPeriodBounds(s.PayPeriod, time.Now())
	from, to := pFrom.Format(dayLayout), pTo.Format(dayLayout)
	if v, err := parseDay(req.From); err == nil {
		from = v.Format(dayLayout)
	}
	if v, err := parseDay(req.To); err == nil {
		to = v.Format(dayLayout)
	}
	if to < from {
		from, to = to, from
	}

	p := models.StaffPayment{
		StaffID:  s.ID,
		BranchID: s.BranchID,
		Amount:   req.Amount,
		From:     from,
		To:       to,
		PaidBy:   h.adminName(r),
		Note:     clampText(req.Note, 200),
		At:       time.Now(),
	}
	res, err := h.Store.StaffPayments.InsertOne(r.Context(), p)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	p.ID = oidOf(res.InsertedID)
	if req.FromSafe {
		h.recordSafeMovement(r.Context(), models.SafeEntry{
			BranchID: s.BranchID, Kind: models.SafeOut, Amount: p.Amount,
			At: p.At, Category: "ish haqi", Note: s.Name, By: p.PaidBy,
			RefKind: models.SafeRefSalary, RefID: p.ID,
		})
	}
	// ⚠️ **The one message this app exists for as much as the clock.** Pay is
	// recorded on somebody else's screen, for a period the employee cannot see
	// from theirs, and until now the first they knew of it was the money — or
	// its absence — at the end of the month. Sent after the write: a relay in
	// another country must never be why a payment fails to record.
	h.notifyStaff(s.ID, "Ish haqi yozildi",
		// ⚠️ The amount as a number inside the sentence rather than a
		// pre-formatted string: "1 200 000 so'm" carries an Uzbek word into a
		// Russian notification, because a captured value is never translated.
		fmt.Sprintf("%d so'm (%s — %s)", req.Amount, from, to),
		map[string]any{"type": "payment"})
	h.logAction(r, ActStaffPay, "staff", s.ID.Hex(), s.Name,
		formatUZS(req.Amount)+" ("+from+" — "+to+")")
	httpx.JSON(w, http.StatusCreated, p)
}

// AdminDeleteStaffPayment reverses a payment entered by mistake. Logged, like
// every other correction to somebody's money.
func (h *Handler) AdminDeleteStaffPayment(w http.ResponseWriter, r *http.Request) {
	paymentID, err := objectID(chi.URLParam(r, "paymentId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var p models.StaffPayment
	if err := h.Store.StaffPayments.FindOne(r.Context(), bson.M{"_id": paymentID}).Decode(&p); err != nil {
		httpx.Error(w, http.StatusNotFound, "to'lov topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, p.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	if _, err := h.Store.StaffPayments.DeleteOne(r.Context(), bson.M{"_id": paymentID}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActStaffPay, "staff", p.StaffID.Hex(), "",
		"to'lov bekor qilindi: "+formatUZS(p.Amount))
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// sameSchedule reports whether two rosters say the same thing.
//
// ⚠️ **Compared field by field rather than with reflect.DeepEqual**, because
// the answer decides whether somebody's phone buzzes: a nil slice and an empty
// one are the same roster, and DeepEqual calls them different — so every save
// of a person who has never been given one would announce a change that did not
// happen, and the third of those teaches them to ignore the second.
func sameSchedule(a, b []models.StaffSchedule) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
