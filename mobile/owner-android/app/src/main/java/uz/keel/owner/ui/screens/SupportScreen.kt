package uz.keel.owner.ui.screens

import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.ArrowBackIosNew
import androidx.compose.material.icons.rounded.ExpandLess
import androidx.compose.material.icons.rounded.ExpandMore
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.rounded.SupportAgent
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
import kotlinx.coroutines.launch
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import uz.keel.design.GhostButton
import uz.keel.design.GlassField
import uz.keel.design.GlassIconButton
import uz.keel.design.KeelTheme
import uz.keel.design.ScreenHeader
import uz.keel.design.glass
import uz.keel.owner.LocalPrefs
import uz.keel.owner.data.HelpArticle
import uz.keel.owner.data.KeelApi
import uz.keel.owner.data.SupportMessage
import uz.keel.owner.data.SupportThread
import uz.keel.owner.data.searchHelp
import uz.keel.owner.t

// Search first, write to a person second.
//
// ⚠️ **The same order the panel uses, and the reason is measured in people's
// time.** Most questions are answered by one paragraph; a screen that opened
// straight into a chat turns every one of them into a person waiting for a
// person. The button through to an operator is never hidden — it is the second
// thing, not the missing thing.
//
// ⚠️ **The articles come from the server, not from this build.** They used to be
// bundled in the panel's own TypeScript, which a native application cannot
// import — and a Kotlin copy would be the copy that stops describing this build
// the first time one is corrected. `/admin/support/articles` is the one copy;
// the *ranking* is ported (`data/HelpSearch.kt`) and says so.

@Composable
fun SupportScreen(api: KeelApi, bottomInset: PaddingValues, onBack: () -> Unit) {
    val c = KeelTheme.colors
    val prefs = LocalPrefs.current
    val lang = prefs.lang.value.code
    var articles by remember { mutableStateOf<List<HelpArticle>>(emptyList()) }
    var query by remember { mutableStateOf("") }
    var open by remember { mutableStateOf<String?>(null) }

    // ⚠️ Loaded once per language and held: the base does not change while
    // somebody has the screen open, and a fetch behind every keystroke is a
    // control that stops feeling instant.
    //
    // ⚠️ An empty list is not an error state here — the screen falls back to its
    // "write to an operator" half, which is the useful half when the network is
    // the problem.
    LaunchedEffect(lang) {
        articles = runCatching { api.helpArticles(lang).articles }.getOrDefault(emptyList())
    }

    val hits = remember(articles, query) { searchHelp(articles, query) }
    val searching = query.isNotBlank()

    var draft by remember { mutableStateOf("") }
    var sending by remember { mutableStateOf(false) }
    var sendError by remember { mutableStateOf("") }
    var threads by remember { mutableStateOf<List<uz.keel.owner.data.SupportThread>>(emptyList()) }
    var messages by remember { mutableStateOf<Map<String, List<uz.keel.owner.data.SupportMessage>>>(emptyMap()) }
    var openThread by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val failed = t.support.failed

    suspend fun loadThreads() {
        threads = runCatching { api.supportThreads().threads }.getOrDefault(emptyList())
    }
    LaunchedEffect(Unit) { loadThreads() }

    // ⚠️ **Polled while the screen is open, never a socket.** The panel holds one
    // because it sits on a desk all day; a phone goes into a pocket and the
    // socket dies quietly — and a screen that says "connected" while nothing
    // arrives is worse than one that never claimed to be.
    LaunchedEffect(openThread) {
        val id = openThread ?: return@LaunchedEffect
        while (true) {
            runCatching { api.supportThread(id) }.onSuccess { v ->
                messages = messages + (id to v.messages)
            }
            kotlinx.coroutines.delay(5_000)
        }
    }

    fun send() {
        val text = draft.trim()
        if (text.isEmpty()) return
        sending = true
        sendError = ""
        scope.launch {
            try {
                // ⚠️ The same ranked articles the panel sends: the assistant
                // answers from these and nothing else, so a question asked from
                // the phone gets the same answer as one asked from the desk.
                val candidates = searchHelp(articles, text).take(4).map {
                    uz.keel.owner.data.AskArticle(it.article.title, it.article.body)
                }
                val res = api.supportAsk(openThread ?: "", text, lang, candidates)
                draft = ""
                openThread = res.threadId
                loadThreads()
            } catch (e: Throwable) {
                sendError = failed
            } finally { sending = false }
        }
    }

    Column(Modifier.fillMaxSize()) {
        ScreenHeader(
            title = t.support.title,
            leading = { GlassIconButton(Icons.Rounded.ArrowBackIosNew, onClick = onBack) },
        )
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState()).imePadding()
                .padding(horizontal = 16.dp)
                .padding(bottom = bottomInset.calculateBottomPadding() + 24.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text(t.support.lead, style = MaterialTheme.typography.bodyMedium, color = c.muted)

            GlassField(
                value = query, onValueChange = { query = it },
                placeholder = t.support.searchPlaceholder,
                trailing = {
                    Icon(Icons.Rounded.Search, null, tint = c.muted, modifier = Modifier.size(18.dp))
                },
            )

            if (searching) {
                Text(
                    if (hits.isEmpty()) t.support.noAnswer else t.support.found,
                    style = MaterialTheme.typography.labelMedium, color = c.muted,
                )
            }

            // ⚠️ With no question typed, the first few are shown rather than
            // nothing: an empty help screen reads as a help screen with no help
            // in it.
            val shown = if (searching) hits.map { it.article } else articles.take(6)
            shown.forEach { a ->
                Column(
                    Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp))
                        .clickable { open = if (open == a.id) null else a.id }
                        .padding(14.dp),
                    verticalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(a.title, Modifier.weight(1f), style = MaterialTheme.typography.bodyLarge, color = c.ink)
                        Icon(
                            if (open == a.id) Icons.Rounded.ExpandLess else Icons.Rounded.ExpandMore,
                            null, tint = c.muted, modifier = Modifier.size(18.dp),
                        )
                    }
                    if (open == a.id) {
                        Text(a.body, style = MaterialTheme.typography.bodyMedium, color = c.inkSoft)
                    }
                }
            }

            // ⚠️ **Never hidden, whatever the search found.** A person whose
            // question is not in the base is exactly the person who needs this,
            // and a button that appears only on failure is one they have to
            // provoke.
            Text(
                t.support.askOperator,
                style = MaterialTheme.typography.titleMedium, color = c.ink,
                modifier = Modifier.padding(top = 10.dp),
            )
            Row(
                verticalAlignment = Alignment.Bottom,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Box(Modifier.weight(1f)) {
                    GlassField(draft, { draft = it }, t.support.placeholder)
                }
                Box {
                    GhostButton(
                        t.support.askOperator.take(0).ifEmpty { "→" },
                        icon = Icons.Rounded.SupportAgent,
                        tint = c.accent,
                        enabled = draft.isNotBlank() && !sending,
                    ) { send() }
                }
            }
            if (sendError.isNotEmpty()) {
                Text(sendError, color = c.danger, style = MaterialTheme.typography.bodyMedium)
            }

            if (threads.isNotEmpty()) {
                Text(
                    t.support.history,
                    style = MaterialTheme.typography.titleMedium, color = c.ink,
                    modifier = Modifier.padding(top = 10.dp),
                )
                threads.forEach { th ->
                    ThreadCard(th, messages[th.id].orEmpty()) {
                        openThread = if (openThread == th.id) null else th.id
                    }
                }
            }
        }
    }
}

/** One conversation, opened on its messages.
 *
 *  ⚠️ **The assistant is labelled as itself.** An answer somebody believes came
 *  from a person, and did not, is the one they act on and then cannot ask a
 *  follow-up about. */
@Composable
private fun ThreadCard(
    thread: SupportThread,
    messages: List<SupportMessage>,
    onToggle: () -> Unit,
) {
    val c = KeelTheme.colors
    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp))
            .clickable(onClick = onToggle).padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(
                thread.subject.ifEmpty { thread.lastText },
                Modifier.weight(1f),
                style = MaterialTheme.typography.bodyLarge, color = c.ink,
            )
            Text(
                when (thread.status) {
                    "waiting" -> t.support.waiting
                    "closed" -> t.support.closed
                    else -> t.support.answered
                },
                style = MaterialTheme.typography.labelSmall,
                color = if (thread.status == "waiting") c.warn else c.muted,
            )
        }
        messages.forEach { m ->
            Column(Modifier.padding(top = 4.dp)) {
                Text(
                    when (m.from) {
                        "owner" -> t.support.you
                        "assistant" -> t.support.assistant
                        else -> m.author.ifEmpty { t.support.operator }
                    },
                    style = MaterialTheme.typography.labelSmall,
                    color = if (m.from == "owner") c.muted else c.accent,
                )
                Text(m.text, style = MaterialTheme.typography.bodyMedium, color = c.inkSoft)
            }
        }
    }
}
