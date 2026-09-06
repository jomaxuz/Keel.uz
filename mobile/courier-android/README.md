# Keel Courier — native (Kotlin + Jetpack Compose)

Kuryer ilovasining **native Android** qayta yozilishi. Eskisi — `mobile/courier`
(Expo/RN 0.86) — o'chirilmagan; bu papka uni almashtiradi. `applicationId`
**bir xil** (`uz.keel.courier`), ya'ni yangi build eskisining ustiga
o'rnatiladi. ⚠️ Ikkalasi boshqa kalit bilan imzolangani uchun birinchi
o'rnatishda eskisini **o'chirish** kerak.

To'rtinchi native ilova: dizayn `mobile/android-design` dan (waiter, owner va TV
bilan bitta modul), qolgan hamma narsa shu yerda ikkinchi nusxa — va har
nusxaning testi bor.

---

## Nima ko'chirildi

| Eski fayl (`mobile/courier`) | Nima qiladi | Native fayl |
|---|---|---|
| `App.tsx` | 5 holat, 3 tab | `MainActivity.kt` |
| `src/session.ts` | Kim, qayerda, oflayn/chiqib ketgan farqi | `Session.kt` |
| `src/tokens.ts` | SecureStore adapteri | `design/TokenStore.kt` |
| `src/tracking.ts` | Joylashuv oqimi, bufer, flush | `location/LocationService.kt` |
| `src/background.ts` | Fon xizmati | **o'sha xizmat** (quyida) |
| `src/gate.ts` | «Yetkazildi» tugmasining qoidasi | `location/Gate.kt` |
| `src/orders.tsx` | Buyurtmalar, karta, yo'l tanlash | `ui/screens/OrdersScreen.kt` |
| `src/shift.tsx` | Smena va joylashuv kartasi | `ui/screens/ShiftCard.kt` |
| `src/earnings.tsx` | Hisob, qo'ldagi naqd, tarix | `ui/screens/EarningsScreen.kt` |
| `src/settings.tsx` | Til, tema, push holati, ikki chiqish | `ui/screens/SettingsScreen.kt` |
| `src/auth.tsx` | Manzil + login | `ui/screens/AuthScreens.kt` |
| `src/offlinescreen.tsx` | Internetsiz ochilganda | `ui/screens/OfflineScreen.kt` |
| `src/push.ts` | Bildirishnoma (Expo relay → **FCM**) | `push/Push.kt` |
| `src/i18n.ts` | UZ/RU/EN | `i18n/Strings.kt` |
| `src/theme.ts`, `ui.ts` | Ranglar, shakllar | `design/` (ulashilgan) |

---

## ⚠️ Joylashuv: ruxsatni Android o'zi so'raydi

**Smena ochilishi bilan** (`off` dan chiqqanda) `ShiftCard` ruxsat dialogini
o'zi ko'taradi — Expo ilovasidagidek tugma kutib turmaydi. Sabab: kuryer
smenani ochib, yo'lga chiqib, eshik oldida «Yetkazildi» bosilmasligidan bilib
qolardi. Ruxsat — sozlama emas, **smenaning o'zi**.

- `ACCESS_FINE_LOCATION` **va** `ACCESS_COARSE_LOCATION` bitta dialogda:
  Android 12 «taxminan» berishga ruxsat beradi, faqat `fine` so'ralsa u javob
  **rad** bo'lib keladi. 100 metrlik nuqta hech nimadan yaxshiroq.
- ⚠️ **`ACCESS_BACKGROUND_LOCATION` yo'q, ataylab.** `location` turidagi
  foreground service ilova ekranda turganda ishga tushirilsa, oddiy
  «while-in-use» ruxsati bilan **ekran o'chganda ham** nuqta olishda davom
  etadi — bu aynan smena. Fon ruxsati faqat tizim ilovani butunlay o'ldirgan
  holatni qoplardi, narxi esa: ikkinchi, qo'rqinchliroq dialog va Play
  Store'ning video so'raydigan tekshiruvi. Voz kechilgani
  `location/LocationService.kt` da yozilgan.
- Doimiy bildirishnoma — narx va u to'lanadi: Android joylashuvni faqat
  foydalanuvchi ko'rib turgan xizmatga beradi. **Past muhimlik** bilan (`shift`
  kanali), chunki u yangi hech nima aytmaydi — buyurtmalar `delivery`
  kanalida, va birini ovozsiz qilgan odam ikkinchisini ham ovozsiz qilmasin.
- 15 soniyada bir marta yuboriladi, 25 metrdan yaqin nuqtalar tashlanadi,
  bufer 50 ta bilan cheklangan (server baribir eng yangisini oladi).

## ⚠️ «Yetkazildi» tugmasi: qoida serverda, tugma ilovada

Server `arrivalBlocked` da tekshiradi: `delivery` buyurtma, manzilida
koordinata bor, filialning `arrivalRadiusM` i noldan katta — kuryerning oxirgi
**yuborilgan** nuqtasi radius ichida va 10 daqiqadan eski emas.

`location/Gate.kt` **xuddi shu hisobni** telefonning **hozirgi** nuqtasi bilan
qiladi. Radius serverdan olinadi (`GET /restaurant`), formula bir xil
(haversine), muddat bir xil. Ilova serverdan **qattiqroq** bo'lishi mumkin,
yumshoqroq emas — tugma bir lahza kechroq ochilsin, mijoz oldida rad javobi
kelmasin. Sabab tugmaning **ustida** yoziladi.

## Bildirishnomalar

Kanal `delivery`, FCM to'g'ridan-to'g'ri (Expo relay orqali emas — server
token shakliga qarab yo'naltiradi, `internal/push/send.go`).
`google-services.json` `mobile/courier` dan ko'chirildi: paket nomi bir xil,
ya'ni Firebase tomonda yangi hech nima sozlash kerak emas.

Til token bilan birga yuboriladi (`POST /courier/push` → `lang`) — matnni
server yozadi.

---

## Ikonka va splash

`python3 tools/icons.py` (manba — `mobile/courier/assets`, ya'ni qolgan
ilovalar bilan **bir xil belgi**): beshta zichlikda launcher (dumaloqi
haqiqatan qirqilgan), adaptive XML, splash, va **bildirishnoma ikonkasi**.

⚠️ Bildirishnoma ikonkasi **oq siluet**: Android uni siluetga aylantiradi —
har shaffof bo'lmagan piksel oq bo'ladi — ya'ni rangli ikonka status barda
to'q sariq dog' bo'lib chiqadi.

---

## Build

```bash
export ANDROID_HOME=$HOME/Android/Sdk
export JAVA_HOME=/usr/lib/jvm/java-17-openjdk-amd64
./gradlew :app:testDebugUnitTest      # gate qoidasi va wire shakli
./gradlew :app:assembleDebug
$ANDROID_HOME/platform-tools/adb install -r app/build/outputs/apk/debug/app-debug.apk
```

Release imzosi: `~/keys/keel-courier.properties` (yoki
`KEEL_COURIER_KEYSTORE_PROPERTIES`). ⚠️ Fayl bo'lmasa build **imzosiz**
chiqadi — o'rnatib bo'lmaydi, ya'ni xato baland ovozda. Debug kaliti bilan
jimgina imzolangan release — aynan yuklab yuboriladigan build.

⚠️ **Kalit alohida** (`keel-courier`): imzo kalitini almashtirib bo'lmaydi, va
bitta kalit to'rtta ilovani bitta portlash radiusiga qo'yadi.

---

## Hali yo'q

1. ⏳ **Telefonda sinalmagan.** APK quriladi, testlar o'tadi; ulangan Android
   telefon yo'q. Birinchi qurilmada tekshiriladigan narsalar: ruxsat dialogi
   smena ochilganda, xizmat ekran o'chganda ham nuqta yuborishi, batareya
   sarfi bir smenada, va «Yetkazildi» ning haqiqiy manzilda ochilishi.
2. ⏳ **Naqd pulni topshirish** — hozir faqat ko'rsatiladi; topshirishni
   panelda admin yozadi (`/admin/couriers/{id}/settle`). Eski ilovada ham
   shunday edi.
3. ⏳ **Push kelganda ro'yxatni darhol yangilash** — hozir bildirishnoma
   keladi, ro'yxat esa 20 soniyalik pollingda yangilanadi.
