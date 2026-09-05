package uz.keel.owner.ui.screens

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Call
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.Star
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import kotlinx.coroutines.launch
import uz.keel.design.Chip
import uz.keel.design.GhostButton
import uz.keel.design.GlassField
import uz.keel.design.KeelTheme
import uz.keel.design.PrimaryButton
import uz.keel.design.ScreenHeader
import uz.keel.design.glass
import uz.keel.design.glassSheet
import uz.keel.owner.LocalPrefs
import uz.keel.owner.data.ApiError
import uz.keel.owner.data.Feedback
import uz.keel.owner.data.KeelApi
import uz.keel.owner.i18n.timeAgo
import uz.keel.owner.t

// What guests said, opened on the ones nobody has answered.
//
// ⚠️ **Publishing to the site stays in the panel.** That puts a guest's name and
// words on the internet, which is a decision made sitting down. The work here is
// ringing somebody this evening.
//
// ⚠️ **"Handled" carries what was done, and the server insists on it too.** A
// row marked handled with nothing written is a row nobody can act on a week
// later — and the person asking a week later is usually the owner.

@Composable
fun FeedbackScreen(api: KeelApi, bottomInset: PaddingValues) {
    val c = KeelTheme.colors
    val ctx = LocalContext.current
    val prefs = LocalPrefs.current
    val scope = rememberCoroutineScope()
    val branchId = prefs.branch.value

    // ⚠️ Opens on the unanswered ones: that is the work. "All" is a reading, and
    // a screen that opened on it would show a wall of five-star rows with the
    // one complaint somewhere inside.
    var onlyOpen by remember { mutableStateOf(true) }
    var list by remember { mutableStateOf<List<Feedback>?>(null) }
    var error by remember { mutableStateOf("") }
    var answering by remember { mutableStateOf<Feedback?>(null) }
    val loadFailed = t.common.loadFailed
    val failed = t.feedback.failed

    suspend fun load() {
        try {
            list = api.feedback(branchId, handled = !onlyOpen).feedback
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else loadFailed
            if (list == null) list = emptyList()
        }
    }
    LaunchedEffect(branchId, onlyOpen) { load() }

    val rows = list
    val average = rows?.filter { it.rating > 0 }?.map { it.rating }?.average()

    Column(Modifier.fillMaxSize()) {
        ScreenHeader(
            title = t.feedback.title,
            subtitle = average?.takeIf { !it.isNaN() }
                ?.let { t.feedback.average(String.format("%.1f", it)) },
        )
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Chip(t.feedback.filterOpen, onlyOpen) { onlyOpen = true }
            Chip(t.feedback.filterAll, !onlyOpen) { onlyOpen = false }
        }
        if (error.isNotEmpty()) {
            Text(
                error, color = c.danger, style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.padding(horizontal = 16.dp),
            )
        }

        if (rows == null) {
            Box(Modifier.fillMaxSize(), Alignment.Center) { CircularProgressIndicator(color = c.accent) }
            return@Column
        }

        LazyColumn(
            Modifier.fillMaxSize(),
            contentPadding = PaddingValues(
                16.dp, 4.dp, 16.dp, bottomInset.calculateBottomPadding() + 24.dp,
            ),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            items(rows, key = { it.id }) { f ->
                FeedbackCard(
                    f = f,
                    onCall = {
                        if (f.phone.isNotBlank()) {
                            runCatching {
                                ctx.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:${f.phone}")))
                            }
                        }
                    },
                    onAnswer = { answering = f },
                )
            }
            if (rows.isEmpty()) {
                item {
                    Column(
                        Modifier.fillMaxWidth().padding(vertical = 56.dp),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        Icon(Icons.Rounded.CheckCircle, null, tint = c.ready, modifier = Modifier.size(32.dp))
                        Text(
                            if (onlyOpen) t.feedback.empty else t.feedback.emptyAll,
                            style = MaterialTheme.typography.titleMedium, color = c.ink,
                        )
                        Text(
                            t.feedback.emptyHint, style = MaterialTheme.typography.bodyMedium,
                            color = c.muted, textAlign = TextAlign.Center,
                        )
                    }
                }
            }
        }
    }

    answering?.let { f ->
        AnswerDialog(
            onClose = { answering = null },
            onSave = { note ->
                answering = null
                scope.launch {
                    try {
                        api.handleFeedback(f.id, note)
                        load()
                    } catch (e: Throwable) {
                        error = if (e is ApiError) e.message else failed
                    }
                }
            },
        )
    }
}

@Composable
private fun FeedbackCard(f: Feedback, onCall: () -> Unit, onAnswer: () -> Unit) {
    val c = KeelTheme.colors
    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp)) {
            // ⚠️ Stars, and the colour carries the low ones: a two beside a five
            // in the same grey is a two nobody sees.
            repeat(5) { i ->
                Icon(
                    Icons.Rounded.Star, null,
                    tint = if (i < f.rating) {
                        if (f.rating <= 3) c.warn else c.accent
                    } else c.line,
                    modifier = Modifier.size(15.dp),
                )
            }
            Box(Modifier.weight(1f))
            Text(
                timeAgo(f.createdAt, t.common.timeAgo),
                style = MaterialTheme.typography.labelMedium, color = c.muted,
            )
        }
        if (f.name.isNotBlank() || f.phone.isNotBlank()) {
            Text(
                listOf(f.name, f.phone).filter { it.isNotBlank() }.joinToString(" · "),
                style = MaterialTheme.typography.bodyMedium, color = c.inkSoft,
            )
        }
        if (f.text.isNotBlank()) {
            Text(f.text, style = MaterialTheme.typography.bodyLarge, color = c.ink)
        }
        f.handledNote?.takeIf { it.isNotBlank() }?.let {
            Text(
                "${t.feedback.answered("")} · $it",
                style = MaterialTheme.typography.labelMedium, color = c.ready,
            )
        }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
            if (f.phone.isNotBlank()) {
                GhostButton(t.feedback.call, icon = Icons.Rounded.Call, onClick = onCall)
            }
            if (f.handledAt == null) {
                Box(Modifier.weight(1f)) {
                    PrimaryButton(t.feedback.answer, onClick = onAnswer)
                }
            }
        }
    }
}

@Composable
private fun AnswerDialog(onClose: () -> Unit, onSave: (String) -> Unit) {
    val c = KeelTheme.colors
    var note by remember { mutableStateOf("") }
    Dialog(onDismissRequest = onClose) {
        Column(
            Modifier.widthIn(max = 400.dp).imePadding()
                .glassSheet(c, RoundedCornerShape(26.dp)).padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(t.feedback.answer, style = MaterialTheme.typography.titleMedium, color = c.ink)
            // ⚠️ Says why the note matters, because otherwise it is filled with
            // "ok" — and "ok" a week later answers nothing.
            Text(t.feedback.answerHint, style = MaterialTheme.typography.bodyMedium, color = c.muted)
            GlassField(note, { note = it }, t.feedback.answerPlaceholder)
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically) {
                GhostButton(t.common.close, Modifier.weight(1f), onClick = onClose)
                Box(Modifier.weight(1f)) {
                    PrimaryButton(t.feedback.answerDo, enabled = note.isNotBlank()) { onSave(note.trim()) }
                }
            }
        }
    }
}
