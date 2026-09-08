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
| Xarita kaliti, Firebase | muhit o'zgaruvchilari (Keel'niki, hammasiga umumiy; `FB_APP_ID` — har ilovaga o'ziniki) |
| Versiya | `KEEL_APP_VERSION_CODE` / `_NAME` |
