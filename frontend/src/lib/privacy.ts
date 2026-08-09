// The privacy notice each restaurant's site serves.
//
// ⚠️ **The restaurant is who the guest gives their data to, and this page says so.** The name,
// phone and address are typed in to get food delivered by *that* restaurant; Keel only runs the
// software it happens on. Pointing the cookie notice at keel.uz's own policy would have told the
// guest the wrong company holds their address — and it is the restaurant they will phone about
// it.
//
// ⚠️ **Not editable by the owner, and that is deliberate.** Every sentence here is a claim about
// what the code does — a 100-day expiry, an anonymous counter, no advertising scripts — and an
// owner given a text box would eventually publish a promise the software does not keep. What is
// filled in per restaurant is only what is true per restaurant: its name and how to reach it.
//
// ⚠️ **So a factual edit here is not a copy edit.** If the retention window, the counter or the
// list of third parties changes in the code, it changes here in the same commit. A privacy page
// that drifts from the software is worse than none: it is a written promise that is no longer
// kept.

import type { Lang } from "@/lib/i18n";

export type PrivacySection = { title: string; body: string[] };
export type PrivacyDoc = {
  title: string;
  intro: string;
  updated: string;
  sections: PrivacySection[];
};

/** `who` is the restaurant's own name; `contact` is how to reach it (phone, address). */
export function privacyDoc(lang: Lang, who: string, contact: string[]): PrivacyDoc {
  const reach = contact.length ? contact.join(" · ") : "";
  if (lang === "ru") return ru(who, reach);
  if (lang === "en") return en(who, reach);
  return uz(who, reach);
}

const UPDATED = "09.08.2026";

function uz(who: string, reach: string): PrivacyDoc {
  return {
    title: "Maxfiylik siyosati",
    updated: `Oxirgi yangilanish: ${UPDATED}`,
    intro: `Bu sahifa ${who} saytida qanday ma'lumot olinishini va u nima uchun kerakligini tushuntiradi. Ma'lumotni ${who} oladi va ishlatadi; sayt va ilova texnik jihatdan "Keel" platformasida ishlaydi.`,
    sections: [
      {
        title: "1. Qanday ma'lumot olinadi",
        body: [
          "— Hisob: ism va telefon raqami. Raqam SMS kod bilan tasdiqlanadi. Telegram orqali kirilsa, raqamni siz o'zingiz yuborganingizda olinadi.",
          "— Buyurtma: yetkazish manzili, xaritadagi nuqta, taomlarga yozgan izohlaringiz, to'lov turi va buyurtma tarkibi.",
          "— Stol bandligi: bron uchun ism, telefon, mehmonlar soni va vaqt.",
          "— Fikr: baho va izoh (siz yozsangiz).",
          "— Tashrif: sayt nechta odam ochganini anonim sanaymiz (pastda batafsil).",
        ],
      },
      {
        title: "2. Nima uchun kerak",
        body: [
          "Ism va telefon — buyurtmani yetkazish va zarur bo'lsa siz bilan bog'lanish uchun. Manzil va xaritadagi nuqta — kuryer yetib borishi va yetkazish narxini hisoblash uchun. Izohlar — oshxona uchun. Baho va fikr — xizmatni yaxshilash uchun.",
          "Reklama uchun xabar yuborilishi mumkin, lekin siz istalgan payt rad etsangiz, boshqa yuborilmaydi. Buyurtma holati va kirish kodlari bunga kirmaydi — ular siz so'ragan xizmat.",
        ],
      },
      {
        title: "3. Cookie va brauzerda saqlanadigan narsalar",
        body: [
          "— Tanlangan til, tanlangan brend va filial — cookie'da.",
          "— Savat, hisobga kirish (sessiya), tema va sevimli taomlar — brauzeringizning o'z xotirasida (localStorage). Ularsiz savat keyingi sahifada bo'shaydi va login yopishmaydi.",
          "— Stol raqami (QR skaner qilinganda) — faqat shu tashrif davomida.",
          "Reklama va kuzatuv skriptlari yo'q: Google Analytics, Yandex Metrika, Facebook pixel va shunga o'xshash hech narsa ishlatilmaydi.",
        ],
      },
      {
        title: "4. Tashriflar qanday sanaladi",
        body: [
          "Brauzeringiz o'zida tasodifiy belgi yaratadi. Serverga yuborilganda u sana bilan birga bir tomonlama shifrlanadi (hash), ya'ni ertaga o'sha brauzer boshqa yozuv bo'ladi — bu yozuvlar bilan odamni kunlar bo'ylab kuzatib bo'lmaydi.",
          "Yozuvlar 100 kundan keyin avtomatik o'chadi. Restoranga faqat umumiy raqam ko'rinadi: nechta odam kirdi va nechta sahifa ochildi.",
        ],
      },
      {
        title: "5. Kimga uzatiladi",
        body: [
          "Ma'lumot sotilmaydi. Faqat xizmatning o'zi uchun zarur bo'lganda uzatiladi:",
          "— To'lov tizimlari (karta bilan to'lasangiz) — buyurtma raqami va summa; karta ma'lumotini biz ko'rmaymiz, uni bank sahifasi oladi.",
          "— SMS xizmati — kod yoki xabar yuborish uchun raqam.",
          "— Tashqi yetkazish xizmati chaqirilsa — manzil va telefon.",
          "— Kassa (POS) tizimi ulangan bo'lsa — buyurtma tarkibi.",
          "— Telegram bot ishlatilsa — Telegram orqali yuborilgan xabarlar.",
        ],
      },
      {
        title: "6. Qancha saqlanadi",
        body: [
          "Buyurtmalar va bronlar hisobingiz mavjud bo'lgan davrda saqlanadi — sizga ham, restoranga ham tarix kerak. SMS kodlari 3 daqiqadan keyin o'chadi. Tashrif yozuvlari 100 kun.",
        ],
      },
      {
        title: "7. Sizning huquqlaringiz",
        body: [
          "Ismingizni va manzillaringizni profilda o'zingiz tahrirlaysiz, telefon raqamini yangi raqamga kelgan kod bilan o'zgartirasiz. Reklama xabarlaridan voz kechish uchun restoranga aytish yetarli.",
          "Ma'lumotingizni o'chirishni yoki nusxasini so'rash uchun restoranga murojaat qiling:",
          reach ? `— ${reach}` : "— restoranning aloqa sahifasidagi telefon orqali",
        ],
      },
      {
        title: "8. Xavfsizlik",
        body: [
          "Parollar qaytarib bo'lmaydigan ko'rinishda (hash) saqlanadi, sayt HTTPS orqali ishlaydi, har restoranning ma'lumoti alohida bazada turadi va boshqa restoran unga yeta olmaydi.",
        ],
      },
    ],
  };
}

function ru(who: string, reach: string): PrivacyDoc {
  return {
    title: "Политика конфиденциальности",
    updated: `Последнее обновление: ${UPDATED}`,
    intro: `На этой странице объясняется, какие данные собираются на сайте ${who} и зачем они нужны. Данные получает и использует ${who}; сайт технически работает на платформе «Keel».`,
    sections: [
      {
        title: "1. Какие данные собираются",
        body: [
          "— Аккаунт: имя и номер телефона. Номер подтверждается кодом из SMS. При входе через Telegram номер поступает только тогда, когда вы сами его отправляете.",
          "— Заказ: адрес доставки, точка на карте, ваши комментарии к блюдам, способ оплаты и состав заказа.",
          "— Бронь стола: имя, телефон, число гостей и время.",
          "— Отзыв: оценка и комментарий (если вы их оставите).",
          "— Посещения: анонимный подсчёт (подробнее ниже).",
        ],
      },
      {
        title: "2. Зачем это нужно",
        body: [
          "Имя и телефон — чтобы доставить заказ и связаться при необходимости. Адрес и точка на карте — чтобы курьер доехал и чтобы рассчитать стоимость доставки. Комментарии — для кухни. Оценка и отзыв — чтобы улучшать сервис.",
          "Возможны рекламные сообщения, но если вы откажетесь, они больше не приходят. Статусы заказа и коды входа сюда не относятся — это услуга, которую вы запросили.",
        ],
      },
      {
        title: "3. Cookie и данные в браузере",
        body: [
          "— Выбранный язык, выбранный бренд и филиал — в cookie.",
          "— Корзина, вход в аккаунт (сессия), тема и избранные блюда — в памяти самого браузера (localStorage). Без них корзина опустеет на следующей странице, а вход не сохранится.",
          "— Номер стола (при сканировании QR) — только на время этого визита.",
          "Рекламных и трекинговых скриптов нет: Google Analytics, Яндекс Метрика, Facebook pixel и подобные не используются.",
        ],
      },
      {
        title: "4. Как считаются посещения",
        body: [
          "Браузер создаёт у себя случайную метку. При отправке на сервер она необратимо хешируется вместе с датой — значит, завтра тот же браузер будет другой записью, и по этим записям нельзя следить за человеком из дня в день.",
          "Записи автоматически удаляются через 100 дней. Ресторан видит только общие числа: сколько человек зашло и сколько страниц открыто.",
        ],
      },
      {
        title: "5. Кому передаются",
        body: [
          "Данные не продаются. Передаются только там, где без этого не работает сама услуга:",
          "— Платёжные системы (при оплате картой) — номер заказа и сумма; данные карты мы не видим, их получает страница банка.",
          "— SMS-сервис — номер для отправки кода или сообщения.",
          "— Внешняя служба доставки, если её вызывают — адрес и телефон.",
          "— Касса (POS), если подключена — состав заказа.",
          "— Telegram, если используется бот — отправленные через него сообщения.",
        ],
      },
      {
        title: "6. Сроки хранения",
        body: [
          "Заказы и брони хранятся, пока существует ваш аккаунт — история нужна и вам, и ресторану. Коды из SMS удаляются через 3 минуты. Записи о посещениях — 100 дней.",
        ],
      },
      {
        title: "7. Ваши права",
        body: [
          "Имя и адреса вы редактируете в профиле сами, номер телефона меняете по коду, приходящему на новый номер. Чтобы отказаться от рекламных сообщений, достаточно сказать об этом ресторану.",
          "Для удаления данных или получения их копии обратитесь в ресторан:",
          reach ? `— ${reach}` : "— по телефону на странице контактов ресторана",
        ],
      },
      {
        title: "8. Безопасность",
        body: [
          "Пароли хранятся в необратимом виде (хеш), сайт работает по HTTPS, данные каждого ресторана лежат в отдельной базе, и другой ресторан не имеет к ним доступа.",
        ],
      },
    ],
  };
}

function en(who: string, reach: string): PrivacyDoc {
  return {
    title: "Privacy policy",
    updated: `Last updated: ${UPDATED}`,
    intro: `This page explains what data is collected on the ${who} website and why it is needed. ${who} receives and uses the data; the site runs technically on the Keel platform.`,
    sections: [
      {
        title: "1. What is collected",
        body: [
          "— Account: your name and phone number. The number is confirmed by an SMS code. If you sign in through Telegram, the number arrives only when you send it yourself.",
          "— Order: delivery address, the point on the map, your notes on dishes, payment method and the contents of the order.",
          "— Table booking: name, phone, number of guests and the time.",
          "— Feedback: a rating and a comment, if you leave one.",
          "— Visits: an anonymous count (details below).",
        ],
      },
      {
        title: "2. Why it is needed",
        body: [
          "Name and phone: to deliver the order and to reach you if something is unclear. Address and map point: so the courier arrives and so the delivery fee can be calculated. Notes: for the kitchen. Rating and comment: to improve the service.",
          "Marketing messages are possible, but once you opt out they stop. Order updates and sign-in codes are not marketing — they are the service you asked for.",
        ],
      },
      {
        title: "3. Cookies and browser storage",
        body: [
          "— Chosen language, chosen brand and branch: in cookies.",
          "— Cart, session, theme and favourite dishes: in your own browser's storage. Without them the cart empties on the next page and signing in does not stick.",
          "— Table number (when a QR code is scanned): only for this visit.",
          "There are no advertising or tracking scripts: no Google Analytics, no Yandex Metrica, no Facebook pixel, nothing of that kind.",
        ],
      },
      {
        title: "4. How visits are counted",
        body: [
          "Your browser creates a random marker and keeps it. When it reaches the server it is hashed one-way together with the date, so tomorrow the same browser is a different row — these rows cannot be used to follow a person from one day to the next.",
          "Rows are deleted automatically after 100 days. The restaurant sees only totals: how many people came and how many pages they opened.",
        ],
      },
      {
        title: "5. Who it is shared with",
        body: [
          "Data is not sold. It is passed on only where the service itself does not work otherwise:",
          "— Payment providers, if you pay by card: the order number and the amount. We never see your card details; the bank's own page takes them.",
          "— SMS gateway: the number, to send a code or a message.",
          "— An external delivery service, if one is called: the address and phone.",
          "— The till (POS), if connected: the contents of the order.",
          "— Telegram, if the bot is used: the messages sent through it.",
        ],
      },
      {
        title: "6. How long it is kept",
        body: [
          "Orders and bookings are kept while your account exists — both you and the restaurant need the history. SMS codes are deleted after 3 minutes. Visit rows after 100 days.",
        ],
      },
      {
        title: "7. Your rights",
        body: [
          "You edit your name and addresses in your profile yourself, and change your phone number with a code sent to the new one. To stop marketing messages, telling the restaurant is enough.",
          "To have your data deleted, or to get a copy of it, contact the restaurant:",
          reach ? `— ${reach}` : "— by the phone number on the restaurant's contact page",
        ],
      },
      {
        title: "8. Security",
        body: [
          "Passwords are stored irreversibly (hashed), the site runs over HTTPS, each restaurant's data lives in its own database, and no other restaurant can reach it.",
        ],
      },
    ],
  };
}
