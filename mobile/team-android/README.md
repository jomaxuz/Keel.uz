# Keel Team — native (Kotlin + Jetpack Compose)

Ishchilar ilovasining **native Android** qayta yozilishi. Eskisi —
`mobile/team` (Expo/RN 0.86) — o'chirilmagan; bu papka uni almashtiradi.
`applicationId` **bir xil** (`uz.keel.team`), ya'ni yangi build eskisining
ustiga o'rnatiladi. ⚠️ Ikkalasi boshqa kalit bilan imzolangani uchun birinchi
o'rnatishda eskisini **o'chirish** kerak.

Beshinchi va oxirgi native ilova: dizayn `mobile/android-design` dan (waiter,
owner, TV va kuryer bilan bitta modul).

---

## Nima ko'chirildi

| Eski fayl (`mobile/team`) | Nima qiladi | Native fayl |
|---|---|---|
| `App.tsx` | 5 holat, ruxsatga qarab 2–4 tab | `MainActivity.kt` |
| `src/session.ts` | Kim, qayerda, oflayn/chiqib ketgan farqi | `Session.kt` |
| `src/tokens.ts` | SecureStore adapteri | `design/TokenStore.kt` |
| `src/profile.tsx` | Soatlar, kalendar, legenda | `ui/screens/ProfileScreen.kt` |
| `src/clock.tsx` | Smena tugmasi + GPS punch | o'sha fayl (`ClockButton`) |
| `src/buy.tsx` (746 q.) | Bozorlik: hamyon, ro'yxat, kirim | `ui/screens/BuyScreen.kt` |
| `src/zakup.tsx` (476 q.) | Bozorlik ro'yxatini yozish | `ui/screens/ZakupScreen.kt` |
| — | Ikkalasining umumiy qismi | `ui/screens/MarketPieces.kt` |
| `src/settings.tsx` | Til, tema, push holati, ikki chiqish | `ui/screens/SettingsScreen.kt` |
| `src/auth.tsx` | Manzil + login | `ui/screens/AuthScreens.kt` |
| `src/offlinescreen.tsx` | Internetsiz ochilganda | `ui/screens/OfflineScreen.kt` |
| `src/push.ts` | Bildirishnoma (Expo relay → **FCM**) | `push/Push.kt` |
| `src/i18n.ts` (590 q.) | UZ/RU/EN | `i18n/Strings.kt` |

---

## ⚠️ Joylashuv: kuryernikidan boshqacha, ataylab

Kuryer ilovasida ruxsat **smena ochilishi bilan** so'raladi, chunki u yerda
ruxsat smenaning o'zi — butun smena davomida oqim ketadi.

Bu yerda esa **tugma bosilganda**: bitta punch uchun bitta nuqta kerak, va
ochilishda so'ralgan ruxsat — nima uchunligi ma'lum bo'lishidan oldin
beriladigan savol, javobi esa «yo'q». Bosilganda so'ralsa — sababi ko'rinib
turadi.

- `fine` va `coarse` bitta dialogda: Android 12 «taxminan» berishga ruxsat
  beradi, faqat `fine` so'ralsa u javob **rad** bo'lib keladi. Filial radiusi
  50 m — taxminiy nuqta ham ko'pincha ichiga tushadi.
- Aniqlik `Balanced`, eng yuqorisi emas: radius 50 m, GPS ochiq havoda
  10–30 m, ya'ni eng yuqori aniqlik javobni o'zgartirmaydi, sarflagan
  soniyalari esa eshik oldida turgan odamniki.
- Fon xizmati **yo'q** va kerak emas.

## ⚠️ Tab faqat ruxsati bor hisobda ko'rinadi

- **Bozor** — `buy` ruxsati.
- **Zakup** — `buyorder` **va** (`void` yoki `stock`): kassirda `buyorder` bor,
  lekin uning telefoni restoranning bozorligi rejalashtiriladigan joy emas.
  Qoida `canWriteHere` da va uning testi bor. ⚠️ Rol **nomi** bilan emas,
  ruxsat bilan so'raladi.
- Ruxsat tortib olinsa va odam o'sha tabda turgan bo'lsa — u «Smena» ga
  qaytariladi: server rad etadigan ekranda qoldirish tugmani ishlamaydigan
  qiladi.

## ⚠️ Hisob-kitob ilovada emas

`/staff/report` kunni, statusini va grafikka nisbatini o'zi qaytaradi — bu ekran
o'sha javobning **ikkinchi chizilishi**, ikkinchi hisoblanishi emas. Telefon o'z
soatlarini qo'shsa, u panel bilan ziddiyatga tushardi va bu **oylik kuni**
topilardi. Ranglar ham shu sababdan `design/StatusColor` dan.

## ⚠️ Bozorlik: qaysi raqam nimani buzadi

- **Oxirgi narx har narx maydonining yonida** — bu raqamning yagona qo'rig'i:
  telefonda yozilgan narx shu masalliqli har bir taomning tannarxini
  o'zgartiradi, va `9 000` o'rniga yozilgan `90 000` keyin oddiy raqamga
  o'xshaydi.
- **`clientId` har yuborishda yaraladi**: bozorda signal zaldagidan yomon, va
  usiz qayta yuborish ikkinchi kirim bo'lardi.
- **O'lchov hech qachon erkin matn**: bozor yalpizni bog'lam bilan, unni qop
  bilan sotadi. Qadoq yozilgan bo'lsa — ikki tomonlama tugma, konversiyani
  **server** qiladi.
- **Qatorni belgilash omborni qimirlatmaydi**: qoldiq faqat safar
  yakunlanganda o'zgaradi.
- **«Yo'q edi» — nol miqdor emas**, alohida javob.

---

## Build

```bash
export ANDROID_HOME=$HOME/Android/Sdk
export JAVA_HOME=/usr/lib/jvm/java-17-openjdk-amd64
./gradlew :app:testDebugUnitTest      # wire shakli va ruxsat qoidasi
./gradlew :app:assembleDebug
$ANDROID_HOME/platform-tools/adb install -r app/build/outputs/apk/debug/app-debug.apk
```

Release imzosi: `~/keys/keel-team.properties` (yoki
`KEEL_TEAM_KEYSTORE_PROPERTIES`). ⚠️ Fayl bo'lmasa build **imzosiz** chiqadi.

⚠️ **Kalit alohida** (`keel-team`): imzo kalitini almashtirib bo'lmaydi, va
bitta kalit beshta ilovani bitta portlash radiusiga qo'yadi. Bu — eng keng
tarqaladigan ilova: restorandagi har bir xodim uni o'z telefoniga o'rnatadi.

Ikonka va splash: `python3 tools/icons.py` (manba — `mobile/team/assets`).

---

## Hali yo'q

1. ⏳ **Telefonda sinalmagan.** APK quriladi, testlar o'tadi; ulangan qurilma
   yo'q. Birinchi qurilmada: ruxsat dialogi tugma bosilganda, punchning
   filial radiusi bilan tekshiruvi, bozorlik formasining klaviatura bilan
   ishlashi.
2. ⏳ **Sana tanlagich yo'q** — zakup ekranida sana matn maydoni
   (`2026-09-07`), Expo versiyasidagidek. Kalendar tanlagich keyingi ish.
3. ⏳ iOS faqat Expo papkasida (`mobile/team`).
