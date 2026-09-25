package uz.keel.courier

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.os.Build
import uz.keel.design.TokenStore
import uz.keel.courier.data.DeviceInfo
import uz.keel.courier.data.KeelApi
import uz.keel.courier.location.LocationService

// What one process owns.
//
// ⚠️ **A container by hand rather than a dependency-injection framework.** There
// are three things in it and they are created once; a graph and its generated
// code would be machinery carrying a single decision.
//
// ⚠️ **The service reads from here too.** `LocationService` runs outside the
// composition and outlives the activity on purpose, and it sends through this
// same `api` — a second client would be a second base address and a second
// token, and the one that goes stale is the one nobody is looking at.
class KeelCourierApp : Application() {

    lateinit var tokens: TokenStore
        private set
    lateinit var api: KeelApi
        private set
    lateinit var prefs: Prefs
        private set

    override fun onCreate() {
        super.onCreate()
        // ⚠️ **All of this before the first screen and before the first
        // request.** A screen that fetched while the store was empty would read
        // the token as absent and send a courier to a login they had already
        // passed — and only on a cold start, which is the hardest kind of bug to
        // be shown.
        tokens = TokenStore(this)
        api = KeelApi(tokens)
        prefs = Prefs(tokens)
        api.device = DeviceInfo(
            id = tokens.deviceId(),
            app = "courier",
            // ⚠️ A model name, never a serial: the row exists so somebody can
            // say "that is my old phone".
            name = "${Build.MANUFACTURER} ${Build.MODEL}".trim(),
        )
        tokens.read(TokenStore.SERVER_ADDRESS)?.let { api.useServer(it) }
        createDeliveryChannel()
        // ⚠️ Created at launch rather than when the shift opens: Android kills
        // the process if a foreground service posts to a channel that does not
        // exist yet, and the shift is opened at a kerb.
        LocationService.ensureChannel(this)
    }

    /** ⚠️ **Created before the first notification can arrive**, or Android files
     *  it under a default channel with no sound. The id has to match
     *  `push.DeliveryChannel` on the server exactly — a message sent to a
     *  channel this phone never created is dropped silently, and the only
     *  symptom is a restaurant saying they sent something.
     *
     *  ⚠️ **Its own channel, not the kitchen's.** Android lets somebody switch a
     *  channel off, and a courier who muted the kitchen's chime must not have
     *  muted "your order was cancelled" with it. */
    private fun createDeliveryChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val channel = NotificationChannel(
            DELIVERY_CHANNEL,
            "Buyurtmalar",
            NotificationManager.IMPORTANCE_HIGH,
        ).apply {
            enableVibration(true)
            vibrationPattern = longArrayOf(0, 250, 250, 250)
        }
        getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
    }

    companion object {
        const val DELIVERY_CHANNEL = "delivery"
    }
}
