// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"testing"
)

func TestResolveProgressUserID(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		instanceID string
		want       string
	}{
		{"both", "user-1", "inst-1", "user:user-1:instance:inst-1"},
		{"user only", "user-1", "", "user:user-1"},
		{"instance only", "", "inst-1", "instance:inst-1"},
		{"neither", "", "", "local"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.userID != "" {
				ctx = context.WithValue(ctx, userContextKey, tc.userID)
			}
			if tc.instanceID != "" {
				ctx = context.WithValue(ctx, instanceContextKey, tc.instanceID)
			}
			if got := ResolveProgressUserID(ctx); got != tc.want {
				t.Fatalf("ResolveProgressUserID() = %q, want %q", got, tc.want)
			}
		})
	}
}
