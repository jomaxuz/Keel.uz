package uz.keel.app.i18n

import uz.keel.design.Lang

// Keel's own words: getting in, choosing a workspace, switching.
//
// ⚠️ **Only what the shell says.** Every role keeps its own dictionary, read
// through its own `t`, and none of those words are repeated here — a second
// copy of "Settings" is a copy that drifts the first time one is reworded.
//
// ⚠️ **Three languages from the first line**, like every Keel app. The first
// screen of this app is the one a Russian-speaking courier meets before anybody
// can help them, and a sign-in only in Uzbek is a sign-in they cannot pass.

data class Words(
    val lang: String,
    val tagline: String,
    val signIn: SignIn,
    val hub: Hub,
    val owner: Space,
    val waiter: Space,
    val courier: Space,
    val team: Space,
    val kinds: Kinds,
    val notes: Notes,
    val retry: String,
    val loading: String,
) {
    data class SignIn(
        val title: String,
        val lead: String,
        val restaurant: String,
        val restaurantHint: String,
        val username: String,
        val password: String,
        val submit: String,
        val badAddress: String,
        val empty: String,
        val wrong: String,
        val offline: String,
        val addTitle: String,
        val addLead: String,
        val back: String,
        val footnote: String,
    )

    data class Hub(
        val morning: String,
        val day: String,
        val evening: String,
        val night: String,
        val question: String,
        val last: String,
        val accounts: String,
        val signOut: String,
        val add: String,
        val leave: String,
        val leaveConfirm: String,
        val leaveYes: String,
        val cancel: String,
        val switch: String,
        val switchHint: String,
    )

    /** One workspace: what it is called and the one line under it. */
    data class Space(val title: String, val lead: String)

    data class Kinds(
        val owner: String,
        val manager: String,
        val staff: String,
        val courier: String,
    )

    data class Notes(
        val expired: String,
        val expiredBody: String,
        val refused: (String, String) -> String,
        val signedOut: String,
    )
}

val UZ = Words(
    lang = "O'zbekcha",
    tagline = "Restoraningiz — bitta ilovada",
    signIn = Words.SignIn(
        title = "Xush kelibsiz",
        lead = "Login va parolingizni kiriting — qaysi ish joyingiz ekanini Keel o'zi aniqlaydi.",
        restaurant = "Restoran",
        restaurantHint = "masalan: osh-markazi",
        username = "Login",
        password = "Parol",
        submit = "Kirish",
        badAddress = "Restoran manzili noto'g'ri — nomini yozing, masalan «osh-markazi»",
        empty = "Login va parolni kiriting",
        wrong = "Login yoki parol noto'g'ri",
        offline = "Internet yo'q yoki restoran serveri javob bermadi — qayta urinib ko'ring",
        addTitle = "Hisob qo'shish",
        addLead = "Shu restoranning boshqa logini — masalan, kuryer yoki ega hisobi.",
        back = "Orqaga",
        footnote = "Ofitsiant, ega, kuryer va xodimlar uchun bitta ilova",
    ),
    hub = Words.Hub(
        morning = "Xayrli tong",
        day = "Xayrli kun",
        evening = "Xayrli kech",
        night = "Xayrli tun",
        question = "Bugun qayerda ishlaymiz?",
        last = "Oxirgi",
        accounts = "Bu telefondagi hisoblar",
        signOut = "Chiqish",
        add = "Hisob qo'shish",
        leave = "Boshqa restoran",
        leaveConfirm = "Barcha hisoblardan chiqib, restoran manzilini o'chiramizmi?",
        leaveYes = "Ha, chiqish",
        cancel = "Bekor qilish",
        switch = "Almashtirish",
        switchHint = "Ish joylari va hisoblar",
    ),
    owner = Words.Space("Boshqaruv", "Bugungi tushum, diqqat, buyurtmalar va hisobotlar"),
    waiter = Words.Space("Zal", "Stollar, cheklar va oshxonadan «tayyor» xabari"),
    courier = Words.Space("Yetkazish", "Smena, buyurtmalar va daromad"),
    team = Words.Space("Xodim", "Davomat, ish haqi, bozorlik va ombor"),
    kinds = Words.Kinds(owner = "Ega", manager = "Menejer", staff = "Xodim", courier = "Kuryer"),
    notes = Words.Notes(
        expired = "Sessiya tugadi",
        expiredBody = "Qaytadan kiring — hisob o'chirilgan yoki muddati o'tgan bo'lishi mumkin.",
        refused = { who, why -> "$who hisobi ochilmadi: $why" },
        signedOut = "Hisobdan chiqildi",
    ),
    retry = "Qayta urinish",
    loading = "Yuklanmoqda…",
)

val RU = Words(
    lang = "Русский",
    tagline = "Ваш ресторан — в одном приложении",
    signIn = Words.SignIn(
        title = "Добро пожаловать",
        lead = "Введите логин и пароль — Keel сам определит, где вы работаете.",
        restaurant = "Ресторан",
        restaurantHint = "например: osh-markazi",
        username = "Логин",
        password = "Пароль",
        submit = "Войти",
        badAddress = "Неверный адрес ресторана — введите название, например «osh-markazi»",
        empty = "Введите логин и пароль",
        wrong = "Неверный логин или пароль",
        offline = "Нет интернета или сервер ресторана не ответил — попробуйте ещё раз",
        addTitle = "Добавить аккаунт",
        addLead = "Другой логин этого ресторана — например, курьера или владельца.",
        back = "Назад",
        footnote = "Одно приложение для официантов, владельца, курьеров и сотрудников",
    ),
    hub = Words.Hub(
        morning = "Доброе утро",
        day = "Добрый день",
        evening = "Добрый вечер",
        night = "Доброй ночи",
        question = "Где работаем сегодня?",
        last = "Последнее",
        accounts = "Аккаунты на этом телефоне",
        signOut = "Выйти",
        add = "Добавить аккаунт",
        leave = "Другой ресторан",
        leaveConfirm = "Выйти из всех аккаунтов и забыть адрес ресторана?",
        leaveYes = "Да, выйти",
        cancel = "Отмена",
        switch = "Сменить",
        switchHint = "Рабочие места и аккаунты",
    ),
    owner = Words.Space("Управление", "Выручка за сегодня, внимание, заказы и отчёты"),
    waiter = Words.Space("Зал", "Столы, чеки и «готово» с кухни"),
    courier = Words.Space("Доставка", "Смена, заказы и заработок"),
    team = Words.Space("Сотрудник", "Посещаемость, зарплата, закупки и склад"),
    kinds = Words.Kinds(owner = "Владелец", manager = "Менеджер", staff = "Сотрудник", courier = "Курьер"),
    notes = Words.Notes(
        expired = "Сессия завершена",
        expiredBody = "Войдите снова — аккаунт мог быть отключён или срок входа истёк.",
        refused = { who, why -> "Аккаунт «$who» не открыт: $why" },
        signedOut = "Вы вышли из аккаунта",
    ),
    retry = "Повторить",
    loading = "Загрузка…",
)

val EN = Words(
    lang = "English",
    tagline = "Your restaurant — in one app",
    signIn = Words.SignIn(
        title = "Welcome",
        lead = "Enter your login and password — Keel works out where you work.",
        restaurant = "Restaurant",
        restaurantHint = "e.g. osh-markazi",
        username = "Login",
        password = "Password",
        submit = "Sign in",
        badAddress = "That restaurant address is not valid — type its name, e.g. “osh-markazi”",
        empty = "Enter your login and password",
        wrong = "Wrong login or password",
        offline = "No internet, or the restaurant's server did not answer — try again",
        addTitle = "Add an account",
        addLead = "Another login at this restaurant — a courier's or the owner's.",
        back = "Back",
        footnote = "One app for waiters, owners, couriers and staff",
    ),
    hub = Words.Hub(
        morning = "Good morning",
        day = "Good afternoon",
        evening = "Good evening",
        night = "Good night",
        question = "Where are we working today?",
        last = "Last",
        accounts = "Accounts on this phone",
        signOut = "Sign out",
        add = "Add an account",
        leave = "Another restaurant",
        leaveConfirm = "Sign out of every account and forget the restaurant's address?",
        leaveYes = "Yes, sign out",
        cancel = "Cancel",
        switch = "Switch",
        switchHint = "Workspaces and accounts",
    ),
    owner = Words.Space("Management", "Today's takings, alerts, orders and reports"),
    waiter = Words.Space("Floor", "Tables, checks and “ready” from the kitchen"),
    courier = Words.Space("Delivery", "Shift, orders and earnings"),
    team = Words.Space("Team", "Attendance, pay, market runs and the store"),
    kinds = Words.Kinds(owner = "Owner", manager = "Manager", staff = "Staff", courier = "Courier"),
    notes = Words.Notes(
        expired = "Session ended",
        expiredBody = "Sign in again — the account may have been switched off or the sign-in expired.",
        refused = { who, why -> "The $who account did not open: $why" },
        signedOut = "Signed out",
    ),
    retry = "Retry",
    loading = "Loading…",
)

val WORDS: Map<Lang, Words> = mapOf(Lang.Uz to UZ, Lang.Ru to RU, Lang.En to EN)
