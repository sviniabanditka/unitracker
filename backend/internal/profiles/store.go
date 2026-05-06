package profiles

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	appdb "github.com/bublya/baby-tracker/backend/internal/db"
)

var (
	ErrNotFound     = errors.New("profile not found")
	ErrInvalidInput = errors.New("invalid input")
)

type Profile struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Store struct{ db *appdb.Database }

func NewStore(db *appdb.Database) *Store { return &Store{db: db} }

func (s *Store) List(ctx context.Context) ([]*Profile, error) {
	rows, err := s.db.Conn().QueryContext(ctx,
		`SELECT id, name, avatar_url, description, created_at, updated_at
		   FROM profiles ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Profile
	for rows.Next() {
		p, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListByIDs returns profiles whose ids are in the given set, in ascending order.
// Empty set returns an empty slice.
func (s *Store) ListByIDs(ctx context.Context, ids []int64) ([]*Profile, error) {
	if len(ids) == 0 {
		return []*Profile{}, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	q := `SELECT id, name, avatar_url, description, created_at, updated_at
	        FROM profiles WHERE id IN (` + strings.Join(placeholders, ",") + `) ORDER BY id ASC`
	rows, err := s.db.Conn().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Profile{}
	for rows.Next() {
		p, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Profile, error) {
	row := s.db.Conn().QueryRowContext(ctx,
		`SELECT id, name, avatar_url, description, created_at, updated_at
		   FROM profiles WHERE id=?`, id)
	p, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (s *Store) Create(ctx context.Context, name string, avatarURL, description *string) (*Profile, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	res, err := s.db.Conn().ExecContext(ctx,
		`INSERT INTO profiles (name, avatar_url, description) VALUES (?, ?, ?)`,
		name, nullableString(avatarURL), nullableString(description))
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
	AvatarURL        *string
	ClearAvatar      bool
	Description      *string
	ClearDescription bool
}

func (s *Store) Update(ctx context.Context, id int64, in UpdateInput) (*Profile, error) {
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
	if in.ClearAvatar {
		sets = append(sets, "avatar_url=NULL")
	} else if in.AvatarURL != nil {
		sets = append(sets, "avatar_url=?")
		args = append(args, *in.AvatarURL)
	}
	if in.ClearDescription {
		sets = append(sets, "description=NULL")
	} else if in.Description != nil {
		sets = append(sets, "description=?")
		args = append(args, *in.Description)
	}
	if len(sets) == 0 {
		return s.GetByID(ctx, id)
	}
	sets = append(sets, "updated_at=CURRENT_TIMESTAMP")
	q := "UPDATE profiles SET " + strings.Join(sets, ", ") + " WHERE id=?"
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
	res, err := s.db.Conn().ExecContext(ctx, `DELETE FROM profiles WHERE id=?`, id)
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

func scanRow(r rowScanner) (*Profile, error) {
	var (
		p      Profile
		avatar sql.NullString
		desc   sql.NullString
	)
	if err := r.Scan(&p.ID, &p.Name, &avatar, &desc, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	if avatar.Valid {
		p.AvatarURL = &avatar.String
	}
	if desc.Valid {
		p.Description = &desc.String
	}
	p.CreatedAt = p.CreatedAt.UTC()
	p.UpdatedAt = p.UpdatedAt.UTC()
	return &p, nil
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
