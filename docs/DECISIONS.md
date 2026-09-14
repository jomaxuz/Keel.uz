# DECISIONS.md — xususiyatlar bo'yicha qarorlar va tuzoqlar

Bu fayl **`CLAUDE.md` ning davomi**: u yerda loyihaning doimiy qismi
(arxitektura, model, API guruhlari, konvensiyalar, umumiy tuzoqlar), bu yerda —
**har bir bo'limning o'z qarorlari**: nima uchun shunday qilingan, qaysi
muqobil rad etilgan va qaysi xato bir marta bo'lib o'tgan.

⚠️ **Har sessiyada o'qilmaydi.** Ish qaysi bo'limga tegsa — **o'sha bo'lim
o'qiladi** (`grep` bilan sarlavha bo'yicha). Sabab: bu matn ~150 KB, va uni
har suhbat boshida yuklash kontekstning yarmini hech kim so'ramagan bo'lim
haqidagi ma'lumotga sarflaydi.

Yozish qoidasi: **bitta fakt bitta joyda**. Bu yerdagi bo'limni `CLAUDE.md` ga
nusxalamang — ikki nusxa birinchi tahrirda ajraladi (bu darsning o'zi
"`composeOrder`" va "`saveStocktake`" bo'limlarida yozilgan).

---

### Brend va filial (ko'p brend / ko'p filial)
- **Kompaniya** (`restaurant` singleton) — valyuta, ijtimoiy tarmoqlar,
  mijozlar bazasi. **Brend** (`brand`) — menyu, nom, logo, sayt matnlari,
  tema, xizmat turlari. **Filial** (`branch`) — manzil, telefon, ish vaqti,
  yetkazish zonalari va narxi, stol xaritasi, kuryerlar, buyurtmalar.
- **Bitta brend + bitta filialli mijoz hech qanday murakkablikni ko'rmaydi**:
  almashtirgichlar umuman render qilinmaydi, so'rovlar filtrsiz ketadi.
  Bu — butun xususiyatning asosiy sharti.
- Backend linzasi: `internal/handlers/scope.go` → `Scope{BrandID, BranchID}`,
  `adminScope(r)` (`?brandId=`/`?branchId=`) va **`clampToAdmin`** — filialga
  biriktirilgan menejer URL orqali kengaya olmaydi.
- ⚠️ **Savatni pishira oladigan filial yaqinrog'idan ustun** (`bestBranch`,
  testda muhrlangan). Ilgari tugagan taom filial **tanlangandan keyin**
  tekshirilardi, ya'ni eng yaqin oshxonadagi bitta tugagan taom butun
  buyurtmani rad etardi — uch kilometr naridagi filialda hammasi bor bo'lsa
  ham. Mehmon lag'moni bor kompaniyadan "lag'mon tugadi" degan javob olardi.
  Afzallik **ataylab tor**: hech kimning yetkazish hududini kengaytirmaydi
  (har nomzod allaqachon o'z `maxKm`/zonasidan o'tgan), faqat shu manzilni
  olishga rozi oshxonalar orasidan qaysi biri chekni olishini o'zgartiradi.
  ⚠️ Hech biri uddasidan chiqmasa **baribir eng yaqini qaytariladi**, `nil`
  emas: aynan o'sha filialning nomi va stop listi "lag'mon tugadi" degan halol
  javobni beradi, `nil` esa uni "bu manzilga yetkazilmaydi" degan **boshqa va
  yolg'on** gapga aylantirardi.
- ⚠️ **Menyudagi "tugadi" bayrog'i qaysi filialniki** — javob mehmondan hali
  so'ralmagan savolga bog'liq: u menyuni ko'rmoqda, manzil esa keyin beriladi.
  Ilgari **standart** filialning (birinchi saralanganining) ro'yxati
  ishlatilardi — bu optimistik ham, pessimistik ham emas, **tasodifiy**, va
  ikki tomonga birdan xato edi: kompaniya yetkaza oladigan taomni yashirardi
  **va** mehmonning o'z filialida tugagan taomni taklif qilardi.
  Qoida (`soldoutlens.go`): filial **tanlangan** bo'lsa (stol QR'i, olib
  ketish filiali, sayt cookie'si) — o'shaniki; **bitta filial** bo'lsa —
  o'shaniki (ya'ni ko'pchilik uchun hech nima o'zgarmadi); **bir nechta filial
  va hech biri tanlanmagan** bo'lsa — faqat **hamma joyda** tugagan bo'lsa
  tugadi deb ko'rsatiladi. Bu — qaysi oshxona pishirishidan qat'i nazar rost
  qoladigan yagona gap. Optimistik tomoni ataylab, va u faqat checkout
  taomni nomi bilan ushlagani uchun arzon.
- ⚠️ **Tugagan taom checkout'da, tasdiqlashdan oldin aytiladi**
  (`/orders/quote` → `soldOut`). Tekshiruvning o'zi yangi emas — `CreateOrder`
  doim rad etardi — lekin u eng oxirida ishlardi, ya'ni mehmon ism, manzil va
  to'lov turini to'ldirib bo'lib "lag'mon yarim soat oldin tugagan"ni eshitardi.
  Bir joyda aytilsa — ma'lumot, ikkinchisida — behuda checkout. Ikkalasi ham
  bitta `soldOutAt` dan o'qiydi: savatni ma'qullagan sahifa va uni rad etgan
  buyurtma — ikkalasidan yomonroq javob.
- **Mijoz filialni tanlamaydi**: yetkazishda `deliveryBranch` manzilni qamrab
  oladigan filiallardan **eng yaqinini** tanlaydi; olib ketish va stolda esa
  mijoz qayerda turganini o'zi biladi (`branchId` so'rovda).
- **Buyurtmaning brendi savatdagi taomlardan** olinadi, brauzer yuborgan
  maydondan emas. Ikki brend aralashgan savat → 400.
- **Savat brend bo'yicha alohida**: `localStorage` kaliti `cart_v2:<brandId>`.
- Saytning linzasi **cookie'da** (`brand`, `branch`) — til bilan bir xil
  naqsh, chunki menyu server'da render qilinadi. `lib/siteBrand.ts` (client)
  va `lib/siteBrand.server.ts` (`getSiteScope()`).
- `GET /restaurant` javobiga filialning manzili/ish vaqti/yetkazish sozlamalari
  va brendning yuzi qatlanadi (`applyBrand`) — shu sabab sayt sahifalari
  o'zgarmadi. Panel uchun `?raw=1` (qatlamsiz).
- `PUT /admin/restaurant` — **qisman yangilash**: faqat yuborilgan maydonlar
  yoziladi. To'liq `$set` sozlamalar sahifasi endi yubormaydigan maydonlarni
  (brend/filialga ko'chganlarini) bo'sh qatorga aylantirib yuborardi.
- Panelda linza `lib/api.ts` dagi `setAdminScope()` da; `scope: true` bergan
  chaqiruvlarga avtomatik qo'shiladi. `AdminScopeProvider` uni **render
  vaqtida** beradi, `scopeKey` esa ekranni qayta yuklaydi.
- **"Tugadi"** (`branch.soldOut []`): menyu brendniki, qozonda nima qolgani
  filialniki. `isAvailable` — taom umuman yo'q; `soldOut` — shu filialda
  bugun tugadi. Bayroq public menyuda hisoblanadi (bazada yo'q), asosiy
  tekshiruv esa `CreateOrder` da, **filial aniqlangandan keyin**.
  Alohida endpoint `PUT /admin/branches/{id}/sold-out` (`$addToSet`/`$pull`);
  `AdminUpdateBranch` `soldOut` ni **ataylab yozmaydi** — ochiq turgan forma
  oshxona tugatgan taomlarni qaytarib qo'yardi. Kassadan kelgan stop list
  **alohida ro'yxatda** (`branch.posSoldOut`) — qarang §"Stop list".
- **Filial prefiksi** `branch.code` (A–Z0–9, 6 belgigacha) buyurtma va bron
  raqami oldiga qo'yiladi: `MRC-YS19-1745`. Bo'sh = prefiks yo'q.
- **Rollar**: `requireOwner` — kompaniya profili, brend CRUD, filial
  ochish/o'chirish; `requireBranchAccess` — menejer faqat o'z filialini.
  Panelda `branchId` qo'yilgan menejer uchun linza qulflanadi (almashtirgich
  o'rniga filial nomi) va kompaniya/brend bo'limlari ko'rinmaydi.
- **Brend modal oynada tanlanadi, filial esa ro'yxatchada**
  (`ScopeSwitcher`). Ikkalasi bir xil tor ro'yxatcha edi, savollari esa bir
  o'lchamda emas: filial — odam allaqachon qarab turgan biznesning bir xonasi,
  brend esa **butun biznes**, va noto'g'ri tanlash egani boshqa menyu, boshqa
  buyurtma va boshqa pulning ustiga olib boradi — ekranda esa bu haqda hech
  nima yozilmaydi. Shuning uchun modal ekran o'rtasini oladi va **har brendning
  o'z logosi** bilan chiziladi: ega o'z bizneslarini belgisidan taniydi,
  ro'yxatdagi tartibidan emas. Logo yo'q bo'lsa — nomning birinchi harfi
  (saytning `BrandMark` i bilan bir qoida, lekin **ko'chirma emas**: u
  mehmonning sahifasiniki va o'z o'lchamini olib yuradi).
- ⚠️ **Brend almashganda sahifa to'liq yangilanadi** (`location.reload()`).
  `ScopedMain` allaqachon `scopeKey` bo'yicha remount qiladi, lekin remount
  komponentni qayta quradi — **serverdan kelgan javobni emas**: `scope: true`
  bermagan har bir chaqiruv eski linzaning javobini qaytarib berardi. Eng
  ko'rinadigan qurboni brifing edi (pastga qarang), va alomati eng yomon
  shakldagi nosozlik: yangi brendning nomi ostida eski brendning kartalari,
  hech qayerda xato yo'q. Tanlov `localStorage` ga **reload'dan oldin**
  yoziladi, aks holda yangi sahifa egani endi tark etgan brendni o'qib oladi.
- **Saytda ham brend modal oynada tanlanadi** (`components/site/BrandSwitch.tsx`).
  Ilgari brendlar headerda yonma-yon tugma bo'lib turardi va har yangi brend
  navbar so'zlarining joyini olardi — uchtasida havolalar siqilib qolardi.
  Endi barda faqat **joriy brend** (logo + nom), ro'yxat esa logolari bilan
  oynada. Tanlanganda cookie yoziladi va **to'liq yuklash** bilan `/menu` ga
  o'tiladi (`location.assign`, `router.refresh()` emas: tema, menyu va savat
  kaliti brendga bog'liq). ⚠️ Oyna `<body>` ga **portal** qilinadi: header'da
  `backdrop-blur` bor, u esa `fixed` bolalarining konteyneriga aylanadi —
  portalsiz oyna 80px lik bar ichida chizilardi.
- ⚠️ **Brend o'chirish tugmasi hech qachon ishlamaydigan holatga tushib
  qolgandi.** Buyurtma ko'rgan filial o'chirilmaydi — **yopiladi** (cheklar
  javobsiz qolmasin), brend esa **hamma** filiallarini sanardi. Ya'ni ishini
  tugatgan ega filiallarni o'chirib, ularning o'chganini ko'rib, brendga
  bosardi va "avval filiallarini o'chiring" degan javobni olardi — endigina
  o'chirgan filiallari haqida. Keyingi qadam **yo'q** edi, bazadan boshqa.
  Endi qoida uch holatli (`brandDeleteAction`, testi bor): **ochiq filial bor**
  → rad etiladi; **faqat yopilganlari qolgan** → brendning o'zi ham
  **yopiladi** (filiallari kabi, va o'sha sabab bilan); **hech nima yo'q** →
  o'chiriladi. ⚠️ Javob **brendning o'z kartochkasida** chiziladi, sahifaning
  tepasida emas: to'rtta brend ochiq turganda tepa bir necha ekran naridadir,
  va o'sha yerdagi rad javobi "tugma umuman ishlamayapti" bo'lib o'qiladi —
  aynan shunday xabar qilingan.

### Chas pik: yuklama ko'rsatiladi, buyurtma qo'lda ko'chiriladi
- ⚠️ **Avtomatik qayta yo'naltirish ataylab yo'q.** Band filialdan boshqasiga
  o'tkazish ko'ringanidan yomonroq: keyingi filial mehmondan **uzoqroq**, ya'ni
  oshxonada tejalgan o'n daqiqa yo'lda yigirma bo'lib qaytadi va ovqat
  "balanslangani uchun" sovuqroq yetadi. Bundan tashqari u tez ishlaydigan
  filialga hammaning ishini yuklab jazolaydi, va bir xil savat besh daqiqa
  oralatib ikki oshxonaga tushadi — mehmon ko'ra olmaydigan sabab bilan.
  Juma kuni soat yettida ko'chirish kerakmi — bu **nechta kuryer yo'lda**
  ekaniga bog'liq, va buni hech qanday chek sanog'i bilmaydi.
- Shuning uchun panelga qaror uchun kerak bo'lgan ikki narsa berilgan:
  `GET /admin/branches/load` (har oshxonada nechta chek pishmoqda, nechtasi
  qabul qilinmagan, eng eskisi necha daqiqa) va `PUT /admin/orders/{id}/branch`.
  Yuklama satri **faqat bir nechta filial bo'lganda** chiziladi.
- **Eng foydali raqam — eng eskisining yoshi**, chek soni emas: endigina qabul
  qilingan beshta chek oddiy kecha, qirq daqiqa kutgan bittasi esa yo'q.
- Qo'lda ko'chirishning qoidalari (`AdminMoveOrderBranch`):
  **pul o'zgarmaydi** (mijoz bilan kelishilgan — manzilni qo'lda tuzatishdagi
  bilan bir qoida), **buyurtma raqami eski prefiksda qoladi** (u chekda va
  mehmonning kuzatuv havolasida), **kuryer bo'shatiladi** (kuryer filialniki),
  **`readyAt` tozalanadi** (yangi oshxona uni pishirmagan), va **taomi tugagan
  filialga ko'chirib bo'lmaydi** (409, taom nomi bilan). Har ko'chirish
  jurnalga tushadi (`order.branch`) — "nega Chilonzor buyurtmasi Sergelida
  pishmoqda" savoli nom bilan javob talab qiladi.

### Call-markaz (`/admin/calls`)
- **Alohida rol yo'q**: telefon ko'targan odam ikki daqiqadan keyin o'sha
  buyurtmani tasdiqlaydigan odamning o'zi. `owner` ham, `manager` ham ko'radi.
- **Bitta so'rov, bitta javob** (`GET /admin/lookup?phone=`): operatorda gap
  boshlagunicha bir necha soniya bor. Mijoz, hozir oshxonadagi buyurtmasi,
  doim buyurtma qiladigan taomlari, manzillari, bronlari, javobsiz shikoyati
  va oldingi qo'ng'iroqlari — hammasi birga. Uch ekrandan yig'ish — mijozdan
  o'z manzilini so'rashning yo'li.
- Kartochkadagi **bloklar tartibi ma'lumot tartibi emas, o'qish tartibi**:
  javobsiz shikoyat → jarayondagi buyurtma → kimligi va odati → tarix.
  Restoran uzr aytishi kerak bo'lgan odamga xushchaqchaq salom bermaslik uchun.
- Buyurtmalar **`customer.phone` bo'yicha ham** topiladi, faqat hisob bo'yicha
  emas: ro'yxatdan o'tmasdan buyurtma bergan yoki xotinining telefonidan
  qo'ng'iroq qilgan odam ham tanilishi kerak.
- **Telefon orqali buyurtma — bitta quvur**: `CreateOrder` ning ichi
  `composeOrder` ga ajratilgan, sayt ham, operator ham o'shani yuritadi.
  Nusxa ko'chirilganda ikkisi ajrab ketardi va bir xil savat telefonda boshqa
  narx bilan chiqardi. Farqi faqat `takenBy` — chekdagi yorliq, qoida emas.
  Operator minimal buyurtmadan pastga ham sota olmaydi: uning ishi buyurtmani
  **qabul qilish**, kelishish emas.
- **Yangi mijoz hisobi ochiladi**, lekin `authProvider: "operator"` bilan —
  qo'ng'iroq qilgan odam hech nimani isbotlamadi. Saytga birinchi kirganda
  baribir SMS'dan o'tadi. Operator yozgan manzil profilga qo'shiladi (takror
  emas — matn yoki ~50 m yaqinlik bo'yicha tekshiriladi).
- **`POST /admin/orders/quote`** kerak, chunki mijoz *nomlanadi*, token bilan
  kelmaydi: ballari va "faqat birinchi buyurtma" kodlari hisobga bog'langan.
  Jami summani ayta olmagan operator uni taxmin qiladi, taxmin qilingan summa
  esa eshik oldida janjal.
- **Jurnal tahrirlanadi** (amallar jurnalidan farqli): qo'ng'iroq odam
  gapirayotganda yoziladi, natija ko'pincha bir daqiqadan keyin aniq bo'ladi.
  Tuzatib bo'lmaydigan jurnalni operator ikkinchi kundan to'ldirmay qo'yadi.
- **Ikki filtr — jurnalning butun ma'nosi**: "kimga qayta qo'ng'iroq qilish
  kerak" va "qaysi qo'ng'iroq buyurtmaga aylandi". Shuning uchun ular tugma,
  ochiladigan ro'yxat ichida emas. Qayta qo'ng'iroqlar **eng yaqini birinchi**
  bo'lib saralanadi va kechikkani belgilanadi — muddati ko'rinmaydigan va'da
  va'da emas.
- **Vaqtsiz "qayta qo'ng'iroq" qabul qilinmaydi** (400): muddatsiz yozuv
  ro'yxatga tushadi-yu hech qachon kelmaydi.
- Ochiq va'dalar **butun jurnal bo'yicha** sanaladi, tanlangan davr bo'yicha
  emas: o'tgan seshanbadagi va'da bugun ham qarz.
- Qo'ng'iroqlar hozircha **qo'lda** yoziladi — ATS integratsiyasi yo'q.
  Mijozda ATS bo'lsa, `POST /admin/calls` tayyor tayanch nuqta.

### Onlayn to'lov: Payme / Click / Uzum / ATMOS
- **To'rtta provayder, bitta shakl.** Har biri mijozni o'z sahifasiga olib
  boradi, pulni oladi va **serverga qo'ng'iroq qilib** aytadi. Restoranni
  himoya qiladigan hamma narsa `handlers/payments.go` da — uch marta yozilgan
  qoida ikki marta yozilgan qoida.
- **Summa buyurtmaniki, callback'niki emas**: 100 000 so'mlik buyurtma uchun
  "1 000 to'landi" degan chaqiruv rad etiladi.
- **To'landi deb belgilash idempotent** — uchalasi ham qayta urinadi (Payme'ning
  takroriy `PerformTransaction` i oddiy trafik). `(provider, providerTxnId)`
  unique indeksi va holat bo'yicha qo'riqlangan `UpdateOne`.
- **Brauzer hech nimani isbotlamaydi**: "muvaffaqiyatli" URL bilan qaytish
  holatni o'zgartirmaydi, faqat serverdan serverga chaqiruv o'zgartiradi. Odam
  to'lab tabni yopsa qaytish umuman bo'lmaydi.
- **To'lanmagan buyurtma oshxonaga tushmaydi.** `queuedAt` — "bu buyurtma
  qachondan oshxonaniki": naqd uchun yaratilgan payt, onlayn uchun bank
  tasdiqlagan payt. Yangi buyurtma jiringlashi shunga qaraydi, aks holda
  oshxona hech qachon to'lanmasligi mumkin bo'lgan chekka chaqiriladi — va pul
  kelganda **chaqirilmaydi**, chunki buyurtma allaqachon eski.
- **Kalitlar `payment_settings` da** (§4), panelga ham qaytarilmaydi.
  **Bo'sh kalit = "saqlangani qolsin"**, "o'chir" emas.
- **Yarim sozlangan provayder taklif qilinmaydi**: `GET /payment-methods` faqat
  yoqilgan **va** to'liq kalitli tizimni qaytaradi — bank xato sahifasiga olib
  boradigan tugma buyurtmani yo'qotadi va mijoz restoranni ayblaydi. Naqd
  har doim ro'yxatda.
- **Payme**: `Basic Paycom:<kalit>`, tiyinda, 12 soat timeout (sabab 4). Vaqt
  belgilari **aynan** qaytariladi — shuning uchun daftarda millisekund
  saqlanadi. Yetkazilgan buyurtmani bekor qilish `-31007` bilan rad etiladi.
- **Click**: `md5(click_trans_id + service_id + SECRET_KEY + merchant_trans_id +
  [merchant_prepare_id, faqat Complete] + amount + action + sign_time)`.
  ⚠️ **Ikki formula bir xil emas** — bittasini ikkalasiga ishlatsangiz Prepare
  o'tadi, Complete yiqiladi va bu Click uzilishiga o'xshaydi. Summa so'mda,
  kasr bilan (`78000.00`).
- **Uzum**: Basic auth, tiyinda, `status` satri + raqamli `errorCode`.
- **Buyurtma raqami — hisob (account)**: maydon nomi sozlanadi
  (`accountField`, standart `order_id`); raqam chekda turadi va avtomatik oqim
  ishlamaganda odam uni qo'lda kiritadi.
- **ATMOS** (`handlers/payatmos.go`) — boshqalardan ikki joyda farq qiladi:
  - ⚠️ **Havola quriladi emas, so'raladi** (`POST /checkout/invoice/create` →
    `checkout.atmos.uz`). Ya'ni yagona provayder bo'lib, havolasi **tarmoq
    sababidan** yiqilishi mumkin: bunda bo'sh qator qaytariladi (tugma
    ko'rsatilmaydi) va sabab **logga** yoziladi — mehmon "STORE_NOT_FOUND"
    bilan hech nima qila olmaydi.
  - ⚠️ **Callback xabar emas, ruxsat**: pul faqat bizdan muvaffaqiyatli status
    kelgandan keyin yechiladi. Xato bilan `status: 0` qaytarish yozuvni
    yo'qotmaydi, **haqiqiy to'lovni kassada rad etadi**.
  - ⚠️ **Imzoning hash funksiyasi hujjatda yozilmagan** (faqat formula:
    `store_id+transaction_id+invoice+amount+api_key`, ajratgichsiz) → md5/sha1/
    sha256 ning uchalasi qabul qilinadi. Zaiflik emas: satr ichida `api_key`
    bor. Qaysi biri mos kelgani logga yoziladi.
  - ⚠️ **Maydon turlari ham yozilmagan**: `store_id`, `transaction_id`,
    `amount` qo'shtirnoq bilan ham, bo'lmasa ham keladi (`atmosScalar`).
    Matn **aynan kelgan holida** saqlanadi — imzo shuning ustidan.
  - **Uchta sir, uchta bayroq**: OAuth juftligi va **alohida** `apiKey`
    (callback kaliti). Noto'g'ri OAuth = havola yo'q; noto'g'ri callback kaliti
    = mehmon to'laydi, tasdiq rad etiladi — yomonroq va hech kim tekshirmaydi,
    shu sababli formada oxirida va alohida nomlangan.
  - **PAN bizga tegmaydi**: `/merchant/pay/*` karta raqamini oladi va har
    tenantni **PCI DSS qamroviga** kiritardi — hosted invoice ataylab tanlangan.
  - Summa **tiyinda**, savat majburiy (fiskal chekka ketadi). `code` — ИКПУ;
    taomda yo'q bo'lsa **yuborilmaydi** (noto'g'ri ИКПУ = noto'g'ri fiskal chek).
  - Sinov: `go run ./cmd/paytest -order <№> -provider atmos -suite`.
    ⚠️ Vosita bir marta aldadi: route qo'shilmaganda `404 page not found`
    javobi `status: 0` bo'lib o'qilardi va **hamma rad etish testi yashil
    chiqardi**. Endi JSON bo'lmagan javob nosozlik.
- Panel: `/admin/settings` → "To'lov tizimlari" (`PaymentsEditor.tsx`).
  Ekranning yarmi — **kabinetga yoziladigan manzillar**, nusxalanadigan qilib.

### Telefoniya: onlinePBX
- **ATS'siz jurnal yoziladi, ATS bilan — o'zi yoziladi.** Kiruvchi qo'ng'iroq
  kelganda operator ekranida mijoz kartochkasi **o'zi ochiladi**, jurnalga
  raqam, yo'nalish, davomiylik, yozuv va kim javob bergani avtomatik tushadi.
  Operator faqat natija va izoh yozadi — bu yagona qism odam biladigan.
- **Beshta hodisa bitta yozuvga tushadi** (`pbxCallId` unique, sparse).
  Hodisalar tartibsiz va takror keladi; `upsert` ikkalasini ham zararsiz qiladi.
- **Hodisa odam yozganini hech qachon o'chirmaydi.** Natija, izoh va qayta
  qo'ng'iroq — operatorniki. Kech kelgan `call-end` yoki qayta yuborilgan
  `call-start` ularni bo'shatib yubormaydi (`$setOnInsert`).
- **Webhook manzilining o'zi kalit**: onlinePBX hech qanday parol yubormaydi,
  shuning uchun manzildagi token generatsiya qilinadi va almashtirilishi
  mumkin. Noto'g'ri token **200 bilan jimgina** rad etiladi — 401 qaytarish
  skanerga "topdim" deb aytish bilan barobar.
- **Har doim 200 qaytariladi**: xato olgan ATS qayta uradi, va tushunmagan
  payload uchun qayta urinish bo'roni hech kimga yordam bermaydi.
- **Kalit uch kun yashaydi va keshlanadi.** Hujjatda ochiq yozilgan: sekundiga
  to'rt-besh marta avtorizatsiya sessiyalarni buzadi, chunki har avtorizatsiya
  yangi kalit berib eskisini o'ldiradi. Faqat `isNotAuth` kelganda yangilanadi.
- **`call/now.json` da tartib muhim**: `from` — **operatorning** ichki raqami,
  chunki ATS avval o'shani jiringlatadi. Teskari qilinsa, mijoz operator
  garnituraga uzanguncha kutib turadi.
- **Yozuv havolasi saqlanmaydi** — onlinePBX ularni imzolaydi va saqlangan
  havola bir kun ishlamay qoladi, bu esa yo'qolgan yozuvga o'xshaydi. Havola
  "tinglash" bosilganda so'raladi.
- **Davomiylik `dialog_duration`dan**, `call_duration`dan emas: qirq soniya
  jiringlab javob berilmagan qo'ng'iroq — nol soniyalik suhbat.
- **`lastEventAt` sozlamalar sahifasidagi eng foydali qator**: kalitlar
  to'g'ri bo'lsa ham manzil onlinePBX paneliga yozilmagan bo'lishi mumkin, va
  ulanish tekshiruvi buni **umuman ko'rsata olmaydi**.
- Panel har 3 soniyada `GET /admin/calls/live` so'raydi (soket emas — bu panel
  hamma joyda shunday qiladi, `AlertBell` kabi). Ekran operator boshqa
  qo'ng'iroqni yozayotgan bo'lsa **egallab olinmaydi**: yarim yozilgan izohni
  yo'qotish, kartochka ochilmaganidan yomonroq.
- ⚠️ **Webhook maydon nomlari** onlinePBX'ning integratsiya hujjatidan olingan,
  rasmiy OpenAPI spetsifikatsiyasida ular yo'q. Shuning uchun payload
  **bardoshli o'qiladi**: `caller`/`caller_id_number` kabi ikkala yozilish ham
  qabul qilinadi, JSON bo'lmasa form-encoded sinaladi, tushunilmagani esa
  tashlanmaydi. Haqiqiy mijozda bir marta tekshirish kerak.

### Bot javob berishi — ikkinchi yarim (webhook)
- ⚠️ **Bot ikki yarimdan iborat, va biz faqat birini qurgan edik**: biz
  Telegram'ga chaqiramiz (buyurtma xabarlari), Telegram esa bizga. Ikkinchisi
  bo'lmasa "Start" bosgan mijoz **jim javob** oladi — va bu "restoranning boti
  buzuq" deb o'qiladi. Panelda alomat yo'q edi: bizning tomonda hech nima
  yiqilmagan.
- **Webhook, polling emas**: bir restoran = bir konteyner, polling har tenantni
  Telegram bilan doim ochiq so'rovda ushlab turardi — mijozi bor-yo'qligidan
  qat'i nazar.
- ⚠️ **Manzilning o'zi kalit** (`telegram_settings.webhookToken`): Telegram
  parol yubormaydi. Token generatsiya qilinadi, almashtiriladi va
  `subtle.ConstantTimeCompare` bilan solishtiriladi; Telegramning o'z
  `secret_token` sarlavhasi ham tekshiriladi. onlinePBX bilan bir naqsh.
- **Har doim 200**, rad etish **jimgina**: 401 skanerga "topdim" deb aytish
  bilan barobar, xato esa qayta urinish bo'roniga olib keladi.
- **Webhook "Ulanishni tekshirish"da ro'yxatdan o'tadi** va har tekshiruvda
  qayta yoziladi (o'z domenini keyin ulagan egada webhook eski domenda qolardi).
  Manzil `PUBLIC_BASE_URL` dan quriladi, **so'rovdan emas** — panel IP yoki
  tunnel orqali ochilgan bo'lishi mumkin.
- ⚠️ **`lastUpdateAt` — sahifadagi eng foydali qator**: tekshirish tugmasi
  *bizning* Telegram'ga yetishimizni isbotlaydi va Telegramning bizga yetishini
  **ko'rsata olmaydi**. Bayroq emas, **vaqt belgisi**: saqlangan bayroq soat
  undan o'tishi bilan eskiradi.
- ⚠️ **Tugmaning ikki turi sinaladi** (`SendMenu`): `web_app` tugmasi mini
  app'ni Telegram ichida ochadi, lekin uni qabul qilish @BotFather'dagi
  sozlamaga bog'liq va rad etilganda **butun xabar** ketmaydi (bot yana jim).
  Zaxira — oddiy `url` tugmasi. Qaysi biri ishlagani **logga** yoziladi.
- **`/start <payload>` — chat deep link'i**, mini app'ning `startapp` idan
  boshqa parametr. `t_<id>` shu yerda yechiladi va saytning `?table=`
  mexanizmiga aylanadi; qiymat **hex id sifatida tekshiriladi**.
- **Xabar fonda yuboriladi**: sekin yuborish "Start"ning qayta yetkazilishiga
  aylanardi — mijoz ikki salom olardi.
- ⚠️ **Restoran nomi brenddan olinadi** (`restaurantName(ctx)`) — o'sha
  tuzoqning **uchinchi** ko'rinishi (eksport fayl nomi, keel.uz hamkorlar
  lentasi): `notifyOrderStatus` `restaurant.name` ni yolg'iz o'qiganda brendi
  bor har tenantda xabarlar **"Restoran"** deb imzolanardi.

### Sayt konstruktori: erkin canvas (konsoldan)
- Panjara qoldi, ustiga **erkin blok** qo'shildi (`canvas`, `popup`): band ichida
  elementlar **foizda** joylashtiriladi (`DesignBox{x,y,w,h,z}`), balandlik `vh`.
  Pikselda emas — `x=340px` 380 px'li telefonda hech qanday joyga tushmaydi.
- ⚠️ **Telefon joylashuvi alohida** (`element.mobile`). Desktop kompozitsiyasini
  telefonga aylantiradigan arifmetika yo'q. Hech bir element telefon
  joylashuviga ega bo'lmasa band **oqim**ga tushadi (chizilgan tartib, to'liq
  kenglik) — aynan shu narsa "chiroyli, lekin telefonda buzuq" natijani
  to'xtatadi. Yarim chizilgan telefon joylashuvi bo'lmaydi: qaror **band**
  darajasida.
- ⚠️ **Ishlaydigan vidjetlar** (`widget-menu`, `widget-hours`…) canvas'da faqat
  joylashadi, ichi o'zgarmaydi: narx, tarjima, variantlar, savat — hammasi
  mavjud kod. Aks holda tahrirlagich buyurtma qabul qila olmaydigan chiroyli
  sahifa yasaydi.
- ⚠️ **`Sanitize` — xavfsizlik chegarasi, va u tenant tomonida o'qishda
  ishlaydi.** Shuning uchun konsol elementlarni maydon-maydon takrorlamaydi
  (`Canvas any` bo'lib o'tadi): drift bo'lganda drift qiladigan nusxa doim
  chegara bo'lmagani bo'ladi. Notanish element turi tashlanadi, har son
  qisiladi, rasm faqat `/uploads/`, havola faqat saytning o'z sahifalari.
- ⚠️ **`customCss` — fayldagi yagona erkin maydon**, va u faqat konsoldan
  yoziladi (ega unga yeta olmaydi). `sanitizeCSS` shakl bo'yicha whitelist:
  `</style`, har qanday `<`, `javascript:`, `expression(`, `@import`, begona
  `url(` — rad etilsa **butun matn** bo'shatiladi (yarim olib tashlangan qoida
  hech kim yozmagan stil). Testda muhrlangan.
- **Jonli ko'rinish kalit bilan**: konsol tenant bazasiga 2 soatlik token yozadi
  (`design_preview`, TTL indeks), sayt `?preview=<token>` bo'lsa **qoralamani**
  chizadi. Muddat kodda ham tekshiriladi — Mongo TTL sweep'i daqiqada bir
  ishlaydi, ya'ni "hujjat yo'q" ≠ "kalit haqiqiy". Token bitta brendga tegishli
  va hech qanday huquq tashimaydi.
- Tahrirlagich **alohida sahifa**: `/console/tenants/{id}/design`.

### Xatolik hisobotlari: konsolga avtomatik tushadi

**Bu — qo'llab-quvvatlash navbatining ikkinchi uchi.** U restoran sezishiga,
"aytishga arziydi" deb qaror qilishiga va tushuntira olishiga bog'liq — va shu
uch qadamning har biri xatoliklarni yo'qotadi: bitta planshetdagi bitta
kassirniki, odamlar aylanib o'tib ketadigani, va hech kim so'z bilan ifodalay
olmaydigani. Bu yo'lda o'sha qadamlarning **hech biri yo'q**.

**Oqim:** ilova → **o'z tenant serveri** → control plane → konsol
(`/console/reports`).

- ⚠️ **Nega tenant serveri orqali?** Hisobot konsolga **kim ekani aniqlangan**
  holda yetib boradi: server allaqachon ushlab turgan tenant kaliti bilan
  uzatiladi (brifing va domen bog'lash ishlatadigan o'sha kalit). To'g'ridan-to'g'ri
  yuboradigan ilova platforma kalitini ko'tarib yurishi kerak bo'lardi — brauzerda,
  kuryer telefonida, Windows o'rnatgichi ichida — va u qaysi restoran ekani
  haqidagi da'vosiga **ishonish** kerak bo'lardi. Ro'yxat esa dalil sifatida
  o'qiladi.
- ⚠️ **Buning narxi ochiq aytiladi: konteyner o'chiq bo'lsa, hech nima
  yetib bormaydi.** Aynan shu nosozlikni konsol Docker'dan **jonli** o'qiydi
  (`attention: "down"`), va ikkisi ataylab boshqa mexanizm: **quvur o'z
  yo'qligi haqidagi xabarni ko'tara olmaydi.**
- ⚠️ **Guruhlangan, oqim emas.** Bitta render sikli — bir tushlikda o'n ming bir
  xil xato, va o'n ming qatorli ro'yxat hech qanday savolga javob bermaydi.
  Barmoq izi (fingerprint) `message` + `where` dan, ichidagi **raqamlar, id'lar
  va tirnoq ichidagi qiymatlar olib tashlanib** olinadi: "order 6f3a not found"
  va "order 91bc not found" — bitta nosozlik.
  ⚠️ **Ataylab qo'pol, birlashtirish tomonga og'gan**: bitta qatorga qo'shilib
  ketgan ikki xato ochilgan zahoti ko'rinadi (ikki xil stek), to'rt yuz qatorga
  bo'linib ketgan bitta xato esa **umuman topilmaydi** — birortasi ham muhimga
  o'xshamaydi. Narx nosimmetrik, qoida ham shunday.
- ⚠️ `(slug, app, key)` **unique** — qabul `upsert`. Indekssiz bir soniyada
  kelgan ikki hisobot bitta nosozlikka ikki guruh yaratadi va shundan keyin
  hisob ikki qatorga bo'linadi: ro'yxat nosozliklar ro'yxati bo'lishdan to'xtab,
  hodisalar ro'yxatiga aylanadi.
- **Bugungi hisob alohida saqlanadi** (`today`/`todayOn`): "qirq marta" bir
  tushlikda va uch oyda butunlay boshqa narsa, ro'yxat esa **hozir** nima
  bo'layotgani bo'yicha saralanadi. **Qurilmalar soni** ham (`users`): bitta
  buzuq planshetdagi bitta kassir va zanjirning hamma kassiri — bir xil hisob,
  butunlay boshqa ertalab.
- **Kunlik chek**: bir restoran kuniga 40 ta **yangi** nosozlik ocha oladi.
  ⚠️ **Chekdan oshgani hisobni to'xtatmaydi** — mavjud guruhlar sanashda davom
  etadi, faqat yangi guruh va yangi namuna yaratilmaydi. Hisobni ham
  to'xtatish bo'ronni **tugagandek** ko'rsatardi.
- **Namuna 5 tadan** (`$slice: -5`). ⚠️ Manfiy — **eng yangilarini** saqlaydi.
  Musbat eskilarini saqlaydi, va bu sinovda **aynan bir xil** ko'rinadi:
  namunalar bor, shunchaki shakli o'zgarib ketgan nosozlikning birinchi
  soatidan.
- ⚠️ **"Tuzatildi" yig'ishni to'xtatmaydi va hech nimani o'chirmaydi.** Guruh
  sanashda davom etadi, va tuzatilgandan keyin **yana** sanay boshlagan guruh —
  bu ekrandagi eng foydali qator: tuzatish ushlamagan. Tuzatilgan paytdagi hisob
  saqlanadi (`resolvedCount`), ya'ni "qaytdimi?" — arifmetika, kimningdir
  xotirasi emas.

**Mijoz tomoni (`frontend/src/lib/report.ts`)**

- ⚠️ **Endpoint autentifikatsiyasiz** (`POST /api/v1/report`): **buzilgan
  sessiya haqidagi hisobotdan ishlaydigan sessiya talab qilib bo'lmaydi.** Eng
  qimmatli hisobotlar aynan ilova normal holatda bo'lmagan paytdan keladi —
  yangilanmagan token, login'dan oldin yiqilgan ekran. Auth ortiga qo'yish
  **aynan shu sinfdan boshqa hammasini** yig'ardi. O'rniga: router'dagi IP
  gate (login va SMS turadigan o'sha), qat'iy hajm cheklari, va platformada
  yana bir chek.
- ⚠️ **Hech qachon vaziyatni yomonlashtirmaydi**: 202 qaytaradi, so'rovdan
  ajratilgan (`context.WithoutCancel`), navbatga qo'yiladi, jim yiqiladi.
  Xatolik ishlovchisi ichida otilgan xato — bu bitta buzuq ekranni, ishlovchisi
  ham buzuq ekranga aylantiradi, eng kam sinaladigan kodda.
- ⚠️ **Ikkala hodisa ham tinglanadi**: `error` **va** `unhandledrejection`.
  `.catch` siz rad etilgan promise `window.onerror` ga **umuman yetib
  bormaydi**, bu kodda esa nosozliklarning ko'pi — kutilgan API chaqiruvlari.
- ⚠️ **`sendBeacon` birinchi**: aytadigan gap paydo bo'ladigan eng keng tarqalgan
  lahza — sahifa ketayotgan lahza (mehmon tabni yopdi). O'sha yerda boshlangan
  `fetch` ni brauzer bekor qiladi.
- ⚠️ **Sahifaga 8 ta nosozlik chegarasi.** Render sikli buni har kadrda
  chaqiradi; chegarasiz birinchi buzuq komponent tab ochiq turgancha uzluksiz
  yuboradi — restoran o'ziga o'zi qilgan DoS.
- ⚠️ **`global-error.tsx` qo'lda, import'siz.** U fayl "React'dan boshqa import
  yo'q" qoidasiga bo'ysunadi, chunki u aynan modul grafi sog'lom bo'lmagan holat
  uchun bor. `report.ts` esa API klientni, u token do'konini import qiladi.
  O'n qator ataylab takrorlangan — **takrorlanish shu yerda maqsad**.
- ⚠️ **Versiya ilovaniki, tenant serveri hech nima muhrlamaydi.** Bu serverning
  o'z versiya konstantasi yo'q (faqat control plane'da bor), va platformanikini
  qo'yish har hisobotda **noto'g'ri binarni** nomlardi — savol esa "deploy
  tuzatdimi?", va o'sha deploy ilovaniki.
- ⚠️ **Mehmon ma'lumoti ham, kalit ham olib yurilmaydi**: xabar, stek, sahifa,
  **rol** (ism emas), versiya. So'rov tanasi ham, sarlavhalar ham hech qachon —
  telefon raqamini ushlay oladigan maydon oxir-oqibat ushlaydi.

`site/error.tsx` dagi izoh **teskarisini** aytardi ("markazlashgan joyga
yuborish har mehmon brauzerini bizning mijozimizga aylantiradi, va biz uni
qo'yadigan joy yuritmaymiz"). Ikkinchi yarmi endi rost emas, birinchisiga esa
javob berildi — mehmon brauzeri **restoranning o'z serveriga** yuboradi.
Izoh o'rnida to'g'rilandi.

**Native ilovalar uchun shartnoma.** Windows kassa/zal, Android ofitsiant va
keyinchalik iOS shu repoda emas. Ular uchun `POST /api/v1/report` shartnomasi:
`{"reports":[{app,message,stack,where,context,branch,role,version,platform,session,at}]}`,
auth yo'q, javob doim 202. `app` — `till|waiter|kitchen|courier|panel|site`.

### Qo'llab-quvvatlash: chat va operator konsoli

**Suhbat Keel konsolida saqlanadi, har bir tenantning bazasida emas.** Operator
ertalab o'ttizta restoranga javob beradi — har birining serverida saqlansa,
bitta ro'yxatni chizish uchun o'ttizta bazaga kirish kerak. ⚠️ Va **konteyneri
o'chgan restoran** — aynan bizga yozadigan restoran — yordam so'ray olmaydigan
restoran bo'lib qolardi.

**⚠️ Restoranning brauzeri konsolga umuman chiqmaydi.** Panel o'z serveriga
yozadi, u esa brifing va domen uchun allaqachon ushlab turgan tenant tokeni
bilan uzatadi. Mijozning domenidan berilgan sahifa platforma kalitini olib
yurmaydi — bu yerdagi hamma narsa shu chegarada.

**⚠️ Jonli kanal: nudge'li long-poll, ikki servis orqali o'tkazilgan socket
emas.** Panel **o'z serveriga** WebSocket ushlaydi (bir xil origin, CORS yo'q,
kalit yo'q), u server esa konsolda bitta so'rovni ochiq ushlab turadi. Javob
kelganda o'sha slug'dagi hamma kutuvchi darhol uyg'onadi. Nudge kanalga
yozmaydi, **yopadi**: yozish uchun hali o'qiyotgan tomon kerak, va aynan muhimi
— bir lahza oldin bekor qilingan so'rov.

**⚠️ Socketni bir martalik chipta ochadi, sessiya tokeni emas.** Brauzer
WebSocket handshake'ga header qo'ya olmaydi (`new WebSocket(url)` umuman header
olmaydi), ya'ni qolgani query string yoki cookie. Bir haftalik sessiya tokenini
URL'ga yozish — uni har bir access log va oldidagi har bir proxy'ga yozish. 30
soniyada yonadigan va bitta socket ochadigan chipta yozib olingan paytda
qadrsiz.

**⚠️ Socket manzilini server aytadi — `rewrites()` tuzog'ining yangi joyi.**
Prodda panel va API bir originda, brauzer manzilni o'zi qura olardi. Dev'da esa
`/api/*` backend'ga Next rewrite orqali boradi, va rewrite HTTP'ni uzatadi,
WebSocket'ni **upgrade qilmaydi** — ya'ni `location` dan qurilgan manzil faqat
dev'da yiqiladi, xato esa widget kodidagi bugga o'xshaydi. Manzil chipta bilan
birga qaytadi. Origin tekshiruvi ham `CORS_ORIGINS` ni qayta ishlatadi: faqat
dev'da yiqiladigan qoida kimdir tunda bo'shashtiradigan qoida.

**⚠️ Qidiruvda text index ishlamaydi va bu tuzatilgan xato.** Mongo'ning text
qidiruvi butun tokenlarni ichida o'zbekcha bo'lmagan stemmer bilan solishtiradi.
O'zbek tili agglyutinativ: operator "printer" deb yozadi, xabarda esa
"printerdan", "printerga", "printerni". Bular to'rt xil token va hech biri mos
kelmaydi — qidiruv ikki qator pastda turgan suhbatni topa olmaydi. Bu eng yomon
turdagi nosozlik: suhbat yo'qdek ko'rinadi, buzuq qidiruvdek emas. O'rniga so'z
boshiga bog'langan regex; skan, lekin support suhbatlari butun platformada
minglar bilan o'lchanadi va **tez, lekin noto'g'ri qidiruv — qidiruv emas**.

Yana:
- Thread **slug bilan ham** cheklangan — bir restoran boshqasining suhbatiga
  yoza olmaydi.
- Yopilgan suhbat mijoz yana yozganda **qayta ochiladi**: har savolga yangi
  thread bitta muammoni ikkiga bo'ladi va yopgan operator javob ishlamaganini
  bilmaydi.
- O'qilmagan xabar **har tomon uchun alohida** sanaladi: bitta bayroq
  "restoranda javob kutyaptimi?" va "navbatda ish bormi?" degan ikkala savolga
  javob bera olmaydi.
- Sarlavha birinchi jumladan **so'z chegarasida** kesiladi — yarim so'zlardan
  iborat navbat ikki marta o'qiladi.
- Konsolda **javob berish va yopish bitta bosish**: alohida yopish tugmasi
  navbatning yarmini abadiy ochiq qoldiradi.
- ~~Konsol tabida rol tekshiruvi yo'q~~ — **2026-09-14 da bekor qilindi**:
  yordam va xatoliklar endi faqat `owner` va `support` roliga ko'rinadi (qarang
  «Konsol xodimlari»). Alohida tex-support roli paydo bo'lgach «kim stolda
  bo'lsa, o'sha javob beradi» degan sabab qolmadi, sotuvchi texnik savolga
  bergan javob esa platforma bajarishi kerak bo'lgan va'da bo'lib qoladi.
- Operator ekranida **mijoz kartochkasi javob yozilayotgan paytda ko'rinadi**,
  jumladan konteyner holati: "panel bo'sh" va "panel o'chgan" — mijozdan bir
  xil jumla, bizdan butunlay boshqa javob.

**Savollar bazasi (FAQ) — `frontend/src/lib/help/`**

⚠️ **Maqolalar panel bilan birga yuboriladi, platformadan olinmaydi.** Ravshan
yechim — ularni konsolda saqlash, shunda javobni o'ttizta konteynerni qayta
chiqarmasdan tuzatish mumkin. U ikki jihatdan noto'g'ri: maqola **shu
build** nima qilishini tavsiflaydi, va platformadan olinadigan javob ertami-kech
restoran ishlatmayotgan versiya haqida gapira boshlaydi — ikkinchidan, yordam
eng kerak bo'lgan payt aynan konteyner hech qayerga chiqa olmaydigan payt.
Bundle ichida qidiruv internetsiz ishlaydi va har doim odam ko'rib turgan
dasturni tavsiflaydi.

⚠️ **Noto'g'ri javob umuman javob yo'qligidan yomon.** Hammasi shu
repozitoriydagi xatti-harakatdan yozilgan: PIN bloki besh daqiqa, chunki
konstanta shunday; kod sahifasi matndan tanlanadi, chunki enkoder shunday
qiladi. Ular o'zgarganda maqola **o'sha commit'da** o'zgaradi.

⚠️ **Qidiruvda ikkala yo'nalish ham kerak** (`help/search.ts`). Maqolada
"chekda", odam "chek" deb yozadi — maqolaning so'zi uzun. Maqolada "til", odam
"tilida" deb yozadi — so'rovniki uzun. Faqat birinchisini tekshirish butun
savolni beshta so'zining bittasida yiqitadi.

⚠️ **Barcha so'z emas, yarmi.** Birinchi qoida "har bir so'z tegishi shart" edi:
"chek rus tilida chiqmayapti" hech nima topmaydi, chunki "chiqmayapti" —
maqoladagi "chiqyapti" ning inkori va o'zbek tili o'sha "ma" ni **so'z o'rtasiga**
qo'yadi. Hech qanday prefiks qoidasi ularni bog'lay olmaydi. Bitta so'z esa juda
kam: "chek" yolg'iz har bir chek maqolasini til haqidagisidan tepaga chiqaradi.

⚠️ **Apostrof — o'zbekcha harf.** So'z chegarasi qoidasi unda bo'linsa,
"o'zgartirish" ikki so'zga aylanadi va "o'zgar" unga mos kelmay qoladi.

⚠️ **Operatorga o'tish har doim ko'rinadi**, javob topilgan-topilmaganidan
qat'i nazar: maqolani o'qib ham hal qila olmagan ega yordamdan chiqish yo'lini
qidirmasligi kerak.


**AI yordamchining javobi (`control/handlers/supportai.go`)**

⚠️ **Model faqat unga berilgan maqolalardan javob beradi, va ular yetmasa
«bilmayman» deydi.** Bu brifingdagi bilan bir xil chok va bir xil sabab:
mahsulot xatti-harakati haqida o'ylab topilgan javob umuman javob yo'qligidan
yomon. Bitta noto'g'ri ko'rsatmaga amal qilgan restoran — fiskal chekni qayta
chop etgan, inventarizatsiyani tozalagan, printer kod sahifasini almashtirgan —
biz keltirgan haqiqiy muammoga ega bo'ladi, va shundan keyin bu oyna aytgan
hech narsaga ishonilmaydi.

⚠️ **Mashina javobi odamni jarayondan chiqarmaydi.** Thread `waiting` holatida
qoladi va operator navbatida turaveradi, faqat "yordamchi allaqachon javob
bergan" belgisi bilan. Aks holda — yopilsa yoki navbatdan chiqsa — noto'g'ri
javob hech kim qaramaydigan javobga aylanadi, ya'ni aynan ushlanishi kerak
bo'lgan holat.

⚠️ **Operator suhbatga kirgandan keyin yordamchi hech nima yozmaydi.** Odamning
javobi ostida paydo bo'lgan mashina jumlasi operator o'zini inkor qilayotgandek
o'qiladi, va ega ikkalasining qaysi biriga amal qilishni bilmaydi.

⚠️ **Maqolalar paneldan keladi, Go'da takrorlanmaydi.** Baza panel bundle'ida
(shuning uchun internetsiz ishlaydi va aynan shu build'ni tavsiflaydi), ya'ni
serverdagi ikkinchi nusxa ajrab ketadigan ikki matn bo'lardi. So'rovni
tahrirlagan odam modelga o'z matnini berib, uni **o'z chatida** qaytarib olishi
mumkin — ya'ni o'ziga o'zi bir narsa aytadi. Zarar radiusi bitta ekran, va
aynan shuning uchun bu yerda ruxsat etiladi, `plan: "enterprise"` esa yo'q.

⚠️ **Yordamchi javobi xabar yuborilgandan **keyin** so'raladi**, oldin emas: ega
send bosdi va uning qatori darhol chiqishi kerak; rad etishi mumkin bo'lgan
model uchun bir necha soniya kutish — buzuq chat. Javob socket orqali keladi,
xuddi operatorniki kabi — bitta kanal, ekranda bitta xatti-harakat.

⚠️ **Nosozliklar jim.** Operator baribir keladi, va "yordamchining kvotasi
tugadi" — bu bizning muammoimiz, mijozning muammosi o'rtasida yozilgan.

**⚠️ Kampaniya matni yozilmasligining sababi ekranga yetib bormasdi (tuzatildi
2026-08-30)**

Ega "menga uchta matn yoz" tugmasini bosadi va shunchaki "yozib bo'lmadi"
degan yozuvni oladi. Ikkita alohida nuqta bir xil natijaga olib kelardi:

1. **Ulanmagan install.** `callControlPath` da bo'sh `CONTROL_URL` tekshiruvi
   yo'q edi, ya'ni so'rov bo'sh manzilga qurilar va Go
   `Post "/internal/campaign-text": unsupported protocol scheme ""` deb javob
   berardi — panel buni oddiy "xatolik" qilib ko'rsatardi. Uchta chaqiruvchida
   o'z tekshiruvi bor edi, to'rtinchisida yo'q. Endi tekshiruv **hamma
   chaqiruvchi o'tadigan bitta joyda** va `ErrNotLinked` tipli xato qaytaradi;
   kampaniya ekrani buni xato emas, "bu serverda yordamchi yo'q" deb ko'rsatadi.
2. **Kalitsiz platforma.** Konsol allaqachon `off: true` qaytarardi (brifing
   bilan bir xil shakl), lekin tenant handleri uchta bayroqni uzatib, aynan
   shuni **tashlab yuborardi**. Sabab butun yo'l bo'ylab mavjud edi va ekrandan
   bir qadam narida yo'qolardi.

⚠️ Panel endi **serverning o'z jumlasini** ko'rsatadi. "Yozib bo'lmadi" egaga
hech nima aytmaydi, sabab esa odatda u hal qila oladigan narsa: kalit
qo'yilmagan, kunlik limit tugagan, segmentda odam yo'q.

### Taklif matni (`/console/outreach`)
Restoranga yoziladigan xabar. Konsolda tanlanadi: **kim** (Telegram orqali
yetkazadi / yangi ochilyapti / allaqachon tizimi bor), **til** (uz / ru),
**qaysi xabar** (birinchi / eslatma). Bitta tugma bilan nusxa olinadi.

- ⚠️ **Matnlar qo'lda yozilgan, AI generatsiya qilmaydi** — garchi buning
  quvuri allaqachon bor bo'lsa ham (`campaigntext.go`). Uch sabab, birinchisi
  hal qiluvchi: model shu bozorda **ko'rishi bilan tanib olinadigan** ohangda
  yozadi. «Здравствуйте! Меня зовут… Я хотел бы предложить…» ikkinchi qatorgacha
  spam deb o'qiladi, va hal qilinayotgan muammoning o'zi — bu xabarlarning
  e'tiborsiz qolishi. Qolgan ikkitasi: har generatsiya pul sarflaydi (uni
  tejashga urinyapmiz) va bugun to'g'ri topilgan ibora ertaga boshqasiga
  aylanadi.
- ⚠️ **Eng muhim qator ataylab bo'sh qoldirilgan.** `{note}` — yozayotgan odam
  aynan shu joyda nima ko'rganini yozadi («Instagramda menyungizni ko'rdim»).
  Shablonni javob keladigan xabardan ajratadigan narsa shu bitta jumla, va uni
  hech qanday generator bera olmaydi: bu faqat yozayotgan odam biladigan fakt.
  Bo'sh qolsa **butun qator tushiriladi** — o'rtada qolgan bo'sh xatboshi
  «generatsiya qilingan» degan eng ochiq belgi.
- **Segmentlar uchta, chunki ular haqida uch xil narsa rost**: Telegram orqali
  yetkazadigan joyda kundalik, tasvirlab bo'ladigan og'riq bor va olib
  tashlanadigan narsa yo'q; yangi ochilayotgan joy hammasini bir vaqtda sotib
  olyapti va «bizda bor» deya olmaydi; tizimi bor joy esa — eng qiyini.
- ⚠️ **«Almashtiring demayapman» olib tashlandi**, va u aynan birinchi qatorda
  turardi. Bu — o'quvchi hali aytmagan e'tirozga berilgan javob, va uni
  birinchi bo'lib ko'tarish o'sha e'tirozni **o'zing ekasan**. Undan ham
  yomoni: butun xabar o'zidan kattaroq narsaning yonida yashashga ruxsat
  so'rovga aylanadi — kechirim so'rab boshlagan tomondan esa hech kim sotib
  olmaydi.
  Rost gap torroq va foydaliroq: iiko ishlatadigan restoran onlayn buyurtmani
  **allaqachon** Delever yoki Zoomda orqali, yoki 15–20% oladigan agregator
  orqali oladi. Solishtiriladigan narsa shu — va uning oylik narxi, mijoz
  bazasining egasi va kimningdir nomidagi domeni bor. Shuning uchun bu
  xabarlar **o'shani, raqamlarda** solishtiradi, kassani esa atigi bir marta
  tilga oladi: buyurtma qayerga tushishi haqidagi fakt sifatida, hech qachon
  tinchlantirish sifatida emas.
- ⚠️ **Qo'lda yozilganining o'zi yetmadi — birinchi to'plam baribir «AI
  yozgan» bo'lib o'qilardi.** Sabab lug'atda emas, **shaklda** edi: har
  xabarda salom, og'riqni aytadigan xatboshi, yechimni aytadigan xatboshi,
  narx qatori, muloyim yakuniy savol. Beshta blok, bir xil tartib, bir xil
  uzunlik — har safar. Shuning uchun yangi to'plam ataylab **notekis**:
  uzunligi ikki qatordan sakkiztagacha; ba'zi xabar umuman taklif qilmaydi,
  bitta savol berib to'xtaydi (bitta so'z bilan javob beriladigan savolga
  qaror talab qiladigan taklifdan ko'ra ko'proq javob beriladi); ba'zisi
  narxni umuman aytmaydi (hech kim qiziqish bildirmasidan turib pul haqida
  gapirish — so'ralmagan savolga javob); yakunlari har xil va hammasi ham
  savol emas; sifatlar o'rniga raqamlar.
- ⚠️ **Sakkizta variant, uchta emas.** Uchta ekranda xilma-xil ko'rinishga
  yetadi va **bir kechaga yetmaydi**: yigirmata joyga yozayotgan odam ularni
  birinchi soatda tugatadi, undan keyingi har bir xabar — takror. Aynan shu
  hisobni cheklashga olib keladi va aynan shuni bir-biriga aytib qo'ygan ikki
  qo'shni restoran sezadi.
- ⚠️ **Bo'sh nom — muloyim so'z emas, ko'rinadigan bo'shliq.** Ilgari u
  «restoraningiz» / «вашего заведения» bilan to'ldirilardi: matn **tugallangan
  ko'rinadi** va yarim xabarda grammatik xato bo'ladi — ruschada «По {name}»,
  «для {name}» va «{name} открывается» uch xil kelishikni talab qiladi, bitta
  so'z esa bir vaqtda uchtasida bo'la olmaydi. «По вашего заведения» — xabarni
  ikkinchi qatorgacha o'chiriladigan qiladigan jumla, va u **yuboriladi**,
  chunki tugallangandek ko'rinadi. Endi `[restoran nomi]` va **nusxa olish
  tugmasi o'chiq** turadi: nom aslida hech qachon noma'lum emas — yozayotgan
  odam unga qarab turibdi.
- ⚠️ **Qo'shimchalar nomga apostrof bilan qo'shiladi** (`{name}'ning`,
  `{current}'ga`): «iiko ga tushadi» deb hech kim yozmaydi. Bo'shliq
  o'rinbosari esa apostrofni **yutadi** — `[restoran nomi]da`, chunki u
  atoqli ot emas.
- ⚠️ **Narxlar landingdan olinadi** (kassa 450 000/oy, buyurtma 800 so'm —
  birinchi pog'ona) va u bilan birga o'zgarishi shart. Sayt rad etadigan narxni
  keltirgan xabar — kechirim so'rashdan boshlanadigan suhbat.

### Qidiruv tizimlari (`/console/seo`)
- ⚠️ **Googlega sahifani itarib bo'lmaydi va ekran buni yashirmaydi.** Talab
  «hamma sahifani Google Search Consolega avtomatik yuboradigan tugma» edi.
  Bunday ommaviy API yo'q: Indexing API hujjatda **faqat** `JobPosting` va
  `BroadcastEvent` uchun, sitemap «ping» manzili esa **2023 yilda yopilgan**.
  Jimgina hech nima qilmaydigan tugma tugmasiz holatdan **yomonroq**: kimdir
  uni bosadi va nega indekslanmayotganini so'rashni to'xtatadi. Shuning uchun
  ekran Google uchun qo'lda bajariladigan bitta qadamni yozadi (sitemapni
  Search Consolega bir marta qo'shish) va uni bajarish osonlashtiriladi.
- **IndexNow esa haqiqiy** va aynan bu yerda muhim: Bing, Yandex, Seznam, Naver
  qabul qiladi. O'zbekistonda Yandexning ulushi tugmani o'zi oqlaydi.
- ⚠️ **Manzillar ro'yxati sitemapdan o'qiladi, fayldagi ro'yxatdan emas.**
  Sitemap allaqachon «qanday sahifalar bor» degan savolning kanonik javobi va u
  maqolalar ma'lumotidan uch tilda generatsiya qilinadi. Ikkinchi ro'yxat
  yozilgan kuni to'g'ri va keyingi maqoladan keyin xato bo'lardi.
- ⚠️ **Alternativalar ham yig'iladi**: sitemap yozuvi bitta kanonik manzil va
  `hreflang` qardoshlarini `<xhtml:link>` da beradi. Faqat `<loc>` ni olish
  o'zbekcha sahifani yuborib, rus va inglizchasini skanerga qoldirardi —
  ya'ni sahifalarning uchdan ikkisi hech qachon yuborilmasdi.
- ⚠️ **Begona hostdagi manzil tashlab yuboriladi**: IndexNow bitta yot manzil
  uchun **butun jo'natmani** rad etadi, va nosozlik bitta 422 bo'lib chiqadi.
- ⚠️ **Dvigatelning javob kodi o'zgartirilmasdan uzatiladi**: 403 — kalit fayli
  o'qilmadi, 422 — manzil boshqa hostda, 200/202 — qabul qilindi. Uchalasining
  yechimi boshqa, «yuborilmadi» esa hech birini aytmaydi.
- **Kalit bo'sh bo'lsa yuborish rad etiladi**, jimgina o'tkazib yuborilmaydi.
  Kalit `INDEXNOW_KEY` da va **ikki joyda bir xil** bo'lishi shart: control uni
  yuboradi, keel-site `/indexnow.txt` da ko'rsatadi — protokolda egalikning
  yagona isboti shu.
- ⚠️ **Sitemapda har til o'z yozuvi bilan turadi — 279 ta `<url>`, 93 emas.**
  Birinchi versiya har sahifani **bir marta** (o'zbekcha manzil bilan) yozib,
  `ru` va `en` ni `<xhtml:link>` sifatida osib qo'yardi: "bitta sahifa, uchta
  manzil" degan fikr mantiqiy ko'rinadi va **Google hujjatlaganidan boshqa**.
  Google sitemapda **har til versiyasi o'z `<url>` elementiga** ega bo'lishini
  talab qiladi, va ularning har biri to'liq alternativalar to'plamini (o'zini
  ham) ko'rsatishi kerak — bir tomonlama e'lon e'tiborga olinmaydi. Bu qoidaga
  sahifalarning `<head>` i allaqachon amal qilardi (`alternatesFor`), sitemap
  esa yo'q.
  ⚠️ **Nosozlik muvaffaqiyatga o'xshaydi**: Search Console faylni qabul qiladi
  va "93 sahifa" deydi — 279 manzilli sayt uchun bu butunlay ishonarli raqam.
  Saytning uchdan ikki qismi boshqa uchdan birning **atributi** sifatida
  yuborilib, skanerga qoldirilardi — ya'ni sitemap bartaraf qilishi kerak
  bo'lgan aynan o'sha kutish, va kutmasligi eng zarur bo'lgan sahifalar rus
  tilidagilar edi.
  `x-default` ham qo'shildi (o'zbekchaga ko'rsatadi): uch tilning hech biri
  o'qiy olmaydigan mehmonga qaysi biri berilishini aytadi.
- **Strukturali ma'lumot chuqurlashtirildi**: har maqolada `TechArticle` va
  `BreadcrumbList` (natijada «Keel › Qo'llanma › Ombor» ko'rinadi), qo'llanma
  indeksida `CollectionPage`, bosh sahifada `FAQPage`. ⚠️ `datePublished`
  **qo'yilmadi**: maqolalarda halol sana yo'q, build vaqti esa Googlega 264
  sahifa bugun ertalab yozilgan va har deployda qayta yozilgan deb aytardi.
  Yo'q maydon hech nima turmaydi, yolg'oni esa da'vo.

### Hamkorlar: tashqi tavsiya va komissiya (`/console/referrers`)
Distributsiya mahsulotdan qiyinroq bo'lib chiqdi. Sovuq DM ishlamaydi, iiko
o'rnatgan restoranda esa allaqachon Delever turibdi. Qoladigan yo'l —
**haftada yigirmata oshxonaga kiradigan odamlar**: fiskal kassa sotuvchisi,
qadoq yetkazib beruvchi, o'n beshta joyni yuritadigan buxgalter. Ular bizga
raqobatchi emas, lekin **oldingi mijoz uchun to'lanmagan bo'lsa** keyingisini
yubormaydi.

- ⚠️ **Bu agent emas va ikkisini birlashtirish faktni yo'qotadi.** Agent bu
  yerda ishlaydi: konsol logini bor, tashrif yozadi, `tenant.createdById` uni
  ko'rsatadi. Hamkor — tashqaridagi biznes, logini kerak emas. **Ikkalasi bir
  mijozda rost bo'lishi mumkin**: kassa firmasi yuborgan leadni agent yopgan,
  va bitta maydon buni ayta olmaydi.
- ⚠️ **Nomi `Referrer`, `Partner` emas**: `Partners` allaqachon band — landing
  sahifadagi to'lov va POS tizimlarining logotiplari. Bitta kodda ikki xil
  narsani bir so'z bilan atash — ikkinchi o'qigan odam uchun tuzoq.
- **Komissiya — kelgan puldan, hisob-fakturadan emas** (`Invoice.Paid`, ya'ni
  haqiqatan qo'ldan qo'lga o'tgan summa). Hisob-fakturaga qarab to'lash —
  hech qachon to'lanmasligi mumkin bo'lgan hisob uchun to'lash, va bu aynan
  to'laydigan pul yo'q oyda sodir bo'ladi.
- ⚠️ **Har to'lov o'z sanasi bilan tekshiriladi.** Martni iyunda to'lagan mijoz
  martning puli uchun oyna ichida, iyunning puli uchun tashqarida. Davr bo'yicha
  yig'ish yig'ilmagan pulga komissiya berardi; hisob-faktura sanasi bo'yicha
  yig'ish esa allaqachon tugagan oyna uchun.
- **Oyna `subscribedAt` dan sanaladi**, tenant yaratilgan kundan emas: sinov
  hech kimga hech nima to'lamaydi, ya'ni ro'yxatdan boshlangan oyna birinchi
  hisob-faktura paydo bo'lgunicha yarim sarflangan bo'lardi.
- ⚠️ **Komissiya saqlanmaydi, hisoblanadi.** Saqlangan raqam yozilgan kuni
  hisob-fakturalar bilan rozi bo'ladi va keyin ulardan ajrab ketadi (to'lov
  bekor qilindi, hisob tuzatildi) — ikkalasi ham bir xil rasmiy ko'rinadi.
- ⚠️ **Foiz 50 bilan, muddat 60 oy bilan chegaralangan.** 500 deb yozilgan
  foiz tushumdan katta komissiya beradi va u **to'lab yuborilganda** bilinadi.
- ⚠️ **Havola — yo'l, so'rov qatori emas** (`keel.uz/h/fiskal`). `?ref=` odatiy
  yechim va bu havolalar bosib o'tadigan yo'lni bosib o'ta olmaydi: varaqadan
  o'qiladi, telefonga teriladi, Telegramga tashlanadi va qaytib nusxalanadi —
  so'rov qatori aynan uzun ko'ringan havoladan qirqib tashlanadigan qism.
- ⚠️ **`/h/<kod>` — sahifa emas, route handler**: Next sahifada cookie
  qo'yishga ruxsat bermaydi. Birinchi versiya sahifa edi, kompilyatsiya bo'ldi
  va **bosma havola bilan kelgan har bir tashrifchiga 500** qaytardi — nima
  ko'rishi kerakligini bilmagani uchun xabar bera olmaydigan yagona auditoriya.
- **Kod cookie'da (biz uchun) va ekranda (ular uchun)**: ro'yxatdan o'tish
  Telegramdagi suhbat, forma emas, ya'ni kodni tenant yozuviga olib boradigan
  avtomatik yo'l yo'q. Halol qo'lda qadam buzuq avtomatikadan yaxshiroq.
  Avtomatik biriktirish uchun o'sha havolaning narigi uchida **bot** kerak.
- ⚠️ **`Location` — nisbiy yo'l, `req.url` dan qurilgan to'liq manzil emas.**
  Konteyner ichida `req.url` ning hosti — **konteynerning o'z nomi**
  (`76c98175c198:3100`), ya'ni bosma varaqadan kelgan har bir tashrifchi
  `https://76c98175c198:3100/?h=otto` ga yuborilardi — brauzerda ochilmaydigan
  manzil. Nisbiy `Location` ni brauzer o'zi turgan domenga nisbatan yechadi,
  demak proxy orqasida ham, `localhost` da ham to'g'ri. **Alomati domenga
  o'xshamaydi**: QR ishlamayapti deb o'ylanadi, aybi esa serverda.

- ⚠️ **Noma'lum kod biriktirishni tozalaydi** (tahrirda), eskisini qoldirmaydi:
  maydon faqat uni tahrirlayotgan odam tomonidan yuboriladi, va jimgina eski
  kanalni saqlash — qo'llanilgandek ko'rinadigan, lekin qo'llanmagan tuzatish,
  ustiga pul bog'langan holda.

### Varaqa: peshtaxtaga qoldiriladigan A5 (`/console/leaflet`)
- ⚠️ **Ega odatda joyda bo'lmaydi.** Soat ikkida restoranga kirgan odam
  kassirni topadi. Qoldiradigan narsasiz bu tashrif hech nima bermaydi;
  varaqa bilan esa daftar yetarlicha bezdirgunicha kassa yonida turadigan QR
  qoladi.
- ⚠️ **Taklif — kassa, sayt emas.** Tizimi yo'q joy sayt orzu qilib
  uxlamaydi; u bugun qancha tushganini bilishni xohlaydi. Saytni birinchi
  qo'yish bizni bozorning **band yarmiga** — sotuv jamoasi bor odamlar
  qarshisiga — olib chiqadi; kassani qo'yish esa alternativasi qog'oz daftar
  bo'lgan odam oldiga.
- **QR — inline SVG**: canvas ekran zichligida chiziladi va printerdan kamera
  zo'rg'a o'qiydigan kulrang kvadrat bo'lib chiqadi — vazifasi skanerlanish
  bo'lgan varaqada.
- ⚠️ **QR atrofida oq chegara** (quiet zone) shart: rangli maydonga taqab
  bosilgan kod umuman o'qilmaydi, va nosozlik **buzuq telefon** bo'lib
  ko'rinadi, yomon maket bo'lib emas.
- ⚠️ **Ranglar tema tokenlaridan olinmaydi**, qo'lda yozilgan: token o'quvchining
  qorong'i rejimi bilan o'zgaradi, qog'oz esa yo'q — qorong'i temada bosilgan
  varaqa qora to'rtburchak va bo'shagan kartrij demakdir.
- Cookie bildirishnomasi ham `@media print` da yashiriladi: u `fixed`, ya'ni
  sahifadan aylanib ketmaydi — narxning ustiga bosiladi.
- ⚠️ **Chop etishda varaqani buzadigan narsa varaqada emas, atrofida edi.**
  Sheet aynan bitta A5 balandligida, lekin konsol layoutining `main` paddingi
  va sahifaning `space-y` oralig'i uni pastga surib, **ikkinchi qog'ozga**
  chiqarardi — ikkinchisi deyarli bo'sh, ya'ni har varaqa ikki qog'oz.
  Shuning uchun `@media print` faqat `.no-print` ni yashirmaydi: `@page` bilan
  o'lcham (`148mm 210mm`) va `margin: 0` beriladi, `html`/`body`/`main` ning
  chekkalari nolga tushiriladi. **Tekshiruv — chop etilgan PDF sahifalari
  soni**, ekrandagi ko'rinish emas: ekranda ikkalasi bir xil.

### O'sish ekranlari uch tilda (`lib/i18n/growth.ts`)
Hamkorlar, taklif matni, qidiruv tizimlari va varaqaning boshqaruvlari
o'zbekcha qattiq yozilgan holda tug'ilgan edi — konsolning qolgani uch tilda
bo'lgani holda.

- **Alohida lug'at, umumiy `dict.ts` ga emas** — konstruktorning
  `editor.ts` si bilan bir qaror: bu yuz qator bitta ishni qiladigan bitta
  odam uchun, va ularni umumiy lug'atga qo'shish qolgan har bir ekranning
  tarjimasini o'qishni qiyinlashtiradi.
- **`GrowthDict` o'zbekchadan olinadi** (`typeof uz`), ya'ni `ru` yoki `en`
  da tushib qolgan kalit — **kompilyatsiya xatosi**, jimgina o'zbekchaga
  qaytadigan so'z emas. Yarim tarjima qilingan ekran aynan shunday paydo
  bo'ladi va uni rus tilidagi o'quvchi aytmaguncha hech kim ko'rmaydi.
- ⚠️ **Ikki ekranda ikki xil til bor va ular bog'lanmaydi.** Taklif matnida
  `lang` — konsolning tili (quti atrofidagi so'zlar), `msgLang` — xabarniki
  (restoranga boradigan so'zlar); varaqada `lang` va `sheetLang` ham shunday.
  Rus tilida ishlaydigan odam o'zbek ko'chasiga o'zbekcha varaqa bosadi, ya'ni
  birini ikkinchisiga bog'lash bu ekranlar mavjud bo'lish sababi bo'lgan
  tanlovni olib tashlaydi.
- ⚠️ **Jumla bo'lakka bo'linmaydi.** Komissiya qoidasi ilgari oltita
  bo'lakdan (`ruleA` + qalin `rulePaid` + `ruleB` + foiz + …) yig'ilardi.
  O'zbekchada foiz jumlaning oxiriga yaqin, ruschada boshiga yaqin keladi —
  bunday yig'ilgan jumla **faqat bitta tilda** to'g'ri bo'lishi mumkin. Endi
  har tilda bitta jumla va ichida `{percent}` / `{months}` o'rinbosarlari;
  qalin ajratish shu narxda tashlab yuborildi.
- Varaqaning **bosiladigan matni** bu yerda emas (`COPY`, sahifaning o'zida):
  u konsol tilidan mustaqil tanlanadi va uch emas, ikki tilda.

### Konsol xodimlari: rollar, agentlar va tashriflar
- Ilgari konsolda **bitta hisob** bor edi — platforma egasining o'zi. Sotuv bir
  odamning ishi bo'lganda ishlaydi va odam yollangan kuni to'xtaydi: eganing
  logini bilan agent to'lovchi mijozni to'xtatishi, har restoranning raqamlarini
  o'qishi va hisob-kitobni o'zgartirishi mumkin.
- To'rt rol, chegara **zarar bo'ladigan joyda** chizilgan (`models/staff.go`):
  `owner` (hammasi; **faqat u** hisob yaratadi va jurnalni o'qiydi), `admin`
  (platformani yuritadi: provisioning, domen, hisob-fakturalar), `manager`
  (sotuv nazorati: hamma mijoz va kim jalb qilgani; server va billingga
  tegmaydi), `agent` (**faqat o'zi jalb qilgan** mijozlar).
- ⚠️ **Agent restoran statistikasini umuman ko'rmaydi.** Unga kimni yozganini va
  nima va'da qilganini bilish kerak; restoranning tushumi — **o'sha
  restoranning ishi**, va uni sotuvchiga berish — chiroyli nomli sizib chiqish.
- ⚠️ **Chegara bitta joyda**: `tenantScope(user)` rolni Mongo **filtriga**
  aylantiradi, va har bir mijoz so'rovi shundan boshlanadi. Filtr, tekshiruv
  emas: o'qib bo'lib qirqiladigan ro'yxat yoniga kimdir `count`, agregat yoki
  eksport qo'shgan kuni sizib chiqadi. Testda muhrlangan.
- ⚠️ **Bo'sh rol — `owner`**, chunki birinchi bootda seed qilingan yagona hisob
  aynan platforma egasining hisobi: uni agent deb o'qish rollar joriy qilingan
  deployда eganing o'zini konsoldan qulflab qo'yardi. Notanish rol esa —
  **agent** (eng kam ruxsat): rol nomidagi xato hech kimga konsolni bermasligi
  kerak.
- **Kim jalb qilgani `createdById` bilan** yoziladi, faqat nom bilan emas: nom
  tahrirlanadi, id esa yo'q — nom bo'yicha filtr o'zini qayta nomlab chiqib
  ketish mumkin bo'lgan filtr.
- **Begonaning mijozi 404**, 403 emas: id'larni taxmin qilayotgan agent mijoz
  **borligini** ham bilmasligi kerak.
- **Oxirgi `owner` ni pasaytirib ham, o'chirib ham bo'lmaydi**: hisob yaratish
  huquqi qolmagan konsolni ichidan tuzatib bo'lmaydi.
- **Tashriflar** (`visit`): agent qayerga borishini **oldindan** yozadi, borgach
  natijani (ijobiy / salbiy / keyin borish) va izohni yozadi. ⚠️ **Salbiy natija
  izohsiz qabul qilinmaydi**: sababsiz "yo'q" bir oydan keyin "bormadim" dan
  farq qilmaydi, va aynan sabab keyingi tashrifni rejalashtirishga arziydigan
  qiladi. "Keyin borish" sanasiz qabul qilinmaydi — muddatsiz va'da va'da emas
  (call-markazdagi bilan bir qoida).
- Tashrif **sessiyadan** imzolanadi (`agentId`), so'rovdan emas: boshqa odam
  nomiga yozib qo'yish mumkin bo'lgan jurnal jurnal emas.
- `GET /me` **hal qilingan ruxsatlarni** qaytaradi (`can.*`), rol nomini emas:
  qoida serverda bitta joyda yashaydi va ikki tomon bir-biridan ajrab ketmaydi.
  Konsol rol ishlatolmaydigan bo'limni yashiradi — server baribir rad etadi,
  lekin har tugmasi "ruxsat yo'q" deydigan sahifa odamga asbobi buzuq ekanini
  o'rgatadi.

**2026-09-14: beshinchi rol, bir nechta rol, bo'limlar va to'liq o'chirish**
- **`support`** (tex-support) — faqat Yordam va Xatoliklar. Mijozlar,
  tashriflar, pul yo'q: «panel bo'sh» degan savolga javob beradigan odamga
  thread va stek kerak, narxlar ro'yxati emas.
- ⚠️ **Bir hisobda bir nechta rol, ruxsatlar — birlashma** (`user.roles`,
  `User.Can(check)`). Hech qachon `CanX(u.RoleOf())` emas: `RoleOf()` — eng keng
  rol, birlashma emas, va agent+support hisobini faqat agent deb so'rash uni
  berilgan navbatdan qulflaydi. `role` ham yoziladi (eng keng rol) — eski
  hisoblar `roles` siz o'qiladi va `RolesOf` shunga qaytadi (bo'sh `role` hali
  ham owner).
- ⚠️ **So'rovda rol yo'qligi owner emas** (`requestRoles`): saqlangan bo'sh rol
  faqat seed qilingan hisob tufayli owner. Tahrirda **bo'sh ro'yxat rad
  etiladi** — aks holda hisob `role` ga qaytib, seed hisobni *kengaytirardi*.
- **Bo'limlar** (`models.Permissions` — router gate'lari va `/me` bitta
  xaritadan, testi router.go'ni o'qib har gate nomi mavjudligini tekshiradi):
  Umumiy ko'rinish — owner; Mijozlar/Tashriflar/Taklif — owner, admin,
  manager, agent; Yordam/Xatoliklar — owner, support; Hamkorlar, Qidiruv,
  Xodimlar — faqat owner; Blog — owner va admin.
- ⚠️ **Faqat owner umumiy ko'rinishga tushadi**; qolganlar logindan keyin (va
  `/console` ochilganda) **render qilinmasdan oldin** o'z ekraniga
  yo'naltiriladi (`consoleHome`): agent/menejer/admin — mijozlarga, support —
  navbatga. Avval render qilish sahifa so'rovlarini yuborib, aynan shu
  odamlarga «ma'lumot yo'q» ni ko'rsatardi.
- ⚠️ **Tab yashirish sahifani himoya qilish emas** (2026-09-14, agent hisobi
  bilan sinovda topildi): agent `/console/seo` ni manzilga yozsa **sahifaning
  o'zi** ochilardi — sarlavhalar, tugmalar, tushuntirish — ostidagi har so'rovni
  server 403 bilan rad etgan holda. Bazadan hech nima chiqmadi, lekin sahifa
  ham ma'lumot, va bu tashqaridan aynan teshik bo'lib ko'rinadi. Endi layout
  **render qilishdan oldin** `canOpen(path, can)` ni so'raydi
  (`lib/consoleAccess.ts` — yo'l → ruxsat xaritasi bitta joyda); ruxsatsiz
  sahifa o'z ekraniga yo'naltiradi, hech bir bo'limi yo'q hisob "ruxsat yo'q"
  ko'radi (cheksiz yo'naltirish emas). ⚠️ Sahifa almashganda ham har renderda
  tekshiriladi — effekt yangi sahifa bir marta chizilgandan keyin ishlaydi.
  ⚠️ **Noma'lum sahifa — yopiq**, va `consoleAccess.test.ts` `app/console` dagi
  **har papka** xaritada borligini tekshiradi: yangi sahifa ochiq eshik emas,
  yiqilgan test bo'lib chiqadi. Server tomoni o'zgarmagan va lokal sinovda
  agent `/seo`, `/blog`, `/referrers`, `/support`, `/reports`, `/staff` dan 403
  oldi.
- **O'chirish ikki xil**: «Vaqtincha o'chirish» (`isActive: false`, login
  saqlanadi) va «Butunlay o'chirish» (`DELETE /staff/{id}`, tasdiq bilan).
  ⚠️ O'zingizni va oxirgi **faol** owner'ni o'chirib bo'lmaydi. Yozuvlar
  qoladi: mijoz, tashrif va jurnal ismni yozilgan paytda nusxalagan.
- **Xodim sahifasi** (`/console/staff/[id]`, `GET /staff/{id}`): hamma raqam
  ish o'zi yozadigan hujjatlardan sanaladi — mijozlar `createdById`, tashriflar
  `agentId`, yordam `operatorId`, hisob-fakturalar `issuedBy`, jurnal
  `actorId`. ⚠️ Tuzatilgan xatoliklar **ism bo'yicha** (`resolvedBy` ismni
  saqlaydi) — ism o'zgarsa eski sanoq tushib qoladi.

### Mini app'da til: tanlanadi, taxmin qilinmaydi
- Saytda tilni **URL + cookie** tashiydi. Mini app'da ikkalasi ham yo'q (bot
  birinchi ochilganda cookie yo'q, manzil satri ham yo'q), shuning uchun
  **birinchi ekran — til tanlash** (`components/site/TelegramLangGate.tsx`).
  Ekran bo'lmasa ilova hamma uchun o'zbekcha ochilardi va almashtirgich
  mijoz **o'qiy olmaydigan menyuni o'qib turib** izlaydigan narsa bo'lardi.
- ⚠️ **Ekranda tarjima qilingan matn yo'q, va bu ataylab**: bitta tilda
  "Tilni tanlang" sarlavhasi — ilovaning bitta tilda ochilishi bilan bir xil
  xato. Variantlar o'z tilida (`LANG_LABEL`), sarlavha o'rnida globus.
- **Tanlov hisobda** (`user.lang`, `PUT /users/me/lang`), cookie'da emas: eng
  kerak bo'ladigan joy — soatlar keyin fon goroutine'idan yuborilgan **bot
  xabari**, va uning o'qiydigan brauzeri yo'q. Uchta maydon uchta boshqa
  savolga javob beradi: cookie — "bu qurilma hozir nima ko'rsatyapti",
  `telegramLang` — Telegramning taxmini, `user.lang` — mijoz **ataylab**
  bergan javob. Shuning uchun ustunlik `notifyLang` da: `lang` → `telegramLang`
  → `uz`, va u testda muhrlangan (ikkalasi odatda bir xil bo'ladi, ya'ni
  noto'g'ri tartib oylar davomida ishlab ko'rinardi).
- ⚠️ **Qiymat allowlist'dan o'tadi** (`langAllowed`): u **xabar shablonini**
  tanlaydi. Notanish qiymat yiqilmaydi — jimgina o'zbekchaga tushadi va
  mijozning tanlovi e'tiborga olinmagandek ko'rinadi.
- **Sayt sarlavhasidagi almashtirgich ham hisobga yozadi** (kirgan mijoz
  uchun). Aks holda ruschaga o'tgan mijoz botdan o'zbekcha xabar olishda davom
  etardi — o'sha nomuvofiqlik boshqa eshikdan qaytib kelardi.
- **Alohida endpoint**, `PUT /users/me` emas: u profil formasi va har chaqiruvda
  ism/manzillarni yozadi, ya'ni birinchi ekrandan yuborilgan til ismi bor
  mijozning ismini o'chirib yuborardi.
- Panel ekrani `--tg-viewport` bilan (100vh emas): 100vh'da eng pastdagi tugma
  Telegramning o'z paneli ostida qolardi, va eng pastdagi tugma — "English".

### POS integratsiyasi: iiko / Syrve / Poster / Clopos / r_keeper
- **Menyu bizniki, kassaga buyurtma ketadi**: bizda rasm, tarjima, combo va
  sayt matnlari bor, ikki joyda menyu yuritish chalkashlik. POS'dan hech nima
  tortilmaydi; har taom kassadagi id'siga **bog'lanadi** (`/admin/pos`).
- **Sozlama filialga tegishli**: zanjirda har oshxonaning o'z terminal guruhi
  bor, va buyurtmaning noto'g'ri kassada chop etilishi umuman chop
  etilmaganidan yomonroq. Bog'lashni boshqa filialdan ko'chirish mumkin
  (`POST /admin/pos/mapping/copy`) — bir xil brend **va** bir xil provayder
  shart (id boshqa kassaning nomlar makonida), standart holatda faqat bo'sh
  taomlar to'ldiriladi.
- **Buyurtma "Tasdiqlash"da ketadi**, yaratilishida emas: kassa — birovning
  buxgalteriyasi, u yerdagi xatoni qo'lda bekor qilish kerak bo'ladi.
- **Ikki marta ketmaydi** (`pos.status` bilan qo'riqlangan): har ortiqcha chek —
  oshxona haqiqatan pishiradigan taom.
- **Bog'lanmagan taom butun buyurtmani to'xtatadi** (`pos.CheckMapped`) —
  chala chek bergan oshxona aynan ko'rganini pishiradi. Xatoda taom nomi bo'ladi.
- **Kassa javob bermasa buyurtma buzilmaydi**: sabab chekda, "Qayta yuborish"
  tugmasi turadi. Bizda bor, kassada yo'q buyurtmani tuzatish mumkin; jimgina
  yo'qolganini — yo'q.
- **`Ping` nima bilan ulanganini aytadi** ("Maracanda · terminal ishlayapti"),
  shunchaki "ulandi" emas; iiko'da terminal guruhining tirikligi ham
  tekshiriladi (o'lik terminalga buyurtma jimgina yutiladi).
- **Syrve — iiko'ning o'zi** (`api-eu.syrve.live`). Adapter bitta: `iiko.go`
  provider nomini tashib yuradi (`c.name`), host shundan tanlanadi. Ro'yxatda
  alohida va kalitlari alohida: egasi tizimini "Syrve" deb biladi.
- **iiko**: `deliveries/create` **asinxron** — 200 hali qabul qilingani emas;
  `commands/status` bilan 12 soniyagacha kutiladi. Token 1 soatlik, keshlanadi.
- **Poster**: `token` query'da, o'qish GET, yozish JSON POST. ⚠️ **Narx
  tiyinda** (`"3500000"` = 35 000 so'm). ⚠️ **`status: 0` — "qabul qilingan"
  emas** ("onlayn buyurtma" bo'lib tushadi, odam qabul qiladi) → `unknown`.
  **Bekor qilish API'si yo'q** → `ErrUnsupported`, panelda "kassada bekor
  qiling". Qator izohi, yetkazish narxi va buyurtma turi uchun maydon yo'q —
  hammasi buyurtma izohiga yig'iladi, aks holda mijozning "piyozsiz"i yo'qoladi.
- **Clopos**: `x-token` sarlavhasi (`Authorization` emas).
  `auto_order_accept` va `auto_order_sent_to_station` **shart** — busiz
  buyurtma qabul qilinmagan holda turadi. Mahsulotlar sahifalanadi.
- ⚠️ **r_keeper restoran tarmog'i ichida**: XML interfeys restorandagi serverda
  (`https://<ip>:<port>/rk7api/v0/xmlinterface.xml`), ya'ni bulutdagi server
  unga port ochilmasa yoki VPN bo'lmasa **yeta olmaydi** (panelda shu
  ogohlantirish bor; UCS o'zi **r_k White Server** ni tavsiya qiladi).
  Narx **tiyinda**, miqdor **mingdan bir**da. `Cancel` ataylab
  avtomatlashtirilmagan — o'z ruxsati va izi bor kassa amali.

### Stop list (`/admin/stop-list` + kassadan avtomatik)
- ⚠️ **Nomi uch tilda ham "Stop list"** (`t.till.stopList`). Xonaning o'zi shu
  so'zni ishlatadi — oshxona ham, ilgari ishlatgan kassa tizimi ham. Tarjima
  qilingan yorliq bitta ekranga ikkinchi nom beradi, va narxi shovqinli
  peshtaxtada "stop listga qo'y" deb aytilayotgan odamga tushadi.
- **To'rt yozuvchi, to'rt ro'yxat**: `branch.soldOut` — peshtaxtadagi odam
  bosgani, `branch.posSoldOut` — kassadan ko'chirilgani
  (`handlers/posstop.go`), `branch.stockSoldOut` — omborning arifmetikasidan
  chiqqani (`handlers/stockstop.go`, §"Tannarx va ombor"). ⚠️ **Bitta maydonga
  qo'shilsa bir-birini bekor qiladi**: sinxronizatsiya oshxona qaytargan taomni yana stopga
  qo'yardi, peshtaxta bosgani esa kassa ushlab turgan stopni ochib yuborardi —
  va har biri "tugma ishlamayapti" bo'lib ko'rinadi. `IsSoldOut` uchalasini
  ham so'raydi, ya'ni sayt, savat, `CreateOrder` va combo tekshiruvi
  o'zgarmadi.
- ⚠️ **Bo'sh ro'yxat "hech nima stopda emas" degani emas.** Mahsulotsiz javob
  qaytargan kassa 200 bilan yiqilgan bo'ladi, va uni yaxshi xabar deb o'qish
  butun stop listni bir zumda sotuvga qaytarardi. Bunda oldingi ro'yxat
  saqlanadi, sabab esa `posSoldOutError` ga yoziladi.
- **Ulanmagan kassa oynani tozalaydi**: aks holda POS'ni almashtirgan restoran
  muzlab qolgan stop listni abadiy ko'tarib yurardi va uni paneldan ochish
  imkoni yo'q edi.
- **Kassadagi stopni paneldan qaytarib bo'lmaydi** (409, sababi bilan): tugma
  "ok" desa, keyingi sinxronizatsiya uni uch daqiqada qaytarib qo'yadi.
- **Bog'lanmagan taom hech qachon avtomatik stopga tushmaydi** — shuning uchun
  ekranda "kassaga bog'lanmagan" va bog'lash 0 bo'lsa ogohlantirish bor.
- Fon sikli har **3 daqiqada** (`cmd/server` dan ishga tushadi), qo'lda
  "Hozir o'qish" — `POST /admin/pos/stop-list/sync`. ⚠️ Sahifadagi eng foydali
  qator — **oxirgi o'qilgan vaqt**, "ulangan" bayrog'i emas: soat undan
  o'tishi bilan bayroq eskiradi (`lastEventAt`/`lastUpdateAt` bilan bir qoida).
- ⚠️ **Yopiq filial so'ralmaydi** (`syncOpenAt`): soat to'rtda hech kim qozon
  bo'shatmaydi, ya'ni bu javobsiz sarf — odatdagi ish vaqtida so'rovlarning
  ~40% i, hech nimadan voz kechmasdan. Ish vaqtidan **15 daqiqa oldin**
  boshlanadi: aks holda kun boshidagi ro'yxat bir interval davomida kechagi
  bo'lib turadi, ya'ni u tunda o'zgargan bo'lishi eng ehtimol bo'lgan paytda.
  Faqat fon sikli — paneldagi "Hozir o'qish" har doim ishlaydi.
  ⚠️ **Ish vaqti bo'sh bo'lsa — so'raladi.** `isOpenNow` bo'sh jadvalga
  `false` qaytaradi (mos hafta kuni topilmaydi), ya'ni uni to'g'ridan-to'g'ri
  o'qish sozlamani to'ldirmagan **har bir** restoranda oynani jimgina
  o'chirardi (bo'sh `mapProvider` = 2GIS bilan bir qoida).
  ⚠️ Sahifada **`paused`** ko'rsatiladi: busiz har tundan keyin "oxirgi
  o'qilgan: 9 soat oldin" chiqadi va buzuq integratsiyaga o'xshaydi — eganing
  keyingi qadami hech qachon xato bo'lmagan kalitlarni qayta kiritish bo'lardi.

#### Stop listga taymer: taom o'zi qaytadi

Kassir "lag'mon tugadi" deb bosadi, oshxona yarim soatdan keyin yangisini
qo'yadi — va taom stop listda qolib ketadi, chunki uni qaytarish **hech kimning
ishi emas**. Endi to'xtatishda muddat tanlanadi (`branch.soldOutUntil`).

- ⚠️ **Muddat o'qiladi, tozalanmaydi.** Yarim tunda ishlaydigan job ikkinchi
  yozuvchi bo'lardi, qulf talab qilardi, konteyner qayta ishga tushganda
  to'xtardi — va restoran buni har bir muddatli taom menyuda qolib ketgan
  ertalab bilib olardi. `IsLimitSoldOut` bu dalilni allaqachon keltirgan; bu —
  boshqa soat uchun o'sha qoida.
- ⚠️ **Muddati o'tgan yozuv ro'yxatda qoladi va bu zararsiz** — chunki hech kim
  massivni to'g'ridan-to'g'ri o'qimaydi. Har bir o'quvchi `IsManualSoldOut` dan
  o'tishi **shart**: massivni to'g'ridan-to'g'ri tekshirgan chaqiruv taomni
  taymer allaqachon bo'shatgandan keyin ham menyudan tashqarida ushlab turardi,
  va xona buni aynan "taymer ishlamayapti" deb o'qiydi. Kassaning savdo
  panjarasi (`soldOutIDs`) shu tarzda ushlandi: ikki ekran, bitta filial,
  qarama-qarshi javob.
- ⚠️ **Parallel ro'yxat, `soldOut` ning elementi emas.** U massivni sayt, savat,
  kassa, combo tekshiruvi va ikkita migratsiya `containsID` orqali o'qiydi;
  element tipini o'zgartirish — bitta o'tkazib yuborilgan chaqiruvi taomlarni
  jimgina to'xtatmay qo'yadigan keng tahrir. `dailyLimits` + `limitSoldOut`
  juftligi bilan bir naqsh.
- ⚠️ **Soat serverniki.** Ekran **davomiylik** yuboradi ("2 soat"), lahza emas:
  CMOS batareyasi o'lgan monoblok elektr o'chgandan keyin 2010-yilni ko'rsatadi
  (oflayn chek vaqtlari shu sababdan `clampOfflineTime` bilan qisqartiriladi),
  va o'sha mashinada hisoblangan muddat yo darhol tugardi, yo hech qachon
  tugamasdi. Ikkalasi ham ekranda hech nima demaydi.
- ⚠️ **"Yopilguncha" filialning o'z jadvalidan** hisoblanadi. Mahalliy yarim tun
  bo'lsa, soat ikkigacha ishlaydigan joyda taom **xizmat o'rtasida** qaytardi.
  Ikkiga yopiladigan xona **ertaga** yopiladi: bugungi deb o'qilsa muddat
  allaqachon o'tgan bo'ladi va stop keyingi o'qishda bo'shaydi — aynan qamrashi
  kerak bo'lgan kechada.
- ⚠️ **Jadvali bo'sh filial — kun oxirigacha**, va yo'nalish ataylab shunday:
  jadvalni to'ldirmagan install ko'pchilik, va darhol tugaydigan zaxira tugmani
  aynan o'shalarda buzuq ko'rsatardi (bo'sh `mapProvider` = 2GIS bilan bir
  o'qish).
- ⚠️ **Standart — muddatsiz.** Tugma shu paytgacha shuni qilgan, va birov uchun
  tanlab qo'yilgan muddat haqiqatan tugagan taomni menyuga qaytarardi — buni
  mehmon buyurtma qilgunча hech kim sezmaydi.
- **Yuqori chegara — 24 soat**: undan narisi uchun halol sozlama "muddatsiz", va
  hech kim yonida bo'lmaydigan muddat keyingi smenani ajablantiradi. Brauzer ham
  shu chegaraga qisqartiradi, aks holda maydon muddat hurmat qilmaydigan raqamni
  ko'rsatib turardi.
- **Tayyor tugmalar yonida daqiqa maydoni ham bor.** Oshxona "yigirma daqiqa"
  deb ham aytadi, va faqat yumaloq raqam taklif qiladigan ekran odamni eng
  yaqinini tanlashga majbur qiladi — keyin taom nega erta qaytganini hech kim
  eslay olmaydi, ya'ni taymerga ishonch aynan shu yerda yo'qoladi.
  ⚠️ **Bo'shatilgan maydon "muddatsiz" degani** ("nol daqiqaga to'xtat" emas —
  u o'tib ketgan muddat va menyudan umuman chiqmagan taom bo'lardi); yonidagi
  partiya maydoni allaqachon shu qoidada.
  ⚠️ **"≈ 21:35" ko'rinishidagi oldindan ko'rsatish yo'q**: uni hisoblaydigan
  soat aynan ishonib bo'lmaydigani, va butun xususiyat shu sababdan davomiylik
  yuboradi. Server javobi bir lahzadan keyin nishonda chiqadi.
- Nishonda **tirik teskari hisob** so'zning o'rnini oladi ("14 daq"), yoniga
  qo'yilmaydi: kartada bitta qator joy bor, va "qachon qaytadi" — "tugadimi" dan
  foydaliroq javob. Panelda ham shunday, aks holda ega ikki soatlik stopni
  butunlay olib tashlangan taomdan ajrata olmay, borib so'rardi.
  - ⚠️ **O'z komponentida**, aks holda bir soniya ikki yuzta kartani qayta
    chizadi: bu ekran monoblokda butun kechani ochiq turadi.
  - ⚠️ **Serverning lahzasiga nisbatan sanaladi, mahalliy kamaytirilmaydi.**
    Smena davomida ochiq turgan ekran suriladi, va suriladigan hisoblagich
    o'n daqiqa oldin qaytgan taom haqida "2 daq" derdi — hisoblagichsizdan
    yomonroq, chunki unga qarab ish qilinadi.
  - ⚠️ **Nolga yetganda karta o'zi hal qilmaydi — ro'yxatni qayta so'raydi.**
    Stopni server bo'shatadi, va o'zicha ag'darilgan karta taom sotuvdami degan
    savolga ikkinchi fikr bo'lardi — stop list aynan shundan qochish uchun
    qurilgan.
  - Oxirgi daqiqa **sekundlarda**: aynan shu daqiqani odam kutib turadi, va
    oltmish soniya davomida "0 daq" qotib qolgan ekranga o'xshaydi.

⚠️ **Muddat allaqachon stopdagi taomga ham qo'yiladi, va aslida u eng ko'p
ishlatiladigan holat.** Birinchi versiyada muddat faqat **to'xtatishda**
so'ralardi, holbuki haqiqiy voqea teskari: kassir somsani stopga qo'yadi, keyin
oshxonaga qo'ng'iroq qiladi — «qancha vaqtda?» — «15 daqiqada». O'sha javobni
yozishning yagona yo'li taomni sotuvga qaytarib, qaytadan to'xtatish edi: taom
qisqa vaqt buyurtma qilinadigan bo'lib qoladi, va buni hech kim ikki marta
qilmaydi.
- ⚠️ **Toggle emas, alohida amal.** `soldOut: false` yuborib qayta to'xtatish
  taomni so'rov ketib kelguncha sotuvga chiqarardi — va aynan o'sha oynada mehmon
  buyurtma qiladi. Server allaqachon ushlab turgan taomga muddat qabul qilardi;
  faqat ekranlar so'ramasdi.
- ⚠️ **Ikki natija bitta tugmani bo'lishmaydi**: biri taomni stopda qoldirib
  faqat qachon qaytishini yozadi, ikkinchisi uni hoziroq sotuvga chiqaradi.
- Tayyor variantlarga **15 va 30 daqiqa** qo'shildi — oshxona aynan shunday
  javob beradi. «Muddatsiz» esa faqat to'xtatishda ko'rinadi: allaqachon
  o'chirilgan taomda u hech nimani o'zgartirmaydigan tugma bo'lardi, va hech
  nima qilmaydigan boshqaruv elementi xonaga ekran ishonchsizligini o'rgatadi.
- Amallar jurnaliga ham yoziladi ("21:00 gacha"): "juma kuni lag'mon nega
  o'chirilgan edi" keyingi hafta so'raladi, va "kechqurunga qadar" bilan "birov
  qaytarishni unutgan" — ikki xil javob.
- ⚠️ **Ikkala eshik ham muddat qo'yadi, va qoidasi bitta modulda**
  (`lib/stopHold.ts`): tayyor variantlar, bo'sh maydonning ma'nosi va 24 soatlik
  chegara. Serverdagi `soldOutHeldBy` aynan shu sababdan bitta funksiya —
  bir mahsulotda "2 soat" ning ikki xil o'qilishi restoranni "qaysi ekran
  yolg'on gapiryapti?" degan savol oldida qoldiradi. Umumiy bo'lmagani —
  **faqat ko'rinish**: kassa monoblokda barmoqqa mo'ljallangan tugma chizadi,
  panel forma chizadi.
- ⚠️ **Panelda dialog faqat to'xtatishda ochiladi**, qaytarish esa avvalgidek
  bitta bosish bo'lib qoladi: qaytarishning muddati yo'q, va uning oldiga savol
  qo'yish hech kim so'ramagan savol bo'lardi.

### Bozorlik: bozorchi ilovadan yozadi, ombor o'zi ko'tariladi

Restoranning eng ko'p uchraydigan kirimi — nakladnoy emas, **bozorlik**: ertalab
soat oltida naqd pul bilan borilgan, qaytgani qog'oz parchasiga yozilgan, ofisga
olib kelingan va vaqti bo'lgan odam tomonidan terilgan. Ko'pincha bir kun kech,
ba'zan umuman terilmagan — va undan keyingi hamma narsa shunga suyanadi: javon,
**keyingi** bozorlikni belgilaydigan xarid ro'yxati, har taomning tannarxi, stop
list.

- **Yangi ruxsat `PermBuy` va yangi rol «Zakupshik»**. ⚠️ **`stock` ning ichiga
  qo'shilmadi, chunki ikkisi deyarli teskari**: omborchi javonni sanaydi va
  buning evaziga **panelga kirish** oladi (`stocklogin.go`); bozorchi panelni
  umuman ko'rmaydi va sanash qila olmaydigan ishni qiladi — **narx yozadi**, va
  bozorda yozilgan narx o'sha masalliqli har bir taomni qayta narxlaydi.
  Qo'shilsa, ikkalasi bir-birining eng qimmat tugmasini olardi.
- ⚠️ **Mavjud install uchun alohida migratsiya** (`EnsureBuyerRole`).
  `seedStaffRoles` nom bo'yicha upsert qilishdan **ataylab** bosh tortadi:
  «Ofitsiant» ni qayta nomlagan restoran har restartda ikkinchisini olardi.
  Marker `migration_state` da va u **tashrifni** yozadi, natijani emas — aks
  holda rolni endigina o'chirgan restoran uni har boot'da qaytadan olardi.
- ⚠️ **O'nta ilgak emas, panelning xarid ro'yxatining o'zi** (`shoppingList`
  bitta funksiya). Bozorda turgan odam va ofisdagi ega bitta javonlarga qaraydi;
  ikkinchi implementatsiya ularni "go'sht tugadimi?" savolida qarama-qarshi
  qo'yardi, va bu bahs pul sarflangandan **keyin** bo'ladi.
- ⚠️ **Filial xodimdan olinadi, so'rovdan emas** — har bir `/staff/*` endpoint
  qoidasi. Aks holda bozorchi hech qachon turmagan oshxona uchun xarid qilardi.
- **Kirim darhol tushadi, ega xabar oladi.** Ertalab soat oltida hech kim
  tasdiqlamaydi, va mahsulot allaqachon binoda. ⚠️ Xabar **bildirishnoma**,
  `LossAlert` **emas**: qo'ng'iroq bitta g'ayrioddiy hodisa uchun, bozorlik esa
  restoran qiladigan eng oddiy ish — uni u yerga yozish kunlik chegarani
  odatiy ishga sarflab, muhim kechada kanalni o'chirib qo'yardi.
- ⚠️ **`clientId` — kirimning oflayn kafolati**, va indeks uni ta'minlaydi
  (`purchase.clientId` sparse unique). Bozorda signal zaldagidan yomon: telefon
  yozuvni saqlab qayta yuboradi, va id'siz qayta yuborish **ikkinchi kirim**
  bo'lardi — javon ikki marta ko'tariladi, hisob ikki marta to'lanadi, narx
  tarixiga bir narx ikki marta yoziladi. Kassa chekining `clientId` si bilan bir
  naqsh va bir sabab.
- ⚠️ **Vaqt: kirim kelajakka sanalmaydi.** Bo'lmagan kundagi narx bugun hech
  nimaga, o'sha kundan keyin hammasiga tegadi — kechikkan ta'sirli xato, va uni
  hech kim telefonga bog'lay olmaydi. Panelning formasi ham shu chiziqni
  chizadi.
- ⚠️ **Katalogda yo'q narsa — eng jim buziladigan joy.** Rad etilsa bozorchi
  yarim bozorlikni yoza olmaydi, ya'ni **umuman yozmay qo'yadi** (bu dars
  yetkazib beruvchi maydoni va void sababi bilan allaqachon to'langan). Erkin
  qo'shilsa katalog «pomidor / Pomidor / tomat» bo'lib to'ladi va texkartalar
  hech biriga ulanmaydi. Shuning uchun: **qo'shiladi, lekin belgilanadi**
  (`ingredient.needsCare`), nom avval mavjudlariga **registrga qaramay**
  solishtiriladi, va birligi **dona** qo'yiladi — taxmin qilingan «kg» bog'lam
  bilan sotiladigan narsada tannarxni uch daraja adashtiradi va **ataylab
  qilingandek** ko'rinadi.
- ⚠️ **Belgini faqat ega saqlashi o'chiradi**, vaqt emas: muddat bilan
  o'chadigan bayroq yarim yozuvni hech kim tanlamagan kunda to'liq yozuvga
  aylantirardi.
- **Zagotovka katalogda ko'rsatilmaydi**: sous pishiriladi, sotib olinmaydi, va
  uni bozorda taklif qilish javonga hech qachon turmagan idishni yozardi.
- **Ilova alohida emas, `Keel Team` ichida** — bo'lim faqat `buy` ruxsati bor
  hisobda ko'rinadi (ofitsiantga KDS ko'rinmagani bilan bir qoida). Bozorchi
  ham xodim: davomati va ish haqi o'sha ilovada, ya'ni ikkinchi ilova unga
  ikkita o'rnatish va ikkita login bo'lardi.

#### Bozorlik ro'yxati: nima so'ralgani ham yozib qo'yiladi

Kirim nima **qaytganini** aytadi. Nima **so'ralgani** hech qayerda yozilmasdi —
shuning uchun «yana go'sht tugab qoldi» degan gapning javobi yo'q edi: u
ro'yxatga umuman kirmaganmi, ro'yxatda bo'lib olinmaganmi, yoki olinib
kutilgandan tez tugaganmi. Uchta boshqa muammo, uchta boshqa yechim, va restoran
faqat alomatni ko'rardi.

- ⚠️ **Ikkita ruxsat, va bo'linish — nazoratning o'zi.** `buyorder` ro'yxatni
  yozadi, `buy` uni bozordan oladi. Bitta hisobda bo'lsa ro'yxat safar ustidan
  nazorat bo'lishdan to'xtaydi va bozorchining o'ziga yozgan eslatmasiga
  aylanadi — ya'ni bu xususiyat nima uchun so'ralgan bo'lsa, o'shani yo'qotadi.
- ⚠️ **Forma bo'sh varaqdan boshlanmaydi** — ombor arifmetikasi hisoblagan
  kam qolganlar bilan to'ladi (`shoppingList`, panel o'qiydigan o'sha funksiya).
  Aks holda restoranda ikkita bozorlik ro'yxati bo'lardi (biri hisoblangan, biri
  yozilgan) va bozorchi qaysi biri rost ekanini bilmasdi — u oxirgi ko'rganiga
  ergashadi.
- ⚠️ **Bozorchining ekranida yuborilgan ro'yxat hisoblanganini
  ALMASHTIRADI**, yoniga qo'yilmaydi — xuddi shu sababdan.
- ⚠️ **Birlik erkin matn emas, va bu jimgina buziladigan joy edi.** Bozor
  myatani bog'lamda, unni qopda, tuxumni patnisda sotadi; ombor kilogramm va
  donada sanaydi. Tarozisiz turgan odam ko'rgani — «5» — kilogrammdagi maydonga
  tushsa, javonga chorak kilo o'rniga besh kilo qo'yiladi. Hech nima xato
  bermaydi: raqam yigirma barobar katta, stop list hech qachon ishlamaydi, va
  farq bir oydan keyin inventarizatsiyada, taxtani ushlab turgan odamdan
  tushuntirish so'ralganda chiqadi.
  **Yechim — masalliqdagi ixtiyoriy `packName` + `packQty`** («bog'lam» = 0.05
  kg, «qop» = 50 kg, «patnis» = 30 dona). Bozorchi birlikni **bosadi**, yozmaydi,
  va bozorda arifmetika qilmaydi.
  - ⚠️ **Konvertatsiyani server qiladi, telefon emas.** Koeffitsiyent
    masalliqning fakti va natija **javonga tushadi**; ekran o'zi hisoblagan
    kilogrammni yuborsa, bu tizimda hech kim tekshira olmaydigan raqamning
    ikkinchi implementatsiyasi bo'lardi. Telefon nima bosilganini yuboradi
    (`pack: true`), server ko'paytiradi.
  - ⚠️ **Narx bo'linadi, ko'paytirilmaydi**: 0.05 kg lik bog'lam 3 000 so'm
    bo'lsa — kilogrammi 60 000. Bu yerdagi xato miqdordagi xatoning teskarisiga
    tushadi va keyin xuddi shunday oddiy ko'rinadi.
  - ⚠️ **Ikkala maydon ham, yoki hech biri.** Nomsiz o'lcham miqdorni ekranda
    ko'rinmaydigan koeffitsiyentga ko'paytirardi — ya'ni maydon oldini olish
    uchun qo'yilgan xato, uni sozlaydigan forma orqali kirib kelardi. Server
    yarim to'ldirilgan juftlikni tashlaydi.
  - ⚠️ **Bozorlik ro'yxati doim saqlash birligida yoziladi.** Ro'yxat birovning
    ertalabki ishiga boradi, va ba'zan qop, ba'zan kilogramm turadigan hujjatni
    har bir o'quvchi qaysi biri ekanini bilishga majbur qilardi — jumladan
    yakunlangan safarni kirimga aylantiradigan arifmetika ham.
  - Preview'da ikkalasi ham o'qiladi («2 qop = 100 kg»): ro'yxat ketishidan
    oldin «2» bir ma'noli bo'lishi kerak.

  ⚠️ **Tozalash yo'qolishi bunga aloqador emas va allaqachon hal qilingan**:
  texkarta miqdorlari **brutto** (§ "Tannarx va ombor"), va tozalash alohida ish
  bo'lsa u **zagotovka + chiqim** bilan yoziladi — «1000 g myatadan 400 g
  toza barg». Yangi mexanizm kerak emas.
- ⚠️ **Belgilash javonni qimirlatmaydi.** Qator belgilanishi — ro'yxat haqidagi
  fakt; ombor faqat safar **yakunlanganda** o'zgaradi. Belgilashda ko'tarilsa,
  bozorchi hali bozorda turganda javonga ovqat qo'yilardi, va bekor qilish uni
  qaytarib olishga majbur bo'lardi — bu tuzatishni tizimda hech nima
  o'g'irlikdan ajrata olmaydi.
- ⚠️ **Kirim bitta funksiyadan yoziladi** (`recordMarketRun`): erkin bozorlik
  ham, yakunlangan ro'yxat ham. Ikki yozuvchi «to'langanmi», «vaqti safarniki
  ekanmi», «katalogda yo'q nom bilan nima bo'ladi» degan uchta savolga ikki xil
  javob berardi — uchalasi ham bittadan xatboshi bilan hal qilingan qoidalar.
- ⚠️ **«Yo'q edi» — nol miqdor emas, alohida javob.** Hech kim tegmagan qator
  bilan qidirib topilmagan qator — ikki xil fakt, va faqat ikkinchisi bo'yicha
  yetkazib beruvchiga qo'ng'iroq qilish kerak.
- **So'ralgani va olingani yonma-yon saqlanadi**: «o'ntasi so'ralgan, oltitasi
  olingan» — bu hujjat aynan shu gapni aytish uchun bor; natijani so'rovning
  ustiga yozish bozorlikda ilgari bo'lgan jimlikni qaytarardi.
- **Kim qayerda yozadi**: kassada ham, ilovada ham — `buyorder` ruxsati bor har
  kim (rol nomidan emas). ⚠️ Ilova ilgari buni yana toraytirardi
  (`buyorder && (void || stock)`) va bu noto'g'ri edi — qarang «Bozorlik yoza
  oladi — rolda emas, odamda».
- **Yangi rol «Omborchi»** (`stock` + `buyorder` + `stockissue`) — javonni sanaydi va ro'yxat
  yozadi, lekin bozorga bormaydi. ⚠️ Mavjud installarga migratsiya bilan
  keladi, va `Menejer` / `Kassir` / `Ish boshqaruvchi` ga `buyorder` bir marta
  qo'shiladi (`buyOrderGranted` markeri **tashrifni** yozadi, natijani emas —
  aks holda endigina olib tashlangan ruxsat har boot'da qaytardi).

#### So'rov ikkiga bo'linadi: bozordan keladigani va skladda turgani

Bozorlik ro'yxati bitta odam yozadigan bitta varaq edi, va u varaqdagi hamma
narsa **bozorchiga** ketardi. Lekin barmen ertalab yozadigan narsa aslida ikki
xil: «5 blok kola, 5 kg shakar» — binoda bor, «5 kg limon, 5 kg apelsin» — yo'q.
Birinchisini bozorga borib sotib olish — pulni ikki marta sarflash; ikkinchisini
omborchidan so'rash — javob berib bo'lmaydigan so'rov.

- ⚠️ **Qayerdan kelishi — tovarning fakti, so'rovniki emas** (`ingredient.source`).
  Barmen bo'sh muzlatgich haqida o'ylayapti, kim nimani olib kelishi haqida emas;
  ro'yxatni ikkiga ajratish — **ombor tarkibini bilish**, va bu aynan uning
  ishiga kirmaydigan yagona narsa. Ajratishni katalog qiladi, ro'yxat o'zi
  bo'linadi.
- ⚠️ **Bo'sh — «bozordan», va shunday qolishi shart.** Bu maydondan oldingi butun
  katalog bo'sh, ya'ni nol qiymatni boshqacha o'qish har bir install'ning hamma
  masallig'ini ekranni hech qachon ochmagan omborchiga jo'natardi — jimgina, va
  har bir so'rov «bajarilayotgandek» ko'rinib. Bo'sh `mapProvider` = 2GIS bilan
  bir qoida.
- ⚠️ **Katalogda yo'q nom — bozorga.** Uni hech kim bu yerdagi javonga qo'ymagan,
  demak omborchiga borgan so'rov butun ertalab javobsiz yotadi va har ekranda
  aynan «kimdir bilan shug'ullanayotgan so'rov» bo'lib ko'rinadi.
- ⚠️ **Bitta so'rov — ikkita hujjat, bitta `groupId`.** Bitta hujjat ichidagi
  ikki xil qator emas: omborchi o'n daqiqada javob beradi, bozorchi to'qqizda, va
  umumiy hujjat ertalabni ikkalasida ham so'z yo'q holatda o'tkazardi («yarmi
  yuborilgan» degan holatni ikkala ish ham bilmaydi). Panel esa so'rovni
  **guruh** bo'yicha ko'rsatadi, chunki ro'yxatni ikkiga bo'lish — serverning
  fikri, barmenning emas.

#### Uchinchi holat: «yuborildi» hali «yetib keldi» emas

Ilgari ikki holat bor edi: ro'yxat yozildi, keyin u kirim bo'ldi. O'sha ikki
fakt orasida odam sumkani shahar bo'ylab olib o'tadi, orqa eshikda kimgadir
beradi va uyiga ketadi. Bu topshirishni hech nima yozmasdi — ya'ni **kelmagan
kilo** bilan **umuman olinmagan kilo** keyinchalik bir xil ko'rinardi, va nihoyat
farqni topgan inventarizatsiya taxtani ushlab turgan odamni ayblardi.

`sent → shipped → done`, va **javon uchinchisida qimirlaydi**:
- ⚠️ **Kirim (`purchase`) qabul qilinganda yoziladi, sanalgan miqdorda.**
  Bozorchining raqami — uning da'vosi, restoran sanagani — fakt. Bozorda yozilgan
  kirim mahsulotni hali avtobusdagi sumkada turganda javonga qo'yardi, va undan
  keyingi har bir tuzatishni tizimda hech nima o'g'irlikdan ajrata olmasdi.
- ⚠️ **Tegilmagan qator yuborilganidek qabul qilinadi** (`ShoppingLine.Took`).
  Hech nimani qayta yozmay imzolagan odam «hammasi to'g'ri» deyapti — bu oddiy
  holat, va nolga tushiradigan standart eng tez rozi bo'lgan odamning butun
  ro'yxatini bo'shatardi. Sanalgan **nol** esa alohida javob: sumka bo'sh kelgan.
- ⚠️ **Erkin bozorlik (ro'yxatsiz) o'zgarmadi va darhol tushaveradi.** Uni hech
  kim so'ramagan, demak sanab oladigan odam ham yo'q: unga qabul bosqichi
  o'ylab topilsa, kirim mavjud bo'lmagan odamni kutib abadiy ochiq qolardi.
- ⚠️ **Yuboruvchi va qabul qiluvchi — ikki maydon, lekin kod ikkinchisini
  birinchisidan farq qilishga majburlamaydi.** Yakshanba kuni ikkalasini bir odam
  qiladigan restoran ertalabni baribir yopa olishi kerak; muhimi — «kim yetib
  keldi dedi» degan savolga javob **alohida** yozilgani.

#### Skladdan chiqarish: hujjat, arifmetika emas

Omborchi kolani barga berganda **hech qanday qoldiq o'zgarmaydi**, va bu — model,
kamchilik emas: masalliqning uyi bitta (`ingredient_placement`), sotuv esa chek
urilganda allaqachon spisat qiladi. Chiqimni yozish bitta shishani ikki marta
ayirardi, va farqni topgan sanoq barmenni ayblardi.

- ⚠️ **Ikki filial bo'lsa — boshqa gap.** `branch.supplyBranchId` boshqa filialni
  ko'rsatsa, chiqarish **jo'natmaga** (`dispatch`) aylanadi: jo'natuvchi javoni
  yuklanganini yo'qotadi, qabul qiluvchi sanab olinganini oladi. Yangi hujjat
  emas — mavjudi, chunki harakat hisoboti va ikkala balans aynan undan o'qiladi.
- ⚠️ **Bo'sh — «o'z ombori»**, va bu hozirgacha o'rnatilgan har bir restoran.
  Filial **o'zini** ko'rsata olmaydi: ikki uchi bitta javon bo'lgan nakladnoy
  bitta kilogrammni ayirib, qo'shib, hech kim imzolay olmaydigan qog'oz qoldirardi.
- ⚠️ **Omborchining tikkasi narx yozmaydi.** Tovar bir marta sotib olingan; bu
  yerda yozilgan ikkinchi narx tarixga kirim bo'lib tushardi va o'sha masalliqli
  har bir taomni qayta narxlardi — ikki xona orasidagi yumushning eng baland
  ovozli xato qilish usuli.
- ⚠️ **Yangi ruxsat `PermStockIssue`, `stock` ning ichida emas.** Javonni sanash
  va uni bo'shatish — ikki xil ish: sanoq nima turganini yozadi, chiqarish esa
  kimga tegishini hal qiladi. Qo'shilsa, muzlatgichni sanash uchun telefon
  berilgan har bir odam bir yashik aroqni shahar bo'ylab jo'natish tugmasini ham
  olardi — restoran esa buni hech qachon so'ralmay bergan bo'lardi.

#### «Bozorlik yoza oladi» — rolda emas, odamda

`buyorder` roldan chiqardi, va ilova uni yana toraytirardi: `buyorder && (void ||
stock)`, ya'ni «boshqaruv yoki omborchi». Bu **kim tugaganini sezishi** haqida
noto'g'ri o'qish edi — bu bo'sh muzlatgichli barmen va unning oxirini ishlatgan
oshpaz, va ikkalasi ham chek bekor qilmaydi, javon ham sanamaydi.

- **Xodim kartochkasida bitta katakcha** (`staff.canBuyOrder`), va bu shu
  mahsulotdagi **yagona odamga beriladigan ruxsat**. Qolgan hammasi — ish: kassir
  pul oladi, oshpaz peshtaxtani boshqaradi, va javob o'sha ishni qilayotgan har
  kimda bir xil. «Ro'yxat yoza oladigan Barmen» degan ikkinchi rolni yasashga
  majbur bo'lgan restoran har odamga bitta rol bilan tugaydi — rollar aynan
  shuning oldini olish uchun bor.
- ⚠️ **Faqat qo'shadi, hech qachon olib qo'ymaydi.** Roli allaqachon bergan odamda
  bu katakcha bo'sh turishi hech nimani o'zgartirmaydi: ba'zan beradigan, ba'zan
  olib qo'yadigan tugma ikkinchi hujjatga qarab o'qiladi, va o'sha o'qish
  shoshilgan kuni noto'g'ri qilinadi. Ilovadagi ikkinchi shart ham shu sababdan
  olib tashlandi — u eganing endigina qo'ygan galochkasini jimgina bekor qilardi,
  va ekran buning sababini aytadigan joyga ega emas.
- **Telefon xabar oladi** (`notifyOrderQueue` / `notifyOrderShipped`): so'rov
  ruxsatga yuboriladi, odamga emas — «omborchi» ikki odam almashib qiladigan ish,
  va hisobga yozilgan xabar har ikkinchi hafta o'qilmay qolardi, panelda esa bu
  so'rovlarga e'tibor bermaydigan omborchidek ko'rinardi.
- ⚠️ Xabar **bildirishnoma**, `LossAlert` emas — bozorlik bo'yicha yozilgan
  qoidaning o'zi: kanal haftada bir marta bo'ladigan g'ayrioddiy hodisa uchun.


#### Seyf: pul jismonan qayerda

⚠️ **Bu — joy, foyda-zarar emas, va butun dizayn shu farqqa suyanadi.**
Moliyaviy hisobot «foyda qildikmi» ga javob beradi; seyf «pul qayerda» ga.
Yashikdan seyfga o'tgan pul **chiqim emas**, bozorchiga berilgan pul esa mahsulot
sotib olgunga qadar **sarflanmagan**. Joyning harakatlarini chiqim deb sanash —
bir pulni ikki marta ayirish. Shuning uchun seyf moliyaviy hisobotga
**umuman tegmaydi**, va buni test qo'riqlaydi.

⚠️ **Har harakat — bitta hujjat, va hech nima chiqarib olinmaydi.** Vasvasa —
seyfni mavjud yozuvlardan (podotchet, inkassatsiya, oylik) hisoblash. Lekin
ularning har biri seyf haqida **ba'zan** bo'ladi: podotchet eganing cho'ntagidan
ham berilishi mumkin, oylik kartaga o'tkazilishi mumkin, yashik esa bankka
ketadigan sumkaga bo'shatilishi mumkin. Pul qayerga ketganini aytmaydigan
hujjatlardan chiqarilgan balans — hech kim ochib ko'rmagan quti haqidagi
ishonchli raqam bo'lardi.

Shuning uchun boshqa ekranlar pul aniq seyfdan chiqqanda/kirganda bitta qator
**taklif qiladi** (podotchet formasidagi «seyfdan» katakchasi), va o'sha qator
`refKind` + `refId` bilan bog'lanadi.
- ⚠️ **Juftlik unique** (sparse): qayta urinish yoki ikki marta bosish o'sha
  hand-over'ni daftariga ikkinchi marta yoza olmaydi. Balansdagi dublikat —
  xatolikning eng yomon turi: ishonarli, noto'g'ri, va tizimda hech nima uni
  sezmaydi.
- ⚠️ **Chaqirgan ishini yiqitmaydi**: ega podotchet berолmay qolishi ikkinchi
  yozuv tushmaganidan yomonroq. Qatorni keyin qo'lda qo'shsa bo'ladi, pul
  berishni esa orqaga qaytarib bo'lmaydi.
- ⚠️ **Belgilanmagan katakcha — «seyfdan emas»**, va bu to'g'ri javob: pul
  boshqa joydan kelgan bo'lishi mumkin.

⚠️ **Balans nolda to'xtamaydi.** Manfiy seyf — daftarга kirim yozilmay qolgani,
va uni yashirish odam pulni sanab ko'rgan yagona ekranni noto'g'ri sanoq bilan
kelishtirib qo'yardi.

⚠️ **Uchta joy — uchta ekran, ataylab**: yashik (smena oxirida sanaladi, kassa
ekranida), podotchet (odamning hisobi, kirimlar yonida), seyf (ofisdagi quti).
Uchalasini bitta songa qo'shadigan ekran — hech kim hech nimaga solishtira
olmaydigan raqam bo'lardi.

**Faqat ega ko'radi**: yashikdagi pulni smena oxirida uni ishlagan odam sanaydi;
seyfda qancha borligini kim ko'rishi mumkinligi — bu boshqa savol.

##### Seyfga ulangan to'rt harakat, va nega beshinchisi yo'q

Seyf daftariga qator yozadigan ekranlar: **podotchet** (`SafeRefAdvance`),
**naqd oylik** (`SafeRefSalary`), **kassa kirim/chiqimi** (`SafeRefCash`) va
**boshqa xarajat** (`SafeRefExpense`). Har birida bitta katakcha, va katakcha
har doim **so'raladi**:
- ⚠️ **Yo'nalishdan chiqarib olinmaydi.** Yashikdan chiqqan pul eshikdagi
  yetkazib beruvchiga ham, seyfga ham ketishi mumkin; kartaga o'tkazilgan oylik
  ham xuddi to'langan, lekin qutini bo'shatmaydi. Taxmin qilgan balans — hech
  kim ochmagan quti haqidagi ishonchli raqam.
- ⚠️ **Inkassatsiyada tomonlar teskari**: kassadan **chiqim** — seyfga
  **kirim**. Yo'nalishni nusxalash ("ikkala joyga ham yozamiz" degan tabiiy
  o'qish) har inkassatsiyani u to'ldirayotgan seyfdan ayirardi, va har bir
  alohida qator to'g'ri ko'rinardi. Testi bor
  (`TestACollectionFillsTheSafeItLeavesTheDrawerFor`).
- Kassa qatorini panel ham, kassa ekrani ham **bir eshikdan** yozadi
  (`addCashEntry`) — ikki tomonda ikki implementatsiya birinchi tahrirda
  ajraydi.

Beshinchisi — **buyurtma tushumi** — ataylab yo'q: pul yashikка tushadi, seyfga
esa faqat inkassatsiya bilan boradi, va uni avtomatik yozish sanalmagan pulni
sanalgan deb ko'rsatardi.

Keyin yana ikkitasi ulandi: **kuryer ish haqi** (`SafeRefCourierPay`) va
**yetkazib beruvchiga to'lov** (`SafeRefPurchase`). Oxirgisi — pul seyfdan
chiqishining eng ko'p uchraydigan yo'li, va u eng oxirida topildi: «to'landi»
tugmasi yetkazib beruvchi bilan hisob-kitob yopilganini aytardi va **qaysi
quti yengillashganini umuman aytmasdi**.
- ⚠️ Bu endpoint ilgari **umuman body qabul qilmasdi**, shuning uchun decode
  xatosi so'rovni rad etmaydi (`_ = httpx.Decode`): eski ekran hech nima
  yubormasa ham to'lov o'tishi kerak. Testi bor.

#### Boshqa xarajatlar: hisobotni optimist qilgan qator

Kirimlar sanalardi, oyliklar sanalardi, qolgan hamma narsa — **ijara, svet,
gaz, suv, soliq, ta'mir, kuryerlarning ish haqi** — hech qayerda. Ya'ni
"kirim − chiqim" har oy oyning o'zidan **yaxshiroq** chiqardi, taxminan bino
qancha turса shuncha, va **har oy bir tomonga**. Doim bir tomonga xato
qiladigan raqam — restoran ishonishni o'rganadigan raqam.

- ⚠️ **Faqat o'z hujjati bo'lmagan narsa.** Kirim — `purchase`, oylik —
  `staff_payment`, va ikkalasining hisobotda o'z qatori bor. Ularni bu yerga
  ham yozish bir pulni ikki marta sanaydi, va ikki marta sanalgan xarajat
  haqiqiysidan farq qilmaydi. Buni ekranning o'zi aytadi — kimdir bilishiga
  tayanmaydi.
- **Kategoriya — erkin matn + takliflar**, qat'iy ro'yxat emas: har restoran
  qo'shnisi to'lamaydigan narsaga pul to'laydi, va yopiq ro'yxat hammasini
  "boshqa" ga jo'natadi.
- ⚠️ **To'lov usuli yoziladi** (naqd / o'tkazma / karta), chunki u qutining
  yengillashganini hal qiladi — bu buxgalteriya nozikligi emas, seyf balansi
  sanoq bilan to'g'ri kelishi yoki kelmasligining farqi.
- ⚠️ **Sana — pul ketgan kun**, yozilgan kun emas (kirim qog'ozidagi qoida),
  va **kelajakka qo'yib bo'lmaydi**: oldinga sanalgan xarajat hali hech kim
  o'qimagan oyga tushadi va o'qilganda uni jimgina o'zgartiradi.
- ⚠️ **Xarajatni o'chirish seyfdagi qatorga tegmaydi**: pul jismonan chiqib
  ketgan. O'chirish "bu xarajat emas edi" deydi, "pul joyida" demaydi.

#### Kuryerga ish haqi: hisoblangani bor edi, berilgani yo'q

Tizim har yetkazishda kuryer qancha ishlab topganini **bilardi**
(`courierEarning` — qoida kuryer kartochkasida), va unga qancha **berilganini**
umuman bilmasdi. Ikki jimgina oqibat: moliyaviy hisobot kuryerlarni
xarajatlardan butunlay tashqarida qoldirardi (kodda buni ochiq yozgan izoh bor
edi — halol, lekin to'g'ri emas), va «o'tgan hafta uchun olganmidim?» degan
savolga faqat kimningdir xotirasi javob berardi.

- ⚠️ **O'z kolleksiyasi** (`courier_payment`), `staff_payment` ga kuryer
  qo'shilmaydi. Tejamkor ko'rinadigan yechim — bitta hujjatga ikkita id
  (`staffId` / `courierId`), va bittasi doim bo'sh. Bo'sh `ObjectID` brauzerga
  `"000…0"` bo'lib boradi, u esa **truthy** — ya'ni har bir ekran hujjatning
  qaysi yarmi haqiqiy ekanini eslab turishi kerak bo'lardi (§10 dagi tuzoq).
- ⚠️ **Ishlab topgan — hisoblanadi, to'langan — yoziladi, va ular bir-biridan
  chiqarilmaydi.** Birinchisi buyurtmalar va qoidadan kelib chiqadi (bajarilgan
  ish haqidagi fakt), ikkinchisi — hujjat (pul haqidagi fakt). Har qaysi
  tomonga chiqarish qoida tahrirlanganda o'zgaradigan raqam beradi — pul
  sanab berilganidan **oylar keyin**.
- ⚠️ **Hisobotda o'z qatori**: «Ishchilarga to'langan» ichiga qo'shilmaydi.
  Kuryer — yetkazish hajmi bilan o'sadigan xarajat, va «yetkazish o'zini
  oqlayaptimi?» degan savol uni oshxona oyligidan alohida ko'rishni talab
  qiladi.
- ⚠️ **«Qabul qilish» va «to'lash» — ikki tugma, ataylab.** Birinchisi kuryer
  **bizning** pulimizni qaytarishi, ikkinchisi biz **unga** ish haqi berishimiz.
  Yo'nalish almashtirgichli bitta forma — kuryerni o'zi qaytargan puli uchun
  mukofotlaydigan joy.
- Qoldiq **manfiy bo'lishi mumkin** (avans berilgan) va shundayligicha
  ko'rsatiladi: keyingi to'lov nega kichik ekanini aynan shu aytadi.

#### Kassa farqi va spisaniya: ko'rsatiladi, lekin jamiga tegmaydi

Ikkalasi ham hisobotda **`info`** qatori, va bu ataylab.

- ⚠️ **Kam chiqqan yashik — sanoq muammosi, xarajat emas.** Hech nima sotib
  olinmagan, hujjat yo'q; uni ayirish sanoq xatosini jimgina xarajatga, ortiqcha
  chiqqanini esa daromadga aylantirardi. Ega «kassir qoplasin» desa — bu kassa
  yozuvi bo'ladi, o'z qatori bilan. **Faqat yopilgan smenalar** sanaladi: ochiq
  smenada sanoq hali yo'q, va uning noli «yashik roppa-rosa to'g'ri» bo'lib
  o'qilardi — pul haqida xato bo'lishning eng tinchlantiruvchi usuli.
- ⚠️ **Davr bo'yicha jami, va gap shunda**: bitta kam kechqurun — shovqin, har
  juma kam chiqadigan yashik — boshqa narsa, va buni faqat qo'shib chiqqan
  raqam ayta oladi.
- ⚠️ **Spisaniya allaqachon "sotilgan taomlar tannarxi" ichida.** Uni yana
  chiqim deb sanash o'sha pomidorni ikki marta ayirish bo'lardi. Ko'rsatiladi,
  chunki «bu oy 4 200 000 so'm chiqindiga ketdi» ega qiladigan ishni
  o'zgartiradi, va buni hisobotda boshqa hech nima aytmaydi.

#### Perechisleniye: pul ishlab topilgan, lekin hali qo'lda emas

⚠️ **«Mijoz to'ladi» bilan «bizda bor» orasida haftalar bor, va tizim bu
oraliq borligini umuman bilmasdi.** Agregator (Yandex Eats, Uzum Tezkor)
mijozdan pulni **buyurtma paytida** oladi va restoranga **oyda bir marta**,
komissiyani ayirib o'tkazadi. Terminal (ekvayring) va onlayn relslar
(Click/Payme/Uzum/ATMOS) xuddi shunday ishlaydi, faqat soati qisqaroq.

Ya'ni Uzum Tezkor orqali 12 mln sotgan fevral — restoranda hali **hech nima
yo'q** bo'lgan fevral; martda kelgan 9,5 mln esa yo tog'ri, yo 200 ming kam —
va bu ikkisini ajratadigan ekran yo'q edi.

⚠️ **Kelgan pul — tushum EMAS.** Sotuv mijoz to'lagan kuni sanalgan; o'tkazma —
o'sha pulning **joyi o'zgarishi**, xuddi yashikdan seyfga olib borilgan naqd
kabi. Kelganini yana daromad deb sanash har agregator sotuvini ikki marta
yozardi — va ikki barobar bo'lgan tushum raqami **mutlaqo ishonarli**
ko'rinadi, xavfliligi ham shunda. Hisobotga **faqat komissiya** tushadi
(chiqim qatori), va buni test qo'riqlaydi.

- ⚠️ **Uchala raqam ham hisobotdagidek yoziladi**, `net` esa
  `gross − commission` dan **hisoblanmaydi**. Ular to'g'ri kelmaganda farq —
  sahifadagi eng qimmatli narsa: qaytarib olingan buyurtma, jarima, o'tgan
  oyning tuzatishi. Bittasini hisoblab chiqarish aynan shu nomuvofiqlikni
  o'chirardi — hujjat esa uni ko'rsatish uchun bor.
- ⚠️ **Komissiya kelgan sana bo'yicha xarajat**, hisobot davri bo'yicha emas:
  komissiyani o'tkazma oladi, va mayda kelgan mart hisoboti bank hisobiga
  tekkan oyga tegishli.
- ⚠️ **Chegara — eng uzoq `periodTo`, eng oxirgi hujjat emas.** Hisobotlar
  tartibsiz keladi (tuzatilgan mart apreldan keyin tushadi), va oxirgi
  hujjatning davrini olish allaqachon yopilgan sotuvlarni qayta ochardi.
- ⚠️ **Naqd, o'tkazma va qarzda hech kim pulimizni ushlab turmaydi**: naqd
  yashikda, o'tkazma to'g'ridan-to'g'ri keladi, qarz esa nomi ma'lum mijozda va
  uning o'z qatori bor. Ularga balans ochish — hech qachon yopilmaydigan qarz.
- ⚠️ **Faqat to'langan buyurtma qarz hisoblanadi**: `pending` onlayn to'lov —
  telefonini qidirayotgan mehmon, va uni sanash hech nima berilmagan odamga
  qarz yozardi.
- **Birinchi perechisleniyegacha** balans «qarz» deb atalmaydi: ekran «shu
  tizim orqali sotilgan» deydi. Aks holda birinchi kunda restoran butun
  tarixi hajmidagi qo'rqinchli raqamni «qarz» sifatida ko'rardi.

⚠️ **Agregator — to'lov usuli, yetkazish xizmati emas.** `delivery_provider`
«ovqatni kim olib boradi» ga javob beradi, bu esa «pulni kim ushlab turadi» ga.
Yandex ikkalasi ham bo'lishi mumkin, va bitta yozuvga qo'shish faqat kuryer
parkini yollagan restoranga yo'q qarzni ko'rsatardi. Shuning uchun ular
`payment_settings.aggregators` da.
- Yoqilgani kassada to'lov turi bo'lib chiqadi (`yandex_eats`, `uzum_tezkor`).
  ⚠️ Server ularni **shartsiz qabul qiladi**, ekran esa faqat yoqilganini
  ko'rsatadi: server rad etadigan usul — chekni yopa olmayotgan kassir.
- ⚠️ Sozlamadagi **foiz faqat formani oldindan to'ldiradi**, pul undan hech
  qachon hisoblanmaydi — sahifaning butun mazmuni ikkisi to'g'ri kelmaganini
  sezish.

##### Provayder bilan "integratsiya": eng ishlaydigani allaqachon bor

Click, Payme, Uzum va ATMOS **har to'lovni shu serverga tasdiqlaydi** (callback
— shusiz buyurtma to'langan deb belgilanmaydi). Ya'ni ular yig'gan pulni bizga
API aytishi shart emas: biz uni **buyurtma-buyurtma ko'rib turganmiz**. Shuning
uchun `/admin/payouts/expected` o'z yozuvimizdan davr bo'yicha summani beradi.

- ⚠️ **Bu — taklif, o'rnini bosuvchi emas.** Formani to'ldiradi, ega esa
  hisobotdagi raqamni **o'zi yozadi**, va ikkisining farqi butun mazmun. O'z
  raqamimizni `gross` ga yozib qo'yish har doim mukammal to'g'ri keladigan va
  hech qachon hech nimani ushlamaydigan hujjat ishlab chiqarardi. Testi bor.
- **Agregatorlar boshqacha**: Yandex Eats buyurtmasi bu tizimga faqat kimdir
  kassaga urgan bo'lsa tushadi, ya'ni raqam peshtaxtaning tartibi qadar aniq.
  Ekran buni yashirmay aytadi («o'z yozuvimiz, agregator hisoboti emas»).
- Haqiqiy API integratsiyasi (Yandex Eats partner API, Uzum Tezkor) shartnoma
  va kabinet kalitlarini talab qiladi — kalitsiz yozilgan klient sinab
  ko'rilmagan kod bo'ladi. Kelganda `payout` hujjati tayyor turadi: import
  qiladigan joyi bor, o'zgartirishi kerak bo'lgani yo'q.

#### Pul qayerda: uchta javob, hech qachon bitta raqam

⚠️ **Naqd, bankdagi va yo'ldagi pul — uchta boshqa "bor"lik**, va ular
qo'shilmaydi: naqdni bugun kechqurun, bankdagini shu hafta, agregatordagini
**boshqa odam hal qilganda** ishlatib bo'ladi. «Bizda 40 mln bor» — platformada
eng ko'p tilga olinadigan va eng noto'g'ri raqam bo'lardi, va aynan shuni ega
bankka yoki ijarachiga aytadi. Testi bor.

- **Naqd**: seyf (daftardan yig'iladi), ochiq kassa yashiklari (har biri o'z
  kutilgan summasi bilan), kuryerlar qo'lidagi, bozorchilardagi podotchet.
  ⚠️ **Faqat ochiq smena** — yopilgani sanalgan va topshirilgan, uni qo'shish
  kechagi tushumni bugun yana sanardi.
  ⚠️ **Kuryer naqdi butun tarix bo'yicha yig'iladi**, oxirgi 200 buyurtma
  bo'yicha emas (kuryer kartochkasi ro'yxat chizgani uchun tanlanma oladi):
  eski yetkazishlarni tashlab, har topshiruvni saqlaydigan oyna qarzni o'z-o'zidan
  kichraytirardi.
- **Bankda**: ⚠️ **tizim bank qoldig'ini bila olmaydi va bilgandek
  ko'rsatmasligi kerak.** U hisobga biz ko'rmaydigan pullar ham tushadi:
  eganing o'z puli, kredit, soliq qaytimi, o'z hisoblari orasidagi o'tkazma.
  Ko'rgan harakatlarimizdan chiqarilgan balans **ko'rmaganimiz qadar** noto'g'ri
  bo'lardi — va balansga o'xshab turardi. Shuning uchun shakl inventarizatsiya
  bilan bir xil: **sana bilan yozilgan sanoq** (`bank_balance`), har safar
  yangi qator (ustiga yozish hisobga qachon oxirgi marta qaralganini yo'q
  qilardi).
- **Yo'lda**: perechisleniye ekranining arifmetikasi qayta ishlatiladi — «Uzum
  bizga qancha qarz» degan savolga ikkita javob bittasidan ko'p.
- ⚠️ Har raqam **sanalganmi yoki hujjatlardan yig'ilganmi** — yozilib turadi:
  ular qarama-qarshi tomonga buziladi (sanalgani **eskiradi**, yig'ilgani
  hujjat tushib qolganda **noto'g'ri bo'ladi**), va qaysi nosozlikni qidirish
  kerakligi ma'lumotning yarmi.

#### Inkassatsiya: muddati tashqaridan belgilangan yagona pul hodisasi

⚠️ **Bu — qonun muddat qo'ygan yagona joy.** Kassa operatsiyalari qoidalari
(7-modda) bank bilan kelishilgan limitdan ortiq har so'mni hisobga o'tkazish
uchun topshirishni talab qiladi; faqat ish haqi qolishi mumkin, va u ham
**3 ish kuni** (8-modda, chekka joylarda 5). Ya'ni seyfda qancha borligi
ma'lum chegaradan keyin eganing shaxsiy ishi bo'lmay qoladi — sanani biladigan
tizim buni ayta oladigan tizimdir.

- **Limit** filialda saqlanadi (`branch.cashLimit`), chunki shartnoma
  bankniki va filialniki. ⚠️ **Bo'sh bo'lsa ogohlantirish yo'q**: nol —
  «bankning raqamini bizga hech kim aytmagan», va uni o'ylab topish
  jimlikdan yomonroq.
- ⚠️ **Inkassatsiya xarajat emas** — pul sarflanmadi, joyi o'zgardi. Uni chiqim
  deb sanash restoranning o'z tushumini o'zidan ayirish bo'lardi (kuryer
  topshirig'idagi bilan bir xato). Moliyaviy hisobot `collection` ni umuman
  o'qimaydi, testi bor.
- ⚠️ **Hisob-kitob hujjatga muzlatiladi**: davrdagi yopilgan smenalar (bu
  tizimdagi Z-hisoblar), ularning sanalgani, kutilgani, farqi va seyf qoldig'i.
  Keyin qayta hisoblansa, eski smena tuzatilgan har safar boshqacha javob
  berardi — va «chiqqan pul turgan pulga to'g'ri keldimi?» degan savol
  hamma imzo chekkanidan oylar keyin fikrini o'zgartirardi.
- ⚠️ **Faqat yopilgan smenalar**: ochiq yashik sanalmagan, va tekshirilmagan
  raqamni qo'shish topshiruvni **taxminga** solishtirish demakdir.
- ⚠️ **Ko'rsatiladigan va yoziladigan raqam bitta funksiyadan** chiqadi
  (`collectionDraft`): imzo chekishdan oldin ko'rilgan raqam bilan hujjatdagi
  raqam ikki xil kod yo'lidan kelsa, ajraydi.
- ⚠️ **Farq nol bo'lsa ham yoziladi**: faqat nomuvofiqlik bo'lganda yozadigan
  hujjat «tekshirildi, to'g'ri» bilan «hech kim tekshirmadi» ni ajratmasdi.
- **Bitta filial**: topshiruv bitta eshikda, bitta sumka bilan bo'ladi.
  «Kompaniya 14 mln topshirdi» — hech kimga berib bo'lmaydigan raqam. Bitta
  filialli restoran bu savolni ko'rmaydi (filial o'zi aniqlanadi).
- **Z-hisob haqida**: onlayn-KKM va virtual kassa foydalanuvchilari uchun
  **kunlik Z olish va kassa kitobi yuritish majburiyati bekor qilingan** —
  soliq organi ma'lumotni kassaning o'zidan oladi. Ya'ni bizdagi smena
  yopilishi qonuniy talab emas, **nazorat vositasi**: yashik sanalgan yagona
  daqiqa. Ekran uni shunday ataydi, «hisobot topshirish» demaydi.

#### Kassada berilgan ish haqi: ikki fakt, ikki nusxa emas

Kassir kuryerga oyligini yashikdan bergani **bitta hujjat** qoldirardi:
«maosh» deb yozilgan kassa chiqimi. Yashik to'g'ri, payroll esa kuryerga
**butun oy qarz** deb turaveradi — natijada oy oxirida yo ikkinchi marta
to'lanadi, yo hech kim eslay olmaydigan seshanba haqida bahs bo'ladi.

- **Endi ikkita hujjat yoziladi**: kassa chiqimi (yashik yengillashdi) **va**
  ish haqi hujjati (odamga to'landi). ⚠️ Bu ikki nusxa emas, **ikki fakt** —
  bittasi yolg'iz turganda ikkinchi savol javobsiz qoladi.
- ⚠️ **Ish haqi hujjati yozilmasa ham kassa yozuvi qoladi**: pul yashikdan
  allaqachon chiqib ketgan, va buni buxgalteriya nozikligi uchun rad etish
  aniq bilgan yagona faktimizni yo'qotish bo'lardi. Xato logga tushadi.
- ⚠️ **Ism yozilmaydi — ro'yxatdan tanlanadi.** Peshtaxtada yozilgan ism bir
  hafta ichida «Aziz», «aziz», «Азиз» va «Aziz kuryer» bo'ladi, va ularning
  birortasi payroll qarzdor bo'lgan odamga ulanmaydi.
- Ro'yxatda **ishchilar ham, kuryerlar ham birga**: kassir «bu staff yozuvimi
  yoki courier yozuvi» deb o'ylamaydi, u ro'parasida turgan **odamni** ko'radi.
  Ishdan bo'shaganlar ro'yxatda yo'q, lekin daftarda qoladi.

#### Kuryer ham oylik oladi: davri bor odamlar qatoriga qo'shildi

⚠️ **Kuryerlar — maosh oladigan yagona odamlar ediki, ularning to'lov davri
yo'q edi.** Pullari «har yetkazishdan» deb o'ylanardi, ya'ni uzluksiz — shuning
uchun ekranlar «jami qancha ishlab topgan» ni ayta olardi (faqat o'sadigan
raqam) va «hozir qancha qarzmiz» ni **umuman** ayta olmasdi.

- `courier.payPeriod` — ishchilarnikidek to'rt oyna (`StaffPayPeriod`), bo'sh
  bo'lsa oylik.
- `PayoutMonthly` rejimi — belgilangan oylik. ⚠️ **Rejim, per-order narx yonida
  turgan qo'shimcha maydon emas**: ikkita raqam «qaysi birini qarzmiz» ni
  ekranni o'qiyotgan odamga qoldirardi, va ikkala javob ham to'g'ri ko'rinadi.
  ⚠️ Oylikdagi kuryer **yetkazishdan alohida pul olmaydi** — ikkalasini berish
  ikki marta to'lash, va qo'sh raqam gavjum oydan farq qilmasdi.
- `/admin/payroll` da kuryerlar **ishchilar bilan bir jadvalda**: oy oxirida
  hisob-kitob qilayotgan ega ikkita ekranda «ishlab topgan» va «to'langan» ning
  ikki xil ma'nosini o'qimasligi kerak.
- ⚠️ Qator `courierId` **yoki** `staffId` olib yuradi, «tur» bayrog'i bilan
  bitta id emas: to'lash tugmasi ikki xil endpointga boradi, va ikkala id ham
  haqiqiy `ObjectID` — bayrog'i adashgan qator oshpazga kuryerning pulini
  jimgina yozib qo'yardi. Frontendda `hasId()` (§10 tuzoq).
- ⚠️ **Ishlab topgan hisoblanadi, to'langan yoziladi** — qoida oldingidek:
  dekabrda tahrirlangan qoida avgustda sanab berilgan pulni qayta yozmasligi
  kerak. Testi `AdminPayCourier` ga toraytirildi.

#### Podotchet: kim restoranning pulini ushlab turibdi

Ikki million so'm ertalab soat oltida seyfdan chiqadi, bir million yetti yuz ellik
mingi ovqat bo'lib qaytadi, ikki yuz ellik mingi pul bo'lib. Tizim bularning
faqat **ovqat** qismini bilardi; qolgani suhbat edi.

⚠️ **Podotchet — chiqim emas, va butun dizayn shu farqqa suyanadi.** Pul
qo'lga berilganda sarflanmaydi; u mahsulot sotib olganda sarflanadi, va u —
moliyaviy hisobot allaqachon sanaydigan `purchase` hujjati. Qo'lga berishni ham
chiqim deb yozish bir pulni **ikki marta** sanardi (naqd chiqdi va ovqat keldi),
va oy bor holidan yomonroq ko'rinardi. Bu daftar boshqa savolga javob beradi:
**hozir bizning pulimizni kim ushlab turibdi.**

- ⚠️ **O'z kolleksiyasi, `cash_entry` emas.** Kassa yozuvi **ochiq smenaga**
  tegishli, bu yerdagi muhim smena esa mavjud emas: pul soat oltida, seyfdan,
  hech kim yashik ochmasidan oldin beriladi. Pul kassadan chiqqan bo'lsa,
  kassir uni har doimgidek yozadi — u yashik haqidagi fakt, bu esa **odam**
  haqidagi.
- ⚠️ **Balans hujjatlardan ayiriladi, saqlanmaydi**: `berilgan − qaytarilgan −
  sarflangan`. Saqlangan jami — daftardagi javobning ikkinchi nusxasi, va u
  birinchi o'chirilgan kirimda ajraydi — jimgina, pul haqidagi raqamda.
- ⚠️ **Faqat *to'langan* kirimlar sarf hisoblanadi.** Kreditga olingan nakladnoy
  hech kimning qo'lidan o'tmagan — u yetkazib beruvchiga qarz; uni bozorchining
  balansidan ayirish uni hali ushlab turgan pulini sarflagandek ko'rsatardi, va
  kamomad pul sanalganda topilardi.
- ⚠️ **Shu sababdan bozorlik `paid: true` bilan yoziladi** (bu ilova chiqqanda
  o'tkazib yuborilgan edi). `paid` — yetkazib beruvchi qarzi hisobotining
  manbasi, va to'lanmagan bozorlikning qarzdor bo'ladigan yetkazib beruvchisi
  **yo'q**: har bir bozorlik o'sha hisobotda abadiy «hech kimga qarz» bo'lib
  turardi, ega esa u raqamga qarab ish qiladi. Panel formasi so'raydi, chunki
  nakladnoy haqiqatan to'lanmagan bo'lishi mumkin; bozorda naqd pul bilan
  turgan odam esa yo'q.
- ⚠️ **Nol ostiga tushishi mumkin, va uni yashirmaslik kerak.** Puli tugab,
  oxirgi yashikni o'z cho'ntagidan olgan bozorchiga restoran qarzdor, va nolda
  to'xtaydigan daftar aynan kimdir kutayotgan qarz haqida jim qolardi.
- ⚠️ **Pul olmagan odam ro'yxatda yo'q.** Paneldan nakladnoy kiritgan menejer
  podotchet ushlab turmaydi, va uni manfiy balans bilan ko'rsatish — bu ekranda
  ayblovdek o'qiladigan, odam haqidagi noto'g'ri gap.
- **Ikki tur, ishorali summa emas** (`out` / `back`): pul ustunidagi manfiy son
  jadval ishlatgan har bir odam tomonidan **tuzatish** deb o'qiladi, bu daftarda
  esa haqiqiy tuzatishlar ham bor.
- **Yozuv o'chiriladi, teskari yozuv bilan qoplanmaydi.** Teskari yozuv toza
  buxgalteriya javobi va bu daftar uchun noto'g'ri: ikkalasi ham haqiqiy pul
  berishga o'xshaydi, va bir oydan keyin o'qiyotgan odam tuzatishni seyfga
  ikkinchi borishdan ajrata olmaydi. Iz — amallar jurnalida.
- Bozorchi o'z balansini **ilovada**, ro'yxatdan oldin ko'radi: bu raqam safar
  bo'ladimi degan savolga javob beradi, va uni bilmagan odam ofisga
  qo'ng'iroq qilish o'rniga taxmin qiladi.
- Panelda `/admin/purchases` ichida — u to'laydigan kirimlarning yonida: «nima
  keldi» va «pulimizni kim ushlab turibdi» bitta nafasda so'raladigan ikki savol,
  va bir qavat naridagi daftarni hech kim solishtirmaydi.

### Kassa buyurtmani qabul qildimi (`handlers/posorder.go`)
- ⚠️ **Yuborish — ko'prikning yarmi.** `SendOrder` POS buyurtmani **qayd
  qilgan** paytda qaytadi, va to'rtta provayderning ikkitasida bu oshxona uni
  ko'rgani **emas**: **Poster** uni `status: 0` bilan "incoming order" qilib
  yozadi va kassadagi odam "Qabul qilish" bosishini kutadi (`autoAccept`
  parametri **yo'q** — `createIncomingOrder` ning butun ro'yxati: `spot_id,
  client_id, first_name, last_name, phone, email, sex, birthday, address,
  comment, products, payment, promotion`); **iiko** asinxron yaratadi.
- ⚠️ **`OrderStatus` uchala adapterda yozilgan edi va uni hech kim
  chaqirmasdi.** Ya'ni panel "kassaga yuborildi" deb yozib, keyin fikr
  bildirishni **butunlay** to'xtatardi: buyurtma butun smena davomida kassa
  ekranida qabul qilinmay yotishi mumkin edi va bu fakt faqat **o'sha xonada**
  mavjud bo'lardi.
- **Kassaning javobi alohida maydon** (`order.pos.till`), `pos.status` ichiga
  qo'shilmaydi: ular boshqa faktlar va **boshqa soatda** o'zgaradi — topshirish
  bir soniyada tugaydi, qabul qilish esa peshtaxtadagi odam qaraganda. Bitta
  maydon ikkalasini ifodalaganda yo oshxona haqida yolg'on gapiriladi, yo
  muvaffaqiyatli yuborishning izi yo'qoladi.
- **Sekinlashadigan so'rov** (`tillCheckDue`): birinchi 10 daqiqa har daqiqada,
  keyin har 5 daqiqada, 6 soatdan keyin umuman so'ralmaydi. Hech kim
  qaramaydigan kassa soatiga 60 so'rovga tushmasligi kerak.
- ⚠️ **Xatoda oldingi hukm saqlanadi**: bir daqiqa yetib bo'lmagan kassa hech
  nimani "qabul qilmagan" holga qaytarmaydi, holatni bo'shatish esa har tarmoq
  uzilishini "kassada kutilmoqda" ogohlantirishiga aylantirardi — ya'ni
  o'zimizning aloqamiz haqidagi muammo birovning oshxonasi haqidagi
  ayblovga.
- ⚠️ **Filtrda `$nin`, `$in` emas** (`pendingTillFilter`): maydon bu
  xususiyatdan **oldin** yuborilgan har bir buyurtmada umuman yo'q, va Mongo'da
  yo'q maydon `$nin` ga **mos keladi**. Ochiq ro'yxat holatlariga `$in` yozish
  bir xil ma'noga o'xshaydi va aynan hech qachon tekshirilmagan buyurtmalarni
  tashlab ketardi. Testda muhrlangan.
- ⚠️ **Kassaga umuman tushmagan buyurtma — alohida va og'irroq nosozlik**
  (`/admin/alerts` → `pos.failed`, `failedPOSFilter`). Odatiy sabab **bitta
  bog'lanmagan taom**: `pos.CheckMapped` butun buyurtmani ataylab rad etadi
  (chala chek chekdan yomonroq), lekin yuborish tasdiqlashda **fonda** ketadi —
  ya'ni operator qatorning yashil bo'lganini ko'radi va o'tib ketadi, mehmon
  kutadi, oshxonada esa chek yo'q **va yo'qligidan xabari yo'q**. Buyurtmalar
  ro'yxatida hech nima yozilmasdi: chek ichini ochmagan odam buni umuman
  bilmasdi.
  - **Nishon buyurtmalar ro'yxatida** (`⚠ Kassaga tushmadi`, qizil) — odam
    allaqachon qarab turgan yagona joy shu edi.
  - `pos.till.state == "waiting"` uchun ham nishon bor, lekin **sariq**: u
    yerda chek hech bo'lmaganda kassa ekranida turibdi.
  - ⚠️ **Vaqt chegarasi yo'q** (kassa javobidagidan farqli): bir soat oldin
    yiqilgan va hali ochiq buyurtma **ko'proq** ko'rsatishga arziydi, kamroq
    emas. Yopilganlari holat filtridan o'zi tushib qoladi.
  - ⚠️ **Bu ham jimgina** — banner qizil, lekin ovozsiz: sabab bitta
    bog'lanmagan taom bo'lgani uchun **har bir** buyurtma yiqiladi, va sozlama
    xatosi davomida har buyurtmada chaladigan signal — odam o'chirib qo'yadigan
    va keyin yoqmaydigan signal.
- ⚠️ **Oldini olish: bog'lanmagan taomlar sanog'i** (`pos.unmapped`,
  `unmappedDishes`). Yuqoridagilarning hammasi **bo'lib o'tgan** nosozlikni
  aytadi; bu esa uni keltirib chiqaradigan **shartni**, buyurtma kelishidan
  oldin. ⚠️ Bo'shliq odatdagi ishdan tug'iladi: restoran POS'ni ulaydi, hamma
  taomni bog'laydi, hammasi ishlaydi — keyingi oyda kimdir menyuga yangi taom
  qo'shadi (bu panel aynan shuning uchun bor) va uni hech kim bog'lamaydi. Taom
  sotilaveradi, to birinchi buyurtmagacha, va o'sha buyurtma **butunlay**
  yiqiladi. Hech kim xato qilmagan va hech nima ogohlantirmagan.
  - ⚠️ **Combo sanalmaydi**: u hech qachon o'zi bo'lib yuborilmaydi —
    `posItems` uni a'zolariga yoyadi, chunki kassada bu sayt o'ylab topgan
    to'plam uchun mahsulot yo'q. Uni sanash bog'lash ekranida **tuzatib
    bo'lmaydigan** muammoni ko'rsatardi, tuzatib bo'lmaydigan ogohlantirishni
    esa ega o'tkazib yuborishni o'rganadi.
  - **Sotuvda bo'lmagan taom ham sanalmaydi**: uni buyurtma qilib bo'lmaydi,
    ya'ni u hech nimani buza olmaydi. Menyuga qaytsa o'shanda sanaladi.
  - **Keshlanadi (2 daqiqa)**, chunki `/admin/alerts` har 15 soniyada **har bir
    ochiq tabda** ishlaydi va ataylab kichik. Bog'lash saqlanganda kesh darhol
    tozalanadi (`forgetUnmapped`) — aks holda ega tuzatgandan keyin
    ogohlantirish yana ikki daqiqa turardi va bu "tuzatish ishlamadi" bo'lib
    o'qilardi.
  - **Buyurtmalar allaqachon yiqilayotgan bo'lsa ko'rsatilmaydi**: qizil banner
    o'sha gapni kuchliroq va buyurtma nomi bilan aytadi, ikki banner esa bitta
    muammoni ikkita qilib ko'rsatadi.
- ⚠️ **Ogohlantirish bor, ovoz yo'q** (`/admin/alerts` → `pos.unaccepted`,
  5 daqiqadan keyin). `AlertBell` dagi har bir ovozning shu paneldа **aynan
  bitta** to'xtatuvchi tugmasi bor; buni to'xtatadigan amal esa **kassada**.
  Bu yerda jimlatib bo'lmaydigan qo'ng'iroq — odam e'tibor bermaslikni
  o'rganadigan qo'ng'iroq, va u o'rganilgan odat qolgan ikkitasiga ham
  ko'chadi. Banner DOM'da oxirgi — teskari ustunda barmoqdan eng uzoq: bu
  xabar, buyruq emas.
- ⚠️ **Ogohlantirish `TillWaiting` ni aniq talab qiladi**, "hal bo'lmagan
  hamma narsani" emas: hech qachon so'rab ulgurmagan buyurtma ham, javob bera
  olmaydigan kassa ham (r_keeper → `TillUnsupported`) bunga kirmasligi kerak —
  u peshtaxtadagi **odam** haqida aniq bir gap aytadi, va uni **o'z
  sukutimizdan** aytish egani ogohlantirishlarga ishonmaslikka o'rgatadi.
- Chekdagi rang endi **kassaning hukmiga** qaraydi: ilgari "yuborildi" o'zi
  yashil qilardi, ya'ni hech kim tegmagan buyurtma haqida "oshxonada" degan
  da'vo. `waiting` — sariq, `cancelled` — qizil.
- **Kassasi yo'q restoran uchun ham shu sahifa**: qidiruv, "faqat sotuvda emas"
  filtri va bitta bosishli stop/qaytarish. Menyu sahifasidagi tugma joyida
  qoldi — bu sahifa boshqa savolga javob beradi ("hozir nima yopiq?").

### Rasmlar: o'lchash va kesh (`?w=`)
- ⚠️ **O'lchangan muammo, xohish emas.** Restoran bosh sahifasi **2.36 MB**
  edi, shundan **1.83 MB — 16 ta rasm**: namuna menyusi 900×675, sifat 95
  (~210 KB har biri), ko'rsatiladigan kartochka esa ~350 px. O'zbekistondagi
  mobil internetda bu ochiladigan sayt va yopib ketiladigan sayt farqi.
- ⚠️ **Yuklangan har bir rasm WebP ga o'giriladi** (`images.Fit`), aslini
  diskka yozmasdan: fayl nomining kengaytmasi **chiqqan baytlarga qarab**
  qo'yiladi. O'lchandi (namuna menyusidagi haqiqiy surat): JPEG 131 KB → 93 KB,
  o'sha suratning PNG eksporti **1.5 MB → 93 KB**. Jonli sinovda eganing 2 MB
  lik PNG banneri **180 KB** bo'ldi.
  - ⚠️ **Shaffoflik bor rasm — lossless, qolgani — lossy.** Bular ikki xil
    surat: tort fotosurati aynan lossy uchun yaratilgan, logotip esa bo'sh
    maydondagi o'tkir chekka — lossy unga halo va alfa chetiga rang qo'shadi,
    va bu ega eng diqqat bilan qaraydigan rasm. Signal — **piksellar**, fayl
    kengaytmasi emas: PNG qilib eksport qilingan fotosuratda alfa yo'q, ya'ni
    u to'g'ri yo'lga tushadi.
  - ⚠️ **Hech qachon kattalashtirmaydi**: tekis grafika yoki shovqinli naqsh
    WebP da **kattaroq** chiqishi mumkin (buni test topdi, restoran emas).
    WebP yutmasa — asl format qoladi.
  - ⚠️ **Animatsiya tegilmaydi** (GIF va animatsiyali WebP): `image.Decode`
    birinchi kadrni qaytaradi va qolganini indamay tashlaydi, ya'ni harakatli
    logotip jimgina qotib qolardi. Shuning uchun tekshiruv **konteynerda**
    (RIFF `ANIM` chunki), dekoddan keyin emas.
  - ⚠️ **`nodynamic` build tegi Dockerfile'da shart**: kutubxona aks holda
    tizimdagi `libwebp` ni dlopen qiladi, alpine'da esa u yo'q — binar
    quriladi, ishga tushadi va **jimgina JPEG saqlaydi**. Server yuklanishda
    qaysi enkoder borligini **log qiladi** (mintaqa qatorining yonida).
  - ⚠️ **Chek printeri ham WebP ni dekod qiladi** endi (`printlogo.go`):
    busiz har bir restoranning cheki logotipsiz chiqib ketardi va panelda
    hammasi joyida ko'rinardi.
- **O'lcham URL'da so'raladi** (`?w=300|600|1200`), yuklashda ikkinchi fayl nomi
  yasalmaydi. Sabab: yuklash vaqtidagi variant **faqat keyin** yuklangan
  rasmlarga yordam berardi, diskdagi hamma narsa esa to'liq hajmda qolardi —
  yoki sayt mavjud bo'lmagan `-600` faylini so'rab, kartochkani buzardi.
  So'rov bo'yicha esa eski va yangi rasm birinchi so'rovdan bir xil ishlaydi.
- Hosila fayllar **diskda keshlanadi** (`uploads/.thumb/<w>/<nom>.<kengaytma>`),
  ya'ni har rasm har o'lcham uchun bir marta o'lchanadi. ⚠️ **Kesh fayli o'z
  kengaytmasini olib yuradi**: hosila WebP, asli esa `.png` bo'lishi mumkin, va
  `http.ServeContent` turni **nomdan** o'qiydi — nom asliniki bo'lsa, WebP
  `image/png` bo'lib beriladi. Shu bilan birga eski (WebP'gacha yozilgan) kesh
  boshqa yo'lda qoladi va o'zi e'tiborsiz qoladi.
  ⚠️ **Eski rasmlar ham shundan yutadi**: diskdagi JPEG/PNG ning `?w=` hosilasi
  endi WebP bo'lib chiqadi — hech bir hujjatdagi havolaga tegmasdan. Yozish **atomik** (tmp + rename):
  sovuq rasmga ikki mehmon bir vaqtda kelsa, yarim yozilgan fayl butun kesh
  umri davomida buzuq rasm bo'lib berilardi.
- ⚠️ **Kengliklar allowlist, diapazon emas** (`thumbWidths`): `?w=` ochiq
  internetdan keladi, va istalgan son server CPU'si va mijoz diskini piksel
  bo'yicha sarflash imkonini berardi (`w=101`, `w=102`…). Noma'lum kenglik
  **xato bermaydi**, aslini beradi — eskirgan sahifa rasmni ko'rsatishi kerak.
- **Yo'l ikki qavat qo'riqlangan**: `path.Clean` va `os.Root` (symlink orqali
  chiqishni ham rad etadi). `.thumb` so'rov yo'li sifatida rad etiladi — kesh
  ommaviy daraxtning qismi emas.
- **`Cache-Control: immutable, 1 yil`** — yuklangan fayl nomi tasodifiy, ya'ni
  almashtirilgan rasm **yangi URL**. Istisno: `seed/` fayllari nomi qat'iy,
  shuning uchun ularga 30 kun va `immutable` yo'q (kelasi versiya boshqa baytni
  o'sha nom bilan yuborishi mumkin). Ilgari `http.FileServer` **hech qanday**
  kesh sarlavhasi qo'ymasdi: qaytib kelgan mehmon 16 rasmni qaytadan
  so'rardi, har biri Fransiyaga borib kelish.
- **Asl fayl ham kichraytiriladi** (yuklashda, 1600 px, `handlers/upload.go`):
  telefon kamerasi 3000×4000 / 4 MB beradi, va u har kecha zaxira nusxaga
  tushadi. Dekod qilinmagan fayl (WebP, animatsiyali GIF) **aynan kelgan
  holida** saqlanadi — yaxshilay olmaganimiz uchun yuklashni rad etish oshxonada
  turgan ega uchun noto'g'ri savdo.
- **Yangi bog'liqlik yo'q**: kichraytirish — maydon o'rtachasi (box filter),
  ~40 qator standart kutubxona (`internal/images`). Kichraytirishda aynan shu
  to'g'ri filtr: har manba piksel bir marta qatnashadi. Alfa bo'yicha
  o'rtachalash ataylab — busiz shaffof piksellar logotip chetiga qora halo
  qo'yadi.
- **Qaysi sahifa nimani so'raydi**: kartochkalar va ro'yxatlar 600, savat va
  logotip 300, muqova va taom sahifasi 1200 (`imageUrl(path, width)`).
  Absolyut URL'ga parametr **qo'shilmaydi** — birovning CDN havolasi bizniki
  emas.
- Natija (jonli o'lchov): 4.93 MB namuna to'plami 600 px da **2.15 MB** (−56%).

### Dizayn tizimi (frontend)
- Ranglar `tailwind.config.ts` da: `brand` (aksent, har restoran uchun
  o'zgartiriladi), `ink` (to'q iliq ko'mir), `cream` (fon). Sahifalarda
  to'g'ridan-to'g'ri `neutral-*`/`rose-*` ishlatilmaydi — faqat shu tokenlar.
- **Chegaralar `border-line` / `border-line-strong`** (`--line`,
  `--line-strong`). Ular `ink/[0.07]` kabi alpha bilan emas, tayyor rang
  sifatida berilgan: qorong'i sirtda 7% chiziq ko'rinmaydi, shuning uchun
  ikki temada alpha ham har xil. `border-ink/*` ishlatilmaydi.
- **Sahifa foni doim `bg-cream`** (`--bg`), kartochkalar `bg-surface`.
  `bg-ink/5` ni sahifa foni sifatida ishlatmang — dark rejimda u oq qatlam
  bo'lib `--surface` bilan ustma-ust tushadi va kartochkalar yo'qoladi
  (aynan shu xato bo'lgan edi).
- ⚠️ **Har bir modal va bannerda ko'rinadigan yopish tugmasi bo'ladi.**
  `components/admin/Modal.tsx` yozilgan kunidan beri fon bosilganda yopilardi,
  va buni ekranda **hech nima aytmasdi** — ya'ni tasodifan ochilgan yoki o'qib
  bo'lingan oyna bosiladigan tugmasiz qolar, tagidagi sahifa esa yopiq turardi.
  Ko'rinmaydigan yo'l — yo'l emas, u faqat muallif biladigan narsa. Endi
  o'ng yuqorida × (**sticky**, chunki forma uzun bo'lsa absolyut tugma ekrandan
  chiqib ketadi — aynan qochgingiz keladigan oynada) va **Escape**.
- ⚠️ **Ogohlantirish banneri ham yopiladi, lekin sanoq bilan** (`AlertBell`):
  yopilgan payt ekrandagi son eslab qolinadi va banner **undan bittasi
  ko'p** bo'lganda qaytadi. Oddiy `hidden` bayrog'i ertalab yopilgan banner
  tufayli **kechqurungi yangi nosozlikni** jimgina yutib yuborardi.
  Yagona istisno — signal (`waiting`): u «buyurtma hali kutyapti» degan
  **gap**, va uni yopish keyingi so'rov rad etadigan yolg'on bo'lardi;
  o'rniga besh daqiqalik «jim tur» bor.
- Takrorlanuvchi klasslar `globals.css` `@layer components` da:
  `.btn / .btn-primary / .btn-ghost / .btn-dark`, `.card`, `.badge*`, `.chip`,
  `.eyebrow`, `.section-title`, `.input`, `.container-page`.
- Shriftlar: **Inter** (matn) + **Playfair Display** (sarlavha, narx) —
  `next/font/google`, `font-sans` / `font-display`.

### Tavsiya etilgan rasm o'lchamlari panelda yozilgan
Ega logotipni, muqovani va bannerni **panelni ochishdan oldin** yasaydi —
odatda Canva'da, oxirgi marta nima yasagan bo'lsa o'sha shaklda. Yuklagandan
keyin o'qiladigan o'lcham — kech o'qilgan o'lcham: rasm allaqachon tayyor va
javob «qaytadan yasang» bo'ladi. Shuning uchun raqamlar **fayl tanlanadigan
ekranda**, tanlash tugmasining yonida turadi (`ImageUpload` ning `hint` i).

⚠️ **Raqamlar o'ylab topilmagan — chizadigan koddan olingan**, aks holda ular
birinchi tahrirda yolg'onga aylanadi:
- **Logotip 512×512 (kvadrat)** — `BrandMark` uni `rounded-xl object-cover`
  bilan kvadrat qilib chizadi va `?w=300` so'raydi. ⚠️ **U bir vaqtning o'zida
  favicon** (`app/layout.tsx` → `icons`), ya'ni juda kichkina holatda ham
  tanilishi kerak — bu tavsiyaning yarmi shundan.
- **Muqova 1200×630** — bu `og:image` ning standart o'lchami, va muqova aynan
  shu (`openGraph.images` + `summary_large_image`). Ikkinchi ishlatilishi —
  «Biz haqimizda» sahifasining foni: u **qoraytiriladi va ustiga matn
  yoziladi**, shuning uchun ichida yozuvi bor rasm tavsiya etilmaydi.
- **Sayt banneri 1200×450 (16:6)** — karusel telefonda `aspect-[16/6]`,
  keng ekranda `sm:aspect-[16/5]`. ⚠️ **Ikki xil kesish**: 16:6 yuklansa keng
  ekranda yuqori-past kesiladi, 16:5 yuklansa telefonda yon tomonlari. Shuning
  uchun o'lcham 16:6 va matnni markazga qo'yish alohida aytilgan — bitta
  «to'g'ri» o'lcham yo'q.
- **Kassa banneri 1200×1800 (2:3)** — allaqachon yozilgan edi; sayt bannerining
  o'zi esa yo'q edi, ya'ni ega birinchi uchraydigan yarmi.

⚠️ **Format va 10 MB chegarasi — bir marta, ikkala yuklovchining tagida.**
Logotip va muqova yonma-yon turadi; har biriga to'rtdan uch qismi bir xil izoh
qo'yish — o'quvchini ikkalasini ham o'tkazib yuborishga o'rgatish. 10 MB muhim:
Canva'ning chop etish sifatidagi eksporti undan oshadi, va rad javobi
**kutishdan keyin** keladi.

### Sayt dizayni (admin tomonidan o'zgartiriladi)
- `restaurant.theme` — asosiy rang, burchak yumaloqligi (px), tugma shakli
  (pill / kartochkalarga mos), shrift juftligi (`classic`/`modern`/`soft`),
  fon ohangi, kartochka soyasi, tugma to'ldirilishi va umumiy o'lcham (root
  font-size). Admin panelda 5 ta **tayyor mavzu** ham bor — ular shu
  maydonlarning tayyor to'plami, xolos.
- Soyalar (`--shadow-card`), tugma ko'rinishi (`--btn-bg/fg/border`) va
  umumiy o'lcham (`--root-size`) ham CSS o'zgaruvchilari — Tailwind
  `shadow-card` shulardan o'qiydi.
- `lib/theme-css.ts` bu sozlamalarni CSS o'zgaruvchilariga aylantiradi va
  `app/layout.tsx` uni **`<head>` ichida inline** beradi — birinchi bo'yashdayoq
  qo'llanadi, ya'ni standart rang "chaqnab" o'tmaydi. Qorong'i tema uchun
  aksent avtomatik ochiqroq qilinadi (aks holda qora fonda yo'qoladi).
- **Burchaklar butun ilova bo'ylab bitta joydan boshqariladi**: `tailwind.config.ts`
  da `rounded-xl/2xl/3xl` → `--radius-md/lg/xl`. Ya'ni komponentlarga tegmasdan
  butun sayt shakli o'zgaradi. `rounded-full` chin pill bo'lib qoladi;
  tugmalar `--radius-btn` ni ishlatadi.
- Shriftlar `next/font` bilan build vaqtida yuklanadi (Inter, Playfair, Nunito),
  tanlov faqat o'zgaruvchini almashtiradi.
- Admin: `/admin/settings` → "Sayt dizayni" (`components/admin/DesignEditor.tsx`),
  jonli ko'rinish bilan (light va dark).

### Sayt matnlari
- `restaurant.content` — bosh sahifadagi shior, "Biz haqimizda" sarlavhasi va
  matni, footer matni. Har biri `{ uz, ru, en }`; bo'sh RU/EN o'zbekchasiga
  tushadi (`lib/i18n/site-content.ts` → `localized()`), bo'sh uz esa standart
  matnga.
- Telefon, manzil, ijtimoiy tarmoqlar, ish vaqti — allaqachon sozlamalarda
  (footer va "Biz haqimizda" o'sha ma'lumotdan o'qiydi).

### Tema (dark / light)
- ⚠️ **Hidratsiya tuzog'i**: `(site)/layout.tsx` dagi `<Suspense>` Header'ni
  **kech gidratlaydi**, root layout'dagi `ThemeProvider`/`CartProvider`
  effektlari esa undan **oldin** ishlab bo'ladi. Shuning uchun Suspense
  ichidagi komponentda **ota-provayderning `mounted` bayrog'iga tayanib
  bo'lmaydi** — u allaqachon rost bo'ladi.
  Ikki to'g'ri yo'l: (a) markup holatga umuman bog'liq bo'lmasin va CSS hal
  qilsin — `ThemeToggle` ikkala ikonkani ham chizadi, `dark:hidden` /
  `dark:block` tanlaydi; (b) bayroq **shu komponentning o'ziniki** bo'lsin —
  Header'dagi savat nishonchasi shunday. Yorliqlar ham holatni emas, amalni
  nomlasin ("Mavzuni almashtirish").
- Semantik ranglar — `globals.css` dagi CSS o'zgaruvchilari (`--bg`,
  `--surface`, `--fg*`, `--brand*`); Tailwind `darkMode: "class"`.
  Doim qorong'i sirtlar uchun alohida `charcoal` tokeni (hero, footer).
- `<html class="dark">` — `lib/theme.tsx` (localStorage), birinchi bo'yashdan
  oldin `app/layout.tsx` dagi inline skript qo'yadi.
- ⚠️ **Standart — light, va qurilma sozlamasi (`prefers-color-scheme`) ataylab
  o'qilmaydi.** Restoran sayti — vitrina: ega aksentni tanlaydi, rasmlarni
  tasdiqlaydi va natijani odamlarga ko'rsatadi; telefoni dark rejimda bo'lgan
  mehmon esa tasdiqlangan restorandan boshqasini ko'rardi. Menyu rasmlari ham
  oq fonda olinadi va tahrirlanadi. Almashtirgich va saqlangan tanlov
  o'zgarmadi — faqat **hech qachon tanlamagan** odam uchun javob o'zgardi.
  Xuddi shu qoida `keel-site` da ham (landing + konsol).
- ⚠️ Inline skript va `lib/theme.tsx` **bir xil qoidani** aytishi shart: biri
  bo'yashdan oldin, ikkinchisi keyin ishlaydi, ya'ni har qanday nomuvofiqlik
  ko'zga ko'rinadigan "chaqnash" bo'lib chiqadi.
- Yangi sahifa yozganda: `bg-white` emas `bg-surface`, `text-neutral-*` emas
  `text-ink/ink-soft/ink-muted` ishlating.

### Ko'p tillilik (UZ / RU / EN)
- Lug'atlar: `src/lib/i18n/dictionaries.ts` (`Dict` tipi `uz` dan olinadi).
- Til `lang` cookie'da. Server komponentda `getTranslations()`
  (`lib/i18n/server.ts`), client komponentda `useI18n()`
  (`lib/i18n/client.tsx`). Almashtirgich: `components/site/LangSwitch.tsx`.
- Narx/sana/hafta kuni: `formatPrice(x, currency, lang)`,
  `weekdayName(day, lang)`, `t.locale`.
- **Vaqt doim 24 soatlik** — `formatTime` / `formatDate` / `formatDateTime`
  (`lib/format.ts`), qo'lda formatlangan (`HH:MM`, `DD.MM.YYYY`).
  `toLocaleTimeString` **ishlatilmaydi**: u qurilma tiliga ergashadi va inglizcha
  telefonda "1:16 PM" chiqaradi — bir ekranda "11:00 — 23:00", boshqasida
  "11:00 AM — 11:00 PM" turgan smena jadvali jadval emas. Qo'lda formatlash
  bir vaqtning o'zida SSR/brauzer hidratsiya farqini ham yo'q qiladi
  (`formatPrice` `Intl` dan qochgani bilan bir sabab).
- **Menyu kontenti ham ko'p tilli**: `category` da `nameRu`/`nameEn`,
  `menu_item` da `nameRu`/`nameEn`/`descriptionRu`/`descriptionEn`.
  O'zbekcha — asosiy (base); tarjima bo'sh bo'lsa avtomatik o'zbekchasi
  ko'rsatiladi (`lib/i18n/content.ts` → `contentName`/`contentDescription`).
  Admin panelda har bir taom/kategoriya formasida "Tarjimalar" bloki bor.
  Buyurtmaga esa **doim base (uz) nomi** yoziladi — admin uchun yagona til.
- **Rol nomi ham menyu kontenti kabi ko'p tilli** (`staff_role`:
  `nameRu`/`nameEn`, o'zbekchasi — base). Ishchi qo'shayotganda rol tanlash
  ro'yxati, `/admin/roles` jadvali va kassaning burchagidagi lavozim
  `contentName`/`contentText` orqali o'qiladi. Sabab: rol nomini **restoran
  yozadi**, ya'ni u lug'atdan kelolmaydi — ruscha panelda o'n bitta o'zbekcha
  so'z turardi va uni tuzatishning yagona yo'li butun ro'yxatni qayta yozish edi.
  ⚠️ **Faqat base majburiy**: bitta tilda ishlaydigan restoran rol qo'shish
  uchun yana ikkita so'z o'ylab topmasligi kerak; bo'sh tarjima base'ga qaytadi.
  ⚠️ Biz yuboradigan o'n bitta rol **migratsiyada tarjima qilinadi**
  (`translateSeededRoles`): seed faqat bo'sh kolleksiyaga tushadi, ya'ni mavjud
  installlarda nom allaqachon yozilgan. Moslash `name` + `seeded` bo'yicha va
  faqat tarjima bo'sh bo'lsa — nomini o'zgartirgan yoki o'z ruschasini yozgan
  restoran o'zinikini saqlab qoladi (shu bilan takroriy ishga tushirish ham
  xavfsiz). Yangi seed roli tarjimasiz qo'shilsa test yiqiladi
  (`TestSeedRolesAreNamedInThreeLanguages`) — bu xato aks holda **jimgina**
  bo'lardi: bo'sh tarjima base'ga qaytadi va ekranda faqat til aralashib ketadi.
- **Admin panel va kuryer ilovasi ham uch tilli**: alohida lug'at
  `src/lib/i18n/admin.ts` (`useAdminT()`), mijoz lug'atidan ajratilgan —
  auditoriyasi boshqa. O'zbekcha manba: `AdminDict` shundan olinadi, ru/en da
  kalit tushib qolsa kompilyatsiya xatosi. Til bir xil `lang` cookie'da, ya'ni
  saytda tanlangan til panelga ham o'tadi. Har ikkalasida `LangSwitch` va
  `ThemeToggle` bor (kirish sahifalarida ham).
- Buyurtma holati yorliqlari (`status`) va "keyingi qadam" tugmasi matnlari
  ham lug'atda; `nextActionLabel(order, t.nextAction)` ko'rinishida chaqiriladi.

### Chegirmalar: promokod va aksiya
- **Bitta model, ikkita tetik** (`promotion`): `trigger: "code"` — mijoz
  kiritadi, `trigger: "auto"` — o'zi ishlaydi. Kassaga ikkalasi bir xil.
- **Yagona narxlash quvuri** `handlers/pricing.go` (`computePrice`):
  `subtotal → eng foydali avtomatik aksiya (faqat BITTA) → promokod →
  yetkazish`. Har qadam faqat **qolganini** oladi, total 0 dan past tushmaydi.
  Ikkita ustma-ust kampaniya yarim narx yasab yubormasligi uchun avtomatik
  aksiyalardan faqat eng foydalisi qo'llanadi.
- `POST /orders/quote` — checkout'ning preview'i, **aynan shu quvurni** yuritadi;
  `CreateOrder` esa hammasini qaytadan hisoblaydi (brauzerga ishonilmaydi).
- **Minimal buyurtma chegirmadan keyin** tekshiriladi: restoranga tushadigan
  pul muhim (`payableSubtotal`, `belowMinimum`).
- Har chegirma `order.discounts` da **nomi va summasi bilan nusxa** — kampaniya
  o'chirilsa ham chek o'zini tushuntiradi.
- Xato promokod halokat emas: buyurtma to'liq narxda o'tadi, `codeError` da
  aniq sabab bo'ladi.
- `usageLimit` `$expr` bilan filtrda qo'riqlanadi; hisob buyurtma
  yaratilgandan **keyin** oshadi. `AdminUpdatePromotion` `usedCount` ni
  formadan qayta yozmaydi.
- Public `GET /promotions` — **faqat aksiyalar**; kodlar hech qachon ro'yxatda
  qaytarilmaydi.
- Panel: `/admin/promotions` (ikki tab, bitta forma).

### Hisobotlar: savdo, kanallar, jamoa
- Uchalasi ham `Report` shakliga tushadi (§"Hisobotlar va Excel eksporti"),
  ya'ni **ekran va Excel bitta hisob**.
- **Savdo dinamikasi** (`salesreport.go`) — davr kun/hafta/oyga bo'linadi va
  **oldingi shuncha uzunlikdagi davr bilan** taqqoslanadi. Yolg'iz jami "oy
  qanday o'tdi" degan savolga javob bera olmaydi: bir xil raqam yaxshi oydan
  keyin ham, yomonidan keyin ham chiqadi.
  - ⚠️ **Bandlar Go'da, local vaqtda kesiladi**, Mongo'ning `$dateToString` i
    bilan emas. Drayver sanani doim UTC beradi (§"Mongo'dan kelgan sana doim
    UTC"), ya'ni Toshkentda 19:00 dan keyingi butun kechki savdo **ertangi**
    kunga tushardi — va grafik baribir haqiqiy oyga o'xshab turardi.
  - ⚠️ **O'rtacha chek olingan buyurtmalar soniga** bo'linadi (`avgCheck`),
    hammasiga emas: aks holda oshxona bandroq bo'lgan sari o'rtacha chek
    pasayadi — o'qilayotgan narsaning teskarisi.
  - Taqqoslash **faqat ekranda**: Excel'ga ikkinchi davrning qatorlari tushsa,
    varaqni belgilagan odam ularni har ustun jamiga qo'shib yuboradi.
  - Bo'sh asos bo'lsa **foiz ko'rsatilmaydi** (nolddan "100% o'sish" ma'lumot
    emas), va bu ekranda yozib qo'yilgan — strelkasiz ekran "hech nima
    o'zgarmadi" bo'lib o'qilmasligi uchun.
- **Kanal analitikasi** (`channelreport.go`) — **ikki kesim, hech qachon
  qo'shilmaydi**: `order.channel` (sayt / Telegram / operator — qayerdan
  berilgan) va `order.type` (yetkazish / olib ketish / stolda — qanday
  yetkazilgan). Ular kesishadi (botdan olib ketishga buyurtma berish odatiy
  hol), ya'ni bitta ro'yxat ustunni haqiqiy jamidan katta qilardi.
  - **"Yangi mijoz" — butun tarixdagi birinchi buyurtmasi shu davrga tushgani**,
    davr ichidagi birinchisi emas: aks holda har davrda hamma yangi bo'lib
    chiqadi va shahardagi eng sodiq bazasi bor restoran "hech kimni ushlab
    tura olmaydi" deb ko'rsatiladi.
  - ⚠️ Mijoz `userId`, u bo'lmasa **telefon** bo'yicha sanaladi (`customerKey`).
    Yalang'och `userId` bo'sh ObjectID tufayli hisobsiz bergan har bir
    buyurtmani **bitta juda faol mijozga** aylantirardi — va "Mijozlar: 1"
    to'rt yuz buyurtma yonida sokin kanalga o'xshaydi, xatoga emas.
  - Eski, `channel` maydonidan oldingi buyurtmalar **"noma'lum"**, "sayt" emas:
    taxmin qilingan qator bilinadigan qatordan farq qilmay qoladi.
- **Jamoa** (`teamreport.go`) — kuryerlar va ishchilar, har biri bitta sahifada.
  Shaxsiy ekranlar "Aziz qanday ishlayapti" ga javob beradi; bu — "bir-biriga
  nisbatan qanday", va u saralangan ro'yxat talab qiladi.
  - **Qoidalar qayta yozilmaydi**: kuryer daromadi `courierEarning` dan, ishchi
    kuni `buildDays`/`payForDay` dan. Qayta hisoblagan hisobot ertami-kechmi
    oylik varaqasi bilan ziddiyatga tushadi va qaysi biri to'g'riligini
    aniqlashning yo'li qolmaydi.
  - **O'rtacha yetkazish vaqti "yo'lga chiqdi" → "yetkazildi"**: oshxonada
    kutgan qirq daqiqa muammo, lekin **bu odamning** muammosi emas. Qayta
    jo'natilgan buyurtmada **oxirgi** yugurish o'lchanadi.
  - ⚠️ Vaqt o'lchanmagan bo'lsa katak **bo'sh**, 0 emas: o'rtacha ustunidagi 0
    "bir zumda yetkazdi" bo'lib o'qiladi. Yonida nechta buyurtmaga
    tayanganligi turadi (ABC/XYZ dagi "necha kun sotilgan" bilan bir qoida).
  - **"Qo'lida" butun tarix bo'yicha**: yig'ilgan − topshirilgan, davrga
    bog'liq emas. Har oy o'zini tozalaydigan qarz qarz emas.
  - Jamida **o'rtacha yo'q**: o'rtachalarning o'rtachasi o'rtacha emas.
  - Ishchida **"Hisoblangan" va "To'langan" alohida ustun**: birinchisi
    kalendardan chiqqan hisob, ikkinchisi kassadan chiqqan pul.

### AI yordamchi: ertalabki brifing (`insight`)

- **Raqamlar bu yerda hisoblanadi, so'zlar modeldan keladi.** Ega
  so'raydigan har bir narsa — kim kelmay qo'ydi, nima sotilmayapti, qaysi ombor
  sanalmagan — bu tizim egalik qiladigan ma'lumot ustidagi aniq arifmetika.
  ⚠️ Modeldan raqam chiqarishni so'rash ularni **ba'zan noto'g'ri** qiladi, va
  narxi assimetrik: bitta noto'g'ri raqamni tutgan ega qolgan to'rttasiga ham
  ishonmay qo'yadi — xususiyat o'chirilmaydi, shunchaki qaytib o'qilmaydi.
- **Fakt nomini aytmagan kartochka tashlanadi** (`insight.Keep`). Model
  javobda faqat kalit qaytaradi; biz yubormagan kalit chizilmaydi. O'ylab
  topilgan statistikaga yashaydigan joy qolmaydi.
- ⚠️ **Brifing linzani so'ramas edi, server esa uni linza bo'yicha saqlaydi.**
  Kunlik kartalar `(kun, til, brend/filial)` bo'yicha yoziladi — bu birinchi
  kundan shunday — lekin panel `/admin/insights` ni **qamrovsiz** so'rardi,
  ya'ni ikki brendli kompaniya ikkalasida ham qamrovsiz nusxani olardi. Ega
  brendni almashtirganda sahifaning o'z raqamlari o'zgarardi, ustidagi kartalar
  esa **joyida qolardi**: yordamchi ekranda bo'lmagan restoran haqida gapirib
  turgandek. Hech nima xato bermaydi va kartalar **rost** — faqat boshqa
  biznesning rosti. Endi chaqiruv `scope: true`, va komponent `scopeKey` ni
  bog'liqlik sifatida oladi.
- ⚠️ **Linza yuklanmaguncha hech nima so'ralmaydi.** Sahifaning birinchi
  lahzasida brend bo'sh, va o'sha paytdagi so'rov shunchaki erta emas: javob
  modelga pul turadi va kunlik slotga yoziladi, ya'ni hech kim qaramaydigan
  qamrov kunning javoblaridan bittasini sarflab qo'yardi — keyin esa ekranda
  bir soniya ko'rinib, almashardi.

**⚠️ Brifing yettita signalizatsiya edi (2026-08-30 da tuzatildi).** Yo'qolgan
mijozlar, tushayotgan hafta, o'lik taomlar, sanalmagan ombor, kam qoldiq,
bekor qilish chempioni, tushuntirilmagan kamomad — hammasi rost, hammasi
muammo. Birgalikda bu ikkinchi haftadayoq ko'z yugurtirib o'tiladigan hisobot.
Prompt'ning o'z qoidasi ("faqat muammo aytadigan brifing bir haftada shovqinga
aylanadi") tayanadigan hech narsa yo'q edi.

Qo'shilgan beshta fakt (`insightgrowth.go`) — eganing o'z savollari:
- `top_dishes` — haftani nima ko'tarib turibdi va ulushi qancha.
  ⚠️ Ro'yxat emas, **ulush**: "Osh 84 ta sotildi" — ega o'zi biladigan raqam;
  "uchta taom haftaning 41% i" — qolgan qirqta bilan nima qilishni o'zgartiradi.
- `dish_movement` — eng ko'p siljigan taom, **ikkala yo'nalishda ham bitta
  faktda**. Ko'tarilgan va tushgan — bitta savol ("menyuda nima o'zgardi"), va
  ikkiga bo'lish bir xil `Area` da biridan mahrum qiladi.
  ⚠️ O'tgan hafta nol sotgan taom chiqarib tashlanadi: u cheksiz o'sgan, bu
  arifmetika, yangilik emas — birinchi versiyada u har haftani egallab olardi.
- `server_output` — zaldagi tarqoqlik. ⚠️ **O'rtacha chek, jami emas**: band
  seksiyadagi ofitsiant ko'proq pul oladi, chunki mehmonlar o'sha yerda.
  Boshqarish mumkin bo'lgani — stol boshiga tushadigan summa.
  ⚠️ Kamida uchta ofitsiant va har birida o'ntadan chek: ikki kishida biri doim
  "zaifroq" (bu reyting, xulosa emas), o'ndan kam chekda o'rtacha — bitta
  katta stol.
- `waste_share` — chiqim xariddan ulush sifatida. ⚠️ Mutlaq raqam yolg'iz hech
  nima demaydi: yarim million so'm kafeda falokat, banket oshxonasida
  yaxlitlash xatosi.
- `quiet_hours` — bo'sh soat va cho'qqi. ⚠️ **Eng kichik soat emas**: har qanday
  restoranning eng jim soati — ochilish soati, va "soat 10 da jimsiz" degan
  kartochka jadval haqidagi kartochka. Soat kun ichida (cho'qqidan ±6) va
  cho'qqidan kamida uch marta jim bo'lishi shart.

⚠️ **Qarorlar so'rovdan ajratilgan** (`pickDishMovement`, `pickServerGap`,
`pickQuietHour`) va testlari bor. Agregatsiya ichida qolgan chegara faqat jonli
bazada ishlaydi, ya'ni uni hech kim tekshirmaydi.

**⚠️ Brifing uch tilli edi-yu, doim o'zbekcha chiqardi.** Til `lang` cookie'sida
turardi, panel esa API'ga boshqa origindan murojaat qiladi — cookie yo'lda
yo'qolar, server zaxiraga (o'zbekcha) tushardi. Xususiyatning butun mahsuloti
jumlalar bo'lgani uchun bu uni **rus egasi uchun butunlay ishlatib bo'lmaydigan**
qilardi, va hech qanday alomat bermasdi. Tuzatildi: `X-Keel-Lang` sarlavhasi har
so'rovda ketadi — qarang «Server xabarlari ham uch tilda» → «To'rtinchi teshik».
Saqlangan brifing kaliti allaqachon tilni o'z ichiga oladi, ya'ni uch til uch
kartochka to'plami bo'lib yashaydi.

**⚠️ Sarlavha muammosi prompt'da edi.** "Title: at most six words, naming the
thing" degan qoida aynan "Haftalik tushum pasayishi" ni chiqaradi — turkum
nomi, xulosa emas. Endi qoida: **sarlavha nima bo'lganini aytadi**, iloji
bo'lsa raqam bilan ("Tushum o'tgan haftadan 18% past"), va misollar bilan
ko'rsatilgan. Tana esa uch bo'lakli: nima bo'lyapti → ehtimoliy sabab (faqat
berilgan raqamlardan, hukm emas) → bugun qilinadigan bitta ish.

### Sozlanadigan KPI dashboard
- `admin_user.dashboard {hidden, order}` — **har admin uchun alohida**, kompaniya
  uchun emas: ega tushum va o'rtacha chekka qaraydi, filial menejeri nima qabul
  qilinmaganiga va kim smenada ekaniga. Umumiy tartib "oxirgi tartiblagan odam
  hamma uchun qaror qildi" degani bo'lardi.
- ⚠️ **Ro'yxat nima o'chiq ekanini yozadi, nima yoqiq ekanini emas.** Bo'sh
  qiymat = bugungi dashboard (bo'sh `mapProvider` = 2GIS, `hidePlan` bilan bir
  qoida): mavjud har bir hisobda sozlama yo'q, ya'ni boshqacha o'qilgan nol
  qiymat chiqqan kuni **hamma adminning** ekranini bo'shatardi. Va keyingi
  versiyada qo'shilgan plitka hammaga o'zi chiqadi — saqlangan allowlist uni
  har bir mavjud hisobdan abadiy yashirardi.
- **Tartib qisman**: nomlangan plitkalar oldinga, qolgani o'z joyida. Ikki
  plitkani tepaga surgan odam qolgan o'n sakkiztasining tartibini hal
  qilmaydi.
- Plitka **guruhini o'zgartira olmaydi**: "Pul" sarlavhasi ostidagi buyurtma
  soni — yolg'on sarlavha, sarlavhalar esa yigirmata raqamni o'qilarli
  qiladigan yagona narsa.
- Reyestr `frontend/src/lib/dashboardTiles.ts` da, id'lar serverdagi
  `dashboardTileIDs` bilan bir xil. ⚠️ Id **saqlanadi**: uni qayta nomlash
  o'chirgan har bir hisobda o'sha plitkani jimgina qaytaradi.

### RFM: baza o'ziga nisbatan
- Qoidali segmentlar (§CRM) **qoldi**, RFM ularning **yoniga** qo'shildi.
  Ikkalasi ham kerak: qoida yomon oydan omon qoladi ("60 kun" yanvarda ham,
  iyulda ham bir xil), ranking narx o'zgarishidan omon qoladi — 30%
  qimmatlashtirgan restoran bir xona "ko'p sarflaydigan" mijoz orttirmaydi,
  qat'iy summa esa orttirdi deydi. "Tug'ilgan kun" va "norozi" RFM'ga umuman
  sig'maydi: ular sotib olish xatti-harakati emas.
- ⚠️ **Bir mijoz — bitta xona** (qoidali segmentlardan farqli, u yerda "uxlab
  qolgan VIP" aynan kerak bo'lgan juftlik). Ikki marta egallash mumkin bo'lgan
  katak katak emas, va yon ustundagi sanoqlar bazadan katta chiqardi.
- Ballar **kvintil bo'yicha, o'z bazasiga nisbatan** (`vipFloor` bilan bir
  mantiq). Xonalar: `champions / loyal / bigSpender / promising / atRisk /
  needsAttention / lost`.
- ⚠️ **Recency teskari**: o'q "oxirgi buyurtmadan beri necha kun", ya'ni kichik
  yaxshi. Boshqa ikkitasi kabi ballansa **eng uzoq ketganlar** ekran tepasida
  "champions" bo'lib turardi — va hamma raqam ishonarli ko'rinardi.
- ⚠️ **Ball qiymat bo'yicha, ro'yxatdagi o'rin bo'yicha emas**: bazaning yarmi
  aynan bir marta buyurtma qilgan, o'rin bo'yicha bo'lish bir xil ikki mijozni
  ikki xil segmentga sochardi — va buni ega ro'yxatni saralagan zahoti ko'radi.
- ⚠️ **Tekshiruvlar tartibi ta'rifning o'zi**: `atRisk` `loyal` dan **oldin**
  (tez-tez buyurtma qilib, endi jim bo'lgan mijoz — bu ekran chiqaradigan eng
  qimmatli narsa; teskari tartibda u "sog'lom" deb belgilanardi), `bigSpender`
  `lost` dan oldin (ketgan banket mijozi qo'ng'iroqqa arziydi).
- ⚠️ Kampaniya auditoriyasida **`rfm:` prefiksi shart**: `lost` ikkala
  ro'yxatda ham bor va boshqa narsani anglatadi (180 kunlik qat'iy chiziq va
  "recency bo'yicha eng pastki beshdan bir"). Bir xil id bo'lsa bittasiga
  mo'ljallangan kampaniya jimgina ikkinchisiga ketardi, va ro'yxatlar sanog'i
  bilan farqlanmaydigan darajada ustma-ust tushadi.
- Baza 10 dan kichik bo'lsa RFM **umuman yo'q** (bo'sh jadval emas — sabab va
  minimal son yoziladi): to'rtinchi mijozni "champion" deb atash restoranning
  yoshi haqidagi gap.

### Web push: uchinchi kanal
- Kampaniyalar endi **SMS · Telegram · brauzer**. Uchtasi **bir-birining
  o'rnini bosmaydi**: SMS hammaga yetadi va pul turadi, Telegram botni ochganga,
  push esa saytda ruxsat berganga — va push **kompyuterda o'tirgan mijozga
  yetadigan yagona kanal**, ofis tushligi uchun aynan shu ko'pchilik.
- `internal/webpush` — RFC 8291 (shifrlash) + RFC 8292 (VAPID), **yangi
  bog'liqliksiz**: `crypto/ecdh` va `crypto/hkdf` stdlib'da. Rasm
  kichraytirgichdagi bilan bir savdo — har tenant nomidan tarmoqqa chiqadigan
  kutubxonaning keyingi versiyasiga ishonish kerak bo'lardi.
- ⚠️ **Test brauzer tomonini yozib deshifrlaydi**, chunki bu yerdagi har bir
  xato serverdan **ko'rinmaydi**: tana shifrlanadi, push xizmati 201
  qaytaradi, bildirishnoma esa kelmaydi. Shu shaklda muhrlangan ikki tuzoq:
  `key_info` dagi kalitlar tartibi (**mijozniki birinchi**) va imzoning
  **xom r‖s** bo'lishi — `ecdsa.SignASN1` bergan DER ham xuddi shu imzo, va
  har bir push xizmati uni "invalid JWT" deb rad etadi (kalitga ishora
  qiladigan xato).
- ⚠️ **Kalitlar sozlanmaydi, generatsiya qilinadi** (`push_settings`, birinchi
  ishlatishda). To'lov yoki SMS kalitlaridan farqli: ro'yxatdan o'tadigan joy
  yo'q, va "provayderda hisob oching" deb boshlanadigan xususiyatni hech bir
  restoran yoqmaydi. **Almashtirilmaydi ham**: har obuna o'zi yaratilgan ochiq
  kalitga bog'langan, ya'ni yangilash butun bazani jimgina o'ldiradi.
- ⚠️ `404`/`410` — qayta urinish emas, **o'lgan obuna** (mijoz sayt
  ma'lumotlarini tozalagan yoki ruxsatni qaytarib olgan): darhol o'chiriladi.
  Aks holda har kampaniyada urinib ko'riladi va "yuborildi" sanog'i qamrovni
  oshirib ko'rsatadi.
- ⚠️ `push_subscription.endpoint` **unique**: service worker brauzer
  yangilanganidan keyin jimgina qayta ro'yxatdan o'tadi, indekssiz mijoz har
  kampaniyani ikki, keyin uch marta olardi — va alomati restoran takrorlay
  olmaydigan shikoyat.
- **Dedublikatsiya hisob bo'yicha, qurilma bo'yicha emas**: telefoni ham,
  noutbuki ham jiringlaydi, lekin bu **bitta odam** — tasdiqlash ekranidagi son
  odamlar soni, bildirishnomalar soni emas.
- Sayt: `public/push-sw.js` **hech nima keshlamaydi** (kuryer worker'idan
  ataylab boshqa: ommaviy saytda keshlovchi worker menyuni va narxni eskitardi),
  `lib/push.ts`, profil sahifasidagi tugma.
- ⚠️ **Ruxsat sahifa ochilganda emas, tugma bosilganda so'raladi.** Brauzer uni
  umr bo'yi **bir marta** so'raydi; rad etilgandan keyin oyna boshqa chiqmaydi
  va sahifa uni qaytara olmaydi (geolokatsiya bilan bir dars, `lib/geo.tsx`).
  Menyuni o'qiyotgan odamga chiqqan oyna — yopiladigan oyna, va u yagona
  imkoniyatni sarflaydi.

### Upsell va kross-sotuv (`handlers/recommend.go`)
- Ikki manba, **ikkalasi ham shart**: 90 kunlik tarixdan **birga sotilganlar**
  (xotirada 30 daqiqa keshlanadi — SMS sender keshi bilan bir naqsh) va
  **ega qo'lda tanlagani** (`menu_item.recommendedIds`).
- ⚠️ **Qo'lda tanlashsiz xususiyat yangi taomni hech qachon ko'tara olmaydi**:
  dushanba qo'shilgan taomning tarixi yo'q, ya'ni avtomatik yarim faqat
  allaqachon sotilayotganini tavsiya qilardi — eganing undan foydalanish
  sababining aynan teskarisi. Qo'lda tanlanganlar **birinchi**: u qaror, sanoq
  esa kuzatuv, va kuzatuv bekor qila oladigan qaror qaror emas.
- Taom **o'zini tavsiya qila olmaydi** (serverda ham tekshiriladi): tanlagich
  butun menyuni ko'rsatadi, ya'ni bu eng oson xato — va natijasi mehmon
  allaqachon qarab turgan taomni taklif qilish, ya'ni buzuq vidjet.
- ⚠️ **Tugagan yoki sotuvda bo'lmagan taom taklif qilinmaydi** (menyudagi bilan
  bir linza). Menyuda kulrang kartochka mehmon **bergan** savolga javob beradi;
  bu yerda hech kim so'ramagan, ya'ni sotib bo'lmaydigan taklif — shovqin, va u
  "restoran o'z menyusini bilmaydi" bo'lib o'qiladi.
- **Savat bo'yicha ballar qo'shiladi**, har taomga alohida emas: uch taomlik
  savatda foydali javob "shu **ovqatga** nima yarashadi", har biriga alohida
  eng kuchli juftlik esa uchta bog'liqsiz javob.
- Bekor qilingan buyurtmalar hisobga kirmaydi: bekor qilish — o'sha taomlar
  birga **ketmagani**ning yagona signali.
- Ko'rinadigan joylar: **taom sahifasi**, **savat**, **checkout** — oxirgi
  ikkalasida ham **tugmadan pastda**. Mehmon bir bosishda to'lashga tayyor
  turganda tugmani ekrandan surib yuboradigan taklif upsell yutish uchun
  buyurtma yo'qotadi.
- ⚠️ Va **call-markaz kartochkasida**, narxi bilan: telefon — upsell haqiqatan
  ishlaydigan, lekin kartochka qo'yib bo'lmaydigan yagona kanal. Operatorda
  o'ylashga uch soniya bor, ya'ni amalda hech kim taklif qilmaydi. Mijozning
  odatidan (`favourites`) quriladi, chunki buyurtma hali olinmagan.

### CRM: segmentlar, kartochka, fikrlar
- **Segmentlar hisoblanadi, saqlanmaydi** (`handlers/crm.go`). Har biri bitta
  jumlada tushuntiriladi va panelda ta'rifi ko'rsatiladi — egasi ta'riflay
  olmaydigan guruhga hech nima yubormaydi.
- Mijoz **bir vaqtda bir nechta segmentda** bo'ladi ("uxlab qolgan VIP" —
  aynan shu juftlik muhim).
- **VIP nisbiy** (eng yuqori 10%), qat'iy summa emas: u restoran turiga qarab
  boshqacha va narxlar oshgani sari eskiradi. Baza <10 kishi bo'lsa VIP yo'q.
- `user.birthday` — **`MM-DD`, yilsiz**. Tabrik uchun kun yetarli.
- Ism/telefon paneldan tahrirlanmaydi (telefon SMS bilan tasdiqlangan).
- `user.source` birinchi buyurtmada avtomatik, faqat bo'sh bo'lsa.
- **Fikr**: `feedback`, bitta buyurtmaga bitta. Kuzatuv sahifasida so'raladi;
  yaxshi baho bir bosishda, izoh maydoni faqat past bahoda. Hisobga tegishli
  buyurtmada egasining tokeni majburiy.
- **Yopishda "nima qilindi" majburiy** — yozuvsiz "ko'rib chiqildi" ro'yxatni
  chiroyli qiladi-yu foyda bermaydi. Panel javobsiz shikoyatlardan ochiladi.
- Segment **"norozi"** — 60 kun ichida javobsiz shikoyat. Yagona segment
  mijozning emas, restoranning o'z xatti-harakati haqida.

### Segmentlarga xabar yuborish (`/admin/campaigns`)
- Segmentlar bor edi-yu **hech qayerga olib bormasdi**: panel "11 ta VIP jim
  bo'lib qoldi" deb aytardi va keyin qiladigan ishni taklif qilmasdi. Bu — o'sha
  ikkinchi yarmi, va u **haqiqiy pul sarflaydi** hamda odamlarning telefoniga
  boradi, shuning uchun kodning ko'p qismi — qo'riqchi.
- ⚠️ **Rad etgan odamga hech qachon yuborilmaydi** (`user.noMarketing`). Bu
  ekrandagi filtr emas, **qattiq istisno**: rad etgandan keyin yana reklama
  olgan mehmon bizga shikoyat qilmaydi, u restoranning mijozi bo'lishni
  to'xtatadi — ya'ni xususiyat o'z maqsadining teskarisini bajaradi. Buyurtma
  holati va kirish kodlari baribir boradi (ular mijoz **so'ragan** xizmat).
- **Auditoriya hisoblanadi, saqlanmaydi** — segmentlarning o'z qoidasi bilan
  bir xil. Saqlangan ro'yxat keyingi buyurtmada yolg'onga aylanadi: kechagi
  buyurtmachi endi "uxlab qolgan" emas, va unga "sizni sog'indik" chegirmasini
  yuborish ishlaydigan xususiyatni buzuqdek ko'rsatadi.
- **VIP kimligini bitta kod hal qiladi**: `customerFactsByUser` mijozlar
  ro'yxati bilan **umumiy**. Ikki joyda ikki hisob — bir ekranda 24, boshqasida
  19, va egasi ikkalasiga ham ishonmasligi to'g'ri bo'lardi.
- ⚠️ **Narx SMS bo'laklarida ko'rsatiladi, qabul qiluvchilarda emas**
  (`smsParts`). Kirill va to'g'ri yozilgan o'zbek harflari (`oʻ`, `gʻ`) GSM-7
  dan tashqarida, ya'ni bitta SMS **70 belgi**, 160 emas — va hisob har bo'lak
  uchun. Xushmuomala oxirgi jumla qo'shgan ega kampaniya narxini ikki barobar
  qilishi mumkin va ekranda hech nima o'zgarmaydi. Testda muhrlangan.
- **Bir vaqtda bitta kampaniya** (409): ustma-ust ketgan ikki yuborish bir
  mijozga bir xil reklamani ikki marta yuboradi, va SMS'ni qaytarib olish yo'q.
- **Bitta telefonga bitta xabar**, hisob soniga qarab emas: bir raqamni
  ishlatadigan oila yoki ikki hisobi bor mijoz uchun ikki marta to'lanmaydi.
- **Shlyuz sozlanmagan bo'lsa rad etiladi** (503), jimgina "yuborildi"
  demaydi: "240 kishiga yuborildi" deb yozib hech kimga yetmagan kampaniya —
  eng yomon natija, chunki ega unga ishonib kutadi va mijozlar qiziqmagan
  degan xulosaga keladi.
- **Faqat owner**: mijozlar bazasi kompaniyaniki, menejer bitta oshxonani
  yuritadi, bu esa bir bosishda **hamma mehmonni** bezovta qila oladigan tugma.
- Yuborish **fonda** ketadi (`context.WithoutCancel`): operator tabni yopgani
  uchun yarim yo'lda to'xtagan kampaniya segmentning tasodifiy yarmiga
  yuborilgan bo'ladi — qaytarib bo'lmaydigan yagona natija. Jarayon har 20
  xabarda bazaga yoziladi, chunki bu — odam qarab turadigan ekran.
- Shlyuzning **birinchi xatosi aynan saqlanadi**: "84 yetmadi" degan yozuv
  "hisobda pul yo'q", "jo'natuvchi nomi tasdiqlanmagan" va "bizning xatomiz"
  ni ajratib bermaydi — uchtasining keyingi qadami butunlay boshqa.

### Loyalty: keshbek ballari
- **1 ball = 1 so'm** — kurs o'rganish shart emas. Sozlamalar kompaniya
  darajasida (`restaurant.loyalty`), chunki mijoz ham shunday.
- **Ball buyurtmada yechiladi, yetkazilganda qo'shiladi**: yechish darhol
  bo'lmasa bir balans ikki buyurtmaga va'da qilinardi; qo'shish kutmasa bekor
  qilingan buyurtma yo'qdan ball yasardi. Ikkalasi ham **idempotent**
  (`loyalty_txn` qo'riqlaydi, buyurtmadagi bayroq emas).
- **Ball bilan to'lash ball keltirmaydi**: keshbek bazasi
  `subtotal − chegirmalar − ishlatilgan ballar`. Aks holda balans o'z-o'zini
  boqadi.
- **`maxRedeemPercent` shifti majburiy** (standart 50%) — shiftsiz katta
  balans buyurtmani bepul qiladi, oshxona esa pishiradi.
- Har harakat `loyalty_txn` da (`earn`/`spend`/`revoke`/`adjust`), balans
  foydalanuvchida `$inc` bilan siljiydi. Ledger — yozuv, balans — kesh.
- Narxlash quvurida ballar **chegirmalardan keyin**: ball mijozning o'z puli,
  aksiya baribir olib tashlaydigan summaga sarflanmasligi kerak.
- Bekor qilinganda: ishlatilgan ball qaytadi, berilgan keshbek olinadi.

### Taom variantlari: nima uchun qayta yozildi

**⚠️ Blok ikki ustunli formaning yarmiga siqilgan edi.** «Taom / To'plam»
kartochkasida `sm:col-span-2` yo'q edi, ya'ni u yarim ustunda turardi va
`OptionsEditor` ning o'z `col-span-2` si hech nima qila olmasdi — u bu grid'ning
farzandi emas. Natijada uchta nom maydoni **20 pikselga** qisqargan, yorliqlari
esa ikki qatorga o'ralgan. Formaning o'ng yarmi esa bo'sh turardi.

**⚠️ Ega «variant guruhi» va «tanlov» deb o'ylamaydi.** U «menda kichik va katta
bor, kattasi besh ming qimmat» deb o'ylaydi. Eski matn — «Variantlar (ixtiyoriy
— masalan hajm yoki qo'shimcha)» — xususiyat nomini aytadi, nima uchun
kerakligini emas. Endi:
- «Mijozdan nima so'raladi?» (guruh nomi o'rniga)
- «Javob varianti» (tanlov o'rniga)
- Bo'sh holatda ikki jumlalik tushuntirish va **bitta ishlangan misol**.

**⚠️ Ishorali farq — narx emas, lekin ega uni narx deb o'qiydi.** "Narx farqi"
deb nomlangan maydon yonidagi «+5000» — bu ekrandagi eng ko'p uchraydigan
chalkashlik: odamlarning yarmi u yerga **yakuniy** narxni yozib, 50 000 so'mlik
taomni 95 000 qilib qo'yadi. Endi yonida «mijoz 50 000 so'm to'laydi» yozilib
turadi — maydon o'z joyida qoldi, noaniqlik ketdi.

**⚠️ Tarjimalar qatorning to'rtdan uchini egallardi.** RU va EN har qatorda
alohida ustun edi, ya'ni ega kelgan maydon — nom — kenglikning chorak qismi
bo'lardi, va restoranlarning ko'pchiligi RU/EN ni umuman to'ldirmaydi. Endi ular
savol ostida yig'ilgan, qator esa nom va narxdan iborat.

**⚠️ «Mijoz shuni ko'radi» qatori.** Forma mavhum: «majburiy», «bir nechta» va
ishorali son bilan ega natijani tasavvur qila olmaydi, va buni bilishning yagona
yo'li saqlab, saytni ochib, qarash edi. Bitta qator ko'rinish butun blokni
joyida tekshirib bo'ladigan narsaga aylantiradi.

### Combo (belgilangan to'plam)
- Combo — **alohida kolleksiya emas**, `menu_item` ning bir turi:
  `comboItems [{menuItemId, qty}]` bo'sh bo'lmasa bu to'plam. Shu sabab rasm,
  tarjima, kategoriya, brend qamrovi, savat va buyurtma — hammasi tayyor ishlaydi.
- `comboBasePrice` va `comboContents` — **hisoblanadi, saqlanmaydi**
  (`bson:"-"`, `handlers/combo.go`). Saqlangan tejash a'zo narxi o'zgarishi
  bilan yolg'onga aylanadi.
- A'zosi tugagan yoki menyudan o'chirilgan to'plam sotilmaydi; a'zoni o'chirishga
  urinish 409 qaytaradi. Chekka tarkib **nusxa** bo'lib yoziladi — oshxona
  chekdan pishiradi.
- **Majburiy variantli taom to'plamga kirmaydi** ("qaysi hajm?" degan savolni
  belgilangan to'plamda so'rash joyi yo'q). Guruhdan tanlanadigan to'plam —
  ataylab qurilmagan.
- Panel: `components/admin/ComboEditor.tsx`, menyu formasida "Taom / To'plam".

### Menyu qidiruvi va filtrlar (`/menu`)
- ⚠️ **Elasticsearch emas, va ataylab emas.** Bitta VPS'dagi bitta restoran
  uchun qidiruv klasteri — butun stackdan ko'p RAM yeydigan JVM. Kerak bo'lgani
  — odamlar **qanday yozishiga** chidaydigan qidiruv.
- **Brauzerda ishlaydi** (`lib/search.ts`): menyu allaqachon sahifada (~100
  taom), har harfga so'rov yuborish esa mobil internetda har safar ~300 ms —
  eng tez bo'lishi kerak bo'lgan ekranni eng sekiniga aylantirardi. Backend
  bir daqiqaga yiqilsa ham qidiruv ishlaydi.
- ⚠️ **Ikkala tomon ham bir alifboga "buklanadi"** (`fold`): faqat so'rovni
  buklash xususiyatni yarim ishlaydigan qilardi — "лагман" hech nima topmasdi,
  "lagmon" hammasini topardi, va bitta yozuvda sinagan odam buni ko'rmasdi.
  Buklash: kirill→lotin, **oltita apostrof shakli** o'chiriladi (`o'`, `oʻ`,
  `o‘`…), `x↔h`, `q↔k`, `v↔w`, `ts↔s`, `u↔o` — bular "xato yozuv" emas, ikki
  yozuvning o'rtada uchrashuvi ("qaymoq"/"kaymak" — bir so'z, ikki odam).
- **Xatoga chidaydi**: chegaralangan Levenshtein (4–6 harf → 1 xato, uzunroq →
  2). Tushib qolgan harf ekranni bo'shatmasligi kerak — mehmon o'z xatosini
  emas, "bu restoranda lag'mon yo'q" ni ko'radi.
- ⚠️ **Har bir so'z mos kelishi shart** (AND): yig'indi bo'yicha saralash
  "achchiq lag'mon" ga barcha lag'monni **va** barcha achchiq taomni qaytarardi
  — ya'ni ko'proq yozish natijani **kengaytirardi**.
- Maydon og'irliklari: nom > bo'lim > teg > to'plam tarkibi > tavsif. Sotuvda
  bo'lmagan taom **pastga tushadi, yo'qolmaydi**: uni qidirgan odam
  "bugun tugadi" javobini olishi kerak.
- ⚠️ **Standart holat serverda chizilgani bilan bir xil** — butun menyu HTML
  ichida. Bu faqat hidratsiya emas: menyu va taom sahifalari — restoran
  topiladigan yagona kontent (§SEO), JS'dan keyin paydo bo'ladigan menyu esa
  Google uchun bo'sh menyu.
- **Filtrlar menyudan kelib chiqadi** (`facets`): to'plami yo'q restoran
  "To'plamlar" tugmasini ko'rmaydi — bo'sh boshqaruv panelni bezak deb
  o'rgatadi. Filtr ikonkasida **faol filtrlar soni**: ikki ekran pastdagi
  qisqa ro'yxatni tushuntiradigan yagona narsa.
- Qidiruv/filtr faol bo'lganda **bo'lim relsi yashiriladi** (uning
  langarlari ekranda yo'q bo'limlarga ishora qiladi).
- **Qidiruv `?q=` da**: bosh sahifadagi quti so'rovni shu bilan uzatadi, va
  topilgan taomni birovga yuborish mumkin. Manzil qatori `replaceState` bilan
  yangilanadi — har harfda `router.push` bo'lsa marshrut qayta chizilar va
  "Orqaga" tugmasi **bitta harf o'chirish** bo'lib qolardi. Dublikat sahifa
  yaratmaydi: canonical faqat **yo'ldan** quriladi (middleware sarlavhasi),
  ya'ni har `/menu?q=…` baribir `/menu` ni haqiqiy manzil deb e'lon qiladi.
- **Bosh sahifada ham bor** — `search` bandi (`DEFAULT_SECTIONS` da, hero'dan
  keyin; konsol konstruktorida qo'shiladi/olib tashlanadi, `bar`/`big`
  variantlari). Bosh sahifa **allaqachon butun menyuni yuklaydi** (kategoriya
  plitkalari va mashhur taomlar o'shandan chiziladi), ya'ni jonli takliflar
  bepul. Quti **taklif beradi, natija sahifasiga aylanmaydi**: oltita taom
  ko'rsatiladi, oxirgi qator esa `/menu?q=…` ga olib chiqadi — mehmon ikki
  marta yozmaydi.

### Ulushlab sotish: yarim non, chorak shisha
Non yarim bo'lak sotiladi, ochilgan alkogol chorak stakan, osh 0.75 porsiya —
va bularning hammasi **ombordan ham shuncha** kamayishi kerak. Shishadagi suvga
esa taalluqli emas: yopiq shishani yarimlab bo'lmaydi.

- **Qaysi taom bo'linishini restoran aytadi** (`menu_item.portions` — foizlar
  ro'yxati: 25, 33, 50, 75). ⚠️ Bo'sh — «faqat butun porsiya», ya'ni shu
  paytgacha yozilgan har bir menyu. Teskarisini standart qilish ikki yuz taomga
  hech kim so'ramagan «yarim» tugmasini qo'yardi.
- ⚠️ **Foiz, kasr emas**: `0.1 + 0.2` ikkilik sanoqda `0.3` emas, bu sonlar esa
  tenglikka solishtiriladi (taom shu ulushni sotadimi?), saqlanadi va **pulga
  ko'paytiriladi**.
- ⚠️ **`Qty` butun son bo'lib qoladi**, chunki u hisobotlar, chek, fiskal
  hujjat va POS ko'prigining hammasida butun son. Yarim nonni sotish uchun uni
  float qilish — o'sha hamma joyga tegish, va birinchi yaxlitlagan joy buni
  **jimgina** qiladi. Shuning uchun qatorda alohida `portion` (foiz):
  «ikkita yarim» — `qty: 2, portion: 50`, ofitsiant ham shunday aytadi.
- ⚠️ **`Price` — allaqachon ulushning narxi**: u serverda `menuLine` da
  hisoblanadi, ya'ni subtotal, chek, fiskal qator va POS ulush borligini
  bilmasdan to'g'ri ishlaydi. Yaxlitlash **bir marta va shu yerda**
  (`models.PortionPrice`): 22 999 ning yarmi 11 499.5, va uni har safar pastga
  yaxlitlash restoranga har sotuvda bir so'mga tushadi.
- ⚠️ **Ulush serverda tekshiriladi** (`AllowsPortion`): ekran bo'linmaydigan
  taomda tugmani ko'rsatmaydi, lekin so'rov baribir yubora oladi — va
  bo'linmaydigan taomning «yarmi» — butun narsani yarim narxga sotish.
- ⚠️ **Yarim non va butun non — ikki qator** (`mergeableLine`, oflayn nusxada
  ham). Birlashtirilsa yarim qatorning ichida yashirinardi: oshxona ikkita
  butun qilardi, mehmon bittayu yarimga to'lardi.
- **Ombor**: `soldDishes` endi **kasr** sanaydi (`PortionFactor`). Ilgari
  xaritalar butun son edi va do'kon yarimni butun deb bilardi — ya'ni javon
  aynan hech kim sanamagan yarimlarcha kam chiqardi, oyiga bir marta, va buni
  qo'lida qog'oz ushlab turgan odamdan so'rashardi. To'plamning ulushi ichidagi
  har bir taomga ham tarqaladi.
- **Ekranlarda kasr bilan yoziladi** («1/2 · Non»), foiz bilan emas: «50%» —
  chegirmaga o'xshaydi, va ikkalasini bir peshtaxtada bir xil odamlar o'qiydi.
  Chekda va oshxona chekida ham nomning oldida (`portionLabel`) — «1 x Non»
  yarim non uchun kesilgan va yarmi tashlangan non degani.
- **Qayerda so'raladi**: kassa va zal ekranida variant oynasi (ulush bo'lsa —
  ochiladi, hatto variantlari bo'lmasa ham), ofitsiant ilovasida esa bitta
  savolli varaq. Butun porsiya — birinchi tugma va standart tanlov: sotuvlarning
  deyarli hammasi shu, va odatdagi holatni qidirtirib qo'yish istisnoni
  tasodifan sotdiradi.

### Menyu variantlari (options)
- Taomga variant guruhlari qo'shiladi: `required` (tanlash shart) va
  `multiple` (bir nechta tanlansa bo'ladi) bayroqlari bilan. Har tanlovda
  `priceDelta` — taom narxiga qo'shiladi (manfiy ham bo'lishi mumkin).
- Admin: `/admin/menu` formasida "Variantlar" bloki
  (`components/admin/OptionsEditor.tsx`), RU/EN tarjimalari bilan.
- Mijoz: variantli taom kartochkasida "Qo'shish" o'rniga **"Tanlash"** —
  taom sahifasida `OptionPicker` orqali tanlaydi.
- Savatda qator kaliti **taom + tanlovlar** (`lineIdFor`): bir taom turli
  variantlar bilan alohida qator bo'ladi. `localStorage` kaliti `cart_v2`.
- **Narx serverda qayta hisoblanadi** (`resolveOptions`): mijoz faqat qaysi
  tanlovni belgilaganini yuboradi, `priceDelta` doim bazadan olinadi;
  majburiy guruh tanlanmasa yoki tanlov menyuda bo'lmasa — 400.
  Buyurtmaga tanlovlar base (uz) nomi bilan yoziladi.

⚠️ **To'liq bo'lmagan savol endi jimgina tashlab yuborilmaydi.** `fromOptionDrafts`
nomi bo'sh yoki nomlangan javobi yo'q guruhni filtrlab tashlaydi — bu **to'g'ri**,
nomsiz savol savol emas. Lekin buni hech kimga aytmasdan qilardi, va ega
tomonidan bu shunday ko'rinardi: hajmlarni to'ldirdim, «Saqlash» bosdim, taom
saqlandi — variantlar yo'q. Hech qayerda xato chiqmaydi.

Ega buni **«saqlash bosilgandan keyin saqlanmayapti»** deb xabar qildi, va
ta'rif to'g'ri edi: aytilmagan to'g'ri filtr ishlamagan saqlashdan farq
qilmaydi.

- `optionProblems()` — qaysi savol yarim to'ldirilganini qaytaradi; saqlash
  **rad etiladi** va qatorning **o'zida** yoziladi (ogohlantirish saqlashda
  chiqsa, ega to'rtta savoldan qaysi biri ekanini o'zi topishi kerak).
- ⚠️ **Butunlay bo'sh savol muammo emas** — u «savol qo'shish» tugmasi hozir
  yaratgan qator. Unga ham e'tiroz bildirish forma ochib fikridan qaytgan odamga
  taomni saqlashni taqiqlash bo'lardi, va hech nima yozilmasidan chiqadigan
  ogohlantirishni odam o'qimay qo'yadi.
- ⚠️ Bu qayta yozishdan **keyin** ehtimoli oshgan edi: bo'sh holatda endi
  ishlangan misol turadi («Kichik / Katta»), ya'ni odam misolni o'qib javoblarni
  yozadi va **savol nomi** qutisini bo'sh qoldirganini sezmaydi.

### Taomga izoh va bekor qilish sababi
- **Har bir savat qatoriga izoh**: mijoz savatda taom ostidagi maydonga
  ("piyozsiz", "achchiq qilmang") yozadi. Izoh `cart_v2` da saqlanadi,
  buyurtmaga `items[].comment` bo'lib boradi (serverda trim + 200 belgi).
  Izoh **qator kimligiga kirmaydi** — bir xil taom+variant baribir bitta qator.
  Admin chekida sariq belgi bilan, kuryer ilovasida taom ostida ko'rinadi.
- **Bekor qilishda sabab majburiy**: admin panelda "Bekor qilish" modal ochadi
  (4 ta tayyor sabab + erkin matn). Holat select'idan `cancelled` tanlansa ham
  shu modal ochiladi — sababsiz bekor qilish yo'li yo'q.
  Sabab `order.cancelReason` da; mijoz uni `/order/{number}` sahifasida va
  profil tarixida ko'radi. Buyurtma qayta tiklansa sabab o'chiriladi.

### Xarita ilovalariga marshrut tugmalari
- `components/map/RouteButtons.tsx` — Yandex Maps / Google Maps / 2GIS uchun
  havolalar; telefonda o'rnatilgan ilovani ochadi, kompyuterda saytni.
  Boshlanish nuqtasi **ataylab bo'sh** — har ilova foydalanuvchi joylashuvini
  o'zi biladi. Koordinata tartibi: Yandex/Google `lat,lng`, 2GIS `lng,lat`.
- **2GIS formati**: `2gis.uz/directions/points/%7C{lng}%2C{lat}`. Eski
  `/routeSearch/rsType/car/to/...` yo'li **o'lgan** — 301 bilan shahar
  sahifasiga tashlaydi (marshrut tuzilmasligi shundan edi). `|` belgisi
  percent-encoded bo'lishi shart.
- Checkout'da "Olib ketish" tanlansa xarita ostida, buyurtma kuzatuvida ham
  (olib ketish buyurtmalari uchun) ko'rsatiladi.

### QR menyu (stol QR kodlari)
- `/admin/qr` — **QR kartochka generatori**: umumiy (restoran) yoki har bir stol
  uchun. Fon (4 ta tayyor ohang), sarlavha, kichik sarlavha, tavsif va pastki
  qator tahrirlanadi; stol kartochkasida **stol raqami katta belgi** bo'lib
  turadi.
- Kartochka **canvas'da chiziladi** (`components/admin/QrPoster.tsx`,
  1200×1700 — chop etishga yetarli). Ya'ni ekranda ko'ringan narsa aynan
  yuklanadi. QR kod oq "plastinka" ustida turadi — qora fonda ham skaner bo'ladi.
- **Yuklab olish**: bitta kartochka → PNG; **"Hammasini ZIP qilib olish"** →
  barcha stollar + umumiy kartochka bitta arxivda. ZIP `lib/zip.ts` da qo'lda
  yoziladi (store-only, kutubxonasiz — PNG allaqachon siqilgan). Sabab: har
  stolga alohida fayl yuborilsa brauzer "bir nechta faylni yuklashga ruxsat
  berasizmi?" deb so'raydi va rad javobida qolganini jimgina tashlab yuboradi.
- **Havola**: stol kartochkasi `"/menu?table=<tableId>"` ga, umumiysi shunchaki
  `/menu` ga olib boradi. Panel boshqa manzilda ochilgan bo'lsa (IP, tunnel),
  "Sayt manzili" maydonida haqiqiy domen yoziladi.
- **Stol konteksti** (`lib/table.tsx`): `?table=` ko'rilganda stol
  `sessionStorage` ga yoziladi (localStorage emas — bu bitta tashrifga tegishli,
  uyga borib yetkazib berish buyurtma qilganda eski stol yopishib qolmasligi
  kerak). Sayt tepasida "N-stoldasiz" chizig'i va "Men stolda emasman" tugmasi.
- **`dinein` buyurtma turi**: checkout'da uchinchi variant ("Stolga") faqat QR
  skaner qilinganda chiqadi va avtomatik tanlanadi; manzil ham, yetkazish
  narxi ham yo'q. Server `tableId` ni xaritadagi stollar bilan solishtiradi —
  noto'g'ri/eskirgan QR bo'lsa 400. Buyurtmaga `tableNumber` nusxa sifatida
  yoziladi (xarita qayta chizilsa ham chek to'g'ri o'qiladi).
- Holat oqimi: `dinein` ham `pickup` kabi **"Yo'lda" bosqichini o'tkazib
  yuboradi**. Statistikada alohida "Stolda (QR)" ko'rsatkichi bor.

### Stol bron qilish (bron tizimi)
- **Xaritani admin chizadi**: `/admin/settings` → "Stol bron qilish" →
  `components/admin/FloorPlanEditor.tsx`. Asbob tanlanadi (to'rtburchak stol /
  dumaloq stol / devor / zona), xaritada sudrab chiziladi; "Ko'chirish"
  rejimida stol sudralib joyi o'zgaradi, stol bosilsa raqami, joylar soni va
  o'lchami tahrirlanadi. Koordinatalar **plan birligida** (`width`×`height`),
  shuning uchun bir chizma telefonda ham, katta ekranda ham to'g'ri ko'rinadi.
- **Stol id'si barqaror**: bron `tableId` ga bog'lanadi, ya'ni stol raqami
  o'zgarsa yoki ko'chirilsa ham eski bronlar buzilmaydi (`tableNumber` chekka
  nusxa sifatida saqlanadi).
- **Mijoz** `/bron` sahifasida sana/vaqt/mehmonlar sonini tanlaydi → xaritada
  bo'sh stollar yashil, bandlari qizil, ishlatilmaydiganlari kulrang. Stol
  bosiladi, ism (+ izoh) yoziladi.
- **Bron uchun SMS tasdiq majburiy** (buyurtmadagidek): kirmagan mijoz
  `/login?next=/bron&reason=booking` ga yuboriladi. Raqam **profildagi
  tasdiqlangan raqamdan** olinadi — formadagi maydon faqat ko'rsatish uchun,
  server uni e'tiborga olmaydi. Sabab: javob bermaydigan raqamga saqlangan
  stol restoranga butun bir kechani yo'qotadi.
  **Panel orqali** (telefon qo'ng'irog'i bilan) qilingan bron bundan mustasno —
  u yerda operator raqamni o'zi yozadi.
- **Bandlikni server hal qiladi** (`internal/handlers/reservations.go`):
  `[at, endsAt)` yarim ochiq oraliq bo'yicha kesishish tekshiriladi va band
  bo'lsa **409** qaytadi — ikki mijoz bir vaqtda bir stolni bosishi mumkin,
  saytdagi rang esa oxirgi so'rov paytidagi holat, xolos. Bekor qilingan bron
  stolni darhol bo'shatadi.
- Boshqa qoidalar ham serverda: `enabled`, `minNoticeMinutes` (juda kech),
  `maxDaysAhead` (juda uzoq), `maxGuests`, stolning joy soni, stol faolligi.
  **Panel orqali qilingan bron** bu ikki vaqt qoidasidan ozod (telefonda "10
  daqiqadan keyin" normal holat).
- Mijoz o'z bronlarini **profilda** ko'radi (`/profile` → "Mening bronlarim"):
  raqam, holat, stol, vaqt, mehmonlar soni, izoh va bekor qilingan bo'lsa —
  restoranning sababi.
- **Panelda** `/admin/reservations`: kelayotgan / bugun / o'tgan / hammasi,
  qidiruv, holat tugmalari (Tasdiqlash → Mehmon keldi → Yakunlash, yoki
  sababi bilan Bekor qilish) va o'ng tomonda **tanlangan payt uchun xarita** —
  "soat 8 da 7-stol bo'shmi?" degan savolga ro'yxat javob bera olmaydi.

- **Xarita ko'rinishi sozlanadi** (`booking.hidePlan`): o'chirilsa mehmon faqat
  vaqt va necha kishiligini aytadi.
  ⚠️ **Tanlov yashiriladi, hisob-kitob emas**: server baribir **haqiqiy stol
  ajratadi** (mos keladigan **eng kichik bo'sh** stol), ya'ni ikki marta bron
  qilish imkonsizligicha qoladi va paneldagi xarita, "soat 8 da 7-stol bo'shmi",
  chekdagi stol raqami — hammasi o'zgarmaydi. Stolsiz bron ularning **hammasiga**
  ikkinchi turdagi bronni o'rgatishni talab qilardi.
  ⚠️ **Eng kichigi, birinchi topilgani emas**: ikki kishini o'n kishilik stolga
  o'tqazish — bir soatdan keyin kelgan o'n kishilik davrani rad etishning yo'li,
  va har bir bron alohida to'g'ri ko'rinadi.
  ⚠️ Maydon **"hide"** (ko'rsatish emas): nol qiymati hozirgi xatti-harakat
  bo'lishi shart — `showPlan` bo'lganda chiqqan kuni **hamma** restoranda stol
  tanlash o'chib qolardi (bo'sh `mapProvider` = 2GIS bilan bir qoida).
  Bo'sh stol qolmasa **409** — mehmon ekranida stol band bo'lib chiqqandagi
  javobning o'zi.
### Oldindan buyurtma (predzakaz)
- Mijoz checkout'da, operator esa call-markazda buyurtmani **keyingi vaqtga**
  bera oladi (`order.scheduledAt`). Bo'sh = oddiy buyurtma, ya'ni "hozir" —
  va bu maydon aynan shuning uchun `omitempty`.
- ⚠️ **Yangi holat ham, fon rejalashtiruvchisi ham yo'q.** Predzakaz
  `queuedAt` si **kelajakka** qo'yilgan holda yoziladi: o'sha maydon
  allaqachon "bu buyurtma qachondan oshxonaniki" degani, va uni o'qiydigan
  hamma narsa (KDS, qo'ng'iroq, sanoqlar) tayyor edi. Yagona o'zgarish —
  kelajakdagi vaqtni o'tgan deb o'qimaslik (`$lte: now`).
  Muqobili qimmat: vaqti kelganda buyurtmalarni ag'daradigan fon sikli —
  ikkinchi yozuvchi, unga qulf kerak, konteyner restartida to'xtaydi, va
  restoran buni **predzakaz umuman pishirilmagan kuni** biladi.
- **`scheduledAt` va'da, `queuedAt` mexanika**: birinchisi chekda, kuzatuv
  sahifasida va KDS kartochkasida turadi; ikkinchisi hech kimga
  ko'rsatilmaydi. Bittasi bilan ikkalasini ifodalash — vaqtni siljitganda
  mijozga aytilgan soatni ham siljitish demakdir.
- **Sozlama filialniki** (`branch.preorder`, §4 dagi `delivery` bilan bir
  sabab): `leadMinutes` — **butun xususiyat bir raqamda**, buyurtma shuncha
  vaqt qolganda ekranga chiqadi va qo'ng'iroq chalinadi. Egasi qo'yadi,
  chunki faqat u biladi (palovga bir soat, kofega o'n daqiqa). Yonida
  `minMinutes` (mijoz uchun eng erta), `maxDays`, `slotMinutes`.
  Bo'sh `enabled` — **o'chiq** (bo'sh `mapProvider` = 2GIS bilan bir qoida).
  `clampPreorder` **saqlashda** ishlaydi, o'qishda emas.
- ⚠️ **Ikki qo'ng'iroq, ikki hodisa** (`/admin/alerts` → `preorders`):
  *kelgani* — yangilik ("go'sht olish kerak"), *vaqti kelgani* — buyruq
  ("boshlang"), va u **soatlar keyin, hech kim hech nimaga tegmagan holda**
  keladi. Shuning uchun ikkinchisining o'z ovozi bor (to'rt nota almashib,
  1175/880) va bannerdagi havola **predzakaz tabiga** olib boradi: buyurtma
  oddiy ro'yxatning tepasida emas, u soatlar oldin berilgan.
  `$lte: now` bo'lmasa qo'ng'iroq **teskari** ishlardi.
- **Operator vaqt chegaralaridan ozod** (paneldagi bron bilan bir qoida):
  telefonda "yigirma daqiqadan keyin" ham, "to'yga" ham normal gap. Lekin
  **filialning kalitidan ozod emas** (bu forma validatsiyasi emas, eganing
  qarori) va o'tgan vaqtga ham yoza olmaydi.
- **Kartaga to'langan predzakaz o'z vaqtida navbatga tushadi**, bank javob
  bergan paytda emas (`afterPaid`) — aks holda naqd va karta ikki xil
  ishlardi va bu "oshxona ekrani o'zi biladi" bo'lib ko'rinardi.
- Mijozga slotlar **ish vaqtidan** quriladi (`PreorderPicker`): oddiy
  `datetime-local` yopiq kunni taklif qilib, hammasi to'ldirilgandan keyin
  rad etardi. Serverdagi tekshiruv baribir yakuniy (`resolvePreorder`).
- Panelda `/admin/orders` → "Oldindan" tabi: server **vaqt bo'yicha**
  saralaydi (`?scheduled=1`), chunki savol "nima oxirgi keldi" emas,
  "nima keyin pishiriladi". Nishon esa **hamma tabda** ko'rinadi — faol
  ro'yxatdagi predzakaz hech kim boshlamagan oddiy buyurtmaga o'xshaydi va
  kimdir uni erta pishirib "tuzatadi".
- Indeks **partial**, sparse emas: `branchId` hamma hujjatda bor, ya'ni
  sparse hech nimani tashlab ketmaydi.

### Yangi buyurtma/bron — ovozli bildirishnoma
- `components/admin/AlertBell.tsx`: **`AlertBell`** (kuzatuvchi) admin
  layout'da **bitta marta** ulanadi — har 15 soniyada `GET /admin/alerts`
  so'raladi va oxirgi ko'rilgan vaqt bilan solishtiriladi; yangisi kelsa ovoz
  chalinadi va pastki o'ng burchakda banner chiqadi. **`SoundToggle`** esa
  faqat tugma va u ikki joyda turadi (yon panel + telefondagi yuqori panel).
  Ular `admin-sound-change` hodisasi orqali gaplashadi.
  **Muhim**: kuzatuvchi ikki marta ulansa har xabar ikki marta chalinadi —
  shu sabab tugma va kuzatuvchi ajratilgan.
- Ovoz **WebAudio bilan sintez qilinadi** — alohida audio fayl yo'q, ya'ni
  mijoz VPS'ida 404 bo'lishi mumkin emas. **Buyurtma — ikki nota (880/1175),
  bron — uch nota (660/880/1320)**: zaldagi odam ekranga qaramay farqlaydi.
- ⚠️ **Kutayotgan ish to'xtovsiz jiringlaydi** (har 15 soniyada, poll bilan bir
  maromda). Bitta chalinish — eshitilishga bitta imkoniyat, va oshxona
  shovqinli, planshet narida, yonidagi odamning qo'li band. Buyurtma esa kimdir
  tasodifan qaramaguncha `pending` da yotardi — ya'ni bu komponent oldini olishi
  kerak bo'lgan nosozlik uning o'zi qoldirgan teshikdan kirardi.
- **Ikki signal, ikki to'xtatuvchi amal**: `orders.pending` — "Qabul qilish",
  `preorders.dueWaiting` — "Tayyorlashni boshlash" (yoki KDS'dagi "Boshlandi").
  ⚠️ Ikkinchisi **faqat `confirmed`** ni sanaydi: vaqti kelgan-u hali qabul
  qilinmagan predzakaz allaqachon birinchisida bor, va uni ikki marta sanash
  bitta buyurtma uchun **ikki ovoz** berardi — har biri boshqa tugma so'rab,
  lekin bosilgani ikkalasini ham o'chirmasdi.
- Banner ikkalasini **alohida nomlaydi** va yonida qaysi tugma jimlatishini
  yozadi: nima bosishni ayta olmaydigan signal — odam o'rganib e'tibor
  bermaydigan signal.
- ⚠️ **Takrorlash tezligi poll tezligidan alohida** (`ALARM_MS` 1 s,
  `POLL_MS` 15 s). Ilgari har pollda bir marta chalinardi, ya'ni chalinishlar
  orasi 15 soniya edi — bu signal emas, "vaqti-vaqti bilan keladigan
  bildirishnoma" bo'lib eshitiladi, va odam yonида bo'lgan yagona chalinish
  aynan o'tkazib yuborilgani bo'lardi. Ikki tezlik ikki savolga javob beradi:
  serverdan qanchalik tez-tez so'raymiz va qanchalik qat'iy aytamiz.
- Ikkalasi ham kutayotgan bo'lsa ovozlar **navbat bilan** chalinadi: bu tezlikda
  ustma-ust qo'yish shovqin, bittasini tashlab ketish esa uni "hech qachon
  eshitilmaydigan" shartga aylantiradi.
- **"Qabul qilish" bosilganda buyurtmalar sahifasi `admin-orders-changed`
  hodisasini otadi** va qo'ng'iroq darhol qayta so'raydi. Shartni server
  hisoblagani uchun aks holda ovoz keyingi pollgacha davom etardi — sekundiga
  bir chalinishda bu "tugma ishlamadi" bo'lib o'qiladi.

### Mehmonlar fikri saytda (`restaurant.reviews`)
- Sozlamalarda bitta tugma bo'limni **ochadi**, lekin nima ko'rinishini
  **`feedback.isPublic` bittalab** hal qiladi.
- ⚠️ **Bu ko'rsatish sozlamasi emas, rozilik chegarasi.** `feedback` dagi har
  bir yozuv mehmon **o'z kuzatuv sahifasida** "buyurtma qanday o'tdi?" degan
  savolga yozgan — restoranga yo'llangan shaxsiy xabar, ustiga ismi bilan.
  Tugma bosilishi bilan hammasini chiqarish internetга hech kim taklif
  qilmagan so'zlarni, jumladan jahl bilan yozilganlarini, **haqiqiy ismlar
  ostida** joylashtirardi.
- Shu sabab bittalab: bu — xususiyatning tasodifan "beshta yulduzli maqtovlar
  devori"ga aylana olmaydigan yagona ko'rinishi ham, chunki ega har birига
  qarashi shart.
- ⚠️ **"Faqat 4+ yulduz" sozlamasi yo'q va bo'lmasligi kerak**: bunday qoida
  bo'limni restoran o'zi haqida yig'gan maqtovga aylantiradi, va o'quvchi buni
  sezgan zahoti butun bo'limga ishonishni to'xtatadi.
- ⚠️ **O'rtacha baho barcha bahodan hisoblanadi**, faqat chiqarilganlardan
  emas: tanlangan fikrlarning o'rtachasi hech narsaning o'rtachasi emas, va u
  mehmon eng ishonadigan joyda turadi.
- Public javobda **alohida tor struktura** (`publicReview`), `models.Feedback`
  filtrlangan ro'yxati emas: modelda telefon, buyurtma raqami va shikoyat
  yuritish bor, va keyin qo'shiladigan maydon qo'lda kengaytiriladigan
  strukturadan o'z-o'zidan o'ta olmaydi. Ism — **faqat birinchi so'z**.
- ⚠️ **Filialsiz yozuvlar ham kiradi** (`branchId: null`): brendlar joriy
  etilishidan oldingi fikrlarda filial yo'q, va qat'iy moslik ularni tashlab,
  egani "tugma yoqilgan, saytda hech nima yo'q" holatida qoldirardi.
- Blok `DEFAULT_SECTIONS` da (menyudan keyin, manzildan oldin) va yoqilmagan
  bo'lsa **hech nima chizmaydi**. Go'даги `DefaultSections()` ga ataylab
  qo'shilmagan — u testda "bugungi sahifa" deb muhrlangan va konsol dizayni
  uchun urug'.
- ⚠️ **Takrorni server hisoblaydi** (`alerts.orders.pending`), brauzerdagi
  "ko'rdim" bayrog'i emas. Bu — "Qabul qilish" tugmasi bilan **bir xil fakt**,
  demak: bir qurilmada bosilgani hammasida ovozni to'xtatadi, ofisdagi ikkinchi
  panel jim qilib bo'lmaydigan ikkinchi signal emas, va yangilangan tab nima
  uchun jiringlayotganini unutmaydi. Sinxronlash kerak emas, chunki haqiqatning
  nusxasi bitta.
- **Yopish tugmasi yo'q, "5 daqiqaga jim" bor**: qabul qilinmagan buyurtma
  haqidagi signalni "yopish" — bir necha soniyada yolg'onga aylanadigan
  bayonot. Snooze esa halol va vaqt bilan chegaralangan (haqiqiy holat:
  operator aynan o'sha buyurtma bo'yicha telefonda). ⚠️ **Yangi buyurtma
  kelsa snooze bekor bo'ladi** — aks holda bir buyurtma uchun bosilgan tugma
  keyingisini yutib yuborardi.
- Ovoz **standart holatda yoqilgan**. Brauzer sahifa bilan muloqotdan oldin
  ovoz bermaydi, shuning uchun paneldagi **birinchi bosish** (istalgan joyda)
  audio'ni jimgina ochadi. 🔕 tugmasi faqat o'chirish uchun; tanlov
  localStorage'da saqlanadi.

### Admin ro'yxatlari: scroll bloki + pagination
- `components/admin/PagedList.tsx` — `usePaged()` (sahifalash), `<ListScroll>`
  (o'z ichida suriladigan blok, `max-h`) va `<Pager>` ("13–15 / 15", Oldingi /
  Keyingi). Har bir ro'yxat sahifa balandligini oshirmasligi kerak: ma'lumot
  ko'paygani sari sahifa cho'zilib, filtrlar ekrandan chiqib ketardi.
- Qo'llanilgan joylar: buyurtmalar (12/sahifa), foydalanuvchilar (25),
  kuryerlar (20), mijoz kartochkasidagi buyurtmalar va manzillar (10),
  kuryer kartochkasidagi tarix (10), adminlar, amallar jurnali (server
  tomonidan "ko'proq yuklash" bilan), menyu (har kategoriya bloki),
  kategoriyalar, tashqi xizmatlar.
- Jadval sarlavhalari `sticky top-0` — suriganda ustun nomlari ko'rinib turadi.

### Dashboard statistikasi (`GET /admin/stats`)
- `internal/handlers/adminstats.go` — davr `?from=&to=` (`YYYY-MM-DD`, `to`
  kiritilgan kun bilan; ikkisi ham bo'sh = butun tarix). Hisob **bazada**
  bajariladi: avval panel oxirgi 200 buyurtmani olib brauzerda qo'shardi, ya'ni
  200 dan oshgach ko'rsatkichlar noto'g'ri bo'lardi.
- Qaytadi: davr bo'yicha buyurtmalar (jami, yetkazilgan, bekor, yetkazish/olib
  ketish, holatlar kesimi), pul (tushum, o'rtacha buyurtma, yetkazish yig'imi,
  naqd), top 8 taom, hamda mijozlar (jami / yangi / **faol** = shu davrda
  buyurtma bergan / manzili bor), kuryerlar (jami, faol, ishda), adminlar
  (owner/manager) va menyu (taomlar, mavjud, kategoriyalar) sanoqlari.
- ⚠️ **Pul kelganda sanaladi, buyurtma tushganda emas** (`received()`).
  Ilgari bekor qilinmagan har bir buyurtma darhol "tushum" edi: hozirgina
  tushgan 100 000 so'mlik buyurtma — hali pishirilmagan, kuryer chiqmagan,
  hech kim to'lamagan — dashboardda pul bo'lib turardi. Raqam **kechroq
  to'g'ri bo'lib chiqardi va kun bo'yi noto'g'ri turardi**, ya'ni hech kim
  unga shubha qilmaydi, shunchaki noto'g'ri reja tuzadi.
  Pul ikki yo'l bilan keladi va buyurtma ikkalasini ham tashiydi:
  **`paymentStatus: paid`** (bank tasdiqladi — karta to'lovi ovqat
  qimirlashidan oldin ham haqiqiy) yoki **`delivered`** (kuryer pul bilan
  qaytdi; olib ketish va stolda uchun ham yakuniy holat).
  **Bekor qilingan hech qachon sanalmaydi, to'langan bo'lsa ham**: u qaytarib
  beriladigan pul.
- **"Kutilayotgan pul"** alohida ko'rsatkich — tushgan, bekor qilinmagan,
  hali olinmagan. Egaga kerak ("bugun yana qancha keladi"), lekin tushum
  emas, va uni tushum deb atash aynan yuqoridagi xato edi.
- **O'rtacha chek** olingan pulni **olingan buyurtmalar soniga** bo'ladi
  (`paid`), hammasiga emas — aks holda oshxona bandroq bo'lgan sari o'rtacha
  chek pasayib ketadi.
- **Eng ko'p sotilgan taomlar ataylab boshqa asosda**: u "nima sotilyapti"
  degan savolga javob beradi, va tasdiqlangan buyurtmadagi taom sotilgan —
  faqat pul hali qo'lga tegmagan. Uni faqat bekor qilish "sotilmagan" qiladi.
- Bekor qilingan buyurtma buyurtmalar soniga kiradi — sahifada shu izoh
  yozilgan.

### Buyurtma manzilini xaritada tuzatish (admin)
- Chekdagi **"Xaritadagi joy"** bo'limi (`components/admin/OrderAddressMap.tsx`):
  mijoz belgilagan nuqta xaritada ko'rinadi, zonalar ham chiziladi.
  `/admin/orders` da nuqtani **bosib ko'chirish** mumkin (mijoz adashib
  belgilagan bo'lsa) — matn `reverseGeocode` bilan avtomatik yangilanadi;
  `/admin/users/[id]` va kuryer kartochkasida faqat ko'rinadi (`readOnly`).
- Nuqta bezak emas: kuryer "Yetkazdim"ni faqat shu nuqtaga `arrivalRadiusM`
  metr yaqin turib bosa oladi — noto'g'ri metka buyurtmani yopishga to'sqinlik
  qiladi. Shu sababli tuzatish imkoniyati kerak.
- **Pul o'zgarmaydi**: `deliveryFee`/`total` mijoz bilan kelishilgani uchun
  qo'lda tuzatilgan nuqta narxni qayta hisoblab yubormaydi. Zona va masofa
  (`deliveryZone`, `distanceKm`) yangilanadi, va yangi nuqtaning **hozirgi
  narxi** javobda qaytadi — boshqa zonaga tushib qolsa panel ogohlantiradi.
- Har tuzatish jurnalga tushadi: `order.address`.

### Buyurtmalar oqimi (admin)
- Holat **qo'lda** o'zgaradi, lekin bir bosishda: har bir buyurtmada
  "keyingi qadam" tugmasi bor (Tasdiqlash → Tayyorlashni boshlash → Yo'lga
  chiqdi → Yetkazildi). Olib ketish buyurtmalari `on_the_way` ni o'tkazib
  yuboradi. Yonidagi select — faqat tuzatish uchun (orqaga qaytarish).
- Ro'yxat har 20 soniyada avtomatik yangilanadi, yangi buyurtmalar soni
  ko'rsatiladi; "Faol" filtri tugallanmagan buyurtmalarni ko'rsatadi.
- `statusHistory` har bir o'zgarish vaqtini yozadi → chekdagi "Vaqt jadvali"
  (shikoyat kelganda "qachon tasdiqlangan/yetkazilgan" savoliga javob).
- Kelajakda avtomatlashtirish mumkin (masalan N daqiqadan keyin avtomatik
  `confirmed`), lekin restoran real holatni o'zi bilgani uchun hozircha
  qo'lda — faqat bosish soni minimallashtirilgan.

### SMS provayderi (har restoran o'zi tanlaydi)
- To'rtta xizmat: **Eskiz**, **Play Mobile**, **getsms.uz**, **OneSignal**;
  beshinchisi — `demo` (SMS ketmaydi).
- **Provayder deploy sozlamasi emas, restoran sozlamasi**: har restoran o'z
  shartnomasini tuzadi va jo'natuvchi nomini moderatsiyadan o'tkazadi.
  Platformaning bitta umumiy hisobi **bitta restoranning moderatsiya muammosini
  qolganlarning loginini o'chirishga** aylantirardi. Panel:
  `/admin/settings` → "SMS provayderi" (`SmsEditor.tsx`).
- Kalitlar `sms_settings` da (§4). **Bo'sh parol = "saqlangani qolsin"**.
- **Yarim to'ldirilgan provayder `demo` ga tushadi**, xato bermaydi (sozlama
  saytning loginini o'ldirmasligi kerak), lekin panel **nima yuborayotganini**
  ko'rsatadi (`active` / `demo` / `missing`).
- **"Sinov SMS" — sahifaning asosiy tugmasi**: kalitlar to'g'ri ko'ringanda ham
  jo'natuvchi nomi moderatsiyadan o'tganmi va hisobda pul bormi — ikkalasi
  **birinchi mijoz kirmoqchi bo'lganda** bilinadi va restoran buni "sayt
  buzilgan" deb o'qiydi.
- **Kod xabarining matni sozlanadi** (`sms_settings.codeTemplate`, `{code}`
  o'rinbosari bilan). Konstanta emas, chunki qaror **restoranniki va pulini u
  to'laydi**: har biri o'z shlyuz shartnomasini tuzadi va o'z matnini
  moderatsiyaga beradi, Toshkentdagi biznes-lanch joyi va Namangandagi oilaviy
  oshxonaning mijozlari esa bir xil emas.
  - **Standart — lotincha o'zbekcha**, va ikkala yarmi ham sababli: o'zbekcha
    viloyatlarda ham tushuniladi (ruscha u yerda ancha zaifroq), sof lotincha
    esa **GSM-7** ichida qoladi — 160 belgi, kirillcha yoki bitta `oʻ` esa uni
    **70** ga tushiradi. ⚠️ Hozirgi 48 belgilik matn uchun narx **ikkalasida
    ham bitta bo'lak** — lotincha beradigan narsa **zaxira**, va uni ega
    restoran nomini qo'shib sarflaydi. Testda muhrlangan.
  - **Bo'sh qiymat — standart matn** (bo'sh `mapProvider` = 2GIS bilan bir
    qoida): bu maydon mavjud har bir installda yo'q.
  - ⚠️ **`{code}` siz shablon saqlanmaydi** (400). Busiz mijoz **kodsiz xabar**
    oladi: shlyuz qabul qiladi, SMS keladi, hech qayerda xato chiqmaydi va odam
    kira olmaydi. `smsCodeText`/`smsTemplateOf` da ikkinchi qo'riqchi ham bor —
    qo'lda yozilgan hujjat har bir loginni buza olmasligi kerak.
  - ⚠️ **Matnni o'zgartirish moderatsiyani bekor qiladi**, shuning uchun saqlash
    `lastTestOk` ni **nolga tushiradi**: shlyuz hech qachon ko'rmagan matn
    ustida turgan yashil belgi "tekshirilgan" deb yolg'on gapiradi, va ega buni
    mijoz kira olmaganda biladi.
  - Yonida **bo'lak sanog'i** (`smsParts`, kampaniyalardagi bilan bir hisob):
    alifbo chegarasi ko'rinmaydi, va xushmuomala qo'shilgan jumla har bir
    loginning narxini jimgina ikkilantirishi mumkin.
- ⚠️ **Mijoz shlyuzning so'zlarini hech qachon ko'rmaydi** (`errSMSSendFailed`).
  Ommaviy kod endpointi ilgari aynan shuni qaytarardi: moderatsiyadan
  o'tmagan Eskiz hisobida login formasiga raqamini yozgan odam
  `SMS yuborilmadi: eskiz send: 400 {"message":"Для теста можно…","id":"6bf8…"}`
  oladi. Bir vaqtning o'zida ikki xato — ko'rsatilayotgan odam uchun o'qib
  bo'lmaydi, **va** restoranning shlyuz holatini login formasiga yeta oladigan
  har kimga oshkor qiladi.
- ⚠️ **Shlyuz xatosi 503, kutish 429** (`smsRequestFailed`). Ilgari `issueCode`
  dan chiqqan **hamma narsa** 429 edi, ya'ni birinchi urinishdayoq rad etgan
  shlyuz uchun mehmonga "juda ko'p so'rov" deyilardi — hech qachon
  o'zgarmaydigan narsani kutishga chaqiruv.
- ⚠️ **Haqiqiy nosozlik panelga yoziladi** (`lastErrorAt`/`lastError`,
  `recordSMSFailure`). Bu `lastTest` dan **boshqa va muhimroq** savol: tugma
  "men tekshirganda ishladimi" ga javob beradi, bu esa "hozir mehmon kira
  olyaptimi" ga — va buni hech kim kuzatmaydi, chunki xato bo'lgan odam
  begona, u shikoyat qilmaydi, **shunchaki ketadi**. Muvaffaqiyatli sinov
  yozuvni tozalaydi: sahifa bir vaqtda ikki javob ko'rsatmasligi kerak, va
  eskirgani aynan qo'rqitadigani bo'lardi.
- ⚠️ **Sinov matni — haqiqiy shablonning o'zi**, unga *o'xshash* matn emas
  (`smsCodeText`, `smsTestText` = `smsCodeText(tmpl, "000000")`). Shlyuz **hisobni
  emas, aynan matnni** moderatsiya qiladi: Eskiz va Play Mobile bitta yozuvni
  tasdiqlaydi va qolganini rad etadi. Ilgari ikkita alohida satr yozilgan edi
  (sinov "Test: SMS sozlamalari tekshirilmoqda…", login esa "Tasdiqlash
  kodi: …"), ya'ni sinov **o'tishi** va haqiqiy kodlar **rad etilishi** mumkin
  edi — bu sahifa oldini olish uchun qurilgan yagona nosozlikni aynan
  tasdiqlangan deb ko'rsatgan bo'lardi. Testda muhrlangan.
- ⚠️ **Eskiz'ning yangi hisobi shablonsiz faqat uchta tayyor matnni qabul
  qiladi** ("Bu Eskiz dan test" va h.k.) va rad javobi **400 + ruscha JSON**
  bo'lib keladi. Bu **imzosiz signup**, buzuq integratsiya emas — shuning uchun
  `eskizNeedsModeration` uni tanib, nima qilish kerakligini yozadi va
  **moderatsiyaga beriladigan aniq matnni** ekranda nusxalanadigan qilib
  ko'rsatadi (qo'lda ko'chirilgan shablon ishonchli tarzda sayt
  yuboradigani bo'lmaydi).
- ⚠️ **Ikkinchi tugma — boshqa savol**: "Eskiz sinov matni bilan tekshirish"
  (`probe`) email/parol va tarmoq yo'lini isbotlaydi, kod xabari yetib
  borishini esa **isbotlamaydi**. Shuning uchun u ikkilamchi joyda turadi, javob
  yashil emas **sariq**, va **`lastTestOk` ga hech qachon `true` yozmaydi**:
  saqlangan bayroq sahifada kunlar davomida turadi va u "mijozning kodi
  keladimi" degan savolga javob beradi — probe esa bu savolni bermagan.
- ⚠️ **getsms.uz rad javobini 200 ichida yuboradi** (noto'g'ri parol,
  tasdiqlanmagan nickname, shartnomadan tashqaridagi raqam) →
  `error`/`error_text`/`error_no` tekshirilmasa sayt yuborilmagan kod haqida
  "yuborildi" deydi. OneSignal'da ham shu (200 + `errors`). Ikkalasi testda
  muhrlangan (`internal/sms/sms_test.go`).
- ⚠️ **Raqam formati har xil**: hamma shlyuz `998XXXXXXXXX`, OneSignal —
  **E.164** (`+998...`). Konversiya bitta joyda: `e164()`.
- OneSignal SMS'ni **Twilio orqali** yuboradi: qimmatroq va jo'natuvchi nomi
  mahalliy tasdiqlangan nom bo'lmaydi.
- Sender **keshlanadi** (`h.sender(ctx)`), kesh kaliti — `updatedAt`: Eskiz
  ~30 kunlik bearer token tutadi, har so'rovda qayta qurish har SMS uchun
  qaytadan login qilardi. `updatedAt` kalit bo'lgani uchun saqlash
  **restartsiz** yangi shlyuzga o'tadi.
- Muhit o'zgaruvchilari (`SMS_PROVIDER` …) faqat **zaxira**.
- **Sinov raqamlari** (`sms_settings.testPhones`, ≤5): demo rejimda faqat shu
  raqamlar uchun kod API javobida qaytariladi — ega shlyuz shartnomasidan oldin
  o'z kirishini sinashi uchun tor shakl; begona raqam 503 oladi. Panelda
  ogohlantirish bor: bu raqamlar bilan istalgan odam kira oladi.
- ⚠️ **Admin parolini tiklash bu ro'yxatga kirmaydi** (`smsUsable`,
  `smsUsableFor` emas): sayt logini eng yomon holatda bitta mijoz hisobini
  beradi, tiklash kodi esa **admin panelni** ochadi — va raqam restoranning o'z
  aloqa sahifasida yozilgan. Shlyuzsiz qolgan ega serverda tiklaydi
  (`cmd/adminreset`).

### Telegram bot va mini app (`internal/telegram`)
- **Har restoranga o'z boti va o'z tokeni** (`telegram_settings` — alohida
  kolleksiya, §4). Sizib chiqqan bot tokeni "birovning hisobidagi pul" emas:
  bu mini app'ni ochgan **har bir mehmonga** restoran nomidan yozish imkoniyati.
- **Imzo — bu kirish.** Mini app "bu 12345-foydalanuvchi" degan satrni
  brauzerdan yuboradi va u butunlay klient nazoratida. `telegram.Verify`: kalit
  `HMAC_SHA256("WebAppData", token)` (tokenning o'zi emas), imzolangan satr —
  `hash` dan boshqa barcha maydonlar **kalit bo'yicha saralangan** holda,
  solishtirish `subtle.ConstantTimeCompare` bilan.
- ⚠️ **Saralash bezak emas**: Go'da map tartibi tasodifiy — saralanmagan
  tekshiruv loginlarni **ba'zan** yiqitardi. Test 20 urinishda muhrlaydi.
- ⚠️ **`auth_date` majburiy** (24 soat): busiz qo'lga tushgan `initData` o'sha
  hisob uchun **abadiy parol**. Sanasiz payload rad etiladi.
- ⚠️ **Telegram telefon raqamini bermaydi**, faqat id va ism → javobda
  `needsPhone`, va mini app uni **checkout'dan oldin** so'raydi (buni
  "tasdiqlash" qadamida bilib olgan ilova allaqachon yutgan buyurtmani
  yo'qotardi). Telegram bergan raqam SMS kodidan **kuchliroq dalil** (hisobdagi
  raqam haqidagi imzolangan bayonot) va SMS oqimining o'rniga qabul qilinadi.
  Raqam boshqa hisobda band bo'lsa **409** — hisoblarni jimgina birlashtirish
  buyurtmalarni, ballarni va manzillarni ko'chiradi.
- Panelda: bo'sh token = saqlangani qoladi; **asosiy tugma — "Ulanishni
  tekshirish"**, chunki o'lgan botning tokeni ishlaydiganidan faqat birinchi
  mehmon kirmoqchi bo'lganda farq qiladi. Tekshiruv bot **nomini** to'ldiradi
  va mini app havolasi shundan quriladi (qo'lda yozilgan nom birovning botini
  ochadi).
- **Mini app muhiti** (`lib/telegram.tsx` + `components/site/TelegramApp.tsx`):
  - **SDK skripti faqat Telegram ichida yuklanadi** (launch URL'dagi
    `tgWebApp…` parametrlaridan aniqlanadi) — aks holda har veb tashrifchiga
    ular ishlatmaydigan xususiyat uchun so'rov qo'shilardi.
  - ⚠️ **`100vh` Telegram ichida yolg'on gapiradi**: halol raqam
    `viewportStableHeight`, `--tg-viewport` CSS o'zgaruvchisi bo'lib beriladi va
    **`viewportChanged` da qayta yoziladi** (aks holda faqat birinchi
    bo'yashda to'g'ri bo'ladi).
  - Kirish **avtomatik**: `initData` serverga o'sha holida ketadi. Ikki marta
    ishlamasligi `useRef` bilan qo'riqlangan (React dev rejimi effektni ikki
    marta ishga tushiradi). Sessiya `useUser()` ga uzatiladi — **ikkinchi
    saqlash joyi yaratilmaydi**.
  - Telegramning **"orqaga" tugmasi** ulanadi; yopishdan oldin tasdiq **faqat
    savat bo'sh bo'lmaganda** (bo'sh savatdagi tasdiq odamlarni uni e'tibordan
    qoldirishga o'rgatadi).
  - Skript yuklanmasa yoki bot ulanmagan bo'lsa **sayt oddiy sayt bo'lib
    ishlaydi** — mini app aynan shu sayt.
- ⚠️ **`TelegramProvider` sahifalarni O'RAB turishi shart.** `(site)/layout.tsx`
  da u `<TelegramApp />` bo'lib **o'z-o'zini yopgan** holda, sahifalarning
  **qo'shnisi** sifatida chizilgan edi — ya'ni `{children}` provayderdan
  tashqarida qolgan va checkout, `CallLink`, `CookieNotice` dagi
  `useTelegram()` **standart kontekstni** o'qigan (`inTelegram: false`).
  Hech nima buzuqqa o'xshamasdi: ko'prikning o'zi ishlardi (u o'z provayderi
  ichida), mijoz kirardi, "orqaga" tugmasi ishlardi. Yagona ko'rinadigan
  oqibat — **har bir mini app buyurtmasi `channel: "web"` bo'lib yozilardi**,
  ya'ni bot uchun pul to'layotgan restoranga "uni hech kim ishlatmayapti"
  deyilardi. Aynan shu maydon javob berishi kerak bo'lgan savolning teskarisi.
- ⚠️ **Aniqlash tashrifga yopishtiriladi** (`sessionStorage`, `tg_miniapp`).
  `tgWebApp…` parametrlari faqat **birinchi manzilda** bo'ladi: mehmon menyuga,
  savatga, checkoutga o'tishi bilan hash yo'qoladi, va o'sha sahifada
  yangilangan ilova bo'sh URL'ga qarab qoladi. SDK tekshiruvi ham qutqarmaydi —
  skript aniqlash **ijobiy** bo'lgandan keyin yuklanadi. Qolgani user-agent
  regexi, ya'ni javob "ba'zan" bo'ladi: Telegramning Android webview'i o'zini
  odatda nomlaydi, iOS'niki odatda yo'q — va bu eng yomon turdagi xato, chunki
  qo'lingizdagi telefonda ishlaydi. `localStorage` emas, `sessionStorage`:
  bu bitta tashrif haqidagi fakt (stol konteksti bilan bir qoida).
- **Buyurtma holati bot orqali** (`handlers/notify.go`) — ⚠️ **pul tejaydigan
  band**: har bir bunday xabar aks holda restoran to'laydigan SMS bo'lardi.
  - **Faqat mijoz harakat qiladigan o'zgarishlar**: `confirmed`, `on_the_way`,
    `delivered`, `cancelled`. `preparing` **ataylab jim** — beshta ping olgan
    mijoz eng muhimini o'qishni to'xtatadi.
  - **Mijozning tilida** (`notifyLang`, §"Mini app'da til").
  - **Holat o'zgarishini hech qachon yiqitmaydi**: bloklangan bot yoki
    Telegramning o'zi ishlamasligi buyurtmaning muammosi emas — fonda
    yuboriladi, xato logga tushadi.
  - Matn **oddiy tekst**: Markdown rejimida taom nomidagi qochirilmagan pastki
    chiziq **butun xabarni** rad ettiradi.
- **Deep link**: `t.me/<bot>/app?startapp=t_<tableId>-b_<branchId>`.
  ⚠️ Telegram `startapp` da faqat `A-Za-z0-9_-` ni qabul qiladi → stol query
  string bilan uzatilmaydi. Yechilgan qiymat **hex id sifatida tekshiriladi**
  (havolani istalgan odam yozishi mumkin) va saytning o'z `?table=`
  mexanizmiga aylantiriladi — ikkinchi manba o'rgatilmaydi.
  QR sahifasidagi "havola qayerga olib boradi" tanlovi **faqat bot ulangan va
  tekshirilgan bo'lsa** chiqadi: kartochkalar chop etiladi.
- ⚠️ `ParseContact` maydon nomlari **jonli bot bilan tekshirilmagan** (ikki
  ko'rinishda bardoshli o'qiladi). Birinchi haqiqiy mijozda ko'rish kerak.

### Mijoz auth (telefon + SMS)
- Asosiy usul — **telefon raqam + bir martalik SMS kod** (`internal/sms`).
- ⚠️ **Demo rejim kodni API javobida qaytarishi mumkin, va bu faqat lokal
  ishlab chiqish uchun.** `SMS_DEMO_EXPOSE_CODE=1` bo'lgandagina qaytariladi;
  standart holatda **o'chiq**, va tenant konteynerlariga bu o'zgaruvchi umuman
  berilmaydi.
  Sababi: shlyuz sozlanmagan install demo sender'ga tushadi, demo sender esa
  kodni javobda beradi — ya'ni yangi ochilgan restoran saytida **istalgan odam
  istalgan raqam bilan kira olardi**: begona raqam uchun kod so'rab, uni
  JSON'dan o'qib, o'sha odam bo'lib kirish. Hech qayerda xato chiqmasdi:
  so'rov muvaffaqiyatli, javob to'g'ri shaklda, faqat JSON'ni o'qigan odam
  ko'radi.
- **Shlyuz yo'q = login umuman yo'q**, kod so'rash 503 bilan rad etiladi
  ("SMS xizmati hali sozlanmagan"). Bu to'g'ri nosozlik: "SMS sozlanmaguncha
  hech kim kira olmaydi" — qo'llab-quvvatlash qo'ng'irog'i, "istalgan odam
  istalgan bo'lib kira oladi" esa qaytarib bo'lmaydigan holat. Rad etish kod
  yaratilishidan **oldin** bo'ladi, va mehmon nima qilishni biladi — jimgina
  qabul qilinsa, u hech qachon kelmaydigan SMS'ni kutib o'tirardi.
- Qaror sof funksiyada (`exposeDemoCode`) va testda muhrlangan: haqiqiy shlyuz
  bayroqdan qat'i nazar kodni hech qachon qaytarmaydi.
- Kodlar `phone_code` kolleksiyasida **bcrypt hash** ko'rinishida, 3 daqiqa
  amal qiladi, 60 soniya cooldown, 5 ta noto'g'ri urinishdan keyin bekor.
- **Buyurtma berish uchun login majburiy**: savatdagi tugma login sahifasiga
  yuboradi (`/login?next=/checkout&reason=order`), checkout ham himoyalangan.
- Profilda mijoz ismi va manzillarini tahrirlaydi; raqamni o'zgartirish
  faqat yangi raqamga kelgan SMS kod bilan tasdiqlanadi.

### Admin parolini tiklash
- Kirish sahifasida "Parolni unutdingizmi?" — hisobga biriktirilgan raqamga
  SMS kod (`/admin/password/forgot` → `/admin/password/reset`, ikkalasi ham
  public: kira olmayotgan odam uchun boshqa yo'l yo'q).
- **Tiklash raqami** panelda qo'yiladi (Hisobim → `RecoveryPhone`) va SMS bilan
  tasdiqlanadi. Seed qilingan birinchi owner'da raqam yo'q — mijozga
  topshirishdan oldin qo'shish shart (DEPLOY.md).
- **Kodlar maqsad bo'yicha ajratilgan** (`phone_code.purpose`:
  `login` / `admin_reset` / `admin_phone`). Sabab: restoran egasi ko'pincha
  saytning ham mijozi, bir xil raqam bilan — mijozning login kodi admin
  panelini ocha olmasligi kerak. `(phone, purpose)` unique, `expiresAt` TTL.
- Zaxira yo'l — serverda `cmd/adminreset` (raqam yo'q yoki telefon yo'qolgan).

### Server xabarlari ham uch tilda (`internal/i18n`)
Panel, kassa, zal va ishchi ilovalari uch tilda edi, **xatolar esa yo'q**. Rus
kassir kassadagi har bir yorliqni o'qiy olardi, keyin smena ochilmaganda unga
«ochiq smena yo'q» deyilardi — peshtaxtada, navbat o'rtasida, mehmon oldida.
Ekranlar tarjima qilingan edi; **faqat nimadir buzilganda chiqadigan** yagona
jumla esa yo'q.

- ⚠️ **Til javob yozuvchisida saqlanadi, kontekstda emas.** `httpx.Error(w,
  status, msg)` so'rovni olmaydi va **1295 marta** chaqiriladi — imzosini
  o'zgartirish har bir joyga tegish demakdir. Tilini ayta oladigan yozuvchi
  xabarni **yoziladigan yagona joyda** tarjima qilishga imkon beradi va bironta
  ham chaqiruv joyi o'zgarmaydi.
- ⚠️ **Kalit — o'zbekcha xabarning o'zi.** Muqobili — har xabar yoniga kod
  qo'yish, ya'ni 1295 tahrir, va bittasi o'tkazib yuborilsa **jimgina**
  o'zbekcha qoladi.
- ⚠️ **Buning ochiq xavfini test qo'riqlaydi**: handlerdagi xabarni qayta
  yozish tarjimasini jimgina uzib qo'yardi. `errors_test.go` `internal/handlers`
  dagi **har bir literalni** o'qiydi va lug'atda yo'qini yiqitadi. Teskarisi
  ham: yuborilmaydigan tarjima — qayta yozilgan xabarning arvohi, va u sanoqni
  haqiqatdan sog'lomroq ko'rsatadi.
- ⚠️ **Middleware eng oxirida qo'shiladi.** Chi qo'shilish tartibida o'raydi,
  ya'ni handler **eng ichkarigi**ni ko'radi: `chimw.Logger` dan oldin qo'yilsa
  bizning o'ram uning tagida ko'milib qoladi, `httpx.Error` dagi tekshiruv hech
  nima topmaydi va **har bir xabar jimgina o'zbekcha qoladi** — aynan
  tuzatilayotgan xato, hech qanday yangi alomatsiz. Buni router qatorining
  o'zi emas, `lang_test.go` ushlaydi.
- **Hammasi tarjima qilinmaydi va tarjima qilinmagani ro'yxatda turadi**
  (`Untranslated`): «invalid id», «forbidden», «bad request» — bular odamga
  emas, **buzuq so'rovga** javob va ular tarmoq panelida so'rovni yozgan odam
  tomonidan o'qiladi. Ro'yxat «ataylab qoldirildi» ni «hali qilinmadi» dan
  ajratadi — testning butun qiymati shu farqda.
- ⚠️ **Xato bo'lmagan jumlalar ham bor**: `permissionName` — kassa PIN
  paneli yonida ko'rsatadigan qator («Chegirma berish uchun ruxsat kerak»). U
  javobning **maydonida** ketadi, ya'ni `httpx.Error` dan o'tmaydi — shuning
  uchun atrofidagi hamma yorliq tarjima qilingandan keyin ham ruscha kassada
  o'zbekcha qolgan edi. `httpx.LangOf(w)` shu ikkinchi yo'l, va u **ham
  yozuvchidan** o'qiydi: bitta savolning ikki manbasi bo'lsa, ular
  kelishmay qoladi.
- **Sarlavha ustun, keyin cookie, oxirida `Accept-Language`**: Windows'i
  ruscha bo'lgan va kassani o'zbekchaga qo'ygan kassir o'zi tanlaganini oladi.
  Sarlavha (`X-Keel-Lang`) esa cookie'dan ham ustun — pastdagi to'rtinchi
  teshikka qarang.
- ⚠️ **Tarjimasi yo'q xabar o'zbekcha qoladi** — bu eski xatti-harakat: o'quvchi
  tarjimani yo'qotadi, jumlani emas.

**Ikkinchi to'lqin: qo'riqchi ko'rmagan uch yuz xabar.** Panelda ba'zi
bildirishnomalar o'zbekcha qolgani xabar qilindi, garchi test yashil bo'lsa
ham. Ikki teshik topildi, va ikkalasi ham **jimgina** edi:
- ⚠️ **Qo'riqchi qatorni o'qirdi, daraxtni emas.** Naqsh
  `httpx.Error(w, …, "…")` ni **bitta qatorda** izlardi, gofmt esa uzun jumlani
  keyingi qatorga tashlaydi — ya'ni test **eng uzun 28 ta xabarni** umuman
  ko'rmagan. Aynan uzunlari nimadir tushuntiradiganlari. **Tahlil qila
  olmaganini jimgina o'tkazib yuboradigan qo'riqchi — hamma ishonadigan
  qo'riqchi.** Endi skaner `go/ast` bilan yuradi.
- ⚠️ **Xabarlarning ko'pi handlerda tug'ilmaydi.** `httpx.Error(w, …,
  err.Error())` **790 marta** yozilgan: jumla pastdagi qatlamda
  (`errors.New`, `fmt.Errorf`) yoziladi va yozuvchiga tayyor qator bo'lib
  keladi. Shuning uchun skaner endi `internal/` ning **hamma paketini** o'qiydi
  — `pos/`, `printer/`, `telegram/`, `menuimport/` va boshqalar.
- **Qiymat ichida bo'lgan xabar uchun naqsh** (`internal/i18n/patterns.go`):
  kalitda `%s` / `%d` bo'lishi mumkin, tayyor qator shu naqshga solishtiriladi
  va ushlangan qiymatlar tarjimaga **o'sha tartibda** qo'yiladi. Qiymatning
  o'zi tarjima qilinmaydi — u taom nomi yoki stol raqami.
- ⚠️ **Deyarli butunlay verbdan iborat kalit hamma narsani ushlaydi.** Shuning
  uchun naqshlar **uzun literali borilari birinchi** sinaladi va qisqasi faqat
  **tashuvchi** bo'lsa ruxsat etiladi: `"telegram: %s"`, `"%s: %s"` — ularning
  ru/en tarjimasi kalitning **aynan o'zi**, ya'ni ular tarjima emas, **shakl**.
  Ushlangan qiymat ham `Localize` dan o'tadi: `fmt.Errorf("%w: %s", …)` ichidagi
  haqiqiy jumla shunda topiladi.
- **Tarjima verblarni qayta tartiblay olmaydi** (`patterns_test.go`): ruscha
  so'z tartibi boshqacha va `%s` ni `%d` dan keyinga ko'chirish tabiiy
  ko'rinadi — natijada chek raqami turadigan joyga stol raqami chiqadi va
  jumlada hech nima g'alati ko'rinmaydi.
- **Ro'yxatdagi 44 ta yangi yozuv** — mashinaga javob: to'lov callback'lari
  (`order is cancelled`), SMS shlyuzining javob kodlari, `webpush:` xatolari.
  Ular tarmoq panelida o'qiladi.

**Panelning o'zida ham uchta jumla lug'atdan tashqarida edi** va uchalasining
ham lug'atda **aynan shu ma'nodagi kaliti bor edi** — jumla tarjima qilingan,
keyin bir necha fayl narida qo'lda qaytadan yozilgan. ⚠️ **Yorliq emas,
bildirishnoma qochadi**: tugma yozuvi yigirmata boshqa yozuv yonida, `t.` ga
qarab turib yoziladi; "nega ishlamadi" degan qator esa keyinroq, tuzatish
o'rtasida, hech kim tegmaydigan `if` shoxida paydo bo'ladi. Sozlamalar
sahifasidagi «Restoran ma'lumotini yuklab bo'lmadi.» aynan shunday edi va
ruscha ega uni **panel ochilmagan kuni** ko'rardi. Qo'riqchisi —
`frontend/src/lib/i18n/hardcoded.test.ts` (so'zlar ro'yxati, til aniqlagich
emas: fe'llar va holatlar, otlar emas — "menyu", "kassa", "filial" ruscha
panelda ham shunday yoziladi).

**Uchinchi teshik: xabar xato emas, javobning maydonida ketadi** (`httpx.T`).
Ulanish tekshiruvi **yiqilmaydi** — «domen hali bu serverga yo'naltirilmagan»
aynan shu tugma beradigan javob, ya'ni u `httpx.JSON` bilan yoziladi, `JSON`
esa hech nimani tarjima qilmaydi. Panelning har bir tekshiruvi shunday edi:
POS, ATS, Telegram, fiskal kassa, domen, menyu importi, kassalar limiti.
- `httpx.T(w, msg)` — `Error` dagi bilan **bir xil katalog va bir xil
  yozuvchi**, farqi shundaki jumla maydonda ketadi (`permissionName` shu
  yo'lning birinchi mijozi edi).
- Qo'riqchisi `TestFieldMessagesGoThroughT`: `httpx.JSON` ichidagi
  `message`/`error`/`hint`/`warning` maydonining qiymati literal yoki
  `err.Error()` bo'lsa, u `httpx.T` dan o'tishi shart. **`note` va `reason`
  ataylab yo'q** — ularni restoranning o'zi yozgan, tarjima qilish = egani
  qayta yozish.
- To'lov callback'lari (`payme`, `click`, `uzum`, `atmos`) ro'yxatdan chetda:
  u maydonni **provayder** o'qiydi, va uning so'zlari protokolning bir qismi.
- ⚠️ **Bitta joyda tarjima yozuvchisiz qurilardi** (`tillCapReached` javob
  tanasini qaytaradi, `w` si yo'q) — jumla shu sababli **yozilayotgan joyda**
  tarjima qilinadi: tilni biladigan narsa yozuvchi, tanani quruvchi emas.

**To'rtinchi teshik: til brauzerdan umuman chiqmasdi** (`X-Keel-Lang`).
Yuqoridagi uch teshik ham *serverdagi* jumlaning tarjimasi haqida edi. To'rtinchisi
undan oldinroq: server **kim o'qiyotganini bilmasdi**. Til `lang` cookie'sida
yozilgan, panel esa API'ga **boshqa origin** orqali murojaat qiladi
(dev'da `localhost:3000` → `localhost:8080`; telefon ilovasida esa umuman
cookie yo'q), va cross-origin `fetch` **hech qanday cookie yubormaydi**. Ya'ni
`middleware.Lang` doim zaxiraga tushardi va har javob o'zbekcha bo'lardi.
- **Alomati eng yomon turdan**: hech nima yiqilmaydi, log'da hech nima yo'q,
  ekran to'liq chizilgan. Noto'g'ri tildagi brifing to'g'ri tildagi brifingga
  **aynan o'xshaydi**, faqat o'qiyotgan odam uni o'qiy olmaydi.
- **Eng ko'zga tashlangani ertalabki brifing** edi (`/admin/insights`): uning
  butun mahsuloti — modeldan kelgan **jumlalar**, ya'ni til noto'g'ri bo'lsa
  xususiyatdan hech nima qolmaydi. Lekin yolg'iz emasdi: kampaniya matni,
  `?lang=` qo'yishni unutgan hisobotlar va **har bir xato xabari** ham shu
  yo'ldan yurardi.
- ⚠️ **Har so'rovda yuboriladigan sarlavha, unutilishi mumkin bo'lgan parametr
  emas.** Hisobotlar `?lang=` ni so'rovga qo'yardi va bu **har bir yangi
  chaqiruv joyi eslab qolgunicha** ishlardi — eslamaganlari (brifing, kampaniya
  yozuvchisi) jimgina, o'zbekchani ravon o'qiydigan odam yozgan kodda buzilardi.
  Endi `X-Keel-Lang` `request()` da bir marta qo'yiladi va handler javobni
  **so'rovdan** so'raydi, chaqiruvchidan emas.
- ⚠️ **CORS ro'yxatiga qo'shilmagan sarlavha umuman jo'natilmaydi** — brauzer
  uni so'rov ketishidan oldin tashlab yuboradi va hech qayerda xato chiqmaydi.
  Ya'ni `router.go` dagi `AllowedHeaders` shu tuzatishning yarmi.
- **Beshta klient, bitta mexanizm**: sayt/panel (`LangProvider` → `setApiLang`),
  Expo ilovalari (`prefs.ts`), va native ilovalar (`KeelApi.prepare` tilni
  `Prefs` yozadigan **o'sha kalitdan** o'qiydi — startda o'rnatilgan maydon
  sozlamalar ekranida almashtirilganda yangilanishi unutiladigan maydondir).
  Mehmon ilovasi `Accept-Language` bilan qolgan: u zaxira pog'onasi va ishlaydi.

**Kassa ekranidagi to'rt jumla brauzerning o'zida tug'iladi** (`lib/fiscal.ts`:
manzil noto'g'ri, brauzer bloklagan, javob kechikdi, ulanib bo'lmadi) — server
ularni faqat qavs ichida qaytaradi. Modul React'dan tashqarida, ya'ni lug'atni
o'zi so'ray olmaydi: jumlalar `t.fiscal` dan **parametr** sifatida beriladi
(`FiscalWords`). Xuddi shu sababdan `STATUS_LABEL` (`lib/orderStatus.ts`)
o'chirildi — kuryer ilovasi uch tilli, xarita esa faqat o'zbekcha edi va
`t.status[…]` allaqachon uchchalasini biladi.

### Xavfsizlik: filial qamrovi bitta obyektli amallarda ham
- ⚠️ **`RequireRole` — bu faqat "qaysidir owner/manager", "qaysi filial" emas.**
  `clampToAdmin` faqat **ro'yxatlarni** qisqartiradi; `_id` bo'yicha bitta
  hujjat oladigan handler esa filtrsiz qolsa, Chilonzor menejeri id yozib
  Yunusobod buyurtmasini o'qiydi, bekor qiladi, manzilini o'zgartiradi yoki
  kuryerini biriktiradi. Qoida KDS'dagi bilan bir xil: **filtr ichida doim
  `branchId`**, `_id` yolg'iz hech qachon hujjat tanlamaydi. `scopedOrderFilter`
  buni buyurtma, fikr uchun beradi; kuryer uchun `courierInScope`; qamrovdan
  tashqarisi **404** (403 emas — menejer boshqa filial hujjati borligini ham
  bilmasligi kerak).
- ⚠️ **`GET /admin/lookup?phone=` eng katta sizib chiqish edi**: telefon raqami
  bo'yicha butun kompaniyaning buyurtmalari, manzillari, bronlari, shikoyatlari
  va qo'ng'iroqlarini qaytarardi — filialsiz. Endi `orderScope` bilan har bir
  operativ ro'yxat filialga qisqartiriladi. Mijoz **profili** kompaniyaniki
  bo'lib qoladi (baza umumiy), lekin **buyurtma tarixi** filialniki.
- Bu CRM'dagi `tenantScope` bilan bir naqsh: chegara **filtr**, tekshiruv emas —
  o'qib bo'lib qirqiladigan ro'yxat yoniga `count`/agregat qo'shilsa sizib
  chiqadi, filtr esa boshidan qo'riqlaydi.

### Xavfsizlik: tezlik chegarasi (rate limit)
- `middleware/ratelimit.go` — IP bo'yicha, xotira-ichi (bir tenant = bir
  konteyner, ya'ni bitta jarayon; Redis kabi umumiy do'kon bu yerda ortiqcha
  bog'liqlik). Ikki gate: `smsGate` (5/daqiqa — har chaqiruv **pul sarflaydi**),
  `authGate` (10/daqiqa — bcrypt **CPU sarflaydi**, konteyner 1 yadroga
  cheklangan). SMS so'rovi, login (admin/kuryer/ishchi), parol tiklashda.
- ⚠️ Bu hisob darajasidagi qo'riqchilarni (SMS cooldown, kod urinishlari)
  **almashtirmaydi**: ular bitta qurbonni himoya qiladi, bu — serverni. Raqamdan
  raqamga yurib SMS billing hujumi va parol/CPU hujumi aynan IP bo'yicha
  ushlanadi.

### Xavfsizlik: JWT_SECRET va buyurtma raqami
- ⚠️ **Standart JWT kaliti bilan server ko'tarilmaydi** (`config.Validate`,
  `cmd/server` boot'da tekshiradi). Bashorat qilinadigan imzo kaliti — to'liq
  autentifikatsiya chetlab o'tish: kimdir kalitni bilsa istalgan hisob uchun
  owner tokeni yasaydi. Keel tenantlari `crypto/rand` 32 bayt oladi; bu tekshiruv
  qo'lda ko'tarilgan yoki `.env` unutilgan install'ni ushlaydi.
- ⚠️ **Buyurtma raqami `crypto/rand` dan** (`orderNumber`). `GET /orders/{number}`
  ochiq va to'liq manzil, xarita nuqtasi, kuryer telefonini qaytaradi — "havola =
  kalit" naqshi, lekin kalit taxmin qilinmasligi shart. Eski raqamning ikki
  belgisi `time.Now().Unix()%100` (soatdan) va qolgani urug'lanmagan
  `math/rand` edi — begonaning manzilini kalitsiz o'qish mumkin edi. Endi
  ~30^8 ≈ 6.5e11, adashtiruvchi belgilarsiz (I/O/0/1 yo'q).
- **Ma'lum cheklov (ochiq teshik emas)**: admin tokeni 7 kun, bekor qilish
  mexanizmi yo'q — parol o'zgartirilsa ham eski token ishlaydi. Tuzatish uchun
  hisobda `tokenVersion` + middleware tekshiruvi kerak; alohida ish.

### Xavfsizlik: chekkadagi javob header'lari
- ⚠️ **Header'lar chekkada (Caddy), har bir sayt blokida** — `control/internal/caddy`
  render'ida `securityHeaders`, `encode` bilan bir naqsh. Caddy'ning global
  bloki javobga tegadigan direktiv qabul qilmaydi, ya'ni oxirgi qo'shilgan
  tenant header'siz qolmasligi uchun bitta helper hamma blokka yoziladi.
  Nosozligi **jimgina**: brauzer yuborilmagan himoyani majburlamaydi.
- **Nima qo'yildi va nega:** HSTS (`max-age`, `includeSubDomains` va `preload`
  YO'Q — tenant o'z domenini olib keladi, uning subdomenlarini da'vo qilish yoki
  ketganidan keyin domenni bizga qulflash bizniki emas); `nosniff`;
  `Referrer-Policy`; `-Server` (versiya bannerini o'chirish).
- ⚠️ **Clickjacking — `X-Frame-Options: DENY` emas, `frame-ancestors`.**
  Telegram mini app — bu **Telegram ichida ochilgan saytning o'zi**
  (`channelreport.go`), ya'ni u saytni web.telegram.org da iframe qiladi. DENY
  har bir mini app'ni buzardi — skanerlarda "to'g'ri" ko'rinadigan, lekin
  funksiyani o'ldiradigan "hardening"ning aynan o'zi. `frame-ancestors` faqat
  joylashtirishga tegadi, skript/stilga emas — sahifani bo'shatib qo'ymaydi.
- ⚠️ **To'liq CSP (`script-src`) ataylab QO'YILMADI.** O'g'irlangan token XSS'ini
  haqiqatan to'xtatadigan narsa shu, va aynan u Next.js'ni jimgina buzadi: Next
  inline hidratsiya skriptlari, xarita SDK'lari (2GIS/Yandex/Google) o'zinikini
  yuklaydi — ko'r-ko'rona `script-src 'self'` saytni konsoldagina ko'rinadigan
  xato bilan bo'shatadi. U nonce'li, sinovdan o'tgan chiqarishni talab qiladi —
  header qatorida emas. Admin tokeni `localStorage` da (bilib qilingan tanlov);
  CSP uni himoyalaydigan yagona nazorat va u alohida ish bo'lib qoladi.

### Xavfsizlik: to'lov callback'lari va regex qidiruv (2026-09-07 auditi)
- ⚠️ **Pul ko'chiradigan callback imzosi constant-time solishtiriladi.** Payme
  Basic auth allaqachon shunday edi; Click MD5 imzosi `strings.EqualFold` bilan
  solishtirilardi — belgi-baholab, birinchi farqda qaytadigan, imzoni belgima-belgi
  taxmin qilishning klassik timing kanali. Endi `subtle.ConstantTimeCompare`
  (`payclick.go`).
- ⚠️ **Operator qidiruvi Mongo `$regex` ga `regexp.QuoteMeta` bilan tushadi.**
  `adminusers.go` (`?q=`) va `auditlog.go` (`?action=`) quote qilmasdan
  yuborardi — faqat autentifikatsiyalangan admin kira oladi va ma'lumot
  chiqmaydi (Mongo `$regex` maydonlararo o'tolmaydi), lekin `(a+)+$` kabi qiymat
  katastrofik backtracking DoS beradi. Qolgan qidiruv joylari `textSearch`
  helper'idan o'tadi; bu ikkitasi qo'lda qurardi.
- **Bog'liqliklar:** `sharp` yangilandi (`npm audit fix`). Qolgan `postcss`
  "high" — Next 15'ning **build-vaqti** bog'liqligi, faqat Next 16 (major,
  breaking) bilan tuzatiladi; real xavf past (build paytida o'z CSS'ini qayta
  ishlaydi, foydalanuvchi kiritmasini emas), shuning uchun **ataylab
  ko'tarilmadi** — alohida, sinovli upgrade.

### Panel adminlari va amallar jurnali
- **Yangi admin — mavjud sayt mijozi**: odam avval saytda telefon + SMS bilan
  kirgan bo'lishi kerak; owner `/admin/admins` da uni qidirib topadi va login +
  vaqtinchalik parol beradi. `admin_user.userId` shu mijozga bog'lanadi
  (bitta mijoz — bitta panel hisobi).
- Yaratilgan hisob `mustChangePassword: true` bo'ladi: birinchi kirishda
  `AdminLayout` uni `/admin/account` ga majburan yuboradi. Parolni tiklash ham
  shu bayroqni qayta qo'yadi (boshqa odam yozgan parol vaqtinchalik).
- **Faqat `owner`**: hisoblarni boshqarish va jurnalni ko'rish
  (`RequireRole(..., "owner")`). Menejer o'ziga huquq qo'sha olmaydi va
  o'zidan keyingi izni o'chira olmaydi. O'z hisobini va oxirgi ownerni
  o'chirish taqiqlangan.
- **Amallar jurnali** (`admin_log`, `handlers/auditlog.go`): har bir o'zgarish
  `h.logAction(...)` bilan yoziladi — kirish, buyurtma holati/bekor qilinishi
  (sababi bilan), kuryer biriktirish, tashqi xizmat chaqirish, kuryer/admin/
  taom/kategoriya/xizmat CRUD, sozlamalar. `action` — barqaror id
  (`order.cancel`), matn panelda tarjima qilinadi. Yozuvlar tahrirlanmaydi va
  o'chirilmaydi; log yozilmasa ham amal bekor qilinmaydi.

### TV ekranlar: ulash, uzish va sanash (1-bosqich)
Zaldagi televizorlar uchun Android TV ilovasi: kontent (video/rasm) va fastfood
uchun buyurtma tablosi. Birinchi bosqichda **ulash, ro'yxat va uzish** qilindi.
- **Ekran kodni ko'rsatadi, kod panelga yoziladi** — teskarisi emas. Televizor
  pult bilan boshqariladi, va zal oldida ekran klaviaturasidan bitta-bitta harf
  terib parol yozish — hech kim oxiriga yetkazmaydigan sozlash. Olti belgini
  ovoz chiqarib o'qish esa ishlaydi.
- ⚠️ **Kioskning kodi bu yerga to'g'ri kelmaydi.** Kiosk kodi filial sirlaridan
  HMAC bilan **hosil qilinadi**, chunki ekran o'zi kimligini biladi. Ulanayotgan
  televizor esa hech kim emas — demak kodni server yaratadi **va eslab turadi**
  (`tv_pairing`).
- ⚠️ **Kod devorda turadi, ya'ni uni bilish yetarli bo'lmasligi kerak.** Kod
  **bir daqiqa** yashaydi va tugashiga yaqin TV yangisini so'raydi (eskisi o'sha
  zahoti o'ladi — `installId` bo'yicha unique indeks), va **so'rov kod bilan
  emas, `pollSecret` bilan** javob oladi.
  ⚠️ **Avval 10 soniya edi, va bu jonli installda ulanishni umuman imkonsiz
  qildi**: menejer olti belgini o'qiydi, noutbukka boradi, panelni topadi,
  filialni tanlaydi va yozadi — kod esa shu orada ikki marta almashgan bo'ladi.
  Nosozlik "ekran buzuq" bo'lib ko'rinadi, aslida u shunchaki odamdan tez edi.
  Oynani uzaytirish xavfsiz, chunki himoya oynada emas: kodni **panel logini**
  va o'sha filialga huquqi bor odamgina ishlata oladi, kod bir martalik, va
  bitta televizorda bir vaqtda bitta kod bo'ladi.
  ⚠️ **Muddatni server aytadi (`expiresIn`), ilova o'z taymerini yuritmaydi.**
  Ilgari ilovada 10 soniyalik konstanta, serverda 90 soniyalik muddat turardi —
  ya'ni devordagi hisob to'qqiz barobar yolg'on edi. Aks holda zaldagi har kim kodni o'qib,
  ilovadan tezroq so'rab, devorga atalgan tokenni olib ketardi.
- ⚠️ **Token bir marta beriladi**: olingandan keyin pairing hujjati o'chadi, ya'ni
  kechikkan takroriy so'rov "expired" oladi. Noto'g'ri `pollSecret` ham aynan shu
  javobni oladi (constant-time solishtirish) — boshqacha javob "to'g'ri
  televizorni topding" degan ma'lumot bo'lardi.
- **Ulashni faqat filialga huquqi bor admin qiladi** (`requireBranchAccess`):
  kodni zaldagi har kim o'qiy oladi, lekin uni Chilonzorga biriktirish paneldan
  bo'ladi. Operator roli bu yerga umuman kira olmaydi.
- **Uzish ikki xil**: bitta ekran (`DELETE .../screens/{id}`) va butun filial
  (`POST .../revoke` → `branch.tvVersion++`). Ikkinchisi — televizor yo'qolgan
  holat uchun; u har bir ekranni qaytadan ulashni talab qiladi, shuning uchun
  panel oldin so'raydi. ⚠️ Versiya **har so'rovda** tekshiriladi: bir yillik
  token uchun "uzdim" degani keyingi so'rov rad etilishi bo'lishi shart.
- ⚠️ **Bir televizor — bir qator** (`installId` unique): zavod sozlamalariga
  qaytarilgan yoki boshqa filialga ko'chirilgan ekran o'z qatorini **almashtiradi**,
  aks holda bitta devor ikkita pullik joyni egallardi.
- **Narxi ekran boshiga** (50 000 so'm/oy) → `subscription.screens` — kassalardagi
  `registers` bilan bir shakl, va cheklov **eshikda** tekshiriladi, devorda osilgan
  ekranda hech qachon emas.

**Ilova tomoni** (`mobile/tv`, 1-bosqich): manzil → kod → ulanish → yurak
urishi. To'rt holat, va ulardan ikkitasi ataylab ajratilgan — "ulanmagan" va
"serverga yetib bo'lmadi" `catch` ichida bir xil ko'rinadi, ma'nosi esa teskari:
internet o'chganda kod ko'rsata boshlaydigan televizor menejerni bekorga qayta
ulashga majbur qiladi va ekranga ishonchni yo'qotadi. Tafsiloti —
`mobile/tv/README.md`.
- ⚠️ Kod muddati **ilovaning o'z soatidan** sanaladi (server `expiresIn` soniya
  yuboradi): arzon televizorning soati oylab noto'g'ri bo'ladi.
- ⚠️ `LEANBACK_LAUNCHER` bo'lmasa APK o'rnatiladi-yu bosh ekranda ko'rinmaydi
  (`plugins/withAndroidTV.js`); `required="false"` bo'lmasa esa telefonga
  o'rnatilmaydi — ya'ni uni ishlab chiqadigan mashinaga ham.

**Konsolda sotiladigan joyi** (`keel-site` → tenant → Kassa obunasi):
**"TV ekranlar"** — bitta raqam, `× 50 000 so'm/oy`.
- ⚠️ **Raqam — huquqning o'zi**, yonida yoqish/o'chirish tugmasi yo'q. Tugma va
  raqam — bir-biriga zid bo'la oladigan ikki fakt ("modul yoqilgan, ekran soni
  nol", "to'rt ekran bor, modul o'chiq"), va ikkinchisi qorong'i zal degani.
  `tvScreens > 0` → tenantga `tv` moduli **va** `screens` chegarasi ketadi
  (`mirrorTill`).
- ⚠️ **Ekran boshiga narxlanadi, filialga emas** — bu ladderdagi yagona shunday
  qator. Qolgan hammasi xona bilan o'lchanadi (bitta oshxona, bitta ombor, bitta
  menejer), televizor esa **bitta xona ichida** ko'payadigan yagona narsa:
  peshtaxta yonida, eshik oldida, kassa tepasida. To'rttasini bittadek narxlash
  modul mavjud bo'lish sababini bekor qiladi; filial bo'yicha narxlash esa
  zanjirga yo'q televizor uchun hisob yozadi.
- ⚠️ **30 ta bilan cheklangan** (`cleanScreens`): bu raqam to'g'ridan-to'g'ri
  hisob-fakturaga ko'payadi, va "4" o'rniga terilgan "40" — sakkiz barobar
  ortiq hisob, yuborilgan va keyin telefonda tushuntiriladigan.
- Cheklov **eshikda**: ekran ulashda tekshiriladi, devorda osilganida hech
  qachon — hisob kechikkani uchun zal qorong'i bo'lib qolmaydi.

### TV kontent: playlist, muddat va oflayn (2-bosqich)
Ekranlar ulangandan keyingi savol — "endi videoni qayerdan yuklayman?". Javob:
**panel → TV ekranlar → Kontent** tabi. Rasm va video yuklanadi, tartib
o'zgartiriladi, rasmga soniya, aksiyaga sana beriladi.
- **Playlist filialniki, ekranniki emas.** Bitta zaldagi ikki televizor bir xil
  restoranning taomini ko'rsatadi; ular orasidagi farq — tablo yoqilganmi, va bu
  allaqachon ekranning o'z xossasi (`TVScreen.mode`). Har ekranga alohida
  ro'yxat bo'lsa, bitta videoni to'rt marta yuklash va to'rt marta almashtirish
  kerak bo'lardi — va esdan chiqqan to'rtinchisi o'tgan oyning aksiyasini
  ko'rsatib turadi.
- ⚠️ **Video alohida endpoint bilan yuklanadi** (`POST /admin/tv/video`).
  `/admin/upload` qabul qilgan hamma narsani WebP'ga o'giradi — videoni rasm
  koderiga bersangiz u kichraymaydi, **buziladi**, va ishonchli nom ostida
  saqlanadi. Cheklov 120 MB, faqat MP4/WebM, va **birinchi baytlar tekshiriladi**:
  kengaytma — da'vo, bu papkani esa butun internet o'qiydi.
- ⚠️ **Fayl xotiraga o'qilmaydi**, `MultipartReader` bilan to'g'ridan-to'g'ri
  diskka oqiziladi: 120 MB `io.ReadAll` — 1 GB'lik VPS'ning sakkizdan biri, ikki
  menejer bir vaqtda yuklasa esa hammasi.
- **Har fayl televizorga yuklab olinadi va o'sha yerdan o'ynaydi.** Bu
  optimizatsiya emas: devordagi ekran bir xil 40 MB klipni kuniga yuzlab marta
  aylantiradi — kassa va ofitsiantlar telefoni turgan **o'sha** wifi orqali.
  Va aynan shu narsa ekranni internetdan mustaqil qiladi: router o'chganda zal
  qorayadigan televizorni bir haftada rozetkadan sug'urib qo'yishadi.
- ⚠️ **Muddatni televizor o'zi hisoblaydi, server ro'yxatni qirqmaydi.** Juma
  kunidan beri internetga chiqmagan ekran shanbada tugagan aksiyani ko'rsatib
  turmasligi kerak. Devordagi eskirgan taklif bo'sh ekrandan **battar**: mehmon
  uni kassada so'raydi. Shuning uchun `startsAt`/`endsAt` ro'yxat bilan birga
  ketadi, `active: false` esa serverda filtrlanadi (u o'zi o'zgarmaydi).
- ⚠️ **Sana — kun, instant emas.** Ega "30 sentyabrda tugaydi" deganda
  o'ttizinchi kunning oxirini nazarda tutadi; sanani so'zma-so'z olsak, aksiya
  o'sha kuni **eshik ochilganda** devordan tushadi va bu "ekran buzuq" bo'lib
  keladi. Server `2006-01-02` ni **mahalliy vaqtda** o'qiydi va tugashga
  `24h - 1s` qo'shadi.
- ⚠️ **Televizorning o'z soati ishlatilmaydi.** Arzon Android TV tarmoqsiz
  yuklanganda 1970 yilda yoki zavod sanasida keladi, va butun jadval shu soatga
  solishtiriladi. Server har `/tv/me` da vaqtni aytadi (`clock.ts`), ilova
  farqni saqlaydi; sovuq yuklashda oxirgi ma'lum farq tiklanadi — taxmin, lekin
  birinchi heartbeat uni bir daqiqada tuzatadi.
- **Ro'yxat versiya bo'yicha o'qiladi** (`branch.tvContentVersion`, heartbeat
  ichida). Har daqiqada butun ro'yxatni tortadigan ekran javobi deyarli doim
  "o'tgan daqiqadagidek" bo'lgan savolni beradi. Har yozuvda versiya oshadi —
  hatto nom o'zgarganda ham: "faqat muhimlarida oshiramiz" degan qoidani
  keyingi tahrir buzadi, narxi esa bitta `$inc`.
- **Videoni o'chirish faylni ham o'chiradi** — kod bazasidagi yagona joy.
  Rasm yuz kilobayt va taom bilan bo'lishilgan bo'lishi mumkin; bir daqiqalik
  1080p — ellik megabayt, va uch yozgi aksiyalar bilan to'lgan disk "sayt
  ishlamayapti" bo'lib keladi. ⚠️ Ikki shart: URL **bizniki** (`/uploads/` va
  ichida `/` yo'q — biz yozgan har fayl tekis, tasodifiy nomli) va **boshqa
  hech bir element** unga ko'rsatmayapti.
- **Ro'yxatda 60 tagacha element**: har biri filialdagi **har** televizorga
  yuklab olinadi, va 8 GB xotirali set yuzta video bilan kechqurunni disk
  to'ldirish bilan o'tkazadi — nosozlik esa yuklashdan haftalar keyin keladi.
- **Tartib ikkita tugma bilan, sudrash bilan emas**: ro'yxat qisqa, uni
  tahrirlayotgan odam ko'pincha noutbuk trekpadida, va noto'g'ri joyga tushgan
  sudrash — hech kim sezmaydigan o'zgarish. ⚠️ Tartib **butun ro'yxat** sifatida
  yuboriladi: "buni yuqoriga" ni server qayta hisoblagan ro'yxatga qo'llash —
  ikki brauzerdagi ikki menejer uchinchi o'ringa ikki element qo'yishining yo'li.
- **Zonalar: qaysi kontent qaysi ekranda** (playlist filialniki degan qoidaning
  bekor qilinishi emas, chegarasi). Ekranda `zone` (u bitta devorga osilgan),
  slaydda `zones[]` (u bir nechta xonaga yo'naltirilishi mumkin).
  ⚠️ **Bo'sh ro'yxat "hech qayerda" emas, "hamma joyda" degani**, va butun
  dizayn shunga tayanadi: migratsiya kerak emas (bugungi har slayd bo'sh
  ro'yxat bilan turibdi va avvalgidek ishlaydi), "hamma TV da bir xil aylanma"
  hech nima bosmasdan olinadigan holat bo'lib qoladi, va **zonani belgilashni
  unutish slaydni ko'proq joyda ko'rsatadi, kamroq emas**. Ekranga alohida
  ro'yxat berish rad etilgan sabab aynan teskarisi edi — unutilgan to'rtinchi
  ro'yxat o'tgan oyning aksiyasini ko'rsatib turardi; standart qiymati "hamma
  joyda" bo'lgan qoida bunday yiqila olmaydi.
- ⚠️ **Zona serverda filtrlanadi, sana esa televizorda** — ajratish tasodifiy
  emas: ekranning zonasi kimdir qabul qilgan qaror va u oflayn paytda
  o'zgarmaydi, sana oynasi esa yarim tunda o'zi yopiladi.
- ⚠️ **Zona kichik harfga tushiriladi** (`cleanZones`, va ekran tomonida ham).
  Slaydda «Zal», ekranda «zal» — bu **hech qayerda o'ynamaydigan** aksiya:
  hamma maydon to'ldirilgan, hech qayerda xato yo'q, yagona alomat — kimdir
  turgan xonadagi qorong'i televizor.
- ⚠️ **Ekranning zonasi o'zgarganda `branch.tvContentVersion` oshiriladi**,
  garchi playlistda hech nima o'zgarmagan bo'lsa ham. Televizor ro'yxatni
  **faqat** o'sha raqam siljiganda qayta o'qiydi; bumpsiz ko'chirilgan ekran
  eski zonasini **abadiy** o'ynaydi. Narxi — filialdagi har ekranning bitta
  qo'shimcha so'rovi; muqobili — hech qachon ko'chirib bo'lmaydigan ekran.
- ⚠️ **Panel har slayd yonida "necha ekranda chiqadi" deb yozadi, nol bo'lsa
  qizil.** Ega aksiyani «terrasa» ga yo'naltiradi, terrasada esa hali televizor
  yo'q — slayd hech qayerda chiqmaydi va buni boshqa hech nima aytmaydi. Bu
  raqam bezak emas, xususiyatning jim yiqilishiga qo'yilgan yagona to'siq.
- **Zonalar ro'yxati alohida saqlanmaydi** — u ekranlardan hosil qilinadi.
  Televizori yo'q zonani yaratib qo'yib, keyin u haqda o'ylab yurish kerak
  bo'lmasin. Yonaki foyda: har televizor endi butun filialning emas, **faqat
  o'ziga tegishli** fayllarni yuklab oladi.
- **Rasm — soniya bilan, video — o'z uzunligicha.** 3–120 soniya oralig'i:
  "30" o'rniga terilgan "300" qotib qolgan televizor, va qotgan televizorga
  qilinadigan birinchi ish — rozetkadan sug'urish.
- ⚠️ **Ro'yxat bo'sh bo'lsa ekran qora emas**: filial nomi va "Kontent yo'q —
  Keel panelida: TV ekranlar → Kontent" turadi. Qora to'rtburchak bilan
  ishlamayotgan ilova bir xil ko'rinadi.
- `board` rejimidagi ekran playlistni **o'ynatmaydi** — tablo keyingi bosqichda.

### TV tablo: qaysi raqam pishmoqda, qaysisi tayyor (3-bosqich)
`board` va `split` rejimidagi ekranlar endi buyurtma raqamlarini ko'rsatadi
(`GET /tv/board`).
- ⚠️ **Faqat raqamlar.** Ism yo'q, taom yo'q, summa yo'q. Bu ekranni xonadagi
  hamma o'qiydi — jumladan buyurtmasi unda bo'lmagan odamlar ham. Ism yozadigan
  tablo — devordagi mijozlar ro'yxati; taom yozadigani esa qirq begonaga
  oltinchi stol nima yeyayotganini aytadi. Chekda turgan raqam yetarli. Server
  `projection` bilan **faqat `number`** ni o'qiydi, ya'ni keyingi tahrir uni
  tasodifan kengaytira olmaydi.
- ⚠️ **Yetkazib berish tabloga chiqmaydi**: tablo "meniki tayyormi?" degan
  savolga **xonada turgan** odam uchun javob beradi. Yetkazishdagi mijoz uyda,
  va uning raqami devorda hech kimga kerak emas — u oshxona ekranida va kuryer
  ilovasida. Shuning uchun `type: pickup | dinein`.
- **"Tayyor" — holat emas, `readyAt` vaqt belgisi** (KDS bo'limidagi qaror shu
  yerda ham amal qiladi). Holat esa **ketganini** bilish uchun o'qiladi:
  `delivered`/`cancelled` chek darhol devordan tushadi.
- ⚠️ **Tayyor raqam 15 daqiqadan keyin o'zi tushadi.** Uni pastga tushiradigan
  ishonchli narsa yo'q: berilgan buyurtma kassada belgilanadi, gavjum kechqurun
  esa belgilanmaydi. Oynasiz tablo bir soat oldin ketgan odamlarning raqamlari
  bilan to'ladi va o'zinikini qidirayotgan mehmon qidirishni tashlaydi.
- **Pishayotganlar filtri KDS bilan bir xil** (`pending` yo'q, `queuedAt <= now`).
  Bitta binodagi ikki ekran bitta buyurtma haqida boshqa-boshqa gapirsa, mehmon
  aynan ularning **o'rtasida** turgan bo'ladi.
- ⚠️ **Ikkita indeks qo'shildi** (`branchId+status+queuedAt`,
  `branchId+readyAt` — partial). KDS ro'yxatini oshpaz qaraganda bir planshet
  o'qiydi; tabloni esa filialdagi **har** televizor har o'n soniyada, butun
  kechqurun, faqat o'sadigan kolleksiyadan so'raydi. Indekssiz bu — restoran
  qabul qilgan har bir buyurtmani daqiqasiga bir necha marta to'liq skanerlash,
  va buni hech kim tegmayotgan ekran uchun.
- ⚠️ **Aloqa uzilsa tablo jim bo'ladi** (2 daqiqadan keyin) — playlist bilan
  **teskari** qoida, va ataylab. Aylanma uzilgan aloqada ham restoranning o'z
  kontentini ko'rsatadi, ya'ni zarari yo'q. Tablo esa **ovqat haqida da'vo
  qiladi**: "tayyor" degan eskirgan raqam mehmonni peshtaxtaga yuboradi va u
  yerda "yo'q" eshitadi. Hech nima ko'rsatmagan ekran yaxshiroq — mehmon
  odamdan so'raydi, ya'ni baribir qiladigan ishini qiladi.
- **Bo'sh tablo — bo'sh jadval emas**, filial nomi. Tushlik bilan kechki ovqat
  orasida sarlavhalari bor, ostida hech nima yo'q ekran soatlab "buzuq" bo'lib
  turadi.
- **`split` da faqat "Tayyor" chizig'i**, ikkala ustun emas: chiziq — bir
  qarash, va yarim metr devor ikkita ro'yxatni ko'tarmaydi. **Tayyor bo'lmasa
  chiziq umuman yo'q** — videoning ustidagi doimiy bo'sh panel restoranning o'z
  ekranini yeydigan mebel. Playlist bo'sh bo'lsa `split` butun tabloga o'tadi.
- **Raqam o'lchami ekran kengligidan hisoblanadi**: bu ilova peshtaxtadagi 32"
  va zaldagi 65" da ishlaydi, bittasiga moslangan o'lcham ikkinchisida o'qib
  bo'lmaydigan yoki kulgili bo'ladi.

### ⚠️ Modul darvozasi hech qachon ishlamagan (`/api/v1` prefiksi)
`moduleFor` yo'lni `/admin/…` ko'rinishidagi jadval bilan solishtirardi, `r.URL.Path`
da esa `/api/v1/admin/…` turadi — chi `Route` so'rov URL'ini qayta yozmaydi.
Ya'ni **hech biri mos kelmagan** va darvoza har bir pullik modulni har bir
installda ochiq qoldirgan. Panel darvozasi bilan bir xato, faqat teskari tomonga
yiqilgan: u yopiq yiqilib omborchini qulflagan, bu esa **ochiq** yiqilib
sotilmagan modulni bepul bergan — shuning uchun uni hech kim sezmagan.
- Tuzatildi: `moduleFor` `panelPath()` bilan prefiksni kesadi, va testda endi
  **haqiqiy yo'l** (`APIBase + "/admin/ingredients"`) bor — ilgari test faqat
  jadvalning o'zini tekshirardi, ya'ni darvoza o'chiq turganda ham yashil edi.
- ⚠️ **Deploy oqibati**: bu tuzatilgach, `stock` moduli bo'lmagan tenantlar
  ombor bo'limini **yo'qotadi** (modulegate.go da yozilgan niyat aynan shu, lekin
  amalda hech qachon kuchga kirmagan). Deploydan **oldin** konsolda kimda qanday
  modul borligini ko'rib chiqish kerak.

### Panelning cheklangan rollari: ombor va operator
Panelda to'liq huquqli ikki rol bor (`owner`, `manager`) va **ikkita cheklangan**
rol: `stock` (omborchi/texnolog) va `operator` (call-markaz). Ikkalasining qoidasi
bitta joyda — `handlers/panelgate.go`.
- ⚠️ **Ruxsat ro'yxati, taqiq ro'yxati emas.** Tokenni admin guruhiga kiritib,
  ko'rmasligi kerak bo'lgan bo'limlarni yopish — ertaga qo'shiladigan **har bir
  yo'l** kimdir eslamaguncha ochiq degani. Bu yerda ro'yxatda yo'q yo'l rad
  etiladi: keyingi tahrirning xatosi "operator kamroq ko'radi" bo'ladi, va u
  o'sha kuni aytiladi.
- ⚠️ **Metod ham qoidaning yarmi.** Buyurtmalar taxtasida kuryer tanlash
  ro'yxati bor, ya'ni operator `/admin/couriers` ni **o'qishi** shart — va shu
  prefiks orqali kuryer hisobini yaratish va o'chirish ham o'tadi. Metodsiz
  prefiks ro'yxati ikkinchisini birinchisi bilan birga beradi, va buni hech bir
  ekran ko'rsatmaydi.
- ⚠️ **`/admin/me` — `/admin/menu` ning prefiksi.** Operator o'z profilida
  hamma narsani qila oladi va menyuni faqat o'qiydi; prefiks sifatida birinchi
  qoida ikkinchisini yutib yuborgan va `PUT /admin/menu/{id}` **ochiq** qolgan
  edi. Shuning uchun `pathRule.exact` bor. Qo'shnisi — `/admin/stock` va
  `/admin/stop-list` tuzog'i; ikkalasi bitta oila.
- ⚠️ **Prefiksni kesish shart: `r.URL.Path` da `/api/v1` turadi.** chi'ning
  `Route` i so'rov URL'ini qayta yozmaydi, ro'yxatlar esa `/admin/...` deb
  yozilgan — ya'ni **hech biri mos kelmasdi**, va deny-by-default darvoza
  omborchiga **butun panelni**, jumladan `/admin/me` ni ham rad etardi. Panel
  esa `/admin/me` dagi xatoni "token eskirgan" deb o'qiydi: to'g'ri parol →
  login sahifasi, qayta-qayta, hech qayerda xato yozilmagan holda. Yopiq
  yiqilgani uchun bu sizib chiqish emas, qulflanish bo'ldi. `handlers.APIBase`
  bitta manba, va middleware testi **aynan shu prefiks bilan** chaqiradi (ilgari
  test `/admin/orders` ni chaqirib o'tib ketardi).
- **Operator ko'radigan uchta bo'lim**: buyurtmalar, bronlar, call-markaz
  (+ o'z hisobi). Menyu/kategoriya/kuryer/filial — faqat o'qish, chunki telefon
  orqali buyurtma shulardan yig'iladi. Mijozlar bazasi, hisobotlar, kassa,
  ombor, sozlamalar, jurnal va hisoblar — yo'q.
- ⚠️ **Yangi hisob `mustChangePassword` bilan tug'iladi**, ya'ni
  `PUT /admin/credentials` ruxsat ro'yxatida bo'lishi shart — aks holda operator
  o'zi yuborilgan yagona ekranda qulflanib qoladi.
- **Panel tomonida ham ro'yxat**: `navFor(role)` faqat o'shalarni chizadi va
  `mayOpen(role, path)` **shu navigatsiyadan o'qiladi** — havola, xatcho'p yoki
  brauzer tiklagan tab orqali kelgan odam o'z bo'limiga qaytariladi. Kirgandan
  keyin qayerga tushishi — `lib/panelRole.ts` → `homeFor()` (operator →
  buyurtmalar, ombor → qoldiqlar): `/admin` — kompaniyaning raqamlari, ikkalasiga
  ham yopiq, va **xatolik bilan ochiladigan panel buzuq hisobga o'xshaydi**.
- ⚠️ **Texnolog roli endi `stock` bilan yuboriladi** (yagona seeded rol).
  Texkartani u yozadi, inventarizatsiyani u qiladi — ya'ni ishi omborning o'zi,
  va bu ruxsat uning panelga kiradigan **yagona eshigi**. Bo'sh yuborilganda har
  restoran texnolog yollab, birinchi sanashda bilardi: berilgan hisob login
  formasini "login yoki parol noto'g'ri" deb rad etadi — bu buzuq parolga
  o'xshaydi, hech kim yoqmagan tugmaga emas.
  Mavjud installlarga migratsiya beradi (`grantTechnologistStock`) — `name` +
  `seeded` bo'yicha, `$addToSet` bilan. ⚠️ **Bir marta ko'riladi va belgilanadi**
  (`stockGranted`): "stocksiz Texnolog" bo'yicha qidirish restoran endigina
  o'chirgan ruxsatni **har boot'da qaytarib** qo'yardi, va kechada o'sib
  chiqadigan ruxsat umuman berilmagandan yomonroq. `EnsureReviewsBand` bilan bir
  shakl: bayroq **tashrifni** yozadi, natijani emas.
- ⚠️ **Texnolog paneldan `stock` ruxsati bilan kiradi** (`staff` hisobi,
  `stocklogin.go`), operator esa haqiqiy `admin_user`. Shuning uchun `/admin/me`
  omborchi uchun **staff** kolleksiyasidan javob beradi (`stockMe`), va javobning
  shakli login bilan **bitta funksiyadan** (`stockUserView`) — ular ajragan payt
  panel odamni kiritib, keyingi so'rovda "tokening eskirgan" deb chiqarib
  yuborardi.

### Mijozni o'chirish: ikki xil, va ikkinchisi qaytarilmaydi
- **Vaqtincha o'chirish** (`status: deleted`) — sayt o'chadi, **hamma narsa
  qoladi**. Martda ketgan mijoz mayda qaytadi, va u qayta yarata olmaydigan
  yagona narsa — o'z ma'lumoti.
- **To'liq o'chirish** (`POST /tenants/{id}/purge`, `handlers/purge.go`) — baza,
  rasmlar va konteyner butunlay o'chadi.
- ⚠️ **Hisob-fakturalar va kunlik yozuvlar ataylab qoladi**: ular mijozniki
  emas, **bizning** hisobimiz. "Martda qancha to'lagan?" savoliga javob yo'qligi
  — yo'q qilingan ma'lumotdan yomonroq. Tenant hujjati ham qoladi
  (`purgedAt/purgedBy/purgeReason`): fakturalar unga ishora qiladi va **slug
  boshqa hech kimga berilmaydi**.
- Qo'riqchilar (`purgeRefusal`, testda muhrlangan): **faqat owner**; mijoz
  **avval vaqtincha o'chirilgan** bo'lishi shart (ikki qadam — noto'g'ri
  o'chirishlarning deyarli hammasi noto'g'ri **qator**, va birinchi qadam uni
  bepul ushlaydi); **slug qo'lda yoziladi** (dialog o'sha refleks bilan
  yopiladi, va u nima o'chayotganini ko'rsata olmaydi); **sabab majburiy**;
  **zaxira eskirgan bo'lsa rad etiladi** — yagona qaytarish yo'li o'sha.
- **Fayllar bir martalik konteyner orqali o'chiriladi** (`PurgeUploads`), chunki
  konsolda `uploads` **faqat o'qish uchun** ulangan va bitta kam ishlatiladigan
  tugma uchun doimiy yozish yo'lini ochish yomon savdo. Bind — **o'sha
  tenantning o'z katalogi**, ildiz emas.
- Javob **qadam-baqadam** qaytadi: konteyner / rasmlar / baza / yozuv / chekka.
  Ular mustaqil yiqiladi va tuzatishlari butunlay boshqa — bitta "xato" qaysi
  yarmi qolganini yashiradi.

### Buyurtma pulini bekor qilish orqali "yo'qotib" bo'lmaydi
- Biz har buyurtmadan pul olamiz, **bekor qilingani bepul** — bu pishirilmasdan
  to'xtatilgan buyurtma uchun to'g'ri.
- ⚠️ **Va u ochiq taklif edi**: kunlik qator har kecha buyurtmaning **hozirgi
  holatidan** qayta quriladi, ya'ni seshanba yetkazilgan buyurtmani chorshanba
  "bekor qilindi" deb belgilash pulni qaytarib olardi. Bitta bosish, qatorda
  hech qanday iz yo'q. Hammasiga shunday qilgan restoran **hech nima to'lamas**,
  mehmonlari, kuryerlari va o'z paneli esa odatdagidek ishlab turardi.
- Yechim: hisob **`statusHistory` ni** o'qiydi (panel unga faqat **qo'shadi**).
  `delivered` ga yetgan buyurtma **abadiy hisobga kiradi**. Halol bekor qilish
  bu holatga hech qachon tegmaydi, ya'ni va'da o'zgarmadi — faqat so'zni
  **orqaga qarab** qo'llab bo'lmaydi.
- ⚠️ **Bu firibgarlik detektori emas va bo'lishi ham kerak emas.** U faqat
  **mukofotni** olib tashlaydi. Niyatni baholaydigan qoida ertami-kechmi
  eshik oldida ovqatdan voz kechgan mehmon uchun restoranni ayblardi.
- Ko'rmaydigan qismi **yozib boriladi**: `reversed` (yetkazilgandan keyin bekor
  — baribir hisobga kiradi) va `cancelledCooked` (oshxonaga tushgan, lekin
  yetkazilmagan — hisobga kirmaydi). Mijoz kartochkasida **faqat nolga teng
  bo'lmaganda** bir jumla bo'lib chiqadi.
- ⚠️ Tuzoq: agregatsiyada **yo'q maydon `null` emas, "missing"** — `$ne` uni
  **rost** deb javob beradi. `$ifNull` siz har bir halol bekor qilish
  "pishirilgandan keyin bekor qilingan" bo'lib sanalardi.
- ⚠️ Tuzoq: `localZone()` Go zonani nomlay olmaganda **`"Local"`** qaytarardi,
  Mongo esa bunday identifikatorni rad etadi — ya'ni kun chegarasi siljishi
  emas, **butun agregatsiya ishlamasligi** (hech qanday qator, hech qanday
  hisob-faktura, hamma tenant uchun).

### VPS resurslari: chegara bor, lekin adolat ham kerak
- Bir tenant = bir konteyner (**faqat Go backend**, ~9 MB), frontend **umumiy
  pul** (`keel-frontend-1..3`), Mongo **umumiy**. Ya'ni "qo'shnini bezovta
  qilish" asosan **umumiy qatlamlarda** bo'ladi, konteynerda emas.
- Konteyner: 512 MB + 1 yadro (shift), **swap o'chiq** (`MemorySwap =
  Memory`), `PidsLimit`, `CpuShares`, `BlkioWeight`.
  ⚠️ **Swap standart holatda ikki barobar** edi: sizib ketayotgan konteyner
  512 MB **diskni** sekin xotira sifatida sarflardi — Mongo, hamma tenantning
  rasmlari va tunlik zaxira turgan o'sha diskda. Alomati "bitta sayt o'chdi"
  emas, "butun quti sekinlashdi" bo'ladi, va uni izlash ancha qiyin.
- ⚠️ **Mongo ulanish puli har tenantda cheklangan** (`SetMaxPoolSize(20)`).
  Standart 100 **har mijozga**, ya'ni shift bizniki emas — sotilgan mijoz
  soniga ko'paytiriladi, va har ulanish serverda thread turadi. Butun platforma
  jami ~20 ulanishda ishlaydi.
- ⚠️ **`order.createdAt` ga alohida indeks kerak**: `(branchId, createdAt)`
  faqat `branchId` dan boshlanadigan so'rovga javob beradi, tunlik hisob esa
  filialsiz `createdAt` bo'yicha o'qiydi → jonli tenantda **COLLSCAN**
  o'lchandi. Bu — eng band restoranning butun tarixi, har kecha, **umumiy**
  Mongo'da.
- ⚠️ **Yangi cheklovlar faqat konteyner qayta yaratilganda qo'llanadi** —
  `docker compose build` tuzog'ining o'sha yuzi. Mavjud tenantlar rollout
  (`Recreate`) bilan yangilanadi; tekshirish: `docker inspect -f
  '{{.HostConfig.MemorySwap}} {{.HostConfig.PidsLimit}}'`.
- **Kam trafikli mijoz allaqachon deyarli hech nima sarflamaydi**: `Memory` —
  **shift**, rezerv emas, va bo'sh turgan tenant ~9 MB oladi. Bu yerda
  qaytarib olinadigan resurs yo'q, ya'ni "bo'sh tenantlarni o'chirib turish"
  kerak emas.

### ИКПУ (fiskal chek kodi)
- `menu_item.ikpu` — **ixtiyoriy** maydon, panelda taom formasida.
- ⚠️ **Bo'sh — bo'sh qolishi kerak.** Kodni restoranning buxgalteri beradi; uni
  taom nomidan chiqarib bo'lmaydi, va ishonarli ko'ringan taxmin **yo'qligidan
  yomonroq**: noto'g'ri ИКПУ — noto'g'ri mahsulotga yozilgan fiskal chek, ya'ni
  restoranning soliq bilan muammosi. Shuning uchun kod bo'lmasa ATMOS savatiga
  **maydonning o'zi yuborilmaydi** (`omitempty`, testda muhrlangan).
- **17 ta raqam**, ajratgichlar (`-`, bo'sh joy) tashlanadi; harf yoki boshqa
  uzunlik — maydon **tozalanadi** (`normalizeIkpu`). Yarim yozilgan kod ham,
  noto'g'ri ustundan nusxa olingan matn ham chekka tushmasligi kerak.
- ⚠️ **Buyurtmaga muzlatilmaydi, menyudan o'qiladi** (`menuIkpu`) — yonidagi nom
  va narxdan farqli. Ular mijoz rozi bo'lgan narsa; bu esa **mahsulot** haqidagi
  fakt, ya'ni buxgalter xatoni tuzatsa hali to'lanmagan buyurtmalarga ta'sir
  qilishi kerak.

### Fiskal provayderlar: ro'yxat va kalitlar

**⚠️ Rahmat POS ikkita mahsulot, va ular bitta id emas.** Paneldagi
«Multikassa» qatori — kassa kompyuteridagi dastur, Rahmat uni qayta sotadi va
lokal adapter allaqachon shuni haydaydi (LAN, autentifikatsiyasiz). Rahmat'ning
**bulutli** virtual kassasi esa internetdan hisob bilan chaqiriladi — boshqa
transport, boshqa kalitlar, boshqa nosozlik turi. Bitta id qilib qo'yish bulutli
mahsulot sotib olgan egani o'z ofis tarmog'iga qaratardi. Shuning uchun
`rahmat` alohida.

Qo'shildi: `rahmat` (bulutli), `qpos`, `arca` — hammasi `Ready: false`.
⚠️ **Adapteri yo'q provayderni yoqib bo'lmaydi.** Tanlash va kalit saqlash
mumkin (ega ko'pincha shartnoma tugashidan oldin sozlaydi), lekin yoqish sabab
bilan rad etiladi: fayl qilyapman deb o'ylagan restoran — yo'q xususiyat emas,
**huquqiy muammo**.

**⚠️ Har provayder uchun alohida maydon → bitta map (tuzatildi).** Kalitlarni
tanlash uchta `switch`, bitta struct literal va `$set` dagi oltita qatorda
takrorlanardi — ya'ni yettinchi provayderni qo'shish **to'rtta ro'yxatni**
tahrirlashni talab qilardi. Bittasini unutish bu yerdagi eng jim nosozlik:
provayder panelda chiqadi, ega loginini yozadi, saqlanadi — va fayl qiluvchi kod
bo'sh kalitlarni o'qib **hech nima yubormaydi**. Buni hech kim inspektor chek
so'ramaguncha bilmaydi. Endi to'rt joy ham bitta `drawers()` dan o'qiydi, va
testi bor: paneldagi har bir provayderning saqlash joyi bo'lishi shart.

**⚠️ Eski panelning saqlashi ishlaydigan kalitlarni o'chirmasligi kerak.** Panel
avval har provayderga alohida maydon yuborardi, endi bitta map. Deploy'dan
keyingi bir necha daqiqada brauzer tabi hali eski panelni ishlatadi — server
uning shaklini e'tiborga olmasa, o'sha tabning keyingi saqlashi ishlab turgan
kalitlar ustiga bo'sh yozadi, jimgina, va restoran ro'yxatdan o'tishni to'xtatadi.
Eski maydonlar **muzlatilgan** holda qoldi: yangi provayder u yerga
qo'shilmaydi, aks holda u yozilgan kunidayoq o'lik bo'ladi.

⚠️ **API'lar ochiq emas.** Rahmat marketing sahifasida «ochiq API» deyiladi,
lekin spetsifikatsiya nashr qilinmagan (2026-08-30 da tekshirildi: rhmt.uz,
epos.uz, arca.uz — hech birida developer hujjati yo'q, faqat support telefoni).
Endpointni taxmin qilish — kompilyatsiya bo'ladigan, review'dan o'tadigan va
ishongan restoranda **bironta ham chek yubormaydigan** kod. Adapter shartnoma
hujjati kelganda yoziladi.

**⚠️ Berilgan uchta hujjat — uchalasi ham Multikassa/Multibank (2026-08-30)**

Ega uchta havola berdi va ular RahmatPOS deb atalgan edi. O'qib chiqilgach:

| Havola | Aslida nima | Holati |
|---|---|---|
| `documenter.getpostman.com/view/6027358/…` | **Multikassa.Pos** — kassa kompyuteridagi lokal API | `multikassa.go` da **allaqachon bor** |
| Drive PDF «методы виртуальной кассы» | **Multikassa Operations API** — kod izohida keltirilgan o'sha integrator PDF'i | `docs/multikassa-operations-api.txt` ga saqlandi |
| `documenter.getpostman.com/view/11774612/…` | **Multibank.Касса** — bulutli platforma: cheklarni, to'lovlarni, statistikani **o'qish** va nomenklatura boshqaruvi | Fiskalizatsiya emas — alohida xususiyat |

⚠️ **Uchinchisi chek fayl qilmaydi.** Unda `fiscal_operations`, `terminal_receipts`,
`my_cashboxes`, `receipt_template`, kassirlar, nomenklatura bor — ya'ni **o'qish
va boshqaruv**. "Chekni ro'yxatdan o'tkazish" endpointi yo'q. Uni fiskal adapter
deb ulash — hech qachon chek yubormaydigan integratsiya.

⚠️ **RahmatPOS bulutli virtual kassasining, QPOS'ning va Arca'ning API'si bu
hujjatlarda yo'q.** Ular ro'yxatda `Ready: false` bo'lib qoladi.

**Hujjat bitta haqiqiy kamchilikni ochdi: qaytarish asl chekni nomlamasdi.**

PDF aniq aytadi: `type = 4` qo'shimcha ravishda `receipt_sale_id` va
`RefundInfo {TerminalID, ReceiptSeq, DateTime, FiscalSign}` olib yuradi, va bu
blok fiskal drayverga uzatiladi. Bizning adapter esa faqat `type: 4` va
qatorlarni yuborardi.

⚠️ Natijasi ikki xil bo'lishi mumkin edi va ikkalasi ham yomon: kassa operatsiyani
rad etadi, **yoki** uni mustaqil manfiy sotuv sifatida qabul qiladi — bizning
hisobimiz to'g'ri chiqadi, davlatning nusxasida esa hech nimaga qarshi
qaytarish qoladi. Endi ikkala imlo ham yuboriladi (PDF'ning `RefundInfo` bloki
va Postman'ning yassi `receipt_gnk_*` maydonlari), va **fiskal belgisiz
qaytarish umuman qurilmaydi** — rad etish kassa ekranida mehmon turganda
ko'rinadi, noto'g'ri fayl qilish esa inspeksiyagacha hech kimga ko'rinmaydi.

⚠️ `RefundInfo` ichidagi maydonlar bu API'ning qolgan hamma joyidan farqli
o'laroq **CamelCase**, va `DateTime` formati `YYYYMMDDHHMMSS` — boshqa har bir
vaqt maydoni `2006-01-02 15:04:05`. Aynan shuning uchun blok qo'lda yozilgan.

**Qaytarish endi haqiqatan fayl qilinadi (ilgari umuman qilinmasdi).**

Adapter to'g'rilangach ikkinchi, kattaroq bo'shliq ko'rindi: `AdminRefundCheck`
pulni qaytarardi va `check.refund` ni yozardi — **fiskal qaytarishni esa hech
kim yubormasdi**. Ya'ni soliq qo'mitasining nusxasida sotuv turaverardi,
qaytarish esa yo'q edi.

- **Ikkita hujjat, ikkita yozuv**: `order.fiscal` (sotuv) va
  `order.fiscalRefund` (qaytarish). ⚠️ **Qaytarish sotuvning yozuvini
  almashtira olmaydi**: aynan sotuvning fiskal belgisi qaytarish nimaga qarshi
  ekanini aytadi — uni ustiga yozish qaytarishning o'z asosini o'chirish
  demakdir.
- **Navbat: avval sotuv, keyin qaytarish** (`nextPendingFiling`,
  `pendingReversal`). Fayl qilinmagan sotuvning belgisi yo'q, ya'ni uni
  nomlaydigan qaytarish ham qurilmaydi — teskari tartib har qaytarishni
  sotuvgacha muvaffaqiyatsiz qaytarardi.
- **Fayl qilinmagan sotuv qaytarilmaydi.** Davlat ko'rmagan chekka qarshi
  qaytarish yuborish — kassa rad etadigan va ega ertalabini yo'qotadigan hujjat.
- ⚠️ **Qaytarish menejerni hech qachon to'sib qo'ymaydi**: `pending` deb
  belgilanadi va relay yoki kassa ekrani uni oladi. Pul qarori mehmon oldida
  turgan odamniki — u ushlab turgan noutbukdan kassaga yetib borish mumkinmi
  yoki yo'qmi, bu qarorning qismi emas.
- ⚠️ **Qaytarishda mehmon cheki qayta chop etilmaydi.** Qog'oz — sotuvning
  cheki; uni qaytarishning belgisi bilan qayta bosish odamning qo'liga
  **ikkinchi sotuvga o'xshagan** hujjat berish demakdir. Qaytarishning qog'ozini
  kassaning o'zi chiqaradi.
- **Fayl qilinmagan qaytarish ham ogohlantiradi** (`unfiledFiscalFilter`) va
  **Z-hisobotni ham to'sadi** (`anyUnfiledFilter`). Bu bir teshikning ikkinchi
  tomoni: sotuv kun yakunida qoladi, kassadan chiqib ketgan pul esa yo'q.
  ⚠️ Ogohlantirishning 5 daqiqalik muhlati qaytarish uchun `refund.at` dan
  sanaladi, `check.closedAt` dan emas — chek bir hafta oldin yopilgan bo'lishi
  mumkin, va yopilishdan sanash har qaytarishni tug'ilishi bilan kechikkan
  qilardi.
- **Javob qaysi hujjatniki ekani buyurtmadan o'qiladi**, relay yoki kassa
  aytganidan emas: javob faqat buyurtma id'sini olib yuradi, buyurtma esa
  ikkitasidan qaysi biri yo'lda ekanini biladi.

**E-POS Mobile — uchinchi adapter, va u telefon** (`docs.epos.uz`).

E-POS uchta mahsulot sotadi, ulardan faqat bittasi bizning dasturimiz uchun chek
fayl qiladi:

| Mahsulot | Nima | Biz uchunmi |
|---|---|---|
| **E-POS Mobile** | Android ilova telefonni kassaga aylantiradi, o'sha telefonda **lokal HTTP API** (`:8765`) | ✅ shu ulandi |
| E-POS Fiscal Bridge | mavjud kassa dasturini E-POS platformasiga ko'chirish | Migratsiya, integratsiya nuqtasi emas |
| E-POS Cashdesk | brauzerdagi kassa ish o'rni | Hali chiqmagan |

- **Lokal**, Multikassa va REGOS kabi — lekin bu safar tom ma'noda telefon.
  Manzil — telefonning restoran Wi-Fi'sidagi manzili; hujjatdagi `localhost`
  telefonning o'z nuqtai nazaridan yozilgan va biz uchun **hech qachon
  ishlamaydigan yagona manzil**. ⚠️ Telefon binodan chiqib ketsa fayl qilish
  to'xtaydi — bu provayder ro'yxatidagi izohda ataylab yozilgan.
- Autentifikatsiya: `X-API-Key` (ilovada Profil → Lokal server).
- ⚠️ **Ikkita masshtab, ikkalasi ham jim**: pul tiyinda, **miqdor mingdan
  birda** (`amount: 1000` = bitta porsiya) — REGOS bilan bir xil kodlash, lekin
  pul summasiga o'xshab ketadigan nom ostida.
- ⚠️ **`ofdSent: false` — muvaffaqiyatsizlik EMAS.** OFD serverlari yiqilsa ilova
  chekni baribir chiqaradi: mehmonda qog'oz bor, fiskal modul belgi qo'ygan,
  javobda belgi keladi. Faqat OFD'ga yetkazish qoladi va ilova buni o'zi
  `/receipts/send-unsent` orqali qayta yuboradi. Buni xato deb o'qish bizning
  qayta urinishimizni **ikkinchi chek** fayl qilishga jo'natardi — restoran bir
  sotuvdan ikki marta soliq to'laydi va buni faqat qog'ozbozlik bilan yechadi.
  Shuning uchun hukmni **belgi** chiqaradi, `ofdSent` emas.
- ⚠️ **`vat` — qatorniki yoki bir birlikniki? Hujjat qarama-qarshi.** E-016
  qoidasi `price × vatPercent / (100 + vatPercent) ±100` deb yozilgan (birlik
  narxi), E-010 esa `totalVAT` qatorlar `vat` yig'indisiga teng bo'lishini
  talab qiladi (bu faqat qatorniki bo'lsa to'g'ri chiqadi). Namunada miqdor
  bitta, ya'ni u ajratmaydi. **Qatorniki yuboriladi** — to'g'ri soliq hujjati
  beradigan yagona o'qilish, va noto'g'ri tanlov **baland** yiqiladi: ikkita
  narsa sotilgan birinchi chekda E-016 kassir oldida rad etadi, inspeksiyada
  emas.
- ⚠️ **`units` — kod maydoni aniqlanmagan.** Hujjat OKEI deydi, namunada
  `1372873` (na OKEI, na davlat chek formatining kichik kodi — bizdagi qiymat
  shu). Unga validatsiya qoidasi biriktirilmagan, ya'ni chekni rad etmaydi —
  narx to'g'ri, qog'ozda birlik nomi noto'g'ri chiqadi. Bir qatorlik tuzatish.
- Smena: `POST /z-report/open|close`. ⚠️ `F-002` — «smena ochilmagan», va u
  **provayder bo'yicha** taqsimlanadi (`NeedsShift`): Multikassa buni `#2D` deb
  yozadi. Bir provayderning kodini boshqasining javoblariga solishtirish
  begona rad etishni avtomatik smena ochishga aylantiradi — allaqachon ochiq
  bo'lishi mumkin bo'lgan kun uchun soliq qo'mitasiga hujjat yuboradi.
- Qaytarish: `refundInfo` **majburiy** (`E-007`) — biz endi shundoq ham
  yuboramiz. ⚠️ `dateTime` formati `YYYYMMDDTHHmmss` (harfli `T` bilan), holbuki
  Multikassa **aynan shu lahzani** hech qanday ajratgichsiz yozadi.

**Yon ta'sir: paneldagi kalit maydonlari `local` dan emas, provayderdan olinadi.**

Panel «lokal kassa — hisob yo'q» deb o'ylardi. Bu **bitta** provayder haqida
rost edi: Multikassa hech kimni autentifikatsiya qilmaydi (uni bino tashqarisidan
yetib bo'lmasligi himoya qiladi). REGOS ham lokal va **login/parol** so'raydi,
E-POS ham lokal va **token** so'raydi.

⚠️ Hech qayerda xato chiqmasdi: provayderni tanlash, saqlash va yoqish
mumkin edi — u shunchaki hech qachon autentifikatsiya qilinmasdi, chunki
kalitini turadigan maydon **ekranda yo'q edi**. Endi har provayder o'zi nimani
so'rashini aytadi (`Info.Needs`), va testi bor: **har bir tayyor provayder
o'z tortmasi ko'rsatadigan maydonlardan qurila olishi shart**.

⚠️ Manzil namunasidagi port ham shu qoidaga bo'ysunadi: faqat hujjatda
**o'qilgan** portlar chiqadi (Multikassa 9090, E-POS 8765), qolganida `PORT` —
noto'g'ri javob emas, savol.

### Markirovka (Asl Belgisi) — ichimliklar
Tafsiloti va manbalari `docs/markirovka.md` da; bu yerda qarorlari.
- ⚠️ **Alohida "Asl Belgisi API" yo'q va kerak emas**: kod **fiskal chek
  ichida** ketadi (`label` maydoni), OFD uni milliy tizimga uzatadi. Ya'ni bu
  butun xususiyat — mavjud quvurga **bitta maydon**.
- **Bayroq mahsulotda** (`menu_item.marked`), qatorда emas. ⚠️ Bo'sh — "yo'q",
  shuning uchun migratsiya yo'q. Bu barda "ochilgan shisha" savolini ham
  yopadi: butun shisha va stakan — menyuda ikki xil taom.
- ⚠️ **Kod qatorga muzlatiladi** (`OrderItem.MarkCode`) — bu ИКПУ qoidasining
  **teskarisi**. ИКПУ mahsulot haqidagi fakt, shuning uchun menyudan o'qiladi
  va buxgalterning tuzatishi to'lanmagan buyurtmalarga yetadi. Bu esa
  **berilgan shisha** haqidagi fakt: menyuda tuzatiladigan hech nima yo'q, va
  sotilgandan keyin o'zgargan kod hech kim sotmagan mahsulotni muomaladan
  chiqarardi.
- ⚠️ **Takror kod butun chek bo'yicha tekshiriladi** (`marking.CheckAll`), bir
  qator bo'yicha emas: pistolet chiyilladi, o'qigani noaniq, yana skanerlandi —
  ikki qator bitta shishani chiqaradi. **Fiskal kassa bunday chekni qabul
  qiladi**, ya'ni ushlanadigan yagona joy — bizniki.
- ⚠️ **Markirovkalangan qator birlashmaydi.** Bir tile'ni to'rt marta bosish
  to'rtta emas, bitta to'rtlik qator bo'lishi — kassaning asosiy qulayligi, va
  bu yerda **noto'g'ri**: ikki shisha — ikki kod.
- **Ikki darvoza**: qator qo'shishda (shisha qo'lda, skaner ikkinchi qo'lda) va
  chek yopishda (qator bayroqdan oldin qo'shilgan bo'lishi mumkin; qonun chek
  haqida). Oflaynda ham kod qatorда saqlanadi va sinxronizatsiyada ketadi.
- ⚠️ **`omitempty` majburiy**: markirovkalanmagan taomda `label` **umuman
  ketmasligi** kerak — bo'sh satrni "markirovkalangan, kodi yo'q" deb o'qigan
  kassa chekni **mehmon oldida** rad etadi. ИКПУ bilan bir qoida.
- **Skaner — HID klaviatura**: drayver ham, qurilma API'si ham yo'q, kod
  "yozilgan matn" bo'lib keladi. Shuning uchun dialog — fokusdagi maydon, va
  ajratgichning uch xil yozilishi normallashtiriladi (firmware sababli rad
  etilgan kodni kassir tuzata olmaydi). ⚠️ `01` prefiksi **DataMatrix'ni
  yonidagi EAN-13 dan** ajratadi.
- ⚠️ **Ochiq: yetkazib berish buyurtmasi** (kod kerakmi — buxgalter/Asl Belgisi
  javobi) va **`label` maydonining aniq nomi** (davlat formatidan olingan,
  provayder hujjatlari bilan tasdiqlanmagan; har adapterda bitta qator).

### Panel navigatsiyasi biznes turiga qarab

- ⚠️ **Qaysi qator ko'rinishi — `needs`, qaysi tartibda — `SHOP_STOCK_ORDER`,
  qanday nomlanishi — `navLabel`.** Uchalasi `frontend/src/lib/adminNav.ts` da,
  layoutning ichida emas: javobni tekshirishning yagona yo'li «o'sha turdagi
  biznes bo'lib kirish» edi, va noto'g'ri javob barcha dorixona va kiyim
  do'konlarida bir vaqtda noto'g'ri edi. Testi yonida.
- ⚠️ **Bu ko'rinish, ruxsat emas.** Qatorni yashirish — xushmuomalalik; kim
  qaysi sahifani ocha olishini server hal qiladi (`handlers/panelgate.go`), va
  manzilni terib kirgan do'kon o'sha ekranni topadi.
- ⚠️ **Do'konning ombori — oshxonaniki emas, va buni tartib aytadi.**
  `NAV_GROUPS` oshxona tartibida yozilgan (javonda nima bor → nima olish kerak
  → nima qanchaga → porsiyaga nima ketadi). Do'kon uni yuqoridan pastga o'qib,
  o'zi eng ko'p ochadigan uchta ekranni (yorliq, markirovka, muddat) eng
  pastdan, oyiga bir marta tegadigan hujjatlar tagidan topardi. Hech nima xato
  emas edi — shunchaki birovning ish tartibi.
- ⚠️ **Tartib — ro'yxat, sahifalar ro'yxati emas.** Qator bor-yo'qligini `needs`
  hal qiladi (sababi o'sha yerda yozilgan); bu yerda ikkinchi «a'zolik
  ro'yxati» bo'lsa, ikkisi bir kun kelishmay qoladi va natija — sababi hech
  qayerda yozilmagan holda yo'qolgan ekran.
- ⚠️ **Gulchi — kompozitsiya qiladigan do'kon**, va u birinchi variantni
  buzdi: `techCards` va `production` tartibda nomlanmagani uchun eng pastga
  tushib qolgandi — aynan shu shikoyat, faqat eng muhtoj do'kon tomonidan.
  Test endi uchala do'kon turini ham tekshiradi.
- ⚠️ **«Masalliqlar» — oshxonaning so'zi.** Ekran bitta (do'kon tovari ham
  ombor qatori), lekin so'z dorixonaning paratsetamol javoni tepasida turardi.
  Odam bunday paytda «so'z erkin ishlatilgan ekan» demaydi — «bu panel boshqa
  kimdir uchun qilingan» deydi. Shuning uchun `goodsList: "Tovarlar"` va
  chiqim maydonining misollari ham («xodimlar ovqati» → «muddati tugadi»).
  «Kirim», «Chiqim», «Inventarizatsiya» esa ombor so'zlari — ikkinchi tarjima
  qilinmaydi, aks holda ikki lug'at ajraydi.
- ⚠️ **Yangi bo'limlar iconsiz chiqqan edi** (`expiring`, `labels`,
  `marking`) — o'n bir qatorli ustunda bu «bo'limga tegishli bo'lmagan uchta
  qator» bo'lib o'qiladi, holbuki do'kon uchun bo'limning o'zi shu. Sana,
  narx yorlig'i va skaner. `suppliers` ham yuk mashinasi edi — kirim bilan
  bir xil shakl; endi qo'l siqish (yuk mashinasi — yetkazish, yetkazib
  beruvchi — telefon qiladigan odam).
- ⚠️ **Sozlamalar ichidagi bo'limlar ham**: do'konda **oshxona cheki**
  ko'rsatilmaydi (uchta chek o'rniga ikkita, matni ham «ikkita chek» deydi) va
  printerga «Oshxona» turi taklif qilinmaydi; restoranga esa «Yorliq» turi
  taklif qilinmaydi. ⚠️ **Ma'lumot shablondan ustun**: allaqachon yoqilgan tur
  baribir ko'rinadi — aks holda yoqilgan sozlamani o'chiradigan katakcha
  qolmaydi (do'kon ichidagi nonvoyxona — haqiqiy narsa).
  Qoidalar `printKindsFor` / `receiptKindsFor` da, o'sha faylda, testi bilan.
- ⚠️ **Bir necha matn umuman neytral qilindi** (yangi kalit qo'shmasdan):
  «Restoranda ishlab turgan kassa» → «Sizda ishlab turgan kassa»,
  «restoraningizga to'g'ri kelmasa» → «biznesingizga», «Oshxonaga necha daqiqa
  oldin» → «Tayyorlashga», «oshxona ekraniga chiqadi» → «tayyorlash ekraniga
  (KDS)», «faqat restoran ochiq bo'lgan vaqtlar» → «faqat ish vaqti».
  Bular do'kon uchun to'g'ri, restoran uchun esa **aniqroq** — masalan gulchi
  ham `composes`, ya'ni oldindan buyurtmani u ham ko'radi.
- ⚠️ **Sozlamalar sahifasining tab'lari ham shu qoida bo'yicha**
  (`settingsTabs`, `settingsTabLabel` — o'sha faylda, chunki savol bitta):
  do'konda «Restoran» → **Do'kon**, «Zal va buyurtma» → **Kassa va javon**
  (do'kon uchun u bo'limda tarozi, markirovka, ballar va kassa ekrani turadi —
  zal umuman yo'q), va tartibda kassa ikkinchi, yetkazish oxiriga yaqin
  (do'konda yetkazish standart holda **o'chiq** — `BusinessType.Defaults` —
  ya'ni uning tab'i har dorixonada uchinchi bo'lib turardi). «Restoran
  profili» sarlavhasi va «Restoran nomi» maydoni ham do'kon so'zi bilan.
  ⚠️ Birinchi tab ikkalasida ham bir xil: brend yuklanmasdan chizilgan sahifa
  o'z joyida ochilsin, keyin sakramasin. ⚠️ Ikkala tartibda ham **hamma tab
  bor** — tushib qolgani manzili yo'q, ochib bo'lmaydigan sahifa bo'lardi.
- ⚠️ **Masofa «restorandan» emas, filialdan o'lchanadi** — matn shunday tuzatildi
  (uch tilda). Bu do'kon uchun to'g'ri so'z, restoran uchun esa **aniqroq**:
  masofa filialning xaritadagi nuqtasidan olinadi, brendning nomidan emas.
- ⚠️ **Tanilmagan biznes turi — restoran, va u shunday emas edi.**
  `lib/types.ts` dagi har bir predikat o'zicha qaytardi (`!t`, `t === "..."`),
  ya'ni yangiroq konsol yozgan brend hammasiga **yo'q** deb javob berardi:
  stol ham, oshxona ham, texkarta ham, do'kon ekranlari ham yo'q — panel
  chizishi mumkin bo'lgan eng bo'sh yon panel, eng yangi mijozda. Go tomonida
  bu allaqachon `known()` bilan tuzatilgan va izohi ham yozilgan edi; panel
  qolib ketgan. Endi `known()` frontendda ham bor.

### Biznes turlari: pishiradigan peshtaxta va narx pog'onasi

- ⚠️ **Ro'yxat o'n ikkitaga chiqdi** (`models/businesstype.go`): restoran, fast
  food, **qahvaxona, nonvoyxona, qandolatxona**, oziq-ovqat, **go'sht do'koni**,
  kiyim, **kosmetika**, gul, dorixona, **xo'jalik mollari**. Tartib — oshxonalar
  oldin, do'konlar keyin: mijoz yaratayotgan odam «pishiradimi yo'qmi» degan
  savolga ro'yxatni ochishdan oldin javob bergan bo'ladi.
- ⚠️ **«Sotgani = olgani» (`SellsGoods`) va «arzon pog'ona» — ikki xil savol,
  va ularni bitta ro'yxat qilish nonvoyxonada xato bo'lardi.** Nonvoyxona,
  qahvaxona va qandolatxona sotadigan narsa bir soat oldin un edi: ular
  **kompozitsiya qiladi** (texkarta, partiya, tannarx — gulchining darsi yana
  uch marta), lekin peshtaxta sifatida do'kon narxida sotiladi. Katalog savoli
  tenantda (`SellsGoods`), narx savoli konsolda (`billing.shopLadder`), va
  ikkinchisi birinchisining ustiga uch turni qo'shadi. Birini ikkinchisi deb
  o'qish — yo non pechiga 450 000 hisoblash, yo o'z nonining tannarxini
  tortib olish.
- ⚠️ **«Buyurtmadan keyin tayyorlanadi» — oshxona ekranining sharti, «pech bor»
  emas.** Nonvoyxona va qandolatxona eshik ochilmasdan **partiya** bilan
  yopadi: ularning hujjati — `production`, KDS emas, va KDS u yerda kun bo'yi
  bo'sh turadi. Qahvaxona esa teskari isbot: hech nima pishirilmaydi, lekin har
  stakan so'ralgandan keyin tayyorlanadi.
- ⚠️ **Namuna katalogi ham ikkiga bo'lindi** (`seed/shop.go`). Do'konning
  namunasi `SellsItself: true` bilan yoziladi (o'zi ombor qatori); pishiradigan
  peshtaxtaniki — **oddiy menyu qatori**, chunki aks holda ustaning yozadigan
  birinchi texkartasi mahsulotni o'ziga ko'rsatardi. Restoranning 48 taomi esa
  nonvoyxonada dorixonadagi bilan bir xil xato.
- ⚠️ **Tarozi biznes turiga qarab yoqilmaydi** (go'sht do'konida ham).
  `scale.enabled` — jimgina pul xatosi beradigan sozlama (qarang «Yorliq»):
  namuna narxlari kilogrammda, lekin prefiksni odam o'zi yoqadi.
- ⚠️ **Konsoldagi ro'yxat endi bitta** (`keel-site/src/lib/biz.ts`): forma ham,
  «Biznes turlari bo'yicha» bloki ham shundan o'qiydi. Ilgari ikkita ro'yxat va
  ikkita lug'at bor edi — yangi tur bittasida ko'rinib, ikkinchisida
  «Restoran» bo'lib o'qilardi: xato ham, bo'shliq ham emas, shunchaki nima
  ko'p sotishni hal qiladigan ekranda restoranlar orasiga qo'shilgan nonvoyxona.
- ⚠️ **Har bir tur uchun alohida blok, mijozi yo'q turlar ham.** Mijozi yo'q
  tur — topilmaning o'zi («biz bu turni sotamiz, hech kim yo'q»), va blokni
  umuman chizmaslik shu savolni ko'rinmas qiladi. Bo'sh blokda oltita nol
  emas, bitta jumla: nol statistikalar yuklanmagan ekran bo'lib o'qiladi.
- ⚠️ **Muddat tugmalari yonida ikkita sana** — server `overviewWindow` ni,
  ya'ni umumiy ekran bilan **bitta** rezolverni ishlatadi (7 kun ikki ekranda
  bir xil 7 kun bo'lsin), va «(N kun buyurtmalardan)» yozuvi qo'lda yozilgan
  davrdan **hisoblab** olinadi, so'rovdan emas: aks holda ikki haftalik oyna
  «30 kun» deb imzolanardi.

### Buxgalteriyaning ikki eshigi: Didox (ЭСФ) va 1C

Ikkalasi ham bitta sababdan: **bir fakt ikki marta terilyapti**. Yetkazib
beruvchining elektron schyot-fakturasi krujkalar eshikdan kirgan tongda
Didox'da turadi — yigirma qator, miqdor, narx, NDS — va omborchi uni bir
tabdan o'qib, ikkinchisiga qo'lda kiritadi. Oyning sotuvi bizda tayyor —
buxgalter uni 1C'ga qo'lda ko'chiradi. Ikkala ko'chirish ham yangi ma'lumot
yaratmaydi: ular bitta raqamning **ikkinchi nusxasini** yaratadi.

- ⚠️ **Ikki integratsiya, bitta ekran emas.** Didox — **hujjat**: yuridik
  dalil, ЭЦП bilan imzolanadi, kontragent ham xuddi shu qog'ozni ko'radi.
  1C — **buxgalteriya**: buxgalterning ishchi nusxasi, o'z-o'zidan yuridik
  kuchi yo'q. Ularni bitta tugma ortiga qo'yish — qulaylik bilan soliq
  hujjatini bitta kalitga bog'lash.
- ⚠️ **Biz hech nimani imzolamaymiz, va hech bir ekran imzoladik demaydi.**
  Har bir Didox imzosi odamning o'z kompyuteridagi E-IMZO kaliti bilan
  yasaladi; serverda kalit yo'q va bo'lmasligi kerak. Statusni imzosiz
  o'zgartiradigan «Qabul qildim» tugmasi — Didox'da imzolanmagan, soliq
  hisobotida yo'q hujjat, ekranda esa «bajarildi». `internal/didox` da `Sign`
  metodi **yo'qligi** — dizayn, va uning testi bor.
- ⚠️ **Bizdagi nusxa hech qachon haqiqat emas.** `edi_document` — ro'yxat va
  bog'lanish uchun oyna; status esa oxirgi tortishdan beri qanday bo'lsa
  shunday. `docId` **unique**: ikki tortish ustma-ust tushsa bitta hujjat ikki
  qator bo'lardi, va omborchi ikkitasidan qaysi biri nusxa ekanini ajrata
  olmaydi.
- ⚠️ **Bitta hujjat — bitta kirim, bir marta.** Ikkinchi import — o'sha
  javondagi ikkinchi kirim: qoldiqda ikki marta sanaladi, hisobotda ikki marta
  to'lanadi, va ikkalasi hamma maydonda bir xil bo'lgani uchun keyin ajratib
  bo'lmaydi. Tekshiruv **yozuvning filtri ichida** (`purchaseId: {$exists:
  false}`), ya'ni ikki ekrandan bir vaqtda bosilgan ikki tugma ham ikkita
  kirim yasay olmaydi.
- ⚠️ **Qatorni mos qo'yishni server taklif qiladi, odam tasdiqlaydi.** Nomlar
  ikki xil kompaniya tomonidan yozilgan — bu to'qqiz martadan sakkiztasida
  to'g'ri, bittasida xato bo'ladigan taqqoslash, va o'sha bittasi go'shtni
  sariyog' javoniga jimgina yozib qo'yadi. Taxmin **aniq nom** bo'yicha, hech
  qanday «boshlanadi», «o'xshaydi» yoki sinonim yo'q. Bog'lanmagan qator
  **o'tkazib yuboriladi**, taxmin qilinmaydi.
- ⚠️ **Yetkazib beruvchi STIR bo'yicha topiladi, nom bo'yicha emas** — bitta
  kompaniya uch qator bo'lib ketishi aynan shu (`models/supplier.go`).
  Shuning uchun `supplier.tin` qo'shildi.
- ⚠️ **Kirimning sanasi — hujjatning o'z sanasi**, bugungi emas
  (`models/purchase.go`), va u **local** o'qiladi: UTC bo'lsa har tongdagi
  faktura Toshkentda oldingi kechaga tushadi.
- ⚠️ **Jami — hujjatning o'z summasi** (NDS bilan), bizning yaxlitlangan
  qatorlarimizning yig'indisi emas: faktura pul haqida to'g'ri, qatorlar narx
  haqida to'g'ri.
- ⚠️ **Summalar JSON'ga satr sifatida yoziladi.** Go o'n million so'mni
  `1e+07` deb yozadi — bu haqiqiy JSON va **noto'g'ri faktura**: qabul qiluvchi
  tomon strukturani qat'iy tekshiradi va hujjatni butunlay rad etadi, xato esa
  bizning ekranimizda normal ko'rinadigan maydonni ko'rsatadi. Testi bor.
  Ishlatilmaydigan obyekt `{}` emas, **`null`**; sana faqat `yyyy-MM-dd`.

**1C tomoni:**

- ⚠️ **Almashinuvni doim 1C boshlaydi, biz hech qachon emas.** Buxgalterning
  1C'i ofisdagi kompyuterda, internetdan ko'rinmaydi — e'lon qilingan protokol
  aynan shuning uchun shunday (`docs/vendor/1c-exchange.md`). 1C'ga «itarib»
  yuboradigan dizayn demonstratsiyada ishlaydi va hech bir haqiqiy ofisda
  ishlamaydi.
- ⚠️ **Javob — oddiy matn, birinchi so'z hamma narsa** (`success`, `progress`,
  `failure`). Bu fayldagi handlerlar ataylab `httpx` ishlatmaydi: JSON xato 1C
  uchun «failure, sababi — mana bu JSON» bo'lib o'qiladi va buxgalter telefonda
  bizga `{"error":...}` ni o'qib beradi. Rad etish **HTTP 200 + `failure`**,
  500 emas: 500 buxgalterni tarmoq muammosi deb IT'chisiga yuboradi.
- ⚠️ **O'z logini, panel hisobi emas.** 1C parolni ofis mashinasidagi formada
  ochiq saqlaydi va ekranda ko'rsatadi; panelni ochadigan login bu yerda
  butun restoranni o'sha maydonga qo'yish bo'lardi. Cookie `checkauth` dan
  keyin beriladi, lekin Basic auth **har so'rovda ham** qabul qilinadi: ba'zi
  konfiguratsiyalar cookie yubormaydi.
- ⚠️ **Katalog yarmi — ombor moduli, va u modul darvozasi ko'rmaydigan
  eshikdan keladi.** `modulegate.go` `/admin/…` yo'llarini filtrlaydi; bu
  endpoint ataylab undan tashqarida, shuning uchun modul shu yerda **qo'lda**
  tekshiriladi. Sotuvni 1C'ga berish esa ochiq: hisobotlar hamma tarifda.
- ⚠️ **Import hech qachon o'chirmaydi.** Kam qator kelgan nomenklatura —
  ko'pincha 1C'da qolib ketgan filtr, mahsulotdan voz kechish emas; o'chirish
  texkartani, javon qoldig'ini va bir yillik kirim tarixini olib ketardi.
  Narxi nol kelgan qator ham narxni **almashtirmaydi**.
- ⚠️ **Ikkala yarim ham beriladi: sotuv va kirim.** Faqat tushum — har qanday
  biznesni juda foydali ko'rsatadigan yarim daftar. Kassa cheki alohida hujjat
  emas — u `order` (`models/check.go`), shuning uchun bitta so'rov saytni ham,
  peshtaxtani ham qamrab oladi.
- ⚠️ **`mode=success` hech nimani belgilamaydi.** 1C «oldim» deydi; bizning
  sotuvimiz va kirimimiz buxgalter import qilganidan qat'i nazar mavjud.
  Bayroq qo'yish «bu sotuv bo'ldi» degan ikkinchi ma'no yaratardi.
- ⚠️ **XML element nomlari ruscha va aynan shunday yoziladi.** 1C ularni matn
  bo'yicha o'qiydi: bitta harf xato bo'lsa hujjat **jimgina** tashlab ketiladi
  — na ularda xato, na bizda, faqat «sotuvlar kelmadi» degan buxgalter. Testi
  nomlarni ushlab turadi.

### Yorliq: do'konning o'z shtrix-kodi va tsennigi
- ⚠️ **Bu Asl Belgisi emas va bo'la olmaydi.** Davlat kodini operator ishlab
  chiqaruvchiga/importyorga beradi — dona hisobida va pullik. Qayta sotuvchi
  do'kon uni faqat skanerlaydi. Bu yerda chiqadigan narsa — do'konning **o'z**
  kodi, GS1 ichki diapazonida (`2x`).
- ⚠️ **Diapazon tarozi bilan bo'lishiladi, va xavf shu yerda.** Tarozi
  yorlig'ining prefiksi `branch.scale.prefix` da, standarti — yalang'och `"2"`,
  ya'ni **butun diapazon**. O'zimiz chiqargan kod kassada og'irlik bo'lib
  o'qilsa — kassa chiyillaydi, mahsulotni ko'rsatadi, chek bosadi va **hech kim
  tortmagan miqdorga** pul oladi; inventarizatsiyagacha bilinmaydi. Shuning
  uchun nomzod kod **o'sha filialning o'z o'quvchisiga** berib ko'riladi
  (`barcode.Allocate(seq, readsAsScale)`), «qaysi prefiks xavfsiz» degan qoida
  bilan emas: qoida bugun to'g'ri, tarozi sozlamasi o'zgargan kuni noto'g'ri.
  Hammasi to'qnashsa — rad javobi **qaysi sozlamani** o'zgartirishni aytadi.
- ⚠️ **Kod bosilayotgan paytda yaraladi**, oldin emas: bazada bor, paketda yo'q
  kod — hech kim skanerlay olmaydigan kod.
- ⚠️ **«Qaysi javon yolg'on gapiryapti» bayroqdan emas, bosilgan narxdan
  hisoblanadi** (`menu_item.labelPrice`). Bayroq narx tahrirlangan zahoti
  eskiradi — bu esa aynan ushlanishi kerak bo'lgan voqea. Uch sabab uch xil ish:
  kodi yo'q (sotuvni to'xtatadi), hech qachon bosilmagan, narxi o'zgargan.
- ⚠️ **Kirim yorliqni taklif qiladi, o'zi bosmaydi.** 200 paket 200 stiker bilan
  javob bersa — bu hech kim so'ramagan rulon va bir haftada o'chirilgan printer.
  Kirim olib tashlaydigan narsa — **terish**: tovar ham, soni ham allaqachon
  ma'lum.
- ⚠️ **Shtrix-kodni printer chizadi** (`GS k`), biz rasterlamaymiz: bitta dither
  nuqtasi noto'g'ri chiziqda — skaner o'qimaydi, do'kon esa buni «yorliq buzuq»
  deb tushunadi. Raqamlar tagida — dog' bosgan stikerni kassir qo'lda tera
  olishi uchun.
- Printerning alohida turi (`label`): ikkalasini oladigan kassa printeri narx
  o'zgarganda chek shaklidagi stikerlar tasmasini chiqarardi.
- ⚠️ **Dizaynni do'kon tanlaydi — oltitadan** (`receipt.LabelStyle`):
  `shelf` (javon yorlig'i), `price` (katta tsennik), `sticker` (shtrix-kod
  stikeri), `compact` (ixcham), `sale` (aksiya), `full` (to'liq). Sabab: yorliq
  bitta narsa emas. Javon tsennigi ikki metrdan o'qiladi, 30 mm stikerga esa
  faqat skaner qaraydi — bitta maket ikkinchisiga **noto'g'ri** bo'ladi
  (tsennik rulonni yeydi, stikerning mayda yozuvi javonda o'qilmaydi).
- ⚠️ **Tanlov karta ko'rinishida, va kartadagi rasm — qog'ozning o'zi.**
  Har dizayn alohida karta: ustida yorliqning **haqiqiy** ko'rinishi (o'sha
  belgilar to'ri, katta shrift aynan ikki barobar — `GS !` shunday ishlaydi —
  haqiqiy EAN-13 chiziqlari, tagida raqamlar, keyin bo'sh qatorlar va kesish
  chizig'i), pastida nomi va nima uchunligi.
  - Shtrix-kod **hisoblab chiziladi** (`lib/ean13.ts`): 95 modul, uchta guard
    uzunroq. Bezak sifatidagi chiziqlar noto'g'ri **enda** bo'lardi, en esa —
    do'kon o'z yorlig'iga solishtirib turgan narsa. (Bu skanerlanishiga kafolat
    emas: unga faqat sinov chiqarish javob beradi.)
  - ⚠️ **En matnga beriladi, qog'ozga emas.** Tashqi qutiga berilsa ustunlarni
    padding yeydi va butun rulon enidagi qator (narx qatori doim shunday)
    kartadan chiqib ketadi — printer to'g'ri bosayotgan bo'lsa ham karta xato
    ko'rsatardi.
  - ⚠️ **Namuna kodi haqiqiy EAN-13 bo'lishi shart.** O'ylab topilgani
    `barcode.Valid` dan o'tmaydi, ya'ni printer uni CODE128 qilib yuboradi-yu
    karta EAN chizadi. Shu ish paytida aynan shunday bo'ldi
    (`2100000000017` — check raqami 2), va buni panelning o'z testi ushladi.
    Repozitoriyda EAN ikkinchi marta o'ylab topilgan; endi ikkala tomonda ham
    testi bor.
- ⚠️ **Tanlov ko'rish orqali qilinadi.** Oltita so'zli ro'yxat qo'lida rulon
  ushlab turgan odamdan qog'ozni tasavvur qilishni so'raydi. Shuning uchun
  panel oltalasini **serverda chizib** ko'rsatadi
  (`GET /admin/labels/design`) — brauzerda qayta chizish ikkinchi maket
  dvigateli bo'lardi va farqi stikerni qo'lida ushlagan do'konda bilinardi
  (chek ko'rinishi bilan bir qoida).
- ⚠️ **`price` dizaynida shtrix-kod yo'q, va bu ataylab.** U javon tsennigi —
  paketga yopishtiriladigan stiker emas; bar narx uchun ajratilgan joyni yeydi.
  Panelga `bars` bayrog'i yuboriladi, aks holda tanlash oynasida kodsiz
  dizaynning tagida ham bar chizilardi.
- ⚠️ **`sale` narx tushmagan bo'lsa oddiy `shelf` ga o'tadi.** Javonda yo'q
  chegirmani yozish — mijoz bizdan so'rashi mumkin bo'lgan yagona xato.
  `oldPrice` — nullable maydon, va panel unga eski narxni **teng** qoldirgan
  bo'lishi mumkin.
- ⚠️ **Katta shrift narxni kesmaydi.** Ikki barobar kenglikda qatorga yarmicha
  belgi sig'adi; sig'magan narx kesilsa — qog'ozda **kichikroq** son turadi,
  ya'ni javon mijoz foydasiga yolg'on gapiradi va kassada janjal chiqadi.
  Sig'masa oddiy o'lchamda bosiladi (`bigPrice`, `centerBig`).
- ⚠️ **Standart eni — 58 mm** (cheklarda 80). Yorliq rulonining eni shu, va
  80 mm ga chizilgan maketning o'ng cheti tushib qoladi — tsennikda bu narxning
  o'zi.
- ⚠️ **Dizayn `receipt_settings` da** (`label`), filialniki — printer ham
  o'shaniki. Ikkala forma bir-birining maydoniga **yozmaydi**: yorliq ekrani
  faqat `label` ni, chek ekrani qolganini `$set` qiladi (aks holda footer
  tahriri dizaynni, dizayn tanlovi esa printerlar ro'yxatini o'chirardi).
- **Ikki jonli xato shu ish bilan tuzatildi:** (1) `cleanPrinters` ruxsat
  etilgan turlar ro'yxatida `label` **yo'q edi** — do'kon printerga «Yorliq»
  turini belgilab saqlaydi, server uni jimgina tashlab yuboradi, va har bosish
  «yorliq bosadigan printer sozlanmagan» deb javob beradi; (2) yorliq matnida
  urg'u `"!"` va `"!!"` bilan yozilgan edi — escpos markerlari esa
  `\x01`/`\x03`, ya'ni qog'ozda urg'u ham yo'q, ikkita undov belgisi bor edi.
  Ikkalasini ham test emas, **qo'ldagi qog'oz** ko'rsatardi; endi testlari bor.

### Markirovka: kirimda skanerlash
- ⚠️ **Butun xususiyat rad javobi qayerda berilishi haqida.** Tizimdan tashqarida
  olingan shisha yoki muomaladan chiqarilgan kod kassada, mijoz oldida rad
  etiladi — kassirning yagona chorasi qayta skanerlash, u esa yordam bermaydi.
  Kirimda skanerlansa, o'sha fakt **ombor xonasida**, qutisi ochiq turganda
  bilinadi.
- ⚠️ **Filial bo'yicha ixtiyoriy** (`branch.markingInbound`). Bugungi har bir
  install faqat kassada skanerlaydi; hech qachon qabul qilinmagan kodni rad
  etadigan tekshiruv **birinchi kuniyoq** har bir markirovkalangan sotuvni rad
  etardi. Yoqilgan bo'lsa — do'kon bo'lmasa ham ko'rinadi: ko'rinmaydigan
  sozlamani o'chirib bo'lmaydi.
- ⚠️ **Har shishaga bir qator, son emas** — markirovkaning butun ma'nosi shu.
  `code` **global unique**: bir kod ikki marta — yo takror skanerlash, yo
  qalbaki, ikkalasi ham odam qaraydigan narsa. Indeks — tekshiruvning o'zi,
  undan oldingi qidiruv emas: ikki kassada ikki qutini ochayotgan ikki odam
  ikkalasi ham «yo'q» deb topadi.
- ⚠️ **Sotilgani belgilanadi, o'chirilmaydi**: «bu shisha o'sha chekda ketgan» —
  keyin so'raladigan yagona savol, va o'chirilgan qator qayta skanerlangan kodni
  hech qachon kelmagan koddan ajratib bo'lmaydigan qiladi.
- ⚠️ **Qirqta shishadan bittasining stikeri o'qilmasa — bu yiqilgan quti emas.**
  Har kod alohida javob oladi (qabul qilindi / allaqachon bor / kod emas), va
  dublikatlar son emas, **kod bilan** qaytariladi: qutini ochayotgan odamning
  qo'lida o'sha shisha turibdi.
- ⚠️ **Koddagi GTIN — davlat katalogidagi mahsulot**, sizning menyungiz emas.
  Ikkalasini moslashtirish — hech kim to'ldirmagan mapping; ekran odamdan
  so'raydi va majburlamaydi.

### Ma'lumotni olib ketish (eksport) — konsol ruxsati bilan
- **Ma'lumot mijozniki va u bilan ketishi kerak**: menyusini, buyurtmalarini va
  bazasini ko'chira olmaydigan restoran mahsulot bilan emas, **chiqish narxi**
  bilan ushlab turilgan bo'ladi.
- ⚠️ **Lekin doimiy tugma emas.** Arxiv — tizim ishlab chiqara oladigan eng
  xavfli fayl: bitta faylda har mehmonning ismi, telefoni, manzili va butun
  tijorat tarixi. Doimiy tugma birovning qo'liga tushgan panel sessiyasini
  jimgina to'liq nusxaga aylantiradi.
- Shuning uchun **muddatli ruxsat** (`export_grant`, mijozning **o'z** bazasida):
  kim ochgani, **nima uchun** (majburiy sabab) va **qachongacha** (1–30 kun).
  Yopish sababsiz va darhol — xavfsiz yo'nalish to'silmaydi.
- Hujjat **tenant bazasiga** konsol tomonidan yoziladi: bitta yozuvchi, bitta
  o'quvchi, va **tenant konteyneridan control plane'ga yangi yo'l ochilmaydi**
  (aynan shu yo'lning yo'qligi bir restoranni ikkinchisiga yeta olmaydigan
  qiladi).
- **Ikki mustaqil qo'riqchi** (`handlers/export.go`): kolleksiyalar
  **allowlist**'i va har maydon nomi ustidan **naqsh bo'yicha tozalash**
  (`password|secret|token|key|hash|jwt|otp|...`). Allowlist "nima chiqishi
  mumkin" deb yozilgan: sirlar ro'yxati har provayder bilan o'sadi, biznes
  ma'lumotlariniki yo'q. Naqsh **kelasi yil qo'shiladigan** maydonni ham ushlaydi.
  `payment_settings`, `sms_settings`, `pbx_settings`, `pos_settings` va
  `phone_code` umuman chiqmaydi.
- ⚠️ Test shu ikki qo'riqchini muhrlaydi (`export_test.go`) va bir marta
  haqiqiy bo'shliqni topdi: Payme'ning maydoni shunchaki **`key`** deb ataladi.
- **Owner only**: mijozlar bazasini binodan olib chiqish smena darajasidagi
  qaror emas, va aynan menejer hisobi ko'p bo'lishiladi.
- Har yuklab olish **ikki joyga** yoziladi: tenant jurnaliga (`data.export`) va
  grant hujjatiga — "yuklab olinganmi?" savoliga javob ruxsat muddati
  tugagandan **keyin** ham qolishi kerak.
- **README uch tilda**, til `?lang=` → cookie → uz. Uch matn **alohida
  yozilgan**: uni ko'pincha boshqa kompaniyaning dasturchisi o'qiydi.
- ⚠️ **Nom brenddan olinadi, kompaniyadan emas** (`exportDisplayName`): brend
  paydo bo'lgach sozlamalar nomni brend hujjatiga saqlaydi va `restaurant.name`
  da seed qilingan **"My Restaurant"** qolib ketadi — sayti "Osh Markazi"
  bo'lgan ega `my-restaurant.zip` yuklab olardi.
- Fayl nomida **ikki yozilish**: ASCII va `filename*=UTF-8''`.
- Arxiv **JSON + rasmlar**, Mongo dump emas (maqsad — boshqa tizimga ko'chish).
  Sanalar odam o'qiydigan ko'rinishda. `README.txt` **nima yo'qligini va nega
  yo'qligini** tushuntiradi.
- Panel: `/admin/settings` → ruxsat bo'lmasa bo'lim umuman render qilinmaydi
  (`DataExport.tsx`); qoida serverda — har so'rovda rol, grant va **soat**.
  Konsol: mijoz kartochkasining oxirida (`ExportGrantPanel.tsx`).

### PIN qabul qilinadi, smena esa so'raladi (kassa va zal)
PIN «kim turibdi» degan savolga javob beradi, davomat esa «u ishdami» degan
savolga. Ilgari bu ikkisi bog'lanmagan edi: odam butun kechani sotib o'tkazishi
va bironta ham smena ochmasligi mumkin edi — bu esa oy oxirida, soatsiz payroll
qatori bo'lib chiqardi, ya'ni o'sha kechadan ancha keyin va uni boshqacha
eslaydigan odamga qarshi.

- **`StaffTillUnlock` da tekshiriladi** — odam turgan ekranda, PIN to'g'ri
  bo'lganidan **keyin**. Ochiq smena yo'q bo'lsa 409 va `needsShift: true`.
- ⚠️ **Ogohlantirish emas, to'siq.** Ishlab turgan ekrandagi lenta — bosib
  o'tiladigan lenta: birinchi mehmon allaqachon turibdi, smenani esa «bir
  daqiqadan keyin» ochasiz. Kassa smenasi darvozasi (`ShiftGate`) aynan shu
  dalilga qurilgan, bu — bir qavat oldinroq.
- ⚠️ **O'z maydoni bilan, faqat jumla emas** (`needsShift`): ekran modal oyna
  chizadi va **qayerga borishni** aytadi. Tanib bo'lmaydigan rad javobi
  padning ostidagi qizil qatorga tushardi — u yerda «PIN noto'g'ri» yashaydi,
  va bu ikkisining javobi bir xil emas (biri to'rt raqamni qayta teradi,
  ikkinchisi boshqa ekranga boradi).
- ⚠️ **Baza javob bermasa — kiritadi** (`shiftAllows`). Bu tekshiruv oldini
  oladigan zarar — soatsiz payroll qatori, keltirishi mumkin bo'lgan zarar esa
  navbat turganda ochilmaydigan kassa. Shuning uchun shubhali holat kassani
  ochadi, faqat halol «ochiq smena yo'q» yopadi. Testi bor, chunki bu
  yo'nalish jumlada emas, kodda yozilishi kerak.
- ⚠️ **PIN yo'q filialdagi zaxira yo'l** (`fallback`) bu tekshiruvdan o'tmaydi:
  u odam allaqachon haqiqiy login bilan kirgan holat, va uni ham yopish PIN
  qo'yilmagan restoranni butunlay to'xtatardi.
- ⚠️ **Filial sozlamasi, standart holatda o'chiq** (`branch.requireShift`).
  Davomatni umuman ishlatmaydigan restoranda bu tekshiruv **hamma PIN ni** rad
  etardi — navbat peshtaxtada, ekranda esa kassir qila oladigan hech nima yo'q.
  Shuning uchun qoidani deploy emas, restoranni yurituvchi odam yoqadi (filial
  formasida). Migratsiya ataylab yo'q: mavjud filiallarga majburan yoqish —
  smena o'rtasida to'xtaydigan kassa.
- ⚠️ **Modalda «Davomat ekranini ochish» tugmasi yo'q, va bu unutilgan emas.**
  Smena bu mashinadan ochilmaydi: davomat odam **qayerdaligini** tekshiradi
  (telefondagi GPS yoki filialdagi kiosk QR), peshtaxtaga mahkamlangan monoblok
  esa faqat «ha, u peshtaxtada» deb javob bera oladi. Tugma qo'yish — xonaga
  mavjud bo'lmagan yo'lni o'rgatish.

### Har bir taomning holati: tayyor va berildi
Ilgari peshtaxtada **butun chek** uchun bitta «Tayyor» tugmasi bor edi. Olti
kishilik stol faqat **oxirgi** taom bitganda «tayyor» bo'lardi — ya'ni undan
oldingi yigirma daqiqada zalga hech nima aytilmasdi, birinchi besh tarelka esa
lampa ostida sovib turardi. Oshpaz taomni bittalab tugatadi; ekranlar ham
shuni ayta olishi kerak.

Endi ikkita vaqt belgisi **qatorda** yashaydi (`OrderItem.ReadyAt`,
`OrderItem.ServedAt`), va ularni faqat bitta fayl yozadi
(`internal/handlers/dishstate.go`):

```
oshxona belgiladi  → ReadyAt   → zal, kassa va ofitsiant telefonida yashil
                                 bo'ladi, yonida «5 daq oldin tayyor bo'ldi»
ofitsiant olib bordi → ServedAt → u endi tashiladigan narsa emas
```

- ⚠️ **Bayroq emas, vaqt belgisi.** «Ikki daqiqa oldin tayyor» va «yigirma
  daqiqa oldin tayyor» — ofitsiantni ikki xil joyga yuboradigan ikki xil holat,
  bayroq esa ikkalasiga bir xil javob beradi. Ekranlar yoshini o'zi hisoblaydi
  (`timeAgo`), matni esa uch tilda.
- ⚠️ **Buyurtma darajasidagi `readyAt` endi hisoblanadi, bosilmaydi.** U chekni
  peshtaxtadan olib ketadigan va ofitsiant/kuryer bildirishnomasi osilgan
  bayroq, shuning uchun qatorlardan chiqariladi: hamma yuborilgan tirik qator
  belgilanganda qo'yiladi, bittasi qaytarib olinganda **tozalanadi** (aks holda
  oshpaz qaytarib olgan taom hech kimga, jumladan o'ziga ham ko'rinmaydi).
  Ikki joy bitta bayroqni yozsa ular ajraydi, va ajralish yo'nalishi — hech
  qachon qaytmaydigan chek.
- ⚠️ **Chek uchun bitta tugma qoldi, lekin u endi qatorlarni ham belgilaydi**
  («Hammasi tayyor»). Haqiqatan hammasi birga bitgan buyurtma uchun sakkiz
  bosish o'rniga bitta — lekin qatorlar belgilanmasa, zal «buyurtma tayyor»
  deb ko'rsatilgan, ichida esa bitta ham yashil taom yo'q chekni ko'rardi.
  Rangning bir marta yolg'on gapirgani — butun xonaning unga ishonishdan
  to'xtashi.
- ⚠️ **Qator ikki xil nomlanadi** (`dishRef`): kassa chekining qatorida
  `lineId` bor (ular tahrirlanadi, bo'linadi, ko'chiriladi — o'rni siljiydi),
  saytdan kelgan buyurtmada esa yo'q va o'rnining o'zi identifikator. Indeks
  `lineId` li qatorga **hech qachon** murojaat qila olmaydi — aks holda endi
  bo'lingan chekda belgi boshqa taomga tushardi, va oshpaz buni «ekran qo'l
  ostida siljidi» deb ko'radi.
- ⚠️ **Belgini qaytarib olish mumkin** — ikkala tomonda ham. Ho'l ekranda
  noto'g'ri taomni belgilash oddiy hol, qaytarib bo'lmaydigan belgi esa hech
  kim bosishga jur'at etmaydigan belgi.
- **«Berildi» — oshxonaning belgisi emas**: lampa ostidagi tarelka va mehmon
  oldidagi tarelka faqat «tayyor» ni biladigan ekranda bir xil ko'rinadi —
  shu sababdan bitta taom ikki marta olib chiqiladi, ikkinchisi esa umuman
  chiqmaydi. Ruxsati `PermWaiter`, va **tayyor bo'lishi shart emas**: bardagi
  choy oshxona ekranidan o'tmaydi.
- **Zal kartochkasida yashil raqam** (`readyWaiting`): peshtaxtada turgan,
  hali olib ketilmagan taomlar soni. «Yuborilmagan» nuqtasi ofitsiant nima
  qilmaganini aytadi; bu esa **oshxona nima qilib qo'yganini** — va bu aynan
  sovib qoladigan yarmi. Ilgari buni bilishning yagona yo'li borib qarash edi.

### KDS — oshxona ekrani (`/staff/kitchen`)
- **Bu kattalashtirilgan buyurtmalar sahifasi emas.** Paneldagi ro'yxat —
  **eganing** ekrani: filtrlar, cheklar, pul, mijoz tarixi. Oshpaz esa butun
  kecha bitta savolga javob beradi: **keyin nima pishiraman.** Shuning uchun
  ekranda chek, kutish vaqti va bitta tugma bor — pul, mijoz, manzil va
  filtrlar ataylab yo'q. Jamini o'qib o'tib taomni topishi kerak bo'lgan
  oshpaz ekranni o'qishni to'xtatadi.
- ⚠️ **"Tayyor" — yangi holat emas, `order.readyAt` vaqt belgisi.** Yetkazishda
  u `preparing` va `on_the_way` orasida, olib ketish va stolda esa butunlay
  boshqa joyda turadi; holat esa kuryer ilovasi, mijozning kuzatuv sahifasi,
  statistika, POS ko'prigi va uchta lug'at tomonidan o'qiladi. Yangi holat
  qo'shish faqat oshxona va peshtaxta biladigan faktni ifodalash uchun
  shularning **hammasiga** tegishni talab qilardi. Vaqt belgisi esa
  qo'shiladi: chek ekrandan ketadi, panelda "Oshxona tayyorladi" nishoni
  chiqadi, holat haqida fikr yuritadigan hech nima o'zgarmaydi.
  Buyurtma orqaga qaytarilsa (`pending`/`confirmed`/`preparing`) `readyAt`
  **tozalanadi** — aks holda hech kim pishirmagan taom "tayyor" bo'lib
  turardi va oshxona ekrani uni boshqa ko'rsatmasdi.
- ⚠️ **KDS'ga ruxsat har bir odamga alohida beriladi** (`staff.canKitchen`,
  `kitchenDenial`). Ilgari **istalgan ishchi** tokeni bilan kirish mumkin edi:
  ofitsiant ham, kassir ham, farrosh ham filialning hamma chekini ko'rardi va —
  bundan yomoni — «Tayyor» bosa olardi, ya'ni chekni peshtaxtadan yo'qotib,
  panelga «oshxona pishirdi» deb aytardi. Umumiy planshet — `staff` rolining
  butun ma'nosi, demak «logini bor» va «oshxona ekranini yuritadi» **bir savol
  emas**.
  - ⚠️ **Nol qiymat — `false`**, ya'ni bu yerda kodning odatdagi «bo'sh qiymat
    bugungi xatti-harakat» qoidasi **ataylab teskari**: standart holatda
    hammada bo'lgan ruxsat — ruxsat emas. Mavjud xodimlar buning o'rniga
    **migratsiya** bilan saqlanadi (`EnsureKitchenAccess`) — aks holda bu
    xususiyat chiqqan deploy har bir jonli oshxonaning ekranini smena o'rtasida
    o'chirardi va planshetda buning sababi yozilmagan bo'lardi.
  - ⚠️ **Lavozim maydoni ruxsat emas**: `position` — erkin matn ("oshpaz"),
    uni tizim hech qayerda o'qimaydi. Uni ruxsat deb qabul qilish kimning
    imlosi mos kelsa o'shanga kalit berish bo'lardi.
  - ⚠️ **`isActive` ham shu yerda tekshiriladi.** Ishdan bo'shatilgan odamning
    tokeni smenadan ancha uzoq yashaydi: kirish sahifasi va `StaffClock`
    tekshirardi, KDS esa **umuman tekshirmasdi** — ya'ni bugun o'chirilgan
    oshpaz kechqurun ham chekni o'qib, «Tayyor» bosa olardi.
  - **Ikki rad javobi ikki xil matn**: biri "menejerdan ruxsat so'rang", ikkinchisi
    "hisobingiz o'chirilgan" — ular odamni **boshqa-boshqa** odamga yuboradi.
  - **403, 404 emas** (filial qamrovidagidan farqli): bu odam shu yerda
    ishlaydi, planshetni ko'rib turibdi, ya'ni ekran borligini bilishi normal.
  - **Ikkala endpoint ham qo'riqlangan**, va harakat qiladigani muhimroq:
    ro'yxatni o'qish — sizib chiqish, «Tayyor» esa **o'zgartiradi**.
  - Tugma profilda faqat ruxsati borga ko'rinadi, **lekin bu faqat xushmuomalalik**:
    qoida serverda. Yashirilgan tugma — taklif, va uni birinchi bo'lib
    tugmasi yo'qolganini sezgan odam qidiradi.
- **Admin tokeni bilan emas, `staff` tokeni bilan ishlaydi.** Peshtaxtadagi
  planshet umumiy va hech qachon chiqmaydi; unda owner tokeni turishi — butun
  biznesni (sozlamalar, mijozlar, to'lovlar) javonda qulfsiz qoldirish.
- **Filial ishchidan olinadi, so'rovdan emas**: Chilonzordagi oshpaz URL'ni
  o'zgartirib Yunusobodning cheklarini ko'ra olmaydi. Amal ham xuddi shunday
  qo'riqlangan — `_id` yolg'iz hech qachon hujjat tanlamaydi, filtr ichida
  doim `branchId` bor.
- **To'lanmagan buyurtma ko'rinmaydi** (`queuedAt` bo'sh) va **`pending` ham
  yo'q**: hali qabul qilinmagan buyurtmani pishirgan oshxona ovqatni
  allaqachon sarflagan bo'ladi.
- **Eng eskisi birinchi, saralash tugmasi yo'q**: shikoyatga aylanish arafasida
  turgan chek — eng ko'p kutgani, va boshqa har qanday tartib uni ochiqda
  qoldiradi.
- **Kutish vaqti serverda hisoblanadi**: peshtaxtadagi planshetning soati
  ko'pincha noto'g'ri, ekran esa aynan shu raqam bilan baholanadi.
- **"Boshlandi" ikki marta bosilsa xato bermaydi** (filtr `confirmed` ga
  toraytirilgan, ikkinchi bosish `ok` qaytaradi): ho'l barmoq va "hech nima
  bo'lmadi shekilli" — oddiy holat, va tarixda to'rtta `preparing` yozuvi
  bittasidan yomonroq javob beradi.
- Ovoz — **panel bilan bir xil ikki nota** (880/1175): ofis va oshxona bir
  hodisaga har xil ovoz chiqarsa, kimdir bittasini eshitmaslikni o'rganadi.
  Birinchi yuklanishda chalinmaydi (planshet uyg'onganda oshxonadagi har bir
  chek uchun jiringlagan ekranning ovozi butunlay o'chiriladi).

### Qurilmaga biriktirish: bir hisob — bir telefon
⚠️ **Restoranda parol identifikator emas.** Ishchining logini ofisdagi
kartochkada yozilgan, kuryer o'zining logini smenani almashtirgan o'rtog'iga
beradi, ofitsiant esa dam olish uchun hisobini hamkasbiga uzatadi — va bu
kirishlarning **hammasi to'g'ri**, ya'ni server e'tiroz bildira olmaydi.
Server ko'ra oladigan yagona narsa — telefon o'zgargani.

Shuning uchun ilovadan kirish hisobni **o'sha o'rnatmaga** bog'laydi, va ikki
tomonlama rad etadi:

```
bir hisob — bir telefon    ikkinchi telefondan kirish rad etiladi
bir telefon — bir hisob    o'sha telefonda ikkinchi hisob rad etiladi
```

- ⚠️ **Ilova bo'yicha, umumiy emas.** Bitta odam Keel Waiter'da ofitsiant,
  Keel Team'da xodim; ega esa bitta telefonda Owner va Waiter'ni ushlashi
  mumkin. Ilovalar bo'ylab bog'lash suiiste'molni emas, **oddiy holatni** rad
  etardi.
- ⚠️ **Brauzer bog'lanmaydi, va bu e'tibordan chetda qolgan joy emas.** Panel
  uydagi noutbukdan ham, restorandagi mashinadan ham, telefon brauzeridan ham
  ochiladi — qulf to'rtta **ilova** haqida, va ular o'zini `X-Keel-Device`
  sarlavhasi bilan tanitadi. Sarlavhasiz so'rov hech nimani o'zgartirmaydi.
- ⚠️ **Parol tekshirilgandan keyin, token berilishidan oldin.** Oldinroq
  bo'lsa — begona odam login taxmin qilib restoran qaysi telefonlarni
  ishlatishini bilib olardi; keyinroq bo'lsa — ilova keyingi so'rovda rad
  etiladigan sessiyani ushlab turardi.
- ⚠️ **Baza javob bermasa kirish o'tkaziladi**: bu tekshiruv oldini oladigan
  zarar — bitta login ikki kishida; keltirishi mumkin bo'lgan zarar — ochilmay
  qolgan restoran. Kassadagi smena darvozasi bilan bir yo'nalish.
- ⚠️ **Ikkita rad javobi ikki xil matn**, chunki ular odamni ikki xil joyga
  yuboradi: «bu telefonda boshqa hisob» (telefon egasiga) va «hisobingiz
  boshqa telefonga biriktirilgan» (ofisga).
- **Chiqish bog'lanishni bekor qilmaydi** — aks holda telefonni uzatish uchun
  chiqib qo'yish yetarli bo'lardi. Bog'lanishni **panel** bo'shatadi.
- ⚠️ **Allaqachon kirgan telefon ham ro'yxatga tushadi (`adoptDevice`).**
  Bog'lash kirish paytida bo'ladi — ya'ni bu xususiyat paydo bo'lishidan
  **oldin** kirgan telefon boshqa hech qachon kirmaydi: tokeni saqlangan, har
  ochilishida `me` chaqiradi, va `touchDevice` ning `UpdateOne` i hech nimaga
  mos kelmasdi. Panelda "qurilma yo'q" deb turardi, cho'ntakda esa bir oydan
  beri ishlab turgan telefon bor edi (birinchi shikoyat — eganing o'z
  telefoni). Yagona davo — chiqib qayta kirish, va buni hech bir ekran
  taklif qilmaydi.
  Endi qatori yo'q o'rnatma **qabul qilinadi**, lekin faqat `bindDevice` ning
  ikkala savoli ham "yo'q" desa: bu hisobda shu ilova uchun qator yo'q **va**
  bu o'rnatma boshqa birovniki emas. Aks holda umumiy hisobning **ikkinchi**
  telefoni bog'lanishni jimgina, hech kim kirmagan holda o'ziga olardi — ya'ni
  qulf aynan o'zi to'sishi kerak bo'lgan narsani qilib berardi.
- ⚠️ **Panel o'z hisobingizda 404 ni yutmaydi.** Boshqa odamniki uchun 404 —
  "ko'rish huquqingiz yo'q" degan oddiy javob va bo'sh ro'yxat to'g'ri. O'z
  hisobingizda esa u faqat **so'rov yiqilgani** bo'lishi mumkin, va o'sha yerda
  "qurilma yo'q" deb yozish — hozir ishlab turgan telefonni umuman bo'lmagan
  telefonga o'xshatish.
- ⚠️ **«O'chirish» tugmasi — qulfning qo'shimchasi emas, uni xavfsiz qiladigan
  narsa.** Ilova qayta o'rnatilsa id yangilanadi, telefon yo'qolsa qaytmaydi,
  ekran juma kuni kechqurun siniydi. Ko'tarib bo'lmaydigan qulf — bu bizga
  qilinadigan telefon qo'ng'irog'i. Panelda: kuryernikida kuryer sahifasida,
  ishchinikida ishchi kartochkasida, eganiki esa **Sozlamalar → Hisob** da
  (va panel hisobining bog'lanishini faqat **ega** bo'shata oladi).
- **IP ham ko'rsatiladi**: bog'lanish o'zi «bu haqiqatan o'shami?» degan
  savolni bera olmaydi — har kuni restoran wifi'sidan, keyin boshqa shahardan
  kirgan kuryer esa beriladigan savol.
- ⚠️ **Id — apparat raqami emas**, ilovaning o'zi yaratadigan qiymat
  (`keel_device_id`, SecureStore'da). Android yillar oldin oddiy ilovalarga
  barqaror qurilma raqamini berishni to'xtatgan, va uni so'rash — bizga kerak
  bo'lmagan identifikatorni so'rash.

### To'rtta telefon ilovasi: kim nimani ko'radi
`mobile/waiter`, `mobile/courier`, `mobile/team`, `mobile/owner` — bittasi
emas, to'rttasi, va sabab bitta: **ekran kimga tegishli.** Ofitsiantda zal,
kuryerda yo'l, qolgan xodimda faqat o'z ishi (smena, davomat, ish haqi), egada
esa raqamlar. Bitta ilovaga hammasini solish har bir odamga boshqa birovning
ekranini berardi (va farroshga zal xaritasini).

⚠️ **Keel Owner — telefondagi panel emas.** Panel — o'tirib qaror qabul
qiladigan joy (menyu, narx, grafik, kampaniya); telefon — kuzatish va javob
qaytarish. Shuning uchun ilovada menyu tahriri, sozlamalar, CRM va ombor
hujjatlari **ataylab yo'q**: ular telefonda yomon bajariladi, va svetofor
oldida narx o'zgartira oladigan ekran oxir-oqibat o'zgartiradi. Ilovadagi
yagona yozuv amali — buyurtmani tasdiqlash va (sabab bilan) bekor qilish.

⚠️ **Ega qaysi turdagi xabar yuborilishini o'zi tanlaydi** (2026-09-14,
Sozlamalar → Integratsiyalar → "Shubhali holatlar haqida xabar"): 8 tur
raqamlangan ro'yxatda, har biriga checkbox. Tartib — pul chiqib ketadigan
tartib: chek bekor qilindi, hisobdan keyin olib tashlandi, katta chegirma,
kassa kamomadi, smena uzoq ochiq, ombor kamomadi, texkarta, paneldagi amal
(`models.AlertKindsInOrder`, frontend'da `ALERT_KINDS`).
- ⚠️ **O'chirilgani saqlanadi, yoqilgani emas** (`alert_settings.muted`): bu
  imkoniyatdan oldingi har restoranda ro'yxat yo'q, va bo'sh "yoqilgan" ro'yxat
  hammasini o'chirardi. Bo'sh "o'chirilgan" — ular bor narsa: hammasi.
- ⚠️ **O'chirilgan — yuborilmaydi, yozilmay qolmaydi.** Hodisa bazaga tushadi,
  panel va ega ilovasining ro'yxatida turadi; faqat Telegram ham, push ham jim.
  Tekshiruv yozuvdan **keyin**, kunlik chegaradan **oldin** — o'chirilgan tur
  boshqa turga kerak bo'lgan xabar o'rnini egallamaydi.
- ⚠️ **Sinov tugmasi filtrni chetlab o'tadi** (`deliverAlert(..., force)`):
  sinov katta chegirma sifatida yuboriladi, va chegirmani o'chirgan ega sinovni
  bosib hech narsa eshitmasa, butun kanalni buzuq deb o'ylardi.
- Noma'lum tur nomi saqlanmaydi (`mutedKinds`): bazada o'chirilgan turga
  o'xshab turib hech narsani o'chirmaydigan xato yozuv bo'lardi.
- **Beshta yangi tur** (`handlers/alertsmore.go`, har biri haqiqiy bazada
  testlangan): **yopilgan chekda pul qaytarildi** (`AdminRefundCheck`,
  chegara `refundFrom`, standart 50 000), **tayyorlangan buyurtma bekor
  qilindi** (sayt/telefon buyurtmasi, `statusHistory` da `preparing` yoki
  `on_the_way` bo'lgan yoki `readyAt` qo'yilgan — chegara yo'q), **kassadan
  katta chiqim** (`addCashEntry` — kassa ham, panel ham shu eshikdan yozadi,
  `cashOutFrom`, 500 000; kategoriya xabar mavzusi), **katta spisaniya**
  (`writeOffFrom`, 300 000; masalliq nomi mavzu), **qarzga katta summa**
  (chek qarzga yopilganda, `debtFrom`, 300 000).
  - ⚠️ **Buyurtma holati emas, tarixi o'qiladi**: ogohlantirish ishlaganda holat
    allaqachon `cancelled`, oshxonaga yetganini faqat qanday harakatlangani
    aytadi. Kutilayotgan buyurtmani bekor qilish — fikridan qaytgan mehmon, eng
    ko'p uchraydigan bekor qilish, va unga xabar kanalni o'chirtiradi.
  - ⚠️ **Kassa cheki bu turga kirmaydi** — uning o'z turi bor (pishirilgan
    qatorlar qiymati bilan, jami emas).
  - ⚠️ **Egasining o'z amali xabar bermaydi** (qaytarish, spisaniya, bekor
    qilish panelda ega qilgan bo'lsa) — "paneldagi amal" turining qoidasi:
    o'quvchini o'ziga xabar qiladigan kanal o'chiriladi.
  - Standart chegaralar ataylab baland: choy qaytarish — fikridan qaytgan
    mehmon, non mashinasiga pul — har ertalab.
  - Ega ilovasi (`owner-android`) turlar nomlarini ham oldi — yangi tur
    telefonda xom id bo'lib chiqmasin.

⚠️ **Loss alertlar endi telefonga ham boradi.** Ular `sendToOwners` orqali
**faqat Telegram**da edi, ya'ni Telegram ulamagan restoran hisobdan keyingi
olib tashlashni ham, kassadagi kamomadni ham **ko'rmasdi** — va hech nima buni
aytmasdi. Endi push Telegram bilan **yonma-yon** yuboriladi (uning xato yo'li
ichida emas: ikki mustaqil kanalni `else if` bilan bog'lash bu kodbazada bir
marta jimgina ishlamay qolgan). Faqat egalarga: menejer — bu xabarlar *haqida*
bo'lgan odamlardan biri.

- **Umumiy skelet, alohida ekranlar**: `session.ts`, `tokens.ts`, `prefs.ts`,
  `theme.ts`, `ui.ts`, `notice.tsx`, `offlinescreen.tsx`, `auth.tsx` uchalasida
  ham bir xil naqshda. Qoidalar esa `frontend/src/lib` dan **import qilinadi**,
  ko'chirilmaydi.
- ⚠️ **Ikonka uchalasida bir xil, splash esa har birida boshqa** (Waiter /
  Courier / Team). Bitta telefonda ular bitta mahsulot bo'lib ko'rinishi kerak,
  lekin qaysi birini ochganingiz — ochilish paytida beriladigan yagona savol.
  Splash `scripts/courier-splash.py` bilan quriladi (so'z argument).
- **Har birining o'z push kanali**: `kitchen`, `delivery`, `team`. Android'da
  kanalni foydalanuvchi o'chiradi, va bittasini o'chirgan odam ikkinchisini ham
  o'chirganini bilmay qoladi.
- ⚠️ **Push matni endi uchala ilovada ham tarjima qilinadi**: til token bilan
  birga saqlanadi (`staff_device.lang`, `courier_device.lang`) va xabar
  `internal/i18n` katalogidan o'tadi. Ilgari ofitsiantning «Tayyor» xabari
  faqat o'zbekcha edi — ekranlari uch tilli ilovada.
- **Keel Team nima uchun kerak bo'ldi**: davomat yagona telefonsiz qism edi
  (`/staff` veb sahifasi), va kassa endi ochiq smenasiz PIN ni rad etadi — ya'ni
  «sahifani topolmadim» «ishni boshlay olmayapman» ga aylandi.

⚠️ **Tez bosganda qotish — telefonda ham, va sabab kassadagi bilan bir xil.**
Windows kassa va zal ekranlarida bu allaqachon yozilgan («bosish ro'yxatga
olinadi»); telefonda esa shikoyat «menyuni tez-tez bosib bo'lmayapti» bo'lib
keldi. Uchta alohida sabab bor edi, va uchalasi ham «qotish» bo'lib ko'rinadi:

1. **Bosishga javob yo'q edi.** Statik uslubli `Pressable` Android'da **hech
   qanday** qaytariq bermaydi: na ripple, na so'nish — o'zgargan holat
   restoran wifi'si orqali serverdan qaytguncha ekranda hech nima bo'lmaydi.
   Odam esa yana bosadi. Endi `src/press.tsx` → **`Tap`**: ripple (uni platforma
   **UI oqimida** chizadi, ya'ni JavaScript band bo'lsa ham ko'rinadi), iOS'da
   so'nish, va standart 8pt `hitSlop` (bu ilovalar yurib, bir qo'lda
   ishlatiladi). ⚠️ **Qaytariq — ish bajarilgani emas**, va ularni ajratish
   butun gap: odamga «bosishing yetib keldi» deb aytadigan lahza aynan
   JavaScript band bo'lgan lahza.
2. **Har bosish — bitta so'rov, va ular navbatda kutardi.** Ofitsiant to'rtta
   kofe qo'shsa, to'rtta ketma-ket borish-kelish bo'lardi va o'sha ikki
   soniyada menyu **hech nima** demasdi. Endi bosishlar 180 ms yig'iladi va
   **bitta** `tillAddLines` bo'lib ketadi (endpoint o'zi shunga qurilgan), va
   raqam **bosilganda** o'zgaradi, javob kelganda emas. ⚠️ Bu **sotuv haqida
   optimizm emas**: narx, stop list va partiya cheklovi baribir serverning
   javobi, rad etilsa raqam qaytadi va serverning o'z so'zi chiqadi. Bu —
   **bosish haqida** halollik, va uni bu ekrandan boshqa hech kim ko'rmaydi.
   ⚠️ «Oshxonaga yuborish», chek chiqarish va ekrandan chiqish **avval
   buferni bo'shatadi**: yig'ish oynasi qisqa, lekin «qisqa» — «hech qachon»
   emas, va jimgina yo'qolgan taom bu fayl chiqara oladigan eng yomon natija.
3. **Menyu boshqa ish ketayotganda o'chirib qo'yilardi** (`disabled={busy}`).
   Chek chiqarayotganda butun menyu bosilmas edi — ya'ni «qotish» so'zma-so'z
   rost edi. Endi qo'shish hech qachon o'chmaydi; navbat baribir tartibni
   saqlaydi.

⚠️ **Pastdagi tugma Android tugmasining ostiga tushmaydi (`useBottomInset`).**
To'rtala ilova ham edge-to-edge chiziladi, ya'ni maket tizim panelining **ostiga**
ham cho'ziladi: pastga qo'yilgan tugma «Orqaga» va «Home» ning ortida qoladi, va
oshxonaga buyurtma yuborishga qaratilgan barmoq ekranni orqaga qaytaradi —
buyurtma yuborilmagan holda. Avval bu faqat ikki joyda hisobga olingan edi.

- ⚠️ **O'lchanadi, taxmin qilinmaydi.** Pastdan ochiladigan varaqlarda
  `paddingBottom: 34` turardi — u **jest chizig'ini** o'tadi va **uch tugmali
  panelni** o'tmaydi, ya'ni xato aynan ilova yozilgan telefonlarda ko'rinmasdi.
  Endi tizimning o'z o'lchovi olinadi; 12 — o'lchov bermaydigan telefonlar
  uchun **bo'shliq**, xavfsizlik emas.
- **`useUI()` qaytaradi** (`{ theme, s, bottom }`), chunki har bir ekran
  allaqachon uni chaqiradi: ikkinchi importni eslash kerak bo'lgan ekran —
  uni unutadigan ekran, va to'rtala ilovaning pasti aynan shundan tizim
  tugmalari ostida qolgan edi.
- Tegadigan joylar: pastdan chiqadigan varaqlar (kuryerdagi buyurtma, ofitsiant
  qatori va stol oynasi), **tab bar ko'rinmaydigan ekranlar** (login, server
  manzili, internet yo'q) va chat maydoni.

⚠️ **Egaga beshta ekran emas, beshta savol.** Owner ilovasiga qo'shilgan
narsalar «panelning yana bir bo'limi» sifatida emas, **telefonda yaxshi
bajariladigan ish** sifatida tanlandi:

- **Fikrlar (o'z tabida)** — bir yulduz kechqurun soat sakkizda keladi, panel
  esa ertalab ochiladi, va oradagi o'n ikki soatda mehmon buni boshqa birovga
  aytib bo'ladi. Shikoyatning javobi — qo'ng'iroq, telefon esa allaqachon
  qo'lda. Ekran **javobsizlaridan** ochiladi (nolgacha tushishi kerak bo'lgan
  ro'yxat), va **saytga chiqarish paneldа qoladi**: u mehmonning ismi va
  so'zlarini internetga qo'yadi, bu esa o'tirib qabul qilinadigan qaror.
- **Ertalabki brifing** — «Bugun» ning ostida, raqamlarning **tagida**: bu ekran
  kuniga yigirma marta tushum uchun ochiladi va kuniga bir marta brifing uchun
  o'qiladi. Alohida so'raladi, ya'ni tarifga kirmagan javob bugungi tushumni
  o'zi bilan tortib tushirmaydi. Til **so'rovda** yuboriladi: telefonda cookie
  yo'q, va server aks holda hammasini o'zbekcha yozardi.
- **Kim ishda** — «hozir» savoli, shuning uchun hisobotda emas, birinchi
  ekranda va bitta qatorda: nechta odam ishda, nechtasi kutilgan-u kelmagan.
  Ro'yxat — bosilganda. ⚠️ «Kelmagan» ni telefon **hisoblamaydi**, serverning
  `todayStatus` ini o'qiydi: grafik, dam kuni va kechada tugagan smena —
  ikkinchi ta'rif yozilsa, aynan shu joyda ajraydi.
- **Obuna** — sana va sanoq, **bayroq emas** («obuna faol» yarim tunda hech kim
  qaramaganda eskiradi — `provisionStatus` darsi). U **ogohlantirmaydi**: kassa
  oxirgi haftada aytadi, panelda to'liq kartochka bor, uchinchi qichqiriq esa
  uchalasini ham o'chirishga o'rgatadi. Bu yerda turishining sababi — «qachon
  to'layman?» stol yonida emas, yo'lda beriladi.
- **Kunlik yakun** — vaqt bo'yicha emas, **smena yopilganda**: soat bo'yicha
  yuborilgan yakun bir restoranda yarim kunni, ikkinchisida ochiq kassani
  yig'adi. Bir kunda **bitta** xabar (oxirgi ochiq smena yopilgani tekshiriladi
  — ikki kassali joyda yarim kun ikki marta kelmasin), va sotuv bo'lmagan kun
  umuman jim. ⚠️ Telegramga **ketmaydi**: loss alert — bir oydan keyin
  qidiriladigan yozuv, yakun esa bir kechaga rost bo'lgan gap, va uni har oqshom
  oladigan guruhdan odamlar chiqib ketadi.
- **Qo'llab-quvvatlash telefonda** — tartib panel bilan bir xil: avval bilim
  bazasi, keyin operator, va operatorga o'tish tugmasi **hech qachon
  yashirilmaydi**. Maqolalar panelning o'z faylidan **import** qilinadi
  (yordamchi faqat shulardan javob beradi; ikkinchi nusxa — bu buildni
  tasvirlashdan to'xtaydigan nusxa). ⚠️ Socket o'rniga **poll**: panel kun bo'yi
  stolda turadi, telefon esa cho'ntakda to'xtaydi, va jimgina o'lgan socket
  «ulangan» deb yozib turadi. ⚠️ Ilgari ega biz bilan Telegram orqali
  bog'lanardi — ya'ni murojaat **hech qayerda yozilmasdi**, va operator konsoli
  bo'sh turardi.
- ⚠️ **Past baholi fikr endi push bilan ham keladi** — loss alertlar bilan bir
  xil sabab: u `sendFeedbackToGroup` orqali **faqat Telegram**da edi. Yonma-yon
  yuboriladi, ichida emas. Faqat past baho (maqtov uchun jiringlagan kanal bir
  haftada o'chiriladi), **menejerga ham** (loss alertdan farqi: bu xabar xodim
  haqida emas, va kechqurun qo'ng'iroq qila oladigan odam ko'pincha aynan u).

### Ishchilar davomati (`/staff` + `/admin/staff` + `/admin/payroll`)
- **Ikki kirish, bir chiqish**: hamma narsa ikkita manbadan hisoblanadi —
  admin yozgan **ish grafigi** va ishchi bosgan **smenalar**. Kunning holati
  (`kam ishlangan` / `grafik bo'yicha` / `ko'p ishlangan` / `chiqmagan`)
  hech qayerda saqlanmaydi, u **taqqoslash** (`handlers/staffattendance.go`).
- **Geolokatsiya majburiy va serverda tekshiriladi** (`geofenceBlocked`).
  Ilova tugmani oldindan o'chiradi, lekin qoida — serverda. Har bosish
  koordinatasi va masofasi bilan saqlanadi (`shift.inAt/outAt`).
- **Radius 50 m (5 emas)**: telefon GPS'i ochiq havoda 10–30 m, bino ichida
  yomonroq. Xato chegarasidan kichik radius firibgarni ushlamaydi, halol
  ishchini eshik oldida qoldiradi. `branch.staffRadiusM` sozlanadi; **0 —
  tekshiruv o'chiq**, shuning uchun eski filiallarga `EnsureStaffDefaults`
  50 yozadi va migratsiya yaratadigan filialga qiymat **qo'lda** beriladi
  (Go nol qiymatni tushirib qoldirmaydi → `$exists:false` uni ko'rmaydi).
- **Smena boshlangan kunga yoziladi** (`shift.date`, local `YYYY-MM-DD`):
  00:40 da chiqqan oshpaz seshanbani yopadi. Serverda **`TZ=Asia/Tashkent`**
  bo'lishi shart, aks holda butun kalendar UTC'da chiziladi.
- **Kun holati punchlarga qarab**, daqiqalarga emas: kirib darhol chiqqan odam
  0 daqiqa ishlagan, lekin "chiqmagan" emas. Bo'sh grafik ham hech kimni
  ayblamaydi — to'ldirilmagan kun dam olish deb o'qiladi.
- **Ikki marta kirish 409** (birlashtirilmaydi): ikkita ochiq smena kalendardagi
  har soatni ikkilantirardi.
- **Oylik maosh oyning grafik kunlariga bo'linadi** va chiqilgan kun uchun
  beriladi — o'rtada ishga kirgan yoki bir hafta kelmagan odam ham to'g'ri
  chiqadi va kalendar bilan kassa bir xil gapiradi.
- **To'lov — yozuv, hisoblagich emas** (`staff_payment`): qaysi davrni yopgani
  va kim bergani bilan. Aprel boshlangandan keyin ham "martda qancha to'ladik"
  javobsiz qolmaydi.
- **Qo'lda tuzatish ataylab qoldirilgan**: telefon o'chadi, chiqish unutiladi —
  tuzatib bo'lmaydigan davomat tizimidan ikkinchi haftada voz kechiladi. Har
  tuzatish `editedBy` bilan imzolanadi va jurnalga tushadi.
- Ishchi hisobi o'chirilsa **smenalar qoladi**.
- Sana yorliqlari `01.08` ko'rinishida va oy nomlari lug'atdan: `uz-UZ`
  locale'da "long" oy **"M08"** bo'lib chiqadi.
- Kalendar bitta komponent (`components/staff/AttendanceCalendar.tsx`),
  ranglar bitta joyda (`lib/attendance.ts`) — panel va ishchi ilovasi bir xil
  kunni har xil rangda ko'rsata olmaydi.

### Kuryerlar va rollar
- **Rollar**: `owner`/`manager` (admin), `user` (mijoz), `courier`, `staff`.
  `middleware.RequireRole` har guruhda rolni tekshiradi — ilgari faqat
  `RequireAuth` bo'lgani uchun mijoz tokeni bilan admin API'ga kirish mumkin edi.
- Kuryer hisobini **admin qo'lda yaratadi** (`/admin/couriers`), o'zi
  ro'yxatdan o'ta olmaydi. Parol bcrypt bilan saqlanadi.
- Status: `off` (ishda emas) / `free` (bo'sh) / `busy` (band). Buyurtma
  biriktirilganda avtomatik `busy`, yetkazilgach — faol buyurtmasi qolmasa —
  `free` bo'ladi (`syncCourierBusy`).
- Kuryer faqat **o'z** buyurtmasini `on_the_way` va `delivered` ga o'tkaza
  oladi; oldingi bosqichlar oshxonaniki.
- **"Yetkazdim" faqat manzilda bosiladi**: kuryerning oxirgi joylashuvi mijoz
  manzilidan `delivery.arrivalRadiusM` metrdan uzoq bo'lsa yoki 10 daqiqadan
  eski bo'lsa — server 400 qaytaradi (`arrivalBlocked`). Ilova tugmani
  o'chirib qo'yadi, lekin qoida serverda. Olib ketish buyurtmalari va
  koordinatasiz manzillar tekshirilmaydi; `arrivalRadiusM = 0` — o'chiq.
  **Zaxira yo'l**: admin panelda holatni baribir qo'lda yopish mumkin (GPS
  ishlamay qolgan holat uchun ataylab qoldirilgan).
- **Naqd pul topshiruvi** (`courier_settlement`): "qo'lidagi naqd" =
  yig'ilgan − topshirilgan. Ilgari faqat yig'ilgan ko'rsatilardi va u hech
  qachon kamaymasdi. Topshirish **ledger yozuvi** sifatida saqlanadi
  (hisoblagichni nolga tushirish emas) va kim qabul qilgani yoziladi —
  nomsiz naqd topshiruv keyin bahsga aylanadi.
- **Daromad** (`courierstats.go`): har kuryerning `payoutMode` i bo'yicha
  hisoblanadi — `deliveryFee` (default, mijoz to'lagan yetkazish narxi),
  `perOrder` (belgilangan summa) yoki `percent` (yetkazish narxidan foiz).
  Olib ketish buyurtmasi hech qachon daromad bermaydi. Davrlar: bugun,
  oxirgi 7 kun, oxirgi 30 kun, jami — `statusHistory` dagi `delivered`
  vaqtiga qarab. "Naqd yig'ilgan" alohida ko'rsatiladi (kuryer qo'lidagi pul).
- Mijoz `/order/{number}` da kuryerni **faqat `on_the_way` bosqichida**
  xaritada ko'radi; boshqa holatlarda joylashuv umuman qaytarilmaydi.

### Tashqi yetkazish xizmatlari (o'z kuryeri yo'q restoranlar uchun)
- `/admin/settings` → "Tashqi yetkazish xizmatlari": har biri `link` (havola),
  `phone` (dispetcher) yoki `api`.
- **Asosiy yo'l — web** (biznes-akkaunt kerak emas). Buyurtmalarda
  "Yetkazishni chaqirish" → modal 3 qadam: (1) havola o'rinbosarlar bilan
  to'ldiriladi va bitta bosishda ochiladi + "chaqirildi" deb yoziladi;
  (2) har bir maydon alohida "Nusxalash" tugmasi bilan; (3) xizmat javobi
  (raqam, `cost`, izoh).
- **Yandex Go Dostavka deeplink** (tayyor namuna):
  `yandex.go.link/route?tariffClass=courier&adj_t=pucm71r&trap_mode=true`
  + `start-lat/start-lon` (restoran) va `end-lat/end-lon` (mijoz).
  Telefonda `yandextaxi://route?...` ga aylanadi va **Dostavka** bo'limida
  ochiladi. ⚠️ `adj_t` (Adjust tokeni) **shart** — bo'lmasa havola 404; eski
  `3.redirect.appmetrica.yandex.com/route` ishlamaydi.
  ⚠️ **`tariffClass` qaysi bo'lim ochilishini belgilaydi va tarif nomlari
  mamlakat bo'yicha farq qiladi**: Rossiyaning `express_d2d` tarifi
  O'zbekistonda sotilmaydi — ilova jim turib **taxi** buyurtmasiga qaytadi.
  O'zbekistonda ishlaydigan tarif — **`express`**.
  ⚠️ **Deeplink faqat koordinata + tarif oladi**: ism, telefon, manzil matni va
  izoh havolada uzatilmaydi — shuning uchun oynada har maydon nusxalanadi.
  ⚠️ O'zbekistonda Yandex Delivery jismoniy shaxsga faqat **Yandex Go ilovasi**
  orqali (`delivery.yandex.uz` — yuridik shaxslar uchun), shuning uchun
  kompyuterdagi administrator uchun **QR kod** chiziladi
  (`components/admin/QrCode.tsx`).
- O'rinbosarlar (`lib/providerLink.ts`): `{number} {name} {phone} {address}
  {comment} {lat} {lng} {total} {subtotal} {deliveryFee} {items} {itemsCount}
  {payment}` + `{pickupName} {pickupAddress} {pickupPhone} {pickupLat}
  {pickupLng}`.
- Yozuvni keyin tahrirlash `calledAt` ni o'zgartirmaydi. Tashqi xizmat
  chaqirilsa **o'z kuryeri bo'shatiladi** (bitta buyurtmani ikkovi olmasin).
- **`kind: "api"`** (`internal/delivery/yandex.go`) — Yandex Delivery B2B cargo
  API: `claims/create|accept|info|cancel`. Token panelda kiritiladi va **hech
  qachon brauzerga qaytarilmaydi**. `request_id = "order-<orderId>"` —
  **idempotent** (tarmoq uzilib qayta chaqirilsa ikkinchi kuryer chaqirilmaydi).
  `apiBaseUrl` sozlanadi (sandbox/mock). Mahalliy xizmatlarda ommaviy API yo'q —
  ular `phone`/`link`; API paydo bo'lsa `clientFor` ga yangi `apiProvider`.

### Uzum Tezkor: marketplace buyurtmani o'zi yuboradi

Hujjat: `docs/vendor/uzum-tezkor-retail.md` (Notion + OpenAPI `.yml`, o'qildi
2026-09-14). ⚠️ Bu **Yandex Eda vendor API** ning nusxasi — Yandex Eats
integratsiyasi deyarli shu kontrakt bo'ladi.

- ⚠️ **Ular bizni chaqiradi, biz ularni emas.** Uzum token oladi
  (`/security/oauth/token`), har buyurtmani `POST /order` qiladi va holatini
  daqiqada bir so'raydi (`GET /order/{id}/status`). Biz server, ular klient.
  Manzil: `{domen}/api/v1/uzum-tezkor`, har yo'l **`v1` bilan ham, `v1` siz
  ham** — ularning hujjati katalogga `v1` qo'yadi, buyurtmaga qo'ymaydi, host
  esa bitta.
- ⚠️ **Yetkazishni doim Uzum kuryeri qiladi.** Shuning uchun buyurtmada manzil,
  yetkazish narxi va bizning kuryer yo'q, va u **alohida tur**:
  `order.type = "uzum_tezkor"` (to'lov usuli va `channel` ham shu). `delivery`
  ga bayroq qo'shish o'rniga — chunki bizning kuryer, zona va kuryer olib
  keladigan pul haqida o'ylaydigan har ekran undan uzoq turishi kerak.
- **Pul Uzum'da**: `paymentStatus: paid`, `paymentMethod: uzum_tezkor` —
  kassada hech kim pul olmaydi, perechisleniye ekrani esa to'langan
  `uzum_tezkor` sotuvlarini allaqachon "Uzum ushlab turgan pul" deb sanaydi.
- ⚠️ **Narx — Uzum sotgani, menyudan qayta hisoblanmaydi.** Sayt buyurtmasi
  brauzerga ishonilmagani uchun qayta narxlanadi; bu yerda narx marketplace
  shartnomasi, va chekdagi boshqa raqam hech kim solishtira olmaydigan hisob.
  ⚠️ `items.price` — **modifikatsiyalar qo'shilgan** bitta pozitsiya narxi
  (spetsifikatsiya "keyingi versiyada tuzatiladi" deydi), shuning uchun
  modifikatsiyalar **option emas, izoh** bo'lib yoziladi: option narx farqini
  olib yuradi va chekda ikki marta qo'shilardi.
- ⚠️ **`quantity` — float**; yarim portsiya oshxona pishira oladigan qator
  emas, shuning uchun butun bo'lmagan son **rad etiladi** (yumaloqlanmaydi).
- **Menyuda o'chirilgan taom ham qabul qilinadi**: Uzum uni allaqachon sotgan,
  rad etish qayta yuborishni boshlaydi va mijoz ko'radigan bekor qilish bilan
  tugaydi. Oshxona buyurtmani ko'radi va o'zi hal qiladi.
- ⚠️ **Qayta yuborish — birinchi buyurtma**: 4xx/5xx olsa Uzum o'sha `eatsId`
  bilan qayta yuboradi. Oldin qidirish ikki qayta urinish orasidagi poygani
  yopmaydi, shuning uchun `(aggregator.provider, aggregator.externalId)`
  **unique partial** indeks, va duplicate-key xatosi birinchi buyurtmaning
  javobiga aylanadi (`200`, o'sha `orderId`).
- ⚠️ **`GET /order/{id}` kelgan tanani bayt-bayt qaytaradi**
  (`aggregator.payload`): Uzum boshqacha tarkibni restoran buyurtmani
  **o'zgartirgani** deb o'qiydi — mijozga push va tranzaksiya. O'z
  qatorlarimizdan qayta qurilgan tana yumaloqlash yoki tushib qolgan maydonda
  farq qilardi, va har biri hech kim qilmagan o'zgarish bo'lardi.
- **Holat xaritasi** (`tezkorStatusOf`, testi bor): `pending→NEW`,
  `confirmed→ACCEPTED_BY_RESTAURANT`, `preparing→COOKING`, `readyAt→READY`,
  `on_the_way→TAKEN_BY_COURIER`, `delivered→DELIVERED`, `cancelled→CANCELLED`.
  ⚠️ `READY` — bizning `readyAt` (status emas) va **faqat qabul qilingandan
  keyin**: Uzum orqaga qadam va qabulni chetlab o'tishni qabul qilmaydi.
- ⚠️ **Panelda "yo'lda" bosqichi tashlab ketilmaydi** (`orderFlow.travels`):
  olib ketish uni o'tkazib yuboradi, lekin Uzum buyurtmasida aynan shu bosqich
  `TAKEN_BY_COURIER`. Tugma "Uzum kuryeriga berildi".
- ⚠️ **15 daqiqa**: qabul qilinmagan buyurtmani Uzum bekor qiladi. Buyurtma
  `queuedAt` bilan keladi (jiringlaydi), queue watch esa 10 daqiqada egaga
  aytadi — alohida qoida kerak emas.
- **Bekor qilish**: Uzum `DELETE` → holat `cancelled`, sabab `"Uzum Tezkor:
  <izoh>"`, ombor qaytadi (`syncOrderStock`) va egaga push. ⚠️ Filtr holat
  bo'yicha: "yetkazildi" bilan poygada bekor qilish uni qaytara olmaydi.
  `PUT` (ular tomonidan o'zgartirish) — **422**: spetsifikatsiya uni kam va
  kelishuv bilan deydi, jimgina qabul qilish esa pishayotgan taomni
  o'zgartirardi.
- **Kalitlar** (`uzum_tezkor_settings`, sozlamalar → "Uzum Tezkor", faqat
  ega): `client_id` + `client_secret`, serverda **faqat SHA-256**; secret
  **bir marta** ko'rsatiladi. Har yangi secret `tokenVersion` ni oshiradi —
  eski tokenlar keyingi so'rovda o'ladi, bir soat kutmaydi. Sozlamalar **har
  so'rovda** o'qiladi (o'chirish darhol ta'sir qiladi). Token — JWT, rol
  `uzum_tezkor`, subject `client_id` (ObjectID emas — boshqa yo'lda hech kimni
  nomlamaydi), 1 soat.
- ⚠️ **Xatolar ularning shaklida** — `[{code, description}]`, 401 esa
  `{reason}` — va `httpx.Error` dan **o'tmaydi**: ularni Uzum support
  muhandisi o'qiydi, tarjima qilinsa sarlavhaga qarab bir kun ruscha, bir kun
  o'zbekcha bo'lardi.
- **So'rov cheklovi** alohida (300/daq): Uzum har so'rov oldidan token oladi va
  har ochiq buyurtmani daqiqada bir so'raydi — `authGate` ning 10 tasi band
  kechqurun uni qulflardi.
- ⚠️ **Identifikatorlar `tezkor*`**: `payuzum.go` (Uzum Bank to'lovi) bir
  paketda `uzum*` nomlarini band qilgan — ikki xil Uzum bir faylda
  adashtirilmasin.
- **Katalog** (`/v1/nomenclature/{storeId}/composition`, soatiga bir) va
  **qoldiq** (`…/availability`, 5 daqiqada bir) — `handlers/uzumtezkormenu.go`.
  `storeId` — filial ID; katalog — o'sha filial **brendining** menyusi.
- ⚠️ **Katalogga faqat savolsiz sotib bo'ladigan taom chiqadi**
  (`tezkorSellable`): model qatori (o'lcham/rang) — hech qachon; **majburiy
  variant guruhi bor taom — yo'q**, chunki Uzum katalogida modifikator guruhi
  yo'q va har buyurtma javobsiz keladi; narxi 0 — Uzum baribir tashlaydi.
  Yopiq kategoriyadagi taom va taomsiz kategoriya ham chiqmaydi — Uzum
  `categoryId` ni yuborilgan kategoriyalar bo'yicha tekshiradi.
- ⚠️ **O'chirilgan yoki stopdagi taom katalogda qoladi, qoldig'i 0**: katalogdan
  tushirish uni oshxona qayta yoqqandan keyin ham bir soatgacha Uzum'da
  yashirardi. Qoldiq — biz sotuvni rad etadigan **hamma** sabab va yangisi yo'q:
  `isAvailable`, `Branch.IsSoldOut` (to'rtala stop list + kunlik limit), stopdagi
  a'zosi bor combo. Kunlik limit — **qolgan son**; qolgan hammasi — 99 ("bor").
  Ro'yxatda yo'q taom Uzum uchun sotilmaydi, shuning uchun qoldiq **katalogdagi
  har bir** taomni beradi.
- ⚠️ **Do'kon uchun majburiy maydonlar halol to'ldiriladi**: `barcode` — taomning
  o'z shtrix-kodi (do'konda bor), aks holda ID; `vendorCode` — ID; `measure` —
  **bo'sh obyekt**. O'ylab topilgan "500 g" mijoz telefonida taom tagida
  yozilardi. ⏳ Uzum buni qabul qilishini test muhitida tekshirish kerak.
- **Rasm hash'i — faylning SHA-1'i** (spetsifikatsiya: hash o'zgarsa Uzum qayta
  yuklaydi; URL hash'i almashtirilgan rasmni sezmasdi). Faqat `/uploads/` dagi
  **o'z** fayllarimiz, `ServeUploads` bilan bir xil tozalanib va `os.Root`
  ichida o'qiladi; hajm + vaqt bo'yicha keshlanadi. Begona serverdagi rasm
  **chiqarilmaydi** — hash'siz rasm butun taomni validatsiyadan yiqitadi.
- **ИКПУ** bo'lsa `serviceCodesUz`, bo'lmasa **umuman yo'q** (ИКПУ qoidasi);
  `oldPrice` — faqat narxdan **yuqori** bo'lsa (aks holda eskirgan maydon).
- ⏳ **Hali yo'q**: perechisleniye API (hujjatda yo'q), Uzum test muhiti, taom
  nomining tili (hozir asosiy — o'zbekcha nom). ⚠️ Bizda har restoran alohida
  server — Uzum "texnologik hamkor"ga bitta host yozsa, so'rovni tenantga
  control yo'naltirishi kerak bo'ladi; hozirgi yozuv har restoranning **o'z
  domeni** bilan ishlaydi.

### Status sahifasi: rang va qisqa uzilishlar (`keel.uz/status`)

Control har daqiqada o'zini tekshiradi (o'z bazasiga ping + ishlashi kerak
bo'lgan har tenant konteyneri) va natijani soatlik bucketga yig'adi
(`control/internal/handlers/status.go`).

- ⚠️ **Qizil — faqat 50% dan past** (2026-09-14). Ilgari bitta muvaffaqiyatsiz
  daqiqa ham butun kunni to'liq qizil qilardi: 90 kunlik chiziqdagi 10 qizil
  kundan 9 tasi 99.65–99.93% edi (1–5 daqiqa), va o'quvchi haqiqiy uzilish
  bo'lgan 96% lik kunni ulardan ajrata olmasdi. Endi to'rt holat: **100% —
  yashil**, **≥50% — sariq "qisman"**, **<50% — qizil**, **ma'lumot yo'q —
  bo'shliq**. Qisman davr yashil **emas** — nimadir yiqilgan, va rang buni
  "ishlamadi" demasdan aytadi. Soat va kun chizig'i bir qoidada
  (`PARTIAL_FROM`).
- ⚠️ **Konteyner ketma-ket ikki tekshiruvda ishlamasa sanaladi**
  (`confirmedDown`, testi bor). Deploy va rollout konteynerni bir necha
  soniyada qayta yaratadi; shu soniyalarga tushgan bitta tekshiruv butun
  daqiqani uzilish qilib yozardi. Qolib ketgan konteyner **ikkinchi
  daqiqasidan** boshlab sanaladi, faqat almashtirilgani — hech qachon. Deploy
  haqida hech narsa bilish shart emas: qoida CI deploy'ga ham, control
  rollout'iga ham, qo'lda `docker restart` ga ham bir xil ishlaydi.
- ⚠️ **Baza tekshiruvi darhol qoladi**: o'z bazasiga yeta olmagan control
  deploy bo'lsa ham hamma uchun ishlamayapti.
- Oldingi natija **xotirada** (bitta control, bitta goroutine): restartdan
  keyingi birinchi tekshiruv hech kimni sanamaydi — ya'ni xato bo'lsa, o'tkazib
  yuborilgan daqiqa tomonga, yo'q uzilish tomonga emas.
- ⚠️ Eski bucketlar **qayta hisoblanmaydi** — 31.08–08.09 dagi 1–5 daqiqalik
  kunlar endi sariq bo'lib ko'rinadi, lekin raqamlari o'sha qoladi.

### Tashrif hisobi (`visit`)
- `POST /visit` — sayt sahifasidan otiladigan mayoq. Restoran paneli nechta
  **buyurtma** kelganini aytardi-yu, nechta odam **qaraganini** aytmasdi — bu
  esa "hech kimga kerak emas" bilan "hech kim topa olmayapti" farqi, va
  ularga javob teskari.
- **Bir kunga bitta qator** (`(date, vid)` unique): noyob tashrifchi —
  qatorlar soni, sahifa ochilishi — `views` yig'indisi. Bitta kolleksiya,
  har sahifaga bitta yozuv.
- ⚠️ **Belgi kun bilan birga hash qilinadi** (`sha256(date|vid)`), ya'ni
  ertaga o'sha brauzer boshqa qator. Bu — odamni kuzatib bo'lmasligini
  ta'minlaydigan xususiyat, va shu sabab 30 kunlik raqam **tashrif-kun**,
  alohida odamlar emas. Ataylab: biroz yuqori raqam odamni kuzata oladigan
  tizimdan yaxshiroq.
- **Qidiruv robotlari o'z-o'zidan tushmaydi**: mayoq JS'dan otiladi, Googlebot
  esa uni ishlatmaydi. Saqlanadigan user-agent ro'yxati kerak emas.
- **Qatorlar muddatli** (TTL, 100 kun): bu — biznes bilan emas, **trafik
  bilan** o'sadigan yagona kolleksiya. Kunlik yig'indi platformaning
  `tenant_day` iga ancha oldin ko'chiriladi.
- Panel/kuryer/ishchi ekranlari sanalmaydi — bu biznesning o'z ishi, mijoz
  tashrifi emas, va uni qo'shish sokin haftani band ko'rsatardi.

### 404 va xatolik sahifalari
- Har ikkala ilovada: `not-found.tsx` (topilmadi), `error.tsx` (sahifa yiqildi),
  `global-error.tsx` (root layoutning o'zi yiqildi).
- ⚠️ **Ildizda, `(site)` ichida emas**: Next guruh chegarasiga faqat o'sha
  guruhdagi marshrutga tushgan manzil uchun boradi; hech nimaga mos kelmagan
  manzil **ildizdagi** faylga tushadi — shuning uchun sahifa o'z chiqish yo'lini
  o'zi chizadi (guruh layouti, ya'ni header/footer qo'llanmaydi).
- ⚠️ **Client komponent — shuning uchun tarjima qilinadi.** Til allaqachon root
  layout o'rnatgan provayderda; bu yerda `headers()` ni qaytadan o'qish
  `/_not-found` ni **har tashrifchi uchun dinamik** qilardi va hech nima
  bermasdi. Noto'g'ri tildagi 404 — mehmon uchun "sayt buzuq" degani.
- ⚠️ **Havolalar `LocaleLink` bilan**: `/ru/...` da yalang'och `href` prefiksni
  tushirib qoldiradi, ya'ni ruscha mehmonning 404'i uni o'zbekchaga o'tkazadi.
- **404 va 500 ataylab boshqa sahifa**: birinchisiga javob — boshqa joyga
  o'tish, ikkinchisiga — kutib qayta urinish. Bitta "xatolik yuz berdi" sahifasi
  odamlarni "qayta urinish" hech qachon ishlamaydi deb o'rgatadi.
- ⚠️ **`global-error` hech nimaga tayanmaydi**: provayder ham, tema ham, dizayn
  tokenlari ham yo'q (ular yiqilgan layoutning ichida) → inline stil, tizim
  shrifti, inline SVG. Va **uch tilda birdan**, chunki bu yagona ekran
  tashrifchining tilini bila olmaydi; taxmin qilish uchdan ikki qismini
  tushunarsiz matn bilan qoldirardi.
- Rasm — **inline SVG**, `currentColor` bilan: yuklangan fayl 404'ning ichidagi
  404 bo'lishi mumkin, va aynan `uploads` buzilgan mijozda yo'q bo'lardi.

### SEO va favicon
- `app/layout.tsx` dagi `generateMetadata` restoran profilidan quriladi:
  sarlavha shabloni `%s | <restoran nomi>`, tavsif, **favicon = yuklangan
  logotip** (`app/icon.svg` — zaxira), Open Graph/Twitter uchun cover rasm.
- Har bo'lim o'z sarlavhasini beradi. Client komponent bo'lgan sahifalar
  (savat, checkout, login, profil, buyurtma kuzatuvi) uchun sarlavha
  yonidagi `layout.tsx` da — shu yerda ular `robots: noindex` ham oladi
  (shaxsiy sahifalar qidiruvga tushmasligi kerak).
- Taom sahifasi `generateMetadata` da nom/tavsif/rasmni tanlangan tilda beradi.
- **`robots.txt` va `sitemap.xml`** (`app/robots.ts`, `app/sitemap.ts`) — ikkalasi
  ham **har host uchun alohida**, chunki bitta build hamma restoranga xizmat
  qiladi: statik fayl bitta saytning sitemap'ini hammaga aytardi. Yandex
  robots.txt dagi `Sitemap:` qatorini asosiy topish yo'li deb biladi.
- Sitemap'da **har bir mavjud taom sahifasi** bor — aynan ular "lag'mon
  yetkazib berish" deb qidirgan odam tushadigan, va boshqa o'n ming restoran
  saytida yo'q kontent. Savat/checkout/buyurtma kuzatuvi ataylab yo'q: ular
  har tashrifchida boshqa va hech qanday natijada chiqmaydi.
- **`metadataBase` va canonical so'rovdan olinadi** (`lib/seo.ts`), build'dan
  emas. Aks holda bitta build hamma restoranga bir xil (ya'ni birovning)
  domenini yozardi — va Google buni xato deb aytmaydi, shunchaki sahifalarni
  birlashtirib yoki tashlab yuboradi.
- **JSON-LD** (`components/site/StructuredData.tsx`) — qidiruv natijasidagi
  ko'k havolani ish vaqti, telefon va xarita nuqtasi bilan kartochkaga
  aylantiradigan narsa. Turi biznesga qarab tanlanadi (`Restaurant`,
  `Pharmacy`, `Florist`…): gulchiga "oshxona" deb aytish — structured data
  butunlay e'tiborga olinmasligining yo'li. To'ldirilmagan maydon
  **yuborilmaydi**, bo'sh qator sifatida emas.
- **Canonical va hreflang bitta joydan** — root layout, `localeAlternates()`
  (`lib/seo.ts`), yo'lni middleware sarlavhasidan oladi. Har sahifa avtomatik
  oladi. ⚠️ Ilgari layout **har sahifa uchun sayt ildizini** canonical deb
  yozardi, ya'ni menyu va har bir taom sahifasi bosh sahifaning nusxasi deb
  e'lon qilinardi. Xato sifatida hech qayerda ko'rinmaydi — sahifalar
  shunchaki chiqmaydi.
- **Qidiruv tizimiga tasdiqlash**: `restaurant.seo { google, yandex }`,
  sozlamalarda domen bo'limining **ostida** (bepul subdomen uchun olingan kod
  o'z domenini tasdiqlamaydi). Kalitlar `restaurant` hujjatida — xarita kaliti
  bilan bir mantiq: tasdiqlash kodining butun vazifasi `<head>` da turish.
  ⚠️ Ikkala konsol ham egaga **butun `<meta ...>` tegini** ko'rsatadi, shuning
  uchun `verificationToken()` teg ham, yalang'och kod ham qabul qiladi. Bo'sh
  qiymat `undefined` bo'lishi shart: bo'sh teg = "teg bor, lekin noto'g'ri".

### Til URL'lari: `/ru/`, `/en/` va hreflang
- Til boshidan cookie'da edi, lekin **robot cookie tashimaydi** — ya'ni Google
  va Yandex uchun har sahifa aynan bitta tilda mavjud edi. Ruscha menyuning
  manzili yo'q edi, demak uni ulashib ham, indekslab ham bo'lmasdi.
- `src/middleware.ts` prefiksni yechadi va `/menu` ga rewrite qiladi. `[lang]`
  segmenti **emas**: sahifa fayllari, `<Link>` lar va API yo'llari o'zgarmadi.
  O'zbekcha prefikssiz — asosiy til, va uning manzillari allaqachon QR
  kartochkalarda va indeksda.
- **URL cookie'dan ustun** (`getLang()`: sarlavha → cookie). Aks holda
  ulashilgan havola qabul qiluvchining tilida ochilardi va yuboruvchi buni
  hech qachon ko'rmasdi. Shu sababli `LangSwitch` **manzilga o'tadi**,
  `refresh()` qilmaydi — aks holda tugma ishlamayotgandek ko'rinadi.
- ⚠️ **Sarlavha avval o'chiriladi, keyin faqat haqiqiy prefiks bo'lsa
  qo'yiladi.** Ikki sabab: mijoz istalgan sarlavhani yubora oladi; va
  prefikssiz URL'da cookie hukmron qolishi kerak. Shartsiz qo'yilganda **admin
  panel, kuryer va ishchi ilovalarining har bir ekrani** o'zbekchaga qadalardi
  — almashtirgich harakatlanib turadi, cookie to'g'ri, til ishlamaydi.
- `isLocalizedPath()` — `/admin`, `/kuryer`, `/staff`, `/kiosk` da til URL'i
  yo'q (login orqasida, indekslanmaydi) va `/ru/admin` hech qachon yasalmaydi.
- `LocaleLink` (`components/site/LocaleLink.tsx`) — `next/link` o'rniga import
  qilinadi, href'lar prefikssiz yoziladi. Robot ruscha sahifalar borligini
  **havolalardan** biladi; sitemap yetarli emas.
- Header'ning `isActive` i `splitLangPath` dan o'tadi — aks holda prefiksli
  URL'da butun navbar yorug'ligini yo'qotadi.
- Sitemap: har sahifa **bir marta**, `alternates.languages` bilan. Uch alohida
  yozuv — aynan hreflang oldini oladigan dublikat muammosi.
- robots: shaxsiy yo'llar **har prefiks uchun ham** yopiladi — yo'l prefiksi
  bo'yicha yozilgan qoida yo'lning o'zi prefiks olishi bilan mos kelmay qoladi.
- Tashrif mayog'i prefiksni yechib yozadi: aks holda eng band sahifa uchta
  sokin sahifaga bo'linardi.

### Kassadagi to'lov tugmalari: nom egasiniki, tur uchta

Egasi sozlamalarda (To'lov tizimlari → "Kassadagi to'lov usullari") kassir
ko'radigan tugmalarni o'zi yozadi: nom ("Humo terminal", "Beznal") va **tur** —
`cash` / `card` / `transfer` (`payment_settings.tillMethods`).

- ⚠️ **Tugma — tur ustidagi nom, yangi tur emas.** Yashik `cash` ni sanaydi,
  smena hisoboti `cash` / `card` / qolganini ajratadi, perechisleniye `card` ni
  terminal hisob-kitobi deb o'qiydi. Tugma id'si `order.paymentMethod` ga
  yozilsa uchalasidan tushib qolardi — "Naqd (dollar)" sotuvi o'zi tushgan
  yashikda yo'q. Shuning uchun buyurtmada **tur qoladi**, tugma esa yonida:
  `paymentOptionId` + muzlatilgan `paymentOptionName`. Hech bir hisobot
  o'zgarmadi.
- **Pulni tugmaning sozlamadagi turi belgilaydi**, ekran yuborgan
  `paymentMethod` emas — u faqat tugma o'chirilgan holatdagi zaxira. ⚠️ Faqat
  kassadagi pul uchun: `debt` yoki `payme` yonidagi tugma id'si ekran xatosi,
  va u qarzni naqdga aylantira olmaydi.
- ⚠️ **O'chirilgan tugma ham topiladi** (`TillMethodByID`): uzilish paytida
  kassada qolgan sotuv o'sha tugmada qilingan, nomini yo'qotish kerak emas.
  Internetsiz yo'l ham id'ni olib yuradi (`PendingSale.methodId`,
  `LocalCheck.paymentMethodId`, `syncCheck.paymentMethodId`).
- **Hech narsa sozlanmagan bo'lsa — eski uchta** (id = tur, nomsiz, kassa o'z
  tilida nomlaydi): bu o'zgarishdan oldingi har sotuv xuddi shunday o'qiladi.
  Standart uchtasini o'chirib bo'lmaydi, faqat yashirish mumkin.
- ⚠️ **Kamida bittasi yoqilgan** — server rad etadi, jimgina qaytarib yoqmaydi
  (sozlamalar egani aytmasdan bekor qilgan bo'lardi). Yangi qatorga id
  **serverda** beriladi, nomdan emas: "Karta" ni "Humo" ga qayta nomlash o'tgan
  oy hisobotida o'sha tugma bo'lib qolishi kerak. Saqlagandan keyin forma
  serverning id'lari bilan yangilanadi — aks holda keyingi saqlash yangi
  qatorlarni ikkinchi marta yaratardi.
- ⚠️ **Eski server `options` yubormaydi** — kassa `methods` dagi turlardan
  standart uchtasini tiklaydi (`payOptionsFrom`), ya'ni yangi kassa eski
  serverda ham tugmasiz qolmaydi.
- ⚠️ **Uzum Tezkor kassada taklif qilinmaydi** (`tillOffer`): buyurtmalari
  panelga o'zi tushadi va Uzum'ga to'langan, kassadagi tugma bir buyurtmani
  ikkinchi marta urishga taklif. Sozlamadagi kalit qoladi (perechisleniye
  ekrani o'qiydi), server esa id'ni **qabul qilishda davom etadi** — o'zgarishdan
  oldin internetsiz navbatga tushgan sotuv yo'qolmasin.

**To'lov oynasi (`app/kassa/PayDialog.tsx`) — ikki yarim.** Chapda **pul**:
to'lanadigan summa eng katta shriftda, taomlar va chegirma, naqd bo'lsa
olingan summa + **tez summa tugmalari** (aniq summa va undan yuqori yumaloq
pullar — 6 ta bosish o'rniga 1) + katta **qaytim**; chegirma yig'ilgan blokda
(ko'p chekda yo'q, va navbat bor kassir uni o'tkazib ko'rmasligi kerak). O'ngda
**usul plitkalari** ikonka bilan: egasining tugmalari, relslar, qarz.
- ⚠️ **Plitkaning accessible nomi — faqat yorliq**, ikonkalar `aria-hidden`:
  kassa testlari va ekran o'quvchi usulni so'z bo'yicha topadi.
- ⚠️ **Bosilgan tugma taklifda bo'lmasa, birinchisiga o'tadi**: panelda "Humo"
  bosilgan paytda ega uni o'chirgan bo'lishi mumkin, va hech bir plitka yonmagan
  oyna server bilmaydigan tugmada chek yopardi.

### Kassada karta: QR skanerlash (Click Pass / Uzum FastPay)

**Muammo.** Kassada to'rt usul bor edi: naqd, **karta**, o'tkazma, qarz. "Karta"
degani — kassir stoldagi bank terminaliga burilib, summani **qayta teradi** va
kutadi. Uch joyda xato bo'lishi mumkin: summa qo'lda teriladi, chek ikkinchi
qog'oz bo'lib chiqadi, va bizning ekranimiz karta o'tgan-o'tmaganini **bilmaydi**.

**Yechim yo'nalishi teskari.** `tillpay.go` allaqachon QR chiqarardi: biz havola
yasaymiz → mehmon **bizning** QR'imizni skanerlaydi → bank callback yuboradi →
kassa poll qilib kutadi. Endi `tillscan.go` teskarisini qiladi: mehmon o'z
ilovasida kodni ochadi → **kassir uni skanerlaydi** → karta **shu so'rovning
ichida** yechiladi. Ikkalasi ham qoladi: QR — kuryer va ilovasi yo'q mehmon
uchun, skaner — tushlik navbatidagi uchun.

- **Ikkita adapter yozildi**: `click_pass` va `uzum_fastpay` (`internal/instore`).
  Hujjatlarining o'qilgan nusxasi — `docs/vendor/`. ⚠️ Ikkala sayt ham JS bilan
  chiziladigan SPA: `curl` hujjat matnini qaytarmaydi, shuning uchun nusxa
  repoda yotadi — "havolaga qara" keyingi sessiyada ishlamaydi.
- ⚠️ **Payme GO — ochiq API yo'q** (2026-08-30 da tekshirildi).
  `developer.help.paycom.uz` faqat Merchant API va Subscribe API'ni, ya'ni
  **e-commerce** tomonini hujjatlaydi; "оплата на месте" kassasi esa "ulangandan
  keyin darhol ishlaydi" deb tasvirlanadi — ya'ni integratsiya nuqtasi yo'q,
  Payme Business ilovasining **o'z skaneri**. Ro'yxatda turadi, `Ready: false`.
- ⚠️ **Bank terminaliga summa yuborish ham shunday.** Hujjatlangan ECR
  protokoli topilmadi: `humocard.uz` Smart PIN Pad — "info@nmpc.uz ga yozing",
  `rhmt.uz` Rahmat POS — "hamkorlik bo'limiga murojaat qiling", `uzkassa.uz` —
  xuddi shunday. Fiskal provayderlardagi bilan bir qaror: **interfeys bor**
  (`instore.Charger`), provayderlar ro'yxatda `Ready: false`, va sabab
  ekranda yozilgan. Endpointni taxmin qilish — kompilyatsiya bo'ladigan,
  review'dan o'tadigan va restoranga terminali haydalayotgandek **ko'rinadigan**
  kod; kassir esa hamon summani qo'lda teradi va buni hech bir ekran aytmaydi.

**Nima qayerda va nega**

- ⚠️ **`click_pass` va `uzum_fastpay` — `click`/`uzum` dan alohida id.** Bank bir
  xil, **dalil boshqa**: u yerda mehmon o'z telefonida checkout sahifada to'ladi
  va callback tasdiqlaydi, bu yerda kassir kodni skanerlaydi va **bizning
  so'rovimiz** kartani yechadi. Qaytarish yo'li ham boshqa (biz bankka
  `payment_id` bilan murojaat qilamiz), settlement hisoboti ham, "kim turgan
  edi?" savolining javobi ham. Ikkalasini bitta id qilish — bu uchta savolning
  **birortasiga** javob bera olmaydigan hisobot.
- ⚠️ **Pul birligi: Click — so'm, Uzum — tiyin**, va ikkala adapter yonma-yon
  turadi. Adashtirish mehmondan yuz barobar ko'p yoki yuz barobar kam yechadi —
  **hech qayerda xato bermay**. Ikki tomonlama testi bor
  (`TestTheAmountIsInEachProvidersOwnUnit`).
- ⚠️ **Vaqt tamg'asi: Click — sekund, Uzum — millisekund**, va Uzum sarlavhani
  `^\d*:(\d{40}):\d*$` regex bilan tekshiradi. Sekund yuborish — 401 emas,
  **403** ("Authorization bilan ishlanish orasi 50 soniyadan oshdi") beradi, ya'ni
  sekin tarmoqqa o'xshaydi. Shuning uchun `msStamp`/`secStamp` alohida
  funksiyalar, izohi bilan, va ikkalasining testi bor.
- ⚠️ **Uzum har doim HTTP 200 qaytaradi**, hatto rad etganda ham — muvaffaqiyat
  mezoni `error_code == 0`. HTTP statusni birinchi tekshirish har rad etishni
  "to'landi" deb o'qirdi.
- ⚠️ **Click'ning "tasdiqlash rejimi" — 30 soniyalik pilta.** Servisda yoqilgan
  bo'lsa, tasdiqlanmagan to'lov bank tomonidan **avtomatik qaytariladi**. Buni
  e'tiborsiz qoldirish: kassirga "to'landi" ko'rsatiladi, mehmon chiqib ketadi,
  yarim daqiqadan keyin pul qaytadi va **hech bir ekranda** bu ko'rinmaydi.
  Shuning uchun tasdiq `Charge` ning ichida, chek yopilishidan **oldin**: chekni
  yopish (oshxonaga yuborish, fiskal fayl, chop etish) cheklanmagan vaqt oladi,
  30 soniya esa byudjet emas.
- ⚠️ **Chek `tillscan.go` da yopilmaydi.** Yopish — `StaffCloseCheck` ning ishi
  va o'sha yerda qoladi: u oshxonaga yuboradi, chegirmani override bilan
  qo'llaydi, chekni fayl qiladi, qog'ozni navbatga qo'yadi va ogohlantiradi.
  To'lov handleri ichidagi yarim nusxa — eskiradigan nusxa. Bu yerda faqat
  "pul keldi" yoziladi, keyin **odatdagi** yopish ishlaydi — bank tasdiqlagan
  QR to'lovi bilan bir xil yo'l.
- ⚠️ **`bankConfirmed()` — bitta ro'yxat, uch joyda so'raladi.** Ilgari
  `tillOnlineMethods` edi va yangi relslar har chaqiruvchiga qo'lda
  qo'shilishi kerak bo'lardi — uchinchisi unutiladigan shakl, va o'shanda rad
  etilgan karta chekni baribir yopadi.
- ⚠️ **Javob kelmasa — "qayta urinish" tugmasi YO'Q, faqat "tekshirish".**
  Muvaffaqiyatli bo'lishi mumkin bo'lgan to'lovni takrorlash — mehmondan ikki
  marta pul yechishning yo'li. Ikkala provayder ham takroriy `order_id` ni rad
  etadi, lekin **o'z ikki marta yechishimizdan birovning himoyasiga** tayanish
  dizayn emas, omad. `TxnID` har urinishda yangi: timeout'dan keyingi urinish
  — o'sha sotuvga qarshi **boshqa** urinish, va eski id bilan so'rasak bank
  aynan biz bilmaydigan urinish haqida javob beradi.
- **Qaytarish bankka ham boradi** (`reverseCounterPay`). ⚠️ Lekin qaytarishni
  **hech qachon to'sib qo'ymaydi**: pul qarori mehmon oldida turgan menejerniki
  va u bank bir soniyadan keyin hamkorlik qilishidan qat'i nazar kuchda.
  Muvaffaqiyatsizlik `counterPay.reverseError` ga yoziladi — "mehmon pulini
  haqiqatan qaytarib oldimi?" savolining yagona javobi shu qatorda.
  ⚠️ `reversedAt` **muvaffaqiyatsizlikda yozilmaydi**: u "bank qaytardi"
  degani, va har holda qo'yilgan vaqt har muvaffaqiyatsiz qaytarishni
  bajarilgandek ko'rsatardi.
- **Uzum fiskal havolani so'raydi** (`/payment/fiscal`) va uni o'z ilovasida
  mehmonga ko'rsatadi. ⚠️ Bu **to'lov paytida yuborilmaydi** — o'shanda chek
  hali mavjud emas. `recordFilingInto` da, kassa javob bergandan keyin. Va u
  yiqilsa sotuvda hech nima o'zgarmaydi: pul olingan, chek davlatga fayl
  qilingan, qog'oz chiqqan — yo'qolgani ilova ichidagi qulaylik havolasi.
- **`cashbox_code`** — bank kabinetidagi hisobotni **odam** o'qiydi, haftalardan
  keyin. Shuning uchun fiskal kassa id'si, bo'lmasa filial **nomi** — ObjectID
  texnik jihatdan mukammal va bu yagona ish uchun foydasiz.

**Sozlamalar: har relsga o'z tortmasi, bitta map**

`payment_settings.inStore` — `Enabled map[string]bool` + `Creds map[string]…`.
Fiskal provayderlardagi dars aynan takrorlanmasin uchun (§"Fiskal provayderlar"):
har provayderga alohida struct qilinsa, yettinchisini qo'shish to'rtta ro'yxatni
tahrirlashni talab qiladi.

- ⚠️ **`inStore` so'rovda `*pointer`.** Deploy'dan keyingi bir necha daqiqada
  ochiq tab hali **eski** sozlamalar sahifasini ushlab turadi va u umuman
  `inStore` yubormaydi. Qiymat sifatida dekod qilinsa, o'sha tabning keyingi
  saqlashi ishlab turgan Click Pass kalitlari ustiga **bo'sh map** yozadi —
  jimgina, sahifa "saqlandi" deb turib. `nil` = "bu panelning fikri yo'q".
  Testi bor (`TestAnOldSettingsPageDoesNotEraseTheCounterRails`).
- ⚠️ **Panel eslamagan rels o'z tortmasini saqlaydi**, va **bo'sh kalit
  saqlangan kalitni qoldiradi** — uchta alohida qoida, uchtasi ham alohida
  yiqiladi.
- ⚠️ **Adapteri yo'q relsni yoqib bo'lmaydi** — panelda ham, serverda ham
  (`in.Enabled && instore.Ready(id)`). Kalitlarni saqlash mumkin: ega ko'pincha
  shartnoma yopilishidan oldin sozlaydi.
- **Provayderlar ro'yxati javob bilan keladi**, panelda qattiq yozilmagan:
  qaysi rels bor, qaysisining adapteri bor va qaysi maydonlarni so'raydi — bular
  **server haqidagi faktlar**, va o'z nusxasini olib yurgan panel provayder
  olib tashlangandan keyin ham uni ko'rsatishda davom etadi.

### Savol brauzerniki emas, bizniki (`components/ui/Ask`)
⚠️ **`window.confirm` bizning ekranimiz emas.** Uni brauzer chizadi — o'z
shriftida, o'z tugmalari bilan, va tepasida restoranning **domen nomi**. Kassa
degan mashinada, mahsulot sotib olgan mijoz uchun, aynan shu lahzada butun
taassurot tushadi. Bundan tashqari u: uslub qabul qilmaydi, «OK / Cancel» dan
boshqa so'z bilmaydi (uni brauzer tili tanlaydi, restoranniki emas), va
**JavaScript oqimini to'xtatadi** — javob berilmaguncha hech nima
yangilanmaydi, hech qanday so'rov ketmaydi, toast chiqmaydi.

- **Bitta primitiv, ikki ko'rinish**: `AskProvider look="till" | "panel"`.
  Kassa barmoq bilan bosiladi (`till-dialog`, katta tugmalar), panel sichqoncha
  bilan o'qiladi. Ikkita komponent bo'lsa ular ajraydi — va ajragan joyi aynan
  **qaytarib bo'lmaydigan amallarning** tasdig'i bo'lardi.
- **Promise qaytaradi**, chunki har bir chaqiruv joyi `if (!confirm(...))
  return;` shaklida edi — o'sha shakl saqlanadi (`if (!(await ask(...)))
  return;`), ya'ni yigirmata ekranni callback atrofida qayta yozish shart emas.
- ⚠️ **Provider bo'lmasa brauzerning o'zi ishlaydi** (fallback). Layout unutilgan
  ekran ishlashda davom etadi — tasdiqni butunlay yo'qotish yomonroq bo'lardi —
  va u ko'rinishidan darrov xabar qilinadigan darajada xunuk.
- ⚠️ **Dialog `appliance` sinfining ichida**, tashqarisida emas: u mashinaning
  bir qismi — belgilanmaydi, zoom bo'lmaydi, va bosish barmoq tushgan joyda
  hisoblanadi. Ilgari aynan shu sirt («layout ichidagi, `<main>` dan tashqari»)
  har bir lokal tuzatishdan chetda qolardi.
- ⚠️ **`z-70`**, chunki ekran klaviaturasi `z-60`: pad ustida terilgan chegirma
  haqidagi savolni pad ortida chizish — javob berib bo'lmaydigan yagona dialog.
- ⚠️ **Ochiq savol ustiga ikkinchisi kelsa, birinchisi «yo'q» bilan yopiladi.**
  Navbatga qo'yib bo'lmaydi: kutayotgan kod qulfni, spinnerni yoki yarim
  yozilgan chekni ushlab turadi, va ustma-ust ikki dialog — kassir o'qimagan
  narsasini tasdiqlashining yo'li.
- **Enter tasdiqlaydi, lekin xavflisini emas**; Escape va fon bosilishi —
  har doim «yo'q» (fon hech qachon tasdiqlamaydi).
- 40 ga yaqin `alert()` ham shu yo'lga o'tdi (`tell`), ya'ni panelda ham
  brauzerning oynasi qolmadi.
- ⚠️ **Provider ro'yxati komponentda: `components/till/TillShell.tsx`.**
  `kassa/layout.tsx` va `zal/layout.tsx` — Next marshrut konvensiyasi, ularni
  **import qilib bo'lmaydi**, shuning uchun Windows ilovasi
  (`backend/desktop/frontend/src/main.tsx`) o'sha ro'yxatni **qo'lda nusxa**
  qilgan edi, «to'rtinchisi qo'shilsa bu yerga ham qo'shiladi» degan izoh bilan.
  `AskProvider` qo'shilganda qo'shilmadi — natijada brauzerdagi kassa o'z
  savolini so'rar, **kassa mashinasidagi kassa esa `window.confirm` chiqarardi**:
  hech nima buzilmagan, shunchaki bir bo'lak yo'q. Shikoyat «yangi build hali
  ham eski» bo'lib keldi, ya'ni eng yomon shakl — ekran to'g'ri ishlaydi va
  faqat noto'g'ri ko'rinadi.
  Endi ro'yxat bitta komponentda va uchala sirt shuni mount qiladi; nusxaning
  ortda qoladigan narsasi qolmadi. Muhri — `TillShell.test.ts`: uchala fayl
  `TillShell` ni mount qiladimi **va** bo'laklarni o'zi mount qilmaydimi
  (yoniga qo'yish — aynan ajrash boshlanadigan joy).
- ⚠️ **Fallback endi ovoz chiqaradi** (`console.error`): jim fallback shu
  xatoning **haftalab yashashiga** sabab bo'lgan. Konsol xatosi crash
  hisobotiga tushadi (`lib/report.ts`), ya'ni keyingi safar qaysi savol
  brauzerning oynasida chiqqani nomi bilan ma'lum bo'ladi.

### Kassa, zal, oshxona, kiosk: tez bosganda qotib qolish (butun app)

**Muammo takrorlanib turardi va har safar boshqa ekran haqida edi.** PIN pad
tuzatildi, keyin menyu setkasi tuzatildi — va shikoyat qaytdi, chunki tuzatish
**odat** edi, standart emas. Odat sifatida ishlaydigan versiyasi yo'q.

Uch kafolat, uchtasi uch xil sababdan yiqiladi:

1. **Bosish ro'yxatga olinadi** — `components/till/TillAppliance.tsx`.
2. **Hech nima zoom bo'lmaydi** — `NoZoom` (TillAppliance uni o'zi mount qiladi).
3. **Hech nima ekrandan ko'chirilmaydi** — CSS + klaviatura/menyu tinglovchilari.

- ⚠️ **`.till` ikkiga bo'lindi: `.appliance` (xulq) va `.till` (palitra).**
  Ilgari sensor qoidalari `.till` ning **ichida** edi, ya'ni ekran faqat
  kassaning krem-sariq ranglarini ham xohlasa mashinaday tutardi. Oshxona
  ekrani (KDS) va filial kioski ikkinchisini emas, birinchisini xohlaydi — va
  natijada **ikkalasi ham** yo'q edi.
- ⚠️ **Sinf sahifada emas, layout'da.** `<main className="till">` sahifaning
  ichida edi, ekran klaviaturasi esa undan **tashqarida** — aynan oldingi
  lokal tuzatishlar o'tkazib yuboradigan sirt. Endi `kassa`, `zal`, `staff`,
  `kiosk` layout'larida.
- ⚠️ **Global qatlam `pointerup` da ishlaydi, `pointerdown` da emas** — va bu
  `tap.ts` bilan qarama-qarshi emas, uning **ikkinchi yarmi**. `tapProps`
  kontaktda ishlaydi va bu klaviatura va taom setkasi uchun to'g'ri; global
  standart esa scroll'ga xavfsiz yarmini oladi, chunki kassa scroll qilinadigan
  ro'yxatlarga to'la: flick boshlangan qatorni ochadigan chek ro'yxati —
  tuzatilayotgan nosozlikdan **yomonroq** nosozlik.
- ⚠️ **Barmoq tushgan element hal qiladi, ko'tarilgan joy emas.** `click` bosish
  **va** qo'yib yuborishni bitta elementda talab qiladi, ya'ni qiya monoblokda
  ikki piksel siljish hech nima bermaydi — "bosdim, o'tmadi" aynan shu.
- ⚠️ **`click` tinglovchisi `capture` fazasida, `document` da.** React o'z
  ishlovchilarini root konteynerga ulaydi, u esa `document` ning **ichida** —
  bubble fazadagi tinglovchi to'xtatmoqchi bo'lgan ishlovchidan **keyin**
  ishlaydi va har tugma ikki marta ishlardi (taom ikki marta qo'shiladi, raqam
  ikki marta kiritiladi).
- ⚠️ **O'z pointer hodisalarini boshqaradigan komponent tegilmaydi**
  (`defaultPrevented`) — aks holda `tapProps` ishlatadigan har bir plitka ikki
  marta ishlardi.
- ⚠️ **Sichqoncha butunlay chetda.** Sichqonchaning bosishi allaqachon zudlik
  bilan va aniq joyga tushadi; muammo — paneldagi barmoq uchi. Aralashish esa
  stol kompyuterida drag'ni, kontekst menyusini va double-click'ni buzardi.
- ⚠️ **Formadagi maydonlar to'liq brauzerning o'zida qoladi**: karetka qo'yish,
  select ochilishi, checkbox. Foydali sinov ikkinchi marta bosilgan checkbox
  emas — **hech kim rozi bo'lmagan chegirma**.
- ⚠️ **Yozish istisno, hammasi.** `user-select: none` maydonga meros bo'lib
  o'tsa, kassir chegirmadagi noto'g'ri raqamni belgilab qayta tera olmaydi —
  yagona chora butun maydonni tozalash bo'lib qoladi, ya'ni **sekinroq**
  kassa; bu sinf esa aynan sekinlikka qarshi. Ekranning o'zidan ko'chirish
  yopiq (kontekst menyu, `selectstart`, `dragstart`, `copy`/`cut`,
  Ctrl+C/X/A) — maydonlarda esa qoladi, ular kassir hozir yozgan matn.
- ⚠️ **Paste to'silmaydi.** U faqat maydonga tushadi, va to'silsa "yozish
  o'rniga qo'yish" rejimiga sozlangan skaner ishlamay qoladi.
- ⚠️ **`click()` — `HTMLElement` ning metodi, va zal sxemasi SVG.** Stol
  `<g role="button">` bo'lib chiziladi, ya'ni yuqoridagi tanlagichga mukammal
  tushadi: barmoq tushgan element yozib olinadi, aks-sado qo'riqchisi
  **qurollanadi**, keyin `el.click()` — SVG guruhida bunday metod yo'q —
  **istisno tashlaydi**. Natijada brauzerning o'z bosishi (ishlab ketadigani)
  o'tib ketayotganda yutib yuborilardi: zal sxemasidagi stolni **barmoq bilan
  umuman ochib bo'lmasdi**, sichqoncha bilan esa mukammal ochilardi — shuning
  uchun u faqat monobloklardan xabar qilindi va brauzerda hech qachon
  takrorlanmadi. Endi metodi yo'q element uchun hodisa qo'lda yuboriladi
  (`activate`), va buning testi bor. Xuddi shu tuzoq yetkazish zonasi kabi har
  qanday SVG ustiga bosiladigan sirtni kutib turgan edi.

### Kassa ekranlari ro'yxati: limit va qaysi mashina

Tarif **kassa ekranlari soni** bilan sotiladi, va cheklov faqat **eshikda**
tekshiriladi (havola berilayotganda) — ishlab turgan kassa hech qachon
to'xtatilmaydi; sababi `handlers/tilldevices.go` da yozilgan.

- ⚠️ **Windows ilovasi limitni raqam bo'lib ko'rsatardi.** Oltinchi monoblokni
  bog'layotgan odam «qurilma kaliti: server 402» degan javobni olardi — yangi
  mashina oldida turgan odamga **status kodi**, va u bilan qiladigan ishi yo'q.
  Server allaqachon sababni so'z bilan va **sanog'i bilan** yuboradi
  (`registers`, `limit`), va setup ekrani — buni ko'rsata oladigan yagona joy:
  «kassa ekranlari limiti tugadi (5 / 5) — panelda ishlatilmayotgan ekranni
  uzing yoki tarifni ko'taring». ⚠️ Sanoq bo'lmasa ham gap gap bo'lib qolishi
  kerak (eski server 402 ni raqamsiz qaytaradi) — `capMessage` va uning testi
  `desktop/tillcap.go` da, **build tegsiz faylda**: restoran eng yomon paytda
  o'qiydigan gap test yeta olmaydigan joyda qolmasligi kerak.
- ⚠️ **Ro'yxatdagi qator qaysi mashina ekanini aytmasdi.** «Kassa 1», «Kassa 2»
  — qatorning nomi, binoda turgan narsaning nomi emas; va aynan shu ro'yxatdan
  limitga yetgan menejer **qaysi birini uzishni** tanlaydi. Yonidagi yagona
  fakt — oxirgi ishlatilgan kuni, u esa ishlayotgan restoranda hamma qator
  uchun bugun. Endi qatorda **filial nomi**, **Windows kompyuter nomi** va
  **IP** turadi: `X-Till-Host` sarlavhasini ilovaning o'zi qo'yadi (proxy har
  so'rovga — `proxy_windows.go`, va bog'lash so'roviga), IP esa
  `clientIP(r)` dan. ⚠️ Sarlavha — **nom, kalit emas**: u hech nimaga ruxsat
  bermaydi va hech qachon ruxsat sifatida o'qilmaydi (proxy'ning «hech qanday
  kalit qo'yilmaydi» qoidasi shu sababdan buzilmaydi).
- ⚠️ **Brauzerdan ulangan ekranda kompyuter nomi bo'sh qoladi, va bu ham
  ma'lumot**: orqasida mashina turmagan qator — odatda tashlab ketilgan
  havola, ya'ni limitga yetgan menejer aynan izlayotgan qator.
- ⚠️ Nom va IP **`touchTillDevice`** da ham yangilanadi (soatiga bir marta,
  `lastSeenAt` bilan birga), faqat havola berilganda emas: panelda QR chiqarib
  uni monoblokda ochish odatiy yo'l, ya'ni havolani bergan kompyuter bilan
  kassa bo'lib qoladigan kompyuter ko'pincha **boshqa** mashina.

### Kassa (POS) va zal: shu sessiyada qo'shilganlar

Bu qism `main` da (`apps/till-flow-tests` merge qilingan va o'chirilgan).

- **Zal sotuvlari** (`/admin/checks`) — buyurtmalar taxtasi kassa cheklarini
  ataylab ko'rsatmaydi (`check: {$exists:false}`, ajratuvchi maydon `check`,
  `type == "dinein"` **emas** — QR bilan buyurtma bergan mehmon ham `dinein`).
  Pul esa hisobotlarda birinchi kundan bor edi; yo'q bo'lgani — **ro'yxat**.
  ⚠️ Jamilar butun filtrlangan davr bo'yicha (sahifa bo'yicha emas), ochiq stol
  pul olmagan, bo'lingan chek **stol** sifatida bir marta sanaladi.
- **Chekni ochish, chop etish/PDF** — chek serverda chiziladi (kassadagi bilan
  bir `receipt.Render`), brauzer chop etadi; **kassa printeriga yuborish**
  alohida tugma (panelni ochgan odam odatda binoda emas). Void qatorlar
  kartochkada **ko'rinadi** va qog'ozda yo'q.
- **Qaytarish** — ⚠️ sotuv qoladi, pul o'zgaradi (`order.refund`, sabab
  majburiy). Jimgina xato tuzatildi: `received()` `delivered` bo'lsa pulni
  sanardi, kassa cheki esa yopilishi bilan `delivered` — qaytarilgan stol
  tushumda qolib ketardi. Kassa qoldig'i **faqat oldingi smenadagi** sotuv
  uchun tuzatiladi (bugungisi `paid` dan chiqishi bilan o'zi ayriladi).
- **Xizmat haqi** (`branch.service`) — ⚠️ filialga tegishli, **faqat stolga**,
  foiz stol o'tirganda chekka **ko'chiriladi**, chegirmadan keyin hisoblanadi,
  yaxlitlash bitta joyda. Oflaynda ham xuddi shunday (foiz qurilmaga
  yetkaziladi, **rate** sim orqali ketadi, summani server qayta hisoblaydi).
- **X/Z hisobot** — X **GET** va hech nimani o'zgartirmaydi, Z esa yopish
  javobida qaytadi (yopib, keyin "chop etishni unutmang" degan ekran — Z
  hisoboti bo'lmagan kunlar demakdir). Sotuv cheklardan sanaladi, smenadagi
  hisoblagichdan emas.
- **Bo'lish va birlashtirish** — bo'lish **ofitsiantniki** (zal ekranida ham),
  hammasini bo'lib bo'lmaydi; birlashtirishda yutilgan chek **bekor qilinadi,
  o'chirilmaydi** (void qatorlari va raqami qoladi), mehmonlar qo'shiladi.
- **Chop etish navbati** (`/admin/settings` → Printerlar) va `/admin/alerts` da
  **jim banner**: chiqmagan chek — tizimdagi eng jim nosozlik (buyurtma
  ekranda, sotuv hisobotda, alomat esa hech kim pishirmagan taom). Chiqqan
  topshiriq qayta yuborilmaydi (404).
- **Windows printerga `net share` siz**: printer nomi spooler orqali so'raladi
  (`winspool.drv`, lazy DLL — cgo yo'q, cross-compile saqlanadi), share yo'li
  zaxira. Datatype **RAW**.
- **Sotuvlar ro'yxati kassada** (`/staff/checks/closed`, `ChecksScreen`) — ikki
  tab: **Ochiq** va **Yopilgan**. Ilgari yopilgan chek haqidagi har savol
  ("yana chiqarib bering", "u naqdmi ketdi?") panel logini talab qilardi, ya'ni
  zaldagi kompyuterda ochiq turgan ega sessiyasi: mijozlar bazasi, to'lov
  kalitlari va hisobotlar.
  - ⚠️ **Faqat bugun va faqat o'z filiali.** Oyni varaqlaydigan kassa — o'g'irlashga
    arziydigan kassa, va bu savol eganing ekraniga tegishli. Oyna **ochiq
    smenadan** boshlanadi (yarim tundan keyin ishlaydigan restoranda kun
    chegarasi ro'yxatni eng band soatda bo'shatardi).
  - ⚠️ **Bekor qilingan cheklar ro'yxatda qoladi**: "6-stolga nima bo'ldi"
    degan savol aynan shularni qidiradi, yashirish esa kassa sotuvni
    yo'qotgandek ko'rsatadi.
  - ⚠️ **Qarz jamiga kirmaydi** (`owed` alohida): bu satr yashik yonida turib
    o'qiladi, va qarzni tushumga qo'shish "pul shu yerda" degan yolg'on bo'lardi.
  - **Yopilgan chekda yagona amal — qog'oz**: qaytarish sababli va menejerning
    qarori, u panelda qoladi; har sotuv ro'yxatidan bir bosish naridagi
    qaytarish "mijoz shikoyat qildi" va "pul qaytdi" ni bitta harakat qilardi.
- **To'lov tizimlari kassada** (`handlers/tillpay.go`) — Payme / Click / Uzum:
  kassir tizimni tanlaydi → ekranda **QR** chiqadi → mehmon **o'z telefonida**
  to'laydi → provayder serverga aytadi → chek **o'zi yopiladi**.
  - ⚠️ **Chek to'lov tasdiqlanmaguncha yopilmaydi** (409). Oson yo'l — tanlagan
    zahoti "to'landi" deb belgilash — har sinovda ishlaydi (sinovda to'lov
    muvaffaqiyatli), navbatda esa bekor qilingan, muddati o'tgan yoki
    **birovning ekranida** qilingan to'lov uchun ovqat berib yuboradi. Saytning
    eng eski qoidasi peshtaxtada: **brauzer hech nimani isbotlamaydi**.
  - ⚠️ **`paidAt` bankniki**, yopish payti emas: smena almashadigan soatda
    ustidan yozish pulni bir smenadan ikkinchisiga ko'chirardi.
  - ⚠️ **Terminal (`card`) bu ro'yxatda emas va bo'lmaydi**: uning o'z cheki,
    o'z hisob-kitobi va bu tizimdan umuman o'tmaydigan puli bor. "Tasdiqlandi"
    deb ko'rsatish — **ko'rmaydigan** yagona to'lovga yashil belgi qo'yish.
  - ⚠️ **ATMOS ham yo'q**: havolasi **so'raladi** (tarmoq sababidan yiqilishi
    mumkin), va ba'zan chiqmaydigan QR peshtaxtada umuman taklif qilinmagandan
    yomonroq — kassir telefon ko'targan mehmonga bo'sh ekranni tushuntiradi.
  - **Ro'yxat serverdan** (`GET /staff/payment-methods`): sozlanmagan tizim
    tugmasi bankning xato sahifasiga olib boradi va mehmon **restoranni**
    ayblaydi (`/payment-methods` bilan bir qoida).
  - **So'rab turiladi (poll), soket emas** — 2 soniyada bir marta, faqat QR
    ekranda turganda. Muvaffaqiyatsiz so'rov ekranda ko'rsatilmaydi: mehmon
    to'lov o'rtasida, kassirning aybi yo'q.
- **Qarz (`method: "debt"`)** — ⚠️ **bu to'lov turi emas, hali to'lamaslikning
  yozuvi**, va u peshtaxtadagi daftarning o'rnini bosadi. Chek `delivered` +
  **`unpaid`** bo'lib yopiladi: ovqat chiqdi, pul kelmadi, va bu farqni butun
  tizim allaqachon tushunadi (`received()` uni tushum deb sanamaydi).
  - ⚠️ **Kimga qarz yozish mumkinligini ega belgilaydi**
    (`user.creditAllowed`, standart — **o'chiq**). Qarz yozish kassirning ishi;
    kimga yozilishini ham o'sha odam tanlasa, peshtaxtadagi eng eski usul
    ishlaydi: kechqurungi kamomad telefon orqali topilgan doimiy mijozning
    nomiga yoziladi va **yashik to'g'ri sanaladi**. Shuning uchun galochkani
    `/admin/users/{id}` da faqat **ega** qo'yadi (`requireOwner`, maydon
    yuborilgandagina tekshiriladi — menejer izoh yozayotganda rad etilmasin),
    va u **amallar jurnalida nomi bilan** yoziladi ("qarz: yoqildi"): oylar
    o'tib so'raladigan savol — kim ruxsat berdi va qachon.
    - **Server — darvoza, ekran emas**: `StaffCloseCheck` mijoz hujjatini
      o'qiydi va ruxsatsiz yopishni rad etadi (so'rovdagi maydonga ishonmaydi).
      Kassa esa buni **oldindan** aytadi: `/staff/customers` javobida
      `creditAllowed` bor, ya'ni "topildi, lekin ruxsat yo'q" va "umuman mijoz
      emas" ikki xil javob bo'lib ko'rinadi — mehmon oldida qizil quti chiqishi
      buzuq kassaga o'xshaydi, qoidaga emas.
    - Menejer ham galochkani **ko'radi** (o'zgartira olmaydi): ko'rmagan
      menejer kassirdan "nega rad etdi?" deb so'raydi, kassir esa bilmaydi.
    - ⚠️ Eski mijozlarda ham **o'chiq**: bayroq yo'q edi, ya'ni "ruxsat
      berilgan" deb o'qish har bir bazadagi har bir odamga qarz ochib berardi.
      Doimiy mijozlarga ega bir marta belgilab chiqadi.
  - ⚠️ **Mijozsiz qarz qabul qilinmaydi** (server ham, tugma ham): nomsiz qarz
    — o'sha daftarning o'zi, ya'ni hech kimning kartochkasida ko'rinmaydigan va
    hech kim so'ramaydigan pul. Kassada mijoz **telefon bo'yicha** topiladi
    (kassir so'ray oladigan va mehmon javob beradigan yagona narsa), yonida
    izoh maydoni ("juma kuni to'laydi").
  - ⚠️ **Qaytarilgan kun — bugungi kun, ovqat yeyilgan kun emas**
    (`POST /admin/debts/{id}/pay`): seshanbadagi kechki ovqat juma kuni
    yopilsa, **jumaning** tushumi va **jumaning** kassasi bo'ladi — aks holda
    kassir bugungi yashikni uch kun oldingi raqamga qarab sanardi. Sotuvning
    o'z sanasi (qamrov, taom sanog'i) o'zgarmaydi.
  - ⚠️ Yopish **qarz filtri bilan qo'riqlangan**, `_id` bilan emas: ikki
    ekranda bosilgan "to'ladi" pulni **bir marta** olishi kerak, bekor qilingan
    yoki allaqachon yopilgan qarz esa 404 berishi kerak — jimgina ikkinchi
    to'lovni bugungi yashikka ko'chirmasligi.
  - ⚠️ **Qarz kassada ham qaytariladi** (`GET /staff/debts?phone=`,
    `POST /staff/debts/{id}/pay`): pul kassaga keladi, ya'ni uni qabul
    qiladigan odam — yashik oldidagi odam. Paneldan yopish qoldi (ega kartani
    telefon orqali solishtiradi), lekin oddiy holat — juma kuni naqd bilan
    kirgan doimiy mijoz, va kassirni mehmon oldida panel logini bor odamni
    izlashga yuborish menejer parolining kassa yoniga yozib qo'yilishining yo'li.
  - ⚠️ **Faqat telefon bo'yicha qidiriladi, ro'yxat berilmaydi**: zaldagi
    ekranda restoranning barcha qarzdorlari — peshtaxta oldida turgan har
    kimga ochilgan mijozlar bazasi. Bo'sh so'rov hech nima qaytaradi.
  - ⚠️ **Z hisobotda "shundan qarz qaytdi" qatori** (`cashFigures.DebtPaid`):
    pul `paidAt` bo'yicha bugungi yashikka **allaqachon** tushadi, sotuv esa
    kechagi — ya'ni qog'ozda yashik smena sotganidan ko'p bo'lib chiqadi va
    buni tushuntiradigan hech nima bo'lmasdi. Tushuntirilmagan farq esa
    yashikni sanagan odamga qo'yilgan ayb. Qator **CounterCash ichida**, unga
    qo'shimcha emas.
  - **Uch joyda ko'rinadi va uchtasi ham majburiy**: mijoz kartochkasida
    (`debtTotal`/`debtCount` + ro'yxat va "qaytardi" tugmasi — qo'ng'iroqda
    turgan odam ikki ekran narida raqam qidirmaydi), dashboardda **`money.debt`
    alohida plitka** (`pending` ichida, lekin nomlangan: qolgani bir soatda
    o'zi keladi, bu esa kimdir qo'ng'iroq qilganda), moliyaviy hisobotda
    "— shundan qarzga" qatori.
  - ⚠️ **X/Z hisobotda alohida qator va sotuvdan tashqarida.** Ilgari to'lov
    turlari `switch` ida `default` ga tushib **"o'tkazma"** bo'lib chiqardi:
    jami ishonarli ko'rinardi, qog'oz o'sha shaklda qolardi, va yagona alomat —
    yashikni hech qachon solishtirib bo'lmasligi edi. Farq uchun esa odam
    ayblanardi. Testda muhrlangan.

### Tannarx va ombor: raqam qayerdan keladi va nimani anglatmaydi

Zanjir: **masalliq → texkarta → kirim → sotuvdan sarf → chiqim → ko'chirish →
sanash → "bo'lishi kerak"**. Har bosqichda raqamning manbasi va **chegarasi**
ekranda yozilgan — bu bo'limning ko'p qismi aynan shu chegaralar haqida.

⚠️ **Qaysi javonda ekani masalliqda emas** — `ingredient_placement` da, har
filial uchun alohida (§4). Har bir hisob shu joylashuv orqali o'qiladi, va
ombor ekranlari shu sababdan **bitta filialni talab qiladi** (§5).

- **Masalliq** (`ingredient`) — narx **sotib olinadigan birlikda** (kg / litr /
  dona), retseptda esa g / ml / dona. ⚠️ **Uchta birlik oilasi va konvertatsiya
  dvigateli yo'q**: erkin matnli birlik ("bog'lam") hisoblab bo'lmaydigan karta
  va *hisoblangandek ko'rinadigan* tannarx yasardi.
- **Texkarta** — `menu_item.recipe`, miqdorlar **brutto** (ombordan chiqadigan).
  ⚠️ Karta bo'lsa u qo'lda kiritilgan `cost` dan **ustun**, va har o'qishda
  qayta hisoblanadi: bitta raqamning ikki manbasi jimgina ajralib ketadi.
  ⚠️ **To'liq bo'lmagan karta taomni narxlamaydi** (o'chirilgan masalliq taomni
  arzonlashtirardi — marja ko'rsatadigan ekranda bu **yaxshi xabar** bo'lib
  ko'rinadi), va ishlatilayotgan masalliqni o'chirish 409 bilan rad etiladi.
- ⚠️ **`cost` ham, `recipe` ham `json:"-"`**: taom o'nlab handler orqali ommaga
  chiqadi, va retsept — miqdorlari to'ldirilgan raqobatchi ro'yxati. Panel uni
  **atayin** so'raydi (`menuItemIO`), ya'ni xavfsiz standart — "yuborilmaydi".
- **Yarim tayyor** — kartasi va **chiqimi** (`output`) bor masalliq. ⚠️ Chiqim
  halollik joyi: 3 kg pomidordan 2 kg sous chiqsa, chiqim 2000. ⚠️ Ikki karta
  bir-birini chaqirishi mumkin — stavkalar **bosqichma-bosqich** hisoblanadi va
  hal bo'lmagani **narxlanmagan** bo'lib qoladi (rekursiya ham, nolga sanash
  ham yo'q).
- ⚠️ **Yaxlitlash bir marta, tayyor taomda**: bir gramm bir so'mdan arzon, ya'ni
  qatorlarni yaxlitlash ko'p taomni nolga tushirardi (`serviceOn` bilan bir
  qoida). Brauzerdagi nusxasi bor, va uni testlar bir xil raqamlar bilan ushlab
  turadi.
- **Narx tarixi** (`ingredient.history`) va **kunlik hisob** (`costledger.go`):
  ⚠️ har sotuv **o'z kunining narxi** bilan hisoblanadi, aks holda bugungi narx
  o'zgarishi allaqachon o'qilgan oyni qayta yozardi. Kunlik kesh: oylik hisobot
  ko'pi bilan 31 marta hisoblanadi.
  ⚠️ **Qo'lda tahrir — "bugundan"** (forma "xato yozgandik"ni "qimmatlashdi"dan
  ajrata olmaydi), **kirim esa o'z sanasi bilan** (o'lchov). Kechikkan
  nakladnoy tarixga sanasi bo'yicha tushadi, lekin **bugungi narxni
  o'zgartirmaydi**.
  ⚠️ **Retsept versiyalanmaydi** — ataylab: narx doim o'zgaradi, karta esa taom
  o'zgarganda o'zgaradi, va o'zgargan taom boshqa taom.
- **Kirim** (`purchase`) — nakladnoyning **o'z jami** ustun (eshik oldidagi
  chegirma hech bir qatorda yo'q); narx o'zgarmagan bo'lsa tarixga yozuv
  qo'shilmaydi.
  - ⚠️ **Tahrir va o'chirish boshqacha ishlaydi, va farq ataylab.** Har narx
    yozuvi endi **qaysi nakladnoydan** kelganini biladi (`PriceEntry.PurchaseID`),
    shuning uchun tahrir aynan o'sha da'voni olib tashlab qaytadan yozadi:
    tahrir "nakladnoy shuni yozishi kerak edi" degan gap, ya'ni narx haqidagi
    da'vo. **O'chirish narxlarni qaytarmaydi** (avvalgidek): u faqat "bu qator
    bu yerda bo'lmasligi kerak" deydi va xato kiritishni ovqat allaqachon
    narxlangandan keyin bekor qilingan yetkazishdan ajrata olmaydi.
  - ⚠️ **Qo'lda kiritilgan narxlarga hech qachon tegilmaydi** (`PurchaseID`
    bo'sh): ular kimning nakladnoyi ekanini ayta olmaydi, taxmin qilish esa
    ataylab qilingan tuzatishni jimgina o'chirardi.
  - **To'langanmi** (`paid`) — mijoz qarzining ko'zgusi, alohida ledger emas:
    summa, sana va kimga qarzdorlik allaqachon nakladnoyda. ⚠️ Yopish **qarz
    filtri bilan** qo'riqlangan, `_id` bilan emas. ⚠️ Migratsiya eski
    kirimlarni to'langan qiladi — busiz "yo'q maydon" qarz bo'lib o'qilib,
    ikki yillik tarixi bor restoran sahifani **o'ylab topilgan** yuz millionli
    qarz bilan ochardi.
- **Yetkazib beruvchi** (`supplier`) — ⚠️ **erkin matn qoladi va ko'rsatish
  ixtiyoriy**: bozorga chiqishning yetkazib beruvchisi yo'q, majburiy qilish
  esa kirim umuman yozilmasligiga olib keladi. Nakladnoy **ikkalasini** tashiydi:
  id (jami va qarz shu bo'yicha guruhlanadi) va **muzlatilgan nom nusxasi**
  (qayta nomlash o'tgan yilgi nakladnoylarni qayta yozmasligi uchun).
  Brend darajasida — zanjir go'shtni bitta qassobdan hamma filialga oladi.
  ⚠️ Hisobotda **qarz davrga bog'liq emas**: martdagi nakladnoy mayda ham qarz.
- **Chiqim** (`writeoff`) — ⚠️ **sabab majburiy** (void/qaytarish/bekor bilan
  bir qoida), **o'sha kunning narxida baholanadi va muzlatiladi**, yarim tayyor
  esa kartasi bo'yicha (buzilgan bir partiya sousni nolga yozmaslik uchun).
- ⚠️ **Sotuvdan sarf combo'ni ochib yoyadi** (`soldDishes`). To'plam buyurtmaga
  **bitta qator** bo'lib tushadi va uning o'z retsepti yo'q — ovqat a'zolarida.
  Ya'ni yuzta oilaviy combo omborga **nol gramm** bo'lib ko'rinardi, va
  yetishmovchilik haftalar keyin sanashda, taxtani ushlab turgan odamdan
  tushuntirish so'ralganda chiqardi. Kassa ko'prigi bu darsni allaqachon
  o'rgangan (`posItems`), shuning uchun funksiya **umumiy va sof**. A'zolar
  **buyurtmadagi muzlatilgan nusxadan** olinadi (menyudan emas: to'plam
  qayta qurilgan bo'lishi mumkin), id'siz eski buyurtmalar uchungina jonli
  ta'rifga tushiladi. Combo tannarxi va stop listi ham shu qoidada.
- **Ko'chirish** (`stock_transfer`) — ⚠️ **chiqim ham, kirim ham emas.** Chiqim
  deb yozilsa isrof hisobotiga hech kim isrof qilmagan sabab qo'shiladi, kirim
  deb yozilsa **xarid narxi tarixga** tushib o'sha masalliqli har bir taomni
  qimmatlashtiradi. **Ikki masalliq orasida** (ombor masalliqniki), va ⚠️
  **birliklar mos kelishi shart**: kilogrammni litrga ko'chirish ikkala balansni
  ham izchil va ikkalasini ham noto'g'ri qoldiradi — buni keyin hech nima
  ko'rmaydi. Qiymat **tashiladi, yaratilmaydi**: jami "ko'chirilgan" deb
  ataladi va moliyaviy hisobotning xarajatlariga kirmaydi.
- **Sanash telefonda** (`/staff/stock`, `PermStock`) — sanash omborda bo'lib
  ofisda yozilardi, ikki marta yozilgan raqam esa ikkinchisida xato, va xato
  aynan **farq** ustuniga tushib kamomaddan farq qilmay qolardi.
  - ⚠️ **Panel va telefon bitta funksiyadan saqlaydi** (`saveStocktake`,
    `stocktakeSheet`) — `composeOrder` bilan bir sabab: ikki nusxa birinchi
    tahrirda ajraladi, va bahs javondan bizning ikki ekranimizga ko'chadi.
  - ⚠️ **O'z ruxsati bor** (KDS'dagi `canKitchen` naqshi): sanash keyingi har
    bir kamomad o'lchanadigan **bazani yozadi**, ya'ni saqlangan sanash undan
    oldin yo'qolgan hamma narsani jimgina kechiradi — bu "yozuvni yo'q qilish"
    yarmi. ⚠️ Farqi: KDS'dan farqli **grandfathering kerak emas** — ekran
    yangi, ya'ni standart rad etish hech kimdan hech nima olmaydi.
  - ⚠️ **Filial ishchidan olinadi, tanadan emas**, va "bo'lishi kerak" raqami
    **son yozilmaguncha ko'rsatilmaydi**: yonida bo'sh katak turgan "9.4
    bo'lishi kerak" yozuvi — katakka 9.4 yoziladigan varaq.
- **Inventarizatsiya** (`stocktake`) — ⚠️ **mahsulot: farq** (kassa smenasi
  bilan bir qoida). "Kutilgan"ni **server** hisoblaydi va saqlashda muzlatadi,
  ekranda esa u **raqam yozilmaguncha ko'rsatilmaydi**; farq bo'lsa izoh
  majburiy. Kutilgan = oxirgi sanash + kirim − texkarta bo'yicha sarf −
  chiqim, ya'ni **to'rtta yozilgan fakt**, tizim yuritgan qoldiq emas — shuning
  uchun ekran **qaysi sanashdan beri** ekanini aytadi.
- ⚠️ **Yarim tayyor sanalmaydi**: u ertalab pishirilgan qozon, va nimadan
  qilingani allaqachon masallig'ining sanog'ida — ikkalasini sanash pomidorni
  ikki marta ayirardi.
- **"Tugayapti"** — `minQty` qo'yilgan masalliqda "bo'lishi kerak" undan pastga
  tushsa. ⚠️ **Nol — "ogohlantirma"**, va **qo'ng'iroq yo'q**: bu "keyingi
  buyurtmada yodda tut", `AlertBell` dagi har ovozning esa panelda aynan bitta
  to'xtatuvchi tugmasi bo'lishi shart.
- **Xarid ro'yxati** (`/admin/shopping`) — o'sha ogohlantirishning **ikkinchi
  yarmi**: ilgari sariq qator "kam qoldi" deb aytardi va shu yerda to'xtardi.
  ⚠️ **Oxirgi yetkazib beruvchi bo'yicha guruhlangan** (eng ko'p emas: butcher
  almashtirgan restoranga yangisini aytish kerak, yillik sanoq esa eskisini
  javob eng noto'g'ri bo'lgan davr davomida nomlab turadi). ⚠️ **Manfiy javon
  bo'sh deb hisoblanadi**: manfiy yarmi o'lchov xatosi, va unga qarab buyurtma
  berish raqamlar eng ishonchsiz paytda ikki barobar oldirardi.
- **Ombor bo'yicha stop list** (`branch.stockStop`, `branch.stockSoldOut`) —
  **uchinchi ro'yxat**, peshtaxta bosgani (`soldOut`) va kassadan kelgani
  (`posSoldOut`) bilan hech qachon qo'shilmaydi: biri odam, biri poller, biri
  arifmetika yozadi, qo'shilsa bir-birini bekor qiladi (§"Stop list" dagi
  bilan bir dars, uchinchi marta).
  - ⚠️ **Standart holatda o'chiq**: sotuvni rad etish tizim qila oladigan eng
    qimmat ish, orqasidagi balans esa taxmin — seshanbaning nakladnoyini
    kiritmagan restoranning kassasi javonda turgan ovqatni rad etardi.
  - ⚠️ **Sanalmagan ombor hech nimani to'xtatmaydi.** Nolning ikki ma'nosi bor
    — "yo'q" va "hech kim aytmagan" — va faqat birinchisi rad etish uchun
    sabab. Bu — xususiyat yoqilgan kuni menyuni bo'shatmaydigan yagona qo'riq.
  - ⚠️ **"Bitta porsiyaga yetmaydi" ham "tugadi" bilan bir gap**, faqat nolga
    emas taomga nisbatan o'lchangan: 200 g go'sht va 500 g'lik karta —
    pishirib bo'lmaydigan taom. **Bitta porsiya, ikkitasi emas**: undan nariga
    o'tish nechta buyurtma kelishi haqidagi bashorat bo'lardi.
- **Masalliq ABC** ("Ombor" hisobotida) — menyu ABC'si **boshqa savolga** javob
  beradi va ikkalasi muntazam kelishmaydi: eng ko'p daromad keltiradigan taom
  ko'pincha eng qimmat masalliqdan qilinmaydi. ⚠️ **Sotib olingan pul bo'yicha**
  saflanadi, kartalar bo'yicha sarf emas: birinchisi o'lchangan (sanasi va
  jami bor nakladnoy), ikkinchisi esa yarim menyusi narxlanmagan taxmin.
  Kesim va "chiziqni kesib o'tgan qator A'da qoladi" qoidasi menyu ABC'sidan
  **aynan** olingan — bitta panelda ikki xil Pareto chegarasi hech nima
  bermaydi.
- **Hisobotlar**: ABC'da tannarx/yalpi foyda ustunlari **faqat tannarx bo'lsa**;
  moliyaviyda kirim **chiqim**, sotilgan taomlar tannarxi va yalpi foyda esa
  **ma'lumot** (ikkinchisini ayirish bir pulni ikki marta sanardi); "Ombor"
  hisoboti — **oqim**, qoldiq emas.
- ⚠️ Har hisobot **o'z qamrovini** aytadi ("tannarx 1/19 taomda kiritilgan —
  yalpi foyda tushumning 30% ini qamraydi"): ikki yuzdan o'ntasini narxlagan
  restoran aks holda kechaning 6% ini tasvirlaydigan ustunga qarab qaror
  qabul qilardi.

### Xarid qarori: qancha va qachongacha

Xarid ro'yxati nima tugaganini bilardi, **qanchalik tez ketishini** esa yo'q.
Eng kam qoldiq — ega bir marta, tinch haftada chizgan chiziq: "javonda to'qqiz
kilodan kam qolsa ayting" deydi va to'qqiz kilo ertangi ovqatmi yoki keyingi
oyning zaxirasimi — ayta olmaydi. Shuning uchun ro'yxat buyruq emas, eslatma
edi, undagi miqdor esa (chiziqqacha yetkazish) eng kichik himoyalanadigan son
edi, to'g'risi emas.

Endi yonida ikkinchi qoida turadi va **kattarog'i g'olib** (`basis` qaysi biri
ekanini aytadi). Uchta o'lchov, uchalasi ham restoran allaqachon yozadigan
hujjatlardan (`internal/handlers/orderplan.go`):

1. **Hafta kunlari bo'yicha sarf** — `stock_movement` dan. ⚠️ **O'rtacha emas,
   aynan hafta kuni bo'yicha**: dushanba shanba emas. Tekis o'rtacha har hafta
   dam olish kunlarini kam, hafta boshini ko'p buyurtiradi — va aynan dam olish
   kunlari savdo bo'lgan joylarda. Prognoz **qoplanadigan kunlarni yuradi**:
   payshanbada olingan uch kunlik zaxira juma-shanba-yakshanba demakdir.
2. **Kirim ritmi** — o'sha masalliq kirimlari orasidagi **mediana** oraliq.
   ⚠️ **O'lchanadi, so'ralmaydi**: "lead time" maydoni — hech kim to'ldirmaydigan
   forma, va aslida savol "yetkazib beruvchi qancha kutadi" emas, "keyingi kirim
   odatda qachon keladi". Mediana, chunki bayramda o'tkazib yuborilgan bitta
   kirim o'rtachani buzadi.
3. **Yaroqlilik muddati** — kirim sanasi bilan `expiresAt` orasidagi mediana,
   sana yozilgan joyda. U **gorizontni qisqartiradi**: besh kun turadigan
   mahsulotga uch haftalik zaxira — to'la javon emas, kechiktirilgan chiqim.

**Gorizont uchta vaqt oralig'idan iborat**, va birinchisini tashlab ketish
birinchi versiyani kam buyurtiradigan qilgan edi. Bugun berilgan buyurtma bugun
javonga chiqmaydi: u **keyingi kirim bilan keladi**, keyin esa **undan
keyingisigacha** yetishi kerak:

```
gorizont = keyingi kirimgacha + bitta to'liq sikl + zaxira
         = (ritm − oxirgi kirimdan beri)  +  ritm  +  ritmning yarmi (max 7)
```

Bu — klassik davriy qayta ko'rish formulasi (lead time + review period +
safety), faqat uchala hadi ham **o'lchangan**, forma orqali yozilgan emas.
Birinchi versiya faqat siklni qoplagan: kirim endigina ketgan ertalab to'g'ri,
qolgan har ertalab bir haftagacha kam. So'ng muddat bilan cheklanadi.

`kerak = prognoz(gorizont) − javondagi − allaqachon so'ralgan`, va natija
**yuqoriga yaxlitlanadi** (dona — butun songa): 71.9 kilodan 71 ga tushirish
hech nima tejamaydi, ekran esa aynan o'zi oldini olishi kerak bo'lgan holatga
qaytadi. Uzun gorizont xavfsiz, chunki javondagi ham, so'ralgani ham ayiriladi —
kunda ikki marta ochilgan ro'yxat ikki marta buyurtma bermaydi.

⚠️ **Bo'sh javon — talab yo'qligi emas, tovar yo'qligi**, va ma'lumotda ikkalasi
bir xil ko'rinadi (nol). Bu — sodda prognozni **doimiy va bir tomonlama**
noto'g'ri qiladigan nuqta: uch kun tugab qolgan mahsulot "uch kun hech kimga
kerak bo'lmagan" bo'lib o'qiladi, demak kamroq buyurtiriladi, demak yana tugaydi.
Shuning uchun: filial **savdo qilgan** kunlar (`tradingDays`) bo'linuvchi bo'ladi,
va **har kuni sotiladigan** qator (savdo kunlarining ≥ 2/3 ida sotilgan) uchun
sotilmagan kunlar bo'linuvchidan **chiqariladi** hamda ekranda «N kun sotuvda
bo'lmagan — o'sha kunlarsiz hisoblandi» deb yoziladi. 2/3 chegarasi ataylab
baland: faqat dam olish kunlari sotiladigan qatorning nollari **haqiqiy**, va
ularni tugash deb o'qish yo'q talabni o'ylab topish bo'lardi.

⚠️ **Allaqachon so'ralgani ayiriladi** (`shopping_order`, `sent`). Ro'yxat
ertalab ham, tushdan keyin ham ochiladi — busiz ikkinchi o'qish bozorchi aynan
o'sha daqiqada qo'lida ushlab turgan narsani yana taklif qiladi, javonda esa
ikki barobar bo'ladi (muddatli mahsulotda bu bir haftada chiqim).

⚠️ **Biznes turiga qarab hech nima o'zgarmaydi, va bu ataylab** — `models/
businesstype.go` butun ombor moduli haqida shu chiziqni chizadi: biznes turiga
qarab o'zgaradigan arifmetika — hech kim tekshira olmaydigan arifmetika, va
aynan shu yerda noto'g'ri raqam **jimgina** bo'ladi. Farq **kodda emas,
ma'lumotda** ko'rinadi: dorixonaning kirimi ikki haftada bir, yogurtida muddat
bor — gorizonti uzun chiqadi va keyin qisqaradi; oshxonaning go'shti haftada
ikki marta keladi va hech qachon sanasi bo'lmaydi.

⚠️ **Kiyim do'koni o'zini o'zi himoya qiladi.** Ko'ylak bir marta sotiladi, va
unga "yana 3 ta oling" degan ishonchli maslahat — bema'nilik. Qoida turga emas
**ma'lumotga** qo'yilgan: prognoz gapirishi uchun qatorda kamida **uch xil
kunda** sotuv **va** kamida **ikkita kirim** bo'lishi kerak. Ikkalasi ham
yo'q — qator eski qoidaga (eng kam qoldiq) qaytadi, ya'ni ekran o'zgarmaydi.
Ikkinchi shart bir vaqtning o'zida ro'yxatning to'lib ketishidan saqlaydi:
kirimlarini yozmaydigan restoranda "keyingi kirimgacha" degan savolning ma'nosi
yo'q, va o'ylab topilgan bir haftalik gorizont butun katalogni har kuni
ro'yxatga chiqarardi.

⚠️ **Hisob qator yonida yozilgan** (`nega shuncha`: necha kunga, kuniga qancha,
kirim ritmi, muddat qisqartirganmi, allaqachon so'ralganmi). Ochib bo'lmaydigan
raqam — yo ko'r-ko'rona bajariladi, yo umuman o'qilmaydi; ikkalasi ham bu ekran
o'rnini bosgan daftardan yomonroq.

### Restoranning o'z ilovasi: bitta kod bazasi, har restoranga bitta build

Restoran «o'z ilovam bo'lsin» deganda mehmon telefonidagi **ikonkani** nazarda
tutadi. Bitta ilova ichida restoran tanlanadigan model buni bermaydi: u agregator
(Uzum Tezkor) modeli, va mehmon birinchi ekranda raqobatchilar ro'yxatini
ko'radi. Shuning uchun **bitta kod bazasi, N ta build** (`mobile/guest-android`).

- ⚠️ **Farq bitta faylda: `app/brand.properties`** (+ ikkita rasm: logo va
  ikonka). Brendlash uchun Gradle'ni, manifestni, temani va uchta Kotlin faylini
  tahrirlashi kerak bo'lgan quvurning nosozligini o'qib bo'lmaydi va diffini
  ko'rib bo'lmaydi.
- ⚠️ **Qiymatlar tenantning o'z `GET /restaurant` javobidan olinadi** (`name`,
  `logoUrl`, `theme.brand`), konsolda qayta yozilmaydi. Ikkinchi nusxa — mehmon
  saytda ko'rgan narsa bilan bir kun ziddiyatga tushadigan narsa.
- ⚠️ **Server manzili build'ga muhrlanadi, so'ralmaydi.** Xodim ilovalari server
  ekrani bilan ochiladi, chunki menejerga manzilni aytish mumkin; mehmondan
  menyudan **oldin** hech nima so'rab bo'lmaydi. Bu esa avtomatik ravishda «har
  restoranga alohida build» degani — tanlov emas, oqibat.
- ⚠️ **Bosh sahifa yo'q, birinchi ekran — menyu.** Restoran ilovasini ochgan
  odam och. Muqova, ish vaqti va «Buyurtma berish» tugmasi — odam bilan menyu
  orasidagi bitta bosish va bitta scroll, va har biri buyurtma yo'qoladigan joy.
- ⚠️ **Yopiq restoran ham menyusini ko'rsatadi.** Uni yashirish bugungi
  xafagarchilikni tejab, ertangi mehmonni yo'qotadi. Sarlavha yopiqligini
  aytadi, menyu joyida qoladi. Tugagan taom ham chiziladi, olib tashlanmaydi:
  yo'q bo'lib qolgan taom mehmonni «men adashganman shekilli» deb o'ylatadi.

#### Birinchi jonli build to'rtta jim nosozlikni ko'rsatdi

Konsoldagi tugma birinchi marta bosilganda «build tugadi, lekin fayl topilmadi:
Preparing "Install Android SDK Build-Tools 35 v.35.0.0"» chiqdi. To'rtta alohida
xato, va **hech biri xato xabari bermagan**:

- ⚠️ **Artefakt yo'li log'ning oxirgi qatoridan o'qilardi.** Docker stdout va
  stderr'ni yozilish vaqti bo'yicha aralashtiradi, Android vositalari esa
  progressni stderr'ga yozadi — ya'ni yo'l chop etilgan, keyin ustiga boshqa
  narsa chop etilgan. Endi **belgilangan qator** (`KEEL_ARTIFACT=`), va o'quvchi
  prefiksni qidiradi, joyni emas. Belgi bo'lmasa — bo'sh, va **log saqlanadi**:
  birinchi versiya uni bir qatorli xabar bilan almashtirardi, ya'ni aynan
  kerakli paytda dalilni yo'q qilardi.
- ⚠️ **AGP build-tools 35.0.0 so'raydi, `compileSdk = 36` bo'lsa ham.** Image'da
  faqat 36.0.0 bor edi, demak har build uni yuklab olardi — «hech nima yuklab
  olinmaydi» degan izohning tagida. Ikkalasi ham pinlandi.
- ⚠️ **Konteynerda DNS faqat IPv6 qaytaradi, ko'prikda esa IPv6 chiqishi yo'q.**
  `curl` buni sezib bir millisekundda IPv4'ga o'tadi; **Java o'tmaydi** —
  `sdkmanager` «Still waiting for package manifests to be fetched remotely» deb
  cheksiz turadi. Bu **osilish**, yiqilish emas: xato yo'q, chiqish kodi yo'q,
  log'da esa sabrga o'xshagan ogohlantirish. Yechim —
  `JAVA_TOOL_OPTIONS=-Djava.net.preferIPv4Stack=true` (⚠️ `GRADLE_OPTS` emas:
  osilgan vosita `sdkmanager` edi, unga Gradle sozlamalari yetib bormaydi).
- ⚠️ **Xotira arifmetikasi o'zim bilan qarama-qarshi edi**: konteyner 3 GB,
  Gradle'ga 2 GB heap, Kotlin daemon'ga yana 1 GB — ikkita JVM'ning **faqat
  heap'lari** shiftga teng. Yadro birini o'ldirdi va Gradle buni «daemon
  disappeared unexpectedly» deb aytdi — bu xabar na xotirani, na cheklovni
  aytadi. Endi Kotlin **bitta JVM ichida** (`kotlin.compiler.execution.strategy=
  in-process`), heap 2 GB, konteyner 3.5 GB.
- ⚠️ **SDK'ni build paytida yuklab olish o'chirildi**
  (`android.builder.sdkDownload=false`): yetishmayotgan komponent endi **baland
  xato** beradi va Dockerfile'dagi bir qatorda tuzatiladi — jimgina qirq
  daqiqalik navbat egallash o'rniga.

Va bittasi ko'rinadigan: **tanlangan APK/AAB chipi oq ustiga oq** edi
(`bg-brand text-white` — bu konsolda och aksent, tugma foni emas). Qolgan konsol
tanlovni ramka va aksentning o'ndan biri bilan belgilaydi; endi bu ham shunday.


#### Oltinchi jim nosozlik: olti daqiqalik ishni oltmish soniyalik mijoz kutardi

Konsoldan qilingan har bir build **aynan 61 soniyada** «tugardi» va fayl
topilmasdi. Sabab `provision.Client` ning umumiy HTTP mijozidagi
`Timeout: 60 * time.Second`.

- ⚠️ **Bu timeout qolgan hamma chaqiruv uchun to'g'ri**: Docker'ning boshqa har
  bir so'rovi millisekundlarda javob beradi, va qisqa muddat qotib qolgan
  soketni **osilish** emas, **xato** qilib ko'rsatadi.
- ⚠️ **`/containers/{id}/wait` esa uzun poll**: to'qqiz daqiqa — normal holat.
  Ikkalasi bitta mijozda bo'lishi mumkin emas edi.
- ⚠️ **Va nosozlik hech qayerda «timeout» demasdi**: so'rov uzilardi, konteyner
  fonda kompilyatsiyani davom ettirardi, konsol esa yarim yo'lda to'xtagan log
  bilan «fayl topilmadi» deb yozardi. Uchta ko'rsatkichning uchtasi ham boshqa
  narsani ko'rsatardi.
- Endi ikkinchi mijoz — **o'z muddatisiz** (`poll`), va yagona soat chaqiruvchining
  konteksti (`RunSpec.Timeout`).

Yonida yana bittasi, xuddi shu sessiyada: **xato sababi log ostida qolib
ketardi.** Kod sababni log oldiga qo'yib, keyin **oxirgi** qirq qatorni olardi —
ya'ni birinchi bo'lib aynan sababni tashlardi, va qirq qator Gradle vazifasi
qolardi. Endi sabab birinchi, log keyin, va testi bor.


#### Yettinchisi: 30 soniyalik `WriteTimeout` va 2,5 MB lik fayl

«Yuklab olish» bosilgandi, «Olinmoqda…» chiqardi, keyin hech nima bo'lmagandek
tugmaga qaytardi. Jurnal aniq aytdi: `200`, lekin **2 162 104 bayt** yuborilgan
va **aynan 30,011 soniya** ketgan.

Sabab — konsolning `http.Server{WriteTimeout: 30 * time.Second}`. Bu qolgan har
bir endpoint uchun **to'g'ri**: javob bir necha kilobayt JSON, va sekin yozuv
ulanishni ushlab turgan mijoz demakdir. APK esa ikki yarim megabayt, va u odam
qanday internetda bo'lsa o'shanda ketadi.

- ⚠️ **Faqat shu javob uchun uzaytiriladi** (`http.ResponseController`), butun
  server uchun emas: himoya qolgan hamma joyda kerak.
- ⚠️ **Uzoqroq muddat, muddatsizlik emas**: o'qishni to'xtatgan mijoz ulanishni
  baribir qo'yib yuborishi kerak. O'n daqiqa — bir necha megabayt uchun juda
  ko'p, abadiylikdan esa juda kam.
- ⚠️ **Yarim ketgan uzatish endi yozib qo'yiladi.** «Faqat to'liq ketganda
  o'chirilsin» qoidasi **ishladi** — artefakt saqlanib qoldi, va bu to'g'ri
  edi — lekin **hech qayerda nima uchunligi yozilmagandi**, ya'ni konsolda
  shunchaki avvalgi holatiga qaytgan tugma ko'rindi. Hech kim tushuntira
  olmaydigan qator — qator yo'qligi bilan barobar.


#### Konsoldagi tugma: navbat, va yuklab olingan fayl o'chiriladi

- ⚠️ **Artefakt yuklab olingan zahoti o'chiriladi, yozuvi esa qoladi.** Har build
  2,5 MB, va hech kim tozalamaydigan papka har mijoz uchun ishlaydigan mashinada
  turadi. Lekin «do'konda qaysi versiya turibdi?» degan savol oylar keyin
  beriladi, va fayl tizimi unga javob bera olmaydi — shuning uchun qatorda
  versiya, SHA-256 va **ikkala ism** (kim bosgan, kim olgan) qoladi.
- ⚠️ **Uzatish tugagandan keyin o'chiriladi, boshlanganda emas.** Birinchi baytda
  o'chirish uzilgan ulanishni yana to'qqiz daqiqalik build'ga aylantiradi, va
  ikkinchi urinish hech nima topmaydi — bu konsol faylni yo'qotgandek o'qiladi.
- ⚠️ **`taken` — o'z holati, `ready` yonidagi bayroq emas.** «Fayl bor» va «fayl
  bor edi» — konsol tugmani shundan chizadi, va holat yonidagi boolean ikki
  fakt: ular ajralganda 404 qaytaradigan yuklab olish tugmasi paydo bo'ladi.
- ⚠️ **`latest.apk` nusxasi olib tashlandi.** O'chirishdan omon qolgan ikkinchi
  nusxa aynan o'chirish oldini olmoqchi bo'lgan narsani qiladi: abadiy qoladi.
- ⚠️ **Yuklab olish `<a href>` emas, `fetch`.** Endpoint konsolning bearer
  tokenini talab qiladi, oddiy havola esa hech nima yubormaydi — bu odatiy
  ko'rinadigan tugmada «sessiya tugagan» bo'lib chiqadi.
- ⚠️ **Navbat bittalik va ishchi bitta.** Bu navbat bo'lishining butun sababi —
  mashina ikkitasini ko'tara olmaydi; o'lchami bir bo'lgan pool esa kimdir
  «foydali» ravishda oshiradigan pool.
- ⚠️ **Bir mijozga bitta tugallanmagan build.** Tugma ko'rinishda o'zgarmasa odam
  ikki marta bosadi, va ikkinchi bosish yana to'qqiz daqiqa sarflab bayt-bayt
  bir xil fayl chiqaradi.
- ⚠️ **`versionCode` o'z yozuvlarimizdan sanaladi, vaqtdan yoki qatorlar sonidan
  emas.** Play kodi oldingisidan **qat'iy katta** bo'lmagan yuklamani rad etadi,
  va rad javobi odam kutgan yuklashning oxirida keladi; qatorlar soni esa
  birinchi o'chirilgan qatordan keyin takrorlanadi.
- ⚠️ **Format oldindan so'raladi.** APK telefonga o'rnatiladi, AAB esa faqat
  Play uchun va umuman o'rnatilmaydi — noto'g'ri tanlov to'qqiz daqiqani
  yo'qotadi va yuklash oxirida bilinadi.
- ⚠️ **Bo'sh `APP_BUILD_IMAGE` — imkoniyat o'chiq**, bo'sh `DOCKER_SOCKET`
  provisioningni o'chirgani bilan bir qoida: noutbukda Android toolchain yo'q va
  konsol baribir ochilishi kerak. Tugma sababini aytadi.


#### Build quvuri: bitta image, bitta qulf, bir marta yaratiladigan kalit

`deploy/appbuild/` — Dockerfile (JDK 17 + Android SDK 36 + Pillow), `build.sh`
va `brandimages.py`. Konsol tugmasi shu konteynerni ishga tushiradi.

- ⚠️ **Host'ga JDK yoki SDK o'rnatilmaydi.** O'sha mashina har bir tenantning
  konteynerini tutib turadi; yonida o'rnatilgan toolchain bir `apt upgrade` dan
  keyin hech kim takrorlay olmaydigan build xatosiga aylanadi. Image ichida
  toolchain — image'ning fakti, va uning tegi bor.
- ⚠️ **Imzo kaliti bir marta yaratiladi va hech qachon almashtirilmaydi.**
  Yo'qolgan kalit bilan imzolangan ilovani hech kim yangilay olmaydi: na biz, na
  restoran, na Google. Skriptda **yangi kalit so'rash yo'li ataylab yo'q**, va
  yaratilgan payt buni har safar ekranga yozadi — hech kim bilmagan kalitni hech
  kim zaxiralamaydi, va u kerak bo'lgan kun — u yo'qolgan kun.
- ⚠️ **`applicationId` slug'dan, nomdan emas.** Restoran nomini o'zgartiradi;
  `applicationId` esa o'zgara olmaydi — o'zgarsa bu yangi ilova, eskisining
  yoniga o'rnatiladi, va har mehmonda o'lik nusxa qoladi.
- ⚠️ **Qulf skriptning o'zida, chaqiruvchida emas.** Konsol navbat qiladi, lekin
  faqat chaqiruvchida turgan qulfni shell'i bor odam yonidan o'tib ketadi — va
  bu aynan oldini olishi kerak bo'lgan to'qnashuv. Yo'li **absolut**
  (`/opt/keel/.appbuild.lock`): `${HOME}` serverni emas, foydalanuvchini
  qulflaydi, va deploy skripti bu darsni allaqachon to'lagan.
- ⚠️ **Rad etmaydi, kutadi** (`flock --timeout 3600`): ikki build bir vaqtda
  mashinani swap'ga tushiradi, lekin ba'zan hech nima qilmaydigan tugma
  undan yomonroq.
- ⚠️ **Manba nusxa ko'chiriladi, checkout ichida qurilmaydi.** Ikki build bitta
  daraxtda bir-birining `brand.properties` ini yozadi va bir restoranning
  ilovasini boshqasining nomi ostida chiqaradi; yiqilgan build esa checkout'da
  birovning ranglarini qoldiradi. Dizayn moduli ham ko'chiriladi va **o'z
  joyini saqlaydi** — u nisbiy yo'l bilan ulanadi.
- ⚠️ **Konteynerga `TZ` beriladi.** Zonasi yo'q konteyner host nima bo'lishidan
  qat'i nazar UTC: soat uchdan o'nda tugagan build 10:10 deb fayllanadi — besh
  soat farq bilan, aynan ikki build'ni ajratish uchun qaralayotgan joyda. Bu
  `cmd/server` dagi `time/tzdata` tuzog'ining boshqa xonadagi ko'rinishi.
- ⚠️ **Profil o'ralgan javob** (`{restaurant, brand, branch, …}`), va `.name` ni
  yuqoridan o'qish `null` beradi — jimgina, chunki `jq` hech nima topmasa ham
  xursand. Ilova modeli ham xuddi shu xatoni qilgan edi: dekodlash o'tadi, hech
  nima yiqilmaydi, va nomsiz restoran doim ochiq bo'lib Afrika sohilidagi nol
  nuqtada turadi. `restaurant` — brend va filial ustiga qo'yilgan **birlashgan**
  ko'rinish, ya'ni mehmon ko'radigan yuz.


#### Ilova mahsulot qarorlari qayta ko'rib chiqildi (ega talabi bilan)

Uchta qaror ataylab bekor qilindi — sabablari yozilgan edi, lekin qaror egasi
boshqacha hal qildi. Ikkalasining ham narxi bor va u shu yerda qoladi:

- **Menyu bir qator emas, ikki ustun.** Ilgari to'liq enli qator edi: uzun
  nomlar («Qo'y go'shtli qazon kabob») to'liq sig'sin uchun. Endi ekranga ikki
  barobar ko'p taom sig'adi, nom esa ikki qatorga cheklanadi — menyu o'qishdan
  ko'ra ko'proq **qarab** tanlanadi.
  - ⚠️ **`LazyVerticalGrid` emas, `chunked(2)`**: bitta o'qda ikkita lazy
    scroller — ishga tushganda qulash, va kategoriya sarlavhalari qatorlar
    orasida to'liq enli qolishi kerak.
  - ⚠️ **Kartochkadagi stepper faqat «sof» qatorni tahrirlaydi** (variantsiz,
    izohsiz). Minus birovning «piyozsiz» yozuvini jimgina o'chirsa, bu kartochka
    o'zi ko'rsatmagan qarorni tahrirlagan bo'lardi. Sof qator bo'lmasa — taom
    oynasi ochiladi.
  - ⚠️ **Majburiy savoli bor taom `+` bilan savatga tushmaydi**, oyna ochiladi:
    aks holda o'lchami tanlanmagan pitsa buyurtmaga tushardi.
- **Savat hisobga kirishni talab qiladi.** Ilgari buyurtma hisobsiz ham
  berilardi («narx ko'rsatishdan oldin telefon so'raydigan ilova bitta marta
  ochiladi»). Endi restoran buyurtmani qaytib qo'ng'iroq qila oladigan odamga
  bog'laydi.
  - ⚠️ **Savat umuman ko'rsatilmaydi, oxirida rad etilmaydi.** Savatni
    to'ldirtirib, checkout'da kirish talab qilish — ikki ekranning eng yomon
    tartibi: ish allaqachon qilingan, talab esa odam to'lovga eng yaqin
    turganda keladi.
  - ⚠️ **Nima uchunligi yoziladi**: sababsiz «kiring» — och odam bilan ovqat
    orasidagi forma; «restoran qayta qo'ng'iroq qilishi uchun» — odam rozi
    bo'ladigan gap.
- **Ikonka butunlay logotip.** Ilgari oq fon + o'rtada logotip edi (qorong'i
  siyohli logotip qorong'i fonda ko'rinmaydi degan sabab bilan). Endi logotip
  ikonkani to'ldiradi.
  - ⚠️ **Narxi — chetlari**: to'ldirilgan tasvirni launcher niqobi yana qirqadi,
    ya'ni tashqi o'n sakkizdan bir qismi yo'qoladi va keng wordmark uchlarini
    yo'qotadi. Logotipi buni ko'tara olmaydigan restoranga ikonka uchun kvadrat
    belgi kerak.
  - ⚠️ **Shaffoflik oq ustiga tekislanadi**: siyoh bilan chizilgan logotip aks
    holda ikonkani qora qilib to'ldirardi.
  - ⚠️ Logotip **fon qatlamiga** qo'yiladi, old qatlam shaffof: adaptiv
    ikonkaning old qatlami niqob ichida hoshiya bilan chiziladi, ya'ni u yerga
    qo'yilgan «to'liq» logotip yana kichrayardi.

#### Xarita: uchala provayder ham, tanlov restoranniki

Sayt `restaurant.mapProvider` bo'yicha 2GIS / Yandex / Google chizadi; ilova ham
endi shunday — **ish vaqtida** o'qiydi, build vaqtida emas. Aks holda provayderni
panelda almashtirgan restoran yangi relizni kutardi, va shu orada uning sayti
bilan ilovasi xarita haqida kelishmasdi.

- ⚠️ **2GIS — WebView, qolgan ikkitasi native, va bu tanlov emas.** 2GIS'ning
  olinadigan Android SDK'si yo'q (Maven yo'llari 404), demak 2GIS tanlagan
  restoranning tanlovini hurmat qilishning yagona yo'li — uning o'z veb
  dvigateli (MapGL), ya'ni sayt chizadigan o'sha dvigatel. Qolgan ikki variant
  yomonroq edi: «2GIS restoraniga Google xaritasini berish» va «boshqa provayder
  tanlashni aytish».
- ⚠️ **Koordinata tartibi faqat `MapEngines.kt` da**: 2GIS `[lng, lat]`, Yandex
  `Point(lat, lng)`, Google `LatLng(lat, lng)`. Uni chaqiruvchi tomonda o'girish —
  CLAUDE.md yozib qo'ygan xato: restoran Orol dengizida paydo bo'ladi va bu
  ma'lumot xatosiga o'xshaydi.
- ⚠️ **Yandex kaliti `initialize` dan oldin va jarayonda bir marta** beriladi;
  SDK qayta kalitlanishni rad etadi. Ya'ni kalitini almashtirgan restoranga
  ilovani qayta ishga tushirish kerak — bu SDK bergan xatti-harakat, biz
  tanlagani emas.
- ⚠️ **MapKit tinglovchini `WeakReference` bilan oladi** va kuchli havola
  bizniki bo'lishi shart (`remember`). Busiz birinchi GC'dan keyin callback
  jimgina to'xtaydi va manzil yangilanmay qoladi.
- ⚠️ **2GIS WebView'i haqiqiy `baseUrl` bilan yuklanadi**: MapGL kalitning domen
  cheklovini tekshiradi, va hech qayerdan yuklangan sahifa bo'sh xarita va
  restoranda hech kim o'qimaydigan konsol xabari bilan rad etiladi.
- ⚠️ **Reverse-geocode bitta yo'ldan va debounce bilan** — uchala dvigatel ham
  bitta `centre` holatiga yozadi. Nominatim sekundiga bitta so'rovga cheklaydi,
  va uni buzish butun o'rnatma bazasini bir vaqtda bloklatadi.
- ⚠️ **Narxi APK hajmida, va u ProGuard ko'rmaydigan joyda.** Yandex MapKit har
  ABI uchun alohida native kutubxona olib yuradi (`libmaps-mobile.so`, 19–29 MB),
  native kod esa minify va shrink'dan butunlay tashqarida — ya'ni release build
  debug bilan bir xil semiz bo'ladi. Birinchi jonli build **101 MB** chiqdi, va
  bu «release qilinmayapti» bo'lib ko'rinadi.
  - `abiFilters` **arm64-v8a va armeabi-v7a** bilan cheklaydi: x86 va x86_64 —
    emulyator va Chromebook, bu yerda mehmonning telefoni emas. 58 MB shu yerda
    ketadi.
  - ⚠️ **`armeabi-v7a` qoldiriladi** (yana 19 MB): minSdk 26, va 32-bitli
    Android 8–10 telefonlari aynan shu ilova yetib borishi kerak bo'lgan arzon
    apparatlar. Sinovda buni sezib bo'lmaydi — arzon telefon o'rnata olmaydi,
    xolos.
  - AAB'ga ta'siri yo'q: Play baribir qurilmaga bitta ABI beradi. arm64
    telefonga tushadigan yuklama ~13 MB (MapKit'ning o'zi siqilganda 11 MB),
    diskda ~29 MB. Katta raqam — **to'g'ridan-to'g'ri APK** yuklab beradigan
    restoranning muammosi.
  - ⚠️ **MapKit'ni provayder bo'yicha shartli qilish rad etildi** (ega qarori):
    AAB Play'da baribir bo'linadi, va shartli qilish «Yandex'ga o'tdim, xarita
    hali ham 2GIS» degan holatni tug'dirardi — uni faqat qayta build tuzatadi.
  - ⚠️ **Hajm ishlash tezligiga ta'sir qilmaydi, o'rnatishlar soniga qiladi.**
    `libmaps-mobile.so` faqat manzil tanlash ekrani ochilganda yuklanadi
    (`System.loadLibrary`), ya'ni menyuning sovuq ishga tushishida umuman
    qatnashmaydi. Yo'qotish do'kon sahifasida bo'ladi — har qo'shimcha megabayt
    o'rnatish foizini tushiradi, va buni hech bir profiler ko'rsatmaydi.


#### Savat, ma'lumot to'ldirish va sozlamalar: ilova saytga tenglashtirildi

⚠️ **«So'rov formati noto'g'ri» — pul `Double` bo'lgani uchun edi.** Ilovada pul
`Double` (dizayn moduli `money()` shunday oladi), serverda esa `int`. kotlinx
`5000.0` deb yozadi, Go esa uni `int` maydonga qabul qilmaydi va **butun
so'rovni** rad etadi — maydonning nomini aytmagan holda. `usePoints: 0.0` ning
o'zi yetarli edi: har bir quote yiqilardi, va mehmon savatda qizil xato ko'rardi.

- ⚠️ **Chegara `toLong()` bilan yopildi, ilovaning tiplari o'zgartirilmadi**:
  `Double` beshta ilova bo'lishadigan `money()` dan o'tadi.
- ⚠️ **So'rov tanalari alohida funksiyaga chiqarildi** (`quoteBody`, `orderBody`,
  `deliveryQuoteBody`) — aynan shu xatoni **testdan yashirgan narsa** ular
  `suspend` funksiya ichida yozilgani edi. Endi `WireShapeTest` daraxtni aylanib
  chiqadi va nuqtali har qanday raqamda yiqiladi; `lat`/`lng` — yagona istisno,
  chunki serverda ular haqiqatan `float64`.

⚠️ **Savat sayt bilan bir chizmada** (`(site)/cart/page.tsx`): qatorlar bitta
kartada, jami ikkinchisida — «taomlar soni + summa», «yetkazib berish —
rasmiylashtirishda», chiziq, **Jami**. Sabab: bu bir odamning bir ekrani, va
ikkisining ajralib ketishi «ilova yomonroq» degan taassurotning boshlanishi.

- ⚠️ **`lineId` qo'shildi va ro'yxat o'shanga bog'landi.** Izoh endi savatda
  yoziladi, izoh esa `key` ning bir qismi — ya'ni `key` bo'yicha kalitlangan
  `LazyColumn` har harfdan keyin maydonni qayta quradi va kursorni yo'qotadi.
  Sayt ham xuddi shu muammoni xuddi shunday yechgan.
- ⚠️ **Birlik narxi ham, qator jami ham ko'rsatiladi**: yolg'iz jami «58 000»
  ikkitalik qator yonida bitta taomning narxi bo'lib o'qiladi.
- ⚠️ **Yetkazib berish nol emas, «rasmiylashtirishda»**: manzil hali yo'q,
  zonalar serverniki — nol yozish hisoblanmagan narxni e'lon qilish bo'lardi.

⚠️ **Checkout hisobga kirgan mehmonga bo'sh forma ko'rsatmaydi**: ism, telefon
va ilgari ishlatilgan manzillar `/users/me` dan keladi. Ikki ekran oldin kirgan
ilovada telefon raqamini qayta terish — mehmon saytga qaytadigan xil kichik
haqorat.

- ⚠️ **Faqat bo'sh maydonga yoziladi**: profil mehmon yozishni boshlagandan keyin
  ham kelishi mumkin, va yozganini almashtirish — u bilan tortishadigan forma.
- ⚠️ **Manzil ro'yxati birinchi, xarita ikkinchi**: har hafta bitta kvartiraga
  buyurtma beradigan odam bir bosishda tugatishi kerak.
- ⚠️ **Manzilni saqlash — buyurtmadan keyin va uni hech qachon yiqita olmaydi**:
  oshxona qabul qilgan buyurtmani profil yozuvi tufayli «xato» deb ko'rsatish bu
  fayldagi eng yomon savdo bo'lardi.
- ⚠️ **`PUT /users/me` butun ro'yxatni almashtiradi** — shuning uchun har saqlash
  `addresses` ni to'liq qaytaradi. Faqat ismni yuborgan saqlash mehmonning
  hamma manzilini jimgina o'chirardi.

⚠️ **Hisob bo'limida «Sozlamalar»**: ism, manzillar, til, mavzu, bildirishnoma,
versiya. Ilgari mavzu yolg'iz turardi, ism va manzilni esa **faqat yangi
buyurtma berib** o'zgartirib bo'lardi, til esa sarlavhadagi almashtirgichda edi.
Topib bo'lmaydigan sozlama — yo'q sozlama.

- ⚠️ **Til serverga ham aytiladi** (`PUT /users/me/lang`): buyurtma haqidagi
  xabarlarni server yozadi va o'zi tarjima qiladi, ya'ni faqat telefonda
  saqlangan til ilova yopiq turganda keladigan push'ga umuman ta'sir qilmaydi.
- ⚠️ **Bildirishnoma holati Android'dan so'raladi**, saqlangan bayroqdan emas:
  mehmon uni tizim sozlamalarida qaytarib olishi mumkin, va «yoqilgan» deb turgan
  ekran o'zi va'da qilgan yagona narsa haqida yolg'on gapiradi.
- ⚠️ **Telefon raqami tahrirlanmaydi** — uni almashtirish ikkinchi SMS aylanmasi
  va boshqa endpoint. Tahrirlanadiganday ko'rinib, eski raqamni saqlab qoladigan
  maydon — umuman bo'lmaganidan yomonroq.


#### Xarita: Google, va CLAUDE.md dagi narx ogohlantirishi bu yerga tegmaydi

⚠️ **U ogohlantirish veb API'lari haqida** — MapGL, Maps JavaScript API, Static
Maps — ular har xarita ochilishi uchun pul oladi. **Android SDK'ning xarita
ko'rsatishi umuman hisoblanmaydi**, ya'ni sayt 2GIS'ni standart qilgan yagona
sabab bu ekranga yetib kelmaydi. (2GIS'ning ochiq Android maven yo'li ham
mavjud emas edi.)

- ⚠️ **Geokodlash baribir Nominatim'da qoladi** (`lib/geocode.ts` bilan bir
  xizmat): Geocoding va Places API'lari **hisoblanadi**, va qidiruv maydoni har
  bosishda bitta so'rov yuboradi. Sayt bu bo'linishni allaqachon shu sababdan
  qilgan. Yon foydasi: xarita kaliti umuman yo'q restoranda ham qidiruv ishlaydi.
- ⚠️ **Nominatim haqiqiy `User-Agent` so'raydi va sekundiga bitta so'rovga
  cheklaydi.** Ikkalasini ham e'tiborsiz qoldirgan ilova butun o'rnatma bazasini
  bir vaqtda bloklatadi — bu «qidiruv ishlamayapti» bo'lib har mehmondan bir
  vaqtda keladi va siyosat muammosiga umuman o'xshamaydi. Shuning uchun kamerada
  debounce (600 ms) va yozishda debounce (400 ms).
- ⚠️ ~~**Bitta provayder, build vaqtida hal qilinadi.**~~ **Bekor qilindi:**
  ilova endi uchalasini ham olib yuradi va tanlovni `mapProvider` dan ish
  vaqtida o'qiydi — yuqoridagi «Xarita: uchala provayder ham» bo'limiga qarang.
  Hajm dalili yolg'on chiqdi: 2GIS'da Android SDK yo'q (u WebView'da chiziladi),
  ya'ni qo'shilgani bitta MapKit.
- ⚠️ **Kalit ilova bo'yicha umuman cheklanmaydi, va bu narx.** Google kalitida
  **bitta** ilova cheklovi bo'ladi — HTTP referer **yoki** Android paket nomi,
  ikkalasi emas. Ilova endi saytning o'z `mapGoogleKey` ini ishlatadi (alohida
  `mapAndroidKey` maydoni olib tashlandi, chunki ega uchun bu ikkinchi kabinet,
  ikkinchi cheklov va javobi yo'q savol edi), demak umumiy kalit **cheklovsiz**
  qolishi va **ikkala** API'ga (Maps JavaScript va Maps SDK for Android) ruxsat
  berishi kerak. Domenga cheklab qo'yilsa Android SDK uni **jimgina** rad etadi:
  kulrang to'r, hech qayerda xato yo'q, va manzilini kirita olmaydigan mehmon.
  Zarar doirasini API cheklovi ushlab turadi (ikkalasi ham hisoblanadigan API
  emas: Android SDK'ning ko'rsatishi bepul), va buni xohlamagan restoranga javob
  — Google konsolida **ikkinchi kalit**, bu yerda ikkinchi maydon emas.
- ⚠️ **Nishon qimirlamaydi — xarita qimirlaydi.** Suriladigan marker barmoq
  aynan qo'yilayotgan narsani yopishi demakdir, telefonda esa oxirgi bir necha
  metr eng muhimi. Nishon markazda qotib turadi, xarita ostidan siljiydi — bu
  yerdagi har bir yetkazish ilovasi shunday qiladi.
- ⚠️ **Joylashuvga ruxsat ishga tushishda emas, xaritada so'raladi.** Kimdir
  «yetkazib berish» deyishidan oldin so'ralgan ruxsatni ko'pchilik bir marta va
  butunlay rad etadi, va rad etish qaytarilmaydi.

#### Hisob ixtiyoriy, va bu dizayn — kamchilik emas

Menyu, savat va **buyurtma berish** — hammasi hisobsiz ishlaydi. Narx
ko'rsatishdan oldin telefon raqami so'raydigan ilova bitta marta ochiladi.
Kirish qo'shadigan narsa: buyurtmalar tarixi, ballar va sevimlilar.

- ⚠️ **Kirish ekranidagi matn nima berishini aytadi, nega kerakligini emas** — u
  kerak emas, va aks holda ikki marta hisobsiz buyurtma bergan odamga yolg'on
  gapirgan bo'lardik.
- ⚠️ **Telefon raqami yozilayotganda qayta formatlanmaydi.** Har kimning o'z
  odati bor (`+998`, `998`, boshida nol, probellar), va yozilayotgan narsani
  qayta yozadigan maydon bilan odam kurashadi. Normallashtirishni server qiladi
  — qoida allaqachon o'sha yerda.
- ⚠️ **Ism kod bosqichida so'raladi**, undan oldin emas: server uni faqat yangi
  hisob uchun o'qiydi, ya'ni qaytgan mehmon bu maydonni umuman ko'rmaydi.
- ⚠️ **«Qayta yuborish» sanoq bilan.** Jimgina hech nima qilmaydigan tugmani
  odam besh marta bosadi, va har bosish yo restoranga SMS turadi, yo ilovani
  buzuq deb o'rgatadi.
- ⚠️ **Yurakcha hisobsiz umuman chizilmaydi.** Kirishga olib boradigan yurakcha —
  menyuda telefon raqami so'rash demakdir.

#### Push: `google-services.json` yo'q, to'rtta satr bor

Firebase'ning Gradle plagini `google-services.json` ni build vaqtida o'qiydi va
`applicationId` ga mos mijoz yozuvi bo'lmasa **build'ni rad etadi**. Har
restoranga alohida build modelida bu har restoranga generatsiya qilingan fayl,
va ikkalasi ajralganda paket nomi haqidagi xato xabari degani.

Firebase o'sha qiymatlarni **kodda** ham qabul qiladi (`FirebaseOptions`), va
to'rtta satr `brand.properties` da — brendlanadigan qolgan hamma narsa bilan
yonma-yon. Plagin ham, fayl ham kerak emas.

- ⚠️ **Bo'sh qiymat — «push yo'q», qulash emas** (xarita kaliti bilan bir qoida):
  menyusini hech kim o'qiy olmaydigan ilova — hech kim uni o'rnatgan sabab
  bo'lmagan imkoniyat uchun — ancha yomon nosozlik.
- ⚠️ **Ruxsat kirgandan **keyin** so'raladi, ishga tushishda emas.** Android 13
  hech nima buyurtma qilmagan odam oldiga kontekstsiz ha/yo'q savolini qo'yadi;
  ko'pchilik bir marta va butunlay yo'q deydi, va rad etish buyurtma
  xabarlarini ham olib ketadi.
- ⚠️ **Bitta xabar turi, va cheklov — imkoniyatning o'zi.** Mehmonda bu ilova
  uchun bitta rubilnik bor, va kelajakda yubormoqchi bo'lgan hamma narsa
  (aksiya, eslatma, yangi menyu) o'sha rubilnikdan o'tadi. Uni o'z buyurtmasidan
  boshqa narsaga sarflasak, u o'chiriladi va buyurtma xabarlari ham u bilan
  ketadi.
- ⚠️ **Kanal telefonda yaratiladi va id serverникi bilan bir xil** (`orders`).
  Yaratilmagan kanalga kelgan xabar ovozsiz va tartibsiz keladi — bu hech kim
  yubormagan xabardan farq qilmaydi va server buzuq deb tashxis qo'yiladi.
- ⚠️ **Mehmon qurilmasi — beshinchi jadval** (`user_device`), `push_subscription`
  ichiga qo'shilmadi: u brauzer obunasi (endpoint + ikki kalit), bu esa FCM
  tokeni. Bitta maydon ikki xil manzil tutishi — bu kitobda allaqachon to'rt
  marta yozilgan xato.
- ⚠️ **Telegram xabari bilan yonma-yon, o'rniga emas.** Botdan buyurtma bergan
  odam botni o'qiydi, ilova o'rnatgan odam ilovani. Ikkalasini ham qilgan mehmon
  ikkita oladi — bu halol natija, va qaysi birini nazarda tutganini taxmin
  qilishdan ancha yaxshi. «Qaysi holatlar xabar bo'ladi» esa **bitta**
  funksiyadan so'raladi (`notifiableStatus`).


#### Kanal: `android` o'ziniki, `web` ichiga qo'shilmaydi

Restoran ilova uchun pul to'laydi va u haqda **bitta** savol beradi: «uni
kimdir ishlatyaptimi?». `orderChannel` bu so'zni bilmaguncha javob panelning
hech qayerida yo'q edi, va u fikr ko'rinishida kelardi. Endi buyurtmada o'z
nishoni bor (`ChannelBadge`) va kanallar hisobotida o'z qatori.


#### Rang restoranniki, apelsin Keel'niki

`KeelWaiterTheme(accent = ...)` — dizayn moduli ataylab **faqat shu ilovada**
ustidan yozadi. Kassa, peshtaxta va kuryer telefoni — **bizning asbobimiz**, va
undagi apelsin bosilgan chek bilan mos kelishi kerak; restoran rangiga
bo'yalgan kassa har o'rnatmani qo'llab-quvvatlovchi odam uchun boshqa mahsulotga
aylantiradi. Mehmon ilovasi esa teskarisi: u restoranning o'z nomi ostida
mehmon telefonida turadi, va undagi Keel apelsini — begona rang.

- ⚠️ **Faqat aksent rollari o'zgaradi**, fon va shisha emas: brend rangi
  peshtaxta uchun tanlangan bitta tus, va u sirtlarni ham bo'yasa ba'zi
  restoranda hech kim tanlamagan yashil zal chiqadi.
- ⚠️ **`onAccent` hisoblanadi, oq deb faraz qilinmaydi.** Och sariq yoki krem
  brend (kam emas) tugmadagi oq yozuvni ko'rinmas qiladi: tugma ishlaydi, hech
  nima xato bermaydi, va «Buyurtma berish» shunchaki yo'q. Yorqinlik chegarasi
  0.55 — ko'z o'rta tuslarni arifmetikadan qorong'iroq o'qiydi, va xato bir
  tomonlama: rang ustidagi kulrang noqulay, oq esa o'qib bo'lmaydigan.
- ⚠️ **Splash va ikonka foni oq, ikkala temada ham.** Logotip qog'oz uchun
  chiziladi va deyarli har doim to'q siyoh: qorong'i splash'da u ko'rinmaydi va
  ishga tushmagan ilovadan farq qilmaydi.


### Kamomad-case: sanoq topgan narsa ish bo'lib qo'yiladi

Arifmetika allaqachon tugagan edi: sanoq har qator uchun **kutilgan**,
**sanalgan**, **farq** va **so'mdagi qiymatini** muzlatib saqlaydi
(`stocktake.go`). Faqat bularning hech biri hech kimga aytilmasdi — topilma
bitta hujjatni ochib qirq qator o'qish bilan yetib boradigan joyda edi, va
o'sha qirq qatorda muhim bo'lgan 12 kg go'sht bir gramm farq qilgan ikki qator
petrushka orasida turardi. Restoranda topilma bor edi, **navbat** yo'q edi.

`/admin/shortages` — o'sha topilma ish sifatida. Yangi raqam **yaratilmaydi**;
qo'shiladigan uch narsa qatorni case'ga aylantiradi:

1. **So'm, tartiblangan.** Bir kilo petrushka va bir kilo go'sht — birlikda
   bitta qator, ertalabda ikki xil narsa. So'm bo'yicha birinchi uch qator
   deyarli doim butun javob.
2. **Davr, nomi bilan.** Kamomad **o'sha ombornikidagi ikki sanoq orasidagi**
   vaqtga tegishli, topilgan kunga emas. Buni yozmaslik uni "kecha kechqurun"
   deb o'qitadi. `since` — o'sha ombordagi **oldingi** sanoq (bo'lmasa `null`:
   "birinchi sanoq, ya'ni butun vaqt uchun").
3. **Javob, bir marta.** Besh sabab + majburiy jumla.

⚠️ **Case saqlanmaydi — javob saqlanadi.** Navbat har o'qishda sanoqlardan
qayta quriladi; `shortage_case` da faqat odam yozgan qism yotadi. Saqlangan
navbat aynan `provisionStatus` kabi eskirardi: keyingi sanoq bekor qilgan
kamomadni "hozir ochiq" deb turgan ekranda ko'rsatib.

⚠️ **Besh sabab, va ularning ichida "o'g'irlangan" yo'q**: `miscount`, `waste`,
`card`, `paperwork`, `lost`. Kamomad ikki sanoq orasidagi **har bir smenani**
qamraydi — odam haqida bu eng zaif dalil, jarayon haqida esa eng kuchlisi.
Sabab **turi** oyiga bir marta "kamomadlarimizning yarmi kiritilmagan kirim"
degan tuzatib bo'ladigan jumlani beradi; **jumlaning o'zi** esa qaysi kirim
ekanini aytadi — shuning uchun ikkalasi ham majburiy.

⚠️ **Javob `InsertOne`, unique indeks esa ikkinchisini rad etadi**
(`(stocktakeId, ingredientId)`). Upsert bo'lsa bir xil navbatni o'qigan ikki
menejerdan **ikkinchi bosgani** jimgina g'olib chiqardi, va ekrandagi izoh hech
kim tahrirlamagan holda yangilanishlar orasida o'zgarardi. Sanoqning o'z izohi
ham xuddi shu sababdan bir martalik.

⚠️ **Qamrov past bo'lsa raqamlar aytgan narsasini anglatmaydi.** Kutilgan
qoldiq faqat **kartasi bor** taomlardan hisoblanadi, ya'ni tushumining yarmi
kartasiz ketadigan restoranda navbat ishonchli, aniq va **o'ylab topilgan**
kamomadlar bilan to'ladi — va ega tekshirgan birinchi qator unga ekranga bo'lgan
ishonchini yo'qottiradi. Shuning uchun `stockcoverage` ulushi **o'sha oyna
uchun** hisoblanadi va 70% dan past bo'lsa ekran tepasida ogohlantirish turadi.
Navbat baribir chiziladi: yashirish "hammasi joyida" degan yolg'on bo'lardi.

⚠️ **Manzili `/admin/stock/` dan tashqarida, va bu ataylab.** Omborchining
ruxsat ro'yxati (`stocklogin.go`) — metodni bilmaydigan prefikslar ro'yxati,
ya'ni `/admin/stock/…` dan berilgan navbatni **javonni o'zi sanagan odam**
yopib qo'yardi. Ko'r sanoq varaqasi aynan shundan himoya qiladi, bir ekran
narida. Narxi: `/admin/shortages` hech nimani meros olmaydi — shuning uchun
`modulegate.go` ga alohida qator kerak bo'ldi (aks holda modulni sotib olmagan
restoran uchun ochiq qolardi), va uning testi bor.

⚠️ **Filiallar bo'ylab, boshqa ombor ekranlaridan farqli.** Ular bitta filialni
talab qiladi, chunki "kompaniyada 9 kg go'sht" — sanab ham, buyurtma berib ham
bo'lmaydigan raqam. Bu yerda hech nima **qo'shilmaydi**: har qator o'z filiali
va o'z omborini aytadi, tartib esa so'm bo'yicha — ya'ni uch filialli ega
so'raydigan yagona savol ("eng yomoni qayerda?") uch ekranni qo'lda
solishtirmasdan javob oladi.

⚠️ **Ortiqcha (surplus) navbatga tushmaydi.** U ham haqiqiy topilma, lekin
boshqasi — odatda ikki marta kiritilgan kirim — va "eng yomoni tepada" deb
tartiblangan ro'yxat ikkalasini sig'dira olmaydi: birining eng yomoni
ikkinchisining eng yaxshisi. Besh sababning hech biri unga to'g'ri kelmaydi
ham.

**Bir xil so'm — bir xil hodisa emas.** Navbat pul bo'yicha tartiblanadi
(qaror puldan boshlanadi), lekin pul ayta olmaydigan narsalar qator yonida
yoziladi, va har biri ekrandagi faktlar ustidagi arifmetika — yangi da'vo emas:

- `share` — bu qator butun sanoq kamomadining necha foizi. Qirq qatorga
  tarqalgan ikki million (jarayon) va yolg'iz go'shtdagi ikki million (bitta
  voqea) — qator qiymati ikkalasida bir xil.
- `days` va `perDay` — 90 kunda bir million **tabiiy yo'qotish**, 4 kunda o'sha
  million esa **hozir ham davom etyapti**.
- `pct` — kutilganning necha foizi yo'q. 400 kilodan 12 kilo — savdo; 14 kilodan
  12 kilo — hodisa. Pul ham, miqdor ham bir xil.
- `repeat` — shu ombor sanoqlarida bu masalliq necha marta kam chiqqan. Ikki
  marta — naqsh, bir marta — kechqurun; odam qiladigan ish ham har xil.
- `used` — **o'sha davrda kartalar qancha sarfladi**. ⚠️ **Nol — bu ekranning
  eng kuchli gapi, va u kamomad emas**: hech bir karta masalliqni sarflamagan
  bo'lsa, «kutilgan» — bu shunchaki kelgan hamma narsa, va farq — kartadagi
  bo'shliq. Qamrov foizi buni ayta olmaydi, chunki u **restoran** haqidagi fakt:
  faqat kartali taomlarda ishlatiladigan masalliq 30% qamrovda ham ishonchli,
  kartasiz taomlarda ishlatilgani esa 90% da ham emas.
- `twin` — **o'sha sanoqda taxminan shuncha pulga ortiqcha chiqqan qator**. Ikki
  o'xshash paket va bitta shtrix-kod aynan shu juftlikni beradi, navbat esa faqat
  kamomadni ko'rsatadi — ya'ni tushuntiradigan yarmi ko'rinmaydi. Pul bo'yicha
  solishtiriladi (12 kg go'sht va 12 kg piyoz bir xil miqdor, lekin hech kim
  ularni kassada adashtirmaydi), farqi ¼ dan kam bo'lsa. Bu — **savol**, hukm
  emas: javobni baribir odam tanlaydi.

### Ombor qamrovi: sotuvning qancha qismi kartalar bilan qoplangan

⚠️ **"Kartasiz taomlar bor" — noto'g'ri savol, va uni so'rash bu ekranni
shovqinga aylantiradi.** Bu gap har bir restoranda ochilgan kunidan boshlab
rost, va bir yildan keyin ham rost. Doim yonib turgan ogohlantirish o'chiriladi,
o'chirish esa tanlab bo'lmaydi — muhimi ham u bilan birga ketadi (`models/alert.go`
shu darsni uzoq yozgan). `tech-cards/page.tsx` allaqachon "filtr, ogohlantirish
emas" qoidasini yozgan edi; u to'g'ri edi, faqat noto'g'ri savol haqida.

**Kamayadigan, ya'ni tugatib bo'ladigan savol:** *kelgan pulning qancha qismi
javondan izsiz chiqdi?* (`handlers/stockcoverage.go`, `GET /admin/stock/coverage`)

- ⚠️ **Tushum bo'yicha, taom soni bo'yicha emas.** 200 tadan 40 tasini yozgan
  restoranda menyu ko'rsatkichi 20%, tushum ko'rsatkichi esa 85% bo'lishi
  mumkin — u holda ombor ishlayapti va hech kimni bezovta qilish shart emas.
  *Boshqa* 40 tasi yozilgan bo'lsa menyu ko'rsatkichi o'sha, ombor esa hech
  nima. Ikkalasini faqat bitta raqam ajratadi.
- ⚠️ **Bu ABC hisobotidagi tannarx qamrovi emas, va ikkisi hech qachon
  qo'shilmaydi.** U *marja* hisoblanadimi deb so'raydi, va qo'lda yozilgan
  `cost` unga javob beradi (`costledger.go` aynan shunga tushadi). Qo'lda
  yozilgan tannarx esa javondan **hech narsa yechmaydi**: restoran tannarx
  qamrovida 100% va ombor qamrovida 0% bo'lishi mumkin.
- ⚠️ **Sotilgani, menyudagisi emas.** Hech kim buyurtma qilmagan taom to'g'ri
  ravishda hech narsa sarflamagan — `factDeadDishes` bilan bir qoida.
- ⚠️ **To'plam a'zolarining hammasi yozilgan bo'lsagina yozilgan hisoblanadi.**
  Combo'ning o'z retsepti bo'sh, ya'ni oddiy tekshiruv uni o'tkazib yuboradi va
  yuzta oilaviy set to'liq hisobga olingandek ko'rinadi (`soldDishes` bu darsni
  allaqachon olgan).
- **Ikki sabab, bitta emas**: `none` (karta yo'q) va `partial` (karta yozilgan,
  keyin o'chirilgan masalliq yoki chiqimsiz zagotovka uni buzgan). Ikkinchisi
  yomonroq — `recipeCost` yo'q qatorni **tashlab ketadi**, ya'ni taom jimgina
  **arzonlashadi**, bu esa marja ekranida yaxshi xabar bo'lib ko'rinadi.
- **Zagotovkalar alohida ro'yxatda**: chiqimi yozilmagan bitta sushi guruchi
  o'nlab rollni "partial" qiladi, va faqat taomlar ro'yxatini ko'rgan odam
  o'nta rollni yozib hech nimani tuzatmaydi.
- **Ro'yxat pul bo'yicha saflanadi, va aynan shu tartib — xususiyatning o'zi.**
  Alifbo bo'yicha bu 200 ta nom va tugamaydigan ish; pul bo'yicha birinchi
  o'nta qator javobning ko'p qismi, ya'ni ishning ko'rinadigan oxiri bor.

**Eslatish — brifingda, qo'ng'iroqda emas** (`factUncostedSales`). `alert.go`
chegarani aniq chizgan: qo'ng'iroq **bitta hodisa** uchun, bu esa **holat**.
Holat haqida jiringlaydigan telefon har kuni jiringlaydi.
- Sozlama `branch.stockCardWarnFrom` / `stockCardWarnOff`, va ⚠️ **"off" deb
  yoziladi, "on" deb emas**: nol qiymat xavfsiz tomonda turishi shart, aks holda
  xususiyat chiqqan kuni mavjud har bir restoranda o'chiq bo'ladi va buni hech
  kim bilmaydi (`mapProvider`, `AlertSettings.WithDefaults`, `provisionStatus`
  — uchtasi ham shu tuzoqqa bir marta tushgan).
- Standart chegara **80%**, 100% emas: tushumining to'rtdan uchini yozgan
  restoranning ombori o'qishga arziydi, va unga har ertalab teskarisini aytish
  brifingni varaqlanadigan narsaga aylantiradi.
- ⚠️ **O'chirish tugmasi eslatma chiqadigan joyda** — `/admin/tech-cards`
  ekranining o'zida, "Sozlamalar" ichida emas: uch ekran naridagi tugma odamni
  sozlamani emas, **ekranni** o'chirishga majbur qiladi, va o'lchov ham eslatma
  bilan birga yo'qoladi.
- Sozlama **o'z chaqiruvi bilan** saqlanadi (`PUT /admin/stock/coverage/warn`),
  filial formasi orqali emas: `AdminUpdateBranch` berilganini almashtiradi, va
  `soldOut` / `kioskSecret` shu tarzda ikki marta nolga tushgan.
- Chegara **rad etilmaydi, qisqartiriladi**: "150% dan past bo'lsa ayt" — bu
  matn xatosi, va eslatmani o'chirgani kelgan odamga qizil quti ko'rsatish uni
  formani yopishga majbur qiladi.

**Yonidagi ikki qo'riq** (bir sababdan — xato jim bo'ladi):
- ⚠️ **Manfiy qoldiq alohida ajratildi** (`balanceRow.negative`). Manfiy —
  kamomad emas, **bo'lishi mumkin bo'lmagan gap**: kirim kiritilmagan yoki karta
  oshxonadan ko'proq oladi. U turgan ekrandagi har bir raqam o'shanga suyanadi,
  shuning uchun ro'yxatning eng boshida va o'z izohi bilan. Ilgari u "kam
  qoldi"lardan **pastda**, alifbo bo'yicha, oldida minus bilan turardi.
  ⚠️ **Faqat sanalgan omborda**: birinchi inventarizatsiyagacha raqam "dunyo
  boshlanganidan beri" degani va oddiy sabablardan manfiy bo'ladi — yangi
  restoranning butun ro'yxati birinchi kuni qizarardi.
- ⚠️ **Sanalmagan ombor pul ko'rsatmaydi** (yorliqda "—"). Sanoqsiz raqam
  arifmetik jihatdan to'g'ri va hech nimani tasvirlamaydi; yorliqdagi son esa
  baholash bo'lib o'qiladi, u esa aynan shu emas.
- ⚠️ **Karta saqlanganda porsiya og'irligi so'raladi** (20 g dan kam yoki 3 kg
  dan ko'p bo'lsa). Kilogrammni gramm o'rniga yozish (`1.5` ↔ `1500`) — keyin
  hech bir ekran ko'rmaydigan yagona karta xatosi: taom deyarli tekin bo'ladi
  (marja ekranida **yaxshi xabar**) va ombordan hech narsa yechilmaydi (javon
  o'g'irliksiz ko'rinadi). ⚠️ **So'raladi, rad etilmaydi**: degustatsiya
  porsiyasi ham, banket patnisi ham haqiqiy, va bloklaydigan forma odamni
  kartani umuman yozmaslikka undaydi — bu esa ekranning o'zi tugatmoqchi bo'lgan
  holat. Donalar qo'shilmaydi: ikki tuxum va 300 g un — 302 emas.

### Spisaniya hujjati: chekka urilganda yoziladi (`stock_movement`)

Ilgari ombor sarfi **hisoblab chiqarilardi**: har ekran "davrdagi buyurtmalar ×
bugungi texkartalar" ni qaytadan yugurtirardi. Arifmetika to'g'ri edi va
**xotirasi yo'q** edi — qachon, qaysi chek, kim olib tashlagani hech qayerda
yozilmasdi.

⚠️ **Hujjatlar arifmetikaning yoniga qo'yilmadi — uni almashtirdi.** Yoniga
qo'yilsa bitta savolga ikki javob bo'lardi, birinchi tahrirda ajrardi va
qaysi biri rost ekanini aniqlashning yo'li qolmasdi (bu dars shu loyihada ikki
marta to'langan: stop list va combo yoyilishi). Endi `consumedInPeriod`
`stock_movement` ni o'qiydi, va kelishmovchilik chiqadigan **ikkinchi yo'l
qolmadi** (testi `TestTheBalanceReadsWrittenMovementsNotOrders`).

⚠️ **Shuning uchun backfill ixtiyoriy emas.** Kolleksiyasi bo'sh, orqasida bir
yillik buyurtmasi bor install sarfni **nol** deb o'qiydi: hamma javon to'la,
stop list bo'shaydi, keyingi inventarizatsiya bir yillik pishirilgan ovqat
hajmida ortiqcha chiqaradi — va bularning hech biri xatoga o'xshamaydi.
Migratsiya boot'da yuguradi va **server usiz ko'tarilmaydi** (dublikat
`pos_settings` indeksi bilan bir qoida). Marker faqat o'tish **tugagandan
keyin** yoziladi: oldin yozilsa, yarmida to'xtatilgan boot doimiy yarim
to'ldirilgan daftar qoldirardi.

⚠️ **Eski buyurtmalar bugungi kartalar bilan yoziladi** — bu aynan arifmetika
har o'qishda qilayotgan ish edi, ya'ni backfill restorandagi mavjud raqamlarni
**qaytadan chiqaradi** va manba almashishi hech bir ekranda hech nimani
o'zgartirmaydi (testi `TestTheWrittenRowsAgreeWithTheOldArithmetic`). Bundan
keyin karta sotuv paytida muzlatiladi, ya'ni **yangi** sotuvlar keyingi retsept
tahriri bilan qayta yozilmaydi.

**Qachon yoziladi: chekka urilganda.**
- Bu tizim kuzata oladigan lahza, va bu **band qilish** ma'nosini beradi —
  oxirgi porsiya oshni ikki stolga va'da qilib bo'lmaydi.
- Arifmetika ham aynan shuni sanardi (ochiq chek doim hisobda edi), shuning
  uchun manba almashdi-yu raqam qimirlamadi.

**Void — bu yerdagi butun nozik joy, va javobni kassaning o'zi allaqachon
yozgan** (`tilllines.go`, `CheckLineVoid.Wasted`):

| Holat | Kassada | Omborda |
|---|---|---|
| Yuborilmagan qator o'chirildi | Qator chekdan butunlay ketadi, sabab so'ralmaydi | Hujjat **bekor qilinadi** — javon tegilmagan |
| Yuborilgan, void, `wasted: false` | Sabab + PIN | **Bekor qilinadi** — oshxona ushlab qolgan |
| Yuborilgan, void, `wasted: true` | Sabab + PIN | **Qoladi**, "isrof" belgisi bilan |

⚠️ **Pishgan ovqatni javonga qaytarish — isrofni tuzatish deb yozish** bo'lardi:
daftar javon bilan kelishadi, daftarning o'zi noto'g'ri bo'ladi, va
inventarizatsiya hech nima topmaydi, chunki uni allaqachon ketgan deb
kutayotgan yozuvga nisbatan hech nima yetishmaydi.

⚠️ **Bekor qilingan chek ham qatorma-qator, o'sha test bilan.** Xato ochilgan
stol hech nima pishirmagan; pass'da ovqati turgan chek pishirgan. `cookedValue`
bu chiziqni PIN va loss alert uchun allaqachon chizgan, va bitta mahsulotda
"nimadir pishdimi" savoliga ikki javob bo'lmasligi kerak.

⚠️ **Birlashtirilgan chek (`mergedIntoId`) bundan mustasno.** U bekor qilinadi
va **qatorlarini o'zida saqlaydi** (ataylab — ular ovqat qayerga ketganining
yozuvi). Oddiy bekor qilish deb o'qilsa ovqat ikki marta sanalardi: qatorlarni
olgan chekda va bu yerda isrof sifatida — ya'ni ikki chek qo'shilgan **har
safar** javon butun bir stolcha kam chiqardi.

**Qurilishi** (`handlers/stocksale.go`):
- ⚠️ **O'nta ilgak emas, bitta reconciler.** Buyurtma o'nlab joydan
  o'zgartiriladi; har biriga `writeOff()` yopishtirish — qoidaning o'n ikki
  nusxasi, va o'n ikkinchisini kimdir unutadi. Unutilgani xato bermaydi:
  u jimgina noto'g'ri javon, haftalar keyin, taxtani ushlab turgan odam
  tomonidan topiladi. `syncOrderStock` buyurtmani o'qiydi, nima bo'lishi
  kerakligini hisoblaydi va **farqni** yozadi.
- ⚠️ **Idempotent** (`TestReconcilingTwiceChangesNothing`) — shuning uchun uni
  o'n joydan, sweep'dan va backfill'dan chaqirish xavfsiz. "Aynan bir marta"
  kassa, sweep va migratsiya bo'ylab hech kim ushlab turolmaydigan xossa.
- ⚠️ **Chaqirgan ishini yiqitmaydi.** Ombor qatori yozilmagani uchun pul ololmay
  qolgan kassir — bir daqiqa kech yozilgan qatordan yomonroq; reconciler
  bo'lgani uchun esa "bir daqiqa kech" hamisha shu: o'sha chekka keyingi tegish
  tuzatadi. Alert bell va print queue bilan bir qoida.
- ⚠️ **Sweep (2 daqiqa) ilgaklardan qolganini oladi**: yiqilgan yozuv, ilgagi
  yo'q yo'l, to'lov callback'i o'zgartirgan holat. Ilgaklar tezligi uchun,
  sweep to'g'riligi uchun.
- ⚠️ **Yoyilish takrorlanmadi.** Bir porsiyalik sun'iy buyurtma `consumedBy` ga
  beriladi, ya'ni combo, quyish o'lchovi va ulush qoidalari **bitta joyda**
  qoladi. Bu uchtasining har biri bu loyihaga bir oylik noto'g'ri raqamga
  tushgan.
- ⚠️ **Karta bir porsiyaga muzlatiladi**, jamiga emas: 3 → 2 tuzatish o'shanda
  bugungi kartani qayta o'qimaydi (sotuvni keyin tahrirlangan retsept bilan
  qayta narxlash bo'lardi) va muzlatilgan jamini 2/3 ga bo'lish har tahrirda
  yaxlitlash to'playdi.
- ⚠️ **Vaqt sotuvniki, yozuvniki emas.** Oflayn kassa bir haftalik chekni bir
  so'rovda yuboradi; yozuv vaqti bilan muhrlansa hammasi bir kunga tushardi va
  bo'shliqning ikki tomonidagi inventarizatsiya qarama-qarshi tomonga
  adashardi.
- ⚠️ **Kartasiz taom qator umuman yozmaydi**, nol qator emas: menyuning ko'p
  qismi qonuniy ravishda kartasiz, va minglab bo'sh qator auditni o'qib
  bo'lmaydigan qiladi. Yetishmayotgani allaqachon qamrov ekranida, nomi bilan.
- ⚠️ **Bekor qilingan qator o'chirilmaydi, "bekor" deb belgilanadi**: qimmat
  taomni urib-o'chirib turgan kassir — ham oddiy xato, ham kuzatilayotganini
  sinash usuli, va yo'qolgan qator ikkalasini ajrata olmaydi.

### Harakat hisoboti nima uchun o'z jamiga yetmasdi

⚠️ **Tsex partiyasi o'qilmasdi.** `AdminStockMovement` beshta faktni o'qirdi
(kirim, sarf, chiqim, ko'chirishning ikki tomoni), `opening`/`closing` esa
`expectedStockByWarehouse` dan kelardi — u esa partiyani **o'qiydi**. Ya'ni
markaziy oshxonasi bor har bir restoranda:

```
opening + in − used − written + movedIn − movedOut  ≠  closing
```

va farq aynan tsex tayyorlagan hajmga teng edi. Faylning o'z izohi buning aksini
va'da qilardi ("cannot disagree with the number that sent somebody here").
Alomati eng yomon turdagi: ega kamomadni **tushuntirish uchun** ochadigan yagona
ekran o'zi kamomad yasardi. Endi `produced` / `producedUsed` ustunlari va
partiya hujjatlari bor, testi `TestTheMovementReportCountsBatchesToo`.

⚠️ **Ko'chirish ustunlari brauzerda umuman chizilmagan edi** — server yuborardi,
kartochka ko'rsatmasdi. Endi to'rttasi ham **bo'lgan taqdirdagina** chiziladi:
bitta oshxonali restoran hech nima ko'chirmaydi va hech nima partiyalamaydi,
ya'ni doimiy to'rtta nol qator — to'rt qator shovqin.

⚠️ **Sotuvning hujjati yo'q edi va endi kunlik qatorlari bor.** Har chek uchun
spisaniya hujjati yozilmaydi (ataylab — ikki manba birinchi tahrirda ajraydi),
lekin "3 kg ketdi" — bu odam shu ekranni ochib **o'tib ketmoqchi** bo'lgan javob.
Endi har kun bitta qator va o'sha kuni eng ko'p olgan uchta taom nomi.
⚠️ Kunlar **mahalliy vaqtda** bo'linadi (drayver UTC qaytaradi), va sanoq
`consumedBy` dan o'tadi — combo/ulush/variant yoyilishining **ikkinchi nusxasi
yozilmadi**, chunki bu loyiha o'sha nusxa uchun bir marta to'lagan.

⚠️ **Kirim yozilmasa ham endi aytiladi** (`factNoPurchases`, 7 kun). Sanalmagan
ombor asta suriladi; kirim kiritilmayotgan ombor esa **faqat bir tomonga**
suriladi — oshxona pishirgan hajmga teng — va qatorlar manfiyga tushadi.
Farqi: bu yerda restoran ishni **qilgan**, nakladnoylar kassaning yonida
turibdi.

### Texkarta o'z ekranida (`/admin/tech-cards`)
Karta taomning formasida yozilardi, va bu **kartalarning yarmini uysiz**
qoldirgan edi.

- **Ikki xil karta bor va ikkalasi ham bitta ish**: *zagotovka* (sous, xamir,
  sushi guruchi — o'zi sotilmaydi) va *taom kartasi*. Birinchisi menyuda yo'q,
  shuning uchun uning kartasi masalliq formasining ichida, yopiq bo'limda,
  "Masalliqlar" degan ekranda yashiringan edi. ⚠️ **Restoranlar uni topmadi** va
  guruchni qirq taomga qo'lda ko'chirdi — aynan `ingredient.recipe` oldini
  olish uchun qo'shilgan yetti nusxa. Model to'g'ri edi, kirish yo'li yo'q edi.
- **Nima uchun taom formasidan olib tashlandi** (iiko yo'li): taom formasi
  narx, rasm, ta'rif uchun haftada ochiladi; karta esa bir marta, bosilgan
  varaqdan, boshqa odam tomonidan yoziladi — va u pishiriladigan narsa
  peshtaxtada sotiladimi yoki chelakda qoladimi, ish bir xil. Menyuda **faqat
  ko'rsatiladi** (qaysi raqam bilan narxlanayotgani) va havola qoladi.
- ⚠️ **Eng xavfli qismi shu bo'ldi: `menuItemIO.Recipe` endi ko'rsatkich.**
  Taom formasi endi `recipe` ni **umuman yubormaydi**, `UpdateMenuItem` esa
  butun hujjatni almashtiradi. Oddiy slice bo'lib qolsa, taomning **narxini**
  o'zgartirish uning **kartasini o'chirardi** — xatosiz, ogohlantirishsiz, va
  natija "hali narxlanmagan taom" bo'lib ko'rinardi, ya'ni menyuning ko'p
  qismi qonuniy ravishda turadigan holat. `keepRecipe` — `keepCost` bilan bir
  qoida, faqat oqibati og'irroq. **Bo'sh massiv esa kartani tozalaydi**:
  "yuborilmadi" va "tozalandi" boshqa javoblar.
- ⚠️ **Masalliqlar formasi kartani hamon qaytarib yuboradi** (`recipe`,
  `output`, `batched`) — u ham `ReplaceOne`. Zagotovkaning **nomini** shu
  yerda tuzatgan odam kartasini yo'qotardi.
- **`PUT /admin/menu/{id}/recipe`** — bitta maydonning `$set`'i, taomning
  o'zi emas: karta ekrani daqiqalar oldin o'qilgan taom nusxasini ushlab
  turadi, uni butunlay qaytarib yuborish menyu ekranida o'zgartirilgan narxni
  jimgina qaytarardi. `recipeDiff` + jurnal + `alertOnRecipeIncrease` —
  `UpdateMenuItem` dagi bilan bir xil, chunki **karta tahriri o'g'rilikni
  arifmetik ko'rinmas qiladigan yagona yo'l**. Brend filtri `_id` yonida, va
  qamrovdan tashqarisi 404.
- **To'plamning kartasi yo'q** (400): tannarxi a'zolaridan chiqadi, karta esa
  ularning **yoniga** qo'shilib bir pulni ikki marta sanardi.
- ⚠️ **`RecipeEditor` yarim tayyorni nolga hisoblardi** — endi serverning
  `rate` si ishlatiladi. Zagotovkaning `price` i 0 (uning narxi partiyasidan
  chiqadi), va editor `price/1000` qilardi: sushi bari kabi kartalari deyarli
  butunlay zagotovkadan iborat restoranda tannarx **yozilayotganda nol**,
  saqlangandan keyin esa haqiqiy raqam bo'lardi. Bitta ekranda ikki javob —
  aynan kartalar tugatish uchun mavjud bo'lgan ajralish.
- ⚠️ Yangi ekran **hech nimani qayta hisoblamaydi**: taomlar `/admin/menu` dan,
  zagotovkalar `/admin/ingredients` dan **allaqachon narxlangan** holda keladi.
  Umumiy `/tech-cards` ro'yxati tannarxning ikkinchi implementatsiyasi bo'lardi.

### Landing tuzilishi: nima bosh sahifada qoladi (`keel-site`)
Ega aytgan gap: «dizayn yaxshi, lekin saytga kirgan odam **ma'lumot
ko'pligidan chalkashib qolarkan**». Tashxis to'g'ri edi, va u dizayn muammosi
emas — **tartib** muammosi. Sahifada **13 ta bo'lim** bor edi.

Ikkita nosozlik, ikkalasi ham uslub bilan bog'liq emas:

- ⚠️ **Kassa bo'limi ikkinchi o'rinda va sahifadagi eng katta blok edi** —
  `page.tsx` ning 300 qatori. Odam bir ekran o'qib, **nega kerakligini
  bilmasdan** turib mahsulotning eng chuqur tafsilotiga tushardi. U endi
  `/kassa` da, raqobatchilar solishtiruvi va integratsiyalar ro'yxati bilan
  birga — uchalasi ham «qaror qilgandan **keyin**» o'qiladigan narsalar.
  **Kesilmadi**: bosh sahifa dalilni ushlaydi, u sahifa isbotni.
- ⚠️ **Komissiya jadvali sakkizinchi, ya'ni narxdan keyin edi.** U — narx
  nega arziydiganini aytadigan yagona blok, va undan **keyin** kelgan narx
  asossiz ko'rinadi. Endi hero'dan darrov keyin.

Yangi tartib argument oqimi bo'yicha: hero → nega arzon → kimlar uchun →
mahsulot → narx va kalkulyator → kassa havolasi → ishga tushish → savollar →
CTA. **8 ta bo'lim.**

- ⚠️ **Ko'chgan bo'limga langar qoldirilmaydi.** `/#till` — endi o'sha bo'lim
  yo'q sahifaning tepasiga olib boradi, va bu «ko'chirilgan» emas, **«buzilgan»**
  bo'lib o'qiladi. Header, footer va hero'dagi havolalar sahifaga aylantirildi.
- **Navigatsiyadan «Integratsiyalar» olib tashlandi.** Ma'lumot ko'pligidan
  shikoyat qilingan sahifada yettita nav elementi — bir daraja yuqoridagi
  o'sha xato.
- `/kassa` sitemapda **0.9** bilan turadi, landingdan keyin ikkinchi: bu
  bozorda odamlar aynan «keel kassa» deb qidiradi, va bu iboraga javob
  beradigan sahifa o'sha ibora bo'yicha chiqishi kerak.
- Bo'lim qobig'i, hull motifi va footer `components/landing/Shell.tsx` ga
  chiqdi — endi landing bitta emas, va ularni bosh sahifa **egallab turishi**
  noto'g'ri bo'lardi.

### Landing suratlari ham uch tilda (`keel-site/public/shots`)
Bosh sahifadagi o'nta surat — **sahifaning dalili, bezagi emas**: butun da'vo
«bu haqiqiy kassa, haqiqiy zal, haqiqiy oshxona ekrani» degani. Ular faqat
o'zbekcha edi, ya'ni tashrifchilarning uchdan ikkisiga bu da'vo **ular
o'qiy olmaydigan tilda** aytilardi. `scripts/landing-shots.mjs` hammasini
uch tilda oladi (10 × 3).

- **Holatlar haqiqiy va `demo` bazasida turadi**: 8-stol ochiq, chunki uni
  kimdir ochgan; oshxona ekranida 14 ta chek bor, chunki 14 ta buyurtma bor.
  Bu yerda hech nima yasalmagan — yasalgan ekran bu sahifadagi **ushlanishi
  mumkin bo'lgan yagona narsa** bo'lardi.
- **Skript hisoblarni o'zi tayyorlaydi** (admin API orqali parol va PIN):
  Mongo'ga to'g'ridan-to'g'ri hash yozish parol yo'lining o'z qoidalarini
  (uzunlik, bcrypt, jurnal) chetlab o'tardi va o'sha yo'l buzilgandan keyin
  ham «ishlayotgandek» qolardi.
- ⚠️ **Kassa suratida taom rasmlari qurilma sozlamasi** (`keel_till_images`,
  standart holatda **o'chiq**) — yuz taomli peshtaxtada nomlar ro'yxati
  tezroq o'qiladi, ya'ni standart to'g'ri. Lekin marketing kadri kulrang
  harflar to'riga aylanadi. Skript uni saytning o'zi saqlaydigan kalit orqali
  yoqadi.
- ⚠️ **Cookie bildirishnomasi CSS bilan yashirilmaydi, «roziman» bilan**:
  yashirish sahifani u turgandek joylashtirib qoldiradi va kadrda hech kim
  tushuntira olmaydigan bo'shliq chiqadi. Skript saytning o'z bayrog'ini
  qo'yadi — natijada qaytib kelgan mehmon ko'radigan ekran.
- ⚠️ **Bosiladigan elementlar `data-help` bilan belgilangan** (`To'lash`
  tugmasi). Matn bo'yicha bosish o'zbekchada ishlaydi, ruschada esa **xato
  bermay** boshqa ekrandan kadr oladi va to'g'ri nom bilan saqlaydi — birinchi
  yugurishda aynan shu bo'ldi: «menyu to'ri» nomi ostida to'lov oynasining
  surati.
- ⚠️ **`shot()` da zaxira yo'q** (`lib/shots.ts`): yetishmagan kadr brauzerda
  **404** bo'lib ko'rinishi kerak. O'zbekchasiga qaytish muammoni tiklaydi va
  ayni paytda uni yashiradi. `check-help.mjs` sahifaning **o'zidan** o'qib
  o'nta nomni tekshiradi (ro'yxat yozilsa, u birinchi yangi bo'limdayoq
  eskirardi — jimgina).

### Bilim bazasi (keel.uz/help)
Uch tilda, 88 maqola, 44 ta ekran surati. Paneldagi qisqa yordam
(`frontend/src/lib/help/articles.ts`) o'z o'rnida qoldi — bu **ikkinchi uy**, va
sabablari boshqa: bittasi odam turgan ekranga javob beradi, ikkinchisi qidiruv
orqali topiladi va **hisobi yo'q odamga** ham ochiladi (qo'ng'iroqlarning katta
qismi aynan shunday).

- **Server tomonda chizilgan, statik**: JavaScript ishlagandan keyingina paydo
  bo'ladigan bilim bazasini hech bir qidiruv tizimi o'qimaydi, va «техкарта как
  составить» deb qidirayotgan restoran egasi — bu sahifa yozilishga arziydigan
  yagona o'quvchi. Har maqola build paytida chiziladi, sitemapda 88 ta manzil.
- ⚠️ **Har kadr uch marta olinadi — har til uchun.** O'zbekcha panelning
  suratiga qarab turgan rus o'quvchiga **o'ziniki bo'lmagan ekran** ko'rsatiladi:
  rasmdagi so'zlar — u o'z ekranidan qidirishi kerak bo'lgan so'zlar, va o'qiy
  olmasa surat bezakka aylanadi. Panel tilini `lang` **cookie**'si beradi (uning
  til manzili yo'q — bu ataylab), shuning uchun har til uchun alohida brauzer
  konteksti. Ommaviy sayt esa manzil prefiksi bilan (`/ru/menu`).
- ⚠️ **Koordinatalar ham har tilda alohida o'lchanadi**: «Zagotovkalar · 3» va
  «Заготовки · 3» — turli kenglik, ya'ni yonidagi tugma boshqa joyda turadi.
  Bitta umumiy koordinata har bir ruscha strelkani **biroz** noto'g'ri
  qilardi — va «biroz noto'g'ri» hech kim xabar bermaydigan variant, chunki u
  hali ham strelkaga o'xshaydi. (Amalda: `tabs` kengligi 0.094 / 0.083 / 0.060.)
- ⚠️ **Rasm uchta, izohlar hamon ma'lumotda.** Chiziq va raqamlar PNG'ga
  **chizilmaydi**: `figures.json` da foizli koordinata, matn esa maqolada.
  Aks holda har ekran uchun **uchta izohli** kadr bo'lardi va tarjimon so'zlarga
  umuman yeta olmasdi.
- ⚠️ **Koordinata DOM'dan o'lchanadi** (`scripts/help-screens.mjs` → `measure`),
  qo'lda qo'yilmaydi: «chapdan taxminan 12%» deb yozilgan to'rtburchak keyingi
  relizda tugma o'n piksel siljiganda **xato bermaydi** — u shunchaki boshqa
  narsani ko'rsatadi, uch tilda, kimdir yozmaguncha.
- ⚠️ **Nishonlar `data-help` atributi bilan belgilanadi**, ko'rinadigan matn
  bilan emas. `button:has-text("Yangi zagotovka")` o'zbekchada mукаммал
  ishlaydi va ruschada **hech nimaga mos kelmaydi** — va bu yerdagi nomuvofiqlik
  xato bermaydi, u shunchaki bitta izohni **tushirib qoldiradi**, ya'ni relizga
  chiqib ketadi. Panelda sakkizta atribut: bu test yoki uslub ilgagi emas, bu
  «shu element hujjatlashtirilgan» degan ochiq shartnoma.
- **Rasm topilmasa o'zbekchasiga qaytadi**, bo'sh joyga emas: bir tilda olinib
  ikkinchisida hali olinmagan kadr — yarim kunlik normal holat, bo'sh joy esa
  yo'q. Ammo buni `check-help.mjs` **xato deb sanaydi** — aynan shu qaytish
  «ruscha o'quvchi o'zbekcha panelni ko'rmoqda» holatini jimgina qaytarib
  keltiradi.
- ⚠️ **Screenshotlar `demo` bazasidan** (B5 Somsa), jonli mijozdan emas: jonli
  restoranning tushumi, telefonlari va xodim ismlari bilan bezatilgan yordam
  sahifasi — o'z mijozlarini sizdirayotgan sahifa. Kadr olishdan oldin panel
  **tinchlantiriladi** (`QUIET`): dev belgisi, «2 ta buyurtma qabul
  qilinmagan» qo'ng'irog'i va yarim yuklangan grafik — o'quvchi o'z ekranidan
  qidiradigan va topmaydigan narsalar.
- **PNG → WebP, 1440px** (`sharp`): 44 kadr ikki barobar piksel bilan 15 MB
  edi. O'n besh megabaytlik yordam sahifasini podvaldagi telefonda hech kim
  ochmaydi — va aynan o'sha odam uchun yozilgan. 2 MB bo'ldi.
- **Bloklar, HTML satri emas** (`types.ts`): tarjimon teg yopishini kuzatmaydi,
  va telefonda qadam raqamli ro'yxat, ogohlantirish esa rangli bo'ladi.
  Ichida faqat ikki belgi: `*qalin*` va `` `Tugma nomi` `` — ikkinchisi butun
  sabab, chunki u **o'quvchi ekranida qidiradigan yorliqni** keltiradi.
- ⚠️ **Lug'atda funksiya bo'lmaydi**: `searchEmpty` avval `(q) => …` edi va
  sahifa 500 qaytardi — funksiyani klient komponentiga uzatib bo'lmaydi.
  `{q}` / `{n}` o'rinbosarli **qator**. Yon foydasi: butun lug'at sof ma'lumot,
  ya'ni tarjimon kodga tegmaydi.
- ⚠️ **`scripts/` ning o'z `package.json` i bor.** Playwright va sharp'ni
  `frontend/` yoki `keel-site/` ga qo'shish ularning Dockerfile'idagi
  `npm ci` ga brauzer yuklab olishni qo'shadi — prod image'ga.
- **Izchillikni skript ushlaydi** (`scripts/check-help.mjs`): bu yerdagi har
  xato **sahifada ko'rinmaydi**. Rus tilida yo'q slug — `hreflang` uchta manzil
  borligini aytgandan keyin 404; nomi o'zgargan `see:` — «shuni ham o'qing»
  qutisida bitta qator kam; olinmagan `fig:` — faqat bitta tilda buzilgan rasm.
  Tekshiruv: bir xil slug va tartib, bir xil bo'lim/rasm/havola, har havola
  mavjud maqolaga, har rasm mavjud faylga, har belgi o'lchangan koordinataga.
- **Sarlavha yo'q, ro'yxat bor**: navigatsiyada «Yordam» — ro'yxatdagi yagona
  **haqiqiy sahifa** (qolgani langar), shuning uchun u `localePath` orqali
  o'tadi. Busiz rus tashrifchi o'zbekcha maqolalar to'plamiga tushardi.

### Markaziy sklad va nakladnoy: filialga jo'natish

Tsex allaqachon bor edi (partiya, `warehouse.kind="production"`), **jo'natish**
yo'q edi. Mavjud `stock_transfer` bitta filialning ikki javoni orasida ishlaydi
(«barmen podvaldan tonik oldi»), zanjirning har kungi harakati esa boshqa:
markaziy sklad go'shtni oladi, dushanba kuni marinovka qiladi va **to'rt
filialga** jo'natadi. Buni soxtalashtirishning ikkala yo'li ham yolg'on:
chiqim — hech kim isrof qilmagan mahsulotni isrof hisobotiga qo'yadi, narigi
uchidagi kirim esa narx tarixiga xarid narxini yozadi va o'sha masalliq
kiradigan har bir taomni qimmatlashtiradi.

`dispatch` hujjati — **ikki javon va orasidagi mashina**:

- ⚠️ **Hujjatda ombor yo'q**, va bu kamchilik emas: nima qayerda turishi
  filial + masalliq haqidagi fakt (`ingredient_placement`), shuning uchun
  jo'natma **kirim kabi** har ikki uchida o'zini o'zi yo'naltiradi. Omborchidan
  har qator uchun ikki xonani so'rash — shoshilinch paytda noto'g'ri javob
  beriladigan savol, va noto'g'ri javob sanashgacha ko'rinmaydi.
- ⚠️ **Jo'natilgan va kelgan — ikki fakt.** Jo'natuvchining javonidan
  **yuklangani** chiqadi; qabul qiluvchining javoniga **sanab olingani**
  tushadi; farqi esa **hech kimning javonida emas** — u yo'l haqidagi topilma.
  Qog'ozdagi uchta imzo (sklad boshlig'i, haydovchi, filial brigadiri) aynan
  shuning uchun bor. Ikkovini jimgina tenglashtirish — o'sha uchta imzo
  yaratadigan yagona raqamni o'chirish.
- ⚠️ **Imzolanmaguncha hech nima kelmaydi.** Aks holda filial pishira olmaydigan
  qoldiqni ko'radi va **stop list** o'sha raqamlarni o'qigani uchun sotuvga
  ruxsat beradi.
- ⚠️ Jo'natuvchi — **ko'rinishdagi filial** (`stockBranch`), tanadagi id emas:
  aks holda bir filial boshqasining javonini id yozib bo'shatardi, va mahsulot
  haqiqiy joyga borgani uchun hech bir ekran xato ko'rsatmasdi.
- ⚠️ Faqat **qabul qiluvchi filial** imzolaydi, va filial **filtr ichida**
  (`toBranchId`), tekshiruvda emas. Imzolash bir marta: `acceptedAt` ning
  yo'qligi ham filtrning bir qismi.
- ⚠️ Ikki uchi ham **kutilgan qoldiqqa** qo'shildi (`expectedStockByWarehouse`
  ning yettinchi fakti) va harakat hisobotiga ustun bo'lib chiqdi. Busiz
  markaziy sklad **jo'natgan hamma narsasi bo'yicha kamomad** ko'rsatardi — va
  keyingi sanoqda uni sanagan odamning aybi bo'lib chiqardi.
- Qiymat **tashiladi, yaratilmaydi** (ko'chirish bilan bir qoida): moliyaviy
  hisobotning xarajatiga kirmaydi.

⚠️ **Bir ertalab — bitta amal, beshta emas.** Markaziy sklad bitta filialga
jo'natib, keyin keyingisini o'ylamaydi: u javon oldida beshta varaq bilan turadi
va qatorni **hammasi bo'ylab** yozadi — blanka aynan shuning uchun bitta varaqda
to'rtta yo'rtma. Shuning uchun ekran ham **jadval**: qator — mahsulot, ustun —
filial, va bitta `POST` hammasini yozadi (`InsertMany`). Beshta alohida saqlash
bo'lsa, yarmida yiqilgan so'rov ikki filialni yuklangan, uchtasini yuklanmagan
qoldiradi — haydovchi esa qo'lida beshalasining qog'ozi bilan turadi.

- Raqamlash **bir marta** hisoblanadi va sanaladi: har yo'rtma alohida so'rasa,
  bir soniyada yozilgan beshtasi bir xil raqam olardi — va raqam aynan ikki
  varaqni bir-biridan ajratish uchun bor.
- **Bo'sh ustun — o'tkazib yuboriladi, rad etilmaydi**: bugun hech nima
  buyurtmagan filial. Bitta bo'sh ustun uchun butun saqlashni rad etish
  omborchini "qaysi ustun ekan?" deb qidirishga majbur qiladi.
- **Bitta filialga ikki ustun — rad etiladi**: bu ustunni ikki marta yozib
  yuborish, va ikkinchisi javondan yana oladi, qog'ozda esa haqiqiy ikkinchi
  reys bo'lib ko'rinadi.
- Jadvalda har qatorning **jamisi javondagi qoldiqqa nisbatan** ko'rsatiladi va
  oshib ketsa qizaradi. ⚠️ **Ogohlantiradi, to'xtatmaydi**: qoldiq — taxmin, va
  eskirgan raqam tufayli haqiqiy mashinani to'xtatgan ekran daftar bilan
  aylanib o'tiladi.

**Nakladnoy** (`frontend/src/lib/nakladnoy.ts`) — bu avval qog'oz, keyin ekran:

- **Bitta A4 (albom) da to'rtta yo'rtma**, har filialga bittadan — restoranlar
  ishlatadigan blankaning aynan o'zi. Beshinchi filial yangi varaqdan boshlanadi;
  yo'rtma ikki betga bo'linmaydi — bo'lingan yo'rtmani imzolab bo'lmaydi.
- ⚠️ **Bo'sh qatorlar — bezak emas.** Tizim biladigan qatorlar bilangina
  chegaralangan blanka — mashina yuklanayotganda filial telefon qilib so'ragan
  ikki yashikni yozib bo'lmaydigan blanka, va o'shanda yozuv chetga tushadi yoki
  umuman yozilmaydi. Bo'sh qatorlar ham **raqamlangan**.
- ⚠️ **Tili chop etish paytida tanlanadi, paneldan olinmaydi**: omborchi bir
  tilda ishlaydi, narigi uchida imzolaydigan brigadir boshqa tilda o'qishi
  mumkin — va bu har jo'natmada o'zgaradi. Shuning uchun modul lug'atni
  **argument** sifatida oladi (hook emas), sahifada esa uch tilli tugma turadi.
- ⚠️ **Sarlavha ikki qatorda va qat'iy.** Bir qatorda eng uzun filial nomi eng
  uzun tilda ("Филиал Beshqayrag'och — дата: ...") 70 mm yo'rtmadan chiqib
  ketadi va **yilni** olib ketadi — butun hujjat aynan sana bo'yicha
  saqlanadi. Qat'iy ikki qator to'rtta yo'rtmani bir sathda ushlaydi.
- ⚠️ **Nomlar matn sifatida yoziladi** (DOM tugunlari, HTML satri emas):
  qatorlarda restoranning o'z nomlari turadi — hech kim tozalamaydigan matn.
  Chek printeri bilan bir qoida.
- Popup emas, **ekrandan tashqaridagi iframe**: popup omborchining brauzerida
  bloklanadi, va bu nosozlik printerda qog'oz tugaganday ko'rinadi.

### Markaziy oshxona (tsex): partiya va ishlab chiqarish hujjati
`pos-reja.md` §7 kechiktirgan uchtadan qolgan ikkitasi (ko'chirish allaqachon
bor). ⚠️ **Bu qo'shimcha maydon emas — raqamning ma'nosini o'zgartiradi.**

- **Odatdagi yarim tayyor umuman zaxira emas**: hech kim "sous"ni sanamaydi,
  pomidorni sanaydi, va sous ishlatgan taom **pomidor** ishlatgan deb o'qiladi
  (`rawInputs`). Bu bitta oshxona uchun to'g'ri, va markaziy oshxona paydo
  bo'lishi bilan **noto'g'ri**: tsex dushanba kuni 40 kg qiladi va uch filialga
  yuboradi — u yerda sous **muzlatgichdagi idish**, pomidor esa hech qachon
  bo'lmagan.
- **Ikki yarim, va bittasi yolg'iz ishlamaydi**: `ingredient.batched` ("javonda
  turadi, o'zi sarflanadi") va `production` hujjati (uni javonga qo'yadi,
  masalliqlarini javondan oladi).
  - ⚠️ Bayroq bor, hujjat yo'q → javon hech qachon to'lmaydi, haqiqiy narsa
    bo'yicha **manfiy** qoldiq.
  - ⚠️ Hujjat bor, bayroq yo'q → masalliqlar **ikki marta** ayriladi (bu yerda
    va taom sotilganda kartadan), va yetishmovchilik haftalar keyin sanashda,
    sanagan odamning aybi bo'lib chiqadi.
- ⚠️ **`MadeInHouse()` endi `DerivedOnly()` ga bo'lindi.** Ilgari bu bitta savol
  edi, chunki ikkalasi bir xil narsa edi. Partiya paydo bo'lgach ular ajradi:
  partiyali yarim tayyor **sanaladi, ko'chiriladi, ogohlantiradi**. Yarim
  tayyorni tashlab ketadigan har bir joy endi shu yangi savolni so'rashi kerak
  — aks holda tsex chiqargan mahsulot uni **qabul qilgan filial uchun
  ko'rinmas** bo'lib qoladi.
- **Ombor turi** (`warehouse.kind = "production"`) — bezak emas: partiya faqat
  shu omborda tayyorlanadi, chunki masalliqlar **aynan shu javondan** chiqadi.
  Har joyda ruxsat berish filialga boshqa filial oshxonasidagi pomidordan sous
  "qilish" imkonini berardi — arifmetika buni qabul qiladi, hech bir ekran
  so'ramaydi.
- ⚠️ **Nima sarflangani hujjatga muzlatiladi**: karta o'zgaradi, martdagi
  partiya esa martda nimani olgan bo'lsa shuni olgan. Qayta hisoblash allaqachon
  sanalgan va solishtirilgan oyni qayta yozardi (chiqim qiymati bilan bir qoida).
- ⚠️ **Qiymat tashiladi, yaratilmaydi** — ko'chirish bilan aynan bir qoida:
  hech nima sotib olinmadi va yo'qolmadi, pomidor sousga aylandi. Jami
  **"tayyorlandi"** deb ataladi va moliyaviy hisobotning xarajatlariga kirmaydi.
- ⚠️ **Bo'sh qiymat — eski xatti-harakat**: `batched` yoqilmagan, `kind` bo'sh.
  Bitta oshxonali restoran hech nimani sezmaydi, migratsiya yo'q.
- Ekran ishlab chiqarish nimani olganini **saqlashdan oldin ko'rsatmaydi**:
  hisobni server kartadan qiladi, brauzerdagi ko'rinish esa tannarxning
  ikkinchi implementatsiyasi bo'lardi — va ega qaraydigani aynan ajrab
  ketgani bo'lardi.

### Hisobotlar va Excel eksporti
- **Bitta shakl, uch chiqish** (`handlers/report.go`): `Report{Title, From, To,
  Columns, Rows, Totals, Note}` → ekran (JSON), Excel (.xlsx), keyinchalik 1C.
  Yangi hisobot — bu so'rov va ustunlar ro'yxati, va u ekranga hamda Excel'ga
  **bir vaqtda** chiqadi.
- ⚠️ **Eksport ekrandagi raqamlarning ikkinchi hisobi emas, o'shaning o'zi.**
  Bir oyni eksport qilib buxgalterga yuborgan ega, keyin "nega jadval panel
  bilan mos emas" degan savolga duch kelsa — unga ikkita javob berilgan va
  qaysi biri noto'g'ri ekanini bilishning yo'li yo'q.
- ⚠️ **CSV emas, haqiqiy .xlsx.** Sabab estetik emas: ma'lumot o'zbekcha matn
  va so'm summalari, CSV ikkalasini ham buzadi. Ruscha/o'zbekcha Windows'dagi
  Excel vergulni o'nlik ajratgich deb o'qiydi — "Lag'mon, katta" ikki katakka
  bo'linadi va 92,000 → 92; BOM'siz kirill va apostroflar krakozyabra bo'ladi.
  Har biri moliyaviy hujjatning jimgina buzilishi.
- ⚠️ **Raqamlar raqam bo'lib yoziladi**, oldindan formatlangan matn emas.
  "92 000" satrlari ekranda bir xil ko'rinadi, lekin ularni yig'ib, saralab
  yoki diagramma qilib bo'lmaydi — bu esa skrinshot o'rniga jadval so'rashning
  asosiy sababi.
- Varaq nomi 31 belgigacha qisqartiriladi va `: \ / ? * [ ]` olib tashlanadi:
  Excel bunday nomni rad etganda **butun faylni** ochmaydi, nomni emas.
- Panelda yuklab olish `downloadReport()` orqali — `<a href>` emas: panel
  bearer token bilan ishlaydi va havola sarlavha tashimaydi, ya'ni brauzer
  401 ga o'tardi va bu operatorga "hech nima bo'lmadi" bo'lib ko'rinardi.

### ABC/XYZ menyu tahlili (`/admin/reports`)
- **ABC** — taomning tushumdagi ulushi (Pareto 80/15/5). **XYZ** — talabning
  barqarorligi (kunlik sotuvning variatsiya koeffitsienti). Qaror ikkisining
  **kesishmasida**: `AX` — hech qachon tugamasligi kerak, `AZ` — pul keltiradi
  lekin to'lqin bilan, `CZ` — menyudan chiqarish nomzodi.
- ⚠️ **Sotilgani hisoblanadi, pul olingani emas** — dashboard'dagi tushum
  boshqa asosda. Tasdiqlangan buyurtmadagi taom sotilgan, kuryer hali pul
  bilan qaytmagan bo'lsa ham. Faqat bekor qilish "sotilmagan" qiladi.
- ⚠️ **80% chizig'ini kesib o'tgan taom A'da qoladi**, B'da emas: kesim
  qo'shishdan **oldingi** jamlanma bo'yicha qilinadi. Aks holda qisqa menyuda
  tushumning 40% ini ko'tarib turgan taom "ikkinchi darajali" deb ko'rsatiladi.
- ⚠️ **Sotuvsiz kun — nol, tushib qolgan kuzatuv emas.** Faqat sotilgan
  kunlar o'rtachalansa, oyiga bir marta yigirma porsiya ketadigan taom eng
  barqaror bo'lib chiqadi — menyudagi eng tartibsiz narsa eng bashoratli deb
  ko'rsatiladi.
- **XYZ chegaralari 25% / 60%**, darslikdagi 10%/25% emas: ular ishlab
  chiqarishdan, u yerda talab shartnomalar bilan silliqlangan. Restoranning
  kunlik porsiya soni kichik butun son, kichik butun son esa shovqinli — kuniga
  uch porsiyada bitta tinch seshanba 30% tebranish. 10% bilan butun menyu Z'ga
  tushardi va hech nima aytmasdi.
- Tebranish yonida **necha kun sotilgani** ko'rsatiladi: o'ttiz kundan ikkitasida
  sotilgan taomning koeffitsienti arifmetik jihatdan to'g'ri va hech nima
  anglatmaydi, va ikkinchi raqamsiz ularni ajratib bo'lmaydi.

### Bo'sh chekni bekor qilish sabab so'ramaydi
⚠️ **Hech nima pishirilmagan chek — yo'qotish emas.** Xato bosilgan tugma, ketib
qolgan mehmon, ikki marta ochilgan stol: bu restorandagi eng ko'p uchraydigan
bekor qilish. Undan sabab **va** menejer PIN i talab qilish hech nimani
himoya qilmaydi — u ekrandagi eng tez-tez takrorlanadigan amalga «bu quti —
rasmiyatchilik» degan odatni o'rgatadi, va quti aynan kerak bo'lgan kuni
rasmiyatchilik bo'lib qoladi (bu `tilloverride.go` yozilgan mantiqning o'zi:
rad etilgan ofitsiant menejerni chaqirmaydi, uning kodini yodlaydi).

- Chegara — **`cookedValue(o)`**: oshxonaga yuborilgan (`firedAt`) va bekor
  qilinmagan qatorlarning summasi. ⚠️ Bu **loss alert** ishlatadigan ta'rifning
  o'zi, va ataylab bitta funksiya: ikki nusxa oxir-oqibat ajraydi, va o'shanda
  bitta bekor qilish menejer PIN iga arziydi-yu ogohlantirishga arzimaydi
  (yoki teskarisi).
- Bo'sh chek jurnalda **«bo'sh chek»** bo'lib yoziladi — bo'sh katak javob
  emas, yo'qolgan yozuvga o'xshaydi.
- Ekran ham shu testni bajaradi (`reasonOptional`), ya'ni kassa server
  kechiradigan narsani so'ramaydi.

### Kassadagi «Onlayn»: ikki ro'yxat, qidiruv va kartochka
- ⚠️ **Ochiq va yakunlangan — ikki ro'yxat, saralangan bitta ro'yxat emas.**
  Ular qarama-qarshi sabab bilan o'qiladi: birinchisi **ish** (kimdir kutyapti
  yoki pul hali kuryerda), ikkinchisi esa mijoz qayta qo'ng'iroq qilganda
  qaraladigan **yozuv**. Aralashtirilsa, kechqurungi qirqta yetkazilgan
  buyurtma kerakli uchtasini ko'mib yuboradi — va butun ekran o'qilmay qoladi.
- ⚠️ **«Yetkazilgan, lekin to'lanmagan» — hali ochiq**, va shuning uchun bu
  `status !== "delivered"` emas, funksiya (`isLive`): kuryer qaytdi, ovqat
  yetdi, pul yetmadi — bu ekran aynan shu qator uchun qurilgan.
- **Qidiruv brauzerda**: bugungi buyurtmalar (ko'pi bilan 200 ta) allaqachon
  ekranda, ya'ni harf terilgan tezlikda filtrlaydi. Mijoz telefon raqamini
  aytib turganda kassir har harf uchun serverga borishni kuta olmaydi.
- **Kartochka alohida so'rov bilan** (`GET /staff/online/{id}`): ro'yxat yarim
  daqiqada bir marta so'raladi va bir qarashda o'qiladi — ikki yuz buyurtmaning
  har bir taomini o'sha so'rovda tashish hech kim qaramaydigan menyucha JSON
  bo'lardi. Tafsilot **ochilganda** olinadi, ya'ni odam to'xtab turgan paytda.
- **Qator butunlay bosiladi**: quloqqa telefon tutgan kassir kichkina belgiga
  emas, qatorga tegadi. Pul qabul qilish tugmasi ham shu kartochkada.

### AI bloklari: sotildi, ko'rsatildi, hisoblanmadi
⚠️ **Bir ekranda sotilgan narsa boshqa ekranda hisoblanadi — va ular
kelishmadi.** Konsolning kassa paneli qo'shimcha AI bloklarini umumiy oylik
narxga qo'shib ko'rsatardi (operator mijoz bilan **1 800 000** ga kelishadi),
`billing.TillMonthly` esa ularni umuman bilmasdi (**1 500 000**) — va aynan shu
funksiya hisob-fakturani yozadi hamda restoranning o'z paneliga
(`mirrorTill` → owner ilovasi, Sozlamalar → Obuna) ko'chiriladi. Ya'ni biz bir
narxni aytib, boshqasini hisoblab, egaga uchta ekranda ikkinchisini
ko'rsatardik. `AIExtraMonthlyFor` yozilgan edi va **hech qayerdan
chaqirilmasdi**.

- Endi `aiExtra` — `TillMonthly` ning **parametri**. Imzo o'zgardi, ya'ni
  kompilyator har bir chaqiruv joyidan javob talab qildi; qo'shimcha qator
  sifatida keyin qo'shish mumkin bo'lganda, aynan shu unutilardi.
- ⚠️ **Tarif blokni o'z ichiga olsa ham hisoblanadi** (modullardan farqi): blok
  — tarif beradigan kunlik limitning **ustiga** qo'shiladigan miqdor, ya'ni
  yuqori tarif uni «allaqachon to'lagan» bo'la olmaydi.
- ⚠️ **Faqat saqlash bosilganda yangilanadigan ko'zgu — eskiradigan ko'zgu, va u
  aynan shunday eskirdi.** Narx kodda o'zgardi, mavjud har bir mijozga esa oylar
  oldin ko'chirilgan raqam ko'rinib turaverdi (panelda ham, kassada ham, ega
  telefonida ham), hisob-faktura esa uchinchisidan yozilardi. Tuzatishning
  yagona yo'li — operator o'sha tenantni ochib, hech nima o'zgartirmay
  «Saqlash» bosishi edi; buni ellik marta hech kim qilmaydi.
  Endi `SyncTillGrants` **soatlik ishda** (`maintain`) har bir tenantning
  nusxasini manbadan qayta yozadi — `SyncEdge` bilan bir dalil: **tikerda
  qayta yoziladigan hujjat uni yaratadigan manbadan ajray olmaydi**. Hech nima
  o'zgarmagan bo'lsa yozuv bir xil bo'ladi va soatiga bitta bo'sh yangilanish
  turadi.

### Kuryer pulini kassaning o'zida belgilash
Kuryerning naqd puli faqat **paneldagi kuryer sahifasidan** yozilardi — ya'ni
pulni **olgan odam** emas, boshqa birov, keyinroq, xotiradan yozardi (yoki
umuman yozmasdi). Pul esa kassada beriladi: kuryer qaytadi va cho'ntagidagi
pulni kassada turgan odamga uzatadi.

Endi kassa ekranining **Onlayn** ro'yxatidagi har qatorda tugma bor
(`POST /staff/online/{id}/paid`).

- ⚠️ **Kuryer bo'yicha emas, buyurtma bo'yicha**: panel kuryerning **butun**
  qoldig'ini yopadi (smena oxiri uchun to'g'ri shakl), kassadagi ekran esa
  buyurtmalar ro'yxati, va u har qator haqida bitta savolga javob beradi —
  «**shuning** puli qaytdimi?».
- ⚠️ **To'lov turi qayta yozilmaydi.** Mijoz kassada emas, checkoutda tanlagan:
  kuryerning terminalidan o'tgan karta pul **bu yashikda emas**, va uning
  ustiga «naqd» deb yozish kassirdan kunlik karta savdosicha ko'p pul talab
  qilardi.
- ⚠️ **Kuryerga hisob-kitob yozuvi faqat naqdda**: `cashWithCouriers`
  «cho'ntaklardagi pul» ni **naqd** buyurtmalardan sanaydi, ya'ni karta uchun
  yozilgan hisob-kitob kuryerning qarzini ikki marta kamaytirardi.
- ⚠️ **Yetkazilmagan buyurtmada rad etiladi**: qarz `delivered` naqd
  buyurtmalardan hisoblanadi, demak yetkazilishidan oldingi yozuv kuryerni
  «ortiqcha to'lagan» qilib ko'rsatadi va `cashWithCouriers` dagi clamp butun
  qoldiqni **nolga** yashiradi. Va bu shunchaki rost emas: pul hali qaytmagan.
- **Ruxsat — kassir** (`PermCashier`), ofitsiantniki emas: bu chek yopish bilan
  bir amal.
- **Ikki marta olinmaydi**: yangilanish `paymentStatus != paid` sharti bilan
  guard qilingan, ya'ni panel va kassa bir vaqtda bosса ham pul bir marta
  yoziladi.

### Keel Team'ga bildirishnoma kelmasligi: kanal
⚠️ **Ilova o'zini boshqa ilova deb ro'yxatdan o'tkazsa, hech nima kelmaydi va
hech nima aytmaydi.** Ikkala telefon ilovasi ham `staff` bo'lib kiradi, server
esa Android kanalini token bilan kelgan `app` maydonidan tanlaydi: Waiter →
`kitchen`, Team → `team`. Telefon **yaratmagan** kanalga yuborilgan xabarni
Android jimgina tashlab yuboradi — qurilmada xato yo'q, jo'natish natijasida
ham yo'q, ro'yxatdan o'tish esa muvaffaqiyatli. Natijada sozlamalar ekrani
yashil turadi, server «yuborildi» deydi, xodim esa hech qachon hech nima
olmaydi.

Aynan shu bo'ldi: umumiy `staffRegisterPush` bu maydonni **umuman
yubormasdi**, ya'ni har bir Team telefoni bazada `app: "waiter"` bo'lib turardi.

Tuzatildi va **test bilan muhrlandi** (`src/lib/pushchannel.test.ts`): xato har
bir faylda alohida qaralganda ko'rinmaydi — ikkalasi birga o'qilgandagina
ko'rinadi, shuning uchun test «qaysi kanalni yaratsang, o'sha ilova bo'lib
ro'yxatdan o't» degan qoidani tekshiradi. Eski yozuvlar keyingi ishga
tushirishda o'zi tuzaladi (har ochilishda qayta ro'yxatdan o'tadi).

### Moliyaviy hisobot va kassa (naqd hisobi)
- ⚠️ **Moliyaviy hisobot foyda hisoboti EMAS**, va buni hisobotning o'zi
  yozadi. Tizimda taom tannarxi yo'q (ingredient ham, texkarta ham), demak
  "kirim − chiqim" — **pul harakati**, foyda emas. Restoran uni foyda deb
  o'qisa, butun ovqat tannarxi qadar oshirib ko'rsatadi. Ma'nosi kod izohida
  emas, ekranda turishi kerak: bunday raqam ertami-kechmi bank arizasiga tushadi.
- ⚠️ **Chegirma va ballar xarajat emas**: pul chiqmagan, u umuman kelmagan.
  Tushum qatori allaqachon ulardan tozalangan, ya'ni ularni yana ayirish
  kampaniyani ikki marta hisoblash bo'lardi. Ular "ma'lumot" sifatida turadi.
- ⚠️ **Kuryer topshirig'i chiqim emas** — bu kuryer bizning nomimizdan yig'gan
  naqdning kassaga kirishi. Uni chiqim deb sanash restoranning o'z tushumini
  o'zidan ayirish demakdir.
- **Kassa smenasi** (`cash_shift`): ochilish qoldig'i → sotuv → topshiriqlar →
  sanash. ⚠️ **Mahsulot — farq, jami emas.** Kutilgan summani ko'rsatib,
  sanalganini yozdirib, faqat ikkinchisini saqlaydigan ekran hech nima
  yozmagan: u ochish uchun qurilgan kamomad uni qilgan bo'lishi mumkin bo'lgan
  odam tomonidan o'chirilgan. Shuning uchun `expected` yopish paytida
  **muzlatiladi**, `variance` saqlanadi (o'qishda qayta hisoblanmaydi), va
  **farq sababsiz saqlanmaydi** (400).
- ⚠️ **Yetkazishdagi naqd to'g'ridan-to'g'ri sanalmaydi**: u kassaga kuryer
  topshirgandan keyin kiradi. Ikkalasini ham sanash har yetkazishni
  ikkilantirardi. Kuryer qo'lidagi pul alohida ko'rsatiladi (`withCouriers`) —
  kamomadni tekshirayotgan ega birinchi navbatda shu raqamni so'raydi.
- Filialga **bitta ochiq smena** (409): ikkita ochiq smenada "kassada qancha
  bo'lishi kerak" savoli javobsiz qoladi.

### Grafiklar (Chart.js)
- `components/admin/Charts.tsx` (panel) va `keel-site/src/components/Charts.tsx`
  — **ko'chirilgan, import qilinmagan**: ikki alohida build, logotip nishonchasi
  bilan bir qaror.
- ⚠️ **Seriya palitrasi qat'iy va restoranning brend rangidan olinmaydi.**
  Tenant o'z aksentini tanlaydi; undan qurilgan kategorik shkala brending
  o'zgarganda ma'nosini o'zgartirardi, va ikki kategoriya ajratib bo'lmaydigan
  ohangga tushishi mumkin edi. Beshta ohang **rang ko'rish nuqsoni uchun
  tekshirilgan** (validator, light va dark alohida).
- **Bitta o'q, doim.** Buyurtma va pul hech qachon ikki shkalali bitta
  grafikda chizilmaydi: bunday grafikning shakli ikki o'qning qayerda
  nollangani bilan hal bo'ladi, ya'ni hech nima bilan.
- ⚠️ **Sanoq bo'lsa o'q butun sonda** (`precision: 0`). Chart.js qadamni
  diapazondan tanlaydi, ya'ni to'rtta buyurtmali grafik `0.5, 1.5, 2.5` deb
  belgilanadi — yarim buyurtma yo'q. Buni faqat **chizilgan grafikka qarab**
  topish mumkin; hech qanday palitra tekshiruvi ko'rsatmaydi.
- Kunlik qatorda **bo'sh kunlar ham bor**: faqat sotuv bo'lgan kunlarni chizish
  yopiq haftani tekis chiziqqa aylantiradi va sokin dushanbani ko'rinmas
  qiladi (XYZ dagi "sotuvsiz kun — nol" qoidasining narigi tomoni).

### QR bilan ishga kirish (filial kiosk ekrani)
- **Bosma QR devorga yozilgan parol.** Uni bir marta rasmga olgan odam uyidan
  turib bir yil davomida "ishga kirdim" bosib yuraveradi. Shuning uchun kod
  **qog'ozda emas, ekranda** turadi va **har 30 soniyada** o'zgaradi
  (`handlers/kiosk.go`): `HMAC-SHA256(branch.kioskSecret, branchId.step)`.
- **Hech narsa saqlanmaydi**: server kodni qaytadan hisoblab solishtiradi.
  Ya'ni amal qilish muddatini kuzatadigan jadval ham, tozalash ham yo'q, va
  server qayta ishga tushsa hech qanday kod bekor bo'lmaydi.
- **±1 qadam qabul qilinadi** (jami ~60 soniya): ekrandan telefongacha yurish
  va telefon soatining bir oz og'ishi uchun.
- **Kod bilan joylashuv bir-birini almashtirmaydi, to'ldiradi.** Rasmga olingan
  kod uchun ham odam **o'sha yerda** turishi kerak; aldangan GPS uchun esa
  **ekrandagi kod** kerak. Ikkalasi ham `StaffClock` da tekshiriladi.
  Shuning uchun 60 soniya ichida kodni do'stiga yuborish ham yordam bermaydi.
- **Kod o'z filialini nomlaydi**: Chilonzor ekranidagi kod Yunusobodda smena
  ocha olmaydi.
- **Solishtirish `subtle.ConstantTimeCompare`** bilan — endpoint ochiq, bayt
  bo'yicha solishtirish taxminning qanchasi to'g'ri kelganini oshkor qiladi.
- **Ekran tokeni** (`role: "kiosk"`, 1 yil) admin paneldan bir marta olinadi va
  havola orqali planshetga o'tadi (`/kiosk?t=...` → localStorage, manzil
  satridan darhol tozalanadi). Planshet yo'qolsa **"Kalitni almashtirish"** —
  `kioskVersion` oshadi va barcha eski tokenlar o'ladi.
- ⚠️ **`kioskSecret` va `kioskVersion` filial formasidan yozilmaydi**
  (`AdminUpdateBranch` da `delete`). Aks holda sozlamalarni saqlash ularni
  nolga tushirib, ekran tokenini jimgina o'ldirardi — `soldOut` bilan bir xil
  tuzoq, faqat oqibati og'irroq.
- Mijoz tomoni: QR **`/staff?c=<kod>` havolasi** — telefonning o'z kamerasi
  ochadi, alohida skaner kutubxonasi kerak emas (stol QR'i bilan bir naqsh).
  Ilova kodni ko'rsa **o'zi** kirish yoki chiqishni belgilaydi (ochiq smenaga
  qarab) va kodni bir martalik ishlatadi.
- **Bitta kod — bir necha ishchi, lekin har biriga bir marta.** Smena
  almashganda 4 oshpaz birga kelib bitta kodni skanerlaydi va har biri **o'zi
  sifatida** kiradi: kod kimligini emas, "shu ekran oldida, shu daqiqada
  turgandim" degan dalilni beradi; kimligi login tokenidan keladi.
  Ammo **bitta ishchi bitta kodni ikki marta ishlata olmaydi**
  (`shift.inCode`/`outCode` — kodning hash'i). Aks holda ikki marta skanerlash
  (kamera bildirishnomasini ikki bosish, sahifani yangilash, "hech narsa
  bo'lmadi shekilli") smenani darhol yopib qo'yardi — yo'nalishni server
  ochiq smenaga qarab tanlaydi.
- **Telefon soati ahamiyatsiz**: qadam serverning soati bo'yicha hisoblanadi.
- **Ekran ishlamay qolsa** hech kim kira olmaydi — bu ataylab shunday. Zaxira
  yo'l: belgini vaqtincha o'chirish yoki smenani paneldan qo'lda yozish
  (panelda shu ogohlantirish yozilgan).
- `branch.requireKioskCode` — o'chirilgan holatda hammasi avvalgidek ishlaydi.

### Joylashuvga ruxsat (ishchi va kuryer ilovalari)
- **Brauzer bir marta so'raydi.** Rad etilgandan keyin `getCurrentPosition`
  darhol xato qaytaradi, dialog esa boshqa chiqmaydi — sahifa uni qaytara
  olmaydi, faqat odamning o'zi brauzer sozlamalaridan yoqadi. Shuning uchun
  "ruxsat berilmagan" yozuvining o'zi foydasiz: u rost, lekin nima qilishni
  aytmaydi.
- `lib/geo.tsx` — `useGeoPermission()`: holatni **Permissions API** dan o'qiydi
  (`prompt` / `granted` / `denied`), o'zgarishini kuzatadi (sozlamalardan
  yoqilsa sahifani yangilash shart emas) va `request()` beradi.
  Safari'da geolokatsiya uchun Permissions API yo'q → holat `unknown`, bu
  "so'rab ko'rish mumkin" degani.
- `components/GeoPermission.tsx` — uch holat, uch ko'rinish: **so'ralmagan** →
  tugma (xato emas), **rad etilgan** → qurilmaga qarab (iOS / Android /
  kompyuter) qadam-baqadam ko'rsatma, **berilgan** → chaqiruvchining o'z
  holat qatori.
- **Ruxsat bosish orqali so'raladi**: iOS Safari faqat foydalanuvchi
  harakatidan keyin dialog ko'rsatadi.
- **`watchPosition` faqat `granted` bo'lgandan keyin** boshlanadi: `prompt`
  holatida u ekran hech narsa tushuntirmasdan turib dialog chiqarardi,
  `denied` da esa faqat xato callback'ini chaqirardi.
- **HTTPS majburiy**: `window.isSecureContext` false bo'lsa (masalan telefondan
  `http://192.168.x.x` orqali ochilgan) brauzer joylashuvni umuman bermaydi —
  bu ruxsat muammosi emas va alohida xabar bilan ajratilgan.
- `PERMISSION_DENIED` dan boshqa xatolar (timeout, GPS ushlamadi) **ruxsat
  muammosi emas** — bunday odamni brauzer sozlamalariga yuborish foydasiz.

### Kuryer PWA va joylashuv (muhim cheklov)
- `/kuryer` — alohida PWA: o'z `manifest.webmanifest` (scope `/kuryer`) va
  `public/courier-sw.js` service worker'i bor, telefonga ilova sifatida
  o'rnatiladi.
- Joylashuv `navigator.geolocation.watchPosition` bilan **sahifa ishlab
  turganda** yig'iladi, 15 soniyada bir marta batch qilib yuboriladi,
  oflaynda `localStorage` da buferlanadi.
- Smena davomida **Wake Lock** olinadi (ekran o'chmaydi) va ilova fon rejimiga
  o'tganda oxirgi nuqta darhol yuboriladi.
- **Ilova butunlay yopilganda joylashuv uzatilmaydi** — bu brauzer cheklovi,
  kod kamchiligi emas: service worker'ga geolocation berilmaydi, yopilgan PWA
  esa umuman ishlamaydi. Haqiqiy fon kuzatuvi uchun native o'ram kerak
  (Capacitor yoki TWA + foreground service). Shu sababli kuryer ekranida
  "ilovani yopmang" ogohlantirishi bor.

### Kuryer ilovasi (Expo): fon rejimi va bildirishnomalar
PWA'ning yuqoridagi cheklovi kod bilan hal qilinmaydi — u brauzerning o'zi.
Shuning uchun `mobile/courier` yozildi (ofitsiant ilovasi bilan bir naqsh:
qoidalar `frontend/src/lib` dan, ekranlar ilovada).

- **Ikki oqim, va ikkinchisi bezak emas**: ekranda ochiq turganda
  `watchPositionAsync`, ochiq bo'lmaganda `expo-task-manager` vazifasi
  (`startLocationUpdatesAsync` + Android foreground service). ⚠️ **Vazifa
  o'zining JavaScript kontekstida ishlaydi** — React yo'q, xotirada token yo'q,
  va tizim uni ilova **o'ldirilgandan keyin** ham uyg'otadi. Ya'ni har
  chaqiruvda `hydrateTokens()` va `setApiBase` qaytadan bajariladi; "ilova
  tirik" deb yozilgan versiya aynan o'zi qutqarishi kerak bo'lgan holatda
  ishlamaydi.
- ⚠️ **Doimiy bildirishnoma — bu narx, va u to'lanadi**: Android joylashuvni
  faqat foydalanuvchi ko'rib turgan xizmatga beradi. Yashirish mumkin ham emas,
  to'g'ri ham emas — joylashuvini yuborayotgan odam buni ko'rishi va smenani
  yopib to'xtatishi kerak.
- ⚠️ **Fon ishlayotganda ekran uyg'oq ushlanmaydi**: `expo-keep-awake` faqat
  fon ruxsati berilmagan telefonda yoqiladi. Aks holda ilova batareyani hech
  narsa uchun yoqardi.
- **Fon ruxsati rad etilishi — javob, nosozlik emas**: oldingi holat qoladi
  (ilova ochiq turganda ishlaydi), va smena kartasi qaysi rejimda ekanini
  **aytadi** — «cho'ntakka solsangiz bo'ladi» yoki «ilovani ochiq qoldiring».
  Kuryer buni bilishi shart, chunki tugma shunga bog'liq.

**Bildirishnomalar: kuryerga taalluqli har bir voqea**
Kanal `delivery` (oshxonaning `kitchen` kanalidan **alohida**: Android'da
kanalni foydalanuvchi o'chiradi, va bittasini o'chirgan odam ikkinchisini ham
o'chirganini bilmay qoladi). Yo'l — Expo relay, ya'ni sertifikat yo'q.

| Voqea | Qayerdan |
|---|---|
| Yangi buyurtma berildi | `AdminAssignCourier` |
| Buyurtma sizdan olindi | `AdminAssignCourier` (yechildi/almashtirildi), `AdminMoveOrderBranch` |
| Buyurtma bekor qilindi | `UpdateOrderStatus` (+ sabab) |
| Buyurtma tayyor — olib chiqing | KDS `notifyReady` |
| Manzil o'zgardi | `AdminUpdateOrderAddress` |
| Naqd qabul qilindi | `AdminSettleCourierCash` |
| Hisob o'chirildi | `AdminUpdateCourier` (faqat `true → false`) |

- ⚠️ **Manzil o'zgarishi — xushmuomalalik emas**: kelish tekshiruvi aynan
  o'sha nuqtaga qaraydi, ya'ni eski manzilda turgan kuryerning tugmasi
  ochilmaydi va sababi boshqa odamning ekranida sodir bo'lgan.
- ⚠️ **«Sizdan olindi» oldingi kuryerga yuboriladi**: busiz u endi o'ziniki
  bo'lmagan manzilga borib, buni eshik oldida biladi.
- ⚠️ **Matn serverda yoziladi, ya'ni telefon uni tarjima qila olmaydi** —
  shuning uchun til **token bilan birga** saqlanadi (`courier_device.lang`) va
  xabar `internal/i18n` katalogidan o'tadi, naqshlari bilan: «#12 buyurtma
  sizga berildi. Manzil: Chilonzor 5» → `"#%s buyurtma sizga berildi. Manzil:
  %s"`. Ushlangan qiymat tarjima qilinmaydi — ko'cha nomi tarjima
  qilinmasligi kerak.
- ⚠️ **Summa jumla ichida raqam bo'lib ketadi** (`"%d so'm naqd pul qabul
  qilindi"`), oldindan formatlangan qator emas: «12000 so'm» ruscha
  bildirishnomaga o'zbekcha so'z olib kirardi.
- ⚠️ **`courier_device` alohida kolleksiya**, `staff_device` ga rol ustuni
  qo'shilmadi: id'lar boshqa kolleksiyalardan keladi, va bitta maydonda ikki xil
  id saqlash — bildirishnoma noto'g'ri odamga borishining klassik yo'li.
  `token` unique (ilova har ochilganda qayta ro'yxatdan o'tadi), `courierId`
  bo'yicha indeks (yuborish faqat shu savolni beradi).
- ⚠️ **Chiqishda token o'chiriladi va fon xizmati to'xtatiladi** — kuryerning
  telefoni sotiladigan yoki keyingi kuryerga beriladigan telefon, va qolib
  ketgan token mijozlarning ismi, raqami va manzilini uni ushlab turgan odamga
  yuboradi.

### Maintenance buyruqlari (`backend/cmd/`)
- `cmd/server` — API serveri.
- `cmd/seedmenu` — namuna menyuni mavjud bazaga yozish (`-db`, `-replace`, `-y`).
- `cmd/paytest` — **to'lov tizimini bank ulanmasdan tekshirish**: Payme/Click/
  Uzum o'rniga o'zi qo'ng'iroq qiladi, kalitlarni bazadan o'qib bank kabi
  imzolaydi (`-order`, `-suite`, `-step`, `-provider`, `-api`). Sabab:
  provayderlar faqat ommaviy HTTPS manzilga chiqadi, va "server bank
  chaqirganda to'g'ri ish qiladimi?" degan savolga javob berish uchun
  shartnoma ham, tunnel ham kerak emas.
- `cmd/adminreset` — **admin parolini tiklash** (`-list`, `-username`,
  `-password`, `-create`, `-force-change`). Admin panelda "parolni unutdim"
  oqimi yo'q — tiklash serverda shu buyruq orqali (DEPLOY.md ga qarang).
- `cmd/demodata` — bir oylik "tirik restoran": buyurtmalar, smenalar, kassa,
  ombor. Marketing skrinshotlari uchun yozilgan (`-seed` bir xil bo'lsa bir xil
  restoran chiqadi).
  Docker image'da barcha `cmd/*` binarlari bor: `/app/adminreset`, `/app/seedmenu`,
  `/app/paytest`, `/app/demodata`.

**⚠️ `cmd/demodata -stock` — jonli tenantga qaratsa bo'ladigan yagona rejim**

To'liq `demodata` **o'ylab topilgan buyurtmalar, xodimlar va tushum** yozadi.
Skrinshot uchun to'g'ri, jonli restoranga esa **noto'g'ri**: ega ro'yxatdan
o'tishi bilan o'sha soxta tushum uning **tarixiga** aylanadi va uni ajratib
olishning yo'li qolmaydi. `-stock` faqat omborni yozadi: masalliq, texkarta,
kirim, chiqim, sanash — buyurtma ham, xodim ham, kassa ham yo'q.

- `-cards-all` — **har taomga** karta (standart: eng ko'p sotiladigan uchdan
  bir qismiga). Sabab: ikkinchi ish — restoranning **o'z menyusini** ega
  ko'rishidan oldin tik qilib qo'yish, va u yerda "uchdan biri narxlangan"
  halol standart emas, **yarim ishlagan import** bo'lib o'qiladi.
- `-wipe-stock` — faqat ombor kolleksiyalari va retseptlarni tozalaydi;
  buyurtma, xodim va tushumga tegmaydi. ⚠️ `-wipe` bilan `-stock` **birga rad
  etiladi**: `-wipe` buyurtmalarni ham o'chiradi, va jonli tenantda qaysi biri
  nazarda tutilganini taxmin qilib bo'lmaydi.
- ⚠️ **Ikki marta ishlatib bo'lmaydi**: masalliq bor bazada rad etadi. Aks
  holda ikkinchi "Asosiy ombor", har masalliqning ikkinchi nusxasi va yana bir
  oylik kirim yozilardi — ekranning butun ma'nosi bo'lgan **qoldiqlar ikki
  barobar** bo'lib ketardi.
- ⚠️ **Taomning faolligi tekshirilmaydi.** Yangi import qilingan menyuda
  **hech bir taom faol emas** (importer ataylab o'chiq qoldiradi — §"Menyuni
  havoladan import qilish"), ya'ni `isAvailable` filtri generatorni aynan eng
  kerakli menyuda "hamma taom o'chiq" deb rad etardi. Texkarta taomning bugun
  sotilayotganiga bog'liq emas.

**Kartalar taom nomiga qarab quriladi, narxiga emas (`kitchen` jadvali)**

⚠️ Ilgari faqat narx diapazoni hal qilardi, ya'ni yapon restoranida "Sushi
burger: qo'y go'shti, piyoz, sabzi" chiqardi — bu **bo'sh ombor sahifasidan
yomonroq**, chunki u ko'rsatilayotgan odamga xato ekani darrov ko'rinadi.

- **Uch tilda**: import qilingan menyu ega e'lon qilgan tilda keladi — `Losos`,
  `Лосось`, `Salmon` bitta baliq, va bittasini tanish menyuning uchdan ikkisini
  zaxira qoidaga tashlaydi.
- ⚠️ **Ikkinchi so'z shart** (`and`): "Avokado maki" va "Maki bodring" —
  ichida baliq yo'q rolllar, umumiy qoida esa ikkalasini ham lososdan quradi:
  tarelkada ham xato, 21 000 so'mlik taomda **62% tannarx** ham. So'z tartibi
  menyudan menyuga o'zgaradi, shuning uchun **juftlik** qidiriladi, ibora emas.
- ⚠️ **Oqsil qoidalari sous va pishirish usulidan yuqorida**: "Teriyaki
  sousidagi buzoq go'shti" — mol go'shti, "teriyaki"ni birinchi o'qigan qoida
  esa uni tovuqdan quradi.
- ⚠️ **To'plamga karta yozilmaydi** (`uncarded`). "Set №10" — boshqa oltita
  taomning likopchasi, uning o'z masallig'i yo'q va model shuni aytadi: combo
  omborga **a'zolariga yoyilib** tushadi (`soldDishes`), ya'ni o'z retsepti
  bo'lsa o'sha baliq **ikki marta** sanalardi. Narx diapazoni unga 0.32 kg qo'y
  go'shti yozardi — likopchada ham, tannarxda ham xato. ⚠️ Ro'yxatda
  "to'plam" ham bor: import qilingan menyu o'zbekcha bo'lishi mumkin.
- ⚠️ **Guruch, ugra va kartoshka — "bulk"**: pozitsiyasi bo'yicha garnir,
  og'irligi bo'yicha porsiya. Boshqa garnirlar kabi 20–80 g qilib o'lchansa,
  bir kosa ramenda bir qoshiq ugra bo'ladi va karta **pishirib bo'lmaydigan**
  tannarx beradi (75 000 so'mlik ramen — 6 700, ya'ni 8.9%).
- ⚠️ **Donali narsalar butun songa qaytariladi.** Miqdorni kichraytirish
  1.191 dona nori yasaydi, va kartaning generatsiya qilingani aynan shu
  qatordan bilinadi.

**Sarf tarixsiz ham o'lchanadi (`assumeUsage`)**

⚠️ `-stock` da buyurtma yozilmaydi, ya'ni "nima pishirildi" ma'lum emas va har
miqdor o'zining "hali hech kim ishlatmagan" zaxirasiga tushardi: javonda uch
kilodan hamma narsa, ikkitadan sotib olingan, minimumi bitta. Endi **kunlik
mehmon soni** taxmin qilinadi va menyuga taqsimlanadi. ⚠️ **Taom boshiga
emas** — 119 ta karta kuniga oltitadan 470 ta mehmon degani, va javonda 400 kg
qo'y go'shti paydo bo'lardi. Restoranda **mehmon soni** bor; u qanchalik
yupqa taqsimlanishi menyuning ishi.

Yamato menyusida (119 taom, jonli import): 109 ta karta, 10 ta to'plam kartasiz,
tannarx **9.5%–42.1%**, manfiy qoldiq yo'q.

### Boshqa POS'dan ko'chirish (iiko, r_keeper, Clopos, Poster, Jowi)

**Bu — e'tiroz, qulaylik emas.** iiko yoki boshqa POS'da ishlayotgan restoran u
yerda dasturni yaxshi ko'rgani uchun qolmaydi. Masalliqlar ro'yxati, texkartalar
va ombor **o'sha yerda**, va ularni qo'lda ko'chirish kimningdir haftalari.
«Hamma narsa boshqa POS'da» — shartnoma imzolagan mijoz bilan ishlaydigan mijoz
orasidagi oxirgi to'siq.

**Fayl, integratsiya emas** — va sabab uchta, uchinchisi hal qiladi:

1. Ularning hammasi Excel'ga eksport qiladi. Birortasi ham texkartani **o'qish**
   uchun API hujjatlamagan, hujjatlagani esa uni shartnomadan keyin beradi —
   restoran esa aynan o'sha shartnomadan chiqayotgan bo'ladi.
2. Chiqib ketayotgan restoranda API kaliti odatda yo'q: litsenziya diler nomida.
3. ⚠️ **Bu yerda wire formatni taxmin qilish foydasizdan ham yomon.** Hujjatsiz
   API'ga qarshi yozilgan adapter kompilyatsiya bo'ladi, review'dan o'tadi va
   texkartani **noto'g'ri birlikda** import qiladi — bu **hisoblangandek
   ko'rinadigan va ming marta noto'g'ri** tannarx. Fiskal paketdagi bilan bir
   qoida.

⚠️ **Bu importda AI umuman ishlatilmaydi**, va ishlatilmasligi kerak ham:
Excel katagida turgan raqamni o'qish uchun model kerak emas, va model o'qigan
tannarx — tekshirib bo'lmaydigan tannarx. Hammasi qat'iy qoidalar:
sarlavha, ajratgich, kodlash, birlik, o'nlik vergul.

**Uch tur:** masalliqlar → texkartalar → ombor qoldig'i. ⚠️ Tartib majburiy va
ekranda yozilgan: texkarta masalliqni **nomi bo'yicha** topadi, ombor esa uni
sanaydi — masalliqlarsiz har bir qator «topilmadi» bo'lib chiqadi va bu fayl
noto'g'ri degan taassurot beradi.

**⚠️ O'lchov birligi — import jimgina ming marta xato bo'ladigan joy**

- Bizda uchta xarid birligi (kg / l / dona), retsept esa mingdan birida
  (`models.PerUnit`). Eksportlar bunga ham, bir-biriga ham mos kelmaydi: iiko
  bir ustunda `кг` va `гр` ni yonma-yon yozadi, Poster `шт`, Clopos `dona`.
- ⚠️ **Tanilmagan birlik — o'sha qatorda xato, hech qachon standart qiymat
  emas.** `pcs` ga tushirish aynan falokat: bir kilo mol go'shti bir donaga
  aylanadi, ya'ni taomdagi 180 g **180 kilo narxida** yoki umuman nolga tushadi —
  ekrandagi har bir raqam esa ishonarli bo'lib qolaveradi.
- ⚠️ **Gramm birlik emas, u kichkina yozilgan kilogramm.** Mamlakatda hech bir
  yetkazib beruvchi gramm bo'yicha hisob-faktura yozmaydi, ya'ni xarid ustunidagi
  `гр` — eksport retsept birligini xarid ustuniga yozib qo'yganini bildiradi.
  Shuning uchun u kg ga o'giriladi **va bu haqda aytiladi**: yonidagi narx
  kilogramm narxi sifatida o'qilishi shart, aks holda masalliq ming marta arzon
  bo'lib, undagi har bir taom bepulday ko'rinadi.
- `0,18 кг` va `180 гр` — bir xil 180 gramm. Ikkalasidan birini o'girmasdan
  saqlash ming barobar xato beradi, **qarama-qarshi tomonlarga**: bir taom bepul,
  keyingisi halokatli ko'rinadi.

**⚠️ O'nlik vergul — bu importdagi eng qimmat belgi**

Bu tizimlarning hammasi rus lokalidan eksport qiladi: 180 gramm `0,180` deb
yoziladi va `ParseFloat` uni o'qiy olmaydi. Vasvasali yechim — o'qilmagan
katakni **nol** deb olish — texkartani hamma miqdori nolga teng qilib import
qiladi, ya'ni har bir taom hech nima turmaydi, ya'ni tannarx hisoboti oshxona
bepul deydi. Arifmetik jihatdan izchil, to'g'ri ko'rinadigan taomlar bilan to'la
ekranda turadi, va **inventarizatsiyada** aniqlanadi.

- Minglik ajratgichi — **bo'shliq**, ba'zan uzilmaydigan bo'shliq: `1 234,56`.
  U olib tashlanadi, ajratgich sifatida o'qilmaydi.
- Yolg'iz vergul — **o'nlik vergul**: `1,5` bu bir yarim kilo, o'n besh emas.
- ⚠️ **O'qib bo'lmagan katak — xato, hech qachon nol.** Bu yerdagi sukut —
  odam tuzatadigan import bilan odam ishonadigan import orasidagi farq.

**Fayl o'qish**

- ⚠️ **Sarlavha birinchi qatorda emas.** Har bir eksport tepasiga hisobot nomi,
  sana oralig'i va restoran nomini yozadi. Birinchi qatorni sarlavha deb olish
  har bir ustunni hisobot sarlavhasi bilan nomlaydi, hech nima mos kelmaydi va
  mutlaqo o'qiladigan fayl o'qib bo'lmaydigan bo'lib ko'rinadi.
- ⚠️ **Ajratgich rus Windows'ida nuqtali vergul**, chunki vergul o'nlik belgisi.
  Noto'g'risi bilan har bir qator bitta katak bo'ladi va fayl bir ustunli
  ko'rinadi.
- ⚠️ **Windows-1251 — odatiy holat**, istisno emas. UTF-8 deb o'qilsa sarlavha
  almashtirish belgilariga aylanadi, birorta ustun tanilmaydi va import faylni
  umuman tushunmagandek ko'rinadi.
- ⚠️ **Sarlavha aniq moslik bo'yicha topiladi, substring bilan emas.** «Цена»
  «Цена продажи» ichida ham, «Средняя цена закупки» ichida ham bor; substring
  qoidasi uchalasini bitta maydonga bog'laydi va oxirgi skanerlangani jimgina
  yutadi — shundan keyin tannarx **sotuv narxidan** hisoblanadi.

**Ustunlar taxmin qilinadi, keyin ko'rsatiladi — taxmin qilinib ishlatilmaydi.**
Beshta tizim, har birining o'z sarlavhalari, uch tilda, va har biri keyingi
relizda ustunni qayta nomlashga haqli. Har bir POS uchun qattiq yozilgan parser
buzilgan kunigacha ishlaydi, nosozlik esa **noto'g'ri ustunni o'qigan import**
bo'ladi — buni qarab turib bilib bo'lmaydi.

**Boshlang'ich qoldiq — inventarizatsiya, kirim emas**

⚠️ Qoldiqni kirim (purchase) sifatida yozish hech qachon bo'lmagan yetkazib
beruvchini, sanani va tannarxni **o'ylab chiqaradi**, va «bu oy nima sotib
oldik» hisoboti ularni abadiy ko'tarib yuradi. Sanoq — halol yozuv: «boshlagan
kunimizda javonda shu turgan».

⚠️ **Birinchi sanoqdagi farq — bu boshlang'ich qoldiqning o'zi, va u katta
bo'lishi kerak.** Bizning daftarimiz hech nima kutmaydi, chunki unga hech nima
kirim qilinmagan — shuning uchun butun javon ortiqcha bo'lib keladi. Bu
**bizning yozuvimiz haqidagi rost gap**, ularning javoni haqidagi muammo emas,
va izoh buni sanoqning o'zida aytadi.

**Boshqa qarorlar**

- ⚠️ **Yomon qator tashlanmaydi, olib yuriladi.** O'qib bo'lmagan qirq qatorni
  jimgina o'tkazib yuboradigan import to'liq ko'rinadigan va to'liq bo'lmagan
  ro'yxat beradi, o'sha qirqta esa haftalar keyin **tannarxi yo'q taomlar**
  bo'lib topiladi. Hammasi qaytadi, muammolilari tepaga chiqadi va tuzatilmaguncha
  belgilab bo'lmaydi.
- ⚠️ **Nomsiz qator muammo deb ko'rsatilmaydi** — u oraliq jami yoki bo'lim
  sarlavhasi. Eksportning o'z formatidan iborat qirqta «muammo» odamni muammo
  ustunini umuman o'qimaslikka o'rgatadi.
- ⚠️ **Texkarta taom bo'yicha guruhlanadi va bir marta yoziladi.** Qatorma-qator
  qo'shish qayta import qilganda har bir retseptni ikkilantirardi, yarim
  bajarilgan import esa taomni **yarim masalliqdan** hisoblangan tannarx bilan
  qoldirardi — noto'g'ri va yaxshi ko'rinadigan raqam.
- ⚠️ **Topilmagan masalliq yaratilmaydi, aytiladi.** Bu yerda yaratilgan
  masalliqning birligi ham, narxi ham bo'lmasdi, ya'ni undagi har bir taom o'sha
  qator uchun nolga hisoblanardi — jimgina to'liq bo'lmagan karta ko'rinib turgan
  yo'q kartadan yomonroq.
- Nomlar `menuimport.NormalName` bilan solishtiriladi: eksport `Lagʻmon` yozadi,
  panelda `Lag'mon` turadi.
- Tugma **masalliqlar sahifasida**, sozlamalarda emas: bu — odam oldida qancha
  yozuv turganini tushunadigan ekran. Sozlamalarga yashirilgan ko'chirish
  vositasini ega qirqta masalliqni qo'lda kiritgandan **keyin** topadi.

### Menyuni havoladan import qilish

**Menyuni qo'lda yozib chiqish — restoranni ishga tushirishdagi eng uzun ish**:
yuzta taom, har birida nom, narx, tavsif va rasm. Deyarli har bir restoranda bu
allaqachon bor — eski saytida, Express24 yoki Uzum Tezkor sahifasida — va uni
qaytadan yozish shartnoma imzolagan mijoz uch hafta ishga tushmasligining sababi.

**Ikki qadam, hech qachon bitta.** `preview` sahifani o'qib **taklif** qiladi,
`apply` esa ega belgilaganini yozadi.
⚠️ Bitta bosishda yuz yigirma taomni jonli menyuga yozadigan import — qo'lda
qaytarib bo'lmaydigan xato, va xatolar **kafolatlangan**: kirish ma'lumoti
birovning sahifasi.

**Taomlar faol bo'lib tushsinmi — bu eganing tanlovi, bitta bosish.**

⚠️ **Standart holat — o'chiq, va shunday qolаdi.** Birovning sahifasidan o'qilgan
narxni to'qsonta taom bilan birga o'qimasdan mehmon oldiga qo'yish — o'sha
sahifa nima yozgan bo'lsa shu narxda sotish, va bahs kassada, raqamni umuman
ko'rmagan kassir bilan bo'ladi.

⚠️ Lekin **o'z menyusini** import qilayotgan ega — bu odatiy holat — shundan
keyin to'qsonta taomni birma-bir yoqib chiqishi kerak bo'lardi. Shuning uchun
belgi bor va u bitta bosish.

⚠️ **Natija ekrani ikkitasidan qaysi biri bo'lganini aytadi.** «Taomlar o'chiq»
ataylab qilingan va aynan «import ishlamadi» deb xabar qilinadigan narsa;
«taomlar faol» esa mehmon buyurtma berishidan oldin bilinishi kerak bo'lgan
narsa. Bitta jumla yozib to'g'risiga umid qilish ikkalasini ham yomon
bajarardi.

**⚠️ 502 Bad Gateway: uzun ish so'rov ichida yashay olmaydi**

Import qo'shishni bosgach bir ozdan keyin 502 kelardi. Sabab: apply har taomga
bitta rasm yuklaydi — to'qsonta taom bu birovning serveriga to'qsonta so'rov,
ya'ni daqiqalar. Router handlerga **30 soniya** beradi (`chimw.Timeout`), chekka
undan ham kam — ulanish ish tugashidan ancha oldin uzilardi.

⚠️ **Va bu nosozlikning eng yomon shakli**: hech nima bo'lmagandek ko'rinadi,
aslida esa import **davom etayotgan va taomlarni yozayotgan** bo'ladi. Tugmani
qayta bosish butun menyuni ikkinchi marta import qilardi.

Endi apply **job id** qaytaradi, ish orqa fonda ketadi, panel esa qanday
ketayotganini so'rab turadi.

- ⚠️ **Preview ham xuddi shu devorga urilardi**: sahifani olish 20 soniya,
  AI yana 60 — 30 soniyalik handlerga qarshi. AI kerak bo'lgan har safar 502
  kelardi, va bu **to'g'ri havola haqida «havola noto'g'ri»** degan taassurot
  beradi. U ham jobga o'tkazildi.
- ⚠️ **Progress bar — serverning o'z hisobi, animatsiya emas.** Belgilangan
  tezlikda to'ladigan bar spinnerdan yomonroq: to'qsonta rasm hali yuklanayotganda
  «deyarli tayyor» deydi va ega 95% da tabni yopadi. Foiz yonida **son** ham
  turadi — «43%» o'zi bu 43 ta taommi yoki 430 tami demaydi.
- ⚠️ **Ish xotirada saqlanadi va qayta ishga tushganda unutiladi.** Job —
  ko'rsatkich, yozuv emas: haqiqatan import qilingani menyuda, bazada. Yarim
  import qolgani ikkala holatda ham shunday, va halol tiklanish — keyingi
  yugurishdagi dublikat tekshiruvi, ishni davom ettira olaman deb ko'rsatadigan
  navbat emas.
- ⚠️ **Job topilmasa 404 emas.** Muddati o'tgan yoki restartda yo'qolgan job —
  yo'q sahifa emas: unga tegishli import tugagan bo'lishi mumkin. Panel
  «yo'qotdik, menyuni tekshiring, qayta bosmang» deydi — «xato» deyish egani
  tugmani qayta bosishga va menyuni ikkilantirishga jo'natardi.
- ⚠️ **Worker `*http.Request` ushlab qolmaydi.** Brend, filial linzasi va
  admin nomi **so'rovda** o'qiladi; handler qaytgandan keyin so'rovdan o'qish —
  alomati «jurnalda amal keyin kirgan odamga yozilib qolgan» bo'ladigan
  data race. Shuning uchun `logActionAs` alohida.

**To'rtta nomlangan o'quvchi, va model — oxirgisi**

⚠️ **Har bir sayt boshqacha, shuning uchun shakllar topib olinmaydi — yozib
qo'yiladi.** Bittasi schema.org e'lon qiladi, keyingisi menyuni JavaScript
bundle holatida yuboradi, uchinchisi o'z JSON API'sidan oladi, to'rtinchisi esa
qo'lda yozilgan oddiy HTML. Handler ichiga yashiringan `if len(dishes) == 0`
zanjiri — bir xil mantiq, lekin hech biri **aytilmagan**, va u egaga sahifasi
**nega** ishlaganini yoki ishlamaganda **to'rttadan qaysi birini** sinash
kerakligini ayta olmaydi.

Endi har biri nomlangan o'quvchi (`menuimport.Readers()`), va javob **qaysi biri
o'qiganini aytadi**. «Import ishlamadi» bilan «bu sahifa hech nima e'lon
qilmaydi — fayldan import qiling» orasidagi farq shu.

**⚠️ Agregatorlar: sahifada hech nima yo'q, hatto matn ham**

Yandex Eats havolasi import qilinmadi, va to'rtala o'quvchi ham **to'g'ri**
ishlagan edi: sahifa 122 KB markup qaytaradi, ichida bitta taom ham, bitta narx
ham, o'qiladigan bitta jumla ham yo'q. Menyu keyin, saytning o'z API'sidan
keladi. Ya'ni modelga **bo'sh satr** beriladi va ega ekranda o'nlab taomni
ko'rib turib «bu sahifada menyu topilmadi» degan javobni oladi. Har jihatdan
to'g'ri, foydasiz, va buzilgan importdan farq qilmaydi — aynan shunday xabar
qilindi ham.

- **Umumiy API o'quvchisi ularni topa olmaydi va bu uning kamchiligi emas.** U
  sayt ildizida `/api/v1/menu` va beshta qo'shnisini sinaydi — restoranning
  **o'z** sayti menyuni shu yerda saqlaydi. Agregatorda esa manzil ichida
  **restoranning slugi** bo'ladi, va slug borligini bilmasdan uni taxmin qilib
  bo'lmaydi.
- Shuning uchun **nomlangan agregator o'quvchisi** (`aggregator.go`,
  `ReaderAggregator`; hozircha Yandex Eats va Uzum Tezkor) — ro'yxatda **birinchi**, va bu tartib qoidasini
  buzmaydi: bu saytning o'zi e'lon qilgan JSON, ya'ni eng aniq manba. Boshqa
  saytlarga **bitta ham so'rov qo'shmaydi** — avval host mos kelishi shart.
- ⚠️ **Yo'ldagi slug API kutgan slug emas.**
  `/en-uz/tashkent/r/sam_plov_restaurant?placeSlug=sam_plov` da API `sam_plov`
  ga 47 ta taom, `sam_plov_restaurant` ga esa **404** qaytaradi. So'rov
  parametri ustun — u aynan ikkalasi farq qilgani uchun bor. Faqat yo'lni
  o'qish mukammal to'g'ri havolaga bo'sh menyu beradi.
- ⚠️ **Yo'naltirishdan keyingi manzil restoranning manzili emas.** Yandex
  **serverimizning** sahifa so'roviga `/showcaptcha?...&retpath=<haqiqiy manzil>`
  yo'naltirishi bilan javob beradi (brauzernikiga — yo'q). O'quvchilarga
  sahifaning manzili sifatida **o'sha** beriladi: undagi slug `showcaptcha`
  bo'lib chiqadi, menyu API'si unga 404 qaytaradi, va ega «bu sayt bizni
  bloklaydi» degan xabarni oladi. Uning havolasi esa boshidan to'g'ri edi.
  Shuning uchun robot devori aniqlansa **manzil ham, sahifa ham qaytariladi**:
  yozilgan havola — bu yerdagi yagona ma'noli manzil.
- ⚠️ **Agregator sahifa umuman o'qilmaganda ham sinaladi.** Yandex bu
  serverning **sahifa** so'rovini to'sadi, o'sha serverning **menyu API'si**
  so'roviga esa 200 beradi. Ya'ni ega so'ragan menyu mavjud — faqat uning
  atrofidagi qobiq yo'q.
  ⚠️ **Bu nosozlik faqat serverda ko'rinadi**: ish stolidan xuddi shu sahifa
  normal ochiladi, ya'ni captcha umuman bo'lmaydi va xatoni takrorlab
  bo'lmaydi. «Menda ishlayapti» bu yerda hech nimani anglatmaydi, va
  `cmd/menucheck` shu sababdan **serverda** ishlatilishi kerak.
- ⚠️ **Rasm — shablon, manzil emas**: `/images/207/abc-{w}x{h}.jpeg`.
  To'ldirilmagan holda saqlansa, nosozlik **importdan keyin**, menyuning
  ichida, har taomda bitta buzuq rasm bo'lib chiqadi.
- **Sotuvda yo'q taomlar ham olinadi**: bu ega qatorma-qator o'qiydigan taklif,
  va ro'yxatda yo'q taom — u qo'lda yozadigan taom, kerak bo'lmagani esa bitta
  belgi. «Bugun agregatorda tugagan» — «menyuda yo'q» degani emas.

**⚠️ Uzum Tezkor: `uzum.uz` emas, `uzumtezkor.uz`**

Xizmatning ikki eshigi bor va ular butunlay boshqacha tutadi: `uzum.uz` har bir
brauzer bo'lmagan mijozni captcha'ga yuboradi va umuman o'qilmaydi;
`uzumtezkor.uz` esa normal javob beradi va menyusi bitta **avtorizatsiyalangan**
so'rov naridadir. Bitta xizmat deb qarash — o'qiladigan saytni o'qilmaydigan deb
belgilash.

- **Token sahifaning o'zida.** Katalog tokensiz 401 qaytaradi, sayt esa har bir
  mehmonga `__NEXT_DATA__` ichida anonim guest token beradi — brauzer keyin
  aynan shuni ishlatadi. Uni sahifadan olish o'zimiz mint qiladigan ikkinchi
  mexanizmdan sodda va kamroq buziladi.
- ⚠️ **`Accept-Language` — bitta til, ro'yxat emas.** Ularning API'si har bir
  brauzer yuboradigan `uz,ru;q=0.9,en;q=0.8` ni **422** bilan rad etadi
  («should be one of [ru en uz]»). Ya'ni fetcherning o'z sarlavhasi bu yerda
  **ustidan yoziladi**, qo'shilmaydi. Til manzildan olinadi — natijada menyu
  ega o'qib turgan tilda keladi.
- ⚠️ **Narx tiyinda, bizniki esa yo'q.** Pita box `7500000` bo'lib keladi, ya'ni
  75 000 so'm. Shundayligicha import qilinsa — yetti yarim millionlik pita, va
  buni yuzta qatorni ko'zdan kechirayotgan odam **sezmaydi**: hamma narx bir xil
  koeffitsiyentga xato, ya'ni bir-biriga mos ko'rinadi. U kassada, mehmon
  oldida chiqadi. Shuning uchun o'girish **chetda**, begona birlik kelgan
  joyda.
- Mahsulot kategoriyani **id** bilan olib keladi; qidiruvsiz har taom ega hech
  qachon ko'rmagan raqam ostida qolardi.

**⚠️ `FetchHeaders` — ikkinchi tarmoq yo'li emas, bitta yo'lning parametri**

Agregator API'siga `Authorization` kerak, va eng oson yo'l — yoniga kichkina
`http.Get` yozish. Aynan shu tarzda tarmoqqa ikkinchi yo'l paydo bo'ladi va unga
manzil tekshiruvini qo'yish esdan chiqadi. Barcha SSRF qoidalari bitta joyda
qoladi, sarlavhalar esa — yagona o'zgaruvchi qism.

**⚠️ Robot tekshiruvi (captcha) bo'sh menyu bo'lib o'qilardi**

Uzum Tezkor (uzum.uz) brauzer bo'lmagan har bir mijozni Yandex SmartCaptcha'ga
yuboradi: fetch **muvaffaqiyatli**, 200, o'n kilobayt — «Siz robot emasmisiz?».
Tanilmasa, to'rtala o'quvchi unda taom topmaydi, so'ng **haqiqiy matn** modelga
beriladi va **puli to'lanadi**, javob esa «bu sahifada menyu yo'q» bo'ladi.
Uchala qismi ham har xil tarzda noto'g'ri, va hech biri egaga aytilishi kerak
bo'lgan yagona gapni aytmaydi: **bu saytni server umuman o'qiy olmaydi va
boshqa havola yordam bermaydi.**

- `BotWall()` **modeldan oldin** tekshiriladi — to'lanmaydigan xato to'langan
  xatodan yaxshiroq.
- **Ishonchli yarmi — oxirgi manzil** (`/showcaptcha`, `/cdn-cgi/challenge`):
  tekshiruv sahifasining matni mehmon tiliga tarjima qilinadi va relizdan
  relizga o'zgaradi, u yo'naltiradigan manzil esa — mexanizm.
- Matn belgilar faqat hujjatning **boshida** izlanadi: haqiqiy menyu sahifasi
  footeridagi skriptda «captcha» so'zi uchrashi mumkin, tekshiruv sahifasi esa
  buni hamma narsadan oldin aytadi — chunki unda boshqa hech nima yo'q.

**`cmd/menucheck` — havoladan javobgacha bitta buyruq**

Savol doim havola bo'lib keladi: «import ishlamadi, mana». Panel faqat qaysi
o'quvchi javob berganini aytadi, hech biri javob bermaganda esa eng qiziq qism
— sahifada **nima bo'lgani** — serverda qolib ketadi. Uni qo'lda takrorlash
fetch + to'rt o'quvchi + matn ajratish demak; bu yozilgunicha shunday ikki
marta qilindi. Buyruq faqat **o'qiydi**, bazaga tegmaydi — mijoz telefonda
turganda prodda ishlatish xavfsiz.

**⚠️ Beshinchi o'quvchi: `embedded` — sahifa ichidagi oqim (`__NEXT_DATA__` emas)**

Jonli import `yamato.delever.uz` da **hech nima** qaytardi, sahifada esa **119 ta
taom va 20 ta kategoriya** bor edi — hammasi biz yuklab olgan HTML ning ichida.
Buning «bo'sh sahifa» bo'lib ko'rinishi uchun **to'rt narsa bir vaqtda** noto'g'ri
bo'lishi kerak edi, va to'rttasi ham odatiy:

1. ⚠️ **`__NEXT_DATA__` — Next.js ning eskisi.** Pages Router ma'lumotni bitta
   `<script id="__NEXT_DATA__">` ichida beradi; **App Router** — ya'ni o'shandan
   keyin qurilgan har bir Next.js sayti — uni `self.__next_f.push([1,"…"])`
   bo'laklari bilan **oqim** qilib yuboradi. Bo'laklarning qo'shilgani JSON
   hujjat **emas**: bu React'ning flight formati, va menyu o'sha matnning
   ichida oddiy JSON bo'lib yotadi.
   - **Parser emas, skaner.** Flight dekoderi yozish — spetsifikatsiyasi ham,
     muvofiqlik va'dasi ham yo'q formatning ichki tuzilishiga bog'lanish demak.
     O'zgarmaydigan narsa — taomlar o'sha yerda oddiy JSON ekani. Shuning uchun
     matn **balansli JSON qiymatlari** uchun supuriladi (`json.Decoder` bilan,
     qavs sanash bilan emas: tavsifidagi bitta `}` sanoqchini buzadi), va
     parse bo'lgan har bir qiymat **o'sha bitta walker**'ga beriladi.
   - ⚠️ **Bo'lak chegarasi taomning o'rtasidan o'tadi.** Har `push` ni alohida
     parse qilgan o'quvchi bu yerda hech nima topmaydi va **vaqti-vaqti bilan**
     topmaydi — eng yomon shakl, chunki kimdir sinagan sahifada ishlaydi.
   - ⚠️ **Qiymatning ichidan qayta boshlanmaydi.** Aks holda har taom o'zining
     nechta qavat ichida ekaniga qarab **o'nlab marta** qaytardi.
2. ⚠️ **Nom — obyekt**, va O'zbekistondagi platformada odatda shunday:
   `"title": {"uz": "Kuksi", "ru": "Кукси"}`. Qator sifatida o'qilganda u
   **umuman yo'q**: nomi yo'q → taom emas → «sahifa hech nima e'lon qilmaydi».
   Tartib — `uz → ru → en`, keyin **saralangan** kalitlardan birinchisi (Go
   xarita tartibini tasodifiy qiladi: saralamasa bitta sahifa har yugurishda
   boshqa tilda import bo'lardi va **saytning xatosiga o'xshardi**). ⚠️ `ru` ga
   tushish muhim: bu payloadlarning yarmi `uz` ni bo'sh qoldiradi — jonli sahifa
   **har bir tavsif** uchun aynan shunday qiladi.
3. ⚠️ **Narx kaliti — `out_price`.** Buni qo'shmasdan o'quvchi 119 ta nomlangan
   narsani narxsiz topadi, birortasini ham taom deb hisoblamaydi va sahifani
   bo'sh deb xabar qiladi. (Nom shakli bilan narx kaliti **birga** yiqilgani —
   umuman hech nima qaytmaganining sababi.)
4. ⚠️ **Rasm — uuid, URL emas.** `"image": "553fb012-…"` — bu id, va uni hech
   qanday «URL'ga o'xshaydimi» tekshiruvi rasmga aylantirmaydi. Sayt shaklni
   **o'zi aytadi**: favicon, logo va og:image bir xil CDN'da, bir xil yo'l ostida
   turadi — shuning uchun asos **sahifadan o'qiladi**, taxmin qilinmaydi.
   ⚠️ Sahifa shaklni ko'rsatmasa, taomlar rasmsiz import qilinadi: bu halol
   natija, har kartochkadagi singan rasm esa yomonroq va tushuntirishi qiyinroq.
   ⚠️ Faqat **uuid** kengaytiriladi: `"image": "burger.jpg"` — papkasini
   bilmagan fayl nomi, `"image": "1"` — ko'rmagan jadvalga id.

⚠️ **Kategoriya id bilan aytiladi, qo'shni ro'yxatda.** Walker'ning «eng yaqin
o'rab turgan konteyner nomi» qoidasi ichma-ich menyu uchun to'g'ri va bu yerda
**hech nima** topmaydi — har taom bo'limsiz import bo'lardi va ega 119 tasini
qo'lda saralardi. Endi ikki yurish: avval `id → nom` indeksi (`indexNames`),
keyin taomlar. Nom bo'yicha ichma-ich aytilgani **birinchi** o'qiladi: id ni
afzal ko'rish tasodifan `categoryId` olib yurgan ichma-ich taom uchun to'g'ri
javobni buzardi.

⚠️ **Delever mamlakatdagi yetkazib berish saytlarining katta qismini yuritadi**,
ya'ni bu bitta restoran emas. Va bu sayt **har qanday** yo'lga 200 + HTML
qaytaradi, ya'ni `api` o'quvchisining oltita so'rovi bu yerda hech qachon
yordam bermaydi — `embedded` undan **oldin** turishi shuning uchun ham muhim.



Undan yuqoridagi har bir qadam sayt **e'lon qilgan raqamlarni** o'qiydi: aniq,
bepul, modelsiz. ⚠️ Narx JSON maydonida turganda uni gapdan o'qishni so'rash —
yomonroq bo'lish uchun pul to'lash.

1. Sahifadagi **schema.org JSON-LD** — agregatorni Google natijalariga
   chiqaradigan narsa, shuning uchun agregatorlarda deyarli doim bor.
2. **Freymvorkning o'z holat blobi** (`__NEXT_DATA__`, `__NUXT__`) — butun menyu
   allaqachon biz yuklab olgan hujjat ichida, ikkinchi so'rov kerak emas.
3. **Saytning o'z menyu API'si.** ⚠️ **Aynan shu yetishmayotgan edi, va bu
   chekka holat emas — odatiy holat.** Bu o'n yillikda qurilgan restoran
   saytlarining ko'pi menyuni brauzerda chizadi: kelgan sahifa — bo'sh qobiq,
   taomlar keyin JSON'dan keladi. Bo'sh qobiq bilan na schema o'quvchi, na model
   hech nima qila olmaydi — **«sahifa bo'sh» aynan shu edi**.
4. **AI** — sahifa matni bo'yicha. Menyuning rasmidan qilingan sayt uchun, va
   boshqa hech nima uchun emas.

⚠️ **Chuqurlik chegarasi — rekursiya qorovuli, boshqa hech nima.** U **uchta**
edi va bu xato edi: iiko web menyusi narxni `itemSizes[] → prices[] → price` da,
ya'ni besh qavat pastda saqlaydi — ya'ni o'quvchi mamlakatdagi eng keng tarqalgan
sayt konstruktorida **umuman hech nima topmasdi** va AI'ga tushib ketardi. Bu
fayl tuzatish uchun yozilgan xatoning o'zi, ehtiyotkorday ko'ringan raqam orqali
qaytib kelgani.

⚠️ **Yurish `relatedButProducts` kabi shoxlarga kirmaydi.** Upsell ro'yxati o'z
narxi bilan butun taomlarni ko'taradi — ular boshqa bo'limga tegishli va u yerda
baribir chiqadi, ya'ni bu yerda import qilish noto'g'ri kategoriyali dublikat.
Modifikator guruhi esa o'lchamlar va qo'shimchalarni ko'taradi — bu mahsulotda
ular taomning **variantlari**, taom emas: import qilish menyuga «Katta» ni
5 000 so'mga qo'yardi.
⚠️ Buni review emas, **test ushladi**: `findPrice` allaqachon bu joylarga
qaramasdi, ya'ni qorovul to'liq ko'rinardi, yurish esa orqa eshikdan kirib
ketgan edi.

⚠️ **Rasm — asli, topilgan birinchi URL emas.** Bu API'lar `src` ni va yonida
o'lchamlari bo'yicha nomlangan variantlar qatorini (`44x44x100.webp`) e'lon
qiladi, Go xaritasidagi birinchi kalit esa runtime xohlagani — ya'ni «birinchisi»
har bir taomga **tasodifiy 44 pikselli eskiz** import qiladi.

⚠️ **Modelning o'z so'zlari eganing muammosi emas.** Ikki provayderning kvota
xabarlari, billing sahifalari va rate-limit havolalari bu ekranga **so'zma-so'z**
chiqib qolgan edi. Birovning hisobi haqida «your credit balance is too low» ni
o'qigan egaga rost, foydasiz va xavotirli narsa aytilgan bo'ladi. AI bu yerda
to'rtinchi o'quvchi; u ishlamasa javob — **keyin nima qilish kerakligi**.

**Strukturali ma'lumot birinchi, model oxirgi**

- Ko'p menyu sahifalarida schema.org JSON-LD bor — agregatorlarda deyarli doim,
  chunki ularni Google natijalariga chiqaradigan narsa shu. Uni o'qish **aniq va
  bepul**: narx — sayt e'lon qilgan raqam, gapdan o'qib olingani emas.
- Model faqat hech nima e'lon qilmagan sahifalar uchun. ⚠️ Narxi yozilgan
  sahifani modelga o'qitish — **kamroq aniq bo'lish uchun pul to'lash**.
- ⚠️ **Narxsiz taom ham olinadi.** Ko'p sahifa taom nomini JSON-LD ga, narxni esa
  boshqa elementga qo'yadi. Ularni tashlash nom, tavsif va rasmni — bir soat
  oladigan qismni — besh soniya oladigan maydon uchun yo'qotardi.
- ⚠️ **Bo'lim (`MenuSection`) taom emas.** Uni taom deb olish menyuga
  «Salatlar» ni **nol so'mga** qo'yadi.

⚠️ **`45.000` ni float deb o'qish — 45.** Menyu bu yerda qirq besh mingni
`45000`, `45 000`, `45,000` va `45.000` deb yozadi, ikkitasi dunyoning boshqa
joyida o'nlik nuqta. Nuqtali variantni float deb o'qish **qirq besh so'm**
beradi — qutida ishonarli ko'rinadigan, mehmon buyurtma bergunicha hech kim xato
demaydigan raqam. Shuning uchun barcha raqam guruhlari birlashtiriladi va
**hech nima kasr deb qaralmaydi**.

⚠️ **Server foydalanuvchi yozgan manzilga so'rov yuborishi — SSRF, va bu yerdagisi
ko'pchilikdan yomonroq.** Bu konteyner Docker tarmog'ida Mongo, control plane va
**boshqa har bir tenantning backendi** yonida turadi. `http://mongo:27017` —
farazий hujum emas, bu manzil qabul qilish uchun turgan qutiga bitta paste.

- Manzil **hal qilinadi** va **chiqqan har bir IP** tekshiriladi (birinchisi
  emas: hostname bitta ochiq va bitta yopiq manzilga hal bo'lishi mumkin).
- **Redirect'lar qo'lda, bittalab quviladi** — ochiq hostname yopiqqa
  yo'naltirishi terilganini tekshiradigan checkdan o'tishning standart yo'li.
- **Nuqtasiz hostname rad etiladi**: `mongo`, `keel-control`, `keel-<slug>` —
  aynan shu to'plam, va hech bir ochiq saytda nuqtasiz nom yo'q.
- `169.254.0.0/16` alohida yoziladi: ichida **bulut metadata xizmati**
  (169.254.169.254) — ijaraga olingan serverda SSRF yeta oladigan eng qimmatli
  narsa.
- ⚠️ **Rasm manzili ham qaytadan tekshiriladi.** U — sahifa tanlagan URL, va
  tarmoqni o'qimoqchi bo'lgan sahifa `<img src="http://169.254.169.254/">`
  yozsa bas. Bu **o'sha eshik, bir qadam ichkarida**, va aynan shunisi esdan
  chiqadi.
- ⚠️ **`::ffff:0:0/96` blok ro'yxatiga qo'shilmaydi**, garchi joyi shunday
  ko'rinsa ham: Go uni `0.0.0.0/0` ga normallashtiradi, ya'ni **internetdagi har
  bir manzilni** bloklaydi. O'zining reviewsidan o'tadigan va butun xususiyatni
  rad etadigan check. Testi bor (ikkala yo'nalish ham).

**Rasmlar ko'chiriladi, havola qilinmaydi.** Manba URL'ini saqlash — saytdagi
har bir taom rasmini **raqobatchining CDN'i** xizmat qilishi degani: ular
yo'lni o'zgartirgan kuni buziladi, menyuni ochgan har bir mehmon uchun ularga
referer boradi, va bu ularning trafigi. Yuklab olingan rasm yuklangan rasm kabi
1600 px gacha kichraytiriladi.

**Dublikat: nomni solishtirish uchun normallashtiriladi**

⚠️ **Muammo — apostrof.** O'zbekcha `Lag'mon` deb yoziladi, va har bir manba bu
belgini boshqacha yozadi: agregator CMS'i chiqaradigan tipografik `ʻ`, telefon
klaviaturasi chiqaradigan `ʼ`, backtick, va odam bosgan oddiy `'`. **To'rtta
qator, bitta taom** — va ularni harfma-harf solishtirish menyuga ikkinchi
Lag'mon qo'shadi, u ro'yxatda **aynan bir xil** ko'rinadi va shundan keyin har
bir hisobotda alohida qator bo'lib qoladi.

- `NormalName()`: kichik harf, bo'shliqlar siqiladi, har xil apostroflar bittaga
  keltiriladi.
- ⚠️ **Va bundan nariga o'tmaydi**: so'z tashlamaydi, o'zak olmaydi.
  «Lag'mon» va «Lag'mon qovurma» — mamlakatning har bir menyusida ikki xil taom,
  va ularni birlashtiradigan qoida ikkinchisini **jimgina import qilmasdi** —
  yo'qligi ko'rinmaydigan yo'q taom.
- ⚠️ Bo'shliq **siqiladi, olib tashlanmaydi**: «Oshpalov» va «Osh palov» bir xil
  ekani ma'lum emas, taxmin esa bir taomga tushadi.
- ⚠️ **Butun brend bo'yicha tekshiriladi, maqsad kategoriya bo'yicha emas.**
  «Import» bo'limidagi va «Issiq taomlar» dagi bir taom — baribir bir taom, va
  kategoriya aynan importning eng noto'g'ri chiqadigan maydoni (u birovning
  sarlavhalaridan keladi).
- ⚠️ **Kategoriya yaratilishidan oldin tekshiriladi**, aks holda hamma taomi
  allaqachon menyuda bo'lgan sahifani qayta import qilish har bosishda **yangi
  bo'sh bo'lim** qoldirardi.
- To'plam **import davomida ham to'ldiriladi**: bir sahifadagi «ommabop»
  karuseli va uning ostidagi menyu bitta taomni ikki marta beradi.

**Yuklab olingan rasmlarni tozalash**

Import har taomga bitta rasm yuklaydi, va ular taomlardan uzoq yashaydi: sahifa
qayta import qilinsa, taomlarning yarmi o'chirilsa, ikkinchi agregator qo'shilsa
— avvalgi har bir yugurishning rasmi diskda qoladi va har kecha zaxiraga tushadi.

⚠️ **Supurgi faqat o'zi yuklab olgan fayllarni ko'radi. Bu — butun xavfsizlik
dizayni, tafsilot emas.** `uploads/` ni o'qib, havolasi topilmagan hamma narsani
o'chiradigan supurgi bir kuni **restoranning logotipini** o'chiradi — chunki
«havolasi topilmadi» degani aslida «yashirinish joylarini qanchalik to'liq
sanadik» degan da'vo, rasm esa `page_design` ning erkin `settings` xaritasida va
maxsus CSS ichida `url(/uploads/…)` bo'lib yashirinadi. Bitta o'tkazib
yuborilgan joy — jonli rasm yo'q bo'ldi, qaytaradigan joyi yo'q.

- Har bir yuklab olingan fayl `import_asset` ga yoziladi, nomzodlar to'plami
  **faqat shu**. Ega yuklagan fayl unda yo'q va reference-check qanchalik xato
  bo'lsa ham o'chirilmaydi.
- ⚠️ **Yaqinda yozilgan fayl axlat emas — u yo'lda.** Ikki import ikki tabda
  ketishi mumkin: biri rasmni yuklab, ro'yxatini davom ettirayotganda ikkinchisi
  tugab supuradi. 15 daqiqalik muhlat shuning uchun.
- ⚠️ **Savolga javob berib bo'lmasa — harakat qilinmaydi.** Baza yetib
  bo'lmaganda o'chiradigan supurgi — aynan o'sha kuni papkani bo'shatadigan
  supurgi.
- ⚠️ Reference-check **butun hujjatni** o'qiydi, nomlangan maydonlarni emas:
  dizaynning rasmi bu kod nomlay olmaydigan kalitlar ostida yotadi.
- Yozuv fayl **o'chgandan keyin** o'chiriladi, teskarisi emas — teskarisi
  o'chmagan faylni ko'zdan yo'qotadi va orphan abadiy qoladi.
- Supurish **import tugagach** va **taom o'chirilganda** ishlaydi. Ikkinchisi
  kerak: taom o'chirish — rasm axlatga aylanishining eng keng tarqalgan yo'li,
  va faqat importga ulash bir marta import qilib keyin menyusini tartibga
  solgan restoranda supurishni **umuman ishlatmasdi**.

**Boshqalar**

- Bo'limlar **yaratiladi**, tashlanmaydi: bo'limsiz menyu — to'qsonta taomning
  yassi ro'yxati, va uni qo'lda saralash ega tejagan ishning ko'p qismi.
- Dublikat **ikki marta** tekshiriladi (preview va apply): ikki bosish orasida
  ega o'sha sahifani ikki marta import qilishi mumkin.
- «Import» tugmasi kategoriya talab qilmaydi, «Taom qo'shish» dan farqli.
  ⚠️ Yangi ro'yxatdan o'tgan restoranda menyu ham, kategoriya ham bo'sh — bu
  tugma eng qimmat bo'lgan aynan o'sha lahza.
- ⚠️ Sahifada tayyor ma'lumot bo'lmay, modelni ishlatishga to'g'ri kelsa — bu
  **ekranda aytiladi**. Ro'yxat ikkalasida ham bir xil tekshirishga arziydi,
  lekin menyu rasmini mashina o'qiganini bilgan odam narxlarni tekshiradi,
  bazadan kelgan deb o'ylagan odam tekshirmaydi.
- AI byudjeti **brifing va kampaniya bilan bitta** — ega eshitgan kunlik limit
  rost bo'lib qolishi uchun. Import bitta chaqiruv.

### Birinchi profil: yangi install o'zini nima deb ataydi
- ⚠️ **«My Restaurant» — o'rnatuvchining so'zi edi, va u hech kim ko'rib
  o'tkazib yuboradigan joyda turmasdi.** Brend ham, filial ham shu profildan
  yaratiladi (`repository.EnsureBrandAndBranch`), ya'ni o'sha so'z saytning
  sarlavhasiga, chekning birinchi qatoriga, Telegram xabarlariga va kassa
  qog'oziga ko'chardi — va do'konga «restoran» derdi. Hech qayerda xato
  chiqmaydi: hamma ekran ishlaydi, nom esa noto'g'ri.
- Endi konsol nomni yuboradi: `BRAND_NAME` (tenant `Name`'i), `BUSINESS_TYPE`
  bilan bir qatorda va **bir xil qoida bilan** — har boot'da yuboriladi,
  **profil yaratilayotgan** boot'da o'qiladi. Konsolda keyin qayta nomlangan
  mijoz o'z panelida yozgan nomini saqlab qoladi (chekida shu turibdi).
- ⚠️ **Nomsiz kelganda ham restoran emas.** Qo'lda yoki eski konsol bilan
  yaratilgan tenant baribir biror nom bilan boshlanishi kerak: do'kon uchun
  «Do'kon», restoran uchun «Restoran». «My Restaurant» esa do'konlarning
  **hammasi** uchun noto'g'ri.
- ⚠️ **Do'kon restoranning menyusini olmaydi, lekin bo'sh ham qolmaydi.**
  Dorixonadagi 48 ta taom «hali kiritilmagan» ham, «ishlamayapti» ham emas —
  **birovning do'koni** deb o'qiladi, va egasining birinchi ishi ularni
  bittalab o'chirish bo'lardi. Bo'sh jadval esa topshirilgan kuni
  «ishlamayapti» deb o'qiladi. Shuning uchun har bir do'kon turiga o'z
  namunasi bor (`internal/seed/shop.go`): oziq-ovqat, dorixona, kiyim, gul.
- ⚠️ **Namuna qisqa — o'n-o'n ikki qator, ellik emas.** Bu kassa, yorliq va
  qoldiq ekranlari ishlashini ko'rsatish uchun, «boshlang'ich katalog» emas:
  do'konning haqiqiy ro'yxati o'z shtrix-kodlaridan keladi, va qolib ketgan
  har bir namuna qatori keyin kimdir topib o'chiradigan qator. Restoranning
  menyusi uzun, chunki u **tahrirlanadi**; do'konniki skanerlanadi.
- ⚠️ **Har bir namuna qatori panelning o'z kodidan o'tadi**
  (`repository.SyncProductStock`). O'zini sotadigan tovarga ombor qatori va bir
  qatorli karta kerak; to'g'ridan-to'g'ri Mongo'ga yozilgani **yarim tovar**
  bo'lardi — sotib bo'ladi, sanab bo'lmaydi, va bu birinchi kundanoq,
  namunaviy ma'lumotda. Shu sababli funksiya `handlers` dan `repository` ga
  ko'chirildi: bitta amal, ikkita chaqiruvchi. `InsertMany` ishlatilmaydi —
  har qatorga ombor qatorining id'si **yozib qaytariladi**.
- ⚠️ **Rasm yo'q.** Seeddagi suratlar — taomlar; do'kon javonini biz
  suratga olmaymiz. Rasmsiz kartochka do'kon ro'yxatida odatiy, birovning
  palovi esa paratsetamol qutisi ustida — yo'q.
- ⚠️ **Kiyim namunasida o'lchamlar yo'q**, garchi butik ularni sotsa ham:
  variantlar menyu ekranidagi tugma bilan **do'konning o'z** o'lchamlaridan
  yaratiladi, tayyor kelgan o'n ikki variant esa o'chiriladigan o'n ikki qator.
- ⚠️ **Gulchida tayyor buket yo'q**, poya va o'ram bor: buket — shu qatorlar
  ustidagi texkarta, va uni do'kon o'zi sotib oladigan gullardan yozadi. Namuna
  buket bu do'konda yo'q poyalarga ishora qilardi, ya'ni tannarxi gul do'koni
  aynan shu uchun sotib oladigan ekranda noto'g'ri chiqardi.

### Namuna menyu (seed)
- `backend/internal/seed/menu.go` — 7 kategoriya, 48 taom (rasmlari bilan).
  Rasmlar `internal/seed/assets/*.jpg` da, Go `embed` orqali binarda; birinchi
  ishga tushishda `UPLOAD_DIR/seed/` ga yoziladi.
- Seed faqat **bo'sh bazada** ishlaydi — mavjud menyu hech qachon o'zgarmaydi.
  Yangi mijozga deploy qilganda menyu shu namunadan boshlanadi va admin
  panelda tahrirlanadi.

**`cmd/demodata` — bir oylik hayot (menyu emas)**
- Menyu bor bazaga **jonli ma'lumot** yozadi: buyurtmalar (har holatda), zaldagi
  ochiq cheklar, oshxona ekranidagi cheklar, kassa smenasi, xodimlar va
  davomat, kuryerlar, mijozlar, ombor (masalliq, texkarta, kirim, chiqim,
  inventarizatsiya), bronlar, fikrlar, tashriflar.
  `go run ./cmd/demodata -db demo -wipe`
- ⚠️ **Nima uchun bor**: marketing screenshotlari (`docs/LANDING_REDESIGN.md`
  §5.1). **Bo'sh ekran — ishlamayotgan mahsulotning surati**: qatorsiz jadval
  va ustunsiz grafik "hali ma'lumot yo'q" emas, "bu ishlamaydi" deb o'qiladi —
  va aynan restoran sotib olish haqida qaror qilayotgan sahifada.
- ⚠️ **`-seed` soatdan emas, qat'iy standart qiymatdan**: screenshot qayta
  olinadi (sarlavha siljidi, tema almashdi), va ikkinchi kadr birinchisi bilan
  **bir xil restoranni** ko'rsatishi shart. Soatdan urug'langan generator har
  qayta olishda boshqa kunlik tushum beradi va sahifa bitta dashboard uchun
  ikki xil daromad da'vo qiladi.
- ⚠️ **`-wipe` butun kolleksiyalarni bo'shatadi**, "shu asbob yozgan qatorlar"ni
  emas — hech nima belgilanmagan, va har hujjatga belgi maydoni qo'yish
  modellarda yo'q maydon bo'lardi. Shuning uchun `-db` **aniq yozilishi** va
  baza nomi qo'lda tasdiqlanishi talab qilinadi. Faqat sinov bazasida.
- ⚠️ **Ombor deliveries buyurtmalardan **keyin** yoziladi**: xarid miqdori
  oyning haqiqiy sarfidan hisoblanadi. "Ishonarli ko'rinadigan" konstanta bilan
  olinganda javonda 290 kg qo'y go'shti turadi va bitta masalliq **minusga**
  tushadi — ikkalasi bir ekranda, ya'ni mehmon birinchi tekshiradigan raqam yo
  bema'ni, yo qizil.
- ⚠️ **Texkarta porsiyadan hisoblanadi, tannarxdan emas.** Birinchi variant
  tannarxni maqsad qilib olib, narxning uchdan birini qatorlarga bo'lardi — va
  arzon masalliqda bu **bitta porsiyaga besh kilo sabzi** so'raydi. Hech narsa
  xato bermaydi: karta saqlanadi, tannarx normal ko'rinadi, xato esa uch ekran
  narida — 349 kg sabzi turgan ombor bo'lib chiqadi. Asosiy masalliq esa
  taomning **narx darajasiga** qarab tanlanadi: guruchga qurilgan 145 000 so'mlik
  steyk hech qanday porsiyada ishonarli tannarxga chiqa olmaydi.
- ⚠️ **Zaldagi eski cheklar oshxona ekranida turmaydi.** `readyAt` qo'yilmasa
  har ochiq chek KDS'ga tushadi va oltitasi 70/60/50 daqiqalik qizil bo'lib
  ekranning tepasini egallaydi — qulab tushayotgan oshxona surati. Faqat eng
  yangi ikki stol pishirilmoqda; qolganlari **berilgan, lekin to'lanmagan** —
  ochiq chek aslida shuni bildiradi.
- ⚠️ **`subscription` hujjatini ham shu asbob yozadi** (odatda uni Keel konsoli
  yozadi): modullar handler darajasida yopiq (`requireModule`), ya'ni obunasiz
  bazada ombor ekranlari "bu bo'lim tarifingizga kirmaydi" deb chiqadi — bo'sh
  install haqida rost gap, va mahsulot haqida yolg'on screenshot.

### Yuk: nima siqiladi, nima keshlanadi

2026-09-03 dagi yuk testi (`docs/LOAD_TEST_2026-09-03.md`) bir savolga aniq
javob berdi: **to'siq mongod** (cho'qqi 139% CPU ≈ 1.4 yadro), Go backendlari
esa hammasi birga 10–20%. Ya'ni sekinlik ilova kodida emas — **bir xil savolni
qayta-qayta bazadan so'rashda**. Va u haqiqatan bir xil savol: mehmonning
to'qqizta so'rovidan **beshtasi** (`/restaurant`, `/menu`, `/categories`,
`/promotions`, `/payment-methods`) ega haftada bir-ikki marta o'zgartiradigan
ma'lumot uchun.

**Kesh backendning o'zida, chekkada emas.** Sabab — kalit kimniki. Bitta Keel
o'rnatmasi 12 restoranga xizmat qiladi, ya'ni chekkadagi kesh kalitida host
bo'lishi **shart**, aks holda bir restoranning menyusi boshqasining domenida
chiqadi (`caddy/pagecache.conf` ning boshidagi izoh — shu qoidaning uzun
varianti). Tenantning backendi esa o'z konteyneri: u **umuman** boshqa restoran
uchun javob bera olmaydi, demak kalit bu yo'nalishda xato bo'lishi mumkin emas.

- ⚠️ **TTL qisqa (10–30 s) va invalidatsiya ataylab yo'q.** Har bir admin
  yozuvidan xabar qilinishi kerak bo'lgan kesh — bir kuni **xabar qilinmaydigan**
  kesh, va alomati "ega narxni o'zgartirdi, sayt eski narxni ko'rsatyapti", ya'ni
  **saqlash ishlamagandek** ko'rinadi. TTL esa hech kim eslamasa ham tugaydi.
  Menyu va aksiyalar 10 s (mehmon aynan shularga amal qilmoqchi), profil,
  kategoriya, brend, to'lov usullari 30 s.
- ⚠️ **Tokenli so'rov hech qachon keshdan javob olmaydi** (`Authorization`
  sarlavhasi bor bo'lsa o'tkazib yuboriladi), `?raw=1` va `?preview=` ham. Panel,
  kassa va ishchi ilovalari xuddi shu marshrutlarni o'qiydi — va aynan ular uchun
  30 soniyalik eskilik "saqlanmadi" bo'lib o'qiladi. Trafikda ular arzimas ulush,
  ya'ni bundan hech nima yo'qolmaydi.
- ⚠️ **Bir vaqtda kelgan so'rovlar bitta so'rovga birlashadi** (`pubEntry.ready`
  kanali): sovuq keshga yuzta so'rov kelsa — bitta so'rov va 99 ta kutuvchi.
  Nginx'dagi `proxy_cache_lock` bilan bir dars: burst **ishga ko'payib
  ketmasligi** kerak.
- ⚠️ **Xaritaning o'lchami cheklangan (512 ta yozuv).** Kalitda begona odam
  yozadigan query string bor (`?brand=aaa`, `?brand=aab`, …). Chegaradan keyin
  hech nima saqlanmaydi va so'rov shu fayl paydo bo'lishidan oldingidek mongoga
  boradi — **eski xatti-harakat = to'lib qolgandagi xatti-harakat**.
- ⚠️ **`Set-Cookie` bor javob va 200 bo'lmagan javob saqlanmaydi**, va ularga
  `Cache-Control` ham qo'yilmaydi (`ccWriter` sarlavhani `WriteHeader` da,
  status ma'lum bo'lgandan keyin qo'yadi). Keshlangan `Set-Cookie` — bir
  mehmonning sessiyasini keyingisiga berish; keshlangan 500 esa bir lahzalik
  nosozlikni butun oyna davomida saytda ushlab turish.
- ⚠️ **Til kalitda.** Bu javoblar server yozadigan gaplarni olib yuradi;
  faqat yo'l bo'yicha keshlansa ruscha mehmonga oldingi o'zbek mehmonining
  javobi ketadi. Brend va filial esa allaqachon query string ichida (cookie emas)
  — shuning uchun kalitda alohida nomlanmaydi.
- ⚠️ **`recorder` `Lang()` metodini beradi.** Handler'ga haqiqiy writer o'rniga
  shu beriladi, haqiqiysi esa `httpx.Error` / `httpx.LangOf` so'roq qiladigan
  `langWriter`. Metodsiz — har keshlangan marshrutdagi har gap jimgina o'zbekcha
  qoladi (`middleware/lang.go` boshidagi tuzoq, pastdan qaytadan kiritilgan).

**Siqish chekkada** (`control/internal/caddy` → `encode zstd gzip`, **har bir
site blokida**, chunki Caddy'ning global bloki javobga tegadigan direktiva
qabul qilmaydi). O'lchangan: `/api/v1/menu` 30 634 → 6 724 bayt = **4.6
barobar**, va test paytida `content-encoding` sarlavhasi **umuman yo'q** edi.

**Indekslar** — kesh oldidagi so'rovning o'zi ham arzon bo'lishi uchun (va
deploydan keyingi birinchi, kesh bo'sh paytdagi so'rov sekin bo'lmasligi uchun):
`menu_item(brandId, isAvailable, sortOrder)`, `category(brandId, isActive,
sortOrder)`, `promotion(brandId, isActive)`. ⚠️ **Filtr maydoni brend bilan sort
orasida turishi shart** — `(brandId, categoryId)` ga qarshi Mongo faqat brendni
ishlatadi va qolganini **xotirada** saralaydi; xotiradagi saralash 32 MB dan
oshsa **umuman yiqiladi**, bu esa fotosurat va uzun tavsifi bor menyu, ya'ni
g'ayrioddiy menyu emas.

⚠️ **`nofile` = 1024 — mehmonlar soniga qo'yilgan chegara.** Har ulanish va
mongo poolidagi har ulanish — bitta deskriptor. Tenant konteynerlari, Caddy va
pagecache 64000 ga ko'tarildi (mongo allaqachon shunday edi). Bu eng yomon
tarzda yiqiladi: `accept()` kunning eng gavjum daqiqasida "too many open files"
qaytaradi va rush tugashi bilan **o'zi tuzaladi** — kimdir qaraganda hammasi
joyida.
