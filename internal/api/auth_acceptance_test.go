// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthAcceptanceSetupLoginStatusLogout(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	handler := srv.Handler()

	setupBody := bytes.NewBufferString(`{"username":"admin","password":"password123"}`)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/setup", setupBody)
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status %d: %s", setupRec.Code, setupRec.Body.String())
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	if cookie := authCookie(setupRec); cookie != nil {
		logoutReq.AddCookie(cookie)
	}
	logoutRec := httptest.NewRecorder()
	handler.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout after setup status %d", logoutRec.Code)
	}

	loginBody := bytes.NewBufferString(`{"username":"admin","password":"password123"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody)
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status %d: %s", loginRec.Code, loginRec.Body.String())
	}
	loginCookie := authCookie(loginRec)
	if loginCookie == nil {
		t.Fatal("expected session cookie after login")
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	statusReq.AddCookie(loginCookie)
	statusRec := httptest.NewRecorder()
	handler.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("status %d", statusRec.Code)
	}
	var status map[string]any
	if err := json.Unmarshal(statusRec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status["authenticated"] != true {
		t.Fatalf("expected authenticated true, got %+v", status)
	}

	finalLogout := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	finalLogout.AddCookie(loginCookie)
	finalRec := httptest.NewRecorder()
	handler.ServeHTTP(finalRec, finalLogout)
	if finalRec.Code != http.StatusNoContent {
		t.Fatalf("final logout status %d", finalRec.Code)
	}
}

func TestAuthAcceptanceDemoBlocksMutations(t *testing.T) {
	srv, _ := newDemoTestServer(t)
	handler := srv.Handler()

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/instances", `{"name":"x","serverUrl":"https://x","username":"u","password":"p"}`},
		{http.MethodPost, "/api/music/playlists", `{"name":"Blocked"}`},
		{http.MethodPost, "/api/music/favorites/track-1", `{"trackTitle":"Song","artistName":"Artist"}`},
		{http.MethodDelete, "/api/music/favorites/track-1", ``},
	}

	for _, ep := range endpoints {
		var body *bytes.Buffer
		if ep.body != "" {
			body = bytes.NewBufferString(ep.body)
		} else {
			body = bytes.NewBuffer(nil)
		}
		req := httptest.NewRequest(ep.method, ep.path, body)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s %s expected 403, got %d: %s", ep.method, ep.path, rec.Code, rec.Body.String())
		}
	}
}
