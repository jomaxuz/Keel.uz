import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { clearAll } from "./store";
import {
  clockRefusal,
  initClock,
  noteServerTime,
  resetClockForTests,
  stamp,
  tillNow,
} from "./clock";

// ⚠️ **This guard exists for a fault nothing else reports.** An old monoblock
// whose CMOS battery has died comes back from a power cut with its clock years
// in the past, and offline the till stamps from that clock — while the fiscal
// register, which is on the restaurant's own network, registers the sale
// anyway. The wrong date reaches a tax document and the evening looks entirely
// ordinary.

/** Move this machine's clock, the way a dead battery does. */
function machineClockAt(iso: string) {
  vi.setSystemTime(new Date(iso));
}

beforeEach(async () => {
  // ⚠️ **Only `Date` is faked.** Faking every timer stops IndexedDB, whose
  // callbacks are real macrotasks — the store then never answers and the test
  // hangs in `beforeEach`, which reads as a bug in the till rather than in the
  // harness.
  vi.useFakeTimers({ toFake: ["Date"] });
  resetClockForTests();
  await clearAll();
});

afterEach(() => {
  vi.useRealTimers();
});

describe("a till that has never spoken to the server", () => {
  it("sells, because refusing there would be a bigger outage than the fault", async () => {
    machineClockAt("2026-08-25T10:00:00.000Z");
    await initClock();
    expect(clockRefusal()).toBeNull();
  });
});

describe("the offset the server teaches it", () => {
  it("stamps the server's time from a machine whose clock is wrong", async () => {
    // The ordinary case: somebody set the hour wrong months ago.
    machineClockAt("2026-08-25T08:00:00.000Z");
    await initClock();
    await noteServerTime("2026-08-25T10:00:00.000Z");
    expect(tillNow().toISOString()).toBe("2026-08-25T10:00:00.000Z");
    expect(stamp()).toBe("2026-08-25T10:00:00.000Z");
  });
});

describe("a clock that has gone backwards", () => {
  it("stops selling, and says which two times disagree", async () => {
    machineClockAt("2026-08-25T20:00:00.000Z");
    await initClock();
    await noteServerTime("2026-08-25T20:00:00.000Z");
    const sold = stamp();

    // The power goes out. It comes back with the battery dead.
    machineClockAt("2010-01-01T00:00:00.000Z");
    const refusal = clockRefusal();
    expect(refusal).not.toBeNull();
    expect(refusal!.afterISO).toBe(sold);
    expect(refusal!.nowISO.startsWith("2010-")).toBe(true);
  });

  it("does not stop selling over a correction of seconds", async () => {
    // ⚠️ Time services adjust by seconds. A till that refused over that would
    // stop a restaurant for a reason nobody could see.
    machineClockAt("2026-08-25T20:00:00.000Z");
    await initClock();
    await noteServerTime("2026-08-25T20:00:00.000Z");
    stamp();
    machineClockAt("2026-08-25T19:59:57.000Z");
    expect(clockRefusal()).toBeNull();
  });

  it("does not stop selling when the clock is merely ahead", async () => {
    // A clock that is ahead is repaired on sync (clampOfflineTime) and stamps
    // an evening that is early. A clock that is behind writes sales before
    // sales that already exist.
    machineClockAt("2026-08-25T20:00:00.000Z");
    await initClock();
    await noteServerTime("2026-08-25T20:00:00.000Z");
    stamp();
    machineClockAt("2026-08-26T09:00:00.000Z");
    expect(clockRefusal()).toBeNull();
  });

  it("opens again the moment the server can be reached", async () => {
    // ⚠️ The way out that does not need anybody to find this file. The server
    // is the authority; a till held shut by a clock it has just corrected would
    // stay shut.
    machineClockAt("2026-08-25T20:00:00.000Z");
    await initClock();
    await noteServerTime("2026-08-25T20:00:00.000Z");
    stamp();
    machineClockAt("2010-01-01T00:00:00.000Z");
    expect(clockRefusal()).not.toBeNull();

    await noteServerTime("2026-08-25T20:05:00.000Z");
    expect(clockRefusal()).toBeNull();
    expect(tillNow().toISOString()).toBe("2026-08-25T20:05:00.000Z");
  });
});

describe("what survives the power cut", () => {
  it("remembers the last moment written, so the guard works on the next start", async () => {
    machineClockAt("2026-08-25T20:00:00.000Z");
    await initClock();
    await noteServerTime("2026-08-25T20:00:00.000Z");
    stamp();
    // ⚠️ `stamp` does not await its own write — the caller is a keystroke. Let
    // the queued write land, which is what the disk does while somebody is
    // reaching for the next dish.
    await Promise.resolve();
    await Promise.resolve();

    // The process dies and comes back with a dead battery.
    resetClockForTests();
    machineClockAt("2010-01-01T00:00:00.000Z");
    await initClock();

    expect(clockRefusal()).not.toBeNull();
  });
});
