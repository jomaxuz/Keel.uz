# Uzum Tezkor — Retail API (партнёрская интеграция)

Manba: https://flash-longship-af1.notion.site/Uzum-Tezkor-Retail-API-3eb48f484ac84504808078ae59c89e1f
(o'qildi 2026-09-14, Uzum Tezkor bizning 2026-09 dagi xatimizga javoban bergan).
⚠️ Notion SPA — `curl`/WebFetch bo'sh qaytaradi; brauzerda, yig'ilgan
bloklarni ochib o'qildi.

Spetsifikatsiya: `Uzum Tezkor Grocery API.yml` (OpenAPI 3.0.1, `title: API
YGrocery`, 2019 qator) — sahifaga ilova, **o'qildi 2026-09-14**. Maydonlar
quyida shu fayldan. ⚠️ Fayl **Yandex Eda vendor API** ning nusxasi
(`application/vnd.eda.picker.*`, `vnd.eats.order.*`, sxema nomlari
`YGrocery*`, misol telefon `+7903…`) — ya'ni Yandex Eats integratsiyasi
yozilsa, deyarli shu kontrakt bo'ladi.

## Model: ular **bizni** so'raydi (pull)

Uzum Tezkor **bizning** serverimizga murojaat qiladi: katalog va qoldiqni
o'zi oladi, buyurtmani bizga POST qiladi, holatini o'zi so'raydi. Ya'ni biz
server tomonini yozamiz, ular klient.

- Bitta **umumiy host** (faqat domen, IP emas) hamkor sozlamasiga yoziladi;
  har so'rovda **bizning nuqta ID'miz** (`placeId` / restoran) keladi.
- Nuqtalar ro'yxati menejer orqali **qo'lda** beriladi (nomi, bizning ID,
  manzil). `GET /restaurants` faqat boshlang'ich tekshiruv uchun, majburiy emas.
- Nuqtani yoqish/o'chirish — ularning ish grafigi va yetkazish zonasi
  grafigi (kabinet/menejer), bizning API emas.

## Autentifikatsiya — OAuth2 **bizda**

- Biz OAuth2 client_credentials yozamiz va ularga `host`, `client_id`,
  `client_secret` beramiz.
- `POST /security/oauth/token`, body `application/x-www-form-urlencoded`:
  `client_id`, `client_secret`, `grant_type=client_credentials`,
  `scope="read write"` → `access_token`.
- Keyin har so'rovda `Authorization: Bearer <token>`; token har
  so'rovlardan oldin olinadi.

## `v1` prefiksi

Hujjatda katalog/qoldiq `v1` bilan, buyurtma va token `v1` siz. Host bitta
bo'lgani uchun **ikkalasini ham** qo'llash kerak (yoki hammasini `v1` bilan:
`POST {HOST}/v1/security/oauth/token`, `POST {HOST}/v1/order`).

## Endpointlar (bizning serverda)

| Metod | Yo'l | Nima |
|---|---|---|
| POST | `/security/oauth/token` | token |
| GET | `/v1/nomenclature/{placeId}/composition` | katalog: kategoriyalar, tovarlar (nom, tavsif, narx, barkod, vazn, hajm) |
| GET | `/v1/nomenclature/{placeId}/availability` | qoldiq (`stock`): `<= 0` — sotilmaydi, `> 0` — sotiladi; **ro'yxatda yo'q tovar — sotilmaydi** |
| POST | `/order` | buyurtma yaratish (mijoz, manzil, pozitsiyalar) → bizning `orderId` |
| GET | `/order/{orderId}` | buyurtma to'liq (`YGroceryOrderV2`); tarkibi o'zgarsa — yangilash |
| GET | `/order/{orderId}/status` | ⚠️ **faqat YAML'da**, Notion sahifasida yo'q: `OrderStatus` |
| PUT | `/order/{orderId}` | ular tomonidan yangilash (do'konlarda kam, kelishuv bilan) |
| DELETE | `/order/{orderId}` | ular tomonidan bekor qilish (sabab bo'sh bo'lishi mumkin) |
| GET | `/restaurants` | nuqtalar ro'yxati (ixtiyoriy) |

- Content-Type: modelda «aktual» deb belgilangani, masalan buyurtma uchun
  `application/vnd.eats.order.v2+json`.
- Javoblar: 200; 400 (validatsiya xatolari ro'yxati); 401 (token yo'q/eskirgan);
  404 (restoran topilmadi); 500. Xatoda `description` ga **matnli sabab**.
- Spetsifikatsiyadan chetga chiqqan javob tanasi ularning validatsiyasidan
  o'tmaydi — turlar va massivdagi majburiy maydonlar qat'iy.

## Chastota

- Katalog (narxlar bilan birga) — soatiga 1 (kelishuv bilan o'zgaradi).
- Qoldiq — 5 daqiqada 1.
- Buyurtma holati — yaratilgandan keyin daqiqada 1.

## Buyurtma

- ⚠️ **15 daqiqa ichida `ACCEPTED_BY_RESTAURANT` bo'lmasa, Uzum buyurtmani
  bekor qiladi.**
- ⚠️ **Qayta yuborish**: 4xx/5xx olsa, bir necha marta qayta yuboradi.
  Dublikat `eatsId` (ularning ID) bo'yicha aniqlanadi va unga **birinchisi
  bilan bir xil** javob: `200`, o'sha `orderId`, o'sha `eatsId`.
- Holatlar: `NEW`, `ACCEPTED_BY_RESTAURANT`, `POSTPONED`, `COOKING`, `READY`,
  `TAKEN_BY_COURIER`, `DELIVERED`, `CANCELLED`.
- Oqim faqat oldinga: `NEW → ACCEPTED_BY_RESTAURANT → COOKING → READY →
  TAKEN_BY_COURIER → DELIVERED`. `CANCELLED` — istalgan bosqichda;
  `POSTPONED` — yakuniy bo'lmagan istalgan bosqichda.
- Bekor qilish: ular — `DELETE`, biz — holatni `CANCELLED` qilib.

## Vaznli tovar

`items.price` — `items.measure.value` vazni uchun narx; `items.measure.quantum`
— savatga qo'shish qadami (0 emas); vaznli tovarda `isCatchWeight`
majburiy.

## Buyurtmani yangilash (qisman bekor / almashtirish)

- Bizdan: `GET /order/{orderId}` da **boshqa tarkib** qaytarsak — yangilash
  sifatida o'qiladi (masalan tortganda vazn kam chiqdi). Kelishuv bilan
  yoqiladi; yoqilsa tasdiqdan keyin buyurtma holat bilan bir chastotada
  so'raladi.
- Narx — katalog parsingidagi narx.
- ⚠️ O'zgarishlarni **bir marta, jamlab** qaytarish kerak (masalan yig'ish
  tugashidan oldin): mijoz har yangilanish uchun push va tranzaksiya ko'radi.

## Hujjatda **yo'q**

- ⚠️ **Moliyaviy ma'lumot** (perechisleniye reyestri, komissiya, zachislenie
  sanasi) — xatimizdagi asosiy uchinchi so'rov. Bu API faqat katalog, qoldiq
  va buyurtma.
- Stop-listni **bizdan** push qilish yo'q — faqat ular 5 daqiqada so'raydi.
- Test muhiti haqida hech narsa yozilmagan.

## Maydonlar (YAML'dan)

### Token — `POST /security/oauth/token`
- So'rov `application/x-www-form-urlencoded`: `client_id`, `client_secret`,
  `grant_type`, `scope` — to'rttasi majburiy.
- Javob `{ "access_token": "..." }` — **yagona majburiy maydon**, qolganlari
  (expires_in va h.k.) Uzum tomonida ishlatilmaydi.
- 401 javob tanasi — `{ "reason": "Access token has been expired..." }`
  (`AuthorizationRequiredResponse`), **massiv emas**.

### Xato — `ErrorListV1`
**Massiv**: `[{ "code": <integer>, "description": "<matn>" }]`, ikkala maydon
majburiy.

### Katalog — `GET /v1/nomenclature/{storeId}/composition`
Content-Type: `application/vnd.eda.picker.nomenclature.v1+json`.
Noto'g'ri pozitsiyalar (juda uzun satr, narx 0.00) **jimgina tashlab
yuborilishi mumkin**.

`{ categories: [...], items: [...] }` — ikkalasi majburiy.

**categories[]**: `id`* (≤64), `name`*, `parentId` (≤64, daraxt),
`sortOrder` (yo'q bo'lsa 100), `images[]` (`hash`*, `url`*).

**items[]** — majburiy: `id` (≤64), `categoryId` (≤64), `name`, `price`,
`description`, `images`, `isCatchWeight`, `measure`, `barcode`, `vendorCode`.
- `price` (double) — **nol narxli tovar tashlab yuboriladi**. `oldPrice`
  (nullable) — «eski/yangi narx» aksiyasi.
- `barcode`: `value`* (satr), `weightEncoding`* (`none` |
  `ean13-tail-gram-4` | `ean13-tail-gram-5`), `type` (enum, masalan `ean13`),
  `values[]`.
- `description` (obyekt): `general`, `composition`, `nutritionalValue`,
  `purpose`, `storageRequirements`, `expiresIn`, `vendorCountry`,
  `vendorName`, `packageInfo` — hammasi ixtiyoriy satr.
- `images[]`: `hash`* — **rasm faylining SHA1'i, uni biz hisoblaymiz**;
  o'zgarsa Uzum rasmni qayta yuklaydi. `url`*.
- `measure`: `value` (integer), `unit` (`GRM` | `MLT`), `quantum` (float).
- `isCatchWeight` (bool) — vaznli tovar.
- `vendorCode` — artikul.
- `vat` (integer, %, yo'q bo'lsa 0).
- `serviceCodesUz`: `mxikCodeUz`* (**ИКПУ**), `packageCodeUz`.
- `volume`: `value`*, `unit`* (`CMQ` | `DMQ`); `location`; `sortOrder`.
- ⚠️ **Katalogda modifikator guruhlari yo'q** — lekin buyurtma pozitsiyasida
  `modifications[]` bor (quyida).

### Qoldiq — `GET /v1/nomenclature/{storeId}/availability`
`application/vnd.eda.picker.availability.v1+json`:
`{ items: [{ id*, stock* (float) }] }`. `0` — ro'yxatdan yo'qoladi;
**ro'yxatda yo'q tovar — sotilmaydi**; `-10.00` kabi mantiqsiz qiymat
tashlab yuborilishi mumkin.

### Buyurtma yaratish — `POST /order`
- Content-Type: `application/vnd.eats.order.v2+json` (aktual), shuningdek
  `vnd.eats.order.marketplace.v3+json`, `vnd.eats.order.yandex.v3+json`,
  `application/json`.
- So'rov tanasi YAML'da `type: object`, lekin tavsif ikki sxemani ajratadi:
  `MarketplaceOrder` (hamkor yetkazadi) va `YandexOrder` (**Uzum Tezkor
  kuryeri** yetkazadi). ⚠️ **Uzum Tezkor faqat o'z kuryeri bilan yetkazadi**
  (2026-09-14, Uzum bilan ishlagan egadan), ya'ni bizga keladigani — ikkinchisi,
  va uning aktual modeli faylda aniqlangan: `YGroceryOrderV2`
  («Актуальная версия модели заказа по схеме доставки "yandex"»,
  `discriminator: "uzum"`). `MarketplaceOrder` bizga tegishli emas.
- Javob 200: `{ "orderId": "<bizning ID>", "result": "OK" }`.

### `YGroceryOrderV2`
Majburiy: `eatsId`, `items`, `comment`, `promos`.
- `discriminator`: `"uzum"`.
- `eatsId` — `DDDDDD-DDDDDD` (`190330-123456`); **idempotentlik kaliti**.
- `restaurantId` — bizning nuqta ID.
- `comment` — satr. `persons` — integer (asboblar soni).
- `items[]` (`YGroceryOrderV2Item`) — majburiy `id`, `quantity`, `price`,
  `modifications`, `promos`:
  - `id` — bizning menyu pozitsiyasi ID; `name`.
  - `quantity` — ⚠️ **float** (V2 da; `3.5` — vaznli).
  - `price` — ⚠️ **bitta pozitsiyaning modifikatsiyalar bilan birga
    narxi**; «keyingi versiyada modifikatsiyasiz narxga tuzatiladi».
  - `modifications[]`: `id`*, `price`*, `quantity`* (integer), `name`.
    Bir taom turli modifikatsiyalar bilan — alohida pozitsiyalar.
  - `promos[]`: `discount`*, `type`* (`GIFT` | `PERCENTAGE` | `FIXED`).
  - `labelCodes[]` — markirovka kodlari.
- `promos[]` — butun buyurtmaga, xuddi shu shakl.
- `paymentInfo`: `itemsCost`* (double), `paymentType`* (`CARD` | `CASH`).
- `deliveryInfo`: `courierArrivementDate`* (RFC3339, soniya bo'lagi bilan),
  `clientName`, `phoneNumber` (**kuryer**), `clientPhoneNumber`.
  **Manzil yo'q — va bu to'g'ri**: mijozga Uzum kuryeri olib boradi,
  restoranga faqat kuryer **qachon kelishi** kerak. Restoran bu buyurtmaga
  kuryer tayinlamaydi va yetkazish narxini hisoblamaydi.

### Holat — `GET /order/{orderId}/status` → `OrderStatus`
`{ status*, comment (≤500), updatedAt (RFC3339) }`. Holatlar va o'tishlar
yuqoridagi bilan bir xil.

### Yangilash — `PUT /order/{orderId}`
Buyurtma **butunlay** yuboriladi. Javob `{ "result": "OK" }`; **422** —
«buyurtmani yangilab bo'lmaydi».

### Bekor qilish — `DELETE /order/{orderId}`
Tana majburiy, `application/json`: `{ eatsId*, comment }`. Javob 200, tanasiz.

### Sarlavhalar
Javoblarda `Cache-Control: private, max-age=0, no-cache, no-store`, `ETag`,
`Vary`, `Expires`, `Pragma` tavsiya qilinadi.

### Umumiy
- `servers: //localhost/` — test muhiti **ko'rsatilmagan**.
- Moliya (perechisleniye, komissiya) — **yo'q**.
