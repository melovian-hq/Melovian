// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package update

import (
	"fmt"
	"os"
	"path/filepath"
)

// replaceBinary swaps target for newPath. Windows locks a running
// executable against overwrite but permits rename, so the old binary is
// renamed aside first, then the new one moved into place. The .old file is
// left for the next boot or a cleanup pass to remove.
func replaceBinary(target, newPath string) error {
	backup := target + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil {
		return fmt.Errorf("move current binary aside: %w", err)
	}
	if err := os.Rename(newPath, target); err != nil {
		_ = os.Rename(backup, target)
		return fmt.Errorf("install new binary: %w", err)
	}
	return nil
}

func selfPath() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(p)
}
