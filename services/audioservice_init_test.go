// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"testing"

	"melovian/internal/libmpv"
	"melovian/internal/libvlc"
)

func TestNativeLibrariesInitialize(t *testing.T) {
	if libmpv.Available() {
		player, err := libmpv.NewPlayer()
		if err != nil {
			t.Fatalf("mpv NewPlayer: %v", err)
		}
		player.Close()
	} else {
		t.Log("mpv library not installed")
	}

	if libvlc.Available() {
		player, err := libvlc.NewPlayer()
		if err != nil {
			t.Fatalf("vlc NewPlayer: %v", err)
		}
		player.Close()
	} else {
		t.Log("vlc library not installed")
	}
}

func TestAudioServiceCapabilitiesMatchInit(t *testing.T) {
	svc := NewAudioService()
	caps := svc.Capabilities()
	defer svc.Shutdown()

	if caps.MpvAvailable {
		if _, err := libmpv.NewPlayer(); err != nil {
			t.Fatalf("mpv reported available but init failed: %v", err)
		}
	}
	if caps.VlcAvailable {
		if _, err := libvlc.NewPlayer(); err != nil {
			t.Fatalf("vlc reported available but init failed: %v", err)
		}
	}

	if caps.NativeAvailable {
		if caps.Backend == "" {
			t.Fatal("native available without backend name")
		}
		if err := svc.LoadURL("http://127.0.0.1:1/nope"); err != nil {
			t.Fatalf("expected load to succeed on noop url open: %v", err)
		}
	}
}
