package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/bublya/baby-tracker/backend/internal/access"
	"github.com/bublya/baby-tracker/backend/internal/auth"
	"github.com/bublya/baby-tracker/backend/internal/backup"
	appdb "github.com/bublya/baby-tracker/backend/internal/db"
	"github.com/bublya/baby-tracker/backend/internal/entries"
	apphttp "github.com/bublya/baby-tracker/backend/internal/http"
	"github.com/bublya/baby-tracker/backend/internal/profiles"
	"github.com/bublya/baby-tracker/backend/internal/settings"
	"github.com/bublya/baby-tracker/backend/internal/trackers"
	"github.com/bublya/baby-tracker/backend/internal/users"
	"github.com/joho/godotenv"
)

const sessionTTL = 30 * 24 * time.Hour

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if path, ok := findDotEnv(); ok {
		if err := godotenv.Load(path); err != nil {
			slog.Warn("load .env", "path", path, "err", err)
		} else {
			slog.Info("loaded .env", "path", path)
		}
	}

	dataDir := envOr("DATA_DIR", "/data")
	addr := envOr("HTTP_ADDR", ":8080")
	cookieSecure := envBool("COOKIE_SECURE", false)

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		slog.Error("create data dir", "err", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(dataDir, "app.db")
	rawDB, err := appdb.Open(dbPath)
	if err != nil {
		slog.Error("open db", "err", err, "path", dbPath)
		os.Exit(1)
	}
	if err := appdb.Migrate(rawDB); err != nil {
		_ = rawDB.Close()
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}
	database := appdb.NewDatabase(rawDB, dbPath)
	defer database.Close()

	usersStore := users.NewStore(database)
	sessionStore := auth.NewSessionStore(database)

	bootstrapCtx, cancelBootstrap := context.WithTimeout(context.Background(), 10*time.Second)
	bootstrapErr := auth.Bootstrap(bootstrapCtx, usersStore, auth.BootstrapConfig{
		Username: os.Getenv("INITIAL_ADMIN_USERNAME"),
		Password: os.Getenv("INITIAL_ADMIN_PASSWORD"),
	})
	cancelBootstrap()
	if bootstrapErr != nil {
		slog.Error("bootstrap admin", "err", bootstrapErr)
		os.Exit(1)
	}

	authSvc := &auth.Service{
		Users:    usersStore,
		Sessions: sessionStore,
		TTL:      sessionTTL,
		Cookie:   auth.CookieConfig{Secure: cookieSecure},
	}
	usersHandlers := users.NewHandlers(usersStore, sessionStore)
	accessStore := access.NewStore(database)
	profilesStore := profiles.NewStore(database)
	profilesHandlers := profiles.NewHandlers(profilesStore, accessStore)
	trackersStore := trackers.NewStore(database)
	trackersHandlers := trackers.NewHandlers(trackersStore, accessStore)
	entriesStore := entries.NewStore(database)
	revisionsStore := entries.NewRevisionStore(database)
	entriesService := entries.NewService(entriesStore, revisionsStore, trackersStore)
	entriesHandlers := entries.NewHandlers(entriesService)
	trackersHandlers.EntryCounter = entriesStore
	accessHandlers := access.NewHandlers(accessStore, trackersStore)

	settingsStore := settings.NewStore(database)
	gate := appdb.NewGate()
	backupSvc, err := backup.NewService(database, gate, settingsStore, dataDir)
	if err != nil {
		slog.Error("backup service", "err", err)
		os.Exit(1)
	}
	if err := backupSvc.Reconcile(context.Background()); err != nil {
		slog.Warn("backup reconcile", "err", err)
	}
	backupHandlers := backup.NewHandlers(backupSvc)
	settingsHandlers := settings.NewHandlers(settingsStore)

	intervalHours := settingsStore.BackupIntervalHours(context.Background())
	scheduler := backup.NewScheduler(backupSvc)
	if err := scheduler.Start(intervalHours); err != nil {
		slog.Error("backup scheduler start", "err", err)
		os.Exit(1)
	}
	defer scheduler.Stop()

	settingsStore.Subscribe(settings.KeyBackupIntervalHours, func(value string) {
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			slog.Warn("scheduler reschedule: invalid interval", "value", value, "err", err)
			return
		}
		if err := scheduler.Reschedule(f); err != nil {
			slog.Warn("scheduler reschedule", "err", err)
			return
		}
		slog.Info("scheduler rescheduled", "interval_hours", f)
	})

	rootCtx, cancelRoot := context.WithCancel(context.Background())
	defer cancelRoot()

	gcDone := make(chan struct{})
	go runSessionGC(rootCtx, sessionStore, gcDone)

	srv := &http.Server{
		Addr: addr,
		Handler: apphttp.NewRouter(apphttp.Deps{
			Auth:         authSvc,
			Users:        usersHandlers,
			Profiles:     profilesHandlers,
			Trackers:     trackersHandlers,
			TrackersStore: trackersStore,
			Entries:      entriesHandlers,
			EntriesStore: entriesStore,
			Access:       accessHandlers,
			AccessStore:  accessStore,
			Backup:       backupHandlers,
			Settings:     settingsHandlers,
			Gate:         gate,
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("http listen", "addr", addr, "data_dir", dataDir, "cookie_secure", cookieSecure)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http listen", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("http shutdown", "err", err)
	}
	cancelRoot()
	<-gcDone
}

func runSessionGC(ctx context.Context, sessions *auth.SessionStore, done chan<- struct{}) {
	defer close(done)
	tick := time.NewTicker(1 * time.Hour)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			gcCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			n, err := sessions.GC(gcCtx)
			cancel()
			if err != nil {
				slog.Warn("session gc", "err", err)
				continue
			}
			if n > 0 {
				slog.Info("session gc", "deleted", n)
			}
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func findDotEnv() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for i := 0; i < 4; i++ {
		candidate := filepath.Join(dir, ".env")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}
