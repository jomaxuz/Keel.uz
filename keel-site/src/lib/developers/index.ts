// keel.uz/developers — the open API manual, in three languages.
//
// ⚠️ **Section ids are shared across the languages** (`#money`, `#signature`):
// a developer in Moscow sends a colleague in Tashkent a link to a heading, and
// it has to land on the same heading in Uzbek.
//
// ⚠️ **Prose is translated, the contract is not.** Field names, paths, headers,
// error codes and every code sample are English in all three languages — they
// are what a program sends and reads, and a translated field name is a field
// that does not exist. The samples live once, in `code.ts`.
//
// The published contract is `docs/open-api.md` in the repo; this page follows
// it. A change to the API is a change to both.

import type { Lang } from "@/lib/i18n/dict";
import * as C from "./code";

export type Block =
  | { kind: "p"; text: string }
  | { kind: "code"; code: string; label?: string }
  | { kind: "table"; head: string[]; rows: string[][] }
  | { kind: "list"; items: string[] }
  | { kind: "note"; text: string };

export interface Section {
  id: string;
  title: string;
  blocks: Block[];
}

export interface DevDoc {
  title: string;
  lead: string;
  description: string;
  ui: { onThisPage: string; copy: string; copied: string; version: string };
  sections: Section[];
}

const p = (text: string): Block => ({ kind: "p", text });
const code = (c: string, label?: string): Block => ({ kind: "code", code: c, label });
const table = (head: string[], rows: string[][]): Block => ({ kind: "table", head, rows });
const list = (items: string[]): Block => ({ kind: "list", items });
const note = (text: string): Block => ({ kind: "note", text });

// ---- Uzbek ----

const uz: DevDoc = {
  title: "Keel API — dasturchilar uchun",
  lead: "Restoraningizni CRM, buxgalteriya, AI xizmatlar va o'z ilovangiz bilan ulang: menyu, buyurtmalar, pul harakati va real vaqtdagi webhook'lar.",
  description:
    "Keel ochiq API hujjati: API kalitlar, menyu, buyurtmalar, pul daftari (buxgalteriya uchun), qoldiqlar, fiskal cheklar va imzolangan webhook'lar.",
  ui: { onThisPage: "Shu sahifada", copy: "Nusxa", copied: "Nusxalandi", version: "Versiya" },
  sections: [
    {
      id: "overview",
      title: "Umumiy",
      blocks: [
        p("Keel'dagi har bir restoran o'z serverida, o'z domenida ishlaydi. API ham restoranning o'z domenida:"),
        code(C.BASE),
        p("Aniq manzilni restoran egasi panelda ko'radi: **Sozlamalar → Integratsiyalar → API va webhook'lar**. Bir restoranning kaliti faqat o'sha restoranni ochadi."),
        p("API ikki yo'nalishda ishlaydi: sizning dasturingiz bizdan **so'raydi** (API kalit bilan), va biz buyurtma yoki pul o'zgarganda sizga **o'zimiz xabar beramiz** (webhook)."),
        note("`/v1` — va'da. Bu yerdagi yo'l, maydon, header va xato kodlari o'zgarmaydi; o'zgarish kerak bo'lsa, `/v2` yonida paydo bo'ladi. Obyektlarga yangi maydonlar qo'shilishi mumkin — bilmaganingizni e'tiborsiz qoldiring."),
      ],
    },
    {
      id: "quickstart",
      title: "Tez boshlash",
      blocks: [
        list([
          "Restoran egasi panelda kalit yaratadi va unga kerakli ruxsatni beradi. Kalit `keel_` bilan boshlanadi va **faqat bir marta** ko'rsatiladi.",
          "Ega sizga ikki narsa beradi: API manzili va kalit.",
          "Kalit to'g'riligini `/ping` bilan tekshiring:",
        ]),
        code(C.PING, "curl"),
        code(C.PING_RESPONSE, "200 OK"),
      ],
    },
    {
      id: "auth",
      title: "Autentifikatsiya va ruxsatlar",
      blocks: [
        p("Har so'rovda `Authorization: Bearer keel_…` header'i. Kalitda ruxsatlar bor:"),
        table(
          ["Ruxsat", "Nima beradi"],
          [
            ["`menu:read`", "`/branches/{id}/menu` — menyu va nima hozir sotuvda"],
            ["`orders:read`", "`/orders`, `/orders/{ref}` — buyurtmalar, **mijoz ismi va telefoni bilan**"],
            ["`finance:read`", "`/money`, `/money/daily`, `/balances`, `/fiscal` — pul, **mijoz ma'lumotisiz**"],
          ],
        ),
        p("`/ping` va `/branches` uchun istalgan faol kalit yetadi. Bekor qilingan kalit keyingi so'rovdayoq ishlamaydi."),
        note("Kalit faqat serverdan ishlatiladi. Uni brauzer yoki mobil ilovaga qo'ymang — u yerda uni har kim o'qiy oladi. Brauzerdan cross-origin so'rovlar ruxsat etilmaydi."),
        p("Cheklov: bir IP manzildan daqiqasiga 120 so'rov (undan keyin `429`)."),
      ],
    },
    {
      id: "conventions",
      title: "Qoidalar",
      blocks: [
        list([
          "Pul — **so'mda butun son** (UZS). Tiyin va kasr yo'q.",
          "Vaqt — RFC 3339, UTC (`2026-09-24T12:30:15.69Z`).",
          "Kun kerak bo'lsa `day` maydonini ishlating — u restoranning o'z kalendar kuni. `occurredAt` ni kesib olish Toshkent vaqti bilan 19:00 dan keyingi hammasini oldingi kunga qo'yadi.",
          "Id — 24 belgili hex. Yo'q id — bo'sh qator yoki umuman maydon yo'q, hech qachon `\"000000000000000000000000\"` emas.",
          "Ro'yxat doim massiv, hech qachon `null` emas.",
        ]),
      ],
    },
    {
      id: "errors",
      title: "Xatolar",
      blocks: [
        code(C.ERROR, "4xx / 5xx"),
        table(
          ["HTTP", "`code`"],
          [
            ["400", "`invalid_request`"],
            ["401", "`unauthorized`"],
            ["403", "`insufficient_scope`"],
            ["404", "`not_found`"],
            ["500", "`internal_error`"],
          ],
        ),
        p("Dasturda `code` bo'yicha qaror qiling; `message` odam uchun va o'zgarishi mumkin."),
      ],
    },
    {
      id: "menu",
      title: "Filiallar va menyu",
      blocks: [
        code(C.BRANCHES),
        code(C.MENU),
        p("`available: false` — restoranning o'z sayti ham buni sotmaydi: o'chirilgan, stop-listda, kombo'ning bir qismi tugagan yoki kunlik limit to'lgan."),
      ],
    },
    {
      id: "orders",
      title: "Buyurtmalar",
      blocks: [
        p("`GET /orders` — eng yangisi birinchi. Filtrlar:"),
        table(
          ["Parametr", ""],
          [
            ["`branchId`", "faqat shu filial"],
            ["`status`", "`pending`, `confirmed`, `preparing`, `on_the_way`, `delivered`, `cancelled`"],
            ["`createdFrom`, `createdTo`", "RFC 3339; boshi kiradi, oxiri kirmaydi"],
            ["`limit`", "1–100, standart 50"],
            ["`cursor`", "oldingi sahifaning `nextCursor` i; oxirgi sahifada `\"\"`"],
          ],
        ),
        code(C.ORDERS),
        p("`GET /orders/{ref}` — bitta buyurtma, raqami (`QVUU-R97U`) yoki id'si bo'yicha. Buyurtma obyekti:"),
        code(C.ORDER, "Order"),
        list([
          "`type`: `delivery`, `pickup`, `dinein` (zaldagi stol), `uzum_tezkor` (Uzum kuryeri olib ketadi).",
          "`address` faqat manzil bo'lsa keladi; `cancelReason`, `tableNumber`, `scheduledAt` bo'sh bo'lsa kelmaydi.",
          "`price` — bitta dona narxi, variantlar qo'shilgan.",
        ]),
      ],
    },
    {
      id: "money",
      title: "Pul daftari",
      blocks: [
        p("`GET /money` (`finance:read`) — restoranda bo'lgan **har bir pul harakati**, bitta ko'rinishda va **allaqachon tasniflangan**. Buxgalteriya xizmatlari uchun: summa, to'lov turi, yetkazib beruvchi va xodim — mijozning ismi yoki telefoni hech qachon yo'q."),
        table(
          ["Parametr", ""],
          [
            ["`from`, `to`", "majburiy; restoran kunlari (`2026-09-01`), ikkalasi ham kiradi, yoki RFC 3339. Ko'pi bilan 31 kun."],
            ["`branchId`", "faqat shu filial"],
          ],
        ),
        code(C.MONEY),
        note("Har pul foyda-zarar emas. Pul kassadan bankka o'tadi, bozorchiga avans beriladi, agregator restoran allaqachon topgan pulni o'tkazadi. Ularni tushum yoki xarajat deb qo'shish bir pulni ikki marta sanaydi. Shuning uchun **har yozuv o'zi nima ekanini aytadi**: `class` va `pnl`."),
        table(
          ["`class`", "`pnl`", "Nima"],
          [
            ["`revenue`", "✅", "To'langan sotuv, sotilgan kunida. Tafsiloti `sale` da."],
            ["`refund`", "✅", "Qaytarilgan pul, qaytarilgan kunida."],
            ["`cost`", "✅", "Yetkazib beruvchidan xarid, xarajat (ijara, kommunal, soliq), tashqi yetkazish xizmati."],
            ["`payroll`", "✅", "To'langan ish haqi — xodimlar va kuryerlar."],
            ["`commission`", "✅", "Agregator yoki ekvayring ushlab qolgan pul."],
            ["`transfer`", "❌", "Xuddi o'sha pul boshqa joyda: kassa → bank, agregator → bank, kuryer → kassa. **Tushum emas.**"],
            ["`advance`", "❌", "Xodimga xarid uchun berilgan naqd (podotchet). Xarajat u sotib olgan kirimda bo'ladi — **u allaqachon `cost`**."],
            ["`manual`", "❌", "Kassa yoki seyfga qo'lda qo'yilgan/olingan naqd, kassirning o'z `category` si bilan. Ko'pincha kirim yoki xarajat hujjatidagi o'sha pul."],
            ["`variance`", "❌", "Smena yopilganda kassa ortiqcha (`in`) yoki kam (`out`) chiqdi. Yo'qolgan pul, sarflangan emas."],
          ],
        ),
        p("Restoran panelidagi moliyaviy hisobot — `pnl: true` yozuvlarining yig'indisi, ya'ni `pnlNet`. Sizning davr raqamingiz undan farq qilsa — nimadir ikki marta sanalgan."),
        p("Chegirma va ballar alohida yozuv emas: pul harakatlanmagan. Ular sotuv ichida (`sale.discountTotal`, `sale.pointsSpent`); `amount` — haqiqatan olingan summa."),
        code(C.MONEY_ENTRY, "MoneyEntry"),
        list([
          "`amount` doim musbat; ishorani `direction` beradi (`in` / `out`, restoran tomonidan).",
          "`source`: `sale`, `refund`, `delivery_service`, `purchase`, `expense`, `salary`, `courier_pay`, `payout`, `collection`, `advance`, `cash_entry`, `safe_entry`, `courier_settlement`, `shift_variance`. Yangilari qo'shilishi mumkin — **qarorni `class` bo'yicha qiling**.",
          "Ixtiyoriy: `method`, `methodName`, `category`, `counterparty`, `from` / `to` (o'tkazmalar uchun: till, safe, bank, aggregator, courier, staff), `note`.",
          "`sale` (faqat `revenue`): `type`, `channel`, `subtotal`, `discountTotal`, `pointsSpent`, `deliveryFee`, `serviceCharge`.",
          "`paid` (faqat xarid): tovar kelgan kuni yoziladi; yetkazib beruvchiga to'langanda `paidAt`.",
        ]),
      ],
    },
    {
      id: "reread",
      title: "Davr hech qachon yopilmaydi — qayta o'qing",
      blocks: [
        p("Daftar so'ragan paytingizda restoran hujjatlaridan hisoblanadi. Hujjatlar tuzatiladi: ikki marta yozilgan xarajat o'chiriladi, kirim narxi tuzatiladi, qarz to'lanadi, pul bir haftadan keyin qaytariladi. Shuning uchun:"),
        list([
          "Davrni qayta oling va saqlaganingizni **almashtiring** — `id` bo'yicha moslang, yo'qolgan id'larni o'chiring. Qo'shib bormang.",
          "Har kuni kamida oxirgi 7 kunni, oy yopilgach esa butun oldingi oyni qayta o'qing — yoki `money.day_changed` webhook'ini ishlating.",
          "`id` barqaror: bir hujjat doim bir xil id beradi.",
        ]),
      ],
    },
    {
      id: "daily",
      title: "Kunlik yig'indilar",
      blocks: [
        p("`GET /money/daily` (`finance:read`) — daftar kun va filial bo'yicha yig'ilgan. Xuddi `/money` yozuvlarining o'zi, shuning uchun ikkalasi hech qachon farq qilmaydi. Parametrlar `/money` niki, davr 93 kungacha (chorak)."),
        code(C.DAILY),
      ],
    },
    {
      id: "balances",
      title: "Pul qayerda — qoldiqlar",
      blocks: [
        p("`GET /balances` (`finance:read`, ixtiyoriy `branchId`) — hozirgi holat. Egasining paneldagi «Pul qayerda» ekrani bilan bir xil raqamlar."),
        code(C.BALANCES),
        note("Umumiy jami **ataylab yo'q**. Naqdni bugun kechqurun, bankdagini shu hafta, agregatordagini esa ular hal qilganda sarflash mumkin. Ularni qo'shish hech narsa haqida rost bo'lmagan raqam beradi."),
        list([
          "`counted: true` — kimdir sanagan (bank ilovasidan o'qilgan qoldiq); eskiradi, `at` ga qarang. `counted: false` — hujjatlardan yig'ilgan; hujjat yetishmasa noto'g'ri bo'ladi.",
          "Bank — **oxirgi sanalgan qoldiq**, harakatlardan hisoblanmaydi: hisobga biz ko'rmaydigan pullar ham tushadi.",
          "`name` — restoranning o'z so'zi; qarorni `kind` bo'yicha qiling: `safe`, `drawer`, `courier`, `advance`, `bank`, `rail`.",
          "`payables.suppliers` — hali to'lanmagan kirimlar; `receivables.guestDebt` — hali to'lanmagan qarz sotuvlar.",
        ]),
      ],
    },
    {
      id: "fiscal",
      title: "Fiskal cheklar",
      blocks: [
        p("`GET /fiscal` (`finance:read`, ≤31 kun) — davr sotuvlari bo'yicha soliq qo'mitasiga yuborilgan cheklar **har holatda** (oy yopilishidan oldin aynan kutilayotgan va xato berganlari quviladi) va shu davrda yopilgan smenalarning Z-hisobotlari."),
        code(C.FISCAL),
        p("`kind` — `sale` yoki `refund`; `status` — `pending`, `filed` yoki `failed`; `error` nega yuborilmaganini aytadi."),
      ],
    },
    {
      id: "webhooks",
      title: "Webhook'lar",
      blocks: [
        p("Ega panelda `https://` manzilingizni qo'shadi va hodisalarni tanlaydi. U bir martalik **imzo kaliti**ni (`whsec_…`) oladi va sizga beradi."),
        table(
          ["Hodisa", "Qachon"],
          [
            ["`order.created`", "buyurtma berildi — sayt, ilova, operator, kassa, Uzum Tezkor"],
            ["`order.status_changed`", "buyurtma holati o'zgardi"],
            ["`money.day_changed`", "kunning pul raqamlari o'zgardi — pastga qarang"],
            ["`ping`", "ega panelda «Sinab ko'rish» ni bosdi"],
          ],
        ),
        code(C.WEBHOOK_REQUEST, "HTTP"),
        list([
          "`data.status` — **shu hodisa** haqidagi holat. `data.order` navbatga qo'yilgan paytdagi surat, shuning uchun ketma-ket tez hodisalarda `data.order.status` keyingisi bo'lishi mumkin.",
          "`createdAt` — o'zgarish bo'lgan vaqt. Oflayn ishlagan kassa savdoni keyin yuboradi, qilingan vaqti bilan.",
          "**10 soniya ichida** istalgan `2xx` bilan javob bering, ishni keyin qiling. Boshqa status, timeout yoki redirect — xato; redirect kuzatilmaydi.",
          "Xatodan keyin qayta urinish: 1 daq, 5 daq, 30 daq, 2 soat, 6 soat, 12 soat, 24 soat (8 urinish, taxminan ikki kun).",
          "Bir hodisa bir necha marta kelishi mumkin — `id` (`Keel-Event-Id`) bo'yicha takrorni tashlang.",
          "Hodisalar navbat bilan yuboriladi, lekin qayta urinish yangi hodisadan keyin kelishi mumkin — **`createdAt` bo'yicha tartiblang**, kelish tartibi bo'yicha emas.",
        ]),
      ],
    },
    {
      id: "money-webhook",
      title: "money.day_changed",
      blocks: [
        code(C.MONEY_EVENT),
        p("Bu **ishora, pul emas: shu kunni qayta o'qing** (`GET /money?from=<day>&to=<day>`) va saqlaganingizni almashtiring. Kunning raqamini o'zgartiradigan har narsada keladi — yangi sotuv, tuzatilgan kirim, o'chirilgan xarajat — oxirgi 35 kun ichida, 10 daqiqada bir tekshiriladi. `entries: 0` — kunning hamma yozuvi o'chirilgan."),
        p("Manzil qo'shilishidan oldingi kunlar uchun hech narsa yuborilmaydi — o'sha davrlarni bir marta o'zingiz o'qib oling."),
      ],
    },
    {
      id: "signature",
      title: "Imzoni tekshirish",
      blocks: [
        code(C.SIGNATURE_FORMULA),
        p("**Xom body baytlari** ishlatiladi — JSON'ni parse qilishdan oldin. Imzo mos kelmasa yoki `t` soatingizdan 5 daqiqadan ko'p farq qilsa — so'rovni rad eting. Vaqt imzo ichida bo'lgani uchun eski yetkazmani qayta yuborib bo'lmaydi."),
        code(C.TEST_VECTOR, "Test vektori"),
        code(C.VERIFY_NODE, "Node.js"),
        code(C.VERIFY_PYTHON, "Python"),
        code(C.VERIFY_PHP, "PHP"),
        p("Ega imzo kalitini almashtirsa, navbatda turgan yetkazmalar yangisi bilan imzolanadi."),
      ],
    },
  ],
};

// ---- Russian ----

const ru: DevDoc = {
  title: "Keel API — для разработчиков",
  lead: "Подключите ресторан к CRM, бухгалтерии, AI-сервисам и своему приложению: меню, заказы, движение денег и вебхуки в реальном времени.",
  description:
    "Документация открытого API Keel: API-ключи, меню, заказы, денежная книга для бухгалтерии, остатки, фискальные чеки и подписанные вебхуки.",
  ui: { onThisPage: "На этой странице", copy: "Копировать", copied: "Скопировано", version: "Версия" },
  sections: [
    {
      id: "overview",
      title: "Обзор",
      blocks: [
        p("Каждый ресторан в Keel работает на своём сервере и своём домене. API — тоже на домене ресторана:"),
        code(C.BASE),
        p("Точный адрес владелец видит в панели: **Настройки → Интеграции → API и вебхуки**. Ключ одного ресторана открывает только этот ресторан."),
        p("API работает в две стороны: ваша программа **запрашивает** у нас данные (по API-ключу), а мы **сами сообщаем** вам, когда меняется заказ или деньги (вебхук)."),
        note("`/v1` — это обещание. Пути, поля, заголовки и коды ошибок здесь не меняются; если понадобится изменить — рядом появится `/v2`. В объекты могут добавляться новые поля — игнорируйте незнакомые."),
      ],
    },
    {
      id: "quickstart",
      title: "Быстрый старт",
      blocks: [
        list([
          "Владелец ресторана создаёт ключ в панели и выбирает разрешения. Ключ начинается с `keel_` и показывается **только один раз**.",
          "Владелец передаёт вам две вещи: адрес API и ключ.",
          "Проверьте ключ через `/ping`:",
        ]),
        code(C.PING, "curl"),
        code(C.PING_RESPONSE, "200 OK"),
      ],
    },
    {
      id: "auth",
      title: "Аутентификация и разрешения",
      blocks: [
        p("В каждом запросе — заголовок `Authorization: Bearer keel_…`. У ключа есть разрешения:"),
        table(
          ["Разрешение", "Что даёт"],
          [
            ["`menu:read`", "`/branches/{id}/menu` — меню и что сейчас в продаже"],
            ["`orders:read`", "`/orders`, `/orders/{ref}` — заказы, **с именем и телефоном клиента**"],
            ["`finance:read`", "`/money`, `/money/daily`, `/balances`, `/fiscal` — деньги, **без данных клиентов**"],
          ],
        ),
        p("Для `/ping` и `/branches` подходит любой действующий ключ. Отозванный ключ перестаёт работать со следующего запроса."),
        note("Ключ используется только на сервере. Не кладите его в браузер или мобильное приложение — там его прочитает кто угодно. Cross-origin запросы из браузера не разрешены."),
        p("Лимит: 120 запросов в минуту с одного IP-адреса (дальше — `429`)."),
      ],
    },
    {
      id: "conventions",
      title: "Соглашения",
      blocks: [
        list([
          "Деньги — **целое число сумов** (UZS). Без тийинов и дробей.",
          "Время — RFC 3339 в UTC (`2026-09-24T12:30:15.69Z`).",
          "Для дня используйте поле `day` — это календарный день ресторана. Если отрезать дату от `occurredAt`, всё после 19:00 по Ташкенту уедет на предыдущий день.",
          "Id — 24 hex-символа. Отсутствующий id — пустая строка или поля нет, никогда не `\"000000000000000000000000\"`.",
          "Списки — всегда массив, никогда не `null`.",
        ]),
      ],
    },
    {
      id: "errors",
      title: "Ошибки",
      blocks: [
        code(C.ERROR, "4xx / 5xx"),
        table(
          ["HTTP", "`code`"],
          [
            ["400", "`invalid_request`"],
            ["401", "`unauthorized`"],
            ["403", "`insufficient_scope`"],
            ["404", "`not_found`"],
            ["500", "`internal_error`"],
          ],
        ),
        p("В коде принимайте решения по `code`; `message` — для человека и может меняться."),
      ],
    },
    {
      id: "menu",
      title: "Филиалы и меню",
      blocks: [
        code(C.BRANCHES),
        code(C.MENU),
        p("`available: false` — ресторан и на своём сайте это не продаст: позиция выключена, в стоп-листе, закончилась часть комбо или исчерпан дневной лимит."),
      ],
    },
    {
      id: "orders",
      title: "Заказы",
      blocks: [
        p("`GET /orders` — сначала новые. Фильтры:"),
        table(
          ["Параметр", ""],
          [
            ["`branchId`", "только этот филиал"],
            ["`status`", "`pending`, `confirmed`, `preparing`, `on_the_way`, `delivered`, `cancelled`"],
            ["`createdFrom`, `createdTo`", "RFC 3339; начало включительно, конец — нет"],
            ["`limit`", "1–100, по умолчанию 50"],
            ["`cursor`", "`nextCursor` предыдущей страницы; на последней — `\"\"`"],
          ],
        ),
        code(C.ORDERS),
        p("`GET /orders/{ref}` — один заказ, по номеру (`QVUU-R97U`) или id. Объект заказа:"),
        code(C.ORDER, "Order"),
        list([
          "`type`: `delivery`, `pickup`, `dinein` (стол в зале), `uzum_tezkor` (забирает курьер Uzum).",
          "`address` есть, только если у заказа есть адрес; `cancelReason`, `tableNumber`, `scheduledAt` опускаются, если пусты.",
          "`price` — цена за единицу, уже с опциями.",
        ]),
      ],
    },
    {
      id: "money",
      title: "Денежная книга",
      blocks: [
        p("`GET /money` (`finance:read`) — **каждое движение денег** в ресторане, в одном виде и **уже классифицированное**. Для бухгалтерских сервисов: суммы, способы оплаты, поставщики и сотрудники — никогда имя или телефон гостя."),
        table(
          ["Параметр", ""],
          [
            ["`from`, `to`", "обязательны; дни ресторана (`2026-09-01`), оба включительно, или RFC 3339. Не больше 31 дня."],
            ["`branchId`", "только этот филиал"],
          ],
        ),
        code(C.MONEY),
        note("Не каждое движение денег — доход или расход. Деньги переходят из кассы в банк, закупщику выдают подотчёт, агрегатор перечисляет деньги, которые ресторан уже заработал. Если сложить их как выручку или расходы, одни и те же деньги посчитаются дважды. Поэтому **каждая запись говорит, что она такое**: `class` и `pnl`."),
        table(
          ["`class`", "`pnl`", "Что это"],
          [
            ["`revenue`", "✅", "Оплаченная продажа, в день продажи. Детали — в `sale`."],
            ["`refund`", "✅", "Возврат денег, в день возврата."],
            ["`cost`", "✅", "Закупка у поставщика, расход (аренда, коммунальные, налоги), внешняя служба доставки."],
            ["`payroll`", "✅", "Выплаченная зарплата — сотрудникам и курьерам."],
            ["`commission`", "✅", "Что удержал агрегатор или эквайринг."],
            ["`transfer`", "❌", "Те же деньги в другом месте: касса → банк, агрегатор → банк, курьер → касса. **Не выручка.**"],
            ["`advance`", "❌", "Наличные, выданные сотруднику на закупку (подотчёт). Расходом они становятся в закупке — **а она уже `cost`**."],
            ["`manual`", "❌", "Наличные, внесённые или изъятые из кассы или сейфа вручную, с `category` кассира. Часто это те же деньги, что уже в закупке или расходе."],
            ["`variance`", "❌", "Касса при закрытии смены оказалась с излишком (`in`) или недостачей (`out`). Пропавшие деньги, а не потраченные."],
          ],
        ),
        p("Финансовый отчёт в панели ресторана — это сумма записей с `pnl: true`, то есть `pnlNet`. Если ваша цифра за период отличается — что-то посчитано дважды."),
        p("Скидки и баллы — не отдельные записи: деньги не двигались. Они внутри продажи (`sale.discountTotal`, `sale.pointsSpent`); `amount` — фактически полученная сумма."),
        code(C.MONEY_ENTRY, "MoneyEntry"),
        list([
          "`amount` всегда положителен; знак задаёт `direction` (`in` / `out`, со стороны ресторана).",
          "`source`: `sale`, `refund`, `delivery_service`, `purchase`, `expense`, `salary`, `courier_pay`, `payout`, `collection`, `advance`, `cash_entry`, `safe_entry`, `courier_settlement`, `shift_variance`. Могут появиться новые — **решайте по `class`**.",
          "Необязательные: `method`, `methodName`, `category`, `counterparty`, `from` / `to` (для переводов: till, safe, bank, aggregator, courier, staff), `note`.",
          "`sale` (только `revenue`): `type`, `channel`, `subtotal`, `discountTotal`, `pointsSpent`, `deliveryFee`, `serviceCharge`.",
          "`paid` (только закупки): записывается в день прихода товара; `paidAt` — когда оплатили поставщику.",
        ]),
      ],
    },
    {
      id: "reread",
      title: "Период никогда не закрыт — перечитывайте",
      blocks: [
        p("Книга вычисляется из документов ресторана в момент запроса. Документы исправляют: удаляют дважды введённый расход, правят цену закупки, гасят долг, возвращают деньги через неделю. Поэтому:"),
        list([
          "Запрашивайте период заново и **заменяйте** сохранённое — сопоставляйте по `id`, удаляйте исчезнувшие id. Не дописывайте.",
          "Каждый день перечитывайте минимум последние 7 дней, а после закрытия месяца — весь прошлый месяц. Или используйте вебхук `money.day_changed`.",
          "`id` стабилен: один документ всегда даёт один id.",
        ]),
      ],
    },
    {
      id: "daily",
      title: "Итоги по дням",
      blocks: [
        p("`GET /money/daily` (`finance:read`) — книга, сложенная по дням и филиалам. Это те же записи `/money`, поэтому цифры никогда не расходятся. Параметры как у `/money`, период до 93 дней (квартал)."),
        code(C.DAILY),
      ],
    },
    {
      id: "balances",
      title: "Где деньги — остатки",
      blocks: [
        p("`GET /balances` (`finance:read`, необязательный `branchId`) — положение на сейчас. Те же цифры, что на экране «Где деньги» в панели владельца."),
        code(C.BALANCES),
        note("Общего итога **нет намеренно**. Наличные можно потратить сегодня вечером, деньги в банке — на этой неделе, а деньги у агрегатора — когда решит агрегатор. Их сумма — число, которое ни о чём не правда."),
        list([
          "`counted: true` — кто-то посчитал (остаток из банковского приложения); устаревает, смотрите `at`. `counted: false` — сложено из документов; неверно, если документа не хватает.",
          "Банк — **последний посчитанный остаток**, не выводится из движений: на счёт приходят деньги, которых мы не видим.",
          "`name` — слова самого ресторана; решайте по `kind`: `safe`, `drawer`, `courier`, `advance`, `bank`, `rail`.",
          "`payables.suppliers` — неоплаченные закупки; `receivables.guestDebt` — неоплаченные продажи в долг.",
        ]),
      ],
    },
    {
      id: "fiscal",
      title: "Фискальные чеки",
      blocks: [
        p("`GET /fiscal` (`finance:read`, ≤31 дня) — чеки, отправленные в налоговую по продажам периода, **в любом статусе** (перед закрытием месяца догоняют как раз ожидающие и ошибочные), и Z-отчёты смен, закрытых в периоде."),
        code(C.FISCAL),
        p("`kind` — `sale` или `refund`; `status` — `pending`, `filed` или `failed`; `error` объясняет, почему не отправлено."),
      ],
    },
    {
      id: "webhooks",
      title: "Вебхуки",
      blocks: [
        p("Владелец добавляет ваш адрес `https://` в панели и выбирает события. Он получает одноразовый **ключ подписи** (`whsec_…`) и передаёт его вам."),
        table(
          ["Событие", "Когда"],
          [
            ["`order.created`", "заказ оформлен — сайт, приложение, оператор, касса, Uzum Tezkor"],
            ["`order.status_changed`", "изменился статус заказа"],
            ["`money.day_changed`", "изменились деньги за день — см. ниже"],
            ["`ping`", "владелец нажал «Проверить» в панели"],
          ],
        ),
        code(C.WEBHOOK_REQUEST, "HTTP"),
        list([
          "`data.status` — статус, о котором **это событие**. `data.order` — снимок на момент постановки в очередь, поэтому при быстрых изменениях `data.order.status` может быть уже следующим.",
          "`createdAt` — когда произошло изменение. Касса, работавшая офлайн, отправит продажи позже — со временем, когда они были.",
          "Ответьте любым `2xx` **за 10 секунд**, а работу делайте потом. Другой статус, таймаут или редирект — ошибка; редиректы не выполняются.",
          "Повторы после ошибки: 1 мин, 5 мин, 30 мин, 2 ч, 6 ч, 12 ч, 24 ч (8 попыток, около двух суток).",
          "Одно событие может прийти несколько раз — отбрасывайте повторы по `id` (`Keel-Event-Id`).",
          "События отправляются по очереди, но повтор может прийти после более нового события — **упорядочивайте по `createdAt`**, а не по времени прихода.",
        ]),
      ],
    },
    {
      id: "money-webhook",
      title: "money.day_changed",
      blocks: [
        code(C.MONEY_EVENT),
        p("Это **подсказка, а не деньги: перечитайте этот день** (`GET /money?from=<day>&to=<day>`) и замените сохранённое. Приходит на всё, что меняет цифры дня — новая продажа, исправленная закупка, удалённый расход — за последние 35 дней, проверка раз в 10 минут. `entries: 0` — все записи дня удалены."),
        p("За дни до регистрации адреса ничего не отправляется — эти периоды прочитайте один раз сами."),
      ],
    },
    {
      id: "signature",
      title: "Проверка подписи",
      blocks: [
        code(C.SIGNATURE_FORMULA),
        p("Используются **сырые байты тела** — до разбора JSON. Отклоняйте запрос, если подпись не совпала или `t` отличается от ваших часов больше чем на 5 минут. Время входит в подпись, поэтому старую доставку нельзя переотправить."),
        code(C.TEST_VECTOR, "Тестовый вектор"),
        code(C.VERIFY_NODE, "Node.js"),
        code(C.VERIFY_PYTHON, "Python"),
        code(C.VERIFY_PHP, "PHP"),
        p("Если владелец сменит ключ подписи, доставки из очереди будут подписаны новым."),
      ],
    },
  ],
};

// ---- English ----

const en: DevDoc = {
  title: "Keel API for developers",
  lead: "Connect a restaurant to a CRM, accounting, AI services or your own app: the menu, orders, every movement of money, and real-time webhooks.",
  description:
    "Keel open API reference: API keys, menu, orders, a money ledger for accounting, balances, fiscal receipts and signed webhooks.",
  ui: { onThisPage: "On this page", copy: "Copy", copied: "Copied", version: "Version" },
  sections: [
    {
      id: "overview",
      title: "Overview",
      blocks: [
        p("Every restaurant on Keel runs on its own server and its own domain. So does its API:"),
        code(C.BASE),
        p("The owner finds the exact address in the panel: **Settings → Integrations → API and webhooks**. A key opens its own restaurant and no other."),
        p("It works both ways: your program **asks** us (with an API key), and we **tell you** when an order or the money changes (webhooks)."),
        note("`/v1` is a promise. Paths, fields, headers and error codes here do not change; if one has to, `/v2` appears beside it. New fields may be added to any object — ignore what you do not know."),
      ],
    },
    {
      id: "quickstart",
      title: "Quick start",
      blocks: [
        list([
          "The restaurant owner creates a key in the panel and picks its permissions. It starts with `keel_` and is shown **once**.",
          "The owner gives you two things: the API address and the key.",
          "Check the key with `/ping`:",
        ]),
        code(C.PING, "curl"),
        code(C.PING_RESPONSE, "200 OK"),
      ],
    },
    {
      id: "auth",
      title: "Authentication and scopes",
      blocks: [
        p("Every request carries `Authorization: Bearer keel_…`. Keys carry scopes:"),
        table(
          ["Scope", "Allows"],
          [
            ["`menu:read`", "`/branches/{id}/menu` — the menu and what is on sale now"],
            ["`orders:read`", "`/orders`, `/orders/{ref}` — orders, **with the customer's name and phone**"],
            ["`finance:read`", "`/money`, `/money/daily`, `/balances`, `/fiscal` — money, **no customer data**"],
          ],
        ),
        p("`/ping` and `/branches` need any live key. A revoked key stops working on the next request."),
        note("Server to server only. Never put a key in a browser or a mobile app — anyone can read it there. Cross-origin browser requests are not allowed."),
        p("Rate limit: 120 requests a minute per IP address (`429` beyond that)."),
      ],
    },
    {
      id: "conventions",
      title: "Conventions",
      blocks: [
        list([
          "Money is an **integer number of so'm** (UZS). No tiyin, no decimals.",
          "Times are RFC 3339 in UTC (`2026-09-24T12:30:15.69Z`).",
          "For the day, use the `day` field — the restaurant's own calendar day. Cutting a date off `occurredAt` puts everything after 19:00 Tashkent time on the previous day.",
          "Ids are 24 hex characters. An absent id is an empty string or a missing field, never `\"000000000000000000000000\"`.",
          "Lists are always arrays, never `null`.",
        ]),
      ],
    },
    {
      id: "errors",
      title: "Errors",
      blocks: [
        code(C.ERROR, "4xx / 5xx"),
        table(
          ["HTTP", "`code`"],
          [
            ["400", "`invalid_request`"],
            ["401", "`unauthorized`"],
            ["403", "`insufficient_scope`"],
            ["404", "`not_found`"],
            ["500", "`internal_error`"],
          ],
        ),
        p("Branch on `code`; `message` is for humans and may change."),
      ],
    },
    {
      id: "menu",
      title: "Branches and menu",
      blocks: [
        code(C.BRANCHES),
        code(C.MENU),
        p("`available: false` means the restaurant would refuse it on its own site too: switched off, on a stop list, a combo with a sold-out part, or a daily limit reached."),
      ],
    },
    {
      id: "orders",
      title: "Orders",
      blocks: [
        p("`GET /orders` — newest first. Filters:"),
        table(
          ["Query", ""],
          [
            ["`branchId`", "only this branch"],
            ["`status`", "`pending`, `confirmed`, `preparing`, `on_the_way`, `delivered`, `cancelled`"],
            ["`createdFrom`, `createdTo`", "RFC 3339; from inclusive, to exclusive"],
            ["`limit`", "1–100, default 50"],
            ["`cursor`", "the previous page's `nextCursor`; `\"\"` on the last page"],
          ],
        ),
        code(C.ORDERS),
        p("`GET /orders/{ref}` — one order, by its number (`QVUU-R97U`) or id. The Order object:"),
        code(C.ORDER, "Order"),
        list([
          "`type`: `delivery`, `pickup`, `dinein` (a table in the room), `uzum_tezkor` (carried by Uzum's courier).",
          "`address` is present only when the order has one; `cancelReason`, `tableNumber`, `scheduledAt` are omitted when empty.",
          "`price` is per unit and includes the options.",
        ]),
      ],
    },
    {
      id: "money",
      title: "The money ledger",
      blocks: [
        p("`GET /money` (`finance:read`) — **every movement of money** in the restaurant, one shape, **already classified**. Built for accounting services: amounts, methods, suppliers and staff — never a guest's name or phone."),
        table(
          ["Query", ""],
          [
            ["`from`, `to`", "required; the restaurant's days (`2026-09-01`), both inclusive, or RFC 3339. At most 31 days."],
            ["`branchId`", "only this branch"],
          ],
        ),
        code(C.MONEY),
        note("Not every movement of money is income or an expense. Money moves from the till to the bank, a buyer is handed cash for the market, an aggregator passes on money the restaurant already earned. Adding those up as income or costs counts the same money twice. So **every entry says what it is**: `class` and `pnl`."),
        table(
          ["`class`", "`pnl`", "What it is"],
          [
            ["`revenue`", "✅", "A paid sale, on the day it was made. Breakdown in `sale`."],
            ["`refund`", "✅", "A sale's money handed back, on the day it was handed back."],
            ["`cost`", "✅", "A purchase from a supplier, an expense (rent, utilities, tax), an outside delivery service."],
            ["`payroll`", "✅", "Wages paid — staff and couriers."],
            ["`commission`", "✅", "What an aggregator or acquirer kept."],
            ["`transfer`", "❌", "The same money in another place: till → bank, aggregator → bank, courier → till. **Not income.**"],
            ["`advance`", "❌", "Cash handed to an employee to spend (the market run). It becomes a cost when it buys something — **and that purchase is already a `cost`**."],
            ["`manual`", "❌", "Cash put into or taken out of a drawer or the safe by hand, with the cashier's own `category`. Often the same money a purchase or expense already records."],
            ["`variance`", "❌", "A drawer counted over (`in`) or under (`out`) at closing. Missing money, not money spent."],
          ],
        ),
        p("The restaurant's own money report is the sum of the `pnl: true` entries: `pnlNet`. If your figure for a period differs, something is counted twice."),
        p("Discounts and points are not entries: no money moved. They sit inside a sale (`sale.discountTotal`, `sale.pointsSpent`); `amount` is what was actually charged."),
        code(C.MONEY_ENTRY, "MoneyEntry"),
        list([
          "`amount` is always positive; `direction` (`in` / `out`, from the restaurant's side) gives the sign.",
          "`source`: `sale`, `refund`, `delivery_service`, `purchase`, `expense`, `salary`, `courier_pay`, `payout`, `collection`, `advance`, `cash_entry`, `safe_entry`, `courier_settlement`, `shift_variance`. New ones may be added — **always decide by `class`**.",
          "Optional: `method`, `methodName`, `category`, `counterparty`, `from` / `to` (for transfers: till, safe, bank, aggregator, courier, staff), `note`.",
          "`sale` (revenue only): `type`, `channel`, `subtotal`, `discountTotal`, `pointsSpent`, `deliveryFee`, `serviceCharge`.",
          "`paid` (purchases only): booked when the goods arrived; `paidAt` once the supplier was paid.",
        ]),
      ],
    },
    {
      id: "reread",
      title: "A period is never final — re-read it",
      blocks: [
        p("The ledger is computed from the restaurant's documents when you ask. Documents get corrected: an expense typed twice is deleted, a purchase price is fixed, a debt is repaid, a refund is given a week later. So:"),
        list([
          "Fetch a period again and **replace** what you stored — match by `id`, delete the ids that are gone. Do not append.",
          "Re-read at least the last 7 days every day, and the whole previous month once it has closed — or use the `money.day_changed` webhook.",
          "`id` is stable: the same document always gives the same id.",
        ]),
      ],
    },
    {
      id: "daily",
      title: "Daily totals",
      blocks: [
        p("`GET /money/daily` (`finance:read`) — the ledger summed per day and branch. The same entries as `/money`, so the two never disagree. Same query as `/money`, up to 93 days (a quarter)."),
        code(C.DAILY),
      ],
    },
    {
      id: "balances",
      title: "Where the money is",
      blocks: [
        p("`GET /balances` (`finance:read`, optional `branchId`) — the position right now. The same figures as the owner's \"Where the money is\" screen."),
        code(C.BALANCES),
        note("There is **deliberately no grand total**. Cash can be spent tonight, the bank this week, and money an aggregator holds when they decide. Adding them gives a number that is true of nothing."),
        list([
          "`counted: true` — somebody counted it (a balance read off the bank's app); it goes stale, check `at`. `counted: false` — added up from documents; wrong when a document is missing.",
          "The bank is the **last counted balance**, never derived from movements: money reaches the account from places we do not see.",
          "`name` is the restaurant's own wording; decide by `kind`: `safe`, `drawer`, `courier`, `advance`, `bank`, `rail`.",
          "`payables.suppliers` — deliveries not yet paid for; `receivables.guestDebt` — sales on credit not yet paid.",
        ]),
      ],
    },
    {
      id: "fiscal",
      title: "Fiscal receipts",
      blocks: [
        p("`GET /fiscal` (`finance:read`, ≤ 31 days) — receipts filed with the tax committee for the period's sales, **in every status** (the pending and failed ones are what gets chased before the month closes), and the Z-reports of shifts closed in the period."),
        code(C.FISCAL),
        p("`kind` is `sale` or `refund`; `status` is `pending`, `filed` or `failed`; `error` says why a filing failed."),
      ],
    },
    {
      id: "webhooks",
      title: "Webhooks",
      blocks: [
        p("The owner registers your `https://` address in the panel and picks events. They receive a one-time **signing secret** (`whsec_…`) to give to you."),
        table(
          ["Event", "When"],
          [
            ["`order.created`", "an order is placed — site, app, phone operator, till, Uzum Tezkor"],
            ["`order.status_changed`", "its status changes"],
            ["`money.day_changed`", "a day's money changed — see below"],
            ["`ping`", "the owner pressed \"Test\" in the panel"],
          ],
        ),
        code(C.WEBHOOK_REQUEST, "HTTP"),
        list([
          "`data.status` is the status **this event** is about. `data.order` is a snapshot taken when the event was queued, so for quick successive changes `data.order.status` may already be the later one.",
          "`createdAt` is when the change happened. A till that was offline sends its sales later, dated when they were made.",
          "Answer with any `2xx` **within 10 seconds**, then do the work. Anything else — another status, a timeout, a redirect — is a failure; redirects are not followed.",
          "Retries after a failure: 1 min, 5 min, 30 min, 2 h, 6 h, 12 h, 24 h (8 attempts, about two days).",
          "The same event can arrive more than once — deduplicate by `id` (`Keel-Event-Id`).",
          "Events are sent one at a time, but a retry can arrive after a newer event — **order by `createdAt`**, not by arrival.",
        ]),
      ],
    },
    {
      id: "money-webhook",
      title: "money.day_changed",
      blocks: [
        code(C.MONEY_EVENT),
        p("A **hint, not the money: re-read that day** (`GET /money?from=<day>&to=<day>`) and replace what you stored. Raised by anything that changes the day's figures — a new sale, an edited purchase, a deleted expense — within the last 35 days, checked every 10 minutes. `entries: 0` means every entry of the day was deleted."),
        p("Nothing is sent for the days before the address was registered — read those periods yourself once."),
      ],
    },
    {
      id: "signature",
      title: "Verifying the signature",
      blocks: [
        code(C.SIGNATURE_FORMULA),
        p("Use the **raw body bytes**, before any JSON parsing. Reject the request if the signature does not match or `t` is more than 5 minutes from your clock. The time is inside the signature, so an old delivery cannot be replayed."),
        code(C.TEST_VECTOR, "Test vector"),
        code(C.VERIFY_NODE, "Node.js"),
        code(C.VERIFY_PYTHON, "Python"),
        code(C.VERIFY_PHP, "PHP"),
        p("When the owner replaces the secret, deliveries still in the queue are signed with the new one."),
      ],
    },
  ],
};

export const developers: Record<Lang, DevDoc> = { uz, ru, en };

/** The API version this page documents. */
export const API_VERSION = "v1";
