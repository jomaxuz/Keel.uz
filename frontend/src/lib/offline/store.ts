// The till's own disk, for the minutes the server is not there.
//
// ⚠️ **Two engines, and they are not equally durable.** Inside the Windows
// application this is SQLite with WAL and `synchronous=FULL`, in a file beside
// the pairing; in a browser it is IndexedDB. The difference is the one that
// matters in a restaurant: IndexedDB survives a reload, a crashed tab and a
// closed lid, and does **not** promise to survive the power going out
// mid-write. Monoblocks run on mains, not batteries — the lights go out and
// come back with the generator, with no warning and nothing flushed — and what
// is being written at that moment is an open table with food on it.
//
// ⚠️ **The engine is chosen here and nowhere else.** Everything above this file
// — what a local check is, the ids the sync is idempotent on, the service
// charge — is the same code in both builds. A screen that asked which engine it
// was talking to would be two tills, and the browser one is the one nobody
// tests on a monoblock.
//
// ⚠️ **No dependency.** IndexedDB's own API wrapped in twenty lines, rather
// than a library: this runs on a monoblock with four gigabytes of RAM, and the
// one thing it must be is small enough that nothing else on the screen slows
// down.

import { bridge } from "@/lib/tillBridge";

const DB_NAME = "keel-till";
// ⚠️ Bumped when a store is added: IndexedDB only creates object stores during
// an upgrade, so a new name on the old version is a store that does not exist
// and every write to it fails — silently, through the `catch` below.
const DB_VERSION = 2;
/** Sales whose payment the server has not confirmed. */
export const PENDING = "pendingSales";
/** Checks opened while the server was unreachable. */
export const LOCAL_CHECKS = "localChecks";
/** Facts about this device rather than about a sale — the clock offset and the
 *  last moment anything was written (see `clock.ts`). ⚠️ On the same disk as
 *  the sales on purpose: it decides whether an evening may be sold at all. */
export const META = "meta";

/** The id every record in both stores is kept under.
 *
 *  ⚠️ The same field on purpose: it is the id the till mints before anybody has
 *  seen the sale, and the one the server's idempotency is built on, so a resend
 *  is the same dinner on both sides of the wire. */
const KEY = "clientId";

function keyOf(value: unknown): string {
  const k = (value as Record<string, unknown> | null)?.[KEY];
  return typeof k === "string" ? k : "";
}

// ---- SQLite, inside the Windows application ----

/** Whether the machine's own database answered. Asked once and remembered:
 *  the answer cannot change while the process is running, and the screens ask
 *  on every save. */
let native: boolean | null = null;

async function nativeStore() {
  const b = bridge();
  if (!b?.StorePut) return null;
  if (native === null) {
    try {
      native = await b.StoreReady();
    } catch {
      native = false;
    }
    if (!native) {
      // ⚠️ Said out loud rather than fallen through silently. The till keeps
      // working on the browser's storage, on a weaker promise than the one it
      // was installed with, and the person who will be asked why an evening is
      // missing deserves the line in the log.
      console.warn("local database unavailable; falling back to IndexedDB");
    }
  }
  return native ? b : null;
}

// ---- IndexedDB, in a browser ----

function open(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(LOCAL_CHECKS)) {
        // Keyed by the same id the sync is idempotent on.
        db.createObjectStore(LOCAL_CHECKS, { keyPath: KEY });
      }
      if (!db.objectStoreNames.contains(PENDING)) {
        // Keyed by the id the till minted for the sale — the same id the
        // server's own idempotency is built on, so a retry is a retry on both
        // sides of the wire.
        db.createObjectStore(PENDING, { keyPath: KEY });
      }
      if (!db.objectStoreNames.contains(META)) {
        // One row, under a fixed id — the same keyPath so storage needs no
        // second rule about where a key comes from.
        db.createObjectStore(META, { keyPath: KEY });
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

/** Whether this build can store anything at all.
 *
 *  ⚠️ Private windows and some locked-down installs answer "yes" to the API and
 *  fail on the first write. Everything below therefore reports failure rather
 *  than throwing, and the caller decides — a till that cannot save locally must
 *  say so, not pretend. */
export function available(): boolean {
  return typeof indexedDB !== "undefined" || bridge()?.StorePut !== undefined;
}

export async function put<T>(store: string, value: T): Promise<boolean> {
  const key = keyOf(value);
  const b = await nativeStore();
  if (b) {
    if (!key) return false;
    try {
      await b.StorePut(store, key, JSON.stringify(value));
      return true;
    } catch {
      return false;
    }
  }
  if (typeof indexedDB === "undefined") return false;
  try {
    const db = await open();
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(store, "readwrite");
      tx.objectStore(store).put(value);
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
      tx.onabort = () => reject(tx.error);
    });
    db.close();
    return true;
  } catch {
    return false;
  }
}

export async function all<T>(store: string): Promise<T[]> {
  const b = await nativeStore();
  if (b) {
    try {
      const rows = await b.StoreAll(store);
      // ⚠️ A row that will not parse is dropped rather than thrown on. One
      // unreadable record must not take the evening's other checks off the
      // screen with it.
      return rows.flatMap((r) => {
        try {
          return [JSON.parse(r) as T];
        } catch {
          console.warn("unreadable local record dropped");
          return [];
        }
      });
    } catch {
      return [];
    }
  }
  if (typeof indexedDB === "undefined") return [];
  try {
    const db = await open();
    const rows = await new Promise<T[]>((resolve, reject) => {
      const tx = db.transaction(store, "readonly");
      const req = tx.objectStore(store).getAll();
      req.onsuccess = () => resolve(req.result as T[]);
      req.onerror = () => reject(req.error);
    });
    db.close();
    return rows;
  } catch {
    return [];
  }
}

/** Empty every store. Used between tests, and by nothing else.
 *
 *  ⚠️ **Cleared, not deleted.** Deleting the database blocks while any
 *  connection is still open — and a blocked delete completes *later*, in the
 *  middle of whatever is running by then. That is a test wiping the next test's
 *  data, and it reads as a bug in the till. The same reasoning holds on the
 *  SQLite side, where the file is opened once for the life of the process. */
export async function clearAll(): Promise<void> {
  const b = await nativeStore();
  if (b) {
    try {
      await b.StoreClear();
    } catch {
      // Nothing to clear, or no database at all.
    }
    return;
  }
  if (typeof indexedDB === "undefined") return;
  try {
    const db = await open();
    const names = Array.from(db.objectStoreNames);
    if (names.length > 0) {
      await new Promise<void>((resolve, reject) => {
        const tx = db.transaction(names, "readwrite");
        for (const name of names) tx.objectStore(name).clear();
        tx.oncomplete = () => resolve();
        tx.onerror = () => reject(tx.error);
        tx.onabort = () => reject(tx.error);
      });
    }
    db.close();
  } catch {
    // Nothing to clear, or no store at all.
  }
}

export async function remove(store: string, key: string): Promise<void> {
  const b = await nativeStore();
  if (b) {
    try {
      await b.StoreRemove(store, key);
    } catch {
      // See below: a failed delete leaves the sale in the queue.
    }
    return;
  }
  if (typeof indexedDB === "undefined") return;
  try {
    const db = await open();
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(store, "readwrite");
      tx.objectStore(store).delete(key);
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
    });
    db.close();
  } catch {
    // A delete that failed leaves the sale in the queue, which is the safe
    // direction: it is sent again and the server recognises it.
  }
}
