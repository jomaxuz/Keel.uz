# Keel Owner — restoran egasi uchun ilova (Android + iOS)

Expo (SDK 57, RN 0.86, React 19). Panel hisobi bilan kiriladi (`admin_token`),
filial linzasi paneldagidek ishlaydi.

## ⚠️ Bu — telefondagi panel emas

Panel — odam **o'tirib qaror qabul qiladigan** joy: menyu, narx, grafik,
kampaniya. Ular klaviatura ishi va o'sha yerda qoladi. Ega telefon bilan
**kuzatadi va javob qaytaradi**: bugun qanday ketyapti, hozir nima talab
qilinyapti, mana bu buyurtmani tasdiqla, 6-stoldan nega 400 000 olib
tashlandi. Har bir ekran shulardan bittasiga uch soniyada javob beradi.

**Ataylab yo'q**: menyu tahriri, sozlamalar, CRM kampaniyalari, ombor
hujjatlari, texkarta. Sababi qiyinligi emas — telefonda ular **yomon**
bajariladi, va svetofor oldida narx o'zgartira oladigan ekran oxir-oqibat
o'zgartiradi.

## Ekranlar

- **Bugun** — tushum (katta raqam), kecha bilan taqqoslash, buyurtmalar,
  o'rtacha chek, yetkazilgan/bekor qilingan. Yuqorida filial linzasi (bir
  filialli restoran uni umuman ko'rmaydi).
  ⚠️ Raqam yonida kontekst bo'lmasa — u ma'lumot emas: «12 400 000» o'zi
  yaxshi ham, yomon ham emas.
- **Diqqat** — ikkita ro'yxat, va ular bir xil emas: **navbat** (tasdiqlanmagan
  buyurtma, bron, kassa qabul qilmagan, chop etilmagan chek) va **sodir
  bo'lgan** (katta chegirma, hisobdan keyin olib tashlash, kassa kamomadi).
  Birinchisi bajariladi, ikkinchisi so'raladi.
  ⚠️ Ekran **hukm chiqarmaydi**: bu hodisalarning har birining oddiy izohi
  bor, va xulosa chiqargan ekran shu qadar tez-tez xato bo'ladiki, uni
  o'chirib qo'yishadi.
- **Buyurtmalar** — jonli ro'yxat, **tasdiqlash**, **bekor qilish** (sabab
  majburiy — mijoz uni kuzatuv sahifasida o'qiydi) va qo'ng'iroq. Boshqa hech
  nima: kuryer, chegirma, manzil tuzatish — bular panelda.
- **Fikrlar** — mehmonlar bahosi, **javobsizlaridan** ochiladi. Har birida
  qo'ng'iroq va «javob berdim» (nima qilganingiz yoziladi — server ham buni
  talab qiladi).
  ⚠️ **Saytga chiqarish paneldа qoladi**: u mehmonning ismini va so'zlarini
  internetga qo'yadi, bu esa svetofor oldida emas, o'tirib qabul qilinadigan
  qaror. Bu yerdagi ish — bugun kechqurun qo'ng'iroq qilish.
- **Hisobot** — bugun / hafta / oy: tushum, o'rtacha chek, ko'p sotilganlar,
  xodimlar soati va to'lanishi kerak bo'lgan summa, kam qolgan mahsulotlar.
  Faqat o'qish uchun.

### «Bugun» ekranidagi ikki qo'shimcha
- **Ertalabki brifing** — raqamlarning **tagida**: bu ekran kuniga yigirma marta
  tushum uchun ochiladi va kuniga bir marta brifing uchun o'qiladi. Alohida
  so'raladi (`/admin/insights`), ya'ni tarifga kirmagan javob bugungi tushumni
  o'zi bilan tortib tushirmaydi. ⚠️ Til **so'rovda** yuboriladi: telefonda
  cookie yo'q, va server aks holda hammasini o'zbekcha yozardi.
- **Kim ishda** — bitta qator (nechta odam ishda, nechtasi kelmagan), ro'yxat
  bosilganda. ⚠️ «Kelmagan» ni ilova hisoblamaydi — serverning `todayStatus` i
  o'qiladi: grafik, dam kuni va kechada tugagan smena bu yerda ikkinchi ta'rif
  bo'lib ajrardi. Vaqtlar ham serverdan `"HH:MM"` bo'lib keladi, timestampdan
  kesilmaydi (mintaqa tuzog'i).

### Sozlamalar → Obuna
Tarif, oylik summa va **sana + sanoq** — «obuna faol» bayrog'i emas (u yarim
tunda hech kim qaramaganda eskiradi). ⚠️ **Ogohlantirmaydi**: kassa oxirgi
haftada aytadi, panelda to'liq kartochka bor. Nol summa — «alohida kelishilgan»,
«bepul» emas.

## Bildirishnomalar

⚠️ **Bu ilova mavjud bo'lishining eng katta sababi.** Loss alertlar — hisobdan
keyin olib tashlangan taom, kassadagi kamomad, katta chegirma — hozirgacha
**faqat Telegram** orqali ketardi (`sendToOwners`), ya'ni Telegram ulamagan
restoran ularning **birortasini ham** ko'rmasdi. Hech nima buzilmagan va hech
nima buni aytmagan: yozuvlar hech kim ochmaydigan ro'yxatda to'planardi.

- Kanal `owner` (boshqa uchtasidan alohida: Android'da kanalni foydalanuvchi
  o'chiradi, va pulga oid xabar oshxona chimesi bilan birga o'chib ketmasligi
  kerak).
- ⚠️ **Push Telegram bilan yonma-yon yuboriladi, o'rniga emas** — va Telegram
  xatosining ichida emas. Ikki mustaqil kanalni `else if` bilan bog'lash bu
  kodbazada bir marta jimgina ishlamay qolishga sabab bo'lgan
  (`CLAUDE.md` dagi Caddy tuzog'i).
- ⚠️ **Loss alertlar faqat egalarga** (`ownersOnly`): menejer — bu xabarlar
  *haqida* bo'lgan odamlardan biri. Telegram yo'li shu chiziqni boshidan
  chizgan, bu ham shuni chizadi.
- **Yangi buyurtma** esa egaga ham, o'sha filial menejeriga ham boradi.
- **Past baholi fikr** — xuddi loss alertlardagi kabi, u ham `sendFeedbackToGroup`
  orqali **faqat Telegram**da edi. Endi yonma-yon push ketadi. Faqat past baho
  (maqtov uchun jiringlagan kanal bir haftada o'chiriladi) va **menejerga ham**:
  bu xabar xodim haqida emas, va kechqurun qo'ng'iroq qila oladigan odam
  ko'pincha aynan u.
- **Kunlik yakun** — soat bo'yicha emas, **oxirgi kassa smenasi yopilganda**:
  tushum, chek soni, o'rtacha chek. Soat bo'yicha yuborilgan yakun bir
  restoranda yarim kunni, ikkinchisida ochiq kassani yig'adi. Sotuv bo'lmagan
  kun umuman jim, va bu xabar Telegramga ketmaydi.
- Matn har qurilmaning tilida quriladi (`admin_device.lang`).

## Firebase

⚠️ `google-services.json` ichida **`uz.keel.owner`** uchun ham yozuv bo'lishi
shart, aks holda Android buildi «No matching client found» bilan to'xtaydi.
FCM V1 kaliti EAS'da allaqachon ulangan (bitta service account to'rtala
loyihaga ham).

## Ishga tushirish va build

```bash
npm start
npx eas-cli build -p android --profile preview
```

## O'lchov asosi (1-sentabr 2026)

| O'lchov | Qiymat |
|---|---|
| JS bundle (Hermes) | **1.8 MB** |
| Modullar | 702 |
| `expo-doctor` | 21/21 |
