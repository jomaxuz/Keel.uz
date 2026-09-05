package uz.keel.design

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.ErrorOutline
import androidx.compose.material.icons.rounded.WarningAmber
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.MutableState
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog

// Saying something that has to be read.
//
// ⚠️ **A line of small text at the bottom of a screen is not a notification.**
// "No printer took it" appeared under a scrolling list, below the buttons, in
// the size everything else on the screen is not — so the one message that
// changes what somebody does next was the one nobody saw. It is a sheet: it
// interrupts, it says what happened, and it is dismissed deliberately.
//
// ⚠️ **Only for what changes the next action.** Everything that can be shown in
// place stays in place — a refused dish is said beside the dish. A modal for
// each of those is a modal people learn to tap through without reading, and then
// this one goes with them.

enum class NoticeKind { Ok, Warn, Error }

data class Note(val kind: NoticeKind, val title: String, val body: String? = null)

val LocalNotice = compositionLocalOf<MutableState<Note?>> { mutableStateOf(null) }

@Composable
fun NoticeHost(state: MutableState<Note?>) {
    val note = state.value ?: return
    val c = KeelTheme.colors
    val tone = when (note.kind) {
        NoticeKind.Error -> c.danger
        NoticeKind.Warn -> c.warn
        NoticeKind.Ok -> c.accent
    }
    val icon = when (note.kind) {
        NoticeKind.Error -> Icons.Rounded.ErrorOutline
        NoticeKind.Warn -> Icons.Rounded.WarningAmber
        NoticeKind.Ok -> Icons.Rounded.CheckCircle
    }
    // ⚠️ The backdrop dismisses. A message with one way out is a message
    // somebody is trapped by, and this one interrupts a person carrying plates.
    Dialog(onDismissRequest = { state.value = null }) {
        Box(
            Modifier.fillMaxSize().clickable(
                interactionSource = androidx.compose.foundation.interaction.MutableInteractionSource(),
                indication = null,
            ) { state.value = null },
            contentAlignment = Alignment.Center,
        ) {
            Column(
                Modifier
                    .widthIn(max = 360.dp)
                    .glassSheet(c, RoundedCornerShape(26.dp))
                    .padding(24.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Box(
                    Modifier.size(56.dp)
                        .background(tone.copy(alpha = 0.15f), CircleShape),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(icon, null, tint = tone, modifier = Modifier.size(28.dp))
                }
                Text(
                    note.title, style = MaterialTheme.typography.titleMedium,
                    color = c.ink, textAlign = TextAlign.Center,
                )
                if (note.body != null) {
                    Text(
                        note.body, style = MaterialTheme.typography.bodyMedium,
                        color = c.inkSoft, textAlign = TextAlign.Center,
                    )
                }
                PrimaryButton(words.ok, Modifier.fillMaxWidth()) { state.value = null }
            }
        }
    }
}
