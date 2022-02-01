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
	w.WriteHeader((http.StatusOK))
}

func (p *ProbesHandler) isReady() bool {
	if !p.ready.Load().(bool) {
		return false
	}
	return true
}

func (p *ProbesHandler) readinessHandler(w http.ResponseWriter, _ *http.Request) {
	if p.isReady() {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(503)
	}
}

// MarkReady sets probe server's state as ready
func (p *ProbesHandler) MarkReady(ready bool) {
	p.ready.Store(ready)
}

// Run starts the probe server
func (p *ProbesHandler) Run(ctx context.Context) error {
	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		mux := http.NewServeMux()
		mux.Handle("/health", http.HandlerFunc(p.healthHandler))
		mux.Handle("/ready", http.HandlerFunc(p.readinessHandler))
		server := &http.Server{
			Addr:    p.address,
			Handler: mux,
		}
		return server.ListenAndServe()
	})
	return eg.Wait()
}
