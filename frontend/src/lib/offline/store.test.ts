import { beforeEach, describe, expect, it, vi } from "vitest";

// ⚠️ **The risk this covers is a till that quietly uses the wrong disk.**
// Inside the Windows application the sales belong in SQLite, where a power cut
// cannot take the last writes; in a browser they belong in IndexedDB. Both
// engines work, so choosing the weaker one inside the application fails at no
// point except the evening the lights go out — and then completely.

type Rec = { clientId: string; total?: number };

/** A stand-in for the Go side: the four calls, backed by a map. */
function fakeApp(ready = true) {
  const rows = new Map<string, string>();
  const app = {
    StoreReady: vi.fn(async () => ready),
    StorePut: vi.fn(async (store: string, key: string, value: string) => {
      rows.set(`${store}/${key}`, value);
    }),
    StoreAll: vi.fn(async (store: string) =>
      [...rows.entries()]
        .filter(([k]) => k.startsWith(`${store}/`))
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([, v]) => v),
    ),
    StoreRemove: vi.fn(async (store: string, key: string) => {
      rows.delete(`${store}/${key}`);
    }),
    StoreClear: vi.fn(async () => {
      rows.clear();
    }),
  };
  (window as unknown as { go?: unknown }).go = { main: { App: app } };
  return app;
}

/** Fresh module, because the "is there a database?" answer is remembered for
 *  the life of the process — which is correct in a till and wrong across
 *  tests. */
async function load() {
  vi.resetModules();
  return import("./store");
}

beforeEach(() => {
  delete (window as unknown as { go?: unknown }).go;
  vi.restoreAllMocks();
});

describe("inside the Windows application", () => {
  it("writes to the machine's own database and not to the browser's", async () => {
    const app = fakeApp();
    const store = await load();

    expect(await store.put(store.LOCAL_CHECKS, { clientId: "c1", total: 42000 })).toBe(true);
    expect(app.StorePut).toHaveBeenCalledWith(
      "localChecks",
      "c1",
      JSON.stringify({ clientId: "c1", total: 42000 }),
    );

    const back = await store.all<Rec>(store.LOCAL_CHECKS);
    expect(back).toEqual([{ clientId: "c1", total: 42000 }]);
  });

  it("asks whether the database is there exactly once", async () => {
    // The screens save on every keystroke; the answer cannot change while the
    // process runs.
    const app = fakeApp();
    const store = await load();
    await store.put(store.PENDING, { clientId: "a" });
    await store.put(store.PENDING, { clientId: "b" });
    await store.all(store.PENDING);
    expect(app.StoreReady).toHaveBeenCalledTimes(1);
  });

  it("falls back to the browser when the database could not be opened", async () => {
    // ⚠️ A till that cannot open its own file still has to sell. The fallback
    // is a weaker promise, which is why the code says so in the log.
    const app = fakeApp(false);
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const store = await load();

    expect(await store.put(store.LOCAL_CHECKS, { clientId: "c1" })).toBe(true);
    expect(app.StorePut).not.toHaveBeenCalled();
    expect(warn).toHaveBeenCalled();
    // And the sale is really there, in the other engine.
    expect(await store.all<Rec>(store.LOCAL_CHECKS)).toEqual([{ clientId: "c1" }]);
    await store.clearAll();
  });

  it("refuses a record with no id rather than inventing a key", async () => {
    // The id is the whole of the idempotency: stored under a made-up key it
    // would sync as a second dinner on every retry.
    fakeApp();
    const store = await load();
    expect(await store.put(store.LOCAL_CHECKS, { total: 42000 })).toBe(false);
  });

  it("drops one unreadable record instead of the whole evening", async () => {
    const app = fakeApp();
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const store = await load();
    await store.put(store.LOCAL_CHECKS, { clientId: "c1" });
    app.StoreAll.mockResolvedValueOnce(['{"clientId":"c1"}', "{not json"]);

    expect(await store.all<Rec>(store.LOCAL_CHECKS)).toEqual([{ clientId: "c1" }]);
    expect(warn).toHaveBeenCalled();
  });

  it("removes by the same key it wrote", async () => {
    const app = fakeApp();
    const store = await load();
    await store.put(store.PENDING, { clientId: "c1" });
    await store.remove(store.PENDING, "c1");
    expect(app.StoreRemove).toHaveBeenCalledWith("pendingSales", "c1");
    expect(await store.all(store.PENDING)).toEqual([]);
  });
});

describe("in a plain browser", () => {
  it("uses IndexedDB and never looks for a Go side", async () => {
    const store = await load();
    expect(store.available()).toBe(true);
    expect(await store.put(store.LOCAL_CHECKS, { clientId: "b1", total: 7 })).toBe(true);
    expect(await store.all<Rec>(store.LOCAL_CHECKS)).toEqual([
      { clientId: "b1", total: 7 },
    ]);
    await store.clearAll();
  });
});

describe("a platform that is neither, handed in from outside", () => {
  it("takes precedence, because the phone has no IndexedDB to fall back to", async () => {
    // ⚠️ This is the failure it fixes: React Native provides no IndexedDB, so
    // `available()` answered false and the queue silently did nothing — an app
    // that looked entirely normal and stopped selling when the wifi dropped.
    const store = await load();
    const rows = new Map<string, string>();
    store.setNativeStore({
      put: async (s2, k, v) => {
        rows.set(`${s2}/${k}`, v);
        return true;
      },
      all: async (s2) =>
        [...rows.entries()]
          .filter(([k]) => k.startsWith(`${s2}/`))
          .map(([, v]) => v),
      remove: async (s2, k) => void rows.delete(`${s2}/${k}`),
      clear: async () => rows.clear(),
    });

    expect(store.available()).toBe(true);
    expect(await store.put(store.LOCAL_CHECKS, { clientId: "p1", total: 9 })).toBe(true);
    expect(await store.all<Rec>(store.LOCAL_CHECKS)).toEqual([
      { clientId: "p1", total: 9 },
    ]);
    // And nothing leaked into the browser's storage on the way past.
    expect(window.localStorage.getItem("p1")).toBeNull();

    await store.remove(store.PENDING, "nope");
    await store.clearAll();
    expect(await store.all(store.LOCAL_CHECKS)).toEqual([]);
  });

  it("still refuses a record with no id", async () => {
    const store = await load();
    store.setNativeStore({
      put: async () => true,
      all: async () => [],
      remove: async () => {},
      clear: async () => {},
    });
    expect(await store.put(store.LOCAL_CHECKS, { total: 1 })).toBe(false);
  });
});
