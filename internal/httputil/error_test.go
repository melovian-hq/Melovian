// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusBadRequest, "invalid_json", "invalid json: boom")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type %q", ct)
	}

	var payload APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Code != "invalid_json" {
		t.Fatalf("code %q", payload.Code)
	}
	if payload.Error != "invalid json: boom" {
		t.Fatalf("error %q", payload.Error)
	}
}

func TestWriteErrorDefaults(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusInternalServerError, "", "")

	var payload APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Code != "error" {
		t.Fatalf("code %q", payload.Code)
	}
	if payload.Error != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("error %q", payload.Error)
	}
}
