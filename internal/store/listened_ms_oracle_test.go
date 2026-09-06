// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import "testing"

func TestClampListenDeltaNeverExceedsTrackDurationOracle(t *testing.T) {
	// A short track must never accept a progress chunk longer than itself.
	got := clampListenDelta(15_000, 4_000)
	if got != 4_000 {
		t.Fatalf("clampListenDelta(15000, 4000)=%d want 4000", got)
	}
}
