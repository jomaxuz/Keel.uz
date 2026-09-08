package uz.keel.guest

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.mutableStateOf
import uz.keel.design.Lang
import uz.keel.design.LangHost
import uz.keel.design.ThemeChoice
import uz.keel.guest.i18n.DICTS
import uz.keel.guest.i18n.Dict

// What this phone remembers between launches.
//
// ⚠️ **The language is not guessed from the system locale.** A phone in
// Uzbekistan is as likely to be set to Russian as to Uzbek, and a menu that
// opens in the wrong one teaches the guest that the app is somebody else's. The
// first launch shows the restaurant's own default and the switch is one tap
// away, in the header, where it can be found without opening settings.
class Prefs(context: Context) {

    private val store = context.getSharedPreferences("keel-guest", Context.MODE_PRIVATE)

    val lang = mutableStateOf(
        Lang.entries.firstOrNull { it.code == store.getString(KEY_LANG, null) } ?: Lang.Uz,
    )
    val theme = mutableStateOf(
        runCatching { ThemeChoice.valueOf(store.getString(KEY_THEME, "System")!!) }
            .getOrDefault(ThemeChoice.System),
    )

    /** The restaurant's accent as the panel last reported it.
     *
     *  ⚠️ **Remembered on the phone, not only held in memory.** The theme wraps
     *  everything, including the first frame — which is drawn before
     *  `/restaurant` has answered. Without a stored value an owner who changed
     *  their colour would see the old one flash on every launch; with it, the
     *  app is only ever wrong once, on the launch after the change.
     *
     *  ⚠️ Empty means "use the build's own" (`Brand.accent`), which is the
     *  colour the panel had when the app was built — never Keel's orange unless
     *  the restaurant genuinely has no colour set. */
    val accent = mutableStateOf(store.getString(KEY_ACCENT, "").orEmpty())

    fun setAccent(hex: String) {
        if (hex == accent.value) return
        accent.value = hex
        store.edit().putString(KEY_ACCENT, hex).apply()
    }

    val dict: Dict get() = DICTS[lang.value]!!

    fun setLang(value: Lang) {
        lang.value = value
        store.edit().putString(KEY_LANG, value.code).apply()
    }

    fun setTheme(value: ThemeChoice) {
        theme.value = value
        store.edit().putString(KEY_THEME, value.name).apply()
    }

    /** ⚠️ The design module deliberately does not know this app; it asks for the
     *  language and its own few words through the composition. */
    fun langHost(): LangHost = object : LangHost {
        override val current: Lang get() = lang.value
        override fun nameOf(l: Lang): String = DICTS[l]!!.lang
        override fun set(l: Lang) = setLang(l)
    }

    private companion object {
        const val KEY_LANG = "lang"
        const val KEY_THEME = "theme"
        const val KEY_ACCENT = "accent"
    }
}

val LocalPrefs = compositionLocalOf<Prefs> { error("Prefs yo'q") }

val t: Dict
    @Composable get() = LocalPrefs.current.dict
