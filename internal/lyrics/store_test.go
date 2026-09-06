// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreSaveLoadAndStats(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	doc := &Document{
		Source:   "lrclib",
		Artist:   "Artist",
		Title:    "Song",
		Synced:   true,
		RawValue: "Hello",
		Lines:    []Line{{Text: "Hello", StartMs: new(1000)}},
	}
	if err := store.Save(root, "inst-1", "track-1", doc); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := store.Load(root, "inst-1", "track-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Source != "lrclib" || loaded.Lines[0].Text != "Hello" {
		t.Fatalf("unexpected loaded doc: %+v", loaded)
	}
	count, bytes, err := store.Stats(root, "inst-1")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if count != 1 || bytes <= 0 {
		t.Fatalf("unexpected stats count=%d bytes=%d", count, bytes)
	}
	if err := store.ClearInstance(root, "inst-1"); err != nil {
		t.Fatalf("ClearInstance: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "inst-1")); !os.IsNotExist(err) {
		t.Fatalf("expected instance dir removed")
	}
}

func TestParseTextSyncedLines(t *testing.T) {
	doc := ParseText("[00:01.00]First\n[00:02.00]Second", FetchInput{})
	if !doc.Synced {
		t.Fatal("expected synced lyrics")
	}
	if len(doc.Lines) != 2 || doc.Lines[1].StartMs == nil {
		t.Fatalf("unexpected lines: %+v", doc.Lines)
	}
}
