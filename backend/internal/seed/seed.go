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

// Bootstrap creates the initial admin user and a default profile if they do not
// yet exist. Safe to run on every startup.
func Bootstrap(ctx context.Context, store *repository.Store, cfg *config.Config) {
	ensureAdmin(ctx, store, cfg)
	ensureRestaurant(ctx, store, cfg)
	ensureMenu(ctx, store, cfg)
}

// firstProfile is the name and the description a brand-new install starts with.
//
// ⚠️ **The console knows the customer's name and nothing was passing it along.**
// Every install wrote the installer's placeholder — "My Restaurant" — into its
// profile, and because the brand and the branch are both created from that
// profile (repository.EnsureBrandAndBranch), a grocery was a grocery called My
// Restaurant: on its own website, on its receipts, in its Telegram messages and
// on the till's paper, until an owner noticed and retyped it. It is sent now
// (BRAND_NAME), and read once, on the boot that creates the profile.
//
// ⚠️ **And the fallback is not a restaurant either.** A tenant provisioned
// without a name — by hand, or by an older console — should start as the kind of
// thing it is. "Do'kon" is wrong for nobody who runs a shop; "My Restaurant" is
// wrong for all of them.
func firstProfile(cfg *config.Config) (name, description string) {
	shop := models.BusinessType(cfg.BusinessType).SellsGoods()
	name = cfg.BrandName
	if name == "" {
		if shop {
			name = "Do'kon"
		} else {
			name = "Restoran"
		}
	}
	if shop {
		// ⚠️ No dishes in it. This sentence is printed on the site's home page
		// until the owner writes their own, and "milliy va zamonaviy taomlar"
		// over a pharmacy is the same mistake as the name.
		return name, "Kundalik mahsulotlar — qulay narxlarda. Yetkazib berish shahar bo'ylab."
	}
	return name, "Milliy va zamonaviy taomlar — har kuni yangi mahsulotlardan tayyorlanadi. Yetkazib berish shahar bo'ylab."
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

func ensureRestaurant(ctx context.Context, store *repository.Store, cfg *config.Config) {
	count, err := store.Restaurant.CountDocuments(ctx, bson.M{})
	if err != nil || count > 0 {
		return
	}
	name, description := firstProfile(cfg)
	hours := make([]models.WorkingHour, 7)
	for i := 0; i < 7; i++ {
		hours[i] = models.WorkingHour{Day: i, Open: "10:00", Close: "23:00"}
	}
	_, _ = store.Restaurant.InsertOne(ctx, models.Restaurant{
		Name:         name,
		Description:  description,
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
	log.Printf("seed: created profile %q", name)
}
