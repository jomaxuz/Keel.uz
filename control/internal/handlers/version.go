package handlers

// What the platform calls itself, in one place.
//
// ⚠️ **A constant in the binary, not a value read from anywhere.** The version has to
// agree with the code that is running, and every other source can disagree with it: a
// file on disk survives a rollback, an environment variable is set by whoever last
// edited the compose file, and a git tag is a fact about a repository rather than about
// the container answering the request.
//
// `Stage` is separate from the number and says what the number *means*. "v0.1" alone
// invites a guest to guess, and the guess people make about a platform holding their
// restaurant's orders is the generous one — so the honest word is printed beside it
// while it is still true.
const (
	Version = "v0.1"
	Stage   = "test"
)
