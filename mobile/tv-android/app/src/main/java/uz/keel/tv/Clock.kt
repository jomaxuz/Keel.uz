package uz.keel.tv

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import uz.keel.design.TokenStore

// What time it is in the restaurant.
//
// ⚠️ **A television's own clock is not usable for this.** A cheap Android set
// with no network at boot comes up in 1970, or on the date it left the factory,
// and stays there until something corrects it. Every schedule this app follows
// — a promotion that ends on the 30th — is a comparison against that clock, so
// getting it wrong means a dining room showing an offer the till stopped
// honouring last week, or hiding one that started this morning.
//
// So the server says what time it is, on every heartbeat, and this holds the
// difference. The device's clock is still used to *count*: it runs at the right
// speed even when it starts from the wrong place.

private const val CLOCK_KEY = "keel_tv_clock"

@Serializable
private data class Reading(val server: Long, val device: Long)

class Clock(private val tokens: TokenStore) {

    private val json = Json { ignoreUnknownKeys = true }

    /** Milliseconds to add to `System.currentTimeMillis()` to get the
     *  restaurant's time. */
    @Volatile
    private var offset: Long = 0

    /** Whether the server has confirmed the time this launch.
     *
     *  ⚠️ Exposed rather than folded into "trust me": what a screen shows
     *  before the first heartbeat differs from what it shows after, and a caller
     *  that could not tell the two apart would guess. */
    @Volatile
    var confirmed: Boolean = false
        private set

    /** Restore the last known offset, so a cold boot with no network has a
     *  better guess than the set's own idea of the date.
     *
     *  ⚠️ **A guess, and knowingly so.** If the clock was reset by the power cut
     *  that caused the reboot, this offset is wrong — but it is wrong by the
     *  same amount the raw clock is, and the first heartbeat that gets through
     *  corrects it. What it buys is the minute before that: the screen comes
     *  back to what it was playing rather than to every dated slide at once. */
    fun restore() {
        val raw = tokens.read(CLOCK_KEY) ?: return
        val saved = runCatching { json.decodeFromString<Reading>(raw) }.getOrNull() ?: return
        offset = saved.server - saved.device
    }

    /** Record what the server just said the time was.
     *
     *  ⚠️ **Called from the heartbeat, not only from the playlist fetch.** The
     *  playlist is re-read only when it changes, which on a normal day is never,
     *  so a set left running for a month would drift on whatever its own clock
     *  does. The heartbeat happens every minute anyway and this costs nothing. */
    fun note(iso: String) {
        val server = parseInstant(iso) ?: return
        val device = System.currentTimeMillis()
        offset = server - device
        confirmed = true
        // ⚠️ Persisted as the pair, not as the difference: a difference is only
        // meaningful against the device clock it was measured from, and a set
        // that lost power may come back with a completely different one.
        tokens.write(CLOCK_KEY, json.encodeToString(Reading.serializer(), Reading(server, device)))
    }

    /** The restaurant's time, as well as this television can know it. */
    fun now(): Long = System.currentTimeMillis() + offset
}

/** An instant off the wire, in milliseconds, or null.
 *
 *  ⚠️ **`Instant.parse`, never a date formatter with a pattern.** Go writes
 *  `time.Time` as RFC 3339 with a variable number of fractional digits and
 *  either `Z` or an offset; a pattern that matched what today's server happens
 *  to send would fail silently on the day it sends one digit fewer — and this
 *  value decides whether a promotion is on the wall. */
fun parseInstant(iso: String?): Long? {
    if (iso.isNullOrBlank()) return null
    return runCatching { java.time.Instant.parse(iso).toEpochMilli() }.getOrNull()
        ?: runCatching { java.time.OffsetDateTime.parse(iso).toInstant().toEpochMilli() }.getOrNull()
}
