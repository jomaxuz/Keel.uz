package uz.keel.waiter.ui.screens

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import uz.keel.waiter.data.ApiError
import uz.keel.waiter.data.Check
import uz.keel.waiter.data.FloorTable
import uz.keel.waiter.data.KeelApi
import uz.keel.design.money
import uz.keel.waiter.t
import uz.keel.design.*

// The room.
//
// ⚠️ **No branch picker, and that is the design.** A monoblock is bound to a
// branch because a machine stands somewhere; a person carries their own, and
// `staff.branchId` is what every till endpoint already reads. Asking a waiter
// which branch they are in is a question whose wrong answer is silent — the
// wrong room's tables, and nothing to say so.

@Composable
fun FloorScreen(
    api: KeelApi,
    bottomInset: androidx.compose.foundation.layout.PaddingValues,
    /** Whether this employee is clocked in.
     *
     *  ⚠️ **A table is opened by somebody who is at work.** A check carries the
     *  waiter who opened it, and one opened outside a shift is a sale attributed
     *  to a person the roster says was not there — which surfaces at payroll, on
     *  the wrong day, as an argument. The server has its own rules; this is the
     *  half that stops the mistake being made at all. */
    shiftOpen: Boolean,
    onOpenCheck: (checkId: String, branchId: String) -> Unit,
) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()
    var tables by remember { mutableStateOf<List<FloorTable>?>(null) }
    var zones by remember { mutableStateOf<List<uz.keel.waiter.data.TableZone>>(emptyList()) }
    var zone by remember { mutableStateOf("") }
    var checks by remember { mutableStateOf<List<Check>>(emptyList()) }
    var branchName by remember { mutableStateOf("") }
    var branchId by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    var busyTable by remember { mutableStateOf("") }
    var refreshing by remember { mutableStateOf(false) }
    // ⚠️ Read here rather than inside `load`: these are composable reads, and
    // capturing them means the message follows a language change while the
    // screen is open — the dictionary is not a constant.
    val failedLoad = t.floor.failedLoad
    val failedOpen = t.floor.failedOpen
    val notice = LocalNotice.current
    val noShiftTitle = t.clock.noShiftTitle
    val noShiftBody = t.clock.noShiftBody

    suspend fun load() {
        try {
            val b = api.branch()
            val ch = api.checks()
            branchName = b.name
            branchId = b.id
            tables = b.booking.tables
            zones = b.booking.zones
            checks = ch.checks
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else failedLoad
            if (tables == null) tables = emptyList()
        }
    }

    LaunchedEffect(Unit) { load() }

    /** Which table each open check is sitting at.
     *
     *  ⚠️ Built once per change rather than searched per tile: a room has forty
     *  tables and this is the kind of scan that is invisible until somebody is
     *  scrolling on a four-year-old phone. */
    val byTable = remember(checks) {
        checks.filter { !it.tableId.isNullOrBlank() }.associateBy { it.tableId!! }
    }

    /** The tabs across the top: the room's own parts, plus the unnamed one.
     *
     *  ⚠️ **An unzoned table is not a table without a home.** A restaurant that
     *  has never split its room keeps every table under no zone at all — the
     *  ordinary state — so a strip built only from the zone list would be empty
     *  on exactly those branches, and a strip that hid unzoned tables would hide
     *  the whole room. The unnamed part appears only when something is in it.
     *
     *  ⚠️ And with no zones at all there are no tabs: one room needs no label,
     *  and a single tab reading "the room" is a control that answers nothing. */
    val unzonedLabel = t.floor.unzoned
    val parts = remember(zones, tables, unzonedLabel) {
        val used = (tables ?: emptyList()).map { it.zoneId ?: "" }.toSet()
        val list = zones.filter { used.contains(it.id) }.sortedBy { it.sort }
            .map { it.id to it.name }.toMutableList()
        if (used.contains("") && list.isNotEmpty()) list.add(0, "" to unzonedLabel)
        list
    }

    // ⚠️ The first tab is selected once the room is known, not on every render:
    // resetting the tab on each poll would send a waiter back to the first zone
    // every fifteen seconds while they were looking at the second.
    LaunchedEffect(parts) {
        if (parts.isNotEmpty() && parts.none { it.first == zone }) zone = parts.first().first
    }

    val shown = remember(tables, parts, zone) {
        if (parts.isEmpty()) tables ?: emptyList()
        else (tables ?: emptyList()).filter { (it.zoneId ?: "") == zone }
    }

    fun open(table: FloorTable) {
        // ⚠️ **Said in a sheet, not as a line under the room.** This changes what
        // the waiter does next — they walk to the Profile tab — and a message
        // that changes the next action is the one that has to interrupt.
        if (!shiftOpen) {
            notice.value = Note(NoticeKind.Warn, noShiftTitle, noShiftBody)
            return
        }
        // ⚠️ An occupied table is opened, not refused: the whole reason a waiter
        // taps a table that already has a check is to add to it.
        val existing = byTable[table.id]
        if (existing != null) {
            onOpenCheck(existing.id, branchId)
            return
        }
        busyTable = table.id
        scope.launch {
            try {
                val check = api.openCheck(table.id)
                onOpenCheck(check.id, branchId)
            } catch (e: Throwable) {
                error = if (e is ApiError) e.message else failedOpen
            } finally {
                busyTable = ""
            }
        }
    }

    if (tables == null) {
        Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            CircularProgressIndicator(color = c.accent)
        }
        return
    }

    Column(Modifier.fillMaxSize()) {
        ScreenHeader(
            title = branchName.ifEmpty { t.floor.title },
            subtitle = "${byTable.size} / ${tables!!.size}",
            trailing = {
                GlassIconButton(Icons.Rounded.Refresh) { scope.launch { load() } }
            },
        )

        if (error.isNotEmpty()) {
            Text(
                error, color = c.danger, textAlign = TextAlign.Center,
                style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
            )
        }

        // ⚠️ **A fixed height and no shrinking**, the same lesson the menu's
        // category strip taught: a row of chips inside a column collapses the
        // moment the list beside it grows, and it collapses hardest on the rooms
        // with the most tables.
        if (parts.isNotEmpty()) {
            LazyRow(
                Modifier.fillMaxWidth().height(54.dp),
                contentPadding = PaddingValues(horizontal = 16.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                items(parts, key = { it.first.ifEmpty { "none" } }) { (id, name) ->
                    Chip(name, id == zone) { zone = id }
                }
            }
        }

        PullToRefreshBox(
            isRefreshing = refreshing,
            // The room changes under a waiter while they are walking.
            onRefresh = { scope.launch { refreshing = true; load(); refreshing = false } },
            modifier = Modifier.fillMaxSize(),
        ) {
            LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 104.dp),
                contentPadding = PaddingValues(
                    start = 16.dp, end = 16.dp, top = 4.dp,
                    bottom = 24.dp + bottomInset.calculateBottomPadding(),
                ),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                items(shown, key = { it.id }) { tb ->
                    TableTile(tb, byTable[tb.id], busyTable == tb.id) { open(tb) }
                }
                if (shown.isEmpty()) {
                    item {
                        Text(
                            t.floor.empty, color = c.muted,
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun TableTile(table: FloorTable, check: Check?, busy: Boolean, onClick: () -> Unit) {
    val c = KeelTheme.colors
    val taken = check != null
    val lift by animateFloatAsState(if (taken) 1f else 0f, tween(220), label = "tableLift")

    Box(
        Modifier
            .aspectRatio(1f)
            .softShadow(RoundedCornerShape(24.dp), elevation = 1.dp + 2.dp * lift, dark = c.dark)
            .glass(c, RoundedCornerShape(24.dp), strong = taken)
            .then(
                if (taken) Modifier
                    .background(c.accentSoft, RoundedCornerShape(24.dp))
                    .border(1.5.dp, c.accent.copy(alpha = 0.6f), RoundedCornerShape(24.dp))
                else Modifier,
            )
            .clickable(enabled = !busy, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(3.dp),
        ) {
            Text(
                table.number,
                style = MaterialTheme.typography.headlineMedium.copy(fontWeight = FontWeight.Bold),
                color = c.ink,
            )
            // ⚠️ Not colour alone: an occupied table says what it owes. Green
            // against red is a distinction roughly one man in twelve cannot
            // make, and the lock screen already learned this.
            Text(
                if (check != null) money(check.total) else t.floor.free,
                style = MaterialTheme.typography.labelMedium,
                color = if (taken) c.accent else c.muted,
            )
        }
        // A table with food the kitchen has not seen yet — the one thing on this
        // screen a guest is actively waiting on.
        if (check != null && check.unfired > 0) {
            Box(
                Modifier.align(Alignment.TopEnd).padding(12.dp)
                    .size(9.dp).background(c.accent, CircleShape),
            )
        }
        // ⚠️ Cooked and standing under the lamp: the other half of the same
        // question, and the one that says "walk to the pass now".
        if (check != null && check.readyWaiting > 0) {
            Box(
                Modifier.align(Alignment.TopStart).padding(12.dp)
                    .size(9.dp).background(c.ready, CircleShape),
            )
        }
    }
}
