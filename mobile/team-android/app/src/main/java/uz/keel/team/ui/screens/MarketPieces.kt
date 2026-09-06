package uz.keel.team.ui.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.SwapHoriz
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import uz.keel.design.KeelTheme
import uz.keel.design.glass
import uz.keel.team.t

// The two pieces both market screens are made of.
//
// ⚠️ **Shared because they carry the same rule, not because they look alike.**
// One person writes the list and another shops it, and the unit has to mean the
// same thing on both screens — a control copied into each is a control that gets
// corrected on one of them.

/** A row that adds something to a list: a shortage, a catalogue match, or a name
 *  the catalogue does not have yet. */
@Composable
fun PickRow(
    title: String,
    subtitle: String,
    icon: ImageVector = Icons.Rounded.Add,
    onClick: () -> Unit,
) {
    val c = KeelTheme.colors
    Row(
        Modifier
            .fillMaxWidth()
            .glass(c, RoundedCornerShape(16.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.bodyLarge, color = c.ink)
            if (subtitle.isNotEmpty()) {
                Text(subtitle, style = MaterialTheme.typography.labelMedium, color = c.muted)
            }
        }
        Icon(icon, null, tint = c.accent, modifier = Modifier.size(20.dp))
    }
}

/** The unit: tapped where a market packaging is written down, a label where it
 *  is not.
 *
 *  ⚠️ **Never free text.** A market sells mint in bunches and flour in sacks;
 *  "5" typed into a field measured in kilos puts five kilos on the shelf instead
 *  of a quarter of one, the figure is then twenty times too high, the stop list
 *  never fires, and the gap surfaces a month later at a count as a shortfall the
 *  person holding the clipboard is asked to explain.
 *
 *  ⚠️ **A switch, never a conversion.** Tapping it changes which unit the typed
 *  number counts; the arithmetic belongs to the server, because the result lands
 *  on a shelf. */
@Composable
fun UnitLabel(
    unit: String,
    packName: String,
    packQty: Double,
    inPacks: Boolean,
    onToggle: () -> Unit,
) {
    val c = KeelTheme.colors
    val hasPack = packName.isNotEmpty() && packQty > 0
    if (!hasPack) {
        Text(t.buy.qty(unit), style = MaterialTheme.typography.labelMedium, color = c.muted)
        return
    }
    Row(
        Modifier.clickable(onClick = onToggle),
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            t.buy.qty(if (inPacks) packName else unit),
            style = MaterialTheme.typography.labelMedium,
            color = c.accent,
        )
        Icon(Icons.Rounded.SwapHoriz, null, tint = c.accent, modifier = Modifier.size(14.dp))
    }
}
