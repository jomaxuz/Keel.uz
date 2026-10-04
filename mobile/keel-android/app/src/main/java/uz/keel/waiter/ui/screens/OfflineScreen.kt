package uz.keel.waiter.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.CloudOff
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import kotlinx.coroutines.delay
import uz.keel.waiter.t
import uz.keel.design.*

// Opened with no network.
//
// ⚠️ **This screen exists because the login screen was standing in for it.** A
// launch asks the server who is signed in; when the request never arrives the
// answer used to be "signed out", so a waiter in a basement, on a dead wifi or
// out of data met a password field. They type the right password, it fails, and
// the app says "could not sign in" — so they type it again, blaming themselves
// for a network they cannot see. None of that is fixable by signing in, and a
// screen that offers a way out it does not have is worse than one that says
// plainly what is wrong.
//
// ⚠️ **It retries by itself.** The ordinary way out is the network coming back —
// walking out of the cellar, the router rebooting — and a screen that only
// cleared on a tap would keep somebody locked out of a shift they are standing
// in the middle of. The button stays because five seconds is a long time when
// you are holding a tray.

/** Short enough that walking back into signal clears the screen while the phone
 *  is still in the hand, long enough to be nothing on a battery. */
private const val RETRY_MS = 5_000L

@Composable
fun OfflineScreen(
    /** ⚠️ Shown because the other cause of this screen is a wrong or dead
     *  address, and the person who can tell is the one reading it. */
    address: String,
    onRetry: () -> Unit,
) {
    val c = KeelTheme.colors
    var tries by remember { mutableIntStateOf(0) }
    val latest by rememberUpdatedState(onRetry)

    LaunchedEffect(Unit) {
        while (true) {
            delay(RETRY_MS)
            tries += 1
            latest()
        }
    }
    // ⚠️ And immediately when the app comes back to the front: a phone brought
    // out of a pocket in a place with signal should not wait out a tick.
    val owner = LocalLifecycleOwner.current
    DisposableEffect(owner) {
        val obs = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) latest()
        }
        owner.lifecycle.addObserver(obs)
        onDispose { owner.lifecycle.removeObserver(obs) }
    }

    Column(
        Modifier.fillMaxSize().padding(28.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(14.dp, Alignment.CenterVertically),
    ) {
        Box(Modifier.size(72.dp).glass(c, CircleShape), contentAlignment = Alignment.Center) {
            Icon(Icons.Rounded.CloudOff, null, tint = c.muted, modifier = Modifier.size(32.dp))
        }
        Text(t.offline.title, style = MaterialTheme.typography.headlineMedium, color = c.ink)
        Box(Modifier.widthIn(max = 380.dp)) {
            Text(
                t.offline.body, style = MaterialTheme.typography.bodyMedium,
                color = c.inkSoft, textAlign = TextAlign.Center,
            )
        }
        Text(address, style = MaterialTheme.typography.labelMedium, color = c.muted)
        Text(
            t.offline.retrying(tries),
            style = MaterialTheme.typography.labelSmall, color = c.muted,
        )
        GhostButton(t.common.retry, icon = Icons.Rounded.Refresh) { tries += 1; latest() }
    }
}
