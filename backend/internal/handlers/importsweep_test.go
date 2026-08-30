package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **The one property this file must never lose.** A sweeper that scanned
// `uploads/` and deleted whatever it could not find a reference to would
// eventually delete the restaurant's logo — because "could not find a
// reference" is a claim about how well the hiding places were enumerated, and a
// picture can sit inside a page design's free-form settings map or inside
// custom CSS as `url(/uploads/…)`. One missed hiding place is a live photograph
// gone with nothing to restore it from.
//
// The candidate set is therefore the importer's own record and nothing else.
// This test reads the source because that is where the property lives: the
// moment somebody replaces the Find with a directory listing, it is gone, and
// nothing else in the suite would notice.
func TestTheSweeperOnlyEverLooksAtWhatItDownloaded(t *testing.T) {
	fn := between(t, readSource(t, "importsweep.go"),
		"func (h *Handler) sweepImportAssets", "\n}\n")

	if !strings.Contains(fn, "h.Store.ImportAssets.Find") {
		t.Fatal("the candidate set no longer comes from the importer's own record")
	}
	for _, forbidden := range []string{"os.ReadDir", "filepath.Walk", "filepath.Glob"} {
		if strings.Contains(fn, forbidden) {
			t.Fatalf("the sweeper reads the upload directory (%s) — it would "+
				"eventually delete a photograph the owner uploaded", forbidden)
		}
	}
}

// ⚠️ **A file written a moment ago is not garbage, it is in flight.** Two
// imports can run in two tabs: one downloads a photograph and is still working
// through its list while the other finishes and sweeps. Without the grace the
// second deletes the first's picture, and the dish is saved pointing at a file
// that no longer exists.
func TestRecentDownloadsAreLeftAlone(t *testing.T) {
	fn := between(t, readSource(t, "importsweep.go"),
		"func (h *Handler) sweepImportAssets", "\n}\n")
	if !strings.Contains(fn, "importSweepGrace") {
		t.Fatal("the sweep has no grace period — a download still in flight is deletable")
	}
	if importSweepGrace <= 0 {
		t.Fatalf("importSweepGrace = %v", importSweepGrace)
	}
}

// ⚠️ **Cannot answer the question, do not act on it.** A sweep that deletes
// when the database is unreachable is a sweep that empties the folder on the
// one day the database is unreachable.
func TestAFailedLookupDeletesNothing(t *testing.T) {
	fn := between(t, readSource(t, "importsweep.go"),
		"func (h *Handler) sweepImportAssets", "\n}\n")
	idx := strings.Index(fn, "h.imageIsReferenced")
	if idx < 0 {
		t.Fatal("the reference check is gone")
	}
	rest := fn[idx:]
	remove := strings.Index(rest, "os.Remove")
	guard := strings.Index(rest, "if err != nil")
	if remove < 0 || guard < 0 || guard > remove {
		t.Fatal("a failed reference lookup is not guarded before the delete")
	}
}

// ⚠️ The fields above are not where a design keeps its pictures: `page_design`
// holds them inside a settings map and inside custom CSS, under keys this code
// cannot name. Checking only named fields would answer "unreferenced" for those
// with complete confidence.
func TestTheReferenceCheckReadsWholeDocuments(t *testing.T) {
	fn := between(t, readSource(t, "importsweep.go"), "func scanForName", "\n}\n")
	if !strings.Contains(fn, "cur.Current.String()") {
		t.Fatal("the reference check no longer looks at the whole document — " +
			"a picture inside a design's settings or CSS would be called unused")
	}
}
