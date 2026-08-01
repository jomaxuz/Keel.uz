"use client";

// Courier session + location reporting for the PWA at /kuryer.
//
// IMPORTANT — what a web app can and cannot do:
// Browsers do not give geolocation to service workers, and a PWA that has been
// closed (swiped away) gets no CPU at all. So positions are reported while the
// app is *running*: in the foreground, and — on Android — for as long as the
// system keeps the backgrounded tab alive, which it throttles heavily.
// To make a shift usable we therefore:
//   • hold a Wake Lock so the screen does not sleep while on shift,
//   • flush the last position immediately when the app is hidden,
//   • buffer positions offline and send them when the network returns.
// Continuous tracking with the app fully closed needs a native wrapper
// (Capacitor / TWA with a foreground service) — see PROGRESS.md.

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { api, clearCourierToken, getCourierToken, setCourierToken } from "./api";
import type { Courier, CourierStatus } from "./types";

export type GeoErrorCode = "denied" | "failed" | "unsupported";

interface BufferedPoint {
  lat: number;
  lng: number;
  accuracy: number;
  at: number;
}

const BUFFER_KEY = "courier_pending_points";
const FLUSH_MS = 15000;

interface CourierContextValue {
  courier: Courier | null;
  loading: boolean;
  login: (token: string, courier: Courier) => void;
  logout: () => void;
  setStatus: (status: CourierStatus) => Promise<void>;
  // Live tracking state, surfaced in the UI.
  tracking: boolean;
  // Last fix from the device, used to gate the "delivered" button.
  position: BufferedPoint | null;
  // Error *code*, translated by the UI (the provider has no dictionary).
  geoError: GeoErrorCode | null;
  lastSentAt: number | null;
  pendingCount: number;
}

const CourierContext = createContext<CourierContextValue | null>(null);

function readBuffer(): BufferedPoint[] {
  try {
    const raw = window.localStorage.getItem(BUFFER_KEY);
    return raw ? (JSON.parse(raw) as BufferedPoint[]) : [];
  } catch {
    return [];
  }
}

function writeBuffer(points: BufferedPoint[]) {
  try {
    // Keep the tail only — we store the last known position, not a track.
    window.localStorage.setItem(BUFFER_KEY, JSON.stringify(points.slice(-50)));
  } catch {
    /* storage unavailable */
  }
}

export function CourierProvider({ children }: { children: ReactNode }) {
  const [courier, setCourier] = useState<Courier | null>(null);
  const [loading, setLoading] = useState(true);
  const [tracking, setTracking] = useState(false);
  const [position, setPosition] = useState<BufferedPoint | null>(null);
  const [geoError, setGeoError] = useState<GeoErrorCode | null>(null);
  const [lastSentAt, setLastSentAt] = useState<number | null>(null);
  const [pendingCount, setPendingCount] = useState(0);

  const bufferRef = useRef<BufferedPoint[]>([]);
  const watchRef = useRef<number | null>(null);
  const wakeLockRef = useRef<{ release: () => Promise<void> } | null>(null);

  // Restore the session on mount.
  useEffect(() => {
    if (!getCourierToken()) {
      setLoading(false);
      return;
    }
    api
      .courierMe()
      .then(setCourier)
      .catch(() => {
        clearCourierToken();
        setCourier(null);
      })
      .finally(() => setLoading(false));
    bufferRef.current = readBuffer();
    setPendingCount(bufferRef.current.length);
  }, []);

  const flush = useCallback(async () => {
    if (!bufferRef.current.length || !getCourierToken()) return;
    const batch = bufferRef.current;
    try {
      await api.courierSendLocation(batch);
      bufferRef.current = [];
      writeBuffer([]);
      setPendingCount(0);
      setLastSentAt(Date.now());
    } catch {
      // Offline or server down — keep the buffer for the next attempt.
    }
  }, []);

  const onShift = !!courier && courier.status !== "off";

  // Watch the device position while the courier is on shift.
  useEffect(() => {
    if (!onShift) {
      if (watchRef.current !== null) {
        navigator.geolocation.clearWatch(watchRef.current);
        watchRef.current = null;
      }
      setTracking(false);
      return;
    }
    if (!("geolocation" in navigator)) {
      setGeoError("unsupported");
      return;
    }

    setGeoError(null);
    watchRef.current = navigator.geolocation.watchPosition(
      (pos) => {
        setTracking(true);
        setGeoError(null);
        const point: BufferedPoint = {
          lat: pos.coords.latitude,
          lng: pos.coords.longitude,
          accuracy: pos.coords.accuracy ?? 0,
          at: pos.timestamp || Date.now(),
        };
        setPosition(point);
        bufferRef.current = [...bufferRef.current, point];
        writeBuffer(bufferRef.current);
        setPendingCount(bufferRef.current.length);
      },
      (err) => {
        setTracking(false);
        setGeoError(err.code === err.PERMISSION_DENIED ? "denied" : "failed");
      },
      { enableHighAccuracy: true, maximumAge: 10000, timeout: 20000 },
    );

    return () => {
      if (watchRef.current !== null) {
        navigator.geolocation.clearWatch(watchRef.current);
        watchRef.current = null;
      }
      setTracking(false);
    };
  }, [onShift]);

  // Send whatever has been collected, on a timer and on the events that matter.
  useEffect(() => {
    if (!onShift) return;
    const timer = setInterval(flush, FLUSH_MS);
    const onHide = () => {
      if (document.visibilityState === "hidden") void flush();
    };
    window.addEventListener("visibilitychange", onHide);
    window.addEventListener("pagehide", onHide);
    window.addEventListener("online", flush);
    return () => {
      clearInterval(timer);
      window.removeEventListener("visibilitychange", onHide);
      window.removeEventListener("pagehide", onHide);
      window.removeEventListener("online", flush);
    };
  }, [onShift, flush]);

  // Keep the screen awake on shift — the only reliable way to keep a web app
  // reporting while the courier rides.
  useEffect(() => {
    let cancelled = false;
    async function acquire() {
      const nav = navigator as Navigator & {
        wakeLock?: { request: (t: "screen") => Promise<{ release: () => Promise<void> }> };
      };
      if (!onShift || !nav.wakeLock) return;
      try {
        const lock = await nav.wakeLock.request("screen");
        if (cancelled) {
          void lock.release();
          return;
        }
        wakeLockRef.current = lock;
      } catch {
        /* denied or unsupported — tracking still works while visible */
      }
    }
    void acquire();
    const reacquire = () => {
      if (document.visibilityState === "visible") void acquire();
    };
    document.addEventListener("visibilitychange", reacquire);
    return () => {
      cancelled = true;
      document.removeEventListener("visibilitychange", reacquire);
      void wakeLockRef.current?.release().catch(() => {});
      wakeLockRef.current = null;
    };
  }, [onShift]);

  const login = useCallback((token: string, c: Courier) => {
    setCourierToken(token);
    setCourier(c);
  }, []);

  const logout = useCallback(() => {
    clearCourierToken();
    setCourier(null);
    bufferRef.current = [];
    writeBuffer([]);
    setPendingCount(0);
  }, []);

  const setStatus = useCallback(async (status: CourierStatus) => {
    await api.courierSetStatus(status);
    setCourier((c) => (c ? { ...c, status } : c));
  }, []);

  return (
    <CourierContext.Provider
      value={{
        courier,
        loading,
        login,
        logout,
        setStatus,
        tracking,
        position,
        geoError,
        lastSentAt,
        pendingCount,
      }}
    >
      {children}
    </CourierContext.Provider>
  );
}

export function useCourier(): CourierContextValue {
  const ctx = useContext(CourierContext);
  if (!ctx) throw new Error("useCourier must be used within a CourierProvider");
  return ctx;
}
