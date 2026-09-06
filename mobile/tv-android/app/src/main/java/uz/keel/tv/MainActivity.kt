package uz.keel.tv

import android.os.Bundle
import android.view.WindowManager
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.compose.viewModel
import kotlinx.coroutines.delay
import uz.keel.design.DesignWords
import uz.keel.design.KeelBackground
import uz.keel.design.KeelWaiterTheme
import uz.keel.design.LocalLangHost
import uz.keel.design.LocalWords
import uz.keel.design.ThemeChoice
import uz.keel.tv.ui.screens.BoardScreen
import uz.keel.tv.ui.screens.BoardStrip
import uz.keel.tv.ui.screens.LoadingScreen
import uz.keel.tv.ui.screens.PairedScreen
import uz.keel.tv.ui.screens.PairingScreen
import uz.keel.tv.ui.screens.PlayerScreen
import uz.keel.tv.ui.screens.ServerScreen

// Keel TV — the screen on the restaurant's wall.
//
// ⚠️ **The fifth app, and the only one nobody holds.** Every other application in
// this repository is picked up by a person who can react to it: a waiter reads an
// error, a cashier presses a button again, a courier restarts something. This one
// hangs above head height in a room full of guests, and the only person who can
// act on anything it says is not in the room. That single fact decides the whole
// design:
//
//   - it never shows an error to the room — a dropped connection is silence, not
//     a red box in front of forty people eating;
//   - it needs no input at all after setup, because there is no keyboard and the
//     remote is in a drawer;
//   - it is paired by a code it *shows*, not by credentials it asks for;
//   - and being unpaired from the panel has to reach it, which is why it says
//     hello every minute rather than trusting a token issued a year ago.

class MainActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        // ⚠️ **Before `super.onCreate`.** The activity wears the splash theme so
        // the launcher has something branded to show instantly; this hands over
        // to the app's own. Called later, the first frame carries the wrong
        // window background — on a wall, in front of a room.
        //
        // ⚠️ **No `setKeepOnScreenCondition`.** The only thing worth waiting for
        // is the saved token, and that is read synchronously in
        // `KeelTvApp.onCreate`. The tempting thing to hold for is the heartbeat,
        // and on a dead connection that never answers — which on a television
        // nobody force-quits is a splash screen for the rest of the day.
        installSplashScreen()
        super.onCreate(savedInstanceState)

        // ⚠️ **A television must never sleep, and Android will put it to sleep.**
        // Without this the dining room's screen shows a screensaver an hour
        // after opening, and the restaurant reports the app as broken —
        // correctly, from where they are standing.
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)

        // ⚠️ **The system bars are hidden, not merely coloured.** A status bar
        // across a restaurant's own display is a strip of somebody else's
        // operating system, and on a set with no touchscreen there is nothing to
        // swipe it back with either.
        WindowCompat.setDecorFitsSystemWindows(window, false)
        WindowInsetsControllerCompat(window, window.decorView).apply {
            hide(WindowInsetsCompat.Type.systemBars())
            systemBarsBehavior =
                WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
        }

        val app = application as KeelTvApp
        setContent {
            CompositionLocalProvider(
                LocalPrefs provides app.prefs,
                // ⚠️ Provided here or the shared controls throw: the design
                // module deliberately does not know this app, and asks for the
                // language and its own few words through the composition.
                LocalLangHost provides app.prefs.langHost(),
                LocalWords provides DesignWords(
                    ok = "OK",
                    retry = app.prefs.dict.server.next,
                    loading = app.prefs.dict.idle.connecting,
                ),
            ) {
                // ⚠️ **Dark, always.** The rest of the product offers the choice;
                // a two-metre panel above a dining table does not, because the
                // light scheme there is a lamp nobody asked for and the person
                // who would switch it back is not in the room.
                KeelWaiterTheme(ThemeChoice.Dark) {
                    KeelBackground { Root(app) }
                }
            }
        }
    }
}

@Composable
private fun Root(app: KeelTvApp) {
    val vm: TvViewModel = viewModel(
        factory = object : ViewModelProvider.Factory {
            @Suppress("UNCHECKED_CAST")
            override fun <T : ViewModel> create(modelClass: Class<T>): T = TvViewModel(
                api = app.api,
                tokens = app.tokens,
                clock = app.clock,
                playlist = app.playlist,
                boardPoller = app.board,
                appVersion = KeelTvApp.APP_VERSION,
                installId = app.installId,
            ) as T
        },
    )

    val state by vm.state.collectAsState()
    val board by vm.board.collectAsState()
    val stored by vm.playlist.items.collectAsState()
    val downloading by vm.playlist.downloading.collectAsState()

    // ⚠️ **Re-evaluated on a timer, not only when the list changes.** A window
    // closes on its own — at midnight, with nobody watching and quite possibly
    // with no internet — and a screen that only re-checked when the panel changed
    // something would show an expired offer until somebody edited the playlist.
    val now = rememberMinuteTick(app.clock)
    val items = stored.filter { playableNow(it, now) }

    val screen = when (val s = state) {
        is TvState.Paired -> s.screen
        is TvState.Offline -> s.screen
        else -> null
    }

    // ⚠️ **A screen set to the order board does not play the loop.** The mode is
    // the wall's answer, not the playlist's: one television by the counter shows
    // numbers all day while the one in the dining room plays the menu, and both
    // read the same branch's playlist.
    val paired = state is TvState.Paired || state is TvState.Offline
    val playing = paired && screen?.mode != "board" && items.isNotEmpty()
    // ⚠️ **`split` with an empty playlist falls back to the whole board**, not to
    // a placeholder. Split means "the loop, with the numbers under it"; with no
    // loop there is still a counter, and the numbers are the half of that screen
    // somebody in the room is actually waiting on.
    val showBoard = paired &&
        (screen?.mode == "board" || (screen?.mode == "split" && !playing))

    Box(Modifier.fillMaxSize()) {
        AnimatedContent(
            targetState = Triple(state::class, playing, showBoard),
            transitionSpec = { fadeIn() togetherWith fadeOut() },
            label = "root",
        ) { _ ->
            when (val s = state) {
                is TvState.Loading -> LoadingScreen()
                is TvState.NoServer -> ServerScreen(onSubmit = vm::useServer)
                is TvState.Unreachable -> ServerScreen(
                    onSubmit = vm::useServer,
                    initial = s.address,
                    unreachable = true,
                    answered = s.answered,
                )

                is TvState.Pairing -> PairingScreen(s.code, s.expiresAt)

                else -> when {
                    showBoard -> BoardScreen(board, screen?.branchName)
                    playing -> PlayerScreen(items)
                    else -> PairedScreen(
                        screen = screen,
                        offline = s is TvState.Offline,
                        downloading = downloading && s !is TvState.Offline,
                        // ⚠️ `null` is "the first heartbeat has not landed",
                        // which is a freshly paired screen's first minute — not
                        // an empty playlist. The two used to be shown with the
                        // same sentence, which sent somebody to a laptop to fix
                        // a television that was four minutes from playing.
                        waiting = s is TvState.Paired && s.contentVersion == null,
                    )
                }
            }
        }

        // ⚠️ Over the video, not beside it: the strip is `split` mode, and it
        // draws only when the counter has something ready — an empty bar across
        // a restaurant's own promotional video is furniture.
        if (playing && screen?.mode == "split") {
            BoardStrip(board, Modifier.align(Alignment.BottomCenter))
        }
    }
}

/** The restaurant's time, refreshed once a minute.
 *
 *  ⚠️ A minute, because that is the resolution the only thing reading it needs:
 *  a slide's window opens and closes on a date. Anything faster is a
 *  recomposition of a wall for nothing. */
@Composable
private fun rememberMinuteTick(clock: Clock): Long {
    var now by remember { mutableLongStateOf(clock.now()) }
    LaunchedEffect(Unit) {
        while (true) {
            now = clock.now()
            delay(60_000)
        }
    }
    return now
}
