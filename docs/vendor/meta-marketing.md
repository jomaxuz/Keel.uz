# Meta Marketing API — reklama (targetolog) uchun o'qilgan nusxa

Manba: `developers.facebook.com` va `developers.meta.com` (1–9-bo'limlar
o'qildi **2026-09-16**, 10–12-bo'limlar **2026-09-20** — Conversions API,
insights maydonlari, `promoted_object` va valyuta birliklari).
Havolalar har bo'limning tagida.

⚠️ **Bu yerda faqat o'qilgan narsa bor.** O'qilmagani oxirgi bo'limda ro'yxat
bilan yozilgan — taxmin qilingan endpoint kompilyatsiya bo'ladi, review'dan
o'tadi va **bironta ham reklama chiqarmaydi** (fiskal provayderlardagi bilan bir
qoida).

---

## 1. Model: biz mijozning reklama akkauntida ishlaymiz

Restoran o'z Meta biznes-portfeliga va reklama akkauntiga egalik qiladi,
reklamaga pulni **Meta'ga o'zi** to'laydi. Biz uchinchi tomon vositasimiz.

Bundan kelib chiqadigan ikkita darvoza:

- **App Review + Business Verification.** O'zimiz egasi bo'lmagan akkauntga
  tegadigan har bir ruxsat (`ads_management`, `ads_read`, `business_management`)
  review talab qiladi; undan oldin ilova egasining biznes-portfeli **huquqiy
  hujjatlar bilan** tasdiqlanishi kerak.
- **Marketing API Access Tier** — ruxsatdan **alohida** ikkinchi darvoza
  (2026-05-04 dan nomi shunday; ilgari «Ads Management Standard Access»):
  - `Limited Access` (eski nomi Standard) — standart holat;
  - `Full Access` (eski nomi Advanced) — **oxirgi 15 kunda 500+ Marketing API
    chaqiruvi** va **xatolik darajasi <15%** (oxirgi 500 chaqiruv bo'yicha
    siljiydigan oyna). Ekran yozuvi (screen recording) endi talab qilinmaydi.

⚠️ **Tovuq va tuxum:** Full uchun chaqiruv kerak, chaqiruv uchun mijoz kerak.
Limited darajada kvota kichik (2-bo'limga qarang) — ya'ni birinchi mijozlar
ataylab kam bo'lishi kerak.

Havolalar:
<https://developers.meta.com/blog/updates-to-ads-management-standard-access-feature/>

---

## 2. Kvota va rate limit

Kvota **reklama akkaunti** bo'yicha, soatlik oynada, «business use case» (BUC)
bo'yicha hisoblanadi:

| Use case | Full Access | Limited (dev) | Ustiga |
|---|---|---|---|
| `ads_management` | 100 000 | **300** | + 40 × faol reklama |
| `custom_audience` | ≥190 000 (maks 700 000) | 5 000 | + 40 × faol auditoriya |
| `ads_insights` | 190 000 | 600 | + 400 × faol reklama |

- Ball tizimi: akkaunt boshiga Limited'da **60**, Full'da **9 000**.
- Cheklovga urilganda xato kodlari: `17`, `613` (subcode 2446079 / 1487742),
  QPS mutatsiyasi uchun `613/5044001`, insights uchun `4` (1504022 / 1504039),
  BUC uchun `80000, 80003, 80004, 80014`.
- Sarfni o'lchaydigan sarlavhalar: `X-Ad-Account-Usage`
  (`acc_id_util_pct`, `reset_time_duration`, `ads_api_access_tier`),
  `X-Business-Use-Case`, `X-FB-Ads-Insights-Throttle`.

⚠️ **Statistikani har daqiqada so'rab bo'lmaydi.** Limited darajada soatiga 300
chaqiruv — bu kunlik surat + kesh degani (maslahatchidagi naqsh).

Havola: <https://developers.facebook.com/docs/marketing-api/overview/rate-limiting/>

---

## 3. Ulanish: Facebook Login for Business

Tech Provider uchun tavsiya etilgan yo'l. App Dashboard'da **konfiguratsiya**
yaratiladi (qaysi token turi, qaysi aktivlar, qaysi ruxsatlar) va u
**`config_id`** oladi.

Login dialogi parametrlari:

- `config_id=<KONFIGURATSIYA>` — ⚠️ `scope` **ishlatilmaydi**, uni almashtiradi;
- `response_type=code` — system user tokeni authorization-code grant talab
  qiladi;
- `override_default_response_type=true`.

Qaytgan `code` **serverdan** tokenga almashtiriladi:

```
GET https://graph.facebook.com/v26.0/oauth/access_token
    ?client_id=<APP_ID>&client_secret=<APP_SECRET>&code=<CODE>
```

Token turlari:

- **User Access Token** — odamning shaxsiy hisobiga bog'langan, qisqa umrli
  (uzaytirilsa ~60 kun);
- **Business Integration System User (BISU)** — mijozning **biznes-portfeliga**
  bog'langan, ⚠️ **standart holda muddatsiz**, server-server ish uchun.

⚠️ **Bizga BISU kerak.** 60 kunlik token — bu ikki oyda jimgina to'xtaydigan
avtomatika; Meta esa kampaniyani ishlatishda davom etadi, biz ko'r bo'lib
qolamiz.

Havolalar:
<https://developers.facebook.com/documentation/facebook-login/facebook-login-for-business>
<https://developers.facebook.com/docs/business-management-apis/system-users/install-apps-and-generate-tokens/>

---

## 4. Obyektlar zanjiri

`campaign` → `ad set` → `ad` → `ad creative` (+ `ad image`).

### 4.1 Kampaniya — `POST /act_{ad_account_id}/campaigns`

Majburiy: `name`, `objective`, **`special_ad_categories`** (massiv; oddiy
restoran uchun `["NONE"]` — ⚠️ maydonning o'zi **majburiy**, bo'sh qoldirib
bo'lmaydi).

`objective` qiymatlari (o'qilgan ro'yxat): `OUTCOME_SALES`, `OUTCOME_LEADS`,
`OUTCOME_TRAFFIC`, `OUTCOME_AWARENESS`, `OUTCOME_ENGAGEMENT`,
`OUTCOME_APP_PROMOTION` + eski to'plam (`CONVERSIONS`, `LINK_CLICKS`,
`MESSAGES`, `LEAD_GENERATION`, `REACH`, `STORE_VISITS`, …). ⚠️ Eskilari
`OUTCOME_*` bilan almashtirilmoqda.

`special_ad_categories`: `NONE`, `EMPLOYMENT`, `HOUSING`, `CREDIT`,
`ISSUES_ELECTIONS_POLITICS`, `ONLINE_GAMBLING_AND_GAMING`,
`FINANCIAL_PRODUCTS_SERVICES`.

`status`: yaratishda `ACTIVE` yoki `PAUSED` (`DELETED`/`ARCHIVED` — faqat
yangilashda). `buying_type`: `AUCTION` (standart) | `RESERVED`.

Byudjet: `daily_budget` yoki `lifetime_budget` **kampaniya darajasida** (CBO) —
⚠️ kampaniya va ad set darajasida **bir vaqtda** byudjet qo'yib bo'lmaydi.

⚠️ **`is_adset_budget_sharing_enabled` — byudjet ad set darajasida bo'lsa
majburiy** (o'qildi 2026-09-20, jonli xatodan): «You must specify True or False
in the field is_adset_budget_sharing_enabled if you are not using campaign
budget.» `true` — ad setlar bir-biriga byudjetining 20% ini bera oladi.
Bizda **`false`**: kampaniyada bitta ad set bor, ya'ni beradigan kishi yo'q, va
ega qo'ygan chegara taxminiy bo'lib qolmasligi kerak.

Havola: <https://developers.facebook.com/docs/marketing-api/reference/ad-campaign-group/>

### 4.2 Ad set — `POST /act_{ad_account_id}/adsets`

Majburiy: `name` (≤400 belgi), `campaign_id`, `targeting` (⚠️ **kamida bitta
davlat** bo'lishi shart), `billing_event`, `optimization_goal`, va
`daily_budget` **yoki** `lifetime_budget`.

`billing_event`: `IMPRESSIONS`, `LINK_CLICKS`, `CLICKS`, `POST_ENGAGEMENT`,
`THRUPLAY`, `PURCHASE`, `APP_INSTALLS`, `PAGE_LIKES`, `OFFER_CLAIMS`,
`LISTING_INTERACTION`, `NONE`.

`optimization_goal` (o'qilgan ro'yxatdan keraklilari): `OFFSITE_CONVERSIONS`,
`LINK_CLICKS`, `LANDING_PAGE_VIEWS`, `REACH`, `IMPRESSIONS`,
`LEAD_GENERATION`, `QUALITY_LEAD`, `CONVERSATIONS`, `POST_ENGAGEMENT`,
`VALUE`, `PROFILE_VISIT`, `AUTOMATIC_OBJECTIVE`.

Qo'shimcha: `bid_strategy`, `bid_amount`, `promoted_object`, `start_time`,
`end_time`, `status`.

Havola: <https://developers.facebook.com/docs/marketing-api/reference/ad-campaign/>

### 4.3 Targeting spec

```json
{
  "geo_locations": {
    "countries": ["UZ"],
    "cities": [{ "key": "2430536", "radius": 12, "distance_unit": "mile" }],
    "custom_locations": [
      { "latitude": 41.31, "longitude": 69.24, "radius": 5,
        "distance_unit": "kilometer" }
    ]
  },
  "age_min": 25,
  "age_max": 40,
  "genders": [1],
  "publisher_platforms": ["facebook", "instagram"],
  "targeting_automation": { "advantage_audience": 1 }
}
```

- ⚠️ **Davlat va o'sha davlat ichidagi shahar birga berilmaydi** — Meta buni
  «overlap» deb rad etadi.
- `targeting_automation.advantage_audience: 1` — **auditoriyani Meta AI tanlaydi**,
  bizdan faqat geo qoladi. 2026 da bu standart shakl.
- Shahar `key` si **qidiruv orqali** topiladi:
  `GET /search?type=adgeolocation&q=Toshkent&location_types=["city"]`.
  ⚠️ Javob **koordinata qaytarmaydi** — xaritada ko'rsatish kerak bo'lsa,
  o'zimizning xarita provayderimiz ishlatiladi (`lib/map/`).

Havolalar:
<https://developers.facebook.com/docs/marketing-api/targeting>
<https://protarik.medium.com/fixing-the-locations-overlap-error-in-meta-ads-api-a-quick-guide-3a1934ee0826>

### 4.4 Rasm — `POST /act_{ad_account_id}/adimages`

- Yuklash: `bytes` (Base64 UTF-8 satr) yoki boshqa akkauntdan `copy_from`.
- ⚠️ **Fayl nomida kengaytma bo'lishi shart** (`sample.jpg`, `sample` emas).
- Javob: `hash`, `url`, `url_128`, `url_256`, `height`, `width`, `name`.
- ⚠️ Javobdagi `url` **vaqtinchalik** va kreativda ishlatilmaydi — faqat `hash`.

Havola: <https://developers.facebook.com/docs/marketing-api/reference/ad-image/>

### 4.5 Kreativ — `POST /act_{ad_account_id}/adcreatives`

```json
{
  "object_story_spec": {
    "page_id": "<PAGE_ID>",
    "instagram_user_id": "<IG_ID>",
    "link_data": {
      "image_hash": "<HASH>",
      "link": "https://<restoran-domeni>/menu",
      "message": "<matn>",
      "name": "<sarlavha, 1..90>",
      "description": "<qo'shimcha>",
      "call_to_action": { "type": "ORDER_NOW", "value": { "link": "..." } }
    }
  }
}
```

⚠️ **Kreativ yaratilgandan keyin o'zgartirilmaydi** — matnni tahrirlash =
yangi kreativ. Ya'ni «AI variant beradi, ega tanlaydi» oqimida tanlangan
variant **yaratishdan oldin** tanlanishi kerak.

Havola: <https://developers.facebook.com/docs/marketing-api/reference/ad-creative/>

### 4.6 Reklama — `POST /act_{ad_account_id}/ads`

Majburiy: `name`, `adset_id`, `creative` (`{"creative_id": "<ID>"}`), `status`.
Qo'shimcha: `tracking_specs`, `conversion_domain` (⚠️ piksel bilan ma'lumot
bo'lishadigan kampaniyalar uchun **majburiy**; faqat domen, to'liq URL emas).

⚠️ Yangi reklama **`pending`** holatida bo'ladi va Meta tasdiqlamaguncha
chiqmaydi.

Havola: <https://developers.facebook.com/docs/marketing-api/reference/adgroup/>

---

## 5. Natija — Insights

`GET /{ad-account-id}/insights` yoki `GET /{campaign-id}/insights`.

- Vaqt: `date_preset` (masalan `last_7d`) yoki `time_range`. Standart — oxirgi
  30 kun.
- Maydonlar: `spend`, `impressions`, `clicks`, `ctr`, `cpc`, `actions`,
  `action_values`, `cost_per_action_type`, `purchase_roas`.
- ⚠️ Faqat kerakli maydonlar so'raladi: ortiqchasi so'rovni sekinlashtiradi va
  kvotani yeydi.

Havola: <https://developers.facebook.com/docs/marketing-api/insights/>

---

## 6. Lead ads (agar «lid yig'ish» qilinsa)

- Forma avval yaratiladi, keyin reklamaga bog'lanadi.
- Lidlar: `GET /{form_id}/leads` (davriy) yoki **`leadgen` webhook** (real
  vaqtda; webhook faqat xabar beradi, lidning o'zi API'dan olinadi).
- ⚠️ **Qo'shimcha ruxsatlar**: `leads_retrieval` + `pages_manage_ads`, ikkalasi
  ham review'dan o'tadi; undan keyin Business Verification.
- ⚠️ **Page access token** ishlatiladi (user token emas) — rate limit'i
  yaxshiroq. Development rejimidagi ilova begona sahifadan lid **ololmaydi**.

Havola: <https://developers.facebook.com/docs/marketing-api/guides/lead-ads/>

---

## 7. Versiya

- Graph/Marketing API **v25.0** — 2026-02-18, **v26.0** — 2026-07-29.
- Har versiya ~2 yil yashaydi: v20.0 — 2026-09-24 da, v21.0 — 2027-01-21 da
  o'chadi.
- ⚠️ **v26.0 ning buzuvchi o'zgarishlari 2026-10-27 dan qolgan barcha
  versiyalarga ham tarqaydi** — ya'ni eski versiyada qolib «o'zgarishdan
  qochish» ishlamaydi.
- v26.0 da: 47 ta Commerce Order Management endpointi bloklandi, Instagram
  Explore Feed va Messenger Stories joylashuvlari olib tashlandi, poll-reklama
  yaratish bloklandi.

**Biz `v26.0` ga bog'laymiz** va versiyani bitta konstantada saqlaymiz.

Havolalar:
<https://developers.facebook.com/docs/graph-api/changelog/version25.0/>
<https://ppc.land/meta-blocks-47-commerce-endpoints-as-graph-api-v26-0-lands-today/>

---

## 8. Advantage+ (2026 dagi shakl)

- 2025-05-29: ASC/AAC oqimlari **bitta** kampaniya yaratish oqimiga
  birlashtirildi; farqni avtomatlashtirish sozlamalari (byudjet, auditoriya,
  joylashuv) belgilaydi.
- Marketing API **v24.0** (2025-10-08): eski uslubdagi kampaniya yaratish
  **taqiqlandi**.
- **v25.0** (2026-Q1): ASC/AAC yaratish **barcha versiyalarda** buziladi.
- Auditoriya: faqat geo, yoki to'liq avtomatika
  (`targeting_automation.advantage_audience: 1`). Auditoriyani kengaytirish —
  standart xatti-harakat.

⚠️ **Xulosa mahsulot uchun:** «AI auditoriyani tanlaydi» — bu bizning
ustunligimiz emas, bu API bayrog'i. Bizning ustunligimiz — **qaysi taomni,
qanday byudjet bilan, qaysi hududga** reklama qilish kerakligini restoranning
o'z sotuvi, tannarxi va mijoz bazasidan bilishimiz.

Havolalar:
<https://ppc.land/meta-launches-unified-api-structure-for-advantage-campaigns/>
<https://ppc.land/meta-deprecates-legacy-campaign-apis-for-advantage-structure/>

---

## 9. Reklama akkauntining o'z shartlari

- Akkauntda **Sahifa** va **to'lov usuli** bo'lishi shart; to'lov usuli yo'q
  akkaunt reklama chiqara olmaydi.
- ⚠️ **Yangi akkauntlar** uchinchi tomon vositalaridan kampaniya yaratishga
  darhol ruxsat bermaydi — Meta avval akkauntning o'z interfeysida tarix
  to'plashini talab qiladi.
- To'lov usulini qo'shish uchun **admin** roli kerak (advertiser yetmaydi).
- O'zbekiston: reklamaga **QQS** qo'shiladi; yuridik shaxs VAT ID kiritsa,
  qo'shilmaydi.

Havolalar:
<https://www.facebook.com/business/help/1408263965992408>
<https://www.facebook.com/business/help/262717434999140>

---

## 10. Conversions API (o'qildi 2026-09-20)

Endpoint:

```
POST https://graph.facebook.com/v26.0/{PIXEL_ID}/events
```

Token `access_token` query parametrida ham qabul qilinadi — ⚠️ **biz sarlavhada
yuboramiz**: query'dagi token yo'ldagi har bir proksi va loglarda qoladi, va bu
pul sarflay oladigan kalit.

Tana:

```json
{ "data": [ {
  "event_name": "Purchase",
  "event_time": 1633552688,
  "event_id": "A-1042",
  "action_source": "website",
  "user_data": { "ph": ["<sha256>"], "em": ["<sha256>"],
                 "client_ip_address": "…", "client_user_agent": "…" },
  "custom_data": { "value": 84000, "currency": "UZS",
                   "order_id": "A-1042",
                   "content_ids": ["…"], "content_type": "product" }
} ] }
```

`action_source`: `website | app | phone_call | physical_store | offline`.
`test_event_code` — **ildizda**, faqat sinov uchun; u bilan yuborilgan hodisa
hech qayerda hisoblanmaydi.

### 10.1 Normalizatsiya va hash (⚠️ aynan shunday)

Hash qilinadiganlar — **SHA-256**, oldin normalizatsiya:

| Maydon | Qoida |
|---|---|
| `em` | trim + kichik harf |
| `ph` | faqat raqam, **davlat kodi bilan**, oldidagi nollar olib tashlanadi |
| `fn`, `ln` | kichik harf, tinish belgisiz |
| `ct` | kichik harf, **bo'shliqsiz**, maxsus belgisiz |
| `st` | 2 harfli kod, kichik |
| `zp` | kichik, bo'shliqsiz, tiresiz |
| `country` | ISO 3166-1 alpha-2, **kichik** (`uz`) |
| `db` | `YYYYMMDD` |
| `ge` | `f` / `m` |
| `external_id` | hash tavsiya etiladi (majburiy emas) |

Hash **qilinmaydiganlar**: `client_ip_address`, `client_user_agent`, `fbc`,
`fbp`, `lead_id`, `page_id`, `subscription_id` va shu qatordagilar.

⚠️ **O'zbek raqami uch xil saqlanadi** (`+998…`, `998…`, `90 123 45 67`) va
uchalasi bitta odam. Davlat kodisiz ketgan raqam **hech kimga mos kelmaydi**, va
buning belgisi yo'q: hodisa qabul qilinadi, buyurtma shunchaki atributsiya
qilinmaydi.

### 10.2 Deduplikatsiya

Piksel va server bitta buyurtmani ikki marta yuboradi. Meta ularni **faqat
`event_id` bir xil bo'lganda** birlashtiradi (`event_name` bilan birga).
Bizda `event_id` = buyurtma raqami.

Havolalar:
<https://developers.facebook.com/docs/marketing-api/conversions-api/using-the-api/>
<https://developers.facebook.com/docs/marketing-api/conversions-api/parameters/customer-information-parameters/>

---

## 11. Insights maydonlari va `promoted_object` (o'qildi 2026-09-20)

- `GET /{campaign-id}/insights?fields=spend,impressions,clicks,actions,action_values`
  `&time_increment=1&time_range={"since":"…","until":"…"}`.
- `level`: `campaign | adset | ad` — **parametr**, maydon emas.
- `actions` — `[{action_type, value}]`. ⚠️ **Bizga kerakligi
  `offsite_conversion.fb_pixel_purchase`**, `purchase` emas: birinchisi
  pikselga (ya'ni bizning serverimiz yuborgan hodisalarga) tegishli, ikkinchisi
  «omni» — Meta har qayerda bo'lgan deb hisoblagan xaridlar. Ikkalasini
  aralashtirish buyurtmalar soni buyurtmalar ro'yxatiga to'g'ri kelmaydigan
  hisobot beradi, va ega aynan shuni solishtiradi.
- `purchase_roas` ham **piksel bo'yicha** hisoblanadi.

`promoted_object` (`OFFSITE_CONVERSIONS` uchun **majburiy**) — uchta
kombinatsiyadan biri:

1. `pixel_id` + `custom_event_type` (standart hodisa),
2. `pixel_id` + `custom_event_type: OTHER` + `custom_event_str`,
3. `application_id` + `object_store_url` + `custom_event_type`.

`custom_event_type` qiymatlari: `PURCHASE`, `LEAD`, `COMPLETE_REGISTRATION`,
`ADD_TO_CART`, `INITIATED_CHECKOUT`, `CONTENT_VIEW`, `SEARCH`, `SUBSCRIBE`,
`START_TRIAL`, `CONTACT`, `FIND_LOCATION`, `SCHEDULE`, `DONATE`,
`SERVICE_BOOKING_REQUEST`, `MESSAGING_CONVERSATION_STARTED_7D`, `OTHER` va h.k.

Havolalar:
<https://developers.facebook.com/docs/marketing-api/insights/>
<https://developers.facebook.com/documentation/ads-commerce/marketing-api/reference/ad-campaign>

---

## 12. ⚠️ Valyuta: byudjet **minor unit** da, va UZS ro'yxatda yo'q

Meta byudjetni **reklama akkaunti valyutasining eng kichik birligida** oladi:
`daily_budget: 50000` dollar akkauntida **$500.00**, sentsiz valyutada esa
50 000. Meta'ning «Currency Codes» sahifasida offset jadvali bor, lekin **UZS u
yerda yo'q** — ya'ni O'zbekistondagi restoranning akkaunti amalda **USD** da
bo'ladi.

Bundan ikkita qaror chiqadi:

1. ⚠️ **Kursni biz o'ylab topmaymiz.** Reja so'mda taklif qiladi (restoran shunda
   sanaydi), byudjet esa **akkaunt valyutasida** kiritiladi. Taxminiy kurs bilan
   qilingan konvertatsiya ekranda haqiqiy raqamdek turardi.
2. ⚠️ **Tekshiruv akkauntning o'zidan olinadi**: `min_daily_budget` (minor unit)
   — uni Meta o'zi qaytaradi, ya'ni undan o'tgan byudjet Meta kutgan birlikda.
   Offset jadvalidagi xato shu yerda «byudjet juda kichik» bo'lib ushlanadi,
   yuz barobar ortiqcha sarf bo'lib emas.

Havola: <https://developers.facebook.com/docs/marketing-api/currencies/>

---

## 13. ⚠️ O'qilmagani (kod yozishdan oldin o'qilishi shart)

| Nima | Nega kerak |
|---|---|
| Ad Account'ni **dasturiy yaratish** (`business_management` Full Access) | Hozircha rejada yo'q — restoran o'z akkauntini o'zi yaratadi |
| Lead forma yaratish API'sining aniq shakli | Lead-reklama qilinsa |
