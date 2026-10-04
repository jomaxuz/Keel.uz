package uz.keel.waiter

import androidx.compose.runtime.Composable
import androidx.compose.runtime.MutableState
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.mutableStateOf
import uz.keel.design.TokenStore
import uz.keel.app.KeelLook
import uz.keel.waiter.i18n.DICTS
import uz.keel.waiter.i18n.Dict
import uz.keel.waiter.i18n.Lang
import uz.keel.waiter.i18n.UZ
import uz.keel.design.*
import uz.keel.waiter.ui.components.MenuView

// Language, appearance and how the menu is drawn: what somebody chose,
// remembered.
//
// ⚠️ **Not the web's mechanism.** On the site the language lives in the URL and
// a cookie, because a page has to be linkable and indexable in each language. A
// phone has neither problem — so the choice is a value on the device, and
// copying the web's machinery would have brought a router along with it.
//
// ⚠️ **Read from what was already hydrated, in the initialiser.** The first
// frame has to be in the right language: a screen that painted Uzbek and then
// switched to Russian is a flash somebody reads as a fault. `TokenStore` reads
// everything on construction, before any of this runs — that ordering is a bug
// the Expo app actually shipped, where the setting was saved correctly the whole
// time and read too early.
class Prefs(private val store: TokenStore, look: KeelLook) {

    // ⚠️ **Keel's, not this role's.** One phone, one person, one language: the
    // four roles in this app read the same two states, so a language chosen in
    // the owner's settings is already the language of the floor. Four copies
    // would each be right until somebody switched — and then three would lie.
    val lang: MutableState<Lang> = look.lang
    val theme: MutableState<ThemeChoice> = look.theme
    /** ⚠️ Remembered per phone, not per restaurant: it is a preference of the
     *  person holding it. A café with forty drinks and no photographs wants a
     *  list; a photographed menu wants the picture. */
    val view: MutableState<MenuView> = mutableStateOf(
        when (store.read(TokenStore.MENU_VIEW)) {
            "cards" -> MenuView.Cards
            "photos" -> MenuView.Photos
            else -> MenuView.List
        },
    )

    val dict: Dict get() = DICTS[lang.value] ?: UZ

    fun setLang(l: Lang) {
        lang.value = l
        store.write(TokenStore.LANG, l.code)
    }

    fun setTheme(c: ThemeChoice) {
        theme.value = c
        store.write(TokenStore.THEME, c.name.lowercase())
    }

    fun setView(v: MenuView) {
        view.value = v
        store.write(TokenStore.MENU_VIEW, v.name.lowercase())
    }
}

/** ⚠️ Handed down rather than passed screen by screen: every screen needs the
 *  dictionary, and a screen that has to remember a second parameter is a screen
 *  that forgets it. */
val LocalPrefs = compositionLocalOf<Prefs> { error("Prefs yo'q") }

val t: Dict
    @Composable get() = LocalPrefs.current.dict

/** How the shared controls reach this app's language.
 *
 *  ⚠️ The design module must not import the app — so the app hands it the two
 *  things a control needs and keeps the dictionary here. */
fun Prefs.langHost(): uz.keel.design.LangHost = object : uz.keel.design.LangHost {
    override val current: Lang get() = lang.value
    override fun nameOf(l: Lang): String = DICTS[l]!!.lang
    override fun set(l: Lang) = setLang(l)
}
