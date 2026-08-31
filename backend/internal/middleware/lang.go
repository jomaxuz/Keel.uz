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

// langOf reads the cookie first and the browser's own preference second.
//
// ⚠️ **The cookie wins**, and the order matters for exactly one person: a
// cashier whose Windows is Russian and who set the till to Uzbek. What they
// chose beats what their machine was installed with — the same precedence the
// site itself uses.
func langOf(r *http.Request) string {
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
