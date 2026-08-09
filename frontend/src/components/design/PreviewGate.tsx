"use client";

// Loads the geometry bridge, and only for a console preview.
//
// ⚠️ **This tiny file is the whole point.** The bridge itself is imported
// dynamically, so its code sits in its own chunk and is fetched only when this
// component actually runs — which is only when the page was rendered with a valid
// preview token. A statically imported bridge would have shipped to every guest
// and merely never executed, and "it ships but does not run" is a weaker promise
// than the one worth making. Same rule as the Telegram SDK, which is loaded only
// inside Telegram.

import { useEffect } from "react";

export default function PreviewGate() {
  useEffect(() => {
    let stop: (() => void) | undefined;
    let cancelled = false;
    void import("./PreviewBridge").then((m) => {
      if (cancelled) return;
      stop = m.startBridge();
    });
    return () => {
      cancelled = true;
      stop?.();
    };
  }, []);
  return null;
}
