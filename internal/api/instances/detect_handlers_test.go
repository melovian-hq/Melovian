// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package instances

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// stubDetectCandidates swaps the probe list and clears the result cache for
// the duration of a test.
func stubDetectCandidates(t *testing.T, urls []string) {
	t.Helper()
	prev := detectCandidateURLs
	detectCandidateURLs = func() []string { return urls }
	resetDetectCache()
	t.Cleanup(func() {
		detectCandidateURLs = prev
		resetDetectCache()
	})
}

func resetDetectCache() {
	detectResultCache.mu.Lock()
	defer detectResultCache.mu.Unlock()
	detectResultCache.servers = nil
	detectResultCache.at = time.Time{}
}

func navidromeAuthFailure(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"subsonic-response":{"status":"failed","version":"1.16.1","type":"navidrome","serverVersion":"0.55.2","openSubsonic":true,"error":{"code":40,"message":"Wrong username or password"}}}`))
}

func TestDetectReportsNavidromeOnAuthFailure(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(navidromeAuthFailure))
	defer upstream.Close()
	stubDetectCandidates(t, []string{upstream.URL})

	servers := detectSubsonicServers(context.Background())
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	server := servers[0]
	if !server.Reachable {
		t.Fatal("expected reachable server")
	}
	if server.URL != upstream.URL {
		t.Fatalf("url %q", server.URL)
	}
	if server.ServerName != "navidrome" {
		t.Fatalf("serverName %q", server.ServerName)
	}
	if server.Version != "0.55.2" {
		t.Fatalf("version %q", server.Version)
	}
}

func TestDetectReportsHealthySubsonic(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/ping.view" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("f") != "json" {
			t.Fatalf("expected f=json, got %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"subsonic-response":{"status":"ok","version":"1.16.1","type":"gonic","serverVersion":"0.16.0"}}`))
	}))
	defer upstream.Close()
	stubDetectCandidates(t, []string{upstream.URL})

	servers := detectSubsonicServers(context.Background())
	if len(servers) != 1 || servers[0].ServerName != "gonic" {
		t.Fatalf("servers %+v", servers)
	}
}

func TestDetectSkipsNonSubsonicAndDeadEndpoints(t *testing.T) {
	garbage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("this is not json"))
	}))
	defer garbage.Close()
	plainJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hello":"world"}`))
	}))
	defer plainJSON.Close()
	// Nothing listens on this address, so the probe gets refused fast.
	dead := "http://127.0.0.1:1"

	stubDetectCandidates(t, []string{garbage.URL, dead, plainJSON.URL})

	servers := detectSubsonicServers(context.Background())
	if len(servers) != 0 {
		t.Fatalf("expected no servers, got %+v", servers)
	}
}

func TestDetectKeepsCandidateOrder(t *testing.T) {
	first := httptest.NewServer(http.HandlerFunc(navidromeAuthFailure))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"subsonic-response":{"status":"ok","version":"1.16.1"}}`))
	}))
	defer second.Close()
	stubDetectCandidates(t, []string{second.URL, "http://127.0.0.1:1", first.URL})

	servers := detectSubsonicServers(context.Background())
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %+v", servers)
	}
	if servers[0].URL != second.URL || servers[1].URL != first.URL {
		t.Fatalf("order violated: %+v", servers)
	}
	// The fallback name applies when the envelope carries no identity.
	if servers[0].ServerName != "Subsonic" {
		t.Fatalf("serverName %q", servers[0].ServerName)
	}
}

func TestDetectCachesResults(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		navidromeAuthFailure(w, r)
	}))
	defer upstream.Close()
	stubDetectCandidates(t, []string{upstream.URL})

	if got := detectSubsonicServers(context.Background()); len(got) != 1 {
		t.Fatalf("first probe: %+v", got)
	}
	if got := detectSubsonicServers(context.Background()); len(got) != 1 {
		t.Fatalf("cached probe: %+v", got)
	}
	if calls != 1 {
		t.Fatalf("expected 1 upstream call, got %d", calls)
	}
}

func TestDetectDoesNotCacheCancelledProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		cancel() // cancel while the probe is in flight
		navidromeAuthFailure(w, r)
	}))
	defer upstream.Close()
	stubDetectCandidates(t, []string{upstream.URL})

	_ = detectSubsonicServers(ctx)
	// The cancelled probe must not poison the cache: the next live request
	// probes again instead of serving the empty result for 30 seconds.
	if got := detectSubsonicServers(context.Background()); len(got) != 1 {
		t.Fatalf("expected fresh probe after cancelled call, got %+v", got)
	}
	if calls != 2 {
		t.Fatalf("expected the live request to probe again, got %d calls", calls)
	}
}

func TestDetectSkipsAlreadyCancelledRequest(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		navidromeAuthFailure(w, r)
	}))
	defer upstream.Close()
	stubDetectCandidates(t, []string{upstream.URL})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := detectSubsonicServers(ctx); len(got) != 0 {
		t.Fatalf("cancelled request should return nothing, got %+v", got)
	}
	if calls != 0 {
		t.Fatalf("cancelled request should not probe, got %d calls", calls)
	}
}

func TestHandleDetectInstancesShape(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(navidromeAuthFailure))
	defer upstream.Close()
	stubDetectCandidates(t, []string{upstream.URL})

	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/api/instances/detect", nil)
	rec := httptest.NewRecorder()
	h.handleDetectInstances(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Servers []detectedServer `json:"servers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Servers) != 1 || payload.Servers[0].ServerName != "navidrome" {
		t.Fatalf("servers %+v", payload.Servers)
	}
}

func TestHandleDetectInstancesEmptyIsArray(t *testing.T) {
	stubDetectCandidates(t, []string{"http://127.0.0.1:1"})

	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/api/instances/detect", nil)
	rec := httptest.NewRecorder()
	h.handleDetectInstances(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Servers []detectedServer `json:"servers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Servers == nil {
		t.Fatal("servers should serialize as an empty array, not null")
	}
}

func TestExtraDetectCandidatesValidation(t *testing.T) {
	t.Setenv("MELOVIAN_DETECT_EXTRA_URLS", "http://nas.local:4533, gopher://x, http://user:pass@evil.local, ,https://box:4040/")
	got := extraDetectCandidates()
	want := []string{"http://nas.local:4533", "https://box:4040"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}
