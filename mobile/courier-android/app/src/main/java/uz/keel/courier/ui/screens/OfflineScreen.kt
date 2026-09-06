package uz.keel.courier.ui.screens

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
import uz.keel.design.GhostButton
import uz.keel.design.KeelTheme
import uz.keel.design.glass
import uz.keel.courier.t

// Opened with no network.
//
// ⚠️ **This screen exists because the login screen was standing in for it.** A
// launch asks the server who is signed in; when the request never arrives the
// answer used to be "signed out", so somebody on a dead connection met a
// password field, typed the right password, and blamed themselves. None of that
// is fixable by signing in, so nothing here offers to.
//
// ⚠️ **It retries by itself.** The ordinary way out is the network coming back —
// a courier rides out of the basement — and a screen that only cleared on a tap
// would lock somebody out mid-shift for as long as they did not think to press
// it.

private const val RETRY_MS = 5_000L

@Composable
fun OfflineScreen(address: String, onRetry: () -> Unit) {
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
    // ⚠️ And immediately when the app comes back to the front: a phone taken out
    // of a pocket where there is signal should not wait out a tick.
    val owner = LocalLifecycleOwner.current
    DisposableEffect(owner) {
        val obs = LifecycleEventObserver { _, e -> if (e == Lifecycle.Event.ON_RESUME) latest() }
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
        // ⚠️ Shown because the other cause of this screen is a wrong or dead
        // address, and the person who can tell is the one reading it.
        Text(address, style = MaterialTheme.typography.labelMedium, color = c.muted)
        Text(t.offline.retrying(tries), style = MaterialTheme.typography.labelSmall, color = c.muted)
        GhostButton(t.common.retry, icon = Icons.Rounded.Refresh) { tries += 1; latest() }
    }
}
