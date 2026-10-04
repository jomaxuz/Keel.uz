package uz.keel.app

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.os.Build
import uz.keel.app.data.AccountStore
import uz.keel.app.data.Kind
import uz.keel.app.data.ShellApi
import uz.keel.app.push.PushHub
import uz.keel.courier.CourierGraph
import uz.keel.courier.location.LocationService
import uz.keel.design.TokenStore
import uz.keel.owner.OwnerGraph
import uz.keel.team.TeamGraph
import uz.keel.waiter.WaiterGraph

// Keel: the waiter, the owner, the courier and the team, in one app.
//
// ⚠️ **Beside the four single-role apps, not instead of them** — its own
// application id (`uz.keel.app`), its own storage and its own device keys
// (`keel-owner`, `keel-staff`, `keel-courier`). A restaurant can move one phone
// at a time, and the old app on the next phone keeps working through the move.
// The television stays its own app: it is a screen on a wall, not a person.
//
// ⚠️ **A role's machinery is made the first time that role is opened.** An
// owner who never carries a plate does not get the waiter's offline queue opened
// on their phone, and a waiter does not pay for the owner's anything at launch.
// So the cost of uniting four apps is the cost of the one actually used.
class KeelApp : Application() {

    lateinit var tokens: TokenStore
        private set
    lateinit var look: KeelLook
        private set
    lateinit var accounts: AccountStore
        private set
    lateinit var shellApi: ShellApi
        private set
    lateinit var push: PushHub
        private set

    private val waiterLazy = lazy { WaiterGraph(this, tokens, look) }
    val waiter: WaiterGraph get() = waiterLazy.value
    val owner: OwnerGraph by lazy { OwnerGraph(tokens, look) }
    val courier: CourierGraph by lazy { CourierGraph(tokens, look) }
    val team: TeamGraph by lazy { TeamGraph(tokens, look) }

    override fun onCreate() {
        super.onCreate()
        // ⚠️ **All of this before the first screen and the first request** —
        // the ordering every Keel app depends on. A screen that read the store
        // before it was hydrated would see every token as absent.
        tokens = TokenStore(this)
        look = KeelLook(tokens)
        accounts = AccountStore(tokens)
        shellApi = ShellApi(tokens) { kind -> deviceInfoFor(appKey(kind), tokens) }
        push = PushHub(this, shellApi, accounts, look)
        createChannels()
        LocationService.ensureChannel(this)
        // ⚠️ **The one graph made at launch, and only for somebody who works
        // the floor.** Its queue may still hold dishes from before the phone was
        // closed, and they are sent by its retry loop — which used to start
        // with the app. Waiting for the floor to be opened would leave four
        // plates unsent while the waiter reads their hours.
        if (uz.keel.app.data.Workspace.Waiter in accounts.workspaces()) waiter
    }

    /** The waiter's graph, only if it was ever made — clearing a room nobody
     *  opened must not open the waiter's database to do it. */
    fun waiterIfMade(): WaiterGraph? = if (waiterLazy.isInitialized()) waiterLazy.value else null

    /** ⚠️ **All four, before the first notification can arrive**, or Android
     *  files it under a default channel with no sound — a working registration
     *  and nothing heard. The ids are the server's (`internal/push`) exactly:
     *  a message sent to a channel this phone never created is dropped
     *  silently. Four channels rather than one so that somebody who mutes the
     *  kitchen's chime has not muted the owner's loss alerts with it. */
    private fun createChannels() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val nm = getSystemService(NotificationManager::class.java)
        listOf(
            KITCHEN_CHANNEL to "Oshxona",
            DELIVERY_CHANNEL to "Buyurtmalar",
            OWNER_CHANNEL to "Diqqat",
            TEAM_CHANNEL to "Ish haqi va grafik",
        ).forEach { (id, name) ->
            nm.createNotificationChannel(
                NotificationChannel(id, name, NotificationManager.IMPORTANCE_HIGH).apply {
                    enableVibration(true)
                    vibrationPattern = longArrayOf(0, 250, 250, 250)
                },
            )
        }
    }

    companion object {
        const val KITCHEN_CHANNEL = "kitchen"
        const val DELIVERY_CHANNEL = "delivery"
        const val OWNER_CHANNEL = "owner"
        const val TEAM_CHANNEL = "team"

        /** The device key each account kind binds with — see `deviceInfoFor`. */
        fun appKey(kind: Kind?): String = when (kind) {
            Kind.Admin -> "keel-owner"
            Kind.Staff -> "keel-staff"
            Kind.Courier -> "keel-courier"
            null -> "keel"
        }
    }
}

/** Which install this is, for the server's one-phone-per-account rule
 *  (`handlers/logindevice.go`). */
data class AppDevice(val id: String, val app: String, val name: String)

/** ⚠️ **Keel's own keys, one per account kind, and never the single-role
 *  apps'.** The server binds an account to one phone *per app key*, unique on
 *  `(app, deviceId)`. Sharing `waiter` with Keel Waiter would make the two apps
 *  evict each other's binding at every sign-in on a phone that has both — the
 *  ordinary state during a move. And one key per *kind* rather than one for all
 *  of Keel: the binding is per account, so an owner who also holds a courier
 *  login keeps both on this phone without either displacing the other.
 *
 *  ⚠️ **The id is the install's own** (`TokenStore.deviceId`) and a reinstall
 *  mints a new one — the panel's "release this phone" button is the answer to
 *  that, as it is for the other apps. */
fun deviceInfoFor(app: String, tokens: TokenStore): AppDevice = AppDevice(
    id = tokens.deviceId(),
    app = app,
    // ⚠️ A model name, never a serial: the row exists so somebody in the office
    // can say "that is my old phone".
    name = "${Build.MANUFACTURER} ${Build.MODEL}".trim(),
)
