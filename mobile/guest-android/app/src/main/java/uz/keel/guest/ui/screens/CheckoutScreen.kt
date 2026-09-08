package uz.keel.guest.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
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
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import uz.keel.design.Chip
import uz.keel.design.GhostButton
import uz.keel.design.GlassField
import uz.keel.design.KeelTheme
import uz.keel.design.Money
import uz.keel.design.PrimaryButton
import uz.keel.design.glass
import uz.keel.guest.Brand
import uz.keel.guest.Cart
import uz.keel.guest.data.ApiError
import uz.keel.guest.data.GeoPoint
import uz.keel.guest.data.KeelApi
import uz.keel.guest.data.Order
import uz.keel.guest.data.OrderQuote
import uz.keel.guest.data.Restaurant
import uz.keel.guest.t

// Turning a basket into an order.
//
// ⚠️ **Every figure on this screen comes from the server** (`/orders/quote`),
// and none of them is added up here. A discount rule, a delivery zone and a
// points ceiling each have logic the app does not carry, and a total the guest
// read that differs from the one they are charged is the worst thing this
// application can do — worse than being slow, worse than being ugly.
//
// ⚠️ **The quote is asked again whenever anything it depends on changes**: the
// basket, the order type, the address, the promo code. A stale quote is the same
// failure as a computed one, arriving later.

@Composable
fun CheckoutScreen(
    api: KeelApi,
    cart: Cart,
    restaurant: Restaurant?,
    bottomInset: PaddingValues,
    onBack: () -> Unit,
    onPlaced: (Order) -> Unit,
) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()

    var type by remember { mutableStateOf("delivery") }
    var name by remember { mutableStateOf("") }
    var phone by remember { mutableStateOf("") }
    var point by remember { mutableStateOf<GeoPoint?>(null) }
    var comment by remember { mutableStateOf("") }
    var promo by remember { mutableStateOf("") }
    var payment by remember { mutableStateOf("cash") }
    var methods by remember { mutableStateOf(listOf("cash")) }
    var quote by remember { mutableStateOf<OrderQuote?>(null) }
    var picking by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }

    // ⚠️ Read in the composition and used in the coroutine. `t` is composable —
    // reaching for it inside a callback compiles nowhere and, worse, would tie a
    // background job to a composition that may already be gone.
    val failedWord = t.checkout.failed
    val nameWord = t.checkout.nameNeeded
    val phoneWord = t.checkout.phoneNeeded
    val addressWord = t.checkout.addressNeeded

    LaunchedEffect(Unit) {
        runCatching { methods = api.paymentMethods().methods.ifEmpty { listOf("cash") } }
    }

    // ⚠️ Re-asked on every input the price depends on — see the file's note.
    LaunchedEffect(cart.lines.toList(), type, point, promo) {
        runCatching {
            quote = api.quote(cart.lines.toList(), type, point.takeIf { type == "delivery" }, promo, 0.0)
            error = ""
        }.onFailure { e -> error = if (e is ApiError) e.message else failedWord }
    }

    if (picking) {
        MapPickerScreen(
            // ⚠️ Opens on the restaurant, not on the guest: see MapPickerScreen.
            start = point ?: restaurant?.address ?: GeoPoint(),
            hasKey = Brand.mapsKey.isNotEmpty(),
            onClose = { picking = false },
            onPicked = {
                point = it
                picking = false
            },
        )
        return
    }

    fun send() {
        when {
            name.isBlank() -> { error = nameWord; return }
            phone.isBlank() -> { error = phoneWord; return }
            type == "delivery" && point == null -> { error = addressWord; return }
        }
        busy = true
        error = ""
        scope.launch {
            try {
                val order = api.createOrder(
                    lines = cart.lines.toList(),
                    type = type,
                    name = name.trim(),
                    phone = phone.trim(),
                    point = point.takeIf { type == "delivery" },
                    comment = comment.trim(),
                    paymentMethod = payment,
                    promoCode = promo.trim(),
                    usePoints = 0.0,
                )
                cart.clear()
                onPlaced(order)
            } catch (e: Throwable) {
                error = if (e is ApiError) e.message else failedWord
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
                bottom = bottomInset.calculateBottomPadding() + 150.dp,
            ),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                Column(
                    Modifier.statusBarsPadding().padding(top = 12.dp),
                    verticalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    Text(
                        t.checkout.title,
                        style = MaterialTheme.typography.headlineMedium,
                        color = c.ink,
                    )
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Chip(t.checkout.delivery, type == "delivery") { type = "delivery" }
                        Chip(t.checkout.pickup, type == "pickup") { type = "pickup" }
                    }
                    GlassField(name, { name = it }, t.checkout.name)
                    GlassField(phone, { phone = it }, t.checkout.phone)

                    if (type == "delivery") {
                        // ⚠️ **The chosen address is shown, not just "chosen".**
                        // A guest who picked the wrong building has one chance to
                        // notice, and it is here.
                        Text(
                            point?.text?.ifBlank { null } ?: t.checkout.address,
                            style = MaterialTheme.typography.bodyMedium,
                            color = if (point == null) c.muted else c.ink,
                        )
                        GhostButton(t.checkout.addressPick, Modifier.fillMaxWidth()) {
                            picking = true
                        }
                        GlassField(comment, { comment = it }, t.checkout.addressHint)
                    } else {
                        GlassField(comment, { comment = it }, t.checkout.comment)
                    }

                    Text(
                        t.checkout.payment,
                        style = MaterialTheme.typography.titleMedium,
                        color = c.ink,
                    )
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        // ⚠️ Drawn from what the server said it can take — a
                        // hard-coded Payme button offers a payment that cannot
                        // complete.
                        methods.forEach { m ->
                            Chip(paymentLabel(m), payment == m) { payment = m }
                        }
                    }
                    GlassField(promo, { promo = it }, t.checkout.promo)
                }
            }

            item {
                val q = quote
                Column(
                    Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp)).padding(12.dp),
                    verticalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    Line(t.checkout.subtotal, q?.subtotal ?: cart.subtotal)
                    if ((q?.discountTotal ?: 0.0) > 0) {
                        Line(t.checkout.discount, -(q?.discountTotal ?: 0.0))
                    }
                    if (type == "delivery") Line(t.checkout.deliveryFee, q?.deliveryFee ?: 0.0)
                    Line(t.checkout.total, q?.total ?: cart.subtotal, strong = true)

                    // ⚠️ **The refusals are stated before the button is pressed,
                    // and each says what to do about it.** "Buyurtma berilmadi"
                    // after a tap is the same information arriving too late to
                    // act on.
                    if (q != null) {
                        if (q.codeError.isNotEmpty()) {
                            Text(q.codeError, style = MaterialTheme.typography.labelMedium, color = c.warn)
                        }
                        if (q.belowMinimum) {
                            Text(
                                t.checkout.minOrder(uz.keel.design.money(q.minOrder)),
                                style = MaterialTheme.typography.labelMedium,
                                color = c.warn,
                            )
                        }
                        if (type == "delivery" && point != null && !q.available) {
                            Text(
                                t.checkout.notDelivered,
                                style = MaterialTheme.typography.labelMedium,
                                color = c.danger,
                            )
                        }
                        if (q.soldOut.isNotEmpty()) {
                            Text(
                                t.checkout.soldOut(q.soldOut.joinToString(", ")),
                                style = MaterialTheme.typography.labelMedium,
                                color = c.danger,
                            )
                        }
                        if (q.pointsEarn > 0) {
                            Text(
                                t.checkout.pointsEarn(uz.keel.design.money(q.pointsEarn)),
                                style = MaterialTheme.typography.labelMedium,
                                color = c.muted,
                            )
                        }
                    }
                }
            }
        }

        Column(
            Modifier
                .align(Alignment.BottomCenter)
                .padding(
                    start = 16.dp,
                    end = 16.dp,
                    bottom = bottomInset.calculateBottomPadding() + 10.dp,
                )
                .fillMaxWidth(),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            if (error.isNotEmpty()) {
                Text(error, style = MaterialTheme.typography.bodyMedium, color = c.danger)
            }
            PrimaryButton(t.checkout.send, Modifier.fillMaxWidth(), busy = busy) { send() }
            GhostButton(t.common.back, Modifier.fillMaxWidth()) { onBack() }
        }
    }
}

@Composable
private fun Line(label: String, amount: Double, strong: Boolean = false) {
    val c = KeelTheme.colors
    Row(verticalAlignment = Alignment.CenterVertically) {
        Text(
            label,
            Modifier.weight(1f),
            style = if (strong) MaterialTheme.typography.titleMedium
            else MaterialTheme.typography.bodyMedium,
            color = if (strong) c.ink else c.muted,
        )
        Money(amount, color = if (strong) c.ink else c.inkSoft)
    }
}

/** What a payment method is called on a button.
 *
 *  ⚠️ **The provider's own name, untranslated.** "Payme" is a brand a guest
 *  recognises on a bank app; translating it would be inventing a word for
 *  something they are about to be handed to. */
@Composable
private fun paymentLabel(method: String): String = when (method) {
    "cash" -> t.checkout.payCash
    "card" -> t.checkout.payCard
    else -> method.replaceFirstChar { it.uppercase() }
}
