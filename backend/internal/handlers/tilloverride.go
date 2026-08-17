package handlers

import (
	"context"
	"errors"
	"net/http"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// The manager's PIN: what makes restrictive permissions survive contact with a
// Friday evening.
//
// # ⚠️ Why refusing is the wrong answer
//
// A waiter needs to write off a burnt steak at nine o'clock and does not hold
// `void`. Refuse, and what happens next is not "they fetch the manager" — it is
// that somebody learns the manager's PIN, and within a fortnight the whole room
// knows it. After that every void in the journal carries one name, always the
// same name, and the attribution the PIN was built for is gone.
//
// So an action the person may not do is not refused. The screen asks for a code
// from somebody who may, the action proceeds, and **both names are recorded**:
//
//	Aziz olib tashladi · Dilnoza tasdiqladi
//
// That sentence is the whole feature. It is more informative than either name
// alone, and it is what the restaurant actually wants to know a month later.
//
// # ⚠️ Verified here, never on the screen
//
// The browser sends a PIN, not a verdict. A client that could say "a manager
// approved this" would be a client that can approve anything.

// actor is who an action is recorded against: the person who did it, and — when
// they needed permission — the person who allowed it.
type actor struct {
	ByID primitive.ObjectID
	By   string
	// Empty unless somebody else's code was used.
	AuthByID primitive.ObjectID
	AuthBy   string
}

// errNeedsOverride asks the screen for a manager's code.
//
// ⚠️ A distinct error rather than a plain 403, because the two mean opposite
// things to the person holding the tablet: one is "fetch somebody", the other
// is "this cannot be done at all". A screen that cannot tell them apart shows
// the wrong one, and the wrong one here is the one that teaches people to share
// codes.
var errNeedsOverride = errors.New("bu amal uchun ruxsat kerak")

// resolveActor decides who this action belongs to.
//
// Three outcomes, and the middle one is the point:
//
//   - the person holds the permission → it is theirs alone;
//   - they do not, and sent a code that does → theirs, authorised by the other;
//   - they do not, and sent nothing usable → errNeedsOverride, so the screen
//     can ask rather than give up.
func (h *Handler) resolveActor(
	ctx context.Context, s models.Staff, perm, pin string,
) (actor, error) {
	if s.Can(perm) {
		return actor{ByID: s.ID, By: s.Name}, nil
	}
	if pin == "" {
		return actor{}, errNeedsOverride
	}
	if err := pinShape(pin); err != nil {
		return actor{}, errNeedsOverride
	}

	// ⚠️ **The same branch, always.** Without it a code that happens to match a
	// manager somewhere else in the chain would authorise a write-off here, and
	// the journal would name somebody who was in another city.
	other, found, err := h.staffByPIN(ctx, s.BranchID, pin)
	if err != nil {
		return actor{}, err
	}
	if !found {
		return actor{}, errNeedsOverride
	}
	h.withRole(ctx, &other)
	if !other.Can(perm) {
		// ⚠️ Reported as "needs permission" rather than "wrong code": the code
		// was right and the person simply may not do this either. Saying "wrong
		// PIN" would send them to try it again, and then to try somebody else's.
		return actor{}, errNeedsOverride
	}
	// ⚠️ **The doer stays the doer.** Recording the manager as having done it
	// would lose the only fact worth keeping — that somebody who could not do
	// this asked for it — and would quietly blame the manager for every
	// write-off in the building.
	return actor{
		ByID: s.ID, By: s.Name,
		AuthByID: other.ID, AuthBy: other.Name,
	}, nil
}

// overrideDenied writes the "ask a manager" reply.
//
// ⚠️ **409, not 403.** A 403 is final and the screens in this codebase treat it
// that way; this is a request for a second person, and the status has to be one
// the client can act on rather than report.
func overrideDenied(w http.ResponseWriter, perm string) {
	httpx.JSON(w, http.StatusConflict, map[string]any{
		"error":          errNeedsOverride.Error(),
		"needsOverride":  true,
		"permission":     perm,
		"permissionName": permLabel(perm),
	})
}

// permLabel names a permission in the sentence the till shows: "Chegirma berish
// uchun ruxsat kerak".
//
// ⚠️ Named rather than shown as its identifier. `discount` on a screen in a
// restaurant is a word from our database, and the person reading it is holding
// plates.
func permLabel(perm string) string {
	switch perm {
	case models.PermVoid:
		return "Pishirilgan taomni olib tashlash"
	case models.PermDiscount:
		return "Chegirma berish"
	case models.PermShift:
		return "Kassa smenasi"
	case models.PermCashier:
		return "To'lovni qabul qilish"
	case models.PermWaiter:
		return "Zal ekrani"
	case models.PermKitchen:
		return "Oshxona ekrani"
	}
	return perm
}
