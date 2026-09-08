package uz.keel.guest.ui.screens

import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.MyLocation
import androidx.compose.material.icons.rounded.Place
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
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import com.google.android.gms.location.LocationServices
import com.google.android.gms.maps.CameraUpdateFactory
import com.google.android.gms.maps.model.CameraPosition
import com.google.android.gms.maps.model.LatLng
import com.google.maps.android.compose.GoogleMap
import com.google.maps.android.compose.MapProperties
import com.google.maps.android.compose.MapUiSettings
import com.google.maps.android.compose.rememberCameraPositionState
import kotlinx.coroutines.FlowPreview
import kotlinx.coroutines.flow.debounce
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.launch
import uz.keel.design.GhostButton
import uz.keel.design.GlassField
import uz.keel.design.KeelTheme
import uz.keel.design.PrimaryButton
import uz.keel.design.glass
import uz.keel.guest.data.GeoPoint
import uz.keel.guest.data.Geocoder
import uz.keel.guest.data.Place
import uz.keel.guest.t

// Where the food is going.
//
// ⚠️ **The pin does not move — the map does.** A draggable marker means a
// fingertip covering the exact thing it is being placed on, and on a phone the
// last few metres are the ones that matter. So the pin is fixed at the centre of
// the screen and the map slides under it, which is what every delivery app here
// does and what guests already know how to use.
//
// ⚠️ **Nothing is asked before this screen.** Location permission is requested
// here, on a tap, for a button that says what it does — not at launch. A
// permission asked before anybody has said they want delivery is one most people
// refuse once and for all, and the refusal is permanent.
//
// ⚠️ **The map draws, and the search does not go to Google.** The Maps SDK for
// Android is not billed for map display; the Geocoding and Places APIs are, and
// a search box fires one call per keystroke. See data/Geocode.kt.

@OptIn(FlowPreview::class)
@Composable
fun MapPickerScreen(
    start: GeoPoint,
    /** Which engine to draw with and the key it needs — the restaurant's own
     *  setting, read at runtime so the app agrees with its site. */
    provider: String,
    mapKey: String,
    onClose: () -> Unit,
    onPicked: (GeoPoint) -> Unit,
) {
    val c = KeelTheme.colors
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val geocoder = remember { Geocoder() }
    val hasKey = mapKey.isNotEmpty()

    // ⚠️ **A restaurant with no maps key gets a sentence, not a grey grid.** The
    // key is their setting and its absence is not the guest's fault, so the text
    // points at the way round it — the note field and the phone number.
    if (!hasKey) {
        MapMissing(t.map.noKey, onClose)
        return
    }

    val camera = rememberCameraPositionState {
        position = CameraPosition.fromLatLngZoom(LatLng(start.lat, start.lng), 16f)
    }
    var address by remember { mutableStateOf(start.text) }
    var query by remember { mutableStateOf("") }
    var matches by remember { mutableStateOf<List<Place>>(emptyList()) }
    var denied by remember { mutableStateOf(false) }
    /** Where the map is pointing. ⚠️ Held here rather than read off one engine:
     *  Google reports through its camera state, the other two through a
     *  callback, and the panel below has to read one thing. */
    var centre by remember { mutableStateOf(start.lat to start.lng) }

    // ⚠️ **Reverse geocoding is debounced on the camera, not fired per frame.**
    // A drag emits a position every few milliseconds; one request each would
    // exhaust Nominatim's rate limit in a second and get the whole install base
    // blocked — which arrives as "search is broken" from every guest at once.
    LaunchedEffect(camera, provider) {
        if (provider != "google") return@LaunchedEffect
        snapshotFlow { camera.position.target }
            .debounce(600)
            .distinctUntilChanged()
            .collect { target -> centre = target.latitude to target.longitude }
    }

    // ⚠️ **One reverse-geocode path for all three engines, and it is debounced.**
    // A drag emits a position every few milliseconds; one request each would
    // exhaust Nominatim's rate limit in a second and get the whole install base
    // blocked — which arrives as "search is broken" from every guest at once.
    LaunchedEffect(centre) {
        kotlinx.coroutines.delay(600)
        geocoder.reverse(centre.first, centre.second)?.let { address = it }
    }

    val permission = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { granted ->
        denied = !granted
        if (granted) scope.launch { moveToMe(context, camera) }
    }

    // ⚠️ **Debounced, and only past three characters** — the same rule the site
    // follows, for the same service's sake.
    LaunchedEffect(query) {
        kotlinx.coroutines.delay(400)
        matches = geocoder.search(query)
    }

    Box(Modifier.fillMaxSize()) {
        // ---- The engine the restaurant chose ----
        //
        // ⚠️ **One `when`, and the coordinate order is each engine's own
        // business.** 2GIS wants [lng, lat] and the other two want them the
        // other way round; converting in the caller is how a restaurant ends up
        // in the Aral Sea. See MapEngines.kt.
        when (provider) {
            "yandex" -> {
                val ready = remember(mapKey) { YandexMap.prepare(context, mapKey) }
                if (ready) {
                    YandexPicker(start, { la, ln -> centre = la to ln })
                } else {
                    // ⚠️ Not a crash and not a grey grid: MapKit refuses to
                    // start without a valid key, and the guest is told what to
                    // do rather than shown a blank rectangle.
                    MapMissing(t.map.noKey, onClose)
                    return
                }
            }

            "google" -> GoogleMap(
                modifier = Modifier.fillMaxSize(),
                cameraPositionState = camera,
                properties = MapProperties(isMyLocationEnabled = false),
                // ⚠️ The stock controls are off: they sit under our own glass
                // panels, and Google's zoom buttons are a second way to do what
                // pinching already does.
                uiSettings = MapUiSettings(
                    zoomControlsEnabled = false,
                    myLocationButtonEnabled = false,
                    mapToolbarEnabled = false,
                ),
            )

            // 2GIS, and anything unrecognised: an empty provider is 2GIS, which
            // is what every install predating the setting runs on.
            else -> TwoGisPicker(mapKey, start, { la, ln -> centre = la to ln })
        }

        // The fixed pin. ⚠️ Lifted by half its height so its **point**, not its
        // centre, sits on the map's centre — the difference is about fifteen
        // metres at street zoom, which is a different building.
        Icon(
            Icons.Rounded.Place,
            contentDescription = null,
            tint = c.accent,
            modifier = Modifier.align(Alignment.Center).size(44.dp).padding(bottom = 22.dp),
        )

        Column(
            Modifier
                .align(Alignment.TopCenter)
                .statusBarsPadding()
                .padding(16.dp)
                .fillMaxWidth(),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            GlassField(query, { query = it }, t.map.search)
            matches.take(5).forEach { place ->
                Text(
                    place.text,
                    Modifier
                        .fillMaxWidth()
                        .glass(c, RoundedCornerShape(14.dp))
                        .clickable {
                            scope.launch {
                                camera.animate(
                                    CameraUpdateFactory.newLatLngZoom(
                                        LatLng(place.lat, place.lng),
                                        17f,
                                    ),
                                )
                            }
                            address = place.text
                            query = ""
                            matches = emptyList()
                        }
                        .padding(horizontal = 12.dp, vertical = 10.dp),
                    style = MaterialTheme.typography.bodyMedium,
                    color = c.ink,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }

        Column(
            Modifier
                .align(Alignment.BottomCenter)
                .navigationBarsPadding()
                .padding(16.dp)
                .fillMaxWidth()
                .glass(c, RoundedCornerShape(20.dp))
                .padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(
                address.ifBlank { t.common.loading },
                style = MaterialTheme.typography.bodyMedium,
                color = c.ink,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            if (denied) {
                Text(t.map.denied, style = MaterialTheme.typography.labelMedium, color = c.muted)
            }
            androidx.compose.foundation.layout.Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                GhostButton(t.map.myLocation, Modifier.weight(1f), icon = Icons.Rounded.MyLocation) {
                    val granted = ContextCompat.checkSelfPermission(
                        context,
                        Manifest.permission.ACCESS_COARSE_LOCATION,
                    ) == PackageManager.PERMISSION_GRANTED
                    if (granted) scope.launch { moveToMe(context, camera) }
                    else permission.launch(Manifest.permission.ACCESS_COARSE_LOCATION)
                }
                Box(Modifier.weight(1f)) {
                    PrimaryButton(t.map.confirm) {
                        onPicked(GeoPoint(centre.first, centre.second, address))
                    }
                }
            }
            GhostButton(t.common.back, Modifier.fillMaxWidth()) { onClose() }
        }
    }
}

/** Centre the map on the phone, when it will say where it is.
 *
 *  ⚠️ **`lastLocation`, not a fresh fix.** A cold GPS lock takes a minute in a
 *  courtyard, and a button that spins for a minute is a button people press
 *  again. The last known point is a street away at worst, and the guest is about
 *  to drag the map anyway. */
private suspend fun moveToMe(
    context: android.content.Context,
    camera: com.google.maps.android.compose.CameraPositionState,
) {
    runCatching {
        val client = LocationServices.getFusedLocationProviderClient(context)
        val task = client.lastLocation
        val location = kotlinx.coroutines.suspendCancellableCoroutine { cont ->
            task.addOnSuccessListener { cont.resume(it) { _, _, _ -> } }
                .addOnFailureListener { cont.resume(null) { _, _, _ -> } }
        }
        if (location != null) {
            camera.animate(
                CameraUpdateFactory.newLatLngZoom(
                    LatLng(location.latitude, location.longitude),
                    17f,
                ),
            )
        }
    }
}

/** What a guest sees when the restaurant has not set a map key up.
 *
 *  ⚠️ **A sentence, not a grey rectangle.** The key is the restaurant's setting
 *  and its absence is not the guest's fault, so the text points at the way round
 *  it — the note field and the phone number — rather than leaving somebody
 *  staring at an empty map wondering what they did wrong. */
@Composable
private fun MapMissing(message: String, onClose: () -> Unit) {
    val c = KeelTheme.colors
    Column(
        Modifier.fillMaxSize().padding(28.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp, Alignment.CenterVertically),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(message, style = MaterialTheme.typography.bodyLarge, color = c.ink)
        GhostButton(t.common.back) { onClose() }
    }
}
