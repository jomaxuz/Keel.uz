package models

import (
	"time"

	"restaurant-backend/internal/receipt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ReceiptSettings is one branch's three receipt designs.
//
// ⚠️ **Per branch, because the printer is.** Paper width is a fact about the
// machine on that counter, and a chain whose second kitchen bought 58 mm rolls
// would otherwise print half a receipt there every time somebody edited the
// design at the first. The wording usually matches across a chain, so the panel
// offers to copy it — but the storage follows the hardware.
//
// The restaurant's name, address and phone are **not** here: they already exist
// on the brand and the branch, and a second copy would be a second thing to
// update when the phone number changes. The template only decides whether they
// are printed.
type ReceiptSettings struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	BranchID primitive.ObjectID `bson:"branchId" json:"branchId"`

	// ⚠️ Three separate templates, not one with fields switched off. They are
	// read by three different people under three different pressures — see the
	// package comment on internal/receipt.
	Kitchen  receipt.Template `bson:"kitchen" json:"kitchen"`
	Till     receipt.Template `bson:"till" json:"till"`
	Customer receipt.Template `bson:"customer" json:"customer"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// DefaultReceipts is what a branch starts with.
//
// ⚠️ **All three enabled, 80 mm, with sensible fields on.** The zero value of
// this document is "print nothing", which for a restaurant that has just
// connected a printer is indistinguishable from a broken printer — and the
// first thing they would do is call us. Defaults that print something are the
// only ones that can be debugged from the paper.
func DefaultReceipts(branchID primitive.ObjectID) ReceiptSettings {
	base := func() receipt.Template {
		return receipt.Template{
			Enabled: true, WidthMM: 80, FeedLines: 3,
			Fields: map[string]bool{},
		}
	}
	k := base()
	// The pass needs the time it was fired and who fired it; nothing else.
	k.Fields = map[string]bool{"comment": true, "time": true, "server": true}

	till := base()
	till.Fields = map[string]bool{"time": true, "cashier": true, "change": true}

	cust := base()
	cust.Footer = "Rahmat! Yana kutamiz."
	cust.Fields = map[string]bool{
		"time": true, "server": true, "address": true, "phone": true, "change": true,
	}

	return ReceiptSettings{
		BranchID: branchID, Kitchen: k, Till: till, Customer: cust,
		UpdatedAt: time.Now(),
	}
}
