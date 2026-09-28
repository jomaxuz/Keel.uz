package handlers

import (
	"strings"
	"testing"

	"keel-control/internal/config"
)

// A full file the way keel-site/src/lib/llms.ts writes it: a `##` heading, a
// blank line, the `URL:` line, then the page.
const sampleFull = `# Keel — full

> summary

## Bosh sahifa

URL: https://keel.uz

Kassa, zal, oshxona.

## Blog › Tarozi

URL: https://keel.uz/blog/tarozi

Birinchi xatboshi.

## Muallifning o'z sarlavhasi

Bu ham postning bir qismi.

## Qo'llanma › Printer

URL: https://keel.uz/help/printer

Printerni ulash.
`

func pageHash(pages []llmsPage, url string) string {
	for _, p := range pages {
		if p.URL == url {
			return p.Hash
		}
	}
	return ""
}

// ⚠️ **An author's own `##` heading does not end the post.** The blog is free
// text; if a subheading inside a post were read as the next page, everything
// below it would stop being watched and an edit there would never be pinged.
func TestSplitKeepsAuthorsHeadingsInsideTheirPost(t *testing.T) {
	pages := splitLLMsFull(sampleFull)
	if len(pages) != 3 {
		t.Fatalf("3 ta sahifa kutilgan, %d: %+v", len(pages), pages)
	}
	edited := strings.Replace(sampleFull, "Bu ham postning bir qismi.", "Tahrirlangan.", 1)
	after := splitLLMsFull(edited)
	if pageHash(pages, "https://keel.uz/blog/tarozi") == pageHash(after, "https://keel.uz/blog/tarozi") {
		t.Fatal("post ichidagi sarlavha ostidagi tahrir sezilmadi")
	}
	for _, u := range []string{"https://keel.uz", "https://keel.uz/help/printer"} {
		if pageHash(pages, u) != pageHash(after, u) {
			t.Fatalf("%s o'zgarmagan edi, lekin xeshi o'zgardi", u)
		}
	}
}

// ⚠️ **A page's own heading belongs to it.** Renaming an article is a change
// to that article — and only to it, not to the page above it.
func TestRenamingAPageChangesOnlyThatPage(t *testing.T) {
	before := splitLLMsFull(sampleFull)
	after := splitLLMsFull(strings.Replace(sampleFull, "## Qo'llanma › Printer", "## Qo'llanma › Printerni ulash", 1))
	got := diffLLMs(before, after)
	if len(got) != 1 || got[0] != "https://keel.uz/help/printer" {
		t.Fatalf("faqat printer maqolasi kutilgan edi: %v", got)
	}
}

// ⚠️ **A removed page is reported too** — submitting it is how the engines
// learn it is gone.
func TestDiffNamesNewAndRemovedPages(t *testing.T) {
	old := []llmsPage{{URL: "a", Hash: "1"}, {URL: "b", Hash: "2"}}
	cur := []llmsPage{{URL: "a", Hash: "1"}, {URL: "c", Hash: "3"}}
	got := diffLLMs(old, cur)
	if strings.Join(got, ",") != "b,c" {
		t.Fatalf("b (o'chgan) va c (yangi) kutilgan: %v", got)
	}
	if len(diffLLMs(cur, cur)) != 0 {
		t.Fatal("o'zgarmagan fayl o'zgarish deb o'qildi")
	}
}

// Six addresses, every language, on our own host — IndexNow refuses a whole
// submission that carries one foreign URL.
func TestLLMsURLsCoverEveryLanguage(t *testing.T) {
	h := &Handler{Cfg: &config.Config{BaseDomain: "keel.uz"}}
	got := strings.Join(h.llmsURLs(), " ")
	for _, want := range []string{
		"https://keel.uz/llms.txt", "https://keel.uz/llms-full.txt",
		"https://keel.uz/ru/llms.txt", "https://keel.uz/ru/llms-full.txt",
		"https://keel.uz/en/llms.txt", "https://keel.uz/en/llms-full.txt",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("%s yo'q: %s", want, got)
		}
	}
}
