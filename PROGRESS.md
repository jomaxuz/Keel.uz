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
