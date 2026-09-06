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

	// The shop's label design.
	//
	// ⚠️ **Here rather than on the branch**, because a label is a piece of paper
	// coming out of a machine on that counter — the same fact that puts the
	// paper width and the printers in this document. A shop that changed its
	// design would otherwise have to find it on a settings page that is about
	// opening hours and delivery zones.
	//
	// ⚠️ **Absent is the shelf label at 58 mm**, not "print nothing": every
	// branch created before this field existed has no entry here. See
	// LabelTemplate.Defaults.
	Label receipt.LabelTemplate `bson:"label,omitempty" json:"label"`

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

	// Which sections of the menu this printer takes, by category id.
	//
	// ⚠️ **Empty means every category, and that is the opposite of `Kinds`
	// above.** The reasons are opposite too: a printer with no kinds chosen is
	// one somebody has not finished setting up, and printing everything on it
	// would be a surprise. A printer with no categories chosen is every
	// restaurant that exists today — one kitchen printer taking all the food —
	// and reading that as "nothing" would stop every kitchen ticket in the
	// product on the day this shipped.
	//
	// ⚠️ **Categories, not dishes.** A restaurant has two or three printers and
	// twenty categories; setting this per dish means visiting two hundred of
	// them, and the setting that takes an afternoon is the setting nobody
	// finishes. `Except` below is for the handful that do not follow their
	// section.
	Categories []string `bson:"categories,omitempty" json:"categories,omitempty"`

	// Dishes that go here whatever their category says, and dishes that never
	// do.
	//
	// ⚠️ **The exceptions are why this is usable at all.** Every menu has a few:
	// the dessert that comes off the bar's ice cream machine, the soup the
	// grill section makes. Without them a restaurant has to reorganise its menu
	// to match its printers, which is the tail wagging the dog.
	Only   []string `bson:"only,omitempty" json:"only,omitempty"`
	Except []string `bson:"except,omitempty" json:"except,omitempty"`

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
// Takes reports whether this printer should be given a particular dish.
//
// ⚠️ **Order matters and it is the order somebody would say out loud**: never
// this dish, always this dish, otherwise this section. An exception that could
// be overridden by a category would not be an exception.
func (p Printer) Takes(menuItemID, categoryID string) bool {
	for _, id := range p.Except {
		if id == menuItemID {
			return false
		}
	}
	for _, id := range p.Only {
		if id == menuItemID {
			return true
		}
	}
	// ⚠️ No categories chosen is every category — see the field's note. This
	// is the line that keeps every existing restaurant printing.
	if len(p.Categories) == 0 {
		return true
	}
	for _, id := range p.Categories {
		if id == categoryID {
			return true
		}
	}
	return false
}

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
		Label:     receipt.LabelTemplate{}.Defaults(),
		UpdatedAt: time.Now(),
	}
}
