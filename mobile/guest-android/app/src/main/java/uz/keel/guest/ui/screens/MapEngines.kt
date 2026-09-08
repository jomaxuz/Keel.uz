package uz.keel.guest.ui.screens

import android.annotation.SuppressLint
import android.webkit.JavascriptInterface
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.viewinterop.AndroidView
import com.yandex.mapkit.MapKitFactory
import com.yandex.mapkit.geometry.Point
import com.yandex.mapkit.map.CameraListener
import com.yandex.mapkit.map.CameraPosition
import com.yandex.mapkit.mapview.MapView
import uz.keel.guest.data.GeoPoint

// ---- Three maps, one screen ----
//
// ⚠️ **The restaurant chooses, and the app agrees with its own site.** The
// panel's `mapProvider` decides which engine draws — 2GIS, Yandex or Google —
// exactly as it decides on the web. An app that always drew Google would show a
// guest a different map from the one on the restaurant's site, with a different
// idea of where the streets are.
//
// ⚠️ **The coordinate order lives here and nowhere else.** 2GIS wants
// `[lng, lat]`, Yandex `Point(lat, lng)` and Google `LatLng(lat, lng)`. CLAUDE.md
// records what happens when that conversion is written in four places and
// reversed in one of them: the restaurant ends up in the Aral Sea and it reads
// as a data error. Each engine converts once, on its way in and on its way out.
//
// ⚠️ **2GIS is a WebView and the other two are native, and that is not a
// preference.** 2GIS publishes no Android SDK we can obtain — their Maven paths
// answer 404 — so the only way to honour a 2GIS restaurant's choice is their web
// engine (MapGL), which is the same one the site draws with. It is the least bad
// of three options: the others were "give 2GIS restaurants a Google map" and
// "tell them to pick another provider".

/** What every engine has to be able to do. */
interface MapEngine {
    /** Draw the map, reporting the centre whenever it settles. */
    @Composable
    fun Draw(start: GeoPoint, onCentre: (Double, Double) -> Unit, modifier: Modifier)
}

// ---- Yandex ----

/** ⚠️ **The key is set before `initialize`, once per process.** MapKit throws if
 *  a map is created without one, and it refuses to be re-keyed afterwards — so
 *  a restaurant that changes its key needs the app restarted, which is the
 *  behaviour the SDK gives us rather than one we chose. */
object YandexMap {
    @Volatile
    private var keyed = false

    fun prepare(context: android.content.Context, key: String): Boolean {
        if (key.isBlank()) return false
        return runCatching {
            if (!keyed) {
                MapKitFactory.setApiKey(key)
                keyed = true
            }
            MapKitFactory.initialize(context)
            true
        }.getOrDefault(false)
    }
}

@Composable
fun YandexPicker(
    start: GeoPoint,
    onCentre: (Double, Double) -> Unit,
    modifier: Modifier = Modifier,
) {
    val listener = remember {
        CameraListener { _, position, _, finished ->
            // ⚠️ Only when the gesture has settled. Every intermediate frame
            // would be a reverse-geocode request, and the service this app uses
            // rate-limits to one a second — see data/Geocode.kt.
            if (finished) onCentre(position.target.latitude, position.target.longitude)
        }
    }
    AndroidView(
        modifier = modifier.fillMaxSize(),
        factory = { ctx ->
            MapView(ctx).also { view ->
                view.mapWindow.map.move(
                    CameraPosition(Point(start.lat, start.lng), 16f, 0f, 0f),
                )
                // ⚠️ **A weak reference, because MapKit asks for one.** The SDK
                // holds listeners weakly on purpose — a map outliving a screen
                // must not keep it alive — which means the strong reference has
                // to be ours: `listener` is remembered by the composition, and
                // without that the callback stops firing after the first GC and
                // the address silently stops updating.
                view.mapWindow.map.addCameraListener(java.lang.ref.WeakReference(listener))
                MapKitFactory.getInstance().onStart()
                view.onStart()
            }
        },
    )
    DisposableEffect(Unit) {
        onDispose {
            runCatching { MapKitFactory.getInstance().onStop() }
        }
    }
}

// ---- 2GIS, through its own web engine ----

/** The page the WebView loads.
 *
 *  ⚠️ **Built here rather than fetched from the restaurant's site.** A page
 *  served by the tenant would carry its cookies, its session and whatever the
 *  site happens to render that week; this is forty lines that draw a map and
 *  report a centre, and nothing else.
 *
 *  ⚠️ **The pin is drawn by the page, not by the map.** MapGL markers move with
 *  the map; a fixed pin over the centre is the pattern every delivery app here
 *  uses, and it is what the native engines do too. */
private fun twoGisHtml(key: String, lat: Double, lng: Double): String = """
<!doctype html><html><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1,user-scalable=no">
<style>html,body,#map{height:100%;margin:0;padding:0}</style>
<script src="https://mapgl.2gis.com/api/js/v1"></script></head>
<body><div id="map"></div><script>
  var map = new mapgl.Map('map', {
    // ⚠️ 2GIS takes [lng, lat]. Yandex and Google take them the other way round,
    // and each engine converts once — see MapEngines.kt.
    center: [$lng, $lat], zoom: 16, key: '$key'
  });
  var t = null;
  map.on('moveend', function () {
    var c = map.getCenter();
    clearTimeout(t);
    t = setTimeout(function () { Bridge.centre(c[1], c[0]); }, 250);
  });
</script></body></html>
"""

@SuppressLint("SetJavaScriptEnabled")
@Composable
fun TwoGisPicker(
    key: String,
    start: GeoPoint,
    onCentre: (Double, Double) -> Unit,
    modifier: Modifier = Modifier,
) {
    AndroidView(
        modifier = modifier.fillMaxSize(),
        factory = { ctx ->
            WebView(ctx).apply {
                settings.javaScriptEnabled = true
                settings.domStorageEnabled = true
                webViewClient = WebViewClient()
                addJavascriptInterface(
                    object {
                        @JavascriptInterface
                        fun centre(lat: Double, lng: Double) {
                            // ⚠️ Posted back to the main thread: the bridge runs
                            // on a WebView worker, and Compose state written from
                            // it throws.
                            post { onCentre(lat, lng) }
                        }
                    },
                    "Bridge",
                )
                // ⚠️ A real base URL, not `null`: MapGL checks the origin
                // against the key's domain restriction, and a page loaded from
                // nowhere is refused with a blank map and a console message
                // nobody in a restaurant is going to read.
                loadDataWithBaseURL(
                    "https://keel.uz/",
                    twoGisHtml(key, start.lat, start.lng),
                    "text/html",
                    "utf-8",
                    null,
                )
            }
        },
    )
}
