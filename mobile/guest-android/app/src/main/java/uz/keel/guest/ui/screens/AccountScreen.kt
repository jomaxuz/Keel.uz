package uz.keel.guest.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import uz.keel.design.Chip
import uz.keel.design.GhostButton
import uz.keel.design.GlassField
import uz.keel.design.KeelTheme
import uz.keel.design.LangSwitch
import uz.keel.design.Money
import uz.keel.design.PrimaryButton
import uz.keel.design.ThemeChoice
import uz.keel.design.glass
import uz.keel.guest.LocalPrefs
import uz.keel.guest.data.ApiError
import uz.keel.guest.data.KeelApi
import uz.keel.guest.data.Loyalty
import uz.keel.guest.data.User
import uz.keel.guest.t

// The guest's own account, and the one screen that is optional.
//
// ⚠️ **Nothing above this screen needs it.** The menu, the basket and placing an
// order all work signed out — that is the design, not an oversight, because an
// app that asks for a phone number before showing a price gets one launch. What
// signing in adds is the guest's own history, their points and their favourites.
//
// ⚠️ **The hint says what signing in gives, never why it is required.** It is not
// required, and a screen that implied otherwise would be lying to somebody who
// has already ordered twice without it.

@Composable
fun AccountScreen(
    api: KeelApi,
    bottomInset: PaddingValues,
    onSignedIn: () -> Unit,
    onSignedOut: () -> Unit,
) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()
    val prefs = LocalPrefs.current

    var user by remember { mutableStateOf<User?>(null) }
    var loyalty by remember { mutableStateOf<Loyalty?>(null) }
    var error by remember { mutableStateOf("") }

    LaunchedEffect(api.signedIn()) {
        if (!api.signedIn()) {
            user = null
            return@LaunchedEffect
        }
        runCatching { user = api.me() }
        runCatching { loyalty = api.loyalty() }
    }

    LazyColumn(
        Modifier.fillMaxSize().imePadding(),
        contentPadding = PaddingValues(
            start = 16.dp,
            end = 16.dp,
            bottom = bottomInset.calculateBottomPadding(),
        ),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            Row(
                Modifier.statusBarsPadding().padding(top = 12.dp).fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    t.account.title,
                    Modifier.weight(1f),
                    style = MaterialTheme.typography.headlineMedium,
                    color = c.ink,
                )
                LangSwitch()
            }
        }

        if (error.isNotEmpty()) {
            item { Text(error, style = MaterialTheme.typography.bodyMedium, color = c.danger) }
        }

        val signedIn = user
        if (signedIn == null) {
            item {
                SignInCard(
                    api = api,
                    onFailed = { error = it },
                    onDone = {
                        user = it
                        error = ""
                        onSignedIn()
                        scope.launch { runCatching { loyalty = api.loyalty() } }
                    },
                )
            }
        } else {
            item {
                Column(
                    Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
                    verticalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    Text(
                        signedIn.firstName.ifBlank { signedIn.phone },
                        style = MaterialTheme.typography.titleLarge,
                        color = c.ink,
                    )
                    Text(
                        signedIn.phone,
                        style = MaterialTheme.typography.bodyMedium,
                        color = c.muted,
                    )
                }
            }
            // ⚠️ **The points card is drawn only where the restaurant runs
            // points.** A balance of zero with no explanation is a feature a
            // guest asks about and nobody at the counter has heard of.
            loyalty?.takeIf { it.enabled }?.let { l ->
                item {
                    Column(
                        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
                        verticalArrangement = Arrangement.spacedBy(4.dp),
                    ) {
                        Text(
                            t.account.points,
                            style = MaterialTheme.typography.bodyMedium,
                            color = c.muted,
                        )
                        Money(l.balance, style = uz.keel.design.BigNumberStyle, color = c.ink)
                        if (l.earnPercent > 0) {
                            Text(
                                t.account.pointsHint(uz.keel.design.qty(l.earnPercent)),
                                style = MaterialTheme.typography.labelMedium,
                                color = c.muted,
                            )
                        }
                    }
                }
            }
        }

        item {
            Column(
                Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Text(
                    t.account.theme,
                    style = MaterialTheme.typography.titleMedium,
                    color = c.ink,
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Chip(t.account.themeSystem, prefs.theme.value == ThemeChoice.System) {
                        prefs.setTheme(ThemeChoice.System)
                    }
                    Chip(t.account.themeLight, prefs.theme.value == ThemeChoice.Light) {
                        prefs.setTheme(ThemeChoice.Light)
                    }
                    Chip(t.account.themeDark, prefs.theme.value == ThemeChoice.Dark) {
                        prefs.setTheme(ThemeChoice.Dark)
                    }
                }
            }
        }

        if (signedIn != null) {
            item {
                GhostButton(t.account.signOut, Modifier.fillMaxWidth()) {
                    scope.launch {
                        // ⚠️ The phone is dropped **before** the token is
                        // cleared. The other order sends the request
                        // unauthenticated, the row stays, and the next person to
                        // hold this phone is told about somebody else's dinner.
                        onSignedOut()
                        api.signOut()
                        user = null
                        loyalty = null
                    }
                }
            }
        }
    }
}

/** Phone, then code. Two steps on one card.
 *
 *  ⚠️ **The number is not reformatted while it is typed.** Every guest here has
 *  their own habit — `+998`, `998`, a leading zero, spaces — and a field that
 *  rewrites what somebody is halfway through typing is a field they fight. The
 *  server normalises it, once, where the rule already lives. */
@Composable
private fun SignInCard(api: KeelApi, onFailed: (String) -> Unit, onDone: (User) -> Unit) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()
    var phone by remember { mutableStateOf("") }
    var name by remember { mutableStateOf("") }
    var code by remember { mutableStateOf("") }
    var sent by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf(false) }
    /** ⚠️ Counted down and shown. A "send again" that silently does nothing for
     *  a minute is a button somebody presses five times, and every press either
     *  costs the restaurant an SMS or teaches the guest the app is broken. */
    var wait by remember { mutableIntStateOf(0) }
    val failedWord = t.account.failed

    LaunchedEffect(wait) {
        if (wait > 0) {
            delay(1000)
            wait -= 1
        }
    }

    Column(
        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(20.dp)).padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Text(t.account.signIn, style = MaterialTheme.typography.titleLarge, color = c.ink)
        Text(t.account.signInHint, style = MaterialTheme.typography.bodyMedium, color = c.muted)

        GlassField(
            phone,
            { phone = it },
            t.account.phone,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
        )

        if (!sent) {
            PrimaryButton(t.account.sendCode, Modifier.fillMaxWidth(), busy = busy) {
                busy = true
                scope.launch {
                    try {
                        val res = api.requestCode(phone.trim())
                        sent = true
                        wait = res.retryAfter
                    } catch (e: Throwable) {
                        onFailed(if (e is ApiError) e.message else failedWord)
                    } finally {
                        busy = false
                    }
                }
            }
        } else {
            Text(
                t.account.codeSent(phone.trim()),
                style = MaterialTheme.typography.labelMedium,
                color = c.muted,
            )
            GlassField(
                code,
                { code = it },
                t.account.code,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword),
            )
            // ⚠️ Asked here rather than before the code: a returning guest never
            // sees this field, because the server only reads it for an account
            // that does not exist yet.
            GlassField(name, { name = it }, t.account.name)
            PrimaryButton(t.account.verify, Modifier.fillMaxWidth(), busy = busy) {
                busy = true
                scope.launch {
                    try {
                        val res = api.verifyCode(phone.trim(), code.trim(), name.trim())
                        api.signIn(res.token)
                        onDone(res.user)
                    } catch (e: Throwable) {
                        onFailed(if (e is ApiError) e.message else failedWord)
                    } finally {
                        busy = false
                    }
                }
            }
            GhostButton(
                if (wait > 0) t.account.resendIn(wait) else t.account.resend,
                Modifier.fillMaxWidth(),
                enabled = wait <= 0,
            ) {
                scope.launch {
                    runCatching { wait = api.requestCode(phone.trim()).retryAfter }
                }
            }
        }
    }
}
