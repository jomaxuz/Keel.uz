package router

import (
	"net/http"
	"time"

	"restaurant-backend/internal/config"
	"restaurant-backend/internal/handlers"
	appmw "restaurant-backend/internal/middleware"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// New builds the application HTTP router.
func New(h *handlers.Handler, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	// Per-IP gates on the routes that are cheap to call and costly to answer.
	// See internal/middleware/ratelimit.go for why these two exist and why
	// in-memory is the right scope here.
	//
	// ⚠️ Each endpoint that sends an SMS or verifies a bcrypt hash carries one
	// of these. `smsGate` is the tighter of the two because every call past it
	// spends money; `authGate` is looser because a person mistyping a password a
	// few times is normal, and only a machine reaches the wall.
	smsGate := appmw.NewRateLimit(5, time.Minute)
	authGate := appmw.NewRateLimit(10, time.Minute)

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Healthcheck.
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Uploaded photographs, at the size the page shows them (`?w=600`) and with
	// caching headers `http.FileServer` never set — see handlers/uploads.go. The
	// menu images were 78% of a restaurant home page's weight.
	r.Get("/uploads/*", h.ServeUploads)
	r.Head("/uploads/*", h.ServeUploads)

	r.Route("/api/v1", func(r chi.Router) {
		// ---- Public ----
		// How many people came to the site, as opposed to how many ordered.
		// Fired from the page itself, which is also what keeps crawlers out of
		// the count.
		r.Post("/visit", h.TrackVisit)
		r.Get("/restaurant", h.GetRestaurant)
		// The VAPID public key a browser needs before it can subscribe to
		// notifications. Public by definition — it is handed to every visitor
		// who is offered the permission, exactly like the map key.
		r.Get("/push/key", h.PushPublicKey)
		// Brands on offer and the branches that serve them.
		r.Get("/brands", h.GetBrands)
		// What the restaurant is hiring for, and one person answering.
		// ⚠️ Applying needs no account — see handlers/jobs.go for why, and for the
		// three rules that take the place of a login.
		r.Get("/vacancies", h.GetVacancies)
		r.Post("/vacancies/{id}/apply", h.ApplyForVacancy)
		r.Get("/categories", h.GetCategories)
		r.Get("/menu", h.GetMenu)
		r.Get("/menu/{id}", h.GetMenuItem)
		// "People usually order this with it" — for a dish page, or for a whole
		// basket at the cart and checkout. A POST because the basket is the
		// input, and a URL carrying eight dish ids gets truncated, logged and
		// cached by something along the way.
		r.Post("/recommendations", h.Recommendations)
		r.Post("/orders", h.CreateOrder)
		r.Get("/orders/{number}", h.TrackOrder)
		// Rating a delivered order, asked for on the tracking page the guest is
		// already looking at.
		r.Get("/orders/{number}/feedback", h.OrderFeedback)
		r.Post("/orders/{number}/feedback", h.SubmitFeedback)
		r.Post("/delivery/quote", h.DeliveryQuote)
		// The whole bill: lines, campaigns, a typed code, delivery. Runs the
		// same pipeline the order will, so the preview cannot disagree with the
		// receipt a second later.
		r.Post("/orders/quote", h.OrderQuote)
		// Which payment methods the checkout may offer: cash, plus every
		// provider that is switched on *and* fully credentialed.
		r.Get("/payment-methods", h.PublicPaymentMethods)
		// The bank link for an order. Public and keyed by the receipt number,
		// so a guest who closed the tab can still pay from another device.
		r.Get("/orders/{number}/pay", h.OrderPayLink)

		// The phone system announcing a call. Public because onlinePBX sends
		// no credentials — the token in the path is the authentication, which
		// is why it is generated rather than typed.
		r.Post("/pbx/onlinepbx/{token}", h.PBXWebhook)
		// The bot's incoming half: what a guest wrote to it. Public because
		// Telegram is the caller, and safe by the secret in the path — Telegram
		// sends no password of its own. Until this existed the bot could only
		// talk, and pressing Start got silence. See handlers/telegrambot.go.
		r.Post("/telegram/{token}", h.TelegramWebhook)

		// ---- The fiscal relay ----
		//
		// The program on the register's PC, asking for receipts to file and
		// reporting what the register said. Public in the same sense the
		// webhooks above are: it carries no user, and the token in the
		// header is the whole authentication — compared in constant time,
		// rotatable from the panel. See handlers/fiscalagent.go.
		//
		// ⚠️ Outside the 30-second router timeout would be wrong, but the
		// hold is deliberately kept under it: asking for work waits ~25s for
		// something to do rather than returning "nothing" ten times a
		// minute, forever, for every branch.
		r.Get("/fiscal/agent/job", h.FiscalAgentJob)
		r.Put("/fiscal/agent/job/{id}", h.FiscalAgentResult)
		// Ending the register's tax day. Its own endpoint because it writes
		// a different document than a filing does.
		r.Put("/fiscal/agent/close-day", h.FiscalAgentCloseDay)
		// The same agent prints: it is the only program that can reach a
		// printer on the restaurant's own network.
		r.Put("/fiscal/agent/print/{id}", h.FiscalAgentPrintResult)

		// ---- Provider callbacks ----
		//
		// Unauthenticated by our middleware on purpose: each provider
		// authenticates itself in its own way, and the adapters verify that
		// before touching anything. These are the URLs pasted into the
		// providers' cabinets, so they must not move.
		r.Post("/payments/payme", h.PaymeCallback)
		r.Post("/payments/click/prepare", h.ClickPrepare)
		r.Post("/payments/click/complete", h.ClickComplete)
		r.Post("/payments/uzum/check", h.UzumCheck)
		r.Post("/payments/uzum/create", h.UzumCreate)
		r.Post("/payments/uzum/confirm", h.UzumConfirm)
		r.Post("/payments/uzum/reverse", h.UzumReverse)
		r.Post("/payments/uzum/status", h.UzumStatus)
		// ATMOS asks permission before charging, so this one endpoint decides
		// whether a real guest's card is debited. See handlers/payatmos.go.
		r.Post("/payments/atmos", h.AtmosCallback)
		// What is on offer today, for the site to advertise. Codes are never
		// listed — a code nobody was given is a leak, not a promotion.
		r.Get("/promotions", h.GetPromotions)

		// ---- Table booking ----
		// The plan is public; making a booking is not. A table held for a
		// number nobody answers is worse than no booking, so the phone must be
		// one the guest proved they own — the same SMS login orders use.
		r.Get("/booking/plan", h.BookingPlan)
		r.Get("/reservations/{number}", h.TrackReservation)

		// ---- Customer auth: phone + one-time SMS code (see internal/sms) ----
		r.With(smsGate).Post("/auth/phone/request", h.PhoneRequestCode)
		r.Post("/auth/phone/verify", h.PhoneVerify)

		// ---- Courier auth (accounts are created in the admin panel) ----
		r.With(authGate).Post("/courier/login", h.CourierLogin)

		// ---- Staff auth (accounts are created in the admin panel) ----
		r.With(authGate).Post("/staff/login", h.StaffLogin)

		// ---- Branch kiosk screen (protected: kiosk JWT) ----
		// The screen at the branch that shows the rotating clock-in code. It
		// only ever reads a code; it can do nothing else.
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireRole(cfg.JWTSecret, "kiosk"))
			r.Get("/kiosk/code", h.KioskCode)
		})

		// ---- Courier (protected: courier JWT) ----
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireRole(cfg.JWTSecret, "courier"))
			r.Get("/courier/me", h.CourierMe)
			r.Put("/courier/status", h.CourierSetStatus)
			r.Post("/courier/location", h.CourierUpdateLocation)
			r.Get("/courier/orders", h.CourierOrders)
			r.Get("/courier/stats", h.CourierMyStats)
			r.Get("/courier/history", h.CourierMyHistory)
			r.Put("/courier/orders/{id}/status", h.CourierAdvanceOrder)
		})

		// ---- Staff (protected: staff JWT) ----
		// The employee's own app: clock in, clock out, and read back the
		// calendar and the wage those punches add up to. Nothing here can write
		// anything except a punch.
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireRole(cfg.JWTSecret, "staff"))
			r.Get("/staff/me", h.StaffMe)
			r.Post("/staff/clock", h.StaffClock)
			r.Get("/staff/report", h.StaffMyReport)
			// The kitchen screen. A staff token rather than an admin one
			// because the tablet by the pass is shared and never logs out —
			// see handlers/kitchen.go. The branch comes from the employee, so
			// no URL can ask for another kitchen's tickets.
			r.Get("/staff/kitchen", h.StaffKitchen)
			r.Put("/staff/kitchen/orders/{id}", h.StaffKitchenAction)

		})

		// ---- Naming yourself at a till (protected: device OR staff JWT) ----
		//
		// ⚠️ **A monoblock is bound to a branch, not signed into by a person.**
		// Nobody types a username and a password between two guests, and asking
		// them to is how a restaurant ends up with one login shared by
		// everybody. The machine holds a long-lived branch token; everything
		// after that is four digits. The staff token is still accepted for
		// tills installed before this shipped. See handlers/tillpin.go.
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireRole(cfg.JWTSecret, "staff", "tilldevice"))
			r.Get("/staff/till/session", h.StaffTillSession)
			// ⚠️ Rate-limited: four digits, and this endpoint is the only thing
			// standing in front of them. The per-branch counter inside is the
			// real guard; this one keeps a script off the server.
			r.With(authGate).Post("/staff/till/unlock", h.StaffTillUnlock)
		})

		// ---- The till and the floor (protected: staff OR till JWT) ----
		//
		// ⚠️ **Two roles, and the second one is deliberately weaker.** A "till"
		// token is bought with four digits tapped in front of the room (see
		// handlers/tillpin.go), so it reaches the floor and the cash drawer and
		// nothing else — not that person's payroll, not their time clock, not
		// the kitchen screen. Those stay in the group above, where the token
		// was bought with a username and a password.
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireRole(cfg.JWTSecret, "staff", "till"))

			// ---- The till: the floor screen and the cashier's screen ----
			//
			// One set of endpoints for both, because they share every rule and
			// differ only in what each person is allowed to do — which is asked
			// per handler (waiter or cashier), not per route group. Splitting
			// them into two prefixes would have meant two code paths pricing
			// the same table.
			//
			// The branch always comes from the employee, exactly as it does for
			// the kitchen screen above.
			r.Get("/staff/checks", h.StaffChecks)
			r.Post("/staff/checks", h.StaffOpenCheck)
			r.Get("/staff/checks/{id}", h.StaffCheck)
			r.Put("/staff/checks/{id}", h.StaffUpdateCheck)
			r.Post("/staff/checks/{id}/lines", h.StaffAddCheckLines)
			r.Delete("/staff/checks/{id}/lines/{lineId}", h.StaffVoidCheckLine)
			// "piyozsiz" against a dish. ⚠️ Refused once the line has been
			// fired: the paper at the pass cannot be edited, and a silent
			// change would leave the screen and the kitchen disagreeing.
			r.Put("/staff/checks/{id}/lines/{lineId}", h.StaffEditCheckLine)
			r.Post("/staff/checks/{id}/lines/move", h.StaffMoveCheckLines)
			// Two bills for one table. A waiter's action: it moves no money
			// and takes nothing off, and sending somebody to fetch the cashier
			// for the most ordinary request in a dining room is how the
			// cashier's PIN ends up known to everyone.
			r.Post("/staff/checks/{id}/split", h.StaffSplitCheck)
			// ...and the other half: two checks become one when a party joins.
			r.Post("/staff/checks/{id}/merge", h.StaffMergeChecks)
			r.Get("/staff/reservations", h.StaffReservations)
			r.Get("/staff/branch", h.StaffBranch)
			r.Post("/staff/checks/{id}/print", h.StaffPrintCheck)
			// Sales a till took while it had no network. ⚠️ Idempotent by the
			// id the till minted — see handlers/tillsync.go.
			r.Post("/staff/checks/sync", h.StaffSyncChecks)
			// Sending to the kitchen and taking payment are separate verbs on
			// purpose: typing a dish is not ordering it, and ordering it is not
			// paying for it. See handlers/tilllines.go.
			r.Post("/staff/checks/{id}/fire", h.StaffFireCheck)
			r.Post("/staff/checks/{id}/close", h.StaffCloseCheck)
			r.Post("/staff/checks/{id}/cancel", h.StaffCancelCheck)

			// ---- Fiscalisation, from the one machine that can reach it ----
			//
			// ⚠️ Two calls per filing, and the pair is the whole design: the
			// registered cash register is a program on a PC inside the
			// restaurant with no route from here, so the server builds the
			// document (POST) and the till carries it across the local network
			// and brings the answer back (PUT). See handlers/tillfiscal.go.
			//
			// The verbs are the honest ones: POST opens a filing attempt, PUT
			// records its outcome. Nothing about the receipt's contents is
			// decided on the far side.
			r.Get("/staff/fiscal", h.StaffFiscalStatus)
			// Sales that took money and have no receipt. The one screen that
			// makes a failed filing findable — see StaffUnfiledChecks.
			r.Get("/staff/fiscal/unfiled", h.StaffUnfiledChecks)
			// ⚠️ Ending the register's day — a tax document, refused while any
			// sale is still unfiled. See handlers/fiscalday.go for why this is
			// a request rather than something the panel can do itself.
			// ---- The cash drawer, from the till ----
			//
			// ⚠️ Where it belongs: the drawer is counted by the person standing
			// in front of it. The alternative was a panel login for every
			// cashier — the customer base, the payment keys and the reports —
			// or a manager counting a drawer somebody else emptied. Guarded by
			// the `shift` permission, which asks a manager rather than refusing.
			r.Get("/staff/cash-shift", h.StaffCashShift)
			r.Post("/staff/cash-shift/open", h.StaffOpenCashShift)
			r.Post("/staff/cash-shift/close", h.StaffCloseCashShift)
			// The X report: what this shift has sold and what should be in
			// the drawer, on paper, changing nothing. A GET because it can be
			// pressed at four in the afternoon by somebody with a suspicion,
			// as often as they like.
			r.Get("/staff/cash-shift/report", h.StaffShiftReport)

			r.Post("/staff/fiscal/close-day", h.StaffCloseFiscalDay)
			r.Put("/staff/fiscal/close-day", h.StaffCloseFiscalDayResult)
			r.Put("/staff/fiscal", h.StaffFiscalCheck)
			r.Post("/staff/checks/{id}/fiscal", h.StaffFileReceipt)
			r.Put("/staff/checks/{id}/fiscal", h.StaffFileReceiptResult)
		})

		// ---- User (protected: customer JWT) ----
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireRole(cfg.JWTSecret, "user"))
			r.Get("/users/me", h.UserMe)
			r.Put("/users/me", h.UpdateUserMe)
			r.Get("/users/me/orders", h.UserOrders)
			// Balance and the statement that explains it.
			r.Get("/users/me/loyalty", h.UserLoyalty)
			r.Get("/users/me/reservations", h.UserReservations)
			r.Post("/reservations", h.CreateReservation)
			// "Write to us" from the contact page: a rating and, if they have any,
			// words. Signed in only — see handlers/sitefeedback.go for why that is
			// about protecting the restaurant rather than the table.
			r.Post("/feedback", h.SubmitSiteFeedback)
			r.With(smsGate).Post("/users/me/phone/request", h.ChangePhoneRequest)
			r.Post("/users/me/phone/verify", h.ChangePhoneVerify)
			// A phone number Telegram vouched for — stronger evidence than an
			// SMS code, and one fewer paid message. Signed in only: the number
			// is written to the account already holding this session.
			r.Post("/users/me/telegram/phone", h.TelegramPhone)
			// The language the guest picked, kept on the account rather than in
			// a cookie — the bot messages them later, with no browser to read
			// one. See handlers/userlang.go.
			r.Put("/users/me/lang", h.UserSetLang)
			// Browser notifications. Behind the session because an audience in
			// this system is always built from order history — a subscription
			// with nobody behind it could be notified but never chosen, and the
			// guest could never revoke it.
			r.Post("/users/me/push", h.PushSubscribe)
			r.Delete("/users/me/push", h.PushUnsubscribe)
			// Dishes marked to come back to. On the account rather than in the
			// browser — see handlers/favorites.go.
			r.Get("/users/me/favorites", h.UserFavorites)
			r.Post("/users/me/favorites/{id}", h.ToggleFavorite)
		})

		// ---- Admin auth ----
		r.With(authGate).Post("/admin/login", h.Login)
		// Forgotten password: a one-time code to the number on the account.
		// Public by necessity — the whole point is that nobody can sign in.
		r.With(smsGate).Post("/admin/password/forgot", h.AdminForgotPassword)
		r.With(authGate).Post("/admin/password/reset", h.AdminResetPassword)

		// ---- Admin (protected). Role check matters: without it any valid
		// token — including a customer's — would be accepted here. ----
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireRole(cfg.JWTSecret, "owner", "manager"))

			r.Get("/admin/me", h.Me)
			r.Put("/admin/credentials", h.ChangeCredentials)
			// The recovery number the reset above texts. Verified by SMS, so a
			// mistyped digit is caught now rather than on the day it is needed.
			r.Post("/admin/me/phone/request", h.AdminPhoneRequest)
			r.Post("/admin/me/phone/verify", h.AdminPhoneVerify)
			// Which handset is this operator's, for click-to-call and for
			// "who answered".
			r.Put("/admin/me/extension", h.AdminSetMyExtension)
			// The front page, arranged per admin. An owner and a branch
			// manager open this screen for different figures, so the layout
			// belongs to the person rather than to the company.
			r.Get("/admin/me/dashboard", h.AdminDashboardTiles)
			r.Put("/admin/me/dashboard", h.AdminSetMyDashboard)
			r.Put("/admin/restaurant", h.UpdateRestaurant)

			r.Get("/admin/categories", h.AdminListCategories)

			// The strip the restaurant edits itself. Read by the site through
			// /restaurant, like the layout — one call per page, not two.
			r.Get("/admin/banners", h.AdminListBanners)
			r.Post("/admin/banners", h.AdminCreateBanner)
			r.Put("/admin/banners/{id}", h.AdminUpdateBanner)
			r.Delete("/admin/banners/{id}", h.AdminDeleteBanner)

			// Hiring: the vacancies, and the people who answered them.
			r.Get("/admin/vacancies", h.AdminListVacancies)
			r.Post("/admin/vacancies", h.AdminSaveVacancy)
			r.Put("/admin/vacancies/{id}", h.AdminSaveVacancy)
			r.Delete("/admin/vacancies/{id}", h.AdminDeleteVacancy)
			r.Get("/admin/job-applications", h.AdminListApplications)
			r.Put("/admin/job-applications/{id}", h.AdminUpdateApplication)
			r.Post("/admin/categories", h.CreateCategory)
			r.Put("/admin/categories/{id}", h.UpdateCategory)
			r.Delete("/admin/categories/{id}", h.DeleteCategory)

			r.Get("/admin/menu", h.AdminListMenu)
			r.Post("/admin/menu", h.CreateMenuItem)
			r.Put("/admin/menu/{id}", h.UpdateMenuItem)
			r.Delete("/admin/menu/{id}", h.DeleteMenuItem)

			r.Post("/admin/upload", h.Upload)

			r.Get("/admin/users", h.AdminListUsers)
			r.Get("/admin/users/{id}", h.AdminGetUser)
			// The restaurant's own notes about a customer: tags, a note, where
			// they came from, their birthday.
			r.Put("/admin/users/{id}", h.AdminUpdateUser)
			r.Get("/admin/tags", h.AdminTags)

			// Ratings and complaints. The unhandled list is the one that has
			// to reach zero.
			r.Get("/admin/feedback", h.AdminListFeedback)
			r.Put("/admin/feedback/{id}/handled", h.AdminHandleFeedback)
			r.Put("/admin/feedback/{id}/public", h.AdminPublishFeedback)

			r.Get("/admin/brands", h.AdminListBrands)
			r.Post("/admin/brands", h.AdminCreateBrand)
			r.Put("/admin/brands/{id}", h.AdminUpdateBrand)
			r.Delete("/admin/brands/{id}", h.AdminDeleteBrand)
			r.Get("/admin/branches", h.AdminListBranches)
			r.Post("/admin/branches", h.AdminCreateBranch)
			r.Put("/admin/branches/{id}", h.AdminUpdateBranch)
			r.Delete("/admin/branches/{id}", h.AdminDeleteBranch)
			// One tap at the counter: "we're out of samsa". Its own route so it
			// never has to load and re-save the whole branch.
			r.Put("/admin/branches/{id}/sold-out", h.AdminSetSoldOut)
			// The token for this branch's kiosk screen. `?rotate=1` replaces the
			// key, which revokes every screen token issued so far — the answer
			// to a tablet that left the building.
			r.Post("/admin/branches/{id}/kiosk", h.AdminKioskToken)

			// Online payment credentials. Owner-only inside the handler: a
			// branch manager does not hold the company's merchant keys.
			r.Get("/admin/payments", h.AdminGetPaymentSettings)
			r.Put("/admin/payments", h.AdminUpdatePaymentSettings)
			// Every attempt against one order, not just the successful one —
			// "the guest says they paid twice" is a ledger question.
			r.Get("/admin/orders/{id}/payments", h.AdminOrderPayments)

			// The SMS gateway login codes go out through. Owner-only, and per
			// restaurant: each one signs its own contract and pays its own bill.
			// The restaurant's own Telegram bot. Owner only, and the token is
			// never returned — see handlers/telegram.go.
			r.Get("/admin/telegram", h.AdminGetTelegram)
			r.Put("/admin/telegram", h.AdminUpdateTelegram)
			r.Post("/admin/telegram/ping", h.AdminPingTelegram)

			r.Get("/admin/sms", h.AdminGetSMS)
			r.Put("/admin/sms", h.AdminUpdateSMS)
			// Sends one real message. Credentials that look right still hide
			// two things — an unmoderated sender name and an empty balance —
			// and both surface at the first guest trying to log in.
			r.Post("/admin/sms/test", h.AdminTestSMS)

			// ---- The till the restaurant already runs ----
			// Per branch: a chain has one terminal group per kitchen, and an
			// order printed at the wrong one is worse than none printed.
			r.Get("/admin/pos", h.AdminGetPOS)
			r.Put("/admin/pos", h.AdminUpdatePOS)
			// Proves the credentials and says what it connected to.
			r.Post("/admin/pos/ping", h.AdminPingPOS)
			// What the till sells, so dishes are mapped by name rather than by
			// pasting 36-character ids.
			r.Get("/admin/pos/products", h.AdminPOSProducts)
			r.Get("/admin/pos/mapping", h.AdminPOSMapping)
			r.Put("/admin/pos/mapping", h.AdminSavePOSMapping)
			// A chain usually runs one iiko for every kitchen, so the second
			// branch's mapping is the first one's — typing it again is
			// transcription, and transcription is where the wrong id creeps in.
			r.Post("/admin/pos/mapping/copy", h.AdminCopyPOSMapping)
			// The retry button on a receipt.
			r.Post("/admin/orders/{id}/pos", h.AdminSendOrderToPOS)

			// What is off sale at this branch right now, and why — the counter's
			// own taps and the till's stop list in one screen.
			r.Get("/admin/stop-list", h.AdminStopList)
			// Reading the till's stop list is a background job; this is the
			// "now" button for the minute after the dish links were edited.
			r.Post("/admin/pos/stop-list/sync", h.AdminSyncPOSStopList)

			// ---- Fiscalisation (ККМ / ОФД) ----
			//
			// Beside the POS routes rather than inside them, because they are
			// two independent choices: a restaurant can run iiko and file
			// through Multikassa, or run our own till and file through the
			// same. Nesting these under /pos would have made one setting look
			// like a sub-option of the other.
			//
			// The provider list is readable by any admin — the panel draws the
			// section from it — while changing the connection and dialling it
			// are owner-only, checked in the handlers: this decides which
			// taxpayer the sales are filed under.
			r.Get("/admin/fiscal/providers", h.AdminFiscalProviders)
			r.Get("/admin/fiscal", h.AdminGetFiscal)
			r.Put("/admin/fiscal", h.AdminUpdateFiscal)
			r.Post("/admin/fiscal/ping", h.AdminFiscalPing)

			// ---- Receipt designs ----
			//
			// ⚠️ The preview is rendered by the printer's own code
			// (internal/receipt), so the paper cannot come out looking
			// different from the thing the owner designed.
			r.Get("/admin/receipts", h.AdminGetReceipts)
			r.Put("/admin/receipts", h.AdminUpdateReceipts)
			r.Post("/admin/receipts/preview", h.AdminPreviewReceipt)
			// ⚠️ The button that answers "is this printer actually reachable" —
			// the one thing an address in a form cannot tell anybody.
			r.Post("/admin/receipts/test-print", h.AdminTestPrint)
			// Mint the relay's credential. Shown once and never again, which is
			// what makes rotating it a revocation rather than a second key.
			r.Post("/admin/fiscal/agent-token", h.AdminFiscalAgentToken)
			// Bind a monoblock to this branch, or cut every one of them loose
			// (`?rotate=1`) when one walks out of the building.
			r.Get("/admin/branches/{id}/till-token", h.AdminTillToken)

			r.Get("/admin/stats", h.AdminStats)
			// Menu analysis: which dishes earn the money (ABC) and which of
			// them can be planned for (XYZ). `?format=xlsx` downloads the same
			// numbers as a spreadsheet rather than recomputing them.
			r.Get("/admin/reports/abc-xyz", h.AdminABCXYZ)
			// Money movement for a period. Not a P&L: there is no cost of
			// goods in this system, and the report says so on its own face.
			r.Get("/admin/reports/finance", h.AdminFinanceReport)
			// What came in against what the dishes sold should have used.
			// ⚠️ A flow, not a balance — see stockreport.go.
			r.Get("/admin/reports/stock", h.AdminStockReport)
			r.Get("/admin/reports/cash", h.AdminCashReport)
			// Sales over time, cut into days, weeks or months, and compared
			// with the period before it — a lone total cannot say whether a
			// month was good, only what it was.
			r.Get("/admin/reports/sales", h.AdminSalesReport)
			// Which door the orders came in through. Two cuts that overlap on
			// purpose (channel and fulfilment type) and are never summed.
			r.Get("/admin/reports/channels", h.AdminChannelReport)
			// Everyone on one page. The per-person screens answer "how is Aziz
			// doing"; these answer "how are they doing compared with each
			// other", which needs every row sorted and in one file.
			r.Get("/admin/reports/couriers", h.AdminCourierReport)
			r.Get("/admin/reports/staff", h.AdminStaffReport)

			// The till. Three actions, all of which move physical cash and all
			// of which are therefore in the audit log.
			r.Get("/admin/cash/shift", h.AdminCashShift)
			r.Post("/admin/cash/shift/open", h.AdminOpenCashShift)
			r.Post("/admin/cash/shift/close", h.AdminCloseCashShift)
			r.Post("/admin/cash/entries", h.AdminAddCashEntry)
			r.Get("/admin/alerts", h.AdminAlerts)
			// How busy each kitchen is, and moving one order between them.
			// Deliberately a person's decision — see handlers/branchload.go.
			r.Get("/admin/branches/load", h.AdminBranchLoad)
			r.Put("/admin/orders/{id}/branch", h.AdminMoveOrderBranch)
			r.Get("/admin/domain-check", h.AdminDomainCheck)
			// The last step of the guide, which used to be a phone call to
			// us. Relayed to the control plane, which verifies DNS itself
			// before it will serve — pointing a domain here is a claim only
			// the registrar account holder can make.
			r.Post("/admin/domain-connect", h.AdminDomainConnect)
			r.Get("/admin/reservations", h.AdminListReservations)
			r.Post("/admin/reservations", h.AdminCreateReservation)
			r.Put("/admin/reservations/{id}/status", h.AdminUpdateReservationStatus)
			r.Delete("/admin/reservations/{id}", h.AdminDeleteReservation)
			r.Get("/admin/orders", h.AdminListOrders)
			// Dining room and counter sales. A sibling of the orders board,
			// not a tab on it: till checks are left off that list on purpose,
			// and until this existed the money was in every report and the
			// sales themselves were on no screen an owner could open.
			// Ingredients and tech cards: what a dish costs, from what goes
			// into it. ⚠️ Costing, not stock — see models/ingredient.go.
			r.Get("/admin/ingredients", h.AdminListIngredients)
			r.Post("/admin/ingredients", h.AdminSaveIngredient)
			r.Put("/admin/ingredients/{id}", h.AdminSaveIngredient)
			r.Delete("/admin/ingredients/{id}", h.AdminDeleteIngredient)

			// Deliveries. ⚠️ Not stock — nothing subtracts what the kitchen
			// used; this is what came in and what it cost, which is what an
			// invoice is. The prices it carries update the ingredients.
			r.Get("/admin/purchases", h.AdminListPurchases)
			r.Post("/admin/purchases", h.AdminCreatePurchase)
			r.Delete("/admin/purchases/{id}", h.AdminDeletePurchase)

			// Food that left without being sold: spoiled, spilled, eaten by
			// the staff. ⚠️ A reason is required — see writeoffs.go.
			r.Get("/admin/writeoffs", h.AdminListWriteOffs)
			r.Post("/admin/writeoffs", h.AdminCreateWriteOff)
			r.Delete("/admin/writeoffs/{id}", h.AdminDeleteWriteOff)

			// Counting the store. ⚠️ The difference is the product — the
			// expected figure is the server's and is frozen when the count is
			// saved (see stocktake.go).
			r.Get("/admin/stocktake/sheet", h.AdminStocktakeSheet)
			r.Get("/admin/stocktake", h.AdminListStocktakes)
			r.Post("/admin/stocktake", h.AdminSaveStocktake)

			// What guests owe. ⚠️ A debt is the sale itself, closed and unpaid
			// — see debts.go.
			r.Get("/admin/debts", h.AdminDebts)
			r.Post("/admin/debts/{id}/pay", h.AdminPayDebt)

			r.Get("/admin/checks", h.AdminListChecks)
			r.Get("/admin/checks/{id}", h.AdminGetCheck)
			// A duplicate of the guest's receipt: the browser prints it (and
			// saves it as PDF), the branch's own printers only when asked.
			r.Post("/admin/checks/{id}/print", h.AdminPrintCheck)
			// Money handed back. ⚠️ The sale stays — the food was cooked and
			// eaten; what changed is the money.
			r.Post("/admin/checks/{id}/refund", h.AdminRefundCheck)

			// What the printers were asked to do, and what came back. ⚠️ A
			// failed print is the quietest failure in the system: the order is
			// on screen, the sale is in the reports, and the only symptom is a
			// plate nobody made.
			r.Get("/admin/print-jobs", h.AdminPrintJobs)
			r.Post("/admin/print-jobs/{id}/retry", h.AdminRetryPrintJob)
			// An order taken over the phone. Runs the same pricing pipeline as
			// the site — an operator takes the order, they do not negotiate it.
			r.Post("/admin/orders", h.AdminCreateOrder)
			// The same bill preview the checkout shows, for a named customer:
			// the operator has to be able to read the total back down the line.
			r.Post("/admin/orders/quote", h.AdminOrderQuote)
			r.Get("/admin/orders/{id}", h.AdminGetOrder)
			r.Put("/admin/orders/{id}/status", h.UpdateOrderStatus)
			r.Put("/admin/orders/{id}/courier", h.AdminAssignCourier)
			r.Put("/admin/orders/{id}/address", h.AdminUpdateOrderAddress)

			r.Put("/admin/orders/{id}/external-delivery", h.AdminCallProvider)
			r.Post("/admin/orders/{id}/external-delivery/call", h.AdminCallProviderAPI)
			r.Post("/admin/orders/{id}/external-delivery/sync", h.AdminSyncProviderAPI)
			r.Post("/admin/orders/{id}/external-delivery/cancel", h.AdminCancelProviderAPI)

			// ---- Call centre ----
			// One answer to "who is ringing?": the customer, what is in the
			// kitchen for them right now, what they usually order, and what
			// was said last time. Assembling that from three screens is how a
			// guest ends up being asked their own address.
			// Telephony. The settings are the owner's; the desk endpoints are
			// used by whoever is answering the phone.
			r.Get("/admin/pbx", h.AdminGetPBX)
			r.Put("/admin/pbx", h.AdminUpdatePBX)
			r.Post("/admin/pbx/ping", h.AdminPingPBX)
			// Polled by the call desk: is a call ringing for me right now?
			r.Get("/admin/calls/live", h.AdminLiveCall)
			// Ring the operator's handset, then the customer.
			r.Post("/admin/calls/dial", h.AdminDial)
			r.Get("/admin/calls/{id}/recording", h.AdminCallRecording)

			r.Get("/admin/lookup", h.AdminCallerLookup)
			r.Get("/admin/calls", h.AdminListCalls)
			r.Get("/admin/calls/stats", h.AdminCallStats)
			r.Post("/admin/calls", h.AdminCreateCall)
			// Editable, unlike the audit log: a call is written down while
			// somebody is still talking.
			r.Put("/admin/calls/{id}", h.AdminUpdateCall)

			r.Get("/admin/promotions", h.AdminListPromotions)
			r.Post("/admin/promotions", h.AdminCreatePromotion)
			r.Put("/admin/promotions/{id}", h.AdminUpdatePromotion)
			r.Delete("/admin/promotions/{id}", h.AdminDeletePromotion)
			// Who used a code and how many of them — counted from the orders,
			// not from the redemption guard.
			r.Get("/admin/promotions/{id}/usage", h.AdminPromotionUsage)

			r.Get("/admin/delivery-providers", h.AdminListProviders)
			r.Post("/admin/delivery-providers", h.AdminCreateProvider)
			r.Put("/admin/delivery-providers/{id}", h.AdminUpdateProvider)
			r.Delete("/admin/delivery-providers/{id}", h.AdminDeleteProvider)

			// Staff attendance. The list carries the numbers, so "who is in and
			// who is short" is answered without opening a card.
			// ---- Roles ----
			//
			// ⚠️ Reading is open to anybody who manages staff (assigning a role
			// needs the list); **writing is owner-only** — deciding who may
			// take money out of the restaurant is not a shift-level decision,
			// and a manager who could widen a role could widen their own.
			r.Get("/admin/roles", h.AdminListRoles)
			r.Post("/admin/roles", h.AdminCreateRole)
			r.Put("/admin/roles/{id}", h.AdminUpdateRole)
			r.Delete("/admin/roles/{id}", h.AdminDeleteRole)

			r.Get("/admin/staff", h.AdminListStaff)
			r.Get("/admin/staff/{id}", h.AdminGetStaff)
			r.Post("/admin/staff", h.AdminCreateStaff)
			r.Put("/admin/staff/{id}", h.AdminUpdateStaff)
			r.Delete("/admin/staff/{id}", h.AdminDeleteStaff)
			// The till code, on its own route. ⚠️ Not a field on the staff
			// form: a form that does not show it would send it empty on every
			// save and lock that person out of the till — the same trap as
			// branch.soldOut. See handlers/tillpin.go.
			r.Put("/admin/staff/{id}/pin", h.AdminSetStaffPin)
			// Correcting attendance by hand: a dead phone, a forgotten
			// clock-out. Always signed with the admin's name.
			r.Post("/admin/staff/{id}/shifts", h.AdminCreateShift)
			r.Put("/admin/staff/{id}/shifts/{shiftId}", h.AdminUpdateShift)
			r.Delete("/admin/staff/{id}/shifts/{shiftId}", h.AdminDeleteShift)
			// Payroll: what is owed, and the ledger of what was handed over.
			r.Get("/admin/payroll", h.AdminPayroll)
			r.Post("/admin/staff/{id}/payments", h.AdminPayStaff)
			r.Delete("/admin/staff/{id}/payments/{paymentId}", h.AdminDeleteStaffPayment)

			r.Get("/admin/couriers", h.AdminListCouriers)
			r.Get("/admin/couriers/{id}", h.AdminGetCourier)
			r.Post("/admin/couriers", h.AdminCreateCourier)
			r.Put("/admin/couriers/{id}", h.AdminUpdateCourier)
			r.Delete("/admin/couriers/{id}", h.AdminDeleteCourier)
			// Cash handed back. A ledger entry, not a counter reset: "how much did
			// Aziz hand in last Tuesday?" has to stay answerable.
			r.Post("/admin/couriers/{id}/settle", h.AdminSettleCourierCash)
		})

		// ---- Admin (owner only) ----
		// Handing out panel accounts and reading the activity log is the
		// owner's business: a manager must not be able to grant themselves
		// more rights or erase the trail behind them.
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireRole(cfg.JWTSecret, "owner"))

			r.Get("/admin/accounts", h.AdminListAccounts)
			r.Post("/admin/accounts", h.AdminCreateAccount)
			r.Put("/admin/accounts/{id}", h.AdminUpdateAccount)
			r.Delete("/admin/accounts/{id}", h.AdminDeleteAccount)

			r.Get("/admin/logs", h.AdminListLogs)

			// Taking the whole business away as one archive. Owner only, and
			// only while the platform has opened a dated grant — the handler
			// checks both again, because a route in the right group is not a
			// permission (see handlers/export.go).
			// One message to one segment. Owner only: the customer base belongs
			// to the company, a manager runs one kitchen, and this is the single
			// button that can annoy every guest at once — and spend real money
			// doing it. See handlers/campaigns.go.
			r.Get("/admin/segments", h.AdminSegments)
			// The customer base ranked against itself on three axes. Beside
			// the rule segments, never instead of them — see handlers/rfm.go.
			r.Get("/admin/rfm", h.AdminRFM)
			r.Get("/admin/campaigns", h.AdminListCampaigns)
			r.Post("/admin/campaigns/preview", h.AdminCampaignPreview)
			r.Post("/admin/campaigns", h.AdminSendCampaign)

			r.Get("/admin/export", h.AdminExportStatus)
			r.Get("/admin/export/archive", h.AdminExportArchive)
		})
	})

	return r
}
