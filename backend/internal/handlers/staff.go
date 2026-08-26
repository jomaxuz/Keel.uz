package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// The employee's own app (/staff).
//
// It does one thing the restaurant cannot get from a paper sheet: it records
// *where* the button was pressed. Everything else on the screen — the calendar,
// the hours, the wage — is read-only reporting built from those punches.

// ---- Auth ----

type staffLoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// StaffLogin issues a JWT with the "staff" role. Accounts are created in the
// admin panel; there is no self-signup.
func (h *Handler) StaffLogin(w http.ResponseWriter, r *http.Request) {
	var req staffLoginRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var s models.Staff
	if err := h.Store.Staff.FindOne(r.Context(),
		bson.M{"username": normalizeUsername(req.Username)}).Decode(&s); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "login yoki parol noto'g'ri")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(s.PasswordHash), []byte(req.Password)) != nil {
		httpx.Error(w, http.StatusUnauthorized, "login yoki parol noto'g'ri")
		return
	}
	if !s.IsActive {
		httpx.Error(w, http.StatusForbidden, "hisob o'chirilgan — ma'muriyat bilan bog'laning")
		return
	}
	token, err := auth.Generate(h.Cfg.JWTSecret, s.ID.Hex(), "staff")
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"token": token, "staff": s})
}

// staffFromCtx loads the employee behind the request's JWT.
func (h *Handler) staffFromCtx(r *http.Request) (models.Staff, bool) {
	claims := middleware.ClaimsFrom(r.Context())
	if claims == nil {
		return models.Staff{}, false
	}
	id, err := objectID(claims.UserID)
	if err != nil {
		return models.Staff{}, false
	}
	var s models.Staff
	if err := h.Store.Staff.FindOne(r.Context(), bson.M{"_id": id}).Decode(&s); err != nil {
		return models.Staff{}, false
	}
	// ⚠️ Every staff-authenticated request comes through here, which is why the
	// role is resolved here: a permission check that ran before this would read
	// the legacy booleans and answer "no" for `void` on somebody whose role
	// says yes.
	h.withRole(r.Context(), &s)
	return s, true
}

// staffWorkplace is what the app needs to draw the "you are N metres away"
// line: where the branch is and how close it insists on.
type staffWorkplace struct {
	BranchID string          `json:"branchId"`
	Name     string          `json:"name"`
	Address  models.GeoPoint `json:"address"`
	RadiusM  int             `json:"radiusM"`
}

// StaffMe returns the signed-in employee, their workplace and whatever shift is
// currently open — everything the home screen renders before its first tap.
func (h *Handler) StaffMe(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	out := map[string]any{"staff": s}
	if branch, err := h.branchByID(r, s.BranchID); err == nil {
		out["workplace"] = staffWorkplace{
			BranchID: branch.ID.Hex(),
			Name:     branch.Name,
			Address:  branch.Address,
			RadiusM:  branch.StaffRadiusM,
		}
	}
	if open, err := h.openShift(r.Context(), s.ID); err == nil {
		out["openShift"] = open
	}
	httpx.JSON(w, http.StatusOK, out)
}

// ---- Clocking in and out ----

type clockRequest struct {
	Action string `json:"action" validate:"required,oneof=in out"`
	// Where the employee is standing. Mandatory: a punch with no position is
	// exactly the punch this feature exists to prevent.
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Accuracy float64 `json:"accuracy"`
	Note     string  `json:"note"`
	// Code scanned from the branch screen. Required only when the branch turns
	// requireKioskCode on; see handlers/kiosk.go for why it rotates.
	Code string `json:"code"`
}

// geofenceBlocked returns why this employee may not clock in or out from where
// they are standing, or "" when they may.
//
// The branch owns the radius, and 0 switches the check off — a tiny café where
// the phone never gets a decent fix is better served by trusting the staff than
// by a button that never works.
func (h *Handler) geofenceBlocked(r *http.Request, s *models.Staff, lat, lng float64) string {
	branch, err := h.branchByID(r, s.BranchID)
	if err != nil {
		return "Sizga filial biriktirilmagan — ma'muriyat bilan bog'laning"
	}
	if branch.StaffRadiusM <= 0 {
		return ""
	}
	if branch.Address.Lat == 0 && branch.Address.Lng == 0 {
		// The rule cannot be applied against an address nobody put on the map.
		// Refusing here would lock the whole branch out of its own app.
		return ""
	}
	if lat == 0 && lng == 0 {
		return "Joylashuv aniqlanmadi — telefonda GPS va joylashuvga ruxsatni yoqing"
	}
	meters := haversineKm(lat, lng, branch.Address.Lat, branch.Address.Lng) * 1000
	if meters > float64(branch.StaffRadiusM) {
		return fmt.Sprintf(
			"Ish joyidan %.0f m uzoqdasiz — kirish/chiqish uchun %d m ichida bo'lishingiz kerak",
			meters, branch.StaffRadiusM)
	}
	return ""
}

// codeAlreadyUsed reports whether this employee has already punched with this
// code. Scoped to the employee on purpose — a code is meant to be scanned by
// everyone arriving at once.
func (h *Handler) codeAlreadyUsed(r *http.Request, staffID primitive.ObjectID, fingerprint string) bool {
	n, err := h.Store.Shifts.CountDocuments(r.Context(), bson.M{
		"staffId": staffID,
		"$or": []bson.M{
			{"inCode": fingerprint},
			{"outCode": fingerprint},
		},
	})
	return err == nil && n > 0
}

// kioskCodeBlocked returns why this punch is refused for want of a valid code,
// or "" when the branch does not ask for one (or the code checks out).
func (h *Handler) kioskCodeBlocked(r *http.Request, s *models.Staff, code string) string {
	branch, err := h.branchByID(r, s.BranchID)
	if err != nil || !branch.RequireKioskCode {
		return ""
	}
	if strings.TrimSpace(code) == "" {
		return "Ish joyidagi ekrandagi QR kodni skaner qiling"
	}
	if !verifyKioskCode(branch, code) {
		// Deliberately one message for "wrong" and "expired": the honest case is
		// almost always an expired code, and saying which is which would tell a
		// guesser whether they were close.
		return "QR kod eskirgan yoki noto'g'ri — ekrandagi yangi kodni skaner qiling"
	}
	return ""
}

// punchAt records where the button was pressed, with the measured distance so
// the admin never has to recompute it from raw coordinates.
func (h *Handler) punchAt(r *http.Request, s *models.Staff, req clockRequest, at time.Time) *models.ShiftPunch {
	p := &models.ShiftPunch{
		Lat: req.Lat, Lng: req.Lng, Accuracy: req.Accuracy, At: at,
	}
	if branch, err := h.branchByID(r, s.BranchID); err == nil &&
		(branch.Address.Lat != 0 || branch.Address.Lng != 0) &&
		(req.Lat != 0 || req.Lng != 0) {
		p.Meters = haversineKm(req.Lat, req.Lng, branch.Address.Lat, branch.Address.Lng) * 1000
	}
	return p
}

// StaffClock is the one button the app has. "in" opens a shift, "out" closes
// the one that is open.
//
// Clocking in twice is refused rather than merged: two open shifts would double
// every hour on the calendar, and the honest fix ("you are already clocked in")
// is also the clearer message.
func (h *Handler) StaffClock(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	if !s.IsActive {
		httpx.Error(w, http.StatusForbidden, "hisob o'chirilgan")
		return
	}
	var req clockRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if msg := h.geofenceBlocked(r, &s, req.Lat, req.Lng); msg != "" {
		httpx.Error(w, http.StatusBadRequest, msg)
		return
	}
	// The code and the position guard different things — a photographed code
	// still needs someone standing at the restaurant, and a spoofed position
	// still needs the code that is only on the screen — so both are checked,
	// never one instead of the other.
	if msg := h.kioskCodeBlocked(r, &s, req.Code); msg != "" {
		httpx.Error(w, http.StatusBadRequest, msg)
		return
	}

	now := time.Now()
	open, openErr := h.openShift(r.Context(), s.ID)

	// One code, one punch — per employee. The direction is decided by whether a
	// shift is open, so scanning the same code twice (a double tap on the camera
	// notification, a reload, "nothing happened so I scanned again") would clock
	// somebody straight back out of the shift they just opened. Two people
	// scanning the same code is fine and expected: they are different employees
	// with their own tokens and their own positions.
	fingerprint := codeFingerprint(req.Code)
	if fingerprint != "" && h.codeAlreadyUsed(r, s.ID, fingerprint) {
		httpx.Error(w, http.StatusConflict,
			"Bu QR kod allaqachon ishlatilgan — ekrandagi yangi kodni skaner qiling")
		return
	}

	if req.Action == "in" {
		if openErr == nil && open != nil {
			httpx.Error(w, http.StatusConflict,
				"Siz allaqachon ishga kirgansiz — avval chiqishni bosing")
			return
		}
		shift := models.Shift{
			StaffID:   s.ID,
			BranchID:  s.BranchID,
			Date:      dayKey(now),
			In:        now,
			InAt:      h.punchAt(r, &s, req, now),
			InCode:    fingerprint,
			Note:      clampText(req.Note, 200),
			CreatedAt: now,
			UpdatedAt: now,
		}
		res, err := h.Store.Shifts.InsertOne(r.Context(), shift)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		shift.ID = oidOf(res.InsertedID)
		httpx.JSON(w, http.StatusCreated, shift)
		return
	}

	// "out"
	if openErr != nil || open == nil {
		httpx.Error(w, http.StatusConflict, "Ochiq smena yo'q — avval ishga kirishni bosing")
		return
	}
	minutes := int(now.Sub(open.In).Minutes())
	if minutes < 0 {
		minutes = 0
	}
	outAt := h.punchAt(r, &s, req, now)
	if _, err := h.Store.Shifts.UpdateByID(r.Context(), open.ID, bson.M{"$set": bson.M{
		"out":       now,
		"outAt":     outAt,
		"outCode":   fingerprint,
		"minutes":   minutes,
		"updatedAt": now,
	}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	open.Out, open.OutAt, open.Minutes = &now, outAt, minutes
	httpx.JSON(w, http.StatusOK, open)
}

// ---- Reporting ----

// StaffReport is the employee's own screen: the calendar for the chosen range,
// the totals under it, and what they have earned against what they have been
// paid.
type StaffReport struct {
	Days   []StaffDay  `json:"days"`
	Totals StaffTotals `json:"totals"`
	From   string      `json:"from"`
	To     string      `json:"to"`

	// Trends, each against the window immediately before it.
	Today StaffTrend `json:"today"`
	Week  StaffTrend `json:"week"`
	Month StaffTrend `json:"month"`

	// The current pay period, whatever range is being browsed above.
	PeriodFrom string `json:"periodFrom"`
	PeriodTo   string `json:"periodTo"`
	PeriodPay  int    `json:"periodPay"`
	PeriodPaid int    `json:"periodPaid"`
	PeriodDue  int    `json:"periodDue"`

	PayMode   models.StaffPayMode   `json:"payMode"`
	PayPeriod models.StaffPayPeriod `json:"payPeriod"`
	Rate      int                   `json:"rate"`
}

// staffReport builds the report for one employee over a range. Shared by the
// employee's app and the admin's staff card, so the two can never disagree.
func (h *Handler) staffReport(r *http.Request, s *models.Staff, from, to time.Time) (*StaffReport, error) {
	shifts, err := h.shiftsInRange(r.Context(), s.ID, from.Format(dayLayout), to.Format(dayLayout))
	if err != nil {
		return nil, err
	}
	days := buildDays(s, shifts, from, to)

	rep := &StaffReport{
		Days:      days,
		Totals:    sumDays(days),
		From:      from.Format(dayLayout),
		To:        to.Format(dayLayout),
		PayMode:   s.PayMode,
		PayPeriod: s.PayPeriod,
	}
	switch s.PayMode {
	case models.PayShift:
		rep.Rate = s.ShiftRate
	case models.PayMonthly:
		rep.Rate = s.MonthlyRate
	default:
		rep.Rate = s.HourlyRate
	}

	// Trends are read over their own windows, independent of what the user is
	// browsing: "5% less than yesterday" must not change because someone
	// widened the calendar.
	now := time.Now()
	today := startOfDay(now)
	trendFrom := today.AddDate(0, 0, -59)
	trendShifts, err := h.shiftsInRange(r.Context(), s.ID,
		trendFrom.Format(dayLayout), today.Format(dayLayout))
	if err != nil {
		return nil, err
	}
	td := buildDays(s, trendShifts, trendFrom, today)
	d := func(t time.Time) string { return t.Format(dayLayout) }

	rep.Today = trendOf(
		workedBetween(td, d(today), d(today)),
		workedBetween(td, d(today.AddDate(0, 0, -1)), d(today.AddDate(0, 0, -1))),
	)
	rep.Week = trendOf(
		workedBetween(td, d(today.AddDate(0, 0, -6)), d(today)),
		workedBetween(td, d(today.AddDate(0, 0, -13)), d(today.AddDate(0, 0, -7))),
	)
	rep.Month = trendOf(
		workedBetween(td, d(today.AddDate(0, 0, -29)), d(today)),
		workedBetween(td, d(today.AddDate(0, 0, -59)), d(today.AddDate(0, 0, -30))),
	)

	pFrom, pTo := payPeriodBounds(s.PayPeriod, now)
	rep.PeriodFrom, rep.PeriodTo = d(pFrom), d(pTo)
	periodShifts, err := h.shiftsInRange(r.Context(), s.ID, rep.PeriodFrom, rep.PeriodTo)
	if err != nil {
		return nil, err
	}
	rep.PeriodPay = sumDays(buildDays(s, periodShifts, pFrom, pTo)).Pay
	rep.PeriodPaid, _ = h.paidBetween(r.Context(), s.ID, rep.PeriodFrom, rep.PeriodTo)
	rep.PeriodDue = rep.PeriodPay - rep.PeriodPaid
	return rep, nil
}

// StaffMyReport answers the employee's own calendar and pay screen.
func (h *Handler) StaffMyReport(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	from, to, err := dateRange(r.URL.Query())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	rep, err := h.staffReport(r, &s, from, to)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, rep)
}

// ---- Roles ----

// withRole fills in what an employee is allowed to do.
//
// ⚠️ **Called wherever a staff record is about to be trusted with a decision.**
// The permissions live on the role, not on the person, so a Staff loaded
// straight out of Mongo answers Can() from the three legacy booleans — which is
// correct for an account nothing has migrated and wrong for everybody else.
// One helper, so the next handler that loads staff cannot quietly get the old
// answer.
func (h *Handler) withRole(ctx context.Context, s *models.Staff) {
	if s == nil || s.RoleID.IsZero() {
		return
	}
	var role models.StaffRole
	if err := h.Store.StaffRoles.FindOne(ctx, bson.M{"_id": s.RoleID}).Decode(&role); err != nil {
		// ⚠️ A missing role leaves the legacy booleans in charge rather than
		// granting nothing: a deleted role must not lock a shift out of the
		// till mid-service. The panel refuses to delete a role that is in use,
		// so this is the belt behind that brace.
		return
	}
	s.RoleName = role.Name
	s.Perms = role.Perms
	// ⚠️ Set only here, where a role document was actually read. An empty
	// list from a real role means "nothing", and that is a different answer
	// from "no role has ever been applied" — see models.Staff.RoleApplied.
	s.RoleApplied = true
}

// withRoles is the same for a list, with one query instead of one per person.
//
// A branch with twenty employees would otherwise open the staff screen with
// twenty extra round trips — the same reason shiftsByStaff exists.
func (h *Handler) withRoles(ctx context.Context, rows []models.Staff) {
	ids := make([]primitive.ObjectID, 0, len(rows))
	seen := map[primitive.ObjectID]bool{}
	for i := range rows {
		if id := rows[i].RoleID; !id.IsZero() && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}
	cur, err := h.Store.StaffRoles.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return
	}
	var roles []models.StaffRole
	if err := cur.All(ctx, &roles); err != nil {
		return
	}
	byID := map[primitive.ObjectID]models.StaffRole{}
	for _, r := range roles {
		byID[r.ID] = r
	}
	for i := range rows {
		if r, ok := byID[rows[i].RoleID]; ok {
			rows[i].RoleName = r.Name
			rows[i].Perms = r.Perms
			// Same flag as withRole, for the same reason: an empty list from a
			// real role means nothing, not "fall back to the old booleans".
			rows[i].RoleApplied = true
		}
	}
}
