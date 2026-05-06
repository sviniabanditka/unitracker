package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bublya/baby-tracker/backend/internal/users"
)

var ErrNoAdminConfigured = errors.New("users table is empty and INITIAL_ADMIN_USERNAME/INITIAL_ADMIN_PASSWORD are not set")

type BootstrapConfig struct {
	Username string
	Password string
}

func Bootstrap(ctx context.Context, store *users.Store, cfg BootstrapConfig) error {
	n, err := store.Count(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if n > 0 {
		if cfg.Username != "" || cfg.Password != "" {
			slog.Warn("INITIAL_ADMIN_* env ignored: users table is not empty")
		}
		return nil
	}
	if strings.TrimSpace(cfg.Username) == "" || cfg.Password == "" {
		return ErrNoAdminConfigured
	}
	hash, err := users.HashPassword(cfg.Password)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}
	u, err := store.Create(ctx, cfg.Username, hash, users.RoleAdmin)
	if err != nil {
		return fmt.Errorf("create bootstrap admin: %w", err)
	}
	slog.Info("bootstrap admin created", "id", u.ID, "username", u.Username)
	return nil
}
