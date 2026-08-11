package sysstat

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeManifest lays out a backup directory the way keel-backup leaves one,
// with `latest` pointing at it.
func writeManifest(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "2026-08-11")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "MANIFEST"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "t_osh.archive.gz"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dir, filepath.Join(root, "latest")); err != nil {
		t.Fatal(err)
	}
	return root
}

func manifest(finished time.Time) string {
	return "date=2026-08-11\nfinishedAt=" + finished.Format(time.RFC3339) + "\nfailures=0\n---\n"
}

// Last night's copy is not an alarm, and an alarm that is always on is not read.
func TestBackupFreshIsNotStale(t *testing.T) {
	b := ReadBackup(writeManifest(t, manifest(time.Now().Add(-6*time.Hour))))
	if !b.Present {
		t.Fatal("manifest was not found")
	}
	if b.Stale {
		t.Errorf("a six-hour-old copy is last night's copy, got stale=true")
	}
	if b.Files != 1 {
		t.Errorf("files = %d, want 1", b.Files)
	}
}

// A run that finishes late is still a run: the threshold has to clear 24 hours
// or every slow night raises an alarm and the alarm stops meaning anything.
func TestBackupLateRunIsNotStale(t *testing.T) {
	b := ReadBackup(writeManifest(t, manifest(time.Now().Add(-30*time.Hour))))
	if b.Stale {
		t.Errorf("30 hours is a late run, not a missed night")
	}
}

// The failure this whole file exists for: copies stopped and nothing else
// changed. Three nights passed that way in August 2026.
func TestBackupMissedNightIsStale(t *testing.T) {
	b := ReadBackup(writeManifest(t, manifest(time.Now().Add(-40*time.Hour))))
	if !b.Stale {
		t.Errorf("40 hours means last night produced nothing, got stale=false")
	}
}

// ⚠️ A manifest with no timestamp must never read as fresh: the zero value of a
// time is not "just now", but every arithmetic on it says so.
func TestBackupWithoutTimestampIsStale(t *testing.T) {
	b := ReadBackup(writeManifest(t, "date=2026-08-11\nfailures=0\n---\n"))
	if !b.Present {
		t.Fatal("manifest was not found")
	}
	if !b.Stale {
		t.Errorf("a manifest that never said when it finished cannot be shown as fresh")
	}
}

// No backups at all is an answer, not an error — normal on a laptop, alarming
// on the production box, and the console says which.
func TestBackupAbsent(t *testing.T) {
	if b := ReadBackup(t.TempDir()); b.Present {
		t.Errorf("present = true with no manifest on disk")
	}
	if b := ReadBackup(""); b.Present {
		t.Errorf("present = true with no backup path configured")
	}
}
