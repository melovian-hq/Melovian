// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"melovian/internal/store"
)

func TestCacheSettingsAndEviction(t *testing.T) {
	srv, db := newTestServer(t)
	instances := store.NewInstanceStore(db)
	inst, err := instances.Create(store.CreateInstanceInput{
		Name:      "Cache",
		ServerURL: "http://example.com",
		Username:  "user",
		Password:  "pass",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	userID := "instance:" + inst.ID
	limitBody := `{"enabled":true,"limitBytes":128}`
	req := httptest.NewRequest(http.MethodPut, "/api/music/settings/cache", strings.NewReader(limitBody))
	req.Header.Set("X-Instance-Id", inst.ID)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put cache settings status %d: %s", rec.Code, rec.Body.String())
	}

	entry := store.DownloadedTrack{
		InstanceID:  inst.ID,
		TrackID:     "old-track",
		Path:        t.TempDir() + "/old.bin",
		ContentType: "audio/mpeg",
		Size:        96,
		CreatedAt:   1,
	}
	if err := srv.downloads.Upsert(entry); err != nil {
		t.Fatalf("Upsert old: %v", err)
	}

	entry = store.DownloadedTrack{
		InstanceID:  inst.ID,
		TrackID:     "new-track",
		Path:        t.TempDir() + "/new.bin",
		ContentType: "audio/mpeg",
		Size:        64,
		CreatedAt:   2,
	}
	if err := srv.downloads.Upsert(entry); err != nil {
		t.Fatalf("Upsert new: %v", err)
	}

	if err := srv.evictDownloads(inst.ID, 100, 0); err != nil {
		t.Fatalf("evictDownloads: %v", err)
	}

	if _, err := srv.downloads.Get(inst.ID, "old-track"); err == nil {
		t.Fatal("expected oldest track to be evicted")
	}
	if _, err := srv.downloads.Get(inst.ID, "new-track"); err != nil {
		t.Fatalf("expected newest track to remain: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/music/settings/cache", nil)
	req.Header.Set("X-Instance-Id", inst.ID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get cache settings status %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode cache settings: %v", err)
	}
	if payload["enabled"] != true {
		t.Fatalf("expected enabled=true, got %v", payload["enabled"])
	}
	if int64(payload["usedBytes"].(float64)) != 64 {
		t.Fatalf("expected usedBytes=64, got %v", payload["usedBytes"])
	}
	_ = userID
}

func TestCreateDownloadRespectsDisabledCache(t *testing.T) {
	srv, db := newTestServer(t)
	instances := store.NewInstanceStore(db)
	inst, err := instances.Create(store.CreateInstanceInput{
		Name:      "CacheOff",
		ServerURL: "http://example.com",
		Username:  "user",
		Password:  "pass",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	body := `{"enabled":false,"limitBytes":5368709120}`
	req := httptest.NewRequest(http.MethodPut, "/api/music/settings/cache", strings.NewReader(body))
	req.Header.Set("X-Instance-Id", inst.ID)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put cache settings status %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/downloads/track-1", nil)
	req.Header.Set("X-Instance-Id", inst.ID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when cache disabled, got %d", rec.Code)
	}
}
