package uz.keel.tv

import androidx.compose.runtime.Immutable
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import uz.keel.design.ServerAddress
import uz.keel.design.TokenStore
import uz.keel.tv.data.ApiError
import uz.keel.tv.data.KeelApi
import uz.keel.tv.data.TVScreenSelf

// What this television is: which restaurant, and whether it has been paired.
//
// ⚠️ **Five states, and the ones that matter are the two that look identical
// from a `catch`.** "Not paired" and "cannot reach the server" mean opposite
// things: one is answered by somebody typing a code into the panel, the other by
// the wifi coming back. A television that showed a pairing code every time the
// internet blinked would be re-paired by a manager who then learns to distrust
// the screen entirely — and it hangs above head height, so distrust is the end
// of it.

@Immutable
sealed interface TvState {
    data object Loading : TvState
    data object NoServer : TvState

    /** Reached the server, not paired yet: this is the code on the screen. */
    data class Pairing(val address: String, val code: String, val expiresAt: Long) : TvState

    /** ⚠️ **Never paired, and the address answers nothing.** Deliberately not
     *  the same as [Offline]: there is nothing to keep playing and nobody to
     *  wait for. Somebody is standing in front of this screen right now with a
     *  remote, and the only useful thing it can do is show the address back to
     *  them — a mistyped restaurant name is by far the likeliest cause, and on a
     *  D-pad it has to come back filled in rather than empty.
     *
     *  @param answered whether the server replied at all. It changes who has to
     *  act: a reply means the address is right and that restaurant's server is
     *  behind, which nothing on the remote can fix. */
    data class Unreachable(val address: String, val answered: Boolean) : TvState

    /** ⚠️ Paired and currently unreachable. **Not** an error screen: the whole
     *  point of this app is that a dining room keeps working when the wifi does
     *  not, and a paired television with nothing to say should say nothing.
     *
     *  @param contentVersion carried across the drop rather than reset — the
     *  playlist on this set is still the branch's playlist, one revision behind
     *  at worst, and clearing it would make the screen re-download everything
     *  the moment the wifi came back, over the wifi that has just been
     *  struggling. */
    data class Offline(
        val address: String,
        val screen: TVScreenSelf?,
        val contentVersion: Int?,
    ) : TvState

    /** @param contentVersion null only until the first heartbeat lands — which
     *  is exactly the cold-boot-with-no-network case, and why the stored
     *  playlist is read before anything is asked. */
    data class Paired(
        val address: String,
        val screen: TVScreenSelf,
        val contentVersion: Int?,
    ) : TvState
}

/** How often a paired screen says hello.
 *
 *  ⚠️ **This is also how quickly "unpair" takes effect on the wall**, so it is a
 *  minute rather than an hour — and not shorter, because it runs all day on a
 *  restaurant's wifi for months. */
private const val HEARTBEAT_MS = 60_000L

/** How often the screen asks whether somebody has claimed it.
 *
 *  ⚠️ Quick, because it decides how long a manager stands in front of the
 *  television after typing the code, wondering whether it worked. */
private const val POLL_MS = 2_000L

/** How long before a code expires the next one is asked for.
 *
 *  ⚠️ **The rotation is driven by the server's own expiry, not a second timer
 *  here.** In the Expo build this was a constant — ten seconds — while the
 *  server said ninety, so the countdown on the wall was wrong by a factor of
 *  nine and the first person to pair a television could not finish typing before
 *  the code moved. */
private const val RENEW_LEAD_MS = 5_000L

/** How long between attempts while the address answers nothing.
 *
 *  ⚠️ **Because the commonest cause is not a typo but a television switched on
 *  before the router.** Somebody opens the restaurant, everything comes on at
 *  once, and this app reaches the network a few seconds early. A screen that
 *  needed a person to press something after that would need one every morning. */
private const val RETRY_MS = 15_000L

class TvViewModel(
    private val api: KeelApi,
    private val tokens: TokenStore,
    private val clock: Clock,
    val playlist: PlaylistStore,
    private val boardPoller: BoardPoller,
    private val appVersion: String,
    private val installId: String,
) : ViewModel() {

    private val _state = MutableStateFlow<TvState>(TvState.Loading)
    val state: StateFlow<TvState> = _state.asStateFlow()

    val board: StateFlow<Board> = boardPoller.board

    /** The secret this television polls with, for the code currently on screen. */
    private var pollSecret: String? = null

    /** The last thing the server said about this screen, kept across a dropped
     *  connection so an offline set can still say which room it belongs to. */
    private var known: TVScreenSelf? = null
    private var version: Int? = null

    // ⚠️ **One job per phase, cancelled on the way out.** The Expo build ran
    // these as effects keyed on the state — and a heartbeat *sets* the state, so
    // an unguarded one was not "one greeting" but a request loop running as fast
    // as the network answered, from a device nobody is watching. Jobs make that
    // impossible to write by accident.
    private var pairing: Job? = null
    private var beating: Job? = null
    private var retrying: Job? = null
    private var boarding: Job? = null
    private var content: Job? = null

    init { start() }

    private fun start() {
        viewModelScope.launch {
            clock.restore()
            // Before anything is asked of the network: this is the whole of what
            // a cold boot with a dead router has to play.
            playlist.loadStored()
            val address = tokens.read(TokenStore.SERVER_ADDRESS)
            if (address.isNullOrBlank()) {
                _state.value = TvState.NoServer
                return@launch
            }
            boot(address)
        }
    }

    /** Start talking to a restaurant's server: the token if we have one, a
     *  pairing code if we do not. */
    private suspend fun boot(address: String) {
        api.useServer(address)
        if (tokens.read(TokenStore.TV_TOKEN) != null) heartbeat(address) else askForCode(address)
    }

    /** Point this television at a restaurant. Answers false when what was typed
     *  cannot be an address at all. */
    fun useServer(address: String): Boolean {
        if (ServerAddress.apiBase(address).isEmpty()) return false
        tokens.write(TokenStore.SERVER_ADDRESS, address)
        // ⚠️ **Every timer carrying the old address is stopped first.** Each of
        // these loops closes over the address it was started for, so a screen
        // corrected from "osh" to "oshxona" would go on retrying "osh" every
        // fifteen seconds — and whichever answered last would set the state.
        // The symptom is a television that flickers between the address form
        // and a pairing code, in front of the person who just fixed the typo.
        stop(pairing, beating, retrying, boarding, content)
        pairing = null; beating = null; retrying = null; boarding = null; content = null
        boardPoller.clear()
        _state.value = TvState.Loading
        viewModelScope.launch { boot(address) }
        return true
    }

    // ---- Not paired ----

    private suspend fun askForCode(address: String) {
        try {
            val res = api.pairStart(installId, appVersion)
            pollSecret = res.pollSecret
            enter(
                TvState.Pairing(
                    address = address,
                    code = res.code,
                    // ⚠️ Counted from *our* clock plus the server's number of
                    // seconds, never from its timestamp against this device's
                    // clock: a cheap television's is routinely months out, and
                    // the countdown would sit at zero or never move.
                    expiresAt = System.currentTimeMillis() + res.expiresIn * 1000L,
                ),
            )
        } catch (e: Throwable) {
            // ⚠️ **The server not answering and the server refusing send somebody
            // to different people.** `ApiError` means the address is right and
            // reachable and the endpoint refused — which on this app almost
            // always means the restaurant's server has not been updated yet, and
            // nobody standing on a chair can fix that. Anything else is a name, a
            // router or a cable, which is exactly what they can.
            val answered = e is ApiError
            // ⚠️ **A code already on the screen stays there.** A television
            // flickering between a code and a message would be unreadable from
            // the only distance it is ever read from — and the code may well
            // still be good when the wifi comes back mid-rotation.
            if (_state.value !is TvState.Pairing) {
                enter(TvState.Unreachable(address, answered))
            }
        }
    }

    /** While unpaired: rotate the code as it expires, and poll for the claim. */
    private fun watchPairing(state: TvState.Pairing) {
        pairing?.cancel()
        pairing = viewModelScope.launch {
            val address = state.address
            var rotateAt = maxOf(
                RENEW_LEAD_MS,
                state.expiresAt - System.currentTimeMillis() - RENEW_LEAD_MS,
            )
            while (isActive) {
                delay(POLL_MS)
                rotateAt -= POLL_MS
                val secret = pollSecret
                if (secret != null) {
                    val res = runCatching { api.pairStatus(installId, secret) }.getOrNull()
                    // A failure here is being offline while waiting to be
                    // paired. The code on screen is already dead; the rotation
                    // below says so by failing too, and there is nothing a
                    // person can do about it from a remote.
                    val token = res?.token
                    val screen = res?.screen
                    if (res?.status == "paired" && token != null && screen != null) {
                        tokens.write(TokenStore.TV_TOKEN, token)
                        known = screen
                        pollSecret = null
                        // ⚠️ Null rather than zero: "not asked yet" and "the
                        // branch has an empty playlist" are different, and the
                        // heartbeat a moment from now is what tells them apart.
                        // Zero here would mean a freshly paired screen never
                        // fetched its content at all.
                        enter(TvState.Paired(address, screen, null))
                        return@launch
                    }
                    // ⚠️ "expired" is deliberately not handled: the rotation
                    // below already asks for a new code as this one runs out,
                    // and reacting here as well would ask for two.
                }
                if (rotateAt <= 0) {
                    askForCode(address)
                    return@launch
                }
            }
        }
    }

    // ---- Paired ----

    /** Am I still paired, and who am I? */
    private suspend fun heartbeat(address: String) {
        try {
            val res = api.me(appVersion)
            known = res.screen
            version = res.contentVersion
            // ⚠️ **Every heartbeat, not only the playlist fetch.** A dated slide
            // is compared against the restaurant's clock, and this set's own is
            // not usable for it — see Clock.kt. The playlist is re-read only
            // when it changes, which on a normal day is never, so this is the
            // only thing keeping the time honest on a screen left running for a
            // month.
            clock.note(res.serverTime)
            enter(TvState.Paired(address, res.screen, res.contentVersion))
        } catch (e: ApiError) {
            // ⚠️ **The server spoke, and it said no.** The screen was unpaired
            // from the panel — so the token goes now and the television asks for
            // a code again. Retrying would leave a set that was deliberately
            // taken out of service still playing.
            tokens.drop(TokenStore.TV_TOKEN)
            pollSecret = null
            askForCode(address)
        } catch (e: Throwable) {
            // The request never arrived. Keep playing, keep what we know.
            enter(TvState.Offline(address, known, version))
        }
    }

    // ---- One place where a state change starts and stops the timers ----

    private fun enter(next: TvState) {
        _state.value = next
        when (next) {
            is TvState.Pairing -> {
                stop(beating, retrying, boarding, content)
                beating = null; retrying = null; boarding = null; content = null
                boardPoller.clear()
                watchPairing(next)
            }

            is TvState.Unreachable -> {
                stop(pairing, beating, boarding, content)
                pairing = null; beating = null; boarding = null; content = null
                boardPoller.clear()
                if (retrying == null) {
                    retrying = viewModelScope.launch {
                        while (isActive) {
                            delay(RETRY_MS)
                            boot(next.address)
                        }
                    }
                }
            }

            is TvState.Paired, is TvState.Offline -> {
                stop(pairing, retrying)
                pairing = null; retrying = null
                val address = when (next) {
                    is TvState.Paired -> next.address
                    is TvState.Offline -> next.address
                    else -> ""
                }
                // ⚠️ Started once and left running, rather than restarted on
                // every heartbeat: the heartbeat sets the state, and a loop
                // recreated from a state change is a loop that runs as fast as
                // the network answers.
                if (beating == null) {
                    beating = viewModelScope.launch {
                        while (isActive) {
                            delay(HEARTBEAT_MS)
                            heartbeat(address)
                        }
                    }
                }
                syncContent(next)
                syncBoard(next)
            }

            else -> Unit
        }
    }

    private fun stop(vararg jobs: Job?) = jobs.forEach { it?.cancel() }

    /** Fetch the loop when the branch says it changed. */
    private fun syncContent(state: TvState) {
        val v = when (state) {
            is TvState.Paired -> state.contentVersion
            is TvState.Offline -> state.contentVersion
            else -> null
        } ?: return
        if (content?.isActive == true) return
        content = viewModelScope.launch { playlist.ensure(v) }
    }

    /** ⚠️ **The board is polled only by the screens that draw one.** A television
     *  in a dining room set to `content` has no use for the counter's numbers,
     *  and every screen in a chain asking for them every ten seconds is load the
     *  restaurant pays for in nothing. */
    private fun syncBoard(state: TvState) {
        val screen = when (state) {
            is TvState.Paired -> state.screen
            is TvState.Offline -> state.screen
            else -> null
        }
        val wants = screen?.mode == "board" || screen?.mode == "split"
        if (!wants) {
            boarding?.cancel()
            boarding = null
            boardPoller.clear()
            return
        }
        if (boarding?.isActive == true) return
        boarding = viewModelScope.launch {
            while (isActive) {
                boardPoller.poll()
                delay(BOARD_POLL_MS)
            }
        }
    }
}
