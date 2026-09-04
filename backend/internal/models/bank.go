package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- The bank account, as far as we can honestly know it ----
//
// ⚠️ **We do not know the bank balance, and the screen must never pretend to.**
// Money reaches that account from rails this system records (an aggregator's
// transfer, an acquirer's settlement) and from a dozen it does not: an owner's
// own deposit, a loan, a tax refund, a transfer between the company's own
// accounts. Deriving a balance from the movements we happen to see would be a
// confident figure that is wrong by everything we cannot see — and wrong in a
// way nobody could spot, because it would look like a balance.
//
// So the shape is the one the stock takes: **a counted figure with a date on
// it**, plus what has moved since. The owner reads the balance off their bank
// app and writes it down; the screen says "12 300 000 as of 3 September, and
// these arrived after it". A number somebody counted, and a number somebody can
// check it against — never a number this system invented.
type BankBalance struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`

	// Which account, in the restaurant's own words ("Ipoteka bank, so'm").
	// Free text: this is read by a person reconciling a statement, not by code.
	Account string `bson:"account" json:"account"`
	Amount  int    `bson:"amount" json:"amount"`
	// When the balance was true — not when it was typed.
	At   time.Time `bson:"at" json:"at"`
	Note string    `bson:"note,omitempty" json:"note,omitempty"`

	By        string    `bson:"by,omitempty" json:"by,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

// MoneyPlace is one place the restaurant's money physically or legally sits.
type MoneyPlace struct {
	// "safe" | "drawer" | "courier" | "advance" | "bank" | "rail"
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Amount int    `json:"amount"`
	// A second line for the screen: how old the figure is, who holds it, what
	// it is waiting on.
	Note string `json:"note,omitempty"`
	// ⚠️ Whether this figure was **counted** (a drawer at close, a balance read
	// off a bank app) or **added up from documents** (the safe's ledger, a
	// rail's unsettled sales). The screen says which, because they fail
	// differently: a counted figure goes stale, a summed one goes wrong.
	Counted bool       `json:"counted,omitempty"`
	At      *time.Time `json:"at,omitempty"`
}

// MoneyPosition is the whole answer to "where is our money".
//
// ⚠️ **Three totals, never one.** Cash in a box, money in a bank account and
// money an aggregator is still holding are three different kinds of having: one
// can be spent tonight, one this week, one when somebody else decides. A single
// "we have X" would be the most quotable and least true number on the platform.
type MoneyPosition struct {
	Cash  []MoneyPlace `json:"cash"`
	Bank  []MoneyPlace `json:"bank"`
	Rails []MoneyPlace `json:"rails"`

	CashTotal  int `json:"cashTotal"`
	BankTotal  int `json:"bankTotal"`
	RailsTotal int `json:"railsTotal"`

	// The bank-agreed ceiling on cash held overnight, and whether it is
	// exceeded. See models/branch cash limit and docs/DECISIONS.md →
	// "Inkassatsiya".
	CashLimit int  `json:"cashLimit,omitempty"`
	OverLimit bool `json:"overLimit,omitempty"`
	// Which branch that ceiling belongs to, so the screen can save a new one
	// without making a single-branch restaurant choose a lens.
	//
	// ⚠️ A zero id marshals as "000…0", which is truthy in the browser — the
	// screen must use hasId(), not the bare value.
	BranchID primitive.ObjectID `json:"branchId,omitempty"`
}
