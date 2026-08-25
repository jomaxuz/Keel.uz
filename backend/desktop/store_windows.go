//go:build windows

package main

// The till's disk, as the screen calls it.
//
// ⚠️ **This is what a monoblock needed and a browser could not give.** The
// offline queue runs on IndexedDB in a browser, which survives a reload, a
// crashed tab and a closed lid — and does not promise to survive the power
// going out mid-write. Restaurants here lose power; the machine comes back with
// the generator, and what was being written is an open table with food on it.
// The store behind these four calls is SQLite with WAL and `synchronous=FULL`
// (internal/tillstore), which is the plan's answer in docs/pos-reja.md §6.
//
// ⚠️ **Four calls and no more, because the shape is already right.** The screen
// keeps every rule about what a local check *is* — the ids, the service charge,
// what may be sold offline — and asks storage only to keep bytes under a key.
// Moving any of that down here would be a second implementation of rules the
// browser build still has to run.

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"restaurant-backend/internal/tillstore"
)

// storeOnce guards the one open file.
//
// ⚠️ **Opened lazily, not at startup.** A machine that has never been offline
// never needs it, and a failure to open must not be a till that will not start:
// the screen falls back to IndexedDB, which is where it was before this
// existed.
var (
	storeOnce sync.Once
	tillDB    *tillstore.Store
)

func (a *App) store() *tillstore.Store {
	storeOnce.Do(func() {
		// Beside the pairing and the log, in %PROGRAMDATA%\Keel: Program Files
		// is not writable by the account a cashier is signed in as, and a till
		// that cannot write its own sales is the failure this file prevents.
		db, err := tillstore.Open(filepath.Join(configDir(), "till.db"))
		if err != nil {
			log.Printf("lokal baza ochilmadi: %v", err)
			return
		}
		tillDB = db
		log.Print("lokal baza ochildi (WAL, synchronous=FULL)")
	})
	return tillDB
}

// StoreReady says whether the screen can rely on this rather than on the
// browser's storage.
//
// ⚠️ **Asked, not assumed.** The screen has two places it can save and they are
// not equally durable; a till that quietly fell back would keep selling and
// tell nobody that the guarantee changed.
func (a *App) StoreReady() bool { return a.store() != nil }

// StorePut writes one record. `value` is the JSON the screen already builds.
func (a *App) StorePut(store, key, value string) error {
	s := a.store()
	if s == nil {
		return errNoStore
	}
	return s.Put(store, key, value)
}

// StoreAll returns every record in one store.
func (a *App) StoreAll(store string) ([]string, error) {
	s := a.store()
	if s == nil {
		return []string{}, errNoStore
	}
	return s.All(store)
}

// StoreRemove deletes one record.
func (a *App) StoreRemove(store, key string) error {
	s := a.store()
	if s == nil {
		return errNoStore
	}
	return s.Remove(store, key)
}

// StoreClear empties every store. Used when a till is signed out, and by the
// tests.
//
// ⚠️ Not called by the sign-out path: unsent sales are money that has not
// reached the server, and tidying a screen is not a reason to delete them. That
// rule is in the exit handler and repeated here because this is the function
// somebody would reach for.
func (a *App) StoreClear() error {
	s := a.store()
	if s == nil {
		return errNoStore
	}
	return s.Clear()
}

// closeStore is called on the way out, so the WAL is checkpointed by a clean
// shutdown rather than left for the next start to recover.
func closeStore() {
	if tillDB != nil {
		tillDB.Close()
	}
}

// SetMode chooses which screen this machine opens: "kassa" or "zal".
//
// ⚠️ **Saved and then the window is reloaded by the screen**, rather than
// swapped in place. The two screens mount different providers and different
// polls, and a live swap would leave whichever one was running holding a table
// it no longer draws — during service, on the machine somebody just changed.
// Reloading is what the setup screen already does after pairing.
func (a *App) SetMode(mode string) error {
	a.cfg.Mode = mode
	if err := saveSettings(a.cfg); err != nil {
		return fmt.Errorf("saqlanmadi: %w", err)
	}
	log.Printf("ekran turi: %s", a.cfg.mode())
	return nil
}
