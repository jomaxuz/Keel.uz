package uz.keel.waiter.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.ArrowBackIosNew
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.CloudUpload
import androidx.compose.material.icons.rounded.ErrorOutline
import androidx.compose.material.icons.rounded.GridView
import androidx.compose.material.icons.rounded.Info
import androidx.compose.material.icons.rounded.MoreVert
import androidx.compose.material.icons.rounded.Print
import androidx.compose.material.icons.rounded.ReceiptLong
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.Send
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshots.SnapshotStateMap
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.NonCancellable
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json
import uz.keel.design.*
import uz.keel.waiter.LocalPrefs
import uz.keel.waiter.data.AddLineRequest
import uz.keel.waiter.data.ApiError
import uz.keel.waiter.data.Check
import uz.keel.waiter.data.CheckLine
import uz.keel.waiter.data.FloorStore
import uz.keel.waiter.data.KeelApi
import uz.keel.waiter.data.LineQtyPayload
import uz.keel.waiter.data.MenuCache
import uz.keel.waiter.data.MenuItem
import uz.keel.waiter.data.OpKind
import uz.keel.waiter.data.Outbox
import uz.keel.waiter.data.ServedPayload
import uz.keel.waiter.data.VoidPayload
import uz.keel.waiter.i18n.timeAgo
import uz.keel.waiter.t
import uz.keel.waiter.ui.components.InlineBanner
import uz.keel.waiter.ui.components.PollWhileVisible
import uz.keel.waiter.ui.components.SectionHeader
import uz.keel.waiter.ui.components.Segmented
import uz.keel.waiter.ui.components.placeholder
import uz.keel.waiter.ui.components.rememberTick

// One table's check: what is on it, and how a dish gets added.
//
// ⚠️ **Adding is not ordering.** Lines land unfired, the waiter reads the table
// back, corrects what they misheard, and *then* sends it. That is the whole
// reason a till is faster than shouting through a hatch, and it is the rule the
// server already enforces — repeated on the screen because a phone that sent on
// every tap would put the rule out of reach.
//
// ⚠️ **Every tap answers on the frame it is made.** The count moves, the
// stepper moves, "served" turns green — and the server's reply replaces all of
// it when it comes. That is not optimism about the sale (price, stop list and
// refusals are still the server's, and a refusal puts the number back with the
// server's own words); it is honesty about the tap. A stepper that waited for a
// round trip before it moved was the "cannot press it quickly" people reported,
// and three quick presses used to send the same stale "2" three times.

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

/** How long a stepper waits for the next press before it sends the number.
 *
 *  ⚠️ **The quantity is absolute, so only the last press matters.** "3" then
 *  "4" then "5" is one request for 5, not three requests racing each other. */
private const val QTY_SETTLE_MS = 350L

/** How often an open check is re-read while it is on screen. ⚠️ The kitchen
 *  ticks dishes ready while the waiter is standing at the table, and "ready two
 *  minutes ago" is the line they are waiting to see. */
private const val POLL_MS = 10_000L

private enum class Tone { Error, Info }

private data class Banner(val text: String, val tone: Tone)

@Composable
fun CheckScreen(
    api: KeelApi,
    outbox: Outbox,
    floor: FloorStore,
    menuCache: MenuCache,
    checkId: String,
    branchId: String,
    /** What the room already knew about this check, drawn while it is re-read. */
    preview: Check?,
    bottomInset: PaddingValues,
    onBack: () -> Unit,
) {
    val c = KeelTheme.colors
    val prefs = LocalPrefs.current
    val scope = rememberCoroutineScope()
    val notice = LocalNotice.current
    val tick = rememberTick()
    val density = LocalDensity.current

    var check by remember { mutableStateOf(preview) }
    var loadError by remember { mutableStateOf("") }
    var banner by remember { mutableStateOf<Banner?>(null) }
    var tab by remember { mutableStateOf("check") }
    var editing by remember { mutableStateOf<CheckLine?>(null) }
    var job by remember { mutableStateOf<String?>(null) }
    var portionFor by remember { mutableStateOf<MenuItem?>(null) }
    var warned by remember { mutableStateOf(false) }
    var firing by remember { mutableStateOf(false) }
    var printing by remember { mutableStateOf(false) }
    /** Whether anything from this screen went to the outbox — so the check is
     *  re-read when the outbox drains, and the phone's guesses hand over to the
     *  server's answer. */
    var hadQueued by remember { mutableStateOf(false) }
    var dockPx by remember { mutableIntStateOf(0) }
    /** Where the waiter was in the menu — kept while they flip to the check. */
    val menuState = remember { MenuPaneState() }

    val failedLoad = t.floor.failedLoad
    val queuedNote = t.outbox.queued
    val firedOnly = t.check.firedOnly

    /** ⚠️ **Taps are queued, not raced.** A mutex serialises every write to this
     *  check, in the order the taps were made, and nothing on the screen is
     *  disabled while it waits: a disabled menu *is* the freeze people report. */
    val queue = remember { Mutex() }
    val buffer = remember { mutableListOf<AddLineRequest>() }
    /** Dishes tapped and not yet on the server's check, by menu item. */
    val pending: SnapshotStateMap<String, Int> = remember { mutableStateMapOf() }
    /** A quantity pressed and not yet confirmed, by line. */
    val qtyOverride: SnapshotStateMap<String, Int> = remember { mutableStateMapOf() }
    /** "Served" pressed and not yet confirmed, by line. */
    val servedOverride: SnapshotStateMap<String, Boolean> = remember { mutableStateMapOf() }
    val qtyJobs = remember { HashMap<String, Job>() }
    var flushJob by remember { mutableStateOf<Job?>(null) }

    fun show(text: String, tone: Tone) {
        banner = Banner(text, tone)
    }

    /** A new answer from the server, here and in the room. */
    fun adopt(next: Check) {
        check = next
        floor.upsert(next)
    }

    /** Work that must finish even if the waiter leaves the screen.
     *
     *  ⚠️ **Not cancelled with the screen, and that is the point.** A dish sent
     *  as the waiter taps "back" used to be cancelled half-way: the outbox then
     *  tried to save it from inside a cancelled coroutine, which fails, so the
     *  dish was neither sent nor queued. The answer lands in state nobody is
     *  drawing any more, which costs nothing. */
    fun keep(block: suspend CoroutineScope.() -> Unit): Job = scope.launch(NonCancellable, block = block)

    suspend fun load() {
        try {
            adopt(api.check(checkId))
            loadError = ""
        } catch (e: CancellationException) {
            throw e
        } catch (e: Throwable) {
            // ⚠️ With a preview on screen a failed re-read is not an error page:
            // the lines are the room's, a few seconds old. Only a refusal (the
            // check was closed, or paid) is worth saying.
            if (check == null) loadError = if (e is ApiError) e.message else failedLoad
            else if (e is ApiError) show(e.message, Tone.Error)
        }
    }

    LaunchedEffect(checkId) { load() }
    // ⚠️ From memory when it is there — the floor fetched it already.
    LaunchedEffect(branchId) { menuCache.ensure(branchId) }
    // The stop list and the other open tables (for moving and merging) come
    // with the room. Asked for beside the check, not after it: a table opened
    // from a notification may never have seen the floor.
    LaunchedEffect(checkId) { floor.refresh() }

    // ⚠️ **Re-read quietly, and never over something unsent.** A reply to a poll
    // that raced a tap would put back the number the tap just changed. So the
    // poll takes the same lock as every write, and skips while anything the
    // waiter pressed is still on its way.
    PollWhileVisible(POLL_MS) {
        floor.refresh()
        val idle = buffer.isEmpty() && pending.isEmpty() && qtyOverride.isEmpty() &&
            servedOverride.isEmpty() && !firing && !printing
        if (idle && queue.tryLock()) {
            try {
                adopt(api.check(checkId))
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                // A missed poll is the next poll's job.
            } finally {
                queue.unlock()
            }
        }
    }

    // ⚠️ **Handed back to the server the moment the queue empties.** While
    // anything is held, the screen counts from its own guesses; once the server
    // has it all it is the one telling the truth, and two sources counting the
    // same coffee is a check that reads double.
    val queued by outbox.pending.collectAsState()
    LaunchedEffect(queued) {
        if (queued == 0 && hadQueued) {
            hadQueued = false
            pending.clear()
            qtyOverride.clear()
            servedOverride.clear()
            load()
        }
    }

    // Banners go by themselves: they report, they do not ask.
    LaunchedEffect(banner) {
        val b = banner ?: return@LaunchedEffect
        delay(if (b.tone == Tone.Info) 4_000L else 6_000L)
        if (banner == b) banner = null
    }

    /** Send what is buffered. */
    fun flushAdds(): Job {
        flushJob?.cancel()
        flushJob = null
        val batch = buffer.toList()
        buffer.clear()
        if (batch.isEmpty()) return scope.launch { }
        return keep {
            queue.withLock {
                // ⚠️ **Through the outbox, so a dropped wifi does not lose the
                // tap.** A restaurant's connection goes for thirty seconds several
                // times a week; every dish tapped in those seconds used to come
                // back as "qo'shib bo'lmadi" and be gone.
                val payload = Json.encodeToString(ListSerializer(AddLineRequest.serializer()), batch)
                val res = outbox.send(checkId, OpKind.AddLines, payload) { opId ->
                    api.addLines(checkId, batch, opId)
                }
                // ⚠️ **The queued case keeps its count**, and that is the whole
                // reason this is three outcomes rather than two. Cleared, the
                // dishes vanish off a screen that is about to send them — so the
                // waiter taps them again and the guest is charged twice.
                when (res) {
                    is Outbox.Outcome.Sent -> {
                        adopt(res.value)
                        drop(pending, batch)
                    }
                    is Outbox.Outcome.Refused -> {
                        // The server's own words: "lag'mon bugun tugadi" is an
                        // answer a waiter can take back to the table.
                        show(res.error.message, Tone.Error)
                        drop(pending, batch)
                        // ⚠️ The stop list is re-read here, because this is the
                        // moment it is known to have moved.
                        floor.refresh()
                    }
                    Outbox.Outcome.Queued -> {
                        hadQueued = true
                        show(queuedNote, Tone.Info)
                    }
                }
            }
        }
    }

    suspend fun sendQty(lineId: String, target: Int) {
        queue.withLock {
            val res = if (target > 0) {
                outbox.send(checkId, OpKind.LineQty, Json.encodeToString(LineQtyPayload(lineId, target))) {
                    api.lineQty(checkId, lineId, target)
                }
            } else {
                outbox.send(checkId, OpKind.VoidLine, Json.encodeToString(VoidPayload(lineId))) {
                    api.voidLine(checkId, lineId)
                }
            }
            when (res) {
                is Outbox.Outcome.Sent -> {
                    adopt(res.value)
                    // Only if no newer press has replaced it.
                    if (qtyOverride[lineId] == target) qtyOverride.remove(lineId)
                }
                is Outbox.Outcome.Refused -> {
                    show(res.error.message, Tone.Error)
                    if (qtyOverride[lineId] == target) qtyOverride.remove(lineId)
                }
                Outbox.Outcome.Queued -> {
                    hadQueued = true
                    show(queuedNote, Tone.Info)
                }
            }
        }
    }

    /** Send every quantity still waiting out its settle, now. ⚠️ Before the
     *  kitchen is sent anything, and before a bill: "make it three" has to be on
     *  the check the kitchen and the guest are about to read. */
    fun settleQtyNow(): List<Job> {
        val ids = qtyJobs.keys.toList()
        return ids.mapNotNull { id ->
            qtyJobs.remove(id)?.cancel()
            val q = qtyOverride[id] ?: return@mapNotNull null
            keep { sendQty(id, q) }
        }
    }

    // ⚠️ **Leaving the screen sends what is buffered, and lets go of the table.**
    // A waiter who taps a dish and immediately walks back to the room would
    // otherwise lose it — the gathering window is short, but "short" is not
    // "never", and a dish that silently did not happen is the worst outcome this
    // file can produce. Through the outbox, so leaving in a dead spot queues it.
    //
    // ⚠️ The hold is a courtesy: it expires by itself after a couple of minutes,
    // which is what makes it safe. Releasing it — after everything above, on the
    // same lock — is what stops a cashier being told somebody is working on a
    // table nobody is standing at.
    DisposableEffect(checkId) {
        onDispose {
            flushAdds()
            settleQtyNow()
            keep {
                queue.withLock {
                    try {
                        api.releaseCheck(checkId)
                    } catch (e: Throwable) {
                        // A courtesy; the hold expires on its own.
                    }
                }
            }
        }
    }

    /** What is on the check as the waiter should see it: the server's lines
     *  with their own unconfirmed presses laid over them. */
    val lines by remember {
        derivedStateOf {
            val out = ArrayList<CheckLine>()
            for (l in check?.lines ?: emptyList()) {
                if (l.voided != null) continue
                val q = qtyOverride[l.lineId]
                val s = servedOverride[l.lineId]
                if (q == null && s == null) {
                    out.add(l)
                    continue
                }
                val qty = q ?: l.qty
                if (qty <= 0) continue
                out.add(
                    l.copy(
                        qty = qty,
                        sum = if (l.qty > 0) l.sum / l.qty * qty else l.price * qty,
                        // "" reads as "just now" — the press is the moment.
                        servedAt = when (s) {
                            null -> l.servedAt
                            true -> l.servedAt ?: ""
                            false -> null
                        },
                    ),
                )
            }
            out as List<CheckLine>
        }
    }

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
    val onCheck by remember {
        derivedStateOf {
            val m = HashMap<String, Int>()
            for (l in lines) {
                val id = l.menuItemId ?: continue
                m[id] = (m[id] ?: 0) + l.qty
            }
            // ⚠️ Plus the taps that have not been sent yet. Without this the
            // number beside a dish is a fact about the network rather than about
            // the check.
            for ((id, n) in pending) m[id] = (m[id] ?: 0) + n
            m as Map<String, Int>
        }
    }

    // What the kitchen has not seen yet, in dishes: the lines typed but not
    // sent, plus the taps still on their way to becoming lines.
    val toSend = lines.filter { !it.fired }.sumOf { it.qty } + pending.values.sum()

    fun add(item: MenuItem, portion: Int? = null) {
        // ⚠️ **Asked, not assumed.** Half a loaf is an ordinary sale and a whole
        // one is the common case; a phone that always added a whole one would
        // leave the waiter correcting it at the till — the walk this removes.
        if (portion == null && !item.portions.isNullOrEmpty()) {
            portionFor = item
            return
        }
        tick()
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

    /** Set a line's quantity. ⚠️ The number moves now; the request goes once the
     *  presses stop — see [QTY_SETTLE_MS]. Removal goes at once: there is no
     *  "next press" after the bin. */
    fun changeQty(line: CheckLine, next: Int) {
        val id = line.lineId
        val target = next.coerceIn(0, 99)
        qtyOverride[id] = target
        tick()
        qtyJobs.remove(id)?.cancel()
        if (target == 0) {
            keep { sendQty(id, 0) }
            return
        }
        qtyJobs[id] = scope.launch {
            delay(QTY_SETTLE_MS)
            qtyJobs.remove(id)
            keep { sendQty(id, target) }
        }
    }

    /** Take one off, from the menu side. */
    fun removeFromLines(item: MenuItem) {
        // ⚠️ **Only from a line the kitchen has not seen.** Once a line is fired
        // the ticket at the pass names a quantity, and quietly lowering it here
        // would leave the paper and the screen disagreeing about one dish.
        //
        // ⚠️ The **newest** unfired line is the one reduced: the tap being undone
        // is almost always the last one.
        val line = lines.lastOrNull { it.menuItemId == item.id && !it.fired }
        if (line == null) {
            // Said rather than silently ignored: pressing minus and having
            // nothing happen is how somebody concludes the screen is stuck.
            show(if ((pending[item.id] ?: 0) > 0) queuedNote else firedOnly, Tone.Info)
            return
        }
        changeQty(line, line.qty - 1)
    }

    fun removeOne(item: MenuItem) {
        // ⚠️ An unsent tap is undone before anything on the check: it is the one
        // the waiter just made, and it costs nothing to take back.
        val bi = buffer.indexOfLast { it.menuItemId == item.id }
        if (bi >= 0) {
            val b = buffer[bi]
            if (b.qty > 1) buffer[bi] = b.copy(qty = b.qty - 1) else buffer.removeAt(bi)
            val left = (pending[item.id] ?: 0) - 1
            if (left > 0) pending[item.id] = left else pending.remove(item.id)
            tick()
            return
        }
        // On its way to the server: wait for it to land, then take one off the
        // line it became.
        if ((pending[item.id] ?: 0) > 0 && queue.isLocked) {
            scope.launch {
                queue.withLock { }
                removeFromLines(item)
            }
            return
        }
        removeFromLines(item)
    }

    /** "The guest has it." Turns green on the press; the reply replaces it. */
    fun toggleServed(l: CheckLine) {
        val id = l.lineId
        val next = l.servedAt == null
        servedOverride[id] = next
        tick()
        keep {
            queue.withLock {
                val res = outbox.send(checkId, OpKind.LineServed, Json.encodeToString(ServedPayload(id, next))) {
                    api.lineServed(checkId, id, next)
                }
                when (res) {
                    is Outbox.Outcome.Sent -> {
                        adopt(res.value)
                        if (servedOverride[id] == next) servedOverride.remove(id)
                    }
                    is Outbox.Outcome.Refused -> {
                        show(res.error.message, Tone.Error)
                        if (servedOverride[id] == next) servedOverride.remove(id)
                    }
                    Outbox.Outcome.Queued -> {
                        hadQueued = true
                        show(queuedNote, Tone.Info)
                    }
                }
            }
        }
    }

    fun fire() {
        if (firing) return
        firing = true
        tick()
        keep {
            // ⚠️ **Whatever is still in the buffer goes first, and this is the
            // one place gathering taps could have cost something real.** A waiter
            // can add a dish and press "send" inside the gathering window; the
            // kitchen would then get the ticket without it. Everything runs on
            // the same lock, so the order is the order of the taps.
            flushAdds().join()
            settleQtyNow().forEach { it.join() }
            // ⚠️ Queued like the rest: a ticket the kitchen never received is the
            // worst of these failures, and it used to vanish with a sentence the
            // waiter had no way to act on.
            val res = queue.withLock {
                outbox.send(checkId, OpKind.Fire, "{}") { api.fire(checkId) }
            }
            when (res) {
                is Outbox.Outcome.Sent -> {
                    adopt(res.value)
                    banner = null
                }
                is Outbox.Outcome.Refused -> show(res.error.message, Tone.Error)
                Outbox.Outcome.Queued -> {
                    hadQueued = true
                    show(queuedNote, Tone.Info)
                }
            }
            firing = false
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
     *  already been handed it. The button is not there until then: a greyed-out
     *  control invites "why can't I", and the answer is the button in its place. */
    fun printBill() {
        if (printing) return
        printing = true
        keep {
            flushAdds().join()
            settleQtyNow().forEach { it.join() }
            try {
                val res = queue.withLock { api.print(checkId, "precheck") }
                adopt(res.check)
                // ⚠️ **Said in a sheet**, and **two different zeros**: "no printer
                // configured here" and "the till app is switched off" both come
                // back as `queued: 0`, and the next move differs entirely.
                notice.value = when {
                    res.queued > 0 -> Note(NoticeKind.Ok, billPrinted)
                    res.tillOff -> Note(NoticeKind.Warn, billTillOff, billTillOffHint)
                    else -> Note(NoticeKind.Warn, billNotQueued, billNotQueuedHint)
                }
            } catch (e: Throwable) {
                notice.value = Note(NoticeKind.Error, billFailed, (e as? ApiError)?.message)
            } finally {
                printing = false
            }
        }
    }

    // ⚠️ **Said when the table opens, not when a button is refused.** The server
    // sends the holder's name for exactly this — and only while the hold is
    // fresh, so a name here means somebody is on it *now*.
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
        Column(Modifier.fillMaxSize()) {
            Row(
                Modifier.fillMaxWidth().statusBarsPadding().padding(horizontal = 12.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                GlassIconButton(Icons.Rounded.ArrowBackIosNew, onClick = onBack)
            }
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                if (loadError.isNotEmpty()) {
                    Column(
                        Modifier.padding(32.dp),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(14.dp),
                    ) {
                        Icon(Icons.Rounded.ErrorOutline, null, tint = c.danger, modifier = Modifier.size(36.dp))
                        Text(loadError, color = c.ink, textAlign = TextAlign.Center)
                        GhostButton(t.common.retry, icon = Icons.Rounded.Refresh) {
                            loadError = ""
                            scope.launch { load() }
                        }
                    }
                } else CircularProgressIndicator(color = c.accent)
            }
        }
        return
    }

    val sections = remember(lines) {
        listOf(
            "draft" to lines.filter { !it.fired },
            "ready" to lines.filter { it.fired && it.servedAt == null && it.readyAt != null },
            "cooking" to lines.filter { it.fired && it.servedAt == null && it.readyAt == null },
            "served" to lines.filter { it.fired && it.servedAt != null },
        )
    }

    val showBill = tab == "check" && lines.isNotEmpty() && toSend == 0
    val dockShown = toSend > 0 || showBill || queued > 0
    val listBottom = if (dockShown) with(density) { dockPx.toDp() } + 12.dp
    else bottomInset.calculateBottomPadding() + 24.dp

    Box(Modifier.fillMaxSize()) {
        Column(Modifier.fillMaxSize()) {
            // ---- Header: which table, and how it is sitting ----
            Row(
                Modifier.fillMaxWidth().statusBarsPadding()
                    .padding(start = 12.dp, end = 12.dp, top = 8.dp, bottom = 4.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                GlassIconButton(Icons.Rounded.ArrowBackIosNew, onClick = onBack)
                Column(Modifier.weight(1f)) {
                    Text(
                        cur.tableNumber?.let { t.check.table(it) } ?: cur.number,
                        color = c.ink, maxLines = 1,
                        style = MaterialTheme.typography.titleLarge.copy(fontWeight = FontWeight.Bold),
                    )
                    val facts = buildList {
                        if (cur.number.isNotBlank()) add("#${cur.number}")
                        if (cur.guests > 0) add(t.look.guests(cur.guests))
                        if (cur.openMin > 0) add(t.look.minutes(cur.openMin))
                    }
                    if (facts.isNotEmpty()) {
                        Text(
                            facts.joinToString(" · "), color = c.muted, maxLines = 1,
                            style = MaterialTheme.typography.labelMedium,
                        )
                    }
                }
                // ⚠️ Behind one button rather than four in the header: these are
                // things a table does occasionally, and four controls above the
                // check would crowd out the two it does constantly.
                GlassIconButton(Icons.Rounded.MoreVert) {
                    settleQtyNow()
                    job = "menu"
                }
            }

            Segmented(
                options = listOf(
                    "check" to "${t.check.tab} · ${lines.size}",
                    "menu" to t.check.menu,
                ),
                selected = tab,
                onSelect = { tab = it },
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 6.dp),
            )

            banner?.let { b ->
                InlineBanner(
                    b.text,
                    if (b.tone == Tone.Error) c.danger else c.accent,
                    if (b.tone == Tone.Error) Icons.Rounded.ErrorOutline else Icons.Rounded.Info,
                    Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
                    onClose = { banner = null },
                )
            }

            if (tab == "check") {
                LazyColumn(
                    Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(16.dp, 6.dp, 16.dp, listBottom),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    for ((key, group) in sections) {
                        if (group.isEmpty()) continue
                        section(key, group) { l ->
                            CheckRow(
                                line = l,
                                onOpen = {
                                    settleQtyNow()
                                    editing = l
                                },
                                onServed = { toggleServed(l) },
                                onMinus = { changeQty(l, l.qty - 1) },
                                onPlus = { changeQty(l, l.qty + 1) },
                            )
                        }
                    }
                    if (lines.isEmpty()) {
                        item(key = "empty") {
                            EmptyCheck { tab = "menu" }
                        }
                    }
                }
            } else {
                val groups = menuCache.groupsFor(branchId)
                when {
                    groups.isNotEmpty() -> MenuPane(
                        groups = groups,
                        onCheck = onCheck,
                        soldOut = floor.soldOut,
                        view = prefs.view.value,
                        onView = { prefs.setView(it) },
                        lang = prefs.lang.value.code,
                        uploadsBase = api.uploadsBase,
                        bottomPad = listBottom,
                        onAdd = { add(it) },
                        onRemove = { removeOne(it) },
                        state = menuState,
                    )
                    menuCache.failed -> Column(
                        Modifier.fillMaxSize().padding(32.dp),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(14.dp, Alignment.CenterVertically),
                    ) {
                        Text(t.look.menuFailed, color = c.ink, textAlign = TextAlign.Center)
                        GhostButton(t.common.retry, icon = Icons.Rounded.Refresh) {
                            scope.launch { menuCache.ensure(branchId, force = true) }
                        }
                    }
                    else -> Column(
                        Modifier.fillMaxSize().padding(16.dp),
                        verticalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        repeat(6) {
                            Box(Modifier.fillMaxWidth().height(62.dp).placeholder(c, RoundedCornerShape(18.dp)))
                        }
                    }
                }
            }
        }

        // ---- The dock: what the table owes, and the one thing to do next ----
        //
        // ⚠️ **Above Android's navigation bar, not under it.** With three-button
        // navigation a control placed at the bottom of the *layout* lands beneath
        // the bar — so the tap meant to send an order to the kitchen presses Back
        // instead, and the waiter is returned to the room with the food unsent.
        if (dockShown) {
            Column(
                Modifier
                    .align(Alignment.BottomCenter)
                    .fillMaxWidth()
                    .onSizeChanged { dockPx = it.height }
                    .padding(horizontal = 12.dp)
                    .padding(bottom = bottomInset.calculateBottomPadding() + 8.dp)
                    .softShadow(RoundedCornerShape(24.dp), elevation = 6.dp, dark = c.dark)
                    .glassSheet(c, RoundedCornerShape(24.dp))
                    .padding(horizontal = 14.dp, vertical = 12.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                // ⚠️ The outbox, said where the waiter is looking — the root's
                // banner would sit on top of this button.
                if (queued > 0) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                    ) {
                        Icon(Icons.Rounded.CloudUpload, null, tint = c.accent, modifier = Modifier.size(15.dp))
                        Text(t.outbox.waiting(queued), color = c.inkSoft, style = MaterialTheme.typography.labelMedium)
                    }
                }
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f)) {
                        Text(t.look.total, color = c.muted, style = MaterialTheme.typography.labelMedium)
                        if (cur.service > 0) {
                            Text(
                                "${t.look.service(percentText(cur.servicePercent))} · ${money(cur.service)}",
                                color = c.muted, maxLines = 1,
                                style = MaterialTheme.typography.labelSmall.merge(MoneyStyle),
                            )
                        }
                    }
                    Money(
                        cur.total,
                        style = MaterialTheme.typography.titleLarge.copy(fontWeight = FontWeight.Bold),
                    )
                }
                // ⚠️ Shown only when there is something the kitchen has not seen.
                // A permanent button invites being pressed on a table that is
                // already cooking, and the second press is a ticket nobody asked
                // for. ⚠️ And the bill takes its place only on the check tab:
                // nobody prints a bill in the middle of taking an order.
                when {
                    toSend > 0 -> PrimaryButton(
                        t.check.fire(toSend), icon = Icons.Rounded.Send,
                        enabled = !firing, busy = firing,
                    ) { fire() }
                    showBill -> GhostButton(
                        if (printing) t.common.loading else t.bill.print,
                        Modifier.fillMaxWidth(),
                        icon = Icons.Rounded.Print, enabled = !printing,
                    ) { printBill() }
                }
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
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text(item.name, style = MaterialTheme.typography.titleMedium, color = c.ink)
                Text(t.check.portionAsk, style = MaterialTheme.typography.bodyMedium, color = c.muted)
                val parts = listOf(100) + (item.portions ?: emptyList())
                FlowRow(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    parts.forEach { p ->
                        // ⚠️ 64dp tall: pressed with a thumb, standing, holding
                        // something else.
                        Box(
                            Modifier.size(width = 96.dp, height = 64.dp)
                                .then(
                                    if (p == 100) Modifier.clip(RoundedCornerShape(16.dp))
                                        .background(c.accentSoft, RoundedCornerShape(16.dp))
                                        .border(1.dp, c.accent.copy(alpha = 0.5f), RoundedCornerShape(16.dp))
                                    else Modifier.glass(c, RoundedCornerShape(16.dp)),
                                )
                                .clickable { portionFor = null; add(item, p) },
                            contentAlignment = Alignment.Center,
                        ) {
                            Text(
                                if (p == 100) t.check.portionWhole else portionText(p),
                                style = MaterialTheme.typography.titleMedium,
                                color = if (p == 100) c.accent else c.ink,
                            )
                        }
                    }
                }
                Text(
                    t.check.portionCancel, color = c.muted, textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp))
                        .clickable { portionFor = null }.padding(10.dp),
                )
            }
        }
    }

    editing?.let { l ->
        LineDialog(api, checkId, l, onDone = { adopt(it) }, onClose = { editing = null })
    }

    job?.let { j ->
        TableActions(
            api = api, check = cur,
            others = floor.checks.filter { it.id != checkId },
            job = j,
            onJob = { job = it },
            onDone = { next ->
                // ⚠️ **A merge hands back the other table's check.** Drawn here
                // under this table's id, the next dish would be written to a
                // check that no longer exists — so the room is where this ends.
                if (next.id.isNotEmpty() && next.id != checkId) {
                    floor.upsert(next)
                    scope.launch { floor.refresh() }
                    onBack()
                } else {
                    adopt(next)
                    scope.launch { floor.refresh() }
                }
            },
            onClose = { job = null },
        )
    }
}

/** One group of lines under its heading. */
private fun LazyListScope.section(
    key: String,
    group: List<CheckLine>,
    row: @Composable (CheckLine) -> Unit,
) {
    item(key = "h:$key") {
        val c = KeelTheme.colors
        val (label, tone) = when (key) {
            "draft" -> t.look.secDraft to c.accent
            "ready" -> t.look.secReady to c.ready
            "cooking" -> t.look.secCooking to c.inkSoft
            else -> t.look.secServed to c.muted
        }
        SectionHeader(label, tone, group.sumOf { it.qty }, Modifier.animateItem())
    }
    items(group, key = { it.lineId }) { l ->
        Box(Modifier.animateItem()) { row(l) }
    }
}

@Composable
private fun EmptyCheck(onMenu: () -> Unit) {
    val c = KeelTheme.colors
    Column(
        Modifier.fillMaxWidth().padding(top = 48.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        Box(Modifier.size(72.dp).glass(c, CircleShape), contentAlignment = Alignment.Center) {
            Icon(Icons.Rounded.ReceiptLong, null, tint = c.muted, modifier = Modifier.size(30.dp))
        }
        Text(
            t.check.empty, color = c.inkSoft, textAlign = TextAlign.Center,
            style = MaterialTheme.typography.bodyMedium,
        )
        GhostButton(t.check.menu, icon = Icons.Rounded.GridView, onClick = onMenu)
    }
}

@Composable
private fun CheckRow(
    line: CheckLine,
    onOpen: () -> Unit,
    onServed: () -> Unit,
    onMinus: () -> Unit,
    onPlus: () -> Unit,
) {
    val c = KeelTheme.colors
    val served = line.servedAt != null
    Row(
        Modifier.fillMaxWidth()
            .glass(c, RoundedCornerShape(18.dp))
            // ⚠️ The whole row opens the dialog rather than a small edit icon:
            // this is used with a thumb, walking, and a target the size of a
            // glyph is why somebody gives up and walks to the till.
            .clickable(onClick = onOpen)
            .padding(start = 10.dp, end = 10.dp, top = 10.dp, bottom = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        // How many, big enough to count from arm's length. Orange while the
        // kitchen has not seen it.
        Box(
            Modifier.size(40.dp).clip(RoundedCornerShape(12.dp))
                .background(if (!line.fired) c.accentSoft else c.field),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                "${line.qty}",
                color = if (!line.fired) c.accent else c.ink,
                style = MaterialTheme.typography.titleMedium.merge(MoneyStyle),
            )
        }

        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(
                buildString {
                    // ⚠️ The line's own frozen name, not the menu's: it is what
                    // the guest agreed to, and a dish renamed at six o'clock must
                    // not rewrite a check opened at five.
                    line.portion?.let { append(portionText(it)).append(" · ") }
                    append(line.name)
                },
                style = MaterialTheme.typography.bodyLarge.copy(fontWeight = FontWeight.Medium),
                color = c.ink, maxLines = 2, overflow = TextOverflow.Ellipsis,
            )
            // What the kitchen was told about this dish. Shown because it is the
            // half of the order a guest will check.
            line.comment?.takeIf { it.isNotBlank() }?.let {
                Text(
                    it, color = c.muted, maxLines = 2, overflow = TextOverflow.Ellipsis,
                    style = MaterialTheme.typography.labelMedium.copy(fontStyle = FontStyle.Italic),
                )
            }
            // ⚠️ **Where the dish is, on the row** — and how long ago, because
            // "two minutes" and "twenty minutes" send a waiter to two different
            // places. Served wins over ready: once the guest has it, when it was
            // cooked stops being the question.
            when {
                !line.fired -> Text(t.check.pending, style = MaterialTheme.typography.labelMedium, color = c.accent)
                served -> Text(
                    "✓ ${t.check.servedAgo(timeAgo(line.servedAt, t.common.timeAgo))}",
                    style = MaterialTheme.typography.labelMedium, color = c.ready,
                )
                line.readyAt != null -> Text(
                    t.check.readyAgo(timeAgo(line.readyAt, t.common.timeAgo)),
                    style = MaterialTheme.typography.labelMedium, color = c.ready,
                )
            }
        }

        Column(
            horizontalAlignment = Alignment.End,
            verticalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            Money(line.sum, style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.SemiBold))
            if (line.fired) {
                // ⚠️ **"The guest has it", one tap, on a fired line only.** A
                // draft cannot have been carried anywhere and the server refuses
                // it; a button that always says no is one people stop pressing.
                ServedToggle(served, onServed)
            } else {
                // ⚠️ A stepper only while the kitchen has not seen it. A fired
                // line goes through the dialog, where a reason is asked for.
                GlassStepper(line.qty, onMinus = onMinus, onPlus = onPlus)
            }
        }
    }
}

@Composable
private fun ServedToggle(served: Boolean, onClick: () -> Unit) {
    val c = KeelTheme.colors
    Row(
        Modifier.height(36.dp).clip(CircleShape)
            .background(if (served) c.ready else c.field, CircleShape)
            .border(1.dp, if (served) c.ready else c.fieldBorder, CircleShape)
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(5.dp),
    ) {
        Icon(
            Icons.Rounded.Check, null,
            tint = if (served) Color.White else c.inkSoft,
            modifier = Modifier.size(16.dp),
        )
        if (!served) {
            Text(t.check.serve, color = c.inkSoft, style = MaterialTheme.typography.labelMedium)
        }
    }
}

/** "10" rather than "10.0" — the rate is whole almost everywhere. */
private fun percentText(p: Double): String =
    if (p % 1.0 == 0.0) p.toInt().toString() else p.toString()

/** Take a sent-or-refused batch back out of the optimistic count.
 *
 *  ⚠️ A batch that stayed "pending" after the server refused it would show a
 *  dish on the check that the kitchen will never see — the failure this screen
 *  exists to prevent, and the mirror of clearing one that is merely queued. */
private fun drop(
    pending: SnapshotStateMap<String, Int>,
    lines: List<AddLineRequest>,
) {
    for (l in lines) {
        val left = (pending[l.menuItemId] ?: 0) - l.qty
        if (left > 0) pending[l.menuItemId] = left else pending.remove(l.menuItemId)
    }
}
