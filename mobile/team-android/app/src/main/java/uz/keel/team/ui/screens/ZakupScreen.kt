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
 *  ⚠️ **Narrower than the permission, and deliberately so.** A cashier holds
 *  `buyorder` because the till is where they first hear that something has run
 *  out — but their phone is not where a restaurant's buying gets planned, and a
 *  section they never use is a section that teaches them to ignore the app.
 *
 *  ⚠️ **Expressed in permissions, never in the role's name.** The spelling of a
 *  job title grants nothing (`models/staffrole.go`), so "management or the
 *  storekeeper" is asked as "may void, or counts the store" — the same test the
 *  till uses to mean management. */
fun canWriteHere(staff: Staff): Boolean =
    staff.perms.contains("buyorder") &&
        (staff.perms.contains("void") || staff.perms.contains("stock"))

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

    val loadFailed = t.zakup.loadFailed
    val sendFailed = t.zakup.sendFailed
    val sentWord = t.zakup.sent

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
            ),
        )
        query = ""
    }

    val ready = lines.filter { (it.qty.toDoubleOrNull() ?: 0.0) > 0 }

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
                done = sentWord(ready.size)
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
                        Text(
                            l.name,
                            Modifier.weight(1f),
                            style = MaterialTheme.typography.bodyLarge,
                            color = c.ink,
                        )
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
                subtitle = if (row.qty > 0) {
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
                    Text(
                        if (o.status == "done") t.zakup.statusDone else t.zakup.statusSent,
                        style = MaterialTheme.typography.labelMedium,
                        color = c.muted,
                    )
                }
            }
        }
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
                    t.zakup.previewBody(forDate),
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.muted,
                )
                Column(
                    Modifier.verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    ready.forEach { l ->
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                l.name,
                                Modifier.weight(1f),
                                style = MaterialTheme.typography.bodyLarge,
                                color = c.ink,
                            )
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
