package uz.keel.tv

import androidx.compose.runtime.Composable
import androidx.compose.runtime.MutableState
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.mutableStateOf
import uz.keel.design.Lang
import uz.keel.design.LangHost
import uz.keel.design.TokenStore
import uz.keel.tv.i18n.DICTS
import uz.keel.tv.i18n.Dict
import uz.keel.tv.i18n.UZ

// The one thing a television remembers about a person: which language they read.
//
// ⚠️ **Read in the initialiser, from a store hydrated before any of this runs.**
// The first frame has to be in the right language — a screen that painted Uzbek
// and switched to Russian is a flash somebody reads as a fault, and on a wall
// there is nobody to tell otherwise.
//
// ⚠️ **No theme choice, and that is not a missing feature.** A television in a
// dining room is dark, always: the light scheme on a two-metre panel above a
// table is a lamp nobody asked for, and the four people who could change it are
// not standing in front of it. The rest of the product keeps its switch.
class Prefs(private val store: TokenStore) {

    val lang: MutableState<Lang> = mutableStateOf(Lang.of(store.read(TokenStore.LANG)))

    val dict: Dict get() = DICTS[lang.value] ?: UZ

    fun setLang(l: Lang) {
        lang.value = l
        store.write(TokenStore.LANG, l.code)
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
