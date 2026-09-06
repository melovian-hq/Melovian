// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

import "testing"

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
