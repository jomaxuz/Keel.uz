package handlers

// ---- Boosting a post the restaurant already published ----
//
// ⚠️ **The other half of what a targetolog does.** Half the work is deciding
// what to advertise and writing it; the other half is "that post did well, put
// money behind it". A restaurant that already posts its food on Instagram has
// the second one sitting there, with the likes and comments already on it —
// and Meta keeps those when the post is promoted, which a freshly assembled
// advert can never have.
//
// ⚠️ **A borrowed creative, never an assembled one.** Nothing of ours travels
// into it: not a headline, not a picture, not a link. The post is the advert,
// exactly as its followers saw it. That is why this path skips every check the
// other one makes about photographs and wording — there is nothing of ours to
// check.

import (
	"context"
	"net/http"
	"strings"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/meta"

	"go.mongodb.org/mongo-driver/bson"
)

// adsPost is one thing the restaurant published, wherever it published it.
//
// ⚠️ **Two sources, one list.** An owner thinks "that post did well", not
// "that Instagram media object". Which network it came from decides how the
// creative is built and is shown as a label, but it is not a choice to make
// before seeing the posts.
type adsPost struct {
	ID      string `json:"id"`
	Source  string `json:"source"`
	Image   string `json:"image"`
	Caption string `json:"caption"`
	Link    string `json:"link"`
	At      string `json:"at"`
	// False only where Meta told us so — an Instagram post it will not boost.
	// A Page post carries no such verdict, and inventing one would grey out
	// posts that are perfectly promotable.
	Blocked bool   `json:"blocked,omitempty"`
	Why     string `json:"why,omitempty"`
}

// AdminAdsInstagram lists the posts this restaurant could put money behind.
func (h *Handler) AdminAdsInstagram(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	ctx := r.Context()
	client, s, err := h.adsClient(ctx)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	out := make([]adsPost, 0, 48)

	// ---- Instagram ----
	//
	// ⚠️ **Tried, and allowed to fail.** Listing Instagram media needs
	// `instagram_basic`, which Meta does not offer inside a Login for Business
	// configuration built on a system user token — the token an ad account
	// needs. So this is the half that may be refused, and its refusal must not
	// take the Page's posts down with it.
	if ig, err := h.instagramID(ctx, client, s.InstagramID, s.AdAccountID); err == nil && ig != "" {
		if posts, err := client.InstagramPosts(ctx, ig, 24); err == nil {
			for _, p := range posts {
				img := p.Thumbnail
				if img == "" {
					img = p.MediaURL
				}
				row := adsPost{
					ID: p.ID, Source: "instagram", Image: img,
					Caption: p.Caption, Link: p.Permalink, At: p.Timestamp,
				}
				if !p.Boost.Eligible {
					row.Blocked = true
					row.Why = strings.Join(p.Boost.Reasons, ", ")
				}
				out = append(out, row)
			}
		}
	}

	// ---- The Page ----
	if s.PageID != "" {
		posts, err := client.PagePosts(ctx, s.PageID, 24)
		if err != nil {
			h.noteMetaError(ctx, err)
			// ⚠️ Only an error when nothing at all came back: a restaurant
			// whose Instagram posts loaded does not need a banner about the
			// Page, and one with neither needs to know why.
			if len(out) == 0 {
				if e := meta.AsError(err); e != nil {
					httpx.Error(w, http.StatusBadGateway, e.Human())
					return
				}
				httpx.Error(w, http.StatusBadGateway, err.Error())
				return
			}
		}
		for _, p := range posts {
			// A post with no picture makes a poor advert and Meta often
			// refuses it outright — left out rather than offered and refused.
			if strings.TrimSpace(p.FullPicture) == "" {
				continue
			}
			out = append(out, adsPost{
				ID: p.ID, Source: "page", Image: p.FullPicture,
				Caption: p.Message, Link: p.Permalink, At: p.CreatedTime,
			})
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"connected": s.PageID != "" || s.InstagramID != "",
		"posts":     nonNil(out),
	})
}

// instagramID resolves the Instagram account to advertise as, and remembers it.
//
// ⚠️ **Two edges for one fact.** The Page's `instagram_business_account` is the
// usual answer and is sometimes empty while the account is perfectly reachable
// from the ad account. An install that asked only the first showed an owner an
// empty grid and no reason for it.
func (h *Handler) instagramID(
	ctx context.Context, client *meta.Client, stored, act string,
) (string, error) {
	if id := strings.TrimSpace(stored); id != "" {
		return id, nil
	}
	if act == "" {
		return "", nil
	}
	id, err := client.ConnectedInstagram(ctx, act)
	if err != nil {
		return "", err
	}
	if id != "" {
		// Written down so the next screen does not ask Meta again — one call of
		// somebody's hourly ceiling for a fact that does not change.
		_ = h.saveAds(ctx, bson.M{"instagramId": id})
	}
	return id, nil
}
