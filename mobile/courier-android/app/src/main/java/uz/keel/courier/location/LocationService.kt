package uz.keel.courier.location

import android.Manifest
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.IBinder
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import com.google.android.gms.location.LocationCallback
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationResult
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import uz.keel.courier.KeelCourierApp
import uz.keel.courier.MainActivity
import uz.keel.courier.R
import uz.keel.courier.data.Point

// Reporting a position while the phone is in a pocket.
//
// ⚠️ **This is the whole reason a courier needs an app rather than the PWA.** A
// browser gives a backgrounded tab no geolocation and eventually no CPU, so the
// position froze the moment the rider put the phone away — and the freeze is
// invisible from both ends until somebody needs it. The visible half is a pin
// stuck on the dispatcher's map. The half that costs money is that "delivered"
// stops opening: the server judges by the last *reported* position and refuses
// one older than ten minutes, so the courier stands at the door unable to close
// the order they have just handed over.
//
// ⚠️ **A foreground service with the ordinary while-in-use permission, and no
// background-location permission at all.** Android treats a running foreground
// service of type `location` as the app being in use, so this keeps receiving
// fixes with the screen off and the app in a pocket — which is the shift. What
// is given up is the case where the system kills the app outright: the service
// restarts, has no foreground to be part of, and gets nothing. The alternative
// costs a second, scarier dialog and a Play Store review that asks for a video,
// to buy back a case that ends with the courier reopening an app they are
// holding anyway.
//
// ⚠️ **The notification is the price and it is worth paying.** Android only
// keeps location running for an app it can show the user is running, and hiding
// that would be both impossible and wrong: somebody whose phone is reporting
// where they are should be able to see that it is, and stop it by ending the
// shift.
class LocationService : Service() {

    private lateinit var scope: CoroutineScope
    private var flusher: Job? = null
    private val client by lazy { LocationServices.getFusedLocationProviderClient(this) }

    /** Fixes waiting for a network.
     *
     *  ⚠️ **Capped.** A courier riding an hour through a dead cell would
     *  otherwise arrive with a thousand points, and the server keeps only the
     *  newest one anyway (`CourierUpdateLocation`). The tail is what matters. */
    private val buffer = ArrayDeque<Point>()

    private val callback = object : LocationCallback() {
        override fun onLocationResult(result: LocationResult) {
            val last = result.lastLocation ?: return
            val fix = Fix(
                lat = last.latitude,
                lng = last.longitude,
                accuracy = last.accuracy.toDouble(),
                at = if (last.time > 0) last.time else System.currentTimeMillis(),
            )
            synchronized(buffer) {
                buffer.addLast(Point(fix.lat, fix.lng, fix.accuracy, fix.at))
                while (buffer.size > MAX_BUFFER) buffer.removeFirst()
            }
            Tracking.publish {
                it.copy(fix = fix, live = true, pending = synchronized(buffer) { buffer.size })
            }
        }
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
        ensureChannel(this)
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        // ⚠️ **Called before anything else can fail.** Android gives a service
        // a few seconds to show its notification and crashes the process if it
        // does not — including on the path where the permission was revoked
        // between the shift opening and this running.
        startForeground(NOTE_ID, notification())

        if (!hasLocationPermission(this)) {
            // Revoked while the shift was open, which Android allows. Nothing
            // here can ask — the dialog belongs to an activity — so the service
            // stops and the card on screen goes back to asking.
            stopSelf()
            return START_NOT_STICKY
        }

        val request = LocationRequest.Builder(INTERVAL_MS)
            // ⚠️ **Balanced, not the highest accuracy.** The arrival radius a
            // restaurant sets is 50–150 m and a phone's GPS is 10–30 outdoors,
            // so the extra seconds and battery the highest setting spends do not
            // change the answer — and they are spent by somebody standing at a
            // gate.
            .setPriority(Priority.PRIORITY_BALANCED_POWER_ACCURACY)
            .setMinUpdateDistanceMeters(DISTANCE_M)
            // A courier stopped at a light still exists. Without this the
            // provider can go quiet for minutes, and the last fix ages past the
            // server's ten-minute window while the bike is not moving.
            .setMinUpdateIntervalMillis(INTERVAL_MS)
            .build()

        runCatching { client.requestLocationUpdates(request, callback, mainLooper) }
            .onFailure {
                Log.i(TAG, "location updates refused: ${it.message}")
                stopSelf()
            }

        if (flusher?.isActive != true) {
            flusher = scope.launch {
                while (isActive) {
                    delay(FLUSH_MS)
                    flush()
                }
            }
        }
        Tracking.publish { it.copy(running = true) }
        // ⚠️ **Not sticky.** A service the system restarts on its own has no
        // foreground to belong to and no permission to use, so it would run,
        // report nothing, and hold a notification saying it was working. The
        // shift is restarted by the app, which can ask.
        return START_NOT_STICKY
    }

    override fun onDestroy() {
        runCatching { client.removeLocationUpdates(callback) }
        // One last attempt with whatever is in hand: the buffer dies with this
        // process, and the fix in it is the one that would open the button.
        scope.launch { flush() }
        flusher = null
        Tracking.publish { it.copy(running = false, live = false) }
        scope.cancel()
        super.onDestroy()
    }

    /** Empty the buffer towards the server.
     *
     *  ⚠️ **Taken before the request and cleared after it succeeds**: a fix that
     *  arrives mid-send must not be dropped with the batch it was not in. */
    private suspend fun flush() {
        val batch = synchronized(buffer) { buffer.toList() }
        if (batch.isEmpty()) return
        val app = application as? KeelCourierApp ?: return
        try {
            app.api.sendLocation(batch)
            synchronized(buffer) { repeat(batch.size) { if (buffer.isNotEmpty()) buffer.removeFirst() } }
            Tracking.publish {
                it.copy(
                    lastSentAt = System.currentTimeMillis(),
                    pending = synchronized(buffer) { buffer.size },
                )
            }
        } catch (e: Throwable) {
            // Offline, or the server is down. The buffer keeps its points and
            // the next tick tries again — this is a bike in a basement, not an
            // error anybody should be shown.
            Log.i(TAG, "location send failed: ${e.message}")
        }
    }

    private fun notification(): Notification {
        val open = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP
            },
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        return NotificationCompat.Builder(this, CHANNEL)
            .setSmallIcon(R.drawable.ic_notification)
            .setColor(ContextCompat.getColor(this, R.color.keel_orange))
            .setContentTitle(getString(R.string.shift_running))
            .setContentText(getString(R.string.shift_sending))
            .setOngoing(true)
            // ⚠️ Low, not the delivery channel's high: this one says nothing
            // new. A courier who is buzzed by the fact that their shift is open
            // learns to mute the app, and the channel they mute next is the one
            // that carries orders.
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .setContentIntent(open)
            .build()
    }

    companion object {
        const val CHANNEL = "shift"
        private const val NOTE_ID = 42
        private const val TAG = "KeelCourier"

        /** How often the buffer is emptied towards the server, and how often a
         *  fix is asked for.
         *
         *  ⚠️ **Not every fix, and not once a minute.** Every fix is a request a
         *  second on a moving bike — battery and mobile data the courier pays
         *  for. Once a minute is worse in the other direction: the server calls
         *  a position older than ten minutes unusable, and a dispatcher watching
         *  a map wants a courier who moves rather than teleports. */
        private const val INTERVAL_MS = 15_000L
        private const val FLUSH_MS = 15_000L

        /** ⚠️ Anything closer is a phone sitting still and reporting jitter,
         *  which costs data and walks the map pin around a parked bike. */
        private const val DISTANCE_M = 25f

        private const val MAX_BUFFER = 50

        fun hasLocationPermission(ctx: Context): Boolean =
            ContextCompat.checkSelfPermission(ctx, Manifest.permission.ACCESS_FINE_LOCATION) ==
                PackageManager.PERMISSION_GRANTED ||
                ContextCompat.checkSelfPermission(ctx, Manifest.permission.ACCESS_COARSE_LOCATION) ==
                PackageManager.PERMISSION_GRANTED

        /** ⚠️ **Created before the service starts**, or Android kills the
         *  process for showing a notification on a channel that does not
         *  exist. */
        fun ensureChannel(ctx: Context) {
            if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
            val channel = NotificationChannel(
                CHANNEL,
                ctx.getString(R.string.shift_running),
                NotificationManager.IMPORTANCE_LOW,
            ).apply { setShowBadge(false) }
            ctx.getSystemService(NotificationManager::class.java)
                .createNotificationChannel(channel)
        }

        fun start(ctx: Context) {
            if (!hasLocationPermission(ctx)) return
            ensureChannel(ctx)
            val intent = Intent(ctx, LocationService::class.java)
            // ⚠️ `startForegroundService` above Oreo, and the service has a few
            // seconds to call `startForeground` or the process is killed — which
            // is why that call is the first line of `onStartCommand`.
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                ctx.startForegroundService(intent)
            } else {
                ctx.startService(intent)
            }
        }

        /** ⚠️ Called when the shift ends and on sign-out. A service that
         *  outlives the shift is a battery complaint and, worse, a phone still
         *  telling a restaurant where somebody is after they have gone home. */
        fun stop(ctx: Context) {
            ctx.stopService(Intent(ctx, LocationService::class.java))
            Tracking.clear()
        }
    }
}
