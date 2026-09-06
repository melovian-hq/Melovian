// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

import "testing"

func TestNoopMediaController(t *testing.T) {
	ctrl := noopMediaController{}
	state := PlaybackState{
		Title:      "Song",
		Artist:     "Artist",
		TrackID:    "trk-1",
		Playing:    true,
		PositionMs: 1500,
		DurationMs: 180000,
	}
	if err := ctrl.Update(state); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := ctrl.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestMediaServiceQueuesActions(t *testing.T) {
	svc := NewMediaService(nil)
	svc.onNext()
	action := svc.PollMediaAction()
	if action.Action != "next" {
		t.Fatalf("expected next, got %q", action.Action)
	}
	if polled := svc.PollMediaAction(); polled.Action != "" {
		t.Fatalf("expected drained action, got %#v", polled)
	}
}

func TestMediaServiceTogglePlayback(t *testing.T) {
	svc := NewMediaService(nil)
	svc.mu.Lock()
	svc.state.Playing = true
	svc.mu.Unlock()
	svc.TogglePlayback()
	if action := svc.PollMediaAction(); action.Action != "pause" {
		t.Fatalf("expected pause, got %q", action.Action)
	}

	svc.mu.Lock()
	svc.state.Playing = false
	svc.mu.Unlock()
	svc.TogglePlayback()
	if action := svc.PollMediaAction(); action.Action != "play" {
		t.Fatalf("expected play, got %q", action.Action)
	}
}
