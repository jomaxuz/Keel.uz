package uz.keel.owner.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.ArrowBackIosNew
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import uz.keel.design.GlassIconButton
import uz.keel.design.KeelTheme
import uz.keel.design.MoneyStyle
import uz.keel.design.ScreenHeader
import uz.keel.design.glass
import uz.keel.design.money
import uz.keel.owner.data.ApiError
import uz.keel.owner.data.CheckDetail
import uz.keel.owner.data.CheckLine
import uz.keel.owner.data.KeelApi
import uz.keel.owner.t
import java.time.LocalDate
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.ZonedDateTime

// One sale, opened from the alert it caused — "why was 400 000 taken off table
// six", answered on the phone the alert arrived on.
//
// ⚠️ **Voided lines stay where they were, crossed out.** A void that leaves no
// trace is the oldest way to take money out of a restaurant; a detail view that
// showed only the live lines would agree with a dishonest check and disagree
// with the kitchen. The server sends them for exactly this reason.
//
// ⚠️ **Reading only.** Refund and reprint live in the panel: both move money or
// paper in a building the owner is usually not standing in, and a button for
// either on the screen somebody opens from a notification is the button pressed
// by accident in a taxi.

/** An instant off the wire, in the phone's own zone.
 *
 *  ⚠️ **`OffsetDateTime`, not `Instant.parse`.** This endpoint moves its times
 *  into the restaurant's zone before sending (`"…T19:42:10+05:00"`), while
 *  anything fresh out of Mongo arrives as `Z`. `Instant.parse` on the older
 *  Android runtimes accepts only the second — and a time that fails to parse is
 *  a blank, not an error. */
internal fun localTime(iso: String?): ZonedDateTime? =
    iso?.takeIf { it.isNotBlank() }?.let {
        runCatching { OffsetDateTime.parse(it).atZoneSameInstant(ZoneId.systemDefault()) }.getOrNull()
    }

/** `19:42` today, `12.09 19:42` any other day. */
internal fun clock(iso: String?): String {
    val z = localTime(iso) ?: return ""
    val hm = "%02d:%02d".format(z.hour, z.minute)
    return if (z.toLocalDate() == LocalDate.now()) hm
    else "%02d.%02d %s".format(z.dayOfMonth, z.monthValue, hm)
}

@Composable
fun CheckScreen(api: KeelApi, id: String, bottomInset: PaddingValues, onBack: () -> Unit) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()
    var check by remember { mutableStateOf<CheckDetail?>(null) }
    var error by remember { mutableStateOf("") }
    val loadFailed = t.common.loadFailed
    val notFound = t.check.notFound

    suspend fun load() {
        try {
            check = api.check(id)
            error = ""
        } catch (e: Throwable) {
            // ⚠️ The server's 404 is English ("check not found") and also means
            // "outside what you may see" — said here in the owner's language.
            error = when {
                e is ApiError && e.status == 404 -> notFound
                e is ApiError -> e.message
                else -> loadFailed
            }
        }
    }
    LaunchedEffect(id) { load() }

    val k = check
    Column(Modifier.fillMaxSize()) {
        ScreenHeader(
            title = if (k != null && k.number.isNotEmpty()) t.check.title(k.number) else t.check.titleEmpty,
            subtitle = k?.let {
                listOfNotNull(
                    it.table.takeIf { s -> s.isNotEmpty() }?.let(t.check.table),
                    it.guests.takeIf { n -> n > 0 }?.let(t.check.guests),
                ).joinToString(" · ").takeIf { s -> s.isNotEmpty() }
            },
            leading = { GlassIconButton(Icons.Rounded.ArrowBackIosNew, onClick = onBack) },
            trailing = { GlassIconButton(Icons.Rounded.Refresh) { scope.launch { load() } } },
        )
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp)
                .padding(bottom = bottomInset.calculateBottomPadding() + 24.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            if (k == null && error.isEmpty()) {
                Box(Modifier.fillMaxWidth().padding(40.dp), Alignment.Center) {
                    CircularProgressIndicator(color = c.accent)
                }
            }
            if (error.isNotEmpty()) {
                Text(error, color = c.danger, style = MaterialTheme.typography.bodyMedium)
            }
            if (k != null) CheckBody(k)
        }
    }
}

@Composable
private fun CheckBody(k: CheckDetail) {
    val c = KeelTheme.colors

    // What state the sale is in, first: a cancelled or refunded check read as an
    // ordinary one is the misreading every figure below would then inherit.
    val (state, tint) = when {
        k.cancelled -> t.check.cancelled to c.danger
        k.refunded -> t.check.refunded to c.warn
        k.open -> t.check.open to c.accent
        else -> t.check.closedAt(clock(k.closedAt)) to c.muted
    }
    Text(
        listOfNotNull(state, t.check.split.takeIf { k.split }).joinToString(" · "),
        style = MaterialTheme.typography.labelLarge, color = tint,
    )

    GlassCard {
        if (k.lines.isEmpty()) {
            Text(t.check.noLines, style = MaterialTheme.typography.bodyMedium, color = c.muted)
        }
        k.lines.forEach { LineRow(it, k.precheckAt) }
    }

    GlassCard {
        FigureLine(t.check.subtotal, money(k.subtotal))
        k.discounts.forEach { d ->
            Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                FigureLine(d.name.ifEmpty { t.check.discount }, "−" + money(d.amount), tint = c.warn)
                // ⚠️ Who applied it and who allowed it — the two names are the
                // content, and they are often not the same person.
                val who = listOfNotNull(
                    d.by.takeIf { it.isNotEmpty() }?.let(t.check.discountBy),
                    d.authBy.takeIf { it.isNotEmpty() }?.let(t.check.approvedBy),
                ).joinToString(" · ")
                if (who.isNotEmpty()) Muted(who)
                if (d.reason.isNotBlank()) Soft(d.reason)
            }
        }
        // An older check carries the total with no breakdown behind it.
        if (k.discounts.isEmpty() && k.discount > 0) {
            FigureLine(t.check.discount, "−" + money(k.discount), tint = c.warn)
        }
        if (k.service > 0) FigureLine(t.check.service(k.servicePercent), money(k.service))
        FigureLine(t.check.total, money(k.total), strong = true)
        if (k.paymentMethod.isNotEmpty()) {
            FigureLine(t.check.payment, t.check.methods[k.paymentMethod] ?: k.paymentMethod)
        }
    }

    k.refund?.let { r ->
        GlassCard {
            FigureLine(t.check.refundTitle, money(r.amount), strong = true, tint = c.warn)
            val who = listOfNotNull(r.by.takeIf { it.isNotEmpty() }, clock(r.at).takeIf { it.isNotEmpty() })
                .joinToString(" · ")
            if (who.isNotEmpty()) Muted(who)
            if (r.reason.isNotBlank()) Soft(r.reason)
        }
    }

    val people = listOfNotNull(
        k.openedBy.takeIf { it.isNotEmpty() }?.let { t.check.openedBy(it, clock(k.openedAt)) },
        k.server.takeIf { it.isNotEmpty() && it != k.openedBy }?.let(t.check.server),
        k.precheckAt?.let { t.check.precheck(clock(it)) },
        k.closedBy.takeIf { it.isNotEmpty() }?.let(t.check.closedBy),
    )
    if (people.isNotEmpty()) {
        GlassCard { people.forEach { Soft(it) } }
    }

    if (k.fiscalError.isNotBlank()) {
        GlassCard {
            Text(t.check.fiscalError, style = MaterialTheme.typography.labelLarge, color = c.danger)
            Soft(k.fiscalError)
        }
    } else if (k.fiscalSign.isNotBlank()) {
        Muted(t.check.fiscalSign(k.fiscalSign))
    }
}

@Composable
private fun LineRow(l: CheckLine, precheckAt: String?) {
    val c = KeelTheme.colors
    val voided = l.voided
    val decoration = if (voided) TextDecoration.LineThrough else null
    Column(Modifier.padding(vertical = 4.dp), verticalArrangement = Arrangement.spacedBy(2.dp)) {
        Row(verticalAlignment = Alignment.Top) {
            Text(
                "${l.qty} × ${l.name}", Modifier.weight(1f),
                style = MaterialTheme.typography.bodyLarge,
                color = if (voided) c.muted else c.ink, textDecoration = decoration,
            )
            // ⚠️ A voided line's `sum` is zero by design; the crossed-out figure
            // is what it would have been, which is the amount the owner asks about.
            Text(
                money(if (voided) l.price * l.qty else l.sum),
                style = MaterialTheme.typography.bodyLarge.merge(MoneyStyle),
                color = if (voided) c.muted else c.ink, textDecoration = decoration,
            )
        }
        val options = l.options.map { it.choice.ifEmpty { it.name } }.filter { it.isNotEmpty() }
        if (options.isNotEmpty()) Muted(options.joinToString(", "))
        if (l.comment.isNotBlank()) Muted(l.comment)
        if (voided) {
            Text(
                listOfNotNull(
                    t.check.voided(l.voidedBy),
                    clock(l.voidedAt).takeIf { it.isNotEmpty() },
                ).joinToString(" · "),
                style = MaterialTheme.typography.labelMedium, color = c.danger,
            )
            // ⚠️ After the bill was shown is the stronger fact: a table that
            // changed its mind versus a total that existed and then did not.
            val after = localTime(l.voidedAt)?.let { v -> localTime(precheckAt)?.let { v.isAfter(it) } } == true
            if (after) {
                Text(t.check.voidAfterBill, style = MaterialTheme.typography.labelMedium, color = c.warn)
            }
            if (l.voidReason.isNotBlank()) Soft(l.voidReason)
            if (l.wasted) Muted(t.check.wasted)
        }
    }
}

// ---- Shared by the pushed screens ----

@Composable
internal fun GlassCard(content: @Composable () -> Unit) {
    val c = KeelTheme.colors
    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) { content() }
}

@Composable
internal fun FigureLine(label: String, value: String, strong: Boolean = false, tint: Color? = null) {
    val c = KeelTheme.colors
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(
            label, Modifier.weight(1f),
            style = MaterialTheme.typography.bodyMedium,
            color = if (strong) c.ink else c.inkSoft,
        )
        Text(
            value,
            style = (if (strong) MaterialTheme.typography.titleMedium
            else MaterialTheme.typography.bodyMedium).merge(MoneyStyle),
            color = tint ?: c.ink,
        )
    }
}

@Composable
internal fun Muted(text: String, color: Color? = null) {
    Text(text, style = MaterialTheme.typography.labelMedium, color = color ?: KeelTheme.colors.muted)
}

@Composable
private fun Soft(text: String) {
    Text(text, style = MaterialTheme.typography.bodyMedium, color = KeelTheme.colors.inkSoft)
}
