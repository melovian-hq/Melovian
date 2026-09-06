// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package localmusic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadEmbeddedCoverRejectsOversizedPicture(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "huge.mp3")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := f.Write(make([]byte, 64)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, _, ok := ReadEmbeddedCover(path); ok {
		t.Fatal("expected no cover from bare file")
	}
}

func TestCoverCacheInvalidateLibrary(t *testing.T) {
	cache := NewCoverCache()
	cache.Set("lib_1", "trk_1", []byte("img"), "image/jpeg")
	cache.Set("lib_2", "trk_2", []byte("other"), "image/jpeg")

	cache.InvalidateLibrary("lib_1")
	if _, _, ok := cache.Get("lib_1", "trk_1"); ok {
		t.Fatal("expected lib_1 cover removed")
	}
	if _, _, ok := cache.Get("lib_2", "trk_2"); !ok {
		t.Fatal("expected lib_2 cover retained")
	}
}
