package uz.keel.team

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.AccessTime
import androidx.compose.material.icons.rounded.ContentPaste
import androidx.compose.material.icons.rounded.Settings
import androidx.compose.material.icons.rounded.ShoppingBasket
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.compose.viewModel
import kotlinx.coroutines.launch
import uz.keel.design.DesignWords
import uz.keel.design.GlassTabBar
import uz.keel.design.KeelBackground
import uz.keel.design.KeelTheme
import uz.keel.design.KeelWaiterTheme
import uz.keel.design.LocalLangHost
import uz.keel.design.LocalNotice
import uz.keel.design.LocalWords
import uz.keel.design.Note
import uz.keel.design.NoticeHost
import uz.keel.design.TabItem
import uz.keel.team.push.rememberPushRegistration
import uz.keel.team.ui.screens.BuyScreen
import uz.keel.team.ui.screens.LoginScreen
import uz.keel.team.ui.screens.OfflineScreen
import uz.keel.team.ui.screens.ProfileScreen
import uz.keel.team.ui.screens.ServerScreen
import uz.keel.team.ui.screens.SettingsScreen
import uz.keel.team.ui.screens.ZakupScreen
import uz.keel.team.ui.screens.canWriteHere

// Keel Team — the app everybody in the restaurant has.
//
// ⚠️ **The smallest of the four on purpose.** A waiter has the floor, a courier
// has the road; a cook, a barman, a dishwasher and a cleaner have one thing the
// system needs from them and one thing they need from it: the shift starts, the
// shift ends, and this is what I have worked. Everything else on their phone
// would be somebody else's screen.
//
// ⚠️ **It exists because attendance was the one thing with no phone at all.**
// The clock-in lived on a web page (`/staff`), so an employee had to be told a
// URL, keep it in a browser tab and find it again every morning — and the till
// now refuses a PIN without an open shift, which turns "I could not find the
// page" into "I cannot start work".

class MainActivity : ComponentActivity() {

    private var pendingTab by mutableStateOf<String?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        // ⚠️ **Before `super.onCreate`.** The activity wears the splash theme so
        // the launcher has something branded to show instantly; this hands over
        // to the app's own. Called later, the first frame carries the wrong
        // window background.
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        pendingTab = intent?.getStringExtra(EXTRA_TAB)

        val app = application as KeelTeamApp
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
                KeelWaiterTheme(app.prefs.theme.value) {
                    KeelBackground {
                        Root(app, pendingTab) { pendingTab = null }
                        NoticeHost(notice)
                    }
                }
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        pendingTab = intent.getStringExtra(EXTRA_TAB)
    }

    companion object {
        const val EXTRA_TAB = "tab"
    }
}

@Composable
private fun Root(app: KeelTeamApp, pendingTab: String?, onConsumed: () -> Unit) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()

    val vm: SessionViewModel = viewModel(
        factory = object : ViewModelProvider.Factory {
            @Suppress("UNCHECKED_CAST")
            override fun <T : ViewModel> create(modelClass: Class<T>): T =
                SessionViewModel(app.api, app.tokens) as T
        },
    )
    val session by vm.state.collectAsState()
    val ready = session as? Session.Ready

    var tab by remember { mutableStateOf("profile") }

    val push = rememberPushRegistration(app.api, ready != null, app.prefs.lang.value.code)

    // Every one of these messages is about hours or pay, and this app has one
    // screen for both.
    LaunchedEffect(pendingTab, ready) {
        if (pendingTab != null && ready != null) {
            tab = pendingTab
            onConsumed()
        }
    }

    /** ⚠️ **The phone is dropped before the token is cleared.** The other order
     *  sends the request unauthenticated, the row stays, and somebody else's pay
     *  slip goes to a phone that has changed hands. */
    fun leave(after: () -> Unit) {
        scope.launch {
            push.forget()
            after()
        }
    }

    val bottomInset = WindowInsets.navigationBars.asPaddingValues()
    // ⚠️ The bar's own height *plus* the system's strip: the bar applies the
    // inset to itself, so a list padded only by the system's still ends with its
    // last row behind the glass.
    val tabsInset = PaddingValues(bottom = 76.dp + bottomInset.calculateBottomPadding())

    // ⚠️ **A tab exists only for the account that may use it.** A buyer is one
    // job in a restaurant, and a cook opening this app should not be shown a
    // screen that would refuse them — a button that says no teaches a room to
    // stop reading the app.
    val canBuy = ready?.staff?.perms?.contains("buy") == true
    val canOrder = ready?.staff?.let { canWriteHere(it) } == true

    // ⚠️ A permission taken away while somebody was standing on that tab leaves
    // them on a screen the server will refuse. Sent back to the one screen every
    // account has.
    LaunchedEffect(canBuy, canOrder) {
        if (tab == "buy" && !canBuy) tab = "profile"
        if (tab == "zakup" && !canOrder) tab = "profile"
    }

    Box(Modifier.fillMaxSize()) {
        AnimatedContent(
            targetState = Triple(session::class, tab, ready?.onShift),
            transitionSpec = { fadeIn() togetherWith fadeOut() },
            label = "root",
        ) { _ ->
            when (val s = session) {
                is Session.Loading -> Box(Modifier.fillMaxSize(), Alignment.Center) {
                    CircularProgressIndicator(color = c.accent)
                }

                is Session.NoServer -> ServerScreen(onChosen = vm::useServer)

                // ⚠️ Before the login screen, not an error on it.
                is Session.Offline -> OfflineScreen(s.address) { vm.probe() }

                is Session.SignedOut -> LoginScreen(
                    address = s.address,
                    onSignIn = { u, p -> vm.signIn(s.address, u, p) },
                    onForget = vm::forgetServer,
                )

                is Session.Ready -> when (tab) {
                    "buy" -> BuyScreen(app.api, tabsInset)
                    "zakup" -> ZakupScreen(app.api, tabsInset)
                    "settings" -> SettingsScreen(
                        staff = s.staff,
                        address = s.address,
                        bottomInset = tabsInset,
                        pushState = push.state,
                        pushDetail = push.detail,
                        onRetryPush = push.retry,
                        onSignOut = { leave { vm.signOut(s.address) } },
                        onForgetServer = { leave { vm.forgetServer() } },
                    )

                    else -> ProfileScreen(
                        api = app.api,
                        staff = s.staff,
                        onShift = s.onShift,
                        bottomInset = tabsInset,
                        onShiftChanged = vm::setOnShift,
                    )
                }
            }
        }

        if (ready != null) {
            GlassTabBar(
                items = buildList {
                    add(TabItem("profile", Icons.Rounded.AccessTime, t.tabs.profile))
                    if (canBuy) add(TabItem("buy", Icons.Rounded.ShoppingBasket, t.tabs.buy))
                    if (canOrder) add(TabItem("zakup", Icons.Rounded.ContentPaste, t.tabs.zakup))
                    add(TabItem("settings", Icons.Rounded.Settings, t.tabs.settings))
                },
                selected = tab,
                onSelect = { tab = it },
                modifier = Modifier.align(Alignment.BottomCenter),
            )
        }
    }
}
