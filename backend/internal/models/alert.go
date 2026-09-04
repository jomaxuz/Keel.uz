package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Telling the owner, while it still matters ----
//
// ⚠️ **The failure mode of this whole feature is that it works too often.**
// A message that arrives most days is a message that gets muted, and the mute
// is not selective — the one that mattered is muted with the rest. Everything
// here is shaped by that: few kinds, thresholds the restaurant sets, a daily
// ceiling, and a bar deliberately set where an honest week produces nothing.
//
// ⚠️ **Patterns do not belong here.** Somebody's void rate being double their
// colleagues' is a real finding and a terrible notification — it is only true
// across a month, it has ordinary explanations, and a phone buzzing about it at
// eight in the evening invites a conversation nobody has prepared for. Patterns
// go to the morning briefing, where they arrive with their numbers and the
// owner is sitting down. What arrives instantly is only what is unusual *as a
// single event*.
//
// ⚠️ **Never phrased as an accusation.** Every kind below has an ordinary
// explanation that happens weekly in a busy restaurant: a guest who complained
// after seeing the bill, a regular given something off, a till that came up
// short because somebody paid a courier from it. The message says what happened
// and who, and stops there.

// AlertKind is what happened.
type AlertKind string

const (
	// A line was taken off a check *after* the guest had been shown the total.
	//
	// ⚠️ **The strongest single-event signal a restaurant has**, because the
	// order of the two events is the whole content: the bill existed, the guest
	// saw it, and then it went down. Voiding before the precheck is ordinary
	// work — a wrong order, a changed mind — and is not raised.
	AlertVoidAfterPrecheck AlertKind = "void_after_precheck"

	// Money taken off a bill by a person, above what this restaurant considers
	// ordinary.
	AlertBigDiscount AlertKind = "big_discount"

	// A till closed with less in it than it should have had.
	AlertCashShort AlertKind = "cash_short"

	// A count that came up short by more than a threshold.
	AlertStockShort AlertKind = "stock_short"

	// A dish's tech card was edited to consume more than it did.
	//
	// ⚠️ **The one channel here that steals without touching money.** The card
	// says 200g and the kitchen puts in 150g; every portion leaves 50g
	// unaccounted for, the stock figures agree with the books perfectly —
	// because the books were changed to agree — and a count finds nothing,
	// since nothing is missing against a card that expects it gone. The only
	// moment it is visible is the moment the card is edited.
	AlertRecipeUp AlertKind = "recipe_up"

	// Something done in the panel that is worth knowing about tonight.
	//
	// ⚠️ **The people this covers are the ones nobody was watching.** Every
	// other kind here comes off the till or the store, where a cashier or a
	// storekeeper is doing something physical. An operator, a call-centre
	// worker or a manager sits in the panel, and the panel's journal recorded
	// them perfectly and told nobody — which on the evening it matters is the
	// same as not recording them.
	AlertPanelAction AlertKind = "panel_action"

	// A check ended without money, with food on it.
	//
	// ⚠️ **The case this whole feature was asked for, and the one it shipped
	// without.** "Take the cash, cancel the check as a mistake" is the first
	// thing anybody describes when asked how a cashier steals — and every
	// trigger was hung on the *close* path, which a cancelled check never
	// reaches. Six kinds of alert, and the headline one was missing.
	AlertCheckCancelled AlertKind = "check_cancelled"
)

// LossAlert is one thing worth telling the owner about now.
//
// ⚠️ **Stored whether or not it was delivered.** The record is the point;
// Telegram being unreachable, a chat never linked, or the daily ceiling already
// reached must not make the event disappear. The panel reads this list, and it
// is the list that survives somebody muting a bot.
type LossAlert struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	Kind     AlertKind          `bson:"kind" json:"kind"`
	At       time.Time          `bson:"at" json:"at"`

	// Who did the thing. ⚠️ Staff, never a guest — the same line this product
	// draws everywhere: attribution of work is the content, and a guest's
	// identity is never part of it.
	ByID primitive.ObjectID `bson:"byId,omitempty" json:"-"`
	By   string             `bson:"by,omitempty" json:"by,omitempty"`
	// Who allowed it, when somebody had to.
	AuthBy string `bson:"authBy,omitempty" json:"authBy,omitempty"`

	// What it was worth, in so'm. The number the message leads with.
	Amount int `bson:"amount" json:"amount"`
	// What was typed at the time, if anything was.
	Reason string `bson:"reason,omitempty" json:"reason,omitempty"`
	// Free context: a dish name, a table, a store.
	Subject string `bson:"subject,omitempty" json:"subject,omitempty"`
	// The order or count it came from, so the panel can open it.
	RefID primitive.ObjectID `bson:"refId,omitempty" json:"refId,omitempty"`
	// The check's own number, and the table it was on.
	//
	// ⚠️ **An owner reading this on a phone cannot open a panel to find out
	// which check it was**, and "6-stol" alone names a table that has had nine
	// checks today. The number is what somebody types into the search box an
	// hour later — it is the difference between a message you can act on and a
	// message you can only feel bad about.
	//
	// ⚠️ **Kept apart from `Subject` rather than glued into it**, because
	// `Subject` was carrying Uzbek words into Russian messages: a number and a
	// table are language-neutral facts, and the words around them belong in
	// the sentence that is written at send time.
	Number string `bson:"number,omitempty" json:"number,omitempty"`
	Table  string `bson:"table,omitempty" json:"table,omitempty"`
	// Whether the guest had already been shown the bill.
	//
	// ⚠️ A fact rather than a sentence, so it can be worded in whichever
	// language the group reads. It is the difference between a table that
	// changed its mind and a total that existed and then did not.
	AfterPrecheck bool `bson:"afterPrecheck,omitempty" json:"afterPrecheck,omitempty"`

	// Delivery, recorded rather than assumed.
	SentAt   *time.Time         `bson:"sentAt,omitempty" json:"sentAt,omitempty"`
	SendErr  string             `bson:"sendErr,omitempty" json:"sendErr,omitempty"`
	SeenAt   *time.Time         `bson:"seenAt,omitempty" json:"seenAt,omitempty"`
	SeenByID primitive.ObjectID `bson:"seenById,omitempty" json:"-"`
}

// AlertSettings is where a restaurant says what counts as unusual.
//
// ⚠️ **Thresholds, not our judgement.** A hundred thousand so'm off a bill is a
// rounding error in one restaurant and a week's profit in another, and a figure
// we chose would be wrong in one of them every single day — which is how a
// notification channel gets muted in its first week.
type AlertSettings struct {
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	Enabled  bool               `bson:"enabled" json:"enabled"`

	// A discount at or above this raises one. 0 uses the default.
	DiscountFrom int `bson:"discountFrom" json:"discountFrom"`
	// A till short by at least this much.
	CashShortFrom int `bson:"cashShortFrom" json:"cashShortFrom"`
	// A count short by at least this much, in money.
	StockShortFrom int `bson:"stockShortFrom" json:"stockShortFrom"`
	// Voids after a precheck are raised from this value up. A one-thousand-so'm
	// tea removed after the bill is a correction; a main course is a question.
	VoidFrom int `bson:"voidFrom" json:"voidFrom"`

	// ⚠️ **The ceiling, and it is the most important field here.** A bad night
	// — a broken till, a trainee, a genuine spree — would otherwise send forty
	// messages, and forty messages is silence. Past it, events are still
	// recorded and the panel still shows them; only the buzzing stops.
	DailyMax int `bson:"dailyMax" json:"dailyMax"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// Defaults, in so'm. Deliberately high: the first week of a notification
// channel decides whether it is ever read again, and a channel that starts
// quiet can be turned down. One that starts noisy is muted before anybody
// finds the setting.
const (
	DefaultDiscountFrom   = 100_000
	DefaultCashShortFrom  = 50_000
	DefaultStockShortFrom = 300_000
	DefaultVoidFrom       = 50_000
	DefaultAlertDailyMax  = 8
)

// DefaultStockCardWarnFrom is the covered-revenue share below which the morning
// briefing says the store cannot account for what is selling.
//
// ⚠️ **Deliberately not 100.** A restaurant that has carded the dishes carrying
// four fifths of its takings has a store worth reading, and telling it
// otherwise every morning is how the briefing teaches somebody to skim. The
// figure a restaurant can act on is "most of my money is now traceable", not
// "every dish is written up" — which nobody has ever finished.
const DefaultStockCardWarnFrom = 80

// WithDefaults fills in what was never set.
//
// ⚠️ A zero here is "never configured", not "alert on everything". Reading it
// the other way would make every restaurant that has not opened the settings
// page alert on every thousand-so'm tea — on the day this shipped.
func (s AlertSettings) WithDefaults() AlertSettings {
	if s.DiscountFrom <= 0 {
		s.DiscountFrom = DefaultDiscountFrom
	}
	if s.CashShortFrom <= 0 {
		s.CashShortFrom = DefaultCashShortFrom
	}
	if s.StockShortFrom <= 0 {
		s.StockShortFrom = DefaultStockShortFrom
	}
	if s.VoidFrom <= 0 {
		s.VoidFrom = DefaultVoidFrom
	}
	if s.DailyMax <= 0 {
		s.DailyMax = DefaultAlertDailyMax
	}
	return s
}
