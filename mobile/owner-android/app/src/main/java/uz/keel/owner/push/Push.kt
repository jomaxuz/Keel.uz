package uz.keel.owner.push

import android.Manifest
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import com.google.firebase.messaging.FirebaseMessaging
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import kotlinx.coroutines.tasks.await
import uz.keel.design.fetchPushToken
import uz.keel.design.pushErrorDetail
import uz.keel.owner.KeelOwnerApp
import uz.keel.owner.MainActivity
import uz.keel.owner.R
import uz.keel.owner.data.KeelApi

// Being told what an owner cannot find out by looking.
//
// ⚠️ **What arrives on this channel is money and who moved it** — a large
// discount, a void after the bill was printed, a shortfall in the drawer. It is
// its own channel for exactly that reason: an owner who muted the kitchen's
// chime must not have muted this with it.
//
// ⚠️ **FCM directly, not Expo's relay.** The four Expo builds still go through
// the relay and the server routes by the token's own shape — see
// `internal/push/send.go`. A native token the relay would silently not deliver
// to is the failure this whole arrangement avoids.

enum class PushState(val key: String) {
    Working("working"), Asking("asking"), Denied("denied"),
    NoDevice("noDevice"), NoProject("noProject"), Failed("failed"),
}

class PushRegistration(
    val state: PushState,
    /** The raw reason it failed, for the settings screen.
     *
     *  ⚠️ **"Failed" was not enough, and the product already learned that.** The
     *  first person to run the Expo build on a real phone read "not registered ·
     *  could not reach the internet or the server", pressed retry, and got the
     *  same line — which is true of at least three completely different faults:
     *  this build has no push credentials, the server is older than the
     *  endpoint, or the phone is offline. None is fixed from the phone, and
     *  whoever *can* fix it needs to know which one it is.
     *
     *  ⚠️ **Deliberately not translated.** It is a diagnostic, read by whoever
     *  is fixing the install, and a translated HTTP status is a status nobody
     *  can search for. */
    val detail: String?,
    val retry: () -> Unit,
    val forget: suspend () -> Unit,
)

@Composable
fun rememberPushRegistration(api: KeelApi, signedIn: Boolean, lang: String): PushRegistration {
    val ctx = LocalContext.current
    var state by remember { mutableStateOf(PushState.Asking) }
    var detail by remember { mutableStateOf<String?>(null) }
    var nonce by remember { mutableIntStateOf(0) }
    val token = remember { arrayOfNulls<String>(1) }
    var asked by remember { mutableStateOf(false) }

    // ⚠️ **Asked after signing in, not at launch.** A prompt on the first screen
    // is answered before anybody knows what the app is for, and the answer to a
    // question you do not understand is "no".
    val ask = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        if (granted) nonce += 1 else state = PushState.Denied
    }

    LaunchedEffect(signedIn, lang, nonce) {
        if (!signedIn) return@LaunchedEffect
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            val granted = ContextCompat.checkSelfPermission(ctx, Manifest.permission.POST_NOTIFICATIONS) ==
                PackageManager.PERMISSION_GRANTED
            if (!granted) {
                if (!asked) { asked = true; ask.launch(Manifest.permission.POST_NOTIFICATIONS) }
                else state = PushState.Denied
                return@LaunchedEffect
            }
        }
        state = PushState.Asking
        detail = null
        try {
            val value = fetchPushToken { FirebaseMessaging.getInstance().token.await() }
            token[0] = value
            api.registerPush(value, lang)
            state = PushState.Working
        } catch (e: Throwable) {
            // ⚠️ Recorded, never shown on the screen somebody is reading their
            // takings on. But available on the settings screen — without it,
            // "nothing ever arrives" has five possible causes and no way to tell
            // them apart from the phone it happened on.
            state = PushState.Failed
            detail = pushErrorDetail(e)
        }
    }

    return PushRegistration(
        state = state,
        detail = detail,
        retry = { nonce += 1 },
        forget = {
            val value = token[0] ?: return@PushRegistration
            token[0] = null
            runCatching { api.forgetPush(value) }
            runCatching { FirebaseMessaging.getInstance().deleteToken().await() }
        },
    )
}

class OwnerMessagingService : FirebaseMessagingService() {

    override fun onNewToken(token: String) {
        // Re-registered by the app on its next launch; a service has no
        // signed-in session of its own to send with.
    }

    override fun onMessageReceived(message: RemoteMessage) {
        val data = message.data
        val title = data["title"] ?: message.notification?.title ?: return
        val body = data["body"] ?: message.notification?.body ?: ""

        val intent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP
            data["tab"]?.let { putExtra(MainActivity.EXTRA_TAB, it) }
        }
        val pending = PendingIntent.getActivity(
            this, 0, intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )

        val note = NotificationCompat.Builder(this, KeelOwnerApp.OWNER_CHANNEL)
            .setSmallIcon(R.drawable.ic_notification)
            .setColor(ContextCompat.getColor(this, R.color.keel_orange))
            .setContentTitle(title)
            .setContentText(body)
            .setStyle(NotificationCompat.BigTextStyle().bigText(body))
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setAutoCancel(true)
            .setContentIntent(pending)
            .build()

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) !=
            PackageManager.PERMISSION_GRANTED
        ) return

        // ⚠️ **Keyed on the server's tag, not on one id.** Two different losses
        // are two facts; keying them together would let the second silently
        // erase the first, and an owner would never learn about it.
        val tag = data["tag"] ?: title
        getSystemService(NotificationManager::class.java).notify(tag.hashCode(), note)
    }
}
