package uz.keel.app.ui

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.tween
import androidx.compose.animation.expandVertically
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.automirrored.rounded.ArrowForward
import androidx.compose.material.icons.rounded.ErrorOutline
import androidx.compose.material.icons.rounded.Lock
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.Storefront
import androidx.compose.material.icons.rounded.Visibility
import androidx.compose.material.icons.rounded.VisibilityOff
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
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import uz.keel.design.GlassIconButton
import uz.keel.design.KeelTheme
import uz.keel.design.LangSwitch
import uz.keel.design.PrimaryButton
import uz.keel.design.glass

/** Why a sign-in did not happen — in words already chosen for the screen. */
sealed interface SignInFailure {
    data object BadAddress : SignInFailure
    data object Offline : SignInFailure
    /** The server's own sentence, or empty for "wrong login or password". */
    data class Server(val message: String) : SignInFailure
}

/** Keel's one door.
 *
 *  ⚠️ **Three fields on one screen, not a restaurant screen and then a login
 *  screen.** The single-role apps asked for the address first, on a page of its
 *  own, because it is set once per phone. Here it is still set once — it is
 *  remembered and drawn filled — but somebody signing in for the first time
 *  answers all three in one breath, and a page per question is two more
 *  "next"s between a new waiter and their first table.
 *
 *  ⚠️ **No role picker.** The person does not choose waiter, owner or courier;
 *  the server says which accounts this login opens (`/app/login`). Asking would
 *  invite the one wrong answer this app exists to remove.
 *
 *  @param fixedAddress set when adding an account: the restaurant is the one
 *  the phone is already in, and is shown rather than asked. */
@Composable
fun SignInScreen(
    initialAddress: String,
    fixedAddress: String?,
    onBack: (() -> Unit)?,
    onSubmit: suspend (address: String, username: String, password: String) -> SignInFailure?,
) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()
    val focus = LocalFocusManager.current
    var address by remember { mutableStateOf(initialAddress) }
    var username by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var show by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf(false) }
    var failure by remember { mutableStateOf<SignInFailure?>(null) }
    val userFocus = remember { FocusRequester() }
    val passFocus = remember { FocusRequester() }
    val words = w

    // ⚠️ A refused password shakes the card, the way a door handle does. The
    // message under it says why; the movement says *that*, to somebody whose
    // eyes are already on the button.
    val shake = remember { Animatable(0f) }
    LaunchedEffect(failure) {
        if (failure is SignInFailure.Server) {
            for (x in listOf(14f, -12f, 9f, -6f, 3f, 0f)) shake.animateTo(x, tween(55))
        }
    }

    // Arrives rather than appears: the mark first, then the card.
    val enter = remember { Animatable(0f) }
    LaunchedEffect(Unit) { enter.animateTo(1f, tween(520)) }

    fun submit() {
        if (busy) return
        val addr = fixedAddress ?: address
        if (username.isBlank() || password.isEmpty()) {
            failure = SignInFailure.Server(words.signIn.empty)
            return
        }
        focus.clearFocus()
        busy = true
        failure = null
        scope.launch {
            val f = onSubmit(addr.trim(), username.trim(), password)
            failure = f
            busy = false
        }
    }

    val message = when (val f = failure) {
        null -> null
        SignInFailure.BadAddress -> words.signIn.badAddress
        SignInFailure.Offline -> words.signIn.offline
        is SignInFailure.Server -> f.message.ifBlank { words.signIn.wrong }
    }

    Box(Modifier.fillMaxSize()) {
        Column(
            Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .imePadding()
                .statusBarsPadding()
                .navigationBarsPadding()
                .padding(horizontal = 22.dp, vertical = 24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            Spacer(Modifier.height(36.dp))
            Box(
                Modifier.graphicsLayer {
                    alpha = enter.value
                    translationY = (1f - enter.value) * 24f
                },
            ) { KeelMark(size = 78.dp, breathe = busy) }

            Spacer(Modifier.height(6.dp))
            Text(
                if (fixedAddress != null) words.signIn.addTitle else words.signIn.title,
                style = MaterialTheme.typography.headlineMedium,
                fontWeight = FontWeight.SemiBold,
                color = c.ink,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(8.dp))
            Text(
                if (fixedAddress != null) words.signIn.addLead else words.signIn.lead,
                style = MaterialTheme.typography.bodyMedium,
                color = c.muted,
                textAlign = TextAlign.Center,
                modifier = Modifier.widthIn(max = 340.dp),
            )
            Spacer(Modifier.height(26.dp))

            Column(
                Modifier
                    .widthIn(max = 420.dp)
                    .fillMaxWidth()
                    .graphicsLayer {
                        alpha = enter.value
                        translationY = (1f - enter.value) * 48f
                        translationX = shake.value * density
                    }
                    .glass(c, RoundedCornerShape(28.dp))
                    .padding(18.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                if (fixedAddress != null) {
                    Row(
                        Modifier
                            .glass(c, RoundedCornerShape(999.dp))
                            .padding(horizontal = 12.dp, vertical = 8.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        Icon(Icons.Rounded.Storefront, null, tint = c.accent, modifier = Modifier.size(16.dp))
                        Text(fixedAddress, style = MaterialTheme.typography.labelLarge, color = c.inkSoft)
                    }
                } else {
                    // ⚠️ The short form is the example, because it is what
                    // somebody knows; `.keel.uz` is drawn after it so the
                    // short form reads as complete.
                    KeelInput(
                        value = address,
                        onValueChange = { address = it; failure = null },
                        placeholder = words.signIn.restaurantHint,
                        icon = Icons.Rounded.Storefront,
                        suffix = if (address.isNotBlank() && !address.contains('.')) ".keel.uz" else null,
                        keyboardOptions = KeyboardOptions(
                            capitalization = KeyboardCapitalization.None,
                            autoCorrectEnabled = false,
                            keyboardType = KeyboardType.Uri,
                            imeAction = ImeAction.Next,
                        ),
                        keyboardActions = KeyboardActions(onNext = { userFocus.requestFocus() }),
                    )
                }
                KeelInput(
                    value = username,
                    onValueChange = { username = it; failure = null },
                    placeholder = words.signIn.username,
                    icon = Icons.Rounded.Person,
                    modifier = Modifier.focusRequester(userFocus),
                    keyboardOptions = KeyboardOptions(
                        capitalization = KeyboardCapitalization.None,
                        autoCorrectEnabled = false,
                        imeAction = ImeAction.Next,
                    ),
                    keyboardActions = KeyboardActions(onNext = { passFocus.requestFocus() }),
                )
                KeelInput(
                    value = password,
                    onValueChange = { password = it; failure = null },
                    placeholder = words.signIn.password,
                    icon = Icons.Rounded.Lock,
                    modifier = Modifier.focusRequester(passFocus),
                    keyboardOptions = KeyboardOptions(
                        capitalization = KeyboardCapitalization.None,
                        keyboardType = KeyboardType.Password,
                        imeAction = ImeAction.Go,
                    ),
                    keyboardActions = KeyboardActions(onGo = { submit() }),
                    visualTransformation = if (show) VisualTransformation.None else PasswordVisualTransformation(),
                    // ⚠️ A password typed on a phone, in a dining room, in a
                    // hurry — the eye is what stops the third failed attempt.
                    trailing = {
                        GlassIconButton(if (show) Icons.Rounded.VisibilityOff else Icons.Rounded.Visibility) {
                            show = !show
                        }
                    },
                )

                AnimatedVisibility(
                    visible = message != null,
                    enter = fadeIn() + expandVertically(),
                    exit = fadeOut() + shrinkVertically(),
                ) {
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .glass(c, RoundedCornerShape(16.dp), strong = true)
                            .padding(horizontal = 12.dp, vertical = 10.dp),
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                        verticalAlignment = Alignment.Top,
                    ) {
                        // ⚠️ The network in the warning colour, a refusal in the
                        // danger one: the first is not the person's fault, and
                        // red would tell them it is.
                        val tint = if (failure == SignInFailure.Offline) c.warn else c.danger
                        Icon(Icons.Rounded.ErrorOutline, null, tint = tint, modifier = Modifier.size(18.dp))
                        Text(message ?: "", style = MaterialTheme.typography.bodyMedium, color = c.ink)
                    }
                }

                Spacer(Modifier.height(2.dp))
                PrimaryButton(
                    words.signIn.submit,
                    icon = Icons.AutoMirrored.Rounded.ArrowForward,
                    busy = busy,
                    enabled = !busy,
                ) { submit() }
            }

            Spacer(Modifier.height(22.dp))
            Text(
                words.signIn.footnote,
                style = MaterialTheme.typography.labelMedium,
                color = c.muted,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(24.dp))
        }

        // ⚠️ **Drawn after the column**, so it is on top and receives the tap —
        // a switch declared first is painted, looks pressable, and silently
        // hands every tap to the scroll behind it.
        Row(
            Modifier
                .fillMaxWidth()
                .statusBarsPadding()
                .padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (onBack != null) {
                GlassIconButton(Icons.AutoMirrored.Rounded.ArrowBack, onClick = onBack)
            }
            Spacer(Modifier.weight(1f))
            LangSwitch()
        }
    }
}
