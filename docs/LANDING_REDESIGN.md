# Landing (keel.uz) dizaynini yangilash — reja

**Holat:** rejalashtirilgan, boshlanmagan. Limit ochilganda shu fayldan
davom etiladi.
**Sana:** 2026-08-27 (ko'rib chiqildi va yozildi)
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

Kerak bo'ladigan kadrlar:
- Kassa ekrani (chek ochiq, taomlar to'ri)
- Zal xaritasi (planshet ko'rinishi)
- Oshxona ekrani (KDS)
- Admin panel: buyurtmalar oqimi, dashboard statistikasi
- Ombor / tannarx ekrani
- Telegram bot va mini app (telefon ramkasida)
- Kuryer PWA (telefon ramkasida)
- Restoran sayti (brauzer ramkasida)

Har biri **light va dark** temada kerak bo'lishi mumkin — sahifa ikkala
temada ishlaydi, va bitta oq screenshot to'q sahifada teshik bo'lib ko'rinadi.
Muqobili: screenshot'ni neytral qurilma ramkasi ichiga solish (ramka temaga
moslashadi, ichidagi rasm o'zgarmaydi) — avval shuni sinash arzonroq.

⚠️ **Monoblok / printer / pul yashigining studiya fotosi bizda yo'q** va uni
men yasay olmayman. Ikki yo'l: (a) apparatni suratga olish kerak, (b) apparat
bo'limi izometrik chizmada qoladi. Qaror qabul qilinmagan.

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

## 6. Ochiq savollar

- Apparat fotosi bo'ladimi (§4 oxiri)?
- Kalkulyator qaysi raqamlarni ko'rsatadi — kassa tarifi + buyurtma pog'onasi
  birgami, yoki faqat kassa? (Hozirgi sahifada ikkalasi ikki alohida bo'lim.)
- Izometrik `Visual3D.tsx` butunlay olib tashlanadimi yoki apparat/abstrakt
  joylarda qoladimi? (824 qator — saqlansa ikki vizual til yonma-yon turadi.)
