package handlers

// What the platform calls itself, in one place per binary.
//
// ⚠️ **A constant in the binary, not a value read from anywhere.** The version has to
// agree with the code that is running, and every other source can disagree with it: a
// file on disk survives a rollback, an environment variable is set by whoever last
// edited the compose file, and a git tag is a fact about a repository rather than about
// the container answering the request.
//
// ⚠️ **And every part of Keel says the same number.** The landing, the console, a
// restaurant's dashboard and the Windows till are one product to the person paying for
// it — "which version?" has to have one answer, or a support call starts by
// establishing which of four numbers is being discussed.
//
// **Change it with `scripts/set-version.sh vX.Y.Z`**, which writes all five places at
// once. version_test.go then fails the build if one of them drifts anyway — the script
// is the convenient path and the test is the one that has to be true.
//
// `Stage` is separate from the number and says what the number *means*. Empty means the
// number stands on its own.
const (
	Version = "v0.2.0"
	Stage   = ""
)
