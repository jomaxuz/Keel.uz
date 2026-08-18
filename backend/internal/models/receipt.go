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

	// The printers this branch has, and which receipts go to each.
	//
	// ⚠️ **A list, not three fields.** A restaurant with one printer at the
	// counter and one at the pass is the common case, but a big kitchen has a
	// second one at the grill and a bar has its own — and the receipt that goes
	// to each is a property of the printer, not of the document.
	Printers []Printer `bson:"printers,omitempty" json:"printers"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// Printer is one machine and what it prints.
type Printer struct {
	ID   string `bson:"id" json:"id"`
	Name string `bson:"name" json:"name"`

	// Where it is, in one line the owner pastes from the printer's self-test
	// page or from Windows. See internal/printer.Parse for what is accepted:
	// tcp://192.168.1.50:9100 · usb://XP-58 · \\PC\XP-58 · serial://COM3 ·
	// device:///dev/usb/lp0
	//
	// ⚠️ **One box, not a form of five.** The person setting this up is reading
	// a sticker, and the difference between one field and five is whether they
	// finish.
	Target string `bson:"target" json:"target"`

	// Which receipts this one prints: kitchen · till · customer · precheck.
	//
	// ⚠️ Empty means **nothing**, not everything: a printer somebody added and
	// has not finished configuring must not start printing every bill in the
	// building on the pass's roll.
	Kinds []string `bson:"kinds,omitempty" json:"kinds"`

	// "latin" (Uzbek) or "cyrillic" (Russian). ⚠️ The single most common way a
	// receipt comes out as a page of nonsense — the printer has no Unicode and
	// prints whatever page it is set to.
	Charset string `bson:"charset,omitempty" json:"charset,omitempty"`

	// Cut the paper, and kick the cash drawer. ⚠️ Both off by default: a
	// printer with no cutter **prints** the cut command instead of ignoring it,
	// and a drawer kick on a machine with no drawer is a click nobody wants.
	Cut     bool `bson:"cut,omitempty" json:"cut,omitempty"`
	FullCut bool `bson:"fullCut,omitempty" json:"fullCut,omitempty"`
	Drawer  bool `bson:"drawer,omitempty" json:"drawer,omitempty"`

	// How many copies. One, unless the kitchen wants a second for the pass.
	Copies int `bson:"copies,omitempty" json:"copies,omitempty"`

	// Off without being deleted — a printer that is broken this week.
	Disabled bool `bson:"disabled,omitempty" json:"disabled,omitempty"`
}

// Prints reports whether this printer is asked for this kind of receipt.
func (p Printer) Prints(kind string) bool {
	if p.Disabled || p.Target == "" {
		return false
	}
	for _, k := range p.Kinds {
		if k == kind {
			return true
		}
	}
	return false
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
