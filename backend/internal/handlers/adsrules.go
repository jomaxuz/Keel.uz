package handlers

// ---- The standing instruction: what happens while nobody is watching ----
//
// ⚠️ **Rules, not judgement.** Everything here is arithmetic against numbers
// the owner typed: a ceiling per day, a spend they are willing to lose without
// a single order, a price per order above which the campaign is not worth
// running. Nothing in this file decides anything a person did not decide first,
// and that is deliberate — a model that could raise a budget on its own reading
// of a week would be spending somebody else's money on its own opinion, and the
// first bad week would be ours rather than a choice they made.
//
// ⚠️ **It can stop, and it can move a budget downwards. Upwards only inside the
// ceiling.** The asymmetry is the whole safety property: the worst this engine
// can do is spend exactly what was authorised, and the best it can do is stop
// spending it sooner than a person would have noticed.
//
// ⚠️ **Every decision is written down and the owner is told.** An advert that
// stops with nobody told reads as our bug, and is found days later by an owner
// wondering why the orders fell off. The reason lives on the campaign, and the
// message goes out through the same channel as every other thing worth knowing
// now (models/alert.go).

import (
	"context"
	"fmt"
	"log"
	"time"

	"restaurant-backend/internal/meta"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// How far a budget may move in one step, and where the tuning thresholds sit.
//
// ⚠️ **Twenty per cent, not "to the ceiling".** Meta re-enters its learning
// phase when a budget moves sharply, which throws away what the campaign has
// learned and costs more than the change was worth. A fifth is the size the
// platform's own guidance treats as safe.
const (
	adsBudgetStep = 0.20
	// Below this share of the owner's ceiling on cost per order, the campaign
	// is doing well enough to be given more — within the ceiling.
	adsCheapAt = 0.60
	// Above this share, it is heading for the ceiling and the budget comes down
	// before it gets there.
	adsDearAt = 0.85
	// How many days of figures a decision is made on.
	//
	// ⚠️ **Three, not one.** One day is a rainy Tuesday, and a rule that paused
	// a working campaign because of one would be turned off within a week —
	// taking the rule that would have saved real money with it.
	adsRuleWindow = 3
)

// StartAdsRules applies the standing instruction after each sync.
//
// ⚠️ **Its own ticker, offset from the sync's.** The rules read what the sync
// wrote, and running them in the same loop would mean a failed sync silently
// becoming a skipped rule — which is the state where a campaign keeps spending
// because nothing looked at it.
func (h *Handler) StartAdsRules(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(adsSyncEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Minute):
			}
			h.applyAdsRules(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (h *Handler) applyAdsRules(ctx context.Context) {
	client, s, err := h.adsClient(ctx)
	if err != nil {
		return
	}
	if !s.Rules.On {
		// Off is the default and the ordinary state: a restaurant that has not
		// set a ceiling has authorised nothing for a rule to act within.
		return
	}
	cur, err := h.Store.AdsCampaigns.Find(ctx, bson.M{
		"status": models.AdCampaignActive,
	}, options.Find().SetLimit(100))
	if err != nil {
		log.Printf("ads rules: %v", err)
		return
	}
	var rows []models.AdCampaign
	if err := cur.All(ctx, &rows); err != nil {
		return
	}
	for _, row := range rows {
		if err := h.applyAdsRule(ctx, client, s, row); err != nil {
			log.Printf("ads rules %s: %v", row.MetaCampaignID, err)
		}
	}
}

// applyAdsRule decides what to do about one campaign.
func (h *Handler) applyAdsRule(
	ctx context.Context, client *meta.Client, s *models.AdsSettings, row models.AdCampaign,
) error {
	rules := s.Rules
	unit := meta.MinorUnits(row.Currency)

	// ---- The campaign has run its course ----
	//
	// ⚠️ **Checked first, and checked here as well as at Meta.** The ad set
	// carries an end time, but a campaign whose end was never set — an older
	// row, a hand-made change in Ads Manager — would otherwise run forever on
	// the owner's card with our screen calling it active.
	if row.EndsAt != nil && time.Now().After(*row.EndsAt) {
		return h.adsRuleAct(ctx, client, row, "stop",
			"kampaniya muddati tugadi", 0, 0)
	}

	// The window the decision is made on, in the account's own money.
	from := time.Now().In(time.Local).AddDate(0, 0, -adsRuleWindow).Format("2006-01-02")
	win := h.adsTotals(ctx, bson.M{
		"metaCampaignId": row.MetaCampaignID,
		"day":            bson.M{"$gte": from},
	})
	spent := int(win.Spend*float64(unit) + 0.5)
	if spent <= 0 {
		// Nothing has been spent yet, so there is nothing to judge. ⚠️ Not the
		// same as "it is doing badly": a campaign Meta is still reviewing
		// spends nothing and would otherwise be stopped for it.
		return nil
	}

	// ---- Money out, nothing back ----
	if rules.NoResultMinor > 0 && win.Purchases == 0 && spent >= rules.NoResultMinor {
		return h.adsRuleAct(ctx, client, row, "pause", fmt.Sprintf(
			"%d kunda %s %s sarflandi, birorta ham buyurtma bo'lmadi",
			adsRuleWindow, money(spent, unit), row.Currency), 0, 0)
	}
	if win.Purchases == 0 {
		// Spending, no orders yet, but under the threshold the owner set —
		// which is the state they explicitly said to leave alone.
		return nil
	}

	// ---- What one order is costing ----
	cost := spent / win.Purchases
	if rules.MaxCostMinor > 0 && cost > rules.MaxCostMinor {
		return h.adsRuleAct(ctx, client, row, "pause", fmt.Sprintf(
			"bitta buyurtma %s %s ga tushdi — chegara %s",
			money(cost, unit), row.Currency, money(rules.MaxCostMinor, unit)), 0, 0)
	}

	// ---- Tuning, and only within what was authorised ----
	if !rules.Tune || rules.MaxCostMinor <= 0 {
		return nil
	}
	ceiling := row.CapMinor
	if rules.MaxDailyMinor > 0 && (ceiling == 0 || rules.MaxDailyMinor < ceiling) {
		// ⚠️ **The lower of the two ceilings, always.** One is this campaign's,
		// one is the section's, and a rule that took the higher would turn the
		// stricter setting into decoration the first time the two disagreed.
		ceiling = rules.MaxDailyMinor
	}
	share := float64(cost) / float64(rules.MaxCostMinor)

	switch {
	case share <= adsCheapAt && ceiling > row.DailyMinor:
		next := int(float64(row.DailyMinor) * (1 + adsBudgetStep))
		if next > ceiling {
			next = ceiling
		}
		if next <= row.DailyMinor {
			return nil
		}
		return h.adsRuleAct(ctx, client, row, "budget", fmt.Sprintf(
			"bitta buyurtma %s %s — chegaradan ancha past",
			money(cost, unit), row.Currency), row.DailyMinor, next)

	case share >= adsDearAt:
		next := int(float64(row.DailyMinor) * (1 - adsBudgetStep))
		// ⚠️ Never below Meta's own floor: an ad set under it is refused, and
		// the refusal would leave the budget where it was — a rule that reports
		// having acted and did not.
		if s.MinDailyBudget > 0 && next < s.MinDailyBudget {
			next = s.MinDailyBudget
		}
		if next >= row.DailyMinor {
			return nil
		}
		return h.adsRuleAct(ctx, client, row, "budget", fmt.Sprintf(
			"bitta buyurtma %s %s — chegaraga yaqinlashdi",
			money(cost, unit), row.Currency), row.DailyMinor, next)
	}
	return nil
}

// adsRuleAct carries out one decision and records it.
//
// ⚠️ **Meta first, our record second.** A row saying "paused" over a campaign
// that is still spending is the one state this section must never reach: the
// screen would say the money had stopped while the card kept being charged.
func (h *Handler) adsRuleAct(
	ctx context.Context, client *meta.Client, row models.AdCampaign,
	action, why string, from, to int,
) error {
	set := bson.M{}
	switch action {
	case "pause", "stop":
		if err := h.adsSwitch(ctx, client, row, false); err != nil {
			return err
		}
		if action == "stop" {
			set["status"] = models.AdCampaignStopped
			set["stoppedAt"] = time.Now()
		} else {
			set["status"] = models.AdCampaignPaused
		}
	case "budget":
		if err := client.SetDailyBudget(ctx, row.AdSetID, to); err != nil {
			return err
		}
		set["dailyMinor"] = to
	default:
		return nil
	}

	decision := models.AdDecision{
		At: time.Now(), Action: action, Why: why, From: from, To: to,
	}
	_, err := h.Store.AdsCampaigns.UpdateByID(ctx, row.ID, bson.M{
		"$set":  set,
		"$push": bson.M{"decisions": bson.M{"$each": []any{decision}, "$slice": -50}},
	})
	if err != nil {
		return err
	}

	// ⚠️ **Only a stop is worth a message.** A budget nudged by a fifth inside
	// the owner's own ceiling is the rules doing exactly what was asked, and a
	// notification for it every six hours is how the channel carrying the ones
	// that matter gets muted.
	if action == "pause" || action == "stop" {
		h.raiseAlert(models.LossAlert{
			BranchID: row.BranchID,
			Kind:     models.AlertAdsPaused,
			At:       time.Now(),
			Subject:  row.Name,
			Reason:   why,
			RefID:    row.ID,
		})
	}
	return nil
}
