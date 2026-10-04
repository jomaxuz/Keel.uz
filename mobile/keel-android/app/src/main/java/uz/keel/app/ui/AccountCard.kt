package uz.keel.app.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.SwapHoriz
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
import uz.keel.app.data.Account
import uz.keel.app.data.Workspace
import uz.keel.design.KeelTheme
import uz.keel.design.glass

/** Who is signed in to this workspace, and the way to the others.
 *
 *  ⚠️ **First on every role's settings screen**, because that is where somebody
 *  goes looking for "sign out" — and in Keel the question behind it is usually
 *  "how do I get to the other one?". The single-role apps had no such question:
 *  the other one was another icon on the home screen.
 *
 *  ⚠️ **Shown even with one workspace.** "Add an account" lives behind it, and
 *  an owner who is handed a courier login has to find where it goes. */
@Composable
fun AccountCard(account: Account, workspace: Workspace, canSwitch: Boolean = true, onSwitch: () -> Unit) {
    val c = KeelTheme.colors
    val words = w
    Row(
        Modifier
            .fillMaxWidth()
            .glass(c, RoundedCornerShape(24.dp), strong = true)
            .then(if (canSwitch) Modifier.clickable(onClick = onSwitch) else Modifier)
            .padding(14.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Avatar(account.name, size = 46.dp)
        Column(Modifier.weight(1f)) {
            Text(
                account.name,
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
                color = c.ink,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                "${workspace.words(words).title} · ${account.label(words)}",
                style = MaterialTheme.typography.labelMedium,
                color = c.muted,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        if (canSwitch) {
            Row(
                Modifier
                    .clip(RoundedCornerShape(999.dp))
                    .background(c.accentSoft)
                    .padding(horizontal = 12.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                Icon(Icons.Rounded.SwapHoriz, null, tint = c.accent, modifier = Modifier.size(16.dp))
                Text(words.hub.switch, style = MaterialTheme.typography.labelLarge, color = c.accent)
            }
        }
    }
}
