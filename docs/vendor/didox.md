# Didox — elektron hujjat aylanishi (EDI / ЭСФ)

**Manba:** <https://api-docs.didox.uz/ru/home> va uning bo'limlari
(`integrators-documents`, `integrators-property-documents`,
`integrators-catalogs`), hamda eski Postman to'plami
<https://documenter.getpostman.com/view/7157122/TVsrEUYF> (DIDOX-1C-INTEGRATION).
**O'qilgan sana:** 2026-09-07.

⚠️ Sayt — SPA: `curl` faqat bo'sh qobiqni qaytaradi, shuning uchun nusxasi shu
yerda (qarang `docs/vendor/README.md`).

---

## 1. Manzillar va kalitlar

| Nima | Qiymat |
|---|---|
| Test | `https://testapi3.didox.uz/` |
| Prod (hamkor) | `https://api-partners.didox.uz/` |
| Hamkor tokenini kim beradi | Didox account menejeri (Telegram `@Didox_account`, +998 50 122 05 18) |
| O'zgarishlar kanali | <https://t.me/didoxapiupdates> |

**Har bir so'rovda ikkita sarlavha:**

```
Partner-Authorization: <PARTNER_TOKEN>   // bizniki, platformaga bitta
user-key: <USER_TOKEN>                   // mijozniki, 360 daqiqa yashaydi
```

### Foydalanuvchi tokeni — uch yo'l

| Yo'l | Endpoint | Qachon |
|---|---|---|
| ЭЦП (E-IMZO) | `POST /v1/auth/{taxId}/token/{locale}` | kalit bilan kirish |
| **Parol** | `POST /v1/auth/{taxId}/password/{locale}` | ro'yxatdan o'tishda berilgan parol bilan — **ЭЦП shart emas** |
| Kompaniyaga kirish | `POST /v1/auth/company/{companyTaxId}/login/{locale}` | jismoniy shaxs o'z tokeni bilan kompaniyaga kiradi |

Token — UUID, **360 daqiqa** amal qiladi. Parol bilan kirishda xatolar:
`422 User not registered`, `422 Incorrect login`, `423` (bloklangan),
`429` (juda ko'p urinish).

---

## 2. Hujjat turlari (`docType`)

| Kod | Hujjat |
|---|---|
| `002` | **Aktsiz hisob-fakturasi (ЭСФ) — aktsiz siz**, eng ko'p ishlatiladigani |
| `001` | Hisob-faktura (eski, ro'yxatda hali uchraydi) |
| `021` | Hisob-faktura (qaytarish) |
| `008` / `081` | ЭСФ (FARM) va uning qaytarishi |
| `023` | Gibrid ЭСФ |
| `041` | TTN (yuk xati) |
| `005` | Bajarilgan ishlar dalolatnomasi (akt) |
| `006` / `062` | Ishonchnoma (eski/yangi) |
| `007` | Shartnoma (GNK) |
| `000` / `010` | Ixtiyoriy hujjat / ko'p tomonlama ixtiyoriy hujjat |
| `052` / `054` | Solishtirish dalolatnomasi / qabul-topshirish dalolatnomasi |

## 3. Holatlar (`status`)

| Kod | Ma'nosi |
|---|---|
| 0 | Qoralama (черновик) |
| 1 | Hamkorning imzosini kutmoqda (biz imzoladik) |
| 2 | **Bizning imzoni kutmoqda** (kiruvchi hujjat) |
| 3 | Imzolangan |
| 4 | Imzodan bosh tortilgan |
| 5 | O'chirilgan |
| 6 | Agent imzosini kutmoqda |
| 8 | Agent imzolagan |
| 40 | Haqiqiy emas |
| 60 | Hamkor: agent imzosini kutmoqda |
| 140 / 240 | TTN: mas'ul shaxs qabul qilgan |
| 160 / 190 | TTN: qabul qiluvchiga yetkazilgan / yuk qaytarilgan |

`status` berilmasa ro'yxat 1, 2, 3, 4, 6, 8, 40 holatlarini qaytaradi.

---

## 4. Endpointlar

### O'qish

| Metod | Endpoint | Nima |
|---|---|---|
| GET | `/v2/documents` | ro'yxat, **`page` va `limit` majburiy** |
| GET | `/v2/documents/statistics/all` | holat bo'yicha sanoq |
| GET | `/v1/documents/{id}` | to'liq ma'lumot: `data.json` (hujjat tanasi), `data.document` (meta), `data.toSign`, `data.relatedDocuments` |
| GET | `/v1/documents/view/{id}/html/{locale}` | chop etish shakli (HTML) — `user-key` bilan |
| GET | `/v1/documents/{id}/pdf/{locale}` | PDF |
| GET | `/v1/documents/{id}/archive` | ZIP: imzolar + PDF + JSON |

`/v2/documents` filtrlari: `owner` (1 — chiquvchi, 0 — **kiruvchi**; berilmasa
chiquvchi), `status`, `doctype` (vergul bilan bir nechta), `partner` (INN),
`name` (raqami), `docDateFromCreated` / `docDateToCreated`,
`dateFromUpdated` / `dateToUpdated`, `hasMarks`, `oneside` va boshqalar.

Ro'yxatdagi qator (muhim maydonlar): `doc_id` (32 belgi), `name` (hujjat
raqami), `doc_date`, `doc_status`, `doctype`, `owner`, `partnerTin`,
`partnerCompany`, `total_sum`, `total_vat_sum`, `total_delivery_sum_with_vat`,
`has_vat`, `has_marks`, `contract_number`, `contract_date`, `created`,
`updated_date`, `branch_num`.

### Yozish

| Metod | Endpoint | Nima |
|---|---|---|
| POST | `/v1/documents/{docType}/create/{locale}` | qoralama yaratadi |
| POST | `/v1/documents/{id}/update/{docType}/{locale}` | **faqat qoralamani** (status 0) tahrirlaydi |
| POST | `/v1/documents/{id}/delete/draft` | qoralamani o'chiradi, **imzo shart emas** |
| POST | `/v1/documents/{id}/tosign` | `{"action":"accept|cancel|reject|…"}` → imzolanadigan ma'lumot yoki tayyor base64 imzo |
| POST | `/v1/documents/{id}/sign` | `{"signature":"<timeStampTokenB64>"}` — chiquvchini imzolash va yuborish; kiruvchini qabul qilish ham shu endpoint |
| POST | `/v1/documents/{id}/reject` | `{"signature":"…","comment":"…"}` — izoh 1-qadamdagi bilan **bir xil bo'lishi shart** |
| POST | `/v1/documents/{id}/delete` | yuborilgan chiquvchini bekor qilish (avval `tosign` `cancel`) |
| POST | `/v1/dsvs/signature/join` | ikki imzoni birlashtirish (kiruvchini qabul qilishda) |

⚠️ **Har bir imzo — E-IMZO kaliti bilan brauzerda yasaladi**: kalitlar
ro'yxati → `keyId` → base64 ustiga imzo → timestamp ilova qilinadi →
`timeStampTokenB64` yuboriladi. Server tomonda kalit yo'q, ya'ni imzolashni
biz mijoz o'rniga qila olmaymiz. Birinchi imzolashdan oldin foydalanuvchida
**oferta imzolangan** bo'lishi kerak.

### Kataloglar

`GET /v1/banks/all`, `/v1/measures/all` (o'lchov birliklari, `Accept-Language`),
`/v1/regions/all`, `/v1/districts/all`.

---

## 5. ЭСФ (002) JSON — tanasi

⚠️ **Qabul qiluvchi tomon strukturani qat'iy tekshiradi** (roumingga
qo'yiladigan talab):
- **Ortiqcha maydon yuborib bo'lmaydi** (jumladan `WithoutExcise`, `Expansion`,
  `Id`, `MeasureId`, `SellerDepartmentId`, `BuyerDepartmentId`).
- Ishlatilmaydigan obyekt **butunlay `null`**, bo'sh `{}` emas
  (`FacturaRentDoc`, `OldFacturaDoc`, `ItemReleasedDoc`,
  `FacturaInvestmentObjectDoc`, `FacturaEmpowermentDoc`, `ForeignCompany`).
- `Count` — 6 xonagacha, qolgan sonlar 2 xonagacha; **nuqta**, razryad ajratgich
  yo'q.
- Sanalar **faqat `yyyy-MM-dd`** (vaqtsiz, mintaqasiz).

```jsonc
{
  "Version": 1,
  "WaybillLocalIds": [],
  "HasMarking": false,
  "HasRent": false,
  "FacturaRentDoc": null,
  "FacturaType": 0,                       // 0 — standart
  "ProductList": {
    "HasCommittent": false,
    "HasLgota": false,
    "Tin": "310529901",                   // sotuvchi INN
    "HideReportCommittent": false,
    "HasExcise": false,
    "HasVat": true,
    "Products": [{
      "OrdNo": 1,
      "LgotaId": null,
      "CommittentName": "", "CommittentTin": "",
      "CommittentVatRegCode": "", "CommittentVatRegStatus": null,
      "Name": "Tovar nomi",
      "CatalogCode": "10112008001000001",  // ИКПУ
      "CatalogName": "ИКПУ nomi",
      "Marks": null,
      "Barcode": "",
      "PackageCode": "1209782",            // qadoq kodi
      "PackageName": "dona",
      "Count": 1,
      "Summa": "10000.00",                 // birlik narxi
      "DeliverySum": "10000.00",           // Count * Summa
      "VatRate": "12",
      "VatSum": "1200.00",
      "ExciseRate": 0, "ExciseSum": 0,
      "DeliverySumWithVat": "11200.00",
      "WithoutVat": false,                 // faqat HasVat=true bo'lganda
      "LgotaType": null, "LgotaName": null, "LgotaVatSum": null,
      "WarehouseId": null,
      "Origin": 3
    }]
  },
  "FacturaDoc":  { "FacturaNo": "SF-0000001", "FacturaDate": "2026-08-04" },
  "ContractDoc": { "ContractNo": "D-000123", "ContractDate": "2026-03-26" },
  "ContractId": null,
  "LotId": "",
  "OldFacturaDoc": null,
  "SellerTin": "310529901",
  "Seller": {
    "Name": "\"KEEL\" MCHJ", "BranchCode": "", "BranchName": "",
    "VatRegCode": "326080220838", "Account": "20208000905656222001",
    "BankId": "00401", "Address": "…", "Director": "…", "Accountant": "…",
    "VatRegStatus": 20
  },
  "ItemReleasedDoc": null,
  "BuyerTin": "302936161",
  "Buyer": { /* Seller bilan bir xil shakl; bir tomonlama ЭСФ'da null */ },
  "FacturaInvestmentObjectDoc": null,
  "FacturaEmpowermentDoc": null,
  "ForeignCompany": null
}
```

Shartnomani ЭСФ bilan bog'lash uchun JSON'ga `"didoxcontractid": ""` qo'shiladi
(Didox'ning xizmat maydoni, hujjat tanasiga tushmaydi).

---

## 6. Keel bu API'dan nimani ishlatadi

| Keel | Didox |
|---|---|
| Kiruvchi hujjatlar ro'yxati | `GET /v2/documents?owner=0&doctype=002,008,001` |
| Hujjat tafsiloti va tovarlari | `GET /v1/documents/{id}` → `data.json.ProductList.Products` |
| Kirim (`purchase`) yasash | tafsilotdagi tovarlar → mavjud masalliq/tovarga bog'lanadi |
| Chop etish shakli | `GET /v1/documents/view/{id}/html/{locale}` |
| Chiquvchi ЭСФ qoralamasi | `POST /v1/documents/002/create/{locale}` |
| Imzolash | **qilinmaydi** — E-IMZO mijozning kompyuterida; panel didox.uz'ga yo'naltiradi |

⚠️ **Imzo va rad etish ataylab qo'shilmadi**: ikkalasi ham E-IMZO kaliti bilan
brauzerda yasalgan `timeStampTokenB64` ni talab qiladi, serverda esa kalit yo'q
va bo'lmasligi kerak. "Qabul qildim" tugmasi imzolamasdan holatni o'zgartirsa,
u yolg'on tugma bo'lardi — hujjat Didox'da imzolanmagan qoladi va soliq
hisobotida ko'rinmaydi.
