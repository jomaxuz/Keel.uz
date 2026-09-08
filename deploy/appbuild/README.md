# Restoran ilovasini build qilish

Bitta kod bazasi (`mobile/guest-android`), har restoranga alohida build. Farq
faqat `app/brand.properties` va ikkita rasmda — qolgan hamma narsa bir xil.

## Image

```bash
docker build -t keel-appbuild:latest deploy/appbuild
```

⚠️ **Host'ga JDK yoki Android SDK o'rnatilmaydi.** O'sha mashina har bir
tenantning konteynerini tutib turadi; yonida o'rnatilgan toolchain bir `apt
upgrade` dan keyin hech kim takrorlay olmaydigan build xatosiga aylanadi.

## Bitta build

```bash
docker run --rm \
  -v /opt/keel:/opt/keel \
  -v /opt/keel/appkeys:/opt/keel/appkeys \
  -v /opt/keel/appbuilds:/opt/keel/appbuilds \
  -v keel-gradle-cache:/root/.gradle \
  -e KEEL_MAPS_KEY -e KEEL_FB_PROJECT_ID -e KEEL_FB_API_KEY \
  -e KEEL_FB_SENDER_ID -e KEEL_FB_APP_ID \
  keel-appbuild:latest \
  /opt/keel/deploy/appbuild/build.sh b5somsa apk
```

⚠️ **`keel-gradle-cache` volume shart**: usiz har build Gradle distributivini va
har bir bog'liqlikni qaytadan yuklaydi — bu o'n daqiqa va bir necha yuz megabayt,
har safar.

⚠️ **Bir vaqtda bitta build.** Gradle va Kotlin kompilyatori bu mashinada
tenantlar bilan yonma-yon turadi; navbatni konsol boshqaradi
(`/opt/keel/.appbuild.lock`, **absolut yo'l** — `$HOME` dagi qulf serverni emas,
foydalanuvchini qulflaydi).

## Artefakt bir marta yashaydi

Konsol faylni **yuklab olingan zahoti o'chiradi** — har build 2,5 MB, va hech
kim tozalamaydigan papka har mijoz uchun ishlaydigan mashinada turadi. Yozuvi
esa qoladi: versiya, SHA-256 va ikkala ism. «Do'konda qaysi versiya turibdi?»
degan savol oylar keyin beriladi, va fayl tizimi unga javob bera olmaydi.

⚠️ **Fayl uzatish tugagandan keyin o'chiriladi, boshlanganda emas.** Birinchi
baytda o'chirish uzilgan ulanishni yana to'qqiz daqiqalik build'ga aylantiradi,
va ikkinchi urinish hech nima topmaydi.

## Xarita kaliti — nega alohida maydon

Saytdagi `mapGoogleKey` — **brauzer** kaliti: uni Google konsolidagi **domen**
ro'yxati himoya qiladi, va shunday cheklangan kalitni Android SDK **rad
etadi**. Rad etish jim: kulrang to'r, hech qayerda xato yo'q, va mehmon
manzilini kirita olmaydi.

Shuning uchun panelda alohida maydon: **Sozlamalar → Xarita → «Android ilova
uchun xarita kaliti»**. U paket nomi va sertifikat izi bo'yicha cheklanadi.

⚠️ Ilova sayt qaysi provayderda bo'lishidan qat'i nazar **Google** SDK bilan
quriladi: uchala SDK'ni solib qo'yish har mehmonning yuklab olishiga o'nlab
megabayt qo'shadi — bir marta qilingandan keyin hech qachon o'zgarmaydigan
tanlov uchun.

## Firebase: har restoranga bitta ilova

FCM registratsiya tokeni Firebase **app id** ga bog'lanadi, va SDK ro'yxatdan
o'tayotganda paket nomini ham yuboradi. Bir restoranning ilovasini boshqasining
id'si bilan ishga tushirish **qo'llab-quvvatlanmaydi**: `getToken()` baribir
muvaffaqiyatli qaytadi va yuborilgan bildirishnoma jimgina hech qayerga
bormaydi — ikkala tomonda ham xato yo'q.

Shuning uchun har restoranga bir marta:

1. Firebase console → **Add app → Android**
2. Paket nomi: `uz.keel.app.<slug>`
3. Chiqqan `mobilesdk_app_id` ni konsolning «Android ilova» bo'limiga qo'ying

`google-services.json` yuklab olish shart emas — bizga faqat o'sha bitta satr
kerak, va ilova Firebase'ni **kodda** sozlaydi (plagin va fayl yo'q).

⚠️ Bo'sh qoldirilsa ilova bildirishnomasiz quriladi. Bu xato emas: ishlamaydigan
yagona narsa — hech kim sozlamagan narsa.

## Imzo kalitlari

`/opt/keel/appkeys/<slug>/release.jks` — **bir marta yaratiladi va hech qachon
almashtirilmaydi**. Yo'qolsa o'sha restoranning ilovasini boshqa hech kim
yangilay olmaydi: na biz, na restoran, na Google.

⚠️ Zaxira nusxasi **shu mashinadan tashqarida** turishi shart. Skript yangi kalit
yaratganda buni har safar ekranga yozadi.

## Nima qayerdan olinadi

| Nima | Qayerdan |
|---|---|
| Nom, logo, aksent rang | tenantning o'z `GET /restaurant` javobi |
| `applicationId` | slug'dan (`uz.keel.app.<slug>`) — **nomdan emas**: restoran nomini o'zgartiradi, id esa o'zgara olmaydi |
| Xarita kaliti | restoranning **o'z** `mapAndroidKey` i (Sozlamalar → Xarita); bo'sh bo'lsa `KEEL_MAPS_KEY` ga tushadi |
| Firebase loyihasi | muhit o'zgaruvchilari (`APP_FIREBASE_*`) — Keel'niki, hammasiga umumiy |
| Firebase **app id** | tenantning `androidAppId` maydoni — **har restoranga alohida**, konsolda qo'lda kiritiladi |
| Versiya | `KEEL_APP_VERSION_CODE` / `_NAME` |
