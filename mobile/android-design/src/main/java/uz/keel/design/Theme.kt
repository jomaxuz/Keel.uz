package uz.keel.design

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.luminance
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.LineHeightStyle
import androidx.compose.ui.unit.sp

// ⚠️ **Material3's `ColorScheme` is filled in but is not where the app reads
// its colours from.** Material's roles (primary, surfaceVariant, outline) do not
// have a name for "the fill of a pane of glass" or "the highlight along its top
// edge", and forcing them into `surfaceContainerHigh` would leave the next
// person guessing which of eleven surface roles means glass. `KeelColors` is
// the app's vocabulary; Material's is filled in only so its own components
// (ripples, text fields, the status bar) do not arrive in purple.

val LocalKeelColors = staticCompositionLocalOf { LightColors }

object KeelTheme {
    val colors: KeelColors
        @Composable @ReadOnlyComposable get() = LocalKeelColors.current
}

/** What the person chose, which is a different question from what the phone is
 *  set to. "system" is the default and the honest one: most people set their
 *  phone once and expect everything to follow it. */
enum class ThemeChoice { System, Light, Dark }

@Composable
fun KeelWaiterTheme(
    choice: ThemeChoice = ThemeChoice.System,
    /** The restaurant's own accent, for the one application that wears it.
     *
     *  ⚠️ **Null is Keel's orange, and that is what the five staff applications
     *  pass.** A till, a pass screen and a courier's phone are *our* tool, and
     *  the orange on them has to match a printed receipt — a colour that
     *  followed the restaurant would make every Keel a different product to the
     *  person supporting it. The guest application is the opposite: it is the
     *  restaurant's own app, sitting on a guest's home screen under the
     *  restaurant's name, and Keel's orange in it would be a stranger's colour.
     *
     *  ⚠️ **One parameter here rather than a second theme function.** Two
     *  entry points is two places to add the next token to, and the one that
     *  gets forgotten is the one used by the app nobody on the team opens
     *  daily. */
    accent: Color? = null,
    content: @Composable () -> Unit,
) {
    val dark = when (choice) {
        ThemeChoice.System -> isSystemInDarkTheme()
        ThemeChoice.Light -> false
        ThemeChoice.Dark -> true
    }
    val colors = (if (dark) DarkColors else LightColors).branded(accent)

    // ⚠️ **Dynamic colour is deliberately off.** Material You would repaint the
    // app in whatever the waiter's wallpaper happens to be, and the orange is
    // the one thing on this screen that has to match a printed receipt.
    val material = if (dark) {
        darkColorScheme(
            primary = colors.accent, onPrimary = colors.onAccent,
            background = colors.bg, onBackground = colors.ink,
            surface = colors.bg, onSurface = colors.ink,
            error = colors.danger,
        )
    } else {
        lightColorScheme(
            primary = colors.accent, onPrimary = colors.onAccent,
            background = colors.bg, onBackground = colors.ink,
            surface = colors.bg, onSurface = colors.ink,
            error = colors.danger,
        )
    }

    CompositionLocalProvider(LocalKeelColors provides colors) {
        MaterialTheme(colorScheme = material, typography = KeelTypography, content = content)
    }
}

/** The same scheme wearing somebody else's colour.
 *
 *  ⚠️ **Only the roles that are genuinely the accent move.** The ground, the
 *  glass and the ink stay ours: a restaurant's brand colour is one hue chosen
 *  for a signboard, and letting it repaint the surfaces would produce, on some
 *  of them, a lime-green dining room nobody chose.
 *
 *  ⚠️ **`onAccent` is computed, never assumed white.** A pale brand — mustard,
 *  cream, spring green, and there are plenty — leaves white text on a primary
 *  button invisible: the button still works, so nothing errors, and the app
 *  ships with an unreadable "Buyurtma berish". Luminance decides, once, here.
 *
 *  ⚠️ **The warm aura follows too.** It is the accent seen through the
 *  background blur; left orange under a blue brand it reads as a stain rather
 *  than as light. */
fun KeelColors.branded(accent: Color?): KeelColors {
    if (accent == null) return this
    return copy(
        accent = accent,
        accentSoft = accent.copy(alpha = if (dark) 0.24f else 0.12f),
        // 0.55 rather than 0.5: the eye reads mid-tones as darker than the
        // arithmetic does, and the failure is one-sided — grey-on-colour is
        // uncomfortable, white-on-pale is unreadable.
        onAccent = if (accent.luminance() > 0.55f) Color(0xFF1A1614) else Color.White,
        auraWarm = accent.copy(alpha = if (dark) 0.30f else 0.20f),
    )
}

// ⚠️ **Tabular figures wherever money is printed.** A total whose digits change
// width jitters as it counts up, and a guest is reading it off the phone next to
// a paper bill. `fontFeatureSettings = "tnum"` is the one line that fixes it.
val MoneyStyle = TextStyle(
    fontFeatureSettings = "tnum",
    fontWeight = FontWeight.SemiBold,
)

private val lineHeight = LineHeightStyle(
    alignment = LineHeightStyle.Alignment.Center,
    trim = LineHeightStyle.Trim.None,
)

// ⚠️ **The one number a screen is opened for gets its own size.** An owner
// checks today's takings standing up, between two other things; at body size it
// is a figure they have to look for. Tabular figures so it does not jitter while
// it counts up.
val BigNumberStyle = TextStyle(
    fontWeight = FontWeight.Bold,
    fontSize = 38.sp,
    lineHeight = 44.sp,
    fontFeatureSettings = "tnum",
)

val KeelTypography = Typography(
    headlineMedium = TextStyle(
        fontFamily = FontFamily.Default, fontWeight = FontWeight.Bold,
        fontSize = 26.sp, lineHeight = 32.sp, lineHeightStyle = lineHeight,
    ),
    titleLarge = TextStyle(
        fontWeight = FontWeight.SemiBold, fontSize = 19.sp,
        lineHeight = 24.sp, lineHeightStyle = lineHeight,
    ),
    titleMedium = TextStyle(
        fontWeight = FontWeight.SemiBold, fontSize = 16.sp,
        lineHeight = 21.sp, lineHeightStyle = lineHeight,
    ),
    bodyLarge = TextStyle(fontSize = 15.sp, lineHeight = 21.sp, lineHeightStyle = lineHeight),
    bodyMedium = TextStyle(fontSize = 14.sp, lineHeight = 19.sp, lineHeightStyle = lineHeight),
    labelMedium = TextStyle(fontSize = 12.sp, lineHeight = 15.sp, lineHeightStyle = lineHeight),
    labelSmall = TextStyle(fontSize = 11.sp, lineHeight = 14.sp, lineHeightStyle = lineHeight),
)
