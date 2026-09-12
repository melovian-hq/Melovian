// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"testing"

	"melovian/internal/api/realtime"
	"melovian/internal/metaloader"
	"melovian/internal/store"
)

func TestEmitScanProgressStaysOnOwningUser(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	alice, err := srv.auth.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("CreateUser alice: %v", err)
	}
	bob, err := srv.auth.CreateUser("bob", "password123")
	if err != nil {
		t.Fatalf("CreateUser bob: %v", err)
	}
	lib, err := srv.localLibraries.CreateForUser(alice.ID, store.CreateLocalLibraryInput{
		Name: "Alice Lib",
		Path: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("CreateForUser: %v", err)
	}

	aliceC := realtime.NewWSClient(alice.ID, 4)
	bobC := realtime.NewWSClient(bob.ID, 4)
	srv.events.Register(aliceC)
	srv.events.Register(bobC)

	srv.emitScanProgress(lib.ID, metaloader.ScanProgress{Processed: 3, Phase: "scanning"})

	select {
	case <-aliceC.Send:
	default:
		t.Fatal("owner missed scan progress")
	}
	select {
	case <-bobC.Send:
		t.Fatal("other user received scan progress")
	default:
	}
}
