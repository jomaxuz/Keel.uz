# Keel TV — zaldagi ekran (Android TV)

> ⚠️ **Bu ilova almashtirildi: `mobile/tv-android` (Kotlin + Compose).**
> Yangisi shu `applicationId` ni (`uz.keel.tv`) oladi, ya'ni televizorda
> eskisining o'rniga o'rnatiladi. Bu papka **hozircha saqlanadi**, chunki uning
> qarorlari va ular ortidagi xatolar tarixi shu yerda yozilgan — yangisi ularni
> ko'chirdi, lekin sabablari birinchi marta shu faylda yozilgan.

Expo (SDK 57, RN 0.86, React 19). Restoran zalidagi televizor: kontent
(video/rasm) va fastfood uchun buyurtma tablosi.

**Hozirgi bosqich (3):** ulash, yurak urishi, paneldan uzish, **kontent**
(rasm + video playlist, oflayn ishlaydi) va **buyurtma tablosi**.

## Kontent qanday ishlaydi

Panelda: **TV ekranlar → Kontent**. Ro'yxat **filialniki** — zaldagi hamma
televizor bir xil aylanmani o'ynatadi, farq faqat rejimda (`content` / `board` /
`split`).

1. Heartbeat (`/tv/me`) har daqiqada `contentVersion` olib keladi.
2. Versiya o'zgargan bo'lsa — `/tv/playlist` o'qiladi va **har fayl setga
   yuklab olinadi** (`expo-file-system`, `Paths.document/tv-content`).
3. O'ynash faylning **lokal nusxasidan** boradi: rasm — `Image`, video —
   `expo-video`.

⚠️ **Streaming emas, yuklab olish.** Devordagi ekran bir xil 40 MB klipni kun
bo'yi aylantiradi — kassa va ofitsiantlar telefoni turgan **o'sha** wifi orqali.
Va aynan shu narsa ekranni internetdan mustaqil qiladi: fayllar setda turgach,
router o'chsa ham aylanma to'xtamaydi.

⚠️ **Manifest `document` da, `cache` da emas** (`playlist.json`): tizim joy
tugaganda cache'ni tozalaydi, va tunda jimgina playlistini yo'qotgan televizor
ertalab ochilishda qora ekran bo'lib chiqadi.

⚠️ **Muddatni ilova o'zi hisoblaydi** (`playableNow`). Server `active: false`
ni filtrlaydi (u o'zi o'zgarmaydi), sanani esa yubormaydi — juma kunidan beri
oflayn ekran shanbada tugagan aksiyani tushirishi kerak. Devordagi eskirgan
taklif bo'sh ekrandan battar: mehmon uni kassada so'raydi.

⚠️ **Vaqt serverdan** (`src/clock.ts`). Arzon Android TV tarmoqsiz yuklanganda
1970 yilda keladi; `/tv/me` har daqiqada haqiqiy vaqtni aytadi, ilova farqni
saqlaydi. Sovuq yuklashda oxirgi ma'lum farq tiklanadi — bu **taxmin**, lekin
birinchi heartbeat uni tuzatadi, va u sotib oladigan narsa shu bir daqiqa.

⚠️ **Bitta player butun aylanmaga** (`player.replace`). Har slaydga yangi player
arzon apparatda dekoderni oqizadi, alomati esa: bir soat yaxshi ishlaydi, keyin
rozetkadan sug'urilmaguncha hech nima ko'rsatmaydi.

## Tablo qanday ishlaydi

`board` — butun ekran (chapda "Tayyorlanmoqda", o'ngda "Tayyor"), `split` —
playlist ustida faqat "Tayyor" chizig'i. `content` ekran tabloni **umuman
so'ramaydi**.

⚠️ **Faqat raqamlar.** Xonadagi hamma o'qiydi, jumladan buyurtmasi tabloda
bo'lmagan odamlar ham: ism yozadigan tablo — devordagi mijozlar ro'yxati.

⚠️ **Aloqa uzilsa tablo jim bo'ladi** (2 daqiqa), playlist esa o'ynayveradi —
qoida ataylab **teskari**. Aylanma uzilgan aloqada ham restoranning o'z
kontenti; tablo esa ovqat haqida da'vo qiladi, va eskirgan "tayyor" mehmonni
peshtaxtaga bekorga yuboradi.

⚠️ **Bo'sh tablo — bo'sh jadval emas**, filial nomi: sarlavhalari bor, ostida
hech nima yo'q ekran tushlik bilan kechki ovqat orasida soatlab "buzuq" bo'lib
turadi. `split` da esa chiziq umuman ko'rinmaydi — videoning ustidagi doimiy
bo'sh panel restoranning o'z ekranini yeydigan mebel.

## Nega alohida ilova, va nega u boshqacha yozilgan

Bu — **hech kim qo'lida ushlamaydigan** yagona ilova. Ofitsiant xatoni o'qiydi,
kassir tugmani qayta bosadi, kuryer ilovani qayta ochadi. Bu esa devorda,
mehmonlar oldida turadi va unga javob bera oladigan odam xonada emas. Shundan:

- **Xonaga hech qachon xato ko'rsatmaydi** — uzilgan internet jimlik bilan
  o'tadi, qirq kishi ovqatlanayotgan zalda qizil quti bilan emas;
- **Sozlangandan keyin hech qanday kiritish talab qilmaydi** — klaviatura yo'q,
  pult esa tortmada;
- **Kod bilan ulanadi**: ekran kodni **ko'rsatadi**, parol so'ramaydi;
- **Uzilganini biladi**: bir yillik tokenga ishonib qolmay, har daqiqada
  "hali ulanganmanmi?" deb so'raydi.

## Ulash oqimi

1. Ilova ochiladi → **restoran manzili** so'raladi (bir marta: `osh` — qolganini
   `lib/serverAddress` hal qiladi, telefon ilovalari va Windows kassa bilan bir
   xil qoida).
2. Ekranda **6 belgilik kod** chiqadi va **bir daqiqa** turadi (muddatni
   server aytadi, ilova o'z taymerini yuritmaydi).
3. Panelda: **TV ekranlar → Ekran qo'shish** → kod + filial + nom + rejim.
4. Ekran keyingi so'rovida tokenini oladi va ishga tushadi.

⚠️ **Kodni bilish yetarli emas**: holat so'rovi kod bilan emas, `pollSecret`
bilan javob oladi (u faqat shu televizorda). Aks holda zaldagi har kim kodni
o'qib, ilovadan tezroq so'rab, devorga atalgan tokenni olib ketardi.

## Nima uchun `react-native-tvos` emas

Oddiy `react-native` Android TV'da ishlaydi; TV forki asosan TV'ga xos fokus
komponentlari uchun kerak. Bizga ular hozircha kerak emas — fokus `onFocus` /
`onBlur` (oddiy View proplari) bilan boshqariladi. Fork esa butun paketni
almashtiradi va uch ilova bilan umumiy bo'lgan `frontend/src/lib` ni
qayta tekshirishni talab qiladi. Kerak bo'lganda qaytiladi.

## Manifest: `plugins/withAndroidTV.js`

⚠️ **Busiz APK o'rnatiladi va ko'rinmaydi.** Android TV bosh ekrani faqat
`LEANBACK_LAUNCHER` e'lon qilgan ilovani ko'rsatadi; oddiy `LAUNCHER` — telefon
uchun. O'rnatish "muvaffaqiyatli" deydi, ochadigan narsa esa yo'q.

Plagin qo'shadigan uchta narsa: `leanback` va `touchscreen` (ikkalasi ham
`required="false"` — aks holda ilova **telefonga** o'rnatilmaydi, ya'ni uni
ishlab chiqadigan mashinaga ham), banner (leanback bosh ekrani har ilovaga
banner chizadi) va `screenOrientation="landscape"`.

## Build va o'rnatish

```bash
npm install
npx tsc --noEmit                      # tekshirish
npx expo export --platform android    # bundle yig'ilishini ko'rish
eas build -p android --profile preview   # bulutda APK
```

**Lokal build** (EAS limitiga bog'liq emas, shu mashinada):

```bash
export JAVA_HOME=~/.jdks/jdk-17.0.13+11
export ANDROID_HOME=~/Android/Sdk
npx expo prebuild --platform android --no-install
cd android && echo "sdk.dir=$HOME/Android/Sdk" > local.properties
./gradlew assembleRelease     # app/build/outputs/apk/release/app-release.apk
```

SDK talablari: JDK 17, `platform-tools`, `platforms;android-36`,
`build-tools;36.0.0`, va ⚠️ **`cmake;3.22.1` bilan `ndk`** — busiz build
`Could not find Ninja on PATH` deb yiqiladi, va bu xato Ninja haqida emas,
o'rnatilmagan NDK haqida.

⚠️ **Lokal build debug kaliti bilan imzolanadi**, EAS esa o'zining kaliti
bilan. Ya'ni biridan ikkinchisiga o'tishda telefon/televizor "o'rnatib
bo'lmadi" deydi — eskisini **o'chirib** keyin o'rnatish kerak. Haqiqiy
tarqatishda bitta doimiy kalit tanlanadi (Play Store'da do'konning o'z
imzolashi bo'ladi).

O'rnatish (hozircha Play Store'siz):
1. **USB** — televizorda "Noma'lum manbalar" yoqiladi, fleshkadagi APK fayl
   menejer orqali ochiladi. ⚠️ Ba'zi televizorlarda fayl menejer yo'q yoki
   ishlab chiqaruvchi buni yopib qo'ygan.
2. **ADB (zaxira, ishonchliroq)** — televizorda ishlab chiquvchi rejimi, keyin
   noutbukdan tarmoq orqali `adb install`.

⚠️ **Samsung va LG'ga o'rnatib bo'lmaydi**: ular Android emas (Tizen va webOS).
Ular uchun keyinchalik alohida ilova.

## Hali yo'q (keyingi bosqichlar)

- Kontent: manifest, faylni yuklab olib qo'yish (checksum bilan), oflayn
  ko'rsatish, disk budjeti.
- Buyurtma tablosi (`preparing` / `tayyor`).
- ⚠️ **Yoqilganda o'zi ishga tushishi** (BOOT_COMPLETED) — bunga kichik native
  qism kerak, va Google TV'ning ba'zilarida tizim baribir o'z ekranini
  birinchi qo'yadi. Ya'ni bu **kafolat emas**, va shu holicha yozilgan.
