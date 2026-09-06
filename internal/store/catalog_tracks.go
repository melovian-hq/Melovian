// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

type CatalogTrack struct {
	ID         string
	RelPath    string
	Title      string
	Artist     string
	Album      string
	TrackNum   int
	DurationMs int
	Format     string
	Genre      string
}

func (s *LocalTrackStore) ListForCatalog(libraryID string) ([]CatalogTrack, error) {
	rows, err := s.db.query(
		`SELECT id, rel_path, title, artist, album, track_num, duration_ms, format, genre
		 FROM local_tracks WHERE library_id = ? AND status = ? AND duplicate_of = ''
		   AND `+localTrackAudioFilter+`
		 ORDER BY artist ASC, album ASC, track_num ASC, title ASC`,
		libraryID, TrackStatusPresent,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []CatalogTrack
	for rows.Next() {
		var track CatalogTrack
		if err := rows.Scan(
			&track.ID, &track.RelPath, &track.Title, &track.Artist, &track.Album,
			&track.TrackNum, &track.DurationMs, &track.Format, &track.Genre,
		); err != nil {
			return nil, err
		}
		if track.Title == "" {
			track.Title = fallbackTitle(track.RelPath)
		}
		items = append(items, track)
	}
	return items, rows.Err()
}
