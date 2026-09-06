// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"melovian/internal/localmusic"
	"melovian/internal/store"
)

type localArtistView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AlbumCount int    `json:"albumCount"`
	CoverArt   string `json:"coverArt"`
}

type localAlbumView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Artist    string `json:"artist"`
	ArtistID  string `json:"artistId"`
	SongCount int    `json:"songCount"`
	Duration  int    `json:"duration"`
	CoverArt  string `json:"coverArt"`
}

type localSongView struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Album        string `json:"album"`
	AlbumID      string `json:"albumId"`
	Artist       string `json:"artist"`
	ArtistID     string `json:"artistId"`
	Track        int    `json:"track"`
	Duration     int    `json:"duration"`
	CoverArt     string `json:"coverArt"`
	Suffix       string `json:"suffix"`
	Genre        string `json:"genre,omitempty"`
	PlayCount    int    `json:"playCount,omitempty"`
	LastPlayedAt string `json:"lastPlayedAt,omitempty"`
	Starred      string `json:"starred,omitempty"`
}

type localArtistsResponse struct {
	Artists []localArtistView `json:"artists"`
}

type localAlbumsResponse struct {
	Albums []localAlbumView `json:"albums"`
}

type localArtistDetailResponse struct {
	Artist localArtistView  `json:"artist"`
	Albums []localAlbumView `json:"albums"`
}

type localAlbumDetailResponse struct {
	Album localAlbumView  `json:"album"`
	Songs []localSongView `json:"songs"`
}

type localSearchResponse struct {
	Artists []localArtistView `json:"artists"`
	Albums  []localAlbumView  `json:"albums"`
	Songs   []localSongView   `json:"songs"`
}

type localSongsResponse struct {
	Songs []localSongView `json:"songs"`
}

func localArtistViewFrom(artist localmusic.Artist) localArtistView {
	return localArtistView{
		ID:         artist.ID,
		Name:       artist.Name,
		AlbumCount: artist.AlbumCount,
		CoverArt:   artist.CoverArt,
	}
}

func localAlbumViewFrom(album localmusic.Album) localAlbumView {
	return localAlbumView{
		ID:        album.ID,
		Name:      album.Name,
		Artist:    album.Artist,
		ArtistID:  album.ArtistID,
		SongCount: album.SongCount,
		Duration:  album.Duration,
		CoverArt:  album.CoverArt,
	}
}

func localSongViewFrom(song localmusic.Song) localSongView {
	return localSongView{
		ID:       song.ID,
		Title:    song.Title,
		Album:    song.Album,
		AlbumID:  song.AlbumID,
		Artist:   song.Artist,
		ArtistID: song.ArtistID,
		Track:    song.Track,
		Duration: song.Duration,
		CoverArt: song.CoverArt,
		Suffix:   song.Suffix,
		Genre:    song.Genre,
	}
}

func localArtistViewsFrom(artists []localmusic.Artist) []localArtistView {
	out := make([]localArtistView, len(artists))
	for i, artist := range artists {
		out[i] = localArtistViewFrom(artist)
	}
	return out
}

func localAlbumViewsFrom(albums []localmusic.Album) []localAlbumView {
	out := make([]localAlbumView, len(albums))
	for i, album := range albums {
		out[i] = localAlbumViewFrom(album)
	}
	return out
}

func localSongViewsFrom(songs []localmusic.Song) []localSongView {
	out := make([]localSongView, len(songs))
	for i, song := range songs {
		out[i] = localSongViewFrom(song)
	}
	return out
}

// localSongViewFromTrack builds a view straight from a store row. Album and
// artist IDs are derived the same way BuildCatalog derives them so the same
// track resolves to the same album page whether it came through the catalog
// or a direct query.
func localSongViewFromTrack(track store.LocalTrack) localSongView {
	artist := localmusic.ArtistName(track.Artist)
	album := localmusic.AlbumName(track.Album)
	sortArtist := localmusic.NormalizeName(artist)
	albumID := localmusic.AlbumIDNormalized(sortArtist, localmusic.NormalizeName(album))
	return localSongView{
		ID:       track.ID,
		Title:    track.Title,
		Album:    album,
		AlbumID:  albumID,
		Artist:   artist,
		ArtistID: localmusic.ArtistIDNormalized(sortArtist),
		Track:    track.TrackNum,
		Duration: track.DurationMs / 1000,
		CoverArt: albumID,
		Suffix:   track.Format,
		Genre:    track.Genre,
	}
}
