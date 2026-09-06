// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"melovian/internal/store"
)

func TestDownloadAndStreamTrack(t *testing.T) {
	const audio = "FAKE-AUDIO-BYTES"
	subsonicSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "stream") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = io.WriteString(w, audio)
	}))
	defer subsonicSrv.Close()
	srv, db := newTestServer(t)
	instances := store.NewInstanceStore(db)
	inst, err := instances.Create(store.CreateInstanceInput{
		Name:      "Downloads",
		ServerURL: subsonicSrv.URL,
		Username:  "user",
		Password:  "pass",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := instances.SetActive(inst.ID); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if err := srv.reloadActiveSubsonic(); err != nil {
		t.Fatalf("reloadActiveSubsonic: %v", err)
	}

	// Download the track.
	req := httptest.NewRequest(http.MethodPost, "/api/downloads/track-9?title=Song&artist=Band", nil)
	req.Header.Set("X-Instance-Id", inst.ID)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("download status %d: %s", rec.Code, rec.Body.String())
	}

	// List should include the cached track.
	req = httptest.NewRequest(http.MethodGet, "/api/downloads", nil)
	req.Header.Set("X-Instance-Id", inst.ID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "track-9") {
		t.Fatalf("list missing track: %d %s", rec.Code, rec.Body.String())
	}

	// Stream should serve the cached bytes from disk.
	req = httptest.NewRequest(http.MethodGet, "/api/downloads/track-9/stream", nil)
	req.Header.Set("X-Instance-Id", inst.ID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("stream status %d", rec.Code)
	}
	if got := rec.Body.String(); got != audio {
		t.Fatalf("expected cached audio %q, got %q", audio, got)
	}

	// Delete and confirm the stream is gone.
	req = httptest.NewRequest(http.MethodDelete, "/api/downloads/track-9", nil)
	req.Header.Set("X-Instance-Id", inst.ID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/downloads/track-9/stream", nil)
	req.Header.Set("X-Instance-Id", inst.ID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rec.Code)
	}
}

func TestExportDownloadUsesReadableFilename(t *testing.T) {
	const audio = "FAKE-AUDIO-BYTES"
	subsonicSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "stream") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = io.WriteString(w, audio)
	}))
	defer subsonicSrv.Close()
	srv, db := newTestServer(t)
	instances := store.NewInstanceStore(db)
	inst, err := instances.Create(store.CreateInstanceInput{
		Name:      "Downloads",
		ServerURL: subsonicSrv.URL,
		Username:  "user",
		Password:  "pass",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := instances.SetActive(inst.ID); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if err := srv.reloadActiveSubsonic(); err != nil {
		t.Fatalf("reloadActiveSubsonic: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/downloads/track-9?title=Song&artist=Band", nil)
	req.Header.Set("X-Instance-Id", inst.ID)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("download status %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/downloads/track-9/export", nil)
	req.Header.Set("X-Instance-Id", inst.ID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("export status %d", rec.Code)
	}
	disposition := rec.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "Band - Song.mp3") {
		t.Fatalf("expected readable filename, got %q", disposition)
	}
	if got := rec.Body.String(); got != audio {
		t.Fatalf("expected cached audio %q, got %q", audio, got)
	}
}

func TestCreateDownloadRejectsLocalTrack(t *testing.T) {
	srv, db := newTestServer(t)
	instances := store.NewInstanceStore(db)
	inst, err := instances.Create(store.CreateInstanceInput{
		Name:      "Downloads",
		ServerURL: "http://example.com",
		Username:  "user",
		Password:  "pass",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := instances.SetActive(inst.ID); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/downloads/trk_abc", nil)
	req.Header.Set("X-Instance-Id", inst.ID)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for local track, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateDownloadSerializesSameTrack(t *testing.T) {
	const audio = "FAKE-AUDIO-BYTES"
	started := make(chan struct{})
	subsonicSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "stream") {
			http.NotFound(w, r)
			return
		}
		<-started
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = io.WriteString(w, audio)
	}))
	defer subsonicSrv.Close()
	srv, db := newTestServer(t)
	instances := store.NewInstanceStore(db)
	inst, err := instances.Create(store.CreateInstanceInput{
		Name:      "Downloads",
		ServerURL: subsonicSrv.URL,
		Username:  "user",
		Password:  "pass",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := instances.SetActive(inst.ID); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if err := srv.reloadActiveSubsonic(); err != nil {
		t.Fatalf("reloadActiveSubsonic: %v", err)
	}

	const workers = 8
	codes := make(chan int, workers)
	for range workers {
		go func() {
			req := httptest.NewRequest(http.MethodPost, "/api/downloads/track-race?title=Song&artist=Band", nil)
			req.Header.Set("X-Instance-Id", inst.ID)
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
			codes <- rec.Code
		}()
	}
	close(started)
	for range workers {
		code := <-codes
		if code != http.StatusOK {
			t.Fatalf("download status %d", code)
		}
	}

	items, err := srv.downloads.List(inst.ID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 cached track, got %d", len(items))
	}
	if items[0].Size != int64(len(audio)) {
		t.Fatalf("cached size %d want %d", items[0].Size, len(audio))
	}
}
