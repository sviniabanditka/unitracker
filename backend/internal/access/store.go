package access

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	appdb "github.com/bublya/baby-tracker/backend/internal/db"
	"github.com/bublya/baby-tracker/backend/internal/users"
)

// Store backs the user_profile_access M:N table.
//
// Admins have implicit access to every profile and never have rows here.
// Members are restricted to whatever rows admin has explicitly granted.
type Store struct{ db *appdb.Database }

func NewStore(db *appdb.Database) *Store { return &Store{db: db} }

// ListProfileIDs returns profile_ids the user has been granted access to.
// For admins, returns all profile ids in the system. The result is sorted asc.
func (s *Store) ListProfileIDs(ctx context.Context, u *users.User) ([]int64, error) {
	if u == nil {
		return nil, errors.New("nil user")
	}
	if u.Role == users.RoleAdmin {
		return s.allProfileIDs(ctx)
	}
	rows, err := s.db.Conn().QueryContext(ctx,
		`SELECT profile_id FROM user_profile_access WHERE user_id=? ORDER BY profile_id ASC`, u.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ListGrantedProfileIDs returns profile_ids granted to the user, ignoring role.
// Used by admin endpoints to inspect grants for any user.
func (s *Store) ListGrantedProfileIDs(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := s.db.Conn().QueryContext(ctx,
		`SELECT profile_id FROM user_profile_access WHERE user_id=? ORDER BY profile_id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// HasAccess reports whether the user can interact with the given profile.
// Admins always pass.
func (s *Store) HasAccess(ctx context.Context, u *users.User, profileID int64) (bool, error) {
	if u == nil {
		return false, errors.New("nil user")
	}
	if u.Role == users.RoleAdmin {
		var n int
		if err := s.db.Conn().QueryRowContext(ctx,
			`SELECT COUNT(*) FROM profiles WHERE id=?`, profileID).Scan(&n); err != nil {
			return false, err
		}
		return n > 0, nil
	}
	var n int
	err := s.db.Conn().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_profile_access WHERE user_id=? AND profile_id=?`,
		u.ID, profileID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Replace atomically sets the user's access list to the given profile_ids.
// grantedBy is the admin performing the change (nullable).
// Returns ErrUnknownProfile if any id does not match an existing profile.
func (s *Store) Replace(ctx context.Context, userID int64, profileIDs []int64, grantedBy *int64) error {
	tx, err := s.db.Conn().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_profile_access WHERE user_id=?`, userID); err != nil {
		return err
	}
	if len(profileIDs) > 0 {
		seen := make(map[int64]struct{}, len(profileIDs))
		for _, pid := range profileIDs {
			if _, ok := seen[pid]; ok {
				continue
			}
			seen[pid] = struct{}{}
			args := []any{userID, pid}
			var by any
			if grantedBy != nil {
				by = *grantedBy
			} else {
				by = nil
			}
			args = append(args, by)
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO user_profile_access (user_id, profile_id, granted_by) VALUES (?, ?, ?)`,
				args...); err != nil {
				if isFKErr(err) {
					return ErrUnknownProfile
				}
				return err
			}
		}
	}
	return tx.Commit()
}

var ErrUnknownProfile = errors.New("unknown profile_id")

func isFKErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "FOREIGN KEY") || strings.Contains(msg, "foreign key")
}

func (s *Store) allProfileIDs(ctx context.Context) ([]int64, error) {
	rows, err := s.db.Conn().QueryContext(ctx, `SELECT id FROM profiles ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// asAny converts an *int64 into the value sql wants — used by Replace.
var _ = sql.ErrNoRows // keep sql import alive in case downstream needs it
