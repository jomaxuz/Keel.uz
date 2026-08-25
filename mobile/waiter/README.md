# Keel Waiter — ofitsiant ilovasi (iOS + Android)

Expo (SDK 57, RN 0.86, React 19). Bitta ilova, har bir ofitsiant o'z hisobiga
kiradi.

## Nega Expo

⚠️ **Tanlovni framework emas, ulashiladigan kod hal qildi.** `frontend/src/lib`
da ~28 000 qator sof TypeScript qoida bor — oflayn navbat, markirovka kodlari,
xizmat haqi arifmetikasi, soat qoidasi, uch tilli lug'at. Ekranlar (~13 000
qator) qaysi texnologiyada bo'lsa ham qaytadan chiziladi, qoidalar esa yo
**ulashiladi**, yo **ikkinchi tilda qaytadan yoziladi**. Ikkinchisi — bu
kodbazaning eng ko'p takrorlangan darsi: bir qoidaning ikki nusxasi ajraydi, va
ajragani restorandagi mashinada qoladi.

Flutter yoki native — Dart/Swift/Kotlin, ya'ni o'sha 28 000 qator ikkinchi
marta. Ustiga: ⚠️ **Mac yo'q**, va EAS Build Linux'dan iOS binary quradi.

⚠️ **Expo Go bilan aralashtirmang.** "Expo sekin, bundle katta" degan tajriba
odatda **Expo Go** — sinov qobig'i, ichida butun SDK bor va dev rejimida
ishlaydi. EAS Build (yoki `npx expo run:android`) faqat o'rnatilgan
modullardan iborat oddiy native binary quradi.

## Qanday ulangan

Fayllar **ko'chirilmagan va nusxalanmagan**. Metro `frontend/src` ni watch
qiladi, `@/` aliasi esa veb ilova va Windows kassadagi bilan **bir xil** — ya'ni
bu yerda yozilgan import u yerdagidek yoziladi.

⚠️ **`lib/` ni alohida paketga ko'chirish tozaroq shakl va noto'g'ri birinchi
qadam**: u veb ilova va kassadagi har bir importga tegadi, va telefon buildi
ishlashini bilishdan **oldin** qilinsa — katta xavf, hech qanday foyda evaziga.

⚠️ **Bitta React nusxasi**: `resolver.nodeModulesPaths` shu loyihaga
qulflangan. Busiz ulashilgan papka o'z `node_modules` idan ikkinchi React
oladi, va natija — "invalid hook call", ya'ni komponentdagi xatoga o'xshaydigan
konfiguratsiya xatosi.

## ⚠️ Tuzoq: Expo'ning tsconfig'i DOM turlarini o'z ichiga oladi

`expo/tsconfig.base` da `"lib": ["DOM", "ESNext"]`. Ya'ni `window`,
`document`, `indexedDB` — **kompilyatsiyadan o'tadi** va telefonda ishlamaydi.
TypeScript bu yerda qo'riqchi emas.

Amaliy oqibati aniq: `lib/offline/store.ts` IndexedDB'ni qidiradi, telefonda uni
topmaydi va `available()` `false` qaytaradi — ya'ni **oflayn navbat jimgina
hech nima qilmaydi**. Bu Windows kassadagi bilan aynan bir sinf xato
("ikkala dvigatel ham ishlaydi, noto'g'risi faqat svet o'chganda bilinadi").

Yaxshi tomoni: yechim **allaqachon shaklda** — dvigatel bitta faylda tanlanadi
(`lib/offline/store.ts`), ya'ni telefonga uchinchi dvigatel (`expo-sqlite`)
o'sha joyga qo'shiladi. Qoidalar tegilmaydi.

## Tokenlar

`src/tokens.ts` — `lib/tokenStore.ts` tikuvining telefon tomoni.

- **SecureStore**, AsyncStorage emas: bu tokenlar smena ocha oladi va pul
  ko'radi, ya'ni keychain/keystore joyi.
- ⚠️ **Startda bir marta xotiraga o'qiladi**: qoidalar tokenni **sinxron**
  so'raydi (`request()` har chaqiruvda o'qiydi), telefon saqlagichi esa async.
  Assimetriya shu bitta faylda qoladi.
- ⚠️ Kalitlar **brauzerdagi bilan bir xil yozilishda**.

## EAS: build va o'lchov

Loyiha Expo hisobiga ulangan (`extra.eas.projectId`). Build profillari
`eas.json` da:

- `preview` — ⚠️ **APK**, `.aab` emas: bu build arzon telefonga **qo'lda
  o'rnatiladi** va o'lchanadi, `.aab` ni esa o'rnatib bo'lmaydi. Do'kon buildi
  aynan shu sababdan alohida profil.
- `development` — dev client, kundalik ish uchun.
- `production` — do'kon uchun.

```bash
npx eas-cli login              # bir marta, interaktiv
npx eas-cli build -p android --profile preview
```

⚠️ **Bu mashinada Android SDK yo'q** (`adb` ham, `ANDROID_HOME` ham), ya'ni
`npx expo run:android` ishlamaydi. EAS bulutda quradi — Mac yo'qligi bilan bir
qatorda, bu ham "nega Expo" savolining amaliy javobi.

## Ishga tushirish

```bash
npm start          # Metro
npm run android    # ulangan qurilma yoki emulyator
```

⚠️ Ilova qaysi serverga borishini **build emas, hisob** hal qiladi
(`setApiBase`): bitta binar har bir restoranga xizmat qiladi, ya'ni manzil
`NEXT_PUBLIC_*` kabi build vaqtida muhrlanishi mumkin emas.

## O'lchov asosi (25-avgust 2026)

⚠️ **Raqamlar birinchi kundan yozib boriladi**, chunki "sekinlashdi" degan
shikoyatga javob berish uchun **nima bilan solishtirishni** bilish kerak.
Xususiyat qo'shilganda bu jadval yangilanadi.

| O'lchov | Qiymat | Qanday olingan |
|---|---|---|
| JS bundle (Hermes bayt-kod) | **1.5 MB** | `npx expo export --platform android` |
| Modullar | 591 | o'sha |
| `expo-doctor` | 21/21 | `npx expo-doctor` |

Bunga kirgani: RN yadrosi, Expo modullari, `expo-secure-store`, va ulashilgan
`lib/api.ts` + `lib/types.ts` + qoidalar. **Kirmagani**: uch tilli lug'at
(~12 000 qator) — u hali import qilinmagan, va import qilinganda faqat
kerakligi kiradi.

⚠️ **Bu APK hajmi emas.** APK ustiga RN runtime va native kutubxonalar
qo'shiladi; haqiqiy raqam EAS build'dan keyin, arzon Android telefonda
o'lchanadi.

## Tezlik: birinchi kundan, keyinga qoldirilmaydi

Sotiladigan telefonlar arzon Android. Qoidalar:

- ro'yxatlar **`FlashList`**, `ScrollView` emas — 200 taomli menyu shu bilan
  yashaydi;
- rasm **hech qachon to'liq o'lchamda emas**: server `?w=300|600|1200` beradi
  (`lib/api.ts` → `imageUrl`);
- render ichida hisob yo'q — kassadagi `useOffline` darsi shu yerda ham amal
  qiladi (bir renderda yaratilgan obyekt butun ekranni qayta yuklaydigan
  siklga aylanishi mumkin);
- Hermes va yangi arxitektura — SDK 57 da standart, o'chirilmaydi.
