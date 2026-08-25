// Package tillstore is the till's own disk: the checks and the sales that have
// not reached the server yet.
//
// ⚠️ **A browser is not a cash register, and that is why this exists.**
// IndexedDB survives a reload, a crashed tab and a closed lid; it does not
// promise to survive the power going out mid-write. Monoblocks run on mains,
// not batteries — the lights go out and come back with the generator, with no
// warning and no chance to flush anything — and what is being written at that
// moment is an open table with food on it.
//
// So the rule from docs/pos-reja.md §6 is the whole design: **every change is
// on disk before it is on the screen.** A screen that shows a dish the disk
// does not have is a screen that lies to the person holding the till.
//
// ⚠️ **WAL plus `synchronous=FULL`, and the second half is the point.** WAL on
// its own defaults to `NORMAL`, which loses the last transactions on a power
// cut — the database stays intact and the writes are gone, which is the worst
// available outcome for a machine taking money. `FULL` fsyncs on every commit:
// one to five milliseconds on an SSD, invisible at the speed a cashier types.
//
// ⚠️ **No cgo.** modernc.org/sqlite is SQLite translated to Go, so the till
// still cross-compiles from the Linux machine this is written on — the same
// reason the spooler is reached through a lazy DLL rather than a C binding. A
// cgo dependency here would mean the Windows build could only be produced on
// Windows, which is the kind of constraint nobody notices until the release
// nobody can cut.
package tillstore

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

// Store is one file on the machine's disk.
type Store struct {
	// ⚠️ One connection, serialised. SQLite would take more, and the till has
	// exactly one writer — the screen in front of somebody. A pool here buys
	// nothing and turns "database is locked" into a class of bug that only
	// appears on a busy evening.
	mu sync.Mutex
	db *sql.DB
}

// Open prepares the file and the schema. A store that cannot be opened is
// returned as an error rather than a silent no-op: a till that cannot save
// locally has to say so, because the alternative is a cashier working an
// evening whose sales are going nowhere.
func Open(path string) (*Store, error) {
	// ⚠️ The pragmas go in the DSN, not in a later Exec. A pragma issued on a
	// pooled connection applies to whichever connection ran it, and the next
	// write can land on another one — so `synchronous=FULL` would be true of
	// some writes and not others, which is indistinguishable from working.
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// ⚠️ `WITHOUT ROWID` and a composite key, because every read is "everything in
// one store" and every write is by that store's own id. There is no third
// access pattern to keep an index for.
const schema = `
CREATE TABLE IF NOT EXISTS records (
  store TEXT NOT NULL,
  key   TEXT NOT NULL,
  value TEXT NOT NULL,
  PRIMARY KEY (store, key)
) WITHOUT ROWID;`

// ErrNoKey is what a record with no id is refused with.
//
// ⚠️ **Refused, not stored under a made-up key.** The id is minted by the till
// before anybody has seen the sale, and it is the whole of the idempotency on
// both sides of the wire: a record without one would sync as a second dinner
// every time it was retried.
var ErrNoKey = errors.New("tillstore: record has no key")

// Put writes one record, replacing whatever was there under that key.
func (s *Store) Put(store, key, value string) error {
	if key == "" {
		return ErrNoKey
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`INSERT INTO records (store, key, value) VALUES (?, ?, ?)
		 ON CONFLICT(store, key) DO UPDATE SET value = excluded.value`,
		store, key, value)
	return err
}

// All returns every record in one store, as the JSON the screen wrote.
//
// ⚠️ Ordered by key so a resend goes out in a stable order. Unordered, two
// reads of the same queue can disagree about which sale is first, and the
// symptom is a receipt list that reshuffles itself while somebody is reading
// it.
func (s *Store) All(store string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(
		`SELECT value FROM records WHERE store = ? ORDER BY key`, store)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// ⚠️ Built empty rather than left nil: this crosses into JavaScript, where
	// a null is a `.length` on the till's own screen. The rule has bitten this
	// codebase twice.
	out := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Remove deletes one record. Deleting what is not there is not an error: the
// caller is a queue draining itself, and a sale removed twice is a sale that
// arrived.
func (s *Store) Remove(store, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM records WHERE store = ? AND key = ?`, store, key)
	return err
}

// Clear empties every store.
//
// ⚠️ Emptied, not deleted: the file is opened once for the life of the process,
// and removing it underneath that handle leaves writes going to a file nothing
// will ever read again — which looks exactly like a till that is saving.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM records`)
	return err
}

// Close releases the file.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}
