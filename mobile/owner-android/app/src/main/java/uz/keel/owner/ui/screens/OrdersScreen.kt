package uz.keel.owner.ui.screens

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
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Call
import androidx.compose.material.icons.rounded.CheckCircle
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import kotlinx.coroutines.launch
import uz.keel.design.GhostButton
import uz.keel.design.GlassField
import uz.keel.design.GlassIconButton
import uz.keel.design.KeelTheme
import uz.keel.design.MoneyStyle
import uz.keel.design.PrimaryButton
import uz.keel.design.ScreenHeader
import uz.keel.design.glass
import uz.keel.design.glassSheet
import uz.keel.design.money
import uz.keel.owner.LocalPrefs
import uz.keel.owner.data.ApiError
import uz.keel.owner.data.KeelApi
import uz.keel.owner.data.Order
import uz.keel.owner.i18n.timeAgo
import uz.keel.owner.t

// The live board, and the three things an owner does to it.
//
// ⚠️ **Accept, refuse, ring. Nothing else.** A courier, a discount, a corrected
// address — those are panel work, done sitting down. A screen that could change
// a price at a traffic light eventually changes one.

@Composable
fun OrdersScreen(api: KeelApi, bottomInset: PaddingValues) {
    val c = KeelTheme.colors
    val ctx = LocalContext.current
    val prefs = LocalPrefs.current
    val scope = rememberCoroutineScope()
    var orders by remember { mutableStateOf<List<Order>?>(null) }
    var error by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf("") }
    var cancelling by remember { mutableStateOf<Order?>(null) }
    val loadFailed = t.common.loadFailed
    val failed = t.orders.failed
    val branchId = prefs.branch.value

    suspend fun load() {
        try {
            // ⚠️ The board leaves till checks out: those are read by somebody
            // asking about a shift, not about a delivery.
            orders = api.orders(branchId = branchId)
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else loadFailed
            if (orders == null) orders = emptyList()
        }
    }
    LaunchedEffect(branchId) { load() }

    fun setStatus(o: Order, status: String, reason: String = "") {
        busy = o.id
        scope.launch {
            try {
                api.setOrderStatus(o.id, status, reason)
                load()
                error = ""
            } catch (e: Throwable) {
                error = if (e is ApiError) e.message else failed
            } finally { busy = "" }
        }
    }

    fun ring(phone: String) {
        if (phone.isBlank()) return
        // ⚠️ `ACTION_DIAL`, never `ACTION_CALL`: dialling puts the number in
        // front of the person and lets them press it. Placing the call needs a
        // permission, and a permission asked so a button can skip one tap is a
        // permission somebody refuses — and then the button does nothing at all.
        runCatching {
            ctx.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:$phone")))
        }
    }

    Column(Modifier.fillMaxSize()) {
        ScreenHeader(
            title = t.orders.title,
            subtitle = orders?.size?.takeIf { it > 0 }?.toString(),
            trailing = { GlassIconButton(Icons.Rounded.Refresh) { scope.launch { load() } } },
        )
        if (error.isNotEmpty()) {
            Text(
                error, color = c.danger, textAlign = TextAlign.Center,
                style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
            )
        }

        val list = orders
        if (list == null) {
            Box(Modifier.fillMaxSize(), Alignment.Center) { CircularProgressIndicator(color = c.accent) }
            return@Column
        }

        LazyColumn(
            Modifier.fillMaxSize(),
            contentPadding = PaddingValues(
                16.dp, 4.dp, 16.dp, bottomInset.calculateBottomPadding() + 24.dp,
            ),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            items(list, key = { it.id }) { o ->
                OrderCard(
                    order = o,
                    busy = busy == o.id,
                    onConfirm = { setStatus(o, "confirmed") },
                    onCancel = { cancelling = o },
                    onCall = { ring(o.phone) },
                )
            }
            if (list.isEmpty()) {
                item {
                    Column(
                        Modifier.fillMaxWidth().padding(vertical = 56.dp),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        Icon(Icons.Rounded.CheckCircle, null, tint = c.ready, modifier = Modifier.size(32.dp))
                        Text(t.orders.empty, style = MaterialTheme.typography.titleMedium, color = c.ink)
                        Text(
                            t.orders.emptyHint, style = MaterialTheme.typography.bodyMedium,
                            color = c.muted, textAlign = TextAlign.Center,
                        )
                    }
                }
            }
        }
    }

    cancelling?.let { o ->
        CancelDialog(
            onClose = { cancelling = null },
            onCancel = { reason -> cancelling = null; setStatus(o, "cancelled", reason) },
        )
    }
}

@Composable
private fun OrderCard(
    order: Order,
    busy: Boolean,
    onConfirm: () -> Unit,
    onCancel: () -> Unit,
    onCall: () -> Unit,
) {
    val c = KeelTheme.colors
    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("#${order.number}", style = MaterialTheme.typography.titleMedium, color = c.ink)
            Text(
                timeAgo(order.createdAt, t.common.timeAgo),
                Modifier.weight(1f),
                style = MaterialTheme.typography.labelMedium, color = c.muted,
            )
            Text(
                money(order.total),
                style = MaterialTheme.typography.titleMedium.merge(MoneyStyle), color = c.ink,
            )
        }
        if (order.customerName.isNotBlank() || order.phone.isNotBlank()) {
            Text(
                listOf(order.customerName, order.phone).filter { it.isNotBlank() }.joinToString(" · "),
                style = MaterialTheme.typography.bodyMedium, color = c.inkSoft,
            )
        }
        if (order.address.isNotBlank()) {
            Text(order.address, style = MaterialTheme.typography.bodyMedium, color = c.muted)
        }
        // ⚠️ The dishes are here because "accept or not" is sometimes answered
        // by what was ordered — a delivery of one drink to the next city is a
        // decision, not a formality.
        order.items.take(6).forEach { it ->
            Text(
                "${it.qty} × ${it.name}",
                style = MaterialTheme.typography.bodyMedium, color = c.inkSoft,
            )
        }
        if (order.items.size > 6) {
            Text("+${order.items.size - 6}", style = MaterialTheme.typography.labelMedium, color = c.muted)
        }
        order.comment.takeIf { it.isNotBlank() }?.let {
            Text(it, style = MaterialTheme.typography.bodyMedium, color = c.warn)
        }

        Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
            if (order.status == "pending") {
                Box(Modifier.weight(1f)) {
                    PrimaryButton(t.orders.confirm, enabled = !busy, busy = busy, onClick = onConfirm)
                }
            }
            if (order.phone.isNotBlank()) {
                GhostButton(t.orders.call, icon = Icons.Rounded.Call, onClick = onCall)
            }
            GhostButton(t.orders.cancel, tint = c.danger, enabled = !busy, onClick = onCancel)
        }
    }
}

/** ⚠️ **The reason is required and the server insists too.** The guest reads it
 *  on the tracking page — "cancelled" with no sentence is the message that
 *  produces a phone call, and the person who has to answer it is the one who
 *  pressed this button. */
@Composable
private fun CancelDialog(onClose: () -> Unit, onCancel: (String) -> Unit) {
    val c = KeelTheme.colors
    var reason by remember { mutableStateOf("") }
    Dialog(onDismissRequest = onClose) {
        Column(
            Modifier.widthIn(max = 400.dp).imePadding()
                .glassSheet(c, RoundedCornerShape(26.dp)).padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(t.orders.cancel, style = MaterialTheme.typography.titleMedium, color = c.ink)
            GlassField(reason, { reason = it }, t.orders.cancelReason)
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically) {
                GhostButton(t.orders.cancelBack, Modifier.weight(1f), onClick = onClose)
                Box(Modifier.weight(1f)) {
                    PrimaryButton(t.orders.cancelDo, enabled = reason.isNotBlank()) {
                        onCancel(reason.trim())
                    }
                }
            }
        }
    }
}
