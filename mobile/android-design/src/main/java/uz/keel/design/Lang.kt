package uz.keel.design

import androidx.compose.runtime.Composable
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.runtime.compositionLocalOf

// What the design system needs to know about an application, and nothing more.
//
// ⚠️ **The module must not know the apps.** A shared component that reached into
// `uz.keel.waiter` would make the owner app depend on the waiter app — and the
// first thing anybody would do about that is copy the component, which is the
// duplication this module exists to end. So the two things a control genuinely
// needs — which language, and half a dozen words of its own — come in through
// the composition, and each app answers in its own dictionary.

/** The three languages a restaurant here is run in.
 *
 *  ⚠️ **One enum for both apps**, because a phone that is Russian in one and
 *  Uzbek in the other is a phone somebody reports as broken. */
enum class Lang(val code: String) {
    Uz("uz"), Ru("ru"), En("en");

    companion object {
        fun of(code: String?): Lang = entries.firstOrNull { it.code == code } ?: Uz
    }
}

/** How an application answers "which language, and what is it called".
 *
 *  ⚠️ `nameOf` returns each language **in itself** — "Русский", never "Ruscha".
 *  A list that names a language in a language somebody cannot read is a list
 *  they have to decode before they can leave the language they cannot read. */
interface LangHost {
    val current: Lang
    fun nameOf(l: Lang): String
    fun set(l: Lang)
}

val LocalLangHost = compositionLocalOf<LangHost> { error("LangHost berilmagan") }

/** The few words the shared controls say on their own behalf.
 *
 *  ⚠️ **Passed in rather than held here.** A dictionary inside this module would
 *  be a third place translations live, and the one that nobody remembers to
 *  update is always the one furthest from the screen. */
data class DesignWords(
    val ok: String,
    val retry: String,
    val loading: String,
)

val LocalWords = compositionLocalOf {
    DesignWords(ok = "OK", retry = "Qayta urinish", loading = "Yuklanmoqda…")
}

val words: DesignWords
    @Composable @ReadOnlyComposable get() = LocalWords.current

/** Minutes as "8 soat 30 daq".
 *
 *  ⚠️ Ported from `lib/attendance.ts` → `formatDuration`, labels passed in so one
 *  helper serves all three languages — the same shape as the original. */
fun formatDuration(minutes: Int, hourLabel: String, minuteLabel: String): String {
    val sign = if (minutes < 0) "-" else ""
    val total = kotlin.math.abs(minutes)
    val h = total / 60
    val m = total % 60
    return when {
        h == 0 -> "$sign$m $minuteLabel"
        m == 0 -> "$sign$h $hourLabel"
        else -> "$sign$h $hourLabel $m $minuteLabel"
    }
}

/** How long ago an ISO instant was. Ported from `lib/orderFlow.ts` → `timeAgo`.
 *
 *  ⚠️ **Never a date arithmetic on a parsed local time.** Times come off the wire
 *  in UTC — the trap this codebase has been bitten by on the server side too —
 *  and this works on the difference between two instants, which is timezone-free
 *  by construction. */
fun timeAgo(
    iso: String?,
    now: String,
    min: (Int) -> String,
    hour: (Int) -> String,
    day: (Int) -> String,
): String {
    if (iso.isNullOrBlank()) return now
    val then = runCatching { java.time.Instant.parse(iso).toEpochMilli() }.getOrNull() ?: return now
    val m = Math.round((System.currentTimeMillis() - then) / 60000.0).toInt()
    if (m < 1) return now
    if (m < 60) return min(m)
    val h = Math.round(m / 60.0).toInt()
    if (h < 24) return hour(h)
    return day(Math.round(h / 24.0).toInt())
}
