package i18n

// ⚠️ **This test is the whole reason keying on the Uzbek text is safe.**
//
// The catalogue is keyed by the message itself, which means rewording a
// sentence in a handler silently drops its translation: nothing fails to
// compile, no request errors, and a Russian cashier is simply shown an Uzbek
// sentence again — at the moment something has already gone wrong, which is the
// only moment these strings are ever read. So the test reads every literal out
// of `internal/handlers` and insists on an entry here or a place on the
// `Untranslated` list.
//
// ⚠️ The list is not a way to make the test pass. It is for the messages that
// answer a malformed request rather than a person ("invalid id", "forbidden"),
// and adding a real one to it moves a visible bug into a file nobody reads.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Only a complete literal argument: `httpx.Error(w, 400, "…")`. A message built
// by concatenation cannot be looked up at all — the runtime string is not this
// one — so demanding an entry for its prefix would be demanding a translation
// that never gets used.
var errCall = regexp.MustCompile(`httpx\.Error\(w, [^,]+, "([^"]*)"\)`)

func TestEveryHandlerMessageIsTranslated(t *testing.T) {
	dir := filepath.Join("..", "handlers")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	seen := map[string][]string{}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range errCall.FindAllStringSubmatch(string(src), -1) {
			seen[m[1]] = append(seen[m[1]], name)
		}
	}

	if len(seen) == 0 {
		t.Fatal("hech qanday xabar topilmadi — httpx.Error chaqiruvining shakli o'zgargan bo'lsa, bu test hech nimani qo'riqlamaydi")
	}

	var missing []string
	for msg, files := range seen {
		if Untranslated[msg] {
			continue
		}
		if _, ok := messages[msg]; !ok {
			missing = append(missing, msg+"  ("+files[0]+")")
		}
	}
	if len(missing) > 0 {
		t.Fatalf("tarjimasi yo'q %d ta xabar — internal/i18n/messages.go ga qo'shing "+
			"(yoki odamga emas, buzuq so'rovga javob bo'lsa Untranslated ro'yxatiga):\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
}

// ⚠️ A catalogue entry for a message no handler sends is not harmless: it is
// the ghost of a message somebody reworded, and it makes the count above look
// healthier than it is.
func TestCatalogueHasNoStrayEntries(t *testing.T) {
	dir := filepath.Join("..", "handlers")
	entries, _ := os.ReadDir(dir)
	var all strings.Builder
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		src, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		all.Write(src)
	}
	// Also read httpx itself: `errBadJSON` is written there, not in a handler.
	if src, err := os.ReadFile(filepath.Join("..", "httpx", "httpx.go")); err == nil {
		all.Write(src)
	}
	body := all.String()

	var stray []string
	for msg := range messages {
		if !strings.Contains(body, `"`+msg+`"`) {
			stray = append(stray, msg)
		}
	}
	if len(stray) > 0 {
		t.Fatalf("hech qayerda yuborilmaydigan %d ta tarjima:\n  %s",
			len(stray), strings.Join(stray, "\n  "))
	}
}

func TestLocalizeFallsBackToUzbek(t *testing.T) {
	// ⚠️ The fallback is the old behaviour, on purpose: a missing entry costs
	// the reader the translation, never the sentence.
	if got := Localize(RU, "bunday xabar yo'q"); got != "bunday xabar yo'q" {
		t.Fatalf("noma'lum xabar o'zgardi: %q", got)
	}
	if got := Localize("", "chek bo'sh"); got != "chek bo'sh" {
		t.Fatalf("til ko'rsatilmaganda o'zbekcha qolishi kerak: %q", got)
	}
	if got := Localize(RU, "chek bo'sh"); got == "chek bo'sh" {
		t.Fatal("ruscha tarjima qaytmadi")
	}
	if got := Localize(EN, "PIN noto'g'ri"); got != "wrong PIN" {
		t.Fatalf("inglizcha tarjima kutilgandek emas: %q", got)
	}
}
