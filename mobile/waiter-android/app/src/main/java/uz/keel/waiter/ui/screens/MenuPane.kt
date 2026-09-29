package uz.keel.waiter.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyGridState
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.GridView
import androidx.compose.material.icons.rounded.Image
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.rounded.ViewList
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import uz.keel.waiter.data.MenuGroup
import uz.keel.waiter.data.MenuItem
import uz.keel.waiter.data.displayName
import uz.keel.design.imageUrl
import uz.keel.design.money
import uz.keel.waiter.t
import uz.keel.design.*
import uz.keel.waiter.ui.components.DishCard
import uz.keel.waiter.ui.components.MenuView

// The menu, as a waiter reads it.
//
// ⚠️ **The names are the restaurant's own text, and they have three versions.**
// `contentName` is the site's rule, ported unchanged: Uzbek is the base and a
// missing translation falls back to it, which is what stops a half-translated
// menu from showing empty rows.

/** Where the waiter was in the menu.
 *
 *  ⚠️ **Held by the check screen, not by this pane.** The pane is thrown away
 *  every time the waiter flips to the check and back — which they do after
 *  every few dishes, to read the table back — and it used to come back on the
 *  first category, scrolled to the top, with the search cleared. The drinks
 *  they were halfway through were three taps away again. */
class MenuPaneState {
    var category by mutableIntStateOf(0)
    var query by mutableStateOf("")
    var onlyAdded by mutableStateOf(false)
    val grid = LazyGridState()
    val chips = LazyListState()
}

@Composable
fun MenuPane(
    groups: List<MenuGroup>,
    /** How many of each dish are already on the check. */
    onCheck: Map<String, Int>,
    /** What the branch has run out of today, from the room's poll. */
    soldOut: Set<String>,
    view: MenuView,
    onView: (MenuView) -> Unit,
    lang: String,
    uploadsBase: String,
    bottomPad: Dp,
    onAdd: (MenuItem) -> Unit,
    /** Take one off. ⚠️ Needed here and not only on the check: a waiter who has
     *  just tapped one too many is looking at the menu, and sending them to
     *  another tab to undo a tap they made a second ago is how a wrong count
     *  survives to the kitchen. */
    onRemove: (MenuItem) -> Unit,
    state: MenuPaneState = remember { MenuPaneState() },
) {
    val c = KeelTheme.colors
    val category = state.category.coerceIn(0, (groups.size - 1).coerceAtLeast(0))
    val query = state.query
    val onlyAdded = state.onlyAdded
    val searching = query.isNotBlank()

    /** What to draw.
     *
     *  ⚠️ **Searching crosses categories, browsing does not.** A waiter typing a
     *  name is answering a guest and does not know or care which section it is
     *  filed under; a waiter tapping through sections is reading the menu the way
     *  it is laid out. Making search obey the selected category would hide the
     *  dish from the person who asked for it by name — which reads as the dish
     *  not existing.
     *
     *  ⚠️ **Matched on the translated name, not the base one.** A Russian-speaking
     *  waiter types what they see; searching the Uzbek text would find nothing.
     *  The base is matched too, so a name with no translation is still reachable.
     *
     *  ⚠️ **Not keyed on the counts unless the filter reads them.** Every tap
     *  changes `onCheck`, and re-filtering two hundred dishes on each one is work
     *  the list does not need when it is not showing "only what is on the check". */
    val countsKey: Any = if (onlyAdded) onCheck else Unit
    val items = remember(groups, category, query, onlyAdded, countsKey, lang) {
        val q = query.trim().lowercase()
        val pool = if (searching) groups.flatMap { it.items }
        else groups.getOrNull(category)?.items ?: emptyList()
        pool.filter {
            if (onlyAdded && (onCheck[it.id] ?: 0) == 0) return@filter false
            if (q.isEmpty()) return@filter true
            it.displayName(lang).lowercase().contains(q) || it.name.lowercase().contains(q)
        }
    }

    /** How many dishes from each category are on the check — said on its chip,
     *  so "did I add the drinks?" is answered without opening every section. */
    val perCategory = remember(groups, onCheck) {
        groups.map { g -> g.items.sumOf { onCheck[it.id] ?: 0 } }
    }

    // ⚠️ A new section starts at its top. The grid kept its scroll offset across
    // categories, so a waiter who switched from a long section to a short one
    // landed in the middle of nothing.
    LaunchedEffect(category, searching, onlyAdded) {
        if (state.grid.firstVisibleItemIndex > 0 || state.grid.firstVisibleItemScrollOffset > 0) {
            state.grid.scrollToItem(0)
        }
    }
    // And the chosen chip stays in view — the strip is wider than the phone.
    // (Index `category` is the chip *before* the chosen one — "on the check"
    // sits first — so the neighbour on the left stays visible as a hint.)
    LaunchedEffect(category, searching) {
        if (!searching && groups.isNotEmpty()) state.chips.animateScrollToItem(category)
    }

    val views = listOf(
        MenuView.List to Icons.Rounded.ViewList,
        MenuView.Cards to Icons.Rounded.GridView,
        MenuView.Photos to Icons.Rounded.Image,
    )

    Column(Modifier.fillMaxSize()) {
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(Modifier.weight(1f)) {
                GlassField(
                    value = query, onValueChange = { state.query = it },
                    placeholder = t.menu.search,
                    keyboardOptions = KeyboardOptions(
                        capitalization = KeyboardCapitalization.None, autoCorrectEnabled = false,
                    ),
                    trailing = {
                        if (searching) GlassIconButton(Icons.Rounded.Close) { state.query = "" }
                        else Icon(Icons.Rounded.Search, null, tint = c.muted, modifier = Modifier.size(18.dp))
                    },
                )
            }
            // ⚠️ Cycled by one button rather than three: this is a setting somebody
            // changes once and then never, and three permanent controls beside a
            // search box is a row of things to press instead of a menu.
            GlassIconButton(views.first { it.first == view }.second, tint = c.ink) {
                onView(views[(views.indexOfFirst { it.first == view } + 1) % views.size].first)
            }
        }

        // ⚠️ **A fixed height and no shrinking.** As a row of chips inside a
        // column this collapsed the moment the list beside it grew — so on
        // exactly the categories with the most dishes, the strip naming them
        // disappeared. It is the one control that says where you are.
        if (!searching) {
            LazyRow(
                Modifier.fillMaxWidth().height(50.dp),
                state = state.chips,
                contentPadding = PaddingValues(horizontal = 16.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                // "On the check" first: it is the filter used while reading the
                // table back, and at the end of a long strip it was never found.
                item(key = "added") {
                    Chip(t.menu.onCheck, onlyAdded, icon = Icons.Rounded.Check) {
                        state.onlyAdded = !onlyAdded
                    }
                }
                items(groups.size, key = { groups[it].category.id.ifEmpty { "i$it" } }) { i ->
                    CategoryChip(
                        groups[i].category.displayName(lang),
                        perCategory.getOrElse(i) { 0 },
                        i == category && !onlyAdded,
                    ) {
                        state.onlyAdded = false
                        state.category = i
                    }
                }
            }
        } else {
            Text(
                t.menu.found(items.size), color = c.muted,
                style = MaterialTheme.typography.labelMedium,
                modifier = Modifier.padding(horizontal = 20.dp, vertical = 6.dp),
            )
        }

        // ⚠️ **A lazy grid, never a Column of everything.** A menu of two hundred
        // dishes with photographs composed all at once is what makes a cheap
        // Android stutter — the same reason the Expo build moved off ScrollView.
        LazyVerticalGrid(
            state = state.grid,
            // ⚠️ **Two, counted, not `Adaptive`.** Adaptive(160.dp) asks how many
            // 160dp columns fit — and on a 320dp phone, after padding and the
            // gap, the answer is one. So the photo view drew a single column of
            // wide cards on exactly the cheap phones it is meant for, which is
            // the list view with pictures rather than a grid.
            columns = if (view == MenuView.List) GridCells.Fixed(1) else GridCells.Fixed(2),
            contentPadding = PaddingValues(16.dp, 4.dp, 16.dp, bottomPad),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalArrangement = Arrangement.spacedBy(if (view == MenuView.List) 8.dp else 10.dp),
            modifier = Modifier.fillMaxSize(),
        ) {
            // ⚠️ With the only-on-check filter the list spans categories, so the
            // selected section would be a lie; the whole check is the section.
            val pool = if (onlyAdded && !searching) {
                groups.flatMap { it.items }.filter { (onCheck[it.id] ?: 0) > 0 }
            } else items
            items(pool, key = { it.id }, contentType = { view }) { item ->
                DishCard(
                    name = item.displayName(lang),
                    price = money(item.price),
                    // ⚠️ 300px wide, never the original. The full-size photograph
                    // of a plate is measured in megabytes; on a restaurant's
                    // connection that is the difference between a menu that opens
                    // and one somebody stops using.
                    imageUrl = imageUrl(item.imageUrl, uploadsBase, 300),
                    count = onCheck[item.id] ?: 0,
                    view = view,
                    soldOut = item.id in soldOut || item.soldOut || !item.isAvailable,
                    onAdd = { onAdd(item) },
                    onRemove = { onRemove(item) },
                )
            }
            if (pool.isEmpty()) {
                item(span = { GridItemSpan(maxLineSpan) }) {
                    Text(
                        if (searching) t.menu.nothingFound else t.check.noItems,
                        color = c.muted, textAlign = TextAlign.Center,
                        modifier = Modifier.fillMaxWidth().padding(24.dp),
                    )
                }
            }
        }
    }
}

/** A category, and how many of its dishes are on the check. */
@Composable
private fun CategoryChip(label: String, count: Int, on: Boolean, onClick: () -> Unit) {
    val c = KeelTheme.colors
    Row(
        Modifier
            .clip(CircleShape)
            .then(if (on) Modifier.background(c.accentSoft, CircleShape) else Modifier.glass(c, CircleShape))
            .border(1.dp, if (on) c.accent.copy(alpha = 0.55f) else c.glassBorder, CircleShape)
            .clickable(onClick = onClick)
            .padding(start = 14.dp, end = if (count > 0) 6.dp else 14.dp, top = 8.dp, bottom = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Text(
            label,
            color = if (on) c.ink else c.inkSoft,
            style = MaterialTheme.typography.bodyMedium.copy(
                fontWeight = if (on) FontWeight.SemiBold else FontWeight.Normal,
            ),
            maxLines = 1,
        )
        if (count > 0) {
            Box(
                Modifier.widthIn(min = 20.dp).height(20.dp).clip(CircleShape).background(c.accent)
                    .padding(horizontal = 6.dp),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    "$count", color = c.onAccent,
                    style = MaterialTheme.typography.labelSmall.merge(MoneyStyle),
                )
            }
        }
    }
}
