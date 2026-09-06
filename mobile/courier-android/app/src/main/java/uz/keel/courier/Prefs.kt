package uz.keel.courier

import androidx.compose.runtime.Composable
import androidx.compose.runtime.MutableState
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.mutableStateOf
import uz.keel.design.Lang
import uz.keel.design.LangHost
import uz.keel.design.ThemeChoice
import uz.keel.design.TokenStore
import uz.keel.courier.i18n.DICTS
import uz.keel.courier.i18n.Dict
import uz.keel.courier.i18n.UZ

// What this phone remembers about the person holding it: the language they read
// and the scheme they can see in.
//
// ⚠️ **Read in the initialiser, from a store hydrated before any of this runs.**
// The first frame has to be in the right language and the right scheme — a
// screen that painted Uzbek on white and then switched is a flash somebody reads
// as a fault. The waiter app shipped that bug once, from a hydration started
// below the provider that read it; the ordering is the whole of the fix.
//
// ⚠️ **The theme is a real choice here.** A courier reads this at a lit door at
// nine in the evening and in the sun at noon, and the phone already knows which
// — "system" is the honest default, and the other two exist for the people whose
// phone is set the way they do not want to read.
class Prefs(private val store: TokenStore) {

    val lang: MutableState<Lang> = mutableStateOf(Lang.of(store.read(TokenStore.LANG)))

    val theme: MutableState<ThemeChoice> = mutableStateOf(
        when (store.read(TokenStore.THEME)) {
            "light" -> ThemeChoice.Light
            "dark" -> ThemeChoice.Dark
            else -> ThemeChoice.System
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

    /** How the shared controls reach this application's language. */
    fun langHost(): LangHost = object : LangHost {
        override val current: Lang get() = lang.value
        override fun nameOf(l: Lang): String = DICTS[l]!!.lang
        override fun set(l: Lang) = setLang(l)
    }
}

val LocalPrefs = compositionLocalOf<Prefs> { error("Prefs yo'q") }

val t: Dict
    @Composable get() = LocalPrefs.current.dict
