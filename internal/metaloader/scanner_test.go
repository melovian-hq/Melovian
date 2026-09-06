// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSignatureStable(t *testing.T) {
	sig := FileSignature("/music/song.flac", 12345, 1700000000)
	if sig == "" {
		t.Fatal("expected non-empty signature")
	}
	if sig != FileSignature("/music/song.flac", 12345, 1700000000) {
		t.Fatal("signature should be stable")
	}
	if sig == FileSignature("/music/song.flac", 12346, 1700000000) {
		t.Fatal("signature should change when size changes")
	}
}

func TestContentHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.mp3")
	if err := os.WriteFile(path, []byte("hello-melovian-local-library"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	hash, err := ContentHash(path, info.Size())
	if err != nil {
		t.Fatalf("ContentHash: %v", err)
	}
	if hash == "" {
		t.Fatal("expected hash")
	}
}
