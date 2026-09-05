package handlers

import (
	"os"
	"strings"
	"testing"
)

// ⚠️ **Read from the source, because the rule being sealed is a line that is
// easy to put back.** Both of these are one `$inc` in the wrong function, and
// both would look correct in review — the whole defect is *where* the counting
// happens, not what it does.
func blogSource(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("blog.go")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func funcBody(t *testing.T, src, name string) string {
	t.Helper()
	i := strings.Index(src, name)
	if i < 0 {
		t.Fatalf("%q not found — was it renamed?", name)
	}
	rest := src[i:]
	j := strings.Index(rest, "\n}\n")
	if j < 0 {
		t.Fatalf("end of %q not found", name)
	}
	return rest[:j]
}

// ⚠️ **Reading a post is not reading a post.**
//
// The counter used to sit inside the read, and the page fetches itself more
// than once per visit: Next builds the title and description in a pass of its
// own and then draws the words. One reader counted two, and switching language
// counted three. The number was wrong from the first day and wrong in the
// flattering direction — the kind nobody questions.
func TestReadingAPostCountsNothing(t *testing.T) {
	fn := funcBody(t, blogSource(t), "func (h *Handler) BlogRead")
	if strings.Contains(fn, "$inc") {
		t.Fatal("BlogRead counts a view again — one reader will count two, " +
			"and three when they switch language")
	}
}

// ⚠️ **A draft is opened by whoever is writing it, repeatedly.** A counter that
// learned to include unpublished posts would report the author's own afternoon
// as an audience.
func TestOnlyPublishedPostsAreCounted(t *testing.T) {
	fn := funcBody(t, blogSource(t), "func (h *Handler) BlogCountView")
	if !strings.Contains(fn, `"published": true`) {
		t.Fatal("the counter no longer excludes drafts")
	}
	if !strings.Contains(fn, "$inc") {
		t.Fatal("the counter does not count")
	}
}

// ⚠️ **A picture's address has to be one the reader's browser can reach.** The
// control plane serves the bytes under `/internal`, which answers inside the
// docker network and nowhere else — so a post whose pictures pointed there
// rendered every one as a broken box, for every reader, while looking correct
// in the editor.
func TestUploadedPicturesGetAnAddressReadersCanReach(t *testing.T) {
	fn := funcBody(t, blogSource(t), "func (h *Handler) ConsoleBlogUpload")
	if strings.Contains(fn, `"/internal/blog/image/"`) {
		t.Fatal("the upload hands back an address only the docker network can open")
	}
	if !strings.Contains(fn, `"/blog-image/"`) {
		t.Fatal("the upload no longer returns the address the site serves")
	}
}
