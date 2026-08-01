# CLAUDE.md — Restaurant Website + Delivery System (Template)

Bu fayl loyihani to'liq tavsiflaydi. Har bir yangi sessiyada shu fayl o'qiladi.
Progress / kunlik ish jurnali uchun alohida fayl bor: **`PROGRESS.md`**.

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
├── CLAUDE.md                 # shu fayl
├── PROGRESS.md               # ish jurnali
├── DEPLOY.md                 # deploy qo'llanmasi (VPS / Vercel)
├── docker-compose.yml        # dev: mongo + backend + frontend
├── docker-compose.prod.yml   # prod: portlar faqat 127.0.0.1, .env dan sozlash
├── .env.prod.example         # prod muhit o'zgaruvchilari namunasi
├── nginx/restaurant.conf     # host nginx reverse proxy (TLS, /api, /uploads)
├── frontend/                 # Next.js + TypeScript (public site + admin panel)
└── backend/                  # Go + MongoDB (REST API + rasm upload/serve)
```

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

## 4. Ma'lumotlar modeli (MongoDB kolleksiyalari)

Har bir restoran alohida deployment bo'lgani uchun bitta restoran ma'lumoti bor.

### `restaurant` (bitta hujjat — singleton)
```
_id, name, description, logoUrl, coverUrl,
phone[], address { text, lat, lng },
socials { instagram, telegram, facebook },
workingHours [ { day: 0-6, open: "09:00", close: "23:00", isClosed: bool } ],
isOpenNow (hisoblanadi),
delivery {
  enabled: bool,
  minOrder: number,
  freeDeliveryFrom: number|null,
  mode: "radius" | "zones",   // narx qanday hisoblanadi (bo'sh = zonalarga qarab)
  // Har zona o'z narxlash usuliga ega (restoran tanlaydi):
  //   pricing "fixed"  → fee (zona ichida bir xil narx)
  //   pricing "perKm"  → baseFee + ceil(km) * perKm (restorandan masofa)
  zones: [ { name, polygon: [[lat,lng]...], pricing, fee, baseFee, perKm } ],
  arrivalRadiusM: number,      // kuryer "Yetkazdim" bosishi uchun masofa (0 = o'chiq)
  // zona yo'q bo'lsa — umumiy radius modeli: { baseFee, perKm, maxKm }
},
currency: "UZS",
// Admin panelda tahrirlanadigan sayt matnlari (har biri { uz, ru, en }):
content { tagline, aboutTitle, aboutText, footerNote },
// Sayt dizayni (bo'sh maydon = standart):
theme { brand, brandDark, radius, buttonShape: "pill"|"match", font,
        background: "warm"|"white"|"cool"|"sand",
        shadow: "none"|"soft"|"strong",
        buttonStyle: "solid"|"outline"|"soft", scale },
updatedAt
```

### `category` (menyu kategoriyalari)
```
_id, name, nameRu, nameEn, slug, sortOrder, isActive, imageUrl
```

### `menu_item` (menyu taomlari)
```
_id, categoryId, name, description, price, oldPrice?,
nameRu, nameEn, descriptionRu, descriptionEn,   // tarjimalar (ixtiyoriy)
imageUrl, images[]?, isAvailable, isPopular, sortOrder,
options [ { name, nameRu, nameEn, required, multiple,
            choices: [ { name, nameRu, nameEn, priceDelta } ] } ],  // masalan hajm
tags [], updatedAt
```

### `order` (buyurtmalar)
```
_id, number (human-readable), status,
  // status: pending → confirmed → preparing → on_the_way → delivered → cancelled
statusHistory [ { status, at } ],   // har bir holat qachon qo'yilgani
cancelReason?,                     // bekor qilish sababi (mijozga ko'rinadi)
customer { name, phone },
type: "delivery" | "pickup" | "dinein",   // dinein — QR orqali stolda
tableId?, tableNumber?,            // dinein: qaysi stol (raqam — nusxa)
address { text, lat, lng, comment },
items [ { menuItemId, name, price, qty, comment?,
          options: [ { name, choice, priceDelta } ] } ],   // price = narx + deltalar
          // comment — mijozning shu taomga izohi ("piyozsiz"), savatda yoziladi
subtotal, deliveryFee, total,
paymentMethod: "cash" | "card",
deliveryZone?, distanceKm?,
createdAt, updatedAt
```

### `reservation` (stol bronlari)
```
_id, number, tableId, tableNumber (snapshot),
userId?, customer { name, phone }, guests,
at, endsAt,                       // yarim ochiq oraliq [at, endsAt)
status: pending → confirmed → seated → done | cancelled,
comment, cancelReason?, statusHistory, createdAt, updatedAt
```
Xarita `restaurant.booking` ichida: `{ enabled, width, height, slotMinutes,
maxDaysAhead, minNoticeMinutes, maxGuests, shapes[], tables[] }`.

### `user` (sayt mijozlari)
```
_id, firstName, lastName,
phone (998XXXXXXXXX, SMS bilan tasdiqlangan),
authProvider: "phone",
addresses [ { label, text, lat, lng, comment } ],
createdAt, updatedAt
```

### `phone_code` (bir martalik SMS kodlar)
```
_id, phone, codeHash (bcrypt), attempts, expiresAt (3 daqiqa), createdAt
```

### `delivery_provider` (tashqi yetkazish xizmatlari)
```
_id, name, kind: "link" | "phone" | "api", url, phone, note,
isActive, sortOrder,
// kind: "api" uchun (token hech qachon brauzerga qaytarilmaydi):
apiProvider ("yandex"), apiToken, apiBaseUrl, apiTariff,
createdAt, updatedAt
```
`order.externalDelivery { providerId, providerName, trackingId, note, cost, calledAt }`
— buyurtma tashqi xizmatga berilganda yoziladi.

### `courier` (kuryerlar — admin qo'lda yaratadi)
```
_id, name, phone, username, passwordHash, vehicle,
status: "off" | "free" | "busy",       // ishda emas / bo'sh / band
isActive, location { lat, lng, accuracy, at },
// Har yetkazish uchun qancha oladi (restoran tanlaydi):
payoutMode: "deliveryFee" | "perOrder" | "percent",
payoutPerOrder, payoutPercent,
createdAt, updatedAt
```

### `staff` (ishchilar — oshpaz, ofitsiant, kassir)
```
_id, branchId, name, phone, username, passwordHash,
position,                          // erkin matn: "oshpaz", "ofitsiant"
schedule [ { day: 0-6, start: "11:00", end: "23:00", isOff } ],  // ish grafigi
payMode: "hourly" | "shift" | "monthly",
hourlyRate, shiftRate, monthlyRate,
payPeriod: "daily" | "10days" | "15days" | "monthly",
isActive, createdAt, updatedAt
```

### `shift` (ishga kirish/chiqish yozuvi)
```
_id, staffId, branchId,
date,                              // smena BOSHLANGAN kun, "YYYY-MM-DD" (local)
in, out?,                          // vaqtlar
inAt?, outAt? { lat, lng, accuracy, meters, at },   // tugma qayerda bosilgani
minutes,                           // chiqishda yoziladi
editedBy?, note?,                  // admin qo'lda yozgan/tuzatgan bo'lsa
createdAt, updatedAt
```

### `staff_payment` (ishchiga berilgan oylik)
```
_id, staffId, branchId, amount, from, to,   // qaysi davrni yopadi
paidBy, note, at
```

### `admin_user` (admin panel foydalanuvchilari)
```
_id, username, passwordHash, role: "owner" | "manager",
mustChangePassword,          // birinchi kirishda parol majburiy o'zgaradi
userId,                      // sayt mijozi (telefon bilan kirgan) — kim ekani
name, phone,                 // shu mijozdan ko'chirilgan
createdBy, lastLoginAt?, createdAt
```

### `admin_log` (adminlar amallari jurnali)
```
_id, adminId, adminName, adminRole,
action,        // "order.cancel", "courier.create", "admin.delete" ...
targetType, targetId, targetLabel, details, at
```

---

## 5. REST API (rejalashtirilgan endpoint'lar)

Base: `/api/v1`

### Public (auth kerak emas)
```
GET    /restaurant                 # restoran profili + ish vaqti + isOpenNow
GET    /categories                 # aktiv kategoriyalar
GET    /menu                       # to'liq menyu (kategoriya bo'yicha guruhlangan)
GET    /menu/:id                   # bitta taom
POST   /orders                     # buyurtma yaratish (login talab qilinadi)
GET    /orders/:number             # buyurtma holatini kuzatish (public track)
POST   /delivery/quote             # manzil bo'yicha yetkazish narxini hisoblash

# Mijoz auth (telefon + bir martalik SMS kod)
POST   /auth/phone/request         # kod yuborish  → { ok, phone, demo, code? }
POST   /auth/phone/verify          # kodni tasdiqlash → { token, user }
GET    /users/me                   # profil (JWT)
PUT    /users/me                   # ism + manzillarni tahrirlash (JWT)
POST   /users/me/phone/request     # raqamni o'zgartirish — kod (JWT)
POST   /users/me/phone/verify      # raqamni o'zgartirish — tasdiq (JWT)
GET    /users/me/orders            # buyurtmalar tarixi (JWT)
```

### Kuryer (JWT, rol `courier`)
```
POST   /courier/login              # login + parol → { token, courier }
GET    /courier/me
PUT    /courier/status             # off | free | busy
POST   /courier/location           # { points: [ {lat,lng,accuracy,at} ] } (batch)
GET    /courier/orders             # menga biriktirilgan (?all=1 — tarix bilan)
GET    /courier/stats              # daromad: bugun / 7 kun / 30 kun / jami
GET    /courier/history            # yetkazilganlar + har biridan ishlangani
PUT    /courier/orders/{id}/status # on_the_way | delivered (faqat o'z buyurtmasi)
```

### Ishchi (JWT, rol `staff`)
```
POST   /staff/login                # login + parol → { token, staff }
GET    /staff/me                   # profil + ish joyi (manzil, radius) + ochiq smena
POST   /staff/clock                # { action: "in"|"out", lat, lng, accuracy }
GET    /staff/report               # ?from=&to= — kalendar, yakun, taqqoslash, oylik
```

### Admin (JWT kerak — `Authorization: Bearer <token>`)
```
POST   /admin/login                # login → JWT
GET    /admin/me
PUT    /admin/restaurant           # profil + ish vaqti + delivery sozlamalari
POST   /admin/categories           # CRUD
PUT    /admin/categories/:id
DELETE /admin/categories/:id
POST   /admin/menu                 # taom CRUD
PUT    /admin/menu/:id
DELETE /admin/menu/:id
POST   /admin/upload               # rasm yuklash → { url }
GET    /admin/users                # mijozlar + buyurtmalar soni/summasi (?q=)
GET    /admin/users/{id}           # bitta mijoz: profil + statistika + barcha buyurtmalar
GET    /admin/stats                # dashboard: ?from=&to= (YYYY-MM-DD, ikkisi ham ixtiyoriy)
GET    /admin/alerts               # yangi buyurtma/bron bormi (ovozli bildirishnoma uchun)
GET    /admin/reservations         # ?scope=upcoming|today|past|all &status= &q=
POST   /admin/reservations         # qo'lda (telefon orqali) bron
PUT    /admin/reservations/{id}/status  # pending→confirmed→seated→done | cancelled (sabab majburiy)
DELETE /admin/reservations/{id}
GET    /admin/orders               # ?status= &q= (№/ism/telefon/manzil) &userId= &limit=
GET    /admin/orders/{id}          # bitta buyurtma (chek)
PUT    /admin/orders/:id/status    # holatni o'zgartirish ({status, reason})
                                  # reason — faqat "cancelled" uchun, majburiy
PUT    /admin/orders/{id}/courier  # kuryer biriktirish (bo'sh id = yechish)
PUT    /admin/orders/{id}/address   # xaritadagi nuqtani tuzatish (mijoz adashgan bo'lsa)
PUT    /admin/orders/{id}/external-delivery  # tashqi xizmat chaqirildi (bo'sh id = bekor)
GET    /admin/delivery-providers   # tashqi yetkazish xizmatlari
POST   /admin/delivery-providers
PUT    /admin/delivery-providers/{id}
DELETE /admin/delivery-providers/{id}

GET    /admin/accounts             # panel adminlari (faqat owner)
POST   /admin/accounts             # { userId, username, password, role }
PUT    /admin/accounts/{id}        # rol / parolni tiklash
DELETE /admin/accounts/{id}
GET    /admin/logs                 # amallar jurnali (?adminId= &action= &q= &limit= &before=)
                                  # q — buyurtma №/ID, ism, izoh bo'yicha qidiruv

GET    /admin/staff                # ishchilar + bugungi holati + davr yakuni (?from=&to=&q=)
GET    /admin/staff/{id}           # kartochka: grafik, kalendar, yakun, to'lovlar
POST   /admin/staff                # ishchi yaratish (login + parol + grafik + oylik)
PUT    /admin/staff/{id}
DELETE /admin/staff/{id}           # smenalar saqlanib qoladi
POST   /admin/staff/{id}/shifts    # smenani qo'lda yozish (telefon o'chgan holat)
PUT    /admin/staff/{id}/shifts/{shiftId}
DELETE /admin/staff/{id}/shifts/{shiftId}
GET    /admin/payroll              # hisob-kitob: har ishchi o'z davri bo'yicha
POST   /admin/staff/{id}/payments  # oylik to'landi (yozuv)
DELETE /admin/staff/{id}/payments/{paymentId}

GET    /admin/couriers             # kuryerlar + oxirgi joylashuvi
GET    /admin/couriers/{id}        # kuryer + daromad statistikasi + tarix
POST   /admin/couriers             # kuryer yaratish (login + parol)
PUT    /admin/couriers/{id}        # tahrirlash (parol bo'sh = o'zgarmaydi)
DELETE /admin/couriers/{id}
```

### Static
```
GET    /uploads/<file>             # yuklangan rasmlar
GET    /health                     # healthcheck
```

---

## 6. Frontend sahifalar (routes)

### Public (`frontend/src/app/(site)/...`)
```
/                     # Bosh sahifa: hero, mashhur taomlar, ish vaqti, CTA
/menu                 # To'liq menyu (kategoriyalar + savatga qo'shish)
/menu/[id]            # Taom sahifasi
/bron                 # Stol bron qilish: xarita, bo'sh stolni tanlash, ism+telefon
/cart                 # Savat
/checkout             # Buyurtma: manzil (Yandex), to'lov turi, tasdiqlash
/order/[number]       # Buyurtma holatini kuzatish
/about                # Restoran haqida, manzil (xarita), aloqa
```

### Kuryer PWA (`frontend/src/app/kuryer/...`)
```
/kuryer/login         # login + parol (hisobni admin beradi)
/kuryer               # smena holati, joylashuv indikatori, mening buyurtmalarim
```

### Ishchi PWA (`frontend/src/app/staff/...`)
```
/staff/login         # login + parol (hisobni admin beradi)
/staff               # "Ishga kirish"/"Ishdan chiqish", kalendar, oylik
```

### Admin (`frontend/src/app/admin/...`)
```
/admin/login
/admin                # Dashboard: davr tanlanadi (bugun/7/30 kun/hammasi/oraliq),
                      # buyurtmalar, pul, odamlar, menyu statistikasi + top taomlar
/admin/menu           # Menyu boshqaruvi (CRUD, drag-sort)
/admin/categories     # Kategoriyalar
/admin/orders         # Buyurtmalar (holat o'zgartirish)
/admin/reservations   # Stol bronlari + tanlangan payt uchun xarita
/admin/qr             # QR kodlar: umumiy yoki har stol uchun (fon + matnlar, PNG)
/admin/users          # Foydalanuvchilar (telefon, buyurtmalar soni)
/admin/users/[id]     # Mijoz kartochkasi: ro'yxatdan o'tgan sana, manzillar,
                      # har bir buyurtma cheki (ID, taomlar, summalar, vaqt jadvali)
/admin/couriers       # Kuryerlar: CRUD + jonli xarita (har 15s yangilanadi)
/admin/couriers/[id]  # Kuryer kartochkasi: daromad (bugun/7/30/jami),
                      # naqd qo'lida, va har bir yetkazish cheki bilan
/admin/staff          # Ishchilar: CRUD, ish grafigi, bugungi davomat taxtasi
/admin/staff/[id]     # Ishchi kartochkasi: kalendar, taqqoslash, smenani tuzatish,
                      # oylik va to'lovlar tarixi
/admin/payroll        # Hisob-kitob: kimga qancha to'lash kerak, to'lov yozuvi
/admin/admins         # Panel adminlari (faqat owner): sayt foydalanuvchisidan
                      # admin yaratish, rol, parolni tiklash
/admin/logs           # Amallar jurnali (faqat owner): kim, qachon, nima qildi
/admin/settings       # Profil, ish vaqti, delivery zonalari, delivery narxi
```

---

## 7. Delivery / Map logikasi

**Map provider: 2GIS (MapGL)** — almashtiriladigan `MapProvider` moduli orqali.
Sabab: MapGL kutubxonasi cheksiz bepul, O'zbekiston (Toshkent/Samarqand) qamrovi
kuchli. Kelajakda Yandex yoki OSM/Leaflet ga faqat bitta modulni almashtirib
o'tish mumkin. Narx (2026): bitta restoran hajmida (oyiga ~15k–60k so'rov) 2GIS
ham, Yandex ham amalda bepul; Yandex geocoder'i alohida pullik bo'lgani uchun
2GIS tanlandi.

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
NEXT_PUBLIC_MAP_API_KEY=...   # 2GIS API key (dev.2gis.com, bepul)
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
- Har bir katta ish bosqichidan keyin **`PROGRESS.md`** yangilanadi.
- Pul birligi: **UZS** (so'm), butun son (tiyin ishlatilmaydi).

### ⚠️ Tuzoq: Go'ning JSON marshalling odatlari (ikki marta tishlagan)
**1) Nil slice `null` bo'lib chiqadi**, `[]` emas. Ro'yxat qaytaradigan har bir
maydon **bo'sh massivga aylantirilishi** kerak (masalan `bookingSettings()`
`Tables`/`Shapes` ni to'ldiradi), aks holda frontendda `null.length` yiqiladi.

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

### Brend va filial (ko'p brend / ko'p filial)
- **Kompaniya** (`restaurant` singleton) — valyuta, ijtimoiy tarmoqlar,
  mijozlar bazasi. **Brend** (`brand`) — menyu, nom, logo, sayt matnlari,
  tema, xizmat turlari. **Filial** (`branch`) — manzil, telefon, ish vaqti,
  yetkazish zonalari va narxi, stol xaritasi, kuryerlar, buyurtmalar.
- **Bitta brend + bitta filialli mijoz hech qanday murakkablikni ko'rmaydi**:
  almashtirgichlar umuman render qilinmaydi, so'rovlar filtrsiz ketadi.
  Bu — butun xususiyatning asosiy sharti.
- Backend linzasi: `internal/handlers/scope.go` → `Scope{BrandID, BranchID}`,
  `adminScope(r)` (`?brandId=`/`?branchId=`) va **`clampToAdmin`** — filialga
  biriktirilgan menejer URL orqali kengaya olmaydi.
- **Mijoz filialni tanlamaydi**: yetkazishda `deliveryBranch` manzilni qamrab
  oladigan filiallardan **eng yaqinini** tanlaydi; olib ketish va stolda esa
  mijoz qayerda turganini o'zi biladi (`branchId` so'rovda).
- **Buyurtmaning brendi savatdagi taomlardan** olinadi, brauzer yuborgan
  maydondan emas. Ikki brend aralashgan savat → 400.
- **Savat brend bo'yicha alohida**: `localStorage` kaliti `cart_v2:<brandId>`.
- Saytning linzasi **cookie'da** (`brand`, `branch`) — til bilan bir xil
  naqsh, chunki menyu server'da render qilinadi. `lib/siteBrand.ts` (client)
  va `lib/siteBrand.server.ts` (`getSiteScope()`).
- `GET /restaurant` javobiga filialning manzili/ish vaqti/yetkazish sozlamalari
  va brendning yuzi qatlanadi (`applyBrand`) — shu sabab sayt sahifalari
  o'zgarmadi. Panel uchun `?raw=1` (qatlamsiz).
- `PUT /admin/restaurant` — **qisman yangilash**: faqat yuborilgan maydonlar
  yoziladi. To'liq `$set` sozlamalar sahifasi endi yubormaydigan maydonlarni
  (brend/filialga ko'chganlarini) bo'sh qatorga aylantirib yuborardi.
- Panelda linza `lib/api.ts` dagi `setAdminScope()` da; `scope: true` bergan
  chaqiruvlarga avtomatik qo'shiladi. `AdminScopeProvider` uni **render
  vaqtida** beradi, `scopeKey` esa ekranni qayta yuklaydi.
- **"Tugadi"** (`branch.soldOut []`): menyu brendniki, qozonda nima qolgani
  filialniki. `isAvailable` — taom umuman yo'q; `soldOut` — shu filialda
  bugun tugadi. Bayroq public menyuda hisoblanadi (bazada yo'q), asosiy
  tekshiruv esa `CreateOrder` da, **filial aniqlangandan keyin**.
  Alohida endpoint `PUT /admin/branches/{id}/sold-out` (`$addToSet`/`$pull`);
  `AdminUpdateBranch` `soldOut` ni **ataylab yozmaydi** — ochiq turgan forma
  oshxona tugatgan taomlarni qaytarib qo'yardi.
- **Filial prefiksi** `branch.code` (A–Z0–9, 6 belgigacha) buyurtma va bron
  raqami oldiga qo'yiladi: `MRC-YS19-1745`. Bo'sh = prefiks yo'q.
- **Rollar**: `requireOwner` — kompaniya profili, brend CRUD, filial
  ochish/o'chirish; `requireBranchAccess` — menejer faqat o'z filialini.
  Panelda `branchId` qo'yilgan menejer uchun linza qulflanadi (almashtirgich
  o'rniga filial nomi) va kompaniya/brend bo'limlari ko'rinmaydi.

### Dizayn tizimi (frontend)
- Ranglar `tailwind.config.ts` da: `brand` (aksent, har restoran uchun
  o'zgartiriladi), `ink` (to'q iliq ko'mir), `cream` (fon). Sahifalarda
  to'g'ridan-to'g'ri `neutral-*`/`rose-*` ishlatilmaydi — faqat shu tokenlar.
- **Chegaralar `border-line` / `border-line-strong`** (`--line`,
  `--line-strong`). Ular `ink/[0.07]` kabi alpha bilan emas, tayyor rang
  sifatida berilgan: qorong'i sirtda 7% chiziq ko'rinmaydi, shuning uchun
  ikki temada alpha ham har xil. `border-ink/*` ishlatilmaydi.
- **Sahifa foni doim `bg-cream`** (`--bg`), kartochkalar `bg-surface`.
  `bg-ink/5` ni sahifa foni sifatida ishlatmang — dark rejimda u oq qatlam
  bo'lib `--surface` bilan ustma-ust tushadi va kartochkalar yo'qoladi
  (aynan shu xato bo'lgan edi).
- Takrorlanuvchi klasslar `globals.css` `@layer components` da:
  `.btn / .btn-primary / .btn-ghost / .btn-dark`, `.card`, `.badge*`, `.chip`,
  `.eyebrow`, `.section-title`, `.input`, `.container-page`.
- Shriftlar: **Inter** (matn) + **Playfair Display** (sarlavha, narx) —
  `next/font/google`, `font-sans` / `font-display`.

### Sayt dizayni (admin tomonidan o'zgartiriladi)
- `restaurant.theme` — asosiy rang, burchak yumaloqligi (px), tugma shakli
  (pill / kartochkalarga mos), shrift juftligi (`classic`/`modern`/`soft`),
  fon ohangi, kartochka soyasi, tugma to'ldirilishi va umumiy o'lcham (root
  font-size). Admin panelda 5 ta **tayyor mavzu** ham bor — ular shu
  maydonlarning tayyor to'plami, xolos.
- Soyalar (`--shadow-card`), tugma ko'rinishi (`--btn-bg/fg/border`) va
  umumiy o'lcham (`--root-size`) ham CSS o'zgaruvchilari — Tailwind
  `shadow-card` shulardan o'qiydi.
- `lib/theme-css.ts` bu sozlamalarni CSS o'zgaruvchilariga aylantiradi va
  `app/layout.tsx` uni **`<head>` ichida inline** beradi — birinchi bo'yashdayoq
  qo'llanadi, ya'ni standart rang "chaqnab" o'tmaydi. Qorong'i tema uchun
  aksent avtomatik ochiqroq qilinadi (aks holda qora fonda yo'qoladi).
- **Burchaklar butun ilova bo'ylab bitta joydan boshqariladi**: `tailwind.config.ts`
  da `rounded-xl/2xl/3xl` → `--radius-md/lg/xl`. Ya'ni komponentlarga tegmasdan
  butun sayt shakli o'zgaradi. `rounded-full` chin pill bo'lib qoladi;
  tugmalar `--radius-btn` ni ishlatadi.
- Shriftlar `next/font` bilan build vaqtida yuklanadi (Inter, Playfair, Nunito),
  tanlov faqat o'zgaruvchini almashtiradi.
- Admin: `/admin/settings` → "Sayt dizayni" (`components/admin/DesignEditor.tsx`),
  jonli ko'rinish bilan (light va dark).

### Sayt matnlari
- `restaurant.content` — bosh sahifadagi shior, "Biz haqimizda" sarlavhasi va
  matni, footer matni. Har biri `{ uz, ru, en }`; bo'sh RU/EN o'zbekchasiga
  tushadi (`lib/i18n/site-content.ts` → `localized()`), bo'sh uz esa standart
  matnga.
- Telefon, manzil, ijtimoiy tarmoqlar, ish vaqti — allaqachon sozlamalarda
  (footer va "Biz haqimizda" o'sha ma'lumotdan o'qiydi).

### Tema (dark / light)
- ⚠️ **Hidratsiya tuzog'i**: `(site)/layout.tsx` dagi `<Suspense>` Header'ni
  **kech gidratlaydi**, root layout'dagi `ThemeProvider`/`CartProvider`
  effektlari esa undan **oldin** ishlab bo'ladi. Shuning uchun Suspense
  ichidagi komponentda **ota-provayderning `mounted` bayrog'iga tayanib
  bo'lmaydi** — u allaqachon rost bo'ladi.
  Ikki to'g'ri yo'l: (a) markup holatga umuman bog'liq bo'lmasin va CSS hal
  qilsin — `ThemeToggle` ikkala ikonkani ham chizadi, `dark:hidden` /
  `dark:block` tanlaydi; (b) bayroq **shu komponentning o'ziniki** bo'lsin —
  Header'dagi savat nishonchasi shunday. Yorliqlar ham holatni emas, amalni
  nomlasin ("Mavzuni almashtirish").
- Semantik ranglar — `globals.css` dagi CSS o'zgaruvchilari (`--bg`,
  `--surface`, `--fg*`, `--brand*`); Tailwind `darkMode: "class"`.
  Doim qorong'i sirtlar uchun alohida `charcoal` tokeni (hero, footer).
- `<html class="dark">` — `lib/theme.tsx` (localStorage + tizim sozlamasi),
  birinchi bo'yashdan oldin `app/layout.tsx` dagi inline skript qo'yadi.
- Yangi sahifa yozganda: `bg-white` emas `bg-surface`, `text-neutral-*` emas
  `text-ink/ink-soft/ink-muted` ishlating.

### Ko'p tillilik (UZ / RU / EN)
- Lug'atlar: `src/lib/i18n/dictionaries.ts` (`Dict` tipi `uz` dan olinadi).
- Til `lang` cookie'da. Server komponentda `getTranslations()`
  (`lib/i18n/server.ts`), client komponentda `useI18n()`
  (`lib/i18n/client.tsx`). Almashtirgich: `components/site/LangSwitch.tsx`.
- Narx/sana/hafta kuni: `formatPrice(x, currency, lang)`,
  `weekdayName(day, lang)`, `t.locale`.
- **Vaqt doim 24 soatlik** — `formatTime` / `formatDate` / `formatDateTime`
  (`lib/format.ts`), qo'lda formatlangan (`HH:MM`, `DD.MM.YYYY`).
  `toLocaleTimeString` **ishlatilmaydi**: u qurilma tiliga ergashadi va inglizcha
  telefonda "1:16 PM" chiqaradi — bir ekranda "11:00 — 23:00", boshqasida
  "11:00 AM — 11:00 PM" turgan smena jadvali jadval emas. Qo'lda formatlash
  bir vaqtning o'zida SSR/brauzer hidratsiya farqini ham yo'q qiladi
  (`formatPrice` `Intl` dan qochgani bilan bir sabab).
- **Menyu kontenti ham ko'p tilli**: `category` da `nameRu`/`nameEn`,
  `menu_item` da `nameRu`/`nameEn`/`descriptionRu`/`descriptionEn`.
  O'zbekcha — asosiy (base); tarjima bo'sh bo'lsa avtomatik o'zbekchasi
  ko'rsatiladi (`lib/i18n/content.ts` → `contentName`/`contentDescription`).
  Admin panelda har bir taom/kategoriya formasida "Tarjimalar" bloki bor.
  Buyurtmaga esa **doim base (uz) nomi** yoziladi — admin uchun yagona til.
- **Admin panel va kuryer ilovasi ham uch tilli**: alohida lug'at
  `src/lib/i18n/admin.ts` (`useAdminT()`), mijoz lug'atidan ajratilgan —
  auditoriyasi boshqa. O'zbekcha manba: `AdminDict` shundan olinadi, ru/en da
  kalit tushib qolsa kompilyatsiya xatosi. Til bir xil `lang` cookie'da, ya'ni
  saytda tanlangan til panelga ham o'tadi. Har ikkalasida `LangSwitch` va
  `ThemeToggle` bor (kirish sahifalarida ham).
- Buyurtma holati yorliqlari (`status`) va "keyingi qadam" tugmasi matnlari
  ham lug'atda; `nextActionLabel(order, t.nextAction)` ko'rinishida chaqiriladi.

### Chegirmalar: promokod va aksiya
- **Bitta model, ikkita tetik** (`promotion`): `trigger: "code"` — mijoz
  kiritadi, `trigger: "auto"` — o'zi ishlaydi. Kassaga ikkalasi bir xil.
- **Yagona narxlash quvuri** `handlers/pricing.go` (`computePrice`):
  `subtotal → eng foydali avtomatik aksiya (faqat BITTA) → promokod →
  yetkazish`. Har qadam faqat **qolganini** oladi, total 0 dan past tushmaydi.
  Ikkita ustma-ust kampaniya yarim narx yasab yubormasligi uchun avtomatik
  aksiyalardan faqat eng foydalisi qo'llanadi.
- `POST /orders/quote` — checkout'ning preview'i, **aynan shu quvurni** yuritadi;
  `CreateOrder` esa hammasini qaytadan hisoblaydi (brauzerga ishonilmaydi).
- **Minimal buyurtma chegirmadan keyin** tekshiriladi: restoranga tushadigan
  pul muhim (`payableSubtotal`, `belowMinimum`).
- Har chegirma `order.discounts` da **nomi va summasi bilan nusxa** — kampaniya
  o'chirilsa ham chek o'zini tushuntiradi.
- Xato promokod halokat emas: buyurtma to'liq narxda o'tadi, `codeError` da
  aniq sabab bo'ladi.
- `usageLimit` `$expr` bilan filtrda qo'riqlanadi; hisob buyurtma
  yaratilgandan **keyin** oshadi. `AdminUpdatePromotion` `usedCount` ni
  formadan qayta yozmaydi.
- Public `GET /promotions` — **faqat aksiyalar**; kodlar hech qachon ro'yxatda
  qaytarilmaydi.
- Panel: `/admin/promotions` (ikki tab, bitta forma).

### CRM: segmentlar, kartochka, fikrlar
- **Segmentlar hisoblanadi, saqlanmaydi** (`handlers/crm.go`). Har biri bitta
  jumlada tushuntiriladi va panelda ta'rifi ko'rsatiladi — egasi ta'riflay
  olmaydigan guruhga hech nima yubormaydi.
- Mijoz **bir vaqtda bir nechta segmentda** bo'ladi ("uxlab qolgan VIP" —
  aynan shu juftlik muhim).
- **VIP nisbiy** (eng yuqori 10%), qat'iy summa emas: u restoran turiga qarab
  boshqacha va narxlar oshgani sari eskiradi. Baza <10 kishi bo'lsa VIP yo'q.
- `user.birthday` — **`MM-DD`, yilsiz**. Tabrik uchun kun yetarli.
- Ism/telefon paneldan tahrirlanmaydi (telefon SMS bilan tasdiqlangan).
- `user.source` birinchi buyurtmada avtomatik, faqat bo'sh bo'lsa.
- **Fikr**: `feedback`, bitta buyurtmaga bitta. Kuzatuv sahifasida so'raladi;
  yaxshi baho bir bosishda, izoh maydoni faqat past bahoda. Hisobga tegishli
  buyurtmada egasining tokeni majburiy.
- **Yopishda "nima qilindi" majburiy** — yozuvsiz "ko'rib chiqildi" ro'yxatni
  chiroyli qiladi-yu foyda bermaydi. Panel javobsiz shikoyatlardan ochiladi.
- Segment **"norozi"** — 60 kun ichida javobsiz shikoyat. Yagona segment
  mijozning emas, restoranning o'z xatti-harakati haqida.

### Loyalty: keshbek ballari
- **1 ball = 1 so'm** — kurs o'rganish shart emas. Sozlamalar kompaniya
  darajasida (`restaurant.loyalty`), chunki mijoz ham shunday.
- **Ball buyurtmada yechiladi, yetkazilganda qo'shiladi**: yechish darhol
  bo'lmasa bir balans ikki buyurtmaga va'da qilinardi; qo'shish kutmasa bekor
  qilingan buyurtma yo'qdan ball yasardi. Ikkalasi ham **idempotent**
  (`loyalty_txn` qo'riqlaydi, buyurtmadagi bayroq emas).
- **Ball bilan to'lash ball keltirmaydi**: keshbek bazasi
  `subtotal − chegirmalar − ishlatilgan ballar`. Aks holda balans o'z-o'zini
  boqadi.
- **`maxRedeemPercent` shifti majburiy** (standart 50%) — shiftsiz katta
  balans buyurtmani bepul qiladi, oshxona esa pishiradi.
- Har harakat `loyalty_txn` da (`earn`/`spend`/`revoke`/`adjust`), balans
  foydalanuvchida `$inc` bilan siljiydi. Ledger — yozuv, balans — kesh.
- Narxlash quvurida ballar **chegirmalardan keyin**: ball mijozning o'z puli,
  aksiya baribir olib tashlaydigan summaga sarflanmasligi kerak.
- Bekor qilinganda: ishlatilgan ball qaytadi, berilgan keshbek olinadi.

### Combo (belgilangan to'plam)
- Combo — **alohida kolleksiya emas**, `menu_item` ning bir turi:
  `comboItems [{menuItemId, qty}]` bo'sh bo'lmasa bu to'plam. Shu sabab rasm,
  tarjima, kategoriya, brend qamrovi, savat va buyurtma — hammasi tayyor ishlaydi.
- `comboBasePrice` va `comboContents` — **hisoblanadi, saqlanmaydi**
  (`bson:"-"`, `handlers/combo.go`). Saqlangan tejash a'zo narxi o'zgarishi
  bilan yolg'onga aylanadi.
- A'zosi tugagan yoki menyudan o'chirilgan to'plam sotilmaydi; a'zoni o'chirishga
  urinish 409 qaytaradi. Chekka tarkib **nusxa** bo'lib yoziladi — oshxona
  chekdan pishiradi.
- **Majburiy variantli taom to'plamga kirmaydi** ("qaysi hajm?" degan savolni
  belgilangan to'plamda so'rash joyi yo'q). Guruhdan tanlanadigan to'plam —
  ataylab qurilmagan.
- Panel: `components/admin/ComboEditor.tsx`, menyu formasida "Taom / To'plam".

### Menyu variantlari (options)
- Taomga variant guruhlari qo'shiladi: `required` (tanlash shart) va
  `multiple` (bir nechta tanlansa bo'ladi) bayroqlari bilan. Har tanlovda
  `priceDelta` — taom narxiga qo'shiladi (manfiy ham bo'lishi mumkin).
- Admin: `/admin/menu` formasida "Variantlar" bloki
  (`components/admin/OptionsEditor.tsx`), RU/EN tarjimalari bilan.
- Mijoz: variantli taom kartochkasida "Qo'shish" o'rniga **"Tanlash"** —
  taom sahifasida `OptionPicker` orqali tanlaydi.
- Savatda qator kaliti **taom + tanlovlar** (`lineIdFor`): bir taom turli
  variantlar bilan alohida qator bo'ladi. `localStorage` kaliti `cart_v2`.
- **Narx serverda qayta hisoblanadi** (`resolveOptions`): mijoz faqat qaysi
  tanlovni belgilaganini yuboradi, `priceDelta` doim bazadan olinadi;
  majburiy guruh tanlanmasa yoki tanlov menyuda bo'lmasa — 400.
  Buyurtmaga tanlovlar base (uz) nomi bilan yoziladi.

### Taomga izoh va bekor qilish sababi
- **Har bir savat qatoriga izoh**: mijoz savatda taom ostidagi maydonga
  ("piyozsiz", "achchiq qilmang") yozadi. Izoh `cart_v2` da saqlanadi,
  buyurtmaga `items[].comment` bo'lib boradi (serverda trim + 200 belgi).
  Izoh **qator kimligiga kirmaydi** — bir xil taom+variant baribir bitta qator.
  Admin chekida sariq belgi bilan, kuryer ilovasida taom ostida ko'rinadi.
- **Bekor qilishda sabab majburiy**: admin panelda "Bekor qilish" modal ochadi
  (4 ta tayyor sabab + erkin matn). Holat select'idan `cancelled` tanlansa ham
  shu modal ochiladi — sababsiz bekor qilish yo'li yo'q.
  Sabab `order.cancelReason` da; mijoz uni `/order/{number}` sahifasida va
  profil tarixida ko'radi. Buyurtma qayta tiklansa sabab o'chiriladi.

### Xarita ilovalariga marshrut tugmalari
- `components/map/RouteButtons.tsx` — Yandex Maps / Google Maps / 2GIS uchun
  havolalar; telefonda o'rnatilgan ilovani ochadi, kompyuterda saytni.
  Boshlanish nuqtasi **ataylab bo'sh** — har ilova foydalanuvchi joylashuvini
  o'zi biladi. Koordinata tartibi: Yandex/Google `lat,lng`, 2GIS `lng,lat`.
- **2GIS formati**: `2gis.uz/directions/points/%7C{lng}%2C{lat}`. Eski
  `/routeSearch/rsType/car/to/...` yo'li **o'lgan** — 301 bilan shahar
  sahifasiga tashlaydi (marshrut tuzilmasligi shundan edi). `|` belgisi
  percent-encoded bo'lishi shart.
- Checkout'da "Olib ketish" tanlansa xarita ostida, buyurtma kuzatuvida ham
  (olib ketish buyurtmalari uchun) ko'rsatiladi.

### QR menyu (stol QR kodlari)
- `/admin/qr` — **QR kartochka generatori**: umumiy (restoran) yoki har bir stol
  uchun. Fon (4 ta tayyor ohang), sarlavha, kichik sarlavha, tavsif va pastki
  qator tahrirlanadi; stol kartochkasida **stol raqami katta belgi** bo'lib
  turadi.
- Kartochka **canvas'da chiziladi** (`components/admin/QrPoster.tsx`,
  1200×1700 — chop etishga yetarli). Ya'ni ekranda ko'ringan narsa aynan
  yuklanadi. QR kod oq "plastinka" ustida turadi — qora fonda ham skaner bo'ladi.
- **Yuklab olish**: bitta kartochka → PNG; **"Hammasini ZIP qilib olish"** →
  barcha stollar + umumiy kartochka bitta arxivda. ZIP `lib/zip.ts` da qo'lda
  yoziladi (store-only, kutubxonasiz — PNG allaqachon siqilgan). Sabab: har
  stolga alohida fayl yuborilsa brauzer "bir nechta faylni yuklashga ruxsat
  berasizmi?" deb so'raydi va rad javobida qolganini jimgina tashlab yuboradi.
- **Havola**: stol kartochkasi `"/menu?table=<tableId>"` ga, umumiysi shunchaki
  `/menu` ga olib boradi. Panel boshqa manzilda ochilgan bo'lsa (IP, tunnel),
  "Sayt manzili" maydonida haqiqiy domen yoziladi.
- **Stol konteksti** (`lib/table.tsx`): `?table=` ko'rilganda stol
  `sessionStorage` ga yoziladi (localStorage emas — bu bitta tashrifga tegishli,
  uyga borib yetkazib berish buyurtma qilganda eski stol yopishib qolmasligi
  kerak). Sayt tepasida "N-stoldasiz" chizig'i va "Men stolda emasman" tugmasi.
- **`dinein` buyurtma turi**: checkout'da uchinchi variant ("Stolga") faqat QR
  skaner qilinganda chiqadi va avtomatik tanlanadi; manzil ham, yetkazish
  narxi ham yo'q. Server `tableId` ni xaritadagi stollar bilan solishtiradi —
  noto'g'ri/eskirgan QR bo'lsa 400. Buyurtmaga `tableNumber` nusxa sifatida
  yoziladi (xarita qayta chizilsa ham chek to'g'ri o'qiladi).
- Holat oqimi: `dinein` ham `pickup` kabi **"Yo'lda" bosqichini o'tkazib
  yuboradi**. Statistikada alohida "Stolda (QR)" ko'rsatkichi bor.

### Stol bron qilish (bron tizimi)
- **Xaritani admin chizadi**: `/admin/settings` → "Stol bron qilish" →
  `components/admin/FloorPlanEditor.tsx`. Asbob tanlanadi (to'rtburchak stol /
  dumaloq stol / devor / zona), xaritada sudrab chiziladi; "Ko'chirish"
  rejimida stol sudralib joyi o'zgaradi, stol bosilsa raqami, joylar soni va
  o'lchami tahrirlanadi. Koordinatalar **plan birligida** (`width`×`height`),
  shuning uchun bir chizma telefonda ham, katta ekranda ham to'g'ri ko'rinadi.
- **Stol id'si barqaror**: bron `tableId` ga bog'lanadi, ya'ni stol raqami
  o'zgarsa yoki ko'chirilsa ham eski bronlar buzilmaydi (`tableNumber` chekka
  nusxa sifatida saqlanadi).
- **Mijoz** `/bron` sahifasida sana/vaqt/mehmonlar sonini tanlaydi → xaritada
  bo'sh stollar yashil, bandlari qizil, ishlatilmaydiganlari kulrang. Stol
  bosiladi, ism (+ izoh) yoziladi.
- **Bron uchun SMS tasdiq majburiy** (buyurtmadagidek): kirmagan mijoz
  `/login?next=/bron&reason=booking` ga yuboriladi. Raqam **profildagi
  tasdiqlangan raqamdan** olinadi — formadagi maydon faqat ko'rsatish uchun,
  server uni e'tiborga olmaydi. Sabab: javob bermaydigan raqamga saqlangan
  stol restoranga butun bir kechani yo'qotadi.
  **Panel orqali** (telefon qo'ng'irog'i bilan) qilingan bron bundan mustasno —
  u yerda operator raqamni o'zi yozadi.
- **Bandlikni server hal qiladi** (`internal/handlers/reservations.go`):
  `[at, endsAt)` yarim ochiq oraliq bo'yicha kesishish tekshiriladi va band
  bo'lsa **409** qaytadi — ikki mijoz bir vaqtda bir stolni bosishi mumkin,
  saytdagi rang esa oxirgi so'rov paytidagi holat, xolos. Bekor qilingan bron
  stolni darhol bo'shatadi.
- Boshqa qoidalar ham serverda: `enabled`, `minNoticeMinutes` (juda kech),
  `maxDaysAhead` (juda uzoq), `maxGuests`, stolning joy soni, stol faolligi.
  **Panel orqali qilingan bron** bu ikki vaqt qoidasidan ozod (telefonda "10
  daqiqadan keyin" normal holat).
- Mijoz o'z bronlarini **profilda** ko'radi (`/profile` → "Mening bronlarim"):
  raqam, holat, stol, vaqt, mehmonlar soni, izoh va bekor qilingan bo'lsa —
  restoranning sababi.
- **Panelda** `/admin/reservations`: kelayotgan / bugun / o'tgan / hammasi,
  qidiruv, holat tugmalari (Tasdiqlash → Mehmon keldi → Yakunlash, yoki
  sababi bilan Bekor qilish) va o'ng tomonda **tanlangan payt uchun xarita** —
  "soat 8 da 7-stol bo'shmi?" degan savolga ro'yxat javob bera olmaydi.

### Yangi buyurtma/bron — ovozli bildirishnoma
- `components/admin/AlertBell.tsx`: **`AlertBell`** (kuzatuvchi) admin
  layout'da **bitta marta** ulanadi — har 15 soniyada `GET /admin/alerts`
  so'raladi va oxirgi ko'rilgan vaqt bilan solishtiriladi; yangisi kelsa ovoz
  chalinadi va pastki o'ng burchakda banner chiqadi. **`SoundToggle`** esa
  faqat tugma va u ikki joyda turadi (yon panel + telefondagi yuqori panel).
  Ular `admin-sound-change` hodisasi orqali gaplashadi.
  **Muhim**: kuzatuvchi ikki marta ulansa har xabar ikki marta chalinadi —
  shu sabab tugma va kuzatuvchi ajratilgan.
- Ovoz **WebAudio bilan sintez qilinadi** — alohida audio fayl yo'q, ya'ni
  mijoz VPS'ida 404 bo'lishi mumkin emas. **Buyurtma — ikki nota (880/1175),
  bron — uch nota (660/880/1320)**: zaldagi odam ekranga qaramay farqlaydi.
- Ovoz **standart holatda yoqilgan**. Brauzer sahifa bilan muloqotdan oldin
  ovoz bermaydi, shuning uchun paneldagi **birinchi bosish** (istalgan joyda)
  audio'ni jimgina ochadi. 🔕 tugmasi faqat o'chirish uchun; tanlov
  localStorage'da saqlanadi.

### Admin ro'yxatlari: scroll bloki + pagination
- `components/admin/PagedList.tsx` — `usePaged()` (sahifalash), `<ListScroll>`
  (o'z ichida suriladigan blok, `max-h`) va `<Pager>` ("13–15 / 15", Oldingi /
  Keyingi). Har bir ro'yxat sahifa balandligini oshirmasligi kerak: ma'lumot
  ko'paygani sari sahifa cho'zilib, filtrlar ekrandan chiqib ketardi.
- Qo'llanilgan joylar: buyurtmalar (12/sahifa), foydalanuvchilar (25),
  kuryerlar (20), mijoz kartochkasidagi buyurtmalar va manzillar (10),
  kuryer kartochkasidagi tarix (10), adminlar, amallar jurnali (server
  tomonidan "ko'proq yuklash" bilan), menyu (har kategoriya bloki),
  kategoriyalar, tashqi xizmatlar.
- Jadval sarlavhalari `sticky top-0` — suriganda ustun nomlari ko'rinib turadi.

### Dashboard statistikasi (`GET /admin/stats`)
- `internal/handlers/adminstats.go` — davr `?from=&to=` (`YYYY-MM-DD`, `to`
  kiritilgan kun bilan; ikkisi ham bo'sh = butun tarix). Hisob **bazada**
  bajariladi: avval panel oxirgi 200 buyurtmani olib brauzerda qo'shardi, ya'ni
  200 dan oshgach ko'rsatkichlar noto'g'ri bo'lardi.
- Qaytadi: davr bo'yicha buyurtmalar (jami, yetkazilgan, bekor, yetkazish/olib
  ketish, holatlar kesimi), pul (tushum, o'rtacha buyurtma, yetkazish yig'imi,
  naqd), top 8 taom, hamda mijozlar (jami / yangi / **faol** = shu davrda
  buyurtma bergan / manzili bor), kuryerlar (jami, faol, ishda), adminlar
  (owner/manager) va menyu (taomlar, mavjud, kategoriyalar) sanoqlari.
- **Bekor qilingan buyurtma pulga qo'shilmaydi** (kassaga tushmaydi), lekin
  buyurtmalar soniga kiradi — sahifada shu izoh yozilgan.

### Buyurtma manzilini xaritada tuzatish (admin)
- Chekdagi **"Xaritadagi joy"** bo'limi (`components/admin/OrderAddressMap.tsx`):
  mijoz belgilagan nuqta xaritada ko'rinadi, zonalar ham chiziladi.
  `/admin/orders` da nuqtani **bosib ko'chirish** mumkin (mijoz adashib
  belgilagan bo'lsa) — matn `reverseGeocode` bilan avtomatik yangilanadi;
  `/admin/users/[id]` va kuryer kartochkasida faqat ko'rinadi (`readOnly`).
- Nuqta bezak emas: kuryer "Yetkazdim"ni faqat shu nuqtaga `arrivalRadiusM`
  metr yaqin turib bosa oladi — noto'g'ri metka buyurtmani yopishga to'sqinlik
  qiladi. Shu sababli tuzatish imkoniyati kerak.
- **Pul o'zgarmaydi**: `deliveryFee`/`total` mijoz bilan kelishilgani uchun
  qo'lda tuzatilgan nuqta narxni qayta hisoblab yubormaydi. Zona va masofa
  (`deliveryZone`, `distanceKm`) yangilanadi, va yangi nuqtaning **hozirgi
  narxi** javobda qaytadi — boshqa zonaga tushib qolsa panel ogohlantiradi.
- Har tuzatish jurnalga tushadi: `order.address`.

### Buyurtmalar oqimi (admin)
- Holat **qo'lda** o'zgaradi, lekin bir bosishda: har bir buyurtmada
  "keyingi qadam" tugmasi bor (Tasdiqlash → Tayyorlashni boshlash → Yo'lga
  chiqdi → Yetkazildi). Olib ketish buyurtmalari `on_the_way` ni o'tkazib
  yuboradi. Yonidagi select — faqat tuzatish uchun (orqaga qaytarish).
- Ro'yxat har 20 soniyada avtomatik yangilanadi, yangi buyurtmalar soni
  ko'rsatiladi; "Faol" filtri tugallanmagan buyurtmalarni ko'rsatadi.
- `statusHistory` har bir o'zgarish vaqtini yozadi → chekdagi "Vaqt jadvali"
  (shikoyat kelganda "qachon tasdiqlangan/yetkazilgan" savoliga javob).
- Kelajakda avtomatlashtirish mumkin (masalan N daqiqadan keyin avtomatik
  `confirmed`), lekin restoran real holatni o'zi bilgani uchun hozircha
  qo'lda — faqat bosish soni minimallashtirilgan.

### Mijoz auth (telefon + SMS)
- Asosiy usul — **telefon raqam + bir martalik SMS kod** (`internal/sms`).
  Provayder `SMS_PROVIDER` bilan tanlanadi: `demo` (default — SMS ketmaydi,
  kod API javobida qaytadi), `eskiz` (notify.eskiz.uz), `playmobile`.
- Kodlar `phone_code` kolleksiyasida **bcrypt hash** ko'rinishida, 3 daqiqa
  amal qiladi, 60 soniya cooldown, 5 ta noto'g'ri urinishdan keyin bekor.
- **Buyurtma berish uchun login majburiy**: savatdagi tugma login sahifasiga
  yuboradi (`/login?next=/checkout&reason=order`), checkout ham himoyalangan.
- Profilda mijoz ismi va manzillarini tahrirlaydi; raqamni o'zgartirish
  faqat yangi raqamga kelgan SMS kod bilan tasdiqlanadi.

### Admin parolini tiklash
- Kirish sahifasida "Parolni unutdingizmi?" — hisobga biriktirilgan raqamga
  SMS kod (`/admin/password/forgot` → `/admin/password/reset`, ikkalasi ham
  public: kira olmayotgan odam uchun boshqa yo'l yo'q).
- **Tiklash raqami** panelda qo'yiladi (Hisobim → `RecoveryPhone`) va SMS bilan
  tasdiqlanadi. Seed qilingan birinchi owner'da raqam yo'q — mijozga
  topshirishdan oldin qo'shish shart (DEPLOY.md).
- **Kodlar maqsad bo'yicha ajratilgan** (`phone_code.purpose`:
  `login` / `admin_reset` / `admin_phone`). Sabab: restoran egasi ko'pincha
  saytning ham mijozi, bir xil raqam bilan — mijozning login kodi admin
  panelini ocha olmasligi kerak. `(phone, purpose)` unique, `expiresAt` TTL.
- Zaxira yo'l — serverda `cmd/adminreset` (raqam yo'q yoki telefon yo'qolgan).

### Panel adminlari va amallar jurnali
- **Yangi admin — mavjud sayt mijozi**: odam avval saytda telefon + SMS bilan
  kirgan bo'lishi kerak; owner `/admin/admins` da uni qidirib topadi va login +
  vaqtinchalik parol beradi. `admin_user.userId` shu mijozga bog'lanadi
  (bitta mijoz — bitta panel hisobi).
- Yaratilgan hisob `mustChangePassword: true` bo'ladi: birinchi kirishda
  `AdminLayout` uni `/admin/account` ga majburan yuboradi. Parolni tiklash ham
  shu bayroqni qayta qo'yadi (boshqa odam yozgan parol vaqtinchalik).
- **Faqat `owner`**: hisoblarni boshqarish va jurnalni ko'rish
  (`RequireRole(..., "owner")`). Menejer o'ziga huquq qo'sha olmaydi va
  o'zidan keyingi izni o'chira olmaydi. O'z hisobini va oxirgi ownerni
  o'chirish taqiqlangan.
- **Amallar jurnali** (`admin_log`, `handlers/auditlog.go`): har bir o'zgarish
  `h.logAction(...)` bilan yoziladi — kirish, buyurtma holati/bekor qilinishi
  (sababi bilan), kuryer biriktirish, tashqi xizmat chaqirish, kuryer/admin/
  taom/kategoriya/xizmat CRUD, sozlamalar. `action` — barqaror id
  (`order.cancel`), matn panelda tarjima qilinadi. Yozuvlar tahrirlanmaydi va
  o'chirilmaydi; log yozilmasa ham amal bekor qilinmaydi.

### Ishchilar davomati (`/staff` + `/admin/staff` + `/admin/payroll`)
- **Ikki kirish, bir chiqish**: hamma narsa ikkita manbadan hisoblanadi —
  admin yozgan **ish grafigi** va ishchi bosgan **smenalar**. Kunning holati
  (`kam ishlangan` / `grafik bo'yicha` / `ko'p ishlangan` / `chiqmagan`)
  hech qayerda saqlanmaydi, u **taqqoslash** (`handlers/staffattendance.go`).
- **Geolokatsiya majburiy va serverda tekshiriladi** (`geofenceBlocked`).
  Ilova tugmani oldindan o'chiradi, lekin qoida — serverda. Har bosish
  koordinatasi va masofasi bilan saqlanadi (`shift.inAt/outAt`).
- **Radius 50 m (5 emas)**: telefon GPS'i ochiq havoda 10–30 m, bino ichida
  yomonroq. Xato chegarasidan kichik radius firibgarni ushlamaydi, halol
  ishchini eshik oldida qoldiradi. `branch.staffRadiusM` sozlanadi; **0 —
  tekshiruv o'chiq**, shuning uchun eski filiallarga `EnsureStaffDefaults`
  50 yozadi va migratsiya yaratadigan filialga qiymat **qo'lda** beriladi
  (Go nol qiymatni tushirib qoldirmaydi → `$exists:false` uni ko'rmaydi).
- **Smena boshlangan kunga yoziladi** (`shift.date`, local `YYYY-MM-DD`):
  00:40 da chiqqan oshpaz seshanbani yopadi. Serverda **`TZ=Asia/Tashkent`**
  bo'lishi shart, aks holda butun kalendar UTC'da chiziladi.
- **Kun holati punchlarga qarab**, daqiqalarga emas: kirib darhol chiqqan odam
  0 daqiqa ishlagan, lekin "chiqmagan" emas. Bo'sh grafik ham hech kimni
  ayblamaydi — to'ldirilmagan kun dam olish deb o'qiladi.
- **Ikki marta kirish 409** (birlashtirilmaydi): ikkita ochiq smena kalendardagi
  har soatni ikkilantirardi.
- **Oylik maosh oyning grafik kunlariga bo'linadi** va chiqilgan kun uchun
  beriladi — o'rtada ishga kirgan yoki bir hafta kelmagan odam ham to'g'ri
  chiqadi va kalendar bilan kassa bir xil gapiradi.
- **To'lov — yozuv, hisoblagich emas** (`staff_payment`): qaysi davrni yopgani
  va kim bergani bilan. Aprel boshlangandan keyin ham "martda qancha to'ladik"
  javobsiz qolmaydi.
- **Qo'lda tuzatish ataylab qoldirilgan**: telefon o'chadi, chiqish unutiladi —
  tuzatib bo'lmaydigan davomat tizimidan ikkinchi haftada voz kechiladi. Har
  tuzatish `editedBy` bilan imzolanadi va jurnalga tushadi.
- Ishchi hisobi o'chirilsa **smenalar qoladi**.
- Sana yorliqlari `01.08` ko'rinishida va oy nomlari lug'atdan: `uz-UZ`
  locale'da "long" oy **"M08"** bo'lib chiqadi.
- Kalendar bitta komponent (`components/staff/AttendanceCalendar.tsx`),
  ranglar bitta joyda (`lib/attendance.ts`) — panel va ishchi ilovasi bir xil
  kunni har xil rangda ko'rsata olmaydi.

### Kuryerlar va rollar
- **Rollar**: `owner`/`manager` (admin), `user` (mijoz), `courier`, `staff`.
  `middleware.RequireRole` har guruhda rolni tekshiradi — ilgari faqat
  `RequireAuth` bo'lgani uchun mijoz tokeni bilan admin API'ga kirish mumkin edi.
- Kuryer hisobini **admin qo'lda yaratadi** (`/admin/couriers`), o'zi
  ro'yxatdan o'ta olmaydi. Parol bcrypt bilan saqlanadi.
- Status: `off` (ishda emas) / `free` (bo'sh) / `busy` (band). Buyurtma
  biriktirilganda avtomatik `busy`, yetkazilgach — faol buyurtmasi qolmasa —
  `free` bo'ladi (`syncCourierBusy`).
- Kuryer faqat **o'z** buyurtmasini `on_the_way` va `delivered` ga o'tkaza
  oladi; oldingi bosqichlar oshxonaniki.
- **"Yetkazdim" faqat manzilda bosiladi**: kuryerning oxirgi joylashuvi mijoz
  manzilidan `delivery.arrivalRadiusM` metrdan uzoq bo'lsa yoki 10 daqiqadan
  eski bo'lsa — server 400 qaytaradi (`arrivalBlocked`). Ilova tugmani
  o'chirib qo'yadi, lekin qoida serverda. Olib ketish buyurtmalari va
  koordinatasiz manzillar tekshirilmaydi; `arrivalRadiusM = 0` — o'chiq.
  **Zaxira yo'l**: admin panelda holatni baribir qo'lda yopish mumkin (GPS
  ishlamay qolgan holat uchun ataylab qoldirilgan).
- **Naqd pul topshiruvi** (`courier_settlement`): "qo'lidagi naqd" =
  yig'ilgan − topshirilgan. Ilgari faqat yig'ilgan ko'rsatilardi va u hech
  qachon kamaymasdi. Topshirish **ledger yozuvi** sifatida saqlanadi
  (hisoblagichni nolga tushirish emas) va kim qabul qilgani yoziladi —
  nomsiz naqd topshiruv keyin bahsga aylanadi.
- **Daromad** (`courierstats.go`): har kuryerning `payoutMode` i bo'yicha
  hisoblanadi — `deliveryFee` (default, mijoz to'lagan yetkazish narxi),
  `perOrder` (belgilangan summa) yoki `percent` (yetkazish narxidan foiz).
  Olib ketish buyurtmasi hech qachon daromad bermaydi. Davrlar: bugun,
  oxirgi 7 kun, oxirgi 30 kun, jami — `statusHistory` dagi `delivered`
  vaqtiga qarab. "Naqd yig'ilgan" alohida ko'rsatiladi (kuryer qo'lidagi pul).
- Mijoz `/order/{number}` da kuryerni **faqat `on_the_way` bosqichida**
  xaritada ko'radi; boshqa holatlarda joylashuv umuman qaytarilmaydi.

### Tashqi yetkazish xizmatlari (o'z kuryeri yo'q restoranlar uchun)
- `/admin/settings` → "Tashqi yetkazish xizmatlari": Yandex Delivery, taksi,
  eshikdan-eshikka firmalari qo'shiladi/o'chiriladi. Har biri `link` (havola)
  yoki `phone` (dispetcher raqami).
- **Asosiy yo'l — web** (biznes-akkaunt kerak emas). Buyurtmalar bo'limida
  "Yetkazishni chaqirish" → modal 3 qadam:
  1. **Xizmatga o'tish** — havola o'rinbosarlar bilan to'ldiriladi va
     "Ochish va chaqirildi deb belgilash" bitta bosishda ochadi + yozadi.
     Tayyor namuna: **Yandex Go Dostavka deeplink** —
     `yandex.go.link/route?tariffClass=courier&adj_t=pucm71r&trap_mode=true`
     + `start-lat/start-lon` (restoran) va `end-lat/end-lon` (mijoz).
     Bu Yandex o'zining dostavka saytida ishlatadigan havola: telefonda
     `yandextaxi://route?...` ga aylanadi va ilova **Dostavka** bo'limida,
     ikkala manzil bilan ochiladi. `adj_t` (Adjust tokeni) **shart** — bo'lmasa
     havola 404. Taxi web sahifasida dostavka bo'limi yo'q, shuning uchun eski
     `3.redirect.appmetrica.yandex.com/route` havolasi ishlamadi.
     **`tariffClass` ilovada qaysi bo'lim ochilishini belgilaydi** va tarif
     nomlari mamlakat bo'yicha farq qiladi: Rossiyaning `express_d2d` tarifi
     O'zbekistonda sotilmaydi — bunda ilova jim turib **taxi** buyurtmasiga
     qaytadi (aynan shu xato bo'lgan edi). **O'zbekistonda ishlaydigan tarif —
     `express`** (avtomobil kuryer; telefonda tekshirilgan, Dostavka bo'limi
     ochiladi). Boshqa tariflar namunalardan olib tashlangan.
     **Deeplink faqat koordinata + tarif oladi**: mijozning ismi, telefoni,
     manzil matni va izohi havolada uzatilmaydi (Yandex hujjatlaridagi
     parametrlar — `start-lat/lon`, `end-lat/lon`, `tariffClass`, `ref`,
     `lang`, AppMetrica id'lari). Shuning uchun chaqirish oynasida har bir
     maydon alohida nusxalanadi; to'liq avtomatik to'ldirish faqat B2B API
     (`kind: "api"`) orqali.
     **Muhim**: O'zbekistonda Yandex Delivery jismoniy shaxsga faqat **Yandex Go
     ilovasi** orqali; `delivery.yandex.uz` — yuridik shaxslar uchun (API yo'li).
     Shuning uchun kompyuterdagi administrator uchun chaqirish oynasida
     **QR kod** chiziladi (`components/admin/QrCode.tsx`, `qrcode-generator`) —
     telefon bilan skaner qilinadi. Deeplink faqat koordinata + tarif oladi
     (ism/telefon nusxalash orqali).
  2. **Ma'lumotlar** — har bir maydon alohida "Nusxalash" tugmasi bilan
     (havolada parametr qabul qilmaydigan formalar uchun) + "Hammasini nusxalash".
  3. **Xizmat javobi** — xizmatdagi raqam, xizmat narxi (`cost`), izoh.
- O'rinbosarlar (`lib/providerLink.ts`): `{number} {name} {phone} {address}
  {comment} {lat} {lng} {total} {subtotal} {deliveryFee} {items} {itemsCount}
  {payment}` va restoran tomoni `{pickupName} {pickupAddress} {pickupPhone}
  {pickupLat} {pickupLng}`. Sozlamalarda bosiladigan chipslar orqali qo'yiladi.
- Yozuvni keyin tahrirlash `calledAt` ni o'zgartirmaydi (topshirilgan vaqt).
- Tashqi xizmat chaqirilsa **o'z kuryeri bo'shatiladi** (bitta buyurtmani
  ikkovi olib ketmasin).
- **`kind: "api"` — tanlov (biznes-akkaunt bo'lsa)** (`internal/delivery/yandex.go`):
  Yandex Delivery B2B cargo API. Server o'zi zayavka yaratadi (`claims/create`),
  tasdiqlaydi (`claims/accept`), holatini o'qiydi (`claims/info`) va bekor
  qiladi (`claims/cancel`). Token restoranning shaxsiy kabinetidan olinadi,
  admin panelda kiritiladi va **hech qachon brauzerga qaytarilmaydi**
  (`hasToken` bayrog'i bilan ko'rsatiladi; bo'sh yuborilsa saqlangani qoladi).
- `request_id = "order-<orderId>"` — **idempotent**: tarmoq uzilib qayta
  chaqirilsa ikkinchi kuryer chaqirilmaydi.
- `apiBaseUrl` sozlanadi — sandbox yoki test mock'iga yo'naltirish uchun.
- Millennium taxi va boshqa mahalliy xizmatlarda ommaviy API yo'q — ular
  `phone`/`link` sifatida ishlaydi. API paydo bo'lsa `clientFor` ga yangi
  `apiProvider` qo'shiladi, qolgani o'zgarmaydi.

### SEO va favicon
- `app/layout.tsx` dagi `generateMetadata` restoran profilidan quriladi:
  sarlavha shabloni `%s | <restoran nomi>`, tavsif, **favicon = yuklangan
  logotip** (`app/icon.svg` — zaxira), Open Graph/Twitter uchun cover rasm.
- Har bo'lim o'z sarlavhasini beradi. Client komponent bo'lgan sahifalar
  (savat, checkout, login, profil, buyurtma kuzatuvi) uchun sarlavha
  yonidagi `layout.tsx` da — shu yerda ular `robots: noindex` ham oladi
  (shaxsiy sahifalar qidiruvga tushmasligi kerak).
- Taom sahifasi `generateMetadata` da nom/tavsif/rasmni tanlangan tilda beradi.

### Joylashuvga ruxsat (ishchi va kuryer ilovalari)
- **Brauzer bir marta so'raydi.** Rad etilgandan keyin `getCurrentPosition`
  darhol xato qaytaradi, dialog esa boshqa chiqmaydi — sahifa uni qaytara
  olmaydi, faqat odamning o'zi brauzer sozlamalaridan yoqadi. Shuning uchun
  "ruxsat berilmagan" yozuvining o'zi foydasiz: u rost, lekin nima qilishni
  aytmaydi.
- `lib/geo.tsx` — `useGeoPermission()`: holatni **Permissions API** dan o'qiydi
  (`prompt` / `granted` / `denied`), o'zgarishini kuzatadi (sozlamalardan
  yoqilsa sahifani yangilash shart emas) va `request()` beradi.
  Safari'da geolokatsiya uchun Permissions API yo'q → holat `unknown`, bu
  "so'rab ko'rish mumkin" degani.
- `components/GeoPermission.tsx` — uch holat, uch ko'rinish: **so'ralmagan** →
  tugma (xato emas), **rad etilgan** → qurilmaga qarab (iOS / Android /
  kompyuter) qadam-baqadam ko'rsatma, **berilgan** → chaqiruvchining o'z
  holat qatori.
- **Ruxsat bosish orqali so'raladi**: iOS Safari faqat foydalanuvchi
  harakatidan keyin dialog ko'rsatadi.
- **`watchPosition` faqat `granted` bo'lgandan keyin** boshlanadi: `prompt`
  holatida u ekran hech narsa tushuntirmasdan turib dialog chiqarardi,
  `denied` da esa faqat xato callback'ini chaqirardi.
- **HTTPS majburiy**: `window.isSecureContext` false bo'lsa (masalan telefondan
  `http://192.168.x.x` orqali ochilgan) brauzer joylashuvni umuman bermaydi —
  bu ruxsat muammosi emas va alohida xabar bilan ajratilgan.
- `PERMISSION_DENIED` dan boshqa xatolar (timeout, GPS ushlamadi) **ruxsat
  muammosi emas** — bunday odamni brauzer sozlamalariga yuborish foydasiz.

### Kuryer PWA va joylashuv (muhim cheklov)
- `/kuryer` — alohida PWA: o'z `manifest.webmanifest` (scope `/kuryer`) va
  `public/courier-sw.js` service worker'i bor, telefonga ilova sifatida
  o'rnatiladi.
- Joylashuv `navigator.geolocation.watchPosition` bilan **sahifa ishlab
  turganda** yig'iladi, 15 soniyada bir marta batch qilib yuboriladi,
  oflaynda `localStorage` da buferlanadi.
- Smena davomida **Wake Lock** olinadi (ekran o'chmaydi) va ilova fon rejimiga
  o'tganda oxirgi nuqta darhol yuboriladi.
- **Ilova butunlay yopilganda joylashuv uzatilmaydi** — bu brauzer cheklovi,
  kod kamchiligi emas: service worker'ga geolocation berilmaydi, yopilgan PWA
  esa umuman ishlamaydi. Haqiqiy fon kuzatuvi uchun native o'ram kerak
  (Capacitor yoki TWA + foreground service). Shu sababli kuryer ekranida
  "ilovani yopmang" ogohlantirishi bor.

### Maintenance buyruqlari (`backend/cmd/`)
- `cmd/server` — API serveri.
- `cmd/seedmenu` — namuna menyuni mavjud bazaga yozish (`-db`, `-replace`, `-y`).
- `cmd/adminreset` — **admin parolini tiklash** (`-list`, `-username`,
  `-password`, `-create`, `-force-change`). Admin panelda "parolni unutdim"
  oqimi yo'q — tiklash serverda shu buyruq orqali (DEPLOY.md ga qarang).
  Docker image'da barcha `cmd/*` binarlari bor: `/app/adminreset`, `/app/seedmenu`.

### Namuna menyu (seed)
- `backend/internal/seed/menu.go` — 7 kategoriya, 48 taom (rasmlari bilan).
  Rasmlar `internal/seed/assets/*.jpg` da, Go `embed` orqali binarda; birinchi
  ishga tushishda `UPLOAD_DIR/seed/` ga yoziladi.
- Seed faqat **bo'sh bazada** ishlaydi — mavjud menyu hech qachon o'zgarmaydi.
  Yangi mijozga deploy qilganda menyu shu namunadan boshlanadi va admin
  panelda tahrirlanadi.

---

## 11. Bosqichlar (Roadmap)

- [ ] **M0** — Skeleton: papka strukturasi, docker-compose, backend/frontend ishga tushadi.
- [ ] **M1** — Backend: modellar, DB ulanish, public menu API + restaurant API.
- [ ] **M2** — Admin auth (JWT) + menu/category CRUD + upload.
- [ ] **M3** — Frontend public: bosh sahifa, menyu, savat.
- [ ] **M4** — Checkout + orders + delivery quote (Yandex).
- [ ] **M5** — Admin panel UI (menu, orders, settings).
- [x] **M6** — Polish, seed data (48 taom), tema + i18n, deploy hujjati
      (`DEPLOY.md`, `nginx/restaurant.conf`, `docker-compose.prod.yml`).
