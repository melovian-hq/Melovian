// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package democatalog

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"melovian/internal/melog"
	"melovian/internal/metadata"
)

type cachedCover struct {
	body        []byte
	contentType string
}

var coverCache sync.Map

func serveCover(w http.ResponseWriter, id string, size int) {
	if id == "" {
		writeErr(w, 10, "missing id")
		return
	}

	if cached, ok := coverCache.Load(id); ok {
		entry := cached.(cachedCover)
		w.Header().Set("Content-Type", entry.contentType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(entry.body)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if body, contentType, ok := fetchCatalogArtwork(ctx, id); ok {
		coverCache.Store(id, cachedCover{body: body, contentType: contentType})
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body) //#nosec G705 -- fetched artwork with image content type
		return
	}

	body := CoverSVG(id, size)
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func fetchCatalogArtwork(ctx context.Context, id string) ([]byte, string, bool) {
	id = melog.Sanitize(id)
	artist, album := coverQuery(id)
	if artist == "" && album == "" {
		return nil, "", false
	}

	var artworkURL string
	var err error
	if album != "" {
		artworkURL, err = metadata.LookupAlbumArtworkURL(ctx, artist, album)
		if err != nil {
			slog.Debug("demo cover itunes album lookup failed", "id", id, "err", err)
		}
	}
	if artworkURL == "" && album != "" {
		// Fall back to a track from the album when the release is too new
		// for a clean album entity match.
		if songs := Get().AlbumSongs(id); len(songs) > 0 {
			artworkURL, err = metadata.LookupSongArtworkURL(ctx, artist, songs[0].Title, album)
			if err != nil {
				slog.Debug("demo cover itunes song lookup failed", "id", id, "err", err)
			}
		}
	}
	if artworkURL == "" && artist != "" {
		artworkURL, err = metadata.LookupArtistArtworkURL(ctx, artist)
		if err != nil {
			slog.Debug("demo cover itunes artist lookup failed", "id", id, "err", err)
		}
	}
	if artworkURL == "" {
		return nil, "", false
	}

	body, contentType, err := metadata.FetchArtwork(ctx, artworkURL)
	if err != nil || len(body) == 0 {
		slog.Debug("demo cover fetch failed", "id", id, "err", err)
		return nil, "", false
	}
	if contentType == "" || !strings.HasPrefix(contentType, "image/") {
		contentType = "image/jpeg"
	}
	return body, contentType, true
}

func coverQuery(id string) (artist, album string) {
	c := Get()
	if a, ok := c.Album(id); ok {
		return a.Artist, a.Name
	}
	if a, ok := c.Artist(id); ok {
		return a.Name, ""
	}
	if s, ok := c.Song(id); ok {
		return s.Artist, s.Album
	}
	if p, ok := c.Playlist(id); ok {
		for _, sid := range p.SongIDs {
			if s, ok := c.Song(sid); ok {
				return s.Artist, s.Album
			}
		}
	}
	return "", ""
}

// WarmArtworkCache prefetches iTunes covers for demo albums and artists.
func WarmArtworkCache() {
	c := Get()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for _, album := range c.Albums {
		if ctx.Err() != nil {
			return
		}
		if _, ok := coverCache.Load(album.ID); ok {
			continue
		}
		if body, contentType, ok := fetchCatalogArtwork(ctx, album.ID); ok {
			coverCache.Store(album.ID, cachedCover{body: body, contentType: contentType})
		}
	}
	for _, artist := range c.Artists {
		if ctx.Err() != nil {
			return
		}
		if _, ok := coverCache.Load(artist.ID); ok {
			continue
		}
		if body, contentType, ok := fetchCatalogArtwork(ctx, artist.ID); ok {
			coverCache.Store(artist.ID, cachedCover{body: body, contentType: contentType})
		}
	}
	slog.Info("demo artwork cache warmed")
}
