package uz.keel.team.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.Checklist
import androidx.compose.material.icons.rounded.Search
import androidx.compose.ui.draw.clip
import uz.keel.design.Chip
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.AddCircleOutline
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import kotlinx.coroutines.launch
import uz.keel.design.GhostButton
import uz.keel.design.GlassField
import uz.keel.design.KeelTheme
import uz.keel.design.PrimaryButton
import uz.keel.design.glass
import uz.keel.design.glassSheet
import uz.keel.design.qty
import uz.keel.team.data.ApiError
import uz.keel.team.data.KeelApi
import uz.keel.team.data.OrderLine
import uz.keel.team.data.ShoppingCatalogRow
import uz.keel.team.data.ShoppingDraftRow
import uz.keel.team.data.ShoppingOrder
import uz.keel.team.data.Staff
import uz.keel.team.t

// Writing the list somebody is sent to the market with.
//
// ⚠️ **The other half of the buying, and the reason it is a separate
// permission.** One person decides what the restaurant needs; another goes and
// gets it and writes down what it cost. Held by the same account the list stops
// being a check on the trip and becomes a note the buyer wrote to themselves.
//
// ⚠️ **The form starts from the shortage the store already worked out.** A blank
// page would be a second, competing shopping list — one the arithmetic produces
// and one a person types — and the buyer would have no way to tell which is
// real.

/** Whether this account writes shopping lists **in this app**.
 *
 *  ⚠️ **Exactly the permission, and no longer narrower than it.** This used to
 *  also demand management or the store — "a cashier's phone is not where buying
 *  gets planned" — and that reading was wrong about who notices things running
 *  out. It is the barman with an empty fridge and the cook who used the last of
 *  the flour, and neither of them voids checks or counts shelves. An owner now
 *  ticks the box on the person (`staff.canBuyOrder`), and a second condition
 *  here would silently overrule the tick they just made — which is the worst
 *  kind of permission bug, because the screen offers no way to find out why.
 *
 *  ⚠️ **Expressed in permissions, never in the role's name.** The spelling of a
 *  job title grants nothing (`models/staffrole.go`). */
fun canWriteHere(staff: Staff): Boolean = staff.perms.contains("buyorder")

/** Whether this account hands out what a request asks for.
 *
 *  ⚠️ **Its own permission rather than a corner of `stock`.** Counting a shelf
 *  and emptying it are different acts, and folded together every person given a
 *  phone to count the fridge would also hold the button that sends a case of
 *  vodka across town. */
fun canIssueHere(staff: Staff): Boolean = staff.perms.contains("stockissue")

/** The word for where a line goes.
 *
 *  ⚠️ Composable because the dictionary is: the language is read out of the
 *  composition, and a plain function reaching for it would be one call away from
 *  a screen that keeps the word it was drawn with when somebody switches
 *  language. */
@Composable
internal fun sourceWord(source: String): String =
    if (source == "store") t.zakup.fromStore else t.zakup.fromMarket

private data class OrderDraft(
    val key: String,
    val ingredientId: String?,
    val name: String,
    val unit: String,
    val qty: String = "",
    val packName: String = "",
    val packQty: Double = 0.0,
    /** Whether the number typed counts packs. ⚠️ A flag, not a converted figure
     *  — the server owns the arithmetic, because the result is what somebody is
     *  sent to buy. */
    val pack: Boolean = false,
    /** Where the line will be answered from. ⚠️ Shown, never chosen: the
     *  catalogue decides, but a line filed wrongly otherwise sits all morning on
     *  a phone belonging to somebody who was never going to answer it. */
    val source: String = "market",
)

@Composable
fun ZakupScreen(api: KeelApi, bottomInset: PaddingValues) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()

    var suggested by remember { mutableStateOf<List<ShoppingDraftRow>>(emptyList()) }
    /** ⚠️ Needed as well as the shortage: a list can ask for something above its
     *  minimum, and without the catalogue the writer had to type the name —
     *  which creates a second ingredient no tech card points at. */
    var catalog by remember { mutableStateOf<List<ShoppingCatalogRow>>(emptyList()) }
    var orders by remember { mutableStateOf<List<ShoppingOrder>>(emptyList()) }
    val lines = remember { mutableStateListOf<OrderDraft>() }
    var query by remember { mutableStateOf("") }
    // Tomorrow: today's shopping has been done by the time anybody writes this.
    var forDate by remember {
        mutableStateOf(java.time.LocalDate.now().plusDays(1).toString())
    }
    var preview by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    var done by remember { mutableStateOf("") }
    var tick by remember { mutableIntStateOf(0) }
    /** The list somebody is standing over with a bag in front of them. */
    var accepting by remember { mutableStateOf<ShoppingOrder?>(null) }

    val loadFailed = t.zakup.loadFailed
    val sendFailed = t.zakup.sendFailed
    val sentWord = t.zakup.sent
    val splitWord = t.zakup.sentSplit

    LaunchedEffect(tick) {
        try {
            val draft = api.buyOrderDraft()
            suggested = draft.rows
            catalog = draft.catalog
            orders = api.buyOrders().orders
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else loadFailed
        }
    }

    val chosen = lines.mapNotNull { it.ingredientId }.toSet()
    val q = query.trim()
    /** Short things first, then the whole catalogue.
     *
     *  ⚠️ **Everything, without typing.** The catalogue used to appear only once
     *  somebody typed — so a restaurant that has never set a minimum on anything
     *  (most of them: the shortage list is opt-in per ingredient) opened this
     *  screen to an empty list and no way to discover the ingredients were
     *  there. A picker whose contents are hidden until you guess a name is one
     *  people conclude is broken.
     *
     *  ⚠️ **Nothing is capped**: the one ingredient somebody cannot find is the
     *  one they type by hand, and that creates a duplicate no tech card points
     *  at. */
    val short = suggested.filter {
        it.ingredientId !in chosen && (q.isEmpty() || it.name.contains(q, true))
    }
    val shortIds = short.map { it.ingredientId }.toSet()
    val rest = catalog
        .filter {
            it.ingredientId !in chosen && it.ingredientId !in shortIds &&
                (q.isEmpty() || it.name.contains(q, true))
        }
        .map {
            ShoppingDraftRow(
                ingredientId = it.ingredientId,
                name = it.name,
                unit = it.unit,
                packName = it.packName,
                packQty = it.packQty,
                source = it.source,
            )
        }
    val shown = short + rest
    /** A name the store does not know. ⚠️ Offered rather than refused: a list
     *  somebody cannot finish writing is a list they write on paper instead, and
     *  then nothing here sees it. */
    val unknown = if (q.length >= 2 && catalog.none { it.name.equals(q, true) }) q else ""

    fun add(row: ShoppingDraftRow?, name: String) {
        done = ""
        lines.add(
            OrderDraft(
                key = row?.ingredientId ?: "new-${System.currentTimeMillis()}",
                ingredientId = row?.ingredientId,
                name = row?.name ?: name,
                unit = row?.unit ?: "",
                qty = if ((row?.qty ?: 0.0) > 0) qty(row!!.qty) else "",
                packName = row?.packName ?: "",
                packQty = row?.packQty ?: 0.0,
                // ⚠️ A typed name the catalogue does not have is something to
                // buy: nobody has ever put it on a shelf here, so routing it to
                // a storekeeper would leave it unanswered while looking, on
                // every screen, exactly like a request being dealt with.
                source = row?.source ?: "market",
            ),
        )
        query = ""
    }

    val ready = lines.filter { (it.qty.toDoubleOrNull() ?: 0.0) > 0 }
    val storeCount = ready.count { it.source == "store" }
    val marketCount = ready.size - storeCount

    fun send() {
        busy = true
        error = ""
        scope.launch {
            try {
                api.createBuyOrder(
                    forDate = forDate,
                    lines = ready.map {
                        OrderLine(
                            ingredientId = it.ingredientId,
                            name = it.name,
                            qty = it.qty.toDoubleOrNull() ?: 0.0,
                            pack = it.pack,
                        )
                    },
                )
                lines.clear()
                preview = false
                // ⚠️ **The split is said out loud.** The writer chose none of
                // it, so a plain "sent" would leave them no way to notice that
                // the lemons they meant for the market went to a storekeeper —
                // the one mistake this routing can make, and one a person fixes
                // in the catalogue in ten seconds if they are told.
                done = if (storeCount > 0 && marketCount > 0) {
                    splitWord(marketCount, storeCount)
                } else {
                    sentWord(ready.size)
                }
                tick += 1
            } catch (e: Throwable) {
                error = if (e is ApiError) e.message else sendFailed
            } finally {
                busy = false
            }
        }
    }

    val waitingCount = orders.count { it.waiting }
    /** Which half of the screen is showing: writing a list, or the lists sent.
     *
     *  ⚠️ **Two tabs rather than one long page.** Writing a list and signing for
     *  what arrived are done at different hours by the same person, and on one
     *  page the accept button sat under the whole catalogue — the one list that
     *  needed them was the one they had to scroll past everything to find. */
    var tab by remember { mutableIntStateOf(0) }
    var jumped by remember { mutableStateOf(false) }
    // ⚠️ Opens on the sent lists once, when something is waiting to be counted.
    // After that the tab is the person's choice and stays theirs.
    LaunchedEffect(waitingCount) {
        if (!jumped && waitingCount > 0) {
            tab = 1
            jumped = true
        }
    }
    val today = java.time.LocalDate.now()
    val days = listOf(
        t.zakup.today to today,
        t.zakup.tomorrow to today.plusDays(1),
        t.zakup.dayAfter to today.plusDays(2),
    )

    Box(Modifier.fillMaxSize()) {
        LazyColumn(
            Modifier.fillMaxSize().imePadding(),
            contentPadding = PaddingValues(
                start = 16.dp,
                end = 16.dp,
                top = 8.dp,
                bottom = bottomInset.calculateBottomPadding() + if (tab == 0 && ready.isNotEmpty()) 96.dp else 16.dp,
            ),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item { MarketHeader(t.zakup.title) }
            item {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Chip(t.zakup.tabNew + if (lines.isNotEmpty()) " · ${lines.size}" else "", tab == 0) { tab = 0 }
                    Chip(t.zakup.tabSent + if (waitingCount > 0) " · $waitingCount" else "", tab == 1) { tab = 1 }
                }
            }
            if (error.isNotEmpty()) item { MarketBanner(error, error = true) }
            if (done.isNotEmpty()) item { MarketBanner(done, error = false) }

            if (tab == 0) {
                if (waitingCount > 0) {
                    item {
                        Text(
                            t.zakup.waitingBanner(waitingCount),
                            Modifier
                                .fillMaxWidth()
                                .clip(RoundedCornerShape(14.dp))
                                .background(c.accentSoft)
                                .clickable { tab = 1 }
                                .padding(horizontal = 14.dp, vertical = 10.dp),
                            style = MaterialTheme.typography.bodyMedium,
                            color = c.ink,
                        )
                    }
                }

                // ---- For which day ----
                //
                // ⚠️ **Three chips, not a date box.** The box took "2026-09-07"
                // typed by hand, and a list is written for tomorrow nine times in
                // ten; a typo there sends somebody to the market on the wrong day.
                item {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        SectionTitle(t.zakup.forDate)
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            days.forEach { (word, day) ->
                                Chip(word, forDate == day.toString()) { forDate = day.toString() }
                            }
                        }
                    }
                }

                item { SectionTitle(t.zakup.listTitle, lines.size) }
                if (lines.isEmpty()) {
                    item {
                        Text(
                            t.zakup.emptyList,
                            Modifier
                                .fillMaxWidth()
                                .border(1.dp, c.line, RoundedCornerShape(18.dp))
                                .padding(16.dp),
                            style = MaterialTheme.typography.bodyMedium,
                            color = c.muted,
                        )
                    }
                }
                items(lines, key = { it.key }) { l ->
                    Row(
                        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp)).padding(10.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        Column(Modifier.weight(1f).padding(start = 4.dp)) {
                            Text(l.name, style = MaterialTheme.typography.bodyLarge, color = c.ink)
                            // ⚠️ On every row, not only where it is surprising:
                            // a badge that appears sometimes is one people stop
                            // reading, and the row it is missing from is the one
                            // that needed it.
                            Pill(sourceWord(l.source), if (l.source == "store") c.inkSoft else c.accent, Modifier.padding(top = 3.dp))
                        }
                        QtyField(
                            value = l.qty,
                            onValueChange = { v ->
                                val i = lines.indexOfFirst { it.key == l.key }
                                if (i >= 0) lines[i] = l.copy(qty = v)
                            },
                            unit = l.unit,
                            packName = l.packName,
                            packQty = l.packQty,
                            inPacks = l.pack,
                            onToggle = {
                                val i = lines.indexOfFirst { it.key == l.key }
                                if (i >= 0) lines[i] = l.copy(pack = !l.pack)
                            },
                            modifier = Modifier.width(150.dp),
                            placeholder = t.zakup.qty,
                        )
                        Box(
                            Modifier.size(30.dp).clip(CircleShape).clickable { lines.removeAll { it.key == l.key } },
                            contentAlignment = Alignment.Center,
                        ) {
                            Icon(Icons.Rounded.Close, null, tint = c.muted, modifier = Modifier.size(18.dp))
                        }
                    }
                }

                item {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        SectionTitle(t.zakup.shortTitle)
                        GlassField(
                            query,
                            { query = it },
                            t.zakup.search,
                            trailing = {
                                Icon(Icons.Rounded.Search, null, tint = c.muted, modifier = Modifier.size(20.dp))
                            },
                        )
                    }
                }
                if (shown.isEmpty() && unknown.isEmpty()) {
                    item {
                        Text(t.zakup.nothingShort, style = MaterialTheme.typography.bodyMedium, color = c.muted)
                    }
                }
                items(shown, key = { it.ingredientId }) { row ->
                    PickRow(
                        title = row.name,
                        // A catalogue row has no shortage figures — only its unit, which
                        // is what the writer needs before typing a number.
                        subtitle = sourceWord(row.source) + " · " + if (row.qty > 0) {
                            "${t.zakup.onHand(qty(row.onHand), row.unit)} · " +
                                t.zakup.need(qty(row.qty), row.unit)
                        } else {
                            t.zakup.unitIs(row.unit)
                        },
                    ) { add(row, row.name) }
                }
                if (unknown.isNotEmpty()) {
                    item {
                        PickRow(
                            title = t.zakup.addNew(unknown),
                            subtitle = "",
                            icon = Icons.Rounded.AddCircleOutline,
                        ) { add(null, unknown) }
                    }
                }
            } else {
                if (orders.isEmpty()) {
                    item {
                        Text(t.zakup.emptySent, style = MaterialTheme.typography.bodyMedium, color = c.muted)
                    }
                }
                // ⚠️ **The ones waiting to be counted first**: they are the only
                // ones anybody has to act on.
                val sorted = orders.sortedByDescending { it.waiting }.take(12)
                items(sorted, key = { it.id }) { o -> SentOrderCard(o) { accepting = o } }
            }
        }

        if (tab == 0 && ready.isNotEmpty()) {
            Box(
                Modifier
                    .align(Alignment.BottomCenter)
                    .padding(bottom = bottomInset.calculateBottomPadding())
                    .padding(horizontal = 12.dp, vertical = 8.dp)
                    .fillMaxWidth()
                    .glassSheet(c, RoundedCornerShape(22.dp))
                    .padding(10.dp),
            ) {
                PrimaryButton(t.zakup.review + " · " + ready.size) { preview = true }
            }
        }
    }

    accepting?.let { order ->
        AcceptDialog(
            api = api,
            order = order,
            onClose = { accepting = null },
            onDone = { message ->
                accepting = null
                done = message
                tick += 1
            },
            onError = { error = it },
        )
    }

    // ⚠️ **A list is somebody else's morning.** The buyer will not be able to ask
    // what "5" meant, so the numbers and the units are read back once on one
    // page, in the words they will arrive in.
    if (preview) {
        Dialog(onDismissRequest = { preview = false }) {
            Column(
                Modifier
                    .widthIn(max = 400.dp)
                    .glassSheet(c, RoundedCornerShape(26.dp))
                    .padding(18.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Text(t.zakup.previewTitle, style = MaterialTheme.typography.headlineMedium, color = c.ink)
                Text(
                    t.zakup.previewBody(forDate) +
                        if (storeCount > 0 && marketCount > 0) {
                            "\n" + t.zakup.splitNote(marketCount, storeCount)
                        } else {
                            ""
                        },
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.muted,
                )
                // ⚠️ `fill = false`: a long list scrolls inside the sheet, and the
                // two buttons stay on screen instead of being pushed off its end.
                Column(
                    Modifier.weight(1f, fill = false).verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    ready.forEach { l ->
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Column(Modifier.weight(1f)) {
                                Text(l.name, style = MaterialTheme.typography.bodyLarge, color = c.ink)
                                Text(sourceWord(l.source), style = MaterialTheme.typography.labelMedium, color = c.muted)
                            }
                            // ⚠️ Read back in both where they differ: the list
                            // travels to somebody else's morning and "2" has to
                            // be unambiguous before it leaves.
                            val n = l.qty.toDoubleOrNull() ?: 0.0
                            Text(
                                if (l.pack && l.packQty > 0) {
                                    "${l.qty} ${l.packName} = ${qty(n * l.packQty)} ${l.unit}"
                                } else {
                                    "${l.qty} ${l.unit}"
                                },
                                style = MaterialTheme.typography.titleMedium,
                                color = c.ink,
                            )
                        }
                    }
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    GhostButton(t.zakup.back, Modifier.weight(1f)) { preview = false }
                    Box(Modifier.weight(1f)) {
                        PrimaryButton(t.zakup.send, busy = busy) { send() }
                    }
                }
            }
        }
    }
}

/** One list that was sent, and where it stands.
 *
 *  ⚠️ **The accept button lives on the card it accepts.** It used to be a row of
 *  identical buttons under the list of cards, and with two lists on their way
 *  nothing said which button belonged to which. */
@Composable
private fun SentOrderCard(o: ShoppingOrder, onAccept: () -> Unit) {
    val c = KeelTheme.colors
    val got = o.lines.count { it.gotAt.isNotEmpty() && !it.missing }
    Column(
        Modifier
            .fillMaxWidth()
            .glass(c, RoundedCornerShape(20.dp))
            .then(if (o.waiting) Modifier.border(1.5.dp, c.accent.copy(alpha = 0.6f), RoundedCornerShape(20.dp)) else Modifier)
            .padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(o.forDate, Modifier.weight(1f), style = MaterialTheme.typography.titleMedium, color = c.ink)
            Pill(sourceWord(o.source), c.inkSoft)
            // ⚠️ **Only the middle state gets a colour.** "Waiting" is what every
            // request looks like for its first hour and "signed for" is the end;
            // the one somebody has to act on is goods that left somebody's hands
            // and reached nobody's.
            Pill(
                when (o.status) {
                    "done" -> t.zakup.statusDone
                    "shipped" -> t.zakup.statusShipped
                    else -> t.zakup.statusSent
                },
                when {
                    o.waiting -> c.accent
                    o.status == "done" -> c.ready
                    else -> c.muted
                },
            )
        }
        // ⚠️ Asked and brought together: "asked for ten, brought six" is the
        // sentence this document exists to make possible.
        ProgressLine(got, o.lines.size)
        Text(
            t.zakup.progress(got, o.lines.size) + if (o.createdBy.isNotEmpty()) " · ${o.createdBy}" else "",
            style = MaterialTheme.typography.labelMedium,
            color = c.muted,
        )
        // ⚠️ **The button only exists on the list that is waiting for it.** An
        // accept button on a list nobody has shopped yet would answer a question
        // nobody has asked.
        if (o.waiting) PrimaryButton(t.zakup.accept, icon = Icons.Rounded.Checklist) { onAccept() }
    }
}

/** ---- Counting what turned up ----
 *
 *  ⚠️ **What reaches the shelf is what the restaurant counted**, not what the
 *  buyer says he handed over. Where the two differ, the difference is the record
 *  — the thing that had nowhere to be written before, and the reason the
 *  delivery is created here rather than at the market.
 *
 *  ⚠️ **A row left alone is accepted as sent.** Signing without retyping
 *  anything is saying "this is right", which is the ordinary case; a form that
 *  demanded every figure again is a form people close. */
@Composable
private fun AcceptDialog(
    api: KeelApi,
    order: ShoppingOrder,
    onClose: () -> Unit,
    onDone: (String) -> Unit,
    onError: (String) -> Unit,
) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()
    val counted = remember(order.id) { mutableStateMapOf<String, String>() }
    var busy by remember { mutableStateOf(false) }
    val sendFailed = t.zakup.sendFailed
    val acceptedWord = t.zakup.accepted
    /** ⚠️ Minted once per dialog rather than per attempt: a phone that retried
     *  after a timeout must not write a second delivery. */
    val clientId = remember(order.id) {
        "acc-" + order.id + "-" + System.currentTimeMillis().toString(36)
    }

    val sent = order.lines.filter { it.gotAt.isNotEmpty() && !it.missing }
    val diffs = sent.count { l ->
        val v = counted[l.id]?.toDoubleOrNull()
        v != null && v != l.gotQty
    }

    Dialog(onDismissRequest = onClose) {
        Column(
            Modifier
                .widthIn(max = 420.dp)
                .glassSheet(c, RoundedCornerShape(26.dp))
                .padding(18.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text(t.zakup.acceptTitle + " · " + order.forDate, style = MaterialTheme.typography.headlineMedium, color = c.ink)
            Text(t.zakup.acceptBody, style = MaterialTheme.typography.bodyMedium, color = c.muted)
            // ⚠️ `fill = false` keeps the two buttons on screen under a long list.
            Column(
                Modifier.weight(1f, fill = false).verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                sent.forEach { l ->
                    val typed = counted[l.id]?.toDoubleOrNull()
                    val differs = typed != null && typed != l.gotQty
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .glass(c, RoundedCornerShape(16.dp))
                            .then(if (differs) Modifier.border(1.5.dp, c.warn, RoundedCornerShape(16.dp)) else Modifier)
                            .padding(10.dp),
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(Modifier.weight(1f).padding(start = 4.dp)) {
                            Text(l.name, style = MaterialTheme.typography.bodyLarge, color = c.ink)
                            Text(
                                t.zakup.sentQty(qty(l.gotQty), l.unit),
                                style = MaterialTheme.typography.labelMedium,
                                color = c.muted,
                            )
                            // ⚠️ **Left alone is "right", and it says so.** The
                            // rule that an untouched row is accepted as sent was
                            // written in a paragraph above the list, which is
                            // where nobody reads it.
                            if (differs) {
                                val d = typed!! - l.gotQty
                                Text(
                                    (if (d > 0) "+" else "−") + qty(kotlin.math.abs(d)) + " " + l.unit,
                                    style = MaterialTheme.typography.labelMedium,
                                    color = c.warn,
                                )
                            } else {
                                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                                    Icon(Icons.Rounded.Check, null, tint = c.ready, modifier = Modifier.size(14.dp))
                                    Text(t.zakup.acceptOk, style = MaterialTheme.typography.labelMedium, color = c.ready)
                                }
                            }
                        }
                        GlassField(
                            value = counted[l.id] ?: "",
                            onValueChange = { v -> counted[l.id] = v.replace(',', '.') },
                            placeholder = qty(l.gotQty),
                            modifier = Modifier.width(110.dp),
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                        )
                    }
                }
            }
            if (diffs > 0) {
                Text(t.zakup.acceptDiffs(diffs), style = MaterialTheme.typography.bodyMedium, color = c.warn)
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                GhostButton(t.zakup.back, Modifier.weight(1f)) { onClose() }
                Box(Modifier.weight(1f)) {
                    PrimaryButton(t.zakup.acceptSend, busy = busy) {
                        busy = true
                        scope.launch {
                            try {
                                api.acceptOrder(
                                    orderId = order.id,
                                    clientId = clientId,
                                    // Only what differs — a row left out keeps
                                    // what it was told.
                                    counted = counted
                                        .mapNotNull { (id, v) ->
                                            v.toDoubleOrNull()?.let { id to it }
                                        }
                                        .filter { it.second >= 0 }
                                        .toMap(),
                                )
                                onDone(acceptedWord)
                            } catch (e: Throwable) {
                                onError(if (e is ApiError) e.message else sendFailed)
                            } finally {
                                busy = false
                            }
                        }
                    }
                }
            }
        }
    }
}
