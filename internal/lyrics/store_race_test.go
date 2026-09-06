// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import (
	"fmt"
	"sync"
	"testing"
)

func TestStoreConcurrentSaveSameTrackRace(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	instanceID := "local"
	trackID := "track-race"

	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			doc := &Document{
				Source: fmt.Sprintf("src-%d", n),
				Lines:  []Line{{Text: fmt.Sprintf("line-%d", n)}},
			}
			if err := store.Save(root, instanceID, trackID, doc); err != nil {
				t.Errorf("Save: %v", err)
			}
		}(i)
	}
	wg.Wait()

	doc, err := store.Load(root, instanceID, trackID)
	if err != nil {
		t.Fatalf("Load after concurrent Save: %v", err)
	}
	if doc == nil || len(doc.Lines) == 0 {
		t.Fatal("expected a valid document after concurrent saves")
	}
}
