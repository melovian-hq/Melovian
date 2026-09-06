// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package libmpv

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// prepareCreateEnv is a no-op on Windows. The msvcrt locale categories differ
// and the GTK locale interaction that affects mpv_create does not apply here.
func prepareCreateEnv() {}

func tryLoadLibrary() error {
	names := []string{"mpv-2.dll", "mpv-1.dll", "libmpv.dll"}
	var lastErr error
	for _, name := range names {
		handle, err := windows.LoadLibrary(windows.StringToUTF16Ptr(name))
		if err != nil {
			lastErr = err
			continue
		}
		libHandle = uintptr(handle)
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("libmpv: %w", lastErr)
	}
	return errUnavailable
}
