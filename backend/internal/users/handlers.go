package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/bublya/baby-tracker/backend/internal/httpx"
	"github.com/go-chi/chi/v5"
)

type SessionInvalidator interface {
	DeleteAllForUser(ctx context.Context, userID int64) error
}

type Handlers struct {
	Store    *Store
	Sessions SessionInvalidator
}

func NewHandlers(store *Store, sessions SessionInvalidator) *Handlers {
	return &Handlers{Store: store, Sessions: sessions}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Patch("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
}

type createRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     Role   `json:"role"`
}

type updateRequest struct {
	Username    *string `json:"username,omitempty"`
	Role        *Role   `json:"role,omitempty"`
	NewPassword *string `json:"new_password,omitempty"`
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list users")
		return
	}
	if list == nil {
		list = []*User{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": list})
}

func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		writeError(w, http.StatusBadRequest, "username required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if !req.Role.Valid() {
		writeError(w, http.StatusBadRequest, "invalid role")
		return
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hash password")
		return
	}
	u, err := h.Store.Create(r.Context(), req.Username, hash, req.Role)
	if err != nil {
		switch {
		case errors.Is(err, ErrUsernameTaken):
			writeError(w, http.StatusConflict, "username already taken")
		case errors.Is(err, ErrInvalidRole), errors.Is(err, ErrInvalidInput):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "create user")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": u})
}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	target, err := h.Store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load user")
		return
	}
	current := FromContext(r.Context())
	if req.Role != nil && *req.Role != target.Role {
		if !req.Role.Valid() {
			writeError(w, http.StatusBadRequest, "invalid role")
			return
		}
		if target.Role == RoleAdmin && *req.Role != RoleAdmin {
			admins, err := h.Store.CountAdmins(r.Context())
			if err != nil {
				writeError(w, http.StatusInternalServerError, "count admins")
				return
			}
			if admins <= 1 {
				writeError(w, http.StatusBadRequest, "cannot demote the last admin")
				return
			}
		}
		if err := h.Store.UpdateRole(r.Context(), id, *req.Role); err != nil {
			writeError(w, http.StatusInternalServerError, "update role")
			return
		}
	}
	if req.Username != nil {
		if err := h.Store.UpdateUsername(r.Context(), id, *req.Username); err != nil {
			switch {
			case errors.Is(err, ErrUsernameTaken):
				writeError(w, http.StatusConflict, "username already taken")
			case errors.Is(err, ErrInvalidInput):
				writeError(w, http.StatusBadRequest, err.Error())
			default:
				writeError(w, http.StatusInternalServerError, "update username")
			}
			return
		}
	}
	if req.NewPassword != nil {
		if len(*req.NewPassword) < 8 {
			writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}
		hash, err := HashPassword(*req.NewPassword)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "hash password")
			return
		}
		if err := h.Store.UpdatePassword(r.Context(), id, hash); err != nil {
			writeError(w, http.StatusInternalServerError, "update password")
			return
		}
		// Reset by admin invalidates all sessions of the affected user;
		// self-resets go through the auth.ChangePassword endpoint instead.
		if current == nil || current.ID != id {
			if h.Sessions != nil {
				_ = h.Sessions.DeleteAllForUser(r.Context(), id)
			}
		}
	}
	updated, err := h.Store.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "reload user")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": updated})
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	current := FromContext(r.Context())
	if current != nil && current.ID == id {
		writeError(w, http.StatusBadRequest, "cannot delete yourself")
		return
	}
	target, err := h.Store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load user")
		return
	}
	if target.Role == RoleAdmin {
		admins, err := h.Store.CountAdmins(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "count admins")
			return
		}
		if admins <= 1 {
			writeError(w, http.StatusBadRequest, "cannot delete the last admin")
			return
		}
	}
	if err := h.Store.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "delete user")
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
