package entries

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	appdb "github.com/bublya/baby-tracker/backend/internal/db"
)

var (
	ErrNotFound     = errors.New("entry not found")
	ErrInvalidInput = errors.New("invalid input")
)

type Entry struct {
	ID         int64           `json:"id"`
	TrackerID  int64           `json:"tracker_id"`
	ProfileID  *int64          `json:"profile_id"`
	Data       json.RawMessage `json:"data"`
	OccurredAt string          `json:"occurred_at"`
	CreatedBy  *int64          `json:"created_by"`
	IsDeleted  bool            `json:"is_deleted"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// dbtx is implemented by both *sql.DB and *sql.Tx, lets the store run inside a transaction.
type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Store struct{ db *appdb.Database }

func NewStore(db *appdb.Database) *Store { return &Store{db: db} }

func (s *Store) DB() *sql.DB { return s.db.Conn() }

type ListFilter struct {
	TrackerID      int64
	From           *time.Time
	To             *time.Time
	Limit          int
	Cursor         string
	IncludeDeleted bool
}

func (s *Store) List(ctx context.Context, f ListFilter) ([]*Entry, string, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	where := []string{"tracker_id=?"}
	args := []any{f.TrackerID}
	if !f.IncludeDeleted {
		where = append(where, "is_deleted=0")
	}
	if f.From != nil {
		where = append(where, "occurred_at>=?")
		args = append(args, f.From.UTC().Format(time.RFC3339))
	}
	if f.To != nil {
		where = append(where, "occurred_at<=?")
		args = append(args, f.To.UTC().Format(time.RFC3339))
	}
	if f.Cursor != "" {
		ts, id, err := decodeCursor(f.Cursor)
		if err != nil {
			return nil, "", err
		}
		where = append(where, "(occurred_at<? OR (occurred_at=? AND id<?))")
		args = append(args, ts, ts, id)
	}
	q := `SELECT id, tracker_id, profile_id, data_json, occurred_at, created_by, is_deleted, created_at, updated_at
	        FROM entries WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY occurred_at DESC, id DESC LIMIT ?`
	args = append(args, f.Limit+1)
	rows, err := s.db.Conn().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	out := []*Entry{}
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, "", err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	var nextCursor string
	if len(out) > f.Limit {
		last := out[f.Limit-1]
		nextCursor = encodeCursor(last.OccurredAt, last.ID)
		out = out[:f.Limit]
	}
	return out, nextCursor, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Entry, error) {
	return getByIDTx(ctx, s.db.Conn(), id)
}

// GetProfileIDForEntry returns the profile_id of the tracker the entry belongs to.
// Used by access middleware on entry-scoped routes.
func (s *Store) GetProfileIDForEntry(ctx context.Context, entryID int64) (int64, error) {
	var pid int64
	err := s.db.Conn().QueryRowContext(ctx,
		`SELECT t.profile_id FROM entries e JOIN trackers t ON t.id = e.tracker_id WHERE e.id=?`,
		entryID).Scan(&pid)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return pid, err
}

func getByIDTx(ctx context.Context, x dbtx, id int64) (*Entry, error) {
	row := x.QueryRowContext(ctx,
		`SELECT id, tracker_id, profile_id, data_json, occurred_at, created_by, is_deleted, created_at, updated_at
		   FROM entries WHERE id=?`, id)
	e, err := scanEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return e, err
}

type CreateInput struct {
	TrackerID  int64
	ProfileID  *int64
	DataJSON   []byte
	OccurredAt time.Time
	CreatedBy  *int64
}

func createTx(ctx context.Context, x dbtx, in CreateInput) (*Entry, error) {
	res, err := x.ExecContext(ctx,
		`INSERT INTO entries (tracker_id, profile_id, data_json, occurred_at, created_by)
		    VALUES (?, ?, ?, ?, ?)`,
		in.TrackerID, nullableInt(in.ProfileID), string(in.DataJSON),
		in.OccurredAt.UTC().Format(time.RFC3339), nullableInt(in.CreatedBy))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return getByIDTx(ctx, x, id)
}

type UpdateInput struct {
	DataJSON   []byte
	OccurredAt *time.Time
}

func updateTx(ctx context.Context, x dbtx, id int64, in UpdateInput) (*Entry, error) {
	sets := []string{}
	args := []any{}
	if len(in.DataJSON) > 0 {
		sets = append(sets, "data_json=?")
		args = append(args, string(in.DataJSON))
	}
	if in.OccurredAt != nil {
		sets = append(sets, "occurred_at=?")
		args = append(args, in.OccurredAt.UTC().Format(time.RFC3339))
	}
	if len(sets) == 0 {
		return getByIDTx(ctx, x, id)
	}
	sets = append(sets, "updated_at=CURRENT_TIMESTAMP")
	q := "UPDATE entries SET " + strings.Join(sets, ", ") + " WHERE id=?"
	args = append(args, id)
	res, err := x.ExecContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return getByIDTx(ctx, x, id)
}

func softDeleteTx(ctx context.Context, x dbtx, id int64) (*Entry, bool, error) {
	res, err := x.ExecContext(ctx,
		`UPDATE entries SET is_deleted=1, updated_at=CURRENT_TIMESTAMP WHERE id=? AND is_deleted=0`, id)
	if err != nil {
		return nil, false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		e, gerr := getByIDTx(ctx, x, id)
		if gerr != nil {
			return nil, false, gerr
		}
		return e, false, nil
	}
	e, err := getByIDTx(ctx, x, id)
	return e, true, err
}

func restoreDeletedTx(ctx context.Context, x dbtx, id int64) (*Entry, bool, error) {
	res, err := x.ExecContext(ctx,
		`UPDATE entries SET is_deleted=0, updated_at=CURRENT_TIMESTAMP WHERE id=? AND is_deleted=1`, id)
	if err != nil {
		return nil, false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		e, gerr := getByIDTx(ctx, x, id)
		if gerr != nil {
			return nil, false, gerr
		}
		return e, false, nil
	}
	e, err := getByIDTx(ctx, x, id)
	return e, true, err
}

func applyRevisionTx(ctx context.Context, x dbtx, id int64, dataJSON []byte, occurredAt string, profileID *int64, isDeleted bool) (*Entry, error) {
	res, err := x.ExecContext(ctx,
		`UPDATE entries
		    SET data_json=?, occurred_at=?, profile_id=?, is_deleted=?, updated_at=CURRENT_TIMESTAMP
		  WHERE id=?`,
		string(dataJSON), occurredAt, nullableInt(profileID), boolToInt(isDeleted), id)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return getByIDTx(ctx, x, id)
}

func (s *Store) Create(ctx context.Context, in CreateInput) (*Entry, error) {
	return createTx(ctx, s.db.Conn(), in)
}

func (s *Store) Update(ctx context.Context, id int64, in UpdateInput) (*Entry, error) {
	return updateTx(ctx, s.db.Conn(), id, in)
}

func (s *Store) SoftDelete(ctx context.Context, id int64) error {
	_, _, err := softDeleteTx(ctx, s.db.Conn(), id)
	return err
}

func (s *Store) CountByTracker(ctx context.Context, trackerID int64) (int, error) {
	var n int
	err := s.db.Conn().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM entries WHERE tracker_id=? AND is_deleted=0`, trackerID).Scan(&n)
	return n, err
}

type rowScanner interface{ Scan(dest ...any) error }

func scanEntry(r rowScanner) (*Entry, error) {
	var (
		e         Entry
		profileID sql.NullInt64
		createdBy sql.NullInt64
		dataText  string
		deleted   int64
	)
	if err := r.Scan(&e.ID, &e.TrackerID, &profileID, &dataText, &e.OccurredAt, &createdBy, &deleted, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return nil, err
	}
	if profileID.Valid {
		v := profileID.Int64
		e.ProfileID = &v
	}
	if createdBy.Valid {
		v := createdBy.Int64
		e.CreatedBy = &v
	}
	e.Data = json.RawMessage(dataText)
	e.IsDeleted = deleted != 0
	e.CreatedAt = e.CreatedAt.UTC()
	e.UpdatedAt = e.UpdatedAt.UTC()
	return &e, nil
}

func nullableInt(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func encodeCursor(ts string, id int64) string {
	raw := fmt.Sprintf("%s|%d", ts, id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(c string) (string, int64, error) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return "", 0, errors.New("invalid cursor")
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return "", 0, errors.New("invalid cursor")
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, errors.New("invalid cursor")
	}
	return parts[0], id, nil
}
