# Reklama (AI targetolog) — qaror va reja

Bu hujjat kod yozishdan **oldin** qabul qilingan qarorlarni, Meta tomonidagi
hisob ishlarini va hali ochiq savollarni yozib qo'yadi — `pos-reja.md` bilan
bir sabab: noto'g'ri faraz ustiga yozilgan kod keyin tashlanadi.

Sana: **2026-09-16**. Holat: **reja, kod yo'q.**

API kontraktlari — `docs/vendor/meta-marketing.md` (o'qilgan nusxa, manba
havolalari va sanasi bilan). Bu yerda **mahsulot va jarayon**.

---

## 0. Nima sotamiz va nega bu boshqacha

**Qo'shimcha xizmat**: oyiga **1 250 000 so'm** (≈100$), kuniga **30 ta AI
so'rovi**. Konsoldan yoqiladi, AI yordamchisi bilan bir mexanizm.

⚠️ **Bizning ustunligimiz «AI auditoriyani tanlaydi» emas.** 2026 da
auditoriyani Meta'ning o'zi tanlaydi — bu API bayrog'i
(`targeting_automation.advantage_audience: 1`), va 500–1000$ oladigan
targetolog ham aslida shuni bosadi.

**Ustunlik — bizda restoranning sotuvi borligi.** Qaysi taom qancha sotildi,
marjasi qancha, qaysi soat bo'sh, qaysi mijoz yo'qolgan, tannarx qancha. Shunga
ko'ra:

- **nimani** reklama qilish kerakligini biz bilamiz (marjasi yuqori va sotuvi
  tushayotgan taom), targetolog esa bilmaydi;
- **natijani** biz haqiqiy buyurtma va foyda bilan o'lchay olamiz, «300 ta klik»
  bilan emas.

⚠️ Shuning uchun bo'limning mahsuloti — **«reklama qancha pul olib keldi»**,
reklama yaratish esa uning bir qismi.

---

## 1. Meta tomonidagi hisob ishlari («KEEL» MChJ ni verifikatsiya qilish)

Bu bo'lim birinchi, chunki eng uzun kutadi va kodga bog'liq emas. Uchta alohida
narsa bor va ular tez-tez chalkashtiriladi:

| # | Nima | Nimaga kerak |
|---|---|---|
| 1 | **Business verification** (biznes-portfel tasdiqlash) | Barcha keyingi narsalarning sharti |
| 2 | **App Review** — `ads_management`, `ads_read` | Mijoz akkauntiga tegish |
| 3 | **Marketing API Access Tier**: Limited → Full | Kvota (2-§) |

### 1.1 Meta nimani solishtiradi

Portfeldagi **yuridik nom**, **manzil**, **telefon** va **sayt** — yuklangan
hujjat bilan mos kelishi kerak. ⚠️ 2026 da eng ko'p uchraydigan rad etish
sababi — **nomning mos kelmasligi**.

Bizning rasmiy ma'lumotlarimiz (saytdan olindi, 2026-09-16):

```
Tashkilot: "KEEL" MChJ
STIR (INN): 313241548
Manzil: Toshkent shahri, Yunusobod tumani, Yunus ota MFY,
        14-mavze, 5-uy, 40-xonadon
Pochta: hello@keel.uz (footer), info@keel.uz (maxfiylik siyosati)
```

⚠️ Portfel nomi **aynan shunday** yozilishi kerak — qo'shtirnoq va imlogacha.
«Keel», «Keel LLC», «KEEL MCHJ» — uchalasi ham nomuvofiqlik hisoblanadi.

### 1.2 Avval saytda tuzatiladigan uchta narsa

Meta tekshiruvchisi saytga qaraydi, va hozir sayt tekshiruvdan **o'ta
olmaydi**:

1. ⚠️ **Footerda huquqiy blok yo'q.** Hozir faqat «© 2026 Keel». Yuridik nom,
   STIR va manzil faqat `/public-offer` va `/privacy-policy` ichida.
   Qo'shiladigan joy: `keel-site/src/components/landing/Shell.tsx` (footer,
   127-qator) va matni `keel-site/src/lib/i18n/dict.ts` (`t.footer.rights`
   yonida).
2. ⚠️ **Saytda telefon raqami umuman yo'q** — na footerda, na ofertada. Meta
   esa biznes telefonini so'raydi va uni hujjat bilan solishtiradi. Bitta
   `+998…` raqam qo'shilishi kerak (ofertadagi va portfeldagi bilan bir xil).
3. **Korporativ pochta** — `hello@keel.uz` bor va bu yaxshi: korporativ domen
   bilan ko'rik bepul pochtaga nisbatan taxminan ikki barobar tez ketadi.

Domen tomondan hammasi joyida: keel.uz HTTPS'da, 7 kundan ancha eski, sarlavhada
nom bor.

### 1.3 Hujjatlar

Meta beshta turni qabul qiladi (hammasi **12 oydan eski bo'lmasligi** va
ustida **nom + manzil** turishi kerak):

1. Ta'sis guvohnomasi / ro'yxatdan o'tkazilganlik hujjati,
2. Biznes ro'yxati yoki litsenziya hujjati,
3. Davlat bergan **soliq** hujjati,
4. **Bank ko'chirmasi** (ko'pincha eng oson: nom ham, manzil ham bitta sahifada),
5. Kommunal to'lov qog'ozi (tashkilot nomiga, oxirgi 12 oy ichida).

O'zbekistondagi MChJ uchun eng ishonchli juftlik: **davlat ro'yxatidan
o'tkazilganlik guvohnomasi** + **bank ma'lumotnomasi/ko'chirmasi**.
⚠️ Bu tavsiya bizniki; Meta mamlakat bo'yicha aniq ro'yxatni portfel ichida
ko'rsatadi — yuborishdan oldin o'sha ro'yxatga qaralsin.

### 1.4 Qadamlar (tartibi muhim)

1. `business.facebook.com` — portfel yaratish. **Nomi = `"KEEL" MChJ`**.
2. **Business settings → Business info**: yuridik nom, manzil, telefon, sayt
   (`https://keel.uz`) — hammasi hujjatdagidek.
3. Telefonni tasdiqlash (SMS yoki qo'ng'iroq).
4. **Security Center → Business verification → Start verification**.
5. Ro'yxatdan o'z tashkilotini tanlash (Meta bazasida topilsa) yoki qo'lda
   kiritish.
6. Hujjat turini tanlab yuklash → **Submit**.

**Muddat:** bir necha daqiqadan 14 ish kunigacha. Hujjat ko'rigi odatda **2–5
ish kuni**; telefon/domen mos kelsa ancha tez. Apellyatsiya — 3–5 ish kuni.

### 1.5 Domenni tasdiqlash (alohida narsa)

Business settings → **Brand safety → Domains** → `keel.uz` → uch usuldan biri:
DNS TXT yozuvi, `<meta>` tegi yoki HTML fayl.

⚠️ Bu verifikatsiyaning bir qismi emas, lekin **piksel va Conversions API**
uchun keyin baribir kerak bo'ladi — va domen tasdiqlanmagani ko'rikda ham
salbiy belgi.

### 1.6 Rad etilsa

- Birinchi navbatda **nomni** solishtiring (portfel ↔ hujjat ↔ sayt).
- ⚠️ **O'sha kuni qayta yubormang** — bir kunda ikkinchi yuborish avtomatik
  belgi oladi. Belgilangan narsani tuzating, **24 soat kutib**, keyin yuboring.
- Boshqa tez-tez uchraydiganlar: sayt «qurilmoqda» holatida, saytda tashkilot
  nomi yoki aloqa ma'lumoti yo'q (bizdagi holat — 1.2 ga qarang), telefon
  hujjatdagi bilan boshqacha.

### 1.7 Verifikatsiyadan keyin

1. **Ilova** yaratish (`developers.facebook.com`), egasi — o'sha portfel.
2. **Facebook Login for Business konfiguratsiyasi** → `config_id` olish
   (token turi: **Business Integration System User**, ⚠️ u muddatsiz).
3. **App Review**: `ads_management`, `ads_read` (lead-formalar qilinsa —
   `leads_retrieval`, `pages_manage_ads`). Har ruxsatga yozma tushuntirish va
   ekran yozuvi kerak.
4. **Marketing API Access Tier**: Limited → **Full** — 15 kunda 500+ chaqiruv
   va xatolik <15%.
   ⚠️ **Tovuq va tuxum**: chaqiruv uchun mijoz kerak, mijoz uchun kvota kerak.
   Shuning uchun birinchi mijozlar ataylab kam bo'ladi (P1 da 5–10 restoran).

---

## 2. Arxitektura: nima qayerda turadi

| Nima | Qayerda | Nega |
|---|---|---|
| Ilova sirlari (`app_id`, `app_secret`, `config_id`) | **Konsol** (`control`) | AI kaliti bilan bir qoida: N ta konteynerda N ta sir emas, va rotatsiya bir joyda |
| Mijozning tokeni, `ad_account_id`, `page_id`, `pixel_id` | **Tenant** bazasi, alohida kolleksiya | Bu mijozning aktivi; `payment_settings` naqshi — javobda faqat `hasToken` |
| Meta bilan gaplashish | **Tenant serveri** | Kvota akkaunt bo'yicha; har restoran o'z kvotasini sarflaydi |
| AI (variantlar, tahlil) | **Konsol** orqali | Kalit bizda, limit bizda — brifing va maslahatchi bilan bir eshik |
| Raqamlar (nima sotildi, marja, qaysi taom) | **Tenant** | Maslahatchidagi chok: **raqamni biz hisoblaymiz, modeldan faqat so'z keladi** |

⚠️ **Model hech qachon Meta'ga to'g'ridan-to'g'ri yozmaydi.** Model matn va
variant beradi; API chaqiruvini server qiladi, va byudjet chegarasi serverda
qattiq.

---

## 3. Ma'lumot modeli (tenant)

- `ads_settings` (singleton) — `token` (sir), `businessId`, `adAccountId`,
  `pageId`, `instagramId`, `pixelId`, `connectedAt`, `connectedBy`,
  `lastCheckAt`, `status`.
  ⚠️ Javobda **hech qachon token qaytmaydi** — faqat `hasToken`.
- `ad_campaign` — bizda yaratilgan har kampaniya: `metaCampaignId`,
  `adsetId`, `adId`, `creativeId`, maqsad, byudjet, geo, tanlangan taom(lar),
  ega tasdiqlagan vaqti, holati.
  ⚠️ **Meta'dagi nusxa emas, bizning qarorimiz yozuvi**: nima uchun shu taom,
  qaysi variant tanlandi, qancha byudjet ruxsat berildi.
- `ad_daily` — kunlik insights surati (`spend`, `impressions`, `clicks`,
  `actions`), kampaniya × kun bo'yicha, **unique** indeks bilan.
  ⚠️ Kesh emas, **tarix**: Meta insights'ni vaqt bo'yicha qayta yozadi, biz
  esa «o'sha kuni nima ko'rgan edik» degan savolga javob bera olishimiz kerak.
- `ad_advice` — AI bergan variantlar va ega tanlagani (kunlik limit shu yerdan
  ham hisoblanadi).

---

## 4. Endpointlar

**Tenant (panel):**
```
GET    /admin/ads/state            ulanganmi, nima yetishmayapti
POST   /admin/ads/connect          Login for Business'dan kelgan code
DELETE /admin/ads/connect          uzish
GET    /admin/ads/assets           sahifalar, akkauntlar, piksellar ro'yxati
POST   /admin/ads/plan             AI: reja + variantlar (kunlik limit shu yerda)
POST   /admin/ads/campaigns        tanlangan variantni Meta'da yaratish
PUT    /admin/ads/campaigns/{id}   pauza / byudjet / to'xtatish
GET    /admin/ads/campaigns        ro'yxat + natija
GET    /admin/ads/report           sarf ↔ buyurtma ↔ foyda
```

**Konsol (`/internal`):** `POST /internal/ads-advice` — brifing va
maslahatchi bilan bir xil naqsh (entitlement + kunlik limit + bayt-bayt bir xil
tizim prompti).

---

## 5. Ega tanlaydi, AI variant beradi

| Qadam | AI beradi | Ega qiladi |
|---|---|---|
| Maqsad | yetkazib berish / zalga tashrif / lid | bittasini bosadi |
| Nimani reklama qilish | **3 ta taom** — sotuvi, marjasi va fotosi bo'yicha | tanlaydi yoki o'zi qo'shadi |
| Geo | filialdan 3 / 5 / 10 km yoki shahar | radiusni suradi |
| Byudjet | 3 ta variant + taxminiy qamrov | raqamni qo'yadi va **maksimum** belgilaydi |
| Matn + rasm | 3 ta matn × mavjud taom fotolari | bittasini tanlaydi |
| Chiqarish | tekshiruv ro'yxati | **«Chiqarish»** ni bosadi |

⚠️ **Kreativ yaratilgandan keyin o'zgartirilmaydi** (Meta qoidasi) — shuning
uchun tanlov **yaratishdan oldin** bo'lishi shart. Bu talab emas, bu API.

⚠️ **Rasm generatsiya qilinmaydi**: restoranning o'z taom fotolari allaqachon
bizda (`uploads/`). Yo'q bo'lsa — ekran shuni aytadi, o'ylab topmaydi.

---

## 6. UI va progress barlar (faqat haqiqiylari)

⚠️ **Yolg'on progress bar — ishonchni eng tez yo'qotadigan narsa.** Quyidagilar
o'lchanadigan narsalar:

1. **Ulanish**: Sahifa → reklama akkaunti → to'lov usuli → ruxsat → piksel
   (5 qadam, har biri Meta'dan tekshiriladi).
2. **Kampaniya yig'ilishi**: maqsad → geo → byudjet → matn → rasm → tekshiruv.
3. **O'rganish bosqichi** — Meta'ning `learning_stage_info` si, ~50 konversiya.
4. **Kunlik byudjet sarfi** — bugun 42 000 / 100 000.
5. **Sarf ↔ foyda** — ikki tomonlama bar, nol chizig'i bilan.
6. **Kunlik AI limiti** — 12/30 (`AIQuota` komponenti allaqachon shunday).

---

## 7. Sotilishi va limit

- `billing.AddonAds = "ads"`, `AddonPrice(AddonAds) = 1_250_000`.
  ⚠️ `AddonPrice` — **allowlist**: narxsiz qo'shimcha konsolda ko'rinadi-yu
  saqlanmaydi (`cleanAddons` uni tashlaydi), va buning testi bor.
- Konsolda: `TillPanel.tsx` ga katakcha — AI qo'shimchasi yonida.
- `TenantTill.Addons` → `mirrorTill` → tenantning `subscription` hujjati.
- Kunlik 30 ta: `BriefingLog` naqshi, ⚠️ **alohida hisoblagich** — ertalabki
  brifing reklama tahlilini yeb qo'ymasligi uchun.
- Panelda: yon panelga «Reklama» qatori, `PANEL_ROUTES` ga `/admin/ads → ads`,
  sotib olinmaganda `UpgradeCard`.
  ⚠️ Nom bo'sh — `/admin/campaigns` **band** (u CRM xabarlari).

---

## 8. Qo'riqlar

1. ⚠️ **Byudjet chegarasi serverda.** Modeldan kelgan raqam hech qachon
   to'g'ridan-to'g'ri Meta'ga ketmaydi; ega qo'ygan kunlik maksimumdan oshirib
   bo'lmaydi. Bitta nolsiz kampaniya — bitta yo'qolgan mijoz.
2. ⚠️ **Token uzilishini ko'rsatish.** Muddatsiz token ham bekor qilinishi
   mumkin (ega ruxsatni olib tashlasa). Uzilganda Meta kampaniyani
   **to'xtatmaydi** — biz ko'r bo'lib qolamiz, ya'ni bu birinchi kundan
   ogohlantirish bo'lishi kerak.
3. **Kvota**: Limited darajada `ads_management` soatiga 300 — statistika
   **kunlik surat + kesh**, har ochilishda so'rov emas.
4. **Rad etilgan reklama**: Meta sababni API'da qaytaradi — uni **odam tiliga**
   tarjima qilish kerak, «rejected» degan so'z foydasiz.
5. **Yangi akkaunt**: uchinchi tomondan kampaniya yaratishga darhol ruxsat
   bermaydi — ulanish ekrani buni **oldindan aytadi**.

---

## 9. Bosqichlar

| Bosqich | Nima chiqadi | Meta'dan kerak |
|---|---|---|
| **P0** ✅ | «Reklama» bo'limi, AI reja va variantlar, ega Ads Manager'da o'zi bosadi | **hech nima** |
| **P1** | Ulanish (Login for Business), insights o'qish, piksel + CAPI, «qancha buyurtma keltirdi» | `ads_read` |
| **P2** | Kampaniyani biz yaratamiz, pauza, byudjet | `ads_management` + review |
| **P3** | Kunlik optimizatsiya qoidalari | o'shaning ustiga |

⚠️ **P0 vaqtinchalik yechim emas** — u review kutilayotganda **sotiladigan**
mahsulot, va u bilan biz Full darajaga kerak bo'lgan chaqiruvlarni ham,
birinchi mijozlarni ham yig'amiz.

---

## 10. Ochiq savollar

1. **Conversions API parametrlari o'qilmagan** (`docs/vendor/meta-marketing.md`
   §10). P1 dan oldin o'qilishi shart — «47 ta buyurtma» degan raqam qaysi
   maydondan olinishini taxmin qilib bo'lmaydi.
2. **Lead-forma yo'li** (targetolog.ai shu modelda ishlaydi) — qo'shimcha ikki
   ruxsat va Page token talab qiladi. Restoranga lid kerakmi yoki buyurtmami —
   hal qilinmagan.
3. **Bir restoran, bir akkaunt** deb faraz qilyapmiz. Zanjir (bir brend, to'rt
   filial) bitta akkauntdan reklama qiladimi yoki har filial o'zinikidanmi?
4. **QQS**: O'zbekistonda reklamaga QQS qo'shiladi (akkauntga VAT ID
   kiritilmasa). Bu mijozning xarajati, lekin ekranda aytilishi kerakmi?
5. **Javobgarlik chegarasi**: reklama matni mijoz nomidan chiqadi. Ofertaga
   bir band kerakmi?

---

## Manbalar

- `docs/vendor/meta-marketing.md` — API kontraktlari (o'qilgan, 2026-09-16)
- [Marketing API access tiers](https://developers.meta.com/blog/updates-to-ads-management-standard-access-feature/)
- [Rate limiting](https://developers.facebook.com/docs/marketing-api/overview/rate-limiting/)
- [Facebook Login for Business](https://developers.facebook.com/documentation/facebook-login/facebook-login-for-business)
- [Meta verification documents (ikkilamchi manba)](https://singhamandeep.com/meta-business-verification-documents-required/)
- [Verification steps (ikkilamchi manba)](https://support.wati.io/en/articles/11462440-what-are-the-steps-to-complete-facebook-business-verification)
- [Rejection reasons (ikkilamchi manba)](https://chakrahq.com/article/meta-business-verification-rejected-reasons/)

⚠️ Verifikatsiya bo'limidagi qadamlar **ikkilamchi manbalardan** — Meta'ning
o'z yordam sahifalari JS bilan chiziladi va WebFetch ularning matnini
qaytarmaydi. Portfel ichidagi ko'rsatma bilan solishtirilsin.
