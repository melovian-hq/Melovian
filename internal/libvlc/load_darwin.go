// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build darwin

package libvlc

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ebitengine/purego"
)

func tryLoadLibrary() error {
	candidates := []string{
		"/Applications/VLC.app/Contents/MacOS/lib/libvlc.dylib",
		"/Applications/VLC.app/Contents/MacOS/lib/libvlc.5.dylib",
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, "Applications/VLC.app/Contents/MacOS/lib/libvlc.dylib"),
		)
	}
	candidates = append(candidates, "libvlc.dylib", "libvlc.5.dylib")

	var lastErr error
	for _, name := range candidates {
		handle, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			lastErr = err
			continue
		}
		libHandle = uintptr(handle)
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("libvlc: %w", lastErr)
	}
	return errUnavailable
}
