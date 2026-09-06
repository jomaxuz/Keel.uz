package uz.keel.tv

import android.util.Log
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import uz.keel.tv.data.KeelApi
import java.io.File
import java.net.HttpURLConnection
import java.net.URL

// What this television plays, and where it keeps it.
//
// ⚠️ **Every file is downloaded onto the set and played from disk.** Not an
// optimisation: a wall-mounted screen loops the same forty-megabyte clip every
// two minutes, all day, on the restaurant's own wifi — the same wifi the till
// and the waiters' phones are on. Streaming it would be a permanent load on the
// one network the business cannot afford to lose, to fetch bytes that never
// change.
//
// ⚠️ **And it is what makes the screen survive the internet.** A dining room
// keeps working when the connection does not; a television that went black
// every time the router blinked would be unplugged within a week. Once the files
// are on the set, the loop needs nothing from us to keep going.
//
// ⚠️ **The list is re-read only when the branch says it changed.** The heartbeat
// carries a version number; comparing it costs nothing, and a screen that
// re-downloaded the playlist every minute would ask a question whose answer is
// almost always "the same as last minute".

/** One thing on the wall, with the copy that is actually played. */
@Serializable
data class PlayItem(
    val id: String,
    /** `image` or `video`. */
    val kind: String,
    /** Where it came from, and the name the local copy is keyed by. */
    val url: String,
    val seconds: Int,
    val startsAt: String? = null,
    val endsAt: String? = null,
    /** The local file's absolute path. ⚠️ This is what is handed to the player,
     *  never the URL. */
    val path: String,
)

@Serializable
private data class Manifest(val version: Int, val items: List<PlayItem>)

/** The local name for one URL.
 *
 *  ⚠️ **Derived from the uploaded name, which is already random and unique**, so
 *  a replaced file is a different name and no stale copy can be served — the
 *  same property the uploads route relies on for its `immutable` header.
 *  Everything outside a safe alphabet is dropped rather than escaped: this
 *  string becomes a path, and a slide whose name walked out of the content
 *  directory would be a file this app deletes on the next prune. */
fun localNameFor(url: String): String {
    val last = url.substringBefore('?').substringAfterLast('/')
    val safe = last.filter { it.isLetterOrDigit() || it == '.' || it == '_' || it == '-' }
    return if (safe.isNotEmpty() && !safe.startsWith(".")) safe else ""
}

/** Whether a slide may be on screen now.
 *
 *  ⚠️ **The television applies this itself, and that is deliberate.** The server
 *  could trim the list — but a screen that has not reached the internet since
 *  Friday would then still be showing an offer that ended on Saturday, and an
 *  expired promotion on a wall is worse than a blank one: a guest asks for it at
 *  the till. The same rule as `models.TVSlidePlayable`. */
fun playableNow(item: PlayItem, now: Long): Boolean {
    parseInstant(item.startsAt)?.let { if (now < it) return false }
    parseInstant(item.endsAt)?.let { if (now > it) return false }
    return true
}

/**
 * The loop on this set: the files, the manifest beside them, and the one rule
 * about when a slide may play.
 *
 * @param dir where the media lives. ⚠️ **`filesDir`, never the cache.** Android
 * empties the cache when storage runs low, and a television that quietly lost
 * its playlist overnight is a black screen at opening time with nobody in the
 * building who knows why.
 */
class PlaylistStore(
    private val api: KeelApi,
    private val clock: Clock,
    private val dir: File,
) {

    private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }

    private val _items = MutableStateFlow<List<PlayItem>>(emptyList())
    val items: StateFlow<List<PlayItem>> = _items.asStateFlow()

    /** Files are arriving right now.
     *
     *  ⚠️ **Visible on purpose.** While a freshly paired screen downloaded its
     *  first two videos, the Expo build told the room "no content — go to the
     *  panel": an instruction to fix something that was working, on a set four
     *  minutes from playing. Downloading is a normal state and it needs to be a
     *  state somebody can see. */
    private val _downloading = MutableStateFlow(false)
    val downloading: StateFlow<Boolean> = _downloading.asStateFlow()

    /** The version the list on disk was built from, so an unchanged version
     *  asks nothing. */
    private var have: Int? = null
    private var syncing = false

    private fun contentDir(): File = dir.also { if (!it.exists()) it.mkdirs() }

    private fun manifestFile() = File(contentDir(), MANIFEST)

    /** Read what is already on this set, before anything is asked of the
     *  network — which is exactly the cold-boot-with-no-router case. */
    suspend fun loadStored() = withContext(Dispatchers.IO) {
        val stored = readManifest() ?: return@withContext
        have = stored.version
        _items.value = stored.items
    }

    private fun readManifest(): Manifest? {
        val file = manifestFile()
        if (!file.exists()) return null
        val parsed = runCatching {
            json.decodeFromString(Manifest.serializer(), file.readText())
        }.getOrNull() ?: return null
        // ⚠️ Only the items whose files are still there. A download interrupted
        // by a power cut leaves a manifest naming a file that does not exist,
        // and the player would sit on it — a still, black television that looks
        // broken and is not.
        return parsed.copy(items = parsed.items.filter { File(it.path).exists() })
    }

    private fun writeManifest(m: Manifest) {
        runCatching { manifestFile().writeText(json.encodeToString(Manifest.serializer(), m)) }
        // A failed write costs nothing this launch: the screen plays what is in
        // memory and downloads again on the next start. Nobody in the room could
        // act on it either way.
    }

    /** Fetch the branch's list if this is not the version we already have. */
    suspend fun ensure(version: Int?) {
        if (version == null) return
        if (have == version) return
        sync()
    }

    suspend fun sync() = withContext(Dispatchers.IO) {
        if (syncing) return@withContext
        syncing = true
        _downloading.value = true
        try {
            val res = api.playlist()
            clock.note(res.serverTime)

            val next = mutableListOf<PlayItem>()
            for (slide in res.slides) {
                val name = localNameFor(slide.url)
                if (name.isEmpty()) continue
                val file = File(contentDir(), name)
                if (!file.exists()) {
                    // ⚠️ One at a time, in order. A branch that just uploaded six
                    // videos would otherwise open six downloads across the
                    // restaurant's wifi at once — and the first item, the one
                    // about to be on screen, would arrive last.
                    //
                    // ⚠️ **Caught per item, and that is not tidiness.** In the
                    // Expo build one unreachable file threw out to the silent
                    // catch below and discarded the whole list — including the
                    // clips that had already downloaded — and the wall said
                    // "no content". One bad file should cost one slide.
                    if (!download(slide.url, file)) continue
                }
                next += PlayItem(
                    id = slide.id,
                    kind = slide.kind,
                    url = slide.url,
                    seconds = slide.seconds,
                    startsAt = slide.startsAt,
                    endsAt = slide.endsAt,
                    path = file.absolutePath,
                )
            }

            writeManifest(Manifest(res.version, next))
            prune(next)
            // ⚠️ Only when every slide the server listed actually landed.
            // Recording the version after a partial download would make the next
            // heartbeat say "nothing changed", and the missing clips would never
            // be retried — the loop would stay one item short until somebody
            // edited it in the panel.
            if (next.size == res.slides.size) have = res.version
            _items.value = next
        } catch (e: Throwable) {
            // ⚠️ **Silence, and the old list keeps playing.** A failed download
            // is not a reason to blank a wall: what is on the set is still the
            // restaurant's content, one revision behind. The next heartbeat
            // brings the version round again.
            Log.i(TAG, "playlist sync failed: ${e.message}")
        } finally {
            syncing = false
            _downloading.value = false
        }
    }

    /** One file onto the set. True when it landed.
     *
     *  ⚠️ **Written to a temporary name and moved into place.** A download cut
     *  off by a power failure otherwise leaves a half file under the name the
     *  manifest points at, and `exists()` says yes — a video the player opens,
     *  fails on, and skips for the rest of the television's life, because it is
     *  never downloaded again.
     *
     *  ⚠️ **`HttpURLConnection`, not the API client.** This is bytes to a file,
     *  not JSON: a forty-megabyte clip decoded into memory on a set with a
     *  gigabyte of RAM is the kind of failure that only shows up on the cheapest
     *  hardware, which is most of them. */
    private fun download(url: String, target: File): Boolean = runCatching {
        val part = File(target.parentFile, target.name + ".part")
        val conn = (URL(url).openConnection() as HttpURLConnection).apply {
            connectTimeout = 15_000
            readTimeout = 60_000
            instanceFollowRedirects = true
        }
        try {
            if (conn.responseCode !in 200..299) return false
            conn.inputStream.use { input -> part.outputStream().use { input.copyTo(it) } }
        } finally {
            conn.disconnect()
        }
        if (!part.renameTo(target)) {
            part.delete()
            return false
        }
        true
    }.getOrElse {
        Log.i(TAG, "download failed: $url — ${it.message}")
        false
    }

    /** Remove local files nothing in the list points at any more.
     *
     *  ⚠️ **After the new list is written, never before.** A television with 8 GB
     *  of storage fills up in a season of promotions otherwise — but deleting
     *  first would mean a failed download leaves the set with neither copy, which
     *  is the one outcome the room can see. */
    private fun prune(items: List<PlayItem>) {
        runCatching {
            val keep = items.map { localNameFor(it.url) }.toMutableSet()
            keep += MANIFEST
            contentDir().listFiles()?.forEach { f ->
                if (f.name !in keep) f.delete()
            }
        }
    }

    private companion object {
        /** ⚠️ A file beside the media, not the secure store: a playlist of sixty
         *  rows is far past what Android's keystore is meant to hold, and the
         *  failure there is silent. */
        const val MANIFEST = "playlist.json"
        const val TAG = "KeelTV"
    }
}
