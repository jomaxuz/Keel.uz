package uz.keel.app.ui

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsFocusedAsState
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Badge
import androidx.compose.material.icons.rounded.DeliveryDining
import androidx.compose.material.icons.rounded.Insights
import androidx.compose.material.icons.rounded.TableRestaurant
import androidx.compose.material3.Icon
import androidx.compose.material3.LocalTextStyle
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.draw.scale
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import uz.keel.app.R
import uz.keel.app.data.Account
import uz.keel.app.data.Kind
import uz.keel.app.data.Workspace
import uz.keel.app.i18n.UZ
import uz.keel.app.i18n.Words
import uz.keel.design.KeelNavy
import uz.keel.design.KeelOrange
import uz.keel.design.KeelOrangeLift
import uz.keel.design.KeelTheme

// The shell's own pieces, on the design module's tokens.
//
// ⚠️ **Nothing here re-decides a colour.** The orange, the glass and the field
// well are `android-design`'s, so the sign-in, the switcher and the four roles
// behind them read as one product rather than a launcher glued to four apps.

val LocalShellWords = compositionLocalOf { UZ }

val w: Words
    @Composable @ReadOnlyComposable get() = LocalShellWords.current

/** Keel's mark on its navy tile — the launcher icon's own foreground, so the
 *  home screen, the splash and this screen show one thing.
 *
 *  @param breathe a slow glow behind the tile, for the one screen somebody
 *  waits on. ⚠️ Slow and faint: a pulse anybody notices is a loading spinner
 *  pretending not to be one. */
@Composable
fun KeelMark(size: Dp = 76.dp, breathe: Boolean = false) {
    val glow = if (breathe) {
        val t = rememberInfiniteTransition(label = "mark")
        t.animateFloat(
            initialValue = 0.85f, targetValue = 1.12f,
            animationSpec = infiniteRepeatable(tween(2600, easing = FastOutSlowInEasing), RepeatMode.Reverse),
            label = "glow",
        ).value
    } else 1f
    Box(Modifier.size(size * 1.6f), contentAlignment = Alignment.Center) {
        Box(
            Modifier
                .size(size * 1.5f)
                .scale(glow)
                .drawBehind {
                    drawCircle(
                        Brush.radialGradient(
                            listOf(KeelOrange.copy(alpha = 0.28f), Color.Transparent),
                            center = Offset(this.size.width / 2, this.size.height / 2),
                            radius = this.size.minDimension / 2,
                        ),
                    )
                },
        )
        Box(
            Modifier
                .size(size)
                .clip(RoundedCornerShape(size * 0.3f))
                .background(KeelNavy)
                .border(1.dp, Color.White.copy(alpha = 0.10f), RoundedCornerShape(size * 0.3f)),
            contentAlignment = Alignment.Center,
        ) {
            Image(
                painterResource(R.drawable.ic_launcher_foreground),
                contentDescription = null,
                modifier = Modifier.size(size * 0.9f),
            )
        }
    }
}

/** A field with its meaning drawn in front of it.
 *
 *  ⚠️ **The icon is the label.** Three fields with placeholders alone become
 *  three identical wells the moment the first is filled — and "which one was
 *  the password?" is asked by somebody with gloves on. The design module's
 *  `GlassField` has no leading slot, so this is the same well with one. */
@Composable
fun KeelInput(
    value: String,
    onValueChange: (String) -> Unit,
    placeholder: String,
    icon: ImageVector,
    modifier: Modifier = Modifier,
    keyboardOptions: KeyboardOptions = KeyboardOptions.Default,
    keyboardActions: KeyboardActions = KeyboardActions.Default,
    visualTransformation: VisualTransformation = VisualTransformation.None,
    suffix: String? = null,
    enabled: Boolean = true,
    trailing: @Composable (() -> Unit)? = null,
) {
    val c = KeelTheme.colors
    val interaction = remember { MutableInteractionSource() }
    val focused by interaction.collectIsFocusedAsState()
    val edge by animateColorAsState(if (focused) c.accent else c.fieldBorder, label = "edge")
    val iconTint by animateColorAsState(if (focused) c.accent else c.muted, label = "icon")
    val shape = RoundedCornerShape(18.dp)
    TextField(
        value = value,
        onValueChange = onValueChange,
        enabled = enabled,
        modifier = modifier
            .fillMaxWidth()
            .clip(shape)
            .background(c.field, shape)
            .border(if (focused) 1.5.dp else 1.dp, edge, shape),
        placeholder = { Text(placeholder, color = c.muted) },
        leadingIcon = { Icon(icon, null, tint = iconTint, modifier = Modifier.size(20.dp)) },
        suffix = suffix?.let { s -> { Text(s, color = c.muted, fontSize = 15.sp) } },
        trailingIcon = trailing,
        singleLine = true,
        keyboardOptions = keyboardOptions,
        keyboardActions = keyboardActions,
        visualTransformation = visualTransformation,
        interactionSource = interaction,
        textStyle = LocalTextStyle.current.copy(color = c.ink, fontSize = 16.sp),
        colors = TextFieldDefaults.colors(
            focusedContainerColor = Color.Transparent,
            unfocusedContainerColor = Color.Transparent,
            disabledContainerColor = Color.Transparent,
            focusedIndicatorColor = Color.Transparent,
            unfocusedIndicatorColor = Color.Transparent,
            disabledIndicatorColor = Color.Transparent,
            cursorColor = c.accent,
            focusedTextColor = c.ink,
            unfocusedTextColor = c.ink,
            disabledTextColor = c.inkSoft,
        ),
    )
}

/** Initials on the brand gradient — who an account is, at a glance. */
@Composable
fun Avatar(name: String, size: Dp = 44.dp, modifier: Modifier = Modifier) {
    val initials = name.trim().split(Regex("\\s+")).filter { it.isNotEmpty() }
        .take(2).joinToString("") { it.first().uppercase() }.ifEmpty { "K" }
    Box(
        modifier
            .size(size)
            .clip(CircleShape)
            .background(Brush.linearGradient(listOf(KeelOrangeLift, KeelOrange))),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            initials,
            color = Color.White,
            fontWeight = FontWeight.SemiBold,
            fontSize = (size.value * 0.38f).sp,
        )
    }
}

fun Workspace.icon(): ImageVector = when (this) {
    Workspace.Owner -> Icons.Rounded.Insights
    Workspace.Waiter -> Icons.Rounded.TableRestaurant
    Workspace.Courier -> Icons.Rounded.DeliveryDining
    Workspace.Team -> Icons.Rounded.Badge
}

fun Workspace.words(w: Words): Words.Space = when (this) {
    Workspace.Owner -> w.owner
    Workspace.Waiter -> w.waiter
    Workspace.Courier -> w.courier
    Workspace.Team -> w.team
}

/** What an account is, in one word: "Ega", "Menejer", the staff member's own
 *  job title when the office wrote one, "Kuryer". */
fun Account.label(w: Words): String = when (kindOf) {
    Kind.Admin -> if (role == "manager") w.kinds.manager else w.kinds.owner
    Kind.Courier -> w.kinds.courier
    Kind.Staff -> roleName?.takeIf { it.isNotBlank() } ?: position.ifBlank { w.kinds.staff }
}

fun Kind.label(w: Words): String = when (this) {
    Kind.Admin -> w.kinds.owner
    Kind.Staff -> w.kinds.staff
    Kind.Courier -> w.kinds.courier
}
