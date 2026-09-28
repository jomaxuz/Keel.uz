package uz.keel.waiter.data

import androidx.compose.foundation.lazy.grid.LazyGridState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

// The room and the menu, held by the process rather than by a screen.
//
// ⚠️ **This is the fix for "every tap waits".** Both used to live in `remember`
// inside the screen that drew them — so switching to the Profile tab and back,
// or closing a check, threw the room away and drew a spinner while it was asked
// for again. On a restaurant's wifi that is a second or two, dozens of times a
// shift, and it is exactly what "the app keeps freezing" meant. Now a screen
// shows what is known at once and refreshes underneath it.
//
// ⚠️ **Stale-then-fresh, never fresh-or-nothing.** A forty-second-old room with
// the right tables on it is worth more to somebody walking than a spinner; the
// poll corrects it within seconds, and a refresh that fails keeps the last
// answer on screen instead of replacing it with an error.

/** Which tables the floor is showing. ⚠️ Kept here with the data, so coming back
 *  from a check lands on the same filter and zone the waiter left — resetting
 *  them sent people to the first zone every time they closed a table. */
enum class FloorFilter { All, Taken, Free, Ready, Mine }

class FloorStore(private val api: KeelApi) {

    var branch by mutableStateOf<BranchInfo?>(null)
        private set
    var checks by mutableStateOf<List<Check>>(emptyList())
        private set
    /** Dishes the branch cannot sell right now. Rides along with the checks
     *  list (`soldOut`), so the check screen needs no request of its own. */
    var soldOut by mutableStateOf<Set<String>>(emptySet())
        private set
    /** The last refresh's failure, or null. ⚠️ Kept apart from the data: a
     *  failed poll over a known room is a warning, not an empty screen. */
    var failure by mutableStateOf<Throwable?>(null)
        private set
    var refreshing by mutableStateOf(false)
        private set

    var zone by mutableStateOf<String?>(null)
    var filter by mutableStateOf(FloorFilter.All)

    /** ⚠️ The scroll position survives a visit to a check for the same reason
     *  the zone does: forty tables is two screens, and the table somebody just
     *  closed is usually not in the first one. */
    val grid = LazyGridState()

    private var branchAt = 0L
    private val gate = Mutex()
    /** Bumped by [clear]. ⚠️ A refresh that was already on the wire when
     *  somebody signed out must not write the old room into the store after it
     *  was emptied — the next person on this phone would open on it. */
    private var generation = 0

    val loaded: Boolean get() = branch != null

    /** Ask for the room.
     *
     *  ⚠️ **The two requests go together, not one after the other.** They are
     *  independent, and in sequence the floor waited for the sum of two round
     *  trips on every open.
     *
     *  ⚠️ **The branch (tables and zones) is asked for rarely.** It changes when
     *  somebody redraws the room in the panel, which is not during service; the
     *  checks change every minute. [full] forces it — the refresh button and a
     *  pull. */
    suspend fun refresh(full: Boolean = false) {
        // One at a time: a poll landing on top of a pull is the same question
        // asked twice, and the slower answer would overwrite the newer one.
        if (!gate.tryLock()) return
        refreshing = true
        val gen = generation
        try {
            val now = System.currentTimeMillis()
            val needBranch = full || branch == null || now - branchAt > BRANCH_MAX_AGE_MS
            coroutineScope {
                val b = if (needBranch) async { api.branch() } else null
                val ch = async { api.checks() }
                val list = ch.await()
                val br = b?.await()
                if (gen != generation) return@coroutineScope
                if (br != null) {
                    branch = br
                    branchAt = now
                }
                checks = list.checks
                soldOut = list.soldOut.toSet()
                failure = null
            }
        } catch (e: CancellationException) {
            throw e
        } catch (e: Throwable) {
            if (gen == generation) failure = e
        } finally {
            refreshing = false
            gate.unlock()
        }
    }

    /** A check a screen just changed, written into the room at once.
     *
     *  ⚠️ Without this the table's total on the floor was the one from before
     *  the waiter added three dishes, until the next poll — and the number on
     *  the tile is what they read to the guest who asks. */
    fun upsert(c: Check) {
        if (c.id.isEmpty()) return
        val i = checks.indexOfFirst { it.id == c.id }
        checks = if (i >= 0) checks.toMutableList().also { it[i] = c } else checks + c
    }

    fun clear() {
        generation++
        branch = null
        checks = emptyList()
        soldOut = emptySet()
        failure = null
        zone = null
        filter = FloorFilter.All
        branchAt = 0L
    }

    companion object {
        /** How long the room's layout is trusted. */
        const val BRANCH_MAX_AGE_MS = 5 * 60_000L
    }
}

/** The menu, per branch, kept between tables.
 *
 *  ⚠️ **It was fetched again on every table opened**, with photographs' worth of
 *  names and prices, and the menu tab was empty until it arrived — the moment a
 *  guest has just said what they want. Now it is fetched once, shown from memory
 *  after that, and refreshed quietly when it is older than a few minutes. What
 *  changes during service (a dish running out) comes from the stop list, not
 *  from here. */
class MenuCache(private val api: KeelApi) {

    private var branchId by mutableStateOf("")
    private var groups by mutableStateOf<List<MenuGroup>>(emptyList())
    private var loadedAt = 0L
    private val gate = Mutex()

    var failed by mutableStateOf(false)
        private set

    /** What is known for [branch]. ⚠️ Empty rather than another branch's menu:
     *  a phone that moved branches must not sell the old room's dishes. */
    fun groupsFor(branch: String): List<MenuGroup> =
        if (branch == branchId) groups else emptyList()

    suspend fun ensure(branch: String, force: Boolean = false) {
        if (branch.isEmpty()) return
        gate.withLock {
            val fresh = branch == branchId && groups.isNotEmpty() &&
                System.currentTimeMillis() - loadedAt < MAX_AGE_MS
            if (fresh && !force) return
            try {
                val g = api.menu(branch)
                groups = g
                branchId = branch
                loadedAt = System.currentTimeMillis()
                failed = false
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                // ⚠️ A failed refresh keeps the menu it had. Only a menu that was
                // never loaded is a failure worth drawing.
                failed = groupsFor(branch).isEmpty()
            }
        }
    }

    fun clear() {
        branchId = ""
        groups = emptyList()
        loadedAt = 0L
        failed = false
    }

    companion object {
        const val MAX_AGE_MS = 5 * 60_000L
    }
}
