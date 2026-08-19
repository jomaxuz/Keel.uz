# Ichimliklar markirovkasi (Asl Belgisi) — aniqlangani va ochiq savollar

> Holat: **tekshirildi (manbalar bilan), kod yozilmagan.** Bu fayl 4-ishning
> texnik shartlarini belgilaydi.

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

## Ochiq savollar (kod yozishdan oldin javob kerak)

1. ⚠️ **Onlayn buyurtma / yetkazib berish.** Qoida **chekka** bog'langan, xonaga
   emas: yetkazib berish buyurtmasi ham fiskal chek oladi, demak kod ham
   kerakdek ko'rinadi. Amalda ko'p joyda buyurtma yig'ilayotganda kassada
   skanerlanadi. **Buxgalter yoki Asl Belgisi qo'llab-quvvatlashidan tasdiq
   olish kerak** — javob "kerak emas" bo'lsa, kassa/zal bilan cheklaymiz.
2. **Barda ochilgan shisha** (stakanga quyilgan ichimlik) — shisha qachon
   muomaladan chiqariladi: ochilganda-mi yoki sotilganda? Menyuda "shisha" va
   "stakan" alohida taom bo'lsa, faqat birinchisiga bayroq qo'yiladi.
3. **Qaysi provayderlar** label ni qo'llab-quvvatlaydi (bizdagi fiskal
   adapterlar ro'yxati bo'yicha).
4. **Qaytarish/bekor qilish**: chek qaytarilganda kod muomalaga qaytadimi va
   buni fiskal tomon o'zi qiladimi.

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
