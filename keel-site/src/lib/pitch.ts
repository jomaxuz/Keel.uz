// keel.uz/pitch — the Pitch Day 3.0 page, as data.
//
// ⚠️ **Every claim here has to be checkable in this repository.** The page is
// read by a jury deciding whether the product is real; one invented number
// ("500 restaurants", "99.9% uptime") is the sentence that makes them doubt
// every true one beside it. So:
//   - no customer counts, revenue, partners or awards — none are recorded here;
//   - integrations are listed as *what we built*, never as partnerships, and
//     the ones not yet proven live are labelled so (see `INTEGRATIONS`);
//   - AI is described as it is wired today (control/internal/ai, handlers/
//     insight.go, advisor.go, supportai.go, menuextract.go) and the rest is
//     marked as planned.
//
// ⚠️ **Copy lives here, the layout in app/pitch/page.tsx.** The page is Uzbek
// only (the competition is), but keeping the words in one typed object means a
// Russian or English version is a second object, not a second page.
//
// ⚠️ **`PITCH_CONFIG` is the one place to edit before submitting** — the demo
// video and the founder's personal links. Empty values render an honest
// placeholder, never a fake player or a dead link.

/** What to change before the jury looks. */
export const PITCH_CONFIG = {
  video: {
    /** A YouTube video id (the part after `watch?v=`). Preferred: it streams
     *  from YouTube, so a 3-minute demo costs keel.uz nothing. */
    youtubeId: "",
    /** Or a direct MP4 address, e.g. "/pitch/demo.mp4" in `public/pitch/`. */
    mp4: "",
    /** The still shown before playing. Defaults to the till screenshot. */
    poster: "",
    /** "3:40" — shown on the player. Leave empty until the video exists. */
    duration: "",
  },
  founder: {
    name: "Yusuf Nurmanov",
    role: "Asoschi · CEO · Full-stack dasturchi",
    /** Personal links. ⚠️ Only fill what really exists — an empty one is
     *  simply not drawn. */
    links: {
      telegram: "",
      linkedin: "",
      github: "",
    },
  },
} as const;

export type ModuleCard = {
  name: string;
  desc: string;
  /** A screenshot from `public/shots`, when one exists for this module. */
  shot?: string;
};

export const PITCH = {
  meta: {
    title: "KEEL — Pitch Day 3.0",
    description:
      "KEEL — restoran, kafe va do'konlar uchun bitta operatsion tizim: kassa, zal, oshxona ekrani, ombor va tannarx, mijozlar, onlayn buyurtma, yetkazish va hisobotlar bir joyda.",
  },

  hero: {
    badge: "Pitch Day 3.0",
    title: "Biznesingizni *bitta tizimda* boshqaring.",
    lead: "KEEL restoranning kundalik ishini — kassa, zal, oshxona, ombor, mijozlar, onlayn buyurtma, yetkazish va hisobotlarni — bitta platformaga birlashtiradi. Bir taom bir marta kiritiladi va hamma joyda ishlaydi.",
    ctaVideo: "Demo videoni ko'rish",
    ctaProduct: "Ishlayotgan mahsulotni ko'rish",
    shotAlt: "KEEL kassa ekrani: chapda menyu, o'ngda stolning ochiq cheki",
  },

  /** The jury's checklist, in the order the organisers listed it. */
  requirements: [
    { id: "problem", label: "Muammo → Yechim" },
    { id: "team", label: "Jamoa" },
    { id: "why-us", label: "Nega aynan biz" },
    { id: "roadmap", label: "Yo'l xaritasi" },
    { id: "tech", label: "Amalga oshirish va texnologiyalar" },
    { id: "demo", label: "Demo video" },
    { id: "demo-about", label: "Demo haqida" },
    { id: "live", label: "Ishlayotgan mahsulot havolasi" },
  ],
  requirementsTitle: "Pitch Day talablari — shu sahifada",

  problem: {
    eyebrow: "01 · Muammo → Yechim",
    title: "Restoran bitta biznes, lekin *o'nta alohida tizim*",
    lead: "Kichik restoran ham har kuni sotuv, ombor, oshxona, mijozlar, onlayn buyurtma, yetkazish, xodimlar va hisobotni boshqaradi. Odatda ularning har biri alohida dasturda, Excel'da yoki daftarda yuritiladi.",
    nodes: [
      "Kassa",
      "Ombor",
      "Oshxona",
      "Mijozlar",
      "Onlayn buyurtma",
      "Yetkazish",
      "Xodimlar",
      "Hisobotlar",
    ],
    pains: [
      {
        title: "Ikki marta ish",
        desc: "Bitta menyu kassada, saytda, botda va agregatorda alohida yuritiladi; narx bir joyda o'zgaradi, boshqasida eskicha qoladi.",
      },
      {
        title: "Ko'rinmaslik",
        desc: "Ega bugungi tushumni, ombordagi qoldiqni va taomning tannarxini bitta joydan ko'ra olmaydi — raqamlar oy oxirida, qo'lda yig'iladi.",
      },
      {
        title: "Uzilgan jarayon",
        desc: "Buyurtma oshxonaga og'zaki boradi, sotilgan taom omborda ayrilmaydi, mijoz kim ekani hech qayerda qolmaydi.",
      },
    ],
    solutionTitle: "Yechim: hammasi *bitta KEEL* ichida",
    solutionLead:
      "KEEL bu jarayonlarni bitta operatsion platformaga birlashtiradi: bitta menyu, bitta mijozlar bazasi, bitta hisobot. Kassada bosilgan tugma oshxonada, omborda va egasining telefonida bir vaqtda ko'rinadi.",
    hub: ["Kassa", "Zal", "Oshxona ekrani", "Ombor va tannarx", "CRM", "Onlayn buyurtma", "Yetkazish", "Hisobotlar"],
  },

  flow: {
    eyebrow: "02 · KEEL qanday ishlaydi",
    title: "Modullar ro'yxat emas — *bitta zanjir*",
    lead: "Bitta buyurtmaning yo'li. Har qadam keyingisiga ma'lumotni o'zi uzatadi — hech kim uni qayta kiritmaydi.",
    steps: [
      {
        n: "1",
        title: "Buyurtma keladi",
        desc: "Kassada, ofitsiant planshetida, saytda, Telegram botda yoki stoldagi QR menyudan — barchasi bitta menyudan.",
        shot: "floor",
        alt: "Zal xaritasi: band stollar, ochiq cheklar summasi va o'tirish vaqti",
      },
      {
        n: "2",
        title: "Kassa cheki",
        desc: "Taom chekka tushadi; kassir yoki ofitsiant uni «Oshxonaga yuborish» bilan oshxonaga jo'natadi.",
        shot: "till",
        alt: "KEEL kassa ekrani: taomlar to'ri va stolning ochiq cheki",
      },
      {
        n: "3",
        title: "Oshxona ekrani",
        desc: "Chek oshxona ekraniga o'zi tushadi, eng uzoq kutgani birinchi turadi. Oshpaz «tayyor» bosadi — zal buni darhol ko'radi.",
        shot: "kds",
        alt: "Oshxona ekrani: navbatdagi cheklar, taom izohlari bilan",
      },
      {
        n: "4",
        title: "Ombor o'zi ayriladi",
        desc: "Sotilgan taom texkarta bo'yicha masalliqlarga yoyiladi va qoldiqdan ayriladi; tannarx va yalpi foyda o'zi hisoblanadi.",
        shot: "stock",
        alt: "Ombor: masalliqlar qoldig'i va harakati",
      },
      {
        n: "5",
        title: "To'lov va fiskal chek",
        desc: "Naqd, karta yoki o'tkazma; fiskal chek ro'yxatdan o'tadi, internet uzilsa navbatda kutadi.",
        shot: "pay",
        alt: "Kassada to'lov oynasi: to'lov usuli, chegirma va qaytim",
      },
      {
        n: "6",
        title: "Mijoz va hisobot",
        desc: "Buyurtma mijozning tarixiga, keshbek va segmentlarga yoziladi; ega savdo, kanal va tannarx hisobotini bitta panelda ko'radi.",
        shot: "dashboard",
        alt: "Boshqaruv paneli: savdo dinamikasi va bugungi ko'rsatkichlar",
      },
    ],
  },

  modules: {
    eyebrow: "03 · Mahsulot",
    title: "Beshta yo'nalish, *bitta ma'lumotlar bazasi*",
    lead: "Ro'yxat to'liq emas — faqat restoranni har kuni yuritadigan asosiy qismlar.",
    groups: [
      {
        name: "Operatsiyalar",
        items: [
          { name: "Kassa (POS)", desc: "Windows monoblok uchun dastur: PIN va rollar, chekni bo'lish va birlashtirish, printer va pul yashigi." },
          { name: "Zal va stollar", desc: "Zal xaritasi, zonalar, ofitsiant ilovasi, bron." },
          { name: "Oshxona ekrani (KDS)", desc: "Cheklar navbati, «tayyor» belgisi, ofitsiantga bildirishnoma." },
          { name: "Xodimlar", desc: "Grafik, telefon orqali geolokatsiyali davomat, oylik hisob-kitobi." },
        ],
      },
      {
        name: "Ombor",
        items: [
          { name: "Ombor va qoldiqlar", desc: "Kirim, chiqim, ko'chirish, inventarizatsiya, filiallararo jo'natma." },
          { name: "Texkarta va tannarx", desc: "Taom va zagotovka kartalari; har taomning tannarxi va foydasi." },
        ],
      },
      {
        name: "Mijozlar",
        items: [
          { name: "CRM", desc: "Mijozlar bazasi, segmentlar va RFM, buyurtmalar tarixi." },
          { name: "Sodiqlik va marketing", desc: "Keshbek, promokod, aksiyalar; SMS, Telegram va push xabarlar." },
        ],
      },
      {
        name: "Raqamli savdo",
        items: [
          { name: "Restoran sayti", desc: "O'z domenida, uch tilda, panel bilan bir menyudan." },
          { name: "Onlayn buyurtma va to'lov", desc: "Sayt, Telegram bot va mini app, QR menyu; onlayn to'lov." },
          { name: "Yetkazish", desc: "Xaritada zona va narx, kuryer ilovasi, buyurtmani eng yaqin filialga yo'naltirish." },
        ],
      },
      {
        name: "Boshqaruv",
        items: [
          { name: "Hisobotlar va moliya", desc: "Savdo, kanal, ABC/XYZ, kassa smenasi, foyda-zarar; Excel'ga eksport." },
          { name: "Ega ilovasi", desc: "Bugungi raqamlar, ogohlantirishlar va ertalabki brifing telefonda." },
        ],
      },
    ],
    gallery: [
      { shot: "orders", alt: "Panel: buyurtmalar oqimi", label: "Buyurtmalar", kind: "browser" },
      { shot: "site", alt: "Restoranning o'z sayti", label: "Restoran sayti", kind: "browser" },
      { shot: "miniapp", alt: "Telegram mini app ichidagi menyu", label: "Telegram mini app", kind: "phone" },
      { shot: "courier", alt: "Kuryer ilovasi: yetkazish buyurtmasi", label: "Kuryer ilovasi", kind: "phone" },
    ],
  },

  live: {
    eyebrow: "04 · Ishlayotgan mahsulot",
    title: "KEEL — *ishlayotgan mahsulot*.",
    lead: "Bu konsepsiya yoki maket emas. Platforma ishlab chiqarish serverida ishlaydi: har yangi restoran alohida server va domen bilan ochiladi, kassa dasturi yuklab olinadi, platforma holati ochiq sahifada kuzatiladi.",
    stages: ["G'oya", "Prototip", "MVP", "Ishga tushirilgan"],
    links: [
      { href: "/", label: "keel.uz", desc: "Mahsulot sayti va narxlar" },
      { href: "/kassa", label: "Kassa", desc: "Kassa, zal va oshxona ekrani" },
      { href: "/download", label: "Kassa dasturi", desc: "Windows uchun yuklab olish" },
      { href: "/help", label: "Qo'llanma", desc: "Har bir modul bo'yicha maqolalar" },
      { href: "/status", label: "Platforma holati", desc: "Jonli uptime monitoringi" },
      { href: "/developers", label: "Ochiq API", desc: "Kalitlar, buyurtmalar, webhook'lar" },
    ],
    note: "Ko'rgazmali (demo) kirish hisobi ochiq e'lon qilinmaydi: sinab ko'rish uchun Telegram orqali yozing — sizga alohida restoran ochib beramiz.",
  },

  team: {
    eyebrow: "05 · Jamoa",
    title: "Kod ham, *restoran jarayoni* ham bitta qo'lda",
    lead: "KEEL'ni asoschi o'zi loyihalagan va yozgan: backend'dan tortib kassa dasturi, Android ilovalar va infratuzilmagacha.",
    responsibilities: [
      "Mahsulot va restoran jarayonlarini loyihalash",
      "Backend va API (Go, MongoDB)",
      "Veb: restoran sayti, admin panel, kassa interfeysi (Next.js, TypeScript)",
      "Windows kassa dasturi (Wails: Go + WebView2)",
      "Native Android ilovalar (Kotlin, Jetpack Compose)",
      "Infratuzilma: Docker, Caddy, ko'p tenantli deploy",
      "Integratsiyalar: to'lov, fiskal, POS, yetkazish",
    ],
    stack: [
      "Go", "TypeScript", "Next.js", "React", "Tailwind CSS", "MongoDB",
      "Kotlin", "Jetpack Compose", "Wails", "Docker", "Caddy", "Claude API",
    ],
    linksTitle: "Havolalar",
  },

  whyUs: {
    eyebrow: "06 · Nega aynan biz",
    title: "Va'da emas — *qilingan ish*",
    lead: "Bu bo'limdagi har bir dalilni yuqoridagi havolalar orqali tekshirish mumkin.",
    points: [
      {
        title: "Mahsulot allaqachon ishlaydi",
        desc: "Kassa, zal, oshxona, ombor, CRM, sayt, yetkazish va hisobotlar bitta kod bazasida yozilgan va ishga tushirilgan.",
      },
      {
        title: "Modullar haqiqatan bog'langan",
        desc: "Sotuv ombordan o'zi ayriladi, oshxonaning «tayyor»i ofitsiant telefoniga boradi, buyurtma mijoz tarixiga yoziladi — alohida dasturlarni bir-biriga ulash emas.",
      },
      {
        title: "Mahalliy bozor uchun qurilgan",
        desc: "Payme, Click, Uzum, ATMOS to'lovlari; fiskal kassa operatorlari; o'zbek, rus va ingliz tili; so'mda hisob; Yandex va 2GIS xaritalari.",
      },
      {
        title: "Internetsiz ishlaydi",
        desc: "Kassa sotuvni qurilmaning o'zida saqlaydi va aloqa qaytganda yuboradi; fiskal chek ham navbatda kutadi.",
      },
      {
        title: "Ko'chib o'tish oson",
        desc: "Mavjud kassani (iiko, Syrve, Poster, Clopos, r_keeper) almashtirish shart emas — KEEL sayt va yetkazish sifatida yonida ishlay oladi.",
      },
      {
        title: "Butun stekni o'zimiz qila olamiz",
        desc: "Server, veb, Windows dasturi, Android ilovalar va infratuzilma — tashqi pudratchisiz. Qaror va tuzatish orasida vaqt yo'qolmaydi.",
      },
    ],
  },

  roadmap: {
    eyebrow: "07 · Yo'l xaritasi",
    title: "Dasturiy ta'minot tayyor. *Keyingi qadam — bozor*",
    lead: "Sanalar ko'rsatilmagan — faqat qaysi bosqich bajarilgan va qaysi biri hozir ketayotgani.",
    stages: [
      { name: "G'oya", state: "done", desc: "Restoran jarayonlari bo'laklarga bo'linib ketgani — muammo aniqlandi." },
      { name: "Prototip", state: "done", desc: "Sayt, menyu va buyurtma oqimi — birinchi ishlaydigan versiya." },
      { name: "MVP", state: "done", desc: "Kassa, zal, oshxona ekrani, yetkazish va admin panel bitta tizimda." },
      { name: "Ishga tushirilgan", state: "done", desc: "Ko'p tenantli platforma: har restoran o'z serverida va domenida, Windows kassa dasturi, Android ilovalar." },
      { name: "Bozorda sinov va birinchi mijozlar", state: "current", desc: "Restoranlarni ulash, ularning fikri bilan mahsulotni tuzatish, onboarding'ni tezlashtirish." },
      { name: "O'sish", state: "next", desc: "Sotuv kanallari va hamkorlar, mijozning o'zi ro'yxatdan o'tishi va sinov muddati." },
      { name: "Kengayish", state: "next", desc: "Filial tarmoqlari va do'konlar uchun chuqurroq imkoniyatlar, yangi integratsiyalar." },
    ],
    stateLabel: { done: "Bajarildi", current: "Hozir", next: "Keyingi" },
  },

  tech: {
    eyebrow: "08 · Amalga oshirish",
    title: "Qanday *qurilgan*",
    lead: "Har bir restoran alohida server konteyneri va alohida ma'lumotlar bazasida ishlaydi: bir mijozdagi nosozlik yoki yuklama boshqasiga o'tmaydi.",
    archTitle: "Arxitektura",
    layers: [
      {
        name: "Mijoz ilovalari",
        items: ["Restoran sayti va admin panel (Next.js)", "Windows kassa (Wails)", "Android: ofitsiant, kuryer, ega, xodim, TV, mehmon", "Telegram bot va mini app"],
      },
      {
        name: "Chekka (edge)",
        items: ["Caddy: avtomatik HTTPS", "Har restoranga o'z domeni"],
      },
      {
        name: "Restoran serveri",
        items: ["Go REST API (chi)", "Har tenantga alohida konteyner", "JWT, rollar, PIN, qurilma bog'lash"],
      },
      {
        name: "Ma'lumotlar",
        items: ["MongoDB — har restoranga alohida baza", "Rasmlar — server diskida"],
      },
      {
        name: "Boshqaruv platformasi",
        items: ["Go: tenant ochish (Docker), billing, yangilanishlar", "Uptime monitoringi, SEO, AI xizmatlari"],
      },
    ],
    stagesTitle: "Ishlab chiqish bosqichlari",
    stages: [
      { name: "Asosiy operatsiyalar", desc: "Menyu, buyurtma, kassa, zal, oshxona.", state: "done" },
      { name: "Bog'langan modullar", desc: "Ombor va texkarta, CRM, hisobotlar, xodimlar — bitta ma'lumot ustida.", state: "done" },
      { name: "Tashqi integratsiyalar", desc: "To'lov, fiskal, POS, yetkazish, telefoniya, EDI va 1C.", state: "done" },
      { name: "Ishonchlilik va oflayn", desc: "Kassada oflayn navbat, fiskal navbat, avtomatik yangilanish, uptime sahifasi.", state: "done" },
      { name: "Tarqatish va onboarding", desc: "Menyuni import qilish, tez ulanish, o'zi ro'yxatdan o'tish.", state: "current" },
      { name: "Aqlli avtomatlashtirish", desc: "AI yordamchilarni chuqurlashtirish va ko'proq jarayonni avtomatlashtirish.", state: "next" },
    ],
    aiTitle: "Sun'iy intellekt — hozir nima ishlaydi",
    aiLead: "Qoida: raqamlarni restoranning o'z serveri hisoblaydi, model faqat ular haqida matn yozadi. Mijozning shaxsiy ma'lumoti modelga yuborilmaydi.",
    aiNow: [
      { name: "Ertalabki brifing", desc: "Ega uchun kechagi kun haqida qisqa xulosa — server hisoblagan raqamlar asosida." },
      { name: "Ega maslahatchisi", desc: "Ega o'z restorani haqida savol beradi, javob o'z raqamlaridan tuziladi." },
      { name: "Yordam assistenti", desc: "Qo'llanma maqolalari asosida javob beradi, operatorga o'tish doim mavjud." },
      { name: "Menyuni sahifadan o'qish", desc: "Eski sayt yoki menyu sahifasidan taomlarni taklif qiladi — ega ko'rib tasdiqlaydi." },
      { name: "Reklama rejasi", desc: "Qaysi taomni qanday byudjet bilan reklama qilishni taklif qiladi — qarorni ega qabul qiladi." },
    ],
    aiDev: "Ishlab chiqishda AI kod yordamchisi (Claude Code) ishlatiladi — kod, test va hujjat bilan birga.",
    aiPlanned: "Rejada: chuqurroq avtomatlashtirish (masalan, xarid va ombor bo'yicha tavsiyalar). Bular hali ishga tushirilmagan.",
    principlesTitle: "Texnik yechimlar",
    principles: [
      { name: "Oflayn kassa", desc: "Sotuv qurilmada saqlanadi, aloqa qaytganda yuboriladi; yuborilmagani panelda ko'rinadi." },
      { name: "Tenant izolyatsiyasi", desc: "Har restoran — alohida konteyner va baza; yangilanish restoranlar bo'yicha bosqichma-bosqich." },
      { name: "Sirlar alohida", desc: "To'lov kalitlari va tokenlar ochiq profildan tashqarida saqlanadi va hech qachon qaytarilmaydi." },
      { name: "Uch til", desc: "Sayt, panel, kassa va ilovalar o'zbek, rus va ingliz tilida; server xabarlari ham tarjima qilinadi." },
      { name: "Ochiq API", desc: "API kalitlar va imzolangan webhook'lar — buxgalteriya va boshqa tizimlar ulanishi uchun." },
    ],
  },

  integrations: {
    eyebrow: "09 · Integratsiyalar",
    title: "Mahalliy bozor bilan *ulangan*",
    lead: "Bular rasmiy hamkorlik emas — ushbu xizmatlarning ochiq API'lari uchun KEEL tomonidan yozilgan ulagichlar.",
    groups: [
      {
        state: "built",
        label: "Kodda yozilgan",
        items: [
          { name: "To'lov", list: "Payme, Click, Uzum, ATMOS" },
          { name: "Fiskal kassa", list: "Multikassa, REGOS, E-POS" },
          { name: "Tashqi kassalar", list: "iiko / Syrve, Poster, Clopos, r_keeper" },
          { name: "Telefoniya va xabarlar", list: "OnlinePBX, SMS, Telegram bot" },
          { name: "Buxgalteriya", list: "Didox (ЭСФ), 1C almashinuvi" },
          { name: "Xaritalar", list: "2GIS, Yandex, Google" },
        ],
      },
      {
        state: "testing",
        label: "Ishlab chiqilmoqda / sinov kutilmoqda",
        items: [
          { name: "Marketpleys", list: "Uzum Tezkor — provayder sinov muhitida tekshirilmagan" },
          { name: "Yetkazish xizmatlari", list: "Yandex yetkazish, BTS" },
          { name: "Reklama", list: "Meta (Facebook / Instagram) — jonli akkauntda hali ishlatilmagan" },
          { name: "Markirovka", list: "Asl Belgisi — ayrim talablar tasdiqlanmoqda" },
        ],
      },
      {
        state: "planned",
        label: "Rejada",
        items: [
          { name: "Boshqa kassalar", list: "Paloma365, Jowi — hujjat kutilmoqda" },
        ],
      },
    ],
  },

  demo: {
    eyebrow: "10 · Demo",
    title: "Demo video",
    placeholderTitle: "Demo video shu yerga joylanadi",
    placeholderLead: "Yakuniy video (1–5 daqiqa) tayyorlanmoqda.",
    play: "Videoni ijro etish",
    aboutTitle: "Demo haqida",
    aboutLead: "Videoda bitta buyurtma restoranning boshidan oxirigacha qanday o'tishi ko'rsatiladi — haqiqiy KEEL ekranlarida, sinov restoranida.",
    aboutSteps: [
      "Mehmon stolga o'tiradi — zal xaritasida stol ochiladi",
      "Taomlar kassa chekiga qo'shiladi va oshxonaga yuboriladi",
      "Oshxona ekranida chek paydo bo'ladi, oshpaz «tayyor» bosadi",
      "To'lov va fiskal chek",
      "Ombor: sotilgan taom masalliqlari qoldiqdan ayrilgani",
      "Mijoz kartochkasi va keshbek",
      "Hisobotlar: tushum, tannarx, foyda",
      "Onlayn buyurtma: sayt yoki Telegram bot orqali — xuddi shu menyudan",
    ],
  },

  liveCta: {
    eyebrow: "11 · Mahsulot",
    title: "Ishlayotgan KEEL mahsulotini ko'rish",
    lead: "keel.uz — mahsulot sayti; u yerdan kassa, qo'llanma va platforma holatiga o'tish mumkin.",
    button: "Ishlayotgan KEEL mahsulotini ko'rish →",
    telegram: "Telegramda yozish",
  },

  finale: {
    title: "Biznes uchun 10 ta alohida tizim emas.\n*Bitta KEEL.*",
    lead: "KEEL — biznesingizni bitta tizimda boshqaring.",
  },
} as const;
