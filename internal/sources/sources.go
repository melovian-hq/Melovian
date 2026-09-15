// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Package sources defines the source-extension contract: the interface a
// music source implements, the registry that holds them, and the
// dispatch, capability, and failure-handling machinery around them.
//
// A source extension serves Subsonic-shaped requests for its instances.
// Bundled sources (subsonic, navidrome) run in process as trusted code;
// third-party source extensions will be adapted to this contract by the
// host runtime.
package sources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"melovian/internal/store"
)

// Capabilities describes optional behaviors a source supports.
type Capabilities struct {
	// SmartPlaylists reports native server-side smart playlist support
	// (Navidrome /api/playlist).
	SmartPlaylists bool `json:"smartPlaylists"`
	// Events reports a server-sent events stream the host can bridge
	// onto the app websocket.
	Events bool `json:"events"`
	// EventsPath is the SSE endpoint relative to the instance URL.
	EventsPath string `json:"eventsPath,omitempty"`
	// EventsAuthMode tells the host how to authenticate the events
	// stream, for example "query-jwt".
	EventsAuthMode string `json:"eventsAuthMode,omitempty"`
}

// Source is a music source backed by an extension. The extension id and
// the source id are the same value so enable state is shared with the
// extension system.
type Source interface {
	// ID is the extension id backing this source, for example "subsonic".
	ID() string
	// DisplayName is the human-facing source name.
	DisplayName() string
	// Caps reports optional behaviors.
	Caps() Capabilities
	// Ping probes an instance and returns the server name and version.
	Ping(ctx context.Context, inst store.SourceInstance) (serverName, version string, err error)
	// ServeREST proxies one Subsonic-shaped request for inst. Implementations
	// must not retain inst or the request past return.
	ServeREST(w http.ResponseWriter, r *http.Request, inst store.SourceInstance, deps Deps)
}

// Errors surfaced to API callers.
var (
	// ErrUnknownSource means no registered source matches the instance.
	ErrUnknownSource = errors.New("unknown source")
	// ErrSourceDisabled means the backing extension is disabled or uninstalled.
	ErrSourceDisabled = errors.New("source extension is disabled")
	// ErrSourceOffline means the failure breaker is open for the instance.
	ErrSourceOffline = errors.New("source temporarily unavailable")
)

// Registry holds the registered sources and per-instance failure breakers.
type Registry struct {
	mu       sync.Mutex
	sources  map[string]Source
	order    []string
	enabled  func(sourceID string) bool
	breakers map[string]*breaker
}

// breaker counts consecutive failures for one instance and opens for a
// cooldown once the threshold is hit, so a dead upstream stops adding
// latency to every request.
type breaker struct {
	failures  int
	openUntil time.Time
}

const (
	breakerThreshold = 5
	breakerCooldown  = 30 * time.Second
)

// New returns an empty registry. enabledFn reports whether the backing
// extension is enabled; nil means every source counts as enabled.
func New(enabledFn func(sourceID string) bool) *Registry {
	return &Registry{
		sources:  map[string]Source{},
		enabled:  enabledFn,
		breakers: map[string]*breaker{},
	}
}

// Register adds a source. Duplicate ids overwrite so tests and dev
// builds can replace bundled sources.
func (r *Registry) Register(s Source) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sources[s.ID()]; !ok {
		r.order = append(r.order, s.ID())
	}
	r.sources[s.ID()] = s
}

// Get returns the source for an id.
func (r *Registry) Get(id string) (Source, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sources[id]
	return s, ok
}

// List returns sources in registration order.
func (r *Registry) List() []Source {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Source, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.sources[id])
	}
	return out
}

// Enabled reports whether the backing extension is enabled.
func (r *Registry) Enabled(sourceID string) bool {
	if r.enabled == nil {
		return true
	}
	return r.enabled(sourceID)
}

// Available resolves the source for an instance and checks that its
// backing extension is enabled.
func (r *Registry) Available(inst store.SourceInstance) (Source, error) {
	src, ok := r.Get(inst.SourceOrDefault())
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownSource, inst.SourceOrDefault())
	}
	if !r.Enabled(src.ID()) {
		return nil, fmt.Errorf("%w: %s", ErrSourceDisabled, src.ID())
	}
	return src, nil
}

// AllowRequest reports whether the breaker for an instance is closed.
func (r *Registry) AllowRequest(instanceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b := r.breakers[instanceID]
	if b == nil {
		return nil
	}
	if b.openUntil.IsZero() || time.Now().After(b.openUntil) {
		return nil
	}
	return ErrSourceOffline
}

// ReportResult records one source call outcome for an instance.
// Success resets the breaker; failures open it after the threshold.
func (r *Registry) ReportResult(instanceID string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b := r.breakers[instanceID]
	if b == nil {
		b = &breaker{}
		r.breakers[instanceID] = b
	}
	if err == nil {
		b.failures = 0
		b.openUntil = time.Time{}
		return
	}
	b.failures++
	if b.failures >= breakerThreshold {
		b.openUntil = time.Now().Add(breakerCooldown)
	}
}

// Offline reports whether the instance breaker is currently open.
func (r *Registry) Offline(instanceID string) bool {
	return r.AllowRequest(instanceID) != nil
}

// SourceInfo is the view of one source for status responses.
type SourceInfo struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Enabled      bool         `json:"enabled"`
	Capabilities Capabilities `json:"capabilities"`
}

// Info returns the status view of every registered source.
func (r *Registry) Info() []SourceInfo {
	out := make([]SourceInfo, 0)
	for _, s := range r.List() {
		out = append(out, SourceInfo{
			ID:           s.ID(),
			Name:         s.DisplayName(),
			Enabled:      r.Enabled(s.ID()),
			Capabilities: s.Caps(),
		})
	}
	return out
}

// PingInstance probes an instance through its source with panic
// recovery and breaker accounting.
func (r *Registry) PingInstance(ctx context.Context, inst store.SourceInstance) (serverName, version string, err error) {
	src, err := r.Available(inst)
	if err != nil {
		return "", "", err
	}
	if err := r.AllowRequest(inst.ID); err != nil {
		return "", "", err
	}
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("source %q panicked: %v", src.ID(), rec)
			serverName, version = "", ""
		}
		r.ReportResult(inst.ID, err)
	}()
	return src.Ping(ctx, inst)
}

// ServeREST dispatches a Subsonic-shaped request to the source backing
// the instance, with enable checks, the failure breaker, and panic
// recovery so a source crash never takes down the app.
func (r *Registry) ServeREST(w http.ResponseWriter, req *http.Request, inst store.SourceInstance, deps Deps) {
	src, err := r.Available(inst)
	if err != nil {
		status := http.StatusServiceUnavailable
		code := "source_unavailable"
		if errors.Is(err, ErrUnknownSource) {
			status = http.StatusBadRequest
			code = "unknown_source"
		}
		http.Error(w, code+": "+err.Error(), status)
		return
	}
	if err := r.AllowRequest(inst.ID); err != nil {
		http.Error(w, "source_unavailable: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	defer func() {
		if rec := recover(); rec != nil {
			r.ReportResult(inst.ID, fmt.Errorf("panic: %v", rec))
			http.Error(w, "source error", http.StatusBadGateway)
			return
		}
		if sw.status >= http.StatusBadGateway {
			r.ReportResult(inst.ID, fmt.Errorf("upstream status %d", sw.status))
		} else {
			r.ReportResult(inst.ID, nil)
		}
	}()
	src.ServeREST(sw, req, inst, deps)
}

// statusWriter records the response status so the breaker can count
// upstream failures without changing how sources write responses.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Flush forwards to the wrapped writer so media streaming is unaffected.
func (s *statusWriter) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// ReadFrom preserves the sendfile fast path the underlying writer has.
func (s *statusWriter) ReadFrom(r io.Reader) (int64, error) {
	if rf, ok := s.ResponseWriter.(io.ReaderFrom); ok {
		return rf.ReadFrom(r)
	}
	return io.Copy(s.ResponseWriter, r)
}

// Unwrap lets http.ResponseController reach the real writer.
func (s *statusWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }
