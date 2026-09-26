package uz.keel.guest.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Directions
import androidx.compose.material.icons.rounded.Storefront
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
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
import uz.keel.guest.data.PickupBranch
import uz.keel.guest.data.Restaurant
import uz.keel.guest.data.User
import uz.keel.guest.data.UserAddress
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
    brandId: String,
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
    val context = LocalContext.current

    // ---- Pickup: which door ----
    //
    // ⚠️ **Always sent, and always shown.** Without a `branchId` the server
    // hands a pickup to the first branch in the list; in a chain that is a
    // guest walking to the wrong building, and with one branch it is still the
    // address they need before leaving the house.
    var branches by remember { mutableStateOf(listOf<PickupBranch>()) }
    var pickupBranch by remember { mutableStateOf("") }

    /** Who is ordering, when the phone knows. ⚠️ Held rather than only read
     *  once: saving a new address means sending the whole list back, so the
     *  list has to still be here at the moment the order is placed. */
    var me by remember { mutableStateOf<User?>(null) }
    var saveAddress by remember { mutableStateOf(false) }

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
    LaunchedEffect(brandId) {
        runCatching { api.branches().branches }.onSuccess { all ->
            branches = all.filter { brandId.isBlank() || it.brandId == brandId }
            if (branches.none { it.id == pickupBranch }) pickupBranch = branches.firstOrNull()?.id.orEmpty()
        }
    }

    // ---- The form is not blank for somebody the restaurant already knows ----
    //
    // ⚠️ **Name, phone and the addresses they have used before**, exactly as the
    // site fills them in. Retyping a phone number that the app signed in with
    // two screens ago is the kind of small insult that makes an app feel like a
    // worse version of the website it came from.
    //
    // ⚠️ **Only into empty fields.** This runs when the profile arrives, which
    // may be after the guest has started typing — overwriting what they wrote
    // would be a form that argues with them.
    LaunchedEffect(Unit) {
        if (!api.signedIn()) return@LaunchedEffect
        runCatching { api.me() }.onSuccess { u ->
            me = u
            if (name.isBlank()) {
                name = listOf(u.firstName, u.lastName).filter { it.isNotBlank() }.joinToString(" ")
            }
            if (phone.isBlank()) phone = u.phone
            // ⚠️ The most recent one, which is the last row: the list is
            // appended to, so the newest address is the one at the end and the
            // one they are most likely to want again.
            if (point == null) {
                u.addresses.lastOrNull()?.let { a ->
                    point = GeoPoint(lat = a.lat, lng = a.lng, text = a.text)
                    if (comment.isBlank()) comment = a.comment
                }
            }
        }
    }

    // ⚠️ Re-asked on every input the price depends on — see the file's note.
    LaunchedEffect(cart.lines.toList(), type, point, promo, pickupBranch) {
        runCatching {
            quote = api.quote(
                cart.lines.toList(), type, point.takeIf { type == "delivery" }, promo, 0.0,
                branchId = pickupBranch,
            )
            error = ""
        }.onFailure { e -> error = if (e is ApiError) e.message else failedWord }
    }

    if (picking) {
        MapPickerScreen(
            // ⚠️ Opens on the restaurant, not on the guest: see MapPickerScreen.
            start = point ?: restaurant?.address ?: GeoPoint(),
            // ⚠️ **The restaurant's own setting, read at runtime.** The build's
            // baked-in Google key is only the fallback for a restaurant that has
            // not filled in its own — see Restaurant.mapKey.
            provider = restaurant?.provider ?: "2gis",
            mapKey = (restaurant?.mapKey ?: "").ifBlank {
                if (restaurant?.provider == "google" || restaurant == null) {
                    Brand.mapsKey
                } else {
                    ""
                }
            },
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
                    branchId = pickupBranch,
                )
                // ⚠️ **After the order, and never allowed to fail it.**
                // Remembering an address is a convenience; an order that was
                // accepted by the kitchen and reported as failed because a
                // profile write timed out would be the worst trade in this
                // file.
                val u = me
                if (saveAddress && u != null && type == "delivery" && point != null) {
                    runCatching {
                        me = api.updateMe(
                            u.firstName,
                            u.lastName,
                            u.addresses + UserAddress(
                                text = point?.text.orEmpty(),
                                lat = point?.lat ?: 0.0,
                                lng = point?.lng ?: 0.0,
                                comment = comment.trim(),
                            ),
                        )
                    }
                }
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
                        // ⚠️ **Saved addresses first, the map second.** A guest
                        // who orders to the same flat every week should be one
                        // tap from done; drawing the map first makes the common
                        // case the slow one.
                        val saved = me?.addresses.orEmpty()
                        if (saved.isNotEmpty()) {
                            Text(
                                t.checkout.savedAddresses,
                                style = MaterialTheme.typography.labelMedium,
                                color = c.muted,
                            )
                            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                                saved.forEach { a ->
                                    Chip(
                                        a.label.ifBlank { a.text }.take(64),
                                        point?.lat == a.lat && point?.lng == a.lng,
                                    ) {
                                        point = GeoPoint(lat = a.lat, lng = a.lng, text = a.text)
                                        comment = a.comment
                                        saveAddress = false
                                    }
                                }
                            }
                        }
                        GhostButton(
                            if (saved.isEmpty()) t.checkout.addressPick else t.checkout.newAddress,
                            Modifier.fillMaxWidth(),
                        ) {
                            picking = true
                        }
                        GlassField(comment, { comment = it }, t.checkout.addressHint)
                        // ⚠️ Offered only for an address that is not already on
                        // the profile: a tick that saves a duplicate teaches the
                        // guest to ignore the list it fills.
                        val known = me?.addresses.orEmpty().any {
                            it.lat == point?.lat && it.lng == point?.lng
                        }
                        if (me != null && point != null && !known) {
                            Chip(t.checkout.saveAddress, saveAddress) {
                                saveAddress = !saveAddress
                            }
                        }
                    } else {
                        PickupBranches(branches, pickupBranch, { pickupBranch = it }) { b ->
                            openRoute(context, b)
                        }
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

/** Where a pickup is collected: the branch's name and street, picked when there
 *  are several, and a route straight into the phone's own map app. */
@Composable
private fun PickupBranches(
    branches: List<PickupBranch>,
    selected: String,
    onSelect: (String) -> Unit,
    onRoute: (PickupBranch) -> Unit,
) {
    val c = KeelTheme.colors
    if (branches.isEmpty()) return
    Text(t.checkout.pickupFrom, style = MaterialTheme.typography.titleMedium, color = c.ink)
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        branches.forEach { b ->
            val on = b.id == selected
            val shape = RoundedCornerShape(16.dp)
            Row(
                Modifier
                    .fillMaxWidth()
                    .clip(shape)
                    .then(
                        if (on) Modifier.background(c.accentSoft).border(1.5.dp, c.accent, shape)
                        else Modifier.glass(c, shape),
                    )
                    .clickable(enabled = branches.size > 1) { onSelect(b.id) }
                    .padding(horizontal = 14.dp, vertical = 12.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    Icons.Rounded.Storefront,
                    null,
                    tint = if (on) c.accent else c.muted,
                    modifier = Modifier.size(22.dp),
                )
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Text(b.name, style = MaterialTheme.typography.titleSmall, color = c.ink)
                    Text(
                        b.address.text.ifBlank { t.checkout.pickupNoBranch },
                        style = MaterialTheme.typography.bodySmall,
                        color = c.muted,
                    )
                }
                if (on && b.address.lat != 0.0) {
                    Row(
                        Modifier
                            .clip(RoundedCornerShape(50))
                            .background(c.accent)
                            .clickable { onRoute(b) }
                            .padding(horizontal = 12.dp, vertical = 8.dp),
                        horizontalArrangement = Arrangement.spacedBy(4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(Icons.Rounded.Directions, null, tint = c.onAccent, modifier = Modifier.size(16.dp))
                        Text(t.checkout.pickupRoute, style = MaterialTheme.typography.labelMedium, color = c.onAccent)
                    }
                }
            }
        }
    }
}

/** Hand the route to whichever map app the guest already uses. */
private fun openRoute(context: android.content.Context, b: PickupBranch) {
    val lat = b.address.lat
    val lng = b.address.lng
    val uri = android.net.Uri.parse("geo:$lat,$lng?q=$lat,$lng(${android.net.Uri.encode(b.name)})")
    runCatching {
        context.startActivity(
            android.content.Intent(android.content.Intent.ACTION_VIEW, uri)
                .addFlags(android.content.Intent.FLAG_ACTIVITY_NEW_TASK),
        )
    }
}
