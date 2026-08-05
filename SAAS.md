# SAAS.md — bitta serverda ko'p restoran (C varianti)

Bu hujjat hozirgi **single-tenant** mahsulotni (har restoranga alohida VPS)
o'zgartirmasdan, uning ustiga **obunali SaaS** qatlamini qanday qo'yishni
tavsiflaydi.

Asosiy qoida: **kod bitta qoladi.** Bitta env bayrog'i (`TENANT_MODE`) ikki
mahsulotni ajratadi. Katta mijoz "o'z serverimda tursin" desa — bugungi
`docker-compose.prod.yml` o'zgarishsiz ishlayveradi.

---

## 1. Qaror va sabab

| | A: har mijozga VPS | B: bitta jarayon, `tenantId` | **C: ilova umumiy, baza alohida** |
|---|---|---|---|
| Kod o'zgarishi | yo'q | juda katta | **kichik** |
| Ma'lumot sizishi xavfi | yo'q | **yuqori** | yo'q |
| 50 mijoz narxi | 50 VPS | 1 VPS | **1 VPS** |
| Ops murakkabligi | yuqori | past | o'rta |

**B tanlanmadi**, chunki loyihada ~30 kolleksiya va yuzlab so'rov bor: bitta
unutilgan `tenantId` filtri — bir restoran ikkinchisining buyurtmalarini
ko'radi. Bu turdagi xato SaaS'ni bir kunda o'ldiradi. C variantida bunday xato
**fizik jihatdan mumkin emas**: har tenant o'z bazasida, o'z jarayonida.

C ning narxi — har tenantga bitta kichik Go konteyneri (~40–60 MB). Bu arzon,
chunki RAM'ni yeydigan qism (Next.js, ~400 MB) **umumiy** bo'lib qoladi.

---

## 2. Umumiy sxema

```
   osh.uz ────┐
 milliy.uz ───┼──▶  Caddy  (TLS + Host bo'yicha yo'naltirish)
 xanum.uz ────┘       │
                      ├── / ..............▶ Next.js (BITTA jarayon)
                      │                        │ SSR
                      ├── /api/*  ─────────▶ backend-osh    ─┐
                      ├── /uploads/* ──────▶ backend-milliy  ├─▶ MongoDB
                      │                      backend-xanum  ─┘   (t_osh,
                      └── control panel ───▶ control-plane        t_milliy,
                                                                  t_xanum)
```

- **Brauzer faqat o'z domenini ko'radi** — CORS ham, mixed-content ham yo'q
  (hozirgi holat shunday, `lib/api.ts:82-86`).
- **Caddy `/api` va `/uploads` ni to'g'ridan-to'g'ri tenant backendiga** beradi.
  SaaS rejimida Next.js'ning `rewrites()` i ishlatilmaydi (u statik, tenant
  ro'yxati esa o'zgarib turadi).
- **Portlar umuman ochilmaydi**: konteynerlar docker tarmog'ida nomi bilan
  topiladi (`backend-osh:8080`). Port taqsimlash muammosi yo'q.

---

## 3. Komponentlar

### 3.1 Caddy — TLS va yo'naltirish

nginx emas, Caddy tanlandi: **on-demand TLS**. Mijoz o'z domenini sizning IP'ga
yo'naltiradi va sertifikat **avtomatik** olinadi — har domen uchun certbot
buyrug'i yozib o'tirilmaydi.

```caddyfile
{
    on_demand_tls {
        # HAR SO'ROV UCHUN EMAS, har yangi domen uchun. Bu qator bo'lmasa
        # istalgan odam domenini sizga yo'naltirib Let's Encrypt limitini
        # kuydiradi.
        ask http://control:9000/internal/tls-ask
    }
    admin :2019
}

# Tenant xaritasi — control plane generatsiya qiladi, keyin `caddy reload`.
(tenants) {
    map {host} {backend} {
        osh.uz            backend-osh
        www.osh.uz        backend-osh
        milliy.uz         backend-milliy
        default           ""
    }
}

https:// {
    import tenants
    tls { on_demand }

    # Noma'lum yoki to'xtatilgan domen
    @unknown expression {backend} == ""
    handle @unknown {
        reverse_proxy control:9000
    }

    @api path /api/* /uploads/*
    handle @api {
        reverse_proxy {backend}:8080
    }

    handle {
        reverse_proxy frontend:3000
    }
}
```

`caddy reload` — **uzilishsiz**; yangi tenant qo'shish uchun restart kerak emas.

### 3.2 Next.js — bitta jarayon, hamma domen

Frontend holatsiz: qaysi domendan kelganini `Host` dan biladi va o'sha
tenantning backendiga so'rov yuboradi. **Bitta build hamma mijozga xizmat
qiladi** — chunki brauzer allaqachon nisbiy `/api/v1` ga murojaat qiladi.

### 3.3 Backend — har tenantga bitta konteyner

Bugungi image, faqat env boshqa:

```
MONGO_DB=t_osh                 # yagona ajratuvchi
JWT_SECRET=<har tenantga alohida tasodifiy>
UPLOAD_DIR=/app/uploads        # volume: ./data/osh/uploads
PUBLIC_BASE_URL=https://osh.uz
CORS_ORIGINS=https://osh.uz
ADMIN_USERNAME / ADMIN_PASSWORD   # birinchi owner
TZ=Asia/Tashkent
```

Handler kodida **bitta qator ham o'zgarmaydi**. `restaurant` singleton,
`payment_settings`, `kioskSecret`, `pbx_settings` — hammasi o'z bazasida.

Har tenantga alohida `JWT_SECRET` — qo'shimcha himoya qatlami: bir tenantning
tokeni ikkinchisida umuman tanilmaydi.

### 3.4 MongoDB — umumiy server, baza alohida

`t_<slug>` nomli baza. Mongo bazani birinchi yozuvda o'zi yaratadi.
Zaxira nusxa `mongodump --db t_osh` — **har mijozni alohida** tiklash mumkin,
bu A variantida ham yo'q edi.

### 3.5 Control plane — yangi kichik xizmat

Sizda hali yo'q bo'lgan yagona qism. Vazifasi:

- tenant CRUD (yaratish, to'xtatish, o'chirish, domen qo'shish)
- Caddyfile generatsiya qilish + reload
- konteyner ko'tarish/to'xtatish (Docker socket orqali)
- `/internal/tls-ask` — domen bizniki bo'lsa 200, aks holda 404
- `/internal/resolve?host=` — frontend SSR uchun `host → slug`
- to'lanmagan/noma'lum domen uchun sahifa
- buyurtmalarni sanash va hisob chiqarish

Til: Go (bitta binar, sizda allaqachon Go bor).

---

## 4. Kod o'zgarishlari (aniq)

### 4.1 `frontend/src/lib/api.ts` — SSR tomonini tenant'ga moslash

Hozir (82-85-qatorlar) `API_URL` — **modul darajasidagi const**, ya'ni jarayon
umriga bitta qiymat. Bu single-tenant uchun to'g'ri, SaaS'da esa hamma so'rov
bitta backendga ketardi.

Barcha chaqiruvlar `req()` (257) va `uploadImage()` (1358) dan o'tgani uchun
o'zgarish **shu ikki joyda**:

```ts
// SaaS: SSR qaysi domen so'raganini bilishi kerak; brauzer tomoni o'zgarmaydi
// (u doim o'z origin'iga — /api/v1 ga — murojaat qiladi).
async function apiBase(): Promise<string> {
  if (typeof window !== "undefined")
    return process.env.NEXT_PUBLIC_API_URL ?? "/api/v1";
  if (process.env.TENANT_MODE !== "saas")
    return process.env.INTERNAL_API_URL ?? "http://localhost:8080/api/v1";

  const { headers } = await import("next/headers");
  const host = (await headers()).get("host") ?? "";
  return `http://backend-${await slugFor(host)}:8080/api/v1`;
}
```

`slugFor(host)` — `control:9000/internal/resolve` ga so'rov, natijasi
**xotirada keshlanadi** (masalan 60 soniya). Standart subdomen
(`osh.sizningdomen.uz`) uchun umuman so'rov kerak emas — slug host'ning o'zida.

⚠️ `headers()` faqat so'rov kontekstida ishlaydi. `api.ts` da `API_URL` const
sifatida eksport qilingan — uni funksiyaga aylantirish `import` qilgan
joylarni ham tekshirishni talab qiladi (hozir faqat shu fayl ichida
ishlatiladi, ya'ni xavfsiz).

### 4.2 `frontend/next.config.ts`

`rewrites()` SaaS rejimida bo'sh qaytaradi — `/api` va `/uploads` ni Caddy
yo'naltiradi. Single rejimda hozirgicha qoladi.

`images.remotePatterns` ga tenant domenlari kerak bo'lishi mumkin — hozir
rasmlar nisbiy `/uploads` dan kelgani uchun ehtimol tegilmaydi, tekshirish
kerak.

### 4.3 Backend

Kod o'zgarmaydi. Faqat `docker-compose.prod.yml` yonida **`docker-compose.saas.yml`**
paydo bo'ladi: caddy + frontend + control + mongo (tenant konteynerlarini
control plane yuritadi, compose emas).

### 4.4 O'zgarmasligi tekshirilishi kerak bo'lgan joylar

- **To'lov callback'lari** (`/api/v1/payments/payme` va h.k.) tenant domeniga
  keladi → Caddy Host bo'yicha to'g'ri backendga beradi. Kalitlar o'z bazasida.
  ✅ ishlaydi, lekin **birinchi tenantda `cmd/paytest` bilan tekshirilsin**.
- **PBX webhook** (`/pbx/onlinepbx/{token}`) — xuddi shunday.
- **Kiosk QR** (`kioskSecret`) — tenant bazasida, muammo yo'q.
- **Cookie'lar** (`lang`, `brand`, `branch`) domen bo'yicha alohida. ✅
- **2GIS kaliti** (`NEXT_PUBLIC_MAP_API_KEY`) endi **umumiy** — bitta build.
  ⚠️ Kalit domen bo'yicha cheklangan bo'lsa, hamma tenant domenini kalit
  sozlamalariga qo'shish kerak (yoki cheklovni olib tashlash).

---

## 5. Yangi tenant ochish oqimi

```
1. control: tenant yozuvi (slug, domenlar, tarif, holat=trial)
2. control: JWT_SECRET generatsiya, papka ./data/<slug>/uploads
3. control: docker run --name backend-<slug> --network saas \
            -e MONGO_DB=t_<slug> ... restaurant-backend:latest
4. backend boot: seed.Bootstrap → owner hisobi + namuna menyu (48 taom)
                 EnsureBrandAndBranch → brend + filial
                 EnsureIndexes, migratsiyalar
5. control: Caddyfile map'ga qator + caddy reload
6. mijozga: manzil + login + vaqtinchalik parol
```

**4-qadam allaqachon yozilgan** (`cmd/server/main.go:44-62`) — shuning uchun
tenant birinchi soniyadanoq to'liq ishlaydigan sayt bo'ladi. Provisioning'ning
qolgani — ~150 qator Go.

Domen: standart `<slug>.sizningdomen.uz` (wildcard DNS). Mijoz o'z domenini
ulasa — A yozuvini sizning IP'ga qaratadi, Caddy sertifikatni o'zi oladi.

---

## 6. To'lov va to'xtatish

**Hisoblash**: har tenant bazasida oylik buyurtmalar soni.

```js
db.order.countDocuments({
  createdAt: { $gte: oyBoshi, $lt: keyingiOy },
  status: { $ne: "cancelled" }
})
```

⚠️ **Qaror kerak**: bekor qilingan buyurtma hisoblanadimi? Tavsiya — **yo'q**
(dashboard statistikasi ham ularni pulga qo'shmaydi). Aks holda mijoz o'zi
bekor qilgan buyurtma uchun to'laydi va bu birinchi janjal mavzusi bo'ladi.
Xuddi shu sabab test buyurtmalari uchun ham yo'l qoldirish kerak.

**To'lanmasa**: `status=suspended` → Caddy map "to'lov kutilmoqda" sahifasiga
yo'naltiradi, konteyner **to'xtatiladi** (RAM bo'shaydi). **Baza o'chirilmaydi** —
to'lov kelganda konteyner qayta ko'tariladi va hammasi joyida. Ma'lumotni
o'chirish faqat qo'lda, N oydan keyin.

---

## 7. Yangilanish oqimi

```
git push main → CI image quradi → control plane:
  har tenant konteynerini navbat bilan yangi image bilan qayta ko'taradi
  (migratsiyalar boot'da o'zi ishlaydi)
```

Navbat bilan — chunki barchasi birdan qayta ishga tushsa Mongo'ga bir vaqtda
50 ta migratsiya uriladi. Frontend bitta bo'lgani uchun u bir marta yangilanadi.

**Rollback**: eski image tegi bilan qayta ko'tarish. Migratsiyalar oldinga
qarab yozilgani uchun (qo'shadi, o'chirmaydi) bu xavfsiz — shu qoidani
saqlash kerak.

---

## 8. Resurs hisobi (taxminiy)

| Komponent | RAM |
|---|---|
| Next.js (umumiy) | ~400 MB |
| Caddy + control | ~100 MB |
| Backend × 50 | ~2.5 GB |
| MongoDB | 2–4 GB |
| **Jami** | **~6–7 GB** |

Ya'ni **8 GB VPS ≈ 50 restoran**. 1000 so'm/buyurtma va o'rtacha 400
buyurtma/oy bilan bu ≈ 20 mln so'm/oy tushum, server xarajati ≈ 500 ming.

**~50 tenantda** Mongo'ni alohida VPS'ga chiqarish kerak bo'ladi.
**~150–200 tenantda** konteyner soni ops yuki bo'ladi — o'sha paytda B
variantiga (bitta jarayon, so'rov bo'yicha baza tanlash) o'tiladi. Bu o'tish
`Handler.Store` ni `h.store(r)` ga aylantirish — mexanik, lekin o'sha paytda
pul ham, sabab ham bo'ladi. **Hozir qilinmaydi.**

---

## 9. Bosqichlar

| | Ish | Taxminiy |
|---|---|---|
| **S1** | Caddy + umumiy frontend + 2 ta tenant **qo'lda** ko'tariladi. `api.ts` o'zgarishi. Ikki domenda ikki restoran ishlashini isbotlash. | 1–2 kun |
| **S2** | ✔️ **BAJARILDI** — tenant CRUD, Docker provisioning, Caddy render + reload, `/internal/resolve`, `TENANT_MODE=saas`. | bajarildi |
| **S3** | `tls-ask` + o'z domenini ulash oqimi + noma'lum domen sahifasi. | 1 kun |
| **S4** | Buyurtma hisoblagichi, hisob-faktura, `suspended` holati. | 2–3 kun |
| **S5** | Mijoz o'zi ro'yxatdan o'tishi + sinov muddati. | 3–5 kun |
| **S6** | Rolling update skripti + har tenantga zaxira nusxa (cron `mongodump`). | 2 kun |

**S1 dan keyin arxitektura isbotlangan bo'ladi** — qolgani avtomatlashtirish.
Birinchi 3–5 mijozni control plane'siz, qo'lda ochish mumkin.

---

## 10. Ochiq savollar

1. **SMS**: Eskiz hisobi umumiymi yoki har restoranga alohida? Umumiy bo'lsa
   arzon va oson, lekin SMS matnida restoran nomi turishi kerak va limit
   umumiy bo'ladi. Hozirgi `SMS_PROVIDER` env — tenant konteyneriga
   beriladi, ya'ni ikkalasi ham mumkin. **Prod'da `demo` qolmasin.**
2. **2GIS kaliti** domen bo'yicha cheklanganmi?
3. **Bekor qilingan buyurtma hisoblanadimi** (6-bo'lim).
4. **CPU cheklovi**: bitta shovqinli tenant hammani sekinlashtirmasligi uchun
   konteynerlarga `--cpus` qo'yiladimi?
5. **Ko'p brend qarori** (PROGRESS.md 1b) SaaS'da yanada muhim: bitta tenant
   ichida ikki brend — bitta domenda bo'limmi yoki ikki domen?
