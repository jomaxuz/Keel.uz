package menuimport

import "testing"

// ⚠️ **Uzum Tezkor is the case this exists for.** uzum.uz sends every
// non-browser client to Yandex SmartCaptcha: the fetch succeeds with 200 and
// ten kilobytes of "Siz robot emasmisiz?". Undetected, four readers find no
// dishes in it and the model is then paid to read the challenge page — after
// which the owner is told their menu page has no menu on it.
func TestTheCaptchaRedirectIsRecognised(t *testing.T) {
	final := "https://www.uzum.uz/tmgrdfrend/showcaptcha?cc=1&retpath=aHR0cHM"
	if !BotWall("<html>Siz robot emasmisiz?</html>", final) {
		t.Fatal("SmartCaptcha yo'naltirishi tanilmadi")
	}
}

func TestAChallengePageWithoutARedirectIsRecognised(t *testing.T) {
	if !BotWall("<html><head><title>Just a moment...</title></head>", "https://x.uz/menu") {
		t.Fatal("Cloudflare oraliq sahifasi tanilmadi")
	}
}

// ⚠️ The word appearing deep in a real page is not a wall. Only the head of the
// document is searched, because a challenge page says so before anything else —
// there is nothing else on it.
func TestAMenuMentioningCaptchaFurtherDownIsNotAWall(t *testing.T) {
	page := "<html><body>" + string(make([]byte, 8000)) + "smartcaptcha</body></html>"
	if BotWall(page, "https://restoran.uz/menyu") {
		t.Fatal("oddiy sahifa robot devori deb belgilandi")
	}
}

func TestAnOrdinaryPageIsNotAWall(t *testing.T) {
	if BotWall("<html><body><h1>Menyu</h1></body></html>", "https://restoran.uz/menyu") {
		t.Fatal("oddiy sahifa robot devori deb belgilandi")
	}
}
