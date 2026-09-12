// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"melovian/internal/api/apishared"
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
				ctx = apishared.WithUserID(ctx, tc.userID)
			}
			if tc.instanceID != "" {
				ctx = apishared.WithInstanceID(ctx, tc.instanceID)
			}
			if got := apishared.ResolveProgressUserID(ctx); got != tc.want {
				t.Fatalf("apishared.ResolveProgressUserID() = %q, want %q", got, tc.want)
			}
		})
	}
}
