package uz.keel.waiter.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.runtime.staticCompositionLocalOf
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
    content: @Composable () -> Unit,
) {
    val dark = when (choice) {
        ThemeChoice.System -> isSystemInDarkTheme()
        ThemeChoice.Light -> false
        ThemeChoice.Dark -> true
    }
    val colors = if (dark) DarkColors else LightColors

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
