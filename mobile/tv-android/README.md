# Keel TV — native (Kotlin + Jetpack Compose)

Zaldagi televizor ilovasining **native Android** qayta yozilishi. Eskisi —
`mobile/tv` (Expo/RN 0.86) — hali o'chirilmagan; bu papka uni almashtiradi.
`applicationId` **bir xil** (`uz.keel.tv`), ya'ni yangi build eskisining ustiga
o'rnatiladi. ⚠️ Ikkalasi boshqa kalit bilan imzolangani uchun birinchi
o'rnatishda eskisini **o'chirish** kerak — telefon/televizor aks holda
«o'rnatib bo'lmadi» deydi.

Waiter va Owner bilan bir xil qaror: dizayn `mobile/android-design` dan keladi,
qolgan hamma narsa shu yerda ikkinchi nusxa — va har nusxaning testi bor.

---

## Nima ko'chirildi

| Eski fayl (`mobile/tv`) | Nima qiladi | Native fayl |
|---|---|---|
| `App.tsx` | 6 holat, rejim tanlash, tablo/pleyer | `MainActivity.kt` |
| `src/session.ts` | Ulash, yurak urishi, oflayn/uzilgan farqi | `Session.kt` |
| `src/store.ts` | SecureStore adapteri, install id | `design/TokenStore.kt` |
| `src/clock.ts` | Server vaqti va farqi | `Clock.kt` |
| `src/playlist.ts` | Yuklab olish, manifest, muddat | `Playlist.kt` |
| `src/board.tsx` (hook) | Tablo so'rovi va eskirish | `Board.kt` |
| `src/board.tsx` (UI) | Tablo va `split` chizig'i | `ui/screens/BoardScreen.kt` |
| `src/player.tsx` | Aylanma (`expo-video` → **Media3**) | `ui/screens/PlayerScreen.kt` |
| `src/screens.tsx` | Manzil, kod, bo'sh ekran | `ui/screens/SetupScreens.kt` |
| `plugins/withAndroidTV.js` | Leanback, banner, landscape | `AndroidManifest.xml` |
| — (yo'q edi) | UZ/RU/EN | `i18n/Strings.kt` |

Ko'chirilgan `@/lib` qoidalari: `serverAddress.ts` (design modulida, ulashilgan),
`api.ts` ning beshta TV metodi, `types.ts` dagi `TVScreenSelf` / `TVSlide`.

---

## Nima o'zgardi (va nega)

- **Til tanlash qo'shildi.** Expo ilovasi faqat o'zbekcha edi. Ilova aytadigan
  gaplar oz — lekin ular orasida *nima qilish kerakligini* aytadiganlari bor,
  va ularni o'qiydigan odam har doim ham o'zbekcha o'qimaydi. Tanlov **faqat
  manzil ekranida** (`LangSwitch`), chunki u televizor hayotida bir marta
  bosiladi.
- **Tema tanlash yo'q — doim qorong'i.** Zaldagi ikki metrli panelda yorug'
  tema — hech kim so'ramagan chiroq, va uni qaytaradigan odam xonada emas.
- **Rasm ekranga qarab kichraytiriladi** (`decodeScaled`). Telefondan yuklangan
  4000 px fotosurat xotirada 48 MB; 1 GB RAM'li televizorda bu — OOM, ya'ni
  qora devor.
- **Video Media3 (ExoPlayer)** bilan, `media3-ui` **siz**: PlayerView butun
  boshqaruv panelini olib keladi, bu ekranda esa bosadigan odam yo'q. Nisbat
  (`contain`) `onVideoSizeChanged` dan olinadi.
- **Fayl `.part` nomiga yuklanadi va keyin ko'chiriladi.** Yarim yuklangan fayl
  `exists()` uchun mavjud — pleyer uni ochadi, yiqiladi, va boshqa hech qachon
  qayta yuklamaydi.
- **Taymerlar `Job` bilan boshqariladi**, effekt bilan emas. Expo'da yurak
  urishi holatni **o'zgartirardi**, holat esa effektni qayta ishga tushirardi —
  himoyasi ref edi. Bu yerda buni tasodifan yozib bo'lmaydi.

## Nima o'zgarmadi (ataylab)

Bularning har biri bir marta sodir bo'lgan xatoning ustiga qurilgan; sabablari
kodda ⚠️ bilan yozilgan va `mobile/tv/README.md` da tarixi bor.

- Xonaga **hech qachon xato ko'rsatilmaydi**; uzilgan internet — jimlik.
- **Uzilgan (`Offline`) va ulanmagan (`Unreachable`) farqlanadi** — birinchisi
  o'ynayveradi, ikkinchisi manzilni qaytadan so'raydi.
- Playlist **oflayn o'ynaydi**, tablo esa **2 daqiqadan keyin jim bo'ladi**.
- Muddatni **televizor o'zi hisoblaydi**, vaqtni **server aytadi**.
- Manifest `filesDir` da (`cacheDir` da emas), eski fayllar **yangisi
  yozilgandan keyin** o'chiriladi.
- Bo'sh tablo — filial nomi; `split` da chiziq umuman ko'rinmaydi.
- Kod **ko'rsatiladi**, parol so'ralmaydi; holat `pollSecret` bilan olinadi.

---

## Ikonka, banner, splash

Hammasi bitta skriptdan: `python3 tools/icons.py` (manba —
`mobile/tv/assets`, ya'ni telefon ilovalari bilan **bir xil belgi**).

- `mipmap-*/ic_launcher(.png|_round.png)` — 48…192 px, `_round` haqiqatan
  dumaloq qirqilgan.
- `mipmap-anydpi-v26/ic_launcher.xml` — adaptive: navy fon + belgi +
  monochrome.
- `drawable-*/splash_icon.png` — 288dp maydon, `Theme.KeelTv.Splash` chizadi
  (`androidx.core:core-splashscreen`, Android 12 gacha ham bir xil).
- `drawable-*/tv_banner.png` — **320×180**, leanback bosh ekrani uchun.
  ⚠️ Busiz televizordagi yorliq bo'sh to'rtburchak bo'ladi va Play ro'yxatga
  qo'shmaydi.

---

## Build

```bash
export ANDROID_HOME=$HOME/Android/Sdk
export JAVA_HOME=/usr/lib/jvm/java-17-openjdk-amd64
./gradlew :app:testDebugUnitTest      # qoidalar va wire shakli
./gradlew :app:assembleDebug
$ANDROID_HOME/platform-tools/adb install -r app/build/outputs/apk/debug/app-debug.apk
```

Release imzosi: `~/keys/keel-tv.properties` (yoki
`KEEL_TV_KEYSTORE_PROPERTIES`). ⚠️ Fayl bo'lmasa build **imzosiz** chiqadi —
o'rnatib bo'lmaydi, ya'ni xato **baland ovozda**. Debug kaliti bilan jimgina
imzolangan release — aynan yuklab yuboriladigan build.

⚠️ **Kalit alohida** (`keel-tv`), waiter va owner'nikidan boshqa: imzo kalitini
almashtirib bo'lmaydi, bitta kalit esa uchta ilovani bitta portlash radiusiga
qo'yadi.

### Televizorga o'rnatish

1. **ADB (ishonchli)** — televizorda ishlab chiquvchi rejimi, keyin tarmoq
   orqali `adb connect <ip>:5555 && adb install -r ...`.
2. **USB** — «Noma'lum manbalar» + fayl menejer. ⚠️ Ba'zi televizorlarda fayl
   menejer yo'q.

⚠️ **Samsung va LG'ga o'rnatib bo'lmaydi** — ular Android emas (Tizen, webOS).

---

## Hali yo'q

1. ⏳ **Televizorda sinalmagan.** APK quriladi, testlar o'tadi; ulangan
   televizor ham, Android TV emulyatori ham yo'q. Birinchi qurilmada
   tekshiriladigan narsalar: D-pad fokusi manzil ekranida, IME'ning ochilishi,
   overscan (`EDGE = 48dp`) haqiqiy pleziqda, video dekoderi.
2. ⏳ **Yoqilganda o'zi ishga tushishi** (`BOOT_COMPLETED`). Native'da yozish
   oson, **ishlashi esa kafolatlanmagan**: Android 10+ fon ilovasiga aktivite
   ochishni taqiqlaydi va Google TV ba'zan baribir o'z ekranini birinchi
   qo'yadi. Yozilmadi — chunki «ishlaydi» deb yozib qo'yib, ertalab qora devor
   qoldirish eng yomon variant.
3. ⏳ Play uchun `.aab` va do'kon sahifasi (banner, skrinshotlar).
