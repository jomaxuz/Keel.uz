package main

import (
	// The control plane files every tenant's day by local calendar date, so it
	// carries the timezone database for the same reason the tenant server does:
	// an alpine image without tzdata silently runs in UTC, and every invoice
	// boundary lands five hours early with nothing in the logs to say so.
	_ "time/tzdata"

	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"keel-control/internal/aggregate"
	"keel-control/internal/config"
	"keel-control/internal/db"
	"keel-control/internal/handlers"
	"keel-control/internal/models"
	"keel-control/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	client, err := db.Connect(ctx, cfg.MongoURI)
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	tenantClient := client
	if cfg.TenantMongoURI != cfg.MongoURI {
		if tenantClient, err = db.Connect(ctx, cfg.TenantMongoURI); err != nil {
			log.Fatalf("tenant mongo connect: %v", err)
		}
	}
	store := repository.New(client.Database(cfg.MongoDB), tenantClient)

	log.Printf("control plane database %q", cfg.MongoDB)
	log.Printf("timezone: %s (now %s)", time.Local, time.Now().Format(time.RFC3339))

	if err := store.EnsureIndexes(ctx); err != nil {
		log.Printf("index setup: %v", err)
	}
	seedUser(ctx, store, cfg)

	h := handlers.New(store, cfg)

	// Nightly is the intent; hourly is the schedule, so a restart at 23:50
	// does not lose the day and a correction is never more than an hour away.
	go maintain(ctx, store, h, time.Hour, 35)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handlers.Router(h, cfg),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	go func() {
		log.Printf("keel control listening on :%s", cfg.Port)
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

// maintain is the one recurring job on this server: collect yesterday's
// numbers, then switch off the demos that have run out.
//
// One ticker rather than two schedulers, and no cron: everything the control
// plane does on a clock is here, in the order it has to happen. Collecting
// first means a demo about to be switched off still has its last day counted.
//
// It runs once at boot as well, so a server that was down over the weekend
// catches up instead of waiting for the top of the hour — and so a deployment
// is never blank until midnight.
//
// ⚠️ Everything below is measured on **this server's clock**. `TZ` must be
// Asia/Tashkent or a demo ends five hours before the customer expects it; the
// timezone is logged above for exactly this reason.
func maintain(ctx context.Context, store *repository.Store, h *handlers.Handler,
	interval time.Duration, days int) {
	run := func() {
		if err := aggregate.Run(ctx, store, days); err != nil {
			log.Printf("aggregate: %v", err)
		}
		h.SweepTrials(ctx)
	}
	run()

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}

// seedUser creates the first dashboard account, and only the first: an upsert
// here would reset the password to the env value on every deploy, which is how
// a production account quietly goes back to admin123.
func seedUser(ctx context.Context, store *repository.Store, cfg *config.Config) {
	name := strings.ToLower(strings.TrimSpace(cfg.AdminUsername))
	if name == "" {
		return
	}
	n, err := store.Users.CountDocuments(ctx, bson.M{})
	if err != nil || n > 0 {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("seed user: %v", err)
		return
	}
	_, err = store.Users.UpdateOne(ctx, bson.M{"username": name}, bson.M{"$setOnInsert": models.User{
		Username:     name,
		PasswordHash: string(hash),
		Name:         "Keel",
		CreatedAt:    time.Now(),
	}}, options.Update().SetUpsert(true))
	if err != nil {
		log.Printf("seed user: %v", err)
		return
	}
	log.Printf("dashboard account %q created", name)
}
