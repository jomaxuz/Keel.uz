# Rahmat POS (Multikassa) — integratsiya bo'yicha savollar

Bu — provayderga yuboriladigan xat uchun tayyor ro'yxat. Har bir savol yonida
**nega so'ralayotgani** va **javob nimani o'zgartirishi** yozilgan, chunki
xatni ko'pincha bizdan boshqa odam yuboradi.

Manbalarimiz: integratorlar uchun PDF (`multikassa-operations-api_fixed_refund_extrainfo_v2.pdf`,
2026-04) va ikkita ochiq Postman kolleksiyasi
(`Multikassa.Pos - интеграция`, `Multibank.Касса`).

---

## 1. ⚠️ Pul birligi — eng muhim savol

**Savol:** `POST /api/v1/operations` da (`module_operation_type: 3`)
`receipt_sum`, `receipt_gnk_receivedcash`, `receipt_gnk_receivedcard`
maydonlari **tiyinda** yuboriladimi yoki **so'mda**? `items[]` ichidagi
`product_price` / `total_product_price` / `product_discount` uchun ham
tasdiqlashingizni so'raymiz.

**Nega so'raymiz:** hujjatlaringiz zid.

- **PDF, 4-bo'lim** aniq yozadi: `receipt_gnk_receivedcash`,
  `receipt_gnk_receivedcard`, `receipt_sum` — **ТИЙИН**; `items[]` ичидагилар —
  **СУМ**.
- **Postman namunasi** esa arifmetik jihatdan ikkalasini bir birlikda
  ko'rsatadi: `70000 + 70000 + 150000`, birinchisiga `discount_percent: 40`
  → `42000 + 70000 + 150000 = 262000` = `receipt_sum`.
- Javoblaringiz ham aralash: `GET /api/v1/zReport` tiyin butun son qaytaradi
  (`"totalSaleCash": "665102000"`), smena yopish esa formatlangan so'm
  (`"totalSaleCash": "6,651,020.00"`).

**Nima o'zgaradi:** xato bo'lsa fiskal chekdagi summa **100 barobar** noto'g'ri
bo'ladi. Biz PDF'ga ergashdik, lekin ishga tushirishdan oldin tasdiq kerak.

---

## 2. ⚠️ CORS — arxitekturani hal qiladi

**Savol:** kassa dasturining `/api/v1/*` endpointlari brauzerdan chaqirilganda
`Access-Control-Allow-Origin` sarlavhasini qaytaradimi? `OPTIONS` (preflight)
so'roviga javob beradimi? Agar ha bo'lsa, qaysi originlarga?

**Nega so'raymiz:** bizning kassa ekranimiz — veb-sahifa. U kassa dasturiga
so'rov yubora oladi, lekin **javobni o'qish** uchun brauzerga CORS sarlavhasi
kerak. Sarlavha bo'lmasa: chek fiskallashadi, biroq **fiskal belgi va QR bizga
qaytmaydi** — ya'ni mehmonga ko'rsatadigan narsamiz qolmaydi.

**Nima o'zgaradi:** javob "ha" bo'lsa, kassa ekranidan to'g'ridan-to'g'ri
ishlaymiz. "Yo'q" bo'lsa, kassa kompyuteriga o'zimizning kichik ulagichimiz
o'rnatiladi (allaqachon yozilgan) — bu ishlaydi, lekin har restoranda
o'rnatish talab qiladi.

---

## 3. Sinov muhiti

**Savol:** integratsiyani tekshirish uchun sinov (test) fiskal moduli yoki
sinov terminali bera olasizmi? Shartnoma imzolashdan oldin sinash imkoni
bormi?

**Nega so'raymiz:** yuqoridagi ikki savolning javobini **jonli restoranda,
haqiqiy cheklar bilan** tekshirish — noto'g'ri fiskal hujjatlar demakdir.

---

## 4. Smena boshqaruvi

**Savol:** sotuvdan oldin smena (`module_operation_type: 1`) **majburiy**
ochilishi kerakmi? Biz uni o'zimiz ochishimiz kerakmi, yoki kassa dasturining
o'zi ochadimi? Bir kunda bir necha marta ochish/yopish mumkinmi?

**Nega so'raymiz:** Postman'dagi xato javobingiz `#2D - Z-отчет не был открыт`
— ya'ni smena ochilmagan bo'lsa sotuv rad etiladi. Bizda ham o'z kassa
smenamiz bor va ikkalasini bog'lash kerak.

---

## 5. Xato kodlari ro'yxati

**Savol:** `#2D`, `#2B`, `#500` kabi kodlarning to'liq ro'yxati bormi?

**Nega so'raymiz:** kassirga "nima qilish kerak" deb aytish uchun. `#2D` —
"smenani oching", ya'ni o'n soniyalik ish; boshqa kodlar esa qo'ng'iroq
talab qilishi mumkin. Ularni ajrata olmasak, hammasi bir xil "xato" bo'lib
ko'rinadi.

---

## 6. Qaytarish (refund)

**Savol:** `module_operation_type: 4` uchun qaysi maydonlar **majburiy**?
PDF `RefundInfo.ReceiptSaleId` ni ko'rsatadi, Postman namunasi esa
`receipt_gnk_receiptseq` + `receipt_gnk_fiscalsign` + `receipt_gnk_time`
yuboradi. Ikkalasi ham ishlaydimi?

**Nega so'raymiz:** sotuv javobida `ReceiptSaleId` degan maydon ko'rinmadi —
faqat `receipt_gnk_fiscalsign`, `receipt_gnk_receiptseq` va
`receipt_gnk_time` bor. Qaytarish uchun nimani saqlashimiz kerakligini
bilishimiz kerak, va uni **sotuv paytida** saqlash kerak.

---

## 7. ИКПУ va qadoq kodi maydonlarining nomi

**Savol:** `ikpu` / `packageCode` (PDF) va `classifier_class_code` /
`product_package` (Postman) — ikkalasi ham qabul qilinadimi, yoki bittasi
eskirganmi?

**Nega so'raymiz:** hozircha ikkala nom bilan **bir xil qiymat** yuboryapmiz
(ular bir xil son, shuning uchun bu xavfsiz). Lekin qaysi biri to'g'ri
ekanini bilsak, ortiqcha maydonni olib tashlaymiz.

---

## 8. Chegirma

**Savol:** qator chegirmasi `product_discount` (summa) bilan yuborilsinmi
yoki `discount_percent` (foiz) bilan? Ikkalasi birga yuborilsa nima bo'ladi?

**Nega so'raymiz:** ikkovi ham yuborilsa chegirma **ikki marta** ayirilishi
mumkin. Hozircha faqat `product_discount` yuboryapmiz (PDF bo'yicha).

---

## 9. Integratsiya rejimi

**Savol:** `POST /api/fiscal/tsc/edit_user_modules` (`integration_mode: true`)
— bizning integratsiyamiz ishlashi uchun bu **oldindan yoqilishi** shartmi?
Uni restoran o'z kabinetidan yoqadimi yoki siz yoqasizmi?

**Nega so'raymiz:** bu bulut API'sida ko'rindi, lekin lokal kassa dasturiga
qanday ta'sir qilishi yozilmagan. Restoranga "avval kabinetdan buni yoqing"
deb aytishimiz kerak bo'lsa, buni oldindan bilishimiz kerak.

---

## 10. Tarif

**Savol:** integratsiya orqali ishlaganda tarif o'zgaradimi? Bir restoranda
bir nechta kassa bo'lsa narx qanday hisoblanadi?

**Nega so'raymiz:** ochiq manbalarda 123 600 so'm/oy raqamini ko'rdik, lekin
uni tasdiqlashimiz kerak — bu restoranga aytadigan narximizning bir qismi.
