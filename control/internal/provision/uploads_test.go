package provision

import (
	"archive/tar"
	"bytes"
	"io"
	"strings"
	"testing"
)

// ⚠️ **The name reaches a shell command and a filesystem path**, so the
// question is not whether it is tidy but whether it can be anything other than
// a name. A slash escapes the tenant's directory, a quote or a semicolon ends
// the argument and starts a command of the caller's choosing, and a leading
// dash turns it into a flag for `mv`.
//
// The handler generates the name and never takes one from a browser — this is
// the second lock, on the function that actually concatenates it.
func TestOnlyAPlainFileNameIsAccepted(t *testing.T) {
	bad := []string{
		"", "..", "../../etc/passwd", "a/b.jpg", "a\\b.jpg",
		"x.jpg; rm -rf /", "x.jpg && echo", "$(id).jpg", "`id`.jpg",
		"-rf.jpg", ".hidden.jpg", "a b.jpg", "x.tar.gz",
		"nodot", strings.Repeat("a", 61) + ".jpg",
	}
	for _, n := range bad {
		if safeUploadName(n) {
			t.Errorf("%q was accepted as a file name", n)
		}
	}
	for _, n := range []string{"d0a1b2c3.jpg", "hero-2.png", "a_b.webp"} {
		if !safeUploadName(n) {
			t.Errorf("%q is an ordinary name and was refused", n)
		}
	}
}

// The archive endpoint takes a tar, and it has to contain exactly the one file
// under exactly the name the copy step will look for — a mismatch is a
// container that exits non-zero with nothing useful to say.
func TestTheArchiveHoldsTheOneFile(t *testing.T) {
	data := []byte("\xff\xd8\xffnot really a jpeg")
	raw, err := singleFileTar("d0a1.jpg", data)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(bytes.NewReader(raw))
	h, err := tr.Next()
	if err != nil {
		t.Fatal(err)
	}
	if h.Name != "d0a1.jpg" {
		t.Errorf("name in the archive is %q", h.Name)
	}
	if h.Size != int64(len(data)) {
		t.Errorf("size %d, want %d", h.Size, len(data))
	}
	got, _ := io.ReadAll(tr)
	if !bytes.Equal(got, data) {
		t.Error("the bytes changed on the way into the archive")
	}
	if _, err := tr.Next(); err != io.EOF {
		t.Error("the archive holds more than the one file")
	}
}
