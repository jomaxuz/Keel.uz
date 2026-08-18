"use client";

// Whether the till can reach the server, and what is waiting if it cannot.
//
// ⚠️ **`navigator.onLine` is not the question.** It answers "is there a cable
// or a wifi association", which on a restaurant's router is true throughout an
// outage of the internet behind it — and false for about a second when the
// access point roams, which would flap a banner in front of a cashier for no
// reason. What the till actually needs to know is whether *our* requests are
// getting through, and it finds that out by making them.

import { useCallback, useEffect, useState } from "react";

import { drainSales, pendingSales } from "./sales";

export interface OfflineState {
  /** False once a request has failed for the network's reasons, true again as
   *  soon as one succeeds. */
  online: boolean;
  /** Payments waiting to be confirmed by the server. */
  pending: number;
  /** Called by the screens: every till request reports what happened. */
  seen: (ok: boolean) => void;
  /** Send what is waiting now — the button on the banner. */
  flush: () => Promise<void>;
}

export function useOffline(active: boolean): OfflineState {
  const [online, setOnline] = useState(true);
  const [pending, setPending] = useState(0);

  const refresh = useCallback(async () => {
    setPending((await pendingSales()).length);
  }, []);

  const flush = useCallback(async () => {
    const left = await drainSales();
    setPending(left);
    // ⚠️ An empty queue is not proof the server is back — there may have been
    // nothing to send. The flag is only turned on by a request that succeeded.
    if (left === 0) void refresh();
  }, [refresh]);

  const seen = useCallback(
    (ok: boolean) => {
      setOnline((was) => {
        if (was === ok) return was;
        // Coming back: send what is waiting immediately rather than on the next
        // tick. The cashier is standing there and the badge is the thing they
        // are looking at.
        if (ok) void flush();
        return ok;
      });
    },
    [flush],
  );

  useEffect(() => {
    if (!active) return;
    void refresh();
    // ⚠️ Slow, and only a fallback. The real signal is a request succeeding or
    // failing, which happens every few seconds anyway because the check list
    // polls — this is for the till that is sitting idle at four in the
    // afternoon with a queue on it.
    const timer = setInterval(() => void flush(), 30_000);
    const back = () => void flush();
    window.addEventListener("online", back);
    return () => {
      clearInterval(timer);
      window.removeEventListener("online", back);
    };
  }, [active, flush, refresh]);

  return { online, pending, seen, flush };
}
