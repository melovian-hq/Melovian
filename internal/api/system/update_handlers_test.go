// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"context"
	"errors"
	"melovian/internal/appconfig"
	"melovian/internal/store"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

// spawnUpdateJob jobs run on a detached context that stopUpdateJobs
// cancels, so shutdown drains them instead of leaking goroutines.
func TestSpawnUpdateJobCancelledOnStop(t *testing.T) {
	h := newTestHandler(t)
	started := make(chan struct{})
	sawCancel := make(chan struct{})
	h.spawnUpdateJob(func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		close(sawCancel)
	})
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("update job did not start")
	}

	h.stopUpdateJobs()
	select {
	case <-sawCancel:
	case <-time.After(2 * time.Second):
		t.Fatal("update job context was not cancelled")
	}
}

func TestRunUpdateCheckRejectsConcurrent(t *testing.T) {
	h := newTestHandler(t)
	h.upd.mu.Lock()
	h.upd.checking = true
	h.upd.mu.Unlock()

	if _, err := h.runUpdateCheck(context.Background(), ""); !errors.Is(err, errUpdateCheckInFlight) {
		t.Fatalf("expected errUpdateCheckInFlight, got %v", err)
	}
}

// A status read while a check is already running must not spawn another
// background goroutine.
func TestGetUpdateStatusDoesNotStackChecks(t *testing.T) {
	h := newTestHandler(t)
	h.upd.mu.Lock()
	h.upd.checking = true
	h.upd.mu.Unlock()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/update/status", nil)
	mux := http.NewServeMux()
	h.Register(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}

	h.upd.mu.Lock()
	spawned := h.upd.bgCtx != nil
	h.upd.mu.Unlock()
	if spawned {
		t.Fatal("status handler spawned a background check while one was in flight")
	}
}

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	dir := t.TempDir()
	db, err := store.OpenDB(filepath.Join(dir, "test.db"), appconfig.Config{DataDir: dir})
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return New(Deps{
		Config:      appconfig.Config{DataDir: dir},
		Preferences: store.NewPreferencesStore(db),
		ServerFn:    func() *http.Server { return &http.Server{} },
	})
}
