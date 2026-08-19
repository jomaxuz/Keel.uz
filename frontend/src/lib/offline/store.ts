// The till's own disk, for the minutes the server is not there.
//
// ⚠️ **This is a browser, and a browser is not a cash register.** IndexedDB
// survives a reload, a crashed tab and a closed lid; it does **not** promise to
// survive the power going out mid-write, which is why the plan puts the real
// offline till in a Windows app with SQLite (docs/pos-reja.md §6). What this
// covers is the failure a restaurant actually meets several times a week: the
// wifi drops, the provider has a bad minute, we deploy. The kitchen keeps
// cooking and the guest keeps paying, and nothing about that should depend on
// our server being reachable at that second.
//
// ⚠️ **No dependency.** IndexedDB's own API wrapped in twenty lines, rather
// than a library: this runs on a monoblock with four gigabytes of RAM, and the
// one thing it must be is small enough that nothing else on the screen slows
// down.

const DB_NAME = "keel-till";
const DB_VERSION = 1;
/** Sales whose payment the server has not confirmed. */
export const PENDING = "pendingSales";
/** Checks opened while the server was unreachable. */
export const LOCAL_CHECKS = "localChecks";

function open(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(LOCAL_CHECKS)) {
        // Keyed by the same id the sync is idempotent on.
        db.createObjectStore(LOCAL_CHECKS, { keyPath: "clientId" });
      }
      if (!db.objectStoreNames.contains(PENDING)) {
        // Keyed by the id the till minted for the sale — the same id the
        // server's own idempotency is built on, so a retry is a retry on both
        // sides of the wire.
        db.createObjectStore(PENDING, { keyPath: "clientId" });
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

/** Whether this browser can store anything at all.
 *
 *  ⚠️ Private windows and some locked-down installs answer "yes" to the API and
 *  fail on the first write. Everything below therefore reports failure rather
 *  than throwing, and the caller decides — a till that cannot save locally must
 *  say so, not pretend. */
export function available(): boolean {
  return typeof indexedDB !== "undefined";
}

export async function put<T>(store: string, value: T): Promise<boolean> {
  if (!available()) return false;
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
  if (!available()) return [];
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
 *  data, and it reads as a bug in the till. */
export async function clearAll(): Promise<void> {
  if (!available()) return;
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
  if (!available()) return;
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
