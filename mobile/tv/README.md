# Keel TV — zaldagi ekran (Android TV)

Expo (SDK 57, RN 0.86, React 19). Restoran zalidagi televizor: kontent
(video/rasm) va fastfood uchun buyurtma tablosi.

**Hozirgi bosqich (1):** ulash, yurak urishi, paneldan uzish. Kontent va tablo —
keyingi bosqichlarda.

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
2. Ekranda **6 belgilik kod** chiqadi, har 10 soniyada yangilanadi.
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
