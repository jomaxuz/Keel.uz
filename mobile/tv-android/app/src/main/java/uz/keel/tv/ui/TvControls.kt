package uz.keel.tv.ui

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import uz.keel.design.KeelTheme
import uz.keel.design.keelGradient
import uz.keel.design.softShadow

// The two controls a television has, and the one rule they both obey.
//
// ⚠️ **The focus ring is the cursor.** A television has no pointer: the only way
// to know which control the D-pad is on is that it looks different — and
// "slightly different" is invisible from the far side of a room. Compose's
// `clickable` is focusable and draws nothing for it, which on a phone is
// correct and here is a screen nobody can get past.
//
// ⚠️ **Overscan.** Older sets cut 3–5% off every edge and there is no way to ask
// how much, so nothing important goes closer than this to the border. A pairing
// code 40 pixels from the edge is a code with a missing character on somebody's
// television — and a code is transcribed, not read.
val EDGE = 48.dp

/** The one control on a screen that acts.
 *
 *  ⚠️ Built from the same gradient and the same shadow as the phone
 *  applications' `PrimaryButton`, at a size a remote and a room can use. Sharing
 *  the shape and the orange is the point; sharing the 54dp height would not be. */
@Composable
fun TvPrimaryButton(
    label: String,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    val c = KeelTheme.colors
    var focused by remember { mutableStateOf(false) }
    val lift by animateFloatAsState(if (focused) 1.04f else 1f, label = "lift")
    val shape = RoundedCornerShape(20.dp)
    Row(
        modifier
            .fillMaxWidth()
            .height(72.dp)
            .scale(lift)
            .softShadow(shape, elevation = if (focused) 6.dp else 3.dp, dark = c.dark)
            .clip(shape)
            .background(keelGradient())
            // ⚠️ A ring rather than a colour change: the button is already the
            // brand's orange, and a focused state that repainted it would make
            // the only action on screen look like a different button.
            .border(
                width = if (focused) 3.dp else 0.dp,
                color = if (focused) Color.White else Color.Transparent,
                shape = shape,
            )
            .onFocusChanged { focused = it.isFocused }
            .clickable(onClick = onClick),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, color = c.onAccent, style = MaterialTheme.typography.headlineMedium)
    }
}
