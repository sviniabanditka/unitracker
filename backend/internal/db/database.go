package db

import (
	"database/sql"
	"errors"
	"sync"
)

// Database wraps a *sql.DB and allows atomic replacement of the underlying
// pool — used by Phase 6 restore to swap the SQLite file without restart.
type Database struct {
	mu   sync.RWMutex
	db   *sql.DB
	path string
}

func NewDatabase(db *sql.DB, path string) *Database {
	return &Database{db: db, path: path}
}

// Conn returns the current *sql.DB. Callers must not retain it across a Replace.
func (d *Database) Conn() *sql.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

func (d *Database) Path() string { return d.path }

// Replace swaps the underlying *sql.DB and returns the previous one (caller closes).
func (d *Database) Replace(newDB *sql.DB) *sql.DB {
	d.mu.Lock()
	defer d.mu.Unlock()
	old := d.db
	d.db = newDB
	return old
}

// Close shuts down the current pool and clears the wrapper.
func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return nil
	}
	err := d.db.Close()
	d.db = nil
	return err
}

// ErrClosed is returned when callers use a wrapper after Close.
var ErrClosed = errors.New("database closed")
