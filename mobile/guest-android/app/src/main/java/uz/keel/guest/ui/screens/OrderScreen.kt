package uz.keel.guest.ui.screens

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.layout.height
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Call
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import uz.keel.guest.Brand
import uz.keel.guest.data.GeoPoint
import uz.keel.guest.data.Restaurant
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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.background
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
import uz.keel.design.GhostButton
import uz.keel.design.KeelTheme
import uz.keel.design.Money
import uz.keel.design.glass
import uz.keel.guest.data.ApiError
import uz.keel.guest.data.KeelApi
import uz.keel.guest.data.Order
import uz.keel.guest.t

// Following an order.
//
// ⚠️ **Public and keyed by the number**, exactly as the site's tracking page is:
// a guest who ordered without signing in has to be able to watch it, and an
// order somebody cannot follow is a phone call to a restaurant that is busy
// cooking it.
//
// ⚠️ **Polled while it is live, and not after.** A finished order polled every
// ten seconds is a battery cost for a fact that will never change again.

@Composable
fun OrderScreen(
    api: KeelApi,
    number: String,
    /** For the map: which engine the restaurant chose, and its key. */
    restaurant: Restaurant?,
    bottomInset: PaddingValues,
    onBack: () -> Unit,
) {
    val c = KeelTheme.colors
    var order by remember { mutableStateOf<Order?>(null) }
    var error by remember { mutableStateOf("") }
    val loadFailed = t.order.loadFailed

    LaunchedEffect(number) {
        while (true) {
            try {
                val fresh = api.order(number)
                order = fresh
                error = ""
                if (fresh.status == "delivered" || fresh.status == "cancelled") break
            } catch (e: Throwable) {
                error = if (e is ApiError) e.message else loadFailed
            }
            kotlinx.coroutines.delay(15_000)
        }
    }

    val delivered = order?.status == "delivered"

    Box(Modifier.fillMaxSize()) {
        LazyColumn(
            Modifier.fillMaxSize(),
            contentPadding = PaddingValues(
                start = 16.dp,
                end = 16.dp,
                bottom = bottomInset.calculateBottomPadding(),
            ),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Column(
                    Modifier.statusBarsPadding().padding(top = 12.dp),
                    verticalArrangement = Arrangement.spacedBy(4.dp),
                ) {
                    Text(
                        t.order.number(number),
                        style = MaterialTheme.typography.headlineMedium,
                        color = c.ink,
                    )
                    if (error.isNotEmpty()) {
                        Text(error, style = MaterialTheme.typography.bodyMedium, color = c.danger)
                    }
                }
            }

            order?.let { o ->
                if (delivered) item { DeliveredBadge() }
                item {
                    Column(
                        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
                        verticalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        // ⚠️ **A cancelled order is not a step on the road** — it is
                        // the end of a different one, and drawing it as the fifth
                        // dot would say the food is on its way.
                        if (o.status == "cancelled") {
                            Text(
                                t.order.cancelled,
                                style = MaterialTheme.typography.titleMedium,
                                color = c.danger,
                            )
                            if (o.cancelReason.isNotEmpty()) {
                                Text(
                                    o.cancelReason,
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = c.muted,
                                )
                            }
                        } else {
                            val steps = listOf(
                                "pending" to t.order.pending,
                                "confirmed" to t.order.confirmed,
                                "preparing" to t.order.preparing,
                                "on_the_way" to t.order.onTheWay,
                                "delivered" to t.order.delivered,
                            ).filter {
                                // Pickup never goes on a road; showing the step
                                // would have a guest waiting at home for it.
                                o.type == "delivery" || it.first != "on_the_way"
                            }
                            val at = steps.indexOfFirst { it.first == o.status }
                            steps.forEachIndexed { i, (_, label) ->
                                Row(
                                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                                    verticalAlignment = Alignment.CenterVertically,
                                ) {
                                    Column(
                                        Modifier
                                            .size(10.dp)
                                            .background(
                                                if (i <= at) c.accent else c.line,
                                                CircleShape,
                                            ),
                                    ) {}
                                    Text(
                                        label,
                                        style = MaterialTheme.typography.bodyLarge,
                                        color = if (i <= at) c.ink else c.muted,
                                    )
                                }
                            }
                        }
                    }
                }

                // ⚠️ **Where the courier is, while it is on its way.** The server has
                // always sent this (`TrackOrder`, only at `on_the_way`) and the site
                // has always drawn it; the app read neither field, so the guest saw
                // "Yo'lda" and nothing else — the one stage where a map is the
                // whole point of opening the screen.
                val fix = o.courier?.location
                if (o.status == "on_the_way" && fix != null && (fix.lat != 0.0 || fix.lng != 0.0)) {
                    item { CourierCard(o, restaurant) }
                }

                item {
                    Column(
                        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
                        verticalArrangement = Arrangement.spacedBy(6.dp),
                    ) {
                        o.items.forEach { line ->
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text(
                                    "${line.qty} × ${line.name}",
                                    Modifier.weight(1f),
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = c.ink,
                                )
                                Money(line.price * line.qty, color = c.inkSoft)
                            }
                        }
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                t.checkout.total,
                                Modifier.weight(1f),
                                style = MaterialTheme.typography.titleMedium,
                                color = c.ink,
                            )
                            Money(o.total, color = c.ink)
                        }
                    }
                }
            }

            item { GhostButton(t.common.back, Modifier.fillMaxWidth()) { onBack() } }
        }
        // Over the list, and never in the way of it: the overlay draws only.
        if (delivered) ConfettiOverlay()
    }
}

@Composable
private fun CourierCard(o: Order, restaurant: Restaurant?) {
    val c = KeelTheme.colors
    val ctx = LocalContext.current
    val courier = o.courier ?: return
    val fix = courier.location ?: return
    val provider = restaurant?.provider ?: "2gis"
    // Same fallback as the checkout's picker: the build's Google key only
    // stands in for a restaurant on Google that has not set its own.
    val key = (restaurant?.mapKey ?: "").ifBlank {
        if (provider == "google" || restaurant == null) Brand.mapsKey else ""
    }
    val home = o.address.takeIf { it.lat != 0.0 || it.lng != 0.0 }
    val minutes = runCatching {
        ((System.currentTimeMillis() - java.time.Instant.parse(fix.at).toEpochMilli()) / 60_000).toInt()
    }.getOrNull()

    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Text(t.order.courierComing(courier.name), style = MaterialTheme.typography.titleMedium, color = c.ink)
        // ⚠️ No key, no map — and not a grey rectangle either: the sentence
        // above and the call button still answer "where is my food?".
        if (key.isNotBlank()) {
            TrackMap(
                provider = provider,
                mapKey = key,
                courier = GeoPoint(fix.lat, fix.lng),
                home = home,
                accent = c.accent,
                modifier = Modifier.fillMaxWidth().height(240.dp).clip(RoundedCornerShape(16.dp)),
            )
        }
        if (minutes != null) {
            Text(t.order.courierSeen(minutes), style = MaterialTheme.typography.labelMedium, color = c.muted)
        }
        if (courier.phone.isNotBlank()) {
            GhostButton(t.order.callCourier, Modifier.fillMaxWidth(), icon = Icons.Rounded.Call) {
                runCatching {
                    ctx.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:${courier.phone}")))
                }
            }
        }
    }
}
