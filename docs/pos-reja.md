# Keel POS — qaror va reja

Bu hujjat kod yozishdan **oldin** qabul qilingan qarorlarni va hali ochiq
qolgan savollarni yozib qo'yadi. Sabab fiskal narx faraziyasi bilan bir xil:
noto'g'ri faraz ustiga yozilgan kod keyin tashlanadi.

Sana: 2026-08-17. Holat: **reja, kod yo'q.**

---

## 0. Qamrov: hozir nima qilamiz, nima qilmaymiz

**Qilamiz** — POS'ni mukammal qilish:
- Windows uchun alohida kassa ilovasi (`.exe`)
- Windows uchun ofitsiant ilovasi
- Rasmli / rangli menyu, 200+ taomga optimizatsiya
- Chek dizayni (kassa, oshxona, mijoz — uchtasi alohida)
- Printer va pul yashigi
- **Oflayn ishlash**
- Kassir PIN kodi

**Hozir qilmaymiz** (POS ishlagandan keyin):
- Ombor
- Texkarta va tannarx
- Inventarizatsiya

⚠️ Bu tartib ataylab. Ombor jonli oshxonada ishlagan kassadan **keyin**
qurilishi kerak — texnologning fikrisiz qurilgan inventarizatsiya faqat u
sezadigan joylarda xato bo'ladi. Lekin ma'lumot modeli 1-kundan ularni
**hisobga olishi** kerak (§7).

---

## 1. Xavfsizlik — birinchi o'rinda, chunki javobgarligi bizda

Bu bo'lim birinchi, tasodifan emas. Qolgan har bir qaror shu bo'limga
tekshiriladi.

### ⚠️ Oflayn rejim yangi xavf yuzasi ochadi
Bugun mijoz ma'lumoti faqat serverda. Oflayn kassa esa **restorandagi
kompyuterda** ma'lumot saqlashni talab qiladi — va o'sha kompyuter qulflanmagan
xonada turadi, xodimlar almashadi, u o'g'irlanishi mumkin.

Shuning uchun **lokal saqlanadigan narsalar ro'yxati oq ro'yxat bo'ladi**, qora
ro'yxat emas:

| Lokal saqlanadi | Hech qachon lokal saqlanmaydi |
|---|---|
| Menyu, narx, variantlar, kategoriya | Mijozlar bazasi |
| Bugungi ochiq cheklar | Telefon raqamlari |
| Bugungi yopilgan cheklar (yuborilmagunicha) | Manzillar |
| Stol xaritasi | Buyurtmalar tarixi |
| Xodimlar ro'yxati (ism + PIN hash) | Loyalty balansi, ballar |
| Fiskal sozlama (manzil) | To'lov kalitlari, SMS/bot tokenlari |

⚠️ **"Kassaga kerak bo'lishi mumkin" — sabab emas.** Zaruriy minimum:
kassir hozir sotayotgan narsa. Qolgan hamma narsa serverdan, ulanish
bo'lganda.

⚠️ **Yuborilgan chek lokal bazadan o'chiriladi.** Aks holda oy oxirida
monoblokda o'ttiz kunlik savdo yotadi va uni hech kim tozalamaydi.

### ⚠️ PIN — autentifikatsiya emas, imzo
Bu farq muhim va oson chalkashtiriladi.

- **Qurilma** server tokeni bilan autentifikatsiya qilinadi (bugungi `staff`
  tokeni yoki agent kaliti). Bu — "bu monoblok shu filialniki".
- **PIN** o'sha qurilmadagi **amalni imzolaydi**: kim urdi, kim void qildi,
  kim chegirma berdi.

4 raqamli PIN — kalit emas, uni tanlab topish mumkin. Shuning uchun:
- PIN **hech qachon** serverga kirish uchun yetarli bo'lmaydi;
- bcrypt bilan hashlanadi (parollar bilan bir qoida);
- urinishlar cheklanadi (5 ta noto'g'ri → o'sha xodim uchun 60 soniya blok);
- **filial ichida unique** — ikki kassirda bir xil PIN bo'lsa, jurnal
  yolg'on gapiradi.

⚠️ **Hozirgi holat xato**: planshet butun kecha **bitta** ishchi tokeni bilan
ochiq turadi, ya'ni void va chegirmadagi "kim qildi" doim planshetga kirgan
odamning nomi. Javobgarlik uchun qurilgan mexanizm login modeli tufayli
ishlamayapti. PIN buni tuzatadi.

### ⚠️ Avtoyangilanish — har restoranga ochiq yo'l
Windows ilovasi o'zini yangilaydi, ya'ni yangilanish kanali **butun
platformaga** kiradigan eshik. Talablar:
- binar **imzolanadi** (code signing sertifikati);
- yangilanish faqat imzo tekshirilgandan keyin qo'llanadi;
- yangilanish manzili qattiq yoziladi, sozlamadan **o'qilmaydi**
  (aks holda o'zgartirilgan sozlama = o'zgartirilgan ilova).

### Mavjud qoidalar o'z kuchida qoladi
- Bir restoran = bir konteyner (izolyatsiya).
- Sirlar `restaurant` hujjatidan tashqarida.
- Karta raqami bizga hech qachon tegmaydi (hosted invoice).
- Filial qamrovi filtr ichida, tekshiruv emas.
- Eksport — muddatli ruxsat bilan.
- Agent kaliti almashtirilsa eskisi **shu zahoti** o'ladi (monoblok
  o'g'irlansa — birinchi qadam shu).

---

## 2. Windows ilovasi: qaysi texnologiya

Sizning fikringiz: Electron/Tauri optimizatsiyani buzadi. **Electron uchun
to'g'ri, Tauri uchun yo'q** — ular bir toifa emas, va farq katta:

| | Binar | RAM (taxminan) | UI kodi |
|---|---|---|---|
| **Electron** | 150+ MB (Chromium ichida) | 250–400 MB | React (mavjud) |
| **Tauri** | ~10 MB (tizim WebView2) | 80–120 MB | React (mavjud) |
| **Wails (Go + WebView2)** | ~15 MB | 80–120 MB | React (mavjud) |
| **Native (C#/WPF)** | ~5 MB | 60–100 MB | **Qaytadan yoziladi** |
| **Flutter Desktop** | ~20 MB | 100–150 MB | **Qaytadan yoziladi** |

Electron haqidagi xavotiringiz o'rinli: 4 GB RAM'li monoblokda u og'ir.
Tauri va Wails esa Windows 10/11 da **allaqachon bor** WebView2 ni ishlatadi —
Chromium'ni o'rab yurmaydi.

### Qaror: **Wails (Go + WebView2)** ✅
Sabablari:
1. **Mavjud React UI qayta ishlatiladi** — kassa ekrani allaqachon yozilgan.
2. **Go** — backend tilimiz, va `cmd/fiscalagent` allaqachon Go'da. Printer,
   COM port, fiskal kassa, lokal navbat — hammasi **bitta jarayonda**,
   ikkinchi til va ikkinchi jamoa ko'nikmasisiz.
3. Binar kichik: bizning `fiscalagent.exe` hozir **9 MB** — bu real o'lchov,
   taxmin emas.
4. Printer va pul yashigi Go'dan to'g'ridan-to'g'ri boshqariladi.

⚠️ **Agent ilovaga aylanadi, ikkinchi dastur qo'shilmaydi.** Bugungi
`fiscalagent` — o'sha ilovaning yadrosi. Restoranda **bitta** `.exe` turadi.

### Nega Tauri emas
Tauri obro'liroq va rasmiy plaginlari ko'proq (avtoyangilagich, single
instance, autostart). Ikki sabab bilan baribir Wails:

1. **Printer, COM port, fiskal kassa, lokal navbat — hammasi Go'da bo'ladi**,
   va `cmd/fiscalagent` allaqachon Go'da. Tauri bilan bu qatlam Rust'ga
   ko'chadi. Printer nosozliklari (model o'ziga xosligi, javob bermagan COM
   porti) aynan shu pastki qatlamda tuzatiladi, va o'sha paytda yangi til
   o'rganish loyihani to'xtatadi.
2. **Tauri'ning eng kuchli ustunligi — rasmiy avtoyangilagich — bizga kamroq
   keladi.** ⚠️ Kassa xizmat vaqtida jimgina qayta ishga tushmasligi kerak.
   To'g'ri siyosat: fonda yuklab olish, **keyingi ishga tushishda** yoki
   paneldan buyruq bilan qo'llash. Imzo tekshiruvi baribir bizniki (§1).

Wails o'zimiz yozadigan narsalar (hammasi kichik): single instance (Windows
mutex), autostart (fiskal agentda allaqachon hujjatlashtirilgan), yangilagich.

### Muhit — tasdiqlangan
Windows 10/11. Windows 7 uchraydigan joyda jamoa uni bepul 10 ga o'tkazadi,
ya'ni WebView2 hamma joyda bo'ladi. Monobloklar: minimal protsessor,
**minimal 4 GB RAM**.

⚠️ 4 GB va zaif protsessorda haqiqiy xavf framework emas — **bir vaqtda dekod
qilinadigan rasmlar soni** (§4). Bularsiz Wails ham, Tauri ham, C# ham
qotadi.

### Oyna: framesiz, to'liq ekran, o'z tugmalarimiz
`Frameless: true`, sudrash uchun `--wails-draggable` CSS, tugmalar
`runtime.Quit()` va h.k.

- **Kichraytirish tugmasi yo'q** — na kassada, na ofitsiantda. Monoblokda
  ilova orqasida hech nima yo'q, kichraytirish esa kassirning ekranni
  tasodifan yo'qotishi va "kassa o'chib qoldi" degan qo'ng'iroq.
- **Yopish tugmasi bor.** ⚠️ Va u qo'rqinchli tasdiq talab qilmaydi: §6 dagi
  chidamlilik qoidalari bajarilsa, yopish svet o'chishi bilan **bir xil yo'ldan**
  o'tadi. Ochiq chek bo'lsa yengil eslatma yetarli ("3 ta ochiq chek
  saqlanadi").
- ⚠️ **Ilova qotganda chiqish yo'li qolishi kerak** (masalan `Ctrl+Shift+Q`) va
  u o'rnatish hujjatida yozilishi shart — aks holda birinchi qotishda monoblok
  tokdan uziladi.

---

## 3. Ofitsiant ilovasi

**Bosqich 1 (hozir)**: Windows ilovasi, kassa bilan **bir xil o'ram**, boshqa
ekran. Sabab: zalda ham monoblok/noutbuk bo'ladi, va bitta build ikki
ilovaga xizmat qiladi.

**Bosqich 2 (keyinroq)**: **Keel Waiter** — Play Market va App Store'da
**yagona** ilova. Ofitsiant ilovani yuklaydi, restoranga ulanadi (kod yoki QR
orqali) va o'z hisobiga kiradi.

⚠️ Bu ikkinchi bosqich **hozirgi qarorga ta'sir qiladi**: yagona mobil ilova
kerak bo'lsa, UI qatlami mobilga ko'chishi kerak. Uchta yo'l:
- React UI'ni saqlab, mobilda **Capacitor/PWA** o'ramiga solish (eng arzon);
- React Native (React bilimi qayta ishlatiladi, UI qayta yoziladi);
- Flutter (hammasi qayta yoziladi, lekin Windows+Android+iOS bitta kod).

Hozir hal qilinmaydi, lekin **yozib qo'yiladi**: ofitsiant ekranini yozganda
uni kassa ekranidan **mustaqil** komponentlar bilan qurish kerak, aks holda
mobilga ko'chirishda kassa mantiqini ham tortib ketadi.

⚠️ Ofitsiant ilovasida printer **yo'q** — u zalda yuradi. Chek oshxonada va
kassada chiqadi. Ya'ni ofitsiantning oflayn talabi kassanikidan yengilroq.

---

## 4. Menyu ko'rinishi: rasm yoki rang

**Restoran o'zi tanlaydi** (sozlamalarda), ikki rejim:

1. **Rasmli plitka** — ko'rgazmali, yangi ofitsiant uchun tez o'rganiladi.
2. **Rangli plitka** — kategoriya rangi + katta matn. ⚠️ **200+ taomli menyuda
   aslida tezroq**: iiko'ning o'zi ko'proq shuni ishlatadi, chunki rasm
   qatorlarni bir-biriga o'xshatib yuboradi va ko'z matnni tezroq skanerlaydi.

Standart qaysi biri? **Rangli** — chunki nol qiymat bugungi xatti-harakat
bo'lishi kerak va rasmsiz ishlaydigan menyu hamma restoranda bor
(bo'sh `mapProvider` = 2GIS qoidasi).

### ⚠️ 200+ taom uchun optimizatsiya — bu UI emas, tizim masalasi
- Rasm **kassa uchun alohida o'lchamda** (`?w=` mexanizmi bor, lekin ~180 px
  kerak — plitka o'lchami).
- Rasm **lokal keshda** saqlanadi (ilova ichida), tarmoqdan qayta so'ralmaydi.
  Oflayn talabi buni baribir majbur qiladi.
- Ekranda bir vaqtda **bir kategoriya** — butun menyu emas.
- Ro'yxat virtuallashtiriladi (ko'rinmayotgan plitka DOM'da bo'lmaydi).
- Qidiruv allaqachon brauzerda va tez (`lib/search.ts`), server so'rovi yo'q.

---

## 5. Chek dizayni — uchta chek, uchta maqsad

Restoran **sozlamalarda o'zi dizayn qiladi**. Uchtasi **alohida**, chunki
ularni uch xil odam o'qiydi:

| Chek | Kim o'qiydi | Nima muhim |
|---|---|---|
| **Oshxona cheki** | Oshpaz | Taom nomi katta, izoh (piyozsiz), stol, vaqt. **Narx yo'q.** |
| **Kassa cheki** | Kassir / ega | Jami, to'lov turi, chegirma, kassir |
| **Mijoz cheki** | Mehmon | Restoran nomi, tarkib, jami, **fiskal QR** |

⚠️ **Oshxona chekida narx bo'lmasligi kerak** — u oshpazga hech nima
aytmaydi va chekni uzaytiradi. Har qo'shimcha qator — pass'da o'qiladigan
qo'shimcha vaqt.

⚠️ **Fiskal QR faqat mijoz chekida.** U mehmonning tekshirish huquqi;
oshxonaga chiqarish qog'oz isrofi.

Dizayn nima o'zgartiradi: sarlavha/logo, footer matni, shrift o'lchami,
qog'oz kengligi (58 mm / 80 mm), qaysi maydonlar chiqishi, til.

⚠️ **Qog'oz kengligi sozlama, taxmin emas**: 58 mm printerga 80 mm chek
yuborilsa matn kesiladi va buni faqat restoran ko'radi.

---

## 6. Oflayn — eng qiyin qism va sotuvning sharti

### Nima oflayn ishlashi **shart**
- Chek ochish, taom qo'shish, olib tashlash
- Oshxonaga yuborish + **oshxona cheki chop etilishi**
- Chek yopish, naqd olish, qaytim
- **Fiskal chek** — ⚠️ va bu ishlaydi, chunki **fiskal kassa lokal**
  (`localhost`/LAN). Internet uzilganda ham chek soliqqa tushadi. Bu katta
  ustunlik va uni ataylab shunday qurdik.
- Kassa smenasini yopish

### Nima oflayn ishlamaydi (va ekranda shunday deyiladi)
- Loyalty ballari (balans serverda)
- Promokod tekshiruvi
- Yetkazib berish buyurtmalari
- Mijoz kartochkasi
- Karta to'lovi — ⚠️ baribir alohida bank terminalida, ya'ni amalda muammo emas

### ⚠️ Svet o'chishi — bu sinxronizatsiya emas, chidamlilik masalasi

Monobloklar tokda ishlaydi, batareyada emas. Svet o'chadi, generator kelganda
qaytadan yonadi. Bu **toza yopilish emas**: ogohlantirish yo'q, xotiradagini
diskka yozishga imkon yo'q. Ilova **istalgan buyruqda** o'lib, qaytganda izchil
holatda ochilishi kerak.

**Asosiy qoida: har o'zgarish ekranda ko'rsatilishidan OLDIN diskka yoziladi.**
Kassir taom qo'shdi → avval SQLite, keyin ekran. Aks holda ekran yolg'on
gapirgan bo'ladi.

⚠️ **SQLite: WAL + `synchronous=FULL`.** Standart `NORMAL` svet o'chganda
oxirgi tranzaksiyalarni **yo'qotishi mumkin** (baza buzilmaydi, lekin yozuv
yo'qoladi) — pul oladigan kassa uchun yaramaydi. `FULL` har commit'da fsync
qiladi: SSD'da 1–5 ms, kassa tezligida sezilmaydi.

Ochiq chek shundan keyin o'z-o'zidan omon qoladi.

#### Chop etish: odam qayta yuboradi, dastur emas
Oshxona cheki chop etilayotganda svet o'chsa, printerdan qaytish aloqasi yo'q —
chiqdimi yoki yo'q, bilib bo'lmaydi.

⚠️ **Avtomatik qayta chop etish qilinmaydi.** U — taxmin: dastur chekning
chiqqan-chiqmaganini bilmaydi. Pass oldidagi odam esa **biladi** — borib
qaraydi. Kassir yoki ofitsiant chekni **qo'lda qayta yuboradi**, va bu hamma
restoranda shunday ishlaydi.

⚠️ **Bu butun bir holat mashinasini olib tashlaydi**: chop etish tasdig'ini
kuzatish, qayta ishga tushganda tasdiqlanmaganlarni topish, ularni belgilash —
hech biri kerak emas. Chop etish "yubordim va unutdim" bo'ladi.

**Qayta yuborishda izoh so'raladi va u chekning o'zida chiqadi.** Umumiy
"TAKROR" so'zidan ko'ra "Svet o'chdi" degan matn oshpazga aynan nima
bo'lganini aytadi — ya'ni izoh belgining **o'rnini bosadi**, yoniga
qo'shilmaydi.

⚠️ **Lekin bu void bilan bir toifa emas.** Void pulni olib chiqadi, shuning
uchun sababi **javobgarlik** uchun va qat'iy talab qilinadi. Qayta chop etish
faqat qog'oz sarflaydi — chekka ham, pulga ham tegmaydi — va sababi **oshxona
bilan muvofiqlashtirish** uchun. Shuning uchun tayyor variantlar yetarli
("svet o'chdi", "chek chiqmadi", "chek yo'qoldi") + erkin matn.

**Bilinadigan xato bilinmaydiganidan ajratiladi.** Printer o'chiq yoki qog'oz
tugagani **bilinadi** (ulanish rad etiladi) — ekranda ko'rsatiladi, odam
taxmin qilib o'tirmaydi. Svet o'chib javob kelmagani **bilinmaydi** — dastur
jim turadi va qaror odamniki.

#### ⚠️ Soat — eng jimgina buziladigan joy
Eski monoblokda CMOS batareyasi o'lgan bo'lsa, svet o'chib yonganda **sana
nolga qaytadi** (masalan 2010-yilga). Oflayn kassa o'z soatidan foydalanadi va
**noto'g'ri sanali fiskal chek** yozadi — bu soliq hujjati, va xato hech qayerda
ko'rinmaydi.

Qoida: **kassaning vaqti hech qachon oxirgi yozilgan hodisadan orqaga
ketmaydi.** Ishga tushganda serverdan vaqt olinadi va siljish saqlanadi; server
yo'q bo'lsa-yu lokal soat oxirgi chekdan oldinda bo'lmasa — ekran ogohlantiradi
va sotish to'xtaydi.

#### Yoqilish: avtologon
⚠️ Generator kelib monoblok yonganda **hech kim login qilmasligi kerak**.
Monoblokda **Windowsga avtomatik kirish** sozlanadi va ilova Startup'ga
qo'yiladi. Busiz monoblok login ekranida turadi va restoran to'xtaydi — bu
dasturiy emas, **o'rnatish** masalasi, lekin biz aytmasak hech kim qilmaydi.

#### UPS
Kichik UPS (~40–60 $) 5–10 daqiqa beradi va qisqa uzilishlarni butunlay
yashiradi. ⚠️ Lekin dastur baribir chidamli bo'lishi shart: UPS batareyasi ikki
yildan keyin o'ladi va buni hech kim sezmaydi.

#### Fiskal kassa ham o'chadi
U ham o'sha tokda. Qayta yonganda smenasi yopiq bo'lishi mumkin → `#2D` →
allaqachon avtomatik ochiladi. Qo'shimcha ish talab qilmaydi.

### ⚠️ Uchta qiyin qaror

**1. Chek raqami to'qnashmasligi kerak.**
Ikki kassa oflayn ishlasa va ikkalasi ham raqam bersa — bir xil raqamli ikki
chek. Yechim: har kassaga **prefiks** (filial prefiksi bilan bir naqsh:
`MRC-K1-1745`). Prefiks qurilma ro'yxatdan o'tganda beriladi, oflayn
yaratilmaydi.

**2. Oflayn ortiqcha sotish mumkin.**
Stop list serverda. Ikki kassa oflayn holda oxirgi porsiyani sotishi mumkin.
⚠️ **Buni qabul qilamiz**: oflayn sotmaydigan kassa — sotib bo'lmaydigan
mahsulot. Restoran oxirgi porsiya uchun uzr so'raydi, bu odatiy ish.

**3. Ulanish qaytganda nima bo'ladi.**
Cheklar serverga **yaratilgan vaqti bilan** yuboriladi, yuborilgan vaqti bilan
emas. Aks holda kechqurungi butun savdo bir daqiqada bo'lgandek ko'rinadi va
soatlik statistika yolg'on gapiradi.
⚠️ Server chekni **id bo'yicha idempotent** qabul qiladi — qayta yuborish
ikkinchi chek yaratmaydi. (`request_id` naqshi, Yandex Delivery'dagi kabi.)

---

## 7. Ombor keyin — lekin model hozir hisobga oladi

Ombor qilinmaydi, ammo ikki narsa **hozir** to'g'ri qo'yilishi kerak, aks
holda keyin og'riqli bo'ladi:

- **Nima qayerga tegishli**: menyu va texkarta — **brendga**; ingredient
  narxi, qoldiq, ombor — **filialga**. Bu `soldOut` qoidasining davomi.
- **Markaziy oshxona (tsex)** mavjudligi. O'zbekistonda keng tarqalgan
  (Evos, Oqtepa naqshi): markazda yarim tayyor mahsulot chiqadi, filiallarga
  tarqaladi. Bunga **ombor turi** (tsex/filial), **ko'chirish hujjati** va
  **ishlab chiqarish hujjati** kerak bo'ladi.

⚠️ Bugun jadval yaratilmaydi; faqat "bir filial = bir ombor" degan faraz
kodga **yozib qo'yilmaydi**.

---

## 8. Tartib

| # | Ish | Nega shu tartibda |
|---|---|---|
| 1 | Buyurtmalar linzasi (`check` cheklari `/admin/orders` dan chiqadi) | Kichik, bugungi shikoyat |
| 2 | **Kassir PIN** | Javobgarlik hozir buzuq |
| 3 | Kassa UI monoblokka: plitka, rasm/rang, katta nishonlar | Ekran hozir juda kichik |
| 4 | Chek dizayni (uchta shablon) | Printerdan oldin kerak |
| 5 | **Windows ilova** (Wails), printer + pul yashigi | Agent shundan o'sadi |
| 6 | **Oflayn** | Eng qiyini; 5 ga bog'liq |
| 7 | Ofitsiant Windows ekrani | Kassa naqshini qayta ishlatadi |
| — | *keyin*: texkarta, tannarx, ombor, mobil Keel Waiter | |

---

## 9. Ochiq savollar

1. ~~Wails yoki Tauri~~ — **Wails tanlandi** (§2).
2. **Printer modellari.** ESC/POS standart, lekin O'zbekistondagi keng
   tarqalgan modellarni bilish kerak (Xprinter, Rongta, Epson?). Ulanish
   USB'mi, tarmoq'mi, COM'mi — ilovaning qaysi kutubxonasi kerakligini shu
   hal qiladi.
3. ~~Monoblok konfiguratsiyasi~~ — **Windows 10/11, minimal protsessor,
   minimal 4 GB RAM** (§2).
4. **Kassa uchun alohida narx?** POS to'liq bo'lgach bu boshqa mahsulot va
   boshqa pul — fiskal narx faraziyasidagi xato takrorlanmasligi kerak.
5. **Code signing sertifikati** — kim oladi, qancha turadi. Yangilanish
   kanali xavfsizligining sharti.

---

## 10. Ruxsatlar va rollar

### ⚠️ Asosiy tanglik: ko'p ruxsat — umumiy PIN demakdir
Har tugmasi "ruxsat yo'q" deydigan ekranda xodimlar bitta yechim topadi —
kassirning PIN kodini hammaga aytish. Shunda javobgarlik butunlay yo'qoladi:
jurnal bor, lekin u doim bitta odamni nomlaydi.

Shuning uchun ruxsat faqat ikki holatda paydo bo'ladi: amal **pulni olib
chiqsa** yoki **yozuvni yo'q qilsa**.

### Oltita ruxsat

| Ruxsat | Nima |
|---|---|
| `waiter` | Chek ochish, taom qo'shish, izoh, oshxonaga yuborish, stol almashtirish |
| `cashier` | To'lov qabul qilish, chekni yopish |
| `void` | Pishirilgan taomni yoki butun chekni olib tashlash, yopilgan chekni qayta ochish |
| `discount` | Chegirma berish |
| `shift` | Kassa smenasini ochish/yopish, kassa kunini yopish (Z-hisobot) |
| `kitchen` | Oshxona ekrani (KDS) |

**Ataylab ruxsat talab qilmaydiganlar**: yuborilmagan qatorni o'chirish (imlo
xatosi), taomga izoh, oshxona chekini qayta chop etish, mehmonlar soni, chekni
boshqa **stolga** ko'chirish.

### ⚠️ Menejer tasdig'i (override) — tizimni ishlaydigan qiladigan qism
Ruxsati yo'q amal **rad etilmaydi** — ekran PIN so'raydi, ruxsati bor odam o'z
kodini teradi, va **ikkala nom ham yoziladi**: *"Aziz olib tashladi, Dilnoza
tasdiqladi"*.

Usiz qat'iy ruxsatlar bir hafta ichida o'chirib qo'yiladi: menejer zalda,
ekran oldida turibdi, va rad etish ofitsiantni ekrandan uzoqlashtirib
oxir-oqibat PIN almashishga olib keladi.

### Rollar — katakcha emas
`staff_role` kolleksiyasi: nom + ruxsatlar to'plami. Xodimga **rol** beriladi.
Oltita katakcha × 40 xodim = 240 ta katakcha, va ular bir-biriga mos kelmay
ketadi.

⚠️ **Rol lavozimning o'rnini bosadi.** CLAUDE.md dagi "lavozim ruxsat emas"
ogohlantirishi kuchda qoladi va aynan shu sabab rol — **id'si bor yozuv**,
terilgan matn emas. Xatoni ushlaydigan narsa shu: "Oshpaz" deb yozgan odamga
kalit berilmaydi, ro'yxatdan tanlangan rolga beriladi.

### Tayyor rollar (restoran tahrirlaydi)

| Rol | waiter | cashier | void | discount | shift | kitchen |
|---|---|---|---|---|---|---|
| Ish boshqaruvchi | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Menejer | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Zal administratori | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| Kassir | ✓ | ✓ | — | — | ✓ | — |
| Barmen | ✓ | ✓ | — | — | — | ✓ |
| Ofitsiant | ✓ | — | — | — | — | — |
| Xostes | — | — | — | — | — | — |
| Oshxona boshlig'i | — | — | — | — | — | ✓ |
| Oshpaz | — | — | — | — | — | ✓ |
| Texnolog | — | — | — | — | — | — |
| Yordamchi xodim | — | — | — | — | — | — |

⚠️ **Ish boshqaruvchi va Menejer bir xil ruxsatga ega, va bu ataylab**: ular
jurnalda **boshqacha nomlanadi** ("Menejer Dilnoza" va "Ish boshqaruvchi
Aziz"), va keyinchalik alohida toraytirilishi mumkin. Bir xil ruxsat — bugungi
holat, abadiy shart emas.

⚠️ **Yangi installda Kassir `void` va `discount` siz** — siz aytganingizdek,
kassirga ham ruxsat kerak. **Lekin migratsiya mavjud kassirlarga ularni
qoldiradi**: bugungi `canCashier` odam hozir ham void va chegirma qila oladi,
va yangilanish odamlarning mavjud huquqini smena o'rtasida tortib olmasligi
kerak (`EnsureKitchenAccess` bilan bir dars). Restoran keyin ataylab
toraytiradi.

### Qoida qayerda yashaydi
⚠️ **Serverda.** Ekran tugmani yashirishi faqat xushmuomalalik — bu kod
bazasining mavjud qoidasi (`kitchenDenial`, `tillDenial`). Override ham
serverda tekshiriladi: brauzer "tasdiqlandi" deb aytolmaydi.

---

## 11. Ochiq kamchiliklar (2026-08-17, jonli sinovdan keyin)

### ✅ 1. Zal cheki qo'ng'iroq chalardi — **bajarildi**

**Alomat**: kassadan yoki zaldan chek ochilsa, buyurtmalar ro'yxatida
ko'rinmaydi, **lekin ovoz chalinadi**.

**Sabab** (tekshirildi, taxmin emas): men `check` filtrini faqat
`AdminListOrders` ga qo'shganman. `AdminAlerts` — butunlay **boshqa endpoint**,
va u hali ham zal cheklarini sanaydi:

- `StaffOpenCheck` chekni `status: pending`, `paymentStatus: "unpaid"` bilan
  yaratadi;
- `AdminAlerts` dagi `pendingOrders` filtri `status == pending` **va**
  `paymentStatus != "pending"`;
- `"unpaid" != "pending"` → **mos keladi** → qo'ng'iroq.

⚠️ Bu men aytganimdan yomonroq holat: ro'yxat bo'sh, ovoz esa chalinadi. Ya'ni
panel **yo'q narsa uchun** jiringlaydi, va operator uni topa olmaydi.

**Tuzatildi** (`handlers/adminstats.go`, `AdminAlerts`): *yetkazishga
tegishli* filtrlarning har biriga `check: {$exists: false}` qo'shildi —
`pendingOrders`, `plain`, `placed`, `due`, `dueWaiting`, `upcoming`.

Ikkala yarmi ham testda muhrlangan (`alertsbell_test.go`): biri olti filtrning
har birida `check` borligini talab qiladi, ikkinchisi esa quyidagi uchtasida
**yo'qligini** talab qiladi. Xato aynan "filtr bir joyga qo'shildi,
ikkinchisiga yo'q" shaklida tug'ilgan edi, ya'ni qaytib kelish yo'li ham shu.

⚠️ **Hammasiga emas**: `pos.failed`, `pos.unaccepted` va `fiscal.unfiled`
aynan zal cheklariga ham tegishli va ular **qolishi kerak**. Qoida "hamma
joydan olib tashlash" emas — "**yetkazish buyurtmasi qabul qilinishi kerak**"
degan ma'nodagi ogohlantirishlardan olib tashlash.

### ✅ 2. Kassa va zal dizayni — **bajarildi**

⚠️ **Ildizi did emas, tuzilma edi: kassa saytning dizayn tizimini o'qiyotgan
edi.** `--radius-btn: 9999px`, `--root-size`, ega tanlagan aksent — bularning
hammasi vitrina uchun to'g'ri va kassa uchun ikki marta noto'g'ri. Ega temani
sinab ko'rgani uchun shakli o'zgaradigan ekran — mushak xotirasi nolga
tushadigan ekran, kattalashtirilgan `--root-size` esa 1024×768 monoblokda
sahifani chetidan chiqaradi.

Shuning uchun `globals.css` da **`.till` qatlami**: o'zi meros olgan
o'zgaruvchilarni qayta e'lon qiladi (`--radius-*` qat'iy 10–14 px) va
`till-chrome / till-row / till-btn / till-panel / till-input / till-label`
beradi. Kassa, zal, PIN paneli va hamma dialog shundan o'qiydi — saytning
`.btn`/`.card` klasslari bu ekranlarda umuman ishlatilmaydi.

Qaror qoidalari:
- ⚠️ **Rang faqat ma'no tashiydi.** Ilgari kategoriya tugmalari to'yingan
  rangda to'ldirilgan edi va taom plitkasining **butun foni** kategoriya
  ohangida bo'lardi — natijada qorong'i temada har plitka loyqa jigarrang
  chiqib, taom nomi ham, narxi ham foniga botib ketardi. Endi rangni **chap
  chiziq** tashiydi, sirt neytral. Ohanglar bitta to'plamdan, tema esa ikkita:
  fon sifatida ishlatilgan ohang ikkalasida bir xil ishlay olmaydi.
- ⚠️ **Rang chizig'i rasmdan yuqorida** (`z-10`): DOM'da rasm undan keyin
  turadi va uni butunlay yopib qo'yardi, ya'ni rasmli har bir plitkada —
  hammasida — rang belgisi umuman mavjud emas edi.
- ⚠️ **Qorong'i ramka, yorug' ish maydoni.** Ramka ikkala temada bir xil,
  ya'ni yorqinligi o'zgaradigan yagona narsa — taomlar panjarasi, va tugmalar
  soat o'nda ham ko'z o'rgangan joyda qoladi.
- ⚠️ **To'ldirilgan tugma bitta, va u chek qaysi bosqichda ekaniga qarab
  ko'chadi**: yuborilmagan qator bo'lsa — "Oshxonaga yuborish", bo'lmasa —
  "To'lash". Ilgari yuborish tugmasi `bg-charcoal` edi, ya'ni **qorong'i temada
  sahifa fonining o'zi**: ekrandagi eng shoshilinch amal eng ko'rinmas narsa
  bo'lib turardi.
- **Raqamlar `tabular-nums`**: narxlar ustunda turadi va ustun bo'ylab
  taqqoslanadi, proporsional raqamlar esa bu taqqoslashni taxminga aylantiradi.

### ⚠️ 2b. Yo'l-yo'lakay topilgan uchta xato (dizayn emas, nuqson)
- **`/kassa` stollar ekranini umuman render qilmasdi.** `TablesScreen` import
  qilingan, `view` holati bor edi, lekin JSX'da yo'q — ya'ni "avval stollar"
  talabi bajarilgan deb hisoblangan, amalda esa kassa hamon menyudan
  ochilardi. Endi ish maydoni ikki ko'rinishli: zal → (chek ochilgach) menyu,
  va "←" bilan zalga qaytiladi (monoblok fullscreen, brauzer tugmasi yo'q).
- **Chek paneli o'ziga kenglik yozayotgan edi** (`lg:w-80`) — u allaqachon
  kengligi berilgan ustun ichida, natijada 24 px chetdan chiqib butun sahifaga
  gorizontal skroll qo'shardi.
- **Peshtaxta plitkasi chek raqamini eng katta matn qilib chizardi**
  (`6XGC-ZNHV`). U `crypto/rand` dan (§Xavfsizlik) — ataylab taxmin
  qilinmaydigan, ya'ni ataylab **o'qilmaydigan**. Endi plitka tartib raqami
  bilan (`#1`), haqiqiy raqam esa mayda: mehmon qaytib kelganda uni chekdan
  solishtirish kerak bo'ladi.

### ✅ 2c. Kassada variant tanlash oynasi — **bajarildi**
Jonli sinovda chiqqan edi: "Osh (palov)" ni bosganda `"hajm" tanlanmagan`.
Server to'g'ri ish qilardi (`resolveOptions` majburiy guruhni rad etadi),
planshet esa faqat `{menuItemId, qty}` yuborardi va **hech bir ekranda javob
berish imkoni yo'q edi** — ya'ni variantli har bir taom kassadan sotilmasdi.
Kamdan-kam holat emas: porsiya hajmi — restoranda eng oddiy variant, va nosozlik
"kassa buzuq" bo'lib ko'rinardi.

`components/till/OptionDialog.tsx`, kassa va zalda birdek.

- ⚠️ **Faqat guruhi bor taomga ochiladi.** Bitta bosish — kassaning butun tezlik
  argumenti; har taomga tasdiq qadami eng ko'p ishlatiladigan ekrandagi har
  bosishni ikkilantirardi. Shart **guruh bor-yo'qligi** bo'yicha, "majburiymi"
  bo'yicha emas: keyin qatorni tahrirlab qo'shiladigan ixtiyoriy
  "qo'shimcha pishloq" ham kassada sotilmaydigan guruh, va mehmon uni
  peshtaxtada so'raydi.
- ⚠️ **Kalit — base (uz) nomi**, tarjima emas: o'sha nom simda ketadi va server
  shunga solishtiradi. Tarjimani saqlash kassirning interfeys tilini
  buyurtmaga yuborardi va ruscha planshetda yiqilardi.
- ⚠️ **Narx ko'rsatiladi**, chunki u plitkadagi narx emas: +5 000 lik katta
  porsiya chekka kassir umuman ko'rmagan raqam bo'lib tushadi, va summani
  mehmonga aytib berish — bu ekran mumkin qiladigan yagona narsa.
- ⚠️ **Majburiy guruhda tanlovni bekor qilib bo'lmaydi**: tugmasi endigina
  o'lgan oyna kassir uchun "ekran qotdi" degani.
- ⚠️ **Plitkada nuqta**: ikki bosish ikki xil ish qiladi, va farqini
  ko'rmagan kassir javob kutayotgan taomni ikkinchi marta bosadi — natijada
  chekda ikki qator yoki bittasi ham yo'q.
- ⚠️ **`busy` bayrog'i**: so'rov ketayotganda ikkinchi bosish taomni ikki marta
  qo'shardi — aynan oyna oldini olishi kerak bo'lgan "chekda ikki qator",
  oynaning o'zi orqali. Restoran wifi'sida bu oyna tasodifan tushish uchun
  yetarlicha keng.
- ⚠️ **Miqdor shu yerda**: oyna allaqachon to'xtatib turibdi, ya'ni so'rash
  bepul bo'lgan yagona payt. Variantsiz taom bitta bosishligicha qoladi.
- **Tanlangan variant chekda yoziladi** (kassa va zal panelida): busiz bir xil
  nomli ikki qator ikki xil narxda turadi va farqini na kassir, na mehmon
  tushuntira oladi. Bosma chek buni doim chizardi — chek yasaladigan ekran esa
  yo'q.

### ✅ 2d. Kassa/zal: faqat yorug' tema, uch til, yangi qulf ekrani
- **Faqat yorug'** (`forcedLight()` — `lib/theme.tsx`). Kassa va zal — sayt emas,
  **jihoz**: kun bo'yi monoblokda, restoran yorug'ligida, printer va pul
  yashigi yonida ishlaydi, undagi taom rasmlari esa oq fonda olingan va
  tahrirlangan. Soat o'nda noto'g'ri tugmani bosib ekranni qorong'iga
  o'tkazgan kassir butun smenaning asbobini o'zgartirgan bo'ladi — va zaldagi
  boshqa hech kim uni qaytarishni bilmaydi.
  ⚠️ **Qoida ikki joyda va ikkalasi bir xil aytishi shart**: `app/layout.tsx`
  dagi inline skript birinchi bo'yashdan **oldin**, `lib/theme.tsx` esa
  **keyin** ishlaydi — har qanday nomuvofiqlik har kassa yuklanishida
  ko'zga ko'rinadigan chaqnash bo'ladi.
  Saqlangan tanlov **o'chirilmaydi**: kassa eganing panelda nima ko'rishini hal
  qilmaydi, faqat `<html>` dagi klass berilmaydi.
- **Tema tugmasi olib tashlandi**: hech nima qilmaydigan tugma yo'q tugmadan
  yomonroq — kassir uni bosadi, hech nima bo'lmaydi, va keyingi haqiqatan
  ishlamagan tugma ham ikki marta bosiladi.
- ⚠️ **Topilgan xato: `/kassa` va `/zal` `UNLOCALIZED` ro'yxatida yo'q edi.**
  Ya'ni ruschaga o'tilsa manzil `/ru/kassa` bo'lardi va **yo'lga bog'langan har
  bir qoida mos kelmay qolardi** — birinchi navbatda `forcedLight()`, ya'ni
  kassa tilini o'zgartirgan zahoti qorayardi. Ilovalarda til URL'i bo'lmaydi
  (§10 "Til URL'lari"), ular login orqasida va indekslanmaydi.
- **Qulf ekrani qayta chizildi**: tepada **Keel** belgisi va nomi (qora,
  Poppins), pastda PIN paneli.
  Guruh (belgi + panel) **ekran o'rtasida**.
  ⚠️ Ostida `overflow-y-auto` qoladi: qisqa ekranda o'rtaga qo'yilgan blok
  sig'masa **ikkala uchini** yo'qotadi, pastdan yo'qotadigani esa
  **backspace turgan qator**. O'rtalash guruh qayerda turishini hal qiladi,
  skroll esa unga har doim yetish mumkinligini.
  ⚠️ Nom **bizning shriftimizda**: tema shriftlari restoranni kiyintiradi
  (menyusi, sayti, cheki), va ularga bizning nomimizni ham qayta terishga
  ruxsat berilsa, belgi har installda boshqacha bo'lardi.
- **Belgi va nom yonma-yon**, panelning **ustida**, orasi bir qator.
  ⚠️ **Pad bilan bitta guruh**: yarim ekran bilan ajratilganda ular ikki narsa
  bo'lib o'qiladi, va oradagi bo'sh tasma displeydagi eng katta shakl edi.
  ⚠️ **Rang bizniki** (`keel.deep`, `tailwind.config.ts`), `brand` emas:
  `brand` — ega paneldan tanlaydigan aksent, ya'ni har installda boshqa, va
  unda chizilgan Keel belgisi har mijozda **boshqa logotip** bo'lardi.
  ⚠️ Rasmiy rang `#F5A524` (`logos/keel-mark.svg`), lekin kassaning oq qulf
  ekranida u rangsiz dog' bo'lib qoladi — shuning uchun bir pog'ona to'qrog'i
  (`#D2870F`). Shtrix qalinligi ham manba fayldagidek (2.6): 2.4 da yarim
  qalin nom yonida ingichka ko'rinadi.
- ⚠️ **Til almashtirgichi qulf ekranida va smena darvozasida ham bor.** Kassa —
  umumiy mashina, va uni sozlagan odam hozir uning oldida turgan odam emas:
  ruscha o'qiydigan kassir faqat o'zbekcha qulf ekraniga duch kelardi va butun
  smenani tuzatadigan almashtirgichga **o'tolmasdi**.

### ✅ 3. PIN'dan keyin smena so'raladi — **bajarildi**
`components/till/ShiftGate.tsx` (`useShift` + darvoza). Ikkala ekranda ham
(`/kassa`, `/zal`) PIN qabul qilingandan keyin birinchi savol — kassa smenasi.

⚠️ **Banner emas, darvoza**: ishlab turgan ekran tepasidagi yozuv o'qilmay
o'tib ketiladi — birinchi mehmon allaqachon turibdi, smenani "bir daqiqadan
keyin" ochish esa hech qachon bo'lmaydi. Smena ochilmaguncha zal chizilmaydi.
Sababi arifmetik: smenasiz ochilgan chek **hech qanday hisobga kirmaydi** —
pul olinadi, chek chiqadi, kechqurungi hisob aynan shu summaga kam bo'ladi va
hech qayerda xato chiqmaydi.

⚠️ **Ruxsati yo'q odamga tugmadan oldin aytiladi**: rad javobidan bilib olish
menejerni chaqirgani borib kelish demakdir. Rad etilganda `OverrideDialog` —
qolgan hamma joydagi bilan bir naqsh, ikkala nom ham yoziladi.

⚠️ **`CashShiftPanel` darvozaga xabar beradi** (`onChanged`): bir faktning
ikki o'quvchisi ajrab ketadi, ya'ni kassir smenani yopib, hali ham ochiq deb
turgan zalda buyurtma qabul qilishda davom etardi.

Eski yozuv (bajarilgan):

- PIN'dan keyin — **kassa smenasi ochiqmi?** Ochiq bo'lmasa **ochish** taklif
  qilinadi (qoldiq so'raladi).
- **Ofitsiant ekranida ham** shu: zalda ham smena tushunchasi bo'lishi kerak.
- Smenani yopish ham shu yerdan.

Backend tayyor (`/staff/cash-shift/*`, `shift` ruxsati, override) — bu
**ekran oqimi** masalasi, model masalasi emas.

### ✅ 4. Stollar: zal va saboy alohida — **bajarildi**
`models.TableZone` (`bookable`, `layout`), `FloorTable.ZoneID`,
`(*BookingSettings).Bookable()` — uch ekran o'qiydigan **bitta** funksiya.
Panel: `components/admin/TableZonesEditor.tsx` (zona CRUD + **raqam oralig'i
bilan stol qo'shish**, masalan 100–130). Kassa/zal: zona **tablari**.

⚠️ **Bo'sh zona ro'yxati bugungi xatti-harakat**: zona qo'shmagan restoranda
hamma stol bron qilinadi va tab chizilmaydi — bitta tabli tab chizig'i butun
ekran allaqachon aytgan narsani takrorlaydi.

⚠️ **Zonasiz stol bron qilinadi** (`ZoneID == ""`), va o'chirilgan zonadagi
stol ham: nol qiymat "bron qilinmaydi" bo'lsa, bu xususiyat chiqqan deploy
har bir mavjud restoranning bron sahifasini bo'shatardi.

⚠️ **Saboy raqamlari mehmonning brauzeriga umuman bormaydi**: `BookingPlan`
bron qilinmaydigan stollarni **ham**, zonalarni **ham** javobdan chiqaradi —
faqat ekranda yashirish emas.

⚠️ **Zona o'chirilsa stollari qolади** (standart zonaga qaytadi): adashib
o'chirilgan zona bilan birga bir kechalik ochiq cheklar va o'sha stollarga
ishora qilgan bronlar ketmasligi kerak.

Test: `internal/handlers/tablezones_test.go` (4 ta, o'tadi).

Eski yozuv (bajarilgan):

Hozirgi model: stollar faqat `booking.tables` (bron uchun chizilgan xarita).
Kerak bo'lgani:

- **Zonalar**: "Zal" (masalan 1–28, restoran xaritasidan olinadi) va "Saboy"
  (masalan 100–130) — alohida guruhlar, alohida raqam oralig'i.
- **Ikki ko'rinish**: xarita (chizma) va **ro'yxat**. Restoran sozlamalardan
  tanlaydi.
- Saboy stollari xaritada chizilmasligi mumkin — ular shunchaki raqamlar
  to'plami.

⚠️ Bu **modelni kengaytiradi**: bugun stolda zona tushunchasi yo'q, va bron
tizimi ham shu ro'yxatni o'qiydi. Saboy stollari bronga chiqmasligi kerak —
ya'ni zonaning "bron qilinadimi" bayrog'i ham kerak bo'ladi.

### ✅ 6. Panel sidebari: sidebar ichida sidebar — **bajarildi**
Ilgari yigirma ikkita yozuv bitta ustunda, guruh sarlavhalari bilan turardi.
Guruhlash yordam berdi, lekin **hammasi baribir bir vaqtda ekranda** edi, ya'ni
beshinchi bo'limga yetish uchun to'rttasini o'qib o'tish kerak edi.

Endi ikki qavat (`app/admin/layout.tsx`): tor **rels** (biznesning qismlari,
ikonka + so'z) va uning yonida **panel** (o'sha qismning ekranlari).

- ⚠️ **Rels navigatsiya qilmaydi.** Guruhga bosilsa faqat ikkinchi ustun
  almashadi. Guruhning birinchi ekraniga o'tadigan rels "Jamoa"da nima borligini
  ko'rmoqchi bo'lgan odam uchun hisobotni yuklab yuborardi — hech kim
  so'ramagan ma'lumot uchun so'rov. Yo'naltiradigani — panel, ko'rsatadigani —
  rels.
- ⚠️ **Ochiq guruhni sahifa hal qiladi**: qo'lda tanlash faqat marshrut
  o'zgargunicha yashaydi. Navigatsiyadan omon qolgan tanlov "Menyu" ro'yxatini
  ko'rsatib turar, orqasidagi ekran esa kassa hisoboti bo'lardi — sahifa bilan
  kelishmaydigan sidebar bitta ortiqcha bosishdan yomonroq.
- ⚠️ **Eng uzun moslik yutadi** (`groupOf`): `/admin` boshqa har bir
  marshrutning prefiksi, ya'ni birinchi moslikni olish hamma ekranni "Bugun"ga
  qo'yardi va rels dashboarddan boshqa hamma joyda noto'g'ri guruhni yoqardi.
- ⚠️ **Hammasi `ownerOnly` bo'lgan guruh relsdan ham yo'qoladi**: bo'sh ustun
  ochadigan ikonka menejerga navigatsiya ishonchsiz ekanini o'rgatadi.
- ⚠️ **Telefonda ikki qavat emas**, o'sha yassi guruhlangan ro'yxat: ochiladigan
  panel ichidagi rels+panel — hech nima o'qilmasidan oldin ikki bosish, va aynan
  panel bir qo'lda ochiladigan qurilmada. Ikkalasi ham **bitta** `NAV_GROUPS`
  dan o'qiydi, ya'ni ajrab keta olmaydi.
- ⚠️ **Guruh ikonkalari ekran ikonkalaridan alohida** (`GROUP_ICONS`): "Menyu"
  guruhiga "Menyu" ekranining kitobini berish ikki qavatni bitta takrorlangan
  ro'yxatga o'xshatadi — ya'ni ichma-ich sidebar bartaraf qiladigan chalkashlik.
- ⚠️ Rels yorlig'i **qirqilmaydi, o'raladi**: "Sozlamal…" — ikkala ishni ham
  bajarmayotgan yorliq.

### ⚠️ 5. Multikassa javob berdi: **PDF to'g'ri**
Pul birligi masalasi **hal bo'ldi**: `receipt_sum`, `receipt_gnk_receivedcash`,
`receipt_gnk_receivedcard` — **tiyinda**; `items[]` ichidagilar — **so'mda**.

Ya'ni `tiyinToSum` va uni muhrlagan test **to'g'ri** edi va o'zgartirilmaydi.

⚠️ **CORS savoli hali javobsiz.** Usiz kassa ekranidan to'g'ridan-to'g'ri
ishlash mumkinmi yoki relay shart — bilinmaydi. Relay allaqachon yozilgan,
ya'ni bu ishni to'xtatmaydi.
