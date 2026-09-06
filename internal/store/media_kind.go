// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

const (
	MediaKindAudio = "audio"
	MediaKindVideo = "video"
)

func NormalizeMediaKind(kind string) string {
	if kind == MediaKindVideo {
		return MediaKindVideo
	}
	return MediaKindAudio
}
