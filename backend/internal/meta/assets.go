package meta

// ---- What the restaurant already owns at Meta ----
//
// Connecting is not one fact but five: a business portfolio, an ad account
// inside it, a Page the adverts are published as, an Instagram account beside
// that Page, and a pixel the orders are reported to. Meta holds all five and
// the owner usually cannot name any of them from memory — so the connect screen
// lists what the token can actually see and the owner points at it.
//
// ⚠️ **We never create any of them.** An ad account made by us would belong to
// a business portfolio of ours, and the restaurant's card could not be put on
// it. The account is theirs; we are a tool with a key to it.

import (
	"context"
	"net/url"
	"strings"
)

// Business is one business portfolio the token can see.
type Business struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AdAccount is one ad account, with the three facts that decide whether a
// campaign can run at all.
type AdAccount struct {
	// ⚠️ **Two ids, and they are not interchangeable.** `ID` is `act_123…`,
	// which is what every edge path wants; `AccountID` is the bare number, which
	// is what the owner reads in Ads Manager. Sending one where the other is
	// expected produces "unsupported get request" — Meta's least helpful error.
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Name      string `json:"name"`
	// The currency Meta bills this account in. ⚠️ **Almost never UZS**: Meta
	// does not list the som among ad-account currencies, so an Uzbek restaurant
	// is on a USD account. Everything about budgets follows from this — see
	// MinorUnits.
	Currency string `json:"currency"`
	// 1 is active. Anything else means no advert will run, whatever we create.
	AccountStatus int `json:"account_status"`
	DisableReason int `json:"disable_reason"`
	// Meta's own floor for a daily budget, in the account currency's minor
	// units. ⚠️ **The guard we actually trust**: it comes from the same system
	// that will reject the campaign, so a budget that passes it is a budget in
	// the units Meta expects — which is the one mistake in this file that costs
	// money rather than a screen.
	MinDailyBudget int `json:"min_daily_budget"`
	// Whether a card is on the account. An account without one creates
	// campaigns happily and runs none of them.
	FundingSource string `json:"funding_source"`
}

// Live says whether this account can actually carry a campaign today.
func (a AdAccount) Live() bool { return a.AccountStatus == 1 }

// Page is a Facebook Page the adverts would be published as.
type Page struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Instagram struct {
		ID string `json:"id"`
	} `json:"instagram_business_account"`
}

// Pixel is where orders are reported back to.
type Pixel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type listOf[T any] struct {
	Data []T `json:"data"`
}

// Businesses lists the portfolios this token was granted.
func (c *Client) Businesses(ctx context.Context) ([]Business, error) {
	var out listOf[Business]
	err := c.Get(ctx, "me/businesses",
		url.Values{"fields": {"id,name"}, "limit": {"50"}}, &out)
	return out.Data, err
}

const adAccountFields = "id,account_id,name,currency,account_status," +
	"disable_reason,min_daily_budget,funding_source"

// AdAccounts lists the ad accounts the token reaches.
//
// ⚠️ **Both edges, merged by id.** A restaurant's account can be owned by the
// portfolio (`owned_ad_accounts`) or merely shared with the person who
// connected (`me/adaccounts`), and an install that listed only one of the two
// showed an owner an empty screen while their account sat in the other. Asking
// twice costs one call on a screen opened once.
func (c *Client) AdAccounts(ctx context.Context, businessID string) ([]AdAccount, error) {
	seen := map[string]bool{}
	var all []AdAccount
	add := func(rows []AdAccount) {
		for _, a := range rows {
			if a.ID == "" || seen[a.ID] {
				continue
			}
			seen[a.ID] = true
			all = append(all, a)
		}
	}
	params := url.Values{"fields": {adAccountFields}, "limit": {"100"}}
	var mine listOf[AdAccount]
	err := c.Get(ctx, "me/adaccounts", params, &mine)
	add(mine.Data)
	if businessID != "" {
		var owned listOf[AdAccount]
		// ⚠️ A failure here is not a failure of the screen: a token with no
		// `business_management` still sees its own accounts above, and the
		// owner can pick one. Only the first error is worth returning.
		if e2 := c.Get(ctx, businessID+"/owned_ad_accounts", params, &owned); e2 == nil {
			add(owned.Data)
			err = nil
		}
	}
	if len(all) > 0 {
		return all, nil
	}
	return all, err
}

// Account reads one ad account on its own — used before every campaign, to
// re-read the currency and Meta's floor rather than trust what we stored.
func (c *Client) Account(ctx context.Context, actID string) (AdAccount, error) {
	var a AdAccount
	err := c.Get(ctx, actID, url.Values{"fields": {adAccountFields}}, &a)
	return a, err
}

// Pages lists the Pages, with the Instagram account attached to each.
func (c *Client) Pages(ctx context.Context) ([]Page, error) {
	var out listOf[Page]
	err := c.Get(ctx, "me/accounts",
		url.Values{
			"fields": {"id,name,instagram_business_account{id}"},
			"limit":  {"100"},
		}, &out)
	return out.Data, err
}

// Pixels lists the pixels on one ad account.
func (c *Client) Pixels(ctx context.Context, actID string) ([]Pixel, error) {
	var out listOf[Pixel]
	err := c.Get(ctx, actID+"/adspixels",
		url.Values{"fields": {"id,name"}, "limit": {"50"}}, &out)
	return out.Data, err
}

// ---- Money, in the units Meta counts it in ----

// zeroDecimal are the ad-account currencies with no minor unit at all.
//
// ⚠️ **This table is the difference between 50 000 and 500 000.** Meta takes
// every budget in the account currency's *minor* units, so `daily_budget=50000`
// is $500.00 on a dollar account and ₩50,000 on a won one. Getting it wrong
// does not fail: it creates a campaign that spends a hundred times what the
// owner typed, and Meta bills their card for it.
var zeroDecimal = map[string]bool{
	"JPY": true, "KRW": true, "VND": true, "CLP": true, "ISK": true,
	"PYG": true, "UGX": true, "RWF": true, "VUV": true, "XAF": true,
	"XOF": true, "XPF": true, "BIF": true, "DJF": true, "GNF": true,
	"KMF": true, "MGA": true, "COP": true, "TWD": true,
}

// MinorUnits is how many of Meta's units make one of the currency the owner
// reads on the screen.
//
// ⚠️ **A table, and a floor that checks it.** The table above is what the
// documentation lists; `AdAccount.MinDailyBudget` is what the account itself
// says, and a budget below that floor is refused before anything is created.
// So an entry missing here is caught as "budget too small" rather than as a
// hundredfold overspend — the failure lands on our side of the wire.
func MinorUnits(currency string) int {
	if zeroDecimal[strings.ToUpper(strings.TrimSpace(currency))] {
		return 1
	}
	return 100
}
