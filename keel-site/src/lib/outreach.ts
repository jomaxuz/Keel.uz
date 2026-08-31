// The messages sent to a restaurant that has never heard of us.
//
// ⚠️ **Written by hand, not generated.** The obvious build is to ask the
// assistant for a message each time — the plumbing for that already exists
// (`control/internal/handlers/campaigntext.go`). It was the wrong answer here
// for three reasons, and the first one is the one that matters: a model writes
// in a cadence people in this market now recognise on sight. "Здравствуйте!
// Меня зовут… Я хотел бы предложить вам…" is read as spam before the second
// line, and the whole problem being solved is that these messages get ignored.
// The other two: every send would cost money we are trying not to spend, and a
// wording that is right would be re-rolled into a different wording tomorrow.
//
// ⚠️ **The specific line is left to the human, on purpose.** `{note}` is where
// the sender writes what they actually noticed — "Instagramda menyungizni
// ko'rdim", "do'stim sizdan buyurtma qilgan ekan". That one sentence is the
// entire difference between a message that gets a reply and a template. No
// generator can supply it, because it is a fact about that restaurant that only
// the person writing knows.
//
// ⚠️ **Prices are quoted from the landing page and must move with it**:
// 450 000 so'm a month for the till (`t.hero.lead`), 800 so'm per online order
// on the first tier (`t.pricing.tiers[0]`). A message that quotes a price the
// site contradicts is a conversation that starts with an apology.

export type OutreachLang = "uz" | "ru";

/** Who the message is for.
 *
 *  ⚠️ Three segments because three different things are true of them, not
 *  because three is a tidy number:
 *
 *   • `telegram` — takes delivery orders through a Telegram admin. Has a real,
 *     daily, describable pain and nothing to rip out.
 *   • `opening` — opening soon. Buying everything at once and actively looking,
 *     which is the only moment "we have that already" cannot be the answer.
 *   • `running` — already runs iiko, Poster, Jowi, Delever, Zoomda. The hard
 *     one, and the one where the pitch has to *start* by saying we are not
 *     asking them to replace it — otherwise the first line is the last one
 *     they read.
 */
export type OutreachKind = "telegram" | "opening" | "running";

/** First contact, or the message that actually gets the replies. */
export type OutreachStage = "first" | "follow";

export type OutreachSlots = {
  /** The restaurant's name. */
  name: string;
  /** What they run today — only used by `running`. */
  current: string;
  /** The one sentence the sender writes themselves. Dropped when empty. */
  note: string;
};

type Bank = Record<
  OutreachKind,
  Record<OutreachLang, { first: string[]; follow: string[] }>
>;

const BANK: Bank = {
  // ───────────────────────── Delivery through a Telegram admin
  telegram: {
    uz: {
      first: [
        `Assalomu alaykum. {name} bo'yicha yozyapman.

{note}

Buyurtma Telegramdan kelsa, admin har birini qo'lda yozib oladi — manzil, taomlar, telefon. Kechqurun band paytda xato aynan shu yerda chiqadi.

Bizda mijoz saytdan yoki Telegram mini appdan buyurtma beradi: taom tanlangan, manzil xaritada, summa hisoblangan. Sizga tayyor buyurtma bo'lib tushadi.

Har buyurtma uchun 800 so'm, bekor qilingani bepul. Agregatorga 15–20% berish shart emas.

Ko'rsatib beray, ikki daqiqa oladi?`,

        `Salom. {name} ning yetkazib berishi haqida.

{note}

Hozir buyurtmalar Telegramda admin orqali ketyapti shekilli. Ishlaydi, lekin ikki narsa yo'qoladi: qaysi taom qancha sotilgani va mijozlar ro'yxati — hammasi yozishmada qolib ketadi.

Sayt va Telegram mini app qilib beramiz, menyuni o'zimiz kiritamiz. Buyurtma o'zi tushadi, mijoz bazasi sizda qoladi.

Abonent to'lovi yo'q — faqat kelgan buyurtma uchun 800 so'm.

Havolani tashlaymi?`,

        `Assalomu alaykum!

{note}

{name} uchun sayt va Telegram mini app qilamiz — buyurtma adminsiz, to'g'ridan-to'g'ri oshxonaga tushadi. Menyuni o'zimiz kiritamiz, sizdan hech nima talab qilinmaydi.

Narxi: kelgan buyurtma uchun 800 so'm. Abonent to'lovi yo'q.

Qiziqsangiz, ishlab turgan namunani tashlayman.`,
      ],
      follow: [
        `Assalomu alaykum. O'tgan safar {name} uchun sayt va Telegram mini app haqida yozgandim.

Vaqtingiz bo'lsa namunani tashlayman — ko'rib, keraksiz desangiz boshqa bezovta qilmayman.`,
      ],
    },
    ru: {
      first: [
        `Здравствуйте. Пишу по поводу {name}.

{note}

Если заказы приходят в Telegram, администратор переписывает каждый вручную — адрес, блюда, телефон. Вечером в загрузку ошибки появляются именно здесь.

У нас гость заказывает на сайте или в Telegram mini app: блюда выбраны, адрес на карте, сумма посчитана. Вам приходит готовый заказ.

800 сум за заказ, отменённые бесплатно. Без 15–20% агрегатора.

Показать? Займёт пару минут.`,

        `Добрый день. По доставке {name}.

{note}

Сейчас заказы идут через администратора в Telegram. Работает, но две вещи теряются: что и сколько продаётся и база гостей — всё остаётся в переписке.

Сделаем сайт и Telegram mini app, меню заведём сами. Заказ приходит сам, база остаётся у вас.

Абонентской платы нет — только 800 сум за пришедший заказ.

Скинуть ссылку?`,

        `Здравствуйте!

{note}

Сделаем для {name} сайт и Telegram mini app — заказ попадает на кухню без администратора. Меню заведём сами, от вас ничего не нужно.

800 сум за заказ, абонентской платы нет.

Если интересно, пришлю рабочий пример.`,
      ],
      follow: [
        `Здравствуйте. Писал по поводу сайта и Telegram mini app для {name}.

Если будет минута — пришлю пример. Посмотрите, и если не нужно, больше не побеспокою.`,
      ],
    },
  },

  // ───────────────────────── Opening soon
  opening: {
    uz: {
      first: [
        `Assalomu alaykum. {name} ochilyapti deb eshitdim.

{note}

Ochilishdan oldin bitta narsani hal qilib qo'ysangiz, keyin ancha oson bo'ladi: kassa. Ofitsiant qog'ozga yozmaydi, kun oxirida tushum o'zi chiqadi, ombor va tannarx ham shu yerda.

Sayt, yetkazib berish va Telegram bot ham shu menyudan ishlaydi — alohida qildirish shart emas.

Oyiga 450 000 so'mdan. 14 kun bepul, menyuni o'zimiz kiritamiz.

Ochilishgacha ulab beray, bo'ladimi?`,

        `Salom! {name} yaqinda ochiladi deb bildim.

{note}

Yangi joyda kassa masalasi odatda oxiriga qoladi va ochilgan kuni daftar bilan ishlanadi. Keyin uni almashtirish qiyinroq bo'ladi.

Bizda kassa, zal, oshxona ekrani, ombor va sayt — bitta dasturda. iiko'dan 33–63% arzon, oyiga 450 000 so'mdan.

14 kun bepul sinab ko'rasiz, menyuni biz kiritamiz. Ochilishga ulguramiz.

Bugun-ertaga gaplashsak bo'ladimi?`,

        `Assalomu alaykum.

{note}

{name} uchun kassa kerak bo'lsa yozing: kassa, zal, oshxona ekrani, ombor — ustiga sayt va Telegram bot. Hammasi bitta joydan.

Oyiga 450 000 so'mdan, 14 kun bepul. Menyuni o'zimiz kiritamiz va ochilishgacha ishga tushiramiz.

Namunani ko'rsataymi?`,
      ],
      follow: [
        `Assalomu alaykum. {name} ning ochilishi bilan bog'liq yozgandim.

Ulgurish uchun bir hafta yetadi. Qiziqsangiz yozing — bo'lmasa boshqa bezovta qilmayman.`,
      ],
    },
    ru: {
      first: [
        `Здравствуйте. Слышал, что {name} скоро открывается.

{note}

Если решить один вопрос до открытия, дальше будет заметно проще: касса. Официант не пишет на бумаге, выручка за день считается сама, склад и себестоимость там же.

Сайт, доставка и Telegram-бот работают из того же меню — отдельно заказывать не нужно.

От 450 000 сум в месяц. 14 дней бесплатно, меню заведём сами.

Успеем подключить до открытия — обсудим?`,

        `Добрый день! Узнал, что {name} готовится к открытию.

{note}

В новом заведении касса обычно откладывается на последний момент, и открываются с тетрадью. Менять потом тяжелее.

У нас касса, зал, кухонный экран, склад и сайт — в одной программе. На 33–63% дешевле iiko, от 450 000 сум в месяц.

14 дней бесплатно, меню заводим мы. К открытию успеваем.

Можем сегодня-завтра созвониться?`,

        `Здравствуйте.

{note}

Если для {name} нужна кассовая программа — напишите: касса, зал, кухонный экран, склад, плюс сайт и Telegram-бот. Всё из одного места.

От 450 000 сум в месяц, 14 дней бесплатно. Меню заведём и запустим до открытия.

Показать пример?`,
      ],
      follow: [
        `Здравствуйте. Писал в связи с открытием {name}.

Чтобы успеть к открытию, хватит недели. Если интересно — напишите, если нет — не буду беспокоить.`,
      ],
    },
  },

  // ───────────────────────── Already runs something
  running: {
    uz: {
      first: [
        `Assalomu alaykum. {name} bo'yicha.

{note}

Sizda {current} borligini bilaman va uni almashtiring demayapman.

Bitta savol: onlayn buyurtmalar uchun oyiga qancha komissiya ketyapti? Agregatorda 15–20%, alohida xizmatlarda abonent to'lovi bor.

Bizda o'z saytingiz va Telegram mini appingiz bo'ladi, buyurtma uchun 800 so'm — abonent to'lovisiz. Kassangizga ulanamiz, ya'ni buyurtma to'g'ridan-to'g'ri {current} ga tushadi.

Oyiga qancha chiqishini hisoblab beraymi? Buyurtmalar sonini aytsangiz kifoya.`,

        `Salom. {name} ga taklif.

{note}

{current} ni almashtirish haqida emas — u qolaveradi.

Gap onlayn buyurtmada: hozir agregator yoki alohida xizmat orqali bo'lsa, har buyurtmadan foiz yoki oylik to'lov ketadi. Bizda o'z saytingiz bo'ladi, buyurtma {current} ga o'zi tushadi, narx — kelgan buyurtma uchun 800 so'm.

Oyiga 500 ta buyurtma bo'lsa, bu 400 ming so'm. Agregatorda o'sha hajm bir necha million turadi.

Solishtirib ko'ramizmi?`,

        `Assalomu alaykum.

{note}

{name} da {current} ishlayotganini ko'rdim. Yaxshi tizim, unga tegmaymiz.

Biz boshqa joyda foydalimiz: o'z saytingiz va Telegram mini app — mijoz agregatorsiz to'g'ridan-to'g'ri sizga buyurtma beradi. Buyurtma {current} ga ulanadi.

Abonent to'lovi yo'q, kelgan buyurtma uchun 800 so'm, bekor qilingani bepul.

Ishlab turgan namunani ko'rsataymi?`,
      ],
      follow: [
        `Assalomu alaykum. {name} uchun onlayn buyurtma haqida yozgandim — {current} ga tegmasdan.

Oyiga taxminan nechta buyurtma olasiz? Bitta raqam aytsangiz, qancha tejashingizni hisoblab beraman.`,
      ],
    },
    ru: {
      first: [
        `Здравствуйте. По {name}.

{note}

Знаю, что у вас {current}, и менять её не предлагаю.

Один вопрос: сколько в месяц уходит на комиссию за онлайн-заказы? У агрегаторов 15–20%, у отдельных сервисов — абонентская плата.

У вас будет свой сайт и Telegram mini app, 800 сум за заказ, без абонентской платы. К кассе подключимся — заказ будет падать прямо в {current}.

Посчитать, сколько выходит в месяц? Достаточно назвать число заказов.`,

        `Добрый день. Предложение для {name}.

{note}

Речь не о замене {current} — она остаётся.

Вопрос в онлайн-заказах: сейчас за них платится процент агрегатору или абонентка отдельному сервису. У вас будет свой сайт, заказ падает в {current} сам, цена — 800 сум за пришедший заказ.

500 заказов в месяц — это 400 000 сум. У агрегатора тот же объём стоит несколько миллионов.

Сравним?`,

        `Здравствуйте.

{note}

Видел, что в {name} работает {current}. Хорошая система, трогать не будем.

Мы полезны в другом: свой сайт и Telegram mini app — гость заказывает напрямую, без агрегатора. Заказ подключается к {current}.

Абонентской платы нет, 800 сум за пришедший заказ, отменённые бесплатно.

Показать рабочий пример?`,
      ],
      follow: [
        `Здравствуйте. Писал про онлайн-заказы для {name} — без изменений в {current}.

Сколько примерно заказов в месяц? По одной цифре посчитаю, сколько получится сэкономить.`,
      ],
    },
  },
};

/** How many variants exist for this choice, so the screen can say "1 / 3". */
export function variantCount(
  kind: OutreachKind,
  lang: OutreachLang,
  stage: OutreachStage,
): number {
  return BANK[kind][lang][stage].length;
}

/** Fills the slots and returns the message ready to send.
 *
 *  ⚠️ **A line holding an empty `{note}` is dropped, not left blank.** That
 *  slot is a whole sentence of its own, so removing the line is safe — and the
 *  alternative, an empty paragraph in the middle of a message, is the single
 *  most obvious sign that something was generated.
 *
 *  ⚠️ **An empty name becomes a neutral word rather than a hole.** "Пишу по
 *  поводу ." is worse than a slightly generic opening, and it is exactly the
 *  kind of thing that gets sent at eleven at night without being re-read.
 */
export function outreachText(
  kind: OutreachKind,
  lang: OutreachLang,
  stage: OutreachStage,
  variant: number,
  slots: OutreachSlots,
): string {
  const list = BANK[kind][lang][stage];
  const raw = list[((variant % list.length) + list.length) % list.length];

  const name = slots.name.trim();
  const current = slots.current.trim();
  const note = slots.note.trim();

  const fallbackName = lang === "uz" ? "restoraningiz" : "вашего заведения";
  const fallbackCurrent = lang === "uz" ? "kassangiz" : "ваша касса";

  return raw
    .split("\n")
    .filter((line) => !(line.includes("{note}") && note === ""))
    .join("\n")
    .replaceAll("{note}", note)
    .replaceAll("{name}", name || fallbackName)
    .replaceAll("{current}", current || fallbackCurrent)
    // Two blank lines are what a dropped `{note}` leaves behind.
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}

/** What people around here actually run, for the "already has something"
 *  segment. Free text is still allowed — this list is a shortcut, not a claim
 *  that it is complete. */
export const RUNNING_SYSTEMS = [
  "iiko",
  "Poster",
  "Jowi",
  "Delever",
  "Zoomda",
  "r_keeper",
  "Clopos",
];
