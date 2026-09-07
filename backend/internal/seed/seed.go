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
	biz := models.BusinessType(cfg.BusinessType)
	name = cfg.BrandName
	if name == "" {
		// ⚠️ **A maker is neither.** "Restoran" over a bakery is the same
		// mistake as "My Restaurant" over a pharmacy, only quieter: it is a
		// word an owner half-accepts and leaves on their own receipts.
		switch {
		case biz.SellsGoods():
			name = "Do'kon"
		case biz == models.BizBakery:
			name = "Nonvoyxona"
		case biz == models.BizCoffee:
			name = "Qahvaxona"
		case biz == models.BizPastry:
			name = "Qandolatxona"
		default:
			name = "Restoran"
		}
	}
	// ⚠️ This sentence is printed on the site's home page until the owner writes
	// their own, so it is the kind of shop rather than "a business": "milliy va
	// zamonaviy taomlar" over a pharmacy is the same mistake as the name, and
	// "kundalik mahsulotlar" over a boutique is the smaller version of it.
	switch biz {
	case models.BizGrocery:
		return name, "Kundalik mahsulotlar — qulay narxlarda. Yetkazib berish shahar bo'ylab."
	case models.BizPharmacy:
		return name, "Dori-darmon va tibbiy buyumlar. Retseptsiz vositalar, har kuni ochiq."
	case models.BizClothing:
		return name, "Kiyim va aksessuarlar — yangi kolleksiya. O'lchamlar do'konda."
	case models.BizFlowers:
		return name, "Gullar va buketlar — har kuni yangi keltiriladi. Yetkazib berish shahar bo'ylab."
	case models.BizCosmetics:
		return name, "Parvarish va bo'yanish vositalari — asl mahsulotlar. Maslahat bilan tanlaymiz."
	case models.BizHardware:
		return name, "Qurilish va xo'jalik mollari — asbob, bo'yoq, mahkamlagich. Do'kondan olib ketasiz."
	case models.BizButcher:
		return name, "Har kuni yangi go'sht — tarozida tortib beriladi. Buyurtmani oldindan qoldirsangiz bo'ladi."
	case models.BizBakery:
		return name, "Har kuni yangi yopilgan non va patir — issiqligicha. Do'konlarga yetkazib beramiz."
	case models.BizCoffee:
		return name, "Qahva va ichimliklar — har bir stakan buyurtmadan keyin tayyorlanadi."
	case models.BizPastry:
		return name, "Tort, keks va shirinliklar — buyurtmaga tayyorlanadi. Yetkazib berish shahar bo'ylab."
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
