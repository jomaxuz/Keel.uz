package uz.keel.guest.ui.screens

import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.isImeVisible
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.runtime.getValue
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
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
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Divider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import uz.keel.design.GhostButton
import uz.keel.design.GlassField
import uz.keel.design.GlassStepper
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Close
import uz.keel.design.GlassIconButton
import uz.keel.design.KeelTheme
import uz.keel.design.Money
import uz.keel.design.PrimaryButton
import uz.keel.design.glass
import uz.keel.design.imageUrl
import uz.keel.guest.Brand
import uz.keel.guest.Cart
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

    // ⚠️ **The list ends where the summary begins — measured, not guessed.**
    // It used to reserve a fixed 96 dp, and the summary is more than twice that
    // (three rows, a button, a second button). So the last dish's note field
    // sat permanently under "Rasmiylashtirish" and could not be tapped, however
    // far one scrolled. The summary reports its own height and the list pads by
    // exactly that, so a new row in the summary cannot bring the bug back.
    val density = LocalDensity.current
    var summaryHeight by remember { mutableStateOf(0.dp) }
    // ⚠️ **Hidden while the keyboard is up.** Somebody typing "piyozsiz" is
    // looking at the field, and a summary pinned above the keyboard would take
    // half of what is left of the screen and cover the very field being typed in.
    val typing = WindowInsets.isImeVisible

    Box(Modifier.fillMaxSize()) {
        LazyColumn(
            Modifier.fillMaxSize().imePadding(),
            contentPadding = PaddingValues(
                start = 16.dp,
                end = 16.dp,
                bottom = if (typing) 16.dp
                else bottomInset.calculateBottomPadding() + 10.dp + summaryHeight + 12.dp,
            ),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                Row(
                    Modifier.statusBarsPadding().padding(top = 12.dp).fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        t.cart.title,
                        Modifier.weight(1f),
                        style = MaterialTheme.typography.headlineMedium,
                        color = c.ink,
                    )
                    GhostButton(t.cart.clear) { cart.clear() }
                }
            }

            // ⚠️ **Keyed on `lineId`, not on `key`.** The note below is part of
            // the key, so keying on it would tear down and rebuild the text
            // field after every character — and take the cursor with it.
            items(cart.lines, key = { it.lineId }) { line ->
                Column(
                    Modifier
                        .fillMaxWidth()
                        .glass(c, RoundedCornerShape(18.dp))
                        .padding(10.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Row(
                        Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        if (line.imageUrl.isNotEmpty()) {
                            AsyncImage(
                                model = imageUrl(line.imageUrl, Brand.uploadsBase, 200),
                                contentDescription = null,
                                contentScale = ContentScale.Crop,
                                modifier = Modifier
                                    .width(60.dp)
                                    .aspectRatio(1f)
                                    .clip(RoundedCornerShape(12.dp)),
                            )
                        }
                        Column(Modifier.weight(1f)) {
                            Text(
                                line.name,
                                style = MaterialTheme.typography.titleMedium,
                                color = c.ink,
                                maxLines = 2,
                                overflow = TextOverflow.Ellipsis,
                            )
                            // ⚠️ **The answers are printed on the line.** Two
                            // plovs that differ only by an option are two lines,
                            // and a basket showing both as "Osh" would look like
                            // a duplicate somebody tries to delete.
                            val detail = line.options.joinToString(" · ") { it.choice }
                            if (detail.isNotEmpty()) {
                                Text(
                                    detail,
                                    style = MaterialTheme.typography.labelMedium,
                                    color = c.muted,
                                    maxLines = 2,
                                    overflow = TextOverflow.Ellipsis,
                                )
                            }
                            // ⚠️ **The unit price, as on the site.** With only a
                            // line total on screen, "58 000" beside a quantity
                            // of two reads as the price of one dish, and the
                            // basket looks wrong rather than the reader.
                            Money(
                                line.price + line.options.sumOf { it.priceDelta },
                                style = MaterialTheme.typography.bodyMedium,
                                color = c.muted,
                            )
                        }
                        // ⚠️ Its own control, as on the site: minus-to-zero
                        // works, but somebody who has decided against a dish
                        // looks for a way to delete it, and the one they find
                        // otherwise is "clear the basket".
                        GlassIconButton(Icons.Rounded.Close) { cart.remove(line.lineId) }
                    }

                    Row(
                        Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        // ⚠️ `removeAtZero`: minus at one deletes the line. A
                        // stepper that stops at one leaves somebody hunting for
                        // a delete button, and the one they find empties the
                        // basket.
                        GlassStepper(
                            value = line.qty,
                            compact = true,
                            onMinus = { cart.setQty(line.lineId, line.qty - 1) },
                            onPlus = { cart.setQty(line.lineId, line.qty + 1) },
                        )
                        Box(Modifier.weight(1f).padding(start = 12.dp)) {
                            Money(
                                line.lineTotal,
                                Modifier.align(Alignment.CenterEnd),
                                color = c.ink,
                                style = MaterialTheme.typography.titleMedium,
                            )
                        }
                    }

                    // ⚠️ **Written here rather than only on the dish sheet.** A
                    // guest remembers "no onion" while looking at the basket,
                    // not while choosing a size — and on the site this field is
                    // right here, so an app without it is the app that lost it.
                    GlassField(
                        line.comment,
                        { cart.setComment(line.lineId, it.take(200)) },
                        t.cart.itemComment,
                    )
                }
            }
        }

        // ---- The summary, as on the site ----
        //
        // ⚠️ **Above the list rather than at the end of it.** A guest with
        // fifteen lines should not scroll to find out what they owe — and the
        // figure is the reason they opened this screen.
        if (!typing) Column(
            Modifier
                .align(Alignment.BottomCenter)
                .padding(
                    start = 16.dp,
                    end = 16.dp,
                    bottom = bottomInset.calculateBottomPadding() + 10.dp,
                )
                .fillMaxWidth()
                .onSizeChanged { summaryHeight = with(density) { it.height.toDp() } }
                .glass(c, RoundedCornerShape(20.dp))
                .padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    t.cart.items(cart.count),
                    Modifier.weight(1f),
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.muted,
                )
                Money(cart.subtotal, style = MaterialTheme.typography.bodyMedium, color = c.ink)
            }
            // ⚠️ **Named and deferred, never shown as zero.** The fee depends on
            // an address nobody has given yet and on zones the owner drew; a
            // basket printing "0" would be quoting a price the server has not
            // calculated, and the guest would read it as free delivery.
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    t.cart.delivery,
                    Modifier.weight(1f),
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.muted,
                )
                Text(
                    t.cart.atCheckout,
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.muted,
                )
            }
            Divider(color = c.line)
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    t.cart.total,
                    Modifier.weight(1f),
                    style = MaterialTheme.typography.titleMedium,
                    color = c.ink,
                )
                Money(cart.subtotal, style = MaterialTheme.typography.titleMedium, color = c.ink)
            }
            PrimaryButton(t.cart.checkout, Modifier.fillMaxWidth()) { onCheckout() }
            GhostButton(t.cart.keepShopping, Modifier.fillMaxWidth()) { onKeepShopping() }
        }
    }
}
