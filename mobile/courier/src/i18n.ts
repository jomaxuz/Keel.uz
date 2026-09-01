// The app's own words, in the three languages a restaurant here is run in.
//
// ⚠️ **Its own dictionary rather than the panel's, and that was measured on the
// waiter app first.** Importing `lib/i18n/admin` costs about **500 KB** of
// bundle: all three languages of every screen in the admin panel, to use about
// seventy strings. The shared thing worth sharing is the *rules* — what an
// arrival radius means, how a price is formatted, what a status is called in
// the data — and those are imported unchanged.
//
// ⚠️ **The refusals are the strings that matter here.** Most of this app is
// four screens a courier learns in a day; the sentences that decide whether
// somebody stands at a gate pressing a dead button are the ones on the arrival
// gate, and they are written to say *what to do*, not what went wrong.
//
// Uzbek is the source of truth: `Dict` is derived from it, so a key missing in
// ru or en is a compile error rather than a screen that falls back to Uzbek in
// front of a Russian-speaking courier.

export const uz = {
  lang: "O'zbekcha",
  appName: "Keel Courier",

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

  tabs: { orders: "Buyurtmalar", earnings: "Hisob", settings: "Sozlamalar" },

  shift: {
    title: "Smena",
    off: "Ishda emasman",
    free: "Bo'shman",
    busy: "Bandman",
    offHint: "Buyurtma berilmaydi",
    freeHint: "Yangi buyurtma keladi",
    busyHint: "Hozir yetkazyapman",
    // ⚠️ Says what the switch causes, not what it is. "Off" on a courier's
    // phone means the dispatcher stops giving them work — worth spelling out
    // before somebody leaves it off overnight and wonders why it is quiet.
    startHint:
      "Ishni boshlash uchun «Bo'shman» ni tanlang — shundan keyin buyurtma beriladi va joylashuv yuboriladi.",
  },

  geo: {
    title: "Joylashuv",
    on: "Yuborilmoqda",
    waiting: "signal kutilmoqda",
    lastSent: (time: string) => `oxirgi: ${time}`,
    queued: (n: number) => `${n} ta navbatda`,
    denied: "Joylashuvga ruxsat berilmagan",
    deniedHint:
      "Telefon sozlamalaridan ruxsat bering: mijoz manzilida ekaningiz shu bilan tekshiriladi.",
    failed: "Joylashuv aniqlanmadi",
    failedHint: "GPS yoqilganini va osmon ochiqligini tekshiring.",
    allow: "Ruxsat berish",
    // ⚠️ The one sentence that explains the persistent battery cost, said once
    // and not on every screen: a courier who does not know why the app must
    // stay open will close it, and then nothing works and nothing says why.
    keepOpen:
      "Ilova ochiq turganda joylashuv yuboriladi. Ekran o'chsa yuborilmaydi — shuning uchun smena davomida ilovani ochiq qoldiring.",
    // ⚠️ Ikkalasi bir xil emas: birinchisida telefonni cho'ntakka solsa bo'ladi,
    // ikkinchisida yo'q. Kuryer buni bilishi kerak, chunki tugma shunga bog'liq.
    inBackground: "Fonda ham yuborilmoqda — telefonni cho'ntakka solsangiz bo'ladi",
  },

  orders: {
    title: (n: number) => (n === 0 ? "Buyurtmalar" : `Buyurtmalar (${n})`),
    empty: "Hozircha buyurtma yo'q",
    emptyHint: "Yangi buyurtma berilganda shu yerda paydo bo'ladi.",
    offEmpty: "Siz ishda emassiz",
    offEmptyHint: "Buyurtma olish uchun yuqoridan «Bo'shman» ni tanlang.",
    cash: (sum: string) => `Naqd olinadi: ${sum}`,
    paid: "Onlayn to'langan",
    pickUp: "Olib chiqdim",
    deliver: "Yetkazildi",
    call: "Qo'ng'iroq",
    route: "Yo'l",
    routeTitle: "Mijozgacha yo'l",
    failed: "Bajarilmadi",
    // Distance to the customer, shown on the card while on the way.
    away: (m: number) => (m < 1000 ? `${m} m` : `${(m / 1000).toFixed(1)} km`),
    atDoor: "Manzildasiz",
    comment: "Izoh",
  },

  // ⚠️ **The gate, in the courier's words.** The server refuses too — that is
  // the rule, this is only the button — but a refusal that arrives after the
  // press is read as a broken app, and the courier is standing in front of a
  // customer while reading it.
  gate: {
    tooFar: (m: number, need: number) =>
      `Mijoz manzilidan ${m} m uzoqdasiz. «Yetkazildi» ni ${need} m ichida bosish mumkin.`,
    noFix: "Joylashuv hali aniqlanmadi — GPS ni yoqing va bir oz kuting.",
    stale: "Joylashuv eskirgan — ilovani ochiq qoldiring, yangilanishini kuting.",
  },

  earnings: {
    title: "Hisob",
    today: "Bugun",
    week: "Hafta",
    month: "Oy",
    all: "Jami",
    deliveries: (n: number) => `${n} ta yetkazish`,
    cashInHand: (sum: string) => `Qo'lingizda ${sum} naqd pul bor — restoranga topshiring.`,
    cashClear: "Naqd pul topshirilgan — qarz yo'q.",
    // ⚠️ One word, not the sentence from the order card: here it sits in a
    // history row where the amount is already beside it.
    cash: "Naqd",
    history: "Yetkazilganlar",
    historyEmpty: "Hali yetkazilgan buyurtma yo'q",
    earned: "Ishlangan",
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
    phone: "Telefon",
    version: "Versiya",
    signOut: "Chiqish",
    changeServer: "Boshqa restoran",
    // ⚠️ Says what it does, because the two are not the same action: a shift
    // ends every day, a phone changes jobs once.
    changeServerHint: "Chiqish va boshqa restoran manzilini kiritish",
  },

  push: {
    title: "Bildirishnomalar",
    working: "Yoqilgan",
    asking: "Tekshirilmoqda…",
    denied: "Ruxsat berilmagan",
    noDevice: "Emulyatorda ishlamaydi",
    noProject: "Ilova sozlamasi to'liq emas",
    failed: "Ro'yxatdan o'tmadi",
    hint: {
      working: "Yangi buyurtma, bekor qilish va manzil o'zgarishi haqida xabar keladi",
      asking: "Bir soniya",
      denied: "Telefon sozlamalaridan bildirishnomalarni yoqing",
      noDevice: "Haqiqiy telefonda sinang",
      noProject: "Bu build eski — yangi versiyani o'rnating",
      failed: "Internet yoki server bilan bog'lanib bo'lmadi",
    },
  },

  notice: { ok: "Tushunarli" },

  common: {
    retry: "Qayta urinish",
    loading: "Yuklanmoqda…",
    cancel: "Bekor qilish",
    close: "Yopish",
  },
};

export type Dict = typeof uz;

export const ru: Dict = {
  lang: "Русский",
  appName: "Keel Courier",

  server: {
    title: "Адрес ресторана",
    hint: "Короткое имя вашего ресторана или полный адрес",
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

  tabs: { orders: "Заказы", earnings: "Счёт", settings: "Настройки" },

  shift: {
    title: "Смена",
    off: "Не на смене",
    free: "Свободен",
    busy: "Занят",
    offHint: "Заказы не дают",
    freeHint: "Придёт новый заказ",
    busyHint: "Сейчас доставляю",
    startHint:
      "Чтобы начать работу, выберите «Свободен» — после этого дают заказы и отправляется геопозиция.",
  },

  geo: {
    title: "Геопозиция",
    on: "Отправляется",
    waiting: "ждём сигнал",
    lastSent: (time: string) => `последняя: ${time}`,
    queued: (n: number) => `${n} в очереди`,
    denied: "Доступ к геопозиции запрещён",
    deniedHint:
      "Разрешите в настройках телефона: так проверяется, что вы на адресе клиента.",
    failed: "Геопозиция не определена",
    failedHint: "Проверьте, включён ли GPS и открыто ли небо.",
    allow: "Разрешить",
    keepOpen:
      "Геопозиция отправляется, пока приложение открыто. При выключенном экране — нет, поэтому держите приложение открытым всю смену.",
    inBackground: "Отправляется и в фоне — телефон можно убрать в карман",
  },

  orders: {
    title: (n: number) => (n === 0 ? "Заказы" : `Заказы (${n})`),
    empty: "Пока заказов нет",
    emptyHint: "Новый заказ появится здесь.",
    offEmpty: "Вы не на смене",
    offEmptyHint: "Чтобы получать заказы, выберите «Свободен» выше.",
    cash: (sum: string) => `Взять наличными: ${sum}`,
    paid: "Оплачено онлайн",
    pickUp: "Забрал",
    deliver: "Доставлено",
    call: "Позвонить",
    route: "Маршрут",
    routeTitle: "Маршрут до клиента",
    failed: "Не выполнено",
    away: (m: number) => (m < 1000 ? `${m} м` : `${(m / 1000).toFixed(1)} км`),
    atDoor: "Вы на адресе",
    comment: "Комментарий",
  },

  gate: {
    tooFar: (m: number, need: number) =>
      `Вы в ${m} м от адреса клиента. «Доставлено» можно нажать в пределах ${need} м.`,
    noFix: "Геопозиция ещё не определена — включите GPS и подождите немного.",
    stale: "Геопозиция устарела — оставьте приложение открытым и дождитесь обновления.",
  },

  earnings: {
    title: "Счёт",
    today: "Сегодня",
    week: "Неделя",
    month: "Месяц",
    all: "Всего",
    deliveries: (n: number) => `${n} доставок`,
    cashInHand: (sum: string) => `У вас на руках ${sum} наличными — сдайте в ресторан.`,
    cashClear: "Наличные сданы — долгов нет.",
    cash: "Наличные",
    history: "Доставленные",
    historyEmpty: "Доставленных заказов пока нет",
    earned: "Заработано",
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
    phone: "Телефон",
    version: "Версия",
    signOut: "Выйти",
    changeServer: "Другой ресторан",
    changeServerHint: "Выйти и ввести адрес другого ресторана",
  },

  push: {
    title: "Уведомления",
    working: "Включены",
    asking: "Проверяем…",
    denied: "Доступ запрещён",
    noDevice: "На эмуляторе не работают",
    noProject: "Настройка приложения неполная",
    failed: "Не зарегистрировано",
    hint: {
      working: "Придёт сообщение о новом заказе, отмене и смене адреса",
      asking: "Секунду",
      denied: "Включите уведомления в настройках телефона",
      noDevice: "Проверьте на реальном телефоне",
      noProject: "Эта сборка устарела — установите новую версию",
      failed: "Не удалось связаться с интернетом или сервером",
    },
  },

  notice: { ok: "Понятно" },

  common: {
    retry: "Повторить",
    loading: "Загрузка…",
    cancel: "Отмена",
    close: "Закрыть",
  },
};

export const en: Dict = {
  lang: "English",
  appName: "Keel Courier",

  server: {
    title: "Restaurant address",
    hint: "Your restaurant's short name or its full address",
    placeholder: "osh",
    next: "Continue",
    bad: "That does not look like an address",
  },

  login: {
    title: "Sign in",
    username: "Username",
    password: "Password",
    submit: "Sign in",
    other: "Another restaurant",
    failed: "Could not sign in",
  },

  tabs: { orders: "Orders", earnings: "Earnings", settings: "Settings" },

  shift: {
    title: "Shift",
    off: "Off shift",
    free: "Available",
    busy: "Busy",
    offHint: "No orders given",
    freeHint: "New orders come in",
    busyHint: "Delivering now",
    startHint:
      "Pick “Available” to start work — orders are given out and your location is sent from then on.",
  },

  geo: {
    title: "Location",
    on: "Sending",
    waiting: "waiting for a signal",
    lastSent: (time: string) => `last: ${time}`,
    queued: (n: number) => `${n} queued`,
    denied: "Location permission was refused",
    deniedHint:
      "Allow it in the phone's settings: this is what proves you are at the customer's address.",
    failed: "No location fix",
    failedHint: "Check that GPS is on and the sky is clear.",
    allow: "Allow",
    keepOpen:
      "Your location is sent while the app is open. With the screen off it is not — so keep the app open through the shift.",
    inBackground: "Sending in the background too — the phone can go in a pocket",
  },

  orders: {
    title: (n: number) => (n === 0 ? "Orders" : `Orders (${n})`),
    empty: "No orders yet",
    emptyHint: "A new order will appear here.",
    offEmpty: "You are off shift",
    offEmptyHint: "Pick “Available” above to get orders.",
    cash: (sum: string) => `Collect in cash: ${sum}`,
    paid: "Paid online",
    pickUp: "Picked up",
    deliver: "Delivered",
    call: "Call",
    route: "Route",
    routeTitle: "Route to the customer",
    failed: "That did not go through",
    away: (m: number) => (m < 1000 ? `${m} m` : `${(m / 1000).toFixed(1)} km`),
    atDoor: "You are at the address",
    comment: "Note",
  },

  gate: {
    tooFar: (m: number, need: number) =>
      `You are ${m} m from the customer's address. “Delivered” can be pressed within ${need} m.`,
    noFix: "No location fix yet — turn GPS on and give it a moment.",
    stale: "The location is stale — keep the app open and wait for an update.",
  },

  earnings: {
    title: "Earnings",
    today: "Today",
    week: "Week",
    month: "Month",
    all: "All time",
    deliveries: (n: number) => `${n} deliveries`,
    cashInHand: (sum: string) => `You are holding ${sum} in cash — hand it in at the restaurant.`,
    cashClear: "Cash handed in — nothing owed.",
    cash: "Cash",
    history: "Delivered",
    historyEmpty: "No deliveries yet",
    earned: "Earned",
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
    phone: "Phone",
    version: "Version",
    signOut: "Sign out",
    changeServer: "Another restaurant",
    changeServerHint: "Sign out and enter another restaurant's address",
  },

  push: {
    title: "Notifications",
    working: "On",
    asking: "Checking…",
    denied: "Permission refused",
    noDevice: "Not available on a simulator",
    noProject: "The app's configuration is incomplete",
    failed: "Not registered",
    hint: {
      working: "You will be told about a new order, a cancellation and a moved address",
      asking: "One moment",
      denied: "Turn notifications on in the phone's settings",
      noDevice: "Try it on a real phone",
      noProject: "This build is out of date — install the new version",
      failed: "Could not reach the internet or the server",
    },
  },

  notice: { ok: "Got it" },

  common: {
    retry: "Try again",
    loading: "Loading…",
    cancel: "Cancel",
    close: "Close",
  },
};

export const LANGS = ["uz", "ru", "en"] as const;
export type Lang = (typeof LANGS)[number];
export const DICTS: Record<Lang, Dict> = { uz, ru, en };
