package uz.keel.courier.i18n

import uz.keel.design.Lang

// The application's own words, in the three languages a restaurant here is run
// in.
//
// ⚠️ **Its own dictionary rather than the panel's**, measured on the waiter
// build: importing the admin dictionary cost about 500 KB to use seventy
// strings. What is worth sharing is the *rules*; the wording of screens that
// exist only here is this application's own.
//
// ⚠️ **The refusals are the strings that matter.** Most of this app is three
// screens a courier learns in a day; the sentences that decide whether somebody
// stands at a gate pressing a dead button are the ones on the arrival gate, and
// they are written to say what to *do*, not what went wrong.
//
// ⚠️ **Data classes with named arguments, not a map.** A missing key would be a
// blank line in front of a Russian-speaking courier; this makes it a compile
// error — which is what `Dict = typeof uz` did in the TypeScript original.

data class Dict(
    val lang: String,
    val appName: String,
    val server: Server,
    val login: Login,
    val tabs: Tabs,
    val shift: Shift,
    val geo: Geo,
    val orders: Orders,
    val gate: GateWords,
    val earnings: Earnings,
    val settings: Settings,
    val push: Push,
    val offline: Offline,
    val common: Common,
) {
    data class Server(
        val title: String,
        val hint: String,
        val placeholder: String,
        val next: String,
        val bad: String,
    )

    data class Login(
        val title: String,
        val username: String,
        val password: String,
        val submit: String,
        val other: String,
        val failed: String,
    )

    data class Tabs(val orders: String, val earnings: String, val settings: String)

    data class Shift(
        val title: String,
        val off: String,
        val free: String,
        val busy: String,
        val offHint: String,
        val freeHint: String,
        val busyHint: String,
        /** ⚠️ Says what the switch causes, not what it is. "Off" on a courier's
         *  phone means the dispatcher stops giving them work — worth spelling
         *  out before somebody leaves it off overnight and wonders why it is
         *  quiet. */
        val startHint: String,
    )

    data class Geo(
        val on: String,
        val waiting: String,
        val lastSent: (String) -> String,
        val queued: (Int) -> String,
        val denied: String,
        val deniedHint: String,
        val allow: String,
        /** ⚠️ Said only when it is true: with the service running the phone can
         *  go in a pocket, and without it the shift dies with the screen. A
         *  courier who does not know which they have finds out at a door with a
         *  shut button. */
        val inBackground: String,
        val keepOpen: String,
    )

    data class Orders(
        val title: (Int) -> String,
        val empty: String,
        val emptyHint: String,
        val offEmpty: String,
        val offEmptyHint: String,
        val cash: (String) -> String,
        val paid: String,
        val pickUp: String,
        val deliver: String,
        val call: String,
        val route: String,
        val routeTitle: String,
        val failed: String,
        val away: (Int) -> String,
        val atDoor: String,
    )

    data class GateWords(
        val tooFar: (Int, Int) -> String,
        val noFix: String,
        val stale: String,
    )

    data class Earnings(
        val title: String,
        val today: String,
        val week: String,
        val month: String,
        val all: String,
        val deliveries: (Int) -> String,
        val cashInHand: (String) -> String,
        val cashClear: String,
        val cash: String,
        val history: String,
        val historyEmpty: String,
    )

    data class Settings(
        val title: String,
        val language: String,
        val theme: String,
        val themeSystem: String,
        val themeLight: String,
        val themeDark: String,
        val account: String,
        val restaurant: String,
        val phone: String,
        val version: String,
        val signOut: String,
        val changeServer: String,
        val changeServerHint: String,
    )

    data class Push(
        val title: String,
        val working: String,
        val asking: String,
        val denied: String,
        val failed: String,
        val hintWorking: String,
        val hintAsking: String,
        val hintDenied: String,
        val hintFailed: String,
    )

    data class Offline(val title: String, val body: String, val retrying: (Int) -> String)

    data class Common(
        val retry: String,
        val loading: String,
        val cancel: String,
        val close: String,
        val ok: String,
    )
}

private fun km(m: Int, unit: String): String =
    if (m < 1000) "$m m" else String.format(java.util.Locale.US, "%.1f %s", m / 1000.0, unit)

val UZ = Dict(
    lang = "O'zbekcha",
    appName = "Keel Courier",
    server = Dict.Server(
        title = "Restoran manzili",
        hint = "Restoraningizning qisqa nomi yoki to'liq manzili",
        placeholder = "osh",
        next = "Davom etish",
        bad = "Bu manzilga o'xshamaydi",
    ),
    login = Dict.Login(
        title = "Kirish",
        username = "Login",
        password = "Parol",
        submit = "Kirish",
        other = "Boshqa restoran",
        failed = "Kirib bo'lmadi",
    ),
    tabs = Dict.Tabs(orders = "Buyurtmalar", earnings = "Hisob", settings = "Sozlamalar"),
    shift = Dict.Shift(
        title = "Smena",
        off = "Ishda emasman",
        free = "Bo'shman",
        busy = "Bandman",
        offHint = "Buyurtma berilmaydi",
        freeHint = "Yangi buyurtma keladi",
        busyHint = "Hozir yetkazyapman",
        startHint = "Ishni boshlash uchun «Bo'shman» ni tanlang — shundan keyin " +
            "buyurtma beriladi va joylashuv yuboriladi.",
    ),
    geo = Dict.Geo(
        on = "Yuborilmoqda",
        waiting = "signal kutilmoqda",
        lastSent = { time -> "oxirgi: " + time },
        queued = { n -> "" + n + " ta navbatda" },
        denied = "Joylashuvga ruxsat berilmagan",
        deniedHint = "Telefon sozlamalaridan ruxsat bering: mijoz manzilida " +
            "ekaningiz shu bilan tekshiriladi.",
        allow = "Ruxsat berish",
        inBackground = "Fonda ham yuborilmoqda — telefonni cho'ntakka solsangiz bo'ladi",
        keepOpen = "Joylashuv xizmati ishga tushmadi — ilovani ochiq qoldiring.",
    ),
    orders = Dict.Orders(
        title = { n -> if (n == 0) "Buyurtmalar" else "Buyurtmalar (" + n + ")" },
        empty = "Hozircha buyurtma yo'q",
        emptyHint = "Yangi buyurtma berilganda shu yerda paydo bo'ladi.",
        offEmpty = "Siz ishda emassiz",
        offEmptyHint = "Buyurtma olish uchun yuqoridan «Bo'shman» ni tanlang.",
        cash = { sum -> "Naqd olinadi: " + sum },
        paid = "Onlayn to'langan",
        pickUp = "Olib chiqdim",
        deliver = "Yetkazildi",
        call = "Qo'ng'iroq",
        route = "Yo'l",
        routeTitle = "Mijozgacha yo'l",
        failed = "Bajarilmadi",
        away = { m -> km(m, "km") },
        atDoor = "Manzildasiz",
    ),
    gate = Dict.GateWords(
        tooFar = { m, need ->
            "Mijoz manzilidan " + m + " m uzoqdasiz. «Yetkazildi» ni " + need +
                " m ichida bosish mumkin."
        },
        noFix = "Joylashuv hali aniqlanmadi — GPS ni yoqing va bir oz kuting.",
        stale = "Joylashuv eskirgan — ilovani ochiq qoldiring, yangilanishini kuting.",
    ),
    earnings = Dict.Earnings(
        title = "Hisob",
        today = "Bugun",
        week = "Hafta",
        month = "Oy",
        all = "Jami",
        deliveries = { n -> "" + n + " ta yetkazish" },
        cashInHand = { sum -> "Qo'lingizda " + sum + " naqd pul bor — restoranga topshiring." },
        cashClear = "Naqd pul topshirilgan — qarz yo'q.",
        cash = "Naqd",
        history = "Yetkazilganlar",
        historyEmpty = "Hali yetkazilgan buyurtma yo'q",
    ),
    settings = Dict.Settings(
        title = "Sozlamalar",
        language = "Til",
        theme = "Ko'rinish",
        themeSystem = "Tizim bo'yicha",
        themeLight = "Yorug'",
        themeDark = "Qorong'i",
        account = "Hisob",
        restaurant = "Restoran",
        phone = "Telefon",
        version = "Versiya",
        signOut = "Chiqish",
        changeServer = "Boshqa restoran",
        changeServerHint = "Chiqish va boshqa restoran manzilini kiritish",
    ),
    push = Dict.Push(
        title = "Bildirishnomalar",
        working = "Yoqilgan",
        asking = "Tekshirilmoqda…",
        denied = "Ruxsat berilmagan",
        failed = "Ro'yxatdan o'tmadi",
        hintWorking = "Yangi buyurtma, bekor qilish va manzil o'zgarishi haqida xabar keladi",
        hintAsking = "Bir soniya",
        hintDenied = "Telefon sozlamalaridan bildirishnomalarni yoqing",
        hintFailed = "Internet yoki server bilan bog'lanib bo'lmadi",
    ),
    offline = Dict.Offline(
        title = "Internet yo'q",
        body = "Ilova serverga ulana olmadi. Mobil internet yoki Wi-Fi ni tekshiring — " +
            "ulanish tiklanishi bilan ekran o'zi ochiladi.",
        retrying = { n -> if (n == 0) "Tekshirilmoqda…" else "Qayta tekshirildi: " + n },
    ),
    common = Dict.Common(
        retry = "Qayta urinish",
        loading = "Yuklanmoqda…",
        cancel = "Bekor qilish",
        close = "Yopish",
        ok = "Tushunarli",
    ),
)

val RU = Dict(
    lang = "Русский",
    appName = "Keel Courier",
    server = Dict.Server(
        title = "Адрес ресторана",
        hint = "Короткое имя вашего ресторана или полный адрес",
        placeholder = "osh",
        next = "Продолжить",
        bad = "Это не похоже на адрес",
    ),
    login = Dict.Login(
        title = "Вход",
        username = "Логин",
        password = "Пароль",
        submit = "Войти",
        other = "Другой ресторан",
        failed = "Не удалось войти",
    ),
    tabs = Dict.Tabs(orders = "Заказы", earnings = "Счёт", settings = "Настройки"),
    shift = Dict.Shift(
        title = "Смена",
        off = "Не на смене",
        free = "Свободен",
        busy = "Занят",
        offHint = "Заказы не дают",
        freeHint = "Придёт новый заказ",
        busyHint = "Сейчас доставляю",
        startHint = "Чтобы начать работу, выберите «Свободен» — после этого дают " +
            "заказы и отправляется геопозиция.",
    ),
    geo = Dict.Geo(
        on = "Отправляется",
        waiting = "ждём сигнал",
        lastSent = { time -> "последняя: " + time },
        queued = { n -> "" + n + " в очереди" },
        denied = "Доступ к геопозиции запрещён",
        deniedHint = "Разрешите в настройках телефона: так проверяется, что вы на " +
            "адресе клиента.",
        allow = "Разрешить",
        inBackground = "Отправляется и в фоне — телефон можно убрать в карман",
        keepOpen = "Служба геопозиции не запустилась — держите приложение открытым.",
    ),
    orders = Dict.Orders(
        title = { n -> if (n == 0) "Заказы" else "Заказы (" + n + ")" },
        empty = "Пока заказов нет",
        emptyHint = "Новый заказ появится здесь.",
        offEmpty = "Вы не на смене",
        offEmptyHint = "Чтобы получать заказы, выберите «Свободен» выше.",
        cash = { sum -> "Взять наличными: " + sum },
        paid = "Оплачено онлайн",
        pickUp = "Забрал",
        deliver = "Доставлено",
        call = "Позвонить",
        route = "Маршрут",
        routeTitle = "Маршрут до клиента",
        failed = "Не выполнено",
        away = { m -> km(m, "км") },
        atDoor = "Вы на адресе",
    ),
    gate = Dict.GateWords(
        tooFar = { m, need ->
            "Вы в " + m + " м от адреса клиента. «Доставлено» можно нажать в пределах " +
                need + " м."
        },
        noFix = "Геопозиция ещё не определена — включите GPS и подождите немного.",
        stale = "Геопозиция устарела — оставьте приложение открытым и дождитесь обновления.",
    ),
    earnings = Dict.Earnings(
        title = "Счёт",
        today = "Сегодня",
        week = "Неделя",
        month = "Месяц",
        all = "Всего",
        deliveries = { n -> "" + n + " доставок" },
        cashInHand = { sum -> "У вас на руках " + sum + " наличными — сдайте в ресторан." },
        cashClear = "Наличные сданы — долгов нет.",
        cash = "Наличные",
        history = "Доставленные",
        historyEmpty = "Доставленных заказов пока нет",
    ),
    settings = Dict.Settings(
        title = "Настройки",
        language = "Язык",
        theme = "Оформление",
        themeSystem = "Как в системе",
        themeLight = "Светлое",
        themeDark = "Тёмное",
        account = "Аккаунт",
        restaurant = "Ресторан",
        phone = "Телефон",
        version = "Версия",
        signOut = "Выйти",
        changeServer = "Другой ресторан",
        changeServerHint = "Выйти и ввести адрес другого ресторана",
    ),
    push = Dict.Push(
        title = "Уведомления",
        working = "Включены",
        asking = "Проверяем…",
        denied = "Доступ запрещён",
        failed = "Не зарегистрировано",
        hintWorking = "Придёт сообщение о новом заказе, отмене и смене адреса",
        hintAsking = "Секунду",
        hintDenied = "Включите уведомления в настройках телефона",
        hintFailed = "Не удалось связаться с интернетом или сервером",
    ),
    offline = Dict.Offline(
        title = "Нет интернета",
        body = "Приложение не смогло связаться с сервером. Проверьте мобильный интернет " +
            "или Wi-Fi — как только связь появится, экран откроется сам.",
        retrying = { n -> if (n == 0) "Проверяем…" else "Проверок: " + n },
    ),
    common = Dict.Common(
        retry = "Повторить",
        loading = "Загрузка…",
        cancel = "Отмена",
        close = "Закрыть",
        ok = "Понятно",
    ),
)

val EN = Dict(
    lang = "English",
    appName = "Keel Courier",
    server = Dict.Server(
        title = "Restaurant address",
        hint = "Your restaurant's short name or its full address",
        placeholder = "osh",
        next = "Continue",
        bad = "That does not look like an address",
    ),
    login = Dict.Login(
        title = "Sign in",
        username = "Username",
        password = "Password",
        submit = "Sign in",
        other = "Another restaurant",
        failed = "Could not sign in",
    ),
    tabs = Dict.Tabs(orders = "Orders", earnings = "Earnings", settings = "Settings"),
    shift = Dict.Shift(
        title = "Shift",
        off = "Off shift",
        free = "Available",
        busy = "Busy",
        offHint = "No orders given",
        freeHint = "New orders come in",
        busyHint = "Delivering now",
        startHint = "Pick “Available” to start work — orders are given out and " +
            "your location is sent from then on.",
    ),
    geo = Dict.Geo(
        on = "Sending",
        waiting = "waiting for a signal",
        lastSent = { time -> "last: " + time },
        queued = { n -> "" + n + " queued" },
        denied = "Location permission was refused",
        deniedHint = "Allow it in the phone's settings: this is what proves you are at " +
            "the customer's address.",
        allow = "Allow",
        inBackground = "Sending in the background too — the phone can go in a pocket",
        keepOpen = "The location service did not start — keep the app open.",
    ),
    orders = Dict.Orders(
        title = { n -> if (n == 0) "Orders" else "Orders (" + n + ")" },
        empty = "No orders yet",
        emptyHint = "A new order will appear here.",
        offEmpty = "You are off shift",
        offEmptyHint = "Pick “Available” above to get orders.",
        cash = { sum -> "Collect in cash: " + sum },
        paid = "Paid online",
        pickUp = "Picked up",
        deliver = "Delivered",
        call = "Call",
        route = "Route",
        routeTitle = "Route to the customer",
        failed = "That did not go through",
        away = { m -> km(m, "km") },
        atDoor = "You are at the address",
    ),
    gate = Dict.GateWords(
        tooFar = { m, need ->
            "You are " + m + " m from the customer's address. “Delivered” can be " +
                "pressed within " + need + " m."
        },
        noFix = "No location fix yet — turn GPS on and give it a moment.",
        stale = "The location is stale — keep the app open and wait for an update.",
    ),
    earnings = Dict.Earnings(
        title = "Earnings",
        today = "Today",
        week = "Week",
        month = "Month",
        all = "All time",
        deliveries = { n -> "" + n + " deliveries" },
        cashInHand = { sum -> "You are holding " + sum + " in cash — hand it in at the restaurant." },
        cashClear = "Cash handed in — nothing owed.",
        cash = "Cash",
        history = "Delivered",
        historyEmpty = "No deliveries yet",
    ),
    settings = Dict.Settings(
        title = "Settings",
        language = "Language",
        theme = "Appearance",
        themeSystem = "Follow the system",
        themeLight = "Light",
        themeDark = "Dark",
        account = "Account",
        restaurant = "Restaurant",
        phone = "Phone",
        version = "Version",
        signOut = "Sign out",
        changeServer = "Another restaurant",
        changeServerHint = "Sign out and enter another restaurant's address",
    ),
    push = Dict.Push(
        title = "Notifications",
        working = "On",
        asking = "Checking…",
        denied = "Permission refused",
        failed = "Not registered",
        hintWorking = "You will be told about a new order, a cancellation and a moved address",
        hintAsking = "One moment",
        hintDenied = "Turn notifications on in the phone's settings",
        hintFailed = "Could not reach the internet or the server",
    ),
    offline = Dict.Offline(
        title = "No internet",
        body = "The app could not reach the server. Check mobile data or Wi-Fi — the " +
            "screen opens by itself as soon as the connection is back.",
        retrying = { n -> if (n == 0) "Checking…" else "Checked " + n + " times" },
    ),
    common = Dict.Common(
        retry = "Try again",
        loading = "Loading…",
        cancel = "Cancel",
        close = "Close",
        ok = "Got it",
    ),
)

val DICTS: Map<Lang, Dict> = mapOf(Lang.Uz to UZ, Lang.Ru to RU, Lang.En to EN)
