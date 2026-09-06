// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"melovian/internal/httputil"
)

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get(httputil.RequestIDHeader))
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(httputil.RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(httputil.ContextWithRequestID(r.Context(), id)))
	})
}

func newRequestID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "req-unknown"
	}
	return hex.EncodeToString(buf[:])
}
