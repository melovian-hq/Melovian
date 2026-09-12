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

// ErrPathOutsideRoots is returned when a path resolves outside every root.
var ErrPathOutsideRoots = fmt.Errorf("path is outside the allowed roots")

// ResolveInside returns the canonical form of path when it lives under one of
// roots. The path must exist so symlink resolution succeeds, and both the
// path and each root are resolved before the containment check so a symlink
// inside an allowed root cannot redirect elsewhere.
func ResolveInside(path string, roots ...string) (string, error) {
	resolved, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		rr, err := filepath.EvalSymlinks(filepath.Clean(root))
		if err != nil {
			continue
		}
		if err := PathEscapesRoot(rr, resolved); err == nil {
			return resolved, nil
		}
	}
	return "", ErrPathOutsideRoots
}
