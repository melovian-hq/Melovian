// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import "testing"

func TestNormalizeDocumentAcceptance(t *testing.T) {
	raw := "First line\n\n  Second line  \n\t\nThird line\n"
	doc, err := NormalizeDocument(&Document{
		Source:   "custom",
		Artist:   "Test Artist",
		Title:    "Test Title",
		RawValue: raw,
	})
	if err != nil {
		t.Fatalf("NormalizeDocument: %v", err)
	}
	if doc == nil {
		t.Fatal("expected document")
		return
	}
	if len(doc.Lines) == 0 {
		t.Fatal("expected non-empty lines")
	}
	for i, line := range doc.Lines {
		if line.Text == "" {
			t.Fatalf("line %d is empty", i)
		}
	}
	if doc.RawValue != raw {
		t.Fatalf("expected original RawValue preserved, got %q", doc.RawValue)
	}
}
