// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type APIError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// WriteInternalError logs the real error with the request id and returns a
// generic message to the client. Internal errors routinely embed filesystem
// paths, SQL details, and upstream hostnames that should never reach a
// caller.
func WriteInternalError(w http.ResponseWriter, r *http.Request, op string, err error) {
	slog.Warn("request failed",
		"op", op,
		"err", err,
		"request_id", RequestIDFromContext(r.Context()),
		"path", r.URL.Path,
	)
	WriteError(w, http.StatusInternalServerError, "internal_error", "internal error")
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	if code == "" {
		code = "error"
	}
	if message == "" {
		message = http.StatusText(status)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIError{
		Error: message,
		Code:  code,
	})
}
