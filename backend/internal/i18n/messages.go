package i18n

// messages is the catalogue. ⚠️ Sorted by the Uzbek text so a diff shows
// what changed rather than where it moved, and so a duplicate key is a
// compile error rather than a silently ignored second entry.
var messages = map[string]pair{
	"#%s buyurtma bekor qilindi": {
		"Заказ #%s отменён",
		"Order #%s was cancelled",
	},
	"#%s buyurtma bekor qilindi: %s": {
		"Заказ #%s отменён: %s",
		"Order #%s was cancelled: %s",
	},
	"#%s buyurtma sizdan olindi": {
		"Заказ #%s забрали у вас",
		"Order #%s was taken off you",
	},
	"#%s buyurtma sizga berildi. Manzil: %s": {
		"Заказ #%s передан вам. Адрес: %s",
		"Order #%s is yours. Address: %s",
	},
	"#%s buyurtma tayyor — olib chiqing": {
		"Заказ #%s готов — забирайте",
		"Order #%s is ready — come and collect it",
	},
	"#%s buyurtmaning manzili o'zgardi: %s": {
		"Адрес заказа #%s изменился: %s",
		"The address of order #%s has changed: %s",
	},
	"#%s · %d so'm": {
		"#%s · %d сум",
		"#%s · %d so'm",
	},
	"%d so'm (%s — %s)": {
		"%d сум (%s — %s)",
		"%d so'm (%s — %s)",
	},
	"%d so'm naqd pul qabul qilindi": {
		"Принято %d сум наличными",
		"%d so'm in cash was accepted",
	},
	"%s API orqali ishlamaydi": {
		"%s не работает через API",
		"%s does not work through the API",
	},
	"%s bilan ulanish hali tayyor emas — provayder API hujjati kutilmoqda": {
		"подключение к %s ещё не готово — ждём документацию API провайдера",
		"the connection to %s is not ready yet — the provider's API documentation is still awaited",
	},
	"%s bo'lib sotilmaydi": {
		"%s не продаётся частями",
		"%s is not sold in parts",
	},
	"%s bu filialda tugagan": {
		"%s закончился в этом филиале",
		"%s has run out at this branch",
	},
	"%s bu restoran uchun menyu qaytarmadi — havolada to'g'ri restoran ochilganini tekshiring": {
		"%s не вернул меню для этого ресторана — проверьте, что по ссылке открыт нужный ресторан",
		"%s returned no menu for this restaurant — check that the link opens the right one",
	},
	"%s bugun tugadi": {
		"%s сегодня закончился",
		"%s has run out today",
	},
	"%s da majburiy tanlov bor (%s) — bunday taomni belgilangan to'plamga qo'shib bo'lmaydi": {
		"у %s есть обязательный выбор (%s) — такое блюдо нельзя добавить в фиксированный набор",
		"%s has a required choice (%s) — a dish like that cannot go into a fixed combo",
	},
	"%s hozircha mavjud emas": {
		"%s сейчас недоступен",
		"%s is not available right now",
	},
	"%s kungi smenangiz o'zgartirildi": {
		"Ваша смена за %s изменена",
		"Your shift on %s was changed",
	},
	"%s menyuda topilmadi — savatni yangilang": {
		"%s не найден в меню — обновите корзину",
		"%s is not on the menu — refresh the basket",
	},
	"%s restoran ichidagi tarmoqda ishlaydi — ulanishni kassa ekranidan (/kassa) tekshiring.": {
		"%s работает во внутренней сети ресторана — проверьте подключение с экрана кассы (/kassa).",
		"%s runs on the restaurant's own network — check the connection from the till screen (/kassa).",
	},
	"%s to'plamining tarkibi aniqlanmadi — menyuda tekshiring": {
		"состав набора %s не определён — проверьте в меню",
		"the contents of the %s combo could not be worked out — check it on the menu",
	},
	"%s uchun API token kiritilmagan": {
		"для %s не введён API-токен",
		"no API token was entered for %s",
	},
	"%s — buyurtma tayyor": {
		"%s — заказ готов",
		"%s — the order is ready",
	},
	"%s — lekin bot javob bera olmaydi: %s": {
		"%s — но бот не может отвечать: %s",
		"%s — but the bot cannot answer: %s",
	},
	"%s-stolda ochiq chek bor (%s)": {
		"за столом %s есть открытый счёт (%s)",
		"table %s has an open check (%s)",
	},
	"%s-stolda ochiq chek bor (%s) — o'shanga qo'shing yoki yoping": {
		"за столом %s есть открытый счёт (%s) — добавьте туда или закройте его",
		"table %s has an open check (%s) — add to it or close it",
	},
	"%s: \"%s\" tanlanmagan": {
		"%s: «%s» не выбрано",
		"%s: \"%s\" has not been chosen",
	},
	"%s: \"%s\" uchun \"%s\" tanlovi yo'q — savatni yangilang": {
		"%s: у «%s» нет варианта «%s» — обновите корзину",
		"%s: \"%s\" has no option \"%s\" — refresh the basket",
	},
	"%s: \"%s\" uchun faqat bitta variant tanlanadi": {
		"%s: для «%s» выбирается только один вариант",
		"%s: only one option can be chosen for \"%s\"",
	},
	"%s: \"%s\" varianti menyuda yo'q — savatni yangilang": {
		"%s: варианта «%s» нет в меню — обновите корзину",
		"%s: the option \"%s\" is not on the menu — refresh the basket",
	},
	"%s: %s": {
		"%s: %s",
		"%s: %s",
	},
	"%s: %s — terminal guruhi javob bermayapti (kassa o'chiqmi?)": {
		"%s: %s — группа терминалов не отвечает (касса выключена?)",
		"%s: %s — the terminal group is not answering (is the till off?)",
	},
	"%s: apiLogin qabul qilinmadi (%s)": {
		"%s: apiLogin не принят (%s)",
		"%s: the apiLogin was refused (%s)",
	},
	"%s: bu apiLogin uchun tashkilot topilmadi — organizationId ni tekshiring": {
		"%s: для этого apiLogin организация не найдена — проверьте organizationId",
		"%s: no organisation was found for this apiLogin — check the organizationId",
	},
	"%s: bu ekran uchun ruxsat yo'q": {
		"%s: нет доступа к этому экрану",
		"%s: no permission for this screen",
	},
	"%s: buyurtma yuborildi, lekin kassa hali tasdiqlamadi — holatini tekshiring": {
		"%s: заказ отправлен, но касса ещё не подтвердила — проверьте статус",
		"%s: the order was sent but the till has not confirmed it — check its status",
	},
	"%s: ish smenangiz ochilmagan — smenani o'z telefoningizdan yoki kiosk QR orqali boshlang": {
		"%s: ваша рабочая смена не открыта — начните смену со своего телефона или по QR-коду киоска",
		"%s: your work shift is not open — start it from your own phone or with the kiosk QR code",
	},
	"%s: ulanib bo'lmadi: %s": {
		"%s: не удалось подключиться: %s",
		"%s: could not connect: %s",
	},
	"API token sozlanmagan": {
		"API-токен не настроен",
		"the API token is not set up",
	},
	"Avval bot tokenini kiriting va saqlang.": {
		"Сначала введите и сохраните токен бота.",
		"Enter and save the bot token first.",
	},
	"Botni shu guruhga qo'shdingizmi va admin qildingizmi?": {
		"Вы добавили бота в эту группу и сделали его админом?",
		"Have you added the bot to this group and made it an admin?",
	},
	"Bu QR kod allaqachon ishlatilgan — ekrandagi yangi kodni skaner qiling": {
		"Этот QR-код уже использован — отсканируйте новый код с экрана",
		"This QR code has already been used — scan the new one on the screen",
	},
	"Buyurtma bekor qilindi": {
		"Заказ отменён",
		"The order was cancelled",
	},
	"Buyurtma olindi": {
		"Заказ забрали",
		"The order was taken back",
	},
	"Buyurtma tayyor": {
		"Заказ готов",
		"The order is ready",
	},
	"CLICK Pass: %s": {
		"CLICK Pass: %s",
		"CLICK Pass: %s",
	},
	"CLICK Pass: javob o'qilmadi": {
		"CLICK Pass: ответ не прочитан",
		"CLICK Pass: the answer could not be read",
	},
	"CLICK Pass: servis id, foydalanuvchi id va maxfiy kalit kerak": {
		"CLICK Pass: нужны service id, user id и секретный ключ",
		"CLICK Pass: the service id, user id and secret key are all needed",
	},
	"Chegirma berish": {
		"Скидка",
		"Giving a discount",
	},
	"Grafik o'zgardi": {
		"График изменился",
		"The roster changed",
	},
	"Guruh yoki kanal ID si kiritilmagan.": {
		"ID группы или канала не указан.",
		"no group or channel ID was given.",
	},
	"Hisob o'chirildi": {
		"Аккаунт отключён",
		"The account was switched off",
	},
	"Hisobingiz vaqtincha o'chirildi — ma'muriyat bilan bog'laning": {
		"Ваш аккаунт временно отключён — свяжитесь с администрацией",
		"Your account has been switched off for now — contact the office",
	},
	"Hisobingiz vaqtincha o'chirildi — restoran bilan bog'laning": {
		"Ваш аккаунт временно отключён — свяжитесь с рестораном",
		"Your account has been switched off for now — contact the restaurant",
	},
	"Ish grafikingiz yangilandi — ilovadan ko'rib qo'ying": {
		"Ваш рабочий график обновлён — посмотрите в приложении",
		"Your roster has been updated — take a look in the app",
	},
	"Ish haqi yozildi": {
		"Зарплата записана",
		"Pay was recorded",
	},
	"Kassa smenasi": {
		"Кассовая смена",
		"The cash shift",
	},
	"Kunlik yakun": {
		"Итоги дня",
		"The day's totals",
	},
	"Manzil o'zgardi": {
		"Адрес изменился",
		"The address has changed",
	},
	"Mehmon fikri": {
		"Отзыв гостя",
		"A guest's review",
	},
	"Naqd qabul qilindi": {
		"Наличные приняты",
		"Cash accepted",
	},
	"Ochiq smena yo'q — avval ishga kirishni bosing": {
		"Нет открытой смены — сначала отметьте приход",
		"No shift is open — clock in first",
	},
	"Omborni sanash": {
		"Пересчёт склада",
		"Counting the store",
	},
	"Oshxona ekrani": {
		"Кухонный экран",
		"The kitchen screen",
	},
	"PIN 4 ta raqamdan iborat bo'lishi kerak": {
		"PIN должен состоять из 4 цифр",
		"the PIN has to be 4 digits",
	},
	"PIN faqat raqamlardan iborat bo'lishi kerak": {
		"PIN должен состоять только из цифр",
		"the PIN can only contain digits",
	},
	"PIN kiritilmagan": {
		"PIN не введён",
		"no PIN was entered",
	},
	"PIN noto'g'ri": {
		"неверный PIN",
		"wrong PIN",
	},
	"POS tizimi to'liq sozlanmagan": {
		"касса настроена не полностью",
		"the POS is not fully set up",
	},
	"POS tizimi yoqilmagan": {
		"касса не включена",
		"the POS is switched off",
	},
	"Pishirilgan taomni olib tashlash": {
		"Удаление приготовленного блюда",
		"Removing a cooked dish",
	},
	"QQS stavkasi 0 dan 100 gacha bo'lishi kerak": {
		"ставка НДС должна быть от 0 до 100",
		"the VAT rate has to be between 0 and 100",
	},
	"QQS stavkasi ko'rsatilmagan (QQS to'lovchisi bo'lmasangiz 0 kiriting)": {
		"ставка НДС не указана (если вы не плательщик НДС, введите 0)",
		"the VAT rate is missing (enter 0 if you do not pay VAT)",
	},
	"QR kod o'qilmadi — qayta skanerlang": {
		"QR-код не прочитан — отсканируйте ещё раз",
		"the QR code was not read — scan it again",
	},
	"SMS shlyuzi sozlanmagan — sozlamalardan provayderni ulang va sinov SMS yuboring": {
		"SMS-шлюз не настроен — подключите провайдера в настройках и отправьте тестовое SMS",
		"the SMS gateway is not set up — connect a provider in the settings and send a test SMS",
	},
	"SMS xizmati hali sozlanmagan — restoran bilan bog'laning": {
		"SMS-сервис ещё не настроен — свяжитесь с рестораном",
		"the SMS service is not set up yet — contact the restaurant",
	},
	"Siz allaqachon ishga kirgansiz — avval chiqishni bosing": {
		"Вы уже отметили приход — сначала нажмите уход",
		"You have already clocked in — clock out first",
	},
	"Smena tuzatildi": {
		"Смена исправлена",
		"A shift was corrected",
	},
	"Tayyor": {
		"Готово",
		"Ready",
	},
	"Telegram bot o'chirilgan — yuqoridagi tugmani yoqing.": {
		"Telegram-бот выключен — включите переключатель выше.",
		"The Telegram bot is off — turn the switch above on.",
	},
	"Telegram bot ulanmagan — sozlamalardan tokenni kiriting va ulanishni tekshiring": {
		"Telegram-бот не подключён — введите токен в настройках и проверьте подключение",
		"The Telegram bot is not connected — enter the token in the settings and check the connection",
	},
	"Telegram orqali kirish hali sozlanmagan": {
		"Вход через Telegram ещё не настроен",
		"Signing in with Telegram is not set up yet",
	},
	"To'lovni qabul qilish": {
		"Приём оплаты",
		"Taking payment",
	},
	"Tushum %d so'm · %d ta chek · o'rtacha %d so'm": {
		"Выручка %d сум · чеков: %d · средний %d сум",
		"Takings %d so'm · %d checks · average %d so'm",
	},
	"Uzum FastPay: %s": {
		"Uzum FastPay: %s",
		"Uzum FastPay: %s",
	},
	"Uzum FastPay: javob o'qilmadi": {
		"Uzum FastPay: ответ не прочитан",
		"Uzum FastPay: the answer could not be read",
	},
	"Uzum FastPay: servis id, kassa id va maxfiy kalit kerak": {
		"Uzum FastPay: нужны service id, id кассы и секретный ключ",
		"Uzum FastPay: the service id, till id and secret key are all needed",
	},
	"Yangi buyurtma": {
		"Новый заказ",
		"A new order",
	},
	"Zal ekrani": {
		"Экран зала",
		"The dining-room screen",
	},
	"agent kaliti qabul qilinmadi — paneldan yangi kalit oling": {
		"ключ агента не принят — возьмите новый ключ в панели",
		"the agent key was refused — take a new one from the panel",
	},
	"arizangiz allaqachon qabul qilingan — tez orada aloqaga chiqamiz": {
		"ваша заявка уже принята — скоро свяжемся",
		"your application is already in — we will be in touch soon",
	},
	"avtomatik o'qish hozir ishlamayapti. Menyu sahifasining boshqa havolasini sinab ko'ring yoki taomlarni fayldan import qiling": {
		"автоматическое чтение сейчас не работает. Попробуйте другую ссылку на страницу меню или импортируйте блюда из файла",
		"automatic reading is not working right now. Try another link to the menu page, or import the dishes from a file",
	},
	"avtomatik o'qish hozircha yoqilmagan": {
		"автоматическое чтение пока не включено",
		"automatic reading is not switched on yet",
	},
	"avtorizatsiya kerak": {
		"требуется авторизация",
		"authorisation required",
	},
	"avval shu brendning filiallarini o'chiring": {
		"сначала удалите филиалы этого бренда",
		"delete this brand's branches first",
	},
	"baho qoldirish uchun o'z hisobingizga kiring": {
		"чтобы оставить оценку, войдите в свой аккаунт",
		"sign in to your own account to leave a rating",
	},
	"bahoni tanlang": {
		"выберите оценку",
		"choose a rating",
	},
	"banner rasmi kerak": {
		"нужна картинка баннера",
		"the banner needs an image",
	},
	"bekor qilingan buyurtma": {
		"заказ отменён",
		"the order was cancelled",
	},
	"bekor qilish sababini yozing": {
		"укажите причину отмены",
		"give a reason for the cancellation",
	},
	"bir masalliqni o'ziga ko'chirib bo'lmaydi": {
		"нельзя переместить ингредиент сам в себя",
		"an ingredient cannot be moved into itself",
	},
	"bir so'rovda 50 tagacha chek": {
		"не более 50 чеков за один запрос",
		"at most 50 receipts per request",
	},
	"birorta ham masalliq topilmadi — avval masalliqlar ro'yxatini import qiling": {
		"не найдено ни одного ингредиента — сначала импортируйте список ингредиентов",
		"no ingredient was found — import the ingredient list first",
	},
	"bitta taom to'plamda ikki marta ko'rsatilgan — sonini oshiring": {
		"одно блюдо указано в наборе дважды — увеличьте количество",
		"one dish is listed twice in the combo — raise its quantity instead",
	},
	"boshlang'ich qoldiq manfiy bo'la olmaydi": {
		"начальный остаток не может быть отрицательным",
		"the opening balance cannot be negative",
	},
	"brend nomini yozing": {
		"введите название бренда",
		"enter the brand's name",
	},
	"brend topilmadi": {
		"бренд не найден",
		"the brand was not found",
	},
	"brendni tanlang": {
		"выберите бренд",
		"choose a brand",
	},
	"bron topilmadi": {
		"бронь не найдена",
		"reservation not found",
	},
	"bron uchun telefon raqamini SMS bilan tasdiqlang": {
		"подтвердите номер телефона по SMS",
		"confirm the phone number by SMS",
	},
	"bron vaqtincha yopiq": {
		"бронирование временно закрыто",
		"reservations are closed for now",
	},
	"bu PIN shu filialda allaqachon ishlatilyapti — boshqasini tanlang": {
		"этот PIN уже используется в этом филиале — выберите другой",
		"this PIN is already in use at this branch — pick another",
	},
	"bu POS tizimi bu amalni qo'llab-quvvatlamaydi": {
		"эта касса не поддерживает такое действие",
		"this POS does not support that action",
	},
	"bu amal faqat egasi uchun": {
		"это действие доступно только владельцу",
		"only the owner can do that",
	},
	"bu amal uchun ruxsat kerak": {
		"для этого действия нужен доступ",
		"this action needs permission",
	},
	"bu bo'lim tarifingizga kirmaydi": {
		"этот раздел не входит в ваш тариф",
		"this section is not part of your plan",
	},
	"bu buyurtma API orqali chaqirilmagan": {
		"этот заказ не был вызван через API",
		"this order was not requested through the API",
	},
	"bu buyurtma allaqachon to'langan": {
		"этот заказ уже оплачен",
		"this order has already been paid",
	},
	"bu buyurtma o'zgargan — ro'yxat yangilandi": {
		"заказ изменился — список обновлён",
		"this order changed — the list has been refreshed",
	},
	"bu buyurtmaga baho allaqachon qoldirilgan": {
		"оценка по этому заказу уже оставлена",
		"this order has already been rated",
	},
	"bu buyurtmaga xizmat chaqirilmagan": {
		"по этому заказу служба не вызывалась",
		"no service was called for this order",
	},
	"bu chek allaqachon qaytarilgan": {
		"по этому чеку уже был возврат",
		"this receipt has already been refunded",
	},
	"bu chekda boshqa brend taomi bor — alohida chek oching": {
		"в этом счёте есть блюдо другого бренда — откройте отдельный счёт",
		"this check has a dish from another brand — open a separate check",
	},
	"bu ekran uzilgan": {
		"этот экран отключён",
		"this screen has been unpaired",
	},
	"bu filial boshqa brendga tegishli": {
		"этот филиал принадлежит другому бренду",
		"that branch belongs to another brand",
	},
	"bu filial oldindan buyurtma qabul qilmaydi": {
		"этот филиал не принимает предзаказы",
		"this branch does not take pre-orders",
	},
	"bu filial sizga biriktirilmagan": {
		"этот филиал за вами не закреплён",
		"this branch is not assigned to you",
	},
	"bu filialda bunday ombor yo'q": {
		"в этом филиале нет такого склада",
		"this branch has no such store",
	},
	"bu filialda ombor bo'yicha to'xtatish yoqilmagan": {
		"в этом филиале стоп-лист по складу не включён",
		"stopping by stock is not switched on at this branch",
	},
	"bu filialga POS tizimi ulanmagan": {
		"к этому филиалу касса не подключена",
		"no POS is connected to this branch",
	},
	"bu foydalanuvchida allaqachon panel hisobi bor": {
		"у этого пользователя уже есть учётная запись в панели",
		"this person already has a panel account",
	},
	"bu hisobda telefon raqami yo'q": {
		"в этой учётной записи нет номера телефона",
		"this account has no phone number",
	},
	"bu hisobga tiklash uchun telefon raqami biriktirilmagan — parolni serverda tiklash kerak (DEPLOY.md)": {
		"к этому аккаунту не привязан телефон для восстановления — пароль сбрасывается на сервере (DEPLOY.md)",
		"this account has no phone number for recovery — the password has to be reset on the server (DEPLOY.md)",
	},
	"bu ichki raqam boshqa adminga biriktirilgan": {
		"этот внутренний номер закреплён за другим админом",
		"that extension belongs to another admin",
	},
	"bu ikki filial har xil brendga tegishli — menyusi ham boshqa, bog'lashni ko'chirish ma'nosiz": {
		"эти два филиала относятся к разным брендам — меню тоже разное, копировать привязки бессмысленно",
		"these two branches belong to different brands — their menus differ too, so copying the mapping makes no sense",
	},
	"bu kassa kun yakunini qo'llab-quvvatlamaydi": {
		"эта касса не поддерживает закрытие дня",
		"this till does not support closing the day",
	},
	"bu kod allaqachon ishlatilgan": {
		"этот код уже использован",
		"that code has already been used",
	},
	"bu kod chekda allaqachon bor": {
		"этот код уже есть в чеке",
		"this code is already on the check",
	},
	"bu kuryer boshqa filialga biriktirilgan": {
		"этот курьер закреплён за другим филиалом",
		"that courier belongs to another branch",
	},
	"bu login band": {
		"этот логин занят",
		"that login is taken",
	},
	"bu manzil ichki tarmoqqa qaraydi — tashqi sayt havolasini kiriting": {
		"этот адрес указывает во внутреннюю сеть — введите ссылку на внешний сайт",
		"that address points into the internal network — enter a link to an external site",
	},
	"bu manzilga yetkazib berilmaydi": {
		"по этому адресу доставки нет",
		"there is no delivery to that address",
	},
	"bu masalliq texkartada ishlatilmoqda: %s": {
		"этот ингредиент используется в техкарте: %s",
		"this ingredient is used in a recipe: %s",
	},
	"bu mijozga qarz yozib bo'lmaydi — ruxsatni ega beradi": {
		"этому клиенту нельзя записать долг — разрешение даёт владелец",
		"this guest may not be given credit — the owner grants that",
	},
	"bu qurilmaning kaliti almashtirilgan — paneldan yangi havola oling": {
		"ключ этого устройства заменён — возьмите новую ссылку в панели",
		"this device's key has been replaced — take a new link from the panel",
	},
	"bu raqam boshqa admin hisobida band": {
		"этот номер занят другим администратором",
		"another administrator is using that number",
	},
	"bu raqam boshqa foydalanuvchida band": {
		"этот номер занят другим пользователем",
		"another customer is using that number",
	},
	"bu raqam boshqa hisobda band — saytga shu raqam bilan kiring": {
		"этот номер занят другим аккаунтом — войдите на сайт с этим номером",
		"that number belongs to another account — sign in to the site with it",
	},
	"bu raqam boshqa hisobga tegishli": {
		"этот номер принадлежит другой учётной записи",
		"that number belongs to another account",
	},
	"bu rasm emas": {
		"это не изображение",
		"that is not an image",
	},
	"bu rol ishchilarga berilgan — avval ularga boshqa rol bering": {
		"эта роль назначена сотрудникам — сначала дайте им другую роль",
		"this role is assigned to staff — give them another role first",
	},
	"bu sahifada menyu topilmadi. Tekshirildi: %s. Menyu brauzerda chizilsa, menyu ochiq turgan sahifaning havolasini bering yoki taomlarni fayldan import qiling": {
		"на этой странице меню не найдено. Проверено: %s. Если меню рисуется в браузере, дайте ссылку на страницу с открытым меню или импортируйте блюда из файла",
		"no menu was found on this page. Checked: %s. If the menu is drawn in the browser, give the link to the page with the menu open, or import the dishes from a file",
	},
	"bu sahifada taomlar topilmadi — menyu sahifasining havolasini bering": {
		"на этой странице блюд не найдено — дайте ссылку на страницу меню",
		"no dishes were found on this page — give the link to the menu page",
	},
	"bu sahifada tayyor ma'lumot yo'q, avtomatik o'qish esa yoqilmagan": {
		"на этой странице нет готовых данных, а автоматическое чтение не включено",
		"this page has no ready data, and automatic reading is off",
	},
	"bu sana juda uzoq": {
		"эта дата слишком далеко",
		"that date is too far ahead",
	},
	"bu sanoq allaqachon izohlangan": {
		"по этому пересчёту комментарий уже оставлен",
		"this count already has a note",
	},
	"bu sayt avtomatik so'rovlarni bloklaydi (robot tekshiruvi), shuning uchun sahifani server o'qiy olmaydi — boshqa havola yordam bermaydi. Menyuni fayldan import qiling": {
		"этот сайт блокирует автоматические запросы (проверка на робота), поэтому сервер не может прочитать страницу — другая ссылка не поможет. Импортируйте меню из файла",
		"this site blocks automated requests (a robot check), so the server cannot read the page — another link will not help. Import the menu from a file",
	},
	"bu segmentda xabar yuboradigan odam yo'q": {
		"в этом сегменте некому отправлять",
		"there is nobody in this segment to write to",
	},
	"bu server mustaqil o'rnatilgan — domen shu serverning o'zida sozlanadi": {
		"этот сервер установлен отдельно — домен настраивается на самом сервере",
		"this server was installed on its own — the domain is set up on the server itself",
	},
	"bu stol buncha mehmonga kichik": {
		"этот стол мал для такого числа гостей",
		"that table is too small for this many guests",
	},
	"bu stol mavjud emas": {
		"такого стола нет",
		"there is no such table",
	},
	"bu stol shu vaqtga band": {
		"этот стол занят на это время",
		"that table is taken at that time",
	},
	"bu taom allaqachon oshxonaga yuborilgan — oshxonaga o'zingiz ayting yoki qatorni olib tashlab qaytadan qo'shing": {
		"это блюдо уже отправлено на кухню — скажите кухне сами или удалите строку и добавьте заново",
		"this dish has already gone to the kitchen — tell the kitchen yourself, or remove the line and add it again",
	},
	"bu taom hali oshxonaga yuborilmagan": {
		"это блюдо ещё не отправлено на кухню",
		"this dish has not been sent to the kitchen yet",
	},
	"bu taom to'plam(lar)da ishlatilgan: %s — avval o'sha to'plamlardan olib tashlang": {
		"это блюдо используется в наборе(ах): %s — сначала уберите его оттуда",
		"this dish is used in combo(s): %s — take it out of them first",
	},
	"bu telefonda boshqa hisob ishlatilyapti — administratorga murojaat qiling": {
		"на этом телефоне используется другой аккаунт — обратитесь к администратору",
		"another account is in use on this phone — ask the office",
	},
	"bu to'lov tizimi hali sozlanmagan": {
		"эта платёжная система ещё не настроена",
		"this payment system is not set up yet",
	},
	"bu to'lov tizimi uchun adapter hali yozilmagan": {
		"для этой платёжной системы адаптер ещё не написан",
		"no adapter has been written for this payment system yet",
	},
	"bu vakansiya endi yopilgan": {
		"эта вакансия уже закрыта",
		"this vacancy is closed",
	},
	"bu vaqt allaqachon o'tib ketgan": {
		"это время уже прошло",
		"that time has already passed",
	},
	"bu vaqtda restoran yopiq — boshqa vaqtni tanlang": {
		"в это время ресторан закрыт — выберите другое время",
		"the restaurant is closed at that time — pick another",
	},
	"bu vaqtga bo'sh stol qolmadi": {
		"на это время свободных столов нет",
		"no free tables at that time",
	},
	"bugun ikkitadan ortiq ariza qabul qilinmaydi — ertaga urinib ko'ring": {
		"больше двух заявок в день не принимается — попробуйте завтра",
		"no more than two applications a day are taken — try tomorrow",
	},
	"bugungi AI limiti tugadi — ertaga qayta urinib ko'ring": {
		"дневной лимит AI исчерпан — попробуйте завтра",
		"today's AI limit is used up — try again tomorrow",
	},
	"bunday login topilmadi": {
		"такой логин не найден",
		"no such login",
	},
	"bunday sayt topilmadi": {
		"такой сайт не найден",
		"no such site was found",
	},
	"bunday stol xaritada yo'q — ekranni yangilang": {
		"такого стола нет на схеме — обновите экран",
		"that table is not on the plan — refresh the screen",
	},
	"buyurtma hali yetkazilmagan": {
		"заказ ещё не доставлен",
		"the order has not been delivered yet",
	},
	"buyurtma topilmadi": {
		"заказ не найден",
		"order not found",
	},
	"buyurtmada yetkazish manzili yo'q": {
		"в заказе нет адреса доставки",
		"the order has no delivery address",
	},
	"chegirma sababini yozing": {
		"укажите причину скидки",
		"give a reason for the discount",
	},
	"chek bekor qilingan — tahrirlab bo'lmaydi": {
		"чек отменён — изменить нельзя",
		"the receipt is void and cannot be edited",
	},
	"chek bo'sh": {
		"чек пуст",
		"the receipt is empty",
	},
	"chek bo'sh — to'lash o'rniga bekor qiling": {
		"счёт пуст — вместо оплаты отмените его",
		"the check is empty — cancel it instead of paying",
	},
	"chek hali yopilmagan": {
		"чек ещё не закрыт",
		"the receipt is still open",
	},
	"chek topilmadi": {
		"чек не найден",
		"receipt not found",
	},
	"chek yopilgan": {
		"чек закрыт",
		"the receipt is closed",
	},
	"chek yopilgan — tahrirlab bo'lmaydi": {
		"чек закрыт — изменить нельзя",
		"the receipt is closed and cannot be edited",
	},
	"chekni o'ziga qo'shib bo'lmaydi": {
		"чек нельзя объединить с самим собой",
		"a receipt cannot be merged into itself",
	},
	"chop etish boshlanmadi (%s): %s": {
		"печать не началась (%s): %s",
		"printing did not start (%s): %s",
	},
	"chop etish ma'lumoti buzuq: %s": {
		"данные для печати повреждены: %s",
		"the print data is broken: %s",
	},
	"chop etish topshirig'i topilmadi": {
		"задание печати не найдено",
		"print job not found",
	},
	"clientId yo'q": {
		"нет clientId",
		"there is no clientId",
	},
	"clopos %s: %s": {
		"clopos %s: %s",
		"clopos %s: %s",
	},
	"clopos: %s uchun mahsulot id raqam bo'lishi kerak (%s)": {
		"clopos: id товара для %s должен быть числом (%s)",
		"clopos: the product id for %s has to be a number (%s)",
	},
	"clopos: buyurtma yaratilmadi (%s)": {
		"clopos: заказ не создан (%s)",
		"clopos: the order was not created (%s)",
	},
	"clopos: kirish rad etildi (%s)": {
		"clopos: вход отклонён (%s)",
		"clopos: access was refused (%s)",
	},
	"clopos: ulanib bo'lmadi: %s": {
		"clopos: не удалось подключиться: %s",
		"clopos: could not connect: %s",
	},
	"clopos: venue_id %d topilmadi. Mavjudlari: %s": {
		"clopos: venue_id %d не найден. Доступные: %s",
		"clopos: venue_id %d was not found. Available: %s",
	},
	"davrni tanlang": {
		"выберите период",
		"choose a period",
	},
	"domen hali bu serverga yo'naltirilmagan — DNS yozuvini tekshiring": {
		"домен ещё не направлен на этот сервер — проверьте DNS-запись",
		"the domain does not point here yet — check the DNS record",
	},
	"domen kerak": {
		"нужен домен",
		"a domain is required",
	},
	"domen noto'g'ri": {
		"домен указан неверно",
		"that domain is not valid",
	},
	"domen yoki API kalit kiritilmagan": {
		"домен или API-ключ не введён",
		"the domain or the API key is missing",
	},
	"ekran rejimi noma'lum": {
		"режим экрана неизвестен",
		"unknown screen mode",
	},
	"ekran tokeni yaroqsiz": {
		"токен экрана недействителен",
		"the screen's token is not valid",
	},
	"ekran topilmadi": {
		"экран не найден",
		"screen not found",
	},
	"ekrandagi kodni kiriting": {
		"введите код с экрана",
		"type the code shown on the screen",
	},
	"eng ko'pi bilan %d kun oldin buyurtma berish mumkin": {
		"заказ можно оформить максимум за %d дней",
		"an order can be placed at most %d days ahead",
	},
	"faqat partiya bilan tayyorlanadigan yarim tayyor mahsulot tanlanadi": {
		"выбирается только полуфабрикат, который готовится партией",
		"only a semi-product that is made in batches can be chosen",
	},
	"farq bor — sababini yozing": {
		"есть расхождение — напишите причину",
		"there is a difference — write the reason",
	},
	"fayl bo'sh": {
		"файл пуст",
		"the file is empty",
	},
	"fayl juda katta": {
		"файл слишком большой",
		"the file is too large",
	},
	"fayl yuborilmadi": {
		"файл не отправлен",
		"no file was sent",
	},
	"faylda varaq yo'q": {
		"в файле нет листа",
		"the file has no sheet",
	},
	"faylni ochib bo'lmadi — Excel yoki CSV bo'lishi kerak": {
		"файл не открылся — нужен Excel или CSV",
		"the file could not be opened — it has to be Excel or CSV",
	},
	"fikringiz allaqachon yuborilgan — rahmat": {
		"ваш отзыв уже отправлен — спасибо",
		"your feedback has already been sent — thank you",
	},
	"filial nomini yozing": {
		"введите название филиала",
		"enter the branch name",
	},
	"filial noto'g'ri": {
		"филиал указан неверно",
		"that branch is not valid",
	},
	"filial tanlanmagan": {
		"филиал не выбран",
		"no branch selected",
	},
	"filial tanlanmagan yoki topilmadi": {
		"филиал не выбран или не найден",
		"no branch was chosen, or it was not found",
	},
	"filial topilmadi": {
		"филиал не найден",
		"branch not found",
	},
	"filiallar har xil kassada (%s va %s) — mahsulot id'lari mos kelmaydi, har birini alohida bog'lash kerak": {
		"филиалы на разных кассах (%s и %s) — id товаров не совпадают, каждый нужно привязывать отдельно",
		"the branches are on different tills (%s and %s) — the product ids do not match, so each has to be mapped separately",
	},
	"filialni tanlang": {
		"выберите филиал",
		"choose a branch",
	},
	"fiskal chek bo'sh": {
		"фискальный чек пуст",
		"the fiscal receipt is empty",
	},
	"fiskal kassa ulanmagan": {
		"фискальная касса не подключена",
		"no fiscal device is connected",
	},
	"fiskal provayder sozlanmagan": {
		"фискальный провайдер не настроен",
		"the fiscal provider is not configured",
	},
	"fiskal provayder tanlanmagan": {
		"фискальный провайдер не выбран",
		"no fiscal provider chosen",
	},
	"fiskallashtirilmagan cheklar bor — avval ularni yuboring": {
		"есть нефискализированные чеки — сначала отправьте их",
		"there are receipts that were never fiscalised — send them first",
	},
	"foydalanuvchi tanlanmagan": {
		"пользователь не выбран",
		"no customer selected",
	},
	"foydalanuvchi topilmadi": {
		"пользователь не найден",
		"customer not found",
	},
	"from: sana noto'g'ri": {
		"from: неверная дата",
		"from: invalid date",
	},
	"hammasini bo'lib bo'lmaydi — chekda kamida bitta taom qolishi kerak": {
		"разделить всё нельзя — в счёте должно остаться хотя бы одно блюдо",
		"everything cannot be split off — at least one dish has to stay on the check",
	},
	"havola juda ko'p marta yo'naltirdi": {
		"ссылка слишком много раз перенаправила",
		"the link redirected too many times",
	},
	"havola kiritilmagan": {
		"ссылка не введена",
		"no link was entered",
	},
	"havola tushunarsiz": {
		"ссылка непонятна",
		"the link cannot be read",
	},
	"hech bir qurilmaga yetkazilmadi": {
		"не доставлено ни на одно устройство",
		"it reached no device",
	},
	"hech bo'lmasa bitta masalliq kerak": {
		"нужен хотя бы один ингредиент",
		"at least one ingredient is required",
	},
	"hech bo'lmasa bitta qator kerak": {
		"нужна хотя бы одна строка",
		"at least one line is needed",
	},
	"hech nima tanlanmagan": {
		"ничего не выбрано",
		"nothing selected",
	},
	"hisob o'chirilgan": {
		"учётная запись отключена",
		"the account is disabled",
	},
	"hisob o'chirilgan — ma'muriyat bilan bog'laning": {
		"учётная запись отключена — обратитесь к администрации",
		"the account is disabled — contact your manager",
	},
	"hisob o'chirilgan — restoran bilan bog'laning": {
		"учётная запись отключена — обратитесь в ресторан",
		"the account is disabled — contact the restaurant",
	},
	"hisob topilmadi": {
		"учётная запись не найдена",
		"account not found",
	},
	"hisobingiz boshqa telefonga biriktirilgan — administratordan uni o'chirishni so'rang": {
		"ваш аккаунт привязан к другому телефону — попросите администратора удалить привязку",
		"your account is bound to another phone — ask the office to release it",
	},
	"hisobingiz filialga biriktirilmagan — administratorga murojaat qiling": {
		"ваша учётная запись не закреплена за филиалом — обратитесь к администратору",
		"your account is not attached to a branch — ask an administrator",
	},
	"holat noto'g'ri": {
		"неверный статус",
		"that status is not valid",
	},
	"hozir boshqa kampaniya yuborilmoqda — tugashini kuting": {
		"сейчас отправляется другая кампания — дождитесь её окончания",
		"another campaign is being sent — wait for it to finish",
	},
	"id noto'g'ri": {
		"неверный id",
		"invalid id",
	},
	"ikkalasi ham bitta omborda — ko'chirishga hojat yo'q": {
		"оба на одном складе — перемещать не нужно",
		"both are in the same store — there is nothing to move",
	},
	"import turi noma'lum": {
		"неизвестный тип импорта",
		"unknown import type",
	},
	"ishchi topilmadi": {
		"сотрудник не найден",
		"employee not found",
	},
	"ismingizni yozing": {
		"введите ваше имя",
		"enter your name",
	},
	"izoh topilmadi": {
		"отзыв не найден",
		"review not found",
	},
	"javobni o'qib bo'lmadi: %s": {
		"ответ не удалось прочитать: %s",
		"the answer could not be read: %s",
	},
	"joriy parol noto'g'ri": {
		"текущий пароль неверный",
		"the current password is wrong",
	},
	"juda ko'p qabul qiluvchi — segmentni toraytiring": {
		"слишком много получателей — сузьте сегмент",
		"too many recipients — narrow the segment",
	},
	"kalit yaratib bo'lmadi": {
		"не удалось создать ключ",
		"the key could not be created",
	},
	"kassa amalni rad etdi (kod %d)": {
		"касса отклонила операцию (код %d)",
		"the till refused the operation (code %d)",
	},
	"kassa bo'sh ro'yxat qaytardi — stop list o'zgartirilmadi": {
		"касса вернула пустой список — стоп-лист не изменён",
		"the till returned an empty list — the stop list was left alone",
	},
	"kassa dasturi javob bermadi": {
		"программа кассы не ответила",
		"the till software did not answer",
	},
	"kassa dasturiga ulanib bo'lmadi — manzilni va planshet kassa bilan bir tarmoqda ekanini tekshiring (%s)": {
		"не удалось подключиться к программе кассы — проверьте адрес и то, что планшет в одной сети с кассой (%s)",
		"could not connect to the till software — check the address, and that the tablet is on the same network as the till (%s)",
	},
	"kassa dasturining manzili kiritilmagan (masalan http://192.168.1.50:9090)": {
		"адрес программы кассы не введён (например http://192.168.1.50:9090)",
		"the till software's address is missing (for example http://192.168.1.50:9090)",
	},
	"kassa fiskal belgi qaytarmadi": {
		"касса не вернула фискальный признак",
		"the till returned no fiscal mark",
	},
	"kassa javob berdi": {
		"касса ответила",
		"the till answered",
	},
	"kassa javob bermadi (HTTP %d)": {
		"касса не ответила (HTTP %d)",
		"the till did not answer (HTTP %d)",
	},
	"kassadan tushunarsiz javob (HTTP %d): %s": {
		"непонятный ответ от кассы (HTTP %d): %s",
		"an unreadable answer from the till (HTTP %d): %s",
	},
	"kiosk sozlanmagan": {
		"киоск не настроен",
		"the kiosk is not set up",
	},
	"kiosk tokeni bekor qilingan — paneldan yangisini oling": {
		"токен киоска отозван — возьмите новый в панели",
		"the kiosk token was revoked — take a new one from the panel",
	},
	"kiosk tokeni yaroqsiz": {
		"токен киоска недействителен",
		"the kiosk token is not valid",
	},
	"kirish talab qilinadi": {
		"требуется вход",
		"sign-in required",
	},
	"kod muddati tugagan, qaytadan so'rang": {
		"срок кода истёк, запросите заново",
		"the code has expired, ask for a new one",
	},
	"kod noto'g'ri": {
		"код неверный",
		"the code is wrong",
	},
	"kod topilmadi yoki eskirgan — ekrandagi yangi kodni kiriting": {
		"код не найден или устарел — введите новый код с экрана",
		"that code is unknown or expired — type the new one shown on the screen",
	},
	"kod topilmadi, qaytadan so'rang": {
		"код не найден, запросите заново",
		"the code was not found, ask for a new one",
	},
	"kod yaqinda yuborilgan, %d soniyadan keyin qayta urining": {
		"код уже отправлен, повторите через %d секунд",
		"a code was just sent, try again in %d seconds",
	},
	"kod yuborilmadi. Birozdan keyin urinib ko'ring yoki restoran bilan bog'laning": {
		"код не отправлен. Попробуйте чуть позже или свяжитесь с рестораном",
		"the code was not sent. Try again shortly, or contact the restaurant",
	},
	"koordinata yo'q": {
		"нет координат",
		"no coordinates",
	},
	"kurs 1 dan %s gacha bo'lishi kerak": {
		"курс должен быть от 1 до %s",
		"the course has to be between 1 and %s",
	},
	"kuryer topilmadi": {
		"курьер не найден",
		"courier not found",
	},
	"kuryerda tugallanmagan buyurtma bor — avval uni boshqa kuryerga bering": {
		"у курьера есть незавершённый заказ — сначала передайте его другому курьеру",
		"the courier has an unfinished order — hand it to another courier first",
	},
	"lavozim nomini yozing": {
		"введите название должности",
		"enter the position",
	},
	"login kamida 3 belgi bo'lishi kerak": {
		"логин должен быть не короче 3 символов",
		"the login must be at least 3 characters",
	},
	"login yoki parol noto'g'ri": {
		"неверный логин или пароль",
		"wrong login or password",
	},
	"loginni yozing": {
		"введите логин",
		"enter a login",
	},
	"ma'lumotlarni yuklab olishga ruxsat berilmagan yoki muddati tugagan": {
		"выгрузка данных не разрешена или срок доступа истёк",
		"the download was never granted, or it has expired",
	},
	"mahsulot topilmadi": {
		"товар не найден",
		"product not found",
	},
	"manba filial tanlanmagan": {
		"не выбран филиал-источник",
		"no source branch selected",
	},
	"manba filial topilmadi": {
		"филиал-источник не найден",
		"source branch not found",
	},
	"manba va nishon bir xil filial": {
		"источник и получатель — один и тот же филиал",
		"source and destination are the same branch",
	},
	"markirovka kodi noto'g'ri": {
		"код маркировки неверный",
		"the marking code is wrong",
	},
	"markirovka kodi yo'q": {
		"нет кода маркировки",
		"there is no marking code",
	},
	"masalliq tanlanmagan": {
		"ингредиент не выбран",
		"no ingredient selected",
	},
	"masalliq topilmadi": {
		"ингредиент не найден",
		"ingredient not found",
	},
	"masalliq va miqdorni tanlang": {
		"выберите ингредиент и количество",
		"choose an ingredient and a quantity",
	},
	"matn ichida %s bo'lishi shart — kod o'sha yerga qo'yiladi": {
		"в тексте обязательно должно быть %s — туда подставляется код",
		"the text has to contain %s — the code goes there",
	},
	"mehmon raqami 1 dan %s gacha bo'lishi kerak": {
		"номер гостя должен быть от 1 до %s",
		"the guest number has to be between 1 and %s",
	},
	"mehmonlar soni juda ko'p": {
		"слишком много гостей",
		"too many guests",
	},
	"mijoz ismini yozing": {
		"введите имя клиента",
		"enter the customer's name",
	},
	"mijoz topilmadi": {
		"клиент не найден",
		"customer not found",
	},
	"miqdor noldan katta bo'lsin": {
		"количество должно быть больше нуля",
		"the quantity has to be above zero",
	},
	"natija noto'g'ri": {
		"неверный результат",
		"that outcome is not valid",
	},
	"natijani tanlang": {
		"выберите результат",
		"choose an outcome",
	},
	"nima qilinganini yozing": {
		"напишите, что было сделано",
		"write down what was done",
	},
	"noma'lum POS tizimi": {
		"неизвестная POS-система",
		"unknown till system",
	},
	"noma'lum POS tizimi: %s": {
		"неизвестная касса: %s",
		"unknown POS: %s",
	},
	"noma'lum SMS provayderi: %s": {
		"неизвестный SMS-провайдер: %s",
		"unknown SMS provider: %s",
	},
	"noma'lum agent kaliti": {
		"неизвестный ключ агента",
		"unknown agent key",
	},
	"noma'lum fiskal provayder": {
		"неизвестный фискальный провайдер",
		"unknown fiscal provider",
	},
	"noma'lum holat": {
		"неизвестный статус",
		"unknown status",
	},
	"noma'lum integratsiya: %s": {
		"неизвестная интеграция: %s",
		"unknown integration: %s",
	},
	"noma'lum rol": {
		"неизвестная роль",
		"unknown role",
	},
	"noma'lum to'lov tizimi": {
		"неизвестная платёжная система",
		"unknown payment provider",
	},
	"noma'lum to'lov turi": {
		"неизвестный способ оплаты",
		"unknown payment method",
	},
	"noma'lum turdagi hisob": {
		"неизвестный тип аккаунта",
		"unknown kind of account",
	},
	"nomini yozing": {
		"введите название",
		"enter a name",
	},
	"noto'g'ri id": {
		"неверный id",
		"invalid id",
	},
	"o'lchov birligi tanilmadi": {
		"единица измерения не распознана",
		"the unit of measure was not recognised",
	},
	"o'lchov birliklari har xil — kilogrammni litrga ko'chirib bo'lmaydi": {
		"единицы измерения разные — килограммы в литры не переместить",
		"the units differ — kilograms cannot be moved into litres",
	},
	"o'tgan vaqtga buyurtma berib bo'lmaydi": {
		"нельзя оформить заказ на прошедшее время",
		"an order cannot be placed for a time that has passed",
	},
	"o'z hisobingizni o'chira olmaysiz": {
		"свою учётную запись удалить нельзя",
		"you cannot delete your own account",
	},
	"obuna kalitlari yetishmayapti": {
		"не хватает ключей подписки",
		"the subscription keys are incomplete",
	},
	"ochiq smena yo'q": {
		"нет открытой смены",
		"no shift is open",
	},
	"ofitsiant topilmadi": {
		"официант не найден",
		"waiter not found",
	},
	"oldindan buyurtma kamida %d daqiqa oldin beriladi": {
		"предзаказ оформляется минимум за %d минут",
		"a pre-order has to be at least %d minutes ahead",
	},
	"olib ketish buyurtmasida manzil yo'q": {
		"у заказа на самовывоз нет адреса",
		"a pickup order has no address",
	},
	"ombor nomi kerak": {
		"нужно название склада",
		"the store needs a name",
	},
	"ombor noto'g'ri": {
		"склад указан неверно",
		"that store is not valid",
	},
	"ombor topilmadi": {
		"склад не найден",
		"store not found",
	},
	"omborni ko'rish uchun filialni tanlang": {
		"выберите филиал, чтобы посмотреть склад",
		"pick a branch to see the store",
	},
	"onlinePBX: %s": {
		"onlinePBX: %s",
		"onlinePBX: %s",
	},
	"onlinePBX: %s domeni qabul qilmadi (%s) — domen va API kalitni tekshiring": {
		"onlinePBX: домен %s не принял (%s) — проверьте домен и API-ключ",
		"onlinePBX: the domain %s refused it (%s) — check the domain and the API key",
	},
	"onlinePBX: %s domeni topilmadi": {
		"onlinePBX: домен %s не найден",
		"onlinePBX: the domain %s was not found",
	},
	"onlinePBX: API kalit qabul qilinmadi": {
		"onlinePBX: API-ключ не принят",
		"onlinePBX: the API key was refused",
	},
	"onlinePBX: domen yoki API kalit kiritilmagan": {
		"onlinePBX: домен или API-ключ не введён",
		"onlinePBX: the domain or the API key is missing",
	},
	"onlinePBX: kutilmagan javob (%s)": {
		"onlinePBX: неожиданный ответ (%s)",
		"onlinePBX: an unexpected answer (%s)",
	},
	"onlinePBX: operatorning ichki raqami ko'rsatilmagan": {
		"onlinePBX: внутренний номер оператора не указан",
		"onlinePBX: the operator's extension is missing",
	},
	"onlinePBX: qaysi raqamga qo'ng'iroq qilish kerak?": {
		"onlinePBX: на какой номер звонить?",
		"onlinePBX: which number should be called?",
	},
	"onlinePBX: sozlanmagan": {
		"onlinePBX: не настроен",
		"onlinePBX: not set up",
	},
	"onlinePBX: ulanib bo'lmadi: %s": {
		"onlinePBX: не удалось подключиться: %s",
		"onlinePBX: could not connect: %s",
	},
	"onlinePBX: yozuv havolasi qaytmadi": {
		"onlinePBX: ссылка на запись не вернулась",
		"onlinePBX: no recording link came back",
	},
	"oxirgi brendni o'chirib bo'lmaydi": {
		"последний бренд удалить нельзя",
		"the last brand cannot be deleted",
	},
	"oxirgi egani boshqa rolga o'tkazib bo'lmaydi": {
		"последнего владельца нельзя перевести в другую роль",
		"the last owner cannot be moved to another role",
	},
	"oxirgi egani o'chirib bo'lmaydi": {
		"последнего владельца удалить нельзя",
		"the last owner cannot be deleted",
	},
	"oxirgi filialni o'chirib bo'lmaydi": {
		"последний филиал удалить нельзя",
		"the last branch cannot be deleted",
	},
	"oxirgi sana boshlanishidan oldin": {
		"конечная дата раньше начальной",
		"the end date is before the start",
	},
	"parol kamida 5 belgi bo'lishi kerak": {
		"пароль должен быть не короче 5 символов",
		"the password must be at least 5 characters",
	},
	"parol kamida 6 belgi bo'lishi kerak": {
		"пароль должен быть не короче 6 символов",
		"the password must be at least 6 characters",
	},
	"partiya faqat ishlab chiqarish omborida tayyorlanadi (tsex)": {
		"партия готовится только на производственном складе (цех)",
		"a batch is only made in a production store (the workshop)",
	},
	"platformaga ulanib bo'lmadi: %s": {
		"не удалось подключиться к платформе: %s",
		"could not connect to the platform: %s",
	},
	"poster %s: %s": {
		"poster %s: %s",
		"poster %s: %s",
	},
	"poster %s: ulanib bo'lmadi: %s": {
		"poster %s: не удалось подключиться: %s",
		"poster %s: could not connect: %s",
	},
	"poster: %d raqamli savdo nuqtasi yo'q. Mavjudlari: %s": {
		"poster: торговой точки с номером %d нет. Доступные: %s",
		"poster: there is no spot numbered %d. Available: %s",
	},
	"poster: %s taomining kassadagi id'si raqam emas (%s)": {
		"poster: id блюда %s в кассе не число (%s)",
		"poster: the till id of the dish %s is not a number (%s)",
	},
	"poster: bu token uchun savdo nuqtasi topilmadi": {
		"poster: для этого токена торговая точка не найдена",
		"poster: no spot was found for this token",
	},
	"poster: buyurtma raqami qaytmadi — kassada tekshiring": {
		"poster: номер заказа не вернулся — проверьте на кассе",
		"poster: no order number came back — check on the till",
	},
	"printer manzili tushunarsiz: %s": {
		"адрес принтера непонятен: %s",
		"the printer address cannot be read: %s",
	},
	"printer manzili yozilmagan": {
		"адрес принтера не указан",
		"the printer address is missing",
	},
	"printer nomi noto'g'ri: %s": {
		"имя принтера неверное: %s",
		"the printer name is wrong: %s",
	},
	"printer nomi yozilmagan": {
		"имя принтера не указано",
		"the printer name is missing",
	},
	"printer ochilmadi (%s): %s": {
		"принтер не открылся (%s): %s",
		"the printer did not open (%s): %s",
	},
	"printer topilmadi": {
		"принтер не найден",
		"printer not found",
	},
	"printer topilmadi (%s): %s": {
		"принтер не найден (%s): %s",
		"the printer was not found (%s): %s",
	},
	"printer turi noma'lum: %s": {
		"тип принтера неизвестен: %s",
		"unknown printer type: %s",
	},
	"printer turi qo'llab-quvvatlanmaydi: %s": {
		"тип принтера не поддерживается: %s",
		"that printer type is not supported: %s",
	},
	"printerga to'liq yozilmadi (%s): %d/%d": {
		"на принтер записано не полностью (%s): %d/%d",
		"the printer was not written to in full (%s): %d/%d",
	},
	"printerga ulanib bo'lmadi (%s): %s": {
		"не удалось подключиться к принтеру (%s): %s",
		"could not connect to the printer (%s): %s",
	},
	"printerga yozib bo'lmadi (%s): %s": {
		"не удалось записать на принтер (%s): %s",
		"could not write to the printer (%s): %s",
	},
	"qanday chek ekani noma'lum": {
		"непонятно, что это за чек",
		"it is not clear what kind of receipt this is",
	},
	"qarz topilmadi yoki allaqachon yopilgan": {
		"долг не найден или уже закрыт",
		"the debt was not found, or is already settled",
	},
	"qarzni kim olayotganini tanlang": {
		"выберите, за кем записан долг",
		"choose whose debt this is",
	},
	"qator noto'g'ri": {
		"строка неверная",
		"the line is wrong",
	},
	"qator topilmadi": {
		"строка не найдена",
		"line not found",
	},
	"qaysi javondan qaysi javonga ko'chirilishini tanlang": {
		"выберите, с какого склада на какой перемещать",
		"pick which store it moves from, and which it moves to",
	},
	"qayta qo'ng'iroq vaqtini yozing": {
		"укажите время повторного звонка",
		"give a time for the call back",
	},
	"qaytarish sababini yozing": {
		"укажите причину возврата",
		"give a reason for the refund",
	},
	"qaytarish uchun asl chekning fiskal belgisi kerak": {
		"для возврата нужен фискальный признак исходного чека",
		"a refund needs the original receipt's fiscal mark",
	},
	"qaytarish uchun asl chekning fiskal belgisi yo'q": {
		"у исходного чека нет фискального признака",
		"the original receipt has no fiscal mark",
	},
	"qo'ng'iroq topilmadi": {
		"звонок не найден",
		"call not found",
	},
	"qurilma aniqlanmadi": {
		"устройство не определено",
		"the device did not identify itself",
	},
	"qurilma tokeni yaroqsiz": {
		"токен устройства недействителен",
		"the device token is not valid",
	},
	"qurilma topilmadi": {
		"устройство не найдено",
		"the device was not found",
	},
	"r_keeper: %s": {
		"r_keeper: %s",
		"r_keeper: %s",
	},
	"r_keeper: %s manzilga ulanib bo'lmadi — server restoran tarmog'i ichidami? (%s)": {
		"r_keeper: не удалось подключиться к %s — сервер внутри сети ресторана? (%s)",
		"r_keeper: could not connect to %s — is the server inside the restaurant's network? (%s)",
	},
	"r_keeper: buyurtma yaratildi, lekin identifikator qaytmadi": {
		"r_keeper: заказ создан, но идентификатор не вернулся",
		"r_keeper: the order was created but no identifier came back",
	},
	"r_keeper: javobni o'qib bo'lmadi: %s": {
		"r_keeper: ответ не удалось прочитать: %s",
		"r_keeper: the answer could not be read: %s",
	},
	"r_keeper: login yoki parol qabul qilinmadi": {
		"r_keeper: логин или пароль не приняты",
		"r_keeper: the login or password was refused",
	},
	"r_keeper: menyuni o'qib bo'lmadi: %s": {
		"r_keeper: меню не удалось прочитать: %s",
		"r_keeper: the menu could not be read: %s",
	},
	"r_keeper: stansiya (kassa) kodi sozlanmagan": {
		"r_keeper: код станции (кассы) не настроен",
		"r_keeper: the station (till) code is not set up",
	},
	"raqamni o'qib bo'lmadi": {
		"число не удалось прочитать",
		"the number could not be read",
	},
	"rasm %d": {
		"изображение %d",
		"image %d",
	},
	"rasm bo'sh": {
		"изображение пустое",
		"the image is empty",
	},
	"recipe kerak": {
		"нужна техкарта",
		"a recipe is required",
	},
	"restoran profili yo'q": {
		"профиль ресторана не заполнен",
		"the restaurant profile is missing",
	},
	"rol nomini yozing": {
		"напишите название роли",
		"write the role's name",
	},
	"rol topilmadi": {
		"роль не найдена",
		"role not found",
	},
	"sababini tanlang yoki yozing": {
		"выберите или напишите причину",
		"pick a reason, or write one",
	},
	"sababini yozing": {
		"укажите причину",
		"give a reason",
	},
	"sahifani ochib bo'lmadi: %s": {
		"страницу не удалось открыть: %s",
		"the page could not be opened: %s",
	},
	"sana formati noto'g'ri": {
		"неверный формат даты",
		"that date format is not valid",
	},
	"sana formati: YYYY-MM-DD": {
		"формат даты: YYYY-MM-DD",
		"the date format is YYYY-MM-DD",
	},
	"sana noto'g'ri": {
		"дата неверная",
		"the date is wrong",
	},
	"sanalgan summa manfiy bo'la olmaydi": {
		"пересчитанная сумма не может быть отрицательной",
		"the counted amount cannot be negative",
	},
	"savatda ikki xil brend taomi bor — alohida buyurtma bering": {
		"в корзине блюда двух брендов — оформите отдельные заказы",
		"the basket has dishes from two brands — place separate orders",
	},
	"sayt %d qaytardi": {
		"сайт вернул %d",
		"the site answered %d",
	},
	"server %d: %s": {
		"сервер %d: %s",
		"server %d: %s",
	},
	"server javobi: %d": {
		"ответ сервера: %d",
		"the server answered: %d",
	},
	"sizning ichki raqamingiz ko'rsatilmagan — Hisobim bo'limida yozing": {
		"ваш внутренний номер не указан — впишите его в разделе «Мой аккаунт»",
		"your extension is missing — enter it under My account",
	},
	"smena allaqachon ochiq": {
		"смена уже открыта",
		"the shift is already open",
	},
	"smena allaqachon yopilgan": {
		"смена уже закрыта",
		"the shift is already closed",
	},
	"smena hali yopilmagan": {
		"смена ещё не закрыта",
		"the shift is still open",
	},
	"smena topilmadi": {
		"смена не найдена",
		"shift not found",
	},
	"so'rov formati noto'g'ri": {
		"неверный формат запроса",
		"the request is malformed",
	},
	"soni 1 dan %s gacha bo'lishi kerak": {
		"количество должно быть от 1 до %s",
		"the quantity has to be between 1 and %s",
	},
	"spooler yo'q": {
		"спулера нет",
		"there is no spooler",
	},
	"stol topilmadi — QR kodni qayta skaner qiling": {
		"стол не найден — отсканируйте QR-код заново",
		"the table was not found — scan the QR code again",
	},
	"stolni tanlang": {
		"выберите стол",
		"choose a table",
	},
	"summa noldan katta bo'lishi kerak": {
		"сумма должна быть больше нуля",
		"the amount has to be more than zero",
	},
	"summani yozing": {
		"введите сумму",
		"enter an amount",
	},
	"taom POS tizimiga bog'lanmagan": {
		"блюдо не привязано к кассе",
		"the dish is not mapped to the POS",
	},
	"taom id noto'g'ri": {
		"неверный id блюда",
		"invalid dish id",
	},
	"taom noto'g'ri": {
		"блюдо указано неверно",
		"that dish is not valid",
	},
	"taom topilmadi": {
		"блюдо не найдено",
		"dish not found",
	},
	"taom topilmadi — ro'yxat yangilandi": {
		"блюдо не найдено — список обновлён",
		"the dish was not found — the list has been refreshed",
	},
	"tarifingizdagi ekranlar soni to'lgan": {
		"количество экранов в вашем тарифе исчерпано",
		"your plan's screens are all in use",
	},
	"telefon raqam kerak": {
		"нужен номер телефона",
		"a phone number is required",
	},
	"telefon raqam noto'g'ri": {
		"неверный номер телефона",
		"that phone number is not valid",
	},
	"telefon raqami noto'g'ri": {
		"неверный номер телефона",
		"that phone number is not valid",
	},
	"telefon raqamini to'liq yozing": {
		"введите номер телефона полностью",
		"enter the full phone number",
	},
	"telefoniya ulanmagan": {
		"телефония не подключена",
		"telephony is not connected",
	},
	"telegram: %s": {
		"telegram: %s",
		"telegram: %s",
	},
	"telegram: %s (fallback: %s)": {
		"telegram: %s (запасной: %s)",
		"telegram: %s (fallback: %s)",
	},
	"telegram: bot tokeni sozlanmagan": {
		"telegram: токен бота не настроен",
		"telegram: the bot token is not set up",
	},
	"telegram: foydalanuvchi id yo'q": {
		"telegram: нет id пользователя",
		"telegram: there is no user id",
	},
	"telegram: foydalanuvchi ma'lumoti o'qilmadi": {
		"telegram: данные пользователя не прочитаны",
		"telegram: the user data could not be read",
	},
	"telegram: foydalanuvchi ma'lumoti yo'q": {
		"telegram: нет данных пользователя",
		"telegram: there is no user data",
	},
	"telegram: imzo to'g'ri kelmadi": {
		"telegram: подпись не сошлась",
		"telegram: the signature did not match",
	},
	"telegram: ma'lumot eskirgan": {
		"telegram: данные устарели",
		"telegram: the data is stale",
	},
	"telegram: telefon raqami topilmadi": {
		"telegram: номер телефона не найден",
		"telegram: no phone number was found",
	},
	"texkarta to'liq emas — masalliqlaridan biri yo'q": {
		"техкарта неполная — одного из ингредиентов нет",
		"the recipe is incomplete — one of its ingredients is missing",
	},
	"to'lov hali tasdiqlanmadi — mijoz to'laganini kuting": {
		"оплата ещё не подтверждена — дождитесь, пока гость заплатит",
		"the payment is not confirmed yet — wait for the guest to pay",
	},
	"to'lov havolasi olinmadi": {
		"ссылка на оплату не получена",
		"no payment link came back",
	},
	"to'lov tasdiqlanmadi va bank uni qaytaradi: %s": {
		"оплата не подтверждена, и банк её вернёт: %s",
		"the payment was not confirmed and the bank will reverse it: %s",
	},
	"to'lov topilmadi": {
		"платёж не найден",
		"payment not found",
	},
	"to'plam ichiga boshqa to'plamni qo'shib bo'lmaydi": {
		"в набор нельзя добавить другой набор",
		"a combo cannot hold another combo",
	},
	"to'plamda kamida ikkita taom bo'lishi kerak": {
		"в наборе должно быть минимум два блюда",
		"a combo needs at least two dishes",
	},
	"to'plamdagi har bir taom soni kamida 1 bo'lishi kerak": {
		"количество каждого блюда в наборе должно быть не меньше 1",
		"every dish in a combo needs a quantity of at least 1",
	},
	"to'plamdagi taom menyuda topilmadi": {
		"блюдо из набора не найдено в меню",
		"a dish in the combo is not on the menu",
	},
	"to'plamga faqat shu brend menyusidagi taomlar qo'shiladi": {
		"в набор добавляются только блюда меню этого бренда",
		"only dishes from this brand's menu can go into the combo",
	},
	"to'plamga variantlar qo'shib bo'lmaydi": {
		"варианты в набор добавить нельзя",
		"options cannot be added to a combo",
	},
	"to'plamning o'z texkartasi yo'q": {
		"у набора нет своей техкарты",
		"a combo has no tech card of its own",
	},
	"to: sana noto'g'ri": {
		"to: неверная дата",
		"to: invalid date",
	},
	"token noto'g'ri": {
		"неверный токен",
		"invalid token",
	},
	"topilmadi": {
		"не найдено",
		"not found",
	},
	"topilmadi yoki allaqachon to'langan": {
		"не найдено или уже оплачено",
		"not found, or already paid",
	},
	"urinishlar soni tugadi, qaytadan so'rang": {
		"попытки закончились, запросите заново",
		"no attempts are left, ask for a new code",
	},
	"ustun sarlavhalari tanilmadi — faylning birinchi qatorida nom, birlik, narx kabi sarlavhalar bo'lishi kerak": {
		"заголовки столбцов не распознаны — в первой строке файла должны быть заголовки: название, единица, цена",
		"the column headings were not recognised — the file's first row needs headings such as name, unit and price",
	},
	"vakansiya topilmadi": {
		"вакансия не найдена",
		"vacancy not found",
	},
	"vaqt formati noto'g'ri": {
		"неверный формат времени",
		"that time format is not valid",
	},
	"vaqt ko'rsatilmagan": {
		"время не указано",
		"no time was given",
	},
	"vaqt noto'g'ri (HH:MM)": {
		"время неверное (HH:MM)",
		"the time is wrong (HH:MM)",
	},
	"vaqtni tanlang": {
		"выберите время",
		"choose a time",
	},
	"webhook uchun HTTPS manzil kerak (hozir: %s)": {
		"для webhook нужен HTTPS-адрес (сейчас: %s)",
		"a webhook needs an HTTPS address (now: %s)",
	},
	"xabar juda uzun": {
		"сообщение слишком длинное",
		"the message is too long",
	},
	"xabar matni bo'sh": {
		"текст сообщения пуст",
		"the message is empty",
	},
	"xaritada joyni belgilang": {
		"отметьте место на карте",
		"mark the spot on the map",
	},
	"xizmat topilmadi": {
		"служба не найдена",
		"service not found",
	},
	"yakunlangan buyurtmani ko'chirib bo'lmaydi": {
		"завершённый заказ переместить нельзя",
		"a finished order cannot be moved",
	},
	"yarim tayyor mahsulot javonda o'zi bo'lib turmaydi: uni tashkil qilgan masalliqlarni ko'chiring": {
		"полуфабрикат сам по себе на полке не лежит: перемещайте ингредиенты, из которых он состоит",
		"a semi-product does not sit on a shelf by itself: move the ingredients it is made of",
	},
	"yetkazib beruvchining nomi kerak": {
		"нужно название поставщика",
		"the supplier needs a name",
	},
	"СТИР (ИНН) kiritilmagan": {
		"СТИР (ИНН) не введён",
		"the taxpayer number (STIR/INN) is missing",
	},
}
