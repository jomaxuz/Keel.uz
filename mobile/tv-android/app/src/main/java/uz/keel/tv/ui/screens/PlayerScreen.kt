package uz.keel.tv.ui.screens

import android.view.SurfaceView
import android.view.ViewGroup
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import androidx.media3.common.MediaItem
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.common.VideoSize
import androidx.media3.exoplayer.ExoPlayer
import androidx.compose.foundation.Image
import androidx.compose.ui.graphics.asImageBitmap
import android.graphics.BitmapFactory
import androidx.compose.runtime.produceState
import androidx.compose.ui.platform.LocalConfiguration
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import uz.keel.tv.PlayItem

// The loop on the wall.
//
// ⚠️ **Nothing here is ever interactive.** No controls, no progress bar, no tap
// target: a television in a dining room is driven by nobody, and the remote is
// in a drawer. What this decides is which item is on screen and when to move on
// — everything else is the file itself, drawn edge to edge.
//
// ⚠️ **Black behind everything, and letterboxed rather than cropped.** A picture
// cropped to fill a 16:9 wall loses whichever edge the price was on, and the
// restaurant will not find out: nobody looks at the screen from the office.
// Letterboxing on black is invisible on a dark set and never eats content.

@Composable
fun PlayerScreen(items: List<PlayItem>, modifier: Modifier = Modifier) {
    val context = LocalContext.current
    var index by remember { mutableIntStateOf(0) }

    // ⚠️ **The index is kept inside the list as the list changes.** A playlist
    // edited in the panel while the screen is on item five, down to three items,
    // would otherwise leave the television pointing past the end — a black
    // rectangle nothing in the app would ever move off.
    val safe = if (items.isEmpty()) 0 else index % items.size
    val current = items.getOrNull(safe)
    LaunchedEffect(safe) { if (index != safe) index = safe }

    // One player for the whole loop, its source replaced as videos come round.
    // ⚠️ Not one per item: creating a player per slide leaks a decoder on cheap
    // hardware, and the symptom is a set that plays fine for an hour and then
    // shows nothing until it is unplugged.
    val player = remember {
        ExoPlayer.Builder(context).build().apply {
            volume = 0f // A dining room has its own sound.
            repeatMode = Player.REPEAT_MODE_OFF
        }
    }
    DisposableEffect(Unit) { onDispose { player.release() } }

    var ratio by remember { mutableFloatStateOf(16f / 9f) }
    val count by rememberUpdatedState(items.size)

    DisposableEffect(player) {
        val listener = object : Player.Listener {
            override fun onVideoSizeChanged(size: VideoSize) {
                // ⚠️ The aspect comes from the file, not from an assumption. A
                // vertical clip somebody shot on a phone and uploaded is a real
                // case, and stretching it across a wall is the version of this
                // nobody would report as a bug — it just looks cheap.
                if (size.width > 0 && size.height > 0) {
                    ratio = size.width * size.pixelWidthHeightRatio / size.height
                }
            }

            override fun onPlaybackStateChanged(state: Int) {
                // ⚠️ With one video in the loop this lands on the same item, and
                // a player that has played to its end will not start again on
                // its own — so the effect below has to see a change. Replaying
                // here is the whole of it.
                if (state != Player.STATE_ENDED) return
                if (count <= 1) {
                    player.seekTo(0)
                    player.play()
                } else {
                    index += 1
                }
            }

            // ⚠️ **A video that fails to open must not stop the loop.** A file
            // truncated by a power cut mid-download, or a clip this particular
            // set's decoder will not take, would otherwise be a still black
            // screen for the rest of the day.
            override fun onPlayerError(error: PlaybackException) {
                if (count > 1) index += 1
            }
        }
        player.addListener(listener)
        onDispose { player.removeListener(listener) }
    }

    LaunchedEffect(current?.id, current?.path) {
        val item = current ?: return@LaunchedEffect
        if (item.kind != "video") {
            // ⚠️ Paused rather than left running: the audio is muted, but a
            // decoder working through a clip nobody can see is heat and, on a
            // cheap set, the reason the next video stutters.
            player.pause()
            return@LaunchedEffect
        }
        ratio = 16f / 9f
        player.setMediaItem(MediaItem.fromUri("file://${item.path}"))
        player.prepare()
        player.play()
    }

    // A picture moves on by its own clock; a video moves on when it ends.
    LaunchedEffect(current?.id, current?.kind) {
        val item = current ?: return@LaunchedEffect
        if (item.kind != "image") return@LaunchedEffect
        delay(item.seconds.coerceAtLeast(3) * 1000L)
        index += 1
    }

    Box(modifier.fillMaxSize().background(Color.Black), Alignment.Center) {
        val item = current ?: return@Box
        if (item.kind == "video") {
            Box(Modifier.fillMaxSize(), Alignment.Center) {
                AndroidView(
                    modifier = Modifier.aspectRatio(ratio),
                    factory = { ctx ->
                        SurfaceView(ctx).apply {
                            layoutParams = ViewGroup.LayoutParams(
                                ViewGroup.LayoutParams.MATCH_PARENT,
                                ViewGroup.LayoutParams.MATCH_PARENT,
                            )
                            player.setVideoSurfaceView(this)
                        }
                    },
                    // ⚠️ Re-attached on every recomposition that replaces the
                    // view: a surface the player no longer holds is a black
                    // rectangle where a video is playing, audibly and
                    // invisibly. (Silently, here — the volume is zero.)
                    update = { player.setVideoSurfaceView(it) },
                )
            }
        } else {
            // ⚠️ **Decoded off the main thread and downsampled to the panel.**
            // No image library: there is one picture on screen at a time and it
            // is a local file. But a photograph straight off a phone is 4000 px
            // wide and 48 MB decoded — on a television with a gigabyte of RAM
            // that is an out-of-memory crash, which on this device means a black
            // wall and a restaurant that cannot tell why.
            val metrics = LocalConfiguration.current
            val bitmap by produceState<android.graphics.Bitmap?>(null, item.path) {
                value = withContext(Dispatchers.IO) {
                    decodeScaled(item.path, metrics.screenWidthDp * 3, metrics.screenHeightDp * 3)
                }
            }
            if (bitmap != null) {
                Image(
                    bitmap = bitmap!!.asImageBitmap(),
                    contentDescription = null,
                    modifier = Modifier.fillMaxSize(),
                    contentScale = ContentScale.Fit,
                )
            }
        }
    }
}

/** One picture, no larger than the panel needs it.
 *
 *  ⚠️ **Bounds first, then a power-of-two sample.** `BitmapFactory` allocates the
 *  full image otherwise, and the file that kills a cheap set is the one an owner
 *  uploaded straight from a phone camera without thinking about it — which is
 *  every file.
 *
 *  ⚠️ Three times the screen's dp rather than exactly it: `inSampleSize` halves,
 *  so the next step down from "slightly too big" is "visibly soft on a two-metre
 *  panel". Headroom is cheaper than a blurred promotion. */
private fun decodeScaled(path: String, maxWidth: Int, maxHeight: Int): android.graphics.Bitmap? =
    runCatching {
        val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        BitmapFactory.decodeFile(path, bounds)
        var sample = 1
        while (bounds.outWidth / (sample * 2) >= maxWidth &&
            bounds.outHeight / (sample * 2) >= maxHeight
        ) {
            sample *= 2
        }
        BitmapFactory.decodeFile(path, BitmapFactory.Options().apply { inSampleSize = sample })
    }.getOrNull()
