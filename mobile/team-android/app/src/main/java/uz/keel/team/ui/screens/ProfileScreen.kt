package uz.keel.team.ui.screens

import android.Manifest
import android.annotation.SuppressLint
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Login
import androidx.compose.material.icons.rounded.Logout
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
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
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await
import uz.keel.design.GhostButton
import uz.keel.design.KeelTheme
import uz.keel.design.Money
import uz.keel.design.PrimaryButton
import uz.keel.design.StatusColor
import uz.keel.design.formatDuration
import uz.keel.design.glass
import uz.keel.team.data.ApiError
import uz.keel.team.data.KeelApi
import uz.keel.team.data.Staff
import uz.keel.team.data.StaffDay
import uz.keel.team.data.StaffReport
import uz.keel.team.t

// How much somebody has worked, and when.
//
// ⚠️ **The same question the panel answers, from the same endpoint.**
// `/staff/report` already returns the day, its status and what it was against
// the roster — so this is a second *drawing* of one answer, never a second
// calculation. A phone that added up its own hours would disagree with the
// payroll screen, and the disagreement would be found on pay day.
//
// ⚠️ **And the colours are not chosen here.** `StatusColor` lives in the shared
// design module beside the panel's own, so a shift that is amber in the office
// cannot be green in somebody's hand. Only the label is this app's.

@Composable
fun ProfileScreen(
    api: KeelApi,
    staff: Staff,
    onShift: Boolean,
    bottomInset: PaddingValues,
    onShiftChanged: (Boolean) -> Unit,
) {
    val c = KeelTheme.colors
    var report by remember { mutableStateOf<StaffReport?>(null) }
    var error by remember { mutableStateOf("") }
    var tick by remember { mutableIntStateOf(0) }
    val loadFailed = t.profile.loadFailed

    LaunchedEffect(tick) {
        try {
            report = api.report()
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else loadFailed
        }
    }

    val data = report
    if (data == null) {
        Box(Modifier.fillMaxSize(), Alignment.Center) {
            if (error.isEmpty()) {
                CircularProgressIndicator(color = c.accent)
            } else {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    Text(error, color = c.danger, style = MaterialTheme.typography.bodyMedium)
                    GhostButton(t.common.retry) { tick += 1 }
                }
            }
        }
        return
    }

    // ⚠️ Read in composition and captured: the dictionary is a composable read,
    // and a helper that reached for it from outside one would not compile — and
    // would be the wrong language if it did.
    val hourWord = t.profile.hour
    val minuteWord = t.profile.minute
    fun dur(minutes: Int) = formatDuration(minutes, hourWord, minuteWord)

    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp)
            .padding(bottom = bottomInset.calculateBottomPadding()),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Row(
            Modifier.statusBarsPadding().padding(top = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(Modifier.weight(1f)) {
                Text(staff.name, style = MaterialTheme.typography.headlineMedium, color = c.ink)
                Text(
                    staff.position.ifEmpty { t.profile.title },
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.muted,
                )
            }
            // ⚠️ Named, not only coloured: "on shift" is the one state on this
            // screen somebody might act on, and a dot alone says nothing to
            // whoever cannot separate these hues.
            if (onShift) {
                Row(
                    Modifier
                        .background(c.accentSoft, RoundedCornerShape(999.dp))
                        .padding(horizontal = 10.dp, vertical = 6.dp),
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Box(Modifier.size(8.dp).background(c.accent, CircleShape))
                    Text(
                        t.profile.onShift,
                        style = MaterialTheme.typography.labelMedium,
                        color = c.accent,
                    )
                }
            }
        }

        // ⚠️ First on the screen, because it is the only thing here somebody
        // *does*. The hours below are a record; this is the act that creates
        // them, and burying it under a calendar would put the daily action
        // beneath the monthly reading.
        ClockButton(api, onShift) { open ->
            onShiftChanged(open)
            tick += 1
        }

        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Trend(t.profile.today, dur(data.today.current), Modifier.weight(1f))
            Trend(t.profile.week, dur(data.week.current), Modifier.weight(1f))
            Trend(t.profile.month, dur(data.month.current), Modifier.weight(1f))
        }

        Column(
            Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
        ) {
            Line(t.profile.worked, dur(data.totals.worked))
            Line(t.profile.expected, dur(data.totals.expected))
            Line(
                t.profile.diff,
                dur(data.totals.diff),
                tone = StatusColor[if (data.totals.diff >= 0) "ok" else "under"],
            )
            if (data.periodPay > 0) {
                Row(
                    Modifier.fillMaxWidth().padding(vertical = 7.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        "—",
                        Modifier.weight(1f),
                        style = MaterialTheme.typography.bodyMedium,
                        color = c.inkSoft,
                    )
                    Money(data.periodPay, style = MaterialTheme.typography.titleMedium)
                }
            }
        }

        Text(t.profile.calendar, style = MaterialTheme.typography.titleLarge, color = c.ink)
        Calendar(data.days)
        if (data.days.isEmpty()) {
            Text(
                t.profile.noData,
                Modifier.fillMaxWidth(),
                style = MaterialTheme.typography.bodyMedium,
                color = c.muted,
                textAlign = TextAlign.Center,
            )
        }
        Legend(data.days)
    }
}

/** Starting and ending a shift, from the thing that has the GPS.
 *
 *  ⚠️ **The phone is where this belongs and it was the last screen to get it.**
 *  The server requires a position and checks it against the branch's own
 *  (`geofenceBlocked`) — so an employee was opening their shift on some other
 *  screen and then working from this one.
 *
 *  ⚠️ **Permission is asked when the button is pressed, not at launch.** A
 *  location prompt on first run, before anybody knows what the app is for, is
 *  answered "no" — and a refused location is awkward to recover. Asked at the
 *  moment it is obviously needed, it is a question with a visible reason. That
 *  is the opposite of the courier app, where the permission *is* the shift and
 *  is asked the moment one opens: here it is one fix on one press. */
@Composable
private fun ClockButton(api: KeelApi, open: Boolean, onChanged: (Boolean) -> Unit) {
    val c = KeelTheme.colors
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    val needLocation = t.clock.needLocation
    val failed = t.clock.failed

    fun punch() {
        busy = true
        error = ""
        scope.launch {
            try {
                val fix = currentFix(ctx)
                if (fix == null) {
                    error = needLocation
                    return@launch
                }
                api.clock(if (open) "out" else "in", fix.first, fix.second, fix.third)
                onChanged(!open)
            } catch (e: Throwable) {
                // The server's own words: "you are 400 m from the branch" is a
                // sentence somebody can act on, and it is the one refusal here
                // that is not a fault.
                error = if (e is ApiError) e.message else failed
            } finally {
                busy = false
            }
        }
    }

    val ask = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions(),
    ) { result ->
        // ⚠️ Named plainly rather than as a failure: refusing is a choice, and
        // the way back is the phone's settings, which is what the sentence says.
        if (result.values.any { it }) punch() else error = needLocation
    }

    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        if (open) {
            GhostButton(
                if (busy) t.common.loading else t.clock.end,
                Modifier.fillMaxWidth(),
                Icons.Rounded.Logout,
            ) { start(ctx, ask::launch, ::punch) }
        } else {
            PrimaryButton(
                label = t.clock.start,
                icon = Icons.Rounded.Login,
                busy = busy,
            ) { start(ctx, ask::launch, ::punch) }
        }
        if (error.isNotEmpty()) {
            Text(error, style = MaterialTheme.typography.labelMedium, color = c.danger)
        }
    }
}

/** Ask if we have to, punch if we can. */
private fun start(
    ctx: android.content.Context,
    ask: (Array<String>) -> Unit,
    punch: () -> Unit,
) {
    val granted = androidx.core.content.ContextCompat.checkSelfPermission(
        ctx, Manifest.permission.ACCESS_FINE_LOCATION,
    ) == android.content.pm.PackageManager.PERMISSION_GRANTED ||
        androidx.core.content.ContextCompat.checkSelfPermission(
            ctx, Manifest.permission.ACCESS_COARSE_LOCATION,
        ) == android.content.pm.PackageManager.PERMISSION_GRANTED
    if (granted) punch() else ask(
        // ⚠️ **Both, in one dialog.** Android 12 lets somebody grant
        // "approximate only", and asking for fine alone means that answer
        // arrives as a refusal — and the branch radius is fifty metres, which
        // an approximate fix often clears.
        arrayOf(
            Manifest.permission.ACCESS_FINE_LOCATION,
            Manifest.permission.ACCESS_COARSE_LOCATION,
        ),
    )
}

/** One position, now.
 *
 *  ⚠️ **Balanced accuracy, not the highest.** The branch's radius is fifty
 *  metres — chosen because a phone's GPS is 10–30 outdoors and worse inside — so
 *  the extra seconds the highest setting spends do not change the answer, and
 *  they are spent with somebody standing at a door. */
@SuppressLint("MissingPermission")
private suspend fun currentFix(ctx: android.content.Context): Triple<Double, Double, Double>? =
    runCatching {
        val client = LocationServices.getFusedLocationProviderClient(ctx)
        val loc = client.getCurrentLocation(Priority.PRIORITY_BALANCED_POWER_ACCURACY, null).await()
            ?: client.lastLocation.await()
            ?: return null
        Triple(loc.latitude, loc.longitude, loc.accuracy.toDouble())
    }.getOrNull()

@Composable
private fun Trend(label: String, value: String, modifier: Modifier = Modifier) {
    val c = KeelTheme.colors
    Column(
        modifier.glass(c, RoundedCornerShape(18.dp)).padding(12.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(label, style = MaterialTheme.typography.labelMedium, color = c.muted)
        Text(value, style = MaterialTheme.typography.titleLarge, color = c.ink)
    }
}

@Composable
private fun Line(label: String, value: String, tone: Color? = null) {
    val c = KeelTheme.colors
    Row(
        Modifier.fillMaxWidth().padding(vertical = 7.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            label,
            Modifier.weight(1f),
            style = MaterialTheme.typography.bodyMedium,
            color = c.inkSoft,
        )
        Text(value, style = MaterialTheme.typography.titleMedium, color = tone ?: c.ink)
    }
}

/** The days, padded so the first one lands under its weekday.
 *
 *  ⚠️ **Monday first.** The roster is written a week at a time and a
 *  Sunday-first grid puts the end of the week at its start — the panel's
 *  calendar reads Monday first for the same reason, and two calendars that
 *  disagree about where a week begins are two calendars nobody trusts. */
@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun Calendar(days: List<StaffDay>) {
    if (days.isEmpty()) return
    val lead = remember(days.first().date) { leadingBlanks(days.first().date) }
    FlowRow(
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        repeat(lead) { Box(Modifier.width(40.dp).height(44.dp)) }
        days.forEach { DayCell(it) }
    }
}

/** How many empty cells go before the first day. ⚠️ Computed from the date
 *  string rather than a parsed instant: the server sends "YYYY-MM-DD" for
 *  exactly this reason, and turning it into a timestamp is how a month starts on
 *  the wrong column for everybody east of Greenwich. */
internal fun leadingBlanks(firstDate: String): Int {
    val date = runCatching { java.time.LocalDate.parse(firstDate) }.getOrNull() ?: return 0
    // ISO: Monday is 1. Monday-first means Sunday is the seventh column.
    return date.dayOfWeek.value - 1
}

@Composable
private fun DayCell(day: StaffDay) {
    val c = KeelTheme.colors
    val colour = StatusColor[day.status] ?: c.muted
    val plain = day.status == "off" || day.status == "upcoming"
    Column(
        Modifier
            // ⚠️ Seven per row on the narrowest phone this is installed on. A
            // percentage width would reflow to six on a small screen and the
            // calendar would stop being a calendar.
            .width(40.dp)
            .height(44.dp)
            // ⚠️ A wash rather than a fill: a grid of saturated squares is a
            // pattern nobody reads, and the number on top has to stay legible in
            // both schemes. The border carries the colour at full strength.
            .background(
                if (plain) Color.Transparent else colour.copy(alpha = 0.13f),
                RoundedCornerShape(10.dp),
            )
            .border(1.dp, if (plain) c.line else colour, RoundedCornerShape(10.dp)),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text(
            day.date.takeLast(2).trimStart('0'),
            style = MaterialTheme.typography.bodyMedium,
            color = c.ink,
        )
        if (day.worked > 0) {
            Text(
                (Math.round(day.worked / 60.0 * 10) / 10.0).toString(),
                style = MaterialTheme.typography.labelSmall,
                color = c.muted,
            )
        }
    }
}

/** The legend — because eight colours is more than anybody memorises, and only
 *  the ones this period actually contains. */
@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun Legend(days: List<StaffDay>) {
    val c = KeelTheme.colors
    val kinds = remember(days) { days.map { it.status }.distinct() }
    FlowRow(
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        kinds.forEach { st ->
            Row(
                horizontalArrangement = Arrangement.spacedBy(6.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(Modifier.size(8.dp).background(StatusColor[st] ?: c.muted, CircleShape))
                Text(
                    t.profile.status[st] ?: st,
                    style = MaterialTheme.typography.labelMedium,
                    color = c.muted,
                )
            }
        }
    }
}
