# Keel Owner — native (Kotlin + Jetpack Compose)

Ega ilovasi. Bu ilova **to'g'ridan-to'g'ri native yozilgan** — qolgan to'rttasidan
farqi shu: waiter, courier, team va TV avval Expo'da bo'lgan va ko'chirilgan,
buning esa Expo nusxasi (`mobile/owner`) skelet bo'lib qolgan.

`applicationId` — `uz.keel.owner`, dizayn `mobile/android-design` dan (beshala
ilova bo'lishadi).

---

## ⚠️ Bu panel emas, va bu asosiy qaror

Panel — odam **o'tirib qaror qabul qiladigan** joy: menyu, narx, grafik,
kampaniya. Ular klaviaturada qoladi. Ega telefonni **kuzatish va javob berish**
uchun ochadi: bugun qanday ketyapti, hozir nima so'ralyapti, bu buyurtmani
qabul qilaymi, oltinchi stoldan nega 400 000 yechilgan. Bu yerdagi har bir ekran
shu savollarning bittasiga uch soniyada javob beradi.

Shundan kelib chiqadi: ekran qo'shishdan oldin savol «bu panelda bormi?» emas,
**«buni telefonda ushlab turgan odam hozir hal qila oladimi?»**.

## Ekranlar

| Tab | Nima uchun ochiladi |
|---|---|
| **Bugun** | Tushum, buyurtma soni, o'rtacha chek, kechagi bilan farqi. Filial linzasi. |
| **Diqqat** | Katta chegirma, chek bekor qilingan, kassada kamomad — javob talab qiladigan narsalar. |
| **Buyurtmalar** | Kelayotgan buyurtmalar; qabul qilish/rad etish. |
| **Fikrlar** | Mehmonlar bahosi va izohi. |
| **Hisobotlar** | Davr bo'yicha raqamlar. |
| **Sozlamalar** | Til, tema, filial, bildirishnoma holati, chiqish. Yordam chati shu yerdan. |

Tab ustida ochiladigan ekranlar (bitta daraja, `MainActivity` → `Overlay`):

| Ekran | Qayerdan | Nima uchun |
|---|---|---|
| **Chek** | Diqqat → pul bilan bog'liq hodisa | «Oltinchi stoldan nega 400 000 yechilgan» — qatorlar, kim olib tashlagani, chegirmani kim qo'ygani va kim tasdiqlagani, qaytarish. |
| **Pul qayerda** | Hisobot → yuqoridagi qator | Naqd / bankda / yo'lda — uchta jami, **hech qachon bitta**. Bank qoldig'i shu yerdan yoziladi. |
| **Yordam** | Sozlamalar | Bilim bazasi va chat. |

⚠️ **Hodisani faqat `kind` sotuv haqida bo'lsa ochish mumkin** (`opensCheck`):
`refId` kamomadda kassa smenasini, texkartada taomni bildiradi. Faqat id bor-yo'qligiga
qarash ega eng xavotirlangan qatorni «topilmadi» bilan ochardi. Bo'sh id esa
`"000…0"` bo'lib keladi — `hasId()`.

⚠️ **Chekda olib tashlangan qatorlar o'z joyida, ustidan chizilgan holda turadi.**
Iz qoldirmaydigan void — restorandan pul olib chiqishning eng eski usuli.
Qaytarish va qayta chop etish tugmasi **yo'q**: ikkalasi ham ega turmagan binoda
pul yoki qog'oz qimirlatadi.

⚠️ **Pul qayerda: telefondan faqat bank qoldig'i yoziladi**, chunki uni aynan
telefonda — bank ilovasida — o'qiydi. Inkassatsiya sumka qo'lda turganda kassada
imzolanadi, limit esa shartnomadan bir marta kiritiladi — ikkalasi panelda.
Server joylarning birinchi to'rttasini o'zbekcha nomlaydi, ilova ularni `kind`
bo'yicha o'z tilida yozadi.

⚠️ **Vaqt `OffsetDateTime` bilan o'qiladi** (`localTime`): chek endpointi vaqtni
`+05:00` bilan yuboradi, Mongo'dan to'g'ridan-to'g'ri kelgani `Z` bilan. Eski
Android'dagi `Instant.parse` birinchisini qabul qilmaydi, va o'qilmagan vaqt xato
emas — bo'sh joy.

⚠️ **Filiallar sessiya bilan olinadi**, har ekranda emas: linza to'rtta ekran
ustida turadi, va menejerni (serverda bitta filialga qisqartiriladi) boshqasini
tanlay oladigandek ko'rsatadigan tanlagich yolg'on bo'lardi.

⚠️ **Filial ro'yxati yuklanmasa ham sessiya ochiladi**: linza — raqamlar ustidagi
qulaylik, va uni deb odamni ilovadan qulflab qo'yish noto'g'ri.

---

## Nima ulashilgan, nima ikkinchi nusxa

- **Ulashilgan** (`../android-design`): ranglar, shisha, tugmalar, tab paneli,
  `TokenStore`, `ServerAddress`, `money`/`qty`, `formatDuration`, `StatusColor`,
  til enumi. ⚠️ Bu yerga nusxalash — ajralishning boshlanishi.
- **Ikkinchi nusxa** (shu yerda, testi bilan): `data/Models.kt` — panel
  tiplarining qo'lda yozilgan nusxasi, va `data/HelpSearch.kt` — bilim
  bazasining saralash qoidasi.

⚠️ **`data/Models.kt` dagi har bir maydon nomi jimgina buziladi.** kotlinx
topa olmagan kalitga standart qiymat qo'yadi, ya'ni noto'g'ri nom xato emas —
ega qaraydigan ekrandagi **ishonchli nol**. Waiter buildida shunday uchta xato
bir vaqtda chiqqan: eng ko'p sotilgan taomlarning narxi nol bo'lgan, xarid
ro'yxati doimo bo'sh, buyurtmalar ro'yxati esa umuman ochilmagan. Yangi maydon
qo'shsangiz `WireShapeTest` ga Go handler'idan **ko'chirilgan** JSON bilan qator
qo'shing.

---

## Bildirishnomalar

Kanal `owner` (`push.OwnerChannel`), FCM to'g'ridan-to'g'ri — Expo relay orqali
emas; server token shakliga qarab yo'naltiradi (`internal/push/send.go`).

⚠️ **Bu kanalda keladigan narsa — pul va uni kim qo'zg'atgani**: katta chegirma,
chek chop etilgandan keyingi void, yashikdagi kamomad. Kanal aynan shu sababdan
alohida: oshxona chimini o'chirgan ega buni ham o'chirib qo'ymasin.

⚠️ **Xatoning sababi Sozlamalar ekranida yoziladi** (`PushRegistration.detail`,
tarjima qilinmaydi): «bu buildda push kredensiali yo'q», «server endpointdan
eski» va «telefon internetsiz» — uchalasi ham telefondan tuzatilmaydi, va
ularsiz uchalasi bir xil «Ro'yxatdan o'tmadi» bo'lib ko'rinadi.

---

## Build

```bash
export ANDROID_HOME=$HOME/Android/Sdk
export JAVA_HOME=/usr/lib/jvm/java-17-openjdk-amd64
./gradlew :app:testDebugUnitTest      # wire shakli + yordam saralashi (15 ta)
./gradlew :app:assembleDebug
$ANDROID_HOME/platform-tools/adb install -r app/build/outputs/apk/debug/app-debug.apk
```

Release imzosi: `~/keys/keel-owner.properties` (yoki
`KEEL_KEYSTORE_PROPERTIES`). ⚠️ Fayl bo'lmasa build **imzosiz** chiqadi —
o'rnatib bo'lmaydi, ya'ni xato baland ovozda. Release APK **2.4 MB**.

⚠️ **Kalit alohida** (`keel-owner`): imzo kalitini almashtirib bo'lmaydi, va
bitta kalit beshta ilovani bitta portlash radiusiga qo'yadi.

---

## Hali yo'q

1. ⏳ **Telefonda sinalmagan.** APK quriladi, 11 ta test o'tadi; ulangan
   qurilma yo'q. Birinchi qurilmada: qorong'i tema, filial linzasi,
   bildirishnoma bosilganda kerakli tabga tushishi.
2. ⏳ Play uchun `.aab` va do'kon sahifasi.
