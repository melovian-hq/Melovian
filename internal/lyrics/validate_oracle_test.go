// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import "testing"

func TestNormalizeDocumentOracleRejectsNilAndEmpty(t *testing.T) {
	if _, err := NormalizeDocument(nil); err == nil {
		t.Fatal("expected error for nil document")
	}

	if _, err := NormalizeDocument(&Document{Source: "test"}); err == nil {
		t.Fatal("expected error for empty lines")
	}

	if _, err := NormalizeDocument(&Document{
		Source: "test",
		Lines:  []Line{{Text: "   "}, {Text: "\t\n"}},
	}); err == nil {
		t.Fatal("expected error when all lines are blank")
	}
}

func TestNormalizeDocumentOracleTrimsAndIsIdempotent(t *testing.T) {
	start := 1200
	doc, err := NormalizeDocument(&Document{
		Source: "test",
		Lines: []Line{
			{Text: "  Hello  "},
			{Text: ""},
			{Text: "\tWorld\n"},
			{Text: "Again", StartMs: &start},
		},
	})
	if err != nil {
		t.Fatalf("NormalizeDocument: %v", err)
	}
	if len(doc.Lines) != 3 {
		t.Fatalf("expected 3 non-blank lines, got %d", len(doc.Lines))
	}
	if doc.Lines[0].Text != "Hello" || doc.Lines[1].Text != "World" || doc.Lines[2].Text != "Again" {
		t.Fatalf("unexpected trimmed lines: %+v", doc.Lines)
	}
	if doc.Lines[2].StartMs == nil || *doc.Lines[2].StartMs != start {
		t.Fatalf("expected StartMs preserved, got %+v", doc.Lines[2].StartMs)
	}
	if doc.RawValue == "" {
		t.Fatal("expected RawValue to be filled")
	}

	again, err := NormalizeDocument(doc)
	if err != nil {
		t.Fatalf("second NormalizeDocument: %v", err)
	}
	if len(again.Lines) != len(doc.Lines) {
		t.Fatalf("idempotent line count changed: %d -> %d", len(doc.Lines), len(again.Lines))
	}
	for i := range doc.Lines {
		if again.Lines[i].Text != doc.Lines[i].Text {
			t.Fatalf("idempotent text changed at %d: %q -> %q", i, doc.Lines[i].Text, again.Lines[i].Text)
		}
	}
	if again.RawValue != doc.RawValue {
		t.Fatalf("idempotent RawValue changed: %q -> %q", doc.RawValue, again.RawValue)
	}
}
