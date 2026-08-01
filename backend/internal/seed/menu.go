package seed

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"restaurant-backend/internal/config"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Demo photos shipped with the template. They are written into UPLOAD_DIR/seed
// on first boot so they are served by the normal /uploads/* route (and survive
// in the Docker uploads volume like any other uploaded image).
//
//go:embed assets/*.jpg
var seedImages embed.FS

const seedImageDir = "seed" // relative to UPLOAD_DIR

type seedItem struct {
	Name          string // uz (base)
	NameRu        string
	NameEn        string
	Description   string
	DescriptionRu string
	DescriptionEn string
	Price         int
	OldPrice      int // 0 = none
	Image         string
	Popular       bool
	Tags          []string
	Options       []models.MenuOption
}

type seedCategory struct {
	Name   string // uz (base)
	NameRu string
	NameEn string
	Slug   string
	Image  string
	Items  []seedItem
}

// demoMenu is the sample menu created on an empty database: national dishes,
// fast food, pizza, salads, desserts and drinks — every item with a photo.
var demoMenu = []seedCategory{
	{
		Name: "Milliy taomlar", NameRu: "Национальные блюда", NameEn: "Uzbek classics",
		Slug: "milliy-taomlar", Image: "osh",
		Items: []seedItem{
			{Name: "Osh (palov)", NameRu: "Плов", NameEn: "Plov (pilaf)",
				Description:   "Devzira guruch, mol go'shti, sabzi va zira bilan an'anaviy tayyorlangan osh.",
				DescriptionRu: "Рис девзира, говядина, морковь и зира — приготовлено по традиционному рецепту.",
				DescriptionEn: "Devzira rice, beef, carrot and cumin — cooked the traditional way.",
				Price:         45000, Image: "osh", Popular: true, Tags: []string{"an'anaviy"}},
			{Name: "Lag'mon", NameRu: "Лагман", NameEn: "Lagman",
				Description:   "Qo'lda cho'zilgan xamir, mol go'shti va yangi sabzavotlar bilan.",
				DescriptionRu: "Тянутая вручную лапша с говядиной и свежими овощами.",
				DescriptionEn: "Hand-pulled noodles with beef and fresh vegetables.",
				Price:         42000, Image: "lagmon", Popular: true},
			{Name: "Manti", NameRu: "Манты", NameEn: "Manti",
				Description:   "Bug'da pishirilgan qo'y go'shtli manti (5 dona), qatiq bilan.",
				DescriptionRu: "Приготовленные на пару манты с бараниной (5 шт.), с катыком.",
				DescriptionEn: "Steamed lamb dumplings (5 pcs) served with yoghurt.",
				Price:         40000, Image: "manti", Popular: true},
			{Name: "Chuchvara", NameRu: "Чучвара", NameEn: "Chuchvara",
				Description:   "Mayda chuchvara — sho'rvada yoki qovurilgan holda.",
				DescriptionRu: "Мелкие пельмени — в бульоне или обжаренные.",
				DescriptionEn: "Small dumplings — in broth or pan-fried.",
				Price:         38000, Image: "chuchvara"},
			{Name: "Somsa", NameRu: "Самса", NameEn: "Samsa",
				Description:   "Tandirda pishirilgan go'shtli somsa (1 dona).",
				DescriptionRu: "Самса с мясом из тандыра (1 шт.).",
				DescriptionEn: "Meat-filled pastry baked in a tandoor (1 pc).",
				Price:         15000, Image: "somsa", Popular: true},
			{Name: "Sho'rva", NameRu: "Шурпа", NameEn: "Shurpa",
				Description:   "Mol go'shti va sabzavotlardan tayyorlangan issiq sho'rva.",
				DescriptionRu: "Горячий суп из говядины и овощей.",
				DescriptionEn: "Hot beef and vegetable soup.",
				Price:         35000, Image: "shurva"},
			{Name: "Tandir non", NameRu: "Тандырная лепёшка", NameEn: "Tandoor bread",
				Description:   "Tandirdan yangi uzilgan issiq non.",
				DescriptionRu: "Горячая лепёшка прямо из тандыра.",
				DescriptionEn: "Hot flatbread fresh from the tandoor.",
				Price:         6000, Image: "non"},
		},
	},
	{
		Name: "Grill", NameRu: "Гриль", NameEn: "Grill",
		Slug: "grill", Image: "shashlik",
		Items: []seedItem{
			{Name: "Mol go'shtli shashlik", NameRu: "Шашлык из говядины", NameEn: "Beef skewers",
				Description:   "Cho'g'da pishirilgan mol go'shti shashlik, piyoz va non bilan.",
				DescriptionRu: "Шашлык из говядины на углях, с луком и лепёшкой.",
				DescriptionEn: "Charcoal-grilled beef skewers with onion and bread.",
				Price:         32000, Image: "shashlik", Popular: true},
			{Name: "Ribeye steyk", NameRu: "Стейк рибай", NameEn: "Ribeye steak",
				Description:   "Grillda pishirilgan mramor go'shti, sabzavot garniri bilan.",
				DescriptionRu: "Мраморная говядина на гриле с овощным гарниром.",
				DescriptionEn: "Grilled marbled beef with a vegetable side.",
				Price:         145000, OldPrice: 165000, Image: "steyk", Tags: []string{"premium"}},
			{Name: "Dimlama", NameRu: "Димлама", NameEn: "Dimlama",
				Description:   "Qozonda dimlangan go'sht va sabzavotlar.",
				DescriptionRu: "Мясо и овощи, томлённые в казане.",
				DescriptionEn: "Meat and vegetables slow-braised in a kazan.",
				Price:         58000, Image: "dimlama"},
		},
	},
	{
		Name: "Burger va fast food", NameRu: "Бургеры и фастфуд", NameEn: "Burgers & fast food",
		Slug: "fast-food", Image: "cheeseburger",
		Items: []seedItem{
			{Name: "Chizburger", NameRu: "Чизбургер", NameEn: "Cheeseburger",
				Description:   "Mol go'shti kotleti, cheddar pishloq, pomidor va maxsus sous.",
				DescriptionRu: "Котлета из говядины, чеддер, помидор и фирменный соус.",
				DescriptionEn: "Beef patty, cheddar, tomato and our house sauce.",
				Price:         42000, Image: "cheeseburger", Popular: true},
			{Name: "Tovuqli burger", NameRu: "Куриный бургер", NameEn: "Chicken burger",
				Description:   "Xrustashiy tovuq filesi, salat bargi va sarimsoqli sous.",
				DescriptionRu: "Хрустящее куриное филе, салат и чесночный соус.",
				DescriptionEn: "Crispy chicken fillet, lettuce and garlic sauce.",
				Price:         38000, Image: "chickenburger"},
			{Name: "Lavash", NameRu: "Лаваш", NameEn: "Lavash wrap",
				Description:   "Tovuq yoki mol go'shti bilan katta lavash.",
				DescriptionRu: "Большой лаваш с курицей или говядиной.",
				DescriptionEn: "Large wrap with chicken or beef.",
				Price:         33000, Image: "lavash", Popular: true},
			{Name: "Hot-dog", NameRu: "Хот-дог", NameEn: "Hot dog",
				Description:   "Issiq bulochka, sosiska, ketchup va gorchitsa.",
				DescriptionRu: "Тёплая булочка, сосиска, кетчуп и горчица.",
				DescriptionEn: "Warm bun, sausage, ketchup and mustard.",
				Price:         22000, Image: "hotdog"},
			{Name: "Fri kartoshka", NameRu: "Картофель фри", NameEn: "French fries",
				Description:   "Xrustashiy fri kartoshka, sous bilan.",
				DescriptionRu: "Хрустящий картофель фри с соусом.",
				DescriptionEn: "Crispy fries served with a dip.",
				Price:         18000, Image: "fri"},
			{Name: "Nagets", NameRu: "Наггетсы", NameEn: "Nuggets",
				Description:   "Tovuq nagetslari (8 dona), 2 xil sous bilan.",
				DescriptionRu: "Куриные наггетсы (8 шт.) с двумя соусами.",
				DescriptionEn: "Chicken nuggets (8 pcs) with two dips.",
				Price:         26000, Image: "nuggets"},
		},
	},
	{
		Name: "Pitsa va pasta", NameRu: "Пицца и паста", NameEn: "Pizza & pasta",
		Slug: "pitsa", Image: "pizza-margarita",
		Items: []seedItem{
			{Name: "Margarita", NameRu: "Маргарита", NameEn: "Margherita",
				Description:   "Pomidor sousi, motsarella va rayhon.",
				DescriptionRu: "Томатный соус, моцарелла и базилик.",
				DescriptionEn: "Tomato sauce, mozzarella and basil.",
				Price:         65000, Image: "pizza-margarita", Popular: true},
			{Name: "Pepperoni", NameRu: "Пепперони", NameEn: "Pepperoni",
				Description:   "Achchiqroq pepperoni kolbasa va ikki xil pishloq.",
				DescriptionRu: "Острая пепперони и два вида сыра.",
				DescriptionEn: "Spicy pepperoni and two kinds of cheese.",
				Price:         78000, Image: "pizza-pepperoni", Popular: true},
			{Name: "To'rt pishloq", NameRu: "Четыре сыра", NameEn: "Four cheese",
				Description:   "Motsarella, cheddar, parmezan va dor blyu.",
				DescriptionRu: "Моцарелла, чеддер, пармезан и дорблю.",
				DescriptionEn: "Mozzarella, cheddar, parmesan and blue cheese.",
				Price:         82000, Image: "pizza-cheese"},
			{Name: "Go'shtli pitsa", NameRu: "Мясная пицца", NameEn: "Meat pizza",
				Description:   "Mol go'shti, tovuq, kolbasa va qo'ziqorin.",
				DescriptionRu: "Говядина, курица, колбаски и грибы.",
				DescriptionEn: "Beef, chicken, sausage and mushrooms.",
				Price:         89000, OldPrice: 99000, Image: "pizza-meat"},
			{Name: "Bolonez pasta", NameRu: "Паста болоньезе", NameEn: "Pasta bolognese",
				Description:   "Penne pasta, go'shtli pomidor sousi va parmezan.",
				DescriptionRu: "Паста пенне, мясной томатный соус и пармезан.",
				DescriptionEn: "Penne pasta, meaty tomato sauce and parmesan.",
				Price:         52000, Image: "pasta"},
		},
	},
	{
		Name: "Salatlar", NameRu: "Салаты", NameEn: "Salads",
		Slug: "salatlar", Image: "salat-sezar",
		Items: []seedItem{
			{Name: "Sezar salat", NameRu: "Салат Цезарь", NameEn: "Caesar salad",
				Description:   "Tovuq filesi, romen salat, krutonlar va sezar sousi.",
				DescriptionRu: "Куриное филе, романо, гренки и соус цезарь.",
				DescriptionEn: "Chicken fillet, romaine, croutons and Caesar dressing.",
				Price:         42000, Image: "salat-sezar", Popular: true},
			{Name: "Grek salat", NameRu: "Греческий салат", NameEn: "Greek salad",
				Description:   "Yangi sabzavotlar, feta pishlog'i va zaytun moyi.",
				DescriptionRu: "Свежие овощи, сыр фета и оливковое масло.",
				DescriptionEn: "Fresh vegetables, feta cheese and olive oil.",
				Price:         38000, Image: "salat-grek"},
			{Name: "Vitamin salat", NameRu: "Витаминный салат", NameEn: "Garden salad",
				Description:   "Mavsumiy yangi sabzavotlardan yengil salat.",
				DescriptionRu: "Лёгкий салат из свежих сезонных овощей.",
				DescriptionEn: "A light salad of fresh seasonal vegetables.",
				Price:         28000, Image: "salat-olivye"},
		},
	},
	{
		Name: "Shirinliklar", NameRu: "Десерты", NameEn: "Desserts",
		Slug: "shirinliklar", Image: "chizkeyk",
		Items: []seedItem{
			{Name: "Chizkeyk", NameRu: "Чизкейк", NameEn: "Cheesecake",
				Description:   "New York uslubidagi chizkeyk, rezavor sous bilan.",
				DescriptionRu: "Чизкейк «Нью-Йорк» с ягодным соусом.",
				DescriptionEn: "New York style cheesecake with berry sauce.",
				Price:         34000, Image: "chizkeyk", Popular: true},
			{Name: "Tiramisu", NameRu: "Тирамису", NameEn: "Tiramisu",
				Description:   "Mascarpone kremi va kofega botirilgan savoyardi.",
				DescriptionRu: "Крем маскарпоне и савоярди, пропитанные кофе.",
				DescriptionEn: "Mascarpone cream and coffee-soaked ladyfingers.",
				Price:         36000, Image: "tiramisu", Popular: true},
			{Name: "Napoleon", NameRu: "Наполеон", NameEn: "Napoleon cake",
				Description:   "Ko'p qatlamli qaymoqli tort bo'lagi.",
				DescriptionRu: "Кусочек многослойного торта с кремом.",
				DescriptionEn: "A slice of layered cream cake.",
				Price:         30000, Image: "napoleon"},
			{Name: "Qatlamli tort", NameRu: "Слоёный торт", NameEn: "Layer cake",
				Description:   "Rangli qatlamlar va yumshoq krem.",
				DescriptionRu: "Цветные коржи и нежный крем.",
				DescriptionEn: "Colourful layers with soft cream.",
				Price:         32000, Image: "medovik"},
			{Name: "Shokoladli tort", NameRu: "Шоколадный торт", NameEn: "Chocolate cake",
				Description:   "Quyuq shokoladli tort, ganache bilan.",
				DescriptionRu: "Насыщенный шоколадный торт с ганашем.",
				DescriptionEn: "Rich chocolate cake with ganache.",
				Price:         35000, Image: "shokolad-tort"},
			{Name: "Brauni", NameRu: "Брауни", NameEn: "Brownie",
				Description:   "Issiq shokoladli brauni, yong'oq bilan.",
				DescriptionRu: "Тёплый шоколадный брауни с орехами.",
				DescriptionEn: "Warm chocolate brownie with nuts.",
				Price:         28000, Image: "brauni"},
			{Name: "Lava cake", NameRu: "Лава-кейк", NameEn: "Lava cake",
				Description:   "Ichi oquvchan shokoladli keks, muzqaymoq bilan.",
				DescriptionRu: "Шоколадный кекс с жидким центром и мороженым.",
				DescriptionEn: "Molten chocolate cake served with ice cream.",
				Price:         38000, Image: "lava-cake", Popular: true},
			{Name: "Muzqaymoq", NameRu: "Мороженое", NameEn: "Ice cream",
				Description:   "Vanil, shokolad yoki qulupnay — tanlash mumkin.",
				DescriptionRu: "Ваниль, шоколад или клубника — на выбор.",
				DescriptionEn: "Vanilla, chocolate or strawberry — your pick.",
				Price:         18000, Image: "muzqaymoq"},
			{Name: "Donut", NameRu: "Пончики", NameEn: "Doughnuts",
				Description:   "Glazurli donut (2 dona).",
				DescriptionRu: "Пончики в глазури (2 шт.).",
				DescriptionEn: "Glazed doughnuts (2 pcs).",
				Price:         20000, Image: "donut"},
			{Name: "Kruassan", NameRu: "Круассан", NameEn: "Croissant",
				Description:   "Yangi pishirilgan sariyog'li kruassan.",
				DescriptionRu: "Свежеиспечённый круассан на сливочном масле.",
				DescriptionEn: "Freshly baked butter croissant.",
				Price:         16000, Image: "kruassan"},
			{Name: "Pancake", NameRu: "Панкейки", NameEn: "Pancakes",
				Description:   "Asal va yangi mevalar bilan pancake.",
				DescriptionRu: "Панкейки с мёдом и свежими фруктами.",
				DescriptionEn: "Pancakes with honey and fresh fruit.",
				Price:         29000, Image: "pancake"},
		},
	},
	{
		Name: "Ichimliklar", NameRu: "Напитки", NameEn: "Drinks",
		Slug: "ichimliklar", Image: "latte",
		Items: []seedItem{
			{Name: "Ko'k choy", NameRu: "Зелёный чай", NameEn: "Green tea",
				Description:   "An'anaviy ko'k choy (choynak).",
				DescriptionRu: "Традиционный зелёный чай (чайник).",
				DescriptionEn: "Traditional green tea (pot).",
				Price:         8000, Image: "kokchoy"},
			{Name: "Qora choy", NameRu: "Чёрный чай", NameEn: "Black tea",
				Description:   "Limon yoki sut bilan qora choy (choynak).",
				DescriptionRu: "Чёрный чай с лимоном или молоком (чайник).",
				DescriptionEn: "Black tea with lemon or milk (pot).",
				Price:         9000, Image: "qorachoy"},
			{Name: "Amerikano", NameRu: "Американо", NameEn: "Americano",
				Description:   "Ikki porsiya espresso va issiq suv.",
				DescriptionRu: "Двойной эспрессо и горячая вода.",
				DescriptionEn: "Double espresso topped with hot water.",
				Price:         18000, Image: "americano"},
			{Name: "Kapuchino", NameRu: "Капучино", NameEn: "Cappuccino",
				Description:   "Espresso va mayin sut ko'pigi.",
				DescriptionRu: "Эспрессо и нежная молочная пена.",
				DescriptionEn: "Espresso with silky milk foam.",
				Price:         24000, Image: "kapuchino", Popular: true},
			{Name: "Muzli latte", NameRu: "Айс-латте", NameEn: "Iced latte",
				Description:   "Sovuq sut, muz va espresso.",
				DescriptionRu: "Холодное молоко, лёд и эспрессо.",
				DescriptionEn: "Cold milk, ice and espresso.",
				Price:         26000, Image: "latte"},
			{Name: "Apelsin fresh", NameRu: "Апельсиновый фреш", NameEn: "Orange juice",
				Description:   "Yangi siqilgan apelsin sharbati, 0.3 l.",
				DescriptionRu: "Свежевыжатый апельсиновый сок, 0,3 л.",
				DescriptionEn: "Freshly squeezed orange juice, 0.3 l.",
				Price:         28000, Image: "fresh", Popular: true},
			{Name: "Limonad", NameRu: "Лимонад", NameEn: "Lemonade",
				Description:   "Uy limonadi — limon va yalpiz, 0.5 l.",
				DescriptionRu: "Домашний лимонад с лимоном и мятой, 0,5 л.",
				DescriptionEn: "House lemonade with lemon and mint, 0.5 l.",
				Price:         22000, Image: "limonad"},
			{Name: "Mokito", NameRu: "Мохито", NameEn: "Mojito",
				Description:   "Alkogolsiz mokito — layma, yalpiz va muz.",
				DescriptionRu: "Безалкогольный мохито — лайм, мята и лёд.",
				DescriptionEn: "Non-alcoholic mojito — lime, mint and ice.",
				Price:         25000, Image: "mokito"},
			{Name: "Milkshake", NameRu: "Молочный коктейль", NameEn: "Milkshake",
				Description:   "Shokoladli milkshake, pechenye bilan.",
				DescriptionRu: "Шоколадный молочный коктейль с печеньем.",
				DescriptionEn: "Chocolate milkshake with cookies.",
				Price:         30000, Image: "milkshake"},
			{Name: "Smuzi", NameRu: "Смузи", NameEn: "Smoothie",
				Description:   "Meva smuzisi — mavsumiy mevalardan.",
				DescriptionRu: "Фруктовое смузи из сезонных фруктов.",
				DescriptionEn: "Fruit smoothie made with seasonal fruit.",
				Price:         27000, Image: "smuzi"},
			{Name: "Ayron", NameRu: "Айран", NameEn: "Ayran",
				Description:   "Sovuq ayron, 0.5 l.",
				DescriptionRu: "Холодный айран, 0,5 л.",
				DescriptionEn: "Chilled ayran, 0.5 l.",
				Price:         12000, Image: "ayron"},
			{Name: "Coca-Cola", NameRu: "Coca-Cola", NameEn: "Coca-Cola",
				Description:   "Gazlangan ichimlik, 0.5 l.",
				DescriptionRu: "Газированный напиток, 0,5 л.",
				DescriptionEn: "Soft drink, 0.5 l.",
				Price:         12000, Image: "kola"},
			{Name: "Suv", NameRu: "Вода", NameEn: "Water",
				Description:   "Gazsiz ichimlik suvi, 0.5 l.",
				DescriptionRu: "Питьевая вода без газа, 0,5 л.",
				DescriptionEn: "Still drinking water, 0.5 l.",
				Price:         6000, Image: "suv"},
		},
	},
}

// ensureMenu writes the demo photos to disk and creates the sample categories
// and menu items — but only when the database has no categories yet, so a real
// restaurant's menu is never touched.
func ensureMenu(ctx context.Context, store *repository.Store, cfg *config.Config) {
	count, err := store.Categories.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Printf("seed: count categories: %v", err)
		return
	}
	if count > 0 {
		return
	}
	if err := WriteMenu(ctx, store, cfg, false); err != nil {
		log.Printf("seed: write demo menu: %v", err)
	}
}

// WriteMenu inserts the demo menu into the given database. With replace=true it
// first deletes the existing categories and items — used by `cmd/seedmenu` to
// bring an older database up to the current sample menu.
func WriteMenu(ctx context.Context, store *repository.Store, cfg *config.Config, replace bool) error {
	if replace {
		if _, err := store.Categories.DeleteMany(ctx, bson.M{}); err != nil {
			return err
		}
		if _, err := store.Menu.DeleteMany(ctx, bson.M{}); err != nil {
			return err
		}
	}
	if err := extractSeedImages(cfg.UploadDir); err != nil {
		log.Printf("seed: extract images: %v", err)
	}

	now := time.Now()
	items := 0
	for ci, cat := range demoMenu {
		res, err := store.Categories.InsertOne(ctx, models.Category{
			Name:      cat.Name,
			NameRu:    cat.NameRu,
			NameEn:    cat.NameEn,
			Slug:      cat.Slug,
			SortOrder: ci,
			IsActive:  true,
			ImageURL:  imagePath(cat.Image),
		})
		if err != nil {
			return fmt.Errorf("insert category %q: %w", cat.Name, err)
		}
		catID, _ := res.InsertedID.(primitive.ObjectID)

		docs := make([]any, 0, len(cat.Items))
		for i, it := range cat.Items {
			var oldPrice *int
			if it.OldPrice > 0 {
				v := it.OldPrice
				oldPrice = &v
			}
			docs = append(docs, models.MenuItem{
				CategoryID:    catID,
				Name:          it.Name,
				NameRu:        it.NameRu,
				NameEn:        it.NameEn,
				Description:   it.Description,
				DescriptionRu: it.DescriptionRu,
				DescriptionEn: it.DescriptionEn,
				Price:         it.Price,
				OldPrice:      oldPrice,
				ImageURL:      imagePath(it.Image),
				IsAvailable:   true,
				IsPopular:     it.Popular,
				SortOrder:     i,
				Options:       it.Options,
				Tags:          it.Tags,
				UpdatedAt:     now,
			})
		}
		if len(docs) == 0 {
			continue
		}
		if _, err := store.Menu.InsertMany(ctx, docs); err != nil {
			return fmt.Errorf("insert items for %q: %w", cat.Name, err)
		}
		items += len(docs)
	}
	log.Printf("seed: created demo menu — %d categories, %d items", len(demoMenu), items)
	return nil
}

func imagePath(name string) string {
	if name == "" {
		return ""
	}
	return "/uploads/" + seedImageDir + "/" + name + ".jpg"
}

// extractSeedImages copies the embedded photos into UPLOAD_DIR/seed, skipping
// files that already exist.
func extractSeedImages(uploadDir string) error {
	dir := filepath.Join(uploadDir, seedImageDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entries, err := fs.ReadDir(seedImages, "assets")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		dst := filepath.Join(dir, e.Name())
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		data, err := seedImages.ReadFile("assets/" + e.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
