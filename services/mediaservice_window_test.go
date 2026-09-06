// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

import "testing"

func TestWindowLifecycleMethodsNoOpWithoutApp(t *testing.T) {
	svc := NewMediaService(nil)

	svc.HideMainWindow()
	svc.MinimizeMainWindow()
	svc.ToggleMaximizeMainWindow()
	svc.HandleTitleBarDoubleClick()
	svc.RequestMainWindowClose()
	svc.SetNativeTitleBarEnabled(true)
	svc.QuitApp()
	svc.RaiseWindow()
}

func TestShutdownClearsPlaybackState(t *testing.T) {
	svc := NewMediaService(nil)
	svc.mu.Lock()
	svc.state = PlaybackState{
		Title:      "Song",
		Artist:     "Artist",
		TrackID:    "trk-1",
		Playing:    true,
		PositionMs: 1500,
		DurationMs: 180000,
	}
	svc.mu.Unlock()

	svc.Shutdown()

	state := svc.GetPlaybackState()
	if state.Title != "" || state.Playing {
		t.Fatalf("expected cleared playback state, got %#v", state)
	}
}

func TestWindowCloseEventNamesMatchDesktopContract(t *testing.T) {
	if MainWindowCloseRequestedEvent != "melovian:window-close-requested" {
		t.Fatalf("unexpected close event %q", MainWindowCloseRequestedEvent)
	}
	if AppQuitRequestedEvent != "melovian:app-quit-requested" {
		t.Fatalf("unexpected quit event %q", AppQuitRequestedEvent)
	}
}
