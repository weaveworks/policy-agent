package probes

import (
	"context"
	"net/http"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

type ProbesHandler struct {
	address string
	ready   *atomic.Value
}

// NewProbesHandler returns a probe server instance
func NewProbesHandler(address string) *ProbesHandler {
	ready := &atomic.Value{}
	ready.Store(false)
	return &ProbesHandler{
		address: address,
		ready:   ready,
	}
}

func (p *ProbesHandler) healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (p *ProbesHandler) isReady() bool {
	if !p.ready.Load().(bool) {
		return false
	}
	return true
}

func (p *ProbesHandler) readinessHandler(w http.ResponseWriter, _ *http.Request) {
	if p.isReady() {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

// MarkReady sets probe server's state as ready
func (p *ProbesHandler) MarkReady(ready bool) {
	p.ready.Store(ready)
}

// Run starts the probe server
func (p *ProbesHandler) Run(_ context.Context) error {
	eg := &errgroup.Group{}
	eg.Go(func() error {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", p.healthHandler)
		mux.HandleFunc("/ready", p.readinessHandler)
		server := &http.Server{
			Addr:    p.address,
			Handler: mux,
		}
		return server.ListenAndServe()
	})
	return eg.Wait()
}
