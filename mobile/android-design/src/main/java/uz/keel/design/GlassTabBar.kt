package uz.keel.design

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.spring
import androidx.compose.foundation.background
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp

/** One destination in the bar. */
data class TabItem(val key: String, val icon: ImageVector, val label: String)

/** The floating glass tab bar.
 *
 *  ⚠️ **Labelled, not icons alone.** Three glyphs with no words is a guess every
 *  new waiter makes on their first evening — the lesson the Expo bar already
 *  carries, and a prettier bar is not a reason to unlearn it.
 *
 *  ⚠️ **`navigationBarsPadding`, always.** These screens draw edge to edge, so a
 *  bar placed at the bottom of the layout sits *behind* Android's Back and Home
 *  — and the tap meant for "Floor" goes back a screen. Reported from real
 *  phones on the support chat, on the Expo build, before this one existed.
 *
 *  ⚠️ **The selected pill moves; the icons do not jump.** The orange capsule is
 *  animated with a spring so the eye follows it between destinations. A bar that
 *  simply repainted would read as two different bars a frame apart. */
@Composable
fun GlassTabBar(
    items: List<TabItem>,
    selected: String,
    onSelect: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val c = KeelTheme.colors
    Row(
        modifier
            .fillMaxWidth()
            .navigationBarsPadding()
            .padding(horizontal = 16.dp, vertical = 10.dp)
            .softShadow(RoundedCornerShape(28.dp), elevation = 6.dp, dark = c.dark)
            .glass(c, RoundedCornerShape(28.dp), strong = true)
            .padding(6.dp),
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        // ⚠️ **Past four destinations the words stop fitting, and a clipped
        // word is worse than none.** Six tabs on a 320dp phone leave about 50dp
        // each; "Sozlamalar" needs more than that at any legible size, so it
        // arrived cut in half. The selected one keeps its label — that is the
        // one somebody is reading — and the rest are their icons, which is how
        // the pill was already drawing attention anyway.
        val labelAll = items.size <= 4
        items.forEach { item ->
            TabCell(
                item = item,
                on = item.key == selected,
                showLabel = labelAll || item.key == selected,
                modifier = Modifier.weight(1f),
            ) { onSelect(item.key) }
        }
    }
}

@Composable
private fun TabCell(
    item: TabItem,
    on: Boolean,
    showLabel: Boolean,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    val c = KeelTheme.colors
    val spring = spring<Float>(dampingRatio = Spring.DampingRatioMediumBouncy, stiffness = Spring.StiffnessLow)
    val glow by animateFloatAsState(if (on) 1f else 0f, spring, label = "tabGlow")
    val lift by animateDpAsState(if (on) 2.dp else 0.dp, label = "tabLift")
    val tint by animateColorAsState(if (on) c.onAccent else c.muted, label = "tabTint")
    val interaction = remember { MutableInteractionSource() }

    Box(
        modifier
            .height(56.dp)
            .clip(RoundedCornerShape(22.dp))
            .selectable(
                selected = on,
                interactionSource = interaction,
                indication = null,
                role = Role.Tab,
                onClick = onClick,
            ),
        contentAlignment = Alignment.Center,
    ) {
        // The pill. Drawn unconditionally and faded by `glow` rather than
        // switched on — ⚠️ a branch here would pop the orange in on the frame
        // the tab changes, which is the one thing an animation is for. Scaled
        // rather than resized, so the label never reflows while it grows.
        Box(
            Modifier
                .matchParentSize()
                .padding(2.dp)
                .graphicsLayer {
                    alpha = glow
                    scaleX = 0.85f + 0.15f * glow
                    scaleY = 0.85f + 0.15f * glow
                }
                .clip(RoundedCornerShape(20.dp))
                .background(keelGradient()),
        )
        Column(
            Modifier.padding(bottom = lift),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(2.dp),
        ) {
            Icon(
                item.icon,
                // ⚠️ The label is the description when it is not drawn — an
                // icon-only tab is unreachable to a screen reader otherwise.
                contentDescription = if (showLabel) null else item.label,
                tint = tint,
                modifier = Modifier.size(21.dp),
            )
            if (showLabel) {
                Text(
                    item.label,
                    color = tint,
                    style = androidx.compose.material3.MaterialTheme.typography.labelSmall,
                    maxLines = 1,
                    overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                )
            }
        }
    }
}
