package uz.keel.tv.ui.screens

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.delay
import uz.keel.design.GlassField
import uz.keel.design.KeelTheme
import uz.keel.design.LangSwitch
import uz.keel.design.glass
import uz.keel.tv.R
import uz.keel.tv.data.TVScreenSelf
import uz.keel.tv.t
import uz.keel.tv.ui.EDGE
import uz.keel.tv.ui.TvPrimaryButton

// What the television draws when it is not playing anything.
//
// ⚠️ **Read from four metres away, by somebody who is not looking for it.** Every
// size here is chosen against that distance, not against a phone: the pairing
// code is the largest thing this app ever draws, because a manager reads it
// aloud across a dining room while typing it into a laptop.
//
// ⚠️ **The same glass, orange and near-black as the phone applications**, from
// the shared module — this is one product, and the screen on the wall is the
// half of it a guest sees. What is not shared is the scale: everything here is
// sized from the width of the set, so one build reads correctly on a 32" in a
// corner shop and on a 65" in a mall.

/** The frame between launch and knowing anything.
 *
 *  ⚠️ **It says something, and that is not decoration.** This drew an empty view
 *  once, and the first time the app got stuck here the result was a black
 *  rectangle on a wall with no code, no message and nothing to press. A blank
 *  screen and a crashed app are the same picture; a screen with the product's
 *  name on it is at least a screen that is running. */
@Composable
fun LoadingScreen() = Centered {
    val c = KeelTheme.colors
    Text(t.appName, style = MaterialTheme.typography.headlineMedium, color = c.ink)
    Text(t.idle.connecting, style = MaterialTheme.typography.titleLarge, color = c.muted)
}

/** The first screen a television ever shows: which restaurant is this.
 *
 *  ⚠️ **One short word, and that is the whole design.** Whoever is installing
 *  knows the restaurant as "osh", not as "https://osh.keel.uz/api/v1" — and on a
 *  remote control every extra character is four button presses. The rule that
 *  turns one into the other is the shared `ServerAddress`, so the phone
 *  applications, the Windows till and this set resolve it identically. */
@Composable
fun ServerScreen(
    onSubmit: (String) -> Boolean,
    /** ⚠️ Comes back **filled in** after a failed attempt. Retyping a word on a
     *  D-pad is four button presses a character, and the likeliest fix is one
     *  wrong letter — an empty field would make the correction cost more than
     *  the mistake. */
    initial: String = "",
    unreachable: Boolean = false,
    /** Whether the server answered at all. ⚠️ It changes who has to act. */
    answered: Boolean = false,
) {
    val c = KeelTheme.colors
    var value by remember(initial) { mutableStateOf(initial) }
    var bad by remember { mutableStateOf(false) }

    Box(Modifier.fillMaxSize()) {
        Column(
            Modifier.fillMaxSize().imePadding().padding(EDGE),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp, Alignment.CenterVertically),
        ) {
            // ⚠️ Keel's own mark, not a stock glyph: this is the one screen
            // where the app shows its own identity rather than the restaurant's.
            Box(
                Modifier.size(104.dp).background(c.navy, RoundedCornerShape(30.dp)),
                contentAlignment = Alignment.Center,
            ) {
                Image(
                    painterResource(R.drawable.ic_launcher_foreground),
                    contentDescription = null,
                    modifier = Modifier.size(96.dp),
                )
            }
            Text(t.appName, fontSize = 44.sp, fontWeight = FontWeight.Bold, color = c.ink)
            Text(t.server.title, style = MaterialTheme.typography.titleLarge, color = c.muted)

            Box(Modifier.widthIn(max = 560.dp)) {
                GlassField(
                    value = value,
                    onValueChange = { bad = false; value = it },
                    placeholder = t.server.placeholder,
                    keyboardOptions = KeyboardOptions(
                        capitalization = KeyboardCapitalization.None,
                        autoCorrectEnabled = false,
                        keyboardType = KeyboardType.Uri,
                    ),
                )
            }
            Box(Modifier.widthIn(max = 560.dp)) {
                TvPrimaryButton(t.server.next) { if (!onSubmit(value)) bad = true }
            }
            Text(
                when {
                    bad -> t.server.bad
                    unreachable && answered -> t.server.outdated
                    unreachable -> t.server.unreachable
                    else -> t.server.hint
                },
                style = MaterialTheme.typography.titleMedium,
                color = if (bad || unreachable) c.accent else c.muted,
                textAlign = TextAlign.Center,
                modifier = Modifier.widthIn(max = 720.dp),
            )
        }
        // ⚠️ After the column, never before it: inside a Box the last child is
        // drawn on top and takes the input, and a full-size column declared
        // later swallows every press meant for this.
        //
        // ⚠️ **The only screen with a language button.** It is pressed once in
        // the life of a television, by the person mounting it; a room full of
        // guests has no use for it and the remote is in a drawer by then.
        LangSwitch(Modifier.align(Alignment.TopEnd).padding(EDGE))
    }
}

/** The pairing code, and nothing else.
 *
 *  ⚠️ **No instructions for the guest, and none for the remote.** This screen
 *  hangs in a public room; whoever needs it is holding a laptop and has already
 *  been told what to do with it. What it does say is *where* to type the code,
 *  because that is the one thing somebody standing in front of it may not
 *  know. */
@Composable
fun PairingScreen(code: String, expiresAt: Long) {
    val c = KeelTheme.colors
    // ⚠️ Sized from the set rather than fixed: this app runs on a 32" screen in
    // a corner shop and a 65" in a mall, and a code that fits one is either
    // unreadable or clipped on the other.
    val width = LocalConfiguration.current.screenWidthDp
    val codeSize = (width / 7).coerceIn(72, 220).sp

    var left by remember(expiresAt) {
        mutableIntStateOf(secondsLeft(expiresAt))
    }
    LaunchedEffect(expiresAt) {
        while (true) {
            left = secondsLeft(expiresAt)
            delay(1000)
        }
    }

    Centered {
        Text(t.pairing.lead, style = MaterialTheme.typography.titleLarge, color = c.muted)
        // Spaced into two halves: six characters read as one word are
        // transcribed with a character in the wrong place.
        Text(
            "${code.take(3)} ${code.drop(3)}",
            fontSize = codeSize,
            fontWeight = FontWeight.ExtraBold,
            // A monospaced face so 8 and B cannot trade places at four metres.
            fontFamily = FontFamily.Monospace,
            letterSpacing = 12.sp,
            color = c.ink,
        )
        Text(t.pairing.where, style = MaterialTheme.typography.titleLarge, color = c.ink)
        // ⚠️ The countdown is told, not hidden: a manager who sees the code
        // change mid-typing needs to know that is normal, or the next thing they
        // do is report a broken screen. And it counts the *real* window — the
        // Expo build showed the server's ninety seconds while quietly fetching a
        // new code every ten.
        Text(
            if (left > 0) t.pairing.renews(left) else t.pairing.renewing,
            style = MaterialTheme.typography.titleMedium,
            color = c.muted,
        )
    }
}

private fun secondsLeft(expiresAt: Long): Int =
    ((expiresAt - System.currentTimeMillis()) / 1000).coerceAtLeast(0).toInt()

/** A paired screen with nothing to play.
 *
 *  ⚠️ **A placeholder that says which screen this is**, because the next thing
 *  somebody does after pairing is walk to the other televisions and pair those —
 *  and four identical black screens is how two of them end up named the same. */
@Composable
fun PairedScreen(
    screen: TVScreenSelf?,
    offline: Boolean,
    downloading: Boolean,
    /** Paired, but the first heartbeat has not landed — so whether there is a
     *  playlist at all is not yet known. */
    waiting: Boolean,
) {
    val c = KeelTheme.colors
    Centered {
        Box(
            Modifier.glass(c, RoundedCornerShape(32.dp)).padding(horizontal = 48.dp, vertical = 36.dp),
        ) {
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Text(
                    screen?.name ?: t.appName,
                    fontSize = 44.sp, fontWeight = FontWeight.Bold, color = c.ink,
                )
                Text(
                    screen?.branchName.orEmpty(),
                    style = MaterialTheme.typography.titleLarge, color = c.muted,
                )
                Text(
                    hint(offline, downloading, waiting),
                    style = MaterialTheme.typography.titleMedium,
                    color = c.muted,
                    textAlign = TextAlign.Center,
                )
            }
        }
    }
}

/** What the room is told while there is nothing to play.
 *
 *  ⚠️ **Four states, because for a long time there was one and it was wrong in
 *  three of them.** A television paired at the wall and given two videos said
 *  "no content — go to the panel" for five straight minutes: sixty seconds
 *  waiting for its first heartbeat, then four downloading the clips. Every
 *  second of that, the message sent somebody to a laptop to fix a screen that
 *  was working — and the one message that must never be wrong is the one that
 *  names a culprit.
 *
 *  The order matters: no connection outranks everything (nothing else can be
 *  known), then work in progress, then not-yet-known, and only what is left is
 *  genuinely an empty playlist. */
@Composable
private fun hint(offline: Boolean, downloading: Boolean, waiting: Boolean): String = when {
    // ⚠️ Said quietly and never as an error: the room is open, the guests are
    // eating, and nothing about a dropped wifi is theirs to worry about.
    offline -> t.idle.offline
    // A minute of video is tens of megabytes over the restaurant's own wifi, and
    // the set is doing exactly what it should. Nobody is asked to do anything.
    downloading -> t.idle.downloading
    waiting -> t.idle.checking
    // ⚠️ Only this one names where to go, because the person reading it has just
    // paired the television and the answer is on a laptop in the back office.
    else -> t.idle.noContent
}

/** The layout every one of these screens is: one column, centred, inside the
 *  overscan margin. */
@Composable
private fun Centered(content: @Composable () -> Unit) {
    Column(
        Modifier.fillMaxSize().padding(EDGE),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(16.dp, Alignment.CenterVertically),
    ) { content() }
}
