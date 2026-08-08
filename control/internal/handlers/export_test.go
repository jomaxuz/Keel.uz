package handlers

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// ⚠️ **A nil slice marshals as `null`, not `[]`.**
//
// This shipped: a grant nobody had downloaded from has no `downloads` field, so
// the decoded slice was nil, the response carried `null`, and the console's
// `grant.downloads.length` replaced the entire customer card with React's error
// screen. It broke on the *first* grant an operator opened — the first moment
// the field could be missing.
//
// The bug survived compilation because an empty slice *was* built and then not
// used in the response: Go saw the variable read by the loop that filled it and
// said nothing. So the rule gets a function and the function gets this test.
func TestDownloadsAlwaysMarshalAsAnArray(t *testing.T) {
	out, err := json.Marshal(map[string]any{"downloads": downloadsJSON(nil)})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "null") {
		t.Fatalf("nil ro'yxat null bo'lib chiqdi: %s", out)
	}
	if !strings.Contains(string(out), `"downloads":[]`) {
		t.Fatalf("bo'sh massiv kutilgan edi: %s", out)
	}

	// And a real history is passed through untouched.
	rows := []exportDownload{{At: time.Now(), By: "yujo", Bytes: 1024, Files: 3}}
	out, err = json.Marshal(map[string]any{"downloads": downloadsJSON(rows)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"by":"yujo"`) {
		t.Fatalf("tarix yo'qoldi: %s", out)
	}
}
