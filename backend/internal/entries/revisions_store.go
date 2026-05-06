package entries

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	appdb "github.com/bublya/baby-tracker/backend/internal/db"
)

const (
	ChangeCreate  = "create"
	ChangeUpdate  = "update"
	ChangeDelete  = "delete"
	ChangeRestore = "restore"
)

var ErrRevisionNotFound = errors.New("revision not found")

type Revision struct {
	ID         int64           `json:"id"`
	EntryID    int64           `json:"entry_id"`
	Data       json.RawMessage `json:"data"`
	OccurredAt string          `json:"occurred_at"`
	ProfileID  *int64          `json:"profile_id"`
	IsDeleted  bool            `json:"is_deleted"`
	ChangeType string          `json:"change_type"`
	ChangedBy  *int64          `json:"changed_by"`
	ChangedAt  time.Time       `json:"changed_at"`
}

type RevisionStore struct{ db *appdb.Database }

func NewRevisionStore(db *appdb.Database) *RevisionStore { return &RevisionStore{db: db} }

type AppendInput struct {
	EntryID    int64
	DataJSON   []byte
	OccurredAt string
	ProfileID  *int64
	IsDeleted  bool
	ChangeType string
	ChangedBy  *int64
}

// appendRevisionTx writes a revision row using the supplied executor (typically a *sql.Tx).
func appendRevisionTx(ctx context.Context, x dbtx, in AppendInput) error {
	_, err := x.ExecContext(ctx,
		`INSERT INTO entry_revisions
		   (entry_id, data_json, occurred_at, profile_id, is_deleted, change_type, changed_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		in.EntryID,
		string(in.DataJSON),
		in.OccurredAt,
		nullableInt(in.ProfileID),
		boolToInt(in.IsDeleted),
		in.ChangeType,
		nullableInt(in.ChangedBy),
	)
	return err
}

func (s *RevisionStore) ListByEntry(ctx context.Context, entryID int64) ([]*Revision, error) {
	rows, err := s.db.Conn().QueryContext(ctx,
		`SELECT id, entry_id, data_json, occurred_at, profile_id, is_deleted, change_type, changed_by, changed_at
		   FROM entry_revisions WHERE entry_id=? ORDER BY changed_at DESC, id DESC`, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Revision{}
	for rows.Next() {
		rev, err := scanRevision(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rev)
	}
	return out, rows.Err()
}

func (s *RevisionStore) GetByID(ctx context.Context, id int64) (*Revision, error) {
	row := s.db.Conn().QueryRowContext(ctx,
		`SELECT id, entry_id, data_json, occurred_at, profile_id, is_deleted, change_type, changed_by, changed_at
		   FROM entry_revisions WHERE id=?`, id)
	rev, err := scanRevision(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRevisionNotFound
	}
	return rev, err
}

func scanRevision(r rowScanner) (*Revision, error) {
	var (
		rev       Revision
		profileID sql.NullInt64
		changedBy sql.NullInt64
		dataText  string
		deleted   int64
	)
	if err := r.Scan(
		&rev.ID, &rev.EntryID, &dataText, &rev.OccurredAt,
		&profileID, &deleted, &rev.ChangeType, &changedBy, &rev.ChangedAt,
	); err != nil {
		return nil, err
	}
	if profileID.Valid {
		v := profileID.Int64
		rev.ProfileID = &v
	}
	if changedBy.Valid {
		v := changedBy.Int64
		rev.ChangedBy = &v
	}
	rev.Data = json.RawMessage(dataText)
	rev.IsDeleted = deleted != 0
	rev.ChangedAt = rev.ChangedAt.UTC()
	return &rev, nil
}
