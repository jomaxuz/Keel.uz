"use client";

// What time the till thinks it is, and when it must stop selling.
//
// ⚠️ **This is the quietest way a till breaks.** An old monoblock whose CMOS
// battery has died comes back from a power cut with its clock years in the
// past. Offline, the till stamps from that clock — and a sale offline still
// registers with the fiscal register, which is on the restaurant's own network.
// So the wrong date reaches a tax document, and nothing on any screen says so:
// the evening looks completely ordinary. `docs/pos-reja.md` §6 names the rule,
// and this is the half that lives on the device; `clampOfflineTime` on the
// server is the other half, and it can only repair what has already been sold.
//
// Two things are kept:
//
//   - **The offset** between the server's clock and this machine's, learned
//     whenever the server answers. A till whose clock is simply wrong — set to
//     the wrong hour by somebody, drifted by a minute a week — then stamps the
//     right time anyway, and that is the ordinary case.
//   - **The last moment anything was written.** Time does not run backwards, so
//     a till that now believes it is *before* its own last sale is a till whose
//     clock cannot be trusted at all — and the offset cannot save it, because
//     the offset is a constant and the clock jumped.
//
// ⚠️ **Selling stops; the screen does not.** The refusal has to be readable and
// fixable at a counter: what the machine thinks the time is, what it should be
// after, and that the way out is the machine's own date or the server coming
// back. A till that merely misbehaves gets restarted, and a restart does not
// change a dead battery.

import { all, META, put } from "./store";

/** How far the clock may drift backwards before it is a jump rather than a
 *  correction.
 *
 *  ⚠️ Not zero. Time services adjust by seconds, and a till that refused to
 *  sell over a two-second correction would stop a restaurant for a reason
 *  nobody could see. Two minutes is the same tolerance the server allows in
 *  the other direction (`clampOfflineTime`), for the same reason. */
const TOLERANCE_MS = 2 * 60 * 1000;

/** One row, in the store the queues already use. ⚠️ Kept on the same disk as
 *  the sales rather than in `localStorage`: this is the fact that decides
 *  whether an evening may be sold at all, and it has to survive the same power
 *  cut the sales do. */
const KEY = "clock";

type ClockRow = {
  clientId: typeof KEY;
  /** serverTime − this machine's clock, in milliseconds. */
  offsetMs: number;
  /** The latest moment this till has written anything at, in till time. */
  lastEventMs: number;
};

let state: ClockRow = { clientId: KEY, offsetMs: 0, lastEventMs: 0 };
let loaded = false;

/** Read what was saved. Called once by the screens before anything is sold.
 *
 *  ⚠️ Failing to load is not failing to sell: a till with no saved offset is a
 *  till on its first evening, and refusing there would make the guard a bigger
 *  outage than the fault it exists for. */
export async function initClock(): Promise<void> {
  if (loaded) return;
  loaded = true;
  const rows = await all<ClockRow>(META);
  const row = rows.find((r) => r.clientId === KEY);
  if (row && Number.isFinite(row.offsetMs) && Number.isFinite(row.lastEventMs)) {
    state = { ...row, clientId: KEY };
  }
}

async function save(): Promise<void> {
  await put(META, state);
}

/** The server told us what time it is. Called from the sync, which carries
 *  `serverTime` in every reply for exactly this. */
export async function noteServerTime(iso: string): Promise<void> {
  const server = Date.parse(iso);
  if (!Number.isFinite(server)) return;
  const offset = server - Date.now();
  // ⚠️ **Learning the offset also clears a refusal**, and it has to: the server
  // is the authority, and a till held shut by a clock the server has just
  // corrected would stay shut until somebody found this file. The last event is
  // pulled forward to the corrected present only when it is now in the future,
  // which is the same jump seen from the other side.
  state.offsetMs = offset;
  const now = Date.now() + offset;
  if (state.lastEventMs > now) state.lastEventMs = now;
  await save();
}

/** What time it is, as this till should stamp it. */
export function tillNow(): Date {
  return new Date(Date.now() + state.offsetMs);
}

/**
 * The moment to write on something being saved right now.
 *
 * ⚠️ Recording the event is the same act as stamping it. A separate "remember
 * this" call is one every future caller can forget, and the first one that does
 * is the one that lets a backwards clock through.
 */
export function stamp(): string {
  const now = Date.now() + state.offsetMs;
  if (now > state.lastEventMs) {
    state.lastEventMs = now;
    // ⚠️ Not awaited. The caller is a keystroke, and the write it is part of is
    // already being persisted; blocking a dish on a second disk round trip
    // would be felt at the counter. Losing this write costs one comparison
    // after a power cut, and the sale itself is what carries the real time.
    void save();
  }
  return new Date(now).toISOString();
}

/**
 * Why this till must not sell, or null.
 *
 * ⚠️ **Only backwards.** A clock that is ahead is repaired by the server on
 * sync (`clampOfflineTime`) and stamps an evening that is merely early;
 * a clock that is behind writes sales *before* sales that already exist, and
 * every report that orders by time then disagrees with the paper.
 */
export function clockRefusal(): { nowISO: string; afterISO: string } | null {
  if (state.lastEventMs === 0) return null;
  const now = Date.now() + state.offsetMs;
  if (now >= state.lastEventMs - TOLERANCE_MS) return null;
  return {
    nowISO: new Date(now).toISOString(),
    afterISO: new Date(state.lastEventMs).toISOString(),
  };
}

/** Used by the tests, and by nothing else. */
export function resetClockForTests(): void {
  state = { clientId: KEY, offsetMs: 0, lastEventMs: 0 };
  loaded = false;
}
