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
// establishing which of four numbers is being discussed. The root VERSION file is where
// a human changes it; version_test.go fails the build when a declaration drifts from
// it, which is the only thing that keeps four constants honest.
//
// `Stage` is separate from the number and says what the number *means*. Empty means the
// number stands on its own.
const (
	Version = "v0.1.0"
	Stage   = ""
)
