package uz.keel.guest.ui.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
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
import uz.keel.design.Money
import uz.keel.design.glass
import uz.keel.design.imageUrl
import uz.keel.guest.Brand
import uz.keel.guest.Cart
import uz.keel.guest.Favorites
import uz.keel.guest.data.ApiError
import uz.keel.guest.data.KeelApi
import uz.keel.guest.data.MenuGroup
import uz.keel.guest.data.MenuItem
import uz.keel.guest.data.Restaurant
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

    var restaurant by remember { mutableStateOf<Restaurant?>(null) }
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
            restaurant = api.restaurant()
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
                                restaurant?.name.orEmpty(),
                                style = MaterialTheme.typography.headlineMedium,
                                color = c.ink,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                            if (restaurant?.isOpenNow == false) {
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
                    if (restaurant?.isOpenNow == false) {
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
                items(group.items, key = { it.id }) { item ->
                    DishCard(
                        item = item,
                        lang = lang,
                        // ⚠️ **Not drawn at all when signed out.** A heart that
                        // opened a sign-in form would be this app asking for a
                        // phone number on the menu — the one thing it never does
                        // before somebody has eaten.
                        favorite = if (api.signedIn()) favorites.holds(item.id) else null,
                        onFavorite = {
                            scope.launch {
                                runCatching {
                                    favorites.replace(api.toggleFavorite(item.id).favorites)
                                }
                            }
                        },
                        onOpen = { opened = item },
                    )
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

/** One dish, as a guest reads it: a photograph, a name, a price.
 *
 *  ⚠️ **A row rather than a grid tile.** A grid fits more dishes on a screen and
 *  gives each of them a name truncated to two words — and the names in a
 *  restaurant menu here are long ("Qo'y go'shtli qazon kabob"). A guest
 *  scrolling past six dishes they can read is better served than one scanning
 *  twelve they cannot. */
@Composable
private fun DishCard(
    item: MenuItem,
    lang: String,
    favorite: Boolean?,
    onFavorite: () -> Unit,
    onOpen: () -> Unit,
) {
    val c = KeelTheme.colors
    val out = !item.isAvailable
    Row(
        Modifier
            .fillMaxWidth()
            .glass(c, RoundedCornerShape(20.dp))
            // ⚠️ The whole card dims, photograph included. Greying only the
            // price leaves a bright picture of something nobody can order.
            .alpha(if (out) 0.55f else 1f)
            .clickable(enabled = !out, onClick = onOpen)
            .padding(10.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (item.imageUrl.isNotEmpty()) {
            AsyncImage(
                // ⚠️ **A width is asked for, not the original.** The server
                // resizes and caches (`/uploads/x.jpg?w=300`); a menu of forty
                // full-size photographs is several megabytes over somebody's
                // mobile data, and the first screen is the one that decides
                // whether they wait.
                model = imageUrl(item.imageUrl, Brand.uploadsBase, 300),
                contentDescription = null,
                contentScale = ContentScale.Crop,
                modifier = Modifier
                    .width(92.dp)
                    .aspectRatio(1f)
                    .clip(RoundedCornerShape(14.dp)),
            )
        }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(
                item.pick(lang),
                style = MaterialTheme.typography.titleMedium,
                color = c.ink,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            val about = item.describe(lang)
            if (about.isNotEmpty()) {
                Text(
                    about,
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.muted,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Money(item.price, color = c.ink)
                // ⚠️ Only where there genuinely is one — see MenuItem.oldPrice:
                // a null read as zero would strike out "0 so'm" on every
                // ordinary dish on the menu.
                if (item.discounted) {
                    Text(
                        uz.keel.design.money(item.oldPrice ?: 0.0),
                        style = MaterialTheme.typography.labelMedium,
                        color = c.muted,
                        textDecoration = TextDecoration.LineThrough,
                    )
                }
                if (out) {
                    Text(
                        t.menu.soldOut,
                        style = MaterialTheme.typography.labelMedium,
                        color = c.danger,
                    )
                }
            }
        }
        if (favorite != null) {
            Icon(
                if (favorite) Icons.Rounded.Favorite else Icons.Rounded.FavoriteBorder,
                contentDescription = null,
                tint = if (favorite) c.accent else c.muted,
                modifier = Modifier
                    .size(22.dp)
                    .clickable(onClick = onFavorite),
            )
        }
    }
}
