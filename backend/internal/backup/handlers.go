package backup

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/bublya/baby-tracker/backend/internal/httpx"
	"github.com/bublya/baby-tracker/backend/internal/users"
	"github.com/go-chi/chi/v5"
)

const restoreDrainTimeout = 30 * time.Second

type Handlers struct {
	Service *Service
}

func NewHandlers(svc *Service) *Handlers { return &Handlers{Service: svc} }

func (h *Handlers) Mount(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}/download", h.Download)
	r.Post("/{id}/restore", h.Restore)
	r.Delete("/{id}", h.Delete)
}

type snapshotResponse struct {
	ID        int64   `json:"id"`
	Filename  string  `json:"filename"`
	SizeBytes int64   `json:"size_bytes"`
	Type      string  `json:"type"`
	Note      *string `json:"note,omitempty"`
	CreatedBy *int64  `json:"created_by,omitempty"`
	CreatedAt string  `json:"created_at"`
}

func toResp(s *Snapshot) snapshotResponse {
	return snapshotResponse{
		ID:        s.ID,
		Filename:  s.Filename,
		SizeBytes: s.SizeBytes,
		Type:      s.Type,
		Note:      s.Note,
		CreatedBy: s.CreatedBy,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
	}
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := ListFilter{Type: q.Get("type")}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		f.Limit = n
	}
	items, err := h.Service.List(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list snapshots")
		return
	}
	resp := make([]snapshotResponse, 0, len(items))
	for _, s := range items {
		resp = append(resp, toResp(s))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": resp})
}

type createBody struct {
	Note *string `json:"note,omitempty"`
}

func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var body createBody
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
	}
	snap, err := h.Service.Create(r.Context(), TypeManual, body.Note, currentUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create snapshot")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"snapshot": toResp(snap)})
}

func (h *Handlers) Download(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	f, snap, err := h.Service.Open(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrSnapshotNotFound):
			writeError(w, http.StatusNotFound, "snapshot not found")
		case errors.Is(err, ErrFileMissing):
			writeError(w, http.StatusGone, "snapshot file missing")
		default:
			writeError(w, http.StatusInternalServerError, "open snapshot")
		}
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+snap.Filename+"\"")
	w.Header().Set("Content-Length", strconv.FormatInt(snap.SizeBytes, 10))
	_, _ = io.Copy(w, f)
}

func (h *Handlers) Restore(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.Service.Restore(r.Context(), id, currentUserID(r), restoreDrainTimeout); err != nil {
		switch {
		case errors.Is(err, ErrSnapshotNotFound):
			writeError(w, http.StatusNotFound, "snapshot not found")
		case errors.Is(err, ErrFileMissing):
			writeError(w, http.StatusGone, "snapshot file missing")
		case errors.Is(err, ErrRestoreTimeout):
			writeError(w, http.StatusServiceUnavailable, "restore timed out waiting for in-flight requests")
		default:
			writeError(w, http.StatusInternalServerError, "restore snapshot: "+err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.Service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrSnapshotNotFound) {
			writeError(w, http.StatusNotFound, "snapshot not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "delete snapshot")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func currentUserID(r *http.Request) *int64 {
	u := users.FromContext(r.Context())
	if u == nil {
		return nil
	}
	v := u.ID
	return &v
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	httpx.WriteJSON(w, code, v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	httpx.WriteErrorStatus(w, code, msg)
}
