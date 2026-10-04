package uz.keel.team

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshots.SnapshotStateMap

// A count in progress, held by the process rather than by the screen.
//
// ⚠️ **A count is forty numbers typed over half an hour in a cold room, and
// every one of them is a walk to a shelf.** Held inside the screen, the draft
// would be thrown away by the most ordinary thing somebody does mid-count:
// tapping another tab to check what was asked for this morning, or turning the
// phone. The web page this replaces lost the lot on a refresh, which is one of
// the reasons counts were still being typed in the office afterwards.
//
// ⚠️ **Kept per store, never merged.** A count is one room (models/warehouse.go),
// and a draft that followed the person rather than the shelf would put the bar's
// numbers on the kitchen's sheet the moment somebody switched stores to look
// something up — where they would look like counts of the kitchen.
//
// ⚠️ **Not written to disk, and that is a decision rather than an omission.**
// What is on the shelf right now is only true right now: a draft restored three
// days after the process died would offer numbers nobody has any reason to
// trust, in the one screen whose whole product is a number somebody trusts. It
// survives tabs, rotation and the keyboard; it does not survive being killed,
// and the store's picker is the last thing this class remembers so the way back
// is one tap.
class CountDraft {

    /** Which room is being counted. ⚠️ Remembered because it is chosen once and
     *  a count outlives several visits to this screen. */
    var warehouse by mutableStateOf("")

    private val byStore = mutableMapOf<String, SnapshotStateMap<String, String>>()

    /** What has been typed for one store, as text.
     *
     *  ⚠️ **Text, not numbers.** "1." and "0" and "" are three different states
     *  of somebody's thumb, and only the middle one is a quantity; parsing on
     *  every keystroke would delete the decimal point as it was typed. */
    fun of(warehouseId: String): SnapshotStateMap<String, String> =
        byStore.getOrPut(warehouseId) { mutableStateMapOf() }

    /** ⚠️ Called only after the server has the count. Clearing on anything else
     *  — a failed save, a switched store — throws away the walk, not the draft. */
    fun clear(warehouseId: String) {
        byStore.remove(warehouseId)
    }
}
