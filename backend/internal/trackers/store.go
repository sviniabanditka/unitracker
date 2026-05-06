package trackers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	appdb "github.com/bublya/baby-tracker/backend/internal/db"
)

var (
	ErrNotFound     = errors.New("tracker not found")
	ErrInvalidInput = errors.New("invalid input")
)

type Tracker struct {
	ID          int64           `json:"id"`
	ProfileID   int64           `json:"profile_id"`
	Name        string          `json:"name"`
	Icon        *string         `json:"icon,omitempty"`
	Color       *string         `json:"color,omitempty"`
	Description *string         `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema_json"`
	IsArchived  bool            `json:"is_archived"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// LibraryTracker is a tracker enriched with its owning profile name, used by the
// "clone from library" picker on the frontend.
type LibraryTracker struct {
	Tracker
	ProfileName string `json:"profile_name"`
}

type Store struct{ db *appdb.Database }

func NewStore(db *appdb.Database) *Store { return &Store{db: db} }

// ListByProfile returns trackers belonging to the given profile.
func (s *Store) ListByProfile(ctx context.Context, profileID int64, includeArchived bool) ([]*Tracker, error) {
	q := `SELECT id, profile_id, name, icon, color, description, schema_json, is_archived, created_at, updated_at
	        FROM trackers WHERE profile_id=?`
	args := []any{profileID}
	if !includeArchived {
		q += ` AND is_archived=0`
	}
	q += ` ORDER BY id ASC`
	rows, err := s.db.Conn().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Tracker{}
	for rows.Next() {
		t, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Library returns all non-archived trackers from profiles in the allowed list,
// each annotated with the owning profile's name. Empty allowed list returns [].
func (s *Store) Library(ctx context.Context, allowedProfileIDs []int64) ([]*LibraryTracker, error) {
	if len(allowedProfileIDs) == 0 {
		return []*LibraryTracker{}, nil
	}
	placeholders := make([]string, len(allowedProfileIDs))
	args := make([]any, len(allowedProfileIDs))
	for i, id := range allowedProfileIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	q := `SELECT t.id, t.profile_id, t.name, t.icon, t.color, t.description, t.schema_json,
	             t.is_archived, t.created_at, t.updated_at, p.name AS profile_name
	        FROM trackers t
	        JOIN profiles p ON p.id = t.profile_id
	       WHERE t.profile_id IN (` + strings.Join(placeholders, ",") + `) AND t.is_archived=0
	       ORDER BY t.id ASC`
	rows, err := s.db.Conn().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*LibraryTracker{}
	for rows.Next() {
		var (
			t          Tracker
			icon       sql.NullString
			color      sql.NullString
			desc       sql.NullString
			schemaText string
			archived   int64
			profName   string
		)
		if err := rows.Scan(&t.ID, &t.ProfileID, &t.Name, &icon, &color, &desc, &schemaText,
			&archived, &t.CreatedAt, &t.UpdatedAt, &profName); err != nil {
			return nil, err
		}
		if icon.Valid {
			t.Icon = &icon.String
		}
		if color.Valid {
			t.Color = &color.String
		}
		if desc.Valid {
			t.Description = &desc.String
		}
		t.Schema = json.RawMessage(schemaText)
		t.IsArchived = archived != 0
		t.CreatedAt = t.CreatedAt.UTC()
		t.UpdatedAt = t.UpdatedAt.UTC()
		out = append(out, &LibraryTracker{Tracker: t, ProfileName: profName})
	}
	return out, rows.Err()
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Tracker, error) {
	row := s.db.Conn().QueryRowContext(ctx,
		`SELECT id, profile_id, name, icon, color, description, schema_json, is_archived, created_at, updated_at
		   FROM trackers WHERE id=?`, id)
	t, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

// GetProfileID is a lightweight lookup used by access middlewares.
func (s *Store) GetProfileID(ctx context.Context, id int64) (int64, error) {
	var pid int64
	err := s.db.Conn().QueryRowContext(ctx,
		`SELECT profile_id FROM trackers WHERE id=?`, id).Scan(&pid)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return pid, err
}

type CreateInput struct {
	ProfileID   int64
	Name        string
	Icon        *string
	Color       *string
	Description *string
	SchemaJSON  []byte
}

func (s *Store) Create(ctx context.Context, in CreateInput) (*Tracker, error) {
	if in.ProfileID <= 0 {
		return nil, ErrInvalidInput
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	if len(in.SchemaJSON) == 0 {
		return nil, ErrInvalidInput
	}
	res, err := s.db.Conn().ExecContext(ctx,
		`INSERT INTO trackers (profile_id, name, icon, color, description, schema_json)
		    VALUES (?, ?, ?, ?, ?, ?)`,
		in.ProfileID, name, nullableString(in.Icon), nullableString(in.Color),
		nullableString(in.Description), string(in.SchemaJSON))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

type UpdateInput struct {
	Name             *string
	Icon             *string
	ClearIcon        bool
	Color            *string
	ClearColor       bool
	Description      *string
	ClearDescription bool
	SchemaJSON       []byte
	IsArchived       *bool
}

func (s *Store) Update(ctx context.Context, id int64, in UpdateInput) (*Tracker, error) {
	sets := []string{}
	args := []any{}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, ErrInvalidInput
		}
		sets = append(sets, "name=?")
		args = append(args, name)
	}
	if in.ClearIcon {
		sets = append(sets, "icon=NULL")
	} else if in.Icon != nil {
		sets = append(sets, "icon=?")
		args = append(args, *in.Icon)
	}
	if in.ClearColor {
		sets = append(sets, "color=NULL")
	} else if in.Color != nil {
		sets = append(sets, "color=?")
		args = append(args, *in.Color)
	}
	if in.ClearDescription {
		sets = append(sets, "description=NULL")
	} else if in.Description != nil {
		sets = append(sets, "description=?")
		args = append(args, *in.Description)
	}
	if len(in.SchemaJSON) > 0 {
		sets = append(sets, "schema_json=?")
		args = append(args, string(in.SchemaJSON))
	}
	if in.IsArchived != nil {
		sets = append(sets, "is_archived=?")
		v := 0
		if *in.IsArchived {
			v = 1
		}
		args = append(args, v)
	}
	if len(sets) == 0 {
		return s.GetByID(ctx, id)
	}
	sets = append(sets, "updated_at=CURRENT_TIMESTAMP")
	q := "UPDATE trackers SET " + strings.Join(sets, ", ") + " WHERE id=?"
	args = append(args, id)
	res, err := s.db.Conn().ExecContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.db.Conn().ExecContext(ctx, `DELETE FROM trackers WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface{ Scan(dest ...any) error }

func scanRow(r rowScanner) (*Tracker, error) {
	var (
		t          Tracker
		icon       sql.NullString
		color      sql.NullString
		desc       sql.NullString
		schemaText string
		archived   int64
	)
	if err := r.Scan(&t.ID, &t.ProfileID, &t.Name, &icon, &color, &desc, &schemaText, &archived, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	if icon.Valid {
		t.Icon = &icon.String
	}
	if color.Valid {
		t.Color = &color.String
	}
	if desc.Valid {
		t.Description = &desc.String
	}
	t.Schema = json.RawMessage(schemaText)
	t.IsArchived = archived != 0
	t.CreatedAt = t.CreatedAt.UTC()
	t.UpdatedAt = t.UpdatedAt.UTC()
	return &t, nil
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
