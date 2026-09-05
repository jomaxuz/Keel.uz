// Package help serves the articles the panel and the owner's phone search.
//
// ⚠️ **Bundled with this server, not fetched from the platform, and that is the
// same argument the panel's own file makes.** An article describes what *this
// build* does; an answer hosted centrally would eventually describe a version
// the restaurant is not running. Embedded in the binary, it is deployed by the
// same commit as the behaviour it explains, and it is still there when the
// container can reach nothing else.
//
// ⚠️ **Moved here so there is one copy.** They used to live only in the panel's
// TypeScript bundle, which a native application cannot import — and the obvious
// fix, a Kotlin copy for the phone, is the copy that stops describing this
// build. The panel reads them from here now as well.
package help

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed articles.json
var raw []byte

// Article is one entry. ⚠️ The field names are the panel's, unchanged: this file
// was lifted out of its bundle and the two must stay readable as the same thing.
type Article struct {
	ID    string `json:"id"`
	Cat   string `json:"cat"`
	Title string `json:"title"`
	Body  string `json:"body"`
	// Words somebody would search that are not in the text.
	//
	// ⚠️ This is where the vocabulary gap goes. An owner types "kassa
	// ochilmayapti"; the article is titled "PIN qabul qilinmayapti". Neither
	// contains the other's words, and without this the search finds nothing for
	// the single most common question there is.
	Keys []string `json:"keys,omitempty"`
}

var (
	once   sync.Once
	byLang map[string][]Article
)

func load() {
	once.Do(func() {
		byLang = map[string][]Article{}
		_ = json.Unmarshal(raw, &byLang)
	})
}

// For returns the articles in a language.
//
// ⚠️ **Falls back to Uzbek, which is the base and the only complete one.** The
// translations lag on purpose — an article is written when the behaviour is —
// so a Russian reader with no Russian article is better served by an Uzbek one
// than by an empty help screen.
func For(lang string) []Article {
	load()
	if a, ok := byLang[lang]; ok && len(a) > 0 {
		return a
	}
	return byLang["uz"]
}

// Langs reports which languages carry their own articles.
func Langs() []string {
	load()
	out := make([]string, 0, len(byLang))
	for k := range byLang {
		out = append(out, k)
	}
	return out
}
