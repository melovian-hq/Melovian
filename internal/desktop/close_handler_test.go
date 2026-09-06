// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package desktop

import "testing"

func TestWindowCloseEventNamesMatchFrontendContract(t *testing.T) {
	if MainWindowCloseRequestedEvent != "melovian:window-close-requested" {
		t.Fatalf("unexpected close event %q", MainWindowCloseRequestedEvent)
	}
	if AppQuitRequestedEvent != "melovian:app-quit-requested" {
		t.Fatalf("unexpected quit event %q", AppQuitRequestedEvent)
	}
}
