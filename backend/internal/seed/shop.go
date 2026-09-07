package seed

// ---- The sample catalogue a shop starts with ----
//
// ⚠️ **A shop cannot be handed a restaurant's menu, and it cannot be handed
// nothing either.** Forty-eight dishes in a pharmacy read as somebody else's
// shop, and the owner's first job becomes deleting them one at a time; an empty
// grid on the day the site is handed over reads as "this does not work" rather
// than as "nothing has been entered yet". So each kind of shop gets a short
// catalogue of things it actually sells.
//
// ⚠️ **Short on purpose — a dozen lines, not fifty.** This is a demonstration
// that the till, the labels and the shelf balances work, not a starter
// catalogue: a real shop's list comes off its own barcodes, and every sample row
// left in it is a row somebody has to find and delete later. The restaurant's
// menu is long because a restaurant's menu is *editable* — a shop's is scanned.
//
// ⚠️ **Every row goes through the same code the panel uses**
// (`repository.SyncProductStock`). A product that sells itself needs a stock row
// and the one-line card the server keeps in step with it; rows written straight
// into Mongo would be sellable and uncountable, which is the one kind of wrong
// number this product never lets through — and it would be wrong from the first
// day, on the demonstration data.
//
// ⚠️ **No photographs.** The seed's images are dishes, and a shop's shelf is not
// photographed by us. A card with no picture is ordinary in a grocery listing;
// a picture of somebody's plov above a box of paracetamol is not.

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
	"restaurant-backend/internal/repository"
)

// shopItem is one product on a shop's shelf.
//
// ⚠️ **`Unit` is the state classifier's code**, the same one the menu form
// writes: 0 piece, 11 kilogram, 10 gram, 41 litre, 22 metre. It decides the word
// on the shelf label and the unit a purchase is written in, and inventing a
// third vocabulary here is how those two stop agreeing.
type shopItem struct {
	Name   string
	NameRu string
	NameEn string
	Price  int
	Unit   int
}

type shopCategory struct {
	Name   string
	NameRu string
	NameEn string
	Slug   string
	Items  []shopItem
}

// shopCatalogues is what each kind of shop starts with.
//
// ⚠️ **Prices are plausible Tashkent ones and they are still made up.** They are
// there so the till, the receipt and the shelf label have something to show; the
// first thing an owner does is correct them, and a price of 0 would have made
// every one of those screens look broken instead.
var shopCatalogues = map[models.BusinessType][]shopCategory{
	models.BizGrocery: {
		{
			Name: "Ichimliklar", NameRu: "Напитки", NameEn: "Drinks",
			Slug: "ichimliklar",
			Items: []shopItem{
				{Name: "Ichimlik suvi, 1.5 l", NameRu: "Питьевая вода, 1.5 л", NameEn: "Still water, 1.5 l", Price: 5000, Unit: 41},
				{Name: "Gazli ichimlik, 1 l", NameRu: "Газированный напиток, 1 л", NameEn: "Fizzy drink, 1 l", Price: 12000, Unit: 41},
				{Name: "Olma sharbati, 1 l", NameRu: "Яблочный сок, 1 л", NameEn: "Apple juice, 1 l", Price: 18000, Unit: 41},
			},
		},
		{
			Name: "Non va sut mahsulotlari", NameRu: "Хлеб и молочное", NameEn: "Bread and dairy",
			Slug: "non-sut",
			Items: []shopItem{
				{Name: "Non", NameRu: "Хлеб", NameEn: "Bread", Price: 4000},
				{Name: "Sut, 1 l", NameRu: "Молоко, 1 л", NameEn: "Milk, 1 l", Price: 13000, Unit: 41},
				{Name: "Tuxum, 10 dona", NameRu: "Яйцо, 10 шт", NameEn: "Eggs, 10", Price: 18000},
			},
		},
		{
			Name: "Bakaleya", NameRu: "Бакалея", NameEn: "Groceries",
			Slug: "bakaleya",
			Items: []shopItem{
				// ⚠️ Sold by the kilo: the row that proves the scale, the weight
				// barcode and the "/ kg" on the shelf label all work.
				{Name: "Guruch", NameRu: "Рис", NameEn: "Rice", Price: 18500, Unit: 11},
				{Name: "Shakar", NameRu: "Сахар", NameEn: "Sugar", Price: 12000, Unit: 11},
				{Name: "Kungaboqar yog'i, 1 l", NameRu: "Подсолнечное масло, 1 л", NameEn: "Sunflower oil, 1 l", Price: 24000, Unit: 41},
			},
		},
		{
			Name: "Uy-ro'zg'or", NameRu: "Для дома", NameEn: "Household",
			Slug: "uy-rozgor",
			Items: []shopItem{
				{Name: "Xo'jalik sovuni", NameRu: "Хозяйственное мыло", NameEn: "Household soap", Price: 6000},
				{Name: "Idish yuvish vositasi", NameRu: "Средство для мытья посуды", NameEn: "Washing-up liquid", Price: 21000},
			},
		},
	},
	models.BizPharmacy: {
		{
			Name: "Og'riq va isitma", NameRu: "Боль и температура", NameEn: "Pain and fever",
			Slug: "ogriq-isitma",
			Items: []shopItem{
				{Name: "Paratsetamol 500 mg, №20", NameRu: "Парацетамол 500 мг, №20", NameEn: "Paracetamol 500 mg, 20", Price: 9000},
				{Name: "Ibuprofen 200 mg, №20", NameRu: "Ибупрофен 200 мг, №20", NameEn: "Ibuprofen 200 mg, 20", Price: 14000},
			},
		},
		{
			Name: "Shamollash", NameRu: "Простуда", NameEn: "Colds",
			Slug: "shamollash",
			Items: []shopItem{
				{Name: "Burun tomchisi", NameRu: "Капли в нос", NameEn: "Nasal drops", Price: 16000},
				{Name: "Tomoq uchun spray", NameRu: "Спрей для горла", NameEn: "Throat spray", Price: 28000},
				{Name: "Vitamin C, №10", NameRu: "Витамин C, №10", NameEn: "Vitamin C, 10", Price: 11000},
			},
		},
		{
			Name: "Tibbiy buyumlar", NameRu: "Медицинские изделия", NameEn: "Medical supplies",
			Slug: "tibbiy-buyumlar",
			Items: []shopItem{
				{Name: "Bint, steril", NameRu: "Бинт стерильный", NameEn: "Sterile bandage", Price: 5000},
				{Name: "Plastir, №10", NameRu: "Пластырь, №10", NameEn: "Plasters, 10", Price: 7000},
				{Name: "Tibbiy niqob, №10", NameRu: "Маска медицинская, №10", NameEn: "Face masks, 10", Price: 10000},
				{Name: "Elektron termometr", NameRu: "Электронный термометр", NameEn: "Digital thermometer", Price: 45000},
			},
		},
	},
	models.BizClothing: {
		// ⚠️ **No sizes in the sample, though a boutique sells them.** Variants
		// are generated from the product's own axes on the menu screen, and a
		// sample that arrived with twelve of them would be twelve rows to delete
		// before the shop has entered anything of its own. The generator is one
		// button away and it is the shop's own sizes that belong in it.
		{
			Name: "Erkaklar", NameRu: "Мужское", NameEn: "Men",
			Slug: "erkaklar",
			Items: []shopItem{
				{Name: "Ko'ylak, oq", NameRu: "Рубашка, белая", NameEn: "Shirt, white", Price: 210000},
				{Name: "Futbolka", NameRu: "Футболка", NameEn: "T-shirt", Price: 95000},
				{Name: "Shim", NameRu: "Брюки", NameEn: "Trousers", Price: 280000},
			},
		},
		{
			Name: "Ayollar", NameRu: "Женское", NameEn: "Women",
			Slug: "ayollar",
			Items: []shopItem{
				{Name: "Ko'ylak", NameRu: "Платье", NameEn: "Dress", Price: 320000},
				{Name: "Bluzka", NameRu: "Блузка", NameEn: "Blouse", Price: 185000},
			},
		},
		{
			Name: "Aksessuarlar", NameRu: "Аксессуары", NameEn: "Accessories",
			Slug: "aksessuarlar",
			Items: []shopItem{
				{Name: "Kamar", NameRu: "Ремень", NameEn: "Belt", Price: 85000},
				{Name: "Paypoq, 3 juft", NameRu: "Носки, 3 пары", NameEn: "Socks, 3 pairs", Price: 35000},
			},
		},
	},
	models.BizFlowers: {
		// ⚠️ **Stems and wrapping, not finished bouquets.** A florist composes,
		// and a bouquet is a technical card over these rows — written by the shop
		// from the flowers it actually buys. A sample bouquet would be a card
		// pointing at stems this shop does not stock, which is worse than none:
		// its cost would be wrong on the one screen a flower shop is bought for.
		{
			Name: "Kesma gullar", NameRu: "Срезанные цветы", NameEn: "Cut flowers",
			Slug: "kesma-gullar",
			Items: []shopItem{
				{Name: "Atirgul, 60 sm", NameRu: "Роза, 60 см", NameEn: "Rose, 60 cm", Price: 18000},
				{Name: "Xrizantema", NameRu: "Хризантема", NameEn: "Chrysanthemum", Price: 14000},
				{Name: "Lola", NameRu: "Тюльпан", NameEn: "Tulip", Price: 12000},
			},
		},
		{
			Name: "O'rash uchun", NameRu: "Упаковка", NameEn: "Wrapping",
			Slug: "orash-uchun",
			Items: []shopItem{
				// Sold by the metre, which is what `unitCode: 22` is for.
				{Name: "O'ram qog'ozi", NameRu: "Упаковочная бумага", NameEn: "Wrapping paper", Price: 9000, Unit: 22},
				{Name: "Lenta", NameRu: "Лента", NameEn: "Ribbon", Price: 4000, Unit: 22},
			},
		},
		{
			Name: "Qo'shimcha", NameRu: "Дополнительно", NameEn: "Extras",
			Slug: "qoshimcha",
			Items: []shopItem{
				{Name: "Otkritka", NameRu: "Открытка", NameEn: "Greeting card", Price: 8000},
				{Name: "Vaza, shisha", NameRu: "Ваза стеклянная", NameEn: "Glass vase", Price: 65000},
			},
		},
	},
	models.BizButcher: {
		// ⚠️ **Everything here is priced per kilo**, because that is the row that
		// proves the counter scale, the weight barcode and the "/ kg" on the
		// shelf label all agree — and a butcher whose sample was priced per piece
		// would find that out at the counter with a queue.
		{
			Name: "Mol go'shti", NameRu: "Говядина", NameEn: "Beef",
			Slug: "mol-goshti",
			Items: []shopItem{
				{Name: "Mol go'shti, suyaksiz", NameRu: "Говядина без кости", NameEn: "Beef, boneless", Price: 105000, Unit: 11},
				{Name: "Mol go'shti, suyakli", NameRu: "Говядина на кости", NameEn: "Beef on the bone", Price: 85000, Unit: 11},
				{Name: "Qiyma", NameRu: "Фарш", NameEn: "Minced beef", Price: 95000, Unit: 11},
			},
		},
		{
			Name: "Qo'y go'shti", NameRu: "Баранина", NameEn: "Lamb",
			Slug: "qoy-goshti",
			Items: []shopItem{
				{Name: "Qo'y go'shti", NameRu: "Баранина", NameEn: "Lamb", Price: 130000, Unit: 11},
				{Name: "Dumba", NameRu: "Курдюк", NameEn: "Fat tail", Price: 70000, Unit: 11},
			},
		},
		{
			Name: "Parranda va baliq", NameRu: "Птица и рыба", NameEn: "Poultry and fish",
			Slug: "parranda-baliq",
			Items: []shopItem{
				{Name: "Tovuq go'shti", NameRu: "Курица", NameEn: "Chicken", Price: 42000, Unit: 11},
				{Name: "Tovuq filesi", NameRu: "Куриное филе", NameEn: "Chicken fillet", Price: 68000, Unit: 11},
				{Name: "Zog'ora baliq", NameRu: "Карп", NameEn: "Carp", Price: 55000, Unit: 11},
			},
		},
	},
	models.BizCosmetics: {
		{
			Name: "Yuz parvarishi", NameRu: "Уход за лицом", NameEn: "Face care",
			Slug: "yuz-parvarishi",
			Items: []shopItem{
				{Name: "Yuvinish geli", NameRu: "Гель для умывания", NameEn: "Face wash", Price: 65000},
				{Name: "Namlantiruvchi krem", NameRu: "Увлажняющий крем", NameEn: "Moisturiser", Price: 120000},
				{Name: "Quyoshdan himoya, SPF 50", NameRu: "Солнцезащитный крем, SPF 50", NameEn: "Sunscreen, SPF 50", Price: 145000},
			},
		},
		{
			Name: "Bo'yanish", NameRu: "Макияж", NameEn: "Make-up",
			Slug: "boyanish",
			Items: []shopItem{
				{Name: "Tush", NameRu: "Тушь для ресниц", NameEn: "Mascara", Price: 85000},
				{Name: "Lab bo'yog'i", NameRu: "Помада", NameEn: "Lipstick", Price: 75000},
			},
		},
		{
			Name: "Soch va tana", NameRu: "Волосы и тело", NameEn: "Hair and body",
			Slug: "soch-tana",
			Items: []shopItem{
				{Name: "Shampun", NameRu: "Шампунь", NameEn: "Shampoo", Price: 58000},
				{Name: "Tana loson", NameRu: "Лосьон для тела", NameEn: "Body lotion", Price: 69000},
			},
		},
	},
	models.BizHardware: {
		{
			Name: "Asboblar", NameRu: "Инструменты", NameEn: "Tools",
			Slug: "asboblar",
			Items: []shopItem{
				{Name: "Bolg'a", NameRu: "Молоток", NameEn: "Hammer", Price: 45000},
				{Name: "Otvyortka to'plami", NameRu: "Набор отвёрток", NameEn: "Screwdriver set", Price: 120000},
				{Name: "Ruletka, 5 m", NameRu: "Рулетка, 5 м", NameEn: "Tape measure, 5 m", Price: 30000},
			},
		},
		{
			Name: "Mahkamlagichlar", NameRu: "Крепёж", NameEn: "Fixings",
			Slug: "mahkamlagichlar",
			Items: []shopItem{
				// Sold by the kilo, the way a hardware shop actually sells them.
				{Name: "Mix", NameRu: "Гвозди", NameEn: "Nails", Price: 18000, Unit: 11},
				{Name: "Sho'rup", NameRu: "Саморезы", NameEn: "Screws", Price: 26000, Unit: 11},
			},
		},
		{
			Name: "Bo'yoq va elektr", NameRu: "Краски и электрика", NameEn: "Paint and electrics",
			Slug: "boyoq-elektr",
			Items: []shopItem{
				{Name: "Suvli bo'yoq, 1 l", NameRu: "Водоэмульсионная краска, 1 л", NameEn: "Emulsion paint, 1 l", Price: 42000, Unit: 41},
				{Name: "Kabel, 2×1.5", NameRu: "Кабель, 2×1.5", NameEn: "Cable, 2×1.5", Price: 12000, Unit: 22},
				{Name: "LED lampa", NameRu: "LED лампа", NameEn: "LED bulb", Price: 22000},
			},
		},
	},
}

// makerCatalogues is what a counter that *makes* what it sells starts with.
//
// ⚠️ **A separate map from the shops', and separately written, because these
// rows are not stock rows.** A bakery's loaf is flour an hour ago: what it sells
// is a menu item with a technical card behind it, not a packet that is its own
// balance. Writing them through the shop's path would mark each one
// `SellsItself` — and the first tech card the baker wrote would be a card on a
// product that is also its own ingredient, which is a loop the cost report
// cannot answer.
//
// ⚠️ **No cards in the sample either, for the florist's reason** (see the
// flowers catalogue above): a recipe pointing at flour this bakery does not buy
// would cost its bread wrongly on the one screen it is bought for. The rows are
// there so the till, the site and the receipt have something real to show; the
// cards are the owner's first real piece of work.
var makerCatalogues = map[models.BusinessType][]shopCategory{
	models.BizBakery: {
		{
			Name: "Non", NameRu: "Хлеб", NameEn: "Bread",
			Slug: "non",
			Items: []shopItem{
				{Name: "Obi non", NameRu: "Лепёшка", NameEn: "Obi non", Price: 4000},
				{Name: "Patir", NameRu: "Патыр", NameEn: "Patir", Price: 8000},
				{Name: "Bulochka", NameRu: "Булочка", NameEn: "Bun", Price: 3500},
			},
		},
		{
			Name: "Yopilgan mahsulot", NameRu: "Выпечка", NameEn: "Baked goods",
			Slug: "yopilgan",
			Items: []shopItem{
				{Name: "Somsa", NameRu: "Самса", NameEn: "Somsa", Price: 12000},
				{Name: "Qatlama", NameRu: "Слоёная лепёшка", NameEn: "Qatlama", Price: 10000},
			},
		},
	},
	models.BizCoffee: {
		{
			Name: "Qahva", NameRu: "Кофе", NameEn: "Coffee",
			Slug: "qahva",
			Items: []shopItem{
				{Name: "Espresso", NameRu: "Эспрессо", NameEn: "Espresso", Price: 15000},
				{Name: "Amerikano", NameRu: "Американо", NameEn: "Americano", Price: 18000},
				{Name: "Kapuchino", NameRu: "Капучино", NameEn: "Cappuccino", Price: 25000},
				{Name: "Latte", NameRu: "Латте", NameEn: "Latte", Price: 27000},
			},
		},
		{
			Name: "Boshqa ichimliklar", NameRu: "Другие напитки", NameEn: "Other drinks",
			Slug: "boshqa-ichimliklar",
			Items: []shopItem{
				{Name: "Choy", NameRu: "Чай", NameEn: "Tea", Price: 12000},
				{Name: "Issiq shokolad", NameRu: "Горячий шоколад", NameEn: "Hot chocolate", Price: 28000},
			},
		},
		{
			Name: "Yoniga", NameRu: "К кофе", NameEn: "With coffee",
			Slug: "yoniga",
			Items: []shopItem{
				{Name: "Kruassan", NameRu: "Круассан", NameEn: "Croissant", Price: 20000},
				{Name: "Chizkeyk, bo'lak", NameRu: "Чизкейк, кусок", NameEn: "Cheesecake, slice", Price: 32000},
			},
		},
	},
	models.BizPastry: {
		{
			Name: "Tortlar", NameRu: "Торты", NameEn: "Cakes",
			Slug: "tortlar",
			Items: []shopItem{
				// ⚠️ Priced per kilo: a cake shop's whole till works that way, and
				// a sample priced per piece would teach the wrong habit on day one.
				{Name: "Medovik", NameRu: "Медовик", NameEn: "Honey cake", Price: 150000, Unit: 11},
				{Name: "Napoleon", NameRu: "Наполеон", NameEn: "Napoleon", Price: 160000, Unit: 11},
			},
		},
		{
			Name: "Shirinliklar", NameRu: "Десерты", NameEn: "Desserts",
			Slug: "shirinliklar",
			Items: []shopItem{
				{Name: "Keks", NameRu: "Кекс", NameEn: "Muffin", Price: 15000},
				{Name: "Ekler", NameRu: "Эклер", NameEn: "Éclair", Price: 18000},
				{Name: "Pirojnoe", NameRu: "Пирожное", NameEn: "Pastry", Price: 20000},
			},
		},
	},
}

// writeShopCatalogue creates the sample a shop starts with.
//
// ⚠️ **One product at a time, not InsertMany.** Each row needs its stock row
// created and its id written back into the product before it is stored — the
// same order the panel does it in. A bulk insert would be faster and would leave
// a catalogue of goods that cannot be counted.
func writeShopCatalogue(
	ctx context.Context, store *repository.Store, biz models.BusinessType,
) error {
	cats, ok := shopCatalogues[biz]
	if !ok {
		return nil
	}
	now := time.Now()
	items := 0
	for ci, cat := range cats {
		res, err := store.Categories.InsertOne(ctx, models.Category{
			Name:      cat.Name,
			NameRu:    cat.NameRu,
			NameEn:    cat.NameEn,
			Slug:      cat.Slug,
			SortOrder: ci,
			IsActive:  true,
		})
		if err != nil {
			return fmt.Errorf("insert category %q: %w", cat.Name, err)
		}
		catID, _ := res.InsertedID.(primitive.ObjectID)

		for i, it := range cat.Items {
			m := models.MenuItem{
				CategoryID:  catID,
				Name:        it.Name,
				NameRu:      it.NameRu,
				NameEn:      it.NameEn,
				Price:       it.Price,
				UnitCode:    it.Unit,
				IsAvailable: true,
				// ⚠️ **The flag that makes it a shop's product at all**: it is
				// its own stock row, its sale takes it off the shelf, and the
				// labels screen will give it a barcode the first time one is
				// printed.
				SellsItself: true,
				SortOrder:   i,
				UpdatedAt:   now,
			}
			// Creates the stock row and writes the one-line card. Exactly what
			// the panel does when somebody adds a product by hand.
			if err := repository.SyncProductStock(ctx, store, &m); err != nil {
				return fmt.Errorf("stock row for %q: %w", it.Name, err)
			}
			if _, err := store.Menu.InsertOne(ctx, m); err != nil {
				return fmt.Errorf("insert %q: %w", it.Name, err)
			}
			items++
		}
	}
	log.Printf("seed: created sample catalogue for %s (%d items)", biz, items)
	return nil
}

// writeMakerCatalogue creates the short menu a bakery, a coffee house or a
// pastry shop starts with.
//
// ⚠️ **Ordinary menu items, written in bulk, with no stock row and no
// `SellsItself`.** What a maker sells is not a packet on a shelf: the loaf is
// costed through a technical card over flour and yeast, which are the stock
// rows. Marking these as their own stock — the shop path above — would make the
// first card an owner writes point a product at itself.
func writeMakerCatalogue(
	ctx context.Context, store *repository.Store, biz models.BusinessType,
) error {
	cats, ok := makerCatalogues[biz]
	if !ok {
		return nil
	}
	now := time.Now()
	items := 0
	for ci, cat := range cats {
		res, err := store.Categories.InsertOne(ctx, models.Category{
			Name:      cat.Name,
			NameRu:    cat.NameRu,
			NameEn:    cat.NameEn,
			Slug:      cat.Slug,
			SortOrder: ci,
			IsActive:  true,
		})
		if err != nil {
			return fmt.Errorf("insert category %q: %w", cat.Name, err)
		}
		catID, _ := res.InsertedID.(primitive.ObjectID)

		docs := make([]any, 0, len(cat.Items))
		for i, it := range cat.Items {
			docs = append(docs, models.MenuItem{
				CategoryID:  catID,
				Name:        it.Name,
				NameRu:      it.NameRu,
				NameEn:      it.NameEn,
				Price:       it.Price,
				UnitCode:    it.Unit,
				IsAvailable: true,
				SortOrder:   i,
				UpdatedAt:   now,
			})
		}
		if len(docs) == 0 {
			continue
		}
		if _, err := store.Menu.InsertMany(ctx, docs); err != nil {
			return fmt.Errorf("insert %q: %w", cat.Name, err)
		}
		items += len(docs)
	}
	log.Printf("seed: created sample catalogue for %s (%d items)", biz, items)
	return nil
}
