package seed

import (
	"context"
	"log"
	"time"

	"restaurant-backend/internal/config"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

// Bootstrap creates the initial admin user and a default restaurant document
// if they do not yet exist. Safe to run on every startup.
func Bootstrap(ctx context.Context, store *repository.Store, cfg *config.Config) {
	ensureAdmin(ctx, store, cfg)
	ensureRestaurant(ctx, store)
	ensureMenu(ctx, store, cfg)
}

func ensureAdmin(ctx context.Context, store *repository.Store, cfg *config.Config) {
	count, err := store.Admins.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Printf("seed: count admins: %v", err)
		return
	}
	if count > 0 {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("seed: hash password: %v", err)
		return
	}
	_, err = store.Admins.InsertOne(ctx, models.AdminUser{
		Username:           cfg.AdminUsername,
		PasswordHash:       string(hash),
		Role:               "owner",
		MustChangePassword: true, // force credential change on first login
		CreatedAt:          time.Now(),
	})
	if err != nil {
		log.Printf("seed: insert admin: %v", err)
		return
	}
	log.Printf("seed: created admin user %q", cfg.AdminUsername)
}

func ensureRestaurant(ctx context.Context, store *repository.Store) {
	count, err := store.Restaurant.CountDocuments(ctx, bson.M{})
	if err != nil || count > 0 {
		return
	}
	hours := make([]models.WorkingHour, 7)
	for i := 0; i < 7; i++ {
		hours[i] = models.WorkingHour{Day: i, Open: "10:00", Close: "23:00"}
	}
	_, _ = store.Restaurant.InsertOne(ctx, models.Restaurant{
		Name:         "My Restaurant",
		Description:  "Milliy va zamonaviy taomlar — har kuni yangi mahsulotlardan tayyorlanadi. Yetkazib berish shahar bo'ylab.",
		CoverURL:     imagePath("cover"),
		Phones:       []string{"+998 90 000 00 00"},
		Address:      models.GeoPoint{Text: "Toshkent", Lat: 41.311081, Lng: 69.240562},
		WorkingHours: hours,
		Currency:     "UZS",
		Delivery: models.DeliverySettings{
			Enabled:        true,
			MinOrder:       50000,
			BaseFee:        10000,
			PerKm:          3000,
			MaxKm:          15,
			ArrivalRadiusM: 150,
		},
		UpdatedAt: time.Now(),
	})
	log.Print("seed: created default restaurant")
}
