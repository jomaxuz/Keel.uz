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

Konteynerga yana ikki qiymat beriladi (`CONTROL_URL`, `CONTROL_TOKEN`) — egasi
o'z domenini o'zi ulashi uchun; 7.2 ga qarang.

⚠️ **Tenant serveri root emas** (`app`, uid 10001). Entrypoint root bo'lib
faqat uploads papkasini `app` ga o'tkazadi va darhol `su-exec` bilan tushadi:
bind mount'ni **Docker root egaligida yaratadi** va konteynerning bunga ta'siri
yo'q, shuning uchun tartib aynan shunday bo'lishi kerak.

Sabab oddiy: bu server internetdan fayl qabul qilib, nomi so'rovdan kelib
chiqadigan fayllarni diskka yozadi — ya'ni xato aynan shu yerda ixtiyoriy fayl
yozuviga aylanadi. Server jarayoniga root umuman kerak emas (8080 ni tinglaydi,
80 ni emas). Rekursiv `chown` faqat papka hali `app` niki bo'lmasa ishlaydi:
bir martalik migratsiya, har restartda takrorlanadigan ish emas.

⚠️ `docker exec` **ENTRYPOINT'ni chetlab o'tadi** va root beradi — rasm
yozadigan buyruqlar (`seedmenu`) `-u 10001` bilan chaqiriladi.

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

### 5.9 Pog'onali narx (volume tiers)

Oyiga 3 000 tagacha 1 000 so'm · 3 000–10 000 → 700 · 10 000 dan yuqori → 500.
Sozlamada: `PRICE_TIERS=3000:1000,10000:700,0:500` (oxirgi band `0` = cheksiz).

⚠️ **Tekis narx eng yaxshi mijozni jazolaydi.** Kuniga 400 buyurtma —
oyiga 12 mln so'm, ya'ni bu yerda o'rta darajali dasturchining oyligi. Aynan
o'sha nuqtada tarmoqning moliyachisi hisob-fakturani o'qishni to'xtatib,
hisob-kitob qila boshlaydi. Sababi qiymat yomonligida emas (bu ularning
tushumining 0,5–2% i, agregatorlarda 15–20%), balki **katta qatorlar
muzokara qilinishida**.

**Shift emas, pog'ona**: shift qo'yilsa undan keyingi har bir buyurtma bizga
umuman pul keltirmaydi — ikkala tomon uchun ham noto'g'ri rag'bat. Pog'onada
marjinal narx musbat qoladi, o'rtacha esa tushadi, ya'ni o'sish baribir pul
keltiradi va suhbat "qimmatlashib ketdingiz" dan "qancha o'ssak, shuncha
arzon" ga aylanadi.

Ko'ringanidan arzonroq: mijozlarning ko'pchiligi birinchi pog'onadan
chiqmaydi, ya'ni platforma yuqoridagi churn jarligini olib tashlash uchun
tushumning bir necha foizini beradi (realistik aralashmada ~7%).

⚠️ **Davr bo'yicha hisoblanadi, kunlik emas** (`models.PriceForOrders`).
Kunlik qo'llansa pog'ona har yarim tunda qaytadan boshlanadi va kuniga 400
buyurtma qiladigan restoran birinchi banddan hech qachon chiqmaydi — narvon
umuman ishlamaydi. Shu sabab `TenantDay.Billable` tekis kunlik baho bo'lib
qoladi, hisob-faktura va mijozlar ro'yxati esa davr buyurtmalari sonidan
qayta hisoblaydi. Ikkalasi farq qilishi mumkin, va **to'g'risi hisob-faktura**.

Mijozning o'z narvoni (`tenant.priceTiers`) platformanikidan ustun —
`pricePerOrder` bilan bir qoida: kelishilgan shart platforma o'zgarishidan
omon qoladi.

**Konsolda "Ulush" ustuni** — hisob ÷ restoran tushumi. Churn'ni oldindan
aytadigan yagona raqam: 2% dan past bo'lsa mijoz sanamaydi, 3% dan oshsa
sanay boshlaydi. Ikkala yarmi alohida ma'nosiz — 12 mln so'm biznesning ham
1% i, ham 20% i bo'lishi mumkin, va bular butunlay boshqa suhbatlar.

---

### 6.0 Bepul xizmat va chegirma

⚠️ **`pricePerOrder: 0` bepul xizmat emas.** Nol narx tasodifan tozalangan
maydondan farq qilmaydi, sababsiz nol summali hisob chiqaradi, va eng yomoni —
demo muddati tugaganda kechki sweep mijozni baribir o'chirib qo'yadi.

Shuning uchun alohida bayroq: `free` + **majburiy** `freeReason` +
`freeUntil` (bo'sh = **muddatsiz**). Va oraliq variant — `discountPercent`.

Nima uchun kerak: birinchi mijozlar o'z hisob-fakturasidan qimmatroq. O'n ikki
filialli tarmoq birinchi haqiqiy foydalanuvchi bo'lishga rozi bo'lsa, u
mahsulotga obro' sotib olayapti — va buning uchun undan oyiga 200 ming so'm
olish mavjud savdolarning eng yomoni. Demak: bepul, ataylab, va shartlari
keyingi odam o'qiy oladigan joyda yozilgan.

Qoidalar bitta predikatda (`Tenant.FreeAt` / `ChargeFor`), uchta joyda emas:
- **Hisob baribir chiqariladi, 0 summa bilan** — oy bo'lgani va ataylab
  hisoblanmagani yozuvi. Hisoblari orasida bo'shliq bor akkauntni keyin hech
  kim tushuntira olmaydi. Sabab hisob izohiga ko'chiriladi.
- **Summa chiqarilganda muzlatiladi** — keyingi oy chegirma o'zgarsa,
  yuborilgan hisob qayta narxlanmaydi.
- **Sweep tegmaydi.** Bu bayroq aynan shuning uchun bor: va'da bilan kelgan
  tarmoqning qatorida hisob ochilgan kundan qolgan demo sanasi turadi, va
  busiz kechki sweep o'n ikki restoranni o'chirib qo'yardi. Bundan yomonroq
  birinchi taassurot yo'q.
- **"Qo'ng'iroq qilish kerak" ro'yxatida chiqmaydi** — pul haqidagi suhbat
  allaqachon boshqacha tugagan. Lekin **qo'lda to'xtatilgan** bo'lsa ko'rinadi:
  uni odam ataylab bosgan.

---

### 6.1 Pul qanday olinadi: naqd, keyin perechisleniye

⚠️ **MVP MChJ'siz boshlanadi.** Yuridik shaxs yo'q — ya'ni shartnoma ham,
imzolangan hisob-faktura ham, pul qabul qiladigan bank hisobi ham yo'q. Odam
restoranga borib **naqd** oladi. Bu tozalanishi kerak bo'lgan vaqtinchalik
chora emas, **hozirgi yagona yo'l**, va daftar buni shunday yozadi.

MChJ ro'yxatdan o'tgach — **perechisleniye**. `transfer` birinchi kundan
modelda turadi (ishlatilmagan holda): keyin qo'shish har bir saqlangan qatorni
qayta o'qib "bu qanday to'langan edi?" degan savolga javob izlashni talab
qilardi.

**Hisob-faktura daftari** (`invoice`, `internal/handlers/invoices.go`):

- **Summa chiqarilganda muzlatiladi.** Kunlik qatorlar to'planaveradi; mijozga
  aytilgan raqam esa o'zgarmasligi kerak — aks holda kelishilgan summa
  yo'qoladi.
- **To'lov — ismi bor yozuv, hisoblagich emas.** Kim berdi, kim oldi, qachon,
  qaysi davr uchun. Imzosiz naqd uch haftadan keyin bahsga aylanadi — kuryerning
  naqd topshiruvi va ishchi oyligi bilan aynan bir sabab. To'lovlar
  **qo'shiladi**, chunki pul bo'lak-bo'lak keladi.
- **Yopilgan davr uchun** chiqariladi, ishlab turgani uchun emas: tugamagan oy
  uchun hisob restorandan hali qilinmagan buyurtmalar pulini so'raydi.
- Davr tenantning **o'z langaridan** olinadi (`billing.Cycle`), ya'ni hisob
  dashboard bilan hech qachon kelishmovchilikka tushmaydi va bir kun ikki
  hisobga tushmaydi (`[from, to)` yarim ochiq).
- `(tenantId, from, to)` **unique** — ikki marta bosish ikkinchi qarz
  yaratmaydi; ikkinchi so'rov mavjud hisobni qaytaradi.
- **Bekor qilishda sabab majburiy** (buyurtma bekor qilish bilan bir qoida):
  izsiz yo'qolgan hisobni keyin hech kim tushuntira olmaydi.
- **Hisob chiqarish hech kimni o'chirmaydi.** `suspended` operator ataylab
  bosadigan tugma bo'lib qoladi — o'zi to'xtatadigan daftar restoranni
  matn xatosi tufayli oflayn qilardi.

---

## 7. Yangilanish oqimi

```
git push main → CI image quradi → control plane:
  har tenant konteynerini navbat bilan yangi image bilan qayta ko'taradi
  (migratsiyalar boot'da o'zi ishlaydi)
```

Navbat bilan — chunki barchasi birdan qayta ishga tushsa Mongo'ga bir vaqtda
50 ta migratsiya uriladi. Frontend bitta bo'lgani uchun u bir marta yangilanadi.

### 7.1 Rolling update (amalga oshirilgan)

`internal/handlers/rollout.go`, konsolda bosh sahifadagi panel.

⚠️ **"Yangilangan" image ID bo'yicha hal qilinadi, teg bo'yicha emas.** Teg —
siljiydigan yorliq: `keel-tenant:latest` deploy'dan keyin boshqa image, lekin
eskisidan yaratilgan konteynerlar eski kodni ishlatishda davom etadi va o'zini
**butunlay sog'lom** ko'rsatadi. Deploy'ning yashil bo'lib turib yolg'on
bo'lishi shundan (2026-08-05 dagi 6-xato). Shuning uchun konsol jonli konteyner
image ID'larini joriy teg bilan solishtiradi — qo'lda qayta yaratilgan yoki
rollout'dan keyin ochilgan mijoz ham hisobga tushadi.

Qoidalar:
- **Bittadan**, va har biri o'z `/health` iga javob bergandan **keyin**
  keyingisiga o'tiladi.
- **Bir vaqtda bitta rollout** (deploy'dagi `flock` bilan bir mantiq: ikkita
  yugurish bitta konteyner nomida to'qnashadi).
- **Ketma-ket 3 xato — to'xtash.** Bitta mijoz — support tiketi; buzuq image —
  avariya, va davom etish ellikta restoranni birma-bir o'chirardi. To'xtash
  qolganini ishlaydigan image'da qoldiradi.
- **To'xtatilgan mijozlar o'tkazib yuboriladi** — konteynerlari ataylab
  o'chirilgan, va qayta yaratish to'lamagan mijozni jimgina onlayn qilardi.

Ishga tushishi: control konteyneri har deploy'da almashtiriladi, shuning uchun
rollout **boot'dan ~90 soniya keyin** o'zi boshlanadi (`ROLLOUT_ON_BOOT=0` —
o'chirish). Oddiy reboot'da hech nima qilmaydi: image ID'lar o'zgarmagan.
Deploy skriptining ichida emas — ellikta restoranni yangilash daqiqalar oladi,
va o'n ikkinchi restoranning sekinligi tufayli yiqilgan deploy tuzatilgan
muammodan battarroq bo'lardi.

### 6.2 Watermark ("Powered by Keel")

⚠️ **Bir muddat umuman ishlamagan.** Control plane `hideWatermark` ni saqlardi,
konsolda galochka bor edi, statistikada "watermarksiz" soni ham sanalardi —
lekin **sayt tomonida qator umuman chizilmagan edi**. Ya'ni bu "yashirish
buzilgan" emas, xususiyatning yarmi yozilmagan edi. Galochka bosilmagan
mijozlarda ham hech nima ko'rinmagani shundan.

Endi `(site)/layout.tsx` har so'rovda `showWatermark()` ni chaqiradi
(`lib/api.ts`): Host → control `/internal/resolve` → `hideWatermark`, 60
soniya keshlanadi (slug bilan bir kesh).

**Nega ish vaqtida, muhitdan emas**: galochka mijoz to'lagan payt konsolda
bosiladi, va konteynerga yaratilishda berilgan qiymat qayta provisioning
qilinmaguncha eski bo'lib qolardi — `rewrites()` ning build vaqtidagi tuzog'i
bilan aynan bir xil, faqat bir qavat narida.

Mustaqil o'rnatilgan sayt (bitta restoran o'z VPS'ida) Keel tenant'i emas va
qatorni hech qachon ko'rsatmaydi. Noma'lum host uchun ham ko'rsatilmaydi:
kimligini aniqlay olmagan saytga birovning brendini yozgandan ko'ra hech nima
yozmagan yaxshi.

---

### 7.2 Mijoz domenini avtomatik ulash (amalga oshirilgan)

Egasi sozlamalarda DNS'ni tekshiradi va "Ulash" ni bosadi — tamom.

**DNS — egalik isboti.** Domenni bizning serverga yo'naltirishni faqat
registrator hisobidagi odam qila oladi, ya'ni bu aynan aytilayotgan da'voning
o'zi. Vercel va Netlify ham shu dalilni qabul qiladi.

Tenant → control kanali (`POST /internal/domain`): token `HMAC(secret, slug)` —
**hisoblanadi, saqlanmaydi** (kiosk kodlari bilan bir naqsh), ya'ni konteyner
kalitini yaratilishi bilan oladi va migratsiya qilinadigan narsa yo'q.
Solishtirish `ConstantTimeCompare`.

⚠️ **Control DNS'ni o'zi qayta tekshiradi.** Tenant ham tekshiradi, lekin bu
qulaylik uchun (javob tez keladi va topilgan IP'lar ko'rsatiladi): tenant
serveri **mijozning tomonidagi mashina**, va u "men tekshirdim" deb aytish
orqali domen egallay olmasligi kerak.

⚠️ **Platformaning o'z nomlari da'vo qilinmaydi** (`keel.uz` va ostidagilar):
aks holda mijoz bizga CNAME qo'yib `admin.keel.uz` ni "ulab" olardi.

Domen o'zgargach konteyner **qayta yaratiladi** — `PUBLIC_BASE_URL` va
`CORS_ORIGINS` yaratilish paytida o'qiladi, ya'ni busiz sayt yangi manzilda
ochilardi-yu chop etadigan har bir mutlaq havola (to'lov callback'i, buyurtma
kuzatuvi, QR) eskisini yozardi.

**Rollback**: eski image tegi bilan qayta ko'tarish. Migratsiyalar oldinga
qarab yozilgani uchun (qo'shadi, o'chirmaydi) bu xavfsiz — shu qoidani
saqlash kerak.

---

## 7.3 Mijoz kartochkasidagi ko'rsatkichlar

`GET /tenants/{id}/live` (`internal/tenantstats`) — **shu bitta mijozning
bazasidan** o'qiladi: bugungi buyurtmalar va bekor qilinganlar, tushum,
o'rtacha chek, buyurtma turlari, hozir oshxonadagi navbat, mijozlar, ishchilar
(shu jumladan **hozir smenada** turganlar), kuryerlar holati bo'yicha, menyu,
filiallar, bronlar va 30 kunlik top taomlar.

**Bu umumiy ro'yxatning teskarisi va ataylab shunday.** Bosh sahifa "hamma
mijoz qanday?" deb so'raydi, va unga har tenant bazasiga qo'ng'iroq qilib
javob berish sahifani **har sotilgan mijoz bilan sekinlashtiradi** — kechki
yig'uvchi (`aggregate`) aynan shuning uchun bor. Bitta mijozning kartochkasi
esa boshqa savol beradi: "hozir shu restoranda nima bo'lyapti?" — va javobning
katta qismi vaqt qatori emas (nechta kuryer bor, nechtasi smenada), soatlik
oldindan hisoblab qo'yib bo'lmaydi. Narx odamning e'tibori bilan chegaralangan,
mijozlar soni bilan emas.

⚠️ **Alohida endpoint, `GET /tenants/{id}` ga qo'shilmagan**: bu chaqiruv sekin
bo'lishi, osilib qolishi yoki umuman yiqilishi mumkin (to'xtatilgan konteyner,
migratsiyadagi tenant). Qo'shib yuborilsa, ularning har biri **konteyner
holatini ko'rsatadigan sahifani** o'ldirardi — ya'ni aynan biror narsa
noto'g'ri bo'lgani uchun ochilgan sahifani. Ajratilgan holda kartochka darhol
chiziladi, raqamlar keyin keladi yoki kelmaydi (sababi bilan).

To'xtatilgan mijozda nollar **to'g'ri**, tirik mijozda esa nol — muammo
belgisi. Shuning uchun javobda `error` bo'ladi va kartochka buni yozadi:
nollarni xotirjamlik deb o'qish mumkin emas.

`TenantDay` ga `cancelled` qo'shildi — sanaladi, lekin **hech qachon hisobga
qo'shilmaydi**. Sababi: qatorlar faqat hisob-faktura qila oladigan narsani
tutsa, "bu mijozda bekor qilishlar ko'payib ketyapti" degan holat ko'rinmay
qoladi — va aynan o'sha mijoz tez orada qo'ng'iroq qiladi.

Grafiklar kutubxonasiz (bosh sahifadagi naqsh davomi). Ranglar — tekshirilgan
kategorik palitra; **bekor qilingan qizil — status rangi**, kategorik uyacha
emas, ya'ni boshqa hech nima uni ishlatmaydi. Sanoq va so'm **hech qachon
bitta grafikda** emas: ikki o'lchov bitta ramkada ma'nosiz kesishish nuqtasi
yasaydi.

---

## 7.4 keel.uz: integratsiyalar, hamkorlar, status

**Integratsiyalar bo'limi** (`components/Integrations.tsx`) — kassa, to'lov,
SMS, xarita, telefoniya, tashqi yetkazish, har biri o'z ikonkasi bilan.
Tugallanmaganlari yashirilmaydi, "tez orada" deb turadi: halol "tez orada"
suhbatni davom ettiradi, yo'qlik esa tugatadi. Oxirgi kartochka — **"ro'yxatda
yo'qmi?"**, va u izoh emas, bo'limning teng yarmi: ro'yxatda bo'lmagan
restoran uchun aynan shu muhimroq.

⚠️ **Ikonkalar chiziladi, yuklanmaydi.** Kassa sotuvchisining logotipiga bizda
litsenziya yo'q, birovning brend faylini hotlink qilish esa ham huquqiy savol,
ham buzilishini kutayotgan rasm. Har guruhga bitta glif — u **nima turdagi**
narsa ekanini aytadi; kimligini nomlar tashiydi.

**Logotip bitta shakl, uch joyda** — `keel-site/src/components/Logo.tsx` dagi
`KeelMark`: sarlavha, favicon (`app/icon.svg`, `app/apple-icon.png`) va
mijozning footer'idagi nishoncha.

⚠️ **Nishoncha `currentColor` da chiziladi, Keel'ning sariq rangida emas.** U
birovning restorani ostida, ular tanlagan palitra ichida turadi — u yerda
ikkinchi brend rangining paydo bo'lishi aynan egani "buni olib tashla" deyishga
undaydigan narsa, va nishoncha faqat **tinch qoldirish oson** bo'lgani uchun
ishlaydi. Shuning uchun u footer'ning o'chgan siyohini oladi, hover'da esa
restoranning o'z aksentini.

Favicon faylida meros oladigan narsa yo'q, shuning uchun u yerda rang
yozilgan — va bu **aksent, siyoh emas**: tab paneli ba'zi mashinalarda qora,
ba'zilarida oq, va palitradagi ikkalasida ham o'qiladigan yagona qiymat —
sariq. Fin qisqartirilmagan: 16px da qisqasi dog'ga o'xshaydi va logo
anonim egri chiziqqa aylanadi.

Nishoncha `frontend` ga **ko'chirilgan, import qilinmagan**: keel.uz va tenant
ilovasi ikki alohida build, va ikkita `<path>` uchun umumiy paket — abadiy
qaraladigan bog'liqlik. Shakl o'zgarsa, ikkalasida o'zgaradi.

⚠️ **Tushum "olingan pul", "buyurtma tushdi" emas** — yig'uvchida ham,
mijoz kartochkasida ham, restoranning o'z dashboardida ham bir xil qoida:
`paymentStatus: paid` (bank tasdiqladi) yoki `delivered` (kuryer pul bilan
qaytdi). Bekor qilingan hech qachon sanalmaydi, to'langan bo'lsa ham.

Uchta joyda bir xil bo'lishi shart: aks holda konsoldagi "Mijozlar tushumi"
restoranning o'z raqamidan doim yuqori turadi va qaysi biri to'g'ri ekanini
hech kim ayta olmaydi.

**Hisob-fakturaga ta'sir qilmaydi**: u `Orders × narx`, va `orders` hamon
oshxonaga yetgan har bir buyurtmani sanaydi. Ovqatni pishirgan restoran
mehmon uyda bo'lmaganida ham hisob oladi.

**Migratsiya**: yig'uvchi har soatda oxirgi 35 kunni qayta yozadi, ya'ni
maydon ma'nosi o'zgarganda yaqin tarix keyingi tikda o'zini tuzatadi —
qo'lda ishga tushiriladigan narsa yo'q. Faqat oynadan eski qatorlar eski
ma'noda qoladi, va ular yozmagan narsani tiklashning iloji yo'q.

**Hamkorlar karuseli** — Keel mijozlari (`GET /partners`, `showcase.go`):
- **Ruxsat so'raladi**: faqat konsolda belgilangan tenant chiqadi
  (`tenant.showcase`, standart **o'chiq**). Mijozning brendini bizning
  marketing sahifamizga qo'yish — uning qarori; so'ramasdan logosini ko'rgan
  mijoz — shikoyati bor mijoz, va tavsiya sahifasi norozilikdan omon qolmaydi.
- **Logo mijozning o'z saytidan, jonli** — o'z bazasidan o'qiladi va o'z
  domenidan beriladi, ya'ni seshanba kuni rebrend qilgan restoran shu yerda
  ham seshanba kuni yangilanadi. Nusxa olish — sekin eskiradigan devor.
- **Faqat `active`**: to'xtatilgan mijozning sayti javob bermaydi, ya'ni logo
  buzilgan rasm bo'lib "to'lov kutilmoqda" sahifasiga olib borardi — ishlaydigan
  platforma ko'rinishi kerak bo'lgan sahifada.
- Karusel 5 daqiqa keshlanadi (landing — eng ko'p so'raladigan sahifa),
  hover'da to'xtaydi va `prefers-reduced-motion` ni hurmat qiladi.
  Logosi yo'q mijoz **nomi bilan** chiziladi, tashlab yuborilmaydi.

**`keel.uz/status`** (`handlers/status.go`) — o'lchangan ishlash vaqti.

⚠️ Status sahifasining asosiy tuzog'i — **standart holatda yashil bo'lish**:
hech nima teskarisini aytmagani uchun "hammasi joyida" deydi. Bunday sahifa
yo'qidan yomonroq — u noto'g'ri bo'lgan yagona soatda dalil sifatida o'qiladi
va aynan o'zi qozonmoqchi bo'lgan ishonchni sarflaydi.

Shuning uchun **uchta holat, ikkitasi emas**: ishladi / nosozlik / **ma'lumot
yo'q**. Uchinchisi bo'shliq bo'lib chiziladi — na yashil, na qizil. Bilmaslikni
ikkalasidan biriga bo'yagan sahifaga ikkinchi marta ishonishmaydi.

Boshqaruv xizmati javob bermasa — **holatning o'zi shu**, va sahifa buni
aytadi. Namuna har daqiqada olinadi (o'z bazasi + ishlashi kerak bo'lgan
mijoz konteynerlari) va **soatlik guruhga** yig'iladi: daqiqasiga bitta hujjat
yiliga yarim million, soatiga bittasi 8760 ta, va "qaysi soatda o'chdi?" —
odam so'raydigan yagona aniqlik.

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

⚠️ **Bu taxmin oyiga 400 buyurtma.** Faol restoran kuniga 400 qiladi, ya'ni
30 barobar ko'p — 5.9 dagi pog'onali narx aynan shu holat uchun. Hajm
taxminini modeldan chiqarayotganda qaysi biri ekanini tekshiring.

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
