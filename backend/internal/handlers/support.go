package handlers

// ---- Support, from the restaurant's side ----
//
// ⚠️ **This server is the panel's only correspondent.** The conversation is
// stored in the control plane, and the panel never talks to it: a page served
// from a customer's own domain must not carry a platform credential, and the
// per-tenant token that reaches the control plane lives here. So everything
// below is a forward — the interesting part is not the proxying, it is the
// socket, which exists so a reply arrives on the owner's screen while they are
// still looking at it.

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"restaurant-backend/internal/httpx"

	"github.com/gorilla/websocket"
)

// ---- Reaching the platform ----

// getControl is the read half of `callControlPath`.
//
// ⚠️ **Its own timeout, passed in, because one of these calls is meant to
// hang.** The wait endpoint holds a request open for half a minute by design;
// sharing the sixty-second client with it would be fine, but sharing a
// *ten*-second one — which is what the first version did — turns the live
// channel into a reconnect loop that never delivers anything.
func (h *Handler) getControl(
	ctx context.Context, path string, query url.Values, wait time.Duration,
) (map[string]any, error) {
	if h.Cfg.ControlURL == "" || h.Cfg.ControlToken == "" {
		return nil, errNoControl
	}
	ctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	u := h.Cfg.ControlURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.Cfg.ControlToken)
	req.Header.Set("X-Keel-Tenant", h.Cfg.TenantSlug)
	res, err := (&http.Client{Timeout: wait}).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	out := map[string]any{}
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	_ = json.Unmarshal(body, &out)
	if res.StatusCode >= 400 {
		msg, _ := out["error"].(string)
		if msg == "" {
			msg = strings.TrimSpace(string(body))
		}
		return nil, errControl(msg)
	}
	return out, nil
}

type errControl string

func (e errControl) Error() string { return string(e) }

const errNoControl = errControl("qo'llab-quvvatlash xizmati sozlanmagan")

// ---- What the panel calls ----

// AdminSupportAsk sends one line to the platform.
func (h *Handler) AdminSupportAsk(w http.ResponseWriter, r *http.Request) {
	admin, err := h.adminUser(r)
	if err != nil {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	var req struct {
		ThreadID string `json:"threadId"`
		Text     string `json:"text"`
		Lang     string `json:"lang"`
		// The help articles the panel's own search ranked for this question.
		//
		// ⚠️ **Sent from the browser rather than duplicated here.** The base
		// lives in the panel's bundle — that is what makes it work with the
		// network down and always describe this build — so a second copy in Go
		// would be two texts that drift. Somebody editing the request can feed
		// the model text of their own and receive it back in their own chat,
		// having told themselves something: the blast radius is one screen,
		// which is why this is acceptable where a posted plan name would not be.
		Articles []supportArticle `json:"articles"`
	}
	if httpx.Decode(r, &req) != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	// ⚠️ **Who is asking is taken from the session, never from the body.** The
	// operator on the other end decides how to answer partly from whether they
	// are talking to the owner or to a cashier, and a name the browser supplies
	// is a name anybody can supply.
	out, err := h.callControlPath(r.Context(), "/internal/support/ask", map[string]any{
		"threadId": req.ThreadID,
		"text":     req.Text,
		"askedBy":  admin.Username,
		"role":     admin.Role,
	})
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	// ⚠️ **The assistant is asked after the reply, not before it.** The owner
	// pressed send and their line has to appear; making them wait several
	// seconds for a model that may decline is a chat that feels broken. The
	// answer arrives on the socket a moment later, the same way an operator's
	// would — one channel, one behaviour on the screen.
	//
	// ⚠️ Its own context, because `r.Context()` is cancelled the instant this
	// response is written. Detached and given a hard ceiling, so a slow engine
	// cannot leave a goroutine per question.
	if id, ok := out["threadId"].(string); ok && id != "" && len(req.Articles) > 0 {
		go h.assistAnswer(id, req.Text, req.Lang, req.Articles)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// assistAnswer asks the platform for a first answer, from our own help text.
//
// ⚠️ **Failures are silent.** An operator is coming either way, and a chat that
// prints "the assistant is over quota" is telling a restaurant about our
// billing in the middle of their problem.
func (h *Handler) assistAnswer(threadID, question, lang string, articles []supportArticle) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	body := map[string]any{
		"threadId": threadID,
		"question": question,
		"lang":     lang,
		"articles": articles,
	}
	if _, err := h.callControlPath(ctx, "/internal/support/assist", body); err != nil {
		log.Printf("support assist: %v", err)
	}
}

// AdminSupportThreads is this restaurant's own history.
func (h *Handler) AdminSupportThreads(w http.ResponseWriter, r *http.Request) {
	if _, err := h.adminUser(r); err != nil {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	out, err := h.getControl(r.Context(), "/internal/support/threads", nil, 20*time.Second)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// AdminSupportThread is one conversation.
func (h *Handler) AdminSupportThread(w http.ResponseWriter, r *http.Request) {
	if _, err := h.adminUser(r); err != nil {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	q := url.Values{"id": {r.URL.Query().Get("id")}}
	out, err := h.getControl(r.Context(), "/internal/support/thread", q, 20*time.Second)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// supportArticle is one help entry, as the panel ranked it.
type supportArticle struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// ---- The live channel ----
//
// ⚠️ **A ticket, because a browser cannot put a header on a WebSocket
// handshake.** The panel authenticates with a Bearer token it holds in memory;
// `new WebSocket(url)` takes no headers, so the only places left are the query
// string and a cookie. Putting the session token in a URL writes a credential
// that is good for a week into every access log and every proxy in front of
// this server. A single-use ticket that expires in half a minute is worth
// nothing by the time it is written down.

const supportTicketTTL = 30 * time.Second

type ticketBook struct {
	mu   sync.Mutex
	live map[string]time.Time
}

var tickets = ticketBook{live: map[string]time.Time{}}

func (b *ticketBook) issue() string {
	id := supportTicketID()
	b.mu.Lock()
	defer b.mu.Unlock()
	// ⚠️ Swept on issue rather than by a timer: this map only grows when
	// somebody opens the widget, so the moment it grows is the moment to tidy
	// it, and there is no goroutine to leak.
	now := time.Now()
	for k, until := range b.live {
		if now.After(until) {
			delete(b.live, k)
		}
	}
	b.live[id] = now.Add(supportTicketTTL)
	return id
}

// spend consumes a ticket. ⚠️ Single use: a ticket that opened one socket must
// not open a second, or a copied URL is a session.
func (b *ticketBook) spend(id string) bool {
	if id == "" {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	until, ok := b.live[id]
	delete(b.live, id)
	return ok && time.Now().Before(until)
}

// supportTicketID is 32 hex characters from the system's CSPRNG.
//
// ⚠️ A ticket is a credential, however short its life. `math/rand` seeded from
// the clock would make every ticket issued in the same millisecond guessable
// from any other — the mistake `orderNumber` documents at length.
func supportTicketID() string {
	b := make([]byte, 16)
	if _, err := crand.Read(b); err != nil {
		panic("support ticket: no randomness: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// AdminSupportTicket mints the one-shot key the socket is opened with, and says
// where to open it.
//
// ⚠️ **The address comes back with the ticket rather than being built in the
// browser, and this is the `rewrites()` trap in a new place.** In production the
// panel and this server share an origin, so `location` would do. In development
// they do not: the panel is Next on :3001 and `/api/*` reaches :8081 through a
// Next rewrite — which proxies HTTP and **does not upgrade a WebSocket**. A
// browser building the URL from its own origin therefore fails in development
// only, with a handshake error nobody reading the panel code would connect to
// the dev proxy.
//
// So the server, which knows its own public address, says it. The same rule
// CLAUDE.md states for the 2GIS key: a value that differs per deployment goes
// in the data, never in the build.
func (h *Handler) AdminSupportTicket(w http.ResponseWriter, r *http.Request) {
	if _, err := h.adminUser(r); err != nil {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"ticket": tickets.issue(),
		"url":    h.supportSocketURL(),
	})
}

// supportSocketURL is this server's own websocket address.
//
// ⚠️ Empty when `PUBLIC_BASE_URL` is not set, and the panel then falls back to
// its own origin — which is right for every deployment where they share one.
// Returning a guess would be worse than returning nothing: a wrong address
// fails the same way a missing one does, and takes longer to work out.
func (h *Handler) supportSocketURL() string {
	base := strings.TrimRight(h.Cfg.PublicBaseURL, "/")
	if base == "" {
		return ""
	}
	if strings.HasPrefix(base, "https://") {
		base = "wss://" + strings.TrimPrefix(base, "https://")
	} else if strings.HasPrefix(base, "http://") {
		base = "ws://" + strings.TrimPrefix(base, "http://")
	}
	return base + "/api/v1/admin/support/ws"
}

// supportUpgrader builds the upgrader with this install's own origin rules.
//
// ⚠️ **Same origin, or an origin the server already trusts for XHR.** A
// permissive check would let any page on the internet open a customer's support
// channel with a ticket it had somehow seen. But "same origin" alone breaks
// development, where the panel is Next on :3001 and this server is :8081 — and
// a rule that only fails in development is a rule somebody weakens at three in
// the morning. `CORS_ORIGINS` is the list an operator has already had to get
// right for every other request; reusing it means there is one answer to "who
// may talk to this server" rather than two that can disagree.
func (h *Handler) supportUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin == "" {
				// A non-browser client, which the single-use ticket already gates.
				return true
			}
			if u, err := url.Parse(origin); err == nil && u.Host == r.Host {
				return true
			}
			for _, allowed := range h.Cfg.CORSOrigins {
				if strings.EqualFold(strings.TrimRight(allowed, "/"), strings.TrimRight(origin, "/")) {
					return true
				}
			}
			return false
		},
	}
}

// How often the socket is pinged when nothing is happening.
//
// ⚠️ **Shorter than any proxy's idle timeout.** Caddy and every load balancer
// in front of this close a connection that has said nothing for a minute, and a
// chat that silently dies after sixty seconds is a chat where the owner types
// into a closed socket and watches nothing happen.
const supportPing = 40 * time.Second

// AdminSupportSocket pushes replies to the panel as they arrive.
func (h *Handler) AdminSupportSocket(w http.ResponseWriter, r *http.Request) {
	if !tickets.spend(r.URL.Query().Get("ticket")) {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	up := h.supportUpgrader()
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// ⚠️ **The reader has to run even though the panel never sends anything.**
	// Without it a close frame from the browser is never read, the connection
	// is held open on this side, and the long poll below keeps a request open
	// against the platform for a tab that was shut ten minutes ago.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go func() {
		<-done
		cancel()
	}()

	since := time.Now().UTC().Format(time.RFC3339Nano)
	pings := time.NewTicker(supportPing)
	defer pings.Stop()
	go func() {
		for {
			select {
			case <-pings.C:
				_ = conn.WriteControl(websocket.PingMessage, nil,
					time.Now().Add(5*time.Second))
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		if ctx.Err() != nil {
			return
		}
		out, err := h.getControl(ctx, "/internal/support/wait",
			url.Values{"since": {since}}, supportWaitTimeout)
		if err != nil {
			// ⚠️ A pause before retrying, and it is the difference between a
			// platform being briefly unreachable and this server hammering it
			// once per failed dial for every open dashboard tab in the country.
			select {
			case <-time.After(5 * time.Second):
				continue
			case <-ctx.Done():
				return
			}
		}
		if now, ok := out["now"].(string); ok && now != "" {
			// ⚠️ The platform's clock, not this container's. Two machines
			// disagreeing by a second means either a message delivered twice or
			// one never delivered at all, and only one of those is noticed.
			since = now
		}
		msgs, _ := out["messages"].([]any)
		if len(msgs) == 0 {
			continue
		}
		if err := conn.WriteJSON(map[string]any{"messages": msgs}); err != nil {
			return
		}
	}
}

// Longer than the platform's own wait, so the request is answered there rather
// than timing out here — a client-side timeout looks like an error and would
// restart the loop every half minute for nothing.
const supportWaitTimeout = 40 * time.Second
