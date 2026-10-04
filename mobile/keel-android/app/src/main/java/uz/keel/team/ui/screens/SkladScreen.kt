package uz.keel.team.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import uz.keel.design.GhostButton
import uz.keel.design.GlassField
import uz.keel.design.KeelTheme
import uz.keel.design.PrimaryButton
import uz.keel.design.glass
import uz.keel.design.qty
import uz.keel.team.data.ApiError
import uz.keel.team.data.KeelApi
import uz.keel.team.data.ShoppingOrder
import uz.keel.team.t

// Handing out what a request asks for.
//
// ⚠️ **Its own screen rather than a mode of the buyer's, because it is a
// different act.** A buyer stands at a stall and writes prices — the one thing
// that reprices every dish on the menu. A storekeeper takes things off a shelf
// that were bought once already, and a second price written here would enter the
// history as a purchase and recost everything the ingredient goes into. So this
// screen has quantities and no money on it at all, which is also why it is safe
// on a phone that lives in a store room.
//
// ⚠️ **Nothing here moves stock in an ordinary restaurant**, and that is the
// model rather than an omission: an ingredient has exactly one home
// (models/warehouse.go), so handing the barman a case of cola changes no
// balance, and the till writes it off when the drink is rung up. Recording an
// issue as stock leaving would subtract the same bottle twice, and the count
// that found the gap would blame the barman. Where the goods genuinely come from
// another branch the server builds the van (`dispatch`) — see
// handlers/buyorderflow.go.

@Composable
fun SkladScreen(api: KeelApi, bottomInset: PaddingValues) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()

    var orders by remember { mutableStateOf<List<ShoppingOrder>>(emptyList()) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    var done by remember { mutableStateOf("") }
    var tick by remember { mutableIntStateOf(0) }
    /** What has been typed but not yet sent, per line. ⚠️ Kept beside the
     *  server's answer rather than written into it: the field is a draft until
     *  the row is ticked, and a screen that edited the document in place would
     *  make an unsaved keystroke look like a fact. */
    val typed = remember { mutableStateMapOf<String, String>() }

    val loadFailed = t.sklad.loadFailed
    val sendFailed = t.sklad.sendFailed
    val shippedWord = t.sklad.shipped
    val nothingWord = t.sklad.noneGiven

    LaunchedEffect(tick) {
        try {
            // ⚠️ **Only what this phone answers.** The endpoint returns
            // everything this account has a part in — the branch's own market
            // lists included — and a storekeeper shown those would be looking at
            // somebody else's morning with buttons that refuse.
            orders = api.buyOrders(openOnly = true).orders
                .filter { it.fromStore && it.open }
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else loadFailed
        }
    }

    fun mark(order: ShoppingOrder, lineId: String, value: Double?, missing: Boolean) {
        scope.launch {
            try {
                val updated = when {
                    missing -> api.markOrderLine(order.id, lineId, missing = true)
                    value == null -> api.markOrderLine(order.id, lineId, clear = true)
                    else -> api.markOrderLine(order.id, lineId, qty = value)
                }
                orders = orders.map { if (it.id == updated.id) updated else it }
                done = ""
            } catch (e: Throwable) {
                error = if (e is ApiError) e.message else sendFailed
            }
        }
    }

    fun ship(order: ShoppingOrder) {
        if (order.lines.none { it.gotAt.isNotEmpty() && !it.missing && it.gotQty > 0 }) {
            error = nothingWord
            return
        }
        busy = true
        error = ""
        scope.launch {
            try {
                api.shipOrder(order.id)
                done = shippedWord
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
                Text(t.sklad.title, style = MaterialTheme.typography.titleLarge, color = c.ink)
                if (error.isNotEmpty()) {
                    Text(error, style = MaterialTheme.typography.bodyMedium, color = c.danger)
                }
                if (done.isNotEmpty()) {
                    Text(done, style = MaterialTheme.typography.bodyMedium, color = c.accent)
                }
            }
        }

        if (orders.isEmpty()) {
            item {
                Text(t.sklad.empty, style = MaterialTheme.typography.bodyMedium, color = c.muted)
            }
        }

        orders.forEach { order ->
            item(key = "head-" + order.id) {
                Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Text(
                        order.forDate,
                        style = MaterialTheme.typography.titleLarge,
                        color = c.ink,
                    )
                    if (order.createdBy.isNotEmpty()) {
                        Text(
                            t.sklad.asked(order.createdBy),
                            style = MaterialTheme.typography.labelMedium,
                            color = c.muted,
                        )
                    }
                }
            }
            items(order.lines, key = { "l-" + order.id + "-" + it.id }) { l ->
                val ticked = l.gotAt.isNotEmpty()
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
                            Text(
                                // ⚠️ What was asked for stays on the row after
                                // it is answered: "asked for ten, gave six" is
                                // the sentence this document exists to make
                                // possible, and a row that showed only the
                                // answer would put back the silence.
                                t.zakup.need(qty(l.qty), l.unit) +
                                    if (l.missing) {
                                        " · " + t.sklad.none
                                    } else if (ticked) {
                                        " · " + t.sklad.give + " " + qty(l.gotQty)
                                    } else {
                                        ""
                                    },
                                style = MaterialTheme.typography.labelMedium,
                                color = if (l.missing) c.danger else c.muted,
                            )
                        }
                    }
                    if (ticked) {
                        GhostButton(t.sklad.undo, Modifier.fillMaxWidth()) {
                            typed.remove(l.id)
                            mark(order, l.id, null, missing = false)
                        }
                    } else {
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(10.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Box(Modifier.weight(1f)) {
                                GlassField(
                                    value = typed[l.id] ?: "",
                                    onValueChange = { v ->
                                        typed[l.id] = v.replace(',', '.')
                                    },
                                    // Pre-filled with what was asked for, so the
                                    // ordinary case — giving exactly that — is
                                    // one tap rather than typing.
                                    placeholder = qty(l.qty),
                                    keyboardOptions = KeyboardOptions(
                                        keyboardType = KeyboardType.Decimal,
                                    ),
                                )
                            }
                            Text(
                                l.unit,
                                style = MaterialTheme.typography.labelMedium,
                                color = c.muted,
                            )
                        }
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            Box(Modifier.weight(1f)) {
                                PrimaryButton(t.sklad.give) {
                                    val v = typed[l.id]?.toDoubleOrNull() ?: l.qty
                                    if (v > 0) mark(order, l.id, v, missing = false)
                                }
                            }
                            // ⚠️ **"There was none" is its own answer, not a
                            // quantity of zero.** A line nobody touched and one
                            // somebody looked for and could not find are
                            // different facts, and only the second is worth
                            // telling anybody about.
                            GhostButton(t.sklad.none, Modifier.weight(1f)) {
                                mark(order, l.id, null, missing = true)
                            }
                        }
                    }
                }
            }
            item(key = "ship-" + order.id) {
                Box(Modifier.widthIn(max = 480.dp)) {
                    PrimaryButton(t.sklad.ship, busy = busy) { ship(order) }
                }
            }
        }
    }
}
