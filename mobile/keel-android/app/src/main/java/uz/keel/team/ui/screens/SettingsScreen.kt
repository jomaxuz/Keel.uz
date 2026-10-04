package uz.keel.team.ui.screens

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
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.DarkMode
import androidx.compose.material.icons.rounded.Home
import androidx.compose.material.icons.rounded.LightMode
import androidx.compose.material.icons.rounded.Logout
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.Smartphone
import androidx.compose.material.icons.rounded.WarningAmber
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import uz.keel.design.KeelTheme
import uz.keel.design.Lang
import uz.keel.design.ThemeChoice
import uz.keel.design.glass
import uz.keel.team.LocalPrefs
import uz.keel.team.data.Staff
import uz.keel.team.i18n.DICTS
import uz.keel.team.push.PushState
import uz.keel.team.t
import androidx.compose.ui.platform.LocalContext
import uz.keel.design.appVersion

// Language, appearance, and the two ways out.

@Composable
fun SettingsScreen(
    staff: Staff,
    address: String,
    bottomInset: PaddingValues,
    pushState: PushState,
    /** ⚠️ The server's or the operating system's own words. See
     *  `PushRegistration.detail`. */
    pushDetail: String?,
    onRetryPush: () -> Unit,
    onSignOut: () -> Unit,
    onForgetServer: () -> Unit,
    /** Keel's account card: who this is, and the way to another workspace.
     *  ⚠️ First on the screen, because "where is the switch?" is asked here. */
    top: @Composable () -> Unit = {},
) {
    val c = KeelTheme.colors
    val prefs = LocalPrefs.current

    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp)
            .padding(bottom = bottomInset.calculateBottomPadding()),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        Text(
            t.settings.title,
            Modifier.statusBarsPadding().padding(top = 8.dp),
            style = MaterialTheme.typography.headlineMedium,
            color = c.ink,
        )

        top()

        Section(t.settings.language) {
            Lang.entries.forEach { l ->
                // ⚠️ Each language names itself, in itself. A list that said
                // "Russian" in Uzbek is a list a Russian speaker has to decode
                // before they can leave the language they cannot read.
                Choice(DICTS[l]!!.lang, on = prefs.lang.value == l) { prefs.setLang(l) }
            }
        }

        Section(t.settings.theme) {
            Choice(
                t.settings.themeSystem,
                icon = Icons.Rounded.Smartphone,
                on = prefs.theme.value == ThemeChoice.System,
            ) { prefs.setTheme(ThemeChoice.System) }
            Choice(
                t.settings.themeLight,
                icon = Icons.Rounded.LightMode,
                on = prefs.theme.value == ThemeChoice.Light,
            ) { prefs.setTheme(ThemeChoice.Light) }
            Choice(
                t.settings.themeDark,
                icon = Icons.Rounded.DarkMode,
                on = prefs.theme.value == ThemeChoice.Dark,
            ) { prefs.setTheme(ThemeChoice.Dark) }
        }

        // ⚠️ **Shown rather than hidden.** "They said they paid me and nothing
        // arrived" has several causes, and this is the only place on the phone
        // that tells them apart — none of them is fixed from the screen somebody
        // reads their hours on.
        Section(t.settings.notifications) {
            Row(
                Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 12.dp),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    if (pushState == PushState.Working) Icons.Rounded.CheckCircle
                    else Icons.Rounded.WarningAmber,
                    null,
                    tint = if (pushState == PushState.Working) c.ready else c.warn,
                    modifier = Modifier.size(18.dp),
                )
                Column(Modifier.weight(1f)) {
                    Text(
                        when (pushState) {
                            PushState.Working -> t.push.working
                            PushState.Asking -> t.push.asking
                            PushState.Denied -> t.push.denied
                            PushState.Failed -> t.push.failed
                        },
                        style = MaterialTheme.typography.bodyLarge,
                        color = c.ink,
                    )
                    Text(
                        when (pushState) {
                            PushState.Working -> t.push.hintWorking
                            PushState.Asking -> t.push.hintAsking
                            PushState.Denied -> t.push.hintDenied
                            PushState.Failed -> t.push.hintFailed
                        },
                        style = MaterialTheme.typography.labelMedium,
                        color = c.muted,
                    )
                    // ⚠️ **The reason, untranslated.** This is the line that
                    // tells "this build has no push credentials" apart from
                    // "the server is older than the endpoint" — neither of
                    // which is fixed from this phone, and both of which read as
                    // "not registered" without it.
                    pushDetail?.let {
                        Text(it, style = MaterialTheme.typography.labelSmall, color = c.warn)
                    }
                }
            }
            if (pushState != PushState.Working) {
                Choice(t.common.retry, icon = Icons.Rounded.Refresh, on = false) { onRetryPush() }
            }
        }

        Section(t.settings.account) {
            InfoRow(staff.name, staff.position)
            InfoRow(t.settings.restaurant, address)
            InfoRow(t.settings.version, appVersion(LocalContext.current))
        }

        // ⚠️ **Two ways out, kept apart on purpose.** A day ends every evening; a
        // phone changes restaurant once, if ever. One button doing both would
        // make the daily action cost the rare one's setup — and the rare one is
        // destructive in a way the daily one is not.
        Row(
            Modifier
                .fillMaxWidth()
                .glass(c, RoundedCornerShape(18.dp))
                .clickable { onSignOut() }
                .padding(14.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(Icons.Rounded.Logout, null, tint = c.ink, modifier = Modifier.size(18.dp))
            Text(t.settings.signOut, style = MaterialTheme.typography.bodyLarge, color = c.ink)
        }

        Row(
            Modifier
                .fillMaxWidth()
                .glass(c, RoundedCornerShape(18.dp))
                .clickable { onForgetServer() }
                .padding(14.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(Icons.Rounded.Home, null, tint = c.danger, modifier = Modifier.size(18.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    t.settings.changeServer,
                    style = MaterialTheme.typography.bodyLarge,
                    color = c.danger,
                )
                Text(
                    t.settings.changeServerHint,
                    style = MaterialTheme.typography.labelMedium,
                    color = c.muted,
                )
            }
        }
    }
}

@Composable
private fun Section(title: String, content: @Composable () -> Unit) {
    val c = KeelTheme.colors
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(
            title,
            Modifier.padding(start = 4.dp),
            style = MaterialTheme.typography.labelMedium,
            color = c.muted,
        )
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
    onClick: () -> Unit,
) {
    val c = KeelTheme.colors
    Row(
        Modifier
            .fillMaxWidth()
            .background(
                if (on) c.accentSoft else Color.Transparent,
                RoundedCornerShape(14.dp),
            )
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 13.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (icon != null) Icon(icon, null, tint = c.muted, modifier = Modifier.size(17.dp))
        Text(label, Modifier.weight(1f), style = MaterialTheme.typography.bodyLarge, color = c.ink)
        // ⚠️ A tick rather than colour alone: the chosen row has to be
        // identifiable without relying on a wash somebody may not see.
        if (on) Icon(Icons.Rounded.Check, null, tint = c.accent, modifier = Modifier.size(19.dp))
    }
}

@Composable
private fun InfoRow(label: String, value: String) {
    val c = KeelTheme.colors
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 13.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, Modifier.weight(1f), style = MaterialTheme.typography.bodyMedium, color = c.inkSoft)
        Text(value, style = MaterialTheme.typography.labelMedium, color = c.muted)
    }
}
