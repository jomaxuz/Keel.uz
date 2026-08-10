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
	"restaurant-backend/internal/repository"
	"restaurant-backend/internal/router"
	"restaurant-backend/internal/seed"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	database, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	log.Printf("connected to MongoDB %q", cfg.MongoDB)

	// Printed on purpose: a wrong timezone corrupts staff attendance quietly,
	// so the very first log line has to say which one is in effect.
	log.Printf("timezone: %s (now %s)", time.Local, time.Now().Format(time.RFC3339))

	store := repository.New(database)
	seed.Bootstrap(ctx, store, cfg)
	// Every install has at least one brand and one branch; an install that
	// predates them is migrated here (see repository/migrate.go).
	if err := repository.EnsureBrandAndBranch(ctx, store); err != nil {
		log.Fatalf("brand/branch migration: %v", err)
	}
	// Branches written before staff attendance existed get a real geofence
	// rather than the zero value, which reads as "no check at all".
	if err := repository.EnsureStaffDefaults(ctx, store); err != nil {
		log.Printf("staff defaults: %v", err)
	}
	// Orders that predate online payment were all settled at the door, so the
	// kitchen could start on them the moment they were placed.
	if err := repository.EnsureQueuedAt(ctx, store); err != nil {
		log.Printf("queuedAt migration: %v", err)
	}
	if err := repository.EnsureIndexes(ctx, store); err != nil {
		log.Printf("index setup: %v", err)
	}

	h := handlers.New(store, cfg)
	r := router.New(h, cfg)

	// Mirrors each connected till's stop list onto the site, so a dish the
	// kitchen stopped at the counter stops being orderable here too. Does
	// nothing at all on an install with no POS connected.
	syncCtx, stopSync := context.WithCancel(ctx)
	defer stopSync()
	h.StartPOSStopSync(syncCtx)

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
