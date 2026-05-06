package profiles

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/bublya/baby-tracker/backend/internal/httpx"
	"github.com/bublya/baby-tracker/backend/internal/users"
	"github.com/go-chi/chi/v5"
)

// AccessLister returns the set of profile IDs accessible to the user.
// Implemented by access.Store.
type AccessLister interface {
	ListProfileIDs(ctx context.Context, u *users.User) ([]int64, error)
}

type Handlers struct {
	Store  *Store
	Access AccessLister
}

func NewHandlers(store *Store, access AccessLister) *Handlers {
	return &Handlers{Store: store, Access: access}
}

// Mount mounts user-accessible profile routes (any logged-in user).
func (h *Handlers) Mount(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Patch("/{id}", h.Update)
}

// MountAdmin mounts admin-only routes (DELETE).
func (h *Handlers) MountAdmin(r chi.Router) {
	r.Delete("/{id}", h.Delete)
}

type createRequest struct {
	Name        string  `json:"name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	u := users.FromContext(r.Context())
	if u == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ids, err := h.Access.ListProfileIDs(r.Context(), u)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list profiles")
		return
	}
	list, err := h.Store.ListByIDs(r.Context(), ids)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list profiles")
		return
	}
	if list == nil {
		list = []*Profile{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": list})
}

func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	p, err := h.Store.Create(r.Context(), req.Name, req.AvatarURL, req.Description)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "name required")
			return
		}
		writeError(w, http.StatusInternalServerError, "create profile")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"profile": p})
}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	// Decode into a raw map so we can distinguish absent vs explicit null.
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	in := UpdateInput{}
	if v, ok := raw["name"]; ok {
		var name string
		if err := json.Unmarshal(v, &name); err != nil {
			writeError(w, http.StatusBadRequest, "invalid name")
			return
		}
		in.Name = &name
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
	if !parseOptional("avatar_url", &in.AvatarURL, &in.ClearAvatar) {
		return
	}
	if !parseOptional("description", &in.Description, &in.ClearDescription) {
		return
	}
	p, err := h.Store.Update(r.Context(), id, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeError(w, http.StatusNotFound, "profile not found")
		case errors.Is(err, ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid input")
		default:
			writeError(w, http.StatusInternalServerError, "update profile")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profile": p})
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.Store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "profile not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "delete profile")
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
