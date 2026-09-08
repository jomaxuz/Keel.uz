package uz.keel.guest

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
import androidx.compose.material.icons.rounded.ReceiptLong
import androidx.compose.material.icons.rounded.RestaurantMenu
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.ShoppingBag
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
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
import kotlinx.coroutines.launch
import uz.keel.design.DesignWords
import uz.keel.design.KeelBackground
import uz.keel.design.KeelWaiterTheme
import uz.keel.design.LocalLangHost
import uz.keel.design.LocalNotice
import uz.keel.design.LocalWords
import uz.keel.design.GlassTabBar
import uz.keel.design.Note
import uz.keel.design.NoticeHost
import uz.keel.design.NoticeKind
import uz.keel.design.TabItem
import uz.keel.guest.data.Restaurant
import uz.keel.guest.push.forgetPush
import uz.keel.guest.push.rememberPush
import uz.keel.guest.ui.screens.AccountScreen
import uz.keel.guest.ui.screens.CartScreen
import uz.keel.guest.ui.screens.CheckoutScreen
import uz.keel.guest.ui.screens.MenuScreen
import uz.keel.guest.ui.screens.OrderScreen
import uz.keel.guest.ui.screens.OrdersTab

// The restaurant's own application.
//
// ⚠️ **There is no home page, and that is the product decision this app is built
// around.** Somebody opening a restaurant's app is hungry: they want the menu. A
// landing page with a cover photograph, opening hours and an "Order now" button
// is one tap and one scroll between a guest and the thing they came for, and
// every one of those is where an order is lost. Everything a landing page would
// have said — the address, the hours, the phone — belongs where somebody goes
// looking for it, not in front of the menu.
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
                    KeelBackground { Root(app, notice) }
                }
            }
        }
    }
}

/** Where in the app we are.
 *
 *  ⚠️ **A sealed state rather than a navigation library.** Four screens, one of
 *  which is a step of another; a graph, a back stack and route strings would be
 *  more machinery than the thing being navigated. The one place it would
 *  genuinely help — deep links from a push — arrives with the notifications, and
 *  it is one more branch here rather than a rewrite. */
private sealed interface Where {
    data object Tabs : Where
    data object Checkout : Where
    data class Tracking(val number: String) : Where
}

@Composable
private fun Root(app: KeelGuestApp, notice: androidx.compose.runtime.MutableState<Note?>) {
    var tab by remember { mutableStateOf("menu") }
    var where by remember { mutableStateOf<Where>(Where.Tabs) }
    var restaurant by remember { mutableStateOf<Restaurant?>(null) }
    val prefs = LocalPrefs.current
    val lang = prefs.lang.value.code
    val placedWord = t.order.placed

    // ⚠️ Read once here as well as inside the menu, because the checkout needs
    // the restaurant's own point to open the map on — and a map that opens on
    // the null island is a map somebody closes.
    LaunchedEffect(lang) {
        runCatching { restaurant = app.api.restaurant() }
    }

    // ⚠️ **Asked for after a sign-in, never at launch.** Android 13 puts a
    // yes/no question with no context in front of somebody who has not ordered
    // anything yet, and most people say no once and permanently — taking the
    // order updates with it. See push/Push.kt.
    var signedIn by remember { mutableStateOf(app.api.signedIn()) }
    val push = rememberPush(app.api, signedIn)
    val scope = rememberCoroutineScope()
    val context = LocalContext.current

    val bottomInset = WindowInsets.navigationBars.asPaddingValues()
    // ⚠️ The bar's own height *plus* the system's strip: the bar applies the
    // inset to itself, so a list padded only by the system's still ends with its
    // last row behind the glass.
    val tabsInset = PaddingValues(bottom = 76.dp + bottomInset.calculateBottomPadding())

    Box(Modifier.fillMaxSize()) {
        AnimatedContent(
            targetState = where to tab,
            transitionSpec = { fadeIn() togetherWith fadeOut() },
            label = "root",
        ) { (place, current) ->
            when (place) {
                is Where.Checkout -> CheckoutScreen(
                    api = app.api,
                    cart = app.cart,
                    restaurant = restaurant,
                    bottomInset = bottomInset,
                    onBack = { where = Where.Tabs },
                    onPlaced = { order ->
                        // ⚠️ **Straight to the bank when there is one.** A "pay
                        // now" button on the next screen is one more tap between
                        // a filled basket and money in the till, and the guest
                        // who does not take it becomes an unpaid order somebody
                        // has to chase — the site draws the same conclusion.
                        app.placed.remember(order.number)
                        notice.value = Note(NoticeKind.Ok, placedWord)
                        where = Where.Tracking(order.number)
                    },
                )

                is Where.Tracking -> OrderScreen(
                    api = app.api,
                    number = place.number,
                    bottomInset = bottomInset,
                    onBack = { where = Where.Tabs },
                )

                Where.Tabs -> when (current) {
                    "cart" -> CartScreen(app.cart, lang, tabsInset) { where = Where.Checkout }
                    "orders" -> OrdersTab(app.placed, tabsInset) { where = Where.Tracking(it) }
                    "account" -> AccountScreen(
                        api = app.api,
                        bottomInset = tabsInset,
                        onSignedIn = {
                            signedIn = true
                            push.ask()
                        },
                        onSignedOut = {
                            scope.launch { forgetPush(context, app.api) }
                            app.favorites.clear()
                            signedIn = false
                        },
                    )
                    else -> MenuScreen(app.api, app.cart, app.favorites, tabsInset)
                }
            }
        }

        // ⚠️ **The bar is hidden on the checkout and the tracking screen.** Both
        // are one job with one way out, and a tab bar under them is an invitation
        // to abandon a half-filled form by accident.
        if (where is Where.Tabs) {
            GlassTabBar(
                items = listOf(
                    TabItem("menu", Icons.Rounded.RestaurantMenu, t.tabs.menu),
                    TabItem(
                        "cart",
                        Icons.Rounded.ShoppingBag,
                        // ⚠️ The count is on the label rather than as a dot: a
                        // badge says "something is in there" and this says what,
                        // which is the number the guest is deciding on.
                        if (app.cart.count > 0) "${t.tabs.cart} (${app.cart.count})"
                        else t.tabs.cart,
                    ),
                    TabItem("orders", Icons.Rounded.ReceiptLong, t.tabs.orders),
                    TabItem("account", Icons.Rounded.Person, t.tabs.account),
                ),
                selected = tab,
                onSelect = { tab = it },
                modifier = Modifier.align(Alignment.BottomCenter),
            )
        }

        NoticeHost(notice)
    }
}
