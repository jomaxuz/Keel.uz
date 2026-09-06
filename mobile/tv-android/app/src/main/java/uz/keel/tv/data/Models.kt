package uz.keel.tv.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// What the server sends this television.
//
// ⚠️ **A hand-written second copy of the panel's types** (`lib/types.ts` →
// `TVScreenSelf`, `TVSlide`), and the copies are kept honest by a test fed the
// JSON the Go handlers actually write — never JSON invented on this side, which
// would agree with this side by construction. kotlinx fills a default for any
// key it does not find, so a renamed field is not an error here: it is a
// confident empty playlist on a wall, with nobody in the room able to tell.

/** What a television is told about itself.
 *
 *  ⚠️ Deliberately narrow — a name, a branch and a mode. This is held by a
 *  device we do not control, hanging in a public room; the branch behind it
 *  carries addresses, keys and delivery settings. */
@Serializable
data class TVScreenSelf(
    val id: String = "",
    val name: String = "",
    /** `content`, `board` or `split`. ⚠️ Kept as text rather than an enum: an
     *  unknown mode from a newer server must leave the screen playing, and an
     *  enum would throw while parsing the answer that says so. */
    val mode: String = "content",
    val branchId: String = "",
    val branchName: String = "",
)

@Serializable
data class TVPairStart(
    val code: String = "",
    val pollSecret: String = "",
    /** Seconds. ⚠️ **The countdown is built from this and our own clock**, never
     *  from `expiresAt` against the set's: a cheap television's clock is
     *  routinely months out, and the number on the wall would sit at zero. */
    val expiresIn: Int = 60,
)

@Serializable
data class TVPairStatus(
    val status: String = "pending",
    val token: String? = null,
    val screen: TVScreenSelf? = null,
)

@Serializable
data class TVMe(
    val screen: TVScreenSelf = TVScreenSelf(),
    val serverTime: String = "",
    /** The branch's playlist revision, carried by the heartbeat so a screen
     *  learns the loop changed without asking a second question every minute. */
    val contentVersion: Int = 0,
)

@Serializable
data class TVSlideWire(
    val id: String = "",
    /** `image` or `video`. */
    val kind: String = "image",
    val url: String = "",
    /** How long a picture stays up. Ignored for a video. */
    val seconds: Int = 8,
    val startsAt: String? = null,
    val endsAt: String? = null,
)

@Serializable
data class TVPlaylistWire(
    val slides: List<TVSlideWire> = emptyList(),
    val version: Int = 0,
    val serverTime: String = "",
)

@Serializable
data class TVBoardWire(
    val cooking: List<String> = emptyList(),
    val ready: List<String> = emptyList(),
    val serverTime: String = "",
)

/** Which install this is, for the header the server binds a screen to. */
data class DeviceInfo(val id: String, val app: String, val name: String)

/** The error body every Keel handler writes. */
@Serializable
data class WireError(
    @SerialName("error") val error: String? = null,
    val message: String? = null,
)
