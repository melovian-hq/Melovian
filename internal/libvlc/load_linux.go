// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build linux && !android

package libvlc

import (
	"fmt"

	"github.com/ebitengine/purego"
)

func tryLoadLibrary() error {
	names := []string{"libvlc.so.5", "libvlc.so"}
	var lastErr error
	for _, name := range names {
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
