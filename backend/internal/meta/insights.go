package meta

// ---- What the money bought, one day at a time ----
//
// ⚠️ **Days, not totals.** Meta will happily return one aggregated row for any
// window, and a section whose whole promise is "what did the advertising bring
// back" cannot answer "which day did it stop working" from one number. Asking
// with `time_increment=1` costs the same call and gives a history we can keep.
//
// ⚠️ **The pixel's purchases, never the omni ones.** `purchase` counts orders
// Meta believes happened anywhere, including in shops we have never heard of;
// `offsite_conversion.fb_pixel_purchase` counts the ones our own server
// reported through the Conversions API — the same orders that are in this
// restaurant's database, with the same ids. Mixing the two produces a report
// whose order count does not match the order list, which is the one comparison
// an owner will certainly make.

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// The action Meta names a pixel purchase with, and the value beside it.
const purchaseAction = "offsite_conversion.fb_pixel_purchase"

// Day is one campaign's one day, as Meta reported it.
type Day struct {
	Date string
	// ⚠️ In the **ad account's** currency, which is usually not the som. The
	// panel shows it with the currency Meta named, never converted: we do not
	// know the rate the owner's bank used, and inventing one would put a made-up
	// number beside real ones.
	Spend       float64
	Impressions int
	Clicks      int
	// Purchases and their value as *reported by us* through the pixel, in the
	// currency our own events carried (so'm).
	Purchases int
	Revenue   float64
}

// metaNumber is every numeric field Meta sends, which is to say a string.
type metaNumber string

func (m metaNumber) f() float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(string(m)), 64)
	return f
}
func (m metaNumber) i() int { return int(m.f() + 0.5) }

type actionRow struct {
	ActionType string     `json:"action_type"`
	Value      metaNumber `json:"value"`
}

type insightRow struct {
	DateStart    string      `json:"date_start"`
	Spend        metaNumber  `json:"spend"`
	Impressions  metaNumber  `json:"impressions"`
	Clicks       metaNumber  `json:"clicks"`
	Actions      []actionRow `json:"actions"`
	ActionValues []actionRow `json:"action_values"`
}

func pick(rows []actionRow, kind string) metaNumber {
	for _, r := range rows {
		if r.ActionType == kind {
			return r.Value
		}
	}
	return ""
}

// Insights reads one campaign, day by day, over a closed window.
//
// ⚠️ **Only the fields we draw.** Meta charges the quota by what is asked for,
// and an insights call with everything on it is slower and counted heavier —
// on a Limited-tier account that is 600 calls an hour for the whole restaurant.
func (c *Client) Insights(ctx context.Context, objectID string, since, until time.Time) ([]Day, error) {
	window, _ := json.Marshal(map[string]string{
		"since": since.Format("2006-01-02"),
		"until": until.Format("2006-01-02"),
	})
	var out listOf[insightRow]
	err := c.Get(ctx, objectID+"/insights", url.Values{
		"fields":         {"spend,impressions,clicks,actions,action_values"},
		"time_increment": {"1"},
		"time_range":     {string(window)},
		"limit":          {"100"},
	}, &out)
	if err != nil {
		return nil, err
	}
	days := make([]Day, 0, len(out.Data))
	for _, r := range out.Data {
		days = append(days, Day{
			Date:        r.DateStart,
			Spend:       r.Spend.f(),
			Impressions: r.Impressions.i(),
			Clicks:      r.Clicks.i(),
			Purchases:   pick(r.Actions, purchaseAction).i(),
			Revenue:     pick(r.ActionValues, purchaseAction).f(),
		})
	}
	return days, nil
}
