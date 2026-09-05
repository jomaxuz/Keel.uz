package uz.keel.waiter

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.os.Build
import uz.keel.waiter.data.DeviceInfo
import uz.keel.waiter.data.KeelApi
import uz.keel.waiter.data.Outbox
import uz.keel.design.TokenStore

// What one process owns.
//
// ⚠️ **A container by hand rather than a dependency-injection framework.** There
// are four things in it and they are created once; a graph, its annotations and
// its generated code would be machinery carrying a single decision. It goes in
// the day there is a reason, not before — the same rule the Expo app applied to
// a navigation library.
class KeelWaiterApp : Application() {

    lateinit var tokens: TokenStore
        private set
    lateinit var api: KeelApi
        private set
    lateinit var prefs: Prefs
        private set
    lateinit var outbox: Outbox
        private set

    override fun onCreate() {
        super.onCreate()
        // ⚠️ **All of this before the first screen and before the first
        // request.** A screen that fetched while the store was still empty would
        // read every token as absent and send somebody to a login they had
        // already passed — and only on a cold start, which is the hardest kind
        // of bug to be shown.
        tokens = TokenStore(this)
        api = KeelApi(tokens)
        prefs = Prefs(tokens)
        // ⚠️ After the store, because the id lives in it, and before the first
        // request, because the login is the call that most needs to carry it.
        api.device = DeviceInfo(
            id = tokens.deviceId(),
            app = "waiter",
            // ⚠️ A model name, never a serial: the row exists so somebody in the
            // office can say "that is my old phone".
            name = "${Build.MANUFACTURER} ${Build.MODEL}".trim(),
        )
        tokens.serverAddress?.let { api.useServer(it) }
        // ⚠️ Opened before the first screen, like the tokens: a tap queued while
        // the store was still unset would be written to nowhere and reported as
        // saved — the worst outcome this app can produce.
        outbox = Outbox(this, api)
        outbox.startRetrying()
        createKitchenChannel()
    }

    /** ⚠️ **Created before the first notification can arrive**, or Android files
     *  it under a default channel with no sound — a working registration and
     *  nothing anybody hears. The id has to match `push.KitchenChannel` on the
     *  server exactly; a message sent to a channel this phone never created is
     *  dropped silently. */
    private fun createKitchenChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val channel = NotificationChannel(
            KITCHEN_CHANNEL,
            "Oshxona",
            // ⚠️ HIGH, and not by reflex: a dish going cold at the pass is worth
            // waking a screen for. Anything that can wait until somebody next
            // looks at their phone must not use this, or the priority stops
            // meaning anything.
            NotificationManager.IMPORTANCE_HIGH,
        ).apply {
            enableVibration(true)
            vibrationPattern = longArrayOf(0, 250, 250, 250)
        }
        getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
    }

    companion object {
        const val KITCHEN_CHANNEL = "kitchen"
    }
}
