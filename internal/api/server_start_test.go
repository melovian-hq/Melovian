// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net"
	"testing"
)

func TestServerStartBindFailure(t *testing.T) {
	blocker, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen blocker: %v", err)
	}
	defer func() { _ = blocker.Close() }()

	srv, _ := newTestServer(t)
	srv.server.Addr = blocker.Addr().String()
	if err := srv.Start(); err == nil {
		t.Fatal("expected bind failure")
	}
}

func TestServerStartSuccess(t *testing.T) {
	srv, _ := newTestServer(t)
	srv.server.Addr = "127.0.0.1:0"
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = srv.Stop(t.Context()) }()

	conn, err := net.Dial("tcp", srv.ListenAddr())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_ = conn.Close()
}
