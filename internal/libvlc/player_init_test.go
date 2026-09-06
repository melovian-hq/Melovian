// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package libvlc

import "testing"

func TestNewPlayerWhenInstalled(t *testing.T) {
	if !Available() {
		t.Skip("libvlc is not installed on this host")
	}
	player, err := NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	player.Close()
}
