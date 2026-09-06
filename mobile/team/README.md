# Keel Team — ishchilar ilovasi (Android + iOS)

> ⚠️ **Bu ilova almashtirildi: `mobile/team-android` (Kotlin + Compose).**
> Yangisi shu `applicationId` ni (`uz.keel.team`) oladi, ya'ni telefonda
> eskisining o'rniga o'rnatiladi. Bu papka **hozircha saqlanadi**: qarorlar va
> ular ortidagi xatolar tarixi shu yerda birinchi marta yozilgan, va iOS hali
> faqat shu yerda.

Expo (SDK 57, RN 0.86, React 19). Restorandagi **har bir xodim** uchun: smenani
ochish/yopish, o'z davomati va ish haqi, va o'ziga tegishli bildirishnomalar.

## Nega alohida ilova

Ofitsiantda zal bor, kuryerda yo'l bor. Oshpaz, barmen, farrosh va oshxona
yordamchisida esa tizim ulardan so'raydigan **bitta** narsa va ular tizimdan
so'raydigan bitta narsa bor: smena boshlandi, smena tugadi, va men qancha
ishladim. Qolgan har qanday ekran — boshqa birovniki.

⚠️ **Davomat yagona telefonsiz qism edi**: u veb sahifada (`/staff`), ya'ni
xodimga havola aytilishi, u brauzer tabida saqlanishi va har ertalab qaytadan
topilishi kerak edi. Ustiga kassa endi **ochiq smenasiz PIN ni rad etadi**
(`branch.requireShift`), ya'ni «sahifani topolmadim» degan gap «ishni boshlay
olmayapman» ga aylandi.

## Ekranlar

- **Smena** — bitta katta tugma (boshlash / yakunlash), ochiq smena, bugungi va
  oylik soatlar, davomat kalendari (grafik bo'yicha / ko'p / kam / chiqmagan).
  ⚠️ Sanoq **serverdan** keladi (`/staff/report`): telefon o'z soatlarini
  qo'shsa, u panel bilan ziddiyatga tushardi va bu **oylik kuni** topilardi.
- **Bozor** — ⚠️ **faqat `buy` ruxsati bor hisobda ko'rinadi.** Restoran uchun
  bozorlik qiladigan odam nima olib kelganini shu yerda yozadi, va u to'g'ridan
  to'g'ri omborga kirim bo'lib tushadi (`/staff/buy`). Ekran nima kam
  qolganidan boshlanadi — bu **panelning xarid ro'yxatining o'zi**
  (`shoppingList`), ikkinchi hisob emas: bozorda turgan odam bilan ofisdagi ega
  "go'sht tugadimi?" degan savolda kelishmovchilikka tushmasligi kerak, chunki
  bu bahs pul sarflangandan keyin bo'ladi.
  ⚠️ **Har narx maydonining yonida oxirgi narx turadi** — bu raqamning yagona
  qo'rig'i: telefonda yozilgan narx shu masalliqli har bir taomning tannarxini
  o'zgartiradi, va `9 000` o'rniga yozilgan `90 000` keyin oddiy raqamga
  o'xshaydi. Uni faqat peshtaxtada turgan odam ushlay oladi.
  ⚠️ **`clientId`** har yuborishda yaraladi: bozorda signal zaldagidan yomon, va
  usiz qayta yuborish ikkinchi kirim bo'lardi — javon ikki marta ko'tarilib,
  hisob ikki marta to'lanadi.
- **Sozlamalar** — til, ko'rinish, hisob, bildirishnomalar holati, chiqish.

⚠️ **Joylashuv smena tugmasi bosilganda so'raladi**, ochilishda emas: server
punchni filial radiusiga solishtiradi (`geofenceBlocked`), va ilova ochilishida
so'ralgan ruxsat — nima uchunligi ma'lum bo'lishidan oldin beriladigan savol.
Aniqligi `Balanced`: radius 50 m, GPS ochiq havoda 10–30 m, ya'ni eng yuqori
aniqlik javobni o'zgartirmaydi.

## Bildirishnomalar

Kanal `team` (ofitsiantning `kitchen` va kuryerning `delivery` kanalidan
alohida: Android'da kanalni foydalanuvchi o'chiradi). Voqealar:

| Voqea | Serverdagi joyi |
|---|---|
| Ish haqi yozildi (summa va davr) | `AdminPayStaff` |
| Smena tuzatildi (qaysi kun) | `AdminUpdateShift` |
| Grafik o'zgardi | `AdminUpdateStaff` (faqat grafik haqiqatan o'zgarsa) |
| Hisob o'chirildi | `AdminUpdateStaff` (`true → false`) |

- ⚠️ **Til token bilan birga yuboriladi** (`POST /staff/push` → `lang`), matnni
  server yozadi va `internal/i18n` katalogidan o'tkazadi. Shu o'zgarish bilan
  ofitsiantning «Tayyor» xabari ham uch tilli bo'ldi — ilgari u faqat
  o'zbekcha edi.
- ⚠️ **Faqat chekkada yuboriladi**: bir xil formani qayta saqlash hech nima
  yubormaydi. Uchinchi keraksiz xabar — ikkinchisini ham o'qimay surib
  tashlashga o'rgatadi.

## ⚠️ Firebase: `uz.keel.team` uchun alohida yozuv kerak

`google-services.json` ichida **har bir paket** uchun yozuv bo'lishi shart.
Hozirgi fayl `uz.keel.courier` va `uz.keel.waiter` ni biladi; Android buildi
`uz.keel.team` yozuvisiz **xato beradi** («No matching client found»), va bu
ataylab shovqinli: fayl unutilsa build o'tib ketardi va ilova jimgina token
ololmasdi.

1. Firebase konsoli → o'sha loyiha (`keel-7f31a`) → **Add app → Android** →
   paket `uz.keel.team`.
2. Yangi `google-services.json` ni yuklab olib, uchala ilovaning papkasiga ham
   qo'ying (fayl uchala paketni ham biladi).
3. FCM V1 kaliti EAS'da **allaqachon ulangan** (bitta service account uchala
   loyihaga ham).

## Ishga tushirish va build

```bash
npm start                                        # Metro
npx eas-cli build -p android --profile preview   # o'rnatiladigan APK
```

Android SDK shart emas: build EAS'da, bulutda quriladi.

## O'lchov asosi (1-sentabr 2026)

| O'lchov | Qiymat | Qanday olingan |
|---|---|---|
| JS bundle (Hermes) | **1.8 MB** | `npx expo export --platform android` |
| Modullar | 704 | o'sha |
| `expo-doctor` | 21/21 | `npx expo-doctor` |
| APK | hali o'lchanmagan | EAS build'dan keyin |

Waiter 750 / Courier 715 modul — bu ilova eng kichigi, va shunday bo'lib
qolishi kerak: unga qo'shiladigan har bir ekran boshqa birovning ishi.
