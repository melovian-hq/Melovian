// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestAudioServiceUnavailableWithoutNativeBackend(t *testing.T) {
	svc := NewAudioService()
	caps := svc.Capabilities()
	if caps.NativeAvailable {
		t.Skip("native audio backend is installed on this host")
	}
	if caps.MpvAvailable || caps.VlcAvailable {
		t.Skip("native audio backend is installed on this host")
	}
	if err := svc.LoadURL("http://127.0.0.1/stream"); err == nil {
		t.Fatal("expected load error when native audio is unavailable")
	}
	svc.Shutdown()
}

func TestSetPreferredBackendNoOpWhenUnchanged(t *testing.T) {
	svc := NewAudioService()
	defer svc.Shutdown()
	caps := svc.Capabilities()
	if !caps.NativeAvailable {
		t.Skip("native audio not available")
	}

	if err := svc.LoadURL("http://127.0.0.1:1/nope"); err != nil {
		t.Fatalf("initial load: %v", err)
	}
	svc.SetPreferredBackend(BackendAuto)
	if err := svc.LoadURL("http://127.0.0.1:1/nope"); err != nil {
		t.Fatalf("load after unchanged SetPreferredBackend: %v", err)
	}
}

func TestResolveBackend(t *testing.T) {
	tests := []struct {
		name      string
		preferred string
		mpvOK     bool
		vlcOK     bool
		want      string
	}{
		{"auto prefers mpv", BackendAuto, true, true, BackendMPV},
		{"auto falls back to vlc", BackendAuto, false, true, BackendVLC},
		{"auto none", BackendAuto, false, false, ""},
		{"mpv requested", BackendMPV, true, true, BackendMPV},
		{"mpv unavailable falls back to vlc", BackendMPV, false, true, BackendVLC},
		{"vlc requested", BackendVLC, true, true, BackendVLC},
		{"vlc unavailable falls back to mpv", BackendVLC, true, false, BackendMPV},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveBackend(tc.preferred, tc.mpvOK, tc.vlcOK); got != tc.want {
				t.Fatalf("resolveBackend(%q, %v, %v) = %q, want %q", tc.preferred, tc.mpvOK, tc.vlcOK, got, tc.want)
			}
		})
	}
}

func TestAudioServicePrepareUnavailable(t *testing.T) {
	svc := NewAudioService()
	defer svc.Shutdown()
	if svc.Capabilities().NativeAvailable {
		t.Skip("native backend available in this environment")
	}
	if err := svc.PrepareURL("http://example.com/track.flac"); err == nil {
		t.Fatal("expected PrepareURL to fail without native backend")
	}
	if svc.HasPrepared("http://example.com/track.flac") {
		t.Fatal("expected HasPrepared false")
	}
	if err := svc.ActivatePrepared(0); err == nil {
		t.Fatal("expected ActivatePrepared to fail")
	}
}

// fakeAudioBackend is an in-memory audioBackend for concurrency tests. The
// channels let tests observe when Play and the first SetVolume land.
type fakeAudioBackend struct {
	mu       sync.Mutex
	volume   float64
	playing  bool
	closed   bool
	playOnce sync.Once
	playCh   chan struct{}
	volOnce  sync.Once
	volCh    chan struct{}
}

func newFakeAudioBackend() *fakeAudioBackend {
	return &fakeAudioBackend{
		playCh: make(chan struct{}),
		volCh:  make(chan struct{}),
	}
}

func (f *fakeAudioBackend) LoadURL(string) error { return nil }

func (f *fakeAudioBackend) Play() error {
	f.mu.Lock()
	f.playing = true
	f.mu.Unlock()
	f.playOnce.Do(func() { close(f.playCh) })
	return nil
}

func (f *fakeAudioBackend) Pause() {
	f.mu.Lock()
	f.playing = false
	f.mu.Unlock()
}

func (f *fakeAudioBackend) Seek(float64) error { return nil }

func (f *fakeAudioBackend) SetVolume(v float64) {
	f.mu.Lock()
	f.volume = v
	f.mu.Unlock()
	f.volOnce.Do(func() { close(f.volCh) })
}

func (f *fakeAudioBackend) GetState() AudioPlaybackState {
	f.mu.Lock()
	defer f.mu.Unlock()
	return AudioPlaybackState{Playing: f.playing}
}

func (f *fakeAudioBackend) ClearEnded() {}

func (f *fakeAudioBackend) Close() {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
}

func (f *fakeAudioBackend) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// The crossfade ramp must not hold opMu, or every other playback op stalls
// for the full fade duration.
func TestActivatePreparedFadeKeepsOpsResponsive(t *testing.T) {
	primary := newFakeAudioBackend()
	standby := newFakeAudioBackend()
	svc := &AudioService{
		backend:     primary,
		standby:     standby,
		preparedURL: "next",
		volume:      1,
	}

	done := make(chan error, 1)
	go func() { done <- svc.ActivatePrepared(1) }()

	// The first fade step signals volCh right after opMu is released.
	waitFor(t, standby.volCh, "crossfade to start")

	opsDone := make(chan struct{})
	go func() {
		defer close(opsDone)
		svc.GetState()
		if err := svc.Play(); err != nil {
			t.Errorf("Play during fade: %v", err)
		}
		svc.Pause()
		if err := svc.Seek(1); err != nil {
			t.Errorf("Seek during fade: %v", err)
		}
	}()
	select {
	case <-opsDone:
	case <-time.After(700 * time.Millisecond):
		t.Fatal("playback ops blocked behind crossfade")
	}

	if err := <-done; err != nil {
		t.Fatalf("ActivatePrepared: %v", err)
	}
	svc.mu.Lock()
	got := svc.backend
	svc.mu.Unlock()
	if got != standby {
		t.Fatal("standby was not promoted to primary")
	}
	if !primary.isClosed() {
		t.Fatal("primary was not closed after swap")
	}
}

// Teardown during the fade must win: the commit re-validates state and
// never resurrects a torn-down pair.
func TestActivatePreparedAbortsAfterShutdown(t *testing.T) {
	primary := newFakeAudioBackend()
	standby := newFakeAudioBackend()
	svc := &AudioService{
		backend:     primary,
		standby:     standby,
		preparedURL: "next",
		volume:      1,
	}

	done := make(chan error, 1)
	go func() { done <- svc.ActivatePrepared(0.4) }()
	waitFor(t, standby.volCh, "crossfade to start")

	svc.Shutdown()
	if err := <-done; err != nil {
		t.Fatalf("ActivatePrepared: %v", err)
	}
	svc.mu.Lock()
	_, isNoop := svc.backend.(noopAudioBackend)
	svc.mu.Unlock()
	if !isNoop {
		t.Fatal("faded standby replaced the post-shutdown backend")
	}
	if !standby.isClosed() {
		t.Fatal("standby leaked after aborted swap")
	}
}

func TestActivatePreparedRejectsConcurrentFade(t *testing.T) {
	primary := newFakeAudioBackend()
	standby := newFakeAudioBackend()
	svc := &AudioService{
		backend:     primary,
		standby:     standby,
		preparedURL: "next",
		volume:      1,
	}

	done := make(chan error, 1)
	go func() { done <- svc.ActivatePrepared(0.4) }()
	waitFor(t, standby.volCh, "crossfade to start")

	if err := svc.ActivatePrepared(0); !errors.Is(err, errNativeUnavailable) {
		t.Fatalf("expected errNativeUnavailable for concurrent fade, got %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("ActivatePrepared: %v", err)
	}
}
