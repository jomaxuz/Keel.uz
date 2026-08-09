package handlers

import (
	"net/http"

	"keel-control/internal/httpx"
	"keel-control/internal/sysstat"
)

// System is the server's own vital signs, for the console.
//
// One VPS runs the control plane, the edge, Mongo and every tenant container,
// so "how much room is left" is a single question with a single answer — and
// it is the one that decides when the next customer stops being sellable.
//
// Disk bites first and most quietly: uploads only ever grow, every deploy
// leaves another image behind, and a full disk stops Mongo writing before
// anybody notices a graph. Which is why Docker's own usage is reported beside
// the filesystem's — on this box they are mostly the same thing, and
// `docker system prune` is the fix nobody remembers until it is too late.
//
// Read live rather than sampled: these numbers are only ever looked at by a
// person, and a cached CPU figure is worse than no CPU figure.
func (h *Handler) System(w http.ResponseWriter, r *http.Request) {
	res := map[string]any{
		"host": sysstat.Read(h.Cfg.DiskPath),
		// Beside the disk figures on purpose: the two failures they describe
		// arrive together. A disk that fills stops the backup first and the
		// databases second, and by then the copy that would have fixed it is
		// the one that did not get written.
		"backup": sysstat.ReadBackup(h.Cfg.BackupPath),
	}
	if h.Docker != nil {
		if df, err := h.Docker.DiskUsage(r.Context()); err == nil {
			res["docker"] = df
		}
	}
	httpx.JSON(w, http.StatusOK, res)
}

// PruneDocker frees the space the dashboard has been reporting as reclaimable.
//
// ⚠️ **Deliberately not `prune -a`, and never volumes.** The figure beside this
// button is images a rebuild orphaned, stopped containers and build cache. `-a`
// would also delete the previous tenant image — the only thing a rollback has to
// roll back to — and `--volumes` would delete every customer's photographs. See
// provision.Client.Prune.
//
// Reports what was actually freed rather than "done": the number is the whole
// reason somebody pressed it, and a button that says nothing invites a second press.
func (h *Handler) PruneDocker(w http.ResponseWriter, r *http.Request) {
	if h.Docker == nil {
		httpx.Error(w, http.StatusBadRequest,
			"bu serverda konteynerlar boshqarilmaydi (DOCKER_SOCKET yo'q)")
		return
	}
	freed, err := h.Docker.Prune(r.Context())
	if err != nil && freed == 0 {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	res := map[string]any{"freed": freed}
	if err != nil {
		// Partly done: one of the three calls failed and the other two freed
		// something. Saying so beats both "ok" and an error, because the number is
		// real and the failure is worth seeing.
		res["warning"] = err.Error()
	}
	// The fresh figures, so the panel does not have to guess what is left.
	if du, e := h.Docker.DiskUsage(r.Context()); e == nil {
		res["docker"] = du
	}
	httpx.JSON(w, http.StatusOK, res)
}
