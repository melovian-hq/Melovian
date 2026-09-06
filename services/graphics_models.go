// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

// GraphicsSettings controls Linux WebKit/NVIDIA workarounds for the desktop app.
type GraphicsSettings struct {
	DisableDmabufRenderer  bool   `json:"disableDmabufRenderer"`
	DisableCompositingMode bool   `json:"disableCompositingMode"`
	NvDisableExplicitSync  string `json:"nvDisableExplicitSync"`
}

// GraphicsEnvironment describes host graphics capabilities and active workarounds.
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
