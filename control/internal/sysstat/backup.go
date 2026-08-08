package sysstat

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Backup is what last night's backup run left behind.
//
// ⚠️ **Read from disk on every request, never stored.** A saved "backups are
// enabled" flag is true from the second it is written and says nothing about
// last night — and every way this actually fails (cron file removed, disk full,
// the mongo container renamed after a compose change, the script replaced by a
// broken edit) leaves that flag perfectly intact while the copies stop. The
// only honest signal is a manifest with a date on it, so that is what is read.
//
// The same lesson as `attention: "down"` in the tenant list, arriving from the
// other side: a stored flag goes stale the moment the clock moves past it, a
// date does not.
type Backup struct {
	// Whether a manifest was found at all. False on a machine where backups
	// were never installed — which is a normal state for a laptop and an
	// alarming one for the production box, and the console says which.
	Present bool `json:"present"`
	// The night this copy was taken, "YYYY-MM-DD".
	Date string `json:"date,omitempty"`
	// When the run finished. Kept separate from Date because a run that starts
	// at 03:30 and finishes at 06:00 is a run worth looking at.
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	// How old the newest copy is, in hours. The number the console actually
	// judges: anything past ~36 hours means a night was missed.
	AgeHours float64 `json:"ageHours"`
	// Databases and uploads archives in that copy, and their total size.
	Files int   `json:"files"`
	Bytes int64 `json:"bytes"`
	// How many dumps the run itself reported as failed. A copy that exists but
	// is missing three restaurants is not a good backup, and the manifest is
	// the only place that difference is recorded.
	Failures int `json:"failures"`
}

// ReadBackup summarises the newest backup under root.
//
// Never returns an error: this is one line on a dashboard, and a missing
// directory is an answer ("no backups here"), not a failure that should blank
// the rest of the page.
func ReadBackup(root string) Backup {
	var b Backup
	if strings.TrimSpace(root) == "" {
		return b
	}
	dir := filepath.Join(root, "latest")
	f, err := os.Open(filepath.Join(dir, "MANIFEST"))
	if err != nil {
		return b
	}
	defer f.Close()
	b.Present = true

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "---" {
			break
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "date":
			b.Date = val
		case "finishedAt":
			if at, err := time.Parse(time.RFC3339, val); err == nil {
				local := at.In(time.Local)
				b.FinishedAt = &local
				b.AgeHours = time.Since(local).Hours()
			}
		case "failures":
			b.Failures, _ = strconv.Atoi(val)
		}
	}

	// Sizes come from the files themselves rather than from the manifest: the
	// manifest lists what the run meant to write, and the question here is what
	// is actually on the disk now.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return b
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".gz") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		b.Files++
		b.Bytes += info.Size()
	}
	return b
}
