# Keel Courier — kuryer ilovasi (Android + iOS)

> ⚠️ **Bu ilova almashtirildi: `mobile/courier-android` (Kotlin + Compose).**
> Yangisi shu `applicationId` ni (`uz.keel.courier`) oladi, ya'ni telefonda
> eskisining o'rniga o'rnatiladi. Bu papka **hozircha saqlanadi**: qarorlar va
> ular ortidagi xatolar tarixi shu yerda birinchi marta yozilgan, va iOS hali
> faqat shu yerda.

Expo (SDK 57, RN 0.86, React 19). Bitta ilova, har bir kuryer o'z hisobiga
kiradi. Ofitsiant ilovasi bilan bir naqsh: qoidalar `frontend/src/lib` dan
import qilinadi, ekranlar shu yerda chiziladi.

## Nega ilova, PWA turgan joyda

`/kuryer` PWA ishlaydi va ishlashda davom etadi — lekin brauzer **yopilgan
ilovaga protsessor bermaydi**, va Android fonga o'tgan tabni bo'g'adi. Ya'ni
kuryerning joylashuvi aynan u telefonni cho'ntagiga solgan paytda to'xtaydi.
Natijasi ikki tomonlama:

- panelda kuryer xaritada **qotib qoladi** — dispetcher uni yo'qotadi;
- ⚠️ **«Yetkazildi» tugmasi ochilmaydi**: server oxirgi yuborilgan nuqtaga
  qaraydi (`arrivalBlocked`), va u o'n daqiqadan eski bo'lsa buyurtmani yopishga
  ruxsat bermaydi. Kuryer mijoz oldida turib tugmani bosa olmaydi.

Ilova aynan shu ikkitasini hal qiladi: haqiqiy joylashuv oqimi
(`watchPositionAsync`), smena davomida ekranni uyg'oq ushlash
(`expo-keep-awake`), va fonga o'tishdan **oldin** buferni yuborish.

**Fon rejimi bor** (`src/background.ts`): `expo-task-manager` vazifasi +
Android foreground service. Smena ochilganda ishga tushadi, yopilganda
to'xtaydi.

- ⚠️ **Vazifa o'zining JavaScript kontekstida ishlaydi**: React yo'q,
  xotirada token yo'q, va tizim uni ilova **o'ldirilgandan keyin** ham
  uyg'otadi. Shuning uchun har chaqiruvda `hydrateTokens()` va `setApiBase`
  qaytadan bajariladi — "ilova tirik" deb yozilgan versiya aynan o'zi
  qutqarishi kerak bo'lgan holatda ishlamaydi.
- ⚠️ **Doimiy bildirishnoma — narx, va u to'lanadi**: Android joylashuvni
  faqat foydalanuvchi ko'rib turgan xizmatga beradi. Yashirish mumkin ham emas,
  to'g'ri ham emas.
- ⚠️ **Fon ishlaganda ekran uyg'oq ushlanmaydi**: `expo-keep-awake` faqat fon
  ruxsati yo'q telefonda yoqiladi.
- Fon ruxsati rad etilsa ilova ishlashda davom etadi (ochiq turganda), va
  smena kartasi qaysi rejimda ekanini aytadi.
- ⚠️ **Play Store**: `ACCESS_BACKGROUND_LOCATION` uchun do'kon alohida
  tushuntirish va video so'raydi. Ilova ichidagi matn shu talabga yozilgan:
  nima uchun kerakligi ruxsat so'ralishidan **oldin** ekranda turadi.

## Ofitsiant ilovasidan nimasi bilan farq qiladi

| | Waiter | Courier |
|---|---|---|
| Ikonka | Keel belgisi | **aynan o'sha** — ikkalasi bitta mahsulot |
| Splash | «Waiter» | «Courier» (`scripts/courier-splash.py`) |
| Paket | `uz.keel.waiter` | `uz.keel.courier` |
| Token | `staff_token` | `courier_token` |
| Oflayn navbat | `expo-sqlite` | yo'q — bu ilova sotmaydi, faqat holat o'zgartiradi |
| Bildirishnoma | `kitchen` kanali | `delivery` kanali (7 xil voqea) |
| Joylashuv | smena ochilganda **bir marta** | smena davomida **oqim + fon xizmati** |

## Ekranlar

- **Buyurtmalar** — smena tugmasi (ishda emasman / bo'shman / bandman),
  joylashuv holati, va tayinlangan buyurtmalar. Har karta: manzil, mijoz,
  naqd/onlayn, taomga izoh, va bitta tugma — «Olib chiqdim» yoki «Yetkazildi».
- **Hisob** — bugun/hafta/oy/jami, **qo'ldagi naqd pul** (topshirilishi kerak),
  va yetkazilganlar tarixi.
- **Sozlamalar** — til, ko'rinish, hisob, chiqish, boshqa restoran.

## ⚠️ Yetkazildi tugmasi: qoida serverda, tugma ilovada

Server `CourierAdvanceOrder` da tekshiradi (`internal/handlers/courier.go` →
`arrivalBlocked`): buyurtma `delivery` bo'lsa, manzilida koordinata bo'lsa va
filialning `arrivalRadiusM` i noldan katta bo'lsa — kuryerning **oxirgi
yuborilgan** nuqtasi shu radius ichida bo'lishi shart, va nuqta 10 daqiqadan
eski bo'lmasligi kerak.

Ilovadagi `src/gate.ts` **xuddi shu hisobni** qiladi, faqat telefonning
**hozirgi** nuqtasi bilan. Uchta qoida:

- ⚠️ **Formula bir xil** (haversine) va radius serverdan olinadi
  (`GET /restaurant` → `delivery.arrivalRadiusM`) — o'ylab topilmaydi. Tugmasi
  ochiq, keyin server rad etadigan ilova — mijoz oldida «buzuq» ilova.
- ⚠️ **Ilova serverdan qattiqroq bo'lishi mumkin, yumshoqroq emas**: uning
  qo'lida telefonning eng yangi nuqtasi bor, serverda esa oxirgi **yuborilgani**.
  Ya'ni tugma bir lahza kechroq ochiladi — xavfsiz yo'nalish shu.
- ⚠️ **Sabab tugmaning ustida yoziladi**, ekranning boshqa joyida emas:
  o'chirilgan tugma va boshqa yerdagi tushuntirish — bu bir necha marta
  bosiladigan tugma.

Radius 0 bo'lsa tekshiruv umuman yo'q (qishloq restorani uchun ataylab),
`pickup` buyurtmada ham, manzilida nuqtasi yo'q buyurtmada ham yo'q — serverdagi
uchta istisnoning aynan o'zi.

## Joylashuv oqimi (`src/tracking.ts`)

- **15 soniyada bir marta** yuboriladi (har nuqtada emas — bu kuryer to'laydigan
  trafik va batareya; daqiqada bir marta emas — server 10 daqiqadan eski nuqtani
  ishlatmaydi).
- **25 metrdan yaqin** nuqtalar yuborilmaydi: turgan telefonning jitteri
  xaritadagi nuqtani sababsiz yurgizadi.
- Bufer **50 ta** nuqta bilan cheklangan — server baribir eng yangisini oladi.
- ⚠️ **Fonga o'tishda darhol yuboriladi** (`AppState`): tizim fondagi ilovani
  ogohlantirmay o'ldiradi, va bufer u bilan ketadi — jumladan eshik oldida
  tugmani ochadigan nuqta ham.
- ⚠️ **Aniqlik `Balanced`, eng yuqorisi emas**: radius 50–150 m, GPS ochiq
  havoda 10–30 m. Eng yuqori aniqlik javobni o'zgartirmaydi, sarflagan
  soniyalari esa darvoza oldida turgan odamniki.

## Bildirishnomalar

Kanal `delivery` — oshxonaning `kitchen` kanalidan **alohida**: Android'da
kanalni foydalanuvchi o'chiradi, va bittasini o'chirgan odam ikkinchisini ham
o'chirganini bilmay qoladi.

| Voqea | Serverdagi joyi |
|---|---|
| Yangi buyurtma berildi (manzili bilan) | `AdminAssignCourier` |
| Buyurtma sizdan olindi | `AdminAssignCourier`, `AdminMoveOrderBranch` |
| Buyurtma bekor qilindi (+ sabab) | `UpdateOrderStatus` |
| Buyurtma tayyor — olib chiqing | KDS `notifyReady` |
| Manzil o'zgardi | `AdminUpdateOrderAddress` |
| Naqd qabul qilindi | `AdminSettleCourierCash` |
| Hisob o'chirildi | `AdminUpdateCourier` |

- ⚠️ **Til token bilan birga yuboriladi** (`POST /courier/push` → `lang`):
  matnni server yozadi, ya'ni telefon uni tarjima qila olmaydi. Server
  `internal/i18n` katalogidan o'tkazadi (naqshlari bilan), ushlangan qiymat —
  ko'cha nomi, buyurtma raqami — tarjima qilinmaydi.
- ⚠️ **Chiqishda token o'chiriladi va fon xizmati to'xtatiladi**: kuryerning
  telefoni keyingi kuryerga beriladigan telefon, va qolib ketgan token
  mijozlarning ismi, raqami va manzilini uni ushlab turgan odamga yuboradi.
- Ruxsat **kirgandan keyin** so'raladi, ochilishda emas. Rad etish — javob:
  ilova ishlashda davom etadi, ro'yxat 20 soniyada bir yangilanadi.

### ⚠️ Android'da FCM shart — busiz token umuman olinmaydi

Expo relay xabarni **FCM orqali** yetkazadi, ya'ni Android build'ida Firebase
sozlanmagan bo'lsa `getExpoPushTokenAsync` **xato beradi** va ilova hech qachon
ro'yxatdan o'ta olmaydi. Ekranda bu «Ro'yxatdan o'tmadi» bo'lib ko'rinadi;
yangi buildda sababi ham yoziladi (`token: Default FirebaseApp is not
initialized…`).

⚠️ **Kredensiallar EAS'da har loyihaga alohida.** 2026-09-01 da tekshirilganda
`keel-courier` da ham, **`keel-waiter` da ham** FCM yo'q edi (EAS GraphQL:
`androidFcm: null`, `googleServiceAccountKeyForFcmV1: null`) — ya'ni ofitsiant
ilovasining Android push'i ham hech qachon ishlamagan. Uning Sozlamalar
ekranida ham «Ro'yxatdan o'tmadi» turgan bo'lishi kerak edi, lekin unga hech
kim qaramagan: yagona alomat — oshxona «Tayyor» bosadi va telefonga hech nima
kelmaydi, bu esa boshqa o'nta sababga o'xshaydi.

**2026-09-01 da ikkalasi ham sozlandi** (Firebase loyihasi `keel-7f31a`, bitta
service account kaliti ikkala ilovaga biriktirildi). Quyidagi qadamlar — yangi
ilova qo'shilganda yoki kalit almashtirilganda kerak bo'ladi:

1. Firebase konsolida loyiha oching (ikkala ilova uchun bittasi yetadi) →
   **Add app → Android** → paket nomi `uz.keel.courier` (waiter uchun
   `uz.keel.waiter`) → `google-services.json` ni yuklab oling.
2. Faylni `mobile/courier/google-services.json` ga qo'ying va `app.json` ga
   `"android": { "googleServicesFile": "./google-services.json" }` qo'shing.
   ⚠️ Bu maxfiy fayl emas — u har bir APK ichida ketadi.
3. Google Cloud konsolida o'sha loyihada **service account key (JSON)**
   yarating (*Firebase Cloud Messaging API* huquqi bilan) va EAS'ga bering:
   `npx eas-cli credentials -p android` → *Push Notifications (FCM V1)* →
   *Upload a service account key*.
4. Qayta build: `npx eas-cli build -p android --profile preview`.

⚠️ **Kredensial build vaqtida ilovaga kiritiladi**, shuning uchun sozlashning
o'zi yetmaydi — eski APK baribir ishlamaydi.

## Ishga tushirish

```bash
npm start          # Metro
npm run android    # ulangan qurilma yoki emulyator
```

⚠️ Ilova qaysi serverga borishini **build emas, hisob** hal qiladi
(`setApiBase`): bitta binar har bir restoranga xizmat qiladi.

## Build (EAS)

```bash
npx eas-cli login
npx eas-cli init            # birinchi marta: extra.eas.projectId ni yozadi
npx eas-cli build -p android --profile preview
```

Profillar `eas.json` da, ofitsiant ilovasidagidek: `preview` — **APK** (qo'lda
o'rnatiladi va o'lchanadi), `production` — **app bundle** (do'kon uchun),
`development` — dev client.

Android SDK shart emas: build **EAS'da, bulutda** quriladi, `npm start` esa
telefondagi Expo Go / dev client bilan ishlaydi. (`npx expo run:android` —
mahalliy native build — SDK talab qiladi, lekin bu yerda kerak emas.)

## O'lchov asosi (1-sentabr 2026)

| O'lchov | Qiymat | Qanday olingan |
|---|---|---|
| JS bundle (Hermes) | **1.9 MB** | `npx expo export --platform android` |
| Modullar | 714 | o'sha |
| `expo-doctor` | 21/21 | `npx expo-doctor` |
| APK | hali o'lchanmagan | EAS build'dan keyin, arzon telefonda |

Ofitsiant ilovasi 1.9 MB / 743 modul — bir xilga yaqin: bu yerda menyu va
oflayn navbat yo'q, lekin fon vazifasi va bildirishnomalar bor.
(Bildirishnoma va fon qo'shilishidan oldin 1.7 MB / 649 edi.)

## Keyingi qadamlar

1. **Haqiqiy telefonda o'lchash** — APK, batareya sarfi bir smenada, va fon
   xizmati tunab qolmasligi.
2. **Naqd pulni topshirish** — hozir faqat ko'rsatiladi; topshirishni panelda
   admin yozadi (`/admin/couriers/{id}/settle`).
3. **Push kelganda ro'yxatni darhol yangilash** — hozir bildirishnoma keladi,
   ro'yxat esa 20 soniyalik pollingda yangilanadi.

## ⚠️ Internetsiz ochilganda: kirish ekrani emas

Ilova ochilganda serverdan «bu kim?» deb so'raydi. So'rov **umuman
yetib bormasa** javob ilgari «chiqib ketgan» bo'lardi — ya'ni podvalda, o'lik
Wi-Fi da yoki interneti tugagan telefonda kuryer parol maydonini ko'rardi:
to'g'ri parolni teradi, u ishlamaydi, ilova esa «kirib bo'lmadi» deydi. Odam
o'zini ayblab yana teradi.

- ⚠️ **Uchta natija, ikkitasi emas**: `ApiError` — server gapirdi (401 ham
  shunga kiradi, ya'ni haqiqatan chiqib ketgan), boshqa har qanday xato esa
  so'rov yetib bormagani. Ikkinchisi endi `offline` holati va o'z ekrani.
- Ekran **o'zi qayta urinadi** (5 soniyada bir marta va ilova old planga
  qaytganda): odatdagi yechim — tarmoqning o'zi qaytishi, va faqat bosilganda
  tozalanadigan ekran odamni smena o'rtasida qulflab qo'yardi.
- Kirish ekranidagi xato ham shu farqni qiladi: parol to'g'ri bo'lsayu tarmoq
  yo'q bo'lsa, «kirib bo'lmadi» emas, «Internet yo'q» deb yoziladi.
