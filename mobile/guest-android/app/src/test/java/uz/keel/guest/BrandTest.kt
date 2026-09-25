package uz.keel.guest

import androidx.compose.ui.graphics.Color
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.keel.design.LightColors
import uz.keel.design.branded

// The branding, tested where it is cheapest to be wrong.
//
// ⚠️ **This is the file that decides whether an app is one restaurant's or
// another's**, and every failure mode it has is silent: a colour that will not
// parse, a white label on a pale button, an address assembled with one slash too
// many. None of them throws, all of them ship.

class BrandTest {

    /** ⚠️ **An owner types this into a panel field.** `#fff`, `E2590D` with no
     *  hash, `  #e2590d  ` and an empty box all reach the build eventually, and
     *  a build that crashed on one would be a restaurant whose app cannot be
     *  made at all — over a colour. */
    @Test
    fun `a brand colour is read the way owners actually type it`() {
        assertEquals(Color(0xFFE2590D), parseColor("#E2590D"))
        assertEquals(Color(0xFFE2590D), parseColor("e2590d"))
        assertEquals(Color(0xFFE2590D), parseColor("  #e2590d  "))
        // Three digits is the shorthand a designer hands over.
        assertEquals(Color(0xFFFFFFFF), parseColor("#fff"))
        // Eight digits carry their own alpha.
        assertEquals(Color(0x80E2590D), parseColor("#80E2590D"))
    }

    /** ⚠️ **Unparseable is null, never a transparent colour.** Six digits carry
     *  no alpha, and the tempting `toLong()` of a bad string returning zero
     *  would paint every button fully transparent — an app that runs, draws
     *  nothing you can press, and reports no error anywhere. */
    @Test
    fun `nonsense is refused rather than turned into an invisible app`() {
        assertNull(parseColor(""))
        assertNull(parseColor("orange"))
        assertNull(parseColor("#12345"))
        assertNull(parseColor("#GGGGGG"))
    }

    /** ⚠️ **White text on a pale brand is invisible, and nothing errors.** A
     *  mustard or cream restaurant is not hypothetical, and the failure ships:
     *  the button works, so no test that clicks it fails, and "Buyurtma berish"
     *  is simply not there. */
    @Test
    fun `a pale brand gets dark text on its buttons`() {
        val pale = LightColors.branded(Color(0xFFF6E27A))
        assertTrue("pale brand kept white ink", pale.onAccent.luminanceIsDark())
        val deep = LightColors.branded(Color(0xFF0B3D91))
        assertTrue("deep brand lost its white ink", !deep.onAccent.luminanceIsDark())
    }

    /** ⚠️ **The mid-tones were the real failure.** A warm #F5A623 orange or a
     *  saffron sits under the old 0.55 luminance cut-off, so it got white
     *  text at under 2:1 — readable on no phone in daylight. Keel's own
     *  orange must still keep white: every staff app and printed receipt
     *  shows it that way. */
    @Test
    fun `a mid-tone brand gets dark text and Keel orange keeps white`() {
        assertTrue(LightColors.branded(Color(0xFFF5A623)).onAccent.luminanceIsDark())
        assertTrue(LightColors.branded(Color(0xFFFFC107)).onAccent.luminanceIsDark())
        assertTrue(!LightColors.branded(Color(0xFFE2590D)).onAccent.luminanceIsDark())
        assertTrue(!LightColors.branded(Color(0xFFC62828)).onAccent.luminanceIsDark())
    }

    /** ⚠️ **Null is Keel's own scheme, untouched.** The five staff applications
     *  pass nothing, and the day this returns a modified copy for them is the
     *  day a till stops matching a printed receipt. */
    @Test
    fun `no brand colour leaves the scheme exactly as it was`() {
        assertEquals(LightColors, LightColors.branded(null))
    }

    private fun Color.luminanceIsDark(): Boolean = this != Color.White
}

// ---- The basket ----

/** ⚠️ **Two lines of the same dish are one line only if every answer matches.**
 *  A plov with extra meat and a plov without are two things to cook, and merging
 *  them by dish id would send the kitchen one ticket for a dish nobody ordered. */
class CartKeyTest {

    @Test
    fun `the same dish with different answers stays two lines`() {
        val plain = CartLine("m1", "Osh", 45000.0, 1)
        val extra = CartLine(
            "m1", "Osh", 45000.0, 1,
            options = listOf(ChosenOption("Qo'shimcha", "Go'sht", 15000.0)),
        )
        val noted = plain.copy(comment = "piyozsiz")
        assertNotEquals(plain.key, extra.key)
        assertNotEquals(plain.key, noted.key)
        // The same answers, whatever the quantity, are the same line.
        assertEquals(plain.key, plain.copy(qty = 4).key)
    }

    /** ⚠️ **Display arithmetic only** — see Cart.kt. Every figure the guest is
     *  charged comes from `/orders/quote`; this one exists so a basket works
     *  with no network. */
    @Test
    fun `a line totals the options into every unit, not once`() {
        val line = CartLine(
            "m1", "Pitsa", 60000.0, 2,
            options = listOf(ChosenOption("O'lcham", "Katta", 15000.0)),
        )
        assertEquals(150000.0, line.lineTotal, 0.0)
    }
}
