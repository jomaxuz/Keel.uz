// What time it is in the restaurant.
//
// ⚠️ **A television's own clock is not usable for this.** A cheap Android set
// with no network at boot comes up in 1970 or on the date it left the factory,
// and it stays there until something corrects it. Every schedule this app
// follows — a promotion that ends on the 30th — is a comparison against that
// clock, so getting it wrong means a dining room showing an offer the till
// stopped honouring last week, or not showing one that started this morning.
//
// So the server says what time it is, on every heartbeat, and this file keeps
// the difference. The device's clock is still used to *count* — it runs at the
// right speed even when it starts from the wrong place.

import { readSaved, saveValue } from "./store";

const CLOCK_KEY = "keel_tv_clock";

/** Milliseconds to add to `Date.now()` to get the restaurant's time. */
let offset = 0;
let known = false;

/** Record what the server just said the time was.
 *
 *  ⚠️ Called from the heartbeat rather than only from the playlist fetch: the
 *  playlist is re-read only when it changes, which on a normal day is never,
 *  and a set left running for a month would drift on whatever its own clock
 *  does. The heartbeat happens every minute anyway and this costs nothing.  */
export function noteServerTime(iso: string): void {
  const server = Date.parse(iso);
  if (!Number.isFinite(server)) return;
  offset = server - Date.now();
  known = true;
  // Persisted as the pair, not as the difference: the difference is only
  // meaningful against the device clock it was measured from, and a set that
  // lost power may come back with a completely different one.
  saveValue(CLOCK_KEY, JSON.stringify({ server, device: Date.now() }));
}

/** Restore the last known offset, so a cold boot with no network still has a
 *  better guess than the television's own idea of the date.
 *
 *  ⚠️ **A guess, and knowingly so.** If the set's clock was reset by the power
 *  cut that caused the reboot, this offset is wrong — but it is wrong by the
 *  same amount the raw clock is, and it is corrected by the first heartbeat
 *  that gets through. What it buys is the minute before that: the screen comes
 *  back to what it was playing rather than to every dated slide at once. */
export function restoreClock(): void {
  const raw = readSaved(CLOCK_KEY);
  if (!raw) return;
  try {
    const { server, device } = JSON.parse(raw) as {
      server: number;
      device: number;
    };
    if (Number.isFinite(server) && Number.isFinite(device)) {
      offset = server - device;
    }
  } catch {
    // A store written by an older build, or half a write. The clock stays the
    // device's own, which is exactly where it was a line ago.
  }
}

/** The restaurant's time, as well as this television can know it. */
export function serverNow(): number {
  return Date.now() + offset;
}

/** Whether the server has confirmed the time this launch.
 *
 *  ⚠️ Used to decide how a dated slide is treated before the first heartbeat
 *  lands — see playlist.ts. It is deliberately not exported as "trust me": the
 *  answer changes what a screen shows, and a caller that cannot tell the two
 *  apart would guess. */
export function clockConfirmed(): boolean {
  return known;
}
