package uz.keel.team.ui.screens

import androidx.compose.foundation.background
import androidx.compose.animation.animateContentSize
import androidx.compose.foundation.border
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.rounded.AccountBalanceWallet
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.ExpandLess
import androidx.compose.material.icons.rounded.ExpandMore
import androidx.compose.material.icons.rounded.Search
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.sp
import uz.keel.design.BigNumberStyle
import uz.keel.design.GhostButton
import uz.keel.design.MoneyStyle
import uz.keel.design.glassSheet
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
    val shippedWord = t.zakup.statusShipped

    LaunchedEffect(tick) {
        try {
            shortlist = api.buyList().groups.flatMap { it.rows }
            catalog = api.buyCatalog().ingredients
            api.buyBalance().let {
                purseBalance = it.balance
                purseIssued = it.issued
                purseSpent = it.spent
            }
            // ⚠️ **The market half only.** The endpoint returns everything this
            // account has a part in, and a store request on a buyer's screen
            // would be a list of things he is being asked to go and buy that are
            // already in the building.
            order = api.buyOrders(openOnly = true).orders
                .firstOrNull { !it.fromStore && it.open }
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
                // ⚠️ **This no longer writes the delivery.** What reaches a
                // shelf is what somebody at the restaurant counted, so the trip
                // ends here as "on its way" and the purchase is written when the
                // person who asked signs for it. A delivery created at the
                // market would put food on a shelf while it was still in a bag
                // on a bus, and every later correction would be
                // indistinguishable from a theft. See handlers/buyorderflow.go.
                val res = api.shipOrder(id)
                supplier = ""
                done = if (res.already) alreadySent else shippedWord
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
    val chosen = lines.mapNotNull { it.ingredientId }.toSet()
    val matches = if (q.isEmpty()) emptyList() else
        catalog.filter { it.name.contains(q, ignoreCase = true) && it.id !in chosen }.take(12)
    // ⚠️ **Offered rather than refused.** A buyer who cannot record half a run
    // stops recording any of it — the lesson this product paid for with the
    // supplier field and the void reason. The server matches the name against
    // what exists before inventing anything, and marks what it does invent.
    val unknown = if (q.length >= 2 && catalog.none { it.name.equals(q, true) }) q else ""
    /** The line whose boxes are open. ⚠️ **One at a time.** Eight lines each with
     *  two boxes and two buttons was a wall of fields at a stall; closed, a line
     *  is one row that says what happened to it, and the next one opens itself. */
    var openLine by remember { mutableStateOf<String?>(null) }

    Box(Modifier.fillMaxSize()) {
        LazyColumn(
            Modifier.fillMaxSize().imePadding(),
            contentPadding = PaddingValues(
                start = 16.dp,
                end = 16.dp,
                top = 8.dp,
                // Room for the bar that floats over the end of the list.
                bottom = bottomInset.calculateBottomPadding() + 96.dp,
            ),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item { MarketHeader(t.buy.title) }

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
                item { PurseCard(balance, purseIssued, purseSpent) }
            }

            if (error.isNotEmpty()) item { MarketBanner(error, error = true) }
            if (done.isNotEmpty()) item { MarketBanner(done, error = false) }

            val sent = order
            if (sent != null) {
                // ---- The list somebody sent ----
                //
                // ⚠️ **Asked and brought are shown together.** "Asked for ten,
                // brought six" is the sentence this whole document exists to make
                // possible; a screen showing only the result would leave the same
                // silence the buying had before.
                val settledCount = sent.lines.count { it.gotAt.isNotEmpty() || it.missing }
                item {
                    Column(
                        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
                        verticalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        Text(t.buy.orderTitle(sent.forDate), style = MaterialTheme.typography.titleMedium, color = c.ink)
                        if (sent.createdBy.isNotEmpty()) {
                            Text(t.buy.orderFrom(sent.createdBy), style = MaterialTheme.typography.labelMedium, color = c.muted)
                        }
                        ProgressLine(settledCount, sent.lines.size)
                        Text(
                            t.buy.progress(settledCount, sent.lines.size),
                            style = MaterialTheme.typography.labelMedium,
                            color = c.muted,
                        )
                    }
                }
                val firstOpen = openLine ?: sent.lines.firstOrNull { it.gotAt.isEmpty() && !it.missing }?.id
                items(sent.lines, key = { it.id }) { line ->
                    OrderLineCard(
                        line = line,
                        pack = catalog.firstOrNull { it.id == line.ingredientId },
                        expanded = line.id == firstOpen,
                        onExpand = { openLine = if (line.id == firstOpen) "" else line.id },
                        onGot = { gotQty, gotPrice, inPacks ->
                            openLine = sent.lines.firstOrNull {
                                it.id != line.id && it.gotAt.isEmpty() && !it.missing
                            }?.id ?: ""
                            mark(line.id) { o, l ->
                                api.markOrderLine(o, l, qty = gotQty, price = gotPrice, pack = inPacks)
                            }
                        },
                        onMissing = {
                            openLine = sent.lines.firstOrNull {
                                it.id != line.id && it.gotAt.isEmpty() && !it.missing
                            }?.id ?: ""
                            mark(line.id) { o, l -> api.markOrderLine(o, l, missing = true) }
                        },
                        onUndo = { mark(line.id) { o, l -> api.markOrderLine(o, l, clear = true) } },
                    )
                }
                item {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        SectionTitle(t.buy.whereTitle)
                        GlassField(supplier, { supplier = it }, t.buy.wherePlaceholder)
                        Text(t.buy.sendHint, style = MaterialTheme.typography.labelMedium, color = c.muted)
                    }
                }
            } else {
                // ---- The run being written ----
                item { SectionTitle(t.buy.basketTitle, lines.size) }
                if (lines.isEmpty()) {
                    item {
                        Text(
                            t.buy.emptyBasket,
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
                    DraftCard(
                        line = l,
                        onChange = { updated ->
                            val i = lines.indexOfFirst { it.key == updated.key }
                            if (i >= 0) lines[i] = updated
                        },
                        onRemove = { lines.removeAll { it.key == l.key } },
                    )
                }

                // ---- Adding: search first, the shortage beneath it ----
                //
                // ⚠️ Hidden while a list is open: a buyer recording the same crate
                // twice — once as a ticked line and once as a free-form row — would
                // raise the shelf twice, and nothing on any screen would say so.
                item {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        SectionTitle(t.buy.addTitle)
                        GlassField(
                            query,
                            { query = it },
                            t.buy.searchPlaceholder,
                            trailing = {
                                Icon(Icons.Rounded.Search, null, tint = c.muted, modifier = Modifier.size(20.dp))
                            },
                        )
                    }
                }
                if (q.isEmpty()) {
                    val short = shortlist.filter { it.ingredientId !in chosen }
                    if (short.isNotEmpty()) {
                        item {
                            Text(t.buy.shortTitle, style = MaterialTheme.typography.labelMedium, color = c.muted)
                        }
                    } else if (lines.isEmpty()) {
                        item {
                            Text(t.buy.nothingShort, style = MaterialTheme.typography.bodyMedium, color = c.muted)
                        }
                    }
                    items(short, key = { "s-" + it.ingredientId }) { row ->
                        PickRow(
                            title = row.name,
                            subtitle = "${t.buy.onHand(qty(row.onHand), row.unit)} · " +
                                t.buy.need(qty(row.suggested), row.unit),
                        ) {
                            val pack = catalog.firstOrNull { it.id == row.ingredientId }
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
                                packName = pack?.packName ?: "",
                                packQty = pack?.packQty ?: 0.0,
                            )
                        }
                    }
                }
                items(matches, key = { it.id }) { row ->
                    PickRow(
                        title = row.name,
                        subtitle = if (row.lastPrice > 0) t.buy.lastPrice(money(row.lastPrice)) else "",
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
                            SectionTitle(t.buy.whereTitle)
                            GlassField(supplier, { supplier = it }, t.buy.wherePlaceholder)
                            // ⚠️ Said before the button rather than after the fact:
                            // this raises the shelf and rewrites prices the moment it
                            // lands, and the person pressing it should know that is
                            // what they are doing.
                            Text(t.buy.sendHint, style = MaterialTheme.typography.labelMedium, color = c.muted)
                        }
                    }
                }
            }
        }

        // ---- The one action, always in reach ----
        //
        // ⚠️ **Floating, not the last row of the list.** At the end of a long
        // list the button was a scroll away from every line, and a buyer at a
        // stall checks the total after each crate.
        val sentOrder = order
        if (sentOrder != null || lines.isNotEmpty()) {
            Row(
                Modifier
                    .align(Alignment.BottomCenter)
                    .padding(bottom = bottomInset.calculateBottomPadding())
                    .padding(horizontal = 12.dp, vertical = 8.dp)
                    .fillMaxWidth()
                    .glassSheet(c, RoundedCornerShape(22.dp))
                    .padding(10.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                if (sentOrder == null) {
                    Column(Modifier.padding(start = 6.dp)) {
                        Text(t.buy.total, style = MaterialTheme.typography.labelMedium, color = c.muted)
                        Text(money(total), style = MaterialTheme.typography.titleLarge.merge(MoneyStyle), color = c.ink)
                    }
                    Box(Modifier.weight(1f)) { PrimaryButton(t.buy.send, busy = busy) { send() } }
                } else {
                    Box(Modifier.weight(1f)) { PrimaryButton(t.buy.finish, busy = busy) { finish() } }
                }
            }
        }
    }
}

/** What the buyer is carrying, as the first thing on the screen. */
@Composable
private fun PurseCard(balance: Double, issued: Double, spent: Double) {
    val c = KeelTheme.colors
    Row(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(22.dp))
            .background(if (balance < 0) c.danger.copy(alpha = 0.12f) else c.accentSoft)
            .padding(16.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        Box(
            Modifier.size(44.dp).clip(CircleShape).background(if (balance < 0) c.danger else c.accent),
            contentAlignment = Alignment.Center,
        ) {
            Icon(Icons.Rounded.AccountBalanceWallet, null, tint = Color.White, modifier = Modifier.size(22.dp))
        }
        Column(Modifier.weight(1f)) {
            Text(t.buy.purse, style = MaterialTheme.typography.labelMedium, color = c.inkSoft)
            Text(
                money(balance),
                style = BigNumberStyle.copy(fontSize = 28.sp, lineHeight = 32.sp),
                color = if (balance < 0) c.danger else c.ink,
            )
            if (balance < 0) {
                Text(t.buy.purseOwed, style = MaterialTheme.typography.labelMedium, color = c.danger)
            }
            if (issued > 0) {
                Text(t.buy.purseOf(money(issued), money(spent)), style = MaterialTheme.typography.labelMedium, color = c.muted)
            }
        }
    }
}

@Composable
private fun DraftCard(line: BuyDraft, onChange: (BuyDraft) -> Unit, onRemove: () -> Unit) {
    val c = KeelTheme.colors
    val sum = (line.qty.toDoubleOrNull() ?: 0.0) * (line.price.toDoubleOrNull() ?: 0.0)
    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp)).padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(line.name, Modifier.weight(1f), style = MaterialTheme.typography.titleMedium, color = c.ink)
            if (sum > 0) {
                Text(money(sum), style = MaterialTheme.typography.bodyMedium.merge(MoneyStyle), color = c.ink)
            }
            Box(
                Modifier.padding(start = 8.dp).size(30.dp).clip(CircleShape).clickable(onClick = onRemove),
                contentAlignment = Alignment.Center,
            ) {
                Icon(Icons.Rounded.Close, null, tint = c.muted, modifier = Modifier.size(18.dp))
            }
        }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            QtyField(
                value = line.qty,
                onValueChange = { onChange(line.copy(qty = it)) },
                unit = line.unit,
                packName = line.packName,
                packQty = line.packQty,
                inPacks = line.pack,
                onToggle = { onChange(line.copy(pack = !line.pack)) },
                modifier = Modifier.weight(1f),
                placeholder = t.buy.qty(""),
            )
            PriceField(
                value = line.price,
                onValueChange = { onChange(line.copy(price = it)) },
                modifier = Modifier.weight(1f),
                placeholder = t.buy.price,
            )
        }
        // ⚠️ **The guard, and the only one this figure has.**
        if (line.lastPrice > 0) {
            Text(t.buy.lastPrice(money(line.lastPrice)), style = MaterialTheme.typography.labelSmall, color = c.muted)
        }
    }
}

/** One line of the list somebody sent, and what came back of it.
 *
 *  ⚠️ **"Could not get it" is its own answer, not a quantity of zero.** A line
 *  nobody touched and one somebody looked for and could not find are different
 *  facts, and only the second is worth ringing a supplier about.
 *
 *  ⚠️ **Closed, a line is one row that says what happened to it** — a tick and
 *  what came back, a cross, or nothing yet. Only the open one has boxes. */
@Composable
private fun OrderLineCard(
    line: uz.keel.team.data.ShoppingLine,
    pack: BuyCatalogRow?,
    expanded: Boolean,
    onExpand: () -> Unit,
    onGot: (Double, Double, Boolean) -> Unit,
    onMissing: () -> Unit,
    onUndo: () -> Unit,
) {
    val c = KeelTheme.colors
    var gotQty by remember(line.id, line.gotAt) {
        mutableStateOf(if (line.gotQty > 0) qty(line.gotQty) else qty(line.qty))
    }
    var price by remember(line.id, line.gotAt) {
        mutableStateOf(if (line.price > 0) money(line.price).replace(" ", "") else "")
    }
    /** ⚠️ Off by default even where a packaging exists: the list asked in the
     *  store's unit, so the figure in front of the buyer is in that unit until
     *  they say otherwise. */
    var inPacks by remember(line.id) { mutableStateOf(false) }
    val settled = line.gotAt.isNotEmpty() && !line.missing

    Column(
        Modifier
            .fillMaxWidth()
            .glass(c, RoundedCornerShape(18.dp))
            .then(if (expanded) Modifier.border(1.5.dp, c.accent.copy(alpha = 0.6f), RoundedCornerShape(18.dp)) else Modifier)
            .animateContentSize(),
    ) {
        Row(
            Modifier.fillMaxWidth().clickable(onClick = onExpand).padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            val (dotBg, dotIcon, dotTint) = when {
                line.missing -> Triple(c.line, Icons.Rounded.Close, c.muted)
                settled -> Triple(c.ready, Icons.Rounded.Check, Color.White)
                else -> Triple(c.accentSoft, null, c.accent)
            }
            Box(Modifier.size(28.dp).clip(CircleShape).background(dotBg), contentAlignment = Alignment.Center) {
                if (dotIcon != null) Icon(dotIcon, null, tint = dotTint, modifier = Modifier.size(16.dp))
            }
            Column(Modifier.weight(1f)) {
                Text(
                    line.name,
                    style = MaterialTheme.typography.bodyLarge,
                    color = if (line.missing) c.muted else c.ink,
                )
                Text(
                    when {
                        line.missing -> t.buy.wasMissing
                        settled -> "${qty(line.gotQty)} ${line.unit}" +
                            if (line.price > 0) " · ${money(line.price)}" else ""
                        else -> t.buy.asked(qty(line.qty), line.unit)
                    },
                    style = MaterialTheme.typography.labelMedium,
                    color = if (settled) c.ready else c.muted,
                )
            }
            if (line.missing || settled) {
                Text(
                    t.buy.undo,
                    Modifier.clip(CircleShape).clickable(onClick = onUndo).padding(horizontal = 8.dp, vertical = 6.dp),
                    style = MaterialTheme.typography.labelMedium,
                    color = c.accent,
                )
            } else {
                Icon(
                    if (expanded) Icons.Rounded.ExpandLess else Icons.Rounded.ExpandMore,
                    null,
                    tint = c.muted,
                    modifier = Modifier.size(22.dp),
                )
            }
        }

        if (expanded && !line.missing) {
            Column(
                Modifier.padding(start = 12.dp, end = 12.dp, bottom = 12.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    QtyField(
                        value = gotQty,
                        onValueChange = { gotQty = it },
                        unit = line.unit,
                        packName = pack?.packName ?: "",
                        packQty = pack?.packQty ?: 0.0,
                        inPacks = inPacks,
                        onToggle = { inPacks = !inPacks },
                        modifier = Modifier.weight(1f),
                        placeholder = t.buy.qty(""),
                    )
                    PriceField(
                        value = price,
                        onValueChange = { price = it },
                        modifier = Modifier.weight(1f),
                        placeholder = if ((pack?.lastPrice ?: 0.0) > 0) money(pack!!.lastPrice) else t.buy.price,
                    )
                }
                if ((pack?.lastPrice ?: 0.0) > 0) {
                    Text(t.buy.lastPrice(money(pack!!.lastPrice)), style = MaterialTheme.typography.labelSmall, color = c.muted)
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    GhostButton(t.buy.noneLeft, Modifier.weight(1f)) { onMissing() }
                    Box(Modifier.weight(1f)) {
                        PrimaryButton(
                            label = if (settled) t.buy.changed else t.buy.got,
                            icon = Icons.Rounded.Check,
                            enabled = (gotQty.toDoubleOrNull() ?: 0.0) > 0,
                        ) {
                            onGot(gotQty.toDoubleOrNull() ?: 0.0, price.toDoubleOrNull() ?: 0.0, inPacks)
                        }
                    }
                }
            }
        }
    }
}
