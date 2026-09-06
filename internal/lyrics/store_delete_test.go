// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import (
	"path/filepath"
	"testing"
)

func TestStoreDeleteRemovesCachedTrack(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	instanceID := "local"
	trackID := "track-1"
	doc := &Document{
		Source: "subsonic",
		Lines:  []Line{{Text: "hello"}},
	}
	if err := store.Save(root, instanceID, trackID, doc); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := store.Load(root, instanceID, trackID); err != nil {
		t.Fatalf("Load before delete: %v", err)
	}
	if err := store.Delete(root, instanceID, trackID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Load(root, instanceID, trackID); err == nil {
		t.Fatal("expected miss after delete")
	}
	path := store.trackPath(root, instanceID, trackID)
	if filepath.Base(path) == "" {
		t.Fatal("unexpected empty path")
	}
}

func TestClearInstanceEmptyDoesNotWipeNestedInstances(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	doc := &Document{Source: "test", Lines: []Line{{Text: "line"}}}

	if err := store.Save(root, "", "root-track", doc); err != nil {
		t.Fatalf("Save root: %v", err)
	}
	if err := store.Save(root, "inst-1", "nested-track", doc); err != nil {
		t.Fatalf("Save nested: %v", err)
	}
	if err := store.ClearInstance(root, ""); err != nil {
		t.Fatalf("ClearInstance empty: %v", err)
	}
	if _, err := store.Load(root, "", "root-track"); err == nil {
		t.Fatal("expected root-level lyrics cleared")
	}
	if _, err := store.Load(root, "inst-1", "nested-track"); err != nil {
		t.Fatalf("nested instance lyrics must survive empty clear: %v", err)
	}
}
