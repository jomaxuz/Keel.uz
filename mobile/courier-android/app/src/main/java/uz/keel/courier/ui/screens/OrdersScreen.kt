package uz.keel.courier.ui.screens

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.background
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
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.CreditCard
import androidx.compose.material.icons.rounded.EditNote
import androidx.compose.material.icons.rounded.Inbox
import androidx.compose.material.icons.rounded.Money
import androidx.compose.material.icons.rounded.Navigation
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.Phone
import androidx.compose.material.icons.rounded.Place
import androidx.compose.material.icons.rounded.PowerSettingsNew
import androidx.compose.material.icons.rounded.WarningAmber
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import uz.keel.design.GhostButton
import uz.keel.design.KeelTheme
import uz.keel.design.LocalNotice
import uz.keel.design.Money
import uz.keel.design.Note
import uz.keel.design.NoticeKind
import uz.keel.design.PrimaryButton
import uz.keel.design.glass
import uz.keel.design.glassSheet
import uz.keel.design.money
import uz.keel.courier.data.ApiError
import uz.keel.courier.data.KeelApi
import uz.keel.courier.data.Order
import uz.keel.courier.location.Tracking
import uz.keel.courier.location.arrivalGate
import uz.keel.courier.t

// The courier's working screen.
//
// ⚠️ **One list, and the next step is a button on the card.** Everything a
// courier does in an evening is "picked it up" and "handed it over"; a screen
// with a detail view behind a tap would put a second press between a bike and
// the road, twice per order.

/** How often the list is re-asked.
 *
 *  ⚠️ Not a socket: the phone is on mobile data, moving between cells, and a
 *  poll that fails is a poll that simply happens again in twenty seconds. A
 *  dropped socket needs reconnect logic nobody would be watching. */
private const val POLL_MS = 20_000L

@Composable
fun OrdersScreen(
    api: KeelApi,
    name: String,
    status: String,
    bottomInset: PaddingValues,
    onStatus: (String) -> Unit,
    onRefreshCourier: () -> Unit,
) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()
    val notice = LocalNotice.current
    val tracking by Tracking.state.collectAsState()

    var orders by remember { mutableStateOf<List<Order>?>(null) }
    var busyId by remember { mutableStateOf<String?>(null) }
    var routeFor by remember { mutableStateOf<Order?>(null) }
    // The branch's arrival radius, in metres. ⚠️ Asked from the server rather
    // than hard-coded: a city-centre restaurant may want 100 m and a village one
    // 500, and the number here has to be the one the server will judge by.
    var radius by remember { mutableIntStateOf(0) }
    var tick by remember { mutableIntStateOf(0) }

    LaunchedEffect(Unit) {
        radius = runCatching { api.arrivalRadiusM() }.getOrDefault(0)
    }

    LaunchedEffect(tick) {
        // ⚠️ Whatever is on screen is kept when a poll fails. A courier under a
        // bridge should not have their list emptied — that reads as "the order
        // was taken off me".
        runCatching { api.orders() }.onSuccess { orders = it }
        if (orders == null) orders = emptyList()
        onRefreshCourier()
        delay(POLL_MS)
        tick += 1
    }

    // ⚠️ Read in composition and captured, not read inside the coroutine: the
    // dictionary is a composable read, and the sentence has to be in the
    // language that is on screen now.
    val failedTitle = t.orders.failed

    fun advance(o: Order) {
        val next = if (o.status == "on_the_way") "delivered" else "on_the_way"
        busyId = o.id
        scope.launch {
            try {
                api.advance(o.id, next)
                tick += 1
            } catch (e: Throwable) {
                // ⚠️ **The server's own sentence, in a sheet.** This is the one
                // refusal that arrives while a customer is watching, and it is
                // usually the arrival check against a position the server has
                // not received yet — a message the courier can act on ("keep
                // the app open a moment") rather than a red line under a button.
                notice.value = Note(
                    NoticeKind.Warn,
                    failedTitle,
                    (e as? ApiError)?.message,
                )
            } finally {
                busyId = null
            }
        }
    }

    val list = orders.orEmpty()
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
            Column(Modifier.statusBarsPadding().padding(top = 8.dp)) {
                Text(name, style = MaterialTheme.typography.headlineMedium, color = c.ink)
                Text(
                    t.orders.title(list.size),
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.muted,
                )
            }
        }
        item { ShiftCard(status = status, tracking = tracking, onStatus = onStatus) }

        if (orders == null) {
            item {
                Box(Modifier.fillMaxWidth().padding(40.dp), Alignment.Center) {
                    CircularProgressIndicator(color = c.accent)
                }
            }
        } else if (list.isEmpty()) {
            item {
                Column(
                    Modifier.fillMaxWidth().padding(vertical = 40.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Icon(
                        if (status == "off") Icons.Rounded.PowerSettingsNew else Icons.Rounded.Inbox,
                        null,
                        tint = c.muted,
                        modifier = Modifier.size(28.dp),
                    )
                    Text(
                        if (status == "off") t.orders.offEmpty else t.orders.empty,
                        style = MaterialTheme.typography.titleLarge,
                        color = c.ink,
                        textAlign = TextAlign.Center,
                    )
                    Text(
                        if (status == "off") t.orders.offEmptyHint else t.orders.emptyHint,
                        style = MaterialTheme.typography.bodyMedium,
                        color = c.muted,
                        textAlign = TextAlign.Center,
                    )
                }
            }
        } else {
            items(list, key = { it.id }) { order ->
                OrderCard(
                    order = order,
                    radius = radius,
                    fix = tracking.fix,
                    busy = busyId == order.id,
                    onAdvance = { advance(order) },
                    onRoute = { routeFor = order },
                )
            }
        }
    }

    routeFor?.let { RouteSheet(it) { routeFor = null } }
}

@Composable
private fun OrderCard(
    order: Order,
    radius: Int,
    fix: uz.keel.courier.location.Fix?,
    busy: Boolean,
    onAdvance: () -> Unit,
    onRoute: () -> Unit,
) {
    val c = KeelTheme.colors
    val ctx = LocalContext.current

    val onTheWay = order.status == "on_the_way"
    val gate = if (onTheWay) arrivalGate(order, fix, radius, t)
    else uz.keel.courier.location.Gate(false, null, null)
    val blocked = gate.reason != null
    val atDoor = gate.applies && !blocked

    Column(
        Modifier
            .fillMaxWidth()
            .glass(c, RoundedCornerShape(22.dp)),
    ) {
        // ⚠️ A stripe rather than a badge: the card is read at a glance from a
        // pocket, and the colour has to survive being half out of it.
        Box(
            Modifier
                .fillMaxWidth()
                .height(4.dp)
                .background(if (onTheWay) (if (atDoor) c.ready else c.warn) else c.accent),
        )
        Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    "#${order.number}",
                    style = MaterialTheme.typography.titleLarge,
                    color = c.ink,
                    modifier = Modifier.weight(1f),
                )
                gate.meters?.let { m ->
                    Row(
                        Modifier
                            .background(
                                (if (atDoor) c.ready else c.warn).copy(alpha = 0.15f),
                                RoundedCornerShape(999.dp),
                            )
                            .padding(horizontal = 9.dp, vertical = 4.dp),
                        horizontalArrangement = Arrangement.spacedBy(4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(
                            if (atDoor) Icons.Rounded.CheckCircle else Icons.Rounded.Navigation,
                            null,
                            tint = if (atDoor) c.ready else c.warn,
                            modifier = Modifier.size(13.dp),
                        )
                        Text(
                            if (atDoor) t.orders.atDoor else t.orders.away(m),
                            style = MaterialTheme.typography.labelMedium,
                            color = if (atDoor) c.ready else c.warn,
                        )
                    }
                }
                Money(order.total, style = MaterialTheme.typography.titleMedium)
            }

            if (order.address.text.isNotEmpty()) {
                CardLine(Icons.Rounded.Place) {
                    Text(
                        order.address.text +
                            if (order.address.comment.isNotEmpty()) " · ${order.address.comment}" else "",
                        style = MaterialTheme.typography.bodyMedium,
                        color = c.inkSoft,
                    )
                }
            }

            CardLine(Icons.Rounded.Person) {
                Text(
                    order.customer.name,
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.inkSoft,
                )
            }

            // What the courier owes the restaurant at the end of the evening,
            // said on every card. ⚠️ Cash and card are otherwise the same
            // screen, and handing back the wrong amount is discovered a day
            // later.
            val cash = order.paymentMethod == "cash"
            CardLine(if (cash) Icons.Rounded.Money else Icons.Rounded.CreditCard, if (cash) c.accent else null) {
                Text(
                    if (cash) t.orders.cash(money(order.total)) else t.orders.paid,
                    style = MaterialTheme.typography.bodyMedium,
                    color = if (cash) c.ink else c.inkSoft,
                )
            }

            val notes = order.items.filter { it.comment.isNotEmpty() }
            if (notes.isNotEmpty()) {
                Banner(Icons.Rounded.EditNote, c.warn) {
                    Text(
                        notes.joinToString(" · ") { "${it.name}: ${it.comment}" },
                        style = MaterialTheme.typography.labelMedium,
                        color = c.warn,
                    )
                }
            }

            // ⚠️ **The reason lives above the button it is holding shut.** A
            // disabled control with its explanation elsewhere on the screen is a
            // control somebody presses repeatedly, in front of a customer,
            // before going looking for the reason.
            gate.reason?.let { reason ->
                Banner(Icons.Rounded.WarningAmber, c.warn) {
                    Text(
                        reason,
                        style = MaterialTheme.typography.labelMedium,
                        color = c.warn,
                    )
                }
            }

            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                GhostButton(t.orders.call, Modifier.weight(1f), Icons.Rounded.Phone) {
                    val digits = order.customer.phone.filter { it.isDigit() }
                    runCatching {
                        ctx.startActivity(
                            Intent(Intent.ACTION_DIAL, Uri.parse("tel:+$digits"))
                                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
                        )
                    }
                }
                if (order.address.lat != null) {
                    GhostButton(t.orders.route, Modifier.weight(1f), Icons.Rounded.Navigation) {
                        onRoute()
                    }
                }
            }

            PrimaryButton(
                label = if (onTheWay) t.orders.deliver else t.orders.pickUp,
                icon = if (onTheWay) Icons.Rounded.CheckCircle else Icons.Rounded.Inbox,
                enabled = !blocked,
                busy = busy,
                onClick = onAdvance,
            )
        }
    }
}

@Composable
private fun CardLine(
    icon: ImageVector,
    tint: Color? = null,
    content: @Composable () -> Unit,
) {
    val c = KeelTheme.colors
    Row(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Icon(icon, null, tint = tint ?: c.muted, modifier = Modifier.size(15.dp))
        Box(Modifier.weight(1f)) { content() }
    }
}

@Composable
private fun Banner(icon: ImageVector, tint: Color, content: @Composable () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .background(tint.copy(alpha = 0.14f), RoundedCornerShape(14.dp))
            .padding(horizontal = 10.dp, vertical = 9.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Icon(icon, null, tint = tint, modifier = Modifier.size(14.dp))
        Box(Modifier.weight(1f)) { content() }
    }
}

/** Which navigator to open.
 *
 *  ⚠️ **A picker rather than one hard-coded app**, and the same three links the
 *  site gives a guest — pointed the other way. Couriers already have a navigator
 *  they trust, and the one that gets them there fastest is the one they know.
 *
 *  ⚠️ **Note the coordinate order**: Yandex and Google take lat,lng and 2GIS
 *  takes lng,lat. Swapping them puts a courier in the Aral Sea, and it looks
 *  like bad data rather than like a bug here.
 *
 *  The origin is left out of every link on purpose: each app uses the phone's
 *  own position, which it knows better than we do. */
@Composable
private fun RouteSheet(order: Order, onClose: () -> Unit) {
    val c = KeelTheme.colors
    val ctx = LocalContext.current
    val lat = order.address.lat ?: return
    val lng = order.address.lng ?: return
    val ll = "$lat,$lng"

    val links = listOf(
        "Yandex" to "https://yandex.uz/maps/?rtext=~$ll&rtt=auto&z=16",
        "Google" to "https://www.google.com/maps/dir/?api=1&destination=$ll&travelmode=driving",
        "2GIS" to "https://2gis.uz/directions/points/%7C$lng%2C$lat",
    )

    Dialog(onDismissRequest = onClose) {
        Column(
            Modifier
                .widthIn(max = 360.dp)
                .glassSheet(c, RoundedCornerShape(26.dp))
                .padding(18.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(t.orders.routeTitle, style = MaterialTheme.typography.titleLarge, color = c.ink)
            if (order.address.text.isNotEmpty()) {
                Text(
                    order.address.text,
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.muted,
                )
            }
            links.forEach { (label, url) ->
                GhostButton(label, Modifier.fillMaxWidth(), Icons.Rounded.Navigation) {
                    runCatching {
                        ctx.startActivity(
                            Intent(Intent.ACTION_VIEW, Uri.parse(url))
                                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
                        )
                    }
                    onClose()
                }
            }
            GhostButton(t.common.close, Modifier.fillMaxWidth()) { onClose() }
        }
    }
}
