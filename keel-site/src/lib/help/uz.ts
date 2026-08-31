// The knowledge base, in Uzbek — and Uzbek is the source of truth.
//
// ⚠️ **Everything here is written from the behaviour in this repository**, not
// from what the product ought to do: the staff radius is 50 metres because that
// is the default in `EnsureStaffDefaults`, an order goes to the till on
// *confirm* and not on create because that is where `posorder.go` sends it, and
// a prep card prices at nothing without a yield because `ratesAt` divides by
// it. An article that is wrong is worse than no article: the person reading it
// is standing at a counter deciding whether to trust the software with their
// evening.
//
// ⚠️ **Say what breaks, not "be careful".** Almost every warning below is a
// mistake somebody has actually made, and the ones that matter most are the
// silent ones — the settings that look filled in, produce no error, and are
// wrong somewhere else entirely.

import type { HelpContent } from "./types";
import { articlesUz } from "./uz.articles";

export const helpUz = {
  ui: {
    title: "Bilim bazasi",
    lead: "Keel bilan ishlashning to'liq qo'llanmasi: sozlash, kundalik ish va nima buzilganda nima qilish kerak.",
    searchPlaceholder: "Qidirish: texkarta, printer, zona…",
    searchEmpty:
      "«{q}» bo'yicha hech nima topilmadi. Boshqa so'z bilan urinib ko'ring yoki bizga yozing.",
    searchCount: "{n} ta maqola",
    allArticles: "Barcha maqolalar",
    inSection: "Bo'lim",
    back: "Orqaga",
    next: "Keyingisi",
    prev: "Oldingisi",
    seeAlso: "Shuni ham o'qing",
    figureHint: "Rasmni kattalashtirish uchun bosing.",
    notFound: "Maqola topilmadi",
    notFoundLead: "Bunday manzil yo'q. Bilim bazasining boshiga qayting.",
    askUs: "Javob topilmadimi?",
    askUsLead:
      "Telegramda yozing — odam javob beradi. Qaysi ekranda ekaningizni va nima kutganingizni yozsangiz, javob birinchi xabardayoq keladi.",
    updated: "Yangilangan",
  },

  sections: [
    {
      id: "start",
      title: "Boshlash",
      lead: "Birinchi kun: panelga kirish, filial, ish vaqti va kim nimani ko'radi.",
    },
    {
      id: "site",
      title: "Sayt",
      lead: "Mehmon ko'radigan sayt: ko'rinish, matnlar, rasmlar va topilishi.",
    },
    {
      id: "menu",
      title: "Menyu",
      lead: "Kategoriya, taom, variant, to'plam, aksiya va stop list.",
    },
    {
      id: "orders",
      title: "Buyurtmalar",
      lead: "Buyurtma kelganidan berilgunicha: holatlar, ovoz, bekor qilish, bron.",
    },
    {
      id: "delivery",
      title: "Yetkazib berish",
      lead: "Xarita, zonalar, narx hisobi va kuryerlar.",
    },
    {
      id: "till",
      title: "Kassa va zal",
      lead: "Kassa dasturi, smena, stollar, oshxona ekrani va kiosk.",
    },
    {
      id: "printers",
      title: "Printerlar",
      lead: "Chek va oshxona printerlarini ulash, va chek chiqmasa nima qilish.",
    },
    {
      id: "stock",
      title: "Ombor va tannarx",
      lead: "Masalliq, texkarta, kirim, chiqim, inventarizatsiya va qoldiq.",
    },
    {
      id: "team",
      title: "Xodimlar",
      lead: "Ishchi qo'shish, rollar va ruxsatlar, davomat va oylik.",
    },
    {
      id: "customers",
      title: "Mijozlar",
      lead: "Baza, ballar, segmentlar, xabar yuborish va fikrlar.",
    },
    {
      id: "integrations",
      title: "Integratsiyalar",
      lead: "To'lov, SMS, Telegram, kassa tizimlari, ATS, fiskal va markirovka.",
    },
    {
      id: "reports",
      title: "Hisobotlar",
      lead: "Nima sotildi, qancha foyda, qayerda kamomad.",
    },
    {
      id: "settings",
      title: "Sozlamalar",
      lead: "Til, tema, xavfsizlik, ma'lumotni olib ketish va obuna.",
    },
  ],

  articles: articlesUz,
} satisfies HelpContent;
