# Keel — Windows ilovasi (Wails)

Kassa va zal ekranlari bitta oynada, chek printeriga yeta oladigan yagona
mashinada. Qaror va sabablari: `docs/pos-reja.md` §2.

## Nega `backend/` moduli ichida

Go'ning `internal/` qoidasi: `restaurant-backend/internal/printer` va
`internal/receipt` ni **faqat shu modul ichidagi kod** import qila oladi, va
`replace` direktivasi buni aylanib o'tolmaydi. Alohida modul aynan shu ikki
paketga — butun ishning maqsadiga — yeta olmasdi.

## Nima qayerda quriladi

| | Qayerda | Nega |
|---|---|---|
| `.exe` | **Windows VM** (`wails build`) | Wails cross-compile'ni qo'llab-quvvatlamaydi |
| Go tomonini tekshirish | Linux ham bo'ladi | `GOOS=windows CGO_ENABLED=0 go build ./desktop` |

Ikkinchi qator kutilmagan, lekin haqiqiy: Wails v2 ning Windows tomoni sof Go
(`go-webview2`), ya'ni **cgo yo'q** — `internal/printer` dagi `winspool.drv`
lazy DLL tanlovi bilan bir sabab. Ya'ni Go tomonidagi xato VM'ni ochmasdan
topiladi; VM faqat frontend build'i, ikonka, manifest va haqiqiy ishga tushirish
uchun kerak.

## Build (VM'da)

```
wails doctor          # Go, npm, WebView2 — uchalasi
wails dev             # ishlab chiqish, jonli qayta yuklash bilan
wails build           # build/bin/keel.exe          — portativ fayl
wails build -nsis     # build/bin/keel-amd64-installer.exe — o'rnatuvchi
```

### Qaysi birini berish kerak

**Restoranga — o'rnatuvchi.** Sababi bitta va u hal qiluvchi: **toza Windows
10'da WebView2 Runtime bo'lmasligi mumkin, va usiz ilova umuman ochilmaydi.**
O'rnatuvchi uni o'zi yuklab olib qo'yadi (`wails_tools.nsh` — Wails'niki, biz
tegmaymiz). Portativ `.exe` esa shunchaki ochilmaydi va sababini aytmaydi.

Bundan tashqari: `Program Files` ga o'rnatiladi, "Dasturlar" ro'yxatida
ko'rinadi va o'chiriladi, Start menyu va ish stolida yorliq bo'ladi.

⚠️ **Windows bilan birga ishga tushadi** (Startup yorlig'i). Kassa — odam
ochishni tanlaydigan dastur emas, mashinaning o'zi shu uchun turibdi; tokdan
o'chib qayta yonganda monoblok kassaga o'zi qaytishi kerak, chunki yorliqni
bosishni biladigan odam ayni paytda peshtaxtada mehmon bilan. Kerak bo'lmasa
Startup papkasidan yorliqni o'chirish kifoya.

⚠️ **O'chirishda `%PROGRAMDATA%\Keel` qoldiriladi.** O'chirishlarning ko'pi —
tuzatilgan versiyani qayta o'rnatayotgan odam, va sozlamani o'chirish ikki
daqiqalik qayta o'rnatishni egadan panel parolini so'rashga aylantiradi.
Haqiqatan ishdan chiqarilayotgan mashina uchun javob boshqa: paneldan filial
kalitini almashtirish (`branch.TillVersion`), u o'sha filialning barcha
tokenlarini o'ldiradi.

### O'rnatuvchi nimasi bilan o'zimizniki

`build/windows/installer/project.nsi` — Wails shabloni asosida, quyidagilar
o'zgartirilgan:

- **Keel ikonkasi**, yon paneldagi rasm (`welcome.bmp`, 164×314) va yuqoridagi
  banner (`header.bmp`, 150×57). ⚠️ Ikkalasi ham **BMP**: NSIS ularni bitmap
  deb o'qiydi va PNG bo'lsa **hech nima demaydi** — ikonkadagi bilan bir tuzoq.
- **Butun matn o'zbekcha**, tugmalar bilan birga. NSIS'da o'zbek tili yo'q,
  shuning uchun ingliz "uyasi" ishlatilib har bir satr almashtirilgan.
  ⚠️ Shu sababli **faqat bitta til qoldirilgan**: yonига rus tilini qo'shsak,
  NSIS tizim tiliga qarab tanlaydi va rus Windows'ida (monobloklarning
  ko'pchiligida) bizning o'zbekcha matnimiz umuman ko'rinmaydi.
- **Papka tanlash sahifasi yo'q.** Monoblok sozlayotgan odamda kassa qayerda
  turishi haqida fikr yo'q, savol esa ikkita xato yo'l ochadi — biri yuklanishda
  ulanmaydigan tarmoq diski, u avtomatik ishga tushirish yorlig'ini o'lik
  havolaga aylantiradi.
- ⚠️ **O'rnatishdan oldin ishlab turgan kassa yopiladi** (`taskkill`). Windows
  ishlab turgan `.exe` ni almashtira olmaydi, NSIS esa buni o'rtada fayl xatosi
  qilib qaytaradi — peshtaxtada yarim o'rnatilgan kassa qoladi. Bu chekka holat
  emas: **birinchisidan keyingi har bir yangilanish** kassa ochiq turgan
  mashinaga keladi, chunki kassa doim ochiq.
- ⚠️ **Tugatish sahifasida "hozir ochish" belgisi ataylab yo'q.** O'rnatuvchi
  administrator huquqi bilan ishlaydi, ya'ni u ochgan dastur ham shunday
  ishlaydi — birinchi ishga tushish esa WebView2 ma'lumot papkasini yaratadi,
  va u Administratorniki bo'lib qolsa ertalab kassani ochgan oddiy
  foydalanuvchini rad etadi.

⚠️ **Imzo hali yo'q.** Imzosiz o'rnatuvchida Windows "Windows protected your
PC" ekranini portativ fayldagidan **kuchliroq** ko'rsatadi, chunki bu o'rnatishga
urinadi. `project.nsi` da `signtool` qatorlari tayyor turibdi, izohda.
`pos-reja.md` §9.

## Umumiy kod

Ekranlar **ko'chirilmaydi** — `frontend/src/app/kassa` va
`frontend/src/components/till` shu yerda qayta kompilyatsiya qilinadi
(`vite.config.ts` dagi `@` aliasi). Nusxa bir oyda ikkita kassa bo'lardi, va
xato haqidagi shikoyat doim ikkinchisiga tushardi.

Narxi — ikkita shim (`src/shims/`): umumiy kod `next/navigation` va
`next/headers` ni import qiladi. Butun uchinchi tomon yuzasi — `react`,
`react-dom`, `react-icons`.

## ⚠️ Chek baytlari bu yerda kodlanmaydi

Server `escpos.Options` ni filialning printer yozuvidan yig'adi (charset, cut,
drawer, feed) va tayyor baytlarni `print_job` navbatiga qo'yadi. Ilova ularni
**qayta kodlamaydi** — `cmd/fiscalagent` bosgan yo'ldan boradi: navbatdan ish
oladi, printerga yozadi, natijani qaytaradi. Ikkinchi kodlagich ikki xil chek
degani, va farqni qo'lida qog'oz ushlab turgan mehmon topadi.

## Sozlash: qo'lda emas, ekrandan

`.exe` birinchi ishga tushganda **ulanish ekrani** chiqadi. Ikki qadam:

1. **Restoran manzili** + ega/menejer logini. Faqat nom yozilsa yetarli —
   `osh` → `osh.keel.uz` (har tenant provisioning'da shu subdomenni oladi,
   `control/handlers/tenants.go`). O'z domeni bo'lsa to'liq yoziladi.
2. **Filial tanlanadi** → ilova qurilma kalitini oladi va sozlamani **o'zi
   yozadi**.

Shundan keyin ekran boshqa chiqmaydi; kassir faqat PIN teradi.

⚠️ **Parol saqlanmaydi.** Login faqat "bu odam shu mashinani bog'lashga
haqli" ekanini isbotlaydi. Saqlanadigan narsa — **filial qurilma kaliti**
(`role: "tilldevice"`, bir yil), va u paneldan bir bosishda bekor qilinadi
(`branch.TillVersion`). Monoblokda panel logini turishi — devorga yozib
qo'yilgan umumiy parolning yo'li.

⚠️ **Filial tanlash alohida qadam**, chunki noto'g'ri filial cheklarni boshqa
oshxonaga, sotuvni boshqa hisobotga yuboradi — va ekranda hech nima xato
ko'rinmaydi.

⚠️ **Nega agent kaliti emas.** `AdminFiscalAgentToken` **har chaqiruvda
almashtiradi**, ya'ni ilova uni so'rasa o'sha filialda ishlab turgan agentni
jimgina o'ldirardi. Bitta mashinaga ikki sir, va ikkinchisini olish birinchisini
buzadi — avtomatik ulanishni imkonsiz qilgan narsa shu edi. Endi agent
endpointlari qurilma kalitini ham qabul qiladi.

### Sozlama qayerda turadi

`%PROGRAMDATA%\Keel\till.json` — ilova yozadi, odam emas.

⚠️ `.exe` yonida **emas**: `Program Files` ichiga yozish uchun admin huquqi
kerak, ya'ni saqlash aynan shu ilova mo'ljallangan mashinalarda yiqilardi.
Eski, qo'lda yozilgan fayl hali ham **o'qiladi** (allaqachon sotayotgan kassa
buzilmasligi uchun), lekin unga qaytib yozilmaydi.

Log: `%PROGRAMDATA%\Keel\till.log` — sozlama bilan yonma-yon.
⚠️ GUI dasturda konsol yo'q, ya'ni bu yagona iz. `.exe` yonida **emas**:
o'rnatuvchi dasturni `Program Files` ga qo'yadi va kassirning hisobi u yerga
yoza olmaydi — log aynan o'zi kerak bo'lgan mashinalarda jimgina yo'q edi.

## Oyna

Framesiz va **maksimallashtirilgan** (`options.Maximised`), haqiqiy fullscreen
emas. `pos-reja.md` §2 ikkalasini talab qiladi — chromesiz to'liq ekran **va**
ilova qotganda chiqish yo'li — lekin haqiqiy fullscreen Alt+Tab va Win+D ni ham
olib qo'yadi, ya'ni yagona chiqish yo'li tok tugmasi bo'lib qoladi. Kassirga
ikkalasi bir xil ko'rinadi.

⚠️ **Sarlavha paneli yo'q, va bo'lmasligi kerak.** Kassa ekranining **o'z
paneli bor** (qulf tugmasi bilan), ya'ni ustiga qo'yilgan ikkinchi panel ham
takror, ham zararli: uning 36 px i sahifani oynadan baland qilib o'ng tomonda
scrollbar chiqaradi. Kassa ildizi allaqachon `h-dvh` va o'z ichida suriladi —
uning atrofiga o'ralgan har qanday narsa ortiqcha.

Chiqish yo'llari: **Alt+F4** va **Ctrl+Shift+Q**. Ular bir xil emas —
birinchisini Windows bajaradi va ilova javob bermay qolganda ham ishlaydi,
ikkinchisi esa JavaScript, ya'ni qotgan webview'ni qutqara olmaydi (u
boshqaruvini yo'qotgan ekran uchun). ⚠️ **Ikkalasi ham o'rnatish hujjatida
yozilishi shart**: ko'rinadigan yopish tugmasi yo'q monoblok tok tugmasi bilan
yopiladi.

## O'lcham: ekrandan o'lchanadi

Kassa dizayni ~**1280 px** kenglikka chizilgan, sotiladigan monobloklar esa
ko'pincha **1024×768**. Ilova ishga tushganda ekran kengligini o'lchaydi va
o'zini shunga moslaydi (1024 → `0.8`), 1 dan yuqoriga hech qachon chiqmaydi va
0.65 dan pastga tushmaydi.

⚠️ **Bu bezak emas edi.** 1:1 da joylashuv 1024 px ga sig'maydi va **toza
yiqilmaydi**: yozuvlar ustma-ust tushadi, pastki qator chetga siqiladi, va
brauzer har kadrda oynasidan katta sahifani qayta hisoblab qayta chizadi.
"Hamma narsa katta" va "hamma narsa sekin" — bitta muammo edi.

Qo'lda o'zgartirish kerak bo'lsa `%PROGRAMDATA%\Keel\till.json`:

```json
{ "zoom": 0.75 }
```

0.5–2 oralig'idan tashqarisi "tanlanmagan" deb o'qiladi va o'lchovga qaytadi.

⚠️ Windows'ning o'z masshtabini ham tekshiring (Sozlamalar → Tizim → Ekran).
125% qo'yilgan bo'lsa uni 100% ga qaytarish to'g'riroq — ikki marta
masshtablash matn chetlarini bulg'aydi.

## Scroll uzuq-uzuq bo'lsa

```json
{ "gpu": "off" }
```

⚠️ **Ikkalasi ham (`gpu`, `zoom`) o'zgargandan keyin ilova qayta ochilishi
shart** — ular oyna qurilayotganda o'qiladi.

⚠️ Qaysi tomon to'g'ri ekanini **drayver hal qiladi**, biz emas: ba'zi
integratsiyalangan chiplarda kompozitsiya scrollni silliq qiladi, boshqalarida
(odatda OEM'ning eski drayveri bilan) aynan u uzadi. Buni bu yerdan bilib
bo'lmaydi, ekran oldida turgan odam esa ikki qiymatni sinab ko'ra oladi.

## Ekran klaviaturasi

Monoblokda klaviatura yo'q, Windows esa desktop rejimida uni **o'zi
taklif qilmaydi**. Ilova ikki narsa qiladi: ishga tushganda
`EnableDesktopModeAutoInvoke` ni yoqadi (HKCU, admin huquqi kerak emas —
to'liq kuchga **keyingi kirishda** kiradi) va har matn maydoniga fokus
tushganda `TabTip.exe` ni ochadi.

⚠️ Fokus **`focusin` orqali** ushlanadi, har maydonga alohida emas: kassa —
o'nlab umumiy komponentdagi yuzlab boshqaruv, va biri unutilsa u kassir to'ldira
olmaydigan maydon bo'lib qoladi.

## Chek chiqarishning ikki yo'li

| Holat | Kim chiqaradi |
|---|---|
| Filialda printer **sozlangan** | Server `escpos.Options` ni yig'adi, baytlarni navbatga qo'yadi, **agent halqasi** chop etadi |
| Sozlanmagan, lekin mashinada printer bor | `PrintLines` binding'i — shu mashinaning printeri, o'z sozlamasi bilan |

Ikkinchisi `lib/print.ts` dagi brauzer dialogini almashtiradi. Ikkalasi ham
**bir xil `escpos.Encode`** ni chaqiradi va **layout'ni qayta hisoblamaydi** —
qatorlar serverdagi `receipt.Render` dan keladi, ya'ni ega tasdiqlagan
ko'rinishdan.

## Avtomatik yangilanish

Kassa o'zini yangilaydi. Ish tartibi va nima uchun aynan shunday qilingani —
`update_windows.go` da; bu yerda **yangi versiyani qanday chiqarish** yozilgan.

### Bir marta: nima o'rnatilgan

O'rnatuvchi (`build/windows/installer/project.nsi`) `KeelKassaUpdate` nomli
<!-- ⚠️ Vazifa nomi mahsulot bilan birga o'zgartirilmadi: uni ishlayotgan
     binar nom bo'yicha qidiradi (update_windows.go), va o'zgartirish
     allaqachon o'rnatilgan har bir mashinada eski vazifani o'chirilgan
     faylga ishora qilgan holda qoldirardi. -->
rejalashtirilgan vazifa yaratadi. U kassaning **o'zini** `--apply-update` bayrog'i
bilan, eng yuqori huquqlar bilan ishga tushiradi.

⚠️ Bu — butun mexanizmning asosi. Kassa `Program Files` da turadi va oddiy
foydalanuvchi nomidan ishlaydi, ya'ni **o'z fayllarini almashtira olmaydi**.
Vazifa o'rnatish paytida — biz allaqachon administrator bo'lgan yagona paytda —
yaratilgani uchun kassa keyin UAC oynasisiz yangilana oladi. Peshtaxtada,
kechqurun soat sakkizda chiqadigan UAC oynasi — kassir yo yopadi, yo kimgadir
qo'ng'iroq qiladi, va ikkala holatda ham mashina eski versiyada qoladi.

### Har safar: yangi versiya chiqarish

1. **`desktop/version.go` dagi `Version` ni oshiring.**
   ⚠️ Buni unutish hech qayerda xato bermaydi — har bir kassa "men allaqachon
   yangiman" deb qaraydi va tuzatish hech kimga yetib bormaydi. Bu — reliz
   albatta teguvchi yagona qator.
2. `wails build --target windows/amd64 -nsis` — natija
   `build/bin/keel-amd64-installer.exe`.
3. Fayl nomiga versiyani qo'ying va serverdagi reliz papkasiga qo'ying
   (`TILL_RELEASE_DIR`, control plane muhitida):
   ```
   keel-1.1.0-installer.exe
   ```
4. Yoniga `latest.json` yozing:
   ```json
   {
     "version": "1.1.0",
     "url": "https://keel.uz/internal/till/download?file=keel-1.1.0-installer.exe",
     "sha256": "<sha256sum natijasi>",
     "notes": "Chek birlashtirish tuzatildi"
   }
   ```
   ⚠️ `sha256` majburiy — u bo'lmasa control plane manifestni umuman bermaydi.
   Kassa bu faylni ochiq internetdan olib, keyin **administrator huquqi bilan
   ishga tushiradi**: tekshirilmagan yuklab olish — bir xil wi-fi'dagi odam
   almashtira oladigan yuklab olish.

### Kassada nima bo'ladi

- Har **6 soatda** manifestni so'raydi. Yangi versiya bo'lsa, o'rnatuvchini
  `%PROGRAMDATA%\Keel\update\` ga yuklab oladi va **sha256 ni tekshiradi**.
  Mos kelmasa — o'chiradi va hech nima qilmaydi.
- Yuklab olingan fayl **darhol o'rnatilmaydi**. O'rnatish kassani yopadi
  (Windows ishlab turgan `.exe` ni almashtirmaydi), shuning uchun u **keyingi
  ishga tushishni** kutadi.
- Kompyuter yoqilganda (yoki elektr o'chib-yonganda — restoranlar aynan shunday
  qayta yuklanadi) kassa: staged faylni ko'radi → vazifani ishga tushiradi →
  vazifa o'rnatuvchini jim rejimda (`/S`) yuritadi → o'rnatuvchi kassani yopadi,
  fayllarni almashtiradi → vazifa kassani `explorer.exe` orqali qayta ochadi.
- ⚠️ `explorer.exe` orqali, chunki vazifa administrator huquqida ishlaydi va u
  ochgan har qanday dastur ham shunday bo'lardi — WebView2 ma'lumot papkasi
  Administrator nomiga yozilib, ertalab kassani ochgan oddiy foydalanuvchini rad
  etardi.
- ⚠️ **Ikki urinishdan keyin to'xtaydi.** Har safar yiqiladigan o'rnatuvchi aks
  holda har yoqilishni qayta ishga tushirish siklga aylantirardi — sotib
  turilgan mashinada.

### Sinov

Bu yo'lning Windows qismi haqiqiy mashinada sinalishi kerak: versiya
solishtirish testda (`update_test.go`), qolgani — `schtasks`, NSIS va WebView2
xatti-harakati, ular Linux'da tekshirilmaydi.
