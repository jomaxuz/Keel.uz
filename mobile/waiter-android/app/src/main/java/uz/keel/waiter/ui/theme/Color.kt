package uz.keel.waiter.ui.theme

import androidx.compose.runtime.Immutable
import androidx.compose.ui.graphics.Color

// Colours, in two schemes.
//
// ⚠️ **Named by role, not by shade** — the rule the Expo app already settled.
// `surface` and `ink` mean the same thing in both schemes and different things
// to the eye, which is the only way one set of screens is written once and read
// correctly in both.
//
// ⚠️ **The brand orange does not flip.** It is the same colour on the till, on
// the site and on the paper receipt. A phone that lightened it in the dark
// would be the one Keel that is a different orange, and a waiter comparing a
// screen against a printed check would see two products.
val KeelOrange = Color(0xFFE2590D)
val KeelOrangeLift = Color(0xFFFF7A2F) // only ever a gradient's top end
val KeelNavy = Color(0xFF000B1C)        // the icon's ground, the app's own identity

@Immutable
data class KeelColors(
    val dark: Boolean,
    /** The page behind the glass. Never transparent — glass over nothing is grey. */
    val bg: Color,
    /** The two blobs the background gradient is built from. Glass has to have
     *  something to refract, and a flat backdrop makes every panel look like a
     *  grey rectangle with a border. */
    val auraWarm: Color,
    val auraCool: Color,
    /** The glass itself: a translucent fill, its hairline, and the highlight
     *  that runs along the top edge. */
    val glass: Color,
    val glassStrong: Color,
    val glassBorder: Color,
    val glassHighlight: Color,
    val ink: Color,
    val inkSoft: Color,
    val muted: Color,
    val line: Color,
    val accent: Color,
    val accentSoft: Color,
    val onAccent: Color,
    val danger: Color,
    val ready: Color,
    val navy: Color,
)

// ⚠️ Not pure white and not pure black. On OLED a black backdrop smears while
// scrolling, and white text on it glares in a dim dining room — so the dark
// scheme is a warm near-black with off-white ink, as the Expo app measured.
val LightColors = KeelColors(
    dark = false,
    bg = Color(0xFFF7F3EE),
    auraWarm = Color(0x33E2590D),
    auraCool = Color(0x1F2563EB),
    glass = Color(0xB8FFFFFF),
    glassStrong = Color(0xE0FFFFFF),
    glassBorder = Color(0x59FFFFFF),
    glassHighlight = Color(0x99FFFFFF),
    ink = Color(0xFF2A2521),
    inkSoft = Color(0xFF57504A),
    muted = Color(0xFF8A8178),
    line = Color(0x1A2A2521),
    accent = KeelOrange,
    accentSoft = Color(0x1FE2590D),
    onAccent = Color.White,
    danger = Color(0xFFC0392B),
    ready = Color(0xFF0F8A5F),
    navy = KeelNavy,
)

val DarkColors = KeelColors(
    dark = true,
    bg = Color(0xFF14110E),
    auraWarm = Color(0x4DE2590D),
    auraCool = Color(0x333B82F6),
    glass = Color(0x2EFFFFFF),
    glassStrong = Color(0x47FFFFFF),
    glassBorder = Color(0x2EFFFFFF),
    glassHighlight = Color(0x4DFFFFFF),
    ink = Color(0xFFF2ECE4),
    inkSoft = Color(0xFFC9C0B6),
    muted = Color(0xFF948A80),
    line = Color(0x1FFFFFFF),
    accent = KeelOrange,
    accentSoft = Color(0x33E2590D),
    onAccent = Color.White,
    danger = Color(0xFFE56A5C),
    ready = Color(0xFF34D399),
    navy = KeelNavy,
)

/** The eight shift statuses, as plain colours.
 *
 *  ⚠️ **A second expression of one vocabulary** — `lib/attendance.ts`
 *  → `STATUS_COLOR`. A day that is amber in the panel must not be green in
 *  somebody's hand, so these are Tailwind's own 500-level hues, copied
 *  deliberately rather than approximated. */
val StatusColor: Map<String, Color> = mapOf(
    "ok" to Color(0xFF22C55E),
    "over" to Color(0xFF0EA5E9),
    "under" to Color(0xFFF59E0B),
    "absent" to Color(0xFFEF4444),
    "extra" to Color(0xFF8B5CF6),
    "open" to KeelOrange,
    "off" to Color(0xFF9CA3AF),
    "upcoming" to Color(0xFF9CA3AF),
)
