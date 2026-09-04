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
  appName: "Keel Team",

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

  tabs: { profile: "Smena", settings: "Sozlamalar", buy: "Bozor", zakup: "Zakup" },

  // Zakup: bozorlikka nima kerakligini yozib yuborish.
  //
  // ⚠️ Bo'lim faqat menejer va omborchida ko'rinadi. Kassirda ruxsat bor, lekin
  // uning telefoni restoranning bozorligi rejalashtiriladigan joy emas — hech
  // qachon ishlatilmaydigan bo'lim ilovani e'tibordan qoldirishga o'rgatadi.
  zakup: {
    title: "Zakup",
    forDate: "Qaysi kunga",
    listTitle: "Ro'yxat",
    qty: "Miqdori",
    shortTitle: "Ombor bo'yicha kam qolganlar",
    search: "Masalliq nomi — yozing va tanlang",
    unitIs: (unit: string) => `o'lchovi: ${unit}`,
    nothingShort: "Hozircha hech narsa kam emas.",
    addNew: (name: string) => `«${name}» ni ro'yxatga qo'shish`,
    onHand: (n: number, unit: string) => `qoldiq ${n} ${unit}`,
    need: (n: number, unit: string) => `kerak ${n} ${unit}`,
    review: "Ko'rib chiqish",
    previewTitle: "Yuborishdan oldin",
    previewBody: (date: string) => `${date} kuniga bozorlik ro'yxati.`,
    back: "Orqaga",
    send: "Yuborish",
    sent: (n: number) => `Yuborildi: ${n} ta qator`,
    sentTitle: "Yuborilganlar",
    statusSent: "bozorda",
    statusDone: "yakunlandi",
    progress: (got: number, all: number) => `${got}/${all} olindi`,
    loadFailed: "Ro'yxatni ochib bo'lmadi",
    sendFailed: "Yuborib bo'lmadi",
  },

  // Bozorlik: nima olib kelingani shu yerda yoziladi va omborga tushadi.
  //
  // ⚠️ Matn savdo tilida: bu ekran do'kon peshtaxtasida, bir qo'lda paket bilan
  // o'qiladi. "Kirim hujjatini shakllantirish" — ofis tili, va uni o'qiydigan
  // odam bu yerda turmaydi.
  buy: {
    // ⚠️ Ro'yxatdan oldin: bu raqam safar bo'ladimi-yo'qmi degan savolga
    // javob beradi, va uni bilmagan odam ofisga qo'ng'iroq qilish o'rniga
    // taxmin qiladi.
    purse: "Qo'lingizdagi pul",
    purseOwed: "Restoran sizga qarzdor",
    purseOf: (issued: string, spent: string) =>
      `${issued} berilgan · ${spent} sarflangan`,
    orderTitle: (d: string) => `Bozorlik ro'yxati — ${d}`,
    orderFrom: (who: string) => `${who} yubordi`,
    asked: (n: number, u: string) => `so'ralgan: ${n} ${u}`,
    got: "Olindi",
    changed: "O'zgartirish",
    // ⚠️ Nol miqdor emas, alohida javob: hech kim tegmagan qator bilan
    // qidirib topilmagan qator — ikki xil fakt.
    noneLeft: "Yo'q edi",
    wasMissing: "Bozorda yo'q edi",
    undo: "Bekor qilish",
    finish: "Yakunlash va omborga kiritish",
    shortTitle: "Nima kam qolgan",
    nothingShort: "Hozircha hech narsa kam emas.",
    basketTitle: "Olinganlar",
    addTitle: "Yana qo'shish",
    searchPlaceholder: "Masalliq nomi",
    addNew: (name: string) => `«${name}» ni yangi masalliq sifatida qo'shish`,
    onHand: (n: number, unit: string) => `qoldiq ${n} ${unit}`,
    need: (n: number, unit: string) => `kerak ${n} ${unit}`,
    qty: (unit: string) => (unit ? `Miqdori (${unit})` : "Miqdori"),
    price: "Narxi",
    // ⚠️ Yagona qo'riq: telefonda yozilgan narx butun menyu tannarxini
    // o'zgartiradi, va 90 000 keyinchalik oddiy raqamga o'xshaydi.
    lastPrice: (sum: string) => `oxirgi marta ${sum}`,
    whereTitle: "Qayerdan",
    wherePlaceholder: "Bozor, do'kon yoki sotuvchi",
    total: "Jami",
    send: "Omborga kiritish",
    sendHint:
      "Yuborilishi bilan ombor qoldig'i ko'tariladi va masalliq narxlari yangilanadi. Ega xabar oladi.",
    sent: (sum: string) => `Kiritildi: ${sum}`,
    alreadySent: "Bu xarid allaqachon kiritilgan.",
    nothingToSend: "Hech bo'lmasa bitta qatorga miqdor yozing.",
    loadFailed: "Ro'yxatni ochib bo'lmadi",
    sendFailed: "Yuborib bo'lmadi",
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
    loadFailed: "Ma'lumotni yuklab bo'lmadi",
  },

  // ⚠️ Kirish ekrani emas: internet yo'qligini parol bilan tuzatib bo'lmaydi.
  offline: {
    title: "Internet yo'q",
    body: "Ilova serverga ulana olmadi. Wi-Fi yoki mobil internetni tekshiring — ulanish tiklanishi bilan ekran o'zi ochiladi.",
    retrying: (n: number) => (n === 0 ? "Tekshirilmoqda…" : `Qayta tekshirildi: ${n}`),
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
      working: "Ish haqi, grafik va smena o'zgarishlari haqida xabar keladi",
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
    // ⚠️ Nisbiy vaqt shu yerda, chunki ilova panel lug'atini import qilmaydi
    // (o'lchangan: +500 KB). Hisoblash esa ulashilgan — `lib/orderFlow`
    // dagi `timeAgo` shu yorliqlarni oladi.
    timeAgo: {
      now: "hozir",
      min: (n: number) => `${n} daq oldin`,
      hour: (n: number) => `${n} soat oldin`,
      day: (n: number) => `${n} kun oldin`,
    },
  },
};

export type Dict = typeof uz;

export const ru: Dict = {
  lang: "Русский",
  appName: "Keel Team",

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

  tabs: { profile: "Смена", settings: "Настройки", buy: "Закуп", zakup: "Заявка" },

  zakup: {
    title: "Закуп",
    forDate: "На какой день",
    listTitle: "Список",
    qty: "Количество",
    shortTitle: "Чего не хватает на складе",
    search: "Название ингредиента — введите и выберите",
    unitIs: (unit: string) => `единица: ${unit}`,
    nothingShort: "Пока всего хватает.",
    addNew: (name: string) => `Добавить «${name}» в список`,
    onHand: (n: number, unit: string) => `остаток ${n} ${unit}`,
    need: (n: number, unit: string) => `нужно ${n} ${unit}`,
    review: "Просмотреть",
    previewTitle: "Перед отправкой",
    previewBody: (date: string) => `Список закупа на ${date}.`,
    back: "Назад",
    send: "Отправить",
    sent: (n: number) => `Отправлено: ${n} строк`,
    sentTitle: "Отправленные",
    statusSent: "на рынке",
    statusDone: "закрыт",
    progress: (got: number, all: number) => `куплено ${got}/${all}`,
    loadFailed: "Не удалось открыть список",
    sendFailed: "Не удалось отправить",
  },

  buy: {
    purse: "Деньги на руках",
    purseOwed: "Ресторан должен вам",
    purseOf: (issued: string, spent: string) =>
      `выдано ${issued} · потрачено ${spent}`,
    orderTitle: (d: string) => `Список закупа — ${d}`,
    orderFrom: (who: string) => `отправил(а) ${who}`,
    asked: (n: number, u: string) => `запрошено: ${n} ${u}`,
    got: "Куплено",
    changed: "Изменить",
    noneLeft: "Не было",
    wasMissing: "На рынке не было",
    undo: "Отменить",
    finish: "Завершить и оприходовать",
    shortTitle: "Чего не хватает",
    nothingShort: "Пока всего хватает.",
    basketTitle: "Куплено",
    addTitle: "Добавить ещё",
    searchPlaceholder: "Название ингредиента",
    addNew: (name: string) => `Добавить «${name}» как новый ингредиент`,
    onHand: (n: number, unit: string) => `остаток ${n} ${unit}`,
    need: (n: number, unit: string) => `нужно ${n} ${unit}`,
    qty: (unit: string) => (unit ? `Количество (${unit})` : "Количество"),
    price: "Цена",
    lastPrice: (sum: string) => `в прошлый раз ${sum}`,
    whereTitle: "Откуда",
    wherePlaceholder: "Рынок, магазин или продавец",
    total: "Итого",
    send: "Оприходовать",
    sendHint:
      "После отправки остаток вырастет и цены ингредиентов обновятся. Владелец получит уведомление.",
    sent: (sum: string) => `Оприходовано: ${sum}`,
    alreadySent: "Этот закуп уже оприходован.",
    nothingToSend: "Укажите количество хотя бы в одной строке.",
    loadFailed: "Не удалось открыть список",
    sendFailed: "Не удалось отправить",
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
    loadFailed: "Не удалось загрузить данные",
  },

  offline: {
    title: "Нет интернета",
    body: "Приложение не смогло связаться с сервером. Проверьте Wi-Fi или мобильный интернет — как только связь появится, экран откроется сам.",
    retrying: (n: number) => (n === 0 ? "Проверяем…" : `Проверок: ${n}`),
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
      working: "Придут сообщения о зарплате, графике и изменениях смены",
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
    timeAgo: {
      now: "только что",
      min: (n: number) => `${n} мин назад`,
      hour: (n: number) => `${n} ч назад`,
      day: (n: number) => `${n} дн назад`,
    },
  },
};

export const en: Dict = {
  lang: "English",
  appName: "Keel Team",

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

  tabs: { profile: "Shift", settings: "Settings", buy: "Market", zakup: "Order" },

  zakup: {
    title: "Shopping",
    forDate: "For which day",
    listTitle: "The list",
    qty: "Quantity",
    shortTitle: "What the store is short of",
    search: "Ingredient name — type to pick",
    unitIs: (unit: string) => `unit: ${unit}`,
    nothingShort: "Nothing is short right now.",
    addNew: (name: string) => `Add "${name}" to the list`,
    onHand: (n: number, unit: string) => `on hand ${n} ${unit}`,
    need: (n: number, unit: string) => `need ${n} ${unit}`,
    review: "Review",
    previewTitle: "Before sending",
    previewBody: (date: string) => `Shopping list for ${date}.`,
    back: "Back",
    send: "Send",
    sent: (n: number) => `Sent: ${n} lines`,
    sentTitle: "Sent",
    statusSent: "at the market",
    statusDone: "closed",
    progress: (got: number, all: number) => `${got}/${all} bought`,
    loadFailed: "Could not open the list",
    sendFailed: "Could not send",
  },

  buy: {
    purse: "Cash on hand",
    purseOwed: "The restaurant owes you",
    purseOf: (issued: string, spent: string) =>
      `${issued} issued · ${spent} spent`,
    orderTitle: (d: string) => `Shopping list — ${d}`,
    orderFrom: (who: string) => `sent by ${who}`,
    asked: (n: number, u: string) => `asked: ${n} ${u}`,
    got: "Bought",
    changed: "Change",
    noneLeft: "None left",
    wasMissing: "The market had none",
    undo: "Undo",
    finish: "Finish and book in",
    shortTitle: "What is short",
    nothingShort: "Nothing is short right now.",
    basketTitle: "Bought",
    addTitle: "Add more",
    searchPlaceholder: "Ingredient name",
    addNew: (name: string) => `Add "${name}" as a new ingredient`,
    onHand: (n: number, unit: string) => `on hand ${n} ${unit}`,
    need: (n: number, unit: string) => `need ${n} ${unit}`,
    qty: (unit: string) => (unit ? `Quantity (${unit})` : "Quantity"),
    price: "Price",
    lastPrice: (sum: string) => `last time ${sum}`,
    whereTitle: "Where from",
    wherePlaceholder: "Market, shop or seller",
    total: "Total",
    send: "Book into the store",
    sendHint:
      "Sending raises the shelf and updates ingredient prices. The owner is told.",
    sent: (sum: string) => `Booked in: ${sum}`,
    alreadySent: "This run is already booked in.",
    nothingToSend: "Put a quantity on at least one line.",
    loadFailed: "Could not open the list",
    sendFailed: "Could not send",
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
    loadFailed: "Could not load the data",
  },

  offline: {
    title: "No internet",
    body: "The app could not reach the server. Check Wi-Fi or mobile data — the screen opens by itself as soon as the connection is back.",
    retrying: (n: number) => (n === 0 ? "Checking…" : `Checked ${n} times`),
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
    timeAgo: {
      now: "just now",
      min: (n: number) => `${n} min ago`,
      hour: (n: number) => `${n} h ago`,
      day: (n: number) => `${n} d ago`,
    },
  },
};

export type Lang = "uz" | "ru" | "en";

export const DICTS: Record<Lang, Dict> = { uz, ru, en };
export const LANGS: Lang[] = ["uz", "ru", "en"];
