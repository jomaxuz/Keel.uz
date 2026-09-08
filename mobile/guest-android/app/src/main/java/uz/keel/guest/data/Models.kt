package uz.keel.guest.data

import kotlinx.serialization.Serializable

// What the public API sends this phone.
//
// ⚠️ **A hand-written second copy of the site's types** (`frontend/src/lib/types.ts`),
// kept honest by a test fed the JSON the Go handlers actually write. kotlinx
// fills a default for any key it does not find, so a renamed field is not an
// error: it is a menu of dishes that all cost nothing.
//
// ⚠️ **Only what a guest screen draws is here.** The public endpoints answer with
// far more — costs, recipes, fiscal codes — and mirroring all of it would be a
// second catalogue to keep in step for no screen's benefit.

/** A name in the three languages a restaurant here is read in.
 *
 *  ⚠️ **Uzbek is the base and the other two are optional**, exactly as the
 *  panel stores them: a category the owner never translated must fall back to
 *  the word they typed, not to an empty chip. Every screen asks through
 *  `pick()`, never by reading `name` directly. */
interface Named {
    val name: String
    val nameRu: String
    val nameEn: String
}

/** The name in this language, falling back to the one that always exists. */
fun Named.pick(lang: String): String = when (lang) {
    "ru" -> nameRu.ifBlank { name }
    "en" -> nameEn.ifBlank { name }
    else -> name
}

@Serializable
data class Category(
    val id: String = "",
    override val name: String = "",
    override val nameRu: String = "",
    override val nameEn: String = "",
    val slug: String = "",
    val sortOrder: Int = 0,
    val isActive: Boolean = true,
    val imageUrl: String = "",
) : Named

@Serializable
data class MenuItem(
    val id: String = "",
    val categoryId: String = "",
    override val name: String = "",
    override val nameRu: String = "",
    override val nameEn: String = "",
    val description: String = "",
    val descriptionRu: String = "",
    val descriptionEn: String = "",
    /** Whole so'm. ⚠️ Never tiyin — the currency has none in practice and a
     *  screen that divided by a hundred would price a plov at 450. */
    val price: Double = 0.0,
    /** What it used to cost, when the restaurant is showing a reduction.
     *
     *  ⚠️ **Nullable, and null is not zero.** Zero would draw a struck-through
     *  "0 so'm" beside every ordinary dish on the menu. */
    val oldPrice: Double? = null,
    val imageUrl: String = "",
    /** ⚠️ **The one field that decides whether a dish can be ordered**, and it
     *  arrives already answering for both reasons: the owner switched it off,
     *  or the branch's stop list took it off today. The phone must not compute
     *  that a second way. */
    val isAvailable: Boolean = true,
) : Named {
    /** The description in this language, falling back to the base. */
    fun describe(lang: String): String = when (lang) {
        "ru" -> descriptionRu.ifBlank { description }
        "en" -> descriptionEn.ifBlank { description }
        else -> description
    }

    val discounted: Boolean get() = (oldPrice ?: 0.0) > price
}

/** One category with its dishes, the shape `GET /menu` answers in.
 *
 *  ⚠️ **Grouped by the server, not here.** The site reads the same shape, and a
 *  phone that regrouped a flat list would sooner or later disagree with it about
 *  which category an item is in — on the screen the whole app exists for. */
@Serializable
data class MenuGroup(
    val category: Category = Category(),
    val items: List<MenuItem> = emptyList(),
)

/** How this restaurant looks and what it is called.
 *
 *  ⚠️ **Read at launch even though the build already carries the name and the
 *  colour.** The build is a photograph taken on the day it was made; a
 *  restaurant that changes its logo would otherwise have to wait for a new
 *  release to see it. What the build carries is the *launcher* identity — the
 *  icon and the splash, which genuinely cannot change without one. */
@Serializable
data class Restaurant(
    val name: String = "",
    val description: String = "",
    val logoUrl: String = "",
    val coverUrl: String = "",
    val phones: List<String> = emptyList(),
    val currency: String = "",
    /** Whether the kitchen is taking orders right now. ⚠️ Computed by the
     *  server from the branch's hours, because a phone's clock is the one thing
     *  this product never trusts. */
    val isOpenNow: Boolean = true,
)
