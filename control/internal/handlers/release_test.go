package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"keel-control/internal/config"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

// ⚠️ **A minor zeroes the patch and a major zeroes both.** Without that, v0.2.7
// raised by a minor gives v0.3.7 — which sorts correctly, reads as a version,
// and claims seven patches that were never made. The mistake is invisible on
// the screen that produced it and permanent in the tags.
func TestRaisingAVersionZeroesWhatIsBelowIt(t *testing.T) {
	cases := []struct {
		cur, part, want string
	}{
		{"v0.2.0", "patch", "v0.2.1"},
		{"v0.2.7", "patch", "v0.2.8"},
		{"v0.2.7", "minor", "v0.3.0"},
		{"v0.2.7", "major", "v1.0.0"},
		{"v1.9.4", "minor", "v1.10.0"},
		// The number the till compares is bare, and the panel offers the
		// console's own, which carries a "v". Both shapes have to parse or the
		// buttons vanish on a deployment where somebody trimmed it.
		{"0.2.0", "patch", "v0.2.1"},
	}
	for _, c := range cases {
		got, ok := NextVersion(c.cur, c.part)
		if !ok || got != c.want {
			t.Errorf("NextVersion(%q, %q) = %q, %v — want %q", c.cur, c.part, got, ok, c.want)
		}
	}
}

// ⚠️ **An unreadable version offers nothing rather than guessing.** This reads
// the constant compiled into this binary, so anything unexpected is a mistake in
// our own source — and an offer built on a misreading is a release that moves
// the platform somewhere nobody chose.
func TestAnUnreadableVersionOffersNoButtons(t *testing.T) {
	for _, bad := range []string{"", "v0.2", "v.0.2.0", "v0.2.0-rc1", "latest", "v0.2.x"} {
		if v, ok := NextVersion(bad, "patch"); ok {
			t.Errorf("NextVersion(%q) offered %q — an unparseable version must offer nothing", bad, v)
		}
		if n := NextVersions(bad); len(n) != 0 {
			t.Errorf("NextVersions(%q) = %v — want none", bad, n)
		}
	}
	// And a button name nobody defined is refused rather than treated as a patch.
	if v, ok := NextVersion("v0.2.0", "hotfix"); ok {
		t.Errorf("an unknown part produced %q — it must refuse", v)
	}
}

// ⚠️ **The till's number is bare and everything else carries a "v".** Comparing
// the two shapes directly reports every till as behind, forever, on a panel
// whose entire job is to report being behind — and the row would be amber on a
// platform where nothing is wrong, which is how a panel stops being read.
func TestTillManifestIsComparedWithoutTheV(t *testing.T) {
	dir := t.TempDir()
	bare := Version
	if len(bare) > 0 && bare[0] == 'v' {
		bare = bare[1:]
	}
	write(t, dir, `{"version":"`+bare+`","url":"https://keel.uz/internal/till/download?file=keel-`+bare+`-installer.exe","sha256":"`+
		"0000000000000000000000000000000000000000000000000000000000000000"+`"}`)

	h := &Handler{Cfg: &config.Config{TillReleaseDir: dir}}
	got := h.tillVersionPart()
	if !got.Match {
		t.Errorf("till manifest %q was read as behind the console's %q", got.Version, Version)
	}
	if got.Unknown {
		t.Error("a manifest that was read is not unknown")
	}
	// The note names the file a monoblock would download — the thing somebody
	// checks against what is actually in the directory.
	if got.Note != "keel-"+bare+"-installer.exe" {
		t.Errorf("note = %q — want the installer's filename", got.Note)
	}
}

// ⚠️ **"Not configured" is not "behind".** A laptop has no till release
// directory and no Docker socket, and reporting those as stale versions puts a
// standing warning on every developer's screen — after which the warning that
// means something is one more amber row.
func TestAnUnaskableTillIsUnknownRatherThanBehind(t *testing.T) {
	for _, h := range []*Handler{
		{Cfg: &config.Config{TillReleaseDir: ""}},
		{Cfg: &config.Config{TillReleaseDir: t.TempDir()}}, // no latest.json
	} {
		got := h.tillVersionPart()
		if !got.Unknown {
			t.Errorf("part %+v — an unreadable manifest must be unknown", got)
		}
		if got.Match {
			t.Error("an unknown part must not claim to match")
		}
	}
	// A manifest that is there but corrupt is also unknown, not behind: the
	// version it would have declared is not a thing we know.
	dir := t.TempDir()
	write(t, dir, `{"version": `)
	h := &Handler{Cfg: &config.Config{TillReleaseDir: dir}}
	if got := h.tillVersionPart(); !got.Unknown {
		t.Errorf("part %+v — a corrupt manifest must be unknown", got)
	}
}

// ⚠️ **The stored release carries no `_id`.** It is written with an upsert on a
// fixed key, and Mongo refuses `_id` inside a `$set` — an empty string is still
// a value. The error names a path nobody wrote, so it reads as a database fault
// rather than as a struct with one field too many; the till grant shipped that
// way once and died on the first sale.
func TestReleaseDocCarriesNoID(t *testing.T) {
	raw, err := bson.Marshal(models.Release{Version: "v0.2.1", Status: "running"})
	if err != nil {
		t.Fatal(err)
	}
	var doc bson.M
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["_id"]; ok {
		t.Error("the release document carries an _id — $set will be refused")
	}
}

// ⚠️ **Releasing is its own permission, and it is owner-only.** It was nearly
// filed under "provision" beside its neighbours on the router: that one
// recreates one customer's container, deliberately chosen, and this one
// rebuilds and replaces all of them from code nobody has looked at since the
// push. A name missing from the map fails closed, so this also checks it is
// there at all.
func TestReleaseIsOwnerOnly(t *testing.T) {
	check, ok := models.Permissions["release"]
	if !ok {
		t.Fatal(`"release" is not in models.Permissions — the gate would refuse everybody`)
	}
	if !check(models.RoleOwner) {
		t.Error("the owner cannot release")
	}
	for _, role := range []string{models.RoleAdmin, models.RoleAgent, models.RoleSupport} {
		if check(role) {
			t.Errorf("%s can release — only the owner may", role)
		}
	}
}

func write(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "latest.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ⚠️ **A 403 from the dispatch is a token permission, and GitHub's sentence
// does not say which one.** "Resource not accessible by personal access token"
// sends the reader to the workflow file, which is never the problem — the
// answer is a fine-grained token carrying **Actions: Read and write** on this
// repository. The hint lives beside GitHub's own words rather than instead of
// them.
func TestAForbiddenDispatchNamesThePermission(t *testing.T) {
	src, err := os.ReadFile("release.go")
	if err != nil {
		t.Fatal(err)
	}
	fn := string(src)
	if !strings.Contains(fn, "http.StatusForbidden") ||
		!strings.Contains(fn, "Actions: Read and write") {
		t.Fatal("a 403 from GitHub no longer names the permission that is " +
			"missing; the next reader will go looking at the workflow file")
	}
}
