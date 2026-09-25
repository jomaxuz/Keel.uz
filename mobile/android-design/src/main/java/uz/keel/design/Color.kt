package uz.keel.design

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
    /** Where something is typed: a sunken well, not another pane of glass.
     *
     *  ⚠️ **Its own role because fields sit on glass.** A field drawn as glass
     *  on a glass card is white on white in the light scheme — the form was
     *  there and nobody could see where to tap. A field is a hole in the
     *  surface, so it goes a shade *darker* than what holds it, with an edge
     *  that is ink rather than light. */
    val field: Color,
    val fieldBorder: Color,
    val ink: Color,
    val inkSoft: Color,
    val muted: Color,
    val line: Color,
    val accent: Color,
    val accentSoft: Color,
    val onAccent: Color,
    val danger: Color,
    /** Something to look at, not something wrong.
     *
     *  ⚠️ **Its own role, because amber and red mean different things and this
     *  product uses both.** A cancelled order is worth an owner's eye; a till
     *  shortfall is worth their evening. `Notice` used to hard-code this hue
     *  inline, which is how the two drift apart. */
    val warn: Color,
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
    // ⚠️ **Denser than the dark scheme's, and edged in ink, not in light.**
    // At 72% white over cream a panel was the page colour again, and its white
    // hairline was white on white — every card, chip and button in the light
    // scheme had no edge at all, which is what "the light theme looks broken"
    // was. On a pale ground the edge has to be darker than both sides of it.
    glass = Color(0xD9FFFFFF),
    glassStrong = Color(0xF5FFFFFF),
    glassBorder = Color(0x262A2521),
    glassHighlight = Color(0x99FFFFFF),
    field = Color(0xFFF0EBE4),
    fieldBorder = Color(0x402A2521),
    ink = Color(0xFF2A2521),
    inkSoft = Color(0xFF57504A),
    // Darkened from #8A8178: that sat near 3.4:1 on the cream ground and lower
    // on glass over the warm aura — hints, prices and every "not sent" label
    // read as a grey smudge in daylight, the light-scheme twin of the dark fix.
    muted = Color(0xFF6F675F),
    line = Color(0x242A2521),
    accent = KeelOrange,
    accentSoft = Color(0x1FE2590D),
    onAccent = Color.White,
    danger = Color(0xFFC0392B),
    warn = Color(0xFFB45309),
    ready = Color(0xFF0F8A5F),
    navy = KeelNavy,
)

// ⚠️ **White over black is grey, and that was the whole mistake.** The first
// dark scheme was too dim, so the fix raised the white overlay — and a panel of
// 26% white on a near-black ground is a **light grey slab**. It was more visible
// and much worse: cold, flat, and nothing to do with a warm dining room or with
// this brand. Contrast was never the thing that was wrong; the *colour* was.
//
// So dark glass is not white with alpha. It is a **warm dark surface**, nearly
// opaque, sitting on a darker ground — which is what a real material does at
// night: it does not lighten toward white, it stays the colour of the room and
// separates by being a shade nearer the light. The separation comes from the
// hairline and the ground beneath, not from washing the panel out.
val DarkColors = KeelColors(
    dark = true,
    // Warm near-black. Not pure: an OLED smears while scrolling, and the panels
    // need something to be a shade lighter than.
    bg = Color(0xFF141110),
    auraWarm = Color(0x4DE2590D),
    auraCool = Color(0x2E3B82F6),
    // ⚠️ A colour, not a white veil. #1F1B18 is the same warm family as the
    // ground, one step up — which reads as a raised panel rather than as fog.
    glass = Color(0xF01F1B18),
    glassStrong = Color(0xFA262119),
    // ⚠️ The edge does the separating in the dark, because a black shadow on a
    // black ground is nothing at all. Warm rather than pure white, or the
    // outline turns blue against the browns it borders.
    glassBorder = Color(0x24FFE7D2),
    // Barely there. At any real strength this becomes the grey slab again — the
    // highlight is a hint of a light source, not a light.
    glassHighlight = Color(0x14FFF3E6),
    // A shade below the panel, like the light one: a well, not a lid.
    field = Color(0xFF171311),
    fieldBorder = Color(0x33FFE7D2),
    ink = Color(0xFFF4EFE8),
    inkSoft = Color(0xFFCFC7BC),
    // Lifted from the original #948A80: at that value the hints, the prices and
    // every "not sent" label sat near 3:1 — legible on a desk, gone in a dim room.
    muted = Color(0xFFA1978C),
    line = Color(0x1FFFE7D2),
    accent = KeelOrange,
    accentSoft = Color(0x3DE2590D),
    onAccent = Color.White,
    danger = Color(0xFFFF8A7A),
    // ⚠️ Lifted for the dark ground: Tailwind's amber-600 disappears into a warm
    // near-black, which is the scheme this is read in.
    warn = Color(0xFFFBBF24),
    ready = Color(0xFF4ADE80),
    navy = KeelNavy,
)

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
