package uz.keel.app.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.ChevronRight
import androidx.compose.material.icons.rounded.PersonAdd
import androidx.compose.material.icons.rounded.Tune
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import uz.keel.app.data.Account
import uz.keel.app.data.Workspace
import uz.keel.design.KeelTheme
import uz.keel.design.glass
import uz.keel.design.glassSheet
import uz.keel.design.keelGradient

/** A quick switch between the workspaces this phone's accounts open.
 *
 *  ⚠️ **A sheet over the work, not a page you leave it for.** The person is in
 *  the middle of something; switching is one tap and a glance, and the screen
 *  they were on is still behind it. The full "manage accounts" page (sign out,
 *  another restaurant) is one row down, because that is the rare thing.
 *
 *  ⚠️ **Shown only when there is a choice.** One workspace never opens this —
 *  its account card is a label, not a door. */
@Composable
fun WorkspaceSwitcher(
    current: Workspace,
    account: Account,
    workspaces: List<Workspace>,
    onPick: (Workspace) -> Unit,
    onAdd: () -> Unit,
    onManage: () -> Unit,
    onDismiss: () -> Unit,
) {
    val c = KeelTheme.colors
    val words = w
    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false),
    ) {
        Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.BottomCenter) {
            Column(
                Modifier
                    .fillMaxWidth()
                    .glassSheet(c, RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp))
                    .navigationBarsPadding()
                    .padding(horizontal = 14.dp, vertical = 14.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                // The grip, so the sheet reads as something you pull down.
                Box(
                    Modifier
                        .padding(bottom = 4.dp)
                        .size(width = 36.dp, height = 4.dp)
                        .clip(CircleShape)
                        .background(c.line)
                        .align(Alignment.CenterHorizontally),
                )
                Text(
                    words.hub.switchTitle,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    color = c.ink,
                    modifier = Modifier.padding(start = 4.dp, bottom = 2.dp),
                )

                workspaces.forEach { ws ->
                    WorkspaceLine(ws, on = ws == current) { if (ws != current) onPick(ws) }
                }

                Spacer(Modifier.height(2.dp))
                QuietRow(Icons.Rounded.PersonAdd, words.hub.add, onAdd)
                QuietRow(Icons.Rounded.Tune, words.hub.manage, onManage)
            }
        }
    }
}

@Composable
private fun WorkspaceLine(ws: Workspace, on: Boolean, onClick: () -> Unit) {
    val c = KeelTheme.colors
    val space = ws.words(w)
    val shape = RoundedCornerShape(20.dp)
    Row(
        Modifier
            .fillMaxWidth()
            .clip(shape)
            .then(if (on) Modifier.border(1.5.dp, c.accent, shape) else Modifier.glass(c, shape))
            .clickable(onClick = onClick)
            .padding(12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Box(
            Modifier
                .size(44.dp)
                .clip(RoundedCornerShape(14.dp))
                .then(if (on) Modifier.background(keelGradient()) else Modifier.background(c.accentSoft)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(ws.icon(), null, tint = if (on) c.onAccent else c.accent, modifier = Modifier.size(22.dp))
        }
        Column(Modifier.weight(1f)) {
            Text(
                space.title,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = if (on) FontWeight.SemiBold else FontWeight.Normal,
                color = c.ink,
            )
            Text(
                space.lead,
                style = MaterialTheme.typography.labelMedium,
                color = c.muted,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        if (on) Icon(Icons.Rounded.Check, null, tint = c.accent, modifier = Modifier.size(20.dp))
        else Icon(Icons.Rounded.ChevronRight, null, tint = c.muted, modifier = Modifier.size(20.dp))
    }
}

@Composable
private fun QuietRow(icon: androidx.compose.ui.graphics.vector.ImageVector, label: String, onClick: () -> Unit) {
    val c = KeelTheme.colors
    Row(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Icon(icon, null, tint = c.muted, modifier = Modifier.size(20.dp))
        Text(label, style = MaterialTheme.typography.bodyMedium, color = c.inkSoft, modifier = Modifier.weight(1f))
        Icon(Icons.Rounded.ChevronRight, null, tint = c.muted, modifier = Modifier.size(18.dp))
    }
}
