package uz.keel.tv

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.keel.tv.data.TVBoardWire
import uz.keel.tv.data.TVMe
import uz.keel.tv.data.TVPairStart
import uz.keel.tv.data.TVPairStatus
import uz.keel.tv.data.TVPlaylistWire

// The models, fed the shape the server actually sends.
//
// ⚠️ **This exists because a renamed field is silent here and loud nowhere.**
// `Models.kt` is a hand-written second copy of the panel's types, and kotlinx
// fills a default for any key it does not find — so a wrong name is not an
// error, it is an empty playlist on a wall with nobody in the room able to say
// why. The waiter build shipped three of these at once.
//
// ⚠️ **The JSON below is copied from the Go handlers**, not invented. A fixture
// written from the Kotlin side would agree with the Kotlin side by construction,
// and would have passed on the day every price read zero.

private val json = Json {
    ignoreUnknownKeys = true
    explicitNulls = false
    coerceInputValues = true
}

class WireShapeTest {

    /** `handlers/tvpair.go` → `TVPairStart`. */
    @Test
    fun `the code arrives with a life measured in seconds`() {
        val res = json.decodeFromString<TVPairStart>(
            """{"code":"7K2M9Q","pollSecret":"a1b2c3","expiresAt":"2026-09-06T11:00:00Z","expiresIn":90}""",
        )
        assertEquals("7K2M9Q", res.code)
        assertEquals("a1b2c3", res.pollSecret)
        // ⚠️ The whole reason this field exists: the countdown is built from it
        // and the set's own clock, never from `expiresAt`, which a television
        // months out of date would read as long expired.
        assertEquals(90, res.expiresIn)
    }

    /** `handlers/tvpair.go` → `TVPairStatus`, both answers. */
    @Test
    fun `pending carries no token, paired carries the screen`() {
        val pending = json.decodeFromString<TVPairStatus>("""{"status":"pending"}""")
        assertEquals("pending", pending.status)
        assertNull(pending.token)

        val paired = json.decodeFromString<TVPairStatus>(
            """{"status":"paired","token":"ey.J.tv",
                "screen":{"id":"s1","name":"Zal","mode":"split",
                          "branchId":"b1","branchName":"Chilonzor"}}""",
        )
        assertEquals("ey.J.tv", paired.token)
        assertEquals("split", paired.screen?.mode)
        assertEquals("Chilonzor", paired.screen?.branchName)
    }

    /** `handlers/tvpair.go` → `TVMe`. */
    @Test
    fun `the heartbeat carries the time and the playlist's version`() {
        val me = json.decodeFromString<TVMe>(
            """{"screen":{"id":"s1","name":"Kassa","mode":"board",
                          "branchId":"b1","branchName":"Yunusobod"},
                "contentVersion":7,"serverTime":"2026-09-06T11:22:33.123456789+05:00"}""",
        )
        assertEquals(7, me.contentVersion)
        assertEquals("board", me.screen.mode)
        // ⚠️ Go writes nanosecond precision and a numeric offset. A date parser
        // built from a pattern would fail on exactly this string, and the
        // failure is a television that believes the year is wrong.
        assertEquals(
            java.time.OffsetDateTime.parse("2026-09-06T11:22:33.123456789+05:00")
                .toInstant().toEpochMilli(),
            parseInstant(me.serverTime),
        )
    }

    /** `handlers/tvslides.go` → `TVPlaylist`. */
    @Test
    fun `a slide keeps its dates and the list keeps its version`() {
        val list = json.decodeFromString<TVPlaylistWire>(
            """{"slides":[
                 {"id":"a1","kind":"image","url":"https://osh.keel.uz/uploads/x.jpg","seconds":8},
                 {"id":"a2","kind":"video","url":"https://osh.keel.uz/uploads/y.mp4","seconds":0,
                  "startsAt":"2026-09-01T00:00:00Z","endsAt":"2026-09-30T23:59:59Z"}],
                "version":7,"serverTime":"2026-09-06T11:00:00Z"}""",
        )
        assertEquals(7, list.version)
        assertEquals(2, list.slides.size)
        assertEquals(8, list.slides[0].seconds)
        // ⚠️ The dates travel with the list rather than being applied by the
        // server, because a screen offline since Friday still has to take a
        // Saturday promotion down by itself.
        assertEquals("2026-09-30T23:59:59Z", list.slides[1].endsAt)
        assertNull(list.slides[0].endsAt)
    }

    /** `handlers/tvboard.go` → `TVBoard`. */
    @Test
    fun `the board is numbers and a time, and nothing else`() {
        val board = json.decodeFromString<TVBoardWire>(
            """{"cooking":["12","13"],"ready":["9"],"serverTime":"2026-09-06T11:00:00Z"}""",
        )
        assertEquals(listOf("12", "13"), board.cooking)
        assertEquals(listOf("9"), board.ready)
    }

    /** An empty branch. ⚠️ Go writes `[]` here rather than `null` — but a build
     *  that regressed to `null` must not throw on a wall, so both are read. */
    @Test
    fun `an empty board is empty, not a crash`() {
        assertTrue(json.decodeFromString<TVBoardWire>("""{"cooking":[],"ready":[]}""").ready.isEmpty())
        assertFalse(
            json.decodeFromString<TVBoardWire>("""{"cooking":null,"ready":null}""")
                .cooking.isNotEmpty(),
        )
    }
}
