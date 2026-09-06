// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package desktop

import (
	"os"
	"runtime"
	"testing"
)

func TestDefaultGraphicsSettings(t *testing.T) {
	defaults := DefaultGraphicsSettings()
	if !defaults.DisableDmabufRenderer {
		t.Fatal("expected DMA-BUF workaround on by default")
	}
	if defaults.DisableCompositingMode {
		t.Fatal("expected compositing enabled by default")
	}
	if defaults.NvDisableExplicitSync != NvExplicitSyncAuto {
		t.Fatalf("expected auto NV explicit sync default, got %q", defaults.NvDisableExplicitSync)
	}
}

func TestSaveAndLoadGraphicsSettings(t *testing.T) {
	dir := t.TempDir()
	settings := GraphicsSettings{
		DisableDmabufRenderer:  false,
		DisableCompositingMode: true,
		NvDisableExplicitSync:  NvExplicitSyncOff,
	}
	if err := SaveGraphicsSettings(dir, settings); err != nil {
		t.Fatalf("SaveGraphicsSettings: %v", err)
	}
	loaded := LoadGraphicsSettings(dir)
	if loaded != settings {
		t.Fatalf("loaded %+v, want %+v", loaded, settings)
	}
}

func TestNvExplicitSyncEnabled(t *testing.T) {
	cases := []struct {
		mode     NvExplicitSyncMode
		wayland  bool
		nvidia   bool
		expected bool
	}{
		{NvExplicitSyncOn, false, false, true},
		{NvExplicitSyncOff, true, true, false},
		{NvExplicitSyncAuto, true, true, true},
		{NvExplicitSyncAuto, true, false, false},
		{NvExplicitSyncAuto, false, true, false},
	}
	for i, tc := range cases {
		settings := GraphicsSettings{NvDisableExplicitSync: tc.mode}
		got := nvExplicitSyncEnabled(settings, tc.wayland, tc.nvidia)
		if got != tc.expected {
			t.Fatalf("case %d: got %v want %v", i, got, tc.expected)
		}
	}
}

func TestApplyWebKitStabilityEnvLinuxDefaults(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux only")
	}

	dir := t.TempDir()
	t.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "")
	t.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "")
	t.Setenv("__NV_DISABLE_EXPLICIT_SYNC", "")
	t.Setenv("XDG_SESSION_TYPE", "x11")
	t.Setenv("WAYLAND_DISPLAY", "")

	applied := ApplyWebKitStabilityEnv(dir)
	if applied["WEBKIT_DISABLE_DMABUF_RENDERER"] != "1" {
		t.Fatalf("expected DMA-BUF disabled by default, got %v", applied)
	}
	if _, ok := applied["WEBKIT_DISABLE_COMPOSITING_MODE"]; ok {
		t.Fatal("compositing should stay enabled by default")
	}
	if os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") != "1" {
		t.Fatal("expected WEBKIT_DISABLE_DMABUF_RENDERER=1")
	}
}

func TestLoadGraphicsSettingsCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := graphicsSettingsPath(dir)
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	loaded := LoadGraphicsSettings(dir)
	if loaded != DefaultGraphicsSettings() {
		t.Fatalf("loaded %+v, want defaults", loaded)
	}
}

func TestLoadGraphicsSettingsInvalidNvMode(t *testing.T) {
	dir := t.TempDir()
	path := graphicsSettingsPath(dir)
	payload := []byte(`{"disableDmabufRenderer":false,"nvDisableExplicitSync":"maybe"}`)
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	loaded := LoadGraphicsSettings(dir)
	if loaded.NvDisableExplicitSync != NvExplicitSyncAuto {
		t.Fatalf("expected auto fallback, got %q", loaded.NvDisableExplicitSync)
	}
	if loaded.DisableDmabufRenderer {
		t.Fatal("expected stored DMA-BUF override to survive invalid NV mode")
	}
}

func TestSaveGraphicsSettingsEmptyDataDir(t *testing.T) {
	err := SaveGraphicsSettings("", DefaultGraphicsSettings())
	if err == nil {
		t.Fatal("expected error for empty data directory")
	}
}

func TestMergeGraphicsSettingsInvalidNvMode(t *testing.T) {
	merged := MergeGraphicsSettings(GraphicsSettings{
		DisableDmabufRenderer: false,
		NvDisableExplicitSync: NvExplicitSyncMode("bogus"),
	})
	if merged.NvDisableExplicitSync != NvExplicitSyncAuto {
		t.Fatalf("expected auto fallback, got %q", merged.NvDisableExplicitSync)
	}
	if merged.DisableDmabufRenderer {
		t.Fatal("expected stored DMA-BUF override to survive invalid NV mode")
	}
}

func TestApplyWebKitRespectsSavedSettings(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux only")
	}

	dir := t.TempDir()
	if err := SaveGraphicsSettings(dir, GraphicsSettings{
		DisableDmabufRenderer:  false,
		DisableCompositingMode: true,
		NvDisableExplicitSync:  NvExplicitSyncOff,
	}); err != nil {
		t.Fatalf("SaveGraphicsSettings: %v", err)
	}

	t.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "")
	t.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "")
	t.Setenv("__NV_DISABLE_EXPLICIT_SYNC", "")

	applied := ApplyWebKitStabilityEnv(dir)
	if _, ok := applied["WEBKIT_DISABLE_DMABUF_RENDERER"]; ok {
		t.Fatal("expected DMA-BUF left enabled when disabled in settings")
	}
	if applied["WEBKIT_DISABLE_COMPOSITING_MODE"] != "1" {
		t.Fatal("expected compositing disabled from saved settings")
	}
	if _, ok := applied["__NV_DISABLE_EXPLICIT_SYNC"]; ok {
		t.Fatal("expected NV explicit sync left off from saved settings")
	}
}
