package uz.keel.guest.ui.screens

import android.annotation.SuppressLint
import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Paint
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import com.google.android.gms.maps.CameraUpdateFactory
import com.google.android.gms.maps.model.BitmapDescriptorFactory
import com.google.android.gms.maps.model.CameraPosition as GCameraPosition
import com.google.android.gms.maps.model.LatLng
import com.google.android.gms.maps.model.LatLngBounds
import com.google.maps.android.compose.GoogleMap
import com.google.maps.android.compose.MapUiSettings
import com.google.maps.android.compose.Marker
import com.google.maps.android.compose.rememberCameraPositionState
import com.google.maps.android.compose.rememberMarkerState
import com.yandex.mapkit.MapKitFactory
import com.yandex.mapkit.geometry.BoundingBox
import com.yandex.mapkit.geometry.Geometry
import com.yandex.mapkit.geometry.Point
import com.yandex.mapkit.map.CameraPosition
import com.yandex.mapkit.map.PlacemarkMapObject
import com.yandex.mapkit.mapview.MapView
import com.yandex.runtime.image.ImageProvider
import uz.keel.guest.data.GeoPoint

// Where the courier is, for the guest waiting at the door.
//
// ⚠️ **Two dots and nothing to drag.** This map answers one question — "how
// far away is my food?" — so it shows the courier and the door and fits both
// on screen. It is not a picker: no pin, no search, no gestures that matter.
//
// ⚠️ **The courier's dot moves; the map does not reload.** The position is
// polled, and rebuilding a WebView or a MapView on every poll is a flash of
// grey every fifteen seconds — which reads as the courier vanishing. Each
// engine keeps its map and moves one marker. The camera is fitted once, on the
// first fix: a map that re-fits itself every poll fights the guest who zoomed.
//
// ⚠️ **Coordinate order stays each engine's own business** (MapEngines.kt):
// 2GIS is [lng, lat], the other two are lat-first.

@Composable
fun TrackMap(
    provider: String,
    mapKey: String,
    courier: GeoPoint,
    /** The delivery address, or null when the order has none on the map. */
    home: GeoPoint?,
    accent: Color,
    modifier: Modifier = Modifier,
) {
    when (provider) {
        "google" -> GoogleTrack(courier, home, accent, modifier)
        "yandex" -> {
            val ctx = LocalContext.current
            val ready = remember(mapKey) { YandexMap.prepare(ctx, mapKey) }
            if (ready) YandexTrack(courier, home, accent, modifier)
        }
        else -> TwoGisTrack(mapKey, courier, home, accent, modifier)
    }
}

@Composable
private fun GoogleTrack(courier: GeoPoint, home: GeoPoint?, accent: Color, modifier: Modifier) {
    val camera = rememberCameraPositionState {
        position = GCameraPosition.fromLatLngZoom(LatLng(courier.lat, courier.lng), 15f)
    }
    val courierMarker = rememberMarkerState(position = LatLng(courier.lat, courier.lng))
    LaunchedEffect(courier.lat, courier.lng) {
        courierMarker.position = LatLng(courier.lat, courier.lng)
    }
    val hue = remember(accent) {
        val hsv = FloatArray(3)
        android.graphics.Color.colorToHSV(accent.toArgb(), hsv)
        hsv[0]
    }
    GoogleMap(
        modifier = modifier,
        cameraPositionState = camera,
        uiSettings = MapUiSettings(zoomControlsEnabled = false, mapToolbarEnabled = false),
        onMapLoaded = {
            if (home != null) {
                val bounds = LatLngBounds.builder()
                    .include(LatLng(courier.lat, courier.lng))
                    .include(LatLng(home.lat, home.lng))
                    .build()
                runCatching { camera.move(CameraUpdateFactory.newLatLngBounds(bounds, 120)) }
            }
        },
    ) {
        Marker(state = courierMarker, icon = BitmapDescriptorFactory.defaultMarker(hue))
        if (home != null) {
            Marker(
                state = rememberMarkerState(position = LatLng(home.lat, home.lng)),
                icon = BitmapDescriptorFactory.defaultMarker(BitmapDescriptorFactory.HUE_AZURE),
            )
        }
    }
}

/** A filled dot with a white ring — the one marker shape both native engines
 *  can be handed as a bitmap. */
private fun dot(color: Int, sizePx: Int): Bitmap {
    val bmp = Bitmap.createBitmap(sizePx, sizePx, Bitmap.Config.ARGB_8888)
    val canvas = Canvas(bmp)
    val r = sizePx / 2f
    val paint = Paint(Paint.ANTI_ALIAS_FLAG)
    paint.color = android.graphics.Color.WHITE
    canvas.drawCircle(r, r, r, paint)
    paint.color = color
    canvas.drawCircle(r, r, r * 0.72f, paint)
    return bmp
}

@Composable
private fun YandexTrack(courier: GeoPoint, home: GeoPoint?, accent: Color, modifier: Modifier) {
    val ctx = LocalContext.current
    val px = (22 * ctx.resources.displayMetrics.density).toInt()
    val holder = remember { arrayOfNulls<PlacemarkMapObject>(1) }
    AndroidView(
        modifier = modifier,
        factory = { c ->
            MapView(c).also { view ->
                val map = view.mapWindow.map
                val objects = map.mapObjects
                holder[0] = objects.addPlacemark(
                    Point(courier.lat, courier.lng),
                    ImageProvider.fromBitmap(dot(accent.toArgb(), px)),
                )
                if (home != null) {
                    objects.addPlacemark(
                        Point(home.lat, home.lng),
                        ImageProvider.fromBitmap(dot(0xFF1A1614.toInt(), px)),
                    )
                    val box = BoundingBox(
                        Point(minOf(courier.lat, home.lat), minOf(courier.lng, home.lng)),
                        Point(maxOf(courier.lat, home.lat), maxOf(courier.lng, home.lng)),
                    )
                    val fit = map.cameraPosition(Geometry.fromBoundingBox(box))
                    // Pulled back a step so neither dot sits on the edge.
                    map.move(CameraPosition(fit.target, (fit.zoom - 0.8f).coerceAtMost(16f), 0f, 0f))
                } else {
                    map.move(CameraPosition(Point(courier.lat, courier.lng), 15f, 0f, 0f))
                }
                MapKitFactory.getInstance().onStart()
                view.onStart()
            }
        },
        update = { holder[0]?.geometry = Point(courier.lat, courier.lng) },
    )
    DisposableEffect(Unit) {
        onDispose { runCatching { MapKitFactory.getInstance().onStop() } }
    }
}

private fun css(color: Color): String = String.format("#%06X", 0xFFFFFF and color.toArgb())

private fun twoGisTrackHtml(key: String, courier: GeoPoint, home: GeoPoint?, accent: String): String = """
<!doctype html><html><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1,user-scalable=no">
<style>html,body,#map{height:100%;margin:0;padding:0}</style>
<script src="https://mapgl.2gis.com/api/js/v1"></script></head>
<body><div id="map"></div><script>
  var map = new mapgl.Map('map', { center: [${courier.lng}, ${courier.lat}], zoom: 15, key: '$key' });
  function dot(lng, lat, color) {
    return new mapgl.CircleMarker(map, { coordinates: [lng, lat], radius: 11,
      color: color, strokeWidth: 4, strokeColor: '#ffffff' });
  }
  var courier = dot(${courier.lng}, ${courier.lat}, '$accent');
  ${if (home != null) """
  dot(${home.lng}, ${home.lat}, '#1A1614');
  map.fitBounds({ southWest: [${minOf(courier.lng, home.lng)}, ${minOf(courier.lat, home.lat)}],
                  northEast: [${maxOf(courier.lng, home.lng)}, ${maxOf(courier.lat, home.lat)}] },
                { padding: { top: 60, bottom: 60, left: 60, right: 60 } });""" else ""}
  // Called from Kotlin on every poll: the marker moves, the map stays.
  function moveCourier(lng, lat) {
    courier.destroy();
    courier = dot(lng, lat, '$accent');
  }
</script></body></html>
"""

@SuppressLint("SetJavaScriptEnabled")
@Composable
private fun TwoGisTrack(key: String, courier: GeoPoint, home: GeoPoint?, accent: Color, modifier: Modifier) {
    val first = remember { courier }
    AndroidView(
        modifier = modifier,
        factory = { ctx ->
            WebView(ctx).apply {
                settings.javaScriptEnabled = true
                settings.domStorageEnabled = true
                webViewClient = WebViewClient()
                // ⚠️ The same real base URL as the picker: MapGL checks the
                // origin against the key's domain restriction.
                loadDataWithBaseURL(
                    "https://keel.uz/",
                    twoGisTrackHtml(key, first, home, css(accent)),
                    "text/html",
                    "utf-8",
                    null,
                )
            }
        },
        update = { web ->
            if (courier != first) {
                web.evaluateJavascript(
                    "window.moveCourier && moveCourier(${courier.lng}, ${courier.lat})",
                    null,
                )
            }
        },
    )
}
