import { useCallback, useEffect, useRef, useState } from "react";
import { AppState } from "react-native";
import * as Location from "expo-location";
import { activateKeepAwakeAsync, deactivateKeepAwake } from "expo-keep-awake";

import { api } from "@/lib/api";

import { startBackgroundUpdates, stopBackgroundUpdates } from "./background";

// Where the courier is, and who is told.
//
// ⚠️ **This is the reason the app exists as an app.** The PWA at `/kuryer` can
// only report a position while its tab is alive, and Android throttles a
// backgrounded tab until the reports stop — so a courier had to hold the phone
// awake with the browser open and hope. A native app gets a real position
// stream, keeps the screen awake through an API meant for it, and flushes the
// last fix when it goes to the background instead of being killed mid-send.
//
// ⚠️ **Two streams, and the second one is not a nicety.** The watcher below
// runs while the app is on screen; `background.ts` keeps reporting when it is
// not, through a foreground service the courier can see. They overlap on
// purpose — the buffer here is what survives a dead network, and the service is
// what survives a pocket — and the server keeps only the newest point either
// way, so a duplicate costs nothing.
//
// ⚠️ **Two consumers, one stream, and they want different things.** The
// restaurant wants the last known position (the panel draws it on a map); the
// button on the order card wants *this* position, now, to decide whether the
// courier is at the door. So the fix is held in state for the screen and
// buffered for the network — and neither waits for the other.

export type GeoState = "unknown" | "granted" | "denied";

export interface Fix {
  lat: number;
  lng: number;
  accuracy: number;
  /** Unix ms, from the device's own timestamp for the fix. */
  at: number;
}

/** How often the buffer is emptied towards the server.
 *
 *  ⚠️ **Not every fix, and not once a minute.** Every fix is a request per
 *  second on a moving bike — battery and mobile data the courier pays for. Once
 *  a minute is worse in the other direction: `arrivalBlocked` on the server
 *  calls a position older than ten minutes unusable, and a dispatcher watching
 *  a map wants a courier that moves rather than teleports. */
const FLUSH_MS = 15000;

/** Distance between reported points. Anything closer is a phone sitting still
 *  and reporting jitter, which costs data and moves the map pin around a
 *  stationary bike. */
const DISTANCE_M = 25;

/** ⚠️ The buffer is capped: a courier riding through an hour without signal
 *  would otherwise arrive with a thousand points, and the server keeps only the
 *  newest one anyway (see `CourierUpdateLocation`). The tail is what matters. */
const MAX_BUFFER = 50;

export interface Tracking {
  /** The last fix this phone took, or null before the first one. */
  fix: Fix | null;
  state: GeoState;
  /** Whether the background service is running — i.e. whether the shift
   *  survives the screen going off. ⚠️ Surfaced because the two cases behave
   *  completely differently and only one of them needs the app kept open. */
  background: boolean;
  /** True once positions are actually arriving, as opposed to permitted. */
  live: boolean;
  /** Unix ms of the last successful send, for the line under the switch. */
  lastSentAt: number | null;
  pending: number;
  /** Ask for permission and take one fix. ⚠️ Must be called from a press. */
  request: () => Promise<void>;
}

export function useTracking(onShift: boolean): Tracking {
  const [fix, setFix] = useState<Fix | null>(null);
  const [state, setState] = useState<GeoState>("unknown");
  const [live, setLive] = useState(false);
  const [background, setBackground] = useState(false);
  const [lastSentAt, setLastSentAt] = useState<number | null>(null);
  const [pending, setPending] = useState(0);

  const buffer = useRef<Fix[]>([]);
  const watcher = useRef<Location.LocationSubscription | null>(null);

  // What the phone already granted, asked once rather than assumed. A courier
  // who allowed it last week must not be shown a permission card today.
  useEffect(() => {
    void Location.getForegroundPermissionsAsync().then((p) =>
      setState(p.granted ? "granted" : p.canAskAgain ? "unknown" : "denied"),
    );
  }, []);

  const push = useCallback((f: Fix) => {
    setFix(f);
    buffer.current = [...buffer.current, f].slice(-MAX_BUFFER);
    setPending(buffer.current.length);
  }, []);

  const flush = useCallback(async () => {
    if (buffer.current.length === 0) return;
    // ⚠️ Taken before the await and cleared after it succeeds: a fix that
    // arrives mid-request must not be dropped with the batch it was not in.
    const batch = buffer.current;
    try {
      await api.courierSendLocation(batch);
      buffer.current = buffer.current.slice(batch.length);
      setPending(buffer.current.length);
      setLastSentAt(Date.now());
    } catch {
      // Offline, or the server is down. The buffer keeps its points and the
      // next tick tries again — this is a bike in a basement, not an error.
    }
  }, []);

  const request = useCallback(async () => {
    const p = await Location.requestForegroundPermissionsAsync();
    if (!p.granted) {
      setState(p.canAskAgain ? "unknown" : "denied");
      return;
    }
    setState("granted");
    try {
      // ⚠️ **Balanced, not the highest accuracy.** The arrival radius a
      // restaurant sets is 50–150 m and a phone's GPS is 10–30 outdoors, so
      // the extra seconds and battery the highest setting spends do not change
      // the answer — and they are spent by somebody standing at a gate.
      const pos = await Location.getCurrentPositionAsync({
        accuracy: Location.Accuracy.Balanced,
      });
      push(toFix(pos));
    } catch {
      // No fix yet. The watcher below keeps trying, and the card says so.
    }
  }, [push]);

  // The background service, for as long as the courier is on shift.
  //
  // ⚠️ **Started here rather than at the first fix**, so the "allow all the
  // time" dialog arrives while somebody is looking at the shift they just
  // opened — not twenty minutes later at a kerb, where it is dismissed.
  useEffect(() => {
    let cancelled = false;
    if (!onShift || state !== "granted") {
      void stopBackgroundUpdates();
      setBackground(false);
      return;
    }
    void startBackgroundUpdates()
      .then((ok) => {
        if (!cancelled) setBackground(ok);
      })
      .catch(() => {
        // Refused, or unavailable on this device. The foreground stream is
        // what the app had before this existed, and it still works.
        if (!cancelled) setBackground(false);
      });
    return () => {
      cancelled = true;
      void stopBackgroundUpdates();
      setBackground(false);
    };
  }, [onShift, state]);

  // The position stream, for as long as the courier is on shift.
  useEffect(() => {
    let cancelled = false;
    if (!onShift || state !== "granted") {
      setLive(false);
      return;
    }

    void (async () => {
      const sub = await Location.watchPositionAsync(
        {
          accuracy: Location.Accuracy.Balanced,
          distanceInterval: DISTANCE_M,
          timeInterval: 10000,
        },
        (pos) => {
          setLive(true);
          push(toFix(pos));
        },
      );
      if (cancelled) {
        sub.remove();
        return;
      }
      watcher.current = sub;
    })();

    return () => {
      cancelled = true;
      watcher.current?.remove();
      watcher.current = null;
      setLive(false);
    };
  }, [onShift, state, push]);

  // Sending, on a timer and on the one event that is not a timer.
  useEffect(() => {
    if (!onShift) return;
    const timer = setInterval(() => void flush(), FLUSH_MS);
    // ⚠️ **Flushed on the way to the background, not after coming back.** The
    // system can kill a backgrounded app without warning, and everything still
    // in the buffer goes with it — including the fix that would have opened the
    // "delivered" button when the courier reopened the app at the door.
    const sub = AppState.addEventListener("change", (s) => {
      if (s !== "active") void flush();
    });
    return () => {
      clearInterval(timer);
      sub.remove();
      void flush();
    };
  }, [onShift, flush]);

  // ⚠️ **The screen is held awake only when the background service is not.**
  // With the service running the phone can sleep and keep reporting, and
  // burning a courier's battery to hold a screen on for nothing is how an app
  // gets uninstalled. Without it — permission refused, or a device that will
  // not run it — the screen is the only thing keeping the stream alive.
  useEffect(() => {
    if (!onShift || background) return;
    void activateKeepAwakeAsync("courier-shift").catch(() => {
      // Some devices refuse it. Tracking still works while the screen is on,
      // which is what the hint under the switch already promises.
    });
    return () => {
      try {
        deactivateKeepAwake("courier-shift");
      } catch {
        // Never held it; nothing to release.
      }
    };
  }, [onShift, background]);

  return { fix, state, live, background, lastSentAt, pending, request };
}

function toFix(pos: Location.LocationObject): Fix {
  return {
    lat: pos.coords.latitude,
    lng: pos.coords.longitude,
    accuracy: pos.coords.accuracy ?? 0,
    at: pos.timestamp || Date.now(),
  };
}

/** Metres between two points, on the sphere.
 *
 *  ⚠️ **The same formula the server uses** (`haversineKm` in
 *  `internal/handlers`), because the button and the rule have to agree: a
 *  courier shown an open button and then refused by the server is a courier who
 *  believes the app is broken, in front of a customer. */
export function metersBetween(
  a: { lat: number; lng: number },
  b: { lat: number; lng: number },
): number {
  const R = 6371000;
  const dLat = ((b.lat - a.lat) * Math.PI) / 180;
  const dLng = ((b.lng - a.lng) * Math.PI) / 180;
  const s =
    Math.sin(dLat / 2) ** 2 +
    Math.cos((a.lat * Math.PI) / 180) *
      Math.cos((b.lat * Math.PI) / 180) *
      Math.sin(dLng / 2) ** 2;
  return R * 2 * Math.atan2(Math.sqrt(s), Math.sqrt(1 - s));
}
