package uz.keel.waiter.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.DeleteOutline
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import kotlinx.coroutines.launch
import uz.keel.waiter.data.ApiError
import uz.keel.waiter.data.Check
import uz.keel.waiter.data.CheckLine
import uz.keel.waiter.data.KeelApi
import uz.keel.waiter.data.money
import uz.keel.waiter.t
import uz.keel.waiter.ui.components.GhostButton
import uz.keel.waiter.ui.components.GlassField
import uz.keel.waiter.ui.components.GlassStepper
import uz.keel.waiter.ui.components.PrimaryButton
import uz.keel.waiter.ui.theme.KeelTheme
import uz.keel.waiter.ui.theme.glass

// Correcting a line that has already been added.
//
// ⚠️ **Without this the app can make a mistake it cannot fix.** Adding a dish is
// one tap, so mis-taps are common — and the only way back would be a walk to the
// till. An app that creates errors and sends somebody else to correct them adds
// work to a shift rather than removing it.
//
// ⚠️ **Fired and unfired are two different acts, and the screen says which.**
// Before the kitchen has seen a line, removing it is a typo being corrected and
// costs nothing. Afterwards the food has been cooked, somebody paid for it in
// ingredients, and taking it off the bill is a write-off — the server asks for a
// reason and may ask for a manager's code. That is the oldest way money leaves a
// restaurant, and the difference is not ours to blur.

@Composable
fun LineDialog(
    api: KeelApi,
    checkId: String,
    line: CheckLine,
    onDone: (Check) -> Unit,
    onClose: () -> Unit,
) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()
    var qty by remember { mutableIntStateOf(line.qty) }
    var comment by remember { mutableStateOf(line.comment ?: "") }
    var reason by remember { mutableStateOf("") }
    var pin by remember { mutableStateOf("") }
    var needPin by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }

    val fired = line.fired
    val failed = t.line.failed

    fun run(work: suspend () -> Check) {
        busy = true
        error = ""
        scope.launch {
            try {
                onDone(work())
                onClose()
            } catch (e: ApiError) {
                // ⚠️ **A refusal that asks for a manager is not a failure.** The
                // server decides who may write off cooked food — the permission
                // may have changed since this person signed in — and the answer
                // is a code, not an error message. Same shape the till's override
                // dialog takes.
                if (e.status == 403 || e.status == 409) needPin = true
                error = e.message
                busy = false
            } catch (e: Throwable) {
                error = failed
                busy = false
            }
        }
    }

    Dialog(onDismissRequest = onClose) {
        Column(
            Modifier
                .widthIn(max = 400.dp)
                .imePadding()
                .glass(c, RoundedCornerShape(26.dp), strong = true)
                .padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(line.name, style = MaterialTheme.typography.titleMedium, color = c.ink)
            Text(
                "${money(line.price)} · ${if (fired) t.line.fired else t.check.pending}",
                style = MaterialTheme.typography.labelMedium, color = c.muted,
            )

            // ⚠️ Quantity only while the kitchen has not seen it. After that the
            // ticket at the pass names a number, and changing it here would leave
            // the paper and the screen disagreeing about one dish.
            if (!fired) {
                Row(
                    Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.Center,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    GlassStepper(
                        qty, enabled = !busy, removeAtZero = false,
                        onMinus = { qty = maxOf(1, qty - 1) },
                        onPlus = { qty = minOf(99, qty + 1) },
                    )
                }
                GlassField(comment, { comment = it }, t.line.commentPlaceholder)
            }

            // ⚠️ The reason is asked **before** the button, not after it: a dialog
            // that refuses on press has already cost the tap, and the server
            // requires one on a fired line.
            if (fired) GlassField(reason, { reason = it }, t.line.reasonPlaceholder)

            if (needPin) {
                GlassField(
                    pin, { pin = it }, t.line.pinPlaceholder,
                    keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(
                        keyboardType = KeyboardType.NumberPassword,
                    ),
                    visualTransformation = PasswordVisualTransformation(),
                )
            }

            if (error.isNotEmpty()) {
                Text(error, color = c.danger, style = MaterialTheme.typography.bodyMedium)
            }

            Row(horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically) {
                GhostButton(
                    if (fired) t.line.writeOff else t.line.remove,
                    icon = Icons.Rounded.DeleteOutline, tint = c.danger, enabled = !busy,
                    modifier = if (fired) Modifier.weight(1f) else Modifier,
                ) {
                    run {
                        api.voidLine(
                            checkId, line.lineId,
                            reason = if (fired) reason.trim() else "",
                            pin = pin,
                        )
                    }
                }

                if (!fired) {
                    PrimaryButton(t.line.save, Modifier.weight(1f), enabled = !busy, busy = busy) {
                        run {
                            var out: Check? = null
                            if (qty != line.qty) out = api.lineQty(checkId, line.lineId, qty)
                            if (comment.trim() != (line.comment ?: "")) {
                                out = api.commentLine(checkId, line.lineId, comment.trim())
                            }
                            // ⚠️ Nothing changed is not an error and not a
                            // request: the dialog is also how somebody looks at a
                            // line.
                            out ?: api.check(checkId)
                        }
                    }
                }
            }
        }
    }
}
