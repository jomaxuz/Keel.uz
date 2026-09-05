package uz.keel.waiter.ui.components

import androidx.compose.animation.animateColorAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.LocalTextStyle
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import uz.keel.waiter.ui.theme.KeelTheme
import uz.keel.waiter.ui.theme.MoneyStyle
import uz.keel.waiter.ui.theme.glass
import uz.keel.waiter.ui.theme.keelGradient
import uz.keel.waiter.ui.theme.softShadow

// The pieces every screen is built from.

/** The one control on a screen that acts.
 *
 *  ⚠️ **One per screen.** Two orange gradients and neither is the primary action
 *  any more — which on the check screen is "send to the kitchen", and a waiter
 *  who has to look for it is a waiter who taps the wrong thing. */
@Composable
fun PrimaryButton(
    label: String,
    modifier: Modifier = Modifier,
    icon: ImageVector? = null,
    enabled: Boolean = true,
    busy: Boolean = false,
    onClick: () -> Unit,
) {
    val c = KeelTheme.colors
    Row(
        modifier
            .fillMaxWidth()
            .height(54.dp)
            .softShadow(RoundedCornerShape(18.dp), elevation = if (enabled) 3.dp else 0.dp, dark = c.dark)
            .clip(RoundedCornerShape(18.dp))
            .background(keelGradient())
            // ⚠️ Dimmed rather than greyed: a disabled control that changes
            // colour entirely reads as a different button, and this one is the
            // same button waiting for something.
            .background(if (enabled) Color.Transparent else c.bg.copy(alpha = 0.45f))
            .clickable(enabled = enabled && !busy, onClick = onClick),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (busy) {
            CircularProgressIndicator(Modifier.size(20.dp), color = c.onAccent, strokeWidth = 2.dp)
        } else {
            if (icon != null) {
                Icon(icon, null, tint = c.onAccent, modifier = Modifier.size(19.dp))
                Box(Modifier.size(8.dp))
            }
            Text(label, color = c.onAccent, style = MaterialTheme.typography.titleMedium)
        }
    }
}

/** A quiet action: the second thing on a sheet, or a way out. */
@Composable
fun GhostButton(
    label: String,
    modifier: Modifier = Modifier,
    icon: ImageVector? = null,
    tint: Color? = null,
    enabled: Boolean = true,
    onClick: () -> Unit,
) {
    val c = KeelTheme.colors
    val ink = tint ?: c.ink
    Row(
        modifier
            .height(54.dp)
            .glass(c, RoundedCornerShape(18.dp))
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = 18.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp, Alignment.CenterHorizontally),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (icon != null) Icon(icon, null, tint = ink, modifier = Modifier.size(18.dp))
        Text(label, color = ink, style = MaterialTheme.typography.bodyLarge)
    }
}

/** A pill: a category, a zone, a filter, a tab inside a screen. */
@Composable
fun Chip(
    label: String,
    on: Boolean,
    modifier: Modifier = Modifier,
    icon: ImageVector? = null,
    onClick: () -> Unit,
) {
    val c = KeelTheme.colors
    val fg by animateColorAsState(if (on) c.accent else c.muted, label = "chipFg")
    Row(
        modifier
            .clip(CircleShape)
            .then(if (on) Modifier.background(c.accentSoft) else Modifier.glass(c, CircleShape))
            .border(1.dp, if (on) c.accent.copy(alpha = 0.55f) else c.glassBorder, CircleShape)
            .clickable(onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 9.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        if (icon != null) Icon(icon, null, tint = fg, modifier = Modifier.size(14.dp))
        Text(
            label,
            color = if (on) c.ink else c.muted,
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

/** The bar across the top of a screen. */
@Composable
fun ScreenHeader(
    title: String,
    subtitle: String? = null,
    modifier: Modifier = Modifier,
    leading: @Composable (() -> Unit)? = null,
    trailing: @Composable (() -> Unit)? = null,
) {
    val c = KeelTheme.colors
    Row(
        modifier
            .fillMaxWidth()
            .statusBarsPadding()
            .padding(horizontal = 16.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        leading?.invoke()
        androidx.compose.foundation.layout.Column(Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.titleLarge, color = c.ink)
            if (subtitle != null) {
                Text(subtitle, style = MaterialTheme.typography.labelMedium, color = c.muted)
            }
        }
        trailing?.invoke()
    }
}

/** A round glass button for a header or a corner. ⚠️ 40dp, which is the
 *  smallest a thumb finds while walking. */
@Composable
fun GlassIconButton(
    icon: ImageVector,
    modifier: Modifier = Modifier,
    tint: Color? = null,
    onClick: () -> Unit,
) {
    val c = KeelTheme.colors
    Box(
        modifier.size(40.dp).glass(c, CircleShape).clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Icon(icon, null, tint = tint ?: c.inkSoft, modifier = Modifier.size(19.dp))
    }
}

/** A text field on glass. */
@Composable
fun GlassField(
    value: String,
    onValueChange: (String) -> Unit,
    placeholder: String,
    modifier: Modifier = Modifier,
    keyboardOptions: KeyboardOptions = KeyboardOptions.Default,
    visualTransformation: VisualTransformation = VisualTransformation.None,
    trailing: @Composable (() -> Unit)? = null,
) {
    val c = KeelTheme.colors
    TextField(
        value = value,
        onValueChange = onValueChange,
        modifier = modifier.fillMaxWidth().glass(c, RoundedCornerShape(16.dp)),
        placeholder = { Text(placeholder, color = c.muted) },
        singleLine = true,
        keyboardOptions = keyboardOptions,
        visualTransformation = visualTransformation,
        trailingIcon = trailing,
        textStyle = LocalTextStyle.current.copy(color = c.ink, fontSize = 16.sp),
        colors = TextFieldDefaults.colors(
            focusedContainerColor = Color.Transparent,
            unfocusedContainerColor = Color.Transparent,
            disabledContainerColor = Color.Transparent,
            // ⚠️ Material's underline removed: it belongs to a filled text field
            // and cuts the glass in half.
            focusedIndicatorColor = Color.Transparent,
            unfocusedIndicatorColor = Color.Transparent,
            cursorColor = c.accent,
        ),
    )
}

/** Money, in the app's own grouping and with tabular figures. */
@Composable
fun Money(amount: Double, modifier: Modifier = Modifier, color: Color? = null, style: androidx.compose.ui.text.TextStyle? = null) {
    val c = KeelTheme.colors
    Text(
        uz.keel.waiter.data.money(amount),
        modifier = modifier,
        color = color ?: c.ink,
        style = (style ?: MaterialTheme.typography.bodyMedium).merge(MoneyStyle),
    )
}

/** Choosing the language before there is anywhere to choose it.
 *
 *  ⚠️ **The setting existed and was unreachable, which is the same as not
 *  existing.** Language lives on the settings screen — and the settings screen is
 *  behind a sign-in. So a phone handed to a Russian-speaking waiter opened in
 *  Uzbek, asked for a restaurant address in Uzbek and a password in Uzbek, and
 *  the only way to change that was to get past the very screens they could not
 *  read. Setting up a phone is exactly when somebody needs this and exactly when
 *  they could not have it.
 *
 *  ⚠️ **Codes, not flags.** A flag names a country, and the language somebody
 *  reads is not where they are — half this country reads Russian and lives here.
 *  Three short words fit where three flags would, and they are unambiguous.
 *
 *  ⚠️ Each language is written **in itself**: a list that said "Ruscha" in Uzbek
 *  is a list a Russian speaker has to decode before they can leave the language
 *  they cannot read. Same rule as the settings screen. */
@Composable
fun LangSwitch(modifier: Modifier = Modifier) {
    val c = KeelTheme.colors
    val prefs = uz.keel.waiter.LocalPrefs.current
    Row(
        modifier.glass(c, CircleShape).padding(3.dp),
        horizontalArrangement = Arrangement.spacedBy(2.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        uz.keel.waiter.i18n.Lang.entries.forEach { l ->
            val on = prefs.lang.value == l
            Box(
                Modifier
                    .clip(CircleShape)
                    .background(if (on) c.accent else Color.Transparent)
                    .clickable { prefs.setLang(l) }
                    .padding(horizontal = 11.dp, vertical = 6.dp),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    l.code.uppercase(),
                    color = if (on) c.onAccent else c.muted,
                    style = MaterialTheme.typography.labelMedium,
                )
            }
        }
    }
}
