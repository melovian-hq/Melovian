// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hanxi/tag"
)

// TrackMetadata holds editable tag fields for a local audio file.
type TrackMetadata struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	TrackNum    int
	DiscNum     int
	Year        int
	Genre       string
}

func ReadTrackMetadata(path string) (TrackMetadata, error) {
	f, err := openAudioFile(path)
	if err != nil {
		return TrackMetadata{}, err
	}
	defer func() { _ = f.Close() }()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return TrackMetadata{}, fmt.Errorf("read tags: %w", err)
	}

	trackNum, _ := m.Track()
	discNum, _ := m.Disc()
	year := m.Year()

	return TrackMetadata{
		Title:       strings.TrimSpace(m.Title()),
		Artist:      strings.TrimSpace(m.Artist()),
		Album:       strings.TrimSpace(m.Album()),
		AlbumArtist: strings.TrimSpace(m.AlbumArtist()),
		TrackNum:    trackNum,
		DiscNum:     discNum,
		Year:        year,
		Genre:       strings.TrimSpace(m.Genre()),
	}, nil
}

func WriteTrackMetadata(path string, meta TrackMetadata) error {
	opts := tag.WriteOptions{
		Title:       meta.Title,
		Artist:      meta.Artist,
		Album:       meta.Album,
		AlbumArtist: meta.AlbumArtist,
		Year:        meta.Year,
		Genre:       meta.Genre,
	}
	if meta.TrackNum > 0 {
		opts.Track = strconv.Itoa(meta.TrackNum)
	}
	return tag.WriteTag(path, opts)
}
