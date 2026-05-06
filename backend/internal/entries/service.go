package entries

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/bublya/baby-tracker/backend/internal/trackers"
)

type TrackerLookup interface {
	GetByID(ctx context.Context, id int64) (*trackers.Tracker, error)
	ListByProfile(ctx context.Context, profileID int64, includeArchived bool) ([]*trackers.Tracker, error)
}

type Service struct {
	Store     *Store
	Revisions *RevisionStore
	Trackers  TrackerLookup
}

func NewService(store *Store, revisions *RevisionStore, lookup TrackerLookup) *Service {
	return &Service{Store: store, Revisions: revisions, Trackers: lookup}
}

type ValidationError struct{ Errors []string }

func (e *ValidationError) Error() string {
	return "validation failed: " + strings.Join(e.Errors, "; ")
}

var (
	ErrTrackerArchived       = errors.New("tracker is archived")
	ErrRevisionEntryMismatch = errors.New("revision does not belong to this entry")
	ErrEntryNotDeleted       = errors.New("entry is not deleted")
)

type CreateRequest struct {
	TrackerID int64
	Data      map[string]any
	UserID    *int64
}

func (svc *Service) Create(ctx context.Context, req CreateRequest) (*Entry, error) {
	t, err := svc.Trackers.GetByID(ctx, req.TrackerID)
	if err != nil {
		return nil, err
	}
	if t.IsArchived {
		return nil, ErrTrackerArchived
	}
	schema, err := trackers.ParseSchema(t.Schema)
	if err != nil {
		return nil, err
	}
	if errs := ValidateData(schema, req.Data); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}
	occurred, err := ExtractOccurredAt(schema, req.Data, time.Now().UTC())
	if err != nil {
		return nil, &ValidationError{Errors: []string{err.Error()}}
	}
	raw, err := json.Marshal(req.Data)
	if err != nil {
		return nil, err
	}

	// Profile_id is denormalised on entries — it's always derived from the
	// tracker, never trusted from the client.
	profileID := t.ProfileID

	var entry *Entry
	err = svc.withTx(ctx, func(tx *sql.Tx) error {
		e, err := createTx(ctx, tx, CreateInput{
			TrackerID:  req.TrackerID,
			ProfileID:  &profileID,
			DataJSON:   raw,
			OccurredAt: occurred,
			CreatedBy:  req.UserID,
		})
		if err != nil {
			return err
		}
		if err := appendRevisionTx(ctx, tx, AppendInput{
			EntryID:    e.ID,
			DataJSON:   raw,
			OccurredAt: e.OccurredAt,
			ProfileID:  e.ProfileID,
			IsDeleted:  e.IsDeleted,
			ChangeType: ChangeCreate,
			ChangedBy:  req.UserID,
		}); err != nil {
			return err
		}
		entry = e
		return nil
	})
	return entry, err
}

type UpdateRequest struct {
	EntryID int64
	Data    map[string]any
	DataSet bool
	UserID  *int64
}

func (svc *Service) Update(ctx context.Context, req UpdateRequest) (*Entry, error) {
	existing, err := svc.Store.GetByID(ctx, req.EntryID)
	if err != nil {
		return nil, err
	}
	if existing.IsDeleted {
		return nil, ErrNotFound
	}
	in := UpdateInput{}
	if req.DataSet {
		t, err := svc.Trackers.GetByID(ctx, existing.TrackerID)
		if err != nil {
			return nil, err
		}
		schema, err := trackers.ParseSchema(t.Schema)
		if err != nil {
			return nil, err
		}
		if errs := ValidateData(schema, req.Data); len(errs) > 0 {
			return nil, &ValidationError{Errors: errs}
		}
		occurred, err := ExtractOccurredAt(schema, req.Data, time.Now().UTC())
		if err != nil {
			return nil, &ValidationError{Errors: []string{err.Error()}}
		}
		raw, err := json.Marshal(req.Data)
		if err != nil {
			return nil, err
		}
		in.DataJSON = raw
		in.OccurredAt = &occurred
	}
	var entry *Entry
	err = svc.withTx(ctx, func(tx *sql.Tx) error {
		e, err := updateTx(ctx, tx, req.EntryID, in)
		if err != nil {
			return err
		}
		if err := appendRevisionTx(ctx, tx, AppendInput{
			EntryID:    e.ID,
			DataJSON:   []byte(e.Data),
			OccurredAt: e.OccurredAt,
			ProfileID:  e.ProfileID,
			IsDeleted:  e.IsDeleted,
			ChangeType: ChangeUpdate,
			ChangedBy:  req.UserID,
		}); err != nil {
			return err
		}
		entry = e
		return nil
	})
	return entry, err
}

type DeleteRequest struct {
	EntryID int64
	UserID  *int64
}

func (svc *Service) Delete(ctx context.Context, req DeleteRequest) error {
	return svc.withTx(ctx, func(tx *sql.Tx) error {
		e, changed, err := softDeleteTx(ctx, tx, req.EntryID)
		if err != nil {
			return err
		}
		if !changed {
			// Already deleted — idempotent, no new revision.
			return nil
		}
		return appendRevisionTx(ctx, tx, AppendInput{
			EntryID:    e.ID,
			DataJSON:   []byte(e.Data),
			OccurredAt: e.OccurredAt,
			ProfileID:  e.ProfileID,
			IsDeleted:  e.IsDeleted,
			ChangeType: ChangeDelete,
			ChangedBy:  req.UserID,
		})
	})
}

type RestoreRevisionRequest struct {
	EntryID    int64
	RevisionID int64
	UserID     *int64
}

func (svc *Service) RestoreRevision(ctx context.Context, req RestoreRevisionRequest) (*Entry, error) {
	rev, err := svc.Revisions.GetByID(ctx, req.RevisionID)
	if err != nil {
		return nil, err
	}
	if rev.EntryID != req.EntryID {
		return nil, ErrRevisionEntryMismatch
	}
	var entry *Entry
	err = svc.withTx(ctx, func(tx *sql.Tx) error {
		e, err := applyRevisionTx(ctx, tx, req.EntryID,
			[]byte(rev.Data), rev.OccurredAt, rev.ProfileID, rev.IsDeleted)
		if err != nil {
			return err
		}
		if err := appendRevisionTx(ctx, tx, AppendInput{
			EntryID:    e.ID,
			DataJSON:   []byte(e.Data),
			OccurredAt: e.OccurredAt,
			ProfileID:  e.ProfileID,
			IsDeleted:  e.IsDeleted,
			ChangeType: ChangeRestore,
			ChangedBy:  req.UserID,
		}); err != nil {
			return err
		}
		entry = e
		return nil
	})
	return entry, err
}

type RestoreDeletedRequest struct {
	EntryID int64
	UserID  *int64
}

func (svc *Service) RestoreDeleted(ctx context.Context, req RestoreDeletedRequest) (*Entry, error) {
	var entry *Entry
	err := svc.withTx(ctx, func(tx *sql.Tx) error {
		e, changed, err := restoreDeletedTx(ctx, tx, req.EntryID)
		if err != nil {
			return err
		}
		if !changed {
			return ErrEntryNotDeleted
		}
		if err := appendRevisionTx(ctx, tx, AppendInput{
			EntryID:    e.ID,
			DataJSON:   []byte(e.Data),
			OccurredAt: e.OccurredAt,
			ProfileID:  e.ProfileID,
			IsDeleted:  e.IsDeleted,
			ChangeType: ChangeRestore,
			ChangedBy:  req.UserID,
		}); err != nil {
			return err
		}
		entry = e
		return nil
	})
	return entry, err
}

func (svc *Service) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := svc.Store.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
