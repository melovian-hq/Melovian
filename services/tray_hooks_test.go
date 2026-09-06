// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

import "testing"

func TestSetTaskbarIntegrationHook(t *testing.T) {
	called := false
	var got bool
	SetTaskbarIntegrationHook(func(enabled bool) {
		called = true
		got = enabled
	})
	t.Cleanup(func() {
		SetTaskbarIntegrationHook(nil)
	})

	setTaskbarIntegrationEnabled(true)
	if !called || !got {
		t.Fatalf("expected hook to receive enabled=true, called=%v got=%v", called, got)
	}

	called = false
	setTaskbarIntegrationEnabled(false)
	if !called || got {
		t.Fatalf("expected hook to receive enabled=false, called=%v got=%v", called, got)
	}
}

func TestSetTaskbarIntegrationEnabledWithoutHook(t *testing.T) {
	SetTaskbarIntegrationHook(nil)
	setTaskbarIntegrationEnabled(true)
}
