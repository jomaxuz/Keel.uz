package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ⚠️ **A job that is dropped reports nothing, and that is what made this
// expensive.** The server had already handed the work out and marked it taken,
// so the queue looked busy; the agent logged neither a failure nor a success,
// because it never reached either line. What a restaurant saw was a printer
// reachable from that very machine, a job in the queue, no paper, and an empty
// log — four facts that agree with each other and point nowhere.

func serving(t *testing.T, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestAPrintJobIsAJob(t *testing.T) {
	// The exact shape the server sends for a receipt: a kind, an address and
	// bytes — and no fiscal call, because printing is not a filing.
	base := serving(t, `{
		"kind": "print",
		"number": "K-12",
		"print": {"id":"abc","target":"tcp://192.168.100.28:9100","name":"Oshxona","payload":"AA=="}
	}`)

	j, ok, err := next(context.Background(), http.DefaultClient, base, "t")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	// ⚠️ This is the whole bug in one assertion. It used to be false, because
	// "is there a job" was answered by looking at the *fiscal* job's URL.
	if !ok {
		t.Fatal("a print job came back as nothing to do")
	}
	if j.Kind != kindPrint || j.Print == nil {
		t.Fatalf("job = %+v", j)
	}
	if j.Print.Target != "tcp://192.168.100.28:9100" {
		t.Fatalf("target = %q", j.Print.Target)
	}
}

func TestAFilingIsStillAJob(t *testing.T) {
	base := serving(t, `{
		"kind": "filing",
		"number": "K-13",
		"job": {"url":"http://127.0.0.1:9999/api","method":"POST"}
	}`)

	j, ok, err := next(context.Background(), http.DefaultClient, base, "t")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if !ok || j.Job.URL == "" {
		t.Fatalf("a filing was dropped: ok=%v job=%+v", ok, j.Job)
	}
}

func TestAnEmptyAnswerIsNothingToDo(t *testing.T) {
	// ⚠️ The ordinary case, and it must stay cheap: a restaurant with no sales
	// asks this several times a minute all afternoon.
	base := serving(t, `{}`)
	_, ok, err := next(context.Background(), http.DefaultClient, base, "t")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if ok {
		t.Fatal("an empty answer was treated as work")
	}
}
