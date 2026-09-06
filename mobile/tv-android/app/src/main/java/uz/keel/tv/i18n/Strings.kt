package uz.keel.tv.i18n

import uz.keel.design.Lang

// Everything this television ever says, in the three languages a restaurant here
// is run in.
//
// ⚠️ **A very short dictionary, and that is the design rather than an
// omission.** Four of these lines are read by one person once, while they mount
// the set; the rest are read by a room, from four metres, and every one of them
// is a sentence somebody might have to act on. A television with more words than
// this is a television explaining itself to people who cannot do anything about
// it.
//
// ⚠️ **Data classes with named arguments, not a map.** A map makes a missing key
// a blank line on a wall in front of forty guests; this makes it a compile error.

data class Dict(
    val lang: String,
    val appName: String,
    val server: Server,
    val pairing: Pairing,
    val idle: Idle,
) {
    data class Server(
        val title: String,
        val hint: String,
        val placeholder: String,
        val next: String,
        val bad: String,
        /** The address was accepted and led to no Keel server. */
        val unreachable: String,
        /** ⚠️ Its own sentence, because it names a different culprit: the server
         *  answered, so the restaurant's Keel is simply older than this app —
         *  and nothing on the remote in somebody's hand can fix that. */
        val outdated: String,
    )

    data class Pairing(
        val lead: String,
        val where: String,
        val renews: (Int) -> String,
        val renewing: String,
    )

    data class Idle(
        val connecting: String,
        val offline: String,
        val downloading: String,
        val checking: String,
        val noContent: String,
        val cooking: String,
        val ready: String,
    )
}

val UZ = Dict(
    lang = "O'zbekcha",
    appName = "Keel TV",
    server = Dict.Server(
        title = "Restoran manzilini kiriting",
        // ⚠️ The short form is the example, because it is what the person
        // mounting the screen knows. A placeholder showing a full URL teaches
        // the wrong answer to everybody who reads it.
        hint = "Masalan: osh — yoki to'liq manzil, agar o'z domeningiz bo'lsa",
        placeholder = "osh",
        next = "Davom etish",
        bad = "Manzil noto'g'ri — restoran nomini yozing, masalan: osh",
        // ⚠️ The causes in the order they actually happen, because the person
        // reading this is standing on a chair with a remote and has no other way
        // to find out which one it is.
        unreachable = "Bu manzilda javob yo'q. Nomini tekshiring, internetni " +
            "tekshiring — ulanish tiklansa ekran o'zi davom etadi.",
        outdated = "Server javob berdi, lekin TV bo'limi yo'q — restoran serveri " +
            "yangilanishi kerak.",
    ),
    pairing = Dict.Pairing(
        lead = "Bu ekranni ulash uchun",
        where = "Keel panelida: TV ekranlar → Ekran qo'shish",
        renews = { s -> "Kod $s soniyadan keyin yangilanadi" },
        renewing = "Kod yangilanmoqda…",
    ),
    idle = Dict.Idle(
        connecting = "Ulanmoqda…",
        offline = "Aloqa yo'q — ulanish tiklanganda o'zi sinxronlashadi",
        downloading = "Kontent yuklanmoqda…",
        checking = "Kontent tekshirilmoqda…",
        noContent = "Kontent yo'q — Keel panelida: TV ekranlar → Kontent",
        cooking = "Tayyorlanmoqda",
        ready = "Tayyor",
    ),
)

val RU = Dict(
    lang = "Русский",
    appName = "Keel TV",
    server = Dict.Server(
        title = "Введите адрес ресторана",
        hint = "Например: osh — или полный адрес, если у вас свой домен",
        placeholder = "osh",
        next = "Продолжить",
        bad = "Неверный адрес — напишите название ресторана, например: osh",
        unreachable = "По этому адресу нет ответа. Проверьте название и интернет — " +
            "когда связь вернётся, экран продолжит сам.",
        outdated = "Сервер ответил, но раздела ТВ нет — сервер ресторана нужно обновить.",
    ),
    pairing = Dict.Pairing(
        lead = "Чтобы подключить этот экран",
        where = "В панели Keel: ТВ экраны → Добавить экран",
        renews = { s -> "Код обновится через $s сек" },
        renewing = "Код обновляется…",
    ),
    idle = Dict.Idle(
        connecting = "Подключение…",
        offline = "Нет связи — синхронизируется, когда она вернётся",
        downloading = "Загрузка контента…",
        checking = "Проверка контента…",
        noContent = "Контента нет — в панели Keel: ТВ экраны → Контент",
        cooking = "Готовится",
        ready = "Готово",
    ),
)

val EN = Dict(
    lang = "English",
    appName = "Keel TV",
    server = Dict.Server(
        title = "Enter the restaurant's address",
        hint = "For example: osh — or the full address if you have your own domain",
        placeholder = "osh",
        next = "Continue",
        bad = "That is not an address — type the restaurant's name, for example: osh",
        unreachable = "Nothing answers at this address. Check the name and the " +
            "internet — the screen carries on by itself once it is back.",
        outdated = "The server answered but has no TV section — the restaurant's " +
            "server needs updating.",
    ),
    pairing = Dict.Pairing(
        lead = "To connect this screen",
        where = "In the Keel panel: TV screens → Add screen",
        renews = { s -> "The code changes in $s s" },
        renewing = "Getting a new code…",
    ),
    idle = Dict.Idle(
        connecting = "Connecting…",
        offline = "No connection — it syncs itself when this comes back",
        downloading = "Downloading content…",
        checking = "Checking for content…",
        noContent = "No content — in the Keel panel: TV screens → Content",
        cooking = "Cooking",
        ready = "Ready",
    ),
)

val DICTS: Map<Lang, Dict> = mapOf(Lang.Uz to UZ, Lang.Ru to RU, Lang.En to EN)
