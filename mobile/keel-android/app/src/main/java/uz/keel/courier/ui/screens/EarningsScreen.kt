package uz.keel.courier.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.Money
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import uz.keel.design.BigNumberStyle
import uz.keel.design.KeelTheme
import uz.keel.design.Money
import uz.keel.design.glass
import uz.keel.design.money
import uz.keel.courier.data.CourierOrderRow
import uz.keel.courier.data.CourierPeriod
import uz.keel.courier.data.CourierStats
import uz.keel.courier.data.KeelApi
import uz.keel.courier.t

// What the evening was worth, and what is still in a pocket.
//
// ⚠️ **Cash in hand is the number this screen exists for.** The rest is
// encouragement; that one is a debt. It is everything collected less everything
// handed back — not the day's takings, which the web app showed first and which
// is wrong twice over: it stays on screen after the money has been handed in,
// and it misses cash still owed from yesterday.

@Composable
fun EarningsScreen(api: KeelApi, bottomInset: PaddingValues) {
    val c = KeelTheme.colors
    var stats by remember { mutableStateOf<CourierStats?>(null) }
    var history by remember { mutableStateOf<List<CourierOrderRow>?>(null) }

    LaunchedEffect(Unit) {
        stats = runCatching { api.stats() }.getOrNull()
        history = runCatching { api.history() }.getOrDefault(emptyList())
    }

    LazyColumn(
        Modifier.fillMaxSize(),
        contentPadding = PaddingValues(
            start = 16.dp,
            end = 16.dp,
            top = 8.dp,
            bottom = bottomInset.calculateBottomPadding(),
        ),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            Text(
                t.earnings.title,
                Modifier.statusBarsPadding().padding(top = 8.dp),
                style = MaterialTheme.typography.headlineMedium,
                color = c.ink,
            )
        }

        item {
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Tile(t.earnings.today, stats?.today, Modifier.weight(1f))
                Tile(t.earnings.week, stats?.week, Modifier.weight(1f))
            }
        }
        item {
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Tile(t.earnings.month, stats?.month, Modifier.weight(1f))
                Tile(t.earnings.all, stats?.all, Modifier.weight(1f))
            }
        }

        stats?.let { s ->
            item {
                when {
                    s.cashInHand > 0 -> CashBanner(
                        Icons.Rounded.Money,
                        c.warn,
                        t.earnings.cashInHand(money(s.cashInHand)),
                    )
                    s.cashSettled > 0 -> CashBanner(
                        Icons.Rounded.CheckCircle,
                        c.ready,
                        t.earnings.cashClear,
                    )
                    else -> Unit
                }
            }
        }

        item {
            Text(t.earnings.history, style = MaterialTheme.typography.titleLarge, color = c.ink)
        }

        val rows = history
        if (rows == null) {
            item {
                Box(Modifier.fillMaxWidth().padding(30.dp), Alignment.Center) {
                    CircularProgressIndicator(color = c.accent)
                }
            }
        } else if (rows.isEmpty()) {
            item {
                Text(
                    t.earnings.historyEmpty,
                    Modifier.fillMaxWidth().padding(vertical = 24.dp),
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.muted,
                )
            }
        } else {
            items(rows, key = { it.id }) { row -> HistoryRow(row) }
        }
    }
}

@Composable
private fun Tile(label: String, period: CourierPeriod?, modifier: Modifier = Modifier) {
    val c = KeelTheme.colors
    Column(
        modifier.glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(2.dp),
    ) {
        Text(label, style = MaterialTheme.typography.labelMedium, color = c.muted)
        // ⚠️ The one number a screen is opened for gets its own size, and
        // tabular figures so it does not jitter while it counts up.
        Money(period?.earnings ?: 0.0, color = c.accent, style = BigNumberStyle)
        Text(
            t.earnings.deliveries(period?.orders ?: 0),
            style = MaterialTheme.typography.labelMedium,
            color = c.muted,
        )
    }
}

@Composable
private fun CashBanner(
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    tint: Color,
    text: String,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .background(tint.copy(alpha = 0.14f), RoundedCornerShape(16.dp))
            .padding(14.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, null, tint = tint, modifier = Modifier.size(18.dp))
        Text(text, style = MaterialTheme.typography.bodyLarge, color = tint)
    }
}

@Composable
private fun HistoryRow(row: CourierOrderRow) {
    val c = KeelTheme.colors
    val ctx = LocalContext.current
    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                "#${row.number}",
                Modifier.weight(1f),
                style = MaterialTheme.typography.titleMedium,
                color = c.ink,
            )
            Money(row.earned, color = c.ready, style = MaterialTheme.typography.titleMedium)
        }
        Text(
            listOfNotNull(
                when_(ctx, row.deliveredAt).takeIf { it.isNotEmpty() },
                row.address.text.takeIf { it.isNotEmpty() },
            ).joinToString(" · "),
            style = MaterialTheme.typography.labelMedium,
            color = c.muted,
        )
        Text(
            "${money(row.total)} · " +
                if (row.paymentMethod == "cash") t.earnings.cash else t.orders.paid,
            style = MaterialTheme.typography.labelMedium,
            color = c.muted,
        )
    }
}

/** When it was delivered, in the phone's own format.
 *
 *  ⚠️ **The instant is parsed, never sliced.** The server sends UTC, and cutting
 *  the first ten characters off it — the shortcut this codebase has been bitten
 *  by twice — shows the previous day to every courier working past midnight. */
private fun when_(ctx: android.content.Context, iso: String): String {
    val ms = runCatching { java.time.Instant.parse(iso).toEpochMilli() }.getOrNull()
        ?: return ""
    val date = java.util.Date(ms)
    val d = android.text.format.DateFormat.getDateFormat(ctx).format(date)
    val time = android.text.format.DateFormat.getTimeFormat(ctx).format(date)
    return "$d $time"
}
