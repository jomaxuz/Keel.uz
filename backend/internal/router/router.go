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

	// Static uploads.
	fs := http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir)))
	r.Handle("/uploads/*", fs)

	r.Route("/api/v1", func(r chi.Router) {
		// ---- Public ----
		r.Get("/restaurant", h.GetRestaurant)
		// Brands on offer and the branches that serve them.
		r.Get("/brands", h.GetBrands)
		r.Get("/categories", h.GetCategories)
		r.Get("/menu", h.GetMenu)
		r.Get("/menu/{id}", h.GetMenuItem)
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
		r.Post("/auth/phone/request", h.PhoneRequestCode)
		r.Post("/auth/phone/verify", h.PhoneVerify)

		// ---- Courier auth (accounts are created in the admin panel) ----
		r.Post("/courier/login", h.CourierLogin)

		// ---- Staff auth (accounts are created in the admin panel) ----
		r.Post("/staff/login", h.StaffLogin)

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
			r.Post("/users/me/phone/request", h.ChangePhoneRequest)
			r.Post("/users/me/phone/verify", h.ChangePhoneVerify)
		})

		// ---- Admin auth ----
		r.Post("/admin/login", h.Login)
		// Forgotten password: a one-time code to the number on the account.
		// Public by necessity — the whole point is that nobody can sign in.
		r.Post("/admin/password/forgot", h.AdminForgotPassword)
		r.Post("/admin/password/reset", h.AdminResetPassword)

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
			r.Put("/admin/restaurant", h.UpdateRestaurant)

			r.Get("/admin/categories", h.AdminListCategories)
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
			// The retry button on a receipt.
			r.Post("/admin/orders/{id}/pos", h.AdminSendOrderToPOS)

			r.Get("/admin/stats", h.AdminStats)
			r.Get("/admin/alerts", h.AdminAlerts)
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
			r.Get("/admin/staff", h.AdminListStaff)
			r.Get("/admin/staff/{id}", h.AdminGetStaff)
			r.Post("/admin/staff", h.AdminCreateStaff)
			r.Put("/admin/staff/{id}", h.AdminUpdateStaff)
			r.Delete("/admin/staff/{id}", h.AdminDeleteStaff)
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
		})
	})

	return r
}
