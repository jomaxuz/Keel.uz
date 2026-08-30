import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// ⚠️ **Re-imported per test, because the ceiling is module state.** `sent` is
// deliberately per page load in production — that is the whole protection — so
// a shared import would let the first test here exhaust it and leave every test
// after it asserting against silence that has nothing to do with the code.
async function freshReporter() {
  vi.resetModules();
  return await import("./report");
}

// The reporter is the code that runs when everything else has already gone
// wrong, so what is tested here is mostly what it refuses to do.

function beacons(): string[] {
  const fn = navigator.sendBeacon as unknown as ReturnType<typeof vi.fn>;
  return fn.mock.calls.map((c: unknown[]) => String(c[1]));
}

beforeEach(() => {
  vi.useFakeTimers();
  Object.defineProperty(navigator, "sendBeacon", {
    configurable: true,
    writable: true,
    value: vi.fn(() => true),
  });
  localStorage.clear();
});

// ⚠️ **Real timers again before anything else runs.** The shared setup file's
// own afterEach awaits an IndexedDB clear; with fake timers still installed
// that promise never settles and the whole file fails on a hook timeout — which
// reads as three broken tests rather than one line of setup.
afterEach(() => {
  vi.useRealTimers();
});

describe("crash reporter", () => {
  // ⚠️ **A render loop calls this on every frame.** Without a ceiling the first
  // broken component posts continuously for as long as the tab is open, which
  // is a denial of service the restaurant runs against itself — and it would
  // arrive as "one restaurant is very broken" rather than as a bug in here.
  it("stops after a handful of distinct faults on one page", async () => {
    const { installReporter, report } = await freshReporter();
    installReporter("panel");
    for (let i = 0; i < 50; i++) report(new Error(`fault ${i}`));
    vi.runAllTimers();

    const posted = beacons()
      .map((b) => JSON.parse(b).reports.length)
      .reduce((a: number, b: number) => a + b, 0);
    expect(posted).toBeGreaterThan(0);
    expect(posted).toBeLessThanOrEqual(8);
  });

  // ⚠️ Anything can be thrown in JavaScript. `err.message` on a string, a
  // number or `undefined` is undefined — which used to arrive as the literal
  // word "undefined" and collapsed every unrelated fault into one row.
  it("describes whatever was thrown, not just Errors", async () => {
    const { installReporter, report } = await freshReporter();
    installReporter("till");
    report("kassa javob bermadi");
    report(404);
    vi.runAllTimers();

    const msgs = beacons().flatMap((b) =>
      JSON.parse(b).reports.map((r: { message: string }) => r.message),
    );
    expect(msgs).toContain("kassa javob bermadi");
    expect(msgs.some((m: string) => m.includes("404"))).toBe(true);
    expect(msgs).not.toContain("undefined");
  });

  // ⚠️ The reporter failing is not itself reportable. A throw inside an error
  // handler turns one broken screen into a broken screen whose error handler is
  // also broken — in the code least likely to be exercised.
  it("never throws, even when the transport does", async () => {
    Object.defineProperty(navigator, "sendBeacon", {
      configurable: true,
      writable: true,
      value: () => {
        throw new Error("beacon exploded");
      },
    });
    const { installReporter, report } = await freshReporter();
    installReporter("panel");
    expect(() => {
      report(new Error("something"));
      vi.runAllTimers();
    }).not.toThrow();
  });
});
