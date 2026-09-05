package uz.keel.design

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.spring
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.DeleteOutline
import androidx.compose.material.icons.rounded.Remove
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp

// Minus, a number, plus.
//
// ⚠️ **One component, because there are two of them and they must agree.** The
// count beside a dish in the menu and the quantity on the check are the same
// fact seen twice; built separately they drift in size and in behaviour, and a
// waiter learns one and is surprised by the other.

@Composable
fun GlassStepper(
    value: Int,
    compact: Boolean = false,
    enabled: Boolean = true,
    /** Whether minus at one removes the dish rather than refusing. */
    removeAtZero: Boolean = true,
    onMinus: () -> Unit,
    onPlus: () -> Unit,
) {
    val c = KeelTheme.colors
    val atFloor = value <= 1 && !removeAtZero
    val removing = removeAtZero && value <= 1
    val btn = if (compact) 30.dp else 38.dp

    Row(
        Modifier
            .glass(c, CircleShape, strong = true)
            .padding(3.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(if (compact) 2.dp else 4.dp),
    ) {
        Box(
            Modifier.size(btn).clip(CircleShape)
                .clickable(enabled = enabled && !atFloor, onClick = onMinus),
            contentAlignment = Alignment.Center,
        ) {
            // ⚠️ The bin rather than a minus at one, when minus means removal. A
            // minus that deletes is one glyph doing two things, and the second
            // cannot be undone from this screen.
            Icon(
                if (removing) Icons.Rounded.DeleteOutline else Icons.Rounded.Remove,
                contentDescription = null,
                tint = when {
                    atFloor -> c.muted
                    removing -> c.danger
                    else -> c.ink
                },
                modifier = Modifier.size(if (compact) 15.dp else 18.dp),
            )
        }
        // ⚠️ The number slides rather than swapping: this is the only thing on
        // screen that tells a waiter their tap landed, and a silent repaint is
        // exactly what "I pressed it and nothing happened" means.
        AnimatedContent(
            targetState = value,
            transitionSpec = {
                val up = targetState > initialState
                val spec = spring<IntOffset>(dampingRatio = Spring.DampingRatioMediumBouncy)
                (slideInVertically(spec) { h -> if (up) h else -h })
                    .togetherWith(slideOutVertically(spec) { h -> if (up) -h else h })
            },
            label = "qty",
        ) { n ->
            Text(
                "$n",
                style = MaterialTheme.typography.bodyMedium.merge(MoneyStyle),
                color = c.ink,
                textAlign = TextAlign.Center,
                modifier = Modifier.widthIn(min = if (compact) 18.dp else 24.dp),
            )
        }
        Box(
            Modifier.size(btn).clip(CircleShape).background(c.accentSoft)
                .clickable(enabled = enabled && value < 99, onClick = onPlus),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                Icons.Rounded.Add, contentDescription = null, tint = c.accent,
                modifier = Modifier.size(if (compact) 15.dp else 18.dp),
            )
        }
    }
}
