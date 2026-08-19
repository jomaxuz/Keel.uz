package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ⚠️ This is the test for a bug that was live.
//
// reportPrint sent the credential as `Authorization: Bearer` while every other
// call sent `X-Agent-Token`, and the server reads only the second. So the print
// result came back 401, `doneAt` was never written, and the queue re-offered
// the job until it hit MaxPrintTries — every receipt printed three times and
// then reported itself as failed. Nothing looked broken from either end: the
// paper came out, and the panel showed a printing failure for it.
//
// The shape that allowed it is a header chosen at four call sites. Now there is
// one, and this test is what keeps a fifth call site from choosing again.
func TestEveryAgentRequestCarriesBothHeaders(t *testing.T) {
	const token = "relay-token-123"

	seen := make(chan *http.Request, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.Clone(context.Background())
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNoContent) // "nothing to do"
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	ctx := context.Background()
	c := srv.Client()

	if _, _, err := next(ctx, c, srv.URL, token); err != nil {
		t.Fatalf("next: %v", err)
	}
	if _, err := report(ctx, c, srv.URL, token, "order1", reply{Status: 200}); err != nil {
		t.Fatalf("report: %v", err)
	}
	if err := reportCloseDay(ctx, c, srv.URL, token, reply{Status: 200}); err != nil {
		t.Fatalf("reportCloseDay: %v", err)
	}
	if err := reportPrint(ctx, c, srv.URL, token, "job1", nil); err != nil {
		t.Fatalf("reportPrint: %v", err)
	}

	close(seen)
	n := 0
	for r := range seen {
		n++
		if got := r.Header.Get("X-Agent-Token"); got != token {
			t.Errorf("%s %s: X-Agent-Token = %q, want %q — the server reads this one",
				r.Method, r.URL.Path, got, token)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+token {
			t.Errorf("%s %s: Authorization = %q, want the bearer form — a paired "+
				"till's device token is a JWT and only arrives this way",
				r.Method, r.URL.Path, got)
		}
	}
	if n != 4 {
		t.Fatalf("%d requests reached the server, want 4", n)
	}
}
