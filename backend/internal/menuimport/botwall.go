package menuimport

import "strings"

// BotWall reports whether what came back is a robot check rather than a page.
//
// ⚠️ **Without this the check reads as an empty menu.** Uzum Tezkor puts the
// whole of uzum.uz behind Yandex SmartCaptcha for anything that is not a
// browser: the fetch succeeds, 200, ten kilobytes — of "Siz robot emasmisiz?".
// Four readers then find no dishes in it, and the page text, which is real
// text, is handed to the model and paid for. The owner is told their menu page
// has no menu on it. Every part of that is wrong in a different way, and none
// of it says the one thing they need to hear: this site will not let a server
// read it, and no link they try will change that.
//
// ⚠️ **The final URL is the reliable half.** A challenge page's wording is
// translated into the visitor's language and rewritten between releases; the
// address it redirects to is machinery. The text markers are a second net for
// walls that answer in place without redirecting.
func BotWall(page, finalURL string) bool {
	u := strings.ToLower(finalURL)
	for _, m := range []string{
		"/showcaptcha",       // Yandex SmartCaptcha — what uzum.uz uses
		"/cdn-cgi/challenge", // Cloudflare
		"/captcha",
	} {
		if strings.Contains(u, m) {
			return true
		}
	}
	// ⚠️ Only the head of the document. A menu page may legitimately contain
	// the word "captcha" somewhere in a footer script; a challenge page says so
	// before anything else, because there is nothing else on it.
	head := page
	if len(head) > 4096 {
		head = head[:4096]
	}
	head = strings.ToLower(head)
	for _, m := range []string{
		"smartcaptcha",
		"just a moment...", // Cloudflare's interstitial title
		"attention required! | cloudflare",
		"checking your browser before accessing",
	} {
		if strings.Contains(head, m) {
			return true
		}
	}
	return false
}
