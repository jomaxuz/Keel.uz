package uz.keel.owner.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import uz.keel.design.Chip
import uz.keel.design.KeelTheme
import uz.keel.design.MoneyStyle
import uz.keel.design.ScreenHeader
import uz.keel.design.glass
import uz.keel.design.money
import uz.keel.owner.LocalPrefs
import uz.keel.owner.data.AdminStats
import uz.keel.owner.data.ApiError
import uz.keel.owner.data.KeelApi
import uz.keel.owner.data.PayrollResponse
import uz.keel.owner.data.ShoppingList
import uz.keel.owner.t

// Today, the week, the month — and only reading.
//
// ⚠️ **Nothing here changes anything, and that is the design.** A report is what
// somebody looks at to decide; the deciding happens in the panel, sitting down.
// A screen that both showed the takings and let you edit a price would be the
// screen where a price gets edited at a traffic light.

private enum class Span { Today, Week, Month }

/** ⚠️ Local calendar days, sent as dates. Every report in this product is filed
 *  by local midnight, and a moment in time would be re-read in UTC on the way. */
private fun rangeFor(span: Span): Pair<String, String> {
    val today = java.time.LocalDate.now()
    val from = when (span) {
        Span.Today -> today
        Span.Week -> today.minusDays(6)
        Span.Month -> today.withDayOfMonth(1)
    }
    return from.toString() to today.toString()
}

@Composable
fun ReportsScreen(api: KeelApi, bottomInset: PaddingValues) {
    val c = KeelTheme.colors
    val prefs = LocalPrefs.current
    val branchId = prefs.branch.value
    var span by remember { mutableStateOf(Span.Today) }
    var stats by remember { mutableStateOf<AdminStats?>(null) }
    var payroll by remember { mutableStateOf<PayrollResponse?>(null) }
    var shopping by remember { mutableStateOf<ShoppingList?>(null) }
    var error by remember { mutableStateOf("") }
    val loadFailed = t.common.loadFailed

    LaunchedEffect(span, branchId) {
        stats = null
        try {
            val (from, to) = rangeFor(span)
            stats = api.stats(from, to, branchId)
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else loadFailed
        }
        // ⚠️ Both are asked separately and neither may take the report down with
        // it: an install with no stock module has no shopping list at all, and
        // that is a normal restaurant rather than a failure.
        payroll = runCatching { api.payroll(branchId) }.getOrNull()
        shopping = runCatching { api.shoppingList(branchId) }.getOrNull()
    }

    val p = stats?.period

    Column(Modifier.fillMaxSize()) {
        ScreenHeader(title = t.reports.title)
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Chip(t.reports.today, span == Span.Today) { span = Span.Today }
            Chip(t.reports.week, span == Span.Week) { span = Span.Week }
            Chip(t.reports.month, span == Span.Month) { span = Span.Month }
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

            if (p != null) {
                Card {
                    Line(t.reports.revenue, money(p.revenue))
                    Line(t.reports.orders, p.orders.toString())
                    Line(
                        t.reports.average,
                        money(if (p.orders > 0) p.revenue / p.orders else 0.0),
                    )
                }

                val top = stats?.top.orEmpty()
                if (top.isNotEmpty()) {
                    Text(t.reports.topDishes, style = MaterialTheme.typography.titleMedium, color = c.ink)
                    Card {
                        top.take(8).forEach { d ->
                            Line("${d.qty} × ${d.name}", money(d.revenue))
                        }
                    }
                }

                payroll?.let { pay ->
                    Text(t.reports.staff, style = MaterialTheme.typography.titleMedium, color = c.ink)
                    Card {
                        pay.rows.take(10).forEach { r -> Line(r.name, money(r.due)) }
                        // ⚠️ The total is what an owner reads this for, and it is
                        // the server's — not a sum computed here, which would
                        // disagree with payroll on the day it mattered.
                        Line(t.reports.owed, money(pay.due), strong = true)
                    }
                }

                shopping?.let { s ->
                    Text(t.reports.stock, style = MaterialTheme.typography.titleMedium, color = c.ink)
                    Card {
                        val lines = s.groups.flatMap { it.lines }
                        if (lines.isEmpty()) {
                            Text(
                                t.reports.stockEmpty,
                                style = MaterialTheme.typography.bodyMedium, color = c.muted,
                            )
                        }
                        lines.take(12).forEach { l ->
                            Line(l.name, "${money(l.need)} ${l.unit}".trim())
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun Card(content: @Composable () -> Unit) {
    val c = KeelTheme.colors
    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) { content() }
}

@Composable
private fun Line(label: String, value: String, strong: Boolean = false) {
    val c = KeelTheme.colors
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(
            label, Modifier.weight(1f),
            style = MaterialTheme.typography.bodyMedium,
            color = if (strong) c.ink else c.inkSoft,
        )
        Text(
            value,
            style = (if (strong) MaterialTheme.typography.titleMedium
            else MaterialTheme.typography.bodyMedium).merge(MoneyStyle),
            color = c.ink,
        )
    }
}
