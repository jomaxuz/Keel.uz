"use client";

// Asking for location, and explaining what to do when the answer is "no".
//
// The hard part is not the API call. It is that a browser asks **once**: after
// the user has denied it, `getCurrentPosition` fails instantly with no prompt,
// and nothing the page does can bring the dialog back. Only the person can, in
// browser settings — which is exactly what nobody knows how to find.
//
// So this module does three things a bare `getCurrentPosition` does not:
//   1. reads the permission state up front, so the UI can show a real button
//      ("Allow location") instead of an error nobody asked for;
//   2. asks from a user gesture, which iOS Safari requires and which makes the
//      prompt appear when the user is expecting it;
//   3. when the answer is already "denied", shows the steps for the device in
//      the user's hand, because that is the only way back.

import { useCallback, useEffect, useState } from "react";

export type GeoState =
  /** Not determined yet (or the browser hides it — Safari has no Permissions
   *  API for geolocation). Asking is still worth a try. */
  | "unknown"
  /** The browser will show the prompt when we ask. */
  | "prompt"
  | "granted"
  /** Denied. The page can no longer prompt; only device settings can undo it. */
  | "denied"
  /** Page is not on HTTPS (or localhost). Browsers refuse location outright,
   *  whatever the permission says — a plain http:// LAN address is the usual
   *  reason this happens during testing. */
  | "insecure"
  | "unsupported";

export interface GeoFix {
  lat: number;
  lng: number;
  accuracy: number;
  at: number;
}

/** Which set of instructions to show. Detected from the user agent, which is
 *  imprecise by nature — so the UI always offers the generic steps too. */
export type GeoPlatform = "ios" | "android" | "desktop";

export function detectPlatform(): GeoPlatform {
  if (typeof navigator === "undefined") return "desktop";
  const ua = navigator.userAgent;
  // iPadOS 13+ reports itself as a Mac; the touch points give it away.
  const iPadOS = /Macintosh/.test(ua) && navigator.maxTouchPoints > 1;
  if (/iPhone|iPad|iPod/.test(ua) || iPadOS) return "ios";
  if (/Android/.test(ua)) return "android";
  return "desktop";
}

function readInitialState(): GeoState {
  if (typeof window === "undefined") return "unknown";
  if (!("geolocation" in navigator)) return "unsupported";
  // Secure context is a hard gate: no permission state can override it.
  if (!window.isSecureContext) return "insecure";
  return "unknown";
}

/**
 * Permission state plus a way to ask for it.
 *
 * `request()` must be called from a click. It resolves to a fix or null, and
 * updates `state` either way, so the UI can move from a button to instructions
 * without a second round trip.
 */
export function useGeoPermission() {
  const [state, setState] = useState<GeoState>("unknown");
  const [checking, setChecking] = useState(false);

  // Read the stored permission, and keep watching it: a user who grants access
  // in browser settings should not have to reload the page to continue.
  useEffect(() => {
    const initial = readInitialState();
    setState(initial);
    if (initial !== "unknown") return;

    let cancelled = false;
    let status: PermissionStatus | null = null;
    const onChange = () => {
      if (!cancelled && status) setState(status.state as GeoState);
    };

    // Safari has no geolocation entry in the Permissions API — the query
    // rejects, and "unknown" is the honest answer there.
    navigator.permissions
      ?.query({ name: "geolocation" as PermissionName })
      .then((s) => {
        if (cancelled) return;
        status = s;
        setState(s.state as GeoState);
        s.addEventListener("change", onChange);
      })
      .catch(() => {
        /* keep "unknown" */
      });

    return () => {
      cancelled = true;
      status?.removeEventListener("change", onChange);
    };
  }, []);

  const request = useCallback((): Promise<GeoFix | null> => {
    if (typeof window === "undefined") return Promise.resolve(null);
    if (!("geolocation" in navigator)) {
      setState("unsupported");
      return Promise.resolve(null);
    }
    if (!window.isSecureContext) {
      setState("insecure");
      return Promise.resolve(null);
    }
    setChecking(true);
    return new Promise((resolve) => {
      navigator.geolocation.getCurrentPosition(
        (pos) => {
          setChecking(false);
          setState("granted");
          resolve({
            lat: pos.coords.latitude,
            lng: pos.coords.longitude,
            accuracy: pos.coords.accuracy ?? 0,
            at: pos.timestamp || Date.now(),
          });
        },
        (err) => {
          setChecking(false);
          // Only PERMISSION_DENIED is a permission problem. A timeout or a
          // failed fix means the answer was yes but the device could not
          // deliver — telling that user to open browser settings is useless.
          setState(err.code === err.PERMISSION_DENIED ? "denied" : "granted");
          resolve(null);
        },
        { enableHighAccuracy: true, maximumAge: 0, timeout: 20000 },
      );
    });
  }, []);

  return { state, checking, request, setState };
}
