# Keel Owner — native Android ilovasi (Kotlin + Compose)

Loyihaning umumiy qoidalari — ildizdagi `CLAUDE.md`. Bu ilovaning o'z qarorlari
`README.md` da; server tomoni `docs/DECISIONS.md` → «Dashboard statistikasi»,
«Hisobotlar», «Mehmonlar fikri», «Qo'llab-quvvatlash: chat va operator konsoli».

⚠️ **Bu panel emas.** Panel — o'tirib qaror qabul qilinadigan joy; bu esa
kuzatish va javob berish. Ekran qo'shishdan oldingi savol «panelda bormi?» emas,
**«telefonni ushlab turgan odam buni hozir hal qila oladimi?»**.

⚠️ **Dizayn `../android-design` dan keladi va u yerda beshta ilova bo'lishadi.**
Rangni, shishani yoki `TokenStore` ni bu yerga nusxalash — ajralishning
boshlanishi. Waiter ilovasida aynan shunday nusxa qolib ketgan edi va uni hech
kim ishlatmasdi (2026-09-06 da o'chirildi).

⚠️ **Har bir `data/Models.kt` maydoni — panel tiplarining ikkinchi nusxasi.**
Nomi o'zgarsa kotlinx xato bermaydi, standart qiymat qo'yadi — ya'ni ega
qaraydigan ekranda ishonchli nol. Yangi maydon qo'shsangiz `WireShapeTest` ga Go
handler'idan **ko'chirilgan** JSON bilan qator qo'shing; o'z tomondan yozilgan
fixture o'z tomoni bilan doim kelishadi.
