// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package desktop

import (
	"os"
	"testing"
)

func TestIsWaylandSessionDetection(t *testing.T) {
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	t.Setenv("WAYLAND_DISPLAY", "")
	if !isWaylandSession() {
		t.Fatal("expected wayland from XDG_SESSION_TYPE")
	}

	t.Setenv("XDG_SESSION_TYPE", "x11")
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	if !isWaylandSession() {
		t.Fatal("expected wayland from WAYLAND_DISPLAY")
	}

	t.Setenv("XDG_SESSION_TYPE", "x11")
	t.Setenv("WAYLAND_DISPLAY", "")
	if isWaylandSession() {
		t.Fatal("expected non-wayland session")
	}
}

func TestDisableWebKitSandboxIfLandlocked(t *testing.T) {
	t.Setenv("WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS", "")
	_ = os.Unsetenv("WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS")

	DisableWebKitSandboxIfLandlocked(false)
	if os.Getenv("WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS") != "" {
		t.Fatal("sandbox must stay enabled when landlock is off")
	}

	DisableWebKitSandboxIfLandlocked(true)
	if os.Getenv("WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS") != "1" {
		t.Fatal("expected sandbox disabled when landlock is active")
	}
}

func TestDisableWebKitSandboxIfLandlockedKeepsUserValue(t *testing.T) {
	t.Setenv("WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS", "0")
	DisableWebKitSandboxIfLandlocked(true)
	if os.Getenv("WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS") != "0" {
		t.Fatal("expected user-set env value to be preserved")
	}
}

func TestSetDefaultEnvSkipsAlreadySetValues(t *testing.T) {
	t.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "0")
	applied := make(map[string]string)
	setDefaultEnv(applied, "WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	if len(applied) != 0 {
		t.Fatalf("expected no override when env already set, got %v", applied)
	}
	if os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") != "0" {
		t.Fatal("expected existing env value to remain unchanged")
	}
}

func TestGraphicsEnvironmentInfoReportsSessionState(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")

	env := GraphicsEnvironmentInfo(dir)
	if !env.Supported {
		t.Fatal("expected linux graphics support")
	}
	if env.Platform != "linux" {
		t.Fatalf("unexpected platform %q", env.Platform)
	}
	if !env.Wayland {
		t.Fatal("expected wayland session detection")
	}
	if env.RequiresRestart != true {
		t.Fatal("expected restart requirement for graphics workarounds")
	}
	if env.Settings != DefaultGraphicsSettings() {
		t.Fatalf("unexpected settings %+v", env.Settings)
	}
}

func TestAppliedWebKitEnvReturnsCopy(t *testing.T) {
	appliedWebKitEnv = map[string]string{"WEBKIT_DISABLE_DMABUF_RENDERER": "1"}
	got := AppliedWebKitEnv()
	got["WEBKIT_DISABLE_DMABUF_RENDERER"] = "0"
	if appliedWebKitEnv["WEBKIT_DISABLE_DMABUF_RENDERER"] != "1" {
		t.Fatal("expected AppliedWebKitEnv to return a defensive copy")
	}
}

func TestApplyWebKitNvidiaExplicitSyncOnWayland(t *testing.T) {
	dir := t.TempDir()
	if err := SaveGraphicsSettings(dir, GraphicsSettings{
		DisableDmabufRenderer:  false,
		DisableCompositingMode: false,
		NvDisableExplicitSync:  NvExplicitSyncOn,
	}); err != nil {
		t.Fatalf("SaveGraphicsSettings: %v", err)
	}

	t.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "")
	t.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "")
	t.Setenv("__NV_DISABLE_EXPLICIT_SYNC", "")
	t.Setenv("XDG_SESSION_TYPE", "x11")
	t.Setenv("WAYLAND_DISPLAY", "")

	applied := ApplyWebKitStabilityEnv(dir)
	if applied["__NV_DISABLE_EXPLICIT_SYNC"] != "1" {
		t.Fatalf("expected explicit sync workaround when forced on, got %v", applied)
	}
}
