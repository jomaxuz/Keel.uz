package handlers

import (
	"net/http"

	"restaurant-backend/internal/middleware"
)

// The spreadsheet speaks the language the panel is being read in.
//
// A report is not an internal artefact: it is downloaded, renamed, forwarded
// and read outside the panel — usually by somebody who has never seen the
// panel at all. An owner working in a Russian dashboard who exports a month
// and opens a sheet headed in Uzbek has to translate his own figures before he
// can send them on, and the column he guesses wrong is a financial mistake
// with our name on it.
//
// The screen has the same problem for the same reason: the period note under
// every report is written here, not in the panel's dictionary, so a Russian
// dashboard was explaining its numbers in Uzbek.
//
// Resolution order is the one the data archive already uses (`exportLang`):
// the panel says so explicitly, the `lang` cookie is the fallback, Uzbek is
// the base.
//
// ⚠️ **The header sits between them because the cookie usually never arrives.**
// The panel and the API are two origins, and a cross-origin `fetch` carries no
// cookies — so every report that did not put `?lang=` in its query was headed
// in Uzbek on a Russian dashboard, and the assistant's cards were written in
// Uzbek for an owner reading Russian. `middleware.LangHeader` is what every
// client now sends; this is the same answer, for the handlers that ask the
// request rather than the writer.
func reportLang(r *http.Request) string {
	return exportLang(
		r.URL.Query().Get("lang"),
		r.Header.Get(middleware.LangHeader),
		cookieValue(r, "lang"),
	)
}

// tr is one phrase in the three languages the panel is read in.
//
// ⚠️ Written at the definition site of each column, note and label rather than
// collected in a central dictionary. A column title and its translations are
// one decision; splitting them means the next column added is the one that
// only ever appears in Uzbek — silently, because a missing translation is not
// an error anywhere.
type tr struct{ uz, ru, en string }

// in picks the translation, falling back to Uzbek. An empty translation falls
// back too: a blank column header is worse than one in the wrong language.
func (t tr) in(lang string) string {
	switch lang {
	case "ru":
		if t.ru != "" {
			return t.ru
		}
	case "en":
		if t.en != "" {
			return t.en
		}
	}
	return t.uz
}

// trTotal is the totals row's label. Every report has one and they must agree:
// a workbook whose sheets end in three different words for "total" reads as
// three different reports.
var trTotal = tr{"Jami", "Итого", "Total"}
