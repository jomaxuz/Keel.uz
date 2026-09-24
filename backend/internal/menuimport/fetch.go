// Package menuimport turns a page on the internet into a menu the owner can
// review.
//
// # Why this exists
//
// Typing a menu in is the longest job in setting a restaurant up: a hundred
// dishes, each with a name in two languages, a price, a description and a
// photograph. Most restaurants already have all of it somewhere — their old
// site, an aggregator's page, a delivery service's listing — and retyping it is
// the reason a signed customer takes three weeks to go live.
//
// # ⚠️ Nothing here writes to the menu
//
// Import produces a **proposal**. The owner sees it, edits it, unticks what
// they do not want, and presses apply. An importer that wrote a hundred and
// twenty dishes straight into a live menu would be a mistake nobody can undo by
// hand — and the mistakes are guaranteed, because the input is somebody else's
// page.
//
// # ⚠️ The server fetching a URL somebody typed is an SSRF, and this one is
// worse than most
//
// This container sits on a Docker network next to Mongo, the control plane and
// every other tenant's backend. A URL of `http://mongo:27017` or
// `http://169.254.169.254/` is not a hypothetical — it is one paste into a box
// that exists to accept pasted addresses. So the address is resolved and every
// address it resolves to is checked, and redirects are re-checked one at a
// time: a public hostname that redirects to a private one is the standard way
// past a check that only looks at what was typed.
package menuimport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"restaurant-backend/internal/netguard"
	"strings"
	"time"
)

const (
	// A menu page is text and markup. Anything beyond this is not a menu, and
	// the tenant container is capped at one core and a modest heap.
	maxPage = 4 << 20
	// Long enough for a slow aggregator, short enough that the owner is not
	// left looking at a spinner wondering whether they typed it wrong.
	fetchTimeout = 20 * time.Second
	// ⚠️ Chased by hand, one at a time, so each hop is checked. `http.Client`'s
	// own following would resolve and dial before we ever saw the target.
	maxRedirects = 5
)

// ErrBlocked is a URL that points inside the network rather than out of it.
//
// ⚠️ Its own error so the panel can say what actually happened. "Could not
// fetch" sends an owner to check their typing; this is not about their typing.
var ErrBlocked = errors.New("bu manzil ichki tarmoqqa qaraydi — tashqi sayt havolasini kiriting")

// Fetch reads a page, refusing anything that is not a public http(s) address.
func Fetch(ctx context.Context, raw string) (string, string, error) {
	return FetchHeaders(ctx, raw, nil)
}

// FetchHeaders is Fetch with extra request headers.
//
// ⚠️ **The same function, not a second one.** An aggregator's menu API wants an
// Authorization header, and the obvious shortcut is a small `http.Get` beside
// this one — which is how a second path to the network appears that nobody
// remembered to put the address check on. Every SSRF rule lives here, so every
// caller gets them: the header map is the only thing that varies.
func FetchHeaders(
	ctx context.Context, raw string, extra map[string]string,
) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	next := strings.TrimSpace(raw)
	if next == "" {
		return "", "", errors.New("havola kiritilmagan")
	}
	// A pasted address usually has no scheme. Assumed rather than refused —
	// "https://" is not something an owner should have to know to type — but
	// assumed as https, never http.
	if !strings.Contains(next, "://") {
		next = "https://" + next
	}

	client := &http.Client{
		// Every hop is inspected below, so the client must not take any itself.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for hop := 0; hop <= maxRedirects; hop++ {
		u, err := url.Parse(next)
		if err != nil {
			return "", "", errors.New("havola tushunarsiz")
		}
		if err := checkPublic(u); err != nil {
			return "", "", err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return "", "", err
		}
		// ⚠️ A browser's User-Agent, and it is not a trick: many menu pages
		// answer a bare Go client with a block page, and importing a block page
		// as a menu is the failure mode that looks like the site having no
		// dishes on it.
		req.Header.Set("User-Agent",
			"Mozilla/5.0 (compatible; KeelMenuImport/1.0; +https://keel.uz)")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/json")
		req.Header.Set("Accept-Language", "uz,ru;q=0.9,en;q=0.8")
		for k, v := range extra {
			req.Header.Set(k, v)
		}

		res, err := client.Do(req)
		if err != nil {
			return "", "", fmt.Errorf("sahifani ochib bo'lmadi: %w", err)
		}

		if loc := res.Header.Get("Location"); res.StatusCode >= 300 &&
			res.StatusCode < 400 && loc != "" {
			res.Body.Close()
			ref, err := u.Parse(loc)
			if err != nil {
				return "", "", errors.New("havola tushunarsiz")
			}
			next = ref.String()
			continue
		}

		defer res.Body.Close()
		if res.StatusCode >= 400 {
			return "", "", fmt.Errorf("sayt %d qaytardi", res.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(res.Body, maxPage))
		if err != nil {
			return "", "", err
		}
		return string(body), u.String(), nil
	}
	return "", "", errors.New("havola juda ko'p marta yo'naltirdi")
}

// checkPublic refuses anything that is not a public http(s) address.
//
// ⚠️ **Every resolved address, not the first.** A hostname can resolve to a
// public address and a private one, and picking the first is picking whichever
// the resolver happened to order first this second.
func checkPublic(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		// file://, gopher://, and every other scheme somebody has ever used to
		// read a server's own disk.
		return ErrBlocked
	}
	host := u.Hostname()
	if host == "" {
		return ErrBlocked
	}
	// ⚠️ Checked by name as well as by address: `localhost` and the Docker
	// service names beside us do not have to resolve to something this function
	// would recognise for the request to be a bad idea.
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") ||
		strings.HasSuffix(lower, ".internal") || !strings.Contains(lower, ".") {
		// ⚠️ A hostname with no dot is a Docker service name — `mongo`,
		// `keel-control`, `keel-<slug>` — which is exactly the set this is here
		// to keep out, and no public site has one.
		return ErrBlocked
	}

	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return errors.New("bunday sayt topilmadi")
	}
	for _, ip := range ips {
		if !publicIP(ip) {
			return ErrBlocked
		}
	}
	return nil
}

// publicIP is the one rule, kept in netguard so the webhook sender reads the
// same list — two copies of a block list drift, and the one that drifts is the
// one nobody is looking at.
func publicIP(ip net.IP) bool { return netguard.PublicIP(ip) }

// Image size caps. A dish photograph is a photograph; anything past this is a
// page that pointed us at a video or a poster.
const maxImage = 6 << 20

// FetchImage downloads one photograph, with the same address rules as the page.
//
// ⚠️ **Checked again, not trusted because the page it came from was.** An image
// address is a URL the page chose, and a page that wants to read this server's
// network only has to put `http://169.254.169.254/` in an `<img src>`. It is
// the same door, one step further in, and it is the one somebody forgets.
func FetchImage(ctx context.Context, raw string) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, "", err
	}
	if err := checkPublic(u); err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent",
		"Mozilla/5.0 (compatible; KeelMenuImport/1.0; +https://keel.uz)")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, "", fmt.Errorf("rasm %d", res.StatusCode)
	}

	// ⚠️ **The content type decides the extension, not the URL.** Half of these
	// addresses end in `/photo` or carry a query string, and a file saved as
	// `.jpg` because the path said so — when it is a WebP — is one the browser
	// still shows and the image resizer cannot read.
	ext := extFor(res.Header.Get("Content-Type"))
	if ext == "" {
		return nil, "", errors.New("bu rasm emas")
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, maxImage))
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", errors.New("rasm bo'sh")
	}
	return data, ext, nil
}

func extFor(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	}
	return ""
}
