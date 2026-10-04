package uz.keel.courier

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
import uz.keel.app.RoleExit
import uz.keel.app.push.KeelPush
import uz.keel.courier.push.PushRegistration
import uz.keel.courier.push.forRole
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
import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.compose.viewModel
import kotlinx.coroutines.launch
import uz.keel.design.DesignWords
import uz.keel.design.GlassTabBar
import uz.keel.design.KeelTheme
import uz.keel.design.LocalLangHost
import uz.keel.design.LocalNotice
import uz.keel.design.LocalWords
import uz.keel.design.Note
import uz.keel.design.NoticeKind
import uz.keel.courier.data.ApiError
import uz.keel.design.TabItem
import uz.keel.courier.location.LocationService
import uz.keel.courier.ui.screens.EarningsScreen
import uz.keel.courier.ui.screens.OfflineScreen
import uz.keel.courier.ui.screens.OrdersScreen
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


/** The courier's half of Keel.
 *
 *  ⚠️ **Everything Keel Courier's `MainActivity` did except the getting in.**
 *  The address and the password are asked once, by Keel, for every role; what
 *  is left is this role's own words and its own screens. The words are
 *  provided here and not by the activity, because the activity serves four
 *  dictionaries and only this composable knows which one it is in.
 *
 *  @param pendingTab a tab from a tapped notification. */
@Composable
fun CourierEntry(
    graph: CourierGraph,
    push: KeelPush,
    exit: RoleExit,
    pendingTab: String?,
    onConsumed: () -> Unit,
) {
    CompositionLocalProvider(
        LocalPrefs provides graph.prefs,
        LocalLangHost provides graph.prefs.langHost(),
        LocalWords provides DesignWords(
            ok = graph.prefs.dict.common.ok,
            retry = graph.prefs.dict.common.retry,
            loading = graph.prefs.dict.common.loading,
        ),
    ) {
        Root(graph, push.forRole(), exit, pendingTab, onConsumed)
    }
}

@Composable
private fun Root(
    app: CourierGraph,
    push: PushRegistration,
    exit: RoleExit,
    pendingTab: String?,
    onConsumed: () -> Unit,
) {
    val c = KeelTheme.colors
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    val notice = LocalNotice.current
    val statusFailed = t.orders.failed

    // ⚠️ **Keyed on the token** — the activity outlives this screen in Keel,
    // and an unkeyed model would hand a new sign-in the last one's session.
    val vm: SessionViewModel = viewModel(
        key = "courier:" + app.tokens.read(uz.keel.design.TokenStore.COURIER_TOKEN).hashCode(),
        factory = object : ViewModelProvider.Factory {
            @Suppress("UNCHECKED_CAST")
            override fun <T : ViewModel> create(modelClass: Class<T>): T =
                SessionViewModel(app.api, app.tokens) as T
        },
    )
    val session by vm.state.collectAsState()
    val ready = session as? Session.Ready

    var tab by remember { mutableStateOf("orders") }



    // ⚠️ **A refused token leaves the role; it does not draw a login here.**
    // Keel has one sign-in for every role, and what follows it depends on what
    // else is signed in on this phone.
    val refused = session is Session.SignedOut || session is Session.NoServer
    LaunchedEffect(refused) { if (refused) exit.expired() }

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

                is Session.NoServer -> Box(Modifier.fillMaxSize())

                // ⚠️ Before the login screen, not an error on it.
                is Session.Offline -> OfflineScreen(s.address) { vm.probe() }

                is Session.SignedOut -> Box(Modifier.fillMaxSize())

                is Session.Ready -> when (tab) {
                    "orders" -> OrdersScreen(
                        api = app.api,
                        name = s.courier.name,
                        status = s.courier.status,
                        bottomInset = tabsInset,
                        onStatus = { next ->
                            // ⚠️ Said, not swallowed: a refused switch left the
                            // card on the old status with nothing to explain why,
                            // and the courier pressed it again, and again.
                            scope.launch {
                                runCatching { vm.setStatus(next) }.onFailure { e ->
                                    notice.value = Note(
                                        NoticeKind.Error,
                                        statusFailed,
                                        (e as? ApiError)?.message,
                                    )
                                }
                            }
                        },
                        onRefreshCourier = vm::refresh,
                    )

                    "earnings" -> EarningsScreen(app.api, tabsInset)

                    else -> SettingsScreen(
                        courier = s.courier,
                        address = s.address,
                        bottomInset = tabsInset,
                        pushState = push.state,
                        pushDetail = push.detail,
                        onRetryPush = push.retry,
                        onSignOut = { leave { exit.signOut() } },
                        onForgetServer = { leave { exit.leaveRestaurant() } },
                        top = exit.card,
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
