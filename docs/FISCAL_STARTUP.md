# Fiskal servis (virtual kassa) — startup yo'l xaritasi

Keel'dan alohida (yoki uning gorizontal kengaytmasi sifatida) **virtual kassa /
fiskal servis** yo'nalishini ochish bo'yicha reja. Asos: bozor va regulyativ
tadqiqot (2026-09-03).

> ⚠️ **Halol cheklov.** Kalit hujjat — **Vazirlar Mahkamasi qarori №943
> (23.11.2019)** — 2019-yilgi. 2026-ga qadar redaksiya, bojlar, muddatlar
> o'zgargan bo'lishi mumkin. Bu hujjatdagi **tuzilma** (OFD → xulosalar →
> reestr + ЦОТУ) barqaror, lekin **aniq raqam va prosedurani soliq.uz'дан yoki
> fiskal yuristdan tasdiqlash shart** (§7 "Aniqlanishi kerak").

---

## 1. Bozor qanday tuzilgan

Zanjir:
```
Sotuv (POS) → fiskal modul CHEKNI IMZOLAYDI (EDS) → OFD → Soliq qo'mitasi
    → fiskal belgi (ФП) + QR qaytadi → mijoz Soliq ilovasida tekshiradi
```

- **OFD (Fiskal ma'lumotlar operatori):** markaziy — **НИЦ «Yangi texnologiyalar»**.
  Har bir chek shu orqali soliqqa boradi.
- **Virtual kassa** = jismoniy quti yo'q; fiskal modul dastur/bulutда, korxona
  INN'iga bog'langan. Server chekni imzolaydi va uzatadi.
- **Bozordagi o'yinchilar:** era, RahmatPOS, REGOS, Multikassa/Multibank,
  UZKASSA, e-POS, QPOS. Ular POS dasturi + virtual kassa + hisobot + markirovka
  + IKPU/MXIK katalogini sotadi.
- **Narx orientiri:** bazaviy tarif ~134 700 so'm/oy dan (Paloma365 misoli),
  kassa boshiga oylik SaaS modeli.

## 2. "Soliqdan sertifikat" aslida nima

Bitta qog'oz emas — **Davlat reestriga kirish**. Dasturing reestrga kirsa,
qonuniy virtual kassa sifatida sotila oladi. Asos: **PKM №943**.

## 3. Reestrga kirish jarayoni (4 qadam)

### 3.1. OFD bilan integratsiya
- **НИЦ «Yangi texnologiyalar»** bilan integratsiya shartnomasi.
- Dasturni ularning fiskal magistraliga texnik ulash (test muhitидан
  boshlanadi).

### 3.2. Ikkita xulosa
- **OFD xulosasi** — integratsiya muvaffaqiyatли.
- **Kiberxavfsizlik markazi** (ГУП «Центр кибербезопасности») xulosasi.

### 3.3. Soliq qo'mitasiga reestrга kiritish arizasi — 10 hujjat
1. Dasturning etalon nusxasi
2. Muvofiqlik sertifikati
3. Reestrga kiritish arizasi
4. Chek namunasi
5. Texnik pasport
6. Ekspluatatsiya hujjatlari
7. Ehtiyot qismlar kafolat xati
8. Kiberxavfsizlik markazi xulosasi
9. OFD xulosasi
10. Patent / mualliflik huquqi

### 3.4. ЦОТУ — texnik xizmat markazi (PKM №943, 3 va 4-ilova)
- Ishlab chiquvchi bilan shartnoma
- Bino egaligi/foydalanish hujjatlari
- **Call-markaz** borligi haqida ma'lumotnoma
- Malakali mutaxassislar (kvalifikatsiya sertifikatli)

> **Namuna:** pospro.uz — ro'yxatdan o'tgan ЦОТУ, qanday ko'rinishini o'rganish
> uchun.

### ⭐ Bizga tegishli imtiyoz
> **O'z avtomatlashtirilgan hisob kompleksini** virtual kassa sifatida reestrга
> kiritayotgan ishlab chiquvchidan **depozit zaxira summasi talab qilinmaydi.**

Ya'ni Keel'ni o'z kompleksimiz sifatida ro'yxatga qo'ysak — depozit shart emas.
Bu kirish to'sig'ini sezilarli pasaytiradi va **eng tabiiy yo'limiz**: era/rahmat
ustiga chiqmasdan, o'zimiz reestrга kiramiz.

## 4. Texnik talablar (PKM №943, 2-ilova) + 2026 majburiy funksiyalar

Dastur 2-ilovadagi texnik talablarga mos bo'lishi shart. **2026-04-01 dan**
qo'shilgan majburiy funksiyalar (provayder qo'llab-quvvatlashi kerak):
- **tasnif.soliq.uz** davlat katalogi bo'yicha sotuvni cheklash (MXIK/IKPU).
- **Retsept dorilar** uchun QR skanerlash.
- **Muddati o'tgan** tovar sotuvини bloklash.
- **Ijtimoiy karta** to'lovini to'g'ri qayta ishlash.
- Elektron chek qog'oz chek bilan **teng huquqли** (talabда qog'oz beriladi).

## 5. Bizda allaqachon bor (Keel aktivlari)

Fiskal platformaning "og'ir" muhandislik qismi qo'limizда:
- **Fiskal agent** (`fiscalagent.exe`) — kassa kompyuteridан cheklarni haydaydi.
- **Chek shakllantirish** — `backend/internal/fiscal/receipt.go`.
- Provayder adapterlari: `multikassa.go`, `epos.go`, `regos.go` (`fiscal.go`).
- **IKPU/MXIK** normalizatsiyasi (`docs/DECISIONS.md` → ИКПУ).
- **Markirovka** (Asl Belgisi / DataMatrix) — `internal/marking`.
- Multi-tenant deploy, POS/KDS/ombor.

Ya'ni raqobatchilardan **oldinda** turibmiz — qolgani asosan regulyativ.

## 6. Bosqichli reja

### Bosqich 0 — hozir (Keel ichida)
- Fiskal agent + 2-3 real operator adapterini mustahkamlash.
- Restoranlar uchun "to'liq onlayn kassa" referensini yig'ish (demo + mijoz).

### Bosqich 1 — akkreditatsiyani o'rganish (kod emas, huquqiy)
- НИЦ «Yangi texnologiyalar» bilan uchrashuv: integratsiya shartnomasi va test
  muhiti shartlari.
- Fiskal yurist bilan PKM №943 aktual redaksiyasini ko'rib chiqish.
- ЦОТУ talablarini aniqlash (bino, call-markaz, mutaxassis — xarajat smetasi).

### Bosqich 2 — texnik moslik
- 2-ilova texnik talablari + 2026 funksiyalari (tasnif, retsept QR, muddat
  bloklash, ijtimoiy karta) bo'yicha bo'shliqlarni yopish.
- OFD test integratsiyasi → OFD xulosasi.
- Kiberxavfsizlik markazi auditi → xulosa.

### Bosqich 3 — reestrga kirish
- 10 hujjatni tayyorlash, ЦОТУ tashkil qilish, arizani topshirish.
- Reestrда paydo bo'lgach — gorizontal sotuv (restorandan tashqari: do'kon,
  xizmat).

### Bosqich 4 — gorizontal mahsulot
- Keel'ning fiskal qatlamini alohida brend/yo'nalish sifatida chiqarish.
- Narx: kassa boshiga oylik (bozor ~135k so'm/oy'дан).

## 7. Aniqlanishi kerak (tasdiqsiz yozib bo'lmaydigan raqamlar)

- PKM №943 ning **2026 aktual redaksiyasi** va o'zgarishlari.
- OFD integratsiya shartnomasining **narxi va muddati** (Yangi texnologiyalar).
- ЦОТУ **minimal talablari**: bino m², call-markaz o'lchami, mutaxassis soni va
  ularning kvalifikatsiya sertifikati qanday olinadi.
- Kiberxavfsizlik markazi **auditi narxi va muddati**.
- Muvofiqlik sertifikati (2-band) kim tomonidan beriladi.
- Depozit imtiyozining **aynan qaysi shart** bilan qo'llanishi.
- 2026-04-01 funksiyalarining texnik spetsifikatsiyasi (tasnif.soliq.uz API,
  retsept QR formati, ijtimoiy karta relslari).

## Manbalar
- BUXGALTER.UZ — dasturlarni soliq tizimlari bilan integratsiya qilish jarayoni
  va 10 hujjat: https://buxgalter.uz/publish/doc/text195312
- NORMA.UZ — fiskal modulni ro'yxatga olish:
  https://www.norma.uz/nashi_obzori/kak_zaregistrirovat_fiskalnyy_modul_i_primenyat_onlayn-kkm
- Soliq qo'mitasi — virtual kassalar reestri:
  https://soliq.uz/page/virtual-kassa-reestri
- Soliq qo'mitasi — onlayn-KKM xizmati:
  https://soliq.uz/services-facilities/online-kkm
- Paloma365 — 2026 talablari va narx orientiri:
  https://paloma365.uz/ru/blog/guides/onlajn-kassa-uzbekistan
- Asosiy hujjat: **Vazirlar Mahkamasi qarori №943 (23.11.2019)**, 1–4 ilovalar.
