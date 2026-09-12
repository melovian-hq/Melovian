// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package realtime

import (
	"sync"
	"testing"
	"time"
)

// Concurrent register + playback broadcast must not race on WSClient identity.
func TestDeviceSyncRegisterBroadcastRaceOracle(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"

	var wg sync.WaitGroup
	for i := range 24 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "dev-" + string(rune('a'+n%12))
			c := mockWSClient("", "", "")
			hub.Register(c)
			c.SetDeviceID(id)
			c.SetScopeKey(scope)
			reg.Register(c, "Browser", "Chromium")
			reg.UpdatePlayback(c, PlaybackSnapshot{
				TrackID:    "t1",
				PositionMs: int64(n * 100),
				Paused:     false,
			})
		}(i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("register/broadcast race oracle timed out")
	}

	list := reg.List(scope)
	if len(list) == 0 {
		t.Fatal("expected registered devices")
	}
}
