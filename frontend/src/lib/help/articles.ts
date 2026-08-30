// The help articles the panel searches.
//
// ⚠️ **Shipped with the panel, not fetched from the platform.** The obvious
// design is to hold these in the control plane so an answer can be corrected
// without releasing thirty containers. It is the wrong one twice over: an
// article describes what *this build* does, and a platform-hosted answer would
// eventually describe a version the restaurant is not running — and the moment
// somebody most needs the help is the moment their container cannot reach
// anything. Bundled, the search works with the network down and always
// describes the software in front of the person reading it.
//
// ⚠️ **An answer that is wrong is worse than no article.** Everything here is
// written from the behaviour in this repository, not from what the product
// ought to do: the PIN lockout is five minutes because that is the constant,
// the code page is chosen from the text because that is what the encoder does.
// When one of those changes, the article beside it changes in the same commit.

import type { Lang } from "@/lib/i18n/dictionaries";

export type HelpCategory =
  "till" | "orders" | "menu" | "stock" | "team" | "money" | "settings";

export type HelpArticle = {
  id: string;
  cat: HelpCategory;
  title: string;
  body: string;
  /** Words somebody would search that are not in the text.
   *
   *  ⚠️ This is where the vocabulary gap goes. An owner types "kassa
   *  ochilmayapti"; the article is titled "PIN qabul qilinmayapti". Neither
   *  contains the other's words, and without this the search finds nothing for
   *  the single most common question there is. */
  keys?: string[];
};

const UZ: HelpArticle[] = [
  // ---- Kassa ----
  {
    id: "pin-locked",
    cat: "till",
    title: "PIN qabul qilinmayapti, kassa ochilmayapti",
    body: "Bir xil PIN ketma-ket 5 marta xato terilsa, o'sha PIN 5 daqiqaga bloklanadi. Bu butun kassani emas, faqat o'sha to'rt raqamni to'xtatadi — boshqa xodim o'z PINi bilan kiraveradi. Ekranda qancha kutish kerakligi yoziladi. PIN esdan chiqqan bo'lsa, panelda Xodimlar bo'limidan yangisini qo'ying.",
    keys: ["pin", "blok", "kassa ochilmayapti", "kirolmayapti", "parol", "kod"],
  },
  {
    id: "till-offline",
    cat: "till",
    title: "Internet uzilsa kassa ishlaydimi",
    body: "Ha. Sotuv qurilmaning o'zida saqlanadi va aloqa tiklanganda avtomatik yuboriladi — kassir navbatda to'xtamaydi. Har chekning o'z identifikatori bor, shuning uchun qayta yuborilganda ikkilanmaydi. Fiskal chek esa aloqa tiklanganda ro'yxatdan o'tadi.",
    keys: ["internet", "oflayn", "aloqa yo'q", "uzilib qoldi"],
  },
  {
    id: "till-fast-tap",
    cat: "till",
    title: "Taom kartochkalari tez bosganda ishlamayapti",
    body: "Bu tuzatilgan. Yangilanish o'rnatilganiga ishonch hosil qiling: kassa ekranida pastda versiya raqami turadi. Agar hali eski bo'lsa, brauzerni yangilang (Ctrl+Shift+R) yoki monoblokni qayta ishga tushiring.",
    keys: ["tez bosish", "bosilmayapti", "ishlamayapti", "sekin", "menyu"],
  },
  {
    id: "till-screens",
    cat: "till",
    title: "Zal va oshxona ekranlarini qanday ochaman",
    body: "Zal — /zal, oshxona — /staff/kitchen manzilidan ochiladi. Har ikkalasi ham xodim login-paroli bilan kiradi, keyin PIN so'raydi. Loginlarni panelda Xodimlar bo'limida yaratasiz; ofitsiantga «Zal», oshpazga «Oshxona» ruxsatini bering.",
    keys: ["zal", "oshxona", "kds", "planshet", "ofitsiant ekrani"],
  },
  {
    id: "till-keyboard",
    cat: "till",
    title: "Klaviatura ekranni pastga surib yuboryapti",
    body: "Bu tuzatilgan: klaviatura endi ekran ustida suzadi va sahifani qimirlatmaydi. Faqat yozilayotgan maydon klaviatura ostida qolsa, u avtomatik yuqoriga suriladi. Eski xatti-harakat qolgan bo'lsa — versiya eski, yangilang.",
    keys: ["klaviatura", "ekran sakraydi", "tepaga chiqib ketyapti"],
  },

  // ---- Buyurtmalar ----
  {
    id: "order-sound",
    cat: "orders",
    title: "Yangi buyurtma kelganda ovoz chiqmayapti",
    body: "Ovoz brauzer tomonidan bloklanadi, agar sahifada hech qachon bosilmagan bo'lsa. Panelning chap pastida «Ovoz yoqilgan» tugmasi bor — bir marta bosing. Telefonda ekran o'chsa ovoz ham to'xtaydi: buyurtmalar ekranini ochiq va telefonni uyg'oq qoldiring.",
    keys: ["ovoz", "signal", "eshitilmayapti", "jiringlamayapti"],
  },
  {
    id: "order-statuses",
    cat: "orders",
    title: "Buyurtma holatlari nimani anglatadi",
    body: "Yangi → Tasdiqlandi → Tayyorlanmoqda → Yo'lda → Yetkazildi. «Yangi» — hali hech kim qabul qilmagan; oshxona ekraniga u tushmaydi, chunki qabul qilinmagan buyurtma bekor qilinishi mumkin. Bekor qilishda sabab majburiy — u mijozning kuzatuv sahifasida ko'rinadi.",
    keys: ["holat", "status", "yangi", "tasdiqlash", "bekor"],
  },
  {
    id: "order-courier",
    cat: "orders",
    title: "Kuryerni buyurtmaga qanday biriktiraman",
    body: "Buyurtmalar ro'yxatida har qatorda «Kuryer tanlang» ro'yxati bor. Kuryer o'z ilovasida (/kuryer) buyurtmani ko'radi, manzil va olinadigan pul bilan. Kuryer ro'yxatda chiqmasa — u «Ishda emas» holatida yoki hisobi o'chirilgan.",
    keys: ["kuryer", "yetkazish", "biriktirish", "kuryer chiqmayapti"],
  },
  {
    id: "delivery-zone",
    cat: "orders",
    title: "Yetkazish narxi noto'g'ri hisoblanyapti",
    body: "Narx ikki xil hisoblanadi: masofa bo'yicha (baza narx + km) yoki xaritada chizilgan zonalar bo'yicha. Sozlamalar → Yetkazish bo'limida qaysi biri yoqilganini tekshiring. Zonalar chizilgan bo'lsa, radius sozlamalari e'tiborga olinmaydi va zonadan tashqariga yetkazilmaydi.",
    keys: ["yetkazish narxi", "zona", "km", "masofa", "dostavka"],
  },
  {
    id: "branch-delivery-off",
    cat: "orders",
    title: "Buyurtmalar noto'g'ri filialga tushyapti",
    body: "Har filialning o'z yetkazish sozlamasi bor. Agar bir filialda yetkazish o'chirilgan bo'lsa, u yetkazish buyurtmalarida umuman qatnashmaydi va hammasi qolganiga ketadi. Sozlamalar → Yetkazish sahifasida har filial uchun: yoqilganmi, xaritada nuqtasi bormi, maksimal masofa nechchi km.",
    keys: ["filial", "boshqa filialga", "noto'g'ri filial"],
  },

  // ---- Menyu ----
  {
    id: "menu-photo",
    cat: "menu",
    title: "Taom rasmini qanday qo'shaman",
    body: "Menyu → taomni oching → rasm maydoniga bosing. Eng yaxshi natija: kvadratga yaqin, 1000×1000 dan katta bo'lmagan surat. Rasm yuklangach sayt, bot va kassada bir vaqtda paydo bo'ladi — alohida yuklash shart emas.",
    keys: ["rasm", "surat", "foto", "yuklash"],
  },
  {
    id: "menu-stop-list",
    cat: "menu",
    title: "Taom tugadi — uni qanday yashiraman",
    body: "Stop list bo'limida taomni belgilang: u saytda ham, botda ham, kassada ham darhol «tugagan» bo'lib turadi. Ombor moduli yoqilgan bo'lsa, masalliq tugaganda taom o'zi stop listga tushadi. Ertasi kuni qo'lda qaytarasiz.",
    keys: ["tugadi", "stop list", "yo'q", "yashirish", "sotuvdan olish"],
  },
  {
    id: "menu-price",
    cat: "menu",
    title: "Narxni o'zgartirsam qayerda o'zgaradi",
    body: "Menyu bitta — narx sayt, Telegram bot, kassa va kuryer ilovasida bir vaqtda o'zgaradi. Allaqachon ochilgan cheklarga tegmaydi: chek yopilgan paytdagi narxni saqlaydi, chunki mijoz o'sha narxga rozi bo'lgan.",
    keys: ["narx", "o'zgartirish", "qimmatlashtirish"],
  },

  // ---- Ombor ----
  {
    id: "stock-negative",
    cat: "stock",
    title: "Ombor qoldig'i manfiy chiqyapti",
    body: "Qoldiq — oxirgi inventarizatsiya + kirimlar − texkarta bo'yicha sarf − chiqimlar. Manfiy son odatda ikkita sababdan: kirim kiritilmagan, yoki texkartadagi miqdor haqiqiydan katta. Avval o'sha masalliqni sanang (Inventarizatsiya), keyin texkartani tekshiring.",
    keys: ["manfiy", "minus", "qoldiq", "noto'g'ri"],
  },
  {
    id: "stock-techcard-where",
    cat: "stock",
    title: "Texkarta qayerda yoziladi",
    body: "Ombor → Texkartalar. Ikki xil karta bor. Zagotovka — oshxona o'zi tayyorlaydigan narsa (sous, xamir, sushi guruchi): masalliqlari va bir partiyadan chiqadigan miqdori yoziladi, narx yozilmaydi. Keyin u taom kartasida oddiy masalliq kabi grammlab ishlatiladi — masalan «Sushi zagotovka» 180 g. Taom kartasi ham shu ekranda, «Taomlar» bo'limida. Taomning o'zida faqat kartaning natijasi ko'rsatiladi.",
    keys: [
      "texkarta",
      "zagotovka",
      "yarim tayyor",
      "sous",
      "kalkulyatsiya",
      "texkarta qayerda",
    ],
  },
  {
    id: "stock-cost",
    cat: "stock",
    title: "Tannarx qanday hisoblanadi",
    body: "Har taomning texkartasi bor: qaysi masalliqdan qancha ketadi. Tannarx = masalliqlar miqdori × oxirgi kirim narxi. Texkartasi yo'q taomning tannarxi ham yo'q — u hisobotlarda «hisoblanmagan» bo'lib turadi va foyda hisobiga qo'shilmaydi.",
    keys: ["tannarx", "sebestoimost", "texkarta", "foyda"],
  },
  {
    id: "stock-branch",
    cat: "stock",
    title: "Ombor ekrani filial tanlashni so'rayapti",
    body: "Ombor har doim bitta filialga tegishli: uchta muzlatgichga tarqalgan «kompaniyada 9 kg go'sht bor» degan raqamni na sanab, na buyurtma berib bo'ladi. Yuqoridagi filial tanlagichdan birini tanlang. Bitta filialli restoranda bu so'ralmaydi.",
    keys: ["filial tanlang", "ombor ochilmayapti", "xatolik"],
  },

  // ---- Jamoa ----
  {
    id: "staff-login",
    cat: "team",
    title: "Xodimga login qanday beraman",
    body: "Xodimlar bo'limida yangi xodim qo'shing: ism, login, parol va PIN. Login-parol bilan ilovaga kiradi, PIN bilan kassada o'zini tanitadi. Ruxsatlar rol orqali beriladi — ofitsiant kassani ochadi, lekin hisobotlarni ko'rmaydi.",
    keys: ["xodim", "login", "parol", "ruxsat", "rol"],
  },
  {
    id: "staff-attendance",
    cat: "team",
    title: "Davomat qanday yoziladi",
    body: "Xodim ilovada «Ishga kirish» tugmasini bosadi; joylashuv restoran atrofida bo'lishi shart, shuning uchun uydan bosib bo'lmaydi. Smena kunlik hisoblanadi va oylik hisob-kitob shundan chiqadi. Bosilmay qolgan smenani panelda qo'lda tuzatasiz.",
    keys: ["davomat", "smena", "ishga kirish", "oylik", "gps"],
  },

  // ---- Pul ----
  {
    id: "receipt-language",
    cat: "money",
    title: "Chek noto'g'ri tilda chiqyapti",
    body: "Chek tili har tur uchun alohida sozlanadi: mehmon cheki, kassa cheki va oshxona cheki. Sozlamalar → Chek bo'limida har biri uchun tilni tanlang — oshxonasi ruscha, mehmoni o'zbekcha restoran odatiy hol. Namuna chek ham o'sha tilda ko'rsatiladi.",
    keys: ["chek tili", "ruscha", "o'zbekcha", "til"],
  },
  {
    id: "receipt-garbage",
    cat: "money",
    title: "Chekda savol belgilari (????) chiqyapti",
    body: "Bu kod sahifasi muammosi: termal printerda Unicode yo'q. Yangi versiyada matnda kirillcha bo'lsa printer avtomatik kirill sahifasiga o'tadi, ya'ni bu o'zi hal bo'ladi. Eski versiyada Sozlamalar → Printerlar bo'limida o'sha printerga «Kirill» belgisini qo'ying.",
    keys: ["????", "savol belgisi", "krakozyabra", "printer", "kirill"],
  },
  {
    id: "cash-variance",
    cat: "money",
    title: "Smena yopilganda farq chiqdi",
    body: "Farq = sanalgan − kutilgan. Manfiy son kamomad, musbat ortiqcha. Eng ko'p uchraydigan sabab: qaytim, kassadan olingan pul (u «Chiqim» sifatida kiritilishi kerak) va qarzga berilgan chek. Har smenaning farqi Kassa hisobotida saqlanadi.",
    keys: ["kamomad", "farq", "smena", "z hisobot", "pul yetishmayapti"],
  },
  {
    id: "online-payment",
    cat: "money",
    title: "Onlayn to'lov ishlamayapti",
    body: "Payme, Click, Uzum va ATMOS uchun kalitlar Sozlamalar → To'lov bo'limida kiritiladi. Kalit saqlangach ekranda faqat «kalit bor» belgisi turadi — bu normal, kalitlar qaytarilmaydi. To'lov o'tmasa avval provayder kabinetida shartnoma faolligini tekshiring.",
    keys: ["payme", "click", "uzum", "to'lov", "karta"],
  },

  // ---- Sozlamalar ----
  {
    id: "domain",
    cat: "settings",
    title: "O'z domenimni qanday ulayman",
    body: "Domenning A yozuvini bizning IP'ga yo'naltiring, keyin panelda Sozlamalar → Domen bo'limiga uni kiriting. Biz DNS haqiqatan bizga ko'rsatayotganini tekshiramiz — bu egalik isboti. Sertifikat avtomatik olinadi, odatda bir necha daqiqada.",
    keys: ["domen", "sayt manzili", "dns", "ssl", "https"],
  },
  {
    id: "map-key",
    cat: "settings",
    title: "Xarita ochilmayapti",
    body: "Xarita provayderi (2GIS, Yandex yoki Google) va uning kaliti Sozlamalar → Xarita bo'limida. Har provayderning kaliti alohida maydonda — birining kalitini boshqasiga qo'ysangiz xarita bo'sh chiqadi. Kalit provayder kabinetida sizning domeningizga ruxsat berilgan bo'lishi kerak.",
    keys: ["xarita", "2gis", "yandex", "google", "kalit", "bo'sh"],
  },
  {
    id: "telegram-bot",
    cat: "settings",
    title: "Telegram botni qanday ulayman",
    body: "BotFather'dan bot yarating va tokenini Sozlamalar → Telegram bo'limiga kiriting. Bot ulangach mijozlar menyuni to'g'ridan-to'g'ri Telegramda ko'radi va buyurtma beradi — buyurtma o'sha panelga tushadi. Bot javob bermasa tokenni qayta kiriting.",
    keys: ["telegram", "bot", "token", "mini app"],
  },
  {
    id: "qr-menu",
    cat: "settings",
    title: "Stol uchun QR kodni qayerdan olaman",
    body: "QR bo'limida har stol uchun kod chiqariladi va chop etish uchun tayyor PDF beriladi. Mehmon kodni skanerlaydi, menyuni ko'radi va o'z stoliga buyurtma beradi — buyurtma o'sha stol raqami bilan oshxonaga tushadi.",
    keys: ["qr", "stol", "kod", "chop etish"],
  },
];

/** Russian and English follow the same ids.
 *
 *  ⚠️ **The id is the contract, not the position.** An article that exists in
 *  one language and not another simply does not appear for that reader — which
 *  is right — but a list matched by index would show them the wrong answer
 *  under the right title, which is worse than showing nothing. */
const RU: HelpArticle[] = [
  {
    id: "pin-locked",
    cat: "till",
    title: "PIN не принимается, касса не открывается",
    body: "Если один и тот же PIN 5 раз подряд введён неверно, он блокируется на 5 минут. Блокируется именно эти четыре цифры, а не касса — другой сотрудник войдёт своим PIN. На экране написано, сколько осталось ждать. Если PIN забыт, задайте новый в разделе Сотрудники.",
    keys: ["пин", "блок", "касса не открывается", "пароль", "код"],
  },
  {
    id: "till-offline",
    cat: "till",
    title: "Работает ли касса без интернета",
    body: "Да. Продажа сохраняется на самом устройстве и уходит на сервер, когда связь вернётся — очередь не останавливается. У каждого чека свой идентификатор, поэтому повторная отправка не задваивает продажу. Фискальный чек регистрируется после восстановления связи.",
    keys: ["интернет", "офлайн", "нет связи"],
  },
  {
    id: "till-screens",
    cat: "till",
    title: "Как открыть экраны зала и кухни",
    body: "Зал — /zal, кухня — /staff/kitchen. Оба входят по логину сотрудника, затем спрашивают PIN. Логины создаются в разделе Сотрудники: официанту дайте доступ «Зал», повару — «Кухня».",
    keys: ["зал", "кухня", "kds", "планшет"],
  },
  {
    id: "order-sound",
    cat: "orders",
    title: "Нет звука при новом заказе",
    body: "Браузер блокирует звук, пока на странице ничего не нажимали. Внизу слева есть кнопка «Звук включён» — нажмите её один раз. На телефоне звук пропадает вместе с погасшим экраном: держите вкладку заказов открытой.",
    keys: ["звук", "сигнал", "не слышно"],
  },
  {
    id: "order-statuses",
    cat: "orders",
    title: "Что означают статусы заказа",
    body: "Новый → Подтверждён → Готовится → В пути → Доставлен. «Новый» никто ещё не принял, и на кухонный экран он не попадает: непринятый заказ ещё могут отменить. При отмене причина обязательна — её видит клиент на странице отслеживания.",
    keys: ["статус", "новый", "отмена", "подтвердить"],
  },
  {
    id: "delivery-zone",
    cat: "orders",
    title: "Стоимость доставки считается неверно",
    body: "Есть два способа: по расстоянию (база + км) и по зонам, нарисованным на карте. Проверьте в Настройки → Доставка, какой включён. Если зоны нарисованы, настройки радиуса не действуют и за пределы зон доставка не считается.",
    keys: ["доставка", "зона", "км", "расстояние"],
  },
  {
    id: "stock-negative",
    cat: "stock",
    title: "Остаток на складе отрицательный",
    body: "Остаток = последняя инвентаризация + приходы − расход по техкартам − списания. Минус обычно означает одно из двух: приход не занесён, либо в техкарте количество больше реального. Сначала пересчитайте позицию, потом проверьте техкарту.",
    keys: ["минус", "отрицательный", "остаток"],
  },
  {
    id: "stock-techcard-where",
    cat: "stock",
    title: "Где составляется техкарта",
    body: "Склад → Техкарты. Карт две. Заготовка — то, что кухня готовит сама (соус, тесто, рис для суши): указываются ингредиенты и выход одной партии, цена не вводится. Дальше она используется в карте блюда как обычный ингредиент, по граммам — например «Заготовка для суши» 180 г. Карта блюда там же, во вкладке «Блюда». В самом блюде показывается только результат карты.",
    keys: ["техкарта", "заготовка", "полуфабрикат", "соус", "калькуляция"],
  },
  {
    id: "stock-cost",
    cat: "stock",
    title: "Как считается себестоимость",
    body: "У каждого блюда есть техкарта: сколько какого ингредиента уходит. Себестоимость = количество × последняя цена прихода. Блюдо без техкарты себестоимости не имеет — в отчётах оно помечено и в расчёт прибыли не входит.",
    keys: ["себестоимость", "техкарта", "прибыль"],
  },
  {
    id: "receipt-language",
    cat: "money",
    title: "Чек печатается не на том языке",
    body: "Язык настраивается отдельно для каждого вида чека: гостевой, кассовый и кухонный. Настройки → Чек — выберите язык для каждого. Ресторан с русской кухней и узбекскими гостями — обычное дело. Образец чека показывается на том же языке.",
    keys: ["язык чека", "русский", "узбекский"],
  },
  {
    id: "receipt-garbage",
    cat: "money",
    title: "На чеке печатаются знаки вопроса (????)",
    body: "Это кодовая страница: в термопринтере нет Unicode. В новой версии принтер сам переключается на кириллицу, если в тексте есть кириллица. На старой — поставьте принтеру «Кириллица» в Настройки → Принтеры.",
    keys: ["????", "кракозябры", "принтер", "кириллица"],
  },
  {
    id: "cash-variance",
    cat: "money",
    title: "При закрытии смены расхождение",
    body: "Расхождение = посчитано − ожидалось. Минус — недостача, плюс — излишек. Чаще всего причина в сдаче, во взятых из кассы деньгах (их нужно проводить как «Расход») и в чеках, отданных в долг. Расхождение каждой смены хранится в кассовом отчёте.",
    keys: ["недостача", "расхождение", "смена", "z отчёт"],
  },
  {
    id: "map-key",
    cat: "settings",
    title: "Карта не открывается",
    body: "Провайдер карты (2GIS, Яндекс или Google) и его ключ — в Настройки → Карта. У каждого провайдера своё поле для ключа: чужой ключ даёт пустую карту. В кабинете провайдера ключ должен быть разрешён для вашего домена.",
    keys: ["карта", "2gis", "яндекс", "ключ", "пустая"],
  },
  {
    id: "telegram-bot",
    cat: "settings",
    title: "Как подключить Telegram-бота",
    body: "Создайте бота у BotFather и вставьте токен в Настройки → Telegram. После подключения клиенты видят меню прямо в Telegram и заказывают оттуда — заказ приходит в ту же панель.",
    keys: ["телеграм", "бот", "токен"],
  },
];

const EN: HelpArticle[] = [
  {
    id: "pin-locked",
    cat: "till",
    title: "The PIN is refused and the till will not open",
    body: "Five wrong tries of the same PIN block that PIN for five minutes. It blocks those four digits, not the till — another member of staff signs in with their own. The screen says how long is left. If the PIN is forgotten, set a new one under Staff.",
    keys: ["pin", "locked", "password", "code"],
  },
  {
    id: "till-offline",
    cat: "till",
    title: "Does the till work without internet",
    body: "Yes. The sale is kept on the device and sent when the connection returns, so the queue does not stop. Every sale carries its own id, so a retry cannot charge twice. The fiscal receipt is filed once the line is back.",
    keys: ["offline", "internet", "no connection"],
  },
  {
    id: "receipt-garbage",
    cat: "money",
    title: "The receipt prints question marks (????)",
    body: "That is the code page: a thermal printer has no Unicode. The current version switches the printer to Cyrillic by itself when the text contains Cyrillic. On an older one, set that printer to «Cyrillic» under Settings → Printers.",
    keys: ["????", "garbled", "printer", "cyrillic"],
  },
  {
    id: "cash-variance",
    cat: "money",
    title: "The shift closed with a difference",
    body: "Difference = counted − expected. Negative is short, positive is over. The usual causes are change, money taken out of the drawer without being entered as an expense, and checks left as debt. Every shift's difference is kept in the cash report.",
    keys: ["short", "variance", "shift", "z report"],
  },
  {
    id: "stock-techcard-where",
    cat: "stock",
    title: "Where a tech card is written",
    body: 'Store → Tech cards. There are two kinds. A prep is something the kitchen makes itself (a sauce, a dough, sushi rice): you enter its ingredients and what one batch yields, and no price. It is then used in a dish\'s card by the gram like any other ingredient — 180 g of "sushi prep", say. Dish cards are on the same screen, under Dishes. The dish form itself only shows what the card works out to.',
    keys: ["tech card", "prep", "semi-finished", "sauce", "recipe"],
  },
  {
    id: "stock-cost",
    cat: "stock",
    title: "How food cost is calculated",
    body: "Every dish has a card saying how much of each ingredient it uses. Cost = quantity × the latest purchase price. A dish with no card has no cost — reports mark it and leave it out of the margin.",
    keys: ["food cost", "recipe", "margin"],
  },
  {
    id: "map-key",
    cat: "settings",
    title: "The map does not load",
    body: "The map provider (2GIS, Yandex or Google) and its key live under Settings → Map. Each provider has its own key field: one provider's key in another's box draws an empty map. The key must also allow your domain in the provider's own console.",
    keys: ["map", "2gis", "yandex", "key", "blank"],
  },
];

export const HELP: Record<Lang, HelpArticle[]> = { uz: UZ, ru: RU, en: EN };
