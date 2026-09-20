package delivery

import "context"

// Service is what a carrier has to be able to do for an order to be handed to
// it: file the request, confirm it, read its state back, and drop it.
//
// ⚠️ **An interface because there are now two of them, and the second one
// arrived with no published contract.** Yandex Delivery's wire format is
// documented; BTS Express's is not (see docs/vendor/bts-express.md), so the
// part of it that will need correcting once a real contract is in hand has to
// be confined to one file. The handlers talk to this and never to a concrete
// client, which is what makes that correction a single file rather than a
// search through the order screens.
//
// ⚠️ **Accept is separate from Create on purpose**, even though a carrier that
// dispatches immediately has nothing to do in it. Yandex's create only drafts
// a claim — until Accept nobody is sent — and folding the two together would
// mean either a Yandex order nobody collects or a BTS order filed twice. A
// carrier without the distinction returns the claim it already has.
type Service interface {
	// Create files the request. requestID is stable per order so a retry after
	// a network timeout re-uses the request instead of ordering a second
	// courier — the one failure mode that costs the restaurant money.
	Create(ctx context.Context, o Order, requestID string) (Claim, error)
	Accept(ctx context.Context, claimID string, version int) (Claim, error)
	Info(ctx context.Context, claimID string) (Claim, error)
	// Cancel drops it. `paid` says whether this is a late cancellation the
	// carrier may charge for — their rule, so the caller states which case it
	// is rather than the client guessing.
	Cancel(ctx context.Context, claimID string, version int, paid bool) error
}

// The two carriers this build knows. Stored on the provider record, so never
// renamed.
const (
	ProviderYandex = "yandex"
	ProviderBTS    = "bts"
)

var (
	_ Service = (*Client)(nil)
	_ Service = (*BTS)(nil)
)
