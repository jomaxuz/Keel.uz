package uz.keel.tv.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import uz.keel.design.KeelTheme
import uz.keel.tv.Board
import uz.keel.tv.t
import uz.keel.tv.ui.EDGE

// The order board.
//
// ⚠️ **Read from four metres away by somebody looking for one number among
// twenty.** That is the only thing this screen has to do well, and it is what
// every size, colour and position here is chosen against — not density. A board
// that fits more numbers by making them smaller has failed at its job while
// looking like it is doing more of it.
//
// ⚠️ **Ready is on the right and louder.** The two columns are not equals: a
// guest checks "is mine ready" many times and "is mine cooking" once, and the
// answer they came for should not need finding.
//
// ⚠️ **Numbers and nothing else.** Everyone in the room reads this, including
// the people whose orders are not on it. A board carrying names is a customer
// list on a wall.

/** The whole wall: `board` mode. */
@Composable
fun BoardScreen(board: Board, branchName: String?) {
    val c = KeelTheme.colors
    // ⚠️ Sized from the set: a number tuned for a 32" screen by a counter is
    // unreadable or absurd on a 65" across a room.
    val numberSize = (LocalConfiguration.current.screenWidthDp / 22).sp

    if (!board.hasAnything) {
        // ⚠️ **Not an empty grid with two headings.** A board drawn with nothing
        // under it reads as broken — and between lunch and dinner it would read
        // that way for hours. The room's own name is a screen that is plainly on.
        Box(Modifier.fillMaxSize().padding(EDGE), Alignment.Center) {
            Text(
                branchName ?: t.appName,
                fontSize = 40.sp, fontWeight = FontWeight.Bold, color = c.muted,
            )
        }
        return
    }

    Row(
        Modifier.fillMaxSize().padding(EDGE),
        horizontalArrangement = Arrangement.spacedBy(EDGE),
    ) {
        Column(Modifier.weight(1f)) {
            BoardColumn(t.idle.cooking, board.cooking, numberSize, ready = false)
        }
        // The ready column is given the visual weight as well as the better side.
        Box(Modifier.width(2.dp).fillMaxSize().background(c.line))
        Column(Modifier.weight(1f).padding(start = EDGE)) {
            BoardColumn(t.idle.ready, board.ready, numberSize, ready = true)
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun BoardColumn(
    title: String,
    numbers: List<String>,
    size: androidx.compose.ui.unit.TextUnit,
    ready: Boolean,
) {
    val c = KeelTheme.colors
    Column(verticalArrangement = Arrangement.spacedBy(20.dp)) {
        Text(
            title.uppercase(),
            fontSize = 28.sp,
            fontWeight = FontWeight.SemiBold,
            letterSpacing = 2.sp,
            color = if (ready) c.ready else c.muted,
        )
        FlowRow(
            horizontalArrangement = Arrangement.spacedBy(24.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            numbers.forEach { n ->
                Text(
                    n,
                    fontSize = size,
                    fontWeight = FontWeight.ExtraBold,
                    // Monospaced, so 8 and B cannot trade places at four metres.
                    fontFamily = FontFamily.Monospace,
                    color = if (ready) c.ink else c.muted,
                )
            }
        }
    }
}

/** `split` mode: the loop plays, with the ready numbers along the bottom.
 *
 *  ⚠️ **Ready only, not both columns.** A strip is a glance, and half a metre of
 *  wall cannot carry two lists — whichever one it carries badly is the one
 *  somebody misreads. Ready is the answer people are waiting for; the rest is on
 *  the receipt.
 *
 *  ⚠️ **It disappears when there is nothing to say**, giving the video the whole
 *  wall back. A permanent empty bar across a promotional video is a restaurant's
 *  own screen eaten by furniture. */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun BoardStrip(board: Board, modifier: Modifier = Modifier) {
    val c = KeelTheme.colors
    if (board.ready.isEmpty()) return
    val size = (LocalConfiguration.current.screenWidthDp / 32).sp
    Row(
        modifier
            .fillMaxWidth()
            // ⚠️ Solid, not translucent: a number over a video is legible on
            // some frames and not on others, and nobody is going to wait for a
            // darker shot. This is the one place in the app that does not use
            // the glass, and the reason is what glass is *for*.
            .background(Color(0xFF000814))
            .padding(horizontal = EDGE, vertical = 16.dp),
        horizontalArrangement = Arrangement.spacedBy(24.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            t.idle.ready.uppercase(),
            fontSize = 22.sp,
            fontWeight = FontWeight.SemiBold,
            letterSpacing = 2.sp,
            color = c.ready,
        )
        board.ready.forEach { n ->
            Text(
                n,
                fontSize = size,
                fontWeight = FontWeight.ExtraBold,
                fontFamily = FontFamily.Monospace,
                color = Color.White,
            )
        }
    }
}
