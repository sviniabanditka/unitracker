package backup

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	appdb "github.com/bublya/baby-tracker/backend/internal/db"
)

const (
	TypeAuto       = "auto"
	TypeManual     = "manual"
	TypePreRestore = "pre-restore"
)

var ErrSnapshotNotFound = errors.New("snapshot not found")

type Snapshot struct {
	ID        int64     `json:"id"`
	Filename  string    `json:"filename"`
	SizeBytes int64     `json:"size_bytes"`
	Type      string    `json:"type"`
	Note      *string   `json:"note,omitempty"`
	CreatedBy *int64    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type snapshotStore struct {
	db *appdb.Database
}

func newSnapshotStore(db *appdb.Database) *snapshotStore {
	return &snapshotStore{db: db}
}

type ListFilter struct {
	Type  string
	Limit int
}

func (s *snapshotStore) List(ctx context.Context, f ListFilter) ([]*Snapshot, error) {
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 100
	}
	where := []string{"1=1"}
	args := []any{}
	if f.Type != "" {
		where = append(where, "type=?")
		args = append(args, f.Type)
	}
	q := `SELECT id, filename, size_bytes, type, note, created_by, created_at
	        FROM snapshots WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, f.Limit)
	rows, err := s.db.Conn().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Snapshot{}
	for rows.Next() {
		x, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *snapshotStore) Get(ctx context.Context, id int64) (*Snapshot, error) {
	row := s.db.Conn().QueryRowContext(ctx,
		`SELECT id, filename, size_bytes, type, note, created_by, created_at
		   FROM snapshots WHERE id=?`, id)
	x, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSnapshotNotFound
	}
	return x, err
}

type insertInput struct {
	Filename  string
	SizeBytes int64
	Type      string
	Note      *string
	CreatedBy *int64
}

func (s *snapshotStore) Insert(ctx context.Context, in insertInput) (*Snapshot, error) {
	res, err := s.db.Conn().ExecContext(ctx,
		`INSERT INTO snapshots (filename, size_bytes, type, note, created_by)
		   VALUES (?, ?, ?, ?, ?)`,
		in.Filename, in.SizeBytes, in.Type, nullableString(in.Note), nullableInt(in.CreatedBy))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// InsertReconciled writes a row with an explicit created_at — used by Reconcile
// so timestamps recovered from filenames preserve original ordering.
func (s *snapshotStore) InsertReconciled(ctx context.Context, in insertInput, createdAt time.Time) (*Snapshot, error) {
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	res, err := s.db.Conn().ExecContext(ctx,
		`INSERT INTO snapshots (filename, size_bytes, type, note, created_by, created_at)
		   VALUES (?, ?, ?, ?, ?, ?)`,
		in.Filename, in.SizeBytes, in.Type, nullableString(in.Note),
		nullableInt(in.CreatedBy), createdAt.UTC())
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *snapshotStore) Delete(ctx context.Context, id int64) error {
	res, err := s.db.Conn().ExecContext(ctx, `DELETE FROM snapshots WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrSnapshotNotFound
	}
	return nil
}

// OldestAuto returns the oldest auto snapshots beyond the keep count, oldest first.
func (s *snapshotStore) OldestAuto(ctx context.Context, keep int) ([]*Snapshot, error) {
	if keep < 0 {
		keep = 0
	}
	rows, err := s.db.Conn().QueryContext(ctx,
		`SELECT id, filename, size_bytes, type, note, created_by, created_at
		   FROM snapshots WHERE type=? ORDER BY created_at DESC, id DESC LIMIT -1 OFFSET ?`,
		TypeAuto, keep)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Snapshot{}
	for rows.Next() {
		x, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func scan(r rowScanner) (*Snapshot, error) {
	var (
		x         Snapshot
		note      sql.NullString
		createdBy sql.NullInt64
	)
	if err := r.Scan(&x.ID, &x.Filename, &x.SizeBytes, &x.Type, &note, &createdBy, &x.CreatedAt); err != nil {
		return nil, err
	}
	if note.Valid {
		v := note.String
		x.Note = &v
	}
	if createdBy.Valid {
		v := createdBy.Int64
		x.CreatedBy = &v
	}
	x.CreatedAt = x.CreatedAt.UTC()
	return &x, nil
}

func nullableString(p *string) any {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return v
}

func nullableInt(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}
