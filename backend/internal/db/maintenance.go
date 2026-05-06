package db

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
)

// Gate prevents new requests from running while maintenance work (e.g. a DB
// restore) is happening. It tracks in-flight HTTP requests via a WaitGroup so
// Enter can wait for them to drain before swapping the pool.
type Gate struct {
	inMaint atomic.Bool
	wg      sync.WaitGroup
}

func NewGate() *Gate { return &Gate{} }

// Active reports whether maintenance mode is currently engaged.
func (g *Gate) Active() bool { return g.inMaint.Load() }

// Middleware rejects new requests with 503 while in maintenance and counts
// in-flight ones via the WaitGroup. /api/health is always allowed through so
// the UI overlay can poll for recovery.
func (g *Gate) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		if g.inMaint.Load() {
			w.Header().Set("Retry-After", "5")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":        "maintenance",
				"message":     "maintenance",
				"error":       "maintenance",
				"maintenance": true,
			})
			return
		}
		g.wg.Add(1)
		defer g.wg.Done()
		next.ServeHTTP(w, r)
	})
}

// ErrAlreadyInMaintenance is returned by Enter if the gate is already engaged.
var ErrAlreadyInMaintenance = errors.New("already in maintenance")

// Enter engages maintenance mode and waits for in-flight requests (other than
// the caller — see PauseSelf) to drain. If ctx is cancelled before drain, the
// gate is released and the error is returned.
func (g *Gate) Enter(ctx context.Context) error {
	if !g.inMaint.CompareAndSwap(false, true) {
		return ErrAlreadyInMaintenance
	}
	done := make(chan struct{})
	go func() {
		g.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		g.inMaint.Store(false)
		return ctx.Err()
	}
}

// Exit releases maintenance mode.
func (g *Gate) Exit() { g.inMaint.Store(false) }

// PauseSelf removes the calling request from the WaitGroup so Enter can
// drain. Must be paired with ResumeSelf before the request returns so the
// middleware's deferred Done balances out. Use this from the handler that
// actually triggers the maintenance work.
func (g *Gate) PauseSelf() { g.wg.Done() }

func (g *Gate) ResumeSelf() { g.wg.Add(1) }
