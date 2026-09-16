package repository

import "go.mongodb.org/mongo-driver/mongo"

// Store bundles access to all MongoDB collections used by the app.
type Store struct {
	DB              *mongo.Database
	Restaurant      *mongo.Collection
	Categories      *mongo.Collection
	Menu            *mongo.Collection
	Ingredients     *mongo.Collection
	Purchases       *mongo.Collection
	MarkedUnits     *mongo.Collection
	WriteOffs       *mongo.Collection
	CourierPayments *mongo.Collection
	Payouts         *mongo.Collection
	BankBalances    *mongo.Collection
	Collections     *mongo.Collection
	Stocktakes      *mongo.Collection
	// What a shortfall on a count turned out to be. ⚠️ Only the answer is
	// stored — the case list itself is derived from the counts on every read.
	// See models/shortagecase.go.
	ShortageCases *mongo.Collection
	Warehouses    *mongo.Collection
	Transfers     *mongo.Collection
	// One branch's store sending goods to another's — the central kitchen's van.
	// ⚠️ Not a transfer: that one moves between two shelves of the same branch.
	// See models/dispatch.go.
	Dispatches  *mongo.Collection
	Productions *mongo.Collection
	// What sales took off the shelf. ⚠️ The **source** for stock consumption,
	// not a journal beside it — see models/stockmovement.go.
	StockMoves *mongo.Collection
	// Cash handed to somebody to spend on the restaurant's behalf. ⚠️ Not an
	// expense — see models/advance.go.
	Advances *mongo.Collection
	// The shopping list somebody is sent to the market with. ⚠️ A request, not
	// a delivery — see models/shoppingorder.go.
	BuyOrders *mongo.Collection
	// Where the restaurant's cash physically is. ⚠️ A place, not a profit and
	// loss — see models/safe.go.
	SafeEntries *mongo.Collection
	// What the restaurant spends that nothing else records — rent, utilities,
	// tax, courier pay. ⚠️ Only what has no document of its own, or the
	// financial report counts it twice. See models/expense.go.
	Expenses *mongo.Collection
	// Which one-off migrations have already run. ⚠️ Needed because the stock
	// backfill is a pass over every order ever placed, and repeating it on
	// every boot would make a restart proportional to the restaurant's age.
	MigrationState *mongo.Collection
	StaffDevices   *mongo.Collection
	// A courier's phone. ⚠️ Its own collection rather than a role column on
	// staff_device: the ids come from different collections, and one field
	// holding two kinds of id is how a notification reaches the wrong person.
	CourierDevices *mongo.Collection
	// An owner's or manager's phone. Fourth device table, third id space —
	// see models/admindevice.go for why they are not one collection.
	AdminDevices *mongo.Collection
	// A **guest's** phone — the restaurant's own application. Fifth device
	// table, fourth id space, and the same reason as the other four: one field
	// holding two kinds of id is how a notification about somebody's dinner
	// reaches a cook. See models/userdevice.go.
	UserDevices *mongo.Collection
	// Which install an account is allowed to sign in from — see
	// models/logindevice.go. One collection for all three id spaces: the shape
	// is identical and the `kind` field keeps them apart.
	LoginDevices *mongo.Collection
	Suppliers    *mongo.Collection
	Placements   *mongo.Collection
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
	Staff      *mongo.Collection
	StaffRoles *mongo.Collection
	Receipts   *mongo.Collection
	PrintJobs  *mongo.Collection
	// One row per till screen bound to a branch. The plan is sold by register
	// count, and this is what makes that count a fact rather than a sentence.
	TillDevices *mongo.Collection
	// One row per television paired to a branch, and the short-lived codes they
	// show while waiting to be paired. Sold per screen, so the count has to be
	// a fact — the same argument as TillDevices next door.
	TVScreens  *mongo.Collection
	TVPairings *mongo.Collection
	// What those screens play: one row per picture or video, ordered, and
	// branch-wide rather than per screen — see models.TVSlide.
	TVSlides      *mongo.Collection
	Shifts        *mongo.Collection
	StaffPayments *mongo.Collection
	// Call centre: what was said on the phone and what came of it. Written by
	// hand by the operator — nothing here observes the line.
	Calls *mongo.Collection
	// Online payment: the provider credentials (their own collection, never
	// part of any public payload) and the transaction ledger.
	PaymentSettings *mongo.Collection
	Payments        *mongo.Collection
	// The credentials Uzum Tezkor calls us with. Its own collection for the
	// same reason as PaymentSettings — see models/uzumtezkor.go.
	UzumTezkorSettings *mongo.Collection
	// The till the restaurant already runs: one connection per branch, and the
	// map from our dishes to its products.
	POSSettings *mongo.Collection
	POSMappings *mongo.Collection
	// The virtual cash register that files this branch's sales with the tax
	// committee. Separate from POSSettings even though both are "the till":
	// a restaurant can run iiko and file through Multikassa, or run our own
	// till and file through the same, and folding them together would make one
	// choice depend on the other.
	FiscalSettings *mongo.Collection
	// ---- The accountant's two doors ----
	//
	// The EDI operator's credentials and our copy of what it holds, and the
	// login an office 1C knocks with. All three are their own collections and
	// none of them is on `restaurant`, for the reason written at the top of
	// every settings model here: that document is returned whole to every
	// visitor of the website. See models/accounting.go.
	EDISettings  *mongo.Collection
	EDIDocuments *mongo.Collection
	OneCSettings *mongo.Collection
	// The phone system: one account per company.
	PBXSettings *mongo.Collection
	// One row per visitor per day: how many people came, not just how many
	// ordered. See handlers/visits.go.
	Visits *mongo.Collection
	// The SMS gateway login codes go out through. Its own collection for the
	// same reason as PaymentSettings — the restaurant profile is public.
	SMSSettings *mongo.Collection
	// Browser push: the restaurant's own VAPID identity (one document, generated
	// once) and one document per subscription a guest granted.
	//
	// ⚠️ The identity is its own collection rather than a field on `restaurant`
	// for the same reason the payment keys are: that document is returned whole
	// to every visitor, and this one holds a private key. The *public* half is
	// handed to browsers deliberately — see models/push.go.
	PushSettings      *mongo.Collection
	PushSubscriptions *mongo.Collection
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
	// The strip the restaurant edits itself, and the jobs it is hiring for.
	Banners *mongo.Collection
	// What the owner was told about, and what this branch calls unusual.
	LossAlerts    *mongo.Collection
	AlertSettings *mongo.Collection
	// One morning's briefing per day, per lens, per language.
	Briefings *mongo.Collection
	// The advisor's material and its answers.
	//
	// ⚠️ **Two collections rather than one, because they expire differently.**
	// The snapshot is the day's picture of the business and is rebuilt tomorrow;
	// an answer is one question already paid for, keyed by that day's snapshot.
	// Folded together, clearing the cache would throw away the figures too — and
	// rebuilding those is a dozen aggregations.
	//
	// ⚠️ The snapshot holds the alias table (`c3` → a name), which is the half
	// that never leaves this server. See handlers/advisor.go.
	AdvisorSnapshots *mongo.Collection
	AdvisorAnswers   *mongo.Collection

	// One campaign plan per day per lens. ⚠️ Holds the figures it was written
	// from as well as the words: the panel draws its bars from those figures,
	// and a plan whose numbers are gone cannot be checked by the owner about to
	// spend money on it. See handlers/adsplan.go.
	AdsPlans *mongo.Collection

	Vacancies       *mongo.Collection
	ImportAssets    *mongo.Collection
	JobApplications *mongo.Collection
}

// New creates a Store from a mongo database handle.
func New(db *mongo.Database) *Store {
	return &Store{
		DB:              db,
		Restaurant:      db.Collection("restaurant"),
		Categories:      db.Collection("category"),
		Menu:            db.Collection("menu_item"),
		Ingredients:     db.Collection("ingredient"),
		Purchases:       db.Collection("purchase"),
		MarkedUnits:     db.Collection("marked_unit"),
		WriteOffs:       db.Collection("writeoff"),
		CourierPayments: db.Collection("courier_payment"),
		Payouts:         db.Collection("payout"),
		BankBalances:    db.Collection("bank_balance"),
		Collections:     db.Collection("collection"),
		Stocktakes:      db.Collection("stocktake"),
		ShortageCases:   db.Collection("shortage_case"),
		Warehouses:      db.Collection("warehouse"),
		Transfers:       db.Collection("stock_transfer"),
		Dispatches:      db.Collection("dispatch"),
		Productions:     db.Collection("production"),
		StockMoves:      db.Collection("stock_movement"),
		Advances:        db.Collection("staff_advance"),
		BuyOrders:       db.Collection("shopping_order"),
		SafeEntries:     db.Collection("safe_entry"),
		Expenses:        db.Collection("expense"),
		MigrationState:  db.Collection("migration_state"),
		StaffDevices:    db.Collection("staff_device"),
		UserDevices:     db.Collection("user_device"),
		CourierDevices:  db.Collection("courier_device"),
		AdminDevices:    db.Collection("admin_device"),
		LoginDevices:    db.Collection("login_device"),
		Suppliers:       db.Collection("supplier"),
		Placements:      db.Collection("ingredient_placement"),
		Orders:          db.Collection("order"),
		Admins:          db.Collection("admin_user"),
		Users:           db.Collection("user"),
		PhoneCodes:      db.Collection("phone_code"),
		Couriers:        db.Collection("courier"),
		Providers:       db.Collection("delivery_provider"),
		AdminLogs:       db.Collection("admin_log"),
		Reservations:    db.Collection("reservation"),
		Promotions:      db.Collection("promotion"),
		LoyaltyTxns:     db.Collection("loyalty_txn"),
		Feedback:        db.Collection("feedback"),
		Settlements:     db.Collection("courier_settlement"),
		CashShifts:      db.Collection("cash_shift"),
		CashEntries:     db.Collection("cash_entry"),
		Brands:          db.Collection("brand"),
		Branches:        db.Collection("branch"),

		Staff:         db.Collection("staff"),
		StaffRoles:    db.Collection("staff_role"),
		Receipts:      db.Collection("receipt_settings"),
		PrintJobs:     db.Collection("print_job"),
		TillDevices:   db.Collection("till_device"),
		TVScreens:     db.Collection("tv_screen"),
		TVPairings:    db.Collection("tv_pairing"),
		TVSlides:      db.Collection("tv_slide"),
		Shifts:        db.Collection("shift"),
		StaffPayments: db.Collection("staff_payment"),
		Calls:         db.Collection("call"),

		PaymentSettings:    db.Collection("payment_settings"),
		Payments:           db.Collection("payment"),
		UzumTezkorSettings: db.Collection("uzum_tezkor_settings"),

		POSSettings:    db.Collection("pos_settings"),
		POSMappings:    db.Collection("pos_mapping"),
		FiscalSettings: db.Collection("fiscal_settings"),
		EDISettings:    db.Collection("edi_settings"),
		EDIDocuments:   db.Collection("edi_document"),
		OneCSettings:   db.Collection("onec_settings"),
		PBXSettings:    db.Collection("pbx_settings"),
		Visits:         db.Collection("visit"),
		SMSSettings:    db.Collection("sms_settings"),

		PushSettings:      db.Collection("push_settings"),
		PushSubscriptions: db.Collection("push_subscription"),
		Designs:           db.Collection("page_design"),
		DesignPreviews:    db.Collection("design_preview"),
		TelegramChats:     db.Collection("telegram_chat"),
		Banners:           db.Collection("banner"),
		LossAlerts:        db.Collection("loss_alert"),
		AlertSettings:     db.Collection("alert_settings"),
		Briefings:         db.Collection("briefing"),
		AdvisorSnapshots:  db.Collection("advisor_snapshot"),
		AdvisorAnswers:    db.Collection("advisor_answer"),
		AdsPlans:          db.Collection("ads_plan"),
		Vacancies:         db.Collection("vacancy"),
		// Photographs the menu importer downloaded. ⚠️ Its own record because
		// the sweeper must never be able to consider a file the owner uploaded
		// — see handlers/importsweep.go.
		ImportAssets:     db.Collection("import_asset"),
		JobApplications:  db.Collection("job_application"),
		TelegramSettings: db.Collection("telegram_settings"),
	}
}
