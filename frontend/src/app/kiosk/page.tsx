"use client";

// The branch screen: one big rotating QR code, and nothing else.
//
// This is what makes the wall poster idea work at all. A printed code never
// changes, so photographing it once buys a lifetime of clocking in from home.
// The code here is valid for 30 seconds — a photograph is worthless before the
// employee has walked to the car park.
//
// The page is meant to be left open on a cheap tablet, so it keeps the screen
// awake, survives the network dropping, and never needs anybody to sign in
// again: the token is issued once from the admin panel.

import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { ApiError, api, clearKioskToken, getKioskToken, setKioskToken } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import QrCode from "@/components/admin/QrCode";
import type { KioskCode } from "@/lib/types";

export default function KioskPage() {
  const t = useAdminT();
  const params = useSearchParams();
  const [data, setData] = useState<KioskCode | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [ready, setReady] = useState(false);
  const [left, setLeft] = useState(0);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // The token arrives once, in the link the admin opens on the tablet. It is
  // moved straight into storage and out of the address bar, so the screen can
  // be reloaded (or the browser restarted) without the link.
  useEffect(() => {
    const t0 = params.get("t");
    if (t0) {
      setKioskToken(t0);
      window.history.replaceState(null, "", "/kiosk");
    }
    setReady(true);
  }, [params]);

  const load = useCallback(async () => {
    if (!getKioskToken()) {
      setError(t.kiosk.noToken);
      return;
    }
    try {
      const next = await api.kioskCode();
      setData(next);
      setLeft(next.expiresIn);
      setError(null);
      // Refresh exactly when this code dies, plus a moment for clock drift.
      timerRef.current = setTimeout(() => void load(), (next.expiresIn + 1) * 1000);
    } catch (e) {
      const msg = e instanceof ApiError ? e.message : t.kiosk.offline;
      setError(msg);
      if (e instanceof ApiError && e.status === 401) clearKioskToken();
      // Keep trying: a tablet on a wall must recover from a wifi blip on its
      // own, without anybody noticing it had stopped.
      timerRef.current = setTimeout(() => void load(), 5000);
    }
  }, [t]);

  useEffect(() => {
    if (!ready) return;
    void load();
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [ready, load]);

  // Countdown, purely so the room can see the code is alive.
  useEffect(() => {
    const id = setInterval(() => setLeft((s) => (s > 0 ? s - 1 : 0)), 1000);
    return () => clearInterval(id);
  }, []);

  // Keep the screen on — a kiosk that has gone black is a kiosk nobody can use.
  useEffect(() => {
    let lock: { release: () => Promise<void> } | null = null;
    const nav = navigator as Navigator & {
      wakeLock?: { request: (t: "screen") => Promise<{ release: () => Promise<void> }> };
    };
    const acquire = () => {
      nav.wakeLock?.request("screen").then((l) => (lock = l)).catch(() => {});
    };
    acquire();
    const again = () => document.visibilityState === "visible" && acquire();
    document.addEventListener("visibilitychange", again);
    return () => {
      document.removeEventListener("visibilitychange", again);
      void lock?.release().catch(() => {});
    };
  }, []);

  const url =
    data && typeof window !== "undefined"
      ? `${window.location.origin}/staff?c=${encodeURIComponent(data.code)}`
      : "";

  return (
    <main className="flex min-h-dvh flex-col items-center justify-center gap-6 bg-white p-6 text-center">
      {error && !data ? (
        <div className="max-w-md">
          <p className="text-lg font-semibold text-red-600">{error}</p>
          <p className="mt-2 text-sm text-neutral-500">{t.kiosk.noTokenHint}</p>
        </div>
      ) : !data ? (
        <p className="text-neutral-400">{t.common.loading}</p>
      ) : (
        <>
          <div>
            <p className="text-2xl font-bold text-neutral-900">{data.branchName}</p>
            <p className="mt-1 text-neutral-500">{t.kiosk.scanToClock}</p>
          </div>

          {/* Deliberately huge: this is read from across a room, by a phone
              camera held at arm's length. */}
          <QrCode value={url} size={420} className="shadow-lg" />

          <div className="w-72">
            <div className="h-2 w-full overflow-hidden rounded-full bg-neutral-200">
              <div
                className="h-full rounded-full bg-orange-600 transition-[width] duration-1000 ease-linear"
                style={{ width: `${(left / (data.stepSecs || 30)) * 100}%` }}
              />
            </div>
            <p className="mt-2 text-sm text-neutral-500">
              {t.kiosk.refreshIn(left)}
            </p>
          </div>

          <p className="max-w-md text-xs leading-relaxed text-neutral-400">
            {t.kiosk.photoWarning}
          </p>
          {error && <p className="text-xs text-amber-600">{error}</p>}
        </>
      )}
    </main>
  );
}
