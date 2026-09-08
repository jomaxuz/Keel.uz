package uz.keel.guest

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import uz.keel.design.DesignWords
import uz.keel.design.KeelBackground
import uz.keel.design.KeelWaiterTheme
import uz.keel.design.LocalLangHost
import uz.keel.design.LocalNotice
import uz.keel.design.LocalWords
import uz.keel.design.Note
import uz.keel.design.NoticeHost
import uz.keel.guest.ui.screens.MenuScreen

// The restaurant's own application.
//
// ⚠️ **There is no home page, and that is the product decision this app is built
// around.** Somebody opening a restaurant's app is hungry: they want the menu.
// A landing page with a cover photograph, opening hours and an "Order now"
// button is one tap and one scroll between a guest and the thing they came for,
// and every one of those is where an order is lost. Everything a landing page
// would have said — the address, the hours, the phone — belongs where somebody
// goes looking for it, not in front of the menu.
class MainActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        // ⚠️ **Before `super.onCreate`.** The activity wears the splash theme so
        // the launcher has the restaurant's logo to show instantly; this hands
        // over to the app's own. Called later, the first frame carries the wrong
        // window background.
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        val app = application as KeelGuestApp
        setContent {
            val notice = remember { mutableStateOf<Note?>(null) }
            CompositionLocalProvider(
                LocalPrefs provides app.prefs,
                LocalNotice provides notice,
                // ⚠️ Provided here or the shared controls throw: the design
                // module deliberately does not know this app, and asks for the
                // language and its own few words through the composition.
                LocalLangHost provides app.prefs.langHost(),
                LocalWords provides DesignWords(
                    ok = app.prefs.dict.common.ok,
                    retry = app.prefs.dict.common.retry,
                    loading = app.prefs.dict.common.loading,
                ),
            ) {
                // ⚠️ **The restaurant's colour, not Keel's.** This is the one
                // application that passes an accent — see the parameter's note
                // in the design module.
                KeelWaiterTheme(app.prefs.theme.value, accent = Brand.accent) {
                    KeelBackground {
                        Box(Modifier.fillMaxSize()) {
                            MenuScreen(app.api)
                            NoticeHost(notice)
                        }
                    }
                }
            }
        }
    }
}
