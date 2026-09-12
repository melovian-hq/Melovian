// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Package apishared holds leaf helpers shared by the api domain packages:
// request context keys, cookie helpers, rate limiting, CORS origin
// allowlists, download naming, and the per-instance Subsonic client
// resolver. It must not import any api domain package.
package apishared

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

func WithDeviceID(ctx context.Context, deviceID string) context.Context {
	return context.WithValue(ctx, deviceContextKey, deviceID)
}

func WithInstanceID(ctx context.Context, instanceID string) context.Context {
	return context.WithValue(ctx, instanceContextKey, instanceID)
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userContextKey, userID)
}

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

func InstanceIDFromRequest(r *http.Request) string {
	if id := strings.TrimSpace(r.Header.Get("X-Instance-Id")); id != "" {
		return id
	}
	return strings.TrimSpace(r.URL.Query().Get("_instance"))
}

func SessionTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(store.SessionCookieName())
	if err != nil {
		return ""
	}
	return cookie.Value
}

func IsPublicAPIPath(path string) bool {
	switch path {
	case "/health", "/metrics", "/api/client-log", "/api/ws", "/api/config", "/api/auth/status", "/api/auth/setup", "/api/auth/login", "/api/auth/logout", "/api/auth/oidc/login", "/api/auth/oidc/callback":
		return true
	default:
		return false
	}
}
