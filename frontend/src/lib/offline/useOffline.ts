"use client";

// Whether the till can reach the server, and what is waiting if it cannot.
//
// ⚠️ **`navigator.onLine` is not the question.** It answers "is there a cable
// or a wifi association", which on a restaurant's router is true throughout an
// outage of the internet behind it — and false for about a second when the
// access point roams, which would flap a banner in front of a cashier for no
// reason. What the till actually needs to know is whether *our* requests are
// getting through, and it finds that out by making them.

import { useCallback, useEffect, useMemo, useState } from "react";

import { drainLocalChecks, drainSales, pendingSales } from "./sales";

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
    const { localChecks } = await import("./checks");
    const owed = (await localChecks()).filter((c) => c.paidAt).length;
    setPending((await pendingSales()).length + owed);
  }, []);

  const flush = useCallback(async () => {
    // ⚠️ The payments first: they are money already taken, and a check opened
    // offline is only a table until it is paid for.
    const left = (await drainSales()) + (await drainLocalChecks());
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

  // ⚠️ **One object, not a fresh one per render.** This is read by screens that
  // put it in a dependency array — the till's check poll does — and a literal
  // built during render is a new identity every time, so the effect holding the
  // poll tears itself down and starts again on every render. That would be
  // merely wasteful if the poll did nothing; it fetches and then calls
  // setState, so each render schedules the work that causes the next render,
  // and each pass schedules two of them. It compounds: within seconds of a PIN
  // being accepted the till is making hundreds of requests a second and the
  // window stops answering. It looked like the app freezing on unlock, because
  // that is when the poll starts.
  //
  // `seen` and `flush` are already stable, so this changes identity only when
  // one of the two values actually changed — which is what a consumer means by
  // "the network changed".
  return useMemo(
    () => ({ online, pending, seen, flush }),
    [online, pending, seen, flush],
  );
}
