package uz.keel.owner

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Insights
import androidx.compose.material.icons.rounded.NotificationsActive
import androidx.compose.material.icons.rounded.ReceiptLong
import androidx.compose.material.icons.rounded.Settings
import androidx.compose.material.icons.rounded.StarOutline
import androidx.compose.material.icons.rounded.Today
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
import uz.keel.owner.push.rememberPushRegistration
import uz.keel.owner.ui.screens.AlertsScreen
import uz.keel.owner.ui.screens.FeedbackScreen
import uz.keel.owner.ui.screens.LoginScreen
import uz.keel.owner.ui.screens.OfflineScreen
import uz.keel.owner.ui.screens.OrdersScreen
import uz.keel.owner.ui.screens.ReportsScreen
import uz.keel.owner.ui.screens.ServerScreen
import uz.keel.owner.ui.screens.SettingsScreen
import uz.keel.owner.ui.screens.SupportScreen
import uz.keel.owner.ui.screens.TodayScreen

// Keel Owner.
//
// ⚠️ **This is not the panel on a phone.** The panel is where somebody sits down
// and decides: the menu, a price, a roster, a campaign. Those stay on a keyboard.
// An owner uses a phone to *watch and answer* — how is today going, what is being
// asked for right now, accept this order, why was 400 000 taken off table six.
// Every screen here answers one of those in three seconds.
//
// ⚠️ **One level of navigation, held here, rather than a router.** Help is the
// only place a tab leads to, and the phone's back button has nothing else to
// mean. A navigation library at this size is a dependency carrying one decision.

class MainActivity : ComponentActivity() {

    private var pendingTab by mutableStateOf<String?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        // ⚠️ **Before `super.onCreate`.** The activity wears the splash theme so
        // the launcher has something branded to show instantly; this hands over
        // to the app's own. Called later, the first frame carries the wrong
        // window background.
        //
        // ⚠️ **No `setKeepOnScreenCondition`.** The only thing it could wait for
        // is the saved token, and that is read synchronously in
        // `KeelOwnerApp.onCreate`. The tempting thing to hold for is the "who is
        // this?" request — and on a dead connection that never answers, which is
        // an app somebody force-quits.
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        pendingTab = intent?.getStringExtra(EXTRA_TAB)

        val app = application as KeelOwnerApp
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
                    ok = app.prefs.dict.notice.ok,
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
private fun Root(app: KeelOwnerApp, pendingTab: String?, onConsumed: () -> Unit) {
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

    var tab by remember { mutableStateOf("today") }
    var help by remember { mutableStateOf(false) }

    val push = rememberPushRegistration(app.api, ready != null, app.prefs.lang.value.code)

    // A notification tap opens the screen it is about: somebody reading "a
    // discount was given" wants the list, not the takings.
    LaunchedEffect(pendingTab, ready) {
        val to = pendingTab ?: return@LaunchedEffect
        if (ready == null) return@LaunchedEffect
        tab = to
        onConsumed()
    }

    // ⚠️ Only while help is open, so back from a tab still leaves the app — an
    // owner who cannot put their phone away is an owner fighting it.
    BackHandler(enabled = help) { help = false }

    val bottomInset = WindowInsets.navigationBars.asPaddingValues()
    // ⚠️ The bar's own height *plus* the system's strip: the bar applies the
    // inset to itself, so a list padded only by the system's still ends with its
    // last row behind the glass.
    val tabsInset = PaddingValues(bottom = 76.dp + bottomInset.calculateBottomPadding())

    Box(Modifier.fillMaxSize()) {
        AnimatedContent(
            targetState = Triple(session::class, tab, help),
            transitionSpec = { fadeIn() togetherWith fadeOut() },
            label = "root",
        ) { _ ->
            when (val s = session) {
                is Session.Loading -> Box(Modifier.fillMaxSize(), Alignment.Center) {
                    CircularProgressIndicator(color = c.accent)
                }

                is Session.NoServer -> ServerScreen(onChosen = vm::useServer)

                // ⚠️ Before the login screen, not instead of an error on it: a
                // launch with no network used to land on the password field,
                // where the right password fails and the app blames the person
                // for a network they cannot see.
                is Session.Offline -> OfflineScreen(s.address) { vm.probe() }

                is Session.SignedOut -> LoginScreen(
                    address = s.address,
                    onSignIn = { u, p -> vm.signIn(s.address, u, p) },
                    onForget = vm::forgetServer,
                )

                is Session.Ready -> {
                    if (help) {
                        SupportScreen(app.api, bottomInset) { help = false }
                    } else {
                        Column(Modifier.fillMaxSize()) {
                            Box(Modifier.weight(1f)) {
                                when (tab) {
                                    "today" -> TodayScreen(app.api, s.branches, tabsInset)
                                    "alerts" -> AlertsScreen(app.api, tabsInset)
                                    "orders" -> OrdersScreen(app.api, tabsInset)
                                    "feedback" -> FeedbackScreen(app.api, tabsInset)
                                    "reports" -> ReportsScreen(app.api, tabsInset)
                                    else -> SettingsScreen(
                                        api = app.api,
                                        user = s.user,
                                        address = s.address,
                                        branchName = s.branches
                                            .firstOrNull { it.id == app.prefs.branch.value }
                                            ?.name.orEmpty(),
                                        bottomInset = tabsInset,
                                        pushState = push.state,
                                        onRetryPush = push.retry,
                                        onOpenHelp = { help = true },
                                        // ⚠️ The phone is dropped **before** the
                                        // token is cleared, or the request goes
                                        // out unauthenticated and the row stays
                                        // — sending tomorrow's alerts to a phone
                                        // that has changed hands.
                                        onSignOut = {
                                            scope.launch { push.forget(); vm.signOut(s.address) }
                                        },
                                        onForgetServer = {
                                            scope.launch { push.forget(); vm.forgetServer() }
                                        },
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }

        // The bar belongs to the tabs, not to a screen you came *into*.
        if (ready != null && !help) {
            GlassTabBar(
                items = listOf(
                    TabItem("today", Icons.Rounded.Today, t.tabs.today),
                    TabItem("alerts", Icons.Rounded.NotificationsActive, t.tabs.alerts),
                    TabItem("orders", Icons.Rounded.ReceiptLong, t.tabs.orders),
                    TabItem("feedback", Icons.Rounded.StarOutline, t.tabs.feedback),
                    TabItem("reports", Icons.Rounded.Insights, t.tabs.reports),
                    TabItem("settings", Icons.Rounded.Settings, t.tabs.settings),
                ),
                selected = tab,
                onSelect = { tab = it },
                modifier = Modifier.align(Alignment.BottomCenter),
            )
        }
    }
}
