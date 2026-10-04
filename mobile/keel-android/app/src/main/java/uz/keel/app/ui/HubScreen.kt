package uz.keel.app.ui

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.Logout
import androidx.compose.material.icons.rounded.ChevronRight
import androidx.compose.material.icons.rounded.PersonAdd
import androidx.compose.material.icons.rounded.Storefront
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay
import uz.keel.app.data.Account
import uz.keel.app.data.Workspace
import uz.keel.design.GhostButton
import uz.keel.design.KeelTheme
import uz.keel.design.LangSwitch
import uz.keel.design.glass
import uz.keel.design.keelGradient
import uz.keel.design.softShadow
import java.util.Calendar

/** "Where are we working today?" — every workspace this phone's accounts open.
 *
 *  ⚠️ **Seen only by somebody with a choice to make.** One account with one
 *  workspace never meets this screen: the sign-in leads straight into it, and
 *  so does every launch. And a person with two is taken back to the one they
 *  had open last, so this is a place one comes *to* (from the card on any
 *  settings screen), not a gate one passes every morning.
 *
 *  ⚠️ **The last one used wears the gradient — and only it.** One orange
 *  gradient per screen is the design module's rule, and here it answers the
 *  question most people opening this have: "where was I?" */
@Composable
fun HubScreen(
    address: String,
    accounts: List<Account>,
    workspaces: List<Workspace>,
    last: Workspace?,
    onOpen: (Workspace) -> Unit,
    onSignOut: (Account) -> Unit,
    onAdd: () -> Unit,
    onLeave: () -> Unit,
) {
    val c = KeelTheme.colors
    val words = w
    val first = accounts.firstOrNull { it.kindOf == workspaces.firstOrNull()?.kind } ?: accounts.firstOrNull()
    val firstName = first?.name?.trim()?.split(' ')?.firstOrNull().orEmpty()

    val hour = remember { Calendar.getInstance().get(Calendar.HOUR_OF_DAY) }
    val greeting = when (hour) {
        in 5..11 -> words.hub.morning
        in 12..16 -> words.hub.day
        in 17..22 -> words.hub.evening
        else -> words.hub.night
    }

    Box(Modifier.fillMaxSize()) {
        Column(
            Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .statusBarsPadding()
                .navigationBarsPadding()
                .padding(horizontal = 18.dp),
        ) {
            Spacer(Modifier.height(64.dp))
            Staggered(0) {
                Column(Modifier.fillMaxWidth()) {
                    Text(
                        if (firstName.isNotEmpty()) "$greeting, $firstName" else greeting,
                        style = MaterialTheme.typography.headlineMedium,
                        fontWeight = FontWeight.SemiBold,
                        color = c.ink,
                    )
                    Spacer(Modifier.height(6.dp))
                    Text(words.hub.question, style = MaterialTheme.typography.bodyLarge, color = c.muted)
                }
            }
            Spacer(Modifier.height(22.dp))

            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                workspaces.forEachIndexed { i, ws ->
                    Staggered(i + 1) {
                        WorkspaceCard(ws, isLast = ws == last, onClick = { onOpen(ws) })
                    }
                }
            }

            Spacer(Modifier.height(28.dp))
            Staggered(workspaces.size + 1) {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text(
                        words.hub.accounts.uppercase(),
                        style = MaterialTheme.typography.labelMedium,
                        color = c.muted,
                        modifier = Modifier.padding(start = 6.dp),
                    )
                    Column(
                        Modifier.fillMaxWidth().glass(c, RoundedCornerShape(24.dp)),
                    ) {
                        accounts.forEachIndexed { i, a ->
                            if (i > 0) {
                                Box(
                                    Modifier.fillMaxWidth().padding(horizontal = 16.dp).height(1.dp)
                                        .background(c.line),
                                )
                            }
                            AccountRow(a) { onSignOut(a) }
                        }
                        Box(Modifier.fillMaxWidth().padding(horizontal = 16.dp).height(1.dp).background(c.line))
                        Row(
                            Modifier
                                .fillMaxWidth()
                                .clickable(onClick = onAdd)
                                .padding(horizontal = 16.dp, vertical = 15.dp),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(12.dp),
                        ) {
                            Box(
                                Modifier.size(40.dp).clip(CircleShape).background(c.accentSoft),
                                contentAlignment = Alignment.Center,
                            ) {
                                Icon(Icons.Rounded.PersonAdd, null, tint = c.accent, modifier = Modifier.size(20.dp))
                            }
                            Text(
                                words.hub.add,
                                style = MaterialTheme.typography.bodyLarge,
                                color = c.ink,
                                modifier = Modifier.weight(1f),
                            )
                            Icon(Icons.Rounded.ChevronRight, null, tint = c.muted, modifier = Modifier.size(20.dp))
                        }
                    }
                }
            }

            Spacer(Modifier.height(18.dp))
            Staggered(workspaces.size + 2) {
                Row(
                    Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.Center,
                ) {
                    GhostButton(
                        words.hub.leave,
                        icon = Icons.Rounded.Storefront,
                        tint = c.muted,
                        onClick = onLeave,
                    )
                }
            }
            Spacer(Modifier.height(28.dp))
        }

        // The restaurant and the language, above the scroll so they take taps.
        Row(
            Modifier.fillMaxWidth().statusBarsPadding().padding(horizontal = 14.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Row(
                Modifier
                    .weight(1f, fill = false)
                    .glass(c, RoundedCornerShape(999.dp))
                    .padding(horizontal = 12.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(7.dp),
            ) {
                Icon(Icons.Rounded.Storefront, null, tint = c.accent, modifier = Modifier.size(15.dp))
                Text(
                    address,
                    style = MaterialTheme.typography.labelMedium,
                    color = c.inkSoft,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Spacer(Modifier.weight(1f))
            LangSwitch()
        }
    }
}

/** One workspace, as a door.
 *
 *  ⚠️ **Presses in under the finger.** The card is the whole target, and a
 *  target that does not move when pressed is one people press twice. */
@Composable
private fun WorkspaceCard(ws: Workspace, isLast: Boolean, onClick: () -> Unit) {
    val c = KeelTheme.colors
    val words = w
    val space = ws.words(words)
    val interaction = remember { MutableInteractionSource() }
    val pressed by interaction.collectIsPressedAsState()
    val scale by animateFloatAsState(if (pressed) 0.97f else 1f, tween(140), label = "press")
    val shape = RoundedCornerShape(26.dp)

    Row(
        Modifier
            .fillMaxWidth()
            .scale(scale)
            .softShadow(shape, elevation = if (isLast) 6.dp else 2.dp, dark = c.dark)
            .glass(c, shape, strong = isLast)
            .clickable(interactionSource = interaction, indication = null, onClick = onClick)
            .padding(16.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        Box(
            Modifier
                .size(54.dp)
                .clip(RoundedCornerShape(18.dp))
                .background(if (isLast) keelGradient() else Brush.linearGradient(listOf(c.accentSoft, c.accentSoft))),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                ws.icon(), null,
                tint = if (isLast) c.onAccent else c.accent,
                modifier = Modifier.size(26.dp),
            )
        }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(3.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    space.title,
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.SemiBold,
                    color = c.ink,
                )
                if (isLast) {
                    Text(
                        words.hub.last,
                        style = MaterialTheme.typography.labelSmall,
                        color = c.accent,
                        modifier = Modifier
                            .clip(RoundedCornerShape(999.dp))
                            .background(c.accentSoft)
                            .padding(horizontal = 8.dp, vertical = 3.dp),
                    )
                }
            }
            Text(space.lead, style = MaterialTheme.typography.bodyMedium, color = c.muted)
        }
        Icon(Icons.Rounded.ChevronRight, null, tint = c.muted, modifier = Modifier.size(22.dp))
    }
}

@Composable
private fun AccountRow(a: Account, onSignOut: () -> Unit) {
    val c = KeelTheme.colors
    val words = w
    Row(
        Modifier.fillMaxWidth().padding(start = 16.dp, end = 8.dp, top = 12.dp, bottom = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Avatar(a.name, size = 40.dp)
        Column(Modifier.weight(1f)) {
            Text(
                a.name, style = MaterialTheme.typography.bodyLarge, color = c.ink,
                maxLines = 1, overflow = TextOverflow.Ellipsis,
            )
            Text(a.label(words), style = MaterialTheme.typography.labelMedium, color = c.muted)
        }
        GhostButton(
            words.hub.signOut,
            icon = Icons.AutoMirrored.Rounded.Logout,
            tint = c.muted,
            onClick = onSignOut,
        )
    }
}

/** Each block rises into place a beat after the one above it.
 *
 *  ⚠️ **Short, and once.** 40ms apart and done in a third of a second: long
 *  enough that the screen reads top to bottom, short enough that nobody waits
 *  on it — and it does not replay when the list changes. */
@Composable
private fun Staggered(index: Int, content: @Composable () -> Unit) {
    val p = remember { Animatable(0f) }
    LaunchedEffect(Unit) {
        delay(40L * index)
        p.animateTo(1f, tween(340, easing = FastOutSlowInEasing))
    }
    Box(
        Modifier
            .widthIn(max = 560.dp)
            .graphicsLayer {
                alpha = p.value
                translationY = (1f - p.value) * 28f
            },
    ) { content() }
}
