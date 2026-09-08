package uz.keel.guest

import androidx.compose.ui.graphics.Color

// ---- What makes this build one restaurant's app ----
//
// ⚠️ **Read, never written.** Every value here comes from `app/brand.properties`
// through `BuildConfig`, and that file is the only thing the build pipeline
// edits. A screen that reached past this object for a colour or an address
// would be a second place a restaurant's identity lives, and the second place is
// the one that keeps Keel's orange after somebody changes the brand.
//
// ⚠️ **The server address is a constant, not a setting.** The five staff
// applications open on a server screen because a manager can be told an address;
// a guest cannot be asked anything before the menu — a guest who is asked
// anything at all closes the app and orders by phone.
object Brand {

    /** `https://navvat.keel.uz` — no trailing slash. */
    val serverUrl: String = BuildConfig.SERVER_URL.trimEnd('/')

    /** `.../api/v1` — the one place the version prefix is spelled. */
    val apiBase: String = "$serverUrl/api/v1"

    /** Where dish photographs are served from. ⚠️ Its own base rather than the
     *  API's: uploads are static files off the same host but outside `/api/v1`,
     *  and a path assembled from the API base gives every image a 404 that
     *  Coil reports as a blank card. */
    val uploadsBase: String = "$serverUrl/uploads"

    /** The restaurant's own accent, from `restaurant.theme.brand`.
     *
     *  ⚠️ **Parsed defensively, because a bad colour must not be a crash.** The
     *  value is typed by an owner into a panel field; `#fff`, `E2590D` with no
     *  hash and an empty string all reach here eventually. An unparseable value
     *  falls back to Keel's orange, which is wrong-looking and running — the
     *  opposite trade from the server address, where wrong-and-running is the
     *  dangerous outcome. */
    val accent: Color = parseColor(BuildConfig.BRAND_ACCENT) ?: Color(0xFFE2590D)
}

/** `#RRGGBB`, `#AARRGGBB` or `#RGB` to a colour, or null when it is none of
 *  those. */
internal fun parseColor(raw: String): Color? {
    val hex = raw.trim().removePrefix("#")
    val full = when (hex.length) {
        3 -> hex.map { "$it$it" }.joinToString("")
        6 -> hex
        8 -> hex
        else -> return null
    }
    val value = full.toULongOrNull(16) ?: return null
    // ⚠️ Six digits carry no alpha, and a colour with alpha 0 is an invisible
    // app rather than a wrong-coloured one.
    return if (full.length == 6) Color(0xFF000000UL.or(value).toLong()) else Color(value.toLong())
}
