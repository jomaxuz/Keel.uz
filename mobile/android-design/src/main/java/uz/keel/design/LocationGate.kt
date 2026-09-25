package uz.keel.design

import android.Manifest
import android.annotation.SuppressLint
import android.app.Activity
import android.content.Context
import android.content.ContextWrapper
import android.content.Intent
import android.content.pm.PackageManager
import android.location.Location
import android.location.LocationManager
import android.net.Uri
import android.provider.Settings
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.IntentSenderRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.platform.LocalContext
import androidx.core.content.ContextCompat
import androidx.core.location.LocationManagerCompat
import com.google.android.gms.common.api.ResolvableApiException
import com.google.android.gms.location.CurrentLocationRequest
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.LocationSettingsRequest
import com.google.android.gms.location.Priority
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await
import kotlinx.coroutines.withTimeoutOrNull

// Getting to a usable position, for the three applications that need one.
//
// ⚠️ **One copy, because three copies had three different bugs.** The waiter's
// clock button asked for FINE alone — which Android 12+ *ignores* (the request
// has to carry COARSE too, or the system drops it and answers "denied" without
// showing anything); the team's asked correctly but never noticed location was
// switched off on the phone; nor did the courier's, and a courier with GPS off
// sat on "waiting for location" all evening.
//
// ⚠️ **"Approximate" is not a position for a 50-metre radius.** Android 12 lets
// somebody grant approximate location only, and an approximate fix is snapped
// to a grid of roughly two kilometres *on purpose*. The branch radius is fifty
// metres — so every punch from an approximate grant is refused as "you are
// 1400 m away", while the person is standing at the till. That read as a broken
// geofence. The gate asks for the upgrade instead, and says why if refused.
//
// ⚠️ **Refused twice is refused for good.** After the second "no" Android stops
// showing the dialog and answers "denied" instantly — so a button that only
// re-asks does nothing at all when pressed. The gate opens the app's own
// settings page then, which is the one place the answer can still change.

enum class LocationProblem { Denied, DeniedForever, Approximate, Off, NoFix }

class LocationGate internal constructor(
    /** Check everything, ask for whatever is missing, then run [onReady].
     *  Problems go to the callback given to [rememberLocationGate]. */
    val request: (onReady: suspend () -> Unit) -> Unit,
)

/**
 * @param precise true for a punch against a geofence; false where a rough
 *   position is still worth having (a courier's track).
 */
@Composable
fun rememberLocationGate(
    precise: Boolean = true,
    onProblem: (LocationProblem) -> Unit,
): LocationGate {
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    val problem by rememberUpdatedState(onProblem)
    val pending = remember { arrayOfNulls<suspend () -> Unit>(1) }

    fun finish() {
        val next = pending[0] ?: return
        pending[0] = null
        scope.launch { next() }
    }

    val resolve = rememberLauncherForActivityResult(
        ActivityResultContracts.StartIntentSenderForResult(),
    ) { res ->
        if (res.resultCode == Activity.RESULT_OK) finish()
        else { pending[0] = null; problem(LocationProblem.Off) }
    }

    /** Location switched on for the whole phone? Raises Android's own dialog,
     *  in place, when Play services can; otherwise sends to the settings page. */
    fun checkOn() {
        scope.launch {
            val req = LocationRequest.Builder(Priority.PRIORITY_HIGH_ACCURACY, 0L).build()
            val settings = LocationSettingsRequest.Builder().addLocationRequest(req).build()
            try {
                LocationServices.getSettingsClient(ctx).checkLocationSettings(settings).await()
                finish()
            } catch (e: ResolvableApiException) {
                resolve.launch(IntentSenderRequest.Builder(e.resolution).build())
            } catch (e: Throwable) {
                // No Play services to ask — the phone's own switch decides.
                if (isLocationOn(ctx)) finish()
                else {
                    pending[0] = null
                    problem(LocationProblem.Off)
                    runCatching { ctx.startActivity(Intent(Settings.ACTION_LOCATION_SOURCE_SETTINGS)) }
                }
            }
        }
    }

    val ask = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions(),
    ) { result ->
        val fine = result[Manifest.permission.ACCESS_FINE_LOCATION] == true || hasFine(ctx)
        val coarse = result[Manifest.permission.ACCESS_COARSE_LOCATION] == true || hasCoarse(ctx)
        when {
            fine || (coarse && !precise) -> checkOn()
            coarse -> { pending[0] = null; problem(LocationProblem.Approximate) }
            else -> {
                pending[0] = null
                // No rationale after a refusal means Android has stopped
                // showing the dialog at all: only the settings page is left.
                val activity = ctx.findActivity()
                val forever = activity != null &&
                    !activity.shouldShowRequestPermissionRationale(Manifest.permission.ACCESS_FINE_LOCATION)
                if (forever) {
                    problem(LocationProblem.DeniedForever)
                    openAppSettings(ctx)
                } else problem(LocationProblem.Denied)
            }
        }
    }

    return remember {
        LocationGate { onReady ->
            pending[0] = onReady
            val enough = hasFine(ctx) || (!precise && hasCoarse(ctx))
            if (enough) checkOn()
            // ⚠️ Both, always: FINE alone is dropped by Android 12+, and with
            // COARSE already granted this same call is the "use precise" upgrade.
            else ask.launch(
                arrayOf(Manifest.permission.ACCESS_FINE_LOCATION, Manifest.permission.ACCESS_COARSE_LOCATION),
            )
        }
    }
}

fun hasFine(ctx: Context): Boolean =
    ContextCompat.checkSelfPermission(ctx, Manifest.permission.ACCESS_FINE_LOCATION) ==
        PackageManager.PERMISSION_GRANTED

fun hasCoarse(ctx: Context): Boolean =
    ContextCompat.checkSelfPermission(ctx, Manifest.permission.ACCESS_COARSE_LOCATION) ==
        PackageManager.PERMISSION_GRANTED

fun isLocationOn(ctx: Context): Boolean {
    val lm = ctx.getSystemService(Context.LOCATION_SERVICE) as? LocationManager ?: return false
    return LocationManagerCompat.isLocationEnabled(lm)
}

fun openAppSettings(ctx: Context) {
    runCatching {
        ctx.startActivity(
            Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS, Uri.fromParts("package", ctx.packageName, null))
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
        )
    }
}

/** One fresh position, for a punch.
 *
 *  ⚠️ **High accuracy, bounded in time.** Balanced accuracy indoors is a
 *  wifi/cell fix, commonly 100–500 m off — against a 50 m radius that is a
 *  refusal for somebody standing inside the restaurant. GPS is asked for, for
 *  at most [timeoutMs]; a recent cached fix (under two minutes, and not worse
 *  than 100 m) is accepted if GPS cannot answer through the ceiling.
 *
 *  ⚠️ **Never an old `lastLocation`.** The last fix the phone happens to hold
 *  may be from home this morning — posted as "clocked in from home". */
@SuppressLint("MissingPermission")
suspend fun currentFix(ctx: Context, timeoutMs: Long = 15_000): Location? {
    val fused = runCatching { LocationServices.getFusedLocationProviderClient(ctx) }.getOrNull()
    if (fused != null) {
        val req = CurrentLocationRequest.Builder()
            .setPriority(Priority.PRIORITY_HIGH_ACCURACY)
            .setMaxUpdateAgeMillis(30_000)
            .setDurationMillis(timeoutMs)
            .build()
        val fresh = withTimeoutOrNull(timeoutMs + 2_000) {
            runCatching { fused.getCurrentLocation(req, null).await() }.getOrNull()
        }
        if (fresh != null) return fresh
        val last = runCatching { fused.lastLocation.await() }.getOrNull()
        if (last != null && last.isRecent() && last.accuracy <= 100f) return last
    }
    // No Play services: the platform's own providers.
    val lm = ctx.getSystemService(Context.LOCATION_SERVICE) as? LocationManager ?: return null
    return listOf(LocationManager.GPS_PROVIDER, LocationManager.NETWORK_PROVIDER)
        .mapNotNull { runCatching { lm.getLastKnownLocation(it) }.getOrNull() }
        .filter { it.isRecent() }
        .minByOrNull { it.accuracy }
}

private fun Location.isRecent(): Boolean =
    System.currentTimeMillis() - time < 2 * 60_000

private tailrec fun Context.findActivity(): Activity? = when (this) {
    is Activity -> this
    is ContextWrapper -> baseContext.findActivity()
    else -> null
}

/** The sentence for each problem, in the three languages.
 *
 *  ⚠️ Here rather than in each app's dictionary: the problems are the gate's,
 *  and three apps translating the same four sentences is three chances for one
 *  of them to say "turn on location" to somebody whose location is on. */
fun LocationProblem.message(lang: Lang): String = when (lang) {
    Lang.Uz -> when (this) {
        LocationProblem.Denied -> "Joylashuvga ruxsat berilmagan. Qayta bosing va ruxsat bering — smena qayerdan ochilgani yoziladi."
        LocationProblem.DeniedForever -> "Joylashuvga ruxsat berilmagan. Sozlamalar ochildi — «Ruxsatlar» → «Joylashuv» → «Ruxsat berish»."
        LocationProblem.Approximate -> "«Aniq joylashuv» kerak: taxminiy joylashuv ~2 km xatolik beradi. Qayta bosing va «Aniq»ni tanlang."
        LocationProblem.Off -> "Telefonda joylashuv (GPS) o'chiq. Yoqing va qaytadan bosing."
        LocationProblem.NoFix -> "Joylashuv aniqlanmadi. Deraza yoki eshikka yaqinroq turib qayta bosing."
    }
    Lang.Ru -> when (this) {
        LocationProblem.Denied -> "Доступ к геолокации не разрешён. Нажмите снова и разрешите — фиксируется, откуда открыта смена."
        LocationProblem.DeniedForever -> "Доступ к геолокации не разрешён. Открыты настройки — «Разрешения» → «Местоположение» → «Разрешить»."
        LocationProblem.Approximate -> "Нужна «точная геолокация»: приблизительная ошибается на ~2 км. Нажмите снова и выберите «Точно»."
        LocationProblem.Off -> "Геолокация (GPS) на телефоне выключена. Включите и нажмите снова."
        LocationProblem.NoFix -> "Не удалось определить местоположение. Подойдите ближе к окну или двери и нажмите снова."
    }
    Lang.En -> when (this) {
        LocationProblem.Denied -> "Location is not allowed. Press again and allow it — where a shift is opened is recorded."
        LocationProblem.DeniedForever -> "Location is not allowed. Settings are open — Permissions → Location → Allow."
        LocationProblem.Approximate -> "Precise location is needed: approximate is off by ~2 km. Press again and choose Precise."
        LocationProblem.Off -> "Location (GPS) is switched off on this phone. Turn it on and press again."
        LocationProblem.NoFix -> "Could not get a position. Move nearer a window or the door and press again."
    }
}
