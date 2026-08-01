// Command seedmenu writes the bundled demo menu (7 categories, 48 dishes with
// photos) into a database that already has data — the normal startup seed only
// runs on a completely empty database.
//
//	go run ./cmd/seedmenu -db restaurant            # add on top of what's there
//	go run ./cmd/seedmenu -db restaurant -replace   # wipe menu first (asks first)
//
// Only the `category` and `menu_item` collections are touched; orders, users and
// the restaurant profile are left alone.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"restaurant-backend/internal/config"
	"restaurant-backend/internal/db"
	"restaurant-backend/internal/repository"
	"restaurant-backend/internal/seed"

	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	dbName := flag.String("db", "", "database name (default: MONGO_DB from .env)")
	replace := flag.Bool("replace", false, "delete existing categories and menu items first")
	yes := flag.Bool("y", false, "skip the confirmation prompt")
	flag.Parse()

	cfg := config.Load()
	if *dbName != "" {
		cfg.MongoDB = *dbName
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("mongo: %v", err)
	}

	store := repository.New(database)
	cats, _ := store.Categories.CountDocuments(ctx, bson.M{})
	items, _ := store.Menu.CountDocuments(ctx, bson.M{})
	fmt.Printf("database %q: %d categories, %d menu items\n", cfg.MongoDB, cats, items)

	if *replace && (cats > 0 || items > 0) && !*yes {
		fmt.Printf("This deletes those %d categories and %d items. Continue? [y/N] ", cats, items)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(answer)), "y") {
			fmt.Println("cancelled")
			return
		}
	}

	if err := seed.WriteMenu(ctx, store, cfg, *replace); err != nil {
		log.Fatalf("seed menu: %v", err)
	}

	cats, _ = store.Categories.CountDocuments(ctx, bson.M{})
	items, _ = store.Menu.CountDocuments(ctx, bson.M{})
	fmt.Printf("done: %d categories, %d menu items\n", cats, items)
}
