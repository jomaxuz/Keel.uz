package uz.keel.owner.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.AutoAwesome
import androidx.compose.material.icons.rounded.ExpandLess
import androidx.compose.material.icons.rounded.ExpandMore
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import uz.keel.design.BigNumberStyle
import uz.keel.design.Chip
import uz.keel.design.GlassIconButton
import uz.keel.design.KeelTheme
import uz.keel.design.MoneyStyle
import uz.keel.design.ScreenHeader
import uz.keel.design.glass
import uz.keel.design.money
import uz.keel.owner.LocalPrefs
import uz.keel.owner.data.AdminStats
import uz.keel.owner.data.ApiError
import uz.keel.owner.data.Branch
import uz.keel.owner.data.BriefingCard
import uz.keel.owner.data.KeelApi
import uz.keel.owner.data.StaffRow
import uz.keel.owner.t

// The screen an owner opens twenty times a day.
//
// ⚠️ **One question, answered before the phone is fully out of the pocket: how
// is today going.** Everything else in the panel is a decision somebody sits
// down to make; this is the number they check between two other things, standing
// up, and it has to be readable in three seconds.
//
// ⚠️ **A figure with nothing beside it is not information.** "12 400 000" is
// neither good nor bad — it is only an answer next to yesterday's, which is why
// the comparison is fetched at the same time and shown in the same block.

/** A local calendar day, `YYYY-MM-DD`.
 *
 *  ⚠️ **The day boundary is local midnight, everywhere in this product.** Sent
 *  as a date rather than a moment: a timestamp would be re-read in UTC on the
 *  way and move the whole window five hours west — the trap this repository has
 *  been bitten by on the server and in the panel. */
private fun day(offset: Long = 0): String =
    java.time.LocalDate.now().plusDays(offset).toString()

@Composable
fun TodayScreen(api: KeelApi, branches: List<Branch>, bottomInset: PaddingValues) {
    val c = KeelTheme.colors
    val prefs = LocalPrefs.current
    val scope = rememberCoroutineScope()
    val branchId = prefs.branch.value

    var stats by remember { mutableStateOf<AdminStats?>(null) }
    var before by remember { mutableStateOf<AdminStats?>(null) }
    var staff by remember { mutableStateOf<List<StaffRow>>(emptyList()) }
    var cards by remember { mutableStateOf<List<BriefingCard>>(emptyList()) }
    var locked by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    val loadFailed = t.common.loadFailed

    suspend fun load() {
        busy = true
        try {
            val today = day()
            stats = api.stats(today, today, branchId)
            // ⚠️ Yesterday is a *different period*, asked separately: slicing a
            // week's series would give a different number, because the series is
            // orders and this block is money collected.
            before = runCatching { api.stats(day(-1), day(-1), branchId) }.getOrNull()
            staff = runCatching { api.staff(branchId) }.getOrDefault(emptyList())
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else loadFailed
        } finally { busy = false }
    }

    val lang = prefs.lang.value.code
    LaunchedEffect(branchId) { load() }
    // ⚠️ **Asked separately from the numbers, and that is the point.** This
    // screen is opened twenty times a day for the takings and read once for the
    // briefing; an answer the tariff does not include must not drag today's
    // revenue down with it, and a failure here must not blank the figures.
    LaunchedEffect(branchId, lang) {
        runCatching { api.insights(branchId, lang) }
            .onSuccess { cards = it.cards; locked = it.entitled == false }
            .onFailure { cards = emptyList() }
    }

    val period = stats?.period
    val revenue = period?.revenue ?: 0.0
    val orders = period?.orders ?: 0
    val average = if (orders > 0) revenue / orders else 0.0
    val yesterday = before?.period?.revenue ?: 0.0
    val change = if (yesterday > 0) Math.round((revenue - yesterday) / yesterday * 100).toInt() else null

    Column(Modifier.fillMaxSize()) {
        ScreenHeader(
            title = t.today.title,
            trailing = { GlassIconButton(Icons.Rounded.Refresh) { scope.launch { load() } } },
        )

        // ⚠️ **A restaurant with one branch never sees this.** A lens over a
        // single choice is a control that answers nothing, and it would sit
        // above the one number this screen exists for.
        if (branches.size > 1) {
            LazyRow(
                Modifier.fillMaxWidth().height(52.dp),
                contentPadding = PaddingValues(horizontal = 16.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                item {
                    Chip(t.today.branchAll, branchId.isEmpty()) { prefs.setBranch("") }
                }
                items(branches.size) { i ->
                    val b = branches[i]
                    Chip(b.name, branchId == b.id) { prefs.setBranch(b.id) }
                }
            }
        }

        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp)
                .padding(bottom = bottomInset.calculateBottomPadding() + 24.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            if (stats == null && error.isEmpty()) {
                Box(Modifier.fillMaxWidth().padding(40.dp), Alignment.Center) {
                    CircularProgressIndicator(color = c.accent)
                }
            }
            if (error.isNotEmpty()) {
                Text(error, color = c.danger, style = MaterialTheme.typography.bodyMedium)
            }

            if (stats != null) {
                // The one number, in the size it deserves.
                Column(
                    Modifier.fillMaxWidth().glass(c, RoundedCornerShape(22.dp)).padding(18.dp),
                    verticalArrangement = Arrangement.spacedBy(4.dp),
                ) {
                    Text(t.today.revenue, style = MaterialTheme.typography.labelMedium, color = c.muted)
                    Text(money(revenue), style = BigNumberStyle, color = c.ink)
                    Text(
                        if (change == null) t.today.noYesterday else t.today.vsYesterday(change),
                        style = MaterialTheme.typography.labelMedium,
                        color = when {
                            change == null -> c.muted
                            change >= 0 -> c.ready
                            else -> c.danger
                        },
                    )
                }

                Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    Tile(t.today.orders, orders.toString(), Modifier.weight(1f))
                    Tile(t.today.average, money(average), Modifier.weight(1f))
                }
                Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    Tile(t.today.delivered, (period?.delivered ?: 0).toString(), Modifier.weight(1f))
                    Tile(
                        t.today.cancelled, (period?.cancelled ?: 0).toString(), Modifier.weight(1f),
                        // ⚠️ Amber, not red: a cancelled order is worth a
                        // glance, not an alarm. There is a plain explanation for
                        // most of them.
                        tone = if ((period?.cancelled ?: 0) > 0) c.warn else null,
                    )
                }

                OnShiftCard(staff)

                // ⚠️ **Underneath the numbers.** This screen is opened twenty
                // times a day for the figures and read once for the briefing;
                // above them it would be in the way nineteen times.
                if (locked) {
                    Text(
                        t.today.briefingLocked,
                        style = MaterialTheme.typography.labelMedium,
                        color = c.muted,
                        modifier = Modifier.padding(top = 4.dp),
                    )
                } else if (cards.isNotEmpty()) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        modifier = Modifier.padding(top = 6.dp),
                    ) {
                        Icon(Icons.Rounded.AutoAwesome, null, tint = c.accent, modifier = Modifier.size(16.dp))
                        Text(t.today.briefing, style = MaterialTheme.typography.titleMedium, color = c.ink)
                    }
                    cards.forEach { BriefingCardView(it) }
                }
            }
        }
    }
}

@Composable
private fun Tile(label: String, value: String, modifier: Modifier = Modifier, tone: Color? = null) {
    val c = KeelTheme.colors
    Column(
        modifier.glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(3.dp),
    ) {
        Text(label, style = MaterialTheme.typography.labelMedium, color = c.muted)
        Text(
            value,
            style = MaterialTheme.typography.titleLarge.merge(MoneyStyle),
            color = tone ?: c.ink,
        )
    }
}

/** ⚠️ **The assistant's card says nothing about what to do.** The panel's
 *  briefing suggests; a phone read at eight in the morning states the fact and
 *  stops, because an owner acting on a suggestion they cannot check is the way
 *  this feature gets switched off. */
@Composable
private fun BriefingCardView(card: BriefingCard) {
    val c = KeelTheme.colors
    var open by remember { mutableStateOf(false) }
    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp))
            .clickable { open = !open }
            .padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(card.title, Modifier.weight(1f), style = MaterialTheme.typography.bodyLarge, color = c.ink)
            Icon(
                if (open) Icons.Rounded.ExpandLess else Icons.Rounded.ExpandMore,
                null, tint = c.muted, modifier = Modifier.size(18.dp),
            )
        }
        if (open) {
            Text(card.body, style = MaterialTheme.typography.bodyMedium, color = c.inkSoft)
        }
    }
}

// Who is at work, in one row that opens.
//
// ⚠️ **A "now" question, which is why it sits on the first screen.** Until this
// existed the only way to answer it at half past ten was to ring somebody.
//
// ⚠️ **"Absent" is not counted here.** `todayStatus` comes from the server —
// the roster, a day off and a shift that ended late are three definitions, and a
// second one written on a phone would disagree with payroll on the day it
// mattered.
@Composable
private fun OnShiftCard(rows: List<StaffRow>) {
    val c = KeelTheme.colors
    var open by remember { mutableStateOf(false) }
    val onShift = rows.filter { it.todayStatus == "open" }
    val absent = rows.filter { it.todayStatus == "absent" }

    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp))
            .clickable { open = !open }
            .padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(
                t.who.now(onShift.size), Modifier.weight(1f),
                style = MaterialTheme.typography.bodyLarge, color = c.ink,
            )
            if (absent.isNotEmpty()) {
                Text(
                    t.who.absent(absent.size),
                    style = MaterialTheme.typography.labelMedium,
                    color = c.warn,
                    modifier = Modifier.background(c.warn.copy(alpha = 0.14f), CircleShape)
                        .padding(horizontal = 9.dp, vertical = 4.dp),
                )
            }
            Icon(
                if (open) Icons.Rounded.ExpandLess else Icons.Rounded.ExpandMore,
                null, tint = c.muted, modifier = Modifier.size(18.dp),
            )
        }
        if (open) {
            if (rows.isEmpty()) {
                Text(t.who.empty, style = MaterialTheme.typography.bodyMedium, color = c.muted)
            }
            // ⚠️ **Read top-down for one purpose — who is here.** Sorting by
            // name would bury the answer among people who are not.
            (onShift + absent + rows.filter { it !in onShift && it !in absent }).forEach { r ->
                Row(
                    Modifier.fillMaxWidth().padding(vertical = 6.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Box(
                        Modifier.size(8.dp).background(
                            when (r.todayStatus) {
                                "open" -> c.ready
                                "absent" -> c.warn
                                else -> c.muted
                            },
                            CircleShape,
                        ),
                    )
                    Column(Modifier.weight(1f)) {
                        Text(r.name, style = MaterialTheme.typography.bodyMedium, color = c.ink)
                        if (r.position.isNotEmpty()) {
                            Text(r.position, style = MaterialTheme.typography.labelSmall, color = c.muted)
                        }
                    }
                    Text(
                        when {
                            r.todayStatus == "absent" -> t.who.notYet
                            r.todayStatus == "off" -> t.who.off
                            // ⚠️ "HH:MM" exactly as the server formatted it —
                            // never sliced from a timestamp, which arrives in
                            // UTC and would print an hour nobody worked.
                            r.todayStatus == "open" && r.todayIn.isNotEmpty() -> t.who.since(r.todayIn)
                            r.todayIn.isNotEmpty() && r.todayOut.isNotEmpty() ->
                                t.who.done(r.todayIn, r.todayOut)
                            else -> ""
                        },
                        style = MaterialTheme.typography.labelMedium, color = c.muted,
                        textAlign = TextAlign.End,
                    )
                }
            }
        }
    }
}
