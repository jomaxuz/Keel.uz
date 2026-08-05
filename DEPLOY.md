# DEPLOY.md — Deploy qo'llanmasi

Bitta restoran = bitta deployment. Ikki variant bor:

- **A variant — hammasi bitta VPS'da** (tavsiya etiladi): nginx + docker
  (frontend + backend + mongo). Eng sodda, hamma narsa bitta domenda.
- **B variant — frontend Vercel'da, backend VPS'da**: sayt Vercel'dan,
  API va rasmlar VPS'dagi `api.` subdomendan.

---

## Hozirgi jonli deployment — Keel (ko'p mijozli)

| | |
|---|---|
| Domen | **keel.uz** + har mijozga `<slug>.keel.uz` |
| VPS | `169.58.131.165`, Ubuntu 24.04, 4 CPU / 8 GB |
| Papka | `/opt/keel` (egasi `deploy-keel`) |
| Chekka | **Caddy** 80/443 da, boshqa hech nima port chiqarmaydi |
| Fayl | `docker-compose.saas.yml` + `.env` |

Bu server **faqat Keel uchun**. Undan oldingi `173.249.8.13` da
`filmorauz.net` turibdi va u nginx'ning 80/443 ini egallagan — Caddy'ning
on-demand TLS'i esa aynan o'sha portlarni talab qiladi. Ikkalasini bir
mashinaga sig'dirish mumkin edi, lekin faqat filmorauz'ning TLS'ini,
sertifikat yangilanishini va mijoz IP ko'rinishini o'zgartirish evaziga —
ishlab turgan begona proyekt uchun bu narx juda qimmat.

### Nima ishlaydi

```
Internet :443 ──▶ Caddy ──┬── keel.uz, www        → keel-site:3100
                          ├── <slug>.keel.uz      → /api, /uploads → keel-<slug>:8080
                          │                         qolgani        → keel-frontend:3000
                          └── mijozning o'z domeni → xuddi shunday
```

Tenant konteynerlarini **compose emas, control plane** yaratadi — har mijozga
bittadan, shu `keel` tarmog'ida. Shuning uchun konteyner nomlari qat'iy:
generatsiya qilingan Caddy konfiguratsiyasi ularga nom bo'yicha murojaat
qiladi.

### Sertifikatlar

Hech qanday certbot yo'q. Caddy **on-demand TLS** ishlatadi: domen birinchi
marta so'ralganda sertifikat o'sha zahoti olinadi, lekin faqat
`/internal/tls-ask` "bu bizniki" desa. Busiz istalgan odam domenini shu IP'ga
yo'naltirib Let's Encrypt limitini kuydirardi.

⚠️ `caddy_data` volumi — barcha sertifikatlar. Uni yo'qotish qayta ishga
tushirishni **uzilishga** aylantiradi, chunki limit haftalik.

### DNS

ahost'da (yoki domen qayerda bo'lsa) uchta A yozuv:

```
keel.uz     A   169.58.131.165
www         A   169.58.131.165
*           A   169.58.131.165     ← mijoz subdomenlari shu orqali
```

Wildcard sertifikat **kerak emas** — on-demand TLS har subdomen uchun alohida
oladi, ya'ni DNS provayderining API'si ham, DNS-01 ham kerak emas.

### Deploy

```bash
ssh root@169.58.131.165
su - deploy-keel
cd /opt/keel && git pull
docker compose -f docker-compose.saas.yml --profile tenant build
docker compose -f docker-compose.saas.yml up -d
```

`--profile tenant` — mijoz backendining image'i (`keel-tenant:latest`) ham
qurilsin degani. U ishga tushirilmaydi: uni control plane har mijoz uchun
alohida ko'taradi.

Avtomatik deploy (CI) hali sozlanmagan. Eski `traderbot.uz` workflow'i
o'chirildi — nishoni yo'q edi.

---

## 0. Oldindan kerak bo'ladigan narsalar

| Narsa | Qayerdan | Izoh |
|---|---|---|
| Domen | mijoz nomiga | A/AAAA yozuv VPS IP'siga |
| VPS | Ubuntu 22.04+ , 2 GB RAM | Docker + nginx o'rnatiladi |
| 2GIS API key | https://dev.2gis.com | bepul, xarita uchun |
| SMS provayder | eskiz.uz yoki Play Mobile | mijoz login kodlari uchun |

---

## A variant — bitta VPS (nginx + docker)

### A1. Serverni tayyorlash

```bash
sudo apt update && sudo apt install -y docker.io docker-compose-plugin nginx certbot python3-certbot-nginx git
sudo systemctl enable --now docker nginx

# Vaqt mintaqasi. Ishchilar davomati kunlarga LOKAL sana bo'yicha yoziladi
# (00:40 da chiqqan oshpaz kechagi smenani yopadi), shuning uchun UTC'da
# qolgan server butun kalendarni bir kunga surib qo'yadi.
sudo timedatectl set-timezone Asia/Tashkent
timedatectl                     # "Time zone: Asia/Tashkent (+05)" ko'rinsin
```

### A2. Kodni olib kelish va sozlash

```bash
sudo mkdir -p /srv && cd /srv
git clone <repo-url> restaurant && cd restaurant

cp .env.prod.example .env
nano .env        # DOMAIN, JWT_SECRET, ADMIN_PASSWORD, 2GIS key, SMS
```

`JWT_SECRET` uchun: `openssl rand -hex 32`.

### A3. Stack'ni ko'tarish

```bash
docker compose -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.prod.yml ps
curl -s localhost:8080/health          # => ok
curl -s -o /dev/null -w '%{http_code}\n' localhost:3000   # => 200
```

Birinchi ishga tushishda backend avtomatik ravishda:
- admin foydalanuvchini yaratadi (`.env` dagi login/parol, birinchi kirishda
  parolni majburiy o'zgartiradi),
- restoran profilini yaratadi,
- **namuna menyuni** (7 kategoriya, 48 taom, rasmlari bilan) yozadi —
  faqat baza bo'sh bo'lsa.

### A4. nginx + HTTPS

```bash
sudo cp nginx/restaurant.conf /etc/nginx/sites-available/restaurant.conf
sudo sed -i 's/example.uz/SIZNING-DOMEN.uz/g' /etc/nginx/sites-available/restaurant.conf
sudo ln -s /etc/nginx/sites-available/restaurant.conf /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo mkdir -p /var/www/certbot

# Sertifikat (birinchi marta 443 bloki hali ishlamaydi — certbot o'zi qo'shadi)
sudo certbot --nginx -d SIZNING-DOMEN.uz -d www.SIZNING-DOMEN.uz

sudo nginx -t && sudo systemctl reload nginx
```

Certbot avtomatik yangilanadi (`systemctl status certbot.timer`).

### A5. Tekshirish

```bash
curl -I https://SIZNING-DOMEN.uz            # 200
curl -s https://SIZNING-DOMEN.uz/api/v1/restaurant | head -c 200
curl -I https://SIZNING-DOMEN.uz/uploads/seed/osh.jpg   # 200
```

Brauzerda: sayt → menyu → savat → checkout (xarita) → admin panel
`/admin/login`.

**Vaqt mintaqasini tekshiring** — ishchilar davomati kunlarga *lokal* sana
bo'yicha yoziladi (soat 00:40 da chiqqan oshpaz kechagi smenani yopadi), ya'ni
UTC'da ishlayotgan konteyner butun kalendarni bir kunga surib qo'yadi:

```bash
docker compose -f docker-compose.prod.yml exec backend date
# → Asia/Tashkent (+05) ko'rinishi kerak. Bo'lmasa .env da TZ=Asia/Tashkent
```

---

## B variant — frontend Vercel'da, backend VPS'da

Sayt: `https://example.uz` (Vercel) · API: `https://api.example.uz` (VPS).

### B1. VPS'da faqat backend + mongo

`docker-compose.prod.yml` dagi `frontend` servisini ishlatmaslik mumkin:

```bash
docker compose -f docker-compose.prod.yml up -d --build mongo backend
```

nginx uchun `api.example.uz` server bloki (`nginx/restaurant.conf` ni nusxalab,
faqat `/api/`, `/uploads/`, `/health` location'larini qoldiring va
`server_name api.example.uz` qiling), keyin:

```bash
sudo certbot --nginx -d api.example.uz
```

`.env` da CORS'ni oching (brauzer to'g'ridan-to'g'ri API'ga so'rov yuborishi
mumkin):

```
CORS_ORIGINS=https://example.uz,https://www.example.uz
```

va `docker compose -f docker-compose.prod.yml up -d backend`.

### B2. Vercel loyihasi

- Repo'ni GitHub'ga push qiling → Vercel → **New Project** → **Root Directory:
  `frontend`** (muhim, repo ildizi emas).
- Framework: Next.js (avtomatik aniqlanadi).
- **Environment Variables** (Production + Preview):

| Kalit | Qiymat |
|---|---|
| `NEXT_PUBLIC_API_URL` | `/api/v1` |
| `NEXT_PUBLIC_UPLOADS_URL` | `/uploads` |
| `BACKEND_ORIGIN` | `https://api.example.uz` |
| `INTERNAL_API_URL` | `https://api.example.uz/api/v1` |
| `NEXT_PUBLIC_MAP_API_KEY` | 2GIS key |

`BACKEND_ORIGIN` — `next.config.ts` dagi rewrites shu manzilga `/api/*` va
`/uploads/*` ni proksilaydi. Shu sababli brauzer faqat Vercel domeni bilan
gaplashadi: CORS ham, mixed-content ham muammo qilmaydi.

- Domenni Vercel'ga ulash: **Settings → Domains → Add** → ko'rsatilgan
  A/CNAME yozuvlarini domen registratorida qo'shing.

### B3. Eslatmalar (B variant)

- Rasmlar baribir VPS diskida (`uploads_data` volume) — Vercel'da fayl
  saqlanmaydi.
- Vercel bepul rejasida serverless funksiyalar ishlaydi; sahifalar cookie
  (til) sababli dinamik render bo'ladi — bu normal.
- VPS o'chsa, sayt ochiladi-yu, menyu bo'sh ko'rinadi (API yo'q).

---

## Admin parolini unutib qo'ysangiz

### 1-yo'l: panelning o'zidan (SMS bilan) — tavsiya etiladi

Kirish sahifasida **"Parolni unutdingizmi?"** tugmasi bor: login yoziladi,
hisobga biriktirilgan raqamga bir martalik kod keladi, so'ng yangi parol
qo'yiladi. Mijoz sizga qo'ng'iroq qilmaydi.

**Shart**: hisobga tiklash raqami biriktirilgan bo'lishi kerak. Seed qilingan
birinchi `owner` da raqam **yo'q**, shuning uchun mijozga topshirishdan oldin:

> Panel → **Hisobim** → "Parolni tiklash raqami" → raqam qo'shiladi va SMS
> kod bilan tasdiqlanadi.

Bu ish **SMS provayderi haqiqiy bo'lganda** bajarilishi kerak
(`SMS_PROVIDER=eskiz` yoki `playmobile`). `demo` rejimda kod SMS bilan
ketmaydi — API javobida qaytadi, ya'ni prod uchun yaramaydi.

Xavfsizlik eslatmasi: tiklash kodi **faqat shu maqsad uchun** beriladi. Restoran
egasi ko'pincha saytning ham mijozi (bir xil raqam) — mijozning login kodi
admin parolini ocha olmaydi va aksincha.

### 2-yo'l: serverda (zaxira)

Raqam biriktirilmagan bo'lsa yoki telefon yo'qolgan bo'lsa, serverda ishlaydigan
buyruq qoladi (serverga SSH kirish huquqining o'zi — autentifikatsiya):

```bash
# VPS'da, loyiha papkasida
docker compose -f docker-compose.prod.yml exec backend /app/adminreset -list
docker compose -f docker-compose.prod.yml exec backend \
  /app/adminreset -username yujo -password 'yangi-kuchli-parol'

# Docker'siz (lokal / dev):
cd backend && go run ./cmd/adminreset -list
go run ./cmd/adminreset -username yujo            # parolni so'raydi (ko'rinmaydi)
go run ./cmd/adminreset -username yangiadmin -create -password '...'
```

Foydali flaglar:

| Flag | Nima qiladi |
|---|---|
| `-list` | mavjud adminlar ro'yxati |
| `-username` | qaysi admin (majburiy) |
| `-password` | yangi parol (yozilmasa — terminal so'raydi) |
| `-create` | admin yo'q bo'lsa yangisini yaratadi |
| `-force-change` | keyingi kirishda parolni majburan almashtiradi |
| `-db` | boshqa baza nomi (default: `MONGO_DB`) |

**Mijozga topshirishdan oldin**: tiklash raqamini biriktiring (1-yo'l), bu
buyruqni bir marta sinab ko'ring, va bir nechta admin bo'lsa ikkinchi `owner`
hisobini zaxira sifatida yarating.

---

## Ishchilar (davomat) — topshirishdan oldin

1. **Filial manzili xaritada belgilangan bo'lsin** (`/admin/settings` → filial):
   kirish/chiqish tugmasi shu nuqtaga nisbatan ishlaydi.
2. **Kirish/chiqish masofasi** — o'sha yerda, standart **50 m**. Telefon GPS'i
   bino ichida 10–30 m xato beradi, shuning uchun 20 m dan kichik qiymat halol
   ishchini ham kirita olmay qoldiradi. Tekshiruvni butunlay o'chirish uchun 0.
3. `/admin/staff` da har bir ishchiga **ish grafigi** yozing (nechidan
   nechigacha) — kam/ko'p ishlaganini sistema aynan shunga qarab aniqlaydi.
4. Ishchi telefonida **`https://SIZNING-DOMEN.uz/staff`** ochiladi, login va
   parol bilan kiriladi, brauzer menyusidan "Ilovani o'rnatish" bosiladi.
   Birinchi kirishda **joylashuvga ruxsat** so'raladi — u majburiy.

---

## Kundalik operatsiyalar

```bash
cd /srv/restaurant

# Yangi versiya
git pull && docker compose -f docker-compose.prod.yml up -d --build

# Namuna menyuni to'la bazaga yozish (eski menyu o'chadi!)
docker compose -f docker-compose.prod.yml exec backend /app/seedmenu -replace

# To'lov tizimini tekshirish (bank o'rniga o'zi qo'ng'iroq qiladi)
docker compose -f docker-compose.prod.yml exec backend \
  /app/paytest -order AB12-3456 -suite

# Loglar
docker compose -f docker-compose.prod.yml logs -f backend

# Baza zaxirasi (kunlik cron uchun)
docker compose -f docker-compose.prod.yml exec -T mongo \
  mongodump --archive --db=restaurant | gzip > /srv/backups/db-$(date +%F).gz

# Rasmlar zaxirasi
docker run --rm -v restaurant_uploads_data:/data -v /srv/backups:/b alpine \
  tar czf /b/uploads-$(date +%F).tgz -C /data .
```

Tiklash: `gunzip -c db-YYYY-MM-DD.gz | docker compose -f docker-compose.prod.yml exec -T mongo mongorestore --archive`.

---

## To'lov tizimlarini ulash va tekshirish

Kalitlar **panelda** kiritiladi: `/admin/settings` → "To'lov tizimlari"
(faqat owner). Kod ichida hech qanday kalit yo'q, `.env` ham talab qilmaydi.

### 1. Provayder kabinetiga manzillarni yozish
Har bir tizim **qayerga qo'ng'iroq qilishini** bilishi kerak. Panelning shu
bo'limida manzillar tayyor holda, nusxalash tugmasi bilan turadi:

| Tizim | Kabinetga yoziladigan manzil |
|---|---|
| Payme | `https://sizning-domen.uz/api/v1/payments/payme` |
| Click | Prepare: `.../api/v1/payments/click/prepare`<br>Complete: `.../api/v1/payments/click/complete` |
| Uzum | `.../api/v1/payments/uzum/check`, `/create`, `/confirm`, `/reverse`, `/status` |

**HTTPS va tashqaridan ochiq domen shart** — bank `localhost` yoki
`192.168.x.x` ga chiqa olmaydi. "Sayt manzili" maydonini to'ldiring: mijoz
to'lovdan keyin shu manzilga qaytadi va yuqoridagi havolalar ham shundan
quriladi.

Kabinetda "buyurtma maydoni" (`account` / `params`) nomi so'raladi — panelda
ham xuddi shu nom yozilishi kerak (standart `order_id`). Unga buyurtma raqami
boradi.

### 2. Bank ulanmasidan oldin tekshirish
`cmd/paytest` — provayderning **o'zini o'ynaydi**: kalitlarni bazadan o'qiydi,
bank yuboradigan chaqiruvlarni aynan o'sha imzo bilan yuboradi va javobni
ko'rsatadi. Merchant kabineti ham, tunnel ham kerak emas.

```bash
cd backend
go run ./cmd/paytest -order AB12-3456          # to'lash oqimi
go run ./cmd/paytest -order AB12-3456 -suite   # rad javoblari ham tekshiriladi
go run ./cmd/paytest -order AB12-3456 -step create   # bosqichma-bosqich
```

`-suite` quyidagilarni tekshiradi: noto'g'ri kalit/imzo, noto'g'ri summa,
mavjud bo'lmagan buyurtma, takroriy chaqiruvlar (provayderlar qayta uradi) va
to'langan buyurtmani ikkinchi marta to'lashga urinish. Oxirida buyurtma
holatini bazadan o'qib ko'rsatadi.

### 3. Provayderning o'z sandbox'i bilan
Payme'da sozlamalarda **"Test rejimi"** ni yoqing va test kalitini kiriting —
checkout `test.paycom.uz` ga ketadi va haqiqiy pul o'tmaydi. Click va Uzum
sandbox'i kabinet orqali beriladi.

Lokal mashinada sinash uchun tunnel kerak (bank ichki tarmoqqa kira olmaydi):

```bash
cloudflared tunnel --url http://localhost:3000
# chiqqan https://... manzilini panelda "Sayt manzili" ga yozing
```

### 4. Ishga tushirishdan oldin
- Payme "Test rejimi" **o'chirilgan** bo'lsin.
- Kalitlar kiritilgan bo'lsin: kiritilmagan tizim checkout'da umuman
  ko'rinmaydi (bu ataylab shunday — bank xato sahifasiga olib boradigan tugma
  buyurtmani yo'qotadi).
- Bitta haqiqiy kichik buyurtma bilan uchdan-uchgacha o'tib ko'ring va
  `/admin/orders` da "To'landi" belgisi chiqqanini tekshiring.

---

## Xavfsizlik ro'yxati

- [ ] `.env` git'ga tushmagan (`.gitignore` da).
- [ ] `JWT_SECRET` — tasodifiy 32+ bayt.
- [ ] Admin paroli birinchi kirishdan keyin o'zgartirilgan.
- [ ] Mongo va backend portlari faqat `127.0.0.1` da (compose shunday qilingan).
- [ ] `ufw`: faqat 22, 80, 443 ochiq.
- [ ] HTTPS sertifikati avtomatik yangilanadi (`certbot.timer`).
- [ ] Zaxira nusxa cron'ga qo'yilgan.
