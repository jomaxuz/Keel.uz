# Keel TV — native Android TV ilovasi (Kotlin + Compose)

Loyihaning umumiy qoidalari — ildizdagi `CLAUDE.md`. Bu ilovaning o'z qarorlari
`README.md` da; server tomoni `docs/DECISIONS.md` → "TV ekranlar".

⚠️ **Bu — hech kim qo'lida ushlamaydigan yagona ilova.** Qolgan to'rttasini odam
ushlaydi va xatoga javob bera oladi; bu esa mehmonlar to'la xonada, bosh
balandligidan yuqorida osilgan. Shundan kelib chiqadigan qoidalar
`MainActivity.kt` ning boshida yozilgan — o'zgartirishdan oldin o'sha yerni
o'qing.

⚠️ **Dizayn `../android-design` dan keladi va u yerda uchta ilova bo'lishadi.**
Rangni, shishani yoki shrift o'lchamini shu papkaga nusxalash — ajralishning
boshlanishi. O'lcham esa bu yerniki: telefon 54dp tugmasi bilan televizor
tugmasi bir narsa emas.

⚠️ **Har bir `data/Models.kt` maydoni — panel tiplarining ikkinchi nusxasi.**
Nomi o'zgarsa kotlinx xato bermaydi, standart qiymat qo'yadi — ya'ni devorda
bo'sh playlist. Yangi maydon qo'shsangiz `WireShapeTest` ga Go handler'idan
**ko'chirilgan** JSON bilan qator qo'shing.
