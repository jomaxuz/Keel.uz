package uz.keel.app.ui

import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.animation.scaleOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.Logout
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.core.content.ContextCompat
import kotlinx.coroutines.launch
import uz.keel.app.KeelApp
import uz.keel.app.RoleExit
import uz.keel.app.data.Kind
import uz.keel.app.data.ShellError
import uz.keel.app.data.Workspace
import uz.keel.courier.CourierEntry
import uz.keel.courier.location.LocationService
import uz.keel.design.GhostButton
import uz.keel.design.KeelTheme
import uz.keel.design.LocalNotice
import uz.keel.design.Note
import uz.keel.design.NoticeKind
import uz.keel.design.PrimaryButton
import uz.keel.design.ServerAddress
import uz.keel.design.glassSheet
import uz.keel.owner.OwnerEntry
import uz.keel.team.TeamEntry
import uz.keel.waiter.WaiterEntry

/** Where a tap from outside wants to go: a notification, or the courier's
 *  shift. Read from the intent's extras by `MainActivity`. */
data class Route(val workspace: Workspace?, val checkId: String?, val tab: String?)

/** What the shell is showing.
 *
 *  ⚠️ **The animation is keyed on this and draws from it** — the waiter app's
 *  lesson: animating on one key while drawing from the live state composes the
 *  incoming screen twice during the cross-fade, and twice is every request
 *  twice and a hold released by the copy being disposed. */
private sealed interface Stage {
    val key: String

    data class SignIn(val adding: Boolean) : Stage {
        override val key: String get() = if (adding) "add" else "signIn"
    }
    data object Hub : Stage {
        override val key: String get() = "hub"
    }
    data class Open(val ws: Workspace) : Stage {
        override val key: String get() = "open:" + ws.key
    }
}

/** Keel's own layer: who is signed in, which workspace is open, and the doors
 *  between them. Everything below a workspace is that role's, unchanged. */
@Composable
fun KeelRoot(app: KeelApp, route: Route?, onRouteConsumed: () -> Unit) {
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    val notice = LocalNotice.current
    val words = w
    val accounts by app.accounts.accounts
    val workspaces = remember(accounts) { app.accounts.workspaces() }

    /** Where a launch, or a sign-out, lands.
     *
     *  ⚠️ **Straight into the work whenever there is no real choice**: one
     *  workspace, or the one used last. The switcher is a door somebody opens,
     *  not a lobby everybody waits in each morning. */
    fun home(): Stage {
        // ⚠️ **A launch never asks "which section?".** The person signed in; the
        // app takes them straight to their work — the one they had open last, or
        // their primary one (a cashier to the floor, an owner to the numbers).
        // The switcher is a door on the settings screen, not a lobby everyone
        // waits in. Showing the hub on launch was the one wrong step this app
        // existed to remove, re-added at the top.
        if (app.accounts.accounts.value.isEmpty()) return Stage.SignIn(adding = false)
        val ws = app.accounts.workspaces()
        val target = app.accounts.last.value?.takeIf { it in ws } ?: ws.firstOrNull()
        return target?.let { Stage.Open(it) } ?: Stage.SignIn(adding = false)
    }

    var stage by remember { mutableStateOf(home()) }
    /** What the hub returns to on back — the workspace it was opened from. */
    var behind by remember { mutableStateOf<Stage?>(null) }
    var confirmLeave by remember { mutableStateOf(false) }
    // The workspace the quick switcher was opened from, or null while closed.
    var switchFrom by remember { mutableStateOf<Stage.Open?>(null) }
    var pendingCheck by remember { mutableStateOf<String?>(null) }
    var pendingTab by remember { mutableStateOf<String?>(null) }

    fun open(ws: Workspace) {
        app.accounts.remember(ws)
        behind = null
        stage = Stage.Open(ws)
    }

    // An account gone from under the screen — a sign-out, a refused token —
    // takes its workspaces with it.
    LaunchedEffect(workspaces) {
        val s = stage
        if (s is Stage.Open && s.ws !in workspaces) stage = home()
        if (s is Stage.Hub && workspaces.isEmpty()) stage = home()
    }

    // ⚠️ **A tap on a notification opens the workspace it was for.** An owner
    // reading the kitchen's "ready" on a waiter's shift, or the reverse, must
    // land where the message leads and not where they last were.
    LaunchedEffect(route, workspaces) {
        val r = route ?: return@LaunchedEffect
        val ws = r.workspace
        if (ws != null && ws in workspaces) {
            pendingCheck = r.checkId
            pendingTab = r.tab
            open(ws)
        }
        onRouteConsumed()
    }

    // The staff account's permissions decide which of its two sections it
    // opens, and they are changed in the office — so they are re-read once a
    // launch. ⚠️ Quietly: offline, the stored copy is good enough.
    LaunchedEffect(Unit) {
        val address = app.accounts.address ?: return@LaunchedEffect
        if (!app.accounts.signedIn(Kind.Staff)) return@LaunchedEffect
        runCatching { app.shellApi.staffMe(address) }.getOrNull()?.let { app.accounts.refresh(it) }
    }

    // ⚠️ **Notifications are asked for after signing in, never on the first
    // screen** — a prompt before anybody knows what the app is for is answered
    // "no". And registered for every account at once: see `PushHub`.
    var asked by rememberSaveable { mutableStateOf(false) }
    val ask = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        if (granted) app.push.retry() else app.push.denied()
    }
    val kinds = accounts.map { it.kind }
    val lang = app.look.lang.value.code
    LaunchedEffect(kinds, lang, app.push.nonce) {
        if (kinds.isEmpty()) return@LaunchedEffect
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            ContextCompat.checkSelfPermission(ctx, Manifest.permission.POST_NOTIFICATIONS) !=
            PackageManager.PERMISSION_GRANTED
        ) {
            if (!asked) {
                asked = true
                ask.launch(Manifest.permission.POST_NOTIFICATIONS)
            } else {
                app.push.denied()
            }
            return@LaunchedEffect
        }
        app.push.sync()
    }

    /** What leaving an account takes with it on this phone. */
    fun letGo(kind: Kind) {
        // ⚠️ The shift's location stops with the courier, or the next person
        // on this phone carries the last one's dot on the dispatcher's map.
        if (kind == Kind.Courier) LocationService.stop(ctx)
        // The room belongs to whoever was signed in — the next person must not
        // see the last one's tables for a second.
        if (kind == Kind.Staff) app.waiterIfMade()?.clear()
    }

    fun signOut(kind: Kind) {
        scope.launch {
            app.push.forget(kind)
            letGo(kind)
            app.accounts.drop(kind)
            notice.value = Note(NoticeKind.Ok, words.notes.signedOut)
            stage = home()
        }
    }

    fun expire(kind: Kind) {
        // ⚠️ Once. The role's session model is keyed on the token, so the
        // screen fading out builds a fresh one with no token — which is
        // refused too, and reports it again.
        if (!app.accounts.signedIn(kind)) return
        letGo(kind)
        app.accounts.drop(kind)
        notice.value = Note(NoticeKind.Warn, words.notes.expired, words.notes.expiredBody)
        stage = home()
    }

    fun leaveRestaurant() {
        confirmLeave = false
        scope.launch {
            for (a in app.accounts.accounts.value.toList()) {
                app.push.forget(a.kindOf)
                letGo(a.kindOf)
                app.accounts.drop(a.kindOf)
            }
            app.accounts.forgetServer()
            stage = Stage.SignIn(adding = false)
        }
    }

    suspend fun submit(address: String, username: String, password: String, adding: Boolean): SignInFailure? {
        if (ServerAddress.apiBase(address).isEmpty()) return SignInFailure.BadAddress
        val res = try {
            app.shellApi.signIn(address, username, password)
        } catch (e: ShellError) {
            return SignInFailure.Server(e.message)
        } catch (e: IllegalArgumentException) {
            return SignInFailure.BadAddress
        } catch (e: Throwable) {
            return SignInFailure.Offline
        }
        app.accounts.useServer(address)
        for (o in res.opened) {
            if (o.account.kindOf == Kind.Staff) app.waiterIfMade()?.clear()
            app.accounts.put(o.account, o.token)
        }
        // ⚠️ **Said, not swallowed.** The password matched an account the
        // server would not open — switched off, or held by another phone — and
        // "why is the courier missing?" has exactly this answer.
        res.refused.firstOrNull()?.let { (kind, why) ->
            notice.value = Note(NoticeKind.Warn, words.notes.refused(kind.label(words), why))
        }
        val all = app.accounts.workspaces()
        // The waiter's queue starts with the floor's first sign-in, as it did
        // with the app's launch in Keel Waiter.
        if (Workspace.Waiter in all) app.waiter
        // Straight into the work, never the picker: a new account opens on its
        // own primary workspace, a first sign-in on the primary one.
        val fresh = res.opened.flatMap { it.account.workspaces() }
        val target = if (adding) fresh.firstOrNull() else all.firstOrNull()
        if (target != null) open(target) else { behind = null; stage = Stage.Hub }
        return null
    }

    BackHandler(enabled = stage is Stage.Hub && behind != null) {
        behind?.let { stage = it }
        behind = null
    }
    BackHandler(enabled = stage == Stage.SignIn(adding = true)) { stage = Stage.Hub }

    Box(Modifier.fillMaxSize()) {
        AnimatedContent(
            targetState = stage,
            contentKey = { it.key },
            transitionSpec = {
                val ease = tween<Float>(320, easing = FastOutSlowInEasing)
                val into = targetState is Stage.Open
                if (into) {
                    // Into a workspace: it comes forward, the door recedes.
                    (fadeIn(tween(260)) + scaleIn(ease, initialScale = 0.94f))
                        .togetherWith(fadeOut(tween(180)) + scaleOut(ease, targetScale = 1.04f))
                } else {
                    // Back out to the doors: they come from behind, the room
                    // sinks away.
                    (fadeIn(tween(260)) + scaleIn(ease, initialScale = 1.05f))
                        .togetherWith(fadeOut(tween(180)) + scaleOut(ease, targetScale = 0.95f))
                }
            },
            label = "keel",
        ) { st ->
            when (st) {
                is Stage.SignIn -> SignInScreen(
                    initialAddress = app.accounts.address.orEmpty(),
                    fixedAddress = if (st.adding) app.accounts.address else null,
                    onBack = if (st.adding) ({ stage = Stage.Hub }) else null,
                    onSubmit = { a, u, p -> submit(a, u, p, st.adding) },
                )

                Stage.Hub -> HubScreen(
                    address = app.accounts.address.orEmpty(),
                    accounts = accounts,
                    workspaces = workspaces,
                    last = app.accounts.last.value,
                    onOpen = ::open,
                    onSignOut = { a -> signOut(a.kindOf) },
                    onAdd = { stage = Stage.SignIn(adding = true) },
                    onLeave = { confirmLeave = true },
                )

                is Stage.Open -> {
                    val account = app.accounts.accountFor(st.ws)
                    if (account != null) {
                        val exit = remember(st.ws, account) {
                            RoleExit(
                                signOut = { signOut(st.ws.kind) },
                                leaveRestaurant = { confirmLeave = true },
                                expired = { expire(st.ws.kind) },
                                card = {
                                    // ⚠️ **One workspace is a label, more than one
                                    // is a door.** A cashier who only has the floor
                                    // taps nothing here; someone who also counts the
                                    // store gets the quick switcher over their work,
                                    // not a page they leave it for.
                                    AccountCard(account, st.ws, canSwitch = workspaces.size > 1) {
                                        switchFrom = st
                                    }
                                },
                            )
                        }
                        WorkspaceView(app, st.ws, exit, pendingCheck, pendingTab) {
                            pendingCheck = null
                            pendingTab = null
                        }
                    }
                }
            }
        }

        val from = switchFrom
        if (from != null) {
            val account = app.accounts.accountFor(from.ws)
            if (account != null) {
                WorkspaceSwitcher(
                    current = from.ws,
                    account = account,
                    workspaces = workspaces,
                    onPick = { ws -> switchFrom = null; open(ws) },
                    onAdd = { switchFrom = null; stage = Stage.SignIn(adding = true) },
                    // The full page, for the rare things: signing an account out,
                    // another restaurant.
                    onManage = { switchFrom = null; behind = from; stage = Stage.Hub },
                    onDismiss = { switchFrom = null },
                )
            }
        }

        if (confirmLeave) {
            LeaveDialog(
                onYes = ::leaveRestaurant,
                onNo = { confirmLeave = false },
            )
        }
    }
}

/** One workspace, as its role draws it. */
@Composable
private fun WorkspaceView(
    app: KeelApp,
    ws: Workspace,
    exit: RoleExit,
    pendingCheck: String?,
    pendingTab: String?,
    onConsumed: () -> Unit,
) {
    val push = app.push.state
    when (ws) {
        Workspace.Waiter -> WaiterEntry(app.waiter, push, exit, pendingCheck, onConsumed)
        Workspace.Owner -> OwnerEntry(app.owner, push, exit, pendingTab, onConsumed)
        Workspace.Courier -> CourierEntry(app.courier, push, exit, pendingTab, onConsumed)
        Workspace.Team -> TeamEntry(app.team, push, exit, pendingTab, onConsumed)
    }
}

/** ⚠️ **Asked, because it is the one exit that costs a setup.** Signing one
 *  account out is the end of a day; forgetting the restaurant signs everybody
 *  out and empties the address, and a thumb that meant the first must not do
 *  the second. */
@Composable
private fun LeaveDialog(onYes: () -> Unit, onNo: () -> Unit) {
    val c = KeelTheme.colors
    val words = w
    Dialog(onDismissRequest = onNo) {
        Column(
            Modifier
                .widthIn(max = 340.dp)
                .glassSheet(c, RoundedCornerShape(28.dp))
                .padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(words.hub.leave, style = MaterialTheme.typography.titleLarge, color = c.ink)
            Text(words.hub.leaveConfirm, style = MaterialTheme.typography.bodyMedium, color = c.inkSoft)
            PrimaryButton(words.hub.leaveYes, icon = Icons.AutoMirrored.Rounded.Logout, onClick = onYes)
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically) {
                GhostButton(words.hub.cancel, tint = c.muted, onClick = onNo)
            }
        }
    }
}
