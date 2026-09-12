// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"melovian/internal/api/apishared"
	"melovian/internal/store"
)

func TestSubsonicForContextDoesNotLeakGlobalClient(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	alice, err := srv.auth.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("CreateUser alice: %v", err)
	}
	bob, err := srv.auth.CreateUser("bob", "password123")
	if err != nil {
		t.Fatalf("CreateUser bob: %v", err)
	}
	inst, err := srv.instances.CreateForUser(alice.ID, store.CreateInstanceInput{
		Name:      "Alice Server",
		ServerURL: "http://alice.example",
		Username:  "alice",
		Password:  "secret",
	})
	if err != nil {
		t.Fatalf("CreateForUser: %v", err)
	}
	if err := srv.instances.SetActive(inst.ID); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if err := srv.reloadActiveSubsonic(); err != nil {
		t.Fatalf("reloadActiveSubsonic: %v", err)
	}

	bobCtx := apishared.WithUserID(context.Background(), bob.ID)
	client := srv.subsonicForContext(bobCtx)
	if client.Enabled() {
		t.Fatal("authenticated user without an instance must not inherit the global Subsonic client")
	}

	anon := srv.subsonicForContext(context.Background())
	if !anon.Enabled() {
		t.Fatal("no-auth path should still use the process-global Subsonic client")
	}

	token, _, err := srv.auth.CreateSession(bob.ID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/music/status", nil)
	req.AddCookie(&http.Cookie{Name: store.SessionCookieName(), Value: token})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["enabled"] != false {
		t.Fatalf("bob must not see alice's server, got %+v", payload)
	}
}

func TestWriteSessionStopsOnCreateFailure(t *testing.T) {
	srv, db := newAuthTestServer(t)
	user, err := srv.auth.CreateUser("admin", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	_ = db.Close()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if err := srv.writeSession(rec, req, user.ID); err == nil {
		t.Fatal("expected session create failure after closed db")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
