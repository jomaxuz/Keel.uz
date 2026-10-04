# Keel — bitta ilova: ofitsiant, ega, kuryer, xodim (Kotlin + Compose)

To'rtta native ilova (`waiter-android`, `owner-android`, `courier-android`,
`team-android`) **bitta** ilovada. Odam bitta login bilan kiradi, server
qaysi hisoblar ochilishini aytadi, va unga faqat o'ziga tegishli ish joylari
ko'rsatiladi. Televizor (`tv-android`) va mehmon ilovasi (`guest-android`)
alohida qoladi: biri devordagi ekran, ikkinchisi restoranning o'z ilovasi.

⚠️ **Eski ilovalar o'chirilmagan va ishlashda davom etadi.** Keel o'z
`applicationId` si bilan (`uz.keel.app`) ularning **yonida** o'rnatiladi —
restoran telefonlarni birma-bir ko'chiradi, va keyingi telefondagi eski ilova
ko'chish davomida ishlab turadi.

---

## Qanday ishlaydi

```
Kirish (restoran + login + parol)
   │  POST /api/v1/app/login  → shu login ochadigan BARCHA hisoblar
   ▼
Hisoblar: admin (ega/menejer) · staff (xodim) · courier — har turdan bittadan
   ▼
Ish joylari:  Boshqaruv (admin) · Zal (staff, waiter ruxsati) ·
              Xodim (staff) · Yetkazish (courier)
   ▼
Bitta ish joyi bo'lsa — to'g'ri unga. Bir nechta bo'lsa — oxirgi ochilganiga;
birinchi marta — «Bugun qayerda ishlaymiz?» ekraniga.
```

- **Rol tanlash so'ralmaydi.** Odam «ofitsiantman» yoki «egaman» deb
  tanlamaydi — server `/app/login` da uchta jadvalga ham qaraydi. Eski
  ilovalarda egasi kuryer ilovasiga to'g'ri parolni yozib «noto'g'ri» degan
  javob olardi; bu ilova aynan shu xatoni yo'qotish uchun.
- **Almashtirish** — har rolning Sozlamalar ekranining eng tepasidagi hisob
  kartochkasi («Almashtirish»). U yerdan: boshqa ish joyi, hisob qo'shish
  (masalan, ega kuryer loginini ham qo'shadi), bitta hisobdan chiqish,
  boshqa restoran.
- **Zal faqat ofitsiantga**, serverning o'z qoidasi bilan
  (`models.Staff.Can(PermWaiter)`: rol ruxsatlari, bo'lmasa eski bayroqlar,
  kassir = ofitsiant). **Xodim bo'limi** — qolgan hammaga, ofitsiantga esa
  faqat unda zalda yo'q narsa bo'lsa (bozorlik, sanoq, ombor, markirovka).
  Aks holda bir xil «soatlarim» ikki eshik ortida turardi.

## Qurilma (device) ID — eski ilovalarga xalaqit bermaydi

Server bitta hisobni **ilova kaliti bo'yicha** bitta telefonga bog'laydi
(`login_device`, `(app, deviceId)` unique). Keel o'z kalitlarini ishlatadi:

| Hisob turi | Keel kaliti | Eski ilova kaliti |
|---|---|---|
| admin | `keel-owner` | `owner` |
| staff | `keel-staff` (zal + xodim bo'limi **bitta**) | `waiter`, `team` |
| courier | `keel-courier` | `courier` |

- ⚠️ Eski kalitni ishlatsak, ikkala ilova o'rnatilgan telefonda (ko'chish
  davridagi oddiy holat) ular har kirishda bir-birining bog'lanishini
  surib chiqarardi.
- ⚠️ Kalit **hisob turi bo'yicha**, butun Keel uchun bitta emas: ega ham,
  kuryer ham bo'lgan odam ikkala hisobni bitta telefonda ushlaydi.
- ID — o'rnatishning o'zi (`TokenStore.deviceId`). Qayta o'rnatish yangisini
  yaratadi; paneldagi «telefonni bo'shatish» tugmasi shu uchun.
- Panelda «Keel · boshqaruv / xodim / kuryer» deb ko'rinadi.

## Tuzilma

```
app/src/main/java/uz/keel/
├── app/            # Keel'ning o'zi
│   ├── KeelApp.kt        # Application: tokenlar, hisoblar, 4 kanal, rollar dangasa
│   ├── MainActivity.kt   # bitta activity, bildirishnoma marshruti
│   ├── KeelLook.kt       # til + tema — 4 rol uchun BITTA holat
│   ├── RoleExit.kt       # rol chiqish yo'llari (chiqish, muddati o'tdi, almashtirish)
│   ├── data/             # Accounts (hisoblar ro'yxati), ShellApi (/app/login, push)
│   ├── push/             # PushHub (barcha hisoblarga bitta token), KeelMessagingService
│   ├── i18n/Words.kt     # faqat Keel'ning o'z so'zlari, uch tilda
│   └── ui/               # Kirish, ish joylari (Hub), hisob kartochkasi
├── waiter/  owner/  courier/  team/
│   # eski ilovalardan ko'chirilgan; farqi: MainActivity → <Rol>Entry,
│   # Application → <Rol>Graph, login ekrani yo'q, push Keel'niki
```

## Nimalar o'zgardi (rollarda)

- `KeelXApp : Application` → `XGraph` — rol **birinchi ochilganda** yaratiladi
  (`KeelApp` dagi `lazy`). Kuryer bo'lmagan odamning telefonida kuryer
  qismi umuman ishga tushmaydi. ⚠️ Istisno — ofitsiant: uning oflayn
  navbatida ilova yopilishidan oldingi taomlar qolgan bo'lishi mumkin,
  shuning uchun zal hisobi bor telefonda u ishga tushishda yaratiladi.
- `MainActivity` → `XEntry(graph, push, exit, pending…)`. Login va
  «restoran manzili» ekranlari olib tashlangan; token rad etilsa rol
  `exit.expired()` ni chaqiradi va Keel uni qayta kirishga olib boradi.
- `SessionViewModel` endi **token bo'yicha kalitlangan** — activity rol
  ekranidan uzoq yashaydi, kalitsiz model yangi kirgan odamga eskisining
  sessiyasini berardi.
- `Prefs` til va temani `KeelLook` dan oladi: egasining sozlamasida tanlangan
  til darhol zal tili ham bo'ladi.
- Sozlamalar ekranlariga `top` slot qo'shildi — u yerda hisob kartochkasi.

## Bildirishnomalar

- ⚠️ **Bitta `FirebaseMessagingService`** (`KeelMessagingService`) — Firebase
  ilovaga faqat bittasini chaqiradi.
- ⚠️ **Token barcha hisoblarga birdaniga yoziladi** (`PushHub.sync`), rol
  ochilganda emas: kun bo'yi zalda ishlagan ega ham loss alertni olishi kerak.
- ⚠️ **Bitta hisobdan chiqish tokenni o'chirmaydi** — u telefondagi boshqa
  hisoblarniki ham. Faqat o'sha hisobning qatori o'chiriladi; token oxirgi
  hisob bilan ketadi.
- Server endi `data.channel` ni ham yuboradi (`internal/push/fcm.go`) — fonda
  tizim chizgan bildirishnomaga bosilganda ham qaysi ish joyi ochilishi
  aniq bo'lsin. `checkId` bo'lsa — har doim zal va o'sha stol.
- To'rt kanal (`kitchen`, `delivery`, `owner`, `team`) ishga tushishda
  yaratiladi: oshxona ovozini o'chirgan odam egasining ogohlantirishini
  o'chirmagan bo'lishi kerak.

## ⚠️ Firebase: bir qadam konsolda

`keel-7f31a` loyihasida hozircha faqat to'rtta eski ilova bor. Push ishlashi
uchun:

1. Firebase konsol → Project settings → **Add app → Android** →
   `uz.keel.app`.
2. Yangi `google-services.json` ni `mobile/keel-android/app/` ga qo'ying.

Fayl bo'lmasa (yoki unda `uz.keel.app` bo'lmasa) build **yiqilmaydi** —
google-services plagini qo'llanmaydi, ilova ishlaydi, sozlamalarda esa push
«sozlanmagan» deb ko'rinadi. Server tomonda hech narsa kerak emas: FCM v1
yuboruvchisi loyiha bo'yicha ishlaydi.

## Build

```bash
cd mobile/keel-android
./gradlew assembleRelease   # imzo: ~/keys/keel-app.properties (yoki KEEL_KEYSTORE_PROPERTIES)
```

Versiya — repozitoriy ildizidagi `VERSION` (boshqa ilovalar bilan bitta raqam).

⚠️ Bu muhitda Android SDK yo'q (`dl.google.com` yopiq), shuning uchun kod
Compose Desktop + stublar bilan **JVM'da tipcheck** qilingan — butun ilova
(~23 ming qator) kompilyatsiya bo'ldi. Haqiqiy `assembleRelease` va
qurilmada sinov hali qilinmagan.
