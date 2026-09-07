package handlers

import (
	"sync"

	"restaurant-backend/internal/config"
	"restaurant-backend/internal/repository"
	"restaurant-backend/internal/sms"
)

// Handler carries dependencies shared by all HTTP handlers.
type Handler struct {
	Store *repository.Store
	Cfg   *config.Config

	// The SMS gateway is a setting the restaurant edits, so the sender is
	// built from the database per request rather than once at boot — see
	// h.sender(). It is cached here because the Eskiz sender holds a bearer
	// token worth re-using; the key is the settings document's updatedAt, so
	// saving the page swaps the gateway without a restart.
	smsMu     sync.Mutex
	smsCached sms.Sender
	smsKey    string

	// The exchange file an office 1C is halfway through uploading.
	//
	// ⚠️ **In memory, and only between two requests seconds apart.** The
	// published protocol uploads a file and then asks for it to be imported;
	// keeping it on disk would mean a temporary file to clean up, a container
	// permission to get right, and a way for one exchange to fill a volume the
	// whole box shares. Losing it to a restart costs 1C one retry, which is
	// what 1C does anyway.
	oneCMu    sync.Mutex
	oneCFiles map[string][]byte
}

func New(store *repository.Store, cfg *config.Config) *Handler {
	return &Handler{Store: store, Cfg: cfg}
}
