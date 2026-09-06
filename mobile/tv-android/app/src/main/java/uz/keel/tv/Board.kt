package uz.keel.tv

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import uz.keel.tv.data.KeelApi

// The order board's half of the state.
//
// ⚠️ **The opposite staleness rule from the playlist, and deliberately so.** A
// loop that keeps playing through a dropped connection is showing the
// restaurant's own content, one revision behind at worst. A board that keeps
// showing numbers is making a claim about food — and a number that says "ready"
// when it is not sends a guest to a counter to be told no. Better a screen that
// says nothing: they ask a person, which is what they would have done anyway.

data class Board(val cooking: List<String> = emptyList(), val ready: List<String> = emptyList()) {
    /** Whether there is anything worth drawing. A quiet afternoon is not a
     *  board. */
    val hasAnything: Boolean get() = cooking.isNotEmpty() || ready.isNotEmpty()
}

/** How often the wall asks.
 *
 *  ⚠️ Not the heartbeat's minute: a guest standing at a counter watching their
 *  number not appear is the whole experience this screen delivers, and a minute
 *  of that is a complaint. */
const val BOARD_POLL_MS = 10_000L

/** How stale an answer may be before the board goes quiet. */
const val BOARD_STALE_MS = 2 * 60_000L

class BoardPoller(private val api: KeelApi, private val clock: Clock) {

    private val _board = MutableStateFlow(Board())
    val board: StateFlow<Board> = _board.asStateFlow()

    /** When the answer being held was true. ⚠️ Server time, not the set's own: a
     *  cheap television's clock is months out, and this comparison decides
     *  whether a room is shown numbers or nothing. */
    private var at: Long = 0

    suspend fun poll() {
        try {
            val res = api.board()
            clock.note(res.serverTime)
            at = clock.now()
            _board.value = Board(res.cooking, res.ready)
        } catch (e: Throwable) {
            // Silence. Whether this becomes a blank board is decided here, by
            // how old the last good answer is — one dropped request on a
            // restaurant's wifi is not a reason to clear a wall.
            if (clock.now() - at > BOARD_STALE_MS) {
                at = 0
                _board.value = Board()
            }
        }
    }

    /** Forget everything. Used when a screen stops drawing a board at all —
     *  numbers held from an hour ago must not reappear if the mode changes
     *  back. */
    fun clear() {
        at = 0
        _board.value = Board()
    }
}
