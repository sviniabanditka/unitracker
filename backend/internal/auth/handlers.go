package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/bublya/baby-tracker/backend/internal/users"
	"github.com/go-chi/chi/v5"
)

func (svc *Service) Mount(r chi.Router) {
	r.Post("/login", svc.Login)
	r.Group(func(pr chi.Router) {
		pr.Use(svc.RequireUser)
		pr.Post("/logout", svc.Logout)
		pr.Get("/me", svc.Me)
		pr.Post("/change-password", svc.ChangePassword)
	})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (svc *Service) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}
	u, err := svc.Users.GetByUsername(r.Context(), req.Username)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := users.ComparePassword(u.PasswordHash, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	sess, err := svc.Sessions.Create(r.Context(), u.ID, svc.TTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create session")
		return
	}
	svc.SetCookie(w, sess.ID, sess.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{"user": u})
}

func (svc *Service) Logout(w http.ResponseWriter, r *http.Request) {
	sess := SessionFromContext(r.Context())
	if sess != nil {
		_ = svc.Sessions.Delete(r.Context(), sess.ID)
	}
	svc.ClearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (svc *Service) Me(w http.ResponseWriter, r *http.Request) {
	u := users.FromContext(r.Context())
	if u == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u})
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (svc *Service) ChangePassword(w http.ResponseWriter, r *http.Request) {
	u := users.FromContext(r.Context())
	sess := SessionFromContext(r.Context())
	if u == nil || sess == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "new password must be at least 8 characters")
		return
	}
	if err := users.ComparePassword(u.PasswordHash, req.OldPassword); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid old password")
		return
	}
	hash, err := users.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hash password")
		return
	}
	if err := svc.Users.UpdatePassword(r.Context(), u.ID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "update password")
		return
	}
	_ = svc.Sessions.DeleteOthersForUser(r.Context(), u.ID, sess.ID)
	w.WriteHeader(http.StatusNoContent)
}
