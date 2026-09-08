package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"keel-control/internal/models"
)

// ⚠️ **The application id is derived from the slug in two places**, and this
// test is the only thing holding them together: `build.sh` computes the id that
// actually ships, and the console computes one only to display. They must agree
// or the panel names an app that does not exist — on the screen somebody reads
// before uploading to Play.
func TestTheConsoleSpellsTheApplicationIdTheWayTheBuildDoes(t *testing.T) {
	cases := map[string]string{
		"b5somsa":     "b5somsa",
		"B5-Somsa":    "b5somsa",
		"navvat_uz":   "navvatuz",
		"osh.markazi": "oshmarkazi",
	}
	for slug, want := range cases {
		if got := slugID(slug); got != want {
			t.Fatalf("slugID(%q) = %q, want %q", slug, got, want)
		}
	}
	// The shell does `tr -cd '[:alnum:]' | tr '[:upper:]' '[:lower:]'`, which is
	// what the loop above mirrors. If one of them changes, this fails.
	script, err := os.ReadFile(filepath.Join("..", "..", "..",
		"deploy", "appbuild", "build.sh"))
	if err == nil && !strings.Contains(string(script), "tr -cd '[:alnum:]'") {
		t.Fatal("build.sh no longer derives the id this way — slugID has drifted")
	}
}

// ⚠️ **Only a ready build has a file.** "There is a file" and "there was one"
// are what the console draws a button from, and a download offered for a build
// whose artifact is gone answers with an error on a button that looked ordinary.
func TestOnlyAReadyBuildIsDownloadable(t *testing.T) {
	ready := models.AppBuild{Status: models.AppReady, Path: "/tmp/x.apk"}
	if !ready.Downloadable() {
		t.Fatal("a ready build with a path is not downloadable")
	}
	// Taken: the file was deleted the moment it went out.
	taken := models.AppBuild{Status: models.AppTaken, Path: ""}
	if taken.Downloadable() {
		t.Fatal("a build whose file was deleted still offers a download")
	}
	// ⚠️ Ready but pathless is the state a failed cleanup leaves behind, and it
	// must not be offered either: the row says yes and the disk says no.
	if (models.AppBuild{Status: models.AppReady}).Downloadable() {
		t.Fatal("a ready build with no path offers a download")
	}
	for _, s := range []string{models.AppQueued, models.AppBuilding, models.AppFailed} {
		if (models.AppBuild{Status: s, Path: "/tmp/x.apk"}).Downloadable() {
			t.Fatalf("an unfinished build (%s) offers a download", s)
		}
	}
}

// ⚠️ **The artifact path is read off the script's last line, never rebuilt from
// the slug and a clock.** Two implementations of "what is this file called"
// drift on the first change to either — and the drift shows up as a build that
// succeeded and a download that 404s.
func TestTheArtifactPathComesFromTheScript(t *testing.T) {
	out := "== building apk\n> Task :app:assembleRelease\n\n== done: /opt/keel/appbuilds/b5somsa/b5somsa-20260908-151026.apk\n/opt/keel/appbuilds/b5somsa/b5somsa-20260908-151026.apk\n"
	if got := lastLine(out); got != "/opt/keel/appbuilds/b5somsa/b5somsa-20260908-151026.apk" {
		t.Fatalf("lastLine gave %q", got)
	}
	// Trailing blank lines are what a shell actually produces.
	if lastLine("/tmp/a.apk\n\n\n") != "/tmp/a.apk" {
		t.Fatal("trailing blank lines defeat lastLine")
	}
	if lastLine("   \n") != "" {
		t.Fatal("an empty output should give an empty path, not whitespace")
	}
}

// ⚠️ **A failed build keeps the end of its output, not the whole of it.** Gradle
// prints thousands of lines; what somebody reading a failure needs is the last
// forty, and the rest is a megabyte stored per failure forever.
func TestOnlyTheTailOfAFailureIsKept(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 500; i++ {
		b.WriteString("line\n")
	}
	got := tailLines(b.String(), 40)
	if n := strings.Count(got, "\n") + 1; n != 40 {
		t.Fatalf("kept %d lines, want 40", n)
	}
	// Shorter than the limit is kept whole.
	if tailLines("one\ntwo", 40) != "one\ntwo" {
		t.Fatal("a short output was trimmed")
	}
}
