package uz.keel.courier.location

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

// Where the courier is, held in one place the whole process reads.
//
// ⚠️ **One stream, two consumers that want different things.** The restaurant
// wants the last known position — the panel draws it on a map. The button on the
// order card wants *this* position, now, to decide whether the courier is at the
// door. So a fix is published here for the screen and buffered for the network,
// and neither waits for the other.
//
// ⚠️ **An object rather than a value passed down the tree.** The service that
// produces fixes is not in the composition and never will be: it outlives the
// activity by design. A `StateFlow` here is the seam between them, and it is the
// only one — a second copy of "where are we" is how a button opens against a
// position the server never received.

/** One position this phone took. */
data class Fix(
    val lat: Double,
    val lng: Double,
    val accuracy: Double,
    /** Unix ms, from the fix itself rather than from when it was read. */
    val at: Long,
)

/** What the shift's location stream is doing, for the card that says so. */
data class TrackingState(
    val fix: Fix? = null,
    /** True once positions are actually arriving, as opposed to permitted. */
    val live: Boolean = false,
    /** Unix ms of the last successful send, for the line under the switch. */
    val lastSentAt: Long? = null,
    /** How many fixes are waiting for a network. */
    val pending: Int = 0,
    /** Whether the foreground service is running — i.e. whether the shift
     *  survives the screen going off. */
    val running: Boolean = false,
)

object Tracking {
    private val _state = MutableStateFlow(TrackingState())
    val state: StateFlow<TrackingState> = _state.asStateFlow()

    fun publish(update: (TrackingState) -> TrackingState) {
        _state.value = update(_state.value)
    }

    /** ⚠️ Cleared when the shift ends, and the fix goes with it: a position from
     *  an hour ago is not where this courier is, and the arrival check would
     *  happily measure from it. */
    fun clear() {
        _state.value = TrackingState()
    }
}

/** Metres between two points, on the sphere.
 *
 *  ⚠️ **The same formula the server uses** (`haversineKm` in
 *  `internal/handlers`), because the button and the rule have to agree: a
 *  courier shown an open button and then refused by the server is a courier who
 *  believes the app is broken, in front of a customer. */
fun metersBetween(aLat: Double, aLng: Double, bLat: Double, bLng: Double): Double {
    val r = 6371000.0
    val dLat = Math.toRadians(bLat - aLat)
    val dLng = Math.toRadians(bLng - aLng)
    val s = Math.sin(dLat / 2) * Math.sin(dLat / 2) +
        Math.cos(Math.toRadians(aLat)) * Math.cos(Math.toRadians(bLat)) *
        Math.sin(dLng / 2) * Math.sin(dLng / 2)
    return r * 2 * Math.atan2(Math.sqrt(s), Math.sqrt(1 - s))
}
