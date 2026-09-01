# Keel Courier — kuryer ilovasi (Android + iOS)

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

⚠️ **Hozircha fon rejimi yo'q.** Ilova yopilgan yoki ekran o'chgan bo'lsa
joylashuv yuborilmaydi — shuning uchun smena kartasida shu ochiq aytilgan.
Keyingi qadam: `expo-task-manager` + Android foreground service (doimiy
bildirishnoma bilan). U alohida qilinadi, chunki Play Store'ning joylashuv
siyosati alohida tekshiruv talab qiladi.

## Ofitsiant ilovasidan nimasi bilan farq qiladi

| | Waiter | Courier |
|---|---|---|
| Ikonka | Keel belgisi | **aynan o'sha** — ikkalasi bitta mahsulot |
| Splash | «Waiter» | «Courier» (`scripts/courier-splash.py`) |
| Paket | `uz.keel.waiter` | `uz.keel.courier` |
| Token | `staff_token` | `courier_token` |
| Oflayn navbat | `expo-sqlite` | yo'q — bu ilova sotmaydi, faqat holat o'zgartiradi |
| Bildirishnoma | bor (oshxona «tayyor» desa) | hali yo'q (pastga qarang) |
| Joylashuv | smena ochilganda **bir marta** | smena davomida **oqim** |

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

⚠️ Bu mashinada Android SDK yo'q (`adb` ham, `ANDROID_HOME` ham), ya'ni
`npx expo run:android` ishlamaydi — EAS bulutda quradi.

## O'lchov asosi (1-sentabr 2026)

| O'lchov | Qiymat | Qanday olingan |
|---|---|---|
| JS bundle (Hermes) | **1.7 MB** | `npx expo export --platform android` |
| Modullar | 649 | o'sha |
| `expo-doctor` | 21/21 | `npx expo-doctor` |
| APK | hali o'lchanmagan | EAS build'dan keyin, arzon telefonda |

Ofitsiant ilovasi 1.9 MB / 743 modul — farq kutilgan: bu yerda menyu, oflayn
navbat va bildirishnomalar yo'q.

## Keyingi qadamlar

1. **Fon rejimi** — `expo-task-manager` + foreground service. Eng katta
   qiymat, eng ko'p siyosat ishi.
2. **Bildirishnoma** — «sizga yangi buyurtma berildi». Serverda kuryer uchun
   push hali yo'q (ofitsiantda bor, `push_settings`), ya'ni bu ikki tomonlama
   ish.
3. **Naqd pulni topshirish** — hozir faqat ko'rsatiladi; topshirishni panelda
   admin yozadi (`/admin/couriers/{id}/settle`).
