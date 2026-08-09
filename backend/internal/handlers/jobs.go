package handlers

import (
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Vacancies, and the people who answer them.
//
// ⚠️ **Applying does not require an account, and that is the decision the rest of this
// file is built around.** Everything else a guest can do here is behind an SMS login,
// because it costs the restaurant money or moves their food. This costs nothing and the
// applicant is not a customer — asking a cook to prove a phone number by SMS before they
// can say "I am interested" loses the applicant, not the spam.
//
// So the spam is handled where it actually is: two applications per number a day, one per
// vacancy ever, and a name that has to look like a name. All three are cheap, none of them
// asks anything of the honest applicant, and each refuses in words rather than silently —
// somebody who thinks they applied and did not is worse than somebody who is told to wait.

// ---- Public ----

// GetVacancies lists what the restaurant is hiring for.
func (h *Handler) GetVacancies(w http.ResponseWriter, r *http.Request) {
	brand, _ := h.publicBrand(r)
	filter := bson.M{"isActive": true}
	if brand != nil {
		filter["$or"] = []bson.M{
			{"brandId": brand.ID},
			{"brandId": bson.M{"$exists": false}},
			{"brandId": primitive.NilObjectID},
		}
	}
	cur, err := h.Store.Vacancies.Find(r.Context(), filter, options.Find().SetSort(bson.D{
		{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: -1},
	}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.Vacancy{}
	_ = cur.All(r.Context(), &rows)
	// ⚠️ The application count is not published. It is the owner's number, and "42 people
	// already applied" is either a reason not to bother or a claim about how desperate the
	// restaurant is — neither is theirs to read.
	for i := range rows {
		rows[i].Applications = 0
	}
	httpx.JSON(w, http.StatusOK, rows)
}

type applyRequest struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Comment string `json:"comment"`
}

// ApplyForVacancy records one person's interest.
func (h *Handler) ApplyForVacancy(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "vakansiya topilmadi")
		return
	}
	var req applyRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	// Two characters, because "Ali" exists and so does "Bo". The check is against an
	// empty box and a stray keystroke, not against unusual names.
	if len([]rune(name)) < 2 || len([]rune(name)) > 60 {
		httpx.Error(w, http.StatusBadRequest, "ismingizni yozing")
		return
	}
	phone, ok := normalizePhone(req.Phone)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "telefon raqami noto'g'ri")
		return
	}
	comment := strings.TrimSpace(req.Comment)
	if len([]rune(comment)) > 500 {
		comment = string([]rune(comment)[:500])
	}

	var vacancy models.Vacancy
	if err := h.Store.Vacancies.FindOne(r.Context(),
		bson.M{"_id": id, "isActive": true}).Decode(&vacancy); err != nil {
		httpx.Error(w, http.StatusNotFound, "bu vakansiya endi yopilgan")
		return
	}

	// ⚠️ One application per number per vacancy, for ever. A second one is not new
	// information — the restaurant already has the number and will ring it — and two rows
	// for one person is how a list of twelve applicants reads as twenty.
	if n, err := h.Store.JobApplications.CountDocuments(r.Context(), bson.M{
		"vacancyId": id, "phone": phone,
	}, options.Count().SetLimit(1)); err == nil && n > 0 {
		httpx.Error(w, http.StatusConflict,
			"arizangiz allaqachon qabul qilingan — tez orada aloqaga chiqamiz")
		return
	}

	// ⚠️ And two a day across all vacancies. Somebody genuinely interested in the kitchen
	// and the hall applies twice; a script applies to everything. The number is low enough
	// to stop a flood and high enough that no honest applicant meets it.
	since := time.Now().Add(-24 * time.Hour)
	if n, err := h.Store.JobApplications.CountDocuments(r.Context(), bson.M{
		"phone": phone, "createdAt": bson.M{"$gte": since},
	}, options.Count().SetLimit(3)); err == nil && n >= 2 {
		httpx.Error(w, http.StatusTooManyRequests,
			"bugun ikkitadan ortiq ariza qabul qilinmaydi — ertaga urinib ko'ring")
		return
	}

	app := models.JobApplication{
		VacancyID: id,
		// The title as it was when they applied: a vacancy renamed next month must not
		// rewrite what somebody applied for.
		VacancyTitle: strings.TrimSpace(vacancy.Title.Uz),
		BranchID:     vacancy.BranchID,
		Name:         name,
		Phone:        phone,
		Comment:      comment,
		Status:       models.JobNew,
		CreatedAt:    time.Now(),
	}
	if _, err := h.Store.JobApplications.InsertOne(r.Context(), app); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// The counter is a convenience for the list screen; a failed increment is not worth
	// failing an application over.
	_, _ = h.Store.Vacancies.UpdateByID(r.Context(), id,
		bson.M{"$inc": bson.M{"applications": 1}})

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Admin ----

func (h *Handler) AdminListVacancies(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := bson.M{}
	if !scope.BrandID.IsZero() {
		filter["$or"] = []bson.M{
			{"brandId": scope.BrandID},
			{"brandId": bson.M{"$exists": false}},
			{"brandId": primitive.NilObjectID},
		}
	}
	cur, err := h.Store.Vacancies.Find(r.Context(), filter, options.Find().SetSort(bson.D{
		{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: -1},
	}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.Vacancy{}
	_ = cur.All(r.Context(), &rows)
	httpx.JSON(w, http.StatusOK, rows)
}

type vacancyRequest struct {
	Title       models.LocalizedText `json:"title"`
	Description models.LocalizedText `json:"description"`
	Salary      string               `json:"salary"`
	Employment  string               `json:"employment"`
	SortOrder   int                  `json:"sortOrder"`
	IsActive    *bool                `json:"isActive"`
	BranchID    string               `json:"branchId"`
}

func (h *Handler) AdminSaveVacancy(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req vacancyRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Title.Uz) == "" {
		httpx.Error(w, http.StatusBadRequest, "lavozim nomini yozing")
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	set := bson.M{
		"title":       req.Title,
		"description": req.Description,
		"salary":      strings.TrimSpace(req.Salary),
		"employment":  strings.TrimSpace(req.Employment),
		"sortOrder":   req.SortOrder,
		"isActive":    active,
		"updatedAt":   time.Now(),
	}
	if branchID, err := objectID(strings.TrimSpace(req.BranchID)); err == nil {
		set["branchId"] = branchID
	}

	if idParam := strings.TrimSpace(chi.URLParam(r, "id")); idParam != "" {
		id, err := objectID(idParam)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "id noto'g'ri")
			return
		}
		if _, err := h.Store.Vacancies.UpdateByID(r.Context(), id,
			bson.M{"$set": set}); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		h.logAction(r, ActVacancySave, "vacancy", id.Hex(), req.Title.Uz, "")
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	set["brandId"] = scope.BrandID
	set["createdAt"] = time.Now()
	set["applications"] = 0
	res, err := h.Store.Vacancies.InsertOne(r.Context(), set)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.InsertedID.(primitive.ObjectID)
	h.logAction(r, ActVacancySave, "vacancy", id.Hex(), req.Title.Uz, "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

func (h *Handler) AdminDeleteVacancy(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	if _, err := h.Store.Vacancies.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ The applications are kept. Somebody applied to a real job, and the restaurant may
	// still want to ring them — deleting the vacancy is closing a position, not
	// unremembering the people who answered it.
	h.logAction(r, ActVacancyDelete, "vacancy", id.Hex(), "", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// AdminListApplications is who applied, newest first.
func (h *Handler) AdminListApplications(w http.ResponseWriter, r *http.Request) {
	filter := bson.M{}
	if v := strings.TrimSpace(r.URL.Query().Get("vacancyId")); v != "" {
		if id, err := objectID(v); err == nil {
			filter["vacancyId"] = id
		}
	}
	switch strings.TrimSpace(r.URL.Query().Get("status")) {
	case models.JobNew:
		filter["status"] = models.JobNew
	case models.JobCalled:
		filter["status"] = models.JobCalled
	case models.JobHired:
		filter["status"] = models.JobHired
	case models.JobRefused:
		filter["status"] = models.JobRefused
	}
	cur, err := h.Store.JobApplications.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(300))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.JobApplication{}
	_ = cur.All(r.Context(), &rows)
	httpx.JSON(w, http.StatusOK, rows)
}

// AdminUpdateApplication moves one application along.
func (h *Handler) AdminUpdateApplication(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	var req struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	switch req.Status {
	case models.JobNew, models.JobCalled, models.JobHired, models.JobRefused:
	default:
		httpx.Error(w, http.StatusBadRequest, "holat noto'g'ri")
		return
	}
	if _, err := h.Store.JobApplications.UpdateByID(r.Context(), id, bson.M{"$set": bson.M{
		"status": req.Status, "note": strings.TrimSpace(req.Note),
	}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActJobStatus, "job", id.Hex(), req.Status, req.Note)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
