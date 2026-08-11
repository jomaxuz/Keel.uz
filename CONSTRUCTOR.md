# CONSTRUCTOR.md — sayt konstruktori va Telegram mini app

Ikki xususiyat, bitta fayl — chunki ular **bir narsaning ikki yuzi**: mini app
restoranning saytini Telegram ichida ko'rsatadi, ya'ni konstruktor chizadigan
har bir maket mini app'ning ham maketi. Ularni alohida rejalashtirish
konstruktorni desktopga, mini app'ni esa telefonga qurishga olib boradi va
ikkinchisi birinchisini buzadi.

---

## 0. Hozir nima bor (boshlanish nuqtasi)

Sayt — **bitta shablon**, va restoran undagi **9 ta sozlagichni** o'zgartiradi
(`restaurant.theme`): aksent rangi, dark rejim aksenti, burchak radiusi, tugma
shakli, shrift juftligi (3 tayyor), fon ohangi (4 ta), soya, tugma
to'ldirilishi, umumiy o'lcham. Ustiga 5 ta tayyor mavzu — ular shu
maydonlarning to'plami, xolos.

Ya'ni **rang va shriftni o'zgartirish mumkin, maketni — yo'q**: bloklarning
tartibi, qaysi bo'lim bor-yo'qligi, hero qanday ko'rinishi — hammasi kodda.

Telegram bo'yicha kodda hech nima yo'q: `Socials.Telegram` — footerdagi havola.
Ya'ni mini app noldan.

---

## ⚠️ 1. Birinchi qaror: "canvas" nimani anglatadi

Bu rejadagi eng muhim savol, va u ikkinchi xususiyatga bog'liq.

**Erkin canvas (piksel bo'yicha joylashtirish) mobil responsive bo'la
olmaydi.** Ega 1400 px ekranda bloklarni sudrab qo'ysa, natijani 380 px
Telegram WebView'ga qayta joylashtirishning **matematik yo'li yo'q** — brauzer
egadan nima nazarda tutganini so'ray olmaydi. Bu narsa Wix'ning eski
muharririda aynan shunday tugagan: har sayt ikkinchi marta, qo'lda, telefon
uchun chiziladi va ko'pchilik uni chizmaydi.

Uch variant bor:

| Variant | Chizuvchi nima his qiladi | Mobil natija |
|---|---|---|
| **(a) Bo'limlar ro'yxati** — bo'lim tanlash, tartib, ko'rinish varianti | "sozlayapman" | mukammal |
| **(b) Erkin canvas + ikkinchi mobil maket** | "chizayapman" | ikkinchi maket chizilsa yaxshi, chizilmasa buzuq |
| **(c) Panjarali canvas (grid snap) + avtomatik mobil yig'ilish** | "chizayapman" | mukammal |

**Qaror qabul qilindi: (c) — panjarali canvas.** 12 ustunli panjara, bloklar ustunlarga yopishadi, kenglik
ustun bilan o'lchanadi. Telefonda esa **o'sha tartibda bitta ustunga**
yig'iladi. Chizuvchi o'zini erkin his qiladi, chiqadigan narsa esa responsive
bo'lib qoladi — chunki maket pikselda emas, **ustunlarda** yozilgan.

Va konstruktorning yonida **telefon ramkasi** doim turadi: mini app o'sha
zahoti ko'rinadi, "keyin tekshiraman" degan qadam yo'qoladi.

⚠️ Bu qaror dizaynni kim chizishidan **qat'i nazar** to'g'ri qoladi. Panjara —
egadan himoya emas, **fizikadan** kelib chiqadigan cheklov: 380 px da bitta
ustundan boshqa narsa yo'q, va buni chizuvchi kim bo'lishidan qat'i nazar
kimdir hisobga olishi kerak. Panjara buni avtomatik qiladi.

---

## 2. Kim chizadi: Keel, pul evaziga

**Dizaynni Keel konsolidan biz chizamiz va alohida pul olamiz. Restoran egasi
o'z panelidan dizayn chiza olmaydi.** Bu qaror rejaning yarmini o'zgartiradi:

- **Konstruktor UI konsolda** (`keel-site/console`), tenantning `/admin` ida
  emas.
- **Ma'lumot esa tenantning o'z bazasida** (`page_design`), va uni konsol
  to'g'ridan-to'g'ri yozadi — aynan `export_grant` bilan bir naqsh: bitta
  hujjat, bitta yozuvchi (konsol), bitta o'quvchi (tenant ilovasi). Sinxronlash
  muammosi yo'q va **tenant konteyneridan control plane'ga yangi yo'l
  ochilmaydi** (o'sha yo'lning yo'qligi bir restoranni ikkinchisiga yeta
  olmaydigan qiladi).
- **Shablon galereyasi endi ixtiyoriy emas, asosiy vosita**: bir maketni bir
  necha mijozga qayta ishlatish — bu ishning tijorat ma'nosi. "Shablon sifatida
  saqlash" va "mijozga qo'llash" birinchi darajali xususiyat.
- **Konstruktor "ahmoqdan himoyalangan" bo'lishi shart emas** — uni o'z
  jamoangiz ishlatadi. Lekin **telefon oynasi va panjara baribir kerak**: sizning
  dizayneringiz ham 380 px da nima chiqishini ko'rmasa, mijoz shikoyat qiladi.

### ⚠️ Va shundan bitta to'qnashuv chiqadi

Hozir ega o'z panelidan aksent rangini, radiusni, shrift juftligini va fon
ohangini o'zgartira oladi (`/admin/settings` → "Sayt dizayni"). Siz pul evaziga
maket chizganingizdan **keyin** ega o'sha tugmalarga tegsa — chizilgan dizaynni
buzadi, va qo'ng'iroq sizga keladi.

Shuning uchun: **konsol chizgan dizayn faol bo'lganda tenantdagi dizayn
muharriri qulflanadi** va o'rniga bir qator yoziladi ("dizayn Keel tomonidan
chizilgan — o'zgartirish uchun murojaat qiling").

Bu repoda allaqachon bor naqsh: `hideWatermark` ham **konsolda** turadi, chunki
"egasining sozlamalar sahifasida bo'lsa, ega uni shunchaki o'chirib
qo'yardi — kioskSecret va soldOut bilan bir xil tuzoq, faqat bu safar
narigi tomonida biznes modeli turadi".

---

## ⚠️ 3. Kontent tegilmaydi, lekin chegara qayerda

Talab aniq: **faqat dizayn, kontent emas.** Navbardagi so'zlar, tugma matnlari,
bo'lim sarlavhalari — kodda va lug'atda qoladi (ular uch tilli, va ega
tarjimani buzsa sayt uch tilda gapirishni to'xtatadi).

Shuning uchun blok **kontentni saqlamaydi, unga bog'lanadi**:

- `menu-grid` bloki — menyuni bazadan oladi; chizuvchi faqat **qaysi
  kategoriyalarni** va **qaysi ko'rinishda** ko'rsatishni tanlaydi
- `hours` bloki — ish vaqtini sozlamalardan oladi
- `gallery` bloki — yuklangan rasmlarni oladi
- `hero` bloki — restoran nomi, shiori va muqovasini oladi

Ya'ni konstruktor **joy va ko'rinishni** boshqaradi, **so'zlarni** — yo'q. Bu
chegara kodda majburlanadi: blokda erkin matn maydoni **yo'q**.

⚠️ Va bu qaror endi ikki tomonlama foyda beradi: kontent bloklarda saqlanmagani
uchun **bitta maketni ikkinchi mijozga qo'llash** shunchaki bloklar ro'yxatini
ko'chirish bo'ladi — matnni tozalash, boshqa restoranning nomini olib tashlash
kerak emas. Shablon galereyasi shu sababdan arzon chiqadi.

---

## 4. Bandlar

### A. Poydevor — blok tizimi (muharrirsiz)

Birinchi navbatda **maketni ma'lumot** qilish kerak, muharrir esa keyin. Aks
holda muharrir o'zi tug'diradigan formatga qurilib qoladi.

- **A1. `page_design` modeli — tenantning o'z bazasida, konsol yozadi.**
  `sections[] { type, variant, cols, style, binding }`, `status:
  draft|published`, `version`, `drawnBy` (konsol operatori). Brend darajasida
  (maket brendning yuzi, filialning emas — CLAUDE.md dagi taqsimot bilan bir
  xil). Yozuvchi — konsol, o'quvchi — tenant; `export_grant` bilan bir naqsh.
- **A2. Renderer.** Server komponenti: bo'limlar ro'yxatini mavjud
  komponentlarga aylantiradi. **Mobil-first**: panjara telefonda bitta ustun.
- **A3. Zaxira yo'l.** Dizayn yo'q = **bugungi maket, bayt-baytga**. Mavjud
  mijozlar hech nima sezmasligi kerak — bu butun xususiyatning sharti.
- **A4. Keshni yangilash.** ⚠️ Hozir sahifalar 30 soniya keshlanadi. Operator
  dizaynni chop etib jonli saytda o'zgarishni ko'rmasa, u chop etish
  ishlamadi deb o'ylab ikkinchi marta bosadi. Yechim: chop etilgandan keyin qisqa oyna davomida tenant
  javobga **`X-Accel-Expires: 0`** qo'yadi. Nginx `Cache-Control` ni
  e'tiborga olmaydi, lekin `X-Accel-Expires` ni **hurmat qiladi** — ya'ni
  mexanizm allaqachon joyida, faqat `proxy_ignore_headers` ro'yxatiga
  qo'shilmasligi kerak.
- **A5. Xavfsizlik.** Dizayn JSON'i CSS'ga aylanadi, ya'ni bu **injection
  yuzasi**. Ranglar, o'lchamlar, shriftlar — **allowlist**; hech qanday erkin
  satr `<style>` ichiga interpolyatsiya qilinmaydi. Testda muhrlanadi.

### B. Konstruktor — **konsolda** (canvas)

- **B1. Canvas.** 12 ustunli panjara, blok palitrasi, sudrab tartiblash, ustun
  kengligini o'zgartirish. Piksel yo'q. Mijoz kartochkasidan ochiladi, ya'ni
  operator qaysi restoranni chizayotganini adashtirmaydi.
- **B2. Ikki oyna.** Yonida telefon ramkasi — **mini app ko'rinishi**. Sizning
  dizayneringiz desktopni chizayotganda telefonni ham ko'radi.
- **B3. Shablonlar galereyasi.** "Shablon sifatida saqlash" va "mijozga
  qo'llash". ⚠️ Qo'llash **nusxa oladi, havola qilmaydi**: shablonni keyin
  tahrirlash o'n mijozning jonli saytini o'zgartirib qo'ymasligi kerak.
- **B4. Qoralama va chop etish.** Yarim chizilgan dizayn jonli saytga
  chiqmasligi kerak. Chop etilganda kesh chetlab o'tiladi (A4).
- **B5. Tenantdagi muharrirni qulflash.** Konsol dizayni faol bo'lsa
  `/admin/settings` dagi "Sayt dizayni" bloki o'rniga tushuntirish qatori
  chiqadi. Aks holda ega pul to'lagan maketni o'zi buzadi.
- **B6. Orqaga qaytish.** "Shablonga qaytarish" — mijoz dizayndan voz kechsa
  yoki chizilgani yoqmasa, standart maketga bir bosishda qaytish.

### C. Telegram mini app

- **C1. Har restoranga o'z boti va o'z tokeni** (qaror qabul qilindi). Token
  **egasining panelida** kiritiladi — bu uning boti, uning shartnomasi, xuddi
  SMS shlyuzi va to'lov kalitlari kabi — va brauzerga **hech qachon
  qaytarilmaydi** — to'lov kalitlari va SMS paroli bilan bir xil naqsh
  (`payment_settings` / `sms_settings` alohida kolleksiya, chunki `restaurant`
  hujjati har tashrifchiga to'liq boradi). Mini app egasining o'z boti ostida
  bo'lishi — bu uning brendi.
- **C2. `initData` tekshiruvi → bizning JWT.** Telegram ichida **SMS kerak
  emas**: `initData` bot tokeni bilan HMAC-SHA256 qilib tekshiriladi va biz o'z
  tokenimizni beramiz. ⚠️ Tekshiruv **serverda**, va imzosiz `initData` — hujum,
  xato emas. Bu SMS xarajatini ham, kirish to'siqini ham olib tashlaydi.
- **C3. Telefon raqami.** Telegram **raqamni bermaydi** (faqat id va ism).
  Buyurtmaga esa raqam kerak. Yechim: bot orqali `requestContact` — Telegram
  o'zi tasdiqlagan raqamni beradi, ya'ni SMS'dan **ishonchliroq**. Telegramdan
  tashqarida esa hozirgi SMS oqimi qoladi.
- **C4. WebApp muhiti.** `viewportStableHeight` (⚠️ `100vh` **ishlatilmaydi** —
  Telegram'da u klaviatura ostida qoladi), safe-area, `BackButton`,
  `MainButton` savat/checkout CTA sifatida, haptika, savat bo'sh bo'lmasa
  yopishdan oldin tasdiq.
- **C5. To'lov**: hozirgi to'rt provayder yetadi (qaror qabul qilindi), Telegram
  Payments qo'shilmaydi.
- **C6. Buyurtma holati — bot orqali.** ⚠️ Bu **pul tejaydigan band**: hozir
  har xabar SMS (pullik), Telegram esa bepul. Mini app'dan buyurtma bergan
  mijozga holat o'zgarishi botdan keladi.
- **C7. Deep link.** `t.me/<bot>/app?startapp=menu` va stol QR'i uchun
  `startapp=table_<id>` — stoldagi QR to'g'ridan-to'g'ri mini app'ni ochadi
  (hozir brauzerni ochadi).
- **C8. Uch til — mini app'ning birinchi ekrani.** ⚠️ Saytda tilni **URL va
  cookie** tashiydi, mini app'da esa ikkalasi ham yo'q: bot birinchi marta
  ochilganda cookie yo'q, manzil satri ham yo'q. Ya'ni ekran bo'lmasa ilova
  hamma uchun o'zbekcha ochiladi va almashtirgichni mijoz **o'qiy olmaydigan
  menyuni o'qib turib** izlashi kerak bo'ladi.
  - ⚠️ **Telegramning o'z tili o'rnini bosmaydi**: u telefon haqidagi taxmin,
    odam haqidagi javob emas — va aynan sezadigan mijozlarda xato bo'ladi
    (inglizcha telefondagi o'zbek; Telegramini o'g'li sozlab bergan rus).
    Shuning uchun u faqat **zaxira** sifatida ishlatiladi (`notifyLang`).
  - ⚠️ **Ekranda tarjima qilingan matn yo'q, va bu ataylab.** Bitta tilda
    "Tilni tanlang" deb yozilgan sarlavha — ilovaning bitta tilda ochilishi
    bilan bir xil xato. Shuning uchun variantlar **o'z tilida** yozilgan
    (`LANG_LABEL`) va sarlavha o'rnida globus turadi.
  - **Tanlov hisobda saqlanadi** (`user.lang`, `PUT /users/me/lang`), cookie'da
    emas: eng kerak bo'ladigan joy — soatlar keyin fon goroutine'idan
    yuborilgan **bot xabari**, va uning o'qiydigan brauzeri yo'q.
    Ustunlik: `user.lang` (tanladi) → `telegramLang` (taxmin) → `uz`.
  - **Sayt sarlavhasidagi almashtirgich ham hisobga yozadi** (kirgan mijoz
    uchun). Aks holda ruschaga o'tgan mijoz botdan o'zbekcha xabar olishda
    davom etardi — aynan shu ekran oldini olgan nomuvofiqlik, faqat boshqa
    eshikdan qaytib kelgan holda.

### D. Mobil sifat — o'lchov bilan, taxmin bilan emas

- **D1. Haqiqiy kengliklarda audit.** 360 / 380 / 414 px — Telegram
  WebView'ning ishchi kengligi. Playwright keshda bor.
- **D2. Vazn byudjeti.** Mini app 3G'da ochiladi. Rasm quvuri allaqachon bor
  (`?w=`), ya'ni asosiy ish qilingan; qolgani — mini app uchun kirish
  sahifasining vaznini o'lchab, chegara qo'yish.
- **D3. Telegram cheklovlari**: hover'ga tayangan hech nima, `44px` dan kichik
  bosish nishoni, va `position: fixed` pastki panel Telegram'ning o'z paneli
  ostida qolmasligi.

---

## 5. Nima birinchi va nega

Tartib texnik bog'liqlikdan chiqadi, xohishdan emas:

1. **A (poydevor)** — mini app ham, konstruktor ham shu maketni ko'rsatadi.
   Blok tizimi telefonni **birinchi** nishon qilib qurilsa, C4 deyarli tekin
   bo'ladi. Teskarisi: desktop uchun qurilgan maketni keyin telefonga
   moslashtirish — butun ishni qaytadan qilish.
2. **C1–C3 (bot, kirish, raqam)** — konstruktordan **mustaqil** va darhol
   foyda beradi: SMS'siz kirish va bepul bildirishnomalar. Bu A bilan
   parallel ketishi mumkin.
3. **B (canvas)** — u A ning ustiga quriladi va A o'zgarsa qayta yozilishi
   kerak, shuning uchun keyin. ⚠️ Lekin u **pul keltiradigan** band: A
   tugagandan keyin uni kechiktirmaslik kerak, chunki A yolg'iz o'zi hech
   qanday daromad bermaydi — u faqat imkoniyat.
4. **D** — har bosqichdan keyin, alohida band emas.

---

## 5.1 Holat (8-avgust, kechqurun)

**A tugadi va jonli tekshirildi.**
- A1 `page_design` modeli + `Sanitize` (test bilan) — `cc31e7d`
- A2/A3 renderer va bloklar; bosh sahifa 334 → 55 qator — `4a48ec3`
- Tekshirildi: matn **aynan bir xil** (2244 belgi, yagona farq — "ochiq/yopiq",
  ya'ni ish vaqti), desktop maketi bir xil (perks hero ustiga chiqishi
  saqlangan), telefonda **360/390/414 px da gorizontal oqish yo'q** va panjara
  bitta ustunga yig'iladi (Playwright bilan).
- ⚠️ A4 (`X-Accel-Expires`) **ataylab B ga surildi**: chop etish tugmasi hali
  yo'q, ya'ni keshni chetlab o'tadigan narsa ham yo'q. B4 bilan birga qilinadi.

**C1–C3 backend tomoni yozildi:**
- `internal/telegram` — `Verify` (HMAC-SHA256, `WebAppData` kaliti, sorted
  `k=v`, `subtle.ConstantTimeCompare`), `CheckFresh` (24 soat), `ParseUser`,
  `ParseContact`, `GetMe`, `SendMessage`, va testlar uchun `Sign`.
  Testlar: haqiqiy imzo o'tadi; **id o'zgartirilgan payload rad etiladi**;
  **boshqa botning imzosi rad etiladi**; imzosiz rad etiladi; tokensiz install
  hech nimani qabul qilmaydi; eski va kelajakdagi sana rad etiladi; tartib
  aralashganda ham 20 urinishda ishlaydi (Go map tartibi tasodifiy — bu test
  "sorted" ni muhrlaydi).
- `models/telegram.go` — `TelegramSettings` **alohida kolleksiyada** (to'lov
  kalitlari va SMS paroli bilan bir sabab: `restaurant` hujjati har
  tashrifchiga to'liq boradi). `user.telegramId` + `telegramUsername`.
- `handlers/telegram.go` — sozlamalar (owner, token qaytarilmaydi, bo'sh token
  = saqlangani qoladi, `ping` bot nomini o'zi to'ldiradi), `POST /auth/telegram`
  (imzo → bizning JWT, `needsPhone` bilan), `POST /users/me/telegram/phone`.
- Routerga ulandi, `go build` va barcha testlar yashil.

**Ertaga birinchi ish — shu yerdan davom:**
1. **Frontend: `TelegramEditor`** (`/admin/settings` da, `SmsEditor` naqshi
   bo'yicha) + `lib/api.ts` va uch tilli lug'at qatorlari. Backend tayyor,
   panelda hali hech nima yo'q.
2. **Haqiqiy bot bilan tekshirish** (@BotFather'da bot, token panelga, `ping`).
   ⚠️ `ParseContact` maydon nomlari **jonli bot bilan tekshirilmagan** —
   imzo sxemasi hujjatlashtirilgan va testda, lekin `requestContact` javobining
   shakli ikki ko'rinishda bardoshli o'qiladi va aynan qaysi biri kelishini
   birinchi haqiqiy mijozda ko'rish kerak.
3. Keyin **C4** (mini app muhiti) yoki **B** (konsoldagi canvas) — tanlov
   sizniki.

---

## 5.2 Holat (9-avgust)

**A** — tugagan va jonli tekshirilgan (yuqoriga qarang).

**C1–C8 — tugadi.**
- C1 panel + backend, C2 imzo bilan kirish, C3 Telegram tasdiqlagan raqam,
  C4 mini app muhiti (`--tg-viewport`, orqaga tugmasi, yopish tasdig'i, SDK faqat
  Telegram ichida yuklanadi), C5 to'lov — hozirgi provayderlar (qo'shimcha kod
  kerak emas), C6 buyurtma holati bot orqali, C7 stol QR'i mini app'ni ochadi,
  C8 uch til (birinchi ekran til tanlash, tanlov hisobda, bot shu tilda yozadi).
- ⚠️ Qolgan yagona tekshirilmagan joy: `requestContact` javobining **aynan
  shakli**. Imzo sxemasi hujjatlashtirilgan va testda; javob esa ikki
  ko'rinishda bardoshli o'qiladi (`lib/telegram.tsx` → `contactPayload`,
  `internal/telegram.ParseContact`). Birinchi haqiqiy botda ko'rish kerak.

**B — birinchi ishlaydigan versiya tayyor.**
- Backend (`control/internal/handlers/design.go`): qoralamani o'qish/saqlash,
  chop etish, shablonga qaytarish, shablon galereyasi (saqlash/qo'llash/o'chirish).
- ⚠️ **Chop etish `brandId` ni ham yozadi**, chunki tenant dizaynni **brend
  bo'yicha** o'qiydi. Busiz hujjat yozilardi-yu sayt uni hech qachon topmasdi —
  va nosozlik **jimgina** bo'lardi: sayt shablonga qaytadi va avvalgidek
  ko'rinadi.
- Konsol UI (`keel-site/src/components/DesignEditor.tsx`): band qo'shish,
  tartiblash, kenglik (ustunlarda), ko'rinish varianti, fon ohangi, bo'shliq,
  yashirish, va **ikki oynali sxematik ko'rinish** (kompyuter + telefon).
  Ko'rinish ataylab sxematik: chop etishdan oldin baholanadigan narsa —
  **struktura** (qaysi bandlar, qanday tartibda, qanchalik keng, telefonda nima
  bo'ladi), va pikselli ko'rinish bu savolga jonli saytdan yaxshi javob bermaydi.
- **B5 bajarildi**: konsol dizayni chop etilgan bo'lsa tenantdagi "Sayt dizayni"
  bloki **qulflanadi va sababini aytadi**. Buning uchun `GET /restaurant`
  `?raw=1` da ham `designLocked` qaytaradi — sozlamalar sahifasi aynan `raw`
  bilan so'raydi.
- **A4 yopildi, lekin boshqacha**: `X-Accel-Expires` qo'shilmadi. Chop etish
  javobida operatorga **30 soniyalik kesh** haqida yozib beriladi, konstruktorning
  ko'rinishi esa keshdan umuman o'tmaydi (u sxematik). Ya'ni "chop etdim,
  ko'rinmayapti" holati tushuntirilgan va har so'rovga qo'shimcha fetch
  qo'shilmadi.

**Qolgan ish (B ning ikkinchi bosqichi):** — ✅ **yopildi (11-avgust)**,
tafsiloti `PROGRESS.md` da.
1. ~~Bandni sudrab tartiblash~~ — allaqachon bor edi (`design/page.tsx`, HTML5
   drag; ↑/↓ yonida qoldirilgan, chunki faqat sudrab tartiblanadigan ro'yxatni
   klaviatura bilan tartiblab bo'lmaydi).
2. ~~`menu-grid` uchun kategoriya tanlash~~ — bor. Lekin yo'lda **ikkita
   jimgina nosozlik** topildi: band sozlamalari umuman saqlanmayotgan edi
   (control plane struct'i `settings` ni tashlab yuborardi), va `menu-grid`
   ularni o'qimasdi.
3. ~~`about`/`gallery`/`cta` ni haqiqiy dizaynda sinash~~ — sinaldi va
   **uchtasi ham hech nima chizmasdi** (balandligi 0). Zaxira zanjiri
   tiklandi.
4. **D** — 360/390/414 va Telegram o'lchamidagi viewport'da o'lchandi
   (gorizontal oqish yo'q). ⚠️ **Haqiqiy telefonda va haqiqiy Telegram
   WebView'ida hali sinalmagan** — buni faqat qo'lda qilish mumkin.

---

## 6. Qabul qilingan qarorlar (reja yopildi)

- **Canvas** — 12 ustunli panjara, avtomatik mobil yig'ilish bilan.
- **Bot** — har restoranga o'z boti va o'z tokeni; token egasining panelida.
- **Mini app'da to'lov** — hozirgi to'rt provayder yetadi.
- **Dizaynni Keel chizadi**, konsol orqali; ega o'zi chiza olmaydi.
- **Qulflangan blok ko'rinib turadi va sababini aytadi** (yashirilmaydi):
  yo'qolgan tugma "buzilgan" bo'lib ko'rinadi va qo'ng'iroq baribir keladi,
  faqat bu safar javob berish qiyinroq bo'ladi.
- **Dizayn puli tizimga kirmaydi.** U ish tashqarisidagi alohida kelishuv,
  ya'ni hisob-faktura tizimiga hech nima qo'shilmaydi. ⚠️ Lekin bir narsa
  qoladi: "bu mijozning dizayni chizilganmi?" degan savolga javob kerak, va
  javob **`page_design` hujjatining o'zi** bo'ladi (`drawnBy`, `publishedAt`).
  B5 dagi qulf ham shundan o'qiydi. Ya'ni alohida bayroq ham, alohida hisob ham
  kerak emas — mavjudlikning o'zi yozuv.

### Birinchi versiyadagi bloklar

Minimal ishlaydigan to'plam — bugungi sahifada allaqachon bor narsalar, faqat
endi ko'chirilishi va almashtirilishi mumkin bo'lgan holda:

| Blok | Ma'lumotni qayerdan oladi | Variantlar |
|---|---|---|
| `hero` | restoran nomi, shior, muqova | to'liq kenglik / yarim / matn chapda |
| `menu-grid` | menyu (tanlangan kategoriyalar) | 2 / 3 / 4 ustun, kartochka balandligi |
| `about` | `content.aboutTitle` + `aboutText` | matn / matn + rasm |
| `hours-address` | filial ish vaqti, manzil, telefon | xarita bilan / xaritasiz |
| `gallery` | yuklangan rasmlar | panjara / lenta |
| `cta` | buyurtma va bron havolalari | keng banner / ikki tugma |
| `footer` | aloqa, ijtimoiy tarmoqlar, ish vaqti | bir qator / uch ustun |

⚠️ **Ro'yxat ataylab qisqa.** Qolgan bloklarni **birinchi haqiqiy dizayndan
keyin** qo'shish kerak: nima kerakligini chizib ko'rgandan keyin bilib olasiz,
oldin esa taxmin qilasiz — va taxmin qilingan blok hech kim ishlatmaydigan
palitrani to'ldiradi.


## 6. Ikkinchi bosqich: erkin canvas va Shopify uslubidagi tahrirlagich

Birinchi versiya **panjara** edi: bandlar, har biri 12 ustunning bir qismi,
ichidagi joylashuv qat'iy. Haqiqiy brief esa Pinterest'dan olingan skrinshot —
navbar boshqacha, footer boshqacha, popup bor, fon shaffofligi bor. Shuning
uchun ikki narsa qo'shildi.

### Erkin blok (`canvas`)

- **Foizda, hech qachon pikselda.** `x=24%` har ekranda aynan bitta joyga
  tushadi; `x=340px` 380 px'li telefonda hech qanday joyga tushmaydi. Band
  balandligi ham `vh` da — shu sabab.
- ⚠️ **Telefon joylashuvi alohida chiziladi.** Desktopdagi kompozitsiyani
  telefonga aylantiradigan arifmetika **yo'q**: yonma-yon turgan elementlar bir
  ustunga tushishi kerak, va qanday tartibda — buni faqat odam biladi. Wix ham,
  Figma'dan sayt yasaydigan har bir tizim ham shu yerga keladi.
  Hech bir element telefon joylashuviga ega bo'lmasa — band **oqim**ga tushadi:
  chizilgan tartibda, to'liq kenglikda, ustma-ust tushmasdan. Bu kelishuv emas,
  aynan shu narsa xususiyatning "chiroyli, lekin telefonda ishlamaydi" bo'lib
  chiqishini to'xtatadi.
- ⚠️ **Ishlaydigan vidjetlar o'z ichini saqlaydi.** Canvas'ga qo'yilgan menyu
  erkin joylashtiriladi va baribir haqiqiy menyuni chizadi — narx, tarjima,
  variantlar va ishlaydigan savat bilan. Dizaynerni menyuni matn va
  to'rtburchaklardan qayta yasashga majburlaydigan tahrirlagich buyurtma qabul
  qila olmaydigan chiroyli sahifa ishlab chiqaradi.
- **Popup** ham canvas, faqat sahifa ustida: bir tashrifga bir marta
  (`sessionStorage`), fon bosilsa yopiladi, Escape yopadi, va yopish tugmasi
  haqiqiy hit-area bilan — telefonda u ekrandagi eng kichik nishon.

### Umumiy CSS — yagona erkin maydon

Qolgan hamma qiymat enum, chunki bu hujjat ommaviy sahifadagi stilga aylanadi.
Lekin rasmdan olingan dizaynda **doim** hech bir enum kutmagan bitta detal
bo'ladi, va escape hatch bo'lmasa u har mijoz uchun kod o'zgarishiga aylanadi —
konstruktor mavjudligining teskarisi.

`sanitizeCSS` **shakl bo'yicha whitelist**, parser emas: `</style`, har qanday
`<`, `javascript:`, `expression(`, `@import` va o'zimizning `/uploads/` dan
boshqa `url(` — hammasi rad etiladi, va rad etilganda **butun matn** bo'shatiladi
(yarim olib tashlangan qoida — hech kim yozmagan stil). Testda muhrlangan.
Maydonni **faqat konsol** yozadi: restoran egasi unga umuman yeta olmaydi.

### Jonli ko'rinish (iframe) — kalit bilan

⚠️ Sxematik ko'rinish "mijoz yuborgan rasmga o'xshadimi" degan savolga javob
bera olmaydi, va butun ish shu. Shuning uchun tahrirlagichning o'ng yarmi —
mijozning **haqiqiy sayti**, chop etilmagan qoralama qo'llangan holda.

Xavfni chegaralaydigan narsalar: kalit **tenantning o'z bazasida** (eksport
ruxsati bilan bir naqsh — bitta yozuvchi, bitta o'quvchi, tenant konteynerdan
konsolga yangi yo'l ochilmaydi), **2 soat** (qoralama chatga tashlanadigan
havola emas), tasodifiy va uzun, bitta brendga tegishli, va **hech qanday huquq
tashimaydi** — u sessiya emas va bo'la olmaydi. Muddat kodda ham tekshiriladi:
Mongo'ning TTL tozalashi daqiqada bir ishlaydi, ya'ni "hujjat yo'q" bilan
"kalit haqiqiy" bir xil fakt emas.

### Alohida sahifa

`/console/tenants/{id}/design`. Kartochka ichidagi panel qoldi — "bu mijozda
hozir nima bor" degan savol hisob-fakturalarga qarab turib beriladi — lekin
tahrirlagichning o'zi to'liq ekran: chapda bandlar va elementlar, o'ngda sayt,
tepasida kompyuter/telefon almashtirgichi.

**Qolgan ish:** elementni sichqoncha bilan sudrash (hozir X/Y/W/H raqamlari),
iframe ichidan element tanlash, temani shu sahifada tahrirlash (backend
`page_design.theme` ni allaqachon qabul qiladi).

### Tahrirlagichning o'zi: uch panel, sudrab boshqarish

Birinchi versiyada element **X/Y/W/H raqamlarini yozib** joylashtirilardi. Bu
tahrirlagich emas, sahifani tasvirlaydigan forma: dizayner eng ko'p qiladigan ish
("ko'rinishi to'g'ri bo'lguncha surish") eng sekin amalga aylanadi.

Endi: chapda bandlar va elementlar, markazda **chizma canvas** (sudrash, sakkiz
nuqtadan o'lchash, strelkalar bilan surish, Shift — 5%), o'ngdagi inspektor esa
o'sha raqamlarni saqlab qoldi — oxirgi 1% ko'pincha klaviaturada qilinadi.

- ⚠️ **Sudrash jonli iframe ichida emas.** Haqiqiy sahifada sudrash tahrirlagich
  kodini **har bir tenantning prod bundle'iga** yuborishni va origin chegarasidan
  o'tishni talab qilardi. Shuning uchun manipulyatsiya konsoldagi sodda, lekin
  aynan **o'sha foizli box**larni chizadigan sirtda bo'ladi, jonli sayt esa
  yonida — almashtirgich bilan. Ular ikki boshqa savolga javob beradi: canvas —
  "element qayerda", sayt — "mijoz yuborgan rasmga o'xshadimi". Bittasi
  ikkinchisining savoliga javob bera olmaydi.
- **Tekislash chiziqlari va 1% ga qadalish**: 0/50/100 va boshqa elementlarning
  chetlari/markazlari bo'yicha. Busiz "markazlashtirilgan" sarlavha sichqoncha
  qo'yib yuborilgan joyga markazlashadi va sahifa sababi ko'rsatib bo'lmaydigan
  darajada tartibsiz ko'rinadi.
- ⚠️ **Undo — zarurat, qulaylik emas.** Bu yerdagi ish uslubi "sudrab ko'raman",
  va qaytarib bo'lmaydigan sudrash urinishni qimmat qiladi. Tarix butun band
  ro'yxatini saqlaydi (kichik), va **bir sudrash = bir yozuv** (pikselga bitta
  emas — aks holda undo stack'idan chiqib bo'lmaydi).
- **Telefon rejimida chizilmagan element punktir bilan** ko'rsatiladi va
  sudralmaydi: saytda u oqimga tushadi, va u yerda sudrash egaga so'ralmagan
  telefon joylashuvini jimgina yaratardi.
- **Konsolning `max-width` i bu sahifada olib tashlangan** (`fullBleed`): qolgan
  har bir konsol sahifasi hujjat, bu esa asbob — markaziy panelning butun vazifasi
  ekran bergancha keng bo'lish.

### Jonli saytda tahrirlash (iframe ustidagi qatlam)

Sudrash **jonli sahifada** bo'ladi, lekin tahrirlagich kodi tenant saytiga
ketmaydi. Ish ikkiga bo'lingan:

- **Sayt faqat geometriyani xabar qiladi** (`components/design/PreviewBridge.tsx`):
  har box uchun to'rt son, hech qanday identifikator, token yoki mijoz ma'lumoti
  yo'q. U **faqat preview tokeni bilan** render qilinganda ulanadi — mehmonning
  sahifasida umuman yo'q — va hech nima yozmaydi: eng yomon holatda tutqichlar
  noto'g'ri joyda turadi.
- **Konsol qaror qiladi** (`components/design/PreviewOverlay.tsx`): tutqichlarni
  chizadi, piksel siljishini **bandning o'z rect'i** bo'yicha foizga aylantiradi
  (40 px 1280 px'li bandda va 390 px'lida boshqa narsa) va tahrirlagichning
  mavjud holatiga yozadi — ya'ni yagona yozuvchi o'zgarmaydi.
- ⚠️ **Xabarning kelib chiqishi tekshiriladi** (`e.origin`): oyna istalgan
  saytdan xabar olishi mumkin. Bu xabar dizaynni o'zgartira olmaydi, lekin
  "bugun zarar qila olmaydi" — tahrirlardan omon qolmaydigan xususiyat.
- **Saqlash sudrash tugaganda** (`pointerup`), harakat paytida emas: har piksel
  uchun saqlash — birovning jonli saytida har piksel uchun sahifa render qilish.
  Sudralayotgan box — konsolning o'z div'i, shuning uchun kechikish sezilmaydi.
- Token **qayta ishlatiladi**, har sudrashda yangisi yaratilmaydi: aks holda har
  biri ikki soat yashaydigan jonli havolalar izi qolardi.
- Almashtirgich bor ("Tahrirlash" / "Faqat ko'rish"): bu panel shunchaki qarash
  uchun ham ishlatiladi, va o'qiyotgan sahifa ustidagi ko'rinmas sudrash
  nishonlari — elementning tasodifan ko'chishining yo'li.
