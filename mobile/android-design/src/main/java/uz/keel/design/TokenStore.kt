package uz.keel.design

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

// Where this phone keeps a token, and which restaurant it belongs to.
//
// ⚠️ **The keystore, not plain preferences.** These tokens are a restaurant's
// floor and its takings: they can open a shift, add a dish and print a bill.
// `EncryptedSharedPreferences` puts them behind the Android keystore, which is
// where a credential belongs; an ordinary preferences file is readable by
// anything that reaches the sandbox — acceptable for a remembered filter, not
// for this.
//
// ⚠️ **Read once into memory at startup, written through behind.** Every call
// asks for a token synchronously, and a disk read on each would be a disk read
// per request. The Expo app carried exactly this asymmetry in exactly one file;
// so does this one.
//
// ⚠️ **Same key spellings as the browser's and the Expo build's.** A token is
// written by one platform and read by another in one case that matters — a QR
// handover — and a divergent name fails silently there.
class TokenStore(context: Context) {

    private val prefs: SharedPreferences = run {
        val key = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        EncryptedSharedPreferences.create(
            context,
            "keel_waiter_secure",
            key,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    }

    private val memory = HashMap<String, String>()

    init {
        // ⚠️ Everything, before the first render. `PrefsProvider`'s equivalent
        // reads the language and the theme in an initialiser — a hydration
        // started *below* it opened every launch in Uzbek on the light theme,
        // however many times somebody chose otherwise. The setting was being
        // saved correctly the whole time; it was being read too early.
        for (k in KEYS) prefs.getString(k, null)?.let { memory[k] = it }
    }

    fun read(key: String): String? = memory[key]

    fun write(key: String, value: String) {
        memory[key] = value
        // Not awaited: the caller is a sign-in about to navigate. The disk write
        // is what makes the *next* launch remember, not what makes this one work.
        prefs.edit().putString(key, value).apply()
    }

    fun drop(key: String) {
        // ⚠️ Memory first and unconditionally. Failing to delete leaves a token
        // on the device after a sign-out, which is the one direction that is not
        // safe — this session is signed out whatever the disk does.
        memory.remove(key)
        prefs.edit().remove(key).apply()
    }

    var staffToken: String?
        get() = read(STAFF_TOKEN)
        set(v) = if (v == null) drop(STAFF_TOKEN) else write(STAFF_TOKEN, v)

    var serverAddress: String?
        get() = read(SERVER_ADDRESS)
        set(v) = if (v == null) drop(SERVER_ADDRESS) else write(SERVER_ADDRESS, v)

    /** This install's own id.
     *
     *  ⚠️ **A reinstall mints a new one, and that is the accepted cost.** The
     *  panel has a button that releases a binding and it exists precisely for
     *  this — a lock nobody can lift turns "I reinstalled the app" into a phone
     *  call to us. Anything more stable is either unavailable to ordinary apps
     *  (Android stopped handing out serials years ago), unstable in its own way,
     *  or an identifier we have no business keeping. */
    fun deviceId(): String = read(DEVICE_ID) ?: mint().also { write(DEVICE_ID, it) }

    private fun mint(): String =
        "${System.currentTimeMillis().toString(36)}-${java.util.UUID.randomUUID()}"

    companion object {
        const val STAFF_TOKEN = "staff_token"
        /** The panel's own session — what the owner application signs in with.
         *
         *  ⚠️ **The browser's spelling.** A token is written by one platform and
         *  read by another in exactly one case that matters — a QR handover —
         *  and a divergent name fails silently there. */
        const val ADMIN_TOKEN = "admin_token"

        /** The television's own long-lived token.
         *
         *  ⚠️ **The browser's spelling again** (`lib/api.ts` → `TV_TOKEN_KEY`).
         *  Nothing hands this one between platforms today, but the two apps
         *  that already diverged on a key name both did it while nobody
         *  expected them to meet. */
        const val TV_TOKEN = "tv_token"
        const val SERVER_ADDRESS = "keel_server_address"
        const val DEVICE_ID = "keel_device_id"
        const val LANG = "keel_lang"
        const val THEME = "keel_theme"
        const val MENU_VIEW = "keel_menu_view"

        /** ⚠️ **Named rather than discovered.** Hydration happens once, before
         *  the first request, so a key nobody listed reads as absent on launch
         *  and sends somebody to a login they had already passed — and only on a
         *  cold start, which is the hardest kind of bug to be shown. */
        private val KEYS = listOf(
            STAFF_TOKEN, ADMIN_TOKEN, TV_TOKEN, SERVER_ADDRESS, DEVICE_ID, LANG, THEME, MENU_VIEW,
            // The owner application's branch lens. ⚠️ Listed here like every
            // other key: hydration happens once, before the first render, and a
            // key nobody named reads as absent on a cold start — which would
            // silently widen an owner's view back to "every branch".
            "keel_owner_branch",
            // What the television last heard the restaurant's clock say. ⚠️ Read
            // here, before the first frame, because the first decision that app
            // makes is whether a dated slide may play — and a set that came back
            // from a power cut may believe it is 1970. See tv-android/Clock.kt.
            "keel_tv_clock",
        )
    }
}
