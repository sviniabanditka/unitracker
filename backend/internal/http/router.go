package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/bublya/baby-tracker/backend/internal/access"
	"github.com/bublya/baby-tracker/backend/internal/auth"
	"github.com/bublya/baby-tracker/backend/internal/backup"
	appdb "github.com/bublya/baby-tracker/backend/internal/db"
	"github.com/bublya/baby-tracker/backend/internal/entries"
	"github.com/bublya/baby-tracker/backend/internal/httpx"
	"github.com/bublya/baby-tracker/backend/internal/profiles"
	"github.com/bublya/baby-tracker/backend/internal/settings"
	"github.com/bublya/baby-tracker/backend/internal/trackers"
	"github.com/bublya/baby-tracker/backend/internal/users"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Deps struct {
	Auth          *auth.Service
	Users         *users.Handlers
	Profiles      *profiles.Handlers
	Trackers      *trackers.Handlers
	TrackersStore *trackers.Store
	Entries       *entries.Handlers
	EntriesStore  *entries.Store
	Access        *access.Handlers
	AccessStore   *access.Store
	Backup        *backup.Handlers
	Settings      *settings.Handlers
	Gate          *appdb.Gate
}

// requireEntryAccess gates /api/entries/{idParam}/... by looking up the entry's
// tracker → profile_id → user access.
func requireEntryAccess(accessStore *access.Store, entriesStore *entries.Store, idParam string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := strconv.ParseInt(chi.URLParam(r, idParam), 10, 64)
			if err != nil {
				httpx.WriteErrorStatus(w, http.StatusNotFound, "not found")
				return
			}
			u := users.FromContext(r.Context())
			if u == nil {
				httpx.WriteErrorStatus(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			pid, err := entriesStore.GetProfileIDForEntry(r.Context(), id)
			if err != nil {
				httpx.WriteErrorStatus(w, http.StatusNotFound, "not found")
				return
			}
			ok, err := accessStore.HasAccess(r.Context(), u, pid)
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

// keep context import alive (referenced indirectly via handler closures).
var _ = context.Background

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	if deps.Gate != nil {
		r.Use(deps.Gate.Middleware)
	}

	r.Get("/api/health", healthHandler(deps.Gate))

	if deps.Settings != nil {
		r.Route("/api/public", func(pr chi.Router) {
			deps.Settings.MountPublic(pr)
		})
	}

	r.Route("/api/auth", func(ar chi.Router) {
		deps.Auth.Mount(ar)
	})

	// Profiles
	r.Route("/api/profiles", func(pr chi.Router) {
		pr.Use(deps.Auth.RequireUser)

		// List + write at the collection level.
		pr.Get("/", deps.Profiles.List)
		pr.Group(func(ar chi.Router) {
			ar.Use(deps.Auth.RequireAdmin)
			ar.Post("/", deps.Profiles.Create)
			ar.Patch("/{id}", deps.Profiles.Update)
			ar.Delete("/{id}", deps.Profiles.Delete)
		})

		// Profile-scoped trackers (gated by access.RequireProfileAccess).
		pr.Route("/{pid}/trackers", func(tr chi.Router) {
			tr.Use(access.RequireProfileAccess(deps.AccessStore, "pid"))
			tr.Get("/", deps.Trackers.ListByProfile)
			tr.Group(func(ar chi.Router) {
				ar.Use(deps.Auth.RequireAdmin)
				ar.Post("/", deps.Trackers.CreateInProfile)
			})
		})
	})

	// Trackers (library + per-tracker routes).
	r.Route("/api/trackers", func(tr chi.Router) {
		tr.Use(deps.Auth.RequireUser)

		tr.Get("/library", deps.Trackers.Library)

		tr.Route("/{tid}", func(ttr chi.Router) {
			ttr.Use(access.RequireTrackerAccess(deps.AccessStore, deps.TrackersStore, "tid"))
			ttr.Get("/", deps.Trackers.Get)
			ttr.Group(func(ar chi.Router) {
				ar.Use(deps.Auth.RequireAdmin)
				ar.Patch("/", deps.Trackers.Update)
				ar.Post("/archive", deps.Trackers.ToggleArchive)
				ar.Delete("/", deps.Trackers.Delete)
			})
			ttr.Route("/entries", func(er chi.Router) {
				deps.Entries.MountTrackerScoped(er)
			})
			ttr.Route("/stats", func(sr chi.Router) {
				deps.Entries.MountTrackerStats(sr)
			})
		})
	})

	// Entry-id-scoped routes (under /api/entries/{id}/...).
	r.Route("/api/entries", func(er chi.Router) {
		er.Use(deps.Auth.RequireUser)
		er.Route("/{id}", func(eir chi.Router) {
			eir.Use(requireEntryAccess(deps.AccessStore, deps.EntriesStore, "id"))
			deps.Entries.MountEntryByID(eir)
		})
	})

	// Dashboard.
	r.Route("/api/dashboard", func(dr chi.Router) {
		dr.Use(deps.Auth.RequireUser)
		dr.Use(access.RequireProfileAccessFromQuery(deps.AccessStore, "profile_id"))
		deps.Entries.MountDashboard(dr)
	})

	// Admin: users + access management.
	r.Route("/api/admin", func(ar chi.Router) {
		ar.Use(deps.Auth.RequireAdmin)
		ar.Route("/users", func(ur chi.Router) {
			deps.Users.Mount(ur)
			ur.Route("/{uid}/profiles", func(uar chi.Router) {
				deps.Access.MountAdminUserAccess(uar)
			})
		})
		if deps.Backup != nil {
			ar.Route("/snapshots", func(sr chi.Router) {
				deps.Backup.Mount(sr)
			})
		}
		if deps.Settings != nil {
			ar.Route("/settings", func(sr chi.Router) {
				deps.Settings.MountAdmin(sr)
			})
		}
	})

	r.HandleFunc("/api/*", apiNotFound)

	spa, err := newSPAHandler()
	if err != nil {
		panic(err)
	}
	r.NotFound(spa.ServeHTTP)

	return r
}

func healthHandler(gate *appdb.Gate) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"status":      "ok",
			"version":     version,
			"time":        time.Now().UTC().Format(time.RFC3339),
			"maintenance": gate != nil && gate.Active(),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func apiNotFound(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteErrorStatus(w, http.StatusNotFound, "not found")
}

// version is overridden at build time via -ldflags="-X .../http.version=<sha>"
var version = "dev"
