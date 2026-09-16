package handlers

// ---- The advertising section ----
//
// ⚠️ **The restaurant's money is spent at Meta, never by us.** The ad account is
// theirs, the card on it is theirs, and every so'm of ad spend is billed to them
// directly. What this section sells is the work around it: deciding what to
// advertise from figures this system already holds, writing the campaign, and
// saying afterwards what it brought back in orders rather than in clicks.
//
// ⚠️ **What is here today is the door, not the room.** Connecting a Meta account
// needs an app that has passed Meta's review; until that exists the honest
// answer is "not connected", said plainly, with the part that works — the plan
// built from the restaurant's own numbers — in front of it. A screen that
// pretended to be connected would be discovered at the worst possible moment: by
// somebody who had just pressed a button expecting an advert to run.
//
// See docs/reklama-reja.md for the phases and docs/vendor/meta-marketing.md for
// the API as actually read.

import (
	"net/http"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// AdminAdsState is what the advertising screen needs before it draws anything.
//
// ⚠️ **Three separate facts, and collapsing them would be the bug.** "Not
// bought", "no platform behind this install" and "bought but no Meta account
// connected" send the owner to three different places — a sales conversation,
// nowhere at all, and a connect button. The advisor tab learned this the
// expensive way: every failure became one empty screen that explained nothing.
func (h *Handler) AdminAdsState(w http.ResponseWriter, r *http.Request) {
	// ⚠️ Owner only, like the campaign button next door: this section connects
	// an account that can spend money, and a manager who could connect one
	// could spend from it.
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	ctx := r.Context()

	// ⚠️ **Read from the mirrored grant, never worked out here.** What a
	// restaurant bought is the console's answer (models/subscription.go); a
	// second opinion in this file is the one that disagrees on the day somebody
	// upgrades.
	entitled := h.subscription(ctx).Has(models.ModAds)

	// A self-hosted restaurant with no platform behind it has no assistant and
	// no Meta app to connect through. Answered as a fact rather than an error:
	// the screen then says so instead of drawing a failure.
	linked := h.Cfg.ControlURL != "" && h.Cfg.ControlToken != ""

	httpx.JSON(w, http.StatusOK, map[string]any{
		"entitled": entitled,
		"on":       linked,
		// ⚠️ Hard-coded false until the connect flow exists, and deliberately
		// still *sent*: the screen is written against the field it will use, so
		// the day the flow lands nothing on the panel has to change shape.
		"connected": false,
	})
}
