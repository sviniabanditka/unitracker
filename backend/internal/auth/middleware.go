package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bublya/baby-tracker/backend/internal/httpx"
	"github.com/bublya/baby-tracker/backend/internal/users"
)

const SessionCookieName = "session_id"

type CookieConfig struct {
	Secure bool
}

type Service struct {
	Users    *users.Store
	Sessions *SessionStore
	TTL      time.Duration
	Cookie   CookieConfig
}

type sessionCtxKey struct{}

func sessionToContext(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, sessionCtxKey{}, s)
}

func SessionFromContext(ctx context.Context) *Session {
	s, _ := ctx.Value(sessionCtxKey{}).(*Session)
	return s
}

func (svc *Service) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		sess, err := svc.Sessions.Get(r.Context(), cookie.Value)
		if err != nil {
			svc.ClearCookie(w)
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if sess.ExpiresAt.Before(time.Now().UTC()) {
			_ = svc.Sessions.Delete(r.Context(), sess.ID)
			svc.ClearCookie(w)
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		u, err := svc.Users.GetByID(r.Context(), sess.UserID)
		if err != nil {
			if errors.Is(err, users.ErrNotFound) {
				_ = svc.Sessions.Delete(r.Context(), sess.ID)
				svc.ClearCookie(w)
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if time.Since(sess.LastUsedAt) > 24*time.Hour {
			_ = svc.Sessions.Touch(r.Context(), sess.ID)
		}
		ctx := users.WithContext(r.Context(), u)
		ctx = sessionToContext(ctx, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (svc *Service) RequireAdmin(next http.Handler) http.Handler {
	return svc.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := users.FromContext(r.Context())
		if u == nil || u.Role != users.RoleAdmin {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func (svc *Service) SetCookie(w http.ResponseWriter, value string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   svc.Cookie.Secure,
	})
}

func (svc *Service) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   svc.Cookie.Secure,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	httpx.WriteJSON(w, code, v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	httpx.WriteErrorStatus(w, code, msg)
}
