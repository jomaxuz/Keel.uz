// The public offer and the privacy policy, in three languages.
//
// ⚠️ **These are live legal documents, read and approved by the company's director on
// 2026-08-09.** They no longer carry a draft warning, which means a change here changes what
// the company is bound by — so a factual edit is not a copy edit. If a term stops being true
// (a price, a retention period, a third party), the document is wrong until it is updated, and
// wrong in the way that gets quoted back at us.
//
// ⚠️ **Every factual claim here is one the platform actually keeps**, and that is the whole
// discipline of writing them: the retention periods, the hashed visit counter, the per-tenant
// databases and the list of third parties are read off the system as built (see CLAUDE.md).
// A policy that promises something the code does not do is worse than no policy — it is a
// document that will be quoted back at us.
//
// Kept out of `dict.ts` on purpose: that file is interface copy, read in fragments, and
// dropping two long legal documents into it would make every other translation harder to
// scan. Here each document is a list of sections so the page renders them without knowing
// what they say.

export interface LegalSection {
  heading: string;
  /** Paragraphs. A list item starts with "— " and is rendered as one. */
  body: string[];
}

export interface LegalDoc {
  title: string;
  updated: string;
  intro: string[];
  sections: LegalSection[];
}

/** The company, in one place. ⚠️ Written once and referenced by both documents: an offer and
 *  a policy that disagree about the address is the first thing a careful reader notices, and
 *  the second thing a careless one uses. */
export const COMPANY = {
  nameUz: '"KEEL" MChJ',
  nameRu: 'ООО "KEEL"',
  nameEn: 'KEEL LLC',
  inn: "313241548",
  oked: "62010",
  directorUz: "Nurmanov Y. A.",
  addressUz:
    "Toshkent shahri, Yunusobod tumani, Yunus ota MFY, 14-mavze, 5-uy, 40-xonadon",
  addressRu:
    "город Ташкент, Юнусабадский район, Yunus ota MFY, 14 mavze, 5-uy, 40-xonadon",
  addressEn:
    "Tashkent, Yunusabad district, Yunus ota MFY, 14 mavze, house 5, apt. 40",
  account: "20208000907516890001",
  site: "keel.uz",
  email: "info@keel.uz",
};

const UPDATED = "2026-08-15";

// ---- Uzbek ----

const offerUz: LegalDoc = {
  title: "Ommaviy taklif (oferta)",
  updated: UPDATED,
  intro: [
    `${COMPANY.nameUz} (keyingi o'rinlarda — Ijrochi) ushbu ommaviy taklif orqali quyidagi shartlarda Keel platformasi xizmatlarini taklif qiladi.`,
    "Xizmatdan foydalanishni boshlash — hisob ochish, saytni ishga tushirish yoki to'lovni amalga oshirish — ushbu shartlarni to'liq qabul qilish hisoblanadi.",
  ],
  sections: [
    {
      heading: "1. Atamalar",
      body: [
        "— Platforma — Keel: restoran uchun veb-sayt, buyurtma qabul qilish va boshqaruv paneli.",
        "— Buyurtmachi — platformadan foydalanadigan yuridik yoki jismoniy shaxs (restoran, do'kon, kafe).",
        "— Mehmon — Buyurtmachining saytida buyurtma beradigan yoki ma'lumot izlaydigan shaxs.",
        "— Hisob davri — to'lov hisoblanadigan davr; odatda bir oy.",
      ],
    },
    {
      heading: "2. Xizmat predmeti",
      body: [
        "Ijrochi Buyurtmachiga quyidagilarni taqdim etadi: alohida domenda ishlaydigan veb-sayt; menyu, buyurtma va yetkazib berish tizimi; boshqaruv paneli; Telegram bot va mini app; xodimlar, kuryerlar va kassa uchun ilovalar.",
        "Xizmat SaaS (obuna) modelida ko'rsatiladi: dasturiy ta'minot Ijrochi serverlarida ishlaydi va Buyurtmachiga foydalanish huquqi beriladi.",
        "Har bir Buyurtmachi uchun alohida ma'lumotlar bazasi va alohida ishlayotgan nusxa ajratiladi.",
      ],
    },
    {
      heading: "3. Xizmat narxi va to'lov tartibi",
      body: [
        "Asosiy to'lov qabul qilingan buyurtmalar soniga qarab hisoblanadi. Bir buyurtma narxi va minimal oylik to'lov Buyurtmachi bilan alohida kelishiladi va hisob-fakturada ko'rsatiladi.",
        "Qo'shimcha xizmatlar alohida to'lanadi. Jumladan: saytdan «Powered by Keel» belgisini olib tashlash — oyiga 3 000 000 (uch million) so'm. Bu xizmat yoqilgan kunlar soniga mutanosib hisoblanadi.",
        "Sayt dizaynini alohida buyurtma bo'yicha ishlab chiqish xizmati platforma obunasiga kirmaydi va alohida kelishiladi.",
        "To'lov Ijrochining hisob raqamiga pul o'tkazish yo'li bilan yoki naqd shaklda amalga oshiriladi. Har bir davr uchun hisob-faktura chiqariladi.",
        "Sinov muddati (demo) berilishi mumkin. Sinov muddatida to'lov olinmaydi.",
      ],
    },
    {
      heading: "4. Tomonlarning majburiyatlari",
      body: [
        "Ijrochi: platformaning ishlashini ta'minlaydi, ma'lumotlarning kunlik zaxira nusxasini oladi, xizmatga tegishli yangilanishlarni chiqaradi va texnik qo'llab-quvvatlash beradi.",
        "Buyurtmachi: sayt mazmuni (menyu, narxlar, rasmlar, matnlar) va uning qonuniyligi uchun javob beradi; boshqaruv paneli hisob ma'lumotlarini sir tutadi; mehmonlar bilan munosabatlarni (buyurtma, yetkazib berish, qaytarish) o'zi hal qiladi.",
        "Buyurtmachi platformadan qonunga xilof faoliyat uchun, shuningdek uchinchi shaxslarning huquqlarini buzadigan mazmun joylashtirish uchun foydalanmaydi.",
      ],
    },
    {
      heading: "5. Ma'lumotlar egaligi",
      body: [
        "Buyurtmachining ma'lumotlari (menyu, buyurtmalar, mijozlar bazasi, moliyaviy yozuvlar) Buyurtmachiga tegishli.",
        "Buyurtmachi so'rovi bo'yicha ma'lumotlar mashina o'qiy oladigan shaklda (JSON va rasmlar arxivi) beriladi. Bu imkoniyat boshqa tizimga o'tish uchun ham amal qiladi.",
        "Ijrochi Buyurtmachining ma'lumotlarini o'z maqsadlarida sotmaydi va uchinchi shaxslarga bermaydi, qonun talab qilgan hollardan tashqari.",
      ],
    },
    {
      heading: "6. Uchinchi tomon xizmatlari",
      body: [
        "Platforma boshqa xizmatlar bilan ishlaydi: SMS shlyuzlari, onlayn to'lov tizimlari (Payme, Click, Uzum, ATMOS), Telegram, xarita xizmatlari, kassa (POS) tizimlari.",
        "Bu xizmatlar bo'yicha shartnoma va to'lovlar Buyurtmachining o'zi tomonidan tuziladi. Ularning ishlamay qolishi yoki shartlarini o'zgartirishi uchun Ijrochi javob bermaydi.",
      ],
    },
    {
      heading: "7. Javobgarlikning cheklanishi",
      body: [
        "Ijrochi xizmatning uzluksiz ishlashiga harakat qiladi, lekin internet, hosting, elektr ta'minoti yoki uchinchi tomon xizmatlaridagi uzilishlar uchun javobgar emas.",
        "Ijrochining javobgarligi oxirgi hisob davri uchun to'langan summa bilan cheklanadi.",
        "Buyurtmachi o'z hisob ma'lumotlaridan foydalanib amalga oshirilgan barcha harakatlar uchun javobgar.",
      ],
    },
    {
      heading: "8. Xizmatni to'xtatib turish va bekor qilish",
      body: [
        "To'lov kechiktirilganda Ijrochi Buyurtmachini ogohlantirib, xizmatni vaqtincha to'xtatib turishga haqli. To'xtatilgan davrda ma'lumotlar saqlanadi.",
        "Buyurtmachi xizmatdan istalgan vaqtda voz kechishi mumkin. Bunda joriy davr uchun to'lov qaytarilmaydi, ma'lumotlar esa yuqoridagi 5-bandga muvofiq beriladi.",
      ],
    },
    {
      heading: "9. Nizolarni hal qilish",
      body: [
        "Nizolar muzokaralar yo'li bilan hal qilinadi. Kelishuvga erishilmasa, ish O'zbekiston Respublikasi qonunchiligiga muvofiq sudda ko'riladi.",
      ],
    },
    {
      heading: "10. Rekvizitlar",
      body: [
        `Tashkilot: ${COMPANY.nameUz}`,
        `STIR (INN): ${COMPANY.inn}`,
        `IFUT (OKED): ${COMPANY.oked}`,
        `Direktor: ${COMPANY.directorUz}`,
        `Manzil: ${COMPANY.addressUz}`,
        `Hisob raqami: ${COMPANY.account}`,
        `Sayt: ${COMPANY.site}`,
      ],
    },
  ],
};

const privacyUz: LegalDoc = {
  title: "Maxfiylik siyosati",
  updated: UPDATED,
  intro: [
    `${COMPANY.nameUz} shaxsiy ma'lumotlarni O'zbekiston Respublikasining «Shaxsga doir ma'lumotlar to'g'risida»gi qonuni talablariga muvofiq qayta ishlaydi.`,
    "Ushbu siyosat Keel platformasi va uning yordamida ishlayotgan restoran saytlariga taalluqli.",
  ],
  sections: [
    {
      heading: "1. Kim ma'lumotni boshqaradi",
      body: [
        "Restoran mehmonlarining ma'lumotlari uchun ma'lumot egasi — restoranning o'zi. Ijrochi bu ma'lumotlarni restoran topshirig'i bo'yicha, xizmat ko'rsatish uchun qayta ishlaydi.",
        "Platformaning o'z mijozlari (restoran egalari, xodimlari) ma'lumotlari uchun ma'lumot egasi — Ijrochi.",
      ],
    },
    {
      heading: "2. Qanday ma'lumotlar yig'iladi",
      body: [
        "— Mehmon haqida: ism, telefon raqami, yetkazib berish manzili va unga izoh, buyurtmalar tarixi, bonus ballari, tanlangan til. Telegram orqali kirganda — Telegram hisobining raqami va ismi.",
        "— Restoran xodimlari haqida: ism, telefon, login, ish grafigi va smena yozuvlari; ishga kirish-chiqish paytidagi joylashuv (agar restoran shu funksiyani yoqqan bo'lsa).",
        "— Texnik ma'lumotlar: so'rov jurnallari, xatolar, xizmat ishlashi haqidagi o'lchovlar.",
        "— Tashriflar hisobi: sahifa ochilishlari kunlik anonim belgi bilan sanaladi. Belgi kun bilan birga xeshlanadi, ya'ni bir brauzerni kunlar bo'ylab kuzatib bo'lmaydi.",
      ],
    },
    {
      heading: "3. Nima uchun ishlatiladi",
      body: [
        "Buyurtmani qabul qilish va yetkazib berish; mehmoz bilan buyurtma yuzasidan bog'lanish; SMS yoki Telegram orqali kirish; bonus ballarini hisoblash; restoran uchun statistika; xizmatning o'zini ishlatish va nosozliklarni topish.",
        "Reklama xabarlari faqat rad etmagan mijozlarga yuboriladi. Rad etish istalgan vaqtda amal qiladi va buyurtma holati haqidagi xabarlarga ta'sir qilmaydi.",
      ],
    },
    {
      heading: "4. Kimga beriladi",
      body: [
        "Ma'lumot faqat xizmat ko'rsatish uchun zarur bo'lgan hollarda uchinchi tomon xizmatlariga uzatiladi: SMS shlyuzi (kirish kodi va xabarlar uchun telefon raqami), onlayn to'lov tizimlari (buyurtma raqami va summasi), Telegram (bot xabarlari), xarita xizmati (manzil koordinatalari), kassa tizimi (buyurtma tarkibi).",
        "Karta raqamlari platformada saqlanmaydi va u orqali o'tmaydi — to'lov to'lov tizimining o'z sahifasida amalga oshiriladi.",
        "Ma'lumot sotilmaydi va reklama maqsadida uchinchi shaxslarga berilmaydi.",
      ],
    },
    {
      heading: "5. Qancha vaqt saqlanadi",
      body: [
        "Buyurtmalar va mijoz ma'lumotlari restoran hisobi faol bo'lgan davrda saqlanadi.",
        "Zaxira nusxalar: kunlik nusxa 14 kun, oy boshidagi nusxa 180 kun saqlanadi.",
        "Tashrif hisobi yozuvlari 100 kundan keyin avtomatik o'chiriladi.",
        "Bir martalik SMS kodlari 3 daqiqadan keyin amaldan chiqadi va o'chiriladi.",
      ],
    },
    {
      heading: "6. Foydalanuvchining huquqlari",
      body: [
        "O'z ma'lumotlarini bilish, tuzatish va o'chirishni so'rash huquqi. Ism va manzillar profil sahifasida o'zgartiriladi; telefon raqamini o'zgartirish SMS kod bilan tasdiqlanadi.",
        "Reklama xabarlaridan voz kechish huquqi.",
        "So'rovlar restoranga yoki Ijrochiga yuboriladi: " + COMPANY.email,
      ],
    },
    {
      heading: "7. Xavfsizlik",
      body: [
        "Parollar qaytarib bo'lmaydigan shaklda (bcrypt) saqlanadi; ulanish HTTPS orqali; har bir restoran ma'lumotlari alohida bazada; to'lov va SMS kalitlari saytga qaytarilmaydigan alohida kolleksiyada.",
        "Boshqaruv paneli amallari jurnalga yoziladi: kim, qachon, nima qilgani.",
      ],
    },
    {
      heading: "8. Cookie fayllari",
      body: [
        "Restoran saytlarida faqat zarur cookie'lardan foydalaniladi: tanlangan til, tanlangan brend va filial, savat va sessiya. U yerda reklama yoki kuzatuv uchun uchinchi tomon skriptlari ishlatilmaydi.",
        "keel.uz saytining o'zida bundan tashqari tashrif statistikasi va reklamamiz natijasini o'lchash vositalari ishlatiladi. Ular sahifa manzili hamda brauzer va qurilma haqidagi umumiy ma'lumotdan foydalanadi; bu sahifada ismingiz, telefoningiz yoki elektron pochtangiz so'ralmaydi.",
      ],
    },
    {
      heading: "9. O'zgarishlar va aloqa",
      body: [
        "Siyosat o'zgarganda yangi tahriri shu sahifada e'lon qiladi va yangilanish sanasi ko'rsatiladi.",
        `Aloqa: ${COMPANY.email} · ${COMPANY.site}`,
        `Tashkilot: ${COMPANY.nameUz}, STIR ${COMPANY.inn}, ${COMPANY.addressUz}`,
      ],
    },
  ],
};

// ---- Russian ----

const offerRu: LegalDoc = {
  title: "Публичная оферта",
  updated: UPDATED,
  intro: [
    `${COMPANY.nameRu} (далее — Исполнитель) настоящей публичной офертой предлагает услуги платформы Keel на изложенных ниже условиях.`,
    "Начало использования сервиса — создание аккаунта, запуск сайта или оплата — означает полное принятие настоящих условий.",
  ],
  sections: [
    {
      heading: "1. Термины",
      body: [
        "— Платформа — Keel: сайт для ресторана, приём заказов и панель управления.",
        "— Заказчик — юридическое или физическое лицо, использующее платформу (ресторан, магазин, кафе).",
        "— Гость — лицо, оформляющее заказ на сайте Заказчика.",
        "— Расчётный период — период, за который начисляется оплата; как правило, один месяц.",
      ],
    },
    {
      heading: "2. Предмет услуг",
      body: [
        "Исполнитель предоставляет: сайт на отдельном домене; меню, заказы и доставку; панель управления; Telegram-бот и мини-приложение; приложения для сотрудников, курьеров и кассы.",
        "Услуга оказывается по модели SaaS: программное обеспечение работает на серверах Исполнителя, Заказчику предоставляется право использования.",
        "Для каждого Заказчика выделяется отдельная база данных и отдельный работающий экземпляр.",
      ],
    },
    {
      heading: "3. Стоимость и порядок оплаты",
      body: [
        "Основная плата начисляется по количеству принятых заказов. Цена за заказ и минимальный месячный платёж согласуются с Заказчиком и указываются в счёте.",
        "Дополнительные услуги оплачиваются отдельно. В том числе: удаление отметки «Powered by Keel» — 3 000 000 (три миллиона) сум в месяц. Начисляется пропорционально числу дней, когда услуга была включена.",
        "Индивидуальная разработка дизайна сайта не входит в подписку и согласуется отдельно.",
        "Оплата производится перечислением на расчётный счёт Исполнителя либо наличными. За каждый период выставляется счёт.",
        "Может быть предоставлен пробный период. В пробный период оплата не взимается.",
      ],
    },
    {
      heading: "4. Обязанности сторон",
      body: [
        "Исполнитель: обеспечивает работу платформы, ежедневное резервное копирование данных, выпуск обновлений и техническую поддержку.",
        "Заказчик: отвечает за содержание сайта (меню, цены, изображения, тексты) и его законность; хранит данные доступа в тайне; самостоятельно решает вопросы с гостями (заказ, доставка, возврат).",
        "Заказчик не использует платформу для противоправной деятельности и для размещения материалов, нарушающих права третьих лиц.",
      ],
    },
    {
      heading: "5. Принадлежность данных",
      body: [
        "Данные Заказчика (меню, заказы, база клиентов, финансовые записи) принадлежат Заказчику.",
        "По запросу Заказчика данные предоставляются в машиночитаемом виде (архив JSON и изображений), в том числе для перехода на другую систему.",
        "Исполнитель не продаёт данные Заказчика и не передаёт их третьим лицам, кроме случаев, предусмотренных законом.",
      ],
    },
    {
      heading: "6. Сервисы третьих лиц",
      body: [
        "Платформа работает с внешними сервисами: SMS-шлюзы, платёжные системы (Payme, Click, Uzum, ATMOS), Telegram, карты, кассовые (POS) системы.",
        "Договоры и оплата по этим сервисам заключаются Заказчиком самостоятельно. Исполнитель не отвечает за их сбои и изменение их условий.",
      ],
    },
    {
      heading: "7. Ограничение ответственности",
      body: [
        "Исполнитель стремится обеспечить бесперебойную работу, но не отвечает за перерывы, вызванные интернетом, хостингом, электроснабжением или сервисами третьих лиц.",
        "Ответственность Исполнителя ограничена суммой, оплаченной за последний расчётный период.",
        "Заказчик отвечает за все действия, совершённые с использованием его данных доступа.",
      ],
    },
    {
      heading: "8. Приостановка и прекращение",
      body: [
        "При просрочке оплаты Исполнитель вправе, предупредив Заказчика, временно приостановить услугу. Данные при этом сохраняются.",
        "Заказчик может отказаться от услуги в любое время. Оплата за текущий период не возвращается, данные предоставляются согласно пункту 5.",
      ],
    },
    {
      heading: "9. Разрешение споров",
      body: [
        "Споры решаются путём переговоров. При отсутствии согласия спор рассматривается в суде в соответствии с законодательством Республики Узбекистан.",
      ],
    },
    {
      heading: "10. Реквизиты",
      body: [
        `Организация: ${COMPANY.nameRu}`,
        `ИНН: ${COMPANY.inn}`,
        `ОКЭД: ${COMPANY.oked}`,
        `Директор: ${COMPANY.directorUz}`,
        `Адрес: ${COMPANY.addressRu}`,
        `Расчётный счёт: ${COMPANY.account}`,
        `Сайт: ${COMPANY.site}`,
      ],
    },
  ],
};

const privacyRu: LegalDoc = {
  title: "Политика конфиденциальности",
  updated: UPDATED,
  intro: [
    `${COMPANY.nameRu} обрабатывает персональные данные в соответствии с законом Республики Узбекистан «О персональных данных».`,
    "Политика распространяется на платформу Keel и на сайты ресторанов, работающие на ней.",
  ],
  sections: [
    {
      heading: "1. Кто распоряжается данными",
      body: [
        "В отношении данных гостей ресторана владельцем данных является сам ресторан. Исполнитель обрабатывает их по поручению ресторана — для оказания услуги.",
        "В отношении данных клиентов платформы (владельцев и сотрудников ресторанов) владельцем данных является Исполнитель.",
      ],
    },
    {
      heading: "2. Какие данные собираются",
      body: [
        "— О гостe: имя, номер телефона, адрес доставки и комментарий к нему, история заказов, бонусные баллы, выбранный язык. При входе через Telegram — идентификатор и имя аккаунта Telegram.",
        "— О сотрудниках ресторана: имя, телефон, логин, график и записи смен; местоположение в момент отметки прихода и уходa (если ресторан включил эту функцию).",
        "— Технические данные: журналы запросов, ошибки, метрики работы сервиса.",
        "— Счёт посещений: открытия страниц считаются по анонимной суточной метке. Метка хешируется вместе с датой, поэтому отследить один браузер по дням невозможно.",
      ],
    },
    {
      heading: "3. Для чего используются",
      body: [
        "Приём и доставка заказа; связь с гостем по заказу; вход по SMS или через Telegram; начисление бонусов; статистика для ресторана; работа самого сервиса и поиск неисправностей.",
        "Рекламные сообщения отправляются только тем, кто не отказался от них. Отказ действует бессрочно и не влияет на сообщения о статусе заказа.",
      ],
    },
    {
      heading: "4. Кому передаются",
      body: [
        "Данные передаются сторонним сервисам только в объёме, необходимом для услуги: SMS-шлюз (номер телефона для кода и сообщений), платёжные системы (номер и сумма заказа), Telegram (сообщения бота), картографический сервис (координаты адреса), кассовая система (состав заказа).",
        "Номера карт не хранятся на платформе и не проходят через неё — оплата совершается на странице платёжной системы.",
        "Данные не продаются и не передаются третьим лицам в рекламных целях.",
      ],
    },
    {
      heading: "5. Сроки хранения",
      body: [
        "Заказы и данные клиентов хранятся, пока аккаунт ресторана активен.",
        "Резервные копии: суточная — 14 дней, копия на начало месяца — 180 дней.",
        "Записи счётчика посещений удаляются автоматически через 100 дней.",
        "Одноразовые SMS-коды действуют 3 минуты и удаляются.",
      ],
    },
    {
      heading: "6. Права пользователя",
      body: [
        "Право знать свои данные, исправить их и потребовать удаления. Имя и адреса меняются в профиле; смена номера подтверждается SMS-кодом.",
        "Право отказаться от рекламных сообщений.",
        "Запросы направляются ресторану или Исполнителю: " + COMPANY.email,
      ],
    },
    {
      heading: "7. Безопасность",
      body: [
        "Пароли хранятся в необратимом виде (bcrypt); соединение по HTTPS; данные каждого ресторана в отдельной базе; ключи платёжных систем и SMS — в отдельной коллекции, не возвращаемой на сайт.",
        "Действия в панели управления записываются в журнал: кто, когда и что сделал.",
      ],
    },
    {
      heading: "8. Файлы cookie",
      body: [
        "На сайтах ресторанов используются только необходимые cookie: выбранный язык, выбранные бренд и филиал, корзина и сессия. Сторонние рекламные и трекинговые скрипты там не используются.",
        "На самом сайте keel.uz дополнительно используются инструменты статистики посещений и измерения результата нашей рекламы. Они используют адрес страницы и общие сведения о браузере и устройстве; имя, телефон и электронную почту на этой странице мы не спрашиваем.",
      ],
    },
    {
      heading: "9. Изменения и связь",
      body: [
        "При изменении политики новая редакция публикуется на этой странице с указанием даты обновления.",
        `Связь: ${COMPANY.email} · ${COMPANY.site}`,
        `Организация: ${COMPANY.nameRu}, ИНН ${COMPANY.inn}, ${COMPANY.addressRu}`,
      ],
    },
  ],
};

// ---- English ----

const offerEn: LegalDoc = {
  title: "Public offer",
  updated: UPDATED,
  intro: [
    `${COMPANY.nameEn} ("the Provider") offers the services of the Keel platform on the terms set out below.`,
    "Starting to use the service — creating an account, launching a site or making a payment — constitutes full acceptance of these terms.",
  ],
  sections: [
    {
      heading: "1. Definitions",
      body: [
        "— Platform — Keel: a website for a restaurant, order taking and a management panel.",
        "— Customer — the business using the platform (restaurant, shop, café).",
        "— Guest — a person ordering on the Customer's site.",
        "— Billing period — the period a charge is calculated for; normally one month.",
      ],
    },
    {
      heading: "2. What is provided",
      body: [
        "A website on its own domain; menu, orders and delivery; a management panel; a Telegram bot and mini app; apps for staff, couriers and the till.",
        "The service is provided as SaaS: the software runs on the Provider's servers and the Customer is granted the right to use it.",
        "Each Customer is given a separate database and a separately running instance.",
      ],
    },
    {
      heading: "3. Price and payment",
      body: [
        "The main charge is calculated from the number of accepted orders. The price per order and the monthly minimum are agreed with the Customer and stated on the invoice.",
        "Add-ons are charged separately, including removal of the “Powered by Keel” badge at 3,000,000 (three million) so'm per month, prorated by the days it was enabled.",
        "Bespoke site design is not part of the subscription and is agreed separately.",
        "Payment is made by transfer to the Provider's account or in cash. An invoice is issued for each period.",
        "A trial period may be granted, during which nothing is charged.",
      ],
    },
    {
      heading: "4. Obligations",
      body: [
        "The Provider: keeps the platform running, takes daily backups, ships updates and provides technical support.",
        "The Customer: is responsible for the content of the site (menu, prices, images, texts) and its lawfulness; keeps panel credentials confidential; deals with guests directly (orders, delivery, refunds).",
        "The Customer will not use the platform for unlawful activity or to publish material infringing the rights of others.",
      ],
    },
    {
      heading: "5. Ownership of data",
      body: [
        "The Customer's data — menu, orders, customer base, financial records — belongs to the Customer.",
        "On request, the data is provided in a machine-readable form (a JSON and image archive), including for migration to another system.",
        "The Provider does not sell the Customer's data and does not pass it to third parties except where required by law.",
      ],
    },
    {
      heading: "6. Third-party services",
      body: [
        "The platform works with external services: SMS gateways, payment providers (Payme, Click, Uzum, ATMOS), Telegram, map providers and POS systems.",
        "Contracts and payments for those are the Customer's own. The Provider is not responsible for their outages or changes to their terms.",
      ],
    },
    {
      heading: "7. Limitation of liability",
      body: [
        "The Provider aims for uninterrupted service but is not liable for outages caused by the internet, hosting, power or third-party services.",
        "The Provider's liability is limited to the amount paid for the last billing period.",
        "The Customer is responsible for everything done with its own credentials.",
      ],
    },
    {
      heading: "8. Suspension and termination",
      body: [
        "If payment is overdue, the Provider may suspend the service after notifying the Customer. Data is retained while suspended.",
        "The Customer may stop using the service at any time. The current period is not refunded, and data is provided under clause 5.",
      ],
    },
    {
      heading: "9. Disputes",
      body: [
        "Disputes are settled by negotiation. Failing agreement, they are heard in court under the law of the Republic of Uzbekistan.",
      ],
    },
    {
      heading: "10. Company details",
      body: [
        `Company: ${COMPANY.nameEn}`,
        `TIN: ${COMPANY.inn}`,
        `Activity code (OKED): ${COMPANY.oked}`,
        `Director: ${COMPANY.directorUz}`,
        `Address: ${COMPANY.addressEn}`,
        `Bank account: ${COMPANY.account}`,
        `Site: ${COMPANY.site}`,
      ],
    },
  ],
};

const privacyEn: LegalDoc = {
  title: "Privacy policy",
  updated: UPDATED,
  intro: [
    `${COMPANY.nameEn} processes personal data in accordance with the Personal Data Act of the Republic of Uzbekistan.`,
    "This policy covers the Keel platform and the restaurant sites running on it.",
  ],
  sections: [
    {
      heading: "1. Who controls the data",
      body: [
        "For a restaurant's guests, the restaurant is the data owner. The Provider processes that data on the restaurant's instruction, to deliver the service.",
        "For the platform's own customers — restaurant owners and their staff — the Provider is the data owner.",
      ],
    },
    {
      heading: "2. What is collected",
      body: [
        "— About a guest: name, phone number, delivery address and its note, order history, loyalty points, chosen language. When signing in through Telegram: the Telegram account id and name.",
        "— About restaurant staff: name, phone, login, schedule and shift records; location at the moment of clocking in or out, where the restaurant has enabled it.",
        "— Technical data: request logs, errors, service metrics.",
        "— Visit counting: page views are counted with an anonymous daily marker. The marker is hashed together with the date, so one browser cannot be followed across days.",
      ],
    },
    {
      heading: "3. What it is used for",
      body: [
        "Taking and delivering an order; contacting the guest about it; signing in by SMS or Telegram; loyalty points; statistics for the restaurant; running the service and finding faults.",
        "Marketing messages go only to guests who have not opted out. An opt-out is permanent and does not affect order-status messages.",
      ],
    },
    {
      heading: "4. Who it is shared with",
      body: [
        "Data goes to third-party services only as far as the service requires: the SMS gateway (a phone number for codes and messages), payment providers (order number and amount), Telegram (bot messages), the map provider (address coordinates), the POS system (order contents).",
        "Card numbers are never stored on or passed through the platform — payment happens on the provider's own page.",
        "Data is not sold and is not shared for advertising.",
      ],
    },
    {
      heading: "5. Retention",
      body: [
        "Orders and customer data are kept while the restaurant's account is active.",
        "Backups: daily copies for 14 days, a start-of-month copy for 180 days.",
        "Visit-counter rows are deleted automatically after 100 days.",
        "One-time SMS codes expire after 3 minutes and are deleted.",
      ],
    },
    {
      heading: "6. Your rights",
      body: [
        "The right to know your data, correct it and ask for its deletion. Name and addresses are editable in the profile; changing a phone number is confirmed by SMS.",
        "The right to opt out of marketing messages.",
        "Requests can go to the restaurant or to the Provider: " + COMPANY.email,
      ],
    },
    {
      heading: "7. Security",
      body: [
        "Passwords are stored irreversibly (bcrypt); connections use HTTPS; each restaurant's data lives in its own database; payment and SMS credentials are held in a separate collection that is never returned to the site.",
        "Actions in the management panel are written to an audit log: who, when and what.",
      ],
    },
    {
      heading: "8. Cookies",
      body: [
        "Restaurant sites use necessary cookies only: chosen language, chosen brand and branch, the cart and the session. No third-party advertising or tracking scripts run there.",
        "On keel.uz itself, visit statistics and tools measuring how our own advertising performs are also used. They work from the page address and general information about the browser and device; your name, phone number and email are not asked for on this page.",
      ],
    },
    {
      heading: "9. Changes and contact",
      body: [
        "When this policy changes, the new version is published on this page with its update date.",
        `Contact: ${COMPANY.email} · ${COMPANY.site}`,
        `Company: ${COMPANY.nameEn}, TIN ${COMPANY.inn}, ${COMPANY.addressEn}`,
      ],
    },
  ],
};

export const PUBLIC_OFFER: Record<string, LegalDoc> = {
  uz: offerUz,
  ru: offerRu,
  en: offerEn,
};

export const PRIVACY_POLICY: Record<string, LegalDoc> = {
  uz: privacyUz,
  ru: privacyRu,
  en: privacyEn,
};
