package uz.keel.app.push

import androidx.compose.runtime.Immutable

/** What happened when this phone subscribed, in the one shape every role's
 *  settings screen can read. `key` is the role enums' own spelling:
 *  working · asking · denied · noProject · failed. */
@Immutable
data class KeelPush(
    val key: String,
    /** The raw reason, untranslated on purpose — a diagnostic somebody
     *  searches for. */
    val detail: String?,
    val retry: () -> Unit,
)
