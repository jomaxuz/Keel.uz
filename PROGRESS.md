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

## Keyingi qadamlar 📋

**1. Git**: repozitoriyada hali birorta commit yo'q — hamma narsa untracked.
   Hozirgi `.gitignore` da faqat `.env` va OS shovqini bor; `node_modules/`,
   `.next/`, `uploads/` va Go binarlari **yo'q**. Birinchi commit'dan oldin
   to'ldirish shart.

**1b. Ochiq qolgan yagona brend savoli**: domen/bo'lim qarori — bitta domenda
   ikki bo'lim (`/restoran`, `/somsa`) yoki ikki domen. Hozircha bitta domen +
   brend cookie'si ishlaydi; qaror mijoz bilan kelishilgach kerak bo'ladi.

**2. Prod tayyorgarligi**: `SMS_PROVIDER=demo` prod'da qolmasligi (demo kodni
   API javobida qaytaradi), `JWT_SECRET`, admin paroli — `DEPLOY.md` ro'yxati.

**3. Docker build sinovi**: `docker-compose.prod.yml` bilan uchdan-uchgacha
   (`next/font/google` build vaqtida internet talab qiladi).

### Ochiq savollar (mijoz uchun)
- Map API key kim oladi? (2GIS, har deploy uchun kerak — bepul olinadi)
- To'lov: hozircha qo'lda (naqd/Payme/Click/Uzum tanlovi) — haqiqiy online
  to'lov integratsiyasi kerakmi?

---

## Ishga tushirish eslatmasi (ertaga davom etganda)

```bash
# Backend (lokal mongo ishlayotgan bo'lsa):
cd backend && cp .env.example .env && go run ./cmd/server
# Admin: seed default admin/admin123, lekin bu deploy’da `yujo` ga o‘zgartirilgan

# Frontend (tugatilgach):
cd frontend && npm install && npm run dev
```

**Eslatmalar / qarorlar:**
- **Single-tenant**: har bir restoran alohida deploy (multi-tenant emas).
- Rasmlar backend `uploads/` papkasida, `/uploads/*` orqali serve.
- Kuryer real-time tracking hozircha yo'q (CLAUDE.md 7-bo'lim).
- Pul birligi UZS, butun son.
- Buyurtma narxi serverda qayta hisoblanadi (client'ga ishonmaymiz).
