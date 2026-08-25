package receipt

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// block is the character grid a thermal printer actually prints.
//
// ⚠️ **Everything here counts runes, never bytes.** Uzbek written properly
// carries `oʻ` and `gʻ`, Russian is Cyrillic, and both are multi-byte in UTF-8.
// A width computed with len() makes every line with a Cyrillic word in it too
// short — by exactly as many characters as there were letters — and the symptom
// is a receipt whose columns drift further right the further down you read.
// The same arithmetic mistake the SMS part counter exists to avoid.
type block struct {
	w     int
	lines []string
}

// raw appends a line as given, trimmed to the paper.
func (b *block) raw(s string) {
	b.lines = append(b.lines, truncate(s, b.w))
}

// line puts one string against the left margin and another against the right.
//
// ⚠️ **The right-hand string wins when they collide.** On a receipt the right
// column is the money, and a label that eats into a total produces a number
// that is wrong rather than a label that is short. So the left is cut and the
// amount is printed whole.
func (b *block) line(left, right string) {
	rw := width(right)
	if rw >= b.w {
		b.raw(right)
		return
	}
	space := b.w - rw
	// One character of gap so the label and the amount never touch, which on a
	// low-contrast thermal print reads as a single word.
	left = truncate(left, max(0, space-1))
	pad := space - width(left)
	b.lines = append(b.lines, left+strings.Repeat(" ", pad)+right)
}

// lineMarked is `line` with an emphasis marker on the front.
//
// ⚠️ **The marker goes on after the padding is worked out, not before.** It is a
// control character with no width on paper, and counting it would shift the
// amount one column left on exactly the lines somebody chose to emphasise —
// which are the lines they are most likely to be looking at.
func (b *block) lineMarked(mark, left, right string) {
	b.line(left, right)
	if mark == "" || len(b.lines) == 0 {
		return
	}
	b.lines[len(b.lines)-1] = mark + b.lines[len(b.lines)-1]
}

// center puts a string in the middle of the paper.
func (b *block) center(s string) {
	// ⚠️ **The marker is taken off before anything is measured.** It is a
	// control character with no width on paper, and counting it shifts the text
	// one column — on precisely the lines somebody chose to emphasise, which
	// are the ones they will be looking at.
	mark, s := splitMark(s)
	s = truncate(s, b.w)
	pad := (b.w - width(s)) / 2
	b.lines = append(b.lines, mark+strings.Repeat(" ", pad)+s)
}

// splitToWidth breaks one line into as many as the paper needs.
//
// ⚠️ **Words are kept whole where they fit.** A phone number or an address cut
// mid-word is harder to read than the same text on two lines, and the one place
// this text appears is the top of a receipt somebody designed.
func splitToWidth(s string, w int) []string {
	if width(s) <= w {
		return []string{s}
	}
	var out []string
	cur := ""
	for _, word := range strings.Fields(s) {
		candidate := word
		if cur != "" {
			candidate = cur + " " + word
		}
		if width(candidate) <= w {
			cur = candidate
			continue
		}
		if cur != "" {
			out = append(out, cur)
		}
		// A single word wider than the paper is cut rather than dropped: an
		// empty line where a restaurant's name should be is worse.
		for width(word) > w {
			out = append(out, truncate(word, w))
			word = string([]rune(word)[len([]rune(truncate(word, w))):])
		}
		cur = word
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// splitMark takes an emphasis marker off the front of a string.
func splitMark(s string) (mark, rest string) {
	if s == "" {
		return "", s
	}
	switch s[0] {
	case 0x01, 0x02, 0x03:
		return s[:1], s[1:]
	}
	return "", s
}

// rule draws a separator across the paper.
func (b *block) rule() {
	b.lines = append(b.lines, strings.Repeat("-", b.w))
}

// wrap breaks a long string across as many lines as it needs.
//
// ⚠️ **Wrapped, not truncated.** A dish name is the one thing on a kitchen
// ticket that has to survive intact — "Lag'mon (achchiq, katta porsiya)" cut at
// 32 characters becomes a different dish. Continuation lines are indented so
// the eye can still find where each item starts.
func (b *block) wrap(s string) {
	// ⚠️ **Taken off before measuring and put back on every line it produces.**
	// A dish name that wraps to two lines has to be emphasised on both — one
	// bold line above a plain one reads as a rendering fault rather than as a
	// long name.
	mark, s := splitMark(s)
	if mark != "" {
		start := len(b.lines)
		b.wrap(s)
		for i := start; i < len(b.lines); i++ {
			b.lines[i] = mark + b.lines[i]
		}
		return
	}
	s = strings.TrimRight(s, " ")
	if width(s) <= b.w {
		b.lines = append(b.lines, s)
		return
	}
	indent := leadingSpaces(s) + "  "
	limit := b.w
	cur := ""
	for _, word := range strings.Fields(s) {
		candidate := word
		if cur != "" {
			candidate = cur + " " + word
		}
		if width(candidate) <= limit {
			cur = candidate
			continue
		}
		if cur != "" {
			b.lines = append(b.lines, cur)
		}
		// A single word longer than the paper is cut rather than dropped: the
		// alternative is an empty line where a name should be.
		cur = indent + word
		for width(cur) > limit {
			b.lines = append(b.lines, truncate(cur, limit))
			cur = indent + trimRunes(cur, limit)
		}
		limit = b.w
	}
	if cur != "" {
		b.lines = append(b.lines, cur)
	}
}

func leadingSpaces(s string) string {
	n := len(s) - len(strings.TrimLeft(s, " "))
	return s[:n]
}

// width is the printed width of a string, in characters.
func width(s string) int { return utf8.RuneCountInString(s) }

// truncate cuts a string to a character count.
func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if width(s) <= n {
		return s
	}
	out := make([]rune, 0, n)
	for _, r := range s {
		if len(out) == n {
			break
		}
		out = append(out, r)
	}
	return string(out)
}

// trimRunes drops the first n characters.
func trimRunes(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return ""
	}
	return string(rs[n:])
}

func itoa(n int) string { return strconv.Itoa(n) }

// money formats an amount the way a receipt reads it.
//
// ⚠️ **Grouped by hand with a space, never with Intl or a locale.** A receipt is
// printed by a machine in Tashkent regardless of what language the panel was in
// when the template was saved, and a thousands separator that follows the
// browser's locale would put commas on some receipts and spaces on others from
// the same restaurant. The same reason formatPrice avoids Intl on the site.
func money(n int, currency string) string {
	neg := n < 0
	if neg {
		n = -n
	}
	digits := strconv.Itoa(n)
	var parts []string
	for len(digits) > 3 {
		parts = append([]string{digits[len(digits)-3:]}, parts...)
		digits = digits[:len(digits)-3]
	}
	parts = append([]string{digits}, parts...)
	out := strings.Join(parts, " ")
	if neg {
		out = "-" + out
	}
	if currency != "" {
		out += " " + currency
	}
	return out
}
