package entries

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// ListInRange returns non-deleted entries with occurred_at in [from, to),
// optionally filtered by profile. Sorted by occurred_at DESC.
func (s *Store) ListInRange(ctx context.Context, profileID *int64, from, to time.Time) ([]*Entry, error) {
	where := []string{"is_deleted=0", "occurred_at>=?", "occurred_at<?"}
	args := []any{from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339)}
	if profileID != nil {
		where = append(where, "profile_id=?")
		args = append(args, *profileID)
	}
	q := `SELECT id, tracker_id, profile_id, data_json, occurred_at, created_by, is_deleted, created_at, updated_at
	        FROM entries WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY occurred_at DESC, id DESC`
	rows, err := s.db.Conn().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Entry{}
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// LastByTracker returns the most recent non-deleted entry for a tracker, or
// nil if none exists. Tracker scope already pins the profile.
func (s *Store) LastByTracker(ctx context.Context, trackerID int64) (*Entry, error) {
	q := `SELECT id, tracker_id, profile_id, data_json, occurred_at, created_by, is_deleted, created_at, updated_at
	        FROM entries WHERE tracker_id=? AND is_deleted=0
	       ORDER BY occurred_at DESC, id DESC LIMIT 1`
	row := s.db.Conn().QueryRowContext(ctx, q, trackerID)
	e, err := scanEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

// TopTrackerIDsSince returns tracker ids ordered by entry count since the
// given timestamp, biggest first. Optionally filtered by profile.
func (s *Store) TopTrackerIDsSince(ctx context.Context, profileID *int64, since time.Time, limit int) ([]int64, error) {
	if limit <= 0 {
		limit = 3
	}
	where := []string{"is_deleted=0", "occurred_at>=?"}
	args := []any{since.UTC().Format(time.RFC3339)}
	if profileID != nil {
		where = append(where, "profile_id=?")
		args = append(args, *profileID)
	}
	q := `SELECT tracker_id, COUNT(*) AS n FROM entries WHERE ` + strings.Join(where, " AND ") +
		` GROUP BY tracker_id ORDER BY n DESC, tracker_id ASC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Conn().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
