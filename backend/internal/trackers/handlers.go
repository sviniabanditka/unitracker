package trackers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/bublya/baby-tracker/backend/internal/httpx"
	"github.com/bublya/baby-tracker/backend/internal/users"
	"github.com/go-chi/chi/v5"
)

// EntryCounter is an optional dependency that lets the tracker handler include
// a "schema changed; N entries exist" warning in PATCH responses.
type EntryCounter interface {
	CountByTracker(ctx context.Context, trackerID int64) (int, error)
}

// AccessLister returns the set of profile IDs accessible to the user.
// Implemented by access.Store; used by the library endpoint.
type AccessLister interface {
	ListProfileIDs(ctx context.Context, u *users.User) ([]int64, error)
}

type Handlers struct {
	Store        *Store
	Access       AccessLister
	EntryCounter EntryCounter
}

func NewHandlers(store *Store, access AccessLister) *Handlers {
	return &Handlers{Store: store, Access: access}
}

// MountProfileScoped mounts /api/profiles/{pid}/trackers. Caller is expected
// to have already gated access via access.RequireProfileAccess.
func (h *Handlers) MountProfileScoped(r chi.Router) {
	r.Get("/", h.ListByProfile)
}

// MountProfileScopedAdmin mounts admin-only /api/profiles/{pid}/trackers (POST).
func (h *Handlers) MountProfileScopedAdmin(r chi.Router) {
	r.Post("/", h.CreateInProfile)
}

// MountLibrary mounts GET /api/trackers/library — list of trackers across all
// profiles the user can access (with profile_name attached).
func (h *Handlers) MountLibrary(r chi.Router) {
	r.Get("/", h.Library)
}

// MountTrackerScoped mounts read-only /api/trackers/{tid} routes. Access is
// gated upstream via access.RequireTrackerAccess.
func (h *Handlers) MountTrackerScoped(r chi.Router) {
	r.Get("/", h.Get)
}

// MountTrackerScopedAdmin mounts admin-only /api/trackers/{tid} writes.
func (h *Handlers) MountTrackerScopedAdmin(r chi.Router) {
	r.Patch("/", h.Update)
	r.Post("/archive", h.ToggleArchive)
	r.Delete("/", h.Delete)
}

var hexColorRe = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

type createRequest struct {
	Name        string          `json:"name"`
	Icon        *string         `json:"icon,omitempty"`
	Color       *string         `json:"color,omitempty"`
	Description *string         `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema_json"`
}

type archiveRequest struct {
	IsArchived bool `json:"is_archived"`
}

func (h *Handlers) ListByProfile(w http.ResponseWriter, r *http.Request) {
	pid, err := strconv.ParseInt(chi.URLParam(r, "pid"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid profile id")
		return
	}
	includeArchived := r.URL.Query().Get("include_archived") == "true"
	list, err := h.Store.ListByProfile(r.Context(), pid, includeArchived)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list trackers")
		return
	}
	if list == nil {
		list = []*Tracker{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"trackers": list})
}

func (h *Handlers) Library(w http.ResponseWriter, r *http.Request) {
	u := users.FromContext(r.Context())
	if u == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	allowed, err := h.Access.ListProfileIDs(r.Context(), u)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list access")
		return
	}
	list, err := h.Store.Library(r.Context(), allowed)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list library")
		return
	}
	if list == nil {
		list = []*LibraryTracker{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"trackers": list})
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := h.Store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "tracker not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load tracker")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tracker": t})
}

func (h *Handlers) CreateInProfile(w http.ResponseWriter, r *http.Request) {
	pid, err := strconv.ParseInt(chi.URLParam(r, "pid"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid profile id")
		return
	}
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Color != nil && *req.Color != "" && !hexColorRe.MatchString(*req.Color) {
		writeError(w, http.StatusBadRequest, "color must be hex (#RGB or #RRGGBB)")
		return
	}
	if len(req.Schema) == 0 {
		writeError(w, http.StatusBadRequest, "schema_json is required")
		return
	}
	parsed, err := ParseSchema(req.Schema)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	canon, err := MarshalSchema(parsed)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "marshal schema")
		return
	}
	t, err := h.Store.Create(r.Context(), CreateInput{
		ProfileID:   pid,
		Name:        req.Name,
		Icon:        req.Icon,
		Color:       req.Color,
		Description: req.Description,
		SchemaJSON:  canon,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "name and schema_json are required")
			return
		}
		writeError(w, http.StatusInternalServerError, "create tracker")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"tracker": t})
}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	in := UpdateInput{}
	if v, ok := raw["name"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			writeError(w, http.StatusBadRequest, "invalid name")
			return
		}
		in.Name = &s
	}
	parseOptional := func(key string, target **string, clear *bool) bool {
		v, ok := raw[key]
		if !ok {
			return true
		}
		if string(v) == "null" {
			*clear = true
			return true
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			writeError(w, http.StatusBadRequest, "invalid "+key)
			return false
		}
		if s == "" {
			*clear = true
			return true
		}
		*target = &s
		return true
	}
	if !parseOptional("icon", &in.Icon, &in.ClearIcon) {
		return
	}
	if !parseOptional("color", &in.Color, &in.ClearColor) {
		return
	}
	if !parseOptional("description", &in.Description, &in.ClearDescription) {
		return
	}
	if in.Color != nil && !hexColorRe.MatchString(*in.Color) {
		writeError(w, http.StatusBadRequest, "color must be hex (#RGB or #RRGGBB)")
		return
	}
	schemaChanged := false
	if v, ok := raw["schema_json"]; ok {
		parsed, err := ParseSchema(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		canon, err := MarshalSchema(parsed)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "marshal schema")
			return
		}
		in.SchemaJSON = canon
		schemaChanged = true
	}
	if v, ok := raw["is_archived"]; ok {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			writeError(w, http.StatusBadRequest, "invalid is_archived")
			return
		}
		in.IsArchived = &b
	}
	// profile_id is intentionally not honoured here — trackers are bound to
	// their owning profile at creation and cannot be moved.
	t, err := h.Store.Update(r.Context(), id, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeError(w, http.StatusNotFound, "tracker not found")
		case errors.Is(err, ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid input")
		default:
			writeError(w, http.StatusInternalServerError, "update tracker")
		}
		return
	}
	warnings := []string{}
	if schemaChanged && h.EntryCounter != nil {
		if n, err := h.EntryCounter.CountByTracker(r.Context(), id); err == nil && n > 0 {
			warnings = append(warnings,
				fmt.Sprintf("Schema changed: %d existing entries may not match the new shape", n))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tracker": t, "warnings": warnings})
}

func (h *Handlers) ToggleArchive(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req archiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	t, err := h.Store.Update(r.Context(), id, UpdateInput{IsArchived: &req.IsArchived})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "tracker not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "archive tracker")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tracker": t})
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.Store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "tracker not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "delete tracker")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	httpx.WriteJSON(w, code, v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	httpx.WriteErrorStatus(w, code, msg)
}
