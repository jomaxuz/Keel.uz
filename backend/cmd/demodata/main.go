// Command demodata fills a database with a month of a plausible restaurant's
// life: orders across every status, open checks on the floor, tickets on the
// pass, a cash shift, staff who clocked in, a stockroom with a food cost.
//
// It exists for the marketing screenshots (`docs/LANDING_REDESIGN.md` §5.1).
// ⚠️ **An empty screen is not a screenshot of a working product** — a table with
// no rows and a chart with no bars say "this does not work" far more loudly than
// they say "no data yet", and they say it on the page where a restaurant is
// deciding whether to buy.
//
//	go run ./cmd/demodata -db demo            # fill on top of what's there
//	go run ./cmd/demodata -db demo -wipe      # clear the demo data first (asks)
//
// ⚠️ **`-wipe` empties whole collections**, not "the rows this tool wrote":
// nothing here is tagged, and a marker field on every document would be a field
// the models do not have and the panel would eventually show. That makes the
// flag safe only on a scratch database, so it insists on `-db` being spelled out
// and on a confirmation that names the database and what is in it.
//
// The menu is not generated: run `cmd/seedmenu` first, or start `cmd/server`
// against an empty database and let the bundled seed do it. This tool cooks
// with whatever dishes it finds, so a restaurant's own menu works too.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"restaurant-backend/internal/config"
	"restaurant-backend/internal/db"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// ⚠️ **One seed, and it is a flag with a fixed default rather than the clock.**
// A screenshot gets retaken — a heading moves, a colour changes, somebody asks
// for the same screen in the other theme — and the second shot has to show the
// same restaurant as the first. With a clock-seeded generator every retake is a
// different day's takings, and the page ends up claiming two revenues for one
// dashboard.
var rng *rand.Rand

func main() {
	dbName := flag.String("db", "", "database name (default: MONGO_DB from .env)")
	days := flag.Int("days", 42, "how many days of history to write")
	seed := flag.Int64("seed", 20260830, "random seed; the same seed writes the same restaurant")
	wipe := flag.Bool("wipe", false, "empty the collections this tool writes before filling them")
	// ⚠️ **The flag that makes this safe to point at a live restaurant.** The
	// whole tool writes a month of invented orders, staff and takings, which is
	// right for a screenshot and wrong for a tenant whose owner is about to be
	// shown the dashboard: invented revenue in a real restaurant's system stops
	// being a demonstration the moment they sign up and it becomes their
	// history. `-stock` writes the stockroom and nothing else.
	stockOnly := flag.Bool("stock", false,
		"only the stockroom: ingredients, tech cards, deliveries, write-offs — no orders, staff or takings")
	allCards := flag.Bool("cards-all", false,
		"write a tech card for every dish, not only the third that sells most")
	wipeStock := flag.Bool("wipe-stock", false,
		"empty only the stock collections and clear every recipe, leaving orders and staff alone")
	replan := flag.Bool("replan", false, "redraw the dining room even if the branch already has one")
	yes := flag.Bool("y", false, "skip the confirmation prompt")
	flag.Parse()

	cfg := config.Load()
	if *dbName != "" {
		cfg.MongoDB = *dbName
	}
	if *wipe && *dbName == "" {
		log.Fatal("-wipe needs -db spelled out: it empties collections, and the database " +
			"it empties should never be the one that happened to be in .env")
	}
	if *wipe && *stockOnly {
		// ⚠️ Refused rather than silently narrowed. `-wipe` empties the orders,
		// the staff and the takings; somebody typing both flags has asked for
		// two different things, and guessing which one they meant on a live
		// tenant is the guess that cannot be undone.
		log.Fatal("-wipe and -stock together: -wipe empties orders and staff too. " +
			"Use -wipe-stock to clear only the stockroom")
	}

	rng = rand.New(rand.NewSource(*seed))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	database, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("mongo: %v", err)
	}
	store := repository.New(database)

	cats, _ := store.Categories.CountDocuments(ctx, bson.M{})
	items, _ := store.Menu.CountDocuments(ctx, bson.M{})
	fmt.Printf("database %q: %d categories, %d menu items\n", cfg.MongoDB, cats, items)
	if items == 0 {
		log.Fatal("no menu to cook from: run `go run ./cmd/seedmenu -db " + cfg.MongoDB + "` first")
	}

	if *wipe {
		if !*yes && !confirmWipe(ctx, store, cfg.MongoDB) {
			fmt.Println("cancelled")
			return
		}
		wipeAll(ctx, store)
	}

	if *wipeStock {
		if !*yes && !confirmWipeStock(ctx, store, cfg.MongoDB) {
			fmt.Println("cancelled")
			return
		}
		wipeStockroom(ctx, store)
	}

	w := newWorld(ctx, store, *days, *stockOnly)
	w.replan = *replan
	w.allCards = *allCards

	if *stockOnly {
		// ⚠️ **Refused rather than added to.** Run twice, this would write a
		// second "Asosiy ombor", a second copy of every ingredient and another
		// month of deliveries — and the balances, which are the whole point of
		// the screen, would double. The way to start again is stated rather
		// than left to be worked out with a mongo shell.
		if n, _ := store.Ingredients.CountDocuments(ctx, bson.M{}); n > 0 {
			log.Fatalf("this database already has %d ingredients. "+
				"Re-run with -wipe-stock to replace the stockroom, or leave it as it is", n)
		}
		w.stockroom(ctx)
		w.deliveries(ctx)
		fmt.Println("done — stockroom only. No orders, staff or takings were written.")
		return
	}

	w.subscription(ctx)
	w.floorPlan(ctx)
	w.people(ctx)
	w.stockroom(ctx)
	w.history(ctx)
	w.tonight(ctx)
	w.deliveries(ctx)
	w.tills(ctx)
	w.diary(ctx)

	fmt.Println("done. Seed", *seed, "— rerun with the same seed for the same restaurant.")
}

// ---------------------------------------------------------------- scaffolding

// The collections this tool owns end to end. ⚠️ `menu_item` is not among them:
// the recipes it writes are a field on somebody's dishes, and clearing the menu
// would take the photographs with it.
var owned = []string{
	"order", "user", "courier", "courier_settlement",
	"staff", "shift", "staff_payment",
	"cash_shift", "cash_entry",
	"reservation", "feedback", "visit",
	"warehouse", "ingredient", "ingredient_placement", "purchase", "writeoff",
	// ⚠️ These two were missing, and the gap was invisible: `-wipe` left the
	// old stocktakes and suppliers behind, the next run inserted more, and the
	// stock screens showed two counts of the same room a month apart.
	"stocktake", "supplier",
}

// stockOwned is the stockroom's half of `owned`, for the narrow wipe.
//
// ⚠️ **Derived by naming them again rather than by slicing `owned`.** A slice
// index is a comment that stops being true the moment somebody inserts a line
// above it, and what it would silently start deleting is the orders.
var stockOwned = []string{
	"warehouse", "ingredient", "ingredient_placement",
	"purchase", "writeoff", "stocktake", "supplier",
}

func confirmWipeStock(ctx context.Context, store *repository.Store, dbName string) bool {
	fmt.Printf("\nThis empties the stockroom in %q:\n", dbName)
	for _, name := range stockOwned {
		n, _ := store.DB.Collection(name).CountDocuments(ctx, bson.M{})
		if n > 0 {
			fmt.Printf("  %-22s %d documents\n", name, n)
		}
	}
	fmt.Printf("and clears the recipe from every menu item. " +
		"Orders, staff and takings are left alone.\nType the database name to continue: ")
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(answer) == dbName
}

// wipeStockroom clears the stock collections and the cards, and nothing else.
func wipeStockroom(ctx context.Context, store *repository.Store) {
	for _, name := range stockOwned {
		if _, err := store.DB.Collection(name).DeleteMany(ctx, bson.M{}); err != nil {
			log.Printf("wipe %s: %v", name, err)
		}
	}
	// ⚠️ The dish itself is never touched — only the two fields this tool
	// wrote. Clearing the menu would take the photographs with it, and on a
	// tenant whose menu was imported from their own site that is the one thing
	// nobody can put back.
	_, _ = store.Menu.UpdateMany(ctx, bson.M{}, bson.M{"$unset": bson.M{"recipe": "", "cost": ""}})
	fmt.Println("stockroom wiped")
}

func confirmWipe(ctx context.Context, store *repository.Store, dbName string) bool {
	fmt.Printf("\nThis empties these collections in %q:\n", dbName)
	for _, name := range owned {
		n, _ := store.DB.Collection(name).CountDocuments(ctx, bson.M{})
		if n > 0 {
			fmt.Printf("  %-22s %d documents\n", name, n)
		}
	}
	fmt.Printf("and clears the recipe from every menu item.\nType the database name to continue: ")
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(answer) == dbName
}

func wipeAll(ctx context.Context, store *repository.Store) {
	for _, name := range owned {
		if _, err := store.DB.Collection(name).DeleteMany(ctx, bson.M{}); err != nil {
			log.Printf("wipe %s: %v", name, err)
		}
	}
	_, _ = store.Menu.UpdateMany(ctx, bson.M{}, bson.M{"$unset": bson.M{"recipe": "", "cost": ""}})
	fmt.Println("wiped")
}

type dish struct {
	ID    primitive.ObjectID
	Name  string
	Price int
	Cat   string
}

// world is everything the generator needs to know about the restaurant it is
// filling: which branch cooks, what is on the menu, who works there.
type world struct {
	store  *repository.Store
	days   int
	now    time.Time
	brand  primitive.ObjectID
	branch models.Branch

	dishes []dish
	// The ones a place actually sells all day. Popularity is not uniform and a
	// dashboard built on a uniform menu has no best-seller, which is the one
	// row an owner looks at first.
	popular []dish

	staff    []models.Staff
	couriers []models.Courier
	guests   []models.User
	waiters  []models.Staff
	cashier  models.Staff

	// The stockroom, kept in memory because the deliveries are sized from what
	// the month actually cooked — see `deliveries`.
	ings      []models.Ingredient
	suppliers []models.Supplier
	mainStore primitive.ObjectID
	recipes   map[primitive.ObjectID][]models.RecipeLine
	// ingredient id -> units consumed by every order that was not cancelled.
	used map[primitive.ObjectID]float64

	replan bool
	// Write a card for every dish rather than for the third that sells most.
	allCards bool
}

func newWorld(ctx context.Context, store *repository.Store, days int, everyDish bool) *world {
	w := &world{
		store: store, days: days, now: time.Now(),
		recipes: map[primitive.ObjectID][]models.RecipeLine{},
		used:    map[primitive.ObjectID]float64{},
	}

	var br models.Brand
	if err := store.Brands.FindOne(ctx, bson.M{}).Decode(&br); err != nil {
		log.Fatalf("no brand in this database: %v", err)
	}
	w.brand = br.ID
	if err := store.Branches.FindOne(ctx, bson.M{"brandId": br.ID}).Decode(&w.branch); err != nil {
		log.Fatalf("no branch for brand %q: %v", br.Name, err)
	}
	fmt.Printf("filling %q / %q\n", br.Name, w.branch.Name)

	// ⚠️ **A tech card belongs to a dish whether or not it is on sale today**,
	// and on a freshly imported menu *nothing* is on sale: the importer leaves
	// dishes switched off on purpose, so somebody reads the prices before a
	// guest can order them. Filtering on availability here meant the stockroom
	// generator refused to run on the one menu it is most useful for, with
	// "every menu item is unavailable" — which is true and unhelpful.
	filter := bson.M{"isAvailable": true}
	limit := int64(200)
	if everyDish {
		filter, limit = bson.M{}, 1000
	}
	cur, err := store.Menu.Find(ctx, filter, options.Find().SetLimit(limit))
	if err != nil {
		log.Fatalf("menu: %v", err)
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var m models.MenuItem
		if cur.Decode(&m) != nil || m.Price <= 0 {
			continue
		}
		w.dishes = append(w.dishes, dish{ID: m.ID, Name: m.Name, Price: m.Price, Cat: m.CategoryID.Hex()})
	}
	if len(w.dishes) == 0 {
		log.Fatal("no priced dishes in this menu — nothing to cook from")
	}
	// A third of the card carries most of the sales, which is roughly what an
	// ABC report on a real restaurant says.
	n := len(w.dishes) / 3
	if n < 4 {
		n = len(w.dishes)
	}
	w.popular = w.dishes[:n]
	return w
}

func (w *world) pickDish() dish {
	// Four times in five the guest orders something from the popular third.
	if rng.Intn(5) > 0 {
		return w.popular[rng.Intn(len(w.popular))]
	}
	return w.dishes[rng.Intn(len(w.dishes))]
}

func oid() primitive.ObjectID { return primitive.NewObjectID() }

func insertMany[T any](ctx context.Context, c *mongo.Collection, docs []T, what string) {
	if len(docs) == 0 {
		return
	}
	any := make([]interface{}, len(docs))
	for i := range docs {
		any[i] = docs[i]
	}
	if _, err := c.InsertMany(ctx, any); err != nil {
		log.Printf("%s: %v", what, err)
		return
	}
	fmt.Printf("  %-22s %d\n", what, len(docs))
}

// ⚠️ Deterministic, unlike the real one in `handlers/util.go`, which draws from
// a CSPRNG because a guessable order number let strangers read each other's
// addresses. Nothing here is reachable from the internet and the same seed has
// to produce the same receipts, so this one is seeded — the alphabet is the
// real one's, so a number in a screenshot has the shape of a number.
func demoNumber() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	out := make([]byte, 0, 9)
	for i := 0; i < 8; i++ {
		if i == 4 {
			out = append(out, '-')
		}
		out = append(out, alphabet[rng.Intn(len(alphabet))])
	}
	return string(out)
}

var (
	firstNames = []string{
		"Aziz", "Dilnoza", "Sardor", "Malika", "Jasur", "Nilufar", "Bekzod", "Shahzoda",
		"Otabek", "Zilola", "Rustam", "Gulnora", "Sanjar", "Madina", "Farrux", "Kamola",
		"Ulug'bek", "Sevara", "Doston", "Nodira", "Alisher", "Charos", "Timur", "Umida",
	}
	lastNames = []string{
		"Karimov", "Yusupova", "Rahimov", "Tosheva", "Ergashev", "Sobirova",
		"Nazarov", "Qodirova", "Xolmatov", "Yo'ldosheva", "Ismoilov", "Abdullayeva",
	}
	streets = []string{
		"Chilonzor 19-kvartal", "Bunyodkor shoh ko'chasi", "Amir Temur ko'chasi",
		"Shota Rustaveli ko'chasi", "Yunusobod 4-mavze", "Mustaqillik shoh ko'chasi",
		"Katta Xirmontepa ko'chasi", "Beshqayrag'och mahallasi", "Farobiy ko'chasi",
		"Qatortol ko'chasi", "Uchtepa 2-mavze", "Kichik Xalqa yo'li",
	}
	addressNotes = []string{
		"2-podyezd, 4-qavat", "Domofon ishlamaydi, qo'ng'iroq qiling", "",
		"Kirish hovlidan", "", "Ofis, 3-qavat, 305-xona", "", "Do'kon yonida",
	}
)

func personName() string {
	return firstNames[rng.Intn(len(firstNames))] + " " + lastNames[rng.Intn(len(lastNames))]
}

func phone() string {
	codes := []string{"90", "91", "93", "94", "97", "98", "99", "88", "77", "33"}
	return fmt.Sprintf("+998 %s %03d %02d %02d",
		codes[rng.Intn(len(codes))], rng.Intn(1000), rng.Intn(100), rng.Intn(100))
}

func hash(pw string) string {
	h, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	return string(h)
}

// ---------------------------------------------------------------- what is sold

// subscription puts this install on the plan that opens every screen.
//
// ⚠️ **Without it half the panel answers 402.** The modules are gated at the
// handler (`requireModule`), so a demo database with no subscription document
// draws "this section is not in your plan" on the stock screens — a true
// sentence about an empty install, and a screenshot that tells a visitor the
// product's inventory does not work.
//
// ⚠️ It is the one thing here the platform normally writes, not the restaurant:
// on a real tenant this document comes from the Keel console. Written by hand
// only because a demo has no console behind it.
func (w *world) subscription(ctx context.Context) {
	now := time.Now()
	until := now.AddDate(0, 1, 0)
	sub := models.Subscription{
		ID: models.SubscriptionID, Enabled: true,
		Plan:      "pro",
		Modules:   []string{models.ModStock, models.ModMultiBranch, models.ModPOSIntegration},
		Registers: 3,
		Branches:  1,
		Monthly:   1_250_000,
		PaidUntil: &until,
		UpdatedAt: now,
	}
	_, err := w.store.DB.Collection("subscription").ReplaceOne(ctx,
		bson.M{"_id": models.SubscriptionID}, sub, options.Replace().SetUpsert(true))
	if err != nil {
		log.Printf("subscription: %v", err)
		return
	}
	fmt.Printf("  %-22s %s\n", "subscription", sub.Plan)
}

// ------------------------------------------------------------------ the floor

// floorPlan draws a dining room if the branch has none.
//
// ⚠️ **Left alone if there is one, and `-wipe` does not clear it.** The plan is
// something a restaurant arranges to match its actual room — the one thing here
// that is not reproducible from anything else, and the one a wipe would destroy
// silently. `-replan` is the separate, deliberate way to redraw it, kept out of
// `-wipe` precisely so "refill the demo" can never mean "rearrange the room".
func (w *world) floorPlan(ctx context.Context) {
	if len(w.branch.Booking.Tables) > 0 && !w.replan {
		fmt.Printf("  %-22s kept (%d tables)\n", "floor plan", len(w.branch.Booking.Tables))
		return
	}
	zone := models.TableZone{ID: "zal", Name: "Asosiy zal", Bookable: true, Layout: models.ZoneMap, Sort: 0}
	var tables []models.FloorTable
	// ⚠️ **Six across, not four.** The plan is drawn to its own aspect and
	// centred, so a nearly square room leaves a quarter of a landscape tablet
	// empty underneath it — and the empty band, not the room, is what the eye
	// lands on. Eighteen covers on a wide plan is also simply what a room that
	// takes bookings looks like; twelve reads as a canteen.
	const cols, rows = 6, 3
	for i := 0; i < cols*rows; i++ {
		col, row := i%cols, i/cols
		seats := []int{2, 4, 4, 6}[rng.Intn(4)]
		shape := "rect"
		if col == 0 || col == cols-1 {
			shape = "circle"
		}
		tables = append(tables, models.FloorTable{
			ID:       fmt.Sprintf("t%d", i+1),
			Number:   fmt.Sprintf("%d", i+1),
			Seats:    seats,
			Shape:    shape,
			X:        float64(70 + col*160),
			Y:        float64(80 + row*150),
			W:        95,
			H:        72,
			IsActive: true,
			ZoneID:   zone.ID,
		})
	}
	// The room's own furniture. A grid of tables floating on nothing is a
	// seating chart; a wall and a bar make it somewhere.
	shapes := []models.FloorShape{
		{Kind: "wall", X: 40, Y: 40, W: 1000, H: 10},
		{Kind: "wall", X: 40, Y: 40, W: 10, H: 480},
		{Kind: "area", Label: "Bar", X: 70, Y: 470, W: 260, H: 60},
		{Kind: "area", Label: "Kirish", X: 880, Y: 470, W: 160, H: 60},
	}
	b := w.branch.Booking
	b.Enabled = true
	b.Zones = []models.TableZone{zone}
	b.Tables = tables
	b.Shapes = shapes
	b.Width, b.Height = 1080, 560
	if b.SlotMinutes == 0 {
		b.SlotMinutes, b.MaxDaysAhead, b.MaxGuests = 30, 14, 12
	}
	w.branch.Booking = b
	_, err := w.store.Branches.UpdateByID(ctx, w.branch.ID, bson.M{"$set": bson.M{"booking": b}})
	if err != nil {
		log.Printf("floor plan: %v", err)
		return
	}
	fmt.Printf("  %-22s %d tables\n", "floor plan", len(tables))
}

// ------------------------------------------------------------------- the crew

func (w *world) people(ctx context.Context) {
	type role struct {
		pos                     string
		kitchen, waiter, cashir bool
		rate                    int
	}
	roles := []role{
		{"Menejer", false, false, true, 9_000_000},
		{"Kassir", false, false, true, 5_500_000},
		{"Ofitsiant", false, true, false, 4_500_000},
		{"Ofitsiant", false, true, false, 4_500_000},
		{"Oshpaz", true, false, false, 7_000_000},
		{"Oshpaz yordamchisi", true, false, false, 4_000_000},
	}
	for i, r := range roles {
		s := models.Staff{
			ID: oid(), BranchID: w.branch.ID,
			Name: personName(), Phone: phone(),
			Username:     fmt.Sprintf("xodim%d", i+1),
			PasswordHash: hash("demo12345"),
			Position:     r.pos,
			PayMode:      models.PayMonthly,
			MonthlyRate:  r.rate,
			PayPeriod:    models.PeriodHalfMon,
			CanKitchen:   r.kitchen, CanWaiter: r.waiter, CanCashier: r.cashir,
			PinHash:   hash(fmt.Sprintf("%04d", 1000+i*11)),
			IsActive:  true,
			CreatedAt: w.now.AddDate(0, -6, 0), UpdatedAt: w.now,
		}
		w.staff = append(w.staff, s)
		if r.waiter {
			w.waiters = append(w.waiters, s)
		}
	}
	w.cashier = w.staff[1]
	insertMany(ctx, w.store.Staff, w.staff, "staff")

	// Attendance. ⚠️ Filed by local calendar day, because that is what the
	// payroll screen groups by — a shift written in UTC lands on the day
	// before for everything that starts after 19:00 here.
	var shifts []models.Shift
	var payments []models.StaffPayment
	for _, s := range w.staff {
		for d := w.days; d >= 1; d-- {
			day := w.now.AddDate(0, 0, -d)
			if day.Weekday() == time.Monday {
				continue // one day off a week, and it shows on the chart
			}
			in := time.Date(day.Year(), day.Month(), day.Day(), 9, rng.Intn(20), 0, 0, time.Local)
			out := in.Add(time.Duration(10*60+rng.Intn(90)) * time.Minute)
			shifts = append(shifts, models.Shift{
				ID: oid(), StaffID: s.ID, BranchID: w.branch.ID,
				Date:      in.Format("2006-01-02"),
				In:        in,
				Out:       &out,
				Minutes:   int(out.Sub(in).Minutes()),
				CreatedAt: in, UpdatedAt: out,
			})
		}
		paid := w.now.AddDate(0, 0, -rng.Intn(10)-3)
		payments = append(payments, models.StaffPayment{
			ID: oid(), StaffID: s.ID, BranchID: w.branch.ID,
			Amount: s.MonthlyRate / 2,
			From:   paid.AddDate(0, 0, -15).Format("2006-01-02"),
			To:     paid.Format("2006-01-02"),
			PaidBy: "Menejer", Note: "Avans", At: paid,
		})
	}
	insertMany(ctx, w.store.Shifts, shifts, "shifts")
	insertMany(ctx, w.store.StaffPayments, payments, "staff payments")

	vehicles := []string{"moto", "car", "moto"}
	statuses := []models.CourierStatus{models.CourierBusy, models.CourierFree, models.CourierFree}
	for i := 0; i < 3; i++ {
		at := w.now.Add(-time.Duration(rng.Intn(9)+1) * time.Minute)
		w.couriers = append(w.couriers, models.Courier{
			ID: oid(), BranchID: w.branch.ID,
			Name: personName(), Phone: phone(),
			Username:     fmt.Sprintf("kuryer%d", i+1),
			PasswordHash: hash("demo12345"),
			Status:       statuses[i],
			Vehicle:      vehicles[i],
			IsActive:     true,
			Location: &models.CourierLocation{
				Lat:      w.branch.Address.Lat + (rng.Float64()-0.5)*0.03,
				Lng:      w.branch.Address.Lng + (rng.Float64()-0.5)*0.03,
				Accuracy: float64(10 + rng.Intn(30)),
				At:       at,
			},
			PayoutMode: models.PayoutPerOrder, PayoutPerOrder: 12000,
			CreatedAt: w.now.AddDate(0, -4, 0), UpdatedAt: at,
		})
	}
	insertMany(ctx, w.store.Couriers, w.couriers, "couriers")

	// Guests. ⚠️ Deliberately uneven: the CRM screens exist to separate the
	// regular from the one-off, and a base where everybody has ordered three
	// times has no segments in it at all.
	for i := 0; i < 60; i++ {
		// ⚠️ **Weighted towards recent, not spread flat.** Uniform over five
		// months puts one signup in the last week, and the dashboard's "new
		// customers" tile then reads 1 — a restaurant nobody is finding. A
		// place that is growing signs most of its base up recently, and
		// squaring a uniform draw is the cheapest way to say so.
		age := rng.Float64()
		created := w.now.AddDate(0, 0, -int(age*age*float64(w.days*4))-1)
		first := firstNames[rng.Intn(len(firstNames))]
		w.guests = append(w.guests, models.User{
			ID: oid(), FirstName: first, LastName: lastNames[rng.Intn(len(lastNames))],
			Phone: phone(), Source: []string{"web", "telegram", "operator"}[rng.Intn(3)],
			Lang: []string{"uz", "uz", "uz", "ru"}[rng.Intn(4)],
			Addresses: []models.UserAddress{{
				Label: "Uy",
				Text:  fmt.Sprintf("%s, %d-uy", streets[rng.Intn(len(streets))], rng.Intn(80)+1),
				Lat:   w.branch.Address.Lat + (rng.Float64()-0.5)*0.05,
				Lng:   w.branch.Address.Lng + (rng.Float64()-0.5)*0.05,
			}},
			CreatedAt: created, UpdatedAt: created,
		})
	}
	insertMany(ctx, w.store.Users, w.guests, "guests")
}

// --------------------------------------------------------------- the stockroom

type stockLine struct {
	name  string
	unit  string
	price int // per unit, so'm
}

// A kitchen's actual shopping list, priced roughly at Tashkent wholesale.
// ⚠️ The prices matter more than the names: the food cost on the menu screen is
// computed from them, and a dish whose cost comes out above its price is a
// screenshot that argues the product cannot add up.
// ⚠️ **The shelf has to cover the menu it is asked to cook, not the menu this
// tool was written against.** The first version held twenty Uzbek staples, and
// a card built from them for a Japanese restaurant reads "Sushi burger: lamb,
// onion, carrot" — visibly wrong to the one person the screen is being shown
// to, which is worse than an empty stock page. Widened to the shelves an
// ordinary city menu draws on, and `kitchen` below picks between them by the
// words in the dish's name.
var pantry = []stockLine{
	{"Mol go'shti", "kg", 95_000},
	{"Qo'y go'shti", "kg", 120_000},
	{"Tovuq filesi", "kg", 42_000},
	{"Guruch (lazer)", "kg", 18_000},
	{"Un (oliy nav)", "kg", 7_500},
	{"Kartoshka", "kg", 6_000},
	{"Piyoz", "kg", 4_500},
	{"Sabzi", "kg", 5_500},
	{"Pomidor", "kg", 12_000},
	{"Bodring", "kg", 9_000},
	{"Paxta yog'i", "l", 22_000},
	{"Sariyog'", "kg", 78_000},
	{"Tuxum", "dona", 1_300},
	{"Pishloq", "kg", 89_000},
	{"Smetana", "kg", 34_000},
	{"Bulka (burger)", "dona", 3_500},
	{"Ko'katlar", "bog'", 3_000},
	{"Tuz", "kg", 3_000},
	{"Ziravorlar", "kg", 65_000},
	{"Coca-Cola 0.5", "dona", 7_000},

	// ---- Fish, rice and the rest of a pan-Asian card ----
	{"Losos (file)", "kg", 165_000},
	{"Tunets (file)", "kg", 145_000},
	{"Krevetka", "kg", 118_000},
	{"Guruch (sushi)", "kg", 26_000},
	{"Nori", "dona", 2_500},
	{"Krem-pishloq", "kg", 72_000},
	{"Avokado", "kg", 46_000},
	{"Kunjut", "kg", 48_000},
	{"Soya sousi", "l", 34_000},
	{"Zanjabil (marinadlangan)", "kg", 38_000},
	{"Vasabi", "kg", 95_000},
	{"Ugra (lag'mon)", "kg", 16_000},
	{"Qo'ziqorin", "kg", 32_000},
	{"Bulg'or qalampiri", "kg", 16_000},
	{"Limon", "kg", 22_000},

	// ---- Sweet, and the bar ----
	{"Qaymoq 33%", "l", 46_000},
	{"Shakar", "kg", 12_000},
	{"Shokolad", "kg", 98_000},
	{"Muzqaymoq", "kg", 45_000},
	{"Choy (damlama)", "kg", 120_000},
	{"Kofe (don)", "kg", 145_000},
	{"Suv 0.5", "dona", 3_000},
	{"Apelsin (fresh)", "kg", 19_000},
}

// bar names the shelves that live behind the counter rather than in the
// kitchen. ⚠️ By name rather than by a flag on stockLine: the split is a fact
// about this restaurant's rooms, not about the ingredient, and a bottle of
// water is a kitchen item in a place with one store.
var barShelf = map[string]bool{
	"Coca-Cola 0.5": true, "Suv 0.5": true, "Choy (damlama)": true,
	"Kofe (don)": true, "Apelsin (fresh)": true, "Limon": true,
}

// kitchen is what a dish is probably made of, judged by the words in its name.
//
// ⚠️ **Three languages of the same word, because the menu was imported.** A
// card imported from a delivery site carries whichever language the owner
// happened to publish — `Losos`, `Лосось` and `Salmon` are the same fish, and
// matching only one of them files two thirds of the menu under the fallback.
//
// ⚠️ **Order matters: the narrow words come first.** "Sushi burger" contains
// both `sushi` and `burger`, and it is a roll. Whichever rule is written first
// wins, so the specific shelves are listed above the general ones.
// ⚠️ **Some dishes are deliberately left without a card**, and a set is the
// clearest case. "Set №10" at 422 000 so'm is a platter of six other dishes —
// it has no ingredients of its own, and the model says so: a combo reaches the
// stockroom by being **exploded into its members** (`soldDishes`), so giving it
// its own recipe would count the same fish twice. Inventing 0.32 kg of lamb for
// it, which is what the price band does, is wrong twice over: wrong on the
// plate and wrong in the cost.
var uncarded = []string{
	"set ", "set№", "set №", "сет", "набор", "combo", "комбо",
	// ⚠️ "To'plam" is the same word in Uzbek, and an imported menu is in
	// whichever language the owner published. Missing it left a 280 000 so'm
	// platter carrying 0.32 kg of lamb at 13.9% food cost.
	"to'plam", "toplam", "тўплам",
	"assorti", "ассорти", "platter", "banket", "банкет",
}

// shelf is one rule: what words point at it, and what it is made of.
type shelf struct {
	words []string
	// ⚠️ **An optional second word that must also be present.** "Avokado maki"
	// and "Maki bodring" are rolls with no fish in them, and the generic roll
	// rule builds both out of salmon: wrong on the plate, and 62% food cost on
	// a 21 000 so'm dish. Word order varies between menus and languages, so the
	// pair is matched rather than a phrase.
	and []string
	// Preferred mains, most likely first. The first one on the shelf is used.
	main []string
	// What goes beside it. Kept short: a card with eleven lines is not a card
	// anybody reads, and this is a demonstration of the screen.
	trim []string
}

var kitchen = []shelf{
	// ---- Rolls, split by what is actually in them ----
	//
	// ⚠️ The narrow pairs come first: every one of these also contains a roll
	// word, and whichever rule is written first wins.
	{rolls, []string{"avokado", "авокадо", "bodring", "огурец", "ogurec", "chuka", "чука", "ovoshn", "овощн", "vegetarian", "вегетариан"},
		[]string{"Avokado", "Bodring"},
		[]string{"Guruch (sushi)", "Nori", "Krem-pishloq", "Kunjut"}},
	{rolls, []string{"krevet", "креветк", "ebi", "shrimp"},
		[]string{"Krevetka"},
		[]string{"Guruch (sushi)", "Nori", "Krem-pishloq", "Avokado", "Kunjut"}},
	{rolls, []string{"tunets", "тунц", "тунец", "tuna"},
		[]string{"Tunets (file)"},
		[]string{"Guruch (sushi)", "Nori", "Avokado", "Kunjut"}},
	{rolls, []string{"tovuq", "куриц", "chicken"},
		[]string{"Tovuq filesi"},
		[]string{"Guruch (sushi)", "Nori", "Krem-pishloq", "Bodring", "Kunjut"}},
	{rolls, nil,
		[]string{"Losos (file)"},
		[]string{"Guruch (sushi)", "Nori", "Krem-pishloq", "Avokado", "Bodring", "Kunjut"}},

	{[]string{"sashimi", "сашими", "tatar", "тартар", "poke", "поке"}, nil,
		[]string{"Losos (file)", "Tunets (file)"},
		[]string{"Guruch (sushi)", "Avokado", "Soya sousi", "Kunjut", "Limon"}},
	{[]string{"krevet", "креветк", "shrimp", "ebi", "tom yam", "том ям"}, nil,
		[]string{"Krevetka"},
		[]string{"Guruch (sushi)", "Qaymoq 33%", "Bulg'or qalampiri", "Limon", "Ziravorlar"}},
	{[]string{"losos", "лосось", "salmon", "baliq", "рыба", "fish", "tunets", "тунец", "unagi", "угорь"}, nil,
		[]string{"Losos (file)", "Tunets (file)"},
		[]string{"Limon", "Sariyog'", "Ko'katlar", "Ziravorlar"}},
	// ⚠️ **A bowl of noodles is priced on what is in it, not on the noodles.**
	// With the noodles as the main, a 75 000 so'm beef ramen costs 6 700 to
	// make — 8.9%, which reads as a pricing error rather than as a menu. The
	// protein leads and the noodles are a bulk line (see `bulky`).
	{noodles, []string{"go'sht", "мясо", "мяс", "beef", "buzoq"},
		[]string{"Mol go'shti"},
		[]string{"Ugra (lag'mon)", "Bulg'or qalampiri", "Piyoz", "Soya sousi"}},
	{noodles, []string{"tovuq", "куриц", "chicken"},
		[]string{"Tovuq filesi"},
		[]string{"Ugra (lag'mon)", "Bulg'or qalampiri", "Sabzi", "Soya sousi"}},
	{noodles, []string{"krevet", "креветк", "shrimp", "baliq", "losos"},
		[]string{"Krevetka"},
		[]string{"Ugra (lag'mon)", "Bulg'or qalampiri", "Piyoz", "Soya sousi"}},
	{noodles, nil,
		[]string{"Ugra (lag'mon)"},
		[]string{"Tovuq filesi", "Bulg'or qalampiri", "Piyoz", "Sabzi", "Soya sousi"}},
	{[]string{"burger", "бургер", "sendvich", "сэндвич", "hot dog", "хот-дог"}, nil,
		[]string{"Bulka (burger)"},
		[]string{"Mol go'shti", "Pishloq", "Pomidor", "Bodring", "Piyoz"}},
	{[]string{"pizza", "пицца"}, nil,
		[]string{"Un (oliy nav)"},
		[]string{"Pishloq", "Pomidor", "Qo'ziqorin", "Bulg'or qalampiri", "Paxta yog'i"}},

	// ---- Salads: the protein is what the name says, and it decides the cost ----
	{salads, []string{"nisuaz", "нисуаз", "tunets", "тунец"},
		[]string{"Tunets (file)"},
		[]string{"Bodring", "Pomidor", "Ko'katlar", "Limon", "Paxta yog'i"}},
	{salads, []string{"sezar", "цезарь", "tovuq", "куриц", "chicken"},
		[]string{"Tovuq filesi"},
		[]string{"Pishloq", "Ko'katlar", "Bulg'or qalampiri", "Paxta yog'i"}},
	{salads, []string{"losos", "лосось", "krevet", "креветк", "salmon"},
		[]string{"Losos (file)", "Krevetka"},
		[]string{"Bodring", "Avokado", "Ko'katlar", "Limon"}},
	{salads, nil,
		[]string{"Bodring", "Pomidor"},
		[]string{"Ko'katlar", "Bulg'or qalampiri", "Pishloq", "Paxta yog'i", "Limon"}},

	// ⚠️ The protein rules sit **above** the sauce and the cooking method:
	// "Teriyaki sousidagi buzoq go'shti" is veal, and a rule that reads
	// "teriyaki" first builds it out of chicken.
	{[]string{"go'sht", "gosht", "мясо", "мяс", "beef", "buzoq", "steyk", "стейк", "medalyon", "медальон", "befstroganov", "бефстроганов", "shashlik", "шашлык", "kabob", "qo'y", "баран"}, nil,
		[]string{"Mol go'shti", "Qo'y go'shti"},
		[]string{"Piyoz", "Qo'ziqorin", "Qaymoq 33%", "Ziravorlar", "Paxta yog'i"}},
	{[]string{"tovuq", "курица", "куриц", "chicken", "file"}, nil,
		[]string{"Tovuq filesi"},
		[]string{"Soya sousi", "Piyoz", "Bulg'or qalampiri", "Ziravorlar", "Paxta yog'i"}},
	{[]string{"teriyaki", "терияки", "gril", "гриль", "yaki"}, nil,
		[]string{"Tovuq filesi", "Mol go'shti"},
		[]string{"Soya sousi", "Bulg'or qalampiri", "Piyoz", "Kunjut"}},

	{[]string{"sho'rva", "шурпа", "sup", "суп", "soup", "mastava", "мастава", "miso", "мисо"}, nil,
		[]string{"Mol go'shti", "Tovuq filesi"},
		[]string{"Kartoshka", "Sabzi", "Piyoz", "Ko'katlar", "Tuz"}},
	{[]string{"osh", "плов", "plov", "guruch", "рис", "rice"}, nil,
		[]string{"Guruch (lazer)"},
		[]string{"Qo'y go'shti", "Sabzi", "Piyoz", "Paxta yog'i", "Ziravorlar"}},
	{[]string{"desert", "десерт", "tort", "торт", "chizkeyk", "чизкейк", "muzqaymoq", "мороженое", "shokolad", "шоколад", "cake", "roll kek"}, nil,
		[]string{"Muzqaymoq", "Shokolad"},
		[]string{"Qaymoq 33%", "Shakar", "Sariyog'", "Tuxum"}},
	{[]string{"kofe", "кофе", "coffee", "latte", "латте", "kapuchino", "капучино", "amerikano", "американо", "espresso", "эспрессо", "raf", "раф"}, nil,
		[]string{"Kofe (don)"},
		[]string{"Qaymoq 33%", "Shakar"}},
	{[]string{"choy", "чай", "tea", "matcha", "матча"}, nil,
		[]string{"Choy (damlama)"},
		[]string{"Shakar", "Limon"}},
	{[]string{"fresh", "фреш", "sok", "сок", "juice", "limonad", "лимонад", "smuzi", "смузи", "mors", "морс"}, nil,
		[]string{"Apelsin (fresh)"},
		[]string{"Shakar", "Limon"}},
	{[]string{"suv", "вода", "water", "cola", "кола", "напиток", "ichimlik", "pepsi", "fanta", "sprite"}, nil,
		[]string{"Suv 0.5", "Coca-Cola 0.5"},
		nil},
	{[]string{"non", "лепешк", "хлеб", "bread", "garnir", "гарнир", "fri", "фри", "kartoshka", "картош"}, nil,
		[]string{"Kartoshka", "Un (oliy nav)"},
		[]string{"Paxta yog'i", "Tuz", "Ziravorlar"}},
}

// The words that say "this is a roll" and "this is a salad", named once because
// each is the first half of several rules above.
var rolls = []string{
	"sushi", "суши", "roll", "ролл", "маки", "maki", "gunkan", "гункан",
	"onigiri", "онигири", "filadel", "филадел", "kaliforn", "калифорн",
	"urama", "урамаки", "nigiri", "нигири", "temaki",
}

var salads = []string{"salat", "салат", "salad", "sezar", "цезарь", "gretsk", "греческ", "nisuaz", "нисуаз"}

var noodles = []string{
	"lag'mon", "lagmon", "лагман", "udon", "удон", "ramen", "рамен",
	"kuksi", "кукси", "ugra", "noodle", "wok", "вок", "yakisoba", "якисоба",
	"soba", "соба", "funchoza", "фунчоза",
}

// bulky names the shelves that are a plate's *body* rather than a garnish.
//
// ⚠️ **Rice, noodles and potato are trimmings by position and portions by
// weight.** Sized with the other trimmings — twenty to eighty grams — a bowl of
// ramen holds a spoonful of noodles, and the cost the card produces is a cost
// nobody could cook to.
var bulky = map[string]bool{
	"Guruch (sushi)": true, "Guruch (lazer)": true, "Ugra (lag'mon)": true,
	"Kartoshka": true, "Un (oliy nav)": true, "Muzqaymoq": true,
}

func (w *world) stockroom(ctx context.Context) {
	wh := models.Warehouse{
		ID: oid(), BranchID: w.branch.ID, Name: "Asosiy ombor", Kind: "kitchen",
		IsActive: true, CreatedAt: w.now.AddDate(0, -6, 0), UpdatedAt: w.now,
	}
	bar := models.Warehouse{
		ID: oid(), BranchID: w.branch.ID, Name: "Bar", Kind: "bar", Sort: 1,
		IsActive: true, CreatedAt: w.now.AddDate(0, -6, 0), UpdatedAt: w.now,
	}
	insertMany(ctx, w.store.Warehouses, []models.Warehouse{wh, bar}, "warehouses")

	suppliers := []models.Supplier{
		{ID: oid(), BrandID: w.brand, Name: "Anhor Savdo", Phone: phone(), IsActive: true, CreatedAt: w.now},
		{ID: oid(), BrandID: w.brand, Name: "Farg'ona Go'sht", Phone: phone(), IsActive: true, Sort: 1, CreatedAt: w.now},
		{ID: oid(), BrandID: w.brand, Name: "Yashil Bozor", Phone: phone(), IsActive: true, Sort: 2, CreatedAt: w.now},
	}
	insertMany(ctx, w.store.Suppliers, suppliers, "suppliers")
	w.suppliers, w.mainStore = suppliers, wh.ID

	var ings []models.Ingredient
	var places []models.IngredientPlacement
	for _, p := range pantry {
		id := oid()
		store := wh.ID
		if barShelf[p.name] {
			store = bar.ID
		}
		ings = append(ings, models.Ingredient{
			ID: id, BrandID: w.brand, Name: p.name, Unit: p.unit,
			Price:     p.price,
			History:   []models.PriceEntry{{Price: p.price, At: w.now.AddDate(0, 0, -w.days)}},
			UpdatedAt: w.now,
		})
		// ⚠️ Brand-wide ingredient, branch-owned shelf — the split the model
		// insists on (CLAUDE.md §4): one catalogue, one store per branch.
		places = append(places, models.IngredientPlacement{
			ID: oid(), BranchID: w.branch.ID, IngredientID: id,
			WarehouseID: store, UpdatedAt: w.now,
		})
	}
	insertMany(ctx, w.store.Ingredients, ings, "ingredients")
	insertMany(ctx, w.store.Placements, places, "placements")
	w.ings = ings

	// Tech cards. ⚠️ **Only the popular third by default**: a card on every item
	// would be a demo claiming somebody typed eighty recipes in, and the "no
	// card yet" state is one the stock screens are built to show.
	//
	// ⚠️ `-cards-all` is for the other job this tool does — standing a real
	// restaurant's own menu up before its owner sees it. There, a third of the
	// dishes priced and the rest blank does not read as an honest default; it
	// reads as an import that half worked.
	cards := w.popular
	if w.allCards {
		cards = w.dishes
	}
	written := 0
	for _, d := range cards {
		lines, cost := w.recipeFor(d, ings)
		if len(lines) == 0 {
			continue
		}
		_, err := w.store.Menu.UpdateByID(ctx, d.ID,
			bson.M{"$set": bson.M{"recipe": lines, "cost": cost}})
		if err == nil {
			w.recipes[d.ID] = lines
			written++
		}
	}
	fmt.Printf("  %-22s %d\n", "tech cards", written)
}

// deliveries writes the month's buying, the losses and the last count.
//
// ⚠️ **Sized from what the kitchen actually cooked**, which is why it runs after
// the orders rather than beside the ingredients. Bought by a plausible-looking
// constant instead, the shelf holds two hundred and ninety kilos of lamb and
// one item goes *negative* — and both are on the same screen, so the number a
// visitor checks first is either absurd or red. The stock screen's whole claim
// is that it knows what is in the room.
func (w *world) deliveries(ctx context.Context) {
	w.assumeUsage()
	// Per-day usage, from the tech cards and the orders that were not cancelled.
	daily := map[primitive.ObjectID]float64{}
	for id, qty := range w.used {
		daily[id] = qty / float64(w.days)
	}
	round := func(in models.Ingredient, v float64) float64 {
		// Eggs and buns come in pieces; nobody buys 4.37 of them.
		if in.Unit == "dona" || in.Unit == "bog'" {
			return float64(int(v + 0.5))
		}
		return float64(int(v*10)) / 10
	}

	// The count that anchors the balance, a fortnight back: roughly a week of
	// stock on the shelf, which is what a kitchen that orders weekly has.
	countAt := w.now.AddDate(0, 0, -14)
	counted := map[primitive.ObjectID]float64{}
	var lines []models.StocktakeLine
	value := 0
	for _, in := range w.ings {
		q := round(in, daily[in.ID]*7)
		if q <= 0 {
			q = round(in, 3) // something nobody has cooked with yet
		}
		counted[in.ID] = q
		lines = append(lines, models.StocktakeLine{
			IngredientID: in.ID, Counted: q, Expected: q + round(in, q*0.02),
			Diff: -round(in, q*0.02), Value: int(q * float64(in.Price)),
		})
		value += int(q * float64(in.Price))
	}
	_, err := w.store.Stocktakes.InsertOne(ctx, models.Stocktake{
		ID: oid(), BranchID: w.branch.ID, WarehouseID: w.mainStore, At: countAt,
		Lines: lines, Value: value, By: "Menejer", CreatedAt: countAt,
	})
	if err != nil {
		log.Printf("stocktake: %v", err)
	} else {
		fmt.Printf("  %-22s 1 (%s)\n", "stocktake", countAt.Format("2006-01-02"))
	}

	// ⚠️ Only the deliveries **after** the count move the balance — the ones
	// before it are already inside the counted figure. They are written anyway,
	// because the buying screen and the supplier report are a history and a
	// month with one delivery in it is not one.
	weeks := w.days / 7
	var purchases []models.Purchase
	for week := weeks; week >= 0; week-- {
		at := w.now.AddDate(0, 0, -week*7-rng.Intn(3))
		for si, sup := range w.suppliers {
			var pl []models.PurchaseLine
			total := 0
			for i, in := range w.ings {
				if i%len(w.suppliers) != si {
					continue
				}
				// A week's cooking, plus a little. After the count there are
				// two deliveries left before today, so the shelf lands at
				// roughly ten days of stock — full, and nowhere near negative.
				qty := round(in, daily[in.ID]*7*(1.1+rng.Float64()*0.2))
				if qty <= 0 {
					qty = round(in, 2)
				}
				pl = append(pl, models.PurchaseLine{IngredientID: in.ID, Qty: qty, Price: in.Price})
				total += int(qty * float64(in.Price))
			}
			// Everything but the most recent week is settled up, so the
			// buying screen has both states on it.
			paid := week > 0
			var paidAt *time.Time
			if paid {
				t := at.Add(48 * time.Hour)
				paidAt = &t
			}
			purchases = append(purchases, models.Purchase{
				ID: oid(), BranchID: w.branch.ID, At: at,
				SupplierID: sup.ID, Supplier: sup.Name,
				Paid: paid, PaidAt: paidAt,
				Lines: pl, Total: total,
				CreatedBy: "Menejer", CreatedAt: at,
			})
		}
	}
	insertMany(ctx, w.store.Purchases, purchases, "purchases")

	// Losses: small, and small against the usage rather than a flat half-kilo,
	// which on eggs is six of them and on lamb is a joint.
	var offs []models.WriteOff
	reasons := []string{"Yaroqlilik muddati", "Sifatsiz mahsulot", "Tayyorlashda yo'qotish", "Sinov taomi"}
	for i := 0; i < 9; i++ {
		in := w.ings[rng.Intn(len(w.ings))]
		qty := round(in, daily[in.ID]*(0.1+rng.Float64()*0.2))
		if qty <= 0 {
			qty = round(in, 1)
		}
		at := w.now.AddDate(0, 0, -rng.Intn(13))
		offs = append(offs, models.WriteOff{
			ID: oid(), BranchID: w.branch.ID, At: at,
			IngredientID: in.ID, Qty: qty,
			Reason: reasons[rng.Intn(len(reasons))],
			Value:  int(qty * float64(in.Price)),
			By:     "Oshpaz", CreatedAt: at,
		})
	}
	insertMany(ctx, w.store.WriteOffs, offs, "write-offs")

	// The minimum is three days of cooking: it is the number that makes the
	// shopping list a list rather than every row at once, and it is how a chef
	// actually sets one.
	for _, in := range w.ings {
		min := round(in, daily[in.ID]*3)
		if min <= 0 {
			min = 1
		}
		_, _ = w.store.Ingredients.UpdateByID(ctx, in.ID, bson.M{"$set": bson.M{"minQty": min}})
	}
}

// assumeUsage stands in for a month of cooking when there was none.
//
// ⚠️ **Only when nothing was cooked**, which is the `-stock` run: no orders were
// written, so `used` is empty and every quantity below would fall to its
// "nobody has cooked with this yet" default — three of everything, bought two at
// a time, a minimum of one. The shelf then holds three kilos of salt beside
// three kilos of salmon, and the shopping list and the ABC report are drawn from
// numbers that mean nothing. It is a demonstration of the screen, so the screen
// has to be showing something a chef would recognise.
//
// ⚠️ It is an assumption and is written down as one: a plausible daily cover,
// spread over the cards. The alternative — inventing the orders as well — is
// exactly what `-stock` exists to avoid.
func (w *world) assumeUsage() {
	if len(w.used) > 0 || len(w.recipes) == 0 {
		return
	}
	popular := map[primitive.ObjectID]bool{}
	for _, d := range w.popular {
		popular[d.ID] = true
	}
	// ⚠️ **A day's covers, divided across the menu — not a portion count per
	// dish.** Fixed per dish, a hundred and nineteen cards at six a day is four
	// hundred and seventy covers, and the shelf ends up holding four hundred
	// kilos of lamb. What a restaurant has is a number of covers; how thinly
	// that spreads is the menu's business, and a longer menu means fewer of
	// each rather than a busier kitchen.
	const coversPerDay = 140
	// A third of the card carries most of them, which is roughly what an ABC
	// report on a real restaurant says — the same split `newWorld` uses.
	nPop := len(w.popular)
	nRest := len(w.dishes) - nPop
	perPopular, perRest := 0.0, 0.0
	if nPop > 0 {
		perPopular = coversPerDay * 0.6 / float64(nPop)
	}
	if nRest > 0 {
		perRest = coversPerDay * 0.4 / float64(nRest)
	}
	for dishID, lines := range w.recipes {
		perDay := perRest
		if popular[dishID] {
			perDay = perPopular
		}
		for _, l := range lines {
			w.used[l.IngredientID] += l.Qty * perDay * float64(w.days)
		}
	}
}

// recipeFor invents a plausible tech card: a protein or a base, two or three
// vegetables, oil and seasoning.
//
// ⚠️ **Portions are sized in the kitchen's units, and the cost comes out of
// them** — not the other way round. The first version worked backwards from a
// target food cost, splitting a third of the menu price across the lines, and
// on a cheap ingredient that arithmetic asks for **five kilos of carrot in one
// portion**. Nothing rejects it: the card saves, the cost looks sane, and the
// error surfaces three screens away as a stockroom holding 349 kg of carrots
// and a shopping list that never stops asking for more.
func (w *world) recipeFor(d dish, ings []models.Ingredient) ([]models.RecipeLine, int) {
	pick := func(names ...string) []models.Ingredient {
		var out []models.Ingredient
		for _, n := range names {
			for _, in := range ings {
				if in.Name == n {
					out = append(out, in)
				}
			}
		}
		return out
	}
	if noCard(d.Name) {
		return nil, 0
	}

	// ⚠️ **The dish's own name first, its price second.** A card is read by the
	// person who cooks from it, and "Losos: lamb, onion, carrot" is not a
	// mistake anybody has to think about — it is simply wrong on the screen
	// being demonstrated. So the name picks the shelf where it can, and the
	// price band is what answers for a dish whose name says nothing.
	var base, trim []models.Ingredient
	if k := shelfFor(d.Name); k != nil {
		base, trim = pick(k.main...), pick(k.trim...)
	}

	// ⚠️ **The main ingredient is otherwise chosen by what the dish costs.** A
	// 145 000 so'm steak whose card is built on rice cannot reach a believable
	// food cost at any portion size a plate can hold — the sizing below then
	// clamps, and the margin column reads 4%. Matching the band is what makes
	// the clamp a safety rail rather than the thing that decides every card.
	if len(base) == 0 {
		switch {
		case d.Price >= 60_000:
			base = pick("Qo'y go'shti", "Mol go'shti")
		case d.Price >= 25_000:
			base = pick("Mol go'shti", "Tovuq filesi", "Guruch (lazer)")
		default:
			base = pick("Un (oliy nav)", "Kartoshka", "Tovuq filesi")
		}
	}
	if len(base) == 0 {
		base = pick("Mol go'shti", "Guruch (lazer)", "Tovuq filesi", "Un (oliy nav)", "Kartoshka")
	}
	if len(trim) == 0 {
		trim = pick("Piyoz", "Sabzi", "Pomidor", "Bodring", "Ko'katlar", "Paxta yog'i", "Tuz", "Ziravorlar")
	}
	if len(base) == 0 || len(trim) == 0 {
		return nil, 0
	}
	// A portion, in the unit the shelf is counted in. lo/hi per unit rather
	// than one range: 0.15 of a bun is not a thing, and 1 kg of salt is a sack.
	portion := func(in models.Ingredient, main bool) float64 {
		var lo, hi float64
		// ⚠️ **The body of the plate is portioned by weight even when it is not
		// the main — but not as generously as a main.** Rice beside fish is a
		// hundred grams, not two hundred and fifty: sized as a main it puts
		// 0.16 kg of rice into one gunkan, which is the line a chef reads and
		// stops trusting the card.
		if bulky[in.Name] && !main && in.Unit == "kg" {
			v := 0.08 + rng.Float64()*0.1
			return float64(int(v*1000)) / 1000
		}
		switch in.Unit {
		case "dona":
			lo, hi = 1, 2
		case "l":
			lo, hi = 0.01, 0.04
		case "bog'":
			lo, hi = 0.05, 0.2
		default: // kg
			if main {
				lo, hi = 0.12, 0.25
			} else {
				lo, hi = 0.02, 0.08
			}
		}
		// Salt and spice are grams whatever else is true.
		if in.Name == "Tuz" || in.Name == "Ziravorlar" {
			lo, hi = 0.002, 0.008
		}
		v := lo + rng.Float64()*(hi-lo)
		if in.Unit == "dona" {
			return float64(int(v + 0.5))
		}
		return float64(int(v*1000)) / 1000
	}

	// ⚠️ The **first** of a matched shelf, not a random one: `main` is listed
	// most-likely-first, and a roll whose card is built on tuna half the time
	// is a menu nobody wrote.
	main := base[0]
	var lines []models.RecipeLine
	cost := 0
	seen := map[primitive.ObjectID]bool{main.ID: true}
	for i := 0; i < 2+rng.Intn(2); i++ {
		in := trim[rng.Intn(len(trim))]
		if seen[in.ID] {
			continue
		}
		seen[in.ID] = true
		qty := portion(in, false)
		lines = append(lines, models.RecipeLine{IngredientID: in.ID, Qty: qty})
		cost += int(qty * float64(in.Price))
	}

	// On a cheap dish the trimmings alone can pass the target — a 6 000 so'm
	// flatbread does not carry 8 000 so'm of oil and spice. Scaled back
	// together so the card keeps its shape, rather than dropping lines.
	target0 := float64(d.Price) * 0.3
	if float64(cost) > target0*0.6 && cost > 0 {
		f := target0 * 0.6 / float64(cost)
		cost = 0
		for i := range lines {
			q := lines[i].Qty * f
			lines[i].Qty = float64(int(q*1000)) / 1000
			for _, in := range trim {
				if in.ID != lines[i].IngredientID {
					continue
				}
				// ⚠️ **Rounded back to a whole one.** Scaling turns a sheet of
				// nori into 1.191 sheets, and a card asking for a fifth of a
				// sheet is a card the chef stops reading — the fraction is not
				// a rounding detail, it is the line that makes the screen look
				// generated. Never below one: a roll with no nori in it is not
				// a smaller portion, it is a mistake.
				if in.Unit == "dona" {
					if lines[i].Qty = float64(int(lines[i].Qty + 0.5)); lines[i].Qty < 1 {
						lines[i].Qty = 1
					}
				}
				cost += int(lines[i].Qty * float64(in.Price))
			}
		}
	}

	// ⚠️ **The main portion is what closes the gap to a believable food cost.**
	// Left to a range, a rice dish came out at 4% and a mince dish at 53% — the
	// margin column then reads as a bug rather than as a menu, and the one
	// screen that has to be trusted on a marketing page is the one about money.
	// So the trimmings are portions and the protein is sized to the plate: aimed
	// at just under a third of the price, and clamped to what a portion can
	// physically be, so a dish stays a dish rather than becoming arithmetic.
	target := float64(d.Price) * (0.25 + rng.Float64()*0.08)
	qty := (target - float64(cost)) / float64(main.Price)
	// ⚠️ **The floor is a portion a kitchen would recognise, not a third of
	// one.** It was 0.06 kg, which on salmon is 9 900 so'm — half of a 21 000
	// so'm maki before anything else is on the card. Thirty grams of fish in a
	// small roll is what a small roll has.
	lo, hi := 0.03, 0.32
	if main.Unit == "dona" {
		// ⚠️ **Two, not three.** Three buns in one burger is the card reading
		// as generated, which is the only thing it must not do.
		lo, hi = 1, 2
	}
	if qty < lo {
		qty = lo
	}
	if qty > hi {
		qty = hi
	}
	if main.Unit == "dona" {
		qty = float64(int(qty + 0.5))
	} else {
		qty = float64(int(qty*1000)) / 1000
	}
	lines = append([]models.RecipeLine{{IngredientID: main.ID, Qty: qty}}, lines...)
	cost += int(qty * float64(main.Price))
	return lines, cost
}

// shelfFor finds the shelf a dish's name points at, or nil.
//
// ⚠️ **Lowercased and substring-matched, in three languages.** An imported menu
// carries whichever language the owner published — and the same dish arrives as
// "Losos", "Лосось" or "Salmon" depending on which site it came from. Matching
// one spelling files two thirds of a card under the fallback, which is the
// state this function exists to get out of.
func shelfFor(name string) *shelf {
	low := strings.ToLower(name)
	for i := range kitchen {
		if !anyWord(low, kitchen[i].words) {
			continue
		}
		// ⚠️ The second word, when the rule has one. Without it "Avokado maki"
		// and "Lososli maki" are the same dish to this function, and both come
		// out of the salmon.
		if len(kitchen[i].and) > 0 && !anyWord(low, kitchen[i].and) {
			continue
		}
		return &kitchen[i]
	}
	return nil
}

func anyWord(low string, words []string) bool {
	for _, w := range words {
		if strings.Contains(low, w) {
			return true
		}
	}
	return false
}

// noCard reports a dish that should not carry a recipe at all. See `uncarded`.
func noCard(name string) bool {
	return anyWord(strings.ToLower(name), uncarded)
}

// ----------------------------------------------------------------- the orders

// orderSeed is everything that varies between one sale and the next.
type orderSeed struct {
	at     time.Time
	kind   string // delivery | pickup | dinein
	status models.OrderStatus
}

func (w *world) buildOrder(s orderSeed) models.Order {
	g := w.guests[rng.Intn(len(w.guests))]
	lines := 1 + rng.Intn(4)
	var items []models.OrderItem
	// ⚠️ **The same dish twice is one line with a quantity, not two lines.**
	// Every till in the world merges them, and a check that shows "1 Chuchvara"
	// above "1 Chuchvara" is the one detail on the screen that says the picture
	// was made up — a cashier reading it would assume the screen had double-fed.
	at := map[primitive.ObjectID]int{}
	subtotal := 0
	for i := 0; i < lines; i++ {
		d := w.pickDish()
		qty := 1
		if rng.Intn(4) == 0 {
			qty = 2
		}
		subtotal += d.Price * qty
		if k, ok := at[d.ID]; ok {
			items[k].Qty += qty
			continue
		}
		item := models.OrderItem{
			MenuItemID: d.ID, Name: d.Name, Price: d.Price, Qty: qty,
			Options: []models.OrderItemOption{},
		}
		if rng.Intn(9) == 0 {
			item.Comment = []string{"Achchiq bo'lmasin", "Piyozsiz", "Alohida qadoqlang"}[rng.Intn(3)]
		}
		at[d.ID] = len(items)
		items = append(items, item)
	}

	o := models.Order{
		ID: oid(), BrandID: w.brand, BranchID: w.branch.ID,
		Number:   demoNumber(),
		Status:   s.status,
		Type:     s.kind,
		Customer: models.OrderCustomer{Name: g.FirstName + " " + g.LastName, Phone: g.Phone},
		UserID:   g.ID,
		Items:    items, Subtotal: subtotal,
		Channel:   []string{"web", "web", "telegram", "telegram", "operator"}[rng.Intn(5)],
		CreatedAt: s.at, UpdatedAt: s.at,
	}
	if o.Channel == "operator" {
		o.TakenBy = w.staff[0].Name
	}

	switch s.kind {
	case "delivery":
		addr := g.Addresses[0]
		o.Address = models.OrderAddress{
			Text: addr.Text, Lat: addr.Lat, Lng: addr.Lng,
			Comment: addressNotes[rng.Intn(len(addressNotes))],
		}
		o.DeliveryFee = []int{0, 10_000, 12_000, 15_000}[rng.Intn(4)]
		o.DistanceKm = float64(int((1+rng.Float64()*7)*10)) / 10
		o.DeliveryZone = "Markaz"
	case "dinein":
		if tables := w.branch.Booking.Tables; len(tables) > 0 {
			t := tables[rng.Intn(len(tables))]
			o.TableID, o.TableNumber = t.ID, t.Number
		}
	}

	// A tenth of the orders carry a discount, and it is named — one lump sum
	// off a bill is a number the guest cannot argue with.
	if rng.Intn(10) == 0 {
		off := subtotal / 10
		o.Discounts = []models.OrderDiscount{{Name: "Doimiy mijoz", Amount: off}}
		o.DiscountTotal = off
	}
	o.Total = subtotal + o.DeliveryFee - o.DiscountTotal

	// What this sale took off the shelf. ⚠️ Cancelled orders are not counted:
	// nothing was cooked, and the stock report agrees — sizing the deliveries
	// from them would leave the shelf permanently over-full.
	if s.status != models.StatusCancelled {
		for _, it := range items {
			for _, line := range w.recipes[it.MenuItemID] {
				w.used[line.IngredientID] += line.Qty * float64(it.Qty)
			}
		}
	}

	o.PaymentMethod = []string{"cash", "cash", "card", "payme", "click"}[rng.Intn(5)]
	if o.PaymentMethod == "cash" {
		o.PaymentStatus = "unpaid"
	} else {
		o.PaymentStatus = "paid"
		paid := s.at.Add(time.Duration(rng.Intn(120)) * time.Second)
		o.PaidAt = &paid
	}

	queued := s.at
	o.QueuedAt = &queued
	o.StatusHistory = w.trail(s)
	if s.status == models.StatusDelivered {
		ready := s.at.Add(time.Duration(12+rng.Intn(20)) * time.Minute)
		o.ReadyAt = &ready
		o.UpdatedAt = ready.Add(time.Duration(10+rng.Intn(25)) * time.Minute)
	}
	if s.status == models.StatusCancelled {
		o.CancelReason = []string{
			"Mijoz bekor qildi", "Manzil yetkazish zonasidan tashqarida",
			"Mahsulot tugadi", "Mijoz telefonga javob bermadi",
		}[rng.Intn(4)]
	}
	if (s.status == models.StatusOnTheWay || s.status == models.StatusDelivered) && s.kind == "delivery" {
		c := w.couriers[rng.Intn(len(w.couriers))]
		o.CourierID, o.CourierName = c.ID, c.Name
	}
	return o
}

// trail is the status history, with the timestamps a real one would carry: the
// panel draws a timeline from it, and a history with one entry draws a line
// with one dot on it.
func (w *world) trail(s orderSeed) []models.StatusEvent {
	steps := []models.OrderStatus{models.StatusPending}
	switch s.status {
	case models.StatusCancelled:
		steps = append(steps, models.StatusCancelled)
	case models.StatusDelivered:
		steps = append(steps, models.StatusConfirmed, models.StatusPreparing)
		if s.kind == "delivery" {
			steps = append(steps, models.StatusOnTheWay)
		}
		steps = append(steps, models.StatusDelivered)
	case models.StatusOnTheWay:
		steps = append(steps, models.StatusConfirmed, models.StatusPreparing, models.StatusOnTheWay)
	case models.StatusPreparing:
		steps = append(steps, models.StatusConfirmed, models.StatusPreparing)
	case models.StatusConfirmed:
		steps = append(steps, models.StatusConfirmed)
	}
	out := make([]models.StatusEvent, 0, len(steps))
	at := s.at
	for _, st := range steps {
		out = append(out, models.StatusEvent{Status: st, At: at})
		at = at.Add(time.Duration(3+rng.Intn(18)) * time.Minute)
	}
	return out
}

// history writes the weeks behind us: every order closed, a few cancelled.
//
// ⚠️ **The day has two peaks and the week has a shape.** A flat generator
// produces a dashboard whose chart is a straight line, and a straight line is
// what a chart looks like when it is broken. Friday and Saturday carry the
// week; lunch and dinner carry the day.
func (w *world) history(ctx context.Context) {
	var orders []models.Order
	// ⚠️ One document per visitor per day, the shape `handlers/visits.go`
	// upserts — not a daily total. The dashboard counts rows for "visitors"
	// and sums `views` for page views, so a single rolled-up row per day would
	// draw a site with one visitor on it.
	var visits []bson.M
	for d := w.days; d >= 1; d-- {
		day := w.now.AddDate(0, 0, -d)
		count := 26 + rng.Intn(14)
		switch day.Weekday() {
		case time.Friday, time.Saturday:
			count += 18
		case time.Monday:
			count -= 8
		}
		// A restaurant grows. The oldest weeks are quieter than this one, which
		// is what makes the "vs last month" figure on the dashboard mean
		// something.
		count = count * (100 - d) / 100
		for i := 0; i < count; i++ {
			at := mealtime(day)
			status := models.StatusDelivered
			if rng.Intn(16) == 0 {
				status = models.StatusCancelled
			}
			kind := []string{"delivery", "delivery", "delivery", "pickup", "dinein", "dinein"}[rng.Intn(6)]
			orders = append(orders, w.buildOrder(orderSeed{at: at, kind: kind, status: status}))
		}
		people := count*3 + rng.Intn(40)
		for v := 0; v < people; v++ {
			seen := mealtime(day)
			visits = append(visits, bson.M{
				"date": day.Format("2006-01-02"),
				"vid":  fmt.Sprintf("demo-%s-%d", day.Format("0102"), v),
				// Most people look at one or two pages; a few read the menu
				// end to end. A flat count per visitor makes the "pages per
				// visit" figure a constant.
				"views":     1 + rng.Intn(4),
				"firstAt":   seen,
				"lastAt":    seen.Add(time.Duration(rng.Intn(9)) * time.Minute),
				"firstPath": []string{"/", "/", "/menu", "/menu", "/bron"}[rng.Intn(5)],
			})
		}
	}
	insertMany(ctx, w.store.Orders, orders, "orders (history)")
	insertMany(ctx, w.store.Visits, visits, "visits")
}

// mealtime puts an order inside a service rather than anywhere in the day.
func mealtime(day time.Time) time.Time {
	var hour int
	switch {
	case rng.Intn(10) < 4:
		hour = 12 + rng.Intn(2) // lunch
	case rng.Intn(10) < 8:
		hour = 19 + rng.Intn(2) // dinner
	default:
		hour = 10 + rng.Intn(12)
	}
	return time.Date(day.Year(), day.Month(), day.Day(), hour, rng.Intn(60), rng.Intn(60), 0, time.Local)
}

// ------------------------------------------------------------------- tonight

// tonight is the part every live screen reads: what is open right now.
//
// The pass, the floor map and the orders list all filter on "now", so this is
// the only section whose timestamps have to be minutes old rather than days.
// ⚠️ Which also means it goes stale: a screenshot session that starts an hour
// after this ran gets a pass full of tickets that have been waiting 70 minutes.
// Rerun it before shooting.
func (w *world) tonight(ctx context.Context) {
	var orders []models.Order
	minsAgo := func(n int) time.Time { return w.now.Add(-time.Duration(n) * time.Minute) }

	// Today's completed trade, so the dashboard's "today" is not zero.
	//
	// ⚠️ **Spread across the hours that have actually happened**, which is not
	// the same as "since opening". Run at nine in the morning, a generator that
	// scatters orders between opening and closing puts every one of them in the
	// future and drops the lot — the tool reported fifteen orders for a day it
	// had written none of, and the dashboard read zero. Before the room could
	// plausibly be open, the window is simply the last few hours.
	opened := time.Date(w.now.Year(), w.now.Month(), w.now.Day(), 9, 0, 0, 0, time.Local)
	if w.now.Sub(opened) < 3*time.Hour {
		opened = w.now.Add(-3 * time.Hour)
	}
	span := int(w.now.Sub(opened).Minutes()) - 10
	for i := 0; i < 20+rng.Intn(10) && span > 0; i++ {
		at := opened.Add(time.Duration(rng.Intn(span)) * time.Minute)
		kind := []string{"delivery", "delivery", "pickup", "dinein"}[rng.Intn(4)]
		orders = append(orders, w.buildOrder(orderSeed{at: at, kind: kind, status: models.StatusDelivered}))
	}

	// The live queue, oldest first: two waiting to be accepted, six on the
	// pass, three out with a courier.
	//
	// ⚠️ **Six, not four, because the pass is a screen and not a list.** The
	// kitchen display lays its tickets out three across; four of them plus the
	// two dine-in checks filled two rows and left the bottom third of a
	// monitor blank, which reads as a quiet kitchen — the opposite of what a
	// picture of a service is for.
	for i := 0; i < 2; i++ {
		o := w.buildOrder(orderSeed{at: minsAgo(1 + i*2), kind: "delivery", status: models.StatusPending})
		orders = append(orders, o)
	}
	passKinds := []string{"delivery", "dinein", "pickup", "delivery", "delivery", "dinein"}
	for i := 0; i < len(passKinds); i++ {
		kind := passKinds[i]
		st := models.StatusPreparing
		// The two newest have been accepted but not started, so the pass shows
		// both states and its one button has something to do.
		if i >= len(passKinds)-2 {
			st = models.StatusConfirmed
		}
		o := w.buildOrder(orderSeed{at: minsAgo(4 + i*4), kind: kind, status: st})
		// ⚠️ The pass filters on `queuedAt <= now` and `readyAt` empty — a
		// ticket without both is a ticket nobody in the kitchen ever sees.
		o.ReadyAt = nil
		orders = append(orders, o)
	}
	// ⚠️ **All three go to the courier who is marked busy.** Spread at random
	// over three couriers, each one's app shows a single order — and a delivery
	// app whose screen holds one job is a picture of a quiet evening. One rider
	// out with a round of three is both busier and more honest: it is why the
	// other two are still marked free.
	for i := 0; i < 3; i++ {
		o := w.buildOrder(orderSeed{at: minsAgo(22 + i*7), kind: "delivery", status: models.StatusOnTheWay})
		ready := o.CreatedAt.Add(14 * time.Minute)
		o.ReadyAt = &ready
		o.CourierID, o.CourierName = w.couriers[0].ID, w.couriers[0].Name
		orders = append(orders, o)
	}

	// Open checks on the floor: five tables eating, one that has asked for the
	// bill. ⚠️ A dine-in order without a `check` is a website order that names
	// a table — the floor screen filters on the check, so these are the only
	// orders that light a table up.
	tables := w.branch.Booking.Tables
	// ⚠️ Not every other table: `i*2` fills a perfect checkerboard, and a room
	// where every occupied table has an empty one on each side is a diagram
	// rather than a dining room. A scattered set reads as a Friday.
	seated := []int{0, 2, 3, 7, 9, 12, 14, 16}
	for i := 0; i < len(seated) && i < len(tables); i++ {
		t := tables[seated[i]%len(tables)]
		// ⚠️ Eight to forty-three minutes, and the step has to shrink as tables
		// are added. The floor screen turns a table red once it has been
		// sitting too long; at a seven-minute step the eighth check is
		// fifty-seven minutes old, so growing the room from twelve covers to
		// eighteen quietly put two reds back on a plan whose whole job is to
		// show that nothing is going wrong.
		opened := minsAgo(8 + i*5)
		o := w.buildOrder(orderSeed{at: opened, kind: "dinein", status: models.StatusPreparing})
		o.TableID, o.TableNumber = t.ID, t.Number
		o.PaymentStatus, o.PaidAt = "unpaid", nil
		o.PaymentMethod = ""
		waiter := w.waiters[i%len(w.waiters)]
		check := models.OrderCheck{
			OpenedAt:   opened,
			OpenedByID: waiter.ID, OpenedBy: waiter.Name,
			ServerID: waiter.ID, ServerName: waiter.Name,
			Guests: 2 + rng.Intn(maxInt(1, t.Seats-1)),
		}
		if i == len(seated)-1 {
			asked := minsAgo(4)
			check.PrecheckAt = &asked
		}
		// ⚠️ **A table that has been eating an hour is not still on the pass.**
		// Left without a ready time, every open check turns into a kitchen
		// ticket, and the six of them arrive at the top of the screen as
		// seventy-, sixty- and fifty-minute reds — a pass that reads as a
		// kitchen in collapse. Only the two newest tables are still cooking;
		// the rest have been served and stay open because nobody has paid yet,
		// which is what an open check actually means.
		if i >= 3 {
			served := opened.Add(time.Duration(9+rng.Intn(6)) * time.Minute)
			o.ReadyAt = &served
		}
		o.Check = &check
		// The lines the kitchen was told about. ⚠️ An unfired line is hidden
		// from the pass on purpose (`kitchenItems`), so a check whose lines are
		// all unfired is an empty ticket — one line on the newest table is left
		// unfired deliberately, because that state is real and worth seeing.
		for li := range o.Items {
			o.Items[li].LineID = fmt.Sprintf("l%d", li+1)
			if i == 0 && li == len(o.Items)-1 {
				continue
			}
			fired := opened.Add(time.Duration(1+li) * time.Minute)
			o.Items[li].FiredAt = &fired
		}
		orders = append(orders, o)
	}
	insertMany(ctx, w.store.Orders, orders, "orders (today)")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// --------------------------------------------------------------- the register

func (w *world) tills(ctx context.Context) {
	var shifts []models.CashShift
	var entries []models.CashEntry

	// Closed days, each with a small variance — a till that balances to the
	// so'm every single day is a till nobody counted.
	for d := 7; d >= 1; d-- {
		day := w.now.AddDate(0, 0, -d)
		open := time.Date(day.Year(), day.Month(), day.Day(), 9, 0, 0, 0, time.Local)
		closed := open.Add(14 * time.Hour)
		expected := 3_000_000 + rng.Intn(4_000_000)
		variance := []int{0, 0, -12_000, 5_000, -3_000}[rng.Intn(5)]
		id := oid()
		note := ""
		if variance < 0 {
			note = "Mayda pul qaytimida farq"
		}
		shifts = append(shifts, models.CashShift{
			ID: id, BranchID: w.branch.ID,
			OpenedAt: open, OpenedBy: w.cashier.Name, OpenedByID: w.cashier.ID,
			OpeningFloat: 300_000,
			ClosedAt:     &closed, ClosedBy: w.cashier.Name,
			Expected: expected, CounterCash: expected, Counted: expected + variance,
			Variance: variance, VarianceNote: note,
			CreatedAt: open, UpdatedAt: closed,
		})
		entries = append(entries, models.CashEntry{
			ID: oid(), ShiftID: id, BranchID: w.branch.ID,
			Kind: models.CashOut, Amount: 150_000 + rng.Intn(200_000),
			Category: "xarajat",
			Note:     []string{"Bozorga ko'kat", "Kuryer avansi", "Xo'jalik mollari"}[rng.Intn(3)],
			By:       w.cashier.Name, ByID: w.cashier.ID, At: open.Add(5 * time.Hour),
		})
	}

	// And the one that is open now — the state the till screen is screenshotted
	// in, and the only one on which "X hisobot" means anything.
	open := time.Date(w.now.Year(), w.now.Month(), w.now.Day(), 9, 0, 0, 0, time.Local)
	if open.After(w.now) {
		open = w.now.Add(-3 * time.Hour)
	}
	id := oid()
	shifts = append(shifts, models.CashShift{
		ID: id, BranchID: w.branch.ID,
		OpenedAt: open, OpenedBy: w.cashier.Name, OpenedByID: w.cashier.ID,
		OpeningFloat: 300_000,
		CreatedAt:    open, UpdatedAt: w.now,
	})
	entries = append(entries, models.CashEntry{
		ID: oid(), ShiftID: id, BranchID: w.branch.ID,
		Kind: models.CashIn, Amount: 500_000, Category: "kirim",
		Note: "Ertalabki qoldiq", By: w.cashier.Name, ByID: w.cashier.ID,
		At: open.Add(2 * time.Minute),
	})
	insertMany(ctx, w.store.CashShifts, shifts, "cash shifts")
	insertMany(ctx, w.store.CashEntries, entries, "cash entries")
}

// ------------------------------------------------------------------ the diary

func (w *world) diary(ctx context.Context) {
	var books []models.Reservation
	tables := w.branch.Booking.Tables
	statuses := []models.ReservationStatus{
		models.ReservationConfirmed, models.ReservationPending,
		models.ReservationConfirmed, models.ReservationSeated, models.ReservationDone,
	}
	for i := 0; i < 9 && len(tables) > 0; i++ {
		t := tables[rng.Intn(len(tables))]
		day := w.now.AddDate(0, 0, i/3)
		at := time.Date(day.Year(), day.Month(), day.Day(), 18+rng.Intn(4), []int{0, 30}[rng.Intn(2)], 0, 0, time.Local)
		st := statuses[i%len(statuses)]
		g := w.guests[rng.Intn(len(w.guests))]
		books = append(books, models.Reservation{
			ID: oid(), BranchID: w.branch.ID, Number: demoNumber(),
			TableID: t.ID, TableNumber: t.Number,
			UserID:   g.ID,
			Customer: models.OrderCustomer{Name: g.FirstName + " " + g.LastName, Phone: g.Phone},
			Guests:   2 + rng.Intn(6),
			At:       at, EndsAt: at.Add(90 * time.Minute),
			Status:  st,
			Comment: []string{"", "Tug'ilgan kun", "", "Deraza yonida", ""}[rng.Intn(5)],
			StatusHistory: []models.ReservationEvent{
				{Status: models.ReservationPending, At: at.Add(-26 * time.Hour)},
				{Status: st, At: at.Add(-24 * time.Hour)},
			},
			CreatedAt: at.Add(-26 * time.Hour), UpdatedAt: at.Add(-24 * time.Hour),
		})
	}
	insertMany(ctx, w.store.Reservations, books, "reservations")

	var notes []models.Feedback
	texts := []string{
		"Osh juda mazali, yetkazish tez bo'ldi. Rahmat!",
		"Hammasi zo'r, faqat bir oz kech keldi.",
		"Ofitsiant juda xushmuomala, yana kelamiz.",
		"Taom issiq va toza qadoqlangan edi.",
		"Narxi sifatiga arziydi.",
		"Buyurtmada bitta salat yetishmadi, lekin tez hal qilishdi.",
	}
	for i := 0; i < 14; i++ {
		g := w.guests[rng.Intn(len(w.guests))]
		at := w.now.AddDate(0, 0, -rng.Intn(w.days))
		stars := []int{5, 5, 5, 4, 4, 3}[rng.Intn(6)]
		f := models.Feedback{
			ID: oid(), UserID: g.ID, BranchID: w.branch.ID,
			Customer:  models.OrderCustomer{Name: g.FirstName + " " + g.LastName, Phone: g.Phone},
			Rating:    stars,
			IsPublic:  stars >= 4,
			CreatedAt: at,
		}
		if i%2 == 0 {
			f.Comment = texts[rng.Intn(len(texts))]
		}
		notes = append(notes, f)
	}
	insertMany(ctx, w.store.Feedback, notes, "feedback")
}
