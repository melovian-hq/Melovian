// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package osutil

import (
	"path/filepath"
	"testing"
)

func TestPathEscapesRootOracle(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "data", "melovian")
	inside := filepath.Join(root, "lyrics", "cache")
	if err := PathEscapesRoot(root, inside); err != nil {
		t.Fatalf("inside path should stay in jail: %v", err)
	}
	escaped := filepath.Join(root, "..", "..", "tmp", "pwned")
	if err := PathEscapesRoot(root, escaped); err == nil {
		t.Fatal("relative escape must be rejected")
	}
	if err := PathEscapesRoot(root, filepath.Join(string(filepath.Separator), "etc", "passwd")); err == nil {
		t.Fatal("absolute escape must be rejected")
	}
}
