// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package libvlc

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func tryLoadLibrary() error {
	dirs := vlcInstallDirs()
	names := []string{"libvlc.dll"}
	var lastErr error

	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		_ = windows.SetDllDirectory(windows.StringToUTF16Ptr(dir))
		for _, name := range names {
			path := filepath.Join(dir, name)
			handle, err := windows.LoadLibrary(windows.StringToUTF16Ptr(path))
			if err != nil {
				lastErr = err
				continue
			}
			libHandle = uintptr(handle)
			return nil
		}
	}

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
		return fmt.Errorf("libvlc: %w", lastErr)
	}
	return errUnavailable
}

func vlcInstallDirs() []string {
	seen := make(map[string]struct{})
	var dirs []string
	add := func(dir string) {
		dir = filepath.Clean(dir)
		if dir == "" || dir == "." {
			return
		}
		if _, ok := seen[dir]; ok {
			return
		}
		if st, err := os.Stat(filepath.Join(dir, "libvlc.dll")); err != nil || st.IsDir() {
			return
		}
		seen[dir] = struct{}{}
		dirs = append(dirs, dir)
	}

	for _, keyPath := range []string{
		`SOFTWARE\VideoLAN\VLC`,
		`SOFTWARE\WOW6432Node\VideoLAN\VLC`,
	} {
		key, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		if installDir, _, err := key.GetStringValue("InstallDir"); err == nil {
			add(installDir)
		}
		_ = key.Close()
	}

	add(filepath.Join(os.Getenv("ProgramFiles"), "VideoLAN", "VLC"))
	add(filepath.Join(os.Getenv("ProgramFiles(x86)"), "VideoLAN", "VLC"))
	add(filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "VideoLAN", "VLC"))
	return dirs
}
