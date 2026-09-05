package uz.keel.waiter.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.CallMerge
import androidx.compose.material.icons.rounded.CheckBox
import androidx.compose.material.icons.rounded.CheckBoxOutlineBlank
import androidx.compose.material.icons.rounded.ChevronRight
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.ContentCut
import androidx.compose.material.icons.rounded.Group
import androidx.compose.material.icons.rounded.TurnRight
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import uz.keel.waiter.data.ApiError
import uz.keel.waiter.data.Check
import uz.keel.waiter.data.KeelApi
import uz.keel.waiter.t
import uz.keel.waiter.ui.components.Money
import uz.keel.waiter.ui.components.PrimaryButton
import uz.keel.waiter.ui.theme.KeelTheme
import uz.keel.waiter.ui.theme.glass

// The three things a table does that a single check cannot express.
//
// ⚠️ **All three exist because a party is not a bill.** Guests arrive at one
// table and pay in two; two tables are pushed together; somebody sits down at the
// wrong one and the food follows them. A till that can only open and close a
// check sends every one of those to the counter — which is where they used to go,
// and the walk this removes.

@Composable
fun TableActions(
    api: KeelApi,
    check: Check,
    /** The branch's other open checks — the only possible destinations. */
    others: List<Check>,
    /** ⚠️ `menu` is a real state, not the absence of one: the sheet opens on a
     *  choice of four occasional actions, and going straight to any of them
     *  would make three of them unreachable. */
    job: String,
    onJob: (String) -> Unit,
    onDone: (Check) -> Unit,
    onClose: () -> Unit,
) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()
    val sheet = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val picked = remember { mutableStateListOf<String>() }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    val failed = t.table.failed

    val lines = check.lines.filter { it.voided == null }

    fun run(work: suspend () -> Check) {
        busy = true
        error = ""
        scope.launch {
            try {
                onDone(work())
                onClose()
            } catch (e: ApiError) {
                error = e.message; busy = false
            } catch (e: Throwable) {
                error = failed; busy = false
            }
        }
    }

    ModalBottomSheet(
        onDismissRequest = onClose,
        sheetState = sheet,
        containerColor = Color.Transparent,
        dragHandle = null,
    ) {
        Column(
            Modifier
                .fillMaxWidth()
                .glass(c, RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp), strong = true)
                .background(c.bg.copy(alpha = 0.92f), RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp))
                .navigationBarsPadding()
                .padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    if (job == "menu") t.table.actions else jobTitle(job),
                    Modifier.weight(1f),
                    style = MaterialTheme.typography.titleLarge, color = c.ink,
                )
                Box(Modifier.size(36.dp).clickable(onClick = onClose), contentAlignment = Alignment.Center) {
                    Icon(Icons.Rounded.Close, null, tint = c.muted, modifier = Modifier.size(20.dp))
                }
            }

            if (job == "menu") {
                listOf(
                    "guests" to Icons.Rounded.Group,
                    "split" to Icons.Rounded.ContentCut,
                    "move" to Icons.Rounded.TurnRight,
                    "merge" to Icons.Rounded.CallMerge,
                ).forEach { (key, icon) ->
                    Row(
                        Modifier.fillMaxWidth()
                            .glass(c, RoundedCornerShape(18.dp))
                            .clickable { onJob(key) }
                            .padding(horizontal = 16.dp, vertical = 15.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        Icon(icon, null, tint = c.accent, modifier = Modifier.size(20.dp))
                        Text(jobTitle(key), Modifier.weight(1f), color = c.ink, style = MaterialTheme.typography.bodyLarge)
                        Icon(Icons.Rounded.ChevronRight, null, tint = c.muted, modifier = Modifier.size(18.dp))
                    }
                }
            }

            if (job == "guests") {
                Text(t.table.guestsHint, color = c.muted, style = MaterialTheme.typography.bodyMedium)
                FlowRow(
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                    verticalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    listOf(1, 2, 3, 4, 5, 6, 8, 10, 12).forEach { n ->
                        val on = check.guests == n
                        Box(
                            Modifier.size(58.dp)
                                .glass(c, RoundedCornerShape(18.dp))
                                .then(if (on) Modifier.background(c.accentSoft, RoundedCornerShape(18.dp)) else Modifier)
                                .clickable(enabled = !busy) { run { api.updateCheck(check.id, n) } },
                            contentAlignment = Alignment.Center,
                        ) {
                            Text("$n", color = if (on) c.accent else c.ink, style = MaterialTheme.typography.titleMedium)
                        }
                    }
                }
            }

            if (job == "split" || job == "move") {
                // ⚠️ **Lines are picked, never "half".** A guest pays for what
                // they ate; splitting a total down the middle is a different
                // thing that reads the same on a screen and is wrong at the table.
                Text(
                    if (job == "split") t.table.splitHint else t.table.moveHint,
                    color = c.muted, style = MaterialTheme.typography.bodyMedium,
                )
                LazyColumn(
                    Modifier.heightIn(max = 260.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    items(lines, key = { it.lineId }) { l ->
                        val on = picked.contains(l.lineId)
                        Row(
                            Modifier.fillMaxWidth()
                                .glass(c, RoundedCornerShape(16.dp))
                                .then(if (on) Modifier.background(c.accentSoft, RoundedCornerShape(16.dp)) else Modifier)
                                .clickable { if (on) picked.remove(l.lineId) else picked.add(l.lineId) }
                                .padding(horizontal = 14.dp, vertical = 12.dp),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(10.dp),
                        ) {
                            Icon(
                                if (on) Icons.Rounded.CheckBox else Icons.Rounded.CheckBoxOutlineBlank,
                                null, tint = if (on) c.accent else c.muted, modifier = Modifier.size(19.dp),
                            )
                            Text(
                                l.name + if (l.qty > 1) " × ${l.qty}" else "",
                                Modifier.weight(1f), color = c.ink,
                                style = MaterialTheme.typography.bodyMedium,
                            )
                            Money(l.sum)
                        }
                    }
                }
            }

            if (job == "merge" || job == "move") {
                Text(t.table.pickTable, color = c.muted, style = MaterialTheme.typography.bodyMedium)
                LazyColumn(
                    Modifier.heightIn(max = 220.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    items(others, key = { it.id }) { o ->
                        Row(
                            Modifier.fillMaxWidth()
                                .glass(c, RoundedCornerShape(16.dp))
                                .clickable(enabled = !busy && (job != "move" || picked.isNotEmpty())) {
                                    run {
                                        if (job == "merge") api.merge(check.id, o.id)
                                        else api.moveLines(check.id, picked.toList(), o.id)
                                    }
                                }
                                .padding(horizontal = 14.dp, vertical = 13.dp),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(10.dp),
                        ) {
                            Icon(Icons.Rounded.TurnRight, null, tint = c.accent, modifier = Modifier.size(18.dp))
                            Text(
                                o.tableNumber?.let { t.check.table(it) } ?: o.number,
                                Modifier.weight(1f), color = c.ink,
                                style = MaterialTheme.typography.bodyMedium,
                            )
                            Money(o.total)
                        }
                    }
                    if (others.isEmpty()) {
                        item {
                            // ⚠️ Said rather than shown as an empty list: there
                            // being no other open table is the ordinary state, not
                            // a fault.
                            Text(
                                t.table.noOthers, color = c.muted, textAlign = TextAlign.Center,
                                modifier = Modifier.fillMaxWidth().padding(12.dp),
                            )
                        }
                    }
                }
            }

            if (error.isNotEmpty()) {
                Text(error, color = c.danger, style = MaterialTheme.typography.bodyMedium)
            }

            if (job == "split") {
                PrimaryButton(
                    t.table.splitDo(picked.size),
                    enabled = !busy && picked.isNotEmpty(), busy = busy,
                ) { run { api.split(check.id, picked.toList()).check } }
            }
        }
    }
}

@Composable
private fun jobTitle(job: String): String = when (job) {
    "guests" -> t.table.guests
    "split" -> t.table.split
    "merge" -> t.table.merge
    "move" -> t.table.move
    else -> t.table.actions
}
