package uz.keel.waiter.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
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
import androidx.compose.material.icons.rounded.DarkMode
import androidx.compose.material.icons.rounded.ErrorOutline
import androidx.compose.material.icons.rounded.Language
import androidx.compose.material.icons.rounded.LightMode
import androidx.compose.material.icons.rounded.Logout
import androidx.compose.material.icons.rounded.Notifications
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.PhoneAndroid
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.Storefront
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
import uz.keel.waiter.LocalPrefs
import uz.keel.waiter.data.KeelApi
import uz.keel.waiter.data.Staff
import uz.keel.waiter.i18n.DICTS
import uz.keel.waiter.i18n.Lang
import uz.keel.waiter.push.PushState
import uz.keel.waiter.t
import uz.keel.design.*
import androidx.compose.material.icons.rounded.Settings

// Language, appearance, and the two ways out.

@Composable
fun SettingsScreen(
    api: KeelApi,
    staff: Staff,
    address: String,
    bottomInset: PaddingValues,
    /** ⚠️ Shown rather than hidden: "the kitchen pressed ready and nothing
     *  arrived" has five possible causes, and without this there is no way to
     *  tell them apart from the phone it happened on. */
    pushState: PushState,
    onRetryPush: () -> Unit,
    onSignOut: () -> Unit,
    onForgetServer: () -> Unit,
    /** Keel's account card: who this is, and the way to another workspace.
     *  ⚠️ First on the screen, because "where is the switch?" is asked here. */
    top: @Composable () -> Unit = {},
) {
    val c = KeelTheme.colors
    val prefs = LocalPrefs.current
    val ctx = LocalContext.current
    var branch by remember { mutableStateOf("") }

    // ⚠️ Asked rather than assumed: the branch is a fact about the account, and
    // showing it is what lets somebody notice they are signed in to the wrong one
    // — which is the failure a branch picker would have caused deliberately.
    LaunchedEffect(Unit) {
        branch = runCatching { api.branch().name }.getOrDefault("")
    }

    val version = remember { appVersion(ctx) }

    Column(Modifier.fillMaxSize()) {
        ScreenHeader(title = t.settings.title)
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp)
                .padding(bottom = bottomInset.calculateBottomPadding() + 24.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            top()
            // ⚠️ **Two rows, not six.** Chosen once in the life of a phone and
            // read constantly — as permanent lists they took a third of the
            // screen to say two words.
            Section(t.settings.title, Icons.Rounded.Settings) {
                PickerRow(
                    label = t.settings.language,
                    icon = Icons.Rounded.Language,
                    // ⚠️ Each language names itself, in itself: a list that said
                    // "Russian" in Uzbek is a list a Russian speaker has to
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
                        if (pushState == PushState.Working) Icons.Rounded.CheckCircle else Icons.Rounded.ErrorOutline,
                        null,
                        tint = if (pushState == PushState.Working) c.accent else c.danger,
                        modifier = Modifier.size(18.dp),
                    )
                    Column(Modifier.weight(1f)) {
                        Text(t.settings.push[key] ?: key, color = c.ink, style = MaterialTheme.typography.bodyMedium)
                        Text(t.settings.pushHint[key] ?: "", color = c.muted, style = MaterialTheme.typography.labelMedium)
                    }
                }
                if (pushState != PushState.Working) {
                    Choice(t.common.retry, Icons.Rounded.Refresh, on = false, tint = c.accent, onClick = onRetryPush)
                }
            }

            Section(t.settings.account, Icons.Rounded.Person) {
                InfoRow(staff.name, staff.position)
                InfoRow(t.settings.restaurant, address)
                if (branch.isNotEmpty()) InfoRow(t.settings.branch, branch)
                InfoRow(t.settings.version, version)
            }

            // ⚠️ Two ways out, kept apart on purpose. A shift ends every evening;
            // a phone changes restaurant once, if ever. One button doing both
            // would make the daily action cost the rare one's setup — and the rare
            // one is destructive in a way the daily one is not.
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
        Text(value, color = c.muted, style = MaterialTheme.typography.labelMedium)
    }
}
