"use client";

// Staff session for the PWA at /staff.
//
// Unlike the courier app this does not report a track: the restaurant does not
// need to know where a cook is all day. It needs one thing — that the person
// pressing "kirdim" is standing at the branch — so a position is watched only
// while the app is open, and it is sent only with a punch.
//
// The watch runs regardless of shift state on purpose: the button has to be
// able to say "you are 40 m away" *before* it is pressed, otherwise the first
// tap is the first anyone hears about the problem.

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import {
  api,
  clearStaffToken,
  getStaffToken,
  setStaffToken,
} from "./api";
import type { Shift, Staff, StaffWorkplace } from "./types";

export type GeoErrorCode = "denied" | "failed" | "unsupported";

export interface StaffPosition {
  lat: number;
  lng: number;
  accuracy: number;
  at: number;
}

interface StaffContextValue {
  staff: Staff | null;
  workplace: StaffWorkplace | null;
  /** The shift currently clocked in, or null when off. */
  openShift: Shift | null;
  loading: boolean;
  login: (token: string, staff: Staff) => void;
  logout: () => void;
  reload: () => Promise<void>;

  position: StaffPosition | null;
  geoError: GeoErrorCode | null;
  /** Metres from the workplace, or null when either side is unknown. */
  distance: number | null;
  /** Why the clock buttons are disabled, or null when they work. */
  blocked: "no-workplace" | "no-fix" | "too-far" | null;
  /** Ask the browser for a fresh fix — what the "try again" button does. */
  refresh: () => void;

  clock: (action: "in" | "out") => Promise<void>;
}

const StaffContext = createContext<StaffContextValue | null>(null);

/** Straight-line metres between two coordinates. Mirrors the server's check —
 *  the server is still the authority, this only drives the button. */
function metersBetween(
  a: { lat: number; lng: number },
  b: { lat: number; lng: number },
): number {
  const R = 6371000;
  const dLat = ((b.lat - a.lat) * Math.PI) / 180;
  const dLng = ((b.lng - a.lng) * Math.PI) / 180;
  const h =
    Math.sin(dLat / 2) ** 2 +
    Math.cos((a.lat * Math.PI) / 180) *
      Math.cos((b.lat * Math.PI) / 180) *
      Math.sin(dLng / 2) ** 2;
  return R * 2 * Math.atan2(Math.sqrt(h), Math.sqrt(1 - h));
}

export function StaffProvider({ children }: { children: ReactNode }) {
  const [staff, setStaff] = useState<Staff | null>(null);
  const [workplace, setWorkplace] = useState<StaffWorkplace | null>(null);
  const [openShift, setOpenShift] = useState<Shift | null>(null);
  const [loading, setLoading] = useState(true);
  const [position, setPosition] = useState<StaffPosition | null>(null);
  const [geoError, setGeoError] = useState<GeoErrorCode | null>(null);
  const watchRef = useRef<number | null>(null);

  const reload = useCallback(async () => {
    if (!getStaffToken()) {
      setStaff(null);
      setLoading(false);
      return;
    }
    try {
      const me = await api.staffMe();
      setStaff(me.staff);
      setWorkplace(me.workplace ?? null);
      setOpenShift(me.openShift ?? null);
    } catch {
      clearStaffToken();
      setStaff(null);
      setWorkplace(null);
      setOpenShift(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  // Watch the device position while the app is open and someone is signed in.
  useEffect(() => {
    if (!staff) return;
    if (!("geolocation" in navigator)) {
      setGeoError("unsupported");
      return;
    }
    setGeoError(null);
    watchRef.current = navigator.geolocation.watchPosition(
      (pos) => {
        setGeoError(null);
        setPosition({
          lat: pos.coords.latitude,
          lng: pos.coords.longitude,
          accuracy: pos.coords.accuracy ?? 0,
          at: pos.timestamp || Date.now(),
        });
      },
      (err) => {
        setGeoError(err.code === err.PERMISSION_DENIED ? "denied" : "failed");
      },
      { enableHighAccuracy: true, maximumAge: 5000, timeout: 20000 },
    );
    return () => {
      if (watchRef.current !== null) {
        navigator.geolocation.clearWatch(watchRef.current);
        watchRef.current = null;
      }
    };
  }, [staff]);

  const refresh = useCallback(() => {
    if (!("geolocation" in navigator)) {
      setGeoError("unsupported");
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        setGeoError(null);
        setPosition({
          lat: pos.coords.latitude,
          lng: pos.coords.longitude,
          accuracy: pos.coords.accuracy ?? 0,
          at: pos.timestamp || Date.now(),
        });
      },
      (err) => {
        setGeoError(err.code === err.PERMISSION_DENIED ? "denied" : "failed");
      },
      { enableHighAccuracy: true, maximumAge: 0, timeout: 20000 },
    );
  }, []);

  const hasPin = !!workplace && (workplace.address.lat !== 0 || workplace.address.lng !== 0);
  const distance =
    position && hasPin ? metersBetween(position, workplace!.address) : null;

  // Why the buttons are off. A branch with the check switched off (radius 0) or
  // with no pin on the map is never blocked — refusing there would lock a whole
  // kitchen out of its own app over a setting nobody filled in.
  let blocked: StaffContextValue["blocked"] = null;
  if (!workplace) {
    blocked = "no-workplace";
  } else if (workplace.radiusM > 0 && hasPin) {
    if (!position) blocked = "no-fix";
    else if (distance !== null && distance > workplace.radiusM) blocked = "too-far";
  }

  const login = useCallback((token: string, next: Staff) => {
    setStaffToken(token);
    setStaff(next);
    void reload();
  }, [reload]);

  const logout = useCallback(() => {
    clearStaffToken();
    setStaff(null);
    setWorkplace(null);
    setOpenShift(null);
  }, []);

  const clock = useCallback(
    async (action: "in" | "out") => {
      await api.staffClock(action, {
        lat: position?.lat ?? 0,
        lng: position?.lng ?? 0,
        accuracy: position?.accuracy ?? 0,
      });
      await reload();
    },
    [position, reload],
  );

  return (
    <StaffContext.Provider
      value={{
        staff,
        workplace,
        openShift,
        loading,
        login,
        logout,
        reload,
        position,
        geoError,
        distance,
        blocked,
        refresh,
        clock,
      }}
    >
      {children}
    </StaffContext.Provider>
  );
}

export function useStaff(): StaffContextValue {
  const ctx = useContext(StaffContext);
  if (!ctx) throw new Error("useStaff must be used within a StaffProvider");
  return ctx;
}
