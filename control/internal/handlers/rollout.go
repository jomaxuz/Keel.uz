package handlers

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"keel-control/internal/httpx"
	"keel-control/internal/middleware"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Rolling out a new tenant image.
//
// A deploy builds and tags the tenant image, and then nothing happens: every
// existing container keeps running the image it was created from. Backend fixes
// reached customers only when somebody pressed "retry" on each one by hand,
// which for fifty restaurants means it never happened.
//
// The design follows from what makes this dangerous rather than tedious:
//
//   - **The tag is not the truth.** A container reports itself perfectly
//     healthy while running last month's code, so "up to date" is decided by
//     comparing image *ids*, never by whether a deploy ran. This is the same
//     trap the deploy script fell into — build the image, leave the container.
//
//   - **One tenant at a time, and only after the previous one serves traffic.**
//     provisionTenant waits on the tenant's own /health before returning, so
//     the rollout advances at the speed of things actually working. Fifty
//     containers restarting together would hit one Mongo with fifty startups.
//
//   - **One rollout at a time.** Two overlapping runs would recreate the same
//     container twice, and the second would find the name taken mid-swap —
//     exactly what the deploy lock exists to prevent, one layer up.
//
//   - **A broken image stops the rollout, a broken tenant does not.** One
//     customer whose container will not come up is a support ticket; an image
//     that fails for everybody is an outage, and continuing through it would
//     take down all fifty restaurants one at a time while reporting progress.
//     Three failures in a row is the line.
//
//   - **Suspended tenants are skipped, not started.** Their containers are
//     stopped on purpose. Recreating one would quietly put a non-paying
//     customer back online; it picks up the new image when it resumes.

// How many consecutive failures mean the image is broken rather than the
// tenant. Three: one is bad luck, two is a coincidence, three is the build.
const rolloutBreaker = 3

// rolloutMu guards against a second run starting while one is in flight. In
// memory rather than in the database because it guards *this process's*
// goroutine; the stored status is what the console reads.
var rolloutMu sync.Mutex

// startRollout claims the right to run. The bool is false when one is already
// going.
func tryStartRollout() bool { return rolloutMu.TryLock() }

func finishRollout() { rolloutMu.Unlock() }

// StartRollout begins moving every live tenant onto the current image.
//
// Returns immediately: a rollout of fifty tenants takes minutes, and an HTTP
// request that waits for it will be cut off by some proxy in the middle,
// leaving the operator with no idea whether it is still going. Progress is
// read back from GET /rollout.
func (h *Handler) StartRollout(w http.ResponseWriter, r *http.Request) {
	if h.Docker == nil {
		httpx.Error(w, http.StatusBadRequest,
			"bu serverda konteynerlar boshqarilmaydi (DOCKER_SOCKET yo'q)")
		return
	}
	// Resolve the tag first. A missing or unbuilt image is the one failure
	// worth refusing up front rather than discovering on the first tenant.
	imageID, err := h.Docker.ImageID(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if !tryStartRollout() {
		httpx.Error(w, http.StatusConflict, "yangilash allaqachon ketmoqda")
		return
	}

	claims := middleware.From(r.Context())
	by := ""
	if claims != nil {
		by = claims.Username
	}
	run := models.Rollout{
		StartedAt: time.Now(),
		StartedBy: by,
		Image:     h.Docker.Image(),
		ImageID:   imageID,
		Status:    "running",
	}
	if err := h.saveRollout(r.Context(), &run); err != nil {
		finishRollout()
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Detached from the request: the operator's browser closing must not
	// abandon half the customers on the old image.
	go func() {
		defer finishRollout()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()
		h.runRollout(ctx, &run)
	}()

	httpx.JSON(w, http.StatusOK, run)
}

// runRollout walks the tenants one at a time.
func (h *Handler) runRollout(ctx context.Context, run *models.Rollout) {
	cur, err := h.Store.Tenants.Find(ctx, bson.M{})
	if err != nil {
		h.endRollout(ctx, run, "failed", err.Error())
		return
	}
	var tenants []models.Tenant
	if err := cur.All(ctx, &tenants); err != nil {
		h.endRollout(ctx, run, "failed", err.Error())
		return
	}

	run.Total = len(tenants)
	consecutive := 0

	for i := range tenants {
		t := tenants[i]
		item := models.RolloutItem{Slug: t.Slug, Name: t.Name, At: time.Now()}

		switch {
		case t.Offline():
			// Stopped on purpose. Recreating would put a non-payer back online.
			item.Status = "skipped"
			item.Note = "to'xtatilgan — yangi image qayta yoqilganda oladi"

		default:
			st, err := h.Docker.Status(ctx, t.Slug)
			if err == nil && st.Status == "running" && st.ImageID == run.ImageID {
				item.Status = "current"
				item.Note = "allaqachon yangi image'da"
				break
			}
			// The real work. provisionTenant waits on the tenant's own health
			// endpoint, so this returns only once the restaurant is serving.
			h.apply(ctx, &t, true)
			if fresh := h.tenantByID(ctx, t.ID); fresh != nil &&
				fresh.ProvisionStatus == "failed" {
				item.Status = "failed"
				item.Note = fresh.ProvisionError
			} else {
				item.Status = "updated"
			}
		}

		if item.Status == "failed" {
			run.Failed++
			consecutive++
		} else {
			if item.Status == "updated" {
				run.Updated++
			}
			run.Done++
			consecutive = 0
		}
		run.Items = append(run.Items, item)
		run.Current = t.Slug
		_ = h.saveRollout(ctx, run)

		if consecutive >= rolloutBreaker {
			// Not this tenant's problem. Stopping leaves the rest on an image
			// that works, which is the better half of a bad situation.
			h.endRollout(ctx, run, "aborted",
				"ketma-ket 3 ta xato — image buzuq bo'lishi mumkin, qolgani to'xtatildi")
			return
		}
		if ctx.Err() != nil {
			h.endRollout(ctx, run, "aborted", "vaqt tugadi")
			return
		}
	}
	status := "done"
	note := ""
	if run.Failed > 0 {
		note = "ba'zi mijozlar yangilanmadi"
	}
	h.endRollout(ctx, run, status, note)
}

func (h *Handler) tenantByID(ctx context.Context, id any) *models.Tenant {
	var t models.Tenant
	if err := h.Store.Tenants.FindOne(ctx, bson.M{"_id": id}).Decode(&t); err != nil {
		return nil
	}
	return &t
}

// saveRollout writes progress to the singleton document the console polls.
func (h *Handler) saveRollout(ctx context.Context, run *models.Rollout) error {
	_, err := h.Store.Rollouts.UpdateOne(ctx, bson.M{"_id": models.RolloutDocID},
		bson.M{"$set": run}, options.Update().SetUpsert(true))
	return err
}

func (h *Handler) endRollout(ctx context.Context, run *models.Rollout, status, note string) {
	run.Status = status
	run.Note = note
	run.Current = ""
	run.FinishedAt = time.Now()
	if err := h.saveRollout(ctx, run); err != nil {
		log.Printf("rollout: natijani yozib bo'lmadi: %v", err)
	}
	log.Printf("rollout %s: %d/%d yangilandi, %d xato %s",
		status, run.Updated, run.Total, run.Failed, note)
}

// RolloutOnBoot rolls stale tenants forward shortly after the control plane
// starts, and is why a deploy reaches customers without anybody pressing
// anything.
//
// Boot is the right trigger because the deploy recreates this container and
// nothing else knows a deploy happened. It is safe to run on *every* start,
// including a host reboot, precisely because "stale" is decided by image id:
// after a reboot every container comes back on the image it had, nothing is
// stale, and this does nothing at all. Only a genuinely new image moves
// anybody.
//
// Deliberately delayed. The control plane must be answering /health first —
// the deploy script waits on it, and a deploy that times out because we were
// busy restarting fifty restaurants would be a worse failure than the one this
// fixes.
func (h *Handler) RolloutOnBoot(ctx context.Context, delay time.Duration) {
	if h.Docker == nil || !h.Cfg.RolloutOnBoot {
		return
	}
	time.Sleep(delay)

	imageID, err := h.Docker.ImageID(ctx)
	if err != nil {
		log.Printf("rollout: image aniqlanmadi: %v", err)
		return
	}
	if !tryStartRollout() {
		return // somebody pressed the button first
	}
	defer finishRollout()

	run := models.Rollout{
		StartedAt: time.Now(),
		StartedBy: "deploy",
		Image:     h.Docker.Image(),
		ImageID:   imageID,
		Status:    "running",
	}
	if err := h.saveRollout(ctx, &run); err != nil {
		log.Printf("rollout: %v", err)
		return
	}
	h.runRollout(ctx, &run)
}

// GetRollout is what the console polls while a rollout runs, and what it shows
// afterwards.
//
// It also answers the question nobody thinks to ask until it matters: **how
// many customers are running old code right now?** That count is computed from
// live container image ids rather than from the last rollout's result, because
// a container recreated by hand — or a tenant provisioned after the rollout
// finished — is not covered by any record of what we did.
func (h *Handler) GetRollout(w http.ResponseWriter, r *http.Request) {
	res := map[string]any{"enabled": h.Docker != nil}

	var run models.Rollout
	if err := h.Store.Rollouts.FindOne(r.Context(),
		bson.M{"_id": models.RolloutDocID}).Decode(&run); err == nil {
		res["last"] = run
	}
	if h.Docker == nil {
		httpx.JSON(w, http.StatusOK, res)
		return
	}

	res["image"] = h.Docker.Image()
	imageID, err := h.Docker.ImageID(r.Context())
	if err != nil {
		res["error"] = err.Error()
		httpx.JSON(w, http.StatusOK, res)
		return
	}
	res["imageId"] = imageID

	cur, err := h.Store.Tenants.Find(r.Context(), bson.M{})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var tenants []models.Tenant
	if err := cur.All(r.Context(), &tenants); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	stale := make([]string, 0)
	current, offline := 0, 0
	for _, t := range tenants {
		if t.Offline() {
			offline++
			continue
		}
		st, err := h.Docker.Status(r.Context(), t.Slug)
		if err != nil || st.ImageID != imageID {
			stale = append(stale, t.Slug)
			continue
		}
		current++
	}
	res["current"] = current
	res["offline"] = offline
	res["stale"] = stale
	res["staleCount"] = len(stale)
	// Whether a rollout is in flight in *this* process. The stored status can
	// say "running" after a restart killed the goroutine, and an operator
	// staring at a progress bar that will never move is worse than one being
	// told to press the button again.
	res["running"] = !rolloutMu.TryLock()
	if !res["running"].(bool) {
		rolloutMu.Unlock()
	}

	httpx.JSON(w, http.StatusOK, res)
}
