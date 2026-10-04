package uz.keel.waiter.ui.components

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.core.spring
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import kotlinx.coroutines.delay
import uz.keel.design.KeelTheme
import uz.keel.design.MoneyStyle
import uz.keel.design.glass
import uz.keel.design.softShadow

// The waiter's own small parts: what the floor and the check are drawn from.
//
// ⚠️ **Here and not in `android-design`.** The design module is shared by six
// applications; these are shaped by one job — a thumb, walking, a tray in the
// other hand — and moving them there would restyle a courier's phone as a side
// effect of a waiter's redesign.

/** Run [block] every [periodMs] while the app is on screen, and once at once
 *  when it comes back to the front.
 *
 *  ⚠️ **Only while visible.** A loop in a plain `LaunchedEffect` keeps asking the
 *  server from a phone in a pocket — battery, data, and a hold on a table nobody
 *  is looking at. ⚠️ And immediately on return: a phone taken out of a pocket
 *  should not show a room that is a minute old for another fifteen seconds. */
@Composable
fun PollWhileVisible(periodMs: Long, block: suspend () -> Unit) {
    val owner = LocalLifecycleOwner.current
    var visible by remember(owner) {
        mutableStateOf(owner.lifecycle.currentState.isAtLeast(Lifecycle.State.STARTED))
    }
    var wasHidden by remember { mutableStateOf(false) }
    DisposableEffect(owner) {
        val obs = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_START -> visible = true
                Lifecycle.Event.ON_STOP -> visible = false
                else -> Unit
            }
        }
        owner.lifecycle.addObserver(obs)
        onDispose { owner.lifecycle.removeObserver(obs) }
    }
    val latest by rememberUpdatedState(block)
    LaunchedEffect(visible) {
        if (!visible) {
            wasHidden = true
            return@LaunchedEffect
        }
        if (wasHidden) {
            wasHidden = false
            latest()
        }
        while (true) {
            delay(periodMs)
            latest()
        }
    }
}

/** A light tick under the thumb.
 *
 *  ⚠️ **Felt, because it is not always seen.** A waiter adding dishes is looking
 *  at the guest, not at the phone; the tick is what says the tap landed without
 *  them having to look down — and not looking down is when people tap twice. */
@Composable
fun rememberTick(): () -> Unit {
    val haptic = LocalHapticFeedback.current
    return remember(haptic) { { haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove) } }
}

/** Two or three choices in one track, the chosen one lifted out of it.
 *
 *  ⚠️ **Not orange.** The one orange control on a screen is the one that acts
 *  ("send to the kitchen"); a tab switch painted the same colour competes with
 *  it, and the waiter looks for the wrong one. The chosen segment is lifted,
 *  not coloured. */
@Composable
fun Segmented(
    options: List<Pair<String, String>>,
    selected: String,
    onSelect: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val c = KeelTheme.colors
    val index = options.indexOfFirst { it.first == selected }.coerceAtLeast(0)
    val track = RoundedCornerShape(16.dp)
    val thumb = RoundedCornerShape(12.dp)
    BoxWithConstraints(
        modifier
            .fillMaxWidth()
            .height(46.dp)
            .clip(track)
            .background(c.field, track)
            .border(1.dp, c.glassBorder, track)
            .padding(4.dp),
    ) {
        val cell = maxWidth / options.size
        val x by animateDpAsState(
            cell * index,
            spring(dampingRatio = Spring.DampingRatioLowBouncy, stiffness = Spring.StiffnessMediumLow),
            label = "segment",
        )
        Box(
            Modifier
                .offset(x = x)
                .width(cell)
                .fillMaxHeight()
                .softShadow(thumb, elevation = 2.dp, dark = c.dark)
                .clip(thumb)
                .background(c.glassStrong, thumb)
                .border(1.dp, c.glassBorder, thumb),
        )
        Row(Modifier.fillMaxSize()) {
            options.forEachIndexed { i, (key, label) ->
                val on = i == index
                val ink by animateColorAsState(if (on) c.ink else c.muted, label = "segmentInk")
                Box(
                    Modifier
                        .weight(1f)
                        .fillMaxHeight()
                        .clip(thumb)
                        .clickable { onSelect(key) },
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        label,
                        color = ink,
                        style = MaterialTheme.typography.bodyMedium.copy(
                            fontWeight = if (on) FontWeight.SemiBold else FontWeight.Medium,
                        ),
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        }
    }
}

/** A small coloured label: "2 tayyor", "3 yangi".
 *
 *  ⚠️ **Words and a number, not a dot.** The floor used to mark these with two
 *  nine-pixel dots in opposite corners, and nobody could say which was which
 *  without being told — the difference between "walk to the pass" and "you have
 *  not sent this" is not something to encode in a corner. */
@Composable
fun Pill(
    text: String,
    tone: Color,
    modifier: Modifier = Modifier,
    solid: Boolean = false,
    icon: ImageVector? = null,
) {
    val c = KeelTheme.colors
    val ink = if (solid) Color.White else tone
    Row(
        modifier
            .clip(CircleShape)
            .background(if (solid) tone else tone.copy(alpha = if (c.dark) 0.22f else 0.13f))
            .padding(horizontal = 7.dp, vertical = 3.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(3.dp),
    ) {
        if (icon != null) Icon(icon, null, tint = ink, modifier = Modifier.size(11.dp))
        Text(
            text,
            color = ink,
            style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
            maxLines = 1,
        )
    }
}

/** A filter with its own count: "Band 5".
 *
 *  ⚠️ **The count is the reason to press it.** A filter that says only "Ready"
 *  has to be pressed to find out there is nothing ready; one that says "Ready 2"
 *  answers the question from across the room. */
@Composable
fun CountChip(
    label: String,
    count: Int,
    on: Boolean,
    tone: Color,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    val c = KeelTheme.colors
    Row(
        modifier
            .clip(CircleShape)
            .then(
                if (on) Modifier.background(tone.copy(alpha = if (c.dark) 0.24f else 0.14f), CircleShape)
                else Modifier.glass(c, CircleShape),
            )
            .border(1.dp, if (on) tone.copy(alpha = 0.7f) else c.glassBorder, CircleShape)
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Box(Modifier.size(8.dp).background(tone, CircleShape))
        Text(label, color = if (on) c.ink else c.inkSoft, style = MaterialTheme.typography.bodyMedium)
        Text(
            "$count",
            color = if (on) c.ink else c.muted,
            style = MaterialTheme.typography.bodyMedium.merge(MoneyStyle),
        )
    }
}

/** Something the waiter should read, in the flow of the screen.
 *
 *  ⚠️ **Tinted, not red text.** A refusal used to be a line of red under the
 *  header, the size of everything else, and a "saved, will send" was drawn in
 *  the same red — so the good news read as a failure and the dish was tapped
 *  again. The tone says which it is; the words say what to do. */
@Composable
fun InlineBanner(
    text: String,
    tone: Color,
    icon: ImageVector,
    modifier: Modifier = Modifier,
    onClose: (() -> Unit)? = null,
) {
    val c = KeelTheme.colors
    val shape = RoundedCornerShape(14.dp)
    Row(
        modifier
            .fillMaxWidth()
            .clip(shape)
            .background(c.glassStrong, shape)
            .background(tone.copy(alpha = if (c.dark) 0.18f else 0.10f), shape)
            .border(1.dp, tone.copy(alpha = 0.35f), shape)
            .padding(start = 12.dp, end = 6.dp, top = 8.dp, bottom = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Icon(icon, null, tint = tone, modifier = Modifier.size(18.dp))
        Text(text, Modifier.weight(1f), color = c.ink, style = MaterialTheme.typography.bodyMedium)
        if (onClose != null) {
            Box(
                Modifier.size(32.dp).clip(CircleShape).clickable(onClick = onClose),
                contentAlignment = Alignment.Center,
            ) {
                Icon(Icons.Rounded.Close, null, tint = c.muted, modifier = Modifier.size(16.dp))
            }
        } else {
            Box(Modifier.size(6.dp))
        }
    }
}

/** The label over one group of lines on a check. */
@Composable
fun SectionHeader(text: String, tone: Color, count: Int, modifier: Modifier = Modifier) {
    val c = KeelTheme.colors
    Row(
        modifier.fillMaxWidth().padding(top = 8.dp, bottom = 2.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Box(Modifier.size(8.dp).background(tone, CircleShape))
        Text(
            text,
            color = tone,
            style = MaterialTheme.typography.labelMedium.copy(fontWeight = FontWeight.SemiBold),
        )
        Text("$count", color = c.muted, style = MaterialTheme.typography.labelMedium.merge(MoneyStyle))
        Box(Modifier.weight(1f).height(1.dp).background(c.line))
    }
}

/** A shape where content is about to be.
 *
 *  ⚠️ **Static, not shimmering.** A shimmer is an animation on every placeholder
 *  of a grid, redrawn every frame on the phones that are slowest to load in the
 *  first place. The shapes alone say "this is coming, and it will look like
 *  this", which is all a spinner in the middle of an empty screen did not. */
fun Modifier.placeholder(colors: uz.keel.design.KeelColors, shape: androidx.compose.ui.graphics.Shape): Modifier =
    this
        .clip(shape)
        .background(colors.glass.copy(alpha = if (colors.dark) 0.55f else 0.6f), shape)
        .border(1.dp, colors.glassBorder, shape)
