package uz.keel.team.ui.screens

import androidx.compose.foundation.clickable
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

    LazyColumn(
        Modifier.fillMaxSize().imePadding(),
        contentPadding = PaddingValues(
            start = 16.dp,
            end = 16.dp,
            top = 8.dp,
            bottom = bottomInset.calculateBottomPadding(),
        ),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        item {
            Column(
                Modifier.statusBarsPadding().padding(top = 8.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                if (error.isNotEmpty()) {
                    Text(error, style = MaterialTheme.typography.bodyMedium, color = c.danger)
                }
                if (done.isNotEmpty()) {
                    Text(done, style = MaterialTheme.typography.bodyMedium, color = c.accent)
                }
                Text(t.zakup.forDate, style = MaterialTheme.typography.titleLarge, color = c.ink)
                GlassField(forDate, { forDate = it }, "2026-09-07")
            }
        }

        if (lines.isNotEmpty()) {
            item {
                Text(t.zakup.listTitle, style = MaterialTheme.typography.titleLarge, color = c.ink)
            }
            items(lines, key = { it.key }) { l ->
                Column(
                    Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp)).padding(12.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Column(Modifier.weight(1f)) {
                            Text(
                                l.name,
                                style = MaterialTheme.typography.bodyLarge,
                                color = c.ink,
                            )
                            // ⚠️ On every row, not only where it is surprising:
                            // a badge that appears sometimes is one people stop
                            // reading, and the row it is missing from is the one
                            // that needed it.
                            Text(
                                sourceWord(l.source),
                                style = MaterialTheme.typography.labelMedium,
                                color = c.muted,
                            )
                        }
                        Icon(
                            Icons.Rounded.Close,
                            null,
                            tint = c.muted,
                            modifier = Modifier
                                .size(20.dp)
                                .clickable { lines.removeAll { it.key == l.key } },
                        )
                    }
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Box(Modifier.weight(1f)) {
                            GlassField(
                                value = l.qty,
                                onValueChange = { v ->
                                    val i = lines.indexOfFirst { it.key == l.key }
                                    if (i >= 0) lines[i] = l.copy(qty = v.replace(',', '.'))
                                },
                                placeholder = t.zakup.qty,
                                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                            )
                        }
                        UnitLabel(l.unit, l.packName, l.packQty, l.pack) {
                            val i = lines.indexOfFirst { it.key == l.key }
                            if (i >= 0) lines[i] = l.copy(pack = !l.pack)
                        }
                    }
                }
            }
        }

        item {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(t.zakup.shortTitle, style = MaterialTheme.typography.titleLarge, color = c.ink)
                GlassField(query, { query = it }, t.zakup.search)
            }
        }

        if (shown.isEmpty() && unknown.isEmpty()) {
            item {
                Text(
                    t.zakup.nothingShort,
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.muted,
                )
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

        if (ready.isNotEmpty()) {
            item { PrimaryButton(t.zakup.review) { preview = true } }
        }

        if (orders.isNotEmpty()) {
            item {
                Text(t.zakup.sentTitle, style = MaterialTheme.typography.titleLarge, color = c.ink)
            }
            items(orders.take(8), key = { it.id }) { o ->
                Row(
                    Modifier.fillMaxWidth().glass(c, RoundedCornerShape(16.dp))
                        .padding(horizontal = 14.dp, vertical = 12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Column(Modifier.weight(1f)) {
                        Text(o.forDate, style = MaterialTheme.typography.bodyLarge, color = c.ink)
                        // ⚠️ Asked and brought together: "asked for ten, brought
                        // six" is the sentence this document exists to make
                        // possible.
                        Text(
                            t.zakup.progress(
                                o.lines.count { it.gotAt.isNotEmpty() && !it.missing },
                                o.lines.size,
                            ) + if (o.createdBy.isNotEmpty()) " · ${o.createdBy}" else "",
                            style = MaterialTheme.typography.labelMedium,
                            color = c.muted,
                        )
                    }
                    Column(horizontalAlignment = Alignment.End) {
                        Text(
                            when (o.status) {
                                "done" -> t.zakup.statusDone
                                "shipped" -> t.zakup.statusShipped
                                else -> t.zakup.statusSent
                            },
                            style = MaterialTheme.typography.labelMedium,
                            // ⚠️ **Only the middle state gets a colour.**
                            // "Waiting" is what every request looks like for its
                            // first hour and "signed for" is the end; the one
                            // somebody has to act on is goods that left
                            // somebody's hands and reached nobody's.
                            color = if (o.waiting) c.accent else c.muted,
                        )
                        Text(
                            sourceWord(o.source),
                            style = MaterialTheme.typography.labelMedium,
                            color = c.muted,
                        )
                    }
                }
            }
            // ⚠️ **The button only exists on the list that is waiting for it.**
            // An accept button on a list nobody has shopped yet would answer a
            // question nobody has asked.
            items(orders.filter { it.waiting }, key = { "acc-" + it.id }) { o ->
                GhostButton(t.zakup.accept, Modifier.fillMaxWidth()) { accepting = o }
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
                    .widthIn(max = 380.dp)
                    .glassSheet(c, RoundedCornerShape(26.dp))
                    .padding(18.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Text(
                    t.zakup.previewTitle,
                    style = MaterialTheme.typography.headlineMedium,
                    color = c.ink,
                )
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
                Column(
                    Modifier.verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    ready.forEach { l ->
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Column(Modifier.weight(1f)) {
                                Text(
                                    l.name,
                                    style = MaterialTheme.typography.bodyLarge,
                                    color = c.ink,
                                )
                                Text(
                                    sourceWord(l.source),
                                    style = MaterialTheme.typography.labelMedium,
                                    color = c.muted,
                                )
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
                                style = MaterialTheme.typography.bodyLarge,
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

    Dialog(onDismissRequest = onClose) {
        Column(
            Modifier
                .widthIn(max = 380.dp)
                .glassSheet(c, RoundedCornerShape(26.dp))
                .padding(18.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text(
                t.zakup.acceptTitle,
                style = MaterialTheme.typography.headlineMedium,
                color = c.ink,
            )
            Text(
                t.zakup.acceptBody,
                style = MaterialTheme.typography.bodyMedium,
                color = c.muted,
            )
            Column(
                Modifier.verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                sent.forEach { l ->
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(Modifier.weight(1f)) {
                            Text(
                                l.name,
                                style = MaterialTheme.typography.bodyLarge,
                                color = c.ink,
                            )
                            Text(
                                qty(l.gotQty) + " " + l.unit,
                                style = MaterialTheme.typography.labelMedium,
                                color = c.muted,
                            )
                        }
                        Box(Modifier.widthIn(max = 120.dp)) {
                            GlassField(
                                value = counted[l.id] ?: "",
                                onValueChange = { v ->
                                    counted[l.id] = v.replace(',', '.')
                                },
                                placeholder = qty(l.gotQty),
                                keyboardOptions = KeyboardOptions(
                                    keyboardType = KeyboardType.Decimal,
                                ),
                            )
                        }
                    }
                }
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
