package uz.keel.guest

import android.app.Application
import uz.keel.design.TokenStore
import uz.keel.guest.data.KeelApi

/** The one place this application's long-lived objects are made.
 *
 *  ⚠️ **No server screen and no sign-in gate at startup**, unlike the five staff
 *  applications: the address is a constant of the build and the menu is public.
 *  What is left to do before the first frame is nothing, which is the point —
 *  a restaurant application that thinks before it draws the menu is one a guest
 *  closes. */
class KeelGuestApp : Application() {

    lateinit var prefs: Prefs
        private set

    lateinit var tokens: TokenStore
        private set

    lateinit var api: KeelApi
        private set

    override fun onCreate() {
        super.onCreate()
        prefs = Prefs(this)
        tokens = TokenStore(this)
        api = KeelApi(tokens).also { it.lang = prefs.lang.value.code }
    }
}
