package uz.keel.guest.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
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
import uz.keel.design.GlassStepper
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

@Composable
fun CartScreen(
    cart: Cart,
    lang: String,
    signedIn: Boolean,
    bottomInset: PaddingValues,
    onSignIn: () -> Unit,
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
            Text(t.cart.emptyHint, style = MaterialTheme.typography.bodyMedium, color = c.muted)
        }
        return
    }

    Box(Modifier.fillMaxSize()) {
        LazyColumn(
            Modifier.fillMaxSize(),
            contentPadding = PaddingValues(
                start = 16.dp,
                end = 16.dp,
                bottom = bottomInset.calculateBottomPadding() + 96.dp,
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
            items(cart.lines, key = { it.key }) { line ->
                Row(
                    Modifier
                        .fillMaxWidth()
                        .glass(c, RoundedCornerShape(18.dp))
                        .padding(10.dp),
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
                        // ⚠️ **The answers are printed on the line.** Two plovs
                        // that differ only by an option are two lines, and a
                        // basket that showed both as "Osh" would look like a
                        // duplicate somebody tries to delete.
                        val detail = buildList {
                            line.options.forEach { add(it.choice) }
                            if (line.comment.isNotEmpty()) add(line.comment)
                        }.joinToString(" · ")
                        if (detail.isNotEmpty()) {
                            Text(
                                detail,
                                style = MaterialTheme.typography.labelMedium,
                                color = c.muted,
                                maxLines = 2,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                        Money(line.lineTotal, color = c.ink)
                    }
                    // ⚠️ `removeAtZero`: minus at one deletes the line. A
                    // stepper that stops at one leaves somebody hunting for a
                    // delete button, and the one they find empties the basket.
                    GlassStepper(
                        value = line.qty,
                        compact = true,
                        onMinus = { cart.setQty(line.key, line.qty - 1) },
                        onPlus = { cart.setQty(line.key, line.qty + 1) },
                    )
                }
            }
        }

        // ⚠️ **The total and the button sit above the list, not at the end of
        // it.** A guest with fifteen lines should not scroll to find out what
        // they owe — and the figure is the reason they opened this screen.
        Column(
            Modifier
                .align(Alignment.BottomCenter)
                .padding(
                    start = 16.dp,
                    end = 16.dp,
                    bottom = bottomInset.calculateBottomPadding() + 10.dp,
                )
                .fillMaxWidth()
                .glass(c, RoundedCornerShape(20.dp))
                .padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    t.cart.subtotal,
                    Modifier.weight(1f),
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.muted,
                )
                Money(cart.subtotal, color = c.ink)
            }
            PrimaryButton(t.cart.checkout, Modifier.fillMaxWidth()) { onCheckout() }
        }
    }
}
