package uz.keel.owner.ui.screens

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
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
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import uz.keel.design.Chip
import uz.keel.design.GlassField
import uz.keel.design.GlassIconButton
import uz.keel.design.KeelTheme
import uz.keel.design.LocalNotice
import uz.keel.design.MoneyStyle
import uz.keel.design.Note
import uz.keel.design.NoticeKind
import uz.keel.design.PrimaryButton
import uz.keel.design.ScreenHeader
import uz.keel.design.glass
import uz.keel.design.money
import uz.keel.owner.LocalPrefs
import uz.keel.owner.data.ApiError
import uz.keel.owner.data.KeelApi
import uz.keel.owner.data.MoneyPlace
import uz.keel.owner.data.MoneyPosition
import uz.keel.owner.t
import java.time.LocalDate
import java.time.temporal.ChronoUnit

// Where the money is — the question an owner asks standing up, in the evening.
//
// ⚠️ **Three totals, never one**, the same rule as the panel's page and the
// server (`handlers/money.go`): cash can be spent tonight, the bank this week,
// an aggregator's balance when somebody else decides. There is no grand total on
// this screen and there must not be one — it is the number that gets quoted to a
// landlord.
//
// ⚠️ **Every row says counted or added up**, because they fail in opposite
// directions: a counted figure goes stale, a summed one goes wrong when a
// document is missing.
//
// ⚠️ **The one thing written from here is the bank balance**, because the phone
// is where it is read: the owner opens the bank's app and types the figure over.
// Inkassatsiya and the cash limit stay in the panel — the first is signed at the
// drawer with the bag in hand, the second is a contract figure entered once.

@Composable
fun MoneyScreen(api: KeelApi, bottomInset: PaddingValues, onBack: () -> Unit) {
    val c = KeelTheme.colors
    val branchId = LocalPrefs.current.branch.value
    val scope = rememberCoroutineScope()
    var pos by remember { mutableStateOf<MoneyPosition?>(null) }
    var error by remember { mutableStateOf("") }
    val loadFailed = t.common.loadFailed

    suspend fun load() {
        try {
            pos = api.money(branchId)
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else loadFailed
        }
    }
    LaunchedEffect(branchId) { load() }

    val p = pos
    Column(Modifier.fillMaxSize()) {
        ScreenHeader(
            title = t.money.title,
            leading = { GlassIconButton(Icons.Rounded.ArrowBackIosNew, onClick = onBack) },
            trailing = { GlassIconButton(Icons.Rounded.Refresh) { scope.launch { load() } } },
        )
        Column(
            Modifier.fillMaxSize().imePadding().verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp)
                .padding(bottom = bottomInset.calculateBottomPadding() + 24.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(t.money.intro, style = MaterialTheme.typography.bodyMedium, color = c.muted)

            if (p == null && error.isEmpty()) {
                Box(Modifier.fillMaxWidth().padding(40.dp), Alignment.Center) {
                    CircularProgressIndicator(color = c.accent)
                }
            }
            if (error.isNotEmpty()) {
                Text(error, color = c.danger, style = MaterialTheme.typography.bodyMedium)
            }

            if (p != null) {
                // ⚠️ A legal ceiling, not a preference — and absent when no limit
                // was entered: inventing the bank's figure is worse than silence.
                if (p.overLimit) {
                    Text(
                        t.money.overLimit(money(p.cashTotal), money(p.cashLimit)),
                        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp)).padding(14.dp),
                        style = MaterialTheme.typography.bodyMedium, color = c.warn,
                    )
                }
                Group(t.money.cash, t.money.cashHint, p.cash, p.cashTotal)
                Group(t.money.bank, t.money.bankHint, p.bank, p.bankTotal)
                Group(t.money.rails, t.money.railsHint, p.rails, p.railsTotal)

                BankForm(
                    accounts = p.bank.map { it.name }.filter { it.isNotBlank() }.distinct(),
                ) { account, amount ->
                    api.saveBankBalance(branchId, account, amount, LocalDate.now().toString())
                    load()
                }
            }
        }
    }
}

@Composable
private fun Group(title: String, hint: String, places: List<MoneyPlace>, total: Double) {
    val c = KeelTheme.colors
    GlassCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(title, Modifier.weight(1f), style = MaterialTheme.typography.titleMedium, color = c.ink)
            Text(
                money(total),
                style = MaterialTheme.typography.titleLarge.merge(MoneyStyle),
                color = if (total < 0) c.danger else c.ink,
            )
        }
        Muted(hint)
        if (places.isEmpty()) {
            Text(t.money.nothingHere, style = MaterialTheme.typography.bodyMedium, color = c.muted)
        }
        places.forEach { PlaceRow(it) }
    }
}

@Composable
private fun PlaceRow(p: MoneyPlace) {
    val c = KeelTheme.colors
    // ⚠️ The server names the first four in Uzbek; a Russian-speaking owner
    // should not read "Kassa yashigi". A bank account and a rail are names the
    // restaurant itself typed, and are shown as they are.
    val label = t.money.kinds[p.kind] ?: p.name
    val at = when (p.kind) {
        // A bank figure is true as of a day — its age is the point, so the date.
        "bank" -> localTime(p.at)?.let { "%02d.%02d.%d".format(it.dayOfMonth, it.monthValue, it.year) }
        else -> clock(p.at).takeIf { it.isNotEmpty() }
    }
    val note = p.note.takeIf { it.isNotBlank() }?.let {
        when (p.kind) {
            "drawer" -> t.money.openedBy(it)
            "rail" -> t.money.settledThrough(it)
            else -> it
        }
    }
    // ⚠️ **A counted figure a week old is flagged**, because that is the way a
    // counted figure goes wrong: nothing about it changes, it just stops being
    // true.
    val stale = p.counted && localTime(p.at)?.let {
        ChronoUnit.DAYS.between(it.toLocalDate(), LocalDate.now()) >= 7
    } == true
    Row(Modifier.fillMaxWidth().padding(top = 6.dp), verticalAlignment = Alignment.Top) {
        Column(Modifier.weight(1f)) {
            Text(label, style = MaterialTheme.typography.bodyLarge, color = c.ink)
            Muted(
                listOfNotNull(if (p.counted) t.money.counted else t.money.summed, at, note).joinToString(" · "),
                color = if (stale) c.warn else null,
            )
        }
        // ⚠️ Negative is shown as negative. A safe below zero is a ledger entry
        // nobody wrote, and clamping it would reconcile the one screen where
        // somebody counts the box with a wrong count.
        Text(
            money(p.amount),
            style = MaterialTheme.typography.bodyLarge.merge(MoneyStyle),
            color = if (p.amount < 0) c.danger else c.ink,
        )
    }
}

@Composable
private fun BankForm(accounts: List<String>, onSave: suspend (String, Long) -> Unit) {
    val c = KeelTheme.colors
    val notice = LocalNotice.current
    val scope = rememberCoroutineScope()
    var account by remember(accounts) { mutableStateOf(accounts.firstOrNull().orEmpty()) }
    var amount by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    val saved = t.money.saved
    val failed = t.money.failed

    GlassCard {
        Text(t.money.bankTitle, style = MaterialTheme.typography.titleMedium, color = c.ink)
        Muted(t.money.bankFormHint)
        if (accounts.size > 1) {
            Row(
                Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                accounts.forEach { a -> Chip(a, account == a) { account = a } }
            }
        }
        GlassField(account, { account = it.take(80) }, t.money.account)
        GlassField(
            amount, { amount = it.filter(Char::isDigit).take(15) }, t.money.balance,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
        )
        // ⚠️ The figure read back grouped: a field of bare digits is where an
        // extra nought goes unnoticed, and this one is a bank balance.
        amount.toLongOrNull()?.let { Muted(money(it.toDouble())) }
        PrimaryButton(
            t.money.save,
            enabled = account.isNotBlank() && amount.isNotEmpty() && !busy,
            busy = busy,
        ) {
            val n = amount.toLongOrNull() ?: return@PrimaryButton
            scope.launch {
                busy = true
                try {
                    onSave(account.trim(), n)
                    amount = ""
                    notice.value = Note(NoticeKind.Ok, saved)
                } catch (e: Throwable) {
                    notice.value = Note(NoticeKind.Error, failed, (e as? ApiError)?.message)
                } finally {
                    busy = false
                }
            }
        }
    }
}
