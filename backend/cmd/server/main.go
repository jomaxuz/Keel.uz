package main

import (
	// Embeds the IANA timezone database in the binary.
	//
	// Without it, an alpine image with no tzdata cannot resolve
	// TZ=Asia/Tashkent and time.Local silently falls back to UTC — which
	// files every staff shift under the wrong calendar day, five hours early,
	// with nothing in the logs to say so. This exact thing happened on the
	// first production deploy.
	_ "time/tzdata"

	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"restaurant-backend/internal/config"
	"restaurant-backend/internal/db"
	"restaurant-backend/internal/handlers"
	"restaurant-backend/internal/images"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/push"
	"restaurant-backend/internal/repository"
	"restaurant-backend/internal/router"
	"restaurant-backend/internal/seed"
)

func main() {
	cfg := config.Load()
	// Refuse to run with a forgeable signing key. Checked before anything else
	// touches the network — a server that comes up on a default secret is worse
	// than one that does not come up at all.
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	database, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	log.Printf("connected to MongoDB %q", cfg.MongoDB)

	// Printed on purpose: a wrong timezone corrupts staff attendance quietly,
	// so the very first log line has to say which one is in effect.
	log.Printf("timezone: %s (now %s)", time.Local, time.Now().Format(time.RFC3339))

	// ⚠️ **Said out loud, beside the timezone, and for the same reason.** Every
	// uploaded photograph is converted to WebP — but the encoder is reached
	// through a system library unless the binary was built with the `nodynamic`
	// tag (see the Dockerfile). A build that missed it keeps working and
	// quietly stores JPEGs for months, which is precisely the class of failure
	// this log line exists to end.
	if err := images.Available(); err != nil {
		log.Printf("⚠ webp encoder unavailable (%v) — rasmlar JPEG/PNG bo'lib saqlanadi", err)
	} else {
		log.Printf("images: webp encoder ready")
	}

	// ⚠️ **Said out loud for the third time, and for the same reason as the
	// other two.** Notifications to the native applications go through Firebase;
	// with no service account they simply do not go, and nothing anywhere raises
	// a hand — a phone registers, the settings screen says "on", and the kitchen
	// presses Ready into silence. This line is the difference between finding
	// that out at boot and finding it out from a restaurant.
	//
	// ⚠️ **Not fatal.** Every install whose staff carry the Expo builds is
	// correct with this unset, and those keep going through the relay.
	if err := push.Configure(cfg.FCMCredentials); err != nil {
		log.Printf("⚠ fcm sozlanmadi (%v) — native ilovalarga bildirishnoma bormaydi", err)
	} else if push.FCMReady() {
		log.Printf("push: fcm ready")
	} else {
		log.Printf("push: expo only (FCM_CREDENTIALS bo'sh)")
	}

	store := repository.New(database)
	seed.Bootstrap(ctx, store, cfg)
	// Every install has at least one brand and one branch; an install that
	// predates them is migrated here (see repository/migrate.go).
	// ⚠️ **Read once, here, and only used when a brand is created.** The control
	// plane sends this on a tenant's first boot the way it sends the admin
	// credentials; on every boot afterwards a brand already exists and this
	// value is ignored entirely.
	if err := repository.EnsureBrandAndBranch(ctx, store, models.BusinessType(cfg.BusinessType)); err != nil {
		log.Fatalf("brand/branch migration: %v", err)
	}
	// Branches written before staff attendance existed get a real geofence
	// rather than the zero value, which reads as "no check at all".
	if err := repository.EnsureStaffDefaults(ctx, store); err != nil {
		log.Printf("staff defaults: %v", err)
	}
	// Branches written before anything had ever run out hold `null` where the
	// stop lists belong, and $addToSet refuses a non-array field — which is the
	// first tap at the counter, not the hundredth.
	// Grandfathers staff who could already open the kitchen screen, so adding
	// the permission does not blank a live pass mid-service.
	// ⚠️ Before the stock screens can read a placement, the ingredients that
	// carried their store on themselves have to be moved onto one — otherwise
	// every shelf reads zero on the deploy that ships this.
	if err := repository.EnsureIngredientPlacements(ctx, store); err != nil {
		log.Printf("ingredient placements: %v", err)
	}
	// ⚠️ Before the supplier page can show a debt, the deliveries that predate
	// the idea have to be settled — otherwise absent reads as owing and the
	// first thing an owner sees is a two-year invented debt.
	if err := repository.EnsureDeliveriesSettled(ctx, store); err != nil {
		log.Printf("settle old deliveries: %v", err)
	}
	if err := repository.EnsureKitchenAccess(ctx, store); err != nil {
		log.Printf("kitchen access migration: %v", err)
	}
	// Seeds the role list and moves existing staff onto it, keeping every
	// ability they already had — see EnsureStaffRoles for why a migrated
	// cashier becomes a floor administrator rather than a "Kassir".
	if err := repository.EnsureStaffRoles(ctx, store); err != nil {
		log.Printf("staff roles migration: %v", err)
	}
	// The buying role, for restaurants that were already running when a market
	// run could not be recorded from a phone. Runs once and leaves a deleted
	// role deleted — see EnsureBuyerRole.
	if err := repository.EnsureBuyerRole(ctx, store); err != nil {
		log.Printf("buyer role migration: %v", err)
	}
	// The storekeeper, and the permission to write a shopping list for the
	// roles that were already standing where it gets written.
	if err := repository.EnsureStorekeeperRole(ctx, store); err != nil {
		log.Printf("storekeeper role migration: %v", err)
	}
	// And the permission to hand out what a request asks for — the storekeeper's
	// own half of the same morning. See EnsureStockIssue.
	if err := repository.EnsureStockIssue(ctx, store); err != nil {
		log.Printf("stock issue migration: %v", err)
	}
	// The job title is the role's name — see EnsurePositionFromRole.
	if err := repository.EnsurePositionFromRole(ctx, store); err != nil {
		log.Printf("position from role: %v", err)
	}
	if err := repository.EnsureSoldOutArrays(ctx, store); err != nil {
		log.Printf("sold-out arrays: %v", err)
	}
	// Orders that predate online payment were all settled at the door, so the
	// kitchen could start on them the moment they were placed.
	if err := repository.EnsureQueuedAt(ctx, store); err != nil {
		log.Printf("queuedAt migration: %v", err)
	}
	// Designs drawn in the console were seeded from a list with no reviews band,
	// and the console could not add one — so every hand-drawn page silently
	// lacked it, however the restaurant had set its reviews.
	if err := repository.EnsureReviewsBand(ctx, store); err != nil {
		log.Printf("reviews band migration: %v", err)
	}
	if err := repository.EnsureIndexes(ctx, store); err != nil {
		log.Printf("index setup: %v", err)
	}

	h := handlers.New(store, cfg)

	// ⚠️ **Before the first request, and fatal if it fails.** Stock consumption
	// is read from written movement rows and from nothing else; an install that
	// began serving with that collection empty and a year of orders behind it
	// would report every shelf full, release every stop list, and turn the next
	// stocktake into a surplus the size of a year's cooking. None of that looks
	// like an error from the outside, which is exactly why the server must not
	// come up without it — the same rule the duplicate POS-settings index
	// follows. Runs once: the marker is written only after the pass completes.
	if err := h.BackfillStockMovements(ctx); err != nil {
		log.Fatalf("stock movement backfill: %v", err)
	}

	r := router.New(h, cfg)

	// Mirrors each connected till's stop list onto the site, so a dish the
	// kitchen stopped at the counter stops being orderable here too. Does
	// nothing at all on an install with no POS connected.
	syncCtx, stopSync := context.WithCancel(ctx)
	defer stopSync()
	h.StartPOSStopSync(syncCtx)
	// Asks each till what became of the orders we handed it. Poster and iiko
	// both file an order before anybody at the counter has accepted it, and
	// without this the panel would never learn the difference.
	h.StartPOSOrderSync(syncCtx)
	// Stops dishes whose store is empty. Does nothing at all on a branch that
	// has not switched it on, which is every branch by default — see
	// handlers/stockstop.go for why refusing a sale is opt-in.
	h.StartStockStopSync(syncCtx)
	// Catches any order whose stock rows were not written by the handler that
	// changed it — a failed write, a path with no hook, a status changed by a
	// payment callback. See handlers/stocksale.go.
	h.StartStockMoveSync(syncCtx)
	// Tells the owner when a cash shift has been open long enough that the
	// drawer has stopped meaning anything. ⚠️ It never closes one: a close
	// writes a counted figure, and a count nobody made is the one thing that
	// would make these numbers worse. See handlers/shiftwatch.go.
	h.StartShiftWatch(syncCtx)
	// Tells the owner about work standing still — an order nobody accepted, a
	// pre-order falling due, a booking nobody answered. ⚠️ The panel's own bell
	// only rings for somebody who already has it open; this is for the hours
	// when nobody does. See handlers/queuewatch.go.
	h.StartQueueWatch(syncCtx)
	// Retries alerts whose Telegram send failed and sends the ones the daily
	// ceiling held back as one digest. ⚠️ Without it an alert had one attempt:
	// a 429, a network blip or a deploy mid-send lost it for good. See
	// handlers/alerts.go.
	h.StartAlertRetry(syncCtx)
	// ---- Advertising ----
	//
	// Pulls what each campaign cost and brought back into a daily snapshot,
	// reports finished orders to Meta so the campaign can be measured in
	// dinners rather than clicks, and applies the ceilings the owner set.
	//
	// ⚠️ **All three do nothing at all on an install with no Meta account
	// connected**, which is almost every one: they read the connection first
	// and return. See handlers/adsinsights.go, adscapi.go and adsrules.go.
	h.StartAdsSync(syncCtx)
	h.StartAdsEvents(syncCtx)
	h.StartAdsRules(syncCtx)

	// Outgoing webhooks: the queue orderEvent fills. ⚠️ Idle on every install
	// with no endpoint registered — the queue is empty, and one indexed read
	// every fifteen seconds is all it costs. See handlers/webhooks.go.
	h.StartWebhookSender(syncCtx)
	// money.day_changed — idle unless an endpoint subscribes to it. See
	// handlers/openfinance.go.
	h.StartMoneyWatch(syncCtx)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Print("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
