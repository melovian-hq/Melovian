// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"melovian/internal/store"
)

func TestUpsertListenedMsSmoke(t *testing.T) {
	srv, db := newTestServer(t)

	body := bytes.NewBufferString(`{
		"positionMs":8000,
		"deltaMs":8000,
		"durationMs":180000,
		"trackTitle":"Smoke Track",
		"artistName":"Artist"
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/music/items/smoke-1", body)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upsert status %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	listened, ok := payload["listenedMs"].(float64)
	if !ok {
		t.Fatalf("listenedMs missing in response: %v", payload)
	}
	if int64(listened) != 8000 {
		t.Fatalf("expected listenedMs 8000, got %v", listened)
	}

	listen := store.NewListenStore(db)
	item, err := listen.Get("local", "smoke-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if item.ListenedMs != 8000 {
		t.Fatalf("disk listened_ms = %d, want 8000", item.ListenedMs)
	}
}

func TestUpsertListenedMsAdversarialHugeDelta(t *testing.T) {
	srv, db := newTestServer(t)

	body := bytes.NewBufferString(`{
		"positionMs":1,
		"deltaMs":999999999,
		"durationMs":30000,
		"trackTitle":"Adv"
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/music/items/adv-1", body)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upsert status %d: %s", rec.Code, rec.Body.String())
	}

	listen := store.NewListenStore(db)
	item, err := listen.Get("local", "adv-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if item.ListenedMs > 15_000 {
		t.Fatalf("huge delta should clamp, got %d", item.ListenedMs)
	}
}

func TestUpsertListenedMsNegativeDeltaIgnored(t *testing.T) {
	srv, db := newTestServer(t)
	listen := store.NewListenStore(db)

	seed := bytes.NewBufferString(`{"positionMs":5000,"deltaMs":5000,"durationMs":180000}`)
	req := httptest.NewRequest(http.MethodPut, "/api/music/items/neg-1", seed)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("seed status %d", rec.Code)
	}

	neg := bytes.NewBufferString(`{"positionMs":1000,"deltaMs":-4000,"durationMs":180000}`)
	req = httptest.NewRequest(http.MethodPut, "/api/music/items/neg-1", neg)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("neg status %d", rec.Code)
	}

	item, err := listen.Get("local", "neg-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if item.ListenedMs != 5000 {
		t.Fatalf("negative delta must not reduce listened_ms, got %d", item.ListenedMs)
	}
}
