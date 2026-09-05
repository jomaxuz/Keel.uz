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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.ArrowBackIosNew
import androidx.compose.material.icons.rounded.ChevronRight
import androidx.compose.material.icons.rounded.ExpandLess
import androidx.compose.material.icons.rounded.ExpandMore
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.rounded.Send
import androidx.compose.material.icons.rounded.SupportAgent
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
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import uz.keel.design.GhostButton
import uz.keel.design.GlassField
import uz.keel.design.GlassIconButton
import uz.keel.design.KeelTheme
import uz.keel.design.ScreenHeader
import uz.keel.design.glass
import uz.keel.owner.LocalPrefs
import uz.keel.owner.data.AskArticle
import uz.keel.owner.data.HelpArticle
import uz.keel.owner.data.KeelApi
import uz.keel.owner.data.SupportMessage
import uz.keel.owner.data.SupportThread
import uz.keel.owner.data.searchHelp
import uz.keel.owner.t

// Search first, write to a person second.
//
// ⚠️ **The same order the panel uses, and it is measured in people's time.**
// Most questions are answered by one paragraph; a screen that opened straight
// into a chat turns every one of them into a person waiting for a person. The
// way through to an operator is never hidden — it is the second thing, not the
// missing thing.
//
// ⚠️ **The articles come from the server.** They used to be bundled in the
// panel's TypeScript, which a native application cannot import — and a Kotlin
// copy would be the copy that stops describing this build. The *ranking* is
// ported (`data/HelpSearch.kt`) and says so.

/** How the base is doing.
 *
 *  ⚠️ **Three states, because two of them produced the same lie.** With the
 *  articles not loaded, a search said "nothing found" — which reads as "there is
 *  no answer to your question" when the truth is "this phone has no articles to
 *  look in". One sends somebody away satisfied that nothing exists; the other
 *  sends them to the operator, which is where they belong. */
private enum class BaseState { Loading, Failed, Ready }

@Composable
fun SupportScreen(api: KeelApi, bottomInset: PaddingValues, onBack: () -> Unit) {
    var openThread by remember { mutableStateOf<String?>(null) }

    // ⚠️ **A page, not a block that expands in a list.** The conversation was an
    // accordion inside the help screen: opening it pushed the search away, and
    // pressing the same row again did not close it. A chat is somewhere you go
    // and come back from — which is also what the phone's back button already
    // means.
    if (openThread != null) {
        ThreadScreen(api, openThread!!, bottomInset) { openThread = null }
        return
    }
    HelpHome(api, bottomInset, onBack) { openThread = it }
}

@Composable
private fun HelpHome(
    api: KeelApi,
    bottomInset: PaddingValues,
    onBack: () -> Unit,
    onOpenThread: (String) -> Unit,
) {
    val c = KeelTheme.colors
    val prefs = LocalPrefs.current
    val lang = prefs.lang.value.code
    val scope = rememberCoroutineScope()

    var articles by remember { mutableStateOf<List<HelpArticle>>(emptyList()) }
    var base by remember { mutableStateOf(BaseState.Loading) }
    var query by remember { mutableStateOf("") }
    var open by remember { mutableStateOf<String?>(null) }
    var draft by remember { mutableStateOf("") }
    var sending by remember { mutableStateOf(false) }
    var sendError by remember { mutableStateOf("") }
    var threads by remember { mutableStateOf<List<SupportThread>>(emptyList()) }
    val failed = t.support.failed

    LaunchedEffect(lang) {
        base = BaseState.Loading
        runCatching { api.helpArticles(lang).articles }
            .onSuccess { articles = it; base = if (it.isEmpty()) BaseState.Failed else BaseState.Ready }
            .onFailure { base = BaseState.Failed }
    }
    LaunchedEffect(Unit) {
        threads = runCatching { api.supportThreads().threads }.getOrDefault(emptyList())
    }

    val hits = remember(articles, query) { searchHelp(articles, query) }
    val searching = query.isNotBlank()

    fun send() {
        val text = draft.trim()
        if (text.isEmpty()) return
        sending = true
        sendError = ""
        scope.launch {
            try {
                // ⚠️ The same ranked articles the panel sends: the assistant
                // answers from these and nothing else, so a question asked from
                // a phone gets the answer one asked from the desk does.
                val candidates = searchHelp(articles, text).take(4)
                    .map { AskArticle(it.article.title, it.article.body) }
                val res = api.supportAsk("", text, lang, candidates)
                draft = ""
                onOpenThread(res.threadId)
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

            when {
                base == BaseState.Loading ->
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        CircularProgressIndicator(Modifier.size(14.dp), color = c.accent, strokeWidth = 2.dp)
                        Text(t.common.loading, style = MaterialTheme.typography.labelMedium, color = c.muted)
                    }
                // ⚠️ Said plainly, and not as "nothing found": the base is what
                // is missing, not the answer. The operator half below still
                // works, which is the useful half when the network is the
                // problem.
                base == BaseState.Failed ->
                    Text(
                        t.support.baseUnavailable,
                        style = MaterialTheme.typography.bodyMedium, color = c.warn,
                    )
                searching ->
                    Text(
                        if (hits.isEmpty()) t.support.noAnswer else t.support.found,
                        style = MaterialTheme.typography.labelMedium, color = c.muted,
                    )
            }

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
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Box(Modifier.weight(1f)) {
                    GlassField(draft, { draft = it }, t.support.placeholder)
                }
                GhostButton(
                    "", icon = Icons.Rounded.Send, tint = c.accent,
                    enabled = draft.isNotBlank() && !sending,
                ) { send() }
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
                    Row(
                        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp))
                            .clickable { onOpenThread(th.id) }
                            .padding(14.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        Column(Modifier.weight(1f)) {
                            Text(
                                th.subject.ifEmpty { th.lastText },
                                style = MaterialTheme.typography.bodyLarge, color = c.ink,
                                maxLines = 2,
                            )
                            Text(
                                statusWord(th.status),
                                style = MaterialTheme.typography.labelSmall,
                                color = if (th.status == "waiting") c.warn else c.muted,
                            )
                        }
                        Icon(Icons.Rounded.ChevronRight, null, tint = c.muted, modifier = Modifier.size(18.dp))
                    }
                }
            }
        }
    }
}

@Composable
private fun statusWord(status: String): String = when (status) {
    "waiting" -> t.support.waiting
    "closed" -> t.support.closed
    else -> t.support.answered
}

/** One conversation, on its own page.
 *
 *  ⚠️ **Polled while it is open, never a socket.** The panel holds one because it
 *  sits on a desk all day; a phone goes into a pocket and the socket dies
 *  quietly — and a screen that says "connected" while nothing arrives is worse
 *  than one that never claimed to be. */
@Composable
private fun ThreadScreen(
    api: KeelApi,
    threadId: String,
    bottomInset: PaddingValues,
    onBack: () -> Unit,
) {
    val c = KeelTheme.colors
    val prefs = LocalPrefs.current
    val scope = rememberCoroutineScope()
    var messages by remember { mutableStateOf<List<SupportMessage>>(emptyList()) }
    var thread by remember { mutableStateOf<SupportThread?>(null) }
    var draft by remember { mutableStateOf("") }
    var sending by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    val failed = t.support.failed

    LaunchedEffect(threadId) {
        while (true) {
            runCatching { api.supportThread(threadId) }.onSuccess {
                thread = it.thread
                messages = it.messages
            }
            delay(5_000)
        }
    }

    Column(Modifier.fillMaxSize()) {
        ScreenHeader(
            title = thread?.subject?.takeIf { it.isNotBlank() } ?: t.support.askOperator,
            subtitle = thread?.status?.let { statusWord(it) },
            leading = { GlassIconButton(Icons.Rounded.ArrowBackIosNew, onClick = onBack) },
        )
        LazyColumn(
            Modifier.weight(1f),
            contentPadding = PaddingValues(16.dp, 4.dp, 16.dp, 8.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            items(messages, key = { it.id }) { m ->
                val mine = m.from == "owner"
                Column(
                    Modifier.fillMaxWidth(),
                    horizontalAlignment = if (mine) Alignment.End else Alignment.Start,
                ) {
                    Text(
                        // ⚠️ **The assistant is labelled as itself.** An answer
                        // somebody believes came from a person, and did not, is
                        // the one they act on and then cannot ask a follow-up
                        // about.
                        when (m.from) {
                            "owner" -> t.support.you
                            "assistant" -> t.support.assistant
                            else -> m.author.ifEmpty { t.support.operator }
                        },
                        style = MaterialTheme.typography.labelSmall,
                        color = if (mine) c.muted else c.accent,
                    )
                    Box(
                        Modifier.glass(c, RoundedCornerShape(16.dp)).padding(12.dp),
                    ) {
                        Text(m.text, style = MaterialTheme.typography.bodyMedium, color = c.ink)
                    }
                }
            }
        }
        if (error.isNotEmpty()) {
            Text(
                error, color = c.danger, style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.padding(horizontal = 16.dp),
            )
        }
        Row(
            Modifier.fillMaxWidth().imePadding()
                .padding(horizontal = 16.dp)
                .padding(bottom = bottomInset.calculateBottomPadding() + 12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Box(Modifier.weight(1f)) {
                GlassField(draft, { draft = it }, t.support.placeholder)
            }
            GhostButton(
                "", icon = Icons.Rounded.Send, tint = c.accent,
                enabled = draft.isNotBlank() && !sending,
            ) {
                val text = draft.trim()
                sending = true
                error = ""
                scope.launch {
                    try {
                        // ⚠️ No articles on a follow-up: the assistant answered
                        // the first question from the base, and a second search
                        // on "and what about the printer" would hand it four
                        // unrelated ones.
                        api.supportAsk(threadId, text, prefs.lang.value.code, emptyList())
                        draft = ""
                        runCatching { api.supportThread(threadId) }.onSuccess { messages = it.messages }
                    } catch (e: Throwable) {
                        error = failed
                    } finally { sending = false }
                }
            }
        }
    }
}
