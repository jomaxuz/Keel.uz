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
- Konsol tabida **rol tekshiruvi yo'q**, yonidagilardan farqli: yordam — kim
  stolda bo'lsa, o'sha javob beradigan ekran.
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
- **O'lcham URL'da so'raladi** (`?w=300|600|1200`), yuklashda ikkinchi fayl nomi
  yasalmaydi. Sabab: yuklash vaqtidagi variant **faqat keyin** yuklangan
  rasmlarga yordam berardi, diskdagi hamma narsa esa to'liq hajmda qolardi —
  yoki sayt mavjud bo'lmagan `-600` faylini so'rab, kartochkani buzardi.
  So'rov bo'yicha esa eski va yangi rasm birinchi so'rovdan bir xil ishlaydi.
- Hosila fayllar **diskda keshlanadi** (`uploads/.thumb/<w>/…`), ya'ni har rasm
  har o'lcham uchun bir marta o'lchanadi. Yozish **atomik** (tmp + rename):
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
- Takrorlanuvchi klasslar `globals.css` `@layer components` da:
  `.btn / .btn-primary / .btn-ghost / .btn-dark`, `.card`, `.badge*`, `.chip`,
  `.eyebrow`, `.section-title`, `.input`, `.container-page`.
- Shriftlar: **Inter** (matn) + **Playfair Display** (sarlavha, narx) —
  `next/font/google`, `font-sans` / `font-display`.

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
  Docker image'da barcha `cmd/*` binarlari bor: `/app/adminreset`, `/app/seedmenu`,
  `/app/paytest`.

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
