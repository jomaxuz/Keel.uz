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
// ⚠️ **Being hand-written is not enough on its own — the first bank still read
// as generated**, because every message had the same shape: greeting, a
// paragraph naming the pain, a paragraph naming the fix, a line with the price,
// a polite closing question. Five blocks, same order, same length, every time.
// That regularity is the tell, not the vocabulary. So this bank is deliberately
// uneven:
//
//   • **Length varies from two lines to eight.** A message that fills a phone
//     screen is scrolled past.
//   • **Not every message pitches.** Some ask one question and stop — a
//     question that can be answered with one word is answered far more often
//     than an offer that needs a decision.
//   • **Not every message names a price.** Quoting money before anyone has
//     shown interest answers a question nobody asked.
//   • **The closings differ**, and some are not questions at all.
//   • **Numbers, not adjectives**: "800 so'm" and "15–20%" belong here,
//     "qulay" and "zamonaviy" do not.
//
// ⚠️ **Eight first-contact variants, not three.** Three is enough to look
// varied on the screen and not enough to survive an evening: a person writing
// to twenty places works through them in the first hour, and after that every
// message is a repeat — which is exactly what gets an account limited and what
// two neighbouring restaurants notice when they compare notes.
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
 *   • `running` — already runs iiko, Poster, Jowi, Delever, Zoomda.
 *
 *  ⚠️ **The `running` messages no longer open by promising not to replace
 *  anything.** They used to: "sizda iiko borligini bilaman va uni almashtiring
 *  demayapman". That line answers an objection the reader has not made yet, and
 *  raising it first is what plants it. Worse, it makes the whole message a
 *  request for permission to exist beside something bigger — and nobody buys
 *  from the party that opened by conceding.
 *
 *  What is actually true is narrower and more useful: a restaurant running iiko
 *  is already taking online orders through Delever or Zoomda, or through an
 *  aggregator taking 15–20%. That is the thing being compared against, and it
 *  is a thing with a monthly cost, an owner of the guest list, and a domain
 *  somebody else's name is on. So these messages compare *that*, in numbers,
 *  and mention the till exactly once — as a fact about where the order lands,
 *  never as reassurance. */
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
        `Assalomu alaykum. {name}'da yetkazib berish bo'yicha savolim bor.

{note}

Buyurtmani admin qo'lda yozib oladimi, yoki saytdan tushadimi?`,

        `Salom. {name} bo'yicha.

{note}

Agregatorlar har buyurtmadan 15–20% oladi. Bizda o'z saytingiz va Telegram mini appingiz bo'ladi, narxi — kelgan buyurtma uchun 800 so'm.

Oyiga taxminan nechta buyurtma olasiz? Farqini hisoblab beraman.`,

        `Assalomu alaykum.

{note}

Kechqurun band paytda admin manzilni noto'g'ri yozib qo'yadi, kuryer adashadi, ovqat sovuq boradi. Bizda mijoz manzilni xaritada o'zi belgilaydi.

Ko'rsatay, ikki daqiqa oladi.`,

        `Salom. {name} uchun sayt va Telegram mini app namunasini tashlasam bo'ladimi? Ko'rasiz, keraksiz desangiz shu bilan tugadi.

{note}`,

        `Assalomu alaykum. {name} bo'yicha yozyapman.

{note}

Telegramda buyurtma olinganda mijozlar ro'yxati yozishmada qolib ketadi: kim ikki marta buyurtma qilgan, kim uch oydan beri yo'q — bilib bo'lmaydi.

Bizda har mijoz bazada turadi va baza sizniki bo'ladi. Menyuni o'zimiz kiritamiz.`,

        `Salom. {name}'ga o'z sayti va Telegram mini appi kerak emasmi? Abonent to'lovi yo'q, faqat kelgan buyurtma uchun 800 so'm.

{note}`,

        `Assalomu alaykum.

{note}

Adminingiz bir kechada nechta buyurtmani qo'lda yozadi? Har biri ikki-uch daqiqa, ustiga xato qilish ehtimoli.

Sayt va mini appdan buyurtma tayyor holda tushadi — admin faqat tasdiqlaydi.`,

        `Salom aleykum. {name} bo'yicha.

{note}

Restoranlar uchun sayt va Telegram mini app qilamiz, menyuni o'zimiz kiritamiz, bir kunda ishga tushadi.

Havolasini tashlaymi?`,
      ],
      follow: [
        `Assalomu alaykum. O'tgan hafta {name} uchun sayt va Telegram mini app haqida yozgandim. Namunani tashlaymi?`,

        `Salom. Javob bo'lmadi — band bo'lgandirsiz.

Bitta savol: onlayn buyurtma sizga qiziqmi yoki umuman kerak emasmi? Qaysi biri bo'lsa ham ayting, shunga qarab bezovta qilmayman.`,

        `Assalomu alaykum. Xabarim yo'qolgan bo'lsa kerak deb qayta yozyapman.

Menyuni biz kiritamiz, sizdan vaqt ketmaydi. Bir ko'rib chiqasizmi?`,
      ],
    },
    ru: {
      first: [
        `Здравствуйте. Вопрос по доставке {name}.

{note}

Заказы администратор переписывает вручную или они приходят с сайта?`,

        `Добрый день. По {name}.

{note}

Агрегаторы берут 15–20% с каждого заказа. У вас будет свой сайт и Telegram mini app, цена — 800 сум за пришедший заказ.

Сколько примерно заказов в месяц? Посчитаю разницу.`,

        `Здравствуйте.

{note}

Вечером в загрузку администратор ошибается в адресе, курьер плутает, еда приезжает холодной. У нас гость сам ставит точку на карте.

Показать? Займёт две минуты.`,

        `Добрый день. Можно прислать пример сайта и Telegram mini app для {name}? Посмотрите, если не нужно — на этом и закончим.

{note}`,

        `Здравствуйте. Пишу по поводу {name}.

{note}

Когда заказы идут через Telegram, база гостей остаётся в переписке: кто заказывал дважды, кто пропал три месяца назад — не видно.

У нас каждый гость в базе, и база ваша. Меню заводим сами.`,

        `Добрый день. {name} нужен свой сайт и Telegram mini app? Абонентской платы нет, только 800 сум за пришедший заказ.

{note}`,

        `Здравствуйте.

{note}

Сколько заказов ваш администратор переписывает за вечер? Каждый — две-три минуты плюс шанс ошибиться.

С сайта и mini app заказ приходит готовым, администратор только подтверждает.`,

        `Здравствуйте. По {name}.

{note}

Делаем заведениям сайт и Telegram mini app, меню заводим сами, запуск за день.

Скинуть ссылку?`,
      ],
      follow: [
        `Здравствуйте. На прошлой неделе писал про сайт и Telegram mini app для {name}. Прислать пример?`,

        `Добрый день. Ответа не было — наверное, было не до того.

Один вопрос: онлайн-заказы вам интересны или это совсем не ваша тема? Скажите как есть, дальше писать не буду.`,

        `Здравствуйте. Пишу повторно, вдруг сообщение потерялось.

Меню заводим мы, вашего времени это не займёт. Посмотрите?`,
      ],
    },
  },

  // ───────────────────────── Opening soon
  opening: {
    uz: {
      first: [
        `Assalomu alaykum. {name} ochilyapti deb eshitdim.

{note}

Qachonga rejalashtirgansiz? Bir haftadan ko'p bo'lsa, kassani birinchi mehmongacha ulab beramiz.`,

        `Salom.

{note}

Yangi joyda odatda daftar bilan ochiladi, kassa esa bir oydan keyin qo'yiladi. Zalni qayta o'rgatish esa boshidan o'rgatishdan qiyinroq.

Kassa, zal, oshxona ekrani, ombor — oyiga 450 000 so'mdan. 14 kun bepul, menyuni o'zimiz kiritamiz.`,

        `Assalomu alaykum! {name} ochilishi haqida bildim.

{note}

Sayt, yetkazib berish va Telegram bot kassa bilan bitta menyudan ishlaydi — alohida qildirish shart emas.

Qanday ko'rinishini ko'rsataymi?`,

        `Salom. {name} uchun kassani tanlab bo'ldingizmi?

{note}`,

        `Assalomu alaykum.

{note}

Kassa, zal, oshxona, ombor, sayt va yetkazish — bitta tizimda, oyiga 450 000 so'mdan, 14 kun bepul.

Ochilishga kassa kerak bo'lsa yozing — ulguramiz.`,

        `Salom. {name}'ning ochilishi bo'yicha.

{note}

Menyuni biz kiritamiz — sizdan faqat taomlar va narxlar ro'yxati kerak, rasmga olib tashlasangiz ham bo'ladi.

Qolganini o'zimiz qilamiz: kassa, zal, oshxona, sayt.`,

        `Assalomu alaykum.

{note}

Kassa va yetkazib berish bo'yicha nima hal qilingan? Hali hech nima bo'lmasa, o'n daqiqada o'zimiznikini ko'rsataman.`,

        `Salom! {name} ochilyapti ekan, muborak bo'lsin.

{note}

Kassa, sayt va yetkazib berish bitta joydan kerak bo'lsa — yozing. 14 kun bepul, menyuni o'zimiz kiritamiz.`,
      ],
      follow: [
        `Assalomu alaykum. {name}'ning ochilishi bilan bog'liq yozgandim. Qaysi kunga rejalashtirdingiz? Bir hafta bo'lsa ham ulguramiz.`,

        `Salom. Kassa masalasi hal bo'ldimi? Bo'lgan bo'lsa ham ayting — ro'yxatdan o'chiraman va boshqa bezovta qilmayman.`,

        `Assalomu alaykum. 14 kunlik bepul sinovni ochilishdan oldin boshlab qo'ysangiz, birinchi mehmon kelganda zal allaqachon o'rgangan bo'ladi.

Boshlaymizmi?`,
      ],
    },
    ru: {
      first: [
        `Здравствуйте. Слышал, {name} скоро открывается.

{note}

На какое число планируете? Если больше недели — успеем поставить кассу до первого гостя.`,

        `Добрый день.

{note}

Новые заведения обычно открываются с тетрадью, а кассу ставят через месяц. Переучивать зал потом тяжелее, чем научить сразу.

Касса, зал, кухонный экран, склад — от 450 000 сум в месяц. 14 дней бесплатно, меню заводим сами.`,

        `Здравствуйте! Узнал про открытие {name}.

{note}

Сайт, доставка и Telegram-бот работают из того же меню, что и касса — заказывать отдельно не нужно.

Показать, как это выглядит?`,

        `Добрый день. Кассу для {name} уже выбрали?

{note}`,

        `Здравствуйте.

{note}

Касса, зал, кухня, склад, сайт и доставка — в одной системе, от 450 000 сум в месяц, 14 дней бесплатно.

Если к открытию нужна касса — напишите, успеваем.`,

        `Добрый день. По открытию {name}.

{note}

Меню заведём мы — от вас нужен только список блюд с ценами, хоть фотографией.

Остальное на нас: касса, зал, кухня, сайт.`,

        `Здравствуйте.

{note}

Что уже решено по кассе и доставке? Если пока ничего — за десять минут покажу наш вариант.`,

        `Здравствуйте! {name} открывается — поздравляю.

{note}

Если касса, сайт и доставка нужны из одного места — напишите. 14 дней бесплатно, меню заводим сами.`,
      ],
      follow: [
        `Здравствуйте. Писал по открытию {name}. На какое число назначили? Даже за неделю успеем.`,

        `Добрый день. С кассой уже определились? Если да — скажите, уберу из списка и больше не побеспокою.`,

        `Здравствуйте. Если запустить бесплатные 14 дней до открытия, к первому гостю зал уже будет обучен.

Начнём?`,
      ],
    },
  },

  // ───────────────────────── Already runs something
  running: {
    uz: {
      first: [
        `Assalomu alaykum. {name} bo'yicha.

{note}

Onlayn buyurtmalar uchun oyiga qancha to'laysiz? Agregatorda 15–20%, alohida xizmatlarda abonent to'lovi.

Bizda kelgan buyurtma uchun 800 so'm, boshqa to'lov yo'q. Buyurtma to'g'ridan-to'g'ri {current}'ga tushadi.

Oyiga nechta buyurtma olasiz? Farqini aniq hisoblab beraman.`,

        `Salom. {name}'ga.

{note}

Agregatordan kelgan mijoz — agregatorning mijozi. Telefon raqami ham, buyurtma tarixi ham sizda qolmaydi.

O'z saytingiz va Telegram mini appingiz bo'lsa, baza sizniki bo'ladi. Buyurtma {current}'ga tushadi.

Namunani ko'rsataymi?`,

        `Assalomu alaykum. {name}'ning o'z sayti va Telegram mini appi bormi?

{note}`,

        `Salom.

{note}

Oyiga 500 ta onlayn buyurtma bo'lsa, bizda 400 ming so'm chiqadi. Agregatorda o'sha hajm bir necha million.

Buyurtma {current}'ga tushadi, ishlash tartibingiz o'zgarmaydi.

Solishtirib ko'ramizmi?`,

        `Assalomu alaykum. {name} bo'yicha.

{note}

Hozir yetkazib berishni kim yuritadi — Delever, Zoomda yoki agregatormi? Oyiga qancha ketishini aytsangiz, taqqoslab beraman.

Bizda buyurtma uchun 800 so'm, abonent to'lovi yo'q.`,

        `Salom. {name}'ga Telegram mini app kerak emasmi? Mijoz botdan chiqmasdan buyurtma beradi, buyurtma {current}'ga tushadi.

{note}

Ishlab turgan namunasi bor, tashlaymi?`,

        `Assalomu alaykum.

{note}

Bitta savol: onlayn buyurtma tomonini kim yuritadi va oyiga qancha turadi?

Javobingizga qarab bizda qancha bo'lishini aniq aytaman — taxminan emas.`,

        `Salom. {name} bo'yicha.

{note}

Google'da restoraningizni qidirgan odam agregator sahifasiga tushadi, u yerda esa yonida yana o'nta restoran turadi.

O'z saytingiz bo'lsa, o'sha odam to'g'ridan-to'g'ri sizga keladi. Buyurtma {current}'ga ulanadi.

Ko'rsatay?`,
      ],
      follow: [
        `Assalomu alaykum. {name} uchun onlayn buyurtma haqida yozgandim.

Oyiga taxminan nechta buyurtma olasiz? Bitta raqam aytsangiz, farqini aniq hisoblab beraman.`,

        `Salom. Menga faqat ikkita narsa kerak: hozir yetkazib berishni kim yuritadi va oyiga qancha turadi.

Bizda arzonroq chiqmasa, o'zim shunday deb aytaman.`,

        `Assalomu alaykum. Namunani tashlab qo'yay, vaqtingiz bo'lganda ochib ko'rasiz.

Buyurtma {current}'ga tushadi, ya'ni ishlash tartibingiz o'zgarmaydi — gap faqat onlayn buyurtmada.`,
      ],
    },
    ru: {
      first: [
        `Здравствуйте. По {name}.

{note}

Сколько в месяц уходит на онлайн-заказы? У агрегаторов 15–20%, у отдельных сервисов — абонентская плата.

У нас 800 сум за пришедший заказ и больше ничего. Заказ падает прямо в {current}.

Сколько заказов в месяц? Посчитаю разницу точно.`,

        `Добрый день. Для {name}.

{note}

Гость, пришедший через агрегатор, — гость агрегатора. Ни телефона, ни истории заказов у вас не остаётся.

Со своим сайтом и Telegram mini app база остаётся у вас. Заказ падает в {current}.

Показать пример?`,

        `Здравствуйте. У {name} есть свой сайт и Telegram mini app?

{note}`,

        `Добрый день.

{note}

500 онлайн-заказов в месяц — это у нас 400 000 сум. У агрегатора тот же объём стоит несколько миллионов.

Заказ падает в {current}, порядок работы не меняется.

Сравним?`,

        `Здравствуйте. По {name}.

{note}

Кто сейчас ведёт доставку — Delever, Zoomda или агрегатор? Скажите, сколько это в месяц, и я сравню.

У нас 800 сум за заказ, абонентской платы нет.`,

        `Добрый день. {name} нужен Telegram mini app? Гость заказывает, не выходя из бота, заказ падает в {current}.

{note}

Есть рабочий пример, прислать?`,

        `Здравствуйте.

{note}

Один вопрос: кто ведёт онлайн-заказы и сколько это стоит в месяц?

По вашему ответу назову нашу цифру — точную, не примерную.`,

        `Добрый день. По {name}.

{note}

Человек, который ищет вас в Google, попадает на страницу агрегатора, где рядом ещё десять заведений.

Со своим сайтом он приходит прямо к вам. Заказ падает в {current}.

Показать?`,
      ],
      follow: [
        `Здравствуйте. Писал про онлайн-заказы для {name}.

Сколько примерно заказов в месяц? По одной цифре посчитаю разницу точно.`,

        `Добрый день. Мне нужны всего две вещи: кто сейчас ведёт доставку и сколько это стоит в месяц.

Если у нас не выйдет дешевле — так и скажу.`,

        `Здравствуйте. Пришлю пример, откроете, когда будет время.

Заказ падает в {current}, порядок работы не меняется — речь только об онлайн-заказах.`,
      ],
    },
  },
};

/** How many variants exist for this choice, so the screen can say "1 / 8". */
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

  // ⚠️ **An empty name becomes a visible blank, not a polite word.** The first
  // version filled it with "restoraningiz" / "вашего заведения", which reads as
  // finished text and is grammatically wrong in half the messages: Russian
  // needs a different case in "По {name}", "для {name}" and "{name} открывается",
  // and one fallback word cannot be in three cases at once. "По вашего
  // заведения" is exactly the kind of sentence that gets a message deleted
  // before the second line — and it would be *sent*, because it looks complete.
  //
  // A bracketed blank cannot be mistaken for finished writing, and the screen
  // refuses to copy while one is present. The name is never actually unknown:
  // the sender is looking at it.
  const fallbackName = lang === "uz" ? "[restoran nomi]" : "[название]";
  const fallbackCurrent = lang === "uz" ? "[kassa]" : "[касса]";

  // ⚠️ **The apostrophe belongs to the name, not to the fallback.** Uzbek
  // attaches case endings to a foreign proper noun with an apostrophe —
  // "iiko'ga", "B5 Somsa'ning" — so the templates carry `{name}'ga`. But the
  // fallback is an ordinary Uzbek word, and "restoraningiz'da" is not something
  // anybody writes. So a filled slot keeps the apostrophe and an empty one
  // swallows it, which is why these two passes exist rather than one.
  const fill = (text: string, slot: string, value: string, fallback: string) =>
    text
      .replaceAll(slot + "'", value ? value + "'" : fallback)
      .replaceAll(slot, value || fallback);

  const filled = raw
    .split("\n")
    .filter((line) => !(line.includes("{note}") && note === ""))
    .join("\n")
    .replaceAll("{note}", note);

  return fill(fill(filled, "{name}", name, fallbackName), "{current}", current, fallbackCurrent)
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
