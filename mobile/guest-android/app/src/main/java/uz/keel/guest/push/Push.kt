package uz.keel.guest.push

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import androidx.core.content.ContextCompat
import com.google.firebase.FirebaseApp
import com.google.firebase.FirebaseOptions
import com.google.firebase.messaging.FirebaseMessaging
import kotlinx.coroutines.tasks.await
import uz.keel.design.fetchPushToken
import uz.keel.guest.Brand
import uz.keel.guest.data.KeelApi

// ---- Telling a guest what happened to their dinner ----
//
// ⚠️ **No `google-services.json` and no Gradle plugin.** The plugin reads that
// file at build time and refuses any build whose `applicationId` has no matching
// client entry — which for a per-restaurant build means one generated file per
// restaurant, checked into nothing, produced by the pipeline, and a build
// failure whenever the two drift. Firebase can be configured in code instead,
// from four strings, and four strings live in `brand.properties` beside
// everything else that makes this app one restaurant's.
//
// ⚠️ **Empty values are "no push", not a crash.** The same rule the maps key
// follows: a restaurant whose app was built before we automated the Firebase
// side still has a working app, and the one thing it does not do is the one
// thing nobody has set up.
//
// ⚠️ **Registered only once somebody signs in.** The token is stored against a
// guest (`/users/me/device`), because the message is about *their* order. A
// token collected before there is anybody to attach it to is a row nothing can
// ever address.

/** The notification channel Android needs to have been told about.
 *
 *  ⚠️ **Created before the first message, and its id matches the server's.** A
 *  channel the phone never created arrives silent and unranked — which looks
 *  exactly like a notification nobody sent, and is diagnosed as a broken server.
 *  The server writes `ChannelID: "orders"` (handlers/userpush.go). */
private const val CHANNEL = "orders"

/** Bring Firebase up from the build's own four strings.
 *
 *  ⚠️ **Returns false rather than throwing.** An app that crashed on launch
 *  because a restaurant has no Firebase entry yet would be an app whose menu
 *  nobody can read — over a feature that is not the reason anybody installed it.
 */
fun initFirebase(context: Context): Boolean {
    if (Brand.firebaseAppId.isEmpty() || Brand.firebaseProjectId.isEmpty()) return false
    if (FirebaseApp.getApps(context).isNotEmpty()) return true
    return runCatching {
        FirebaseApp.initializeApp(
            context,
            FirebaseOptions.Builder()
                .setApplicationId(Brand.firebaseAppId)
                .setProjectId(Brand.firebaseProjectId)
                .setApiKey(Brand.firebaseApiKey)
                .setGcmSenderId(Brand.firebaseSenderId)
                .build(),
        )
        createChannel(context)
        true
    }.getOrDefault(false)
}

private fun createChannel(context: Context) {
    if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
    val manager = context.getSystemService(NotificationManager::class.java) ?: return
    manager.createNotificationChannel(
        NotificationChannel(
            CHANNEL,
            // ⚠️ The channel's name is what the guest sees in the phone's own
            // settings, so it says what the messages are about rather than
            // naming a system: "Buyurtma" is a thing they recognise, "orders"
            // is a word from our database.
            "Buyurtma",
            NotificationManager.IMPORTANCE_HIGH,
        ),
    )
}

/** What this phone is doing about notifications. */
data class PushState(val on: Boolean, val ask: () -> Unit)

/** Register this phone against the signed-in guest, asking for permission the
 *  first time.
 *
 *  ⚠️ **Asked after a sign-in, never at launch.** Android 13 asks the guest a
 *  yes/no question with no context attached; asked before they have ordered
 *  anything, most people say no once and permanently, and the refusal takes the
 *  order updates with it. Asked when they have just signed in to follow an
 *  order, the question answers itself. */
@Composable
fun rememberPush(api: KeelApi, signedIn: Boolean): PushState {
    val context = LocalContext.current
    var granted by remember {
        mutableStateOf(
            Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU ||
                ContextCompat.checkSelfPermission(
                    context,
                    Manifest.permission.POST_NOTIFICATIONS,
                ) == PackageManager.PERMISSION_GRANTED,
        )
    }
    val launcher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { granted = it }

    LaunchedEffect(signedIn, granted) {
        if (!signedIn || !granted) return@LaunchedEffect
        if (!initFirebase(context)) return@LaunchedEffect
        // ⚠️ Failure is swallowed. A phone with no Play services, or one that
        // simply could not reach Firebase this minute, still has a working app
        // — and the tracking screen shows the same facts.
        runCatching {
            val token = fetchPushToken { FirebaseMessaging.getInstance().token.await() }
            api.registerDevice(token, api.lang)
        }
    }

    return PushState(
        on = granted,
        ask = {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU && !granted) {
                launcher.launch(Manifest.permission.POST_NOTIFICATIONS)
            }
        },
    )
}

/** Drop this phone's token, before the session is cleared.
 *
 *  ⚠️ **This order, not the other one.** Clearing the token first sends the
 *  request unauthenticated, the row stays behind, and the next person to hold
 *  this phone is told about somebody else's dinner. */
suspend fun forgetPush(context: Context, api: KeelApi) {
    if (!initFirebase(context)) return
    runCatching {
        val token = FirebaseMessaging.getInstance().token.await()
        api.forgetDevice(token)
    }
    // Deleted at the Firebase end too, so a phone that is signed out stops
    // costing a delivery attempt per order.
    runCatching { FirebaseMessaging.getInstance().deleteToken().await() }
}
