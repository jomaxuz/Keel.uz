package uz.keel.team.ui.screens

import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.camera.core.CameraSelector
import androidx.camera.core.ExperimentalGetImage
import androidx.camera.core.ImageAnalysis
import androidx.camera.core.Preview
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.content.ContextCompat
import androidx.lifecycle.compose.LocalLifecycleOwner
import com.google.mlkit.vision.barcode.BarcodeScannerOptions
import com.google.mlkit.vision.barcode.BarcodeScanning
import com.google.mlkit.vision.barcode.common.Barcode
import com.google.mlkit.vision.common.InputImage
import kotlinx.coroutines.launch
import uz.keel.design.Chip
import uz.keel.design.GhostButton
import uz.keel.design.GlassField
import uz.keel.design.KeelTheme
import uz.keel.design.PickerOption
import uz.keel.design.PickerRow
import uz.keel.design.PrimaryButton
import uz.keel.design.glass
import uz.keel.team.data.ApiError
import uz.keel.team.data.KeelApi
import uz.keel.team.data.LabelRow
import uz.keel.team.data.MarkItem
import uz.keel.team.data.Staff
import uz.keel.team.t

// Unpacking a delivery with the camera that is already in the room.
//
// ⚠️ **The scanner was a pistol wired to a counter, and the box is in the store
// room.** Scanning at goods-in is the whole point of the marking feature — a
// code that was never received is refused in front of a guest otherwise — but
// it could only be done at a machine with a scanner attached. So a delivery was
// carried to the counter, or scanned later from memory, which is the same as
// not scanning it.
//
// ⚠️ **Two halves, two permissions, and they are opposites.** Scanning reads
// what the **state** issued and can never invent one; printing invents what the
// **shop** owns. A screen that called both "markirovka" would teach a room that
// we can print state codes — which is the one thing nobody may do, and the
// mistake this codebase has already written down twice (labels.go,
// DECISIONS → «Yorliq»).

/** Whether this account scans marking codes. ⚠️ The permission, never the job
 *  title: `Staff.Position` is free text (models/staffrole.go). */
fun canScanHere(staff: Staff): Boolean = staff.perms.contains("marking")

/** Whether this account prints the shop's own labels. */
fun canLabelHere(staff: Staff): Boolean = staff.perms.contains("label")

@Composable
fun MarkScreen(
    api: KeelApi,
    staff: Staff,
    bottomInset: PaddingValues,
) {
    val c = KeelTheme.colors
    val canScan = canScanHere(staff)
    val canLabel = canLabelHere(staff)
    // ⚠️ The half they hold opens first. Somebody with one permission must not
    // meet a switcher whose other side refuses them.
    var half by remember { mutableStateOf(if (canScan) "scan" else "label") }

    Column(Modifier.fillMaxSize()) {
        Column(
            Modifier.statusBarsPadding().padding(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(
                if (half == "scan") t.mark.title else t.mark.labelsTitle,
                style = MaterialTheme.typography.titleLarge,
                color = c.ink,
            )
            if (canScan && canLabel) {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Chip(t.mark.title, on = half == "scan") { half = "scan" }
                    Chip(t.mark.labelsTitle, on = half == "label") { half = "label" }
                }
            }
        }
        if (half == "scan") {
            ScanHalf(api, bottomInset)
        } else {
            LabelHalf(api, bottomInset)
        }
    }
}

// ---- Scanning ----

@Composable
private fun ScanHalf(api: KeelApi, bottomInset: PaddingValues) {
    val c = KeelTheme.colors
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()

    var items by remember { mutableStateOf<List<MarkItem>>(emptyList()) }
    var itemId by remember { mutableStateOf("") }
    var held by remember { mutableIntStateOf(0) }
    /** What the camera has read and nobody has saved yet.
     *
     *  ⚠️ **Kept in the order it was read**, like the panel's: the person is
     *  comparing it against a box, top to bottom. */
    val codes = remember { mutableStateListOf<String>() }
    var scanning by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf(false) }
    var note by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }

    var granted by remember {
        mutableStateOf(
            ContextCompat.checkSelfPermission(ctx, Manifest.permission.CAMERA) ==
                PackageManager.PERMISSION_GRANTED,
        )
    }
    // ⚠️ **The words are read here and used below.** `t` is a composable
    // getter: reading it inside a callback or a plain function is a compile
    // error, and the reason is worth keeping — a lambda that read the
    // dictionary would keep the language it was created with, so a screen
    // already on display would go on speaking Uzbek after somebody switched to
    // Russian. Every screen in this app hoists its strings for that reason.
    val needCameraWord = t.mark.needCamera
    val nothingWord = t.mark.nothingScanned
    val savedWord = t.mark.saved
    val duplicateWord = t.mark.duplicate
    val badWord = t.mark.badCode
    val sendFailedWord = t.mark.sendFailed
    val loadFailed = t.mark.loadFailed

    // ⚠️ **Asked when the button is pressed, never at launch** — the rule the
    // clock-in screen already follows. A permission asked before anybody knows
    // what it is for is answered "no", and the way back is the system settings.
    val ask = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { ok ->
        granted = ok
        scanning = ok
        if (!ok) error = needCameraWord
    }
    LaunchedEffect(Unit) {
        try {
            items = api.markItems().items
            if (itemId.isEmpty()) itemId = items.firstOrNull()?.id.orEmpty()
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else loadFailed
        }
    }
    LaunchedEffect(itemId) {
        held = if (itemId.isEmpty()) 0 else {
            runCatching { api.markHeld(itemId).held }.getOrDefault(0)
        }
    }

    fun save() {
        if (codes.isEmpty()) {
            error = nothingWord
            return
        }
        busy = true
        error = ""
        scope.launch {
            try {
                val res = api.receiveMarks(itemId, codes.toList())
                // ⚠️ **What was refused stays on the screen, what was taken
                // leaves it.** A box with one unreadable sticker is
                // thirty-nine bottles filed and one still in somebody's hand,
                // and clearing the lot would lose exactly the one they have to
                // look at.
                val refused = (res.duplicates + res.bad).toSet()
                val left = codes.filter { it in refused }
                codes.clear()
                codes.addAll(left)
                note = buildString {
                    append(savedWord(res.added))
                    if (res.duplicates.isNotEmpty()) {
                        append(" · ").append(duplicateWord)
                            .append(": ").append(res.duplicates.size)
                    }
                    if (res.bad.isNotEmpty()) {
                        append(" · ").append(badWord)
                            .append(": ").append(res.bad.size)
                    }
                }
                held = runCatching { api.markHeld(itemId).held }.getOrDefault(held)
            } catch (e: Throwable) {
                error = if (e is ApiError) e.message else sendFailedWord
            } finally {
                busy = false
            }
        }
    }

    LazyColumn(
        Modifier.fillMaxSize().imePadding(),
        contentPadding = PaddingValues(
            start = 16.dp, end = 16.dp, top = 4.dp,
            bottom = bottomInset.calculateBottomPadding(),
        ),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        item {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                if (error.isNotEmpty()) {
                    Text(error, style = MaterialTheme.typography.bodyMedium, color = c.danger)
                }
                if (note.isNotEmpty()) {
                    Text(note, style = MaterialTheme.typography.bodyMedium, color = c.accent)
                }
                if (items.isEmpty()) {
                    Text(
                        t.mark.noItems,
                        style = MaterialTheme.typography.bodyMedium,
                        color = c.muted,
                    )
                } else {
                    Box(Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp))) {
                        // ⚠️ **What is in the box is the person's answer, not the
                        // code's.** The GTIN inside a DataMatrix names a product
                        // in the national catalogue, which is not this menu —
                        // matching them would be a mapping nobody has filled in.
                        PickerRow(
                            label = t.mark.pickItem,
                            options = items.map { PickerOption(it.id, it.name) },
                            selected = itemId,
                        ) { itemId = it }
                    }
                    Text(
                        t.mark.held(held),
                        style = MaterialTheme.typography.labelMedium,
                        color = c.muted,
                    )
                }
            }
        }

        if (scanning && granted) {
            item {
                Box(
                    Modifier.fillMaxWidth().height(260.dp)
                        .clip(RoundedCornerShape(18.dp)),
                ) {
                    CameraBox { code ->
                        // ⚠️ **The same sticker sits in the frame for many
                        // frames.** Without this one line a single bottle
                        // becomes forty scans, and the save then reports
                        // thirty-nine duplicates of itself — which reads as the
                        // box being wrong rather than the camera being fast.
                        if (code !in codes) codes.add(code)
                    }
                }
            }
        }

        item {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Box(Modifier.weight(1f)) {
                    if (scanning) {
                        GhostButton(t.mark.stop, Modifier.fillMaxWidth()) { scanning = false }
                    } else {
                        PrimaryButton(t.mark.scan) {
                            error = ""
                            if (granted) {
                                scanning = true
                            } else {
                                ask.launch(Manifest.permission.CAMERA)
                            }
                        }
                    }
                }
            }
        }

        item {
            Text(
                t.mark.scanned(codes.size),
                style = MaterialTheme.typography.labelMedium,
                color = c.muted,
            )
        }

        items(codes, key = { it }) { code ->
            Row(
                Modifier.fillMaxWidth().glass(c, RoundedCornerShape(16.dp))
                    .padding(horizontal = 12.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(
                    // The tail is what differs between two bottles of the same
                    // drink; the head is the same GTIN on every one of them.
                    code.takeLast(24),
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.ink,
                    modifier = Modifier.weight(1f),
                )
                GhostButton(t.common.cancel) { codes.remove(code) }
            }
        }

        if (codes.isNotEmpty()) {
            item {
                PrimaryButton(t.mark.save(codes.size), busy = busy) { save() }
            }
        }
    }
}

/** The camera, bound to this screen's lifecycle.
 *
 *  ⚠️ **DataMatrix only.** A bottle carries the state's DataMatrix and an
 *  ordinary EAN-13 side by side, and a scanner pointed at the wrong one is the
 *  commonest mistake at a counter — the panel's screen can only refuse it
 *  afterwards, by shape. A camera can simply not look at the other symbology,
 *  which is the one advantage it has over the pistol. */
@OptIn(ExperimentalGetImage::class)
@Composable
private fun CameraBox(onCode: (String) -> Unit) {
    val ctx = LocalContext.current
    val owner = LocalLifecycleOwner.current
    val view = remember { PreviewView(ctx) }
    val scanner = remember {
        BarcodeScanning.getClient(
            BarcodeScannerOptions.Builder()
                .setBarcodeFormats(Barcode.FORMAT_DATA_MATRIX)
                .build(),
        )
    }

    AndroidView(factory = { view }, modifier = Modifier.fillMaxSize())

    DisposableEffect(owner) {
        val future = ProcessCameraProvider.getInstance(ctx)
        val executor = ContextCompat.getMainExecutor(ctx)
        future.addListener({
            val provider = future.get()
            val preview = Preview.Builder().build().also {
                it.surfaceProvider = view.surfaceProvider
            }
            val analysis = ImageAnalysis.Builder()
                // ⚠️ Only the newest frame. Queued frames are a queue of stale
                // pictures of a bottle that has already been put down.
                .setBackpressureStrategy(ImageAnalysis.STRATEGY_KEEP_ONLY_LATEST)
                .build()
            analysis.setAnalyzer(executor) { proxy ->
                val image = proxy.image
                if (image == null) {
                    proxy.close()
                    return@setAnalyzer
                }
                scanner.process(
                    InputImage.fromMediaImage(image, proxy.imageInfo.rotationDegrees),
                ).addOnSuccessListener { found ->
                    for (b in found) {
                        val raw = b.rawValue ?: continue
                        if (raw.isNotBlank()) onCode(raw)
                    }
                }.addOnCompleteListener {
                    // ⚠️ Always, on every path: an unclosed proxy stops the
                    // analyser after a handful of frames and the preview simply
                    // freezes — with no error anywhere.
                    proxy.close()
                }
            }
            provider.unbindAll()
            runCatching {
                provider.bindToLifecycle(
                    owner, CameraSelector.DEFAULT_BACK_CAMERA, preview, analysis,
                )
            }
        }, executor)

        onDispose {
            runCatching { ProcessCameraProvider.getInstance(ctx).get().unbindAll() }
            scanner.close()
        }
    }
}

// ---- The shop's own labels ----

@Composable
private fun LabelHalf(api: KeelApi, bottomInset: PaddingValues) {
    val c = KeelTheme.colors
    val scope = rememberCoroutineScope()

    var rows by remember { mutableStateOf<List<LabelRow>>(emptyList()) }
    val copies = remember { mutableStateMapOf<String, String>() }
    var busy by remember { mutableStateOf(false) }
    var note by remember { mutableStateOf("") }
    var error by remember { mutableStateOf("") }
    var tick by remember { mutableIntStateOf(0) }

    // Hoisted for the reason the other half spells out: `t` is a composable
    // getter, and a lambda that read it would keep the language it was built
    // with.
    val loadFailed = t.mark.loadFailed
    val nothingWord = t.mark.nothingScanned
    val printedWord = t.mark.printed
    val newBarcodeWord = t.mark.newBarcode
    val sendFailedWord = t.mark.sendFailed

    LaunchedEffect(tick) {
        try {
            rows = api.labelCandidates().items
            error = ""
        } catch (e: Throwable) {
            error = if (e is ApiError) e.message else loadFailed
        }
    }

    fun print() {
        val wanted = rows.mapNotNull { row ->
            val n = copies[row.id]?.trim()?.toIntOrNull() ?: 0
            if (n > 0) row.id to n else null
        }.toMap()
        if (wanted.isEmpty()) {
            error = nothingWord
            return
        }
        busy = true
        error = ""
        scope.launch {
            try {
                val res = api.printLabels(wanted)
                note = buildString {
                    append(printedWord(res.queued))
                    // ⚠️ Named rather than counted: a code invented at this
                    // moment is a fact about the catalogue, and this is the only
                    // screen anybody will ever see it happen on.
                    if (res.barcoded.isNotEmpty()) {
                        append(" · ").append(newBarcodeWord(res.barcoded.joinToString(", ")))
                    }
                }
                copies.clear()
                tick += 1
            } catch (e: Throwable) {
                error = if (e is ApiError) e.message else sendFailedWord
            } finally {
                busy = false
            }
        }
    }

    LazyColumn(
        Modifier.fillMaxSize().imePadding(),
        contentPadding = PaddingValues(
            start = 16.dp, end = 16.dp, top = 4.dp,
            bottom = bottomInset.calculateBottomPadding(),
        ),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        item {
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                if (error.isNotEmpty()) {
                    Text(error, style = MaterialTheme.typography.bodyMedium, color = c.danger)
                }
                if (note.isNotEmpty()) {
                    Text(note, style = MaterialTheme.typography.bodyMedium, color = c.accent)
                }
                if (rows.isEmpty()) {
                    Text(
                        t.mark.labelsEmpty,
                        style = MaterialTheme.typography.bodyMedium,
                        color = c.muted,
                    )
                }
            }
        }

        items(rows, key = { it.id }) { row ->
            Row(
                Modifier.fillMaxWidth().glass(c, RoundedCornerShape(18.dp))
                    .padding(horizontal = 12.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Column(Modifier.weight(1f)) {
                    Text(row.name, style = MaterialTheme.typography.bodyLarge, color = c.ink)
                    Text(
                        when (row.reason) {
                            // ⚠️ First, because it is the one that stops a sale
                            // rather than misdescribing it: no code, no scan, no
                            // ring-up.
                            "noBarcode" -> t.mark.reasonNoBarcode
                            "never" -> t.mark.reasonNever
                            else -> t.mark.reasonPrice
                        },
                        style = MaterialTheme.typography.labelMedium,
                        color = if (row.reason == "noBarcode") c.danger else c.muted,
                    )
                }
                Box(Modifier.width(88.dp)) {
                    GlassField(
                        value = copies[row.id] ?: "",
                        onValueChange = { v -> copies[row.id] = v.filter { it.isDigit() } },
                        placeholder = t.mark.copies,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    )
                }
            }
        }

        if (rows.isNotEmpty()) {
            item { PrimaryButton(t.mark.print, busy = busy) { print() } }
        }
    }
}
