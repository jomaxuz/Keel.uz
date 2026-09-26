package uz.keel.guest.ui.screens

import androidx.compose.animation.animateContentSize
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.isImeVisible
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.DeleteOutline
import androidx.compose.material.icons.rounded.EditNote
import androidx.compose.material.icons.rounded.RestaurantMenu
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import uz.keel.design.GlassField
import uz.keel.design.GlassStepper
import uz.keel.design.KeelTheme
import uz.keel.design.Money
import uz.keel.design.PrimaryButton
import uz.keel.design.glass
import uz.keel.design.imageUrl
import uz.keel.design.money
import uz.keel.guest.Brand
import uz.keel.guest.Cart
import uz.keel.guest.CartLine
import uz.keel.guest.t

// What the guest has chosen, before any of it is priced.
//
// ⚠️ **The totals here are the phone's arithmetic and the checkout's are the
// server's**, and the difference is deliberate rather than sloppy: this screen
// has to work with no network, and it is only ever the sum of what is on it.
// Every figure a guest is actually charged comes from `/orders/quote`.
//
// ⚠️ **Built to the same plan as the site's basket** (`(site)/cart/page.tsx`),
// because it is the same screen for the same person: one card holding the
// lines, one card holding the summary. A guest who ordered on the site last
// week should not have to learn where things are — and the two drifting apart
// is how "the app is worse" starts, one small difference at a time.

@OptIn(androidx.compose.foundation.layout.ExperimentalLayoutApi::class)
@Composable
fun CartScreen(
    cart: Cart,
    lang: String,
    signedIn: Boolean,
    bottomInset: PaddingValues,
    onSignIn: () -> Unit,
    onKeepShopping: () -> Unit,
    onCheckout: () -> Unit,
) {
    val c = KeelTheme.colors

    // ---- Signed out ----
    //
    // ⚠️ **The basket is not shown at all, rather than shown and refused at the
    // end.** Letting somebody fill it and meeting them with a sign-in form at
    // the checkout is the worst order of those two screens: the work is already
    // done and the demand arrives when they are closest to paying.
    //
    // ⚠️ **It says what the number is for.** "Sign in" with no reason reads as
    // a form standing between a hungry person and dinner; "so the restaurant
    // can ring you back" is a sentence somebody agrees with.
    if (!signedIn) {
        Column(
            Modifier.fillMaxSize().padding(32.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp, Alignment.CenterVertically),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                t.cart.signInTitle,
                style = MaterialTheme.typography.titleLarge,
                color = c.ink,
                textAlign = TextAlign.Center,
            )
            Text(
                t.cart.signInHint,
                style = MaterialTheme.typography.bodyMedium,
                color = c.muted,
                textAlign = TextAlign.Center,
            )
            PrimaryButton(t.cart.signInAction) { onSignIn() }
        }
        return
    }

    if (cart.lines.isEmpty()) {
        Column(
            Modifier.fillMaxSize().padding(32.dp),
            verticalArrangement = Arrangement.spacedBy(6.dp, Alignment.CenterVertically),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(t.cart.empty, style = MaterialTheme.typography.titleLarge, color = c.ink)
            // ⚠️ **An empty basket is told what to do, not merely reported.**
            // "Savat bo'sh" alone is a dead end on the one screen a guest
            // reaches by tapping a tab out of curiosity.
            Text(
                t.cart.emptyHint,
                style = MaterialTheme.typography.bodyMedium,
                color = c.muted,
                textAlign = TextAlign.Center,
            )
            PrimaryButton(t.cart.keepShopping) { onKeepShopping() }
        }
        return
    }

    // ⚠️ **The list ends where the checkout bar begins — measured, not
    // guessed.** A fixed pad once left the last dish's note permanently under
    // the button; the bar reports its own height and the list pads by exactly
    // that.
    val density = LocalDensity.current
    var barHeight by remember { mutableStateOf(0.dp) }
    // ⚠️ **Hidden while the keyboard is up.** Somebody typing "piyozsiz" is
    // looking at the field; a bar pinned above the keyboard would cover it.
    val typing = WindowInsets.isImeVisible
    // Which notes are open. ⚠️ **Closed by default**: a text field under every
    // dish made the basket a form, and a note is the exception, not the rule.
    // A line that already has one is shown open.
    var notesOpen by remember { mutableStateOf(setOf<String>()) }

    Box(Modifier.fillMaxSize()) {
        LazyColumn(
            Modifier.fillMaxSize().imePadding(),
            contentPadding = PaddingValues(
                start = 16.dp,
                end = 16.dp,
                bottom = if (typing) 16.dp
                else bottomInset.calculateBottomPadding() + 10.dp + barHeight + 12.dp,
            ),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                Row(
                    Modifier.statusBarsPadding().padding(top = 12.dp, bottom = 2.dp).fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Column(Modifier.weight(1f)) {
                        Text(t.cart.title, style = MaterialTheme.typography.headlineMedium, color = c.ink)
                        Text(t.cart.items(cart.count), style = MaterialTheme.typography.bodyMedium, color = c.muted)
                    }
                    // A quiet text action: emptying the basket is rare, and a
                    // big button beside the title invited it.
                    Row(
                        Modifier
                            .clip(RoundedCornerShape(50))
                            .clickable { cart.clear() }
                            .padding(horizontal = 10.dp, vertical = 8.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(4.dp),
                    ) {
                        Icon(Icons.Rounded.DeleteOutline, null, tint = c.muted, modifier = Modifier.size(18.dp))
                        Text(t.cart.clear, style = MaterialTheme.typography.labelLarge, color = c.muted)
                    }
                }
            }

            // ⚠️ **Keyed on `lineId`, not on `key`.** The note is part of the
            // key, so keying on it would rebuild the text field after every
            // character — and take the cursor with it.
            items(cart.lines, key = { it.lineId }) { line ->
                LineCard(
                    line = line,
                    noteOpen = line.comment.isNotEmpty() || line.lineId in notesOpen,
                    onOpenNote = { notesOpen = notesOpen + line.lineId },
                    onNote = { cart.setComment(line.lineId, it.take(200)) },
                    onMinus = { cart.setQty(line.lineId, line.qty - 1) },
                    onPlus = { cart.setQty(line.lineId, line.qty + 1) },
                    onRemove = { cart.remove(line.lineId) },
                )
            }
        }

        // ---- The checkout bar ----
        //
        // ⚠️ **One row: what they owe, and the button.** The old summary was
        // five rows and two full-width buttons — half the screen covering the
        // basket it summarised. "Back to the menu" is the tab bar's job.
        //
        // ⚠️ **The fee is named, never shown as zero**: it depends on an address
        // nobody has given yet (see Strings.Cart.atCheckout).
        if (!typing) Row(
            Modifier
                .align(Alignment.BottomCenter)
                .padding(
                    start = 16.dp,
                    end = 16.dp,
                    bottom = bottomInset.calculateBottomPadding() + 10.dp,
                )
                .fillMaxWidth()
                .onSizeChanged { barHeight = with(density) { it.height.toDp() } }
                .glass(c, RoundedCornerShape(22.dp))
                .padding(start = 16.dp, end = 8.dp, top = 8.dp, bottom = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Column(Modifier.weight(1f)) {
                Text(t.cart.total, style = MaterialTheme.typography.labelMedium, color = c.muted)
                Money(
                    cart.subtotal,
                    color = c.ink,
                    style = MaterialTheme.typography.titleLarge.copy(fontWeight = FontWeight.Bold),
                )
                Text(
                    t.cart.feeLater,
                    style = MaterialTheme.typography.labelSmall,
                    color = c.muted,
                    maxLines = 2,
                )
            }
            PrimaryButton(t.cart.checkout, Modifier.width(168.dp)) { onCheckout() }
        }
    }
}

/** One dish in the basket: picture, name, what was chosen, price and count on
 *  one line, and the note tucked away until somebody wants it. */
@Composable
private fun LineCard(
    line: CartLine,
    noteOpen: Boolean,
    onOpenNote: () -> Unit,
    onNote: (String) -> Unit,
    onMinus: () -> Unit,
    onPlus: () -> Unit,
    onRemove: () -> Unit,
) {
    val c = KeelTheme.colors
    val unit = line.price + line.options.sumOf { it.priceDelta }
    Column(
        Modifier
            .fillMaxWidth()
            .glass(c, RoundedCornerShape(20.dp))
            .animateContentSize()
            .padding(10.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            val pic = Modifier.size(84.dp).clip(RoundedCornerShape(16.dp))
            if (line.imageUrl.isNotEmpty()) {
                AsyncImage(
                    model = imageUrl(line.imageUrl, Brand.uploadsBase, 200),
                    contentDescription = null,
                    contentScale = ContentScale.Crop,
                    modifier = pic,
                )
            } else {
                Box(pic.background(c.accentSoft), contentAlignment = Alignment.Center) {
                    Icon(Icons.Rounded.RestaurantMenu, null, tint = c.accent, modifier = Modifier.size(28.dp))
                }
            }
            Column(Modifier.weight(1f).height(84.dp)) {
                Row(verticalAlignment = Alignment.Top) {
                    Text(
                        line.name,
                        Modifier.weight(1f),
                        style = MaterialTheme.typography.titleSmall,
                        color = c.ink,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                    // ⚠️ Its own control, as on the site: somebody who has
                    // decided against a dish looks for a way to delete it, and
                    // the one they would otherwise find is "clear the basket".
                    Icon(
                        Icons.Rounded.Close,
                        null,
                        tint = c.muted,
                        modifier = Modifier
                            .padding(start = 6.dp)
                            .size(26.dp)
                            .clip(CircleShape)
                            .clickable(onClick = onRemove)
                            .padding(4.dp),
                    )
                }
                // ⚠️ **The answers are printed on the line.** Two plovs that
                // differ only by an option are two lines, and both shown as
                // "Osh" look like a duplicate somebody tries to delete.
                val detail = line.options.joinToString(" · ") { it.choice }
                if (detail.isNotEmpty()) {
                    Text(
                        detail,
                        style = MaterialTheme.typography.labelSmall,
                        color = c.muted,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                Spacer(Modifier.weight(1f))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f)) {
                        Money(
                            line.lineTotal,
                            color = c.ink,
                            style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                        )
                        // ⚠️ **The unit price, when it differs from the line.**
                        // "58 000" beside a count of two otherwise reads as the
                        // price of one dish.
                        if (line.qty > 1) {
                            Text(
                                money(unit) + " × " + line.qty,
                                style = MaterialTheme.typography.labelSmall,
                                color = c.muted,
                            )
                        }
                    }
                    // `removeAtZero`: minus at one deletes the line.
                    GlassStepper(value = line.qty, compact = true, onMinus = onMinus, onPlus = onPlus)
                }
            }
        }

        // ⚠️ **Here rather than only on the dish sheet** — a guest remembers
        // "no onion" while looking at the basket — but folded into a link
        // until it is wanted.
        if (noteOpen) {
            GlassField(line.comment, onNote, t.cart.itemComment)
        } else {
            Row(
                Modifier
                    .clip(RoundedCornerShape(50))
                    .clickable(onClick = onOpenNote)
                    .padding(horizontal = 6.dp, vertical = 4.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(4.dp),
            ) {
                Icon(Icons.Rounded.EditNote, null, tint = c.accent, modifier = Modifier.size(18.dp))
                Text(t.cart.addComment, style = MaterialTheme.typography.labelMedium, color = c.accent)
            }
        }
    }
}
