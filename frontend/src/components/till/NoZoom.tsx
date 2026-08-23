"use client";

/**
 * The till is an appliance, so it does not pinch.
 *
 * ⚠️ **Zoom on a fixed panel is not a feature, it is a fault report.** A
 * monoblock is one screen size that never changes and the layout is built for
 * it; a stray two-finger drag while reaching across the counter leaves the
 * dish grid at 140% with the totals off the right edge, and there is no visible
 * control to put it back. What the cashier sees is a till that "broke", and
 * what they do is call somebody. Double-tap zoom is worse: it fires from an
 * ordinary impatient double press on a tile.
 *
 * ⚠️ **The viewport meta is not enough and never was.** `user-scalable=no` is
 * ignored by Safari (deliberately, since iOS 10) and by desktop browsers
 * entirely — and the desktop ones are what most of these monoblocks run. So the
 * gestures are refused here, one by one, where the browser actually offers
 * them:
 *
 *   - `gesturestart/change/end` — Safari's own pinch events.
 *   - a two-finger `touchmove` — every other touch browser's pinch.
 *   - `wheel` with Ctrl held — a trackpad pinch and a mouse wheel zoom, the
 *     desktop pair, and the one that actually happens on a counter machine
 *     with a mouse plugged in.
 *   - Ctrl/⌘ with `+ - 0` — a keypad or scanner sending a stray modifier.
 *   - a second tap inside 300 ms — double-tap zoom, refused only when it is
 *     *not* on something meant to be double-pressed.
 *
 * ⚠️ **Scrolling is untouched.** Every listener here refuses zoom specifically:
 * a one-finger `touchmove` passes through, because a till that cannot scroll
 * its check is worse than one that can be pinched.
 *
 * ⚠️ **Non-passive on purpose.** `touchmove` and `wheel` are passive by default
 * in every modern browser, and a passive listener's `preventDefault()` is
 * ignored with a console warning — the version of this that "worked in
 * development" did nothing at all on the machine it was for.
 */

import { useEffect } from "react";

export default function NoZoom() {
  useEffect(() => {
    const stop = (e: Event) => e.preventDefault();

    const onTouchMove = (e: TouchEvent) => {
      if (e.touches.length > 1) e.preventDefault();
    };

    const onWheel = (e: WheelEvent) => {
      if (e.ctrlKey || e.metaKey) e.preventDefault();
    };

    const onKey = (e: KeyboardEvent) => {
      if (!(e.ctrlKey || e.metaKey)) return;
      if (["+", "=", "-", "_", "0"].includes(e.key)) e.preventDefault();
    };

    // ⚠️ The gap, not a timer: two taps inside 300 ms are a double-tap however
    // slowly the second one arrives after that.
    let lastTap = 0;
    const onTouchEnd = (e: TouchEvent) => {
      const now = Date.now();
      if (now - lastTap < 300) e.preventDefault();
      lastTap = now;
    };

    const opts = { passive: false } as const;
    document.addEventListener("gesturestart", stop, opts);
    document.addEventListener("gesturechange", stop, opts);
    document.addEventListener("gestureend", stop, opts);
    document.addEventListener("touchmove", onTouchMove, opts);
    document.addEventListener("touchend", onTouchEnd, opts);
    document.addEventListener("wheel", onWheel, opts);
    window.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("gesturestart", stop);
      document.removeEventListener("gesturechange", stop);
      document.removeEventListener("gestureend", stop);
      document.removeEventListener("touchmove", onTouchMove);
      document.removeEventListener("touchend", onTouchEnd);
      document.removeEventListener("wheel", onWheel);
      window.removeEventListener("keydown", onKey);
    };
  }, []);

  return null;
}
