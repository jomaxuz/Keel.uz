import { afterEach, beforeEach, describe, expect, it } from "vitest";

import {
  apiOverride,
  dropSession,
  dropToken,
  readSession,
  readToken,
  setApiBase,
  setSessionStore,
  setTokenStore,
  uploadsOverride,
  writeSession,
  writeToken,
} from "./tokenStore";

// ⚠️ **This is the seam a phone needs, and the risk is that it changes the
// web.** Every rule in `lib/` is plain TypeScript and runs anywhere; the two
// things that are not are where a token is kept and where the server is. They
// were assumed in thirty places and are asked in one now — so what has to be
// sealed is that asking produces exactly the old answers when nobody has
// swapped anything.

/** The shape React Native will hand in: a map, standing in for a store that is
 *  really async underneath and hydrated at startup. */
function memoryStore() {
  const rows = new Map<string, string>();
  return {
    rows,
    get: (k: string) => rows.get(k) ?? null,
    set: (k: string, v: string) => void rows.set(k, v),
    remove: (k: string) => void rows.delete(k),
  };
}

beforeEach(() => {
  setApiBase("", "");
  window.localStorage.clear();
  window.sessionStorage.clear();
});

afterEach(() => {
  setSessionStore({
    get: (k) => window.sessionStorage.getItem(k),
    set: (k, v) => window.sessionStorage.setItem(k, v),
    remove: (k) => window.sessionStorage.removeItem(k),
  });
  // Back to the browser for whatever runs next.
  setTokenStore({
    get: (k) => window.localStorage.getItem(k),
    set: (k, v) => window.localStorage.setItem(k, v),
    remove: (k) => window.localStorage.removeItem(k),
  });
  setApiBase("", "");
});

describe("the browser, which must not have changed", () => {
  it("keeps tokens in localStorage under the same keys", () => {
    // The keys are what an installed till already has on disk: reading them
    // through a different name would sign every machine out on deploy.
    writeToken("admin_token", "abc");
    expect(window.localStorage.getItem("admin_token")).toBe("abc");
    expect(readToken("admin_token")).toBe("abc");

    dropToken("admin_token");
    expect(window.localStorage.getItem("admin_token")).toBeNull();
    expect(readToken("admin_token")).toBeNull();
  });

  it("reads an absent token as absent rather than throwing", () => {
    expect(readToken("nothing_here")).toBeNull();
  });
});

describe("a platform that is not a browser", () => {
  it("takes over completely once it says so", () => {
    const store = memoryStore();
    setTokenStore(store);

    writeToken("staff_token", "pin-token");
    expect(store.rows.get("staff_token")).toBe("pin-token");
    expect(readToken("staff_token")).toBe("pin-token");
    // ⚠️ And nothing leaked into the browser's storage on the way past: a
    // token written to two places is a token cleared from one of them.
    expect(window.localStorage.getItem("staff_token")).toBeNull();
  });

  it("is read synchronously, because every request reads one", () => {
    // The reason the interface is not async: `request()` reads a token on the
    // way into each call, so React Native's adapter hydrates once at startup
    // and carries the asymmetry by itself.
    const store = memoryStore();
    store.rows.set("user_token", "hydrated");
    setTokenStore(store);
    expect(readToken("user_token")).toBe("hydrated");
  });
});

describe("where the server is", () => {
  it("is nothing until somebody says, so the web app keeps its own default", () => {
    expect(apiOverride()).toBe("");
    expect(uploadsOverride()).toBe("");
  });

  it("is a whole address once set, because a phone has no origin", () => {
    setApiBase("https://osh.keel.uz/api/v1", "https://osh.keel.uz/uploads");
    expect(apiOverride()).toBe("https://osh.keel.uz/api/v1");
    expect(uploadsOverride()).toBe("https://osh.keel.uz/uploads");
  });
});

describe("the session, which is a different lifetime and not a smaller store", () => {
  it("is the browser's sessionStorage by default", () => {
    writeSession("keel_till_token", "unlocked");
    expect(window.sessionStorage.getItem("keel_till_token")).toBe("unlocked");
    expect(readSession("keel_till_token")).toBe("unlocked");
    dropSession("keel_till_token");
    expect(readSession("keel_till_token")).toBeNull();
  });

  it("can be memory, which on a phone is exactly the intended lifetime", () => {
    // ⚠️ A till session belongs to this sitting at this screen: closing the app
    // must lock it, and a killed app *is* the end of the session. The two
    // platforms differ in mechanism and agree on meaning.
    const rows = new Map<string, string>();
    setSessionStore({
      get: (k) => rows.get(k) ?? null,
      set: (k, v) => void rows.set(k, v),
      remove: (k) => void rows.delete(k),
    });
    writeSession("keel_till_token", "unlocked");
    expect(rows.get("keel_till_token")).toBe("unlocked");
    // And it did not also land somewhere that survives a restart.
    expect(window.sessionStorage.getItem("keel_till_token")).toBeNull();
    expect(window.localStorage.getItem("keel_till_token")).toBeNull();
  });
});
