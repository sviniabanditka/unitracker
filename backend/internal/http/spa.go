package http

import (
	"embed"
	"errors"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

type spaHandler struct {
	files     fs.FS
	indexHTML []byte
}

func newSPAHandler() (*spaHandler, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	idx, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		return nil, err
	}
	return &spaHandler{files: sub, indexHTML: idx}, nil
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	clean := strings.TrimPrefix(r.URL.Path, "/")
	if clean == "" {
		h.serveIndex(w)
		return
	}

	f, err := h.files.Open(clean)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			h.serveIndex(w)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stat, err := f.Stat()
	_ = f.Close()
	if err != nil || stat.IsDir() {
		h.serveIndex(w)
		return
	}

	http.FileServer(http.FS(h.files)).ServeHTTP(w, r)
}

func (h *spaHandler) serveIndex(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(h.indexHTML)
}
