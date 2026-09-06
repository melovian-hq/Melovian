// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package localmusic

import (
	"os"

	"github.com/dhowden/tag"
)

const MaxEmbeddedCoverBytes = 4 << 20

func ReadEmbeddedCover(path string) ([]byte, string, bool) {
	f, err := os.Open(path) //#nosec G304 -- path validated against library root before call
	if err != nil {
		return nil, "", false
	}
	defer func() { _ = f.Close() }()

	meta, err := tag.ReadFrom(f)
	if err != nil {
		return nil, "", false
	}
	picture := meta.Picture()
	if picture == nil || len(picture.Data) == 0 {
		return nil, "", false
	}
	if len(picture.Data) > MaxEmbeddedCoverBytes {
		return nil, "", false
	}
	mime := picture.MIMEType
	if mime == "" {
		mime = "image/jpeg"
	}
	return picture.Data, mime, true
}
