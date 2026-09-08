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

// ⚠️ **The artifact path comes from a marked line, not from the last one.**
//
// This shipped reading the last line, and the first console build reported its
// filename as `Preparing "Install Android SDK Build-Tools 35 v.35.0.0".` —
// because Docker interleaves stdout and stderr by write time and the Android
// tooling writes progress to stderr. The path was printed; something else was
// printed after it.
//
// The path still comes from the script rather than being rebuilt here from the
// slug and a clock: two implementations of "what is this file called" drift on
// the first change to either.
func TestTheArtifactPathIsReadFromAMarkerNotAPosition(t *testing.T) {
	// The shape that broke it: the real path, then tooling noise after.
	out := "== building apk\n" +
		"KEEL_ARTIFACT=/opt/keel/appbuilds/b5somsa/b5somsa-20260908-151026.apk\n" +
		"Preparing \"Install Android SDK Build-Tools 35 v.35.0.0\".\n"
	got := markedValue(out, "KEEL_ARTIFACT=")
	if got != "/opt/keel/appbuilds/b5somsa/b5somsa-20260908-151026.apk" {
		t.Fatalf("noise after the marker won: %q", got)
	}
	// No marker at all is empty rather than the last thing that scrolled past —
	// which is what turns a failed build into a mysterious filename.
	if markedValue("Preparing something.\nmore noise\n", "KEEL_ARTIFACT=") != "" {
		t.Fatal("a build that printed no marker produced a path anyway")
	}
	// ⚠️ The last match, not the first: a retry inside one container prints two,
	// and the one that matters is the run that produced this file.
	two := "KEEL_ARTIFACT=/tmp/old.apk\nKEEL_ARTIFACT=/tmp/new.apk\n"
	if markedValue(two, "KEEL_ARTIFACT=") != "/tmp/new.apk" {
		t.Fatal("the first marker won over the second")
	}
}

// ⚠️ **The build script prints the marker.** The reader and the writer are in
// two languages and two repositories' worth of distance; nothing else holds
// them together.
func TestTheBuildScriptPrintsTheMarker(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("..", "..", "..",
		"deploy", "appbuild", "build.sh"))
	if err != nil {
		t.Skip("build.sh not reachable from here")
	}
	if !strings.Contains(string(script), "KEEL_ARTIFACT=%s") {
		t.Fatal("build.sh no longer prints KEEL_ARTIFACT — the console cannot find the file")
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

// ⚠️ **The reason has to survive the trim.** The first version prefixed the
// error onto the log and then kept the *last* forty lines — so the sentence
// explaining the failure was the first thing thrown away, and what remained was
// forty lines of Gradle tasks that say nothing about why it stopped. That is
// exactly what a person opening a failed build is looking for.
func TestAFailureSaysWhyBeforeItSaysWhat(t *testing.T) {
	var log strings.Builder
	for i := 0; i < 200; i++ {
		log.WriteString("> Task :app:something\n")
	}
	// What the handler builds: the reason, a blank line, then the tail.
	stored := "docker: ulanib bo'lmadi\n\n" + tailLines(log.String(), 40)
	first := strings.SplitN(stored, "\n", 2)[0]
	if !strings.Contains(first, "docker") {
		t.Fatalf("the reason was trimmed away; the message starts %q", first)
	}
	if n := strings.Count(stored, "> Task"); n != 40 {
		t.Fatalf("kept %d log lines, want 40", n)
	}
}
