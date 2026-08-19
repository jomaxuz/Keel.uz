"use client";

// What the till says back, in the corner, briefly.
//
// ⚠️ **A message that pushes the screen down is a message that moves the
// buttons.** The notice and the refusal used to be strips above the room: a
// cashier reaching for a table watched the whole grid jump a row, and pressed
// the tile that had moved into their finger. The receipt of the last mistake
// stayed there until somebody dismissed it, which nobody does mid-service.
//
// ⚠️ **It closes itself after five seconds**, because none of these need an
// answer — "sent to the kitchen", "the check is closed", "the dish has run
// out". The one thing this must never carry is a state: a lost connection is
// not a notification, it stays a banner, because it is still true a minute
// later.

import { useEffect } from "react";

export type Toast = {
  id: number;
  text: string;
  kind: "info" | "error";
};

/** How long a message stays. Five seconds is long enough to read a sentence
 *  twice and short enough that the next one is not queued behind it. */
const LIFETIME = 5000;

export default function Toasts({
  items,
  onDismiss,
}: {
  items: Toast[];
  onDismiss: (id: number) => void;
}) {
  useEffect(() => {
    if (items.length === 0) return;
    // One timer per message, cleared together: a single timer for the newest
    // would leave an older one on screen forever the moment two arrive.
    const timers = items.map((t) =>
      window.setTimeout(() => onDismiss(t.id), LIFETIME),
    );
    return () => timers.forEach(window.clearTimeout);
  }, [items, onDismiss]);

  if (items.length === 0) return null;

  return (
    // ⚠️ Over everything and pointer-transparent except the cards themselves:
    // the till behind it stays usable while a message is up, which is the
    // whole reason it is not a strip.
    <div className="pointer-events-none fixed right-3 top-3 z-[60] flex w-[min(22rem,calc(100vw-1.5rem))] flex-col gap-2">
      {items.map((t) => (
        <button
          key={t.id}
          onClick={() => onDismiss(t.id)}
          className={`pointer-events-auto rounded-[12px] px-3.5 py-2.5 text-left text-[14px] font-medium shadow-lg transition ${
            t.kind === "error"
              ? "bg-danger text-white"
              : "bg-ink text-white/95"
          }`}
        >
          {t.text}
        </button>
      ))}
    </div>
  );
}
