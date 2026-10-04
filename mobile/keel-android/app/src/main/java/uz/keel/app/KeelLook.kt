package uz.keel.app

import androidx.compose.runtime.MutableState
import androidx.compose.runtime.mutableStateOf
import uz.keel.design.Lang
import uz.keel.design.ThemeChoice
import uz.keel.design.TokenStore

// The language and the scheme this phone is read in.
//
// ⚠️ **One pair of states for the whole app, handed to every role's `Prefs`.**
// The four apps this one unites each read the language into a state of their
// own, which was right while each was alone on the phone. Here they share a
// screen: a language chosen in the owner's settings has to already be the
// language of the floor, and of this app's own sign-in and switcher.
//
// ⚠️ **Read in the initialiser, from a store hydrated before any of this runs**
// — the same ordering every Keel app depends on. A first frame painted in Uzbek
// and then switched is a flash somebody reads as a fault.
class KeelLook(private val store: TokenStore) {

    val lang: MutableState<Lang> = mutableStateOf(Lang.of(store.read(TokenStore.LANG)))

    val theme: MutableState<ThemeChoice> = mutableStateOf(
        when (store.read(TokenStore.THEME)) {
            "light" -> ThemeChoice.Light
            "dark" -> ThemeChoice.Dark
            else -> ThemeChoice.System
        },
    )

    fun setLang(l: Lang) {
        lang.value = l
        store.write(TokenStore.LANG, l.code)
    }
}
