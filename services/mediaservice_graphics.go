// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

import "melovian/internal/desktop"

func (s *MediaService) SetDataDir(dataDir string) {
	s.dataDir = dataDir
}

func (s *MediaService) GetGraphicsEnvironment() GraphicsEnvironment {
	if s.dataDir == "" {
		return GraphicsEnvironment{Supported: false}
	}
	return toGraphicsEnvironment(desktop.GraphicsEnvironmentInfo(s.dataDir))
}

func (s *MediaService) SaveGraphicsSettings(settings GraphicsSettings) error {
	if s.dataDir == "" {
		return nil
	}
	return desktop.SaveGraphicsSettings(s.dataDir, toDesktopGraphicsSettings(settings))
}

func toGraphicsEnvironment(env desktop.GraphicsEnvironment) GraphicsEnvironment {
	return GraphicsEnvironment{
		Supported:           env.Supported,
		Platform:            env.Platform,
		Wayland:             env.Wayland,
		Nvidia:              env.Nvidia,
		Settings:            toServiceGraphicsSettings(env.Settings),
		Defaults:            toServiceGraphicsSettings(env.Defaults),
		AppliedEnv:          env.AppliedEnv,
		RequiresRestart:     env.RequiresRestart,
		RestartRequiredNote: env.RestartRequiredNote,
	}
}

func toServiceGraphicsSettings(settings desktop.GraphicsSettings) GraphicsSettings {
	return GraphicsSettings{
		DisableDmabufRenderer:  settings.DisableDmabufRenderer,
		DisableCompositingMode: settings.DisableCompositingMode,
		NvDisableExplicitSync:  string(settings.NvDisableExplicitSync),
	}
}

func toDesktopGraphicsSettings(settings GraphicsSettings) desktop.GraphicsSettings {
	return desktop.GraphicsSettings{
		DisableDmabufRenderer:  settings.DisableDmabufRenderer,
		DisableCompositingMode: settings.DisableCompositingMode,
		NvDisableExplicitSync:  desktop.NvExplicitSyncMode(settings.NvDisableExplicitSync),
	}
}
