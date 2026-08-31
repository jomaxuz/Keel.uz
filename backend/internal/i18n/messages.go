package i18n

// messages is the catalogue. ⚠️ Sorted by the Uzbek text so a diff shows
// what changed rather than where it moved, and so a duplicate key is a
// compile error rather than a silently ignored second entry.
var messages = map[string]pair{
	"Chegirma berish": {
		"Скидка",
		"Giving a discount",
	},
	"Kassa smenasi": {
		"Кассовая смена",
		"The cash shift",
	},
	"Oshxona ekrani": {
		"Кухонный экран",
		"The kitchen screen",
	},
	"Omborni sanash": {
		"Пересчёт склада",
		"Counting the store",
	},
	"Pishirilgan taomni olib tashlash": {
		"Удаление приготовленного блюда",
		"Removing a cooked dish",
	},
	"To'lovni qabul qilish": {
		"Приём оплаты",
		"Taking payment",
	},
	"Zal ekrani": {
		"Экран зала",
		"The dining-room screen",
	},
	"Guruh yoki kanal ID si kiritilmagan.": {
		"ID группы или канала не указан.",
		"no group or channel ID was given.",
	},
	"Ochiq smena yo'q — avval ishga kirishni bosing": {
		"Нет открытой смены — сначала отметьте приход",
		"No shift is open — clock in first",
	},
	"PIN noto'g'ri": {
		"неверный PIN",
		"wrong PIN",
	},
	"QQS stavkasi 0 dan 100 gacha bo'lishi kerak": {
		"ставка НДС должна быть от 0 до 100",
		"the VAT rate has to be between 0 and 100",
	},
	"QR kod o'qilmadi — qayta skanerlang": {
		"QR-код не прочитан — отсканируйте ещё раз",
		"the QR code was not read — scan it again",
	},
	"avtorizatsiya kerak": {
		"требуется авторизация",
		"authorisation required",
	},
	"avval shu brendning filiallarini o'chiring": {
		"сначала удалите филиалы этого бренда",
		"delete this brand's branches first",
	},
	"bahoni tanlang": {
		"выберите оценку",
		"choose a rating",
	},
	"banner rasmi kerak": {
		"нужна картинка баннера",
		"the banner needs an image",
	},
	"bekor qilish sababini yozing": {
		"укажите причину отмены",
		"give a reason for the cancellation",
	},
	"bir so'rovda 50 tagacha chek": {
		"не более 50 чеков за один запрос",
		"at most 50 receipts per request",
	},
	"brend nomini yozing": {
		"введите название бренда",
		"enter the brand's name",
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
	"bu buyurtma API orqali chaqirilmagan": {
		"этот заказ не был вызван через API",
		"this order was not requested through the API",
	},
	"bu buyurtma o'zgargan — ro'yxat yangilandi": {
		"заказ изменился — список обновлён",
		"this order changed — the list has been refreshed",
	},
	"bu buyurtmaga baho allaqachon qoldirilgan": {
		"оценка по этому заказу уже оставлена",
		"this order has already been rated",
	},
	"bu chek allaqachon qaytarilgan": {
		"по этому чеку уже был возврат",
		"this receipt has already been refunded",
	},
	"bu filial boshqa brendga tegishli": {
		"этот филиал принадлежит другому бренду",
		"that branch belongs to another brand",
	},
	"bu filialda bunday ombor yo'q": {
		"в этом филиале нет такого склада",
		"this branch has no such store",
	},
	"bu foydalanuvchida allaqachon panel hisobi bor": {
		"у этого пользователя уже есть учётная запись в панели",
		"this person already has a panel account",
	},
	"bu hisobda telefon raqami yo'q": {
		"в этой учётной записи нет номера телефона",
		"this account has no phone number",
	},
	"bu kassa kun yakunini qo'llab-quvvatlamaydi": {
		"эта касса не поддерживает закрытие дня",
		"this till does not support closing the day",
	},
	"bu kuryer boshqa filialga biriktirilgan": {
		"этот курьер закреплён за другим филиалом",
		"that courier belongs to another branch",
	},
	"bu login band": {
		"этот логин занят",
		"that login is taken",
	},
	"bu raqam boshqa admin hisobida band": {
		"этот номер занят другим администратором",
		"another administrator is using that number",
	},
	"bu raqam boshqa foydalanuvchida band": {
		"этот номер занят другим пользователем",
		"another customer is using that number",
	},
	"bu raqam boshqa hisobga tegishli": {
		"этот номер принадлежит другой учётной записи",
		"that number belongs to another account",
	},
	"bu sana juda uzoq": {
		"эта дата слишком далеко",
		"that date is too far ahead",
	},
	"bu sanoq allaqachon izohlangan": {
		"по этому пересчёту комментарий уже оставлен",
		"this count already has a note",
	},
	"bu segmentda xabar yuboradigan odam yo'q": {
		"в этом сегменте некому отправлять",
		"there is nobody in this segment to write to",
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
	"bu vakansiya endi yopilgan": {
		"эта вакансия уже закрыта",
		"this vacancy is closed",
	},
	"bu vaqt allaqachon o'tib ketgan": {
		"это время уже прошло",
		"that time has already passed",
	},
	"bu vaqtga bo'sh stol qolmadi": {
		"на это время свободных столов нет",
		"no free tables at that time",
	},
	"bunday login topilmadi": {
		"такой логин не найден",
		"no such login",
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
	"chop etish topshirig'i topilmadi": {
		"задание печати не найдено",
		"print job not found",
	},
	"davrni tanlang": {
		"выберите период",
		"choose a period",
	},
	"domen kerak": {
		"нужен домен",
		"a domain is required",
	},
	"domen noto'g'ri": {
		"домен указан неверно",
		"that domain is not valid",
	},
	"fayl juda katta": {
		"файл слишком большой",
		"the file is too large",
	},
	"fayl yuborilmadi": {
		"файл не отправлен",
		"no file was sent",
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
	"filial topilmadi": {
		"филиал не найден",
		"branch not found",
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
	"hech bo'lmasa bitta masalliq kerak": {
		"нужен хотя бы один ингредиент",
		"at least one ingredient is required",
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
	"hisobingiz filialga biriktirilmagan — administratorga murojaat qiling": {
		"ваша учётная запись не закреплена за филиалом — обратитесь к администратору",
		"your account is not attached to a branch — ask an administrator",
	},
	"holat noto'g'ri": {
		"неверный статус",
		"that status is not valid",
	},
	"id noto'g'ri": {
		"неверный id",
		"invalid id",
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
	"kiosk sozlanmagan": {
		"киоск не настроен",
		"the kiosk is not set up",
	},
	"kirish talab qilinadi": {
		"требуется вход",
		"sign-in required",
	},
	"koordinata yo'q": {
		"нет координат",
		"no coordinates",
	},
	"kuryer topilmadi": {
		"курьер не найден",
		"courier not found",
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
	"noma'lum to'lov tizimi": {
		"неизвестная платёжная система",
		"unknown payment provider",
	},
	"noma'lum to'lov turi": {
		"неизвестный способ оплаты",
		"unknown payment method",
	},
	"nomini yozing": {
		"введите название",
		"enter a name",
	},
	"noto'g'ri id": {
		"неверный id",
		"invalid id",
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
	"parol kamida 5 belgi bo'lishi kerak": {
		"пароль должен быть не короче 5 символов",
		"the password must be at least 5 characters",
	},
	"parol kamida 6 belgi bo'lishi kerak": {
		"пароль должен быть не короче 6 символов",
		"the password must be at least 6 characters",
	},
	"printer topilmadi": {
		"принтер не найден",
		"printer not found",
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
	"qator topilmadi": {
		"строка не найдена",
		"line not found",
	},
	"qayta qo'ng'iroq vaqtini yozing": {
		"укажите время повторного звонка",
		"give a time for the call back",
	},
	"qaytarish sababini yozing": {
		"укажите причину возврата",
		"give a reason for the refund",
	},
	"qo'ng'iroq topilmadi": {
		"звонок не найден",
		"call not found",
	},
	"qurilma topilmadi": {
		"устройство не найдено",
		"device not found",
	},
	"recipe kerak": {
		"нужна техкарта",
		"a recipe is required",
	},
	"restoran profili yo'q": {
		"профиль ресторана не заполнен",
		"the restaurant profile is missing",
	},
	"rol topilmadi": {
		"роль не найдена",
		"role not found",
	},
	"sababini yozing": {
		"укажите причину",
		"give a reason",
	},
	"sana formati noto'g'ri": {
		"неверный формат даты",
		"that date format is not valid",
	},
	"sanalgan summa manfiy bo'la olmaydi": {
		"пересчитанная сумма не может быть отрицательной",
		"the counted amount cannot be negative",
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
	"stolni tanlang": {
		"выберите стол",
		"choose a table",
	},
	"summani yozing": {
		"введите сумму",
		"enter an amount",
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
	"to'lov topilmadi": {
		"платёж не найден",
		"payment not found",
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
	"vakansiya topilmadi": {
		"вакансия не найдена",
		"vacancy not found",
	},
	"vaqt formati noto'g'ri": {
		"неверный формат времени",
		"that time format is not valid",
	},
	"vaqtni tanlang": {
		"выберите время",
		"choose a time",
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
	"yetkazib beruvchining nomi kerak": {
		"нужно название поставщика",
		"the supplier needs a name",
	},
}
