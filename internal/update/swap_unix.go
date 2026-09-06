// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package update

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// replaceBinary atomically swaps target for newPath. The staged file must
// live on the same filesystem as target, which StageUpdate guarantees by
// staging inside the target's directory.
func replaceBinary(target, newPath string) error {
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("stat target: %w", err)
	}
	if err := os.Chmod(newPath, info.Mode().Perm()|0o755); err != nil {
		return fmt.Errorf("chmod staged binary: %w", err)
	}
	// Keep the old binary so a failed rename can roll back and so a running
	// process keeps its inode until exit.
	backup := target + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil {
		return fmt.Errorf("move current binary aside: %w", err)
	}
	if err := os.Rename(newPath, target); err != nil {
		_ = os.Rename(backup, target)
		return fmt.Errorf("install new binary: %w", err)
	}
	_ = os.Remove(backup)
	return nil
}

// selfPath resolves the running executable, with /proc/self/exe preferred
// on Linux so the path survives symlink resolution the same way for every
// caller.
func selfPath() (string, error) {
	if runtime.GOOS == "linux" {
		if p, err := os.Readlink("/proc/self/exe"); err == nil && p != "" {
			return p, nil
		}
	}
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(p)
}
