package uz.keel.app.push

import android.Manifest
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import uz.keel.app.KeelApp
import uz.keel.app.MainActivity
import uz.keel.app.R
import uz.keel.app.data.Workspace

/** What arrives while Keel is running — for all four roles.
 *
 *  ⚠️ **One service, because Firebase calls exactly one.** Each single-role app
 *  had its own, and they differed only in the channel and in where a tap led.
 *  Both are decided here from what the server sent: the `channel` key (copied
 *  into `data` by `internal/push/fcm.go` for this app's sake) and, for the
 *  floor, the check.
 *
 *  ⚠️ **Shown even while the app is open**, as every Keel app does: a waiter
 *  with the room on screen is the person most likely to be walking, and the
 *  owner with the floor open still needs the loss alert. And while the app is
 *  in the background the system draws the message itself, from the
 *  `notification` block — the tap then reaches `MainActivity` carrying the same
 *  data keys, which is why routing reads them and not this service's extras
 *  alone. */
class KeelMessagingService : FirebaseMessagingService() {

    override fun onNewToken(token: String) {
        // Re-registered by the app on its next launch: a service has no
        // signed-in session of its own to send with.
    }

    override fun onMessageReceived(message: RemoteMessage) {
        val data = message.data
        val title = data["title"] ?: message.notification?.title ?: return
        val body = data["body"] ?: message.notification?.body ?: ""
        val checkId = data["checkId"]
        // ⚠️ The check wins over the channel: a staff row's channel follows
        // the section it was registered from, and the kitchen's "ready" has to
        // open the table whichever that was.
        val workspace = if (checkId != null) Workspace.Waiter
        else Workspace.ofChannel(data["channel"] ?: message.notification?.channelId)
        val channel = when (workspace) {
            Workspace.Waiter -> KeelApp.KITCHEN_CHANNEL
            Workspace.Courier -> KeelApp.DELIVERY_CHANNEL
            Workspace.Owner -> KeelApp.OWNER_CHANNEL
            Workspace.Team, null -> KeelApp.TEAM_CHANNEL
        }

        val intent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP
            putExtra(MainActivity.EXTRA_CHANNEL, channel)
            checkId?.let { putExtra(MainActivity.EXTRA_CHECK_ID, it) }
            data["tab"]?.let { putExtra(MainActivity.EXTRA_TAB, it) }
        }
        // ⚠️ A request code per message: one shared code makes the system
        // reuse the last PendingIntent, and the owner's alert then opens the
        // courier's orders.
        val requestCode = (channel + (checkId ?: data["tag"] ?: title)).hashCode()
        val pending = PendingIntent.getActivity(
            this, requestCode, intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )

        val note = NotificationCompat.Builder(this, channel)
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

        // ⚠️ **The single-role apps' keys, each kept, prefixed by channel.**
        // The floor keys on check + tag (two dishes are two facts, one table's
        // repeats replace each other); the owner keys on the tag or the message
        // (every alert is its own); the others on the tag or the title. The
        // prefix stops a courier's "new order" replacing an owner's alert that
        // happened to share a title.
        val tag = data["tag"]
        val key = when (workspace) {
            Workspace.Waiter -> when {
                checkId == null -> title
                tag.isNullOrEmpty() -> checkId
                else -> "$checkId:$tag"
            }
            Workspace.Owner -> tag ?: message.messageId ?: "$title#${System.currentTimeMillis()}"
            else -> tag ?: title
        }
        getSystemService(NotificationManager::class.java).notify("$channel/$key".hashCode(), note)
    }
}
