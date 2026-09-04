import { describe, expect, it } from "vitest";

import {
  MAX_HOLD_MINUTES,
  holdClock,
  stopHoldBody,
  typedHold,
} from "./stopHold";

// ⚠️ **Tested here rather than through either screen, because that is the whole
// reason it is a module.** The counter and the panel both offer this control,
// and two readings of "2 hours" in one product is how a restaurant ends up
// unable to say which screen is wrong. The rule has one home and one test; the
// screens are only markup over it.
describe("how long a manual stop holds", () => {
  // ⚠️ **A duration on the wire, never a moment.** The server turns it into an
  // instant on its own clock: a till whose CMOS battery has died reports 2010
  // after a power cut, and a deadline computed in the browser would either lift
  // the second it was written or never lift at all. Neither failure says
  // anything on screen — the dish is simply wrong about being available.
  it("sends a duration, not a deadline", () => {
    expect(stopHoldBody(120)).toEqual({ minutes: 120 });
    expect(stopHoldBody("close")).toEqual({ untilClose: true });
  });

  // ⚠️ **Open-ended sends nothing at all**, which is what this button meant
  // before deadlines existed. A body carrying `minutes: 0` would be a deadline
  // already in the past, and the dish would never leave the menu.
  it("sends no hold at all for an open-ended stop", () => {
    expect(stopHoldBody(null)).toBeUndefined();
  });

  // ⚠️ **An emptied box says "never mind"** — the reading every other field in
  // this product has, and the one somebody expects when they clear a number
  // they typed by mistake.
  it("reads an empty box as no deadline", () => {
    expect(typedHold("")).toEqual({ text: "", hold: null });
    expect(typedHold("0")).toEqual({ text: "0", hold: null });
  });

  it("keeps the digits and selects the hold they name", () => {
    expect(typedHold("25")).toEqual({ text: "25", hold: 25 });
  });

  // ⚠️ Letters are dropped rather than refused: a numeric keypad on a monoblock
  // still emits the odd stray character, and a field that rejected the whole
  // entry would lose the digits somebody had already got right.
  it("keeps only the digits", () => {
    expect(typedHold("4a5").text).toBe("45");
  });

  // ⚠️ Clamped to the same day the server clamps to, so the box cannot display
  // a figure the deadline will not honour. Past a day, "until somebody says
  // otherwise" is the honest setting anyway.
  it("clamps a typed hold to the ceiling the server applies", () => {
    expect(typedHold("9999").hold).toBe(MAX_HOLD_MINUTES);
    expect(MAX_HOLD_MINUTES).toBe(24 * 60);
  });

  // ⚠️ Formatted from the absolute instant the server sent, never from a
  // duration: "90 minutes left" is stale the moment it is drawn, and both of
  // these screens stay open for a whole shift.
  it("shows a deadline as a wall clock", () => {
    const at = new Date();
    at.setHours(21, 5, 0, 0);
    expect(holdClock(at.toISOString())).toBe(
      at.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
    );
  });

  // A row whose deadline never arrived, or arrived broken, must not print
  // "Invalid Date" over the picture of a dish.
  it("says nothing rather than something wrong for an unusable date", () => {
    expect(holdClock("")).toBe("");
    expect(holdClock("kecha")).toBe("");
  });
});
