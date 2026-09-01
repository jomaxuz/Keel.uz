// The app's own words, in the three languages a restaurant here is run in.
//
// ⚠️ **Its own dictionary rather than the panel's, and that was measured on the
// waiter app first**: importing `lib/i18n/admin` costs about 500 KB of bundle —
// all three languages of every screen in the panel — to use eighty strings.
// What is worth sharing is the *rules*, and those are imported unchanged.
//
// ⚠️ **This app deliberately says less than the panel.** It is read on a phone,
// usually while somebody is doing something else, and every sentence here has
// to survive being read in three seconds at arm's length. Where the panel
// explains, this labels.
//
// Uzbek is the source of truth: `Dict` is derived from it, so a key missing in
// ru or en is a compile error rather than a screen that falls back to Uzbek in
// front of a Russian-speaking owner.

export const uz = {
  lang: "O'zbekcha",
  appName: "Keel Owner",

  server: {
    title: "Restoran manzili",
    hint: "Restoraningizning qisqa nomi yoki to'liq manzili",
    placeholder: "osh",
    next: "Davom etish",
    bad: "Bu manzilga o'xshamaydi",
  },

  login: {
    title: "Panelga kirish",
    username: "Login",
    password: "Parol",
    submit: "Kirish",
    other: "Boshqa restoran",
    failed: "Kirib bo'lmadi",
  },

  tabs: {
    today: "Bugun",
    alerts: "Diqqat",
    orders: "Buyurtmalar",
    reports: "Hisobot",
    settings: "Sozlamalar",
  },

  today: {
    title: "Bugun",
    revenue: "Tushum",
    orders: "Buyurtmalar",
    average: "O'rtacha chek",
    delivered: "Yetkazilgan",
    cancelled: "Bekor qilingan",
    // ⚠️ Kechagi bilan solishtirish — raqamning yonidagi yagona kontekst.
    // «12 400 000» o'zi yaxshi ham, yomon ham emas.
    vsYesterday: (percent: number) =>
      percent === 0
        ? "kecha bilan bir xil"
        : percent > 0
          ? `kechaga nisbatan +${percent}%`
          : `kechaga nisbatan ${percent}%`,
    noYesterday: "kecha ma'lumot yo'q",
    branchAll: "Hamma filial",
    updated: (time: string) => `Yangilandi: ${time}`,
  },

  alerts: {
    title: "Diqqat",
    empty: "Hozircha hech nima talab qilinmayapti",
    emptyHint: "Yangi buyurtma yoki muammo chiqsa shu yerda ko'rinadi.",
    pendingOrders: (n: number) => `${n} ta buyurtma tasdiqlanmagan`,
    preorders: (n: number) => `${n} ta oldindan buyurtma`,
    reservations: (n: number) => `${n} ta bron tasdiqlanmagan`,
    posUnaccepted: (n: number) => `${n} ta buyurtmani kassa qabul qilmadi`,
    posFailed: (n: number) => `${n} ta buyurtma kassaga umuman bormadi`,
    posUnmapped: (n: number) => `${n} ta taom kassaga bog'lanmagan`,
    printFailed: (n: number) => `${n} ta chek chiqarilmadi`,
    // Loss alerts: nima bo'lgani va qancha.
    // ⚠️ Server yangi turini o'ylab topishi mumkin — noma'lumi kalitning
    // o'zi bo'lib chiqadi: so'z haqida so'rasa bo'ladi, bo'shliq haqida yo'q.
    kinds: {
      void_after_precheck: "Hisobdan keyin olib tashlandi",
      big_discount: "Katta chegirma",
      cash_short: "Kassada kamomad",
      stock_short: "Ombor kamomadi",
      recipe_up: "Texkarta oshirildi",
      panel_action: "Paneldagi amal",
      check_cancelled: "Chek bekor qilindi",
    },
    lossTitle: "Pul bilan bog'liq",
    lossEmpty: "Bugun bunday hodisa yo'q",
    by: (who: string) => `${who}`,
  },

  orders: {
    title: "Buyurtmalar",
    empty: "Ochiq buyurtma yo'q",
    emptyHint: "Yangi buyurtma kelganda shu yerda paydo bo'ladi.",
    confirm: "Tasdiqlash",
    cancel: "Bekor qilish",
    call: "Qo'ng'iroq",
    cancelReason: "Sabab (majburiy)",
    cancelDo: "Bekor qilish",
    cancelBack: "Yopish",
    failed: "Bajarilmadi",
    ago: (ago: string) => ago,
  },

  reports: {
    title: "Hisobot",
    today: "Bugun",
    week: "Hafta",
    month: "Oy",
    revenue: "Tushum",
    orders: "Buyurtmalar",
    average: "O'rtacha chek",
    topDishes: "Ko'p sotilgan",
    staff: "Xodimlar",
    hours: (h: string) => `${h} soat`,
    owed: "To'lanishi kerak",
    stock: "Kam qolgan mahsulot",
    stockEmpty: "Hammasi yetarli",
    empty: "Bu davrda ma'lumot yo'q",
  },

  settings: {
    title: "Sozlamalar",
    language: "Til",
    theme: "Ko'rinish",
    themeSystem: "Tizim bo'yicha",
    themeLight: "Yorug'",
    themeDark: "Qorong'i",
    account: "Hisob",
    branch: "Filial",
    restaurant: "Restoran",
    version: "Versiya",
    signOut: "Chiqish",
    changeServer: "Boshqa restoran",
    changeServerHint: "Chiqish va boshqa restoran manzilini kiritish",
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
      working: "Yangi buyurtma, katta chegirma, kassa farqi haqida xabar keladi",
      asking: "Bir soniya",
      denied: "Telefon sozlamalaridan bildirishnomalarni yoqing",
      noDevice: "Haqiqiy telefonda sinang",
      noProject: "Bu build eski — yangi versiyani o'rnating",
      failed: "Internet yoki server bilan bog'lanib bo'lmadi",
    },
  },

  offline: {
    title: "Internet yo'q",
    body: "Ilova serverga ulana olmadi. Wi-Fi yoki mobil internetni tekshiring — ulanish tiklanishi bilan ekran o'zi ochiladi.",
    retrying: (n: number) => (n === 0 ? "Tekshirilmoqda…" : `Qayta tekshirildi: ${n}`),
  },

  notice: { ok: "Tushunarli" },

  common: {
    retry: "Qayta urinish",
    loading: "Yuklanmoqda…",
    close: "Yopish",
    loadFailed: "Ma'lumotni yuklab bo'lmadi",
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
  appName: "Keel Owner",

  server: {
    title: "Адрес ресторана",
    hint: "Короткое имя вашего ресторана или полный адрес",
    placeholder: "osh",
    next: "Продолжить",
    bad: "Это не похоже на адрес",
  },

  login: {
    title: "Вход в панель",
    username: "Логин",
    password: "Пароль",
    submit: "Войти",
    other: "Другой ресторан",
    failed: "Не удалось войти",
  },

  tabs: {
    today: "Сегодня",
    alerts: "Важное",
    orders: "Заказы",
    reports: "Отчёт",
    settings: "Настройки",
  },

  today: {
    title: "Сегодня",
    revenue: "Выручка",
    orders: "Заказы",
    average: "Средний чек",
    delivered: "Доставлено",
    cancelled: "Отменено",
    vsYesterday: (percent: number) =>
      percent === 0
        ? "как вчера"
        : percent > 0
          ? `к вчерашнему +${percent}%`
          : `к вчерашнему ${percent}%`,
    noYesterday: "за вчера данных нет",
    branchAll: "Все филиалы",
    updated: (time: string) => `Обновлено: ${time}`,
  },

  alerts: {
    title: "Важное",
    empty: "Сейчас ничего не требует внимания",
    emptyHint: "Новый заказ или проблема появятся здесь.",
    pendingOrders: (n: number) => `${n} заказов не подтверждено`,
    preorders: (n: number) => `${n} предзаказов`,
    reservations: (n: number) => `${n} броней не подтверждено`,
    posUnaccepted: (n: number) => `${n} заказов касса не приняла`,
    posFailed: (n: number) => `${n} заказов вообще не дошли до кассы`,
    posUnmapped: (n: number) => `${n} блюд не привязаны к кассе`,
    printFailed: (n: number) => `${n} чеков не напечатано`,
    kinds: {
      void_after_precheck: "Удалено после счёта",
      big_discount: "Большая скидка",
      cash_short: "Недостача в кассе",
      stock_short: "Недостача на складе",
      recipe_up: "Техкарта увеличена",
      panel_action: "Действие в панели",
      check_cancelled: "Счёт отменён",
    },
    lossTitle: "Связанное с деньгами",
    lossEmpty: "Сегодня таких событий нет",
    by: (who: string) => `${who}`,
  },

  orders: {
    title: "Заказы",
    empty: "Открытых заказов нет",
    emptyHint: "Новый заказ появится здесь.",
    confirm: "Подтвердить",
    cancel: "Отменить",
    call: "Позвонить",
    cancelReason: "Причина (обязательно)",
    cancelDo: "Отменить заказ",
    cancelBack: "Закрыть",
    failed: "Не выполнено",
    ago: (ago: string) => ago,
  },

  reports: {
    title: "Отчёт",
    today: "Сегодня",
    week: "Неделя",
    month: "Месяц",
    revenue: "Выручка",
    orders: "Заказы",
    average: "Средний чек",
    topDishes: "Больше всего продано",
    staff: "Сотрудники",
    hours: (h: string) => `${h} ч`,
    owed: "К выплате",
    stock: "Заканчивается",
    stockEmpty: "Всего хватает",
    empty: "За этот период данных нет",
  },

  settings: {
    title: "Настройки",
    language: "Язык",
    theme: "Оформление",
    themeSystem: "Как в системе",
    themeLight: "Светлое",
    themeDark: "Тёмное",
    account: "Аккаунт",
    branch: "Филиал",
    restaurant: "Ресторан",
    version: "Версия",
    signOut: "Выйти",
    changeServer: "Другой ресторан",
    changeServerHint: "Выйти и ввести адрес другого ресторана",
    notifications: "Уведомления",
    push: {
      working: "Включены",
      asking: "Проверяем…",
      denied: "Доступ запрещён",
      noDevice: "На эмуляторе не работают",
      noProject: "Настройка приложения неполная",
      failed: "Не зарегистрировано",
    },
    pushHint: {
      working: "Придут сообщения о новом заказе, большой скидке, расхождении в кассе",
      asking: "Секунду",
      denied: "Включите уведомления в настройках телефона",
      noDevice: "Проверьте на реальном телефоне",
      noProject: "Эта сборка устарела — установите новую версию",
      failed: "Не удалось связаться с интернетом или сервером",
    },
  },

  offline: {
    title: "Нет интернета",
    body: "Приложение не смогло связаться с сервером. Проверьте Wi-Fi или мобильный интернет — как только связь появится, экран откроется сам.",
    retrying: (n: number) => (n === 0 ? "Проверяем…" : `Проверок: ${n}`),
  },

  notice: { ok: "Понятно" },

  common: {
    retry: "Повторить",
    loading: "Загрузка…",
    close: "Закрыть",
    loadFailed: "Не удалось загрузить данные",
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
  appName: "Keel Owner",

  server: {
    title: "Restaurant address",
    hint: "Your restaurant's short name or its full address",
    placeholder: "osh",
    next: "Continue",
    bad: "That does not look like an address",
  },

  login: {
    title: "Sign in to the panel",
    username: "Username",
    password: "Password",
    submit: "Sign in",
    other: "Another restaurant",
    failed: "Could not sign in",
  },

  tabs: {
    today: "Today",
    alerts: "Attention",
    orders: "Orders",
    reports: "Reports",
    settings: "Settings",
  },

  today: {
    title: "Today",
    revenue: "Takings",
    orders: "Orders",
    average: "Average check",
    delivered: "Delivered",
    cancelled: "Cancelled",
    vsYesterday: (percent: number) =>
      percent === 0
        ? "same as yesterday"
        : percent > 0
          ? `+${percent}% on yesterday`
          : `${percent}% on yesterday`,
    noYesterday: "nothing for yesterday",
    branchAll: "All branches",
    updated: (time: string) => `Updated: ${time}`,
  },

  alerts: {
    title: "Attention",
    empty: "Nothing needs you right now",
    emptyHint: "A new order or a problem shows up here.",
    pendingOrders: (n: number) => `${n} orders not accepted`,
    preorders: (n: number) => `${n} pre-orders`,
    reservations: (n: number) => `${n} bookings not confirmed`,
    posUnaccepted: (n: number) => `${n} orders the till has not accepted`,
    posFailed: (n: number) => `${n} orders never reached the till`,
    posUnmapped: (n: number) => `${n} dishes are not mapped to the till`,
    printFailed: (n: number) => `${n} receipts did not print`,
    kinds: {
      void_after_precheck: "Removed after the bill",
      big_discount: "Large discount",
      cash_short: "Till short",
      stock_short: "Stock short",
      recipe_up: "Recipe increased",
      panel_action: "Panel action",
      check_cancelled: "Check cancelled",
    },
    lossTitle: "About money",
    lossEmpty: "Nothing of this kind today",
    by: (who: string) => `${who}`,
  },

  orders: {
    title: "Orders",
    empty: "No open orders",
    emptyHint: "A new order will appear here.",
    confirm: "Accept",
    cancel: "Cancel",
    call: "Call",
    cancelReason: "Reason (required)",
    cancelDo: "Cancel the order",
    cancelBack: "Close",
    failed: "That did not go through",
    ago: (ago: string) => ago,
  },

  reports: {
    title: "Reports",
    today: "Today",
    week: "Week",
    month: "Month",
    revenue: "Takings",
    orders: "Orders",
    average: "Average check",
    topDishes: "Best sellers",
    staff: "Staff",
    hours: (h: string) => `${h} h`,
    owed: "Owed",
    stock: "Running out",
    stockEmpty: "Everything is in stock",
    empty: "Nothing for this period",
  },

  settings: {
    title: "Settings",
    language: "Language",
    theme: "Appearance",
    themeSystem: "Follow the system",
    themeLight: "Light",
    themeDark: "Dark",
    account: "Account",
    branch: "Branch",
    restaurant: "Restaurant",
    version: "Version",
    signOut: "Sign out",
    changeServer: "Another restaurant",
    changeServerHint: "Sign out and enter another restaurant's address",
    notifications: "Notifications",
    push: {
      working: "On",
      asking: "Checking…",
      denied: "Permission refused",
      noDevice: "Not available on a simulator",
      noProject: "The app's configuration is incomplete",
      failed: "Not registered",
    },
    pushHint: {
      working: "You will be told about a new order, a large discount, a till shortfall",
      asking: "One moment",
      denied: "Turn notifications on in the phone's settings",
      noDevice: "Try it on a real phone",
      noProject: "This build is out of date — install the new version",
      failed: "Could not reach the internet or the server",
    },
  },

  offline: {
    title: "No internet",
    body: "The app could not reach the server. Check Wi-Fi or mobile data — the screen opens by itself as soon as the connection is back.",
    retrying: (n: number) => (n === 0 ? "Checking…" : `Checked ${n} times`),
  },

  notice: { ok: "Got it" },

  common: {
    retry: "Try again",
    loading: "Loading…",
    close: "Close",
    loadFailed: "Could not load the data",
    timeAgo: {
      now: "just now",
      min: (n: number) => `${n} min ago`,
      hour: (n: number) => `${n} h ago`,
      day: (n: number) => `${n} d ago`,
    },
  },
};

export const LANGS = ["uz", "ru", "en"] as const;
export type Lang = (typeof LANGS)[number];
export const DICTS: Record<Lang, Dict> = { uz, ru, en };
