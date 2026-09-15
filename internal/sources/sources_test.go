// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"melovian/internal/store"
)

type fakeSource struct {
	id        string
	caps      Capabilities
	pingErr   error
	serveBody string
	status    int
	panics    bool
	pings     int
	serves    int
}

func (f *fakeSource) ID() string          { return f.id }
func (f *fakeSource) DisplayName() string { return f.id }
func (f *fakeSource) Caps() Capabilities  { return f.caps }

func (f *fakeSource) Ping(ctx context.Context, inst store.SourceInstance) (string, string, error) {
	f.pings++
	if f.panics {
		panic("boom")
	}
	if f.pingErr != nil {
		return "", "", f.pingErr
	}
	return "TestServer", "1.0", nil
}

func (f *fakeSource) ServeREST(w http.ResponseWriter, r *http.Request, inst store.SourceInstance, deps Deps) {
	f.serves++
	if f.panics {
		panic("boom")
	}
	if f.status != 0 {
		w.WriteHeader(f.status)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(f.serveBody))
}

func testInstance(sourceID string) store.SourceInstance {
	return store.SourceInstance{
		ID:        "inst-1",
		ServerURL: "http://music.example.com",
		Username:  "u",
		Password:  "p",
		SourceID:  sourceID,
	}
}

func TestRegistryGetAndList(t *testing.T) {
	reg := New(nil)
	sub := &fakeSource{id: "subsonic"}
	nav := &fakeSource{id: "navidrome"}
	reg.Register(sub)
	reg.Register(nav)

	if got, ok := reg.Get("subsonic"); !ok || got != sub {
		t.Fatal("expected subsonic source")
	}
	if _, ok := reg.Get("missing"); ok {
		t.Fatal("expected miss for unknown source")
	}
	list := reg.List()
	if len(list) != 2 || list[0].ID() != "subsonic" || list[1].ID() != "navidrome" {
		t.Fatalf("unexpected list order: %#v", list)
	}
}

func TestRegistryEnabledFromExtensionState(t *testing.T) {
	disabled := map[string]bool{"navidrome": true}
	reg := New(func(id string) bool { return !disabled[id] })
	reg.Register(&fakeSource{id: "subsonic"})
	reg.Register(&fakeSource{id: "navidrome"})

	if !reg.Enabled("subsonic") {
		t.Fatal("subsonic should be enabled")
	}
	if reg.Enabled("navidrome") {
		t.Fatal("navidrome should be disabled")
	}
	if _, err := reg.Available(testInstance("navidrome")); !errors.Is(err, ErrSourceDisabled) {
		t.Fatalf("expected ErrSourceDisabled, got %v", err)
	}
	if _, err := reg.Available(testInstance("bogus")); !errors.Is(err, ErrUnknownSource) {
		t.Fatalf("expected ErrUnknownSource, got %v", err)
	}
}

func TestRegistryNilEnabledFnTreatsAllEnabled(t *testing.T) {
	reg := New(nil)
	reg.Register(&fakeSource{id: "subsonic"})
	if _, err := reg.Available(testInstance("subsonic")); err != nil {
		t.Fatalf("expected available, got %v", err)
	}
}

func TestBreakerOpensAfterThreshold(t *testing.T) {
	reg := New(nil)
	reg.Register(&fakeSource{id: "subsonic", pingErr: errors.New("down")})
	inst := testInstance("subsonic")

	for i := 0; i < breakerThreshold; i++ {
		if _, _, err := reg.PingInstance(context.Background(), inst); err == nil {
			t.Fatal("expected ping error")
		}
	}
	if !reg.Offline(inst.ID) {
		t.Fatal("breaker should be open")
	}
	if _, _, err := reg.PingInstance(context.Background(), inst); !errors.Is(err, ErrSourceOffline) {
		t.Fatalf("expected ErrSourceOffline, got %v", err)
	}
}

func TestBreakerResetsOnSuccess(t *testing.T) {
	reg := New(nil)
	src := &fakeSource{id: "subsonic", pingErr: errors.New("down")}
	reg.Register(src)
	inst := testInstance("subsonic")

	for i := 0; i < breakerThreshold-1; i++ {
		_, _, _ = reg.PingInstance(context.Background(), inst)
	}
	src.pingErr = nil
	if _, _, err := reg.PingInstance(context.Background(), inst); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	src.pingErr = errors.New("down")
	if _, _, err := reg.PingInstance(context.Background(), inst); err == nil {
		t.Fatal("expected ping error")
	}
	if reg.Offline(inst.ID) {
		t.Fatal("breaker should have reset on success")
	}
}

func TestServeRESTDispatch(t *testing.T) {
	reg := New(nil)
	src := &fakeSource{id: "subsonic", serveBody: "ok"}
	reg.Register(src)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/ping.view", nil)
	reg.ServeREST(rec, req, testInstance("subsonic"), Deps{})

	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if src.serves != 1 {
		t.Fatalf("expected 1 serve, got %d", src.serves)
	}
}

func TestServeRESTUnknownSource(t *testing.T) {
	reg := New(nil)
	reg.Register(&fakeSource{id: "subsonic"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/ping.view", nil)
	reg.ServeREST(rec, req, testInstance("bogus"), Deps{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "unknown_source") {
		t.Fatalf("expected unknown_source body, got %q", rec.Body.String())
	}
}

func TestServeRESTDisabledSource(t *testing.T) {
	reg := New(func(id string) bool { return false })
	reg.Register(&fakeSource{id: "subsonic"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/ping.view", nil)
	reg.ServeREST(rec, req, testInstance("subsonic"), Deps{})

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", rec.Code)
	}
}

func TestServeRESTBadGatewayFeedsBreaker(t *testing.T) {
	reg := New(nil)
	reg.Register(&fakeSource{id: "subsonic", status: http.StatusBadGateway})
	inst := testInstance("subsonic")

	for i := 0; i < breakerThreshold; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/ping.view", nil)
		reg.ServeREST(rec, req, inst, Deps{})
	}
	if !reg.Offline(inst.ID) {
		t.Fatal("breaker should open after repeated 502s")
	}
}

func TestServeRESTSuccessResetsBreaker(t *testing.T) {
	reg := New(nil)
	src := &fakeSource{id: "subsonic", status: http.StatusBadGateway}
	reg.Register(src)
	inst := testInstance("subsonic")

	for i := 0; i < breakerThreshold-1; i++ {
		reg.ServeREST(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil), inst, Deps{})
	}
	src.status = 0
	src.serveBody = "ok"
	reg.ServeREST(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil), inst, Deps{})
	src.status = http.StatusBadGateway
	reg.ServeREST(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil), inst, Deps{})
	if reg.Offline(inst.ID) {
		t.Fatal("breaker should have reset on success")
	}
}

func TestServeRESTPanicContained(t *testing.T) {
	reg := New(nil)
	src := &fakeSource{id: "subsonic", panics: true}
	reg.Register(src)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/ping.view", nil)
	reg.ServeREST(rec, req, testInstance("subsonic"), Deps{})

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("got %d, want 502", rec.Code)
	}
}

func TestPingPanicContained(t *testing.T) {
	reg := New(nil)
	reg.Register(&fakeSource{id: "subsonic", panics: true})

	_, _, err := reg.PingInstance(context.Background(), testInstance("subsonic"))
	if err == nil || !strings.Contains(err.Error(), "panicked") {
		t.Fatalf("expected panic error, got %v", err)
	}
}

func TestInfoReportsCapabilities(t *testing.T) {
	reg := New(func(id string) bool { return id != "navidrome" })
	reg.Register(&fakeSource{id: "subsonic"})
	reg.Register(&fakeSource{id: "navidrome", caps: Capabilities{SmartPlaylists: true, Events: true}})

	info := reg.Info()
	if len(info) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(info))
	}
	if !info[0].Enabled {
		t.Fatal("subsonic should report enabled")
	}
	if info[1].Enabled {
		t.Fatal("navidrome should report disabled")
	}
	if !info[1].Capabilities.SmartPlaylists || !info[1].Capabilities.Events {
		t.Fatal("navidrome capabilities missing")
	}
}

func TestDefaultRegistryHasBundledSources(t *testing.T) {
	reg := DefaultRegistry(nil)
	if _, ok := reg.Get("subsonic"); !ok {
		t.Fatal("subsonic source missing")
	}
	nav, ok := reg.Get("navidrome")
	if !ok {
		t.Fatal("navidrome source missing")
	}
	if !nav.Caps().SmartPlaylists || !nav.Caps().Events {
		t.Fatal("navidrome should declare smart playlists and events")
	}
	if nav.Caps().EventsPath != "/api/events" {
		t.Fatalf("unexpected events path %q", nav.Caps().EventsPath)
	}
}

func TestBreakerCooldownExpires(t *testing.T) {
	reg := New(nil)
	reg.Register(&fakeSource{id: "subsonic", pingErr: errors.New("down")})
	inst := testInstance("subsonic")

	for i := 0; i < breakerThreshold; i++ {
		_, _, _ = reg.PingInstance(context.Background(), inst)
	}
	reg.mu.Lock()
	reg.breakers[inst.ID].openUntil = time.Now().Add(-time.Second)
	reg.mu.Unlock()

	if reg.Offline(inst.ID) {
		t.Fatal("breaker should allow requests after cooldown")
	}
}
