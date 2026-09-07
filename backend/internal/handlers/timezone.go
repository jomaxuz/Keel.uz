package handlers

// ---- The restaurant's own hours, in Go and in Mongo ----
//
// ⚠️ **Two vocabularies for one fact.** Go carries a `*time.Location`; Mongo's
// `$dateToString` and `$dayOfWeek` take a *name*, and it must be one the server
// recognises. They agree in production and come apart on a developer's machine,
// which is the worst possible place for a difference to hide: the aggregation
// is rejected, the handler swallows the error the way every read here does, and
// the screen draws itself with nothing on it.

import "time"

// local is a database time in the restaurant's own hours.
//
// ⚠️ **The driver decodes every time as UTC**, whatever `TZ` says — CLAUDE.md
// pays for this twice. It changes nothing for a full timestamp the browser
// parses, and everything the moment anybody formats one.
func local(t time.Time) time.Time { return t.In(time.Local) }

// mongoTZ is the timezone as an aggregation will accept it.
//
// ⚠️ **`time.Local.String()` is not always a timezone name, and when it is not,
// the pipeline is refused.** With `TZ` set — which is how every container runs,
// because CLAUDE.md's alpine trap forced it — the name is "Asia/Tashkent" and
// everything works. With `TZ` unset the zone is still read from
// `/etc/localtime`, the clock is still right, and the name degrades to the
// literal string **"Local"** — which Mongo answers with `unrecognized time zone
// identifier`. Nothing crashes: the aggregation returns an error, the caller
// returns an empty map, and the screen simply has no rows on it. That is how
// the sold-per-day rows on the movement report were quietly empty on a
// developer's machine, and how the ordering forecast was found to be.
//
// So a name that is not one falls back to the current UTC offset, which every
// Mongo version accepts. ⚠️ **An offset, not a zone**: it cannot know about a
// daylight saving change, so a summer aggregation run in winter would be an
// hour out. That is the right trade here — this path exists only where `TZ` is
// unset (never in production), and the alternative is a screen with nothing on
// it and no error anywhere.
func mongoTZ() string {
	name := time.Local.String()
	// A real IANA name always has a region: "Asia/Tashkent", "Europe/London".
	// "UTC" is the one exception, and Mongo takes it.
	for _, c := range name {
		if c == '/' {
			return name
		}
	}
	if name == "UTC" {
		return name
	}
	return time.Now().Format("-07:00")
}
