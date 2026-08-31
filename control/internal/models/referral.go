package models

import (
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Who sent us a customer we could not have reached.
//
// ⚠️ **This is not the same thing as an agent, and folding them together loses
// the fact that matters.** An agent works here: they have a console login, they
// plan visits, and `tenant.createdById` says which of them signed a customer up.
// A referrer is a business outside — the firm that sells fiscal registers, the
// packaging supplier, the accountant who keeps twelve restaurants' books. They
// walk into twenty kitchens a week, they are not competing with us, and they
// have no interest in a login. Both can be true of one customer: an agent
// closed the deal, the register firm sent it.
//
// ⚠️ **It exists because a referral you cannot count is a referral you cannot
// pay for**, and a partner who is not paid the second month stops sending
// anybody. The arrangement is only worth making if the arithmetic is written
// down where both sides can read it.
//
// ⚠️ Named `Referrer` rather than `Partner` because `Partners` is already taken
// by the landing page's showcase — the logos of payment providers and till
// systems we integrate with. Two very different things called the same word in
// one codebase is a bug waiting for whoever reads it second.
type Referrer struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`

	// What to call them on an invoice and in conversation.
	Name string `bson:"name" json:"name"`

	// The code in the link they hand out: keel.uz/h/<code>.
	//
	// ⚠️ **Short and sayable, because it travels by voice.** A register
	// engineer standing in a kitchen says "keel.uz slash h slash fiskal" — a
	// UUID does not survive that trip, and a code nobody can repeat is a code
	// that never gets used.
	Code string `bson:"code" json:"code"`

	// A phone number or a Telegram handle. Free text: half of these
	// arrangements are with a person, not a company.
	Contact string `bson:"contact,omitempty" json:"contact,omitempty"`
	Note    string `bson:"note,omitempty" json:"note,omitempty"`

	// What they earn: this share of what the customers they sent **actually
	// paid us**, for the first `Months` months of that customer's subscription.
	//
	// ⚠️ **Of money collected, not of money invoiced.** A commission on an
	// invoice pays out on a bill that may never be settled, which means the
	// referrer is paid for a customer we are chasing — and the one month that
	// happens is the month there is no money to pay it with.
	//
	// ⚠️ **Bounded in time on purpose.** A percentage for ever turns every
	// customer into a permanent cost and makes the arrangement impossible to
	// end without an argument. Zero months means no limit, and it is a choice
	// somebody has to type.
	Percent int `bson:"percent" json:"percent"`
	Months  int `bson:"months" json:"months"`

	IsActive bool `bson:"isActive" json:"isActive"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// ⚠️ The same alphabet a slug uses, and for the same reason: this ends up in a
// URL that gets printed on paper and read out over a phone.
var referrerCodeRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,23}$`)

// NormalizeReferrerCode lowercases and trims a code as typed.
//
// ⚠️ Case is dropped rather than preserved: the code is read off a leaflet by
// somebody typing on a phone keyboard, and "Fiskal" failing where "fiskal"
// works is a support message about a link that does not work.
func NormalizeReferrerCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

// ValidReferrerCode reports whether a code can be put in a link.
func ValidReferrerCode(code string) bool {
	return referrerCodeRe.MatchString(code)
}

// CommissionWindow is the last day a customer's payments still earn commission,
// given when they started paying. The zero time means "no limit".
//
// ⚠️ Counted from `subscribedAt` and not from the day the tenant was created: a
// trial pays nothing, so a commission window that starts at signup would be
// half spent before the first invoice exists.
func (rf Referrer) CommissionWindow(subscribedAt *time.Time) time.Time {
	if rf.Months <= 0 || subscribedAt == nil {
		return time.Time{}
	}
	return subscribedAt.AddDate(0, rf.Months, 0)
}

// Commission is what this referrer earns on one collected payment.
//
// ⚠️ **Rounded down.** Rounding a commission up pays out money that was never
// collected, one so'm at a time, and the ledger stops balancing for a reason
// nobody can find.
func (rf Referrer) Commission(collected int) int {
	if rf.Percent <= 0 || collected <= 0 {
		return 0
	}
	return collected * rf.Percent / 100
}
