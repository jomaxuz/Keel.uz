package uz.keel.owner.data

import kotlinx.serialization.Serializable

// Searching the help articles.
//
// ⚠️ **A second implementation of one rule, and it is written down rather than
// hidden.** The original is `frontend/src/lib/help/search.ts`; there is nothing
// to import across the boundary, so the two are kept honest the way
// `serverAddress` is — same examples, both sides, and the reasoning copied with
// the code so a future edit knows what it is allowed to change.
//
// ⚠️ The *articles* are not duplicated: they come from the server
// (`/admin/support/articles`), which is the whole reason that endpoint exists.

@Serializable
data class HelpArticle(
    val id: String = "",
    val cat: String = "",
    val title: String = "",
    val body: String = "",
    /** Words somebody would search that are not in the text.
     *
     *  ⚠️ This is where the vocabulary gap goes. An owner types "kassa
     *  ochilmayapti"; the article is titled "PIN qabul qilinmayapti". Neither
     *  contains the other's words, and without this the search finds nothing for
     *  the single most common question there is. */
    val keys: List<String> = emptyList(),
)

@Serializable
data class HelpArticles(val articles: List<HelpArticle> = emptyList())

data class HelpHit(val article: HelpArticle, val score: Int)

private const val WEIGHT_TITLE = 6
private const val WEIGHT_KEYS = 4
private const val WEIGHT_BODY = 1

/** The shortest article word that may swallow a longer query term. */
private const val STEM_MIN = 3

/** ⚠️ **The apostrophe is a letter in Uzbek** — "o'zgartirish" is one word — so
 *  it is deliberately not a boundary. A rule that split on it would make that
 *  two words and stop "o'zgar" from matching it. */
private fun splitWords(text: String): List<String> =
    text.lowercase().split(Regex("[^\\p{L}\\p{N}'’]+")).filter { it.isNotEmpty() }

/** Does any word in `text` relate to `term`?
 *
 *  ⚠️ **Both directions, because Uzbek suffixes both ways round.** The article
 *  says "chekda" and the person types "chek" — the article's word is longer. The
 *  article says "til" and the person types "tilida" — the query's is. Checking
 *  only the first direction made a whole natural question fail on one of its
 *  five words. */
private fun relates(text: String, term: String): Boolean =
    splitWords(text).any { w ->
        w.startsWith(term) || (w.length >= STEM_MIN && term.startsWith(w))
    }

/**
 * Ranks articles against what somebody typed.
 *
 * ⚠️ **Most of the query has to hit, not all of it and not one word.** One word
 * is too little: "chek" alone puts every receipt article above the one actually
 * about language. All of them is too much, and it was the first rule there — an
 * owner typing "chek rus tilida chiqmayapti" gets nothing, because "chiqmayapti"
 * is the negative of the article's "chiqyapti" and Uzbek puts that "ma" in the
 * *middle* of the word. No prefix rule can relate them, and a help search that
 * answers a keyword but not a sentence is one people stop typing sentences into.
 */
fun searchHelp(articles: List<HelpArticle>, query: String): List<HelpHit> {
    val terms = splitWords(query).filter { it.length >= 3 }
    if (terms.isEmpty()) return emptyList()

    val hits = mutableListOf<HelpHit>()
    for (article in articles) {
        var score = 0
        var matched = 0
        for (term in terms) {
            val best = when {
                relates(article.title, term) -> WEIGHT_TITLE
                article.keys.any { relates(it, term) } -> WEIGHT_KEYS
                relates(article.body, term) -> WEIGHT_BODY
                else -> 0
            }
            if (best == 0) continue
            matched++
            score += best
        }
        // ⚠️ Rounded up, so a two-word question needs both and a three-word one
        // needs two. Rounding down would let one word out of two answer, which
        // is the "chek" problem again.
        if (matched > 0 && matched * 2 >= terms.size) hits.add(HelpHit(article, score))
    }
    // ⚠️ Ties broken by id rather than left to list order, so the same question
    // asked twice gives the same answers in the same order. A help list that
    // reshuffles looks like the answer changed.
    return hits.sortedWith(compareByDescending<HelpHit> { it.score }.thenBy { it.article.id })
}

// ---- Talking to a person ----

@kotlinx.serialization.Serializable
data class SupportMessage(
    val id: String = "",
    val threadId: String = "",
    /** "owner" | "operator" | "assistant". ⚠️ The assistant is labelled as
     *  itself: an answer somebody believes came from a person, and did not, is
     *  the one they act on and then cannot ask a follow-up about. */
    val from: String = "",
    val author: String = "",
    val text: String = "",
    val at: String = "",
)

@kotlinx.serialization.Serializable
data class SupportThread(
    val id: String = "",
    val subject: String = "",
    /** "waiting" | "open" | "closed". */
    val status: String = "",
    val lastText: String = "",
    val lastAt: String = "",
    val unreadForOwner: Int = 0,
)

@kotlinx.serialization.Serializable
data class SupportThreads(val threads: List<SupportThread> = emptyList())

@kotlinx.serialization.Serializable
data class SupportThreadView(
    val thread: SupportThread = SupportThread(),
    val messages: List<SupportMessage> = emptyList(),
)

@kotlinx.serialization.Serializable
data class SupportAskResponse(
    val threadId: String = "",
    val message: SupportMessage = SupportMessage(),
)

@kotlinx.serialization.Serializable
data class AskArticle(val title: String, val body: String)
