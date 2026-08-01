// Address search + reverse geocoding via OpenStreetMap Nominatim (free, no key).
//
// 2GIS geocoding is billed separately and the MapGL key does not cover it, so we
// use Nominatim for suggestions. Results are limited to Uzbekistan. For high
// production volume, self-host Nominatim or switch to a paid provider — the
// call sites only depend on the Place shape below.

export interface Place {
  text: string;
  lat: number;
  lng: number;
}

const BASE = "https://nominatim.openstreetmap.org";

interface NominatimResult {
  display_name: string;
  lat: string;
  lon: string;
}

// Search places by free-text query (debounce at the call site).
export async function searchPlaces(
  query: string,
  signal?: AbortSignal,
): Promise<Place[]> {
  const q = query.trim();
  if (q.length < 3) return [];
  const url =
    `${BASE}/search?format=jsonv2&addressdetails=0&limit=6` +
    `&countrycodes=uz&accept-language=uz&q=${encodeURIComponent(q)}`;
  const res = await fetch(url, { signal });
  if (!res.ok) return [];
  const data = (await res.json()) as NominatimResult[];
  return data.map((r) => ({
    text: r.display_name,
    lat: parseFloat(r.lat),
    lng: parseFloat(r.lon),
  }));
}

// Reverse geocode a point to a human-readable address.
export async function reverseGeocode(
  lat: number,
  lng: number,
  signal?: AbortSignal,
): Promise<string | null> {
  const url =
    `${BASE}/reverse?format=jsonv2&accept-language=uz` +
    `&lat=${lat}&lon=${lng}`;
  const res = await fetch(url, { signal });
  if (!res.ok) return null;
  const data = (await res.json()) as { display_name?: string };
  return data.display_name ?? null;
}
