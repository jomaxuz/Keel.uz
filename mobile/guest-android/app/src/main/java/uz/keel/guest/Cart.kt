package uz.keel.guest

import android.content.Context
import androidx.compose.runtime.mutableStateListOf
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import uz.keel.guest.data.MenuItem
import uz.keel.guest.data.MenuOption
import uz.keel.guest.data.OptionChoice

// ---- What the guest has chosen, and the one place it lives ----
//
// ⚠️ **The basket survives the app being closed.** Somebody builds an order,
// gets a call, comes back twenty minutes later — and an empty basket at that
// moment is an order that never happens. It is written on every change rather
// than on the way out: `onStop` is not guaranteed and the write is a few hundred
// bytes.
//
// ⚠️ **Prices here are for display only.** Every figure the guest is charged
// comes from `/orders/quote`; a total added up on the phone that disagrees with
// what the card is debited is the worst thing this app can do, and a promo rule
// the app does not know about is enough to cause it.

/** One chosen option: which question, which answer, and what it added. */
@Serializable
data class ChosenOption(
    /** ⚠️ **Always the base (Uzbek) names.** The server re-resolves the price
     *  from them, so a Russian-speaking guest's order must not send Russian
     *  labels — it would arrive as an option the menu has no row for. */
    val group: String,
    val choice: String,
    val priceDelta: Double,
)

/** One line of the basket. */
@Serializable
data class CartLine(
    val menuItemId: String,
    /** Frozen at the moment it was added, like every other name this product
     *  stores: a dish renamed overnight must not rewrite what somebody chose. */
    val name: String,
    val price: Double,
    val qty: Int,
    val comment: String = "",
    val options: List<ChosenOption> = emptyList(),
    val imageUrl: String = "",
) {
    /** What this line shows on the basket screen. ⚠️ Display only — see the
     *  file's note. */
    val lineTotal: Double get() = (price + options.sumOf { it.priceDelta }) * qty

    /** ⚠️ **Two lines of the same dish are the same line only if every answer
     *  matches**, comment included. A plov with extra meat and a plov without
     *  are two things to cook, and merging them by dish id would send the
     *  kitchen one ticket for a dish nobody ordered. */
    val key: String
        get() = menuItemId + "|" + comment + "|" +
            options.joinToString(",") { it.group + "=" + it.choice }

    internal fun quoteJson(): JsonObject = buildJsonObject {
        put("menuItemId", JsonPrimitive(menuItemId))
        put("qty", JsonPrimitive(qty))
        put("options", JsonArray(options.map { it.json() }))
    }

    internal fun orderJson(): JsonObject = buildJsonObject {
        put("menuItemId", JsonPrimitive(menuItemId))
        put("name", JsonPrimitive(name))
        put("price", JsonPrimitive(price))
        put("qty", JsonPrimitive(qty))
        put("comment", JsonPrimitive(comment))
        put("options", JsonArray(options.map { it.json() }))
    }
}

private fun ChosenOption.json(): JsonObject = buildJsonObject {
    put("name", JsonPrimitive(group))
    put("choice", JsonPrimitive(choice))
    put("priceDelta", JsonPrimitive(priceDelta))
}

class Cart(context: Context) {

    private val store = context.getSharedPreferences("keel-guest-cart", Context.MODE_PRIVATE)
    private val json = Json { ignoreUnknownKeys = true }

    val lines = mutableStateListOf<CartLine>()

    init {
        // ⚠️ **A basket that will not parse is dropped, never thrown.** The
        // stored shape changes when a field is added, and an app that crashed on
        // launch because of last week's basket is an app somebody uninstalls
        // rather than debugs.
        runCatching {
            store.getString(KEY, null)?.let {
                lines.addAll(json.decodeFromString<List<CartLine>>(it))
            }
        }
    }

    val count: Int get() = lines.sumOf { it.qty }
    val subtotal: Double get() = lines.sumOf { it.lineTotal }

    fun add(line: CartLine) {
        val i = lines.indexOfFirst { it.key == line.key }
        if (i >= 0) lines[i] = lines[i].copy(qty = lines[i].qty + line.qty) else lines.add(line)
        save()
    }

    /** ⚠️ **Down to zero removes the line.** A stepper that stops at one leaves
     *  the guest hunting for a delete button, and the one they find is the one
     *  that empties the whole basket. */
    fun setQty(key: String, qty: Int) {
        val i = lines.indexOfFirst { it.key == key }
        if (i < 0) return
        if (qty <= 0) lines.removeAt(i) else lines[i] = lines[i].copy(qty = qty)
        save()
    }

    fun clear() {
        lines.clear()
        save()
    }

    private fun save() {
        store.edit()
            .putString(KEY, json.encodeToString(lines.toList()))
            .apply()
    }

    private companion object {
        const val KEY = "lines"
    }
}

/** Turn a dish and a set of answers into a basket line. */
fun cartLineOf(
    item: MenuItem,
    qty: Int,
    comment: String,
    chosen: Map<MenuOption, List<OptionChoice>>,
): CartLine = CartLine(
    menuItemId = item.id,
    name = item.name,
    price = item.price,
    qty = qty,
    comment = comment,
    options = chosen.flatMap { (group, choices) ->
        choices.map { ChosenOption(group.name, it.name, it.priceDelta) }
    },
    imageUrl = item.imageUrl,
)

/** The order numbers this phone has placed.
 *
 *  ⚠️ **Numbers, not orders.** What an order *is* changes on the server every
 *  few minutes — that is the whole point of tracking it — and a copy kept here
 *  would be a second, staler answer to "what did I order", shown beside the
 *  live one. The number is the only part that never changes.
 *
 *  ⚠️ **Newest first and capped.** A guest who orders weekly for two years does
 *  not want to scroll 2023 to find tonight, and the account's own history is the
 *  place for the long list.
 */
class PlacedOrders(context: Context) {

    private val store = context.getSharedPreferences("keel-guest-orders", Context.MODE_PRIVATE)
    val numbers = mutableStateListOf<String>()

    init {
        runCatching {
            store.getString(KEY, null)?.split('\n')?.filter { it.isNotBlank() }?.let {
                numbers.addAll(it)
            }
        }
    }

    fun remember(number: String) {
        if (number.isBlank()) return
        numbers.remove(number)
        numbers.add(0, number)
        while (numbers.size > LIMIT) numbers.removeAt(numbers.lastIndex)
        store.edit().putString(KEY, numbers.joinToString("\n")).apply()
    }

    private companion object {
        const val KEY = "numbers"
        const val LIMIT = 20
    }
}

/** The dishes this guest keeps.
 *
 *  ⚠️ **Held for the whole app, not by the menu screen.** The heart is drawn on
 *  a row that scrolls in and out of the list; state owned there would forget
 *  itself on every scroll, and the tap that saved a dish would appear to undo
 *  itself.
 *
 *  ⚠️ **Empty when signed out, and the heart is simply not drawn.** A heart that
 *  opens a sign-in form is a menu screen that asks for a phone number — which is
 *  the one thing this app never does before somebody has eaten. */
class Favorites {
    val ids = mutableStateListOf<String>()

    fun holds(id: String): Boolean = ids.contains(id)

    fun replace(all: List<String>) {
        ids.clear()
        ids.addAll(all)
    }

    fun clear() = ids.clear()
}
