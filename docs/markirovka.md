# Ichimliklar markirovkasi (Asl Belgisi) — aniqlangani va ochiq savollar

> Holat: **yozildi (25-avgust 2026).** Kassa/zal uchun to'liq quvur ishlaydi:
> menyudagi bayroq → kassada skanerlash → chek qatorida saqlanish → fiskal
> chekning `label` maydoni. ⚠️ **Ikkita band hali ochiq va ikkalasi ham
> bizdan tashqarida** — pastdagi "Ochiq savollar" ga qarang.

## Qonun nima talab qiladi

- Tizim — **«Asl Belgisi»** milliy raqamli markirovka tizimi (operator:
  CRPT Turon). Kod turi — ⚠️ **DataMatrix**, QR **emas** (ikkalasi ham
  "kvadrat kod"ga o'xshaydi, lekin skaner sozlamasi va format boshqa).
- **Suv va alkogolsiz ichimliklar** chakana savdoda **2025-yil 1-martdan**
  majburiy: sotuv «Asl Belgisi»da aks etishi shart. Alyuminiy bankadagi
  ichimliklar bo'yicha ma'lumot uzatish 2025-yil 1-avgustdan.
- **Umumiy ovqatlanish (HoReCa) ham** shu qoidaga kiradi.
- Jarima: birinchi marta — chorak sof tushumining **2%**, bir yil ichida
  takrorlansa — **20%**.

## Mexanika (biz uchun eng muhim qism)

Chakana sotuvda "muomaladan chiqarish" quyidagicha ketadi:

1. Sotuvchi kodni **skaner yoki kamera** bilan o'qiydi (skaner onlayn kassaga
   ulangan).
2. **Onlayn kassa kodni chekka qo'shadi** — fiskal chek qatoridagi **label**
   maydoni.
3. Chek **OFD** ga ketadi, OFD esa uni «Asl Belgisi» MATiga uzatadi.

⚠️ **Ya'ni bizga alohida "Asl Belgisi API" kerak emas** — kod **fiskal chek
ichida** ketadi. Bizda fiskal modul allaqachon bor (`internal/fiscal`,
`cmd/fiscalagent`), demak ish shu quvurga bitta maydon qo'shishdan boshlanadi.

## Bizda nima qilish kerak

1. **Menyuda bayroq**: taom "markirovkalanadi"mi (shishadagi suv, gazli
   ichimlik...). ⚠️ Bo'sh qiymat — "yo'q": mavjud har bir menyuda bu maydon yo'q.
2. **Kassa/zalda skaner**: pistolet skanerlar odatda **HID klaviatura** bo'lib
   ishlaydi — alohida drayver shart emas, kod "yozilgan matn" bo'lib keladi
   (oxirida Enter). Bunday taom chekka qo'shilganda **kod so'raladi**; kodsiz
   qatorni yopib bo'lmaydi.
3. **Kod qatorga saqlanadi** (`OrderItem.MarkCode`), va ⚠️ **bitta chekda ikki
   bir xil kod bo'lmasligi** kerak — validatsiya kassada ham, serverda ham.
4. **Fiskal payload**: label maydoni har markirovkalangan qator uchun.
   ⚠️ Aniqlanishi kerak: bizdagi fiskal provayderlar API'sida bu maydon qanday
   nomlanadi va formati (`internal/fiscal` adapterlari bo'yicha bittalab).
5. **Format tekshiruvi**: DataMatrix'dan kelgan satr GS1 ajratgichlari bilan
   keladi. Kamida uzunlik va prefiks tekshiruvi — noto'g'ri kod fiskal chekni
   rad ettiradi, va bu mehmon oldida bo'ladi.

## Nima yozildi (25-avgust 2026)

- `menu_item.marked` — panelda taom formasida, fiskal maydonlar yonida.
  ⚠️ **Bo'sh — "yo'q"**, ya'ni bu maydongacha yozilgan har bir menyu tegilmagan
  qoladi. Migratsiya kerak emas.
- `internal/marking` — kod nima bo'lishi mumkinligi. Skaner **HID klaviatura**,
  ya'ni matn bo'lib keladi: ajratgich uch xil yoziladi (`\x1d`, chop etiladigan
  `"\x1d"`, ASCII 232) va uchalasi ham bir xil chegara — firmware sababli rad
  etilgan kodni kassir tuzata olmaydi. ⚠️ `01` prefiksi tekshiruvi **DataMatrix'ni
  yonidagi EAN-13 dan ajratadi**: skaner qayerga qaratilsa o'shani o'qiydi.
- ⚠️ **Takror kod — bir qatorda ko'rinmaydigan nosozlik.** Pistolet chiyilladi,
  o'qigan-o'qimagani noaniq, yana skanerlandi — ikki qator bitta shishani
  muomaladan chiqaradi, ikkinchisi esa muomalada qoladi. Fiskal kassa bunday
  chekni **qabul qiladi**, ya'ni ushlanadigan yagona joy — bizniki
  (`CheckAll`, butun chek bo'yicha).
- ⚠️ **Markirovkalangan qator hech qachon birlashmaydi** (kassada ham, oflaynda
  ham): ikki shisha — ikki kod, bitta kod ostidagi ikkita esa bittasini
  hujjatlashtirib ikkitasini beradi.
- **Ikkita darvoza**: qator qo'shilganda (kassir shishani qo'lida ushlab turadi,
  skaner ikkinchi qo'lida — skanerlash faqat shu payt bepul) va chek
  yopilganda (qator bayroqdan oldin qo'shilgan bo'lishi mumkin, va qonun
  chek haqida).
- **Fiskal chekda `label` maydoni** (`internal/fiscal` → ikkala adapter).
  ⚠️ `omitempty`: markirovkalanmagan taomda maydon **umuman ketmasligi** kerak —
  bo'sh satrni "markirovkalangan, kodi yo'q" deb o'qigan kassa chekni rad etadi,
  va bu mehmon oldida bo'ladi.
- Kassada qoidalar takrorlangan (`lib/marking.ts`) — ataylab: rad javobi shisha
  ushlab turgan odam qayta skanerlay oladigan joyda bo'lishi kerak.

## Ochiq savollar

⚠️ **Ikkitasi javob kutmoqda va ikkalasi ham kod bilan hal bo'lmaydi** —
javobni buxgalter yoki provayder beradi:

1. ⚠️ **Onlayn buyurtma / yetkazib berish.** Qoida **chekka** bog'langan, xonaga
   emas: yetkazib berish buyurtmasi ham fiskal chek oladi, demak kod ham
   kerakdek ko'rinadi. Amalda ko'p joyda buyurtma yig'ilayotganda kassada
   skanerlanadi. **Buxgalter yoki Asl Belgisi qo'llab-quvvatlashidan tasdiq
   kerak.** Hozircha yozilgani: **kassa va zal**, sayt/bot buyurtmasida kod
   so'ralmaydi. ⚠️ Bu **taxmin**, va tanlangan yo'nalish ataylab: kod
   so'ramaydigan sayt — kamchilik; har yetkazib berish buyurtmasida kod
   so'raydigan sayt — umuman berilmaydigan buyurtma.
2. ⚠️ **`label` maydonining nomi.** Davlat chek formatidan olingan
   (Asl Belgisi yordam markazi), **Multikassa va REGOS hujjatlari bo'yicha
   tasdiqlanmagan**. Har ikkala adapterda **bitta qatorda** yozilgan — tuzatish
   kerak bo'lsa ikkita qator, qidiruv emas.

Javob topilgan ikkitasi:

3. ✅ **Barda ochilgan shisha** — bayroq **mahsulotda**, ya'ni savol o'zi
   javob bo'ldi: butun sotiladigan shisha markirovkalanadi, stakanga quyilgani
   esa menyuda **boshqa taom** (boshqa narx bilan) va unga bayroq qo'yilmaydi.
   Restoranlar buni qog'ozda allaqachon shunday yuritadi.
4. ✅ **Qaytarish** — kod qatorda turadi, qaytarish cheki o'sha qatorlardan
   quriladi, ya'ni kod qaytarish hujjatiga **o'zi boradi**. Muomalaga qaytarishni
   fiskal tomon qiladi; bizda qo'shimcha ish yo'q.

## Manbalar

- Asl Belgisi yordam markazi — onlayn KKM va markirovka (kod chekning label
  maydonida, OFD orqali MATga):
  https://help.crpt-turon.uz/hc/ru/articles/4419966010257
- Spot.uz — 2025-yil 1-martdan majburiy, DataMatrix, jarimalar:
  https://www.spot.uz/ru/2025/03/27/sbg/
- CRPT Turon — skaner tanlash (DataMatrix o'qiy oladigan model kerak):
  https://help.crpt-turon.uz/hc/uz/articles/4419959937041
- Buxgalter.uz — PP-190 bo'yicha o'zgarishlar va HoReCa majburiyatlari:
  https://buxgalter.uz/publish/doc/text208507_9_glavnyh_izmeneniy_v_sfere_obyazatelnoy_markirovki_po_pp-190
