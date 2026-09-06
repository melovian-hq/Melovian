// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader

import "testing"

func FuzzFileSignature(f *testing.F) {
	f.Add("track.mp3", int64(1024), int64(1700000000))
	f.Add("", int64(-1), int64(-99))
	f.Add("/music/artist/album/flac file.flac", int64(0), int64(0))

	f.Fuzz(func(t *testing.T, path string, size int64, mtime int64) {
		first := FileSignature(path, size, mtime)
		second := FileSignature(path, size, mtime)

		if first == "" {
			t.Fatal("expected non-empty signature")
		}
		if len(first) != 16 {
			t.Fatalf("expected 16-char hex signature, got len %d", len(first))
		}
		if first != second {
			t.Fatalf("signature must be deterministic")
		}

		if size < 0 || mtime < 0 {
			clamped := FileSignature(path, max(size, 0), max(mtime, 0))
			if first != clamped {
				t.Fatal("negative size/mtime should clamp consistently")
			}
		}
	})
}
