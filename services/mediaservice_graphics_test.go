// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"runtime"
	"testing"

	"melovian/internal/desktop"
)

func TestGetGraphicsEnvironmentWithoutDataDir(t *testing.T) {
	svc := NewMediaService(nil)
	env := svc.GetGraphicsEnvironment()
	if env.Supported {
		t.Fatal("expected unsupported graphics environment without data dir")
	}
}

func TestSaveGraphicsSettingsWithoutDataDir(t *testing.T) {
	svc := NewMediaService(nil)
	if err := svc.SaveGraphicsSettings(GraphicsSettings{
		DisableDmabufRenderer: true,
	}); err != nil {
		t.Fatalf("expected no-op save without data dir, got %v", err)
	}
}

func TestGraphicsSettingsRoundTrip(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux graphics settings persistence")
	}

	dir := t.TempDir()
	svc := NewMediaService(nil)
	svc.SetDataDir(dir)

	want := GraphicsSettings{
		DisableDmabufRenderer:  false,
		DisableCompositingMode: true,
		NvDisableExplicitSync:  "off",
	}
	if err := svc.SaveGraphicsSettings(want); err != nil {
		t.Fatalf("SaveGraphicsSettings: %v", err)
	}

	env := svc.GetGraphicsEnvironment()
	if !env.Supported {
		t.Fatal("expected supported graphics environment")
	}
	if env.Settings != want {
		t.Fatalf("settings %+v, want %+v", env.Settings, want)
	}
}

func TestGraphicsEnvironmentConversionPreservesAppliedEnv(t *testing.T) {
	applied := map[string]string{"WEBKIT_DISABLE_DMABUF_RENDERER": "1"}
	got := toGraphicsEnvironment(desktop.GraphicsEnvironment{
		Supported:           true,
		Platform:            "linux",
		Wayland:             true,
		Nvidia:              false,
		Settings:            desktop.DefaultGraphicsSettings(),
		Defaults:            desktop.DefaultGraphicsSettings(),
		AppliedEnv:          applied,
		RequiresRestart:     true,
		RestartRequiredNote: "restart",
	})
	if got.AppliedEnv["WEBKIT_DISABLE_DMABUF_RENDERER"] != "1" {
		t.Fatalf("unexpected applied env %+v", got.AppliedEnv)
	}
	if got.RestartRequiredNote != "restart" {
		t.Fatalf("unexpected restart note %q", got.RestartRequiredNote)
	}
}

func TestDesktopGraphicsSettingsConversion(t *testing.T) {
	service := GraphicsSettings{
		DisableDmabufRenderer:  true,
		DisableCompositingMode: false,
		NvDisableExplicitSync:  "on",
	}
	desktopSettings := toDesktopGraphicsSettings(service)
	if desktopSettings.NvDisableExplicitSync != desktop.NvExplicitSyncOn {
		t.Fatalf("unexpected nv mode %q", desktopSettings.NvDisableExplicitSync)
	}
	roundTrip := toServiceGraphicsSettings(desktopSettings)
	if roundTrip != service {
		t.Fatalf("round trip %+v, want %+v", roundTrip, service)
	}
}
