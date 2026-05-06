package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	appdb "github.com/bublya/baby-tracker/backend/internal/db"
)

var ErrSessionNotFound = errors.New("session not found")

type Session struct {
	ID         string    `json:"id"`
	UserID     int64     `json:"user_id"`
	ExpiresAt  time.Time `json:"expires_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type SessionStore struct{ db *appdb.Database }

func NewSessionStore(db *appdb.Database) *SessionStore { return &SessionStore{db: db} }

func (s *SessionStore) Create(ctx context.Context, userID int64, ttl time.Duration) (*Session, error) {
	id, err := newSessionID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	expires := now.Add(ttl)
	if _, err := s.db.Conn().ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		id, userID, expires); err != nil {
		return nil, err
	}
	return &Session{
		ID:         id,
		UserID:     userID,
		ExpiresAt:  expires,
		LastUsedAt: now,
		CreatedAt:  now,
	}, nil
}

func (s *SessionStore) Get(ctx context.Context, id string) (*Session, error) {
	sess := &Session{}
	err := s.db.Conn().QueryRowContext(ctx,
		`SELECT id, user_id, expires_at, last_used_at, created_at FROM sessions WHERE id=?`, id).
		Scan(&sess.ID, &sess.UserID, &sess.ExpiresAt, &sess.LastUsedAt, &sess.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	sess.ExpiresAt = sess.ExpiresAt.UTC()
	sess.LastUsedAt = sess.LastUsedAt.UTC()
	sess.CreatedAt = sess.CreatedAt.UTC()
	return sess, nil
}

func (s *SessionStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.Conn().ExecContext(ctx, `DELETE FROM sessions WHERE id=?`, id)
	return err
}

func (s *SessionStore) DeleteAllForUser(ctx context.Context, userID int64) error {
	_, err := s.db.Conn().ExecContext(ctx, `DELETE FROM sessions WHERE user_id=?`, userID)
	return err
}

func (s *SessionStore) DeleteOthersForUser(ctx context.Context, userID int64, keepID string) error {
	_, err := s.db.Conn().ExecContext(ctx, `DELETE FROM sessions WHERE user_id=? AND id<>?`, userID, keepID)
	return err
}

func (s *SessionStore) Touch(ctx context.Context, id string) error {
	_, err := s.db.Conn().ExecContext(ctx, `UPDATE sessions SET last_used_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	return err
}

func (s *SessionStore) GC(ctx context.Context) (int64, error) {
	res, err := s.db.Conn().ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func newSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
