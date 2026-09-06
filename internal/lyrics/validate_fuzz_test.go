// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import "testing"

func FuzzNormalizeDocument(f *testing.F) {
	f.Add("Line one\nLine two")
	f.Add("")
	f.Add("   \n\t  ")

	f.Fuzz(func(t *testing.T, raw string) {
		doc, err := NormalizeDocument(&Document{
			Source:   "test",
			RawValue: raw,
		})
		if err != nil {
			return
		}
		if doc == nil || len(doc.Lines) == 0 {
			t.Fatal("successful normalize must return non-empty lines")
		}
		for _, line := range doc.Lines {
			if line.Text == "" {
				t.Fatal("normalized lines must not be empty")
			}
		}
	})
}
