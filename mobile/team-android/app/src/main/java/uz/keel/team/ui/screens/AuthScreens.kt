package uz.keel.team.ui.screens

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.ArrowForward
import androidx.compose.material.icons.rounded.Visibility
import androidx.compose.material.icons.rounded.VisibilityOff
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
import uz.keel.design.GhostButton
import uz.keel.design.GlassField
import uz.keel.design.GlassIconButton
import uz.keel.design.KeelTheme
import uz.keel.design.LangSwitch
import uz.keel.design.PrimaryButton
import uz.keel.team.R
import uz.keel.team.data.ApiError
import uz.keel.team.t

// Getting in: which restaurant, and who.
//
// ⚠️ **Two questions, two screens, and they are remembered separately.** This
// phone changes restaurant once, if ever; it signs in when a manager hands the
// app over. Collapsing them would make the rare action's setup the price of the
// common one.

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
            // ⚠️ Keel's own mark: this is the one screen where the app shows its
            // own identity rather than the restaurant's, and a stock glyph here
            // would be exactly the wrong thing.
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
            // ⚠️ The short form is the example, because it is what somebody
            // knows. A placeholder showing the full URL teaches the wrong answer
            // to everybody who reads it.
            Text(
                t.server.hint,
                style = MaterialTheme.typography.labelMedium,
                color = c.muted,
                textAlign = TextAlign.Center,
            )
            if (bad) {
                Text(t.server.bad, color = c.danger, style = MaterialTheme.typography.labelMedium)
            }
            Box(Modifier.widthIn(max = 360.dp)) {
                PrimaryButton(t.server.next, icon = Icons.Rounded.ArrowForward) {
                    if (!onChosen(address)) bad = true
                }
            }
        }
        // ⚠️ After the column, never before it: inside a Box the last child is
        // drawn on top and takes the touches, and a full-size scrolling column
        // declared later swallows every tap meant for this.
        //
        // ⚠️ **The language is chosen before the sign-in, not after.** The
        // setting lives on the settings screen, and the settings screen is
        // behind a password — so a phone handed to a Russian-speaking cook asked
        // for a restaurant address in Uzbek and a password in Uzbek, and the
        // only way out was through the screens they could not read.
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
    // ⚠️ Read in composition: a network failure has to be worded in the language
    // that is on screen now, and the dictionary is a composable read.
    val offlineWording = t.offline.title

    fun submit() {
        busy = true
        error = ""
        scope.launch {
            try {
                onSignIn(username.trim(), password)
            } catch (e: ApiError) {
                // The server spoke: wrong password, or an account switched off.
                error = e.message
            } catch (e: Throwable) {
                // ⚠️ **The request never arrived, and saying "could not sign in"
                // here blames the person for a network they cannot see.** They
                // then retype a password that was right the first time.
                error = offlineWording
            } finally {
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
            Text(address, style = MaterialTheme.typography.bodyMedium, color = c.muted)

            Box(Modifier.widthIn(max = 360.dp)) {
                GlassField(
                    value = username,
                    onValueChange = { error = ""; username = it },
                    placeholder = t.login.username,
                    keyboardOptions = KeyboardOptions(
                        capitalization = KeyboardCapitalization.None,
                        autoCorrectEnabled = false,
                    ),
                )
            }
            Box(Modifier.widthIn(max = 360.dp)) {
                GlassField(
                    value = password,
                    onValueChange = { error = ""; password = it },
                    placeholder = t.login.password,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                    visualTransformation =
                        if (show) VisualTransformation.None else PasswordVisualTransformation(),
                    trailing = {
                        // ⚠️ A courier types this with one hand at a counter.
                        // Being able to look at what was typed is the difference
                        // between one attempt and three.
                        GlassIconButton(
                            if (show) Icons.Rounded.VisibilityOff else Icons.Rounded.Visibility,
                        ) { show = !show }
                    },
                )
            }
            if (error.isNotEmpty()) {
                Text(
                    error,
                    color = c.danger,
                    style = MaterialTheme.typography.labelMedium,
                    textAlign = TextAlign.Center,
                )
            }
            Box(Modifier.widthIn(max = 360.dp)) {
                PrimaryButton(t.login.submit, busy = busy) { submit() }
            }
            GhostButton(t.login.other) { onForget() }
        }
        LangSwitch(Modifier.align(Alignment.TopEnd).statusBarsPadding().padding(12.dp))
    }
}
