package middleware

// ---- Which language this request is answered in ----
//
// ⚠️ **Carried on the ResponseWriter, not in the context, and that is the whole
// trick.** `httpx.Error(w, status, msg)` takes no request — it is called 1295
// times and changing its signature would touch every one of them. A writer that
// can be asked its language lets the message be translated at the one place it
// is written, with no call site changing at all.
//
// ⚠️ **It has to be the last middleware added.** Chi wraps in order, so the
// handler sees the *innermost* wrapper — added before `chimw.Logger`, this
// wrapper would be buried under Logger's own and the type assertion in
// `httpx.Error` would quietly find nothing. Nothing would break; every message
// would simply stay Uzbek, which is precisely the bug being fixed.

import (
	"net/http"
	"strings"

	"restaurant-backend/internal/i18n"
)

// LangCookie is the cookie the panel, the till and the staff apps already set
// when somebody picks a language. They have no language in their URLs — that
// is by design, they are not pages anybody links to — so the cookie is the
// only thing that knows.
const LangCookie = "lang"

// LangHeader is the same answer, sent explicitly by the caller.
//
// ⚠️ **The cookie does not reach this server, and that is the ordinary case
// rather than the exception.** The panel is served from one origin and the API
// answers on another (`NEXT_PUBLIC_API_URL`), and a cross-origin `fetch` sends
// no cookies — so every sentence written by the server for a panel read in
// Russian arrived in Uzbek, including the morning briefing, whose words are the
// whole feature. Nothing failed and nothing logged: a briefing in the wrong
// language looks exactly like a briefing.
//
// The phone apps have no cookie jar at all and are in the same position, so
// this is one mechanism for every client rather than a patch for the browser.
const LangHeader = "X-Keel-Lang"

// Lang tags the response with the language the caller reads.
//
// ⚠️ **On the writer and nowhere else.** The obvious second home is the request
// context, so a handler can reach it too — but then there are two ways to ask
// the same question and they can disagree. Everything that needs the answer
// already has the writer: `httpx.Error` for messages, `httpx.LangOf` for the
// answers that carry a sentence in a field.
func Lang(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&langWriter{ResponseWriter: w, lang: langOf(r)}, r)
	})
}

// langOf reads what the caller said, then the cookie, then the browser's own
// preference.
//
// ⚠️ **Anything the caller chose wins**, and the order matters for exactly one
// person: a cashier whose Windows is Russian and who set the till to Uzbek.
// What they chose beats what their machine was installed with — the same
// precedence the site itself uses.
//
// ⚠️ The header before the cookie because it is the more specific statement:
// only a client that knows which language its screen is drawn in sends it, and
// a stale cookie from another tab must not outrank the tab that is asking.
func langOf(r *http.Request) string {
	switch h := strings.ToLower(strings.TrimSpace(r.Header.Get(LangHeader))); h {
	case i18n.RU, i18n.EN, i18n.UZ:
		return h
	}
	if c, err := r.Cookie(LangCookie); err == nil {
		switch c.Value {
		case i18n.RU, i18n.EN, i18n.UZ:
			return c.Value
		}
	}
	// A courtesy for callers that never set the cookie — the Telegram mini app
	// on its first request, a browser opening a tracking link. Only the first
	// tag is read: quality values would be a parser for a header this is not
	// making a decision from.
	head := r.Header.Get("Accept-Language")
	if head == "" {
		return i18n.UZ
	}
	first := strings.ToLower(strings.TrimSpace(strings.Split(strings.Split(head, ",")[0], ";")[0]))
	switch {
	case strings.HasPrefix(first, "ru"):
		return i18n.RU
	case strings.HasPrefix(first, "en"):
		return i18n.EN
	}
	return i18n.UZ
}

// langWriter is an http.ResponseWriter that knows who is reading.
type langWriter struct {
	http.ResponseWriter
	lang string
}

// Lang satisfies the interface `httpx.Error` asserts against. It is declared
// nowhere shared on purpose: an interface with one method and one implementer
// does not need a home of its own, and putting it in `httpx` would make this
// package import that one.
func (w *langWriter) Lang() string { return w.lang }

// Flush keeps streaming responses working through the wrapper.
//
// ⚠️ Without it, a wrapped writer silently stops being an http.Flusher and any
// endpoint that streams buffers until it finishes — which looks like a slow
// server, not like a broken wrapper.
func (w *langWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets chi's own helpers (and anything else that looks) reach the
// writer underneath.
func (w *langWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
