"use client";

// Shown instead of the room when this machine's clock has gone backwards.
//
// ⚠️ **A gate, not a banner, for the reason the shift is a gate**: a note at
// the top of a working screen gets worked past, because the first guest is
// already standing there. What is behind it is worse than a missing figure — a
// sale registers with the fiscal register whether or not our server is
// reachable, so a wrong date reaches a tax document and nothing else on any
// screen says so.
//
// ⚠️ **It names both times and both ways out.** "Clock error" is answered with
// a restart, and a restart does not charge a dead CMOS battery. What a cashier
// can act on is: this is what the machine thinks it is, this is what it has to
// be after, fix the date or get the server back.

import { useEffect, useState } from "react";

import { useAdminT } from "@/lib/i18n/admin";
import { clockRefusal, initClock } from "@/lib/offline/clock";

/** How the two moments are shown: enough to see a year that is wrong at a
 *  glance, without a locale's worth of formatting behind it. */
function readable(iso: string): string {
  return iso.replace("T", " ").slice(0, 16);
}

export default function ClockGate({
  refusal,
  onRecheck,
}: {
  refusal: { nowISO: string; afterISO: string };
  onRecheck: () => void;
}) {
  const t = useAdminT();
  // ⚠️ Re-read on a timer as well as on the button: the ordinary way out is
  // the server coming back by itself, and a screen that only cleared on a tap
  // would keep a restaurant shut until somebody thought to press it.
  const [, tick] = useState(0);
  useEffect(() => {
    const timer = setInterval(() => tick((n) => n + 1), 10_000);
    return () => clearInterval(timer);
  }, []);

  return (
    <div className="flex h-full items-center justify-center p-6">
      <div className="card max-w-lg space-y-4 p-6 text-center">
        <h1 className="text-xl font-semibold text-danger">{t.till.clockTitle}</h1>
        <p className="text-sm text-ink">{t.till.clockBody}</p>
        <dl className="mx-auto grid max-w-xs grid-cols-2 gap-x-4 gap-y-1 text-sm">
          <dt className="text-right text-ink-muted">{t.till.clockNow}</dt>
          <dd className="text-left tabular-nums">{readable(refusal.nowISO)}</dd>
          <dt className="text-right text-ink-muted">{t.till.clockAfter}</dt>
          <dd className="text-left tabular-nums">
            {readable(refusal.afterISO)}
          </dd>
        </dl>
        <p className="text-sm text-ink-muted">{t.till.clockFix}</p>
        <button type="button" className="btn-primary" onClick={onRecheck}>
          {t.till.clockRetry}
        </button>
      </div>
    </div>
  );
}

/** The refusal, re-read often enough that the screen clears itself.
 *
 *  ⚠️ Polled rather than pushed: what lifts it is the server answering a sync
 *  somewhere else entirely, and a subscription between the two would be a
 *  second mechanism to keep honest for no gain at ten seconds' resolution. */
export function useClockRefusal(active: boolean) {
  const [refusal, setRefusal] = useState<ReturnType<typeof clockRefusal>>(null);
  useEffect(() => {
    if (!active) return;
    let alive = true;
    const read = () => alive && setRefusal(clockRefusal());
    // ⚠️ **Loaded before the first read, and the first read is what decides
    // whether the room is drawn.** Reading an unloaded offset answers "no
    // refusal" — which is the safe direction for a till that has never been
    // offline and the wrong one for the machine this guard exists for.
    void initClock().then(read);
    const timer = setInterval(read, 10_000);
    return () => {
      alive = false;
      clearInterval(timer);
    };
  }, [active]);
  return { refusal, recheck: () => setRefusal(clockRefusal()) };
}
