// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func jsonFieldNames(t *testing.T, value any) map[string]struct{} {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out := make(map[string]struct{}, len(payload))
	for key := range payload {
		out[key] = struct{}{}
	}
	return out
}

func assertGoJSONTags(t *testing.T, goType reflect.Type, serialized map[string]struct{}) {
	t.Helper()
	if goType.Kind() == reflect.Pointer {
		goType = goType.Elem()
	}
	for field := range goType.Fields() {
		if !field.IsExported() {
			continue
		}
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		parts := strings.Split(tag, ",")
		name := parts[0]
		omitempty := slices.Contains(parts[1:], "omitempty")
		if omitempty {
			continue
		}
		if _, ok := serialized[name]; !ok {
			t.Errorf("field %s with json tag %q missing from serialized output", field.Name, name)
		}
	}
}

func TestAudioCapabilitiesJSONContract(t *testing.T) {
	cap := AudioCapabilities{
		NativeAvailable: true,
		Backend:         BackendAuto,
		MpvAvailable:    true,
		VlcAvailable:    false,
	}
	keys := jsonFieldNames(t, cap)
	assertGoJSONTags(t, reflect.TypeFor[AudioCapabilities](), keys)
	for _, required := range []string{"nativeAvailable", "backend", "mpvAvailable", "vlcAvailable"} {
		if _, ok := keys[required]; !ok {
			t.Fatalf("missing required key %q", required)
		}
	}
}

func TestAudioPlaybackStateJSONContract(t *testing.T) {
	state := AudioPlaybackState{
		CurrentTime: 12.5,
		Duration:    180,
		Playing:     true,
		Ended:       false,
		Error:       "",
	}
	keys := jsonFieldNames(t, state)
	assertGoJSONTags(t, reflect.TypeFor[AudioPlaybackState](), keys)
	for _, required := range []string{"currentTime", "duration", "playing", "ended", "error"} {
		if _, ok := keys[required]; !ok {
			t.Fatalf("missing required key %q", required)
		}
	}
}

func TestPlaybackStateJSONContract(t *testing.T) {
	state := PlaybackState{
		Title:         "Song",
		Artist:        "Artist",
		Album:         "Album",
		TrackID:       "trk-1",
		CoverArtID:    "cover-1",
		CoverArtURL:   "/cover",
		DurationMs:    200000,
		PositionMs:    1000,
		Playing:       true,
		CanPlay:       true,
		CanPause:      true,
		CanGoNext:     true,
		CanGoPrevious: false,
	}
	keys := jsonFieldNames(t, state)
	assertGoJSONTags(t, reflect.TypeFor[PlaybackState](), keys)
	for _, required := range []string{
		"title", "artist", "album", "trackId", "coverArtId", "coverArtUrl",
		"durationMs", "positionMs", "playing",
		"canPlay", "canPause", "canGoNext", "canGoPrevious",
	} {
		if _, ok := keys[required]; !ok {
			t.Fatalf("missing required key %q", required)
		}
	}
}

func TestPendingMediaActionJSONContract(t *testing.T) {
	action := PendingMediaAction{
		Action:  "play",
		SeekMs:  5000,
		OpenURI: "melovian://open",
	}
	keys := jsonFieldNames(t, action)
	assertGoJSONTags(t, reflect.TypeFor[PendingMediaAction](), keys)
	for _, required := range []string{"action", "seekMs", "openUri"} {
		if _, ok := keys[required]; !ok {
			t.Fatalf("missing required key %q", required)
		}
	}
}

func TestGraphicsSettingsJSONContract(t *testing.T) {
	settings := GraphicsSettings{
		DisableDmabufRenderer:  true,
		DisableCompositingMode: false,
		NvDisableExplicitSync:  "auto",
	}
	keys := jsonFieldNames(t, settings)
	assertGoJSONTags(t, reflect.TypeFor[GraphicsSettings](), keys)
	for _, required := range []string{
		"disableDmabufRenderer",
		"disableCompositingMode",
		"nvDisableExplicitSync",
	} {
		if _, ok := keys[required]; !ok {
			t.Fatalf("missing required key %q", required)
		}
	}
}

func TestGraphicsEnvironmentJSONContract(t *testing.T) {
	env := GraphicsEnvironment{
		Supported: true,
		Platform:  "linux",
		Wayland:   true,
		Nvidia:    false,
		Settings: GraphicsSettings{
			DisableDmabufRenderer:  true,
			DisableCompositingMode: false,
			NvDisableExplicitSync:  "auto",
		},
		Defaults: GraphicsSettings{
			DisableDmabufRenderer:  true,
			DisableCompositingMode: false,
			NvDisableExplicitSync:  "auto",
		},
		AppliedEnv: map[string]string{
			"WEBKIT_DISABLE_DMABUF_RENDERER": "1",
		},
		RequiresRestart: true,
	}
	keys := jsonFieldNames(t, env)
	assertGoJSONTags(t, reflect.TypeFor[GraphicsEnvironment](), keys)
	for _, required := range []string{
		"supported",
		"platform",
		"wayland",
		"nvidia",
		"settings",
		"defaults",
		"appliedEnv",
		"requiresRestart",
	} {
		if _, ok := keys[required]; !ok {
			t.Fatalf("missing required key %q", required)
		}
	}
}
