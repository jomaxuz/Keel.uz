package handlers

import (
	"strings"
	"testing"
)

// ⚠️ The delivery board never shows the dining room — not behind a filter,
// absent.
//
// The person reading this screen is taking calls and watching couriers. A
// table's running tab is not something they can accept, assign or dispatch, and
// a busy room produces a few hundred of them a day. An earlier version offered
// it as a toggle, which was still offering it: a control that only ever adds
// noise is one somebody eventually leaves switched on.
//
// ⚠️ **`$exists: false`, not a comparison against the order type.** A guest
// scanning the table QR and ordering from their own phone is also `dinein` and
// *must* reach this board, because somebody has to accept it. Splitting on type
// would make QR orders vanish — the failure that surfaces as a guest complaint
// rather than a bug report.
func TestOrdersBoardExcludesTillChecks(t *testing.T) {
	src := readSource(t, "admin.go")
	fn := between(t, src, "func (h *Handler) AdminListOrders", "\n}\n")

	if !strings.Contains(fn, `filter["check"] = bson.M{"$exists": false}`) {
		t.Fatal("the orders board no longer excludes till checks")
	}
	// No way back in through a query parameter.
	if strings.Contains(fn, `Query().Get("till")`) {
		t.Fatal("a query parameter can put the dining room back on the board")
	}
	if strings.Contains(fn, `"type"`) {
		t.Fatal("the board filters on the order type — QR orders would vanish")
	}
}
