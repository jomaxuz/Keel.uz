package handlers

// ---- The playlist those televisions play ----
//
// Pictures and videos, in a loop, per branch. The panel side is a list somebody
// reorders; the television side is one endpoint it re-reads only when the
// branch says the list has moved.
//
// ⚠️ **Branch-wide, not per screen.** Two televisions in one room show the same
// restaurant's food — what differs between them is the order board, and that is
// already a property of the screen (models.TVScreen.Mode). A playlist per screen
// would mean uploading the same video four times and remembering to change it
// four times; the one nobody remembers is the one still showing last month's
// promotion.
//
// ⚠️ **The window is sent to the wall, not applied here.** A screen that has
// not reached the internet since Friday must still stop showing an offer that
// ended on Saturday — an expired promotion on a wall is worse than a blank
// screen, because a guest asks for it at the till.

import (
	"errors"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// How many items one branch may loop.
//
// ⚠️ **A cap, because every one of these is downloaded onto every television in
// the branch.** A set with 8 GB of storage and a hundred videos is a screen
// that spends its evening filling its disk instead of playing, and the failure
// arrives weeks after the upload that caused it.
const tvSlideLimit = 60

var errTVSlideLimit = errors.New("ro'yxatda 60 tadan ortiq element bo'lishi mumkin emas")

// bumpTVContent tells this branch's televisions that the loop has changed.
//
// ⚠️ Called from every write here, including the ones that look harmless: a
// renamed slide changes nothing on a wall, but a slide switched off changes
// everything, and a rule of "bump on the interesting ones" is a rule somebody
// gets wrong on the next edit. The cost is one increment.
func (h *Handler) bumpTVContent(r *http.Request, branchID primitive.ObjectID) {
	_, _ = h.Store.Branches.UpdateByID(r.Context(), branchID,
		bson.M{"$inc": bson.M{"tvContentVersion": 1}})
}

// tvSlideView is one row as the panel reads it.
//
// ⚠️ **The dates go out as ready `YYYY-MM-DD` strings as well as timestamps.**
// A `time.Time` reaches the browser in UTC, so slicing a date out of it in
// JavaScript lands on the previous day for every restaurant east of Greenwich —
// the trap CLAUDE.md records against the console's billing period, and the same
// answer: the server sends the string it means.
func tvSlideView(s models.TVSlide) map[string]any {
	out := map[string]any{
		"id":       s.ID.Hex(),
		"branchId": s.BranchID.Hex(),
		"kind":     s.Kind,
		"url":      s.URL,
		"name":     s.Name,
		"seconds":  s.Seconds,
		"order":    s.Order,
		"active":   s.Active,
	}
	if s.StartsAt != nil {
		out["startsAt"] = *s.StartsAt
		out["startsOn"] = s.StartsAt.In(time.Local).Format("2006-01-02")
	}
	if s.EndsAt != nil {
		out["endsAt"] = *s.EndsAt
		out["endsOn"] = s.EndsAt.In(time.Local).Format("2006-01-02")
	}
	return out
}

// AdminTVSlides lists a branch's playlist, in loop order.
func (h *Handler) AdminTVSlides(w http.ResponseWriter, r *http.Request) {
	branchID, ok := h.tvBranch(w, r)
	if !ok {
		return
	}
	rows, err := h.tvSlidesOf(r, branchID, false)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ An empty slice, never nil — the panel reads `.length` off this, and Go
	// marshals a nil slice as `null`. The trap this codebase has hit twice.
	out := []map[string]any{}
	for _, s := range rows {
		out = append(out, tvSlideView(s))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"slides": out,
		"limit":  tvSlideLimit,
	})
}

// tvSlidesOf loads a branch's playlist in loop order.
func (h *Handler) tvSlidesOf(r *http.Request, branchID primitive.ObjectID, activeOnly bool) ([]models.TVSlide, error) {
	filter := bson.M{"branchId": branchID}
	if activeOnly {
		filter["active"] = true
	}
	cur, err := h.Store.TVSlides.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{
			{Key: "order", Value: 1},
			{Key: "createdAt", Value: 1},
		}))
	if err != nil {
		return nil, err
	}
	rows := []models.TVSlide{}
	if err := cur.All(r.Context(), &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

type tvSlideRequest struct {
	Kind    string `json:"kind"`
	URL     string `json:"url"`
	Name    string `json:"name"`
	Seconds int    `json:"seconds"`
	Active  *bool  `json:"active"`
	// `YYYY-MM-DD`, or empty for "no boundary". ⚠️ A date rather than an
	// instant: an owner setting a promotion knows which day it ends and does
	// not know, and should not have to decide, whether that means 00:00 or
	// 23:59. The end of the day is what they mean, and the server says so.
	StartsOn string `json:"startsOn"`
	EndsOn   string `json:"endsOn"`
}

// AdminCreateTVSlide adds one item to the end of a branch's loop.
func (h *Handler) AdminCreateTVSlide(w http.ResponseWriter, r *http.Request) {
	branchID, ok := h.tvBranch(w, r)
	if !ok {
		return
	}
	var req tvSlideRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	kind := strings.TrimSpace(req.Kind)
	if !models.ValidTVSlideKind(kind) {
		httpx.Error(w, http.StatusBadRequest, "fayl turi noma'lum")
		return
	}
	url := strings.TrimSpace(req.URL)
	if url == "" {
		httpx.Error(w, http.StatusBadRequest, "avval faylni yuklang")
		return
	}
	start, end, err := tvSlideWindow(req.StartsOn, req.EndsOn)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	n, err := h.Store.TVSlides.CountDocuments(r.Context(), bson.M{"branchId": branchID})
	// ⚠️ A failed count does not refuse. Mongo being briefly unreachable is not
	// a full playlist, and the same rule the screen cap follows next door.
	if err == nil && n >= tvSlideLimit {
		httpx.Error(w, http.StatusBadRequest, errTVSlideLimit.Error())
		return
	}

	now := time.Now()
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	slide := models.TVSlide{
		BranchID: branchID,
		Kind:     kind,
		URL:      url,
		Name:     clampText(req.Name, 80),
		Seconds:  models.ClampTVSlideSeconds(req.Seconds),
		// At the end of the loop, which is where somebody who just uploaded
		// something expects to find it.
		Order:     int(now.Unix()),
		Active:    active,
		StartsAt:  start,
		EndsAt:    end,
		CreatedAt: now,
		UpdatedAt: now,
	}
	res, err := h.Store.TVSlides.InsertOne(r.Context(), slide)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	slide.ID = oidOf(res.InsertedID)
	h.bumpTVContent(r, branchID)
	h.logAction(r, ActSettingsUpdate, "branch", branchID.Hex(), slide.Name,
		"TV kontent qo'shildi")
	httpx.JSON(w, http.StatusCreated, tvSlideView(slide))
}

type tvSlidePatch struct {
	Name    *string `json:"name"`
	Seconds *int    `json:"seconds"`
	Active  *bool   `json:"active"`
	// ⚠️ Pointers, so an empty string can mean "clear this boundary" while an
	// absent field means "leave it alone". A promotion that has been extended
	// and one that has had its end date removed are different acts, and a plain
	// string cannot tell them apart.
	StartsOn *string `json:"startsOn"`
	EndsOn   *string `json:"endsOn"`
}

// AdminUpdateTVSlide edits one item of the loop.
func (h *Handler) AdminUpdateTVSlide(w http.ResponseWriter, r *http.Request) {
	branchID, slide, ok := h.tvSlideOfBranch(w, r)
	if !ok {
		return
	}
	var req tvSlidePatch
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := bson.M{"updatedAt": time.Now()}
	unset := bson.M{}
	if req.Name != nil {
		set["name"] = clampText(*req.Name, 80)
	}
	if req.Seconds != nil {
		set["seconds"] = models.ClampTVSlideSeconds(*req.Seconds)
	}
	if req.Active != nil {
		set["active"] = *req.Active
	}
	// The window is read as a pair even when only one half was sent: "ends
	// before it starts" is a slide that never plays, and it has to be refused
	// against what the row will hold, not against what arrived.
	startsOn, endsOn := slideDay(slide.StartsAt), slideDay(slide.EndsAt)
	if req.StartsOn != nil {
		startsOn = strings.TrimSpace(*req.StartsOn)
	}
	if req.EndsOn != nil {
		endsOn = strings.TrimSpace(*req.EndsOn)
	}
	if req.StartsOn != nil || req.EndsOn != nil {
		start, end, err := tvSlideWindow(startsOn, endsOn)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if start != nil {
			set["startsAt"] = *start
		} else {
			unset["startsAt"] = ""
		}
		if end != nil {
			set["endsAt"] = *end
		} else {
			unset["endsAt"] = ""
		}
	}
	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	if _, err := h.Store.TVSlides.UpdateByID(r.Context(), slide.ID, update); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.bumpTVContent(r, branchID)
	h.logAction(r, ActSettingsUpdate, "branch", branchID.Hex(), slide.Name,
		"TV kontent sozlandi")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// AdminRemoveTVSlide drops one item from the loop, and its file with it.
func (h *Handler) AdminRemoveTVSlide(w http.ResponseWriter, r *http.Request) {
	branchID, slide, ok := h.tvSlideOfBranch(w, r)
	if !ok {
		return
	}
	if _, err := h.Store.TVSlides.DeleteOne(r.Context(),
		bson.M{"_id": slide.ID, "branchId": branchID}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.dropTVVideo(r, slide)
	h.bumpTVContent(r, branchID)
	h.logAction(r, ActSettingsUpdate, "branch", branchID.Hex(), slide.Name,
		"TV kontent o'chirildi")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// dropTVVideo removes the file behind a deleted video, if nothing else uses it.
//
// ⚠️ **Videos only, and this is the one place in the codebase that deletes an
// upload.** A photograph is a hundred kilobytes and may be shared with a dish;
// a minute of 1080p is fifty megabytes that nothing else will ever reference,
// and a tenant's disk filling up with promotions from three summers ago is a
// support call that arrives as "the site is down".
//
// ⚠️ **Only under our own upload directory, and only after checking no other
// slide points at it.** Two branches can be handed the same URL by an owner who
// copied it, and a delete that took the file would blank the other one's wall.
func (h *Handler) dropTVVideo(r *http.Request, slide models.TVSlide) {
	if slide.Kind != models.TVSlideVideo {
		return
	}
	name := uploadNameOf(slide.URL)
	if name == "" {
		return
	}
	n, err := h.Store.TVSlides.CountDocuments(r.Context(), bson.M{"url": slide.URL})
	if err != nil || n > 0 {
		return
	}
	_ = os.Remove(filepath.Join(h.Cfg.UploadDir, name))
}

// uploadNameOf is the file name behind one of our own upload URLs, or "" for
// anything that is not one.
//
// ⚠️ Cleaned as an absolute path and made relative again, the same way the
// uploads route does it: this string reaches a `filepath.Join`, and a stored
// URL is only as trustworthy as whatever wrote it.
func uploadNameOf(url string) string {
	i := strings.Index(url, "/uploads/")
	if i < 0 {
		return ""
	}
	name := strings.TrimPrefix(path.Clean("/"+url[i+len("/uploads/"):]), "/")
	// ⚠️ **A flat name, and nothing else.** Everything AdminTVUpload writes is
	// a random name directly in the upload directory, so a stored URL naming a
	// directory is not one of ours — it is a seeded demo file, a derivative
	// under `.thumb`, or a path somebody assembled. None of those are this
	// function's to remove, and `path.Clean` alone would hand over the last two.
	if name == "" || strings.ContainsRune(name, '/') || strings.HasPrefix(name, ".") {
		return ""
	}
	return name
}

type tvReorderRequest struct {
	IDs []string `json:"ids"`
}

// AdminReorderTVSlides writes the loop's order from the list the panel shows.
//
// ⚠️ **The whole order, sent as one list, rather than "move this one up".** The
// panel already knows the order it is drawing; a per-row nudge has to be applied
// against a list the server re-derives, and two managers dragging rows in two
// browsers is how a playlist ends up with two items at position three.
func (h *Handler) AdminReorderTVSlides(w http.ResponseWriter, r *http.Request) {
	branchID, ok := h.tvBranch(w, r)
	if !ok {
		return
	}
	var req tvReorderRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	for i, raw := range req.IDs {
		id, err := objectID(strings.TrimSpace(raw))
		if err != nil {
			continue
		}
		// ⚠️ The branch is in the filter, not merely checked: `_id` alone must
		// never select a document, so an id from another restaurant's playlist
		// moves nothing rather than reordering somebody else's dining room.
		_, _ = h.Store.TVSlides.UpdateOne(r.Context(),
			bson.M{"_id": id, "branchId": branchID},
			bson.M{"$set": bson.M{"order": i, "updatedAt": time.Now()}})
	}
	h.bumpTVContent(r, branchID)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- The television's side ----

// TVPlaylist is what a paired screen plays.
//
// ⚠️ **Read only when the version moves.** The heartbeat already carries the
// branch's content version once a minute; a screen that re-downloaded the list
// every time would spend a restaurant's evening asking a question whose answer
// is almost always "the same as last minute".
func (h *Handler) TVPlaylist(w http.ResponseWriter, r *http.Request) {
	screen, branch, err := h.tvScreen(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	rows, err := h.tvSlidesOf(r, screen.BranchID, true)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ Switched-off slides are dropped here; **dated ones are not**. "Off" is
	// a decision somebody made and it does not change on its own, so the wall
	// need never know about it. A window does change on its own — including
	// while the screen is offline — so the dates travel with the list and the
	// television applies them itself.
	out := []map[string]any{}
	for _, s := range rows {
		item := map[string]any{
			"id":      s.ID.Hex(),
			"kind":    s.Kind,
			"url":     s.URL,
			"seconds": s.Seconds,
		}
		if s.StartsAt != nil {
			item["startsAt"] = *s.StartsAt
		}
		if s.EndsAt != nil {
			item["endsAt"] = *s.EndsAt
		}
		out = append(out, item)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"slides":  out,
		"version": branch.TVContentVersion,
		// The screen's own clock is not trusted for anything, the playing
		// window least of all — see TVMe.
		"serverTime": time.Now(),
	})
}

// ---- Shared plumbing ----

// tvBranch resolves the branch in the URL and the caller's right to it.
func (h *Handler) tvBranch(w http.ResponseWriter, r *http.Request) (primitive.ObjectID, bool) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return id, false
	}
	if err := h.requireBranchAccess(r, id); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return id, false
	}
	return id, true
}

// tvSlideOfBranch loads the slide named in the URL, inside the branch named in
// the URL — the same rule as tvScreenOfBranch, and for the same reason.
func (h *Handler) tvSlideOfBranch(w http.ResponseWriter, r *http.Request) (primitive.ObjectID, models.TVSlide, bool) {
	var slide models.TVSlide
	branchID, ok := h.tvBranch(w, r)
	if !ok {
		return branchID, slide, false
	}
	slideID, err := objectID(chi.URLParam(r, "slideId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return branchID, slide, false
	}
	if err := h.Store.TVSlides.FindOne(r.Context(), bson.M{
		"_id": slideID, "branchId": branchID,
	}).Decode(&slide); err != nil {
		httpx.Error(w, http.StatusNotFound, "element topilmadi")
		return branchID, slide, false
	}
	return branchID, slide, true
}

var errTVSlideWindow = errors.New("tugash sanasi boshlanish sanasidan oldin")

// tvSlideWindow turns two typed days into the instants a television compares
// against.
//
// ⚠️ **The end date is the end of that day, not its midnight.** An owner who
// types 30 September means the promotion runs through the thirtieth; taking the
// date literally would take it off the wall as the restaurant opened that
// morning, and the report of that arrives as "the screen is broken".
//
// ⚠️ **Local time, deliberately.** The dates are typed by somebody standing in
// the restaurant; parsed as UTC they would move the boundary five hours in
// Tashkent — the same trap the container's missing tzdata produced, on the same
// clock, in the opposite direction.
func tvSlideWindow(startsOn, endsOn string) (*time.Time, *time.Time, error) {
	start, err := parseSlideDay(startsOn, false)
	if err != nil {
		return nil, nil, err
	}
	end, err := parseSlideDay(endsOn, true)
	if err != nil {
		return nil, nil, err
	}
	if start != nil && end != nil && end.Before(*start) {
		return nil, nil, errTVSlideWindow
	}
	return start, end, nil
}

var errTVSlideDate = errors.New("sana noto'g'ri")

func parseSlideDay(day string, endOfDay bool) (*time.Time, error) {
	day = strings.TrimSpace(day)
	if day == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", day, time.Local)
	if err != nil {
		return nil, errTVSlideDate
	}
	if endOfDay {
		t = t.Add(24*time.Hour - time.Second)
	}
	return &t, nil
}

// slideDay renders a stored boundary back into the day it was typed as.
//
// ⚠️ `.In(time.Local)` before formatting: the driver hands every time.Time back
// with a UTC location, so formatting it raw names the previous day for every
// evening boundary — the trap CLAUDE.md records twice.
func slideDay(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.In(time.Local).Format("2006-01-02")
}
