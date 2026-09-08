package uz.keel.guest.i18n

import uz.keel.design.Lang

// Every word this application says on its own behalf.
//
// ⚠️ **A typed dictionary rather than string resources**, exactly as the five
// staff applications do it: a missing key is a compile error here and a blank
// space on the screen there. In an application a guest sees, a blank space is
// the restaurant looking unfinished.
//
// ⚠️ **Uzbek is the base and all three are written together.** A language added
// later is a language that stays half-done — the lesson `lib/i18n/growth.ts`
// paid for on the web side.
//
// ⚠️ **The restaurant's own words are not here.** Dish names, category names and
// descriptions come from the panel in whatever languages the owner filled in,
// and fall back to Uzbek (`Named.pick`). Text an owner typed can never come out
// of a dictionary we ship.

data class Dict(
    /** This language, named in itself — "Русский", never "Ruscha". */
    val lang: String,
    val menu: Menu,
    val common: Common,
) {
    data class Menu(
        /** ⚠️ **There is no home page and this is the first thing on screen.**
         *  A restaurant application opened by somebody who is hungry has one
         *  job, and a landing page is a tap between them and it. */
        val search: String,
        val searchEmpty: (String) -> String,
        val all: String,
        val soldOut: String,
        val closed: String,
        val closedHint: String,
        val empty: String,
        val loadFailed: String,
        val retry: String,
    )

    data class Common(
        val ok: String,
        val retry: String,
        val loading: String,
        val back: String,
    )
}

private val uz = Dict(
    lang = "O'zbekcha",
    menu = Dict.Menu(
        search = "Taom qidirish",
        searchEmpty = { q -> "«$q» bo'yicha hech nima topilmadi" },
        all = "Hammasi",
        soldOut = "Tugadi",
        closed = "Hozir yopiq",
        // ⚠️ Menyu baribir ko'rsatiladi. Yopiq restoranning menyusini
        // yashirish — ertaga keladigan odamni bugun yo'qotish.
        closedHint = "Menyuni ko'rishingiz mumkin, buyurtma ish vaqtida qabul qilinadi",
        empty = "Menyu hali to'ldirilmagan",
        loadFailed = "Menyuni ochib bo'lmadi",
        retry = "Qayta urinish",
    ),
    common = Dict.Common(
        ok = "Yaxshi",
        retry = "Qayta urinish",
        loading = "Yuklanmoqda…",
        back = "Orqaga",
    ),
)

private val ru = Dict(
    lang = "Русский",
    menu = Dict.Menu(
        search = "Поиск по меню",
        searchEmpty = { q -> "По запросу «$q» ничего не найдено" },
        all = "Всё",
        soldOut = "Закончилось",
        closed = "Сейчас закрыто",
        closedHint = "Меню можно посмотреть, заказы принимаются в рабочее время",
        empty = "Меню пока не заполнено",
        loadFailed = "Не удалось открыть меню",
        retry = "Повторить",
    ),
    common = Dict.Common(
        ok = "Хорошо",
        retry = "Повторить",
        loading = "Загрузка…",
        back = "Назад",
    ),
)

private val en = Dict(
    lang = "English",
    menu = Dict.Menu(
        search = "Search the menu",
        searchEmpty = { q -> "Nothing found for \"$q\"" },
        all = "All",
        soldOut = "Sold out",
        closed = "Closed right now",
        closedHint = "You can browse the menu; orders are taken during opening hours",
        empty = "The menu has not been filled in yet",
        loadFailed = "Could not open the menu",
        retry = "Try again",
    ),
    common = Dict.Common(
        ok = "OK",
        retry = "Try again",
        loading = "Loading…",
        back = "Back",
    ),
)

val DICTS: Map<Lang, Dict> = mapOf(Lang.Uz to uz, Lang.Ru to ru, Lang.En to en)
