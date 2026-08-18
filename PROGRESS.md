# PROGRESS.md — Ish jurnali

Bu faylda loyiha ustidagi ish jarayoni yozib boriladi. Har bir sessiya /
bosqichdan keyin yangilanadi. Loyihaning to'liq tavsifi — `CLAUDE.md`.

Format: sana → nima qilindi → keyingi qadam.

---

## 2026-07-16 — M0 + M1 + M2 (backend) bajarildi ✅

### Hujjatlar
- `CLAUDE.md` yozildi — loyiha to'liq tavsifi, arxitektura, ma'lumot modeli,
  API endpoint'lar, roadmap.
- `PROGRESS.md` (shu fayl) yaratildi.
- Papka strukturasi: monolit `frontend/` + `backend/`, `docker-compose.yml`.

### Backend (Go + MongoDB) — TO'LIQ ISHLAYDI, test qilindi ✅
Struktura: `backend/cmd/server` + `backend/internal/{config,db,models,repository,handlers,middleware,router,auth,httpx,seed}`

- `go build ./...` — **toza kompilyatsiya** (EXIT 0), `go mod tidy` bajarildi.
- **Modellar** (`internal/models/models.go`): Restaurant (singleton, workingHours,
  delivery zonalar/radius), Category, MenuItem (options bilan), Order, AdminUser.
- **Config** (`.env`), **Mongo ulanish** (`db/mongo.go`), **Store/repository**.
- **Seed** (`seed/seed.go`): birinchi ishga tushganda admin user + default
  restoran yaratadi (ADMIN_USERNAME/PASSWORD `.env` dan).
- **Auth**: JWT (`auth/jwt.go`) + bcrypt parol + `RequireAuth` middleware.
- **Public API**: `GET /restaurant` (isOpenNow hisoblanadi), `/categories`,
  `/menu` (kategoriya bo'yicha guruhlangan), `/menu/{id}`, `POST /orders`
  (narxlar serverda qayta hisoblanadi — tampering himoyasi), `/orders/{number}`
  (track), `POST /delivery/quote`.
- **Admin API** (JWT): `login`, `me`, `PUT /restaurant`, category CRUD,
  menu CRUD, `POST /upload` (rasm → `uploads/`), orders ro'yxati + status.
- **Delivery logikasi** (`handlers/util.go`): point-in-polygon (zonalar) +
  Haversine (radius modeli) + minOrder + freeDeliveryFrom.
- **Static**: `/uploads/*` fayl serve, `/health`.

**Smoke-test natijalari (hammasi ✅):**
- `/health` → ok; `/api/v1/restaurant` → seed data + isOpenNow.
- Admin login → JWT (192 belgi).
- Category create → menu item create → public `/menu` guruhlangan ko'rsatdi.
- Auth guard: token'siz `POST /admin/categories` → **401**.
- Delivery quote: yaqin (1.3km) → 16000 so'm; uzoq (44km, maxKm=15) → mavjud emas.

### Docker
- `docker-compose.yml` (mongo + backend + frontend + volumes) yozildi.
- `backend/Dockerfile` (multi-stage) yozildi.
- **Eslatma**: lokal mashinada MongoDB allaqachon `127.0.0.1:27017` da ishlayapti,
  shuning uchun dev test o'sha lokal mongo'ga ulanib qilindi (`MONGO_DB=restaurant_test`).
  Compose'dagi mongo porti to'qnashadi — dev'da lokal mongo ishlatilsa bo'ladi.

### Frontend (Next.js + TS + Tailwind) — BOSHLANDI, TUGALLANMAGAN ⏳
- `frontend/` papka: `src/{app,components,lib}` + `public/`.
- Yozilgan: `package.json` (Next 15, React 19, Tailwind 3), `tsconfig.json`.
- **QILINMAGAN** (bu yerda to'xtadik): tailwind config, postcss, `next.config`,
  `app/layout.tsx`, `globals.css`, API client (`lib/api.ts`), sahifalar
  (home, menu, cart, checkout, admin), `npm install`, ishga tushirish testi.

---

## 2026-07-17 — Frontend skeleton (M0 davomi) TUGALLANDI ✅

Kechagi to'xtagan joydan davom etildi — frontend skeleton to'liq yakunlandi,
build va dev server test qilindi.

### Yozilgan / sozlangan
- **Konfiguratsiya**: `next.config.ts` (standalone output, image remotePatterns),
  `postcss.config.js`, `tailwind.config.ts` (brand rang palitrasi), `next-env.d.ts`,
  `.env.local.example` (API_URL, UPLOADS_URL, 2GIS MAP_API_KEY), `.gitignore`,
  `.dockerignore`, `.eslintrc.json`.
- **App**: `src/app/layout.tsx` (lang="uz"), `src/app/globals.css` (Tailwind +
  `.btn`, `.btn-primary`, `.btn-ghost`, `.container-page` komponentlari),
  `src/app/page.tsx` (bosh sahifa skeleton — backend'dan `getRestaurant`,
  isOpenNow, xatoda ham ochiladi try/catch bilan).
- **API layer**: `src/lib/types.ts` — backend Go modellariga to'liq mos TS turlar
  (Restaurant, Category, MenuItem, Order, DeliveryQuote, LoginResponse, ...).
  `src/lib/api.ts` — fetch wrapper (`ApiError`, JWT localStorage token
  get/set/clear, `imageUrl()` helper, public + admin endpointlar, SSR uchun
  `revalidate`/`cache` hint).
- **Dependencies**: `package.json`'ga qo'shildi: @tanstack/react-query,
  react-hook-form, zod, eslint, eslint-config-next.
- **Docker**: `frontend/Dockerfile` (multi-stage, Next standalone output).

### Xavfsizlik / muhim
- **next 15.1.6 → 15.5.20** ga ko'tarildi (CVE-2025-66478 kritik zaiflik).
  Qolgan 2 ta moderate audit — next'ning ichki postcss'i, breaking changesiz
  tuzatib bo'lmaydi, e'tiborsiz.
- `react/no-unescaped-entities` ESLint qoidasi **off** qilindi — o'zbekcha
  matnda apostrof (') ko'p ishlatiladi.

### Test natijalari (✅)
- `npm install` — 368 paket, toza.
- `npm run build` — **muvaffaqiyatli kompilyatsiya**, `/` statik prerender.
- `npm run dev` → `curl http://localhost:3000/` → **HTTP 200** (backend
  o'chiq bo'lsa ham sahifa ochiladi).

---

## 2026-07-17 (2) — M3 Public UI TUGALLANDI ✅

Public sayt UI to'liq yozildi va real backend ma'lumoti bilan test qilindi.

### Struktura
- Route group **`src/app/(site)/`** — umumiy layout (CartProvider + Header +
  Footer). Restoran ma'lumoti layout'da server-side olinadi.
- Sahifalar: `/` (home), `/menu`, `/menu/[id]`, `/cart`, `/about`.

### Yozilgan fayllar
- **State**: `src/lib/cart.tsx` — CartProvider (useReducer + localStorage
  `cart_v1`, hydrate/persist, add/setQty/remove/clear, count + subtotal).
- **Utils**: `src/lib/format.ts` — `formatPrice` (uz-UZ, "so'm"), `weekdayName`.
- **Komponentlar**: `components/site/{Header,Footer}.tsx` (Header client —
  savat sonini ko'rsatadi), `components/menu/MenuItemCard.tsx` (client, add),
  `components/menu/AddToCartControl.tsx` (client, miqdor tanlash).
- **Sahifalar**:
  - `/` — hero (cover, isOpenNow), mashhur taomlar grid, ish vaqti jadvali.
  - `/menu` — yon kategoriya navigatsiyasi + kategoriya bo'yicha bo'limlar.
  - `/menu/[id]` — taom sahifasi (async params, 404 → notFound()).
  - `/cart` — qatorlar, miqdor +/−, o'chirish, xulosa, "Buyurtma berish"
    (→ `/checkout`, M4'da yoziladi).
  - `/about` — aloqa, ijtimoiy tarmoqlar, ish vaqti (xarita M4'da).

### Test natijalari (✅ real backend + Mongo bilan)
- Backend qayta ishga tushirildi (fresh seed: admin + restoran).
- Admin API orqali seed qilindi: 1 kategoriya (Milliy taomlar) + 3 taom
  (Osh, Manti [oldPrice bilan], Lag'mon; Osh/Manti = popular).
- `npm run build` — **toza** (7 route, `/menu/[id]` dynamic).
- Dev test: `/`, `/menu`, `/cart`, `/about` → **HTTP 200**, real ma'lumot
  render bo'ldi (home'da mashhur taomlar, menu'da 3 taom).
- `/menu/{id}` → 200 (Osh sahifasi, tavsif+narx); noto'g'ri id → **404**.

### Eslatma
- Rasmlar hozircha yo'q (`imageUrl` bo'sh) → "Rasm yo'q" placeholder ishlaydi.
- Cart option price-delta hozircha display'ga qo'shilmaydi; backend baza
  narxidan qayta hisoblaydi. To'liq options M4 checkout'da.

---

## 2026-07-17 (3) — M4 Checkout + Xarita TUGALLANDI ✅

Checkout oqimi, 2GIS xarita moduli va buyurtma kuzatish yozildi, real backend
bilan test qilindi.

### Paketlar
- `@2gis/mapgl` (MapGL kutubxonasi), `@hookform/resolvers` (RHF + Zod).
- `.env.local` yaratildi (`.env.local.example` dan). **MAP_API_KEY hozircha
  bo'sh** — komponent buni chiroyli boshqaradi (placeholder ko'rsatadi).

### Yozilgan fayllar
- **Xarita moduli** (almashtiriladigan): `src/components/map/AddressMap.tsx` —
  2GIS MapGL, click bilan marker qo'yish → `{lat,lng}`. Props kontrakti
  `{ value, onChange, center }` — provayderni almashtirish = faqat shu faylni
  almashtirish. Key yo'q bo'lsa placeholder. 2GIS `[lng,lat]` tartibi.
  Geocoding ISHLATILMAYDI (pullik) — matn manzil qo'lda kiritiladi.
- **Checkout** `src/app/(site)/checkout/page.tsx` (client, RHF + Zod):
  aloqa (ism/telefon), delivery/pickup toggle, xarita + manzil matni + izoh,
  to'lov turi (naqd/karta), buyurtma xulosasi. `delivery/quote` **debounce
  300ms** bilan (nuqta yoki subtotal o'zgarganda). minOrder tekshiruvi,
  `available=false` va bo'sh savat holatlari. Submit → `POST /orders` →
  `/order/{number}` ga redirect + savatni tozalaydi.
- **Track** `src/app/(site)/order/[number]/page.tsx` (client): status stepper
  (pending→confirmed→preparing→on_the_way→delivered), cancelled alohida,
  **20s avtomatik poll** + qo'lda "Yangilash" (realtime kuryer yo'q — CLAUDE §7).

### Test natijalari (✅ real backend + Mongo)
- Seed restoran delivery: minOrder 50000, baseFee 10000, perKm 3000, maxKm 15.
- `delivery/quote` (3.27km) → **22000 so'm** (10000 + 3000×~4km).
- `npm run build` — **toza** (9 route; checkout/order dynamic).
- `POST /orders` (Osh×2 = 70000, delivery) → number **XV55-5653**,
  server deliveryFee=22000, **total=92000** (server Haversine qayta hisobladi),
  status pending.
- `/checkout`, `/order/{number}` → **HTTP 200**; track API to'g'ri qaytardi.

### Eslatma / mijoz uchun
- **Brauzerda xarita + to'liq delivery checkout'ni sinash uchun**
  `NEXT_PUBLIC_MAP_API_KEY` kerak (dev.2gis.com, bepul). Key'siz: pickup
  buyurtma to'liq ishlaydi, delivery uchun xarita placeholder ko'rsatadi.
- Cart option price-delta hali total'ga qo'shilmaydi (M5/keyin).

---

## 2026-07-17 (4) — M5 Admin panel UI TUGALLANDI ✅

Admin panel to'liq yozildi va real backend bilan uchdan-uchgacha test qilindi.

### Backend o'zgarishi
- **Yangi endpoint** `GET /admin/menu` (`AdminListMenu`) — barcha taomlarni
  (yashirinlarni ham) qaytaradi (`?categoryId=` filtri bilan). Public `/menu`
  faqat `isAvailable` ko'rsatgani uchun admin ro'yxatiga kerak edi.
  `go build ./...` — toza (EXIT 0).
- **Eslatma**: eski backend binary porti band qilib turgani uchun qayta ishga
  tushirishda `fuser -k 8080/tcp` bilan eski jarayonni o'ldirish kerak bo'ldi.

### Frontend
- **API client** (`lib/api.ts`) admin metodlari: adminCategories/create/update/
  delete, adminMenu/create/update/delete, adminOrders(status)/updateOrderStatus,
  updateRestaurant, `uploadImage()` (multipart). Update'lar to'liq obyekt
  yuboradi (backend ReplaceOne/full-set).
- **Auth guard** `app/admin/layout.tsx` (client): token tekshiradi, `api.me()`
  bilan validatsiya (401 → login), sidebar navigatsiya + logout. `/admin/login`
  chrome'siz.
- **Komponentlar**: `components/admin/{Modal,ImageUpload}.tsx`,
  `lib/orderStatus.ts` (status label/badge/tartib).
- **Sahifalar**:
  - `/admin/login` — JWT login → localStorage → redirect.
  - `/admin` — dashboard: bugungi buyurtmalar/tushum/faol stats + so'nggi jadval.
  - `/admin/orders` — status filtri, inline status select, ochiladigan tafsilot
    (taomlar, manzil, masofa, to'lov).
  - `/admin/categories` — CRUD (modal, rasm, slug auto, aktiv toggle).
  - `/admin/menu` — CRUD kategoriya bo'yicha guruhlangan, to'liq forma (narx,
    oldPrice, teglar, rasm, available/popular).
  - `/admin/settings` — profil (logo/cover/telefon/ijtimoiy), manzil (xarita),
    ish vaqti (7 kun), delivery radius modeli. To'liq obyekt round-trip.

### Test natijalari (✅ uchdan-uchgacha, real backend + Mongo)
- `npm run build` — **toza** (14 route, 6 admin sahifasi).
- Barcha admin sahifalari → **HTTP 200**.
- **Menu CRUD**: yashirin taom (isAvailable=false) admin/menu'da ko'rindi (4),
  public'da yo'q; update→visible → public'da ko'rindi; delete → 3 ga qaytdi.
- **Order status**: pending→preparing → public track "preparing" aks ettirdi;
  `?status=preparing` filtri 1 qaytardi.
- **Upload**: PNG → `{url}` qaytdi, `/uploads/<file>` serve → **HTTP 200**.
- **Settings**: minOrder 50000→45000 saqlandi va tiklandi (full-replace OK).

### Eslatma
- Menyu option (hajm/variant) editor'i hozircha yo'q (options=[] yuboriladi) —
  M6 polish'da qo'shilishi mumkin. Delivery polygon zona UI ham yo'q (radius
  modeli tahrirlanadi; mavjud zonalar saqlanadi).

---

## 2026-07-17 (5) — Manzil qidiruv, Delivered animatsiya, Telegram login ✅

### A. Checkout manzil avtomatik-to'ldirish + xarita
- `lib/geocode.ts` — Nominatim (OSM, bepul) orqali `searchPlaces` + `reverseGeocode`
  (countrycodes=uz). 2GIS geocoder pullik bo'lgani uchun OSM tanlandi.
- `components/map/AddressAutocomplete.tsx` — yozganda (debounce 400ms) haqiqiy
  manzillar tavsiya; tanlaganda lat/lng.
- `AddressMap` — value o'zgarganda marker + markazni ko'chiradi (setCenter).
  Checkout: tavsiya tanlansa marker ko'chadi; xaritada bosilsa reverse-geocode
  bilan manzil matni to'ladi.

### B. WebGL / xarita + Hydration tuzatildi
- `reactStrictMode: false` — StrictMode ikki-marta-mount WebGL kontekst poygasini
  keltirib "Failed to obtain WebGL context" bergan.
- `formatPrice` endi deterministik (Intl.NumberFormat emas) — SSR/client hydration
  mos kelmasligi tuzatildi.
- `AddressMap`: WebGL pre-check, rAF bilan defer, "Qayta urinish", aniq xatolar.

### C. Yetkazildi animatsiyasi
- `components/order/DeliveredCelebration.tsx` — CSS konfetti + sakraydigan ✓
  (globals.css keyframes). `/order/[number]` da status=delivered bo'lganda.

### D. To'lov turlari
- Naqd / Payme / Click / Uzum (eski "card" olib tashlandi). Backend validatsiya
  `oneof=cash payme click uzum`, `lib/payment.ts` labellar.

### E. Admin: birinchi-kirish parol majburiy + login/parol tahrirlash
- Backend: `AdminUser.mustChangePassword` (seed=true), `PUT /admin/credentials`
  (joriy parol tekshiruvi, login band emasligi). Frontend: majburiy redirect
  `/admin/account`, "Hisob" sahifasi.

### F. Telegram orqali login/signup (foydalanuvchilar) 🆕
- **Arxitektura qarori**: har restoran = alohida bot + token (single-tenant).
  Token `backend/.env` da (`TELEGRAM_BOT_TOKEN`, `TELEGRAM_BOT_USERNAME`).
- Backend: `models.User` + `Users` kolleksiya, `POST /auth/telegram`
  (HMAC-SHA256 imzo tekshiruvi + upsert + user JWT), `GET /users/me`,
  `GET /users/me/orders`. `Order.userId` — kirgan user buyurtmaga bog'lanadi
  (`optionalUserID`).
- Frontend: `lib/user.tsx` (UserProvider, user_token), `TelegramLoginButton`
  (widget), `/login`, `/profile` (tarix), Header'da kirish/profil, checkout
  ismni avtomatik to'ldiradi.
- **Cheklov**: Telegram widget `localhost`da ishlamaydi — BotFather `/setdomain`
  (tunnel yoki prod domen) kerak.

### Test natijalari (✅ real backend + Mongo)
- `go build` + `npm run build` — toza (17 route).
- Nominatim: "Amir Temur" → haqiqiy UZ manzillar koordinatalari bilan.
- Telegram auth: to'g'ri imzo → JWT; **buzilgan imzo → 401**; user→me→orders OK;
  buyurtma userId bilan bog'landi; token'siz → 401.
- Credentials/payme/card testlari (oldingi sessiyada) OK.

**Eslatma**: seed admin brauzerda `yujo`ga o'zgartirilgan (majburiy oqim ishladi).

---

## 2026-07-28 — Dizayn yangilandi + namuna menyu (48 taom, rasmlari bilan) ✅

### Dizayn (frontend)
Yangi vizual til: iliq "issiq non" palitrasi + display serif sarlavhalar.

- `tailwind.config.ts`: `brand` (to'q sariq `#e2590d` + `dark/light/tint`),
  `ink` (iliq ko'mir: DEFAULT/soft/muted), `cream` fon, `shadow-card` /
  `shadow-card-hover`, `animate-fade-up`. Eski `neutral-*` ranglar butun
  `src/` bo'ylab yangi tokenlarga ko'chirildi (0 ta qoldiq).
- `globals.css`: yangi komponent klasslari — `.card`, `.badge*`, `.chip`,
  `.eyebrow`, `.section-title`, `.input`, `.btn-dark`, `.no-scrollbar`.
  `.btn*` endi pill (rounded-full) + hover/active mikro-animatsiya.
- `layout.tsx`: `next/font/google` — **Inter** (matn) + **Playfair Display**
  (sarlavha/narx), `--font-sans` / `--font-display` CSS vars orqali.
- **Header**: balandroq, logo mark (nom bosh harfi), aktiv sahifa pill'i
  (`usePathname`), savat ikonkasi + badge, mobil uchun alohida gorizontal nav.
- **Footer**: to'q (ink) fon, 4 ustun, sotsial havolalar, sahifalar.
- **MenuItemCard**: 3xl radius, hover'da ko'tarilish + rasm zoom, "Mashhur" /
  "Chegirma" badge'lari, mavjud emas holati, ikonkali "Qo'shish" tugmasi.
- **Bosh sahifa**: cover'li hero (ochiq/yopiq badge, taomlar soni / eng past
  narx / min buyurtma statistikalari) → 3 ta "perk" kartasi (hero ustiga
  chiqib turadi) → **kategoriya plitkalari** (rasm + taomlar soni) →
  mashhur taomlar → ish vaqti (bugungi kun ajratilgan) + aloqa/CTA bloki.
- **Menyu sahifasi**: sarlavha bloki, **sticky kategoriya chip-rail'i**
  (har birida taomlar soni), har bo'lim uchun kategoriya rasmi + soni.
- **About**: cover'li hero + card'lardagi aloqa/ish vaqti.

### Namuna menyu (backend seed)
- `backend/internal/seed/menu.go` (yangi): **7 kategoriya, 48 taom** —
  Milliy taomlar (7), Grill (3), Burger va fast food (6), Pitsa va pasta (5),
  Salatlar (3), Shirinliklar (11), Ichimliklar (13). Har birida UZ tavsif,
  realistik UZS narx, `isPopular` bayroqlari, ikkitasida `oldPrice` (chegirma).
- **Rasmlar**: 49 ta foto (48 taom + cover) `internal/seed/assets/` da,
  Go `embed` orqali binarga kiradi va birinchi ishga tushishda
  `UPLOAD_DIR/seed/` ga yoziladi → oddiy `/uploads/seed/*.jpg` route'i
  bilan beriladi (Docker volume'da ham saqlanadi, S3 shart emas).
  Manba: Unsplash (bepul), yuklab olingan va taomga mosligi vizual tekshirilgan.
- Seed **faqat bo'sh bazada** ishlaydi (`category` kolleksiyasi bo'sh bo'lsa) —
  haqiqiy restoran menyusi hech qachon ustiga yozilmaydi.
- `ensureRestaurant` endi `coverUrl = /uploads/seed/cover.jpg` va uzunroq
  tavsif bilan yaratadi.

### Test
- `go build ./...` ✅; backend `MONGO_DB=restaurant_seedtest` bilan ishga
  tushirildi → `GET /api/v1/menu` = **7 kategoriya / 48 taom**,
  `GET /uploads/seed/osh.jpg` → 200. (Test bazasi keyin o'chirildi.)
- `npx tsc --noEmit` ✅, `npm run build` ✅ (barcha sahifalar).
- Headless Chrome screenshot bilan tekshirildi: `/`, `/menu`, `/menu/[id]`,
  `/about`, `/cart`, `/admin/login` — layout va rasmlar joyida.

**Eslatma**: `next/font/google` build vaqtida internet talab qiladi
(Docker build'da ham) — offline build kerak bo'lsa shriftlarni lokal qilish lozim.

---

## 2026-07-28 (2) — Dark/Light tema + ko'p tillilik (UZ/RU/EN) + deploy hujjati ✅

### Tema (dark / light)
- Barcha semantik ranglar endi **CSS o'zgaruvchilari** (`globals.css`):
  `--bg, --surface, --fg, --fg-soft, --fg-muted, --brand*`. Tailwind ularni
  `rgb(var(--x) / <alpha-value>)` orqali oladi → `darkMode: "class"`.
- `.dark` klassi `<html>` ga qo'yiladi. **Flash yo'q**: `app/layout.tsx` da
  inline skript birinchi bo'yashdan oldin `localStorage.theme` (yoki tizim
  sozlamasi `prefers-color-scheme`) bo'yicha klassni qo'yadi. `<html>` ga
  `suppressHydrationWarning` — aks holda React attribute mismatch beradi.
- `lib/theme.tsx` — ThemeProvider + `useTheme()`, tanlov `localStorage` da.
- `components/site/ThemeToggle.tsx` — quyosh/oy tugmasi (header'da).
- Doim qorong'i qoladigan sirtlar uchun alohida token: **`charcoal`**
  (hero, footer, dark CTA) — ular ikkala temada ham oq matnli.
- `bg-white` → `bg-surface` butun `src/` bo'ylab (admin panel ham tema'ga
  bo'ysunadi). `color-scheme` ham o'rnatiladi → input/scrollbar to'g'ri.

### Ko'p tillilik (UZ / RU / EN)
- `lib/i18n/dictionaries.ts` — uchala til uchun to'liq lug'at (nav, home,
  menu, item, cart, checkout, order, login, profile, about, footer, payment,
  map, hafta kunlari). Tip xavfsiz: `type Dict = typeof uz`, `ru`/`en` shu
  tipga bo'ysunadi (kalit tushib qolsa — kompilyatsiya xatosi).
- Til **cookie**da (`lang`, 1 yil) saqlanadi:
  - server komponentlar (`/`, `/menu`, `/about`, `/menu/[id]`) —
    `lib/i18n/server.ts` → `getTranslations()` (`cookies()`),
  - client komponentlar — `lib/i18n/client.tsx` → `useI18n()`, boshlang'ich
    qiymat server layout'dan (hydration mismatch bo'lmasligi uchun).
  - Til almashtirilganda `router.refresh()` — server sahifalar ham qayta
    render bo'ladi.
- `components/site/LangSwitch.tsx` — **navbar tagidagi** UZ/RU/EN segment
  tugmasi (mobil navigatsiya bilan bitta qatorda).
- `formatPrice(amount, currency, lang)` — "so'm / сум / UZS";
  `weekdayName(day, lang)`; sanalar `t.locale` bilan.
- Zod validatsiya xabarlari ham tarjima qilinadi (`makeSchema(t)` checkout'da).
- `generateMetadata` — `/menu`, `/about` sarlavhalari ham tilga qarab.
- **Cheklov**: menyu *kontenti* (taom nomi/tavsifi) bazada bitta tilda —
  uni ko'p tilli qilish uchun `menu_item`/`category` ga `name_ru`, `name_en`
  maydonlari + admin panelda tab kerak bo'ladi (kelajakdagi ish).
- Admin panel matnlari o'zbekcha qoldi (restoran egasi uchun), lekin u ham
  dark/light temaga bo'ysunadi.

### Deploy
- **`nginx/restaurant.conf`** — host'dagi nginx uchun: 80→443 redirect,
  certbot ACME, TLS, gzip, `client_max_body_size 12m` (rasm yuklash),
  `/api/` va `/uploads/` → Go (8080), `/_next/static/` uzoq kesh,
  qolgani → Next (3000).
- **`docker-compose.prod.yml`** — mongo + backend + frontend, barcha portlar
  faqat `127.0.0.1` da; `.env` dan sozlamalar; `NEXT_PUBLIC_*` build-arg
  sifatida uzatiladi (`frontend/Dockerfile` ga ARG/ENV qo'shildi).
- **`.env.prod.example`**, root **`.gitignore`** (`.env` himoyasi).
- **`DEPLOY.md`** — ikki variant: (A) bitta VPS (nginx+docker+certbot),
  (B) frontend Vercel + backend VPS (`api.` subdomen, CORS, Vercel env
  jadvali, Root Directory = `frontend`). Telegram `/setdomain` qoidalari,
  backup/restore buyruqlari, xavfsizlik ro'yxati.
- `frontend/src/app/icon.svg` — favicon (favicon.ico 404 yo'qoldi).

### Test
- `tsc --noEmit` ✅, `next lint` ✅ (0 warning), `npm run build` ✅.
- Puppeteer (headless Chrome) bilan tekshirildi:
  - UZ→RU→EN almashtirish: server render matnlari o'zgaradi, cookie saqlanadi;
  - tema tugmasi: `dark` klassi, `localStorage`, sahifa yangilangach saqlanadi;
  - konsolda **xato/warning yo'q**, 404 yo'q (hydration warning tuzatildi);
  - skrinshotlar: home UZ light, home RU dark, menu EN dark, cart UZ dark,
    checkout EN dark, admin login dark — hammasi joyida.

---

## 2026-07-28 (3) — Menyu kontenti ham RU/EN bo'ldi ✅

Oldingi bosqichda faqat interfeys tarjima qilingan edi; endi **taom va
kategoriya nomlari/tavsiflari** ham uch tilda.

### Backend
- `models.go`: `Category` ga `nameRu`, `nameEn`; `MenuItem` ga `nameRu`,
  `nameEn`, `descriptionRu`, `descriptionEn`. O'zbekcha — **base**.
- Admin handlerlari butun modelni decode qilgani uchun CRUD avtomatik ishladi
  (qo'shimcha kod shart bo'lmadi). Public API barcha maydonlarni qaytaradi —
  tilni frontend tanlaydi (bitta javob, keshlash oson).
- `seed/menu.go`: 7 kategoriya + 48 taomning **hammasi** uchun RU va EN
  nom/tavsif yozildi.

### Frontend
- `lib/i18n/content.ts` — `contentName()` / `contentDescription()`:
  tanlangan til bo'yicha matn, tarjima bo'sh bo'lsa **base (uz)** ga tushadi.
- Ishlatilgan joylar: MenuItemCard, menyu sahifasi (kategoriya sarlavhalari va
  chip'lar), bosh sahifadagi kategoriya plitkalari, taom sahifasi, savat va
  checkout ro'yxati.
- `CartLine` ga `nameRu`/`nameEn` qo'shildi — savatdagi nomlar ham til
  almashtirilganda darhol o'zgaradi (eski saqlangan savatlar base nomda
  qoladi — bu kutilgan fallback).
- **Buyurtmaga base (uz) nomi yuboriladi** — admin panelda buyurtmalar bitta
  tilda ko'rinadi.
- Admin panel: kategoriya formasida "Nomi (RU/EN)", taom formasida alohida
  **"Tarjimalar"** bloki (nom + tavsif RU/EN, placeholder — o'zbekchasi).

### Test
- `go build` + `go vet` ✅, `tsc` ✅, `next lint` ✅, `npm run build` ✅.
- Yangi bazada seed → `/api/v1/menu`: 48 taom, har birida `nameRu`/`nameEn`
  va tavsiflar to'g'ri keldi.
- Admin API round-trip: `PUT /admin/menu/{id}` bilan `nameRu`/`descriptionEn`
  saqlandi va qaytib o'qildi ✅.
- Brauzerda: RU menyusi ("Национальные блюда", "Плов"), EN menyusi
  ("Uzbek classics", "Plov (pilaf)"); savatga qo'shilgan taom RU→EN
  almashtirilganda nomini o'zgartirdi ✅; buyurtma yaratish ishlaydi ✅.
- Savat xulosasidagi "Yetkazib berish / Checkout'da hisoblanadi" qatori
  RU'da yopishib qolgani tuzatildi (gap + text-right).

---

## 2026-07-28 (4) — SMS login, savat stepper, admin "Foydalanuvchilar" ✅

### 1. Eski baza 48 talik menyuga o'tkazildi
- Yangi buyruq: **`go run ./cmd/seedmenu -db <baza> [-replace] [-y]`** —
  namuna menyuni to'la bazaga ham yozadi (oddiy seed faqat bo'sh bazada
  ishlaydi). `-replace` eski kategoriya/taomlarni o'chiradi (tasdiq so'raydi).
- `restaurant` bazasi ko'chirildi: 1 kategoriya / 3 taom → **7 / 48**.
  Buyurtmalar (6) va foydalanuvchilar (3) tegilmadi. Eski menyu zaxirasi:
  `scratchpad/restaurant-menu-backup.json`.

### 2. Til almashtirgich navbarga ko'chdi
- `LangSwitch` endi tema tugmasi yonida, navbar ichida. Tagidagi qator faqat
  **mobil navigatsiya** uchun qoldi (md dan kichik ekranlarda).
- Menyu sahifasidagi sticky chip-rail offset'i shunga moslandi.

### 3. Menyuda +/− stepper
- `MenuItemCard`: taom savatda bo'lsa "Qo'shish" tugmasi o'rniga
  **− soni +** ko'rinadi (savat holati bilan bog'langan, 0 ga tushsa
  qatordan chiqadi). Sonlar header'dagi savat badge'i bilan sinxron.

### 4. Telefon + SMS kod bilan login (asosiy usul)
Sabab: buyurtmani faqat tasdiqlangan raqamli mijoz bera oladi; raqam profilda
saqlanadi va admin ko'ra oladi.

**Backend**
- `internal/sms` — `Sender` interfeysi + 3 ta provayder:
  - **demo** (default): SMS ketmaydi, kod API javobida qaytadi;
  - **eskiz** (`https://notify.eskiz.uz`, email+parol → bearer token,
    `POST /api/message/sms/send`);
  - **playmobile** (broker-api, basic auth, JSON `messages[]`).
  Provayder `SMS_PROVIDER` bilan tanlanadi; credential yo'q bo'lsa demo'ga
  tushadi.
- `POST /auth/phone/request` — raqam normalizatsiya (`+998 90 123 45 67` →
  `998901234567`), 6 xonali kod, **bcrypt hash** bilan `phone_code`'da,
  3 daqiqa TTL, 60 soniya cooldown, 5 urinish limiti.
- `POST /auth/phone/verify` — kod to'g'ri bo'lsa user yaratiladi/topiladi va
  JWT beriladi (`authProvider: "phone"`).
- `PUT /users/me` — ism + manzillar; `POST /users/me/phone/request|verify` —
  raqamni **qayta tasdiqlash orqali** o'zgartirish (band raqam → 409).
- `User` modeliga `authProvider` va `addresses[]` qo'shildi.

**Frontend**
- `components/auth/PhoneLogin.tsx` — 2 bosqichli forma; demo rejimda kod
  ekranda ko'rsatiladi.
- `/login` — **Telefon raqam / Telegram** tab'lari (Telegram bot sozlangan
  bo'lsagina ko'rinadi), `?next=` va `?reason=order` qo'llab-quvvatlanadi.
- **Savatdagi "Buyurtma berish" endi login talab qiladi** → mehmon
  `/login?next=/checkout&reason=order` ga yuboriladi; checkout sahifasi ham
  himoyalangan. Kirgan mijozning ismi/raqami avtomatik to'ldiriladi,
  saqlangan manzillari chip sifatida chiqadi.
- Profil: "Mening ma'lumotlarim" kartasi — ism, tasdiqlangan raqam,
  manzillar CRUD, raqamni SMS bilan o'zgartirish.

### 5. Admin → Foydalanuvchilar
- `GET /admin/users?q=` — mijozlar ro'yxati + buyurtmalar soni va summasi
  (bitta aggregation bilan).
- `/admin/users` sahifasi: qidiruv, 4 ta stat (jami / telefon orqali /
  Telegram orqali / buyurtmalar summasi), jadval (avatar, raqam, kirish usuli,
  manzillar soni, buyurtmalar, summa, sana).

### Test
- `go build` + `go vet` ✅, `tsc` ✅, `next lint` ✅.
- API: kod so'rash → demo kod; noto'g'ri kod → 401; cooldown → 429;
  verify → JWT + user; `PUT /users/me` → ism/manzil saqlandi; token'siz → 401;
  raqam almashtirish → yangi raqam; band raqam → 409; `/admin/users` → 5 ta
  mijoz, `?q=` filtri ishladi, token'siz → 401.
- Brauzer (Puppeteer): menyuda stepper ishladi (1→3→2, header badge sinxron);
  savatdan "Buyurtma berish" → login sahifasi; demo kod bilan kirish →
  checkout'ga qaytdi; profil raqamni ko'rsatdi; **konsolda xato yo'q**.
- Admin panel: Foydalanuvchilar sahifasi 6 ta mijozni ko'rsatdi.

**Eslatma**: prod'da `SMS_PROVIDER=eskiz|playmobile` qilinadi va shartnoma
bo'yicha sender name / shablon moderatsiyadan o'tkaziladi. Demo rejimda kod
API javobida qaytgani uchun **prod'da demo qoldirilmaydi**.

---

## 2026-07-28 (5) — Tuzatishlar: til popup, footer, savat sinxroni, profil xaritasi ✅

### 1. Til almashtirgich → popup
- Navbarda endi **faqat joriy til** ko'rinadi (globus + `UZ ▾`); bosilganda
  ustidan ochiladigan popup'da uchala til to'liq nomi bilan.
  Tashqariga bosilsa yoki `Esc` bilan yopiladi.

### 2. Footer light rejimda ham yorug'
- Footer `bg-charcoal` (doim qora) edi → `bg-surface` + `text-ink` ga o'tdi,
  ya'ni temaga bo'ysunadi. Hero hamon qorong'i (fon rasm ustidagi gradient).

### 3. "menu item not found" xatosi tuzatildi
- **Sabab**: brauzerdagi savat eski menyu ID'lari bilan qolib ketgan edi
  (`seedmenu -replace` eski taomlarni o'chirgan) — backend buyurtmani rad
  etardi.
- **Yechim**: `CartProvider` yuklanganda savatni jonli menyu bilan solishtiradi
  — yo'q bo'lgan/mavjud bo'lmagan qatorlar chiqarib tashlanadi, qolganlarining
  nomi/narxi/rasmi yangilanadi. Savat sahifasida sariq ogohlantirish chiqadi.
- Backend xabari ham tushunarli bo'ldi: `"<taom> menyuda topilmadi — savatni
  yangilang"`, mavjud emas holati uchun ham o'zbekcha xabar.

### 4. Profilda manzil — 2GIS xarita + qo'lda yozish (sinxron)
- Yangi umumiy komponent **`components/map/AddressPicker.tsx`**: matn kiritish
  (avtomatik takliflar bilan) + xarita markeri, ikkalasi sinxron —
  xaritaga bosilsa reverse-geocode matnni to'ldiradi, taklif tanlansa marker
  ko'chadi. Checkout ham, profil ham shuni ishlatadi.
- Profil: har bir manzil uchun "Xaritada" tugmasi → xarita + izoh maydoni.
- **Checkout**: saqlangan manzillar kartochka sifatida chiqadi (birinchisi
  avtomatik tanlanadi) + "**+ Yangi manzil**" varianti xaritani ochadi;
  yangi manzil uchun "profilimga saqlash" belgisi bor (buyurtmadan keyin
  profilga qo'shiladi).
- Avtomatik takliklar endi faqat foydalanuvchi yozganda ochiladi (saqlangan
  manzil to'ldirilganda ro'yxat o'z-o'zidan ochilib xaritani to'sib qo'yardi).
- Telefon maydonlari `+998 ` bilan oldindan to'ldirilmaydi (placeholder bor) —
  raqam qo'shib yuborilishining oldini oladi.

### 5. Admin parolini unutish holati
- Yangi buyruq **`backend/cmd/adminreset`**:
  `-list`, `-username`, `-password` (yozilmasa terminal so'raydi, ekranda
  ko'rinmaydi), `-create`, `-force-change`, `-db`.
- Docker image endi barcha `cmd/*` binarlarini o'z ichiga oladi →
  `docker compose exec backend /app/adminreset -username yujo -password '...'`.
- `DEPLOY.md` ga alohida bo'lim: nega panelda "parolni unutdim" yo'q
  (single-tenant, email/SMS infratuzilmasi yo'q) va tiklash tartibi.

### Test
- `go build` + `go vet` ✅, `tsc` ✅, `next lint` ✅, `npm run build` ✅.
- `go run ./cmd/adminreset -list` → mavjud admin ko'rindi (parol tegilmadi).
- Brauzer (Puppeteer, konsol toza):
  - eskirgan savat → ogohlantirish chiqdi va qator olib tashlandi;
  - til popup: navbarda `UZ`, ochilganda 3 ta variant;
  - light rejimda footer foni `rgb(255,255,255)`;
  - telefon bilan kirish → checkout → **saqlangan manzil avtomatik tanlandi**
    ("Uy — Chilonzor 5-kvartal"), yetkazish narxi hisoblandi, buyurtma
    muvaffaqiyatli yaratildi (`/order/BW43-4050`) — eski xato takrorlanmadi;
  - profilda "Xaritada" → 2GIS canvas yuklandi, matn va marker sinxron.

---

## 2026-07-28 (6) — Admin: buyurtmalar boshqaruvi + mijoz kartochkasi ✅

### Buyurtmalar bo'limi qayta ishlandi
Savol "har bir statusni qo'lda o'zgartirish qiyin-ku?" — javob: **holat qo'lda
qoladi** (restoran real holatni faqat o'zi biladi), lekin **bir bosishga**
tushirildi:

- Har bir buyurtmada **"keyingi qadam"** tugmasi: `Tasdiqlash →
  Tayyorlashni boshlash → Yo'lga chiqdi → Yetkazildi`. Olib ketish
  buyurtmalarida `on_the_way` o'tkazib yuboriladi ("Berildi").
  Yonidagi select faqat tuzatish uchun (orqaga qaytarish, bekor qilish).
- **"Faol" filtri** (default) — tugallanmagan buyurtmalar; status bo'yicha
  filtrlar va **qidiruv** (№, ism, telefon, manzil) qo'shildi.
- Ro'yxat **har 20 soniyada avtomatik yangilanadi**, yangi kelganlar soni
  ko'rsatiladi, yangi buyurtma kartasi ajratib turadi.
- Karta ochilganda **to'liq chek**: taomlar × soni va narxi, oraliq summa,
  yetkazish (km bilan), jami, to'lov turi, mijoz + telefon (bosilsa
  qo'ng'iroq), manzil + 2GIS havolasi, **vaqt jadvali**.
- Tez amallar: qo'ng'iroq, mijoz ko'rinishi (`/order/<№>`), bekor qilish.

### Buyurtma tarixi (statusHistory)
- `Order.statusHistory [{status, at}]` — yaratilganda `pending` yoziladi,
  har bir status o'zgarishi `$push` bilan qo'shiladi.
- Shu tufayli chekda "qachon tasdiqlandi / tayyorlandi / yetkazildi"
  ko'rinadi — shikoyat kelganda asosiy savolga javob.

### Mijoz kartochkasi — `/admin/users/[id]`
- Ro'yxatdagi har bir qator bosiladi (yoki "Batafsil →").
- Sahifada: ism/telefon/kirish usuli, **ro'yxatdan o'tgan sana**,
  buyurtmalar soni (yetkazilgan/bekor), umumiy summa, **oxirgi buyurtma**,
  **saqlangan manzillar** (2GIS havolasi bilan) va **butun buyurtmalar
  tarixi** — har biri ochilib to'liq chekni ko'rsatadi (ID, taomlar, summalar,
  vaqt jadvali).
- Ro'yxat jadvaliga "Oxirgi buyurtma" ustuni qo'shildi (`$max: createdAt`
  aggregation bilan, qo'shimcha so'rovsiz).
- Chekdan mijoz profiliga va profildan buyurtmalar bo'limiga o'tish havolalari.

### Backend
- `GET /admin/orders` — `?status=`, `?q=` (№/ism/telefon/manzil), `?userId=`,
  `?limit=`; yangi `GET /admin/orders/{id}`.
- Yangi `GET /admin/users/{id}` — user + stats (soni, summasi, yetkazilgan,
  bekor, birinchi/oxirgi buyurtma) + barcha buyurtmalar.
- `PUT /admin/orders/{id}/status` endi tarixga yozadi va vaqtni qaytaradi.
- Umumiy komponent `components/admin/OrderReceipt.tsx` (buyurtmalar sahifasi
  ham, mijoz kartochkasi ham shuni ishlatadi), `lib/orderFlow.ts` — keyingi
  status/label va vaqt formatlari.

### Test
- `go build` + `go vet` ✅, `tsc` ✅, `next lint` ✅, `npm run build` ✅.
- API: `?q=BW43` → 1 natija; status 2 marta o'zgartirildi → `statusHistory`
  ikkala vaqtni yozdi; `/admin/users/{id}` → profil + 1 buyurtma + statistika.
- Brauzer (konsol toza): buyurtmalar sahifasida 5 ta faol buyurtma, har birida
  to'g'ri "keyingi qadam" tugmasi; chek ochildi (taomlar, manzil, vaqt
  jadvali); mijoz kartochkasi statistikalar va chek bilan ochildi.

---

## 2026-07-29 — Telegram login olib tashlandi + menyu variantlari + zona chizish ✅

Prioritet B bo'yicha ish: Telegram login kerak emas deb qaror qilindi, uning
o'rniga qolgan ikkita funksional bo'shliq (options editor, polygon zonalar)
yopildi.

### 1. Telegram login butunlay olib tashlandi
Sabab: mijozlar telefon + SMS kod bilan kiradi, ikkinchi usul kerak emas
(widget real domen, BotFather `/setdomain` va bot boshqaruvini talab qilardi).
- Backend: `TelegramAuth`, `TelegramWebAppAuth`, `verifyTelegramHash`,
  `verifyWebAppInitData`, `issueUserToken` o'chirildi; `/auth/telegram` va
  `/auth/telegram/webapp` route'lari, `TELEGRAM_BOT_TOKEN/USERNAME` config'lari
  ham. `User` modelidan `telegramId`, `username`, `photoUrl` chiqarildi
  (ular faqat Telegram tufayli bor edi).
- Frontend: `TelegramLoginButton.tsx` o'chirildi, `/login` bitta telefon
  formasiga aylandi, Mini App skripti `layout.tsx` dan olib tashlandi,
  `TelegramUser` tipi → `SiteUser`, avatarlar bosh harf doirasiga o'tdi.
  Admin "Foydalanuvchilar"da "Kirish usuli" ustuni olib tashlandi, statistika
  kartalari "Buyurtma bergan" / "Manzil saqlagan" ga almashtirildi.
- Config/docs: `.env.local.example`, `.env.prod.example`,
  `docker-compose.prod.yml`, `frontend/Dockerfile`, `nginx/restaurant.conf`,
  `next.config.ts`, `CLAUDE.md`, `DEPLOY.md` tozalandi.
- **Qoldi**: `restaurant.socials.telegram` — bu aloqa havolasi, login emas.

### 2. Menyu variantlari (options) — to'liq oqim
- **Model**: `MenuOption` ga `nameRu/nameEn`, `required`, `multiple`;
  `OptionChoice` ga `nameRu/nameEn`. `OrderItem.Options` endi
  `[]string` emas — `[{name, choice, priceDelta}]`, `Price` esa taom narxi +
  tanlangan deltalar.
- **Server narxni qayta hisoblaydi** (`resolveOptions`, util.go): mijoz faqat
  qaysi tanlovni belgilaganini aytadi, `priceDelta` doim bazadan olinadi.
  Majburiy guruh tanlanmasa / tanlov menyuda bo'lmasa / single guruhda ikki
  tanlov bo'lsa — 400 (o'zbekcha xabar bilan).
- **Admin**: `components/admin/OptionsEditor.tsx` — guruh qo'shish/o'chirish,
  majburiy + ko'p tanlov bayroqlari, tanlovlar jadvali (UZ/RU/EN + narx farqi).
  Bo'sh nomli guruh/tanlov saqlashda tashlab yuboriladi.
- **Mijoz**: `components/menu/OptionPicker.tsx` (radio / checkbox uslubida
  chip'lar, narx farqi ko'rinadi), taom sahifasida; majburiy guruh
  tanlanmasa qizil xato. Kartochkada variantli taomda "Qo'shish" o'rniga
  **"Tanlash"** (taom sahifasiga olib boradi).
- **Savat**: qator kaliti endi taom + tanlovlar (`lineIdFor`) — bir taom
  turli variantlar bilan alohida qator. `localStorage` kaliti `cart_v1` →
  **`cart_v2`**. Menyu bilan sinxronlashda tanlovlar ham qayta tekshiriladi
  (o'chirilgan/qayta nomlangan tanlov bo'lsa qator olib tashlanadi).
- Savat, checkout xulosasi va admin chekida tanlovlar ko'rinadi.

### 3. Yetkazib berish zonalari (polygon) — admin UI
- `components/map/ZoneMap.tsx` (2GIS MapGL: polygon + polyline + vertex
  markerlari; `AddressMap` kabi almashtiriladigan modul) va
  `components/admin/DeliveryZonesEditor.tsx`.
- `/admin/settings` da yangi bo'lim: zona qo'shish, nom + narx, xaritada
  bosib nuqta qo'yish, oxirgi nuqtani bekor qilish, chizmani tozalash,
  zonani o'chirish. Aktiv zona to'q sariq, qolganlari kulrang.
- **Backend tuzatish**: 3 nuqtadan kam zonalar endi e'tiborga olinmaydi —
  ilgari yarim chizilgan bitta zona butun yetkazib berishni o'chirib
  qo'yardi (zona modeli yoqilib, hech qayerga to'g'ri kelmasdi).

### Test (✅ real backend + Mongo, `restaurant_opttest`)
- `go build` + `go vet` ✅, `tsc --noEmit` ✅, `next lint` ✅ (0 warning),
  `npm run build` ✅ (20 route).
- Options API: `PUT /admin/menu/{id}` bilan 2 guruh saqlandi va public
  `/menu/{id}` da qaytdi; buyurtma "Katta + Qazi" ×2 → server unit narxni
  **68000** qildi (45000+8000+15000), **mijoz yuborgan `priceDelta:0`
  e'tiborga olinmadi**; majburiy tanlanmasa 400, single guruhda 2 tanlov 400,
  yo'q tanlov 400, multiple guruhda 2 tanlov ✅.
- Zonalar: zona ichida → 12000 "Markaz"; tashqarisida → mavjud emas;
  faqat chala (2 nuqtali) zona → **radius modeliga qaytdi** (22000);
  zonasiz → radius ✅.
- Sahifalar: `/`, `/menu`, `/menu/[id]`, `/cart`, `/checkout`, `/login`,
  `/about`, `/admin/login` → 200; variantli taom sahifasida "Hajm/Katta/
  Qazi/majburiy" render bo'ldi, menyuda "Tanlash" tugmasi chiqdi; dev
  konsolida xato yo'q.

**Eslatma**: mijozlarning eski savatlari (`cart_v1`) o'qilmaydi — bir marta
bo'shab qoladi. Bu ataylab: eski qatorlarda variant ma'lumoti yo'q.

---

## 2026-07-29 (2) — Zonalar mijozga ham ko'rinadi ✅

Admin panelda chizilgan zona endi mijoz tomonida ham ko'rinadi.

- `AddressMap` ga ixtiyoriy `zones` propi qo'shildi: polygonlar marker ostiga
  chiziladi (to'q sariq, yarim shaffof). Faqat 3+ nuqtali zonalar — backend
  bilan bir xil qoida.
- **Xarita zonaga moslashadi** (`fitBounds`): mijoz hali nuqta tanlamagan
  bo'lsa xarita butun yetkazish hududini ko'rsatadi. Aks holda zona shahar
  hajmida bo'lgani uchun xarita uning ichida qolib, chegara ko'rinmasdi
  (birinchi urinishda aynan shu bo'ldi). Moslashtirish bir marta ishlaydi —
  keyin foydalanuvchi xaritani o'zi boshqaradi.
- `AddressPicker` xarita tagida **legenda** ko'rsatadi: zona nomi + narxi va
  "zonadan tashqariga yetkazilmaydi" izohi (UZ/RU/EN).
- Ulangan joylar: **checkout** (yangi manzil kiritilganda) va **profil**
  ("Xaritada" tugmasi). Profil endi restoran profilini ham yuklaydi —
  xarita markazi va zonalar uchun.

### Test
- `tsc` ✅, `next lint` ✅ (0 warning).
- Headless Chrome (CDP, demo-SMS bilan kirilgan sessiya): `/checkout` da
  legenda "Zona 1 — 10 000 so'm" chiqdi, xaritada polygon **ko'rindi**
  (skrinshot bilan tasdiqlangan), konsol toza. Test mijozi keyin bazadan
  o'chirildi.

**Eslatma**: dev server ishlab turganda `npm run build` ishlatmaslik kerak —
ikkalasi bitta `.next` papkasini ishlatadi va dev serverni buzadi
(bu sessiyada bir marta shu sabab 500 xatosi chiqdi).

---

## 2026-07-29 (3) — Zona ichiga metka bug'i, saqlangan manzil xaritasi, zona narxlash ✅

### 1. BUG: zona ichiga metka qo'yib bo'lmasdi 🐞
**Sabab**: 2GIS MapGL polygonlari sukut bo'yicha `interactive: true` — polygon
xarita click'ini o'ziga oladi. Shu sababli mijoz faqat zonadan **tashqariga**
metka qo'ya olardi (ya'ni aynan yetkazib berilmaydigan joyga).
**Yechim**: `interactive: false` — `AddressMap` (mijoz) va `ZoneMap` (admin)
polygonlarida. Admin tomonda ham xuddi shu muammo bor edi: chizilgan zona
ichiga yangi nuqta qo'shib bo'lmasdi.
**Tekshirildi**: headless Chrome'da xarita markaziga bosilganda nuqta
41.28151, 69.27057 tanlandi va yetkazish narxi 10 000 so'm hisoblandi.

### 2. Saqlangan manzil uchun ham xarita
- Checkout'da xarita endi **doim** ko'rinadi (ilgari faqat "+ Yangi manzil"da).
  Saqlangan manzil tanlansa marker o'sha nuqtaga qo'yiladi va zonalar
  ustiga chiziladi — mijoz manzili zona ichidami-yo'qmi darhol ko'radi va
  kerak bo'lsa markerni aniqlashtiradi (bu **faqat shu buyurtmaga** ta'sir
  qiladi; profildagi manzil "profilimga saqlash" belgilanmasa o'zgarmaydi).
- "Profilimga saqlash" belgisi endi faqat yangi manzil rejimida ko'rinadi.

### 3. Har zona uchun alohida narxlash usuli
Restoran o'zi hal qiladi:
- **Belgilangan narx** (`pricing: "fixed"`) — zona ichida har qanday manzilga
  bir xil `fee`.
- **Masofa bo'yicha** (`pricing: "perKm"`) — `baseFee + ceil(km) * perKm`,
  masofa restorandan Haversine bilan.
- `pricing` bo'sh bo'lgan eski hujjatlar `fixed` kabi ishlaydi (moslik saqlandi).
- Admin: har zona kartochkasida ikki tugmali tanlov + tegishli maydonlar va
  jonli misol ("8 000 + 3 000 × 4 km = 20 000 so'm").
- Mijoz legendasida km rejimi `8 000 so'm + 3 000 so'm/km` ko'rinishida.

### Test
- `go build` + `go vet` ✅, `tsc` ✅, `next lint` ✅, `npm run build` ✅.
- Alohida test bazasida (`restaurant_zonetest`, port 8081 — jonli ma'lumotga
  tegilmadi): fixed → 15 000; perKm 8000+3000/km → 1.11 km da **14 000**,
  3.34 km da **20 000** (ikkalasi ham kutilganidek); `pricing`siz eski zona
  → 7 000 (fixed); zonadan tashqarida → mavjud emas ✅.
- Brauzerda: saqlangan manzil ("Uy") tanlanganda xarita marker bilan ochildi,
  zona chizildi, jami 100 000 so'm to'g'ri hisoblandi (skrinshot). Test
  mijozi va test bazasi keyin o'chirildi.

---

## 2026-07-29 (4) — Yetkazib berish sozlamalari birlashtirildi + saqlash bug'i ✅

### 1. BUG: "masofa bo'yicha" zona saqlangach fixed'ga qaytardi 🐞
**Sabab kodda emas edi** — ishlab turgan backend jarayoni `pricing` maydoni
qo'shilishidan **oldin** ishga tushirilgan edi. `UpdateRestaurant` butun
strukturani `$set` qilgani uchun eski binar bilmagan maydonlar (`pricing`,
`baseFee`, `perKm`) har saqlashda o'chib ketardi.
**Yechim**: backend yangi kod bilan qayta ishga tushirildi. Model o'zgargandan
keyin `go run ./cmd/server` ni qayta yoqish shart — aks holda yangi maydonlar
jimgina yo'qoladi. (Bu eslatma sifatida yozib qo'yildi.)

### 2. Ikkita yetkazib berish bo'limi bittaga birlashtirildi
Ilgari "Yetkazib berish (radius modeli)" va "Yetkazib berish zonalari
(xaritada)" alohida turardi va qaysi biri ishlayotgani tushunarsiz edi.

Endi **bitta "Yetkazib berish" bo'limi**:
- yoqish, minimal buyurtma, bepul yetkazish chegarasi;
- so'ng bitta savol — **"Narx qanday hisoblanadi?"** — ikki kartochka:
  **Masofa bo'yicha** (boshlang'ich + har km + maksimal masofa) yoki
  **Xaritadagi zonalar bo'yicha** (zona editori);
- tanlanmagan usulning maydonlari ko'rinmaydi, lekin ma'lumot bazada saqlanib
  qoladi (radius rejimida "chizilgan zonalar saqlanadi, ishlatilmaydi" deb
  yoziladi).

**Backend**: `delivery.mode` (`"radius" | "zones"`) qo'shildi. Bo'sh bo'lsa
(eski hujjatlar) — chizilgan zona bor-yo'qligiga qarab aniqlanadi, ya'ni
avvalgi xatti-harakat. `mode: "zones"` bo'lsa-yu ishlaydigan zona bo'lmasa,
radius modeliga qaytadi — hech kimga yetkazmay qo'ymaslik uchun.

### Test (alohida `restaurant_modetest` bazasi, port 8081 — jonli ma'lumotga tegilmadi)
- perKm zona saqlandi → `pricing=perKm, baseFee=8000, perKm=3000` **saqlanib
  qoldi**; ikkinchi marta saqlangach ham o'zgarmadi ✅ (asosiy bug tasdiqlandi
  va yopildi).
- `mode=zones` → 1.11 km da 14 000 (8000 + 2×3000) ✅.
- `mode=radius` → zona nomi bo'sh, radius narxi 16 000, zonalar bazada turibdi ✅.
- `mode=zones` + chala zona → radius'ga qaytdi, `available: true` ✅.
- `go build`/`go vet` ✅, `tsc` ✅, `next lint` ✅.

**Eslatma**: bitta loyiha papkasida ikkita `next dev` (yoki `next dev` +
`next build`) parallel ishlamaydi — ikkalasi `.next` ni bo'lishadi va dev
server 500 qaytara boshlaydi. Bu sessiyada ikki marta shu bo'ldi.

---

## 2026-07-29 (5) — Kuryer tizimi (PWA) + buyurtmani qabul qilish tugmasi ✅

### 1. Buyurtmalar: "Qabul qilish" tugmasi
Yangi (`pending`) buyurtma kartochkasining **eng oldida** katta
"✓ Qabul qilish" tugmasi va "Yangi" belgisi paydo bo'ldi. Boshqa bosqichlarda
avvalgidek "keyingi qadam" tugmasi ishlaydi (takrorlanmasligi uchun `pending`
holatida u ko'rsatilmaydi).

### 2. XAVFSIZLIK: rollar ajratildi 🔐
Admin route'lari faqat `RequireAuth` ishlatardi — ya'ni **istalgan haqiqiy
token**, jumladan mijozning SMS-login tokeni, admin API'ga kira olardi
(menyuni tahrirlash, barcha mijozlarni ko'rish, buyurtma holatini
o'zgartirish). Yangi `middleware.RequireRole` qo'shildi:
- admin guruhi → `owner`, `manager`
- mijoz guruhi → `user`
- kuryer guruhi → `courier`

Tekshirildi: mijoz tokeni bilan `/admin/couriers` → **403**.

### 3. Kuryer tizimi
**Backend** (`internal/handlers/courier.go`, `admincouriers.go`):
- `courier` kolleksiyasi: ism, telefon, login, bcrypt parol, transport,
  status (`off`/`free`/`busy`), `isActive`, oxirgi joylashuv.
- Kuryer API: login, me, status, location (batch), mening buyurtmalarim,
  o'z buyurtmasini `on_the_way`/`delivered` ga o'tkazish.
- Admin API: kuryer CRUD + `PUT /admin/orders/{id}/courier` (biriktirish).
- Status avtomatik: biriktirilganda `busy`, yetkazilgach (boshqa faol
  buyurtmasi qolmasa) `free`. Faol buyurtmali kuryerni o'chirib bo'lmaydi (409).
- `Order` ga `courierId` + `courierName` (nom denormalizatsiya qilingan —
  hisob o'chirilsa ham chek to'g'ri o'qiladi).

**Admin panel** — yangi `/admin/couriers`:
- kuryer qo'shish/tahrirlash (login + parol admin tomonidan beriladi),
  faol/nofaol, transport turi;
- jadval: holat, oxirgi joylashuv va qachon yuborilgani;
- **jonli xarita** — barcha kuryerlar + restoran, har 15 soniyada yangilanadi.
- Buyurtmalar sahifasida har bir yetkazish buyurtmasiga **kuryer tanlash**
  select'i (biriktirilmagan bo'lsa to'q sariq ramka bilan ajralib turadi).

**Kuryer PWA** — `/kuryer`:
- `public/manifest.webmanifest` (scope `/kuryer`, standalone) +
  `public/courier-sw.js` → telefonga ilova sifatida o'rnatiladi.
- `/kuryer/login` — admin bergan login/parol.
- Asosiy ekran: smena tugmalari (Ishda emas / Bo'sh / Band), joylashuv
  indikatori, mening buyurtmalarim — mijoz, telefon (bosilsa qo'ng'iroq),
  manzil, taomlar, naqd summa, "Xaritada ochish" va bitta tugma
  ("Olib chiqdim" → "Yetkazdim").
- `lib/courier.tsx`: `watchPosition`, 15 soniyalik batch yuborish,
  oflayn bufer (`localStorage`), fon rejimiga o'tganda darhol flush,
  `online` hodisasida qayta yuborish, smenada **Wake Lock**.

**Mijoz**: `/order/{number}` da buyurtma `on_the_way` bo'lsa kuryer va
manzil xaritada ko'rinadi + kuryerga qo'ng'iroq havolasi. Boshqa
bosqichlarda joylashuv umuman qaytarilmaydi.

### ⚠️ Fon rejimida kuzatuv — brauzer cheklovi
"Ilovadan chiqib ketilganda ham geolokatsiya uzatilib tursin" degan talab
**veb-texnologiyalarda to'liq bajarilmaydi**: service worker'ga geolocation
berilmaydi, yopilgan PWA esa umuman ishlamaydi (iOS'da ayniqsa qat'iy).
Shu sababli maksimal darajada quyidagilar qilindi:
- smenada Wake Lock (ekran o'chmaydi, ilova tirik qoladi);
- fon rejimiga o'tishda oxirgi nuqta darhol yuboriladi;
- oflayn bufer — aloqa tiklanganda hammasi yuboriladi;
- kuryer ekranida "ilovani yopmang" ogohlantirishi.
Haqiqiy fon kuzatuvi uchun **native o'ram** kerak (Capacitor yoki TWA +
Android foreground service) — bu keyingi bosqich sifatida ochiq qoldi.

### Test (alohida `restaurant_ctest` bazasi, port 8081)
- kuryer yaratish ✅, band login → 409 ✅, login ✅, noto'g'ri parol → 401 ✅;
- **rollar**: kuryer tokeni admin API'ga → 403, admin tokeni kuryer API'ga → 403 ✅;
- status `free` ✅, batch joylashuvdan **eng yangi nuqta** olindi ✅;
- biriktirilgach kuryer `busy` ✅, kuryer ro'yxatida buyurtma ko'rindi ✅;
- `on_the_way` → mijoz track'ida kuryer joylashuvi ko'rindi ✅;
- `delivered` → kuryer `free` ✅, track'da joylashuv **yashirildi** ✅;
- begona buyurtmani o'zgartirish → 404 ✅; faol buyurtmali kuryerni
  o'chirish → 409 ✅.
- Sahifalar: `/kuryer`, `/kuryer/login`, `/manifest.webmanifest`,
  `/courier-sw.js`, `/admin/couriers` → 200 ✅; manifest scope `/kuryer`,
  display `standalone` ✅.
- `go build`/`go vet` ✅, `tsc` ✅, `next lint` ✅.

---

## 2026-07-29 (6) — Admin panel va kuryer ilovasi uch tilli + tema tugmasi ✅

Ilgari admin panel faqat o'zbekcha edi (CLAUDE.md da ataylab shunday yozilgan
edi) va til/tema almashtirgichi yo'q edi. Endi ikkalasida ham bor.

### Lug'at
- Yangi **`src/lib/i18n/admin.ts`** — admin panel + kuryer ilovasi uchun
  alohida lug'at (UZ/RU/EN), mijoz lug'atidan ajratilgan. O'zbekcha manba:
  `AdminDict = typeof adminUz`, shuning uchun ru/en da kalit tushib qolsa
  **kompilyatsiya xatosi** beradi.
- `useAdminT()` bir xil `lang` cookie'ni o'qiydi — saytda tanlangan til
  panelga ham, kuryer ilovasiga ham o'tadi.
- Buyurtma holatlari (`STATUS_LABEL`) va "keyingi qadam" tugmasi yorliqlari
  ham lug'atga ko'chdi. `nextActionLabel(order, t.nextAction)` — olib ketish
  buyurtmalari uchun alohida matn saqlanib qoldi.
- Kuryer provayderi (`lib/courier.tsx`) endi geolokatsiya xatosini **kod**
  sifatida qaytaradi (`denied`/`failed`/`unsupported`), matn UI da tarjima
  qilinadi.

### Til va tema tugmalari
`LangSwitch` + `ThemeToggle` qo'shildi:
- admin sidebar'iga (mobil yuqori panelga ham),
- admin login sahifasiga,
- kuryer ilovasining bosh ekraniga va login sahifasiga.

Tema allaqachon CSS o'zgaruvchilari orqali ishlagan, endi panelning o'zidan
almashtirsa bo'ladi (avval faqat sayt tomonida edi).

### Tarjima qilingan joylar
Layout/nav, login, boshqaruv paneli, buyurtmalar, kuryerlar, foydalanuvchilar
(ro'yxat + kartochka), kategoriyalar, menyu, sozlamalar, hisob, hamda
komponentlar: `OrderReceipt`, `ImageUpload`, `OptionsEditor`,
`DeliveryZonesEditor`, kuryer ilovasining ikkala ekrani.

Tarjima qilinmagani (ataylab): brend nomlari (Instagram/Telegram/Facebook),
PWA ilova nomi ("Kuryer"), va kod izohlari.

### Test
- `tsc` ✅, `next lint` ✅ (0 warning), `npm run build` ✅.
- `lang` cookie bilan tekshirildi: `/admin/login` → "Admin panel / Kirish",
  "Админ-панель / Войти", "Admin panel / Sign in"; `/kuryer/login` →
  "Kuryer kirishi", "Вход для курьера", "Courier sign-in" ✅.
- Barcha sahifalar 200, dev konsolida xato yo'q ✅.

**Eslatma**: bitta skript o'rtada xato bilan to'xtaganda fayl **umuman
yozilmaydi** (yozish oxirida bo'lgani uchun) — `users/[id]` sahifasi shu
sababli yarim tarjimasiz qolib ketgan edi, keyin qayta qilindi. Shuning uchun
tarjimadan keyin qolgan matnlarni grep bilan qayta tekshirish kerak.

---

## 2026-07-29 (7) — Kuryer daromadi va yetkazishlar tarixi ✅

### To'lov usuli — restoran tanlaydi
Har kuryerga alohida qoida (`payoutMode`):
- **`deliveryFee`** (default) — mijoz to'lagan yetkazish narxi to'liq kuryerga;
- **`perOrder`** — har yetkazilgan buyurtma uchun belgilangan summa;
- **`percent`** — yetkazish narxidan foiz.

Olib ketish (pickup) buyurtmasi hech qachon daromad bermaydi. Eski
hujjatlarda `payoutMode` bo'sh — u `deliveryFee` kabi o'qiladi.

### Backend (`internal/handlers/courierstats.go`)
- `courierEarning(order, courier)` — bitta buyurtmadan ishlangani.
- Davrlar: **bugun / oxirgi 7 kun / oxirgi 30 kun / jami**. Vaqt
  `statusHistory` dagi `delivered` yozuvidan olinadi (`updatedAt` emas —
  buyurtma keyin boshqa sabab bilan tegilsa statistika buzilmasin).
- Har davrda: yetkazishlar soni, ishlangan summa, **naqd yig'ilgan**
  (kuryer qo'lidagi pul) va olib borilgan buyurtmalar qiymati.
- Yangi endpointlar: `GET /courier/stats`, `GET /courier/history`,
  `GET /admin/couriers/{id}` (kuryer + statistika + to'liq tarix).

### Admin panel
- Kuryer formasiga **to'lov usuli** tanlagichi (uch kartochka + tegishli
  maydon: summa yoki foiz).
- Yangi **`/admin/couriers/[id]`** sahifasi: profil, to'lov qoidasi, to'rtta
  davr kartochkasi (daromad + yetkazishlar soni + naqd), hozir qo'lidagi
  buyurtmalar soni, va har bir yetkazish — ochilganda **to'liq chek**
  (taomlar, manzil, vaqt jadvali) hamda o'sha yetkazishdan ishlangan summa.
- Ro'yxatdagi kuryer ismi endi kartochkaga havola.

### Kuryer ilovasi
- Ikkita tab: **Buyurtmalar** / **Daromad**.
- Daromad tabida: bugun / 7 kun / 30 kun / jami kartochkalari, qo'lidagi
  naqd pul haqida ogohlantirish, va yetkazishlar tarixi — har qatorda sana,
  manzil, buyurtma summasi va **+ishlangan summa**.
- Statistika faqat tab ochilganda so'raladi (ortiqcha so'rov yo'q).

### Test (alohida `restaurant_etest` bazasi, port 8081)
Har uch usul uchun ikkitadan buyurtma (naqd + onlayn) yetkazilib tekshirildi:
- `deliveryFee` → 32 000 (2 × 16 000) ✅
- `perOrder` 15 000 → 30 000 ✅
- `percent` 50% → 16 000 ✅
- Har holatda **naqd yig'ilgan** faqat naqd buyurtmadan hisoblandi ✅
- Kuryer tarixi (2 ta) va admin kartochkasidagi jami mos keldi ✅
- Olib ketish buyurtmasi → daromad **0** ✅
- Kuryer tokeni bilan admin kartochkasi → **403** ✅
- `go build`/`go vet` ✅, `tsc` ✅, `next lint` ✅.

---

## 2026-07-29 (8) — Dark rejimda kartochkalar + "Yetkazdim" uchun geofence ✅

### 1. Dark rejimda kartochkalar fon bilan qo'shilib ketardi
**Sabab ikkita edi:**
- `--surface` (32 28 26) `--bg` (20 17 15) dan atigi ~12 birlik farq qilardi,
  qorong'i sirt esa `shadow-card` ni yutadi — ya'ni ajratuvchi hech narsa
  qolmagan;
- admin sahifa foni `bg-ink/5` edi — dark rejimda bu **oq 5%** qatlam bo'lib,
  natijada fon `--surface` bilan deyarli bir xil rangga tushardi.

**Tuzatildi:**
- dark: `--bg` 16 14 13 ga tushirildi, `--surface` 39 35 32 ga ko'tarildi;
- yangi **`--line` / `--line-strong`** tokenlari (`border-line`,
  `border-line-strong`, `divide-line`). Ular alpha bilan emas, tayyor rang:
  light'da `rgb(28 25 23 / .09)`, dark'da `rgb(255 255 255 / .14)` — chunki
  qorong'i sirtda 7% chiziq umuman ko'rinmaydi;
- butun `src/` bo'ylab ko'chirildi: `border-ink/[0.07]` va `border-ink/10` →
  `border-line`, `border-ink/15` → `border-line-strong`,
  `divide-ink/[0.07]` → `divide-line` (125 ta joy, 0 ta qoldiq);
- admin sahifa va login foni `bg-ink/5` → `bg-cream`.

Tekshirildi: headless Chrome'da dark rejim skrinshotlari — menyu va savat
sahifalarida kartochkalar aniq ajralib turadi.

### 2. "Yetkazdim" faqat mijoz manzilida
- `delivery.arrivalRadiusM` sozlamasi (default **150 m**, sozlamalarda
  o'zgartiriladi, **0 — tekshiruv o'chiq**).
- Server (`arrivalBlocked`): kuryerning oxirgi joylashuvi manzildan radiusdan
  uzoq bo'lsa yoki **10 daqiqadan eski** bo'lsa — 400 va tushunarli xabar
  (necha metr uzoqligi bilan). Joylashuv umuman yo'q bo'lsa ham bloklanadi.
- Kuryer ilovasi: tugma o'chirilgan holatda turadi, tepasida sabab
  ("Manzilgacha 320 m — yaqinroq boring" / "Joylashuv aniqlanmadi"), manzilda
  bo'lsa yashil "✓ Manzildasiz". Masofa qurilmadagi joriy koordinatadan
  hisoblanadi — lekin bu faqat ko'rsatkich, **qoida serverda**.
- Olib ketish buyurtmalari va koordinatasiz manzillar tekshirilmaydi.
- **Zaxira yo'l ataylab qoldirildi**: admin panelda holatni qo'lda yopish
  mumkin — GPS ishlamay qolsa buyurtma osilib qolmasin.

### Test (alohida `restaurant_gtest` bazasi, port 8081)
- joylashuv yuborilmagan → bloklandi ✅
- 1112 m uzoqda → bloklandi, xabarda masofa ko'rsatildi ✅
- 15 daqiqalik eski joylashuv (manzilda) → bloklandi ✅
- manzilda (~33 m) va yangi → **delivered** ✅
- uzoqda turgan kuryer bloklandi, lekin **admin yopa oldi** ✅
- `arrivalRadiusM = 0` → tekshiruv o'chdi ✅
- pickup buyurtma → tekshirilmadi ✅
- `go build`/`go vet` ✅, `tsc` ✅, `next lint` ✅.

**Eslatma**: jonli bazadagi restoran hujjatida bu maydon yo'q edi (ya'ni 0 =
o'chiq). So'ralgan xatti-harakat ishlashi uchun **150 m** qilib qo'yildi;
sozlamalardan o'zgartirish mumkin.

---

## 2026-07-29 (9) — Buyurtma qidiruvi, logout tugmasi, sayt logotipi ✅

### 1. Buyurtmalarni ID bo'yicha qidirish
Qidiruv aslida raqam bo'yicha ham ishlardi (regex `$options: "i"`), lekin uch
holatda tushib qolardi:
- chekdan **`#` bilan** ko'chirib qo'yilsa (`#MZ04-4091`) — mos kelmasdi;
- chekda ko'rinadigan **Mongo `_id`** umuman qidirilmasdi;
- kiritilgan matn to'g'ridan-to'g'ri regexga ketardi, ya'ni `(` kabi belgi
  natijani buzardi.

Tuzatildi: `#` olib tashlanadi, `regexp.QuoteMeta` bilan belgilar
neytrallanadi, 24-belgili ObjectID bo'lsa `_id` bo'yicha ham qidiriladi,
qo'shimcha `courierName` ham qamrab olindi.

Test: to'liq raqam / kichik harflar / `#` bilan / raqam bo'lagi / Mongo ID /
ism / telefon / manzil → hammasi topdi; `(test` va mos kelmaydigan matn →
0 ta, xatosiz ✅.

### 2. Chiqish tugmasi ko'rinmasdi
Profil va kuryer sahifasida "Chiqish" `<button>` edi, lekin oddiy matn kabi
ko'rinardi — bosiladigan joy ekani bilinmasdi. Endi `btn-ghost` uslubida,
chiqish ikonkasi bilan.

### 3. Saytda logotip ko'rinmasdi
Sozlamalarda yuklangan `logoUrl` **hech qayerda ishlatilmasdi** — header ham,
footer ham restoran nomining bosh harfini ko'rsatardi (cover esa ishlagan,
shuning uchun farq sezilgan).

Yangi `components/site/BrandMark.tsx`: logotip yuklangan bo'lsa rasm, aks
holda avvalgi harf-plitka (yangi deploy'da bo'sh quti chiqmasligi uchun).
Header va footer shunga o'tkazildi.

Brauzerda tekshirildi: header'da yuklangan logotip chiqdi ✅.

**Eslatma**: logotip shaffof PNG bo'lsa qorong'i header'da xira ko'rinishi
mumkin — kontrastli fayl qo'yish tavsiya etiladi.

---

## 2026-07-29 (10) — Sayt matnlari va dizayn tahrirlagichi ✅

### 1. Sayt matnlari sozlamalarda
Telefon/manzil/ijtimoiy tarmoqlar **allaqachon** tahrirlanardi (footer va
"Biz haqimizda" ularni bazadan o'qiydi) — yetishmayotgani sayt *matnlari*
edi. Qo'shildi `restaurant.content`:
- bosh sahifadagi **shior**,
- **"Biz haqimizda"** sarlavhasi va uzun matni (yangi bo'lim sifatida
  sahifada chiqadi),
- **footer matni**.

Har biri uch tilda (`{uz, ru, en}`); bo'sh RU/EN o'zbekchasiga, bo'sh o'zbekcha
esa avvalgi standart matnga tushadi — ya'ni hech narsa yozmasa sayt hozirgidek
qoladi. Sozlamalarda har biri uchun UZ/RU/EN maydonlari bor.

### 2. Dizayn tahrirlagichi 🎨
`/admin/settings` → **"Sayt dizayni"**. Restoran o'zgartira oladi:
- **Asosiy rang** — 10 ta tayyor namuna + rang tanlagich + hex kiritish.
  Qorong'i tema uchun ochiqroq varianti **avtomatik** hisoblanadi (aks holda
  to'q rang qora fonda yo'qoladi); xohlasa qo'lda ham beriladi.
- **Burchaklar yumaloqligi** — 0 dan 28 px gacha slayder.
- **Tugmalar shakli** — dumaloq (pill) yoki kartochkalarga mos.
- **Shriftlar** — Klassik (Inter + Playfair), Zamonaviy (Inter), Yumshoq (Nunito).
- **Jonli ko'rinish** — yorug' va qorong'i variantda, saytdagi aynan o'sha CSS
  o'zgaruvchilari bilan.

**Qanday ishlaydi**: `lib/theme-css.ts` sozlamani CSS o'zgaruvchilariga
aylantiradi, `app/layout.tsx` esa uni `<head>` ichiga **inline** qo'yadi —
birinchi bo'yashdayoq qo'llanadi, standart rang chaqnab o'tmaydi.

**Muhim yechim**: burchaklar bitta joydan boshqariladi — `tailwind.config.ts`
da `rounded-xl/2xl/3xl` CSS o'zgaruvchilariga bog'landi. Shu sababli 400+
komponentga tegmasdan butun saytning shakli o'zgaradi. `rounded-full` chin
pill bo'lib qoladi, tugmalar alohida `--radius-btn` ni ishlatadi.

`getRestaurant` keshi 60 s dan **10 s** ga tushirildi — egasi rangni saqlagach
o'zgarishni deyarli darhol ko'radi.

### Test
- `go build`/`go vet` ✅, `tsc` ✅, `next lint` ✅.
- Jonli bazaga sinov temasi yozildi (ko'k, 8px burchak, kvadratroq tugmalar,
  Inter): sahifa HTML'ida `--brand: 37 99 235`, `--radius-xl: 8px`,
  `--radius-btn: 4px`, `--font-sans: var(--font-inter)` chiqdi; skrinshotda
  butun menyu sahifasi ko'k va burchakli ko'rinishga o'tdi ✅.
- Matnlar: bosh sahifada shior, `/about` da yangi matn bloki, RU tilida
  ruscha variant chiqdi ✅.
- **Test ma'lumotlari keyin bazadan tozalandi** (`content` va `theme` olib
  tashlandi) — sayt standart ko'rinishga qaytdi.

---

## 2026-07-29 (11) — Dizayn boshqaruvlari kengaytirildi + til popup fix ✅

### 1. Til popup chapga kirib ketardi 🐞
`LangSwitch` popup'i doim `right-0` bilan **chapga** o'sardi. Sayt header'ida
bu to'g'ri (tugma o'ngda), lekin admin sidebar'ida tugma chap chekkada —
popup ekrandan chiqib ketardi.

Endi popup ochilishidan oldin (`useLayoutEffect` — bo'yashdan oldin, ya'ni
sakramaydi) joy o'lchanadi va bo'sh tomon tanlanadi. Mobil uchun eni ham
`max-w-[calc(100vw-1.5rem)]` bilan cheklangan.

Tekshirildi: mobil (390px) header'da popup to'liq ko'rinadi; chap chekkaga
ko'chirilgan holatda `left: 12, right: 188` — ekran ichida ✅.

### 2. Dizayn tahrirlagichi kengaytirildi
Yangi boshqaruvlar:
- **Tayyor mavzular** — 5 ta bir bosishli ko'rinish (Issiq/Minimal/Yashil/
  Nafis/Ko'k). Har biri quyidagi sozlamalarning tayyor to'plami, keyin qo'lda
  moslashtirsa bo'ladi.
- **Fon ohangi** — Iliq / Oq / Salqin / Qumrang (light va dark uchun juftlik).
- **Kartochka soyasi** — Yo'q / Yumshoq / Kuchli.
- **Tugma to'ldirilishi** — To'liq rang / Konturli / Yumshoq (och fon).
- **Matn va oraliqlar** — 14–18px root o'lcham (butun sayt zichligi).

Buning uchun yana uchta narsa CSS o'zgaruvchisiga o'tkazildi:
`--shadow-card` / `--shadow-card-hover` (Tailwind `shadow-card` shundan
o'qiydi), `--btn-bg/fg/border` (`.btn-primary` ko'rinishi) va `--root-size`.
Ya'ni yangi sozlamalar ham komponentlarga tegmasdan butun saytga tarqaladi.

### Test
- `go build` ✅, `tsc` ✅, `next lint` ✅, `npm run build` ✅.
- Barcha sahifalar 200, dev konsolida xato yo'q.

**Eslatma**: `fuser -k` port'ni har doim ham bo'shatmaydi — bu safar eski
`next-server` tirik qolib, yangi dev server 3001 ga o'tib ketdi. Port band
bo'lsa `ss -lptn` bilan PID topib `kill -9` qilish kerak.

---

## 2026-07-29 (12) — Tugma bug'i, tashqi yetkazish xizmatlari, SEO ✅

### 1. Kartochkadagi "Qo'shish" tugmasi (radius 0 / Minimal shablon) 🐞
Ikonka-only tugmalar (`px-3.5 py-2` + SVG) `rounded-full` bo'lganda doira
bo'lib chiroyli ko'rinardi; burchak 0 bo'lganda esa o'sha padding katta
kvadrat blokka aylanib, kartochka ichini egallab qolardi.

Yechim: yangi `.btn-icon` (aniq 40×40, padding'siz). Kartochkadagi qo'shish
tugmasi va header'dagi savat tugmasi mobil holatda shu o'lchamda, `sm:` dan
yuqorida esa avvalgidek matn bilan kengayadi. Endi tugma har qanday
burchak radiusida bir xil mutanosib.

### 2. Tashqi yetkazish xizmatlari 🚚
O'z kuryeri yo'q restoranlar uchun. Yangi `delivery_provider` kolleksiyasi va
`/admin/settings` → **"Tashqi yetkazish xizmatlari"**: qo'shish, tahrirlash,
faol/nofaol, o'chirish. Har biri:
- **`link`** — havola, buyurtma ma'lumotlari o'rinbosarlar orqali avtomatik
  qo'yiladi: `{address} {phone} {name} {number} {comment} {lat} {lng} {total}`;
- **`phone`** — dispetcher raqami.

Tayyor namunalar bir bosishda qo'shiladi (Yandex Delivery, Millennium taxi,
eshikdan-eshikkacha).

Buyurtmalar bo'limida har bir yetkazish buyurtmasida **"Yetkazishni
chaqirish"** tugmasi → modal: xizmat tanlanadi, buyurtma ma'lumotlari tayyor
blok sifatida ko'rsatiladi (nusxa olish tugmasi bilan), "Ochish" havolani
to'ldirilgan holda ochadi yoki "Qo'ng'iroq" raqamni teradi. So'ng
"Chaqirildi deb belgilash" — buyurtmaga xizmat nomi, vaqti va (ixtiyoriy)
xizmatdagi raqami yoziladi. Ro'yxatda 🚚 belgisi bilan ko'rinadi.

**Muhim qaror**: tashqi xizmat chaqirilganda **o'z kuryeri bo'shatiladi** —
bitta buyurtmani ikkovi olib ketmasligi uchun.

**Ochiq aytilgan cheklov**: haqiqiy API integratsiyasi (xizmatda avtomatik
zayavka yaratish) har provayder bilan shartnoma va API kalit talab qiladi —
buni hozir qilib bo'lmaydi. Hozirgi yechim har qanday xizmat bilan bugundan
ishlaydi va model (`kind` maydoni) keyinchalik haqiqiy integratsiyani
qo'shishga tayyor.

### 3. Favicon va SEO
- **Favicon endi yuklangan logotip** (`app/icon.svg` — zaxira).
- Sarlavha shabloni: `%s | <restoran nomi>`; bosh sahifada faqat restoran nomi.
- Har bo'lim o'z sarlavhasi bilan: Menyu, Biz haqimizda, Savat, Buyurtma
  berish, Kirish, Profil, `#<buyurtma raqami>`.
- **Taom sahifasi** — nom, tavsif va rasm tanlangan tilda, Open Graph bilan
  (havola ulashilganda taom rasmi chiqadi).
- Open Graph / Twitter card butun sayt uchun (cover rasm bilan).
- **Shaxsiy sahifalar `noindex`**: savat, checkout, profil va buyurtma
  kuzatuvi — ular qidiruvga tushmasligi kerak.
- Client komponent sahifalar uchun sarlavha yonidagi `layout.tsx` da
  (client komponent `generateMetadata` eksport qila olmaydi).

### Test
- `go build`/`go vet` ✅, `tsc` ✅, `next lint` ✅, `npm run build` ✅.
- Xizmatlar (alohida `restaurant_ptest` bazasi): qo'shish ✅, buyurtmaga
  chaqirish + tracking yozildi ✅, kuryerli buyurtmada kuryer bo'shatildi ✅,
  bekor qilish ✅, xizmat o'chirilganda eski buyurtmada nomi saqlanib qoldi ✅.
- SEO (brauzerda): `/` → `Maracanda` + `<link rel="icon">` logotipga,
  `/menu` → `Menyu | Maracanda`, taom → `Osh (palov) | Maracanda` + og:image,
  `/cart` → `noindex, follow` ✅.
- Tugma: mobil 390px da 40×40, kartochkadan chiqmaydi ✅ (skrinshot).

---

## 2026-07-29 (13) — Yandex Delivery bilan haqiqiy API integratsiyasi ✅

### Nima qilindi
`internal/delivery/yandex.go` — Yandex Delivery (Yandex Go B2B "cargo claims")
API mijozi:
- `POST /b2b/cargo/integration/v2/claims/create?request_id=…` — zayavka yaratish
- `…/claims/accept` — tasdiqlash (shundan keyingina kuryer yuboriladi)
- `…/claims/info` — holat, narx, tayinlangan kuryer
- `…/claims/cancel` — bekor qilish (`free` / `paid`)

Auth: `Authorization: Bearer <token>`. Token restoranning **o'z kabinetidan**
olinadi va admin panelda kiritiladi.

### Muhim yechimlar
- **Idempotentlik**: `request_id = "order-<orderId>"`. Tarmoq uzilib operator
  qayta bossa, xizmat o'sha zayavkani qaytaradi — ikkinchi kuryer chaqirilmaydi.
- **Token brauzerga chiqmaydi**: `json:"-"` bilan yashirilgan, o'rniga
  `hasToken` bayrog'i qaytadi. Formadan bo'sh token yuborilsa saqlangani
  o'chmaydi (chunki forma uni hech qachon ko'rmaydi).
- **`create` muvaffaqiyatli, `accept` xato** bo'lsa — zayavka baribir buyurtmaga
  yoziladi, aks holda operator qayta bosib ikkinchi zayavka yaratardi.
- **`apiBaseUrl` sozlanadi** — sandbox yoki test uchun.
- API orqali chaqirilganda **o'z kuryeri bo'shatiladi**.

### Frontend
- Sozlamalarda xizmat turi endi uchta: havola / telefon / **API (avtomatik)**.
  API turida token, tarif va API manzili maydonlari.
- Buyurtmadagi modalda API xizmati tanlansa — nusxa olish shart emas:
  **"Avtomatik chaqirish"** tugmasi. So'ng zayavka raqami, holati, narxi,
  tayinlangan kuryer (ismi va telefoni), kuzatuv havolasi, "Holatni yangilash"
  va "Zayavkani bekor qilish".

### Test — Yandex API kontraktini takrorlaydigan mock bilan
Haqiqiy token bo'lmagani uchun (u restoran shartnomasi bilan beriladi) API'ning
o'zi bilan bir xil yo'llar, auth va JSON shakllarini takrorlaydigan mock server
yozildi va integratsiya **uchdan-uchgacha** sinaldi:
- xizmat qo'shish → `kind: api`, `hasToken: true`, **token javobda yo'q** ✅
- chaqirish → `create` + `accept`, zayavka `accepted`, narx 28 000 ✅
- `sync` → `performer_found`, kuryer "Ravshan K. +998901112233", kuzatuv
  havolasi ✅
- **idempotentlik** — qayta chaqirilganda claim o'zgarmadi ✅
- bekor qilish → zayavka bekor, buyurtmadan tozalandi ✅
- noto'g'ri token → xizmatning **o'z xabari** ko'rsatildi ("Token noto'g'ri") ✅
- tokensiz xizmat → "API token kiritilmagan" ✅
- olib ketish buyurtmasi → "yetkazish manzili yo'q" ✅
- bo'sh token bilan saqlash → saqlangan token o'chmadi ✅
- Yandex'ga ketgan so'rov tekshirildi: to'g'ri path, `request_id`, Bearer
  auth, `taxi_class: express`, `source`/`destination` nuqtalari
  koordinatalari va kontaktlari, tarkibi "Osh (palov) × 3", qiymati 135000 UZS ✅

### ⚠️ Prod'ga chiqishdan oldin
1. Restoranning Yandex Delivery shartnomasi va **haqiqiy tokeni** kerak —
   usiz ishlab chiqarish serveriga umuman ulanib bo'lmaydi.
2. Endpoint tafsilotlari (yo'llar, maydon nomlari) Yandex hujjatining
   **joriy versiyasi** bilan solishtirilishi shart — API vaqt o'tib
   o'zgarishi mumkin. `apiBaseUrl` shu sababli sozlanadigan qilingan.
3. **Millennium taxi va boshqa mahalliy xizmatlarda ommaviy API yo'q** —
   ular telefon/havola rejimida ishlaydi. Agar shartnoma bo'yicha API
   hujjati berilsa, `clientFor` ga yangi `apiProvider` qo'shiladi.

---

## 2026-07-30 (14) — Web orqali yetkazish chaqirish mukammallashtirildi ✅

**Muammo**: API integratsiyasi (13-kun) restoranda **Yandex biznes-akkaunt**
talab qiladi. Ko'p kichik restoranda u yo'q — ularga oddiy web orqali chaqirish
qulayroq. Shuning uchun web yo'li to'liq ishlangan, API esa **tanlov** sifatida
o'z joyida qoldi.

### 1. Chaqirish oynasi 3 qadamga bo'lindi (`CallDeliveryModal.tsx`)
1. **Xizmatga o'tish** — havola buyurtma ma'lumotlari bilan to'ldirilib
   ochiladi; **"Ochish va chaqirildi deb belgilash"** — bitta bosishda ham
   ochadi, ham buyurtmaga yozadi. Yonida havola ko'rinishi (`details`) va
   to'lmagan o'rinbosarlar haqida ogohlantirish.
2. **Formaga qo'yiladigan ma'lumotlar** — har bir maydon (olib ketish manzili,
   restoran telefoni, mijoz manzili, izoh, koordinata, ism, telefon, taomlar,
   naqd olinadigan summa) **alohida "Nusxalash" tugmasi** bilan. Havolada
   parametr qabul qilmaydigan xizmatlar (Yandex Delivery sayti) formasi shu
   bilan tez to'ldiriladi. "Hammasini nusxalash" ham bor.
3. **Xizmat javobi** — xizmatdagi raqam, **xizmat narxi** (yangi maydon) va
   izoh saqlanadi.

### 2. O'rinbosarlar kengaytirildi (`lib/providerLink.ts`)
Restoran tomoni qo'shildi: `{pickupName}`, `{pickupAddress}`, `{pickupPhone}`,
`{pickupLat}`, `{pickupLng}`; buyurtma tomoni: `{items}`, `{itemsCount}`,
`{subtotal}`, `{deliveryFee}`, `{payment}`. Sozlamalarda o'rinbosarlar
**bosiladigan chipslar** — bosilsa havolaga qo'shiladi.

### 3. Tayyor namunalar (`ProvidersEditor.tsx`)
- **Yandex Go — kuryer** va **Yandex Go — express**: haqiqiy deeplink
  (`3.redirect.appmetrica.yandex.com/route?start-lat=...&end-lat=...&tariffClass=courier`)
  — Yandex Go ilovasi/sayti **ikkala manzil tayyor** holda ochiladi. Biznes
  akkaunt kerak emas.
- **Yandex Delivery (sayt)** — `dostavka.yandex.ru/call-courier/`, maydonlar
  nusxalash orqali.
- Millennium taxi (telefon) va Yandex Delivery API (biznes-akkaunt bilan).

### 4. Backend
- `ExternalDelivery.Cost` (`cost`) — xizmat narxi qo'lda yoziladi.
- Yozuvni keyin tahrirlash (`raqam`/`narx` qo'shish) **`calledAt` ni
  o'zgartirmaydi** — "qachon topshirildi" savoli javobsiz qolmasin.

### 5. Chek (`OrderReceipt.tsx`)
Buyurtma chekida "Tashqi yetkazish xizmatlari" bloki: kim, qachon chaqirilgan,
xizmatdagi raqam, narx, holat, izoh.

**Cheklov**: Yandex Go deeplink faqat **koordinata va tarif** qabul qiladi
(rasmiy hujjat: yandex.com/support/taxi-distr/en/api/deeplinks) — mijoz ismi,
telefoni, izohi havolada yuborilmaydi, ular nusxalash orqali qo'yiladi. Bu
Yandex cheklovi, kod kamchiligi emas.

---

## 2026-07-30 (15) — Taomga izoh, bekor qilish sababi, marshrut tugmalari ✅

### 1. Har bir taomga izoh 📝
- Savatda har qator ostida izoh maydoni ("piyozsiz", "achchiq qilmang").
  `CartLine.comment` → `localStorage` (`cart_v2`) → buyurtma `items[].comment`.
- Izoh **qator kimligiga kirmaydi**: bir xil taom+variant baribir bitta qator
  bo'lib qoladi, izoh o'sha qatorning hammasiga tegishli.
- Serverda `clampText` bilan trim + 200 belgi (mijoz matniga ishonmaymiz).
- Ko'rinadigan joylar: checkout xulosasi, **admin cheki** (sariq belgi bilan —
  oshxona o'tkazib yubormasin), **kuryer ilovasi** taom ro'yxati.

### 2. Bekor qilish sababi majburiy ❌
- Admin panelda `confirm()` o'rniga **`CancelOrderModal`**: 4 ta tayyor sabab
  (mijoz bekor qildi / taom qolmagan / hudud tashqarisida / aloqa yo'q) +
  erkin matn. Sababsiz tugma ishlamaydi.
- Holat select'idan `cancelled` tanlansa ham shu modal ochiladi —
  `changeStatus` sababsiz bekor qilishni o'zi to'sadi.
- `order.cancelReason` (300 belgigacha). Mijoz uni `/order/{number}` da va
  profil tarixida ko'radi. Buyurtma qayta tiklansa (`confirmed` va h.k.)
  sabab `$unset` qilinadi — eski sabab osilib qolmaydi.
- Tekshirildi (alohida test bazasida): bekor → sabab track javobida;
  qayta tiklash → sabab yo'q.

### 3. Marshrut tugmalari 🗺️
- Yangi `components/map/RouteButtons.tsx`: **Yandex Maps / Google Maps / 2GIS**.
  Telefonda o'rnatilgan ilova ochiladi, kompyuterda sayt. SDK ham, API key ham
  kerak emas.
  - Yandex: `yandex.uz/maps/?rtext=~{lat},{lng}&rtt=auto`
  - Google: `google.com/maps/dir/?api=1&destination={lat},{lng}`
  - 2GIS: `2gis.uz/routeSearch/rsType/car/to/{lng},{lat}` (e'tibor: lng,lat!)
- Boshlanish nuqtasi ataylab bo'sh qoldirilgan — har ilova foydalanuvchi
  joylashuvini bizdan yaxshiroq biladi.
- Joylashgan joyi: checkout'da "Olib ketish" xaritasi ostida va buyurtma
  kuzatuvi sahifasida (olib ketish buyurtmalari uchun).

---

## 2026-07-30 (16) — 2GIS/Yandex havolalari, panel adminlari, amallar jurnali ✅

### 1. 2GIS marshruti tuzilmayotgani tuzatildi 🗺️
Eski `2gis.uz/routeSearch/rsType/car/to/{lng},{lat}` yo'li **o'lgan** — `curl`
bilan tekshirilganda **301 → `2gis.uz/tashkent`** (shuning uchun shahar
sahifasiga tashlab yuborardi). Ishlaydigan format:
`2gis.uz/directions/points/%7C{lng}%2C{lat}` (302 bilan kanonik marshrut
URL'iga o'tadi). `|` percent-encoded bo'lishi shart. Yandex Maps va Google
havolalari tekshirildi — ikkisi ham 200.

### 2. Yandex Go — taxi emas, **Dostavka** ochiladigan bo'ldi 🚚
Internetdan aniqlangani: Yandex o'zining `dostavka.yandex.ru` saytida telefon
uchun **`https://yandex.go.link/route?tariffClass=express_d2d&adj_t=pucm71r&
adj_campaign=web_ru_delivery&trap_mode=true`** havolasini beradi. Sinovda bu
havola `yandextaxi://route?tariffClass=express_d2d&start-lat=…&end-lat=…`
deeplinkiga aylandi — ya'ni ilova **Dostavka** bo'limida, ikkala manzil tayyor
holda ochiladi. `adj_t` (Adjust tokeni) bo'lmasa havola **404** qaytaradi.
- Namunalar yangilandi: "Yandex Go — Dostavka (kuryer)" (`express_d2d`) va
  "Yandex Go — Cargo" (`cargo`). Eski appmetrica/taxi namunasi olib tashlandi.
- **O'zbekistondagi haqiqat**: jismoniy shaxs Yandex kuryerini faqat **Yandex Go
  ilovasi** orqali chaqiradi; `delivery.yandex.uz` — yuridik shaxslar uchun
  (bizda bu API yo'li). Shuning uchun kompyuterda ishlayotgan administrator
  uchun chaqirish oynasiga **QR kod** qo'shildi
  (`components/admin/QrCode.tsx`, `qrcode-generator`, inline SVG — tashqi
  so'rov yo'q): telefon bilan skaner qilinadi va ilova to'ldirilgan holda
  ochiladi.

⚠️ Bazada eski "Yandex Go — kuryer" xizmati saqlanib qolgan bo'lsa, uni
o'chirib namunadan yangisini qo'shish kerak (havola yangilangan).

### 3. Panel adminlari `/admin/admins` (faqat owner) 👥
- Yangi admin — **saytda telefon + SMS bilan kirgan mavjud mijoz**. Owner uni
  ro'yxatdan qidirib topadi (`/admin/users` qidiruvi), login va vaqtinchalik
  parol beradi, rol tanlaydi (owner/manager).
- Hisob `mustChangePassword: true` bilan yaratiladi → birinchi kirishda panel
  uni `/admin/account` ga majburan yuboradi (bu mexanizm avvaldan bor edi).
  "Parolni tiklash" ham shu bayroqni qayta qo'yadi.
- Backend: `AdminUser.userId/name/phone/createdBy/lastLoginAt`,
  `handlers/adminaccounts.go`. Qoidalar: bitta mijoz — bitta hisob, login band
  bo'lsa 409, o'z hisobini o'chirish va **oxirgi owner**ni o'chirish/rolini
  o'zgartirish taqiq.
- Marshrutlar `RequireRole(..., "owner")` guruhida — menejer o'ziga huquq
  qo'sha olmaydi.

### 4. Amallar jurnali `/admin/logs` (faqat owner) 📜
- `admin_log` kolleksiyasi + `handlers/auditlog.go` (`h.logAction`).
  Yoziladigan amallar: panelga kirish, buyurtma holati, **bekor qilish (sababi
  bilan)**, kuryer biriktirish, tashqi xizmat chaqirish, kuryer CRUD, admin
  create/update/delete, o'z login/parolini o'zgartirish, taom/kategoriya CRUD,
  yetkazish xizmatlari CRUD, sozlamalarni saqlash.
- `action` — barqaror id (`order.cancel`), matn panelda tarjima qilinadi
  (uz/ru/en). Guruh bo'yicha filtr (`?action=order`), admin bo'yicha filtr,
  `?before=` bilan "yana yuklash".
- Log yozilmasa ham amal bekor qilinmaydi (`logAction` xatoni yutadi).

### Tekshirildi (alohida test bazasida, keyin o'chirildi)
- Mijozdan admin yaratish ✓, takror urinish 409 ✓, mavjud bo'lmagan mijoz 404 ✓
- Yangi admin login → `mustChangePassword: true` ✓, parol o'zgartirish ✓
- Menejer `/admin/accounts` va `/admin/logs` ga **403** ✓
- O'z hisobini o'chirish taqiqi ✓, ownerni o'chirish ✓
- Jurnal yozuvlari: login, order.status, order.cancel (sabab), courier.create/
  delete, provider.create/delete, admin.create/delete/credentials ✓
- Hech kirmagan hisobda `lastLoginAt` JSON'da **yo'q** (avval 0001-yil chiqardi)

---

## 2026-07-30 (17) — Yandex Go tarifi (Dostavka o'rniga taxi ochilgani) + mobil layout ✅

### 1. QR kod Yandex Go'ning **taxi** bo'limini ochardi
Sabab: havoladagi `tariffClass` ilovada qaysi bo'lim ochilishini belgilaydi, va
tarif nomlari mamlakat bo'yicha farq qiladi. Bizda Rossiyaning
`express_d2d` (eshikdan-eshikka) turardi — O'zbekistonda bu tarif sotilmaydi,
shuning uchun ilova jim turib **oddiy taxi buyurtmasiga** qaytardi (ikkala
manzil bilan). Deeplink formati o'zi to'g'ri: `dostavka.yandex.ru` hozir ham
`yandex.go.link/route?tariffClass=…&adj_t=…&trap_mode=true` beradi (tekshirildi).

- Namuna havola default tarifi → **`courier`** (piyoda kuryer, UZ).
- `lib/providerLink.ts`: `isYandexGoLink` / `yandexGoTariff` /
  `withYandexGoTariff` + `YANDEX_GO_TARIFFS` (courier, express, cargo,
  express_d2d).
- Tarif almashtirgich vaqtincha ikki joyga qo'yildi (sozlamalar + QR yoni), chunki
  qaysi tarif shaharda sotilishini faqat o'sha shahardagi telefon aytadi.

### 1b. Tarif aniqlandi → almashtirgich olib tashlandi
Telefonda tekshirildi: **`express`** (avtomobil kuryer) to'g'ri ishlaydi — QR
skaner qilinganda ilova aynan **Dostavka** bo'limida, ikkala manzil bilan
ochiladi. Shu sababli:
- Namuna bitta qoldi: **"Yandex Go — Dostavka"** (`tariffClass=express`).
  `courier` / `cargo` / `express_d2d` namunalari va tarif tugmalari o'chirildi
  (`YANDEX_GO_TARIFFS` va yordamchilari, `goTariff*` lug'at kalitlari ham).
- **Eslatma**: bazada allaqachon saqlangan xizmat havolasi eski tarif bilan
  qolgan bo'lishi mumkin — sozlamalardagi havola maydonini `express` ga
  tuzatish yoki xizmatni o'chirib namuna chipsini qayta bosish kifoya.

### 1c. Ilovaga hamma ma'lumotni avtomatik yozish — deeplink bilan **imkonsiz**
Yandex Go deeplink'i faqat quyidagilarni oladi (hujjatlashtirilgan):
`start-lat/lon`, `end-lat/lon`, `tariffClass`, `ref`, `lang`, AppMetrica id'lari.
Mijozning **ismi, telefoni, manzil matni, izohi uchun parametr yo'q** — ular
URL orqali uzatilmaydi. Shuning uchun chaqirish oynasidagi "2-qadam" o'z
kuchida qoladi: har bir maydon alohida "Nusxalash" tugmasi bilan (+ "Hammasini
nusxalash" — ilovadagi izoh maydoniga bitta qo'yish uchun).
To'liq avtomatik zayavka faqat **B2B API** (`kind: "api"`, Yandex Delivery
biznes-akkaunt + token) orqali: server o'zi ism, telefon, manzil va izohni
yuboradi (`internal/delivery/yandex.go`) — bu yo'l loyihada allaqachon bor.

### 2. Mobil layout — haqiqiy o'lchov bilan
Playwright + chromium bilan 360px va 390px ekranda har bir sahifa o'lchandi
(gorizontal surilish, ekrandan chiqib ketgan elementlar, kichik tugmalar):

- **Checkout yon tomonga surilardi** (360px ekranda sahifa 420px) — asosiy
  xato. Sababi: `form.grid ... lg:grid-cols-[1fr_340px]` da mobil uchun
  ustun berilmagan → ustun `auto` bo'lib, ichidagi `truncate` (nowrap) manzil
  matnining **min-content** kengligiga cho'zilgan. Endi hamma joyda base
  `grid-cols-1` va `lg:grid-cols-[minmax(0,1fr)_…]` (checkout, savat, bosh
  sahifa, taom sahifasi, about, footer).
- **Savat qatori** telefonda sig'masdi (rasm + nom + hisoblagich + summa bitta
  qatorda ≈ 400px): hisoblagich va summa mobil ekranda ikkinchi qatorga tushdi,
  tugmalar kattalashdi.
- Uzun restoran nomi: header'da `truncate` + `min-w-0`, hero sarlavhasi
  telefonda `text-4xl` (48px → 36px), about `text-3xl`; kuryer ilovasida
  kuryer nomi ham `truncate`.
- Buyurtma kuzatuvi: kartochka `p-6 sm:p-8`, sarlavha qatori `flex-wrap`.
- Dark rejimda oq qolib ketadigan ranglar (`bg-amber-50`, `bg-rose-50`,
  `bg-emerald-100`) `dark:` juftini oldi.
- **Buzuq savat butun saytni yiqitardi**: `cart_v2` dagi kutilmagan shakl
  (eski build yoki yarim yozilgan qiymat) to'g'ridan-to'g'ri reducer'ga tushib
  `state.lines.reduce` xatosi bilan **har bir sahifani** o'ldirardi (savatni
  emas — `CartProvider` layout'da). Endi `parseStored()` faqat haqiqiy
  qatorlarni oladi, qolgani tashlanadi.

Tekshirildi: 360/390px da bosh sahifa, menyu, taom, savat, checkout (xarita +
zonalar bilan), profil, buyurtma kuzatuvi, login, kuryer PWA va kuryer login —
hech birida gorizontal surilish yo'q; uzun restoran nomi bilan ham. Admin panel
ataylab tekshirilmadi (kompyuter uchun).

---

## 2026-07-30 (18) — Buyurtma manzili xaritada + adminda tuzatish ✅

Admin panel → Buyurtmalar → chekdagi manzil ostida **"Xaritadagi joy"**: mijoz
belgilagan nuqta xaritada ko'rinadi (zonalar ham chizilgan), va **nuqtani
bosib ko'chirish** mumkin — mijoz adashib belgilagan bo'lsa admin to'g'rilaydi.

- **Backend**: `PUT /admin/orders/{id}/address` (`AdminUpdateOrderAddress`).
  Faqat `delivery` buyurtmalarda, koordinata chegaralari tekshiriladi.
  **Pul ataylab tegilmaydi** — mijoz bilan kelishilgan summani orqavorotdan
  qayta hisoblash noto'g'ri bo'lardi; zona/masofa (tavsifiy maydonlar)
  yangilanadi va yangi nuqtaning hozirgi narxi javobda qaytadi. Jurnalga
  `order.address` yoziladi.
- **Frontend**: `components/admin/OrderAddressMap.tsx` — `AddressMap` ustida
  qurilgan, nuqta ko'chirilganda manzil matni `reverseGeocode` bilan
  yangilanadi, "Saqlash / Qaytarish" faqat o'zgarish bo'lsa chiqadi.
  `OrderReceipt` ga `editableAddress` proplari: `/admin/orders` — tahrirlanadi,
  `/admin/users/[id]` va kuryer kartochkasi — faqat ko'rish.
- Nima uchun muhim: kuryer "Yetkazdim"ni nuqtaga `arrivalRadiusM` metr yaqin
  turib bosadi — noto'g'ri metka buyurtmani yopolmaslikka olib keladi.
- Lug'atga `receipt.address*` kalitlari va jurnal yorlig'i (uz/ru/en).

**Tekshirildi** (Playwright bilan admin panelda, haqiqiy buyurtmada):
xarita chizildi ✓, nuqta bosilganda ko'chdi ✓, manzil matni geokodlandi ✓,
saqlangach bazada `address.lat/lng` o'zgardi ✓, `total`/`deliveryFee` o'zgarmadi
✓, `deliveryZone` qayta hisoblandi ✓, jurnalda `order.address` ✓.
Test uchun yaratilgan vaqtinchalik admin (`qa_owner`), mijoz va buyurtmalar
o'chirildi — bazada faqat `yujo` qoldi.

---

## 2026-07-30 (19) — Admin ro'yxatlari chegaralandi + to'liq dashboard statistikasi ✅

### 1. Har bir ro'yxat o'z blokida suriladi, kerak bo'lsa sahifalanadi
Muammo: ma'lumot ko'paygani sari sahifa cho'zilib ketardi (filtrlar va qidiruv
ekrandan chiqib ketardi).

- Yangi umumiy modul `components/admin/PagedList.tsx`: `usePaged()`,
  `<ListScroll>` (`max-h` + `overflow-y-auto`), `<Pager>` ("13–15 / 15",
  Oldingi/Keyingi). Filtr o'zgarsa sahifa 1 ga qaytadi, qatorlar kamaysa
  sahifa raqami chegaraga tushadi.
- Qo'llandi: buyurtmalar (12/sahifa), foydalanuvchilar (25), kuryerlar (20),
  mijoz kartochkasi — buyurtmalar (10) va saqlangan manzillar, kuryer
  kartochkasi — tarix (10), adminlar, amallar jurnali (o'zining serverdagi
  "ko'proq yuklash"i bilan birga), menyu (har kategoriya bloki alohida),
  kategoriyalar, tashqi yetkazish xizmatlari, dashboardning so'nggi
  buyurtmalari va top taomlari.
- Jadval sarlavhalari `sticky top-0 bg-surface` — surilganda ustun nomlari
  joyida qoladi.
- `MenuRow` `<li>` dan `<div>` ga o'tdi (endi `<ul>` ichida emas).

### 2. Dashboard: davr tanlanadigan to'liq statistika
- **Backend** `GET /admin/stats?from=&to=` (`internal/handlers/adminstats.go`).
  Hisob bazada bajariladi — ilgari panel oxirgi 200 buyurtmani olib brauzerda
  qo'shardi (200 dan oshgach son noto'g'ri bo'lardi va "nechta mijoz bor?"
  degan savolga umuman javob yo'q edi).
- **Davr**: Bugun / 7 kun / 30 kun / Hammasi / **Oraliq** (ikki sana tanlanadi;
  `to` kiritilgan kun bilan hisoblanadi). Noto'g'ri sana → 400.
- **Ko'rsatkichlar**: buyurtmalar (jami, yetkazilgan, yetkazish, olib ketish,
  bekor), pul (tushum, o'rtacha buyurtma, yetkazish yig'imi, naqd),
  odamlar (ro'yxatdagi mijozlar, yangi mijozlar, **faol mijozlar** — shu davrda
  buyurtma berganlar, manzili saqlanganlar, kuryerlar + ishdagilar, panel
  adminlari owner/manager kesimida), menyu (taomlar, mavjud, kategoriyalar),
  holatlar kesimi (foiz chizig'i bilan) va **top 8 ko'p sotilgan taom**.
- Bekor qilingan buyurtmalar tushumga qo'shilmaydi — sahifada izoh bor.

**Tekshirildi** (Playwright, admin panelda): dashboard 18 kartochka bilan
chizildi ✓, davr tugmalari va oraliq sanalari ishlaydi ✓ (`bugun` 4 buyurtma,
`hammasi` 15) ✓, noto'g'ri sana 400 ✓; buyurtmalar ro'yxati "1–12 / 15" →
"Keyingi" → "13–15 / 15" ✓, sahifa balandligi 1017px da qoldi (ilgari butun
ro'yxat bo'yiga cho'zilardi) ✓; foydalanuvchilar va menyu sahifalari ham
chegaralangan ✓. Test uchun yaratilgan `qa_owner` admin o'chirildi.

---

## 2026-07-30 (20) — Stol bron tizimi + ovozli bildirishnoma + jurnal qidiruvi ✅

### 1. Amallar jurnalida qidiruv
`GET /admin/logs?q=` — buyurtma raqami (`#AL23-9004`), bazadagi ID, admin ismi,
nishon nomi yoki izoh bo'yicha. Panelda qidiruv maydoni (250ms debounce).
Ya'ni "shu buyurtmaga kim nima qilgan?" degan savolga bir qidiruvda javob.

### 2. Bron tizimi (yangi katta bo'lim)
**Xaritani admin o'zi chizadi** (`/admin/settings` → "Stol bron qilish"):
asbob tanlanadi (to'rtburchak stol / dumaloq stol / devor / zona) va xaritada
sudrab chiziladi; "Ko'chirish" rejimida stollar sudraladi, stol bosilsa raqami,
joylar soni, o'lchami va faolligi tahrirlanadi. Koordinatalar plan birligida —
bitta chizma telefonda ham, katta ekranda ham to'g'ri ko'rinadi.

**Mijoz** `/bron` sahifasida sana, vaqt va mehmonlar sonini tanlaydi; xaritada
bo'sh stollar yashil, band stollar qizil, ishlatilmaydiganlari kulrang bo'lib
turadi. Stolni bosadi, ism (+ izoh) yozadi. **Bron uchun telefon raqam SMS kod
bilan tasdiqlangan bo'lishi shart** (buyurtmadagi bilan bir xil oqim).

**Bandlikni server hal qiladi**: `[at, endsAt)` oraliqlari kesishsa **409**
qaytadi. Bu shart edi — ikki mijoz bir vaqtda bir stolni bosishi mumkin, sayt
esa faqat oxirgi so'rovdagi holatni ko'rsatadi. Bekor qilingan bron stolni
darhol bo'shatadi.

**Panelda** `/admin/reservations`: kelayotgan / bugun / o'tgan / hammasi,
qidiruv, holat oqimi (Tasdiqlash → Mehmon keldi → Yakunlash; Bekor qilish
sababi bilan) va o'ng tomonda tanlangan payt uchun xarita.

Yangi: `restaurant.booking` (xarita + qoidalar), `reservation` kolleksiyasi,
`GET /booking/plan`, `POST /reservations`, `GET /reservations/{number}`,
`GET /users/me/reservations`, `GET|POST /admin/reservations`,
`PUT /admin/reservations/{id}/status`, `DELETE /admin/reservations/{id}`,
`middleware.OptionalAuth` (token bo'lsa oladi, bo'lmasa o'tkazadi).

### 3. Ovozli bildirishnoma (admin)
`components/admin/AlertBell.tsx` — panelning har sahifasida. Har 15 soniyada
`GET /admin/alerts` (juda yengil: sanoq + oxirgi vaqt) so'raladi; yangi
buyurtma yoki bron kelsa **ovoz** chalinadi va banner chiqadi.
- Ovoz **WebAudio bilan sintez qilinadi** — audio fayl yo'q, mijoz VPS'ida
  404 bo'lishi mumkin emas.
- Brauzer ovozni foydalanuvchi bosmaguncha bermaydi, shuning uchun yon panelda
  🔔 tugmasi bor: bir marta bosiladi va tanlov localStorage'da qoladi.

**Tekshirildi** (server API + Playwright):
xarita saqlandi ✓ · bo'sh stolga bron 201 ✓ · kesishuvchi vaqt **409** ✓ ·
slot tugagach o'sha stol yana bo'sh ✓ · nofaol stol / ko'p mehmon / o'tgan
vaqt / juda uzoq sana / qisqa telefon → 400 (aniq xabar bilan) ✓ ·
`/booking/plan` band stolni ko'rsatadi, bekor qilingach bo'shaydi ✓ ·
saytda (390px telefon) mijoz stol tanlab bron qildi ✓ · panel o'sha bronni
ko'rsatdi va "Tasdiqlash" ishladi ✓ · plan muharririda sudrab yangi stol
chizildi va saqlandi (bazada 4 ta stol) ✓ · jurnal qidiruvi bron raqami
bo'yicha 2 ta yozuvni topdi ✓ · ovoz tugmasi panelda ✓

**Eslatma**: sinov uchun chizilgan **namuna xarita** (4 stol) bazada qoldi va
bron yoqilgan — Sozlamalardan o'chirib o'zingiznikini chizishingiz mumkin.
Test bronlari va vaqtinchalik `qa_owner` admin o'chirildi.

---

## 2026-07-30 (21) — Bron uchun SMS tasdiq ✅

Bron endi **tasdiqlangan raqam** talab qiladi — buyurtma bilan bir xil qoida.
Sabab: javob bermaydigan raqamga saqlangan stol restoranga butun bir kechani
yo'qotadi.

- `POST /reservations` mijoz JWT'si talab qilinadigan guruhga ko'chirildi
  (`RequireRole(..., "user")`). Token yo'q/yaroqsiz → **401**.
- Raqam **profildagi tasdiqlangan raqamdan** olinadi: mijoz formaga boshqa
  raqam yozsa ham serverda e'tiborga olinmaydi (tekshirildi: formada
  `+998900000000` yozilgan bo'lsa ham bazaga `998905550011` yozildi).
  Ism bo'sh bo'lsa profildagi ismdan olinadi.
- Saytda: kirmagan mijoz ogohlantirish ko'radi va tugma "Kirib, bron qilish"
  bo'lib `/login?next=/bron&reason=booking` ga yuboradi; login sahifasida
  sabab matni chiqadi. Kirgach telefon maydoni tasdiqlangan raqam bilan
  to'ldirilib bloklanadi ("SMS bilan tasdiqlangan raqamingiz").
- **Panel orqali bron bundan mustasno** — telefon qo'ng'irog'ida operator
  raqamni o'zi yozadi (`POST /admin/reservations` avvalgidek ishlaydi).
- Endi ishlatilmayotgan `middleware.OptionalAuth` o'chirildi.

**Tekshirildi**: tokensiz 401 ✓ · yaroqsiz token 401 ✓ · SMS login → 201 va
bazada tasdiqlangan raqam ✓ · bron profil tarixida ko'rinadi ✓ · panel orqali
bron 201 (operator yozgan raqam bilan) ✓ · saytda kirmagan holatda ogohlantirish
va login'ga yo'naltirish ✓ · kirgach telefon bloklangan holda to'ldirildi va
bron o'tdi ✓

---

## 2026-07-30 (22) — Bronlar tarixi mijoz profilida ✅

`/profile` sahifasida buyurtmalar tarixidan keyin **"Mening bronlarim"** bo'limi:
bron raqami, holat yorlig'i (Kutilmoqda / Tasdiqlandi / Mehmon keldi /
Yakunlandi / Bekor qilindi), stol raqami, sana-vaqt, mehmonlar soni va izoh.
Bekor qilingan bronda **restoranning sababi** ko'rinadi — buyurtmadagi bilan
bir xil qoida, ya'ni mijoz "nega bekor qilindi?" deb qo'ng'iroq qilmaydi.

`GET /users/me/reservations` allaqachon bor edi; sahifa shuni o'qiydi.
Lug'atga `profile.noBookings` va `booking.guestsCount` qo'shildi (uz/ru/en).

**Tekshirildi** (Playwright, 390px telefon): bo'lim sarlavhasi ✓ · bron raqami ✓ ·
holat yorlig'i ✓ · stol yorlig'i ✓ · "2026-07-31 20:00 · 2 mehmon" qatori ✓ ·
admin bekor qilgach sabab profilda ko'rindi ✓ · gorizontal surilish yo'q ✓

---

## 2026-07-30 (23) — Ovozli bildirishnoma: bron uchun alohida ovoz + ikki xato ✅

Bronlar uchun ovoz (20-yozuvda) allaqachon ishlar edi — sinov buni tasdiqladi
(bron kelganda nota chalindi, banner chiqdi). Muammo **sezilmasligida** edi:
ovoz standart holatda o'chiq va tugma yon panel pastida turardi.

- **Ovoz endi standart yoqilgan**. Brauzer sahifa bilan muloqotdan oldin ovoz
  bermaydi, shuning uchun paneldagi **birinchi bosish** (istalgan joyda)
  audio'ni jimgina ochadi. Tugma faqat o'chirish uchun qoladi.
- **Bron va buyurtma har xil jaranglaydi**: buyurtma — ikki nota (880/1175),
  bron — uch nota (660/880/1320). Zaldagi odam ekranga qaramay farqlaydi.
- Tugma **telefondagi yuqori panelga** ham qo'shildi (u yerda yon panel yashirin).

**Yo'l-yo'lakay ikki xato tuzatildi:**
1. Tugmani ikkinchi joyga qo'yish `AlertBell` ni ikki marta ulagan edi →
   har xabar **ikki marta** chalinardi (sinovda 6 ta nota ko'rindi). Endi
   komponent ikkiga bo'lindi: `AlertBell` (kuzatuvchi, layout'da bir marta) va
   `SoundToggle` (faqat tugma, xohlagancha nusxada). Ular
   `admin-sound-change` hodisasi orqali sinxronlanadi.
2. Ovoz o'chirilgan holatdan yoqilganda "namuna" ovoz chalinmay qolgan edi —
   endi kuzatuvchi hodisadagi `preview` bayrog'ini eshitib chaladi.

**Tekshirildi** (Playwright, AudioContext ustidan josuslik bilan):
standart holat "Ovoz yoqilgan" ✓ · bron → aynan 3 nota (660/880/1320), bir
marta ✓ · buyurtma → aynan 2 nota (880/1175) ✓ · bannerlar ✓ · o'chirilganda
nota yo'q ✓ · qayta yoqilganda namuna ovoz ✓ · tanlov localStorage'da ✓ ·
telefonda tugma ko'rinadi ✓ · panelda tugma bitta (ikkilanmagan) ✓

---

## 2026-07-30 (24) — QR menyu: stol QR kodlari + "stolga" buyurtma ✅

### 1. QR kartochka generatori (`/admin/qr`)
- Umumiy (restoran) yoki **har bir stol uchun alohida** QR.
- Kartochkada: orqa fon (4 ta ohang), sarlavha, kichik sarlavha, tavsif,
  pastki qator (telefon) va stol kartochkasida **katta stol raqami**.
- Kartochka **canvas'da 1200×1700** chiziladi (`components/admin/QrPoster.tsx`)
  — ekrandagi ko'rinish aynan yuklanadigan PNG. "Hamma stollar" tugmasi har
  stolga alohida fayl beradi. QR oq plastinka ustida — qora fonda ham skaner
  bo'ladi.
- "Sayt manzili" maydoni: panel IP yoki tunnel orqali ochilgan bo'lsa ham QR
  haqiqiy domenga ishora qilishi uchun.

### 2. QR → mijoz o'sha stolda
- `lib/table.tsx`: `?table=<id>` ko'rilsa stol **sessionStorage** ga yoziladi
  (localStorage emas — stol bitta tashrifga tegishli; uyda yetkazib berish
  buyurtma qilinganda eski stol yopishib qolmasligi kerak). Stol raqami
  xaritadan tekshirib olinadi.
- Sayt tepasida "N-stoldasiz · buyurtma shu stolga keltiriladi" chizig'i va
  "Men stolda emasman" tugmasi.

### 3. `dinein` buyurtma turi
- Checkout'da uchinchi variant **"Stolga"** — faqat QR skaner qilinganda
  ko'rinadi va avtomatik tanlanadi. Manzil ham, yetkazish narxi ham yo'q.
- Server `tableId` ni xaritadagi stollar ro'yxatiga solishtiradi: noto'g'ri
  yoki eskirgan QR → 400 ("stol topilmadi — QR kodni qayta skaner qiling").
  Buyurtmaga `tableNumber` nusxa bo'lib yoziladi.
- Holat oqimi `pickup` kabi **"Yo'lda"ni o'tkazib yuboradi**; chekda va
  ro'yxatlarda "1-stol (QR menyu)" ko'rinadi; dashboardda alohida
  **"Stolda (QR)"** ko'rsatkichi.

**Tekshirildi**: `dinein` buyurtma 201, yetkazish narxi 0, stol raqami yozildi ✓ ·
noto'g'ri/bo'sh `tableId` → 400 ✓ · kuzatuv sahifasi stolni ko'rsatadi ✓ ·
statistika `dineIn: 1` ✓ · QR sahifasi kartochkani chizdi (1200×1700, QR
piksellari bor) ✓ · fon almashtirilganda kartochka qayta chizildi ✓ · sarlavha
tahriri darhol aks etdi ✓ · yuklab olish `qr-stol-1.png` ✓ · telefonda
`/menu?table=tbl1` → "1-stoldasiz" chizig'i ✓ · checkout'da "Stolga" avtomatik
tanlandi ✓ · buyurtma berildi va kuzatuvda stol ko'rindi ✓

---

## 2026-07-30 (25) — QR kartochkalarni bitta ZIP qilib yuklash ✅

Ilgari "Hamma stollar" tugmasi har stolga alohida PNG yuborardi — brauzer
"bir nechta faylni yuklashga ruxsat berasizmi?" deb so'raydi va rad javobida
qolganini jimgina tashlab yuboradi. Endi hammasi **bitta arxiv**.

- `lib/zip.ts` — kutubxonasiz, qo'lda yozilgan ZIP (store-only: PNG allaqachon
  siqilgan, uni qayta siqish deflate implementatsiyasini olib keladi-yu hech
  nima yutmaydi). CRC32, local header, central directory, EOCD va to'g'ri
  MS-DOS sana/vaqt.
- `saveBlob()` — havolani DOM'ga qo'yib bosadi va olib tashlaydi (Chrome'dan
  tashqarida "detached" havola yuklashni ishonchli boshlamaydi);
  `revokeObjectURL` kechiktiriladi (Safari aks holda yuklashni bekor qiladi).
- Arxivda barcha faol stollar + **umumiy restoran kartochkasi**:
  `qr-kodlar.zip` → `qr-stol-1.png`, …, `qr-restoran.png`.

**Tekshirildi**: bitta kartochka → `qr-restoran.png` ✓ · arxiv → `qr-kodlar.zip` ✓ ·
`unzip -t` xatosiz, `unzip -l` 4 ta fayl va to'g'ri sana/vaqt ko'rsatdi ✓ ·
ochilgan PNG'lar 1200×1700 ✓ · **chop etilgan QR haqiqatan to'g'ri havolani
beradi**: PNG'dan o'qilgan 29×29 modul to'ri `?table=tbl1` havolasining
kodlanishi bilan bir xil (**0 farq**) ✓

---

## 2026-07-30 (26) — QAROR: ko'p brend + ko'p filial arxitekturasi 📐

Mijoz holati (haqiqiy): bitta kompaniyada **milliy taomlar restorani** (zal,
bron, keng menyu) va **somsa tarmog'i** — restoran yonida bitta nuqta + boshqa
joylarda 6–7 filial. Narxlar hamma filialda bir xil, har filial o'zi pishiradi,
kuryerlar filialga biriktirilgan, kassalar alohida. Tizim **kompaniya boshiga**
sotiladi.

### Qaror
Bitta deploy ichida ikki o'lchov qo'shiladi:

```
Kompaniya (bitta baza, bitta mijozlar bazasi)
├── Brend "Restoran"  → menyu A, dizayn A, zal + bron + yetkazish
│     └── filial 1
└── Brend "Somsa"     → menyu B, dizayn B, olib ketish + yetkazish
      └── filial 1..8
```

- **Brend** = menyu, dizayn, sayt matnlari, xizmat turlari (yetkazish / olib
  ketish / zalda / bron).
- **Filial** = manzil, ish vaqti, yetkazish zonalari va minimal buyurtma,
  kuryerlar, buyurtmalar oqimi, stol xaritasi, "tugadi" bayrog'i.
- **Mijoz kompaniya darajasida**: telefon raqami, manzillari va butun tarixi
  ikkala brend uchun bitta. Aynan shu talab tufayli **ikki alohida deploy
  varianti rad etildi** — u holda bazalar ham alohida bo'lardi va keyin
  birlashtirib bo'lmasdi.
- **Savat brend bo'yicha alohida**: osh bilan somsa boshqa oshxonada pishadi va
  boshqa kuryer olib boradi — bitta savatga sig'maydi.
- Mijoz filialni **tanlamaydi**: yetkazishda manzilidan kelib chiqib server
  o'zi topadi (bir nechta zona qamrasa — eng yaqini), olib ketishda esa masofa
  bo'yicha ro'yxat ko'rsatiladi.

### Muhim shart
**Bitta brend + bitta filialli mijoz hech qanday murakkablikni ko'rmaydi** —
almashtirgichlar umuman chizilmaydi. Aks holda shablonning asosiy auditoriyasi
(kichik restoran) uchun mahsulot buziladi.

### Ochiq savol (2-bosqichgacha kerak emas)
Domen: bitta domenda ikki bo'lim (`/restoran`, `/somsa`) — mijoz bir marta
kiradi; yoki ikki domen — brend jihatdan chiroyliroq, lekin sessiya domenga
bog'langani uchun har domenda bir martadan SMS kod kiritiladi (hisob baribir
bitta). SSO qo'shilsa yashirish mumkin.

### Bosqichlar
1. Model va migratsiya (brand/branch, mavjud ma'lumot "Restoran / 1-filial"
   bo'lib ko'chadi), panelda almashtirgich, filial CRUD.
2. Oqimlar: menyu/buyurtma/kuryer/statistika brend+filial kesimida; saytda
   brend bo'limlari, brend savati, manzildan filialni topish.
3. Bron va QR filialga bog'lanadi, "tugadi" bayrog'i, ovoz filial bo'yicha,
   buyurtma raqamiga filial prefiksi, menejer huquqlari.

---

## 2026-07-30 (27) — 1-BOSQICH: brend + filial modeli va migratsiya ✅

### Backend
- `models.Brand` (menyu + ko'rinish + xizmat turlari) va `models.Branch`
  (manzil, ish vaqti, yetkazish sozlamalari, stol xaritasi, tayyorlash vaqti).
  Kolleksiyalar: `brand`, `branch`.
- **Qamrov maydonlari**: `category`/`menu_item` → `brandId`; `order` →
  `brandId`+`branchId`; `courier`, `reservation` → `branchId`;
  `admin_user` → `branchId` (bo'sh = butun kompaniya, ya'ni owner).
- **Migratsiya** (`repository/migrate.go`, har ishga tushishda, idempotent):
  brend yo'q bo'lsa restoran profilidan **brend #1** va **filial #1** yaratiladi
  (manzil, telefonlar, ish vaqti, yetkazish zonalari, stol xaritasi filialga
  ko'chadi), mavjud barcha hujjatlarga qamrov maydonlari qo'yiladi. Eski
  maydonlar **o'chirilmaydi** — orqaga qaytish hech narsa yo'qotmaydi.
- Indekslar: `branch.brandId`, `category/menu_item.brandId`,
  `order.branchId+createdAt`, `courier.branchId`, `reservation.branchId+at`.
- API: `GET /brands` (public), `GET|POST|PUT|DELETE /admin/brands`,
  `/admin/branches`. **O'chirish qoidalari**: oxirgi brend/filial o'chmaydi;
  filiali bor brend o'chmaydi; **buyurtmasi bor filial o'chirilmaydi — yopiladi**
  (cheklar javobsiz qolmasligi uchun). Hammasi amallar jurnaliga tushadi.

### Frontend
- `lib/adminScope.tsx` — panel qaysi brend/filial ko'zi bilan qarayotgani
  (localStorage'da eslab qolinadi; brendda bitta filial bo'lsa avtomatik o'sha,
  bir nechta bo'lsa "Hamma filiallar").
- `components/admin/ScopeSwitcher.tsx` — yon panelda va telefondagi yuqori
  panelda. **Bitta brend + bitta filialda umuman chizilmaydi.**
- `components/admin/BranchesEditor.tsx` — Sozlamalar → "Brendlar": brend
  qo'shish/o'chirish, xizmat turlari (yetkazish / olib ketish / stolda / bron),
  filial qo'shish, nomi, telefonlari, tayyorlash vaqti, faolligi va xaritada
  manzili. Ya'ni to'qqizinchi somsa nuqtasi — tugma, deploy emas.

**Tekshirildi** (haqiqiy bazada): migratsiya `Maracanda` brendi/filialini
yaratdi ✓ · 16 buyurtma, 1 kuryer, 1 bron `branchId` oldi ✓ · 7 kategoriya,
48 taom `brandId` oldi ✓ · kompaniya hujjati va sayt buzilmadi ✓ ·
1 brend/1 filialda almashtirgich **yo'q** ✓ · brend + 2 filial qo'shilgach
almashtirgich chiqdi va ro'yxatlar to'g'ri ✓ · buyurtmali filialni o'chirish →
**yopildi** (`deactivated: true`) ✓ · bo'sh filial o'chdi ✓ · filiali bor
brendni o'chirish → 400 ✓ · sayt sahifalari toza (gorizontal surilish yo'q,
konsol xatosi yo'q) ✓

Sinov uchun yaratilgan "Somsa" brendi va filiallari o'chirildi, asl filial
qayta faollashtirildi — baza avvalgi holatida.

---

---

## 2026-07-31 (28) — 2-BOSQICH: oqimlar brend + filial kesimida ✅

1-bosqich modelni qo'ydi; bu bosqich butun tizimni shu linzadan o'qiydigan
qildi. Bitta brend + bitta filialli mijoz uchun **hech narsa o'zgarmadi** —
so'rovlar ham, ekranlar ham avvalgidek.

### Backend — `internal/handlers/scope.go`
- **`Scope`** — brend/filial linzasi. Nol id "filtrsiz" degani, ya'ni bitta
  filialli install'da so'rovlar aynan avvalgidek quriladi.
- **`adminScope(r)`** `?brandId=`/`?branchId=` ni o'qiydi va **`clampToAdmin`**
  bilan cheklaydi: `admin_user.branchId` qo'yilgan menejer URL'da nima yozsa
  ham o'z filialidan chiqa olmaydi.
- **`deliveryBranch`** — manzilni qamrab oladigan filiallar orasidan **eng
  yaqinini** tanlaydi va narxini ham qaytaradi (manzil ikki marta narxlanmasin).
  Mijoz filialni **tanlamaydi**.
- `quoteDelivery` → **`quoteDeliveryFrom(delivery, origin, ...)`**: narx endi
  filialning o'z sozlamalari va o'z manzilidan hisoblanadi.

### Backend — qamrov qo'llanilgan joylar
- Admin ro'yxatlari: kategoriya/menyu (brend), buyurtma/bron/kuryer (filial),
  statistika va **ovozli bildirishnoma** (filial). Mijozlar va panel adminlari
  ataylab **kompaniya darajasida** qoldi.
- Yangi yozuvlar linzadan muhr oladi (`scopeBrand`); `ReplaceOne` bilan
  yangilashda `keepBrandID` eski brendni saqlaydi.
- **Kuryer biriktirish**: boshqa filial kuryeri biriktirilmaydi (aks holda u
  "Yetkazdim"ni hech qachon bosa olmasdi — manzil 300 km narida).
- **Kuryerning kelish radiusi** filialdan olinadi.
- `POST /orders`: brend **savatdagi taomlardan** aniqlanadi (brauzer yuborgan
  maydondan emas); ikki brend aralashgan savat → 400. Filial: yetkazishda
  manzildan, olib ketish/stolda esa `branchId` dan.
- Bron va stol xaritasi filialga bog'landi — 7-stol har filialda bor,
  bandlik tekshiruvi endi filial ichida.
- `GET /restaurant` javobiga filialning manzili/telefoni/ish vaqti/yetkazish
  sozlamalari va **brendning yuzi** (nom, logo, matnlar, tema) qatlanadi.
  Shu bitta joy tufayli sayt sahifalariga tegilmadi. `?raw=1` — panel uchun.
- **`PUT /admin/restaurant` endi qisman yangilash**: sozlamalar sahifasi
  brendga/filialga tegishli maydonlarni yubormaydi, to'liq `$set` esa ularni
  bo'sh qatorga aylantirib kompaniya hujjatini yeb qo'yardi.

### Frontend
- `lib/api.ts`: **`setAdminScope()`** — linza modul o'zgaruvchisida, `scope:
  true` bergan admin chaqiruvlariga avtomatik qo'shiladi. `SiteScope`
  (`?brand=`, `?branchId=`) public chaqiruvlar uchun.
- `AdminScopeProvider` linzani **render vaqtida** beradi (effektda emas — aks
  holda yangi ekranning birinchi so'rovi filtrsiz ketardi); `scopeKey` bilan
  `ScopedMain` ekranni qayta yuklaydi, `AlertBell` esa "ko'rilgan" belgisini
  tozalaydi (boshqa filialning buyurtmasi qo'ng'iroq chalmasin).
- **Savat brend bo'yicha alohida** (`cart_v2:<brandId>`) — osh bilan somsa
  boshqa oshxonada pishadi.
- Sayt linzasi **cookie'da** (`brand`, `branch`) — menyu server'da
  render qilinadi, localStorage kech qolardi. `?brand=`/`?branch=` (QR
  kartochkadan) cookie'ni almashtiradi; stoldan chiqilganda filial cookie'si
  o'chadi (uyga borib yetkazish buyurtma qilganda eski filial yopishmasin).
- `components/site/BrandSwitch.tsx` — **bitta brendda umuman chizilmaydi**.
- Checkout: yetkazishda qaysi filial xizmat qilishini ko'rsatadi (ikki filial
  bo'lsa), olib ketishda filial tanlanadi va xarita/marshrut o'sha filialga.
- `/bron` da filial tanlagichi; `/admin/qr` kartochkasi havolaga brend va
  filialni yozadi (kartochka stolga yillab yopishtiriladi).
- Sozlamalar: kompaniya / brend / filial maydonlari **uchga bo'lib** saqlanadi.
  Filial tanlanmagan bo'lsa (ko'p filialli holat) avval tanlash so'raladi.

**Tekshirildi** (haqiqiy bazada, sinov uchun "Somsa Lavash" brendi + 2 filial
yaratilib): brend menyulari ajralgan (48 va 1) ✓ · Chilonzor manzili →
Chilonzor filiali, Yunusobod manzili → Yunusobod filiali, ikkalasidan uzoq →
"yetkazib berilmaydi" ✓ · buyurtma to'g'ri `brandId`+`branchId` bilan yozildi ✓ ·
aralash savat → 400 ✓ · admin ro'yxatlari va statistika kesimlarda to'g'ri
(16 / 1) ✓ · **filialga biriktirilgan menejer** URL'da boshqa filialni so'rasa
ham faqat o'zinikini ko'rdi ✓ · SSR: `brand=somsa` cookie bilan menyu, sarlavha
va "Biz haqimizda" manzili almashdi ✓ · sinov brendi o'chirilgach almashtirgich
**yo'qoldi**, sayt avvalgi holatiga qaytdi ✓ · `go build`, `tsc --noEmit`,
`next lint`, `npm run build` — toza ✓

Sinov ma'lumotlari (brend, 2 filial, 1 kategoriya, 1 taom, 1 buyurtma, vaqtinchalik
admin) o'chirildi — bazada 16 buyurtma, 48 taom, `yujo` admin qoldi.

⚠️ Bazada 27-kunlik sinovdan **"Filial nomi"** degan bo'sh filial qolib ketgan —
panelda o'chirish kerak (u borligi uchun almashtirgich chizilishi mumkin).


---

## 2026-07-31 (29) — 3-BOSQICH: "tugadi", filial prefiksi, menejer huquqlari ✅

Brend/filial ishining oxirgi uchdan biri. Bitta filialli mijoz uchun yana
hech narsa o'zgarmadi.

### 1. "Tugadi" — filial bo'yicha mavjudlik
- Menyu **brendniki** (hamma filialda bir xil), lekin **qozonda nima qolgani**
  filialniki. Shuning uchun `branch.soldOut []ObjectID` — taomda emas, filialda.
- Ikki xil "yo'q" ajratildi: `isAvailable` — taom umuman menyuda yo'q;
  `soldOut` — shu filialda bugun tugadi. Saytda ikkalasi ham tugmani
  o'chiradi, faqat matni boshqa ("Bugun tugadi").
- Public `/menu` va `/menu/{id}` javobiga hisoblanadigan `soldOut` bayrog'i
  qo'shiladi (bazada saqlanmaydi).
- **Asosiy himoya serverda**: `CreateOrder` filial aniqlangandan **keyin**
  tekshiradi — mijoz bir filialning menyusini ko'rgan bo'lishi, buyurtmani esa
  boshqasi olishi mumkin.
- Panel: `/admin/menu` da har qatorda bir bosishli tugma (optimistik, chunki u
  ish qizigan paytda bosiladi). Filial tanlanmagan bo'lsa tugma **yo'q** —
  "qayerda tugadi?" degan savolga javob yo'q.
- Alohida endpoint `PUT /admin/branches/{id}/sold-out` va `$addToSet`/`$pull`:
  ikki kishi bir vaqtda ikki taomni belgilasa biri ikkinchisini yo'qotmasin.
  Shu sababli **`AdminUpdateBranch` `soldOut` ni umuman yozmaydi** — bir soat
  ochiq turgan forma oshxona tugatgan taomlarni qaytarib qo'yardi.

### 2. Filial prefiksi (`branch.code`)
- `branchOrderNumber(code)` → `MRC-YS19-1745`. Telefonda o'qilgan raqam
  qaysi oshxonaniki ekanini **qidirmasdan** aytadi. Bron raqamiga ham qo'yiladi.
- Kod faqat A–Z va 0–9 (6 belgigacha) — u ovoz chiqarib o'qiladi.
- Bo'sh bo'lsa prefiks qo'yilmaydi: bitta filialda u hech nima demaydi.

### 3. Menejer huquqlari
- Yangi yordamchilar: **`requireOwner`** (kompaniya profili, brend CRUD, filial
  ochish/o'chirish) va **`requireBranchAccess`** (menejer faqat o'z filialini
  tahrirlaydi va faqat o'z filialida "tugadi" belgilaydi).
- Panelda: `AdminScopeProvider` `/admin/me` ni ham o'qiydi; `branchId` qo'yilgan
  menejer uchun linza **o'sha filialga qulflanadi** va almashtirgich o'rniga
  filial nomi turadi — jim turadigan tugma tugmasizlikdan yomon.
- Sozlamalarda kompaniya/brend bo'limlari (profil, sayt matnlari, dizayn,
  brendlar) menejerga **ko'rinmaydi**; manzil, ish vaqti, yetkazish, zal
  xaritasi va tashqi xizmatlar — ko'rinadi. `save()` ham owner bo'lmagan
  chaqiruvlarni umuman yubormaydi (403 filial saqlanmagandek ko'rinardi).

**Tekshirildi** (haqiqiy bazada, vaqtinchalik owner + manager hisoblari bilan):
kod saqlandi va buyurtma raqami `MRC-YS19-1745` bo'ldi ✓ · "tugadi" → public
menyuda bayroq, buyurtma **"Lag'mon bugun tugadi"** bilan rad etildi, "bor"
qilingach bayroq yo'qoldi ✓ · menejer uchun: kompaniya profili 403, brend 403,
filial ochish 403, **boshqa** filialni yozish 403, boshqa filialda "tugadi" 403,
o'z filialida "tugadi" 200, jurnal 403 ✓ · `go vet`, `gofmt`, `tsc --noEmit`,
`next lint`, `npm run build` — toza ✓

Sinov ma'lumotlari o'chirildi (2 vaqtinchalik admin, 1 buyurtma, sinov uchun
qo'yilgan `MRC` kodi ham qaytarib olindi) — bazada 16 buyurtma, 48 taom,
`yujo` admin qoldi.


---

## 2026-07-31 (30) — Admin parolini SMS bilan tiklash ✅

Har bir mijoz o'z VPS'ida ishlagani uchun "parolni unutdim" ilgari SSH sessiya
va `cmd/adminreset` degani edi — terminal ochib ko'rmagan restoran egasidan
keladigan qo'ng'iroq. Endi kirish sahifasidan hal bo'ladi.

### Avval xavfsizlik teshigi yopildi: kodlar maqsad bo'yicha ajratildi
`phone_code` faqat telefon bo'yicha kalitlangan edi. **Restoran egasi ko'pincha
saytning ham mijozi — bir xil raqam bilan.** Ya'ni "kirib, tushlik buyurtma
qiling" uchun kelgan olti xonali kod admin panelini ochib bera olardi.
- `PhoneCode.Purpose` qo'shildi (`login` / `admin_reset` / `admin_phone`),
  `issueCode` va `checkCode` shu bo'yicha kalitlanadi.
- `(phone, purpose)` bo'yicha **unique indeks** — ikki oqim bir-birini
  o'chirib yubormaydi.
- `expiresAt` bo'yicha **TTL indeks** — muddati o'tgan kodlar o'zi o'chadi
  (ilgari kolleksiya faqat o'sardi).

### Tiklash oqimi
- `POST /admin/password/forgot {username}` → hisobdagi raqamga kod, javobda
  **niqoblangan raqam** (`+998 ** *** ** 67`).
- `POST /admin/password/reset {username, code, newPassword}` → parol
  o'rnatiladi, `mustChangePassword` tozalanadi.
- Ikkalasi ham **public** — kira olmayotgan odam uchun boshqa yo'l yo'q.
  60 soniyalik cooldown (`issueCode`) egasining telefonini spamdan saqlaydi.
- **Jurnalga yoziladi** (`admin.password.reset`) — o'sha paytda hech kim
  tizimga kirmagan bo'lsa ham "kim parolni o'zgartirdi?" javobsiz qolmasin.
- Login sahifasida uch qadamli forma (`components/admin/ForgotPassword.tsx`).

### Tiklash raqami (busiz oqim ishlamaydi)
Seed qilingan birinchi `owner` da telefon **yo'q**, ya'ni eng keng tarqalgan
hisob uchun tiklash umuman ishlamas edi. Panel → Hisobim →
`components/admin/RecoveryPhone.tsx`: raqam **SMS bilan tasdiqlanadi** (noto'g'ri
raqam aynan kerak bo'lgan kuni ma'lum bo'lishi eng yomon variant), raqam yo'q
bo'lsa sariq ogohlantirish turadi.

### Qaror: mavjud bo'lmagan login uchun "topilmadi" deyiladi
Hisob borligini oshkor qiladi. Ataylab: bu bitta restoranning o'z paneli,
tiklash baribir egasining qo'lidagi telefonni talab qiladi, "agar hisob
mavjud bo'lsa..." degan javob esa loginini noto'g'ri yozgan odamni hech
narsasiz qoldiradi — ya'ni aynan shu oqim yo'q qilmoqchi bo'lgan qo'ng'iroq.

**Tekshirildi**: telefonsiz hisob → aniq ko'rsatma bilan xato ✓ · noto'g'ri
login → 404 ✓ · kod → niqoblangan raqam ✓ · noto'g'ri kod rad etildi, to'g'risi
o'tdi, yangi parol bilan kirish OK ✓ · **mijoz login kodi bilan admin parolini
tiklash — "kod topilmadi"** ✓ · admin tiklash kodi bilan mijoz login —
"kod noto'g'ri" ✓ · ikkalasi o'z maqsadida baribir ishlaydi ✓ · panelda raqam
qo'shildi, tasdiqlandi, tiklash **yangi raqamga** ketdi ✓ · auth'siz urinish
401 ✓ · jurnalda `admin.phone` va `admin.password.reset` yozuvlari ✓ ·
`go vet`, `gofmt`, `tsc`, `next lint`, `npm run build` — toza ✓

`DEPLOY.md` qayta yozildi: 1-yo'l panelning o'zidan (SMS), 2-yo'l serverda
(zaxira). Mijozga topshirishdan oldin tiklash raqamini biriktirish shart, va bu
faqat **haqiqiy SMS provayderi** bilan ishlaydi (`demo` da kod SMS'siz qaytadi).

Sinov ma'lumotlari o'chirildi (vaqtinchalik admin, mijoz, kodlar, 5 jurnal
yozuvi) — bazada 16 buyurtma, 48 taom, 8 mijoz, `yujo` admin qoldi.


---

## 2026-07-31 (31) — A-BOSQICH: Combo (belgilangan to'plam) ✅

Loyalty / promokod / aksiya / combo so'rovining birinchi bo'lagi. Qolgan
uchtasi keyingi bosqichlarda (pastdagi rejaga qarang).

### Asosiy qaror: combo — alohida kolleksiya emas, **menyu elementining turi**
`menu_item` ichida `comboItems [{menuItemId, qty}]`. Bo'sh bo'lmasa — bu combo.
Sabab: to'plamga rasm, tarjima, kategoriya, tartib, brend, mavjudlik bayrog'i va
savatdagi o'rni kerak — bularning **hammasi taom uchun allaqachon ishlaydi**.
Ikkinchi tur qilinsa, savat, buyurtma, QR menyu va admin ro'yxatiga bularning
hammasini ikkinchi marta o'rgatish kerak bo'lardi.

### Tejash hech qachon saqlanmaydi
"Alohida olinganda 50 000" raqami **har so'rovda hisoblanadi** (`comboBasePrice`,
`comboContents` — `bson:"-"`). Saqlangan nusxa a'zo taom narxi o'zgarishi bilan
yolg'onga aylanadi, noto'g'ri tejash esa umuman ko'rsatmaslikdan yomon.

### To'plam — bo'linmas
- A'zolaridan biri **tugagan** bo'lsa combo ham "Bugun tugadi" bo'ladi
  (`decorateCombos`), va buyurtma serverda rad etiladi — yarim to'plam to'plam
  emas.
- A'zosi **menyudan o'chirilgan** bo'lsa combo sotilmaydi va admin panelda
  o'sha taomni o'chirishga urinish **409** bilan to'xtaydi, qaysi to'plamda
  ishlatilgani nomi bilan aytiladi.
- Chekka **tarkib nusxa** bo'lib yoziladi (`OrderItem.ComboItems`): oshxona
  chekdan pishiradi, "Oilaviy combo" degan yozuv esa buyruq emas.

### Majburiy variantli taom to'plamga kirmaydi
"2 × Lag'mon" — aniq buyruq; "2 × Lag'mon (qaysi hajm?)" — yo'q, va belgilangan
to'plamda so'rash joyi yo'q. Server ham, admin UI ham rad etadi. Guruhdan
tanlanadigan to'plam — ataylab qurilmagan boshqa xususiyat.

### Panel
`components/admin/ComboEditor.tsx` — taomlarni qidirib qo'shadi, sonini
o'zgartiradi va **jonli** "alohida olinganda / tejaladi" qatorini ko'rsatadi.
To'plam alohida olganidan arzon bo'lmasa sariq ogohlantirish chiqadi. Menyu
formasida "Taom / To'plam" tugmalari — ikkalasi bir vaqtda bo'lmaydi.

**Tekshirildi**: majburiy variantli taom → rad ✓ · bitta taomli to'plam → rad ✓ ·
takror taom → rad ✓ · to'g'ri combo yaratildi, menyuda tarkib + "10 000 so'm
tejaysiz" + ustidan chizilgan 50 000 ✓ · buyurtma chekida tarkib yozildi ✓ ·
a'zosi tugadi → combo "Bugun tugadi", buyurtma "Ko'k choy bugun tugadi" bilan
rad etildi ✓ · a'zoni o'chirishga urinish → 409, to'plam nomi bilan ✓ ·
taom sahifasida tarkib bloki ✓ · `go vet`, `gofmt`, `tsc`, `next lint`,
`npm run build` — toza ✓

⚠️ Sinov paytida **ikkita `next-server` jarayoni** bir vaqtda ishlab turgani
aniqlandi (eskisi javob berardi) — natijalar chalkashdi. Kelgusi sinovlarda
`npm start` dan oldin portni tekshirish kerak.

Sinov ma'lumotlari o'chirildi — bazada 48 taom, 0 combo, 16 buyurtma, `yujo`.

### Keyingi bosqichlar (kelishilgan)
- **B**: chegirma mexanizmi — promokod va aksiya **bitta qoida, ikki tetik**
  (kod bilan / avtomatik). Serverda yagona narxlash quvuri, 0 dan past
  tushmaydi, har chegirma chekda alohida qator.
- **C**: loyalty — **keshbek ballari** (buyurtmadan N% qaytadi, keyingisida
  to'lovga ishlatiladi), balans + tranzaksiyalar ledgeri, bekor qilinganda
  qaytarib olinadi.
- Stacking qarori: **aksiya + promokod + ball uchalasi ham** qo'llanadi, qat'iy
  tartibda, total hech qachon manfiy emas.


---

## 2026-07-31 (32) — B-BOSQICH: chegirma mexanizmi (promokod + aksiya) ✅

### Asosiy qaror: bitta qoida, ikkita tetik
`promotion` kolleksiyasi promokod uchun ham, aksiya uchun ham bitta. Ular
kassaga bir xil ko'rinadi — buyurtmadan pul oladigan qoida — va faqat **nima
ishga tushirishi** bilan farq qiladi: mijoz kiritgan kod, yoki shunchaki kun va
soat. Alohida yozilsa bir xil amal muddati, bir xil cheklov va bir xil chek
qatorini ikki marta yozib, keyin ular bir-biriga mos kelmasligini kashf qilardik.

### Yagona narxlash quvuri (`handlers/pricing.go`)
Bepul buyurtma xatosi aynan shu yerda tug'iladi, shuning uchun **bitta** quvur,
qat'iy tartibda, va har qadam faqat **qolganini** olib tashlaydi:

```
qatorlar (combo narxi bilan)  → subtotal
  → eng foydali AVTOMATIK aksiya (faqat bittasi)
  → promokod
  → yetkazish narxi (bepul yetkazish uni nolga tushiradi)
= total, har qadamda 0 dan past emas
```

- **Faqat bitta avtomatik aksiya** qo'llanadi — eng foydalisi. Egasi unutgan
  ikkita ustma-ust kampaniya ("seshanba −20%", "iyul −30%") aks holda iyul
  seshanbasida yarim narxga aylanardi.
- Foizli chegirmaga **shift** (`maxDiscount`) — "20% chegirma" katta banket
  buyurtmasida cheksiz va'daga aylanmasin.
- Chek quvurni **qaytadan** yuritadi: brauzer qaytargan raqamga hech qachon
  ishonilmaydi. Preview faqat preview.

### Minimal buyurtma — chegirmadan **keyin** tekshiriladi
Restoranga tushadigan pul muhim: promokod savatni filial minimumidan pastga
tushirsa, yetkazish zarar bo'lardi. Checkout `payableSubtotal` va
`belowMinimum` ni alohida oladi va buni mijozga tushuntiradi.

### Boshqa tafsilotlar
- **Har chegirma chekda alohida qator** (`order.discounts`), nomi va summasi
  **nusxa** bo'lib saqlanadi — kampaniya o'chirilsa ham "nega 12 000 kam?"
  degan savolga chek o'zi javob beradi.
- Xato promokod **halokat emas**: buyurtma to'liq narxda o'tadi, sabab esa
  aniq ("promokod 50 000 so'mdan yuqori buyurtmalar uchun" — mijoz o'zi
  tuzatadi, telefon qilmaydi).
- `usageLimit` `$expr` bilan filtrda qo'riqlanadi — oxirgi kupon ikki kishiga
  ketmasin. Hisob **buyurtma yaratilgandan keyin** oshadi.
- `AdminUpdatePromotion` `usedCount` ni forma ustiga yozmaydi — aks holda
  ishlatilgan kod qayta tarqalardi.
- Public `GET /promotions` **faqat aksiyalarni** beradi; hech kimga
  berilmagan kod — bu promokod emas, sizib chiqish.

### Panel va sayt
- `/admin/promotions` — ikki tab (Aksiyalar / Promokodlar), bitta forma:
  tur → chegirma → shartlar (sana, soat, hafta kunlari, buyurtma turi,
  kategoriya) → cheklovlar (faqat kod uchun).
- Checkout'da promokod maydoni, jonli chegirma qatorlari va bepul yetkazish.

**Tekshirildi**: 110% / qisqa kod / tursiz — rad ✓ · takror kod → 409 ✓ ·
ikki aksiyadan **kattarog'i** tanlandi (30 000 > 12 600) ✓ · stacking:
126 000 → −30 000 → qolganidan −15% = −14 400 → 81 600 ✓ · aksiya minimalidan
past → faqat kod ✓ · **100% aksiya + 999 999 so'mlik kod → total 0, manfiy
emas**, kod "chegirma bermaydi" deb aytdi ✓ · bepul yetkazish yetkazishda
ishladi, olib ketishda ishlamadi ✓ · haqiqiy buyurtma ikkala chegirmani chekka
yozdi, `usedCount` 1/1 bo'ldi, ikkinchi urinish "ishlatib bo'lingan" ✓ ·
public `/promotions` kodni ko'rsatmadi ✓ · `go vet`, `gofmt`, `tsc`,
`next lint`, `npm run build` — toza ✓

Sinov ma'lumotlari o'chirildi — bazada 16 buyurtma, 0 aksiya, `yujo`.

⚠️ Ish jarayonida topilgan: `router.go` ga marshrut qo'shish **jimgina
o'tmagan** (skriptda assert yo'q edi, tab soni mos kelmagan) va endpoint 405
qaytardi. Kelgusida har bir matn almashtirishga assert qo'yish shart.


---

## 2026-07-31 (33) — Promokod sozlamalari to'ldirildi + C-BOSQICH: loyalty ✅

### Promokod sozlamalari: ikkita haqiqiy kamchilik yopildi
So'ralgan sozlamalarning aksari B-bosqichda allaqachon bor edi (`usageLimit`,
`perUserLimit`, `firstOrderOnly`, `startsAt`/`endsAt`, soat, hafta kunlari,
buyurtma turi, `minOrder`, `maxDiscount`). Yetishmagani:

1. **Hisoblanadigan holat** (`Promotion.StatusAt`): `running` / `idle` (bugun
   emas) / `scheduled` / `expired` / `usedUp` / `off`. Ilgari muddati tugagan
   kod ro'yxatda yashil "Faol" bo'lib turardi — egasi uni reklama qilishda
   davom etardi. Endi beshta holat farqlanadi, chunki "muddati tugagan" va
   "dushanbada emas" bir xil javob emas.
2. **"Kim ishlatgan"** (`GET /admin/promotions/{id}/usage`): buyurtmalardan
   hisoblanadi, `usedCount` dan emas. **"Necha kishi" va "necha marta"
   ajratilgan** — bitta doimiy mijoz kodni besh marta ishlatsa, bu bitta
   odam, va bu farq egasining xulosasini o'zgartiradi.

### C-bosqich: keshbek ballari
**1 ball = 1 so'm** — ataylab, shunda "6 000 ballingiz bor" va "6 000 so'm
chegirma" bir xil jumla bo'ladi, hech kim kurs o'rganmaydi.

Ikkita qoida hammasini belgilaydi:
- **Ball buyurtma berilganda yechiladi, yetkazilganda qo'shiladi.** Yechish
  darhol bo'lishi kerak — aks holda bir xil balans ikki buyurtmaga va'da
  qilinardi; qo'shish esa kutishi kerak — aks holda bekor qilingan buyurtma
  yo'qdan ball yasab beradi.
- **Ball bilan to'lash ball keltirmaydi.** Aks holda balans o'z-o'zini
  boqib, asta-sekin bepul ovqatga aylanadi.

Har harakat `loyalty_txn` ga yoziladi. Balans foydalanuvchida — tezlik uchun
kesh; **ledger esa yozuv**, chunki "mening 6 000 im qayerga ketdi?" degan
savolga oylar o'tib ham javob bo'lishi kerak. Balans `$inc` bilan siljiydi:
bir vaqtda hisoblangan ikki total bir-birini yo'q qilmasin.

**Shift majburiy** (`maxRedeemPercent`, standart 50%): shiftsiz katta balans
buyurtmani butunlay bepul qiladi, oshxona esa uni baribir pishiradi.

Narxlash quvurida ballar **uchinchi** — chegirmalardan keyin. Ataylab: ball —
mijozning o'z puli, va aksiya baribir olib tashlaydigan summaga sarflansa,
balans bekorga yonib ketardi.

**Tekshirildi**: holatlar — beshtasi ham to'g'ri ✓ · "kim ishlatgan": 3 marta,
**2 kishi** ✓ · xush kelibsiz ballari bir marta ✓ · 50% shift ishladi
(16 000 buyurtma, 10 000 balans → maks 8 000) ✓ · balansdan ko'p so'rash
qirqildi ✓ · **to'liq sikl**: 10 000 yechildi → 0, yetkazilgach +5 800,
bekor qilingach +10 000 −5 800 → **boshlang'ich 10 000 ga qaytdi** ✓ ·
**idempotentlik**: "yetkazildi" 3 marta bosilganda bir marta to'landi,
"bekor" 2 marta bosilganda bir marta qaytdi ✓ · ball bilan to'langanda
keshbek kamaydi (6 300 → 5 800 — o'z-o'zini boqmaydi) ✓ · loyalty o'chiq
bo'lsa va kirmagan mijozga ballar taklif qilinmadi ✓ · `go vet`, `gofmt`,
`tsc`, `next lint`, `npm run build` — toza ✓

Sinov ma'lumotlari o'chirildi va **loyalty sozlamasi bazadan olib tashlandi**
(u meniki edi, mijozniki emas) — 16 buyurtma, 8 mijoz, 0 aksiya, `yujo`.

⚠️ Yana bir marta: **eski server jarayoni** (17:19 da ishga tushgan) yangi
binar o'rniga javob berib turgani aniqlandi va sinov noto'g'ri natija berdi.
Endi har ishga tushirishda `ps -o lstart` bilan jarayon vaqti tekshirilyapti.
`pgrep -f` ham o'z buyrug'ini topadi — `pgrep -x` ishlatish kerak.


---

## 2026-07-31 (34) — REGRESSIYA TUZATILDI: sozlamalar sahifasi yo'qolgan edi 🐛

**Mijoz xabari**: "settings bo'limida eski sozlamalar qani? sayt dizaynini
o'zgartirish va boshqalar".

### Sabab — 3-bosqichda men kiritgan xato
Menejer huquqlarini qo'shganda sozlamalar sahifasiga shart qo'ygandim:
*bir nechta filial bor, lekin bittasi tanlanmagan → "avval filialni tanlang"* —
va **butun sahifani** shu xabar bilan almashtirgandim.

Bazada ikkita filial bor (`Maracanda` + 30-iyuldagi sinovdan qolgan
**`Filial nomi`**), shuning uchun panel "Hamma filiallar" rejimida ochiladi va
sozlamalar sahifasi butunlay to'xtab qolgan edi: dizayn, sayt matnlari, profil,
ballar, brendlar — hammasi ko'rinmay ketgan.

**Mantiqdagi xato**: manzil, ish vaqti, yetkazish va zal xaritasi haqiqatan
filialga tegishli, lekin **dizayn, matnlar, profil va ballar filialga umuman
bog'liq emas**. Ularni yashirish uchun hech qanday sabab yo'q edi.

### Tuzatish
- Butun sahifani to'xtatuvchi `return` olib tashlandi.
- `Section` komponentiga `blockedBy` qo'shildi: filialga tegishli **to'rtta**
  bo'lim filial tanlanmaganda maydonlar o'rniga izoh ko'rsatadi. Saqlanmaydigan
  maydonga yozdirish maydon yo'qligidan yomon.
- Qolgan hamma bo'lim doim ochiq.

### Ikkinchi xabar: "almashtirgich qayerda?"
Adolatli e'tiroz edi. Almashtirgich **allaqachon** to'g'ri joyda — yon panelda,
`LangSwitch`/`ThemeToggle` ostida — lekin yorliqsiz kichkina `select` bo'lgani
uchun bezakdek ko'rinardi va topilmasdi.

- `ScopeSwitcher` **butunlay qayta yozildi**: native `<select>` o'rniga
  `LangSwitch` dagidek **tugma + ochiladigan ro'yxat**. Sabab: yon panelda
  yalang'och `select` yorliqdek o'qiladi — tepasidagi dumaloq tugmalar yonida
  unda "bosiladigan" belgisi yo'q, va mijoz "bosilmayapti" deb xabar berdi.
  Endi sarlavhali yorliq, o'q belgisi, tashqariga bosilganda yopilish va
  "Hamma filiallar" holatida rangli fon bor.
- Avval tanlashni xabar ichiga ham qo'ygandim; **mijoz talabi bilan olib
  tashlandi**. Uning fikri to'g'ri: linza bitta joyda yashashi kerak, aks holda
  panelda ikkita bir xil boshqaruv bor degan taassurot qoladi. Xabar endi
  qayerga qarashni aniq aytadi ("chap paneldagi «Filial» ro'yxati").

### Saboq
Bo'limni **rolga qarab** yashirish va **kontekst yetishmaganiga** qarab
yashirish — ikki xil narsa. Ikkinchisi hech qachon butun sahifaga
qo'llanmasligi kerak: faqat kontekstga muhtoj bo'lakka.

⚠️ Sabab hali turibdi: **`Filial nomi`** — mening sinovimdan qolgan bo'sh
filial (0 buyurtma, 0 kuryer, 0 bron). U turgani uchun panel ko'p filialli
rejimda ochiladi. Mijozdan o'chirishga ruxsat so'raldi.


---

## 2026-07-31 (35) — 🐛 Bo'sh `ObjectID` JSON'da truthy bo'lib chiqdi

**Mijoz xabari**: "almashtirgich bosilmayapti, Maracanda degan yozuv turibdi".

### Sabab
Go'da **`json:",omitempty"` massivlarga ta'sir qilmaydi**, `primitive.ObjectID`
esa `[12]byte`. Demak qo'yilmagan `branchId` brauzerga
**`"000000000000000000000000"`** bo'lib boradi — JS'da **truthy**.

3-bosqichda men yozgan `pinned: !!me?.branchId` shu sababli **har doim rost**
edi: har bir admin "filialga biriktirilgan menejer" deb hisoblanib,
`ScopeSwitcher` bosilmaydigan yorliqni ko'rsatardi. Filial esa `"0000..."` id
bo'yicha topilmagani uchun yorliqda brend nomi ("Maracanda") chiqqan —
mijoz ko'rgan aynan shu.

Sinab tasdiqlandi:
```
bo'sh ObjectID JSON'da: {"branchId":"000000000000000000000000"}
```

### Tuzatish
- `lib/id.ts`: **`hasId()`** va **`realId()`**. Nomlangan yordamchi kerak,
  chunki xato **har bir call site'da to'g'ridek ko'rinadi**.
- Tuzatilgan joylar: `adminScope` (`pinned` va filial tanlash),
  `/admin/orders` (`courierId` — biriktirilmagan buyurtma biriktirilgandek
  ko'rinardi va "kuryer tanlang" ajratmasi ishlamasdi), `OrderReceipt`
  (u yerda ilgari qo'lda solishtirish qo'yilgan edi — endi umumiy yordamchi).
- CLAUDE.md ga tuzoq sifatida yozildi.

### Yo'l-yo'lakay
`ScopeSwitcher` native `<select>` dan `LangSwitch` dagidek **tugma + ochiladigan
ro'yxat** ga o'tkazildi: yon panelda yalang'och select yorliqdek o'qiladi.
(Bu almashtirish o'zi muammoni hal qilmagan bo'lardi — asl sabab yuqoridagi
`pinned` edi.)


---

## 2026-07-31 (36) — CRM: segmentlar, mijoz kartochkasi, fikr va shikoyatlar ✅

Mijoz "CRM qilib bera olasanmi?" deb so'radi. Aniqlashtirilgach ma'lum bo'ldiki,
poydevor **allaqachon bor** edi (tasdiqlangan telefonli mijozlar, buyurtma
statistikasi, ballar, promokod cheklovlari, SMS shlyuzi) — noldan qurish emas,
ustiga qo'yish kerak edi. Kelishilgan qamrov: **segmentlar + kartochka** va
**fikr/shikoyatlar**. SMS kampaniyalar **ataylab qoldirildi** (pul turadi,
Eskiz moderatsiyasi va rozilik/obunani bekor qilish kerak — alohida qaror).

### CRM-A: segmentlar
`handlers/crm.go` — hisoblanadi, saqlanmaydi. Har segment **bitta jumlada
tushuntiriladi**, chunki egasi ta'riflay olmaydigan guruhga hech nima
yubormaydi: uxlab qolgan (60 kun), yo'qolgan (180), tug'ilgan kun (7 kun
ichida), doimiy (≥3 buyurtma, oxirgisi 30 kun ichida), yangi, buyurtmasiz.

- **Mijoz bir vaqtda bir nechta segmentda** bo'lishi mumkin: "uxlab qolgan VIP"
  aynan e'tibor talab qiladigan juftlik, uni bitta yorliqqa siqib bo'lmaydi.
- **VIP nisbiy — eng yuqori 10%**. "1 000 000 so'm" somsa do'koni va to'yxona
  uchun boshqa narsa, va narxlar oshgani sari jimgina eskiradi. Baza 10
  kishidan kam bo'lsa VIP umuman berilmaydi.
- Chegaralar nomlangan konstantalar va panelda ta'rifi ko'rsatiladi.

### CRM-A: kartochka
`user` ga `birthday` / `tags` / `note` / `source`.
- **Tug'ilgan kun `MM-DD`, yilsiz**: tabriklash uchun kun yetarli, yil esa
  restoranga kerak emas va so'ralsa javob berish kamayadi. Sana kiritilsa yil
  tashlanadi, `02-31` rad etiladi.
- **Ism va telefon paneldan tahrirlanmaydi**: telefon SMS bilan tasdiqlangan
  va butun hisob shunga bog'langan.
- **Manba avtomatik** (birinchi buyurtma QR bo'lsa `qr`, aks holda `site`),
  faqat bo'sh bo'lsa yoziladi — operator tuzatishi ustiga yozilmaydi.
- Teglar mavjudlaridan taklif qilinadi (`GET /admin/tags`) — aks holda "VIP"
  va "VIP " ikki guruh bo'lardi.

### CRM-B: fikr va shikoyatlar
`feedback` kolleksiyasi. Baho **kuzatuv sahifasida** so'raladi — mijoz allaqachon
qarab turgan joyda, yetkazilgandan keyin. Ertasiga email emas (o'shanda javob
berishmaydi) va yetkazilishdan oldin emas (u hech nimani baholamaydi).
- **Yaxshi baho bir bosishda ketadi, izoh maydoni faqat past bahoda ochiladi** —
  eshitish kerak bo'lgan odam aynan norozi mijoz.
- Bitta buyurtmaga bitta baho. Buyurtma hisobga tegishli bo'lsa **egasining
  tokeni majburiy**: raqamni bilish buyurtmaga *qarash* uchun yetarli, lekin
  uning nomidan gapirish uchun emas.
- **Yopishda "nima qilindi" majburiy.** Yozuvsiz "ko'rib chiqildi" — ro'yxatni
  chiroyli qiladi-yu restoranni aqlliroq qilmaydi.
- Panel **javobsiz shikoyatlardan ochiladi**, o'rtacha bahodan emas: o'rtacha —
  xursand yoki xafa bo'ladigan raqam, javobsiz shikoyat esa ketayotgan mijoz.
- Yangi segment **"norozi"**: 60 kun ichida javobsiz shikoyat qoldirgan. Bu
  yagona segment mijozning emas, **restoranning o'z xatti-harakati** haqida —
  shuning uchun ro'yxatda birinchi turadi.

**Tekshirildi**: segmentlar haqiqiy bazada to'g'ri (VIP yo'q — 10 kishidan kam) ✓ ·
teglar takrori/bo'shi tozalanadi ✓ · `02-31` rad ✓ · segment va teg filtrlari ✓ ·
baho: begona odam rad etildi, egasi o'tdi, ikkinchi urinish 409 ✓ · izohsiz
yopish rad ✓ · izoh bilan yopildi va kim yopgani yozildi ✓ · shikoyat ochiq
bo'lsa mijoz "norozi" segmentiga tushdi, yopilgach chiqdi ✓ · `go vet`,
`gofmt`, `tsc`, `next lint` — toza ✓

Sinov ma'lumotlari o'chirildi (haqiqiy mijozga yozilgan sinov izohlarim ham).


---

## 2026-07-31 (37) — Kuryer xaritasi, qo'ng'iroq tugmasi, va naqd pul hisobi 🐛

Mijozning uchta kuzatuvi. Uchinchisi — haqiqiy kamchilik.

### 1. Kuryer sahifasida xarita tanlovi
Ilgari **bitta qattiq havola** (2GIS) edi. Endi tugma pastdan chiqadigan varaq
ochadi va uchta variant beradi (Yandex / Google / 2GIS) — mavjud
`RouteButtons` komponenti orqali. Sabab: kuryerning o'zi biladigan navigator
uni tezroq yetkazadi. Varaq pastdan chiqadi, chunki bu — eshik oldida, ko'pincha
bir qo'l bilan ushlangan telefon.

### 2. Buyurtmalar ro'yxatida kuryerga qo'ng'iroq
`🛵 Ism` qator ichidagi **oddiy matn** edi — ism bor, lekin buyurtmani olib
ketayotgan odamga yetib borish yo'li yo'q. Endi alohida `tel:` tugmasi
(telefon raqami kuryerlar ro'yxatidan olinadi).

### 3. "Qo'lidagi naqd" — mijoz haq edi
Bu raqam **yetkazilgan naqd buyurtmalar yig'indisi** edi, ya'ni topshirish
tushunchasi umuman yo'q edi: u faqat o'sardi va ikkinchi kundan boshlab hech
nima anglatmasdi. Kuryer hamma pulni topshirsa ham eski summa turaverardi.

**Tuzatish — `courier_settlement` ledgeri**:
- `qo'lidagi = jami yig'ilgan − topshirilgan`. Hammasini topshirgan kuryer
  **nol** ko'radi.
- **Hisoblagichni nolga tushirish emas, yozuv qo'shish**: "Aziz o'tgan seshanba
  qancha topshirgan edi?" degan savol javobsiz qolmasligi kerak.
- Har yozuvda **kim qabul qilgani** bor — nomsiz naqd topshiruv aynan keyin
  bahsga aylanadigan yozuv.
- Admin kartochkasida katta "qo'lidagi naqd" raqami, "qabul qilish" tugmasi
  (summa oldindan to'ldirilgan, qisman topshirish uchun tahrirlanadi) va
  oxirgi topshiruvlar ro'yxati. Amallar jurnaliga ham tushadi.
- Kuryer ilovasida endi **`cashInHand`** ko'rsatiladi. Ilgari "bugungi
  yig'ilgan" turardi: u pul topshirilgandan keyin ham ekranda qolardi va
  kechagi qarzni ko'rsatmasdi — ikki tomonlama noto'g'ri.
- Davr kesimidagi raqam qoldi, lekin yorlig'i **"yig'ilgan naqd"** ga
  o'zgartirildi — u haqiqatan shu.
- Topshirilgan summa yig'ilganidan oshsa **manfiy emas, nol** ko'rsatiladi:
  bu buxgalteriya xatosi, restoran kuryerdan qarzdor degani emas.

**Tekshirildi**: 811 000 yig'ilgan → 100 000 topshirildi → **qo'lida 711 000** ✓ ·
yozuvda kim qabul qilgani va izoh ✓ · nol summa rad ✓ · `go vet`, `gofmt`,
`tsc`, `next lint` — toza ✓

Sinov ma'lumotlari o'chirildi.


---

## 2026-07-31 (38) — To'rtta xato: qidiruv havolasi, marshrut yo'nalishi, qo'ng'iroq tugmasi, hidratsiya 🐛

### 1. Fikrlar → buyurtma havolasi bo'sh ro'yxatga olib borardi
`/admin/orders?q=RAQAM` havolasi ishlamasdi, chunki sahifa **URL dagi `q` ni
umuman o'qimasdi** (`useState("")`). Ustiga-ustak standart filtr **"Faol"** —
fikr esa doim **yetkazilgan** buyurtma haqida, ya'ni topilsa ham ko'rinmasdi.
- `q` endi URL dan olinadi.
- Qidiruv bilan kelinganda filtr **"Hammasi"** dan ochiladi: bitta buyurtma
  qidirilyapti, va u odatda tugagan buyurtma.

### 2. Kuryer varag'ida "Restoranga yo'l" yozilgan edi
`RouteButtons` mijozni restoranga yo'naltirish uchun yozilgan va sarlavhasi
qattiq kodlangan edi. Kuryerga esa **teskari yo'nalish** kerak. Endi sarlavha
parametr: standarti avvalgidek, kuryer ilovasi **"Mijozga yo'l"** beradi.

### 3. Kuryerga qo'ng'iroq hali ham tugmaga o'xshamasdi
Ro'yxatdagisi rangsiz chegara bilan matnga o'xshab turardi — endi brend rangli,
telefon ikonkasi bilan. **Va chekda kuryer umuman yo'q edi**: bitta buyurtma
bilan ishlash uchun ochiladigan ekranda uni olib ketayotgan odamga yetib borish
yo'li bo'lishi kerak. Chekka kuryer qatori + qo'ng'iroq tugmasi qo'shildi
(telefon kuryerlar ro'yxatidan olinadi — buyurtmada faqat **ism** saqlanadi,
ataylab: hisob o'chirilsa ham chek o'qilishi kerak).

### 4. Hidratsiya xatosi — `ThemeToggle`
Ikonka `mounted` bilan qo'riqlangan edi, lekin **`aria-label` va `title` yo'q**.
Server yorug' temani, brauzer esa (bosh qismidagi inline skript allaqachon
qo'ygan) qorong'ini ko'rsatardi — faqat yorliqlarning farqi ham React'ga butun
daraxtni tashlab yuborish uchun yetarli.
Endi `dark = mounted && theme === "dark"` — **har bir temaga bog'liq atribut**
mount'ni kutadi. Tekshirildi: server HTML'ida oy ikonkasi va "Yorug'" yorlig'i,
ya'ni birinchi klient renderi bilan aynan bir xil.

**Tekshirildi**: `tsc`, `next lint`, `npm run build` — toza ✓ · SSR chiqishi
tekshirildi ✓


---

## 2026-07-31 (39) — Hidratsiya xatosi: haqiqiy sabab topildi 🐛

O'tgan safargi tuzatish (`mounted` bayrog'i) **yordam bermadi** — mijoz xatoni
qayta yubordi. Sabab boshqa joyda edi.

### Nima bo'layotgan edi
`(site)/layout.tsx` da `<Suspense>` bor (u `TableProvider` uchun, chunki u
`useSearchParams` ishlatadi) va u **Header'ni kech gidratlaydi**. `ThemeProvider`
esa root layout'da — Suspense'dan **tashqarida**. Ketma-ketlik:

1. Root gidratlanadi → `ThemeProvider` effekti ishlaydi → `mounted = true`,
   tema `dark` bo'ladi.
2. **Keyin** Suspense chegarasi gidratlanadi va `ThemeToggle` ni
   `mounted = true` bilan chizadi.
3. Server HTML esa yorug' temani chizgan edi → nomuvofiqlik.

Ya'ni **ota-provayderga tegishli `mounted` bayrog'i bu yerda hech qachon
ishlamaydi**: chegara gidratlanguncha u allaqachon rost bo'lib ulguradi.

### Tuzatish
`ThemeToggle` dan React holati **butunlay olib tashlandi**: ikkala ikonka ham
doim chiziladi, qaysi biri ko'rinishini **CSS hal qiladi** (`dark:hidden` /
`dark:block`), `<html class="dark">` ni esa bosh qismidagi inline skript
birinchi bo'yashdan oldin qo'ygan. Markup server va klientda **aynan bir xil**,
to'g'ri ikonka darhol ko'rinadi, chaqnash yo'q.
`aria-label`/`title` ham barqaror: holatni emas, **amalni** nomlaydi
("Mavzuni almashtirish") — holatga bog'liq yorliq server bilan kelisha olmaydi.

### Yo'l-yo'lakay: savat nishonchasi ham xuddi shunday edi
`CartProvider` ham Suspense'dan tashqarida, ya'ni savatda mahsulot bo'lsa
nishoncha aynan shu xatoni bergan bo'lardi. Mijoz hali sezmagan (savati bo'sh
edi). Tuzatish boshqacha: bayroq **komponentning o'ziga** tegishli
(`Header` ichida `useState` + `useEffect`) — u o'z gidratatsiyasi paytida
yolg'on bo'ladi, serverga mos tushadi, keyin darhol rostga o'tadi.

### Saboq (CLAUDE.md ga yozildi)
`<Suspense>` ichidagi komponentda **ota-provayderning "mounted" bayrog'iga
tayanib bo'lmaydi**. Ikki yo'l: holatga bog'liq bo'lmagan markup (CSS hal
qiladi), yoki bayroq aynan shu komponentniki bo'lsin.


---

## 2026-07-31 (40) — Bron sahifasida `null.length` xatosi 🐛

**Xato**: `Cannot read properties of null (reading 'length')` — `bron/page.tsx`.

### Sabab
Go'da **nil slice JSON'da `[]` emas, `null`** bo'lib chiqadi. Zal xaritasi hech
qachon chizilmagan filial (`yunusobod`) tanlanganda `booking.tables` `null`
bo'lib kelardi, sahifa esa `booking.tables.length` ni o'qirdi.

Bu 2-bosqichda bron filialga bog'langanda paydo bo'lgan: ilgari faqat bitta
xarita bor edi va u har doim to'ldirilgan edi.

### Tuzatish — ikkala tomondan
- **Backend (asosiy)**: `bookingSettings()` endi `Tables` va `Shapes` ni bo'sh
  massivga aylantiradi. Ro'yxat kutilgan joyda `null` qaytaradigan API har bir
  iste'molchini himoyalanishga majbur qiladi — biri esa unutdi.
- **Frontend (himoya)**: `const tables = booking?.tables ?? []`. Eski serverga
  tushib qolsa ham sahifa yiqilmasin.

Tekshirildi: xaritasiz filial endi `"tables":[],"shapes":[]` qaytaradi ✓ ·
`/bron` ikkala filialda ham ochiladi ✓

⚠️ Bu **`omitempty` tuzog'ining ikkinchi ko'rinishi** (birinchisi — bo'sh
`ObjectID` truthy string bo'lib chiqishi, 35-yozuv). Go'ning JSON marshalling
odatlari bu loyihada ikki marta tishladi.


---

## 2026-07-31 (41) — Mijoz sahifasida kuryerga qo'ng'iroq tugmasi 🐛

38-yozuvda men **noto'g'ri ekranni** tushungan ekanman: admin panelidagi
buyurtmalar bo'limiga tugma qo'yganman, mijoz esa **o'z buyurtmasini kuzatadigan
sahifani** (`/order/<raqam>`, xarita tepasi) nazarda tutgan ekan.

U yerda "Kuryerga qo'ng'iroq" tagi chizilgan **matn havolasi** edi. Endi to'liq
tugma: brend rangi, telefon ikonkasi, telefonda barmoq bilan bosishga yetarli
baland. Sabab: bu tugmani bosayotgan odam odatda eshik oldida turib "ovqatim
qani?" deb o'ylayapti — u bosiladigan eng ko'zga tashlanadigan narsa bo'lishi
kerak.

Ko'rinish shartlari o'zgarmadi: faqat buyurtma **"Yo'lda"** bosqichida va kuryer
biriktirilganda (mijoz kuryerning raqamini boshqa paytda ko'rmasligi kerak).
Admin paneliga qo'yilgan tugmalar ham qoldi — ular dispetcher uchun foydali.

---

# 2026-07-31 — KUN XULOSASI

Uzun kun: **ikkita katta xususiyat bloki + bitta CRM + sakkizta xato tuzatish**.
Tafsilotlar yuqoridagi 28–41 yozuvlarda; bu yerda faqat nima qilinganini topish
uchun ro'yxat.

## 2026-08-01 (42) — Ishchilar: davomat, ish grafigi va oylik hisob-kitobi ✨

Kuryerdan keyingi ikkinchi rol: **ishchi** (oshpaz, ofitsiant, kassir). Alohida
PWA `/staff`, panelda `/admin/staff` va `/admin/payroll`.

**Nima qilindi**
- **`/staff` PWA** (o'z manifest'i va service worker'i, `/kuryer` dan mustaqil):
  bitta katta tugma — "Ishga kirish" / "Ishdan chiqish". Uch tab: Bugun
  (kirish/chiqish soati, grafik bilan solishtirish, kechagi/haftalik/oylik
  taqqoslash), Kalendar, Oylik.
- **Geolokatsiya majburiy**: tugma faqat filial manzilidan `branch.staffRadiusM`
  metr ichida ishlaydi. Qoida **serverda** (`geofenceBlocked`), ilova faqat
  oldindan "N m uzoqdasiz" deb ogohlantiradi. Har bosish qayerda bosilgani
  bilan yoziladi (`shift.inAt/outAt`, masofasi bilan) — "ilova uydan ham
  kiritdi" degan bahsni faqat saqlangan koordinata hal qiladi.
- **Ish grafigi** (`staff.schedule`, har hafta kuni uchun `start`/`end`/`isOff`):
  sistema shu grafikka qarab har kunni **kam ishlangan / grafik bo'yicha /
  ko'p ishlangan / grafikdan tashqari / ishga chiqilmagan** deb belgilaydi
  (±15 daqiqa bag'rikenglik bilan). Kechasi yopiladigan smena (18:00→02:00)
  ham to'g'ri hisoblanadi.
- **Kalendar** (`components/staff/AttendanceCalendar.tsx`) — ikkala tomon uchun
  bitta komponent, ranglar bitta joydan (`lib/attendance.ts`). Kunga bosilsa
  grafik/amaldagi vaqt, kirish-chiqishlar va ortiqcha/kam soat ko'rinadi.
- **Oylik**: `payMode` — soatbay / smenabay / oylik maosh; `payPeriod` — kunlik /
  10 kunlik / 15 kunlik / oylik. Oylik maosh **o'sha oyning grafik kunlariga**
  bo'linadi va chiqilgan kun uchun beriladi — o'rtada ishga kirgan odam ham
  to'g'ri hisoblanadi.
- **`/admin/payroll` (Hisob-kitob)**: har ishchi **o'z davri** bo'yicha —
  hisoblangan, to'langan, to'lash kerak. To'lov `staff_payment` da **yozuv**
  sifatida (hisoblagichni nolga tushirish emas), qaysi davrni yopgani va kim
  bergani bilan.
- **Qo'lda tuzatish**: telefon o'chib qolsa yoki chiqish unutilsa admin
  smenani qo'shadi/tahrirlaydi; yozuv `editedBy` bilan imzolanadi va jurnalga
  tushadi (`staff.create/update/delete/shift/pay`).
- Filial sozlamalarida yangi maydon: **kirish/chiqish masofasi** (standart 50 m,
  20 m dan kichik qiymatda ogohlantirish chiqadi).
- i18n: `admin.ts` ga `staff` bo'limi uch tilda; navigatsiyaga "Ishchilar" va
  "Hisob-kitob".

**Qarorlar / tuzoqlar**
- **50 metr, 5 emas**: telefon GPS'i ochiq havoda 10–30 m, bino ichida undan
  ham yomon aniqlikda ishlaydi — oshxona esa aynan bino ichida. Radius xato
  chegarasidan kichik bo'lsa u firibgarni ushlamaydi, halol ishchini eshik
  oldida qoldiradi. Shuning uchun standart 50 m, lekin maydon sozlanadigan
  (0 = tekshiruv o'chiq).
- **`staffRadiusM: 0` "tekshiruv yo'q" degani** — shuning uchun eski filiallarga
  `EnsureStaffDefaults` 50 yozadi, va migratsiya yaratadigan birinchi filialga
  ham qiymat qo'lda beriladi: Go nol qiymatni **tushirib qoldirmaydi**, yozadi,
  ya'ni `$exists: false` backfill uni ko'rmaydi.
- **Kun holati punchlarga qarab aniqlanadi, daqiqalarga emas**: kirib darhol
  chiqqan odam 0 daqiqa ishlagan, lekin "ishga chiqmagan" emas.
- **Smena boshlangan kunga yoziladi**: 00:40 da chiqqan oshpaz seshanbani
  yopadi, chorshanbani ochmaydi. Serverda `TZ=Asia/Tashkent` bo'lishi shart,
  aks holda butun kalendar UTC'da chiziladi.
- **Ikki marta kirish rad etiladi** (birlashtirilmaydi): ikkita ochiq smena
  kalendardagi har soatni ikkilantirardi.
- Ishchi hisobi o'chirilsa **smenalar qoladi** — oylik tarixi va "o'sha kechada
  kim bor edi" savoli hisobdan uzoq yashaydi.
- Sana yorlig'i `01.08` ko'rinishida: `uz-UZ` locale'da "long/short" oy
  **"M08"** bo'lib chiqadi, shuning uchun oy nomlari lug'atga yozildi.

**Sinov**: `staff_smoke` bazasida uchdan-uchgacha — 2 km dan kirish rad etildi,
filialda qabul qilindi (2.2 m), ikkinchi kirish 409, chiqish yozildi; 09:00–24:00
kun "ko'p ishlangan" (900/720 daq, 300 000 so'm), 11:00–17:00 "kam ishlangan";
oylik maosh 6 000 000/27 = 222 222 kuniga; uch xil to'lov davri bitta
hisob-kitob sahifasida. Brauzerda (Playwright) panel va PWA — konsol xatosiz.

---

## 2026-08-01 (43) — `/ishchi` → `/staff`, va butun loyihada 24 soatlik vaqt 🐛

**1. Marshrut nomi.** Ishchi PWA `/ishchi` dan **`/staff`** ga ko'chirildi:
papka, manifest `start_url`/`scope`, service worker qobig'i va scope'i,
paneldagi ko'rsatma matni (uz/ru/en), hujjatlar. SW kesh nomi ham
`ishchi-v1` → `staff-v1` (eski qobiq keshda qolmasligi uchun). Backend API
allaqachon `/api/v1/staff/...` edi — ziddiyat yo'q.

**2. AM/PM yo'q qilindi.** Hamma joyda `toLocaleTimeString` / `toLocaleString`
ishlatilardi — ular **qurilma tiliga ergashadi**, ya'ni inglizcha telefonda
smena "1:16 PM" bo'lib chiqardi. Endi bitta joyda, qo'lda:

- `lib/format.ts` → `formatTime` (`HH:MM`), `formatDate` (`DD.MM.YYYY`),
  `formatDateTime`. `orderFlow.ts` dagi eski `formatDateTime` shularga
  yo'naltirildi (o'nlab sahifa undan import qiladi).
- Almashtirilgan joylar: ishchi kalendari va ilovasi, bronlar, buyurtmalar
  ro'yxati ("yangilandi"), kuryer ilovasi, fikrlar, promokod tarixi, kuryer
  kartochkasi, mijozlar ro'yxati va kartochkasi, profil.

**Nega qo'lda, `Intl` bilan emas** — ikkita sabab, ikkalasi ham shu loyihada
tishlagan: (a) 24 soatlik format qurilma tiliga bog'liq bo'lmasligi kerak;
(b) `Intl` Node (SSR) va brauzerda har xil ajratgich berib **hidratsiya
xatosini** keltirib chiqaradi — `formatPrice` ham aynan shu sababdan `Intl` dan
qochadi.

**Qolgan cheklov**: `<input type="time">` (ish grafigi va smenani tuzatish
formalarida) — brauzerning o'z vidjeti, uni HTML orqali 24 soatlikka majburlab
bo'lmaydi. U qurilma til sozlamasiga ergashadi; o'zbek/rus tilidagi telefonda
24 soatlik, US English'da AM/PM ko'rsatadi. Saqlanadigan qiymat esa har doim
`HH:MM` (24 soatlik), ya'ni faqat ko'rinish farq qiladi.

**Sinov**: `formatTime` en-US jarayonida 00:07 / 09:05 / 12:00 / 13:16 / 23:47
uchun to'g'ri; loyihada bironta `toLocaleTimeString`/`toLocaleString` qolmadi;
`/staff` brauzerda ochilib, "Ishga kirish" bosildi va SW `/staff` scope'ida
ro'yxatdan o'tdi.

---

## 2026-08-01 (44) — Birinchi prod deploy: traderbot.uz 🚀

VPS `173.249.8.13` (Ubuntu 24.04), papka `/opt/jomaxuz`, domen
**traderbot.uz** Cloudflare orqali. Sayt jonli, Let's Encrypt sertifikati bilan.

**Serverda nima bor edi** (ehtiyot bo'lish kerak bo'lgan narsalar): `filmorauz.net`
va `api.filmorauz.net` systemd servislari, `tradebot` Telegram boti, ufw
(DROP policy), Docker esa **umuman yo'q** edi.

**Yo'lda chiqqan uchta xato**

1. **Portlar to'qnashardi** — `filmorauz-backend` 8080 da, uning frontendi
   3000 da. Shablonning standart portlari aynan shular. Endi `.env` dan
   sozlanadi (`FRONTEND_PORT`/`BACKEND_PORT`/`MONGO_PORT`); bu deployment
   3100 / 8090 / 27018 da.
2. **Vaqt mintaqasi jimgina UTC bo'lardi** — alpine'da `tzdata` yo'q, Go
   `TZ=Asia/Tashkent` ni hal qila olmay `time.Local` ni UTC qoldiradi va
   **xato bermaydi**. Har bir ishchi smenasi besh soat oldingi kunga
   yozilardi. `import _ "time/tzdata"` + `tzdata` paketi + boot'da mintaqani
   log qilish. Host'ning tizim vaqtiga tegilmadi (Berlin'da qoldi) — boshqa
   saytlar bor.
3. **`http2 on;` nginx 1.24 da yo'q** — u 1.25.1 da paydo bo'lgan, Ubuntu
   24.04 esa 1.24 beradi va nginx umuman ishga tushmaydi. `listen ... ssl
   http2` shakliga qaytarildi. Konfiguratsiya sinovi (`nginx -t`) buni
   reload'dan oldin ushladi, ya'ni mavjud saytlar bir soniya ham yiqilmadi.

**Sertifikat**: certbot webroot (`/var/www/certbot`), `certbot.timer` faol,
ACME yo'li internetdan tekshirildi — avtomatik yangilanish ishlaydi.

**Deploy key ishlatib bo'lmadi**: `jomaxuz` tashkiloti siyosati deploy
key'larni taqiqlagan. Shuning uchun serverda hech qanday GitHub kaliti
saqlanmaydi — yangilash noutbukdan **SSH agent forwarding** bilan:
`ssh -A root@173.249.8.13 /opt/jomaxuz/deploy.sh`.

---

## 2026-08-01 (45) — QR bilan ishga kirish: aylanadigan kod 🔐

Savol: devorga QR ilib qo'ysak, ishchi uni rasmga olib uydan skaner qilsachi?
Javob: **bosma QR bunga umuman qarshi tura olmaydi** — u devorga yozilgan
parol, bir marta rasmga olingani bir yil ishlaydi. Shuning uchun kod qog'ozda
emas, **ekranda** turadi va har 30 soniyada o'zgaradi.

**Qanday ishlaydi**
- `handlers/kiosk.go`: kod = `HMAC-SHA256(branch.kioskSecret, branchId.step)`,
  `step = unix/30`. Serverda **hech narsa saqlanmaydi** — qaytadan hisoblab
  solishtiriladi (jadval ham, tozalash ham, restartda yo'qoladigan holat ham
  yo'q). ±1 qadam qabul qilinadi (~60s), solishtirish
  `subtle.ConstantTimeCompare` bilan.
- `/kiosk` — filialdagi planshetda ochiq turadigan ekran: katta QR, orqaga
  sanoq, wake lock, tarmoq uzilsa o'zi tiklanadi. Token havola orqali bir
  marta beriladi va manzil satridan darhol tozalanadi.
- Ishchi tomoni: QR — oddiy **`/staff?c=<kod>` havolasi**, ya'ni telefonning
  o'z kamerasi ochadi va alohida skaner kutubxonasi kerak emas (stol QR'i
  bilan bir naqsh). Ilova ochiq smenaga qarab kirish yoki chiqishni **o'zi**
  aniqlaydi.
- Planshet yo'qolsa — **"Kalitni almashtirish"**: `kioskVersion` oshadi va
  barcha eski ekran tokenlari o'ladi.

**Asosiy qaror**: kod joylashuvni **almashtirmaydi, to'ldiradi**. Rasmga
olingan kod uchun ham odam o'sha yerda turishi kerak; aldangan GPS uchun esa
ekrandagi kod kerak (Android'da "mock location" bepul va oson — geofence
yolg'iz o'zi ham qalqon emas). Ikkalasi ham `StaffClock` da tekshiriladi,
shuning uchun kodni 60 soniya ichida do'stiga yuborish ham yordam bermaydi.

**Tuzoq (takrorlangan)**: `kioskSecret`/`kioskVersion` filial formasidan
yozilmaydi — `AdminUpdateBranch` da `delete` qilinadi. Aks holda sozlamalarni
saqlash ularni nolga tushirib, ekran tokenini jimgina o'ldirardi. `soldOut`
bilan bir xil naqsh.

**Sinov**: kodsiz rad · soxta kod rad · **eskirgan (rasmga olingan) kod 70
soniyadan keyin rad** · to'g'ri kod, lekin 3 km uzoqdan rad · kalit
almashtirilgach eski ekran rad · filial sozlamalarini saqlash ekranni
buzmadi. Brauzerda: kiosk ekrani + telefondan skanerlash → smena ochildi.

---

## 2026-08-01 (46) — Deploy "tugadi" deganiga ishonib bo'lmasdi 🐛

Kiosk deploy'idan keyin sayt yangi commit'da ko'rinardi-yu, `/kiosk` **404**
qaytarardi: `git pull` o'tgan, `docker compose build` esa tugamagan.

Sabab — CI ning ssh chaqiruvida **keepalive yo'q edi**. Yig'ish bir necha
daqiqa davom etadi va uzun jim qoladigan paytlari bor; oradagi ulanish uzilib,
server skriptni SIGHUP bilan o'ldirgan.

Tuzatish ikki qatlamda:
1. `ServerAliveInterval` / `ServerAliveCountMax` — ulanish jim turgani uchun
   uzilmaydi.
2. **Yig'ilgan commit `/version.txt` da** chiqadi; deploy skripti ham, CI ishi
   ham jonli sayt **aynan shu commit'dan** ekanini tekshiradi. Ilgari tekshiruv
   200 kutardi — eski konteyner ham 200 qaytaradi, ya'ni tugallanmagan deploy
   muvaffaqiyatli ko'rinardi. Yolg'on yashil belgi yolg'on qizildan battar.

Yo'l-yo'lakay: commit matnida teskari qo'shtirnoq ishlatilgani uchun shell uni
buyruq sifatida bajarib yuborgan (lokal `docker compose build` ishga tushgan).
Commit matnlari endi heredoc bilan beriladi.

---

## Qurilgan xususiyatlar

| # | Nima | Yozuv |
|---|---|---|
| 1 | **Brend/filial 2-bosqich** — barcha oqimlar brend+filial kesimida, sayt linzasi, brend bo'yicha savat | 28 |
| 2 | **Brend/filial 3-bosqich** — "tugadi" bayrog'i, buyurtma raqamiga filial prefiksi, menejer huquqlari | 29 |
| 3 | **Admin parolini SMS bilan tiklash** + tiklash raqami | 30 |
| 4 | **Combo** (belgilangan to'plam) | 31 |
| 5 | **Chegirma mexanizmi** — promokod va aksiya bitta qoida, yagona narxlash quvuri | 32 |
| 6 | **Promokod sozlamalari** (holat, "kim ishlatgan") + **loyalty** (keshbek ballari) | 33 |
| 7 | **CRM** — segmentlar, mijoz kartochkasi, fikr va shikoyatlar | 36 |
| 8 | **Kuryer naqd puli** — topshirish ledgeri | 37 |

## Tuzatilgan xatolar

| Xato | Sabab | Yozuv |
|---|---|---|
| Sozlamalar sahifasi butunlay yo'qolgan | Kontekst yetishmasligini **butun sahifaga** qo'llagandim | 34 |
| Filial almashtirgichi bosilmasdi | Go'da bo'sh `ObjectID` → `"0000..."` → JS'da **truthy** | 35 |
| Fikrlardan buyurtmaga o'tilmasdi | URL dagi `q` o'qilmasdi + "Faol" filtri yetkazilganni yashirardi | 38 |
| Kuryer varag'ida "Restoranga yo'l" | Sarlavha qattiq kodlangan, yo'nalish teskari | 38 |
| Hidratsiya xatosi | `<Suspense>` Header'ni kech gidratlaydi, ota-provayder `mounted` allaqachon rost | 39 |
| Savat nishonchasi (oldindan topildi) | O'sha sabab | 39 |
| Bron sahifasida `null.length` | Go'da nil slice → JSON `null` | 40 |
| Mijoz sahifasida qo'ng'iroq tugmasi | Men noto'g'ri ekranni tuzatgandim | 41 |

## Takrorlangan tuzoqlar (CLAUDE.md ga yozildi)

1. **Go'ning JSON odatlari ikki marta tishladi**: nil slice → `null`, bo'sh
   `ObjectID` → truthy qator. Ikkalasi ham "yo'q" degan narsani "bor, lekin
   g'alati" qilib yuboradi.
2. **`<Suspense>` ichida ota-provayderning `mounted` bayrog'iga tayanib
   bo'lmaydi** — chegara gidratlanguncha u rost bo'lib ulguradi.
3. **Ish jarayoni**: eski server jarayoni yangi kod o'rniga javob berib turgani
   **ikki marta** noto'g'ri sinov natijasini berdi. `pgrep -f` o'z buyrug'ini
   ham topadi — `pgrep -x` va `ps -o lstart` bilan tekshirish kerak.
4. **Har matn almashtirishga `assert`**: `router.go` ga marshrut qo'shish
   jimgina o'tmagan va endpoint 405 qaytargan edi.

## Baza holati (kun oxirida)
19 buyurtma, 48 taom, 8 mijoz, 2 filial (`Maracanda`, `yunusobod`), 1 brend,
2 aksiya/promokod, 1 fikr, 0 combo.

**Mening** sinov ma'lumotlarim har bosqichdan keyin tozalangan (vaqtinchalik
adminlar, sinov brendi/filiallari, buyurtmalar, kodlar, ballar ledgeri,
haqiqiy mijozga yozilgan sinov izohlari ham). Yuqoridagi qo'shimcha yozuvlar —
**mijozning o'z sinovi** (3 buyurtma, 2 aksiya, 1 fikr), ular ataylab
qoldirilgan.

## Ochiq qolgan
- **To'lov**: checkout'da Payme/Click/Uzum tugmalari bor, lekin pul yechilmaydi
- **Deploy sinovi**: `docker-compose.prod.yml` hech qachon uchdan-uchgacha
  ishga tushirilmagan
- **Git**: repozitoriyada hali **0 ta commit**
- **SMS kampaniyalar**: CRM'da ataylab qoldirilgan (pul, moderatsiya, rozilik)
- Bazada `yunusobod` — mening sinovimdan qolgan bo'sh filial

---

## 2026-08-03 — Call-markaz (admin panel ichida) ✅

Admin panelga call-markaz operatori uchun ish stoli qo'shildi. **Alohida rol
yaratilmadi** — telefon ko'targan odam ikki daqiqadan keyin o'sha buyurtmani
tasdiqlaydigan odamning o'zi; ikkiga bo'lish ishning yarmini qilish uchun
chiqib-kirishni talab qilardi. `owner` ham, `manager` ham ko'ra oladi.

### Backend
- `models/call.go` — `call` kolleksiyasi: yo'nalish (kiruvchi/chiquvchi),
  raqam, natija, izoh, qayta qo'ng'iroq va'dasi, operator, davomiyligi.
  Natija ro'yxati: buyurtma / bron / ma'lumot / shikoyat / qayta qo'ng'iroq /
  bermadi / javob bermadi / keraksiz.
- `handlers/callcenter.go` — **`GET /admin/lookup?phone=`**: bitta so'rovda
  mijoz, jarayondagi buyurtmalari, oxirgi buyurtmalari, doim buyurtma
  qiladigan taomlari, manzillari, bronlari, javobsiz shikoyatlari va oldingi
  qo'ng'iroqlari. Raqam `+998 90 123 45 67` ko'rinishida ham topiladi.
- **`POST /admin/orders`** — operator buyurtmasi. `CreateOrder` ning ichi
  `composeOrder` ga ajratildi: sayt va telefon **bitta** narxlash quvurini
  yuritadi. Bazada bo'lmagan raqamga hisob avtomatik ochiladi
  (`authProvider: "operator"` — SMS tekshiruvidan o'tmagan, `source: "phone"`),
  operator yozgan manzil profilga saqlanadi (takrori qo'shilmaydi).
  Buyurtmada `takenBy` — kim yozgani chekda ko'rinadi.
- **`POST /admin/orders/quote`** — checkout'ning aynan o'zi, lekin mijoz
  *nomlanadi* (token bilan emas): operator jami summani telefonda aytishi
  uchun. `OrderQuote` `quote()` ga ajratildi.
- `handlers/calls.go` — jurnal: `GET/POST/PUT /admin/calls` (qidiruv, natija,
  yo'nalish, operator, sana oralig'i, **ochiq qayta qo'ng'iroqlar**) va
  `GET /admin/calls/stats` (standart — bugun).
- Jurnal **tahrirlanadi** (amallar jurnalidan farqli): qo'ng'iroq odam
  gapirayotganda yoziladi, tuzatib bo'lmasa operator umuman yozmay qo'yadi.

### Frontend
- `/admin/calls` — ish stoli: raqam qidiruv → mijoz kartochkasi → natija.
  Tepasida bugungi to'rt raqam: qo'ng'iroqlar, buyurtmaga aylangani,
  konversiya, ochiq qayta qo'ng'iroqlar (kechikkani alohida).
- `components/admin/CallerCard.tsx` — bloklar **operator o'qish tartibida**:
  javobsiz shikoyat → hozir oshxonadagi buyurtma → kim ekani va nima
  buyurtma qilishi → tarix.
- `components/admin/OperatorOrderModal.tsx` — telefon orqali buyurtma:
  qidiruvli menyu, variantlar, manzil (xarita + saqlangan manzillar),
  promokod, ball, va **serverdan kelgan** jami summa.
- `components/admin/CallLog.tsx` — filtrlar + "Qo'ng'iroq qilindi" tugmasi.
- Lug'at `calls` bloki uz/ru/en, nav'ga "Call-markaz", chekda "Telefon orqali
  qabul qildi: …", amallar jurnalida `order.create`.

### Tekshirildi (lokal, haqiqiy baza)
- Noma'lum raqam → bo'sh kartochka; mavjud mijoz → 13 buyurtma, segmentlar,
  jarayondagi buyurtma, top 5 taom.
- Qayta qo'ng'iroq vaqtsiz → 400; ochiq qayta qo'ng'iroqlar filtri; stats.
- Yangi raqamga buyurtma → hisob ochildi, manzil saqlandi, qo'ng'iroq yozuvi
  buyurtmaga bog'landi, amallar jurnaliga tushdi.
- Minimal buyurtmadan past → 400; noto'g'ri telefon → 400.
- Sinov yozuvlari bazadan tozalandi.

**Ochiq**: qo'ng'iroqlar hozircha **qo'lda** yoziladi (ATS/webhook integratsiyasi
yo'q) — mijozda ATS bo'lsa `POST /admin/calls` tayyor tayanch nuqta.

---

## 2026-08-03 — Onlayn to'lov: Payme, Click, Uzum ✅

Sozlamalardan kalitlar kiritiladi va butun tizim ishlaydi. Uchala protokol ham
**ikki mustaqil manbadan** tekshirib olindi (Click'ning o'z PHP kutubxonasi,
Payme rasmiy hujjati, Uzum uchun ikkita mustaqil ochiq implementatsiya).

### Backend
- `models/payment.go` — `payment_settings` (singleton, **alohida kolleksiya**:
  `restaurant` hujjati saytga to'liq qaytariladi) va `payment` daftari.
  Buyurtmada `paymentStatus`, `paidAt`, `queuedAt`.
- `handlers/payments.go` — umumiy qatlam: summa **buyurtmadan** olinadi,
  to'landi deb belgilash **idempotent** (unique `(provider, providerTxnId)` +
  holat bo'yicha qo'riqlangan yangilash), brauzerning qaytishi hech nimani
  o'zgartirmaydi.
- `handlers/paypayme.go` — JSON-RPC, 6 metod, `Basic Paycom:<kalit>`, tiyin,
  12 soat timeout, `-32504/-31001/-31003/-31007/-31008/-3105x`.
- `handlers/payclick.go` — Prepare/Complete, md5 imzo (ikki formula alohida),
  `-1/-2/-3/-4/-5/-6/-8/-9`.
- `handlers/payuzum.go` — check/create/confirm/reverse/status, Basic auth,
  tiyin, `10001..10009`, `99999`.
- `GET /payment-methods` — checkout faqat **to'liq sozlangan** tizimni
  ko'rsatadi. `GET /orders/{number}/pay` — bank havolasi (raqam bo'yicha,
  boshqa qurilmadan ham to'lash uchun).
- `queuedAt` migratsiyasi: eski buyurtmalar naqd edi → `createdAt`.

### Frontend
- Checkout: usullar serverdan olinadi, buyurtma yaratilgach **to'g'ridan-to'g'ri
  bank sahifasiga** o'tadi.
- `/order/{number}`: "To'lov kutilmoqda" + "To'lash" tugmasi, "To'lov qabul
  qilindi", "Qaytarildi".
- `/admin/orders`: to'lanmagan buyurtmada bir bosishlik tugma o'chiq
  (ro'yxatdan qo'lda o'zgartirish qoladi), holat nishonchalari.
- `/admin/settings` → "To'lov tizimlari": kalitlar (**hech qachon qaytarilmaydi**,
  bo'sh = saqlangani qoladi) + **kabinetga yoziladigan manzillar** nusxalash
  tugmasi bilan. Faqat owner.

### Tekshirildi (lokal, haqiqiy so'rovlar)
- Payme: noto'g'ri kalit → -32504; noto'g'ri summa → -31001; yo'q buyurtma →
  -31050; boshqa tizimning buyurtmasi → -31051; Create idempotent; ikkinchi
  parallel tranzaksiya → -31008; **12 soatlik timeout ishladi**; Perform
  takrori xato emas, o'sha natija; to'langandan keyin qayta to'lash rad etiladi.
- Click: buzuq imzo → -1; Prepare → `merchant_prepare_id`; Complete → to'landi;
  Complete takrori → -4.
- Uzum: noto'g'ri auth → 10001; noto'g'ri serviceId → 10006; noto'g'ri summa →
  99999; create/confirm takrorlari idempotent; reverse → `refunded`.
- Bo'sh kalit bilan saqlash kalitlarni **o'chirmadi** (callback eski kalit bilan
  ishlayverdi).
- `queuedAt` faqat to'lov tasdiqlangach qo'yildi.
- Sinov yozuvlari bazadan tozalandi.

### Tekshirish quroli — `cmd/paytest`
Provayderning o'zini o'ynaydi: kalitlarni bazadan o'qiydi, bank yuboradigan
chaqiruvlarni **aynan o'sha imzo bilan** yuboradi, javobni va oxirida
buyurtmaning bazadagi holatini ko'rsatadi. Merchant kabineti ham, tunnel ham
kerak emas.

```bash
go run ./cmd/paytest -order AB12-3456 -suite
```
`-suite` rad javoblarini ham tekshiradi (noto'g'ri kalit/imzo, summa, yo'q
buyurtma, takroriy chaqiruvlar, ikki marta to'lash). Uchala provayderda ham
hamma tekshiruv ✓ bo'ldi. DEPLOY.md ga to'liq bo'lim yozildi (kabinetga
yoziladigan manzillar, sandbox, tunnel, ishga tushirish ro'yxati).

**Ochiq / mijoz bilan aniqlanadigan**: kalitlar har bir restoranning o'z
kabinetidan olinadi. Uzum kabinetida `params` maydon nomlari, Payme'da esa
`account` maydoni sozlanadi — panelda "Buyurtma maydoni nomi" shu uchun bor
(standart `order_id`). Uzum checkout havolasi (`apelsin.uz/open-service`) va
Payme sandbox hosti mijozning shartnomasiga qarab farq qilishi mumkin.

---

## 2026-08-03 — POS integratsiyasi: iiko, Clopos, r_keeper ✅

Uchala tizimning hujjati o'rganildi (Clopos'ning to'liq OpenAPI spetsifikatsiyasi,
Payme uslubidagi rasmiy iiko metodlari, r_keeper uchun UCS hujjati + ochiq
implementatsiyalar). Qarorlar: **menyu bizda qoladi**, POS'ga faqat buyurtma
ketadi; buyurtma **tasdiqlanganda** yuboriladi; r_keeper adapteri yozildi,
ulanish mijoz chiqqanda hal qilinadi.

### Backend
- `internal/pos/` — bitta `Provider` interfeysi (`Ping`, `Products`,
  `SendOrder`, `OrderStatus`, `Cancel`) va uch adapter. Adapterlar bazani
  bilmaydi, `pos.Order` bizning atamalarimizda.
- `iiko.go` — token keshi (1 soat), `nomenclature` + `stop_lists`,
  `deliveries/create` va **asinxron tasdiqni kutish** (`commands/status`).
- `clopos.go` — `x-token`, sahifalangan mahsulotlar, `auto_order_accept`.
- `rkeeper.go` — XML, `GetRefData MENUITEMS` + `GetOrderMenu`, tiyin/mingdan bir
  birliklari, o'z-o'zidan imzolangan sertifikat uchun alohida transport.
- `models/pos.go` — `pos_settings` (filial bo'yicha, kalitlar qaytarilmaydi),
  `pos_mapping` (unique `branchId+menuItemId`), buyurtmada `pos` bloki.
- `handlers/pos.go` — sozlama, ping, mahsulotlar, bog'lash, yuborish.
  Yuborish buyurtmaning `pos.status` i bilan **idempotent**.
- `UpdateOrderStatus` da `confirmed` bo'lganda avtomatik yuboriladi va
  **hech qachon tasdiqlashni to'xtatmaydi**.

### Frontend
- `/admin/settings` → "POS tizimi": tizim tanlanadi, kalitlar kiritiladi,
  **"Ulanishni tekshirish"** nima bilan ulanganini aytadi. r_keeper tanlansa
  tarmoq ogohlantirishi chiqadi.
- `/admin/pos` — bog'lash ekrani: yuqorida "nechta bog'lanmagan", "faqat
  bog'lanmaganlar" filtri va **"Nomi bo'yicha moslash"** (faqat aynan bitta
  mos keladigan nom — noaniqlarini odam hal qiladi).
- Chekda POS holati va "Qayta yuborish".

### Tekshirildi
- iiko'ga soxta kalit bilan **haqiqiy** so'rov → "Login … is not authorized"
  (so'rov shakli to'g'ri). Clopos → 400 bilan sababi. r_keeper → "restoran
  tarmog'i ichidami?" degan aniq xabar.
- Lokal stub bilan uchdan-uchgacha: ping → "Stub Restoran", mahsulotlar
  ro'yxati, bog'lanmagan taom → **rad etildi** (nomi bilan), bog'langach →
  yuborildi (`iiko №1042`), qayta yuborish → **ikkinchi chek ketmadi**,
  "Tasdiqlash" → avtomatik yuborildi.
- Kassaga borgan tana tekshirildi: to'g'ri `productId`, miqdor, narx,
  `externalNumber`, `+998…` telefon, `DeliveryByClient`.

**Ochiq**: r_keeper'ni haqiqiy mijozga ulash — tarmoq masalasi (port + TLS
yoki VPN), yoki UCS bilan White Server shartnomasi. Clopos `integrator_id`
alohida so'raladi. iiko `paymentTypeId` onlayn to'langan buyurtmani yopish
uchun kerak — mijozning iiko sozlamalaridan olinadi.

---

## 2026-08-03 — Telefoniya: onlinePBX call-markazga ulandi ✅

Rasmiy OpenAPI spetsifikatsiyasi topildi (`api.onlinepbx.ru/api-scheme.yaml`,
v2.10.1) va o'qib chiqildi. Call-markazdagi "qo'ng'iroqlar qo'lda yoziladi"
degan ochiq nuqta yopildi.

### Backend
- `internal/pbx/onlinepbx.go` — S3 uslubidagi avtorizatsiya (`auth.json` →
  `key_id:key`, `x-pbx-authentication`), **kalit keshi** (uch kun yashaydi,
  faqat `isNotAuth` da yangilanadi), `call/now.json`, `mongo_history/search.json`,
  yozuv havolasi.
- `handlers/pbx.go` — webhook qabul qiluvchi: `call-start`, `call-user-start`,
  `call-answered`, `call-missed`, `call-transfer-answered`, `call-end`.
  Hammasi `pbxCallId` bo'yicha **bitta yozuvga** tushadi; operator yozgan
  natija/izoh hech qachon o'chirilmaydi.
- `GET /admin/calls/live` (ekran ochilishi), `POST /admin/calls/dial`
  (click-to-call), `GET /admin/calls/{id}/recording`, `PUT /admin/me/extension`.
- Webhook manzilidagi token generatsiya qilinadi va almashtirilishi mumkin;
  noto'g'ri token 200 bilan jimgina rad etiladi.

### Frontend
- `/admin/calls`: **"Qo'ng'iroq kelmoqda"** banneri va kartochka o'zi ochiladi
  (har 3 soniyada so'rov). Operator boshqa qo'ng'iroqni yozayotgan bo'lsa ekran
  egallanmaydi. Qidiruv yonida ☎ tugmasi (operator raqami bo'lsa).
- Jurnalda: "Natijasi belgilanmagan" belgisi, **"Faqat yozilmaganlar"** filtri
  (smena oxiridagi ro'yxat), "ATS" belgisi va **yozuvni tinglash**.
- `/admin/settings` → "Telefoniya": domen, API kalit, standart ichki raqam,
  ulanish tekshiruvi, **nusxalanadigan webhook manzili** va "oxirgi hodisa
  qachon keldi" qatori. `/admin/account` → "Mening ichki raqamim".

### Tekshirildi
- Beshta hodisa → **bitta yozuv**; telefon `+998 90 111 22 33` dan
  normallashdi; `ringing` holati; ichki raqam 101 → operator `yujo` topildi;
  davomiylik `dialog_duration` (72), `call_duration` (95) emas; yozuv bayrog'i;
  hangup sababi.
- Chiquvchi qo'ng'iroqda mijoz `callee` dan olindi.
- Javobsiz qo'ng'iroq → natija avtomatik `missed`.
- **Operator yozib bo'lgandan keyin kech kelgan `call-end` va qayta yuborilgan
  `call-start` natija/izohni o'chirmadi.**
- Noto'g'ri token → 200, lekin bazada hech nima yaratilmadi.
- Mavjud mijoz raqami → yozuvga ismi (`Yusuf`) avtomatik bog'landi.
- Haqiqiy onlinePBX API'ga ulanish sinaldi; xato xabari egaga tushunarli
  qilib tuzatildi ("domeni qabul qilmadi — domen va API kalitni tekshiring").

**Ochiq**: webhook maydon nomlari onlinePBX'ning integratsiya hujjatidan
(rasmiy OpenAPI'da webhook yo'q). Payload bardoshli o'qiladi, lekin haqiqiy
mijozda bir marta tekshirish kerak. Sinov yozuvlari bazadan tozalandi.

---

## 2026-08-04 — POS: Syrve va Poster qo'shildi ✅

To'qqizta so'ralgan kassa tizimi o'rganib chiqildi, natija `POS_INTEGRATIONS.md`
da. Hujjati **ochiq va to'liq** bo'lgan ikkitasi yozildi; qolganlari sotuvchi
javobiga yoki aniqlashtirishga bog'liq.

### Syrve — iiko'ning o'zi
Syrve — iiko'ning xalqaro brendi, **bir xil bulut API** (`api-eu.syrve.live`).
Shu sabab yangi adapter yozilmadi: `iiko.go` provider'ni **nomi bilan** oladi
(`c.name`) va Syrve uchun standart host almashadi. Xato xabarlari ham shu
nomdan quriladi — Syrve sotib olgan odamga "iiko: apiLogin qabul qilinmadi"
deb aytish, uning kassasi haqida gapirilayotganini yashiradi.

Ro'yxatda **alohida** turadi (iiko ichiga yashirilmadi): egasi o'z tizimini
"Syrve" deb biladi, va ro'yxatda faqat "iiko" ni ko'rsa qo'llab-quvvatlanmaydi
deb xulosa qiladi. Kalitlar ham **alohida saqlanadi** — iiko'ni sinab ko'rib,
keyin Syrve'ga o'tgan restoran birining apiLogin'i bilan ikkinchisiga
ulanmasligi kerak.

### Poster — `pos/poster.go`
`token` query'da, o'qishlar GET, yagona yozish — JSON POST. Metodlar:
`spots.getSpots` (Ping), `menu.getProducts`, `incomingOrders.createIncomingOrder`,
`incomingOrders.getIncomingOrder`.

Ikkita tuzoq ataylab test bilan qulflandi:
- **Pul tiyinda.** `price: "3500000"` — bu 35 000 so'm. So'mda yuborilgan
  buyurtma kassaga **yuz barobar arzon** tushadi va kassa indamay qabul qiladi.
  Aynan r_keeper darsi, boshqa valyutada.
- **`status: 0` — qabul qilingan emas.** Buyurtma "onlayn buyurtma" bo'lib
  tushadi va kassada odam uni qabul qilishi kerak. 200 javobini "oshxonada"
  deb o'qish — iiko'ning asinxron `create` i bergan yolg'onning aynan o'zi.
  Shu sabab `0` → `unknown`, `1` → `accepted`, `7` → `cancelled`.

**Poster'da bekor qilish yo'q** — `incomingOrders` da faqat create va read
(hujjatlar ro'yxati bo'yicha tekshirildi). `Cancel` → `ErrUnsupported`, va
panelda shu ogohlantirish yozilgan. O'ylab topilgan endpoint chaqirishdan
ko'ra, "kassada bekor qiling" deyish to'g'riroq.

Poster'da **qator izohi maydoni yo'q**, yetkazish narxi va buyurtma turi ham.
Hammasi buyurtma izohiga yig'iladi (`posterComment`) — mijozning "piyozsiz"i
tashlab yuborilsa, taom qaytib keladi.

### Testlar (loyihadagi birinchi `_test.go`)
`internal/pos/poster_test.go` — stub server bilan 7 ta test: tiyin
konvertatsiyasi (ikki yo'nalishda ham), `visible:0` va `hidden:1` bayroqlari,
noto'g'ri `spot_id` da Ping mavjud filiallarni sanashi, to'lanmagan buyurtma
"oldindan to'langan" deb belgilanmasligi, `status 0` "qabul qilindi" deb
o'qilmasligi, Poster'ning 200 ichidagi xato tanasi egaga yetib borishi, va
Syrve'ning o'z hostiga tushishi. Hammasi ✓.

### Panel
`/admin/settings` → "POS tizimi" da ikkita yangi yorliq. iiko va Syrve
formasi **bitta komponentdan** chiziladi (`IikoFields`) — nusxalangan forma
vaqt o'tib ajrab ketadi. Poster bloki ostida uning ikki cheklovi (kassada
qabul qilish, bekor qilish yo'qligi) yozilgan. Lug'at uz/ru/en.

**Ochiq**: Jowi va Paloma — hujjat so'rovi yuborilishi kerak; AliPOS va
Neon Alisa — sotuvchi bilan aloqa; Dodo/Yaros/Loook — aniqlashtirish
(`POS_INTEGRATIONS.md`, 7–9-bo'limlar).

---

## 2026-08-04 — Keel: brend, keel.uz sayti va control plane ✅

Mahsulot brendi **Keel** deb nomlandi (kemaning tubidagi asosiy nur —
ko'rinmaydi, hammasi shunga tayanadi). `keel.uz` bo'sh ekani tekshirilgan.

### Ikkita yangi xizmat, bitta repoda

```
control/     Go — tenantlar, statistika, billing, Caddy uchun ichki endpointlar
keel-site/   Next.js — keel.uz landing + Keel kabineti
```

**Nega tenant backendiga qo'shilmadi:** control plane hamma mijozning
ma'lumotini va Docker soketini boshqaradi. U restoran konteynerining ichida
tursa, **har bir mijoz boshqa mijozlarni o'chira oladigan kodni tashiydi**.
Bundan tashqari hozirgi backend **single-tenant** — faqat o'z bazasini biladi,
va aynan shu narsa ma'lumot sizib chiqishini imkonsiz qiladi. Alohida VPS esa
kerak emas: ikkalasi ~250 MB, restoran konteynerlari yonida turadi.

### control/
- `models` — `tenant` (slug, domenlar, holat, narx, watermark), `tenant_day`
  (kunlik agregat), `user` (Keel xodimi).
- **Slug hech qachon o'zgarmaydi**: u bazani (`t_<slug>`), konteynerni va
  standart domenni nomlaydi.
- **Statistika agregatdan o'qiladi, tenant bazalaridan emas.** Umumiy ekranni
  har mijozning bazasiga so'rov yuborib chizish — mijoz sotilgani sari
  sekinlashadigan yagona egri chiziq. Hisob-faktura ham **shu qatorlardan**
  quriladi, ya'ni ekrandagi raqam bilan hisobdagi raqam farq qila olmaydi.
- **Bekor qilingan buyurtma hisoblanmaydi** (`status != cancelled`) — o'zi
  bekor qilgan buyurtma uchun pul so'rash mijoz bilan birinchi janjal.
- `(tenantId, date)` unique: agregator qayta ishga tushsa oyni ikkilantirmaydi.
- **`/internal/tls-ask`** — Caddy'ning on-demand TLS darvozasi. Busiz istalgan
  odam domenini IP'ga yo'naltirib Let's Encrypt limitini kuydiradi.
- **Watermark bayrog'i control plane'da**, tenant sozlamalarida emas: uni
  restoran o'zi o'chira olsa, o'chiradi. `kioskSecret`/`soldOut` bilan bir xil
  naqsh, faqat narigi tomonida biznes modeli turadi.

### keel-site/
- Landing: hero, kimlar uchun (8 soha), imkoniyatlar, POS ro'yxati, narx,
  FAQ, CTA. **Light/dark** va **uz/ru/en** — til `lang` cookie'da va
  **serverda** o'qiladi, ya'ni sahifa allaqachon tarjima qilingan holda keladi.
- Dizayn: chuqur dengiz ohangi + iliq amber aksent. Belgi — **kema tanasi
  kesimi va tagidagi keel qanoti**, bitta shtrix, `currentColor`da.
  Shu sabab u bir vaqtning o'zida logotip ham, favicon ham, va mijoz
  saytidagi monoxrom **"Powered by Keel"** belgisi ham bo'la oladi.
- Kabinet: kirish, umumiy ko'rsatkichlar + 30 kunlik grafik, mijozlar ro'yxati
  (qidiruv/filtr/yaratish), mijoz kartochkasi (holat, narx, domenlar,
  watermark, kunlik jadval). Grafiklar div'larda — bitta qator uchun
  kutubxona ulanmaydi.

### Tekshirildi (haqiqiy Mongo, uchdan-uchgacha)
login → token; tenant yaratish (trial, `maracanda.keel.uz`, 1000 so'm);
noto'g'ri slug → 400; takroriy slug → **409**; `resolve` domen bo'yicha topdi;
begona domenga `tls-ask` → **404**, o'zinikiga → 200; tokensiz → **401**;
`stats` bo'sh kunlarni ham qatorga qo'shdi. Sinov bazasi o'chirildi.
`go vet` toza, `next build` toza, ikkala tema skrinshotda tekshirildi.

**Ochiq**: tenant konteynerini avtomatik ko'tarish (Docker API) va Caddyfile
generatsiyasi hali yozilmagan — hozircha tenant qo'lda ochiladi
(`SAAS.md`, S2 bosqichi).

---

## 2026-08-04 — S2: tenant avtomatik ishga tushadi ✅

Console'dagi "Yaratish" tugmasi endi **haqiqatan ishlaydigan sayt** beradi:
konteyner ko'tariladi, baza urug'lanadi, Caddy domenni o'rganadi.

### `control/internal/provision` — Docker
Engine API'ga **soket orqali to'g'ridan-to'g'ri** murojaat (SDK'siz — ishlatilgani
besh endpoint, SDK esa butun bog'liqlik daraxtini olib kelardi).
- **Port ochilmaydi**: konteynerlar `keel` tarmog'ida nomi bilan topiladi
  (`keel-<slug>:8080`). Har tenantga port ochish — har biri tashqaridan
  bazaga yo'l demak.
- **Har tenantga o'z `JWT_SECRET`i.** Bir restoranning tokeni ikkinchisida
  ruxsatsiz emas — **umuman o'qilmaydi**.
- `Ensure` yo'qini yaratadi, to'xtaganini ishga tushiradi; **hech qachon
  o'chirib qayta yaratmaydi** — uploads volumi va ortidagi tirik oshxona
  bizniki emas.
- CPU va xotira cheklangan (1 CPU, 512 MB): shovqinli tenant qolganlarni
  cho'ktirmasligi kerak.

### `control/internal/caddy` — chekka
Butun konfiguratsiya **har safar qaytadan** chiziladi va admin API'ga
`text/caddyfile` sifatida yuboriladi (Caddy o'zi tekshiradi; xato konfiguratsiya
butunlay rad etiladi va eskisi ishlayveradi).
- Bitta manbadan (tenant kolleksiyasi) qurilgan konfiguratsiya undan **ajrab
  keta olmaydi**; joyida tahrirlangani esa birinchi o'tkazib yuborilgan reload'da
  ajraydi, va alomati — mijoz domeni birovning do'konini ko'rsatishi.
- To'xtatilgan tenant **sahifa oladi**, sukut emas: javob bermaydigan domen
  uzilishga o'xshaydi va restoran noto'g'ri narsa haqida qo'ng'iroq qiladi.

### ⚠️ Sinov ochgan kamchilik: "tirik" ≠ "tayyor"
Birinchi versiya konteyner **ishga tushgani**ni tekshirardi. Mongo'ga ulana
olmayotgan server esa butun ulanish timeout'i davomida **tirik turadi** — va
`ready` deb yozilib, admin paroli o'chirilardi. Ya'ni buzuq tenant sog'lom
ko'rinardi va unga hech kim mijoz qo'ng'iroq qilgunicha qaramasdi.

Yagona halol signal — tenantning **o'z `/health`iga javobi** (`WaitHealthy`).
Bu iiko adapteridagi "yuborildi ≠ qabul qilindi" darsining aynan o'zi.
Xato bo'lsa xabarga **tenantning o'z log qatorlari** qo'shiladi — foydali
jumla deyarli hech qachon bizniki emas.

### Qolgan qismlar
- `apply()` ikkala yarmini bajaradi va natijani tenantga yozadi. **Hech biri
  so'rovni yiqitmaydi** — POS integratsiyasidagi qoida: yozilgan-u ishga
  tushmagan mijoz bitta tugma narida, yarim yo'lda 500 bergan yaratish esa
  operatorni taxmin qilishga majbur qiladi.
- **Parol faqat muvaffaqiyatdan keyin o'chiriladi.** Xato bo'lsa saqlanadi,
  aks holda qayta urinish owner hisobini yarata olmasdi.
- `frontend/src/lib/api.ts` — `TENANT_MODE=saas` da SSR backendni **Host
  bo'yicha** topadi (control plane'dan, 60 s kesh). Bayroqsiz bitta-restoran
  mahsuloti **avvalgidek** ishlaydi.
- Console'da `ProvisionCard`: holat, konteyner, xato matni va "Qayta urinish".

### Tekshirildi (haqiqiy Docker, haqiqiy konteynerlar)
Mijoz yaratildi → konteyner ko'tarildi → `/health` javob berdi → baza
urug'landi (48 taom, owner hisobi) → **console'da yozilgan parol bilan admin
panelga kirildi**. To'xtatish → konteyner `Exited`; qayta faollashtirish →
`Up`, **48 taom joyida**. Buzuq Mongo bilan → `failed` + tenantning log
qatorlari + **parol saqlanib qoldi**. Caddy renderi 4 ta test bilan qoplangan.
Sinov konteynerlari, tarmoq va image o'chirildi.

**Ochiq**: rolling update (`Recreate` yozilgan, navbat bilan chaqiruvchi yo'q)
va provisioning tugagach uploads papkasining egaligi (konteyner root yozadi).

---

## 2026-08-04 — kunning yakuni

Uchta katta blok: **POS** (Syrve + Poster), **Keel brendi va keel.uz**
(landing + console + control plane), **S2 provisioning** (tenant avtomatik
ko'tariladi). Har biri yuqorida alohida yozilgan.

Kod holati: `control`, `backend`, `keel-site`, `frontend` — hammasi quriladi,
`go vet` toza, testlar o'tadi (POS 7 ta, Caddy 4 ta).

---

## 2026-08-05 uchun reja — demo, obuna va o'chirish 📋

Ertangi maqsad: **restoran egasiga 14 kunlik demo berish va undan keyin pul
olish yoki o'chirish** — to'liq console orqali.

### 1. Yaratishda demo tanlanadi

Hozir har yangi tenant **majburan** `trial` bo'ladi va `TRIAL_DAYS` qo'shiladi.
Kerak: formada belgi — *"14 kunlik demo berilsinmi?"* va kun soni.

- Demo **yoqilgan**: `status: trial`, `trialEndsAt = bugun + N kun`.
- Demo **o'chirilgan**: `status: active`, `subscribedAt = bugun` — hisob
  birinchi kundan ketadi.

### 2. Obuna sanasi — hisobning langari

Bu ertangi kunning **eng katta o'zgarishi**, chunki hozirgi hisob-kitob
**kalendar oyi** bo'yicha ishlaydi (`monthTotals`, `monthStart`).

Talab: *"qaysi kuni obuna bo'lsa, o'sha kundan hisob ketaveradi"* — ya'ni
davr tenantning o'ziniki: 17-avgustda obuna bo'lgan restoranning davri
**17-avgust → 16-sentabr**, keyingisi 17-sentabrdan.

- `tenant.subscribedAt` qo'shiladi (demo tugab, pul to'langan kun).
- `billingPeriod(t, now)` → `[boshi, oxiri)`; kunlik agregat qatorlari
  (`tenant_day`) allaqachon `YYYY-MM-DD` bo'lgani uchun **istalgan oynani
  yig'ish oson** — yangi ma'lumot yig'ish shart emas.
- ⚠️ **31-kun tuzog'i**: 31-yanvarda obuna bo'lgan mijozning fevraldagi davri
  qayerda tugaydi? Qoida — **oyning oxirgi kuniga qisqartiriladi**. Buni
  hozir hal qilmasak, yiliga bir marta bitta mijozda kun "yo'qoladi" va
  sababini topish qiyin bo'ladi.
- Console'dagi "shu oy" ustuni **"joriy davr"** ga aylanadi. Umumiy ko'rinish
  (bizning tushumimiz) kalendar oyida qoladi — u boshqa savolga javob beradi.

### 3. Console'da buyurtmalar ko'rinadi

Qisman bor (ro'yxatda "shu oy", kartochkada kunlik jadval). Qo'shiladi:
- **Jami** buyurtmalar (butun umr) va **joriy davr** yonma-yon;
- kartochkada davr chegaralari va **"keyingi hisob qachon"**;
- ro'yxatga filtr: *demo muddati tugagan*, *to'lov kutilmoqda*.

### 4. O'chirish

`suspended` allaqachon konteynerni to'xtatadi va sahifa ko'rsatadi — ya'ni
texnik qism tayyor. Qo'shiladi:
- ro'yxatdan **bir bosishda to'xtatish/yoqish** (kartochkaga kirmasdan);
- **demo tugagan** tenantlar uchun alohida ro'yxat va ogohlantirish;
- `deleted` holati — ma'lumot saqlanadi, lekin ro'yxatni chalkashtirmaydi.
  **O'chirish ≠ bazani o'chirish**: baza N oy saqlanadi, qaytib kelgan mijoz
  menyusini qaytadan kiritmasin.

### 5. Demo muddati — qaror qabul qilindi ✔️

**Demo tugaganda tenant avtomatik o'chadi.** Demo — va'da emas, muddat.
**To'lamagan mijoz esa avtomatik o'chirilmaydi**: tushlik payti ishlayotgan
restoranni kechikkan hisob uchun o'chirish — mijozni yo'qotishning eng tez
yo'li. U qo'ng'iroqdan keyin, qo'lda o'chiriladi.

Aniq mexanika:

- **Supurgi** (sweep) mavjud soatlik tiker ichida yuritiladi (`aggregate.Every`
  yonida) — alohida cron kerak emas.
- Shart: `status == "trial"` **va** `trialEndsAt < hozir` → `suspended`,
  konteyner to'xtaydi, Caddy qayta yuklanadi.
- **Faqat `trial`ga tegadi.** Demo o'rtasida to'lagan mijoz `active` bo'ladi va
  `subscribedAt` qo'yiladi — supurgi uni ko'rmaydi ham.
- **Idempotent**: allaqachon `suspended` bo'lgani chetlab o'tiladi, ya'ni tiker
  har soat ishlaganda hech nima takrorlanmaydi.
- **Har avtomatik o'chirish log'ga yoziladi.** Sayt nega o'chgani haqidagi
  savol albatta beriladi, va "o'zi o'chib qoldi" degan javob yaramaydi.
- **Ogohlantirish hisoblanadi, saqlanmaydi**: `trialEndsAt` gacha 3 kundan kam
  qolgan tenantlar console'da belgilanadi. Saqlangan bayroq eskiradi, sana esa
  eskirmaydi.

⚠️ Bitta ehtiyot: supurgi **serverning soati** bo'yicha ishlaydi va `TZ`
`Asia/Tashkent` bo'lishi shart — aks holda demo mijoz kutganidan besh soat
oldin uziladi. Bu loyihada allaqachon ikki marta tishlagan tuzoq.

---

## 2026-08-05 — Obuna sanasi hisobning langari bo'ldi (reja, 2-band) ✅

Hisob endi **kalendar oyi bo'yicha emas, mijozning o'z davri bo'yicha**:
17-avgustda obuna bo'lgan restoranning davri 17-avgust → 17-sentabr, keyingisi
17-sentabrdan. Kalendar oyi hech kimniki emas — u oyning oxirida kelgan
mijozga deyarli bepul birinchi davr, boshida kelganiga esa to'liq davrni
bir xil pulga beradi.

### `control/internal/billing` — sof arifmetika
DB'ga ham, modelga ham tegmaydi: bu kod yilning bir oyida noto'g'ri, qolgan
o'n bir oyida to'g'ri bo'ladigan turdagi kod, shuning uchun aynan o'sha oylar
bilan test qilinadigan joyda turadi.
- **31-kun tuzog'i**: anchor kuni maqsad oyda bo'lmasa **oyning oxirgi kuniga
  qisqartiriladi**, va qisqartirish **doim asl anchor'dan** o'lchanadi:
  31-yanv → 28-fev → **31-mart**, 28-mart emas. Qisqartirishni oldinga tashish
  har qisqa oyda hisob kunini bir kunga orqaga suradi va buni bir yildan keyin
  hech kim topa olmaydi.
- `Cycle` `now` anchor'dan **oldin** bo'lsa **birinchi davrni** qaytaradi
  (operator kelasi oyning sanasini yozib yuborgan holat): obunadan oldingi
  vaqt uchun hisob yozish kechikkan hisobdan yomonroq.
- 7 test, jumladan **"har kun aynan bitta davrga tushadi"** invarianti: 400 kun
  × 6 xil anchor — bo'shliq ham, ustma-ustlik ham yo'q.

### `tenantPeriod` — davr kimniki
- **Demoga oylik sikl berilmaydi**: demo — muddat, sikl emas. Hali to'lashga
  rozi bo'lmagan mijoz uchun "keyingi hisob" ko'rsatish — operatorning
  mavjud bo'lmagan summani aytishi.
- `subscribedAt` yo'q eski `active` mijoz **ochilgan kundan** hisoblanadi va
  buni **aytadi** (`anchored: false`, panelda sariq ogohlantirish): javob
  baribir eng yaxshi javob, lekin hisobni tuzatayotgan odam qaysi sanaga
  suyanayotganini bilishi kerak.
- Buzuq ma'lumot (`trialEndsAt` boshlanishdan oldin) ham **yig'iladigan oyna**
  beradi: jadvaldan jimgina yo'qolgan qator noto'g'risidan yomonroq.

### Yozish tomoni — langar o'z-o'zidan siljimaydi
- Demo → `active` bo'lganda `subscribedAt` **o'sha kuni yoziladi**: pul qachon
  kelganini keyin hech kim eslab kelmaydi, va yozilmasa birinchi hisob mijoz
  ochilgan kundan, ya'ni demo haftalari bilan birga sanalardi.
- **Mavjud sana hech qachon avtomatik ustiga yozilmaydi** (`before.SubscribedAt
  == nil` sharti): `suspended` → `active` qaytish langarni siljitmaydi.
- Aniq sana operator tuzatishi sifatida qabul qilinadi, lekin **faqat qo'lda
  yozilganda** — telefon raqamini saqlashning yon ta'siri sifatida emas.
- `parseDay` **mahalliy yarim tun** beradi va chegara qo'yadi (2020 dan oldin
  yo'q, +62 kundan narida yo'q): yili `2206` deb terilgan sana `Cycle` uchun
  abadiy "birinchi davr" yasardi.

### 🐛 Uchdan-uchgacha sinov ochgan xato: Mongo sanani UTC qaytaradi
Barcha to'rt oyna **aynan bir kun erta** boshlandi. Sabab kodda emas edi —
drayver har `time.Time` ni **UTC location** bilan dekod qiladi, ya'ni mahalliy
yarim tun `19:00 (oldingi kun)` bo'lib qaytadi va undan kun boshini olish
oynani suradi. `TZ=Asia/Tashkent` to'g'ri bo'lsa ham. **Jimgina yiqiladi**:
oyna baribir haqiqiy oy, faqat noto'g'ri oy.
- Tuzatish: bazadan kelgan har sana ishlatilishidan oldin `local()`.
- **Brauzerda ham xuddi shu edi**: `subscribedAt` JSON'ga UTC bo'lib chiqadi va
  `slice(0,10)` date-input'ga oldingi kunni qo'yib, saqlanganda uni yozib
  yuborardi. Server endi **tayyor `period.anchor`** (`"YYYY-MM-DD"`) yuboradi.
- Regressiya testi sanani **`.UTC()` bilan** beradi — aynan drayver
  qaytaradigan ko'rinishda. CLAUDE.md ga tuzoq sifatida yozildi.

### Console
- Ro'yxatda "shu oy" o'rnida **"Joriy davr"** (ostida oyna sanalari) va yangi
  **"Jami (butun vaqt)"** ustuni. Umumiy ko'rinish kalendar oyida qoldi — u
  bizning tushumimiz haqidagi boshqa savol.
- Kartochkada 4 ta ko'rsatkich: davr buyurtmalari, hisob, jami, va **keyingi
  hisob sanasi** (demo uchun — demo tugash sanasi). Davrning yopilish sanasi
  ochiq intervalning oxiri, ya'ni to'g'ri saqlanadigan yagona sana.
- **Kartochkadagi yig'indi endi serverdan**: ilgari brauzer `days` ro'yxatini
  qo'shardi, ro'yxat esa 120 qator bilan chegaralangan — mijoz shu chegaradan
  oshgan kuni yig'indi jimgina, **kam tomonga** noto'g'ri bo'lardi.
- 🐛 Yo'l-yo'lakay: `days` so'rovi `date` bo'yicha **o'sish** tartibida
  saralanib limitlangani uchun uzoq yashagan mijozning **birinchi** 120 kunini
  qaytarardi — oylar oldin to'xtagan grafik "tinch mijoz"ga o'xshaydi.

### Tekshirildi (haqiqiy Mongo, uchdan-uchgacha)
Sinov bazasiga 4 mijoz (o'rta oy / 31-kun / demo / langarsiz) va har oyna
chegarasiga bittadan kun qatori qo'yildi. To'rtala oyna **aniq** chiqdi,
chegara kunlari to'g'ri tomonda; ochiq intervalning oxiri sanalmadi.
Yozish tomoni: demo→active langar qo'ydi, izoh saqlash langarni siljitmadi,
qayta faollashtirish ham; aniq tuzatish o'tdi va davr unga ergashdi; to'rtta
buzuq sana **400** bilan rad etildi va langarsiz mijoz langarsiz qoldi;
tokensiz so'rov **401**. Javobda `adminPassword`, `jwtSecret` yoki tenant
paroli **umuman yo'q** (alohida tekshirildi). Sinov bazasi o'chirildi.
`go vet` toza, 14 test o'tadi, `next build` va `tsc` toza.

**Ochiq**: console filtrlari (3-band), bir bosishda to'xtatish va supurgi
(4–5-band).

---

## 2026-08-05 — Yaratishda demo tanlanadi (reja, 1-band) ✅

Ilgari har yangi mijoz **majburan** `trial` bo'lardi. Endi formada belgi:
demo beriladimi va necha kun. Demo o'chirilsa mijoz `active` bo'ladi va
`subscribedAt` bugunga qo'yiladi — hisob birinchi kundan ketadi.

- **Farq yorliq emas**: demo muddat bilan keladi va muddat o'tganda o'chadi,
  obuna esa langar bilan keladi va taymer bilan o'chmaydi. Shuning uchun
  tanlov bir marta, yaratishda qilinadi — to'lashga rozi bo'lgan restoran ikki
  haftadan keyin hech kim bermoqchi bo'lmagan demo tugagani uchun uzilmasin.
- **`trial` — ko'rsatkich (`*bool`)**: "maydon yo'q" bilan "yo'q" bir xil emas.
  Demoni eslatmagan chaqiruv demo oladi — bu maydon paydo bo'lishidan oldin
  yaratilgan har bir mijoz olgan narsa. O'chirish **terilishi** kerak.
- **Kun soni chegaralangan (1–365)**: qo'lda teriladigan maydon, va 1400 kunlik
  demo ro'yxatda to'lovchi mijozdek turadi — kimdir oylarni sanamaguncha.
  `0` esa xato emas, sozlamadagi standart (14 kun).
- **Muddat bugunning boshidan** sanaladi: soat 23:50 da ochilgan demo 09:00 da
  ochilganidan bir kun qisqa bo'lmasligi kerak.
- Panelda standart holat — **demo yoqilgan**: o'ylab o'tirmasdan yaratilgan
  mijoz hisob emas, sinov oladi.

### ⚠️ Yana o'sha UTC yuzi
`trialEndsAt` — xom timestamp va JSON'ga UTC bo'lib chiqadi, ya'ni undan kun
kesib olish oldingi kunni beradi (sinov skripti aynan shunga ilindi).
Console uni **umuman o'qimaydi**: demo tugash sanasi `period.to` da, allaqachon
mahalliy kun qatori sifatida. `api.ts` da maydon ustiga shu ogohlantirish
yozildi.

### Tekshirildi (haqiqiy Mongo)
Demo eslatilmagan → 14 kunlik demo; `trialDays: 30` → 30 kun; `trial: false` →
`active` + bugungi langar + oylik sikl; `trialDays: 1400` → **400** va mijoz
umuman yaratilmadi; `trialDays: 0` → standart 14. Javobda admin paroli va
`jwtSecret` yo'q. 2-band to'plamlari ham qayta yuritildi — regressiya yo'q.
Sinov bazasi o'chirildi. `go vet`, 14 test, `tsc`, `next build` toza.

---

## 2026-08-05 — Console: ogohlantirish va filtrlar (reja, 3-band) ✅

Bandning ustunlari (**joriy davr** va **jami**) 2-bandda tushgan edi; bugungi
qism — **kimga qo'ng'iroq qilish kerak**.

### Ogohlantirish hisoblanadi, saqlanmaydi
`tenantAttention` har so'rovda sanadan hisoblaydi: **demo tugayapti** (3 kun
yoki kamroq), **demo tugagan**, **to'lov kutilmoqda** (`suspended`). Saqlangan
bayroq soat undan o'tgan zahoti eskiradi, sana esa eskirmaydi.
- **Kunlar butun kalendar kuni bo'yicha**, 24 soatlik bloklar bo'yicha emas:
  bugun kechqurun tugaydigan demo bilan ertaga ertalab tugaydigani — "0 kun"
  va "1 kun", ikkalasi ham "0" emas.
- Kun soni **serverda** sanaladi: brauzer UTC bo'lib kelgan timestamp ustida
  sana arifmetikasi qilmasligi kerak (bugun ikki marta tishlagan tuzoq).
- **Muddati yozilmagan demo tugagan deb belgilanmaydi** — yo'q maydon asosida
  ayblov qo'yilmaydi.
- Noma'lum filtr **400** qaytaradi, "filtrsiz"ga aylanmaydi: hammani ko'rsatgan
  tugma "hech kimga qo'ng'iroq kerak emas" deb o'qiladi.

### 🐛 Yo'l-yo'lakay topilgan tuzoq: ikkita `$or` bir-birini yeydi
Qidiruv `filter["$or"]` yozadi, yangi filtr ham `$or` xohlaydi — Go'da
ikkinchi tayinlash birinchisini **jimgina** o'chiradi. Alomati: qidiruv
ishlayotgandek ko'rinib, so'ralmagan mijozlarni qaytaradi. Shart endi
`$and` ro'yxati sifatida yig'iladi. Sinovda maxsus tekshirildi: "osh" hamma
nomda bor, ya'ni ro'yxatni **filtr** toraytiradi — ikkala shart ham tirik.

### Console
- Ro'yxatda **uchta filtr tugmasi** (ochiladigan ro'yxat emas — call-markaz
  jurnalidagi bir xil sabab: bular ro'yxatning butun ma'nosi, select ichiga
  yashirilgan savolni hech kim so'ramaydi). Qayta bosilsa o'chadi.
- Har qatorda holat yonida **nishoncha** ("Demoga 2 kun qoldi", "Demo 5 kun
  oldin tugagan", "To'lov kutilmoqda") — filtrlamasdan, ro'yxatni ko'z bilan
  yugurtirish yetarli bo'lsin. Izlab topiladigan ogohlantirish kech keladi.
- Umumiy ekranda **"E'tibor talab qiladi"** paneli, har biri filtrlangan
  ro'yxatga havola. **Faqat kimdir bo'lsa chiziladi**: doimiy nol qatori bir
  haftada bezakka aylanadi va "3" yozilgan kuni ko'rinmay qoladi.
- Filtr **URL'da** (`?attention=`) — havola sifatida yuborsa bo'ladi.
  `useSearchParams` uchun `Suspense` chegarasi qo'yildi.

### Tekshirildi (haqiqiy Mongo)
Chegaraning ikki tomoniga qo'yilgan 8 mijoz: 4 kun (jim), 3 kun (birinchi
ogohlantirilgan kun), 1 kun, bugun tugagan, 5 kun oldin tugagan, to'lovchi,
to'xtatilgan va muddatsiz demo. Nishonchalar aynan to'g'ri; har tugma faqat
o'z mijozlarini qaytardi; filtr + qidiruv, filtr + holat va ziddiyatli juftlik
ham to'g'ri; noma'lum filtr **400**; umumiy ekran tallisi ro'yxat bilan mos.
1- va 2-band to'plamlari qayta yuritildi — regressiya yo'q. Sinov bazasi
o'chirildi. `go vet` toza, **18 test** o'tadi, `tsc` va `next build` toza.

**Eslatma**: "to'lov kutilmoqda" — `suspended` holati (modelda u aynan "pul
to'lanmagani uchun o'chirilgan" degani). Haqiqiy hisob-faktura daftari yo'q,
shuning uchun "to'lagan/to'lamagan" bundan nozikroq ayta olinmaydi; kerak
bo'lsa bu alohida ish.

---

## 2026-08-05 — O'chirish va bir bosishli to'xtatish (reja, 4-band) ✅

### `deleted` — o'chirish bazani o'chirish emas
Yangi holat: mijoz ro'yxatdan chiqadi, sayti to'xtaydi, **lekin bazasi
saqlanib qoladi**. Sabab ikkita: martda qaytib kelgan restoran menyusini
qaytadan kiritmasin, va bu tugma butun platformada **orqaga qaytarib
bo'lmaydigan yagona amal** bo'lib qolmasin. Diskni bo'shatish — alohida,
keyingi, ataylab qilinadigan qaror.
- **`Offline()` — bitta predikat** (`suspended` yoki `deleted`), uchta joyda
  (konteyner, Caddy, parolni tozalash) alohida taqqoslash o'rniga: to'rtinchi
  holat paydo bo'lganda o'tkazib yuborilgan taqqoslash to'xtatilgan mijozning
  konteynerini ishlab turgan holda qoldirardi.
- Chekkada `suspended` bilan bir xil: **sahifa ko'rsatiladi, sukut emas** —
  javob bermaydigan domen uzilishga o'xshaydi.
- Ro'yxatda yashiriladi, lekin **`?status=deleted` bilan topiladi**: qayta
  topib bo'lmaydigan yozuvga hech kim ishonmaydi.
- **Slug band bo'lib qoladi** (baza nomi `t_<slug>`). Endi xato xabari buni
  aytadi — o'chirilgan mijoz ro'yxatda ko'rinmagani uchun operator bo'sh
  slugni band deb eshitardi.
- Umumiy ekranda **alohida sanaladi va jamiga kirmaydi**: "40 ta mijozimiz
  bor" qatori har ketgan mijozdan keyin o'z-o'zidan o'sib ketmasin.

### Oxirgi hisob o'chirish paytida olinadi
Agregator `deleted` mijozlarni **o'tkazib yuboradi** — ketgan mijozning
raqamlari o'zgarmaydi, va har soatda uning bazasiga ulanish "hech qachon
ketmaydigan xarajat"ga aylanadi. Lekin shunchaki o'tkazib yuborish oxirgi
kunning bir qismini yo'qotardi, va hisob aynan o'sha qatorlardan quriladi.
Shuning uchun holat o'zgarishidan **oldin** shu mijoz uchun bir marta
`aggregate.One` yuritiladi. Yig'ib bo'lmasa ham o'chirish to'xtamaydi:
ulanib bo'lmaydigan mijozni yopa olmaslik operatorni tupikka qo'yadi.

### Bir bosishda to'xtatish/yoqish
Ro'yxatning oxirgi ustunida bitta tugma va u **doim nima qilishini yozadi**
(hozirgi holatini emas — shunday yorliqli tugma xato bosiladi). Har bosishda
tasdiq so'raladi va savolda **sayt nomi** hamda mijozlari nima ko'rishi
yoziladi: qatorlar bir-biridan bir qator narida, tugma esa tushlik paytida
ishlayotgan restoranni uzadi.
**"Mijozni o'chirish" ro'yxatda yo'q** — u kartochkada, Saqlash tugmasidan
uzoqda. Tasdiq matni bazaning **o'chirilmasligini** ochiq aytadi: nima
yo'q qilishini bilmagan odam yo tugmadan umuman qochadi, yo bir marta bosib
ko'radi.

### Tekshirildi (haqiqiy Mongo)
`t_paid` bazasiga 3 taom va 3 buyurtma (biri bekor qilingan) qo'yildi.
O'chirishdan oldin kunlik qator **yo'q** edi; o'chirishdan keyin qator paydo
bo'ldi va unda **2 buyurtma** — bekor qilingani hisoblanmadi. Mijoz ro'yxatdan
yo'qoldi, `?status=deleted` bilan topildi, ogohlantirish bermadi, alohida
sanaldi; **menyu 3 ta bo'lib qoldi**. Slug bilan yangi mijoz ochishga urinish
**409**. Qaytarilgandan keyin holat `active`, ro'yxatda, **menyu joyida**.
Muddati o'tgan demo o'chirilgach ogohlantirish ro'yxatidan chiqdi. Noma'lum
holat **400**. 1–3-band to'plamlari qayta yuritildi — regressiya yo'q. Sinov
bazalari o'chirildi. `go vet` toza, 18 test o'tadi, `tsc` va `next build` toza.

---

## 2026-08-05 — Supurgi: demo tugaganda avtomatik o'chadi (reja, 5-band) ✅

Kunning oxirgi bandi. Demo — **muddat**, va'da emas: tugagandan keyin ishlab
turgan demo jimgina bepul mahsulotga aylanadi, qaror qabul qilishi kerak
bo'lgan mijoz esa hech qachon qaror qilmaydi — bu unga ham yomon, chunki
hech kim qo'ng'iroq qilmaydi.

- **Faqat demolar tegiladi.** To'lamagan mijoz **hech qachon** taymer bilan
  o'chirilmaydi: tushlik payti buyurtma qabul qilayotgan restoranni kechikkan
  hisob uchun uzish — uni yo'qotishning eng tez yo'li. Uni operator
  qo'ng'iroqdan keyin qo'lda to'xtatadi, va console'dagi "To'lov kutilmoqda"
  filtri uni operator ko'z oldiga qo'yadi.
- **Shart konsoldagi tugmadan olinadi** (`attentionFilter(trial_expired)`) —
  nusxa emas, **aynan o'sha**. Ikkisi ajrab ketsa ro'yxat hech kim tegmaydigan
  mijozni "tugagan" deb ko'rsatardi yoki supurgi ro'yxat ogohlantirmagan
  mijozni o'chirardi.
- **Shart yozuvning o'zida** (`UpdateOne` filtrida `status: "trial"`),
  oldindan tekshirilmaydi: ro'yxatni o'qish bilan qatorni yozish orasida
  operator pulni olgan bo'lishi mumkin, va to'lagandan bir soniya keyin
  o'chirilgan mijoz — mumkin bo'lgan eng yomon birinchi kun. Shu guard bir
  vaqtning o'zida **idempotentlikni** ham beradi: allaqachon o'zgargan qator
  `ModifiedCount = 0` qaytaradi va hech nima takrorlanmaydi.
- **`autoSuspendedAt` saqlanadi**, faqat log emas. Sayt nega o'chgani haqidagi
  savol bir necha hafta o'tib beriladi — server logi allaqachon aylanib
  ketgan bo'ladi, va "o'zi o'chib qoldi" javob emas. Kartochkada shu sana
  bilan yoziladi. Holat `suspended` dan chiqqanda **tozalanadi**, aks holda u
  keyinroq odam ataylab qilgan to'xtatishni tasvirlab qolardi.
- **Bitta tiker** (`cmd/server/maintain`): avval agregat, keyin supurgi —
  shu tartibda, bitta soat bo'yicha. Ikki tikerga bo'lish o'chirilayotgan
  demoning oxirgi kuni sanalmasdan qolishiga olib kelardi. Boot'da ham bir
  marta ishlaydi: dam olish kunlari o'chib turgan server ertalab quvib
  yetadi. ⚠️ Hammasi **serverning soati** bo'yicha — `TZ=Asia/Tashkent`
  shart, va u boot'da log qilinadi.
- `aggregate.Every` o'chirildi (jadval endi `cmd/server` da) — o'lik kod
  qolmasin.

### Tekshirildi (haqiqiy Mongo, serverni qayta ishga tushirib)
Chegaraning ikki tomonidagi 8 mijoz bilan: bir supurgidan keyin **faqat
`today` (bugun tugagan) va `over` (5 kun oldin tugagan)** `suspended` bo'ldi
va sana bilan belgilandi; `paid` **active bo'lib qoldi**, `soon`/`edge`/`far`
demo bo'lib qoldi, muddatsiz demo tegilmadi, qo'lda to'xtatilgani
**belgilanmadi**. Server qayta ishga tushirildi — ikkinchi supurgi hech
nimani o'zgartirmadi va sanani qayta yozmadi. To'lov: `today` → `active`,
belgi tozalandi va **bugungi kun langari qo'yildi**; uchinchi supurgi unga
tegmadi. Log qatori joyida. 1–4-band to'plamlari qayta yuritildi — regressiya
yo'q. Sinov bazalari o'chirildi. `go vet` toza, 18 test, `tsc` va
`next build` toza.

---

## 2026-08-05 — reja bo'yicha ishning yakuni

Ertalabki rejadagi **beshala band ham bajarildi**: demo tanlovi, obuna sanasi
langari, console ko'rsatkichlari va filtrlari, o'chirish, supurgi. Ya'ni
restoran egasiga demo berish → hisobni yuritish → pul olish yoki o'chirish
oqimi to'liq console orqali ishlaydi.

Kun davomida **uchta haqiqiy xato** uchdan-uchgacha sinovda topildi (unit
testlar ko'rmagan): Mongo sanani UTC qaytarishi (butun hisob davri bir kun
erta), `days` ro'yxatining teskari tomondan cheklanishi (kartochkadagi
yig'indi kam tomonga noto'g'ri), va ikkita `$or` ning bir-birini yeyishi
(qidiruv jimgina o'chardi).

**Ochiq**: rolling update, uploads papkasining egaligi (S2 dan qolgan),
hisob-faktura daftari (agar "to'lov kutilmoqda" `suspended` dan nozikroq
bo'lishi kerak bo'lsa).

---

## 2026-08-05 — Eski VPS tozalandi 🧹

`173.249.8.13` dan **traderbot restoran proyekti** (jomaxuz) va **traderbot
telegram boti** (tradebot) butunlay olib tashlandi. O'sha serverda egasining
ikkinchi, ishlab turgan proyekti — `filmorauz.net` — bor, shuning uchun har
qadam undan oldin va keyin o'lchandi: `filmorauz.net` 200, `api.filmorauz.net`
404 (ilovaning `/` uchun normal javobi) — o'zgarish yo'q.

O'chirilganlar: 3 konteyner, 2 nomlangan + 1 anonim volume, 2 image, tarmoq,
`/opt/jomaxuz`, `/opt/tradebot`, nginx bloki, `traderbot.uz` sertifikati,
`/usr/local/bin/jomaxuz-deploy`, `authorized_keys` dagi cheklangan kalit va
`tradebot.service`.

Zaxira ikki joyda (92 MB): serverda `/root/backup-2026-08-05/` va egasining
kompyuterida `~/vps-backup-2026-08-05/` — mongodump, uploads, kod, systemd
unit, nginx bloki.

### ⚠️ Uch marta ataylanmagan prod deploy
Serverga SSH orqali ulanishga urinish **har safar to'liq deploy'ni ishga
tushirardi** — image'lar qaytadan quriladi, konteynerlar recreate qilinadi.
Buyruqlar umuman bajarilmasdi; `sftp` ham *"Ensure the remote shell produces
no output"* bilan yiqilardi.

Diagnoz ikki marta noto'g'ri bo'ldi. Avval `sshd_config` dagi global
`ForceCommand` deb o'ylandi — u yerda faqat Ubuntu'ning izohga olingan
namunasi turgan ekan. Keyin `/root/.ssh/rc` deb — u ham yo'q edi.

Haqiqiy sabab: **ulanish parol bilan emas, kalit bilan o'tardi.** Egasining
noutbukidagi `ssh-agent` da uchta kalit yuklangan va birinchi taklif
qilinadigani aynan `command="/usr/local/bin/jomaxuz-deploy"` bilan cheklangan
CI kaliti edi. Parol umuman ishtirok etmagan.

`-o PubkeyAuthentication=no` bilan darhol oddiy shell olindi. Ya'ni server
sozlamasi **to'g'ri** edi — muammo mijoz tomonidagi kalit tartibida.

Dars: "SSH parol bilan ulandim" degan taxmin tekshirilmaydigan taxmin. SSH
avval kalitlarni sinaydi, va agent'dagi tartib ko'rinmaydi.

### Yo'l-yo'lakay
Konsolda ekranda 70 kun oldingi `soft lockup` xabarlari turgan ekan (tty1 hech
kim kirmagani uchun eski chiqishni saqlab qolgan). Ular jonli deb o'ylanib
keraksiz shov-shuv ko'tarildi — uptime 113 kun, xabarlarning yadro vaqti esa
42-kunga to'g'ri kelardi, va yuk 0.07 edi.

---

## 2026-08-05 — Keel jonli: yangi VPS, `keel.uz` ishga tushdi 🚀

Yangi server `169.58.131.165` (Ubuntu 24.04, 4 CPU, 8 GB, 94 GB bo'sh, toza).
Alohida server tanlandi, chunki filmorauz nginx'ning 80/443 ini egallagan va
Caddy'ning on-demand TLS'i aynan o'sha portlarni talab qiladi. Bir mashinaga
sig'dirish mumkin edi — faqat filmorauz'ning TLS'ini, sertifikat
yangilanishini va mijoz IP ko'rinishini o'zgartirish evaziga.

### Yozilgan yetishmayotgan qismlar
`control/` va `keel-site/` uchun Dockerfile umuman yo'q edi, Keel compose'i
ham (SAAS.md da "paydo bo'ladi" deb turardi):

- `docker-compose.saas.yml` — caddy, control, site, frontend, mongo +
  quriladigan-yu ishga tushirilmaydigan `tenant` image'i
- `control/Dockerfile`, `keel-site/Dockerfile`, `caddy/Caddyfile` (bootstrap)
- `.env.saas.example`

**Tarmoq va konteyner nomlari qat'iy** (`keel`, `keel-control`,
`keel-frontend`, `keel-site`, `keel-caddy`): generatsiya qilingan Caddy
konfiguratsiyasi ularga nom bo'yicha murojaat qiladi, compose o'zgartirib
qo'ygan nom chekkani bo'shliqqa qaratadi. Mongo porti umuman chiqarilmaydi —
bu hostdagi yagona konteynerda bizning ham, har mijozning ham bazasi turadi.

### 🐛 Caddy qayta ishga tushganda hamma mijoz yo'qolardi
Caddy restart'da bootstrap faylini qaytadan o'qiydi, ya'ni control plane
yuborgan haqiqiy konfiguratsiya yo'qoladi. Eski kodda chekka **faqat tenant
tahrirlanganda** sinxronlanardi — server qayta yuklangach har bir mijoz
"sozlanmoqda" sahifasida qolardi, kimdir tenantni tahrirlagunicha, va server
oylab shunday turishi mumkin. Endi `SyncEdge` boot'da ham, soatlik tikerda ham
ishlaydi (bootda 10 marta 3 soniyada qayta urinadi — Caddy hali ko'tarilmagan
bo'lishi mumkin).

### Sertifikat va DNS
Domen ahost'dan Cloudflare'ga ko'chirildi — **faqat DNS sifatida, kulrang
bulut**. CDN proksisi bu arxitekturaga to'g'ri kelmaydi: bepul tarifda
wildcard proksi qilinmaydi (`*.keel.uz` — bu har bir mijoz), proksi ortida
Caddy TLS qo'l berishini ko'rmaydi va `tls-ask` chaqirilmaydi, va `*` kulrang
bo'lgani uchun origin IP baribir e'lon qilinadi — ya'ni faqat `keel.uz` ni
proksi qilish hech nimani yashirmaydi.

⚠️ **Rate limit.** Caddy domen sozlanmasdan **oldin** ishga tushirilgani uchun
DNS `SERVFAIL` paytida qayta-qayta ACME'ga urinib, Let's Encrypt limitiga
urildi (soatiga 5 muvaffaqiyatsiz avtorizatsiya). Limit tugagach ham Caddy
backoff'da turadi — Caddy va control qayta ishga tushirilib majburlandi.
To'g'ri tartib: **avval DNS, keyin Caddy**.

`www` uchun alohida yozuv shart emas — `*` uni ham qamraydi.

### Tekshirildi
`keel.uz`, `www.keel.uz`, `/console` — 200, Let's Encrypt. Konsolda mijoz
yaratildi (`testrest`): konteyner ko'tarildi, backend sog'lom, 7 kategoriya /
48 taom urug'landi, `testrest.keel.uz` uchun sertifikat **avtomatik** olindi.
Begona domen (`begona-test.uz`) rad etildi — `tls-ask` darvozasi ishlayapti.

---

## 2026-08-05 — CI/CD: uch marta yiqildi, uchtasi ham boshqa sabab 🐛

`main` ga push → Actions `deploy-keel@169.58.131.165` ga ulanadi →
`/usr/local/bin/keel-deploy`. Kalit **root'da emas, `deploy-keel` da**, forced
command bilan. Serverda doimiy GitHub kaliti yo'q: Actions vaqtinchalik
`GITHUB_TOKEN` ini SSH buyrug'i sifatida uzatadi, token askpass yordamchisiga
beriladi (URL'ga emas — URL git ishlaganda `ps` da har hisobga ko'rinadi).

**1-yiqilish: prefiks.** Tekshiruv `gh[a-z]_` ni talab qilardi. Diagnostika
yo'q edi — logda bitta "rad etildi" qatori.

**2-yiqilish: uzunlik.** Diagnostika qo'shilgach ko'rindi: token `ghs_` bilan
boshlanadi, lekin **377 belgi** — 255 chegarasidan uzun.

Ikkalasining ildizi bitta: tekshiruvni **himoya qiladigan narsaga** emas,
**tanish ko'rinishga** bog'lash. Askpass qo'shilgandan keyin qat'iy format
tekshiruvi allaqachon ortiqcha edi. Endi faqat o'zgarmaydigan ikki shart:
bo'sh emas va bo'sh joy tutmaydi (ya'ni shell parchasi emas).

**3-yiqilish: nom to'qnashuvi.** Qo'lda deploy va CI deploy'i bir necha soniya
farq bilan ustma-ust tushdi. Compose konteynerni almashtirishdan oldin
vaqtincha qayta nomlaydi, ikkinchi yugurish o'sha nomni band topadi. Endi
`flock` — bir vaqtda bitta deploy, manbasidan qat'i nazar — va qolib ketgan
`<id>_keel-*` konteynerlar tozalanadi.

### 🐛 Eng xavflisi: deploy kodni olib kelardi-yu ishga tushirmasdi
`build` va `up -d` alohida chaqirilardi. Compose yangi image'ni qurdi va eski
konteynerni qoldirdi:

```
tag :latest    :  a5e7dacd...
konteyner      :  0feb5d1a...
compose yorlig':  0cf41e66...   ← uchta har xil ID
```

CI "muvaffaqiyatli" tugardi — konteynerlar sog'lom, tekshiruvlar yashil,
serverda to'g'ri commit — lekin **yangi kod ishlamasdi**. Yagona alomat
chekka konfiguratsiyasida yo'nalishning yo'qligi edi.

Bu turdagi xato eng yomoni: hamma ko'rsatkich yashil, deploy esa yolg'on.
Topilmasa, keyingi har bir tuzatish "deploy qilindi" deb hisoblanib, aslida
ishlamay turardi. Endi quriladigan uchta xizmat `--force-recreate --no-deps`
bilan **aniq** almashtiriladi (mongo va caddy chetda: mongo hamma bazani
tutadi, caddy restart chekkani bootstrap'ga qaytaradi).

⚠️ `/usr/local/bin/keel-deploy` **git bilan yangilanmaydi** — u deploy
qilinadigan daraxtdan tashqarida, ataylab: forced command o'zi tortadigan
kodga bog'liq bo'lsa, cheklovning ma'nosi qolmaydi. Repodagi nusxa —
`deploy/keel-deploy`, o'zgartirilsa qo'lda ko'chiriladi (DEPLOY.md da).

---

## 2026-08-05 — Konsolga kirish 500 qaytarardi 🐛

Brauzerda: `Unexpected token 'I', "Internal S"... is not valid JSON`.

`keel.uz` butunligicha Next.js jarayoniga berilgan edi, u esa `/api/*` ni
control plane'ga uzatadi deb hisoblangandi. Uzatmaydi: **Next `rewrites()`
manzillarini build vaqtida marshrutlar manifestiga muhrlaydi**, ya'ni
konteynerga berilgan `CONTROL_ORIGIN` umuman e'tiborga olinmaydi. Har login
site konteynerining o'z ichidagi `localhost:9000` ga borardi.

Endi `/api/*` ni **Caddy** yo'naltiradi — xuddi tenant saytlarida bo'lgani
kabi. Brauzer bir xil origin'da qoladi, frontend esa control plane qayerdaligini
bilishi shart emas, va uni o'zgartirish uchun qayta build kerak emas.

Test qo'shildi: asosiy blokda `/api/*` control'ga ketishi **va** catch-all
undan keyin turishi (oldinda tursa yutib yuboradi). Mavjud testlar faqat
tenant bloklarini qoplagani uchun buni ko'rmagan edi.

---

## 2026-08-05 — Har restoran o'z 2GIS kaliti; o'z domeni uchun qo'llanma ✅

### Xarita kaliti tenantga ko'chdi
Endi har restoran o'zinikini admin panelda kiritadi — platforma hammaning
kvotasini bitta hisobda ko'tarmaydi. Kalit build'ga muhrlanmaydi (bitta build
hamma mijozga xizmat qiladi), ish vaqtida restoran profilidan o'qiladi
(`lib/mapKey.ts`, sahifaga bitta so'rov, uchala xarita komponenti baham
ko'radi).

⚠️ **Kalit ochiq va boshqacha bo'la olmaydi.** MapGL brauzer kutubxonasi:
kalit sahifada `load({ key })` ga uzatiladi va DevTools'da ko'rinadi — uni
qayerda saqlasak ham. Hech bir xarita SDK'si boshqacha ishlamaydi. Himoyani
2GIS kabinetidagi **domen cheklovi** beradi.

Bu **to'lov kalitlarining aynan teskarisi**: ular `restaurant` hujjatidan
ataylab chiqarilgan (u har tashrifchiga qaytariladi), bu esa ataylab ichida.
Modelda va sozlamalar sahifasida shu yozilgan — aks holda kimdir buni
"xavfsizlik tuzatishi" deb yashiradi va xarita ishlamay qoladi.

`null` (yuklanmoqda) va `""` (sozlanmagan) ajratilgan: aks holda "sozlanmagan"
xabari har sahifada chaqnab, egani allaqachon to'g'ri turgan sozlamani
qidirishga yuborardi.

### O'z domeni uchun qo'llanma + DNS tekshiruvi
Sozlamalarda uch qadam va tugma. Aynan DNS qadami **jimgina** buziladi: yozuv
registratorda saqlanadi, ko'rinib hech nima o'zgarmaydi, va ega "hali
tarqalmagan" bilan "noto'g'ri yozganman" ni ajrata olmaydi.

`GET /admin/domain-check?domain=` domenni ham, saytning **hozirgi** manzilini
ham yechadi va solishtiradi. Kutilayotgan IP hech qayerda sozlanmagan —
tenant allaqachon o'sha yerda ishlayapti, ya'ni haqiqatdan ajrab qoladigan
sozlama yo'q.

**Oxirgi qadam qo'lda, va sabab sahifada yozilgan**: domen ro'yxatga tushishi
bilan biz uning nomiga HTTPS sertifikati so'raymiz. Restoran istagan domenni
o'zi yoza olsa, `google.com` ni yozib bizni birovning domeni uchun sertifikat
so'rashga majbur qiladi. Bu dangasalik emas, chegara.

### 🐛 Domen qo'shilsa ham konteyner eski manzilni yozardi
`Ensure()` ishlab turgan konteynerga tegmaydi, `PUBLIC_BASE_URL` va
`CORS_ORIGINS` esa **yaratilish paytida** birlamchi domendan olinadi. Ya'ni
o'z domenini qo'shgan mijozning sayti yangi manzilda ochilardi-yu, chop
etadigan har bir mutlaq havola — to'lov callback'i, buyurtma kuzatuvi, QR
kod — eskisini yozardi.

Endi domen o'zgarganda `Recreate` chaqiriladi (u yozilgan-u chaqiruvchisi yo'q
edi). Holat o'zgarishida qayta yaratilmaydi — u faqat to'xtatadi yoki ishga
tushiradi. Uploads volumi va baza tegilmaydi.

---

## 2026-08-05 — kunning haqiqiy yakuni

Ertalab: rejadagi beshala band (demo, obuna langari, console filtrlari,
o'chirish, supurgi). Kechqurun: **Keel jonli ishga tushdi.**

`https://keel.uz` — sayt va konsol, Let's Encrypt sertifikati bilan. Birinchi
mijoz konsoldan yaratildi va `<slug>.keel.uz` da avtomatik sertifikat bilan
ochildi. CI/CD ishlayapti. Eski server tozalandi, filmorauz tegilmadi.

### Kun davomida topilgan xatolar

Unit testlar ko'rmagan, faqat uchdan-uchgacha sinovda chiqqanlari:

1. Mongo sanani UTC qaytaradi → butun hisob davri bir kun erta
2. `days` ro'yxati teskari tomondan cheklangan → kartochkadagi yig'indi kam
3. Ikkita `$or` bir-birini yeydi → qidiruv jimgina o'chadi
4. Caddy restart'da bootstrap'ga qaytadi → hamma mijoz "sozlanmoqda" da
5. Next `rewrites()` build'ga muhrlanadi → konsolga kirish 500
6. Compose image quradi-yu konteynerni almashtirmaydi → **yashil, lekin yolg'on deploy**
7. `Ensure` konteynerga tegmaydi → o'z domeni qo'shilsa havolalar eski

Oltinchisi eng xavflisi: hamma ko'rsatkich yashil bo'lib turadi.

### Ochiq
- `SMS_PROVIDER=demo` — haqiqiy mijoz kirmasdan oldin Eskiz kalitlari
- Rolling update: tenant konteynerlari eski image'da qoladi, yangi backend
  faqat qayta provisioning'dan keyin yetadi
- uploads papkasining egaligi (S2 dan qolgan)
- Mijoz domenini to'liq avtomatik ulash (DNS tekshiruvi egalik isboti sifatida)
- Hisob-faktura daftari, agar "to'lov kutilmoqda" `suspended` dan nozikroq
  bo'lishi kerak bo'lsa

---

## 2026-08-06 — SMS provayderlari, rolling update, domen, uploads, hisoblar ✅

Kecha ochiq qolgan beshala band yopildi.

### 1. SMS: to'rt provayder, har restoran o'zi tanlaydi
Eskiz, Play Mobile, **getsms.uz** va **OneSignal**. Beshinchisi — `demo`.

Asosiy qaror: **provayder deploy sozlamasi emas, restoran sozlamasi**. Har
restoran o'z shartnomasini tuzadi, o'z jo'natuvchi nomini moderatsiyadan
o'tkazadi va kalitlarni panelda kiritadi. Platformaning bitta umumiy hisobi
hammaning kodini bitta shartnomaga bog'lardi — va **bitta restoranning
moderatsiya muammosi qolganlarning loginini o'chirardi**.

Kalitlar `sms_settings` da (alohida kolleksiya, `payment_settings` bilan bir
sabab: `restaurant` hujjati har tashrifchiga to'liq boradi). Bo'sh parol =
"saqlangani qolsin".

⚠️ **Ikkala yangi provayder ham rad javobini 200 ichida yuboradi.** getsms.uz
noto'g'ri parolni, tasdiqlanmagan nickname'ni va shartnomadan tashqaridagi
raqamni muvaffaqiyatga o'xshatib qaytaradi; OneSignal 200 + `errors` beradi.
Tekshirilmasa sayt yuborilmagan kod haqida "yuborildi" deydi va mijoz kutib
qoladi. Testda muhrlandi. Raqam formati ham har xil: hamma shlyuz
`998XXXXXXXXX`, OneSignal esa **E.164** (`+998...`).

**"Sinov SMS" — sahifaning asosiy tugmasi.** Kalitlar to'g'ri ko'ringanda ham
jo'natuvchi nomi tasdiqlanganmi va hisobda pul bormi — ko'rinmaydi, va ikkalasi
ham birinchi mijoz kirmoqchi bo'lganda bilinadi. Sinov matni ataylab haqiqiy
kod xabariga o'xshatilgan: shlyuz **shablonni** moderatsiya qiladi.

Sender keshlanadi (kalit — `updatedAt`): Eskiz ~30 kunlik token tutadi va har
so'rovda qayta qurish har SMS uchun qaytadan login qilardi. Kesh kaliti
`updatedAt` bo'lgani uchun saqlash **restartsiz** yangi shlyuzga o'tadi.

### 2. Rolling update — deploy endi mijozgacha yetadi
`GET/POST /rollout` + konsolning bosh sahifasida panel.

Eng muhim qism: **"yangilangan"ni image ID bo'yicha hal qilish**, teg bo'yicha
emas. Konteyner o'zi yaratilgan image'da qolib, o'zini butunlay sog'lom deb
ko'rsatadi — deploy'ning yashil bo'lib turib yolg'on bo'lishi shundan. Endi
konsol jonli konteyner image ID'larini joriy teg bilan solishtiradi, ya'ni
qo'lda qayta yaratilgan yoki rollout'dan keyin ochilgan mijoz ham hisobga
tushadi.

Qoidalar: **bittadan**, va har biri o'z `/health` iga javob bergandan keyin
keyingisiga o'tiladi; **bir vaqtda bitta rollout** (deploy'dagi `flock` bilan
bir mantiq); **ketma-ket 3 xato — to'xtash** (bitta mijoz — support tiketi,
buzuq image — avariya, va davom etish ellikta restoranni birma-bir o'chirardi);
**to'xtatilgan mijozlar o'tkazib yuboriladi**, aks holda to'lamagan mijoz
jimgina qayta yoqilardi.

Ishga tushishi: control konteyneri har deploy'da almashtiriladi, shuning uchun
rollout **boot'dan ~90 soniya keyin** o'zi boshlanadi (`ROLLOUT_ON_BOOT=0` —
o'chirish). Oddiy reboot'da hech nima qilmaydi: image ID'lar o'zgarmagan.
Deploy skriptining ichida emas — ellikta restoranni yangilash daqiqalar oladi
va o'n ikkinchi restoran sekinligi tufayli yiqilgan deploy tuzatilgan
muammodan battarroq bo'lardi.

### 3. Mijoz domenini avtomatik ulash
Endi oxirgi qadam ham egasining o'zida: DNS to'g'ri bo'lsa "Ulash" tugmasi
chiqadi va domen o'zi ulanadi.

**DNS — egalik isboti.** Domenni bizning serverga yo'naltirishni faqat
registrator hisobidagi odam qila oladi, ya'ni bu aynan aytilayotgan da'voning
o'zi (Vercel va Netlify ham shu dalilni qabul qiladi). Shu sababli boshqa
tekshiruv kerak emas.

Tenant → control kanali: token `HMAC(secret, slug)` — hisoblanadi, saqlanmaydi
(kiosk kodlari bilan bir naqsh), ya'ni konteyner kalitini shunchaki yaratilishi
bilan oladi va migratsiya qilinadigan narsa yo'q. Solishtirish
`ConstantTimeCompare`, chunki endpoint har tenant konteyneridan yetib boradi.

⚠️ **Control DNS'ni o'zi qayta tekshiradi.** Tenant ham tekshiradi — lekin u
qulaylik uchun: tenant serveri mijozning tomonidagi mashina, va u "men
tekshirdim" deb aytish orqali domen egallay olmasligi kerak. Platformaning o'z
nomlari (`keel.uz` va ostidagilar) umuman da'vo qilinmaydi: aks holda mijoz
bizga CNAME qo'yib `admin.keel.uz` ni "ulab" olardi.

### 4. uploads papkasining egaligi
Tenant serveri endi **root emas** (`app`, uid 10001). Entrypoint root bo'lib
faqat uploads papkasini `app` ga o'tkazadi va darhol `su-exec` bilan tushadi —
bind mount'ni Docker root egaligida yaratadi, konteynerning bunga ta'siri yo'q,
shuning uchun tartib aynan shunday bo'lishi kerak.

Nega muhim: bu server internetdan fayl qabul qilib, nomi so'rovdan kelib
chiqadigan fayllarni diskka yozadi — ya'ni xato aynan shu yerda ixtiyoriy fayl
yozuviga aylanadi. Server jarayoniga root umuman kerak emas (u 8080 ni
tinglaydi, 80 ni emas).

Rekursiv `chown` faqat papka hali bizniki bo'lmasa ishlaydi: bu root'da
ishlagan install uchun bir martalik migratsiya, o'n ming rasmli mijozning har
restartida takrorlanadigan ish emas. Docker'da tekshirildi: uid 10001, yozuv
ishlaydi, root'dan qolgan fayllar bir marta ko'chadi, ikkinchi ishga tushishda
qayta chown bo'lmaydi.

⚠️ `docker exec` ENTRYPOINT'ni chetlab o'tadi va **root** beradi — `seedmenu`
kabi rasm yozadigan buyruqlar `-u 10001` bilan chaqiriladi (DEPLOY.md).

### 5. Hisob-faktura daftari — va naqd to'lov
Ilgari yagona moliyaviy holat `suspended` edi: "pul kelmadi" deydi, xolos —
qancha, qaysi oy uchun, kim olishi kerakligini emas.

**Naqd — vaqtinchalik chora emas, MVP'ning o'zi.** Keel MChJ ochilmasdan
boshlanadi: yuridik shaxs yo'q, shartnoma yo'q, pul qabul qiladigan bank hisobi
yo'q. Odam restoranga borib naqd oladi. Daftar buni **birinchi darajali fakt**
sifatida yozadi, bank ishtirok etgandek ko'rsatmaydi. `transfer` birinchi
kundan modelda turadi (ishlatilmagan holda) — keyin qo'shish har bir saqlangan
qatorni qayta o'qishni talab qilardi.

Ikkita qoida buni jamlagichdan daftarga aylantiradi:
- **Summa chiqarilganda muzlatiladi.** Kunlik qatorlar to'planaveradi; mijozga
  aytilgan raqam esa o'zgarmaydi.
- **To'lov — ismi bor yozuv.** Imzosiz naqd uch haftadan keyin bahsga aylanadi
  (kuryerning naqd topshiruvi va ishchi oyligi bilan bir sabab). To'lovlar
  qo'shiladi, chunki pul bo'lak-bo'lak keladi.

Hisob **yopilgan** davr uchun chiqariladi, ishlab turgani uchun emas; davr
tenantning o'z langaridan olinadi, ya'ni dashboard bilan hech qachon
kelishmovchilikka tushmaydi (31-kunga langarlangan mijoz qisqa oylarda orqaga
surilmasligi ham testda). `(tenantId, from, to)` unique — ikki marta bosish
ikkinchi qarz yaratmaydi. Bekor qilishda sabab majburiy.

Hisob chiqarish hech kimni o'chirmaydi: `suspended` operator ataylab bosadigan
tugma bo'lib qoladi.

---

## 2026-08-06 (kechqurun) — konsol ko'rsatkichlari va watermark ✅

### 🐛 Watermark umuman chizilmagan edi
Galochka bosilmagan mijozda ham "Powered by Keel" ko'rinmasdi. Sabab kutilgandan
oddiyroq: control plane `hideWatermark` ni saqlardi, konsolda galochka bor edi,
statistikada "watermarksiz" soni ham sanalardi — **lekin sayt tomonida qator
umuman yozilmagan edi**. Ya'ni "yashirish buzilgan" emas, xususiyatning ikkinchi
yarmi mavjud emas edi.

Endi `(site)/layout.tsx` har so'rovda `showWatermark()` ni chaqiradi: Host →
control `/internal/resolve` → `hideWatermark`, slug bilan bir keshda (60 s).

**Ish vaqtida, muhitdan emas** — galochka mijoz to'lagan payt bosiladi, va
konteynerga yaratilishda berilgan qiymat qayta provisioning qilinmaguncha eski
bo'lib qolardi. Bu `rewrites()` ning build vaqtidagi tuzog'i bilan aynan bir
xil, faqat bir qavat narida. Mustaqil o'rnatilgan sayt (Keel tenant'i emas) va
noma'lum host qatorni hech qachon ko'rsatmaydi.

### Mijoz kartochkasi: raqamlar va grafiklar
`GET /tenants/{id}/live` (`internal/tenantstats`) — bugungi buyurtmalar va
bekor qilinganlar, tushum, o'rtacha chek, buyurtma turlari, hozir oshxonadagi
navbat, mijozlar (jami / yangi / buyurtma bergan), ishchilar (**hozir smenada**
turganlar bilan), kuryerlar holati bo'yicha, menyu, filiallar, brendlar,
bronlar va 30 kunlik top taomlar.

**Bu umumiy ro'yxatning ataylab teskarisi.** Bosh sahifa har tenant bazasiga
qo'ng'iroq qilsa, har sotilgan mijoz bilan sekinlashadi — kechki yig'uvchi
shuning uchun bor. Bitta kartochka esa boshqa savol beradi ("hozir shu
restoranda nima bo'lyapti?"), va javobning katta qismi vaqt qatori emas.

⚠️ **Alohida endpoint**, `GET /tenants/{id}` ga qo'shilmagan: bu chaqiruv sekin
bo'lishi yoki yiqilishi mumkin (to'xtatilgan konteyner, migratsiyadagi tenant),
va qo'shib yuborilsa **konteyner holatini ko'rsatadigan sahifani** o'ldirardi —
ya'ni aynan biror narsa noto'g'ri bo'lgani uchun ochilgan sahifani.

To'xtatilgan mijozda nollar to'g'ri, tirik mijozda nol — muammo belgisi.
Shuning uchun javobda `error` bor va kartochka buni yozadi: nollarni xotirjamlik
deb o'qish mumkin emas.

`TenantDay` ga `cancelled` qo'shildi — **sanaladi, lekin hisobga qo'shilmaydi**.
Qatorlar faqat hisob-faktura qila oladigan narsani tutsa, "bu mijozda bekor
qilishlar ko'payyapti" ko'rinmay qoladi — va aynan o'sha mijoz qo'ng'iroq qiladi.

Grafiklar kutubxonasiz. Palitra tekshiruvchi skript bilan **ikkala fon uchun**
tekshirildi (qo'shni CVD ΔE 9.2 / 9.4, oddiy ko'rish 27.6 / 26.5). Bekor
qilingan qizil — **status rangi**, kategorik uyacha emas. Sanoq va so'm hech
qachon bitta grafikda emas: ikki o'lchov bitta ramkada ma'nosiz kesishish
nuqtasi yasaydi. Chizib ko'rilganda bitta muammo chiqdi — eng katta qiymat ikki
sana orasida turib **uchinchi sana** bo'lib o'qilardi; u grafik tepasiga
ko'chirildi.

---

## 2026-08-06 (kech) — uchta xato

### 🐛 2GIS kaliti hech qachon xaritaga yetib bormagan
Restoran kalitni panelda kiritib saqlaydi, baza to'g'ri yozadi — lekin xarita
baribir ishlamaydi.

Sabab bir qatorda: `GET /restaurant` javobi **o'ralgan**
(`{ restaurant, brand, branch, isOpenNow }`), `lib/mapKey.ts` esa kalitni
**yuqori darajadan** o'qirdi (`d.mapApiKey`). Ya'ni qiymat doim `undefined`
bo'lib, har xarita jimgina platforma kalitiga tushardi — u esa Keel'da bo'sh.
Natija: ega mutlaqo to'g'ri kalitni kiritadi, saqlaydi, va hech qayerda hech
qanday xato ko'rinmaydi.

### 🐛 Konsolda tanlangan filtr — qora tugmada qora yozuv
`text-cream` ishlatilgan edi. `cream` — **tenant ilovasining** tokeni,
keel-site palitrasida bunday rang yo'q; klass hech nima qilmagan va matn `ink`
ni meros qilib olgan, ya'ni `bg-ink` ustida qora bo'lgan. Tanlangan filtr bo'sh
yorliqqa o'xshab qolgan. `text-surface` ga o'tkazildi (konsolning boshqa
joylarida allaqachon shu ishlatiladi).

### Bo'sh grafik nima uchun bo'sh ekanini aytmaydi
"30 kunlik ma'lumot yo'q, eng faol mijozlar yo'q — bular ishlayaptimi o'zi?"
degan savolga ekran javob bera olmasdi, va bu — asl kamchilik. Bo'sh
grafikning **ikkita sababi bir xil ko'rinadi**: hali hech kim buyurtma
bermagan, yoki yig'uvchi birorta marta ishlamagan. Faqat ikkinchisi muammo.

Endi `aggregate.Run` o'z hisobotini yozadi (`collector_run`): qachon ishlagan,
nechta mijoz bazasiga yetgan, nechtasiga yeta olmagan (sababi bilan), nechta
kunlik yozuv chiqqan. `GET /stats` uni qaytaradi, konsol esa **faqat
tushuntirishga narsa bo'lganda** bitta qator chiqaradi. Yonida **"Hozir
yig'ish"** tugmasi — savolga bir bosishda javob beradi, keyingi soatlik tikni
kutmasdan.

Ranglar va sanoqlar kodi tekshirildi — "to'xtatilgan" hisoblagichida xato
topilmadi (`counts[t.Status]++`, to'g'ri ishlaydi). Agar raqam baribir 0
bo'lsa, mijozning holati haqiqatan `suspended` emas (masalan hali `trial`) —
buni ro'yxatdagi holat filtri bilan tekshirish mumkin.

---

## 2026-08-06 (tun) — keel.uz landing, status va bepul xizmat ✅

### Integratsiyalar bo'limi
Kassa, to'lov, SMS, xarita, telefoniya, tashqi yetkazish — har biri o'z
ikonkasi bilan. Ilgari faqat kassalar qatori bor edi. Tugallanmaganlari
yashirilmaydi ("tez orada"): halol "tez orada" suhbatni davom ettiradi, yo'qlik
esa tugatadi. Oxirgi kartochka — **"ro'yxatda yo'qmi? — ulab beramiz"**, va u
izoh emas, bo'limning teng yarmi.

⚠️ **Ikonkalar chiziladi, yuklanmaydi**: kassa sotuvchisining logotipiga bizda
litsenziya yo'q, hotlink esa ham huquqiy savol, ham buzilishini kutayotgan
rasm. Har guruhga bitta glif nima turdagi narsa ekanini aytadi; kimligini
nomlar tashiydi.

### Hamkorlar karuseli
`GET /partners` — Keel mijozlari, logolari o'z saytlaridan, havolasi o'z
domeniga.

**Ruxsat so'raladi**: faqat konsolda belgilangan tenant chiqadi
(`showcase`, standart o'chiq). Mijozning brendini bizning marketing
sahifamizga so'ramasdan qo'yish — shikoyati bor mijoz demak, va tavsiya
sahifasi norozilikdan omon qolmaydi. Faqat `active`: to'xtatilgan mijozning
logosi buzilgan rasm bo'lib "to'lov kutilmoqda" sahifasiga olib borardi.

Logo jonli o'qiladi (nusxa olingani sekin eskirardi), 5 daqiqa keshlanadi,
hover'da to'xtaydi, `prefers-reduced-motion` ni hurmat qiladi, logosi yo'q
mijoz nomi bilan chiziladi.

### keel.uz/status
⚠️ Status sahifasining asosiy tuzog'i — **standart holatda yashil bo'lish**:
hech nima teskarisini aytmagani uchun "hammasi joyida" deydi. Bunday sahifa
yo'qidan yomonroq: u noto'g'ri bo'lgan yagona soatda dalil sifatida o'qiladi va
aynan o'zi qozonmoqchi bo'lgan ishonchni sarflaydi.

Shuning uchun **uchta holat**: ishladi / nosozlik / **ma'lumot yo'q** —
uchinchisi bo'shliq, na yashil na qizil. Namuna har daqiqada (o'z bazasi +
ishlashi kerak bo'lgan mijoz konteynerlari), soatlik guruhga yig'iladi:
daqiqasiga bitta hujjat yiliga yarim million, soatiga bittasi 8760 ta.
Boshqaruv xizmati javob bermasa — holatning o'zi shu, va sahifa buni aytadi
(dev'da tekshirildi: aynan shu xabar chiqdi).

### Bepul xizmat va chegirma
⚠️ **`pricePerOrder: 0` bepul xizmat emas.** Nol narx tasodifan tozalangan
maydondan farq qilmaydi, sababsiz nol summali hisob chiqaradi, va eng yomoni —
demo tugaganda kechki sweep mijozni baribir o'chirib qo'yadi.

Alohida bayroq: `free` + **majburiy** `freeReason` + `freeUntil` (bo'sh =
muddatsiz), va oraliq variant `discountPercent`.

Sabab: birinchi mijozlar o'z hisob-fakturasidan qimmatroq. O'n ikki filialli
tarmoq birinchi haqiqiy foydalanuvchi bo'lsa, u mahsulotga obro' sotib
olayapti — undan oyiga 200 ming so'm olish mavjud savdolarning eng yomoni.

Qoidalar bitta predikatda (`FreeAt` / `ChargeFor`):
- **Hisob baribir chiqariladi, 0 bilan** — oy bo'lgani va ataylab
  hisoblanmagani yozuvi; sabab izohga ko'chiriladi. Hisoblari orasida bo'shliq
  bor akkauntni keyin hech kim tushuntira olmaydi.
- **Sweep tegmaydi** — va'da bilan kelgan tarmoqning qatorida hisob ochilgan
  kundan qolgan demo sanasi turadi, va busiz sweep o'n ikki restoranni
  o'chirib qo'yardi.
- **"Qo'ng'iroq qilish" ro'yxatida chiqmaydi**, lekin **qo'lda to'xtatilgan**
  bo'lsa ko'rinadi: uni odam ataylab bosgan.

Hammasi testda muhrlangan (`free_test.go`), jumladan "nil = muddatsiz" — uni
"nol vaqtda tugagan" deb o'qish va'da berilgan mijozni birinchi kundan
hisoblab qo'yardi.

---

## 2026-08-06 — logotip: favicon va footer nishonchasi

keel.uz da favicon umuman yo'q edi. `KeelMark` (bitta shtrix, `currentColor`)
allaqachon shu maqsad uchun chizilgan ekan — endi u sarlavha, favicon
(`app/icon.svg` + `app/apple-icon.png`) va mijozning footer'idagi nishoncha
sifatida ishlatiladi.

⚠️ **Nishoncha `currentColor` da qoldi, Keel'ning sarig'ida emas.** U birovning
restorani ostida, ular tanlagan palitra ichida turadi — u yerda ikkinchi brend
rangi aynan egani "buni olib tashla" deyishga undaydi, va nishoncha faqat
tinch qoldirish oson bo'lgani uchun ishlaydi. Footer'ning o'chgan siyohini
oladi, hover'da restoranning o'z aksentini. Ikkala temada chizib tekshirildi.

Favicon faylida meros oladigan narsa yo'q → rang yozilgan, va bu **aksent**:
tab paneli ba'zi mashinalarda qora, ba'zilarida oq, va palitrada ikkalasida
ham o'qiladigan yagona qiymat — sariq.

Nishoncha `frontend` ga ko'chirildi, import qilinmadi: ikki alohida build, va
ikkita `<path>` uchun umumiy paket — abadiy qaraladigan bog'liqlik.

---

## 2026-08-06 — 🐛 Tushum buyurtma tushganda sanalardi

Restoran egasi topdi: 100 000 so'mlik buyurtma tushishi bilan dashboardda
"tushum" bo'lib chiqardi — hali pishirilmagan, kuryer chiqmagan, hech kim
to'lamagan.

Kod `o.Status != cancelled` bo'lsa yetarli deb hisoblardi. Ya'ni raqam
**kechroq to'g'ri bo'lib chiqardi va kun bo'yi noto'g'ri turardi** — bu
dashboarddagi raqam uchun eng yomon shakl: hech kim unga shubha qilmaydi,
shunchaki noto'g'ri reja tuzadi.

Endi pul **kelganda** sanaladi (`received()`), va u ikki yo'l bilan keladi:
- `paymentStatus: paid` — bank tasdiqladi. Karta to'lovi ovqat qimirlashidan
  oldin ham haqiqiy, va yetkazish bir soatdan keyin bo'lsa ham haqiqiy
  qoladi. Faqat `delivered` ni kutish oldindan to'lov oladigan restoranni
  kam ko'rsatardi.
- `delivered` — kuryer pul bilan qaytdi. Naqd shu degani, va olib ketish
  hamda stolda uchun ham yakuniy holat.

**Bekor qilingan hech qachon sanalmaydi, to'langan bo'lsa ham**: u qaytarib
beriladigan pul, va `paymentStatus` refund'dan keyin `paid` dan chiqadi —
lekin oradagi vaqtda ham sanalmasligi kerak.

Yoniga **"Kutilayotgan pul"** qo'shildi: tushgan, bekor qilinmagan, hali
olinmagan. Egaga kerak ("bugun yana qancha keladi"), lekin tushum emas.

O'rtacha chek endi olingan pulni **olingan buyurtmalar soniga** bo'ladi —
hammasiga bo'lish ikki asosni aralashtirardi va oshxona bandroq bo'lgan sari
o'rtacha chekni pasaytirardi.

Eng ko'p sotilgan taomlar ataylab eski asosda qoldi: u "nima sotilyapti"
degan savolga javob beradi, va tasdiqlangan buyurtmadagi taom sotilgan.

Xuddi shu tuzatish konsoldagi mijoz kartochkasiga ham qo'llandi — ikki ekran
bir kun haqida turlicha gapirmasligi kerak.

⚠️ **Ochiq qoldi**: control plane'ning kechki yig'uvchisi (`TenantDay.Revenue`)
hali eski asosda — bekor qilinmagan hammasini qo'shadi. Hisob-faktura
buyurtmalar **soniga** qurilgani uchun pulga ta'sir qilmaydi, lekin konsolning
"Mijozlar tushumi" ustuni restoranning o'z dashboardidan yuqori turadi.
Tuzatish saqlangan qatorlarni ham qamraydi (eski qatorlar eski ma'noda
qoladi), shuning uchun ataylab alohida qoldirildi.

---

## 2026-08-06 — yig'uvchi ham tuzatildi; katta summalar ustunda bo'linardi

### Yig'uvchi endi olingan pulni sanaydi
Oldingi yozuvda "ochiq qoldi" deb belgilangan band yopildi: `TenantDay.Revenue`
ham `paymentStatus: paid` yoki `delivered` qoidasiga o'tdi. Endi uchta joy —
yig'uvchi, konsoldagi mijoz kartochkasi va restoranning o'z dashboardi — bir
kun haqida bir xil gapiradi.

**Hisob-fakturaga ta'sir qilmadi**: u `Orders × narx`, va `orders` hamon
oshxonaga yetgan har bir buyurtmani sanaydi. Ovqatni pishirgan restoran
mehmon uyda bo'lmaganida ham hisob oladi.

**Migratsiya kerak bo'lmadi**: yig'uvchi har soatda oxirgi 35 kunni qayta
yozadi, ya'ni maydon ma'nosi o'zgarganda yaqin tarix keyingi tikda o'zini
tuzatadi. Platforma kecha ishga tushgani uchun butun tarix shu oynada.

### 🐛 Katta summa ustun bo'lib bo'linardi
Konsoldagi mijozlar jadvalida katta tushum "128 450 / 000" bo'lib ikki qatorga
bo'linardi — raqam ustuni raqam bo'lishdan to'xtardi.

Sabab: `money()` va `formatPrice()` guruhlar orasiga **oddiy bo'shliq**
qo'yardi, u esa satr uzilish nuqtasi. Tor katakda brauzer aynan o'sha yerdan
uzadi. Endi **U+00A0** (uzilmas bo'shliq), va pul kataklariga
`whitespace-nowrap` — ajratuvchi allaqachon uzilmas, lekin pul ustuni bitta
CSS o'zgarishi bilan yana yiqiladigan holatda turmasligi kerak.

⚠️ Bu faqat summalar uzayganda ko'rinadi — ya'ni ustun eng muhim bo'lgan
paytda. Ikkala formatter ham tuzatildi (restoran paneli va konsol).

---

## 2026-08-06 — mijozlar jadvali qayta tuzildi (uzilmas bo'shliq yetmadi)

Oldingi tuzatish raqamning **bo'linishini** to'xtatdi, lekin jadval baribir
sig'masdi: 7 ta ustunning uchtasi uzun so'm summasi, sahifa esa 1180px bilan
cheklangan. Uzilmas bo'shliq bilan raqam endi bo'linmaydi — o'rniga jadval
chiqib ketadi va "Amallar" tugmasi kesiladi. Ya'ni belgi almashtirish
yetarli emas edi, tuzilishning o'zi noto'g'ri.

Bu safar taxmin qilmasdan: haqiqiy sahifa stub API bilan ishga tushirildi va
brauzerdan `scrollWidth − clientWidth` o'lchandi. Eski holatda toshib
ketardi; yangisida 1280 / 1180 / 1024 da **overflow: 0**.

Uchta o'zgarish:

- **Summalar qisqartirildi**: `128,4 mln`, `1,28 mlrd`. So'mda kichik birlik
  yo'q va kattaliklar katta — haqiqiy raqam o'n-o'n bir belgi, va bir qatorda
  uchtasi hech qanday noutbukda sig'maydi. Bu, qolaversa, odamlar ovoz
  chiqarib aytadigan shakl. **Aniq qiymat doim `title` da**, va joyi bor har
  bir ekranda (mijoz kartochkasi, hisob-faktura) to'liq qoladi:
  yaxlitlangan raqam ro'yxatni ko'zdan kechirish uchun, mijozga aytish uchun
  emas. 1 mln dan pastda aniq qoladi — dastlabki hisoblar bir necha ming
  so'm, va ularni "0,2 mln" ga aylantirish ustunni nollar ustuniga aylantirardi.
- **Davr oralig'idan yil olib tashlandi** (`06.07 — 06.08`): qatordagi eng
  keng element edi va eng kam muhimi. To'liq sanalar mijoz kartochkasida.
- **"Ochilgan" ustuni nom ostiga ko'chdi** — bitta ustun kamaydi, va u
  hech kim ko'z yugurtirib qidirmaydigan ma'lumot.

---

## 2026-08-06 — pog'onali narx, "ulush" ustuni, landingda narx

Egasining savolidan boshlandi: "kuniga 400 buyurtma qiladigan restoran oyiga
12 mln so'm to'larkan — bu yaxshi narxmi?"

Hisob shuni ko'rsatdi: **foiz sifatida arzon** (ularning tushumining 0,5–2%,
agregatorlarda 15–20%), lekin **absolyut raqam katta** — 12 mln bu yerda o'rta
dasturchining oyligi, va aynan o'sha nuqtada tarmoqning moliyachisi hisob-kitob
qila boshlaydi. Yana bir narsa: bizning xarajatimiz buyurtma bilan o'smaydi
(400/kunlik restoran 20/kunlikdan deyarli farq qilmaydi), ya'ni yuqorida
chegirma berish amalda hech nima turmaydi.

⚠️ Yo'l-yo'lakay ma'lum bo'ldiki, **SAAS.md 400 buyurtma/OY deb hisoblagan** —
ya'ni butun "50 restoran = 20 mln/oy" bahosi bu hajmda hech qachon
sinalmagan. Hujjatga ogohlantirish yozildi.

### Pog'onali narx
3 000 gacha 1 000 · 3 000–10 000 → 700 · 10 000+ → 500.

**Shift emas, pog'ona**: shift qo'yilsa undan keyingi buyurtma bizga umuman
pul keltirmaydi — ikkala tomon uchun noto'g'ri rag'bat. Pog'onada marjinal
narx musbat qoladi, o'rtacha tushadi. 400/kun: 12 → **8,9 mln**, o'rtacha 742
so'm. Realistik 50 mijozli aralashmada platforma atigi **−7%** beradi, chunki
ko'pchilik birinchi pog'onadan chiqmaydi.

⚠️ **Davr bo'yicha, kunlik emas.** Kunlik qo'llansa pog'ona har yarim tunda
qaytadan boshlanadi va katta restoran birinchi banddan hech qachon chiqmaydi —
narvon umuman ishlamaydi. `TenantDay.Billable` tekis kunlik baho bo'lib
qoladi; hisob-faktura va mijozlar ro'yxati davr buyurtmalari sonidan qayta
hisoblaydi. Sakkizta test bilan mahkamlandi.

### "Ulush" ustuni
Hisob ÷ restoran tushumi, ranglar bilan (2% gacha jim, 2–3% sariq, 3%+ qizil).
Churn'ni oldindan aytadigan yagona raqam. Tushum bo'lmasa `—`, "0%" emas:
noldan ulush 0% emas, **noma'lum**, va "0%" ro'yxatdagi eng arzon mijozdek
o'qilardi.

### Landing
Narx blokida pog'ona jadvali: "Qancha ko'p buyurtma — shuncha kam to'laysiz".
Izohda aniq misol (12 000 buyurtma → o'rtacha 742 so'm, tekis narxdan 26%
arzon), chunki o'sayotgan restoran baribir shu hisobni o'zi qiladi — javobni
sahifada topgani yaxshi.

Ustun qo'shilgach jadval kengligi qayta o'lchandi: 1280/1180/1024 da
overflow 0.

---

## 2026-08-06 — landing matni: asosiy dalil oldinga chiqarildi

Sarlavha "Biznesingiz nimaga tayanadi" edi — mavhum, va hech nima sotmaydi.
Endi sahifa aynan restoran o'ylayotgan narsadan boshlanadi:

**"Doimiy mijozingiz sizga 20% turmasligi kerak."**

Hero'dagi uchta raqam ham almashtirildi: "Keel oladi 1–2%" · "Agregatorlar
15–20%" · "Oylik to'lov yo'q". Taqqoslash birinchi ekranda ko'rinadi.

### Yangi "Nega arzon" bo'limi
Uchta o'lchamdagi aniq jadval: oyiga 1 500 / 3 000 / 12 000 buyurtma uchun
restoran tushumi, agregator komissiyasi, bizning hisob va **sizda qoladi**.
Oxirgi ustun aksent rangida — sahifa aynan shu raqam uchun bor.

⚠️ **Halol ogohlantirish jadvalning ostida, izoh sifatida emas, dalilning
qismi sifatida**: agregator yangi mijoz olib keladi, biz esa yo'q — biz o'z
kanalini beramiz. Buni yashirgan taqqoslashni ikkalasini ham yuritgan
birinchi restoran egasi darhol tutadi, va o'shanda butun sahifaga ishonch
yo'qoladi. Ochiq aytilganda esa dalil kuchliroq bo'ladi: gap agregatordan
ketishda emas, **qayta keladigan mijoz uchun abadiy 20% to'lamaslikda**.

### Matn faqat restoranga qurilib qolmasin
Birinchi tahrirdan keyin hero restoranga tor bo'lib qolgan edi — agregator
dalili ovqat yetkazish tilida yozilgani uchun. Keel esa dorixona, gul do'koni,
butik va oddiy do'kon uchun ham.

Tuzatildi: hero yozuvi **"Sotadigan har qanday biznes uchun"**, matn oxirida
sohalar sanaladi, va "agregator" o'rniga **"agregator va marketplace'lar"** —
Uzum Market kabi platformalar ham xuddi shunday komissiya oladi. Bo'limlardagi
ovqat tili ham almashtirildi: "taom" → "narsa", "oshxonaga tushmaydi" →
"tayyorlashga tushmaydi", "Menyu bizda" → "Katalog bizda", "bitta oshxona,
bitta menyu" → "bitta joy, bitta katalog".

"Kimlar uchun" bo'limidagi "Menyu ham, tokcha ham, katalog ham" va
restoran/kafe kartochkalari ataylab qoldirildi — u yerda sohalar ro'yxati
maqsadning o'zi.

### Savol-javob
"Ma'lumotlarim kimga ko'rinadi?" olib tashlandi. O'rniga sotuvga
yordam beradigan savol: **"Agregatorda ham turibman — ikkalasini birga
yuritsam bo'ladimi?"** Javobi ha, va ko'pchilik aynan shunday boshlaydi —
bu o'tishdagi eng katta qo'rquvni olib tashlaydi.

---

## 2026-08-06 — trafik, SEO, fikr qutisi va server holati

### 🐛 Beshta yulduzda izoh yozib bo'lmasdi
Baho 3 dan yuqori bo'lsa komponent uni **darhol yuborardi** va izoh maydoni
umuman chiqmasdi. Natijada biznes ishlata oladigan sharhlar yo'qolardi
("kuryer juda xushmuomala edi"), va bundan yomoni — egaga yetib boradigan
yagona yozma fikr **faqat shikoyat** bo'lib qolardi.

Endi qutisi hamma bahoda chiqadi, savol esa uchiga qarab o'zgaradi ("Nima
noto'g'ri bo'ldi?" / "Nima yoqdi?") va ixtiyoriy ekani yozib qo'yilgan —
xursand mijoz baribir yozmasdan yuborishi mumkin. Backend allaqachon har
bahoda izohni qabul qilardi; cheklov faqat frontendda edi.

### Tashrif hisobi
`POST /visit` — sayt sahifasidan otiladigan mayoq. Panel nechta buyurtma
kelganini aytardi-yu nechta odam qaraganini aytmasdi, bu esa "hech kimga
kerak emas" bilan "hech kim topa olmayapti" farqi.

Bir kunga bitta qator (`(date, vid)` unique) → noyob tashrifchi = qatorlar
soni, sahifa ochilishi = `views` yig'indisi.

⚠️ **Belgi kun bilan hash qilinadi**, ya'ni ertaga o'sha brauzer boshqa qator.
Shuning uchun 30 kunlik raqam **tashrif-kun**, alohida odamlar emas — va bu
ataylab: biroz yuqori raqam odamni kuzata oladigan tizimdan yaxshiroq.
Qidiruv robotlari o'z-o'zidan tushmaydi (mayoq JS'dan otiladi). Qatorlar TTL
bilan 100 kundan keyin o'chadi: bu biznes bilan emas, **trafik bilan**
o'sadigan yagona kolleksiya.

Konsolda: kartochkada bugungi/30 kunlik trafik va 30 kunlik grafik; kunlik
qatorlarga ham qo'shildi.

### SEO
`robots.txt` va `sitemap.xml` — ikkalasi ham **har host uchun**, chunki bitta
build hamma restoranga xizmat qiladi. Sitemap'da har bir mavjud taom sahifasi
(aynan ular qidiruvdan odam tushadigan sahifalar). `metadataBase` va canonical
so'rovdan olinadi: build'ga muhrlansa hamma restoran birovning domenini
yozardi, va Google buni xato deb aytmaydi — shunchaki sahifalarni birlashtirib
yuboradi.

JSON-LD qo'shildi — ko'k havolani ish vaqti, telefon va xarita nuqtasi bilan
kartochkaga aylantiradigan narsa. Turi biznesga qarab (`Restaurant`,
`Pharmacy`, `Florist`…): gulchiga "oshxona" deb aytish structured data butunlay
e'tiborga olinmasligining yo'li.

keel.uz uchun ham robots/sitemap va to'liq metadata.

### Server holati
`GET /system` — CPU, xotira, disk, yuk, uptime + Docker nimani egallagani.

⚠️ Disk birinchi bo'lib va eng jimgina tugaydi. Shuning uchun "bo'shatish
mumkin" alohida ajratilgan: image'lar bitta buyruq bilan qaytadi, volume'lar
— mijoz fotosuratlari va qaytmaydi.

⚠️ Xotira `MemAvailable` bo'yicha ("free" sog'lom Linux'da deyarli nol),
disk `Bavail` bo'yicha (oxirgi foizlar root zahirasi). `UPLOADS_ROOT`
konteynerga read-only mount qilindi — busiz statfs konteynerning o'z
overlay'ini o'lchaydi.

---

## 2026-08-06 — kunning yakuni

O'n bir yozuv, o'n bitta commit. Yuqorida har biri alohida; bu yerda kun
nimadan iborat bo'lgani.

### Yopilgan bandlar (5-avgustdan qolgani)
Beshalasi ham: SMS provayderlari, rolling update, domenni avtomatik ulash,
uploads egaligi, hisob-faktura daftari.

### Qo'shilgan yangi narsalar
- **SMS**: to'rt provayder, har restoran o'zi tanlaydi va o'z kalitini qo'yadi
- **Rolling update**: deploy endi mijozgacha o'zi yetadi (image ID bo'yicha)
- **Domen**: DNS egalik isboti, oxirgi qadam ham egasida
- **Hisob-faktura daftari** + naqd to'lov (MChJ'gacha yagona yo'l)
- **Bepul xizmat va chegirma** — katta tarmoqni jalb qilish uchun
- **Pog'onali narx** (1000 / 700 / 500) — eng yaxshi mijozni jazolamaslik uchun
- **Konsol**: mijoz kartochkasida jonli raqamlar va grafiklar, rollout paneli,
  yig'uvchi holati, hisoblar bloki, "ulush" ustuni, server holati
- **keel.uz**: integratsiyalar, hamkorlar karuseli, status sahifasi, favicon,
  qayta yozilgan matn (agregator taqqoslashi bilan)
- **SEO**: robots, sitemap (har host uchun), JSON-LD, canonical
- **Tashrif hisobi**: nechta odam kirdi, nafaqat nechta buyurtma

### Topilgan xatolar
Ko'pchiligi bir xil shaklda: **kod to'g'ri, ulanishi noto'g'ri**, va hech
qayerda xato chiqmaydi.

1. **2GIS kaliti hech qachon xaritaga yetmagan** — javob o'ralgan, kalit
   yuqori darajadan o'qilardi. Ega to'g'ri kalit kiritadi, saqlaydi, xato yo'q.
2. **Watermark umuman chizilmagan** — control saqlardi, konsolda galochka bor
   edi, sayt tomonida esa qator yozilmagan.
3. **Konsolda tanlangan filtr** — `text-cream` bu palitrada yo'q, qora fonda
   qora yozuv.
4. **Hamkorlar/status/domen 404 olardi** — uchta chaqiruvchi prefiksni taxmin
   qildi, uchalasi ham xato. Har biri 404 ni ataylab yutadi, ya'ni landing
   aynan avariyaga o'xshab turdi.
5. **Tushum buyurtma tushganda sanalardi** — egasining o'zi topdi. Raqam
   kechroq to'g'ri bo'lardi va kun bo'yi noto'g'ri turardi.
6. **Katta summa ustunda bo'linardi** — avval uzilmas bo'shliq qo'ydim,
   yetmadi; jadvalning o'zi qayta tuzildi.
7. **Beshta yulduzda izoh yozib bo'lmasdi** — egaga yetadigan yagona yozma
   fikr faqat shikoyat bo'lib qolardi.
8. **Tashrif mayog'i umuman route qilinmagan** — handler yozilgan,
   kompilyatsiya bo'lgan, deploy qilingan, hech qayerga bog'lanmagan.
9. **Deploy skripti o'zini deploy qilmaydi** — tuzatish commit qilingan-u
   serverda eski skript ishlab turgan. Yagona iz bitta `echo` dagi so'z farqi.
10. **keel-site tekshiruvida bitta urinish** — control ga o'ttiz berilgan edi.

### Nima o'rganildi
⚠️ **Handler to'g'ri bo'lishi bilan handler ulangani bir narsa emas.** Bir
kunda ikki marta (4 va 8). Ikkalasi ham faqat **haqiqiy routerni aylanib
chiqib** topiladi — endi ikkala tomonda ham router testi bor, va ikkalasi ham
xatoni ushlashi tekshirildi.

⚠️ **404 ni yutadigan chaqiruvchi xatoni ko'rinmas qiladi.** Hamkorlar qatori
bo'sh chiqadi, status "javob bermayapti" deydi, mayoq jim turadi — hammasi
o'z haqiqiy nosozligi uchun to'g'ri xatti-harakat, va hammasi noto'g'ri
manzildan farq qilmaydi.

⚠️ **"Yashil, lekin yolg'on" oilasi kattaroq ekan.** Kecha bu konteyner
image'i edi; bugun deploy skriptining o'zi, va route qilinmagan handler.

### Infratuzilma
- Prod `169.58.131.165` da (eski xotiradagi IP boshqa quti)
- Repozitoriya nomi `template` → `Keel.uz`
- Deploy skripti serverda yangilandi va endi o'zining eskirganini aytadi
- ⚠️ **GitHub Actions'da katta uzilish** (15:22 UTC dan) — kechki deploy'lar
  navbatda qotib qoldi. Sabab bizda emas; kod qo'lda deploy qilindi
  (git bundle → build → almashtirish → tekshiruv) va jonli ishlayapti.

---

## 2026-08-06 — 🔒 demo SMS rejimi kodni javobda qaytarardi

Shlyuz sozlanmagan install demo sender'ga tushadi, demo sender esa bir martalik
kodni **API javobida** beradi — bu uning butun maqsadi, chunki busiz dasturchi
pullik hisobsiz oqimni sinay olmaydi.

Jonli tenantda esa bu shuni anglatardi: yangi ochilgan restoran saytida
**istalgan odam istalgan bo'lib kira olardi**. Begona raqam uchun kod
so'raysiz, uni JSON'dan o'qiysiz, va siz o'shasiz — buyurtmalari, manzillari,
tarixi bilan.

⚠️ Hech qayerda xato chiqmasdi: so'rov muvaffaqiyatli, javob to'g'ri shaklda,
va faqat JSON'ni o'qigan odam ko'rardi. Shu sababli u deploy bo'lib ketgan.

Endi:
- Kod faqat `SMS_DEMO_EXPOSE_CODE=1` bo'lganda qaytariladi. Standart **o'chiq**;
  lokal `docker-compose.yml` va `.env.example` da yoqilgan, prod va SaaS
  compose'larida umuman yo'q, va control tenant konteynerlariga uzatmaydi.
- **Shlyuz yo'q = login yo'q**: kod so'rash 503 bilan rad etiladi, va rad etish
  kod yaratilishidan oldin bo'ladi. Bu to'g'ri nosozlik — "SMS sozlanmaguncha
  hech kim kira olmaydi" qo'llab-quvvatlash qo'ng'irog'i, "istalgan odam
  istalgan bo'lib kira oladi" esa qaytarib bo'lmaydi. Mehmon nima qilishni
  biladi; jimgina qabul qilinsa, u hech qachon kelmaydigan SMS'ni kutardi.
- Qaror sof funksiyaga ajratildi (`exposeDemoCode`) va testda muhrlandi,
  jumladan: **haqiqiy shlyuz bayroqdan qat'i nazar kodni qaytarmaydi** —
  bayroq faqat demo yo'lini bo'shatadi.

To'rtta chaqiruv joyi ham o'tkazildi: mijoz login, raqam almashtirish, admin
parolini tiklash, admin tiklash raqami.

---

## 2026-08-07 — til URL'lari, tasdiqlash teglari, hisob turtkisi va narx poli

Kechagi ro'yxatdagi 4, 5, 6, 7-bandlar. 2 va 3 serverga tegadi va ochiq qoldi
(pastda).

### Qidiruv tizimlariga saytni tasdiqlash (4)
`restaurant.seo { google, yandex }` — sozlamalarda, domen bo'limining ostida.
Joyi ataylab shu yerda: bepul subdomen uchun olingan kod egasining o'z domenini
tasdiqlamaydi, ya'ni bu **domendan keyingi** qadam.

Kalitlar `restaurant` hujjatining ichida — to'lov kalitlarining teskarisi va
xarita kaliti bilan bir mantiq: tasdiqlash kodining butun vazifasi sahifa
`<head>` ida turish. Uni yashirish faqat tasdiqlashni buzadi.

⚠️ **Ikkala konsol ham egaga butun `<meta ...>` tegini ko'rsatib "nusxa oling"
deydi**, shuning uchun maydonga ko'pincha teg tushadi. Yopishtirilgan teg
`content` atributining ichiga yozilsa, sahifa normal chiziladi, hech qayerda
xato chiqmaydi va tasdiqlash ishlamaydi — egada esa maydondan shubhalanish
uchun sabab yo'q. `verificationToken()` ikkalasini ham qabul qiladi.

Bo'sh qiymat `undefined` bo'lishi shart: Next bo'sh satr uchun bo'sh teg
chizadi, bo'sh teg esa ikkala konsol uchun "teg bor, lekin noto'g'ri".

### Ko'p tillilik va SEO: `/ru/`, `/en/`, hreflang (5)
Sayt boshidan uch tilli edi, lekin faqat **cookie** orqali — robot esa cookie
tashimaydi. Ya'ni Google va Yandex uchun har sahifa aynan bitta tilda mavjud
edi: Toshkent restoranining ruscha menyusining **manzili yo'q edi**, demak
uni ulashib ham, indekslab ham bo'lmasdi. Aynan mijozlarning ko'pi ishlatadigan
qidiruvda.

Yechim `middleware.ts`: `/ru/menu` → `/menu` ga rewrite + til sarlavhasi.
`[lang]` marshrut segmenti emas — sahifa fayllari, `<Link>` lar va API yo'llari
o'zgarmadi; butun ilovani bir papka pastga ko'chirish ikki qator URL ishi uchun
juda katta narx. O'zbekcha **prefikssiz** qoladi: u asosiy til va uning
manzillari allaqachon QR kartochkalarda va indeksda.

⚠️ **Sarlavha avval o'chiriladi, keyin faqat haqiqiy prefiks bo'lsa qo'yiladi.**
Ikki sabab, ikkalasi ham jim: (a) mijoz istalgan sarlavhani yubora oladi, ya'ni
qiymat bizdan kelishi kerak; (b) prefikssiz URL'da **cookie** hukmron qolishi
shart — sarlavha shartsiz qo'yilganda har prefikssiz so'rov o'zbekchaga
qadaldi, va bunga **admin panel, kuryer va ishchi ilovalarining har bir ekrani**
kirardi. Ularda til URL'i yo'q, demak tillari shunchaki ishlamay qolardi —
almashtirgich esa harakatlanib turardi va cookie to'g'ri bo'lardi.

⚠️ **URL cookie'dan ustun.** Aks holda ulashilgan havola qabul qiluvchining
o'z tilida ochilardi, va yuboruvchi buni hech qachon ko'rmasdi — o'z ekranida
hammasi joyida edi. Shu sababli til almashtirgich endi **manzilga o'tadi**,
shunchaki `refresh()` qilmaydi: `/menu` da turib rus tilini tanlash va faqat
qayta chizish o'zbekchani qaytarardi, ya'ni tugma ishlamayotgandek ko'rinardi.

Yo'l davomida topilgan eski xato: root layout **har sahifa uchun sayt ildizini**
canonical deb e'lon qilardi. Bu qidiruv tizimiga menyu va har bir taom sahifasi
— bu yerdagi yagona reyting olishga arziydigan sahifalar — bosh sahifaning
nusxasi deb aytish. Xato sifatida hech qayerda ko'rinmaydi, sahifalar shunchaki
chiqmaydi. Endi canonical middleware bergan yo'ldan quriladi.

Boshqa joylar: `LocaleLink` (havolalar prefiksni saqlaydi — robot ruscha
sahifalar borligini **havolalardan** biladi), Header'ning `isActive` i (prefiksli
URL'da butun navbar yorug'ligini yo'qotardi), sitemap (har sahifa **bir marta**,
uch alternativa bilan — uch alohida yozuv aynan hreflang oldini oladigan dublikat
muammosi), robots (`/ru/checkout` — yo'l prefiksi bo'yicha yozilgan qoida yo'lning
o'zi prefiks olishi bilan mos kelmay qoladi), va tashrif mayog'i (uch tilni bitta
qatorga yig'adi, aks holda eng band sahifa uchta sokin sahifaga bo'linardi).

**Tekshirildi** (`next start` + haqiqiy backend): uch tilda kontent, canonical,
hreflang, sitemap, robots, prefiksli havolalar, va regressiya sifatida —
admin/kuryer/ishchi ekranlarida cookie tili, soxta sarlavhaning rad etilishi,
URL'ning cookie'dan ustunligi.

### "Davri yopildi, hisob chiqarilmagan" (6)
Yangi ogohlantirish `invoice_due`. Yagona **o'zimiz haqimizdagi** turtki: daftar
nima hisoblanganini yozadi, hisob chiqarishni esa hech kim so'ramasdi.

⚠️ **Eng jim ishlaydigan nosozlik turi**: hisob chiqarilmagan mijoz shikoyat
qilmaydi, mahsulotdan foydalanishda davom etadi va **to'lab bo'lgan mijozdan
umuman farq qilmaydi**.

Qoidalar: yopilgan davr — hozir ochiq turgan davr boshlangan kun; ochiq davrni
hisoblash hali o'sib turgan summani muzlatish bo'lardi. **Bekor qilingan
(void) hisob sanalmaydi** — u aynan noto'g'ri bo'lgani uchun bekor qilinadi, va
uni "hisoblangan" deb qabul qilish o'sha davrni jimgina yig'ib bo'lmaydigan
qilardi. Birinchi davr ichidagi mijoz navbatga tushmaydi (birinchi kuniyoq
ro'yxatga tushgan mijoz — operatorga navbat shovqin ekanini o'rgatadigan narsa).

Rangi qizil emas, sariq: hech nima buzilmagan va hech kim norozi emas — shunchaki
hech kim so'ramagan pul turibdi. To'xtatilgan mijoz yonida uni favqulodda holat
qilib ko'rsatish ekrandagi eng foydali ikki rangning farqini yo'qotadi.

⚠️ Bu yagona ogohlantirish **tenant hujjatining xususiyati emas** — u
daftarga bog'liq, ya'ni Mongo filtri bo'la olmaydi. Shu sababli filtr Go
tomonda qo'llanadi: muqobil yo'l — bir xil ma'noni abadiy saqlashi kerak bo'lgan
Mongo ifodasi va Go funksiyasi, va ular kelishmay qolgan kuni belgi bir narsani,
filtr boshqasini ko'rsatadi.

### Minimal oylik to'lov (7)
Pog'onali narx yuqoridan chegaralaydi; bu — pastdan. Kuniga 5 buyurtma qiladigan
restoran ~150 ming to'laydi va qo'ng'iroqlar, menyu tuzatishlari, "nega printer
chop etmayapti" — hammasi 400/kunlik mijoz bilan bir xil vaqt oladi.

⚠️ **Buyurtmasiz davr hech qachon hisoblanmaydi.** Nol buyurtma deyarli doim
"sayt hali ishga tushmagan" yoki "restoran yopiq edi" degani — mijoz bizdan
hech nima olmagan va buni biladi. Foydalanmagan oy uchun kelgan hisob — mijozni
yo'qotishning eng tez yo'li. Mavjudlik uchun pul olish himoya qilinadigan model,
lekin bu yerda hech kim unga rozi bo'lmagan, va u **polning yon ta'siri** sifatida
kelmasligi kerak.

Boshqa qoidalar: bepul shartlar poldan ustun (va'da berilgan bepul —
bepul); chegirma **polga** qo'llanadi, uning ostiga emas (aks holda chegirma
aynan uni so'ragan kichik mijozlar uchun hech nima qilmaydi); har mijoz o'z
polini saqlaydi (`tenant.minMonthly`, 0 = umumiy sozlama); **standart 0 —
o'chiq**, chunki pol haqiqiy mijozlar qarzini o'zgartiradi va bu deploy'ning
yon ta'siri emas, qaror bo'lishi kerak.

Hisob-fakturaga sabab yoziladi: restoran o'z buyurtmalarini sanay oladi, va
uning arifmetikasi bilan bizniki orasidagi tushuntirilmagan farq — eng yaxshi
holatda qo'ng'iroq.

### Jonli serverda topilgani (2 va 3-bandlar)

**2 — GitHub Actions tiklandi va tekshirildi.** Uzilish tugagan; bugungi push
avtomatik deploy'ni ishga tushirdi, image'lar qurildi, konteynerlar
almashtirildi. ⚠️ Kechagi oxirgi commit (`d0c889b`, 22:47) deploy bo'lmagan
ekan — CI 22:40 da `776a724` ni chiqargan va undan keyingi ish ketmagan.

⚠️ **Tekshirishning o'zi deyarli aldab ketdi**: push'dan ~30 soniya keyin
serverdagi `git HEAD` allaqachon yangi edi. Lekin kod yangilash — skriptning
**birinchi** qadami; to'rt image undan keyin quriladi. HEAD'ga qarab
"deploy tugadi" deyish — konteyner image'i tuzog'ining aynan o'zi. To'g'ri
belgi: qulf bo'shadimi va konteynerlar yoshi.

**3 — Domen ulash ishlamayotgan ekan, va sababi topildi.** `traderbot.uz`
bazada ulangan, javob `ok`, konsolda ro'yxatda — Caddy konfiguratsiyasida esa
**umuman yo'q** edi (na route, na TLS subject). Sababi `apply()` dagi
`else if`: konteyner ko'tarilmasa chekka qayta yozilmasdi.

Tuzatilgandan keyin: sertifikat **bugun olindi** (Let's Encrypt, 09:48 UTC),
`https://traderbot.uz` 200 qaytardi, uch til, hreflang va canonical ham
jonli tekshirildi. Ya'ni oqimning o'zi to'g'ri edi — faqat oxirgi qadam
hech qachon bajarilmasdi.

Sinovdan keyin `traderbot.uz` yechildi va `kfc` test tenanti yopildi.

### Uchinchi marta bir xil shakl
Bir kunda uchta: (1) ulangan domen chekkaga yetmasdi, (2) `provisionStatus:
"ready"` konteyner umuman yo'q bo'lsa ham, (3) deploy qulfi serverni emas,
foydalanuvchini qulflardi — CI `deploy-keel`, odam `root`, ikkalasi boshqa
fayl. Uchalasida ham hech qayerda xato chiqmaydi va hamma ko'rsatkich yashil.

⚠️ **Ochiq qolgan**: `provisionStatus` — saqlangan bayroq, va u eskiradi.
`kfc` bazada "ready" turgan holda konteyneri yo'q edi. `Attention` bilan bir
xil dars ("saqlangan bayroq soat undan o'tishi bilan eskiradi"), lekin bu
yerda hali qo'llanmagan. `containerStatus` faqat mijoz kartochkasida
hisoblanadi; ro'yxatda va turtkilarda yo'q.

---

## 2026-08-07 (kechqurun) — "sayt ishlamayapti" turtkisi va keel.uz SEO

### `provisionStatus` o'rniga hisoblangan holat
Ertalab topilgan xato yopildi: `attention: "down"` — faol yoki demo mijozning
sayti ishlamayapti, va buni hech kim xohlamagan.

Docker'dan **jonli** o'qiladi (`Docker.States()`), va **butun sahifa uchun
bitta so'rov** — har tenantga alohida inspect ro'yxatni N ta chaqiruvga
aylantirardi, bu esa Mongo'da ataylab qochilgan naqsh.

Qoidalari:
- **Hamma narsadan oldin**, hatto bepul shartlardan ham. Qorong'i sayt puldan
  muhimroq: qator o'qilayotgan paytda restoran buyurtma yo'qotayapti. Bepul
  mijoz *to'lov uchun ta'qib qilinishdan* ozod, ishlaydigan saytdan emas — u
  odatda aynan butun erta mahsulot tayanadigan anchor mijoz.
- ⚠️ **"So'ray olmadim" — "o'chiq" emas.** Docker soketi yo'q holat noutbukda
  normal, va doim yonib turgan ogohlantirishni hech kim o'qimaydi. `stateOf()`
  bo'sh xarita (so'ralmagan) bilan ro'yxatda yo'qlikni (`absent`) ajratadi.
- **`restarting` ham o'chiq**: qayta ishga tushib turgan konteyner oraliqda
  o'zini ishlayapti deb ko'rsatadi — buzilgan tenant sog'lomdan aynan shunday
  farqlanmay qoladi.
- `suspended`/`deleted` tekshirilmaydi — ular biz o'chirgan saytlar, va ularni
  nosozlik deb ko'rsatish har ketgan mijozni abadiy navbatda ushlab turardi.
- Ekrandagi yagona **to'ldirilgan** nishon. Pushti fon uchta boshqa pushti
  nishon yonida ko'rinmay ketadi — aynan shu bo'lgan edi.

### keel.uz SEO
Sayt topilishi kerak bo'lgan yagona joy, shuning uchun bu yerda SEO bezak emas.

- **Til URL'lari** `/ru`, `/en` + hreflang. ⚠️ Bu yerda sabab tenant saytidan
  kuchliroq: Toshkentdagi restoran egasi "сайт для ресторана с доставкой" deb
  **ruscha** qidiradi, va sahifaning ruscha varianti manzilsiz edi.
- **Canonical yo'l bo'yicha**, konstanta emas.
- ⚠️ **`summary_large_image` rasmsiz e'lon qilingan edi.** Bu betaraf standart
  emas: Telegram va WhatsApp — mahsulot aynan shu yerda ulashiladi —
  deklaratsiyani bajaradi va **bo'sh katta kartochka** chizadi, u esa o'lik yoki
  chala havolaga o'xshaydi. Kichik kartochka undan yaxshiroq bo'lardi.
  `opengraph-image.tsx` uni generatsiya qiladi (PNG fayl emas), shuning uchun u
  sahifa bilan bir xil uch tilda qoladi va sarlavhasi sahifadagidan ajrab keta
  olmaydi. Uch tilda ham chizilishi ko'rib tekshirildi (kirill ham).
- **JSON-LD**: `Organization`, `WebSite`, `SoftwareApplication` + narvon
  `Offer` sifatida — qidiruv natijasi ostiga "1000 so'm / buyurtma" qo'yadi.
- **Tasdiqlash teglari** muhit o'zgaruvchilaridan (bitta domendagi bitta sayt).

Tekshirildi: uch tilda kontent, canonical, hreflang, sitemap alternates,
robots (prefiksli shaxsiy yo'llar bilan), OG rasm (uz va ru), JSON-LD tarkibi,
va regressiya sifatida — konsol tili cookie bilan, soxta sarlavhaning rad
etilishi, URL'ning cookie'dan ustunligi.

---

## 2026-08-07 — ATMOS to'lov integratsiyasi (to'rtinchi provayder)

Beshta provayder so'raldi: Kaspi, Epay, TipTop Pay, Atmos, Anorbank. Hujjatlarni
o'qib chiqqach ikkita narsa aniqlandi, va ikkalasi ham kod yozishdan oldin hal
qilindi.

### ⚠️ Beshtadan uchtasi — Qozog'iston
Kaspi, ePay (Halyk) va TipTop Pay O'zbekistonda ishlamaydi: ular **KZT**.
Loyihaning butun puli esa UZS butun son (narx narvoni, `MIN_MONTHLY`, loyalty
"1 ball = 1 so'm", hisob-fakturalar), SMS shlyuzlari faqat O'zbekiston, telefon
formati `998XXXXXXXXX`, xarita 2GIS. Ya'ni bu "yana uchta adapter" emas, balki
**bozorga chiqish** — va uning birinchi qadami adapter emas, tenant darajasida
valyuta va davlat. Qaror: hozircha faqat O'zbekiston.

### ⚠️ Ikkala O'zbekiston provayderi ham karta raqamini serverga oladi
Atmos'ning `/merchant/pay/*` va Anorbank'ning butun API'si `pan` + `exp` +
OTP bilan ishlaydi — ya'ni mijozning kartasi restoranning saytiga kiritiladi va
**har bir tenant konteyneri PCI DSS qamroviga tushadi**. Bu hozirgi uchtasining
teskarisi: Payme/Click/Uzum'da karta Keel'ga umuman tegmaydi.

Atmos'ning **hosted invoice**'i bor (`/checkout/invoice/create` →
`checkout.atmos.uz` havolasi) — u bilan PAN bizga tegmaydi va arxitektura
o'zgarmaydi. Shu tanlandi. Anorbank hujjatida (29 sahifa) hosted sahifa yo'q,
shuning uchun u kechiktirildi.

### Qilingani
`handlers/payatmos.go` — OAuth2 token (keshlangan), invoice yaratish, callback.
Boshqa uchtasidan ikki tomoni bilan farq qiladi:

**Havola quriladi emas, so'raladi.** Yagona provayder bo'lib, uning havolasi
tarmoq sababidan yiqilishi mumkin: bunda bo'sh qator qaytariladi (tugma
ko'rsatilmaydi) va sabab logga yoziladi — mehmon "STORE_NOT_FOUND" bilan hech
nima qila olmaydi, restoran esa uni buyurtma raqami yonida topadi.

**Callback xabar emas, ruxsat.** Pul faqat biz `status: 1` desak yechiladi.
Ya'ni xato bilan rad etish yozuvni yo'qotmaydi — **haqiqiy to'lovni kassada
rad etadi**. Har bir rad etish shu sababli pul olish noto'g'ri bo'ladigan
holat: yo'q buyurtma, allaqachon to'langan, bekor qilingan, summa mos emas.

### Hujjatdagi ikkita bo'shliq
- **Imzoning hash funksiyasi yozilmagan** — faqat formula bor. md5/sha1/sha256
  uchalasi qabul qilinadi. Bu zaiflik emas: hash qilinadigan satr ichida
  `api_key` bor, bittasini yasay olmagan odam uchalasini ham yasay olmaydi.
- **Maydonlar turi yozilmagan** — `store_id`, `transaction_id`, `amount`
  qo'shtirnoq bilan ham, bo'lmasa ham kelishi mumkin. Faqat bittasini kutish
  har bir to'lovni rad etardi, va ikkala xato tashqaridan bir xil ko'rinadi:
  mehmon to'lov sahifasiga yetadi, to'laydi, va "bo'lmadi" deb eshitadi.

### ⚠️ Sinov vositasining o'zi bir marta aldadi
`paytest` ning birinchi yugurishida to'rtta "rad etish" testi **yashil**
chiqdi — aslida route umuman qo'shilmagan edi va `404 page not found` javobi
`status: 0` bo'lib o'qilardi. Ya'ni "to'g'ri rad etdi" bilan "endpoint yo'q"
bir xil ko'rinardi. Endi JSON bo'lmagan javob nosozlik hisoblanadi.

Router testi ham qo'shildi (`TestProviderCallbackPaths`) va **xatoni
ushlashi tekshirildi**: route olib tashlansa test yiqiladi. Bu marshrutlar
tizimda yagona bo'lib, ularning manzili **birovning kabinetiga yoziladi** —
ya'ni ularni o'z saytimizni sinab topib bo'lmaydi.

Yakuniy tekshiruv haqiqiy backend va Mongo bilan: oltala holat ham o'z
sababi bilan, buyurtma `paid` bo'ldi va oshxona navbatiga tushdi.

---

## 2026-08-07 — hisobot qatlami, Excel eksporti va ABC/XYZ

So'ralgani katta ro'yxat edi: fiskalizatsiya, 1C, naqd hisobi, moliyaviy
hisobotlar, ABC/XYZ + Excel, va ikkita agregator. To'rtta savol berildi va
javoblar ko'lamni sezilarli qisqartirdi:

- **Fiskalizatsiya — ish kerak emas.** POS (iiko/Clopos) o'zi fiskallashtiradi,
  buyurtma unga allaqachon ketadi. Eng yaxshi natija: qurilmagan modul.
- **Moliyaviy hisobot — faqat tushum.** Tannarx yo'q, ya'ni foyda ko'rsatilmaydi.
  Yolg'on raqamdan yaxshiroq.
- **Agregatorlar — shartnoma yo'q.** Kalitsiz yozilgan adapter — taxmin
  qilingan API.
- **1C — konfiguratsiya aniq emas**, universal eksport.

### Qurilgani: bitta hisobot qatlami
`handlers/report.go` — `Report{Title, From, To, Columns, Rows, Totals, Note}`,
va undan ekran (JSON) ham, Excel ham chiqadi. Sabab: odatda ekran avval
quriladi, eksport esa oylar keyin **so'rovning ikkinchi nusxasi** ustiga
yopishtiriladi, va ikki raqam bir-biriga mos kelmay qolganini hech kim payqamaydi.

⚠️ **CSV emas, haqiqiy .xlsx.** Ma'lumot o'zbekcha matn va so'm summalari, CSV
ikkalasini ham buzadi: ruscha/o'zbekcha Windows'dagi Excel vergulni o'nlik
ajratgich deb o'qiydi ("Lag'mon, katta" ikkiga bo'linadi, 92,000 → 92), BOM'siz
esa apostrof va kirill krakozyabra bo'ladi. Har biri moliyaviy hujjatning
jimgina buzilishi.

⚠️ **Raqamlar raqam bo'lib yoziladi.** "92 000" satrlari ekranda bir xil
ko'rinadi, lekin yig'ib, saralab yoki diagramma qilib bo'lmaydi — bu esa
skrinshot o'rniga jadval so'rashning asosiy sababi.

### ABC/XYZ (`/admin/reports`)
ABC — tushumdagi ulush, XYZ — talabning barqarorligi. Qaror kesishmada: AX
tugamasligi kerak, AZ to'lqin bilan keladi, CZ menyudan chiqadi.

Uchta qaror testda muhrlandi:
- ⚠️ **80% chizig'ini kesib o'tgan taom A'da qoladi** — kesim qo'shishdan
  *oldingi* jamlanma bo'yicha. Aks holda qisqa menyuda tushumning 40% ini
  ko'tarib turgan taom "ikkinchi darajali" bo'lib chiqadi.
- ⚠️ **Sotuvsiz kun — nol, tushib qolgan kuzatuv emas.** Faqat sotilgan kunlar
  o'rtachalansa, oyiga bir marta yigirma porsiya ketadigan taom eng barqaror
  bo'lib chiqardi — menyudagi eng tartibsiz narsa eng bashoratli deb.
- **Chegaralar 25%/60%**, darslikdagi 10%/25% emas: ular ishlab chiqarishdan,
  u yerda talab shartnomalar bilan silliqlangan. Kuniga uch porsiyada bitta
  tinch seshanba 30% tebranish — 10% bilan butun menyu Z'ga tushardi.

Tebranish yonida necha kun sotilgani turadi: o'ttiz kundan ikkitasida sotilgan
taomning koeffitsienti arifmetik to'g'ri va hech nima anglatmaydi.

Tekshirildi: haqiqiy backend + Mongo + brauzer — sahifa, ikki o'qli filtr
(tanlangan guruhning o'z summasi bilan), va yuklash tugmasi haqiqatan faylni
berdi. Fayl qaytadan o'qildi: bitta varaq, formatlangan summalar, raqamlar
matn emas.

---

## 2026-08-07 — naqd hisobi, moliyaviy hisobot va grafiklar

### Moliyaviy hisobot
Uchta savol uchta bo'limda ajratilgan: **kirim** (haqiqatan qo'lga tushgan),
**chiqim** (haqiqatan berilgan), **kutilayotgan** (tushgan, hali olinmagan —
va ataylab jamilar tashqarisida).

⚠️ **Bu foyda hisoboti emas va hisobotning o'zi shuni yozadi.** Tannarx yo'q,
demak "kirim − chiqim" pul harakati. Ma'nosi kod izohida emas, ekranda turishi
kerak: bunday raqam ertami-kechmi bank arizasiga tushadi.

Ikkita narsa xarajat emas va testda muhrlandi:
- **Chegirma va ballar** — pul chiqmagan, u umuman kelmagan; tushum qatori
  allaqachon ulardan tozalangan, ya'ni yana ayirish ikki marta hisoblash.
- **Kuryer topshirig'i** — bu kuryer bizning nomimizdan yig'gan naqdning
  kassaga kirishi. Chiqim deb sanash restoranning tushumini o'zidan ayirish.

### Kassa (naqd hisobi)
⚠️ **Mahsulot — farq, jami emas.** Kutilgan summani ko'rsatib, sanalganini
yozdirib, faqat ikkinchisini saqlaydigan ekran hech nima yozmagan: u ochish
uchun qurilgan kamomad uni qilgan bo'lishi mumkin bo'lgan odam tomonidan
o'chirilgan. Shuning uchun `expected` yopishda muzlatiladi, `variance`
saqlanadi, va **farq sababsiz saqlanmaydi**.

⚠️ **Yetkazishdagi naqd to'g'ridan-to'g'ri sanalmaydi** — u kassaga kuryer
topshirgandan keyin kiradi, ikkalasini sanash har yetkazishni ikkilantirardi.
Kuryer qo'lidagi pul alohida ko'rsatiladi: kamomadni tekshirayotgan ega
birinchi navbatda shu raqamni so'raydi.

Jonli sinaldi: smena ochish, ikkinchi marta ochishning rad etilishi, sababsiz
chiqimning rad etilishi, sababsiz yopishning rad etilishi, va −5 000 farqning
sabab bilan yozilishi.

### Grafiklar (Chart.js)
Uch sahifada: restoran dashboardi, keel konsoli va mijozlar ro'yxati.
Konsoldagi qo'lda yasalgan div-grafik olib tashlandi.

Palitra **rang ko'rish nuqsoni uchun tekshirildi** (validator, light va dark
alohida). ⚠️ Seriya ranglari **restoranning brend rangidan olinmaydi**: tenant
o'z aksentini tanlaydi va undan qurilgan shkala brending o'zgarganda ma'nosini
o'zgartirardi.

⚠️ **Ko'z bilan ko'rib bitta xato topildi**, va uni hech qanday tekshiruv
ko'rsatmasdi: y o'qida `0.5, 1.5, 2.5` turardi — yarim buyurtma yo'q. Chart.js
qadamni diapazondan tanlaydi. Tuzatildi (`precision: 0`) va qayta ko'rildi.

### ⚠️ Tekshirilmagani
keel konsolidagi grafiklar **aynan bir xil komponentdan** quriladi va ikkala
build ham toza o'tdi, lekin men ularni chizilgan holda ko'rmadim — buning
uchun control plane'ni lokal ko'tarish kerak edi. Restoran panelidagilar
brauzerda ko'rildi.

---

## 2026-08-08 — zaxira nusxa, eksport, KDS, kampaniyalar, light tema

### Zaxira nusxa (`deploy/keel-backup`, `keel-restore`)
`SAAS.md` dagi **S6** bandi hujjatda turardi, kodda esa yo'q edi: butun repoda
`mongodump` faqat uchta `.md` faylda uchrardi. To'lovchi mijozlar bilan bu
ro'yxatdagi yagona **qaytarib bo'lmaydigan** xavf edi.

Uch qoida skript ichida sababi bilan yozilgan:
- **Bazalar ro'yxati Mongo'dan** olinadi, tenant kolleksiyasidan emas — xato
  bilan o'chirilgan mijoz aynan nusxasi kerak bo'ladigan mijoz.
- **Bitta baza yiqilsa qolganlari davom etadi**, xatolar oxirida yig'iladi.
- **Har arxiv o'qib ko'riladi** (`gzip -t` + `mongorestore --dryRun`) — o'qib
  bo'lmaydigan dump zaxira emas, va buni kerak bo'lgan kuni bilish eng yomon.

`keel-restore` ataylab **jonli baza yoniga** tiklaydi: tiklash so'ralgan payt —
nima buzilganini eng kam bilinadigan payt, va ustiga yozish mijozning yagona
haqiqiy nusxasini yo'q qilishi mumkin.

Konsolda oxirgi nusxaning **yoshi** ko'rsatiladi, diskdagi manifestdan.
"Zaxira yoqilgan" bayrog'i yozilgan kunidan abadiy rost bo'lib turadi va cron
o'chirilganini ko'rsata olmaydi — `attention: "down"` bilan bir dars.

### Ma'lumotni olib ketish (eksport) — konsol ruxsati bilan
Menyusini va mijozlar bazasini ko'chira olmaydigan restoran mahsulot bilan
emas, **chiqish narxi** bilan ushlab turilgan bo'ladi. Lekin arxiv — tizim
ishlab chiqara oladigan eng xavfli fayl, va doimiy tugma birovning qo'liga
tushgan sessiyani jimgina to'liq nusxaga aylantiradi.

Yechim: **muddatli ruxsat** konsoldan (kim, nima uchun, qachongacha), mijozning
**o'z bazasiga** yoziladi — bitta hujjat, bitta yozuvchi, bitta o'quvchi va
tenantdan control plane'ga yangi yo'l ochilmaydi.

Ikki mustaqil qo'riqchi: kolleksiyalar **allowlist**'i ("nima chiqishi mumkin",
"nima chiqmasligi" emas) va maydon nomlari ustidan **naqsh** bo'yicha tozalash.
⚠️ Test haqiqiy bo'shliq topdi: **Payme'ning maydoni shunchaki `key`** deb
ataladi, ya'ni `apiKey`/`secretKey` kabi qo'shma nomlar ro'yxati eng qisqasini
va eng xavflisini o'tkazib yuborardi.

### KDS — oshxona ekrani (`/staff/kitchen`)
⚠️ **"Tayyor" yangi holat emas, `order.readyAt` vaqt belgisi.** Holat kuryer
ilovasi, kuzatuv sahifasi, statistika, POS ko'prigi va uchta lug'at tomonidan
o'qiladi; yangi holat faqat oshxona biladigan faktni ifodalash uchun shularning
hammasiga tegishni talab qilardi. Vaqt belgisi esa qo'shiladi.

Ekran **`staff` tokeni** bilan ishlaydi: peshtaxtadagi planshet umumiy va hech
qachon chiqmaydi. Filial ishchidan olinadi — filtr ichida doim `branchId` bor,
ya'ni `_id` yolg'iz hech qachon hujjat tanlamaydi.

### Segmentlarga xabar yuborish (`/admin/campaigns`)
⚠️ **Narx SMS bo'laklarida ko'rsatiladi.** Kirill va to'g'ri yozilgan o'zbek
harflari (`oʻ`, `gʻ`) GSM-7 dan tashqarida: bitta SMS **70 belgi**, 160 emas,
va hisob har bo'lak uchun. Xushmuomala oxirgi jumla kampaniya narxini ikki
barobar qiladi va ekranda hech nima o'zgarmaydi.

`user.noMarketing` — **qattiq istisno**, ekrandagi filtr emas. Auditoriya har
yuborishda qaytadan hisoblanadi (saqlangan ro'yxat keyingi buyurtmada
yolg'onga aylanadi). Shlyuz sozlanmagan bo'lsa 503 — "240 kishiga yuborildi"
deb hech kimga yetmagan kampaniya eng yomon natija.

### Standart tema — light
Restoran sayti va keel.uz endi `prefers-color-scheme` ni **o'qimaydi**. Sayt —
vitrina: ega aksentni tanlaydi va natijani odamlarga ko'rsatadi, telefoni dark
rejimda bo'lgan mehmon esa tasdiqlanmagan ko'rinishni ko'rardi.

### Qoldirilgani: taom tannarxi
Tannarx POS'dan kelishi kerak (biz POS emasmiz). Lekin API'lar teng bermaydi:
**Poster** mahsulot ro'yxatida `cost` beradi, **iiko** esa `nomenclature` da
tannarxni **umuman bermaydi** — faqat narx (`sizePrices.currentPrice`).
iiko'da tannarx OLAP hisobotlarida, ya'ni boshqa endpoint, va uning maydonlarini
haqiqiy hisob bilan tekshirmasdan yozish — taxmin. Taxmin bilan yozilgan
tannarx eng yomon natija: raqam chiqadi, noto'g'ri bo'ladi va uni hech kim
shubha ostiga olmaydi. Mijozning iiko kabinetidan tekshirilgandan keyin.

---

## 2026-08-10 — Stop list: kassadan avtomatik + paneldagi o'z bo'limi 🛑

**Muammo.** Kassasi bor restoran "lag'mon tugadi" ni **bir marta**, oshxonada
aytadi — sayt esa buni eshitmasdi. Taom buyurtma qilinardi, pul olinardi, keyin
kimdir qo'ng'iroq qilib uzr so'rardi. Panelda ikkinchi marta aytishni so'rash —
rush paytida ikkita stop listni qo'lda ushlab turishni so'rash, ya'ni saytdagi
ro'yxat doim noto'g'ri bo'lardi.

**Backend** (`internal/handlers/posstop.go`, yangi):
- Fon sikli har 3 daqiqada har bir ulangan filialning kassasidan
  `Products()` ni o'qiydi; `Unavailable` mahsulotlar mavjud
  **taom↔mahsulot bog'lashi** orqali bizning taomlarga ko'chiriladi.
- **Ikkinchi ro'yxat**: `branch.posSoldOut` (+ `posSoldOutAt`,
  `posSoldOutError`). Bitta maydonga qo'shilsa ikki yozuvchi bir-birini bekor
  qilardi. `IsSoldOut` ikkalasini so'raydi → sayt, savat, `CreateOrder`,
  combo tekshiruvi **o'zgarmadi**.
- Bo'sh javob = **nosozlik** (`errPOSEmptyCatalogue`), oldingi ro'yxat qoladi.
  Ulanmagan kassa esa oynani tozalaydi.
- Kassadagi stopni paneldan qaytarish **409** (sababi bilan).
- `AdminUpdateBranch` `posSoldOut*` ni yozmaydi (`soldOut` bilan bir tuzoq).
- Yangi endpointlar: `GET /admin/stop-list`, `POST /admin/pos/stop-list/sync`.
- Test: `posstop_test.go` — bog'lanmagan taom hech qachon stopga tushmaydi,
  probel bilan kelgan id baribir mos keladi, bo'sh natija `nil` emas.

**Frontend**: yangi sahifa `/admin/stop-list` (yon panelda "Stop list") —
qidiruv, "faqat sotuvda emas" filtri, bir bosishli stop/qaytarish, kassa
bloki (oxirgi o'qilgan vaqt, xato, "Hozir o'qish"). Kassasiz restoran uchun
ham to'liq ishlaydi. Menyu sahifasidagi tugma joyida qoldi, lekin kassadagi
taom uchun **tugma o'rniga sabab** ko'rsatiladi. Lug'at uch tilda.

---

## 2026-08-10 (2) — Menyu qidiruvi: xatolarga chidaydigan, filtrlar bilan 🔎

**Qaror**: Elasticsearch **emas**. Bitta VPS'dagi bitta restoranga qidiruv
klasteri — butun stackdan ko'p RAM. Kerak bo'lgani — odamlar qanday yozishiga
chidaydigan qidiruv, va u **brauzerda** ishlaydi: menyu allaqachon sahifada,
har harfga so'rov esa mobil internetda ~300 ms.

- `lib/search.ts` — buklash (`fold`) + chegaralangan Levenshtein + skoring.
  Kirill→lotin, oltita apostrof shakli, `x↔h`, `q↔k`, `v↔w`, `ts↔s`, `u↔o`.
  Jonli tekshirildi: `лагман`, `lagʻmon`, `lagmn`, `shurva`, `kaymak`,
  `самса`, `shashlix` — hammasi to'g'ri taomni birinchi qaytardi.
- Har bir so'z mos kelishi shart (AND) — aks holda ko'proq yozish natijani
  kengaytirardi. Sotuvda bo'lmagan taom pastga tushadi, yo'qolmaydi.
- `components/menu/MenuBrowser.tsx` — qidiruv qatori + **filtr ikonkasi**
  (faol filtrlar soni bilan): tartib, narx oralig'i, bo'limlar, teglar,
  "faqat sotuvdagilar / mashhur / chegirmali / to'plamlar". Filtrlar
  menyudan kelib chiqadi — bo'sh boshqaruv ko'rsatilmaydi.
- Standart holat serverda chizilgani bilan bir xil: 48 ta taom HTML ichida
  (SSR HTML'da tekshirildi) — SEO o'zgarmadi.
- Langar (`scroll-mt`) o'lchamlari **chizilgan sahifadan o'lchandi**, padding
  klasslaridan qo'shib hisoblanmadi (oldingi safar aynan shu 44px surilib
  ketgan edi). Desktop: farq 1px.
- Lug'at uch tilda (`t.search`), light/dark tekshirildi.

---

## 2026-08-10 (3) — Qidiruv bosh sahifada ham 🏠🔎

- Yangi band turi **`search`** (`models/design.go` → `BlockSearch`, variantlari
  `bar` / `big`). `Sanitize` `blockVariants` ga qarab tekshirgani uchun backend
  bilmagan tur jimgina tashlanardi — shuning uchun avval o'sha yerga qo'shildi.
- `DEFAULT_SECTIONS` da hero'dan keyin: yarim mehmon nima xohlashini biladi,
  ularning yo'li esa "hero → kategoriyalar → qaysi bo'limda lag'mon bor?" edi.
  Konsol konstruktorida qo'shish/olib tashlash mumkin (palitra + yorliqlar
  uch tilda).
- `components/menu/HomeSearch.tsx` — jonli takliflar (6 ta), oxirgi qator
  `/menu?q=…` ga olib chiqadi. **Qo'shimcha so'rov yo'q**: bosh sahifa
  allaqachon butun menyuni yuklaydi.
- `/menu` endi `?q=` ni **serverda** o'qiydi — natija birinchi bo'yashda
  turadi. Manzil qatori `replaceState` bilan yangilanadi (har harfda
  `router.push` bo'lsa "Orqaga" bitta harf o'chirish bo'lib qolardi).
  Canonical faqat yo'ldan qurilgani uchun `?q=` dublikat sahifa yaratmaydi.
- Jonli tekshirildi: bosh sahifada `лагман` → taklifda `Lag'mon` → Enter →
  `/menu?q=…` da 1 ta natija, telefon va kompyuterda.

---

## 2026-08-10 (4) — Filtr ikonkasi, telefonda 2 ustun, uchta xarita provayderi

**1. Filtr tugmasi holatga qarab ikonkani almashtiradi** (voronka ↔ ×) va
yorlig'i ham. Rangli voronka "filtr bor" deydi-yu, bosish panelni ochadimi
yoki yopadimi — aytmaydi; telefonda esa panel taomlarni ekrandan siqib
chiqaradi, ya'ni savol aynan shu.

**2. Telefonda menyu ikki ustun** (`grid-cols-2`, `sm:` dan yuqorisi
o'zgarmadi). Bitta ustun menyuni ikki barobar uzun qilardi. Kartochka
ixchamlashtirildi (matn o'lchamlari, ichki bo'shliq) va narx qatori
**hech qachon o'ralmaydi** — o'ralganda tugma alohida qatorga tushib, har
kartochkani balandroq qilar va qisqartirish maqsadi yo'qolardi.
360 / 390 / 430 px da tekshirildi, gorizontal scroll yo'q.

**3. Xarita: 2GIS / Yandex / Google — restoran o'zi tanlaydi.**
- `restaurant.mapProvider` + **har provayderga alohida kalit**
  (`mapApiKey` / `mapYandexKey` / `mapGoogleKey`). Bitta umumiy maydon bo'lsa,
  provayderni almashtirib qaytgan ega bir xizmatga ikkinchisining kalitini
  uzatardi — bo'sh xarita va konsoldagi xato.
- Bo'sh `mapProvider` = 2GIS (eski installar uchun).
- **Bitta interfeys, uchta dvigatel**: `lib/map/engine.ts` (shartnoma),
  `twogis.ts` / `yandex.ts` / `google.ts`, `useMapEngine()` (yaratish, xato,
  tozalash). Ilgari to'rtta komponentning har birida "provayderni almashtirish
  uchun faqat shu faylni o'zgartiring" degan izoh bor edi — to'rtta fayl bitta
  fayl emas. `AddressMap`, `ZoneMap`, `LiveMap` endi faqat lat/lng biladi.
- Koordinata tartibi (2GIS `[lng,lat]`, Yandex `[lat,lng]`, Google `{lat,lng}`)
  faqat dvigatelda.
- Jonli tekshirildi (soxta kalit bilan, tarmoq so'rovlari bo'yicha):
  Yandex → `api-maps.yandex.ru/2.1` yuklandi va **haqiqiy xarita + metka
  chizildi**; Google → `maps/api/js` + `map.js`/`marker.js`/`poly.js` (ya'ni
  xarita va qatlamlar qurildi, kalit soxta bo'lgani uchun Google o'z xato
  panelini ko'rsatdi); 2GIS → `mapgl.2gis.com/api/js` + kalit tekshiruvi.
  Sozlamalarda uchala tugma, har biriga mos kalit maydoni va "kalitni qayerdan
  olish" izohi.
- ✅ **Yandex jonli tasdiqlandi** (2026-08-10, `b5somsa.keel.uz`): ega o'z
  kalitini qo'ydi, xarita ishladi va **yetkazish zonasi chizildi** — ya'ni
  dvigateldagi eng nozik ikki joy (koordinata tartibi va polygon ustidan
  bosish) haqiqiy kalit bilan to'g'ri chiqdi.
- ⚠️ **Google hali haqiqiy kalit bilan sinalmagan.** Birinchi shu provayderga
  o'tgan mijozda zona chizish va kuryer metkasi ko'zdan kechirilsin.

---

## 2026-08-11 — zaxira nusxa uch kecha olinmagan ekan 💾

Bandni "o'rnatilganmi?" deb tekshirishdan boshladik va javob **ha** edi: fayllar
joyida, skript qo'lda mukammal ishlaydi, 8-avgustdagi nusxa turibdi. Va aynan
shundan keyingi bironta ham kecha olinmagan.

**Sabab**: `/etc/cron.d/keel-backup` — repoga symlink, nishoni esa
`deploy-keel` egaligida va group-writable (664). Cron `/etc/cron.d` dagi bunday
faylni ishga tushirmaydi. Shikoyati bor, lekin faqat o'z jurnalida:
`(*system*keel-backup) WRONG FILE OWNER` — daqiqada bir marta, uch kun davomida,
va boshqa hech qayerda hech nima o'zgarmaydi.

⚠️ **Bu "ishlamayapti" ning eng yomon shakli**: fayl `/etc/cron.d` da turadi,
`ls` uni ko'rsatadi, skriptni qo'lda ishga tushirsangiz 33 MB nusxa oladi. Ya'ni
har qanday tekshiruv — o'rnatilganmi, skript to'g'rimi, disk bormi — **yashil
javob beradi**. Yolg'on gapiradigan yagona narsa — nusxaning o'zi yo'qligi.
`keel-deploy` dagi "quraman-yu almashtirmayman" bilan bir naqsh: har bir qism
sog'lom, natija esa yo'q.

**Signal aslida bor edi.** Konsol nusxa **yoshini** ko'rsatadi va 36 soatdan
oshganda qizil qatorga aylanadi — ya'ni 9-avgustdan beri konsol buni aytib
turgan. Mexanizm ishladi, unga qaralmadi. Bu — o'sha "bayroq emas, sana"
qoidasining o'zini oqlagan joyi va ayni paytda uning chegarasi: **hech kim
ochmaydigan sahifadagi to'g'ri raqam ogohlantirish emas.** (Keyingi qadam
sifatida ochiq: `attention: "down"` kabi, eskirgan zaxirani konsol bosh
sahifasiga chiqarish.)

**Qilingan ishlar:**
- Cron fayli root egaligidagi **nusxa** bilan almashtirildi (repodagi egalikni
  o'zgartirib bo'lmaydi — u holda `deploy-keel` `git pull` qila olmaydi).
  Cron `RELOAD` qaytardi, shikoyat to'xtadi.
- Bugungi nusxa qo'lda olindi: `/srv/keel/backups/2026-08-11`, 33 MB,
  `failures=0` (4 baza + 3 `uploads`).
- **Tiklash birinchi marta jonli mijozda sinaldi** — `t_b5somsa`, yagona
  haqiqiy tenant (8-avgustda faqat `t_kfc` sinalgan edi). Yoniga tiklandi
  (`--uploads` bilan) va jonli baza bilan solishtirildi: **36 kolleksiya, farq
  0** (48 taom, 7 kategoriya, 7 buyurtma, 101 jurnal yozuvi, 23 tashrif),
  **63 indeksning hammasi** joyida, rasmlar 131 fayl / 17 MB — bir xil. Sinov
  bazasi va katalogi keyin o'chirildi.
- `DEPLOY.md` va `keel-backup.cron` da sabab yozildi: `ln -sf` **noto'g'ri
  buyruq** va uni yozgan har bir kishi shu natijani oladi.

⚠️ **9 va 10-avgust qaytarib bo'lmaydi.** Hech nima yo'qolmadi (hech kim
tiklashni so'ramadi), lekin o'sha ikki kunlik holat endi mavjud emas.

**Ikkinchi yarim: signalni o'qiladigan joyga ko'chirish** (`console/backup-alert`).

Kartochkadagi qator to'g'ri edi va foydasiz bo'ldi — bu ikkisidan **yomonrog'i**,
chunki u panelni kuzatib turgandek ko'rsatadi. Shuning uchun 36 soatdan oshganda
(yoki nusxa umuman yo'q bo'lganda, yoki `failures > 0` bo'lganda) konsol bosh
sahifasida **eng tepada to'ldirilgan qizil banner** chiqadi.

- **Eng tepada, hatto `down` dan ham yuqori** — eng shoshilinch bo'lgani uchun
  emas (qorong'i sayt hozir buyurtma yo'qotayotgan bo'ladi, bu esa yo'q), balki
  bu sahifadagi **yagona qaytarib bo'lmaydigan** nosozlik bo'lgani uchun:
  hisob-faktura bir haftadan keyin ham yozilaveradi, olinmagan kecha esa yo'q.
  Va u o'zini ko'rsatmaydigan yagona nosozlik — qolgan har bir qator platforma
  qimirlaganda qimirlaydi.
- **Faqat nosozlikda ko'rinadi**, pastdagi qatordan farqli (u doim turadi).
  Ikkisining vazifasi boshqa: qator **o'rgatadi** ("nusxalar olinyapti"), banner
  **to'xtatadi** — doim turgan banner ikkinchi haftada hech kimni to'xtatmaydi.
  Bu — `calls` qatoridagi "nol turmaydi" qoidasining o'zi.
- **Chegara serverda** (`sysstat.StaleAfter`), ikkala joy bitta bayroqni
  o'qiydi. Ikki marta yozilgan 36 birinchi tahrirdan keyin o'zi bilan
  kelishmay qoladi — va natijasi qizil banner ostidagi kulrang qator bo'lardi.
- **Bitta so'rov, bitta haqiqat**: `backup` maydoni bosh sahifa allaqachon
  so'raydigan `/stats` javobiga qo'shildi va **o'sha `ReadBackup`** dan keladi.
  Ikkinchi fikr emas — ikkalasi "nusxa olinyaptimi?" degan savolga har xil
  javob bera olmasligi kerak.
- **Banner tekshiriladigan buyruqlarni yozib beradi.** Nosozlikni nomlab,
  keyingi qadamni aytmagan ogohlantirish o'quvchini boshlagan joyida qoldiradi.
- ⚠️ Testda muhrlangani (`backup_test.go`): **vaqt belgisi yo'q manifest
  eskirgan hisoblanadi.** Nol vaqt har qanday arifmetikada "hozirgina" bo'lib
  chiqadi — bu fayl tasodifan bera oladigan yagona eng yomon javob.

---

## 2026-08-11 (2) — konstruktorning ikkinchi bosqichi: uchta jimgina nosozlik 🎨

Reja "kategoriya tanlash UI'sini qo'shish" edi. UI qo'shildi, lekin uni
qo'shishdan oldin **hech qachon ishlamagan uchta narsa** topildi — va uchalasi
ham bir xil shaklda: hech qayerda xato yo'q, javob 200, ekranda hammasi joyida.

**1. Band sozlamalari bazaga umuman yetib bormasdi.**
Konsol panelni schema'dan chizadi va yozgan qiymatlarini `settings` da
yuboradi; control plane'dagi `designSection` struct'ida esa bunday maydon
**yo'q** edi. Go noma'lum JSON maydonini jimgina tashlaydi, ya'ni har bir
sarlavha, har bir "nechta taom", har bir fon — saqlash paytida yo'qolardi.
⚠️ **Jonli mijozda tekshirildi**: `t_b5somsa` dizaynida 8 ta band bor va
**birortasida ham `settings` yo'q**. Panel ishlayotgandek ko'rinardi, chunki
tahrirlagich o'z holatini ushlab turadi — qayta yuklangunicha.
`Blocks` ham xuddi shunday tashlanardi (galereya suratlari shu yerda yashaydi).

**2. `menu-grid` sozlamalarini o'qimasdi.** Birinchisi tuzatilganda ham
ko'rinmasdi: bu band `settings` emas, eski `binding` ni o'qirdi. Endi
**avval `settings`, keyin `binding`**, va **har maydon alohida** — yangi konsol
`categories` ni yozib, `popularOnly` ni eski joyida qoldirgan hujjatda
"hammasi yoki hech nima" o'qish tanlovning yarmini yo'qotardi.

**3. `about`, `gallery`, `cta` — balandligi 0.** Uchalasi ham schema
versiyasiga o'tganda **zaxira zanjirini yo'qotgan**: `cta` sarlavha yozilmaguncha
umuman chizilmasdi, `gallery` qo'lda qo'shilgan suratlarni talab qilardi
(ular esa 1-nosozlik tufayli saqlanmasdi ham), `about` esa faqat "Biz
haqimizda" to'ldirilgan bo'lsa ishlardi.

**4. Va shundan to'rtinchisi topildi: `perks` ham nol edi — u esa standart
maketda.** Ya'ni bu bitta dizayndagi bitta bo'sh band emas: hero ostidagi uch
kartochka **har bir restoran saytidan** yo'qolgan edi, jumladan konstruktorni
umuman ochmaganlarникidan. CLAUDE.md aynan shu holatni nom bilan ogohlantiradi
("blok to'plami faqat yangi g'oyalarni qamrasa, perks chizig'i har bir mavjud
saytdan jimgina tushib qolardi") — va aynan shu bo'lgan. "Mavjud mijozlar hech
qanday o'zgarish ko'rmaydi" — butun xususiyatning sharti, va u buzilgan edi.

⚠️ **Qaysi biri jonli edi — aniqlashtirildi**, chunki birinchi xulosam
noto'g'ri edi. `t_b5somsa` dizayni **qoralama** (`status: draft`, hech qachon
chop etilmagan), demak uning sayti standart shablonni chizadi — ya'ni unda
`perks` yo'qolgan, `about`/`gallery`/`cta` esa umuman ishtirok etmaydi.
Chop etilgan dizayn `t_kfc` da (`navbar, canvas, menu-grid, about, footer`),
va uning `aboutText` i bo'sh — ya'ni **`about` bandi o'sha yerda haqiqatan
bo'sh chiziq bo'lib turibdi**. Xulosa: `perks` hammaga tegadi, qolgan uchtasi
hozircha bitta chop etilgan dizaynga.
Zanjir tiklandi — `cta` lug'atdagi matnga, `gallery` restoranning **taom
rasmlariga** (har tenantda bor yagona surat manbai), `about` restoran nomi va
tavsifiga tushadi. Bu `SchemaBlocks` fayli o'z sarlavhasida e'lon qilgan
qoidaning o'zi; uchta band undan chetda qolgan edi.

**Va nihoyat ko'zlangan ish**: `menu-grid` va `categories` bandlariga
**kategoriya tanlagichi**. Schema'da yangi `categories` turi, tenantning o'z
kategoriyalari dizayn javobi bilan birga keladi (alohida so'rov emas: keyin
kelgan ro'yxat saqlangan tanlovni bo'sh nishonlar bo'lib ko'rsatardi, va
buni odam "ishim yo'qoldi" deb o'qiydi), konsolda esa bosiladigan nishonlar.
- ⚠️ **Massivlar `sanitizeSettings` da umuman tashlanardi** — ya'ni tanlov
  saqlanmasdi. Endi massiv **faqat id ro'yxati** bo'lishi mumkin (24 belgili
  hex, 24 tagacha); id bo'lmagan qiymat **yolg'iz o'zi** tashlanadi, butun
  ro'yxat emas: bo'shatilgan tanlov "hammasi" degani, ya'ni boshqa sahifa, va
  u ataylab qilingandek ko'rinadi.
- **O'chirilgan kategoriya bandni bo'shatmaydi**: mos kelmagan id qoldirilsa
  band torayadi; hech nima qolmasa butun menyuga tushadi. Bo'sh band buzuq
  sayt bo'lib o'qiladi.

**Jonli ma'lumot bilan solishtirildi** (chop etishdan oldin): b5somsa'ning
bazasi va rasmlari lokalga tiklandi, o'sha qoralama dizayn eski va yangi kod
bilan 390 px da chizildi. Eski: 8 banddan **4 tasi nol** (perks, gallery,
about, cta) — sahifa 4393 px. Yangi: 8/8 chizildi — 6147 px. Dizaynsiz
standart shablonda ham xuddi shunday: `perks` 0 → 561 px.
⚠️ **`?preview=` bu farqni ko'rsata olmaydi**: preview jonli saytda boshqa
*dizayn hujjatini* chizadi, kod esa o'sha-o'sha konteynerniki. Shuning uchun
solishtirish lokal — bu deploy'dan oldin ko'rishning yagona yo'li edi.

**Lokal o'lchov** (seed menyu, 7 band, kategoriya tanlangan holda): uch banddan
keyin sahifa 7/7 chizildi, `menu-grid` faqat tanlangan bo'limdagi 3 taomni
ko'rsatdi, plitkalar 2 ta bo'ldi. **360 / 390 / 414 px da gorizontal oqish
yo'q** (`/`, `/menu`, `/cart`, `/bron`, `/about`, `/login`), Telegram
o'lchamidagi viewport'da ham. 32 px dan kichik yagona nishonlar — footerdagi
matn havolalari.

⚠️ **Haqiqiy telefonda va haqiqiy Telegram WebView'ida sinalmagan**: emulyatsiya
qilingan viewport Telegram'ning o'z panellarini, klaviatura ochilganda
o'zgaradigan `viewportStableHeight` ni va iOS Safari'ning odatlarini
ko'rsatmaydi. Buni qo'lda qilish kerak.

⚠️ Kuzatuv (nosozlik emas): `cta` bandi lug'atdagi matnga tushganda
`hours-address` bandidagi buyurtma paneli bilan **bir xil matnni** ikki marta
ko'rsatadi. Ikkalasini bir sahifaga qo'ygan dizayner buni ko'radi va
sarlavhani o'zgartiradi — lekin bilib turgani ma'qul.

---

## 2026-08-11 (3) — to'liq o'chirish, hisobni aldash va resurs adolati 🔐

Uchta so'ralgan ish. Ikkitasida yo'l-yo'lakay jiddiy narsa chiqdi.

**1. Mijozni to'liq o'chirish** (`console/tenant-purge`). Ilgari "o'chirish"
bitta edi va u hech nimani o'chirmasdi. Endi ikkitasi: vaqtincha (avvalgidek,
hammasi qoladi) va **to'liq** — baza, rasmlar, konteyner. Hisob-fakturalar
ataylab qoladi. Batafsil qoidalar CLAUDE.md da.

**2. Bekor qilish orqali pul to'lamaslik — ochiq edi.**
Savol "aldasa nima bo'ladi?" edi; javob: **aldash mumkin edi, va oson.** Kunlik
qator har kecha buyurtmaning hozirgi holatidan qayta quriladi, ya'ni seshanba
yetkazilgan buyurtmani chorshanba "bekor qilindi" deb belgilash yetarli edi.
Bitta bosish, qatorda iz yo'q, mijozning o'z paneli va kuryeri odatdagidek
ishlaydi. Hammasiga shunday qilgan restoran hech nima to'lamasdi.

Endi hisob **`statusHistory` ni** o'qiydi — panel unga faqat qo'shadi — va
`delivered` ga yetgan buyurtma abadiy hisobga kiradi. Halol bekor qilish bu
holatga hech qachon tegmaydi, ya'ni **va'da o'zgarmadi**.
⚠️ Ataylab **detektor emas**: faqat mukofot olib tashlandi. Niyatni
baholaydigan qoida bir kun eshik oldida ovqatdan voz kechgan mehmon uchun
restoranni ayblardi. Ko'rmagani (`reversed`, `cancelledCooked`) yozib boriladi
va mijoz kartochkasida **faqat nolga teng bo'lmaganda** bir jumla bo'lib
chiqadi. Haqiqiy Mongo'da to'rt holatli test bilan muhrlandi.

Isbotlash paytida **ikkita jimgina nosozlik** topildi:
- `localZone()` Go zonani nomlay olmaganda `"Local"` qaytarardi — Mongo bunday
  identifikatorni rad etadi, ya'ni **butun agregatsiya ishlamaydi**: hech qanday
  qator, hech qanday hisob-faktura, hamma tenant uchun. Kun chegarasi siljishi
  emas, umuman yo'qlik. Serverda `TZ=Asia/Tashkent` bo'lgani uchun ko'rinmagan.
- Agregatsiyada **yo'q maydon `null` emas**: `$ne` uni rost deb javob beradi,
  ya'ni `$ifNull` siz **har bir halol bekor qilish** "pishirilgandan keyin bekor
  qilingan" bo'lib sanalardi.

**3. Resurs adolati.** Avval o'lchandi: tenant konteyneri — faqat Go backend,
~9 MB; frontend umumiy pul; Mongo umumiy. Ya'ni "qo'shnini bezovta qilish"
umumiy qatlamlarda.
- Konteynerga: **swap o'chirildi** (standart ikki barobar edi — sizib
  ketayotgan tenant 512 MB **diskni** sekin xotira sifatida ishlatardi, o'sha
  diskda Mongo va hamma zaxira turadi), `PidsLimit`, `CpuShares`,
  `BlkioWeight`.
- **Mongo puli tenantda 20 ga cheklandi** (standart 100 — shift bizniki emas,
  sotilgan mijoz soniga ko'paytiriladi; butun platforma jami ~20 ulanishda
  ishlayapti).
- **`order.createdAt` indeksi qo'shildi**: jonli tenantda tunlik hisob so'rovi
  **COLLSCAN** ekani o'lchandi — eng band restoranning butun tarixi, har kecha,
  umumiy Mongo'da.
- ⚠️ **Yangi cheklovlar faqat qayta yaratilganda qo'llanadi** — mavjud ikki
  konteyner hali eski sozlamada; rollout kerak.
- **"Kam trafikli mijoz resurs sarflaydi" — o'lchov bo'yicha yo'q**: `Memory`
  shift, rezerv emas; bo'sh tenant ~9 MB. Qaytarib olinadigan narsa yo'q,
  ya'ni bu yerda ish qilinmadi va bu ataylab.

---

## 2026-08-11 (4) — ИКПУ va bron xaritasini yashirish 🧾🪑

**ИКПУ** (`PROGRESS` dagi 6-band yopildi). `menu_item.ikpu` — ixtiyoriy maydon,
taom formasida, 17 ta raqam. Ajratgichlar tashlanadi; harf yoki boshqa uzunlik
maydonni **tozalaydi**, chunki yarim yozilgan kod chekka tushmasligi kerak.
⚠️ Kod bo'lmasa ATMOS savatiga **maydonning o'zi yuborilmaydi** — o'rinbosar
emas, umuman yo'q. Testda ikkalasi ham muhrlangan (kodli qatorda `code` bor,
kodsizida JSON'da `"code"` **umuman yo'q**).
⚠️ Buyurtmaga muzlatilmaydi, **menyudan o'qiladi**: nom va narx mijoz rozi
bo'lgan narsa, ИКПУ esa mahsulot haqidagi fakt — buxgalter xatoni tuzatsa hali
to'lanmagan buyurtmalarga ta'sir qilishi kerak.

**Bron xaritasi endi yashirilishi mumkin** (`booking.hidePlan`, sozlamalarda
"Mehmon stolni o'zi tanlasin"). O'chirilsa mehmon faqat vaqt va necha kishiligini
aytadi.
⚠️ **Tanlov yashirildi, hisob-kitob emas**: server baribir haqiqiy stol
ajratadi, ya'ni ikki marta bron qilish imkonsizligicha qoladi va paneldagi
xarita, chekdagi stol raqami — hech nima o'zgarmadi. Stolsiz bron ularning
hammasiga ikkinchi turdagi bronni o'rgatishni talab qilardi.
⚠️ **Mos keladigan eng kichik stol**, birinchi topilgani emas: ikki kishini o'n
kishilik stolga o'tqazish — bir soatdan keyin kelgan katta davrani rad etishning
yo'li, va har bir bron alohida to'g'ri ko'rinadi. Testda muhrlangan.
Bo'sh stol qolmasa **409** — stol band bo'lib chiqqandagi javobning o'zi.
⚠️ Maydon **"hide"**: nol qiymati hozirgi xatti-harakat bo'lishi shart, aks
holda chiqqan kuni hamma restoranda stol tanlash o'chib qolardi. Sozlamalardagi
matn esa teskari ("mehmon o'zi tanlasin", standart holatda belgilangan) — ega
o'qiydigan jumla hozirgi saytini tasvirlashi kerak.

---

## 2026-08-11 (7) — perks: egasining o'z so'zlari, va qidiruv ustidagi kartochkalar 🃏

Ikki nosozlik, bittasi ikkinchisining yonida ko'rindi.

**1. Kartochkalar qidiruv qutisining ustiga chiqib qolgan edi.** Sabab —
`PerksBlock` dagi `-mt-10`: u strip hero'ning **to'g'ridan-to'g'ri** ostida
turgan paytda yozilgan va o'shanda to'g'ri edi. Endi orasida `banners` va
`search` bor.

⚠️ **Va tuzatish "yana o'lchash" emas**: strip hero yonida turadimi-yo'qmi —
**ish vaqtidagi** savol. `banners` bannersiz, `search` menyusiz hech nima
chizmaydi, ya'ni bir xil tartibdagi ikki restoranda ikki xil qo'shni. Qo'shnisi
nima chizishiga bog'liq masofa — hech kim hech nimani o'zgartirmasdan buziladigan
masofa. Pull-up butunlay olib tashlandi, o'rniga oddiy `py-10 sm:py-12`.

**2. Uchta kartochka — birov yozgan va'dalar.** "30–45 daqiqada", "har kuni
bozordan", "naqd yoki karta" — namuna restoran uchun rost, yetkazib bermaydigan
nonvoyxona uchun yolg'on. Ularni o'zgartirishning yagona yo'li konsol edi, ya'ni
**bizdan so'rash**. O'z nomidan aytilgan da'voni tuzata olmaydigan restoran
sahifaning qolganiga ham ishonmay qo'yadi.

Endi `/admin/settings` → "Sayt matnlari" da: ko'rsatish belgisi, ikonka tanlash
(nomlar ro'yxatidan — yozilmaydi) va har kartochkaga uch tilli sarlavha/matn.

⚠️ **Ikki boshqaruv, chunki ikki xohish bor**: bo'sh ro'yxat — "hech qachon
ochmagan" (standart matn chiqadi, ya'ni mavjud har bir restoran uchun hech nima
o'zgarmadi), `hidePerks` — "umuman kerak emas". Oxirgi kartochkani o'chirishni
ikkinchisi deb o'qish chiqqan kuni **hamma saytda** stripni bo'shatardi
(`booking.hidePlan` va bo'sh `mapProvider` bilan bir qoida).

⚠️ **O'chirish belgisi konsol chizgan banddan ham ustun**: kartochkalar
da'vosi — biznes haqidagi bayonot, bezak emas, va uning javobgari restoran.
So'zlarning o'zi esa dizayn bo'lsa dizayndan olinadi — biz pul olgan qism o'sha.

⚠️ Go tomonda `SiteContent` ga slice qo'shilishi bilan `b.Content !=
SiteContent{}` **kompilyatsiya bo'lmay qoldi** (slice solishtirilmaydi) — qoida
`IsEmpty()` ga ko'chdi va testda muhrlandi: metodni yangi maydon qo'shilganda
unutish mumkin, kompilyator esa endi buni aytmaydi.

Tekshirildi: prod build (`next build` + `next start`) da qidiruv va strip
orasida 72 px (telefonda 64 px), bazaga yozilgan ikki kartochka jonli sahifada
chiqdi, `hidePerks` bilan strip yo'qoldi. Sinov ma'lumotlari o'chirildi.

---

## 2026-08-11 (6) — perks tuzatilmagan ekan: o'z tekshiruvim uni o'chirgan 🩹

Deploy'dan keyin `perks` jonli saytda baribir chiqmadi. Sabab kodda emas edi:
eski/yangi suratlarni olish uchun `git checkout ef696b0 -- .../design`, keyin
`git checkout HEAD -- .../design` qilgan edim — va o'sha paytda **HEAD hali
tuzatishni o'z ichiga olmasdi**, ya'ni ikkinchi buyruq commit qilinmagan
tuzatishimni o'chirib yubordi. Keyingi commit (`e8d58cd`) faqat `PROGRESS.md`
ni oldi, xabari esa kod tuzatilgandek yozilgan.

⚠️ **Dars**: `git checkout <ref> -- <yo'l>` vaqtincha solishtirish uchun
ishlatilganda **saqlanmagan ishni yo'q qiladi**, va u hech nima demaydi.
Buning uchun `git stash` yoki alohida worktree kerak edi. Tekshiruv o'zi
tekshirayotgan narsani buzdi.

Yo'l-yo'lakay ikkita noto'g'ri xulosam ham tuzatildi:
- **"Prod build'da ishlamaydi, dev'da ishlaydi"** — yo'q. Eski `npm start`
  jarayoni 3000-portni ushlab turgan, ya'ni yangi build umuman xizmat
  qilmayotgan edi. Bir xil kod ikkala rejimda bir xil ishlaydi.
- **"`search` bandi ham yo'qolgan"** — yo'q, u joyida. Qidiruv qutisi SSR'da
  matn chizmaydi (faqat `input`), men esa matn bo'yicha qidirgan edim.

Endi prod build'da uchala perk kartochkasi ham chizilyapti.

---

## 2026-08-11 (5) — 404 va xatolik sahifalari 🧭

Ikkala ilovada ham yo'q edi: brauzerning o'z 404'i chiqardi.

- **Restoran sayti**: bo'sh likop va yoniga qo'yilgan pichoq-vilka — "bu sahifa
  menyuda yo'q". Xatolik sahifasida qaynab toshgan qozon.
- **keel.uz**: qirg'oqqa o'tirib qolgan kema (brendning o'z shakli); xatolikda
  rozetkadan chiqib ketgan vilka.
- Uch tilda, mehmon o'qiyotgan tilda; `/ru/...` da prefiks saqlanadi.
- **404 va 500 alohida sahifa**: birinchisiga javob — boshqa joyga o'tish,
  ikkinchisiga — kutib qayta urinish. Bitta umumiy sahifa "qayta urinish"
  tugmasi hech qachon ishlamaydi deb o'rgatadi.
- keel.uz xatolik sahifasi avval **holat sahifasiga** yuboradi: xato ko'rgan
  odamning savoli "menda muammomi yoki ularda" — bosh sahifa unga javob emas.
- ⚠️ `global-error` hech nimaga tayanmaydi (provayder ham, tema ham yiqilgan) →
  inline stil va **uch til birdan**. Unda yana bir qator bor: "mijozlar saytlari
  mustaqil ishlaydi" — buni o'qiyotgan odam ko'pincha o'z restorani ham
  o'chganmi deb qo'rqadi.
- Rasm inline SVG: yuklangan fayl 404 ichidagi 404 bo'lishi mumkin edi.

---

## Keyingi katta ish: konstruktor + Telegram mini app 🎨

Reja alohida faylda: **`CONSTRUCTOR.md`** (bandlar, tartib va javob kutayotgan
savollar bilan).

Qisqasi: hozir sayt bitta shablon va restoran undagi 9 ta sozlagichni
o'zgartiradi (rang, shrift, radius…) — maketni emas. Konstruktor maketni ham
ochadi, lekin **kontentni emas**: bloklar ma'lumotga bog'lanadi, erkin matn
maydoni yo'q.

⚠️ Ikki xususiyat bitta faylda, chunki ular bir narsaning ikki yuzi: mini app
saytni Telegram ichida ko'rsatadi, ya'ni konstruktor chizadigan har bir maket
mini app'ning ham maketi. Alohida rejalashtirish konstruktorni desktopga, mini
app'ni telefonga qurishga olib boradi va ikkinchisi birinchisini buzadi.

Qarorlar qabul qilindi (`CONSTRUCTOR.md` 6-bo'lim): panjarali canvas, har
restoranga o'z boti, hozirgi to'lov provayderlari, **dizaynni Keel konsolidan
chizadi** (ega o'zi emas), qulflangan blok sababi bilan ko'rinadi, dizayn puli
tizimga kirmaydi.

⚠️ Eng muhim texnik qaror: **erkin canvas mobil responsive bo'la olmaydi** —
1400 px da sudralgan blokni 380 px ga qayta joylashtirishning matematik yo'li
yo'q. Shuning uchun panjara: maket pikselda emas, **ustunlarda** yozilgani
uchun telefonda o'zi yig'iladi.

Boshlash nuqtasi — **A1** (`page_design` modeli).

---

## Keyingi qadamlar 📋

**0. Zaxira nusxa — ✅ yopildi (11-avgust).** Cron tuzatildi (uch kecha
   olinmagan edi), tiklash jonli mijozda sinaldi. Pastdagi kun yozuviga qarang.
   ⚠️ Qolgan ochiq qaror: nusxalar hali ham **off-site emas** — server
   yo'qolsa ular ham yo'qoladi.

**1. Haqiqiy SMS kalitlari.** Kod tayyor va to'rt provayder ulanadi, lekin
   birinchi mijozning o'z hisobi hali yo'q. Bu kod ishi emas — shartnoma va
   moderatsiya ishi.

   Xavfsizlik tomoni **yopildi**: demo rejim endi kodni qaytarmaydi
   (`SMS_DEMO_EXPOSE_CODE`, standart o'chiq), va shlyuzsiz login umuman
   ishlamaydi. Ya'ni kutish xavfsiz — faqat mijoz kelgunicha SMS sozlangan
   bo'lishi kerak, aks holda uning mijozlari kira olmaydi.

**2. SEO natijasini kutish.** robots/sitemap/JSON-LD jonli va tekshirilgan,
   lekin indekslash kunlar oladi. Tasdiqlash meta tegi sozlamasi **qo'shildi**
   (7-avgust) — endi saytlarni Google Search Console va Yandex Webmaster'ga
   qo'shish qoldi, va bu kutish ishi.

**3. Minimal oylik to'lovni yoqish.** Kod tayyor va tekshirilgan, lekin
   `MIN_MONTHLY` standart holatda **0 — o'chiq**, ataylab: pol haqiqiy
   mijozlar qarzini o'zgartiradi. Summani tanlash — narx qarori, kod ishi emas.
   Yoqishdan oldin: hozirgi mijozlarning davr summalari poldan yuqorimi?
   Poldan past bo'lganini avval ogohlantirmasdan hisobga qo'shish — kelishuvni
   bir tomonlama o'zgartirish.

**5. Qolgan to'rt provayder.**
   - **Anorbank** — hujjat o'qilgan (BM-Merchant API v6.1). ⚠️ To'liq
     to'g'ridan-to'g'ri API: `pan` + `exp` + OTP bizning serverimizga keladi,
     hosted sahifa yo'q. Ya'ni uni qo'shish PCI DSS qarori, kod ishi emas.
     Oqim: login → `unregistered-check` → `hold/v2/otp` → `hold/v2/confirm`,
     summa tiyinda, valyuta ISO 860. Base URL — `ip:port`, ya'ni r_keeper
     kabi tarmoq ichida bo'lishi mumkin.
   - **Kaspi, ePay (Halyk), TipTop Pay** — uchalasi ham Qozog'iston va KZT.
     Ularning oldida tenant darajasida valyuta va davlat kerak; adapterlar
     eng oxirgi qadam. TipTop Pay (ex-CloudPayments KZ) hozirgi callback
     arxitekturasiga eng yaqini.

**6. ИКПУ kodlari.** ATMOS savati fiskal chekka (OFD) ketadi va har qatorda
   ИКПУ kutadi. Bizda taomda bunday maydon yo'q — hozircha yuborilmaydi
   (o'rinbosar emas, umuman yo'q). Haqiqiy fiskal chek kerak bo'lganda
   `menu_item` ga ixtiyoriy `ikpu` maydoni qo'shiladi.

### Ochiq savollar (mijoz uchun)
- Brend/domen qarori: bitta domenda ikki bo'lim (`/restoran`, `/somsa`) yoki
  ikki domen. Hozircha bitta domen + brend cookie'si ishlaydi.
- 2GIS kaliti endi **har restoran o'zinikini** kiritadi (sozlamalarda). Kalitni
  2GIS kabinetida o'z domeniga bog'lash — restoran egasining ishi.

---

## 2026-08-12 — Oldindan buyurtma (predzakaz) 🕒

Mijoz ham, operator ham buyurtmani **keyingi vaqtga** bera oladi, va oshxona
u haqda **o'zi kerak bo'lgan paytda** eshitadi.

### Asosiy qaror: yangi holat ham, fon rejalashtiruvchisi ham yo'q
Oldindan berilgan buyurtma `queuedAt` si **kelajakka** qo'yilgan holda
yoziladi — o'sha maydon allaqachon "bu buyurtma qachondan oshxonaniki"
degani edi, va uni o'qiydigan hamma narsa (KDS, qo'ng'iroq, "kutilayotgan"
sanoqlari) tayyor turgan edi. Ya'ni o'zgarish bitta: kelajakdagi vaqtni
o'tgan deb o'qimaslik (`$lte: now`).
- ⚠️ **Muqobili qimmat**: buyurtmalarni vaqti kelganda ag'daradigan fon sikli
  — ikkinchi yozuvchi, unga qulf kerak, konteyner restartida to'xtaydi, va
  restoran buni **predzakaz umuman pishirilmagan kuni** biladi.
- `order.scheduledAt` — mijozga berilgan **va'da** (chek, kuzatuv sahifasi,
  KDS kartochkasi); `queuedAt` — mexanika. Ikkisi ataylab boshqa maydon.

### Sozlama: `branch.preorder` (filialniki)
`leadMinutes` — **butun xususiyat bir raqamda**: buyurtma shuncha vaqt
qolganda ekranga chiqadi va qo'ng'iroq chalinadi. Egasi qo'yadi, chunki
faqat u biladi (palovga bir soat, kofega o'n daqiqa). Yonida: mijoz uchun
minimal muddat (`minMinutes`), gorizont (`maxDays`), qadam (`slotMinutes`).
- Filialniki, chunki zonalar bilan bir sabab: pishiradigan oshxona qaysi
  ekanini faqat o'zi biladi, va 21:00 da yopiladigan filial ikkinchisining
  kechki slotlarini sotolmaydi.
- `enabled` nol qiymati — **o'chiq**: mavjud har bir install bugun shunday
  ishlaydi (bo'sh `mapProvider` = 2GIS bilan bir qoida).
- `clampPreorder` **saqlashda** ishlaydi, o'qishda emas: 5000 daqiqa yozgan
  ega saqlangan raqamni ko'rishi kerak.

### Ikki qo'ng'iroq, ikki hodisa
`GET /admin/alerts` ga `preorders {upcoming, newestAt, dueAt}` qo'shildi.
- **Kelgani** — yangilik ("go'sht olish kerak"), buyurtma ovozi bilan.
- **Vaqti kelgani** — buyruq ("boshlang"), va u **soatlar keyin, hech kim
  hech nimaga tegmagan holda** keladi. Shuning uchun o'z ovozi bor:
  to'rt nota almashib (1175/880/1175/880) — "yangi buyurtma keldi" bilan
  adashtirib bo'lmaydi, chunki javobi butunlay boshqa.
- ⚠️ `$lte: now` bo'lmasa qo'ng'iroq **teskari** ishlardi: predzakaz berilgan
  zahoti chalinib, kerak bo'lgan paytda jim qolardi.

### Operator chegaralardan ozod (bronlardagi bilan bir qoida)
Telefonda "yigirma daqiqadan keyin" ham, "to'yga" ham normal gap. Operator
`minMinutes`, `maxDays` va ish vaqtidan ozod — lekin **filialning o'z
kalitidan emas** (bu forma validatsiyasi emas, eganing qarori) va o'tgan
vaqtdan ham emas. Testda muhrlangan.

### Tekkan joylar
- Backend: `handlers/preorder.go` (+ test), `orders.go`, `payments.go`
  (kartaga to'langan predzakaz **o'z vaqtida** navbatga tushadi, bank javob
  bergan paytda emas), `kitchen.go`, `adminstats.go`, `admin.go`
  (`?scheduled=1`, vaqt bo'yicha saralash), `brands.go`, `public.go`,
  `repository/migrate.go` (partial indeks — sparse **ishlamaydi**, chunki
  `branchId` hamma hujjatda bor).
- Frontend: `components/site/PreorderPicker.tsx` (slotlar ish vaqtidan
  quriladi — `datetime-local` mijozga yopiq kunni taklif qilardi),
  checkout, kuzatuv sahifasi, `AlertBell`, `/admin/orders` (yangi tab +
  nishonlar), `OperatorOrderModal`, `OrderReceipt`, KDS, sozlamalar, uch til.

### Qo'shimcha: qabul qilinmagan buyurtma to'xtovsiz jiringlaydi
Bir marta chalingan ovoz — eshitilishga bir imkoniyat. Endi `pending` da
buyurtma turgan ekan, qo'ng'iroq har pollda (15 s) takrorlanadi va **"Qabul
qilish" bosilganda o'chadi**.
- ⚠️ **Shartni server beradi** (`alerts.orders.pending`) — u tugma bilan bir
  xil fakt. Ya'ni bir qurilmada qabul qilingani hammasida ovozni to'xtatadi,
  ikkinchi panel jim qilib bo'lmaydigan ikkinchi signal emas, va tab
  yangilanganda hech nima unutilmaydi. Brauzerdagi "ko'rdim" bayrog'i bularning
  hech birini bera olmasdi.
- **Signal bannerida yopish yo'q** (u "nimadir bo'ldi" emas, "nimadir hali
  kutyapti" degan bayonot), o'rniga **"5 daqiqaga jim"**. Yangi buyurtma kelsa
  snooze bekor bo'ladi.
- Birinchi pollda ham chalinadi: ertalab ochilgan panel tunda kelgan uchta
  qabul qilinmagan buyurtma haqida to'rtinchisini kutmasdan aytishi kerak.

### Predzakaz vaqti kelganda ham takrorlanadi
`preorders.dueWaiting` — vaqti kelgan, qabul qilingan, lekin hali hech kim
pishirmayotgan predzakazlar. Nol bo'lmaguncha qo'ng'iroq takrorlanadi;
to'xtatuvchi amal — **"Tayyorlashni boshlash"** (yoki KDS'dagi "Boshlandi").
- ⚠️ **Faqat `confirmed`**: vaqti kelgan-u hali qabul qilinmagan predzakaz
  allaqachon `orders.pending` da bor. Ikki marta sanash bitta buyurtma uchun
  ikki xil ovoz berardi, har biri boshqa tugma so'rab — va bosilgani ikkalasini
  ham o'chirmasdi.
- Banner ikkalasini alohida nomlaydi va qaysi tugma jimlatishini yozadi.
- Signal predzakazniki bo'lsa havola **predzakaz tabiga** olib boradi: buyurtma
  soatlar oldin berilgan, oddiy ro'yxatning tepasida emas.
- Ega qo'rqqan holat (lead 60, oshxona 30 daqiqada boshlaydi → yarim soat
  jiringlash) uchun javob — **"5 daqiqaga jim"**. `lead` ni kichraytirib
  "tuzatish" o'rniga aynan shu tugma bor.

### Ovoz sekundiga bir marta, va darhol to'xtaydi
Ilgari qo'ng'iroq har pollda (15 s) chalinardi — chalinishlar orasi shuncha
bo'lgani uchun u signal emas, "vaqti-vaqti bilan keladigan bildirishnoma"
bo'lib eshitilardi. Endi `ALARM_MS` (1 s) `POLL_MS` (15 s) dan **alohida**:
ikki tezlik ikki savolga javob beradi — serverdan qanchalik tez-tez so'raymiz
va qanchalik qat'iy aytamiz. Ikkalasi kutayotgan bo'lsa ovozlar navbat bilan.
"Qabul qilish" bosilganda sahifa `admin-orders-changed` hodisasini otadi va
qo'ng'iroq darhol qayta so'raydi — aks holda sekundiga bir chalinayotgan ovoz
keyingi pollgacha davom etib, "tugma ishlamadi" bo'lib o'qilardi.

## 2026-08-12 (2) — Mehmonlar fikri saytda ⭐

Sozlamalarda bitta tugma bo'limni **ochadi**, nima ko'rinishini esa
**`feedback.isPublic` bittalab** hal qiladi.

⚠️ **Bu ko'rsatish sozlamasi emas, rozilik chegarasi.** `feedback` dagi har bir
yozuv mehmon o'z kuzatuv sahifasida "buyurtma qanday o'tdi?" savoliga yozgan —
restoranga yo'llangan shaxsiy xabar, ustiga ismi bilan. Tugma bosilishi bilan
hammasini chiqarish internetga hech kim taklif qilmagan so'zlarni, jumladan
jahl bilan yozilganlarini, haqiqiy ismlar ostida joylashtirardi. Bittalab
tanlash — bu xususiyatning tasodifan "maqtovlar devori"ga aylana olmaydigan
yagona ko'rinishi ham.

Qarorlar:
- **"Faqat 4+ yulduz" sozlamasi yo'q**: bunday qoida bo'limni restoran o'zi
  haqida yig'gan maqtovga aylantiradi, va o'quvchi sezgan zahoti ishonmaydi.
- **O'rtacha baho barcha bahodan**, chiqarilganlardan emas — tanlangan
  fikrlarning o'rtachasi hech narsaning o'rtachasi emas.
- Javobda **alohida tor struktura**: modelda telefon, buyurtma raqami va
  shikoyat yuritish bor. Ism — faqat birinchi so'z.
- ⚠️ **Filialsiz yozuvlar ham kiradi**: brendlardan oldingi fikrlarda
  `branchId` yo'q, qat'iy moslik ularni tashlab, egani "tugma yoqilgan, saytda
  hech nima yo'q" holatida qoldirardi.
- Har chiqarish/olib tashlash **jurnalga tushadi** (`feedback.published`):
  "kimning so'zi saytga chiqdi va kim chiqardi" — bir marta, va faqat muammo
  bo'lgandan keyin so'raladigan savol.
- Blok `DEFAULT_SECTIONS` da (menyudan keyin), yoqilmagan bo'lsa hech nima
  chizmaydi. Go'даги `DefaultSections()` ga tegilmadi — u testda "bugungi
  sahifa" deb muhrlangan.

### Filial qamrovi haqida ogohlantirish (jonli mijozdan chiqdi)
`jizbiz.keel.uz` da Yangiyo'lga belgilangan manzil baribir Chilonzor filialiga
ketardi. Kodda xato yo'q: `deliveryBranch` faqat **yetkazish yoqilgan va
manzilni qamrab oladigan** filiallar orasidan eng yaqinini oladi, Yangiyo'l va
Sergelida esa `delivery.enabled = false` edi — ya'ni nomzod ham emas.

⚠️ **Muammo shundaki, buni hech bir ekran aytmasdi.** Sozlamalar sahifasi
to'ldirilgandek ko'rinadi, saqlash muvaffaqiyatli, xato yo'q — natija esa
butunlay boshqa joyda bilinadi. Endi "Yetkazib berish" bo'limi to'rt holatda
ogohlantiradi:
- **yetkazish o'chiq** (faqat bir nechta filial bo'lganda) — filial yetkazish
  buyurtmalarida umuman qatnashmaydi;
- **xaritada nuqta yo'q** — masofa shundan o'lchanadi;
- ⚠️ **`maxKm = 0`** — bu "yetkazmaydi" emas, **"cheklov yo'q"**; bir nechta
  filialda chegarasiz filial har qanday manzil uchun nomzod bo'lib qoladi;
- **`baseFee = perKm = 0`** — yetkazish bepul chiqadi.

Matnlar uch tilda. Ogohlantirish faqat filial tanlangan holatda chiqadi
(tanlanmaganda bo'lim allaqachon `branchGate` bilan yopiq).

## 2026-08-12 (3) — Ko'p filial: tugagan taom va chas pik 🍜

Uch qadam, so'ralgan tartibda.

### 1. Tugagan taom checkout'da aytiladi
`/orders/quote` endi `soldOut` qaytaradi — **hal qilingan filialdagi** tugagan
taomlar nomi bilan. Tekshiruvning o'zi yangi emas (`CreateOrder` doim rad
etardi), lekin u eng oxirida ishlardi: mehmon ism, manzil va to'lov turini
to'ldirib bo'lib "lag'mon tugagan"ni eshitardi. ⚠️ Bir joyda aytilsa —
ma'lumot, ikkinchisida — behuda checkout. Ikkalasi ham bitta `soldOutAt` dan
o'qiydi: savatni ma'qullagan sahifa va uni rad etgan buyurtma ikkalasidan
yomonroq javob.

### 2. Savatni pishira oladigan filial afzal
⚠️ Ilgari tugagan taom filial **tanlangandan keyin** tekshirilardi, ya'ni eng
yaqin oshxonadagi bitta tugagan taom butun buyurtmani rad etardi — uch kilometr
naridagi filialda hammasi bor bo'lsa ham. Endi `bestBranch` savatni to'liq
pishira oladigan eng yaqin filialni oladi.
- Afzallik **ataylab tor**: hech kimning hududi kengaymaydi — har nomzod
  allaqachon o'z `maxKm`/zonasidan o'tgan.
- ⚠️ Hech biri uddasidan chiqmasa **baribir eng yaqini** qaytariladi, `nil`
  emas: o'sha filialning stop listi "lag'mon tugadi" degan halol javobni
  beradi, `nil` esa uni "bu manzilga yetkazilmaydi" degan boshqa va yolg'on
  gapga aylantirardi. Qoida toza funksiyaga ajratilgan va testda muhrlangan
  (aynan shu fallback eng oson yo'qoladigan yarim).
- Combo ichidagi taomlar ham hisobga olinadi (`basketDishes`).
- ⚠️ Yo'l-yo'lakay: `comboMembers` o'zgaruvchisi endi hech qayerda o'qilmay
  qolgan edi — CLAUDE.md dagi o'sha tuzoq (Go xato bermaydi, chunki uni
  to'ldirayotgan sikl "o'qish" hisoblanadi). O'chirildi.

### 3. Chas pik: yuklama ko'rsatiladi, ko'chirish qo'lda
⚠️ **Avtomatik qayta yo'naltirish ataylab qilinmadi.** Band filialdan
boshqasiga o'tkazish ovqatni mehmondan uzoqlashtiradi — oshxonada tejalgan o'n
daqiqa yo'lda yigirma bo'lib qaytadi; tez filial hammaning ishini olib
jazolanadi; bir xil savat besh daqiqa oralatib ikki oshxonaga tushadi. Va
"ko'chirish kerakmi" savoli nechta kuryer yo'lda ekaniga bog'liq, buni chek
sanog'i bilmaydi.

O'rniga: `GET /admin/branches/load` (pishayotgan, qabul qilinmagan, eng
eskisining yoshi) va `PUT /admin/orders/{id}/branch`. Yuklama satri faqat bir
nechta filialda chiziladi. **Eng foydali raqam — eng eskisining yoshi**, chek
soni emas.

Ko'chirish qoidalari: pul o'zgarmaydi, buyurtma raqami eski prefiksda qoladi
(mehmonning kuzatuv havolasi), kuryer bo'shatiladi, `readyAt` tozalanadi,
taomi tugagan filialga ko'chirib bo'lmaydi (409), va har ko'chirish jurnalga
tushadi.

### 4. Menyudagi "tugadi" bayrog'i (o'sha kunning davomi)
Ochiq qolgan joy yopildi. Ilgari bayroq **standart** filialning ro'yxatidan
olinardi — tasodifiy filialdan — va ikki tomonga birdan xato edi: kompaniya
yetkaza oladigan taomni yashirardi **va** mehmonning o'z filialida tugagan
taomni taklif qilardi.

`soldoutlens.go`: filial tanlangan bo'lsa (stol QR'i, olib ketish, sayt
cookie'si) — o'shaniki; bitta filial bo'lsa — o'shaniki; bir nechta filial va
hech biri tanlanmagan bo'lsa — **kesishma**, ya'ni faqat hamma joyda tugagan
taom tugadi deb ko'rsatiladi. Bu qaysi oshxona pishirishidan qat'i nazar rost
qoladigan yagona gap, va uning optimistik tomoni faqat checkout taomni nomi
bilan ushlagani uchun arzon.

Combo tekshiruvi ham shu linzadan o'tadi: `resolveCombo`/`decorateCombos` endi
filial emas, **funksiya** oladi — aks holda taom "bor", ichida o'sha taom
bo'lgan to'plam esa "tugagan" bo'lib chiqardi.

## 2026-08-12 (4) — Xavfsizlik auditi va tuzatishlar 🔐

Loyihaga "hakker nigohi" bilan qaralib topilgan teshiklar, tartib bilan.

**1. Filial qamrovi bitta obyektli amallarda.** Eng katta muammo va tizimli:
`RequireRole` faqat "qaysidir owner/manager" ni isbotlaydi, `clampToAdmin` esa
faqat ro'yxatlarni qisqartiradi. `_id` bo'yicha oladigan handlerlar filtrsiz
edi — Chilonzor menejeri Yunusobod buyurtmasini id yozib o'qish/bekor
qilish/manzil o'zgartirish/kuryer biriktirish, birovning shikoyatini saytga
chiqarish. `scopedOrderFilter`/`courierInScope` qo'shildi; qamrovdan tashqarisi
404. ⚠️ `GET /admin/lookup?phone=` — filialsiz butun kompaniya ma'lumotini
qaytarardi (manzillar, tarix, id'lar); endi `orderScope` bilan qisqartirilgan.

**2. Rate limit** (`middleware/ratelimit.go`): SMS billing hujumi (raqamdan
raqamga yurish) va login brute-force/bcrypt-CPU hujumi. IP bo'yicha, xotira-ichi
(bir tenant = bir konteyner). smsGate 5/daq, authGate 10/daq. Hisob darajasidagi
cooldown'ni almashtirmaydi — u qurbonni, bu serverni himoya qiladi.

**3. JWT_SECRET** standart bo'lsa server ko'tarilmaydi (`config.Validate`):
bashorat qilinadigan imzo = to'liq auth chetlab o'tish. `.env.example` yangilandi.

**4. Buyurtma raqami `crypto/rand` dan**: ochiq kuzatuv sahifasi manzil va
kuryer telefonini beradi, eski raqam esa soatdan + urug'lanmagan `math/rand`
dan edi — taxmin qilib begonaning manzilini o'qish mumkin edi. ~6.5e11 makon.

Hammasi testda muhrlangan (rate limit, JWT validate, order number). Toza
chiqqan joylar: sirlar `json:"-"`, regex `QuoteMeta`, uploads traversal
qo'riqchisi, kuryer faqat o'z buyurtmasi, webhook `ConstantTimeCompare`, XSS
yo'q (React).

**Ma'lum cheklov**: admin tokeni 7 kun, bekor qilish yo'q — ochiq teshik emas,
`tokenVersion` sxemasi keyingi ish.

Oxirida: tuzatishlar `b5somsa.keel.uz` (test restoran) da jonli tekshirildi.

### Keyingi qadam
- Mijozga eslatma (bot xabari "buyurtmangiz bir soatdan keyin") — hozircha
  yo'q, va u SMS emas **Telegram** orqali bo'lishi kerak (pul tejaydi).
- Predzakaz vaqtini paneldan ko'chirish (mijoz qo'ng'iroq qilib so'rasa) —
  hozircha bekor qilib qaytadan yozish kerak.

---

## Ishga tushirish eslatmasi (ertaga davom etganda)

```bash
# Backend (lokal mongo ishlayotgan bo'lsa):
cd backend && cp .env.example .env && go run ./cmd/server
# Admin: seed default admin/admin123, lekin bu deploy’da `yujo` ga o‘zgartirilgan

# Frontend (tugatilgach):
cd frontend && npm install && npm run dev
```

**Keel (ko'p mijozli) uchun**: `DEPLOY.md` → "Hozirgi jonli deployment".

**Eslatmalar / qarorlar:**
- **Tenant ilovasi single-tenant bo'lib qoladi**: u faqat o'z bazasini biladi.
  Ko'p mijozlilik control plane darajasida — aynan shu narsa ma'lumot sizib
  chiqishini imkonsiz qiladi.
- Rasmlar backend `uploads/` papkasida, `/uploads/*` orqali serve.
- Kuryer real-time tracking hozircha yo'q (CLAUDE.md 7-bo'lim).
- Pul birligi UZS, butun son.
- Buyurtma narxi serverda qayta hisoblanadi (client'ga ishonmaymiz).

---

## 2026-08-14 — Marketing va analitika bloki 📊

Branch: `dashboard/analytics-marketing`. Sakkizta bo'lim, tartibi ataylab:
avval o'lchash, keyin o'lchanadigan narsani qilish.

### 1. Hisobotlar — mavjud `Report` shakliga tushdi
Har biri **ekran + Excel birdan**, ya'ni eksport ekrandagi raqamlarning ikkinchi
hisobi emas (`handlers/report.go` dagi qoida).

- **Savdo dinamikasi** (`/admin/reports/sales`, `salesreport.go`) — davr kun /
  hafta / oyga bo'linadi, **oldingi shuncha uzunlikdagi davr bilan taqqoslanadi**
  (o'sish %), eng yaxshi davr, soatlar bo'yicha yuklama.
  ⚠️ Bandlar **local vaqtda** kesiladi, `$dateToString` bilan emas: drayver
  sanani doim UTC beradi, ya'ni Toshkentda soat 19:00 dan keyingi butun kechki
  savdo ertangi kunga tushardi — grafik baribir haqiqiy oyga o'xshab turardi.
  ⚠️ O'rtacha chek **olingan buyurtmalar soniga** bo'linadi: hammasiga bo'linsa
  oshxona bandroq bo'lgan sari o'rtacha chek pasayadi.
- **Kanal analitikasi** (`/admin/reports/channels`, `channelreport.go`) — ikki
  kesim, **hech qachon qo'shilmaydi**: `channel` (sayt / Telegram / operator —
  buyurtma qayerdan berilgan) va `type` (yetkazish / olib ketish / stolda —
  qanday yetkazilgan). Ular kesishadi (botdan olib ketish buyurtma qilish
  mumkin), shuning uchun bitta ro'yxat ustunni jamidan katta qilardi.
  "Yangi mijoz" — **butun tarixdagi birinchi** buyurtmasi shu davrga tushgani;
  aks holda har davrda hamma yangi bo'lib chiqadi.
  ⚠️ Mijoz `userId` bilan, u bo'lmasa telefon bilan sanaladi — yalang'och
  `userId` bo'sh ObjectID tufayli hamma mehmonni bitta mijozga aylantirardi.
- **Kuryerlar** (`/admin/reports/couriers`, `teamreport.go`) — yetkazgan soni,
  daromadi (o'z `payoutMode` i bo'yicha, qayta hisoblanmaydi), tashigan summa,
  naqd. **O'rtacha vaqt "yo'lga chiqdi" → "yetkazildi"**, ya'ni oshxonada
  kutgan vaqt kuryerga yozilmaydi; qayta jo'natilgan buyurtmada **oxirgi**
  yugurish o'lchanadi. Vaqt o'lchanmagan bo'lsa katak **bo'sh**, 0 emas —
  o'rtacha ustunidagi 0 "bir zumda yetkazdi" bo'lib o'qiladi.
  "Qo'lida" — **butun tarix** bo'yicha yig'ilgan minus topshirilgan, davrga
  bog'liq emas: har oy o'zini tozalaydigan qarz qarz emas.
- **Ishchilar** (`/admin/reports/staff`) — `buildDays`/`payForDay` ni qayta
  ishlatadi, ya'ni hisobot va kalendar bir kun haqida bahslasha olmaydi.
  "Hisoblangan" va "To'langan" — **ikki alohida ustun**.

Panel: `/admin/reports` endi to'rt tabli (`Savdo · Menyu · Kanallar · Jamoa`),
davr tanlagichi **tablardan yuqorida** va hammasiga tegishli.

### 2. Sozlanadigan KPI dashboard
`admin_user.dashboard {hidden, order}` — **har admin uchun alohida**: ega
tushumga qaraydi, filial menejeri qabul qilinmaganiga.
⚠️ Ro'yxat **nima o'chiq** ekanini yozadi, nima yoqiq ekanini emas. Bo'sh
qiymat = bugungi dashboard (`hidePlan` bilan bir qoida), va keyingi versiyada
qo'shilgan plitka hammaga o'zi chiqadi — allowlist bo'lganda u har bir mavjud
hisobdan abadiy yashirin qolardi.
Plitkalar reyestri: `lib/dashboardTiles.ts` (panel va sozlash ekrani bitta
ro'yxatdan o'qiydi). Tartib **qisman**: nomlanganlar oldinga, qolgani o'z
joyida.

### 3. RFM segmentatsiya (`handlers/rfm.go`, `/admin/rfm`)
Mavjud 8 ta qoidali segment **qoldi**, RFM ularning **yoniga** qo'shildi.
Sabab: qoida yomon oydan omon qoladi ("60 kun" yanvarda ham, iyulda ham bir xil),
ranking narx o'zgarishidan omon qoladi (30% qimmatlashtirgan restoran bir xona
"ko'p sarflaydigan" mijoz orttirmaydi). "Tug'ilgan kun" va "norozi" RFM'ga
umuman sig'maydi.
- Ballar **kvintil bo'yicha, o'z bazasiga nisbatan** (`vipFloor` bilan bir
  mantiq), 7 xona: `champions / loyal / bigSpender / promising / atRisk /
  needsAttention / lost`. **Bir mijoz — bitta xona** (qoidali segmentlardan
  farqli: u yerda "uxlab qolgan VIP" aynan kerak).
- ⚠️ **Recency teskari**: o'q "oxirgi buyurtmadan beri necha kun", ya'ni kichik
  yaxshi. To'g'ridan-to'g'ri ballansa eng uzoq ketganlar "champions" bo'lib
  ekran tepasiga chiqardi va hamma raqam ishonarli ko'rinardi.
- ⚠️ **Ball qiymat bo'yicha, ro'yxatdagi o'rin bo'yicha emas**: bazaning yarmi
  aynan bir marta buyurtma qilgan, o'rin bo'yicha bo'lish ularni turli
  segmentlarga sochardi.
- ⚠️ **Kampaniya auditoriyasida `rfm:` prefiksi shart** — `lost` ikkala
  ro'yxatda bor va boshqa narsani anglatadi.
- Baza 10 dan kichik bo'lsa RFM umuman yo'q (bo'sh jadval emas — sabab va
  minimal son yoziladi).

### 4. Uchinchi kanal: web push
Kampaniyalar endi **SMS · Telegram · Brauzer bildirishnomasi**.
- `internal/webpush` — RFC 8291 (shifrlash) + RFC 8292 (VAPID), **yangi
  bog'liqliksiz**: `crypto/ecdh` va `crypto/hkdf` stdlib'da (rasm
  kichraytirgichdagi bilan bir savdo). ~200 qator.
- ⚠️ Test **brauzer tomonini yozib deshifrlaydi**. Bu yerdagi har bir xato
  serverdan ko'rinmaydi: tana shifrlanadi, push xizmati 201 qaytaradi, va
  bildirishnoma shunchaki kelmaydi. `key_info` dagi ikki kalit tartibi (mijozniki
  birinchi) va imzoning **xom r‖s** (DER emas) — ikkalasi ham shu shaklda.
- ⚠️ **Kalitlar sozlanmaydi, generatsiya qilinadi** (`push_settings`, birinchi
  ishlatishda) va **hech qachon almashtirilmaydi**: har obuna o'zi yaratilgan
  ochiq kalitga bog'langan.
- ⚠️ `404`/`410` — qayta urinish emas, **o'lgan obuna**: darhol o'chiriladi.
- Obuna `push_subscription`, **endpoint unique** — service worker jimgina qayta
  ro'yxatdan o'tadi, indekssiz mijoz har kampaniyani ikki marta olardi.
- Sayt: `public/push-sw.js` (hech nima keshlamaydi — ommaviy saytda keshlovchi
  worker menyuni eskitardi), `lib/push.ts`, profil sahifasidagi tugma.
  ⚠️ **Ruxsat sahifa ochilganda emas, tugma bosilganda so'raladi**: brauzer uni
  umr bo'yi bir marta so'raydi (geolokatsiya bilan bir dars).

### 5. Upsell va kross-sotuv (`handlers/recommend.go`)
Ikki manba: **birga sotilganlar** (90 kunlik tarix, xotirada 30 daqiqa
keshlanadi) va **ega qo'lda tanlagani** (`menu_item.recommendedIds`).
⚠️ Qo'lda tanlash **shart**: avtomatik yarim **yangi taomni hech qachon**
ko'tara olmaydi — tarixi yo'q, ya'ni faqat allaqachon sotilayotgani tavsiya
qilinardi. Qo'lda tanlanganlar birinchi: u qaror, sanoq esa kuzatuv.
Ko'rinadigan joylar: **taom sahifasi**, **savat**, **checkout** (ikkalasida ham
tugmadan **pastda** — mehmon bir bosishda to'lashga tayyor turganda ustiga
qo'yilgan taklif buyurtmani yo'qotadi) va **call-markaz kartochkasi** (telefon —
upsell haqiqatan ishlaydigan, lekin ekran qo'yib bo'lmaydigan yagona kanal;
narxi bilan, operator menyuni ochmasdan aytishi uchun).
Tugagan taom taklif qilinmaydi (menyudagi bilan bir linza).

### Holat
`go build ./...` toza, `go test ./...` yashil (yangi: `salesreport_test.go`,
`channelreport_test.go`, `teamreport_test.go`, `dashboardprefs_test.go`,
`rfm_test.go`, `recommend_test.go`, `webpush/webpush_test.go`).
`npm run build` toza.

### Keyingi qadam
- Jonli mijozda tekshirish: web push **HTTPS** talab qiladi, ya'ni domen
  ulanmaguncha lokalda `localhost` dan boshqa joyda ishlamaydi.
- `/admin/reports` va `/admin/rfm` haqiqiy ma'lumotda ko'rilmagan — seed bazada
  RFM 10 mijozlik chegaraga yetmaydi.

---

## 2026-08-14 (2) — keel.uz landing: mahsulot bilan moslashtirildi 🛟

Branch: `keel-site/product-facts` (analitika branchidan tarmoqlangan — landing
o'sha ishdagi hisobotlar va marketing haqida ham yozadi, ya'ni ikkalasi birga
merge bo'lishi kerak).

Landing sotuv sahifasi mahsulotdan orqada qolgan edi. Uchta nomuvofiqlik:

- ⚠️ **ATMOS integratsiyalar ro'yxatida yo'q edi** — lekin maxfiylik siyosatida
  bor. Ya'ni huquqiy hujjat mahsulot sahifasidan aniqroq turgan, va to'lov
  tizimini so'ragan restoran "yo'q ekan" degan javob olardi.
- ⚠️ **Telegram butunlay yo'q edi.** Faqat aloqa kanali sifatida ("Telegramda
  yozing"), mahsulot sifatida emas — bot ham, mini app ham. O'zbekistonda bu
  eng ko'p so'raladigan narsalardan biri.
- ⚠️ **Sahifa narx haqida o'zi bilan ziddiyatda edi**: narx blokida pog'onalar
  (1000 / 700 / 500), FAQ'da esa tekis 1 000 so'm. Narxda o'zi bilan
  qarama-qarshi sahifa — ishonchni yo'qotishning eng tez yo'li.

Qilingani:
- **Xususiyatlar 8 dan 12 ga** (uch tilda): Telegram bot va mini app, QR menyu
  va stol bron, oshxona ekrani, hisobotlar qo'shildi; "Mijozlar bazasi"
  marketingni ham qamrab oladigan qilib kengaytirildi (segmentlar, RFM,
  SMS/Telegram/push). To'r `lg:grid-cols-4`, ya'ni 12 — uch tekis qator.
- **Integratsiyalar 6 dan 7 guruhga**: ATMOS to'lovlarga, yangi **Telegram**
  guruhi (Bot API + Mini App) o'z ikonkasi bilan. Ikonka chiziladi,
  yuklanmaydi — bo'lim boshidagi qoida.
- **FAQ**: narx javobi pog'onalar bilan to'g'rilandi; ikkita yangi savol —
  "bot sizniki yoki meniki" va "bir nechta filialga alohida sayt kerakmi".
- **"Hammasi kiradi" ro'yxati**: Telegram bot, oshxona ilovasi va marketing.

Tekshirilgani: `internal/pos` da beshta adapter (iiko/Syrve bitta faylda),
`internal/delivery` da faqat Yandex, `models/payment.go` da to'rtta provayder —
ro'yxatdagi "tez orada" belgilari (Jowi, Paloma, AliPOS, Millennium) hali ham
to'g'ri.

`npm run build` toza.

### Ataylab qo'shilmagani
Ko'p brend / ko'p filial **xususiyat kartochkasi sifatida** qo'shilmadi — u
FAQ'da tushuntirildi. Sabab: bitta filialli restoran (mijozlarning ko'pchiligi)
uchun bu murakkablikni sotuv sahifasining birinchi ekraniga chiqarish mahsulotni
o'zi bo'lganidan og'irroq ko'rsatadi.

---

## 2026-08-15 — Excel panel tilida + "Eng band soatlar" qayta chizildi

### 1. Hisobotlar (ekran ham, Excel ham) panel tilida
Muammo: panel ruschada bo'lsa ham yuklab olingan `.xlsx` **doim o'zbekcha**
edi — sarlavha, ustun nomlari, "Jami" qatori, davr izohi. Ekrandagi izoh ham
shu yerdan keladi, ya'ni ruscha dashboard raqamlarini o'zbekcha tushuntirardi.

- `handlers/reportlang.go` — `reportLang(r)` (`?lang=` → `lang` cookie → uz,
  ya'ni arxiv eksportidagi `exportLang` bilan bir qoida) va `tr{uz,ru,en}`.
  ⚠️ Tarjimalar **ustun ta'rifi yonida** yoziladi, markaziy lug'atda emas:
  uzoqdagi jadvalning buzilish usuli — keyingi qo'shilgan ustun faqat
  o'zbekchada qolishi, va buni hech nima xato deb aytmaydi.
- Yetti hisobot: ABC/XYZ, savdo, kanallar, kuryerlar, ishchilar, moliya, kassa.
  Kanal/tur nomlari, moliya moddalari va `kirim/chiqim` yorliqlari ham.
- ⚠️ **Fayl nomi tildan mustaqil** (`Report.Slug`): `fileSlug` faqat ASCII
  qoldiradi, ya'ni "Отчёт о продажах" **bo'sh** qatorga aylanadi va papkadagi
  har bir hisobot `hisobot.xlsx` bo'lib tushardi.
- Frontend: `panelLang()` cookie'dan o'qiladi va `reportQuery`/`downloadReport`
  ga **avtomatik** qo'shiladi. Har ekranga qoldirilsa, unutilgan chaqiruv jim
  turadi — hech kim buni xato deb yozmaydi, shunchaki raqamlarni qayta teradi.
- Testlar (`reportlang_test.go`): til tanlash tartibi, **har bir ustunning**
  ru'da o'zgarishi (ABC/XYZ dan tashqari), izohlar va fayl nomi.

### 2. `/admin/reports` → "Eng band soatlar"
Muammo (mijoz aytgani): soatlar har xil rangda, ustunlar esa bir xil —
"juda noaniq". Sabab: soatlar `BreakdownChart` bilan chizilardi, ya'ni
**kategorik palitra** bilan — beshta rang aylanib, har soatga ma'nosiz rang
berardi; bundan tashqari ro'yxat gorizontal edi (kun reyting bo'lib o'qilardi)
va **faqat sotuv bo'lgan soatlar** ko'rsatilardi (12:00 va 15:00 yonma-yon
tursa, tushlikdan keyingi tinchlik ko'rinmaydi — aynan o'sha izlanadi).

- Yangi `HoursChart` (`Charts.tsx`): vertikal, soat tartibida, **bitta ohang** —
  eng band soat aksentda, qolgani o'sha rangning washida. Miqdorni faqat
  balandlik anglatadi.
- Diapazon **birinchi savdo soatidan oxirgisigacha, bo'shliqlari bilan**;
  yopiq soatlar chiqarilmaydi (nolga qadalgan chorak grafik hech nima aytmaydi).
- Grafik ustida nima sanalgani bir jumlada, ostida **eng band soat matn bilan**
  ("Eng band soat: 19:00 — 42 ta buyurtma (jamining 18%)") — bu ekrandan
  ovoz chiqarib takrorlanadigan yagona qator.
- Uch tilda (`admin.ts`: `busiestHint`, `peak`).

`npx tsc --noEmit`, `next lint`, `go build ./...`, `go test ./internal/handlers`
— toza.

### Tekshirilmagani
Grafik **jonli ma'lumot bilan ko'z bilan ko'rilmadi** (buyurtmasi bor baza
kerak) — kompilyatsiya va testlar toza, lekin rendering skrinshoti yo'q.

---

## 2026-08-15 (2) — Narx siyosati: har bir pog'onada 20% past 💰

**Qaror**: Zoomda'ning e'lon qilingan har bir pog'onasidan **20% past**
turamiz, katta tarmoqda **40%**. Menyu kiritish bepulligi endi e'lon
qilinadi. Narx bir vaqtda **kodda, saytda va konsolda** o'zgardi — og'izdagi
gap bilan sahifadagi raqam farq qilishi ishonchni yo'qotadigan yagona narsa.

### Pog'onaning shakli o'zgardi (asosiy texnik qism)
`models.PriceForOrders` **partiyali (marjinal)** edi: birinchi 3 000 ta
yuqori narxda, faqat oshgani keyingi bandda. ⚠️ Bunday narvon **matematik
jihatdan** eng past bandga hech qachon yetmaydi — o'rtacha doim oxirgi
tekkan banddan yuqorida turadi. Raqobatchilar esa **butun hajmni** band
narxida sotadi, ya'ni bizning past ko'rsatkichimiz ularning yuqori
ko'rsatkichidan **qimmat** chiqardi (6 000 buyurtmada bizniki o'rtacha 850,
ularniki tekis 700). Raqamni pasaytirish yetarli emas edi — **usul**
o'zgardi.

Yangi qoida — "arzoni qaysi bo'lsa, o'sha": band **to'liq to'lanib erta
sotib olinishi** mumkin.
`min(N×800, max(N,3000)×560, max(N,15000)×400, max(N,50000)×300)`.
Ikki xossa birga saqlanadi: o'rtacha narx **aynan** e'lon qilingan band
narxiga tushadi, va hisob buyurtma o'sganda **hech qachon pasaymaydi**
(har nomzod o'suvchi, minimumi ham — testda muhrlangan).

⚠️ Yon ta'siri ataylab qabul qilindi: band sotib olingandan keyin uning
chegarasigacha qo'shimcha buyurtma bepul ("tekis joy"). Bu shift emas —
tekis joy chegaralangan. Evaziga e'lon qilingan raqam **rost** bo'ladi.

- `PRICE_TIERS=3000:800,15000:560,50000:400,0:300`, baza `PRICE_PER_ORDER=800`
  (`config.go`, `docker-compose.saas.yml`, `.env.saas.example`,
  `control/.env.example`). `MIN_MONTHLY` **0 bo'lib qoladi** — "buyurtmasiz
  oy 0 so'm" raqobatchi ayta olmaydigan yagona qator.
- `tiers_test.go` qayta yozildi. Yangi testlar: o'rtacha aynan band narxiga
  tushishi, hisobning pasaymasligi, va **`TestTwentyPercentUnderTheCompetitor
  AtEveryVolume`** — sahifa aytadigan gapni raqobatchining e'lon qilingan
  narvoniga qarab tekshiradi. Band siljisa, qaysi hajm 20% dan chiqib
  ketgani darhol bilinadi: "har bir hajmda arzonmiz" — yo rost, yo
  chop etmasligimiz kerak bo'lgan da'vo.

### keel.uz (uch tilda)
- Narx bloki: 800 / 560 / 400 / 300, va izoh **shaklni** tushuntiradi
  ("pog'ona butun hajmga tushadi, o'rtachangiz aynan shu raqam").
- Yangi **"Taqqoslash"** bo'limi — Keel / Zoomda / Delever jadvali, sakkiz
  hajm bo'yicha, "Zoomda'dan" ustuni bilan. Ega baribir ikki tabni ochadi;
  o'zi ochishiga qo'yib berish — taqqoslashni halol ramkalash imkonini
  yo'qotish. ⚠️ Shu sababli **`honest` qatori bor**: ularda koll-markaz bor,
  bizda yo'q. Raqibning bitta haqiqiy ustunligini o'zimiz aytish — qolgan
  har bir raqamimizni ishonarli qiladigan eng arzon usul.
- Ikkita yangi karta: **"Menyuni biz kiritamiz — bepul"** (ega hisoblay
  oladigan narx emas, hisoblay olmaydigan **o'tish narxi** — aynan shu
  to'xtatadi, shuning uchun javob raqam bilan: 0 so'm) va **"Tarmoqlar"**
  (50 000 dan yuqori 300 so'm, filial boshiga to'lov yo'q).
- FAQ'ga ikkita savol qo'shildi ("menyuni kim kiritadi", "Zoomda/Delever'dan
  farqingiz"), to'lov savolining javobi yangilandi.
- `StructuredData` narvoni to'rt bandga o'tdi (u qidiruv natijasidagi narx
  qatorini chizadi — sahifadan farq qilsa, butun blok ishonchsiz bo'ladi).
- `compare` jadvalidagi Keel ustuni va hero'dagi ulush yangi narxdan qayta
  hisoblandi (1–2% → 0,5–1%).

### Konsol
Buyurtma narxi maydoni ostida pog'ona izohi. ⚠️ Maydon **tekis narx**, ya'ni
bandning raqamini shu yerga yozish har bir buyurtmani o'sha narxda hisoblaydi
— izoh aynan shu xatoni to'xtatadi. Tarmoq bilan kelishilgan narx bu
maydonda emas, mijozning **o'z pog'onasida** yashaydi.

### Hujjatlar
- `SAAS.md` §5.9 qayta yozildi (shakl, sabab va tuzoq).
- `~/Desktop/keel-marketing-strategiya.txt`: 0 va 0B bo'limlari to'liq qayta
  yozildi, e'tirozlar (Zoomda, koll-markaz, tarmoq), nishon segment va
  yakuniy raqamlar yangilandi; **14-bo'lim** qo'shildi — Instagram uchun 3D
  vizual promptlari, caption va target sozlamasi.

`go build ./...`, `go test ./internal/...`, `npx tsc --noEmit`,
`npm run build` — toza.

### Tekshirilmagani
- Yangi narx **jonli hisob-faktura bilan** ko'rilmadi (testlar toza, lekin
  bazasi bor tenantda oy yopilishi kuzatilmagan).
- Mavjud mijozlarga narx pasaygani haqida hali **xabar berilmagan** — narx
  faqat pasayadi, ya'ni majburiy emas, lekin bu qo'ng'iroq qilish uchun
  bahona (strategiya faylidagi ro'yxatda).
- Eski Instagram prompt/caption fayllarida (`~/Desktop/keel-instagram-*`)
  hali eski narx yozilgan.

---

## 2026-08-15 (3) — keel.uz'da Meta piksel (rozilik bilan) 📈

Reklama kampaniyasi uchun kerak edi: pikselsiz qayta target ham, "necha kishi
saytni **ochdi**" ham o'lchanmaydi — faqat bosilgani ko'rinadi.

- `lib/pixel.ts` — rozilik holati, Meta'ning base kodi (minified qator emas,
  ko'chirib yozilgan — nima ishlashini o'qish uchun minified kodni ochish
  kerak bo'lmasin) va ikkita hodisa. `loadPixel` **idempotent**: React dev
  rejimida effekt ikki marta ishlaydi va ikkinchi `init` kampaniya baholanadigan
  har bir raqamni ikkilantirardi — bu **yaxshi xabarga o'xshaydigan** xato turi.
- `components/MetaPixel.tsx` — rozilikdan keyin yuklaydi, `pathname`
  o'zgarganda `PageView` (Next hujjatni qayta yuklamaydi, ya'ni butun sayt
  bitta ko'rish bo'lib yozilardi), va **bitta delegatsiyalangan click
  listener** bilan Telegram tugmasiga `Lead`. ⚠️ Har tugmaga `onClick` emas:
  ular beshta, va keyingi qo'shilgani jimgina sanalmay qolardi — sanalmay
  qolgan konversiya buzuq ko'rinmaydi, **yomonroq reklama** bo'lib ko'rinadi.
- ⚠️ **`Lead` — aynan Telegram tugmasi**, "narxgacha skroll qildi" emas.
  Meta siz Lead deb bergan narsani qidiradi: arzon harakatni bersangiz, hisob
  o'sha arzon harakatni qiladigan odamlarni sotib oladi.

### Rozilik: cookie oynasi endi haqiqiy tanlov
Oynaning o'z izohida yozilgan edi: bu sahifada kuzatuv yo'q, shuning uchun
"rad etish" tugmasi "roziman" bilan **aynan bir xil kodni** ishga tushirardi,
va hech nima o'zgartirmaydigan tugma odamlarni bu oynalarni bezak deb
o'qishga o'rgatadi. Piksel bu faktni o'zgartirdi — demak oyna ham o'zgardi:
- `ads` yoqilganda **ikki tugma**, va rad etish rostdan ham hech nima
  yuklanmasligini bildiradi (Playwright bilan tekshirildi: rozilikdan oldin
  ham, rad etishdan keyin ham `facebook.com` ga **nol so'rov**).
- Piksel yo'q bo'lsa — eski bir tugmali oyna va eski matn, chunki o'shanda u
  rost.
- ⚠️ **Yangi kalit** (`keel_ads_consent_v1`), eski `cookie_notice_v1` emas:
  eski oyna "o'chiradigan narsa yo'q" degan edi, va o'sha jumlaga berilgan
  javob **bu** savolga javob emas. Kalitni qayta ishlatish "o'qidim" ni
  "roziman" ga aylantirardi — orqaga qarab, va aynan eski matnga ishongan
  odamlar uchun.
- Rad etish tugmasi **birinchi va teng og'irlikda**: burchakdagi xira havolaga
  yashirilgan rad etish — bu oynalarni hech kim o'qimay qo'yishining sababi.

### Sozlama va hujjat
- `META_PIXEL_ID` — **render vaqtida** o'qiladi, `NEXT_PUBLIC_` emas
  (§"Next.js rewrites build vaqtida muhrlanadi" bilan bir tuzoq: bundle'ga
  kirgan qiymatni konteynerda qo'yish hech nima qilmaydi, piksel esa jimgina
  yo'q bo'lardi — bu "hech kim bosmagan kampaniya"ga o'xshaydi).
  `docker-compose.saas.yml` + `.env.saas.example`. Bo'sh = piksel yo'q.
- ⚠️ **Faqat keel.uz.** Tenant saytlariga reklama skripti berilmaydi:
  restorandan ovqat buyurtma qilgan mehmon bizning reklama hisobimiz
  tomonidan o'lchanishga rozilik bermagan, restoran ham mijozlarini bizga
  topshirmagan.
- Maxfiylik siyosati uch tilda yangilandi (8-bo'lim endi ikki xatboshi:
  tenant saytlari — kuzatuvsiz; keel.uz — rozilik bilan piksel), sana
  `2026-08-15` ga ko'chdi.

Tekshirildi (Playwright, real brauzer): rozilikdan oldin nol so'rov · rad
etishdan keyin nol so'rov · roziman → `fbevents.js` + `ev=PageView` ·
Telegram tugmasi → `ev=Lead` · qayta yuklashda oyna qaytmaydi.
`npx tsc --noEmit`, `npm run build` — toza.

### Tekshirilmagani
Haqiqiy piksel ID bilan Events Manager'da ko'rilmadi (test ID ishlatildi) —
ID qo'yilgach Meta Pixel Helper bilan bir marta tasdiqlash kerak.

---

## 2026-08-15 (4) — Piksel jonli, va taqqoslashdagi xato tuzatildi 📌

### Taqqoslash bo'limi bizda bor xususiyatni inkor qilardi
`rivals.honest` "ularda koll-markaz bor, bizda yo'q" deb tugardi. Yarmi rost,
yarmi esa **shu repoda allaqachon bor mahsulot**: `/admin/calls` kiruvchi
qo'ng'iroqda mijoz kartochkasini o'zi ochadi (oxirgi buyurtmasi, manzillari,
odati, javobsiz shikoyati), operator buyurtmani sayt bilan **bir xil
narxlash quvuri** orqali qabul qiladi, har qo'ng'iroq tahrirlanadigan
jurnalga tushadi, onlinePBX hammasini avtomatik to'ldiradi. Va bu narxning
ichida.

⚠️ Xato turi muhim: sahifa **xususiyat ro'yxatini solishtirayotgan** o'quvchiga
o'zida bor narsani yo'q deb aytdi — va buni aynan **ishonish uchun** yozilgan
xatboshida qildi. Haqiqiy farq **odamda**: ularning tarifiga telefonni
ko'taradigan operator ham kiradi.

Endi xatboshi avval nima borligini aytadi, keyin o'sha bitta farqni nomlaydi —
va bu foydaliroq gap ham, chunki kichik restoran telefonni baribir o'zi
ko'taradi, ya'ni "tan olish" u yerda hech nimani tan olmagan bo'lardi.
Koll-markaz `included` ro'yxatiga va modul taqqoslashiga ham qo'shildi
(Delever'da har biri alohida qator). Uch tilda. Strategiya faylidagi
e'tiroz javobi ham qayta yozildi.

### Meta piksel jonli
- `META_PIXEL_ID=1480126137255279`, `META_DOMAIN_VERIFICATION=...` prod
  `.env` ga qo'shildi, `site` konteyneri `--force-recreate --no-deps` bilan
  almashtirildi (build kerak emas — qiymatlar render vaqtida o'qiladi).
- **keel.uz Meta'da tasdiqlandi** (`Verified`) — meta-teg orqali.
- Dataset "Keel.uz marketing" Keel.uz biznes-portfoliosida; Facebook sahifasi
  portfolioga ulandi.
- ⚠️ **Piksel yangi datasetga Meta tomonidan avtomatik ulangan CAPI Gateway
  (Datahash, `capig.datah04.com`) bilan keladi.** U begona emas, lekin
  28 kunda "aktivatsiya" talab qiladi. Tasdiqlanmasa o'zi o'chadi — va
  o'chgani ma'qul: brauzer pikseli yetarli, maxfiylik siyosatida esa faqat
  Meta yozilgan.

### Tekshiruv (jonli keel.uz, foydalanuvchi brauzeri)
`PageView` → `GET facebook.com/tr` 200, `id=1480126137255279` ✓
`Lead` (Telegram tugmasi) → `POST facebook.com/tr` 200 ✓
Rozilikdan oldin — nol so'rov ✓

⚠️ **Tuzoq (yozib qo'yildi): birinchi hodisa GET, keyingilari POST beacon.**
Avtomatlashtirilgan testda `ev=Lead` ni **URL bo'yicha** qidirish uni topa
olmaydi — tanasi so'rov tanasida. Playwright bilan qilingan birinchi
tekshiruv aynan shu sababdan "hodisa ketmayapti" degan noto'g'ri xulosa
berdi.

### Tekshirilmagani / qolgani
- **Reklama akkaunti Keel.uz portfoliosida yo'q.** Mavjud akkaunt
  (`1410283180046276`) shaxsiy va FilmoraUz uchun ishlatiladi; portfolioga
  ko'chirish **qaytarib bo'lmaydi**, shuning uchun qaror egasiniki.
- **Instagram ulanmadi**: Meta bu qadamda Instagram'ga **kirishni** talab
  qiladi (parol) va ulangan reklama akkauntini ham qaytarib bo'lmaydigan
  tarzda portfolioga ko'chirishi mumkin.
- `Автоматически передавать информацию о страницах и товарах` **yoqiq**
  (Meta standarti): hodisa bilan birga sahifa sarlavhasi, tavsifi va narx
  pog'onalari ham ketyapti (`ap[contents]`). Kerak bo'lmasa o'chiriladi.

---

## 2026-08-17 — O'z kassamiz (POS): poydevor + kassa ekrani ✅

Bozordagi POS'lar eski, qiyin va qimmat degan qarordan keyin Keel'ga **o'z
kassasi** qo'shila boshlandi. Narx modeli — Keel'dagi kabi har buyurtmadan,
lekin **boshqa raqamda va shift bilan** (pastda "Ochiq savol").

### Asosiy qaror: ochiq chek — bu `order`, ikkinchi hujjat turi emas
`models/check.go`. Sabab: buyurtmadan keyingi hamma narsa allaqachon bor va
ishlaydi — KDS, chek, statistika, ABC/XYZ, moliyaviy hisobot, kassa smenasi,
chegirma, loyalty, birovning kassasiga ko'prik. Ikkinchi kolleksiya bularning
**har biriga** ikkinchi xil sotuvni o'rgatishni talab qilardi, va birinchi
unutilgani restoranning o'z tushumini jimgina kam ko'rsatardi.

Farqi bitta: chek **bir soat davomida yig'iladi**, tayyor holda kelmaydi. Buni
ifodalaydigan timestamp allaqachon bor edi — `queuedAt` ("bu qachondan
oshxonaniki"). Ochiq chekda u yo'q, demak pass ko'rmaydi, qo'ng'iroq
chalinmaydi va tushum sanalmaydi.

Uch holat, va hech biri yangi `status` emas:
`check != nil && closedAt == nil` → zalda ochiq; `closedAt != nil` → to'landi;
`check == nil` → kassa sotuvi emas.

**Natijasi o'lchandi**: `cash.go` ga **bir qator ham tegilmadi** — smena
allaqachon `dinein` + naqd + `paid` + `paidAt` ni sanaydi, ya'ni naqdga
yopilgan chek kassa qoldig'iga o'zi tushdi.

### Qatorlar: `firedAt` — chek darajasida emas, **qator darajasida**
`OrderItem.FiredAt`. Zalda taomlar bosqichma-bosqich yuboriladi: salat hozir,
asosiysi yigirma daqiqadan keyin. Chek darajasidagi bayroq ofitsiantni yo
butun kechki ovqatni birdan yuborishga, yo ikkinchi chek ochishga majburlardi —
haqiqiy zal ikkalasini ham qilmaydi.
- KDS faqat **yuborilgan** qatorlarni ko'radi (`kitchenItems`). Saytdan kelgan
  buyurtmada hech bir qatorda `firedAt` yo'q, demak hammasi ko'rinadi — eski
  xatti-harakat o'zgarmadi.
- ⚠️ **Yuborish `readyAt` ni tozalaydi.** Ikkinchi taom pass'ga oshpaz
  birinchisini "tayyor" degandan ancha keyin keladi, va `readyAt` turgan chek
  KDS filtridan tushib qoladi — ya'ni asosiy taom buyurtma qilinar, puli
  olinar va **hech kimga ko'rsatilmasdi**.

### Ikki ruxsat, chegara pul turgan joyda
`staff.canWaiter` / `staff.canCashier` (`Staff.Can`, kassir ofitsiantni o'z
ichiga oladi). Bittaga birlashtirish likop ko'tara oladigan har kimga chekni
"chegirma 100%" bilan yopish imkonini berardi.
- ⚠️ Nol qiymat **false**, va `canKitchen` dan farqli **migratsiya kerak
  emas**: bu xususiyatdan oldin hech kim kassada ishlay olmasdi, ya'ni
  saqlanadigan xatti-harakat yo'q. `canKitchen` aynan **mavjud** ruxsatni
  olib tashlagani uchun backfill talab qilgan edi.
- Rad javobi ikki xil matn: "kassa amallariga ruxsat yo'q — kassirni chaqiring"
  ofitsiantni **kassirga** yuboradi, "zal ekraniga ruxsat yo'q" esa
  administratorga. Noto'g'ri matn — kassir parolini so'rashning yo'li.

### Olib tashlash: bitta tugma, ikki amal
- **Yuborilmagan qator** — imlo xatosi, izsiz o'chadi. Ofitsiantdan o'z
  terish xatosini oqlashni talab qilish uni "." yozishga o'rgatadi va haqiqiy
  voidlardagi sabablarni qadrsizlantiradi.
- **Yuborilgan qator** — mavjud ovqat. Qator hujjatda **qoladi** (kim, qachon,
  nega, tashlab yuborildimi) va faqat **kassir** qila oladi. Izsiz void —
  restorandan pul olib chiqishning eng eski yo'li.
- Qisman void qatorni ikkiga bo'ladi: qolgani tirik, olingani alohida qator.

### Bitta narx quvuri (nusxa emas)
`handlers/orderline.go` — `menuLine`/`menuLines` `composeOrder` dan **ajratib
olindi**. Endi sayt, operator va kassa bir savatni bir xil narxlaydi. Nusxa
jimgina ajrab ketardi va alomati eng yomoni bo'lardi: bir xil savat telefonda
boshqa, stolda boshqa narx, ikkalasi ham o'z ekranida to'g'ri ko'rinadi.

### Yozilgan fayllar
Backend: `models/check.go`, `handlers/till.go`, `handlers/tilllines.go`,
`handlers/tillclose.go`, `handlers/orderline.go`, `handlers/till_test.go`;
`models/models.go` (`Order.Check`, `OrderItem.LineID/FiredAt/Void`,
`LiveItems`), `models/staff.go` (`Can`), `handlers/kitchen.go`
(`kitchenItems`), `handlers/adminstaff.go`, `router.go`, `repository/migrate.go`
(ikkita indeks).

Frontend: `app/kassa/{layout,page,CheckPanel,PayDialog,VoidDialog,NewCheckDialog}.tsx`,
`lib/types.ts` (`Check`, `CheckLine`, `CheckLineVoid`), `lib/api.ts` (`till*`),
`lib/i18n/admin.ts` (`till` bo'limi — uz/ru/en), `app/admin/staff/page.tsx`
(ruxsat belgilari), `app/staff/login` (`?next=`).

Kassa ekrani uch ustun: **qaysi stol** → **nima xohlaydi** → **qancha qarz**.
Bu ishning tartibi; jamini o'rtaga qo'ygan har qanday joylashuv oxirgi qadamni
qidiruvga aylantiradi.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` hammasi yashil ✓ ·
`tsc --noEmit` ✓ · `npm run build` ✓ (`/kassa` marshrut sifatida chiqdi).
Yangi testlar: ruxsat chegarasi, void jamiga ta'sir qilmasligi, KDS
yuborilmagan qatorni yashirishi, saytdan kelgan buyurtma o'zgarmagani.

### Keyingi qadam
1. **Ofitsiant ekrani** (`/zal`) — kassa ekranidan keyingi ish. Backend tayyor:
   ayni endpointlar, `?mine=1` filtri va `canWaiter` ruxsati bor.
2. Chek chop etish (ESC/POS), naqd yashigi.
3. Kassa smenasini kassa ekranidan ochish/yopish (hozir faqat paneldan).
4. Chekni bo'lish (`splitFromId` modelda bor, handler yo'q).
5. Oflayn rejim — eng qiyini va **sotuvning sharti**.

### ⚠️ Ochiq savol (kod emas, biznes)
**Fiskal chek (ККМ/ОФД) provayderi va uning har chekdan oladigan narxi.**
Bu narx bizning har chekdan olinadigan summamiz ustiga qo'shiladi, ya'ni butun
narx modelini belgilaydi. Kod yozishdan oldin aniqlanishi kerak edi va hali
aniqlanmagan. Tavsiya: **o'zimiz sertifikatlanmaymiz, ro'yxatdan o'tgan
virtual kassa provayderiga ulanamiz** — to'lov provayderlariga ulangandagi
bilan bir naqsh.

---

## 2026-08-17 — Fiskal chek: poydevor va olti provayder ✅

Kassa ekrani pul oladi, lekin **cheki yo'q edi**. Zaldagi to'lov soliq
qo'mitasida ro'yxatdan o'tmasa, restoran bizning ekranimiz **yonida** eski
kassasini saqlab qoladi va har chekni ikki marta uradi — ya'ni biz POS'ni
almashtirmadik, ustiga ikkinchi ish qo'shdik.

### Nima aniqlandi (qidiruv natijasi)
- ⚠️ **Chek tanasi standart, provayderniki emas**: `Name`, `SPIC` (=ИКПУ),
  `PackageCode`, `Price`, `Amount`, `VAT`, `VATPercent`, `Discount`, `Units`,
  + `ReceivedCash`/`ReceivedCard`. **Pul tiyinda.** Demak provayderlar orasidagi
  farq — avtorizatsiya va transport, tana emas (iiko/Syrve bilan bir holat).
- **Narx faraziyasi noto'g'ri edi**: Multikassa/Rahmat — **123 600 so'm/oy**,
  har chekdan emas. Ya'ni fiskal narxi bizning har-chekdan olinadigan
  summamizga qo'shilmaydi; u restoranning alohida qat'iy xarajati.
- ⚠️ **Birortasida ham ochiq API hujjati yo'q** — hammasi shartnomadan keyin
  (onlinePBX bilan bir devor).

### Shu sabab: provayderdan mustaqil hamma narsa yozildi, adapterlar yo'q
`fiscal.New` har olti provayder uchun **`ErrNoAdapter`** qaytaradi. Endpoint
o'ylab topilsa — kompilyatsiya bo'ladigan, review'dan o'tadigan va **bitta ham
chek yubormaydigan** kod bo'lardi.
- ⚠️ **Ro'yxat baribir to'liq ko'rsatiladi** (`Providers()`, `ready: false`):
  o'z provayderini topmagan ega "bizni qo'llab-quvvatlamaydi" deb xulosa qiladi,
  "qo'llab-quvvatlaymiz, hujjat kutilmoqda" esa sotuvchi ayta oladigan javob.
- Rad javobi **aybni bizga yozadi** ("API hujjati kutilmoqda"), aks holda ega
  hech qachon xato bo'lmagan parolni qayta terib chiqadi.

### Chek quruvchi (`internal/fiscal/receipt.go`) — uchta jimgina xato
1. ⚠️ **QQS narx ichida, ustiga qo'shilmaydi**: `price*rate/(100+rate)`.
   Ko'zga ko'rinadigan formula (`*rate/100`) soliqni ~12% ga oshiradi va
   chiqqan raqamlar butunlay ishonarli ko'rinadi.
2. ⚠️ **Buyurtma darajasidagi chegirma qatorlarga aniq bo'linishi shart**
   (`distribute`, eng katta qoldiq usuli). Har qatorni alohida yaxlitlash bir
   necha tiyinni yo'qotadi, va qatorlari olingan pulga teng kelmaydigan chek —
   rad etilgan (yoki qabul qilingan-u noto'g'ri) hujjat.
3. ⚠️ **To'lov taqsimoti qatorlarga moslanadi, teskarisi emas**: qatorlar
   tovarni tasvirlaydi va menyu bilan tekshiriladi; naqd/karta esa ma'lum
   jamiga qo'shilishi kerak bo'lgan ikki son. Teskarisi — kassa balansi uchun
   nima sotilganini tahrirlash bo'lardi.

### Menyudagi uch yangi maydon
`packageCode`, `vatPercent`, `unitCode` (+ `normalizeFiscal` — bitta funksiya,
ikkala call site uchun).
- ⚠️ **`vatPercent` — pointer, chunki 0 haqiqiy javob.** "QQS'siz" va
  "to'ldirilmagan" bir xil chek beradi-yu teskari narsani anglatadi. Bu —
  kodning odatdagi "bo'sh qiymat = bugungi xatti-harakat" qoidasi
  **ishlamaydigan** yagona joyi: soliq stavkasining "bugungi"si yo'q. Shuning
  uchun `FiscalSettings.VatPercent` ham pointer va **yoqish uchun majburiy**.
- ⚠️ **`packageCode` ИКПУ'ga tegishli**, taomga emas — ИКПУ tozalansa u ham
  tozalanadi, aks holda menyuda hech nimaga ishora qilmaydigan, lekin
  to'ldirilgandek ko'rinadigan raqam qoladi.
- **`unitCode` nol qiymati "dona"** — porsiya aynan shu, ya'ni mavjud har bir
  taom allaqachon to'g'ri (bo'sh `mapProvider` = 2GIS bilan bir qoida).

### Sozlamalar
`fiscal_settings`, **filial darajasida** (`branchId` unique — `pos_settings`
bilan bir sabab: kassa joyga ro'yxatdan o'tadi). Har provayderga **alohida
tortma**; sirlar qaytarilmaydi, **bo'sh sir = saqlangani qolsin**.
- **Saqlash va yoqish — ikki xil amal**: forma kunlar davomida to'ldiriladi,
  yoqish esa "sotuvlar ro'yxatdan o'tyapti" degan da'vo, va yolg'on da'voni
  inspektor topadi. `fiscalEnableRefusal` — sof funksiya, testda muhrlangan.
- **Sahifadagi eng foydali qator — `lastReceiptAt`**, tekshiruv bayrog'i emas
  (`lastEventAt`/`lastUpdateAt` bilan bir qoida).

### Yozilgan fayllar
Backend: `internal/fiscal/{fiscal,receipt,receipt_test}.go`,
`models/fiscal.go`, `handlers/{fiscal,fiscal_test}.go`, `models/models.go`
(uch maydon), `handlers/admin.go` (`normalizeFiscal`, `codeDigits`),
`repository/{store,migrate}.go`, `router.go` (4 marshrut).

Frontend: `components/admin/FiscalEditor.tsx`, `app/admin/settings/page.tsx`,
`app/admin/menu/page.tsx` (uch maydon), `lib/types.ts`, `lib/api.ts`,
`lib/i18n/admin.ts` (`fiscal` + menyu kalitlari, uz/ru/en).

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` hammasi yashil ✓ ·
`tsc --noEmit` ✓ · `npm run build` ✓

### Keyingi qadam
1. **Reestrni qo'lda ochib** ro'yxatni tasdiqlash (sahifa JS bilan chiziladi).
2. 2–3 provayderdan API hujjati va tarif; **eng muhim savol** — o'z nomimizdan
   ulay olamizmi (hamkor) yoki har restoran o'zi shartnoma tuzadimi. Bu
   ulash oqimini belgilaydi.
3. Birinchi adapter → `fiscal.New` da bitta qator, `ready: true`.
4. Chekni kassa yopilganda yuborish (`tillclose.go` ga ulanish) + qayta urinish.

---

## 2026-08-17 — Multikassa: birinchi haqiqiy adapter ✅

Rahmat POS (Multikassa) hujjatlari qo'lga tegdi: integratorlar uchun PDF
(2026-04) va ikkita Postman kolleksiyasi. Postman sahifalari JS bilan
chiziladi, shuning uchun `documenter.gw.postman.com/api/collections/...`
orqali JSON holida olindi.

### ⚠️ Eng muhim topilma: bu ikki xil API, va fiskallashtiruvchisi lokal
- **Multikassa.Pos** — chekni fiskallashtiradigan API, va u restoran ichidagi
  kassa kompyuterida turgan dastur: `http://192.168.14.65:9090/api/v1/operations`.
  **Hech qanday avtorizatsiya yo'q** — na token, na parol. Bu kamchilik emas,
  xavfsizlik modeli: manzil bino tashqarisidan yo'naltirilmaydi, va aynan shu
  sabab **bizning konteynerimiz ham unga yeta olmaydi**. `r_keeper` devorining
  o'zi.
- **Multibank.Касса** — `api.multibank.uz`, Bearer token. Bu **o'qish va
  kabinet** API'si (cheklar, statistika, kassalar, kassirlar, nomenklatura,
  billing). **Sotuvni fiskallashtira olmaydi.** Undan keraklisi bittasi:
  `POST /api/fiscal/tsc/edit_user_modules` → `integration_mode: true`.

### Shu sabab: ikkita transport (`Info.Local`)
- **server-dialled** — biz `Client` tutamiz va o'zimiz chaqiramiz (odatdagi).
- **local** — biz so'rovni **quramiz**, restoranda turgan kimdir **yuboradi**.
  U kimdir — **kassa ekrani**: kassirning planshetи allaqachon kassa bilan bir
  tarmoqda, va bu butun tizimda unga yeta oladigan yagona mashina.

⚠️ **Bo'linish faqat transportda.** Chek tanasi baribir serverda, buyurtmadan
quriladi — brauzerda hech qachon yig'ilmaydi. Brauzerning ishi bitta tarmoq
sakrashi: unga **shaffof blob** beriladi va javob o'sha holida qaytariladi.
Narxlash quvurining qoidasi, og'irroq hujjatga qo'llangan. Kassa natija haqida
yolg'on gapira oladi — bu qabul qilingan: u faqat **o'z restoranining** cheklari
haqida yolg'on gapira oladi, ya'ni soliq qo'mitasining o'z yozuvi rad etadigan
o'zini-o'zi aldash. Himoya qilishga arziydigan chiziq — chekning **ichi**.

### ⚠️ Ikki hujjat zid, va biri 100× farq qiladi
| Nima | PDF (2026-04) | Postman | Qaror |
|---|---|---|---|
| Pul | `receipt_sum`/`received*` **tiyin**, `items[]` **so'm** | ikkalasi bir birlikda | **PDF** |
| ИКПУ | `ikpu` + `packageCode` | `classifier_class_code` + `product_package` | **ikkala nom bilan** |
| Chegirma | `product_discount` (summa) | `discount_percent` (foiz) | **PDF** |

- **Nom** arzon yechildi: ИКПУ va klassifikator kodi — **bir xil son**, ya'ni
  ikkala yozilishda ham **bir qiymat** yuboriladi va hech bir o'qish
  ziddiyatga tushmaydi. Notanish maydonni JSON dekoder tashlab yuboradi;
  yo'q maydon esa rad etilgan chek.
- **Chegirma** shunday yechilmaydi (ikkovini yuborish ikki marta ayirilishi
  mumkin) → PDF yutadi, faqat `product_discount`.
- ⚠️ **Pulni bu yerdan hal qilib bo'lmaydi**, va u eng xavflisi. PDF'ga
  ergashildi, konversiya **bitta joyda** (`tiyinToSum`), va **testda
  muhrlangan** — provayder jonli terminalda tasdiqlaganda tuzatish bitta
  funksiya bo'ladi va test nima o'zgarganini aytadi.

### Kichik, lekin jimgina buzadigan narsalar
- ⚠️ **`mkSum` o'nlik matn sifatida yoziladi, `float64` orqali emas**:
  chegirma qatorlarga tiyingacha aniq bo'linadi, ya'ni ulush 3333 tiyin
  bo'lishi mumkin — `33.33` esa float64'da aniq emas va soliqqa
  `33.329999999999998` bo'lib ketardi.
- ⚠️ **HTTP status hukm emas, dalil**: bu kassa biznes rad javoblarini **500**
  bilan va o'qiladigan tana bilan beradi (`#2D - Z-отчет не был открыт` —
  smena ochilmagan, kassir buni o'n soniyada tuzatadi). Statusni o'qish uni
  "tarmoq uzilgan" ga aylantirardi — boshqa odamning muammosi va noto'g'risi.
- ⚠️ **Fiskal belgi kelmasa — fiskallashtirilmagan**, `success: true` bo'lsa
  ham. Bu — butun amalning maqsadi, va uni "yuborildi" deb yozish keyin
  aniqlab bo'lmaydigan yagona natija.
- ⚠️ **ИКПУ menyudan o'qiladi, chekdan emas** (`menuFiscal`) — nom va narxdan
  farqli. Ular mijoz rozi bo'lgan narsa; kod esa **mahsulot** haqidagi fakt.
  Lookup **ATMOS bilan umumiy**: ikkalasi ham chek yuboradi, ikki nusxa esa
  aynan yomon tomonga siljirdi — buxgalter kodni tuzatadi, onlayn chek to'g'ri
  chiqadi, peshtaxtadagi chek esa eskisini ko'tarib yuraveradi.
- `force_to_print: false` — planshetga ulangan printer yo'q, va birovning
  mashinasidagi chop dialogida osilgan yuborish mehmon peshtaxtada turganda
  osiladi. Mehmonning nusxasi — qaytgan QR.

### Oqim
`POST /staff/checks/{id}/fiscal` (server ishni quradi, `pending` yozadi) →
brauzer kassaga yuboradi → `PUT /staff/checks/{id}/fiscal` (server javobni
o'qiydi va yozadi). Ulanish tekshiruvi ham shunday: `GET/PUT /staff/fiscal`.

⚠️ **Fiskallashtirish pul olingandan keyin**, hech qachon oldin: oldin
yuborilsa karta rad etilishi bekor qila oladigan sotuv ro'yxatga tushadi, va
ortiqcha chek **qaytarish hujjati** bilan tuzatiladi — xatoning qimmat
yo'nalishi. Teskari bo'shliq (to'langan, hali yuborilmagan chek) ko'rinadi,
qayta yuboriladi, va `pending` aynan shuning uchun bor.

⚠️ **Yuborish yiqilsa to'lov yiqilmaydi.** Chek yopilgan, pul kassada.
Ekrandagi matn buni ataylab ta'kidlaydi: "to'lov qabul qilinmadi" deb o'qigan
kassir pulni **ikki marta** oladi — bu yerdagi mehmonga yetadigan yagona xato.

### ⚠️ Ochiq risk: brauzer HTTPS sahifadan HTTP kassaga so'rov yubormaydi
Ikki qoida birdan: **mixed content** va **Private Network Access**. Ikkalasi
ham brauzerning ataylab qilgan ishi va JS'dan aylanib o'tib bo'lmaydi.
- Alomat eng yomoni: `TypeError: Failed to fetch` — uzilgan kabel, noto'g'ri
  port va o'chiq kompyuter **bir xil** shu xatoni beradi. Ya'ni yagona
  *sozlama* sababi kassir ajrata olmaydigan sabab bo'lardi.
- Shuning uchun `lib/fiscal.ts` buni **oldindan tekshiradi** (`blockedReason`)
  va aniq nima qilishni yozadi: planshetda Chrome → sayt sozlamalari →
  "Insecure content" → Allow, yoki kassa manzilini HTTPS orqali ochish.
- **Hal qilinmagan**: jonli mijozda qaysi yo'l amaliy ekani sinalmagan.
  Variantlar — boshqariladigan planshetda Chrome siyosati
  (`InsecureContentAllowedForUrls`), kassa oldiga sertifikatli reverse proxy,
  yoki lokal relay. Birinchi mijozda hal qilinadi.

### Yozilgan fayllar
Backend: `internal/fiscal/{multikassa,multikassa_test}.go`, `fiscal.go`
(`Request`/`Encoder`/`EncoderFor`/`IsLocal`/`ErrLocalProvider`), `receipt.go`
(`Cashier`), `handlers/tillfiscal.go` (+`menuFiscal`), `handlers/fiscal.go`
(lokal provayder uchun manzil talabi va ping rad javobi), `handlers/payatmos.go`
(`menuIkpu` endi `menuFiscal` ustida), `models/{models,fiscal}.go`, `router.go`.

Frontend: `lib/fiscal.ts`, `app/kassa/FiscalPanel.tsx`, `app/kassa/PayDialog.tsx`,
`components/admin/FiscalEditor.tsx`, `lib/{types,api}.ts`, `lib/i18n/admin.ts`.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` hammasi yashil ✓ ·
`tsc --noEmit` ✓ · `npm run build` ✓
Yangi testlar: pul birligi (100×), kasrli so'mning omon qolishi, ИКПУ ikkala
nom bilan va bo'sh bo'lsa umuman yuborilmasligi, chegirma bir marta, 500
ichidagi rad javobi, fiskal belgisiz javobning rad etilishi, JSON bo'lmagan
javob, lokal provayderga manzil majburiyligi (va begona provayderning
manzili hisobga o'tmasligi).

### Keyingi qadam
1. **Mixed content'ni jonli planshetda hal qilish** — sotuvning sharti.
2. Smena: kassa `#2D` bilan rad etadi, ya'ni `open shift` (type 1) va
   `close shift` (type 2) bizning `cash_shift` ga ulanishi kerak.
3. Qaytarish (type 4) — encoder shakli ma'lum, kassada qaytarish oqimi yo'q.
4. Qolgan besh provayderdan API hujjati.

---

## 2026-08-17 — Fiskal relay + provayder savollari ✅

Oldingi yozuvdagi "hal qilinmagan risk" ikki tomondan yopildi.

### ⚠️ Avval: riskni noto'g'ri tavsiflagan edim
Uchta brauzer qoidasi bor, men faqat birinchisini aytgandim:

| Qoida | Yechim |
|---|---|
| Mixed content | **`localhost` bunga kirmaydi** |
| Private Network Access | `localhost` da masala emas |
| **CORS** | **noma'lum — provayderdan so'raladi** |

⚠️ **`localhost` — bizning hiyla emas, provayderning o'z tavsiyasi**: PDF'da
asosiy manzil sifatida `http://localhost:8080` va `http://localhost:12346`
yozilgan, ya'ni Multikassa integratsiya qiladigan dastur **o'sha kompyuterda**
ishlashini kutadi. Brauzerlar `localhost` ni "potentially trustworthy origin"
deb biladi.

⚠️ **Va shu yerda xato yozgan edim**: `blockedReason` `http://localhost:8080`
ni ham to'sardi — provayder tavsiya qilgan sozlamani rad etib, restoranga
hech qachon kerak bo'lmagan brauzer sozlamasini bo'shatishni aytardi.
`isLoopback` bilan tuzatildi.

Qolgan yagona haqiqiy noma'lum — **CORS**: sarlavha bo'lmasa chek
fiskallashadi-yu **fiskal belgi va QR bizga qaytmaydi**, ya'ni butun maqsad
yo'qoladi.

### Relay: uchala qoidaga ham bog'liq bo'lmagan yo'l
`cmd/fiscalagent` — kassa kompyuterida turadigan kichik Go binari (Windows
`.exe` ~9 MB). U bizning serverga **chiqishga** ulanadi, chek so'raydi,
`localhost` ga yuboradi va javobni qaytaradi. Chiquvchi ulanishga hech qanday
brauzer qoidasi qo'llanmaydi va tarmoqda **hech qanday port ochilmaydi**.

⚠️ **Navbat yo'q, va bu ataylab.** Ish **hisoblanadi, saqlanmaydi**:
`fiscal.status == "pending"` bo'lgan buyurtma — kutayotgan chek, va bu maydon
allaqachon boshqa sabab bilan bor edi. Alohida `job` kolleksiyasi bir faktning
ikkinchi yozuvi bo'lardi va konteyner yuborish o'rtasida qayta ishga tushgan
birinchi kunda ajrab ketardi — natijasi ikki marta yuborilgan yoki umuman
yuborilmagan chek. Bonusi: butun yo'l restartga chidamli, deploy hech qanday
chekni yo'qotmaydi.

⚠️ **Qaysi yo'l ishlatilishini vaqt belgisi hal qiladi, sozlama emas**
(`agentSeenAt`, `agentAlive = 90s`). Relay so'nggi paytda ish so'ragan bo'lsa
cheklar unga ketadi, aks holda kassa ekrani o'zi chaqiradi. Sozlanadigan
bayroq bo'lsa: Windows yangilanishi uchun qayta yuklangan kompyuter har chekni
jimgina yutib turardi, to kimdir katakchani eslamaguncha. Konsoldagi
`attention: "down"` bilan bir qoida — saqlangan bayroq soat undan o'tishi
bilan eskiradi.

⚠️ **Ikki yozuvchi bo'lmaydi**: relay tirik bo'lsa server kassa ekraniga
`{queued: true}` qaytaradi va ish **bermaydi**. Bitta sotuvni ikki chaqiruvchi
yuborsa — ikki marta ro'yxatdan o'tgan chek, va uni tuzatish qaytarish hujjati
bilan bo'ladi.

⚠️ **Kutish chegaralangan (20s) va tugagach ekran baribir ko'rsatiladi**:
burchakdagi kompyuter Windows yangilanishi o'rtasida bo'lishi mumkin, va
kassirni spinner oldida ushlab turish navbatni u tuzata olmaydigan narsa uchun
to'xtatadi. Chek `pending` bo'lib qoladi — u aynan shunday — va relay qaytgan
zahoti yuboradi.

⚠️ **Bo'sh kalit hech qachon mos kelmaydi** (`matchAgentToken`, testda
muhrlangan). Relay o'rnatmagan har bir filialda `agentToken == ""` — bitta
tushib qolgan tekshiruv bo'sh sarlavha yuborgan birinchi skanerga **shu
xususiyatni yoqmagan hamma restoranning** jonli sotuvini berardi. Endpoint
ochiq internetda, ya'ni bu fayldagi eng qimmat qator.

- Kalit **bir marta ko'rsatiladi** (`POST /admin/fiscal/agent-token`) —
  shuning uchun almashtirish haqiqiy bekor qilish, ikkinchi ishlaydigan kalit
  emas. Almashtirilganda `agentSeenAt` **tozalanadi**: yangi kalit
  yetib-yetmaganini bilish kerak bo'lgan aynan o'sha daqiqada "ulangan" deb
  turgan qator yolg'on bo'lardi.
- Solishtirish `subtle.ConstantTimeCompare`, va qidiruv **Mongo so'rovi emas**:
  baza solishtiruvi doimiy vaqtli emas.
- **401 qaytariladi** (webhook'lardan farqli): so'rovchi biz yozgan dastur,
  ekraniga hech kim qaramaydi, va u "kalitim noto'g'ri" deb **o'z jurnaliga**
  yoza olishi kerak — jimgina abadiy qayta urinish o'rniga.
- Long-poll (~25s), soket emas — `AlertBell` bilan bir tanlov: har proxy'dan
  o'tadi, yangi bog'liqlik yo'q, va qayta ulanish shunchaki qayta so'rash.
  Router timeout'idan (30s) qisqa, aks holda har jim daqiqa agent jurnaliga
  xato yozardi va haqiqiy nosozliklar topilmay qolardi.
- ⚠️ **Yozish yo'li ikkala transport uchun umumiy** (`recordFiling`): ular
  faqat soketni kim ushlagani bilan farq qiladi. Ikki nusxa eng muhim narsada
  ajrab ketardi — qaysi javob "yuborildi" hisoblanishida — va farqi tekshiruv
  paytida bilinardi.
- Agent **hech qanday biznes mantiq tutmaydi va tutmasligi kerak**: hujjat
  serverda quriladi, u bir sakrash tashiydi. Navbatni jarayonda saqlash har
  Windows yangilanishida yo'qolardi va omon qolganini ikki marta yuborardi —
  shuning uchun qayta urinish **so'rab olish** orqali bo'ladi.

### Provayder savollari
`docs/multikassa-savollar.md` — 10 ta savol, har birida **nega so'ralayotgani**
va **javob nimani o'zgartirishi**. Eng muhim ikkitasi: **pul birligi** (100×)
va **CORS** (arxitekturani hal qiladi). Qolganlari: sinov muhiti, smena,
xato kodlari ro'yxati, qaytarish maydonlari, maydon nomlari, chegirma,
integratsiya rejimi, tarif.

### Yozilgan fayllar
Backend: `cmd/fiscalagent/main.go`, `handlers/{fiscalagent,fiscalagent_test}.go`,
`handlers/tillfiscal.go` (`recordFiling` ajratildi, `queued`, `relay`),
`models/fiscal.go` (`AgentToken`, `AgentSeenAt`), `router.go` (3 marshrut).

Frontend: `lib/fiscal.ts` (`isLoopback`), `app/kassa/PayDialog.tsx`
(`waitForFiling`), `components/admin/FiscalEditor.tsx` (relay bo'limi),
`lib/{types,api}.ts`, `lib/i18n/admin.ts`.

Docs: `docs/multikassa-savollar.md`.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` hammasi yashil ✓ ·
`GOOS=windows go build ./cmd/fiscalagent` ✓ · `tsc --noEmit` ✓ ·
`npm run build` ✓
Yangi testlar: bo'sh kalit hech qachon mos kelmasligi (ikki tomondan),
relay faqat tirikligida afzal ko'rilishi, poll oynasi router timeout'idan
qisqaligi, kalitning uzunligi va takrorlanmasligi, relay o'z mashinasiga
qaytishi.

### Keyingi qadam
1. **Savollarni yuborish** — ikkita javob arxitekturani yopadi.
2. Agentni Windows xizmati sifatida o'rnatish (hozir qo'lda ishga tushiriladi)
   va DEPLOY.md ga qo'llanma.
3. Smena: `open shift` / `close shift` ni `cash_shift` ga bog'lash.
4. Qaytarish (type 4) — encoder shakli ma'lum, oqim yo'q.

---

## 2026-08-17 — Yuborilmagan cheklar ko'rinadigan bo'ldi + smena o'zi ochiladi ✅

Savollar provayderga yuborildi (javob kutilmoqda). Javobga bog'liq bo'lmagan
ishlar bajarildi.

### ⚠️ Eng katta bo'shliq: "keyin qayta yuborish mumkin" degan va'da
Oldingi ishda kod izohida "chek qayta yuborilishi mumkin" deb yozgan edim, va
unga **qaytadigan yo'l yo'q edi**. Bu — CLAUDE.md dagi `pos.failed` darsining
og'irroq varianti: kassaga tushmagan buyurtmani restoran bir soatda biladi
(mehmon kutib turibdi), fiskallashtirilmagan chekni esa **hech nima**
bildirmaydi — ovqat chiqdi, mehmon ketdi, hamma ekran normal ko'rinadi. Bo'shliq
tekshiruvda ochiladi.

### Yechim: chek yopilganda `pending` qo'yiladi
`StaffCloseCheck` endi pul olingan **o'sha daqiqada** `fiscal.status: pending`
yozadi, yuborishga urinilganda emas. Ikki natijasi bor va ikkalasi ham
"cheklarni yuboradigan tizim" bilan "odatda yuboradigan tizim" farqi:
- Relay aynan shu holatni so'raydi, ya'ni chek **brauzer umuman
  qatnashmasa ham** yuboriladi. Kassirning tabi pul olish bilan yuborish
  orasida yiqilsa chek jimgina yo'qolardi — bu bo'shliq o'z-o'zidan yopildi.
- "To'landi, lekin yuborilmadi" **aniq savolga** aylandi. Bayroq faqat
  urinilganda yozilsa, eng muhim holatlar — umuman urinilmaganlar —
  ogohlantirish uchun ko'rinmas bo'lardi.

⚠️ **Ishlaydigan adapter borligiga bog'langan**, faqat sozlamaga emas: so'rov
qura olmaydigan provayder har sotuvni hech qachon yuborilmaydigan qarzga
aylantirardi, va ustiga qurilgan ogohlantirish doimiy qizil nishon bo'lardi —
odamlar o'chirib qo'yadigan turi.

### ⚠️ Mongo tuzog'i — bu safar teskari tomondan
`unfiledFiscalFilter` da **`$in`, `$nin` emas** — `pendingTillFilter` ning
aynan aksi, bir xil xatti-harakatning ikkinchi yuzi. Yo'q maydon `$nin` ga
**mos keladi**, ya'ni "filed emas" degan filtr kassa ulanishidan **oldingi**
har bir chekni va kassasi yo'q restoranning hamma cheklarini yig'ib olardi.
Ular yuborilmagan chek emas — ular hech qachon chek qarz bo'lmagan sotuvlar.

⚠️ **Yosh chegarasi yo'q**: bir soat oldin yiqilgan va hali yuborilmagan chek
**ko'proq** e'tiborga arziydi, kamroq emas (POS bannerdagi bilan bir qoida).
U filtrdan faqat **yuborilgani** uchun chiqadi.

### Ko'rinadigan joylar
- `/admin/alerts` → `fiscal.unfiled` — banner, **ovozsiz**
  (`pos.unaccepted` bilan bir hukm): to'xtatuvchi amal — qayta yuborish, va u
  kassa kompyuteri o'chiq bo'lsa yana yiqilishi mumkin. Hech qanday amal
  bilan jimlatib bo'lmaydigan signal — odam o'chirib qo'yadigan signal, va bu
  odat qolgan ikkitasiga ham ko'chadi.
- **Kassa ekranida** (`UnfiledPanel`) — chek panelining **tepasida**, va
  ro'yxat bo'sh bo'lsa **umuman chizilmaydi**. Doim turgan "hammasi yaxshi"
  paneli — o'qilmaydigan panel, bu esa bo'sh bo'lmagan yagona kunda
  ko'rinishi kerak. Kassada, chunki bularning ko'pini tuzata oladigan odam
  (smenani ochish, kompyuterni yoqish) o'sha yerda turibdi.
- "Hammasini qayta yuborish" **ketma-ket**, parallel emas: bularning hammasi
  bitta kompyuterdagi bitta kassaga uriladi, va allaqachon qiynalayotgan
  mashinaga bir vaqtda o'nta so'rov — tuzatiladigan qoloqni osilgan kassaga
  aylantirishning yo'li.

### `#2D` — smena endi o'zi ochiladi
Kassa smena ochilmagan bo'lsa sotuvni `#2D` bilan rad etadi. Bu **har kuni
ertalab, birinchi sotuvda** bo'ladi va tuzatilishi butunlay mexanik.
- Xato sifatida ko'rsatilsa, kassir buni **o'rganishi, eslab qolishi va mehmon
  kutib turganda qilishi** kerak bo'lardi.
- ⚠️ **Oldindan tekshirilmaydi**: har chekka bitta qo'shimcha so'rov qo'shardi.
  ⚠️ **Oldindan ochilmaydi ham**: allaqachon ochiq kun uchun soliq qo'mitasiga
  smena ochish hujjatini yuborish demakdir. Kassaning o'zi — bu savolning
  yagona ishonchli manbasi.
- ⚠️ **Kod bo'yicha tanib olinadi (`#2D`), matn bo'yicha emas**: matn ruscha
  keladi va provayder uni istalgan versiyada o'zgartirishi mumkin. Matnga
  bog'langan tekshiruv sinovda ishlab, yangilanishdan keyin jimgina to'xtardi —
  alomati "har kuni birinchi sotuv yiqiladi".
- ⚠️ **Bu holatda nosozlik yozilmaydi**: chek `pending` bo'lib qoladi, chunki
  haqiqat shu — chekning o'zida hech nima rad etilmagan, faqat kassaning
  holatida. Nosozlik yozilsa **har restoranning har ertalabki birinchi
  sotuvi** ogohlantirishga tushardi, va har kuni ochilishda qizil bo'ladigan
  ogohlantirishni hafta oxiriga borib hech kim o'qimaydi.
- Relayda ham shu: server javobida `{"next": <job>}` qaytaradi, agent uni
  bajaradi va keyingi so'rovda o'sha chek qaytadan beriladi. Busiz **tor
  sikl** bo'lardi: rad etilgan chek `pending` qoladi → agent yana so'raydi →
  yana rad etiladi.
- Brauzer tomonida **bir marta** qayta uriniladi: ochish yordam bermasa kassa
  biz tushunmagan sabab bilan rad etyapti, va abadiy urinish mehmon
  peshtaxtada turganda jimgina aylanish bo'lardi.

### O'rnatish qo'llanmasi
`docs/fiskal-agent.md` — restoranga beriladigan hujjat: fayl, sinov, xatolar
jadvali, avtoyuklash (Task Scheduler `/sc onstart` — foydalanuvchi kirganda
emas, kassa kompyuteri ko'pincha kirilmagan turadi; yoki NSSM xizmati),
yangilash, va ulagichsiz ishlash varianti.

⚠️ Qo'llanmada alohida yozilgan: **yangilash paytida yopilgan cheklar
yo'qolmaydi**, chunki navbat ulagichning ichida emas, **serverda**.

### Yozilgan fayllar
Backend: `handlers/tillclose.go` (yopishda `pending`), `handlers/tillfiscal.go`
(`unfiledFiscalFilter`, `StaffUnfiledChecks`, `shiftJobFor`),
`handlers/fiscalagent.go` (`next` javobi), `handlers/adminstats.go`
(`fiscal.unfiled`), `fiscal/multikassa.go` (`OpenShift`, `NeedsShift`,
`ShiftOpener`), `cmd/fiscalagent/main.go` (follow-up job), `router.go`.

Frontend: `app/kassa/UnfiledPanel.tsx`, `app/kassa/{PayDialog,page}.tsx`,
`lib/api.ts`, `lib/i18n/admin.ts`.

Docs: `docs/fiskal-agent.md`.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ ·
`GOOS=windows go build ./cmd/fiscalagent` ✓ · `tsc --noEmit` ✓ ·
`npm run build` ✓
Yangi testlar: `#2D` kod bo'yicha tanilishi (va boshqa rad javoblari,
boshqa provayder, `nil` bunga kirmasligi), smena ochishda `items` **`[]`**
bo'lishi (`null` emas — Go'ning JSON tuzog'i, bu safar birovning parseriga
boradigani).

### Keyingi qadam
1. **Provayder javobi** — pul birligi va CORS.
2. Smenani yopish (Z-hisobot) `cash_shift` bilan bog'lash.
3. Qaytarish (type 4) — 6-savolning javobi kerak.

---

## 2026-08-17 — Kassa kuni (Z-hisobot) `cash_shift` bilan bog'landi ✅

### ⚠️ Avval: bu ikkita smena, va ular bir xil emas
- **`cash_shift` — bizniki**: menejer ochadi, kassani sanaydi, yopadi. Smena
  almashganda **bir kunda ikki marta** bo'lishi mumkin.
- **Fiskal kun — kassaniki**: birinchi sotuv ochadi (`#2D` mantiqi), Z-hisobot
  yopadi, va u **soliq hujjati**.

Ularni birlashtirish ikki tomonga ham buzilardi: har smena almashuvida fiskal
kunni yopish bitta soliq kunini ikkiga bo'lardi va soat to'rtda Z-hisobot
yuborardi; kassaning o'ziga qoldirish esa kassa sanog'i yonida kassaning o'z
raqamlari hech qachon turmasligini anglatardi.

### Bog'lanish: **so'rov**, chaqiruv emas
⚠️ `cash_shift` **paneldan** yopiladi — u ko'pincha uydagi noutbuk. Kassa esa
restoran tarmog'idagi kompyuter. Panel unga hech qachon yeta olmaydi.

Shuning uchun kassa smenasini yopish **kunni yopishni so'raydi**
(`closeDayRequestedAt` — bitta nullable vaqt belgisi, navbat kolleksiyasi emas),
va yetadigan kim bo'lsa — relay yoki kassa ekrani — bajaradi.

⚠️ **Tartib bepul chiqadi**: relay avval cheklarni so'raydi, keyin kun yakunini,
ya'ni Z-hisobot yuborilmagan chekdan **hech qachon oldinga o'ta olmaydi**.
Tartibning o'zi qo'riqchi — unutiladigan alohida tekshiruv yo'q.

⚠️ **Kassa sanog'ini hech qachon to'smaydi**: pul sanash — o'sha endpointning
ishi va u allaqachon bajarilgan. Burchakdagi kompyuterda yuborilmagan chek bor
deb sanoqni yozmaslik — pulni tushuntirilmagan qoldirish. Sabab smena yonida
`fiscalNote` bo'lib qaytadi.

### ⚠️ Ushlangan xato: qo'riqchi noto'g'ri filtrni ishlatardi
Boshida `unfiledFiscalFilter` ishlatilgan edi — unda ogohlantirish uchun
**5 daqiqalik muhlat** bor. Ya'ni yangi yuborilmagan chek Z-hisobotni
to'smasdi, va aynan **kechqurungi oxirgi sotuvlar** o'sha oynaning ichida
bo'ladi. Chek kun jamlanayotganda yo'lda bo'lsa — yo kun raqamlaridan tushib
qoladi, yo ertangi kunga tushadi, va ikkalasini ham keyin tuzatib bo'lmaydi.

Endi alohida `anyUnfiledFilter` — **muhlatsiz**. Ikkalasi ham testda muhrlangan,
va test ikkinchisining muhlati **saqlanishini** ham tekshiradi (busiz har sotuv
to'langan zahoti qo'ng'iroq chalardi).

### Z-hisobot smenaga yoziladi (`CashShift.Fiscal`)
⚠️ **Bu — o'sha kunlik tushumning ikkinchi, mustaqil sanog'i**: smenaning
`expected` i biz yozgan buyurtmalardan, bu esa ularni davlatga topshirgan
mashinadan. Kassa kam chiqqanda birinchi foydali savol — naqd qaysi biriga mos
kelishi, va busiz bu savolni berib bo'lmaydi, faqat bahslashish mumkin.

⚠️ **`expected` ni tuzatmaydi**: u yopish paytida ataylab muzlatilgan, va
ikkinchi manba kelganda o'zini qayta yozadigan kamomad — tekshirib bo'lmaydigan
kamomad.

⚠️ **Qaytarishlar alohida qator**, jamiga qo'shilmaydi: ko'p qaytarish bo'lgan-u
balansga kelgan kun bilan sokin kun — ikki boshqa hikoya, va qo'shish uni
yashiradi.

### ⚠️ Bir provayderdan uchinchi pul formati
Kun yakuni javobi **formatlangan so'm** (`"6,651,020.00"`), ularning `zReport`
endpointi esa **yalang'och tiyin** (`"665102000"`), sotuv so'rovi esa yuqorida
tiyin va qatorlarda so'm. Shuning uchun `ParseZReport` ning o'z parseri bor
(`mkMoney`) va testda muhrlangan — bu raqam ega **haqiqiy pul bilan**
solishtiradigan raqam.
⚠️ Tiyin **yaxlitlanmaydi, kesiladi**: bularning har biri allaqachon bilingan
summalarning **jami**, va jamini yuqoriga yaxlitlash uni bo'laklari
yig'indisidan katta qilishi mumkin — kassani tekshirayotgan odam uchun bu
"kassa ortiqcha" bo'lib o'qiladi.

### Boshqa qarorlar
- **Ikki alohida endpoint** (`/fiscal/agent/job` va `/fiscal/agent/close-day`,
  `POST/PUT /staff/fiscal/close-day`): ular boshqa hujjat yozadi, va bitta
  endpoint maydonga qarab tarmoqlansa — bitta noto'g'ri tarmoq Z-hisobotni
  sotuvning natijasi deb yozadi.
- Ish turi **nomlanadi** (`kind: filing|closeDay`), `orderId` borligidan
  taxmin qilinmaydi: relay bir marta xato taxmin qilsa, aynan muhim kunda
  qiladi.
- **So'rov javob qanday bo'lishidan qat'i nazar tozalanadi**: omon qolgan
  so'rov relayni har pollda Z-hisobot yuborishga majburlardi, va Z-hisobot —
  siklda qayta uriniladigan amal emas. Nosozlik smenaga yoziladi.
- Kassa ekranida tugma **tasdiq bilan** (bu ilovada kam uchraydi va bu yerda
  o'rinli: qaytarib bo'lmaydi, tunda bosiladi, ekran esa tez va ho'l barmoq
  bilan ishlatiladi) va **yuborilmagan cheklar ro'yxatidan pastda** — aks holda
  kassir rad javobini uning sababidan oldin ko'rardi.

### ⚠️ Bajarilmagani: paneldagi ko'rinish
`cash_shift` ning **frontendi umuman yo'q** — backend endpointlari bor
(`/admin/cash/shift/*`, `/admin/reports/cash`), lekin panelda sahifasi yo'q.
Bu men qo'shgan bo'shliq emas, oldindan shunday edi. Shuning uchun Z-hisobot
**saqlanadi va API'da qaytadi**, lekin panelda ko'rsatiladigan joyi yo'q.
Kassa sahifasini qurish alohida ish.

### Yozilgan fayllar
Backend: `handlers/fiscalday.go`, `handlers/fiscalday_test.go`,
`fiscal/multikassa.go` (`CloseShift`, `ShiftCloser`, `ZReport`, `ParseZReport`,
`mkMoney`), `handlers/tillfiscal.go` (`anyUnfiledFilter`),
`handlers/fiscalagent.go` (ish turlari, `FiscalAgentCloseDay`),
`handlers/cash.go` (yopishda so'rov), `models/{fiscal,models}.go`,
`cmd/fiscalagent/main.go`, `router.go`.

Frontend: `app/kassa/CloseDayButton.tsx`, `app/kassa/page.tsx`,
`lib/{types,api}.ts`, `lib/i18n/admin.ts`.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ ·
`GOOS=windows go build ./cmd/fiscalagent` ✓ · `tsc --noEmit` ✓ ·
`npm run build` ✓
Yangi testlar: Z-hisobot summalari so'mda o'qilishi (100×/1000× emas),
Z-hisobot bo'lmagan javoblar rad etilishi, kun yopish `items: []` yuborishi,
qo'riqchida muhlat **yo'qligi** va ogohlantirishda **borligi**, ikkala filtr
ham `$nin` ishlatmasligi, `withBranch` umumiy filtrni o'zgartirmasligi.

### Keyingi qadam
1. **Provayder javobi** — pul birligi va CORS.
2. **Kassa sahifasi panelda** (`/admin/cash`) — smena, sanoq, va yonida
   Z-hisobot. Hozir backend bor, ekran yo'q.
3. Qaytarish (type 4) — 6-savolning javobi kerak.

---

## 2026-08-17 — Kassa sahifasi (`/admin/cash`) ✅

Backend allaqachon bor edi (`/admin/cash/shift/*`, `/admin/reports/cash`), ekran
yo'q edi — ya'ni kassa smenasini faqat API orqali ochib-yopish mumkin edi va
Z-hisobot hech qayerda ko'rinmasdi.

### ⚠️ Sahifaning mahsuloti — farq, jami emas
Bu modeldagi izohning ekrandagi davomi. "Bo'lishi kerak: 1 240 000" ni
ko'rsatib, sanalganini yozdirib, faqat ikkinchisini saqlaydigan ekran **hech
nima yozmagan**: ochish uchun qurilgan kamomad uni qilgan bo'lishi mumkin
bo'lgan odam tomonidan o'chirilgan.

⚠️ **Shuning uchun kutilgan summa sanoq maydonining yonida turmaydi** —
u faqat **raqam kiritilgandan keyin** taqqoslash bilan birga chiqadi. Bo'sh
maydon yonidagi kutilgan summa ko'chirib yozishga taklif, va ekrandan o'qilgani
uchun mos kelgan sanoq — aynan shu sahifa oldini olish uchun qurilgan yozuv.

### Ko'rinadigan tuzilma
- **Bo'lishi kerak** — katta raqam, ostida qanday chiqqani (qoldiq, peshtaxta
  naqdi, kuryerlar topshirgani, qo'lda kirim/chiqim).
- ⚠️ **"Kuryerlar qo'lida" ro'yxatdan tashqarida va vizual ajratilgan**, chunki
  u jamiga **kirmaydi**: topshirilmagan naqd haqiqiy pul, lekin u bu kassada
  emas. Ustunga qo'shish har yetkazishni ikkilantirardi, umuman ko'rsatmaslik
  esa kam chiqqan kassa haqidagi birinchi savolni javobsiz qoldirardi.
- Qo'lda harakatlar ro'yxati, kirim/chiqim formasi (sababi majburiy).
- Yopish: sanoq → farq → farq bo'lsa sabab maydoni ochiladi.
- Smena yo'q bo'lsa: ochish formasi **va oxirgi yopilgan smena** — "kassa
  oxirgi marta qachon sanalgan va qanday chiqqan" degan savol bo'sh ekrandan
  javob olmaydi.

### Z-hisobot shu yerda ko'rinadi
Yopilgan smena ostida kassaning **o'z** hisoboti alohida ramkada, va tagida
bir jumla: bu raqamlar fiskal kassadan, yuqoridagilar bizning
buyurtmalarimizdan. ⚠️ **Ikki mustaqil sanoq bir ekranda bo'lgandagina** "naqd
qaysi biriga mos keladi?" degan savolni berish mumkin — busiz faqat
bahslashish mumkin. Yuqoridagi raqamlarni **tuzatmaydi**.

Qaytarishlar bu yerda ham alohida qator.

### Smena yopilganda fiskal kun so'rovi
Yopish javobidagi `fiscalNote` ekranda ko'rsatiladi (odatda
"fiskallashtirilmagan cheklar bor"). ⚠️ Yutib yuborilmaydi: sanoq baribir
muvaffaqiyatli bo'ldi, va o'sha cheklarni yubortira oladigan yagona odam —
shu ekranga qarab turgan menejer.

### Navigatsiya
`Hisob-kitob` yonida (`LuBanknote`): ikkalasi ham nomi bilan binodan chiqadigan
pul, va tunda kassani sanaydigan odam odatda smenani ham to'laydigan odam.

### Yozilgan fayllar
Frontend: `app/admin/cash/page.tsx`, `app/admin/layout.tsx` (nav),
`lib/types.ts` (`CashShift`, `CashFigures`, `CashEntry`), `lib/api.ts`
(`cashShift`, `openCashShift`, `closeCashShift`, `addCashEntry`),
`lib/i18n/admin.ts` (`cash` bo'limi + nav, uz/ru/en).

Backend o'zgarmadi — endpointlar allaqachon bor edi.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ · `tsc --noEmit` ✓ ·
`npm run build` ✓ (`/admin/cash` 4.54 kB).

### Keyingi qadam
1. **Provayder javobi** — pul birligi va CORS.
2. Kassa tarixi (`/admin/reports/cash` allaqachon bor — hisobot sahifasiga
   qo'shish).
3. Qaytarish (type 4) — 6-savolning javobi kerak.

---

## 2026-08-17 — Kassa tarixi hisobot sahifasida ✅

`/admin/reports` ga beshinchi tab: **Kassa**. Backend hisoboti allaqachon bor
edi (`AdminCashReport`, Excel bilan birga), ekranga chiqmagan edi.

### ⚠️ Qatorlar — smenalar, jamlanma emas
Davr jami noli seshanba 80 000 kamomad va payshanba 80 000 ortiqchani anglatishi
mumkin — ikki boshqa odam bilan ikki boshqa suhbat, "hech nima bo'lmadi"
degan bitta raqamga qo'shilgan. Jami qator eng pastda, chunki bu yerdagi eng
qiziq bo'lmagan narsa.

⚠️ **Balanslashmagan smenalar jadval tepasida alohida sanaladi** (kamomad va
ortiqcha **alohida, hech qachon qo'shilmaydi**): pul yetishmasligi — odam
haqidagi savol, ortiqcha esa odatda noto'g'ri sanoq yoki yozilmagan chiqim.
O'ttiz qatorli ro'yxatda uchtasi muhim bo'lsa, o'sha uchtasi ko'milib ketadi —
va ular sahifani ochishning yagona sababi.

Har farq **o'z jumlasi bilan** turadi (server sababsiz saqlamaydi) — bu uni
ayblovdan yozuvga aylantiradigan narsa.

### ⚠️ Yo'lda topilgan xato: fiskal taqqoslash noto'g'ri edi
Boshida `counted` (butun kassa sanog'i) kassaning **naqd sotuvi** bilan
solishtirilgan edi. Bu har smenada katta farq beradi va ma'nosiz: `counted`
ichida boshlang'ich qoldiq, kuryerlar topshirgani va qo'lda harakatlar ham bor,
kassa esa faqat **sotuvni** biladi. Har qatorda chiqadigan "farq" — haqiqiy
farqni ko'rinmas qiladigan narsa.

Tuzatish: **`CashShift.CounterCash`** — peshtaxta naqdi endi `Expected` bilan
birga **muzlatiladi**. Bu bizning yagona raqamimiz bo'lib, kassa javob
beradigan savolga javob beradi. Ikkalasi mos kelmasa — bir xil sotuvlarning
ikki yozuvidan biri xato, va aynan shu so'ralishi kerak bo'lgan savol.

⚠️ Bu o'tgan yozuvdagi da'voni ham tuzatadi: `counterCash` muzlatilmasa, "ikki
mustaqil sanoq" solishtirib bo'lmaydigan ikki sanoq bo'lardi.

Fiskal farq **faqat nol bo'lmaganda** ko'rsatiladi: ikki mustaqil sanoq joyga
arziydi aynan farq qilganda, har qatorda yonma-yon chizish esa muhim
qatorlarni topishni qiyinlashtiradi.

### Tab tartibi
Kassa **oxirgi**, chunki u yagona teskari o'qiladigan tab: qolganlari "qanday
ishladik" ga javob beradi, bu — "biror narsa yo'qoldimi", va bu savol
qolganlaridan **keyin** beriladi, o'rniga emas.

### Yozilgan fayllar
Backend: `models/models.go` (`CashShift.CounterCash`), `handlers/cash.go`
(yopishda muzlatish).

Frontend: `components/admin/reports/CashReport.tsx`, `app/admin/reports/page.tsx`
(tab), `app/admin/cash/page.tsx` (fiskal blokda peshtaxta naqdi yonma-yon),
`lib/{types,api}.ts` (`CashReportResponse`, `cashReport`),
`lib/i18n/admin.ts` (`reports.cash` + tab nomi, uz/ru/en).

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ · `tsc --noEmit` ✓ ·
`npm run build` ✓ (`/admin/reports` 5.38 kB, `/admin/cash` 4.55 kB).

### Keyingi qadam
1. **Provayder javobi** — pul birligi va CORS.
2. Qaytarish (type 4) — 6-savolning javobi kerak.

---

## 2026-08-17 — POS yo'nalishi bo'yicha qarorlar yozib qo'yildi 📝

Kod yozilmadi. `docs/pos-reja.md` — kassa/POS bo'yicha qabul qilingan qarorlar
va ochiq savollar. Qisqacha:

- **Windows uchun alohida `.exe`** kassa va ofitsiant ilovasi. Electron rad
  etildi (150+ MB, 4 GB monoblokda og'ir), lekin ⚠️ **Tauri Electron emas** —
  u tizim WebView2 ni ishlatadi, ~10 MB. Tavsiya: **Wails (Go + WebView2)**,
  chunki mavjud React UI qayta ishlatiladi va `cmd/fiscalagent` allaqachon
  Go'da — printer, COM port, fiskal kassa, lokal navbat **bitta jarayonda**.
  Restoranda **bitta** `.exe` turadi: bugungi agent o'sha ilovaning yadrosi.
- **Ofitsiant**: avval Windows, keyin **Keel Waiter** — do'konlarda yagona
  ilova, restoranga ulanib hisobga kiradi. ⚠️ Bu hozirgi kodga ta'sir qiladi:
  ofitsiant ekrani kassadan **mustaqil** komponentlar bilan qurilishi kerak.
- **Menyu**: rasmli yoki rangli plitka, restoran tanlaydi. ⚠️ Standart —
  **rangli**, chunki 200+ taomda u aslida tezroq va nol qiymat bugungi
  xatti-harakat bo'lishi kerak.
- **Uchta chek alohida dizayn qilinadi**: oshxona (⚠️ narxsiz), kassa, mijoz
  (⚠️ fiskal QR faqat shunda). Qog'oz kengligi — sozlama, taxmin emas.
- **Oflayn** — sotuvning sharti. ⚠️ Fiskal chek oflayn **ishlaydi**, chunki
  kassa lokal. Uchta qiyin qaror: chek raqami prefiksi (to'qnashmasligi
  uchun), ortiqcha sotishni **qabul qilish**, va serverga yaratilgan vaqti
  bilan idempotent yuborish.
- **Ombor/texkarta/inventarizatsiya — hozir emas.** Lekin model ikki narsani
  hisobga oladi: nima brendniki va nima filialniki, hamda **markaziy oshxona**
  (tsex) mavjudligi.

### ⚠️ Xavfsizlik birinchi bo'lim, tasodifan emas
- **Oflayn yangi xavf yuzasi ochadi**: bugun mijoz ma'lumoti faqat serverda,
  oflayn kassa esa restorandagi qulflanmagan kompyuterda saqlashni talab
  qiladi. Shuning uchun lokal saqlanadigan narsalar **oq ro'yxat**: menyu,
  bugungi cheklar, stol xaritasi, PIN hashlari. Mijozlar bazasi, telefonlar,
  manzillar, tarix, ballar — **hech qachon**. Yuborilgan chek lokal bazadan
  o'chiriladi.
- ⚠️ **PIN — autentifikatsiya emas, imzo.** Qurilma server tokeni bilan
  tasdiqlanadi; PIN o'sha qurilmadagi **amalni imzolaydi**. 4 raqam kalit
  bo'la olmaydi. Filial ichida unique, bcrypt, urinishlar cheklangan.
- ⚠️ **Hozirgi holat xato**: planshet butun kecha bitta ishchi tokeni bilan
  ochiq, ya'ni void/chegirmadagi "kim qildi" doim planshetga kirgan odam.
  Javobgarlik uchun qurilgan mexanizm login modeli tufayli ishlamayapti.
- ⚠️ **Avtoyangilanish — butun platformaga ochiq eshik**: imzolangan binar,
  imzo tekshirilgandan keyin qo'llash, yangilanish manzili sozlamadan
  o'qilmaydi.

### Tartib
1. Buyurtmalar linzasi · 2. Kassir PIN · 3. Kassa UI monoblokka ·
4. Chek dizayni · 5. Windows ilova + printer · 6. Oflayn ·
7. Ofitsiant ekrani · *keyin*: texkarta, ombor, mobil Keel Waiter.

### Ochiq savollar (javobsiz kod yozilmaydi)
Wails/Tauri o'lchovi · O'zbekistonda tarqalgan printer modellari va ulanish
turi · monoblok RAM va Windows versiyasi (WebView2 Windows 7 da yo'q) ·
POS uchun alohida narx · code signing sertifikati.

---

## 2026-08-17 — Windows ilovasi: Wails tanlandi, svet o'chishi hisobga olindi 📝

Kod yozilmadi. `docs/pos-reja.md` yangilandi.

### Qaror: **Wails (Go + WebView2)**
Muhit tasdiqlandi — **Windows 10/11** (7 uchrasa jamoa bepul 10 ga o'tkazadi),
monobloklar **minimal protsessor, minimal 4 GB RAM**. Ya'ni WebView2 hamma
joyda bor va C#/WPF majburiyati yo'q.

Tauri emas, ikki sabab bilan: (1) printer, COM port, fiskal kassa va lokal
navbat **Go'da** bo'ladi va `cmd/fiscalagent` allaqachon Go'da — Tauri bilan bu
qatlam Rust'ga ko'chardi, va printer nosozliklari aynan o'sha qatlamda
tuzatiladi; (2) Tauri'ning eng kuchli ustunligi — rasmiy avtoyangilagich —
bizga kamroq keladi, chunki ⚠️ **kassa xizmat vaqtida jimgina qayta ishga
tushmasligi kerak** (fonda yuklab olish, keyingi startda qo'llash).

⚠️ **C# native rad etildi**, lekin unumdorlik uchun emas: oflayn talab kassaga
lokal biznes mantiq berishga majbur qiladi, va C# bu mantiqni **majburan
ikkilantirardi** — narxlash quvurining ikkinchi nusxasi, ikki tilda,
kompilyator ham testlar ham chegaradan o'tmagan holda. `composeOrder` ni nusxa
ko'chirishdan qochgan qaror bilan bir mantiq.

### Oyna
Framesiz, to'liq ekran, o'z tugmalarimiz. **Kichraytirish tugmasi yo'q** — na
kassada, na ofitsiantda: monoblokda ilova orqasida hech nima yo'q, va
kichraytirish "kassa o'chib qoldi" degan qo'ng'iroqqa aylanadi. ⚠️ Ilova
qotganda chiqish yo'li (`Ctrl+Shift+Q`) qolishi va hujjatda yozilishi shart.

### ⚠️ Svet o'chishi — bu sinxronizatsiya emas, chidamlilik
Monobloklar tokda ishlaydi. Svet o'chadi, generator kelganda qaytadan yonadi —
va bu **toza yopilish emas**: ogohlantirish yo'q, diskka yozishga imkon yo'q.

- **Asosiy qoida**: har o'zgarish **ekranda ko'rsatilishidan oldin** diskka
  yoziladi. Aks holda ekran yolg'on gapirgan bo'ladi.
- ⚠️ **SQLite: WAL + `synchronous=FULL`.** Standart `NORMAL` svet o'chganda
  oxirgi tranzaksiyalarni **yo'qotishi mumkin** — pul oladigan kassa uchun
  yaramaydi. SSD'da fsync 1–5 ms.
- ⚠️ **Chop etish**: printerdan qaytish aloqasi yo'q. Qayta chiqarsak oshxona
  ikki marta pishiradi; chiqarmasak mehmon ovqatini **umuman olmaydi** —
  ikkinchisi yomonroq va tuzatib bo'lmaydi. Shuning uchun tasdiqlanmagan chop
  ishlari qayta chiqariladi, sarlavhasida **`TAKROR`**.
- ⚠️ **Soat — eng jimgina buziladigan joy.** Eski monoblokda CMOS batareyasi
  o'lgan bo'lsa sana nolga qaytadi va oflayn kassa **noto'g'ri sanali fiskal
  chek** yozadi. Qoida: vaqt hech qachon oxirgi yozilgan hodisadan orqaga
  ketmaydi; mos kelmasa sotish to'xtaydi.
- ⚠️ **Avtologon**: generator kelganda hech kim login qilmasligi kerak. Bu
  dasturiy emas, **o'rnatish** masalasi — lekin biz aytmasak hech kim qilmaydi.
- **UPS** (~40–60 $) qisqa uzilishlarni yashiradi, lekin dastur baribir
  chidamli bo'lishi shart: batareya ikki yildan keyin o'ladi va buni hech kim
  sezmaydi.
- Fiskal kassa ham o'chadi → qayta yonganda `#2D` → **allaqachon avtomatik
  ochiladi**, qo'shimcha ish yo'q.

⚠️ Va shundan chiroyli xulosa: **agar svet o'chishi xavfsiz bo'lsa, yopish
tugmasi ham xavfsiz** — u aynan o'sha yo'ldan o'tadi. Qo'rqinchli tasdiq oynasi
kerak emas.

### Ochiq savollar (qolgani)
Printer modellari va ulanish turi · POS uchun alohida narx · code signing
sertifikati.

---

## 2026-08-17 (tuzatish) — Oshxona chekini qayta chop etish: odam, dastur emas 📝

Oldingi yozuvdagi "tasdiqlanmagan chop ishlari avtomatik qayta chiqariladi va
`TAKROR` deb belgilanadi" degan qaror **rad etildi**.

⚠️ **Avtomatik qayta chop etish — taxmin.** Dastur chekning chiqqan-chiqmaganini
bilmaydi; pass oldidagi odam esa **biladi**, borib qaraydi. Kassir yoki
ofitsiant chekni qo'lda qayta yuboradi — hamma restoranda shunday ishlaydi.

⚠️ **Bu butun bir holat mashinasini olib tashlaydi**: chop etish tasdig'ini
kuzatish, qayta ishga tushganda tasdiqlanmaganlarni topish, ularni belgilash.
Chop etish "yubordim va unutdim" bo'ladi.

- **Qayta yuborishda izoh so'raladi va u chekda chiqadi.** "Svet o'chdi" degan
  matn umumiy `TAKROR` dan ko'ra oshpazga ko'proq narsa aytadi — izoh
  belgining **o'rnini bosadi**.
- ⚠️ **Void bilan bir toifa emas**: void pulni olib chiqadi, sababi
  javobgarlik uchun va qat'iy. Qayta chop etish faqat qog'oz sarflaydi, sababi
  **oshxona bilan muvofiqlashtirish** uchun → tayyor variantlar + erkin matn
  yetarli.
- **Bilinadigan xato bilinmaydiganidan ajratiladi**: printer o'chiq/qog'oz
  tugagani bilinadi → ekranda ko'rsatiladi; svet o'chib javob kelmagani
  bilinmaydi → dastur jim, qaror odamniki.

---

## 2026-08-17 — Buyurtmalar linzasi + kassir PIN ✅

POS rejasining 1- va 2-ishlari. Ikkalasi ham Windows ilovasidan mustaqil.

### 1. Zal cheklari buyurtmalar boardidan chiqdi
`applyTillLens` — standart holatda `check` bo'lgan buyurtmalar **ko'rsatilmaydi**;
`?till=only` faqat zal, `?till=1` ikkalasi. Band zal kuniga bir necha yuz chek
beradi va boardni o'qiydigan odam yetkazishni kuzatib, telefonga javob beryapti.

- ⚠️ **Ajratuvchi maydon `check`, `type == "dinein"` emas.** QR orqali mehmon
  o'z telefonidan bergan `dinein` buyurtma **shu boardga tushishi kerak** — uni
  kimdir qabul qilishi shart. Turi bo'yicha ajratsak, QR buyurtmalar jimgina
  yo'qolardi, va bu mehmonning shikoyati bo'lib chiqardi, xato hisoboti emas.
- ⚠️ **Noma'lum qiymat standartga qaytadi**, "hammasi"ga emas.
- Qoida **funksiyada** (`applyTillLens`), handler ichidagi `switch` emas — test
  o'sha koddan o'tadi. ⚠️ Birinchi yozgan testim switch'ni **takrorlagan** edi:
  u abadiy yashil bo'lib, qoida ostidan siljib ketardi.
- Panelda `Zal (kassa)` / `Hammasi + zal` chiplari, va ular **faqat restoran
  haqiqatan kassa ishlatsa** chiziladi ("hech qachon chek urilganmi").
  ⚠️ **Qidiruv ikkala yarimga ham kiradi**: chek raqamini yopishtirgan odam
  bitta buyurtmani qidiryapti va u qaysi yarimdan kelganini bilmaydi.

Statistika, hisobotlar va tushum tegilmadi.

### 2. Kassir PIN — javobgarlik tuzatildi
⚠️ **Nima buzuq edi**: monoblokda bitta hisob butun kecha ochiq turardi, ya'ni
har void, har chegirma va har yopilgan chek **soat oltida ekranni ochgan
odamga** yozilardi. Bu yozuvlar bitta savolga javob berish uchun bor, va login
modeli o'sha javobni jimgina buzib turgan edi.

**Model — ikki fakt, biri yolg'iz yetarli emas:**
- **Qurilma** qayerdaligini isbotlaydi (filial tokeni, bir marta login/parol
  bilan olingan).
- **PIN** kim turganini aytadi.

⚠️ **PIN parol emas va hech qachon parol sifatida ishlatilmaydi.** U qo'lda
topilishi mumkin; xavfsiz qiladigan narsa — u **faqat o'sha filialning tokenini
tutgan qurilmadan** qabul qilinishi. Egalik + bilim, karta bilan bir savdo.

- Token roli **`till`**, `staff` emas: zal va kassaga yetadi, o'sha odamning
  oyligiga, davomatiga va oshxona ekraniga **yetmaydi**. To'rt raqam bularning
  hech biriga yetmasligi kerak.
- ⚠️ **`auth.Generate` emas** — u 7 kun beradi. `GenerateLong` bilan **14 soat**:
  parol bilan olingan token va to'rt raqam bilan olingan token bir xil narxda
  bo'lmasligi kerak. Testda muhrlangan (smenadan uzun, haftadan qisqa).
- ⚠️ **Bekorchilikda avtoqulf (3 daqiqa)** — PINni ma'noli qiladigan narsa aynan
  shu. Busiz soat oltidagi bitta ochish butun kechani qoplaydi, ya'ni PIN
  tuzatishi kerak bo'lgan xato oldiga qo'yilgan yana bir ekran bo'lardi.
- ⚠️ **Nomzodlar to'plami tor** (`staffByPIN`): shu filial, ishlayotgan, kassaga
  ruxsati bor, PINi bor. bcrypt ataylab ~60 ms — butun kompaniyani skanerlash
  bir bosishni bir necha soniyaga aylantirardi. **Filial chegarasi ham shu
  yerda**: busiz boshqa filialdagi mos PIN bu restoranning voidini begona
  nom bilan imzolardi.
- ⚠️ **PIN filial ichida unique** (409). Ikki odamda bir xil kod — birinchi
  topilgani yutadi, ya'ni jurnal **noto'g'ri odamni nomlaydi**, va bu hech
  kimni nomlamaganidan yomonroq, chunki unga ishonishadi. Rad javobida
  **kimning PINi ekani aytilmaydi** — aks holda taxmin qilib chiqadigan admin
  qidiruv jadvaliga ega bo'lardi.
- ⚠️ **Oddiy kodlar rad etiladi** (`0000`, `1234`…): PIN mehmon va hamkasb
  oldida kuniga o'nlab marta teriladi, ya'ni hamma birinchi taxmin qiladigan
  kod — hamma ishlatadigan kod.
- ⚠️ **Blok filial bo'yicha** (5 urinish → 60 s), IP bo'yicha emas: bitta
  ulanish ortidagi restoran boshqa filial sinalgani uchun o'z kassirlarini
  qulflab qo'yardi. Muddat tugaganda **sanoq nolga tushadi**, aks holda
  keyingi bitta xato darhol qayta qulflardi.
- ⚠️ **Alohida endpoint** `PUT /admin/staff/{id}/pin` — `soldOut` va
  `kioskSecret` bilan bir naqsh: PINni ko'rsatmaydigan forma uni har saqlashda
  bo'sh yuborardi, ya'ni telefon raqamini tuzatish odamni kassadan qulflab
  qo'yardi.
- ⚠️ **PIN jurnalga yozilmaydi** — u tirik kalit, va sirlarni yozadigan audit
  izi o'g'irlash uchun ikkinchi joy.
- ⚠️ **Qulf ma'lumotdan kelib chiqadi, sozlamadan emas**: "shu filialda kimdir
  PIN oldimi". PIN tarqatmagan restoran avvalgidek ishlaydi — yangilanish
  jonli kassani smena o'rtasida hech kim aytmagan katakcha uchun qulflamasligi
  kerak.
- Lock ekrani **tor javob** qaytaradi (`tillPersonView`): ism va ikki ruxsat.
  Ortidagi hujjatda oylik, jadval va telefon bor, ekran esa ochiq xonada
  turadi. Testda maydonlar soni muhrlangan.
- Token **`sessionStorage`** da: ilova yopilsa qulflanadi. `localStorage`
  bo'lsa keyingi odamga oldingisining nomini berardi.

### Yozilgan fayllar
Backend: `handlers/tillpin.go`, `handlers/tillpin_test.go`,
`handlers/orderslens_test.go`, `handlers/admin.go` (`applyTillLens`),
`handlers/adminstaff.go`, `models/staff.go` (`PinHash`, `HasPin`,
`WithPinFlag`), `router.go` (guruh ikkiga bo'lindi: `staff` va `staff|till`).

Frontend: `app/kassa/PinPad.tsx`, `app/kassa/page.tsx` (qulf + avtoqulf),
`app/admin/staff/page.tsx` (`PinField`), `app/admin/orders/page.tsx` (linza),
`lib/api.ts` (`tillBearer`, ikki token), `lib/types.ts`, `lib/i18n/admin.ts`.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ · `tsc --noEmit` ✓ ·
`npm run build` ✓ · yangi marshrutlar tokensiz **401** ✓
Yangi testlar: linzaning standarti va noma'lum qiymati, `check` bo'yicha
ajratish (turi bo'yicha emas), oddiy PINlarning rad etilishi, PIN shakli,
blokning filial bo'yicha bo'lishi va muddat tugaganda nolga tushishi, till
tokenining staff tokenidan qisqaligi, lock ekrani javobining torligi.

### Keyingi qadam
3. Kassa UI monoblokka: plitka, rasm/rang, katta nishonlar.
4. Chek dizayni (uchta shablon).
5. Windows ilova (Wails) + printer.

---

## 2026-08-17 — Kassa ekrani monoblokka qayta chizildi ✅

POS rejasining 3-ishi. Muhit: 15", odatda **1024×768**, zaif protsessor,
minimal 4 GB RAM.

### ⚠️ Kategoriyalar endi yon tomonga surilmaydi
Gorizontal surilib ketadigan chip tasmasi sensorli ekran uchun noto'g'ri
boshqaruv: aylantirgichi ham, g'ildiragi ham yo'q, ya'ni chetdan chiqib ketgan
kategoriya foydalanuvchi uchun **umuman mavjud emas**, ko'ringanlari esa
qimirlab turadigan nishon. Endi ular **o'raladi** va hammasi ekranda.

### Rang — bezak emas, topish usuli
`lib/tillColors.ts`. Mehmon bilan gaplashib turgan kassir ikki yuz nomni
o'qimaydi — ichimliklar doim turadigan burchakka qo'l cho'zadi. Kategoriya
rangi bir xil to'rtburchaklar to'rini **hududlari bor joyga** aylantiradi, va
aynan shu sabab rangli kassa rasmlisidan tez bo'lishi mumkin.

- ⚠️ **Kategoriya id'sidan hisoblanadi, saqlanmaydi.** Menyuni bir kechada
  kiritgan odam ustiga yana rang tanlamaydi; taqsimot qurilmalar va qayta
  yuklashlar orasida barqaror; kategoriyani qayta nomlagan restoran xodimlari
  o'rgangan rangni yo'qotmaydi.
- ⚠️ **Qizil va yashil hech qachon ikki kategoriyaning yagona farqi emas** —
  deuteranopiya taxminan har o'n ikki erkakdan bittasida, restoranda esa bu
  har o'n ikki kassirdan bittasi. Palitra ko'k/sariq/binafsha/jigarrangga
  tayanadi.
- ⚠️ **Fon — past alfa tint, to'q to'ldirish emas.** Bitta to'liq rang light
  temada qora matn ostida ham, dark temada oq matn ostida ham o'qilishi kerak
  bo'lardi — buni hech qanday qiymat uddalamaydi. Tint `--surface` ustiga
  tushadi va tema bilan birga o'zgaradi; to'liq rang esa **yon chiziqda**,
  ustiga hech nima yozilmaydigan joyda. `--line` tokenidagi bilan bir qoida.
- ⚠️ **Dinamik Tailwind klass emas, inline `style`**: hisoblangan klass nomi
  purge bo'lib ketadi va rang jonli buildda yo'qoladi.

### Rasmlar — **qurilma sozlamasi**, kompaniyaniki emas
Yoqish/o'chirish kassa sarlavhasida, `localStorage` da. ⚠️ Rasm yorug' 15"
panelda yaxshi protsessor bilan yordam beradi va yonidagi 4 GB monoblokda
zarar qiladi — bu **mashina haqidagi fakt**, restoran haqidagi emas. Zaif
kassa ularni o'chiradi va qo'shni filialda hech nima o'zgarmaydi.

Kompaniya darajasidagi standart **chek dizayni sozlamalari bilan birga**
(4-ish) keladi — hozir yarim ulangan sozlama maydonini jo'natmaslik uchun.

- Rasm `?w=300` — server ruxsat bergan **eng kichik** o'lcham, plitka ~180 px.
  Aslini so'rash 4 GB monoblokka 4 MB telefon suratini qirq marta qo'yardi.
- `loading="lazy"` + qat'iy balandlik: brauzer faqat ekrandagini dekod qiladi
  va portret surat plitkani cho'zmaydi.

### ⚠️ Qidiruv natijasi cheklangan (60 ta)
Ikki harf butun menyuni topishi mumkin, ikki yuz plitkani rasm bilan chizish
esa bu ekranni **ko'rinadigan tarzda qotiradigan** yagona narsa. Chegara
saxiy va unga deyarli yetilmaydi, lekin yetilganda ekran **shunday deydi** —
qidirilgan taomni jimgina yashiradigan ro'yxat sekin ro'yxatdan yomonroq.

### Qolgan o'lchamlar
- Sarlavha **48 px** — 768 px balandlikda har qator dish gridiga tegmagan
  qator, va sarlavha ovqat sotmaydi.
- Plitka **min 88 px** — barmoq va yarim, chunki bosayotgan odam qo'liga
  qaramaydi.
- Chek ro'yxatida **stol raqami katta** (`font-display text-lg`): qatordagi
  qolgan hamma narsa kontekst, kassir esa uni bir qadam naridan o'qiydi.
- Uch ustun: cheklar 224 px · menyu (qolgani) · chek 320 px.

### Yozilgan fayllar
`lib/tillColors.ts`, `app/kassa/MenuGrid.tsx`, `app/kassa/page.tsx`,
`lib/i18n/admin.ts`.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ · `tsc --noEmit` ✓ ·
`npm run build` ✓ (`/kassa` 11.8 kB).

⚠️ **Ko'z bilan tekshirilmadi**: `/kassa` ishchi logini ortida, bazadagi yagona
hisob `mara` va uning paroli menda yo'q. 1024×768 da render qilib ko'rish
uchun parol kerak.

### Keyingi qadam
4. Chek dizayni (uchta shablon) + rasm ko'rsatishning kompaniya standarti.
5. Windows ilova (Wails) + printer.

---

## 2026-08-17 — Kassa faqat PIN bilan: qurilma tokeni ✅

Uch tuzatish, uchalasi ham jonli sinovdan keyin.

### 1. PIN — aniq 4 raqam
Pad oltita nuqta chizardi, ikkitasi ortiqcha edi. ⚠️ **Uzunlik diapazon emas,
qat'iy**: o'zgaruvchan uzunlikda pad nechta raqam kutilayotganini ko'rsata
olmaydi va o'zi yubora olmaydi — ikkalasi ham binodagi eng band ekranda
qo'shimcha bosish. To'rtinchi raqamda **o'zi yuboriladi**, tasdiq tugmasi
olib tashlandi (u hech qachon bosilishi to'g'ri bo'lmaydigan boshqaruv edi).

### 2. ⚠️ Oddiy kodlar endi ruxsat etiladi
Oldingi versiya `1234` va `0000` ni rad etardi. **Bu bu ekran uchun noto'g'ri
savdo edi**: PIN smenada o'nlab marta, likop ko'targan odam tomonidan
teriladi, va eslab qololmaydigan kod monoblok yoniga yopishtirilgan qog'ozga
aylanadi — xonadagi hamma o'qiy oladigan va hech qachon o'zgarmaydigan kod,
ya'ni taxmin qilinadiganidan **yomonroq**.

Himoyani kodning murakkabligi bermaydi: u faqat filial tokenini tutgan
qurilmadan qabul qilinadi, besh xato bir daqiqa turadi, va u zal bilan
kassadan boshqa hech qayerga yetmaydi.

⚠️ **Filial ichidagi unikallik qoldi** — bu "murakkablik cheklovi" emas,
to'g'rilik sharti: ikki odamda bir xil kod bo'lsa jurnal noto'g'ri odamni
nomlaydi.

### 3. Qurilma tokeni — kassada login yo'q
⚠️ **Monoblok filialga bog'lanadi, unga odam kirmaydi.** Ikki mehmon orasida
login va parol teriladigan ekran — bir hisobni hamma bilan bo'lishib, uni
devorga yozib qo'yadigan restoran. iiko ham shunday ishlaydi.

- `Branch.TillVersion` + `GET /admin/branches/{id}/till-token` → `tilldevice`
  rolli, **1 yillik** token. Kiosk ekranidagi naqshning aynan o'zi.
- Panelda **Sozlamalar → Filial → Kassa qurilmasi**: havola + QR (monoblokda
  klaviatura yo'q, panel esa boshqa mashinada — havola kamera orqali ketadi).
- `/kassa?t=...` tokenni oladi va **manzil satridan darhol tozalaydi**: URL'dagi
  bir yillik token brauzer tarixiga, skrinshotga va xatcho'plarga tushadi.
- Token `localStorage` da (odamning tokeni esa `sessionStorage` da): bu sessiya
  emas, "bu mashina shu restoranniki". Har restartda qayta bog'lanadigan kassa —
  har ertalab qo'ng'iroq.
- ⚠️ **Almashtirish filialdagi BARCHA kassalarni o'chiradi** — tokenlar qurilma
  kimligini tashimaydi, ya'ni undan mayda bekor qilish yo'q. Tasdiq oynasida
  shu yozilgan, chunki tuzatish — har monoblokka yangi havola bilan borish.
- ⚠️ **Eski loginli kassalar ishlayveradi** (`tillBranch` ikkala tokenni ham
  qabul qiladi): deployda ularni sindirish restoranni smena o'rtasida hech kim
  aytmagan o'rnatish qadami uchun to'xtatardi.
- ⚠️ **Ruxsatlar qulfni ochgan odamdan olinadi, quridmadan emas** — monoblokda
  o'z ruxsati yo'q, va "kirgan hisob"dan o'qish aynan PIN tugatgan chalkashlik.
- Bog'langan qurilmada tugma **"Chiqish" emas, "Qulflash"**: chiqadigan hisob
  yo'q, qurilma tokenini tozalash esa sotish uchun paneldan yangi havola
  talab qilardi.

### ⚠️ Yo'lda topilgan tuzoq: `next build` va `next dev` bir `.next` ni bo'lishadi
Dev server ishlab turganda `npm run build` ishga tushirilgan edi — build dev
artefaktlarini bosib ketdi va **hamma sahifa 500** qaytardi
(`Cannot read properties of undefined (reading '/_app')`). Kod aybdor emas edi.
Tuzatish: dev serverni to'xtatish, `.next` ni o'chirish, qayta ishga tushirish.

### Yozilgan fayllar
Backend: `handlers/tillpin.go` (`AdminTillToken`, `tillDeviceBranch`,
`tillBranch`, `pinDigits`), `models/models.go` (`Branch.TillVersion`),
`router.go` (yangi guruh: `staff` yoki `tilldevice`).

Frontend: `components/admin/TillDeviceSettings.tsx`,
`components/admin/BranchesEditor.tsx`, `app/kassa/{page,PinPad}.tsx`,
`app/admin/staff/page.tsx`, `lib/api.ts`, `lib/i18n/admin.ts`.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ · `tsc --noEmit` ✓ ·
`npm run build` ✓ · dev serverlarda `/`, `/kassa`, `/admin/*` → **200** ✓

### Ochiq savol
"PIN bilan kiradi **va smena ochadi yoki yopadi**" — qaysi smena nazarda
tutilgan? Ishchining davomat smenasi (`/staff/clock`, hozir GPS bilan) yoki
kassa smenasi (`cash_shift`, hozir faqat paneldan)? Ikkalasi ham kassadan
ochilishi mumkin, lekin qoidalari boshqa — birinchisi oylikka ta'sir qiladi.

---

## 2026-08-17 — Ruxsatlar va rollar: model, migratsiya, override ✅ (yarim)

Kelishilgan model `docs/pos-reja.md` §10 da. Backend yadrosi tayyor; panel UI va
kassa ekranidagi dialog qolgan.

### Oltita ruxsat, rol orqali beriladi
`waiter · cashier · void · discount · shift · kitchen`. `staff_role`
kolleksiyasi: nom + ruxsatlar to'plami, xodimga **rol** beriladi.

⚠️ **Rol lavozimning o'rnini bosadi, lekin CLAUDE.md dagi "lavozim ruxsat emas"
ogohlantirishi kuchda qoladi** — va aynan shu sabab rol **id'si bor yozuv**,
terilgan matn emas. "Oshpaz" deb yozgan odamga kalit berilmaydi; ro'yxatdan
tanlangan rolga beriladi.

11 ta tayyor rol: Ish boshqaruvchi · Menejer · Zal administratori · Kassir ·
Barmen · Ofitsiant · Xostes · Oshxona boshlig'i · Oshpaz · Texnolog · Yordamchi
xodim. Hammasi **tahrirlanadi va o'chiriladi** — o'zgartirib bo'lmaydigan rol
noto'g'ri rol berish orqali aylanib o'tiladi.

⚠️ **Ish boshqaruvchi va Menejer ruxsati bir xil, ataylab**: jurnalda boshqacha
nomlanadi, va keyin alohida toraytirilishi mumkin. Bugun bir xil — abadiy shart
emas.

### ⚠️ Migratsiya: hech kim mavjud huquqini yo'qotmaydi
Yangi installda **Kassir `void` va `discount` siz** (siz aytganingizdek). Lekin
bugungi `canCashier` odam hozir ham void va chegirma qila oladi — shuning uchun
migratsiya ularni **"Zal administratori"** ga ko'chiradi, "Kassir" ga emas.

Bu birinchi qarashda xato ko'rinadi, keyin ruxsatlarni o'qiganda to'g'ri
bo'ladi: bugungi kassir — bu kod endi zal administratori deb ataydigan narsa.
Ularni "Kassir" deb nomlash tartibliroq bo'lardi va **ikkita qobiliyatni
jimgina olib tashlardi** (`EnsureKitchenAccess` bilan bir dars).

⚠️ Migratsiya **ikki yarmida ham idempotent**: rollar faqat bo'sh kolleksiyaga
seed qilinadi (aks holda "Ofitsiant"ni qayta nomlagan restoran har restartda
ikkinchisini olardi), xodimlar esa faqat roli yo'q bo'lsa biriktiriladi.

### ⚠️ Legacy hisob yangi ruxsatlarni **olmaydi**
`void` va `discount` rollargacha mavjud emas edi, ya'ni "ha" degan eski bayroq
yo'q. Migratsiya qilinmagan hisob **rad etiladi**, taxmin qilinmaydi — "ha" deb
taxmin qilish migratsiyani ishlatmagan har restorandagi har ofitsiantga
pishirilgan taomni chekdan olib tashlash imkonini berardi. Testda muhrlangan.

### ⚠️ Menejer tasdig'i (override) — tizimni ishlaydigan qiladigan qism
Ruxsati yo'q amal **rad etilmaydi**. Ekran ruxsati bor odamdan PIN so'raydi, va
**ikkala nom ham yoziladi**: *"Aziz olib tashladi, Dilnoza tasdiqladi"*.

- ⚠️ **Rad etish nima uchun noto'g'ri javob**: ofitsiant menejerni chaqirmaydi —
  bir hafta ichida menejerning PIN kodi butun zalga ma'lum bo'ladi, va shundan
  keyin har void bitta nomni tashiydi. Ya'ni qat'iy ruxsat o'zi tuzatishi kerak
  bo'lgan narsani buzadi.
- ⚠️ **Qiluvchi qiluvchi bo'lib qoladi**: menejerni "qilgan" deb yozish yagona
  saqlashga arziydigan faktni yo'qotadi va har hisobdan chiqarishda uni
  ayblardi.
- ⚠️ **409, 403 emas**: 403 yakuniy javob va ekranlar uni shunday o'qiydi; bu
  esa ikkinchi odam so'rovi, va ekran unga javob bera olishi kerak.
- ⚠️ **Bir xil filial** (`staffByPIN` filialga bog'langan): aks holda zanjirning
  boshqa shahridagi menejerning kodi shu yerdagi hisobdan chiqarishni
  tasdiqlardi.
- ⚠️ Rad javobi "PIN noto'g'ri" emas, "ruxsat kerak": kod to'g'ri bo'lib, o'sha
  odam ham qila olmasligi mumkin, va "noto'g'ri" deyish uni boshqa PIN sinashga
  yuboradi.

`void` va `discount` shu mexanizmga ulandi. `CheckLineVoid` ga `authBy`
qo'shildi; chegirmada tasdiqlovchi nomi chek yorlig'iga kiradi.

### Yozilgan fayllar
`models/staffrole.go`, `models/staffrole_test.go`, `models/staff.go`
(`RoleID`, `Perms`, `Can` qayta yozildi), `models/check.go` (`AuthBy`),
`handlers/tilloverride.go`, `handlers/staff.go` (`withRole`, `withRoles`),
`handlers/tilllines.go`, `handlers/tillclose.go`,
`repository/{store,migrate}.go`, `cmd/server/main.go`.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ · `tsc --noEmit` ✓

### Qolgan ish (shu yo'nalishda)
- Rollar CRUD API + paneldagi rollar bo'limi
- Ishchi yaratishda rol tanlash (hozir `position` erkin matn)
- Kassa ekranida override dialogi (PIN so'rash)
- `shift` ruxsatini kassa smenasi va Z-hisobotga ulash

---

## 2026-08-17 — Rollar CRUD va panel ✅

### `/admin/roles` — rollar va ruxsatlar
Jadval: rol nomi · ruxsatlari · **nechta ishchi tutadi** · tahrirlash/o'chirish.

⚠️ **Ishchilar soni ataylab ko'rsatiladi**: o'n bir kishi tutgan "Ofitsiant"ni
kengaytirish — hech kim tutmaganini kengaytirishdan boshqa amal, va bu raqamsiz
ega buni bilmaydi.

⚠️ **Ruxsati yo'q rol bo'sh katak emas, aytiladi** ("Kassa ruxsatlari yo'q"):
xostesning hisobi bor va kassada ishi yo'q — bu haqiqiy javob, bo'sh katak esa
yuklanmagan qator bo'lib o'qiladi.

⚠️ **Override haqidagi jumla tahrirlash oynasining ichida**, chunki aynan u
qat'iy rollarni ishlaydigan qiladi — va buni bilmagan ega hammaga hamma narsani
beradi.

### Kirish qoidasi
⚠️ **O'qish — ishchilarni boshqara oladigan har kimga; yozish — faqat egaga.**
Rolni kengaytira oladigan menejer **o'zinikini** kengaytira oladi, va bu
smena darajasidagi qaror emas. Lekin rol biriktiradigan har ekran ro'yxatni
o'qiy olishi shart.

⚠️ **Rollar kompaniya darajasida, filialga bog'lanmagan**: bir oshxonada taom
hisobdan chiqara oladigan, ikkinchisida yo'q "Kassir" — hech kim boshida
ushlab turolmaydigan qoida, va u eng avval filiallar orasida yuradigan
menejerni chalkashtiradi.

### Ikki qo'riqchi
- ⚠️ **Ishchilar tutgan rol o'chirilmaydi** (409, sababi bilan). Roli yo'qolgan
  hisob legacy bayroqlarga qaytadi (`withRole`), ya'ni ko'pchilik uchun smena
  o'rtasida kassani yo'qotish — va planshet sababini aytmaydi. Avval qayta
  biriktirish — hech kimni ajablantirmaydigan yagona tartib.
- ⚠️ **Notanish ruxsat saqlanmaydi, tashlanadi**: `"superuser"` yuborgan mijoz
  bazada keyingi versiya ma'no berishi mumkin bo'lgan so'z qoldirmasligi kerak.

### Ishchi formasida rol
Rol tanlash **lavozim maydonidan yuqorida**, va tagida bir jumla: ⚠️ *ruxsatlar
roldan olinadi, lavozim shunchaki izoh va tizim uni o'qimaydi*. Ikki maydon
bir-biriga o'xshaydi va faqat bittasi kassani ochadi — CLAUDE.md dagi
ogohlantirish endi ekranda ham turadi.

`roleId` — **pointer**: yubormagan eski mijoz saqlangan rolni o'chirmaydi
(`isActive` bilan bir qoida). Server id'ni **tekshiradi**: mavjud bo'lmagan rol
odamni legacy bayroqlarga qaytarardi.

### Jonli tekshiruv
Migratsiya ishga tushdi va o'zini to'g'ri tutdi:

```
rollar: 11
mara: canCashier=true → rol "Zal administratori" [waiter cashier void discount shift]
```

⚠️ Ya'ni mavjud kassir **void va chegirmani saqlab qoldi**, "Kassir" roliga
(unda ular yo'q) tushmadi — aynan mo'ljallangan xatti-harakat.

### Yozilgan fayllar
Backend: `handlers/staffroles.go`, `handlers/adminstaff.go` (`roleId`,
`withRoles`), `router.go` (4 marshrut).

Frontend: `app/admin/roles/page.tsx`, `app/admin/staff/page.tsx` (rol
tanlagich), `app/admin/layout.tsx` (nav, owner-only), `lib/{types,api}.ts`,
`lib/i18n/admin.ts`.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ · `tsc --noEmit` ✓ ·
`/admin/roles` **200** ✓ · `/admin/roles` API tokensiz **401** ✓

### Qolgan ish
- Kassa ekranida override dialogi (PIN so'rash)
- `shift` ruxsatini kassa smenasi va Z-hisobotga ulash
- Kassa smenasi kassadan ochiladi/yopiladi
- Sozlamalarga alohida sidebar
- Chekni boshqa stolga ko'chirish

---

## 2026-08-17 — Kassa ekranida override dialogi ✅

Ruxsatlar tizimining oxirgi va eng muhim yarmi. Usiz server 409 qaytarardi va
ekran uni ko'rsata olmasdi.

### Refusal — javob emas, so'rov
⚠️ Ruxsati yo'q amal rad etilmaydi: PIN pad chiqadi, **ruxsati bor odam** o'z
kodini teradi, amal o'tadi va **ikkala nom ham yoziladi**.

Menejer zalda, ekran allaqachon ofitsiantning oldida. Rad etish esa uni
ekrandan uzoqlashtiradi va bir hafta ichida menejerning PIN kodi butun zalga
ma'lum bo'ladi — shundan keyin har void bitta nomni tashiydi, ya'ni ruxsat
tizimi o'zi himoya qilishi kerak bo'lgan narsani buzadi.

### ⚠️ Sabab rad javobidan omon qoladi
Void sababini (yoki chegirma summasini va sababini) qaytadan yozdirish — sabab
"." ga aylanishining yo'li, va sabab bu yozuvning butun ma'nosi. Kutayotgan
amal to'liq saqlanadi va menejer PIN terganda **aynan o'sha** qayta yuboriladi.

Chegirmada bu ayniqsa muhim: summani qayta terish band kassirning tasdiqlangan
chegirmadan **boshqasini** berishiga olib keladi.

### Boshqa qarorlar
- ⚠️ **Dialog ruxsatni nomlaydi, nosozlikni emas**: "ruxsat yo'q" ofitsiantga
  bajaradigan hech nima aytmaydi; "Chegirma berish — ruxsati bor xodim PIN
  kodini kiritsin" nima so'rashni aniq aytadi.
- ⚠️ **Xato PIN dialogni yopmaydi** — sababi yo'qolardi. Xabar "bu PIN bu
  amalni bajara olmaydi": kod to'g'ri bo'lib, o'sha odam ham qila olmasligi
  mumkin.
- Pad to'rtinchi raqamda **o'zi yuboradi**, qulf ekrani bilan bir xil: ikki pad
  boshqacha ishlasa, birini tez qiladigan mushak xotirasi ikkinchisini
  noto'g'ri qiladi.
- `ApiError` endi javob tanasini tashiydi (`needsOverride`, `permissionName`) —
  ba'zi rad javoblari **so'rov**, va ekran ularni ajrata olishi kerak.

### ⚠️ Testlar bazaga tegmaydigan yo'llarni tekshiradi
`Handler{}` — `Store` siz. Bazaga yetgan har yo'l panic qiladi, va **panic
o'zi tasdiq**: ruxsat bor-yo'qligini aniqlash uchun Mongo'ga borish binodagi
eng band tugmaga qo'shimcha so'rov qo'shardi.

Muhrlangan: ruxsati bor odamning amali **ikkinchi nomsiz** yoziladi (yo'q
tasdiqlovchini yozish menejerning nomini ko'rmagan ishiga qo'yardi), ruxsat
yo'qligi **so'raydi**, noto'g'ri shakldagi PIN bazaga bormaydi, ishdan
bo'shatilgan xodim **o'z vakolatida ham** ishlay olmaydi, va har ruxsatning
odam o'qiydigan nomi bor.

### Yozilgan fayllar
Frontend: `app/kassa/OverrideDialog.tsx`, `app/kassa/CheckPanel.tsx` (void),
`app/kassa/PayDialog.tsx` (chegirma), `lib/api.ts` (`ApiError.data`,
`needsOverride`, `permissionName`), `lib/i18n/admin.ts`.
Backend: `handlers/tilloverride_test.go`.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ · `tsc --noEmit` ✓ ·
`/kassa` va `/admin/roles` **200** ✓

### Qolgan ish
- `shift` ruxsatini kassa smenasi va Z-hisobotga ulash
- Kassa smenasi kassadan ochiladi/yopiladi
- Sozlamalarga alohida sidebar
- Chekni boshqa stolga ko'chirish

---

## 2026-08-17 — Kassa smenasi kassa ekranidan ✅

### Nima uchun bu yerda bo'lishi kerak edi
⚠️ Kassa oqshom oxirida uning oldida turgan odam tomonidan sanaladi. Shu paytgacha
buning yagona yo'li admin panel edi — ya'ni yo har kassirga panel logini berish
(mijozlar bazasi, to'lov kalitlari, hisobotlar), yo menejer ertasi kuni kelib
**boshqa odam bo'shatgan** kassani sanashi. Ikkalasi ham qochilayotgan narsadan
yomonroq.

### ⚠️ Arifmetika takrorlanmadi — ajratib olindi
`openShiftFor` va `closeShiftFor` `cash.go` da, panel bilan **umumiy**. Ikki
implementatsiya "kassada qancha bo'lishi kerak" degan savolga oxir-oqibat ikki
xil javob berardi, va o'shanda restoranda yo'qolgan pul haqida **ikkita javob**
bo'lardi, qaysi biri to'g'riligini aniqlash yo'lisiz.

Panel handlerlari ham shu funksiyalarga o'tkazildi — ya'ni bu nusxa emas,
ko'chirish.

### `shift` ruxsati va override
Ochish ham, yopish ham `shift` ruxsatini talab qiladi va yo'q bo'lsa
**menejerdan PIN so'raydi**. Ekranda kutayotgan amal saqlanadi: ⚠️ sanoqni
qayta terish kassaning **hech kim tasdiqlamagan** summada yopilishiga olib
keladi.

⚠️ Smenaga yoziladigan nom — `"Aziz (Dilnoza)"`: `CashShift` ochgan/yopgan
odamni matn sifatida saqlaydi, va halol javob "kassir, menejerning ruxsati
bilan". Faqat menejerni yozish uni ko'rmagan sanoq uchun mas'ul qilardi; faqat
kassirni yozish ruxsat kerak bo'lganini yashirardi.

### Ekrandagi qoidalar
- ⚠️ **Kutilgan summa raqam kiritilgunicha ko'rsatilmaydi** — panel ekranidagi
  bilan bir qoida: bo'sh maydon yonidagi raqam ko'chirib yozishga taklif, va
  ekrandan o'qilgani uchun mos kelgan sanoq — bu narsaning butun ma'nosini
  yo'q qiladi.
- **Yig'ilgan holatda ham bitta raqam ko'rinadi**: hozir kassada qancha
  bo'lishi kerak.
- ⚠️ **"Kuryerlar qo'lida" ro'yxatdan tashqarida** — u jamiga kirmaydi, qo'shish
  har yetkazishni ikkilantirardi.
- ⚠️ **Kassa smenasi paneli Z-hisobot tugmasidan yuqorida**: kassa avval
  sanaladi, soliq kuni keyin yopiladi — va kun yopish yuborilmagan cheklar
  bo'lsa rad etiladi. Teskari tartibda kassir rad javobini uni keltirib
  chiqaradigan amaldan **oldin** ko'rardi.
- Smena yopilganda fiskal kunni yopish so'raladi (paneldagi bilan bir xil), va
  ⚠️ **bu yerdan bajarilishi ham osonroq** — bu ekran aynan restoran tarmog'ida
  turgan ekran.

### Yozilgan fayllar
Backend: `handlers/tillcash.go`, `handlers/cash.go` (`openShiftFor`,
`closeShiftFor` ajratildi; panel handlerlari ularga o'tkazildi), `router.go`.

Frontend: `app/kassa/CashShiftPanel.tsx`, `app/kassa/page.tsx`, `lib/api.ts`,
`lib/i18n/admin.ts`.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ · `tsc --noEmit` ✓ ·
`/staff/cash-shift` tokensiz **401** ✓ · `/kassa` **200** ✓

### Qolgan ish
- Sozlamalarga alohida sidebar
- Chekni boshqa stolga ko'chirish

---

## 2026-08-17 — Sozlamalar tablari va chekni stolga ko'chirish ✅

### Sozlamalar: 22 bo'lim → 6 tab
`Restoran · Sayt · Zal va buyurtma · Yetkazish · To'lov va kassa ·
Integratsiyalar`.

⚠️ **Ko'rinmaydigan bo'lim o'chirilmaydi, `hidden` bilan yashiriladi.** Bu
yerdagi bir nechta bo'lim — chizilayotgan stol xaritasi, tahrirlanayotgan
filial, yarim chizilgan yetkazish zonalari — **o'z holatini tutadigan
komponentlar**. Tab almashtirish uchun ularni unmount qilish o'sha ishni tashlab
yuborardi, va ega buni qaytib kelganda bilardi. Narxi: hammasi baribir bir marta
render bo'ladi — ya'ni tablardan **oldingi** holat, regressiya emas.

⚠️ **Tab URL'da emas.** Sahifa bitta saqlanmagan qoralamani tutadi, va tablar
orasida yuradigan "orqaga" tugmasi sahifalar orasida yurayotgandek ko'rinardi —
keyin kimdir uni tahrirlari saqlanib qolgan deb o'ylab bosardi.

⚠️ **Alohida marshrutlar emas**: bo'limlar bitta qoralama va bitta Saqlash
tugmasini bo'lishadi. Ularni marshrutlarga bo'lish yo har marshrutga alohida
saqlash (yarim to'ldirilgan formani yo'qotishning olti xil yo'li), yo
qoralamani ular orasida tashish — ya'ni o'sha sahifa, ustiga qo'shimcha
qadamlar bilan.

Telefonda chiplarga o'raladi: bu ekran ish stolidan qanchalik ko'p ochilsa,
oshxonadan ham shunchalik ochiladi.

### Chekni boshqa stolga ko'chirish
Backend allaqachon qo'llab-quvvatlagan (`StaffUpdateCheck`), UI yo'q edi.

- ⚠️ **Ruxsat ham, sabab ham so'ralmaydi.** To'rtinchi stoldan oltinchisiga
  ko'chirish chekdan hech nima olmaydi va oshxonadan hech nima olib qo'ymaydi —
  u xona haqidagi faktni to'g'irlaydi. Qo'riqlash ofitsiant bilan odamlarni
  o'tqazishning oddiy ishi orasiga menejerni qo'yardi, va ruxsatlar aynan
  shunday o'chirib qo'yiladi.
- ⚠️ **Band stollar ko'rsatiladi va bosilmaydi**, yashirilmaydi: 7-stolni topa
  olmagan ofitsiant ekranni xato deb hisoblaydi va keyingi qiladigan ishi —
  o'sha mehmonga **ikkinchi chek** ochish.
- ⚠️ **"Stolsiz" varianti qoladi**: peshtaxtaga ko'chgan mehmon ham, boshidan
  noto'g'ri stolga ochilgan chek ham shuni talab qiladi — faqat yon tomonga
  ko'chira oladigan oyna o'zini ochishga sabab bo'lgan xatoni tuzata olmaydi.

### Yozilgan fayllar
Frontend: `app/admin/settings/page.tsx` (`GroupContext`, rail, 22 bo'lim
teglandi), `app/kassa/MoveTableDialog.tsx`, `app/kassa/CheckPanel.tsx`,
`app/kassa/page.tsx`, `lib/i18n/admin.ts`.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ · `tsc --noEmit` ✓ ·
`/kassa`, `/admin/settings`, `/admin/roles`, `/admin/cash` → **200** ✓

### POS rejasidagi holat
Bajarildi: 1 buyurtmalar linzasi · 2 kassir PIN · 3 kassa UI monoblokka ·
ruxsatlar va rollar · override · kassa smenasi kassadan · sozlamalar tablari ·
stolga ko'chirish.

Qolgan (rejadagi tartib bo'yicha): **4 chek dizayni (uchta shablon)** ·
**5 Windows ilova (Wails) + printer** · **6 oflayn** · **7 ofitsiant ekrani**.

---

## 2026-08-17 — Chek dizayni: uchta shablon va jonli ko'rinish ✅

POS rejasining 4-ishi. Printerdan **oldin** kerak edi, chunki 5-ish (Windows
ilova) uni tayyor holda topishi kerak.

### ⚠️ Chek — sahifa emas, belgilar to'ri
Termal printer matnni joylashtirmaydi: u qatorda qat'iy sondagi monoshirift
belgi chiqaradi va kesadi. **58 mm — 32 belgi, 80 mm — 48.** Bu yerdagi hamma
narsa shu to'r ustidagi arifmetika, va chekni buzishning eng keng tarqalgan
yo'li — 58 mm printerga 48 belgi yuborish: har qatorning oxiri yo'qoladi, ya'ni
mijoz nusxasida **jami summa ustuni**.

### ⚠️ Ko'rinish serverda chiziladi
Paneldagi preview va printerdan chiqadigan qog'oz — **bitta funksiya**
(`internal/receipt`). Brauzerda chizilgan preview o'sha belgilar to'rining
ikkinchi implementatsiyasi bo'lardi, va ular ajrab ketardi — farqni esa cheki
o'zi dizayn qilgan narsaga o'xshamaydigan restoran topardi. Hisobotlardagi
bilan bir qoida: bitta hisob, ikkita chiqish.

### ⚠️ Uchta chek — uchta o'quvchi, bitta shablon emas
- **Oshxona cheki** — pass'da, uch chek kutib turganda o'qiladi. **Narx yo'q, va
  uni yoqadigan sozlama ham yo'q**: narx oshpazga hech nima aytmaydi va chekni
  uzaytiradi. Stol raqami eng katta narsa, chunki to'g'ri bo'lishi shart bo'lgan
  yagona narsa u. Taom izohi (`piyozsiz`) **hech qachon kesilmaydi**.
- **Kassa cheki** — pul va uni kim olgani.
- **Mijoz cheki** — **yagona** fiskal belgi tashiydigan nusxa: uni tekshiradigan
  odam faqat mehmon, va pass'ga chiqarish qog'ozni undan foydalana olmaydigan
  odamga sarflash.

### Bu yerda buziladigan narsalar (testda muhrlangan)
- ⚠️ **Belgilar sanaladi, baytlar emas.** To'g'ri yozilgan o'zbekchada `oʻ` va
  `gʻ`, ruschada kirill — ikkalasi ham UTF-8 da ko'p baytli. `len()` bilan
  hisoblangan kenglik har bunday qatorni **harflar soniga teng** qisqartiradi,
  va alomat — pastga qarab ustunlarning o'ngga siljib borishi. SMS bo'lak
  sanog'idagi bilan bir arifmetika.
- ⚠️ **To'qnashuvda o'ng ustun yutadi**: o'ngda pul turadi, va jamiga kirib
  ketgan yorliq qisqa so'z emas, **noto'g'ri son** beradi.
- ⚠️ **Uzun taom nomi o'raladi, kesilmaydi**: 32 belgida kesilgan
  "Lag'mon (achchiq, katta porsiya)" — boshqa taom, va oshxona chekda
  yozilganini pishiradi.
- ⚠️ **Pul qo'lda guruhlanadi, `Intl` bilan emas**: bir restoran ba'zi cheklarda
  vergul, ba'zilarida bo'shliq chiqarmasligi kerak — shablon saqlanganda panel
  qaysi tilda bo'lganiga qarab. Saytdagi `formatPrice` bilan bir sabab.
- ⚠️ **Yo'q maydon — ko'rsatiladi.** Maydon paydo bo'lishidan oldin saqlangan
  har shablonda yozuv yo'q, va uni "o'chiq" deb o'qish yaxshi chop etilayotgan
  qatorni jimgina yo'qotardi.
- ⚠️ **Standart — hammasi yoqilgan, 80 mm.** Hujjatning nol qiymati "hech nima
  chop etilmasin", va bu endigina printer ulagan restoran uchun **buzuq
  printerdan farq qilmaydi**.

### Sozlama filialda
`receipt_settings`, filial bo'yicha — ⚠️ **chunki printer o'sha yerda**. Qog'oz
kengligi o'sha peshtaxtadagi mashina haqidagi fakt, va 58 mm rulon olgan
ikkinchi oshxona birinchisida dizayn tahrirlangan har safar yarim chek chop
etardi.

Restoran nomi, manzili, telefoni **bu yerda emas** — ular brend va filialda
allaqachon bor, ikkinchi nusxa esa telefon o'zgarganda yangilanadigan ikkinchi
joy bo'lardi. Shablon faqat ular chop etiladimi-yo'qmi degan savolni hal qiladi.

### Preview namunasi ataylab noqulay
Uzun taom nomi (o'ralishi kerak), izoh, chegirma va qaytim. ⚠️ Sarlavhada
restoranning **haqiqiy** nomi va manzili: "Restoran" deb yozilgan preview egaga
o'z nomi 58 mm ga sig'adimi degan savolga javob bermaydi.

### Jonli tekshiruv (58 mm, haqiqiy render)
```
|          OSH MARKAZI|
|--------------------------------|
|#MRC-A1-1745              7-stol|
|Qaymoqli achchiq lag'mon, katta|
|  porsiya|
|  2 × 45 000 so'm    90 000 so'm|
|JAMI                 90 000 so'm|
```

### Yozilgan fayllar
Backend: `internal/receipt/{receipt,block,receipt_test}.go`,
`models/receipt.go`, `handlers/receipts.go`, `repository/store.go`, `router.go`.

Frontend: `components/admin/ReceiptEditor.tsx`, `app/admin/settings/page.tsx`,
`lib/{types,api}.ts`, `lib/i18n/admin.ts`.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ · `tsc --noEmit` ✓ ·
`/admin/receipts` tokensiz **401** ✓ · `/admin/settings` **200** ✓
Yangi testlar: hech nima qog'ozdan oshmasligi (58 va 80), kirill va apostrof
qatorni qisqartirmasligi, oshxona chekida pul yo'qligi, fiskal belgi faqat
mijoz nusxasida, yo'q maydon ko'rsatilishi, uzun nom o'ralishi, summaning
kesilmasligi, pulning har safar bir xil guruhlanishi.

### Qolgan ish
5. Windows ilova (Wails) + printer — ⚠️ printer modellari hali noma'lum
6. Oflayn
7. Ofitsiant ekrani

---

## 2026-08-17 — Ofitsiant ekrani `/zal` ✅

POS rejasining 7-ishi. Printer va Multikassa javoblariga bog'liq emas.

### ⚠️ Umumiy komponentlar ajratildi
`MenuGrid · TablesScreen · PinPad · OverrideDialog · VoidDialog ·
MoveTableDialog · NewCheckDialog` → `components/till/`.

Reja (§3) buni talab qilgan edi: **ofitsiant ekrani kassadan mustaqil
komponentlar bilan qurilishi kerak**, aks holda mobilga ko'chirishda kassa
mantiqini ham tortib ketardi. `/zal` ning `app/kassa/` dan import qilishi aynan
o'sha bog'lanishni yaratardi.

### ⚠️ Bu kassaning to'lovsiz nusxasi emas
Ikki ekran boshqa savolga javob beradi va **boshqa qo'lda** turadi: kassa —
peshtaxtadagi monoblok, sotuvni **tugatadi**; bu — stollar orasida yuriladigan
planshet, sotuvni **boshlaydi**. Bitta komponentni bo'lishish kassaga
kiritilgan har o'zgarishni hech kim o'ylamayotgan ekranga qarab tekshirishni
talab qilardi — va aynan shu ekran telefonga ko'chadi.

### Ekrandagi qarorlar
- ⚠️ **"Oshxonaga yuborish" — rangini o'zgartiradigan yagona boshqaruv**, va
  faqat yuboriladigan narsa bo'lganda. Ofitsiantning yagona haqiqiy nosozligi —
  buyurtmani terib, yubormasdan ketish, va buni mehmon **yigirma daqiqadan
  keyin** biladi.
- ⚠️ **Stol ochilganda darhol menyu ochiladi**, bo'sh chek emas: ofitsiant
  stol yonida turibdi va hozir nima xohlashlarini eshitadi — oraga qo'yilgan
  bo'sh ro'yxat hech nima bermaydigan bosish.
- ⚠️ **"Mening stollarim" standart, "hammasi" bir bosishda**: hamma stolni
  ko'rsatadigan ekran o'qib o'tiladigan ro'yxat, faqat o'zinikini
  ko'rsatadigani esa kimdir erta ketganda stolni tashlab qo'yadi.
- ⚠️ **Rasm yo'q, faqat rang**: planshet mobil internetda va qo'lda yuradi, va
  rasmlar to'ri uni mehmon oldida sekin qiladigan yagona narsa.
- ⚠️ **To'lov tugmasi yo'q, va u yashirilgani uchun emas**: planshet ko'tarib
  yurgan ofitsiantda yashik ham, printer ham, bank terminali ham yo'q. Doim
  boshqa joyga yuboradigan tugma — ekran noto'g'ri ekanini o'rgatadigan tugma.
- Bekorchilikda avtoqulf (3 daqiqa) — ⚠️ bu yerda kassadan **muhimroq**: stolda
  qolgan planshet — istalgan odam olib ketadigan ekran, va undan berilgan har
  buyurtma oldingi ofitsiantning nomini tashirdi.

### Taomga izoh (yangi endpoint)
`PUT /staff/checks/{id}/lines/{lineId}` — ⚠️ **faqat yuborishdan oldin, keyin
rad etiladi (409)**. Chek chop etilgandan keyin pass'dagi qog'ozda eski matn
turadi va buni dasturiy o'zgartirib bo'lmaydi: jimgina tahrir ekran bilan
oshxonani bir taom haqida ziddiyatga soladi, va buni mehmon aniqlaydi. Rad
javobi ishlaydigan yagona ko'rsatmani beradi: oshxonaga o'zingiz ayting yoki
qatorni olib tashlab qaytadan qo'shing.

⚠️ Yuborilgan qatorda **qalam belgisi umuman chizilmaydi** — doim "yo'q"
javobini beradigan tugma ishonchni yo'qotadi.

⚠️ Izoh uchun `waiter` dan ortiq ruxsat ham, sabab ham so'ralmaydi: u
restoranga hech nima turmaydi va bu ekranning xonaning narigi tomoniga
qichqirish o'rniga mavjud bo'lish sababi.

### Yozilgan fayllar
Backend: `handlers/tilllines.go` (`StaffCommentCheckLine`), `router.go`.

Frontend: `app/zal/{page,layout,OrderPanel}.tsx`, `components/till/` (7 fayl
ko'chirildi), `lib/api.ts` (`tillCommentLine`), `lib/i18n/admin.ts`.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ · `tsc --noEmit` ✓ ·
`/zal` **200** ✓ · `/kassa` **200** ✓ · `/admin/settings` **200** ✓

### Qolgan ish
5. **Windows ilova (Wails) + printer** — ⚠️ printer modellari kutilmoqda
6. **Oflayn** — 5 ga bog'liq

Ikkalasi ham javobsiz savollarga tayanadi (printer modellari; Multikassa'dan
pul birligi va CORS).

---

## 2026-08-17 — Jonli sinovdan keyingi kamchiliklar 📝

Kod yozilmadi. `docs/pos-reja.md` §11 ga yozildi.

### ⚠️ Tuzatilmagan xato: zal cheki qo'ng'iroq chalyapti
Alomat: chek buyurtmalar ro'yxatida ko'rinmaydi, **lekin ovoz chiqadi**.

Tekshirdim — sabab men o'ylaganimdan boshqa va **men qoldirgan bo'shliq**:
`check` filtrini faqat `AdminListOrders` ga qo'shganman. `AdminAlerts` butunlay
boshqa endpoint va u hali ham zal cheklarini sanaydi:

- `StaffOpenCheck` → `status: pending`, `paymentStatus: "unpaid"`
- `AdminAlerts.pendingOrders` → `status == pending` va `paymentStatus != "pending"`
- `"unpaid" != "pending"` → **mos keladi** → qo'ng'iroq

⚠️ Bu men aytganimdan **yomonroq** holat: ro'yxat bo'sh, ovoz chalinadi. Panel
yo'q narsa uchun jiringlaydi va operator uni topa olmaydi.

Tuzatish: yetkazishga tegishli filtrlarga (`pendingOrders`, `plain`, `placed`,
`due`, `dueWaiting`, `upcoming`) `check: {$exists: false}` qo'shiladi.
⚠️ **`pos.failed`, `pos.unaccepted`, `fiscal.unfiled` ga tegilmaydi** — ular
aynan zal cheklariga ham tegishli.

### Boshqa uchta kamchilik
- **UI arzon ko'rinadi** — talab "iiko darajasida, lekin chiroyli" edi.
  Bloklar juda katta va cho'ziq, ierarxiya va kontrast yetishmaydi. Alohida
  dizayn bosqichi kerak.
- **PIN'dan keyin smena so'ralmaydi** — kassada ham, zalda ham: smena ochiq
  bo'lmasa ochish taklif qilinishi kerak. Backend tayyor, bu ekran oqimi.
- **Stollar: zal va saboy alohida zonalar** (1–28 va 100–130 kabi), xarita va
  **ro'yxat** ko'rinishi, sozlamalardan tanlanadi. ⚠️ Modelni kengaytiradi:
  bugun stolda zona yo'q, va bron tizimi shu ro'yxatni o'qiydi — saboy
  stollari bronga chiqmasligi uchun zonada "bron qilinadimi" bayrog'i kerak.

### ✅ Multikassa javob berdi: PDF to'g'ri
Pul birligi hal bo'ldi — `receipt_sum` va `received*` **tiyinda**, `items[]`
**so'mda**. `tiyinToSum` va uni muhrlagan test **to'g'ri** edi, o'zgartirilmaydi.

⚠️ **CORS savoli hali javobsiz** — lekin relay yozilgani uchun bu ishni
to'xtatmaydi.

---

## 2026-08-17 — Qo'ng'iroq xatosi tuzatildi + REGOS adapteri ✅

### 1. ⚠️ Zal cheki endi qo'ng'iroq chalmaydi
`AdminAlerts` da yetkazishga tegishli olti filtrga `check: {$exists: false}`
qo'shildi: `pendingOrders`, `plain`, `placed`, `due`, `dueWaiting`, `upcoming`.

⚠️ **`pos.failed`, `pos.unaccepted`, `fiscal.unfiled` ga tegilmadi** — ular
aynan zal cheklarini sanash uchun bor, va u yerdan olib tashlash "pul olindi,
chek yuborilmadi" ogohlantirishini o'chirardi.

Ikkala qoida ham testda muhrlangan, va test **shaklni** tekshiradi: filtr bir
joyga qo'shilib ikkinchisiga qo'shilmasligi — aynan shu xatoning o'zi edi.

### 2. REGOS VCR — ikkinchi haqiqiy adapter
docs.regos.uz **ochiq**, ya'ni bu ro'yxatdagi yagona hujjatlashtirilgan
provayder. Shakli: **JSON-RPC 2.0**, bitta endpoint, `auth` —
`Base64(login:parol)` va u **sarlavhada emas, tananing ichida**.

Multikassa kabi **lokal** (hujjat: `Sys.*` dan boshqa har metod ulangan
printerni talab qiladi), ya'ni bir xil transportdan foydalanadi. Farqi:
⚠️ **bulutli sinov muhiti bor** (`vcr-test.regos.uz`) — ya'ni bu adapterni
birorta restoran hech nima imzolashdan **oldin** sinash mumkin.

⚠️ **Bitta so'rovda uchta har xil miqyoslash bor, va birortasi ko'rinmaydi:**
1. **Pul — tiyin** (`900000` = 9 000 so'm)
2. **Miqdor — mingdan bir** (`1000` = **bitta** porsiya)
3. **QQS — foiz × 100** (`1200` = 12%)

Ikkinchisi eng xavflisi: butun sonni shundayligicha yuborish porsiyaning
mingdan biri uchun chek yozadi, va **jami baribir to'g'ri ko'rinadi**, chunki u
alohida yuboriladi. Uchalasi ham bitta joyda aylantiriladi va testda muhrlangan.

Boshqa qarorlar:
- ⚠️ **`ok` hal qiladi, HTTP status emas** — rad javobi 200 ichida keladi.
- ⚠️ **Buyurtma raqami `code` maydoniga ketadi** — REGOS uni takrorlanishni
  tekshirish uchun ishlatadi. Ya'ni javob yo'qolgandan keyin qayta yuborilgan
  chekni **kassaning o'zi** rad etadi, ikki marta yozmaydi.
- **To'lovning ikkala yarmi ham yuboriladi**: zalda qisman naqd, qisman karta
  odatiy hol, va bittaga yig'ish kassa bilan ziddiyatga tushadigan chek beradi.
- `Sys.GetInfo` ulanish tekshiruvi sifatida — u **printersiz ham ishlaydigan**
  uch metoddan biri, ya'ni "kassaga yetib bo'lmadi" bilan "kassa bor, printeri
  yo'q" ni ajratadi.
- `ZReport.Open` / `ZReport.Close` allaqachon mavjud `ShiftOpener` /
  `ShiftCloser` interfeyslariga tushdi — kassa oqimi o'zgarmadi.

⚠️ Test tuzatildi: "ready" **adapter bor** degani, "bu filial formani
to'ldirgan" degani emas. REGOS bo'sh kalitlarda `ErrNotConfigured` qaytaradi va
ikkalasini chalkashtirish har kelajakdagi adapterni hech nimasiz ishlay
oladigandek ko'rsatishga majburlardi.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓
Yangi testlar: qo'ng'iroq filtrlari (ikki tomondan), REGOS miqyoslashlari,
`ok` bayrog'i, `code` takroriy kaliti, Base64 auth, bo'sh ИКПУ, JSON bo'lmagan
javob, printersiz Hello, smena ochish/yopish.

### Qolgan ish (tartib bo'yicha)
2. Stol zonalari (zal / saboy, ro'yxat ko'rinishi) — model o'zgaradi
3. PIN'dan keyin smena oqimi
4. Dizayn bosqichi (iiko darajasida)
5. Sidebar ichida sidebar

---

## 2026-08-17 — Zonalar, smena darvozasi, dizayn, sidebar, variantlar ✅

Kelishilgan ro'yxat oxirigacha bajarildi. To'liq izohlar —
`docs/pos-reja.md` §11; bu yerda faqat nima qilingani va nima uchun.

### Stol zonalari (zal / saboy)
`models.TableZone{Bookable, Layout}` + `FloorTable.ZoneID`;
`(*BookingSettings).Bookable()` — uch ekran o'qiydigan **bitta** funksiya.
Panelda `TableZonesEditor` (zona CRUD + **raqam oralig'i bilan stol qo'shish**,
100–130 bir bosishda), kassa/zalda zona **tablari**.
- ⚠️ Zonasiz stol ham, o'chirilgan zonadagi stol ham **bron qilinadi**: nol
  qiymat "bron qilinmaydi" bo'lsa, bu deploy har bir mavjud restoranning bron
  sahifasini bo'shatardi.
- ⚠️ Saboy raqamlari mehmonning brauzeriga **umuman bormaydi** — `BookingPlan`
  ularni javobdan chiqaradi, faqat ekranda yashirmaydi.
- Zona o'chirilsa stollari qoladi (standart zonaga qaytadi).
- Test: `internal/handlers/tablezones_test.go` (4 ta).

### PIN'dan keyin smena (`ShiftGate`)
Ikkala ekranda ham. ⚠️ **Banner emas, darvoza**: ishlab turgan ekran tepasidagi
yozuv o'qilmay o'tib ketiladi, smenasiz ochilgan chek esa hech qanday hisobga
kirmaydi — pul olinadi, kechqurungi hisob shu summaga kam bo'ladi va hech
qayerda xato chiqmaydi. `CashShiftPanel` darvozaga `onChanged` beradi: aks
holda kassir smenani yopib, zal hali ochiq deb ishlashda davom etardi.

### Dizayn: `.till` qatlami
⚠️ **Ildizi did emas, tuzilma edi** — kassa **saytning** dizayn tizimini
o'qiyotgan edi (`--radius-btn: 9999px`, `--root-size`, ega tanlagan aksent).
Ega temani sinab ko'rgani uchun shakli o'zgaradigan kassa mushak xotirasini
nolga tushiradi, kattalashtirilgan shrift esa 1024×768 monoblokda sahifani
chetdan chiqaradi. `globals.css` da `.till` o'zgaruvchilarni qayta e'lon qiladi
va `till-chrome / till-row / till-btn / till-panel / till-input / till-label`
beradi.
- ⚠️ **Rang faqat ma'no tashiydi**: ilgari taom plitkasining butun foni
  kategoriya ohangida edi — ohanglar bitta to'plamdan, tema esa ikkita, ya'ni
  qorong'ida har plitka loyqa jigarrang bo'lib nom ham, narx ham botib ketardi.
- ⚠️ Rang chizig'i **rasm ostida qolib ketgan** edi (`z-10` yo'q): rasmli har
  bir plitkada, demak hammasida, rang belgisi mavjud emasdi.
- ⚠️ To'ldirilgan tugma **chek qaysi bosqichda ekaniga qarab ko'chadi**.
  Ilgari "Oshxonaga yuborish" `bg-charcoal` edi — qorong'i temada sahifa
  fonining o'zi, ya'ni eng shoshilinch amal eng ko'rinmas narsa.

### Yo'l-yo'lakay topilgan nuqsonlar
- ⚠️ **`/kassa` stollar ekranini umuman render qilmasdi**: `TablesScreen`
  import qilingan, `view` holati bor, JSX'da yo'q. "Avval stollar" talabi
  bajarilgan deb hisoblanardi, amalda kassa hamon menyudan ochilardi.
- Chek paneli o'ziga qayta kenglik yozardi (`lg:w-80`) — 24 px chetdan chiqib
  butun sahifaga gorizontal skroll qo'shardi.
- Peshtaxta plitkasi eng katta matn qilib chek raqamini (`6XGC-ZNHV`)
  chizardi. U `crypto/rand` dan — ataylab taxmin qilinmaydigan, demak ataylab
  o'qilmaydigan. Endi `#1`, haqiqiy raqam mayda.
- ⚠️ **`/kassa` va `/zal` `UNLOCALIZED` da yo'q edi**: ruschaga o'tilsa manzil
  `/ru/kassa` bo'lardi va yo'lga bog'langan har bir qoida mos kelmay qolardi —
  birinchi navbatda `forcedLight()`, ya'ni kassa tilini o'zgartirgan zahoti
  qorayardi.

### Sidebar ichida sidebar (`/admin`)
Tor **rels** (biznesning beshta qismi) + **panel** (o'sha qismning ekranlari).
- ⚠️ **Rels navigatsiya qilmaydi**: guruhning birinchi ekraniga o'tadigan rels
  "Jamoa"da nima borligini ko'rmoqchi bo'lgan odam uchun hisobotni yuklardi.
- ⚠️ Ochiq guruhni **sahifa** hal qiladi; qo'lda tanlash marshrut o'zgargunicha
  yashaydi. Sahifa bilan kelishmaydigan sidebar ortiqcha bosishdan yomonroq.
- ⚠️ **Eng uzun moslik yutadi** (`groupOf`): `/admin` boshqa har bir
  marshrutning prefiksi.
- Telefonda ataylab yassi ro'yxat; ikkalasi bitta `NAV_GROUPS` dan o'qiydi.

### Kassada variant tanlash (`OptionDialog`)
⚠️ Busiz **variantli har bir taom kassadan sotilmasdi**: server to'g'ri ish
qilardi (`resolveOptions`), planshet esa faqat `{menuItemId, qty}` yuborardi va
hech bir ekranda javob berish imkoni yo'q edi.
- Faqat **guruhi bor** taomga ochiladi (bitta bosish — kassaning tezlik
  argumenti). Shart "majburiymi" bo'yicha emas: ixtiyoriy guruh ham aks holda
  sotilmaydi.
- Kalit — **base (uz) nomi**: tarjimani saqlash kassirning interfeys tilini
  buyurtmaga yuborardi.
- ⚠️ `busy` bayrog'i: so'rov ketayotganda ikkinchi bosish taomni ikki marta
  qo'shardi — aynan oyna oldini olishi kerak bo'lgan xato, oynaning o'zi orqali.
- **Tanlangan variant chekda yoziladi**: busiz bir xil nomli ikki qator ikki
  xil narxda turadi va farqini hech kim tushuntira olmaydi.

### Kassa/zal: faqat yorug' tema, uch til, yangi qulf ekrani
- **Faqat yorug'** (`forcedLight()`): kassa sayt emas, **jihoz** — kun bo'yi
  monoblokda, restoran yorug'ligida, oq fonda olingan taom rasmlari bilan.
  ⚠️ Qoida ikki joyda (inline skript + `theme.tsx`) va bir xil aytishi shart,
  aks holda har yuklanishda chaqnash. Saqlangan tanlov o'chirilmaydi.
- Tema tugmasi olib tashlandi: hech nima qilmaydigan tugma yo'q tugmadan
  yomonroq.
- Til almashtirgichi **qulf ekranida va smena darvozasida ham**: kassa umumiy
  mashina, ruscha o'qiydigan kassir o'zbekcha qulf ekranidan o'tolmasdi.
- Qulf ekrani: **Keel** belgisi va nomi yonma-yon (belgi `keel.deep #D2870F`,
  nom qora Poppins), pastida PIN paneli, guruh ekran o'rtasida.
  ⚠️ Rang `brand` emas: `brand` — eganing aksenti, unda chizilgan belgi har
  mijozda boshqa logotip bo'lardi.
  ⚠️ Ostida `overflow-y-auto`: qisqa ekranda o'rtaga qo'yilgan blok sig'masa
  ikkala uchini yo'qotadi, pastdan yo'qotadigani esa backspace turgan qator.

### Tekshiruv
`go build ./...` ✓ · `go test ./internal/...` yashil ✓ · `next build` ✓
Kassa va zal brauzerda qo'lda sinaldi (PIN → zal → chek → variant → chek).

### Qolgan ish
- **Wails Windows ilovasi + printerlar** — to'xtatilgan (printer hozircha
  kerak emas). Oflayn ish shunga bog'liq.
- **Multikassa CORS** — provayderdan javob kutilyapti.
- `/kassa` va `/zal` ekran oqimi uchun avtomatik test yo'q (backend qismi
  testda muhrlangan).

---

## 2026-08-18 — Kassa va zal oqimi uchun test ✅

Kechagi ro'yxatda qolgan yagona o'zimizga bog'liq ish. Sabab §11 dagi
nuqsonlarning **shakli**: `TablesScreen` import qilingan-u JSX'da yo'q edi,
`OptionDialog` umuman yo'q edi, `/kassa` da zalga qaytish tugmasi yo'q edi —
uchalasida ham **server to'g'ri ishlardi va testda muhrlangan edi**. Go testi
bu xatolarning birortasi uchun ham qizarmaydi.

### Nima qurildi
`frontend/` da birinchi brauzer testi: **vitest + Testing Library + jsdom**
(`vitest.config.mts`, `npm test`). Faqat kassa va zal: `include` ataylab
nomlangan, `src/**` glob'i bugun hech nima yig'maydi va keyin yarim yozilgan
fayllarni yig'a boshlaydi.

- `src/test/tillServer.ts` — **soxta kassa serveri**: menyu (oddiy taom,
  variantli taom, sotuvda yo'q taom), ikki zona, stollar, cheklar, smena, PIN.
  Narxlash qoidalari **takrorlanmaydi** — o'zi bilan bahslashadigan soxta
  server o'zini tekshiradi; lekin **majburiy guruh tekshiruvi bor**, aks holda
  javobsiz variantni qo'sha oladigan test kassa buzuq turganda ham yashil
  bo'lardi.
- `src/test/setup.ts` — faqat `api` almashtiriladi. `ApiError`, token
  yordamchilari va `imageUrl` **haqiqiy** qoladi: ekranlar
  `err instanceof ApiError` bo'yicha shoxlanadi va qurilma tokenini
  `localStorage` da, odamnikini `sessionStorage` da tutadi.
- `src/test/tillFlow.tsx` — PIN terish, smena ochish, stol/taom plitkasini
  topish.

24 ta test: qulf ekrani, smena darvozasi, zal (zona tablari), chek ochish,
bir bosishda taom, variant dialogi (majburiy guruh, base uz nomlari, ikki
bosish qo'riqchisi), ruxsatlar, va PIN'siz eski install.

### ⚠️ Test darhol ikkita jonli xatoni topdi
**1) Bog'langan monoblokda menyu ham, stollar ham, cheklar ham yuklanmasdi.**
`/kassa` da bu uchala so'rov `if (!staff) return` ortida edi — bog'langan
monoblokda esa **staff hisobi umuman yo'q** (paneldagi havola bilan
bog'lanishning butun ma'nosi shu). Ekran "Stollar chizilmagan — Sozlamalar →
Stol bron qilish bo'limida chizing" deb yozardi, ya'ni **sozlama xatosiga
o'xshardi**, va ega allaqachon to'g'ri to'ldirilgan sahifani qayta
to'ldirgan bo'lardi. Hech qanday so'rov yiqilmagan, hech qayerda xato yo'q.

`/zal` da xuddi shu narsa boshqa yarmidan: u `person` ga bog'langan edi, ya'ni
**PIN belgilamagan** (eski, login bilan ishlaydigan) filialda zal bo'sh
ochilardi.

Ikkalasida ham endi bitta ifoda: `unlocked = person || (staff && !pinsUsed)` —
`useShift` allaqachon aynan shuni so'rayotgan edi.

**2) Uskunaning o'zidagi tuzoq** (testda yozib qo'yilgan): `formatPrice`
minglarni **uzilmas bo'sh joy** bilan ajratadi, Testing Library esa DOM
matnini solishtirishdan oldin bo'sh joylarni normallashtiradi — ya'ni ikki
qator ekranda **bir xil ko'rinadi** va mos kelmaydi. `price()` yordamchisi
ikkalasini ham hal qiladi.

### Tekshiruv
`npm test` → 24/24 ✓ · `tsc --noEmit` ✓ · `next build` ✓ · `next lint` (yangi
fayllarda ogohlantirish yo'q).

### Qolgan ish
- **CI'da ishlamaydi**: repoda faqat deploy workflow'i bor. `npm test` ni
  push'da yuritadigan ish alohida qadam.
- Wails Windows ilovasi + printerlar — to'xtatilgan (oflayn ish shunga bog'liq).
- Multikassa CORS — provayderdan javob kutilyapti.

### Qo'shimcha (o'sha kuni)
- **CI qo'shildi** (`.github/workflows/test.yml`): `go build` + `go test` ikkala
  modul uchun (`backend` va `control`), va frontendda `npm test` + `tsc`.
  Har push va har PR'da. ⚠️ **Deploy ishiga zanjirlanmagan**: ular mustaqil
  yiqiladi, va testni deploy'ning oldiga qo'yish `apply()` dagi `else if`
  xatosining o'sha shakli bo'lardi. ⚠️ `TZ=Asia/Tashkent` beriladi — yuguruvchi
  UTC'da turadi, smena va kassa kuni esa mahalliy kun chegarasi bo'yicha
  kesiladi.
- **Kassa va zal sarlavhasida qulflash tugmasi — endi ikonka**: ochiq qulf
  (`LuLockOpen`), qulf ekranida esa yopiq qulf (`LuLock`). Bir obyektning ikki
  holati "bu mashina hozir qulflangan" ni jumladan tez aytadi, va sarlavha —
  768 px'li ekranda taomlar panjarasidan olinadigan qator. Login bilan
  ishlaydigan tillda ikonka boshqa (`LuLogOut`): hisobdan chiqish umumiy
  mashinani qulflash emas.
  ⚠️ Matn **yorliq bo'lib qoladi** (`aria-label` + `title`) — ikonkali tugma
  yorlig'ini yo'qotishning odatiy yo'li, va yo'qolganda ekranda hech nima
  o'zgarmaydi. Testda muhrlangan.

---

## 2026-08-18 — Kassa va zal dizayni: jihoz, sayt emas ✅

Kechagi `.till` qatlami burchak va shriftni ajratgan edi; bu bosqich **butun
palitrani** ajratadi va ekranlarni iiko darajasidagi zichlik va ierarxiyaga
keltiradi.

### Palitra: sayt iliq, kassa sovuq
Saytning tokenlari iliq (krem qog'oz, iliq kulranglar, eganing aksenti) — va bu
vitrina uchun to'g'ri. Kassa esa **o'sha xonada turgan mashina**: kechqurun
lyuminestsent yorug'likda iliq kulranglar loyqalanadi, oq fonda olingan taom
rasmlari kremda dog'dek turadi. `.till` endi `--bg/--surface/--line/--fg` ni
sovuq neytralga, `--font-display` ni sans'ga qayta e'lon qiladi va hammasi
**qat'iy**: bir tarmoqning ikki filialida kassa boshqa rangda bo'lmasligi kerak.

### ⚠️ To'ldirilgan tugma — qora, eganing aksenti emas
`--brand` — eganing rangi, va u tanlaydigan aksentlarning yarmi (sariq, salat,
pushti) oq matn bilan peshtaxta narigi tomonidan **o'qilmaydi** — kuniga ming
marta bosiladigan yagona tugmada. Qora har installda o'qiladi va aksentni
**ma'no** uchun bo'shatadi. Keel belgisi `brand` da chizilmasligi bilan bir
sabab.

Aksent endi faqat holat: **amber** — "stolda odam bor / oshxonaga
yuborilmagan", **qizil** — "juda uzoq kutdi" (45 daqiqa; qisqaroq chegara soat
sakkizda butun zalni qizartiradi va doim qizil zal hech nima demaydi).

### ⚠️ `text-danger` hech qachon kompilyatsiya qilinmagan
Kassada, panelda va dialoglarda har bir xato qatori `text-danger` bilan
yozilgan — lekin bunday rang Tailwind konfiguratsiyasida **umuman yo'q edi**,
ya'ni klass generatsiya qilinmasdi. Hech nima buzuqqa o'xshamasdi: jumla
o'sha yerda, to'g'ri, va **jimgina** oddiy siyoh rangida. Endi `--danger`
semantik token (ikki tema uchun alohida) va u mavjud har bir chaqiruvni birdan
tuzatadi.

### Ekran bo'yicha
- **Stollar**: plitka **to'ldirilmaydi, bo'yaladi** — ilgari band stol solid
  `brand` + oq matn edi, ya'ni plitka mavjud bo'lish sababi bo'lgan ikki raqam
  (qancha va qancha vaqt) birinchi bo'lib yo'qolardi. Yon chiziq holatni to'liq
  kuchda tashiydi. Zona tabida **band stollar soni**: qaramayotgan zonang aynan
  unutiladigan zona.
- **Menyu**: kategoriya chipi — rang **nuqta** bo'lib qoldi (kategoriyaning
  o'zligi), tanlov esa **to'ldirish** bilan aytiladi. Ilgari o'zlik holatni
  aytardi va sakkizta to'yingan rang tepada bir-biri bilan raqobatlashardi.
  Variantli taom nuqtasi nom yonidan **plitka burchagiga** ko'chdi: inline holda
  u taom nomining bir qismi ("Osh." kabi) edi va ikki qatorli nomda qayerga
  tushishi noma'lum edi.
- **Chek**: soni endi **o'z kvadratida**, nomdan oldin — ilgari "2 × 30 000"
  bo'lib pastda turardi, ya'ni chekni mehmonga o'qib berayotgan kassir uni
  jumladan ajratib olishi kerak edi. "O'chirish" tagi chizilgan 12 px matn edi
  (sensorli ekranda **tegmaydigan** nishon, va tegmagani yuqoridagi qatorning
  narxiga tushadi) — endi 44 px qizil ikonka tugma. Pastda pul **o'z blokida**,
  tugmalar tepasida emas.
- **Tugmalar ierarxiyasi**: ilgari futerda to'rtta bir xil kenglikdagi tugma
  turardi, ya'ni ekranning shakli chek nimani kutayotganini aytmay qolgandi.
  Endi bitta to'ldirilgan amal (yuborish → to'lash), tagida ikkita **jim**
  tugma (ko'chirish, bekor qilish).
- **Zal**: `btn` / `btn-ghost` / `input` — saytning klasslari edi, aynan `.till`
  qatlami mavjud bo'lish sababiga qarshi; hammasi `till-*` ga o'tkazildi.
- Fokus halqasi qo'shildi (kassalar barkod skaneri va USB klaviatura bilan
  yuritiladi), dialoglar yagona `till-dialog` sirtiga o'tdi, PIN nuqtalari
  eganing aksentidan **Keel** rangiga.

### Tekshiruv
`npm test` 26/26 ✓ · `tsc --noEmit` ✓ · `next build` ✓ · `next lint` toza.
⚠️ Brauzerda ko'z bilan hali ko'rilmadi — jonli ekranda tekshirish kerak.

---

## 2026-08-18 — Kassa va zal: Keel POS maketi bo'yicha ✅

Manba: claude.ai/design → **"Keel POS"** (`Keel POS.dc.html`) — 1920×1080 kassa
va 1280×800 ofitsiant terminali. Maketning **dizayn tili** to'liq ko'chirildi;
maketda bor-u mahsulotda yo'q **funksiyalar qo'shilmadi** (pastda ro'yxat).

### Palitra maketdan
Iliq yer (`#F4F1EC`), oq panellar, `#E4DED4` chiziqlar, sovuq matn
(`#05101A / #16344A / #6C8397 / #8FA6B8`), aksent — **Keel amber `#F5A524`**.
- ⚠️ **Aksent qora matn bilan.** Bu kechagi "to'ldirilgan tugma qora" qaroridan
  yaxshiroq yechim: eganing rangi baribir ishlatilmaydi, lekin tugma jonli
  ekranda **ko'rinadi** — oq matnli aksent o'qilmasligi muammosi amber ustidagi
  **qora** matn bilan hal bo'ladi.
- ⚠️ **Ramka endi oq, qora emas.** Ilgari "qorong'i ramka, yorug' ish maydoni"
  edi; maket buni boshqacha va yaxshiroq hal qiladi — butun mashina bitta
  yorug' sirt, ajratuvchi narsa — soch chizig'i va yer rangi. 1080p ekranda
  qora ramka video pleerga o'xshab qolardi.
- ⚠️ **Raqamlar monoshirift** (`--font-num`): pul, vaqt va kodlar ustunda
  taqqoslanadi va mehmonga ovoz chiqarib o'qiladi.

### Ekranlar
- **Yagona sarlavha** (`components/till/TillChrome.tsx`) — ikkala ekranda bitta
  komponent: Keel belgisi, smena holati (yashil nuqta + soat), filial nomi,
  kim ishlayotgani (bosh harflari bilan), **soat** (monoblok fullscreen —
  boshqa soat yo'q), til va qulf. ⚠️ Soat faqat mount'dan keyin chiziladi:
  serverda peshtaxtaning soati yo'q va SSR gidratsiya xatosi beradi.
- **Zal**: "Mening stollarim / Hamma stollar" endi **segment** (ikkala holat
  ham ko'rinadi — o'zini qayta nomlaydigan tugmani yarim odam teskari o'qiydi)
  va yonida **bo'sh/band sanog'i**. Stol kartochkasi maketdagidek: katta raqam,
  holat nuqtasi, "N joy · holat", vaqt va summa.
- **Menyu**: rasmsiz rejimda plitkada **rangli kvadrat** (kategoriya rangi +
  taomning bosh harfi) — maketdagi ikonka kvadratining o'rni; rasm bilan
  rasmning o'zi. Narx katta, "so'm" alohida va jim.
- **Chek**: sarlavha (stol · mehmon, ofitsiant · vaqt, `#raqam`), qatorlar,
  jami bloki, aksent tugma.

### ⚠️ Yangi imkoniyat: qator sonini o'zgartirish (+ / −)
Maketning asosiy o'zaro ta'siri — chekdagi **stepper**. Busiz bitta taomni
uch marta sotish uchun plitka uch marta bosiladi va chekda **uchta qator**
paydo bo'ladi.
- Backend: `PUT /staff/checks/{id}/lines/{lineId}` endi `qty` ni ham qabul
  qiladi (`StaffEditCheckLine`).
- ⚠️ **Ikkala maydon ham pointer** (`Comment *string`, `Qty *int`): oddiy satr
  bo'lsa "bu so'rovda izoh yo'q" va "izohni o'chir" farqlanmaydi — "+" bosgan
  kassir mehmonning "piyozsiz"ini **jimgina** o'chirib yuborardi.
- ⚠️ **Nol qabul qilinmaydi**: qatorni olib tashlash — boshqa amal, boshqa
  yozuv (yuborilmagan qator o'chadi, yuborilgani kassir va sabab talab qiladi),
  va nolda jimgina void qiladigan stepper — kassaning "ovqat qayerga ketdi"
  degan savolga javob bera olmasligi.
- ⚠️ **Yuborilgandan keyin o'zgarmaydi** (409): pass'dagi qog'ozda eski son
  turadi. Yuborilgan qatorda faqat "olib tashlash" (sabab bilan) qoladi.
- ⚠️ **Pul qayta hisoblanadi** (`applyCheckTotals`) — izoh uchun kerak emasdi,
  son uchun bu hisobning o'zi.
- Testda muhrlangan: `TestApplyLineEdit` (5 holat).

### Maketda bor, mahsulotda yo'q — **qo'shilmadi**
Ishlamaydigan tugma yo'q tugmadan yomon (bu — kassadagi tema tugmasi bilan bir
qoida). Ro'yxat, kelgusi bosqichlar uchun:
- Chap ikonkali navigatsiya (Buyurtma / Stollar / Chek / Kuryer / Hisobot)
- "Kunlik sotuv" sarlavhada — kassada bunday endpoint yo'q
- Rejim segmenti (Zal / Olib ketish / Yetkazish) — chek faqat zalniki
- **Xizmat haqi 10%** — narxlash quvurida yo'q
- Chegirma / Stolni birlashtirish / Bo'lib to'lash / Qaytarish paneli —
  chegirma `PayDialog` ichida bor, qolgan uchtasi yo'q
- To'lov usuli chek panelida (bizda alohida `PayDialog`)
- Stolning "Hisob" holati va qator holatlari (Berildi / Tayyorlanmoqda) —
  bizda ikki holat bor: yuborilgan / yuborilmagan
- Taom kodi (`101`) va taom emojisi — menyu modelida yo'q

### Tekshiruv
`npm test` 26/26 ✓ · `tsc` ✓ · `next build` ✓ · lint toza ·
`go build` + `go test ./internal/...` ✓ (yangi: `TestApplyLineEdit`).
Dev serverlar ishlab turibdi — ko'z bilan ko'rish qoldi.

### Tuzatish: dizayn tili emas, **tuzilishi** ham (o'sha kuni)
Birinchi urinishda faqat palitra, shrift va boshqaruvlar ko'chirilgan edi —
ekranlarning **joylashuvi** eskiligicha qolgani uchun natija maketga
o'xshamasdi. Endi tuzilishi ham maketdagidek:

- **Chap ikonkali rels** (`components/till/TillNav.tsx`) — maketda beshta punkt
  bor, bizda **uchtasi mavjud**: Stollar, Menyu, Kassa. ⚠️ Yo'q ekranlar uchun
  tugma qo'yilmadi: "hali yo'q" deb javob beradigan rels — xodimni tugma
  bosmaslikka o'rgatadigan rels. Relsda **nuqta** bor: zalda oshxonaga
  yuborilmagan chek yoki fiskallashtirilmagan sotuv bo'lsa.
- **Chek endi doimiy ustun** (o'ngda, eng keng). Ilgari u faqat menyu
  ko'rinishida chizilardi **va** kassa/fiskal panellarining tagida turardi —
  ya'ni mehmon kutayotgan raqam kassir zalga qaragan zahoti yo'qolardi.
- **Pastki amal paneli**: stolni ko'chirish, chekni bekor qilish, o'ngda —
  kassa. Ilgari bular chek ostida to'rtta bir xil tugma bo'lib turardi.
- **Kassa/fiskal panellari — alohida manzil** (relsdagi "Kassa"), chek ustida
  emas.
- **To'lov usuli chekda tanlanadi** (Naqd / Karta / O'tkazma), keyin
  "To'lovni tasdiqlash" dialogni **o'sha usul bilan** ochadi: mehmon "karta"
  deb chekni o'qib berayotganda aytadi, dialog esa har safar naqddan
  boshlanardi.
- **Zal ikki panelli**: chap tomonda zal yoki menyu, o'ngda **doim** shu
  stolning buyurtmasi. Ilgari uchta to'liq ekran edi — menyudan chiqmasdan
  "stol qancha bo'ldi?" degan savolga javob yo'q edi.
- ⚠️ **Tor ekranda yashirilmaydi, ustma-ust tushadi**: rels gorizontal qatorga,
  chek ustunning tagiga o'tadi. `hidden lg:flex` bo'lsa telefonda ochilgan
  ekranda navigatsiya ham, jami ham umuman bo'lmasdi.

### Ikki xato (jonli sinovdan keyin)

**1. ⚠️ Tanlangan tugma oq fonda oq bo'lib ko'rinmasdi.** Dizayn bosqichida
`.till` tokenlari qayta nomlangan, beshta "tanlangan" holat esa eski
`--till-action` ga murojaat qilib qolgan edi. `rgb(var(--till-action))` CSS'da
**xato emas** — u shunchaki yaroqsiz rang, ya'ni fon umuman bo'yalmaydi va oq
matn oq sirtda qoladi. Tanlangan stol, tanlangan hajm va tanlangan to'lov
usuli — uchalasi ham ko'rinmasdi, va **hech nima** qizarmadi: build ham, tiplar
ham, oqim testlari ham (ular tugma nima **qilishini** tekshiradi, qanday
ko'rinishini emas).
Endi hammasi amber tanlov (`--till-accent-tint` + amber hoshiya va matn), va
`src/app/kassa/tokens.test.ts` **har bir `var(--till-…)` e'lon qilinganini**
tekshiradi — aynan shu turdagi jimgina xatoni ushlaydi.

**2. ⚠️ Bir taomni ikki marta bosganda ikkита qator qo'shilardi.** Kassa
**bosish bilan** ishlatiladi: to'rtta kofe — plitkani to'rt marta bosish, va
to'rtta bir xil qator mehmonga o'qib berib bo'lmaydigan chek beradi (va sonini
tuzatish uchun qatorlarni bittalab o'chirish kerak).
`StaffAddCheckLines` endi mos qatorga **qo'shadi** (`mergeableLine`):
- ⚠️ **Faqat oshxona ko'rmagan qatorga**: yuborilgan qatorda pass'dagi qog'oz
  sonni nomlaydi, uni jimgina o'stirish qog'oz bilan ekranni bir-biriga
  qarama-qarshi qo'yardi. Yuborilgani joyida qoladi, yangisi yoniga tushadi —
  bu ayni halol o'qish: "ikkitasi pishmoqda, yana bittasi so'raldi".
- ⚠️ **Taomning o'zi yetarli emas**: variantlar va izoh ham mos kelishi shart.
  "Osh (katta)" va "Osh (kichik)" — boshqa taom, "piyozsiz" yozilgan qatorga
  oddiysini qo'shish esa oshxonaga ikkalasi uchun noto'g'ri buyruq yuboradi.
- Variantlar **tartibdan qat'i nazar** solishtiriladi (`sameOptions`): dialog
  guruhlarni qaysi tartibda chizsa, shu tartibda yuboradi.
- Testlar: `TestMergeableLine` (8 holat) va `TestSameOptionsIgnoresOrder`;
  frontendda soxta server ham xuddi shunday birlashtiradi va oqim testi ikki
  bosishdan **bitta qator, soni 2** chiqishini muhrlaydi.

---

## 2026-08-18 — iiko funksionali: zal uch ko'rinishda ✅

Manba: mijoz bergan **iiko Front** skrinshotlari (4 ta). Talab — dizayn aynan
o'sha bo'lishi shart emas, **funksional** shunday bo'lsin.

### iiko nima qiladi (skrinshotlardan)
Схема зала · Все столы · По официантам · Быстрый чек — pastdagi to'rt tugma;
chapda ofitsiantlar ro'yxati (chek soni va summasi bilan); "По официантам"da
har chek **ichi ko'rinadigan kartochka**; buyurtma ekranida mehmonlar
(ГОСТЬ 1/2), kurslar (I·II·III), пречек, перенос, скидка/надбавка, son uchun
raqamli panel.

### Bu bosqichda qilingani — zalning uch ko'rinishi
Uchalasi ham **bizda allaqachon bor ma'lumotdan** quriladi, ya'ni backend
o'zgarmadi:
- **Zal sxemasi** (`TillFloorPlan.tsx`) — stollar **egasi chizgan joyda**:
  koordinatalar, shakl (to'rtburchak/doira), devor va nomlangan zonalar
  (`booking.shapes`), plan o'lchami. Stolda raqam, summa, necha daqiqa va
  oshxonaga yuborilmagan bo'lsa nuqta.
  ⚠️ **Alohida komponent**, bron sahifasiniki emas: u mehmonga "bo'shmi?"
  deb javob beradi, kassaga esa yana uchta savol kerak — va bitta komponentni
  ikkala auditoriyaga egish mehmon sahifasida pul ko'rsatish bilan tugaydi.
  ⚠️ **Sxema faqat chizilgan bo'lsa taklif qilinadi**: koordinata standart
  holatda 0, ya'ni stol raqamlarini kiritib, plan muharririni ochmagan
  filialda hamma stol chap yuqori burchakda uyulib qolardi — bu "kassa buzuq"
  bo'lib ko'rinadi.
- **Ro'yxat** — avvalgi plitkalar panjarasi.
- **Ofitsiantlar bo'yicha** — chapda ofitsiantlar (chek soni + summasi),
  o'ngda **ichi ko'rinadigan chek kartochkalari**: stol, vaqt, oltitagacha
  qator (soni + nomi, oshxonadagisi ko'k, yuborilmagani amber), jami.
  ⚠️ Soat sakkizda beriladigan savol "7-stol bandmi" emas — buni zalning o'zi
  aytadi — **"7-stol nimani kutyapti"**, va har bir kassa buni bilish uchun
  chekni ochishga majbur qilsa, chek ochishni istamagan odam uni ochadi.
- **Peshtaxta cheki har uch ko'rinishda** ochiladi: "bitta kofe olib ketishga"
  qaysi ekranga qarab turganingizdan qat'i nazar keladi.

### ⚠️ Ko'rinish holatdan emas, ma'lumotdan hisoblanadi
Avval `useEffect` profil yuklangach ko'rinishni sxemaga **almashtirardi** —
ya'ni panjara barmoq tushayotgan paytda almashib, bosish almashtirilayotgan
plitkaga tegardi (testda tasodifiy yiqilish bo'lib chiqdi). Endi
`view = tanlangan ?? (sxema bormi ? "plan" : "grid")` — sof funksiya, mount'dan
keyin hech nima sakramaydi.

### Keyingi bosqich (bu bosqichda **yo'q**)
Bular backend modelini o'zgartiradi, shuning uchun alohida:
- **Mehmonlar bo'yicha bo'lish** (ГОСТЬ 1 / ГОСТЬ 2) — qatorga mehmon raqami
- **Kurslar** (I / II / III) — qatorga kurs raqami, kurs bo'yicha yuborish
- **Пречек / chop etish** — printer (Wails ilovasi to'xtatilgan)
- **Перенос** — qatorlarni boshqa chekka ko'chirish (hozir butun chek ko'chadi)
- **Chegirma/nadbavka foizda** — hozir summada, `PayDialog` ichida
- **Rezervlar sanog'i** pastki panelda — bron tizimi bor, ulash qoldi

---

## 2026-08-18 — Kurslar, mehmonlar, ko'chirish, bronlar ✅

iiko skrinshotlaridagi funksionalning ikkinchi qismi. Uchtasi chek modeliga
tegdi, shuning uchun backend + test bilan.

### Mehmonlarga bo'lish (ГОСТЬ 1 / ГОСТЬ 2)
`OrderItem.Guest` — ⚠️ **nol "stol" degani**, "nolinchi mehmon" emas: bo'linmagan
buyurtma — bir kompaniyaga bitta hisob, va aksariyat ovqat shunday tugaydi.
Shu sabab bu maydon chiqqan kuni bironta ham mavjud chek o'zgarmadi.
- **Tanlangan tab — keyingi taom qayerga tushishi.** Butun mexanizm shu, va
  aynan shuning uchun bo'lish buyurtma **olinayotganda** bo'ladi: ofitsiant
  allaqachon stol atrofida "sizga nima?" deb so'rab yuribdi.
- ⚠️ **Yuborilgandan keyin ham o'zgartirsa bo'ladi** — va bu qolgan hamma
  tahrirdan farq qiladi. Nima pishirilishi chek chop etilganda muzlaydi; kim
  to'lashi esa **likopchalar yig'ishtirilganda** hal qilinadi. Bitta
  "yuborilgan qator tahrirlanmaydi" qoidasi bo'lish so'raladigan yagona
  daqiqada bo'lishni imkonsiz qilardi (`cooksAffected`).
- ⚠️ Ikki mehmonning bir xil taomi **ikki qator bo'lib qoladi** — birlashtirish
  bittasiga ikkovining hisobini yozardi (`mergeableLine`).

### Kurslar (I · II · III)
`OrderItem.Course` — nol "hamma narsa bilan birga".
- ⚠️ **Kurs — reja, holat emas**: qachon yuborishni mo'ljallaganini kurs
  aytadi, yuborilgan-yuborilmaganini `FiredAt`. Chekka "ikkinchi kurs ketdi"
  deb yozish — qatorlar allaqachon biladigan narsa haqida yolg'on gapirishi
  mumkin bo'lgan ikkinchi joy.
- `POST /fire` endi ixtiyoriy `course` oladi; **tanasiz so'rov — hammasi**,
  ya'ni kurslardan oldingi har bir ekran o'zgarishsiz ishlaydi.
- Panelda kurs kutayotgan bo'lsa har biriga alohida tugma, **faqat bittadan
  ko'p kurs kutayotgan bo'lsa**: hammasini yuborish odatiy holat va katta
  tugmani saqlaydi.

### Qatorlarni boshqa chekka ko'chirish (ПЕРЕНОС)
`POST /staff/checks/{id}/lines/move` — belgilangan qatorlar boshqa ochiq chekka.
- ⚠️ Qatorlar **hamma narsasi bilan** ko'chadi (yuborilgani yuborilgan bo'lib
  qoladi, izoh, mehmon): qayta yaratish oshxonaga ikkinchi kechki ovqatni
  buyurtma qilardi.
- ⚠️ **Bekor qilingan qator ko'chmaydi** — u shu chekda hisobdan chiqarilgan
  ovqatning yozuvi, va uni olib ketish aybni ham olib ketadi.
- ⚠️ Manzil chek **o'sha filial filtri** bilan yuklanadi: aks holda ofitsiant
  boshqa filialning chek id'sini yozib, taomni boshqa binodagi hisobga
  ko'chirardi.
- ⚠️ Manba yozilgandan keyin manzil yozilmasa **qatorlar qaytariladi**: yarim
  bajarilgan ko'chirish ovqatni ikkala hisobdan ham yo'qotadi.

### Chegirma foizda (СКИДКА %)
Restoran chegirmani foizda kelishadi ("xodimlar stoliga o'n foiz"), kassa esa
summa yozishi kerak — 216 000 lik chekdan 10% ni qo'lda hisoblash navbat
oldida bajariladigan arifmetika. Ikkala maydon bir-birini yangilaydi; **simda
summa ketadi**, chunki mehmon aynan shuni to'laydi va hisobot shuni qo'shadi.

### Bronlar kassada (РЕЗЕРВОВ)
`GET /staff/reservations` — bugungi, hali oldinda turgan bronlar, filial
bo'yicha. ⚠️ **Bron telefonda kelishiladi va kechqurun soat yettigacha yashab
qolishi kerak**: uni panelga qo'ng'iroqni ko'targan odam yozadi, kerak
bo'ladigan odam esa ikki soatdan keyin kassa oldida turadi. Band stolga
kirgizilgan mehmon — stoli bor restoran eshigidan qaytarilgan kompaniya.
Ekranda **tasma**, sahifa emas; broni yo'q restoranda umuman chizilmaydi.

### Testlar
Go: `TestLineEditAfterFiring`, `TestApplyLineEditGuestAndCourse`,
`TestMergeKeepsGuestsAndCoursesApart`. Frontend: mehmon tabi taomni qayerga
qo'yishi, ikki mehmonning bir xil taomi ikki qator bo'lishi, bir kursni
yuborish qolganiga tegmasligi. Jami 34 test.

### Hali yo'q
- **Пречек / chop etish** — printer ishi to'xtatilgan (Wails), shu sababli
  hech qanday chop etish tugmasi qo'yilmadi.

---

## 2026-08-18 — Kassa o'z filialini o'qiydi, sozlamalar tuzatildi ✅

### ⚠️ 1. Sozlamalar sahifasi butunlay yiqilardi
`TypeError: Cannot read properties of null (reading 'filter')` — `ZonesEditor`.
Sabab — **Go'ning nil slice tuzog'i, shu strukturada uchinchi marta**:
`booking.zones` Mongo'da yo'q → JSON'da `null` → tahrirlagich `null.filter`.
- Sahifa **buzilib ishlamadi, balki yiqildi**: bo'lim ham, maydon ham
  nomlanmagan holda butun ekran "Oshxonada nimadir noto'g'ri ketdi" bo'lardi,
  va qolgan barcha sozlama zal chizilmaguncha yetib bo'lmaydigan bo'lib qolardi.
- ⚠️ Sahifada himoya **bor edi va ishlamasdi**: `zones: []` standart qiymatdan
  keyin `...(rest.booking ?? {})` Mongo'ning `null` ini **ustiga qaytarardi** —
  `??` yo'q `booking` ni qo'riqlaydi, uning **ichidagi** `null` ni emas.
- Serverda qoida `bookingSlices()` ga ajratildi va **hamma yo'lda** qo'llanadi:
  `/admin/branches`, `/restaurant`, `/restaurant?raw=1`. ⚠️ To'liq
  `bookingSettings()` emas: u slot uzunligi va plan o'lchamini ham to'ldiradi —
  zal chizadigan ekran uchun to'g'ri, hujjatni **saqlaydigan** sahifa uchun
  noto'g'ri (ochilgan har bir filialga standart qiymatlar yozilib ketardi).
- Testda muhrlangan: `TestBookingSlicesAreNeverNull`.

### ⚠️ 2. Kassa boshqa filialning zalini chizardi
Kassa va zal `GET /restaurant` dan o'qirdi — u esa "bu **mehmon** qaysi
filialdan xizmat olyapti" degan savolga javob beradi (sayt standarti yoki
cookie). Kassa filialga **tokeni bilan** tegishli, ya'ni ikki filialli
kompaniyada Yunusobod peshtaxtasi Chilonzorning zal sxemasini chizardi: stollar
soni to'g'ri, shakli to'g'ri, **binosi boshqa**. Hech nima buzuq ko'rinmasdi va
birinchi alomat ofitsiantning 7-stolni topa olmasligi bo'lardi.
- Yangi `GET /staff/branch` — nomi, valyutasi va **o'z** zal sxemasi.
  ⚠️ Bu yerda plan `bookingSettings()` orqali (o'lchamlari to'ldirilgan holda)
  keladi: bu ekran zalni **chizadi**.

### 3. Peshtaxta raqamlari sozlamalardan
Sozlamalar → Stol bron qilish da **"list" turidagi zona** (saboy) — aynan
peshtaxta: 100–130 raqamlari, stol emas, shuning uchun joy soni ham,
koordinatasi ham yo'q. Endi kassa ularni **zal panjarasida emas, peshtaxta
tasmasida** chizadi, va tasma har uch ko'rinishda ko'rinadi.
⚠️ Sxemada ular umuman chizilmaydi: koordinatasi yo'q, ya'ni hammasi 0,0 da
uyulib qolardi.

---

## 2026-08-18 — Hisob (precheck) va chop etish ✅

### Chop etishni brauzer qiladi, server emas
Printer drayveri hali yo'q (Wails ilovasi to'xtatilgan, `pos-reja.md` §2),
lekin **monoblokda chek printeri oddiy Windows printeri** bo'lib turadi.
Shuning uchun ekran serverdan **tayyor qatorlarni** so'raydi va o'zi chop
etadi (`lib/print.ts`).
- ⚠️ **Qog'ozni server chizadi** (`internal/receipt`) — aynan sozlamalardagi
  ko'rinishni chizadigan kod. Brauzerda ikkinchi joylashuv yozilsa, ular
  ertami-kechmi ajrab ketadi va farqni **qo'lida chek ushlagan mehmon**
  topadi.
- ⚠️ **Yashirin iframe, yangi oyna emas**: popup standart holatda bloklanadi va
  blok **jimgina** bo'ladi — kassir "chop etish" bosadi, hech nima bo'lmaydi.
- ⚠️ `@page { size: 58mm auto }` — chek printeri uzluksiz lentaga bosadi, A4
  sahifa qutisi chekni bo'sh varaqning burchagiga qo'yardi.
- ⚠️ Matn `textContent` bilan qo'yiladi: "&lt;b&gt;" deb nomlangan taom — taom,
  belgilash emas.

### Hisob (precheck) — alohida hujjat turi
`receipt.Precheck`: bir xil taomlar, bir xil jami — va **ayni shu xavfli**.
- ⚠️ **Fiskal chekka o'xshamasligi shart**: fiskal chekka o'xshagan qog'ozni
  olgan mehmonga sotuv ro'yxatdan o'tgani aytilgan bo'ladi, aslida esa yo'q.
  Qog'ozda **"HISOB — fiskal chek emas"** yozuvi bor, va u eganing tahrir
  qiladigan footer'iga qo'shilmagan: hujjatni halol qiladigan yagona jumla
  qog'ozni tejash uchun o'chiriladigan sozlama bo'lishi mumkin emas.
- ⚠️ **"To'landi" va "qaytim" yo'q, va ular renderer'da tozalanadi**,
  chaqiruvchiga ishonilmaydi: karta rad etilgandan keyin qayta hisob
  chiqarilgan chek allaqachon berilgan pulni ko'tarib yuradi.
- Testda muhrlangan: `TestPrecheckCannotBeMistakenForTheReceipt`.

### ⚠️ Hisob so'ralgani — stolning uchinchi holati
`check.precheckAt` — iiko zal sxemasidagi uchinchi rang. Hisob so'ragan stol
na "ovqatlanyapti", na "ketdi": u **kartochka mashinasi bilan qaytib borish
kerak bo'lgan** stol, va shu paytgacha buni faqat chekni chop etgan odam
bilardi. Zal sxemasida, plitkada va ofitsiant kartochkasida ko'k bo'lib
chiziladi.
- **Bayroq emas, vaqt belgisi** (`readyAt` bilan bir sabab): "yigirma daqiqa
  oldin so'radi" va "hozir so'radi" — boshqa vaziyat.
- ⚠️ Hisob **yoshdan ustun**: hisob so'ragan stolning qirq daqiqasi
  ovqatlanayotgan stolning qirq daqiqasidan boshqa muammo.
- Qayta chop etish vaqtni **surmaydi**: qog'ozini yo'qotgan mehmon qaytadan
  kuta boshlagani yo'q.

### Endpoint
`POST /staff/checks/{id}/print` `{kind}` → `{lines, widthMM, check}`.
⚠️ Notanish tur **jimgina mijoz chekiga tushmaydi** (400): u fiskal belgini
tashiydi, va ekrandagi xato tufayli ro'yxatdan o'tgan sotuvni da'vo qiladigan
qog'oz chiqmasligi kerak. Hisob **POST va yozadi**; qolgan uchtasi faqat
o'qiydi.

---

## 2026-08-18 — 1024×768: yozuvlar bir-birining ustiga chiqardi ✅

Mijoz aynan monoblok o'lchamida sinab ko'rdi va ikkala ekranda ham matn
qalashib ketgani chiqdi. Brauzerda 1024×768 da takrorlab, sababini topdim.

### ⚠️ Ildizi bitta: kassa boshqaruvlari **o'ralardi**
Tugma matni bir so'z uzun bo'lsa, u tugmani **qisqartirmasdan** uch qatorga
cho'zardi — va tugma o'z panelining ostidan chiqib, tagidagi narsaning ustiga
tushardi. 1024 px da:
- pastki amal paneli (`h-[4.25rem]`) beshta to'liq jumlani ko'tarolmay,
  hammasi taomlar panjarasining ustiga oqib chiqqan edi;
- zal sarlavhasidagi "Mening stollarim / Hamma stollar" ikki qatorga bo'linib
  sarlavhadan chiqib ketgan;
- to'lov chiplaridagi "Karta (terminal)" o'z tugmasidan tashqariga chiqqan.

Endi `.till-btn*`, `.till-seg`, `.till-chip-btn` — hammasi
**`whitespace-nowrap`**, va qoida CSS izohida yozib qo'yilgan: sig'maydigan
yorliq **qisqartiriladi**, sig'maydigan qator **suriladi**.

### Qisqa yorliqlar (to'liq nomi tooltip va `aria-label` da qoladi)
Pastki panel: Hisob · Ko'chirish · Qatorlar · Bekor · Kassa.
Zal sarlavhasi: Meniki · Hammasi. Chek paneli: Naqd · Karta · O'tkazma.
⚠️ Uzunlik tooltipda **bepul**, 1024 px li qatorda esa emas.

### ⚠️ Sarlavhadagi "rol" — aslida ruxsat tavsifi edi
Ism ostida `t.roles.hints.cashier` chiqarilgan: *"To'lovni qabul qilish va
chekni yopish"* — bu rollar sahifasidagi jumla, va u to'g'ri soatning ustiga
chiqib ketgan. Endi bitta so'z: **Kassir / Ofitsiant**.
Ism va rol ikkalasi ham `truncate` va kengligi cheklangan: xodim nomi erkin
matn, va uzuni soat bilan qulf tugmasini ekrandan chiqarib yuborardi.

### Boshqa tuzatishlar
- **Smena 1024 px da ham ko'rinadi**: jumla yashiriladi, **nuqta va soat
  qoladi** — qaysi smenaga sotayotganini ko'rmagan kassir buni hisob-kitobda
  biladi, va o'shanda javob "farq" bo'ladi.
- Taom plitkasida narx va "so'm" **yonma-yon** (ilgari plitka ikki chetiga
  tarqalib, orasidagi bo'shliq raqamdan keng edi).
- Mehmon tablari o'ng chetdan qirqilmaydi.

### Tekshiruv
Brauzerda 1024×768 da: kassa (zal sxemasi, menyu, chek), zal (sxema,
ofitsiantlar) — qalashish yo'q. `npm test` 35/35 ✓ · `tsc` ✓ · `next build` ✓ ·
lint toza.

---

## 2026-08-18 — Printerlar: ESC/POS, LAN · USB · COM ✅

Mijoz "hamma printer va hamma ulanish turi" dedi. Bajarildi — brauzer oynasi
zaxira bo'lib qoldi, asosiy yo'l esa **haqiqiy ESC/POS**.

### ⚠️ Chop etishni agent bajaradi, chunki boshqa iloji yo'q
Server ma'lumot markazida, printer esa oshxonadagi javonda. Brauzer soket ocha
olmaydi, serverdan restoran tarmog'iga yo'l yo'q. **Fiskal agent** allaqachon
o'sha kompyuterda ishlaydi va serverdan ish so'raydi — chop etish **o'sha
navbatga** qo'shildi (`kind: "print"`).
- ⚠️ **Baytlarni server quradi**, agent bir qadam tashiydi va yozadi: joylashuv,
  kod sahifasi va kesish qoidasi — hammasi shu yerda, testda. Restoranda
  qarovsiz ishlaydigan dasturning logini hech kim o'qimaydi.
- ⚠️ **Chop etish fiskaldan oldin beriladi**: oshxona cheki — hali pishirilmagan
  taom va stolda o'tirgan mehmon; fiskal hujjat esa bir daqiqadan keyin ham
  qabul qilinadi va qayta so'raladi.
- ⚠️ Ish **olinadi, o'chirilmaydi** (`takenAt`): Windows yangilanishi yoki tok
  uzilishi — va o'chirilgan ish oshxona ko'rmagan chek bo'lardi. 2 daqiqadan
  keyin qayta beriladi, 3 urinishdan keyin **sababi bilan** qoladi.

### ESC/POS (`internal/escpos`) — va uning haqiqiy zaif joyi
⚠️ **Alifbo.** Termal printerda Unicode yo'q: u tanlangan kod sahifasidagi 256
belgini biladi. To'g'ri yozilgan **oʻ va gʻ** (U+02BB / U+2018) hech bir
sahifada yo'q — va ular chek shablonining **standart matnida** bor. Endi
`Fold()` ularni ASCII apostrofiga aylantiradi (qolgan tirnoq, tire va uch nuqta
ham). ⚠️ `formatPrice` minglarni **uzilmas bo'sh joy** bilan ajratadi — u ham
CP437 da yo'q, ya'ni **har bir chekdagi har bir summa** o'rtasida "?" bilan
chiqardi. Testda muhrlangan.
- Kirill uchun **CP866** (`ESC t 17`), lotin uchun CP437.
- Pul yashigi **kesishdan oldin** ochiladi: sekin printerda kesish oxirgi
  bo'ladi, kassirning qo'li esa allaqachon yashikda.
- Kesuvchisi yo'q printer kesish buyrug'ini **chop etadi**, shuning uchun
  kesish — sozlama.
- Fiskal QR (`GS ( k`) — matn ayta olmaydigan yagona narsa.

### Ulanish turlari (`internal/printer`) — hammasi, drayversiz
| Yozuv | Nima |
|---|---|
| `tcp://192.168.1.50:9100` yoki `192.168.1.50` | tarmoq (port yozilmasa 9100) |
| `usb://XP-58` | Windows printer ulashuvi (`\\localhost\XP-58`) |
| `\\KASSA-PC\XP-58` | Windows'dagi ko'rinishidan nusxa |
| `serial://COM3`, `com://COM10` | COM port (⚠️ COM10 dan yuqorisi `\\.\` talab qiladi — klassik nosozlik) |
| `device:///dev/usb/lp0` | Linux USB |
⚠️ **Yangi bog'liqlik yo'q**: hammasi oddiy soket yoki fayl yozuvi. `winspool`
cgo orqali bo'lsa agent Linux'da qurilmasdi va imzolanadigan ikkinchi narsa
paydo bo'lardi.

### Sozlamalarda
Chek dizayni sahifasida **Printerlar** bo'limi: nomi, **bitta manzil qutisi**,
nima chiqarishi (oshxona · hisob · kassa · mijoz), alifbo, kesish, pul yashigi,
nusxa soni, vaqtincha o'chirish.
- ⚠️ **Yangi printer hech nima chiqarmaydi**: manzilni yozib, nima
  chiqarishini tanlamay ketgan odam butun binoning cheklarini pass'ning
  rulosiga yubormasligi kerak.
- ⚠️ **"Sinov cheki" — sahifadagi eng foydali tugma** (SMS sahifasidagi bilan
  bir sabab): manzil to'g'ri yozilgan bo'lib, printer o'chiq, boshqa
  quyi tarmoqda yoki boshqa nom bilan ulashilgan bo'lishi mumkin — formadan
  bu to'rttasi bir xil ko'rinadi, farqi soat sakkizda bilinadi.
- Manzil **saqlashda tekshiriladi**, oshxona kutayotganda emas.

### Oshxona cheki o'zi chiqadi
"Oshxonaga yuborish" bosilganda kitchen ticket navbatga tushadi — printer
sotib olishning butun sababi shu. ⚠️ **Faqat shu bosishda yuborilgan qatorlar**:
yigirma daqiqadan keyingi ikkinchi kurs starterlarni qayta chiqarsa, oshpaz
ularni yana pishiradi va qog'ozda buni aytadigan hech nima yo'q.
⚠️ Navbatga **yozuvdan keyin** qo'yiladi: ulanmagan printer buyurtmaning
oshxona ekraniga tushmasligiga sabab bo'lmasligi kerak.

### Tekshiruv
Soxta tarmoq printeri (9100 da tinglovchi) ga haqiqiy ish yuborildi: 326 bayt,
`ESC @` → CP437 → matn (`Lag'mon`, `Ko'k choy`, `92 000` — hammasi o'qiladi) →
pul yashigi → qismli kesish → QR. `go test ./internal/...` ✓ · `npm test` 35/35
✓ · `tsc` · lint · `go vet` toza.

### Hali yo'q
- **Windows spooler orqali to'g'ridan-to'g'ri** (printer ulashilmagan bo'lsa):
  hozir printer `net share` bilan ulashiladi yoki tarmoq/COM ishlatiladi.
- Chop etish navbatini paneldan ko'rish (nima chiqmadi va nega).

### Chekda restoran logotipi (o'sha kuni)
- **Rastr bilan yuboriladi, printerga saqlanmaydi.** ESC/POS da "NV logo" bor —
  printerning o'z fleshiga vendor dasturi bilan, har printerga alohida, uning
  oldida turib yuklanadi; logotipini almashtirgan restoran o'sha dasturni
  qaytadan qidirardi. `GS v 0` esa har chekka bir necha kilobayt turadi va
  hamma modelda bir xil ishlaydi.
- ⚠️ **Bir nuqta — bir bit, va logotipdagi butun muammo shu.** Termal kalla
  nuqtani yo yoqadi, yo yo'q; kulrang yo'q. Shuning uchun **Floyd–Steinberg**
  dithering: oddiy chegara qo'yish yumshoq chetlarni zinapoyaga, gradientni esa
  yo qora blokka yo hech nimaga aylantiradi — ya'ni logotip yo'qoladi.
- ⚠️ **Kenglik butun baytga yaxlitlanadi**: sakkizga bo'linmaydigan kenglik
  birinchisidan keyingi har bir qatorni suradi — bu ozgina tor logotip emas,
  **diagonal chizilgan dog'**.
- ⚠️ **Shaffof fon — oq.** Logotip ko'pincha shaffof fonli PNG bo'lib saqlanadi;
  "rangi yo'q" ni qora deb o'qish belgisi o'yib olingan qora to'rtburchak
  chiqarardi.
- ⚠️ **Balandligi cheklangan** (480 nuqta): poster yuklagan odam har chekka
  ketadigan rulo uzunligini ko'rmaydi.
- ⚠️ **Oshxona chekida hech qachon chiqmaydi**, sozlama qanday bo'lishidan
  qat'i nazar — pass'dagi har bir nuqta vaqt va qog'oz, va oshpazga qaysi
  restoranda ishlashini aytish shart emas. Sozlamalarda ham **ko'rsatilmaydi**:
  hech nima qilmaydigan tugma qolgan tugmalarga ham ishonchni yo'qotadi.
- **Sozlama har chek turida alohida** (`template.logo`), standart holatda
  **o'chiq**: logotip faqat qanday chiqishini kimdir ko'rgandan keyin
  yaxshilanish bo'ladi.
- **Rastr keshlanadi** (URL + kenglik bo'yicha, 30 daqiqa): band juma — mingta
  chek, va har biri uchun PNG dekod qilish umumiy serverda bekorga sarf.
- ⚠️ **Rasm topilmasa chek baribir chiqadi**: yuklanmagan logotip, o'chirilgan
  fayl, WebP/SVG (printer o'qiy olmaydi) — hammasi logotipsiz chek beradi.
  Yo'q rasm hech qachon mehmonning hisobiga turmasligi kerak.
- ⚠️ **Yo'l `uploads` ichida ekani tekshiriladi**: URL ega tahrirlay oladigan
  hujjatdan keladi.
- Brauzer zaxira yo'li logotipni **rasm sifatida** chizadi (HTML chop etadi),
  ya'ni u yerda dithering ham, rastr ham kerak emas.

### ⚠️ "To'lash tugmalari chiqmayapti" — chiqayotgan edi (o'sha kuni)
Mijoz kassada to'lov tugmalarini ko'rmadi. Brauzerda takrorlandi: tugmalar
**o'sha yerda** edi — chek bo'sh bo'lganda `disabled`, va `opacity-40` da
och sarg'ish fon oq panelda **yo'q** bo'lib o'qiladi.
- Nosozlik "o'chirilgan" emas, "mavjud emas" bo'lib ko'rinishida: sabab
  aytilmagan bo'lsa, odam ekranni buzuq deb hisoblaydi.
- Endi `till-btn-accent:disabled` — **60%** (bor, lekin hali emas), va tagida
  sabab: *"Avval taom qo'shing — to'lash uchun chek bo'sh"*.
- ⚠️ Kassir bo'lmagan xodimga (faqat ofitsiant) tugmalar **umuman
  chizilmaydi** — bu ataylab; endi u holatda ham bitta qator yoziladi:
  *"To'lovni faqat kassir qabul qiladi"*. Bo'sh joy "sizga ruxsat yo'q" degani
  emas, "kassa buzuq" degani bo'lib o'qilardi.

### To'lovdan keyin chek o'zi chiqadi (o'sha kuni)
Kassir "To'lovni tasdiqlash" bosganda ikkita chek navbatga tushadi: **kassa
nusxasi** (pul yashigini ochadigan) va **mijoz cheki** — fiskal belgisi va QR
bilan.
- ⚠️ **Mijoz cheki yopilishda emas, fiskal javob kelganda chiqadi**: belgi
  o'shanda paydo bo'ladi, va u — chekdagi mehmon **tekshira oladigan** yagona
  narsa. Yopilishda chiqarilsa, qog'ozda aynan shu qism bo'lmasdi.
- ⚠️ **Kassasi yo'q restoranda esa darhol** (`skip`): kutadigan narsa yo'q, va
  peshtaxtada puli qo'lida turgan odam bor.
- ⚠️ **Kassa rad etsa ham chiqadi**: sotuv bo'lib o'tdi va odam turibdi.
  Fiskal tomoni — restoranning ishi (fiskallashtirilmagan sotuvlar
  ogohlantirishi buni allaqachon nomlaydi), mehmonniki emas.
- ⚠️ **Bir sotuv — bir chek** (`check.receiptAt`): navbatga ikki joydan
  qo'yiladi (yopilish va fiskal), va rad etilgan fiskalni qayta yuborish
  ikkinchisini yana chaqiradi. Bir ovqatga ikki qog'oz — mehmonning "qaysi
  biri haqiqiy?" degan savoli. Testda muhrlangan.
- **Kassa nusxasi birinchi**: pul yashigi o'shanda ochiladi, ya'ni pul hali
  kassirning qo'lida turganda.
- To'langan ekranda **"Chekni chiqarish"** tugmasi: printeri yo'q filialda
  brauzer chiqaradi, va eshik oldida yana bitta so'ragan mehmon uchun.

---

## 2026-08-18 — Oflayn: serverdagi shartnoma (1-qadam) 🚧

`pos-reja.md` §6 ning **server tomoni**. Klient (Wails yoki brauzer) hali
tanlanmagan, lekin bu qism ikkalasiga ham bir xil kerak — shundan boshlandi.

### ⚠️ Butun oflayn xavfsizligi bitta maydonda
`order.clientId` — sotuvni **kassa o'zi** nomlaydi, server ko'rishidan oldin.
Yuboruvchi qayta uradi (birinchi urinish yetib bordimi — bilolmaydi), va
kassaning o'z id'siz ikkinchi urinish **ikkinchi kechki ovqat** bo'lardi: ikki
marta hisoblangan, kunlik tushumda ikki marta sanalgan, oshxona ekrani qarab
tursa ikki marta pishirilgan.
- **Unique + sparse indeks** (`clientId`) — kafolatni beradigan narsa kod emas,
  aynan shu indeks. Onlayn sotilgan har bir chekda bu maydon yo'q, va ular
  aksariyat.
- `POST /staff/checks/sync` — bir so'rovda 50 tagacha chek; har biriga alohida
  javob (`id`, `number`, `duplicate`, yoki `error`). ⚠️ **Xato bo'lgan chek
  sababi bilan bir marta rad etiladi**: buzuq chekni abadiy qayta yuboradigan
  navbat orqasidagi yaxshi cheklarni hech qachon yetkazmaydi.

### ⚠️ Narxlar — bu yerda kassaniki, va bu yagona to'g'ri joy
Boshqa hamma joyda "planshet qaysi taomni aytadi, narxni server aytadi" —
chunki mehmon hali to'lamagan. Bu yerda **to'lagan**: pul yashikda, chek
cho'ntagida. Bir soatdan keyin kimdir tahrirlagan menyu bo'yicha qayta hisoblash
kunlik tushumni kassadagi naqd bilan qarama-qarshi qo'yardi.

### ⚠️ Soat — eng jimgina buziladigan joy
Eski monoblokda CMOS batareyasi o'lgan bo'lsa, svet o'chib yonganda sana
**yillarga orqaga** ketadi va kassa butun kechani restoran mavjud bo'lmagan
yilga yozadi — hech bir ekran buni aytmaydi, lekin u hisobotga, smena
hisobiga va fiskal chekda **soliq hujjatiga** yetib boradi.
`clampOfflineTime`: kelajak emas, ikki haftadan eski emas; tashqarisi —
"biz eshitgan payt" (ko'rinadigan darajada noto'g'ri, ko'rinmaydigan darajada
emas). ⚠️ Bir daqiqalik siljish **saqlanadi**: kassa soatlari sekundlarga
og'adi, va ularni "hozir"ga tortish aynan oldini olmoqchi bo'lgan narsani
qilardi — butun kechani sinxronizatsiya daqiqasiga ko'chirish.

### Vaqt belgilari haqiqiy
- `createdAt` / `statusHistory` — chek **ochilgan** va **yopilgan** payt;
- ⚠️ `queuedAt` — **oshxonaga aytilgan** payt (`firstFired`), sinxronizatsiya
  payti emas: "stol qancha kutdi" degan har bir hisobot shuni o'qiydi, va
  hozirgi vaqtni yozish butun kechki pishirishni bir zumda bo'lgandek
  ko'rsatardi;
- ⚠️ `check.receiptAt` **to'ldirilgan holda** keladi: qog'oz restoranda soatlar
  oldin chiqqan. Bo'sh qoldirilsa, ulanish qaytgan daqiqada butun kechaning
  cheklari birdan chop etilardi.
- Fiskal belgi ham qabul qilinadi: ⚠️ kassa **lokal**, ya'ni internetsiz ham
  sotuv ro'yxatdan o'tadi — bu qismning kutishi shart emas.

### Keyingi qadam — klient tanlanishi kerak
`pos-reja.md` §8 tartibida oflayn **Windows ilovasidan keyin** turadi
(SQLite + WAL + `synchronous=FULL`), chunki brauzer diskka ishonchli yoza
olmaydi va lokal agentga ulana olmaydi (mixed content / private network / CORS
— fiskal agent aynan shuning uchun **tashqariga** ulanadi).

### B: brauzerda oflayn — to'lov yo'qolmaydi (2-qadam)
Klient tanlovi: **avval B (brauzer), keyin A (Wails)**.

⚠️ **Kassada yo'qotib bo'lmaydigan yagona payt — pul qo'l almashgan payt.**
Qolgan hamma narsa tarmoqni kutishi mumkin: taom qo'shish, stol ko'chirish,
chek chop etish. To'lov kuta olmaydi — naqd yashikda, mehmon ketdi, va
yopilmagan sotuv **hech kim o'tirmagan stolda ochiq chek** bo'lib qoladi,
smena hisobida esa o'zining butun summasi qadar farq beradi.

- `lib/offline/store.ts` — IndexedDB, **bog'liqliksiz** (20 qator). ⚠️ Brauzer
  kassa apparati emas: IndexedDB qayta yuklashdan, yiqilgan tabdan omon
  qoladi, **svet o'chishidan** kafolat bermaydi — shuning uchun haqiqiy oflayn
  kassa baribir Wails + SQLite (A bosqichi). Bu qism restoranda haftada bir
  necha marta bo'ladigan uzilishni yopadi: wifi tushdi, provayder, bizning
  deploy.
- ⚠️ **Qayta urinish — yopish, yangi sotuv emas.** Chek serverda allaqachon bor
  (stol ochilganda, tarmoq soz paytda yaratilgan). Uni "oflayn sotuv" sifatida
  yuborish **bir ovqatni ikki marta** yozardi. Navbat *niyatni* saqlaydi — shu
  chek, shu usul, shu chegirma — va server "yopildi" degunicha takrorlaydi.
- ⚠️ **409 / 404 — muvaffaqiyat.** Navbat to'lishining odatiy sababi: server
  pulni oldi, javob qaytishda yo'qoldi. Buni xato deb o'qish to'langan sotuvni
  navbatda abadiy ushlab turardi — va smena oxirida buni o'qigan odam
  qo'rqardi.
- ⚠️ **`navigator.onLine` — boshqa savolga javob**: u "kabel yoki wifi bormi"
  deydi, restoran routeri esa internet uzilganda ham "bor" deb turadi; access
  point almashganda esa bir soniyaga "yo'q" bo'lib, kassir oldida banner
  chaqnaydi. Till **o'z so'rovlari** yetayotganini o'lchaydi: har 15 soniyadagi
  chek so'rovi — eng arzon halol javob.
- ⚠️ **Saqlab bo'lmasa — ochiq aytiladi**: qulflangan brauzer, private oyna,
  to'la disk. Bunda yagona xavfsiz qadam odamniki: *"aloqa qaytguncha chekni
  yopmang"*.
- Testda: tarmoq xatosi bilan rad javobini ajratish, to'lovning qurilmada
  saqlanishi va aloqa qaytganda yuborilishi, va 409 dan keyin navbatning
  bo'shashi. ⚠️ Testlar **ketma-ket** yuritiladi (`fileParallelism: false`):
  navbat — butun yugurish uchun umumiy baza.
- Yo'l-yo'lakay: chek panelidagi tugma **"To'lash"** bo'ldi — dialogdagi
  "To'lovni tasdiqlash" bilan bir xil nom ikkita bo'lib qolgandi, va bir
  ekranda bir xil nomli ikki tugma telefonda tushuntirib bo'lmaydigan tugma.

### Keyingi: B ning ikkinchi yarmi va A
- **B2**: oflayn holda **yangi chek ochish** (hozir mavjud chek yopiladi) —
  `POST /staff/checks/sync` allaqachon tayyor va idempotent.
- **A**: Wails ilovasi + SQLite (WAL, `synchronous=FULL`) + printer/yashik.

### B2: oflaynda chek ochish (o'sha kuni)
Endi server yo'q paytda **stol ochish, taom qo'shish, oshxonaga belgilash va
to'lash** — hammasi shu qurilmada. Aloqa qaytganda sotuv **bir butun** bo'lib
`POST /staff/checks/sync` orqali topshiriladi.

- ⚠️ **Stol rad etilmaydi, ochiladi.** Mehmonlar o'tirishdi; wifi tushgani
  uchun buyurtmani boshlay olmaydigan kassa — yonida qog'oz daftar turadigan
  kassa, va daftar hech qachon hisobotga tushmaydi.
- ⚠️ **Lokal chek — server cheki emas, va ekran shuni aytadi.** Ikki fakt
  odamlarga yetadi: **oshxona ekrani uni ko'rmaydi** (kurs belgilash faqat shu
  yerda yoziladi — ofitsiant borib aytadi), va **stop list tekshirilmaydi**
  (ikki kassa oxirgi porsiyani ikki marta sotishi mumkin — reja buni ataylab
  qabul qiladi: sotmaydigan kassa sotib bo'lmaydigan mahsulot).
- ⚠️ **Butun sotuv yuboriladi, uni yasagan qadamlar emas.** Oflayn ochilgan chek
  serverda hech qachon bo'lmagan — takrorlash uchun narsa yo'q. Qadamlarni
  yuborish serverdan "qurilmadan ochiq chek" qabul qilishni talab qilardi: bu
  stolga egalik qilishning ikkinchi yo'li, va ikki kassa bitta stolni o'ziniki
  deb bilgan kunning birinchi nosozligi.
- ⚠️ **Chek raqami** lokal `OFF-XXXX` bilan chiqadi; serverda **band bo'lsa
  yangisi beriladi** — mehmon cho'ntagidagi qog'oz uchun eskisi saqlanadi,
  lekin ikki oflayn kassa bir xil raqam yasasa, biri ikkinchisining sotuvini
  jimgina o'chirib yuborardi (buni faqat yo'qolgan sotuv bilan bilib bo'lardi).
- **Narx qurilmadagi menyudan** — u bir daqiqa oldin serverdan kelgan va aynan
  mehmonga aytilgan narx.
- **Birlashtirish qoidasi bir xil**: bir taomni to'rt marta bosish — bitta
  qator, soni 4. Aks holda oflayn qurilgan chek onlayn qurilganidan boshqacha
  o'qilardi.
- Oflaynda **taklif qilinmaydi**: chek chop etish, stolni ko'chirish, chekni
  sababi bilan bekor qilish, kurs bo'yicha yuborish — bulari serverning
  hukmini yoki qog'ozni talab qiladi. Rad etib emas, **ko'rsatmasdan**.

### ⚠️ Yo'l-yo'lakay topilgan ikki xato
- **Chek so'rovi lokal chekni ekrandan o'chirardi**: poll serverning
  ro'yxatida yo'q ochiq chekni tozalaydi (to'g'ri — uni boshqa birov yopgan),
  lekin lokal chek u yerda **hech qachon** bo'lmaydi — ya'ni kassir stol
  ochgandan bir poll keyin chek yo'qolardi.
- **Testlarda bazani o'chirish keyingi testni buzardi**: ochiq ulanish borida
  `deleteDatabase` **bloklanadi** va keyinroq — allaqachon boshqa test
  ishlayotganda — bajariladi. Endi bazaning ichi tozalanadi. Alomat: kassadagi
  xatoga o'xshagan tasodifiy yiqilish.

## Zal va peshtaxta sotuvlari paneldа (`/admin/checks`)

Savol shundan boshlandi: dashboardning «Buyurtmalar» bo'limida faqat onlayn
zakazlar ko'rinadi — shundaymi, va zal/saboy sotuvlari qayerda?

**Ha, va bu ataylab**: `AdminListOrders` da `check: {$exists: false}` turadi —
ochiq stollar bilan to'lgan zal kimdir qabul qilishi kerak bo'lgan yetkazish
buyurtmalarini ko'mib yuborardi. Ajratuvchi maydon **`check`, `type ==
"dinein"` emas**: stol QR'idan o'z telefoni bilan buyurtma bergan mehmon ham
`dinein` beradi va u ro'yxatda **qolishi shart**.

**Pul hech qachon yo'qolmagan.** `ordersInRange` faqat davr va qamrov bo'yicha
filtrlaydi, ya'ni kassa sotuvi birinchi kundan beri savdo, kanal va moliyaviy
hisobotlarda. Endi bu **testda muhrlangan** (`TestReportsStillCountTillSales`):
ikki ekrandan birini ishlayotgan odamning eng tabiiy keyingi qadami —
«hisobotlarni ham moslashtirish», ya'ni o'sha istisnoni ko'chirish, va u
restoranning o'z tushumidan butun zalni jimgina olib tashlardi.

Yo'q bo'lgani — **oradagi ro'yxat**: ega seshanba 4.2 mln bo'lganini o'qiy
olardi-yu, *qaysi sotuvlar ekanini* ko'ra olmasdi — har bir smena bahsi shu
savoldan boshlanadi.

- **Sahifa alohida, tab emas**, va birinchi qatorida ikkinchi yarmi qayerdaligi
  yozilgan: bitta «buyurtma» so'zining ikki xil to'plamini ko'rsatgan ikki ekran
  — birov ulardan birini pul haqida yolg'on gapiryapti deb xulosa qiladigan yo'l.
- **Davr `createdAt` bo'yicha kesiladi** — hisobotlar bilan bir xil maydon.
  `closedAt` jozibaliroq (23:50 da ochilib 00:20 da to'langan chek), lekin oyi
  hisobotnikidan boshqacha chegaralangan ro'yxat — bitta savolga ikki javob.
- **Jamilar butun filtrlangan to'plam bo'yicha**, ekrandagi sahifa bo'yicha
  emas: «Keyingi» bosilganda o'zgaradigan jami — aynan nusxa olinadigan raqam.
- **Ochiq stol pul olmagan**: uning joriy summasi sotuvga ham, o'rtacha chekka
  ham kirmaydi. Bir mehmonga to'g'ri keladigan summa hech kim mehmon sanamagan
  bo'lsa **nol** — «kimdir eslab qolgan stollar» o'rtachasi emas.
- **Bekor qilingan (void) taom sotilmagan**: chekda qoladi, sanoqqa kirmaydi.
- Filtrlar: davr, holat (hammasi/ochiq/yopilgan), joy (zal/peshtaxta), qidiruv
  (chek raqami, stol, ofitsiant). ⚠️ Ikkala segment ham **imzolangan** —
  yonma-yon turgan va birinchi varianti bir xil («Hammasi») ikki boshqaruv
  bitta buzuq boshqaruv bo'lib o'qiladi.
- Jadval **o'z qutisida suriladi** (`min-w-[720px]`): 1024 px'li monoblokda
  sakkiz ustun bir-birining ustiga chiqishi — bu panel allaqachon bir marta
  yeb ko'rgan nosozlik.

### Chekni ochish, chop etish va PDF
- Ro'yxatdagi qatorga bosilsa **chek kartochkasi** ochiladi (drawer): vaqtlar va
  kim ochgani/yopgani, to'lov turi, qatorlar (variant, izoh, mehmon/kurs),
  oraliq jami → chegirmalar → jami, fiskal belgi yoki kassaning xatosi.
- ⚠️ **Bekor qilingan (void) qator ko'rinadi** — ustidan chizilgan, sababi, kim
  bekor qilgani va **tayyorlangan-tayyorlanmagani** bilan. Izsiz void —
  restorandan pul olib chiqishning eng eski usuli, shuning uchun qator hujjatda
  qoladi; faqat tirik qatorlarni yuborish bu ekranni **nohalol chek bilan
  kelishadigan** qilardi. Qog'ozda esa yo'q (mehmon yemagan taom).
- ⚠️ **Alohida endpoint**, `adminOrder` emas: buyurtma hujjatida stol uchun
  mijoz/manzil/kuryer maydonlari bo'sh yoki ma'nosiz, kassaning o'z faktlari
  (void, mehmon raqami, kurs) esa panelning `Order` shaklida umuman yo'q.
- **Chop etish/PDF — brauzerda** (`lib/print.ts`, kassadagi bilan bir helper),
  chunki panelni ochgan odam odatda binoda emas; brauzerning o'z oynasida
  «PDF sifatida saqlash» ham shu yerda. **Kassa printeriga yuborish — alohida
  tugma**: hech kim turmagan peshtaxtadan chiqqan qog'oz eng yaxshi holatda
  chalkashlik.
- ⚠️ **Chekni server chizadi** (`receipt.Render`, mijoz shabloni bilan) —
  kassadagi bilan **bir xil layout**. Ikkinchi joyda chizilgani ertami-kechmi
  ajrab ketadi, va farqni qog'ozni ushlab turgan mehmon topadi.
- `_id` yolg'iz hech qachon hujjat tanlamaydi: `scopedOrderFilter` + `check`
  mavjudligi, qamrovdan tashqarisi ham, kassa cheki bo'lmagan buyurtma ham
  bir xil **404**. Har chop etish amallar jurnalida (`check.print`).

### Hisobni bo'lish (split) va pulni qaytarish

**Bo'lish** (`POST /staff/checks/{id}/split`) — mehmon raqamlari **yig'ilardi-yu
ishlatilmasdi**: ofitsiant har qatorni kim buyurtma qilganiga belgilay olardi,
lekin ovqat oxirida stolga ikki chek berishning yagona yo'li — **hech kim
buyurtma bermasdan oldin** ikki chek ochish, ya'ni stol o'tirgan payt kim nima
yeyishini taxmin qilish edi. Restoranlar bu hisobni qog'ozda qilishining sababi
shu.
- **Ofitsiantniki, kassirniki emas**: pul ko'chmaydi va chekdan hech nima
  ayrilmaydi. Zal ichidagi eng oddiy so'rov uchun kassirni chaqirish — kassir
  PIN'ini hammaga aytish bilan tugaydi.
- Qoidalar qator ko'chirishdan olingan: **void qator hech qachon ko'chmaydi**
  (u aynan shu chekdan hisobdan chiqarilgan taomning yozuvi), **pishayotgan
  taom yangi chekni ham oshxonaniki qiladi**, va manba yozuvi yiqilsa
  yaratilgan yarim **o'chiriladi** — bir taom uchun ikki marta pul olish
  bo'linish umuman bo'lmaganidan yomonroq.
- ⚠️ **Hammasini bo'lib bo'lmaydi** (400): bo'sh chek qoladi, uni hech kim
  to'lay olmaydi va zal ekranida buyurtma kutayotgan stolga o'xshab turadi.
- ⚠️ **Bo'linish ikkinchi sotuv emas** (`check.splitFromId`): to'rt chek so'ragan
  kompaniya bitta kechki ovqat yegan. Sotuvlar sahifasi **stollarni** sanaydi,
  qog'ozlarni emas — aks holda kecha bandroq ko'rinadi va o'rtacha chek stol
  haqiqatda sarflagan summaning to'rtdan biriga tortiladi. Pul esa to'liq
  sanaladi.
- UI: mavjud "Taomlarni ko'chirish" oynasida birinchi manzil — **"Yangi chek"**.
  Undan oldingi savol bir xil: qaysi taomlar.

**Qaytarish** (`POST /admin/checks/{id}/refund`) — ⚠️ **sotuv qoladi**. Uni
bekor qilish ikki jihatdan yolg'on: ovqat pishirilgan va yeyilgan (oshxonaning
kechasi, mahsulot, ofitsiantning ishi — hammasi bo'lgan), va yo'qolgan sotuv
sababni ham o'zi bilan olib ketadi. O'zgargani — pul, shuning uchun pul
yoziladi (`order.refund`: kim, qachon, qancha, **sabab majburiy**).
- ⚠️ **Topilgan jimgina xato**: `received()` `status == delivered` bo'lsa pulni
  sanardi, kassa cheki esa yopilishi bilan `delivered` bo'ladi — ya'ni
  qaytarilgan stol **tushumda qolib ketardi**. Yetkazishda bu ko'rinmasdi
  (u yerda `paymentStatus` `paid` dan chiqadi va buyurtma o'zi tushib qoladi).
  Jonli tekshirildi: qaytarishdan keyin tushum aynan 42 000 ga kamaydi.
- ⚠️ **Kassa qoldig'i faqat oldingi smenadagi sotuv uchun tuzatiladi**
  (`correctDrawer`): joriy smenaning kutilgan summasi "shu smenada to'langan
  naqd sotuvlar"dan quriladi, ya'ni bugungi sotuv `paid` dan chiqishi bilan pul
  o'zi ayriladi — yana yozuv qo'shish **ikki marta** ayirardi. Kartaga qaytarish
  yashikka umuman tegmaydi.
- **Faqat to'liq qaytarish**: qisman qaytarish qator bo'yicha miqdor, mavjud
  chegirmalar bilan hisob va aynan o'sha qism uchun fiskal qaytarish talab
  qiladi — bularning yarmi hali yo'q, va "bitta taom" ni jimgina "butun stol"
  ga aylantirgan xususiyat yo'qligidan yomonroq.
- Panelda: chek kartochkasida sabab maydoni (tasdiq oynasi emas — "ishonchingiz
  komilmi?" bosiladi, to'ldirilishi shart maydon esa odamni nima bo'lganini
  aytishga majbur qiladi). Ro'yxatda **"Qaytarilgan"** nishoni va ustidan
  chizilgan summa; KPI'da alohida qator ("sotuv tushdi" va "ikki stolga pul
  qaytardik" — boshqa-boshqa kechalar).
- ⚠️ **Bekor qilingan chek endi hech narsa sifatida sanaladi**: summasi
  hujjatda qolgani uchun u sotuvga qo'shilib ketardi, mehmonlari esa qamrovga.

### Windows printerga `net share` siz chop etish
Kassa monoblokidagi USB chek printeri Windows'da oddiy o'rnatilgan printer, va
unga fayl yo'li orqali yetish uchun uni **ulashish** (`net share`) kerak edi —
Windows 10/11 da bu tarmoq aniqlanishi, ba'zan parol so'rovi va ba'zan
restoran o'zgartira olmaydigan siyosat degani. Ya'ni butun integratsiyaning eng
mo'rt joyi eng oddiy printerda edi.
- Endi **printerning o'z nomi spooler orqali** so'raladi (`winspool.drv`,
  `syscall.NewLazyDLL` bilan — **cgo yo'q**, ya'ni agent hamon Linux'dan
  cross-compile qilinadi). Share yo'li **zaxira** bo'lib qoldi.
- ⚠️ **Sozlamani hech kim qayta yozmaydi**: `usb://XP-58` allaqachon nomni
  tashiydi — u ilgari `\\localhost\XP-58` ichiga qo'shilib **yo'qolardi**.
  Endi `Target.Name` bo'lib saqlanadi.
- ⚠️ **Datatype "RAW"**: boshqasi baytlarni drayverga beradi, u esa ESC/POS'ni
  hujjat deb chizmoqchi bo'ladi — natija bir modelda to'g'ri, keyingisida
  boshqaruv kodlari bosilgan varaq.
- ⚠️ **`usb://SERVER/XP-58` — boshqa kompyuterning printeri**: bu mashinaning
  spooleri u haqda hech nima bilmaydi, so'rash bitta tushunarli xatoni ikkita
  chalkash xatoga aylantirardi. Faqat `\\SERVER\XP-58` yo'li ishlatiladi.
- Spooler rad etsa sabab **logga** yoziladi va share bilan urinib ko'riladi
  (ikkinchi urinishning xatosi birinchisining sababini o'chirmasligi kerak).
- Spooleri yo'q tizimda (Linux, test) `errNoSpooler` darhol qaytadi — bu xato
  emas, va logga yozilmaydi.

### Kassa X/Z hisoboti va xizmat haqi

**X va Z** — bitta qog'oz, ikki savol. X smena o'rtasida o'qiladi va **hech
nimani o'zgartirmaydi** ("hozir qancha sotildi, kassada qancha bo'lishi
kerak"), Z esa kunni yopadi, kutilgan summani muzlatadi va ega saqlaydigan
qog'oz bo'ladi. Ikkalasini bitta sarlavha ostida chiqarish — X'ni kun yakuni
deb topshirish imkonini berardi, va bu restorandagi aniq bo'lishi shart bo'lgan
yagona hujjat.
- X — **GET** (`/staff/cash-shift/report`): uni soat to'rtda shubha bilan
  ochgan odam necha marta bossa ham eng yomoni qog'oz sarflaydi.
- Z — **yopish javobida qaytadi**, alohida tugma emas: yashikni yopib, keyin
  "chop etishni unutmang" degan ekran — Z hisoboti umuman bo'lmagan kunlar
  demakdir.
- ⚠️ **Sotuv va yashik — ikki alohida blok**: sotuv to'lov turlari bo'yicha,
  yashik esa qoldiq + naqd sotuv + kuryer topshirig'i ± qo'lda kirim/chiqim.
  Kartadagi sotuv birinchisida bor, ikkinchisida yo'q — faqat bittasini
  ko'rsatgan hisobot kassirni ayblash uchun ishlatiladigan hisobot.
- ⚠️ **Sotuv cheklardan sanaladi**, smenadagi hisoblagichdan emas: sotuv
  yopilganda oshib boradigan raqam bir marta ikki marta yozilsa yoki jarayon
  qayta ishga tushsa siljiydi, va buni faqat Z hisoboti hisobotlarga zid
  kelganda bilib bo'ladi — o'shanda ikkalasiga ham ishonib bo'lmaydi.
- ⚠️ **Qaytarilgan qatori nol bo'lsa ham chiqadi**: qatorning yo'qligi "hech
  nima qaytarilmagan" dan farq qilmaydi, va yashikni tekshirayotgan odam aynan
  shu raqamni qidiradi. Taomlar ro'yxati esa **yo'q** — bu pul hisoboti, va
  soat ikkida ikki yuz qator lenta hech kim ikkinchi marta o'qimaydigan hisobot.

**Xizmat haqi** (`branch.service`) — ⚠️ **filialga tegishli, kompaniyaga emas**:
zanjirning ofitsiantli restorani xizmat haqi oladi, savdo markazidagi
peshtaxtasi olmaydi, va bitta raqam ikkalasiga ham qo'yilsa olib ketiladigan
kofega xizmat haqi qo'shiladi — bu xususiyatning aynan mehmonlar shikoyat
qiladigan ko'rinishi.
- ⚠️ **Faqat stolga** (`tableId` bor bo'lsa) — buni sozlama bila olmaydi, kassa
  biladi.
- ⚠️ **Foiz stol o'tirganda chekka ko'chiriladi** (`order.servicePercent`),
  to'lov paytida o'qilmaydi: soat sakkizda foizni o'zgartirgan restoran
  allaqachon ovqatlanayotgan stollarni qayta narxlamasligi kerak, va keyingi oy
  qayta chop etilgan chek mehmon **to'lagan** summani aytishi shart. Chegirmani
  chekka nom va summa bilan ko'chirish bilan bir qoida.
- ⚠️ **Chegirmadan keyin hisoblanadi**: 20% chegirma olib, keyin to'liq
  summadan xizmat haqi to'lagan mehmon — o'ziga berilgan chegirma uchun pul
  to'layapti, va u bu chekni eng diqqat bilan o'qiydi.
- ⚠️ **Yaxlitlash bitta joyda** (`serviceOn`, yarimdan yuqoriga): ikki joyda
  yaxlitlash — panel, qog'oz va yashik bir so'mga farq qilishi, va bir so'm —
  odam butun kechani izlaydigan narsa.
- ⚠️ **Oflayn sotuvga qo'shilmaydi**: qurilma nimani olgan bo'lsa o'sha yozildi,
  mehmon ketgan. Keyin foiz qo'shish yashikda bo'lmagan pulni yozib, kamomadni
  kassirning muammosiga aylantirardi. (Foiz oflayn do'konga qo'shilgach
  yopiladi.)
- Ekranda ham, hisobda ham, chekda ham **alohida qator, foizi bilan**: jamiga
  qo'shib yuborilgan xizmat haqi — dunyo bo'ylab restoran cheklariga eng
  ko'p bildiriladigan e'tiroz, va mehmon qo'lida javob bera oladigan yagona
  hujjat turibdi.

### Stolning shakli kechqurun o'zgaradi: bo'lish zalda, birlashtirish ikkalasida

**Bo'lish endi ofitsiant ekranida ham.** Bu qaror bo'linishning o'z qoidasiga
zid edi: bo'lish — **ofitsiantning** ishi, plastinkalar yig'ilganda stolda hal
qilinadi, lekin u faqat kassada bor edi — ya'ni so'ralgan odam peshtaxtaga
borib **boshqa birovdan** buni so'rashi kerak edi. Endi `/zal` da ham
"Taomlarni ko'chirish / Yangi chek" oynasi bor (bir xil komponent — ikki nusxa
ertami-kechmi ikki xil qoida bo'ladi).
- ⚠️ Manzil ro'yxatida **hamma ochiq cheklar**, faqat shu ofitsiantniki emas:
  yonidagi stol bilan hisobni bo'layotgan yoki qo'shayotgan mehmon kimning
  uchastkasi ekanini bilmaydi, zalning standart ko'rinishi esa "meniki" —
  ya'ni manzillarning yarmi ko'rinmay qolardi.

**Birlashtirish** (`POST /staff/checks/{id}/merge`) — bo'lishning ikkinchi
yarmi va xuddi shunday oddiy kecha: peshtaxtadagi ikki do'st stolga o'tadi,
juftlikka to'rt kishi qo'shiladi, ikki stol tug'ilgan kun uchun surib
qo'yiladi. Bu bo'lmasa ofitsiant bir chekni ovoz chiqarib o'qib, ikkinchisiga
qayta yozadi — vaqtlar, kurslar va iz yo'qoladi, oshxona pishirib bo'lgan taom
esa qayta yuboriladi.
- ⚠️ **Yutilgan chek bekor qilinadi, o'chirilmaydi**: unda void qatorlar,
  allaqachon chop etilgan bo'lishi mumkin bo'lgan raqam va kim ochgani bor.
  O'chirish uchalasini ham yo'q qiladi; bekor qilingan hujjat esa o'zini
  tushuntiradi — va bekor qilingan chek hech qayerda sanalmagani uchun kechaning
  tushumi va qamrovi to'g'ri qoladi.
- **Sabab qaysi chekka ketganini nomlaydi** (`birlashtirildi → HBQR-UF8K`):
  yolg'iz "bekor qilindi" bir oydan keyin hech kim harakat qila olmaydigan
  javob.
- ⚠️ **Mehmonlar qo'shiladi** (2 + 3 = 5): surib qo'yilgan ikki stol ikkala
  davrani ham o'tqazadi, va "bir mehmonga" — zal yuritiladigan ikki raqamdan
  biri.
- ⚠️ **Void qatorlar joyida qoladi**: ular yozilgan chekning yozuvi, va ularni
  ko'chirish aybni o'sha taomni hech qachon olib tashlamagan ofitsiantning
  chekiga ko'chirardi.
- ⚠️ **Avval manzil yoziladi, keyin manba bo'shatiladi**: ikkinchi yozuv
  yiqilsa taom ikki chekda bo'ladi va odam ikkalasini ham ko'radi; teskarisi
  esa uni **hech qayerda** qoldirardi.
- Jonli tekshirildi: 2 qator, 5 mehmon, 122 000; yutilgani `cancelled`.

### Chop etish navbati paneldan ko'rinadi
⚠️ **Chiqmagan chek — tizimdagi eng jim nosozlik.** Boshqa hamma narsa kimgadir
ko'rinadi: fiskal chek ketmasa ogohlantirish chiqadi, kassaga tushmagan
buyurtma qizil nishon oladi, karta to'lovi yiqilsa mehmon peshtaxtada turadi.
Oshxona cheki chiqmasa esa **hech qanday iz qolmaydi**: buyurtma ekranda,
sotuv hisobotlarda, yagona alomat — yigirma daqiqadan keyin hech kim
pishirmagan taom, va uni kutayotgan odam sezadi.

Navbat har urinishni va printerning **o'z so'zlarini** birinchi kundan yozib
kelgan. Ularni hech kim o'qiy olmasdi.
- `GET /admin/print-jobs` — oxirgi sutka (`?hours=`, 200 qator chegarasi).
  ⚠️ Vaqt bo'yicha chegaralangan: navbat biznes bilan emas, **trafik bilan**
  o'sadigan yagona kolleksiya, va hammasini o'qiydigan ekran restoran yaxshi
  ishlagani sari sekinlashadi.
- ⚠️ **Avval chiqmaganlari**: vaqt bo'yicha saralangan ro'yxat "bugun nima chop
  etdik" degan savolga javob beradi, buni esa hech kim so'ramaydi. Savol —
  "nima **chiqmadi**", va band kechada bu to'rt yuz qator orasidagi uchtasi.
- Sanoq **nosozliklarniki**, navbat uzunligi emas ("412 topshiriq" — printerlar
  bandligi, muammo emas).
- `POST /admin/print-jobs/{id}/retry` — ⚠️ **allaqachon chiqqan topshiriq qayta
  yuborilmaydi** (404): "chiqmadi" degan savolga sotuvning o'z qayta chop etish
  tugmasi javob beradi va u yangi hujjat quradi; tugagan topshiriqni jimgina
  qayta yuborish mehmonga ikki chek, oshxonaga ikki ticket beradi — va oshxona
  ikkalasini ham bajaradi.
- ⚠️ **Saqlangan baytlar qayta yuboriladi, hujjat qayta qurilmaydi**: narx
  o'zgargandan keyin qayta chizilgan chek — mehmon to'lagan hujjat emas.
- Joyi: **Sozlamalar → Printerlar ostida**. "Nega hech nima chiqmadi" deb
  so'ragan odam allaqachon shu yerda, o'zi yozgan manzilga qarab turadi;
  alohida sahifani esa uni izlagan odam topadi, ya'ni hech kim.

### Oflaynda ham xizmat haqi (yuqoridagi cheklov yopildi)
Xizmat haqi kiritilganda oflayn yo'lda **ataylab qoldirilmagan** edi: qurilmada
foiz yo'q edi, va server keyin qo'shsa yashikda bo'lmagan pulni yozardi. Lekin
natijasi shu bo'lardi: **bir xil stol wifi ishlaganiga qarab ikki xil summa
to'laydi**, va kam to'lagani buni hech qachon bilmaydi. Endi foiz qurilmaga
yetkaziladi.
- `GET /staff/branch` javobiga `servicePercent` qo'shildi — qurilma stol
  ochilganda uni **lokal chekka ko'chiradi** (serverdagi bilan bir qoida: soat
  sakkizda foiz o'zgarsa, allaqachon o'tirgan stol qayta narxlanmaydi).
- Brauzerdagi `serviceOn` — serverdagining aynan nusxasi (chegirmadan keyin,
  yarimdan yuqoriga yaxlitlash). ⚠️ Import qiladigan joy yo'q, qoida Go'da
  yashaydi; ikki nusxani halol ushlab turadigan yagona narsa — ikkalasini bir
  xil raqamlar bilan tekshiradigan test.
- ⚠️ **Sim orqali foiz ketadi, summa emas**: kassa **nima olganini** biladi
  (chekni chop etib pulni olgan — server hozirgi sozlamani qo'ysa mehmon
  ko'rmagan summani yozardi), server esa **arifmetikani** biladi, ya'ni
  uzilishdan qismlari qo'shilmaydigan sotuv chiqa olmaydi.
- 0–100 dan tashqaridagi foiz **tashlanadi**, qisqartirilmaydi: bu yaxlitlash
  bo'yicha kelishmovchilik emas, hech kim ishlatmasligi kerak bo'lgan payload.
