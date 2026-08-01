package handlers

import (
	"restaurant-backend/internal/config"
	"restaurant-backend/internal/repository"
	"restaurant-backend/internal/sms"
)

// Handler carries dependencies shared by all HTTP handlers.
type Handler struct {
	Store *repository.Store
	Cfg   *config.Config
	SMS   sms.Sender
}

func New(store *repository.Store, cfg *config.Config) *Handler {
	sender := sms.New(sms.Config{
		Provider:           cfg.SMSProvider,
		From:               cfg.SMSFrom,
		EskizEmail:         cfg.EskizEmail,
		EskizPassword:      cfg.EskizPassword,
		EskizBaseURL:       cfg.EskizBaseURL,
		PlayMobileURL:      cfg.PlayMobileURL,
		PlayMobileLogin:    cfg.PlayMobileLogin,
		PlayMobilePassword: cfg.PlayMobilePassword,
	})
	return &Handler{Store: store, Cfg: cfg, SMS: sender}
}
