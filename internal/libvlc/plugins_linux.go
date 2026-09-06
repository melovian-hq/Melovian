// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build linux && !android

package libvlc

import (
	"os"
	"path/filepath"
)

func prepareVLCEnvironment() {
	if os.Getenv("VLC_PLUGIN_PATH") != "" {
		return
	}

	candidates := []string{
		"/usr/lib/vlc/plugins",
		"/usr/lib64/vlc/plugins",
		"/usr/lib/x86_64-linux-gnu/vlc/plugins",
		"/usr/lib/aarch64-linux-gnu/vlc/plugins",
		"/usr/lib/arm-linux-gnueabihf/vlc/plugins",
		"/usr/lib32/vlc/plugins",
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			_ = os.Setenv("VLC_PLUGIN_PATH", filepath.Clean(candidate))
			return
		}
	}
}
