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

| Variant | Ega nima his qiladi | Mobil natija |
|---|---|---|
| **(a) Bo'limlar ro'yxati** — bo'lim tanlash, tartib, ko'rinish varianti | "sozlayapman" | mukammal |
| **(b) Erkin canvas + ikkinchi mobil maket** | "chizayapman" | ega chizsa yaxshi, chizmasa buzuq |
| **(c) Panjarali canvas (grid snap) + avtomatik mobil yig'ilish** | "chizayapman" | mukammal |

**Tavsiyam — (c).** 12 ustunli panjara, bloklar ustunlarga yopishadi, kenglik
ustun bilan o'lchanadi. Telefonda esa **o'sha tartibda bitta ustunga**
yig'iladi. Ega o'zini chizayotgandek his qiladi, chiqadigan narsa esa
responsive bo'lib qoladi — chunki maket pikselda emas, **ustunlarda** yozilgan.

Va konstruktorning yonida **telefon ramkasi** doim turadi: ega mini app'ni
o'sha zahoti ko'radi, "keyin tekshiraman" degan qadam yo'qoladi.

---

## ⚠️ 2. Ikkinchi qaror: kontent tegilmaydi, lekin chegara qayerda

Talab aniq: **faqat dizayn, kontent emas.** Navbardagi so'zlar, tugma matnlari,
bo'lim sarlavhalari — kodda va lug'atda qoladi (ular uch tilli, va ega
tarjimani buzsa sayt uch tilda gapirishni to'xtatadi).

Shuning uchun blok **kontentni saqlamaydi, unga bog'lanadi**:

- `menu-grid` bloki — menyuni bazadan oladi; ega faqat **qaysi
  kategoriyalarni** va **qaysi ko'rinishda** ko'rsatishni tanlaydi
- `hours` bloki — ish vaqtini sozlamalardan oladi
- `gallery` bloki — yuklangan rasmlarni oladi
- `hero` bloki — restoran nomi, shiori va muqovasini oladi

Ya'ni ega **joy va ko'rinishni** boshqaradi, **so'zlarni** — yo'q. Bu chegara
kodda majburlanadi: blokda erkin matn maydoni **yo'q**.

---

## 3. Bandlar

### A. Poydevor — blok tizimi (muharrirsiz)

Birinchi navbatda **maketni ma'lumot** qilish kerak, muharrir esa keyin. Aks
holda muharrir o'zi tug'diradigan formatga qurilib qoladi.

- **A1. `page_design` modeli.** `sections[] { type, variant, cols, style,
  binding }`, `status: draft|published`, `version`. Brend darajasida (maket
  brendning yuzi, filialning emas — CLAUDE.md dagi taqsimot bilan bir xil).
- **A2. Renderer.** Server komponenti: bo'limlar ro'yxatini mavjud
  komponentlarga aylantiradi. **Mobil-first**: panjara telefonda bitta ustun.
- **A3. Zaxira yo'l.** Dizayn yo'q = **bugungi maket, bayt-baytga**. Mavjud
  mijozlar hech nima sezmasligi kerak — bu butun xususiyatning sharti.
- **A4. Keshni yangilash.** ⚠️ Hozir sahifalar 30 soniya keshlanadi. Ega
  dizaynni chop etib, saytida o'zgarishni ko'rmasa — bu qo'llab-quvvatlash
  qo'ng'irog'i. Yechim: chop etilgandan keyin qisqa oyna davomida tenant
  javobga **`X-Accel-Expires: 0`** qo'yadi. Nginx `Cache-Control` ni
  e'tiborga olmaydi, lekin `X-Accel-Expires` ni **hurmat qiladi** — ya'ni
  mexanizm allaqachon joyida, faqat `proxy_ignore_headers` ro'yxatiga
  qo'shilmasligi kerak.
- **A5. Xavfsizlik.** Dizayn JSON'i CSS'ga aylanadi, ya'ni bu **injection
  yuzasi**. Ranglar, o'lchamlar, shriftlar — **allowlist**; hech qanday erkin
  satr `<style>` ichiga interpolyatsiya qilinmaydi. Testda muhrlanadi.

### B. Konstruktor (canvas)

- **B1. Canvas.** 12 ustunli panjara, blok palitrasi, sudrab tartiblash,
  ustun kengligini o'zgartirish. Piksel yo'q.
- **B2. Ikki oyna.** Yonida telefon ramkasi — **mini app ko'rinishi**. Ega
  desktopni chizayotganda telefonni ham ko'radi.
- **B3. Qoralama va chop etish.** Yarim chizilgan dizayn jonli saytga
  chiqmasligi kerak. "Shablonga qaytarish" tugmasi ham — ega o'zini
  qulflab qo'ymasligi uchun.
- **B4. Cheklovlar ekranda.** Blokni 3 ustundan kichik qilib bo'lmasligi,
  menyu bloki telefonda ikki ustunga o'tishi — bularni **ega chizayotganda**
  ko'rsatish kerak, keyin emas.

### C. Telegram mini app

- **C1. Har restoranga o'z boti.** Token sozlamalarda, brauzerga **hech qachon
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
- **C5. Buyurtma holati — bot orqali.** ⚠️ Bu **pul tejaydigan band**: hozir
  har xabar SMS (pullik), Telegram esa bepul. Mini app'dan buyurtma bergan
  mijozga holat o'zgarishi botdan keladi.
- **C6. Deep link.** `t.me/<bot>/app?startapp=menu` va stol QR'i uchun
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

## 4. Nima birinchi va nega

Tartib texnik bog'liqlikdan chiqadi, xohishdan emas:

1. **A (poydevor)** — mini app ham, konstruktor ham shu maketni ko'rsatadi.
   Blok tizimi telefonni **birinchi** nishon qilib qurilsa, C4 deyarli tekin
   bo'ladi. Teskarisi: desktop uchun qurilgan maketni keyin telefonga
   moslashtirish — butun ishni qaytadan qilish.
2. **C1–C3 (bot, kirish, raqam)** — konstruktordan **mustaqil** va darhol
   foyda beradi: SMS'siz kirish va bepul bildirishnomalar. Bu A bilan
   parallel ketishi mumkin.
3. **B (canvas)** — eng ko'rinadigan, lekin eng oxirgi: u A ning ustiga
   quriladi va A o'zgarsa qayta yozilishi kerak bo'ladi.
4. **D** — har bosqichdan keyin, alohida band emas.

---

## 5. Javob kutayotgan savollar

1. **Canvas erkinligi**: (c) panjarali — tavsiyam. Yoki ega haqiqatan pikselda
   joylashtirishni xohlaydimi (u holda telefon uchun ikkinchi maket
   majburiy bo'ladi)?
2. **Bot**: har restoran o'z botini yaratadimi (tavsiyam — SMS va to'lov bilan
   bir xil mantiq), yoki ulanmaganlar uchun platformaning umumiy boti ham
   bo'lsinmi?
3. **Mini app'da to'lov**: hozirgi provayderlar (Payme/Click/Uzum/ATMOS)
   yetarlimi, yoki Telegram Payments ham kerakmi? (Ikkinchisi alohida band.)
4. **Dizaynni kim chizadi**: ega o'zi, yoki siz mijoz uchun chizib berasizmi?
   Javob B3 ni o'zgartiradi (shablon galereyasi kerakmi).
