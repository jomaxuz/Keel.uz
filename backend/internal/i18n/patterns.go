package i18n

// ---- Messages that carry a value in them ----
//
// ⚠️ **Two thirds of what a person reads never reaches the catalogue as a
// whole sentence.** `httpx.Error(w, 409, err.Error())` is written 790 times,
// and the error behind it was built with a name, a count or a wrapped cause
// inside it: `fmt.Errorf("poster %s: ulanib bo'lmadi: %w", …)`, or plain
// concatenation — `name + " bu filialda tugagan"`. By the time the message is
// written it is one runtime string that is in the catalogue under no key at
// all, so exact lookup misses every one of them and the owner reads Uzbek.
//
// So a key may carry verbs (`%s`, `%d`) and is matched against the finished
// string: the literal halves have to line up, the verbs capture what sat
// between them, and the captures are dropped into the translation in the same
// order. The value itself is never translated — it is a POS name, a table
// number, a supplier — and it should not be.
//
// ⚠️ **A key that is nearly all verb would match nearly everything.** `"%s"`
// as a key would swallow every message in the product and answer with whatever
// translation happened to sort first. Patterns are therefore tried
// longest-literal-first, and `patterns_test.go` refuses a key whose fixed text
// is too short to identify it.

import (
	"regexp"
	"sort"
	"strings"
	"sync"
)

// verbRe is the whole of the format language the catalogue speaks: a string
// and a number. `%w` never appears here — the wrapped error is already a
// string by the time the message is written, so it is keyed as `%s`.
var verbRe = regexp.MustCompile(`%[sd]`)

// minFixed is how much literal text a pattern needs before it is allowed to
// claim a message. Short enough for "%s: %s"-shaped keys to be rejected, long
// enough that a real sentence fragment passes.
const minFixed = 8

type patternEntry struct {
	key   string
	re    *regexp.Regexp
	verbs []string // in the order they appear in the key
	fixed int      // literal characters, for ordering
	t     pair
}

var (
	patternsOnce sync.Once
	patterns     []patternEntry
)

// buildPatterns compiles every catalogue key that has a verb in it.
//
// ⚠️ Ordered by falling literal length, not by map order: Go randomises map
// iteration, so without this the winner between two patterns that both match
// would change between runs — a translation that is right in the test and
// wrong in production, or the other way round.
func buildPatterns() {
	for key, t := range messages {
		locs := verbRe.FindAllStringIndex(key, -1)
		if len(locs) == 0 {
			continue
		}
		e := patternEntry{key: key, t: t}
		var b strings.Builder
		b.WriteString(`\A`)
		last := 0
		for _, loc := range locs {
			lit := key[last:loc[0]]
			e.fixed += len(lit)
			b.WriteString(regexp.QuoteMeta(lit))
			v := key[loc[0]:loc[1]]
			e.verbs = append(e.verbs, v)
			if v == "%d" {
				b.WriteString(`(-?\d+)`)
			} else {
				// Non-greedy: the anchors decide the ends, and a greedy
				// capture would eat the literal that follows it.
				b.WriteString(`(.+?)`)
			}
			last = loc[1]
		}
		tail := key[last:]
		e.fixed += len(tail)
		b.WriteString(regexp.QuoteMeta(tail))
		b.WriteString(`\z`)
		// ⚠️ **A carrier is allowed to be nearly all verb.** `"telegram: %s"`
		// and `"%s: %s"` are not translations, they are shapes: their ru and en
		// read exactly like the key, and they exist so the message *inside*
		// them can be found. `fmt.Errorf("%w: %s", ErrUnmapped, name)` is the
		// common case — the sentence the owner has to act on is the wrapped
		// error, and it never reaches the catalogue whole. Because a carrier
		// rewrites nothing, matching one too eagerly costs nothing either.
		if e.fixed < minFixed && !(t.ru == key && t.en == key) {
			continue
		}
		re, err := regexp.Compile(b.String())
		if err != nil {
			continue
		}
		e.re = re
		patterns = append(patterns, e)
	}
	sort.Slice(patterns, func(i, j int) bool {
		if patterns[i].fixed != patterns[j].fixed {
			return patterns[i].fixed > patterns[j].fixed
		}
		return patterns[i].key < patterns[j].key
	})
}

// localizePattern answers a message that was built around a value.
func localizePattern(lang, msg string) (string, bool) {
	patternsOnce.Do(buildPatterns)
	for _, p := range patterns {
		m := p.re.FindStringSubmatch(msg)
		if m == nil {
			continue
		}
		out := p.t.pick(lang)
		if out == "" {
			return "", false
		}
		// ⚠️ **The captured halves are localized too, and that is the point
		// of the carriers.** A value is usually a name and comes back
		// unchanged, but a wrapped error is a whole message and comes back
		// translated. Each capture is strictly shorter than the message it
		// came out of, so the recursion ends.
		vals := make([]string, 0, len(m)-1)
		for _, v := range m[1:] {
			vals = append(vals, Localize(lang, v))
		}
		return fill(out, vals), true
	}
	return "", false
}

// fill puts the captured values back, in the order the verbs appear.
//
// ⚠️ **Order, not position.** A translation may not renumber its verbs —
// Russian word order differs from Uzbek often enough that the temptation is
// real — and `patterns_test.go` insists the sequence matches the key, because
// a swapped pair would print the table number where the receipt number goes
// and nothing about the sentence would look wrong.
func fill(format string, vals []string) string {
	var b strings.Builder
	last, i := 0, 0
	for _, loc := range verbRe.FindAllStringIndex(format, -1) {
		b.WriteString(format[last:loc[0]])
		if i < len(vals) {
			b.WriteString(vals[i])
		}
		i++
		last = loc[1]
	}
	b.WriteString(format[last:])
	return b.String()
}
