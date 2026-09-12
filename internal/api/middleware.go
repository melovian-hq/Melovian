// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"melovian/internal/api/apishared"
	"melovian/internal/httputil"
	"melovian/internal/store"
)

type InstanceResolver interface {
	ResolveInstanceID(r *http.Request) (string, error)
}

func AuthMiddleware(auth *store.AuthStore, demo bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if demo || auth == nil || !auth.Enabled() || apishared.IsPublicAPIPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		token := apishared.SessionTokenFromRequest(r)
		userID, err := auth.UserIDFromToken(token)
		if err != nil {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
			return
		}

		next.ServeHTTP(w, r.WithContext(apishared.WithUserID(r.Context(), userID)))
	})
}

func Middleware(resolver InstanceResolver, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := apishared.WithDeviceID(r.Context(), r.Header.Get("X-Device-Id"))
		if r.URL.Path == "/api/ws" {
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		instanceID, err := resolver.ResolveInstanceID(r)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		ctx = apishared.WithInstanceID(ctx, instanceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ChainAuthInstance(auth *store.AuthStore, demo bool, resolver InstanceResolver, next http.Handler) http.Handler {
	return AuthMiddleware(auth, demo, Middleware(resolver, next))
}
