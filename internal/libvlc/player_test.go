// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package libvlc

import "testing"

func TestAvailableWithoutLibrary(t *testing.T) {
	if Available() {
		t.Skip("libvlc is installed on this host")
	}
}

func TestNewPlayerWithoutLibrary(t *testing.T) {
	if Available() {
		t.Skip("libvlc is installed on this host")
	}
	if _, err := NewPlayer(); err == nil {
		t.Fatal("expected error when libvlc is unavailable")
	}
}
