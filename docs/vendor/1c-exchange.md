# 1C — "Обмен с сайтом" (CommerceML 2) protokoli

**Manba:** <https://v8.1c.ru/tekhnologii/obmen-dannymi-i-integratsiya/standarty-i-formaty/protokol-obmena-s-saytom/>
**O'qilgan sana:** 2026-09-07.

⚠️ Bu — **1C tomonidan ochiq e'lon qilingan** protokol: har bir tipik
konfiguratsiya ("1С:Бухгалтерия", "Управление торговлей", O'zbekiston uchun
mahalliylashtirilganlari ham) shu bo'yicha sayt bilan almashadi. Ya'ni mijozning
buxgalteri hech nima yozmaydi — 1C'da "Обмен с сайтом" sozlamasiga manzil,
login va parol kiritadi.

## Asosiy qoida

**Almashinuvni doim 1C boshlaydi.** Sayt hech qachon 1C'ga o'zi murojaat
qilmaydi (uni ko'pincha internetdan ko'rib ham bo'lmaydi). Har bir qadam —
bitta HTTP GET, javob **oddiy matn**, birinchi qatori `success`, `progress`
yoki `failure`.

## Qadamlar (`type` va `mode`)

```
GET <manzil>?type=catalog&mode=checkauth
    → success\n<cookie nomi>\n<cookie qiymati>
       Basic auth bilan keladi; keyingi so'rovlar shu cookie bilan.

GET <manzil>?type=catalog&mode=init
    → zip=no
      file_limit=<bayt>

POST <manzil>?type=catalog&mode=file&filename=import.xml   (tanasi — fayl)
    → success

GET <manzil>?type=catalog&mode=import&filename=import.xml
    → progress   (o'sha so'rovni qaytadan yuborish kerak)
    → success    (tugadi)
    → failure\n<sabab>

GET <manzil>?type=sale&mode=checkauth        // buyurtmalarni olish
GET <manzil>?type=sale&mode=init
GET <manzil>?type=sale&mode=query            // javob — CommerceML 2 XML
GET <manzil>?type=sale&mode=success          // 1C oldi, belgilab qo'yish mumkin
```

`type` faqat ikkita: `catalog` (nomenklatura, narxlar, qoldiqlar) va `sale`
(hujjatlar/buyurtmalar). Fayllar **CommerceML 2**, UTF-8.

## CommerceML 2 dan bizga keraklisi

- `import.xml` → `КоммерческаяИнформация / Каталог / Товары / Товар`:
  `Ид`, `Штрихкод`, `Наименование`, `БазоваяЕдиница`, `Группы`,
  `ЗначенияРеквизитов`.
- `offers.xml` → `ПакетПредложений / Предложения / Предложение`: `Ид`,
  `Цены/Цена/ЦенаЗаЕдиницу`, `Количество`.
- `sale` javobi → `КоммерческаяИнформация / Документ`: `Ид`, `Номер`, `Дата`,
  `ХозОперация`, `Роль`, `Валюта`, `Сумма`, `Контрагенты`, `Товары/Товар`
  (`Ид`, `Наименование`, `БазоваяЕдиница`, `ЦенаЗаЕдиницу`, `Количество`,
  `Сумма`), `ЗначенияРеквизитов`.

⚠️ **Element nomlari ruscha va aynan shunday yoziladi** — 1C ularni matn
bo'yicha o'qiydi. Bitta harf xato bo'lsa 1C hujjatni jimgina tashlab ketadi.

## Keel qanday ishlatadi

| Yo'nalish | `type` | Nima |
|---|---|---|
| 1C → Keel | `catalog` | nomenklatura va narxlar: yangi tovar/masalliq yaratiladi, mavjudining nomi/narxi yangilanadi |
| Keel → 1C | `sale` | davr uchun sotuv hujjatlari (kassa cheklari va onlayn buyurtmalar) va kirimlar |

Autentifikatsiya — **Basic auth**, login va parol paneldan beriladi
(`/admin/1c`). Cookie — bir martalik sessiya markeri; biz uni imzolangan qiymat
sifatida beramiz va keyingi so'rovlarda tekshiramiz.
