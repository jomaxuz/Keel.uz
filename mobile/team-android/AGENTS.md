# Keel Team — native Android ilovasi (Kotlin + Compose)

Loyihaning umumiy qoidalari — ildizdagi `CLAUDE.md`. Bu ilovaning o'z qarorlari
`README.md` da; server tomoni `docs/DECISIONS.md` → «Ishchilar davomati»,
«Bozorlik: bozorchi ilovadan yozadi», «Bozorlik ro'yxati», «Podotchet».

⚠️ **Bu ilovada uchta raqam bor va uchalasi ham boshqa joyni buzadi.** Soatlar
(oylik), bozorlikdagi narx (butun menyu tannarxi) va miqdor (ombor qoldig'i).
Ularning har biriga tegishli izohlar kod ichida — o'zgartirishdan oldin o'qing.

⚠️ **Hisob-kitob bu yerda emas.** `/staff/report` javobi ikkinchi marta
chiziladi, ikkinchi marta hisoblanmaydi: telefon o'z soatlarini qo'shsa, farq
oylik kuni topiladi.

⚠️ **Tab ruxsat bilan so'raladi, rol nomi bilan emas** (`canWriteHere`). Rol
nomining imlosi hech nima bermaydi — `models/staffrole.go`.

⚠️ **Dizayn `../android-design` dan keladi va u yerda beshta ilova bo'lishadi.**
Rangni yoki shishani bu yerga nusxalash — ajralishning boshlanishi.

⚠️ **Har bir `data/Models.kt` maydoni — panel tiplarining ikkinchi nusxasi.**
Nomi o'zgarsa kotlinx xato bermaydi, standart qiymat qo'yadi — ya'ni odam o'z
oyligini tekshiradigan ekranda nol. Yangi maydon qo'shsangiz `WireShapeTest` ga
Go handler'idan **ko'chirilgan** JSON bilan qator qo'shing.
