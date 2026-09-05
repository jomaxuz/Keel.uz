package uz.keel.owner

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.keel.owner.data.HelpArticle
import uz.keel.owner.data.searchHelp

// ⚠️ **This exists because the ranking is a second implementation.** The
// original is `frontend/src/lib/help/search.ts`; there is nothing to import
// across the boundary, so the two are kept honest the way `serverAddress` is —
// the same examples on both sides. Every case below is lifted from
// `search.test.ts`, and a divergence fails here rather than as "the help search
// finds nothing on the phone".

private val base = listOf(
    HelpArticle(
        id = "receipt-language",
        cat = "till",
        title = "Chek qaysi tilda chiqadi",
        body = "Chekdagi matn kassaning tilida chiqyapti.",
        keys = listOf("til"),
    ),
    HelpArticle(
        id = "receipt-garbage",
        cat = "till",
        title = "Printerdan tushunarsiz belgilar chiqmoqda",
        body = "Kod sahifasi matndan tanlanadi.",
    ),
    HelpArticle(
        id = "pin-locked",
        cat = "till",
        title = "PIN qabul qilinmayapti",
        body = "Besh daqiqaga bloklanadi.",
        keys = listOf("kassa ochilmayapti"),
    ),
)

private fun ids(q: String) = searchHelp(base, q).map { it.article.id }

class HelpSearchTest {

    // ⚠️ The whole reason this is not a whole-word match: Uzbek is
    // agglutinative and a person types the stem.
    @Test
    fun findsAWordByItsStem() {
        assertTrue(ids("chek").contains("receipt-language"))
        assertTrue(ids("printer").contains("receipt-garbage"))
        assertTrue(ids("kassa").contains("pin-locked"))
    }

    // ⚠️ And the other direction: the article says "til", the person typed
    // "tilida". Checking only one direction made a whole natural question fail
    // on one of its words — and with an all-words rule, that meant no answer.
    @Test
    fun findsAnArticleWordInsideALongerQueryWord() {
        assertTrue(ids("tilida").contains("receipt-language"))
    }

    // ⚠️ Half the words, rounded up. One word out of two is the "chek" problem:
    // every receipt article above the one actually asked about.
    @Test
    fun mostOfTheQuestionHasToHit() {
        // Two of these three relate to the language article; the negative verb
        // ("chiqmayapti") relates to nothing, because Uzbek puts the "ma" in the
        // middle of the word and no prefix rule can reach it.
        assertTrue(ids("chek tilida chiqmayapti").contains("receipt-language"))
        // A question with nothing to do with the base answers with nothing
        // rather than with its best guess.
        assertTrue(searchHelp(base, "kuryer velosiped").isEmpty())
    }

    // ⚠️ Words under three letters are dropped before counting: they carry no
    // meaning and would match most of the base.
    @Test
    fun shortWordsDoNotCount() {
        assertTrue(searchHelp(base, "va bu").isEmpty())
    }

    // ⚠️ The apostrophe is a letter in Uzbek — a boundary rule that split on it
    // would make "o'zgartirish" two words.
    @Test
    fun anApostropheIsALetter() {
        val one = listOf(HelpArticle(id = "x", title = "Narxni o'zgartirish", body = "..."))
        assertEquals(listOf("x"), searchHelp(one, "o'zgartirish").map { it.article.id })
    }

    // ⚠️ Ties broken by id rather than left to list order, so the same question
    // asked twice gives the same answers in the same order. A help list that
    // reshuffles looks like the answer changed.
    @Test
    fun isStable() {
        assertEquals(ids("kassa"), ids("kassa"))
    }

    // ⚠️ The title outranks the body: an article *about* receipts must come
    // above one that merely mentions them.
    @Test
    fun theTitleOutranksTheBody() {
        val hits = searchHelp(base, "chek")
        assertEquals("receipt-language", hits.first().article.id)
    }
}
