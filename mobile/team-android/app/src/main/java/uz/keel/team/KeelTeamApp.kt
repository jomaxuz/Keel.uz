package uz.keel.team

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.os.Build
import uz.keel.design.TokenStore
import uz.keel.team.data.DeviceInfo
import uz.keel.team.data.KeelApi

// What one process owns.
//
// ⚠️ **A container by hand rather than a dependency-injection framework.** There
// are three things in it and they are created once; a graph and its generated
// code would be machinery carrying a single decision.
class KeelTeamApp : Application() {

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
        // the token as absent and send somebody to a login they had already
        // passed — and only on a cold start, which is the hardest kind of bug to
        // be shown.
        tokens = TokenStore(this)
        api = KeelApi(tokens)
        prefs = Prefs(tokens)
        api.device = DeviceInfo(
            id = tokens.deviceId(),
            app = "team",
            // ⚠️ A model name, never a serial: the row exists so somebody can
            // say "that is my old phone".
            name = "${Build.MANUFACTURER} ${Build.MODEL}".trim(),
        )
        tokens.read(TokenStore.SERVER_ADDRESS)?.let { api.useServer(it) }
        createTeamChannel()
    }

    /** ⚠️ **Created before the first notification can arrive**, or Android files
     *  it under a default channel with no sound. The id has to match
     *  `push.TeamChannel` on the server exactly — a message sent to a channel
     *  this phone never created is dropped silently, and the only symptom is a
     *  manager saying they announced something. */
    private fun createTeamChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val channel = NotificationChannel(
            TEAM_CHANNEL,
            "Ish haqi va grafik",
            NotificationManager.IMPORTANCE_HIGH,
        ).apply {
            enableVibration(true)
            vibrationPattern = longArrayOf(0, 250, 250, 250)
        }
        getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
    }

    companion object {
        const val TEAM_CHANNEL = "team"

        /** ⚠️ Here rather than in `BuildConfig` so the settings screen and the
         *  device row the panel shows are reading one string. */
        const val APP_VERSION = "2.0.0"
    }
}
