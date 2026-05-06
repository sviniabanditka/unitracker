package users

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	appdb "github.com/bublya/baby-tracker/backend/internal/db"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrUsernameTaken = errors.New("username already taken")
	ErrInvalidRole   = errors.New("invalid role")
	ErrInvalidInput  = errors.New("invalid input")
)

type Store struct{ db *appdb.Database }

func NewStore(db *appdb.Database) *Store { return &Store{db: db} }

func (s *Store) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.Conn().QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) CountAdmins(ctx context.Context) (int, error) {
	var n int
	err := s.db.Conn().QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role='admin'`).Scan(&n)
	return n, err
}

func (s *Store) GetByID(ctx context.Context, id int64) (*User, error) {
	row := s.db.Conn().QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, created_at, updated_at FROM users WHERE id=?`, id)
	return scanRow(row)
}

func (s *Store) GetByUsername(ctx context.Context, username string) (*User, error) {
	row := s.db.Conn().QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, created_at, updated_at FROM users WHERE username=? COLLATE NOCASE`,
		strings.TrimSpace(username))
	return scanRow(row)
}

func (s *Store) List(ctx context.Context) ([]*User, error) {
	rows, err := s.db.Conn().QueryContext(ctx,
		`SELECT id, username, password_hash, role, created_at, updated_at FROM users ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*User
	for rows.Next() {
		u, err := scanRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) Create(ctx context.Context, username, passwordHash string, role Role) (*User, error) {
	username = strings.TrimSpace(username)
	if username == "" || passwordHash == "" {
		return nil, ErrInvalidInput
	}
	if !role.Valid() {
		return nil, ErrInvalidRole
	}
	res, err := s.db.Conn().ExecContext(ctx,
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
		username, passwordHash, string(role))
	if err != nil {
		if isUniqueErr(err) {
			return nil, ErrUsernameTaken
		}
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) UpdateUsername(ctx context.Context, id int64, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return ErrInvalidInput
	}
	res, err := s.db.Conn().ExecContext(ctx,
		`UPDATE users SET username=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, username, id)
	if err != nil {
		if isUniqueErr(err) {
			return ErrUsernameTaken
		}
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateRole(ctx context.Context, id int64, role Role) error {
	if !role.Valid() {
		return ErrInvalidRole
	}
	res, err := s.db.Conn().ExecContext(ctx,
		`UPDATE users SET role=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, string(role), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdatePassword(ctx context.Context, id int64, hash string) error {
	if hash == "" {
		return ErrInvalidInput
	}
	res, err := s.db.Conn().ExecContext(ctx,
		`UPDATE users SET password_hash=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, hash, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.db.Conn().ExecContext(ctx, `DELETE FROM users WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

type scanner interface{ Scan(dest ...any) error }

func scan(s scanner) (*User, error) {
	u := &User{}
	if err := s.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	u.CreatedAt = u.CreatedAt.UTC()
	u.UpdatedAt = u.UpdatedAt.UTC()
	return u, nil
}

func scanRow(row *sql.Row) (*User, error) {
	u, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func scanRows(rows *sql.Rows) (*User, error) { return scan(rows) }

func isUniqueErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed: UNIQUE")
}
