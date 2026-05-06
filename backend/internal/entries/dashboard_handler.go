package entries

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// MountDashboard registers GET / on the caller's router (auth-protected).
// Path resolution: /api/dashboard.
func (h *Handlers) MountDashboard(r chi.Router) {
	r.Get("/", h.Dashboard)
}

type dashboardTodayItem struct {
	TrackerID    int64         `json:"tracker_id"`
	TrackerName  string        `json:"tracker_name"`
	TrackerIcon  *string       `json:"tracker_icon,omitempty"`
	TrackerColor *string       `json:"tracker_color,omitempty"`
	Entry        entryResponse `json:"entry"`
}

type dashboardLastItem struct {
	TrackerID    int64          `json:"tracker_id"`
	TrackerName  string         `json:"tracker_name"`
	TrackerIcon  *string        `json:"tracker_icon,omitempty"`
	TrackerColor *string        `json:"tracker_color,omitempty"`
	Entry        *entryResponse `json:"entry"`
}

type dashboardResponse struct {
	Date           string               `json:"date"`
	Today          []dashboardTodayItem `json:"today"`
	LastPerTracker []dashboardLastItem  `json:"last_per_tracker"`
	TopTrackers    []int64              `json:"top_trackers"`
}

// Dashboard returns a one-shot summary used by the home view.
//
// Query params:
//
//	profile_id (required) — int, scopes the dashboard to a single profile.
//	date       (optional) — YYYY-MM-DD, day for the "today" timeline. Defaults to UTC today.
func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	pidStr := q.Get("profile_id")
	if pidStr == "" {
		writeError(w, http.StatusBadRequest, "profile_id is required")
		return
	}
	pid, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid profile_id")
		return
	}
	profileID := &pid

	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if v := q.Get("date"); v != "" {
		d, err := time.Parse("2006-01-02", v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid date (want YYYY-MM-DD)")
			return
		}
		dayStart = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	}
	dayEnd := dayStart.Add(24 * time.Hour)

	store := h.Service.Store

	// Today timeline.
	todayEntries, err := store.ListInRange(r.Context(), profileID, dayStart, dayEnd)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load today entries")
		return
	}

	// Trackers list — only those belonging to the active profile.
	trackerList, err := h.Service.Trackers.ListByProfile(r.Context(), pid, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load trackers")
		return
	}
	byID := map[int64]struct {
		Name  string
		Icon  *string
		Color *string
	}{}
	for _, t := range trackerList {
		byID[t.ID] = struct {
			Name  string
			Icon  *string
			Color *string
		}{Name: t.Name, Icon: t.Icon, Color: t.Color}
	}

	today := make([]dashboardTodayItem, 0, len(todayEntries))
	for _, e := range todayEntries {
		meta := byID[e.TrackerID]
		today = append(today, dashboardTodayItem{
			TrackerID:    e.TrackerID,
			TrackerName:  meta.Name,
			TrackerIcon:  meta.Icon,
			TrackerColor: meta.Color,
			Entry:        toResp(e),
		})
	}

	// Last entry per tracker.
	last := make([]dashboardLastItem, 0, len(trackerList))
	for _, t := range trackerList {
		e, err := store.LastByTracker(r.Context(), t.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "load last entry")
			return
		}
		item := dashboardLastItem{
			TrackerID:    t.ID,
			TrackerName:  t.Name,
			TrackerIcon:  t.Icon,
			TrackerColor: t.Color,
		}
		if e != nil {
			er := toResp(e)
			item.Entry = &er
		}
		last = append(last, item)
	}

	// Top trackers over the trailing 7 days from dayEnd.
	weekStart := dayEnd.Add(-7 * 24 * time.Hour)
	top, err := store.TopTrackerIDsSince(r.Context(), profileID, weekStart, 3)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load top trackers")
		return
	}
	if top == nil {
		top = []int64{}
	}

	writeJSON(w, http.StatusOK, dashboardResponse{
		Date:           dayStart.Format("2006-01-02"),
		Today:          today,
		LastPerTracker: last,
		TopTrackers:    top,
	})
}
