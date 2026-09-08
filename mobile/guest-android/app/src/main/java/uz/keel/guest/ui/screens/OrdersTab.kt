package uz.keel.guest.ui.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import uz.keel.design.KeelTheme
import uz.keel.design.glass
import uz.keel.guest.PlacedOrders
import uz.keel.guest.t

/** What this phone has ordered.
 *
 *  ⚠️ **It works without an account**, which is the ordinary first order: a tab
 *  that was empty until somebody signed in would answer "where is my food" with
 *  nothing, at exactly the moment they are asking. The account's own history
 *  joins this when the sign-in arrives; it does not replace it. */
@Composable
fun OrdersTab(placed: PlacedOrders, bottomInset: PaddingValues, onOpen: (String) -> Unit) {
    val c = KeelTheme.colors

    if (placed.numbers.isEmpty()) {
        Column(
            Modifier.fillMaxSize().padding(32.dp),
            verticalArrangement = Arrangement.spacedBy(6.dp, Alignment.CenterVertically),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(t.order.none, style = MaterialTheme.typography.titleLarge, color = c.ink)
            Text(t.order.noneHint, style = MaterialTheme.typography.bodyMedium, color = c.muted)
        }
        return
    }

    LazyColumn(
        Modifier.fillMaxSize(),
        contentPadding = PaddingValues(
            start = 16.dp,
            end = 16.dp,
            bottom = bottomInset.calculateBottomPadding(),
        ),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        item {
            Text(
                t.order.title,
                Modifier.statusBarsPadding().padding(top = 12.dp),
                style = MaterialTheme.typography.headlineMedium,
                color = c.ink,
            )
        }
        items(placed.numbers, key = { it }) { number ->
            Text(
                t.order.number(number),
                Modifier
                    .fillMaxWidth()
                    .glass(c, RoundedCornerShape(16.dp))
                    .clickable { onOpen(number) }
                    .padding(horizontal = 14.dp, vertical = 14.dp),
                style = MaterialTheme.typography.bodyLarge,
                color = c.ink,
            )
        }
    }
}
