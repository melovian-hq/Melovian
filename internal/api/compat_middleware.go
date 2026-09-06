// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"strconv"

	"melovian/internal/brand"
	"melovian/internal/compat"
	"melovian/internal/httputil"
)

// CompatMiddleware advertises server version/capabilities and refuses clients
// older than compat.MinClientVersion.
func CompatMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(compat.HeaderServerVersion, compat.Version)
		w.Header().Set(compat.HeaderAPIVersion, strconv.Itoa(compat.APIVersion))
		w.Header().Set(compat.HeaderCapabilities, compat.JoinCapabilities(compat.Capabilities))

		clientVersion := r.Header.Get(compat.HeaderClientVersion)
		if compat.ClientTooOld(clientVersion) {
			httputil.WriteJSON(w, http.StatusUpgradeRequired, map[string]any{
				"error":            "client_too_old",
				"message":          "This " + brand.Name + " client is too old for this server. Update the client.",
				"minClientVersion": compat.MinClientVersion,
				"serverVersion":    compat.Version,
				"apiVersion":       compat.APIVersion,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
