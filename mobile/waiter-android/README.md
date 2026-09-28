# Keel Waiter — native (Kotlin + Jetpack Compose)

Ofitsiant ilovasining **native Android** qayta yozilishi. Eskisi —
`mobile/waiter` (Expo/React Native) — **hali ham jonli**, va u ishlab turgan
holda qoladi: bu papka almashtiruvchi, birinchi kundan uni o'chiradigan narsa
emas.

⚠️ **Nima uchun bu qaror qimmat ekani yozib qo'yiladi.** `mobile/waiter/README.md`
Expo tanlanganini bitta sabab bilan asoslaydi: `frontend/src/lib` dagi ~28 000
qator sof TypeScript qoida **ulashiladi**, ikkinchi tilda qayta yozilmaydi.
Native Kotlin bu ulanishni uzadi — ya'ni ulashilgan har bir qoida **ikkinchi
nusxaga** aylanadi, va bu kodbazaning eng ko'p takrorlangan darsi: *bir
qoidaning ikki nusxasi ajraydi, va ajragani restorandagi telefonda qoladi.*
Qaror qabul qilindi; shuning uchun bu README ikki nusxa qayerda ekanini
**ro'yxat qilib** boradi, va ularning har biri uchun testni talab qiladi.

---

## Nima ko'chiriladi (ekran bo'yicha)

Eski ilovada 4 949 qator (App + 22 fayl). Bo'limlar:

| Eski fayl | Nima qiladi | Native fayl |
|---|---|---|
| `App.tsx` | 4 holat: `noServer / offline / signedOut / ready`, 3 tab | ✅ `MainActivity.kt` |
| `auth.tsx` | Server manzili + login | ✅ `ui/screens/AuthScreens.kt` |
| `floor.tsx` | Zal: stollar, zonalar, band/bo'sh | ✅ `ui/screens/FloorScreen.kt` |
| `check.tsx` (773 q.) | Chek: qatorlar, menyu, fire, precheck, portion | ✅ `ui/screens/CheckScreen.kt` |
| `menu.tsx` | Menyu: qidiruv, kategoriya, 3 ko'rinish | ✅ `MenuPane.kt` + `DishCard.kt` |
| `line.tsx` | Qator tahriri: qty, izoh, void/write-off, PIN | ✅ `ui/screens/LineDialog.kt` |
| `table.tsx` | Mehmon soni, bo'lish, ko'chirish, birlashtirish | ✅ `ui/screens/TableActions.kt` |
| `profile.tsx` + `clock.tsx` | Davomat: kalendar, soatlar, smena + GPS | ✅ `ui/screens/ProfileScreen.kt` |
| `settings.tsx` | Til, tema, push holati, ikki chiqish yo'li | ✅ `ui/screens/SettingsScreen.kt` |
| `notice.tsx` | Keyingi amalni o'zgartiradigan xabar (sheet) | ✅ `ui/components/Notice.kt` |
| `offlinescreen.tsx` | Internetsiz ochilganda — login ekrani **emas** | ✅ `ui/screens/OfflineScreen.kt` |
| `stepper.tsx` | Minus / son / plus | ✅ `GlassStepper.kt` |
| `press.tsx` | Bosishga javob (ripple) | ✅ Compose `clickable` (platforma o'zi chizadi) |
| `theme.ts`, `ui.ts` | Ranglar va shakllar | ✅ `ui/theme/` |
| `i18n.ts` (601 q.) | UZ/RU/EN lug'at | ✅ `i18n/Strings.kt` |
| `tokens.ts` | SecureStore adapteri | ✅ `TokenStore.kt` |
| `session.ts` | ⚠️ **Uch natija, ikkita emas** (ApiError / tarmoq / ok) | ✅ `Session.kt` |
| `device.ts` | Install id + `X-Keel-*` sarlavhalar | ✅ `KeelWaiterApp.kt` + `KeelApi.kt` |
| `push.ts` | Bildirishnoma ro'yxatdan o'tishi | ✅ `push/Push.kt` — ⚠️ **backend ishi qoldi** |
| `money.ts` | So'm formati | ✅ `ServerAddress.kt` |
| `offline.ts` | SQLite navbat | ✅ `data/Outbox.kt` (Room) — quyida |

Ko'chirilgan `@/lib` qoidalari: `api.ts` (22 metod), `types.ts` (14 tur),
`serverAddress.ts`, `i18n/content.ts` (`contentName`), `attendance.ts`
(`STATUS_COLOR`, `formatDuration`), `orderFlow.ts` (`timeAgo`),
`offline/store.ts` (seam).

---

## ⚠️ Push: backendga tegadigan yagona ish

Bu **rejaning yagona bloker**i, va u ilovada emas — serverda.

`internal/push/expo.go` har bir bildirishnomani **Expo relay**ga yuboradi
(`https://exp.host/--/api/v2/push/send`), va `staff_device.token` — Expo
tokeni (`ExponentPushToken[...]`). Native ilova FCM registratsiya tokenini
oladi; **Expo uni qabul qilmaydi**. Ya'ni:

- Serverga **FCM v1 yo'nalishi** qo'shiladi (`google-services.json` allaqachon
  `mobile/waiter/` da bor, service-account kaliti kerak bo'ladi).
- ⚠️ **Qo'shimcha, almashtiruvchi emas**: kuryer, team, owner va TV ilovalari
  hali Expo'da. Yo'nalish **token shakliga** qarab tanlanadi
  (`ExponentPushToken[` prefiksi → Expo, aks holda FCM), sozlamaga emas — bir
  ilova ko'chganda qolgan to'rttasi jimgina jim bo'lib qolmasin.
- Kanal nomi o'zgarmaydi: `kitchen` (`push.KitchenChannel`). ⚠️ Ikki yozilish
  mos bo'lishi shart — telefon yaratmagan kanalga kelgan xabar **jimgina**
  tushiriladi.

---

## Dizayn: Apple uslubidagi «glass», Keel to'q sarig'i

Primary — **`#E2590D`**, o'zgarmaydi (`ui/theme/Color.kt`).
⚠️ **Brend rangi temaga qarab ochilmaydi**: u kassada, saytda va qog'oz chekda
bir xil, va telefonda boshqacha bo'lsa mehmon ikki mahsulot ko'radi.

⚠️ **Android'da `backdrop-filter` yo'q, va buni yashirish — bu yerdagi asosiy
tuzoq.** iOS'da material **orqasidagini** oladi; Android'da `RenderEffect`
**o'zi qo'yilgan qatlamni** xiralashtiradi. Shuning uchun shisha haqiqatan
ishlaydigan shaklda qurilgan: har ekranda bir marta chiziladigan **xira fon**
(`KeelBackground` — ikkita yumshoq nur dog'i) va uning ustidagi yarim shaffof
panellar (`Modifier.glass`). O'z fonini xiralashtirmoqchi bo'lgan panel yo
hech nima qilmaydi, yo o'z matnini xiralashtiradi.

⚠️ **`RenderEffect` — API 31+.** `minSdk = 26`, chunki bu yerda sotiladigan
telefonlar Android 8–11: blur **yo'q** bo'lgan holat dizaynning ishlaydigan
holati bo'lishi shart, kamchiligi emas. Fallback — biroz quyuqroq to'ldirish
(`glassStrong`), ya'ni «muzlagan» ko'rinish. ⚠️ **Hech qachon CPU blur
kutubxonasi**: arzon telefonda har kadrdagi blur — Expo ilovasi bir relizni
yo'qotib tuzatgan aynan o'sha qotish.

Shishani shisha qiladigan narsa — **yuqori qirradagi yorug'lik**
(`glassHighlight`). Tekis yarim shaffof quti — «opacity: 0.7»; gradient stopi
bitta, va farqi shu.

- **Tablar** (`GlassTabBar.kt`): pastda suzib turgan shisha kapsula, tanlangani
  — prujinali animatsiya bilan siljiydigan to'q sariq tabletka.
  ⚠️ **Yozuvli, faqat ikonka emas** — uchta glif har yangi ofitsiantning
  birinchi kechasidagi taxmin. ⚠️ `navigationBarsPadding()` — busiz «Zal»ga
  qilingan bosish Android'ning «Orqaga» tugmasiga tushadi (haqiqiy
  telefonlarda, support chatida xabar qilingan).
- **Taom cardlari** (`DishCard.kt`): shisha karta, rasm ustida gradient scrim va
  narx tabletkasi, plus — to'q sariq gradientli doira.
  ⚠️ Chekka qo'shilishi bilan plus **stepper**ga aylanadi va karta bosilmaydigan
  bo'ladi: bitta taomni qo'shishning ikki yo'li bir necha piksel va **bitta**
  bilan farq qiladi, va bu farq faqat stolda bilinadi.
  ⚠️ Rasmsiz taom **nomlangan bo'shliq** oladi.
  ⚠️ Rasm hech qachon to'liq o'lchamda emas — `imageUrl(path, uploadsBase, 300)`.

---

### Zal va chek (2026-09-28 qayta chizilgan)

- **Zal**: soni bilan filtrlar (Hammasi / Band / Bo'sh / Tayyor / Meniki —
  oxirgi ikkitasi faqat bo'sh bo'lmasa), stol kartasida raqam, daqiqa, summa
  va **bitta** eng muhim nishon (tayyor → yangi → hisob berilgan → mehmonlar).
  ⚠️ Ikki burchakdagi rangli nuqta emas — qaysi biri nima ekanini hech kim
  aytib bera olmasdi. Tayyor taomli stol — yashil chegara (animatsiya emas:
  har tayyor stolga cheksiz kadr arzon telefonda qimmat).
- **Chek**: qatorlar guruhlarda — *Yuborilmagan → Tayyor — olib boring →
  Oshxonada → Berilgan*; pastda doimiy panel (Jami + bitta asosiy amal).
  Chek/Menyu — segment, ⚠️ **to'q sariq emas**: ekrandagi yagona to'q sariq
  «Oshxonaga yuborish».
- Holat jarayonda (`data/FloorStore.kt`), ekranda emas — qarang
  `docs/DECISIONS.md` → «Native ofitsiantda qotishning yana beshta sababi».

## Nima qolgan

⚠️ **Bu ro'yxat 2026-09-06 da tekshirildi**: to'rttadan uchtasi bajarilgan edi
va ro'yxat eskirgan holicha turgan edi. Eskirgan «hali yo'q» ro'yxati —
qilingan ishni ikkinchi marta rejalashtirishga chaqiruv.

1. ✅ **Oflayn navbat ulandi.** `data/Outbox.kt` (Room) va u chek oqimidan
   o'tadi: `CheckScreen` qatorlarni `outbox.send(...)` orqali yuboradi, javobi
   `Sent / Refused / Queued`, va navbatdagilar soni tab panelida ko'rinadi.
2. ✅ **Splash va release imzosi bajarilgan** (`core-splashscreen`,
   `~/keys/keel-waiter.jks`). Release APK **3.1 MB**.
3. ✅ **FCM v1 serverda** (`internal/push/fcm.go`), yo'nalish token shakliga
   qarab. Ilova FCM tokenini oladi va `kitchen` kanalida ko'rsatadi.
4. ✅ **Wire testi qo'shildi** (`WireShapeTest`, 7 ta): chek, qator, void,
   zal, menyu va hisobot — JSON `handlers/till.go` dan ko'chirilgan. Ilgari bu
   ilovada **umuman test yo'q edi**, va aynan shu ilova nomi o'zgargan
   maydondan uchta xatoni bir vaqtda yuborgan.
5. ⏳ **Telefonda sinalmagan.** APK quriladi, testlar o'tadi; ulangan qurilma
   yo'q.

⚠️ **`data/TokenStore.kt` o'chirildi** (2026-09-06): u `android-design` dagi
ulashilgan nusxaning ikkinchi nusxasi edi va **hech kim ishlatmasdi** — ilova
`uz.keel.design.TokenStore` ni import qiladi. Kompilyator bunday faylni
ko'rmaydi: u quriladi, tahrirlanadi va hech qayerga ta'sir qilmaydi. Aynan shu
modul mavjudligining sababi.

### Tasdiqlangan muhit (5-sentabr 2026)

| Narsa | Qiymat |
|---|---|
| JDK | 17.0.20 (`/usr/lib/jvm/java-17-openjdk-amd64`) |
| Android SDK | `~/Android/Sdk`, platform **36**, build-tools 36.0.0 |
| Gradle | 9.3.1 (wrapper) |
| AGP / Kotlin | 8.13.0 / 2.2.20 |
| Compose BOM | **2026.05.01** |
| Debug APK | **24 MB** |

⚠️ **Compose BOM 2026.05.01, eng oxirgisi emas.** 2026.06+ `compileSdk 37` va
AGP 9.1 talab qiladi, o'rnatilgan SDK'da esa faqat platform 36 bor. Yangilash
kerak bo'lsa **ikkalasi birga**: `sdkmanager "platforms;android-37"` + AGP 9.x.
Bittasini yangilash `checkDebugAarMetadata` da yiqiladi.

⚠️ **24 MB — debug APK**, ya'ni Expo'ning 68 MB `preview` buildi bilan
to'g'ridan-to'g'ri solishtirib bo'lmaydi (u universal, har protsessor uchun
kutubxonalar bilan). Halol taqqoslash release `.aab` chiqqanda bo'ladi.

## Ishga tushirish

```bash
export ANDROID_HOME=$HOME/Android/Sdk
export JAVA_HOME=/usr/lib/jvm/java-17-openjdk-amd64
./gradlew :app:assembleDebug
$ANDROID_HOME/platform-tools/adb install -r app/build/outputs/apk/debug/app-debug.apk
```

⚠️ Ilova qaysi serverga borishini **build emas, hisob** hal qiladi
(`KeelApi.useServer`): bitta binar har bir restoranga xizmat qiladi, ya'ni
manzil `BuildConfig` ga muhrlanishi mumkin emas — bu `NEXT_PUBLIC_*` allaqachon
qilgan xato.
