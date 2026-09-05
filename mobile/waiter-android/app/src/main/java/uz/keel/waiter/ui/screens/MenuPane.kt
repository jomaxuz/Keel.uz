package uz.keel.waiter.ui.screens

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
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
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
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import uz.keel.waiter.data.MenuGroup
import uz.keel.waiter.data.MenuItem
import uz.keel.waiter.data.displayName
import uz.keel.waiter.data.imageUrl
import uz.keel.waiter.data.money
import uz.keel.waiter.t
import uz.keel.waiter.ui.components.Chip
import uz.keel.waiter.ui.components.DishCard
import uz.keel.waiter.ui.components.GlassField
import uz.keel.waiter.ui.components.GlassIconButton
import uz.keel.waiter.ui.components.MenuView
import uz.keel.waiter.ui.theme.KeelTheme

// The menu, as a waiter reads it.
//
// ⚠️ **The names are the restaurant's own text, and they have three versions.**
// `contentName` is the site's rule, ported unchanged: Uzbek is the base and a
// missing translation falls back to it, which is what stops a half-translated
// menu from showing empty rows.

@Composable
fun MenuPane(
    groups: List<MenuGroup>,
    /** How many of each dish are already on the check. */
    onCheck: Map<String, Int>,
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
) {
    val c = KeelTheme.colors
    var category by remember { mutableIntStateOf(0) }
    var query by remember { mutableStateOf("") }
    var onlyAdded by remember { mutableStateOf(false) }

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
     *  The base is matched too, so a name with no translation is still reachable. */
    val items = remember(groups, category, query, onlyAdded, onCheck, lang) {
        val q = query.trim().lowercase()
        val pool = if (searching) groups.flatMap { it.items }
        else groups.getOrNull(category)?.items ?: emptyList()
        pool.filter { it ->
            if (onlyAdded && (onCheck[it.id] ?: 0) == 0) return@filter false
            if (q.isEmpty()) return@filter true
            it.displayName(lang).lowercase().contains(q) || it.name.lowercase().contains(q)
        }
    }

    val views = listOf(
        MenuView.List to Icons.Rounded.ViewList,
        MenuView.Cards to Icons.Rounded.GridView,
        MenuView.Photos to Icons.Rounded.Image,
    )

    Column(Modifier.fillMaxSize()) {
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(Modifier.weight(1f)) {
                GlassField(
                    value = query, onValueChange = { query = it },
                    placeholder = t.menu.search,
                    keyboardOptions = KeyboardOptions(
                        capitalization = KeyboardCapitalization.None, autoCorrectEnabled = false,
                    ),
                    trailing = {
                        if (searching) GlassIconButton(Icons.Rounded.Close) { query = "" }
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
                Modifier.fillMaxWidth().height(52.dp),
                contentPadding = PaddingValues(horizontal = 16.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                items(groups.size) { i ->
                    val g = groups[i]
                    Chip(g.category.displayName(lang), i == category) { category = i }
                }
            }
        }

        Box(
            Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
            contentAlignment = Alignment.CenterStart,
        ) {
            if (searching) {
                Text(t.menu.found(items.size), color = c.muted, style = MaterialTheme.typography.labelMedium)
            } else {
                Chip(t.menu.onCheck, onlyAdded, icon = Icons.Rounded.Check) { onlyAdded = !onlyAdded }
            }
        }

        // ⚠️ **A lazy grid, never a Column of everything.** A menu of two hundred
        // dishes with photographs composed all at once is what makes a cheap
        // Android stutter — the same reason the Expo build moved off ScrollView.
        // One column in list view, adaptive in the two card views.
        LazyVerticalGrid(
            columns = if (view == MenuView.List) GridCells.Fixed(1) else GridCells.Adaptive(160.dp),
            contentPadding = PaddingValues(16.dp, 4.dp, 16.dp, bottomPad),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
            modifier = Modifier.fillMaxSize(),
        ) {
            items(items, key = { it.id }) { item ->
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
                    soldOut = item.soldOut || !item.isAvailable,
                    onAdd = { onAdd(item) },
                    onRemove = { onRemove(item) },
                )
            }
            if (items.isEmpty()) {
                item {
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
