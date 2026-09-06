package uz.keel.tv

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

// The two rules the television applies on its own, and both of them decide what
// a room is looking at.

private fun item(startsAt: String? = null, endsAt: String? = null) = PlayItem(
    id = "1", kind = "image", url = "https://osh.keel.uz/uploads/x.jpg",
    seconds = 8, startsAt = startsAt, endsAt = endsAt, path = "/data/x.jpg",
)

private fun at(iso: String) = java.time.Instant.parse(iso).toEpochMilli()

class PlaylistRulesTest {

    /** ⚠️ **The set applies the window itself.** The server could trim the list,
     *  but a screen that has not reached the internet since Friday would then
     *  still be showing an offer that ended on Saturday — and an expired
     *  promotion on a wall is worse than a blank one: a guest asks for it at the
     *  till. Same rule as `models.TVSlidePlayable`. */
    @Test
    fun `a window that has closed takes the slide down`() {
        val promo = item(startsAt = "2026-09-01T00:00:00Z", endsAt = "2026-09-30T23:00:00Z")
        assertFalse(playableNow(promo, at("2026-08-31T12:00:00Z")))
        assertTrue(playableNow(promo, at("2026-09-15T12:00:00Z")))
        assertFalse(playableNow(promo, at("2026-10-01T12:00:00Z")))
    }

    @Test
    fun `a slide with no dates always plays`() {
        assertTrue(playableNow(item(), at("2026-09-15T12:00:00Z")))
    }

    /** ⚠️ **An unparseable date must not hide a slide.** A malformed value from
     *  an older server would otherwise blank a wall permanently, and the
     *  failure would look like an empty playlist rather than like a bad date. */
    @Test
    fun `a date nobody can read is not a closed window`() {
        assertTrue(playableNow(item(endsAt = "yesterday"), at("2026-09-15T12:00:00Z")))
    }

    /** ⚠️ **The local name comes from the uploaded one, which is already random
     *  and unique** — so a replaced file is a different name and no stale copy
     *  can be played. Anything outside a safe alphabet is dropped rather than
     *  escaped: this string becomes a path. */
    @Test
    fun `a local name is the file's, and only ever a file name`() {
        assertEquals("abc123.jpg", localNameFor("https://osh.keel.uz/uploads/abc123.jpg"))
        assertEquals("abc123.jpg", localNameFor("https://osh.keel.uz/uploads/abc123.jpg?w=1200"))
        // A name that tried to walk out of the content directory comes back as
        // an ordinary name — never as a path.
        assertEquals("passwd", localNameFor("https://x/uploads/../../etc/passwd"))
        assertEquals("", localNameFor("https://x/uploads/.."))
        assertEquals("", localNameFor("https://osh.keel.uz/uploads/"))
    }
}
