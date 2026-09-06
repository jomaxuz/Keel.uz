package uz.keel.tv

import android.app.Application
import android.os.Build
import uz.keel.design.TokenStore
import uz.keel.tv.data.DeviceInfo
import uz.keel.tv.data.KeelApi
import java.io.File

// What one process owns.
//
// ⚠️ **A container by hand rather than a dependency-injection framework.** There
// are six things in it, they are created once, and a graph with generated code
// would be machinery carrying a single decision — the same call the two phone
// applications made.
class KeelTvApp : Application() {

    lateinit var tokens: TokenStore
        private set
    lateinit var api: KeelApi
        private set
    lateinit var prefs: Prefs
        private set
    lateinit var clock: Clock
        private set
    lateinit var playlist: PlaylistStore
        private set
    lateinit var board: BoardPoller
        private set

    /** This television's own id.
     *
     *  ⚠️ **It is what makes re-pairing replace this screen's row rather than
     *  spend a second paid slot.** Not a serial or a MAC address: those are
     *  unavailable to ordinary apps, shared between identical cheap sets, or a
     *  privacy question we have no reason to open. A random id answers the only
     *  thing asked of it — is this the same television as before. */
    val installId: String get() = tokens.deviceId()

    override fun onCreate() {
        super.onCreate()
        // ⚠️ **All of this before the first screen and before the first
        // request.** A screen that fetched while the store was empty would read
        // the token as absent and show a pairing code — in a dining room, at
        // opening time, on a set that was paired months ago.
        tokens = TokenStore(this)
        api = KeelApi(tokens)
        prefs = Prefs(tokens)
        clock = Clock(tokens)
        // ⚠️ `filesDir`, never `cacheDir`: Android empties the cache when
        // storage runs low, and a television that quietly lost its playlist
        // overnight is a black screen at opening time.
        playlist = PlaylistStore(api, clock, File(filesDir, "tv-content"))
        board = BoardPoller(api, clock)
        api.device = DeviceInfo(
            id = installId,
            app = "tv",
            // ⚠️ A model name, never a serial: the row exists so somebody can
            // say "that is the screen over the counter".
            name = "${Build.MANUFACTURER} ${Build.MODEL}".trim(),
        )
        tokens.read(TokenStore.SERVER_ADDRESS)?.let { api.useServer(it) }
    }

    companion object {
        /** ⚠️ **The version string lives here rather than in `BuildConfig`**, for
         *  the reason the Expo build kept it in `App.tsx`: the panel's "this
         *  screen is running an old build" has to be a fact about the code
         *  actually answering, and it is sent on every heartbeat. */
        const val APP_VERSION = "2.0.0"
    }
}
