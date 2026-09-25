package uz.keel.guest.ui.screens

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.withFrameMillis
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.scale
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.rotate
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import uz.keel.design.KeelTheme
import uz.keel.guest.t
import kotlin.random.Random

// The moment the food arrives.
//
// ⚠️ **The site's celebration, and the same one.** The tracking page has always
// thrown confetti and bounced a green tick when an order reached `delivered`
// (`frontend/src/components/order/DeliveredCelebration.tsx`); the app went from
// "Yo'lda" to a fifth grey-to-orange dot and said nothing. A guest who orders
// from the restaurant's own app should not get the quieter of the two.
//
// ⚠️ **Same palette as the site, not the restaurant's accent.** Confetti in one
// colour is a stain falling down the screen; six is a celebration — and the
// site's six are what the guest may already have seen there.
//
// ⚠️ **Finite, unlike the site's.** A browser tab gets closed; this screen is
// left open on a phone on the table, and a loop that never stops is a battery
// cost for a moment that is over. The pieces fall for a few seconds and stop.

private val ConfettiColors = listOf(
    Color(0xFFE11D48), Color(0xFFF59E0B), Color(0xFF10B981),
    Color(0xFF3B82F6), Color(0xFF8B5CF6), Color(0xFFEC4899),
)

private val Emerald = Color(0xFF10B981)

/** How long the pieces keep falling. */
private const val CONFETTI_MS = 7_000L

private class Piece(
    val x: Float,          // 0..1 across the screen
    val delay: Long,       // ms before it starts
    val duration: Long,    // ms for one fall
    val color: Color,
    val size: Float,       // dp
    val spin: Float,       // total degrees over one fall
    val sway: Float,       // horizontal drift, fraction of width
)

/** Confetti over the whole screen. Draws nothing once it is done. */
@Composable
fun ConfettiOverlay(modifier: Modifier = Modifier) {
    val pieces = remember {
        List(60) { i ->
            Piece(
                x = Random.nextFloat(),
                delay = Random.nextLong(0, 2_000),
                duration = Random.nextLong(2_500, 4_500),
                color = ConfettiColors[i % ConfettiColors.size],
                size = 6f + Random.nextFloat() * 6f,
                spin = 360f + Random.nextFloat() * 360f,
                sway = (Random.nextFloat() - 0.5f) * 0.12f,
            )
        }
    }
    // ⚠️ A piece in mid-air when time is up finishes its fall rather than
    // vanishing: each one gets a whole number of falls, and the overlay ends
    // when the last of them lands.
    val falls = remember { pieces.map { maxOf(1L, (CONFETTI_MS - it.delay) / it.duration) } }
    val end = remember { pieces.indices.maxOf { pieces[it].delay + falls[it] * pieces[it].duration } }
    var elapsed by remember { mutableLongStateOf(0L) }
    LaunchedEffect(Unit) {
        val start = withFrameMillis { it }
        while (elapsed < end) {
            elapsed = withFrameMillis { it } - start
        }
    }
    if (elapsed >= end) return

    Canvas(modifier.fillMaxSize()) {
        val w = size.width
        val h = size.height
        pieces.forEachIndexed { i, p ->
            val local = elapsed - p.delay
            if (local < 0 || local / p.duration >= falls[i]) return@forEachIndexed
            val f = (local % p.duration).toFloat() / p.duration
            val y = -0.1f * h + f * 1.2f * h
            val x = (p.x + p.sway * kotlin.math.sin(f * 6.28f)) * w
            val pw = p.size.dp.toPx()
            val ph = pw * 0.4f
            rotate(p.spin * f, pivot = Offset(x, y)) {
                drawRect(
                    color = p.color.copy(alpha = 1f - f * 0.6f),
                    topLeft = Offset(x - pw / 2, y - ph / 2),
                    size = Size(pw, ph),
                )
            }
        }
    }
}

/** The green tick that pops in and then bobs, with the two lines under it. */
@Composable
fun DeliveredBadge(modifier: Modifier = Modifier) {
    val c = KeelTheme.colors
    val haptics = LocalHapticFeedback.current
    val pop = remember { Animatable(0f) }
    LaunchedEffect(Unit) {
        haptics.performHapticFeedback(HapticFeedbackType.LongPress)
        pop.animateTo(1f, spring(dampingRatio = 0.45f, stiffness = Spring.StiffnessLow))
    }
    val bob by rememberInfiniteTransition(label = "bob").animateFloat(
        initialValue = 0f,
        targetValue = -8f,
        animationSpec = infiniteRepeatable(tween(1_100), RepeatMode.Reverse),
        label = "bobY",
    )

    Column(
        modifier.fillMaxWidth().padding(vertical = 12.dp).scale(pop.value),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Box(
            Modifier
                .graphicsLayer { translationY = bob.dp.toPx() }
                .size(96.dp)
                .shadow(14.dp, CircleShape, ambientColor = Emerald, spotColor = Emerald)
                .background(Emerald, CircleShape),
            contentAlignment = Alignment.Center,
        ) {
            Icon(Icons.Rounded.Check, null, tint = Color.White, modifier = Modifier.size(56.dp))
        }
        Text(
            t.order.deliveredTitle,
            Modifier.padding(top = 10.dp),
            style = MaterialTheme.typography.headlineMedium,
            color = if (c.dark) Color(0xFF34D399) else Color(0xFF059669),
            textAlign = TextAlign.Center,
        )
        Text(
            t.order.deliveredText,
            style = MaterialTheme.typography.bodyMedium,
            color = c.muted,
            textAlign = TextAlign.Center,
        )
    }
}
