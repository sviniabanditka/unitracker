package settings

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/bublya/baby-tracker/backend/internal/httpx"
	"github.com/go-chi/chi/v5"
)

const (
	maxAppNameLen   = 64
	minIntervalHrs  = 0.05
	minRetention    = 1
	maxRetention    = 10000
	maxIntervalHrs  = 24 * 30 // a month is plenty
)

var allowedLocales = map[string]struct{}{"en": {}, "uk": {}}

type Handlers struct{ Store *Store }

func NewHandlers(s *Store) *Handlers { return &Handlers{Store: s} }

// MountAdmin registers GET / and PATCH / under the caller's router (admin-protected).
func (h *Handlers) MountAdmin(r chi.Router) {
	r.Get("/", h.GetAll)
	r.Patch("/", h.Patch)
}

// MountPublic registers GET /info on the caller's router with no auth.
func (h *Handlers) MountPublic(r chi.Router) {
	r.Get("/info", h.PublicInfo)
}

func (h *Handlers) GetAll(w http.ResponseWriter, r *http.Request) {
	items, err := h.Store.All(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load settings")
		return
	}
	out := map[string]string{
		KeyBackupIntervalHours:  strconv.FormatFloat(DefaultBackupIntervalHours, 'f', -1, 64),
		KeyBackupRetentionCount: strconv.Itoa(DefaultBackupRetentionCount),
		KeyAppName:              DefaultAppName,
		KeyDefaultLocale:        DefaultLocaleCode,
	}
	for _, s := range items {
		out[s.Key] = s.Value
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) Patch(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(body) == 0 {
		writeError(w, http.StatusBadRequest, "no fields to update")
		return
	}
	clean, fieldErrs := validate(body)
	if len(fieldErrs) > 0 {
		httpx.WriteValidationError(w, "validation failed", fieldErrs)
		return
	}
	if err := h.Store.SetMany(r.Context(), clean); err != nil {
		writeError(w, http.StatusInternalServerError, "save settings")
		return
	}
	items, err := h.Store.All(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "reload settings")
		return
	}
	out := map[string]string{}
	for _, s := range items {
		out[s.Key] = s.Value
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) PublicInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"app_name":       h.Store.AppName(r.Context()),
		"default_locale": h.Store.DefaultLocale(r.Context()),
	})
}

// validate filters input to known keys and validates value formats.
// Returns the cleaned map and a per-field error map.
func validate(in map[string]string) (map[string]string, map[string]string) {
	clean := map[string]string{}
	errs := map[string]string{}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := strings.TrimSpace(in[k])
		switch k {
		case KeyBackupIntervalHours:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				errs[k] = "must be a number"
				continue
			}
			if f < minIntervalHrs {
				errs[k] = "must be ≥ " + strconv.FormatFloat(minIntervalHrs, 'f', -1, 64)
				continue
			}
			if f > maxIntervalHrs {
				errs[k] = "must be ≤ " + strconv.Itoa(maxIntervalHrs)
				continue
			}
			clean[k] = strconv.FormatFloat(f, 'f', -1, 64)
		case KeyBackupRetentionCount:
			n, err := strconv.Atoi(v)
			if err != nil {
				errs[k] = "must be an integer"
				continue
			}
			if n < minRetention {
				errs[k] = "must be ≥ " + strconv.Itoa(minRetention)
				continue
			}
			if n > maxRetention {
				errs[k] = "must be ≤ " + strconv.Itoa(maxRetention)
				continue
			}
			clean[k] = strconv.Itoa(n)
		case KeyAppName:
			if v == "" {
				errs[k] = "required"
				continue
			}
			if len([]rune(v)) > maxAppNameLen {
				errs[k] = "must be ≤ " + strconv.Itoa(maxAppNameLen) + " characters"
				continue
			}
			clean[k] = v
		case KeyDefaultLocale:
			if _, ok := allowedLocales[v]; !ok {
				errs[k] = "must be one of: en, uk"
				continue
			}
			clean[k] = v
		default:
			errs[k] = "unknown setting"
		}
	}
	return clean, errs
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	httpx.WriteJSON(w, code, v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	httpx.WriteErrorStatus(w, code, msg)
}
