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
| JS bundle (Hermes bayt-kod) | **1.9 MB** | `npx expo export --platform android` |
| Modullar | 743 | o'sha |
| `expo-doctor` | 21/21 | `npx expo-doctor` |
| APK (`preview`, universal) | **68 MB** | telefonda o'lchandi, 25-avgust |

### ⚠️ 68 MB — nima uchun, va nima uni kamaytiradi

`preview` profili **universal APK** beradi: ichida **har bir protsessor
arxitekturasi** uchun alohida native kutubxonalar to'plami bor (`arm64-v8a`,
`armeabi-v7a` va h.k.), va telefon ulardan **bittasini** ishlatadi. Ya'ni
68 MB ning katta qismi — o'sha telefon hech qachon ochmaydigan fayllar.

- **Do'kon buildi bunday emas**: `production` profili **app bundle** (`.aab`)
  beradi, Play Store esa har telefonga faqat unga tegishli qismni yuboradi.
  Kutilayotgan yuklab olish hajmi sezilarli darajada kichik.
- ⚠️ **Lekin `.aab` ni qo'lda o'rnatib bo'lmaydi**, shuning uchun o'lchov
  buildi baribir APK bo'lib qoladi — bu raqamni do'kondagi hajm deb o'qimaslik
  kerak.
- Qo'shildi: **ProGuard** (yetib bo'lmaydigan kodni olib tashlaydi) va
  **resource shrinking** — ikkinchisini ko'pincha unutishadi, holbuki RN
  ilovasida paketning katta qismi aynan kutubxonalarning rasm va tarjimalari.

### O'lchangan ikkita qaror

⚠️ **Panel lug'ati import qilinmadi** (`lib/i18n/admin`): **+500 KB**
(1.6 → 2.1 MB) — uch tildagi butun admin paneli, ~60 ta satr uchun. Ilova o'z
lug'atini oldi (`src/i18n.ts`). Ulashishga arziydigan narsa — **qoidalar**
(xizmat haqi, markirovka, soat), va ular o'zgarishsiz import qilinadi. Faqat
o'sha yerda ekranlar bo'lgan matn ulashilgan ma'no emas.

⚠️ **Ikonkalar oila yo'lidan import qilinadi**, paket indeksidan emas:
`@expo/vector-icons` indeksi o'zi tashiydigan **har bir** to'plamni qayta
eksport qiladi, va har birining glif jadvali bor. Bitta nomni indeksdan olish
hammasini bundle'ga tortadi — **2.0 MB / 688 modul**, `@expo/vector-icons/Feather`
dan esa **1.6 MB / 634 modul**. Bitta qator, 400 KB.

⚠️ JS bundle bularning ichida **1.6 MB** — ya'ni 68 MB ning sababi bizning
kodimiz emas, native qatlam. Bizning kodimizni optimallashtirish bu raqamni
deyarli o'zgartirmaydi; nima o'zgartirishini bilib turish esa keyingi safar
noto'g'ri joyni qidirmaslikni anglatadi.

Bunga kirgani: RN yadrosi, Expo modullari, `expo-secure-store`, va ulashilgan
`lib/api.ts` + `lib/types.ts` + qoidalar. **Kirmagani**: uch tilli lug'at
(~12 000 qator) — u hali import qilinmagan, va import qilinganda faqat
kerakligi kiradi.

⚠️ **Bu APK hajmi emas.** APK ustiga RN runtime va native kutubxonalar
qo'shiladi; haqiqiy raqam EAS build'dan keyin, arzon Android telefonda
o'lchanadi.

## Menyu: uch ko'rinish, qidiruv va tillar

⚠️ **Nomlar restoranning o'z matni**, va ularning uchta versiyasi bor.
`lib/i18n/content.ts` (`contentName`) — saytning qoidasi, o'zgarishsiz import
qilingan: o'zbekcha asos, tarjima bo'lmasa unga qaytadi. Bu **import qilishga
arziydigan qoida**, panel lug'ati esa arzimagan matn edi — ikkalasi bir
faylning ikki xil qismi, va farqi o'lchangan.

- ⚠️ **Qidiruv kategoriyadan o'tadi, ko'rish esa o'tmaydi.** Nom bo'yicha
  qidirayotgan ofitsiant mehmonga javob berayapti va taom qaysi bo'limda
  ekanini bilmaydi; tanlangan kategoriya bilan cheklash taomni **aynan uni
  so'ragan odamdan** yashirardi, va bu "taom yo'q" bo'lib o'qiladi.
- ⚠️ **Tarjima qilingan nom bo'yicha qidiriladi**: ruscha o'qiydigan ofitsiant
  ko'rgan narsasini yozadi. Asos ham qidiriladi, ya'ni tarjimasiz nom ham
  topiladi.
- ⚠️ **Kategoriya chizig'iga qat'iy balandlik** berildi: ustun ichidagi chiplar
  qatori yonidagi ro'yxat o'sishi bilan **siqilib yo'qolardi** — ya'ni eng ko'p
  taomli kategoriyalarda aynan ularning nomini aytadigan chiziq g'oyib bo'lardi.
- **Uch ko'rinish** (`list` / `cards` / `photos`), telefonda saqlanadi: rasmsiz
  qahvaxona ro'yxatni, suratga olingan menyu esa rasmni xohlaydi. Bitta tugma
  bilan aylanadi — uchta doimiy boshqaruv qidiruv yonida menyu emas, bosiladigan
  narsalar qatori bo'lardi.
- ⚠️ **`FlatList`, `ScrollView` emas** — `ScrollView` har bir bolani render
  qiladi, va rasmli ikki yuz taom arzon Android'da aynan shunday qotadi.
  FlashList tezroq, lekin u **native bog'liqlik**: yangi build va noto'g'ri
  bo'lishi mumkin bo'lgan yangi narsa. U haqiqiy menyu o'lchanib, yetarli
  bo'lmagani ko'ringanda qo'shiladi.
- ⚠️ Rasm **hech qachon to'liq o'lchamda emas** — `imageUrl(path, 300)`.
  Rasmsiz taom **nomlangan bo'shliq** oladi: menyusining yarmini suratga olgan
  restoranda qolgan yarmi buzuqdek ko'rinmasligi kerak.

## Smena: ofitsiant ishining to'liq doirasi

Ilova endi brauzerdagi ofitsiant paneli qila oladigan narsalarni qiladi.

- ⚠️ **Qatorni tuzatish** (son, izoh, olib tashlash). Busiz ilova **o'zi tuzata
  olmaydigan xato yarata olardi** — noto'g'ri bosilgan taom uchun kassaga
  borish kerak edi, ya'ni ilova ishni **qo'shardi**.
- ⚠️ **Yuborilgan va yuborilmagan — ikki xil amal**, va ekran qaysiligini
  aytadi. Oshxona ko'rmagan qator — tuzatilayotgan xato, tekin. Ko'rgandan
  keyin ovqat pishirilgan, masalliqqa pul ketgan, va uni hisobdan chiqarish —
  **chiqim**: server sabab so'raydi va menejer kodini so'rashi mumkin.
- **Hisob (precheck)** — ofitsiant ishining tugash nuqtasi. ⚠️ **Faqat hammasi
  yuborilgandan keyin**: yuborilmagan taom turganda chop etilgan hisob —
  **noto'g'ri bo'lishi aniq** hisob, va u mehmon qo'liga allaqachon berilgan.
  ⚠️ `queued: 0` xato emas va shundayligicha aytiladi — hech bir printer
  olmadi, va halol keyingi qadam kassa, qayta urinish emas.
- **Oflayn** — `expo-sqlite`, seam ortidagi **uchinchi** dvigatel. ⚠️ Ilgari
  navbat IndexedDB qidirardi, telefonda topmasdi, va `available()` false
  qaytarardi: ilova butunlay normal ko'rinib, **wifi uzilishi bilan sotishni
  to'xtatardi**. Restoranda bu haftada bir necha marta.
- **Davomat** — smena telefondan ochiladi va yopiladi. ⚠️ Server joylashuvni
  talab qiladi (`geofenceBlocked`), GPS esa **telefonda**; ilgari ofitsiant
  smenani boshqa ekrandan ochib, keyin telefonda ishlardi. Ruxsat **tugma
  bosilganda** so'raladi: birinchi ekrandagi so'rov ilova nima uchunligi
  ma'lum bo'lishidan oldin beriladi. Aniqlik **Balanced** — filial radiusi
  50 m, telefon GPS'i ochiq havoda 10–30 m, ya'ni eng yuqori aniqlik
  javobni o'zgartirmaydi va sarflagan soniyalari eshik oldida turgan odamniki.
- **Mehmonlar soni, chekni bo'lish, ko'chirish, birlashtirish** — bitta
  varaqda. ⚠️ **Qatorlar tanlanadi, "yarmi" emas**: mehmon **o'zi yegani**
  uchun to'laydi, jamini teng bo'lish esa ekranda bir xil ko'rinadigan va
  stolda noto'g'ri boshqa narsa.

## Bildirishnomalar

⚠️ **Ofitsiant qarab bilolmaydigan yagona narsa.** Qolgan hamma narsa — o'zi
ochadigan ekran; ovqatning tayyor bo'lishi esa binoning boshqa qismida
sodir bo'ladi, va muqobillari: qo'ng'iroq, baqirish, yoki borib qarash.

- Oshxona "Tayyor" bosganda server chekning **ofitsiantiga** yuboradi
  (`check.serverId`). ⚠️ **Filialga emas**: umumiy xabar — xodimlarni
  bildirishnomani o'qimay surib tashlashga o'rgatish, va keyin muhimi ham
  ular bilan birga ketadi.
- Yo'l: `expo-notifications` → Expo relay → FCM/APNs. ⚠️ Bizda **hech qanday
  sertifikat yo'q**, ya'ni jimgina muddati o'tadigan narsa ham yo'q.
- ⚠️ **Ruxsat kirgandan keyin so'raladi**, ochilishda emas: birinchi ekrandagi
  so'rov ilova nima uchunligi ma'lum bo'lishidan oldin beriladi, va
  tushunilmagan savolning javobi "yo'q" — iOS'da esa bu deyarli qaytarib
  bo'lmaydi.
- ⚠️ **Rad etish — haqiqiy javob**: ilova ishlashda davom etadi.
- ⚠️ **Chiqishda token o'chiriladi**, va bu tozalik emas: qolib ketgan token
  ertangi stollarni uyiga ketgan odamga yuboradi, va u buni o'z tomonidan
  o'chira olmaydi.
- ⚠️ Server faqat **`DeviceNotRegistered`** ni doimiy deb biladi va o'sha
  tokenni o'chiradi. Tezlik chegarasi bizning muammomiz, va uning ustidan
  token o'chirish ishlab turgan telefonni jimgina obunadan chiqarardi.

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

## ⚠️ Internetsiz ochilganda: kirish ekrani emas

Ilova ochilganda serverdan «bu kim?» deb so'raydi. So'rov **umuman
yetib bormasa** javob ilgari «chiqib ketgan» bo'lardi — ya'ni podvalda, o'lik
Wi-Fi da yoki interneti tugagan telefonda ofitsiant parol maydonini ko'rardi:
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
