# POS_INTEGRATIONS.md — yangi kassa tizimlari bo'yicha tadqiqot

Mavjud: **iiko**, **Clopos**, **r_keeper** (`backend/internal/pos/`).
So'ralgan: Jowi, Poster, Paloma, Syrve, Yaros, AliPOS, Loook, Neon Alisa, Dodo Pizza.

Har biri `pos.Provider` interfeysiga tushishi kerak: `Ping`, `Products`,
`SendOrder`, `OrderStatus`, `Cancel`.

---

## Qisqa xulosa

| Tizim | Holat | Ish hajmi |
|---|---|---|
| **Syrve** | ✔️ **YOZILDI** (iiko adapteri, o'z hosti bilan) | bajarildi |
| **Poster** | ✔️ **YOZILDI** (`pos/poster.go` + testlar) | bajarildi |
| **Paloma365** | 🟡 API bor, Swagger yopiq, shartnoma kerak | hujjat kelgach 1 kun |
| **Jowi** | 🟡 Public API bor, hujjat login orqasida | hujjat kelgach 1 kun |
| **AliPOS** | 🟠 Ommaviy hujjat topilmadi | sotuvchi bilan aloqa |
| **Neon Alisa** | 🟠 Ommaviy hujjat topilmadi | sotuvchi bilan aloqa |
| **Dodo Pizza** | 🔴 Boshqa holat — pastga qarang | ehtimol kerak emas |
| **Yaros** | ❓ Topilmadi — aniqlashtirish kerak | — |
| **Loook** | ❓ Bu POS emas — aniqlashtirish kerak | — |

---

## 1. Syrve — ✅ eng oson, deyarli tayyor

**Syrve — iiko'ning xalqaro brendi.** Bir xil mahsulot, bir xil bulut API:

```
api-eu.iiko.services   ≡   api-eu.syrve.live
```

Ikkala manzil ham bir xil "Syrve Cloud API" hujjatini ochadi.

**Bizda nima bor:** `pos/iiko.go` allaqachon yozilgan va `IikoBaseURL`
**sozlanadigan**. Ya'ni adapter kodini qayta yozish kerak emas.

**Nima qilinadi:**
- `pos.Syrve = "syrve"` provider id qo'shiladi.
- `New()` da `case Syrve:` → `newIiko(cfg, client)` , faqat `IikoBaseURL`
  bo'sh bo'lsa `https://api-eu.syrve.live/api/1` standart qilib beriladi.
- `Name()` `"syrve"` qaytarishi uchun adapterga bitta maydon (`label`).
- Panelda alohida yorliq va sozlama bloki (iiko bilan bir xil maydonlar).

⚠️ **Nega alohida provider qilinadi, iiko'ning ichiga yashirilmaydi:** restoran
egasi o'z tizimini "Syrve" deb biladi. Ro'yxatda "iiko" ni ko'rsa, ulanmaydi.
Bu texnik emas, tanish masalasi.

---

## 2. Poster (joinposter) — ✅ hujjat to'liq, bugun yozilishi mumkin

Hujjat **ochiq va GitHub'da**: `github.com/joinposter/docs`.

### Buyurtma yuborish

```
POST https://joinposter.com/api/incomingOrders.createIncomingOrder?token=<TOKEN>
```

Autentifikatsiya — **query'dagi `token`** (sarlavha emas).

| Maydon | Tur | Majburiy | Izoh |
|---|---|---|---|
| `spot_id` | int | ✅ | savdo nuqtasi (filial) |
| `phone` | string | ✅* | `client_id` berilmasa majburiy |
| `client_id` | int | — | mavjud mijoz |
| `first_name`, `last_name` | string | — | mijoz ismi |
| `address` | string | — | yetkazish manzili |
| `comment` | string | — | onlayn buyurtmaga izoh |
| `products[]` | array | ✅ | pastga qarang |
| `payment` | object | — | oldindan to'langan bo'lsa |

`products[]` elementi:

| Maydon | Tur | Izoh |
|---|---|---|
| `product_id` | int | ✅ kassadagi id |
| `count` | int | ✅ miqdor |
| `price` | int | **tiyinda** — berilmasa kassaning narxi olinadi |
| `modificator_id` | int | variantlar uchun |
| `modification` | JSON string | `[{"m":<id>,"a":<qty>}]` |

`payment`: `{ type: 0|1, sum: <tiyin>, currency: "UZS" }` — `1` = to'langan.

### Javob

```json
{ "response": {
    "incoming_order_id": 123,
    "status": 0,            // 0 = yangi, 1 = qabul qilindi, 7 = bekor
    "spot_id": 1,
    "products": [ { "product_id": 169, "count": 1 } ]
} }
```

### ⚠️ Ikkita muhim tuzoq

**1. Narx tiyinda.** r_keeper bilan bir xil tuzoq: so'mda yuborilgan buyurtma
kassada **yuz barobar arzon** tushadi va kassa uni indamay qabul qiladi.

**2. `incomingOrder` — bu hali kassadagi buyurtma emas.** U "kiruvchi
buyurtma" bo'lib tushadi va xodim uni **qabul qilishi** kerak (`status: 0` →
`1`). Ya'ni bizning `Status` xaritamiz:
- `0` → `accepted` emas, **`unknown`/kutilmoqda**
- `1` → `accepted`
- `7` → `cancelled`

Buni noto'g'ri xaritalash "oshxonaga yuborildi" deb yozilgan, aslida hech kim
ko'rmagan buyurtmani beradi — iiko'dagi `commands/status` bilan bir xil dars.

### Qolgan metodlar
- Mahsulotlar (`Products`): `menu.getProducts`
- Filiallar (`Ping`): `spots.getSpots` → ulangan nuqta nomi qaytariladi
- Bekor qilish: `incomingOrders.updateIncomingOrderStatus` (status 7)

---

## 3. Paloma365 — 🟡 API bor, ruxsat kerak

- Hujjat: `help.paloma365.com/knowledgebase/api-dok/`, Swagger **SwaggerHub**da.
- **Ikkita API**: *Delivery API* (bizga kerakli) va asosiy API (1C uchun).
- Autentifikatsiya: **API AUTHKEY** — kabinetdan olinadi
  (*Предприятие → Управление → Настройки аккаунта*).
- Kod namunalari: `github.com/Vladsoftik/Paloma365_public`.
- O'zbekistonda bor (`paloma365.uz`).

⚠️ Hujjatda ochiq yozilgan: **"tijorat maqsadida foydalanishda alohida
shartnoma tuziladi"**. Ya'ni texnik ish boshlanishidan oldin huquqiy qadam bor.

**Kerak:** mijoz akkaunti yoki Paloma bilan shartnoma → Swagger'ga kirish.

---

## 4. Jowi — 🟡 public API bor, hujjat login orqasida

- O'zbek kompaniyasi (Toshkent) — **aloqa qilish eng oson bo'lgani**.
- `docs.jowi.club` mavjud, lekin **login talab qiladi**.
- Kompaniya o'zi ochiq e'lon qilgan: *"Мы предоставляем публичное API для
  системы ресторанов Jowi"*, ulanish bepul.
- API qamrovi (ular sanagan): restoranlar ro'yxati, onlayn menyu, zal
  sxemasi, chek yaratish, mehmon ro'yxati, stol broni, cheklar tarixi.

**Bu bizga to'liq mos** — menyu + chek yaratish bor.

**Kerak:** `docs.jowi.club` ga kirish (yoki `help@jowi.app` ga so'rov).

---

## 5. AliPOS — 🟠 hujjat topilmadi

- `alipos.uz` — O'zbekiston, restoran/kafe/bar/karaoke/sauna.
- Ommaviy API hujjati **topilmadi**.
- **Kerak:** to'g'ridan-to'g'ri so'rov. Mahalliy kompaniya bo'lgani uchun
  javob berish ehtimoli yuqori.

## 6. Neon Alisa — 🟠 hujjat topilmadi

- `neonalisa.com` / `neonalisa.kz` — **bitta mahsulot** ("NEON ALISA"),
  ikkita emas. Gibrid tizim: restoranda ham, bulutda ham server.
- Saytida "ko'pchilik POS tizimlari bilan integratsiya" deb yozilgan, ya'ni
  integratsiya qatlami bor — lekin **ommaviy hujjat yo'q**.
- **Kerak:** sotuvchidan API hujjati.

---

## 7. Dodo Pizza — 🔴 bu boshqa holat

Texnik tomoni bor: `docs.dodois.io` (Stoplight), **Dodo IS Marketplace** —
franchayzi restoranlar uchun ilovalar bozori, ochiq API bilan.

**Lekin savol texnik emas:** Dodo Pizza — franchayzing tarmog'i. Uning
**o'z sayti, o'z ilovasi va o'z yetkazish xizmati bor**, va franchayzi
shartnomasi odatda mustaqil buyurtma kanalini taqiqlaydi. Ya'ni Dodo
franchayzisiga "o'z saytingiz" sotib bo'lmaydi.

Bundan tashqari Dodo IS API asosan **hisobot va analitika** uchun mo'ljallangan
(marketplace ilovalari), tashqi buyurtmani kassaga qo'yish uchun emas.

**Tavsiya:** ro'yxatdan olib tashlash — aniq bir Dodo franchayzisi mijoz
bo'lib, o'zi so'ramaguncha.

---

## 8. Yaros — ❓ topilmadi

Qidiruvda "Yaros" nomli kassa tizimi topilmadi. Aniqlashtirish kerak:
sayt manzili yoki to'liq nomi. Ehtimol mahalliy yoki juda kichik tizim.

## 9. Loook — ❓ bu POS emas

**Loook — O'zbekistondagi pitseriya tarmog'i**, kassa tizimi sotuvchisi emas.
Ehtimol ularning o'z ichki tizimi bor, yoki siz ularni **mijoz** sifatida
nazarda tutgansiz. Aniqlashtirish kerak.

---

## Arxitektura: 12 ta provider bo'lganda nima o'zgaradi

Hozir `pos.Config` — **yassi struct**, har provider maydonlari prefiks bilan
(`IikoAPILogin`, `CloposClientID`, `RKeeperURL`). Uchtada bu toza. **O'n
ikkitada ~60 maydonli struct** bo'ladi va `pos_settings` hujjati ham shunga
qarab shishadi.

**Taklif:** provider sozlamalarini yassi maydonlar o'rniga

```go
type Config struct {
    Provider string
    Fields   map[string]string   // provider'ning o'z kalitlari
    BaseURL  string
}
```

ko'rinishiga o'tkazish, har adapter faqat o'z kalitlarini o'qiydi, va panel
formasi **provider tavsifidan** (maydon nomi, yorlig'i, sirmi) generatsiya
qilinadi. Aks holda har yangi POS uchun backend struct + Mongo model +
frontend forma — uch joyda qo'lda ish.

⚠️ Migratsiya kerak: mavjud `iiko`/`clopos`/`rkeeper` sozlamalari yangi
ko'rinishga ko'chiriladi. **Kalitlar hech qachon brauzerga qaytarilmasligi**
qoidasi saqlanadi (`hasKey` bayrog'i).

Bu qaror **Poster'dan oldin** qabul qilinishi kerak — keyin uchta emas,
to'rtta adapterni ko'chirish kerak bo'ladi.

---

## Tavsiya etilgan tartib

1. **Syrve** — bir soatlik ish, darhol qiymat beradi.
2. **`Config` refaktori** — 12 ta providerdan oldin, hozir arzon.
3. **Poster** — hujjat to'liq, mustaqil yozish mumkin.
4. **Jowi** + **Paloma** — hujjat so'rovi bugun yuborilsin, kelgach yoziladi.
5. **AliPOS** + **Neon Alisa** — sotuvchi javobiga bog'liq.
6. **Dodo** / **Yaros** / **Loook** — aniqlashtirilgach.

---

## Manbalar

- Poster hujjati (ochiq): https://github.com/joinposter/docs
- Poster `createIncomingOrder`: https://github.com/joinposter/docs/blob/master/ru/web/incomingOrders/createIncomingOrder.md
- Syrve Cloud API: https://api-eu.iiko.services/index.html
- Syrve hujjati: https://en.syrve.help/articles/#!api/getting-started-api
- Paloma365 API: https://help.paloma365.com/knowledgebase/api-dok/
- Paloma365 namunalar: https://github.com/Vladsoftik/Paloma365_public
- Jowi hujjati (login): https://docs.jowi.club/
- Dodo IS API: https://docs.dodois.io/
- Neon Alisa: https://neonalisa.com/
- AliPOS: https://alipos.uz/
