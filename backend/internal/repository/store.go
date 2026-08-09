package repository

import "go.mongodb.org/mongo-driver/mongo"

// Store bundles access to all MongoDB collections used by the app.
type Store struct {
	DB           *mongo.Database
	Restaurant   *mongo.Collection
	Categories   *mongo.Collection
	Menu         *mongo.Collection
	Orders       *mongo.Collection
	Admins       *mongo.Collection
	Users        *mongo.Collection
	PhoneCodes   *mongo.Collection
	Couriers     *mongo.Collection
	Providers    *mongo.Collection
	AdminLogs    *mongo.Collection
	Reservations *mongo.Collection
	Promotions   *mongo.Collection
	LoyaltyTxns  *mongo.Collection
	Feedback     *mongo.Collection
	Settlements  *mongo.Collection
	CashShifts   *mongo.Collection
	CashEntries  *mongo.Collection
	Brands       *mongo.Collection
	Branches     *mongo.Collection
	// Staff attendance: the accounts, their clock-in/out records and the
	// salary actually handed over.
	Staff         *mongo.Collection
	Shifts        *mongo.Collection
	StaffPayments *mongo.Collection
	// Call centre: what was said on the phone and what came of it. Written by
	// hand by the operator — nothing here observes the line.
	Calls *mongo.Collection
	// Online payment: the provider credentials (their own collection, never
	// part of any public payload) and the transaction ledger.
	PaymentSettings *mongo.Collection
	Payments        *mongo.Collection
	// The till the restaurant already runs: one connection per branch, and the
	// map from our dishes to its products.
	POSSettings *mongo.Collection
	POSMappings *mongo.Collection
	// The phone system: one account per company.
	PBXSettings *mongo.Collection
	// One row per visitor per day: how many people came, not just how many
	// ordered. See handlers/visits.go.
	Visits *mongo.Collection
	// The SMS gateway login codes go out through. Its own collection for the
	// same reason as PaymentSettings — the restaurant profile is public.
	SMSSettings *mongo.Collection
	// The restaurant's own Telegram bot. Its own collection for the same reason
	// as PaymentSettings and SMSSettings — see models/telegram.go.
	TelegramSettings *mongo.Collection
	// The page layout, drawn in the Keel console and written straight into this
	// database — one document per brand. See models/design.go.
	Designs *mongo.Collection
	// Short-lived tokens that let the console preview an unpublished design on
	// the live site. Written by the control plane, read here.
	DesignPreviews *mongo.Collection
	// ⚠️ What the bot knows about a chat **before** it knows who the person is.
	//
	// A guest can tap a language, and be asked for a number, minutes before any
	// account exists — Telegram gives an id and a name, never a number. Without
	// somewhere to keep that choice it is lost, and the bot answers the next message
	// in whatever language the phone happens to be set to. That is exactly what
	// shipped: a guest tapped "O'zbekcha" and was thanked in Russian.
	TelegramChats *mongo.Collection
}

// New creates a Store from a mongo database handle.
func New(db *mongo.Database) *Store {
	return &Store{
		DB:           db,
		Restaurant:   db.Collection("restaurant"),
		Categories:   db.Collection("category"),
		Menu:         db.Collection("menu_item"),
		Orders:       db.Collection("order"),
		Admins:       db.Collection("admin_user"),
		Users:        db.Collection("user"),
		PhoneCodes:   db.Collection("phone_code"),
		Couriers:     db.Collection("courier"),
		Providers:    db.Collection("delivery_provider"),
		AdminLogs:    db.Collection("admin_log"),
		Reservations: db.Collection("reservation"),
		Promotions:   db.Collection("promotion"),
		LoyaltyTxns:  db.Collection("loyalty_txn"),
		Feedback:     db.Collection("feedback"),
		Settlements:  db.Collection("courier_settlement"),
		CashShifts:   db.Collection("cash_shift"),
		CashEntries:  db.Collection("cash_entry"),
		Brands:       db.Collection("brand"),
		Branches:     db.Collection("branch"),

		Staff:         db.Collection("staff"),
		Shifts:        db.Collection("shift"),
		StaffPayments: db.Collection("staff_payment"),
		Calls:         db.Collection("call"),

		PaymentSettings: db.Collection("payment_settings"),
		Payments:        db.Collection("payment"),

		POSSettings:      db.Collection("pos_settings"),
		POSMappings:      db.Collection("pos_mapping"),
		PBXSettings:      db.Collection("pbx_settings"),
		Visits:           db.Collection("visit"),
		SMSSettings:      db.Collection("sms_settings"),
		Designs:          db.Collection("page_design"),
		DesignPreviews:   db.Collection("design_preview"),
		TelegramChats:    db.Collection("telegram_chat"),
		TelegramSettings: db.Collection("telegram_settings"),
	}
}
