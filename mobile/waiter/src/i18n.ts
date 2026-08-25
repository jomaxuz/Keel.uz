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
    table: (n: string) => `${n}-stol`,
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

  settings: {
    title: "Sozlamalar",
    language: "Til",
    theme: "Ko'rinish",
    themeSystem: "Tizim bo'yicha",
    themeLight: "Yorug'",
    themeDark: "Qorong'i",
    account: "Hisob",
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
    table: (n: string) => `Стол ${n}`,
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

  settings: {
    title: "Настройки",
    language: "Язык",
    theme: "Оформление",
    themeSystem: "Как в системе",
    themeLight: "Светлое",
    themeDark: "Тёмное",
    account: "Аккаунт",
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
    table: (n: string) => `Table ${n}`,
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

  settings: {
    title: "Settings",
    language: "Language",
    theme: "Appearance",
    themeSystem: "Follow the system",
    themeLight: "Light",
    themeDark: "Dark",
    account: "Account",
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
