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
