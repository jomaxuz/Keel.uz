package i18n

// ⚠️ **This test is the whole reason keying on the Uzbek text is safe.**
//
// The catalogue is keyed by the message itself, which means rewording a
// sentence anywhere silently drops its translation: nothing fails to compile,
// no request errors, and a Russian cashier is simply shown an Uzbek sentence
// again — at the moment something has already gone wrong, which is the only
// moment these strings are ever read. So the test reads every message the API
// can put in front of a person out of `internal/` and insists on an entry here
// or a place on the `Untranslated` list.
//
// ⚠️ **It reads the syntax tree, not the line.** The first version of this
// test matched `httpx.Error(w, …, "…")` with a regular expression, which meant
// a call whose message sat on the next line — because gofmt wrapped it, which
// it does to every long sentence — was not scanned at all. Twenty-eight
// messages were untranslated for that reason alone, all of them long ones,
// which is to say all of them the ones that actually explain something. A
// guard that silently skips what it cannot parse is worse than no guard: it is
// a guard everybody trusts.
//
// ⚠️ **Most messages never reach `httpx.Error` as a literal.** They are built
// by the layer underneath — `errors.New("smena allaqachon ochiq")` in a
// service, wrapped or concatenated on the way up — and arrive at the writer as
// `err.Error()`. Those are scanned at their source instead, which is why this
// walks every package under `internal/` and not just the handlers.
//
// ⚠️ The list is not a way to make the test pass. It is for the messages that
// answer a malformed request rather than a person ("invalid id", "forbidden"),
// and adding a real one to it moves a visible bug into a file nobody reads.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// found is one message the code can put in front of a person.
type found struct {
	msg  string
	from string
}

// scanMessages walks internal/ and returns every message a person can be
// shown, keyed as the catalogue keys it: values inside the sentence become
// verbs.
func scanMessages(t *testing.T) []found {
	t.Helper()
	root := filepath.Join("..")
	var out []found
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, p, nil, 0)
		if perr != nil {
			t.Fatalf("%s: %v", p, perr)
		}
		name := filepath.Base(p)
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := ce.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, _ := sel.X.(*ast.Ident)
			if pkg == nil {
				return true
			}
			switch {
			// httpx.Error(w, status, msg) — the message as written.
			case pkg.Name == "httpx" && sel.Sel.Name == "Error" && len(ce.Args) >= 3:
				if s, ok := messageKey(ce.Args[2]); ok {
					out = append(out, found{s, name})
				}
			// httpx.T(w, msg) — the message a response carries in a field.
			case pkg.Name == "httpx" && sel.Sel.Name == "T" && len(ce.Args) >= 2:
				if s, ok := messageKey(ce.Args[1]); ok {
					out = append(out, found{s, name})
				}
			// errors.New / fmt.Errorf — the message as the layer below wrote
			// it, before it was handed up as err.Error().
			case pkg.Name == "errors" && sel.Sel.Name == "New" && len(ce.Args) == 1:
				if s, ok := messageKey(ce.Args[0]); ok {
					out = append(out, found{s, name})
				}
			case pkg.Name == "fmt" && sel.Sel.Name == "Errorf" && len(ce.Args) >= 1:
				if s, ok := messageKey(ce.Args[0]); ok {
					out = append(out, found{s, name})
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// verbInSource is what a value looks like in Go before it is printed. `%w`,
// `%v` and `%q` all arrive at the reader as text, so the catalogue keys them
// the same way it keys `%s` — the pattern has to match the finished string,
// not the format that produced it.
var verbInSource = regexp.MustCompile(`%[+#]?[0-9.]*[a-zA-Z]`)

// messageKey turns the argument into the key the catalogue would hold, or
// reports that there is no key to hold.
//
// ⚠️ A concatenation is keyed with a verb where the value goes — the reader is
// shown one sentence and the catalogue has to hold that one sentence, however
// the code happened to build it.
func messageKey(e ast.Expr) (string, bool) {
	var b strings.Builder
	if !writeKey(&b, e) {
		return "", false
	}
	return normalizeVerbs(b.String()), true
}

func writeKey(b *strings.Builder, e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return false
		}
		s, err := strconv.Unquote(v.Value)
		if err != nil {
			return false
		}
		b.WriteString(s)
		return true
	case *ast.BinaryExpr:
		if v.Op != token.ADD {
			return false
		}
		left := &strings.Builder{}
		right := &strings.Builder{}
		okL, okR := writeKey(left, v.X), writeKey(right, v.Y)
		if !okL && !okR {
			// Neither half is text we can key on; the message is built
			// entirely out of values and is not a sentence.
			return false
		}
		if okL {
			b.WriteString(left.String())
		} else {
			b.WriteString("%s")
		}
		if okR {
			b.WriteString(right.String())
		} else {
			b.WriteString("%s")
		}
		return true
	default:
		return false
	}
}

// normalizeVerbs rewrites the printing verbs to the two the catalogue speaks.
func normalizeVerbs(s string) string {
	return verbInSource.ReplaceAllStringFunc(s, func(v string) string {
		if strings.HasSuffix(v, "d") {
			return "%d"
		}
		return "%s"
	})
}

func TestEveryMessageIsTranslated(t *testing.T) {
	seen := map[string][]string{}
	for _, f := range scanMessages(t) {
		seen[f.msg] = append(seen[f.msg], f.from)
	}
	if len(seen) < 400 {
		t.Fatalf("faqat %d ta xabar topildi — skaner buzilgan bo'lsa, bu test hech nimani qo'riqlamaydi", len(seen))
	}

	var missing []string
	for msg, files := range seen {
		if Untranslated[msg] || strings.TrimSpace(msg) == "" {
			continue
		}
		// ⚠️ `Covered`, not `localizePattern`: a carrier such as "%s: %s"
		// matches almost any sentence with a colon in it, so the looser check
		// answered yes for messages nobody had translated — the test went green
		// and the sentence still arrived in Uzbek. Found the day a refusal was
		// written as `person.Name + ": …"`.
		if Covered(msg) {
			continue
		}
		missing = append(missing, msg+"  ("+files[0]+")")
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("tarjimasi yo'q %d ta xabar — internal/i18n/messages.go ga qo'shing "+
			"(yoki odamga emas, buzuq so'rovga javob bo'lsa Untranslated ro'yxatiga):\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
}

// allLiterals is every string literal in the tree.
//
// ⚠️ **Needed because not every message travels through `httpx.Error` or
// `errors.New`.** `permLabelUz` returns "Chegirma berish" from a switch and
// the caller hands it to `i18n.Localize` itself — the line the till prints
// beside its PIN pad. Judging those by the scan alone would call seven real
// translations stray and invite somebody to delete them.
func allLiterals(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	fset := token.NewFileSet()
	_ = filepath.WalkDir(filepath.Join(".."), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, p, nil, 0)
		if perr != nil {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if s, err := strconv.Unquote(lit.Value); err == nil {
					out[s] = true
				}
			}
			return true
		})
		return nil
	})
	return out
}

// ⚠️ A catalogue entry for a message nothing sends is not harmless: it is the
// ghost of a message somebody reworded, and it makes the count above look
// healthier than it is.
func TestCatalogueHasNoStrayEntries(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range scanMessages(t) {
		seen[f.msg] = true
	}
	lits := allLiterals(t)
	var stray []string
	for msg := range messages {
		if !seen[msg] && !lits[msg] {
			stray = append(stray, msg)
		}
	}
	sort.Strings(stray)
	if len(stray) > 0 {
		t.Fatalf("hech qayerda yuborilmaydigan %d ta tarjima:\n  %s",
			len(stray), strings.Join(stray, "\n  "))
	}
}

// ⚠️ The same, for the list: a message that was reworded leaves its old text
// behind here too, and there the cost is worse than a stale line — the list is
// read as "we decided not to translate this".
func TestUntranslatedListHasNoStrayEntries(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range scanMessages(t) {
		seen[f.msg] = true
	}
	lits := allLiterals(t)
	var stray []string
	for msg := range Untranslated {
		if !seen[msg] && !lits[msg] {
			stray = append(stray, msg)
		}
	}
	sort.Strings(stray)
	if len(stray) > 0 {
		t.Fatalf("hech qayerda yuborilmaydigan %d ta \"tarjima qilinmaydi\" yozuvi:\n  %s",
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

// A file that no longer parses would make every scan above return nothing,
// which is why the count is checked; this keeps the walk itself honest.
func TestScannerReadsTheTree(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "handlers")); err != nil {
		t.Fatalf("internal/handlers topilmadi: %v", err)
	}
}

// ⚠️ **The second way a sentence reaches a person, and the one that stayed
// Uzbek after `httpx.Error` was fixed.** A connection check does not fail:
// "the domain does not point here yet" is the answer that screen exists to
// give, so it is written with `httpx.JSON` — and `JSON` translates nothing.
// Every provider check in the panel (POS, PBX, Telegram, fiscal, domain,
// import) answered that way, beside labels that were all translated, on the
// only day anybody reads them.
//
// So a human sentence in a response *field* has to go through `httpx.T`, and
// this refuses the ones that do not. Only literals and `err.Error()` are
// judged: a value that comes from elsewhere cannot be told apart from a name
// or a stored record by reading the tree.
func TestFieldMessagesGoThroughT(t *testing.T) {
	// The keys a person reads. "note", "reason" and their kin are data the
	// restaurant typed in — translating those would be rewriting the owner.
	human := map[string]bool{"message": true, "error": true, "hint": true, "warning": true}

	// ⚠️ Answers to a machine, not to a person: the payment providers call
	// these and read the field themselves. Their wording is part of a protocol.
	callbacks := map[string]bool{
		"payatmos.go": true, "payclick.go": true, "paypayme.go": true, "payuzum.go": true,
	}

	var bare []string
	fset := token.NewFileSet()
	_ = filepath.WalkDir(filepath.Join("..", "handlers"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		name := filepath.Base(p)
		if callbacks[name] {
			return nil
		}
		f, perr := parser.ParseFile(fset, p, nil, 0)
		if perr != nil {
			t.Fatalf("%s: %v", p, perr)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := ce.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "JSON" {
				return true
			}
			if pkg, _ := sel.X.(*ast.Ident); pkg == nil || pkg.Name != "httpx" {
				return true
			}
			ast.Inspect(ce, func(m ast.Node) bool {
				kv, ok := m.(*ast.KeyValueExpr)
				if !ok {
					return true
				}
				k, ok := kv.Key.(*ast.BasicLit)
				if !ok || k.Kind != token.STRING {
					return true
				}
				key, _ := strconv.Unquote(k.Value)
				if !human[key] || wrapped(kv.Value) || !judgeable(kv.Value) {
					return true
				}
				bare = append(bare, fmt.Sprintf("%s:%d  %q",
					name, fset.Position(kv.Pos()).Line, key))
				return true
			})
			return true
		})
		return nil
	})
	sort.Strings(bare)
	if len(bare) > 0 {
		t.Fatalf("javob maydonidagi %d ta jumla tarjimasiz ketyapti — httpx.T(w, …) ga o'rang:\n  %s",
			len(bare), strings.Join(bare, "\n  "))
	}
}

// wrapped reports whether the value already goes through the translator.
func wrapped(e ast.Expr) bool {
	ce, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "T" {
		if pkg, _ := sel.X.(*ast.Ident); pkg != nil && pkg.Name == "httpx" {
			return true
		}
	}
	// permLabel is httpx.T with the permission's own name looked up first.
	if id, ok := ce.Fun.(*ast.Ident); ok && id.Name == "permLabel" {
		return true
	}
	return false
}

// judgeable reports whether the tree says enough about the value to insist on
// anything: a sentence written here, or an error's own text.
func judgeable(e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return false
		}
		s, err := strconv.Unquote(v.Value)
		return err == nil && strings.TrimSpace(s) != ""
	case *ast.BinaryExpr:
		return judgeable(v.X) || judgeable(v.Y)
	case *ast.CallExpr:
		sel, ok := v.Fun.(*ast.SelectorExpr)
		return ok && sel.Sel.Name == "Error" && len(v.Args) == 0
	}
	return false
}
