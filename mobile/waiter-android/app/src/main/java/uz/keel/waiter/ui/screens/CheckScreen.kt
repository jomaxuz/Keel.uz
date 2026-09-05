package uz.keel.waiter.ui.screens

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.ArrowBackIosNew
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.ChevronRight
import androidx.compose.material.icons.rounded.Description
import androidx.compose.material.icons.rounded.GridView
import androidx.compose.material.icons.rounded.MoreVert
import androidx.compose.material.icons.rounded.Print
import androidx.compose.material.icons.rounded.Send
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshots.SnapshotStateMap
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json
import kotlinx.coroutines.sync.withLock
import uz.keel.waiter.LocalPrefs
import uz.keel.waiter.data.AddLineRequest
import uz.keel.waiter.data.ApiError
import uz.keel.waiter.data.Check
import uz.keel.waiter.data.CheckLine
import uz.keel.waiter.data.KeelApi
import uz.keel.waiter.data.MenuGroup
import uz.keel.waiter.data.LineQtyPayload
import uz.keel.waiter.data.MenuItem
import uz.keel.waiter.data.OpKind
import uz.keel.waiter.data.Outbox
import uz.keel.waiter.data.ServedPayload
import uz.keel.waiter.data.VoidPayload
import uz.keel.waiter.i18n.timeAgo
import uz.keel.waiter.t
import uz.keel.design.*
import uz.keel.waiter.ui.components.MenuView

// One table's check: what is on it, and how a dish gets added.
//
// ⚠️ **Adding is not ordering.** Lines land unfired, the waiter reads the table
// back, corrects what they misheard, and *then* sends it. That is the whole
// reason a till is faster than shouting through a hatch, and it is the rule the
// server already enforces — repeated on the screen because a phone that sent on
// every tap would put the rule out of reach.

/** How a part reads: a fraction, never a percentage. "50%" is a discount; "1/2"
 *  is half a loaf, and the two are read by the same people. */
fun portionText(percent: Int): String = when (percent) {
    25 -> "1/4"
    33 -> "1/3"
    50 -> "1/2"
    75 -> "3/4"
    else -> "$percent%"
}

/** How long taps are gathered before they go.
 *
 *  ⚠️ Short enough that a single tap still feels immediate — the count has
 *  already moved, so this is only the network — and long enough to catch the
 *  second, third and fourth of a run. Somebody adding four of something taps
 *  them in well under a second. */
private const val GATHER_MS = 180L

@Composable
fun CheckScreen(
    api: KeelApi,
    outbox: Outbox,
    checkId: String,
    branchId: String,
    bottomInset: PaddingValues,
    onBack: () -> Unit,
) {
    val c = KeelTheme.colors
    val prefs = LocalPrefs.current
    val scope = rememberCoroutineScope()
    val notice = LocalNotice.current

    var check by remember { mutableStateOf<Check?>(null) }
    var groups by remember { mutableStateOf<List<MenuGroup>>(emptyList()) }
    var others by remember { mutableStateOf<List<Check>>(emptyList()) }
    /** Dishes the branch has run out of today.
     *
     *  ⚠️ **It rides along with the check poll, and that is the only place it
     *  can come from.** The menu is fetched once when the table opens; a dish
     *  that runs out afterwards — tapped on another phone, stopped by the
     *  kitchen, past its batch for today — stayed pressable for the rest of the
     *  evening. The server refuses it either way, but a waiter finds that out
     *  after promising it to the table. */
    var soldOut by remember { mutableStateOf<Set<String>>(emptySet()) }
    var error by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var tab by remember { mutableStateOf("check") }
    var editing by remember { mutableStateOf<CheckLine?>(null) }
    var job by remember { mutableStateOf<String?>(null) }
    var portionFor by remember { mutableStateOf<MenuItem?>(null) }
    var warned by remember { mutableStateOf(false) }

    val failedLoad = t.floor.failedLoad
    val failedAdd = t.check.failedAdd
    val queuedNote = t.outbox.queued
    val failedFire = t.check.failedFire
    val lineFailed = t.line.failed
    val firedOnly = t.check.firedOnly

    /** ⚠️ **Taps are queued, not raced.** Every tap used to be a request that
     *  disabled the screen, so a waiter adding four coffees in two seconds got
     *  four requests racing to write the same check — the last reply winning and
     *  the count jumping backwards. A mutex serialises them, and `busy` is not
     *  set for adding: a disabled menu *is* the freeze people report. */
    val queue = remember { Mutex() }
    val buffer = remember { mutableListOf<AddLineRequest>() }
    val pending: SnapshotStateMap<String, Int> = remember { mutableStateMapOf() }
    var flushJob by remember { mutableStateOf<Job?>(null) }

    suspend fun load() {
        try {
            check = api.check(checkId)
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else failedLoad
        }
    }

    LaunchedEffect(checkId) { load() }

    // ⚠️ **Reloaded the moment the queue empties.** While anything is held, the
    // dishes are counted from this screen's own buffer; once the server has them
    // it is the one telling the truth, and two sources counting the same coffee
    // is a check that reads double. `queued` falling to zero is the handover.
    val queued by outbox.pending.collectAsState()
    LaunchedEffect(queued) {
        if (queued == 0 && pending.isNotEmpty()) {
            pending.clear()
            load()
        }
    }

    // ⚠️ Loaded beside the check rather than on the way into the menu tab: the
    // wait belongs to opening the table, when somebody is already standing at
    // it, not to the moment a guest has just said what they want.
    LaunchedEffect(branchId) {
        groups = runCatching { api.menu(branchId) }.getOrDefault(emptyList())
    }
    suspend fun loadRoom() {
        runCatching { api.checks() }.onSuccess { r ->
            others = r.checks.filter { it.id != checkId }
            soldOut = r.soldOut.toSet()
        }
    }
    LaunchedEffect(checkId) { loadRoom() }

    /** Send what is buffered. */
    fun flushAdds(): Job {
        flushJob?.cancel()
        flushJob = null
        val lines = buffer.toList()
        buffer.clear()
        if (lines.isEmpty()) return scope.launch { }
        return scope.launch {
            queue.withLock {
                // ⚠️ **Through the outbox, so a dropped wifi does not lose the
                // tap.** A restaurant's connection goes for thirty seconds
                // several times a week; every dish tapped in those seconds used
                // to come back as "qo'shib bo'lmadi" and be gone, and the waiter
                // retyped an order they had already taken, at the table.
                val payload = Json.encodeToString(
                    ListSerializer(AddLineRequest.serializer()), lines,
                )
                val res = outbox.send(checkId, OpKind.AddLines, payload) { opId ->
                    api.addLines(checkId, lines, opId)
                }
                // ⚠️ **The queued case keeps its count, and that is the whole
                // reason this is three outcomes rather than two.** Cleared, the
                // dishes vanish off a screen that is about to send them — so the
                // waiter taps them again and the guest is charged twice, which is
                // precisely the failure the queue was added to prevent.
                when (res) {
                    is Outbox.Outcome.Sent -> {
                        check = res.value
                        error = ""
                        drop(pending, lines)
                    }
                    is Outbox.Outcome.Refused -> {
                        // The server's own words: "lag'mon bugun tugadi" is an
                        // answer a waiter can take back to the table. Refused
                        // means the dish really is not on the check.
                        error = res.error.message
                        drop(pending, lines)
                        // ⚠️ **And the stop list is re-read here, because this is
                        // the moment it is known to have moved.** Otherwise the
                        // tile the waiter was just refused stays pressable, and
                        // they press it again.
                        loadRoom()
                    }
                    Outbox.Outcome.Queued -> error = queuedNote
                }
            }
        }
    }

    // ⚠️ **Leaving the screen sends what is buffered, and lets go of the table.**
    // A waiter who taps a dish and immediately walks back to the room would
    // otherwise lose it — the gathering window is short, but "short" is not
    // "never", and a dish that silently did not happen is the worst outcome this
    // file can produce.
    //
    // ⚠️ The hold is a courtesy: it expires by itself after a couple of minutes,
    // which is what makes it safe. Releasing is what stops a cashier being told
    // somebody is working on a table nobody is standing at.
    DisposableEffect(checkId) {
        onDispose {
            val lines = buffer.toList()
            buffer.clear()
            // A separate scope: this one is cancelled with the screen, and the
            // dish must still reach the server.
            CoroutineScope(kotlinx.coroutines.Dispatchers.IO).launch {
                if (lines.isNotEmpty()) runCatching { api.addLines(checkId, lines) }
                runCatching { api.releaseCheck(checkId) }
            }
        }
    }

    val lines = check?.lines?.filter { it.voided == null } ?: emptyList()
    val unfired = lines.count { !it.fired }

    /** How many of each dish are already on this check.
     *
     *  ⚠️ **The menu has to say what has already been added.** Tapping a tile
     *  four times is how a waiter enters four coffees, and with nothing counting
     *  back at them the only way to know whether the third tap registered is to
     *  switch tabs and read the check. That is the moment somebody taps again to
     *  be sure — and the guest is charged for five.
     *
     *  Summed across lines rather than read off one: the server merges a repeat
     *  into an unfired line but starts a new one once the kitchen has seen it. */
    val onCheck = remember(lines, pending.toMap()) {
        val m = HashMap<String, Int>()
        for (l in lines) {
            val id = l.menuItemId ?: continue
            m[id] = (m[id] ?: 0) + l.qty
        }
        // ⚠️ Plus the taps that have not been sent yet. Without this the number
        // beside a dish is a fact about the network rather than about the check,
        // and it is the only thing on screen that tells a waiter their tap landed.
        for ((id, n) in pending) m[id] = (m[id] ?: 0) + n
        m
    }

    fun add(item: MenuItem, portion: Int? = null) {
        // ⚠️ **Asked, not assumed.** Half a loaf is an ordinary sale and a whole
        // one is the common case; a phone that always added a whole one would
        // leave the waiter correcting it at the till — the walk this removes.
        if (portion == null && !item.portions.isNullOrEmpty()) {
            portionFor = item
            return
        }
        // Off the wire for a whole one: that is what every line was before parts
        // existed, and the server stores it as absent.
        val part = if (portion != null && portion != 100) portion else null
        // ⚠️ Merged by dish **and** by part: two halves and a whole loaf are two
        // different lines, and summing them would sell one and a half of
        // something nobody ordered.
        val same = buffer.indexOfFirst { it.menuItemId == item.id && it.portion == part }
        if (same >= 0) buffer[same] = buffer[same].copy(qty = buffer[same].qty + 1)
        else buffer.add(AddLineRequest(item.id, 1, part))
        pending[item.id] = (pending[item.id] ?: 0) + 1
        flushJob?.cancel()
        flushJob = scope.launch { delay(GATHER_MS); flushAdds() }
    }

    /** Take one off, from the menu side.
     *
     *  ⚠️ **Only from a line the kitchen has not seen.** Once a line is fired the
     *  ticket at the pass names a quantity, and quietly lowering it here would
     *  leave the paper and the screen disagreeing about one dish.
     *
     *  ⚠️ The **newest** unfired line is the one reduced: the tap being undone is
     *  almost always the last one, and reducing the oldest would take away a dish
     *  somebody added deliberately ten minutes ago. */
    fun removeOne(item: MenuItem) {
        val line = lines.lastOrNull { it.menuItemId == item.id && !it.fired }
        if (line == null) {
            // Said rather than silently ignored: pressing minus and having
            // nothing happen is how somebody concludes the screen is stuck.
            error = firedOnly
            return
        }
        scope.launch {
            queue.withLock {
                val res = if (line.qty > 1) {
                    outbox.send(
                        checkId, OpKind.LineQty,
                        Json.encodeToString(LineQtyPayload(line.lineId, line.qty - 1)),
                    ) { api.lineQty(checkId, line.lineId, line.qty - 1) }
                } else {
                    outbox.send(
                        checkId, OpKind.VoidLine,
                        Json.encodeToString(VoidPayload(line.lineId)),
                    ) { api.voidLine(checkId, line.lineId) }
                }
                when (res) {
                    is Outbox.Outcome.Sent -> { check = res.value; error = "" }
                    is Outbox.Outcome.Refused -> error = res.error.message
                    Outbox.Outcome.Queued -> error = queuedNote
                }
            }
        }
    }

    fun changeQty(line: CheckLine, next: Int) {
        scope.launch {
            queue.withLock {
                val res = if (next > 0) {
                    outbox.send(
                        checkId, OpKind.LineQty,
                        Json.encodeToString(LineQtyPayload(line.lineId, next)),
                    ) { api.lineQty(checkId, line.lineId, next) }
                } else {
                    outbox.send(
                        checkId, OpKind.VoidLine,
                        Json.encodeToString(VoidPayload(line.lineId)),
                    ) { api.voidLine(checkId, line.lineId) }
                }
                when (res) {
                    is Outbox.Outcome.Sent -> { check = res.value; error = "" }
                    is Outbox.Outcome.Refused -> error = res.error.message
                    Outbox.Outcome.Queued -> error = queuedNote
                }
            }
        }
    }

    /** "The guest has it." ⚠️ Optimism would be wrong here: the answer is the
     *  whole check, and half the room is looking at the same table on another
     *  screen. The reply replaces it. */
    fun toggleServed(l: CheckLine) {
        val cur = check ?: return
        busy = true
        scope.launch {
            val served = l.servedAt == null
            val res = outbox.send(
                cur.id, OpKind.LineServed,
                Json.encodeToString(ServedPayload(l.lineId, served)),
            ) { api.lineServed(cur.id, l.lineId, served) }
            when (res) {
                is Outbox.Outcome.Sent -> { check = res.value; error = "" }
                is Outbox.Outcome.Refused -> error = res.error.message
                Outbox.Outcome.Queued -> error = queuedNote
            }
            busy = false
        }
    }

    fun fire() {
        busy = true
        scope.launch {
            // ⚠️ **Whatever is still in the buffer goes first, and this is the
            // one place gathering taps could have cost something real.** A waiter
            // can add a dish and press "send" inside the gathering window; the
            // kitchen would then get the ticket without it. The flush runs on the
            // same lock this then takes, so the order is the order of the taps.
            flushAdds().join()
            // ⚠️ Queued like the rest, and that is a real decision rather than
            // consistency: a ticket the kitchen never received is the worst of
            // these failures, and it was the one that used to vanish with a
            // sentence the waiter had no way to act on.
            val res = queue.withLock {
                outbox.send(checkId, OpKind.Fire, "{}") { api.fire(checkId) }
            }
            when (res) {
                is Outbox.Outcome.Sent -> { check = res.value; error = "" }
                is Outbox.Outcome.Refused -> error = res.error.message
                Outbox.Outcome.Queued -> error = queuedNote
            }
            busy = false
        }
    }

    val billPrinted = t.bill.printed
    val billNotQueued = t.bill.notQueued
    val billNotQueuedHint = t.bill.notQueuedHint
    val billTillOff = t.bill.tillOff
    val billTillOffHint = t.bill.tillOffHint
    val billFailed = t.bill.failed

    /** Print the bill for this table.
     *
     *  ⚠️ **Only once everything has been sent.** A bill printed while a dish is
     *  still unfired is a bill that is about to be wrong — and the guest has
     *  already been handed it. The button is hidden rather than disabled: a
     *  greyed-out control invites "why can't I", and the answer is one line up. */
    fun printBill() {
        busy = true
        scope.launch {
            flushAdds().join()
            try {
                val res = queue.withLock { api.print(checkId, "precheck") }
                check = res.check
                // ⚠️ **Said in a sheet, not in small text under the buttons.**
                // `queued: 0` changes what the waiter does next — they walk to
                // the till — and the one message that changes the next action was
                // the one nobody saw.
                //
                // ⚠️ **Two different zeros.** "No printer configured here" and
                // "the till app is switched off" both come back as `queued: 0`,
                // and the next move differs entirely: one is somebody else's
                // settings problem, the other is walking over and pressing a
                // power button. Saying one sentence for both sent people to the
                // wrong place.
                notice.value = when {
                    res.queued > 0 -> Note(NoticeKind.Ok, billPrinted)
                    res.tillOff -> Note(NoticeKind.Warn, billTillOff, billTillOffHint)
                    else -> Note(NoticeKind.Warn, billNotQueued, billNotQueuedHint)
                }
            } catch (e: Throwable) {
                notice.value = Note(NoticeKind.Error, billFailed, (e as? ApiError)?.message)
            } finally { busy = false }
        }
    }

    // ⚠️ **Said when the table opens, not when a button is refused.** The server
    // sends the holder's name for exactly this — and only while the hold is
    // fresh, so a name here means somebody is on it *now*. A table that opens and
    // then refuses every press reads as a broken till; one that says "Dilnoza is
    // on this one" reads as a colleague.
    val heldBy = check?.heldBy
    val heldTitle = t.check.heldTitle
    val heldBody = t.check.heldBody
    LaunchedEffect(heldBy) {
        if (heldBy.isNullOrBlank() || warned) return@LaunchedEffect
        warned = true
        notice.value = Note(NoticeKind.Warn, heldTitle(heldBy), heldBody)
    }

    val cur = check
    if (cur == null) {
        Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            if (error.isNotEmpty()) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error, color = c.danger, textAlign = TextAlign.Center)
                    Text(
                        t.check.back, color = c.accent,
                        modifier = Modifier.padding(12.dp).clickable(onClick = onBack),
                    )
                }
            } else CircularProgressIndicator(color = c.accent)
        }
        return
    }

    val bottomPad = bottomInset.calculateBottomPadding() +
        if (unfired > 0 || (tab == "check" && lines.isNotEmpty())) 96.dp else 24.dp

    Box(Modifier.fillMaxSize()) {
        Column(Modifier.fillMaxSize()) {
            ScreenHeader(
                title = cur.tableNumber?.let { t.check.table(it) } ?: cur.number,
                leading = { GlassIconButton(Icons.Rounded.ArrowBackIosNew, onClick = onBack) },
                trailing = {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        Money(cur.total, style = MaterialTheme.typography.titleLarge)
                        // ⚠️ Behind one button rather than four in the header:
                        // these are things a table does occasionally, and four
                        // controls above the check would crowd out the two it
                        // does constantly.
                        GlassIconButton(Icons.Rounded.MoreVert) { job = "menu" }
                    }
                },
            )

            Row(
                Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 2.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Chip("${t.check.tab} · ${lines.size}", tab == "check", icon = Icons.Rounded.Description) { tab = "check" }
                Chip(t.check.menu, tab == "menu", icon = Icons.Rounded.GridView) { tab = "menu" }
            }

            if (error.isNotEmpty()) {
                Text(
                    error, color = c.danger, textAlign = TextAlign.Center,
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp),
                )
            }

            if (tab == "check") {
                LazyColumn(
                    Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(16.dp, 8.dp, 16.dp, bottomPad),
                    verticalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    items(lines, key = { it.lineId }) { l ->
                        CheckRow(
                            line = l, busy = busy,
                            onOpen = { editing = l },
                            onServed = { toggleServed(l) },
                            onMinus = { changeQty(l, l.qty - 1) },
                            onPlus = { changeQty(l, l.qty + 1) },
                        )
                    }
                    if (lines.isEmpty()) {
                        item {
                            Text(
                                t.check.empty, color = c.muted, textAlign = TextAlign.Center,
                                modifier = Modifier.fillMaxWidth().padding(24.dp),
                            )
                        }
                    }
                }
            } else {
                MenuPane(
                    groups = groups,
                    onCheck = onCheck,
                    soldOut = soldOut,
                    view = prefs.view.value,
                    onView = { prefs.setView(it) },
                    lang = prefs.lang.value.code,
                    uploadsBase = api.uploadsBase,
                    bottomPad = bottomPad,
                    onAdd = { add(it) },
                    onRemove = { removeOne(it) },
                )
            }
        }

        // ⚠️ **Above Android's navigation bar, not under it.** With three-button
        // navigation a control placed at the bottom of the *layout* lands beneath
        // the bar — so the tap meant to send an order to the kitchen presses Back
        // instead, and the waiter is returned to the room with the food unsent.
        Column(
            Modifier.align(Alignment.BottomCenter)
                .padding(bottom = bottomInset.calculateBottomPadding() + 8.dp)
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            // ⚠️ **The bill belongs on the check tab only.** Asking for it is the
            // end of the waiter's job — the table says "hisob" and somebody goes
            // to fetch it. Hidden while the menu is open: nobody prints a bill in
            // the middle of taking an order.
            AnimatedVisibility(
                visible = tab == "check" && lines.isNotEmpty() && unfired == 0,
                enter = fadeIn() + slideInVertically { it },
                exit = fadeOut() + slideOutVertically { it },
            ) {
                Row(
                    Modifier.fillMaxWidth()
                        .softShadow(RoundedCornerShape(18.dp), elevation = 4.dp, dark = c.dark)
                        .glass(c, RoundedCornerShape(18.dp), strong = true)
                        .clickable(enabled = !busy) { printBill() }
                        .padding(horizontal = 18.dp, vertical = 16.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    Icon(Icons.Rounded.Print, null, tint = c.ink, modifier = Modifier.size(19.dp))
                    Text(t.bill.print, Modifier.weight(1f), color = c.ink, style = MaterialTheme.typography.bodyLarge)
                    Money(cur.total)
                }
            }

            // ⚠️ Shown only when there is something the kitchen has not seen. A
            // permanent button invites being pressed on a table that is already
            // cooking, and the second press is a ticket nobody asked for.
            AnimatedVisibility(
                visible = unfired > 0,
                enter = fadeIn() + slideInVertically { it },
                exit = fadeOut() + slideOutVertically { it },
            ) {
                PrimaryButton(
                    t.check.fire(unfired), icon = Icons.Rounded.Send,
                    enabled = !busy, busy = busy,
                ) { fire() }
            }
        }
    }

    // ⚠️ **One question, four buttons, no keyboard.** This opens with a plate in
    // the waiter's other hand: the whole portion is first because it is nearly
    // every sale, and the parts are the ones the restaurant said this dish can be
    // cut into.
    portionFor?.let { item ->
        Dialog(onDismissRequest = { portionFor = null }) {
            Column(
                Modifier.widthIn(max = 380.dp).glassSheet(c, RoundedCornerShape(26.dp)).padding(20.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Text(item.name, style = MaterialTheme.typography.titleMedium, color = c.ink)
                Text(t.check.portionAsk, style = MaterialTheme.typography.bodyMedium, color = c.muted)
                val parts = listOf(100) + (item.portions ?: emptyList())
                androidx.compose.foundation.layout.FlowRow(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    parts.forEach { p ->
                        // ⚠️ 64dp tall: pressed with a thumb, standing, holding
                        // something else.
                        Box(
                            Modifier.widthIn(min = 96.dp).size(width = 96.dp, height = 64.dp)
                                .glass(c, RoundedCornerShape(16.dp))
                                .clickable { portionFor = null; add(item, p) },
                            contentAlignment = Alignment.Center,
                        ) {
                            Text(
                                if (p == 100) t.check.portionWhole else portionText(p),
                                style = MaterialTheme.typography.titleMedium, color = c.ink,
                            )
                        }
                    }
                }
                Text(
                    t.check.portionCancel, color = c.muted, textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth().padding(8.dp).clickable { portionFor = null },
                )
            }
        }
    }

    editing?.let { l ->
        LineDialog(api, checkId, l, onDone = { check = it }, onClose = { editing = null })
    }

    job?.let { j ->
        TableActions(
            api = api, check = cur, others = others, job = j,
            onJob = { job = it }, onDone = { check = it }, onClose = { job = null },
        )
    }
}

@Composable
private fun CheckRow(
    line: CheckLine,
    busy: Boolean,
    onOpen: () -> Unit,
    onServed: () -> Unit,
    onMinus: () -> Unit,
    onPlus: () -> Unit,
) {
    val c = KeelTheme.colors
    Row(
        Modifier.fillMaxWidth()
            .glass(c, RoundedCornerShape(18.dp))
            // ⚠️ The whole row opens the dialog rather than a small edit icon:
            // this is used with a thumb, walking, and a target the size of a
            // glyph is why somebody gives up and walks to the till.
            .clickable(onClick = onOpen)
            .padding(horizontal = 14.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(
                buildString {
                    // ⚠️ The line's own frozen name, not the menu's: it is what
                    // the guest agreed to, and a dish renamed at six o'clock must
                    // not rewrite a check opened at five.
                    line.portion?.let { append(portionText(it)).append(" · ") }
                    append(line.name)
                    if (line.qty > 1) append(" × ").append(line.qty)
                },
                style = MaterialTheme.typography.bodyLarge, color = c.ink,
                maxLines = 2, overflow = TextOverflow.Ellipsis,
            )
            // What the kitchen was told about this dish. Shown because it is the
            // half of the order a guest will check.
            line.comment?.takeIf { it.isNotBlank() }?.let {
                Text(it, style = MaterialTheme.typography.labelMedium, color = c.muted)
            }
            // ⚠️ An unfired line says so. The kitchen has not seen it, and that is
            // the one thing here a guest may be waiting on.
            if (!line.fired) {
                Text(t.check.pending, style = MaterialTheme.typography.labelMedium, color = c.accent)
            }
            // ⚠️ **Where the dish is, on the row.** The kitchen ticks each one as
            // it finishes, and this is where a waiter walking the room finds out
            // — and how long ago, because "two minutes" and "twenty minutes" send
            // them to two different places. Served wins over ready: once the guest
            // has it, when it was cooked stops being the question.
            when {
                line.servedAt != null -> Text(
                    "✓ ${t.check.servedAgo(timeAgo(line.servedAt, t.common.timeAgo))}",
                    style = MaterialTheme.typography.labelMedium, color = c.ready,
                )
                line.readyAt != null -> Text(
                    t.check.readyAgo(timeAgo(line.readyAt, t.common.timeAgo)),
                    style = MaterialTheme.typography.labelMedium, color = c.ready,
                )
            }
        }
        Money(line.sum)

        // ⚠️ **"The guest has it", one tap, on a fired line only.** A draft cannot
        // have been carried anywhere and the server refuses it; a button that
        // always says no is one people stop pressing. It sits before the chevron
        // because it is what this screen is opened for during service.
        if (line.fired) {
            Box(
                Modifier.size(40.dp)
                    .background(
                        if (line.servedAt != null) c.ready else c.glassStrong, CircleShape,
                    )
                    .clickable(enabled = !busy, onClick = onServed),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    Icons.Rounded.Check, null,
                    tint = if (line.servedAt != null) androidx.compose.ui.graphics.Color.White else c.muted,
                    modifier = Modifier.size(18.dp),
                )
            }
            Icon(Icons.Rounded.ChevronRight, null, tint = c.muted, modifier = Modifier.size(18.dp))
        } else {
            // ⚠️ A stepper only while the kitchen has not seen it. A fired line
            // goes through the dialog, where a reason is asked for — the paper at
            // the pass names a quantity, and changing it with two taps and no
            // record is the difference between correcting a typo and writing off
            // cooked food.
            GlassStepper(line.qty, enabled = !busy, onMinus = onMinus, onPlus = onPlus)
        }
    }
}

/** Take a sent-or-refused batch back out of the optimistic count.
 *
 *  ⚠️ A batch that stayed "pending" after the server refused it would show a
 *  dish on the check that the kitchen will never see — the failure this screen
 *  exists to prevent, and the mirror of clearing one that is merely queued. */
private fun drop(
    pending: androidx.compose.runtime.snapshots.SnapshotStateMap<String, Int>,
    lines: List<AddLineRequest>,
) {
    for (l in lines) {
        val left = (pending[l.menuItemId] ?: 0) - l.qty
        if (left > 0) pending[l.menuItemId] = left else pending.remove(l.menuItemId)
    }
}
