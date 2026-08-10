// Which map this restaurant draws with, and the key for it.
//
// Both come from the restaurant profile the browser already fetches, never from
// the build: one build serves every tenant on the platform, so anything baked in
// at build time is the same wrong value for all of them.
//
// ⚠️ The key is **public and cannot be otherwise**. Every map SDK is a browser
// library — the key is handed to it inside the page and is visible in DevTools
// wherever we keep it. What protects it is the domain restriction set in the
// provider's own console: a copied key does not work on anybody else's site.
// This is the deliberate opposite of the payment credentials, which are kept out
// of this response precisely because it goes to every visitor.

import { useEffect, useState } from "react";

import { API_URL } from "@/lib/api";

export const MAP_PROVIDERS = ["2gis", "yandex", "google"] as const;
export type MapProviderId = (typeof MAP_PROVIDERS)[number];

/** ⚠️ Empty means 2GIS. Every install that predates the setting is on 2GIS, and
 *  reading the zero value as anything else would blank their map on deploy. */
export function normalizeProvider(value: string | undefined): MapProviderId {
  return (MAP_PROVIDERS as readonly string[]).includes(value ?? "")
    ? (value as MapProviderId)
    : "2gis";
}

export interface MapConfig {
  provider: MapProviderId;
  key: string;
}

const FALLBACK_KEY = process.env.NEXT_PUBLIC_MAP_API_KEY ?? "";

// One request per page load, shared by every map on it: the profile is small,
// already cached by the browser, and four map components asking separately would
// be four identical requests.
let pending: Promise<MapConfig> | null = null;

export function loadMapConfig(): Promise<MapConfig> {
  if (typeof window === "undefined") {
    return Promise.resolve({ provider: "2gis", key: FALLBACK_KEY });
  }
  if (!pending) {
    pending = fetch(`${API_URL}/restaurant`)
      .then((r) => (r.ok ? r.json() : null))
      // ⚠️ The profile arrives **wrapped**: `{ restaurant, brand, branch,
      // isOpenNow }`. Read from the top level, the key was always `undefined`
      // and every map silently fell back to the platform key — which is empty
      // on Keel, so a restaurant could type a perfectly good key, save it, and
      // still have no map, with nothing anywhere saying why.
      .then((d) => {
        const rest = d?.restaurant ?? {};
        const provider = normalizeProvider(rest.mapProvider);
        // A key per provider, so switching back and forth never hands one
        // provider the other's key — which fails as a blank map and a console
        // error nobody in a kitchen reads.
        const key =
          provider === "yandex"
            ? (rest.mapYandexKey as string) || ""
            : provider === "google"
              ? (rest.mapGoogleKey as string) || ""
              : (rest.mapApiKey as string) || FALLBACK_KEY;
        return { provider, key };
      })
      // A profile that will not load is somebody else's error to report; the map
      // falls back to the platform key rather than disappearing.
      .catch(() => ({ provider: "2gis" as MapProviderId, key: FALLBACK_KEY }));
  }
  return pending;
}

/** Re-reads the profile next time. Called after the key is edited, so the
 *  admin's own zone map picks up the new one without a page reload. */
export function forgetMapConfig() {
  pending = null;
}

/** This restaurant's map, or `null` while it is still being fetched.
 *
 *  `null` and "no key" are deliberately different: "still loading" must not
 *  render the "no key configured" message, which would flash on every page with
 *  a map and send owners looking for a setting that is already correct. */
export function useMapConfig(): MapConfig | null {
  const [config, setConfig] = useState<MapConfig | null>(null);
  useEffect(() => {
    let alive = true;
    loadMapConfig().then((c) => {
      if (alive) setConfig(c);
    });
    return () => {
      alive = false;
    };
  }, []);
  return config;
}
