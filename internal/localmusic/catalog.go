// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package localmusic

import (
	"crypto/rand"
	"math/big"
	"sort"
	"strings"

	"melovian/internal/store"
)

type Artist struct {
	ID         string
	Name       string
	AlbumCount int
	CoverArt   string
	sortName   string
}

type Album struct {
	ID         string
	Name       string
	Artist     string
	ArtistID   string
	SongCount  int
	Duration   int
	CoverArt   string
	sortName   string
	sortArtist string
}

type Song struct {
	ID         string
	Title      string
	Album      string
	AlbumID    string
	Artist     string
	ArtistID   string
	Track      int
	Duration   int
	CoverArt   string
	Suffix     string
	Genre      string
	sortTitle  string
	sortArtist string
	sortAlbum  string
}

type Catalog struct {
	Artists        []Artist
	Albums         map[string]Album
	Songs          map[string]Song
	artistsByID    map[string]Artist
	albumsByArtist map[string][]string
	songsByAlbum   map[string][]string
	albumsByName   []Album
	albumsByIDDesc []Album
	songsByIDAsc   []Song
}

type LibraryStats struct {
	SongCount   int
	AlbumCount  int
	ArtistCount int
}

func BuildCatalog(tracks []store.CatalogTrack) Catalog {
	albums := make(map[string]Album, len(tracks)/8+1)
	artists := make(map[string]Artist, len(tracks)/16+1)
	songs := make(map[string]Song, len(tracks))
	albumsByArtist := make(map[string][]string)
	songsByAlbum := make(map[string][]string)
	artistIDs := make(map[string]string, len(tracks)/16+1)
	albumIDsByArtist := make(map[string]map[string]string, len(tracks)/16+1)
	normalized := make(map[string]string, len(tracks)/4+1)
	normalizeCached := func(value string) string {
		if cached, ok := normalized[value]; ok {
			return cached
		}
		cached := NormalizeName(value)
		normalized[value] = cached
		return cached
	}

	for _, track := range tracks {
		artistName := ArtistName(track.Artist)
		albumName := AlbumName(track.Album)
		sortArtist := normalizeCached(artistName)
		sortAlbum := normalizeCached(albumName)
		sortTitle := normalizeCached(track.Title)

		artistID, ok := artistIDs[sortArtist]
		if !ok {
			artistID = ArtistIDNormalized(sortArtist)
			artistIDs[sortArtist] = artistID
		}

		byAlbum := albumIDsByArtist[sortArtist]
		if byAlbum == nil {
			byAlbum = make(map[string]string, 4)
			albumIDsByArtist[sortArtist] = byAlbum
		}
		albumID, ok := byAlbum[sortAlbum]
		if !ok {
			albumID = AlbumIDNormalized(sortArtist, sortAlbum)
			byAlbum[sortAlbum] = albumID
		}
		durationSec := track.DurationMs / 1000

		songs[track.ID] = Song{
			ID:         track.ID,
			Title:      track.Title,
			Album:      albumName,
			AlbumID:    albumID,
			Artist:     artistName,
			ArtistID:   artistID,
			Track:      track.TrackNum,
			Duration:   durationSec,
			CoverArt:   albumID,
			Suffix:     track.Format,
			Genre:      track.Genre,
			sortTitle:  sortTitle,
			sortArtist: sortArtist,
			sortAlbum:  sortAlbum,
		}

		if _, ok := albums[albumID]; !ok {
			albums[albumID] = Album{
				ID:         albumID,
				Name:       albumName,
				Artist:     artistName,
				ArtistID:   artistID,
				CoverArt:   albumID,
				sortName:   sortAlbum,
				sortArtist: sortArtist,
			}
			albumsByArtist[artistID] = append(albumsByArtist[artistID], albumID)
		}
		album := albums[albumID]
		album.SongCount++
		album.Duration += durationSec
		albums[albumID] = album

		if _, ok := artists[artistID]; !ok {
			artists[artistID] = Artist{
				ID:       artistID,
				Name:     artistName,
				CoverArt: albumID,
				sortName: sortArtist,
			}
		}

		songsByAlbum[albumID] = append(songsByAlbum[albumID], track.ID)
	}

	for artistID, albumIDs := range albumsByArtist {
		artist := artists[artistID]
		artist.AlbumCount = len(albumIDs)
		artists[artistID] = artist

		sort.Slice(albumIDs, func(i, j int) bool {
			return albums[albumIDs[i]].sortName < albums[albumIDs[j]].sortName
		})
	}

	for albumID, ids := range songsByAlbum {
		sort.Slice(ids, func(i, j int) bool {
			si, sj := songs[ids[i]], songs[ids[j]]
			if si.Track != sj.Track {
				return si.Track < sj.Track
			}
			return si.sortTitle < sj.sortTitle
		})
		songsByAlbum[albumID] = ids
	}

	artistList := make([]Artist, 0, len(artists))
	artistsByID := make(map[string]Artist, len(artists))
	for _, artist := range artists {
		artistList = append(artistList, artist)
		artistsByID[artist.ID] = artist
	}
	sort.Slice(artistList, func(i, j int) bool {
		return artistList[i].sortName < artistList[j].sortName
	})

	albumList := make([]Album, 0, len(albums))
	for _, album := range albums {
		albumList = append(albumList, album)
	}

	albumsByName := make([]Album, len(albumList))
	copy(albumsByName, albumList)
	sort.Slice(albumsByName, func(i, j int) bool {
		return albumsByName[i].sortName < albumsByName[j].sortName
	})

	albumsByIDDesc := make([]Album, len(albumList))
	copy(albumsByIDDesc, albumList)
	sort.Slice(albumsByIDDesc, func(i, j int) bool {
		return albumsByIDDesc[i].ID > albumsByIDDesc[j].ID
	})

	songList := make([]Song, 0, len(songs))
	for _, song := range songs {
		songList = append(songList, song)
	}
	songsByIDAsc := make([]Song, len(songList))
	copy(songsByIDAsc, songList)
	sort.Slice(songsByIDAsc, func(i, j int) bool {
		return songsByIDAsc[i].ID < songsByIDAsc[j].ID
	})

	return Catalog{
		Artists:        artistList,
		Albums:         albums,
		Songs:          songs,
		artistsByID:    artistsByID,
		albumsByArtist: albumsByArtist,
		songsByAlbum:   songsByAlbum,
		albumsByName:   albumsByName,
		albumsByIDDesc: albumsByIDDesc,
		songsByIDAsc:   songsByIDAsc,
	}
}

func (c Catalog) Stats() LibraryStats {
	return LibraryStats{
		SongCount:   len(c.Songs),
		AlbumCount:  len(c.Albums),
		ArtistCount: len(c.Artists),
	}
}

func (c Catalog) Artist(id string) (Artist, []Album, bool) {
	artist, ok := c.artistsByID[id]
	if !ok {
		return Artist{}, nil, false
	}

	albumIDs := c.albumsByArtist[id]
	albums := make([]Album, len(albumIDs))
	for i, albumID := range albumIDs {
		albums[i] = c.Albums[albumID]
	}
	return artist, albums, true
}

func (c Catalog) Album(id string) (Album, []Song, bool) {
	album, ok := c.Albums[id]
	if !ok {
		return Album{}, nil, false
	}
	songIDs := c.songsByAlbum[id]
	songs := make([]Song, len(songIDs))
	for i, songID := range songIDs {
		songs[i] = c.Songs[songID]
	}
	return album, songs, true
}

func (c Catalog) Song(id string) (Song, bool) {
	song, ok := c.Songs[id]
	return song, ok
}

func (c Catalog) CoverTrackID(id string) (string, bool) {
	if songIDs, ok := c.songsByAlbum[id]; ok && len(songIDs) > 0 {
		return songIDs[0], true
	}
	if _, ok := c.Songs[id]; ok {
		return id, true
	}
	if albumIDs, ok := c.albumsByArtist[id]; ok && len(albumIDs) > 0 {
		if songIDs, ok := c.songsByAlbum[albumIDs[0]]; ok && len(songIDs) > 0 {
			return songIDs[0], true
		}
	}
	return "", false
}

func (c Catalog) AlbumList(listType string, size int, offset int) []Album {
	if offset < 0 {
		offset = 0
	}
	switch listType {
	case "alphabeticalByName", "alphabetical":
		return sliceAlbums(c.albumsByName, size, offset)
	case "frequent":
		albums := append([]Album{}, c.albumsByName...)
		sort.Slice(albums, func(i, j int) bool {
			if albums[i].SongCount == albums[j].SongCount {
				return albums[i].sortName < albums[j].sortName
			}
			return albums[i].SongCount > albums[j].SongCount
		})
		return sliceAlbums(albums, size, offset)
	case "random":
		albums := append([]Album{}, c.albumsByName...)
		shuffleAlbums(albums)
		return sliceAlbums(albums, size, offset)
	default:
		return sliceAlbums(c.albumsByIDDesc, size, offset)
	}
}

func (c Catalog) RandomSongs(size int) []Song {
	songs := append([]Song{}, c.songsByIDAsc...)
	shuffleSongs(songs)
	return sliceSongs(songs, size)
}

func shuffleSongs(songs []Song) {
	shuffleSlice(len(songs), func(i, j int) {
		songs[i], songs[j] = songs[j], songs[i]
	})
}

func shuffleAlbums(albums []Album) {
	shuffleSlice(len(albums), func(i, j int) {
		albums[i], albums[j] = albums[j], albums[i]
	})
}

func shuffleSlice(length int, swap func(i, j int)) {
	for i := length - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return
		}
		swap(i, int(jBig.Int64()))
	}
}

func sliceAlbums(albums []Album, size int, offset int) []Album {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(albums) {
		return nil
	}
	albums = albums[offset:]
	if size <= 0 || size >= len(albums) {
		return albums
	}
	return albums[:size]
}

func sliceSongs(songs []Song, size int) []Song {
	if size <= 0 || size >= len(songs) {
		return songs
	}
	return songs[:size]
}

func (c Catalog) Search(query string, limit int) ([]Artist, []Album, []Song) {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return nil, nil, nil
	}
	if limit <= 0 {
		limit = 20
	}

	var artists []Artist
	artists = make([]Artist, 0, limit)
	for _, artist := range c.Artists {
		if strings.Contains(artist.sortName, needle) {
			artists = append(artists, artist)
			if len(artists) >= limit {
				break
			}
		}
	}

	var albums []Album
	albums = make([]Album, 0, limit)
	for _, album := range c.Albums {
		if strings.Contains(album.sortName, needle) ||
			strings.Contains(album.sortArtist, needle) {
			albums = append(albums, album)
			if len(albums) >= limit {
				break
			}
		}
	}

	var songs []Song
	songs = make([]Song, 0, limit)
	for _, song := range c.Songs {
		if strings.Contains(song.sortTitle, needle) ||
			strings.Contains(song.sortArtist, needle) ||
			strings.Contains(song.sortAlbum, needle) {
			songs = append(songs, song)
			if len(songs) >= limit {
				break
			}
		}
	}

	return artists, albums, songs
}
