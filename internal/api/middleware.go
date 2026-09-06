// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"melovian/internal/store"
)

type ctxKey int

const (
	deviceContextKey ctxKey = iota
	instanceContextKey
	userContextKey
)

func UserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(userContextKey).(string); ok {
		return v
	}
	return ""
}

func InstanceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(instanceContextKey).(string); ok {
		return v
	}
	return ""
}

func DeviceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(deviceContextKey).(string); ok {
		return v
	}
	return ""
}

func ResolveProgressUserID(ctx context.Context) string {
	userID := UserIDFromContext(ctx)
	instanceID := InstanceIDFromContext(ctx)
	if userID != "" && instanceID != "" {
		return fmt.Sprintf("user:%s:instance:%s", userID, instanceID)
	}
	if userID != "" {
		return fmt.Sprintf("user:%s", userID)
	}
	if instanceID != "" {
		return fmt.Sprintf("instance:%s", instanceID)
	}
	return "local"
}

func instanceIDFromRequest(r *http.Request) string {
	if id := strings.TrimSpace(r.Header.Get("X-Instance-Id")); id != "" {
		return id
	}
	return strings.TrimSpace(r.URL.Query().Get("_instance"))
}

type InstanceResolver interface {
	ResolveInstanceID(r *http.Request) (string, error)
}

func AuthMiddleware(auth *store.AuthStore, demo bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if demo || auth == nil || !auth.Enabled() || isPublicAPIPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		token := sessionTokenFromRequest(r)
		userID, err := auth.UserIDFromToken(token)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Middleware(resolver InstanceResolver, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), deviceContextKey, r.Header.Get("X-Device-Id"))
		if r.URL.Path == "/api/ws" {
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		instanceID, err := resolver.ResolveInstanceID(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctx = context.WithValue(ctx, instanceContextKey, instanceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ChainAuthInstance(auth *store.AuthStore, demo bool, resolver InstanceResolver, next http.Handler) http.Handler {
	return AuthMiddleware(auth, demo, Middleware(resolver, next))
}
