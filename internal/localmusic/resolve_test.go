// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package localmusic

import "testing"

func TestContentTypeForFormatVideo(t *testing.T) {
	if got := ContentTypeForFormat("mp4"); got != "video/mp4" {
		t.Fatalf("mp4=%q", got)
	}
	if got := ContentTypeForFormat("m4v"); got != "video/mp4" {
		t.Fatalf("m4v=%q", got)
	}
	if got := ContentTypeForFormat("webm"); got != "video/webm" {
		t.Fatalf("webm=%q", got)
	}
	if got := ContentTypeForFormat("mp3"); got != "audio/mpeg" {
		t.Fatalf("mp3=%q", got)
	}
}
