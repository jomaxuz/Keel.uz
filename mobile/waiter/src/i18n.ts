// The app's own words, in the three languages a restaurant here is run in.
//
// ⚠️ **Its own dictionary rather than the panel's, and that was measured.**
// Importing `lib/i18n/admin` costs **500 KB** of bundle (1.6 MB → 2.1 MB): all
// three languages of every screen in the admin panel, to use about sixty
// strings. The shared thing worth sharing is the *rules* — what a service
// charge is, what a marking code may be, when a clock is untrustworthy — and
// those are imported unchanged. Wording for screens that exist only here is
// not shared meaning; it is this app's own text.
//
// ⚠️ **Where a word names a meaning defined elsewhere it is not redefined
// here.** The day statuses are the example: their colours come from
// `lib/attendance.ts`, so a shift that is amber in the panel cannot be green on
// a phone. Only the label is local.
//
// Uzbek is the source of truth: `Dict` is derived from it, so a key missing in
// ru or en is a compile error rather than a screen that falls back to Uzbek in
// front of a Russian-speaking waiter.

export const uz = {
  lang: "O'zbekcha",
  appName: "Keel Waiter",

  server: {
    title: "Restoran manzili",
    hint: "Restoraningizning qisqa nomi yoki to'liq manzili",
    placeholder: "osh",
    next: "Davom etish",
    bad: "Bu manzilga o'xshamaydi",
  },

  login: {
    title: "Kirish",
    username: "Login",
    password: "Parol",
    submit: "Kirish",
    other: "Boshqa restoran",
    failed: "Kirib bo'lmadi",
  },

  tabs: { floor: "Zal", profile: "Profil", settings: "Sozlamalar" },

  floor: {
    title: "Zal",
    empty: "Bu filialda stol qo'shilmagan",
    free: "bo'sh",
    unzoned: "Zal",
    failedOpen: "Ochib bo'lmadi",
    failedLoad: "Yuklab bo'lmadi",
  },

  check: {
    back: "Zal",
    tab: "Chek",
    menu: "Menyu",
    empty: "Chek bo'sh — menyudan tanlang",
    noItems: "Bu bo'limda taom yo'q",
    pending: "yuborilmagan",
    fire: (n: number) => `Oshxonaga yuborish (${n})`,
    failedAdd: "Qo'shib bo'lmadi",
    failedFire: "Yuborib bo'lmadi",
    firedOnly: "Bu taom allaqachon oshxonaga ketgan — chekdan hisobdan chiqaring",
    heldTitle: (who: string): string => `${who} shu stolda ishlayapti`,
    heldBody:
      "Ko'rishingiz mumkin, lekin o'zgartirsangiz uning ishi yo'qolishi mumkin. Avval u bilan gaplashing.",
    table: (n: string) => `${n}-stol`,
  },

  menu: {
    search: "Taom qidirish",
    onCheck: "Chekdagilar",
    found: (n: number) => `${n} ta topildi`,
    nothingFound: "Hech nima topilmadi",
    view: "Ko'rinish",
  },

  line: {
    fired: "oshxonaga yuborilgan",
    save: "Saqlash",
    remove: "Olib tashlash",
    writeOff: "Hisobdan chiqarish",
    commentPlaceholder: "Izoh: piyozsiz, achchiq…",
    reasonPlaceholder: "Sabab (majburiy)",
    pinPlaceholder: "Menejer kodi",
    failed: "Bajarilmadi",
  },

  bill: {
    print: "Hisob",
    printed: (n: number): string => "Hisob chop etildi",
    notQueued: "Printerga yuborilmadi",
    notQueuedHint:
      "Bu filialda hisobni oladigan printer topilmadi. Hisobni kassadan chiqaring.",
    failed: "Hisobni chiqarib bo'lmadi",
  },

  table: {
    guests: "Mehmonlar soni",
    split: "Chekni bo'lish",
    merge: "Stollarni birlashtirish",
    move: "Taomni ko'chirish",
    guestsHint: "Stolda necha kishi o'tiribdi",
    splitHint: "Alohida to'laydigan taomlarni belgilang",
    moveHint: "Ko'chiriladigan taomlarni belgilang",
    pickTable: "Qaysi stolga",
    noOthers: "Boshqa ochiq stol yo'q",
    splitDo: (n: number): string => `${n} ta taomni ajratish`,
    failed: "Bajarilmadi",
    actions: "Amallar",
  },

  profile: {
    title: "Profil",
    today: "Bugun",
    week: "Hafta",
    month: "Oy",
    worked: "Ishlangan",
    expected: "Grafik bo'yicha",
    diff: "Farq",
    onShift: "Smena ochiq",
    calendar: "Kalendar",
    // ⚠️ Read beside coloured squares, so each one has to be a phrase somebody
    // can act on rather than a category name.
    status: {
      ok: "Grafik bo'yicha",
      over: "Ko'p ishlangan",
      under: "Kam ishlangan",
      absent: "Chiqmagan",
      extra: "Grafiksiz kun",
      open: "Hozir ishlayapti",
      off: "Dam olish",
      upcoming: "Hali kelmagan",
    },
    hour: "soat",
    minute: "daq",
    noData: "Bu davrda yozuv yo'q",
  },

  notice: { ok: "Tushunarli" },

  clock: {
    in: "Smenani boshlash",
    out: "Smenani yakunlash",
    needLocation:
      "Joylashuvga ruxsat berilmagan. Telefon sozlamalaridan yoqing — smena qayerdan ochilgani yoziladi.",
    failed: "Bajarilmadi",
  },

  settings: {
    title: "Sozlamalar",
    language: "Til",
    theme: "Ko'rinish",
    themeSystem: "Tizim bo'yicha",
    themeLight: "Yorug'",
    themeDark: "Qorong'i",
    account: "Hisob",
    notifications: "Bildirishnomalar",
    push: {
      working: "Yoqilgan",
      asking: "Tekshirilmoqda…",
      denied: "Ruxsat berilmagan",
      noDevice: "Emulyatorda ishlamaydi",
      noProject: "Ilova sozlamasi to'liq emas",
      failed: "Ro'yxatdan o'tmadi",
    },
    pushHint: {
      working: "Oshxona taom tayyor deganda xabar keladi",
      asking: "Bir soniya",
      denied: "Telefon sozlamalaridan bildirishnomalarni yoqing",
      noDevice: "Haqiqiy telefonda sinang",
      noProject: "Bu build eski — yangi versiyani o'rnating",
      failed: "Internet yoki server bilan bog'lanib bo'lmadi",
    },
    restaurant: "Restoran",
    branch: "Filial",
    signOut: "Chiqish",
    changeServer: "Boshqa restoran",
    // ⚠️ Says what it does, because the two are not the same action: a shift
    // ends every day, a phone changes jobs once.
    changeServerHint: "Chiqish va boshqa restoran manzilini kiritish",
    version: "Versiya",
  },

  common: {
    retry: "Qayta urinish",
    loading: "Yuklanmoqda…",
  },
};

export type Dict = typeof uz;

export const ru: Dict = {
  lang: "Русский",
  appName: "Keel Waiter",

  server: {
    title: "Адрес ресторана",
    hint: "Короткое название вашего ресторана или полный адрес",
    placeholder: "osh",
    next: "Продолжить",
    bad: "Это не похоже на адрес",
  },

  login: {
    title: "Вход",
    username: "Логин",
    password: "Пароль",
    submit: "Войти",
    other: "Другой ресторан",
    failed: "Не удалось войти",
  },

  tabs: { floor: "Зал", profile: "Профиль", settings: "Настройки" },

  floor: {
    title: "Зал",
    empty: "В этом филиале столы не добавлены",
    free: "свободен",
    unzoned: "Зал",
    failedOpen: "Не удалось открыть",
    failedLoad: "Не удалось загрузить",
  },

  check: {
    back: "Зал",
    tab: "Чек",
    menu: "Меню",
    empty: "Чек пуст — выберите из меню",
    noItems: "В этом разделе нет блюд",
    pending: "не отправлено",
    fire: (n: number) => `Отправить на кухню (${n})`,
    failedAdd: "Не удалось добавить",
    failedFire: "Не удалось отправить",
    firedOnly: "Это блюдо уже ушло на кухню — спишите его в чеке",
    heldTitle: (who: string): string => `${who} сейчас работает с этим столом`,
    heldBody:
      "Смотреть можно, но при изменении его работа может пропасть. Сначала поговорите с ним.",
    table: (n: string) => `Стол ${n}`,
  },

  menu: {
    search: "Поиск блюда",
    onCheck: "В чеке",
    found: (n: number) => `Найдено: ${n}`,
    nothingFound: "Ничего не найдено",
    view: "Вид",
  },

  line: {
    fired: "отправлено на кухню",
    save: "Сохранить",
    remove: "Убрать",
    writeOff: "Списать",
    commentPlaceholder: "Комментарий: без лука, острое…",
    reasonPlaceholder: "Причина (обязательно)",
    pinPlaceholder: "Код менеджера",
    failed: "Не выполнено",
  },

  bill: {
    print: "Счёт",
    printed: (n: number): string => "Счёт напечатан",
    notQueued: "На принтер не ушло",
    notQueuedHint:
      "В этом филиале не нашлось принтера для счёта. Распечатайте счёт на кассе.",
    failed: "Не удалось напечатать счёт",
  },

  table: {
    guests: "Количество гостей",
    split: "Разделить чек",
    merge: "Объединить столы",
    move: "Перенести блюдо",
    guestsHint: "Сколько человек за столом",
    splitHint: "Отметьте блюда, за которые платят отдельно",
    moveHint: "Отметьте блюда для переноса",
    pickTable: "На какой стол",
    noOthers: "Других открытых столов нет",
    splitDo: (n: number): string => `Отделить ${n} блюд`,
    failed: "Не выполнено",
    actions: "Действия",
  },

  profile: {
    title: "Профиль",
    today: "Сегодня",
    week: "Неделя",
    month: "Месяц",
    worked: "Отработано",
    expected: "По графику",
    diff: "Разница",
    onShift: "Смена открыта",
    calendar: "Календарь",
    status: {
      ok: "По графику",
      over: "Переработка",
      under: "Недоработка",
      absent: "Не вышел",
      extra: "День вне графика",
      open: "Работает сейчас",
      off: "Выходной",
      upcoming: "Ещё не наступил",
    },
    hour: "ч",
    minute: "мин",
    noData: "За этот период записей нет",
  },

  notice: { ok: "Понятно" },

  clock: {
    in: "Начать смену",
    out: "Закончить смену",
    needLocation:
      "Доступ к геолокации не разрешён. Включите его в настройках телефона — фиксируется, откуда открыта смена.",
    failed: "Не выполнено",
  },

  settings: {
    title: "Настройки",
    language: "Язык",
    theme: "Оформление",
    themeSystem: "Как в системе",
    themeLight: "Светлое",
    themeDark: "Тёмное",
    account: "Аккаунт",
    notifications: "Уведомления",
    push: {
      working: "Включены",
      asking: "Проверяем…",
      denied: "Доступ не разрешён",
      noDevice: "На эмуляторе не работает",
      noProject: "Настройка приложения неполная",
      failed: "Не зарегистрировано",
    },
    pushHint: {
      working: "Придёт сообщение, когда кухня отметит блюдо готовым",
      asking: "Секунду",
      denied: "Включите уведомления в настройках телефона",
      noDevice: "Попробуйте на настоящем телефоне",
      noProject: "Эта сборка устарела — установите новую",
      failed: "Не удалось связаться с интернетом или сервером",
    },
    restaurant: "Ресторан",
    branch: "Филиал",
    signOut: "Выйти",
    changeServer: "Другой ресторан",
    changeServerHint: "Выйти и ввести адрес другого ресторана",
    version: "Версия",
  },

  common: {
    retry: "Повторить",
    loading: "Загрузка…",
  },
};

export const en: Dict = {
  lang: "English",
  appName: "Keel Waiter",

  server: {
    title: "Restaurant address",
    hint: "Your restaurant's short name, or its full address",
    placeholder: "osh",
    next: "Continue",
    bad: "That does not look like an address",
  },

  login: {
    title: "Sign in",
    username: "Login",
    password: "Password",
    submit: "Sign in",
    other: "Another restaurant",
    failed: "Could not sign in",
  },

  tabs: { floor: "Floor", profile: "Profile", settings: "Settings" },

  floor: {
    title: "Floor",
    empty: "No tables in this branch yet",
    free: "free",
    unzoned: "The room",
    failedOpen: "Could not open",
    failedLoad: "Could not load",
  },

  check: {
    back: "Floor",
    tab: "Check",
    menu: "Menu",
    empty: "Nothing on the check — pick from the menu",
    noItems: "Nothing in this section",
    pending: "not sent",
    fire: (n: number) => `Send to the kitchen (${n})`,
    failedAdd: "Could not add",
    failedFire: "Could not send",
    firedOnly: "That dish has already gone to the kitchen — write it off on the check",
    heldTitle: (who: string): string => `${who} is on this table`,
    heldBody:
      "You can look, but changing it may lose their work. Speak to them first.",
    table: (n: string) => `Table ${n}`,
  },

  menu: {
    search: "Search a dish",
    onCheck: "On the check",
    found: (n: number) => `${n} found`,
    nothingFound: "Nothing found",
    view: "View",
  },

  line: {
    fired: "sent to the kitchen",
    save: "Save",
    remove: "Remove",
    writeOff: "Write off",
    commentPlaceholder: "Note: no onion, extra spicy…",
    reasonPlaceholder: "Reason (required)",
    pinPlaceholder: "Manager's code",
    failed: "Did not go through",
  },

  bill: {
    print: "Bill",
    printed: (n: number): string => "The bill is printing",
    notQueued: "No printer took it",
    notQueuedHint:
      "No printer in this branch accepted the bill. Print it from the till.",
    failed: "Could not print the bill",
  },

  table: {
    guests: "How many guests",
    split: "Split the check",
    merge: "Merge tables",
    move: "Move dishes",
    guestsHint: "How many people are at the table",
    splitHint: "Tick the dishes being paid for separately",
    moveHint: "Tick the dishes to move",
    pickTable: "Onto which table",
    noOthers: "No other table is open",
    splitDo: (n: number): string => `Split off ${n} dishes`,
    failed: "Did not go through",
    actions: "Actions",
  },

  profile: {
    title: "Profile",
    today: "Today",
    week: "Week",
    month: "Month",
    worked: "Worked",
    expected: "Rostered",
    diff: "Difference",
    onShift: "On shift",
    calendar: "Calendar",
    status: {
      ok: "As rostered",
      over: "Overtime",
      under: "Short",
      absent: "Absent",
      extra: "Unrostered day",
      open: "Working now",
      off: "Day off",
      upcoming: "Not yet",
    },
    hour: "h",
    minute: "m",
    noData: "Nothing recorded in this period",
  },

  notice: { ok: "Got it" },

  clock: {
    in: "Start the shift",
    out: "End the shift",
    needLocation:
      "Location is not allowed. Turn it on in the phone's settings — where a shift was opened is recorded.",
    failed: "Did not go through",
  },

  settings: {
    title: "Settings",
    language: "Language",
    theme: "Appearance",
    themeSystem: "Follow the system",
    themeLight: "Light",
    themeDark: "Dark",
    account: "Account",
    notifications: "Notifications",
    push: {
      working: "On",
      asking: "Checking…",
      denied: "Not allowed",
      noDevice: "Not available on a simulator",
      noProject: "The app's configuration is incomplete",
      failed: "Not registered",
    },
    pushHint: {
      working: "You are told when the kitchen marks a dish ready",
      asking: "One moment",
      denied: "Turn notifications on in the phone's settings",
      noDevice: "Try it on a real phone",
      noProject: "This build is out of date — install the newer one",
      failed: "Could not reach the internet or the server",
    },
    restaurant: "Restaurant",
    branch: "Branch",
    signOut: "Sign out",
    changeServer: "Another restaurant",
    changeServerHint: "Sign out and enter another restaurant's address",
    version: "Version",
  },

  common: {
    retry: "Try again",
    loading: "Loading…",
  },
};

export type Lang = "uz" | "ru" | "en";

export const DICTS: Record<Lang, Dict> = { uz, ru, en };
export const LANGS: Lang[] = ["uz", "ru", "en"];
