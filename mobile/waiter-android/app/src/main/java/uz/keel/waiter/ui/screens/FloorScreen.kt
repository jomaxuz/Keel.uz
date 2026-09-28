package uz.keel.waiter.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.CloudOff
import androidx.compose.material.icons.rounded.ErrorOutline
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.Schedule
import androidx.compose.material.icons.rounded.Restaurant
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.launch
import uz.keel.design.*
import uz.keel.waiter.data.ApiError
import uz.keel.waiter.data.Check
import uz.keel.waiter.data.FloorFilter
import uz.keel.waiter.data.FloorStore
import uz.keel.waiter.data.FloorTable
import uz.keel.waiter.data.KeelApi
import uz.keel.waiter.data.MenuCache
import uz.keel.waiter.t
import uz.keel.waiter.ui.components.CountChip
import uz.keel.waiter.ui.components.InlineBanner
import uz.keel.waiter.ui.components.Pill
import uz.keel.waiter.ui.components.PollWhileVisible
import uz.keel.waiter.ui.components.placeholder

// The room.
//
// ⚠️ **No branch picker, and that is the design.** A monoblock is bound to a
// branch because a machine stands somewhere; a person carries their own, and
// `staff.branchId` is what every till endpoint already reads. Asking a waiter
// which branch they are in is a question whose wrong answer is silent — the
// wrong room's tables, and nothing to say so.
//
// ⚠️ **Drawn from `FloorStore`, never from a spinner.** The room is kept by the
// process, so coming back from a check or another tab shows it at once and the
// refresh happens underneath. It used to be thrown away with the screen, and
// every return to the floor was a second of nothing — the freeze people meant.

/** How often the room is asked for while it is on screen. ⚠️ The kitchen ticks
 *  dishes ready and colleagues open tables; a room that only refreshed on a pull
 *  showed "ready" to nobody until somebody thought to pull. The till asks every
 *  twenty seconds; a waiter walking the floor needs it a little sooner. */
private const val POLL_MS = 15_000L

@Composable
fun FloorScreen(
    api: KeelApi,
    store: FloorStore,
    menu: MenuCache,
    /** Who is holding the phone — for the "mine" filter. */
    staffId: String,
    bottomInset: PaddingValues,
    /** Whether this employee is clocked in.
     *
     *  ⚠️ **A table is opened by somebody who is at work.** A check carries the
     *  waiter who opened it, and one opened outside a shift is a sale attributed
     *  to a person the roster says was not there — which surfaces at payroll, on
     *  the wrong day, as an argument. The server has its own rules; this is the
     *  half that stops the mistake being made at all. */
    shiftOpen: Boolean,
    onOpenCheck: (checkId: String, branchId: String, preview: Check?) -> Unit,
) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()
    val notice = LocalNotice.current
    var openError by remember { mutableStateOf("") }
    var busyTable by remember { mutableStateOf("") }
    var pulling by remember { mutableStateOf(false) }
    val failedLoad = t.floor.failedLoad
    val failedOpen = t.floor.failedOpen
    val noShiftTitle = t.clock.noShiftTitle
    val noShiftBody = t.clock.noShiftBody

    // ⚠️ **The menu is fetched here, while nobody is waiting for it.** The first
    // table of the evening used to open onto an empty menu tab — the moment a
    // guest has just said what they want.
    LaunchedEffect(Unit) {
        store.refresh()
        store.branch?.id?.let { menu.ensure(it) }
    }
    PollWhileVisible(POLL_MS) { store.refresh() }

    val branch = store.branch
    val branchId = branch?.id ?: ""
    val tables = branch?.booking?.tables ?: emptyList()
    val zones = branch?.booking?.zones ?: emptyList()
    val checks = store.checks

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
        val used = tables.map { it.zoneId ?: "" }.toSet()
        val list = zones.filter { used.contains(it.id) }.sortedBy { it.sort }
            .map { it.id to it.name }.toMutableList()
        if (used.contains("") && list.isNotEmpty()) list.add(0, "" to unzonedLabel)
        list
    }

    // ⚠️ The first zone is chosen once the room is known, and **kept** after
    // that — in the store, so it survives a visit to a check. Resetting it on
    // every poll, or on every return, sent a waiter back to the first zone while
    // they were working the second.
    LaunchedEffect(parts) {
        if (parts.isNotEmpty() && parts.none { it.first == store.zone }) store.zone = parts.first().first
    }

    val zone = store.zone
    val zoneTables = remember(tables, parts, zone) {
        if (parts.isEmpty()) tables else tables.filter { (it.zoneId ?: "") == zone }
    }

    // taken, free, ready, mine — in this zone, because that is what is on screen.
    val counts = remember(zoneTables, byTable, staffId) {
        var taken = 0
        var ready = 0
        var mine = 0
        for (tb in zoneTables) {
            val ch = byTable[tb.id] ?: continue
            taken++
            if (ch.readyWaiting > 0) ready++
            if (staffId.isNotEmpty() && ch.serverId == staffId) mine++
        }
        intArrayOf(taken, zoneTables.size - taken, ready, mine)
    }

    val filter = store.filter
    val shown = remember(zoneTables, byTable, filter, staffId) {
        zoneTables.filter { tb ->
            val ch = byTable[tb.id]
            when (filter) {
                FloorFilter.All -> true
                FloorFilter.Taken -> ch != null
                FloorFilter.Free -> ch == null
                FloorFilter.Ready -> (ch?.readyWaiting ?: 0) > 0
                FloorFilter.Mine -> ch != null && ch.serverId == staffId
            }
        }
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
        // taps a table that already has a check is to add to it. And it opens on
        // what the room already knows about it — every line is in this list.
        val existing = byTable[table.id]
        if (existing != null) {
            onOpenCheck(existing.id, branchId, existing)
            return
        }
        // One table at a time: a second tap while the first is opening is the
        // same guest, not a second one.
        if (busyTable.isNotEmpty()) return
        busyTable = table.id
        openError = ""
        scope.launch {
            try {
                val check = api.openCheck(table.id)
                store.upsert(check)
                onOpenCheck(check.id, branchId, check)
            } catch (e: Throwable) {
                openError = if (e is ApiError) e.message else failedOpen
            } finally {
                busyTable = ""
            }
        }
    }

    val mineTone = if (c.dark) Color(0xFF60A5FA) else Color(0xFF2563EB)

    Column(Modifier.fillMaxSize()) {
        // ---- Header: where, and the one button that re-asks ----
        Row(
            Modifier.fillMaxWidth().statusBarsPadding()
                .padding(start = 20.dp, end = 16.dp, top = 12.dp, bottom = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(Modifier.weight(1f)) {
                if (branch != null && branch.name.isNotBlank()) {
                    Text(
                        branch.name, color = c.muted, maxLines = 1,
                        style = MaterialTheme.typography.labelMedium,
                    )
                }
                Text(t.floor.title, color = c.ink, style = MaterialTheme.typography.headlineMedium)
            }
            Box(
                Modifier.size(42.dp).glass(c, CircleShape)
                    .clickable(enabled = !store.refreshing) { scope.launch { store.refresh(full = true) } },
                contentAlignment = Alignment.Center,
            ) {
                if (store.refreshing && store.loaded) {
                    CircularProgressIndicator(Modifier.size(18.dp), color = c.accent, strokeWidth = 2.dp)
                } else {
                    Icon(Icons.Rounded.Refresh, null, tint = c.inkSoft, modifier = Modifier.size(20.dp))
                }
            }
        }

        // ---- What is in the room, as filters that answer before they are pressed ----
        if (store.loaded && tables.isNotEmpty()) {
            LazyRow(
                Modifier.fillMaxWidth().height(52.dp),
                contentPadding = PaddingValues(horizontal = 16.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                item(key = "all") {
                    CountChip(t.look.all, zoneTables.size, filter == FloorFilter.All, c.muted) {
                        store.filter = FloorFilter.All
                    }
                }
                item(key = "taken") {
                    CountChip(t.look.taken, counts[0], filter == FloorFilter.Taken, c.accent) {
                        store.filter = if (filter == FloorFilter.Taken) FloorFilter.All else FloorFilter.Taken
                    }
                }
                item(key = "free") {
                    CountChip(t.look.free, counts[1], filter == FloorFilter.Free, c.inkSoft) {
                        store.filter = if (filter == FloorFilter.Free) FloorFilter.All else FloorFilter.Free
                    }
                }
                // ⚠️ Only when there is something ready: a permanent "Ready 0" is
                // a chip read forty times an evening to learn nothing.
                if (counts[2] > 0 || filter == FloorFilter.Ready) {
                    item(key = "ready") {
                        CountChip(t.look.ready, counts[2], filter == FloorFilter.Ready, c.ready) {
                            store.filter = if (filter == FloorFilter.Ready) FloorFilter.All else FloorFilter.Ready
                        }
                    }
                }
                if (counts[3] > 0 || filter == FloorFilter.Mine) {
                    item(key = "mine") {
                        CountChip(t.look.mine, counts[3], filter == FloorFilter.Mine, mineTone) {
                            store.filter = if (filter == FloorFilter.Mine) FloorFilter.All else FloorFilter.Mine
                        }
                    }
                }
            }
        }

        // ⚠️ **A fixed height and no shrinking**, the same lesson the menu's
        // category strip taught: a row of chips inside a column collapses the
        // moment the list beside it grows, and it collapses hardest on the rooms
        // with the most tables.
        if (parts.isNotEmpty()) {
            LazyRow(
                Modifier.fillMaxWidth().height(48.dp),
                contentPadding = PaddingValues(horizontal = 16.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                items(parts, key = { it.first.ifEmpty { "none" } }) { (id, name) ->
                    Chip(name, id == zone) { store.zone = id }
                }
            }
        }

        // ---- What the waiter should know before tapping ----
        Column(
            Modifier.fillMaxWidth().padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            // ⚠️ Said before the first tap rather than after it: the sheet still
            // interrupts a tap, but a waiter who can read this from the room does
            // not have to be refused first.
            if (!shiftOpen && store.loaded) {
                InlineBanner(noShiftBody, c.warn, Icons.Rounded.Schedule, Modifier.padding(top = 4.dp))
            }
            if (openError.isNotEmpty()) {
                InlineBanner(openError, c.danger, Icons.Rounded.ErrorOutline, onClose = { openError = "" })
            }
            // ⚠️ A failed poll over a known room is a warning, not an empty
            // screen: the tables on it were right a moment ago.
            if (store.failure != null && store.loaded) {
                InlineBanner(t.look.staleRoom, c.warn, Icons.Rounded.CloudOff)
            }
        }

        val gridPadding = PaddingValues(
            start = 16.dp, end = 16.dp, top = 8.dp,
            bottom = 24.dp + bottomInset.calculateBottomPadding(),
        )

        when {
            // Never loaded, and the last try failed: the one case with nothing
            // to show, so it gets the whole screen and a way to try again.
            !store.loaded && store.failure != null -> {
                val f = store.failure
                EmptyRoom(
                    text = if (f is ApiError) f.message else failedLoad,
                    tone = c.danger,
                    action = t.common.retry,
                ) { scope.launch { store.refresh(full = true) } }
            }

            // ⚠️ Shapes where the tables will be, not a spinner in the middle:
            // the room is about to look like this, and saying so is calmer than
            // a wheel.
            !store.loaded -> LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 100.dp),
                contentPadding = gridPadding,
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
                userScrollEnabled = false,
                modifier = Modifier.fillMaxSize(),
            ) {
                items(12) {
                    Box(Modifier.aspectRatio(TILE_RATIO).placeholder(c, RoundedCornerShape(22.dp)))
                }
            }

            else -> PullToRefreshBox(
                isRefreshing = pulling,
                // The room changes under a waiter while they are walking.
                onRefresh = {
                    scope.launch {
                        pulling = true
                        store.refresh(full = true)
                        pulling = false
                    }
                },
                modifier = Modifier.fillMaxSize(),
            ) {
                LazyVerticalGrid(
                    state = store.grid,
                    columns = GridCells.Adaptive(minSize = 100.dp),
                    contentPadding = gridPadding,
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                    verticalArrangement = Arrangement.spacedBy(10.dp),
                    modifier = Modifier.fillMaxSize(),
                ) {
                    items(shown, key = { it.id }) { tb ->
                        TableTile(tb, byTable[tb.id], busyTable == tb.id) { open(tb) }
                    }
                    if (shown.isEmpty()) {
                        item(span = { GridItemSpan(maxLineSpan) }) {
                            Text(
                                if (tables.isEmpty()) t.floor.empty else t.look.noMatch,
                                color = c.muted, textAlign = TextAlign.Center,
                                style = MaterialTheme.typography.bodyMedium,
                                modifier = Modifier.fillMaxWidth().padding(vertical = 40.dp),
                            )
                        }
                    }
                }
            }
        }
    }
}

/** Width over height of a table tile: a little taller than wide, so the number,
 *  the bill and one badge fit without the text shrinking on a 360dp phone. */
private const val TILE_RATIO = 0.92f

@Composable
private fun TableTile(table: FloorTable, check: Check?, busy: Boolean, onClick: () -> Unit) {
    val c = KeelTheme.colors
    val shape = RoundedCornerShape(22.dp)
    val taken = check != null
    val readyWaiting = check?.readyWaiting ?: 0
    val unfired = check?.unfired ?: 0

    // ⚠️ **The edge says the most urgent thing, and green outranks orange.** A
    // dish under the lamp goes cold; a draft only waits. Drawn as the border
    // rather than a pulse: an animation on every ready table is a frame drawn
    // forever, on the phones that can least afford it.
    val edge = when {
        readyWaiting > 0 -> c.ready
        taken -> c.accent.copy(alpha = 0.55f)
        else -> c.glassBorder
    }

    Box(
        Modifier
            .aspectRatio(TILE_RATIO)
            .clip(shape)
            .then(
                if (taken) {
                    Modifier
                        .background(c.glassStrong, shape)
                        .background(
                            Brush.verticalGradient(listOf(c.accentSoft, c.accentSoft.copy(alpha = 0f))),
                            shape,
                        )
                } else {
                    Modifier.glass(c, shape)
                },
            )
            .border(if (readyWaiting > 0) 2.dp else 1.dp, edge, shape)
            .clickable(enabled = !busy, onClick = onClick)
            .padding(12.dp),
    ) {
        Column(Modifier.fillMaxSize()) {
            Row(verticalAlignment = Alignment.Top) {
                Text(
                    table.number,
                    Modifier.weight(1f),
                    style = MaterialTheme.typography.headlineMedium.copy(fontSize = 24.sp, lineHeight = 28.sp),
                    color = if (taken) c.ink else c.inkSoft,
                    maxLines = 1,
                )
                if (check != null) {
                    // How long the table has been sitting. The server computes
                    // it — the phone's clock is wrong as often as the till's.
                    if (check.openMin > 0) {
                        Text(
                            t.look.minutes(check.openMin),
                            color = c.muted,
                            style = MaterialTheme.typography.labelSmall.merge(MoneyStyle),
                            modifier = Modifier.padding(top = 4.dp),
                        )
                    }
                } else if (table.seats > 0) {
                    Row(
                        Modifier.padding(top = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(2.dp),
                    ) {
                        Icon(Icons.Rounded.Person, null, tint = c.muted, modifier = Modifier.size(12.dp))
                        Text("${table.seats}", color = c.muted, style = MaterialTheme.typography.labelSmall)
                    }
                }
            }

            Spacer(Modifier.weight(1f))

            if (check != null) {
                // ⚠️ Not colour alone: an occupied table says what it owes. Green
                // against red is a distinction roughly one man in twelve cannot
                // make, and the lock screen already learned this.
                Text(
                    money(check.total),
                    color = c.ink, maxLines = 1,
                    style = MaterialTheme.typography.bodyMedium.merge(MoneyStyle)
                        .copy(fontWeight = FontWeight.SemiBold),
                )
                Spacer(Modifier.height(5.dp))
                // ⚠️ **One badge, the most urgent.** The tile is a hundred dp
                // wide; two badges side by side clip, and the one that clips is
                // whichever came second, not whichever matters less.
                when {
                    readyWaiting > 0 -> Pill(t.look.readyBadge(readyWaiting), c.ready, solid = true)
                    unfired > 0 -> Pill(t.look.newBadge(unfired), c.accent)
                    check.precheckAt != null -> Pill(t.look.billGiven, c.warn)
                    check.guests > 0 -> Pill(t.look.guests(check.guests), c.muted)
                    else -> Spacer(Modifier.height(18.dp))
                }
            } else {
                Text(t.floor.free, color = c.muted, style = MaterialTheme.typography.labelMedium)
            }
        }

        if (busy) {
            Box(
                Modifier.matchParentSize().background(c.bg.copy(alpha = 0.45f), shape),
                contentAlignment = Alignment.Center,
            ) {
                CircularProgressIndicator(Modifier.size(22.dp), color = c.accent, strokeWidth = 2.dp)
            }
        }
    }
}

@Composable
private fun EmptyRoom(text: String, tone: Color, action: String, onAction: () -> Unit) {
    val c = KeelTheme.colors
    Column(
        Modifier.fillMaxSize().padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(14.dp, Alignment.CenterVertically),
    ) {
        Box(Modifier.size(72.dp).glass(c, CircleShape), contentAlignment = Alignment.Center) {
            Icon(Icons.Rounded.Restaurant, null, tint = tone, modifier = Modifier.size(32.dp))
        }
        Text(text, color = c.ink, textAlign = TextAlign.Center, style = MaterialTheme.typography.bodyLarge)
        GhostButton(action, icon = Icons.Rounded.Refresh, onClick = onAction)
    }
}
