package uz.keel.guest.i18n

import uz.keel.design.Lang

// Every word this application says on its own behalf.
//
// ⚠️ **A typed dictionary rather than string resources**, exactly as the five
// staff applications do it: a missing key is a compile error here and a blank
// space on the screen there. In an application a guest sees, a blank space is
// the restaurant looking unfinished.
//
// ⚠️ **Uzbek is the base and all three are written together.** A language added
// later is a language that stays half-done — the lesson `lib/i18n/growth.ts`
// paid for on the web side.
//
// ⚠️ **The restaurant's own words are not here.** Dish names, category names and
// descriptions come from the panel in whatever languages the owner filled in,
// and fall back to Uzbek (`Named.pick`). Text an owner typed can never come out
// of a dictionary we ship — and neither can a refusal the server wrote: those
// arrive already translated (`middleware.Lang`), which is why the API sends
// `Accept-Language` on every request.

data class Dict(
    /** This language, named in itself — "Русский", never "Ruscha". */
    val lang: String,
    val tabs: Tabs,
    val menu: Menu,
    val dish: Dish,
    val cart: Cart,
    val checkout: Checkout,
    val map: MapPick,
    val order: Order,
    val common: Common,
) {
    data class Tabs(
        val menu: String,
        val cart: String,
        val orders: String,
    )

    data class Menu(
        /** ⚠️ **There is no home page and this is the first thing on screen.**
         *  A restaurant application opened by somebody who is hungry has one
         *  job, and a landing page is a tap between them and it. */
        val search: String,
        val searchEmpty: (String) -> String,
        val all: String,
        val soldOut: String,
        val closed: String,
        val closedHint: String,
        val empty: String,
        val loadFailed: String,
        val retry: String,
    )

    data class Dish(
        val add: String,
        val comment: String,
        val commentHint: String,
        /** ⚠️ Phrased as the instruction it is, never as "required": a label
         *  that names a rule leaves somebody looking for what to do about it. */
        val choose: String,
        val chooseMany: String,
        val needsChoice: (String) -> String,
    )

    data class Cart(
        val title: String,
        val empty: String,
        val emptyHint: String,
        val subtotal: String,
        val checkout: String,
        val clear: String,
    )

    data class Checkout(
        val title: String,
        val delivery: String,
        val pickup: String,
        val dinein: String,
        val name: String,
        val phone: String,
        val address: String,
        val addressPick: String,
        val addressHint: String,
        val comment: String,
        val payment: String,
        val payCash: String,
        val payCard: String,
        val promo: String,
        val subtotal: String,
        val discount: String,
        val deliveryFee: String,
        val total: String,
        val points: (String) -> String,
        val pointsEarn: (String) -> String,
        val minOrder: (String) -> String,
        val notDelivered: String,
        val soldOut: (String) -> String,
        val send: String,
        val nameNeeded: String,
        val phoneNeeded: String,
        val addressNeeded: String,
        val failed: String,
    )

    /** The address picker's own words. One step of the checkout — a guest never
     *  opens it for its own sake. */
    data class MapPick(
        val title: String,
        val search: String,
        val myLocation: String,
        val confirm: String,
        val noKey: String,
        val denied: String,
    )

    data class Order(
        val title: String,
        val number: (String) -> String,
        val pending: String,
        val confirmed: String,
        val preparing: String,
        val onTheWay: String,
        val delivered: String,
        val cancelled: String,
        val none: String,
        val noneHint: String,
        val payNow: String,
        val loadFailed: String,
        val placed: String,
    )

    data class Common(
        val ok: String,
        val retry: String,
        val loading: String,
        val back: String,
        val close: String,
    )
}

private val uz = Dict(
    lang = "O'zbekcha",
    tabs = Dict.Tabs(menu = "Menyu", cart = "Savat", orders = "Buyurtmalar"),
    menu = Dict.Menu(
        search = "Taom qidirish",
        searchEmpty = { q -> "«$q» bo'yicha hech nima topilmadi" },
        all = "Hammasi",
        soldOut = "Tugadi",
        closed = "Hozir yopiq",
        // ⚠️ Menyu baribir ko'rsatiladi. Yopiq restoranning menyusini
        // yashirish — ertaga keladigan odamni bugun yo'qotish.
        closedHint = "Menyuni ko'rishingiz mumkin, buyurtma ish vaqtida qabul qilinadi",
        empty = "Menyu hali to'ldirilmagan",
        loadFailed = "Menyuni ochib bo'lmadi",
        retry = "Qayta urinish",
    ),
    dish = Dict.Dish(
        add = "Savatga",
        comment = "Izoh",
        commentHint = "Masalan: piyozsiz",
        choose = "Tanlang",
        chooseMany = "Bir nechtasini tanlash mumkin",
        needsChoice = { name -> "«$name» bo'yicha tanlov qiling" },
    ),
    cart = Dict.Cart(
        title = "Savat",
        empty = "Savat bo'sh",
        emptyHint = "Menyudan taom tanlang",
        subtotal = "Taomlar",
        checkout = "Rasmiylashtirish",
        clear = "Tozalash",
    ),
    checkout = Dict.Checkout(
        title = "Buyurtma",
        delivery = "Yetkazib berish",
        pickup = "Olib ketish",
        dinein = "Zalda",
        name = "Ismingiz",
        phone = "Telefon",
        address = "Manzil",
        addressPick = "Xaritada tanlash",
        addressHint = "Uy, kvartira, mo'ljal",
        comment = "Izoh",
        payment = "To'lov",
        payCash = "Naqd",
        payCard = "Karta",
        promo = "Promokod",
        subtotal = "Taomlar",
        discount = "Chegirma",
        deliveryFee = "Yetkazib berish",
        total = "Jami",
        points = { n -> "$n ball ishlatildi" },
        pointsEarn = { n -> "Bu buyurtmadan $n ball yig'iladi" },
        minOrder = { sum -> "Eng kam buyurtma — $sum" },
        notDelivered = "Bu manzilga yetkazib berilmaydi",
        soldOut = { names -> "Hozir yo'q: $names" },
        send = "Buyurtma berish",
        nameNeeded = "Ismingizni yozing",
        phoneNeeded = "Telefon raqamingizni yozing",
        addressNeeded = "Manzilni xaritada belgilang",
        failed = "Buyurtma yuborilmadi",
    ),
    map = Dict.MapPick(
        title = "Manzilni belgilang",
        search = "Ko'cha, uy, mo'ljal",
        myLocation = "Men turgan joy",
        confirm = "Shu yerga",
        // ⚠️ Kalit yo'qligi — restoranning sozlamasi, mehmonning aybi emas.
        // Shuning uchun matn odamni telefon qilishga yo'naltiradi.
        noKey = "Xarita sozlanmagan — manzilni izoh maydoniga yozing yoki restoranga qo'ng'iroq qiling",
        denied = "Joylashuvga ruxsat berilmadi — nuqtani qo'lda suring",
    ),
    order = Dict.Order(
        title = "Buyurtmalar",
        number = { n -> "Buyurtma $n" },
        pending = "Qabul qilinmoqda",
        confirmed = "Qabul qilindi",
        preparing = "Tayyorlanmoqda",
        onTheWay = "Yo'lda",
        delivered = "Yetkazildi",
        cancelled = "Bekor qilindi",
        none = "Hali buyurtma yo'q",
        noneHint = "Bergan buyurtmangiz shu yerda ko'rinadi",
        payNow = "To'lash",
        loadFailed = "Buyurtmani ochib bo'lmadi",
        placed = "Buyurtmangiz qabul qilindi",
    ),
    common = Dict.Common(
        ok = "Yaxshi",
        retry = "Qayta urinish",
        loading = "Yuklanmoqda…",
        back = "Orqaga",
        close = "Yopish",
    ),
)

private val ru = Dict(
    lang = "Русский",
    tabs = Dict.Tabs(menu = "Меню", cart = "Корзина", orders = "Заказы"),
    menu = Dict.Menu(
        search = "Поиск по меню",
        searchEmpty = { q -> "По запросу «$q» ничего не найдено" },
        all = "Всё",
        soldOut = "Закончилось",
        closed = "Сейчас закрыто",
        closedHint = "Меню можно посмотреть, заказы принимаются в рабочее время",
        empty = "Меню пока не заполнено",
        loadFailed = "Не удалось открыть меню",
        retry = "Повторить",
    ),
    dish = Dict.Dish(
        add = "В корзину",
        comment = "Комментарий",
        commentHint = "Например: без лука",
        choose = "Выберите",
        chooseMany = "Можно выбрать несколько",
        needsChoice = { name -> "Выберите «$name»" },
    ),
    cart = Dict.Cart(
        title = "Корзина",
        empty = "Корзина пуста",
        emptyHint = "Выберите блюдо в меню",
        subtotal = "Блюда",
        checkout = "Оформить",
        clear = "Очистить",
    ),
    checkout = Dict.Checkout(
        title = "Заказ",
        delivery = "Доставка",
        pickup = "Самовывоз",
        dinein = "В зале",
        name = "Ваше имя",
        phone = "Телефон",
        address = "Адрес",
        addressPick = "Выбрать на карте",
        addressHint = "Дом, квартира, ориентир",
        comment = "Комментарий",
        payment = "Оплата",
        payCash = "Наличными",
        payCard = "Картой",
        promo = "Промокод",
        subtotal = "Блюда",
        discount = "Скидка",
        deliveryFee = "Доставка",
        total = "Итого",
        points = { n -> "Списано баллов: $n" },
        pointsEarn = { n -> "За заказ начислится $n баллов" },
        minOrder = { sum -> "Минимальный заказ — $sum" },
        notDelivered = "По этому адресу доставки нет",
        soldOut = { names -> "Сейчас нет: $names" },
        send = "Заказать",
        nameNeeded = "Укажите имя",
        phoneNeeded = "Укажите телефон",
        addressNeeded = "Отметьте адрес на карте",
        failed = "Заказ не отправлен",
    ),
    map = Dict.MapPick(
        title = "Отметьте адрес",
        search = "Улица, дом, ориентир",
        myLocation = "Я здесь",
        confirm = "Сюда",
        noKey = "Карта не настроена — впишите адрес в комментарий или позвоните в ресторан",
        denied = "Доступ к геопозиции не дан — перетащите точку вручную",
    ),
    order = Dict.Order(
        title = "Заказы",
        number = { n -> "Заказ $n" },
        pending = "Принимается",
        confirmed = "Принят",
        preparing = "Готовится",
        onTheWay = "В пути",
        delivered = "Доставлен",
        cancelled = "Отменён",
        none = "Заказов пока нет",
        noneHint = "Здесь появятся ваши заказы",
        payNow = "Оплатить",
        loadFailed = "Не удалось открыть заказ",
        placed = "Заказ принят",
    ),
    common = Dict.Common(
        ok = "Хорошо",
        retry = "Повторить",
        loading = "Загрузка…",
        back = "Назад",
        close = "Закрыть",
    ),
)

private val en = Dict(
    lang = "English",
    tabs = Dict.Tabs(menu = "Menu", cart = "Basket", orders = "Orders"),
    menu = Dict.Menu(
        search = "Search the menu",
        searchEmpty = { q -> "Nothing found for \"$q\"" },
        all = "All",
        soldOut = "Sold out",
        closed = "Closed right now",
        closedHint = "You can browse the menu; orders are taken during opening hours",
        empty = "The menu has not been filled in yet",
        loadFailed = "Could not open the menu",
        retry = "Try again",
    ),
    dish = Dict.Dish(
        add = "Add",
        comment = "Note",
        commentHint = "For example: no onion",
        choose = "Choose",
        chooseMany = "You can pick more than one",
        needsChoice = { name -> "Choose \"$name\"" },
    ),
    cart = Dict.Cart(
        title = "Basket",
        empty = "The basket is empty",
        emptyHint = "Pick something from the menu",
        subtotal = "Food",
        checkout = "Checkout",
        clear = "Clear",
    ),
    checkout = Dict.Checkout(
        title = "Order",
        delivery = "Delivery",
        pickup = "Pickup",
        dinein = "At a table",
        name = "Your name",
        phone = "Phone",
        address = "Address",
        addressPick = "Pick on the map",
        addressHint = "House, flat, landmark",
        comment = "Note",
        payment = "Payment",
        payCash = "Cash",
        payCard = "Card",
        promo = "Promo code",
        subtotal = "Food",
        discount = "Discount",
        deliveryFee = "Delivery",
        total = "Total",
        points = { n -> "$n points used" },
        pointsEarn = { n -> "This order earns $n points" },
        minOrder = { sum -> "Minimum order is $sum" },
        notDelivered = "We do not deliver to this address",
        soldOut = { names -> "Not available: $names" },
        send = "Place the order",
        nameNeeded = "Enter your name",
        phoneNeeded = "Enter your phone number",
        addressNeeded = "Mark the address on the map",
        failed = "The order was not sent",
    ),
    map = Dict.MapPick(
        title = "Mark the address",
        search = "Street, house, landmark",
        myLocation = "Where I am",
        confirm = "Deliver here",
        noKey = "The map is not set up — write the address in the note or call the restaurant",
        denied = "Location was not allowed — drag the pin instead",
    ),
    order = Dict.Order(
        title = "Orders",
        number = { n -> "Order $n" },
        pending = "Being accepted",
        confirmed = "Accepted",
        preparing = "Being cooked",
        onTheWay = "On the way",
        delivered = "Delivered",
        cancelled = "Cancelled",
        none = "No orders yet",
        noneHint = "Your orders will show up here",
        payNow = "Pay",
        loadFailed = "Could not open the order",
        placed = "Your order has been accepted",
    ),
    common = Dict.Common(
        ok = "OK",
        retry = "Try again",
        loading = "Loading…",
        back = "Back",
        close = "Close",
    ),
)

val DICTS: Map<Lang, Dict> = mapOf(Lang.Uz to uz, Lang.Ru to ru, Lang.En to en)
