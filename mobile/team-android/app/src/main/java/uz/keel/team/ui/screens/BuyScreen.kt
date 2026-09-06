package uz.keel.team.ui.screens

import androidx.compose.foundation.background
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.AddCircleOutline
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.SwapHoriz
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import uz.keel.design.GlassField
import uz.keel.design.KeelTheme
import uz.keel.design.PrimaryButton
import uz.keel.design.glass
import uz.keel.design.money
import uz.keel.design.qty
import uz.keel.team.data.ApiError
import uz.keel.team.data.BuyCatalogRow
import uz.keel.team.data.BuyLine
import uz.keel.team.data.KeelApi
import uz.keel.team.data.ShoppingOrder
import uz.keel.team.data.ShoppingRow
import uz.keel.team.t

// The market run, written at the stall.
//
// ⚠️ **The delivery a restaurant actually has, and the one that was recorded
// worst.** A supplier sends an invoice; a market run sends somebody with cash at
// six in the morning, and what came back was written on a scrap of paper,
// carried to an office and typed in by whoever had time — a day late, and
// sometimes not at all. Everything downstream stands on it: the shelf, the
// shopping list that decides the *next* run, the cost of every dish, the stop
// list.
//
// ⚠️ **Nothing here computes anything the server has not.** The shortage list
// comes from `/staff/buy/list` — the same function the panel's screen reads —
// because a buyer at a market and an owner in an office must not disagree about
// whether the kitchen is out of beef. That argument happens after the money is
// spent.
//
// ⚠️ **The last price is beside every box on purpose.** It is the only guard the
// figure has: a price typed here reprices every dish that uses the ingredient,
// and `9 000` entered as `90 000` looks like an ordinary number forever
// afterwards. Nothing downstream can tell them apart; a person standing at the
// stall can, if the last one is in front of them.

/** One line being built.
 *
 *  ⚠️ **Kept as text, not numbers.** A half-typed "12" must not become 12 and
 *  then 120 as somebody types 120, and an emptied box has to stay empty rather
 *  than snapping back to 0. */
private data class BuyDraft(
    val key: String,
    val ingredientId: String?,
    val name: String,
    val unit: String,
    val lastPrice: Double,
    val qty: String = "",
    val price: String = "",
    val packName: String = "",
    val packQty: Double = 0.0,
    /** Whether the figures typed count packs. ⚠️ A flag, not a converted number:
     *  the factor is a fact about the ingredient and the result lands on a
     *  shelf, so the server does the arithmetic. */
    val pack: Boolean = false,
)

@Composable
fun BuyScreen(api: KeelApi, bottomInset: PaddingValues) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()

    var purseBalance by remember { mutableStateOf<Double?>(null) }
    var purseIssued by remember { mutableStateOf(0.0) }
    var purseSpent by remember { mutableStateOf(0.0) }
    /** The list somebody sent this buyer. ⚠️ When there is one it **replaces**
     *  the computed shortage: two shopping lists on one screen is a buyer with
     *  no way to tell which one the restaurant actually asked for, and the one
     *  they follow will be whichever they saw last. */
    var order by remember { mutableStateOf<ShoppingOrder?>(null) }
    var shortlist by remember { mutableStateOf<List<ShoppingRow>>(emptyList()) }
    var catalog by remember { mutableStateOf<List<BuyCatalogRow>>(emptyList()) }
    val lines = remember { mutableStateListOf<BuyDraft>() }
    var query by remember { mutableStateOf("") }
    var supplier by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    var done by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var tick by remember { mutableIntStateOf(0) }

    val loadFailed = t.buy.loadFailed
    val sendFailed = t.buy.sendFailed
    val alreadySent = t.buy.alreadySent
    val nothingToSend = t.buy.nothingToSend
    val sentWord = t.buy.sent
    val createdWord = t.buy.created

    LaunchedEffect(tick) {
        try {
            shortlist = api.buyList().groups.flatMap { it.rows }
            catalog = api.buyCatalog().ingredients
            api.buyBalance().let {
                purseBalance = it.balance
                purseIssued = it.issued
                purseSpent = it.spent
            }
            order = api.buyOrders(openOnly = true).orders.firstOrNull()
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else loadFailed
        }
    }

    fun add(
        ingredientId: String?,
        name: String,
        unit: String = "",
        lastPrice: Double = 0.0,
        qtyText: String = "",
        priceText: String = "",
        packName: String = "",
        packQty: Double = 0.0,
    ) {
        done = ""
        // ⚠️ Twice is one line, not two: the same tile pressed again at a stall
        // is somebody checking, not a second bag of the same thing.
        if (ingredientId != null && lines.any { it.ingredientId == ingredientId }) return
        lines.add(
            BuyDraft(
                key = ingredientId ?: "new-${System.currentTimeMillis()}",
                ingredientId = ingredientId,
                name = name,
                unit = unit,
                lastPrice = lastPrice,
                qty = qtyText,
                price = priceText,
                packName = packName,
                packQty = packQty,
            ),
        )
        query = ""
    }

    fun send() {
        val ready = lines.filter { (it.qty.toDoubleOrNull() ?: 0.0) > 0 }.map {
            BuyLine(
                ingredientId = it.ingredientId,
                newName = if (it.ingredientId == null) it.name else null,
                qty = it.qty.toDoubleOrNull() ?: 0.0,
                price = it.price.toDoubleOrNull() ?: 0.0,
                pack = it.pack,
            )
        }
        if (ready.isEmpty()) {
            error = nothingToSend
            return
        }
        busy = true
        error = ""
        scope.launch {
            try {
                // ⚠️ **The id is minted here and kept for the retry.** A market
                // has worse signal than a dining room; without it a resend after
                // a timeout is a second delivery — the shelf raised twice and
                // the invoice paid twice.
                val res = api.buyCreate(
                    clientId = "buy-${System.currentTimeMillis()}-${(0..0xffff).random().toString(16)}",
                    supplier = supplier.trim(),
                    lines = ready,
                )
                lines.clear()
                supplier = ""
                query = ""
                done = if (res.already) alreadySent else sentWord(money(res.purchase.total))
                // ⚠️ What had to be invented is said to the person who pressed
                // the button, not left for a manager to find: an ingredient
                // created at a market has no unit, no minimum and no card.
                if (res.created.isNotEmpty()) {
                    done = done + "\n" + createdWord(res.created.joinToString(", "))
                }
                tick += 1
            } catch (e: Throwable) {
                error = if (e is ApiError) e.message else sendFailed
            } finally {
                busy = false
            }
        }
    }

    fun finish() {
        val id = order?.id ?: return
        busy = true
        error = ""
        scope.launch {
            try {
                val res = api.finishOrder(
                    orderId = id,
                    // ⚠️ Minted and kept for the retry, exactly as a free-form
                    // run does, and for the same reason.
                    clientId = "ord-$id-${System.currentTimeMillis().toString(36)}",
                    supplier = supplier.trim(),
                )
                supplier = ""
                done = if (res.already) alreadySent
                else sentWord(money(res.purchase?.total ?: 0.0))
                tick += 1
            } catch (e: Throwable) {
                error = if (e is ApiError) e.message else sendFailed
            } finally {
                busy = false
            }
        }
    }

    fun mark(lineId: String, body: suspend (String, String) -> ShoppingOrder) {
        val id = order?.id ?: return
        scope.launch {
            try {
                order = body(id, lineId)
                done = ""
            } catch (e: Throwable) {
                error = if (e is ApiError) e.message else sendFailed
            }
        }
    }

    val total = lines.sumOf {
        (it.qty.toDoubleOrNull() ?: 0.0) * (it.price.toDoubleOrNull() ?: 0.0)
    }
    val q = query.trim()
    val matches = if (q.isEmpty()) emptyList() else
        catalog.filter { it.name.contains(q, ignoreCase = true) }.take(12)
    // ⚠️ **Offered rather than refused.** A buyer who cannot record half a run
    // stops recording any of it — the lesson this product paid for with the
    // supplier field and the void reason. The server matches the name against
    // what exists before inventing anything, and marks what it does invent.
    val unknown = if (q.length >= 2 && catalog.none { it.name.equals(q, true) }) q else ""

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
        // ---- What is still in the buyer's pocket ----
        //
        // ⚠️ **First, and before the list.** It is the number that decides
        // whether this trip happens at all, and somebody who has to ring the
        // office to find out how much they are carrying will guess instead.
        //
        // ⚠️ **Negative is shown rather than clamped.** A buyer who ran out and
        // paid for the last crate themselves is owed money, and a purse that
        // stopped at zero would be silent about exactly the debt they are
        // waiting on.
        purseBalance?.let { balance ->
            item {
                Column(
                    Modifier
                        .statusBarsPadding()
                        .padding(top = 8.dp)
                        .fillMaxWidth()
                        .glass(c, RoundedCornerShape(20.dp))
                        .padding(14.dp),
                    verticalArrangement = Arrangement.spacedBy(2.dp),
                ) {
                    Text(t.buy.purse, style = MaterialTheme.typography.labelMedium, color = c.muted)
                    Text(
                        money(balance),
                        style = MaterialTheme.typography.headlineMedium,
                        color = if (balance < 0) c.danger else c.ink,
                    )
                    if (balance < 0) {
                        Text(
                            t.buy.purseOwed,
                            style = MaterialTheme.typography.labelMedium,
                            color = c.muted,
                        )
                    }
                    if (purseIssued > 0) {
                        Text(
                            t.buy.purseOf(money(purseIssued), money(purseSpent)),
                            style = MaterialTheme.typography.labelMedium,
                            color = c.muted,
                        )
                    }
                }
            }
        }

        if (error.isNotEmpty()) {
            item { Text(error, style = MaterialTheme.typography.bodyMedium, color = c.danger) }
        }
        if (done.isNotEmpty()) {
            item { Text(done, style = MaterialTheme.typography.bodyMedium, color = c.accent) }
        }

        val sent = order
        if (sent != null) {
            // ---- The list somebody sent ----
            //
            // ⚠️ **Asked and brought are shown together.** "Asked for ten,
            // brought six" is the sentence this whole document exists to make
            // possible; a screen showing only the result would leave the same
            // silence the buying had before.
            item {
                Column {
                    Text(
                        t.buy.orderTitle(sent.forDate),
                        style = MaterialTheme.typography.titleLarge,
                        color = c.ink,
                    )
                    if (sent.createdBy.isNotEmpty()) {
                        Text(
                            t.buy.orderFrom(sent.createdBy),
                            style = MaterialTheme.typography.labelMedium,
                            color = c.muted,
                        )
                    }
                }
            }
            items(sent.lines, key = { it.id }) { line ->
                OrderLineCard(
                    line = line,
                    pack = catalog.firstOrNull { it.id == line.ingredientId },
                    onGot = { gotQty, gotPrice, inPacks ->
                        mark(line.id) { o, l ->
                            api.markOrderLine(o, l, qty = gotQty, price = gotPrice, pack = inPacks)
                        }
                    },
                    onMissing = { mark(line.id) { o, l -> api.markOrderLine(o, l, missing = true) } },
                    onUndo = { mark(line.id) { o, l -> api.markOrderLine(o, l, clear = true) } },
                )
            }
            item {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(
                        t.buy.whereTitle,
                        style = MaterialTheme.typography.titleLarge,
                        color = c.ink,
                    )
                    GlassField(supplier, { supplier = it }, t.buy.wherePlaceholder)
                    PrimaryButton(t.buy.finish, busy = busy) { finish() }
                    Text(
                        t.buy.sendHint,
                        style = MaterialTheme.typography.labelMedium,
                        color = c.muted,
                    )
                }
            }
        } else {
            // ---- What the kitchen is short of ----
            if (lines.isEmpty()) {
                item {
                    Text(
                        t.buy.shortTitle,
                        Modifier.statusBarsPaddingIfNoPurse(purseBalance),
                        style = MaterialTheme.typography.titleLarge,
                        color = c.ink,
                    )
                }
                if (shortlist.isEmpty()) {
                    item {
                        Text(
                            t.buy.nothingShort,
                            style = MaterialTheme.typography.bodyMedium,
                            color = c.muted,
                        )
                    }
                }
                items(shortlist, key = { it.ingredientId }) { row ->
                    PickRow(
                        title = row.name,
                        subtitle = "${t.buy.onHand(qty(row.onHand), row.unit)} · " +
                            t.buy.need(qty(row.suggested), row.unit),
                    ) {
                        add(
                            ingredientId = row.ingredientId,
                            name = row.name,
                            unit = row.unit,
                            lastPrice = row.price,
                            // ⚠️ Pre-filled with what is short, **not** locked to
                            // it: a market sells what it has, and a buyer who
                            // came back with more must be able to say so.
                            qtyText = qty(row.suggested),
                            priceText = if (row.price > 0) money(row.price).replace(" ", "") else "",
                        )
                    }
                }
            }

            // ---- The run being written ----
            if (lines.isNotEmpty()) {
                item {
                    Text(
                        t.buy.basketTitle,
                        style = MaterialTheme.typography.titleLarge,
                        color = c.ink,
                    )
                }
                items(lines, key = { it.key }) { l ->
                    DraftCard(
                        line = l,
                        onChange = { updated ->
                            val i = lines.indexOfFirst { it.key == updated.key }
                            if (i >= 0) lines[i] = updated
                        },
                        onRemove = { lines.removeAll { it.key == l.key } },
                    )
                }
            }

            // ---- Anything else the market had ----
            //
            // ⚠️ Hidden while a list is open: a buyer recording the same crate
            // twice — once as a ticked line and once as a free-form row — would
            // raise the shelf twice, and nothing on any screen would say so.
            item {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(
                        t.buy.addTitle,
                        style = MaterialTheme.typography.titleLarge,
                        color = c.ink,
                    )
                    GlassField(query, { query = it }, t.buy.searchPlaceholder)
                }
            }
            items(matches, key = { it.id }) { row ->
                PickRow(
                    title = row.name,
                    subtitle = if (row.lastPrice > 0) money(row.lastPrice) else "",
                ) {
                    add(
                        ingredientId = row.id,
                        name = row.name,
                        unit = row.unit,
                        lastPrice = row.lastPrice,
                        priceText = if (row.lastPrice > 0) money(row.lastPrice).replace(" ", "") else "",
                        packName = row.packName,
                        packQty = row.packQty,
                    )
                }
            }
            if (unknown.isNotEmpty()) {
                item {
                    PickRow(title = t.buy.addNew(unknown), subtitle = "", icon = Icons.Rounded.AddCircleOutline) {
                        add(ingredientId = null, name = unknown)
                    }
                }
            }

            if (lines.isNotEmpty()) {
                item {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(
                            t.buy.whereTitle,
                            style = MaterialTheme.typography.titleLarge,
                            color = c.ink,
                        )
                        GlassField(supplier, { supplier = it }, t.buy.wherePlaceholder)
                        Row(
                            Modifier.fillMaxWidth().padding(top = 6.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text(
                                t.buy.total,
                                Modifier.weight(1f),
                                style = MaterialTheme.typography.bodyLarge,
                                color = c.ink,
                            )
                            Text(
                                money(total),
                                style = MaterialTheme.typography.titleLarge,
                                color = c.ink,
                            )
                        }
                        PrimaryButton(t.buy.send, busy = busy) { send() }
                        // ⚠️ Said before the button rather than after the fact:
                        // this raises the shelf and rewrites prices the moment it
                        // lands, and the person pressing it should know that is
                        // what they are doing.
                        Text(
                            t.buy.sendHint,
                            style = MaterialTheme.typography.labelMedium,
                            color = c.muted,
                        )
                    }
                }
            }
        }
    }
}

/** ⚠️ The purse clears the status bar when it is drawn; without it the first
 *  heading sat under the clock. */
private fun Modifier.statusBarsPaddingIfNoPurse(purse: Double?): Modifier =
    if (purse == null) this.then(Modifier.statusBarsPadding().padding(top = 8.dp)) else this

@Composable
private fun DraftCard(line: BuyDraft, onChange: (BuyDraft) -> Unit, onRemove: () -> Unit) {
    val c = KeelTheme.colors
    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp)).padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                line.name,
                Modifier.weight(1f),
                style = MaterialTheme.typography.bodyLarge,
                color = c.ink,
            )
            Icon(
                Icons.Rounded.Close,
                null,
                tint = c.muted,
                modifier = Modifier.size(20.dp).clickable(onClick = onRemove),
            )
        }
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                UnitLabel(line.unit, line.packName, line.packQty, line.pack) {
                    onChange(line.copy(pack = !line.pack))
                }
                GlassField(
                    value = line.qty,
                    onValueChange = { onChange(line.copy(qty = it.replace(',', '.'))) },
                    placeholder = "",
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                )
            }
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(t.buy.price, style = MaterialTheme.typography.labelMedium, color = c.muted)
                GlassField(
                    value = line.price,
                    onValueChange = { onChange(line.copy(price = it.filter { ch -> ch.isDigit() })) },
                    placeholder = "",
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                )
                // ⚠️ **The guard, and the only one this figure has.**
                if (line.lastPrice > 0) {
                    Text(
                        t.buy.lastPrice(money(line.lastPrice)),
                        style = MaterialTheme.typography.labelSmall,
                        color = c.muted,
                    )
                }
            }
        }
    }
}

/** One line of the list somebody sent, and what came back of it.
 *
 *  ⚠️ **"Could not get it" is its own answer, not a quantity of zero.** A line
 *  nobody touched and one somebody looked for and could not find are different
 *  facts, and only the second is worth ringing a supplier about. */
@Composable
private fun OrderLineCard(
    line: uz.keel.team.data.ShoppingLine,
    pack: BuyCatalogRow?,
    onGot: (Double, Double, Boolean) -> Unit,
    onMissing: () -> Unit,
    onUndo: () -> Unit,
) {
    val c = KeelTheme.colors
    var gotQty by remember(line.id) {
        mutableStateOf(if (line.gotQty > 0) qty(line.gotQty) else "")
    }
    var price by remember(line.id) {
        mutableStateOf(if (line.price > 0) money(line.price).replace(" ", "") else "")
    }
    /** ⚠️ Off by default even where a packaging exists: the list asked in the
     *  store's unit, so the figure in front of the buyer is in that unit until
     *  they say otherwise. */
    var inPacks by remember(line.id) { mutableStateOf(false) }
    val settled = line.gotAt.isNotEmpty()

    Column(
        Modifier
            .fillMaxWidth()
            .glass(c, RoundedCornerShape(18.dp))
            .padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                line.name,
                Modifier.weight(1f),
                style = MaterialTheme.typography.bodyLarge,
                color = if (settled) c.accent else c.ink,
            )
            Text(
                t.buy.asked(qty(line.qty), line.unit),
                style = MaterialTheme.typography.labelMedium,
                color = c.muted,
            )
        }

        if (line.missing) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    t.buy.wasMissing,
                    Modifier.weight(1f),
                    style = MaterialTheme.typography.labelMedium,
                    color = c.muted,
                )
                Text(
                    t.buy.undo,
                    Modifier.clickable(onClick = onUndo),
                    style = MaterialTheme.typography.labelMedium,
                    color = c.accent,
                )
            }
            return@Column
        }

        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                UnitLabel(line.unit, pack?.packName ?: "", pack?.packQty ?: 0.0, inPacks) {
                    inPacks = !inPacks
                }
                GlassField(
                    value = gotQty,
                    onValueChange = { gotQty = it.replace(',', '.') },
                    placeholder = "",
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                )
            }
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(t.buy.price, style = MaterialTheme.typography.labelMedium, color = c.muted)
                GlassField(
                    value = price,
                    onValueChange = { price = it.filter { ch -> ch.isDigit() } },
                    placeholder = "",
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                )
            }
        }

        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Box(Modifier.weight(1f)) {
                PrimaryButton(
                    label = if (settled) t.buy.changed else t.buy.got,
                    enabled = (gotQty.toDoubleOrNull() ?: 0.0) > 0,
                ) {
                    onGot(
                        gotQty.toDoubleOrNull() ?: 0.0,
                        price.toDoubleOrNull() ?: 0.0,
                        inPacks,
                    )
                }
            }
            uz.keel.design.GhostButton(t.buy.noneLeft, Modifier.weight(1f)) { onMissing() }
        }
    }
}
