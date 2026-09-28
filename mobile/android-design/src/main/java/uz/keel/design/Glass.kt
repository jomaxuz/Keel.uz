package uz.keel.design

import android.graphics.RenderEffect
import android.graphics.Shader
import android.os.Build
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.asComposeRenderEffect
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

// Glass, and the honest version of it.
//
// ⚠️ **Android has no `backdrop-filter`, and pretending otherwise is the whole
// trap here.** On iOS a material samples whatever is behind it; on Android a
// `RenderEffect` blurs *the view it is set on*, not the backdrop. So the glass
// is built the way it actually works: a blurred **background layer** drawn once
// per screen, and translucent panels on top of it. A panel that tried to blur
// its own backdrop would either do nothing or blur its own text.
//
// ⚠️ **RenderEffect is API 31+.** Below that the blur is simply absent, and the
// design has to survive that — the phones sold into restaurants here are exactly
// the ones that will not have it. The fallback is a slightly more opaque fill
// (`glassStrong`), which reads as frosted rather than as broken. Never a
// stack-blur library: a per-frame CPU blur on a cheap phone is the stutter the
// Expo app spent a release removing.

/** The page behind everything: the ground colour and two soft lights.
 *
 *  ⚠️ **Glass needs something to refract.** With a flat backdrop every panel is
 *  a grey rectangle with a border, which is the version of this design that
 *  looks cheap. The two blobs are what makes the same panel read as glass at
 *  the top of a screen and as glass at the bottom.
 *
 *  ⚠️ **No blur on this layer any more, and that was a frame-rate bug.** It
 *  carried a 120dp `RenderEffect` over the whole screen — and a render effect is
 *  applied again every time the window is redrawn, which during a scroll is
 *  every frame. On Android 12+ phones that was the single most expensive thing
 *  on screen, and it bought nothing: a radial gradient that fades to
 *  transparent at its own radius is already a soft light. Below API 31 the blur
 *  never ran at all, so this is also the look most restaurants' phones always
 *  had. */
@Composable
fun KeelBackground(content: @Composable BoxScope.() -> Unit) {
    val c = KeelTheme.colors
    Box(Modifier.fillMaxSize().background(c.bg)) {
        Box(
            Modifier
                .fillMaxSize()
                .drawBehind {
                    drawCircle(
                        brush = Brush.radialGradient(
                            listOf(c.auraWarm, Color.Transparent),
                            center = Offset(size.width * 0.15f, size.height * 0.08f),
                            radius = size.minDimension * 0.75f,
                        ),
                        radius = size.minDimension * 0.75f,
                        center = Offset(size.width * 0.15f, size.height * 0.08f),
                    )
                    drawCircle(
                        brush = Brush.radialGradient(
                            listOf(c.auraCool, Color.Transparent),
                            center = Offset(size.width * 0.95f, size.height * 0.72f),
                            radius = size.minDimension * 0.8f,
                        ),
                        radius = size.minDimension * 0.8f,
                        center = Offset(size.width * 0.95f, size.height * 0.72f),
                    )
                },
        )
        content()
    }
}

/** Blur this layer, where the platform can. Silently nothing below API 31 —
 *  which is why every caller has a fallback that does not depend on it. */
fun Modifier.blurCompat(radius: Dp): Modifier =
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
        graphicsLayer {
            renderEffect = RenderEffect
                .createBlurEffect(radius.toPx(), radius.toPx(), Shader.TileMode.DECAL)
                .asComposeRenderEffect()
        }
    } else this

/** One pane of glass: translucent fill, hairline edge, and a highlight along
 *  the top.
 *
 *  ⚠️ **The highlight is the part that does the work.** A flat translucent box
 *  reads as a faded rectangle; the light running along its top edge is what the
 *  eye reads as a surface catching a light. It is one gradient stop, and it is
 *  the difference between "glassmorphic" and "opacity: 0.7".
 *
 *  @param strong for panels that carry text over busy content — a check row
 *  over a photograph — where the softer fill loses contrast. Also the fallback
 *  used below API 31, where there is no blur to do the separating. */
fun Modifier.glass(
    colors: KeelColors,
    shape: Shape = RoundedCornerShape(22.dp),
    strong: Boolean = false,
): Modifier {
    val blurred = Build.VERSION.SDK_INT >= Build.VERSION_CODES.S
    val fill = if (strong || !blurred) colors.glassStrong else colors.glass
    return this
        .clip(shape)
        .background(fill, shape)
        .background(
            Brush.verticalGradient(
                0f to colors.glassHighlight.copy(alpha = colors.glassHighlight.alpha * 0.55f),
                0.45f to Color.Transparent,
            ),
            shape,
        )
        .border(1.dp, colors.glassBorder, shape)
}

/** Glass that has to stand on its own — a dialog or a bottom sheet.
 *
 *  ⚠️ **A pane needs a ground before it needs a film.** Over a list, glass works
 *  because the page is behind it. A dialog floats over a dimmed screen and a
 *  bottom sheet over nothing at all, so the same modifier there is a translucent
 *  film over a shadow — light text on a half-lit photograph, which is where "I
 *  cannot read anything in the dark" comes from. The page colour goes down
 *  first, the glass on top of it.
 *
 *  ⚠️ Not fully opaque: at 1.0 this stops being glass and becomes a Material
 *  card, and the whole screen stops matching itself. */
fun Modifier.glassSheet(colors: KeelColors, shape: Shape): Modifier = this
    .background(colors.bg.copy(alpha = if (colors.dark) 0.92f else 0.86f), shape)
    .glass(colors, shape, strong = true)

/** The brand's own gradient, for the one control on a screen that acts.
 *
 *  ⚠️ **One per screen.** Two orange gradients on one screen and neither is the
 *  primary action any more — which on the check screen is "send to the kitchen",
 *  and a waiter who has to look for it is a waiter who taps the wrong thing. */
fun keelGradient(): Brush = Brush.linearGradient(
    listOf(KeelOrangeLift, KeelOrange),
    start = Offset.Zero,
    end = Offset(0f, Float.POSITIVE_INFINITY),
)

/** A soft shadow for glass sitting over a list.
 *
 *  ⚠️ **Drawn by the platform, because a shadow drawn by hand is not a shadow.**
 *  This began as a rounded rectangle filled with translucent black and offset —
 *  which has no blur, so it rendered as a hard grey slab sticking out from under
 *  the control, and on the first emulator screenshot the primary button looked
 *  like it was sitting on a grey card nobody designed.
 *
 *  ⚠️ **And then it was still too heavy, which is the second lesson.** Tinting
 *  the platform's spot colour up to make the shadow "visible" gives a dark band
 *  under every control — the same grey slab, blurred. Glass is separated from
 *  what is behind it by its own edge and its highlight; the shadow only has to
 *  stop it floating. So elevation stays in single digits, the spot colour is
 *  barely there, and the ambient one is softer still.
 */
fun Modifier.softShadow(
    shape: Shape = RoundedCornerShape(22.dp),
    /** How lifted this is, in the platform's own terms. ⚠️ Past about 8dp the
     *  shadow acquires a hard near edge, which is what made the hand-drawn one
     *  look wrong to begin with. */
    elevation: Dp = 3.dp,
    dark: Boolean = false,
): Modifier = shadow(
    elevation = elevation,
    shape = shape,
    clip = false,
    // ⚠️ Deliberately faint. A shadow somebody notices is a shadow that is
    // already too strong — on a light ground it reads as dirt under the control.
    ambientColor = Color.Black.copy(alpha = if (dark) 0.22f else 0.10f),
    spotColor = Color.Black.copy(alpha = if (dark) 0.30f else 0.14f),
)
