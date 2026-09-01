package i18n

// The pattern catalogue has two ways to be quietly wrong, and both of them
// print a sentence that reads perfectly.

import (
	"strings"
	"testing"
)

// ⚠️ **A translation may not renumber its verbs.** Russian word order differs
// from Uzbek often enough that moving `%s` past `%d` looks like the right
// thing to do — and the values are filled in the order they appear, so a
// swapped pair prints the table number where the receipt number goes and
// nothing about the result looks wrong.
func TestPatternVerbsLineUp(t *testing.T) {
	for key, tr := range messages {
		want := verbRe.FindAllString(key, -1)
		if len(want) == 0 {
			continue
		}
		for lang, got := range map[string][]string{
			"ru": verbRe.FindAllString(tr.ru, -1),
			"en": verbRe.FindAllString(tr.en, -1),
		} {
			if strings.Join(got, "") != strings.Join(want, "") {
				t.Errorf("%q (%s): kalitda %v, tarjimada %v — qiymatlar tartib bo'yicha qo'yiladi",
					key, lang, want, got)
			}
		}
	}
}

// ⚠️ **A pattern that is nearly all verb matches nearly everything.** It has
// to earn its place either with literal text of its own or by being a carrier
// — a shape whose translations read exactly like the key, so that claiming a
// message it should not have claimed changes nothing about it.
func TestShortPatternsAreCarriersOnly(t *testing.T) {
	patternsOnce.Do(buildPatterns)
	for _, p := range patterns {
		if p.fixed >= minFixed {
			continue
		}
		if p.t.ru != p.key || p.t.en != p.key {
			t.Errorf("%q: qisqa naqsh, lekin tarjimasi kalitdan farq qiladi — u boshqa xabarni ham ushlab qolishi mumkin", p.key)
		}
	}
}

// Every pattern has to match the message it was written for; a key that
// matches nothing is a translation nobody will ever be shown.
func TestPatternMatchesItsOwnKey(t *testing.T) {
	patternsOnce.Do(buildPatterns)
	for _, p := range patterns {
		sample := p.key
		for _, v := range p.verbs {
			if v == "%d" {
				sample = strings.Replace(sample, v, "7", 1)
			} else {
				sample = strings.Replace(sample, v, "X", 1)
			}
		}
		if !p.re.MatchString(sample) {
			t.Errorf("%q: o'z namunasiga (%q) mos kelmadi", p.key, sample)
		}
	}
}

func TestLocalizeFillsValuesIn(t *testing.T) {
	got := Localize(RU, "Lag'mon bugun tugadi")
	if got != "Lag'mon сегодня закончился" {
		t.Fatalf("qiymat o'z joyida emas: %q", got)
	}
	// A number keeps its place, and the sentence around it is translated.
	if got := Localize(EN, "kod yaqinda yuborilgan, 42 soniyadan keyin qayta urining"); got != "a code was just sent, try again in 42 seconds" {
		t.Fatalf("raqamli naqsh: %q", got)
	}
	// ⚠️ The carrier case: the wrapper is left as written and the message
	// inside it is translated. Before this, the owner read the provider's name
	// in Latin and the sentence after it in Uzbek.
	if got := Localize(RU, "taom POS tizimiga bog'lanmagan: Lag'mon"); got != "блюдо не привязано к кассе: Lag'mon" {
		t.Fatalf("o'ralgan xabar tarjima qilinmadi: %q", got)
	}
	// A message that matches nothing comes back as it was written.
	if got := Localize(RU, "bunday jumla katalogda yo'q"); got != "bunday jumla katalogda yo'q" {
		t.Fatalf("noma'lum xabar o'zgardi: %q", got)
	}
}

// ⚠️ Exact entries win over patterns. Both can match the same string —
// "%s: %s" matches most sentences with a colon in them — and the whole
// sentence is always the better answer.
func TestExactBeatsPattern(t *testing.T) {
	const msg = "onlinePBX: sozlanmagan"
	if got, want := Localize(RU, msg), messages[msg].ru; got != want {
		t.Fatalf("naqsh aniq yozuvni bosib ketdi: %q", got)
	}
}
