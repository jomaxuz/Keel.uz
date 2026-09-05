package uz.keel.owner

import androidx.compose.runtime.Composable
import androidx.compose.runtime.MutableState
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.mutableStateOf
import uz.keel.design.Lang
import uz.keel.design.LangHost
import uz.keel.design.ThemeChoice
import uz.keel.design.TokenStore
import uz.keel.owner.i18n.DICTS
import uz.keel.owner.i18n.Dict
import uz.keel.owner.i18n.UZ

// Language and appearance: what somebody chose, remembered.
//
// ⚠️ **Read in the initialiser, from a store hydrated before any of this runs.**
// The first frame has to be in the right language — a screen that painted Uzbek
// and then switched to Russian is a flash somebody reads as a fault. The waiter
// application shipped that bug once; the ordering is the whole of the fix.
class Prefs(private val store: TokenStore) {

    val lang: MutableState<Lang> = mutableStateOf(Lang.of(store.read(TokenStore.LANG)))
    val theme: MutableState<ThemeChoice> = mutableStateOf(
        when (store.read(TokenStore.THEME)) {
            "light" -> ThemeChoice.Light
            "dark" -> ThemeChoice.Dark
            else -> ThemeChoice.System
        },
    )

    /** Which branch the numbers are about.
     *
     *  ⚠️ **Remembered, because it is a question an owner answers once a day and
     *  not once a screen.** Somebody who runs three restaurants opens this to
     *  look at one of them; resetting to "all" on every launch would make the
     *  most common reading the one that costs two taps. */
    val branch: MutableState<String> = mutableStateOf(store.read(BRANCH) ?: "")

    val dict: Dict get() = DICTS[lang.value] ?: UZ

    fun setLang(l: Lang) {
        lang.value = l
        store.write(TokenStore.LANG, l.code)
    }

    fun setTheme(c: ThemeChoice) {
        theme.value = c
        store.write(TokenStore.THEME, c.name.lowercase())
    }

    fun setBranch(id: String) {
        branch.value = id
        if (id.isEmpty()) store.drop(BRANCH) else store.write(BRANCH, id)
    }

    /** How the shared controls reach this application's language. */
    fun langHost(): LangHost = object : LangHost {
        override val current: Lang get() = lang.value
        override fun nameOf(l: Lang): String = DICTS[l]!!.lang
        override fun set(l: Lang) = setLang(l)
    }

    companion object {
        const val BRANCH = "keel_owner_branch"
    }
}

val LocalPrefs = compositionLocalOf<Prefs> { error("Prefs yo'q") }

val t: Dict
    @Composable get() = LocalPrefs.current.dict
