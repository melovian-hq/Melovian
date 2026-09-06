// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package desktop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const graphicsSettingsFile = "graphics-settings.json"

// NvExplicitSyncMode controls the NVIDIA Wayland explicit-sync workaround.
type NvExplicitSyncMode string

const (
	NvExplicitSyncAuto NvExplicitSyncMode = "auto"
	NvExplicitSyncOn   NvExplicitSyncMode = "on"
	NvExplicitSyncOff  NvExplicitSyncMode = "off"
)

// GraphicsSettings stores user preferences for Linux WebKit/NVIDIA workarounds.
// Changes take effect on the next application launch.
type GraphicsSettings struct {
	DisableDmabufRenderer  bool               `json:"disableDmabufRenderer"`
	DisableCompositingMode bool               `json:"disableCompositingMode"`
	NvDisableExplicitSync  NvExplicitSyncMode `json:"nvDisableExplicitSync"`
}

// GraphicsEnvironment describes the host graphics stack for the settings UI.
type GraphicsEnvironment struct {
	Supported           bool              `json:"supported"`
	Platform            string            `json:"platform"`
	Wayland             bool              `json:"wayland"`
	Nvidia              bool              `json:"nvidia"`
	Settings            GraphicsSettings  `json:"settings"`
	Defaults            GraphicsSettings  `json:"defaults"`
	AppliedEnv          map[string]string `json:"appliedEnv"`
	RequiresRestart     bool              `json:"requiresRestart"`
	RestartRequiredNote string            `json:"restartRequiredNote,omitempty"`
}

func DefaultGraphicsSettings() GraphicsSettings {
	return GraphicsSettings{
		DisableDmabufRenderer:  true,
		DisableCompositingMode: false,
		NvDisableExplicitSync:  NvExplicitSyncAuto,
	}
}

func graphicsSettingsPath(dataDir string) string {
	return filepath.Join(dataDir, graphicsSettingsFile)
}

func LoadGraphicsSettings(dataDir string) GraphicsSettings {
	defaults := DefaultGraphicsSettings()
	raw, err := os.ReadFile(graphicsSettingsPath(dataDir))
	if err != nil {
		return defaults
	}
	var stored GraphicsSettings
	if err := json.Unmarshal(raw, &stored); err != nil {
		return defaults
	}
	return MergeGraphicsSettings(stored)
}

func MergeGraphicsSettings(partial GraphicsSettings) GraphicsSettings {
	defaults := DefaultGraphicsSettings()
	out := defaults
	out.DisableDmabufRenderer = partial.DisableDmabufRenderer
	out.DisableCompositingMode = partial.DisableCompositingMode
	switch partial.NvDisableExplicitSync {
	case NvExplicitSyncOn, NvExplicitSyncOff, NvExplicitSyncAuto:
		out.NvDisableExplicitSync = partial.NvDisableExplicitSync
	default:
		out.NvDisableExplicitSync = NvExplicitSyncAuto
	}
	return out
}

func SaveGraphicsSettings(dataDir string, settings GraphicsSettings) error {
	if dataDir == "" {
		return fmt.Errorf("data directory is required")
	}
	merged := MergeGraphicsSettings(settings)
	raw, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	path := graphicsSettingsPath(dataDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func nvExplicitSyncEnabled(settings GraphicsSettings, wayland, nvidia bool) bool {
	switch settings.NvDisableExplicitSync {
	case NvExplicitSyncOn:
		return true
	case NvExplicitSyncOff:
		return false
	default:
		return wayland && nvidia
	}
}
