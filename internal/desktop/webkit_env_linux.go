// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package desktop

import (
	"log/slog"
	"maps"
	"os"
	"strings"
)

var appliedWebKitEnv map[string]string

func ApplyWebKitStabilityEnv(dataDir string) map[string]string {
	settings := LoadGraphicsSettings(dataDir)
	wayland := isWaylandSession()
	nvidia := isNvidiaDriver()

	applied := make(map[string]string)
	if settings.DisableDmabufRenderer {
		setDefaultEnv(applied, "WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}
	if settings.DisableCompositingMode {
		setDefaultEnv(applied, "WEBKIT_DISABLE_COMPOSITING_MODE", "1")
	}
	if nvExplicitSyncEnabled(settings, wayland, nvidia) {
		setDefaultEnv(applied, "__NV_DISABLE_EXPLICIT_SYNC", "1")
	}

	appliedWebKitEnv = applied
	if len(applied) > 0 {
		slog.Info("applied graphics stability environment", "vars", applied)
	}
	return applied
}

func AppliedWebKitEnv() map[string]string {
	if appliedWebKitEnv == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(appliedWebKitEnv))
	maps.Copy(out, appliedWebKitEnv)
	return out
}

func GraphicsEnvironmentInfo(dataDir string) GraphicsEnvironment {
	defaults := DefaultGraphicsSettings()
	settings := LoadGraphicsSettings(dataDir)
	wayland := isWaylandSession()
	nvidia := isNvidiaDriver()

	return GraphicsEnvironment{
		Supported:           true,
		Platform:            "linux",
		Wayland:             wayland,
		Nvidia:              nvidia,
		Settings:            settings,
		Defaults:            defaults,
		AppliedEnv:          AppliedWebKitEnv(),
		RequiresRestart:     true,
		RestartRequiredNote: "Graphics workarounds apply on the next launch.",
	}
}

func isWaylandSession() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("XDG_SESSION_TYPE")), "wayland") {
		return true
	}
	return strings.TrimSpace(os.Getenv("WAYLAND_DISPLAY")) != ""
}

func isNvidiaDriver() bool {
	if _, err := os.Stat("/proc/driver/nvidia/version"); err == nil {
		return true
	}
	if _, err := os.Stat("/sys/module/nvidia"); err == nil {
		return true
	}
	return false
}

func setDefaultEnv(applied map[string]string, key, value string) {
	if os.Getenv(key) != "" {
		return
	}
	_ = os.Setenv(key, value)
	applied[key] = value
}

// DisableWebKitSandboxIfLandlocked turns off WebKit's bubblewrap sandbox when
// landlock is active. Landlock's hooks deny the mount namespace setup bwrap
// performs ("Failed to make / slave") no matter how broad the ruleset is, so
// WebKit's sandbox can never launch under it. Disabling explicitly avoids a
// failed bwrap spawn on every web process start. When landlock is off the
// sandbox is left alone so it can run.
func DisableWebKitSandboxIfLandlocked(landlockEnabled bool) {
	if !landlockEnabled {
		return
	}
	if os.Getenv("WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS") != "" {
		return
	}
	_ = os.Setenv("WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS", "1")
	slog.Info("webkit sandbox disabled", "reason", "bubblewrap cannot nest under landlock")
}
