package uz.keel.waiter.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.ArrowForward
import androidx.compose.material.icons.rounded.Storefront
import androidx.compose.material.icons.rounded.Visibility
import androidx.compose.material.icons.rounded.VisibilityOff
import androidx.compose.foundation.Image
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import uz.keel.waiter.R
import uz.keel.waiter.data.ApiError
import uz.keel.waiter.t
import uz.keel.waiter.ui.components.GhostButton
import uz.keel.waiter.ui.components.LangSwitch
import uz.keel.waiter.ui.components.GlassField
import uz.keel.waiter.ui.components.GlassIconButton
import uz.keel.waiter.ui.components.PrimaryButton
import uz.keel.waiter.ui.theme.KeelTheme
import uz.keel.waiter.ui.theme.glass

// Getting in: which restaurant, and who.

/** ⚠️ **The one screen that shows the app's own identity rather than the
 *  restaurant's.** After this, every screen belongs to the restaurant. */
@Composable
fun ServerScreen(onChosen: (String) -> Boolean) {
    val c = KeelTheme.colors
    var address by remember { mutableStateOf("") }
    var bad by remember { mutableStateOf(false) }

    Box(Modifier.fillMaxSize()) {
    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .imePadding()
            .navigationBarsPadding()
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(12.dp, Alignment.CenterVertically),
    ) {
        // ⚠️ **Keel's own mark, not a Material glyph.** This was
        // `Icons.Rounded.Storefront` — a generic shopfront, which says
        // "a shop" and not "this product". It is the one screen where the app
        // shows its own identity rather than the restaurant's, so a stock icon
        // is exactly the wrong thing on it. The drawable is the launcher icon's
        // foreground layer: the same mark on the home screen, the tab and here.
        Box(
            Modifier.size(74.dp).background(c.navy, RoundedCornerShape(24.dp)),
            contentAlignment = Alignment.Center,
        ) {
            Image(
                painterResource(R.drawable.ic_launcher_foreground),
                contentDescription = null,
                modifier = Modifier.size(66.dp),
            )
        }
        Text(t.appName, style = MaterialTheme.typography.headlineMedium, color = c.ink)
        Text(t.server.title, style = MaterialTheme.typography.bodyMedium, color = c.muted)

        Box(Modifier.widthIn(max = 360.dp)) {
            GlassField(
                value = address,
                onValueChange = { bad = false; address = it },
                placeholder = t.server.placeholder,
                keyboardOptions = KeyboardOptions(
                    capitalization = KeyboardCapitalization.None,
                    autoCorrectEnabled = false,
                    keyboardType = KeyboardType.Uri,
                ),
            )
        }
        // ⚠️ The short form is the example, because it is what somebody knows. A
        // placeholder showing the full URL teaches the wrong answer to everybody
        // who reads it.
        Text(
            t.server.hint, style = MaterialTheme.typography.labelMedium,
            color = c.muted, textAlign = TextAlign.Center,
        )
        if (bad) Text(t.server.bad, color = c.danger, style = MaterialTheme.typography.labelMedium)

        Box(Modifier.widthIn(max = 360.dp)) {
            PrimaryButton(t.server.next, icon = Icons.Rounded.ArrowForward) {
                if (!onChosen(address)) bad = true
            }
        }
    }
    // ⚠️ **After the column, not before it.** Inside a Box the last child is
    // drawn on top and is the one that receives touches — and the column above
    // fills the screen and scrolls, so a switch declared first is painted, looks
    // pressable, and silently swallows every tap into the scroll behind it.
    LangSwitch(Modifier.align(Alignment.TopEnd).statusBarsPadding().padding(12.dp))
    }
}

@Composable
fun LoginScreen(
    address: String,
    onSignIn: suspend (String, String) -> Unit,
    onForget: () -> Unit,
) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()
    var username by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var show by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    // ⚠️ Read in composition, not inside `submit`: the dictionary is a
    // composable read, and a network failure has to be worded in the language
    // that is on screen now.
    val offlineWording = "${t.offline.title} — ${t.common.retry.lowercase()}"

    fun submit() {
        busy = true
        error = ""
        scope.launch {
            try {
                onSignIn(username.trim(), password)
            } catch (e: ApiError) {
                // ⚠️ The server's own words. It tells a wrong password apart
                // from a switched-off account, and those send somebody to two
                // different people.
                error = e.message
                busy = false
            } catch (e: Throwable) {
                // ⚠️ **And a network failure is neither.** "Could not sign in"
                // under a correct password is what makes somebody type it a
                // third time — the same distinction the launch makes, said in
                // one line because here there is only one line to say it in.
                error = offlineWording
                busy = false
            }
        }
    }

    Box(Modifier.fillMaxSize()) {
    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .imePadding()
            .navigationBarsPadding()
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(12.dp, Alignment.CenterVertically),
    ) {
        Text(t.login.title, style = MaterialTheme.typography.headlineMedium, color = c.ink)
        Row(
            Modifier.glass(c, RoundedCornerShape(999.dp)).padding(horizontal = 12.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            Icon(Icons.Rounded.Storefront, null, tint = c.muted, modifier = Modifier.size(14.dp))
            Text(address, style = MaterialTheme.typography.labelMedium, color = c.muted)
        }

        Box(Modifier.widthIn(max = 360.dp)) {
            GlassField(
                value = username, onValueChange = { username = it },
                placeholder = t.login.username,
                keyboardOptions = KeyboardOptions(
                    capitalization = KeyboardCapitalization.None, autoCorrectEnabled = false,
                ),
            )
        }
        Box(Modifier.widthIn(max = 360.dp)) {
            GlassField(
                value = password, onValueChange = { password = it },
                placeholder = t.login.password,
                keyboardOptions = KeyboardOptions(
                    capitalization = KeyboardCapitalization.None,
                    keyboardType = KeyboardType.Password,
                ),
                visualTransformation = if (show) VisualTransformation.None else PasswordVisualTransformation(),
                // ⚠️ A password typed on a phone, in a dining room, by somebody
                // in a hurry — the eye is what stops the third failed attempt.
                trailing = {
                    GlassIconButton(
                        if (show) Icons.Rounded.VisibilityOff else Icons.Rounded.Visibility,
                    ) { show = !show }
                },
            )
        }

        if (error.isNotEmpty()) {
            Text(
                error, color = c.danger, textAlign = TextAlign.Center,
                style = MaterialTheme.typography.bodyMedium,
            )
        }

        Box(Modifier.widthIn(max = 360.dp)) {
            PrimaryButton(t.login.submit, busy = busy, enabled = !busy) { submit() }
        }
        GhostButton(t.login.other, tint = c.muted, onClick = onForget)
    }
    LangSwitch(Modifier.align(Alignment.TopEnd).statusBarsPadding().padding(12.dp))
    }
}
