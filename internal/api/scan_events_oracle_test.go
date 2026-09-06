// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"testing"

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

	aliceC := &wsClient{userID: alice.ID, send: make(chan []byte, 4)}
	bobC := &wsClient{userID: bob.ID, send: make(chan []byte, 4)}
	srv.events.register(aliceC)
	srv.events.register(bobC)

	srv.emitScanProgress(lib.ID, metaloader.ScanProgress{Processed: 3, Phase: "scanning"})

	select {
	case <-aliceC.send:
	default:
		t.Fatal("owner missed scan progress")
	}
	select {
	case <-bobC.send:
		t.Fatal("other user received scan progress")
	default:
	}
}
