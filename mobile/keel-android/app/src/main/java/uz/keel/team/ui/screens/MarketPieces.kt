package uz.keel.team.ui.screens

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.SwapHoriz
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import uz.keel.design.GlassField
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
 *  the catalogue does not have yet.
 *
 *  ⚠️ **The plus is a filled circle, not a bare glyph.** It is the only thing on
 *  the row that says "tap me", and at 20dp in orange on glass it read as
 *  decoration — people tapped the name, hesitated, and tapped again. */
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
            .padding(start = 14.dp, end = 10.dp, top = 10.dp, bottom = 10.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.bodyLarge, color = c.ink)
            if (subtitle.isNotEmpty()) {
                Text(subtitle, style = MaterialTheme.typography.labelMedium, color = c.muted)
            }
        }
        Box(
            Modifier.size(32.dp).clip(CircleShape).background(c.accentSoft),
            contentAlignment = Alignment.Center,
        ) {
            Icon(icon, null, tint = c.accent, modifier = Modifier.size(18.dp))
        }
    }
}

/** The top of a market screen: what this is, and one line of where it stands. */
@Composable
fun MarketHeader(title: String, subtitle: String = "") {
    val c = KeelTheme.colors
    Column(Modifier.statusBarsPadding().padding(top = 12.dp, bottom = 2.dp)) {
        Text(title, style = MaterialTheme.typography.headlineMedium, color = c.ink)
        if (subtitle.isNotEmpty()) {
            Text(subtitle, style = MaterialTheme.typography.bodyMedium, color = c.muted)
        }
    }
}

/** A section heading with an optional count beside it. */
@Composable
fun SectionTitle(text: String, count: Int? = null) {
    val c = KeelTheme.colors
    Row(
        Modifier.fillMaxWidth().padding(top = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(text, style = MaterialTheme.typography.titleMedium, color = c.ink)
        if (count != null && count > 0) Pill(count.toString(), c.accent)
    }
}

/** A small rounded label: a status, a count, where a line goes. */
@Composable
fun Pill(text: String, tint: Color, modifier: Modifier = Modifier) {
    Text(
        text,
        modifier
            .clip(CircleShape)
            .background(tint.copy(alpha = 0.14f))
            .padding(horizontal = 9.dp, vertical = 3.dp),
        style = MaterialTheme.typography.labelMedium,
        color = tint,
    )
}

/** How much of a list is done, as a bar — read at arm's length at a stall. */
@Composable
fun ProgressLine(done: Int, all: Int, modifier: Modifier = Modifier) {
    val c = KeelTheme.colors
    val f by animateFloatAsState(if (all == 0) 0f else done.toFloat() / all, label = "progress")
    Box(
        modifier.fillMaxWidth().height(6.dp).clip(CircleShape).background(c.line),
    ) {
        Box(
            Modifier
                .fillMaxHeight()
                .fillMaxWidth(f)
                .clip(CircleShape)
                .background(if (done == all && all > 0) c.ready else c.accent),
        )
    }
}

/** A message after an action: red when it failed, green when it landed. */
@Composable
fun MarketBanner(text: String, error: Boolean) {
    val c = KeelTheme.colors
    val tint = if (error) c.danger else c.ready
    Text(
        text,
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .background(tint.copy(alpha = 0.12f))
            .padding(horizontal = 14.dp, vertical = 10.dp),
        style = MaterialTheme.typography.bodyMedium,
        color = tint,
    )
}

/** A quantity, with its unit **inside** the box.
 *
 *  ⚠️ **One control rather than a label above a field.** The unit used to sit
 *  over the box as a caption, and a caption is what people skip: "5" went into a
 *  kilo field meant for bunches. Written in the box, beside the digits, it is
 *  read with the number — and where a packaging exists it is the switch. Same
 *  rule as [UnitSwitch]: a switch, never a conversion. */
@Composable
fun QtyField(
    value: String,
    onValueChange: (String) -> Unit,
    unit: String,
    packName: String,
    packQty: Double,
    inPacks: Boolean,
    onToggle: () -> Unit,
    modifier: Modifier = Modifier,
    placeholder: String = "",
) {
    GlassField(
        value = value,
        onValueChange = { onValueChange(it.replace(',', '.')) },
        placeholder = placeholder,
        modifier = modifier,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
        trailing = { UnitSwitch(unit, packName, packQty, inPacks, onToggle) },
    )
}

/** A price in so'm, digits only. */
@Composable
fun PriceField(value: String, onValueChange: (String) -> Unit, modifier: Modifier = Modifier, placeholder: String = "") {
    val c = KeelTheme.colors
    GlassField(
        value = value,
        onValueChange = { onValueChange(it.filter { ch -> ch.isDigit() }) },
        placeholder = placeholder,
        modifier = modifier,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
        trailing = {
            Text(t.buy.currency, Modifier.padding(end = 12.dp), style = MaterialTheme.typography.labelMedium, color = c.muted)
        },
    )
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
private fun UnitSwitch(unit: String, packName: String, packQty: Double, inPacks: Boolean, onToggle: () -> Unit) {
    val c = KeelTheme.colors
    val hasPack = packName.isNotEmpty() && packQty > 0
    val shown = if (hasPack && inPacks) packName else unit
    if (!hasPack) {
        if (shown.isNotEmpty()) {
            Text(shown, Modifier.padding(end = 12.dp), style = MaterialTheme.typography.labelMedium, color = c.muted)
        }
        return
    }
    Row(
        Modifier
            .padding(end = 8.dp)
            .clip(CircleShape)
            .background(c.accentSoft)
            .clickable(onClick = onToggle)
            .padding(horizontal = 10.dp, vertical = 6.dp),
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(shown, style = MaterialTheme.typography.labelMedium, color = c.accent)
        Icon(Icons.Rounded.SwapHoriz, null, tint = c.accent, modifier = Modifier.size(14.dp))
    }
}
