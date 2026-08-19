// Command fiscalagent files receipts with a cash register that nothing else can
// reach.
//
// # Why it exists
//
// The registered virtual cash register is a program on a PC inside the
// restaurant, listening on the local network with no authentication. Our server
// cannot route to it. The till screen in a browser can — but only if three
// browser rules line up, and we control none of them: mixed content, private
// network access, and whether the register answers CORS at all.
//
// This program depends on none of them. It runs on the register's own PC,
// connects **outwards** to our server, asks for receipts to file, files them
// against localhost and reports back. Nothing has to be opened on the
// restaurant's network and nothing has to be configured in a browser.
//
// # Running it
//
//	fiscalagent -server https://restoran.example.uz/api/v1 -token <from the panel>
//
// The register's address defaults to what the panel has stored for the branch,
// which the server puts into each job — this program is told where to call, it
// does not decide.
//
// ⚠️ **It holds no business logic and must never grow any.** The document is
// built on the server from the order; this carries it one hop and brings the
// reply back verbatim. Every temptation to be clever here — retrying on a body
// that "looks like" an error, skipping a filing that seems duplicated — moves a
// piece of tax handling onto an unattended PC in a restaurant, where nobody
// will ever read its log.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// job is what the server asks us to do. Deliberately the same shape the till
// screen receives, because it is the same work.
type job struct {
	// What this is: "filing" for a receipt, "closeDay" for the Z-report that
	// ends the register's tax day.
	//
	// ⚠️ Read from the field rather than inferred from whether an order id is
	// present. The two write different documents at the far end and a guess
	// would be wrong exactly once, on the day it matters most.
	Kind    string `json:"kind"`
	OrderID string `json:"orderId"`
	Number  string `json:"number"`
	Job     call   `json:"job"`
	// A receipt to print. ⚠️ Bytes and an address, nothing else: the layout,
	// the code page and the cut were decided on the server, where they are
	// tested and where somebody reads the log.
	Print *printJob `json:"print,omitempty"`
}

// printJob is one receipt for one printer.
type printJob struct {
	ID      string `json:"id"`
	Target  string `json:"target"`
	Name    string `json:"name"`
	Payload string `json:"payload"` // base64
}

const (
	kindFiling   = "filing"
	kindCloseDay = "closeDay"
	kindPrint    = "print"
)

// call is one HTTP request the server wants made against the register. Opaque
// on purpose — see the note at the top of the file.
type call struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	TimeoutMS int               `json:"timeoutMs"`
}

// reply is what we saw, in the shape the server parses.
//
// ⚠️ The body goes back **whatever the status was**. This register answers
// business refusals with HTTP 500 and a readable sentence, so discarding the
// body on a non-2xx would throw away the only thing that tells anyone what went
// wrong.
type reply struct {
	Status       int    `json:"status"`
	Body         string `json:"body"`
	NetworkError string `json:"networkError,omitempty"`
}

func main() {
	server := flag.String("server", "", "Keel API base, e.g. https://restoran.example.uz/api/v1")
	token := flag.String("token", "", "agent token from the panel (Settings → Fiscal register)")
	verbose := flag.Bool("v", false, "log every filing, not just failures")
	flag.Parse()

	if *server == "" {
		*server = os.Getenv("KEEL_SERVER")
	}
	if *token == "" {
		*token = os.Getenv("KEEL_AGENT_TOKEN")
	}
	if *server == "" || *token == "" {
		log.Fatal("-server va -token kerak (yoki KEEL_SERVER / KEEL_AGENT_TOKEN)")
	}
	base := strings.TrimRight(*server, "/")

	// ⚠️ Ctrl-C and the Windows service stop must actually stop it. Without this
	// the poll's own long timeout keeps the process alive for another half
	// minute after every restart, which during an update looks like two agents
	// running at once.
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("fiskal agent ishga tushdi: %s", base)
	run(ctx, base, *token, *verbose)
}

// run is the whole loop: ask for work, do it, report, repeat.
func run(ctx context.Context, base, token string, verbose bool) {
	// The long poll is held open by the server for ~25s, so the client's own
	// timeout has to be comfortably longer or every quiet minute looks like a
	// network fault.
	poller := &http.Client{Timeout: 60 * time.Second}
	reporter := &http.Client{Timeout: 30 * time.Second}

	// ⚠️ Backoff on failure, and only on failure. A server that is down, a
	// token that is wrong and a restaurant with no sales must not look the same
	// from here: the first two escalate the wait, the third returns "nothing to
	// do" instantly and immediately asks again.
	wait := time.Second
	for ctx.Err() == nil {
		j, ok, err := next(ctx, poller, base, token)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			log.Printf("serverdan ish so'rab bo'lmadi: %v (%s dan keyin qayta)", err, wait)
			sleep(ctx, wait)
			if wait < time.Minute {
				wait *= 2
			}
			continue
		}
		wait = time.Second
		if !ok {
			continue
		}

		// ⚠️ **Printing is handled first and reported separately.** It shares
		// this loop because it shares the reason the loop exists — the printer,
		// like the register, is inside the restaurant and nothing outside can
		// reach it — but a receipt is not a tax document and must not go
		// through the filing path.
		if j.Kind == kindPrint && j.Print != nil {
			perr := printOne(ctx, *j.Print)
			if perr != nil {
				log.Printf("chop etib bo'lmadi (%s): %v", j.Print.Name, perr)
			} else if verbose {
				log.Printf("chek %s: %s ga chiqarildi", j.Number, j.Print.Name)
			}
			if err := reportPrint(ctx, reporter, base, token, j.Print.ID, perr); err != nil {
				log.Printf("chop etish natijasini yuborib bo'lmadi: %v", err)
			}
			continue
		}

		res := file(ctx, j)

		// ⚠️ Ending the day is reported to its own endpoint and gets no
		// follow-up. It is a tax document, not an operation to retry in a loop:
		// the server clears the request whatever the answer was, and a failure
		// is recorded on the cash shift where somebody will read it.
		if j.Kind == kindCloseDay {
			if err := reportCloseDay(ctx, reporter, base, token, res); err != nil {
				log.Printf("kun yakunini yuborib bo'lmadi: %v", err)
			} else if res.NetworkError != "" || res.Status >= 400 {
				log.Printf("kassa kun yakunini qabul qilmadi: %s",
					firstLine(res.Body, res.NetworkError))
			} else {
				log.Print("kassa kuni yopildi (Z-hisobot)")
			}
			continue
		}

		follow, err := report(ctx, reporter, base, token, j.OrderID, res)
		if err != nil {
			// ⚠️ **Not retried here, and that is on purpose.** The sale is still
			// marked pending on the server, so the next request for work offers
			// it again — the filing is retried by being asked for, not by this
			// program remembering it. A queue kept in this process would be lost
			// on every Windows update, and would double-file whatever survived.
			log.Printf("chek %s: natijani yuborib bo'lmadi: %v", j.Number, err)
			continue
		}
		// ⚠️ The server may answer "do this first" — in practice, opening the
		// register's day, which the first sale every morning runs into. Done
		// here and its result deliberately discarded: the sale is still pending
		// on the server, so the next request for work offers it again and it
		// goes through. Acting on the reply rather than retrying blindly is what
		// keeps this from being a tight loop that files nothing.
		if follow != nil {
			log.Printf("kassa smenasi ochilmoqda")
			fj := job{OrderID: j.OrderID, Number: j.Number}
			fj.Job = *follow
			if r := file(ctx, fj); r.NetworkError != "" || r.Status >= 400 {
				log.Printf("smenani ochib bo'lmadi: %s",
					firstLine(r.Body, r.NetworkError))
			}
			continue
		}
		if res.NetworkError != "" || res.Status >= 400 {
			log.Printf("chek %s: kassa qabul qilmadi (HTTP %d) %s",
				j.Number, res.Status, firstLine(res.Body, res.NetworkError))
		} else if verbose {
			log.Printf("chek %s: yuborildi", j.Number)
		}
	}
	log.Print("fiskal agent to'xtadi")
}

// next asks for the next filing. The second result is false when there is
// simply nothing to do, which is the ordinary answer.
func next(ctx context.Context, c *http.Client, base, token string) (job, bool, error) {
	var j job
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/fiscal/agent/job", nil)
	if err != nil {
		return j, false, err
	}
	req.Header.Set("X-Agent-Token", token)

	res, err := c.Do(req)
	if err != nil {
		return j, false, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))

	switch res.StatusCode {
	case http.StatusNoContent:
		return j, false, nil
	case http.StatusOK:
		if err := json.Unmarshal(body, &j); err != nil {
			return j, false, fmt.Errorf("javobni o'qib bo'lmadi: %w", err)
		}
		return j, j.Job.URL != "", nil
	case http.StatusUnauthorized:
		// Named rather than folded into the generic failure: the fix is a person
		// pasting a new token, and no amount of retrying will do it.
		return j, false, fmt.Errorf("agent kaliti qabul qilinmadi — paneldan yangi kalit oling")
	}
	return j, false, fmt.Errorf("server %d: %s", res.StatusCode, firstLine(string(body), ""))
}

// file makes the call to the cash register.
//
// ⚠️ Never returns an error: every outcome is a reply the server can record,
// which is what keeps a failed filing retryable instead of lost. A timeout and
// a refused connection are both reported as themselves, because they send
// somebody to different places.
func file(ctx context.Context, j job) reply {
	timeout := time.Duration(j.Job.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	method := j.Job.Method
	if method == "" {
		method = http.MethodPost
	}
	var body io.Reader
	if method != http.MethodGet && j.Job.Body != "" {
		body = strings.NewReader(j.Job.Body)
	}
	req, err := http.NewRequestWithContext(callCtx, method, j.Job.URL, body)
	if err != nil {
		return reply{NetworkError: "manzil noto'g'ri: " + err.Error()}
	}
	for k, v := range j.Job.Headers {
		req.Header.Set(k, v)
	}

	res, err := (&http.Client{}).Do(req)
	if err != nil {
		if callCtx.Err() == context.DeadlineExceeded {
			return reply{NetworkError: "kassa dasturi vaqtida javob bermadi"}
		}
		return reply{NetworkError: "kassa dasturiga ulanib bo'lmadi: " + err.Error()}
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	return reply{Status: res.StatusCode, Body: string(raw)}
}

// report hands the outcome back.
// report hands the outcome back, and may be told what to do next.
//
// The follow-up exists for one real case: the register refusing because its day
// has not been opened. See the call site.
func report(
	ctx context.Context, c *http.Client, base, token, orderID string, r reply,
) (*call, error) {
	payload, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		base+"/fiscal/agent/job/"+orderID, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Agent-Token", token)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("server %d: %s", res.StatusCode, firstLine(string(body), ""))
	}
	var out struct {
		Next *call `json:"next"`
	}
	// A body we cannot read is not a failure: the filing was recorded, and the
	// follow-up is an optimisation. Erroring here would re-report a result the
	// server already has.
	_ = json.Unmarshal(body, &out)
	return out.Next, nil
}

// reportCloseDay hands back the Z-report.
func reportCloseDay(
	ctx context.Context, c *http.Client, base, token string, r reply,
) error {
	payload, err := json.Marshal(r)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		base+"/fiscal/agent/close-day", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("X-Agent-Token", token)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return fmt.Errorf("server %d: %s", res.StatusCode, firstLine(string(body), ""))
	}
	return nil
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}

// firstLine keeps a log line to one line. A cash register that answers with an
// HTML error page would otherwise put a screenful into a file somebody reads
// months later, looking for the sentence above it.
func firstLine(body, fallback string) string {
	s := strings.TrimSpace(body)
	if s == "" {
		s = fallback
	}
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}
