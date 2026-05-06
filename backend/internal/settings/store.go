package settings

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"sync"
	"time"

	appdb "github.com/bublya/baby-tracker/backend/internal/db"
)

const (
	KeyBackupIntervalHours  = "backup_interval_hours"
	KeyBackupRetentionCount = "backup_retention_count"
	KeyAppName              = "app_name"
	KeyDefaultLocale        = "default_locale"
)

var ErrNotFound = errors.New("setting not found")

type Setting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Subscriber func(value string)

type Store struct {
	db *appdb.Database

	mu   sync.RWMutex
	subs map[string][]Subscriber
}

func NewStore(db *appdb.Database) *Store {
	return &Store{db: db, subs: map[string][]Subscriber{}}
}

// Subscribe registers fn to be invoked synchronously after a successful Set on key.
// Subscribers run after the DB write commits; if a subscriber panics it is logged and skipped.
func (s *Store) Subscribe(key string, fn Subscriber) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs[key] = append(s.subs[key], fn)
}

func (s *Store) notify(key, value string) {
	s.mu.RLock()
	subs := append([]Subscriber(nil), s.subs[key]...)
	s.mu.RUnlock()
	for _, fn := range subs {
		fn(value)
	}
}

func (s *Store) Get(ctx context.Context, key string) (string, error) {
	var v string
	err := s.db.Conn().QueryRowContext(ctx,
		`SELECT value FROM settings WHERE key=?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return v, err
}

func (s *Store) Set(ctx context.Context, key, value string) error {
	if _, err := s.db.Conn().ExecContext(ctx,
		`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		   ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=CURRENT_TIMESTAMP`,
		key, value); err != nil {
		return err
	}
	s.notify(key, value)
	return nil
}

// SetMany applies several key/value updates in one transaction. After commit,
// subscribers are notified for each changed key in input order.
func (s *Store) SetMany(ctx context.Context, kv map[string]string) error {
	if len(kv) == 0 {
		return nil
	}
	tx, err := s.db.Conn().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		   ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=CURRENT_TIMESTAMP`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for k, v := range kv {
		if _, err := stmt.ExecContext(ctx, k, v); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for k, v := range kv {
		s.notify(k, v)
	}
	return nil
}

func (s *Store) All(ctx context.Context) ([]*Setting, error) {
	rows, err := s.db.Conn().QueryContext(ctx,
		`SELECT key, value, updated_at FROM settings ORDER BY key ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Setting{}
	for rows.Next() {
		var x Setting
		if err := rows.Scan(&x.Key, &x.Value, &x.UpdatedAt); err != nil {
			return nil, err
		}
		x.UpdatedAt = x.UpdatedAt.UTC()
		out = append(out, &x)
	}
	return out, rows.Err()
}

// IntOr reads a setting and parses it as int; if missing or invalid, returns fallback.
func (s *Store) IntOr(ctx context.Context, key string, fallback int) int {
	v, err := s.Get(ctx, key)
	if err != nil {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// FloatOr reads a setting as float64 (used for fractional intervals like 0.05h).
func (s *Store) FloatOr(ctx context.Context, key string, fallback float64) float64 {
	v, err := s.Get(ctx, key)
	if err != nil {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

// StringOr reads a setting as a non-empty string; falls back when missing or empty.
func (s *Store) StringOr(ctx context.Context, key, fallback string) string {
	v, err := s.Get(ctx, key)
	if err != nil || v == "" {
		return fallback
	}
	return v
}

// Defaults applied when the row is missing.
const (
	DefaultBackupIntervalHours  = 6.0
	DefaultBackupRetentionCount = 20
	DefaultAppName              = "Tracker"
	DefaultLocaleCode           = "en"
)

func (s *Store) BackupIntervalHours(ctx context.Context) float64 {
	return s.FloatOr(ctx, KeyBackupIntervalHours, DefaultBackupIntervalHours)
}
func (s *Store) BackupRetentionCount(ctx context.Context) int {
	return s.IntOr(ctx, KeyBackupRetentionCount, DefaultBackupRetentionCount)
}
func (s *Store) AppName(ctx context.Context) string {
	return s.StringOr(ctx, KeyAppName, DefaultAppName)
}
func (s *Store) DefaultLocale(ctx context.Context) string {
	return s.StringOr(ctx, KeyDefaultLocale, DefaultLocaleCode)
}
