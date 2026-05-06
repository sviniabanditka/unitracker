package access

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

// TrackerLookup loads a tracker by id; the tracker provides a ProfileID.
// Implemented by trackers.Store (returns *trackers.Tracker which has ProfileID).
type TrackerLookup interface {
	GetProfileID(ctx context.Context, trackerID int64) (int64, error)
}

type Handlers struct {
	Store    *Store
	Trackers TrackerLookup
}

func NewHandlers(store *Store, trackers TrackerLookup) *Handlers {
	return &Handlers{Store: store, Trackers: trackers}
}

// MountAdminUserAccess mounts /api/admin/users/{uid}/profiles routes.
// Caller wraps with RequireAdmin.
func (h *Handlers) MountAdminUserAccess(r chi.Router) {
	r.Get("/", h.ListGranted)
	r.Put("/", h.Replace)
}

func (h *Handlers) ListGranted(w http.ResponseWriter, r *http.Request) {
	uid, err := strconv.ParseInt(chi.URLParam(r, "uid"), 10, 64)
	if err != nil {
		httpx.WriteErrorStatus(w, http.StatusBadRequest, "invalid user id")
		return
	}
	ids, err := h.Store.ListGrantedProfileIDs(r.Context(), uid)
	if err != nil {
		httpx.WriteErrorStatus(w, http.StatusInternalServerError, "list access")
		return
	}
	if ids == nil {
		ids = []int64{}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"profile_ids": ids})
}

type replaceBody struct {
	ProfileIDs []int64 `json:"profile_ids"`
}

func (h *Handlers) Replace(w http.ResponseWriter, r *http.Request) {
	uid, err := strconv.ParseInt(chi.URLParam(r, "uid"), 10, 64)
	if err != nil {
		httpx.WriteErrorStatus(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var body replaceBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteErrorStatus(w, http.StatusBadRequest, "invalid body")
		return
	}
	current := users.FromContext(r.Context())
	var grantedBy *int64
	if current != nil {
		v := current.ID
		grantedBy = &v
	}
	if err := h.Store.Replace(r.Context(), uid, body.ProfileIDs, grantedBy); err != nil {
		if errors.Is(err, ErrUnknownProfile) {
			httpx.WriteValidationError(w, "validation failed", map[string]string{
				"profile_ids": "contains unknown profile_id",
			})
			return
		}
		httpx.WriteErrorStatus(w, http.StatusInternalServerError, "replace access")
		return
	}
	ids, err := h.Store.ListGrantedProfileIDs(r.Context(), uid)
	if err != nil {
		httpx.WriteErrorStatus(w, http.StatusInternalServerError, "reload access")
		return
	}
	if ids == nil {
		ids = []int64{}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"profile_ids": ids})
}

// RequireProfileAccess returns a chi middleware that 404s when the current user
// can't access the profile given by URL parameter `pidParam`.
//
// Admins pass through unconditionally.
func RequireProfileAccess(store *Store, pidParam string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pid, err := strconv.ParseInt(chi.URLParam(r, pidParam), 10, 64)
			if err != nil {
				httpx.WriteErrorStatus(w, http.StatusNotFound, "not found")
				return
			}
			u := users.FromContext(r.Context())
			if u == nil {
				httpx.WriteErrorStatus(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			ok, err := store.HasAccess(r.Context(), u, pid)
			if err != nil {
				httpx.WriteErrorStatus(w, http.StatusInternalServerError, "access check")
				return
			}
			if !ok {
				httpx.WriteErrorStatus(w, http.StatusNotFound, "not found")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireProfileAccessFromQuery gates a route by checking access to a profile
// id read from the request query string. 404 when missing/denied.
func RequireProfileAccessFromQuery(store *Store, queryName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.URL.Query().Get(queryName)
			if raw == "" {
				httpx.WriteErrorStatus(w, http.StatusBadRequest, queryName+" is required")
				return
			}
			pid, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				httpx.WriteErrorStatus(w, http.StatusBadRequest, "invalid "+queryName)
				return
			}
			u := users.FromContext(r.Context())
			if u == nil {
				httpx.WriteErrorStatus(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			ok, err := store.HasAccess(r.Context(), u, pid)
			if err != nil {
				httpx.WriteErrorStatus(w, http.StatusInternalServerError, "access check")
				return
			}
			if !ok {
				httpx.WriteErrorStatus(w, http.StatusNotFound, "not found")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireTrackerAccess looks up the tracker's profile_id and gates access.
//
// Returns 404 on missing tracker or denied access.
func RequireTrackerAccess(store *Store, trackers TrackerLookup, tidParam string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tid, err := strconv.ParseInt(chi.URLParam(r, tidParam), 10, 64)
			if err != nil {
				httpx.WriteErrorStatus(w, http.StatusNotFound, "not found")
				return
			}
			u := users.FromContext(r.Context())
			if u == nil {
				httpx.WriteErrorStatus(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			pid, err := trackers.GetProfileID(r.Context(), tid)
			if err != nil {
				httpx.WriteErrorStatus(w, http.StatusNotFound, "not found")
				return
			}
			ok, err := store.HasAccess(r.Context(), u, pid)
			if err != nil {
				httpx.WriteErrorStatus(w, http.StatusInternalServerError, "access check")
				return
			}
			if !ok {
				httpx.WriteErrorStatus(w, http.StatusNotFound, "not found")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
