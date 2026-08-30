package handlers

// ---- Writing the message, not sending it ----
//
// ⚠️ **The assistant never sends.** It proposes text; the owner reads it and
// presses send. A campaign that went out on its own would be a bill and a
// reputation, and neither comes back — the reputation least of all, because the
// people who received it are exactly the regulars it was written for.
//
// ⚠️ **Nothing personal leaves the restaurant here either.** The request says
// which segment and how many people are in it, never who they are. The message
// is written for a group and delivered to it by the machine already holding the
// list.

import (
	"errors"
	"net/http"
	"strings"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// AdminCampaignText proposes what to say to a segment.
func (h *Handler) AdminCampaignText(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		Segment string `json:"segment"`
		Channel string `json:"channel"`
		Offer   string `json:"offer"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	channel := campaignChannel(req.Channel)
	segment := strings.TrimSpace(req.Segment)

	// How many people, so the model writes for forty rather than for an unknown
	// number — and so the owner sees the cost of the send beside the text.
	list, _, _, err := h.audience(r.Context(), segment, channel)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	res, err := h.callControlPath(r.Context(), "/internal/campaign-text", map[string]any{
		"lang":       reportLang(r),
		"segment":    segment,
		"channel":    channel,
		"people":     len(list),
		"restaurant": h.restaurantName(r.Context()),
		// ⚠️ **The offer is the owner's words, passed through and never
		// invented.** A discount the assistant made up is one a guest arrives
		// expecting, and the argument at the till is with a cashier who has
		// never heard of it.
		"offer": strings.TrimSpace(req.Offer),
		// ⚠️ The budget, spelled out for the channel it is actually for. See
		// the note on `charBudget`.
		"maxChars": charBudget(channel),
	})
	if errors.Is(err, ErrNotLinked) {
		// ⚠️ **Not an error, the way the briefing already answers.** A server
		// with no platform behind it simply has no assistant; the campaign
		// screen works exactly as it did before, and the owner writes their own
		// message. A 502 here made a missing feature look like a broken one.
		httpx.JSON(w, http.StatusOK, map[string]any{
			"variants": []any{}, "off": true, "people": len(list), "channel": channel,
		})
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}

	variants, _ := res["variants"].([]any)
	out := make([]map[string]any, 0, len(variants))
	for _, v := range variants {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		text := strings.TrimSpace(str(m["text"]))
		if text == "" {
			continue
		}
		item := map[string]any{"text": text, "chars": len([]rune(text))}
		if channel == models.CampaignSMS {
			// ⚠️ **Counted with the function the invoice uses**, never with a
			// character limit written here. Uzbek Latin's apostrophes and any
			// Cyrillic at all fall outside GSM-7, where one part is seventy
			// characters and not a hundred and sixty — so a limit guessed in
			// this file would have quietly doubled the cost of every send, and
			// the only place it would show is a bill.
			item["parts"] = smsParts(text)
		}
		out = append(out, item)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"variants": out, "people": len(list), "channel": channel,
		"entitled": res["entitled"], "capped": res["capped"],
		"monthly": res["monthly"],
		// ⚠️ **Forwarded, and it was the missing half of the bug.** The platform
		// already answers `off: true` when it has no key configured — the same
		// shape the briefing uses — and this handler passed on three flags and
		// dropped that one. So a restaurant on a platform with no key pressed
		// "write me three messages", got an empty list with no reason, and the
		// panel fell through to "nothing was written". The reason existed the
		// whole way down and was thrown away one hop from the screen.
		"off": res["off"],
	})
}

// charBudget is roughly how long a message on this channel should be.
//
// ⚠️ **Advice to the model, not a rule enforced on the answer.** The rule that
// matters is `smsParts`, applied above to whatever comes back — a model told to
// stay under seventy characters usually does, and "usually" is a second part on
// a thousand-person send. Giving it the number anyway makes the usual case
// right, which is cheaper than making the unusual case visible.
func charBudget(channel string) int {
	switch channel {
	case models.CampaignTelegram:
		return 400
	case models.CampaignPush:
		// A notification is truncated by the phone, not by us, and where it cuts
		// depends on the device. Short enough that no device cuts it.
		return 120
	default:
		// One SMS part in the alphabet these messages are actually written in.
		return 70
	}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
