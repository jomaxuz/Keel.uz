package uz.keel.waiter.push

import android.Manifest
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import com.google.firebase.messaging.FirebaseMessaging
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import kotlinx.coroutines.tasks.await
import uz.keel.waiter.KeelWaiterApp
import uz.keel.waiter.MainActivity
import uz.keel.waiter.R
import uz.keel.waiter.data.KeelApi

// Being told the food is ready.
//
// ⚠️ **The one thing a waiter cannot find out by looking.** Every other screen
// here is something they can open — the room, the check, their hours. Food
// reaching the pass happens in another part of the building, and the alternatives
// are a bell, a shout, or walking over to check. That walk is what this removes.
//
// ⚠️ **FCM directly, not Expo's relay — and the server does not know that yet.**
// The Expo build registered an `ExponentPushToken[...]`, and
// `internal/push/expo.go` posts every notification to `exp.host`. This app has an
// FCM registration token, which that endpoint will not accept. Until the server
// grows an FCM v1 sender (chosen by the token's *shape*, so the four apps still
// on Expo keep working), registration succeeds here and **nothing arrives**. The
// settings screen says "on" in that state, and that is the honest report of what
// this app knows: it registered.

/** What happened when this phone tried to subscribe.
 *
 *  ⚠️ **Every failure here used to be silent, and that was right for the person
 *  and wrong for everybody else.** An app that shows an error about notifications
 *  on the screen somebody is taking an order on is worse than the missing
 *  feature — but with nothing recorded anywhere, "the kitchen pressed ready and
 *  nothing arrived" has five possible causes and no way to tell them apart. So it
 *  is not shown; it is *available*, on the settings screen, where somebody goes
 *  when they are already asking the question. */
enum class PushState(val key: String) {
    Working("working"),
    Asking("asking"),
    Denied("denied"),
    NoDevice("noDevice"),
    NoProject("noProject"),
    Failed("failed"),
}

class PushRegistration(
    val state: PushState,
    val retry: () -> Unit,
    /** ⚠️ Awaited by the sign-out path **before** the token is cleared, or the
     *  request goes out unauthenticated and the row stays — sending the next
     *  evening's tables to whoever went home. */
    val forget: suspend () -> Unit,
)

@Composable
fun rememberPushRegistration(
    api: KeelApi,
    signedIn: Boolean,
    /** The language this phone reads. ⚠️ Sent with the token and re-sent when it
     *  changes: the notification is written on the server, so it is the one text
     *  in this app the device cannot translate for itself. */
    lang: String,
): PushRegistration {
    val ctx = LocalContext.current
    var state by remember { mutableStateOf(PushState.Asking) }
    var nonce by remember { mutableIntStateOf(0) }
    val token = remember { arrayOfNulls<String>(1) }

    // ⚠️ **Permission is asked after signing in, not at launch.** A prompt on the
    // first screen is asked before anybody knows what the app is for, and the
    // answer to a question you do not understand is "no".
    var permissionAsked by remember { mutableStateOf(false) }
    val ask = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        // ⚠️ **Refusal is a real answer and the app keeps working.** Somebody who
        // says no still has the room, the check and their hours; nagging them
        // every launch is the reason an app gets uninstalled.
        if (granted) nonce += 1 else state = PushState.Denied
    }

    LaunchedEffect(signedIn, lang, nonce) {
        if (!signedIn) return@LaunchedEffect
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            val granted = ContextCompat.checkSelfPermission(ctx, Manifest.permission.POST_NOTIFICATIONS) ==
                PackageManager.PERMISSION_GRANTED
            if (!granted) {
                if (!permissionAsked) {
                    permissionAsked = true
                    ask.launch(Manifest.permission.POST_NOTIFICATIONS)
                } else {
                    state = PushState.Denied
                }
                return@LaunchedEffect
            }
        }
        state = PushState.Asking
        try {
            val value = FirebaseMessaging.getInstance().token.await()
            token[0] = value
            // Registered on every launch: the token can be re-issued after a
            // reinstall, and the server keys on it so a phone handed to somebody
            // else moves to them.
            api.registerPush(value, lang)
            state = PushState.Working
        } catch (e: Throwable) {
            // ⚠️ Silently, on the screen. Every reason this fails — no Play
            // services, a refused permission, no network — leaves an app that
            // works, and an error banner about notifications on a screen somebody
            // is taking an order on is worse than the missing feature.
            state = PushState.Failed
        }
    }

    return PushRegistration(
        state = state,
        retry = { nonce += 1 },
        forget = {
            val value = token[0] ?: return@PushRegistration
            token[0] = null
            runCatching { api.forgetPush(value) }
            // The server prunes a dead token on the next event it fails to
            // deliver, so a failure here signs out either way.
            runCatching { FirebaseMessaging.getInstance().deleteToken().await() }
        },
    )
}

/** What arrives while the app is running.
 *
 *  ⚠️ **Shown even while the app is open.** A waiter with the room on screen is
 *  the person most likely to be walking, and the whole message is "go to the
 *  pass" — swallowing it because the app happens to be foregrounded is the one
 *  case where it is least likely to be read some other way.
 *
 *  ⚠️ **A data message, not a notification payload.** Firebase draws a
 *  `notification` block itself and only when the app is backgrounded — which is
 *  exactly the half this needs to control. The server sends the text in `data`,
 *  and this builds it. */
class KitchenMessagingService : FirebaseMessagingService() {

    override fun onNewToken(token: String) {
        // Re-registered by the app on its next launch; nothing to send here,
        // because a service has no signed-in session of its own.
    }

    override fun onMessageReceived(message: RemoteMessage) {
        val data = message.data
        val title = data["title"] ?: message.notification?.title ?: return
        val body = data["body"] ?: message.notification?.body ?: ""
        val checkId = data["checkId"]

        // A tap opens the table rather than the room: somebody reading this on
        // the move has already decided where they are going.
        val intent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP
            if (checkId != null) putExtra(MainActivity.EXTRA_CHECK_ID, checkId)
        }
        val pending = PendingIntent.getActivity(
            this, checkId?.hashCode() ?: 0, intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )

        val note = NotificationCompat.Builder(this, KeelWaiterApp.KITCHEN_CHANNEL)
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

        // ⚠️ Keyed on the check, so two messages about one table replace each
        // other rather than stacking — a shade with six copies of the same table
        // is a shade nobody reads.
        getSystemService(NotificationManager::class.java)
            .notify(checkId?.hashCode() ?: title.hashCode(), note)
    }
}
