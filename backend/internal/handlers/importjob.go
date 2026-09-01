package handlers

// ---- Long imports, and why they cannot live inside a request ----
//
// ⚠️ **This is what the 502 was.** Applying an import downloads a photograph
// per dish onto the restaurant's disk: ninety dishes is ninety requests to
// somebody else's server, which is minutes. The router gives a handler thirty
// seconds (`chimw.Timeout`) and the edge gives it less, so the connection was
// cut long before the work finished — and the owner saw a gateway error while
// the import was, in fact, still running and still writing dishes.
//
// That is the worst shape a failure can have: it looks like nothing happened
// and something did. Pressing the button again would then import the whole menu
// a second time.
//
// So the work is started, the request returns immediately with a job id, and
// the panel asks how it is going. The progress bar the owner sees is the real
// count of what has been written, not an animation.
//
// ⚠️ **In memory, and it is forgotten on restart.** A job is a progress
// indicator, not a record: what was actually imported is the menu, which is in
// the database. A restart mid-import leaves a half-imported menu either way,
// and the honest recovery is the duplicate check on the next run — not a job
// queue pretending the work can be resumed.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"restaurant-backend/internal/httpx"
)

// How long a finished job stays readable. ⚠️ Long enough that a browser which
// lost its connection mid-import can come back and read the result, short
// enough that nothing accumulates.
const importJobTTL = 30 * time.Minute

// ImportJob is the progress of one running import.
type ImportJob struct {
	ID    string `json:"id"`
	Stage string `json:"stage"`
	// Where it has got to. ⚠️ Both numbers, because a bar with no total is a
	// spinner with extra steps — and the owner wants to know whether to wait.
	Done  int `json:"done"`
	Total int `json:"total"`

	Finished bool `json:"finished"`
	// The owner's sentence when it went wrong, never a transport error.
	Error string `json:"error,omitempty"`
	// Whatever the finished job produced, passed through to the panel
	// unchanged — the same shape the synchronous version used to return.
	Result map[string]any `json:"result,omitempty"`

	startedAt time.Time
}

// Percent is the number on the bar.
func (j *ImportJob) Percent() int {
	if j.Total <= 0 {
		return 0
	}
	p := j.Done * 100 / j.Total
	if p > 100 {
		return 100
	}
	return p
}

type importJobs struct {
	mu   sync.Mutex
	jobs map[string]*ImportJob
}

var jobs = &importJobs{jobs: map[string]*ImportJob{}}

func (s *importJobs) start(stage string, total int) *ImportJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Swept here rather than on a timer: this map only grows when an import
	// starts, so the moment to clean it is the moment one does.
	for id, j := range s.jobs {
		if time.Since(j.startedAt) > importJobTTL {
			delete(s.jobs, id)
		}
	}
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	j := &ImportJob{
		ID: hex.EncodeToString(b), Stage: stage, Total: total,
		startedAt: time.Now(),
	}
	s.jobs[j.ID] = j
	return j
}

func (s *importJobs) get(id string) *ImportJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := s.jobs[id]
	if j == nil {
		return nil
	}
	// Copied out under the lock: the caller serialises it while the worker is
	// still writing to it, and a JSON encoder reading a struct being mutated is
	// a data race that shows up as a wrong number once a week.
	out := *j
	return &out
}

func (s *importJobs) update(id string, fn func(*ImportJob)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j := s.jobs[id]; j != nil {
		fn(j)
	}
}

// AdminImportJob reports how a running import is going.
func (h *Handler) AdminImportJob(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	j := jobs.get(chi.URLParam(r, "id"))
	if j == nil {
		// ⚠️ Not a 404. A job that has aged out or was lost to a restart is not
		// a missing page — the import it belonged to may well have finished —
		// and the panel needs a shape it can render rather than an error.
		httpx.JSON(w, http.StatusOK, map[string]any{
			"finished": true, "expired": true,
		})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"id": j.ID, "stage": j.Stage, "done": j.Done, "total": j.Total,
		"percent": j.Percent(), "finished": j.Finished,
		"error": httpx.T(w, j.Error), "result": j.Result,
	})
}

// runImportJob does the work off the request, and cannot be cancelled by the
// browser going away.
//
// ⚠️ `context.WithoutCancel`: the request's context dies the moment the id is
// returned, and an import that stopped because the owner closed the tab would
// leave a menu half-written with nothing to say so.
func runImportJob(ctx context.Context, j *ImportJob, work func(context.Context) (map[string]any, error)) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Minute)
	go func() {
		defer cancel()
		res, err := work(ctx)
		jobs.update(j.ID, func(j *ImportJob) {
			j.Finished = true
			if err != nil {
				j.Error = err.Error()
				return
			}
			j.Result = res
		})
	}()
}
