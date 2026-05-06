package entries

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bublya/baby-tracker/backend/internal/httpx"
	"github.com/bublya/baby-tracker/backend/internal/trackers"
	"github.com/bublya/baby-tracker/backend/internal/users"
	"github.com/go-chi/chi/v5"
)

type Handlers struct{ Service *Service }

func NewHandlers(svc *Service) *Handlers { return &Handlers{Service: svc} }

// MountTrackerScoped registers /api/trackers/{tid}/entries (GET, POST).
func (h *Handlers) MountTrackerScoped(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
}

// MountTrackerStats registers /api/trackers/{tid}/stats (GET).
// Mounted under the same {tid} chi route as entries.
func (h *Handlers) MountTrackerStats(r chi.Router) {
	r.Get("/", h.Stats)
}

// MountEntryByID registers handlers under /api/entries/{id} — the {id} chi
// param is bound by the caller's Route("/{id}", ...) block.
func (h *Handlers) MountEntryByID(r chi.Router) {
	r.Get("/", h.Get)
	r.Patch("/", h.Update)
	r.Delete("/", h.Delete)
	r.Get("/revisions", h.ListRevisions)
	r.Post("/restore/{rid}", h.RestoreRevision)
	r.Post("/restore-deleted", h.RestoreDeleted)
}

type entryResponse struct {
	ID         int64           `json:"id"`
	TrackerID  int64           `json:"tracker_id"`
	ProfileID  *int64          `json:"profile_id"`
	Data       json.RawMessage `json:"data"`
	OccurredAt string          `json:"occurred_at"`
	CreatedBy  *int64          `json:"created_by"`
	IsDeleted  bool            `json:"is_deleted"`
	CreatedAt  string          `json:"created_at"`
	UpdatedAt  string          `json:"updated_at"`
}

type revisionResponse struct {
	ID         int64           `json:"id"`
	EntryID    int64           `json:"entry_id"`
	Data       json.RawMessage `json:"data"`
	OccurredAt string          `json:"occurred_at"`
	ProfileID  *int64          `json:"profile_id"`
	IsDeleted  bool            `json:"is_deleted"`
	ChangeType string          `json:"change_type"`
	ChangedBy  *int64          `json:"changed_by"`
	ChangedAt  string          `json:"changed_at"`
}

func toResp(e *Entry) entryResponse {
	return entryResponse{
		ID:         e.ID,
		TrackerID:  e.TrackerID,
		ProfileID:  e.ProfileID,
		Data:       e.Data,
		OccurredAt: e.OccurredAt,
		CreatedBy:  e.CreatedBy,
		IsDeleted:  e.IsDeleted,
		CreatedAt:  e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  e.UpdatedAt.Format(time.RFC3339),
	}
}

func toRevResp(r *Revision) revisionResponse {
	return revisionResponse{
		ID:         r.ID,
		EntryID:    r.EntryID,
		Data:       r.Data,
		OccurredAt: r.OccurredAt,
		ProfileID:  r.ProfileID,
		IsDeleted:  r.IsDeleted,
		ChangeType: r.ChangeType,
		ChangedBy:  r.ChangedBy,
		ChangedAt:  r.ChangedAt.Format(time.RFC3339),
	}
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	tid, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tracker id")
		return
	}
	q := r.URL.Query()
	f := ListFilter{TrackerID: tid, IncludeDeleted: q.Get("include_deleted") == "true"}
	if v := q.Get("from"); v != "" {
		t, err := parseFlexibleTime(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid from")
			return
		}
		f.From = &t
	}
	if v := q.Get("to"); v != "" {
		t, err := parseFlexibleTime(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid to")
			return
		}
		f.To = &t
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		f.Limit = n
	}
	f.Cursor = q.Get("cursor")
	items, next, err := h.Service.Store.List(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list entries")
		return
	}
	resp := make([]entryResponse, 0, len(items))
	for _, e := range items {
		resp = append(resp, toResp(e))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": resp, "next_cursor": next})
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	e, err := h.Service.Store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "entry not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load entry")
		return
	}
	if e.IsDeleted && r.URL.Query().Get("include_deleted") != "true" {
		writeError(w, http.StatusNotFound, "entry not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entry": toResp(e)})
}

type createBody struct {
	Data map[string]any `json:"data"`
}

func currentUserID(r *http.Request) *int64 {
	u := users.FromContext(r.Context())
	if u == nil {
		return nil
	}
	v := u.ID
	return &v
}

func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	tid, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tracker id")
		return
	}
	var body createBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Data == nil {
		body.Data = map[string]any{}
	}
	e, err := h.Service.Create(r.Context(), CreateRequest{
		TrackerID: tid,
		Data:      body.Data,
		UserID:    currentUserID(r),
	})
	if err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error":   "validation failed",
				"details": ve.Errors,
			})
			return
		}
		if errors.Is(err, trackers.ErrNotFound) {
			writeError(w, http.StatusNotFound, "tracker not found")
			return
		}
		if errors.Is(err, ErrTrackerArchived) {
			writeError(w, http.StatusBadRequest, "tracker is archived")
			return
		}
		writeError(w, http.StatusInternalServerError, "create entry")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"entry": toResp(e)})
}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req := UpdateRequest{EntryID: id, UserID: currentUserID(r)}
	if v, ok := raw["data"]; ok {
		var data map[string]any
		if err := json.Unmarshal(v, &data); err != nil {
			writeError(w, http.StatusBadRequest, "invalid data")
			return
		}
		if data == nil {
			data = map[string]any{}
		}
		req.Data = data
		req.DataSet = true
	}
	// profile_id in update body is intentionally ignored — the entry's profile
	// is bound via its tracker and cannot be changed.
	e, err := h.Service.Update(r.Context(), req)
	if err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error":   "validation failed",
				"details": ve.Errors,
			})
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "entry not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "update entry")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entry": toResp(e)})
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.Service.Delete(r.Context(), DeleteRequest{
		EntryID: id,
		UserID:  currentUserID(r),
	}); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "entry not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "delete entry")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) ListRevisions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := h.Service.Store.GetByID(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "entry not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load entry")
		return
	}
	revs, err := h.Service.Revisions.ListByEntry(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list revisions")
		return
	}
	resp := make([]revisionResponse, 0, len(revs))
	for _, rev := range revs {
		resp = append(resp, toRevResp(rev))
	}
	writeJSON(w, http.StatusOK, map[string]any{"revisions": resp})
}

func (h *Handlers) RestoreRevision(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	rid, err := strconv.ParseInt(chi.URLParam(r, "rid"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid revision id")
		return
	}
	e, err := h.Service.RestoreRevision(r.Context(), RestoreRevisionRequest{
		EntryID:    id,
		RevisionID: rid,
		UserID:     currentUserID(r),
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrRevisionNotFound):
			writeError(w, http.StatusNotFound, "revision not found")
		case errors.Is(err, ErrRevisionEntryMismatch):
			writeError(w, http.StatusBadRequest, "revision belongs to a different entry")
		case errors.Is(err, ErrNotFound):
			writeError(w, http.StatusNotFound, "entry not found")
		default:
			writeError(w, http.StatusInternalServerError, "restore revision")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entry": toResp(e)})
}

func (h *Handlers) RestoreDeleted(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	e, err := h.Service.RestoreDeleted(r.Context(), RestoreDeletedRequest{
		EntryID: id,
		UserID:  currentUserID(r),
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeError(w, http.StatusNotFound, "entry not found")
		case errors.Is(err, ErrEntryNotDeleted):
			writeError(w, http.StatusBadRequest, "entry is not deleted")
		default:
			writeError(w, http.StatusInternalServerError, "restore entry")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entry": toResp(e)})
}

func parseFlexibleTime(s string) (time.Time, error) {
	for _, l := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(l, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time format")
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	httpx.WriteJSON(w, code, v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	httpx.WriteErrorStatus(w, code, msg)
}
