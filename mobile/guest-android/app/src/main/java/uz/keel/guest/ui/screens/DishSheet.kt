package uz.keel.guest.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import coil3.compose.AsyncImage
import uz.keel.design.Chip
import uz.keel.design.GhostButton
import uz.keel.design.GlassField
import uz.keel.design.GlassStepper
import uz.keel.design.KeelTheme
import uz.keel.design.PrimaryButton
import uz.keel.design.glassSheet
import uz.keel.design.imageUrl
import uz.keel.design.money
import uz.keel.guest.Brand
import uz.keel.guest.CartLine
import uz.keel.guest.cartLineOf
import uz.keel.guest.data.MenuItem
import uz.keel.guest.data.OptionChoice
import uz.keel.guest.data.pick
import uz.keel.guest.t

// One dish, and the questions the kitchen needs answered before it can cook it.
//
// ⚠️ **A required group has no default, and that is not an oversight.** Picking
// one for the guest sends a pizza in a size nobody chose — the order arrives, the
// kitchen cooks it, and the complaint comes at the door. So the button stays
// disabled and says which question is unanswered, rather than being silently
// unpressable.
//
// ⚠️ **The price on the button moves with the answers.** An option that adds
// 12 000 so'm and a total that only appears in the basket is how a guest finds
// out at checkout; the figure they tapped should be the figure they see.

@Composable
fun DishSheet(item: MenuItem, lang: String, onClose: () -> Unit, onAdd: (CartLine) -> Unit) {
    val c = KeelTheme.colors
    var qty by remember { mutableIntStateOf(1) }
    var comment by remember { mutableStateOf("") }
    /** Which answers are ticked, per question. */
    val chosen = remember { mutableStateMapOf<String, List<OptionChoice>>() }

    val groups = item.options
    val missing = groups.firstOrNull { it.required && chosen[it.name].isNullOrEmpty() }
    val extra = chosen.values.flatten().sumOf { it.priceDelta }
    val each = item.price + extra

    Dialog(onDismissRequest = onClose) {
        Column(
            Modifier
                .heightIn(max = 620.dp)
                .glassSheet(c, RoundedCornerShape(26.dp))
                .padding(16.dp)
                .imePadding(),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Column(
                Modifier.weight(1f, fill = false).verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                if (item.imageUrl.isNotEmpty()) {
                    AsyncImage(
                        // Wider than the menu's thumbnail, and still asked for
                        // at a width: this is one photograph, not forty.
                        model = imageUrl(item.imageUrl, Brand.uploadsBase, 600),
                        contentDescription = null,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier
                            .fillMaxWidth()
                            .aspectRatio(16f / 10f)
                            .clip(RoundedCornerShape(18.dp)),
                    )
                }
                Text(
                    item.pick(lang),
                    style = MaterialTheme.typography.headlineMedium,
                    color = c.ink,
                )
                val about = item.describe(lang)
                if (about.isNotEmpty()) {
                    Text(about, style = MaterialTheme.typography.bodyMedium, color = c.muted)
                }

                groups.forEach { group ->
                    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text(
                            group.pick(lang),
                            style = MaterialTheme.typography.titleMedium,
                            color = c.ink,
                        )
                        Text(
                            if (group.multiple) t.dish.chooseMany else t.dish.choose,
                            style = MaterialTheme.typography.labelMedium,
                            color = c.muted,
                        )
                        Row(
                            Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            // ⚠️ Wrapped by the row rather than a grid: the
                            // choices are two or three short words, and a grid
                            // gives "Katta" its own line beside empty space.
                            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                                group.choices.chunked(2).forEach { pair ->
                                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                        pair.forEach { choice ->
                                            val on = chosen[group.name]?.contains(choice) == true
                                            val label = choice.pick(lang) +
                                                if (choice.priceDelta != 0.0) {
                                                    "  +" + money(choice.priceDelta)
                                                } else {
                                                    ""
                                                }
                                            Chip(label, on) {
                                                val cur = chosen[group.name].orEmpty()
                                                chosen[group.name] = when {
                                                    // ⚠️ A single-answer group
                                                    // replaces rather than
                                                    // toggles: two sizes ticked
                                                    // is an order the kitchen
                                                    // cannot read.
                                                    !group.multiple -> listOf(choice)
                                                    on -> cur - choice
                                                    else -> cur + choice
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }

                GlassField(comment, { comment = it }, t.dish.commentHint)
            }

            Row(
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                // ⚠️ `removeAtZero = false`: minus at one refuses rather than
                // removing. There is nothing to remove — the dish is not in the
                // basket yet, and a sheet that closed itself on a stray tap
                // would lose the answers somebody just gave.
                GlassStepper(
                    value = qty,
                    removeAtZero = false,
                    onMinus = { qty = (qty - 1).coerceAtLeast(1) },
                    onPlus = { qty += 1 },
                )
                Box(Modifier.weight(1f)) {
                    PrimaryButton(
                        // ⚠️ The figure on the button is what this costs with
                        // the answers given — see the file's note.
                        label = t.dish.add + "  ·  " + money(each * qty),
                        enabled = missing == null,
                    ) {
                        onAdd(
                            cartLineOf(
                                item = item,
                                qty = qty,
                                comment = comment.trim(),
                                chosen = groups.associateWith { chosen[it.name].orEmpty() }
                                    .filterValues { it.isNotEmpty() },
                            ),
                        )
                        onClose()
                    }
                }
            }
            // ⚠️ **The reason is written out, not implied by a dead button.** A
            // guest who cannot see why the button does nothing concludes the app
            // is broken, and they are not wrong to.
            if (missing != null) {
                Text(
                    t.dish.needsChoice(missing.pick(lang)),
                    style = MaterialTheme.typography.labelMedium,
                    color = c.warn,
                )
            }
            GhostButton(t.common.close, Modifier.fillMaxWidth()) { onClose() }
        }
    }
}
