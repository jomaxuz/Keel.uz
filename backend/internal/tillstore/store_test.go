package tillstore

import (
	"path/filepath"
	"testing"
)

// ⚠️ **What is tested here is durability, and durability is the one property a
// unit test cannot fully reach**: a real power cut is not something a test can
// stage. So the test asserts the settings that decide the answer — the journal
// mode and the sync level — beside the behaviour built on them. Getting those
// two pragmas wrong is silent: everything works, every day, until the evening
// the lights go out.

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "till.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestDurabilityPragmasAreWhatWasAskedFor(t *testing.T) {
	s := open(t)
	var mode string
	if err := s.db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
	var sync int
	if err := s.db.QueryRow(`PRAGMA synchronous`).Scan(&sync); err != nil {
		t.Fatalf("synchronous: %v", err)
	}
	// 2 is FULL. ⚠️ 1 is NORMAL, which is WAL's default and loses the last
	// transactions on a power cut — the database survives and the sales do
	// not, which is the outcome this whole file exists to refuse.
	if sync != 2 {
		t.Fatalf("synchronous = %d, want 2 (FULL)", sync)
	}
}

func TestARecordSurvivesTheProcessThatWroteIt(t *testing.T) {
	// The power cut, as closely as a test can stand in for it: the handle is
	// gone and the file is opened again by something new.
	path := filepath.Join(t.TempDir(), "till.db")
	first, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := first.Put("localChecks", "c1", `{"clientId":"c1","total":42000}`); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	second, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer second.Close()
	got, err := second.All("localChecks")
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(got) != 1 || got[0] != `{"clientId":"c1","total":42000}` {
		t.Fatalf("after reopen = %v", got)
	}
}

func TestOneKeyIsOneRecordHoweverManyTimesItIsWritten(t *testing.T) {
	// ⚠️ The screen saves the whole check on every keystroke — a dish added, a
	// quantity changed. Appending instead of replacing would turn one table
	// into forty checks, and the sync would send every one of them.
	s := open(t)
	for _, v := range []string{`{"n":1}`, `{"n":2}`, `{"n":3}`} {
		if err := s.Put("localChecks", "c1", v); err != nil {
			t.Fatalf("put: %v", err)
		}
	}
	got, err := s.All("localChecks")
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(got) != 1 || got[0] != `{"n":3}` {
		t.Fatalf("got %v, want the last write only", got)
	}
}

func TestStoresDoNotSeeEachOther(t *testing.T) {
	// Two queues share one file and must not share a record: draining the
	// pending sales would otherwise take the open tables with it.
	s := open(t)
	if err := s.Put("localChecks", "c1", `{"a":1}`); err != nil {
		t.Fatal(err)
	}
	if err := s.Put("pendingSales", "c1", `{"b":2}`); err != nil {
		t.Fatal(err)
	}
	checks, _ := s.All("localChecks")
	sales, _ := s.All("pendingSales")
	if len(checks) != 1 || checks[0] != `{"a":1}` {
		t.Fatalf("checks = %v", checks)
	}
	if len(sales) != 1 || sales[0] != `{"b":2}` {
		t.Fatalf("sales = %v", sales)
	}
}

func TestAnEmptyStoreIsAnEmptyListAndNotNothing(t *testing.T) {
	// ⚠️ The Go-to-JavaScript rule this codebase has been bitten by twice: a
	// nil slice marshals to `null`, and the till's own screen then reads
	// `.length` of it.
	s := open(t)
	got, err := s.All("localChecks")
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if got == nil {
		t.Fatal("empty store returned nil, which reaches the screen as null")
	}
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestARecordWithNoIDIsRefused(t *testing.T) {
	// The id is the idempotency on both sides of the wire. Stored under a
	// made-up key it would sync as a second dinner on every retry.
	if err := open(t).Put("localChecks", "", `{"total":42000}`); err != ErrNoKey {
		t.Fatalf("err = %v, want ErrNoKey", err)
	}
}

func TestRemovingWhatIsNotThereIsNotAnError(t *testing.T) {
	// The caller is a queue draining itself; a sale removed twice is a sale
	// that arrived.
	if err := open(t).Remove("pendingSales", "never-existed"); err != nil {
		t.Fatalf("remove: %v", err)
	}
}

func TestClearEmptiesTheFileRatherThanReplacingIt(t *testing.T) {
	// ⚠️ The file is opened once for the life of the process. Deleting it
	// underneath that handle leaves writes going somewhere nothing will read,
	// which looks exactly like a till that is saving.
	s := open(t)
	if err := s.Put("localChecks", "c1", `{"a":1}`); err != nil {
		t.Fatal(err)
	}
	if err := s.Clear(); err != nil {
		t.Fatalf("clear: %v", err)
	}
	got, _ := s.All("localChecks")
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	// Still usable afterwards, which is what "emptied" has to mean.
	if err := s.Put("localChecks", "c2", `{"a":2}`); err != nil {
		t.Fatalf("put after clear: %v", err)
	}
	got, _ = s.All("localChecks")
	if len(got) != 1 {
		t.Fatalf("got %v", got)
	}
}
