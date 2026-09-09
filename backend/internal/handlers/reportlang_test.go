package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"restaurant-backend/internal/middleware"
)

func TestReportLangPrefersParamThenHeaderThenCookie(t *testing.T) {
	req := func(query, header, cookie string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/admin/reports/sales"+query, nil)
		if header != "" {
			r.Header.Set(middleware.LangHeader, header)
		}
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: "lang", Value: cookie})
		}
		return r
	}
	cases := []struct {
		name, query, header, cookie, want string
	}{
		{"param wins", "?lang=ru", "uz", "uz", "ru"},
		// ⚠️ **The case this whole tier exists for.** The panel and the API are
		// two origins, so the cookie below is simply never sent — and every
		// handler that reads this (the briefing among them) answered in Uzbek
		// on a Russian dashboard, silently.
		{"the header carries it when no cookie arrives", "", "ru", "", "ru"},
		{"the header outranks the cookie", "", "ru", "uz", "ru"},
		{"cookie is the fallback", "", "", "ru", "ru"},
		{"nothing said is Uzbek", "", "", "", "uz"},
		{"an unknown language is Uzbek", "?lang=de", "", "", "uz"},
		{"an unknown header falls through", "", "tr", "ru", "ru"},
	}
	for _, c := range cases {
		if got := reportLang(req(c.query, c.header, c.cookie)); got != c.want {
			t.Errorf("%s: lang = %q, want %q", c.name, got, c.want)
		}
	}
}

// Every column title has to move when the language does.
//
// ⚠️ This is the check the feature exists for: an untranslated column is not
// an error anywhere — it compiles, it renders, and it arrives in the
// accountant's inbox as the one Uzbek word in a Russian sheet. A column added
// next year without its `tr` triple fails here instead.
func TestReportColumnsAreTranslated(t *testing.T) {
	// Titles that are the same word in every language and must not be flagged.
	sameEverywhere := map[string]bool{"ABC": true, "XYZ": true}

	for name, cols := range map[string][2][]Column{
		"abc":      {abcColumns("uz", true), abcColumns("ru", true)},
		"channels": {channelColumns("uz"), channelColumns("ru")},
		"sales":    {salesColumns(groupDay, "uz"), salesColumns(groupDay, "ru")},
		"couriers": {courierReportColumns("uz"), courierReportColumns("ru")},
		"staff":    {staffReportColumns("uz"), staffReportColumns("ru")},
		"finance":  {financeColumns("uz"), financeColumns("ru")},
		"cash":     {cashColumns("uz"), cashColumns("ru")},
	} {
		uz, ru := cols[0], cols[1]
		if len(uz) != len(ru) {
			t.Fatalf("%s: %d columns in uz, %d in ru", name, len(uz), len(ru))
		}
		for i := range uz {
			if uz[i].Key != ru[i].Key {
				t.Fatalf("%s: column %d is %q in uz and %q in ru", name, i, uz[i].Key, ru[i].Key)
			}
			if uz[i].Title == ru[i].Title && !sameEverywhere[uz[i].Title] {
				t.Errorf("%s: column %q is %q in both languages", name, uz[i].Key, uz[i].Title)
			}
		}
	}
}

// The notes under the titles are read on screen as well as in the sheet, so
// they follow the panel too.
func TestReportNotesAreTranslated(t *testing.T) {
	for name, notes := range map[string][2]string{
		"abc":      {abcNote("uz"), abcNote("ru")},
		"channels": {channelNote("uz"), channelNote("ru")},
		"sales":    {salesNote("uz"), salesNote("ru")},
		"couriers": {courierReportNote("uz"), courierReportNote("ru")},
		"staff":    {staffReportNote("uz"), staffReportNote("ru")},
		"finance":  {financeNote("uz", false), financeNote("ru", false)},
	} {
		if notes[0] == notes[1] {
			t.Errorf("%s: the note is identical in uz and ru", name)
		}
		if strings.TrimSpace(notes[1]) == "" {
			t.Errorf("%s: the ru note is empty", name)
		}
	}
}

// ⚠️ The download's file name must not depend on the language it was
// downloaded in. `fileSlug` keeps only ASCII, so a Russian title reduces to
// nothing and every report in the folder would be called `hisobot.xlsx` —
// which is the same thing as having no name at all.
func TestReportFileNameIsLanguageIndependent(t *testing.T) {
	for _, lang := range []string{"uz", "ru", "en"} {
		rec := httptest.NewRecorder()
		rep := &Report{
			Title:   salesTitle(groupDay, lang),
			Slug:    "savdo-day",
			From:    "2026-08-01",
			To:      "2026-08-15",
			Columns: salesColumns(groupDay, lang),
		}
		if err := writeXLSX(rec, rep); err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		name := rec.Header().Get("Content-Disposition")
		if !strings.Contains(name, `filename="savdo-day-`) {
			t.Errorf("%s: file name is %q", lang, name)
		}
	}
}
