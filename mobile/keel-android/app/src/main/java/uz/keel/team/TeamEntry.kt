package uz.keel.team

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
import androidx.compose.material.icons.automirrored.rounded.FactCheck
import androidx.compose.material.icons.rounded.Inventory2
import androidx.compose.material.icons.rounded.QrCodeScanner
import androidx.compose.material.icons.rounded.Settings
import androidx.compose.material.icons.rounded.ShoppingBasket
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.runtime.Composable
import uz.keel.app.RoleExit
import uz.keel.app.push.KeelPush
import uz.keel.team.push.PushRegistration
import uz.keel.team.push.forRole
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
import uz.keel.design.TabItem
import uz.keel.team.ui.screens.BuyScreen
import uz.keel.team.ui.screens.OfflineScreen
import uz.keel.team.ui.screens.ProfileScreen
import uz.keel.team.ui.screens.SanoqScreen
import uz.keel.team.ui.screens.SettingsScreen
import uz.keel.team.ui.screens.ZakupScreen
import uz.keel.team.ui.screens.SkladScreen
import uz.keel.team.ui.screens.MarkScreen
import uz.keel.team.ui.screens.canCountHere
import uz.keel.team.ui.screens.canLabelHere
import uz.keel.team.ui.screens.canScanHere
import uz.keel.team.ui.screens.canIssueHere
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


/** The team's half of Keel.
 *
 *  ⚠️ **Everything Keel Team's `MainActivity` did except the getting in.**
 *  The address and the password are asked once, by Keel, for every role; what
 *  is left is this role's own words and its own screens. The words are
 *  provided here and not by the activity, because the activity serves four
 *  dictionaries and only this composable knows which one it is in.
 *
 *  @param pendingTab a tab from a tapped notification. */
@Composable
fun TeamEntry(
    graph: TeamGraph,
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
    app: TeamGraph,
    push: PushRegistration,
    exit: RoleExit,
    pendingTab: String?,
    onConsumed: () -> Unit,
) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()

    // ⚠️ **Keyed on the token** — the activity outlives this screen in Keel,
    // and an unkeyed model would hand a new sign-in the last one's session.
    val vm: SessionViewModel = viewModel(
        key = "team:" + app.tokens.read(uz.keel.design.TokenStore.STAFF_TOKEN).hashCode(),
        factory = object : ViewModelProvider.Factory {
            @Suppress("UNCHECKED_CAST")
            override fun <T : ViewModel> create(modelClass: Class<T>): T =
                SessionViewModel(app.api, app.tokens) as T
        },
    )
    val session by vm.state.collectAsState()
    val ready = session as? Session.Ready

    var tab by remember { mutableStateOf("profile") }



    // ⚠️ **A refused token leaves the role; it does not draw a login here.**
    // Keel has one sign-in for every role, and what follows it depends on what
    // else is signed in on this phone.
    val refused = session is Session.SignedOut || session is Session.NoServer
    LaunchedEffect(refused) { if (refused) exit.expired() }

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
    // ⚠️ Its own permission rather than a corner of `stock`: counting a shelf
    // and emptying it are different acts, and folded together every person given
    // a phone to count the fridge would also hold the button that sends a case
    // of vodka across town.
    val canIssue = ready?.staff?.let { canIssueHere(it) } == true
    // ⚠️ `stock`, which is the technologist's permission and the panel's own
    // door for them (handlers/stocklogin.go). Counting writes the baseline every
    // later shortfall is measured from, so it is not a screen a cook opens by
    // accident — and asked in permissions rather than in the typed job title,
    // because a title grants nothing (models/staffrole.go).
    val canCount = ready?.staff?.let { canCountHere(it) } == true
    // ⚠️ **One tab for two permissions, and it appears for either.** Scanning
    // what the state issued and printing what the shop owns are different acts
    // with different keys — but they are the same minute of somebody's morning,
    // stood over an open box, and two tabs for one box is a phone nobody reads.
    // The screen itself draws only the half the account holds.
    val canMark = ready?.staff?.let { canScanHere(it) || canLabelHere(it) } == true

    // ⚠️ A permission taken away while somebody was standing on that tab leaves
    // them on a screen the server will refuse. Sent back to the one screen every
    // account has.
    LaunchedEffect(canBuy, canOrder, canIssue, canCount, canMark) {
        if (tab == "buy" && !canBuy) tab = "profile"
        if (tab == "zakup" && !canOrder) tab = "profile"
        if (tab == "sklad" && !canIssue) tab = "profile"
        if (tab == "sanoq" && !canCount) tab = "profile"
        if (tab == "mark" && !canMark) tab = "profile"
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

                is Session.NoServer -> Box(Modifier.fillMaxSize())

                // ⚠️ Before the login screen, not an error on it.
                is Session.Offline -> OfflineScreen(s.address) { vm.probe() }

                is Session.SignedOut -> Box(Modifier.fillMaxSize())

                is Session.Ready -> when (tab) {
                    "buy" -> BuyScreen(app.api, tabsInset)
                    "zakup" -> ZakupScreen(app.api, tabsInset)
                    "sklad" -> SkladScreen(app.api, tabsInset)
                    // ⚠️ The draft comes from the application, not from here: a
                    // count is half an hour of walking a store, and this
                    // `AnimatedContent` throws the screen away every time
                    // somebody taps another tab. See CountDraft.
                    "sanoq" -> SanoqScreen(app.api, app.counting, tabsInset)
                    "mark" -> MarkScreen(app.api, s.staff, tabsInset)
                    "settings" -> SettingsScreen(
                        staff = s.staff,
                        address = s.address,
                        bottomInset = tabsInset,
                        pushState = push.state,
                        pushDetail = push.detail,
                        onRetryPush = push.retry,
                        onSignOut = { leave { exit.signOut() } },
                        onForgetServer = { leave { exit.leaveRestaurant() } },
                        top = exit.card,
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
                    if (canIssue) add(TabItem("sklad", Icons.Rounded.Inventory2, t.tabs.sklad))
                    // ⚠️ The auto-mirrored one: a clipboard with a tick reads
                    // right-to-left in Arabic, and the plain icon is deprecated
                    // for exactly that.
                    if (canCount) {
                        add(TabItem("sanoq", Icons.AutoMirrored.Rounded.FactCheck, t.tabs.sanoq))
                    }
                    if (canMark) add(TabItem("mark", Icons.Rounded.QrCodeScanner, t.tabs.mark))
                    add(TabItem("settings", Icons.Rounded.Settings, t.tabs.settings))
                },
                selected = tab,
                onSelect = { tab = it },
                modifier = Modifier.align(Alignment.BottomCenter),
            )
        }
    }
}
