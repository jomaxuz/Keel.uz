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
}

func New(store *repository.Store, cfg *config.Config) *Handler {
	return &Handler{Store: store, Cfg: cfg}
}
