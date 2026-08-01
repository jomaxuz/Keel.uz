// Command adminreset restores access to the admin panel when the password was
// changed and forgotten. It runs on the server (where the database lives), so
// physical/SSH access to the VPS is the authentication.
//
//	go run ./cmd/adminreset -list
//	go run ./cmd/adminreset -username yujo -password 'new-strong-pass'
//	go run ./cmd/adminreset -username yujo            # asks for the password
//	go run ./cmd/adminreset -username newowner -create
//
//	In Docker: docker compose -f docker-compose.prod.yml exec backend \
//	               /app/adminreset -username yujo -password '...'
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"restaurant-backend/internal/config"
	"restaurant-backend/internal/db"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func main() {
	username := flag.String("username", "", "admin username to reset")
	password := flag.String("password", "", "new password (omit to be prompted)")
	create := flag.Bool("create", false, "create the user if it does not exist")
	list := flag.Bool("list", false, "list admin usernames and exit")
	force := flag.Bool("force-change", false, "require a password change at next login")
	dbName := flag.String("db", "", "database name (default: MONGO_DB)")
	flag.Parse()

	cfg := config.Load()
	if *dbName != "" {
		cfg.MongoDB = *dbName
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("mongo: %v", err)
	}
	store := repository.New(database)

	if *list {
		cur, err := store.Admins.Find(ctx, bson.M{})
		if err != nil {
			log.Fatalf("list: %v", err)
		}
		var admins []models.AdminUser
		if err := cur.All(ctx, &admins); err != nil {
			log.Fatalf("list: %v", err)
		}
		fmt.Printf("admins in %q:\n", cfg.MongoDB)
		for _, a := range admins {
			fmt.Printf("  %-20s role=%-8s created=%s\n",
				a.Username, a.Role, a.CreatedAt.Format("2006-01-02"))
		}
		return
	}

	if *username == "" {
		log.Fatal("-username is required (use -list to see the existing ones)")
	}

	pass := *password
	if pass == "" {
		pass = promptPassword()
	}
	if len(pass) < 8 {
		log.Fatal("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}

	var existing models.AdminUser
	err = store.Admins.FindOne(ctx, bson.M{"username": *username}).Decode(&existing)
	switch {
	case err == nil:
		_, err = store.Admins.UpdateOne(ctx,
			bson.M{"_id": existing.ID},
			bson.M{"$set": bson.M{
				"passwordHash":       string(hash),
				"mustChangePassword": *force,
			}})
		if err != nil {
			log.Fatalf("update: %v", err)
		}
		fmt.Printf("password updated for %q\n", *username)
	case errors.Is(err, mongo.ErrNoDocuments):
		if !*create {
			log.Fatalf("admin %q not found (add -create to make a new one)", *username)
		}
		_, err = store.Admins.InsertOne(ctx, models.AdminUser{
			Username:           *username,
			PasswordHash:       string(hash),
			Role:               "owner",
			MustChangePassword: *force,
			CreatedAt:          time.Now(),
		})
		if err != nil {
			log.Fatalf("insert: %v", err)
		}
		fmt.Printf("admin %q created\n", *username)
	default:
		log.Fatalf("lookup: %v", err)
	}
}

// promptPassword reads the password without echoing it to the terminal.
func promptPassword() string {
	fmt.Print("New password: ")
	b, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		log.Fatalf("read password: %v", err)
	}
	pass := strings.TrimSpace(string(b))

	fmt.Print("Repeat: ")
	b2, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		log.Fatalf("read password: %v", err)
	}
	if pass != strings.TrimSpace(string(b2)) {
		fmt.Println("passwords do not match")
		os.Exit(1)
	}
	return pass
}
