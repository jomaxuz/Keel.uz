// Which 2GIS key the map should load with.
//
// Each restaurant brings its own, entered in the admin panel, so the platform
// does not carry everybody's quota on one account. The key therefore cannot be
// baked into the build — one build serves every tenant — and has to be read at
// run time from the restaurant profile the browser already fetches.
//
// ⚠️ The key is **public and cannot be otherwise**. MapGL is a browser library:
// it is handed to `load({ key })` inside the page, so it is visible in DevTools
// wherever we keep it. Every map SDK works this way. What protects it is the
// domain restriction set in the 2GIS account — a copied key does not work on
// anybody else's site. This is the exact opposite of the payment credentials,
// which are kept out of this response on purpose.

import { useEffect, useState } from "react";

import { API_URL } from "@/lib/api";

const FALLBACK = process.env.NEXT_PUBLIC_MAP_API_KEY ?? "";

// One request per page load, shared by every map on it: the profile is small,
// already cached by the browser, and four map components asking separately
// would be four identical requests.
let pending: Promise<string> | null = null;

export function loadMapKey(): Promise<string> {
  if (typeof window === "undefined") return Promise.resolve(FALLBACK);
  if (!pending) {
    pending = fetch(`${API_URL}/restaurant`)
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => (d?.mapApiKey as string) || FALLBACK)
      // A profile that will not load is somebody else's error to report; the
      // map falls back to the platform key rather than disappearing.
      .catch(() => FALLBACK);
  }
  return pending;
}

/** Re-reads the profile next time. Called after the key is edited, so the
 *  admin's own zone map picks up the new one without a page reload. */
export function forgetMapKey() {
  pending = null;
}

/** The key for this restaurant, or `null` while it is still being fetched.
 *
 *  `null` and `""` are deliberately different: "still loading" must not render
 *  the "no key configured" message, which would flash on every page with a map
 *  and send owners looking for a setting that is already correct. */
export function useMapKey(): string | null {
  const [key, setKey] = useState<string | null>(null);
  useEffect(() => {
    let alive = true;
    loadMapKey().then((k) => {
      if (alive) setKey(k);
    });
    return () => {
      alive = false;
    };
  }, []);
  return key;
}
