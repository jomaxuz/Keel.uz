# Keel Guest — mehmon ilovasi (Kotlin + Compose)

Loyihaning umumiy qoidalari — ildizdagi `CLAUDE.md`. Bu ilovaning o'z qarorlari
`README.md` da; server tomoni `docs/DECISIONS.md` → «Restoranning o'z ilovasi».

⚠️ **Bu ilovani mehmon ko'radi.** Xodim ilovalaridagi «bo'lim ishlatilmasa
e'tibordan qoladi» qoidasi bu yerda kuchliroq: ishlatilmaydigan ekran emas,
**o'rnatilmagan ilova** bo'ladi.

⚠️ **Bosh sahifa yo'q va bo'lmaydi.** Birinchi ekran — menyu. Sabab `README.md`
da yozilgan; uni qayta muhokama qilishdan oldin o'sha xatboshini o'qing.

⚠️ **Brendlanadigan hamma narsa `app/brand.properties` da.** Rangni, nomni yoki
server manzilini boshqa faylga yozsangiz, quvur uni yangilay olmaydi va o'sha
qiymat butun parkda Keel'niki bo'lib qoladi.

⚠️ **`KeelWaiterTheme(accent = ...)` faqat shu ilovada beriladi.** Beshta xodim
ilovasi hech nima bermaydi va Keel apelsin rangida qoladi — kassa ekrani bosilgan
chek bilan mos kelishi kerak.

⚠️ **Dizayn `../android-design` dan keladi va uni oltita ilova bo'lishadi.**
Rangni yoki shishani bu yerga nusxalash — ajralishning boshlanishi.

⚠️ **Har bir `data/Models.kt` maydoni — sayt tiplarining ikkinchi nusxasi**
(`frontend/src/lib/types.ts`). Nomi o'zgarsa kotlinx xato bermaydi, standart
qiymat qo'yadi — ya'ni menyudagi hamma taom bepul bo'lib ko'rinadi. Yangi maydon
qo'shsangiz testga Go handler'idan **ko'chirilgan** JSON bilan qator qo'shing.

⚠️ **Mehmondan menyudan oldin hech nima so'ralmaydi** — na server manzili, na
telefon raqami. Menyu, kategoriya va taom **ochiq**; token faqat o'z buyurtmasi
va ballarini qo'shadi.

⚠️ **Hisob ixtiyoriy va shunday qoladi.** Menyu, savat va buyurtma berish
hisobsiz ishlaydi; kirish faqat tarix, ballar va sevimlilarni qo'shadi. Biror
ekran hisob talab qila boshlasa — bu mahsulot qarorini bekor qilish, va u
`docs/DECISIONS.md` da yozilgan.

⚠️ **Push uchun `google-services.json` qo'shmang.** Plagin `applicationId` ga
mos mijoz yozuvi bo'lmagan har qanday build'ni rad etadi — har restoranga
alohida build modelida bu ishlamaydi. Firebase to'rtta satrdan kodda
sozlanadi (`push/Push.kt`), va satrlar `brand.properties` da.

⚠️ **Bo'sh kalit — o'chirilgan imkoniyat, qulash emas.** Xarita kaliti ham,
Firebase qiymatlari ham bo'sh bo'lishi mumkin: ilova ishlaydi, faqat o'sha bitta
narsani qilmaydi.
