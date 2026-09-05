package uz.keel.design

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.ExpandMore
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog

// One setting, one row, one sheet.
//
// ⚠️ **A list of every option, always open, is a settings screen that scrolls.**
// Language and appearance are chosen once in the life of a phone and read
// constantly — as three and three permanent rows they took a third of the screen
// to say two words. The row states what is chosen; the sheet is where changing
// it happens.
//
// ⚠️ **Shared, so the two applications cannot disagree about what a setting looks
// like.** They are the same product on two people's phones.

data class PickerOption(
    val id: String,
    val label: String,
    val icon: ImageVector? = null,
)

/** A settings row that opens a sheet. */
@Composable
fun PickerRow(
    label: String,
    options: List<PickerOption>,
    selected: String,
    icon: ImageVector? = null,
    onSelect: (String) -> Unit,
) {
    val c = KeelTheme.colors
    var open by remember { mutableStateOf(false) }
    val current = options.firstOrNull { it.id == selected }

    Row(
        Modifier.fillMaxWidth()
            .clickable { open = true }
            .padding(horizontal = 12.dp, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        if (icon != null) Icon(icon, null, tint = c.muted, modifier = Modifier.size(17.dp))
        Text(label, Modifier.weight(1f), color = c.ink, style = MaterialTheme.typography.bodyMedium)
        // ⚠️ The chosen value on the row itself: a settings screen that made you
        // open a sheet to find out what is set is a settings screen you open
        // twice.
        Text(
            current?.label.orEmpty(),
            color = c.muted,
            style = MaterialTheme.typography.bodyMedium,
        )
        Icon(Icons.Rounded.ExpandMore, null, tint = c.muted, modifier = Modifier.size(18.dp))
    }

    if (open) {
        Dialog(onDismissRequest = { open = false }) {
            Column(
                Modifier.widthIn(max = 340.dp)
                    .glassSheet(c, RoundedCornerShape(26.dp))
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(4.dp),
            ) {
                Text(
                    label,
                    style = MaterialTheme.typography.titleMedium, color = c.ink,
                    modifier = Modifier.padding(start = 8.dp, bottom = 6.dp),
                )
                options.forEach { o ->
                    val on = o.id == selected
                    Row(
                        Modifier.fillMaxWidth()
                            .clip(RoundedCornerShape(16.dp))
                            .background(if (on) c.accentSoft else Color.Transparent)
                            .clickable { onSelect(o.id); open = false }
                            .padding(horizontal = 14.dp, vertical = 14.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        if (o.icon != null) {
                            Icon(o.icon, null, tint = c.muted, modifier = Modifier.size(18.dp))
                        }
                        Text(
                            o.label, Modifier.weight(1f),
                            color = c.ink, style = MaterialTheme.typography.bodyLarge,
                        )
                        // ⚠️ A tick rather than colour alone: the chosen row has
                        // to be identifiable without relying on a wash somebody
                        // may not see.
                        if (on) Icon(Icons.Rounded.Check, null, tint = c.accent, modifier = Modifier.size(19.dp))
                    }
                }
            }
        }
    }
}
