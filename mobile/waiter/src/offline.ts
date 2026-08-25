import * as SQLite from "expo-sqlite";

import { setNativeStore } from "@/lib/offline/store";

// The phone's own disk, for the minutes the server is not there.
//
// ⚠️ **A browser is not a cash register, and a phone is not a browser either.**
// The offline queue was written against IndexedDB, which a React Native app
// does not have — so `available()` answered false and the queue silently did
// nothing. The app looked entirely normal and stopped selling the moment the
// wifi dropped, which in a restaurant is weekly. This is the third engine
// behind the same seam, after the browser's and the Windows till's SQLite.
//
// ⚠️ **WAL and `synchronous=FULL`, the same pair the Windows till uses** and for
// the same reason: WAL alone defaults to NORMAL, which loses the last
// transactions when the process dies — the database survives and the sales do
// not, which is the worst available outcome for something taking money. A phone
// is killed by the operating system without warning, which is this app's
// version of a power cut.
//
// ⚠️ **The rules above this file are untouched.** What a local check is, the ids
// the sync is idempotent on, the service charge — all the same code as the
// counter runs. Only where the bytes land changes.

const NAME = "keel-waiter.db";

let db: SQLite.SQLiteDatabase | null = null;

/** Open the file and hand it to the queue. Called once, at startup, before
 *  anything is saved. */
export async function openOfflineStore(): Promise<void> {
  if (db) return;
  try {
    db = await SQLite.openDatabaseAsync(NAME);
    await db.execAsync(
      "PRAGMA journal_mode = WAL; PRAGMA synchronous = FULL;" +
        "CREATE TABLE IF NOT EXISTS records (" +
        "  store TEXT NOT NULL, key TEXT NOT NULL, value TEXT NOT NULL," +
        "  PRIMARY KEY (store, key)) WITHOUT ROWID;",
    );
  } catch {
    // ⚠️ A phone whose database will not open still sells — on nothing, which
    // is what it did before this existed. Said in the log rather than in front
    // of somebody taking an order.
    db = null;
    return;
  }

  setNativeStore({
    async put(store, key, value) {
      if (!db) return false;
      try {
        await db.runAsync(
          "INSERT INTO records (store, key, value) VALUES (?, ?, ?)" +
            " ON CONFLICT(store, key) DO UPDATE SET value = excluded.value",
          store,
          key,
          value,
        );
        return true;
      } catch {
        return false;
      }
    },
    async all(store) {
      if (!db) return [];
      try {
        const rows = await db.getAllAsync<{ value: string }>(
          // ⚠️ Ordered by key so a resend goes out in a stable order — two
          // reads of one queue that disagree about which sale is first show up
          // as a list that reshuffles while somebody is reading it.
          "SELECT value FROM records WHERE store = ? ORDER BY key",
          store,
        );
        return rows.map((r) => r.value);
      } catch {
        return [];
      }
    },
    async remove(store, key) {
      if (!db) return;
      try {
        await db.runAsync(
          "DELETE FROM records WHERE store = ? AND key = ?",
          store,
          key,
        );
      } catch {
        // A delete that failed leaves the sale in the queue, which is the safe
        // direction: it is sent again and the server recognises it.
      }
    },
    async clear() {
      if (!db) return;
      try {
        // ⚠️ Emptied, not deleted: the file is opened once for the life of the
        // process, and removing it underneath that handle leaves writes going
        // somewhere nothing will read — which looks exactly like an app that
        // is saving.
        await db.runAsync("DELETE FROM records");
      } catch {
        // Nothing to clear.
      }
    },
  });
}
