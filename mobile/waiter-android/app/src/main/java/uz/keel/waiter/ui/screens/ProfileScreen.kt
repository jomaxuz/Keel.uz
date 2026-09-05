package uz.keel.waiter.ui.screens

import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Login
import androidx.compose.material.icons.rounded.Logout
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import kotlinx.coroutines.launch
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume
import uz.keel.waiter.data.ApiError
import uz.keel.waiter.data.KeelApi
import uz.keel.waiter.data.Staff
import uz.keel.waiter.data.StaffDay
import uz.keel.waiter.data.StaffReport
import uz.keel.waiter.data.money
import uz.keel.waiter.i18n.formatDuration
import uz.keel.waiter.t
import uz.keel.waiter.ui.components.GhostButton
import uz.keel.waiter.ui.components.PrimaryButton
import uz.keel.waiter.ui.components.ScreenHeader
import uz.keel.waiter.ui.theme.KeelTheme
import uz.keel.waiter.ui.theme.StatusColor
import uz.keel.waiter.ui.theme.glass

// How much somebody has worked, and when.
//
// ⚠️ **The same question the panel answers, from the same endpoint.**
// `/staff/report` already returns the day, its status and what it was against the
// roster — so this is a second *drawing* of one answer, never a second
// calculation. A phone that added up its own hours would disagree with the
// payroll screen, and the disagreement would be found on pay day.
//
// ⚠️ **And the colours are not chosen here.** `StatusColor` is
// `lib/attendance.ts` ported, so a shift that is amber in the office cannot be
// green in somebody's hand. Only the label is this app's.

@Composable
fun ProfileScreen(api: KeelApi, staff: Staff, bottomInset: PaddingValues) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()
    var report by remember { mutableStateOf<StaffReport?>(null) }
    var error by remember { mutableStateOf("") }
    val failedLoad = t.floor.failedLoad

    suspend fun load() {
        try {
            report = api.staffReport()
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else failedLoad
        }
    }
    LaunchedEffect(Unit) { load() }

    val r = report
    if (r == null) {
        Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            if (error.isNotEmpty()) {
                Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text(error, color = c.danger, textAlign = TextAlign.Center)
                    GhostButton(t.common.retry) { scope.launch { load() } }
                }
            } else CircularProgressIndicator(color = c.accent)
        }
        return
    }

    val openDay = r.days.firstOrNull { it.open }
    // ⚠️ Labels read in composition and closed over, so `dur` is an ordinary
    // function: the same helper `lib/attendance.ts` takes labels for, and for
    // the same reason — one implementation, three languages.
    val hourLabel = t.profile.hour
    val minuteLabel = t.profile.minute
    fun dur(m: Int) = formatDuration(m, hourLabel, minuteLabel)

    Column(Modifier.fillMaxSize()) {
        ScreenHeader(
            title = staff.name,
            subtitle = staff.position.ifEmpty { t.profile.title },
            trailing = {
                // ⚠️ Named, not only coloured: "on shift" is the one state on
                // this screen somebody might act on, and a dot alone says nothing
                // to whoever cannot separate these hues.
                if (openDay != null) {
                    Row(
                        Modifier.background(c.accentSoft, CircleShape).padding(horizontal = 12.dp, vertical = 7.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                    ) {
                        Box(Modifier.size(8.dp).background(c.accent, CircleShape))
                        Text(t.profile.onShift, color = c.accent, style = MaterialTheme.typography.labelMedium)
                    }
                }
            },
        )

        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState())
                .padding(start = 16.dp, end = 16.dp, top = 4.dp)
                .padding(bottom = bottomInset.calculateBottomPadding() + 24.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            // ⚠️ First on the screen, because it is the only thing here somebody
            // *does*. The hours below are a record; this is the act that creates
            // them, and burying it under a calendar would put the daily action
            // beneath the monthly reading.
            ClockButton(api, openDay != null) { scope.launch { load() } }

            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                Trend(t.profile.today, dur(r.today.current), Modifier.weight(1f))
                Trend(t.profile.week, dur(r.week.current), Modifier.weight(1f))
                Trend(t.profile.month, dur(r.month.current), Modifier.weight(1f))
            }

            Column(
                Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                StatLine(t.profile.worked, dur(r.totals.worked))
                StatLine(t.profile.expected, dur(r.totals.expected))
                StatLine(
                    t.profile.diff, dur(r.totals.diff),
                    tone = if (r.totals.diff >= 0) StatusColor["ok"] else StatusColor["under"],
                )
                if (r.periodPay > 0) StatLine("—", money(r.periodPay))
            }

            Text(t.profile.calendar, style = MaterialTheme.typography.titleMedium, color = c.ink)

            /** ⚠️ **Monday first.** The roster is written a week at a time and a
             *  Sunday-first grid puts the end of the week at its start — the
             *  panel's calendar reads Monday first for the same reason, and two
             *  calendars that disagree about where a week begins are two
             *  calendars nobody trusts. */
            val lead = remember(r.days) {
                r.days.firstOrNull()?.let {
                    val d = runCatching { java.time.LocalDate.parse(it.date) }.getOrNull()
                    // DayOfWeek: Monday is 1, so Monday-first padding is value-1.
                    d?.dayOfWeek?.value?.minus(1) ?: 0
                } ?: 0
            }
            FlowRow(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(6.dp),
                verticalArrangement = Arrangement.spacedBy(6.dp),
                maxItemsInEachRow = 7,
            ) {
                repeat(lead) { Box(Modifier.width(40.dp).height(46.dp)) }
                r.days.forEach { DayCell(it) }
            }

            if (r.days.isEmpty()) {
                Text(t.profile.noData, color = c.muted, textAlign = TextAlign.Center, modifier = Modifier.fillMaxWidth())
            }

            // The legend, because eight colours is more than anybody memorises —
            // and only the ones this period actually contains.
            FlowRow(horizontalArrangement = Arrangement.spacedBy(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                r.days.map { it.status }.distinct().forEach { st ->
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                        Box(Modifier.size(8.dp).background(StatusColor[st] ?: c.muted, CircleShape))
                        Text(t.profile.status[st] ?: st, color = c.muted, style = MaterialTheme.typography.labelMedium)
                    }
                }
            }
        }
    }
}

@Composable
private fun DayCell(day: StaffDay) {
    val c = KeelTheme.colors
    val colour = StatusColor[day.status] ?: c.muted
    val plain = day.status == "off" || day.status == "upcoming"
    Box(
        // ⚠️ A wash rather than a fill: a grid of saturated squares is a pattern
        // nobody reads, and the number on top has to stay legible in both schemes.
        // The border carries the colour at full strength.
        Modifier.width(40.dp).height(46.dp)
            .background(
                if (plain) Color.Transparent else colour.copy(alpha = if (c.dark) 0.16f else 0.12f),
                RoundedCornerShape(12.dp),
            )
            .border(1.dp, if (plain) c.line else colour.copy(alpha = 0.7f), RoundedCornerShape(12.dp)),
        contentAlignment = Alignment.Center,
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Text(
                day.date.takeLast(2).trimStart('0'),
                style = MaterialTheme.typography.labelMedium, color = c.ink,
            )
            if (day.worked > 0) {
                Text(
                    "${Math.round(day.worked / 6.0) / 10.0}",
                    style = MaterialTheme.typography.labelSmall, color = c.muted,
                )
            }
        }
    }
}

@Composable
private fun Trend(label: String, value: String, modifier: Modifier = Modifier) {
    val c = KeelTheme.colors
    Column(
        modifier.glass(c, RoundedCornerShape(20.dp)).padding(vertical = 14.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(2.dp),
    ) {
        Text(label, style = MaterialTheme.typography.labelMedium, color = c.muted)
        Text(value, style = MaterialTheme.typography.titleMedium, color = c.ink)
    }
}

@Composable
private fun StatLine(label: String, value: String, tone: Color? = null) {
    val c = KeelTheme.colors
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(label, Modifier.weight(1f), color = c.inkSoft, style = MaterialTheme.typography.bodyMedium)
        Text(value, color = tone ?: c.ink, style = MaterialTheme.typography.bodyMedium)
    }
}

// Starting and ending a shift, from the thing that has the GPS.
//
// ⚠️ **The phone is where this belongs and it was the last screen to get it.**
// The server requires a position and checks it against the branch's own
// (`geofenceBlocked`) — so an employee was opening their shift on some other
// screen and then working from this one.
//
// ⚠️ **Permission is asked when the button is pressed, not at launch.** A prompt
// on first run, before anybody knows what the app is for, is answered "no" — and
// on both platforms a refused location is awkward to recover. Asked at the moment
// it is obviously needed, it is a question with a visible reason.
@Composable
private fun ClockButton(api: KeelApi, open: Boolean, onChanged: () -> Unit) {
    val c = KeelTheme.colors
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    val needLocation = t.clock.needLocation
    val failed = t.clock.failed

    suspend fun punch() {
        busy = true
        error = ""
        try {
            val client = LocationServices.getFusedLocationProviderClient(ctx)
            // ⚠️ **Balanced accuracy, not the highest.** The branch's radius is
            // 50 metres — chosen because a phone's GPS is 10–30 outdoors and
            // worse inside — so the extra seconds the highest setting spends do
            // not change the answer, and they are spent with somebody standing at
            // a door.
            val loc = suspendCancellableCoroutine { cont ->
                @Suppress("MissingPermission")
                client.getCurrentLocation(Priority.PRIORITY_BALANCED_POWER_ACCURACY, null)
                    .addOnSuccessListener { cont.resume(it) }
                    .addOnFailureListener { cont.resume(null) }
            }
            if (loc == null) { error = failed; return }
            api.staffClock(
                if (open) "out" else "in",
                loc.latitude, loc.longitude, loc.accuracy.toDouble(),
            )
            onChanged()
        } catch (e: Throwable) {
            // The server's own words: "you are 400 m from the branch" is a
            // sentence somebody can act on, and it is the one refusal that is not
            // a fault.
            error = if (e is ApiError) e.message else failed
        } finally { busy = false }
    }

    val ask = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        // ⚠️ Named plainly rather than as a failure: refusing is a choice, and the
        // way back is the system settings, which is what the sentence says.
        if (granted) scope.launch { punch() } else error = needLocation
    }

    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        if (open) {
            GhostButton(
                t.clock.end, Modifier.fillMaxWidth(),
                icon = Icons.Rounded.Logout, enabled = !busy,
            ) { start(ctx, ask) { scope.launch { punch() } } }
        } else {
            PrimaryButton(t.clock.start, icon = Icons.Rounded.Login, enabled = !busy, busy = busy) {
                start(ctx, ask) { scope.launch { punch() } }
            }
        }
        if (error.isNotEmpty()) {
            Text(error, color = c.danger, style = MaterialTheme.typography.bodyMedium)
        }
    }
}

private fun start(
    ctx: android.content.Context,
    ask: androidx.activity.result.ActivityResultLauncher<String>,
    go: () -> Unit,
) {
    val granted = ContextCompat.checkSelfPermission(ctx, Manifest.permission.ACCESS_FINE_LOCATION) ==
        PackageManager.PERMISSION_GRANTED ||
        ContextCompat.checkSelfPermission(ctx, Manifest.permission.ACCESS_COARSE_LOCATION) ==
        PackageManager.PERMISSION_GRANTED
    if (granted) go() else ask.launch(Manifest.permission.ACCESS_FINE_LOCATION)
}
