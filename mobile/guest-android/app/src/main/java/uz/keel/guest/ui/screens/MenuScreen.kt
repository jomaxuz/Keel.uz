package uz.keel.guest.ui.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material.icons.Icons
import androidx.compose.foundation.background
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Favorite
import androidx.compose.material.icons.rounded.FavoriteBorder
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import coil3.compose.AsyncImage
import uz.keel.design.Chip
import uz.keel.design.GhostButton
import uz.keel.design.GlassField
import uz.keel.design.KeelTheme
import uz.keel.design.LangSwitch
import uz.keel.design.GlassStepper
import uz.keel.design.Money
import uz.keel.design.glass
import uz.keel.design.imageUrl
import uz.keel.design.money
import uz.keel.guest.Brand
import uz.keel.guest.Cart
import uz.keel.guest.Favorites
import uz.keel.guest.cartLineOf
import uz.keel.guest.data.ApiError
import uz.keel.guest.data.KeelApi
import uz.keel.guest.data.MenuGroup
import uz.keel.guest.data.MenuItem
import uz.keel.guest.data.RestaurantResponse
import uz.keel.guest.data.pick
import uz.keel.guest.t
import uz.keel.guest.LocalPrefs
import androidx.compose.runtime.CompositionLocalProvider

// ---- The first screen, because there is no other ----
//
// ⚠️ **No home page.** Somebody who opened a restaurant's application is hungry.
// A cover photograph, the opening hours and an "Order now" button are one tap
// and one scroll between them and the menu, and each of those is where an order
// is lost. What a landing page would have said belongs where people go looking
// for it.
//
// ⚠️ **A closed kitchen still shows the menu.** Hiding it — or replacing it with
// "we are closed" — loses tomorrow's guest to save them a disappointment
// tonight. The header says the kitchen is shut and the menu stays where it was.
//
// ⚠️ **Sold out is drawn, never hidden.** A dish that vanishes makes a guest who
// came for it think they misremembered the restaurant; a dish greyed out with
// one word on it answers the question they actually have.

@Composable
fun MenuScreen(
    api: KeelApi,
    cart: Cart,
    favorites: Favorites,
    bottomInset: PaddingValues,
) {
    val c = KeelTheme.colors
    val prefs = LocalPrefs.current
    val lang = prefs.lang.value.code

    var profile by remember { mutableStateOf<RestaurantResponse?>(null) }
    var groups by remember { mutableStateOf<List<MenuGroup>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf("") }
    var query by remember { mutableStateOf("") }
    var category by remember { mutableStateOf("") }
    var tick by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()
    /** The dish somebody tapped. ⚠️ Held here rather than inside the card: a
     *  sheet owned by a row disappears when that row scrolls out of the list. */
    var opened by remember { mutableStateOf<MenuItem?>(null) }

    val loadFailed = t.menu.loadFailed

    // ⚠️ **Keyed on the language as well as the retry.** Dish names come from
    // the panel and the server picks the language from the header — a menu
    // loaded in Uzbek stays Uzbek after the switch unless it is asked again.
    LaunchedEffect(tick, lang) {
        api.lang = lang
        loading = true
        try {
            profile = api.restaurant()
            groups = api.menu()
            // ⚠️ Read here rather than only on the account screen: the heart is
            // on the first screen, and a menu that drew every dish unkept until
            // somebody visited their profile would look like it had forgotten.
            if (api.signedIn()) {
                runCatching { favorites.replace(api.me().favorites) }
            }
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else loadFailed
        } finally {
            loading = false
        }
    }

    val q = query.trim()
    /** What the list draws.
     *
     *  ⚠️ **Search crosses the categories and the chips step aside for it.** A
     *  guest typing "lag'mon" is not asking "is there lag'mon in soups"; a
     *  search filtered inside the selected category answers a question nobody
     *  asked and looks like an empty menu. */
    val shown = remember(groups, q, category, lang) {
        groups
            .filter { q.isNotEmpty() || category.isEmpty() || it.category.id == category }
            .map { g ->
                if (q.isEmpty()) g
                else g.copy(
                    items = g.items.filter {
                        it.pick(lang).contains(q, true) || it.describe(lang).contains(q, true)
                    },
                )
            }
            .filter { it.items.isNotEmpty() }
    }

    Box(Modifier.fillMaxSize()) {
        LazyColumn(
            Modifier.fillMaxSize().imePadding(),
            contentPadding = PaddingValues(
                start = 16.dp,
                end = 16.dp,
                bottom = bottomInset.calculateBottomPadding(),
            ),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Column(
                    Modifier.statusBarsPadding().padding(top = 12.dp),
                    verticalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Column(Modifier.weight(1f)) {
                            Text(
                                // ⚠️ The live name, not the build's: a
                                // restaurant that renames itself should not have
                                // to wait for a release to be called it.
                                profile?.restaurant?.name.orEmpty(),
                                style = MaterialTheme.typography.headlineMedium,
                                color = c.ink,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                            if (profile?.isOpenNow == false) {
                                Text(
                                    t.menu.closed,
                                    style = MaterialTheme.typography.labelMedium,
                                    color = c.warn,
                                )
                            }
                        }
                        // ⚠️ **In the header, not in a settings screen.** The
                        // language is the first thing a Russian-speaking guest
                        // needs and the last thing they will go hunting for.
                        LangSwitch()
                    }
                    if (profile?.isOpenNow == false) {
                        Text(
                            t.menu.closedHint,
                            style = MaterialTheme.typography.bodyMedium,
                            color = c.muted,
                        )
                    }
                    GlassField(query, { query = it }, t.menu.search)
                }
            }

            // ---- The categories ----
            // ⚠️ Hidden while searching: the chips would be a filter on top of a
            // filter, and the one a guest forgot they had set is the one that
            // makes the search look broken.
            if (q.isEmpty() && groups.size > 1) {
                item {
                    LazyRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        item {
                            Chip(t.menu.all, category.isEmpty()) { category = "" }
                        }
                        items(groups, key = { it.category.id }) { g ->
                            Chip(g.category.pick(lang), category == g.category.id) {
                                category = g.category.id
                            }
                        }
                    }
                }
            }

            when {
                loading && groups.isEmpty() -> item {
                    Box(Modifier.fillMaxWidth().padding(top = 48.dp), Alignment.Center) {
                        CircularProgressIndicator(color = c.accent)
                    }
                }

                error.isNotEmpty() -> item {
                    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        Text(error, style = MaterialTheme.typography.bodyMedium, color = c.danger)
                        GhostButton(t.menu.retry) { tick += 1 }
                    }
                }

                shown.isEmpty() -> item {
                    Text(
                        if (q.isEmpty()) t.menu.empty else t.menu.searchEmpty(q),
                        style = MaterialTheme.typography.bodyMedium,
                        color = c.muted,
                    )
                }
            }

            shown.forEach { group ->
                item(key = "c-${group.category.id}") {
                    Text(
                        group.category.pick(lang),
                        Modifier.padding(top = 6.dp),
                        style = MaterialTheme.typography.titleLarge,
                        color = c.ink,
                    )
                }
                // ⚠️ **Two to a row, and the pairs are made here rather than by
                // a grid.** A `LazyVerticalGrid` cannot be nested inside this
                // list — two lazy scrollers on one axis is a runtime crash — and
                // the categories have to stay as full-width headings between the
                // rows. Chunking is the shape that gives both.
                items(group.items.chunked(2), key = { it.first().id }) { pair ->
                    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                        pair.forEach { item ->
                            Box(Modifier.weight(1f)) {
                                DishCard(
                                    item = item,
                                    lang = lang,
                                    qty = cart.qtyOf(item.id),
                                    // ⚠️ **Not drawn at all when signed out.** A
                                    // heart that opened a sign-in form would be
                                    // this app asking for a phone number on the
                                    // menu.
                                    favorite = if (api.signedIn()) {
                                        favorites.holds(item.id)
                                    } else {
                                        null
                                    },
                                    onFavorite = {
                                        scope.launch {
                                            runCatching {
                                                favorites.replace(
                                                    api.toggleFavorite(item.id).favorites,
                                                )
                                            }
                                        }
                                    },
                                    onOpen = { opened = item },
                                    onAdd = {
                                        // ⚠️ **A dish with a compulsory question
                                        // goes to the sheet, never straight into
                                        // the basket.** Adding it here would put
                                        // a pizza of no chosen size on somebody's
                                        // order — the same rule the sheet's
                                        // disabled button follows.
                                        if (item.options.any { g -> g.required }) {
                                            opened = item
                                        } else {
                                            cart.add(cartLineOf(item, 1, "", emptyMap()))
                                        }
                                    },
                                    onStep = { delta ->
                                        val line = cart.plainLine(item.id)
                                        if (line == null) {
                                            opened = item
                                        } else {
                                            cart.setQty(line.key, line.qty + delta)
                                        }
                                    },
                                )
                            }
                        }
                        // ⚠️ An odd last dish keeps its half of the row rather
                        // than stretching across it: a card twice the width of
                        // every other reads as a different kind of thing.
                        if (pair.size == 1) Spacer(Modifier.weight(1f))
                    }
                }
            }
        }
    }

    opened?.let { item ->
        DishSheet(
            item = item,
            lang = lang,
            onClose = { opened = null },
            onAdd = { cart.add(it) },
        )
    }
}

/** One dish on the menu: a photograph, a name, a price, and a way to add it.
 *
 *  ⚠️ **Two to a row rather than one.** A full-width row fits a longer name —
 *  and the names here are long ("Qo'y go'shtli qazon kabob") — but it shows half
 *  as many dishes per screen, and a menu is browsed by looking rather than by
 *  reading. The name is given two lines to make up for the narrower card.
 *
 *  ⚠️ **The add control is on the card, so the commonest order never opens a
 *  sheet.** A guest ordering two of something they know should not have to open
 *  a dish, read it and come back. A dish with a compulsory question is the
 *  exception: `+` opens the sheet, because adding it here would put a pizza of
 *  no chosen size on the order.
 *
 *  ⚠️ **The stepper only ever edits the plain line** — the one with no options
 *  and no note. A minus that silently removed somebody's "no onion" would be
 *  this card editing a decision it never showed them. */
@Composable
private fun DishCard(
    item: MenuItem,
    lang: String,
    qty: Int,
    favorite: Boolean?,
    onFavorite: () -> Unit,
    onOpen: () -> Unit,
    onAdd: () -> Unit,
    onStep: (Int) -> Unit,
) {
    val c = KeelTheme.colors
    val out = !item.isAvailable
    Column(
        Modifier
            .fillMaxWidth()
            .glass(c, RoundedCornerShape(20.dp))
            // ⚠️ The whole card dims, photograph included. Greying only the
            // price leaves a bright picture of something nobody can order.
            .alpha(if (out) 0.55f else 1f)
            .clickable(enabled = !out, onClick = onOpen)
            .padding(8.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Box(Modifier.fillMaxWidth()) {
            if (item.imageUrl.isNotEmpty()) {
                AsyncImage(
                    // ⚠️ **A width is asked for, not the original.** The server
                    // resizes and caches (`/uploads/x.jpg?w=400`); a menu of
                    // forty full-size photographs is several megabytes over
                    // somebody's mobile data, and the first screen is the one
                    // that decides whether they wait.
                    model = imageUrl(item.imageUrl, Brand.uploadsBase, 400),
                    contentDescription = null,
                    contentScale = ContentScale.Crop,
                    modifier = Modifier
                        .fillMaxWidth()
                        .aspectRatio(1f)
                        .clip(RoundedCornerShape(14.dp)),
                )
            }
            if (favorite != null) {
                Icon(
                    if (favorite) Icons.Rounded.Favorite else Icons.Rounded.FavoriteBorder,
                    contentDescription = null,
                    tint = if (favorite) c.accent else c.onAccent,
                    modifier = Modifier
                        .align(Alignment.TopEnd)
                        .padding(6.dp)
                        .size(22.dp)
                        .clickable(onClick = onFavorite),
                )
            }
        }

        Text(
            item.pick(lang),
            style = MaterialTheme.typography.titleMedium,
            color = c.ink,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
        )

        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column {
                Money(item.price, color = c.ink)
                // ⚠️ Only where there genuinely is one — see MenuItem.oldPrice:
                // a null read as zero would strike out "0 so'm" on every
                // ordinary dish on the menu.
                if (item.discounted) {
                    Text(
                        money(item.oldPrice ?: 0.0),
                        style = MaterialTheme.typography.labelSmall,
                        color = c.muted,
                        textDecoration = TextDecoration.LineThrough,
                    )
                }
            }
            when {
                out -> Text(
                    t.menu.soldOut,
                    style = MaterialTheme.typography.labelMedium,
                    color = c.danger,
                )
                // ⚠️ **The stepper replaces the plus rather than sitting beside
                // it.** Two controls for one decision is the card asking a
                // question it has already been given the answer to.
                qty > 0 -> GlassStepper(
                    value = qty,
                    compact = true,
                    onMinus = { onStep(-1) },
                    onPlus = { onStep(1) },
                )
                else -> Box(
                    Modifier
                        .size(34.dp)
                        .clip(RoundedCornerShape(12.dp))
                        .background(c.accentSoft)
                        .clickable(onClick = onAdd),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        Icons.Rounded.Add,
                        contentDescription = t.dish.add,
                        tint = c.accent,
                        modifier = Modifier.size(20.dp),
                    )
                }
            }
        }
    }
}
