// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package osutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PathEscapesRoot reports whether path is outside root after Clean.
// Both arguments should be absolute. Symlinks are not resolved here.
func PathEscapesRoot(root, path string) error {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes root")
	}
	return nil
}
