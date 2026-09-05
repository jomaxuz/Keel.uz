package uz.keel.waiter.ui.components

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.spring
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.animation.scaleOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.ImageNotSupported
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import uz.keel.waiter.ui.theme.KeelTheme
import uz.keel.waiter.ui.theme.MoneyStyle
import uz.keel.waiter.ui.theme.glass
import uz.keel.waiter.ui.theme.keelGradient
import uz.keel.waiter.ui.theme.softShadow

/** How the menu is drawn. ⚠️ Three, because one answer does not fit two
 *  restaurants: a café with forty drinks and no photographs wants a list it can
 *  scan, a photographed menu wants the picture, and a waiter who knows the menu
 *  by heart wants the tightest rows. Remembered per phone — it is a preference
 *  of the person holding it, not of the restaurant. */
enum class MenuView { List, Cards, Photos }

/** One dish.
 *
 *  ⚠️ **The count is the point of this card, not the picture.** Tapping a tile
 *  four times is how a waiter enters four coffees, and with nothing counting back
 *  the only way to know the third tap landed is to switch tabs and read the
 *  check — which is the moment somebody taps again and the guest is charged for
 *  five. So the plus becomes a stepper the instant there is one on the check,
 *  and the number animates so the change is *seen*.
 *
 *  ⚠️ **The card stops being tappable once it carries a stepper.** Two ways to
 *  add one dish — the card and the plus — differ by a few pixels and by one, and
 *  the difference is only discovered at the table. */
@Composable
fun DishCard(
    name: String,
    price: String,
    imageUrl: String?,
    count: Int,
    view: MenuView,
    soldOut: Boolean,
    onAdd: () -> Unit,
    onRemove: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val c = KeelTheme.colors
    val shape = RoundedCornerShape(if (view == MenuView.List) 18.dp else 22.dp)
    val added = count > 0

    Box(
        modifier
            .then(if (added && !soldOut) Modifier else Modifier.clickable(enabled = !soldOut, onClick = onAdd))
            .softShadow(shape, elevation = 2.dp, dark = c.dark)
            .glass(c, shape)
            // ⚠️ The added state is said by the edge, not by a tint over the
            // photograph: a wash across a plate is a plate nobody can read.
            .then(
                if (added) Modifier.background(
                    Brush.verticalGradient(listOf(c.accentSoft, Color.Transparent)), shape,
                ) else Modifier,
            ),
    ) {
        when (view) {
            MenuView.List -> ListRow(name, price, count, soldOut, onAdd, onRemove)
            MenuView.Cards -> CardBody(name, price, null, count, soldOut, onAdd, onRemove)
// ⚠️ A dish with no photograph still gets a **named gap** in the
            // photo view, drawn by `CardBody` — a restaurant that has
            // photographed half its menu must not have the other half look
            // broken.
            MenuView.Photos -> CardBody(name, price, imageUrl, count, soldOut, onAdd, onRemove, photoView = true)
        }
    }
}

@Composable
private fun ListRow(
    name: String, price: String, count: Int, soldOut: Boolean,
    onAdd: () -> Unit, onRemove: () -> Unit,
) {
    val c = KeelTheme.colors
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Column(Modifier.weight(1f)) {
            Text(
                name,
                style = MaterialTheme.typography.bodyLarge,
                color = if (soldOut) c.muted else c.ink,
                maxLines = 2, overflow = TextOverflow.Ellipsis,
            )
        }
        Text(price, style = MaterialTheme.typography.bodyMedium.merge(MoneyStyle), color = c.inkSoft)
        AddControl(count, soldOut, onAdd, onRemove, compact = false)
    }
}

@Composable
private fun CardBody(
    name: String, price: String, imageUrl: String?, count: Int, soldOut: Boolean,
    onAdd: () -> Unit, onRemove: () -> Unit, photoView: Boolean = false,
) {
    val c = KeelTheme.colors
    Column {
        if (photoView) {
            Box(Modifier.fillMaxWidth().aspectRatio(1.45f)) {
                // ⚠️ Never the original. `imageUrl(path, 300)` — the server
                // resizes on request, and a full-size photograph of a plate is
                // measured in megabytes: on a restaurant's connection that is
                // the difference between a menu that opens and one nobody uses.
                if (imageUrl != null) {
                    AsyncImage(
                        model = imageUrl,
                        contentDescription = null,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier.fillMaxSize(),
                    )
                } else {
                    Box(
                        Modifier.fillMaxSize().background(c.glassStrong),
                        contentAlignment = Alignment.Center,
                    ) {
                        Icon(
                            Icons.Rounded.ImageNotSupported, contentDescription = null,
                            tint = c.muted, modifier = Modifier.size(22.dp),
                        )
                    }
                }
                // The scrim under the price pill — a white pill on a pale plate
                // is a pill nobody sees.
                Box(
                    Modifier.fillMaxSize().background(
                        Brush.verticalGradient(
                            0.55f to Color.Transparent,
                            1f to Color.Black.copy(alpha = 0.45f),
                        ),
                    ),
                )
                Text(
                    price,
                    style = MaterialTheme.typography.labelMedium.merge(MoneyStyle),
                    color = Color.White,
                    modifier = Modifier
                        .align(Alignment.BottomStart)
                        .padding(10.dp)
                        .clip(CircleShape)
                        .background(Color.Black.copy(alpha = 0.35f))
                        .padding(horizontal = 10.dp, vertical = 5.dp),
                )
            }
        }
        Row(
            Modifier.fillMaxWidth().padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(
                    name,
                    style = MaterialTheme.typography.bodyMedium,
                    color = if (soldOut) c.muted else c.ink,
                    maxLines = 2, overflow = TextOverflow.Ellipsis,
                )
                // ⚠️ With no photograph the price has no pill to live in, so it
                // goes under the name. A card that shows a price in two places
                // depending on the picture is two cards.
                if (!photoView) {
                    Text(price, style = MaterialTheme.typography.labelMedium.merge(MoneyStyle), color = c.inkSoft)
                }
            }
            AddControl(count, soldOut, onAdd, onRemove, compact = true)
        }
    }
}

/** Plus, or a stepper once there is one on the check.
 *
 *  ⚠️ **44dp targets, and a larger one on the compact form.** This is pressed
 *  with a thumb, walking, with a tray in the other hand — and the two buttons
 *  sit next to each other, so a small target is not merely hard to hit, it is
 *  easy to hit *the wrong one*. */
@Composable
private fun AddControl(
    count: Int, soldOut: Boolean, onAdd: () -> Unit, onRemove: () -> Unit, compact: Boolean,
) {
    val c = KeelTheme.colors
    if (soldOut) return
    AnimatedContent(
        targetState = count > 0,
        transitionSpec = {
            (fadeIn(spring()) + scaleIn(spring(dampingRatio = Spring.DampingRatioMediumBouncy), 0.8f))
                .togetherWith(fadeOut(spring()) + scaleOut(spring(), 0.8f))
        },
        label = "addControl",
    ) { added ->
        if (added) {
            GlassStepper(count, compact, onMinus = onRemove, onPlus = onAdd)
        } else {
            Box(
                Modifier
                    .size(if (compact) 34.dp else 40.dp)
                    .clip(CircleShape)
                    .background(keelGradient())
                    .clickable(onClick = onAdd),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    Icons.Rounded.Add, contentDescription = null,
                    tint = c.onAccent, modifier = Modifier.size(if (compact) 18.dp else 20.dp),
                )
            }
        }
    }
}
