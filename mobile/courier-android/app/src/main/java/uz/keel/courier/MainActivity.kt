package uz.keel.courier

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
import androidx.compose.material.icons.rounded.Payments
import androidx.compose.material.icons.rounded.ReceiptLong
import androidx.compose.material.icons.rounded.Settings
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
import androidx.compose.ui.platform.LocalContext
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
import uz.keel.courier.location.LocationService
import uz.keel.courier.push.rememberPushRegistration
import uz.keel.courier.ui.screens.EarningsScreen
import uz.keel.courier.ui.screens.LoginScreen
import uz.keel.courier.ui.screens.OfflineScreen
import uz.keel.courier.ui.screens.OrdersScreen
import uz.keel.courier.ui.screens.ServerScreen
import uz.keel.courier.ui.screens.SettingsScreen

// Keel Courier.
//
// ⚠️ **Five states, not a boolean.** "Which restaurant", "who is signed in" and
// "can the phone reach anything" are three questions with three answers, and the
// third one used to be answered with the second's screen — a courier in a
// basement met a password field, typed the right password, and blamed
// themselves.
//
// ⚠️ **One level of navigation, held here, rather than a router.** Three tabs
// and a sheet; a navigation library at this size is a dependency carrying one
// decision.

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

        val app = application as KeelCourierApp
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
private fun Root(app: KeelCourierApp, pendingTab: String?, onConsumed: () -> Unit) {
    val c = KeelTheme.colors
    val ctx = LocalContext.current
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

    var tab by remember { mutableStateOf("orders") }

    val push = rememberPushRegistration(app.api, ready != null, app.prefs.lang.value.code)

    // Every one of these messages is about an order, so a tap lands on the list.
    LaunchedEffect(pendingTab, ready) {
        if (pendingTab != null && ready != null) {
            tab = pendingTab
            onConsumed()
        }
    }

    // ⚠️ **The shift's service follows the shift, not the screen.** Started when
    // the status leaves "off" and stopped the moment it returns — a service that
    // outlived the shift would be a battery complaint and, worse, a phone still
    // telling a restaurant where somebody is after they have gone home.
    val onShift = ready != null && ready.courier.status != "off"
    LaunchedEffect(onShift) {
        if (onShift) LocationService.start(ctx) else LocationService.stop(ctx)
    }

    val bottomInset = WindowInsets.navigationBars.asPaddingValues()
    // ⚠️ The bar's own height *plus* the system's strip: the bar applies the
    // inset to itself, so a list padded only by the system's still ends with its
    // last row behind the glass.
    val tabsInset = PaddingValues(bottom = 76.dp + bottomInset.calculateBottomPadding())

    /** ⚠️ **The phone is dropped before the token is cleared.** The other order
     *  sends the request unauthenticated, the row stays, and tomorrow's
     *  addresses — names, phone numbers, doors — go to whoever holds this
     *  handset next. The service is stopped for the same reason. */
    fun leave(after: () -> Unit) {
        scope.launch {
            push.forget()
            LocationService.stop(ctx)
            after()
        }
    }

    Box(Modifier.fillMaxSize()) {
        AnimatedContent(
            targetState = Triple(session::class, tab, ready?.courier?.status),
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
                    "orders" -> OrdersScreen(
                        api = app.api,
                        name = s.courier.name,
                        status = s.courier.status,
                        bottomInset = tabsInset,
                        onStatus = { next -> scope.launch { runCatching { vm.setStatus(next) } } },
                        onRefreshCourier = vm::refresh,
                    )

                    "earnings" -> EarningsScreen(app.api, tabsInset)

                    else -> SettingsScreen(
                        courier = s.courier,
                        address = s.address,
                        bottomInset = tabsInset,
                        pushState = push.state,
                        onRetryPush = push.retry,
                        onSignOut = { leave { vm.signOut(s.address) } },
                        onForgetServer = { leave { vm.forgetServer() } },
                    )
                }
            }
        }

        if (ready != null) {
            GlassTabBar(
                items = listOf(
                    TabItem("orders", Icons.Rounded.ReceiptLong, t.tabs.orders),
                    TabItem("earnings", Icons.Rounded.Payments, t.tabs.earnings),
                    TabItem("settings", Icons.Rounded.Settings, t.tabs.settings),
                ),
                selected = tab,
                onSelect = { tab = it },
                modifier = Modifier.align(Alignment.BottomCenter),
            )
        }
    }
}
