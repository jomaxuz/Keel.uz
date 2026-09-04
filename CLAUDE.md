# CLAUDE.md — Restaurant Website + Delivery System (Template)

Bu fayl — loyihaning **doimiy qismi**: arxitektura, ma'lumot modeli, API
guruhlari, konvensiyalar va butun kodga tegadigan tuzoqlar. Har bir yangi
sessiyada shu fayl o'qiladi, shuning uchun u **qisqa qoladi**.

- Har bir xususiyatning o'z qarorlari va tuzoqlari → **`docs/DECISIONS.md`**
  (§11 dagi jadval qaysi bo'lim kerakligini aytadi; **butunlay o'qilmaydi**).
- Kunlik ish jurnali → **`PROGRESS.md`**.

---

## 1. Loyiha maqsadi (Business context)

Bu — **restoranlar uchun tayyor "website + delivery system" shabloni (template)**.
Maqsad: O'zbekiston shaharlaridagi (Toshkent, Samarqand va boshqalar) restoranlarga
xizmat sifatida sotish. Har bir restoran uchun shu shablon alohida **deploy**
qilinadi (bitta restoran = bitta deployment). Ya'ni bu **multi-tenant emas** —
har bir mijoz o'zining VPS'ida o'zining nusxasini oladi.

Yetkazib beriladigan qism:
1. **Public website** — restoran sayti: menyu (to'liq), profil, ish vaqti,
   aloqa, manzil (xarita), foto, va h.k.
2. **Admin panel** — restoran egasi uchun: ish vaqtini o'zgartirish, menyuni
   yaratish / tahrirlash / o'chirish (CRUD), kategoriyalar, restoran profilini
   boshqarish, buyurtmalarni ko'rish.
3. **Delivery system (yetkazib berish)** — to'liq ishlaydigan buyurtma + yetkazib
   berish tizimi. **Yandex Maps** orqali manzil tanlash, yetkazib berish zonasi va
   narxini hisoblash. **Kuryerni real-time ko'rsatish HOZIRCHA SHART EMAS.**

---

## 2. Arxitektura (High-level)

**Monolit** deployment. Bitta VPS'da ishlaydi. Ikki qism:

```
softmax/
├── CLAUDE.md                 # shu fayl (doimiy qism)
├── docs/DECISIONS.md         # xususiyatlar bo'yicha qarorlar (kerakli bo'lim o'qiladi)
├── PROGRESS.md               # ish jurnali
├── DEPLOY.md                 # deploy qo'llanmasi (VPS / Vercel)
├── docker-compose.yml        # dev: mongo + backend + frontend
├── docker-compose.prod.yml   # prod: portlar faqat 127.0.0.1, .env dan sozlash
├── .env.prod.example         # prod muhit o'zgaruvchilari namunasi
├── nginx/restaurant.conf     # host nginx reverse proxy (TLS, /api, /uploads)
├── frontend/                 # Next.js + TypeScript (public site + admin panel)
├── backend/                  # Go + MongoDB (REST API + rasm upload/serve)
└── mobile/                   # Expo ilovalari, qoidalari frontend/src/lib dan
    ├── waiter/               # ofitsiant: zal, chek, menyu, davomat
    ├── courier/              # kuryer: smena, joylashuv oqimi, yetkazish
    ├── team/                 # qolgan xodimlar: davomat, ish haqi, xabarlar,
    │                         #   bozorchiga «Bozor» (faqat `buy` ruxsatida)
    ├── owner/                # ega: bugungi raqamlar, diqqat, buyurtma, hisobot
    └── tv/                   # zaldagi televizor (Android TV): kontent va tablo
```

⚠️ **`mobile/` dagi ilovalar `frontend/src/lib` ni ko'chirmaydi** — Metro uni
watch qiladi va `@/` aliasi veb ilovadagi bilan bir xil. Ya'ni qoida bir joyda
yoziladi va uch joyda (sayt, kassa, telefon) bir xil javob beradi; nusxa esa
ajraydi, va ajragani restorandagi telefonda qoladi. Tafsiloti — har ilovaning
o'z `README.md` ida.

- **Frontend** (`frontend/`): Next.js (App Router) + TypeScript + Tailwind CSS.
  Public sayt va admin panel bitta Next.js app ichida (`/` public, `/admin` panel).
- **Backend** (`backend/`): Go (chi router) + MongoDB (official driver).
  REST API beradi, JWT auth, va yuklangan menyu rasmlarini `uploads/` papkadan
  static `/uploads/*` route orqali serve qiladi.
- **Rasmlar**: admin panel orqali yuklanadi → backend `uploads/` papkasiga
  diskka saqlanadi → `/uploads/<file>` URL orqali ochiladi. VPS'da bu papka
  yetarli (S3 va h.k. shart emas). Docker'da bu **volume** sifatida mount qilinadi.

### Ma'lumot oqimi
```
Browser ──HTTP──▶ Next.js (SSR/CSR) ──REST──▶ Go API ──▶ MongoDB
                                     └──▶ /uploads/*.jpg (static)
Browser ──JS──▶ Yandex Maps API (manzil tanlash, xarita)
```

---

## 3. Texnologiyalar (Tech stack)

### Backend
- **Go** (1.26+)
- **chi** — HTTP router (`github.com/go-chi/chi/v5`)
- **MongoDB** driver — `go.mongodb.org/mongo-driver`
- **JWT** — `github.com/golang-jwt/jwt/v5` (admin auth)
- **bcrypt** — parol hash (`golang.org/x/crypto/bcrypt`)
- **godotenv** — `.env` (`github.com/joho/godotenv`)
- Validatsiya: `github.com/go-playground/validator/v10`

### Frontend
- **Next.js** (App Router) + **TypeScript**
- **Tailwind CSS** — styling
- **React Hook Form + Zod** — formalar va validatsiya
- **TanStack Query** (react-query) — server state / API
- **Yandex Maps** — `@pbe/react-yandex-maps` yoki to'g'ridan-to'g'ri JS API
- Til: UZ (asosiy) + RU (keyingi bosqich uchun tayyor struktura)

### Infra
- **Docker** + **docker-compose** (mongo, backend, frontend)
- **Nginx** (prod'da reverse proxy — keyingi bosqich)

---

## 4. Ma'lumotlar modeli (MongoDB)

Maydonlar Go modellarida (`backend/internal/models/`) — bu yerda faqat
kolleksiyalar ro'yxati va koddan ko'rinmaydigan qarorlar.

- **Asosiy**: `restaurant` (singleton — kompaniya profili + `content`, `seo`,
  `theme`, `loyalty`, `booking`), `brand`, `branch`, `category`, `menu_item`,
  `order`, `reservation`, `user`, `promotion`, `feedback`, `loyalty_txn`,
  `visit`, `banner`, `vacancy`, `job_application`, `page_design`.
- **Zaldagi televizorlar**: `tv_screen`, `tv_pairing`, `tv_slide` (playlist —
  ⚠️ **filialniki, ekranniki emas**: ekran bilan farq qiladigan narsa rejim, u
  esa `tv_screen.mode` da).
- **Xodimlar**: `admin_user`, `admin_log`, `login_device` (bir hisob — bir
  telefon, ilova bo'yicha), `courier`, `courier_device`,
  `courier_settlement`,
  `staff`, `shift`, `staff_payment`.
- **Kassa / moliya**: `cash_shift`, `cash_entry`, `payment`, `staff_advance`
  (podotchet — ⚠️ **chiqim emas**: pul kirim sotib olganda sarflanadi, va u
  allaqachon moliyaviy hisobotda; qarang `docs/DECISIONS.md` → "Podotchet"),
  `shopping_order` (bozorlik ro'yxati — ⚠️ **so'rov, kirim emas**: ikkalasi
  faqat safar yakunlanganda uchrashadi).
- **Tannarx va ombor**: `stock_movement` (⚠️ **sotuv sarfining manbasi** —
  chekka urilganda yoziladi, arifmetika endi undan o'qiydi; qarang
  `docs/DECISIONS.md` → "Spisaniya hujjati"), `migration_state` (bir martalik
  migratsiyalar markeri), `ingredient` (kartasi bo'lsa — yarim tayyor mahsulot),
  `warehouse`, `ingredient_placement`, `purchase` (kirim), `writeoff`,
  `stock_transfer` (ko'chirish), `production` (tsex partiyasi), `stocktake`,
  `supplier`, `print_job`. Texkarta
  esa alohida kolleksiya emas — `menu_item.recipe` (taom kartasi) va
  `ingredient.recipe` + `output` (zagotovka: sous, xamir, sushi guruchi).
  Ikkalasi bitta ekranda yoziladi (`/admin/tech-cards`) — qarang
  `docs/DECISIONS.md` → "Texkarta o'z ekranida".
- **Integratsiya sozlamalari (singleton)**: `payment_settings` (+ `inStore` —
  kassada QR skanerlab karta yechish relslari; qarang `docs/DECISIONS.md` →
  "Kassada karta"), `sms_settings`,
  `pbx_settings`, `telegram_settings`, `push_settings` (VAPID juftligi —
  sozlanmaydi, birinchi ishlatishda generatsiya qilinadi va **hech qachon
  almashtirilmaydi**: har obuna o'zi yaratilgan ochiq kalitga bog'langan);
  `pos_settings` + `pos_mapping` (filial darajasida), `telegram_chat`.
- **Brauzer bildirishnomalari**: `push_subscription` — bir brauzer, bir hujjat.
  ⚠️ `endpoint` **unique**: service worker brauzer yangilanganidan keyin jimgina
  qayta ro'yxatdan o'tadi, indekssiz mijoz har kampaniyani ikki marta olardi.
- **Boshqa**: `phone_code`, `delivery_provider`, `call`, `campaign`,
  `export_grant`, `design_preview`.

Buyurtma holati: `pending → confirmed → preparing → on_the_way → delivered |
cancelled` (`statusHistory` har o'zgarish vaqtini yozadi; `cancelled` uchun
`cancelReason` majburiy). `order.type`: `delivery | pickup | dinein`.
Bron holati: `pending → confirmed → seated → done | cancelled`.

⚠️ **Sirlar `restaurant` hujjatidan tashqarida.** Profil har tashrifchiga
to'liq qaytariladi, shuning uchun to'lov kalitlari, SMS parollari, ATS va bot
tokenlari alohida kolleksiyalarda — bitta unutilgan `json:"-"` sizib chiqqan
kalit demakdir. Xarita kaliti ataylab teskarisi: u brauzerga kerak
(`restaurant.mapApiKey`), himoyani 2GIS kabinetidagi domen cheklovi beradi.

⚠️ **`pos_settings.branchId` unique bo'lishi shart.** Sozlamalar `upsert` bilan
saqlanadi: indekssiz bir vaqtda kelgan ikki so'rov bitta filialga ikki hujjat
yozadi, `FindOne` esa **birini** oladi — alomati "saqlangan sozlama o'zgarib
qaytdi". Migratsiya dublikatlarni tozalaydi (oxirgisi qoladi); tozalanmasa
indeks yaratilmaydi va **server ko'tarilmaydi**.

⚠️ **`ingredient_placement` ham xuddi shu sababdan alohida**: masalliq
brendniki (texkarta uni id bo'yicha nomlaydi, ya'ni zanjirning uchta oshxonasi
bitta katalogdan pishiradi), ombor esa filialniki (eshigi bor xona). Ilgari
`ingredient.warehouseId` ikkalasi bo'lishga urinardi, va bu **bitta filialgacha**
ishlaydi. Ikkitasida nosozlik jim va to'liq edi: Chilonzor omboriga joylangan
kartoshkani Yunusobodda umuman sanab bo'lmasdi, Yunusobodning sarfi esa
Chilonzorning javoniga yozilardi — hech qayerda xato chiqmay.
`(branchId, ingredientId)` unique; qoida bir so'z uzunroq bo'ldi: **bir
masalliq, bir ombor — har filialda**. Bo'sh joylashuv = umumiy ombor (bo'sh
`mapProvider` = 2GIS bilan bir qoida). `ingredient.warehouseId` **faqat
migratsiya o'qiydigan** meros maydon bo'lib qoldi.

⚠️ **`pos_mapping` alohida, chunki menyu brendniki, kassa filialniki**: bir
brendni ikki filial ikki iiko hisobidan sotsa, bitta lag'monning ikki id'si
bo'ladi. `(branchId, menuItemId)` unique.

---

## 5. REST API

Base: `/api/v1`. To'liq ro'yxat — `backend/internal/router/router.go`
(bu yerda faqat guruhlar va koddan ko'rinmaydigan qarorlar).

- **Public**: `/restaurant` (`?raw=1` — brend/filial qatlamisiz, panel uchun),
  `/categories`, `/menu`, `/promotions` (faqat aksiyalar — kodlar hech qachon
  qaytarilmaydi), `/orders` + `/orders/quote`, `/orders/{number}` (kuzatuv),
  `/delivery/quote`, `/payment-methods`, `/visit`, `/reservations`,
  `/recommendations` (POST — savat yoki taomga tavsiya), `/push/key` (VAPID
  ochiq kaliti — brauzerga beriladi, xarita kaliti bilan bir toifada).
- **Mijoz auth** (telefon + bir martalik SMS kod): `/auth/phone/request|verify`,
  `/users/me` (+ `/orders`, `/lang`, `/phone/request|verify`, `/push`).
- **To'lov callback'lari** (provayder chaqiradi, public): `/payments/payme`
  (JSON-RPC), `/payments/click/prepare|complete`, `/payments/uzum/*`,
  `/payments/atmos/*`. Webhook'lar: `/pbx/onlinepbx/{token}`,
  `/telegram/{token}` — manzildagi token **kalit** (qarang §10).
- **Kuryer** (`role: courier`): `/courier/login|me|status|location|orders|
  stats|history`, `/courier/push` (POST/DELETE — telefonning Expo tokeni va
  **tili**; matnni server yozadi, ya'ni telefon uni tarjima qila olmaydi).
- **Ishchi** (`role: staff`): `/staff/login|me|clock|report`, `/staff/push`
  (telefon tokeni va **tili** — matnni server yozadi),
  `/staff/kitchen` (KDS), `/staff/warehouses|stocktake/sheet|stocktake`
  (omborni telefonda sanash — `PermStock`, filial ishchidan olinadi).
- **Televizor** (`role: tv`): `/tv/pair/start|status` (ochiq — ulanayotgan set
  hali hech kim emas), `/tv/me` (yurak urishi + `contentVersion`), `/tv/playlist`
  (⚠️ faqat versiya o'zgarganda o'qiladi; `active` serverda filtrlanadi, **sana
  esa televizorda** — oflayn ekran ham tugagan aksiyani tushirishi kerak),
  `/tv/board` (⚠️ **faqat raqamlar** — `projection` bilan; olib ketish va zal,
  yetkazish emas).
- **Kiosk** (`role: kiosk`): `/kiosk/*`.
- **Admin** (`owner`/`manager`): `/admin/*` — profil, menyu/kategoriya CRUD,
  `/admin/push` (ega telefonining tokeni va tili — loss alertlar shu orqali
  ham boradi, Telegram bilan yonma-yon),
  `/admin/devices/{kind}/{id}` + `DELETE /admin/devices/{deviceId}` (qaysi
  telefon qaysi hisobga biriktirilgan, va uni bo'shatish),
  upload, buyurtmalar, bronlar, kuryerlar, ishchilar, payroll, kassa,
  hisobotlar, CRM/segmentlar/kampaniyalar, call-markaz, POS (+ stop list),
  to'lov/SMS/PBX/
  Telegram sozlamalari, hisoblar va amallar jurnali, eksport.
- **Ombor** (`owner`/`manager`): `/admin/ingredients` (+ `/placement` — shu
  filial masalliqni qaysi omborda saqlaydi), `/admin/warehouses`,
  `/admin/purchases` (+ `/{id}` PUT tahrir, `/{id}/pay`), `/admin/suppliers`
  + `/admin/reports/suppliers`, `/admin/writeoffs`, `/admin/transfers`,
  `/admin/stocktake`, `/admin/stock/balances|movement|shopping-list`.
- **Texkarta**: `PUT /admin/menu/{id}/recipe` (bitta maydonning `$set`'i).
  ⚠️ **Taom formasi endi `recipe` ni umuman yubormaydi** — karta o'z ekranida
  yoziladi, va `UpdateMenuItem` butun hujjatni almashtiradi: `keepRecipe`
  busiz taomning narxini o'zgartirish uning kartasini o'chirardi (qarang
  `docs/DECISIONS.md` → "Texkarta o'z ekranida").

Konvensiyalar:
- ⚠️ **Xato xabarlari o'zbekcha yoziladi va server tarjima qiladi.**
  `httpx.Error(w, status, "ochiq smena yo'q")` — chaqiruv joyi shundayligicha
  qoladi; til javob **yozuvchisida** (`middleware.Lang`, `lang` cookie'si) va
  tarjima `internal/i18n` da, kalit sifatida **o'zbekcha matnning o'zi**.
  Yangi xabar qo'shsangiz `internal/i18n/messages.go` ga ham qo'shing — aks
  holda test yiqiladi (qarang `docs/DECISIONS.md` → "Server xabarlari ham uch
  tilda"). Xato bo'lmagan, lekin odamga ko'rinadigan matn uchun —
  `httpx.LangOf(w)`.
- **Sirlar hech qachon qaytarilmaydi** — sozlamalar javobida faqat `hasKey` /
  `hasPassword` bayrog'i. **Bo'sh kalit yuborilsa saqlangani qoladi**, o'chmaydi
  (aks holda bitta maydonni tuzatayotgan ega to'lovni/SMS'ni jimgina o'chiradi).
- `PUT /admin/restaurant` — **qisman yangilash**: faqat yuborilgan maydonlar
  yoziladi (to'liq `$set` brend/filialga ko'chgan maydonlarni bo'shatardi).
  Xuddi shunday `AdminUpdateBranch` `soldOut`, `kioskSecret`, `kioskVersion` ni
  **ataylab yozmaydi**.
- Filial linzasi `?brandId=`/`?branchId=` (`adminScope`), `clampToAdmin` bilan
  qisqartiriladi — menejer URL orqali kengaya olmaydi.
- ⚠️ **Ombor ekranlari bundan mustasno: ular bitta filialni talab qiladi**
  (`stockBranch`, 400 `errPickBranch`). Uchta muzlatgichga tarqalgan
  "kompaniyada 9 kg go'sht bor" — sanab ham, buyurtma berib ham, pishirib ham
  bo'lmaydigan raqam; ilgari arifmetika uni baribir chiqarardi. Bitta filialli
  restoran buni **hech qachon ko'rmaydi** — filial u uchun o'zi aniqlanadi.
- Bekor qilish uchun `reason` majburiy (buyurtma ham, bron ham).
- Static: `GET /uploads/<file>` (`?w=300|600|1200`), `GET /health`.

---

## 6. Frontend sahifalar (routes)

Fayl tizimidan ko'rinadi (`frontend/src/app/`), shuning uchun bu yerda faqat
tuzilma:

- **Public** — `(site)/`: `/`, `/menu`, `/menu/[id]`, `/bron`, `/cart`,
  `/checkout`, `/order/[number]`, `/profile`, `/login`, `/about`, `/vakansiya`.
- **Ilovalar**: `/kuryer` (PWA), `/staff` + `/staff/kitchen` (KDS) +
  `/staff/stock` (omborni sanash), `/kiosk` (filial ekrani) — har birida
  `login` sahifasi, hisobni admin beradi.
- **Panel** — `/admin/…`: `login`, dashboard, `menu`, `categories`, `orders`,
  `reservations`, `promotions`, `reports`, `qr`, `pos`, `stop-list`, `calls`,
  `campaigns`, `users/[id]`, `couriers/[id]`, `staff/[id]`, `payroll`, `admins`, `logs`,
  `settings`, `account`.
- **Panel → Ombor bo'limi**: `stock` (qoldiqlar), `shopping` (xarid ro'yxati),
  `ingredients`, `tech-cards` (zagotovka + taom kartalari), `purchases`,
  `suppliers`, `writeoffs`, `transfers`, `stocktake`.
- Til prefikslari (`/ru/`, `/en/`) faqat public sahifalarda —
  `isLocalizedPath()` (§10 "Til URL'lari").

---

## 7. Delivery / Map logikasi

**Xaritani restoran o'zi tanlaydi: 2GIS / Yandex / Google** (`restaurant.mapProvider`,
standart — 2GIS). Sozlamalarda tanlanadi, **har birining kaliti alohida**
(`mapApiKey` / `mapYandexKey` / `mapGoogleKey`).
- ⚠️ **Bitta umumiy kalit maydoni bo'lmasligi kerak**: Yandex'ni sinab ko'rib
  2GIS'ga qaytgan ega bir provayderga ikkinchisining kalitini uzatib qo'yardi —
  bu bo'sh xarita va konsoldagi xato bo'lib chiqadi, restoranda esa hech kim
  konsolga qaramaydi. POS kalitlaridagi bilan bir qoida: har provayderga o'z
  tortmasi.
- ⚠️ **Bo'sh `mapProvider` — 2GIS**: bu sozlamadan oldingi har bir install
  2GIS'da, va nol qiymatni boshqacha o'qish ularning hammasida xaritani
  o'chirardi.
- **Bitta interfeys, uchta dvigatel** (`lib/map/`): `engine.ts` — shartnoma,
  `twogis.ts` / `yandex.ts` / `google.ts` — bajarilishi, `useMapEngine()` —
  yaratish, xato va tozalash. Ilgari har bir xarita komponenti to'g'ridan-to'g'ri
  MapGL'ga yozilgan va har birida "provayderni almashtirish uchun faqat shu
  faylni o'zgartiring" degan izoh bor edi — **to'rtta fayl bitta fayl emas**.
- ⚠️ **Koordinata tartibi faqat dvigatelda**: 2GIS `[lng, lat]`, Yandex
  `[lat, lng]`, Google `{lat, lng}`. Uch joyda to'g'ri, to'rtinchisida teskari
  qilingan almashtirish restoranni Orol dengiziga qo'yadi va ma'lumot xatosiga
  o'xshaydi.
- WebGL faqat 2GIS uchun shart (`requiresWebGL`) — eski telefonda kulrang
  quti chiqsa, javob "Yandex'ga o'ting" bo'lishi mumkin.
- ⚠️ **Google xarita ochilishi uchun pul oladi** (qolgan ikkisi bu hajmda
  amalda bepul) — sozlamalar sahifasida shu yozilgan, chunki hisobni ega to'laydi.
- Geokodlash (manzil qidiruvi) provayderdan **mustaqil**: Nominatim
  (`lib/geocode.ts`), ya'ni xarita almashtirilsa ham qidiruv o'zgarmaydi.

- **Checkout**'da mijoz xaritada manzilni tanlaydi (marker qo'yadi yoki
  qidiradi) → `lat/lng` + matn manzil olinadi.
- `POST /delivery/quote` backend'ga `lat/lng` yuboriladi → backend:
  - Agar **zonalar (polygon)** sozlangan bo'lsa: nuqta qaysi zonaga tushishini
    tekshiradi (point-in-polygon) → o'sha zonaning `fee`'si.
  - Yoki **radius** modeli: restorandan masofani (Haversine) hisoblab
    `baseFee + perKm * km`.
  - `minOrder`, `freeDeliveryFrom` ham qo'llanadi.
- Admin panelda **bitta "Yetkazib berish" bo'limi**: yoqish, minimal buyurtma,
  bepul yetkazish, so'ng "Narx qanday hisoblanadi?" — **masofa bo'yicha**
  (radius) yoki **xaritadagi zonalar bo'yicha**. Tanlov `delivery.mode` da
  saqlanadi; ikkinchisining maydonlari yashiriladi (lekin ma'lumot saqlanadi).
  `mode` bo'sh bo'lsa (eski hujjatlar) chizilgan zona bor-yo'qligiga qarab
  aniqlanadi. `mode: "zones"` bo'lsa-yu ishlaydigan zona bo'lmasa — radius
  modeliga qaytadi (hech kimga yetkazmay qo'ymaslik uchun).
- ⚠️ **Sozlamalar sahifasi filialning qamrovi haqida ogohlantiradi**, chunki bu
  yerdagi xatolar **jimgina** bo'ladi: forma to'ldirilgandek ko'rinadi, hech
  nima xato bermaydi, va natija butunlay boshqa joyda — buyurtma umuman
  kelmaydigan filial yoki hammasini o'ziga oladigan filial bo'lib bilinadi.
  To'rt holat: **yetkazish o'chiq** (bir nechta filialda — u yetkazish
  buyurtmalarida umuman qatnashmaydi), **xaritada nuqta yo'q** (masofa shundan
  o'lchanadi), **`maxKm = 0`** ("cheklov yo'q", "yetkazmaydi" emas — bir
  nechta filialda bu eng xavflisi), **`baseFee = perKm = 0`** (bepul yetkazish).
  Jonli mijozda aynan birinchisi bo'lgan: ikki filialda yetkazish o'chiq edi va
  Yangiyo'lga berilgan buyurtma Chilonzorga ketardi — buni hech bir ekran
  aytmasdi.
- **Zonalar xaritada chiziladi**, har zonaga nom + narxlash usuli: **belgilangan narx** yoki **km bo'yicha** (`baseFee + ceil(km)*perKm`,
  masofa restorandan Haversine bilan).
- **Zonalar mijozga ham ko'rinadi**: checkout va profil xaritasida polygon
  chiziladi + tagida legenda (zona nomi, narxi). Manzil tanlanmagan bo'lsa
  xarita butun zonaga moslashadi (`fitBounds`).
- **Muhim**: xaritadagi polygonlar `interactive: false` bo'lishi shart —
  aks holda MapGL polygon click'ni yutadi va zona *ichiga* metka qo'yib
  bo'lmaydi (faqat tashqarisiga ishlaydi).
  Kamida 3 nuqtali zona "ishlaydigan" hisoblanadi; ishlaydigan zona bo'lsa
  radius sozlamalari (baseFee/perKm/maxKm) e'tiborga olinmaydi va zonadan
  tashqariga yetkazilmaydi.
- **Kuryer real-time tracking — hozircha YO'Q.** Buyurtma statusi qo'lda
  (admin) o'zgartiriladi: `on_the_way` bo'lganda mijoz "yo'lda" deb ko'radi.
- Xarajatni kamaytirish: xarita faqat checkout'da yuklanadi, suggest debounce
  (300ms), lat/lng bir marta saqlanadi, masofa backend'da Haversine bilan.
- 2GIS API key `.env`'da: `NEXT_PUBLIC_MAP_API_KEY` (dev.2gis.com dan, bepul).

---

## 8. Muhit o'zgaruvchilari (.env)

### backend/.env
```
PORT=8080
MONGO_URI=mongodb://localhost:27017
MONGO_DB=restaurant
JWT_SECRET=change-me
UPLOAD_DIR=./uploads
PUBLIC_BASE_URL=http://localhost:8080
CORS_ORIGINS=http://localhost:3000
```

### frontend/.env.local
```
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_UPLOADS_URL=http://localhost:8080/uploads
NEXT_PUBLIC_MAP_API_KEY=...   # faqat zaxira: kalit endi paneldan (Sozlamalar → Xarita)
```

---

## 9. Ishga tushirish (Dev)

```bash
# Mongo (docker)
docker compose up -d mongo

# Backend
cd backend && cp .env.example .env && go run ./cmd/server

# Frontend
cd frontend && cp .env.local.example .env.local && npm install && npm run dev
```

Prod: `DEPLOY.md` ga qarang (`docker-compose.prod.yml` + host nginx + certbot,
yoki frontend Vercel'da + backend VPS'da).

---

## 10. Konvensiyalar (Conventions)

- **Kod izohlari va commit**: inglizcha. **Docs (CLAUDE.md, PROGRESS.md)**: aралаш
  (asosan o'zbekcha).
- Backend: `internal/` package'lari, handler → service → repository qatlamlari.
- Frontend: `src/` ichida, komponentlar `src/components`, API `src/lib/api`.
- ⚠️ **`data-help="..."` — surat vositalari qo'lini uzatadigan element**
  (bilim bazasi izohlari va keel.uz landing suratlari). Test yoki uslub ilgagi
  emas: `scripts/help-screens.mjs` shu atribut bo'yicha koordinata o'lchaydi,
  `scripts/landing-shots.mjs` esa shu bo'yicha bosadi. Ko'rinadigan matn
  bo'yicha izlash o'zbekchada ishlab **ruschada hech nimaga mos kelmaydi** va
  xato bermaydi — izoh tushib qoladi yoki kadr noto'g'ri ekrandan olinadi.
  Atributli elementni ko'chirsangiz atributni ham ko'chiring.
- Har bir katta ish bosqichidan keyin **`PROGRESS.md`** yangilanadi.
- Pul birligi: **UZS** (so'm), butun son (tiyin ishlatilmaydi).

### Git: branch mahsulot bo'limi bo'yicha nomlanadi
- ⚠️ **Backend/frontend bo'yicha emas, ko'rinadigan bo'lim bo'yicha.** Bitta
  xususiyat deyarli doim Go endpointi **va** uni chizadigan sahifadan iborat —
  ularni ikki branchga bo'lish yarim ishlaydigan `main` beradi (endpoint bor,
  chaqiradigan sahifa yo'q; bugun aynan shu bo'ldi: `/admin/banners` 404).
  Shuning uchun branch **kim ko'radigan bo'lim** bo'yicha nomlanadi, va backend
  o'zi xizmat qiladigan bo'limga tegishli hisoblanadi.

  | Prefiks | Bo'lim | Qayerga tegadi |
  |---|---|---|
  | `website/` | Restoran sayti (mehmon ko'radigan) | `frontend/src/app/(site)`, public API |
  | `dashboard/` | Restoran paneli (ega ko'radigan) | `frontend/src/app/admin`, `/admin/*` API |
  | `console/` | Keel konsoli (biz ko'radigan) | `keel-site/src/app/console`, `control/` |
  | `keel-site/` | keel.uz landing (mijoz topadigan) | `keel-site/src/app` (konsoldan tashqari) |
  | `bot/` | Telegram bot va mini app | `internal/telegram`, mini app qismlari |
  | `apps/` | Kuryer, ishchi, KDS, kiosk ilovalari | `frontend/src/app/{kuryer,staff,kiosk}` |
  | `infra/` | Deploy, docker-compose, caddy, zaxira | `deploy/`, `caddy/`, compose fayllari |

  Masalan: `website/branches-page`, `dashboard/vacancies`, `console/staff-roles`,
  `bot/feedback-stars`, `infra/backup-verify`.
- **Ikki bo'limga tegsa** — ishni **boshlagan** bo'lim bo'yicha nomlanadi. Menyu
  modeliga tegib panelni ham, saytni ham o'zgartirgan ish `dashboard/` bo'ladi,
  agar sabab panelda bo'lsa.
- **Bir vaqtda bitta ishlaydigan branch.** Yetti branch — yetti xususiyat emas,
  bitta oqimning yetti bo'lagi, va ro'yxat "hozir nima jonli?" degan savolga
  javob bera olmaydi.
- **Merge qilingan branch darhol o'chiriladi**: `git push origin --delete <nom>`
  va `git branch -d <nom>`. Kommitlar `main` da — nom faqat "bu merge
  qilinganmi?" savolini qaytadan tug'diradi.
- Kichik tuzatish (matn, izoh, bir qatorli xato) **`main` ga to'g'ridan-to'g'ri**.
- ⚠️ **Deploy `main` dan ketadi**: branchdagi ish merge qilinmaguncha jonli
  serverda yo'q. "Push qildim, nega ko'rinmayapti?" savolining javobi shu.

### Versiya: bitta joyda, va nimani anglatishi bilan
- `control/internal/handlers/version.go` — `Version` va `Stage` konstantalari.
  ⚠️ **Binardagi konstanta**, fayldan yoki muhit o'zgaruvchisidan o'qilmaydi:
  versiya **ishlab turgan kod** bilan mos bo'lishi shart, qolgan har manba esa
  undan ajrab ketishi mumkin (diskdagi fayl rollbackdan omon qoladi, env'ni
  compose faylini oxirgi tahrirlagan odam qo'yadi, git tegi esa repozitoriy
  haqidagi fakt — so'rovga javob berayotgan konteyner haqidagi emas).
- `Stage` raqamdan **alohida** va raqam nimani anglatishini aytadi. "v0.1"
  yolg'iz mehmonni taxmin qilishga undaydi, va restoranining buyurtmalarini
  tutib turgan platforma haqida odam **saxiy** taxmin qiladi — shuning uchun
  halol so'z hali rost bo'lib turganda yonida yozilади.
- Ko'rinadigan joyi — `/status`: bu sahifa nimadir buzuqqa o'xshaganda ochiladi,
  va "qaysi versiya?" — "ishlayaptimi?" dan keyingi birinchi savol.

### ⚠️ Tuzoq: Go'ning JSON marshalling odatlari (ikki marta tishlagan)
**1) Nil slice `null` bo'lib chiqadi**, `[]` emas. Ro'yxat qaytaradigan har bir
maydon **bo'sh massivga aylantirilishi** kerak (masalan `bookingSettings()`
`Tables`/`Shapes` ni to'ldiradi), aks holda frontendda `null.length` yiqiladi.

⚠️ **Bu ikkinchi marta ham chiqdi, va compiler ham, testlar ham ko'rmadi**
(`control/internal/handlers/export.go`): bo'sh massiv **qurildi va javobda
ishlatilmadi** — Go xato bermadi, chunki o'zgaruvchi *o'qilgan* edi (uni
to'ldirayotgan sikl tomonidan). Grant hujjatida `downloads` maydoni yo'q edi
(hech kim hali yuklab olmagan), demak nil slice → `null` → konsolda
`grant.downloads.length` → **butun mijoz kartochkasi** React xato ekraniga
almashdi. Birinchi ochilgan grantda, ya'ni maydon birinchi marta yo'q bo'lishi
mumkin bo'lgan paytda.
Shuning uchun endi qoida **funksiyada** yashaydi (`downloadsJSON`) va uning
testi bor: javobga boradigan yo'l shu funksiyadan o'tadi, va "keyingi tahrir
uni chetlab o'tolmaydi".

**2) Bo'sh `ObjectID` JSON'da yo'qolmaydi**
`json:"...,omitempty"` **massivlarga ta'sir qilmaydi**, `primitive.ObjectID`
esa `[12]byte`. Ya'ni qo'yilmagan `branchId` / `courierId` / `userId`
brauzerga **`"000000000000000000000000"`** bo'lib boradi — JavaScript'da bu
**truthy**. Shu sababli `if (order.courierId)` doim rost bo'ladi va har bir
call site'da to'g'ridek ko'rinadi.
- Frontendda **doim `hasId()` / `realId()`** (`lib/id.ts`) ishlatiladi, yalang'och
  truthiness emas.
- Bu xato bir marta jiddiy oqibat bergan: `pinned: !!me?.branchId` tufayli
  **har bir admin filialga biriktirilgandek** ko'rinib, filial almashtirgichi
  bosilmaydigan yorliqqa aylangan edi.

### ⚠️ Tuzoq: alpine konteynerda vaqt mintaqasi jimgina UTC bo'ladi
`TZ=Asia/Tashkent` berilgan bo'lsa ham, `alpine` image'ida **`tzdata` yo'q** —
Go mintaqani topa olmay `time.Local` ni UTC qoldiradi va **hech qanday xato
bermaydi**. Natijada har bir ishchi smenasi besh soat oldingi kunga yozilardi.
Birinchi prod deploy'da aynan shu chiqdi.
- `cmd/server` da **`import _ "time/tzdata"`** — baza binaryning ichida, ya'ni
  qanday image bo'lishidan qat'i nazar ishlaydi.
- Ikkala Dockerfile'da ham `tzdata` paketi (konteyner soati va `date` uchun),
  frontend konteyneriga ham `TZ` beriladi (SSR sanani shu soat bilan chizadi).
- Server boot'da **vaqt mintaqasini log qiladi** — keyingi xato ko'rinib tursin.
- Host'ning tizim vaqtini o'zgartirish shart emas va tavsiya etilmaydi (bir
  VPS'da boshqa saytlar bo'lishi mumkin): mintaqa konteyner darajasida beriladi.

### ⚠️ Tuzoq: Mongo'dan kelgan sana **doim UTC**, `TZ` to'g'ri bo'lsa ham
Yuqoridagi tuzoqning ikkinchi yuzi va u `TZ` tuzatilgandan **keyin** ham
qoladi: drayver har qanday `time.Time` ni **UTC location** bilan dekod qiladi.
Ya'ni mahalliy yarim tunda yozilgan sana `19:00 (oldingi kun)` bo'lib qaytadi,
va undan kun boshini olish butun oynani bir kun oldinga suradi.
- **Jimgina yiqiladi**: oyna baribir haqiqiy oy bo'lib qoladi, faqat noto'g'ri
  oy. Keel'da bu har mijozning hisob davri bir kun erta boshlanishi edi.
- Qoida: bazadan kelgan vaqt **ishlatilishidan oldin `.In(time.Local)`**
  (`control/internal/handlers/billing.go` → `local()`).
- **Brauzerga ham shu tegadi**: `time.Time` JSON'ga UTC bo'lib chiqadi, shuning
  uchun `subscribedAt.slice(0,10)` oldingi kunni beradi. Sana kerak bo'lsa
  server **tayyor `"YYYY-MM-DD"` qatorini** yuboradi (`period.anchor`), brauzer
  timestamp'dan kesib olmaydi.
- Testda ushlash uchun sana **`.UTC()` bilan** beriladi — aynan drayver
  qaytaradigan ko'rinishda (`TestTenantPeriodAnchorsFromUTCDates`).

### ⚠️ Tuzoq: Next.js `rewrites()` build vaqtida muhrlanadi
`next.config.ts` dagi `rewrites()` **build paytida** marshrutlar manifestiga
yoziladi. Ya'ni ichida `process.env.X` ishlatilsa, konteynerga ish vaqtida
berilgan `X` **umuman e'tiborga olinmaydi** — build paytidagi qiymat (yoki
standart) muhrlanib qoladi.

Keel'da bu konsolga kirishni buzgan edi: `/api/*` Next orqali `localhost:9000`
ga ketardi va brauzerga JSON o'rniga "Internal Server Error" qaytardi.
- Qoida: **manzillarni chekka (Caddy) yo'naltirsin**, Next emas. Shunda
  frontend hech nima qayerdaligini bilishi shart emas va o'zgartirish uchun
  qayta build kerak bo'lmaydi.
- `NEXT_PUBLIC_*` ham xuddi shunday — bundle'ga kiradi. Har tenantga har xil
  bo'lishi kerak bo'lgan qiymat (masalan 2GIS kaliti) **build'ga emas,
  ma'lumotga** joylashtiriladi.

### ⚠️ Tuzoq: `docker compose build` + `up -d` konteynerni almashtirmaydi
Alohida chaqirilganda compose yangi image quradi va **eski konteynerni ishlab
turgan holda qoldiradi**. Natijada tag bitta image'ni, konteyner ikkinchisini,
compose'ning o'z yorlig'i uchinchisini ko'rsatadi.

Bu eng xavfli xato turi: commit to'g'ri, konteynerlar sog'lom, sog'liq
tekshiruvlari yashil — **yangi kod esa ishlamayapti**.
- Quriladigan xizmatlar `up -d --force-recreate --no-deps <xizmatlar>` bilan
  **aniq** almashtiriladi.
- `--no-deps` mongo va caddy'ni chetda qoldiradi: mongo hostdagi hamma bazani
  tutadi, caddy restart chekkani bootstrap konfiguratsiyasiga qaytaradi.
- Bir vaqtda ikki deploy ketmasin — `flock`. Compose konteynerni almashtirishdan
  oldin qayta nomlaydi, ikkinchi yugurish esa o'sha nomni band topadi.
- ⚠️ **Qulf yo'li qat'iy bo'lishi shart** (`/opt/keel/.deploy.lock`), `$HOME`
  emas. `${HOME}/...` **serverni emas, foydalanuvchini** qulflaydi: CI
  `deploy-keel` nomidan, odam esa `root` nomidan ishlaydi, ya'ni ikkalasi
  boshqa fayl oladi va bir-birini umuman ko'rmaydi. Ikkala qulf ham serverda
  yotardi, har biri mukammal ushlangan, hech nimani himoya qilmasdi — va
  oldini olishi kerak bo'lgan yagona to'qnashuv aynan "odam CI bilan bir vaqtda
  deploy qiladi". **Hech kim talashmagan qulf ishlaydigan qulfdan farq
  qilmaydi.**
- ⚠️ **`git HEAD` deploy tugaganini bildirmaydi.** Kod yangilash — skriptning
  birinchi qadami, to'rt image undan keyin quriladi. Push'dan yarim daqiqa
  keyin serverdagi HEAD to'g'ri bo'ladi-yu konteynerlar hali eski. To'g'ri
  belgi: **qulf bo'shadimi** va konteynerlar yoshi.

### ⚠️ Tuzoq: birgalikda bajarilmasligi kerak bo'lgan ikki ishni `else if` bog'lash
`apply()` da konteynerni ko'tarish va Caddy'ni qayta yozish `else if` bilan
zanjirlangan edi — ya'ni konteyner ko'tarilmasa **chekka umuman qayta
yozilmasdi**. `syncEdge` esa butun platformaning konfiguratsiyasini qayta
yozadi, demak bitta kasal mijoz **barcha** mijozlarning domen o'zgarishini
jimgina to'xtatib qo'yardi.

Aynan shu tarzda jonli domen yo'qoldi: ega ulaydi, baza qabul qiladi, javob
`ok` deydi, konsolda ro'yxatda turadi — Caddy esa u haqda hech nima eshitmagan.
Yagona alomat: sayt HTTPS'da ochilmaydi, va bu DNS muammosi yoki sekin
sertifikatga o'xshaydi, ya'ni **birovning aybiga**.
- Mustaqil ravishda yiqiladigan ikki ish mustaqil bajariladi, va xato **qaysi
  yarimda** ekani yoziladi — savol birinchi navbatda shu, va tuzatishlari
  butunlay boshqa.
- Chekka konfiguratsiyasi tenant ro'yxatidan (Mongo) olinadi, konteyner nima
  qilishidan emas — ya'ni ular haqiqatan bog'liq emas.

### ⚠️ Saqlangan bayroq eskiradi (`provisionStatus`) — tuzatildi
`kfc` tenantida `provisionStatus: "ready"`, `provisionError: ""` turgan holda
konteyner **umuman yo'q** edi. Konsol mijozni to'liq ishlayapti deb ko'rsatardi.

Bu `Attention` da ataylab qo'llanilgan darsning teskarisi: "saqlangan bayroq
soat undan o'tishi bilan eskiradi, sana esa eskirmaydi".

Endi `attention: "down"` bor va u **Docker'dan jonli** o'qiladi
(`Docker.States()` — butun sahifa uchun **bitta** so'rov, har tenantga alohida
emas). Qoidalari:
- **Hamma narsadan oldin tekshiriladi**, hatto bepul shartlardan ham: qorong'i
  sayt puldan muhimroq, va bepul mijoz *to'lov uchun ta'qib qilinishdan* ozod,
  ishlaydigan saytdan emas — u odatda aynan anchor mijoz.
- ⚠️ **Bo'sh qator "o'chiq" emas.** Docker soketi yo'q — noutbukdagi normal
  holat, va doim yonib turgan ogohlantirishni hech kim o'qimaydi. `stateOf()`
  "so'ralmagan" (bo'sh xarita) bilan "ro'yxatda yo'q" (`absent`) ni ajratadi.
- **`restarting` ham o'chiq hisoblanadi**: qayta ishga tushib turgan konteyner
  oraliqda o'zini ishlayapti deb ko'rsatadi — buzilgan tenant sog'lomdan aynan
  shunday farqlanmay qoladi.
- `suspended`/`deleted` tekshirilmaydi: ular **biz** o'chirgan saytlar.
- Ekranda yagona **to'ldirilgan** nishon: yonida uchta pushti nishon turganda
  pushti fon ko'rinmay ketadi — aynan shu bo'lgan edi.
- ⚠️ `invoice_due` kabi, bu ham Mongo filtri **bo'la olmaydi** (holat hech
  qayerda saqlanmaydi) — filtr Go tomonda.

### ⚠️ Tuzoq: SSH kalitni parolingizdan oldin sinaydi
"Parol bilan ulandim" — tekshirilmaydigan taxmin. SSH avval `ssh-agent` dagi va
standart nomli kalitlarni taklif qiladi, va agentdagi **tartib ko'rinmaydi**.
Agar birinchi mos kelgan kalit `authorized_keys` da `command="..."` bilan
cheklangan bo'lsa, har ulanish o'sha buyruqqa aylanadi — parol umuman
ishtirok etmaydi.

Keel'da bu uch marta ataylanmagan prod deploy'ga sabab bo'lgan, va diagnoz
ikki marta noto'g'ri qo'yilgan (`sshd_config` da `ForceCommand`, keyin
`~/.ssh/rc` — ikkalasi ham yo'q edi).
- Ajratish uchun: `ssh -o PubkeyAuthentication=no -o PreferredAuthentications=password`
- Cheklovni **kalitga** qo'ying (`restrict,command="..."`), global
  `ForceCommand` ga emas — u hammani, jumladan sizni ham qulflaydi.

### ⚠️ Xarita kaliti sir emas va sir bo'la olmaydi
MapGL — brauzer kutubxonasi: kalit sahifada `load({ key })` ga uzatiladi va
DevTools'da ko'rinadi, uni qayerda saqlashimizdan qat'i nazar. Hech bir xarita
SDK'si (Google, Yandex, Mapbox) boshqacha ishlamaydi.
- Himoyani **provayder kabinetidagi domen cheklovi** beradi, yashirinlik emas
  (uchalasida ham shunday).
- Shuning uchun uchala kalit ham (`mapApiKey`, `mapYandexKey`, `mapGoogleKey`)
  ataylab `restaurant` hujjatining ichida va brauzerga qaytariladi — bu **to'lov kalitlarining teskarisi**, ular
  o'sha hujjatdan ataylab chiqarilgan (`payment_settings`). Ikkalasining sababi
  bir xil: hujjat har tashrifchiga to'liq boradi.
- Buni "xavfsizlik tuzatishi" deb yashirsangiz, xarita ishlamay qoladi.


---

## 11. Xususiyatlar bo'yicha qarorlar → `docs/DECISIONS.md`

Har bir bo'limning qarorlari, muqobillari va tuzoqlari **`docs/DECISIONS.md`**
da. ⚠️ **U har sessiyada o'qilmaydi** — ish qaysi bo'limga tegsa, o'sha
sarlavha `grep` bilan topiladi va **faqat o'sha bo'lim** o'qiladi:

```bash
grep -n '^### ' docs/DECISIONS.md          # sarlavhalar ro'yxati
sed -n '/^### Stop list/,/^### /p' docs/DECISIONS.md
```

Kod o'zgartirishdan **oldin** tegishli bo'lim o'qiladi: u yerda ko'p qaror
"bir marta bo'lib o'tgan xato" ustiga qurilgan, va koddan ko'rinmaydi.

**Qayerda nima bor:**

| Ish qayerga tegsa | `docs/DECISIONS.md` dagi bo'lim |
|---|---|
| Brend / filial qamrovi, chas pik | Brend va filial · Chas pik |
| Buyurtma oqimi, holatlar, manzil | Buyurtmalar oqimi · Buyurtma manzilini xaritada tuzatish · Oldindan buyurtma |
| Menyu, qidiruv, variant, combo, izoh | Menyu qidiruvi va filtrlar · Menyu variantlari · Ulushlab sotish · Combo · Taomga izoh va bekor qilish sababi |
| Narx, chegirma, ball | Chegirmalar · Loyalty |
| Stol: QR, bron, zal | QR menyu · Stol bron qilish · Kassa (POS) va zal |
| Kassa cheki, smena, qarz, X/Z | Kassa (POS) va zal · Moliyaviy hisobot va kassa |
| Ombor, tannarx, sanash | Tannarx va ombor |
| Kartasiz sotuv, qamrov, manfiy qoldiq | Ombor qamrovi: sotuvning qancha qismi kartalar bilan qoplangan |
| Spisaniya, void, chek bekor, backfill | Spisaniya hujjati: chekka urilganda yoziladi |
| Harakat hisoboti, partiya, kunlik sotuv | Harakat hisoboti nima uchun o'z jamiga yetmasdi |
| Texkarta: zagotovka, taom kartasi | Texkarta o'z ekranida |
| Bilim bazasi, yordam, screenshot | Bilim bazasi (keel.uz/help) |
| Markaziy oshxona, tsex, partiya | Markaziy oshxona (tsex): partiya va ishlab chiqarish hujjati |
| Stop list (3 ro'yxat) | Stop list · Kassa buyurtmani qabul qildimi |
| POS: iiko/Syrve/Poster/Clopos/r_keeper | POS integratsiyasi · Kassa buyurtmani qabul qildimi |
| Onlayn to'lov, callback | Onlayn to'lov: Payme / Click / Uzum / ATMOS |
| Kassada karta: QR skanerlash, bank terminali | Kassada karta: QR skanerlash (Click Pass / Uzum FastPay) |
| Tez bosganda qotish, zoom, copy (kassa/zal/KDS/kiosk) | Kassa, zal, oshxona, kiosk: tez bosganda qotib qolish |
| SMS, mijoz auth, admin parol | SMS provayderi · Mijoz auth · Admin parolini tiklash |
| Telegram bot, mini app, til | Telegram bot va mini app · Bot javob berishi (webhook) · Mini app'da til |
| Telefon, call-markaz, ATS | Call-markaz · Telefoniya: onlinePBX |
| CRM, segment, kampaniya, push, upsell | CRM · Segmentlarga xabar yuborish · RFM · Web push · Upsell |
| Kuryer, tashqi yetkazish | Kuryerlar va rollar · Kuryer PWA · Tashqi yetkazish xizmatlari · Joylashuvga ruxsat |
| Ishchi, KDS, davomat, kiosk | KDS · Har bir taomning holati · Ishchilar davomati · QR bilan ishga kirish |
| Bozorchi, zakupshik, podotchet | Bozorlik: bozorchi ilovadan yozadi (+ Bozorlik ro'yxati · Podotchet) |
| Panel adminlari, jurnal, eksport | Panel adminlari va amallar jurnali · Ma'lumotni olib ketish |
| Panel roli: operator, omborchi | Panelning cheklangan rollari: ombor va operator |
| TV ekran, Android TV, kontent | TV ekranlar: ulash, uzish va sanash · TV kontent: playlist, muddat va oflayn · TV tablo: qaysi raqam pishmoqda, qaysisi tayyor |
| AI yordamchi, ertalabki brifing | AI yordamchi: ertalabki brifing |
| Qo'llab-quvvatlash, chat, ticket | Qo'llab-quvvatlash: chat va operator konsoli |
| Xatolik hisoboti, crash, konsol reports | Xatolik hisobotlari: konsolga avtomatik tushadi |
| Hisobot, Excel, grafik, dashboard | Hisobotlar va Excel eksporti · Hisobotlar: savdo/kanallar/jamoa · ABC/XYZ · Dashboard statistikasi · Sozlanadigan KPI dashboard · Grafiklar |
| Yangi sahifa / komponent yozish | Dizayn tizimi · Tema (dark/light) · Ko'p tillilik · Til URL'lari · 404 va xatolik sahifalari · Admin ro'yxatlari |
| Sayt ko'rinishi, matn, SEO, rasm | Sayt dizayni · Sayt matnlari · SEO va favicon · Rasmlar (`?w=`) · Tavsiya etilgan rasm o'lchamlari · Sayt konstruktori |
| keel.uz landingi: qaysi bo'lim qayerda | Landing tuzilishi: nima bosh sahifada qoladi |
| Xavfsizlik | Xavfsizlik: filial qamrovi / rate limit / JWT_SECRET · Mijozni o'chirish |
| Xato xabari, server matni, tarjima | Server xabarlari ham uch tilda |
| Keel konsoli, tenantlar, VPS | Konsol xodimlari · VPS resurslari · Buyurtma pulini bekor qilish · Mijozni o'chirish |
| Hamkor, tavsiya, komissiya, varaqa | Hamkorlar: tashqi tavsiya va komissiya · Varaqa · O'sish ekranlari uch tilda |
| Taklif xabari, sovuq yozish | Taklif matni |
| SEO, sitemap, IndexNow, Google | Qidiruv tizimlari |
| Fiskal kassa, provayderlar | Fiskal provayderlar: ro'yxat va kalitlar |
| Markirovka, DataMatrix, skaner | Markirovka (Asl Belgisi) — ichimliklar |
| Menyu importi, havoladan | Menyuni havoladan import qilish |
| Boshqa POS'dan ko'chirish (iiko, Poster…) | Boshqa POS'dan ko'chirish |
| Kesh, siqish, indeks, yuk | Yuk: nima siqiladi, nima keshlanadi |
| Boshqa | ИКПУ · Tashrif hisobi · Maintenance buyruqlari · Namuna menyu · Mehmonlar fikri · Yangi buyurtma ovozi |

### Qo'shni hujjatlar
- **`PROGRESS.md`** — ish jurnali (har katta bosqichdan keyin yangilanadi).
- **`DEPLOY.md`** — prod deploy (VPS / Vercel), `cmd/adminreset`.
- **`SAAS.md`** — Keel platformasi rejasi; **`CONSTRUCTOR.md`** — sayt
  konstruktori; **`POS_INTEGRATIONS.md`** — kassa provayderlari tafsiloti;
  `docs/` — fiskal agent, markirovka, POS reja.
- **`scripts/`** — deploy image'iga kirmaydigan ishchi vositalar, **o'z
  `package.json` i bilan**: `help-screens.mjs` (bilim bazasi uchun panel
  suratlari + annotatsiya koordinatalari), `landing-shots.mjs` (keel.uz bosh
  sahifasidagi o'nta surat), `shot-lib.mjs` (ikkalasining umumiy qismi),
  `check-help.mjs` (uch tilning izchilligi va suratlarning to'liqligi),
  `loadtest.js` (k6 yuk testi — ⚠️ **saqlanadi, chunki boshqa skript bilan
  olingan ikkinchi o'lchov taqqoslash emas**: "kesh shiftni uch barobar
  oshirdi" bilan "men yengilroq test yozdim" tashqaridan bir xil ko'rinadi;
  standart holda **faqat o'qiydi**, buyurtma yozish uchun `-e WRITE=1`). ⚠️ Playwright'ni `frontend/` yoki `keel-site/` ga qo'shib
  bo'lmaydi — ularning Dockerfile'i `npm ci` qiladi.
- **`docs/vendor/`** — provayder hujjatlarining **o'qilgan nusxasi** (manba
  havolasi va sanasi bilan). ⚠️ Saqlanadi, chunki bu saytlar JS bilan
  chiziladigan SPA: `curl` ularda hujjat matnini qaytarmaydi, ya'ni "havolaga
  qara" degan izoh keyingi sessiyada ishlamaydi.

### Yangi qaror qayerga yoziladi
- **Xususiyat qarori, tuzoq, "nega shunday"** → `docs/DECISIONS.md`, tegishli
  `###` bo'limiga (yangi bo'lim bo'lsa — yuqoridagi jadvalga bir qator).
- **Butun loyihaga tegadigan** (model, API konvensiyasi, umumiy tuzoq,
  branch qoidasi) → shu fayl.
- ⚠️ **Ikkalasiga ham emas**: takrorlangan qoida birinchi tahrirda ajraydi va
  keyin qaysi biri rost ekanini aniqlashning yo'li qolmaydi.
