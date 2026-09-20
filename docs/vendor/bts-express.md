# BTS Express (bts.uz) — pochta/kuryer integratsiyasi

**Manba:** <https://bts.uz/> · kabinet <https://new.bts.uz/> · `https://api.bts.uz/`
**Qidirilgan sana:** 2026-09-20.

---

## ⚠️ Eng muhimi: API hujjati **ochiq e'lon qilinmagan**

Bu fayl odatdagi `docs/vendor/` yozuvlaridan farq qiladi: qolganlari
**o'qilgan hujjatning nusxasi**, bu esa **nima yo'qligining** yozuvi. Shuning
uchun u shu yerda turadi — aks holda keyingi sessiya xuddi shu qidiruvni
boshidan takrorlaydi va xuddi shu joyga keladi.

Tekshirilgani (2026-09-20):

| Manzil | Natija |
|---|---|
| `bts.uz/robots.txt`, `sitemap.xml` | 40+ sahifa, **birortasi ham API/developer emas** |
| `bts.uz/ru/e-commersiya` | 404 (footerdagi havola ishlamaydi) |
| `new.bts.uz` (shaxsiy kabinet) | Yii2, **server-rendered**; JSON API yo'q, jQuery formalar |
| `api.bts.uz/` va uning ostidagi har bir yo'l (`/swagger`, `/swagger.json`, `/openapi.json`, `/docs`, `/api/v1`, `/health`) | hammasi **bir xil** `<title>BTS.UZ</title>` + `noindex` bo'sh HTML — ya'ni catch-all, hujjat emas |
| `cabinet.bts.uz` | mavjud emas |
| Web qidiruv (uz/ru/en, GitHub, npm) | BTS Express uchun **hech qanday ochiq API hujjati yo'q** |

Ya'ni: **`api.bts.uz` bor, lekin u shartnoma va kalitsiz hech nima aytmaydi.**
Hujjatni BTS'ning o'zidan olish kerak (`info@bts.uz`, 1230 / +998 71 207-08-09).

## Shundan kelib chiqqan qaror

Integratsiya **`delivery_provider` ning `kind: "api"`** ichiga qo'yildi —
Yandex Delivery bilan bir tokchaga, chunki savol bir xil: "buyurtmani tashqi
xizmatga topshir, keyin holatini so'ra". Yangi kolleksiya ham, yangi ekran ham
qo'shilmadi.

⚠️ **Prod manzil kodda yo'q va ataylab yo'q.** Yandex'niki e'lon qilingan
(`b2b.taxi.yandex.net`), BTS'niki emas — va **taxmin qilingan manzil eng yomon
turdagi xato** bo'lardi: forma to'ldirilgandek ko'rinadi, saqlanadi, hech nima
xato bermaydi, birinchi buyurtma esa hech qayerga ketmaydi. Shuning uchun
`apiBaseUrl` **majburiy**: bo'sh bo'lsa panel "BTS uchun API manzili
kiritilmagan" deb aytadi va hech nima yubormaydi.

⚠️ **Wire format ham shartnomadan keladi.** `bts.go` dagi so'rov/javob
maydonlari — kuryer xizmatlarining odatiy shakli (`token` sarlavhasi,
`create` / `status` / `cancel`), lekin ular **tasdiqlanmagan**. Mijozni
yoqishdan oldin BTS bergan hujjat bilan solishtirib chiqish kerak; farq bo'lsa
tuzatiladigan joy **bitta fayl** (`backend/internal/delivery/bts.go`), chunki
handler qatlami `delivery.Service` interfeysi orqali gaplashadi.

## BTS'dan so'raladigan to'rt narsa

1. **Base URL** — prod va sandbox (agar bo'lsa).
2. **Autentifikatsiya** — token qanday sarlavhada keladi (`Authorization: Bearer`,
   `token:`, `X-API-KEY`?) va u qayerdan olinadi.
3. **Buyurtma yaratish** — majburiy maydonlar (jo'natuvchi, oluvchi, manzil
   formati — viloyat/tuman kodi kerakmi yoki matn yetarlimi, og'irlik, e'lon
   qilingan qiymat, naqd to'lov / COD bormi).
4. **Holat** — qaytadigan holat kodlari ro'yxati va nakladnoy (trek) raqami
   qaysi maydonda.

⚠️ **COD (yetkazishda to'lov) alohida so'raladi.** Onlayn do'konlarning
ko'pchiligi O'zbekistonda aynan shunday sotadi, va agar BTS pulni o'zi yig'ib
keyin o'tkazsa, u **tushum emas, `payout`** (qarang CLAUDE.md → "Kelmagan pul").
Buni shartnoma aytib beradi; taxmin qilinmaydi.
