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
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.ChevronRight
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
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import uz.keel.design.GlassIconButton
import uz.keel.design.KeelTheme
import uz.keel.design.MoneyStyle
import uz.keel.design.ScreenHeader
import uz.keel.design.glass
import uz.keel.design.money
import uz.keel.owner.data.AdminAlerts
import uz.keel.owner.data.ApiError
import uz.keel.owner.data.KeelApi
import uz.keel.owner.data.LossAlert
import uz.keel.owner.data.opensCheck
import uz.keel.owner.i18n.timeAgo
import uz.keel.owner.t

// Two lists, and they are not the same kind of thing.
//
// ⚠️ **A queue is work somebody does; a loss is a fact somebody asks about.**
// The first half is orders nobody has accepted, bookings nobody has answered,
// tickets the till never took — each of those is finished by acting on it. The
// second half already happened: a large discount, a void after the bill was
// printed, a shortfall in the drawer. Acting on those the way you act on the
// first is how an owner ends up interrogating a waiter over an ordinary evening.
//
// ⚠️ **The screen does not draw a conclusion.** Every one of these events has a
// plain explanation more often than not, and a screen that judged would be wrong
// often enough that somebody switches it off — and then it is not there on the
// day it was right.

@Composable
fun AlertsScreen(api: KeelApi, bottomInset: PaddingValues, onOpenCheck: (String) -> Unit) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()
    var alerts by remember { mutableStateOf<AdminAlerts?>(null) }
    var losses by remember { mutableStateOf<List<LossAlert>>(emptyList()) }
    var error by remember { mutableStateOf("") }
    val loadFailed = t.common.loadFailed

    suspend fun load() {
        try {
            alerts = api.alerts()
            losses = runCatching { api.lossAlerts().alerts }.getOrDefault(emptyList())
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else loadFailed
            if (alerts == null) alerts = AdminAlerts()
        }
    }
    LaunchedEffect(Unit) { load() }

    val a = alerts
    // The queue, as sentences. ⚠️ Only what is non-zero: a list of noughts is a
    // list nobody reads, and the empty state below says the useful thing.
    val queue = buildList {
        if (a != null) {
            if (a.orders.pending > 0) add(t.alerts.pendingOrders(a.orders.pending))
            if (a.preorders.upcoming > 0) add(t.alerts.preorders(a.preorders.upcoming))
            if (a.reservations.pending > 0) add(t.alerts.reservations(a.reservations.pending))
            if (a.pos.unaccepted > 0) add(t.alerts.posUnaccepted(a.pos.unaccepted))
            if (a.pos.failed > 0) add(t.alerts.posFailed(a.pos.failed))
            if (a.pos.unmapped > 0) add(t.alerts.posUnmapped(a.pos.unmapped))
            if (a.print.failed > 0) add(t.alerts.printFailed(a.print.failed))
        }
    }

    Column(Modifier.fillMaxSize()) {
        ScreenHeader(
            title = t.alerts.title,
            trailing = { GlassIconButton(Icons.Rounded.Refresh) { scope.launch { load() } } },
        )
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp)
                .padding(bottom = bottomInset.calculateBottomPadding() + 24.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            if (a == null && error.isEmpty()) {
                Box(Modifier.fillMaxWidth().padding(40.dp), Alignment.Center) {
                    CircularProgressIndicator(color = c.accent)
                }
            }
            if (error.isNotEmpty()) {
                Text(error, color = c.danger, style = MaterialTheme.typography.bodyMedium)
            }

            queue.forEach { line ->
                Row(
                    Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp)).padding(14.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    Box(Modifier.size(8.dp).background(c.accent, CircleShape))
                    Text(line, Modifier.weight(1f), style = MaterialTheme.typography.bodyLarge, color = c.ink)
                }
            }

            // ⚠️ **Empty is the good state and is said as one.** A blank screen
            // reads as a screen that failed to load; this reads as a restaurant
            // with nothing waiting.
            if (a != null && queue.isEmpty() && losses.isEmpty()) {
                Column(
                    Modifier.fillMaxWidth().padding(vertical = 48.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Icon(Icons.Rounded.CheckCircle, null, tint = c.ready, modifier = Modifier.size(34.dp))
                    Text(t.alerts.empty, style = MaterialTheme.typography.titleMedium, color = c.ink)
                    Text(
                        t.alerts.emptyHint, style = MaterialTheme.typography.bodyMedium,
                        color = c.muted, textAlign = TextAlign.Center,
                    )
                }
            }

            if (losses.isNotEmpty()) {
                Text(
                    t.alerts.lossTitle,
                    style = MaterialTheme.typography.titleMedium, color = c.ink,
                    modifier = Modifier.padding(top = 8.dp),
                )
                losses.forEach { l -> LossRow(l) { onOpenCheck(l.refId) } }
            } else if (a != null && queue.isNotEmpty()) {
                Text(
                    t.alerts.lossEmpty,
                    style = MaterialTheme.typography.labelMedium, color = c.muted,
                    modifier = Modifier.padding(top = 8.dp),
                )
            }
        }
    }
}

/** One thing that already happened.
 *
 *  ⚠️ **The amount, the person and the reason, and no verdict.** "Who" is here
 *  because an owner's next move is a conversation, not a screen — and the reason
 *  is here because most of the time it is the answer.
 *
 *  ⚠️ **Tappable only when it is about a sale.** The same `refId` names a cash
 *  shift or a dish for other kinds, and opening those as a check is a "not
 *  found" on the row the owner was most worried about. */
@Composable
private fun LossRow(l: LossAlert, onOpen: () -> Unit) {
    val c = KeelTheme.colors
    val opens = l.opensCheck()
    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp))
            .then(if (opens) Modifier.clickable(onClick = onOpen) else Modifier)
            .padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(
                t.alerts.kinds[l.kind] ?: l.kind,
                Modifier.weight(1f),
                style = MaterialTheme.typography.bodyLarge, color = c.ink,
            )
            if (l.amount != 0.0) {
                Text(
                    money(l.amount),
                    style = MaterialTheme.typography.bodyLarge.merge(MoneyStyle),
                    color = c.warn,
                )
            }
        }
        val who = l.by ?: ""
        val when_ = timeAgo(l.at, t.common.timeAgo)
        // ⚠️ The check number and the table: "6-stol" alone names a table that
        // has had nine checks today.
        val where = t.alerts.checkRef(l.number, l.table)
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                listOfNotNull(who.takeIf { it.isNotEmpty() }, where.takeIf { it.isNotEmpty() }, when_)
                    .joinToString(" · "),
                Modifier.weight(1f),
                style = MaterialTheme.typography.labelMedium, color = c.muted,
            )
            if (opens) {
                Icon(Icons.Rounded.ChevronRight, null, tint = c.muted, modifier = Modifier.size(18.dp))
            }
        }
        l.subject?.takeIf { it.isNotBlank() }?.let {
            Text(it, style = MaterialTheme.typography.bodyMedium, color = c.inkSoft)
        }
        l.reason?.takeIf { it.isNotBlank() }?.let {
            Text(it, style = MaterialTheme.typography.bodyMedium, color = c.inkSoft)
        }
    }
}
