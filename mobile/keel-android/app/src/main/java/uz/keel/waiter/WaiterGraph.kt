package uz.keel.waiter

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import uz.keel.app.KeelLook
import uz.keel.app.deviceInfoFor
import uz.keel.design.TokenStore
import uz.keel.waiter.data.DeviceInfo
import uz.keel.waiter.data.FloorStore
import uz.keel.waiter.data.KeelApi
import uz.keel.waiter.data.MenuCache
import uz.keel.waiter.data.Outbox

// What the waiter's half of Keel owns.
//
// ⚠️ **Made the first time somebody opens the floor, not when the app starts.**
// In Keel Waiter this was the `Application` itself; here it is one of four, and
// an owner who never carries a plate should not have the waiter's queue opened
// on their phone. `KeelApp.waiter` creates it once and keeps it: the outbox
// retries on its own loop, and two of them over one database would send every
// dish twice.
class WaiterGraph(context: Context, val tokens: TokenStore, look: KeelLook) {

    val api: KeelApi = KeelApi(tokens)
    val prefs: Prefs = Prefs(tokens, look)
    /** ⚠️ Opened before the first screen can queue anything: a tap queued while
     *  the store was unset would be written to nowhere and reported as saved. */
    val outbox: Outbox
    /** The room and the menu, kept between screens — see `FloorStore.kt`. */
    val floor: FloorStore
    val menu: MenuCache

    init {
        // ⚠️ **`keel-staff`, not `waiter`.** The server binds one phone per
        // account *per app*, and Keel Waiter may still be installed beside this
        // during the move — sharing its key would make each of them evict the
        // other at every sign-in. One key for the whole staff account: the
        // floor and the team section are the same person on the same phone.
        api.device = deviceInfoFor("keel-staff", tokens).let { DeviceInfo(it.id, it.app, it.name) }
        tokens.serverAddress?.let { api.useServer(it) }
        outbox = Outbox(context.applicationContext, api)
        outbox.startRetrying()
        floor = FloorStore(api)
        menu = MenuCache(api)
        watchNetwork(context.applicationContext)
    }

    /** What a sign-out leaves behind: nothing of the last person's room. */
    fun clear() {
        floor.clear()
        menu.clear()
    }

    /** ⚠️ **The queue is drained the moment the network is back**, not on the
     *  next tick of the retry loop. Walking out of the cellar with four dishes
     *  held used to mean up to eight more seconds of "yuborilmoqda" with full
     *  signal — long enough for the waiter to tap them again. */
    private fun watchNetwork(context: Context) {
        val cm = context.getSystemService(ConnectivityManager::class.java) ?: return
        runCatching {
            cm.registerDefaultNetworkCallback(object : ConnectivityManager.NetworkCallback() {
                override fun onAvailable(network: Network) {
                    outbox.flushSoon()
                }
            })
        }
    }
}
