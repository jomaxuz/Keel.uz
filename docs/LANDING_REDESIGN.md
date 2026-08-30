# Landing (keel.uz) dizaynini yangilash — reja

**Holat:** §5.3–§5.6 bajarildi (2026-08-30). §5.1 ham bajarildi: kadrlar
olinadigan muhit tayyor (quyida §4a). Qolgani — kadrlarni olish va joylash
(§5.2), telefonda/Lighthouse tekshiruvi (§5.7).
**Sana:** 2026-08-27 (ko'rib chiqildi va yozildi), 2026-08-30 (kod qatlami)
**Branch:** `keel-site/landing-redesign` (bo'lim — `keel-site/`, ya'ni
konsoldan tashqari `keel-site/src/app`)

---

## 1. Nima uchun

Hozirgi `keel.uz` **hujjatga o'xshaydi, sotuvchi sahifaga emas**: sakkizta
bo'lim ketma-ket, har biri chapga tekislangan eyebrow + sarlavha + lead +
kartochkalar to'ri. Ko'zni ushlaydigan joy yo'q.

⚠️ **Sahifada bironta ham haqiqiy rasm yo'q.** `keel-site/public` papkasi
umuman mavjud emas; butun vizual qism qo'lda chizilgan izometrik SVG
(`Visual3D.tsx`, 824 qator). Monoblok chizmasiga qarab "bu nima ekan" degan
savol tug'iladi — mijoz esa bu savolni bermaydi, u ketadi.

Namuna sifatida `millypos.uz` ko'rildi (2026-08-27 da to'liq scroll qilindi).
⚠️ **Clone qilinmaydi** — faqat stil olinadi, quyida aynan qaysi qismi.

## 2. Millypos'dan olinadigan narsalar

1. **Har bir kartochka ichida mahsulotning mini-maketi.** Eng katta farq shu:
   u yerda hech bir da'vo faqat matn bo'lib qolmaydi — "ko'p filial" yozilgan
   joyda uchta filial kartasi chizilgan, "tahlil" da haqiqiy ustunli grafik,
   "mobil ilova" da iPhone ramkasi ichida ekran.
2. **Suzuvchi mikro-nishonlar** — maket burchagidan chiqib turgan rangli
   yumaloq kvadrat + oq ikonka + yumshoq soya. Arzon, lekin sahifani jonlantiradi.
3. **Ikonkalar 40–44px badge'da**: brend rangining ~10% foni, ichida to'q
   rangdagi glyph. Bizdagi 20px ingichka chiziq sarlavha yonida yo'qolib ketadi.
4. **Bo'lim sarlavhalari markazda va ikki rangli** (birinchi yarmi brend
   rangida, ikkinchisi asosiy matn rangida) — bo'limlar orasida ritm beradi.
5. **Scroll-reveal** — har blok pastdan chiqib paydo bo'ladi.
6. **Uch interaktiv blok**: narx kalkulyatori, 3 qadamli ishga tushish
   timeline'i, FAQ akkordeon.

## 3. Olinmaydigan narsalar

- ⚠️ **Ko'k rang olinmaydi.** To'q sariq (`signal-500`) + krem fon qoladi:
  bu millypos'ning ko'k-oq'idan ko'ra esda qoladi va allaqachon bizning
  belgimiz. Yashil **ikkinchi rang** sifatida faqat "tejash / ulangan /
  ishlayapti" ma'nosida kiritiladi, bezak sifatida emas.
- Ularning tuzilishi (bo'limlar tartibi, matnlari, kalkulyator maydonlari)
  ko'chirilmaydi.

## 4. Rasmlar: qayerdan olinadi

**Manba — `b5somsa.keel.uz` test tenanti.** Bu bizning sinov restoranimiz,
jonli mijoz emas, ya'ni screenshot olish xavfsiz.

⚠️ **Jonli mijoz panelidan screenshot olinmaydi** — u yerda haqiqiy
buyurtmalar, telefon raqamlari va tushum raqamlari bor, va marketing
sahifasida ular bir marta e'lon qilinsa qaytarib bo'lmaydi.

Kerak bo'ladigan kadrlar (✅ — olingan):
- ✅ **Kassa ekrani** (chek ochiq, taomlar to'ri) — `public/shots/till.webp`,
  1600×1000, 83 KB. Kassa bo'limida, `screen` ramkasida.
  ⚠️ Kategoriya sifatida **Ichimliklar** tanlangan: "Milliy taomlar" da 7 taom
  bor va to'rning pastki yarmi bo'sh qoladi — bu "menyu hali kiritilmagan"
  degan ma'noni beradi, ya'ni suratning maqsadiga teskari.
  ⚠️ Ekran o'lchami 1440×900 (2x), 1366×768 emas: kichikroq balandlikda
  kategoriya tugmalari ikki qatorga tushadi va chek ro'yxati yarim qatordan
  kesiladi.
- ✅ **Zal xaritasi** (planshet) — `public/shots/floor.webp`, 1400×973, 44 KB.
  "Uchta ekran" bloki kartochkalar to'ridan **qatorlarga** o'zgartirildi:
  yonma-yon turganda xarita 490px bo'lib qolardi va stol summalari o'qilmasdi
  — ya'ni "screenshot bor" deydi, hech nima ko'rsatmaydi. Qatorda 640px.
  ⚠️ Zal 12 stoldan **18 stolga** kengaytirildi (6×3) va devor + "Bar" /
  "Kirish" maydonlari qo'shildi (`-replan`): kvadratga yaqin zal landshaft
  planshetning to'rtdan birini bo'sh qoldiradi, va ko'z xonaga emas, o'sha
  bo'shliqqa tushadi.
  ⚠️ Chek yoshlari qadami 7 → 5 daqiqaga tushirildi: stol soni oshgach
  sakkizinchi chek 57 daqiqa bo'lib **qizarardi** — hech nima yomon
  ketmayotganini ko'rsatishi kerak bo'lgan ekranda ikkita qizil.
- ✅ **Oshxona ekrani (KDS)** — `public/shots/kds.webp`, 1500×938, 54 KB.
  ⚠️ Passdagi cheklar 6 dan **9 taga** oshirildi: KDS cheklarni uchtadan
  yotqizadi, oltitasi ikki qator bo'lib monitorning pastki uchdan birini bo'sh
  qoldirardi — jim oshxona surati, ya'ni xizmat suratining teskarisi.
- Admin panel: buyurtmalar oqimi, dashboard statistikasi
- Ombor / tannarx ekrani
- Telegram bot va mini app (telefon ramkasida)
- Kuryer PWA (telefon ramkasida)
- Restoran sayti (brauzer ramkasida)

⚠️ **Kadr olishdan oldin `nextjs-portal` yashiriladi** (Next dev indikatori,
chap pastki burchakdagi "N" doirasi). U mahsulotning qismi emas, lekin ikkita
chiqarilgan webp ichiga sezilmay tushib ketdi va u yerda mijozning o'z
kassasida turgan begona nishonga o'xshaydi. `scratchpad/capture.py` uchala
kadrni shu qoida bilan oladi.

⚠️ **`npm run build` va `next dev` bitta papkada bir vaqtda ishlamaydi**:
build `.next` ni qayta yozadi va dev-server undan keyin CSS'siz sahifa beradi.
Bu **layout xatosiga o'xshaydi** — bir marta yarim soat "grid nega
ishlamayapti?" deb qidirildi, aslida sabab server edi. Kadr olishdan oldin
dev-serverni qayta ishga tushiring.

Har biri **light va dark** temada kerak bo'lishi mumkin — sahifa ikkala
temada ishlaydi, va bitta oq screenshot to'q sahifada teshik bo'lib ko'rinadi.
Muqobili: screenshot'ni neytral qurilma ramkasi ichiga solish (ramka temaga
moslashadi, ichidagi rasm o'zgarmaydi) — avval shuni sinash arzonroq.

⚠️ **Monoblok / printer / pul yashigining studiya fotosi bizda yo'q** va uni
men yasay olmayman. Ikki yo'l: (a) apparatni suratga olish kerak, (b) apparat
bo'limi izometrik chizmada qoladi. Qaror qabul qilinmagan.

## 4a. Kadrlar olinadigan muhit (2026-08-30)

⚠️ **Jonli tenantda emas, uning nusxasida.** b5somsa'da namuna menyu
allaqachon bor edi (7 kategoriya, 48 taom), lekin ekranlarni jonli qiladigan
narsa yo'q edi: buyurtma, ochiq chek, kassa smenasi, ombor qoldig'i. Ularni
jonli tenantda yaratish jonli saytga soxta buyurtma yozish, Telegram botga
xabar yuborish va kelajakdagi hisobotlarni buzish demakdir — screenshot uchun
esa lokal nusxa xuddi shunday ko'rinadi.

Muhit:
- Baza: lokal mongo, `demo` (b5somsa profili — nom, logo, manzil, ish vaqti,
  yetkazish zonalari — ommaviy API'dan ko'chirilgan). Logo va cover
  `backend/uploads/b5-*.jpg` ga yuklab olingan, prod'ga hotlink yo'q.
- Ma'lumot: **`cmd/demodata`** (qarang `docs/DECISIONS.md` → "Namuna menyu").
- Xarita: 2GIS (lokal dev kaliti). Tenantning Yandex kaliti domenga bog'langan,
  ya'ni localhost'da bo'sh xarita chiziladi.
- Brend rangi **ko'k qoldirilgan** (`#2563eb`) — mijoz o'z rangini tanlaydi, va
  screenshot aynan shuni ko'rsatadi.

Ishga tushirish:
```bash
docker compose up -d mongo
cd backend && MONGO_DB=demo PORT=8081 go run ./cmd/server     # bir marta: menyu + brend + filial
MONGO_DB=demo go run ./cmd/demodata -db demo -wipe            # ma'lumot
cd ../frontend && BACKEND_ORIGIN=http://localhost:8081 \
  INTERNAL_API_URL=http://localhost:8081/api/v1 npx next dev -p 3001
```

Hisoblar (faqat lokal demo): panel `admin` / `admin123`; xodimlar
`xodim1..xodim6` / `demo12345`, PIN `1000 + (n-1)*11` (kassir `xodim2`/`1011`,
ofitsiant `xodim3`/`1022`, oshpaz `xodim5`/`1044`); kuryerlar
`kuryer1..3` / `demo12345`.

⚠️ **`tonight` qismi eskiradi**: zal, kassa va KDS "hozir"ga qarab filtrlaydi,
ya'ni bir soat oldin yozilgan ma'lumot 70 daqiqalik qizil cheklarga aylanadi.
Kadr olishdan oldin `demodata` qayta ishga tushiriladi.

## 5. Ish bosqichlari

1. `b5somsa.keel.uz` da namuna menyu va bir nechta buyurtma tayyorlash
   (bo'sh ekran screenshot uchun yaroqsiz — bo'sh jadval mahsulotni
   ishlamayotgandek ko'rsatadi).
2. Screenshotlarni olish (yuqoridagi ro'yxat), `keel-site/public/shots/` ga
   joylash. ⚠️ Hajmga e'tibor: bu sahifa O'zbekiston mobil internetida
   ochiladi — `next/image` + webp, va kartochka ichidagi maket uchun to'liq
   ekran emas, **kesilgan bo'lak** yetarli.
3. Dizayn tizimi qatlami: badge'li ikonka, maketli kartochka, suzuvchi
   nishon, qurilma ramkasi — komponent sifatida (`Icons.tsx` dagi path'lar
   qoladi, faqat o'rov o'zgaradi).
4. Bo'limlarni qayta yig'ish: markazlashgan ikki rangli sarlavhalar, ritm
   almashinuvi.
5. Yangi bloklar: narx kalkulyatori (tarif + filial soni + oylik buyurtma →
   jami oylik), 3 qadamli timeline, FAQ akkordeon.
6. Scroll-reveal (⚠️ `prefers-reduced-motion` bilan — `partners-track` da
   allaqachon shu qoida bor, yangi animatsiya undan chetga chiqmasin).
7. Telefonda tekshirish (dark + light), Lighthouse.

## 6. Ochiq savollar — hal qilindi (2026-08-30)

- **Kalkulyator ikkalasini ham hisoblaydi**: kassa tarifi + filial soni + oylik
  buyurtma → bitta oylik summa. Sahifa ikki narsani ikki xil shaklda sotadi
  (filialga oylik, buyurtmaga dona), va ular bir-biridan uzoqda joylashgan —
  faqat bittasini o'qigan mehmon noto'g'ri arifmetika qilyapti, va aynan o'sha
  raqamni raqobatchi bilan taqqoslaydi.
  ⚠️ **Har bir narx lug'atdan o'qiladi** (`till.plans[i].price`,
  `pricing.tiers[i].price`), qayta yozilmaydi: yuqoridagi jadval bilan jimgina
  ixtilof qiladigan kalkulyator — sahifaning mijoz oldida o'zi bilan
  bahslashishi.
  ⚠️ Enterprise ataylab yo'q: uning narxi kelishiladi, ya'ni halol javob — gap,
  raqam emas; oxirgi variantda javobni bo'shatib qo'yadigan select esa buzuq
  kalkulyator.
- **`Visual3D.tsx` qoladi** — apparat va abstrakt joylarda. Endi u qurilma
  ramkasi ichida (`components/landing/Frame.tsx`): ramka temaga moslashadi,
  ichidagi rasm esa screenshot kelganda almashadi.
- **Apparat fotosi** — hali ochiq (§4 oxiri).
