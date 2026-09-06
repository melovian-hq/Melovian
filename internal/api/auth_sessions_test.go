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

func TestAuthSessionsAndPassword(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	handler := srv.Handler()

	setupBody := bytes.NewBufferString(`{"username":"admin","password":"password123"}`)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/setup", setupBody)
	setupReq.Header.Set("Content-Type", "application/json")
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status %d", setupRec.Code)
	}
	cookie := authCookie(setupRec)
	if cookie == nil {
		t.Fatal("expected session cookie")
	}

	sessionsReq := httptest.NewRequest(http.MethodGet, "/api/auth/sessions", nil)
	sessionsReq.AddCookie(cookie)
	sessionsRec := httptest.NewRecorder()
	handler.ServeHTTP(sessionsRec, sessionsReq)
	if sessionsRec.Code != http.StatusOK {
		t.Fatalf("sessions status %d", sessionsRec.Code)
	}
	var sessionsPayload map[string]any
	if err := json.Unmarshal(sessionsRec.Body.Bytes(), &sessionsPayload); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	sessions, ok := sessionsPayload["sessions"].([]any)
	if !ok || len(sessions) != 1 {
		t.Fatalf("expected one session, got %+v", sessionsPayload)
	}

	changeBody := bytes.NewBufferString(`{"currentPassword":"password123","newPassword":"password456"}`)
	changeReq := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", changeBody)
	changeReq.Header.Set("Content-Type", "application/json")
	changeReq.AddCookie(cookie)
	changeRec := httptest.NewRecorder()
	handler.ServeHTTP(changeRec, changeReq)
	if changeRec.Code != http.StatusNoContent {
		t.Fatalf("change password status %d body %s", changeRec.Code, changeRec.Body.String())
	}
}
