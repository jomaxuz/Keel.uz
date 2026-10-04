package uz.keel.courier.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.LocalShipping
import androidx.compose.material.icons.rounded.MyLocation
import androidx.compose.material.icons.rounded.PowerSettingsNew
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.luminance
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import uz.keel.design.KeelTheme
import uz.keel.design.LocalLangHost
import uz.keel.design.LocationProblem
import uz.keel.design.message
import uz.keel.design.rememberLocationGate
import uz.keel.design.glass
import uz.keel.courier.location.LocationService
import uz.keel.courier.location.TrackingState
import uz.keel.courier.t

// The top of the orders screen: am I working, and does the restaurant know where
// I am.
//
// ⚠️ **Both questions in one card, because they are one question.** Being on
// shift is what starts the location stream, and the location is what opens the
// "delivered" button — a courier who saw only the switch would have no way to
// tell "no orders tonight" from "the dispatcher's map lost me an hour ago".

@Composable
fun ShiftCard(
    status: String,
    tracking: TrackingState,
    onStatus: (String) -> Unit,
) {
    val c = KeelTheme.colors
    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(22.dp)).padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text(t.shift.title, style = MaterialTheme.typography.labelMedium, color = c.muted)

        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            ShiftOption(
                label = t.shift.off,
                hint = t.shift.offHint,
                icon = Icons.Rounded.PowerSettingsNew,
                // ⚠️ Colour by meaning rather than by position: "off" is the
                // state that stops work arriving, and it reads as a warning for
                // exactly as long as it is true.
                tint = c.warn,
                on = status == "off",
                modifier = Modifier.weight(1f),
            ) { onStatus("off") }
            ShiftOption(
                label = t.shift.free,
                hint = t.shift.freeHint,
                icon = Icons.Rounded.CheckCircle,
                tint = c.ready,
                on = status == "free",
                modifier = Modifier.weight(1f),
            ) { onStatus("free") }
            ShiftOption(
                label = t.shift.busy,
                hint = t.shift.busyHint,
                icon = Icons.Rounded.LocalShipping,
                tint = c.accent,
                on = status == "busy",
                modifier = Modifier.weight(1f),
            ) { onStatus("busy") }
        }

        if (status == "off") {
            Text(t.shift.startHint, style = MaterialTheme.typography.labelMedium, color = c.muted)
        } else {
            LocationRow(tracking)
        }
    }
}

@Composable
private fun ShiftOption(
    label: String,
    hint: String,
    icon: ImageVector,
    tint: Color,
    on: Boolean,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    val c = KeelTheme.colors
    // ⚠️ **Ink chosen from the fill, not assumed white.** In the dark scheme
    // `warn` and `ready` are lifted to bright amber and mint so they read on a
    // near-black ground — and white on those was the selected option nobody
    // could read. Same luminance rule as `branded()`.
    val onTint = if (tint.luminance() > 0.45f) Color(0xFF1A1614) else Color.White
    Column(
        modifier
            .clip(RoundedCornerShape(16.dp))
            .background(if (on) tint else Color.Transparent, RoundedCornerShape(16.dp))
            .border(1.dp, if (on) tint else c.glassBorder, RoundedCornerShape(16.dp))
            .clickable(onClick = onClick)
            // ⚠️ Tall enough to be hit with a glove on. This is the control a
            // courier presses at a kerb, and the three sit side by side.
            .heightIn(min = 92.dp)
            .padding(horizontal = 6.dp, vertical = 12.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(4.dp, Alignment.CenterVertically),
    ) {
        Icon(icon, null, tint = if (on) onTint else c.muted, modifier = Modifier.size(19.dp))
        Text(
            label,
            style = MaterialTheme.typography.titleMedium,
            color = if (on) onTint else c.ink,
            textAlign = TextAlign.Center,
        )
        Text(
            hint,
            style = MaterialTheme.typography.labelSmall,
            color = if (on) onTint.copy(alpha = 0.85f) else c.muted,
            textAlign = TextAlign.Center,
        )
    }
}

/** What the location stream is doing, in one line a courier can act on. */
@Composable
private fun LocationRow(tracking: TrackingState) {
    val c = KeelTheme.colors
    val ctx = LocalContext.current
    val lang = LocalLangHost.current.current
    var granted by remember { mutableStateOf(LocationService.hasLocationPermission(ctx)) }
    var problem by remember { mutableStateOf<LocationProblem?>(null) }

    // ⚠️ **Android asks by itself, the moment the shift opens.** The Expo build
    // put a button here and waited to be pressed — so a courier could open a
    // shift, ride out, and discover at a door that nothing had ever been sent.
    //
    // ⚠️ **Through the gate, not a bare permission request.** A granted
    // permission with GPS switched off on the phone started the service, which
    // then received nothing, and the card sat on "waiting" all evening with no
    // hint of why. The gate raises Android's own "turn on location" dialog.
    // Approximate is accepted here (`precise = false`): a rough track is still
    // worth more than none, and the dispatcher sees the accuracy.
    val gate = rememberLocationGate(precise = false) { problem = it }
    fun begin() {
        problem = null
        gate.request {
            granted = true
            LocationService.start(ctx)
        }
    }
    LaunchedEffect(Unit) { begin() }

    problem?.let { p ->
        Row(
            Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(16.dp))
                .background(c.accentSoft, RoundedCornerShape(16.dp))
                // A tap tries again: after the courier has flipped the switch,
                // or granted the permission in the settings page it opened.
                .clickable { begin() }
                .padding(12.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(Icons.Rounded.MyLocation, null, tint = c.warn, modifier = Modifier.size(18.dp))
            Text(
                p.message(lang),
                Modifier.weight(1f),
                style = MaterialTheme.typography.labelMedium,
                color = c.ink,
            )
        }
        return
    }

    // Still asking — the system dialog is up, and it says more than a row would.
    if (!granted) return

    val time = tracking.lastSentAt?.let {
        android.text.format.DateFormat.getTimeFormat(ctx).format(java.util.Date(it))
    }
    Row(
        Modifier
            .fillMaxWidth()
            .background(c.accentSoft.copy(alpha = 0.35f), RoundedCornerShape(16.dp))
            .padding(12.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        // A filled dot rather than a tick: this is a state that is true right
        // now, and it stops being true while nobody is looking at it.
        Box(
            Modifier
                .size(10.dp)
                .background(if (tracking.live) c.ready else c.warn, CircleShape),
        )
        Column(Modifier.weight(1f)) {
            Text(
                if (tracking.live) t.geo.on else "${t.geo.on} · ${t.geo.waiting}",
                style = MaterialTheme.typography.bodyLarge,
                color = c.ink,
            )
            val line = listOfNotNull(
                time?.let { t.geo.lastSent(it) },
                if (tracking.pending > 0) t.geo.queued(tracking.pending) else null,
            ).joinToString(" · ")
            if (line.isNotEmpty()) {
                Text(line, style = MaterialTheme.typography.labelMedium, color = c.muted)
            }
            // ⚠️ **Which of the two modes this is, said plainly.** With the
            // service running the phone can go in a pocket; without it the
            // screen has to stay on, and a courier who does not know which they
            // have will find out at a door with a shut button.
            Text(
                if (tracking.running) t.geo.inBackground else t.geo.keepOpen,
                style = MaterialTheme.typography.labelMedium,
                color = c.muted,
            )
        }
    }
}
