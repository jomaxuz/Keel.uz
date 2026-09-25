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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.rounded.Warehouse
import androidx.compose.foundation.background
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.sp
import uz.keel.design.BigNumberStyle
import uz.keel.design.glassSheet
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import uz.keel.design.Chip
import uz.keel.design.GlassField
import uz.keel.design.KeelTheme
import uz.keel.design.Money
import uz.keel.design.PrimaryButton
import uz.keel.design.glass
import uz.keel.design.qty
import uz.keel.team.CountDraft
import uz.keel.team.data.ApiError
import uz.keel.team.data.KeelApi
import uz.keel.team.data.SavedCount
import uz.keel.team.data.StocktakeSheetRow
import uz.keel.team.data.Staff
import uz.keel.team.data.Warehouse
import uz.keel.team.t
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

// Counting the store, on the phone that is already in the counter's hand.
//
// ⚠️ **The count happened in the store and was typed in the office.** Somebody
// walked the shelves with a clipboard, carried the paper to a computer and
// entered forty numbers a second time. A number written twice is wrong the
// second time, and the error lands in the one figure the whole module exists to
// produce — the variance — where it cannot be told apart from a shortfall. The
// web page (`/staff/stock`) was the first answer to that and it works; what it
// could not do is be found. An employee had to be told a URL and keep it in a
// browser tab, which is the same reason attendance moved off a web page and into
// this app.
//
// ⚠️ **The expected figure is not on this screen, and it is not on the wire
// either** (handlers/stocktake.go). Both screens used to hide it until a number
// had been typed; the rule was right and the blind was cosmetic — type anything,
// read the figure, correct the entry. The server no longer sends it, and the
// variance arrives with the saved count, when it is a finding rather than a
// target. A count is insert-only, so nothing that comes back can be edited to
// meet it.
//
// ⚠️ **Nothing here is a second calculation.** The sheet, the expected figures
// and the variance are the panel's own (`stocktakeSheet`, `saveStocktake`), or
// the argument moves from the shelf to two of our screens.

/** Whether this account counts the store.
 *
 *  ⚠️ **The permission, never the job title.** `Staff.Position` is free text
 *  (models/staffrole.go), so reading "Texnolog" off it would hand the baseline
 *  every later shortfall is measured from to whoever spells it the same way. The
 *  shipped Texnolog role is the one that carries `stock`, and a restaurant that
 *  unticks it has said something this screen must obey. */
fun canCountHere(staff: Staff): Boolean = staff.perms.contains("stock")

/** The day a timestamp fell on, in the phone's own zone.
 *
 *  ⚠️ **Parsed, never sliced.** Every date the driver hands back is UTC
 *  (CLAUDE.md), so `since.take(10)` is the previous day for every count taken
 *  after 05:00 local — which is every count. It would read as an ordinary date
 *  and be wrong by one in the one sentence that says how far back the
 *  measurement reaches. */
private fun dayOf(iso: String?): String? {
    if (iso.isNullOrEmpty()) return null
    return runCatching {
        Instant.parse(iso).atZone(ZoneId.systemDefault())
            .format(DateTimeFormatter.ofPattern("dd.MM.yyyy"))
    }.getOrNull()
}

@Composable
fun SanoqScreen(api: KeelApi, draft: CountDraft, bottomInset: PaddingValues) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()

    var stores by remember { mutableStateOf<List<Warehouse>>(emptyList()) }
    var rows by remember { mutableStateOf<List<StocktakeSheetRow>>(emptyList()) }
    var since by remember { mutableStateOf<String?>(null) }
    var query by remember { mutableStateOf("") }
    /** Show only what has not been counted yet. ⚠️ A filter rather than a
     *  reordering: rows that moved as they were filled would take the shelf's
     *  order away from somebody walking along it, which is the order they are
     *  counting in. */
    var onlyLeft by remember { mutableStateOf(false) }
    var note by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    var saved by remember { mutableStateOf<SavedCount?>(null) }
    var tick by remember { mutableIntStateOf(0) }

    val store = draft.warehouse
    // ⚠️ Held by the process, not by this screen: a count is forty numbers typed
    // over half an hour, and tapping another tab to check something is the most
    // ordinary thing somebody does in the middle of one. See CountDraft.
    val typed = draft.of(store)

    val loadFailed = t.sanoq.loadFailed
    val sendFailed = t.sanoq.sendFailed

    LaunchedEffect(Unit) {
        try {
            stores = api.warehouses().warehouses
        } catch (e: Throwable) {
            // ⚠️ Not fatal, and not reported. A restaurant that never split its
            // stores has none, and the undivided store is still countable — an
            // error here would refuse a count over a list that was empty anyway.
            stores = emptyList()
        }
    }

    LaunchedEffect(store, tick) {
        try {
            val sheet = api.stocktakeSheet(store)
            rows = sheet.rows
            since = sheet.since
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else loadFailed
        }
    }

    /** Only what was actually typed.
     *
     *  ⚠️ **A blank box is "not reached yet", never zero.** Saving untouched
     *  rows as zero reads the next morning as a catastrophic shortfall, and it
     *  is the classic way a half-finished count gets saved anyway. */
    val filled = rows.mapNotNull { r ->
        val v = typed[r.ingredientId]?.trim()?.toDoubleOrNull()
        if (v != null && v >= 0) r.ingredientId to v else null
    }.toMap()

    val q = query.trim()
    val shown = rows.filter {
        (q.isEmpty() || it.name.contains(q, true)) &&
            (!onlyLeft || it.ingredientId !in filled)
    }

    fun save() {
        if (filled.isEmpty()) return
        busy = true
        error = ""
        scope.launch {
            try {
                val res = api.saveStocktake(store, filled, note.trim())
                saved = res
                // ⚠️ Cleared only now, and only for this store: a failed save
                // that emptied the boxes would throw away the walk rather than
                // the draft.
                draft.clear(store)
                note = ""
                tick += 1
            } catch (e: Throwable) {
                error = if (e is ApiError) e.message else sendFailed
            } finally {
                busy = false
            }
        }
    }

    Box(Modifier.fillMaxSize()) {
        LazyColumn(
            Modifier.fillMaxSize().imePadding(),
            contentPadding = PaddingValues(
                start = 16.dp,
                end = 16.dp,
                top = 8.dp,
                // Room for the save bar that floats over the end of the list.
                bottom = bottomInset.calculateBottomPadding() + if (filled.isNotEmpty()) 96.dp else 16.dp,
            ),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            // ⚠️ **What the count is measured from, in words.** Without it the
            // variance looks like a running balance the system has been keeping
            // all along — it is not: it is the last count plus four recorded
            // facts, and the first count of a store reaches back to everything
            // that ever arrived.
            item {
                MarketHeader(t.sanoq.title, dayOf(since)?.let { t.sanoq.since(it) } ?: t.sanoq.neverCounted)
            }
            if (error.isNotEmpty()) item { MarketBanner(error, error = true) }

            // ⚠️ **A count is one room.** Handing somebody walking into the bar a
            // list that also holds forty kitchen ingredients is how counts get
            // abandoned halfway — and a half-counted list saved is a shortfall
            // report about a shelf nobody looked at.
            //
            // ⚠️ **Chips, not a drop-down.** A restaurant has two or three rooms;
            // a picker hid them behind a tap, and the room being counted was a
            // word in small print instead of the thing the screen is about.
            if (stores.isNotEmpty()) {
                item {
                    val mainName = t.sanoq.mainStore
                    val all = buildList {
                        add("" to mainName)
                        stores.forEach { add(it.id to it.name) }
                    }
                    LazyRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        items(all, key = { it.first.ifEmpty { "main" } }) { (id, name) ->
                            Chip(name, store == id, icon = Icons.Rounded.Warehouse) {
                                // ⚠️ The draft is not cleared: switching stores to
                                // look something up and switching back is one
                                // person's normal count, and the numbers belong to
                                // the shelf either way.
                                draft.warehouse = id
                                query = ""
                                saved = null
                            }
                        }
                    }
                }
            }

            if (rows.isNotEmpty()) {
                // How far along the shelf this is. ⚠️ Counted rows rather than a
                // percentage: the question somebody stops to ask is "how many are
                // left", and they are about to walk to each one.
                item {
                    Column(
                        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
                        verticalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Column(Modifier.weight(1f)) {
                                Text(
                                    "${filled.size} / ${rows.size}",
                                    style = BigNumberStyle.copy(fontSize = 28.sp, lineHeight = 32.sp),
                                    color = if (filled.size == rows.size) c.ready else c.ink,
                                )
                                Text(
                                    t.sanoq.progress(filled.size, rows.size),
                                    style = MaterialTheme.typography.labelMedium,
                                    color = c.muted,
                                )
                            }
                            Chip(t.sanoq.onlyLeft, on = onlyLeft) { onlyLeft = !onlyLeft }
                        }
                        ProgressLine(filled.size, rows.size)
                    }
                }
                item {
                    GlassField(
                        query,
                        { query = it },
                        t.sanoq.search,
                        trailing = {
                            Icon(Icons.Rounded.Search, null, tint = c.muted, modifier = Modifier.size(20.dp))
                        },
                    )
                }
            }

            items(shown, key = { it.ingredientId }) { r ->
                val text = typed[r.ingredientId] ?: ""
                val done = r.ingredientId in filled
                Row(
                    Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp))
                        .padding(start = 12.dp, end = 8.dp, top = 8.dp, bottom = 8.dp),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    // ⚠️ A mark rather than a colour on the whole row: the row
                    // still has to be readable and re-typeable, and a filled row
                    // somebody corrects is the ordinary case.
                    Box(
                        Modifier.size(26.dp).clip(CircleShape).background(if (done) c.ready else c.line),
                        contentAlignment = Alignment.Center,
                    ) {
                        if (done) Icon(Icons.Rounded.Check, null, tint = Color.White, modifier = Modifier.size(16.dp))
                    }
                    Text(
                        r.name,
                        Modifier.weight(1f),
                        style = MaterialTheme.typography.bodyLarge,
                        color = c.ink,
                    )
                    GlassField(
                        value = text,
                        // ⚠️ The comma is what the phone's decimal key produces
                        // here, and `"1,5".toDouble()` is null — the row would
                        // silently drop out of the count as "not reached yet".
                        onValueChange = { v -> typed[r.ingredientId] = v.replace(',', '.') },
                        placeholder = t.sanoq.qtyHint,
                        modifier = Modifier.width(140.dp),
                        // ⚠️ **"Next" walks the shelf.** The keyboard's own key
                        // moves to the row below, so forty numbers are typed
                        // without reaching up to tap forty boxes.
                        keyboardOptions = KeyboardOptions(
                            keyboardType = KeyboardType.Decimal,
                            imeAction = ImeAction.Next,
                        ),
                        trailing = {
                            Text(r.unit, Modifier.padding(end = 12.dp), style = MaterialTheme.typography.labelMedium, color = c.muted)
                        },
                    )
                }
            }

            if (rows.isEmpty()) {
                item {
                    Text(t.sanoq.empty, style = MaterialTheme.typography.bodyMedium, color = c.muted)
                }
            } else if (shown.isEmpty()) {
                item {
                    Text(t.sanoq.allDone, style = MaterialTheme.typography.bodyMedium, color = c.muted)
                }
            }

            if (filled.isNotEmpty()) {
                // ⚠️ **Always offered, never demanded.** It used to appear only
                // when something disagreed, which this screen cannot know any more
                // — and could not be made to know without handing back the figures
                // the count exists to test. A box that is always there says
                // nothing about whether the count matched; a box that appeared
                // would say everything.
                item { GlassField(note, { note = it }, t.sanoq.notePlaceholder) }
            }

            // ---- What it came to ----
            //
            // ⚠️ **Shown after the save and only after it.** This is the one place
            // the counter meets the expected figures, and by now the count has been
            // written and cannot be changed — so it is a finding rather than a
            // target, and it is worth reading here rather than in an office
            // tomorrow: the shelf is still an arm's length away, and the commonest
            // cause of a variance is a bag somebody counted from the wrong row.
            saved?.let { res ->
                val names = rows.associate { it.ingredientId to it.name }
                val off = res.lines.filter { it.diff != 0.0 }.sortedBy { it.value }
                item {
                    Column(
                        Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(20.dp))
                            .background(if (res.value < 0) c.danger.copy(alpha = 0.12f) else c.ready.copy(alpha = 0.12f))
                            .padding(16.dp),
                        verticalArrangement = Arrangement.spacedBy(4.dp),
                    ) {
                        Text(t.sanoq.savedTitle, style = MaterialTheme.typography.titleLarge, color = c.ink)
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(6.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text(t.sanoq.savedTotal, style = MaterialTheme.typography.bodyMedium, color = c.inkSoft)
                            Money(
                                res.value.toDouble(),
                                color = if (res.value < 0) c.danger else c.ink,
                                style = MaterialTheme.typography.titleMedium,
                            )
                        }
                        if (off.isEmpty()) {
                            Text(t.sanoq.noDiff, style = MaterialTheme.typography.bodyMedium, color = c.ready)
                        }
                    }
                }
                items(off, key = { "d-" + it.ingredientId }) { l ->
                    Row(
                        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(16.dp))
                            .padding(horizontal = 12.dp, vertical = 10.dp),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(Modifier.weight(1f)) {
                            Text(
                                names[l.ingredientId] ?: l.ingredientId,
                                style = MaterialTheme.typography.bodyLarge,
                                color = c.ink,
                            )
                            // Counted beside expected, in that order: what somebody
                            // walked up to and looked at is the fact, and the other
                            // number is what the books thought.
                            Text(
                                t.sanoq.countedVs(qty(l.counted), qty(l.expected)),
                                style = MaterialTheme.typography.labelMedium,
                                color = c.muted,
                            )
                        }
                        Pill(
                            (if (l.diff > 0) "+" else "") + qty(l.diff),
                            if (l.diff < 0) c.danger else c.ready,
                        )
                    }
                }
            }
        }

        // ⚠️ **The save button floats.** At the end of forty rows it was forty
        // rows away, and a counter who cannot see it wonders whether the numbers
        // are being kept at all.
        if (filled.isNotEmpty()) {
            Box(
                Modifier
                    .align(Alignment.BottomCenter)
                    .padding(bottom = bottomInset.calculateBottomPadding())
                    .padding(horizontal = 12.dp, vertical = 8.dp)
                    .fillMaxWidth()
                    .glassSheet(c, RoundedCornerShape(22.dp))
                    .padding(10.dp),
            ) {
                PrimaryButton(t.sanoq.save(filled.size), icon = Icons.Rounded.Check, busy = busy) { save() }
            }
        }
    }
}
