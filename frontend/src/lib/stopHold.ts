// How long a manual stop holds, for the two screens that offer it.
//
// ⚠️ **One module because the panel and the till offer the same control on the
// same list.** `soldOutHeldBy` on the server is a single function for exactly
// this reason: two copies of a stop-list rule eventually disagree, and a
// restaurant where the counter's "2 hours" means something different from the
// panel's has no way to tell which screen is lying. What is deliberately *not*
// shared is the markup — the till draws monoblock controls sized for a thumb
// and the panel draws a form — but what an empty box means, and where the
// ceiling is, has one answer.

/** A chosen hold: minutes, "until we close", or open-ended. */
export type StopHold = number | "close" | null;

/** The body the API takes, or nothing at all for an open-ended stop.
 *
 *  ⚠️ **A duration, never a moment.** The server turns it into an instant on
 *  its own clock: a till whose CMOS battery has died reports 2010 after a power
 *  cut, and a deadline computed in the browser would either lift the second it
 *  was written or never lift at all. Neither failure says anything on screen —
 *  the dish is simply wrong about being available. The panel sends it the same
 *  way, so there is one shape on the wire rather than one per screen. */
export function stopHoldBody(
  hold: StopHold,
): { minutes?: number; untilClose?: boolean } | undefined {
  if (hold === null) return undefined;
  if (hold === "close") return { untilClose: true };
  return { minutes: hold };
}

/** The longest hold either screen will send.
 *
 *  ⚠️ The server clamps to the same day. Past it "until somebody says
 *  otherwise" is the honest setting, and a deadline nobody will be present for
 *  is one that surprises the next shift. Matching the ceiling here means the
 *  field cannot display a figure the deadline will not honour. */
export const MAX_HOLD_MINUTES = 24 * 60;

/** What a typed minutes box means: the text to keep, and the hold it selects.
 *
 *  ⚠️ **Empty is "no deadline", not "stop it for zero minutes."** The second
 *  would be a deadline already in the past and a dish that never leaves the
 *  menu — and an emptied field says "never mind" everywhere else in this
 *  product, which is the reading somebody expects. */
export function typedHold(text: string): { text: string; hold: StopHold } {
  const digits = text.replace(/\D/g, "").slice(0, 4);
  const n = Number(digits);
  return {
    text: digits,
    hold: n > 0 ? Math.min(n, MAX_HOLD_MINUTES) : null,
  };
}

/** A deadline as a wall clock, in the reader's own timezone.
 *
 *  ⚠️ **Formatted from the absolute instant the server sent**, never from a
 *  duration: "90 minutes left" is stale the moment it is drawn, and both of
 *  these screens stay open for a whole shift. */
export function holdClock(until: string): string {
  const at = new Date(until);
  return Number.isNaN(at.getTime())
    ? ""
    : at.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}
