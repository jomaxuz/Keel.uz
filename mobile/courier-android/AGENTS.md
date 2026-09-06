# Keel Courier — native Android ilovasi (Kotlin + Compose)

Loyihaning umumiy qoidalari — ildizdagi `CLAUDE.md`. Bu ilovaning o'z qarorlari
`README.md` da; server tomoni `docs/DECISIONS.md` → «Kuryerlar va rollar»,
«Kuryer PWA», «Joylashuvga ruxsat».

⚠️ **Ikki narsa mijoz oldida buziladi, va faqat shu ikkitasi.** «Yetkazildi»
tugmasi (`location/Gate.kt`) va joylashuv oqimi (`location/LocationService.kt`).
Qolgan hamma narsani kuryer qayta bosib tuzatadi; bu ikkitasi esa eshik oldida,
mijoz qarab turganda bilinadi. O'zgartirishdan oldin ikkalasining boshidagi
izohni o'qing.

⚠️ **Gate serverning nusxasi** (`internal/handlers/courier.go` →
`arrivalBlocked`). Formulani, radiusni yoki muddatni bu yerda «yaxshilash» —
tugmasi ochiq, keyin server rad etadigan ilova demakdir. Ilova serverdan
qattiqroq bo'lishi mumkin, yumshoqroq emas.

⚠️ **Dizayn `../android-design` dan keladi va u yerda to'rtta ilova bo'lishadi.**
Rangni yoki shishani bu yerga nusxalash — ajralishning boshlanishi.

⚠️ **Har bir `data/Models.kt` maydoni — panel tiplarining ikkinchi nusxasi.**
Nomi o'zgarsa kotlinx xato bermaydi, standart qiymat qo'yadi — ya'ni kuryer naqd
pul yig'adigan kartada nol. Yangi maydon qo'shsangiz `WireShapeTest` ga Go
handler'idan **ko'chirilgan** JSON bilan qator qo'shing.
