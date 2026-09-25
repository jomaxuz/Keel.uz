package uz.keel.owner.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.ChevronRight
import androidx.compose.material.icons.rounded.DarkMode
import androidx.compose.material.icons.rounded.ErrorOutline
import androidx.compose.material.icons.rounded.HelpOutline
import androidx.compose.material.icons.rounded.Language
import androidx.compose.material.icons.rounded.LightMode
import androidx.compose.material.icons.rounded.Logout
import androidx.compose.material.icons.rounded.Notifications
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.PhoneAndroid
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.Storefront
import androidx.compose.material.icons.rounded.Wallet
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import uz.keel.design.KeelTheme
import uz.keel.design.Lang
import uz.keel.design.MoneyStyle
import uz.keel.design.PickerOption
import uz.keel.design.PickerRow
import uz.keel.design.ScreenHeader
import uz.keel.design.ThemeChoice
import uz.keel.design.glass
import uz.keel.design.money
import uz.keel.owner.LocalPrefs
import uz.keel.owner.data.AdminUser
import uz.keel.owner.data.KeelApi
import uz.keel.owner.data.Subscription
import uz.keel.owner.i18n.DICTS
import uz.keel.owner.push.PushState
import uz.keel.owner.t
import uz.keel.design.appVersion

// Language, appearance, the subscription, help, and the two ways out.

@Composable
fun SettingsScreen(
    api: KeelApi,
    user: AdminUser,
    address: String,
    branchName: String,
    bottomInset: PaddingValues,
    pushState: PushState,
    /** ⚠️ The server's or the operating system's own words. See
     *  `PushRegistration.detail`. */
    pushDetail: String?,
    onRetryPush: () -> Unit,
    onOpenHelp: () -> Unit,
    onSignOut: () -> Unit,
    onForgetServer: () -> Unit,
) {
    val c = KeelTheme.colors
    val prefs = LocalPrefs.current
    val ctx = LocalContext.current
    var sub by remember { mutableStateOf<Subscription?>(null) }

    LaunchedEffect(Unit) { sub = runCatching { api.subscription() }.getOrNull() }

    val version = remember { appVersion(ctx) }

    Column(Modifier.fillMaxSize()) {
        ScreenHeader(title = t.settings.title)
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp)
                .padding(bottom = bottomInset.calculateBottomPadding() + 24.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            // ⚠️ **Help first, because somebody opens this screen to find it.**
            // A row rather than a button floating over four screens: a permanent
            // control is in the way on all four.
            Row(
                Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp))
                    .clickable(onClick = onOpenHelp).padding(16.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Icon(Icons.Rounded.HelpOutline, null, tint = c.accent, modifier = Modifier.size(20.dp))
                Column(Modifier.weight(1f)) {
                    Text(t.settings.help, style = MaterialTheme.typography.bodyLarge, color = c.ink)
                    Text(t.settings.helpHint, style = MaterialTheme.typography.labelMedium, color = c.muted)
                }
                Icon(Icons.Rounded.ChevronRight, null, tint = c.muted, modifier = Modifier.size(18.dp))
            }

            // ⚠️ **Two rows, not six.** These are chosen once in the life of
            // a phone and read constantly; as permanent lists they took a third
            // of the screen to say two words, and pushed everything an owner
            // actually opens this screen for below the fold.
            Card {
                PickerRow(
                    label = t.settings.language,
                    icon = Icons.Rounded.Language,
                    // ⚠️ Each language named in itself: a list that said
                    // "Ruscha" in Uzbek is a list a Russian speaker has to
                    // decode before they can leave the language they cannot read.
                    options = Lang.entries.map { PickerOption(it.code, DICTS[it]!!.lang) },
                    selected = prefs.lang.value.code,
                ) { prefs.setLang(Lang.of(it)) }
                PickerRow(
                    label = t.settings.theme,
                    icon = Icons.Rounded.DarkMode,
                    options = listOf(
                        PickerOption("system", t.settings.themeSystem, Icons.Rounded.PhoneAndroid),
                        PickerOption("light", t.settings.themeLight, Icons.Rounded.LightMode),
                        PickerOption("dark", t.settings.themeDark, Icons.Rounded.DarkMode),
                    ),
                    selected = prefs.theme.value.name.lowercase(),
                ) {
                    prefs.setTheme(
                        when (it) {
                            "light" -> ThemeChoice.Light
                            "dark" -> ThemeChoice.Dark
                            else -> ThemeChoice.System
                        },
                    )
                }
            }

            Section(t.settings.notifications, Icons.Rounded.Notifications) {
                val key = pushState.key
                Row(
                    Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    Icon(
                        if (pushState == PushState.Working) Icons.Rounded.CheckCircle
                        else Icons.Rounded.ErrorOutline,
                        null,
                        tint = if (pushState == PushState.Working) c.accent else c.danger,
                        modifier = Modifier.size(18.dp),
                    )
                    Column(Modifier.weight(1f)) {
                        Text(t.settings.push[key] ?: key, color = c.ink, style = MaterialTheme.typography.bodyMedium)
                        Text(t.settings.pushHint[key] ?: "", color = c.muted, style = MaterialTheme.typography.labelMedium)
                        // ⚠️ **The reason, untranslated and selectable.** This is
                        // the line that tells "this build has no push
                        // credentials" apart from "the server is older than the
                        // endpoint" — neither of which is fixed from this phone,
                        // and both of which read as "not registered" without it.
                        pushDetail?.let {
                            Text(
                                it,
                                color = c.warn,
                                style = MaterialTheme.typography.labelSmall,
                            )
                        }
                    }
                }
                if (pushState != PushState.Working) {
                    Choice(t.common.retry, Icons.Rounded.Refresh, on = false, tint = c.accent, onClick = onRetryPush)
                }
            }

            // ⚠️ **The date and the count, never an "active" flag.** A boolean
            // goes stale at midnight while nobody is looking; a date does not.
            //
            // ⚠️ **And this screen does not warn.** The till says so in the last
            // week and the panel has the whole card — a phone that nags about
            // money is a phone an owner stops opening.
            Section(t.settings.plan, Icons.Rounded.Wallet) {
                val s = sub
                when {
                    s == null || !s.enabled ->
                        InfoRow(t.settings.planNone, t.settings.planNoneHint)
                    else -> {
                        InfoRow(
                            t.settings.planMonthly,
                            // ⚠️ Zero is "agreed separately", not "free": printing
                            // a nought here would read as a promise.
                            if (s.monthly > 0) money(s.monthly) else t.settings.planIndividual,
                        )
                        InfoRow(
                            if (s.paidUntil.isNotEmpty()) t.settings.planUntil(s.paidUntil)
                            else t.settings.planNoDate,
                            s.notice?.let { n ->
                                when {
                                    n.level == "expired" -> t.settings.planOverdue(n.days)
                                    n.days == 0 -> t.settings.planDueToday
                                    else -> t.settings.planDaysLeft(n.days)
                                }
                            } ?: "",
                        )
                    }
                }
            }

            Section(t.settings.account, Icons.Rounded.Person) {
                InfoRow(user.name.ifEmpty { user.username }, user.role)
                InfoRow(t.settings.restaurant, address)
                if (branchName.isNotEmpty()) InfoRow(t.settings.branch, branchName)
                InfoRow(t.settings.version, version)
            }

            // ⚠️ Two ways out, kept apart. A day ends every evening; a phone
            // changes restaurant once, if ever — and the rare one is destructive
            // in a way the daily one is not.
            Row(
                Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp))
                    .clickable(onClick = onSignOut).padding(horizontal = 16.dp, vertical = 15.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Icon(Icons.Rounded.Logout, null, tint = c.ink, modifier = Modifier.size(19.dp))
                Text(t.settings.signOut, Modifier.weight(1f), color = c.ink, style = MaterialTheme.typography.bodyLarge)
            }
            Row(
                Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp))
                    .clickable(onClick = onForgetServer).padding(horizontal = 16.dp, vertical = 15.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Icon(Icons.Rounded.Storefront, null, tint = c.danger, modifier = Modifier.size(19.dp))
                Column(Modifier.weight(1f)) {
                    Text(t.settings.changeServer, color = c.danger, style = MaterialTheme.typography.bodyLarge)
                    Text(t.settings.changeServerHint, color = c.muted, style = MaterialTheme.typography.labelMedium)
                }
            }
        }
    }
}

@Composable
private fun Card(content: @Composable () -> Unit) {
    val c = KeelTheme.colors
    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(6.dp),
        verticalArrangement = Arrangement.spacedBy(2.dp),
    ) { content() }
}

@Composable
private fun Section(title: String, icon: ImageVector, content: @Composable () -> Unit) {
    val c = KeelTheme.colors
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Row(
            Modifier.padding(start = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            Icon(icon, null, tint = c.muted, modifier = Modifier.size(14.dp))
            Text(title, color = c.muted, style = MaterialTheme.typography.labelMedium)
        }
        Column(
            Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(6.dp),
            verticalArrangement = Arrangement.spacedBy(2.dp),
        ) { content() }
    }
}

@Composable
private fun Choice(
    label: String,
    icon: ImageVector? = null,
    on: Boolean,
    tint: Color? = null,
    onClick: () -> Unit,
) {
    val c = KeelTheme.colors
    Row(
        Modifier.fillMaxWidth()
            .background(if (on) c.accentSoft else Color.Transparent, RoundedCornerShape(14.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 13.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        if (icon != null) Icon(icon, null, tint = tint ?: c.muted, modifier = Modifier.size(17.dp))
        Text(label, Modifier.weight(1f), color = tint ?: c.ink, style = MaterialTheme.typography.bodyMedium)
        // ⚠️ A tick rather than colour alone: the chosen row has to be
        // identifiable without relying on a wash somebody may not see.
        if (on) Icon(Icons.Rounded.Check, null, tint = c.accent, modifier = Modifier.size(18.dp))
    }
}

@Composable
private fun InfoRow(label: String, value: String) {
    val c = KeelTheme.colors
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, Modifier.weight(1f), color = c.inkSoft, style = MaterialTheme.typography.bodyMedium)
        Text(
            value, color = c.muted,
            style = MaterialTheme.typography.labelMedium.merge(MoneyStyle),
        )
    }
}
