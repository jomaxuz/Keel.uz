package uz.keel.waiter.i18n

// The app's own words, in the three languages a restaurant here is run in.
//
// ⚠️ **Its own dictionary rather than the panel's, and that was measured on the
// Expo build.** Importing the admin dictionary cost 500 KB of bundle — all three
// languages of every screen in the panel, to use about sixty strings. The thing
// worth sharing is the *rules*; wording for screens that exist only here is this
// app's own text.
//
// ⚠️ **Where a word names a meaning defined elsewhere it is not redefined.** The
// day statuses are the example: their colours come from `StatusColor`, which is
// `lib/attendance.ts` ported deliberately, so a shift that is amber in the panel
// cannot be green on a phone. Only the label is local.
//
// ⚠️ **Data classes with named arguments, not a `Map<String, String>`.** A map
// makes a missing key a blank line in front of a Russian-speaking waiter; this
// makes it a compile error. That is exactly the property the TypeScript version
// had (`Dict = typeof uz`) and it is the reason this file is verbose.

enum class Lang(val code: String) {
    Uz("uz"), Ru("ru"), En("en");

    companion object {
        fun of(code: String?): Lang = entries.firstOrNull { it.code == code } ?: Uz
    }
}

data class Dict(
    val lang: String,
    val appName: String,
    val server: Server,
    val login: Login,
    val tabs: Tabs,
    val floor: Floor,
    val check: Check,
    val menu: Menu,
    val line: Line,
    val bill: Bill,
    val table: Table,
    val profile: Profile,
    val offline: Offline,
    val notice: Notice,
    val clock: Clock,
    val settings: Settings,
    val outbox: Outbox,
    val common: Common,
) {
    data class Server(val title: String, val hint: String, val placeholder: String, val next: String, val bad: String)
    data class Login(val title: String, val username: String, val password: String, val submit: String, val other: String, val failed: String)
    data class Tabs(val floor: String, val profile: String, val settings: String)
    data class Floor(val title: String, val empty: String, val free: String, val unzoned: String, val failedOpen: String, val failedLoad: String)
    data class Check(
        val back: String, val tab: String, val menu: String, val empty: String, val noItems: String,
        val pending: String, val portionAsk: String, val portionWhole: String, val portionCancel: String,
        val readyAgo: (String) -> String, val servedAgo: (String) -> String,
        val serve: String, val unserve: String, val fire: (Int) -> String,
        val failedAdd: String, val failedFire: String, val firedOnly: String,
        val heldTitle: (String) -> String, val heldBody: String, val table: (String) -> String,
    )
    data class Menu(val search: String, val onCheck: String, val found: (Int) -> String, val nothingFound: String, val view: String)
    data class Line(val fired: String, val save: String, val remove: String, val writeOff: String, val commentPlaceholder: String, val reasonPlaceholder: String, val pinPlaceholder: String, val failed: String)
    data class Bill(val print: String, val printed: String, val notQueued: String, val notQueuedHint: String, val tillOff: String, val tillOffHint: String, val failed: String)
    data class Table(
        val guests: String, val split: String, val merge: String, val move: String,
        val guestsHint: String, val splitHint: String, val moveHint: String, val pickTable: String,
        val noOthers: String, val splitDo: (Int) -> String, val failed: String, val actions: String,
    )
    data class Profile(
        val title: String, val today: String, val week: String, val month: String,
        val worked: String, val expected: String, val diff: String, val onShift: String,
        val calendar: String, val status: Map<String, String>,
        val hour: String, val minute: String, val noData: String,
    )
    data class Offline(val title: String, val body: String, val retrying: (Int) -> String)
    data class Notice(val ok: String)
    data class Clock(val start: String, val end: String, val needLocation: String, val failed: String)
    data class Settings(
        val title: String, val language: String, val theme: String,
        val themeSystem: String, val themeLight: String, val themeDark: String,
        val account: String, val notifications: String,
        val push: Map<String, String>, val pushHint: Map<String, String>,
        val restaurant: String, val branch: String, val signOut: String,
        val changeServer: String, val changeServerHint: String, val version: String,
    )
    /** What the phone is holding because it could not send it.
     *
     *  ⚠️ **Said, not hidden.** An app holding four dishes it has not managed to
     *  send looks exactly like an app that sent them, and the difference reaches
     *  the guest. It is also not an error: nothing was lost and nothing needs
     *  retyping, which is the opposite of what "qo'shib bo'lmadi" told people. */
    data class Outbox(val queued: String, val waiting: (Int) -> String)
    data class Common(val retry: String, val loading: String, val timeAgo: TimeAgo)
    data class TimeAgo(val now: String, val min: (Int) -> String, val hour: (Int) -> String, val day: (Int) -> String)
}

val UZ = Dict(
    lang = "O'zbekcha",
    appName = "Keel Waiter",
    server = Dict.Server(
        title = "Restoran manzili",
        hint = "Restoraningizning qisqa nomi yoki to'liq manzili",
        placeholder = "osh", next = "Davom etish", bad = "Bu manzilga o'xshamaydi",
    ),
    login = Dict.Login("Kirish", "Login", "Parol", "Kirish", "Boshqa restoran", "Kirib bo'lmadi"),
    tabs = Dict.Tabs("Zal", "Profil", "Sozlamalar"),
    floor = Dict.Floor("Zal", "Bu filialda stol qo'shilmagan", "bo'sh", "Zal", "Ochib bo'lmadi", "Yuklab bo'lmadi"),
    check = Dict.Check(
        back = "Zal", tab = "Chek", menu = "Menyu",
        empty = "Chek bo'sh — menyudan tanlang", noItems = "Bu bo'limda taom yo'q",
        pending = "yuborilmagan",
        portionAsk = "Qancha sotiladi?", portionWhole = "1 porsiya", portionCancel = "Bekor qilish",
        readyAgo = { "$it tayyor bo'ldi" }, servedAgo = { "$it berildi" },
        serve = "Berildi", unserve = "Bekor qilish",
        fire = { "Oshxonaga yuborish ($it)" },
        failedAdd = "Qo'shib bo'lmadi", failedFire = "Yuborib bo'lmadi",
        firedOnly = "Bu taom allaqachon oshxonaga ketgan — chekdan hisobdan chiqaring",
        heldTitle = { "$it shu stolda ishlayapti" },
        heldBody = "Ko'rishingiz mumkin, lekin o'zgartirsangiz uning ishi yo'qolishi mumkin. Avval u bilan gaplashing.",
        table = { "$it-stol" },
    ),
    menu = Dict.Menu("Taom qidirish", "Chekdagilar", { "$it ta topildi" }, "Hech nima topilmadi", "Ko'rinish"),
    line = Dict.Line(
        "oshxonaga yuborilgan", "Saqlash", "Olib tashlash", "Hisobdan chiqarish",
        "Izoh: piyozsiz, achchiq…", "Sabab (majburiy)", "Menejer kodi", "Bajarilmadi",
    ),
    bill = Dict.Bill(
        print = "Hisob", printed = "Hisob chop etildi", notQueued = "Printerga yuborilmadi",
        notQueuedHint = "Bu filialda hisobni oladigan printer topilmadi. Hisobni kassadan chiqaring.",
        tillOff = "Kassa yoqilmagan",
        tillOffHint = "Monoblokdagi Keel kassa dasturini oching — chek o'sha orqali chiqadi. Navbatga qo'yilmadi.",
        failed = "Hisobni chiqarib bo'lmadi",
    ),
    table = Dict.Table(
        guests = "Mehmonlar soni", split = "Chekni bo'lish", merge = "Stollarni birlashtirish",
        move = "Taomni ko'chirish", guestsHint = "Stolda necha kishi o'tiribdi",
        splitHint = "Alohida to'laydigan taomlarni belgilang", moveHint = "Ko'chiriladigan taomlarni belgilang",
        pickTable = "Qaysi stolga", noOthers = "Boshqa ochiq stol yo'q",
        splitDo = { "$it ta taomni ajratish" }, failed = "Bajarilmadi", actions = "Amallar",
    ),
    profile = Dict.Profile(
        title = "Profil", today = "Bugun", week = "Hafta", month = "Oy",
        worked = "Ishlangan", expected = "Grafik bo'yicha", diff = "Farq",
        onShift = "Smena ochiq", calendar = "Kalendar",
        // ⚠️ Read beside coloured squares, so each is a phrase somebody can act
        // on rather than a category name.
        status = mapOf(
            "ok" to "Grafik bo'yicha", "over" to "Ko'p ishlangan", "under" to "Kam ishlangan",
            "absent" to "Chiqmagan", "extra" to "Grafiksiz kun", "open" to "Hozir ishlayapti",
            "off" to "Dam olish", "upcoming" to "Hali kelmagan",
        ),
        hour = "soat", minute = "daq", noData = "Bu davrda yozuv yo'q",
    ),
    // ⚠️ Kirish ekrani emas: internet yo'qligini parol bilan tuzatib bo'lmaydi.
    offline = Dict.Offline(
        title = "Internet yo'q",
        body = "Ilova serverga ulana olmadi. Wi-Fi yoki mobil internetni tekshiring — ulanish tiklanishi bilan ekran o'zi ochiladi.",
        retrying = { if (it == 0) "Tekshirilmoqda…" else "Qayta tekshirildi: $it" },
    ),
    notice = Dict.Notice("Tushunarli"),
    clock = Dict.Clock(
        start = "Smenani boshlash", end = "Smenani yakunlash",
        needLocation = "Joylashuvga ruxsat berilmagan. Telefon sozlamalaridan yoqing — smena qayerdan ochilgani yoziladi.",
        failed = "Bajarilmadi",
    ),
    settings = Dict.Settings(
        title = "Sozlamalar", language = "Til", theme = "Ko'rinish",
        themeSystem = "Tizim bo'yicha", themeLight = "Yorug'", themeDark = "Qorong'i",
        account = "Hisob", notifications = "Bildirishnomalar",
        push = mapOf(
            "working" to "Yoqilgan", "asking" to "Tekshirilmoqda…", "denied" to "Ruxsat berilmagan",
            "noDevice" to "Emulyatorda ishlamaydi", "noProject" to "Ilova sozlamasi to'liq emas",
            "failed" to "Ro'yxatdan o'tmadi",
        ),
        pushHint = mapOf(
            "working" to "Oshxona taom tayyor deganda xabar keladi", "asking" to "Bir soniya",
            "denied" to "Telefon sozlamalaridan bildirishnomalarni yoqing",
            "noDevice" to "Haqiqiy telefonda sinang",
            "noProject" to "Bu build eski — yangi versiyani o'rnating",
            "failed" to "Internet yoki server bilan bog'lanib bo'lmadi",
        ),
        restaurant = "Restoran", branch = "Filial", signOut = "Chiqish",
        changeServer = "Boshqa restoran",
        // ⚠️ Says what it does, because the two are not the same action: a shift
        // ends every day, a phone changes jobs once.
        changeServerHint = "Chiqish va boshqa restoran manzilini kiritish",
        version = "Versiya",
    ),
    outbox = Dict.Outbox(
        queued = "Internet yo'q — saqlandi, ulanish tiklanganda yuboriladi",
        waiting = { "$it ta amal yuborilmoqda…" },
    ),
    common = Dict.Common(
        retry = "Qayta urinish", loading = "Yuklanmoqda…",
        timeAgo = Dict.TimeAgo("hozir", { "$it daq oldin" }, { "$it soat oldin" }, { "$it kun oldin" }),
    ),
)

val RU = Dict(
    lang = "Русский",
    appName = "Keel Waiter",
    server = Dict.Server("Адрес ресторана", "Короткое название вашего ресторана или полный адрес", "osh", "Продолжить", "Это не похоже на адрес"),
    login = Dict.Login("Вход", "Логин", "Пароль", "Войти", "Другой ресторан", "Не удалось войти"),
    tabs = Dict.Tabs("Зал", "Профиль", "Настройки"),
    floor = Dict.Floor("Зал", "В этом филиале столы не добавлены", "свободен", "Зал", "Не удалось открыть", "Не удалось загрузить"),
    check = Dict.Check(
        back = "Зал", tab = "Чек", menu = "Меню",
        empty = "Чек пуст — выберите из меню", noItems = "В этом разделе нет блюд",
        pending = "не отправлено",
        portionAsk = "Сколько продаём?", portionWhole = "1 порция", portionCancel = "Отмена",
        readyAgo = { "готово $it" }, servedAgo = { "подано $it" },
        serve = "Подал", unserve = "Отменить",
        fire = { "Отправить на кухню ($it)" },
        failedAdd = "Не удалось добавить", failedFire = "Не удалось отправить",
        firedOnly = "Это блюдо уже ушло на кухню — спишите его в чеке",
        heldTitle = { "$it сейчас работает с этим столом" },
        heldBody = "Смотреть можно, но при изменении его работа может пропасть. Сначала поговорите с ним.",
        table = { "Стол $it" },
    ),
    menu = Dict.Menu("Поиск блюда", "В чеке", { "Найдено: $it" }, "Ничего не найдено", "Вид"),
    line = Dict.Line(
        "отправлено на кухню", "Сохранить", "Убрать", "Списать",
        "Комментарий: без лука, острое…", "Причина (обязательно)", "Код менеджера", "Не выполнено",
    ),
    bill = Dict.Bill(
        print = "Счёт", printed = "Счёт напечатан", notQueued = "На принтер не ушло",
        notQueuedHint = "В этом филиале не нашлось принтера для счёта. Распечатайте счёт на кассе.",
        tillOff = "Касса не включена",
        tillOffHint = "Откройте программу Keel на моноблоке — чек печатается через неё. В очередь не поставлено.",
        failed = "Не удалось напечатать счёт",
    ),
    table = Dict.Table(
        guests = "Количество гостей", split = "Разделить чек", merge = "Объединить столы",
        move = "Перенести блюдо", guestsHint = "Сколько человек за столом",
        splitHint = "Отметьте блюда, за которые платят отдельно", moveHint = "Отметьте блюда для переноса",
        pickTable = "На какой стол", noOthers = "Других открытых столов нет",
        splitDo = { "Отделить $it блюд" }, failed = "Не выполнено", actions = "Действия",
    ),
    profile = Dict.Profile(
        title = "Профиль", today = "Сегодня", week = "Неделя", month = "Месяц",
        worked = "Отработано", expected = "По графику", diff = "Разница",
        onShift = "Смена открыта", calendar = "Календарь",
        status = mapOf(
            "ok" to "По графику", "over" to "Переработка", "under" to "Недоработка",
            "absent" to "Не вышел", "extra" to "День вне графика", "open" to "Работает сейчас",
            "off" to "Выходной", "upcoming" to "Ещё не наступил",
        ),
        hour = "ч", minute = "мин", noData = "За этот период записей нет",
    ),
    offline = Dict.Offline(
        title = "Нет интернета",
        body = "Приложение не смогло связаться с сервером. Проверьте Wi-Fi или мобильный интернет — как только связь появится, экран откроется сам.",
        retrying = { if (it == 0) "Проверяем…" else "Проверок: $it" },
    ),
    notice = Dict.Notice("Понятно"),
    clock = Dict.Clock(
        start = "Начать смену", end = "Закончить смену",
        needLocation = "Доступ к геолокации не разрешён. Включите его в настройках телефона — фиксируется, откуда открыта смена.",
        failed = "Не выполнено",
    ),
    settings = Dict.Settings(
        title = "Настройки", language = "Язык", theme = "Оформление",
        themeSystem = "Как в системе", themeLight = "Светлое", themeDark = "Тёмное",
        account = "Аккаунт", notifications = "Уведомления",
        push = mapOf(
            "working" to "Включены", "asking" to "Проверяем…", "denied" to "Доступ не разрешён",
            "noDevice" to "На эмуляторе не работает", "noProject" to "Настройка приложения неполная",
            "failed" to "Не зарегистрировано",
        ),
        pushHint = mapOf(
            "working" to "Придёт сообщение, когда кухня отметит блюдо готовым", "asking" to "Секунду",
            "denied" to "Включите уведомления в настройках телефона",
            "noDevice" to "Попробуйте на настоящем телефоне",
            "noProject" to "Эта сборка устарела — установите новую",
            "failed" to "Не удалось связаться с интернетом или сервером",
        ),
        restaurant = "Ресторан", branch = "Филиал", signOut = "Выйти",
        changeServer = "Другой ресторан",
        changeServerHint = "Выйти и ввести адрес другого ресторана", version = "Версия",
    ),
    outbox = Dict.Outbox(
        queued = "Нет интернета — сохранено, отправим при появлении связи",
        waiting = { "Отправляется: $it" },
    ),
    common = Dict.Common(
        retry = "Повторить", loading = "Загрузка…",
        timeAgo = Dict.TimeAgo("только что", { "$it мин назад" }, { "$it ч назад" }, { "$it дн назад" }),
    ),
)

val EN = Dict(
    lang = "English",
    appName = "Keel Waiter",
    server = Dict.Server("Restaurant address", "Your restaurant's short name, or its full address", "osh", "Continue", "That does not look like an address"),
    login = Dict.Login("Sign in", "Login", "Password", "Sign in", "Another restaurant", "Could not sign in"),
    tabs = Dict.Tabs("Floor", "Profile", "Settings"),
    floor = Dict.Floor("Floor", "No tables in this branch yet", "free", "The room", "Could not open", "Could not load"),
    check = Dict.Check(
        back = "Floor", tab = "Check", menu = "Menu",
        empty = "Nothing on the check — pick from the menu", noItems = "Nothing in this section",
        pending = "not sent",
        portionAsk = "How much is being sold?", portionWhole = "Whole", portionCancel = "Cancel",
        readyAgo = { "ready $it" }, servedAgo = { "served $it" },
        serve = "Served it", unserve = "Undo",
        fire = { "Send to the kitchen ($it)" },
        failedAdd = "Could not add", failedFire = "Could not send",
        firedOnly = "That dish has already gone to the kitchen — write it off on the check",
        heldTitle = { "$it is on this table" },
        heldBody = "You can look, but changing it may lose their work. Speak to them first.",
        table = { "Table $it" },
    ),
    menu = Dict.Menu("Search a dish", "On the check", { "$it found" }, "Nothing found", "View"),
    line = Dict.Line(
        "sent to the kitchen", "Save", "Remove", "Write off",
        "Note: no onion, extra spicy…", "Reason (required)", "Manager's code", "Did not go through",
    ),
    bill = Dict.Bill(
        print = "Bill", printed = "The bill is printing", notQueued = "No printer took it",
        notQueuedHint = "No printer in this branch accepted the bill. Print it from the till.",
        tillOff = "The till is not running",
        tillOffHint = "Open the Keel till app on the monoblock — that is what prints. Nothing was queued.",
        failed = "Could not print the bill",
    ),
    table = Dict.Table(
        guests = "How many guests", split = "Split the check", merge = "Merge tables",
        move = "Move dishes", guestsHint = "How many people are at the table",
        splitHint = "Tick the dishes being paid for separately", moveHint = "Tick the dishes to move",
        pickTable = "Onto which table", noOthers = "No other table is open",
        splitDo = { "Split off $it dishes" }, failed = "Did not go through", actions = "Actions",
    ),
    profile = Dict.Profile(
        title = "Profile", today = "Today", week = "Week", month = "Month",
        worked = "Worked", expected = "Rostered", diff = "Difference",
        onShift = "On shift", calendar = "Calendar",
        status = mapOf(
            "ok" to "As rostered", "over" to "Overtime", "under" to "Short",
            "absent" to "Absent", "extra" to "Unrostered day", "open" to "Working now",
            "off" to "Day off", "upcoming" to "Not yet",
        ),
        hour = "h", minute = "m", noData = "Nothing recorded in this period",
    ),
    offline = Dict.Offline(
        title = "No internet",
        body = "The app could not reach the server. Check Wi-Fi or mobile data — the screen opens by itself as soon as the connection is back.",
        retrying = { if (it == 0) "Checking…" else "Checked $it times" },
    ),
    notice = Dict.Notice("Got it"),
    clock = Dict.Clock(
        start = "Start the shift", end = "End the shift",
        needLocation = "Location is not allowed. Turn it on in the phone's settings — where a shift was opened is recorded.",
        failed = "Did not go through",
    ),
    settings = Dict.Settings(
        title = "Settings", language = "Language", theme = "Appearance",
        themeSystem = "Follow the system", themeLight = "Light", themeDark = "Dark",
        account = "Account", notifications = "Notifications",
        push = mapOf(
            "working" to "On", "asking" to "Checking…", "denied" to "Not allowed",
            "noDevice" to "Not available on a simulator", "noProject" to "The app's configuration is incomplete",
            "failed" to "Not registered",
        ),
        pushHint = mapOf(
            "working" to "You are told when the kitchen marks a dish ready", "asking" to "One moment",
            "denied" to "Turn notifications on in the phone's settings",
            "noDevice" to "Try it on a real phone",
            "noProject" to "This build is out of date — install the newer one",
            "failed" to "Could not reach the internet or the server",
        ),
        restaurant = "Restaurant", branch = "Branch", signOut = "Sign out",
        changeServer = "Another restaurant",
        changeServerHint = "Sign out and enter another restaurant's address", version = "Version",
    ),
    outbox = Dict.Outbox(
        queued = "No internet — saved, it goes as soon as the connection is back",
        waiting = { "$it waiting to send…" },
    ),
    common = Dict.Common(
        retry = "Try again", loading = "Loading…",
        timeAgo = Dict.TimeAgo("just now", { "$it min ago" }, { "$it h ago" }, { "$it d ago" }),
    ),
)

val DICTS: Map<Lang, Dict> = mapOf(Lang.Uz to UZ, Lang.Ru to RU, Lang.En to EN)

/** Minutes as "8 soat 30 daq".
 *
 *  ⚠️ Ported from `lib/attendance.ts` → `formatDuration`, labels passed in so
 *  one helper serves all three languages — the same shape as the original. */
fun formatDuration(minutes: Int, hourLabel: String, minuteLabel: String): String {
    val sign = if (minutes < 0) "-" else ""
    val total = kotlin.math.abs(minutes)
    val h = total / 60
    val m = total % 60
    return when {
        h == 0 -> "$sign$m $minuteLabel"
        m == 0 -> "$sign$h $hourLabel"
        else -> "$sign$h $hourLabel $m $minuteLabel"
    }
}

/** How long ago an ISO instant was. Ported from `lib/orderFlow.ts` → `timeAgo`.
 *
 *  ⚠️ **Never `toLocalDate()` on a raw parse.** Times come off the wire in UTC —
 *  the trap this codebase has been bitten by on the server side too — and this
 *  works on the difference between two instants, which is timezone-free by
 *  construction. */
fun timeAgo(iso: String?, labels: Dict.TimeAgo): String {
    if (iso.isNullOrBlank()) return labels.now
    val then = runCatching { java.time.Instant.parse(iso).toEpochMilli() }.getOrNull() ?: return labels.now
    val min = Math.round((System.currentTimeMillis() - then) / 60000.0).toInt()
    if (min < 1) return labels.now
    if (min < 60) return labels.min(min)
    val h = Math.round(min / 60.0).toInt()
    if (h < 24) return labels.hour(h)
    return labels.day(Math.round(h / 24.0).toInt())
}
