package uz.keel.guest

import android.app.Application
import uz.keel.design.TokenStore
import uz.keel.guest.data.KeelApi

/** The one place this application's long-lived objects are made.
 *
 *  ⚠️ **No server screen and no sign-in gate at startup**, unlike the five staff
 *  applications: the address is a constant of the build and the menu is public.
 *  What is left to do before the first frame is nothing, which is the point —
 *  a restaurant application that thinks before it draws the menu is one a guest
 *  closes. */
class KeelGuestApp : Application() {

    lateinit var prefs: Prefs
        private set

    lateinit var tokens: TokenStore
        private set

    lateinit var api: KeelApi
        private set

    /** ⚠️ **One basket for the whole application, held here.** A basket owned by
     *  a screen empties when that screen leaves the composition — which happens
     *  on every rotation, and on the tab switch a guest makes to check a price. */
    lateinit var cart: Cart
        private set

    /** The order numbers this phone has placed.
     *
     *  ⚠️ **Kept locally rather than read from the account.** A guest orders
     *  without signing in — that is the ordinary first order — and an "Orders"
     *  tab that was empty until they had a login would be a tab that answers
     *  "where is my food" with nothing. The account's own history arrives with
     *  the sign-in and joins this, it does not replace it. */
    lateinit var placed: PlacedOrders
        private set

    /** ⚠️ Held here rather than by the menu screen: the heart is drawn on a row
     *  that scrolls out of the list, and state owned there would forget itself
     *  on every scroll. */
    val favorites = Favorites()

    override fun onCreate() {
        super.onCreate()
        prefs = Prefs(this)
        tokens = TokenStore(this)
        api = KeelApi(tokens).also { it.lang = prefs.lang.value.code }
        cart = Cart(this)
        placed = PlacedOrders(this)
    }
}
