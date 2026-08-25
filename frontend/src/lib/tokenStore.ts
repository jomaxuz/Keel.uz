// Where a token is kept, and how the app is reached.
//
// ⚠️ **The seam a second platform needs, and the only one it needs.** Every
// rule in `lib/` is plain TypeScript — the offline queue, the marking codes,
// the service charge, the clock — and all of it runs unchanged anywhere
// JavaScript runs. What does not is this: the browser keeps tokens in
// `localStorage` and reads the API address out of `NEXT_PUBLIC_*` at build
// time, and a phone has neither.
//
// So the platform is asked once, here, instead of being assumed in thirty
// places. Same shape as `tillBridge` for the Windows app: one hand-written
// interface, and every screen above it stays the same code.
//
// ⚠️ **The interface is synchronous, and that is a decision rather than an
// oversight.** A token is read on the way into every request — `getToken()` is
// called by `request()` itself — so an async read would make every call site
// await something that is almost always already known. React Native's stores
// (SecureStore, AsyncStorage) *are* async, so its adapter reads once at
// startup into memory and writes through in the background. That is the right
// place for the asymmetry: one adapter carries it, and nothing above this file
// learns about it.

/** What the app uses to remember a token between launches. */
export interface TokenStore {
  get(key: string): string | null;
  set(key: string, value: string): void;
  remove(key: string): void;
}

/** The browser's own, and the default: nothing changes for the web app.
 *
 *  ⚠️ Guarded on `window` rather than assumed, because this module is imported
 *  by server-rendered code too — a Next.js render on the server has no storage
 *  and must read every token as absent rather than throwing. */
const browserStore: TokenStore = {
  get(key) {
    if (typeof window === "undefined") return null;
    try {
      return window.localStorage.getItem(key);
    } catch {
      // ⚠️ A private window and some locked-down installs answer the API and
      // throw on use. A till that cannot remember a token still has to load —
      // it asks for the PIN again, which is the correct outcome.
      return null;
    }
  },
  set(key, value) {
    if (typeof window === "undefined") return;
    try {
      window.localStorage.setItem(key, value);
    } catch {
      // Sign-in still works for this session; it is forgotten on reload.
    }
  },
  remove(key) {
    if (typeof window === "undefined") return;
    try {
      window.localStorage.removeItem(key);
    } catch {
      // Nothing stored, nothing to remove.
    }
  },
};

/** The other lifetime, and it is a different question rather than a smaller
 *  one.
 *
 *  ⚠️ **A till session belongs to this sitting at this screen.** Closing the app
 *  must lock it, and a token that survived a restart would hand the next person
 *  the last one's name — which is the whole point of the PIN. On the web that
 *  is `sessionStorage`; on a phone it is memory, because a killed app *is* the
 *  end of the session. The platforms differ in mechanism and agree exactly on
 *  meaning, which is why this is a seam and not a special case. */
const browserSession: TokenStore = {
  get(key) {
    if (typeof window === "undefined") return null;
    try {
      return window.sessionStorage.getItem(key);
    } catch {
      return null;
    }
  },
  set(key, value) {
    if (typeof window === "undefined") return;
    try {
      window.sessionStorage.setItem(key, value);
    } catch {
      // Unlocked for this render; the PIN is asked again on reload.
    }
  },
  remove(key) {
    if (typeof window === "undefined") return;
    try {
      window.sessionStorage.removeItem(key);
    } catch {
      // Nothing stored, nothing to remove.
    }
  },
};

let store: TokenStore = browserStore;
let session: TokenStore = browserSession;

/** Hand the rules a different place to keep tokens.
 *
 *  ⚠️ Called **before the first request**, not lazily: a screen that fetched
 *  while the store was still the browser's would read every token as absent and
 *  send somebody to a login they had already passed. */
export function setTokenStore(next: TokenStore): void {
  store = next;
}

export function readToken(key: string): string | null {
  return store.get(key);
}

export function writeToken(key: string, value: string): void {
  store.set(key, value);
}

export function dropToken(key: string): void {
  store.remove(key);
}

/** Swap the session store. ⚠️ On a phone this is memory on purpose: it is not a
 *  weaker version of the store above, it is the correct lifetime. */
export function setSessionStore(next: TokenStore): void {
  session = next;
}

export function readSession(key: string): string | null {
  return session.get(key);
}

export function writeSession(key: string, value: string): void {
  session.set(key, value);
}

export function dropSession(key: string): void {
  session.remove(key);
}

// ---- Where the server is ----

let apiBase = "";
let uploadsBase = "";

/** Point the client at a server.
 *
 *  ⚠️ **A phone has no origin to be relative to.** The web app talks to its own
 *  host and lets the edge route `/api` — which is why the default is a path and
 *  not a URL. An app installed from a store has to be told the whole address,
 *  and for Keel it is per restaurant: the same binary serves every customer, so
 *  this is set after sign-in from what the account says, never baked into the
 *  build. `NEXT_PUBLIC_*` is the opposite of what a shared binary needs — it is
 *  sealed at build time (see CLAUDE.md on `rewrites()`).
 *
 *  Empty strings leave the web app's own defaults alone. */
export function setApiBase(api: string, uploads?: string): void {
  apiBase = api;
  if (uploads !== undefined) uploadsBase = uploads;
}

export function apiOverride(): string {
  return apiBase;
}

export function uploadsOverride(): string {
  return uploadsBase;
}
