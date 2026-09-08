package uz.keel.guest.data

import io.ktor.client.HttpClient
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.statement.bodyAsText
import io.ktor.http.encodeURLParameter
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

// Address search and reverse geocoding, via OpenStreetMap Nominatim.
//
// ⚠️ **Deliberately not Google's**, even though the map on the same screen is.
// The Maps SDK for Android draws for free; the Geocoding and Places APIs are
// billed per call, and a search box fires one per keystroke. The site made
// exactly this split for exactly this reason (`lib/geocode.ts`) — and it means
// the search keeps working for a restaurant that has no maps key at all.
//
// ⚠️ **Nominatim asks for a real User-Agent and rate-limits to one call a
// second.** An app that ignored either gets its whole install base blocked at
// once, which arrives as "search does not work" from every guest simultaneously
// and looks nothing like a policy problem. Hence the debounce at the call site
// and the identifier here.

@Serializable
private data class NominatimResult(
    val display_name: String = "",
    val lat: String = "",
    val lon: String = "",
)

/** One place somebody can be delivered to. */
data class Place(val text: String, val lat: Double, val lng: Double)

class Geocoder {

    private val json = Json { ignoreUnknownKeys = true }
    private val client = HttpClient(OkHttp)

    /** Free-text search, limited to Uzbekistan the way the site limits it.
     *
     *  ⚠️ **Failure is an empty list, never an exception.** This runs while
     *  somebody types; a thrown error would put a red line under a search box
     *  for a service that is simply busy, and the map underneath still works. */
    suspend fun search(query: String): List<Place> {
        val q = query.trim()
        if (q.length < 3) return emptyList()
        val url = "$BASE/search?format=jsonv2&addressdetails=0&limit=6" +
            "&countrycodes=uz&accept-language=uz&q=${q.encodeURLParameter()}"
        return runCatching {
            json.decodeFromString<List<NominatimResult>>(get(url)).mapNotNull { it.place() }
        }.getOrDefault(emptyList())
    }

    /** A point to something a courier can read.
     *
     *  ⚠️ **Null rather than a fabricated string.** The guest types the flat
     *  number and the landmark themselves; an address invented from coordinates
     *  ("Unnamed Road") reads as a real answer and stops them writing the real
     *  one. */
    suspend fun reverse(lat: Double, lng: Double): String? = runCatching {
        val url = "$BASE/reverse?format=jsonv2&accept-language=uz&lat=$lat&lon=$lng"
        json.decodeFromString<NominatimResult>(get(url)).display_name.ifBlank { null }
    }.getOrNull()

    private suspend fun get(url: String): String =
        client.get(url) {
            // ⚠️ Required by Nominatim's policy, and the address is ours rather
            // than a restaurant's: they block the identifier, and one
            // restaurant's traffic must not take down every other app we build.
            header("User-Agent", "KeelGuest/1.0 (+https://keel.uz)")
        }.bodyAsText()

    private fun NominatimResult.place(): Place? {
        val la = lat.toDoubleOrNull() ?: return null
        val ln = lon.toDoubleOrNull() ?: return null
        return Place(display_name, la, ln)
    }

    private companion object {
        const val BASE = "https://nominatim.openstreetmap.org"
    }
}
