// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"melovian/internal/appconfig"
	"melovian/internal/store"
)

func TestSentryServerSettingsRoundTrip(t *testing.T) {
	srv, _ := newTestServer(t)
	if err := srv.initSentryFromStore(); err != nil {
		t.Fatalf("initSentryFromStore: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/settings/sentry", nil)
	getRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status %d: %s", getRec.Code, getRec.Body.String())
	}

	body := bytes.NewBufferString(`{
		"enabled": true,
		"dsn": "https://deadbeefdeadbeefdeadbeefdeadbeef@glitchtip.example/1",
		"frontendDsn": "",
		"environment": "test",
		"release": "melovian@test",
		"tracesSampleRate": 0.1,
		"clientReportingAllowed": true
	}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/settings/sentry", body)
	putRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("put status %d: %s", putRec.Code, putRec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(putRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	effective, ok := payload["effective"].(map[string]any)
	if !ok || effective["enabled"] != true {
		t.Fatalf("expected effective.enabled true, got %#v", payload["effective"])
	}
	if !srv.effectiveSentryConfig().Enabled() {
		t.Fatal("expected runtime sentry enabled")
	}
}

func TestSentryClientSettingsRoundTrip(t *testing.T) {
	srv, _ := newTestServer(t)

	getReq := httptest.NewRequest(http.MethodGet, "/api/music/settings/sentry-client", nil)
	getRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status %d", getRec.Code)
	}

	putReq := httptest.NewRequest(
		http.MethodPut,
		"/api/music/settings/sentry-client",
		bytes.NewBufferString(`{"enabled":false}`),
	)
	putRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("put status %d: %s", putRec.Code, putRec.Body.String())
	}

	settings, err := srv.loadSentryClientSettings("local")
	if err != nil {
		t.Fatalf("loadSentryClientSettings: %v", err)
	}
	if settings.Enabled {
		t.Fatal("expected client reporting disabled")
	}
}

func TestClientSentryPayloadRequiresClientOptIn(t *testing.T) {
	srv, _ := newTestServer(t)
	stored := appconfig.DefaultStoredSentrySettings()
	stored.Enabled = true
	stored.DSN = "https://deadbeefdeadbeefdeadbeefdeadbeef@glitchtip.example/1"
	if err := srv.refreshSentryRuntime(stored); err != nil {
		t.Fatalf("refreshSentryRuntime: %v", err)
	}
	if err := srv.preferences.Set("local", store.PrefKeySentryClient, `{"enabled":false}`); err != nil {
		t.Fatalf("Set: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("config status %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := payload["sentry"]; ok {
		t.Fatal("expected no sentry payload when client reporting disabled")
	}
}

func TestSentryTestEventWhenDisabled(t *testing.T) {
	srv, _ := newTestServer(t)
	if err := srv.initSentryFromStore(); err != nil {
		t.Fatalf("initSentryFromStore: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/settings/sentry/test", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSentryTestEventWhenEnabled(t *testing.T) {
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(collector.Close)

	host := strings.TrimPrefix(collector.URL, "http://")
	dsn := fmt.Sprintf("http://deadbeefdeadbeefdeadbeefdeadbeef@%s/1", host)

	srv, _ := newTestServer(t)
	stored := appconfig.DefaultStoredSentrySettings()
	stored.Enabled = true
	stored.DSN = dsn
	stored.Environment = "test"
	if err := srv.refreshSentryRuntime(stored); err != nil {
		t.Fatalf("refreshSentryRuntime: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/settings/sentry/test", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok true, got %#v", payload)
	}
	eventID, _ := payload["eventId"].(string)
	if eventID == "" {
		t.Fatalf("expected eventId, got %#v", payload)
	}
}
